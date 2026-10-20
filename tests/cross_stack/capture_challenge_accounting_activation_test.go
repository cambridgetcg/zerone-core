package cross_stack_test

import (
	"encoding/json"
	"fmt"
	"testing"

	abci "github.com/cometbft/cometbft/abci/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	"github.com/stretchr/testify/require"

	capturechallengetypes "github.com/zerone-chain/zerone/x/capture_challenge/types"
)

// Exercise the installed capture_defense -> capture_challenge adapter and the
// real application BeginBlock order. The two contexts select their marker in
// genesis; no upgrade handler or live chain is simulated by this fixture.
func TestCaptureChallengeAccountingAutomaticTimingThroughApp(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		t.Run(fmt.Sprint(enabled), func(t *testing.T) {
			app := repairsMarkersApp(t)
			genesis := genesisStateWithValSet(t, app, app.DefaultGenesis())
			var challengeGenesis capturechallengetypes.GenesisState
			// This module owns an encoding/json genesis path, not proto JSON.
			require.NoError(t, json.Unmarshal(genesis[capturechallengetypes.ModuleName], &challengeGenesis))
			// Explicit opt-out is essential when rehearsing pre-carrier behavior
			// using native DefaultGenesis, which otherwise selects every repair.
			challengeGenesis.ChallengeAccountingEnabled = enabled
			challengeJSON, err := json.Marshal(&challengeGenesis)
			require.NoError(t, err)
			genesis[capturechallengetypes.ModuleName] = challengeJSON
			bz, err := json.Marshal(genesis)
			require.NoError(t, err)
			_, err = app.InitChain(&abci.RequestInitChain{ChainId: repairsMarkersChainID,
				AppStateBytes: bz, ConsensusParams: simtestutil.DefaultConsensusParams})
			require.NoError(t, err)
			_, err = app.FinalizeBlock(&abci.RequestFinalizeBlock{Height: 1})
			require.NoError(t, err)
			_, err = app.Commit()
			require.NoError(t, err)
			ctx := app.NewContextLegacy(true, cmtproto.Header{ChainID: repairsMarkersChainID, Height: 1000})
			selected, err := app.CaptureChallengeKeeper.ChallengeAccountingEnabled(ctx)
			require.NoError(t, err)
			require.Equal(t, enabled, selected)
			domain := "aw121-persistent-flag"
			app.CaptureDefenseKeeper.RecordVerificationFromKnowledge(ctx, domain, "round-1",
				[]string{"synthetic-validator"}, []bool{true}, []uint64{1000})
			_, err = app.BeginBlocker(ctx)
			require.NoError(t, err)
			ids := app.CaptureChallengeKeeper.GetChallengesByDomain(ctx, domain)
			require.Len(t, ids, 1, "real defense analysis must create the first challenge")
			oldID := ids[0]
			old, found := app.CaptureChallengeKeeper.GetChallenge(ctx, oldID)
			require.True(t, found)
			require.Equal(t, "0", old.Stake)
			require.Equal(t, uint64(21000), old.ReviewDeadline)

			ctx = ctx.WithBlockHeight(int64(old.ReviewDeadline))
			_, err = app.BeginBlocker(ctx)
			require.NoError(t, err)
			want := 1
			if enabled {
				want = 2
			}
			require.Len(t, app.CaptureChallengeKeeper.GetChallengesByDomain(ctx, domain), want,
				"activation advances replacement to the deadline even without a failed refund")
			old, found = app.CaptureChallengeKeeper.GetChallenge(ctx, oldID)
			require.True(t, found)
			require.Equal(t, capturechallengetypes.ChallengeStatus_CHALLENGE_STATUS_EXPIRED, old.Status)
			metrics, found := app.CaptureDefenseKeeper.GetCaptureMetrics(ctx, domain)
			require.True(t, found)
			require.True(t, metrics.Flagged)

			ctx = ctx.WithBlockHeight(22000)
			_, err = app.BeginBlocker(ctx)
			require.NoError(t, err)
			require.Len(t, app.CaptureChallengeKeeper.GetChallengesByDomain(ctx, domain), 2,
				"legacy replacement waits until the next analysis; enabled replacement deduplicates")
		})
	}
}
