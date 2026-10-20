package keeper_test

import (
	"context"
	"errors"
	"testing"

	"cosmossdk.io/log"
	"cosmossdk.io/store"
	storemetrics "cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/codec"
	addresscodec "github.com/cosmos/cosmos-sdk/codec/address"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/zerone-chain/zerone/x/qualification/keeper"
	"github.com/zerone-chain/zerone/x/qualification/types"
)

const refundStake = "100000000"

// Actual SDK bank/auth stores participate in CacheContext. Faults can happen
// before or AFTER the real transfer; a map-only bank cannot prove rollback.
type refundBank struct {
	bankkeeper.BaseKeeper
	failBefore, failAfter bool
	refundCalls           int
}

func (b *refundBank) SendCoinsFromModuleToAccount(ctx context.Context, module string, addr sdk.AccAddress, coins sdk.Coins) error {
	b.refundCalls++
	if b.failBefore {
		return errors.New("injected pre-transfer failure")
	}
	if err := b.BaseKeeper.SendCoinsFromModuleToAccount(ctx, module, addr, coins); err != nil {
		return err
	}
	if b.failAfter {
		return errors.New("injected post-transfer failure")
	}
	return nil
}

type refundHarness struct {
	k    keeper.Keeper
	ctx  sdk.Context
	bank *refundBank
	key  *storetypes.KVStoreKey
	auth authkeeper.AccountKeeper
}

func newRefundHarness(t *testing.T) *refundHarness {
	t.Helper()
	h := newLegacyRefundHarness(t)
	require.NoError(t, h.k.EnableSingleRefund(h.ctx))
	return h
}

func newLegacyRefundHarness(t *testing.T) *refundHarness {
	t.Helper()
	keys := storetypes.NewKVStoreKeys(types.StoreKey, authtypes.StoreKey, banktypes.StoreKey)
	db := dbm.NewMemDB()
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	state := store.NewCommitMultiStore(db, log.NewNopLogger(), storemetrics.NewNoOpMetrics())
	state.SetIAVLSyncPruning(true)
	for _, name := range []string{types.StoreKey, authtypes.StoreKey, banktypes.StoreKey} {
		state.MountStoreWithDB(keys[name], storetypes.StoreTypeIAVL, db)
	}
	require.NoError(t, state.LoadLatestVersion())
	ctx := sdk.NewContext(state, cmtproto.Header{Height: 100}, false, log.NewNopLogger())
	registry := codectypes.NewInterfaceRegistry()
	authtypes.RegisterInterfaces(registry)
	banktypes.RegisterInterfaces(registry)
	cdc := codec.NewProtoCodec(registry)
	authority := authtypes.NewModuleAddress("gov").String()
	perms := map[string][]string{types.ModuleName: nil, "refund_test_fund": {authtypes.Minter}}
	ak := authkeeper.NewAccountKeeper(cdc, runtime.NewKVStoreService(keys[authtypes.StoreKey]), authtypes.ProtoBaseAccount, perms, addresscodec.NewBech32Codec("zrn"), "zrn", authority)
	ak.InitGenesis(ctx, *authtypes.DefaultGenesisState())
	bk := bankkeeper.NewBaseKeeper(cdc, runtime.NewKVStoreService(keys[banktypes.StoreKey]), ak, map[string]bool{}, authority, log.NewNopLogger())
	bk.InitGenesis(ctx, banktypes.DefaultGenesisState())
	for _, name := range []string{types.ModuleName, "refund_test_fund"} {
		ak.GetModuleAccount(ctx, name)
	}
	bank := &refundBank{BaseKeeper: bk}
	k := keeper.NewKeeper(runtime.NewKVStoreService(keys[types.StoreKey]), cdc, authority, bank)
	genesis := types.DefaultGenesis()
	genesis.SingleRefundEnabled = false // Rehearse pre-carrier state explicitly.
	k.InitGenesis(ctx, genesis)
	return &refundHarness{k: k, ctx: ctx, bank: bank, key: keys[types.StoreKey], auth: ak}
}

