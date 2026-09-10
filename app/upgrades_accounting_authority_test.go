package app

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"cosmossdk.io/core/appmodule"
	"cosmossdk.io/core/header"
	"cosmossdk.io/log"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	abci "github.com/cometbft/cometbft/abci/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/baseapp"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/require"

	"github.com/zerone-chain/zerone/internal/accountingmigration"
	govtypes "github.com/zerone-chain/zerone/x/gov/types"
	stakingtypes "github.com/zerone-chain/zerone/x/staking/types"
)

func newAccountingAuthorityFixture(t *testing.T) (*ZeroneApp, sdk.Context, dbm.DB) {
	t.Helper()
	db := dbm.NewMemDB()
	app := NewZeroneApp(log.NewNopLogger(), db, nil, true, simtestutil.NewAppOptionsWithFlagHome(t.TempDir()), baseapp.SetChainID("accounting-fixture"))
	genesis, err := json.Marshal(sdk053IBC10GenesisWithValidator(t, app))
	require.NoError(t, err)
	ctx := app.NewUncachedContext(false, cmtproto.Header{ChainID: "accounting-fixture"})
	_, err = app.InitChainer(ctx, &abci.RequestInitChain{ChainId: "accounting-fixture", AppStateBytes: genesis})
	require.NoError(t, err)
	return app, ctx, db
}

func accountingLegacyFixture(t *testing.T, app *ZeroneApp, ctx sdk.Context) module.VersionMap {
	t.Helper()
	// Exercise the original accounting binary target, without appending the
	// later survival handoff delta to its frozen named release.
	for name, version := range map[string]uint64{"knowledge": 6, "vesting_rewards": 2} {
		app.ModuleManager.Modules[name] = archivedH3AccountingModule{app.ModuleManager.Modules[name].(appmodule.AppModule), version}
	}
	ctx.KVStore(app.keys[stakingtypes.StoreKey]).Delete(stakingtypes.AccountingSafetyKey)
	ctx.KVStore(app.keys[govtypes.StoreKey]).Delete(govtypes.AccountingSafetyKey)
	ctx.KVStore(app.keys["knowledge"]).Delete(append([]byte{0x7f, 0x01}, []byte(accountingAuthorityNativeMarker)...))
	vm := accountingAuthoritySourceVersionMap()
	require.NoError(t, app.UpgradeKeeper.SetModuleVersionMap(ctx, vm))
	return vm
}

func TestAccountingAuthorityNativeGenesisAndRestart(t *testing.T) {
	app, ctx, db := newAccountingAuthorityFixture(t)
	require.True(t, app.ZeroneStakingKeeper.AccountingSafetyEnabled(ctx))
	require.True(t, app.ZeroneGovKeeper.AccountingSafetyEnabled(ctx))
	require.True(t, app.UpgradeKeeper.HasHandler(UpgradeNameAccountingAuthorityV1))
	require.Contains(t, app.KnownUpgradeNames(), UpgradeNameAccountingAuthorityV1)
	require.Equal(t, uint64(2), app.CurrentModuleVersionMap()[stakingtypes.ModuleName])
	require.Equal(t, uint64(3), app.CurrentModuleVersionMap()[govtypes.ModuleName])
	app.CommitMultiStore().Commit()
	require.NoError(t, app.ValidateAccountingAuthorityStartup())
	// Exercise the real loadLatest constructor, including the initially empty DB
	// case above. The legacy startup guard must admit current native genesis.
	restarted := NewZeroneApp(log.NewNopLogger(), db, nil, true, simtestutil.NewAppOptionsWithFlagHome(t.TempDir()))
	require.NoError(t, restarted.ValidateAccountingAuthorityStartup())
}

