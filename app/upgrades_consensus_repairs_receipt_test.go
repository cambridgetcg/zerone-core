package app

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"testing"

	"cosmossdk.io/core/header"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/stretchr/testify/require"
	cc "github.com/zerone-chain/zerone/x/capture_challenge/types"
	cd "github.com/zerone-chain/zerone/x/capture_defense/types"
	emergency "github.com/zerone-chain/zerone/x/emergency/types"
	gov "github.com/zerone-chain/zerone/x/gov/types"
	knowledge "github.com/zerone-chain/zerone/x/knowledge/types"
	ontology "github.com/zerone-chain/zerone/x/ontology/types"
	qualification "github.com/zerone-chain/zerone/x/qualification/types"
	bridge "github.com/zerone-chain/zerone/x/substrate_bridge/types"
)

func TestConsensusRepairsReceiptSummarizesEveryCensus(t *testing.T) {
	app, ctx, _ := consensusRepairsFixture(t)
	const count = 100
	for i := 1; i <= count; i++ {
		id := fmt.Sprintf("record-%03d", i)
		repairPutJSON(t, app, ctx, "zerone_gov", gov.SeatElectionKey(uint64(i)), &gov.SeatElectionProposal{ProposalId: uint64(i), Stage: "voting"})
		repairPutProto(t, app, ctx, "knowledge", knowledge.AugmentationBountyKey(id+"\x00"), &knowledge.AugmentationBounty{Id: id + "\x00"})
		repairPutProto(t, app, ctx, "capture_defense", []byte("\x05physics/"+id), &cd.VerificationHistoryEntry{Domain: "physics", RoundId: id, SubmitBlocks: []uint64{1, 2}})
		repairPutProto(t, app, ctx, "qualification", []byte("\x01validator/"+id), &qualification.DomainQualification{Validator: "validator", Domain: id, Status: qualification.QualificationStatus_QUALIFICATION_STATUS_EXPIRED, StakedAmount: "42"})
		repairPutProto(t, app, ctx, "capture_challenge", []byte("\x02"+id), &cc.DomainBountyPool{Domain: id, Balance: "0"})
		// Distinct arbitrary statuses must not leak a record-sized histogram into the receipt.
		repairPutProto(t, app, ctx, "zerone_ontology", []byte("\x02"+id), &ontology.Domain{Name: id, Status: id})
		repairPutProto(t, app, ctx, "emergency", emergency.CeremonyKey(id), &emergency.EmergencyCeremony{Id: id, Type: string(emergency.CeremonyHalt), Phase: string(emergency.PhaseFinalized), StartBlock: 3, ProposalData: []byte{0xff}})
		repairPutJSON(t, app, ctx, "zerone_gov", []byte("\x14"+id), &gov.PhaseTransitionProposal{LipID: id, TargetPhase: gov.ResearchFundPhase_RESEARCH_FUND_PHASE_GENESIS_PAIR, IsRollback: true, Stage: gov.PhaseTransitionStagePending})
		repairPutJSON(t, app, ctx, "zerone_gov", gov.LIPKey(id), &gov.LIP{Id: id, Category: gov.CategoryPhaseRollback})
		repairPutProto(t, app, ctx, "substrate_bridge", bridge.AdapterKey(id), &bridge.AdapterRegistration{AdapterId: id, RequiredQualificationDomain: "physics", MinQualificationStatus: bridge.QualificationStatus_QUALIFICATION_STATUS_DISTINGUISHED})
		key := append([]byte("\x02channel-0\x00"), binary.BigEndian.AppendUint64(nil, uint64(i))...)
		repairPutJSON(t, app, ctx, "zerone_ibcrl", key, map[string]interface{}{"channel_id": "channel-0", "sequence": i, "denom": "uzrn", "amount": "1"})
	}
	ctx.KVStore(app.keys["zerone_gov"]).Set([]byte{0x12}, binary.BigEndian.AppendUint64(nil, count+1))
	height := app.CommitMultiStore().Commit().Version + 1
	plan := upgradetypes.Plan{Name: UpgradeNameConsensusRepairsV1, Height: height}
	require.NoError(t, app.UpgradeKeeper.ScheduleUpgrade(ctx, plan))
	ctx = ctx.WithBlockHeight(height).WithHeaderInfo(header.Info{Height: height})
	report, err := app.CensusConsensusRepairsV1(ctx)
	require.NoError(t, err)
	compact, err := report.receipt()
	require.NoError(t, err)
	full, err := json.Marshal(report)
	require.NoError(t, err)
	require.Len(t, report.PendingRollbacks, count)
	require.Equal(t, 1, bytes.Count(full, []byte(`"lip_id":"record-001"`)), "rollback appears once, not again in findings")
	for _, finding := range report.Findings {
		require.NotContains(t, finding.Reason, "pending legacy rollback")
	}
	require.Contains(t, string(full), "EXPIRED retained amount=42")
	require.Contains(t, string(full), "unsupported qualification minimum retained")
	require.Contains(t, string(full), "NUL bounty ID")
	require.Contains(t, string(full), "out-of-window finalized halt retained")
	require.Len(t, compact.Censuses, len(consensusRepairsCensusNames))
	for _, name := range consensusRepairsCensusNames {
		summary, ok := compact.Censuses[name]
		require.True(t, ok, name)
		require.GreaterOrEqual(t, summary.Records, count, name)
		require.Equal(t, report.Censuses[name], summary.ConsensusRepairsSize, name)
		require.Equal(t, report.Digests[name], summary.Digest, name)
		held := 0
		for _, finding := range report.Findings {
			if finding.Census == name {
				held++
			}
		}
		require.Equal(t, held, summary.HeldFindings, name)
		require.Zero(t, summary.Blockers, name)
		if name == "halt_reasons" {
			require.Equal(t, count, summary.PendingRollbacks)
		} else {
			require.Zero(t, summary.PendingRollbacks, name)
		}
	}
	require.NoError(t, app.UpgradeKeeper.ApplyUpgrade(ctx, plan))
	raw := app.KnowledgeKeeper.ReadMigrationMarker(ctx, consensusRepairsReceiptKey)
	var receipt consensusRepairsReceipt
	require.NoError(t, json.Unmarshal([]byte(raw), &receipt))
	require.Equal(t, compact, receipt.Census, "the handler writes the compact form, not the offline report")
	for _, field := range []string{`"findings":`, `"domain_statuses":`, `"lip_id":`, `"key_hex":`, `"reason":`, `"census_reads":`} {
		require.NotContains(t, raw, field)
	}
	require.Less(t, len(raw), 8<<10)
	app.CommitMultiStore().Commit()
	require.NoError(t, app.ValidateAccountingAuthorityStartup())
	t.Logf("synthetic all-census report=%d bytes receipt=%d bytes", len(full), len(raw))
}

