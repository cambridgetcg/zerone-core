package keeper_test

import (
	"testing"

	"cosmossdk.io/log"
	"cosmossdk.io/store"
	storemetrics "cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/zerone-chain/zerone/x/capture_defense/keeper"
	"github.com/zerone-chain/zerone/x/capture_defense/types"
)

func setupTimingKeeper(t *testing.T) (keeper.Keeper, sdk.Context) {
	t.Helper()
	key := storetypes.NewKVStoreKey(types.StoreKey)
	db := dbm.NewMemDB()
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	state := store.NewCommitMultiStore(db, log.NewNopLogger(), storemetrics.NewNoOpMetrics())
	// v1.1.1 (the legacy SDK lineage) does not expose sync pruning. Use it
	// where supported without making this fixture require the newer API.
	if syncPruning, ok := any(state).(interface{ SetIAVLSyncPruning(bool) }); ok {
		syncPruning.SetIAVLSyncPruning(true)
	}
	state.MountStoreWithDB(key, storetypes.StoreTypeIAVL, db)
	require.NoError(t, state.LoadLatestVersion())
	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	k := keeper.NewKeeper(runtime.NewKVStoreService(key), cdc, testAddr(100))
	ctx := sdk.NewContext(state, cmtproto.Header{Height: 1000}, false, log.NewNopLogger())
	return k, ctx
}

func TestCaptureTimingStatePaths(t *testing.T) {
	for _, tc := range []struct {
		name   string
		blocks []uint64
		timing uint64
	}{
		{"reported flip", []uint64{2718285, 2718286, 2718292, 2718287, 2718288, 2718288, 2718287, 2718285, 2718288}, 0},
		{"below", []uint64{100, 103}, types.BPSScale},
		{"equal", []uint64{100, 104}, 0},
		{"above", []uint64{100, 105}, 0},
		{"uint64 limit", []uint64{^uint64(0) - 1, ^uint64(0)}, types.BPSScale},
	} {
		for _, automatic := range []bool{false, true} {
			path := "AnalyzeDomain"
			if automatic {
				path = "BeginBlocker"
			}
			t.Run(tc.name+"/"+path, func(t *testing.T) {
				k, ctx := setupTimingKeeper(t)
				require.NoError(t, k.EnableIntegerVariance(ctx))
				server := keeper.NewMsgServerImpl(k)
				msg := &types.MsgRecordVerification{
					Authority: k.GetAuthority(), Domain: "timing", RoundId: "round-1",
					SubmitBlocks: append([]uint64(nil), tc.blocks...),
				}
				for i := range tc.blocks {
					msg.Validators = append(msg.Validators, testAddr(i+1))
					msg.Verdicts = append(msg.Verdicts, true)
				}
				require.NoError(t, msg.ValidateBasic())
				_, err := server.RecordVerification(ctx, msg)
				require.NoError(t, err)
				before, found := k.GetVerificationHistory(ctx, msg.Domain, msg.RoundId)
				require.True(t, found)
				beforeBytes, err := proto.Marshal(before)
				require.NoError(t, err)

				n := uint64(len(tc.blocks))
				share := uint64(types.BPSScale) / n
				hhi := n * (share * share / types.BPSScale)
				top3 := min(n, 3) * types.BPSScale / n
				want := &types.CaptureMetrics{
					Domain: msg.Domain, HerfindahlIndex: hhi,
					TimingCorrelation: tc.timing, VerdictCorrelation: types.BPSScale,
					Top3Share:           top3,
					RiskScore:           (hhi*40 + tc.timing*20 + types.BPSScale*20 + top3*20) / 100,
					TotalParticipations: n, AnalyzedAtBlock: uint64(ctx.BlockHeight()),
					Flagged: hhi > types.DefaultParams().HhiThreshold,
				}
				if automatic {
					require.NoError(t, k.BeginBlocker(ctx))
				} else {
					response, err := server.AnalyzeDomain(ctx, &types.MsgAnalyzeDomain{Sender: testAddr(1), Domain: msg.Domain})
					require.NoError(t, err)
					require.Equal(t, want.RiskScore, response.RiskScore)
					require.Equal(t, want.Flagged, response.Flagged)
				}
				stored, found := k.GetCaptureMetrics(ctx, msg.Domain)
				require.True(t, found)
				wantBytes, err := proto.Marshal(want)
				require.NoError(t, err)
				gotBytes, err := proto.Marshal(stored)
				require.NoError(t, err)
				require.Equal(t, wantBytes, gotBytes, "exact stored metrics, including timing and risk")
				after, found := k.GetVerificationHistory(ctx, msg.Domain, msg.RoundId)
				require.True(t, found)
				afterBytes, err := proto.Marshal(after)
				require.NoError(t, err)
				require.Equal(t, beforeBytes, afterBytes, "analysis must not rewrite its history")
			})
		}
	}
}

func TestCaptureTimingRetainsLegacyMetricsUntilAnalysis(t *testing.T) {
	k, ctx := setupTimingKeeper(t)
	legacy := &types.CaptureMetrics{
		Domain: "legacy", TimingCorrelation: types.BPSScale,
		RiskScore: 123456, AnalyzedAtBlock: 50,
	}
	genesis := types.DefaultGenesis()
	genesis.IntegerVarianceEnabled = false // rehearse a pre-carrier import
	genesis.CaptureMetrics = []*types.CaptureMetrics{legacy}
	k.InitGenesis(ctx, genesis)
	wantBytes, err := proto.Marshal(legacy)
	require.NoError(t, err)
	assertRetained := func() {
		t.Helper()
		stored, found := k.GetCaptureMetrics(ctx, legacy.Domain)
		require.True(t, found)
		gotBytes, err := proto.Marshal(stored)
		require.NoError(t, err)
		require.Equal(t, wantBytes, gotBytes)
	}
	assertRetained()
	require.NoError(t, k.EnableIntegerVariance(ctx))
	assertRetained() // activation writes only the marker, not historical metrics
	// There is no verification-history field in this module's genesis. Analysis
	// without retained observations must not invent or erase historical metrics.
	integerVariance, err := k.IntegerVarianceEnabled(ctx)
	require.NoError(t, err)
	require.Nil(t, k.AnalyzeCaptureRisk(ctx, legacy.Domain, k.GetParams(ctx), integerVariance))
	require.NoError(t, k.BeginBlocker(ctx))
	assertRetained()
	exported := k.ExportGenesis(ctx)
	require.Len(t, exported.CaptureMetrics, 1)
	exportedBytes, err := proto.Marshal(exported.CaptureMetrics[0])
	require.NoError(t, err)
	require.Equal(t, wantBytes, exportedBytes)

	k.SetVerificationHistory(ctx, &types.VerificationHistoryEntry{
		Domain: legacy.Domain, RoundId: "retained",
		Validators: []string{testAddr(1), testAddr(2)}, Verdicts: []bool{true, true},
		SubmitBlocks: []uint64{100, 104},
	})
	assertRetained()
	require.NoError(t, k.BeginBlocker(ctx))
	stored, found := k.GetCaptureMetrics(ctx, legacy.Domain)
	require.True(t, found)
	require.Zero(t, stored.TimingCorrelation)
	require.Equal(t, uint64(ctx.BlockHeight()), stored.AnalyzedAtBlock)
	require.NotEqual(t, legacy.RiskScore, stored.RiskScore)
}
