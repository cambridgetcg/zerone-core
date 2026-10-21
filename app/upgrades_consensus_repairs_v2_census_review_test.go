package app

import (
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"testing"

	sdkmath "cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/require"
	gov "github.com/zerone-chain/zerone/x/gov/types"
	knowledgekeeper "github.com/zerone-chain/zerone/x/knowledge/keeper"
	knowledge "github.com/zerone-chain/zerone/x/knowledge/types"
)

func TestConsensusRepairsV2C2NeverLowersImportedOrWrappedScores(t *testing.T) {
	imported := &knowledge.AgentCalibration{Address: "zrn1imported", TotalSubmissions: 1, Accepted: 1, DisprovenCount: 1, CalibrationScoreBps: 1_000_000, LastUpdatedBlock: 42}
	wrapped := &knowledge.AgentCalibration{Address: "zrn1wrap", TotalSubmissions: 1, Accepted: 1, DisprovenCount: 18_446_744_073_710, LastUpdatedBlock: 43}
	wrapped.CalibrationScoreBps = knowledgekeeper.ComputeAgentCalibrationScore(wrapped)
	require.Equal(t, uint64(551_616), wrapped.CalibrationScoreBps, "legacy uint64 product wraps")
	require.Zero(t, knowledgekeeper.ComputeSmoothedAgentCalibrationScore(wrapped))
	challenger := &knowledge.AgentCalibration{Address: "challenger-only", ChallengesIssued: 3, CalibrationScoreBps: 100}
	raised := &knowledge.AgentCalibration{Address: "raised", TotalSubmissions: 1, Accepted: 1, DisprovenCount: 1}
	unchanged := &knowledge.AgentCalibration{Address: "unchanged", TotalSubmissions: 1, Accepted: 1, CalibrationScoreBps: 1_000_000}
	for _, held := range []*knowledge.AgentCalibration{imported, wrapped, challenger} {
		t.Run(held.Address, func(t *testing.T) {
			a, ctx, _ := consensusRepairsV2Fixture(t, held, raised, unchanged)
			key := knowledge.AgentCalibrationKey(held.Address)
			before := claimIntegrityAllStores(t, a, ctx)
			report, updates, err := a.censusConsensusRepairsV2(ctx, false)
			require.NoError(t, err)
			require.Len(t, updates, 1)
			require.Equal(t, "raised", updates[0].Address)
			found := false
			for _, f := range report.Findings {
				if f.Census == "q5_calibration" && f.Key == hex.EncodeToString(key) {
					found = true
					require.Equal(t, "held", f.Outcome)
					require.Contains(t, f.Reason, "C2 hold")
					require.Contains(t, f.Reason, fmt.Sprintf("stored_score=%d smoothed_score=%d", held.CalibrationScoreBps, knowledgekeeper.ComputeSmoothedAgentCalibrationScore(held)))
				}
			}
			require.True(t, found)
			require.Equal(t, before, claimIntegrityAllStores(t, a, ctx))
			// A write attempt to the held key must fail; byte equality alone
			// would miss a gratuitous identical write.
			service := claimRecordFaultService{runtime.NewKVStoreService(a.keys["knowledge"]), "set", key, true}
			a.KnowledgeKeeper = knowledgekeeper.NewKeeper(service, a.appCodec, a.KnowledgeKeeper.GetAuthority(), a.BankKeeper, a.KnowledgeKeeper.GetStakingKeeper())
			ctx, plan := scheduleRepairsV2(t, a, ctx)
			require.NoError(t, a.ValidateAccountingAuthorityStartup())
			require.NoError(t, a.UpgradeKeeper.ApplyUpgrade(ctx, plan))
			require.Equal(t, before["knowledge:"+string(key)], ctx.KVStore(a.keys["knowledge"]).Get(key))
			got, ok := a.KnowledgeKeeper.GetAgentCalibration(ctx, "raised")
			require.True(t, ok)
			require.Equal(t, uint64(500_000), got.CalibrationScoreBps)
		})
	}
}

