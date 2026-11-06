package cross_stack_test

import (
	"bytes"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	emergencytypes "github.com/zerone-chain/zerone/x/emergency/types"
	govkeeper "github.com/zerone-chain/zerone/x/gov/keeper"
	govtypes "github.com/zerone-chain/zerone/x/gov/types"
)

// Use the application's real emergency adapter, bank and module registration;
// ceremonies and expired proposals are synthetic retained-history fixtures.
func TestHaltReasonOnAcrossApplication(t *testing.T) {
	for _, justification := range []string{"none", "gridlock", "research_fund"} {
		t.Run(justification, func(t *testing.T) {
			h := NewTestHarness(t)
			on, err := h.EmergencyKeeper.HaltReasonScopingEnabled(h.Ctx)
			require.NoError(t, err)
			require.True(t, on, "DefaultGenesis selects the marker through module InitGenesis")
			require.NoError(t, h.EmergencyKeeper.EnableHaltReasonScoping(h.Ctx))
			h.GovKeeper.SetResearchFundGovernanceState(h.Ctx, &govtypes.ResearchFundGovernanceState{CurrentPhase: govtypes.ResearchFundPhase_RESEARCH_FUND_PHASE_BALANCED, PhaseStartedAtBlock: uint64(h.Height())})
			reasons := []string{"validator double-sign incident"}
			if justification == "research_fund" {
				reasons = append(reasons, emergencytypes.HaltReasonResearchFund)
			}
			for _, reason := range reasons {
				payload, err := proto.Marshal(&emergencytypes.EmergencyHaltProposal{Id: reason, Reason: reason})
				require.NoError(t, err)
				require.NoError(t, h.EmergencyKeeper.SetCeremony(h.Ctx, &emergencytypes.EmergencyCeremony{Id: reason, Type: "halt", Phase: "finalized", StartBlock: uint64(h.Height()), ProposalData: payload}))
			}
			if justification == "gridlock" {
				for i := 0; i < govtypes.RollbackGridlockThreshold; i++ {
					h.GovKeeper.SetResearchSpendProposal(h.Ctx, &govtypes.ResearchSpendProposal{ProposalId: uint64(i + 1), Stage: string(govtypes.ResearchStageExpired)})
				}
			}
			query, err := govkeeper.NewQueryServerImpl(h.GovKeeper).ResearchFundGovernance(h.Ctx, &govtypes.QueryResearchFundGovernanceRequest{})
			require.NoError(t, err)
			require.Equal(t, uint64(len(reasons)), query.CurrentConditions.EmergencyHaltsFromMisuse)
			proposer := sdk.AccAddress(bytes.Repeat([]byte{42}, 20))
			require.NoError(t, h.FundAccount(proposer, sdk.NewCoins(sdk.NewInt64Coin("uzrn", 1_000_000_000_000))))
			// Cache models transaction rollback on rejection; no network or signing.
			cache, write := h.Ctx.CacheContext()
			response, err := govkeeper.NewMsgServerImpl(h.GovKeeper).SubmitLIP(cache, &govtypes.MsgSubmitLIP{Proposer: proposer.String(), Title: "justified rollback", Category: govtypes.CategoryPhaseRollback, InitialStake: "1000000000000"})
			if justification == "none" {
				require.ErrorIs(t, err, emergencytypes.ErrHaltEvidence)
			} else {
				require.NoError(t, err)
				write()
				meta, found := h.GovKeeper.GetPhaseTransitionMeta(h.Ctx, response.LipId)
				require.True(t, found)
				meta.ActivationBlock = uint64(h.Height())
				h.GovKeeper.SetPhaseTransitionMeta(h.Ctx, meta)
			}
			if justification == "none" {
				h.GovKeeper.SetPhaseTransitionMeta(h.Ctx, &govtypes.PhaseTransitionProposal{LipID: "LIP-1", IsRollback: true, Stage: govtypes.PhaseTransitionStagePending, TargetPhase: govtypes.ResearchFundPhase_RESEARCH_FUND_PHASE_OBSERVER, ActivationBlock: uint64(h.Height())})
			}
			_, err = h.App.BeginBlocker(h.Ctx)
			require.NoError(t, err)
			meta, found := h.GovKeeper.GetPhaseTransitionMeta(h.Ctx, "LIP-1")
			require.True(t, found)
			if justification == "none" {
				require.Equal(t, govtypes.PhaseTransitionStageCancelled, meta.Stage)
				require.Equal(t, govtypes.ResearchFundPhase_RESEARCH_FUND_PHASE_BALANCED, h.GovKeeper.GetResearchFundPhase(h.Ctx))
			} else {
				require.Equal(t, govtypes.PhaseTransitionStageActivated, meta.Stage)
				require.Equal(t, govtypes.ResearchFundPhase_RESEARCH_FUND_PHASE_OBSERVER, h.GovKeeper.GetResearchFundPhase(h.Ctx))
			}
		})
	}
}
