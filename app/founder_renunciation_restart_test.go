package app

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"os"
	"path/filepath"
	"testing"

	corestore "cosmossdk.io/core/store"
	sdkmath "cosmossdk.io/math"
	upgradetypes "cosmossdk.io/x/upgrade/types"

	abci "github.com/cometbft/cometbft/abci/types"
	dbm "github.com/cosmos/cosmos-db"
	sdkruntime "github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	knowledgekeeper "github.com/zerone-chain/zerone/x/knowledge/keeper"
	knowledgetypes "github.com/zerone-chain/zerone/x/knowledge/types"
	vestingrewardstypes "github.com/zerone-chain/zerone/x/vesting_rewards/types"
)

const founderRestartFeeAmount int64 = 1_000_000

func markerStoreKey(name string) []byte {
	key := []byte{0x7F, 0x01}
	return append(key, []byte(name)...)
}

func deleteRestartMarker(app *ZeroneApp, ctx sdk.Context, name string) {
	ctx.KVStore(app.keys[knowledgetypes.StoreKey]).Delete(markerStoreKey(name))
}

func setRestartDoneHeight(app *ZeroneApp, ctx sdk.Context, name string, height int64) {
	key := make([]byte, 9+len(name))
	key[0] = upgradetypes.DoneByte
	binary.BigEndian.PutUint64(key[1:9], uint64(height))
	copy(key[9:], name)
	ctx.KVStore(app.keys[upgradetypes.StoreKey]).Set(key, []byte{1})
}

func setLegacyFounderParams(t *testing.T, app *ZeroneApp, ctx sdk.Context, founder string) {
	t.Helper()
	params := legacyFounderParams()
	params.FounderAddress = founder
	bz, err := proto.Marshal(params)
	require.NoError(t, err)
	ctx.KVStore(app.keys[vestingrewardstypes.StoreKey]).Set(vestingrewardstypes.ParamsKey, bz)
}

func setLegacyVestingModulePermissions(t *testing.T, app *ZeroneApp, ctx sdk.Context) {
	t.Helper()
	setRestartModulePermissions(
		t,
		app,
		ctx,
		vestingrewardstypes.ModuleName,
		authtypes.Minter,
		authtypes.Burner,
	)
}

func setRestartModulePermissions(
	t *testing.T,
	app *ZeroneApp,
	ctx sdk.Context,
	name string,
	permissions ...string,
) {
	t.Helper()
	address := authtypes.NewModuleAddress(name)
	existing := app.AccountKeeper.GetAccount(ctx, address)
	if existing == nil {
		existing = app.AccountKeeper.GetModuleAccount(ctx, name)
	}
	moduleAccount, ok := existing.(sdk.ModuleAccountI)
	require.True(t, ok)
	rebuilt := authtypes.NewModuleAccount(
		authtypes.NewBaseAccount(
			moduleAccount.GetAddress(),
			nil,
			moduleAccount.GetAccountNumber(),
			moduleAccount.GetSequence(),
		),
		name,
		permissions...,
	)
	app.AccountKeeper.SetModuleAccount(ctx, rebuilt)
}

func firstRestartUserAccount(t *testing.T, app *ZeroneApp, ctx sdk.Context) sdk.AccAddress {
	t.Helper()
	for _, account := range app.AccountKeeper.GetAllAccounts(ctx) {
		if _, isModule := account.(sdk.ModuleAccountI); !isModule {
			return account.GetAddress()
		}
	}
	require.FailNow(t, "restart fixture has no user account")
	return nil
}

