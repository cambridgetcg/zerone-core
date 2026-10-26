package integration_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	emergencytypes "github.com/zerone-chain/zerone/x/emergency/types"
	govmodule "github.com/zerone-chain/zerone/x/gov"
	govkeeper "github.com/zerone-chain/zerone/x/gov/keeper"
	govtypes "github.com/zerone-chain/zerone/x/gov/types"
)

func TestGovernanceHaltReasonSelection(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		for _, justification := range []string{"none", "gridlock", "research_fund"} {
			t.Run(fmt.Sprintf("enabled=%t/%s", enabled, justification), func(t *testing.T) {
				h := setupGovHarness(t, 50_000_000)
				h.mockEK.haltReasonScoping = enabled
				h.mockEK.unknownHaltEvidence = true
				if justification == "research_fund" {
					h.mockEK.halts[emergencytypes.HaltReasonResearchFund] = 1
				}
				h.keeper.SetResearchFundGovernanceState(h.ctx, &govtypes.ResearchFundGovernanceState{CurrentPhase: govtypes.ResearchFundPhase_RESEARCH_FUND_PHASE_BALANCED, PhaseStartedAtBlock: 40_000_000})
				if justification == "gridlock" {
					for i := 0; i < govtypes.RollbackGridlockThreshold; i++ {
						h.keeper.SetResearchSpendProposal(h.ctx, &govtypes.ResearchSpendProposal{ProposalId: uint64(i + 1), Stage: string(govtypes.ResearchStageExpired)})
					}
				}
				query, err := govkeeper.NewQueryServerImpl(h.keeper).ResearchFundGovernance(h.ctx, &govtypes.QueryResearchFundGovernanceRequest{})
				require.NoError(t, err)
				wantGate := h.mockEK.halts[emergencytypes.HaltReasonResearchFund]
				if enabled {
					wantGate++
				}
				require.Equal(t, wantGate, query.CurrentConditions.EmergencyHaltsFromMisuse)
				_, err = govkeeper.NewMsgServerImpl(h.keeper).SubmitLIP(h.ctx, &govtypes.MsgSubmitLIP{Title: "rollback", Proposer: h.voter1, Category: govtypes.CategoryPhaseRollback, InitialStake: "1000000000000"})
				if justification == "none" {
					if enabled {
						require.ErrorIs(t, err, emergencytypes.ErrHaltEvidence)
					} else {
						require.ErrorIs(t, err, govtypes.ErrNoRollbackJustification)
					}
				} else {
					require.NoError(t, err)
				}
				// Seed a due legacy-admitted rollback to test the actual module hook.
				h.keeper.SetPhaseTransitionMeta(h.ctx, &govtypes.PhaseTransitionProposal{LipID: "LIP-1", IsRollback: true, Stage: govtypes.PhaseTransitionStagePending, TargetPhase: govtypes.ResearchFundPhase_RESEARCH_FUND_PHASE_OBSERVER, ActivationBlock: uint64(h.ctx.BlockHeight())})
				require.NoError(t, govmodule.NewAppModule(nil, h.keeper).BeginBlock(h.ctx))
				meta, found := h.keeper.GetPhaseTransitionMeta(h.ctx, "LIP-1")
				require.True(t, found)
				if enabled && justification == "none" {
					require.Equal(t, govtypes.PhaseTransitionStageCancelled, meta.Stage)
				} else {
					require.Equal(t, govtypes.PhaseTransitionStageActivated, meta.Stage)
				}
			})
		}
	}
}
