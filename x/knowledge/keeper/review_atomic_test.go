package keeper

import (
	"github.com/stretchr/testify/require"
	"github.com/zerone-chain/zerone/x/knowledge/types"
	"testing"
)

func TestRecordRoundAtomicPrimaryAndIndexes(t *testing.T) {
	for _, tc := range []struct {
		op     string
		prefix []byte
	}{{"get", types.VerificationRoundKeyPrefix}, {"get", types.ClaimRoundIndexPrefix}, {"set", types.VerificationRoundKeyPrefix}, {"set", types.ClaimRoundIndexPrefix}, {"set", activeRoundKey("")}, {"delete", activeRoundKey("")}} {
		for _, after := range []bool{false, true} {
			if tc.op == "get" && after {
				continue
			}
			t.Run(tc.op+string(tc.prefix)+map[bool]string{false: "/before", true: "/after"}[after], func(t *testing.T) {
				f := setupHandoff(t)
				require.NoError(t, f.k.EnableRecordIntegrity(f.ctx))
				round := &types.VerificationRound{Id: "record-round", ClaimId: "claim", Phase: types.VerificationPhase_VERIFICATION_PHASE_COMMIT, CommitmentScheme: 2, CommitmentChainId: "source-chain"}
				if tc.op == "delete" {
					require.NoError(t, f.k.SetVerificationRound(f.ctx, round))
					round.Phase = types.VerificationPhase_VERIFICATION_PHASE_EXPIRED
				}
				before := f.snapshot(t)
				*f.kFault = handoffFault{tc.op, tc.prefix, after}
				require.Error(t, f.k.SetVerificationRound(f.ctx, round))
				require.Equal(t, before, f.snapshot(t))
				require.Empty(t, f.ctx.EventManager().Events())
				*f.kFault = handoffFault{}
				require.NoError(t, f.k.SetVerificationRound(f.ctx, round))
			})
		}
	}
}

// This is a local keeper mapping test with an explicitly supplied result. It
// does not assert that any particular reviewers or ante-signed tx produced it.
func TestRecordAcceptedRelationRetainsDeclaredMethod(t *testing.T) {
	f := setupHandoff(t)
	require.NoError(t, f.k.EnableRecordIntegrity(f.ctx))
	params := types.DefaultParams()
	require.NoError(t, f.k.SetParams(f.ctx, &params))
	claim := &types.Claim{Id: "edge-claim", FactContent: "A public fixture has a declared derivation edge.", Domain: "physics", Category: "empirical", Submitter: "author", Stake: "200000", MethodId: "M-COMPUTATIONAL", ReasoningTrace: "Exact declared trace.", Relations: []*types.ClaimRelation{{TargetFactId: "fact", Relation: types.RelationType_RELATION_TYPE_SUPPORTS, MethodId: "M-FORMAL", Inference: types.InferenceType_INFERENCE_TYPE_DEDUCTIVE, InferenceStrengthBps: 900000}}}
	require.NoError(t, f.k.SetClaim(f.ctx, claim))
	round := &types.VerificationRound{Id: "edge-round", ClaimId: claim.Id, Phase: types.VerificationPhase_VERIFICATION_PHASE_AGGREGATION, CommitmentScheme: 2, CommitmentChainId: "source-chain"}
	require.NoError(t, f.k.SetVerificationRound(f.ctx, round))
	id, err := f.k.createFactFromClaim(f.ctx, claim, round, 800000)
	require.NoError(t, err)
	fact, found := f.k.GetFact(f.ctx, id)
	require.True(t, found)
	require.Equal(t, claim.MethodId, fact.MethodId)
	require.Equal(t, claim.ReasoningTrace, fact.ReasoningTrace)
	relations, err := f.k.GetFactRelations(f.ctx, id)
	require.NoError(t, err)
	require.Len(t, relations, 1)
	require.Equal(t, "M-FORMAL", relations[0].MethodId)
	require.Equal(t, claim.Relations[0].Inference, relations[0].Inference)
}