func TestAccountingAuthorityGenesisRequiresBothExplicitSelections(t *testing.T) {
	for _, name := range []string{stakingtypes.ModuleName, govtypes.ModuleName} {
		t.Run(name, func(t *testing.T) {
			app := NewZeroneApp(log.NewNopLogger(), dbm.NewMemDB(), nil, true, simtestutil.NewAppOptionsWithFlagHome(t.TempDir()))
			genesis := sdk053IBC10GenesisWithValidator(t, app)
			var selection map[string]json.RawMessage
			require.NoError(t, json.Unmarshal(genesis[name], &selection))
			delete(selection, "accounting_safety_enabled")
			var err error
			genesis[name], err = json.Marshal(selection)
			require.NoError(t, err)
			encoded, err := json.Marshal(genesis)
			require.NoError(t, err)
			ctx := app.NewUncachedContext(false, cmtproto.Header{})
			_, err = app.InitChainer(ctx, &abci.RequestInitChain{AppStateBytes: encoded})
			require.ErrorContains(t, err, "requires explicit accounting_safety_enabled=true")
			require.False(t, app.ZeroneStakingKeeper.AccountingSafetyEnabled(ctx))
			require.False(t, app.ZeroneGovKeeper.AccountingSafetyEnabled(ctx))
		})
	}
}

func TestAccountingAuthorityStartupRefusesLegacyMixedAndForgedState(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*ZeroneApp, sdk.Context)
	}{
		{"legacy", func(app *ZeroneApp, ctx sdk.Context) { accountingLegacyFixture(t, app, ctx) }},
		{"mixed", func(app *ZeroneApp, ctx sdk.Context) {
			vm := app.CurrentModuleVersionMap()
			vm[stakingtypes.ModuleName] = 1
			require.NoError(t, app.UpgradeKeeper.SetModuleVersionMap(ctx, vm))
		}},
		{"unknown", func(app *ZeroneApp, ctx sdk.Context) {
			vm := app.CurrentModuleVersionMap()
			vm[govtypes.ModuleName] = 4
			require.NoError(t, app.UpgradeKeeper.SetModuleVersionMap(ctx, vm))
		}},
		{"missing version", func(app *ZeroneApp, ctx sdk.Context) {
			ctx.KVStore(app.keys[upgradetypes.StoreKey]).Delete(append([]byte{upgradetypes.VersionMapByte}, []byte(govtypes.ModuleName)...))
		}},
		{"missing staking marker", func(app *ZeroneApp, ctx sdk.Context) {
			ctx.KVStore(app.keys[stakingtypes.StoreKey]).Delete(stakingtypes.AccountingSafetyKey)
		}},
		{"corrupt gov marker", func(app *ZeroneApp, ctx sdk.Context) {
			ctx.KVStore(app.keys[govtypes.StoreKey]).Set(govtypes.AccountingSafetyKey, []byte{2})
		}},
		{"missing native lineage", func(app *ZeroneApp, ctx sdk.Context) {
			ctx.KVStore(app.keys["knowledge"]).Delete(append([]byte{0x7f, 0x01}, []byte(accountingAuthorityNativeMarker)...))
		}},
		{"unapproved applied upgrade", func(app *ZeroneApp, ctx sdk.Context) {
			writeSDK053IBC10TestDoneHeight(app, ctx, UpgradeNameAccountingAuthorityV1, 1)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			app, ctx, _ := newAccountingAuthorityFixture(t)
			tc.change(app, ctx)
			app.CommitMultiStore().Commit()
			require.Error(t, app.ValidateAccountingAuthorityStartup())
		})
	}
}

func TestAccountingAuthorityOwnsEveryMigrationDelta(t *testing.T) {
	app, _, _ := newAccountingAuthorityFixture(t)
	target := app.CurrentModuleVersionMap()
	legacy := app.CurrentModuleVersionMap()
	legacy[stakingtypes.ModuleName], legacy[govtypes.ModuleName] = 1, 2
	for _, name := range append(app.KnownUpgradeNames(), "unrelated-plan") {
		if name == UpgradeNameAccountingAuthorityV1 {
			continue
		}
		require.ErrorContains(t, requireAccountingTransitionOwner(name, legacy, target), "sole owner")
	}
	require.NoError(t, requireAccountingTransitionOwner(UpgradeNameAccountingAuthorityV1, legacy, target))
	require.NoError(t, requireAccountingTransitionOwner("unrelated-plan", target, target))
	for _, pair := range [][2]uint64{{0, 0}, {1, 3}, {2, 2}, {3, 3}, {2, 4}} {
		invalid := module.VersionMap{stakingtypes.ModuleName: pair[0], govtypes.ModuleName: pair[1]}
		require.Error(t, requireAccountingTransitionOwner(UpgradeNameAccountingAuthorityV1, invalid, target))
		require.Error(t, requireAccountingTransitionOwner(UpgradeNameAccountingAuthorityV1, legacy, invalid))
	}
	for _, name := range []string{stakingtypes.ModuleName, govtypes.ModuleName} {
		invalid := module.VersionMap{stakingtypes.ModuleName: 1, govtypes.ModuleName: 2}
		delete(invalid, name)
		require.Error(t, requireAccountingTransitionOwner(UpgradeNameAccountingAuthorityV1, invalid, target))
	}
}

