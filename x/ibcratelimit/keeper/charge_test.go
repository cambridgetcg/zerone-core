package keeper

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"math"
	"math/big"
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
	"github.com/stretchr/testify/require"

	"github.com/zerone-chain/zerone/x/ibcratelimit/types"
)

func chargeFixture(t *testing.T) (Keeper, sdk.Context, *storetypes.KVStoreKey) {
	return chargeFixtureWithActivation(t, true)
}

func chargeFixtureWithActivation(t *testing.T, enabled bool) (Keeper, sdk.Context, *storetypes.KVStoreKey) {
	t.Helper()
	key := storetypes.NewKVStoreKey(types.StoreKey)
	db := dbm.NewMemDB()
	state := store.NewCommitMultiStore(db, log.NewNopLogger(), storemetrics.NewNoOpMetrics())
	state.SetIAVLSyncPruning(true)
	state.MountStoreWithDB(key, storetypes.StoreTypeIAVL, db)
	require.NoError(t, state.LoadLatestVersion())
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	k := NewKeeper(runtime.NewKVStoreService(key), cdc, "authority")
	ctx := sdk.NewContext(state, cmtproto.Header{Height: 1}, false, log.NewNopLogger())
	genesis := types.DefaultGenesis()
	genesis.RefundIdentityEnabled = false
	k.InitGenesis(ctx, genesis)
	if enabled {
		require.NoError(t, k.EnableRefundIdentity(ctx))
	}
	return k, ctx, key
}

func chargeRule() *types.RateLimit {
	return &types.RateLimit{
		ChannelId: "channel-0", Denom: "uzrn", MaxSend: "100", MaxRecv: "100",
		WindowBlocks: 100, WindowStart: 0, CurrentSend: "0", CurrentRecv: "0",
	}
}

func TestChargeExistingRuleUsesExplicitZeroIdentity(t *testing.T) {
	k, ctx, key := chargeFixture(t)
	// These are the exact legacy JSON shape and key, bypassing the new setter.
	legacy, err := json.Marshal(chargeRule())
	require.NoError(t, err)
	ctx.KVStore(key).Set(rateLimitKey("channel-0", "uzrn"), legacy)
	require.Nil(t, ctx.KVStore(key).Get(ruleGenerationKey("channel-0", "uzrn")))
	charge, err := k.ChargeSendQuota(ctx, "channel-0", "uzrn", big.NewInt(30))
	require.NoError(t, err)
	require.NotNil(t, charge)
	require.Zero(t, charge.ruleGeneration)
	require.Zero(t, charge.windowStart)
	k.RecordPacketCharge(ctx, 1, charge)
	var record packetChargeRecord
	require.NoError(t, json.Unmarshal(ctx.KVStore(key).Get(packetFlowKey("channel-0", 1)), &record))
	require.NotNil(t, record.RuleGeneration)
	require.NotNil(t, record.WindowStart)
	require.Zero(t, *record.RuleGeneration)
	require.Zero(t, *record.WindowStart)
	flow, found := k.GetPacketFlow(ctx, "channel-0", 1)
	require.True(t, found, "legacy public flow reader still decodes the four original fields")
	require.Equal(t, "30", flow.Amount)
	// A newly constructed keeper must get the identity from state, not memory.
	reopened := NewKeeper(k.storeService, k.cdc, k.authority)
	reopened.ReversePacketFlow(ctx, "channel-0", 1)
	rl, found := reopened.GetRateLimit(ctx, "channel-0", "uzrn")
	require.True(t, found)
	require.Equal(t, "0", rl.CurrentSend)

	// A legacy generation-zero rule must also separate a subsequent re-add.
	k.DeletePacketFlowForActivation(ctx, true, "channel-0", 1)
	old, err := k.ChargeSendQuota(ctx, "channel-0", "uzrn", big.NewInt(30))
	require.NoError(t, err)
	k.RecordPacketCharge(ctx, 2, old)
	k.DeleteRateLimit(ctx, "channel-0", "uzrn")
	k.SetRateLimitForActivation(ctx, true, chargeRule())
	require.Equal(t, uint64(1), k.ruleGeneration(ctx, "channel-0", "uzrn"))
	_, err = k.ChargeSendQuota(ctx, "channel-0", "uzrn", big.NewInt(100))
	require.NoError(t, err)
	k.ReversePacketFlow(ctx, "channel-0", 2)
	rl, _ = k.GetRateLimit(ctx, "channel-0", "uzrn")
	require.Equal(t, "100", rl.CurrentSend)
}