func (h *refundHarness) stake(t *testing.T, name string) string {
	t.Helper()
	addr := testAddr(name)
	coins := sdk.NewCoins(sdk.NewInt64Coin("uzrn", 100_000_000))
	require.NoError(t, h.bank.MintCoins(h.ctx, "refund_test_fund", coins))
	require.NoError(t, h.bank.BaseKeeper.SendCoinsFromModuleToAccount(h.ctx, "refund_test_fund", sdk.MustAccAddressFromBech32(addr), coins))
	_, err := keeper.NewMsgServerImpl(h.k).QualifyByStake(h.ctx, &types.MsgQualifyByStake{Validator: addr, Domain: "physics", StakeAmount: refundStake})
	require.NoError(t, err)
	return addr
}

func (h *refundHarness) qualification(t *testing.T, addr string) *types.DomainQualification {
	t.Helper()
	q, found := h.k.GetQualification(h.ctx, addr, "physics")
	require.True(t, found)
	return q
}

func (h *refundHarness) balance(addr string) string {
	return h.bank.GetBalance(h.ctx, sdk.MustAccAddressFromBech32(addr), "uzrn").Amount.String()
}

func (h *refundHarness) escrow() string {
	return h.bank.GetBalance(h.ctx, authtypes.NewModuleAddress(types.ModuleName), "uzrn").Amount.String()
}

func (h *refundHarness) withdraw(addr string) error {
	// Mirror the SDK caller's transaction cache: a failed message is discarded.
	ctx, write := h.ctx.CacheContext()
	_, err := keeper.NewMsgServerImpl(h.k).WithdrawQualification(ctx, &types.MsgWithdrawQualification{Validator: addr, Domain: "physics"})
	if err == nil {
		write()
	}
	return err
}

func TestStakeRefundExpiryThenWithdraw(t *testing.T) {
	for _, neutral := range []bool{false, true} {
		for _, status := range []types.QualificationStatus{
			types.QualificationStatus_QUALIFICATION_STATUS_ACTIVE,
			types.QualificationStatus_QUALIFICATION_STATUS_PROBATIONARY,
		} {
			name := status.String()
			if neutral {
				name += "_neutral"
			}
			t.Run(name, func(t *testing.T) {
				h := newRefundHarness(t)
				h.k.SetReviewNeutralityPolicy(func(context.Context) (bool, error) { return neutral, nil })
				a := h.stake(t, "refund-a")
				q := h.qualification(t, a)
				q.Status = status
				h.k.SetQualification(h.ctx, q)
				h.ctx = h.ctx.WithBlockHeight(h.ctx.BlockHeight() + 1)
				b := h.stake(t, "refund-b")
				// This endorsement survives expiry and must be cleaned by withdrawal.
				_, err := keeper.NewMsgServerImpl(h.k).EndorseQualification(h.ctx, &types.MsgEndorseQualification{Endorser: testAddr("refund-endorser"), Validator: a, Domain: "physics", Weight: 1})
				require.NoError(t, err)
				h.ctx = h.ctx.WithBlockHeight(int64(q.ExpiresAt))
				require.NoError(t, h.k.BeginBlocker(h.ctx))
				t.Logf("after expiry: A=%s B=%s escrow=%s", h.balance(a), h.balance(b), h.escrow())
				require.Equal(t, refundStake, h.balance(a))
				require.Equal(t, refundStake, h.escrow())
				expired := h.qualification(t, a)
				require.Equal(t, types.QualificationStatus_QUALIFICATION_STATUS_EXPIRED, expired.Status)
				require.NoError(t, h.withdraw(a))
				t.Logf("after withdrawal: A=%s B=%s escrow=%s", h.balance(a), h.balance(b), h.escrow())
				require.Equal(t, refundStake, h.balance(a), "expiry followed by withdrawal must not pay twice")
				require.Empty(t, expired.StakedAmount)
				require.Equal(t, refundStake, h.escrow(), "B's escrow must remain backed")
				require.Equal(t, "0", h.balance(b))
				require.Equal(t, types.QualificationStatus_QUALIFICATION_STATUS_ACTIVE, h.qualification(t, b).Status)
				require.Equal(t, refundStake, h.qualification(t, b).StakedAmount)
				require.Empty(t, h.k.GetEndorsementsByTarget(h.ctx, a, "physics"))
				require.NotContains(t, h.k.GetValidatorsByDomain(h.ctx, "physics"), a)
				require.ErrorIs(t, h.withdraw(a), types.ErrQualificationNotFound)
				require.NoError(t, h.withdraw(b))
				require.Equal(t, refundStake, h.balance(b))
				require.Equal(t, "0", h.escrow())
				require.Equal(t, "200000000", h.bank.GetSupply(h.ctx, "uzrn").Amount.String())
				// Deleting the settled record still permits a new, separately funded stake.
				require.NoError(t, h.k.QualifyByStake(h.ctx, a, "physics", refundStake))
				require.Equal(t, refundStake, h.escrow())
				require.Equal(t, "0", h.balance(a))
			})
		}
	}
}