func createPendingFounderRestart(
	t *testing.T,
	withFees bool,
) (*ZeroneApp, dbm.DB, string, upgradetypes.Plan, sdk.AccAddress) {
	t.Helper()
	db := dbm.NewMemDB()
	home := t.TempDir()
	app := newRestartTestApp(t, db, home)
	initRestartTestChain(t, app)

	// Finalize H-1, then write the exact persisted halt evidence into that
	// block's root before Commit. This avoids letting the already-H2 test
	// binary observe its own future handler one block early.
	_, err := app.FinalizeBlock(&abci.RequestFinalizeBlock{
		Height: app.LastBlockHeight() + 1,
	})
	require.NoError(t, err)
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
	founder := sdk.AccAddress(bytes.Repeat([]byte{0x42}, 20))
	setLegacyFounderParams(t, app, ctx, founder.String())
	setLegacyVestingModulePermissions(t, app, ctx)
	// Deliberate unrelated drift proves H2 does not carry a global auth-state
	// repair under the founder-renunciation plan.
	setRestartModulePermissions(
		t,
		app,
		ctx,
		vestingrewardstypes.DevelopmentFundModuleName,
		authtypes.Minter,
	)

	if withFees {
		payer := firstRestartUserAccount(t, app, ctx)
		require.NoError(t, app.BankKeeper.SendCoinsFromAccountToModule(
			ctx,
			payer,
			authtypes.FeeCollectorName,
			sdk.NewCoins(sdk.NewCoin(
				sdk.DefaultBondDenom,
				sdkmath.NewInt(founderRestartFeeAmount),
			)),
		))
	}

	plan := upgradetypes.Plan{
		Name:   UpgradeNameFounderRenunciationV1,
		Height: app.LastBlockHeight() + 2,
		Info:   canonicalH2PlanInfo,
	}
	require.NoError(t, app.UpgradeKeeper.ScheduleUpgrade(ctx, plan))
	require.NoError(t, app.UpgradeKeeper.DumpUpgradeInfoToDisk(plan.Height, plan))
	_, err = app.Commit()
	require.NoError(t, err)
	require.Equal(t, app.LastBlockHeight()+1, plan.Height)

	pending := newRestartTestApp(t, db, home)
	return pending, db, home, plan, founder
}

type founderConsensusSnapshot struct {
	params              []byte
	versionMap          map[string]uint64
	vestingPermissions  []string
	developmentPerms    []string
	vestingBalance      string
	researchBalance     string
	developmentBalance  string
	feeCollectorBalance string
	distributionBalance string
	founderBalance      string
	supply              string
	h1Marker            string
	h1MarkerFound       bool
	h2Marker            string
	h2MarkerFound       bool
	h2PlanDigest        string
	h2DigestFound       bool
	h1Done              int64
	h2Done              int64
	plan                upgradetypes.Plan
	planFound           bool
}

func snapshotFounderConsensus(
	t *testing.T,
	app *ZeroneApp,
	founder sdk.AccAddress,
) founderConsensusSnapshot {
	t.Helper()
	ctx := restartReadContext(app)
	params := append(
		[]byte(nil),
		ctx.KVStore(app.keys[vestingrewardstypes.StoreKey]).Get(vestingrewardstypes.ParamsKey)...,
	)
	vm, err := app.UpgradeKeeper.GetModuleVersionMap(ctx)
	require.NoError(t, err)
	vmCopy := make(map[string]uint64, len(vm))
	for name, version := range vm {
		vmCopy[name] = version
	}
	_, permissions, err := app.readVestingRewardsModuleAccountPermissions(ctx)
	require.NoError(t, err)
	developmentAccount := app.AccountKeeper.GetAccount(
		ctx,
		authtypes.NewModuleAddress(vestingrewardstypes.DevelopmentFundModuleName),
	)
	require.NotNil(t, developmentAccount)
	developmentModule, ok := developmentAccount.(sdk.ModuleAccountI)
	require.True(t, ok)
	h1Marker, h1Found, err := app.KnowledgeKeeper.ReadMigrationMarkerPresenceChecked(
		ctx,
		consolidationMigrationMarker,
	)
	require.NoError(t, err)
	h2Marker, h2Found, err := app.KnowledgeKeeper.ReadMigrationMarkerPresenceChecked(
		ctx,
		founderRenunciationMigrationMarker,
	)
	require.NoError(t, err)
	h2PlanDigest, h2DigestFound, err := app.KnowledgeKeeper.ReadMigrationMarkerPresenceChecked(
		ctx,
		founderRenunciationPlanIdentityMarker,
	)
	require.NoError(t, err)
	h1Done, err := app.UpgradeKeeper.GetDoneHeight(ctx, UpgradeNameConsolidationSafetyV1)
	require.NoError(t, err)
	h2Done, err := app.UpgradeKeeper.GetDoneHeight(ctx, UpgradeNameFounderRenunciationV1)
	require.NoError(t, err)
	plan, err := app.UpgradeKeeper.GetUpgradePlan(ctx)
	planFound := err == nil
	if !planFound {
		require.ErrorIs(t, err, upgradetypes.ErrNoUpgradePlanFound)
	}
	balance := func(address sdk.AccAddress) string {
		return app.BankKeeper.GetAllBalances(ctx, address).String()
	}
	return founderConsensusSnapshot{
		params:              params,
		versionMap:          vmCopy,
		vestingPermissions:  append([]string(nil), permissions...),
		developmentPerms:    append([]string(nil), developmentModule.GetPermissions()...),
		vestingBalance:      balance(authtypes.NewModuleAddress(vestingrewardstypes.ModuleName)),
		researchBalance:     balance(authtypes.NewModuleAddress(vestingrewardstypes.ResearchFundModuleName)),
		developmentBalance:  balance(authtypes.NewModuleAddress(vestingrewardstypes.DevelopmentFundModuleName)),
		feeCollectorBalance: balance(authtypes.NewModuleAddress(authtypes.FeeCollectorName)),
		distributionBalance: balance(authtypes.NewModuleAddress(distrtypes.ModuleName)),
		founderBalance:      balance(founder),
		supply:              app.BankKeeper.GetSupply(ctx, sdk.DefaultBondDenom).Amount.String(),
		h1Marker:            h1Marker,
		h1MarkerFound:       h1Found,
		h2Marker:            h2Marker,
		h2MarkerFound:       h2Found,
		h2PlanDigest:        h2PlanDigest,
		h2DigestFound:       h2DigestFound,
		h1Done:              h1Done,
		h2Done:              h2Done,
		plan:                plan,
		planFound:           planFound,
	}
}

