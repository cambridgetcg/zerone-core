package ibcratelimit

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	corestore "cosmossdk.io/core/store"
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
	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	porttypes "github.com/cosmos/ibc-go/v10/modules/core/05-port/types"
	ibcexported "github.com/cosmos/ibc-go/v10/modules/core/exported"
	"github.com/stretchr/testify/require"

	"github.com/zerone-chain/zerone/internal/activation"
	"github.com/zerone-chain/zerone/x/ibcratelimit/keeper"
	"github.com/zerone-chain/zerone/x/ibcratelimit/types"
)

// Only transport is synthetic. Reads/writes and gas use the runtime adapter
// against synchronous IAVL; the trace also pins ignored legacy write errors.
type dispatchStore struct {
	corestore.KVStoreService
	markerReads int
	writes      []string
	fault       string
}

type dispatchKV struct {
	corestore.KVStore
	trace *dispatchStore
}

func (s *dispatchStore) OpenKVStore(ctx context.Context) corestore.KVStore {
	return dispatchKV{s.KVStoreService.OpenKVStore(ctx), s}
}

func (s dispatchKV) Get(key []byte) ([]byte, error) {
	if bytes.Equal(key, types.RefundIdentityEnabledKey) {
		s.trace.markerReads++
		if s.trace.fault == "marker" {
			return nil, errors.New("injected marker read")
		}
	}
	if key[0] == types.RuleGenerationPrefix[0] && s.trace.fault == "generation" {
		return nil, errors.New("injected generation read")
	}
	return s.KVStore.Get(key)
}

func (s dispatchKV) Set(key, value []byte) error {
	s.trace.writes = append(s.trace.writes, fmt.Sprintf("set:%x:%x", key, value))
	if s.trace.fault == "set" {
		return errors.New("injected write")
	}
	return s.KVStore.Set(key, value)
}

func (s dispatchKV) Delete(key []byte) error {
	s.trace.writes = append(s.trace.writes, fmt.Sprintf("delete:%x", key))
	if s.trace.fault == "delete" {
		return errors.New("injected delete")
	}
	return s.KVStore.Delete(key)
}

type dispatchTransport struct {
	porttypes.IBCModule
	porttypes.ICS4Wrapper
	calls int
	err   error
}

func (a *dispatchTransport) called(ctx sdk.Context) {
	a.calls++
	ctx.EventManager().EmitEvent(sdk.NewEvent("test.transport"))
}

func (a *dispatchTransport) SendPacket(ctx sdk.Context, _, _ string, _ clienttypes.Height, _ uint64, _ []byte) (uint64, error) {
	a.called(ctx)
	return 7, a.err
}

func (a *dispatchTransport) OnRecvPacket(ctx sdk.Context, _ string, _ channeltypes.Packet, _ sdk.AccAddress) ibcexported.Acknowledgement {
	a.called(ctx)
	return channeltypes.NewResultAcknowledgement([]byte{1})
}

func (a *dispatchTransport) OnAcknowledgementPacket(ctx sdk.Context, _ string, _ channeltypes.Packet, _ []byte, _ sdk.AccAddress) error {
	a.called(ctx)
	return a.err
}

func (a *dispatchTransport) OnTimeoutPacket(ctx sdk.Context, _ string, _ channeltypes.Packet, _ sdk.AccAddress) error {
	a.called(ctx)
	return a.err
}

type dispatchFixture struct {
	k     keeper.Keeper
	ctx   sdk.Context
	key   *storetypes.KVStoreKey
	im    IBCMiddleware
	trace *dispatchStore
	app   *dispatchTransport
}

func newDispatchFixture(t *testing.T, enabled bool) *dispatchFixture {
	t.Helper()
	key := storetypes.NewKVStoreKey(types.StoreKey)
	db := dbm.NewMemDB()
	ms := store.NewCommitMultiStore(db, log.NewNopLogger(), storemetrics.NewNoOpMetrics())
	ms.SetIAVLSyncPruning(true)
	ms.MountStoreWithDB(key, storetypes.StoreTypeIAVL, db)
	require.NoError(t, ms.LoadLatestVersion())
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	trace := &dispatchStore{KVStoreService: runtime.NewKVStoreService(key)}
	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	k := keeper.NewKeeper(trace, cdc, "authority")
	ctx := sdk.NewContext(ms, cmtproto.Header{Height: 100}, false, log.NewNopLogger())
	gs := types.DefaultGenesis()
	gs.RefundIdentityEnabled = false // rehearse pre-carrier state explicitly
	k.InitGenesis(ctx, gs)
	if enabled {
		require.NoError(t, k.EnableRefundIdentity(ctx))
	}
	a := &dispatchTransport{}
	return &dispatchFixture{k, ctx, key, NewIBCMiddleware(a, a, k), trace, a}
}