func TestStakeRefundWithdrawBeforeExpiry(t *testing.T) {
	h := newRefundHarness(t)
	a := h.stake(t, "early-a")
	q := h.qualification(t, a)
	h.ctx = h.ctx.WithBlockHeight(h.ctx.BlockHeight() + 1)
	b := h.stake(t, "early-b")
	unlock := q.GrantedAt + h.k.GetParams(h.ctx).StakeLockPeriod
	h.ctx = h.ctx.WithBlockHeight(int64(unlock - 1))
	require.ErrorIs(t, h.withdraw(a), types.ErrStakeLocked)
	require.Equal(t, "0", h.balance(a))
	require.Equal(t, "200000000", h.escrow())
	require.True(t, proto.Equal(q, h.qualification(t, a)))
	h.ctx = h.ctx.WithBlockHeight(int64(unlock))
	require.NoError(t, h.withdraw(a))
	require.ErrorIs(t, h.withdraw(a), types.ErrQualificationNotFound)
	h.ctx = h.ctx.WithBlockHeight(int64(q.ExpiresAt))
	require.NoError(t, h.k.BeginBlocker(h.ctx))
	require.Equal(t, refundStake, h.balance(a))
	require.Equal(t, refundStake, h.escrow())
	require.NoError(t, h.withdraw(b))
	require.Equal(t, refundStake, h.balance(b))
	require.Equal(t, "0", h.escrow())
}

func TestStakeRefundLegacyExpiredRefuses(t *testing.T) {
	for _, paid := range []bool{false, true} {
		name := "unpaid_or_locked"
		if paid {
			name = "already_paid"
		}
		t.Run(name, func(t *testing.T) {
			h := newRefundHarness(t)
			a := h.stake(t, "legacy-a")
			h.ctx = h.ctx.WithBlockHeight(h.ctx.BlockHeight() + 1)
			b := h.stake(t, "legacy-b")
			q := h.qualification(t, a)
			q.Status = types.QualificationStatus_QUALIFICATION_STATUS_EXPIRED
			h.k.SetQualification(h.ctx, q)
			if paid {
				// Reproduce the old successful refund while leaving StakedAmount intact.
				require.NoError(t, h.bank.BaseKeeper.SendCoinsFromModuleToAccount(h.ctx, types.ModuleName, sdk.MustAccAddressFromBech32(a), sdk.NewCoins(sdk.NewInt64Coin("uzrn", 100_000_000))))
			}
			h.ctx = h.ctx.WithBlockHeight(int64(q.ExpiresAt))
			beforeBalance, beforeEscrow := h.balance(a), h.escrow()
			require.NoError(t, h.k.BeginBlocker(h.ctx))
			err := h.withdraw(a)
			require.ErrorIs(t, err, types.ErrAlreadyExpired)
			require.ErrorContains(t, err, "reconciliation")
			require.Equal(t, beforeBalance, h.balance(a))
			require.Equal(t, beforeEscrow, h.escrow())
			require.True(t, proto.Equal(q, h.qualification(t, a)), "retain evidence, do not silently forfeit an unpaid claim")
			require.NoError(t, h.withdraw(b))
			require.Equal(t, refundStake, h.balance(b))
		})
	}
}