func TestConsensusRepairsV2Q5CountsFactsWithoutChargingBudget(t *testing.T) {
	a, ctx, _ := consensusRepairsV2Fixture(t)
	base, err := a.CensusConsensusRepairsV2(ctx)
	require.NoError(t, err)
	var legacyBase, presentBase, unreadableBase int
	for _, f := range base.Findings {
		if f.Census == "q5_calibration" && f.Key == "" {
			_, err := fmt.Sscanf(f.Reason, "retained fact snapshots (streamed): legacy_or_unknown_zero=%d present_zero=%d unreadable=%d", &legacyBase, &presentBase, &unreadableBase)
			require.NoError(t, err)
		}
	}
	const count = 50_001 // Combined history exceeded Q5's old 100k row budget.
	for i := 0; i < count; i++ {
		id := fmt.Sprintf("stream-%06d", i)
		repairPutProto(t, a, ctx, "knowledge", knowledge.AgentCalibrationKey(id), &knowledge.AgentCalibration{Address: id})
		repairPutProto(t, a, ctx, "knowledge", append([]byte{1}, []byte(id)...), &knowledge.Fact{Id: id, HasCalibrationSnapshot: i%2 == 0})
	}
	report, err := a.CensusConsensusRepairsV2(ctx)
	require.NoError(t, err)
	require.Equal(t, count, report.Censuses["q5_calibration"].Records)
	require.Len(t, report.Reads["q5_calibration"], 1)
	require.Equal(t, "5a", report.Reads["q5_calibration"][0].Prefix)
	found := false
	for _, f := range report.Findings {
		if f.Census == "q5_calibration" && f.Key == "" {
			found = true
			require.Contains(t, f.Reason, fmt.Sprintf("legacy_or_unknown_zero=%d present_zero=%d unreadable=%d", legacyBase+count/2, presentBase+(count+1)/2, unreadableBase))
		}
	}
	require.True(t, found)
	// PR37 still retains facts under its own independent budget.
	require.Equal(t, base.Censuses["pr37_support"].Records+count, report.Censuses["pr37_support"].Records)
}

