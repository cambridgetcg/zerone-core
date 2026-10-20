package keeper_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"testing"

	corestore "cosmossdk.io/core/store"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/zerone-chain/zerone/x/capture_defense/keeper"
	"github.com/zerone-chain/zerone/x/capture_defense/types"
)

func TestCaptureTimingLegacySourceFrozen(t *testing.T) {
	// This is the sole float-guard exception. Pin the entire small file so the
	// exception cannot grow to cover unrelated arithmetic or a repaired legacy
	// algorithm. Its function body is verbatim from detection.go at 2e37c4c.
	data, err := os.ReadFile("detection_legacy.go")
	require.NoError(t, err)
	digest := sha256.Sum256(data)
	require.Equal(t, "ccfbd7e2dda092529d02736b1314e78167a7146021fafeacb09e44e978c61eac", hex.EncodeToString(digest[:]))
}

func TestCaptureTimingMarkerSelectsArithmetic(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		for _, automatic := range []bool{false, true} {
			t.Run(fmt.Sprintf("enabled=%t/automatic=%t", enabled, automatic), func(t *testing.T) {
				k, ctx := setupTimingKeeper(t)
				gs := types.DefaultGenesis()
				gs.IntegerVarianceEnabled = enabled
				k.InitGenesis(ctx, gs)
				server := keeper.NewMsgServerImpl(k)
				_, err := server.RecordVerification(ctx, &types.MsgRecordVerification{
					Authority: k.GetAuthority(), Domain: "timing", RoundId: "overflow",
					Validators: []string{testAddr(1), testAddr(2)}, Verdicts: []bool{true, true},
					SubmitBlocks: []uint64{^uint64(0), ^uint64(0)},
				})
				require.NoError(t, err)
				ctx = ctx.WithEventManager(sdk.NewEventManager())
				if automatic {
					require.NoError(t, k.BeginBlocker(ctx))
				} else {
					_, err = server.AnalyzeDomain(ctx, &types.MsgAnalyzeDomain{Sender: testAddr(1), Domain: "timing"})
					require.NoError(t, err)
				}
				metrics, found := k.GetCaptureMetrics(ctx, "timing")
				require.True(t, found)
				var wantTiming uint64
				if enabled {
					wantTiming = types.BPSScale
				}
				require.Equal(t, wantTiming, metrics.TimingCorrelation)
				wantRisk := (metrics.HerfindahlIndex*40 + wantTiming*20 + metrics.VerdictCorrelation*20 + metrics.Top3Share*20) / 100
				require.Equal(t, wantRisk, metrics.RiskScore)
				if !automatic {
					require.Equal(t, sdk.Events{sdk.NewEvent("zerone.capture_defense.domain_analyzed",
						sdk.NewAttribute("domain", "timing"),
						sdk.NewAttribute("risk_score", fmt.Sprint(wantRisk)),
						sdk.NewAttribute("hhi", fmt.Sprint(metrics.HerfindahlIndex)),
						sdk.NewAttribute("flagged", fmt.Sprint(metrics.Flagged)),
						sdk.NewAttribute("creed_commitment", "9"),
					)}, ctx.EventManager().Events())
				}
			})
		}
	}
}