func TestRuleGenerationOnlyChangesForRuleReplacement(t *testing.T) {
	k, ctx, _ := chargeFixture(t)
	k.SetRateLimitForActivation(ctx, true, chargeRule())
	first := k.ruleGeneration(ctx, "channel-0", "uzrn")
	require.Equal(t, uint64(1), first)
	charge, err := k.ChargeSendQuota(ctx, "channel-0", "uzrn", big.NewInt(30))
	require.NoError(t, err)
	require.NoError(t, k.CheckAndUpdateRecvQuotaForActivation(ctx, true, "channel-0", "uzrn", big.NewInt(20)))
	require.Equal(t, first, k.ruleGeneration(ctx, "channel-0", "uzrn"))
	k.ReverseSendCharge(ctx, charge)
	require.Equal(t, first, k.ruleGeneration(ctx, "channel-0", "uzrn"))
	k.ResetExpiredWindowsForActivation(ctx.WithBlockHeight(100), true)
	require.Equal(t, first, k.ruleGeneration(ctx, "channel-0", "uzrn"))
	k.DeleteRateLimit(ctx, "channel-0", "uzrn")
	require.Equal(t, first, k.ruleGeneration(ctx, "channel-0", "uzrn"), "tombstone retained")
	k.SetRateLimitForActivation(ctx, true, chargeRule())
	require.Equal(t, first+1, k.ruleGeneration(ctx, "channel-0", "uzrn"))
	for _, field := range []string{"send", "receive", "window"} {
		rl, _ := k.GetRateLimit(ctx, "channel-0", "uzrn")
		before := k.ruleGeneration(ctx, "channel-0", "uzrn")
		switch field {
		case "send":
			rl.MaxSend = "200"
		case "receive":
			rl.MaxRecv = "200"
		case "window":
			rl.WindowBlocks = 200
		}
		k.SetRateLimitForActivation(ctx, true, rl)
		require.Equal(t, before+1, k.ruleGeneration(ctx, "channel-0", "uzrn"))
	}
}

func TestRefundIgnoresInPlaceRuleReplacement(t *testing.T) {
	k, ctx, _ := chargeFixture(t)
	k.SetRateLimitForActivation(ctx, true, chargeRule())
	charge, err := k.ChargeSendQuota(ctx, "channel-0", "uzrn", big.NewInt(30))
	require.NoError(t, err)
	k.RecordPacketCharge(ctx, 1, charge)
	rl := chargeRule()
	rl.MaxSend = "200"
	rl.CurrentSend = "100"
	k.SetRateLimitForActivation(ctx, true, rl)
	k.ReversePacketFlow(ctx, "channel-0", 1)
	rl, _ = k.GetRateLimit(ctx, "channel-0", "uzrn")
	require.Equal(t, "100", rl.CurrentSend)
}

func TestRefundExpiredUnresetWindowDoesNotWrite(t *testing.T) {
	for _, height := range []int64{100, 101, -1} {
		t.Run(big.NewInt(height).String(), func(t *testing.T) {
			k, ctx, key := chargeFixture(t)
			k.SetRateLimitForActivation(ctx, true, chargeRule())
			charge, err := k.ChargeSendQuota(ctx, "channel-0", "uzrn", big.NewInt(30))
			require.NoError(t, err)
			before := rawChargeState(ctx, key)
			k.ReverseSendCharge(ctx.WithBlockHeight(height), charge)
			require.Equal(t, before, rawChargeState(ctx, key))
		})
	}
}

func TestMalformedChargeMetadataCannotAuthorizeRefund(t *testing.T) {
	for _, raw := range []string{
		`{`,
		`{"channel_id":"channel-0","sequence":1,"denom":"uzrn","amount":"30"}`,
		`{"channel_id":"channel-0","sequence":1,"denom":"uzrn","amount":"30","window_start":0}`,
		`{"channel_id":"channel-0","sequence":1,"denom":"uzrn","amount":"30","rule_generation":1}`,
		`{"channel_id":"channel-0","sequence":1,"denom":"uzrn","amount":"-30","window_start":0,"rule_generation":1}`,
		`{"channel_id":"channel-0","sequence":1,"denom":"uzrn","amount":"bad","window_start":0,"rule_generation":1}`,
		`{"channel_id":"channel-1","sequence":1,"denom":"uzrn","amount":"30","window_start":0,"rule_generation":1}`,
		`{"channel_id":"channel-0","sequence":2,"denom":"uzrn","amount":"30","window_start":0,"rule_generation":1}`,
	} {
		t.Run(raw, func(t *testing.T) {
			k, ctx, key := chargeFixture(t)
			k.SetRateLimitForActivation(ctx, true, chargeRule())
			// Also charge channel-1: the record's channel binding must be what
			// prevents a corrupt channel-0 record from debiting that other rule.
			other := chargeRule()
			other.ChannelId = "channel-1"
			k.SetRateLimitForActivation(ctx, true, other)
			_, err := k.ChargeSendQuota(ctx, "channel-1", "uzrn", big.NewInt(100))
			require.NoError(t, err)
			_, err = k.ChargeSendQuota(ctx, "channel-0", "uzrn", big.NewInt(100))
			require.NoError(t, err)
			ctx.KVStore(key).Set(packetFlowKey("channel-0", 1), []byte(raw))
			before := rawChargeState(ctx, key)
			k.ReversePacketFlow(ctx, "channel-0", 1)
			require.Equal(t, before, rawChargeState(ctx, key))
		})
	}
}

