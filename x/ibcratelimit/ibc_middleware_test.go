package ibcratelimit_test

import (
	"encoding/json"
	"errors"
	"math/big"
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
	transfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	porttypes "github.com/cosmos/ibc-go/v10/modules/core/05-port/types"
	"github.com/stretchr/testify/require"

	"github.com/zerone-chain/zerone/x/ibcratelimit"
	"github.com/zerone-chain/zerone/x/ibcratelimit/keeper"
	"github.com/zerone-chain/zerone/x/ibcratelimit/types"
)

// Only the transport seam is synthetic: all quota, rule and flow operations use
// the real keeper and middleware against a fresh synchronous IAVL store.
type packetSender struct {
	porttypes.ICS4Wrapper
	sequence uint64
	err      error
}

func (s *packetSender) SendPacket(_ sdk.Context, _, _ string, _ clienttypes.Height, _ uint64, _ []byte) (uint64, error) {
	if s.err != nil {
		return 0, s.err
	}
	s.sequence++
	return s.sequence, nil
}

type packetReceiver struct {
	porttypes.IBCModule
	acks, timeouts int
	err            error
}

func (a *packetReceiver) OnAcknowledgementPacket(_ sdk.Context, _ string, _ channeltypes.Packet, _ []byte, _ sdk.AccAddress) error {
	a.acks++
	return a.err
}

func (a *packetReceiver) OnTimeoutPacket(_ sdk.Context, _ string, _ channeltypes.Packet, _ sdk.AccAddress) error {
	a.timeouts++
	return a.err
}

type middlewareFixture struct {
	keeper keeper.Keeper
	ctx    sdk.Context
	im     ibcratelimit.IBCMiddleware
	sender *packetSender
	app    *packetReceiver
}

func newMiddlewareFixture(t *testing.T) *middlewareFixture {
	return newMiddlewareFixtureWithActivation(t, true)
}

func newMiddlewareFixtureWithActivation(t *testing.T, enabled bool) *middlewareFixture {
	t.Helper()
	key := storetypes.NewKVStoreKey(types.StoreKey)
	db := dbm.NewMemDB()
	state := store.NewCommitMultiStore(db, log.NewNopLogger(), storemetrics.NewNoOpMetrics())
	state.SetIAVLSyncPruning(true)
	state.MountStoreWithDB(key, storetypes.StoreTypeIAVL, db)
	require.NoError(t, state.LoadLatestVersion())
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	k := keeper.NewKeeper(runtime.NewKVStoreService(key), cdc, "authority")
	ctx := sdk.NewContext(state, cmtproto.Header{Height: 100}, false, log.NewNopLogger())
	genesis := types.DefaultGenesis()
	genesis.RefundIdentityEnabled = false
	k.InitGenesis(ctx, genesis)
	if enabled {
		require.NoError(t, k.EnableRefundIdentity(ctx))
	}
	sender, app := &packetSender{}, &packetReceiver{}
	return &middlewareFixture{k, ctx, ibcratelimit.NewIBCMiddleware(app, sender, k), sender, app}
}

func (f *middlewareFixture) addRule(t *testing.T) {
	t.Helper()
	_, err := keeper.NewMsgServerImpl(f.keeper).AddRateLimit(f.ctx, &types.MsgAddRateLimit{
		Authority: "authority", ChannelId: "channel-0", Denom: "uzrn",
		MaxSend: "100", MaxRecv: "100", WindowBlocks: 10,
	})
	require.NoError(t, err)
}

func (f *middlewareFixture) send(t *testing.T, amount string) (uint64, error) {
	t.Helper()
	data, err := json.Marshal(transfertypes.FungibleTokenPacketData{Denom: "uzrn", Amount: amount})
	require.NoError(t, err)
	return f.im.SendPacket(f.ctx, "transfer", "channel-0", clienttypes.ZeroHeight(), 0, data)
}

func (f *middlewareFixture) mustSend(t *testing.T, amount string) uint64 {
	t.Helper()
	seq, err := f.send(t, amount)
	require.NoError(t, err)
	return seq
}

func (f *middlewareFixture) refund(t *testing.T, mode string, seq uint64) {
	t.Helper()
	packet := channeltypes.Packet{SourcePort: "transfer", SourceChannel: "channel-0", Sequence: seq}
	if mode == "timeout" {
		require.NoError(t, f.im.OnTimeoutPacket(f.ctx, "ics20-1", packet, nil))
	} else {
		ack := channeltypes.NewErrorAcknowledgement(errors.New("remote refusal"))
		require.NoError(t, f.im.OnAcknowledgementPacket(f.ctx, "ics20-1", packet, ack.Acknowledgement(), nil))
	}
	_, found := f.keeper.GetPacketFlow(f.ctx, "channel-0", seq)
	require.False(t, found, "terminal callback must clean up the flow")
}

