package cross_stack_test

import (
	"context"
	"errors"
	"testing"

	"cosmossdk.io/core/header"
	corestore "cosmossdk.io/core/store"
	"github.com/cosmos/cosmos-sdk/types/module"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	zeroneapp "github.com/zerone-chain/zerone/app"
	claimingpottypes "github.com/zerone-chain/zerone/x/claiming_pot/types"
	knowledgekeeper "github.com/zerone-chain/zerone/x/knowledge/keeper"
	knowledgetypes "github.com/zerone-chain/zerone/x/knowledge/types"
	liquiditypooltypes "github.com/zerone-chain/zerone/x/liquiditypool/types"
	vestingrewardstypes "github.com/zerone-chain/zerone/x/vesting_rewards/types"
)

const h1MigrationMarker = "upgrade_marker_consolidation-safety-v1"

func h1Prestate(t *testing.T, h *TestHarness) module.VersionMap {
	t.Helper()
	target := h.App.CurrentModuleVersionMap()
	require.Equal(t, uint64(6), target[knowledgetypes.ModuleName])
	require.Equal(t, uint64(2), target[claimingpottypes.ModuleName])
	require.Equal(t, uint64(5), target[liquiditypooltypes.ModuleName])
	require.Equal(t, uint64(1), target[vestingrewardstypes.ModuleName],
		"H1 source must not contain the H2 vesting transition")

	fromVM := make(module.VersionMap, len(target))
	for name, version := range target {
		fromVM[name] = version
	}
	fromVM[knowledgetypes.ModuleName] = 5
	fromVM[claimingpottypes.ModuleName] = 1
	fromVM[liquiditypooltypes.ModuleName] = 3
	fromVM[vestingrewardstypes.ModuleName] = 1
	return fromVM
}

func TestH1NamedUpgradeProvesExactBoundaryAndDoneHeight(t *testing.T) {
	h := NewTestHarness(t)
	h.Ctx = h.Ctx.WithHeaderInfo(header.Info{Height: h.Height(), ChainID: testChainID})
	fromVM := h1Prestate(t, h)
	target := h.App.CurrentModuleVersionMap()

	legacyLiquidity := h.App.LiquidityPoolKeeper.GetParams(h.Ctx)
	legacyLiquidity.ProtocolFeeBps = 450_000
	legacyLiquidity.MaxPools = 0
	legacyLiquidity.AllowedPoolDenoms = nil
	legacyLiquidity.PoolCreators = nil
	h.App.LiquidityPoolKeeper.SetParams(h.Ctx, legacyLiquidity)
	vestingBefore := proto.Clone(h.VestingRewardsKeeper.GetParams(h.Ctx)).(*vestingrewardstypes.Params)

	toVM, err := h.App.RunUpgradeHandlerForTests(
		h.Ctx,
		zeroneapp.UpgradeNameConsolidationSafetyV1,
		fromVM,
		h.Height(),
	)
	require.NoError(t, err)
	require.Equal(t, target, toVM, "H1 must produce the binary's exact module map")
	require.Equal(t, uint64(6), toVM[knowledgetypes.ModuleName])
	require.Equal(t, uint64(2), toVM[claimingpottypes.ModuleName])
	require.Equal(t, uint64(5), toVM[liquiditypooltypes.ModuleName])
	require.Equal(t, uint64(1), toVM[vestingrewardstypes.ModuleName])

	marker, found, err := h.KnowledgeKeeper.ReadMigrationMarkerPresenceChecked(h.Ctx, h1MigrationMarker)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, "migrated", marker)
	require.Equal(t, "true", h.KnowledgeKeeper.ReadMigrationMarker(h.Ctx, "migration_v6_complete"))
	doneHeight, err := h.App.UpgradeKeeper.GetDoneHeight(h.Ctx, zeroneapp.UpgradeNameConsolidationSafetyV1)
	require.NoError(t, err)
	require.Equal(t, h.Height(), doneHeight,
		"H1 proof includes x/upgrade's completed height")

	migratedLiquidity := h.App.LiquidityPoolKeeper.GetParams(h.Ctx)
	require.Zero(t, migratedLiquidity.ProtocolFeeBps)
	require.Equal(t, liquiditypooltypes.DefaultParams().MaxPools, migratedLiquidity.MaxPools)
	require.Empty(t, migratedLiquidity.AllowedPoolDenoms)
	require.Empty(t, migratedLiquidity.PoolCreators)
	require.True(t, proto.Equal(vestingBefore, h.VestingRewardsKeeper.GetParams(h.Ctx)),
		"H1 must leave vesting_rewards v1 params byte-semantically unchanged")
}

