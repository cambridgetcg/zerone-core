package cross_stack_test

import (
	sdkmath "cosmossdk.io/math"
	"crypto/sha256"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	kt "github.com/zerone-chain/zerone/x/knowledge/types"
	"google.golang.org/protobuf/proto"
	"testing"
)

// Direct aggregation fixtures supply a synthetic result, not signed reviews.
// Persist their unfinished round so native completion reads authoritative state.
// Existing rounds (including real transaction-path rounds) are never replaced.
func storeUnfinalizedFixtureRound(t *testing.T, h *TestHarness, round *kt.VerificationRound) {
	t.Helper()
	if _, found := h.KnowledgeKeeper.GetVerificationRound(h.Ctx, round.Id); found {
		return
	}
	staged := proto.Clone(round).(*kt.VerificationRound)
	staged.Phase = kt.VerificationPhase_VERIFICATION_PHASE_AGGREGATION
	staged.Verdict = kt.Verdict_VERDICT_UNSPECIFIED
	staged.VerdictBlock = 0
	require.NoError(t, h.KnowledgeKeeper.SetVerificationRound(h.Ctx, staged))
}

// Direct graph fixtures bypass admission but must still back any stated
// collateral. Use a canonical synthetic signer and actually collect its funds.
func fundDirectChallengeFixture(t *testing.T, h *TestHarness, claim *kt.Claim) {
	t.Helper()
	sum := sha256.Sum256([]byte("funded-direct-challenge:" + claim.Id))
	addr := sdk.AccAddress(sum[:20])
	claim.Submitter = addr.String()
	amount, ok := sdkmath.NewIntFromString(claim.Stake)
	require.True(t, ok)
	require.True(t, amount.IsPositive())
	coins := sdk.NewCoins(sdk.NewCoin("uzrn", amount))
	require.NoError(t, h.FundAccount(addr, coins))
	require.NoError(t, h.App.BankKeeper.SendCoinsFromAccountToModule(h.Ctx, addr, kt.ModuleName, coins))
}