func (f *middlewareFixture) requireUsage(t *testing.T, want string) {
	t.Helper()
	rl, found := f.keeper.GetRateLimit(f.ctx, "channel-0", "uzrn")
	require.True(t, found)
	require.Equal(t, want, rl.CurrentSend)
}

func (f *middlewareFixture) requireFull(t *testing.T) {
	t.Helper()
	f.requireUsage(t, "100")
	_, err := f.send(t, "1")
	require.ErrorIs(t, err, types.ErrSendRateExceeded)
}

func TestRefundUnchargedSendPreservesNewUsage(t *testing.T) {
	for _, initial := range []string{"unconfigured", "disabled"} {
		for _, mode := range []string{"timeout", "error_ack"} {
			t.Run(initial+"/"+mode, func(t *testing.T) {
				f := newMiddlewareFixture(t)
				if initial == "disabled" {
					f.addRule(t)
					f.keeper.SetParams(f.ctx, &types.Params{Enabled: false})
				}
				old := f.mustSend(t, "80")
				_, found := f.keeper.GetPacketFlow(f.ctx, "channel-0", old)
				if found {
					t.Error("uncharged send must not create a reversal record")
				}
				if initial == "unconfigured" {
					f.addRule(t)
				} else {
					f.keeper.SetParams(f.ctx, &types.Params{Enabled: true})
				}
				f.mustSend(t, "100")
				f.refund(t, mode, old)
				f.requireFull(t)
			})
		}
	}
}

func TestUnchargedRefundsCannotRecycleWindowBudget(t *testing.T) {
	for _, initial := range []string{"unconfigured", "disabled"} {
		t.Run(initial, func(t *testing.T) {
			f := newMiddlewareFixture(t)
			if initial == "disabled" {
				f.addRule(t)
				f.keeper.SetParams(f.ctx, &types.Params{Enabled: false})
			}
			old := []uint64{f.mustSend(t, "100"), f.mustSend(t, "100"), f.mustSend(t, "100")}
			if initial == "unconfigured" {
				f.addRule(t)
			} else {
				f.keeper.SetParams(f.ctx, &types.Params{Enabled: true})
			}
			f.mustSend(t, "100")
			accepted := 100
			for _, seq := range old {
				f.refund(t, "timeout", seq)
				if _, err := f.send(t, "100"); err == nil {
					accepted += 100
				} else {
					require.ErrorIs(t, err, types.ErrSendRateExceeded)
				}
			}
			t.Logf("accepted while enabled in one capped window: %d (cap 100)", accepted)
			require.Equal(t, 100, accepted, "old code admits 400 through the cap of 100")
			f.requireFull(t)
		})
	}
}

func TestRefundRemovedThenReaddedRulePreservesNewUsage(t *testing.T) {
	for _, mode := range []string{"timeout", "error_ack"} {
		t.Run(mode, func(t *testing.T) {
			f := newMiddlewareFixture(t)
			f.addRule(t)
			old := f.mustSend(t, "80")
			_, err := keeper.NewMsgServerImpl(f.keeper).RemoveRateLimit(f.ctx, &types.MsgRemoveRateLimit{
				Authority: "authority", ChannelId: "channel-0", Denom: "uzrn",
			})
			require.NoError(t, err)
			// Identical configuration, height and window: a config hash or creation
			// height alone cannot distinguish this new rule from the old one.
			f.addRule(t)
			f.mustSend(t, "100")
			f.refund(t, mode, old)
			f.requireFull(t)
		})
	}
}

func TestRefundPreviousWindowPreservesNewUsage(t *testing.T) {
	for _, reset := range []string{"send", "begin_block", "receive"} {
		for _, mode := range []string{"timeout", "error_ack"} {
			t.Run(reset+"/"+mode, func(t *testing.T) {
				f := newMiddlewareFixture(t)
				f.addRule(t)
				old := f.mustSend(t, "80")
				f.ctx = f.ctx.WithBlockHeight(110)
				switch reset {
				case "begin_block":
					f.keeper.ResetExpiredWindowsForActivation(f.ctx, true)
				case "receive":
					require.NoError(t, f.keeper.CheckAndUpdateRecvQuotaForActivation(f.ctx, true, "channel-0", "uzrn", big.NewInt(1)))
				}
				f.mustSend(t, "100")
				f.refund(t, mode, old)
				f.requireFull(t)
			})
		}
	}
}