func (f *dispatchFixture) seed() {
	f.k.SetRateLimit(f.ctx, &types.RateLimit{ChannelId: "channel-0", Denom: "uzrn", MaxSend: "100", MaxRecv: "100", WindowBlocks: 10, WindowStart: 100, CurrentSend: "40", CurrentRecv: "0"})
	f.k.SetPacketFlow(f.ctx, &types.PacketFlow{ChannelId: "channel-0", Sequence: 7, Denom: "uzrn", Amount: "30"})
}

func (f *dispatchFixture) resetMeter() {
	f.ctx = f.ctx.WithGasMeter(storetypes.NewInfiniteGasMeter()).WithEventManager(sdk.NewEventManager())
	f.trace.markerReads, f.trace.writes = 0, nil
}

func (f *dispatchFixture) snapshot() map[string]string {
	kv := f.ctx.KVStore(f.key)
	it := kv.Iterator(nil, nil)
	defer it.Close()
	result := map[string]string{}
	for ; it.Valid(); it.Next() {
		result[string(it.Key())] = string(it.Value())
	}
	return result
}

func TestRefundIdentityMarkerOffMatchesFrozenMiddleware(t *testing.T) {
	for _, scenario := range []string{"send", "unconfigured", "disabled", "limit", "invalid_json", "invalid_amount", "send_failure", "disabled_failure", "set_error", "timeout", "delete_error", "callback_error", "ack_error", "ack_success", "ack_malformed", "recv", "recv_limit", "recv_invalid"} {
		t.Run(scenario, func(t *testing.T) {
			var states [2]map[string]string
			var writes [2][]string
			var events [2]sdk.Events
			var gas [2]uint64
			var results [2]string
			for i, legacy := range []bool{false, true} {
				f := newDispatchFixture(t, false)
				f.seed()
				data := []byte(`{"denom":"uzrn","amount":"30"}`)
				packet := channeltypes.Packet{SourceChannel: "channel-0", DestinationChannel: "channel-0", Sequence: 7, Data: data}
				switch scenario {
				case "unconfigured":
					f.k.DeleteRateLimit(f.ctx, "channel-0", "uzrn")
				case "disabled", "disabled_failure":
					f.k.SetParams(f.ctx, &types.Params{Enabled: false})
				case "limit", "recv_limit":
					data = []byte(`{"denom":"uzrn","amount":"101"}`)
				case "invalid_json", "recv_invalid":
					data = []byte(`{`)
				case "invalid_amount":
					data = []byte(`{"amount":"bad"}`)
				case "set_error":
					f.trace.fault = "set"
				case "delete_error":
					f.trace.fault = "delete"
				}
				if scenario == "send_failure" || scenario == "disabled_failure" || scenario == "callback_error" {
					f.app.err = errors.New("transport refusal")
				}
				f.resetMeter()
				var err error
				var seq uint64
				switch scenario {
				case "timeout", "delete_error", "callback_error":
					if legacy {
						err = f.im.onTimeoutPacketLegacy(f.ctx, "ics20-1", packet, nil)
					} else {
						err = f.im.OnTimeoutPacket(f.ctx, "ics20-1", packet, nil)
					}
				case "ack_error", "ack_success", "ack_malformed":
					ack := channeltypes.NewErrorAcknowledgement(errors.New("remote refusal")).Acknowledgement()
					if scenario == "ack_success" {
						ack = channeltypes.NewResultAcknowledgement([]byte{1}).Acknowledgement()
					} else if scenario == "ack_malformed" {
						ack = []byte("{")
					}
					if legacy {
						err = f.im.onAcknowledgementPacketLegacy(f.ctx, "ics20-1", packet, ack, nil)
					} else {
						err = f.im.OnAcknowledgementPacket(f.ctx, "ics20-1", packet, ack, nil)
					}
				case "recv", "recv_limit", "recv_invalid":
					packet.Data = data
					var ack ibcexported.Acknowledgement
					if legacy {
						ack = f.im.onRecvPacketLegacy(f.ctx, "ics20-1", packet, nil)
					} else {
						ack = f.im.OnRecvPacket(f.ctx, "ics20-1", packet, nil)
					}
					results[i] = string(ack.Acknowledgement())
				default:
					if legacy {
						seq, err = f.im.sendPacketLegacy(f.ctx, "transfer", "channel-0", clienttypes.ZeroHeight(), 0, data)
					} else {
						seq, err = f.im.SendPacket(f.ctx, "transfer", "channel-0", clienttypes.ZeroHeight(), 0, data)
					}
				}
				results[i] += fmt.Sprintf("/%d/%v/%d", seq, err, f.app.calls)
				gas[i], events[i], writes[i] = f.ctx.GasMeter().GasConsumed(), f.ctx.EventManager().Events(), f.trace.writes
				require.Equal(t, 1-i, f.trace.markerReads, "exactly one ungassed selector read per entry")
				states[i] = f.snapshot()
				for key := range states[i] {
					require.NotEqual(t, types.RuleGenerationPrefix[0], key[0], "off must never install a generation")
				}
			}
			require.Equal(t, states[1], states[0])
			require.Equal(t, writes[1], writes[0])
			require.Equal(t, events[1], events[0])
			require.Equal(t, results[1], results[0])
			require.Equal(t, gas[1], gas[0])
		})
	}
}

