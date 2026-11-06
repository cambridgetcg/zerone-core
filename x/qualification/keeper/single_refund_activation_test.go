package keeper_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
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
	"google.golang.org/protobuf/proto"

	qualificationmodule "github.com/zerone-chain/zerone/x/qualification"
	"github.com/zerone-chain/zerone/x/qualification/keeper"
	"github.com/zerone-chain/zerone/x/qualification/types"
)

func refundRawState(h *refundHarness) map[string][]byte {
	out := make(map[string][]byte)
	iter := h.ctx.KVStore(h.key).Iterator(nil, nil)
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		out[string(iter.Key())] = bytes.Clone(iter.Value())
	}
	return out
}

func TestSingleRefundActivation(t *testing.T) {
	h := newLegacyRefundHarness(t)
	a := h.stake(t, "marker-owner")
	before := refundRawState(h)
	enabled, err := h.k.SingleRefundEnabled(h.ctx)
	require.NoError(t, err)
	require.False(t, enabled, "construction and explicitly legacy genesis must not activate")
	require.Equal(t, before, refundRawState(h), "getter is read-only")
	require.Equal(t, []byte{0x40}, types.SingleRefundEnabledKey)

	cache, write := h.ctx.CacheContext()
	require.NoError(t, h.k.EnableSingleRefund(cache))
	enabled, err = h.k.SingleRefundEnabled(cache)
	require.NoError(t, err)
	require.True(t, enabled)
	require.Equal(t, before, refundRawState(h), "uncommitted marker stays in caller cache")
	write()
	after := refundRawState(h)
	require.Equal(t, []byte{1}, after[string(types.SingleRefundEnabledKey)])
	delete(after, string(types.SingleRefundEnabledKey))
	require.Equal(t, before, after, "activation must not rewrite records, indexes or params")
	require.Equal(t, refundStake, h.escrow())
	require.Equal(t, "0", h.balance(a))
	before = refundRawState(h)
	require.NoError(t, h.k.EnableSingleRefund(h.ctx))
	require.Equal(t, before, refundRawState(h), "setter is idempotent")

	// A new keeper over the same store sees the persisted selector, not a field
	// in the old keeper. This is not a disk-restart or logical-genesis proof.
	other := keeper.NewKeeper(runtime.NewKVStoreService(h.key), codec.NewProtoCodec(codectypes.NewInterfaceRegistry()), h.k.GetAuthority(), h.bank)
	enabled, err = other.SingleRefundEnabled(h.ctx)
	require.NoError(t, err)
	require.True(t, enabled)
}

type refundMarkerFaultService struct {
	corestore.KVStoreService
	getErr, setErr error
}

func (s refundMarkerFaultService) OpenKVStore(ctx context.Context) corestore.KVStore {
	return refundMarkerFaultStore{KVStore: s.KVStoreService.OpenKVStore(ctx), getErr: s.getErr, setErr: s.setErr}
}

type refundMarkerFaultStore struct {
	corestore.KVStore
	getErr, setErr error
}

func (s refundMarkerFaultStore) Get(key []byte) ([]byte, error) {
	if bytes.Equal(key, types.SingleRefundEnabledKey) && s.getErr != nil {
		return nil, s.getErr
	}
	return s.KVStore.Get(key)
}

func (s refundMarkerFaultStore) Set(key, value []byte) error {
	if bytes.Equal(key, types.SingleRefundEnabledKey) && s.setErr != nil {
		return s.setErr
	}
	return s.KVStore.Set(key, value)
}