func TestFounderRenunciationExecutesThroughRealPreBlockAndRestarts(t *testing.T) {
	app, db, home, plan, founder := createPendingFounderRestart(t, true)
	before := snapshotFounderConsensus(t, app, founder)
	require.Equal(t, plan, before.plan)
	require.True(t, before.planFound)
	require.False(t, before.h2MarkerFound)
	require.False(t, before.h2DigestFound)
	require.EqualValues(t, 0, before.h2Done)
	require.ElementsMatch(t, []string{authtypes.Minter, authtypes.Burner}, before.vestingPermissions)
	require.Equal(t, []string{authtypes.Minter}, before.developmentPerms)

	// This succeeds only if x/upgrade.GetUpgradePlan still exposes the exact
	// committed plan while ApplyUpgrade invokes the handler in real PreBlock.
	response, err := app.FinalizeBlock(&abci.RequestFinalizeBlock{Height: plan.Height})
	require.NoError(t, err)
	require.NotNil(t, response)
	_, err = app.Commit()
	require.NoError(t, err)

	after := snapshotFounderConsensus(t, app, founder)
	require.False(t, after.planFound)
	require.True(t, after.h1MarkerFound)
	require.Equal(t, "migrated", after.h1Marker)
	require.True(t, after.h2MarkerFound)
	require.Equal(t, "migrated", after.h2Marker)
	require.True(t, after.h2DigestFound)
	expectedPlanDigest, err := founderRenunciationPlanIdentityDigest(plan)
	require.NoError(t, err)
	require.Equal(t, expectedPlanDigest, after.h2PlanDigest)
	require.EqualValues(t, 1, after.h1Done)
	require.Equal(t, plan.Height, after.h2Done)
	require.Equal(t, map[string]uint64(app.CurrentModuleVersionMap()), after.versionMap)
	require.Empty(t, after.vestingPermissions)
	require.Equal(t, before.developmentPerms, after.developmentPerms)
	params, err := app.VestingRewardsKeeper.GetStoredParamsChecked(restartReadContext(app))
	require.NoError(t, err)
	require.NoError(t, validateRetiredFounderRenunciationParams(params))

	// Fees collected under H-1 state are routed by V2 BeginBlock immediately
	// after H2 runs in PreBlock at H. The retired founder receives nothing.
	require.Equal(t, before.supply, after.supply)
	require.Equal(t, before.vestingBalance, after.vestingBalance)
	require.Equal(t, "1000000"+sdk.DefaultBondDenom, before.feeCollectorBalance)
	require.Equal(t, "", after.feeCollectorBalance)
	require.Equal(t, sdkmath.NewInt(33_300).String()+sdk.DefaultBondDenom, after.researchBalance)
	require.Equal(t, sdkmath.NewInt(196_700).String()+sdk.DefaultBondDenom, after.developmentBalance)
	require.Equal(t, sdkmath.NewInt(770_000).String()+sdk.DefaultBondDenom, after.distributionBalance)
	require.Equal(t, "", after.founderBalance)

	// Persisted completed evidence, including the historical disk packet, is
	// accepted by a fresh process.
	restarted := newRestartTestApp(t, db, home)
	require.Equal(t, after, snapshotFounderConsensus(t, restarted, founder))
}

