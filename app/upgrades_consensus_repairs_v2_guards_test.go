package app

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"testing"

	"cosmossdk.io/core/header"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/stretchr/testify/require"
	knowledgekeeper "github.com/zerone-chain/zerone/x/knowledge/keeper"
	knowledge "github.com/zerone-chain/zerone/x/knowledge/types"
)

func TestConsensusRepairsV2WriterFailuresDiscardScoresAndMarkers(t *testing.T) {
	for _, prefix := range [][]byte{knowledge.AgentCalibrationKey("z-later"), {0x92}, {0x93}, append([]byte{0x7f, 1}, []byte(consensusRepairsV2ReceiptKey)...)} {
		t.Run(fmt.Sprintf("key=%x", prefix), func(t *testing.T) {
			a, ctx, _ := consensusRepairsV2Fixture(t)
			seedRepairsV2Calibration(t, a, ctx)
			require.NoError(t, a.KnowledgeKeeper.SetAgentCalibration(ctx, &knowledge.AgentCalibration{Address: "z-later", TotalSubmissions: 2, Accepted: 2, DisprovenCount: 1, CalibrationScoreBps: 500_000}))
			ctx, plan := scheduleRepairsV2(t, a, ctx)
			service := claimRecordFaultService{runtime.NewKVStoreService(a.keys["knowledge"]), "set", prefix, true}
			a.KnowledgeKeeper = knowledgekeeper.NewKeeper(service, a.appCodec, a.KnowledgeKeeper.GetAuthority(), a.BankKeeper, a.KnowledgeKeeper.GetStakingKeeper())
			before := claimIntegrityAllStores(t, a, ctx)
			err := a.UpgradeKeeper.ApplyUpgrade(ctx, plan)
			require.ErrorContains(t, err, "injected claim-record write failure")
			require.Equal(t, before, claimIntegrityAllStores(t, a, ctx))
		})
	}
}

func TestConsensusRepairsV2SourceAndReceiptRefusals(t *testing.T) {
	for _, fault := range []string{"wrong chain", "wrong plan", "info", "wrong source", "extra version", "receipt", "done", "v1 marker", "v1 receipt", "wrong height", "unscheduled"} {
		t.Run(fault, func(t *testing.T) {
			a, ctx, _ := consensusRepairsV2Fixture(t)
			switch fault {
			case "wrong chain":
				ctx = ctx.WithChainID("zerone-1")
			case "wrong source":
				vm := consensusRepairsV2SourceVersionMap()
				vm["knowledge"] = 12
				require.NoError(t, a.UpgradeKeeper.SetModuleVersionMap(ctx, vm))
			case "extra version":
				require.NoError(t, a.UpgradeKeeper.SetModuleVersionMap(ctx, module.VersionMap{"extra": 1}))
			case "receipt":
				require.NoError(t, a.KnowledgeKeeper.WriteMigrationMarker(ctx, consensusRepairsV2ReceiptKey, "{}"))
			case "done":
				writeSDK053IBC10TestDoneHeight(a, ctx, UpgradeNameConsensusRepairsV2, 1)
			case "v1 marker":
				ctx.KVStore(a.keys["knowledge"]).Delete([]byte{0x90})
			case "v1 receipt":
				require.NoError(t, a.KnowledgeKeeper.WriteMigrationMarker(ctx, consensusRepairsReceiptKey, "{}"))
			}
			plan := upgradetypes.Plan{Name: UpgradeNameConsensusRepairsV2, Height: a.CommitMultiStore().Commit().Version + 1}
			if fault != "done" && fault != "unscheduled" {
				require.NoError(t, a.UpgradeKeeper.ScheduleUpgrade(ctx, plan))
			}
			ctx = ctx.WithBlockHeight(plan.Height).WithHeaderInfo(header.Info{Height: plan.Height})
			if fault == "wrong plan" {
				plan.Name = UpgradeNameConsensusRepairsV1
			}
			if fault == "info" {
				plan.Info = "extra"
			}
			if fault == "wrong height" {
				ctx = ctx.WithBlockHeight(plan.Height + 1)
			}
			before := claimIntegrityAllStores(t, a, ctx)
			require.Error(t, a.UpgradeKeeper.ApplyUpgrade(ctx, plan))
			require.Equal(t, before, claimIntegrityAllStores(t, a, ctx))
		})
	}
	for _, fault := range []string{"missing receipt", "missing done", "wrong chain", "wrong height", "marker count", "false readback", "missing census", "bad digest", "negative count", "blocker"} {
		t.Run("restart/"+fault, func(t *testing.T) {
			a, ctx, _ := consensusRepairsV2Fixture(t)
			ctx, plan := scheduleRepairsV2(t, a, ctx)
			require.NoError(t, a.UpgradeKeeper.ApplyUpgrade(ctx, plan))
			key := append([]byte{0x7f, 1}, []byte(consensusRepairsV2ReceiptKey)...)
			var receipt consensusRepairsReceipt
			require.NoError(t, json.Unmarshal(ctx.KVStore(a.keys["knowledge"]).Get(key), &receipt))
			switch fault {
			case "missing receipt":
				ctx.KVStore(a.keys["knowledge"]).Delete(key)
			case "missing done":
				doneKey := append([]byte{upgradetypes.DoneByte}, binary.BigEndian.AppendUint64(nil, uint64(plan.Height))...)
				doneKey = append(doneKey, []byte(plan.Name)...)
				ctx.KVStore(a.keys["upgrade"]).Delete(doneKey)
			case "wrong chain":
				receipt.ChainID = "wrong"
			case "wrong height":
				receipt.Height++
			case "marker count":
				receipt.Markers = receipt.Markers[:3]
			case "false readback":
				receipt.Markers[3].Enabled = false
			case "missing census":
				delete(receipt.Census.Censuses, "q5_calibration")
			case "bad digest":
				receipt.Census.Digest = "bad"
			case "negative count":
				s := receipt.Census.Censuses["q5_calibration"]
				s.Records = -1
				receipt.Census.Censuses["q5_calibration"] = s
			case "blocker":
				s := receipt.Census.Censuses["q5_calibration"]
				s.Blockers = 1
				receipt.Census.Censuses["q5_calibration"] = s
			}
			if fault != "missing receipt" && fault != "missing done" {
				raw, err := json.Marshal(receipt)
				require.NoError(t, err)
				ctx.KVStore(a.keys["knowledge"]).Set(key, raw)
			}
			a.CommitMultiStore().Commit()
			require.Error(t, a.ValidateAccountingAuthorityStartup())
		})
	}
}