func TestSingleRefundMarkerErrorsRefuseBeforeEffects(t *testing.T) {
	for _, mode := range []string{"read_error", "zero", "unknown", "extra_byte", "empty"} {
		t.Run(mode, func(t *testing.T) {
			h := newLegacyRefundHarness(t)
			a := h.stake(t, "marker-fault")
			q := h.qualification(t, a)
			// If the marker read moves after decay, this record will be changed.
			params := h.k.GetParams(h.ctx)
			params.DecayCheckIntervalBlocks = 1
			params.DecayMinSamples = 1
			h.k.SetParams(h.ctx, params)
			q.Metrics = &types.QualificationMetrics{TotalVerifications: 100, CorrectVerifications: 1, AccuracyBps: 10_000}
			h.k.SetQualification(h.ctx, q)
			sentinel := errors.New("injected marker read failure")
			if mode == "read_error" {
				service := refundMarkerFaultService{KVStoreService: runtime.NewKVStoreService(h.key), getErr: sentinel}
				h.k = keeper.NewKeeper(service, codec.NewProtoCodec(codectypes.NewInterfaceRegistry()), h.k.GetAuthority(), h.bank)
			} else {
				value := map[string][]byte{"zero": {0}, "unknown": {2}, "extra_byte": {1, 0}, "empty": {}}[mode]
				h.ctx.KVStore(h.key).Set(types.SingleRefundEnabledKey, value)
			}
			h.ctx = h.ctx.WithBlockHeight(int64(q.ExpiresAt)).WithEventManager(sdk.NewEventManager())
			before := refundRawState(h)
			enabled, err := h.k.SingleRefundEnabled(h.ctx)
			require.False(t, enabled)
			require.Error(t, err)
			if mode == "read_error" {
				require.ErrorIs(t, err, sentinel)
			}
			require.Error(t, h.k.EnableSingleRefund(h.ctx), "setter must not repair corruption")
			require.Error(t, h.k.BeginBlocker(h.ctx))
			_, err = keeper.NewMsgServerImpl(h.k).WithdrawQualification(h.ctx, &types.MsgWithdrawQualification{Validator: a, Domain: "physics"})
			require.Error(t, err, "direct call must refuse before transfer or deletion")
			require.Equal(t, before, refundRawState(h))
			require.Empty(t, h.ctx.EventManager().Events())
			require.Zero(t, h.bank.refundCalls)
			require.Equal(t, refundStake, h.escrow())
			require.Equal(t, "0", h.balance(a))
		})
	}
}

func TestSingleRefundMarkerWriteError(t *testing.T) {
	h := newLegacyRefundHarness(t)
	before := refundRawState(h)
	sentinel := errors.New("injected marker write failure")
	service := refundMarkerFaultService{KVStoreService: runtime.NewKVStoreService(h.key), setErr: sentinel}
	k := keeper.NewKeeper(service, codec.NewProtoCodec(codectypes.NewInterfaceRegistry()), h.k.GetAuthority(), h.bank)
	require.ErrorIs(t, k.EnableSingleRefund(h.ctx), sentinel)
	enabled, err := k.SingleRefundEnabled(h.ctx)
	require.NoError(t, err)
	require.False(t, enabled)
	require.Equal(t, before, refundRawState(h))
}

// This deliberately preserves the predecessor's unsafe behavior while OFF.
// It is an activation-boundary characterization, not an acceptable paid-state
// invariant. The enabled refund suite separately requires single settlement.
func TestSingleRefundDisabledPreservesLegacy(t *testing.T) {
	for _, mode := range []string{"paid", "probationary", "fail_before", "fail_after", "locked", "nil_bank"} {
		t.Run(mode, func(t *testing.T) {
			h := newLegacyRefundHarness(t)
			if mode == "locked" {
				params := h.k.GetParams(h.ctx)
				params.QualificationPeriod, params.StakeLockPeriod = 10, 20
				h.k.SetParams(h.ctx, params)
			}
			a := h.stake(t, "legacy-off-a")
			q := h.qualification(t, a)
			if mode == "probationary" {
				q.Status = types.QualificationStatus_QUALIFICATION_STATUS_PROBATIONARY
				h.k.SetQualification(h.ctx, q)
				h.k.SetReviewNeutralityPolicy(func(context.Context) (bool, error) { return true, nil })
			}
			h.ctx = h.ctx.WithBlockHeight(h.ctx.BlockHeight() + 1)
			b := h.stake(t, "legacy-off-b")
			h.bank.failBefore, h.bank.failAfter = mode == "fail_before", mode == "fail_after"
			if mode == "nil_bank" {
				h.k = keeper.NewKeeper(runtime.NewKVStoreService(h.key), codec.NewProtoCodec(codectypes.NewInterfaceRegistry()), h.k.GetAuthority(), nil)
			}
			h.ctx = h.ctx.WithBlockHeight(int64(q.ExpiresAt)).WithEventManager(sdk.NewEventManager())
			require.NoError(t, h.k.BeginBlocker(h.ctx))
			q.Status = types.QualificationStatus_QUALIFICATION_STATUS_EXPIRED
			require.True(t, proto.Equal(q, h.qualification(t, a)), "legacy retains the amount even when paid")
			paid := mode == "paid" || mode == "probationary" || mode == "fail_after"
			if paid {
				require.Equal(t, refundStake, h.balance(a))
				require.Equal(t, refundStake, h.escrow())
			} else {
				require.Equal(t, "0", h.balance(a))
				require.Equal(t, "200000000", h.escrow())
			}
			var expiries int
			for _, event := range h.ctx.EventManager().Events() {
				require.NotEqual(t, "zerone.qualification.stake_refund_pending", event.Type)
				if event.Type == "zerone.qualification.qualification_expired" {
					expiries++
					require.Equal(t, sdk.NewEvent(event.Type, sdk.NewAttribute("validator", a), sdk.NewAttribute("domain", "physics")), event)
				}
			}
			require.Equal(t, 1, expiries)
			h.bank.failBefore, h.bank.failAfter = false, false
			if mode == "locked" {
				require.ErrorIs(t, h.withdraw(a), types.ErrStakeLocked)
				h.ctx = h.ctx.WithBlockHeight(int64(q.GrantedAt + h.k.GetParams(h.ctx).StakeLockPeriod))
			}
			require.NoError(t, h.withdraw(a), "no legacy quarantine before activation")
			_, found := h.k.GetQualification(h.ctx, a, "physics")
			require.False(t, found)
			if paid {
				require.Equal(t, "200000000", h.balance(a), "known predecessor double refund")
				require.Equal(t, "0", h.escrow())
			} else if mode == "nil_bank" {
				require.Equal(t, "0", h.balance(a), "legacy nil adapter deletes without paying")
				require.Equal(t, "200000000", h.escrow())
			} else {
				require.Equal(t, refundStake, h.balance(a))
				require.Equal(t, refundStake, h.escrow())
			}
			require.Equal(t, "0", h.balance(b))
		})
	}
}