func TestConsensusRepairsReceiptSizeIndependentOfHeldRecords(t *testing.T) {
	var sizes []int
	for _, count := range []int{0, 20_000} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			app, ctx, _ := consensusRepairsFixture(t)
			for i := 0; i < count; i++ {
				id := fmt.Sprintf("bounty\x00%05d", i)
				repairPutProto(t, app, ctx, "knowledge", knowledge.AugmentationBountyKey(id), &knowledge.AugmentationBounty{Id: id})
			}
			height := app.CommitMultiStore().Commit().Version + 1
			plan := upgradetypes.Plan{Name: UpgradeNameConsensusRepairsV1, Height: height}
			require.NoError(t, app.UpgradeKeeper.ScheduleUpgrade(ctx, plan))
			ctx = ctx.WithBlockHeight(height).WithHeaderInfo(header.Info{Height: height})
			report, err := app.CensusConsensusRepairsV1(ctx)
			require.NoError(t, err)
			held := 0
			for _, finding := range report.Findings {
				if finding.Census == "reformulations" && finding.Key != "" {
					held++
				}
			}
			require.Equal(t, count, held, "offline details are not truncated")
			require.NoError(t, app.UpgradeKeeper.ApplyUpgrade(ctx, plan))
			raw := app.KnowledgeKeeper.ReadMigrationMarker(ctx, consensusRepairsReceiptKey)
			var receipt consensusRepairsReceipt
			require.NoError(t, json.Unmarshal([]byte(raw), &receipt))
			require.Equal(t, count+1, receipt.Census.Censuses["reformulations"].HeldFindings)
			require.Equal(t, report.Digests["reformulations"], receipt.Census.Censuses["reformulations"].Digest)
			require.Less(t, len(raw), 8<<10)
			sizes = append(sizes, len(raw))
			t.Logf("synthetic held records=%d receipt=%d bytes", count, len(raw))
		})
	}
	require.Len(t, sizes, 2)
	require.Less(t, sizes[1]-sizes[0], 128, "only decimal count widths may grow, never per-record entries")
}
