package app

import (
	"context"
	"encoding/json"
	"testing"

	"cosmossdk.io/core/header"
	corestore "cosmossdk.io/core/store"
	"cosmossdk.io/log"
	sdkmath "cosmossdk.io/math"
	upgradetypes "cosmossdk.io/x/upgrade/types"

	abci "github.com/cometbft/cometbft/abci/types"
	cmted25519 "github.com/cometbft/cometbft/crypto/ed25519"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	cmttypes "github.com/cometbft/cometbft/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/baseapp"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	"github.com/cosmos/cosmos-sdk/server"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/require"

	knowledgetypes "github.com/zerone-chain/zerone/x/knowledge/types"
	vestingrewardskeeper "github.com/zerone-chain/zerone/x/vesting_rewards/keeper"
	vestingrewardstypes "github.com/zerone-chain/zerone/x/vesting_rewards/types"
)

const (
	restartTestChainID       = "zerone-startup-restart-1"
	restartCanonicalPlanInfo = `{"packet":"ipfs://consolidation-h1","sha256":"0123456789abcdef"}`
)

func restartAppOptions(home string) servertypes.AppOptions {
	return simtestutil.NewAppOptionsWithFlagHome(home)
}

func newRestartTestApp(t *testing.T, db dbm.DB, home string) *ZeroneApp {
	t.Helper()
	return NewZeroneApp(
		log.NewNopLogger(),
		db,
		nil,
		true,
		restartAppOptions(home),
		baseapp.SetChainID(restartTestChainID),
	)
}

func restartGenesisWithValidator(t *testing.T, app *ZeroneApp) GenesisState {
	t.Helper()
	genesis := app.DefaultGenesis()
	privVal := cmted25519.GenPrivKey()
	validator := cmttypes.NewValidator(privVal.PubKey(), 1)
	valSet := cmttypes.NewValidatorSet([]*cmttypes.Validator{validator})

	accountKey := cmted25519.GenPrivKey()
	account := authtypes.NewBaseAccount(accountKey.PubKey().Address().Bytes(), nil, 0, 0)
	accountBalance := banktypes.Balance{
		Address: account.GetAddress().String(),
		Coins:   sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdkmath.NewInt(100_000_000_000))),
	}
	genesis[authtypes.ModuleName] = app.AppCodec().MustMarshalJSON(
		authtypes.NewGenesisState(authtypes.DefaultParams(), []authtypes.GenesisAccount{account}),
	)

	bondAmount := sdk.DefaultPowerReduction
	validators := make([]stakingtypes.Validator, 0, len(valSet.Validators))
	delegations := make([]stakingtypes.Delegation, 0, len(valSet.Validators))
	for _, cometValidator := range valSet.Validators {
		publicKey, err := cryptocodec.FromCmtPubKeyInterface(cometValidator.PubKey)
		require.NoError(t, err)
		publicKeyAny, err := codectypes.NewAnyWithValue(publicKey)
		require.NoError(t, err)
		operator := sdk.ValAddress(cometValidator.Address).String()
		validators = append(validators, stakingtypes.Validator{
			OperatorAddress:   operator,
			ConsensusPubkey:   publicKeyAny,
			Status:            stakingtypes.Bonded,
			Tokens:            bondAmount,
			DelegatorShares:   sdkmath.LegacyOneDec(),
			Description:       stakingtypes.Description{},
			Commission:        stakingtypes.NewCommission(sdkmath.LegacyZeroDec(), sdkmath.LegacyZeroDec(), sdkmath.LegacyZeroDec()),
			MinSelfDelegation: sdkmath.ZeroInt(),
		})
		delegations = append(delegations, stakingtypes.NewDelegation(
			account.GetAddress().String(),
			operator,
			sdkmath.LegacyOneDec(),
		))
	}
	genesis[stakingtypes.ModuleName] = app.AppCodec().MustMarshalJSON(
		stakingtypes.NewGenesisState(stakingtypes.DefaultParams(), validators, delegations),
	)

	bondedPoolBalance := banktypes.Balance{
		Address: authtypes.NewModuleAddress(stakingtypes.BondedPoolName).String(),
		Coins:   sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, bondAmount)),
	}
	totalSupply := accountBalance.Coins.Add(bondedPoolBalance.Coins...)
	genesis[banktypes.ModuleName] = app.AppCodec().MustMarshalJSON(banktypes.NewGenesisState(
		banktypes.DefaultGenesisState().Params,
		[]banktypes.Balance{accountBalance, bondedPoolBalance},
		totalSupply,
		nil,
		nil,
	))
	return genesis
}

