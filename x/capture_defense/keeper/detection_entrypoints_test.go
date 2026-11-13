package keeper_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	corestore "cosmossdk.io/core/store"
	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/zerone-chain/zerone/internal/activation"
	"github.com/zerone-chain/zerone/x/capture_defense/keeper"
	"github.com/zerone-chain/zerone/x/capture_defense/types"
)

type timingMarkerCountingService struct {
	corestore.KVStoreService
	reads *int
}

type timingMarkerCountingStore struct {
	corestore.KVStore
	reads *int
}

func (s timingMarkerCountingService) OpenKVStore(ctx context.Context) corestore.KVStore {
	return timingMarkerCountingStore{s.KVStoreService.OpenKVStore(ctx), s.reads}
}

func (s timingMarkerCountingStore) Get(key []byte) ([]byte, error) {
	if bytes.Equal(key, types.IntegerVarianceEnabledKey) {
		*s.reads++
	}
	return s.KVStore.Get(key)
}

func countTimingMarkerReads(f *integerVarianceFixture) *int {
	reads := new(int)
	service := timingMarkerCountingService{runtime.NewKVStoreService(f.key.(*storetypes.KVStoreKey)), reads}
	f.k = keeper.NewKeeper(service, codec.NewProtoCodec(codectypes.NewInterfaceRegistry()), "authority")
	return reads
}

func seedTimingDomains(f *integerVarianceFixture) {
	for _, domain := range []string{"alpha", "beta", "gamma"} {
		f.k.SetVerificationHistory(f.ctx, &types.VerificationHistoryEntry{
			Domain: domain, RoundId: "overflow", Validators: []string{testAddr(1), testAddr(1)},
			Verdicts: []bool{true, true}, SubmitBlocks: []uint64{^uint64(0), ^uint64(0)},
		})
	}
}

func TestCaptureTimingEntryPointsReadMarkerOnce(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		for _, path := range []string{"message", "missing domain", "invalid message", "scheduled block", "idle block", "disabled analysis", "no history"} {
			t.Run(fmt.Sprintf("enabled=%t/%s", enabled, path), func(t *testing.T) {
				f := newIntegerVarianceFixture(t)
				gs := types.DefaultGenesis()
				gs.IntegerVarianceEnabled = false
				if path == "disabled analysis" {
					gs.Params.RiskAnalysisInterval = 0
				}
				f.k.InitGenesis(f.ctx, gs)
				if enabled {
					require.NoError(t, f.k.EnableIntegerVariance(f.ctx))
				}
				if path != "no history" {
					seedTimingDomains(f)
				}
				f.ctx = f.ctx.WithBlockHeight(1000)
				reads := countTimingMarkerReads(f)
				server := keeper.NewMsgServerImpl(f.k)
				switch path {
				case "message":
					_, err := server.AnalyzeDomain(f.ctx, &types.MsgAnalyzeDomain{Domain: "alpha"})
					require.NoError(t, err)
				case "missing domain":
					response, err := server.AnalyzeDomain(f.ctx, &types.MsgAnalyzeDomain{Domain: "missing"})
					require.NoError(t, err)
					require.Equal(t, &types.MsgAnalyzeDomainResponse{}, response)
				case "invalid message":
					response, err := server.AnalyzeDomain(f.ctx, &types.MsgAnalyzeDomain{})
					require.ErrorIs(t, err, types.ErrInvalidDomain)
					require.Nil(t, response)
				case "idle block":
					require.NoError(t, f.k.BeginBlocker(f.ctx.WithBlockHeight(1)))
				default:
					require.NoError(t, f.k.BeginBlocker(f.ctx))
				}
				require.Equal(t, 1, *reads, "one selection even for empty/invalid requests or multiple domains")
				if path == "scheduled block" || path == "message" {
					domains := []string{"alpha"}
					if path == "scheduled block" {
						domains = []string{"alpha", "beta", "gamma"}
					}
					var want uint64
					if enabled {
						want = types.BPSScale
					}
					for _, domain := range domains {
						metrics, found := f.k.GetCaptureMetrics(f.ctx, domain)
						require.True(t, found)
						require.Equal(t, want, metrics.TimingCorrelation)
					}
				}
			})
		}
	}
}

func TestCaptureTimingHelpersUseSuppliedSelection(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		for _, path := range []string{"analysis", "wrapper", "automatic"} {
			t.Run(fmt.Sprintf("enabled=%t/%s", enabled, path), func(t *testing.T) {
				f := newIntegerVarianceFixture(t)
				seedTimingDomains(f)
				// A helper must not read this marker: the caller already selected
				// its path. Corruption here detects accidental nested reads.
				f.ctx.KVStore(f.key).Set(types.IntegerVarianceEnabledKey, []byte{2})
				reads := countTimingMarkerReads(f)
				params := types.DefaultParams()
				require.NotPanics(t, func() {
					switch path {
					case "analysis":
						require.NotNil(t, f.k.AnalyzeCaptureRisk(f.ctx, "alpha", params, enabled))
					case "wrapper":
						require.NotNil(t, f.k.RunCaptureDetection(f.ctx, "alpha", params, enabled))
					case "automatic":
						require.NoError(t, f.k.RunAutoAnalysis(f.ctx, params, enabled))
					}
				})
				require.Zero(t, *reads)
				metrics, found := f.k.GetCaptureMetrics(f.ctx, "alpha")
				require.True(t, found)
				var want uint64
				if enabled {
					want = types.BPSScale
				}
				require.Equal(t, want, metrics.TimingCorrelation)
			})
		}
	}
}

