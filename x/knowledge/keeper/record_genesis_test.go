package keeper_test

import (
	"bytes"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"github.com/zerone-chain/zerone/x/knowledge/types"
	"google.golang.org/protobuf/proto"
	"testing"
)

func TestRecordGenesisPreservesCompletedRoundsSelectedIndexAndUnpaidPlan(t *testing.T) {
	k, ctx := setupKnowledgeTest(t)
	require.NoError(t, k.EnableRecordIntegrity(ctx))
	recipient := sdk.AccAddress(bytes.Repeat([]byte{9}, 20)).String()
	old := &types.VerificationRound{Id: "old-round", ClaimId: "claim", Phase: types.VerificationPhase_VERIFICATION_PHASE_COMPLETE, VerdictBlock: 20}
	selected := &types.VerificationRound{Id: "selected-round", ClaimId: "claim", Phase: types.VerificationPhase_VERIFICATION_PHASE_COMPLETE, VerdictBlock: 21,
		VerifierRewardSettlement: &types.VerifierRewardSettlement{CreatedAtBlock: 21, WithheldTotal: "0", Payments: []*types.VerifierRewardPayment{{Verifier: recipient, Amount: "55", Withheld: "0"}}}}
	gs := types.DefaultGenesis()
	gs.PendingClaims = []*types.Claim{{Id: "claim", VerificationRoundId: "selected-round"}, {Id: "unknown-history", VerificationRoundId: "not-preserved-by-old-export"}}
	gs.CompletedRounds = []*types.VerificationRound{old, selected}
	require.NoError(t, k.InitGenesis(ctx, gs))
	exported := k.ExportGenesis(ctx)
	require.True(t, exported.RecordIntegrityEnabled)
	require.Empty(t, exported.ActiveRounds)
	require.Len(t, exported.CompletedRounds, 2)
	require.True(t, proto.Equal(old, exported.CompletedRounds[0]))
	require.True(t, proto.Equal(selected, exported.CompletedRounds[1]))
	k2, ctx2 := setupKnowledgeTest(t)
	require.NoError(t, k2.InitGenesis(ctx2, exported))
	selectedAfter, found := k2.GetVerificationRound(ctx2, "selected-round")
	require.True(t, found)
	require.True(t, proto.Equal(selected, selectedAfter))
	oldAfter, found := k2.GetVerificationRound(ctx2, "old-round")
	require.True(t, found)
	require.True(t, proto.Equal(old, oldAfter))
	require.Nil(t, oldAfter.VerifierRewardSettlement, "unknown legacy payment remains unknown")
	exportedAgain := k2.ExportGenesis(ctx2)
	require.Len(t, exportedAgain.CompletedRounds, 2)
	require.Equal(t, "selected-round", exportedAgain.PendingClaims[0].VerificationRoundId)
	require.Empty(t, k2.ReadMigrationMarker(ctx2, "migration_v8_complete"), "genesis does not fabricate applied upgrade")
}

func TestRecordGenesisRejectsMalformedRoundCollectionsBeforeWrites(t *testing.T) {
	for _, kind := range []string{"nil-claim", "nil-round", "missing-claim", "duplicate", "terminal-in-active", "preseeded-v2"} {
		t.Run(kind, func(t *testing.T) {
			k, ctx := setupKnowledgeTest(t)
			gs := types.DefaultGenesis()
			gs.RecordIntegrityEnabled = false
			gs.ReviewNeutralityEnabled = false
			gs.ClaimRecordsEnabled = false
			gs.PendingClaims = []*types.Claim{{Id: "claim"}}
			round := &types.VerificationRound{Id: "round", ClaimId: "claim", Phase: types.VerificationPhase_VERIFICATION_PHASE_COMMIT}
			gs.ActiveRounds = []*types.VerificationRound{round}
			switch kind {
			case "nil-claim":
				gs.PendingClaims = append(gs.PendingClaims, nil)
			case "nil-round":
				gs.ActiveRounds = append(gs.ActiveRounds, nil)
			case "missing-claim":
				round.ClaimId = "missing"
			case "duplicate":
				gs.ActiveRounds = append(gs.ActiveRounds, round)
			case "terminal-in-active":
				round.Phase = types.VerificationPhase_VERIFICATION_PHASE_COMPLETE
			case "preseeded-v2":
				round.CommitmentScheme = 2
				round.CommitmentChainId = "original-chain"
			}
			require.Error(t, gs.Validate())
			require.Error(t, k.InitGenesis(ctx, gs))
			_, found := k.GetClaim(ctx, "claim")
			require.False(t, found)
			enabled, err := k.RecordIntegrityEnabled(ctx)
			require.NoError(t, err)
			require.False(t, enabled)
		})
	}
}