type postMigrationMarkerFailureService struct {
	store *postMigrationMarkerFailureStore
}

func (s postMigrationMarkerFailureService) OpenKVStore(context.Context) corestore.KVStore {
	return s.store
}

type postMigrationMarkerFailureStore struct {
	h1Reads int
}

func (s *postMigrationMarkerFailureStore) Get(key []byte) ([]byte, error) {
	name := ""
	if len(key) >= 2 {
		name = string(key[2:])
	}
	switch name {
	case consolidationMigrationMarker:
		s.h1Reads++
		if s.h1Reads > 1 {
			return nil, errors.New("injected post-migration H1 marker read failure")
		}
		return []byte("migrated"), nil
	case consolidationNativeLineageMarker,
		founderRenunciationMigrationMarker,
		founderRenunciationNativeLineageMarker:
		return nil, nil
	default:
		return nil, nil
	}
}

func (*postMigrationMarkerFailureStore) Has([]byte) (bool, error) { return false, nil }
func (*postMigrationMarkerFailureStore) Set([]byte, []byte) error {
	return errors.New("unexpected marker write")
}
func (*postMigrationMarkerFailureStore) Delete([]byte) error { return nil }
func (*postMigrationMarkerFailureStore) Iterator([]byte, []byte) (corestore.Iterator, error) {
	return nil, errors.New("unexpected iterator")
}
func (*postMigrationMarkerFailureStore) ReverseIterator(
	[]byte,
	[]byte,
) (corestore.Iterator, error) {
	return nil, errors.New("unexpected reverse iterator")
}

func TestFailedFounderRenunciationPreBlockRollsBackEveryConsensusMutation(t *testing.T) {
	app, db, home, plan, founder := createPendingFounderRestart(t, true)
	before := snapshotFounderConsensus(t, app, founder)

	// The injected failure occurs on the handler's second H1-marker read:
	// after RunMigrations rewrites Params and after permission reconciliation,
	// but before the H2 marker, VersionMap, plan-clear, or done writes.
	failingMarkers := &postMigrationMarkerFailureStore{}
	app.KnowledgeKeeper = knowledgekeeper.NewKeeper(
		postMigrationMarkerFailureService{store: failingMarkers},
		app.appCodec,
		"",
		nil,
		nil,
	)
	response, err := app.FinalizeBlock(&abci.RequestFinalizeBlock{Height: plan.Height})
	require.ErrorContains(t, err, "injected post-migration H1 marker read failure")
	if response != nil {
		require.Empty(t, response.Events)
	}
	require.Equal(t, 2, failingMarkers.h1Reads)

	// A fresh process reads the committed root, not the failed FinalizeBlock
	// cache. Every observed consensus surface remains byte-for-byte/logically
	// identical and the exact pending H2 lineage is still restartable.
	restarted := newRestartTestApp(t, db, home)
	after := snapshotFounderConsensus(t, restarted, founder)
	require.Equal(t, before, after)
}

type finalFounderMarkerFailureState struct {
	writes      []string
	digestValue string
	panicFinal  bool
}

type finalFounderMarkerFailureService struct {
	delegate corestore.KVStoreService
	state    *finalFounderMarkerFailureState
}

func (s finalFounderMarkerFailureService) OpenKVStore(ctx context.Context) corestore.KVStore {
	return &finalFounderMarkerFailureStore{
		KVStore: s.delegate.OpenKVStore(ctx),
		state:   s.state,
	}
}

type finalFounderMarkerFailureStore struct {
	corestore.KVStore
	state *finalFounderMarkerFailureState
}