func TestRefundSameRuleAndWindowReversesOnce(t *testing.T) {
	for _, mode := range []string{"timeout", "error_ack"} {
		t.Run(mode, func(t *testing.T) {
			f := newMiddlewareFixture(t)
			f.addRule(t)
			first, second := f.mustSend(t, "30"), f.mustSend(t, "40")
			f.refund(t, mode, first)
			f.requireUsage(t, "40")
			f.refund(t, mode, first)
			f.requireUsage(t, "40")
			// Disabling after a real charge does not erase its refund entitlement.
			f.keeper.SetParams(f.ctx, &types.Params{Enabled: false})
			f.refund(t, mode, second)
			f.requireUsage(t, "0")
			if mode == "timeout" {
				require.Equal(t, 3, f.app.timeouts)
			} else {
				require.Equal(t, 3, f.app.acks)
			}
		})
	}
}

func TestSendFailureOnlyReversesActualCharge(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		t.Run(map[bool]string{false: "disabled", true: "enabled"}[enabled], func(t *testing.T) {
			f := newMiddlewareFixture(t)
			f.addRule(t)
			f.mustSend(t, "40")
			f.keeper.SetParams(f.ctx, &types.Params{Enabled: enabled})
			f.sender.err = errors.New("send failed")
			seq, err := f.send(t, "20")
			require.ErrorIs(t, err, f.sender.err)
			require.Zero(t, seq)
			f.requireUsage(t, "40")
			_, found := f.keeper.GetPacketFlow(f.ctx, "channel-0", 2)
			require.False(t, found)
		})
	}
}

func TestSuccessfulAcknowledgementKeepsUsage(t *testing.T) {
	f := newMiddlewareFixture(t)
	f.addRule(t)
	seq := f.mustSend(t, "100")
	packet := channeltypes.Packet{SourceChannel: "channel-0", Sequence: seq}
	ack := channeltypes.NewResultAcknowledgement([]byte{1})
	require.NoError(t, f.im.OnAcknowledgementPacket(f.ctx, "ics20-1", packet, ack.Acknowledgement(), nil))
	f.requireFull(t)
	_, found := f.keeper.GetPacketFlow(f.ctx, "channel-0", seq)
	require.False(t, found)
	f.refund(t, "timeout", seq)
	f.requireFull(t)
}

func TestLegacyFlowWithoutChargeIdentityDoesNotReverse(t *testing.T) {
	for _, mode := range []string{"timeout", "error_ack"} {
		t.Run(mode, func(t *testing.T) {
			f := newMiddlewareFixture(t)
			f.addRule(t)
			f.keeper.SetPacketFlow(f.ctx, &types.PacketFlow{
				ChannelId: "channel-0", Sequence: 42, Denom: "uzrn", Amount: "80",
			})
			f.mustSend(t, "100")
			f.refund(t, mode, 42)
			f.requireFull(t)
		})
	}
}

func TestRefundCallbackFailureCallerCacheRollsBack(t *testing.T) {
	for _, mode := range []string{"timeout", "error_ack"} {
		t.Run(mode, func(t *testing.T) {
			f := newMiddlewareFixture(t)
			f.addRule(t)
			seq := f.mustSend(t, "100")
			parent := f.ctx
			cached, _ := parent.CacheContext()
			f.ctx = cached
			f.app.err = errors.New("upstream refund failed")
			packet := channeltypes.Packet{SourceChannel: "channel-0", Sequence: seq}
			var err error
			if mode == "timeout" {
				err = f.im.OnTimeoutPacket(cached, "ics20-1", packet, nil)
			} else {
				ack := channeltypes.NewErrorAcknowledgement(errors.New("remote refusal"))
				err = f.im.OnAcknowledgementPacket(cached, "ics20-1", packet, ack.Acknowledgement(), nil)
			}
			require.ErrorIs(t, err, f.app.err)
			f.requireUsage(t, "0")
			_, found := f.keeper.GetPacketFlow(cached, "channel-0", seq)
			require.False(t, found, "mutation happened only inside the failed cache")
			// This is the SDK caller contract: do not write a failed transaction's
			// cache. The original charge and receipt remain retryable.
			f.ctx = parent
			f.requireFull(t)
			_, found = f.keeper.GetPacketFlow(parent, "channel-0", seq)
			require.True(t, found)
			f.app.err = nil
			f.refund(t, mode, seq)
			f.requireUsage(t, "0")
		})
	}
}