func TestStakeRefundFailurePendingWithdrawal(t *testing.T) {
	for _, after := range []bool{false, true} {
		name := "before_transfer"
		if after {
			name = "after_transfer"
		}
		t.Run(name, func(t *testing.T) {
			h := newRefundHarness(t)
			a := h.stake(t, "pending-a")
			q := h.qualification(t, a)
			h.ctx = h.ctx.WithBlockHeight(h.ctx.BlockHeight() + 1)
			b := h.stake(t, "pending-b")
			h.bank.failBefore, h.bank.failAfter = !after, after
			h.ctx = h.ctx.WithBlockHeight(int64(q.ExpiresAt)).WithEventManager(sdk.NewEventManager())
			require.NoError(t, h.k.BeginBlocker(h.ctx), "a refund error must not halt BeginBlock")
			require.Equal(t, "0", h.balance(a))
			require.Equal(t, "200000000", h.escrow(), "even a post-transfer failure must roll back")
			pending := h.qualification(t, a)
			require.Equal(t, types.QualificationStatus_QUALIFICATION_STATUS_SUSPENDED, pending.Status)
			require.Equal(t, refundStake, pending.StakedAmount)
			require.False(t, h.k.IsQualified(h.ctx, a, "physics"))
			require.Zero(t, h.k.GetQualificationWeight(h.ctx, a, "physics"))
			_, err := keeper.NewMsgServerImpl(h.k).RenewQualification(h.ctx, &types.MsgRenewQualification{Validator: a, Domain: "physics"})
			require.ErrorIs(t, err, types.ErrNotActive)
			var pendingEvents int
			for _, event := range h.ctx.EventManager().Events() {
				if event.Type == "zerone.qualification.stake_refund_pending" {
					pendingEvents++
				}
				require.NotEqual(t, banktypes.EventTypeTransfer, event.Type, "discard failed transfer events")
			}
			require.Equal(t, 1, pendingEvents)
			require.NoError(t, h.k.BeginBlocker(h.ctx))
			require.Equal(t, 1, h.bank.refundCalls, "no implicit retry loop")
			require.Error(t, h.withdraw(a))
			require.Equal(t, "0", h.balance(a))
			require.Equal(t, "200000000", h.escrow())
			h.bank.failBefore, h.bank.failAfter = false, false
			require.NoError(t, h.withdraw(a))
			require.ErrorIs(t, h.withdraw(a), types.ErrQualificationNotFound)
			require.Equal(t, refundStake, h.balance(a))
			require.Equal(t, refundStake, h.escrow())
			require.NoError(t, h.withdraw(b))
			require.Equal(t, refundStake, h.balance(b))
			require.Equal(t, "0", h.escrow())
		})
	}
}

func TestStakeRefundExpiryBeforeUnlock(t *testing.T) {
	h := newRefundHarness(t)
	params := h.k.GetParams(h.ctx)
	params.QualificationPeriod = 10
	params.StakeLockPeriod = 20
	h.k.SetParams(h.ctx, params)
	a := h.stake(t, "locked-expiry")
	q := h.qualification(t, a)
	h.ctx = h.ctx.WithBlockHeight(int64(q.ExpiresAt))
	require.NoError(t, h.k.BeginBlocker(h.ctx))
	require.Equal(t, types.QualificationStatus_QUALIFICATION_STATUS_SUSPENDED, h.qualification(t, a).Status)
	require.Equal(t, refundStake, h.qualification(t, a).StakedAmount)
	require.Equal(t, refundStake, h.escrow())
	require.Equal(t, 0, h.bank.refundCalls)
	require.ErrorIs(t, h.withdraw(a), types.ErrStakeLocked)
	h.ctx = h.ctx.WithBlockHeight(int64(q.GrantedAt + params.StakeLockPeriod))
	require.NoError(t, h.withdraw(a))
	require.Equal(t, refundStake, h.balance(a))
	require.Equal(t, "0", h.escrow())
}

