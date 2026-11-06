package keeper_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/zerone-chain/zerone/x/emergency/keeper"
	"github.com/zerone-chain/zerone/x/emergency/types"
	govmodule "github.com/zerone-chain/zerone/x/gov"
	govtypes "github.com/zerone-chain/zerone/x/gov/types"
)

// Unknown evidence is a forward-gate veto, never a veto of an independently
// justified rollback. Exercise admission and due execution through real entries.
func TestHaltReasonUnknownDoesNotVetoJustifiedRollback(t *testing.T) {
	for _, evidence := range []string{"prose", "malformed_payload", "malformed_envelope", "mismatched_id", "old_non_halt_key_mismatch"} {
		for _, justification := range []string{"none", "gridlock", "research_fund"} {
			for _, unknownFirst := range []bool{false, true} {
				for _, entry := range []string{"rollback", "begin", "query"} {
					t.Run(fmt.Sprintf("%s/%s/unknownFirst=%t/%s", evidence, justification, unknownFirst, entry), func(t *testing.T) {
						k, g, ctx, key := setupHaltReasonKeepers(t)
						unknownID, validID := "z-unknown", "a-valid"
						if unknownFirst {
							unknownID, validID = "a-unknown", "z-valid"
						}
						switch evidence {
						case "prose":
							storeHaltReasonCeremony(t, k, ctx, unknownID, "validator double-sign incident", "finalized", 45_000_000)
						case "malformed_payload":
							require.NoError(t, k.SetCeremony(ctx, &types.EmergencyCeremony{Id: unknownID, Type: "halt", Phase: "finalized", StartBlock: 45_000_000, ProposalData: []byte{0xff}}))
						case "malformed_envelope":
							ctx.KVStore(key).Set(types.CeremonyKey(unknownID), []byte{0xff})
						case "old_non_halt_key_mismatch":
							require.NoError(t, k.SetCeremony(ctx, &types.EmergencyCeremony{Id: "wrong-envelope-id", Type: "resume", Phase: "finalized", StartBlock: 100}))
							ctx.KVStore(key).Set(types.CeremonyKey(unknownID), ctx.KVStore(key).Get(types.CeremonyKey("wrong-envelope-id")))
							ctx.KVStore(key).Delete(types.CeremonyKey("wrong-envelope-id"))
						case "mismatched_id":
							storeHaltReasonCeremony(t, k, ctx, unknownID, types.HaltReasonResearchFund, "finalized", 45_000_000)
							ceremony, found := k.GetCeremony(ctx, unknownID)
							require.True(t, found)
							ceremony.Id = unknownID + "-wrong-payload-id"
							require.NoError(t, k.SetCeremony(ctx, ceremony))
							ctx.KVStore(key).Delete(types.CeremonyKey(unknownID))
						}
						if justification == "gridlock" {
							for i := 0; i < govtypes.RollbackGridlockThreshold; i++ {
								g.SetResearchSpendProposal(ctx, &govtypes.ResearchSpendProposal{ProposalId: uint64(i + 1), Stage: string(govtypes.ResearchStageExpired)})
							}
						}
						if justification == "research_fund" {
							storeHaltReasonCeremony(t, k, ctx, validID, types.HaltReasonResearchFund, "finalized", 45_000_000)
						}
						require.NoError(t, k.EnableHaltReasonScoping(ctx))
						before := haltReasonStoreBytes(t, ctx, key)
						count, unknown := keeper.NewGovEmergencyAdapter(k).CountResearchFundHalts(ctx, 40_000_000)
						require.True(t, unknown)
						wantCount := uint64(0)
						if justification == "research_fund" {
							wantCount = 1
						}
						require.Equal(t, wantCount, count)
						if entry == "begin" {
							g.SetPhaseTransitionMeta(ctx, &govtypes.PhaseTransitionProposal{LipID: "rollback", Stage: govtypes.PhaseTransitionStagePending, IsRollback: true, ActivationBlock: uint64(ctx.BlockHeight()), TargetPhase: govtypes.ResearchFundPhase_RESEARCH_FUND_PHASE_OBSERVER})
							require.NoError(t, govmodule.NewAppModule(nil, g).BeginBlock(ctx))
							meta, found := g.GetPhaseTransitionMeta(ctx, "rollback")
							require.True(t, found)
							if justification == "none" {
								require.Equal(t, govtypes.PhaseTransitionStageCancelled, meta.Stage)
								require.Equal(t, govtypes.ResearchFundPhase_RESEARCH_FUND_PHASE_BALANCED, g.GetResearchFundPhase(ctx))
							} else {
								require.Equal(t, govtypes.PhaseTransitionStageActivated, meta.Stage)
								require.Equal(t, govtypes.ResearchFundPhase_RESEARCH_FUND_PHASE_OBSERVER, g.GetResearchFundPhase(ctx))
							}
						} else {
							response, err := haltEntry(g, ctx, entry)
							if entry == "rollback" && justification == "none" {
								require.ErrorIs(t, err, types.ErrHaltEvidence)
							} else {
								require.NoError(t, err)
							}
							if entry == "query" {
								require.Equal(t, wantCount+1, response.(*govtypes.QueryResearchFundGovernanceResponse).CurrentConditions.EmergencyHaltsFromMisuse)
								_, met := g.WithHaltReasonScoping(true).CheckPhaseExitConditions(ctx)
								require.False(t, met)
							}
						}
						require.Equal(t, before, haltReasonStoreBytes(t, ctx, key), "evidence is read-only")
					})
				}
			}
		}
	}
}
