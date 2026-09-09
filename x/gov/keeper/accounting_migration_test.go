package keeper

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"cosmossdk.io/log"
	"cosmossdk.io/store"
	storemetrics "cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"github.com/zerone-chain/zerone/x/gov/types"
)

func migrationPreflightFixture(t *testing.T) (Keeper, sdk.Context, *storetypes.KVStoreKey) {
	t.Helper()
	key := storetypes.NewKVStoreKey(types.StoreKey)
	db := dbm.NewMemDB()
	ms := store.NewCommitMultiStore(db, log.NewNopLogger(), storemetrics.NewNoOpMetrics())
	ms.MountStoreWithDB(key, storetypes.StoreTypeIAVL, db)
	require.NoError(t, ms.LoadLatestVersion())
	return NewKeeper(nil, key, "authority", nil, nil), sdk.NewContext(ms, cmtproto.Header{Height: 100}, false, log.NewNopLogger()), key
}

func TestAccountingMigrationQuiescencePreservesTerminalHistoryAndEscrow(t *testing.T) {
	k, ctx, key := migrationPreflightFixture(t)
	k.SetLIP(ctx, &types.LIP{Id: "LIP-1", Category: types.CategoryPhaseTransition, Stage: types.StatusPassed, StakedAmount: "123456"})
	k.SetResearchSpendProposal(ctx, &types.ResearchSpendProposal{ProposalId: 1, Stage: string(types.ResearchStageExecuted), Amount: "100"})
	k.SetSeatElection(ctx, &types.SeatElectionProposal{ProposalId: 1, Stage: types.SeatStagePassed})
	k.SetPhaseTransitionMeta(ctx, &types.PhaseTransitionProposal{LipID: "LIP-1", Stage: types.PhaseTransitionStageActivated})
	original := bytes.Clone(ctx.KVStore(key).Get(types.LIPKey("LIP-1")))
	require.NoError(t, k.ValidateAccountingMigration(ctx))
	require.Equal(t, original, ctx.KVStore(key).Get(types.LIPKey("LIP-1")))
	require.False(t, k.AccountingSafetyEnabled(ctx), "preflight never activates the transition")
}

func TestAccountingMigrationRejectsEveryNonterminalProcess(t *testing.T) {
	for _, tc := range []struct {
		name    string
		key     []byte
		payload any
	}{
		{"text draft", types.LIPKey("LIP-1"), &types.LIP{Id: "LIP-1", Category: types.CategoryText, Stage: types.StatusDraft}},
		{"LIP voting", types.LIPKey("LIP-1"), &types.LIP{Id: "LIP-1", Category: types.CategoryParameter, Stage: types.StatusVoting}},
		{"research voting", types.ResearchSpendKey(1), &types.ResearchSpendProposal{ProposalId: 1, Stage: string(types.ResearchStageVoting)}},
		{"seat nominated", types.SeatElectionKey(1), &types.SeatElectionProposal{ProposalId: 1, Stage: types.SeatStageNominated}},
		{"seat runoff", types.SeatElectionKey(1), &types.SeatElectionProposal{ProposalId: 1, Stage: types.SeatStageRunoff}},
		{"phase pending", types.PhaseTransitionKey("LIP-1"), &types.PhaseTransitionProposal{LipID: "LIP-1", Stage: types.PhaseTransitionStagePending}},
		{"unknown LIP stage", types.LIPKey("LIP-1"), &types.LIP{Id: "LIP-1", Stage: "future"}},
		{"unknown research stage", types.ResearchSpendKey(1), &types.ResearchSpendProposal{ProposalId: 1, Stage: "future"}},
		{"unknown seat stage", types.SeatElectionKey(1), &types.SeatElectionProposal{ProposalId: 1, Stage: "future"}},
		{"unknown phase stage", types.PhaseTransitionKey("LIP-1"), &types.PhaseTransitionProposal{LipID: "LIP-1", Stage: "future"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			k, ctx, key := migrationPreflightFixture(t)
			k.SetLIP(ctx, &types.LIP{Id: "LIP-1", Category: types.CategoryPhaseTransition, Stage: types.StatusPassed})
			raw, err := json.Marshal(tc.payload)
			require.NoError(t, err)
			ctx.KVStore(key).Set(tc.key, raw)
			require.Error(t, k.ValidateAccountingMigration(ctx))
			require.Equal(t, raw, ctx.KVStore(key).Get(tc.key))
			require.False(t, k.AccountingSafetyEnabled(ctx))
		})
	}
}

func TestAccountingMigrationMalformedRecordsCannotHideAProcess(t *testing.T) {
	for _, raw := range []string{
		`null`, `{}`, `{"id":"LIP-1","stage":"passed"} {}`,
		`{"id":"LIP-1","stage":"voting","stage":"passed"}`,
		`{"id":"LIP-1","Stage":"passed"}`,
		`{"id":"LIP-1","stage":"passed","future":true}`,
		`{"id":"different","stage":"passed"}`,
		`{"id":"LIP-1","stage":"passed","param_changes":[{"module":"m","module":"n"}]}`,
		strings.Repeat("[", maxAccountingProcessJSONDepth+1) + "0" + strings.Repeat("]", maxAccountingProcessJSONDepth+1),
		strings.Repeat(" ", maxAccountingProcessValueBytes+1),
	} {
		k, ctx, key := migrationPreflightFixture(t)
		ctx.KVStore(key).Set(types.LIPKey("LIP-1"), []byte(raw))
		require.Error(t, k.ValidateAccountingMigration(ctx), "must reject malformed process")
		require.Equal(t, []byte(raw), ctx.KVStore(key).Get(types.LIPKey("LIP-1")))
	}
	k, ctx, key := migrationPreflightFixture(t)
	ctx.KVStore(key).Set(types.SeatElectionKey(2), []byte(`{"proposal_id":1,"stage":"passed"}`))
	require.Error(t, k.ValidateAccountingMigration(ctx), "primary numeric key must match payload")
}

func TestAccountingMigrationRejectsOrphanPhaseMetadata(t *testing.T) {
	k, ctx, _ := migrationPreflightFixture(t)
	k.SetPhaseTransitionMeta(ctx, &types.PhaseTransitionProposal{LipID: "missing", Stage: types.PhaseTransitionStageActivated})
	require.Error(t, k.ValidateAccountingMigration(ctx))
}