func TestConsensusRepairsV2HeldHistoryDeterminismAndRecordBound(t *testing.T) {
	a, ctx, _ := consensusRepairsV2Fixture(t)
	store := ctx.KVStore(a.keys["knowledge"])
	repairPutProto(t, a, ctx, "knowledge", []byte("\x01zero"), &knowledge.Fact{Id: "zero"})
	repairPutProto(t, a, ctx, "knowledge", []byte("\x02old"), &knowledge.Claim{Id: "old", References: []string{"zero", "missing"}})
	repairPutProto(t, a, ctx, "knowledge", []byte("\x03old-round"), &knowledge.VerificationRound{Id: "old-round", ClaimId: "old", Phase: knowledge.VerificationPhase_VERIFICATION_PHASE_COMPLETE, Verdict: knowledge.Verdict_VERDICT_INCONCLUSIVE})
	repairPutProto(t, a, ctx, "knowledge", []byte("\x62inactive"), &knowledge.AugmentationBounty{Id: "inactive"})
	store.Set([]byte("\x6binactive"), []byte("0"))
	store.Set([]byte("\x35CasePreserved"), binary.BigEndian.AppendUint64(nil, 9))
	before := claimIntegrityAllStores(t, a, ctx)
	one, err := a.CensusConsensusRepairsV2(ctx)
	require.NoError(t, err)
	two, err := a.CensusConsensusRepairsV2(ctx)
	require.NoError(t, err)
	require.Equal(t, one, two)
	raw, err := json.Marshal(one)
	require.NoError(t, err)
	require.Contains(t, string(raw), "resolved_zero_support=1 missing_support=1")
	require.Contains(t, string(raw), "INCONCLUSIVE")
	require.Equal(t, before, claimIntegrityAllStores(t, a, ctx))
	ctx, plan := scheduleRepairsV2(t, a, ctx)
	require.NoError(t, a.UpgradeKeeper.ApplyUpgrade(ctx, plan))
	for _, key := range []string{"\x01zero", "\x02old", "\x03old-round", "\x62inactive", "\x6binactive", "\x35CasePreserved"} {
		require.Equal(t, before["knowledge:"+key], ctx.KVStore(a.keys["knowledge"]).Get([]byte(key)))
	}
	b, bctx, _ := consensusRepairsV2Fixture(t)
	for i := 0; i <= consensusRepairsMaxRecords; i++ {
		bctx.KVStore(b.keys["knowledge"]).Set(binary.BigEndian.AppendUint64([]byte{0x5a}, uint64(i)), []byte{1})
	}
	_, err = b.CensusConsensusRepairsV2(bctx)
	require.ErrorContains(t, err, "q5_calibration census resource ceiling")
}
