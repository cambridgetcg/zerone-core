package keeper_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
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

type timingWrite struct {
	Operation string `json:"operation"`
	Key       string `json:"key"`
	Value     string `json:"value,omitempty"`
}

type timingTraceService struct {
	corestore.KVStoreService
	writes *[]timingWrite
}

type timingTraceStore struct {
	corestore.KVStore
	writes *[]timingWrite
}

func (s timingTraceService) OpenKVStore(ctx context.Context) corestore.KVStore {
	return timingTraceStore{s.KVStoreService.OpenKVStore(ctx), s.writes}
}

func (s timingTraceStore) Set(key, value []byte) error {
	*s.writes = append(*s.writes, timingWrite{"set", hex.EncodeToString(key), hex.EncodeToString(value)})
	return s.KVStore.Set(key, value)
}

func (s timingTraceStore) Delete(key []byte) error {
	*s.writes = append(*s.writes, timingWrite{Operation: "delete", Key: hex.EncodeToString(key)})
	return s.KVStore.Delete(key)
}

type timingTraceStep struct {
	Name     string          `json:"name"`
	Writes   []timingWrite   `json:"writes"`
	Events   sdk.Events      `json:"events"`
	Gas      uint64          `json:"gas"`
	Response json.RawMessage `json:"response"`
	Error    string          `json:"error"`
}

func captureTimingLegacyTrace(t *testing.T) []timingTraceStep {
	t.Helper()
	f := newIntegerVarianceFixture(t)
	var writes []timingWrite
	service := timingTraceService{runtime.NewKVStoreService(f.key.(*storetypes.KVStoreKey)), &writes}
	k := keeper.NewKeeper(service, codec.NewProtoCodec(codectypes.NewInterfaceRegistry()), testAddr(100))
	server := keeper.NewMsgServerImpl(k)
	var trace []timingTraceStep
	step := func(name string, run func(sdk.Context) (any, error)) {
		t.Helper()
		writes = nil
		ctx := f.ctx.WithBlockHeight(1000).WithGasMeter(storetypes.NewGasMeter(10_000_000)).WithEventManager(sdk.NewEventManager())
		response, err := run(ctx)
		out := timingTraceStep{Name: name, Writes: writes, Events: ctx.EventManager().Events(), Gas: ctx.GasMeter().GasConsumed()}
		var marshalErr error
		out.Response, marshalErr = json.Marshal(response)
		require.NoError(t, marshalErr)
		if err != nil {
			out.Error = err.Error()
		}
		trace = append(trace, out)
	}
	step("legacy genesis", func(ctx sdk.Context) (any, error) {
		gs := types.DefaultGenesis()
		gs.IntegerVarianceEnabled = false // a predecessor chain has no marker
		k.InitGenesis(ctx, gs)
		return nil, nil
	})
	step("empty domain error", func(ctx sdk.Context) (any, error) {
		return server.AnalyzeDomain(ctx, &types.MsgAnalyzeDomain{Sender: testAddr(1)})
	})
	step("no history", func(ctx sdk.Context) (any, error) {
		return server.AnalyzeDomain(ctx, &types.MsgAnalyzeDomain{Sender: testAddr(1), Domain: "missing"})
	})
	step("no participants", func(ctx sdk.Context) (any, error) {
		k.SetVerificationHistory(ctx, &types.VerificationHistoryEntry{Domain: "empty", RoundId: "empty"})
		return server.AnalyzeDomain(ctx, &types.MsgAnalyzeDomain{Sender: testAddr(1), Domain: "empty"})
	})
	msg := &types.MsgRecordVerification{
		Authority: k.GetAuthority(), Domain: "timing", RoundId: "overflow",
		Validators: []string{testAddr(1), testAddr(2)}, Verdicts: []bool{true, true},
		SubmitBlocks: []uint64{^uint64(0), ^uint64(0)},
	}
	step("wrong authority error", func(ctx sdk.Context) (any, error) {
		return server.RecordVerification(ctx, &types.MsgRecordVerification{Authority: testAddr(99)})
	})
	step("mismatched timing error", func(ctx sdk.Context) (any, error) {
		return server.RecordVerification(ctx, &types.MsgRecordVerification{
			Authority: k.GetAuthority(), Validators: msg.Validators, Verdicts: msg.Verdicts, SubmitBlocks: []uint64{1},
		})
	})
	step("record overflowing round", func(ctx sdk.Context) (any, error) { return server.RecordVerification(ctx, msg) })
	step("analyze overflowing round", func(ctx sdk.Context) (any, error) {
		return server.AnalyzeDomain(ctx, &types.MsgAnalyzeDomain{Sender: testAddr(1), Domain: msg.Domain})
	})
	step("record nil timing", func(ctx sdk.Context) (any, error) {
		return server.RecordVerification(ctx, &types.MsgRecordVerification{
			Authority: k.GetAuthority(), Domain: "timing", RoundId: "nil",
			Validators: msg.Validators, Verdicts: msg.Verdicts,
		})
	})
	step("record correlated round", func(ctx sdk.Context) (any, error) {
		return server.RecordVerification(ctx, &types.MsgRecordVerification{
			Authority: k.GetAuthority(), Domain: "timing", RoundId: "tight",
			Validators: msg.Validators, Verdicts: msg.Verdicts, SubmitBlocks: []uint64{100, 103},
		})
	})
	step("analyze mixed history", func(ctx sdk.Context) (any, error) {
		return server.AnalyzeDomain(ctx, &types.MsgAnalyzeDomain{Sender: testAddr(1), Domain: msg.Domain})
	})
	step("begin block mixed history", func(ctx sdk.Context) (any, error) { return nil, k.BeginBlocker(ctx) })
	return trace
}

func TestCaptureTimingMarkerOffMatchesPredecessor(t *testing.T) {
	// Captured by running this trace against the unmodified detection.go blob
	// 3085b4f83fd55ab0611f11d5e583b687791d7bac (2e37c4c and 77521a0d)
	// via a compiler overlay. The real IAVL-backed SDK store records every
	// Set/Delete, exact event order/attributes, response, error and charged gas.
	// Stable vectors avoid the architecture-sensitive variance==4 boundary.
	got, err := json.MarshalIndent(captureTimingLegacyTrace(t), "", "  ")
	require.NoError(t, err)
	// Pin all 12 steps (22 exact writes and three errors), not just the final
	// metrics. Print the complete trace if any byte, event or gas value drifts.
	digest := sha256.Sum256(append(got, '\n'))
	require.Equal(t, "34dc1089412670523d9a47b408dddc7dca96e83804f13832181ae2097e7614c3", hex.EncodeToString(digest[:]), "trace:\n%s", got)
}