func (s *finalFounderMarkerFailureStore) Set(key, value []byte) error {
	switch {
	case bytes.Equal(key, markerStoreKey(founderRenunciationPlanIdentityMarker)):
		s.state.writes = append(s.state.writes, founderRenunciationPlanIdentityMarker)
		s.state.digestValue = string(value)
		return s.KVStore.Set(key, value)
	case bytes.Equal(key, markerStoreKey(founderRenunciationMigrationMarker)):
		s.state.writes = append(s.state.writes, founderRenunciationMigrationMarker)
		if s.state.panicFinal {
			panic("injected final H2 marker panic")
		}
		return errors.New("injected final H2 marker write failure")
	default:
		return s.KVStore.Set(key, value)
	}
}

func TestFinalFounderRenunciationMarkerFailureRollsBackDigestAndEveryMutation(t *testing.T) {
	tests := []struct {
		name       string
		panicFinal bool
		message    string
	}{
		{name: "error", message: "injected final H2 marker write failure"},
		{name: "panic", panicFinal: true, message: "injected final H2 marker panic"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			app, db, home, plan, founder := createPendingFounderRestart(t, true)
			before := snapshotFounderConsensus(t, app, founder)
			state := &finalFounderMarkerFailureState{panicFinal: tc.panicFinal}
			app.KnowledgeKeeper = knowledgekeeper.NewKeeper(
				finalFounderMarkerFailureService{
					delegate: sdkruntime.NewKVStoreService(app.keys[knowledgetypes.StoreKey]),
					state:    state,
				},
				app.appCodec,
				"",
				nil,
				nil,
			)

			response, err := app.FinalizeBlock(&abci.RequestFinalizeBlock{Height: plan.Height})
			require.ErrorContains(t, err, tc.message)
			if response != nil {
				require.Empty(t, response.Events)
			}
			require.Equal(t, []string{
				founderRenunciationPlanIdentityMarker,
				founderRenunciationMigrationMarker,
			}, state.writes)
			expectedDigest, digestErr := founderRenunciationPlanIdentityDigest(plan)
			require.NoError(t, digestErr)
			require.Equal(t, expectedDigest, state.digestValue)

			// The digest write reached the real FinalizeBlock cache before the
			// completion seal failed. A fresh process must nevertheless observe the
			// exact pre-H2 root, including an absent digest and pending plan.
			restarted := newRestartTestApp(t, db, home)
			require.Equal(t, before, snapshotFounderConsensus(t, restarted, founder))
		})
	}
}

func completeFounderRestart(
	t *testing.T,
) (*ZeroneApp, dbm.DB, string, upgradetypes.Plan, sdk.AccAddress) {
	t.Helper()
	app, db, home, plan, founder := createPendingFounderRestart(t, false)
	_, err := app.FinalizeBlock(&abci.RequestFinalizeBlock{Height: plan.Height})
	require.NoError(t, err)
	_, err = app.Commit()
	require.NoError(t, err)
	return app, db, home, plan, founder
}

func TestCompletedFounderRenunciationRestartRejectsPlanIdentityDrift(t *testing.T) {
	app, db, home, plan, _ := completeFounderRestart(t)
	drifted := plan
	drifted.Info = `{"packet":"canonical-but-different"}`
	require.NoError(t, app.UpgradeKeeper.DumpUpgradeInfoToDisk(drifted.Height, drifted))
	require.Panics(t, func() { _ = newRestartTestApp(t, db, home) })
}

func TestCompletedFounderRenunciationRestartRejectsInvalidPersistedPlanDigest(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(sdk.Context, *ZeroneApp)
	}{
		{
			name: "absent",
			mutate: func(ctx sdk.Context, app *ZeroneApp) {
				ctx.KVStore(app.keys[knowledgetypes.StoreKey]).
					Delete(markerStoreKey(founderRenunciationPlanIdentityMarker))
			},
		},
		{
			name: "empty",
			mutate: func(ctx sdk.Context, app *ZeroneApp) {
				ctx.KVStore(app.keys[knowledgetypes.StoreKey]).
					Set(markerStoreKey(founderRenunciationPlanIdentityMarker), []byte{})
			},
		},
		{
			name: "forged",
			mutate: func(ctx sdk.Context, app *ZeroneApp) {
				ctx.KVStore(app.keys[knowledgetypes.StoreKey]).Set(
					markerStoreKey(founderRenunciationPlanIdentityMarker),
					[]byte("0000000000000000000000000000000000000000000000000000000000000000"),
				)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			app, db, home, _, _ := completeFounderRestart(t)
			tc.mutate(restartMutationContext(app), app)
			commitRestartMutation(t, app)
			require.Panics(t, func() { _ = newRestartTestApp(t, db, home) })
		})
	}
}