func TestSingleRefundActivationHoldsActualLegacyExpiries(t *testing.T) {
	for _, mode := range []string{"paid", "failed", "locked"} {
		t.Run(mode, func(t *testing.T) {
			h := newLegacyRefundHarness(t)
			params := h.k.GetParams(h.ctx)
			params.QualificationPeriod, params.StakeLockPeriod = 10, 10
			if mode == "locked" {
				params.StakeLockPeriod = 20
			}
			h.k.SetParams(h.ctx, params)
			a := h.stake(t, "activation-legacy")
			q := h.qualification(t, a)
			h.bank.failBefore = mode == "failed"
			h.ctx = h.ctx.WithBlockHeight(int64(q.ExpiresAt))
			require.NoError(t, h.k.BeginBlocker(h.ctx), "produce the record through actual legacy expiry")
			q = h.qualification(t, a)
			require.Equal(t, types.QualificationStatus_QUALIFICATION_STATUS_EXPIRED, q.Status)
			require.Equal(t, refundStake, q.StakedAmount)
			balance, escrow := h.balance(a), h.escrow()
			require.NoError(t, h.k.EnableSingleRefund(h.ctx))
			h.bank.failBefore = false
			h.ctx = h.ctx.WithBlockHeight(int64(q.GrantedAt + params.StakeLockPeriod + 1)).WithEventManager(sdk.NewEventManager())
			before, calls := refundRawState(h), h.bank.refundCalls
			require.NoError(t, h.k.BeginBlocker(h.ctx))
			require.ErrorIs(t, h.withdraw(a), types.ErrAlreadyExpired)
			require.ErrorIs(t, h.k.QualifyByStake(h.ctx, a, "physics", refundStake), types.ErrQualificationExists,
				"the temporary hold also blocks requalification until separately authorized resolution")
			require.Equal(t, before, refundRawState(h))
			require.Equal(t, calls, h.bank.refundCalls)
			require.Equal(t, balance, h.balance(a))
			require.Equal(t, escrow, h.escrow())
			require.Empty(t, h.ctx.EventManager().Events())
		})
	}
}

func TestStakeRefundExpiryAtUnlockBoundary(t *testing.T) {
	for _, status := range []types.QualificationStatus{
		types.QualificationStatus_QUALIFICATION_STATUS_ACTIVE,
		types.QualificationStatus_QUALIFICATION_STATUS_PROBATIONARY,
	} {
		t.Run(status.String(), func(t *testing.T) {
			h := newRefundHarness(t)
			params := h.k.GetParams(h.ctx)
			params.QualificationPeriod, params.StakeLockPeriod = 10, 10
			h.k.SetParams(h.ctx, params)
			a := h.stake(t, "exact-boundary")
			q := h.qualification(t, a)
			q.Status = status
			h.k.SetQualification(h.ctx, q)
			require.Equal(t, q.ExpiresAt, q.GrantedAt+params.StakeLockPeriod)
			h.ctx = h.ctx.WithBlockHeight(int64(q.ExpiresAt - 1)).WithEventManager(sdk.NewEventManager())
			require.NoError(t, h.k.BeginBlocker(h.ctx))
			require.Zero(t, h.bank.refundCalls)
			require.Equal(t, "0", h.balance(a))
			require.Equal(t, refundStake, h.escrow())
			require.True(t, proto.Equal(q, h.qualification(t, a)))
			h.ctx = h.ctx.WithBlockHeight(int64(q.ExpiresAt))
			require.NoError(t, h.k.BeginBlocker(h.ctx))
			require.Equal(t, 1, h.bank.refundCalls, "refund happens AT expiry, not one block later")
			require.Equal(t, refundStake, h.balance(a))
			require.Equal(t, "0", h.escrow())
			expired := h.qualification(t, a)
			require.Equal(t, types.QualificationStatus_QUALIFICATION_STATUS_EXPIRED, expired.Status)
			require.Empty(t, expired.StakedAmount)
			for _, event := range h.ctx.EventManager().Events() {
				require.NotEqual(t, "zerone.qualification.stake_refund_pending", event.Type)
			}
			require.NoError(t, h.withdraw(a))
			require.Equal(t, 1, h.bank.refundCalls)
			require.Equal(t, refundStake, h.balance(a))
		})
	}
}