type timingChallengeErrorKeeper struct {
	err   error
	calls []string
}

func (k *timingChallengeErrorKeeper) AutoSubmitChallenge(_ context.Context, domain string, _, _ uint64, _ string) error {
	k.calls = append(k.calls, domain)
	return k.err
}

func TestCaptureTimingChallengeErrorHandling(t *testing.T) {
	// Obtain a real reader fault, then wrap it as a downstream keeper would.
	faultFixture := newIntegerVarianceFixture(t)
	faultFixture.ctx.KVStore(faultFixture.key).Set(types.IntegerVarianceEnabledKey, []byte{2})
	_, markerFault := faultFixture.k.IntegerVarianceEnabled(faultFixture.ctx)
	require.ErrorIs(t, markerFault, activation.ErrMarkerFault)
	ordinary := errors.New("challenge unavailable")

	for _, enabled := range []bool{false, true} {
		var baseline []byte
		for _, tc := range []struct {
			name string
			err  error
			halt bool
		}{
			{"success", nil, false},
			{"ordinary", ordinary, false},
			{"wrapped ordinary", fmt.Errorf("submit: %w", ordinary), false},
			{"sentinel text is not a fault", errors.New(activation.ErrMarkerFault.Error()), false},
			{"wrapped marker fault", fmt.Errorf("challenge marker: %w", markerFault), true},
			{"joined marker fault", errors.Join(ordinary, markerFault), true},
		} {
			t.Run(fmt.Sprintf("enabled=%t/%s", enabled, tc.name), func(t *testing.T) {
				f := newIntegerVarianceFixture(t)
				gs := types.DefaultGenesis()
				gs.IntegerVarianceEnabled = false
				f.k.InitGenesis(f.ctx, gs)
				if enabled {
					require.NoError(t, f.k.EnableIntegerVariance(f.ctx))
				}
				seedTimingDomains(f)
				var writes []timingWrite
				service := timingTraceService{runtime.NewKVStoreService(f.key.(*storetypes.KVStoreKey)), &writes}
				f.k = keeper.NewKeeper(service, codec.NewProtoCodec(codectypes.NewInterfaceRegistry()), "authority")
				challenge := &timingChallengeErrorKeeper{err: tc.err}
				f.k.SetChallengeKeeper(challenge)
				f.k.SetPartnershipsKeeper(newMockPartnershipsKeeper())
				var logs bytes.Buffer
				f.ctx = f.ctx.WithBlockHeight(1000).WithEventManager(sdk.NewEventManager()).
					WithGasMeter(storetypes.NewGasMeter(10_000_000)).
					WithLogger(log.NewLogger(&logs, log.OutputJSONOption(), log.ColorOption(false)))
				var err error
				require.NotPanics(t, func() { err = f.k.BeginBlocker(f.ctx) })
				if tc.halt {
					require.ErrorIs(t, err, activation.ErrMarkerFault)
					require.ErrorIs(t, err, tc.err)
					require.Equal(t, []string{"alpha"}, challenge.calls, "stop the loop on the first marker fault")
					_, found := f.k.GetCaptureMetrics(f.ctx, "beta")
					require.False(t, found)
					require.Empty(t, logs.String(), "a consensus fault must not be logged and skipped")
					return
				}
				require.NoError(t, err)
				require.Equal(t, []string{"alpha", "beta", "gamma"}, challenge.calls)
				trace, marshalErr := json.Marshal(timingTraceStep{
					Writes: writes, Events: f.ctx.EventManager().Events(), Gas: f.ctx.GasMeter().GasConsumed(),
				})
				require.NoError(t, marshalErr)
				if tc.err == nil {
					baseline = trace
					require.Empty(t, logs.String())
				} else {
					require.Equal(t, string(baseline), string(trace), "ordinary failures must preserve all writes, events and gas")
					lines := strings.Split(strings.TrimSpace(logs.String()), "\n")
					require.Len(t, lines, 3)
					for i, line := range lines {
						var entry map[string]any
						require.NoError(t, json.Unmarshal([]byte(line), &entry))
						require.Equal(t, "auto-challenge submission failed", entry["message"])
						require.Equal(t, "x/capture_defense", entry["module"])
						require.Equal(t, challenge.calls[i], entry["domain"])
						require.Equal(t, tc.err.Error(), entry["err"])
					}
				}
			})
		}
	}
}