func TestAccountingAuthorityModuleMigrationsRequireScopedOwnerAndPreserveClaims(t *testing.T) {
	app, ctx, _ := newAccountingAuthorityFixture(t)
	fromVM := accountingLegacyFixture(t, app, ctx)
	// Nonzero legacy custody and a real claimant ensure migration is more than
	// an empty-store marker test. This mint is synthetic fixture funding only.
	coins := sdk.NewCoins(sdk.NewInt64Coin("uzrn", 111000))
	require.NoError(t, app.BankKeeper.MintCoins(ctx, "tokens", coins))
	require.NoError(t, app.BankKeeper.SendCoinsFromModuleToModule(ctx, "tokens", stakingtypes.ModuleName, coins))
	address := sdk.AccAddress(bytes.Repeat([]byte{7}, 20)).String()
	app.ZeroneStakingKeeper.SetValidator(ctx, &stakingtypes.Validator{
		OperatorAddress: address, Did: "did:zrn:fixture", SelfDelegation: "111000", DelegatedStake: "0", TotalStake: "111000",
	})
	app.ZeroneStakingKeeper.SetDelegation(ctx, &stakingtypes.Delegation{DelegatorAddress: address, ValidatorAddress: address, Amount: "111000"})
	beforeClaim := bytes.Clone(ctx.KVStore(app.keys[stakingtypes.StoreKey]).Get(stakingtypes.DelegationKey(address, address)))
	beforeSupply := app.BankKeeper.GetSupply(ctx, "uzrn")
	_, err := app.ModuleManager.RunMigrations(ctx, app.configurator, fromVM)
	require.ErrorContains(t, err, "explicit")
	require.False(t, app.ZeroneStakingKeeper.AccountingSafetyEnabled(ctx))
	require.False(t, app.ZeroneGovKeeper.AccountingSafetyEnabled(ctx))
	_, err = accountingmigration.WithOwner(ctx, UpgradeNameSDK053IBC10)
	require.Error(t, err)

	cache, write := ctx.CacheContext()
	authorized, err := accountingmigration.WithOwner(cache, UpgradeNameAccountingAuthorityV1)
	require.NoError(t, err)
	toVM, err := app.ModuleManager.RunMigrations(authorized, app.configurator, fromVM)
	require.NoError(t, err)
	require.Equal(t, app.CurrentModuleVersionMap(), toVM)
	require.NoError(t, app.UpgradeKeeper.SetModuleVersionMap(authorized, toVM))
	write()
	require.True(t, app.ZeroneStakingKeeper.AccountingSafetyEnabled(ctx))
	require.True(t, app.ZeroneGovKeeper.AccountingSafetyEnabled(ctx))
	require.Equal(t, beforeClaim, ctx.KVStore(app.keys[stakingtypes.StoreKey]).Get(stakingtypes.DelegationKey(address, address)))
	require.Equal(t, beforeSupply, app.BankKeeper.GetSupply(ctx, "uzrn"))
	require.NoError(t, app.ZeroneStakingKeeper.ValidateAccountingSafety(ctx))
}

func TestAccountingAuthorityMigrationFailureDoesNotActivateEitherModule(t *testing.T) {
	app, ctx, _ := newAccountingAuthorityFixture(t)
	fromVM := accountingLegacyFixture(t, app, ctx)
	// A real bank surplus has no inferred owner and must refuse migration.
	coins := sdk.NewCoins(sdk.NewInt64Coin("uzrn", 1))
	require.NoError(t, app.BankKeeper.MintCoins(ctx, "tokens", coins))
	require.NoError(t, app.BankKeeper.SendCoinsFromModuleToModule(ctx, "tokens", stakingtypes.ModuleName, coins))
	cache, _ := ctx.CacheContext()
	authorized, err := accountingmigration.WithOwner(cache, UpgradeNameAccountingAuthorityV1)
	require.NoError(t, err)
	_, err = app.ModuleManager.RunMigrations(authorized, app.configurator, fromVM)
	require.ErrorContains(t, err, "bank balance")
	require.False(t, app.ZeroneStakingKeeper.AccountingSafetyEnabled(ctx))
	require.False(t, app.ZeroneGovKeeper.AccountingSafetyEnabled(ctx))
	vm, err := app.UpgradeKeeper.GetModuleVersionMap(ctx)
	require.NoError(t, err)
	require.Equal(t, fromVM, vm)
}