func TestCaptureTimingLegacyHistoryRecomputedAtNextInterval(t *testing.T) {
	f := newIntegerVarianceFixture(t)
	gs := types.DefaultGenesis()
	gs.IntegerVarianceEnabled = false
	f.k.InitGenesis(f.ctx, gs)
	f.k.SetVerificationHistory(f.ctx, &types.VerificationHistoryEntry{
		Domain: "retained", RoundId: "before-carrier",
		Validators: []string{testAddr(1), testAddr(2)}, Verdicts: []bool{true, true},
		SubmitBlocks: []uint64{^uint64(0), ^uint64(0)},
	})
	f.ctx = f.ctx.WithBlockHeight(1000)
	require.NoError(t, f.k.BeginBlocker(f.ctx))
	old, found := f.k.GetCaptureMetrics(f.ctx, "retained")
	require.True(t, found)
	require.Zero(t, old.TimingCorrelation, "the old uint64 sum overflows")
	before := f.snapshot(t)

	// Simulated carrier write only, not an application upgrade rehearsal.
	f.ctx = f.ctx.WithBlockHeight(1500)
	require.NoError(t, f.k.EnableIntegerVariance(f.ctx))
	before[string(types.IntegerVarianceEnabledKey)] = "\x01"
	require.Equal(t, before, f.snapshot(t), "activation must not migrate metrics or history")
	require.NoError(t, f.k.BeginBlocker(f.ctx.WithBlockHeight(1501)))
	require.Equal(t, before, f.snapshot(t), "do not eagerly recompute outside the interval")

	require.NoError(t, f.k.BeginBlocker(f.ctx.WithBlockHeight(2000)))
	current, found := f.k.GetCaptureMetrics(f.ctx, "retained")
	require.True(t, found)
	require.Equal(t, uint64(types.BPSScale), current.TimingCorrelation)
	require.Equal(t, old.RiskScore+types.BPSScale/5, current.RiskScore)
	require.Equal(t, old.Flagged, current.Flagged)
	require.Equal(t, old.HerfindahlIndex, current.HerfindahlIndex)
	require.Equal(t, uint64(2000), current.AnalyzedAtBlock)
	after := f.snapshot(t)
	for key, value := range before {
		if key[0] != types.CaptureMetricsKeyPrefix[0] {
			require.Equal(t, value, after[key], "history and other records remain immutable")
		}
	}
	require.Len(t, after, len(before))
}

type unreadableTimingMarkerService struct{ corestore.KVStoreService }
type unreadableTimingMarkerStore struct{ corestore.KVStore }

func (s unreadableTimingMarkerService) OpenKVStore(ctx context.Context) corestore.KVStore {
	return unreadableTimingMarkerStore{s.KVStoreService.OpenKVStore(ctx)}
}

func (s unreadableTimingMarkerStore) Get(key []byte) ([]byte, error) {
	if bytes.Equal(key, types.IntegerVarianceEnabledKey) {
		return nil, fmt.Errorf("injected marker read failure")
	}
	return s.KVStore.Get(key)
}

func TestCaptureTimingBadMarkerFailsClosed(t *testing.T) {
	for _, unreadable := range []bool{false, true} {
		for _, path := range []string{"analysis", "wrapper", "message", "automatic"} {
			t.Run(fmt.Sprintf("unreadable=%t/%s", unreadable, path), func(t *testing.T) {
				f := newIntegerVarianceFixture(t)
				f.ctx = f.ctx.WithBlockHeight(1000)
				f.k.SetVerificationHistory(f.ctx, &types.VerificationHistoryEntry{
					Domain: "timing", RoundId: "existing", Validators: []string{testAddr(1), testAddr(2)},
					SubmitBlocks: []uint64{1, 1},
				})
				f.ctx.KVStore(f.key).Set(types.IntegerVarianceEnabledKey, []byte{2})
				wantError := "malformed integer variance marker"
				if unreadable {
					service := unreadableTimingMarkerService{runtime.NewKVStoreService(f.key.(*storetypes.KVStoreKey))}
					f.k = keeper.NewKeeper(service, codec.NewProtoCodec(codectypes.NewInterfaceRegistry()), "authority")
					wantError = "read integer variance marker: injected marker read failure"
				}
				before := f.snapshot(t)
				require.PanicsWithError(t, wantError, func() {
					switch path {
					case "analysis":
						f.k.AnalyzeCaptureRisk(f.ctx, "timing", types.DefaultParams())
					case "wrapper":
						f.k.RunCaptureDetection(f.ctx, "timing", types.DefaultParams())
					case "message":
						_, _ = keeper.NewMsgServerImpl(f.k).AnalyzeDomain(f.ctx, &types.MsgAnalyzeDomain{Domain: "timing"})
					case "automatic":
						_ = f.k.BeginBlocker(f.ctx)
					}
				})
				require.Equal(t, before, f.snapshot(t), "never overwrite state after an unreadable selector")
				require.Empty(t, f.ctx.EventManager().Events())
			})
		}
	}
}
