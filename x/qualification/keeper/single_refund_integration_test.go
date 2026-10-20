package keeper_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"testing"

	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/zerone-chain/zerone/x/qualification/keeper"
	"github.com/zerone-chain/zerone/x/qualification/types"
)

type refundTraceStep struct {
	State  map[string]map[string]string
	Events sdk.Events
	Error  string
	Gas    uint64
}

// Record every raw qualification, bank and auth KV entry, ordered events, the
// exact returned error and gas for one call. Inspection uses a separate meter.
func snapshotRefundTrace(t *testing.T, h *refundHarness, err error) refundTraceStep {
	t.Helper()
	step := refundTraceStep{Gas: h.ctx.GasMeter().GasConsumed(), Events: h.ctx.EventManager().Events(), State: make(map[string]map[string]string)}
	if err != nil {
		step.Error = err.Error()
	}
	ctx := h.ctx.WithGasMeter(storetypes.NewInfiniteGasMeter())
	for name, key := range h.keys {
		entries := make(map[string]string)
		iter := ctx.KVStore(key).Iterator(nil, nil)
		for ; iter.Valid(); iter.Next() {
			entries[hex.EncodeToString(iter.Key())] = hex.EncodeToString(iter.Value())
		}
		require.NoError(t, iter.Error())
		require.NoError(t, iter.Close())
		step.State[name] = entries
	}
	return step
}

// Goldens come from executing these fixtures with the exact begin_blocker.go
// and msg_server.go at 77521a0daf14420836483ae9d8ea3fbcc1ae0d40 via a Go compiler
// overlay (the pre-PR120 implementation), not from the repaired implementation.
// The current genesis schema is shared by both runs with the marker explicitly
// off. This pins raw state effects, full event ordering, errors AND gas, including
// the unsafe legacy double refund and uncached post-transfer failure. It is not
// a whole-app replay/AppHash assertion. Never refresh from the enabled path.
func TestSingleRefundDisabledPredecessorTrace(t *testing.T) {
	for _, tc := range []struct{ mode, digest string }{
		{"paid", "ae85ca5bc610590bbdb5f6ce30e635d9cd13c57f2daccfea670a260d2fef0213"},
		{"probationary", "030a662e4a5eb3647ba1be06eba404c4924e2d41cd8535dc591a1c7cb6502b05"},
		{"fail_before", "da7085e307120daee0e1e16f6efa4ee7eef5662aea03bd6d91fc38c69299a12e"},
		{"fail_after", "1eac44dd582fc8915afcb157a1cfc345e39cc73817dd2e49b4833689eca7c0c4"},
		{"locked", "ce8cf61f0d3f070d0769198861ba29d23b14f93d6e5863a89810912f79cf94e9"},
		{"nil_bank", "ced970292c1c43b14c4c5beee64c07ec9805253c3d7f82ed478233effb8e9f20"},
	} {
		t.Run(tc.mode, func(t *testing.T) {
			h := newLegacyRefundHarness(t)
			params := h.k.GetParams(h.ctx)
			params.QualificationPeriod, params.StakeLockPeriod = 10, 10
			if tc.mode == "locked" {
				params.StakeLockPeriod = 20
			}
			h.k.SetParams(h.ctx, params)
			a := h.stake(t, "trace-a")
			q := h.qualification(t, a)
			if tc.mode == "probationary" {
				q.Status = types.QualificationStatus_QUALIFICATION_STATUS_PROBATIONARY
				q.ProbationUntil = q.ExpiresAt + 5
				h.k.SetQualification(h.ctx, q)
			}
			h.ctx = h.ctx.WithBlockHeight(h.ctx.BlockHeight() + 1)
			h.stake(t, "trace-b")
			_, err := keeper.NewMsgServerImpl(h.k).EndorseQualification(h.ctx, &types.MsgEndorseQualification{Endorser: testAddr("trace-endorser"), Validator: a, Domain: "physics", Weight: 1})
			require.NoError(t, err)
			h.bank.failBefore, h.bank.failAfter = tc.mode == "fail_before", tc.mode == "fail_after"
			if tc.mode == "nil_bank" {
				h.k = keeper.NewKeeper(runtime.NewKVStoreService(h.key), codec.NewProtoCodec(codectypes.NewInterfaceRegistry()), h.k.GetAuthority(), nil)
			}
			h.ctx = h.ctx.WithBlockHeight(int64(q.ExpiresAt)).WithEventManager(sdk.NewEventManager()).WithGasMeter(storetypes.NewGasMeter(1_000_000))
			err = h.k.BeginBlocker(h.ctx)
			trace := []refundTraceStep{snapshotRefundTrace(t, h, err)}
			require.NoError(t, err)
			// The first withdrawal retains the injected fault or lock; retry only
			// after resetting that condition, then try a duplicate withdrawal.
			for attempt := 0; attempt < 3; attempt++ {
				if attempt == 1 {
					h.bank.failBefore, h.bank.failAfter = false, false
					h.ctx = h.ctx.WithBlockHeight(int64(q.GrantedAt + params.StakeLockPeriod))
				}
				h.ctx = h.ctx.WithEventManager(sdk.NewEventManager()).WithGasMeter(storetypes.NewGasMeter(1_000_000))
				err = h.withdraw(a)
				trace = append(trace, snapshotRefundTrace(t, h, err))
			}
			encoded, err := json.Marshal(trace) // JSON sorts map keys deterministically.
			require.NoError(t, err)
			digest := fmt.Sprintf("%x", sha256.Sum256(encoded))
			t.Logf("predecessor trace %s: %s; gas=%d,%d,%d,%d", tc.mode, digest, trace[0].Gas, trace[1].Gas, trace[2].Gas, trace[3].Gas)
			require.Equal(t, tc.digest, digest)
		})
	}
}