// The synthetic fixture proves both admitted predecessor lineage classes; the
// separate two-binary rehearsal supplies real native predecessor execution.
func committedAccountingSource(t *testing.T, migratedH3 bool) (*ZeroneApp, sdk.Context, module.VersionMap) {
	t.Helper()
	app, ctx, _ := newAccountingAuthorityFixture(t)
	fromVM := accountingLegacyFixture(t, app, ctx)
	self := sdk.AccAddress(bytes.Repeat([]byte{7}, 20)).String()
	other := sdk.AccAddress(bytes.Repeat([]byte{8}, 20)).String()
	app.ZeroneStakingKeeper.SetValidator(ctx, &stakingtypes.Validator{OperatorAddress: self, Did: "did:zrn:fixture", SelfDelegation: "100", DelegatedStake: "50", TotalStake: "150"})
	app.ZeroneStakingKeeper.SetDelegation(ctx, &stakingtypes.Delegation{DelegatorAddress: self, ValidatorAddress: self, Amount: "100"})
	app.ZeroneStakingKeeper.SetDelegation(ctx, &stakingtypes.Delegation{DelegatorAddress: other, ValidatorAddress: self, Amount: "50"})
	app.ZeroneGovKeeper.SetLIP(ctx, &govtypes.LIP{Id: "terminal-history", Category: govtypes.CategoryUpgrade, Stage: govtypes.StatusFailed, StakedAmount: "9"})
	for name, amount := range map[string]int64{stakingtypes.ModuleName: 150, govtypes.ModuleName: 9} {
		coins := sdk.NewCoins(sdk.NewInt64Coin("uzrn", amount))
		require.NoError(t, app.BankKeeper.MintCoins(ctx, "tokens", coins))
		require.NoError(t, app.BankKeeper.SendCoinsFromModuleToModule(ctx, "tokens", name, coins))
	}
	if migratedH3 {
		ctx.KVStore(app.keys["knowledge"]).Delete(append([]byte{0x7f, 0x01}, []byte(sdk053IBC10NativeMarker)...))
		seedSDK053IBC10CompletedTestLineage(t, app, ctx)
		for i := 0; i < 3; i++ {
			app.CommitMultiStore().Commit()
		}
	}
	latest := app.CommitMultiStore().Commit().Version
	return app, ctx.WithBlockHeight(latest), fromVM
}