func setRestartVestingGenesis(
	t *testing.T,
	_ *ZeroneApp,
	genesis GenesisState,
	mutate func(*vestingrewardstypes.GenesisState),
) {
	t.Helper()
	var state vestingrewardstypes.GenesisState
	require.NoError(t, json.Unmarshal(genesis[vestingrewardstypes.ModuleName], &state))
	mutate(&state)
	raw, err := json.Marshal(&state)
	require.NoError(t, err)
	genesis[vestingrewardstypes.ModuleName] = raw
}

func appendRestartAuthAccount(
	t *testing.T,
	app *ZeroneApp,
	genesis GenesisState,
	account authtypes.GenesisAccount,
) {
	t.Helper()
	var state authtypes.GenesisState
	app.AppCodec().MustUnmarshalJSON(genesis[authtypes.ModuleName], &state)
	accounts, err := authtypes.UnpackAccounts(state.Accounts)
	require.NoError(t, err)
	accounts = append(accounts, account)
	genesis[authtypes.ModuleName] = app.AppCodec().MustMarshalJSON(
		authtypes.NewGenesisState(state.Params, accounts),
	)
}

type nativeParamsProofStoreService struct {
	value []byte
}

func (s nativeParamsProofStoreService) OpenKVStore(context.Context) corestore.KVStore {
	return nativeParamsProofStore{value: s.value}
}

type nativeParamsProofStore struct {
	value []byte
}

func (s nativeParamsProofStore) Get([]byte) ([]byte, error) {
	return append([]byte(nil), s.value...), nil
}
func (nativeParamsProofStore) Has([]byte) (bool, error) { return false, nil }
func (nativeParamsProofStore) Set([]byte, []byte) error { return nil }
func (nativeParamsProofStore) Delete([]byte) error      { return nil }
func (nativeParamsProofStore) Iterator([]byte, []byte) (corestore.Iterator, error) {
	return nil, nil
}
func (nativeParamsProofStore) ReverseIterator([]byte, []byte) (corestore.Iterator, error) {
	return nil, nil
}

func assertFounderNativeMarkersAbsent(t *testing.T, app *ZeroneApp) {
	t.Helper()
	ctx := app.NewUncachedContext(false, cmtproto.Header{ChainID: restartTestChainID})
	for _, marker := range []string{
		consolidationNativeLineageMarker,
		founderRenunciationNativeLineageMarker,
		consolidationMigrationMarker,
		founderRenunciationMigrationMarker,
		founderRenunciationPlanIdentityMarker,
	} {
		_, found, err := app.KnowledgeKeeper.ReadMigrationMarkerPresenceChecked(ctx, marker)
		require.NoError(t, err)
		require.False(t, found, marker)
	}
}

func initRestartTestChain(t *testing.T, app *ZeroneApp) {
	t.Helper()
	genesisBytes, err := json.Marshal(restartGenesisWithValidator(t, app))
	require.NoError(t, err)
	_, err = app.InitChain(&abci.RequestInitChain{
		ChainId:         restartTestChainID,
		AppStateBytes:   genesisBytes,
		ConsensusParams: simtestutil.DefaultConsensusParams,
	})
	require.NoError(t, err)
	_, err = app.FinalizeBlock(&abci.RequestFinalizeBlock{Height: 1})
	require.NoError(t, err)
	_, err = app.Commit()
	require.NoError(t, err)
}

func restartMutationContext(app *ZeroneApp) sdk.Context {
	height := app.LastBlockHeight() + 1
	return app.NewUncachedContext(false, cmtproto.Header{Height: height, ChainID: restartTestChainID}).
		WithHeaderInfo(header.Info{Height: height, ChainID: restartTestChainID})
}

func restartReadContext(app *ZeroneApp) sdk.Context {
	height := app.LastBlockHeight()
	return app.NewContext(true).WithHeaderInfo(header.Info{Height: height, ChainID: restartTestChainID})
}

func deleteNativeLineageMarker(app *ZeroneApp, ctx sdk.Context) {
	key := append([]byte{0x7F, 0x01}, []byte(consolidationNativeLineageMarker)...)
	ctx.KVStore(app.keys[knowledgetypes.StoreKey]).Delete(key)
}