func TestStakeRefundPendingEventPairing(t *testing.T) {
	for _, reason := range []string{"stake_locked", "transfer_failed"} {
		t.Run(reason, func(t *testing.T) {
			h := newRefundHarness(t)
			params := h.k.GetParams(h.ctx)
			params.QualificationPeriod, params.StakeLockPeriod = 10, 10
			if reason == "stake_locked" {
				params.StakeLockPeriod = 20
			} else {
				h.bank.failAfter = true
			}
			h.k.SetParams(h.ctx, params)
			a := h.stake(t, "paired-event")
			q := h.qualification(t, a)
			h.ctx = h.ctx.WithBlockHeight(int64(q.ExpiresAt)).WithEventManager(sdk.NewEventManager())
			require.NoError(t, h.k.BeginBlocker(h.ctx))
			require.Equal(t, sdk.Events{
				sdk.NewEvent("zerone.qualification.stake_refund_pending", sdk.NewAttribute("validator", a), sdk.NewAttribute("domain", "physics"), sdk.NewAttribute("reason", reason)),
				sdk.NewEvent("zerone.qualification.qualification_expired", sdk.NewAttribute("validator", a), sdk.NewAttribute("domain", "physics")),
			}, h.ctx.EventManager().Events())
			require.Equal(t, types.QualificationStatus_QUALIFICATION_STATUS_SUSPENDED, h.qualification(t, a).Status)
			require.Equal(t, refundStake, h.qualification(t, a).StakedAmount)
			require.Equal(t, refundStake, h.escrow())
			require.Equal(t, "0", h.balance(a))
		})
	}
}

// These tests hold the consensus-repairs-v1 marker for zerone-core #120 to the
// module's side of the activation contract (internal/activation tests the reads
// and writes): the key is pinned, native genesis selects the marker, the
// genesis field writes the marker and nothing else, genesis JSON carries it
// exactly with a file written before the field existing importing as off, and
// a malformed marker is never exported as off. Every import and export goes
// through the module's own genesis JSON path. The repair's own tests can build
// on the fixture: genesisJSON(t, &off) turns the marker off and
// EnableSingleRefund turns it on.

const singleRefundGenesisField = "single_refund_enabled"

type singleRefundFixture struct {
	ctx sdk.Context
	key storetypes.StoreKey
	k   keeper.Keeper
	am  qualificationmodule.AppModule
	cdc codec.JSONCodec
}

func newSingleRefundFixture(t *testing.T) *singleRefundFixture {
	t.Helper()
	key := storetypes.NewKVStoreKey(types.StoreKey)
	db := dbm.NewMemDB()
	ms := store.NewCommitMultiStore(db, log.NewNopLogger(), storemetrics.NewNoOpMetrics())
	ms.MountStoreWithDB(key, storetypes.StoreTypeIAVL, db)
	require.NoError(t, ms.LoadLatestVersion())
	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	k := keeper.NewKeeper(runtime.NewKVStoreService(key), cdc, "authority", nil)
	ctx := sdk.NewContext(ms, cmtproto.Header{ChainID: "consensus-repairs-v1-marker", Height: 100}, false, log.NewNopLogger())
	return &singleRefundFixture{ctx: ctx, key: key, k: k, am: qualificationmodule.NewAppModule(cdc, k), cdc: cdc}
}

func (f *singleRefundFixture) marker() []byte {
	return f.ctx.KVStore(f.key).Get(types.SingleRefundEnabledKey)
}

func (f *singleRefundFixture) enabled(t *testing.T) bool {
	t.Helper()
	enabled, err := f.k.SingleRefundEnabled(f.ctx)
	require.NoError(t, err)
	return enabled
}