func TestAccountingAuthorityCommittedHandlerPreservesClaimsAndReceipt(t *testing.T) {
	for _, migratedH3 := range []bool{false, true} {
		t.Run(map[bool]string{false: "native-H3", true: "migrated-H3"}[migratedH3], func(t *testing.T) {
			app, ctx, fromVM := committedAccountingSource(t, migratedH3)
			info, err := app.BuildAccountingAuthorityPlanInfo(ctx)
			require.NoError(t, err)
			cached, _ := ctx.CacheContext()
			cachedInfo, err := app.BuildAccountingAuthorityPlanInfo(cached)
			require.NoError(t, err)
			require.Equal(t, info, cachedInfo)
			plan := upgradetypes.Plan{Name: UpgradeNameAccountingAuthorityV1, Height: ctx.BlockHeight() + 2, Info: info}
			require.NoError(t, app.UpgradeKeeper.ScheduleUpgrade(ctx, plan))
			require.NoError(t, app.UpgradeKeeper.DumpUpgradeInfoToDisk(plan.Height, plan))
			// Scheduled early state is refused, then the exact committed H-1
			// state and matching local handoff are admitted.
			require.Error(t, app.ValidateAccountingAuthorityStartup())
			require.Equal(t, plan.Height-1, app.CommitMultiStore().Commit().Version)
			require.NoError(t, app.ValidateAccountingAuthorityStartup())
			wrongLocalPlan := plan
			wrongLocalPlan.Info += " "
			require.NoError(t, app.UpgradeKeeper.DumpUpgradeInfoToDisk(plan.Height, wrongLocalPlan))
			require.Error(t, app.ValidateAccountingAuthorityStartup())
			require.NoError(t, app.UpgradeKeeper.DumpUpgradeInfoToDisk(plan.Height, plan))
			beforeGov := bytes.Clone(ctx.KVStore(app.keys[govtypes.StoreKey]).Get(govtypes.LIPKey("terminal-history")))
			beforeSupply := app.BankKeeper.GetSupply(ctx, "uzrn")
			ctx = ctx.WithBlockHeight(plan.Height).WithHeaderInfo(header.Info{Height: plan.Height, ChainID: ctx.ChainID()})
			cache, write := ctx.CacheContext()
			require.NoError(t, app.UpgradeKeeper.ApplyUpgrade(cache, plan))
			require.False(t, app.ZeroneStakingKeeper.AccountingSafetyEnabled(ctx), "outer block cache must remain atomic")
			write()
			vm, err := app.UpgradeKeeper.GetModuleVersionMap(ctx)
			require.NoError(t, err)
			require.Equal(t, accountingAuthorityTargetVersionMap(), vm)
			require.Equal(t, uint64(1), fromVM[stakingtypes.ModuleName])
			require.True(t, app.ZeroneStakingKeeper.AccountingSafetyEnabled(ctx))
			require.True(t, app.ZeroneGovKeeper.AccountingSafetyEnabled(ctx))
			require.Equal(t, beforeGov, ctx.KVStore(app.keys[govtypes.StoreKey]).Get(govtypes.LIPKey("terminal-history")))
			require.Equal(t, beforeSupply, app.BankKeeper.GetSupply(ctx, "uzrn"))
			require.Equal(t, "150", app.BankKeeper.GetBalance(ctx, app.AccountKeeper.GetModuleAddress(stakingtypes.ModuleName), "uzrn").Amount.String())
			require.Equal(t, "9", app.BankKeeper.GetBalance(ctx, app.AccountKeeper.GetModuleAddress(govtypes.ModuleName), "uzrn").Amount.String())
			require.Equal(t, plan.Height, app.CommitMultiStore().Commit().Version)
			require.NoError(t, app.ValidateAccountingAuthorityStartup())
			metadata, err := app.accountingAuthorityGenesisMetadata(ctx, plan.Height)
			require.NoError(t, err)
			require.Equal(t, "migrated", metadata.Origin)
			require.Equal(t, plan.Info, metadata.Receipt.Info)
			require.Equal(t, accountingPlanDigest(plan), metadata.Receipt.PlanSHA256)
			require.Error(t, app.UpgradeKeeper.ApplyUpgrade(ctx, plan), "applied transition cannot replay")

			// Exported provenance is retained on import without manufacturing a
			// migration done record for the new chain.
			// Remove test-only version wrappers before module interface discovery
			// for export; the persisted historical version map remains untouched.
			for _, name := range []string{"knowledge", "vesting_rewards"} {
				app.ModuleManager.Modules[name] = app.ModuleManager.Modules[name].(archivedH3AccountingModule).AppModule
			}
			genesis, err := app.ModuleManager.ExportGenesis(ctx, app.appCodec)
			require.NoError(t, err)
			genesis[accountingAuthorityGenesisKey], err = json.Marshal(metadata)
			require.NoError(t, err)
			raw, err := json.Marshal(genesis)
			require.NoError(t, err)
			imported := NewZeroneApp(log.NewNopLogger(), dbm.NewMemDB(), nil, true, simtestutil.NewAppOptionsWithFlagHome(t.TempDir()), baseapp.SetChainID("imported-accounting-fixture"))
			importCtx := imported.NewUncachedContext(false, cmtproto.Header{ChainID: "imported-accounting-fixture"})
			_, err = imported.InitChainer(importCtx, &abci.RequestInitChain{ChainId: "imported-accounting-fixture", AppStateBytes: raw})
			require.NoError(t, err)
			done, err := imported.UpgradeKeeper.GetDoneHeight(importCtx, UpgradeNameAccountingAuthorityV1)
			require.NoError(t, err)
			require.Zero(t, done)
			exportedAgain, err := imported.accountingAuthorityGenesisMetadata(importCtx, 0)
			require.NoError(t, err)
			require.Equal(t, "imported-migrated", exportedAgain.Origin)
			require.Equal(t, metadata.Receipt, exportedAgain.Receipt)
			imported.CommitMultiStore().Commit()
			require.NoError(t, imported.ValidateAccountingAuthorityStartup())
		})
	}
}