func TestStakeRefundMissingBankPreservesClaim(t *testing.T) {
	h := newRefundHarness(t)
	a := h.stake(t, "missing-bank")
	q := h.qualification(t, a)
	// Remove only the qualification adapter, leaving the real funded bank intact.
	h.k = keeper.NewKeeper(runtime.NewKVStoreService(h.key), codec.NewProtoCodec(codectypes.NewInterfaceRegistry()), h.k.GetAuthority(), nil)
	h.ctx = h.ctx.WithBlockHeight(int64(q.ExpiresAt))
	require.NoError(t, h.k.BeginBlocker(h.ctx))
	require.Equal(t, types.QualificationStatus_QUALIFICATION_STATUS_SUSPENDED, h.qualification(t, a).Status)
	require.Equal(t, refundStake, h.qualification(t, a).StakedAmount)
	require.ErrorContains(t, h.withdraw(a), "bank keeper is not configured")
	require.Equal(t, refundStake, h.escrow())
	require.Equal(t, "0", h.balance(a))
}

func TestStakeRefundGenesisPreservesSettlementState(t *testing.T) {
	for _, mode := range []string{"settled", "pending", "legacy_unknown"} {
		t.Run(mode, func(t *testing.T) {
			h := newRefundHarness(t)
			a := h.stake(t, "export-a")
			q := h.qualification(t, a)
			h.ctx = h.ctx.WithBlockHeight(h.ctx.BlockHeight() + 1)
			b := h.stake(t, "export-b")
			h.ctx = h.ctx.WithBlockHeight(int64(q.ExpiresAt))
			switch mode {
			case "pending":
				h.bank.failBefore = true
			case "legacy_unknown":
				q.Status = types.QualificationStatus_QUALIFICATION_STATUS_EXPIRED
				h.k.SetQualification(h.ctx, q)
			}
			require.NoError(t, h.k.BeginBlocker(h.ctx))
			// The foundation's genesis field preserves the selector together with
			// the records and real bank/auth state; no manual re-enabling on import.
			genesis := h.k.ExportGenesis(h.ctx)
			require.True(t, genesis.SingleRefundEnabled)
			encoded, err := proto.Marshal(genesis)
			require.NoError(t, err)
			decoded := new(types.GenesisState)
			require.NoError(t, proto.Unmarshal(encoded, decoded))
			restored := newLegacyRefundHarness(t)
			restored.ctx = restored.ctx.WithBlockHeight(h.ctx.BlockHeight())
			restored.auth.InitGenesis(restored.ctx, *h.auth.ExportGenesis(h.ctx))
			restored.bank.InitGenesis(restored.ctx, h.bank.ExportGenesis(h.ctx))
			restored.k.InitGenesis(restored.ctx, decoded)
			enabled, err := restored.k.SingleRefundEnabled(restored.ctx)
			require.NoError(t, err)
			require.True(t, enabled, "logical genesis preserves the activation marker")
			require.True(t, proto.Equal(genesis, restored.k.ExportGenesis(restored.ctx)))
			require.Equal(t, h.escrow(), restored.escrow())
			require.Equal(t, h.balance(a), restored.balance(a))
			if mode == "legacy_unknown" {
				require.ErrorIs(t, restored.withdraw(a), types.ErrAlreadyExpired)
				require.Equal(t, "200000000", restored.escrow())
			} else {
				require.NoError(t, restored.withdraw(a))
				require.Equal(t, refundStake, restored.balance(a))
				require.Equal(t, refundStake, restored.escrow())
			}
			require.NoError(t, restored.withdraw(b))
			require.Equal(t, refundStake, restored.balance(b))
			require.Equal(t, "200000000", restored.bank.GetSupply(restored.ctx, "uzrn").Amount.String())
		})
	}
}