func TestConsensusRepairsV2ReachableFX7DeficitHeldNotActivationBlocker(t *testing.T) {
	a, ctx, _ := consensusRepairsV2Fixture(t)
	sponsor, recipient := settlementAddress(231), settlementAddress(232)
	fund := authtypes.NewModuleAddress(knowledge.TrainingFundModuleName)
	require.True(t, a.BankKeeper.GetBalance(ctx, fund, "uzrn").IsZero())
	coins := sdk.NewCoins(sdk.NewInt64Coin("uzrn", 300))
	require.NoError(t, a.BankKeeper.MintCoins(ctx, knowledge.BootstrapFundModuleName, coins))
	require.NoError(t, a.BankKeeper.SendCoinsFromModuleToAccount(ctx, knowledge.BootstrapFundModuleName, sponsor, coins))
	for _, id := range []string{"paid", "retained"} {
		require.NoError(t, a.KnowledgeKeeper.SetAugmentationBounty(ctx, &knowledge.AugmentationBounty{Id: id, SponsorAddress: sponsor.String(), RewardPerVariant: 100, MaxVariants: 1, Active: true}))
		require.NoError(t, a.KnowledgeKeeper.EscrowAugmentationBounty(ctx, sponsor.String(), id, sdkmath.NewInt(150)))
	}
	params, err := a.KnowledgeKeeper.GetParams(ctx)
	require.NoError(t, err)
	params.ReformulationSuperiorBonusBps = 1_000_000
	require.NoError(t, a.KnowledgeKeeper.SetParams(ctx, params))
	// The live payout path accepts a 200 payout after a 150 escrow; the
	// retained second bounty then has 150 locked against only 100 in bank.
	require.NoError(t, a.KnowledgeKeeper.ReleaseAugmentationPayout(ctx, "paid", recipient.String(), sdkmath.NewInt(200)))
	ctx, plan := scheduleRepairsV2(t, a, ctx)
	before := claimIntegrityAllStores(t, a, ctx)
	offline, err := a.CensusConsensusRepairsV2(ctx)
	require.ErrorContains(t, err, "FX7 underbacked")
	require.NotContains(t, err.Error(), "canonical non-negative")
	held, _, err := a.censusConsensusRepairsV2(ctx, false)
	require.NoError(t, err)
	found := false
	for _, f := range held.Findings {
		if f.Census == "fx7_accounting" && f.Module == "bank" {
			found = true
			require.Equal(t, "held", f.Outcome)
			require.NotEmpty(t, f.Key)
			require.Contains(t, f.Reason, "balance_uzrn=100 escrow_uzrn=150 deficit_uzrn=50")
			require.Contains(t, f.Reason, hex.EncodeToString(knowledge.TrainingFundEscrowLockedKey("retained")))
		}
	}
	require.True(t, found)
	require.Equal(t, len(offline.Findings), len(held.Findings))
	require.Equal(t, before, claimIntegrityAllStores(t, a, ctx))
	require.Panics(t, func() { a.KnowledgeKeeper.ExportGenesis(ctx) }, "repaired export must still refuse")
	require.NoError(t, a.ValidateAccountingAuthorityStartup())
	require.NoError(t, a.UpgradeKeeper.ApplyUpgrade(ctx, plan))
	var receipt consensusRepairsReceipt
	require.NoError(t, json.Unmarshal([]byte(a.KnowledgeKeeper.ReadMigrationMarker(ctx, consensusRepairsV2ReceiptKey)), &receipt))
	require.Equal(t, 2, receipt.Census.Censuses["fx7_accounting"].HeldFindings)
	require.Zero(t, receipt.Census.Censuses["fx7_accounting"].Blockers)
	after := claimIntegrityAllStores(t, a, ctx)
	for key, value := range before {
		if key[:min(8, len(key))] != "upgrade:" {
			require.Equal(t, value, after[key], "held bytes %x", key)
		}
	}
	a.CommitMultiStore().Commit()
	require.NoError(t, a.ValidateAccountingAuthorityStartup())
	require.Panics(t, func() { a.KnowledgeKeeper.ExportGenesis(ctx) })
}

func TestConsensusRepairsV2FX8ReportsExactFaultKeysAndCounter(t *testing.T) {
	a, ctx, _ := consensusRepairsV2Fixture(t)
	store := ctx.KVStore(a.keys["zerone_gov"])
	keys := [][]byte{gov.ResearchSpendKey(7), gov.ResearchSpendCounterKey, gov.DistinctVoterKey("bad"), gov.ResearchCommunityVoteKey(9, "missing")}
	store.Set(keys[0], []byte(`{"proposal_id":7,"amount":"0"}`))
	store.Set(keys[1], binary.BigEndian.AppendUint64(nil, ^uint64(0)-1))
	store.Set(keys[2], binary.BigEndian.AppendUint64(nil, 4))
	store.Set(keys[3], []byte("yes"))
	held, _, err := a.censusConsensusRepairsV2(ctx, false)
	require.NoError(t, err)
	seen := map[string]bool{}
	for _, f := range held.Findings {
		if f.Census == "fx8_retained" && f.Key != "" {
			require.Equal(t, "held", f.Outcome)
			seen[f.Key] = true
			if f.Key == "08" {
				require.Contains(t, f.Reason, "stored=18446744073709551614")
			}
		}
	}
	for _, key := range keys {
		require.True(t, seen[hex.EncodeToString(key)], "missing fault key %x", key)
	}
	_, err = a.CensusConsensusRepairsV2(ctx)
	require.Error(t, err)
	require.Panics(t, func() { a.ZeroneGovKeeper.ExportGenesis(ctx) })
	ctx, plan := scheduleRepairsV2(t, a, ctx)
	require.NoError(t, a.ValidateAccountingAuthorityStartup())
	require.NoError(t, a.UpgradeKeeper.ApplyUpgrade(ctx, plan))
}