func TestH1RetainsDeterministicModulePermissionReconcile(t *testing.T) {
	h := NewTestHarness(t)
	moduleAccount := h.AccountKeeper.GetModuleAccount(h.Ctx, liquiditypooltypes.ModuleName)
	require.NotNil(t, moduleAccount)

	// Reproduce historical x/auth permission drift without changing balances.
	h.AccountKeeper.SetModuleAccount(h.Ctx, authtypes.NewModuleAccount(
		authtypes.NewBaseAccount(
			moduleAccount.GetAddress(),
			nil,
			moduleAccount.GetAccountNumber(),
			moduleAccount.GetSequence(),
		),
		liquiditypooltypes.ModuleName,
	))
	require.Empty(t,
		h.AccountKeeper.GetModuleAccount(h.Ctx, liquiditypooltypes.ModuleName).GetPermissions(),
		"precondition: stored permissions drifted from maccPerms",
	)

	toVM, err := h.App.RunUpgradeHandlerForTests(
		h.Ctx,
		zeroneapp.UpgradeNameConsolidationSafetyV1,
		h1Prestate(t, h),
		h.Height(),
	)
	require.NoError(t, err)
	require.Equal(t, uint64(1), toVM[vestingrewardstypes.ModuleName])
	require.ElementsMatch(t,
		[]string{authtypes.Minter, authtypes.Burner},
		h.AccountKeeper.GetModuleAccount(h.Ctx, liquiditypooltypes.ModuleName).GetPermissions(),
		"H1 retains the reviewed deterministic x/auth permission repair",
	)
}

func TestH1RejectsEveryInexactVersionMapBeforeMigration(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(module.VersionMap)
		message string
	}{
		{
			name: "missing knowledge",
			mutate: func(vm module.VersionMap) {
				delete(vm, knowledgetypes.ModuleName)
			},
			message: "requires exact prestate knowledge=5",
		},
		{
			name: "knowledge already target",
			mutate: func(vm module.VersionMap) {
				vm[knowledgetypes.ModuleName] = 6
			},
			message: "requires exact prestate knowledge=5",
		},
		{
			name: "claiming pot already target",
			mutate: func(vm module.VersionMap) {
				vm[claimingpottypes.ModuleName] = 2
			},
			message: "requires exact prestate claiming_pot=1",
		},
		{
			name: "liquidity intermediate v4",
			mutate: func(vm module.VersionMap) {
				vm[liquiditypooltypes.ModuleName] = 4
			},
			message: "requires exact prestate liquiditypool=3",
		},
		{
			name: "vesting moved by another binary",
			mutate: func(vm module.VersionMap) {
				vm[vestingrewardstypes.ModuleName] = 2
			},
			message: "requires exact prestate vesting_rewards=1",
		},
		{
			name: "unrelated module catchup",
			mutate: func(vm module.VersionMap) {
				vm["bank"]--
			},
			message: "refuses unrelated migration for module \"bank\"",
		},
		{
			name: "unknown module entry",
			mutate: func(vm module.VersionMap) {
				vm["unknown-h1-sidecar"] = 1
			},
			message: "refuses unknown module version entry \"unknown-h1-sidecar\"",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := NewTestHarness(t)
			fromVM := h1Prestate(t, h)
			tc.mutate(fromVM)
			_, err := h.App.RunMigrationsForPlanForTests(
				h.Ctx,
				zeroneapp.UpgradeNameConsolidationSafetyV1,
				fromVM,
				h.Height(),
			)
			require.ErrorContains(t, err, tc.message)
			_, found, markerErr := h.KnowledgeKeeper.ReadMigrationMarkerPresenceChecked(h.Ctx, h1MigrationMarker)
			require.NoError(t, markerErr)
			require.False(t, found)
		})
	}
}