func TestRefundIdentityFaultAbortsBeforeMiddlewareWrites(t *testing.T) {
	for _, fault := range []string{"malformed", "marker"} {
		for _, entry := range []string{"send", "recv", "ack", "timeout", "begin", "add", "remove", "params"} {
			t.Run(fault+"/"+entry, func(t *testing.T) {
				f := newDispatchFixture(t, false)
				f.seed()
				if fault == "malformed" {
					f.ctx.KVStore(f.key).Set(types.RefundIdentityEnabledKey, []byte{0})
				} else {
					f.trace.fault = fault
				}
				before := f.snapshot()
				f.resetMeter()
				var err error
				packet := channeltypes.Packet{SourceChannel: "channel-0", Sequence: 7}
				switch entry {
				case "send":
					_, err = f.im.SendPacket(f.ctx, "transfer", "channel-0", clienttypes.ZeroHeight(), 0, nil)
				case "recv":
					func() {
						defer func() { err, _ = recover().(error) }()
						f.im.OnRecvPacket(f.ctx, "ics20-1", packet, nil)
					}()
				case "ack":
					err = f.im.OnAcknowledgementPacket(f.ctx, "ics20-1", packet, nil, nil)
				case "timeout":
					err = f.im.OnTimeoutPacket(f.ctx, "ics20-1", packet, nil)
				case "begin":
					err = NewAppModule(nil, f.k).BeginBlock(f.ctx)
				case "add":
					_, err = keeper.NewMsgServerImpl(f.k).AddRateLimit(f.ctx, &types.MsgAddRateLimit{})
				case "remove":
					_, err = keeper.NewMsgServerImpl(f.k).RemoveRateLimit(f.ctx, &types.MsgRemoveRateLimit{})
				case "params":
					_, err = keeper.NewMsgServerImpl(f.k).UpdateParams(f.ctx, &types.MsgUpdateParams{})
				}
				require.ErrorIs(t, err, activation.ErrMarkerFault)
				require.Equal(t, 1, f.trace.markerReads)
				require.Empty(t, f.trace.writes)
				require.Empty(t, f.ctx.EventManager().Events())
				require.Zero(t, f.ctx.GasMeter().GasConsumed())
				require.Zero(t, f.app.calls)
				require.Equal(t, before, f.snapshot())
			})
		}
	}
}

func TestRefundIdentityCorruptGenerationDoesNotBlockCallback(t *testing.T) {
	for _, fault := range []string{"malformed", "generation"} {
		for _, mode := range []string{"timeout", "ack"} {
			t.Run(fault+"/"+mode, func(t *testing.T) {
				f := newDispatchFixture(t, true)
				f.seed() // old rule without a generation key
				_, err := f.im.SendPacket(f.ctx, "transfer", "channel-0", clienttypes.ZeroHeight(), 0, []byte(`{"denom":"uzrn","amount":"30"}`))
				require.NoError(t, err)
				if fault == "malformed" {
					f.ctx.KVStore(f.key).Set(append([]byte{3}, []byte("channel-0\x00uzrn")...), []byte{1})
				} else {
					f.trace.fault = fault
				}
				f.app.calls = 0
				packet := channeltypes.Packet{SourceChannel: "channel-0", Sequence: 7}
				if mode == "timeout" {
					err = f.im.OnTimeoutPacket(f.ctx, "ics20-1", packet, nil)
				} else {
					ack := channeltypes.NewErrorAcknowledgement(errors.New("remote refusal"))
					err = f.im.OnAcknowledgementPacket(f.ctx, "ics20-1", packet, ack.Acknowledgement(), nil)
				}
				require.NoError(t, err)
				require.Equal(t, 1, f.app.calls)
				_, found := f.k.GetPacketFlow(f.ctx, "channel-0", 7)
				require.False(t, found)
				rl, found := f.k.GetRateLimit(f.ctx, "channel-0", "uzrn")
				require.True(t, found)
				require.Equal(t, "70", rl.CurrentSend)
				// Charges still fail closed, unlike metadata-only quota refunds.
				require.Panics(t, func() {
					_, _ = f.im.SendPacket(f.ctx, "transfer", "channel-0", clienttypes.ZeroHeight(), 0, []byte(`{"denom":"uzrn","amount":"1"}`))
				})
			})
		}
	}
}