func TestAccountingAuthorityPreflightRefusesDriftAndIncompleteEvidence(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*ZeroneApp, sdk.Context)
		want   string
	}{
		{"extra custom key", func(app *ZeroneApp, ctx sdk.Context) {
			ctx.KVStore(app.keys[govtypes.StoreKey]).Set([]byte{0x18, 0x01}, []byte{1})
		}, "complete committed"},
		{"new custody denomination", func(app *ZeroneApp, ctx sdk.Context) {
			coins := sdk.NewCoins(sdk.NewInt64Coin("testcoin", 1))
			require.NoError(t, app.BankKeeper.MintCoins(ctx, "tokens", coins))
			require.NoError(t, app.BankKeeper.SendCoinsFromModuleToModule(ctx, "tokens", govtypes.ModuleName, coins))
		}, "complete committed"},
		{"active governance", func(app *ZeroneApp, ctx sdk.Context) {
			app.ZeroneGovKeeper.SetLIP(ctx, &govtypes.LIP{Id: "active", Stage: govtypes.StatusVoting, StakedAmount: "0"})
		}, "terminal"},
		{"unowned bank surplus", func(app *ZeroneApp, ctx sdk.Context) {
			coins := sdk.NewCoins(sdk.NewInt64Coin("uzrn", 1))
			require.NoError(t, app.BankKeeper.MintCoins(ctx, "tokens", coins))
			require.NoError(t, app.BankKeeper.SendCoinsFromModuleToModule(ctx, "tokens", stakingtypes.ModuleName, coins))
		}, "bank balance"},
		{"unknown module version", func(app *ZeroneApp, ctx sdk.Context) {
			vm, _ := app.UpgradeKeeper.GetModuleVersionMap(ctx)
			vm["unexpected"] = 0
			require.NoError(t, app.UpgradeKeeper.SetModuleVersionMap(ctx, vm))
		}, "exact full"},
		{"preexisting safety marker", func(app *ZeroneApp, ctx sdk.Context) {
			require.NoError(t, app.ZeroneGovKeeper.EnableAccountingSafety(ctx))
		}, "markers absent"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			app, ctx, _ := committedAccountingSource(t, false)
			_, err := app.BuildAccountingAuthorityPlanInfo(ctx)
			require.NoError(t, err)
			cache, _ := ctx.CacheContext()
			tc.change(app, cache)
			_, err = app.BuildAccountingAuthorityPlanInfo(cache)
			require.ErrorContains(t, err, tc.want)
			require.False(t, app.ZeroneGovKeeper.AccountingSafetyEnabled(ctx))
		})
	}
	app, ctx, vm := committedAccountingSource(t, false)
	info, err := app.BuildAccountingAuthorityPlanInfo(ctx)
	require.NoError(t, err)
	plan := upgradetypes.Plan{Name: UpgradeNameAccountingAuthorityV1, Height: ctx.BlockHeight() + 1, Info: info}
	app.ZeroneGovKeeper.SetLIP(ctx, &govtypes.LIP{Id: "later-terminal", Stage: govtypes.StatusWithdrawn, StakedAmount: "0"})
	app.CommitMultiStore().Commit()
	plan.Height++
	require.ErrorContains(t, app.validateAccountingAuthoritySource(ctx.WithBlockHeight(plan.Height), plan, vm), "drifted")
	require.False(t, app.ZeroneStakingKeeper.AccountingSafetyEnabled(ctx))
}

func TestAccountingAuthorityPlanAndGenesisMetadataFailClosed(t *testing.T) {
	app, ctx, _ := committedAccountingSource(t, false)
	info, err := app.BuildAccountingAuthorityPlanInfo(ctx)
	require.NoError(t, err)
	for _, malformed := range []string{"", " " + info, info + "{}", strings.Replace(info, `"chain_id":`, `"chain_id":"duplicate","chain_id":`, 1), strings.Replace(info, `"chain_id":`, `"Chain_ID":`, 1), strings.Replace(info, AccountingAuthorityPlanInfoSchema, "unknown", 1), strings.Replace(info, `"schema":`, `"unknown":`, 1), strings.Repeat("x", 1025)} {
		_, err := parseAccountingAuthorityPlanInfo(malformed)
		require.Error(t, err)
	}
	for _, malformed := range []string{`{}`, `null`, `{"schema":"zerone.accounting-authority/genesis-v1"}`, `{"schema":"zerone.accounting-authority/genesis-v1","origin":"migrated"}`, `{"Schema":"zerone.accounting-authority/genesis-v1","origin":"native"}`, `{"schema":"zerone.accounting-authority/genesis-v1","origin":"native","origin":"native"}`, `{"schema":"zerone.accounting-authority/genesis-v1","origin":"unknown"}`} {
		_, err := parseAccountingAuthorityGenesis([]byte(malformed))
		require.Error(t, err)
	}
}