func TestSingleRefundPendingExpiryClearsProbationOnlyWhenEnabled(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		for _, reason := range []string{"stake_locked", "transfer_failed"} {
			t.Run(fmt.Sprintf("enabled=%t/%s", enabled, reason), func(t *testing.T) {
				h := newLegacyRefundHarness(t)
				if enabled {
					require.NoError(t, h.k.EnableSingleRefund(h.ctx))
				}
				h.k.SetReviewNeutralityPolicy(func(context.Context) (bool, error) { return true, nil })
				params := h.k.GetParams(h.ctx)
				params.QualificationPeriod, params.StakeLockPeriod = 10, 10
				if reason == "stake_locked" {
					params.StakeLockPeriod = 20
				} else {
					h.bank.failAfter = true
				}
				h.k.SetParams(h.ctx, params)
				a := h.stake(t, "pending-probation")
				q := h.qualification(t, a)
				q.Status = types.QualificationStatus_QUALIFICATION_STATUS_PROBATIONARY
				q.ProbationUntil = q.ExpiresAt + 5
				h.k.SetQualification(h.ctx, q)
				h.ctx = h.ctx.WithBlockHeight(int64(q.ExpiresAt))
				require.NoError(t, h.k.BeginBlocker(h.ctx))
				want := proto.Clone(q).(*types.DomainQualification)
				want.Status = types.QualificationStatus_QUALIFICATION_STATUS_EXPIRED
				if enabled {
					want.Status = types.QualificationStatus_QUALIFICATION_STATUS_SUSPENDED
					want.ProbationUntil = 0
				}
				require.True(t, proto.Equal(want, h.qualification(t, a)))
			})
		}
	}
}

func TestSingleRefundActivationSurfacesDepletedLegacyEscrow(t *testing.T) {
	h := newLegacyRefundHarness(t)
	a := h.stake(t, "depleted-a")
	q := h.qualification(t, a)
	h.ctx = h.ctx.WithBlockHeight(h.ctx.BlockHeight() + 1)
	b := h.stake(t, "depleted-b")
	h.ctx = h.ctx.WithBlockHeight(int64(q.ExpiresAt))
	require.NoError(t, h.k.BeginBlocker(h.ctx))
	require.NoError(t, h.withdraw(a), "actual legacy double refund drains B's backing")
	require.Equal(t, "200000000", h.balance(a))
	require.Equal(t, "0", h.escrow())
	require.NoError(t, h.k.EnableSingleRefund(h.ctx))
	h.ctx = h.ctx.WithBlockHeight(h.ctx.BlockHeight() + 1).WithEventManager(sdk.NewEventManager())
	require.NoError(t, h.k.BeginBlocker(h.ctx), "a real insufficient-funds error must not halt the block")
	pending := h.qualification(t, b)
	require.Equal(t, types.QualificationStatus_QUALIFICATION_STATUS_SUSPENDED, pending.Status)
	require.Equal(t, refundStake, pending.StakedAmount)
	require.Equal(t, sdk.Events{
		sdk.NewEvent("zerone.qualification.stake_refund_pending", sdk.NewAttribute("validator", b), sdk.NewAttribute("domain", "physics"), sdk.NewAttribute("reason", "transfer_failed")),
		sdk.NewEvent("zerone.qualification.qualification_expired", sdk.NewAttribute("validator", b), sdk.NewAttribute("domain", "physics")),
	}, h.ctx.EventManager().Events())
	before := refundRawState(h)
	h.ctx = h.ctx.WithEventManager(sdk.NewEventManager())
	require.ErrorContains(t, h.withdraw(b), "insufficient funds")
	require.Equal(t, before, refundRawState(h))
	require.Empty(t, h.ctx.EventManager().Events())
	require.Equal(t, "200000000", h.balance(a), "no clawback or redistribution")
	require.Equal(t, "0", h.balance(b))
	require.Equal(t, "0", h.escrow())
}

func TestSingleRefundActivationPreservesOtherLegacyWithdrawals(t *testing.T) {
	for _, status := range []types.QualificationStatus{
		types.QualificationStatus_QUALIFICATION_STATUS_ACTIVE,
		types.QualificationStatus_QUALIFICATION_STATUS_PROBATIONARY,
		types.QualificationStatus_QUALIFICATION_STATUS_SUSPENDED,
	} {
		t.Run(status.String(), func(t *testing.T) {
			h := newLegacyRefundHarness(t)
			a := h.stake(t, "legacy-exit")
			q := h.qualification(t, a)
			q.Status = status
			h.k.SetQualification(h.ctx, q)
			require.NoError(t, h.k.EnableSingleRefund(h.ctx))
			h.ctx = h.ctx.WithBlockHeight(int64(q.GrantedAt + h.k.GetParams(h.ctx).StakeLockPeriod))
			require.NoError(t, h.withdraw(a))
			require.Equal(t, refundStake, h.balance(a))
			require.Equal(t, "0", h.escrow())
			require.Equal(t, 1, h.bank.refundCalls)
			require.ErrorIs(t, h.withdraw(a), types.ErrQualificationNotFound)
		})
	}
}