func TestCompletedFounderRenunciationRestartAcceptsConsensusDigestWithoutDiskPacket(t *testing.T) {
	app, db, home, _, founder := completeFounderRestart(t)
	want := snapshotFounderConsensus(t, app, founder)
	path := filepath.Join(home, "data", upgradetypes.UpgradeInfoFilename)
	require.NoError(t, os.Remove(path))
	restarted := newRestartTestApp(t, db, home)
	require.Equal(t, want, snapshotFounderConsensus(t, restarted, founder))
}

func TestFounderRenunciationRestartRejectsCorruptPersistedEvidence(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*ZeroneApp, sdk.Context)
	}{
		{
			name: "missing strict params",
			mutate: func(app *ZeroneApp, ctx sdk.Context) {
				ctx.KVStore(app.keys[vestingrewardstypes.StoreKey]).
					Delete(vestingrewardstypes.ParamsKey)
			},
		},
		{
			name: "corrupt strict params",
			mutate: func(app *ZeroneApp, ctx sdk.Context) {
				ctx.KVStore(app.keys[vestingrewardstypes.StoreKey]).
					Set(vestingrewardstypes.ParamsKey, []byte{0xff, 0xff})
			},
		},
		{
			name: "present empty migration marker",
			mutate: func(app *ZeroneApp, ctx sdk.Context) {
				ctx.KVStore(app.keys[knowledgetypes.StoreKey]).
					Set(markerStoreKey(founderRenunciationMigrationMarker), []byte{})
			},
		},
		{
			name: "native vesting permission drift",
			mutate: func(app *ZeroneApp, ctx sdk.Context) {
				setRestartModulePermissions(
					t,
					app,
					ctx,
					vestingrewardstypes.ModuleName,
					authtypes.Minter,
				)
			},
		},
		{
			name: "malformed done key panics inside SDK reader",
			mutate: func(app *ZeroneApp, ctx sdk.Context) {
				ctx.KVStore(app.keys[upgradetypes.StoreKey]).
					Set([]byte{upgradetypes.DoneByte}, []byte{1})
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			db := dbm.NewMemDB()
			home := t.TempDir()
			app := newRestartTestApp(t, db, home)
			initRestartTestChain(t, app)
			_, err := app.FinalizeBlock(&abci.RequestFinalizeBlock{
				Height: app.LastBlockHeight() + 1,
			})
			require.NoError(t, err)
			tc.mutate(app, restartMutationContext(app))
			_, err = app.Commit()
			require.NoError(t, err)
			require.Panics(t, func() { _ = newRestartTestApp(t, db, home) })
		})
	}
}

func TestFounderRenunciationRestartRejectsUnreadableDiskPacketBeforeLoad(t *testing.T) {
	db := dbm.NewMemDB()
	home := t.TempDir()
	app := newRestartTestApp(t, db, home)
	initRestartTestChain(t, app)
	path := filepath.Join(home, "data", upgradetypes.UpgradeInfoFilename)
	require.NoError(t, os.WriteFile(path, []byte("{"), 0o600))
	require.Panics(t, func() { _ = newRestartTestApp(t, db, home) })
}

func TestFounderRenunciationEvidenceReaderPropagatesMarkerReadErrors(t *testing.T) {
	db := dbm.NewMemDB()
	home := t.TempDir()
	app := newRestartTestApp(t, db, home)
	initRestartTestChain(t, app)
	injected := errors.New("injected marker store read error")
	app.KnowledgeKeeper = knowledgekeeper.NewKeeper(
		startupReadErrorStoreService{err: injected},
		app.appCodec,
		"",
		nil,
		nil,
	)
	_, err := app.readFounderRenunciationEvidence(
		restartReadContext(app),
		app.LastBlockHeight(),
	)
	require.ErrorIs(t, err, injected)
}
