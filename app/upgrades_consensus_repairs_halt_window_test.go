package app

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	emergencykeeper "github.com/zerone-chain/zerone/x/emergency/keeper"
	emergency "github.com/zerone-chain/zerone/x/emergency/types"
	gov "github.com/zerone-chain/zerone/x/gov/types"
	"google.golang.org/protobuf/proto"
)

func TestConsensusRepairsHaltPayloadsOnlyDecodedInsideWindow(t *testing.T) {
	for _, start := range []uint64{9, 10, 20, 21} {
		for _, fault := range []string{"undecodable", "wrong identity"} {
			t.Run(fmt.Sprintf("start=%d/%s", start, fault), func(t *testing.T) {
				app, ctx, _ := consensusRepairsFixture(t)
				ctx = ctx.WithBlockHeight(20)
				app.ZeroneGovKeeper.SetResearchFundGovernanceState(ctx, &gov.ResearchFundGovernanceState{
					CurrentPhase: gov.ResearchFundPhase_RESEARCH_FUND_PHASE_OBSERVER, PhaseStartedAtBlock: 10,
				})
				payload := []byte{0xff}
				if fault == "wrong identity" {
					var err error
					payload, err = proto.Marshal(&emergency.EmergencyHaltProposal{Id: "other", Reason: emergency.HaltReasonResearchFund})
					require.NoError(t, err)
				}
				repairPutProto(t, app, ctx, "emergency", emergency.CeremonyKey("halt"), &emergency.EmergencyCeremony{
					Id: "halt", Type: string(emergency.CeremonyHalt), Phase: string(emergency.PhaseFinalized), StartBlock: start, ProposalData: payload,
				})
				before := claimIntegrityAllStores(t, app, ctx)
				count, unknown := emergencykeeper.NewGovEmergencyAdapter(app.EmergencyKeeper).CountResearchFundHalts(ctx, 10)
				require.Zero(t, count)
				inside := start >= 10 && start <= 20
				require.Equal(t, inside, unknown)
				report, err := app.CensusConsensusRepairsV1(ctx)
				if inside {
					require.ErrorContains(t, err, "census blocker")
				} else {
					require.NoError(t, err)
					found := false
					for _, finding := range report.Findings {
						if finding.Module == "emergency" && finding.Key != "" {
							require.Equal(t, "held", finding.Outcome)
							require.Contains(t, finding.Reason, "out-of-window finalized halt")
							found = true
						}
					}
					require.True(t, found)
				}
				require.Equal(t, before, claimIntegrityAllStores(t, app, ctx))
			})
		}
	}
}