func TestAccountingAuthoritySourceIdentitiesMustMatchCommittedRoots(t *testing.T) {
	for _, name := range []string{"hidden version", "hidden accounting lineage", "changed module account"} {
		t.Run(name, func(t *testing.T) {
			app, ctx, _ := committedAccountingSource(t, false)
			_, err := app.BuildAccountingAuthorityPlanInfo(ctx)
			require.NoError(t, err)
			var cacheChange func(sdk.Context)
			var wantStore string
			switch name {
			case "hidden version":
				vm, err := app.UpgradeKeeper.GetModuleVersionMap(ctx)
				require.NoError(t, err)
				vm["unreviewed-module"] = 0
				require.NoError(t, app.UpgradeKeeper.SetModuleVersionMap(ctx, vm))
				cacheChange = func(cache sdk.Context) {
					cache.KVStore(app.keys[upgradetypes.StoreKey]).Delete(append([]byte{upgradetypes.VersionMapByte}, []byte("unreviewed-module")...))
				}
				wantStore = "upgrade"
			case "hidden accounting lineage":
				require.NoError(t, app.KnowledgeKeeper.WriteMigrationMarker(ctx, accountingAuthorityNativeMarker, "genesis"))
				cacheChange = func(cache sdk.Context) {
					cache.KVStore(app.keys["knowledge"]).Delete(append([]byte{0x7f, 0x01}, []byte(accountingAuthorityNativeMarker)...))
				}
				wantStore = "knowledge"
			case "changed module account":
				address := authtypes.NewModuleAddress("tokens")
				account := app.AccountKeeper.GetAccount(ctx, address)
				require.NotNil(t, account)
				cacheChange = func(cache sdk.Context) {
					app.AccountKeeper.SetModuleAccount(cache, authtypes.NewModuleAccount(authtypes.NewBaseAccount(address, nil, account.GetAccountNumber(), account.GetSequence()+1), "tokens", maccPerms["tokens"]...))
				}
				wantStore = authtypes.StoreKey
			}
			latest := app.CommitMultiStore().Commit().Version
			cache, _ := ctx.WithBlockHeight(latest).CacheContext()
			cacheChange(cache)
			_, err = app.BuildAccountingAuthorityPlanInfo(cache)
			require.ErrorContains(t, err, "complete committed "+wantStore)
		})
	}
}

func TestAccountingAuthorityCandidateRefusesHistoricalH3Piggyback(t *testing.T) {
	app := newSDK053IBC10ScheduledPreflightFixture(t, true, strings.Repeat("a", 64))
	// Restore the actual candidate modules after constructing the archived H3
	// fixture. Its named dry-run must reject the new accounting delta before H3
	// writes its marker or any accounting module marker.
	for _, name := range []string{stakingtypes.ModuleName, govtypes.ModuleName} {
		app.ModuleManager.Modules[name] = app.ModuleManager.Modules[name].(archivedH3AccountingModule).AppModule
	}
	before := app.CommitMultiStore().LastCommitID()
	report, err := app.VerifyScheduledActivationPrestate()
	require.ErrorContains(t, err, "sole owner")
	require.False(t, report.ActivationReady)
	require.Equal(t, before, app.CommitMultiStore().LastCommitID())
	ctx := app.NewUncachedContext(true, cmtproto.Header{Height: before.Version})
	require.False(t, app.ZeroneStakingKeeper.AccountingSafetyEnabled(ctx))
	require.False(t, app.ZeroneGovKeeper.AccountingSafetyEnabled(ctx))
	require.Empty(t, app.KnowledgeKeeper.ReadMigrationMarker(ctx, sdk053IBC10UpgradeMarker))
}
