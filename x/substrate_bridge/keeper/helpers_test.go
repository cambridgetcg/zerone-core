package keeper_test

import (
	"context"
	"testing"

	dbm "github.com/cosmos/cosmos-db"

	"cosmossdk.io/log"
	"cosmossdk.io/store"
	storemetrics "cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"

	sdkmath "cosmossdk.io/math"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/x/substrate_bridge/keeper"
	"github.com/zerone-chain/zerone/x/substrate_bridge/types"
)

func setupSubstrateBridgeKeeper(t *testing.T) (keeper.Keeper, sdk.Context) {
	t.Helper()

	storeKey := storetypes.NewKVStoreKey(types.StoreKey)
	db := dbm.NewMemDB()
	cms := store.NewCommitMultiStore(db, log.NewNopLogger(), storemetrics.NewNoOpMetrics())
	cms.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	if err := cms.LoadLatestVersion(); err != nil {
		t.Fatalf("load store: %v", err)
	}

	registry := codectypes.NewInterfaceRegistry()
	types.RegisterInterfaces(registry)
	cdc := codec.NewProtoCodec(registry)

	k := keeper.NewKeeper(cdc, storeKey, "authority-addr", nil, nil, nil, nil)

	ctx := sdk.NewContext(cms, cmtproto.Header{Height: 1}, false, log.NewNopLogger())

	if err := k.SetParams(ctx, types.DefaultParams()); err != nil {
		t.Fatalf("set params: %v", err)
	}

	return k, ctx
}

// stubBankKeeper records every SendCoinsFromModuleToAccount call so
// tests can assert payment recipients and amounts.
type stubBankKeeper struct {
	payments map[string]sdkmath.Int // recipient_addr → cumulative paid
}

func (s *stubBankKeeper) SendCoinsFromAccountToModule(ctx context.Context, from sdk.AccAddress, mod string, coins sdk.Coins) error {
	return nil
}

func (s *stubBankKeeper) SendCoinsFromModuleToAccount(ctx context.Context, mod string, to sdk.AccAddress, coins sdk.Coins) error {
	if s.payments == nil {
		s.payments = map[string]sdkmath.Int{}
	}
	cur := s.payments[to.String()]
	for _, c := range coins {
		cur = cur.Add(c.Amount)
	}
	s.payments[to.String()] = cur
	return nil
}

func (s *stubBankKeeper) BurnCoins(ctx context.Context, mod string, coins sdk.Coins) error {
	return nil
}

// setupSubstrateBridgeKeeperWithBank is a variant of setupSubstrateBridgeKeeper
// that wires a stubBankKeeper to record payments.
func setupSubstrateBridgeKeeperWithBank(t *testing.T) (keeper.Keeper, sdk.Context) {
	t.Helper()

	storeKey := storetypes.NewKVStoreKey(types.StoreKey)
	db := dbm.NewMemDB()
	cms := store.NewCommitMultiStore(db, log.NewNopLogger(), storemetrics.NewNoOpMetrics())
	cms.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	if err := cms.LoadLatestVersion(); err != nil {
		t.Fatalf("load store: %v", err)
	}

	registry := codectypes.NewInterfaceRegistry()
	types.RegisterInterfaces(registry)
	cdc := codec.NewProtoCodec(registry)

	bk := &stubBankKeeper{}
	k := keeper.NewKeeper(cdc, storeKey, "authority-addr", nil, nil, bk, nil)

	ctx := sdk.NewContext(cms, cmtproto.Header{Height: 1}, false, log.NewNopLogger())

	if err := k.SetParams(ctx, types.DefaultParams()); err != nil {
		t.Fatalf("set params: %v", err)
	}

	return k, ctx
}

// setupTwoAttestations writes "att-upstream" (block 10) and "att-downstream"
// (block 20) into the keeper for tests that need a pre-populated pair.
func setupTwoAttestations(t *testing.T, k keeper.Keeper, ctx sdk.Context) {
	t.Helper()
	require := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatalf("setupTwoAttestations: %v", err)
		}
	}
	require(k.WriteAttestation(ctx, &types.ExternalAttestation{
		AttestationId: "att-upstream", Submitter: "alice", SubmittedAtBlock: 10,
		Status: types.AttestationStatus_ATTESTATION_STATUS_SETTLED,
	}))
	require(k.WriteAttestation(ctx, &types.ExternalAttestation{
		AttestationId: "att-downstream", Submitter: "bob", SubmittedAtBlock: 20,
		Status: types.AttestationStatus_ATTESTATION_STATUS_SETTLED,
	}))
}