func TestH1RejectsPreseededMarkerBeforeAnyMigration(t *testing.T) {
	for _, markerValue := range []string{"forged", "migrated"} {
		t.Run(markerValue, func(t *testing.T) {
			h := NewTestHarness(t)
			require.NoError(t, h.KnowledgeKeeper.WriteMigrationMarker(h.Ctx, h1MigrationMarker, markerValue))
			fromVM := h1Prestate(t, h)

			_, err := h.App.RunUpgradeHandlerForTests(
				h.Ctx,
				zeroneapp.UpgradeNameConsolidationSafetyV1,
				fromVM,
				h.Height(),
			)
			require.ErrorContains(t, err, "requires migration marker")
			require.ErrorContains(t, err, "to be absent before execution")
			marker, found, readErr := h.KnowledgeKeeper.ReadMigrationMarkerPresenceChecked(h.Ctx, h1MigrationMarker)
			require.NoError(t, readErr)
			require.True(t, found)
			require.Equal(t, markerValue, marker)
			storedVM, vmErr := h.App.UpgradeKeeper.GetModuleVersionMap(h.Ctx)
			require.NoError(t, vmErr)
			require.Equal(t, uint64(5), storedVM[knowledgetypes.ModuleName])
			require.Equal(t, uint64(1), storedVM[claimingpottypes.ModuleName])
			require.Equal(t, uint64(3), storedVM[liquiditypooltypes.ModuleName])
			require.Equal(t, uint64(1), storedVM[vestingrewardstypes.ModuleName])
			doneHeight, doneErr := h.App.UpgradeKeeper.GetDoneHeight(h.Ctx, zeroneapp.UpgradeNameConsolidationSafetyV1)
			require.NoError(t, doneErr)
			require.Zero(t, doneHeight)
		})
	}
}

func TestH1RejectsHistoricalEmptyOrUnreadableMarker(t *testing.T) {
	tests := []struct {
		name    string
		store   corestore.KVStore
		message string
	}{
		{
			name:    "historical empty value",
			store:   h1MarkerStore{getValue: []byte{}},
			message: "requires migration marker",
		},
		{
			name:    "read error",
			store:   h1MarkerStore{getErr: errors.New("marker read failed")},
			message: "cannot verify migration marker absence",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := NewTestHarness(t)
			h.App.KnowledgeKeeper = knowledgekeeper.NewKeeper(
				h1MarkerStoreService{store: tc.store}, nil, "", nil, nil,
			)
			fromVM := h1Prestate(t, h)

			_, err := h.App.RunUpgradeHandlerForTests(
				h.Ctx,
				zeroneapp.UpgradeNameConsolidationSafetyV1,
				fromVM,
				h.Height(),
			)
			require.ErrorContains(t, err, tc.message)
			storedVM, vmErr := h.App.UpgradeKeeper.GetModuleVersionMap(h.Ctx)
			require.NoError(t, vmErr)
			require.Equal(t, uint64(5), storedVM[knowledgetypes.ModuleName])
			require.Equal(t, uint64(1), storedVM[claimingpottypes.ModuleName])
			require.Equal(t, uint64(3), storedVM[liquiditypooltypes.ModuleName])
			require.Equal(t, uint64(1), storedVM[vestingrewardstypes.ModuleName])
			doneHeight, doneErr := h.App.UpgradeKeeper.GetDoneHeight(h.Ctx, zeroneapp.UpgradeNameConsolidationSafetyV1)
			require.NoError(t, doneErr)
			require.Zero(t, doneHeight)
		})
	}
}

func TestH1CannotRideUnrelatedPlanAndUnrelatedNoopWorksAfterH1(t *testing.T) {
	h := NewTestHarness(t)
	fromVM := h1Prestate(t, h)
	_, err := h.App.RunMigrationsForPlanForTests(
		h.Ctx,
		zeroneapp.UpgradeNameTestnetV2,
		fromVM,
		h.Height(),
	)
	require.ErrorContains(t, err, "cannot carry the \"consolidation-safety-v1\" bundle")

	target := h.App.CurrentModuleVersionMap()
	toVM, err := h.App.RunMigrationsForPlanForTests(
		h.Ctx,
		zeroneapp.UpgradeNameTestnetV2,
		target,
		h.Height(),
	)
	require.NoError(t, err, "an unrelated no-op remains valid after the H1 boundary")
	require.Equal(t, target, toVM)
}

type h1MarkerStoreService struct {
	store corestore.KVStore
}

func (s h1MarkerStoreService) OpenKVStore(context.Context) corestore.KVStore {
	return s.store
}

type h1MarkerStore struct {
	getValue []byte
	getErr   error
}

func (s h1MarkerStore) Get([]byte) ([]byte, error) {
	return s.getValue, s.getErr
}

func (s h1MarkerStore) Has([]byte) (bool, error) {
	return s.getValue != nil, s.getErr
}

func (s h1MarkerStore) Set([]byte, []byte) error {
	return errors.New("H1 marker test store is read-only")
}

func (s h1MarkerStore) Delete([]byte) error {
	return errors.New("H1 marker test store is read-only")
}

func (s h1MarkerStore) Iterator([]byte, []byte) (corestore.Iterator, error) {
	return nil, errors.New("H1 marker test store does not support iteration")
}

func (s h1MarkerStore) ReverseIterator([]byte, []byte) (corestore.Iterator, error) {
	return nil, errors.New("H1 marker test store does not support reverse iteration")
}