func commitRestartMutation(t *testing.T, app *ZeroneApp) {
	t.Helper()
	_, err := app.FinalizeBlock(&abci.RequestFinalizeBlock{Height: app.LastBlockHeight() + 1})
	require.NoError(t, err)
	_, err = app.Commit()
	require.NoError(t, err)
}

func restartPreVersionMap(app *ZeroneApp) map[string]uint64 {
	return consolidationPreVersionMap(app.CurrentModuleVersionMap())
}

func TestNativeH1AndH2LineageIsWrittenAtGenesisAndAcceptedOnRestart(t *testing.T) {
	db := dbm.NewMemDB()
	home := t.TempDir()
	app := newRestartTestApp(t, db, home)
	initRestartTestChain(t, app)

	ctx := restartReadContext(app)
	value, found, err := app.KnowledgeKeeper.ReadMigrationMarkerPresenceChecked(
		ctx,
		consolidationNativeLineageMarker,
	)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, consolidationNativeLineageValue, value)
	_, h1Found, err := app.KnowledgeKeeper.ReadMigrationMarkerPresenceChecked(
		ctx,
		consolidationMigrationMarker,
	)
	require.NoError(t, err)
	require.False(t, h1Found)
	h2Value, h2NativeFound, err := app.KnowledgeKeeper.ReadMigrationMarkerPresenceChecked(
		ctx,
		founderRenunciationNativeLineageMarker,
	)
	require.NoError(t, err)
	require.True(t, h2NativeFound)
	require.Equal(t, founderRenunciationNativeLineageValue, h2Value)
	_, h2Found, err := app.KnowledgeKeeper.ReadMigrationMarkerPresenceChecked(
		ctx,
		founderRenunciationMigrationMarker,
	)
	require.NoError(t, err)
	require.False(t, h2Found)
	_, h2DigestFound, err := app.KnowledgeKeeper.ReadMigrationMarkerPresenceChecked(
		ctx,
		founderRenunciationPlanIdentityMarker,
	)
	require.NoError(t, err)
	require.False(t, h2DigestFound)
	params, err := app.VestingRewardsKeeper.GetStoredParamsChecked(ctx)
	require.NoError(t, err)
	require.NoError(t, validateRetiredFounderRenunciationParams(params))
	accountFound, permissions, err := app.readVestingRewardsModuleAccountPermissions(ctx)
	require.NoError(t, err)
	require.True(t, !accountFound || len(permissions) == 0)
	require.NotNil(t, newRestartTestApp(t, db, home))
}