func TestRefundIdentityGenesisAndBeginBlockReadOnce(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		f := newDispatchFixture(t, false)
		gs := types.DefaultGenesis()
		gs.RefundIdentityEnabled = enabled
		for _, channel := range []string{"channel-0", "channel-1"} {
			gs.RateLimits = append(gs.RateLimits, &types.RateLimit{
				ChannelId: channel, Denom: "uzrn", MaxSend: "100", MaxRecv: "100",
				WindowBlocks: 10, WindowStart: 0, CurrentSend: "30", CurrentRecv: "20",
			})
		}
		f.resetMeter()
		f.k.InitGenesis(f.ctx, gs)
		require.Equal(t, 1, f.trace.markerReads, "genesis selection is outside the rule loop")
		f.resetMeter()
		require.NoError(t, NewAppModule(nil, f.k).BeginBlock(f.ctx))
		require.Equal(t, 1, f.trace.markerReads, "block selection is outside the rule loop")
		for _, rule := range f.k.GetAllRateLimits(f.ctx) {
			require.Equal(t, "0", rule.CurrentSend)
			require.Equal(t, uint64(100), rule.WindowStart)
		}
	}
}

func TestRefundIdentityMiddlewareChargeGas(t *testing.T) {
	var gas [2]uint64
	for i, enabled := range []bool{false, true} {
		f := newDispatchFixture(t, enabled)
		f.seed()
		f.resetMeter()
		_, err := f.im.SendPacket(f.ctx, "transfer", "channel-0", clienttypes.ZeroHeight(), 0, []byte(`{"denom":"uzrn","amount":"30"}`))
		require.NoError(t, err)
		gas[i] = f.ctx.GasMeter().GasConsumed()
		require.Equal(t, 1, f.trace.markerReads)
	}
	t.Logf("middleware charge gas: marker-off=%d marker-on=%d delta=%d (synthetic transport, legacy generation-0 rule)", gas[0], gas[1], gas[1]-gas[0])
	require.Greater(t, gas[1], gas[0])
}

func TestRefundIdentityLegacyFlowTransition(t *testing.T) {
	for _, mode := range []string{"timeout", "ack"} {
		t.Run(mode, func(t *testing.T) {
			f := newDispatchFixture(t, false)
			f.seed()
			// Charge with the actual marker-off middleware, then activate in place.
			_, err := f.im.SendPacket(f.ctx, "transfer", "channel-0", clienttypes.ZeroHeight(), 0, []byte(`{"denom":"uzrn","amount":"30"}`))
			require.NoError(t, err)
			before := f.snapshot()
			require.NoError(t, f.k.EnableRefundIdentity(f.ctx))
			after := f.snapshot()
			delete(after, string(types.RefundIdentityEnabledKey))
			require.Equal(t, before, after, "activation must not backfill receipts/counters")
			packet := channeltypes.Packet{SourceChannel: "channel-0", Sequence: 7}
			if mode == "timeout" {
				err = f.im.OnTimeoutPacket(f.ctx, "ics20-1", packet, nil)
			} else {
				ack := channeltypes.NewErrorAcknowledgement(errors.New("remote refusal"))
				err = f.im.OnAcknowledgementPacket(f.ctx, "ics20-1", packet, ack.Acknowledgement(), nil)
			}
			require.NoError(t, err)
			rl, _ := f.k.GetRateLimit(f.ctx, "channel-0", "uzrn")
			require.Equal(t, "70", rl.CurrentSend, "legacy receipt never reduces current quota")
			_, found := f.k.GetPacketFlow(f.ctx, "channel-0", 7)
			require.False(t, found)
			require.Equal(t, 2, f.app.calls, "send and token-refund callback both forwarded")
			// The first post-activation receipt on the retained rule is explicit 0.
			_, err = f.im.SendPacket(f.ctx, "transfer", "channel-0", clienttypes.ZeroHeight(), 0, []byte(`{"denom":"uzrn","amount":"1"}`))
			require.NoError(t, err)
			for key, value := range f.snapshot() {
				if key[0] == types.PacketFlowPrefix[0] {
					var record map[string]json.RawMessage
					require.NoError(t, json.Unmarshal([]byte(value), &record))
					require.JSONEq(t, "0", string(record["rule_generation"]))
				}
			}
		})
	}
}