func TestRuleGenerationCannotWrapOrDecodeCorruptState(t *testing.T) {
	for _, malformed := range []bool{false, true} {
		k, ctx, key := chargeFixture(t)
		bz := make([]byte, 8)
		binary.BigEndian.PutUint64(bz, math.MaxUint64)
		if malformed {
			bz = []byte{1}
		}
		ctx.KVStore(key).Set(ruleGenerationKey("channel-0", "uzrn"), bz)
		before := rawChargeState(ctx, key)
		require.Panics(t, func() { k.SetRateLimitForActivation(ctx, true, chargeRule()) })
		require.Equal(t, before, rawChargeState(ctx, key))
	}
}

func rawChargeState(ctx sdk.Context, key *storetypes.KVStoreKey) map[string][]byte {
	result := make(map[string][]byte)
	iter := ctx.KVStore(key).Iterator(nil, nil)
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		result[string(iter.Key())] = bytes.Clone(iter.Value())
	}
	return result
}

// The runtime adapter normally wraps void SDK writes and returns nil. These
// synthetic errors exercise the keeper's advertised KVStoreService contract,
// not a claim about real disk failures. Failed caches must be discarded.
type chargeFaultService struct {
	corestore.KVStoreService
	prefix byte
	after  bool
	delete bool
}

type chargeFaultStore struct {
	corestore.KVStore
	fault *chargeFaultService
}

func (s *chargeFaultService) OpenKVStore(ctx context.Context) corestore.KVStore {
	return chargeFaultStore{s.KVStoreService.OpenKVStore(ctx), s}
}

func (s chargeFaultStore) Set(key, value []byte) error {
	if s.fault.delete || key[0] != s.fault.prefix {
		return s.KVStore.Set(key, value)
	}
	if s.fault.after {
		if err := s.KVStore.Set(key, value); err != nil {
			return err
		}
	}
	return errors.New("injected charge write failure")
}

func (s chargeFaultStore) Delete(key []byte) error {
	if !s.fault.delete || key[0] != s.fault.prefix {
		return s.KVStore.Delete(key)
	}
	if s.fault.after {
		if err := s.KVStore.Delete(key); err != nil {
			return err
		}
	}
	return errors.New("injected charge delete failure")
}

func TestChargeWriteFailuresAbortAndCallerCacheRollsBack(t *testing.T) {
	for _, operation := range []string{"generation", "quota", "packet", "cleanup"} {
		for _, after := range []bool{false, true} {
			t.Run(operation+map[bool]string{false: "/before", true: "/after"}[after], func(t *testing.T) {
				k, ctx, key := chargeFixture(t)
				k.SetRateLimitForActivation(ctx, true, chargeRule())
				charge, err := k.ChargeSendQuota(ctx, "channel-0", "uzrn", big.NewInt(40))
				require.NoError(t, err)
				k.RecordPacketCharge(ctx, 1, charge)
				before := rawChargeState(ctx, key)
				cached, _ := ctx.CacheContext()
				fault := &chargeFaultService{KVStoreService: k.storeService, after: after}
				bad := NewKeeper(fault, k.cdc, k.authority)
				switch operation {
				case "generation":
					fault.prefix = types.RuleGenerationPrefix[0]
					rl := chargeRule()
					rl.MaxSend = "200"
					require.Panics(t, func() { bad.SetRateLimitForActivation(cached, true, rl) })
				case "quota":
					fault.prefix = types.RateLimitKeyPrefix[0]
					require.Panics(t, func() {
						_, _ = bad.ChargeSendQuota(cached, "channel-0", "uzrn", big.NewInt(20))
					})
				case "packet":
					fault.prefix = types.PacketFlowPrefix[0]
					newCharge, err := bad.ChargeSendQuota(cached, "channel-0", "uzrn", big.NewInt(20))
					require.NoError(t, err)
					require.Panics(t, func() { bad.RecordPacketCharge(cached, 2, newCharge) })
				case "cleanup":
					fault.prefix, fault.delete = types.PacketFlowPrefix[0], true
					bad.ReversePacketFlow(cached, "channel-0", 1)
					require.Panics(t, func() { bad.DeletePacketFlowForActivation(cached, true, "channel-0", 1) })
				}
				// Never call the cache writer after an error/panic. Parent bytes,
				// including usage, generation and the outstanding receipt, survive.
				require.Equal(t, before, rawChargeState(ctx, key))
				// A normal terminal retry reverses the original charge, not a second one.
				k.ReversePacketFlow(ctx, "channel-0", 1)
				k.DeletePacketFlowForActivation(ctx, true, "channel-0", 1)
				rl, _ := k.GetRateLimit(ctx, "channel-0", "uzrn")
				require.Equal(t, "0", rl.CurrentSend)
			})
		}
	}
}