func TestNativeFounderRenunciationGenesisRejectsInvalidStateBeforeLineageMarkers(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*testing.T, *ZeroneApp, GenesisState)
		wantErr string
	}{
		{
			name: "missing params",
			mutate: func(t *testing.T, app *ZeroneApp, genesis GenesisState) {
				setRestartVestingGenesis(t, app, genesis, func(state *vestingrewardstypes.GenesisState) {
					state.Params = nil
				})
			},
			wantErr: "vesting_rewards params are required",
		},
		{
			name: "corrupt genesis",
			mutate: func(_ *testing.T, _ *ZeroneApp, genesis GenesisState) {
				genesis[vestingrewardstypes.ModuleName] = []byte(`{"params":"corrupt"}`)
			},
			wantErr: "decode native vesting_rewards genesis",
		},
		{
			name: "legacy params",
			mutate: func(t *testing.T, app *ZeroneApp, genesis GenesisState) {
				setRestartVestingGenesis(t, app, genesis, func(state *vestingrewardstypes.GenesisState) {
					state.Params = legacyFounderParams()
				})
			},
			wantErr: "transaction-presence block rewards are permanently retired",
		},
		{
			name: "missing stored params after module init",
			mutate: func(_ *testing.T, app *ZeroneApp, _ GenesisState) {
				app.VestingRewardsKeeper = vestingrewardskeeper.NewKeeper(
					app.appCodec,
					nativeParamsProofStoreService{},
					nil,
					nil,
					"",
				)
			},
			wantErr: "vesting_rewards params are missing",
		},
		{
			name: "corrupt stored params after module init",
			mutate: func(_ *testing.T, app *ZeroneApp, _ GenesisState) {
				app.VestingRewardsKeeper = vestingrewardskeeper.NewKeeper(
					app.appCodec,
					nativeParamsProofStoreService{value: []byte{0xff, 0xff}},
					nil,
					nil,
					"",
				)
			},
			wantErr: "unmarshal params",
		},
		{
			name: "minter permission",
			mutate: func(t *testing.T, app *ZeroneApp, genesis GenesisState) {
				base := authtypes.NewBaseAccount(
					authtypes.NewModuleAddress(vestingrewardstypes.ModuleName), nil, 1, 0,
				)
				appendRestartAuthAccount(t, app, genesis, authtypes.NewModuleAccount(
					base, vestingrewardstypes.ModuleName, authtypes.Minter,
				))
			},
			wantErr: "native vesting_rewards module account retains permissions",
		},
		{
			name: "burner permission",
			mutate: func(t *testing.T, app *ZeroneApp, genesis GenesisState) {
				base := authtypes.NewBaseAccount(
					authtypes.NewModuleAddress(vestingrewardstypes.ModuleName), nil, 1, 0,
				)
				appendRestartAuthAccount(t, app, genesis, authtypes.NewModuleAccount(
					base, vestingrewardstypes.ModuleName, authtypes.Burner,
				))
			},
			wantErr: "native vesting_rewards module account retains permissions",
		},
		{
			name: "base account at module address",
			mutate: func(t *testing.T, app *ZeroneApp, genesis GenesisState) {
				appendRestartAuthAccount(t, app, genesis, authtypes.NewBaseAccount(
					authtypes.NewModuleAddress(vestingrewardstypes.ModuleName), nil, 1, 0,
				))
			},
			wantErr: "is not a module account",
		},
		{
			name: "wrong module name at module address",
			mutate: func(t *testing.T, app *ZeroneApp, genesis GenesisState) {
				base := authtypes.NewBaseAccount(
					authtypes.NewModuleAddress(vestingrewardstypes.ModuleName), nil, 1, 0,
				)
				appendRestartAuthAccount(t, app, genesis, authtypes.NewModuleAccount(
					base, "wrong-module-name",
				))
			},
			wantErr: `has module name "wrong-module-name"`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			app := newRestartTestApp(t, dbm.NewMemDB(), t.TempDir())
			genesis := restartGenesisWithValidator(t, app)
			tc.mutate(t, app, genesis)
			genesisBytes, err := json.Marshal(genesis)
			require.NoError(t, err)
			_, err = app.InitChain(&abci.RequestInitChain{
				ChainId:         restartTestChainID,
				AppStateBytes:   genesisBytes,
				ConsensusParams: simtestutil.DefaultConsensusParams,
			})
			require.ErrorContains(t, err, tc.wantErr)
			assertFounderNativeMarkersAbsent(t, app)
		})
	}
}

func TestNativeFounderRenunciationGenesisAcceptsPermissionlessModuleAccount(t *testing.T) {
	db := dbm.NewMemDB()
	home := t.TempDir()
	app := newRestartTestApp(t, db, home)
	genesis := restartGenesisWithValidator(t, app)
	base := authtypes.NewBaseAccount(
		authtypes.NewModuleAddress(vestingrewardstypes.ModuleName), nil, 1, 0,
	)
	appendRestartAuthAccount(t, app, genesis, authtypes.NewModuleAccount(
		base, vestingrewardstypes.ModuleName,
	))
	genesisBytes, err := json.Marshal(genesis)
	require.NoError(t, err)
	_, err = app.InitChain(&abci.RequestInitChain{
		ChainId:         restartTestChainID,
		AppStateBytes:   genesisBytes,
		ConsensusParams: simtestutil.DefaultConsensusParams,
	})
	require.NoError(t, err)
	_, err = app.FinalizeBlock(&abci.RequestFinalizeBlock{Height: 1})
	require.NoError(t, err)
	_, err = app.Commit()
	require.NoError(t, err)
	found, permissions, err := app.readVestingRewardsModuleAccountPermissions(restartReadContext(app))
	require.NoError(t, err)
	require.True(t, found)
	require.Empty(t, permissions)
	require.NotNil(t, newRestartTestApp(t, db, home))
}

