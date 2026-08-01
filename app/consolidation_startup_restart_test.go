package app

import (
	"encoding/json"
	"testing"

	"cosmossdk.io/core/header"
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

func TestNativeConsolidationLineageIsWrittenAtGenesisAndAcceptedOnRestart(t *testing.T) {
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
	require.NotNil(t, newRestartTestApp(t, db, home))

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

func TestCompletedH1EvidenceIsAcceptedOnRestart(t *testing.T) {
	db := dbm.NewMemDB()
	home := t.TempDir()
	app := newRestartTestApp(t, db, home)
	initRestartTestChain(t, app)

	ctx := restartMutationContext(app)
	deleteNativeLineageMarker(app, ctx)
	plan := upgradetypes.Plan{
		Name:   UpgradeNameConsolidationSafetyV1,
		Height: app.LastBlockHeight() + 1,
		Info:   restartCanonicalPlanInfo,
	}
	require.NoError(t, app.UpgradeKeeper.SetModuleVersionMap(ctx, restartPreVersionMap(app)))
	require.NoError(t, app.UpgradeKeeper.DumpUpgradeInfoToDisk(plan.Height, plan))
	require.NoError(t, app.UpgradeKeeper.ApplyUpgrade(ctx, plan))
	commitRestartMutation(t, app)

	restarted := newRestartTestApp(t, db, home)
	done, err := restarted.UpgradeKeeper.GetDoneHeight(
		restartReadContext(restarted),
		UpgradeNameConsolidationSafetyV1,
	)
	require.NoError(t, err)
	require.Positive(t, done)
	require.LessOrEqual(t, done, restarted.LastBlockHeight())
}