func (f *singleRefundFixture) initGenesis(t *testing.T, raw json.RawMessage) {
	t.Helper()
	require.NoError(t, f.am.ValidateGenesis(f.cdc, nil, raw))
	f.am.InitGenesis(f.ctx, f.cdc, raw)
}

// genesisJSON is the module's default genesis JSON with the marker field set
// on, set off, or (nil) removed, as in a file written before the field existed.
func (f *singleRefundFixture) genesisJSON(t *testing.T, field *bool) json.RawMessage {
	t.Helper()
	var fields map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(f.am.DefaultGenesis(f.cdc), &fields))
	require.JSONEq(t, "true", string(fields[singleRefundGenesisField]), "native genesis selects the marker")
	if field == nil {
		delete(fields, singleRefundGenesisField)
	} else {
		fields[singleRefundGenesisField] = json.RawMessage(strconv.FormatBool(*field))
	}
	raw, err := json.Marshal(fields)
	require.NoError(t, err)
	return raw
}

func (f *singleRefundFixture) snapshot(t *testing.T) map[string]string {
	t.Helper()
	iter := f.ctx.KVStore(f.key).Iterator(nil, nil)
	defer iter.Close()
	entries := map[string]string{}
	for ; iter.Valid(); iter.Next() {
		entries[string(iter.Key())] = string(iter.Value())
	}
	require.NoError(t, iter.Error())
	return entries
}

// singleRefundExported reads the marker field of exported genesis JSON. Encoders that
// omit false fields leave it out, which reads as off.
func singleRefundExported(t *testing.T, raw json.RawMessage) bool {
	t.Helper()
	var fields map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(raw, &fields))
	value, ok := fields[singleRefundGenesisField]
	if !ok {
		return false
	}
	var on bool
	require.NoError(t, json.Unmarshal(value, &on))
	return on
}

func TestSingleRefundGenesisFieldWritesOnlyTheMarker(t *testing.T) {
	enabled, disabled := true, false
	on, off := newSingleRefundFixture(t), newSingleRefundFixture(t)
	on.initGenesis(t, on.genesisJSON(t, &enabled))
	off.initGenesis(t, off.genesisJSON(t, &disabled))
	want := off.snapshot(t)
	want[string(types.SingleRefundEnabledKey)] = "\x01"
	require.Equal(t, want, on.snapshot(t), "the genesis field writes the marker and nothing else")
}

// A marker that is neither absent nor {1} is a fault, so ExportGenesis panics
// rather than export it as off.
func TestSingleRefundExportRefusesMalformedMarker(t *testing.T) {
	f := newSingleRefundFixture(t)
	f.initGenesis(t, f.genesisJSON(t, nil))
	for _, bad := range [][]byte{{0}, {2}} {
		t.Run(fmt.Sprintf("%x", bad), func(t *testing.T) {
			f.ctx.KVStore(f.key).Set(types.SingleRefundEnabledKey, bad)
			require.Panics(t, func() { f.am.ExportGenesis(f.ctx, f.cdc) }, "a malformed marker is never exported as off")
		})
	}
}

func TestSingleRefundNativeGenesisSelectsMarker(t *testing.T) {
	require.Equal(t, []byte{0x40}, types.SingleRefundEnabledKey)
	gs := types.DefaultGenesis()
	require.True(t, gs.SingleRefundEnabled, "a new chain starts with the repair selected")
	require.NoError(t, gs.Validate())
	gs.SingleRefundEnabled = false
	require.NoError(t, gs.Validate(), "off stays valid")

	f := newSingleRefundFixture(t)
	f.initGenesis(t, f.am.DefaultGenesis(f.cdc))
	require.Equal(t, []byte{1}, f.marker())
	require.True(t, f.enabled(t))
}

func TestSingleRefundGenesisRoundTripsMarker(t *testing.T) {
	on, off := true, false
	for _, tc := range []struct {
		name  string
		field *bool
		want  bool
	}{
		{"on", &on, true},
		{"off", &off, false},
		{"file without the field", nil, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := newSingleRefundFixture(t)
			source.initGenesis(t, source.genesisJSON(t, tc.field))
			require.Equal(t, tc.want, source.enabled(t))
			if !tc.want {
				require.Nil(t, source.marker(), "off writes no marker")
			}
			exported := source.am.ExportGenesis(source.ctx, source.cdc)
			require.Equal(t, tc.want, singleRefundExported(t, exported))

			imported := newSingleRefundFixture(t)
			imported.initGenesis(t, exported)
			require.Equal(t, tc.want, imported.enabled(t))
			require.Equal(t, tc.want, singleRefundExported(t, imported.am.ExportGenesis(imported.ctx, imported.cdc)))
		})
	}
}