func TestExactPendingH1RestartRequiresMatchingCommittedAndDiskPlans(t *testing.T) {
	db := dbm.NewMemDB()
	home := t.TempDir()
	app := newRestartTestApp(t, db, home)
	initRestartTestChain(t, app)

	// Construct the historical H1-1 halt state without allowing this new
	// binary's registered handler to observe the future plan in PreBlock.
	_, err := app.FinalizeBlock(&abci.RequestFinalizeBlock{Height: app.LastBlockHeight() + 1})
	require.NoError(t, err)
	ctx := restartMutationContext(app)
	deleteNativeLineageMarker(app, ctx)
	require.NoError(t, app.UpgradeKeeper.SetModuleVersionMap(ctx, restartPreVersionMap(app)))
	plan := upgradetypes.Plan{
		Name:   UpgradeNameConsolidationSafetyV1,
		Height: app.LastBlockHeight() + 2,
		Info:   restartCanonicalPlanInfo,
	}
	require.NoError(t, app.UpgradeKeeper.ScheduleUpgrade(ctx, plan))
	require.NoError(t, app.UpgradeKeeper.DumpUpgradeInfoToDisk(plan.Height, plan))
	_, err = app.Commit()
	require.NoError(t, err)
	require.Equal(t, app.LastBlockHeight()+1, plan.Height)
	// An H2 binary must never execute H1. Even an otherwise exact H1 pending
	// halt is outside H2's four accepted states; operators must first complete
	// H1 with the accepted H1 binary.
	require.Panics(t, func() { _ = newRestartTestApp(t, db, home) })

	require.Panics(t, func() {
		_ = NewZeroneApp(
			log.NewNopLogger(),
			db,
			nil,
			true,
			simtestutil.AppOptionsMap{
				"home":                        home,
				server.FlagUnsafeSkipUpgrades: []int{int(plan.Height)},
			},
			baseapp.SetChainID(restartTestChainID),
		)
	})
}

func TestPlanlessLegacyRestartRefusesBeforeAnyABCIBehaviorIsReachable(t *testing.T) {
	db := dbm.NewMemDB()
	home := t.TempDir()
	app := newRestartTestApp(t, db, home)
	initRestartTestChain(t, app)

	ctx := restartMutationContext(app)
	deleteNativeLineageMarker(app, ctx)
	require.NoError(t, app.UpgradeKeeper.SetModuleVersionMap(ctx, restartPreVersionMap(app)))
	commitRestartMutation(t, app)

	// No application is returned, so K6 BeginBlock/conjecture behavior, P2
	// issuance, substrate behavior, and every L5 message/quote are unreachable.
	require.Panics(t, func() { _ = newRestartTestApp(t, db, home) })
}

func TestMarkerWithoutDoneRefusesCompletedRestart(t *testing.T) {
	db := dbm.NewMemDB()
	home := t.TempDir()
	app := newRestartTestApp(t, db, home)
	initRestartTestChain(t, app)

	ctx := restartMutationContext(app)
	deleteNativeLineageMarker(app, ctx)
	require.NoError(t, app.KnowledgeKeeper.WriteMigrationMarker(
		ctx,
		consolidationMigrationMarker,
		"migrated",
	))
	require.NoError(t, app.UpgradeKeeper.SetModuleVersionMap(ctx, app.CurrentModuleVersionMap()))
	commitRestartMutation(t, app)
	require.Panics(t, func() { _ = newRestartTestApp(t, db, home) })
}

func TestCompletedH1WithoutExactPendingH2PlanIsRejected(t *testing.T) {
	db := dbm.NewMemDB()
	home := t.TempDir()
	app := newRestartTestApp(t, db, home)
	initRestartTestChain(t, app)

	ctx := restartMutationContext(app)
	deleteRestartMarker(app, ctx, consolidationNativeLineageMarker)
	deleteRestartMarker(app, ctx, founderRenunciationNativeLineageMarker)
	require.NoError(t, app.KnowledgeKeeper.WriteMigrationMarker(
		ctx,
		consolidationMigrationMarker,
		"migrated",
	))
	setRestartDoneHeight(app, ctx, UpgradeNameConsolidationSafetyV1, 1)
	require.NoError(t, app.UpgradeKeeper.SetModuleVersionMap(
		ctx,
		founderRenunciationPreVersionMap(app.CurrentModuleVersionMap()),
	))
	setLegacyFounderParams(t, app, ctx, "")
	commitRestartMutation(t, app)

	// H1 completion alone is necessary but not sufficient: exact committed and
	// local H2 plans at latest+1 are mandatory for the V1 executable surface.
	require.Panics(t, func() { _ = newRestartTestApp(t, db, home) })
}
