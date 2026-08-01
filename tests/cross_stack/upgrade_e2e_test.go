package cross_stack_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"cosmossdk.io/core/header"
	"cosmossdk.io/log"
	sdkmath "cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"

	upgradetypes "cosmossdk.io/x/upgrade/types"
	abci "github.com/cometbft/cometbft/abci/types"
	cmted25519 "github.com/cometbft/cometbft/crypto/ed25519"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/cosmos/cosmos-sdk/baseapp"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	sdkgov "github.com/cosmos/cosmos-sdk/x/gov"
	sdkgovtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	govv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/require"

	zeroneapp "github.com/zerone-chain/zerone/app"
	claimingpottypes "github.com/zerone-chain/zerone/x/claiming_pot/types"
	zeroneemergencytypes "github.com/zerone-chain/zerone/x/emergency/types"
	zeronegovtypes "github.com/zerone-chain/zerone/x/gov/types"
	knowledgetypes "github.com/zerone-chain/zerone/x/knowledge/types"
	liquiditypooltypes "github.com/zerone-chain/zerone/x/liquiditypool/types"
	substratebridgetypes "github.com/zerone-chain/zerone/x/substrate_bridge/types"
	vestingrewardstypes "github.com/zerone-chain/zerone/x/vesting_rewards/types"
)

// ─── Wave 10: end-to-end upgrade pipeline tests ─────────────────────────

func passExpeditedSDKGovProposal(
	t *testing.T,
	h *TestHarness,
	proposer sdk.AccAddress,
	message sdk.Msg,
) govv1.Proposal {
	t.Helper()
	params, err := h.App.GovKeeper.Params.Get(h.Ctx)
	if err != nil {
		params = govv1.DefaultParams()
	}
	if len(params.MinDeposit) == 0 || len(params.ExpeditedMinDeposit) == 0 {
		params.MinDeposit = sdk.NewCoins(
			sdk.NewInt64Coin(zeroneapp.BondDenom, 1),
		)
		params.ExpeditedMinDeposit = sdk.NewCoins(
			sdk.NewInt64Coin(zeroneapp.BondDenom, 2),
		)
	}
	require.NoError(t, h.App.GovKeeper.Params.Set(h.Ctx, params))
	proposal, err := h.App.GovKeeper.SubmitProposal(
		h.Ctx,
		[]sdk.Msg{message},
		"ipfs://zerone-upgrade-lifecycle-test",
		"Zerone upgrade lifecycle test",
		"exercise submit, deposit, vote, tally, and SDK message execution",
		proposer,
		true,
	)
	require.NoError(t, err)

	params, err = h.App.GovKeeper.Params.Get(h.Ctx)
	require.NoError(t, err)
	deposit := sdk.NewCoins(params.ExpeditedMinDeposit...)
	require.False(t, deposit.Empty())
	require.NoError(t, h.FundAccount(proposer, deposit))
	activated, err := h.App.GovKeeper.AddDeposit(
		h.Ctx,
		proposal.Id,
		proposer,
		deposit,
	)
	require.NoError(t, err)
	require.True(t, activated)
	require.NoError(t, h.App.GovKeeper.AddVote(
		h.Ctx,
		proposal.Id,
		proposer,
		govv1.NewNonSplitVoteOption(govv1.OptionYes),
		"verified release",
	))

	proposal, err = h.App.GovKeeper.Proposals.Get(h.Ctx, proposal.Id)
	require.NoError(t, err)
	require.Equal(t, govv1.StatusVotingPeriod, proposal.Status)
	require.NotNil(t, proposal.VotingEndTime)
	endTime := proposal.VotingEndTime.Add(time.Nanosecond)
	h.currentHeight++
	h.Ctx = h.Ctx.
		WithBlockHeight(h.currentHeight).
		WithBlockTime(endTime).
		WithBlockHeader(cmtproto.Header{
			Height:  h.currentHeight,
			ChainID: testChainID,
			Time:    endTime,
		})
	require.NoError(t, sdkgov.EndBlocker(h.Ctx, h.App.GovKeeper))

	proposal, err = h.App.GovKeeper.Proposals.Get(h.Ctx, proposal.Id)
	require.NoError(t, err)
	require.Equal(t, govv1.StatusPassed, proposal.Status, proposal.FailedReason)
	return proposal
}

func sdkGovLifecycleVoter(t *testing.T, h *TestHarness) sdk.AccAddress {
	t.Helper()
	stakingParams, err := h.App.StakingKeeper.GetParams(h.Ctx)
	if err != nil || stakingParams.BondDenom == "" {
		stakingParams = stakingtypes.DefaultParams()
		stakingParams.BondDenom = zeroneapp.BondDenom
		require.NoError(t, h.App.StakingKeeper.SetParams(h.Ctx, stakingParams))
	}
	voter := sdk.AccAddress(bytes.Repeat([]byte{0x5a}, 20))
	if h.App.AccountKeeper.GetAccount(h.Ctx, voter) == nil {
		h.App.AccountKeeper.SetAccount(
			h.Ctx,
			h.App.AccountKeeper.NewAccountWithAddress(h.Ctx, voter),
		)
	}
	consensusPubKey, err := cryptocodec.FromCmtPubKeyInterface(
		cmted25519.GenPrivKey().PubKey(),
	)
	require.NoError(t, err)
	consensusPubKeyAny, err := codectypes.NewAnyWithValue(consensusPubKey)
	require.NoError(t, err)
	validator := stakingtypes.Validator{
		OperatorAddress: sdk.ValAddress(voter).String(),
		ConsensusPubkey: consensusPubKeyAny,
		Status:          stakingtypes.Bonded,
		Tokens:          sdkmath.NewInt(1_000_000),
		DelegatorShares: sdkmath.LegacyNewDec(1_000_000),
		Commission: stakingtypes.NewCommission(
			sdkmath.LegacyZeroDec(),
			sdkmath.LegacyZeroDec(),
			sdkmath.LegacyZeroDec(),
		),
		MinSelfDelegation: sdkmath.OneInt(),
	}
	require.NoError(t, h.App.StakingKeeper.SetValidator(h.Ctx, validator))
	require.NoError(
		t,
		h.App.StakingKeeper.SetValidatorByConsAddr(h.Ctx, validator),
	)
	require.NoError(
		t,
		h.App.StakingKeeper.SetValidatorByPowerIndex(h.Ctx, validator),
	)
	require.NoError(t, h.App.StakingKeeper.SetDelegation(
		h.Ctx,
		stakingtypes.NewDelegation(
			voter.String(),
			validator.OperatorAddress,
			validator.DelegatorShares,
		),
	))
	bonded := sdk.NewCoins(
		sdk.NewCoin(zeroneapp.BondDenom, validator.Tokens),
	)
	require.NoError(t, h.App.BankKeeper.MintCoins(
		h.Ctx,
		"tokens",
		bonded,
	))
	require.NoError(t, h.App.BankKeeper.SendCoinsFromModuleToModule(
		h.Ctx,
		"tokens",
		stakingtypes.BondedPoolName,
		bonded,
	))
	return voter
}

func activationSafetySourceVM(h *TestHarness) module.VersionMap {
	current := h.App.CurrentModuleVersionMap()
	source := make(module.VersionMap, len(current))
	for name, version := range current {
		source[name] = version
	}
	source[zeroneemergencytypes.ModuleName] = 1
	source[sdkgovtypes.ModuleName] = 5
	source[zeronegovtypes.ModuleName] = 2
	return source
}

func sdk053IBC10SourceVM(h *TestHarness) module.VersionMap {
	source := activationSafetySourceVM(h)
	source["ibc"] = 6
	source["transfer"] = 5
	source["interchainaccounts"] = 3
	source["capability"] = 1
	source["feeibc"] = 2
	return source
}

func seedConsolidationBoundary(t *testing.T, h *TestHarness) {
	t.Helper()
	require.NoError(t, h.KnowledgeKeeper.WriteMigrationMarker(
		h.Ctx,
		"upgrade_marker_consolidation-safety-v1",
		"migrated",
	))
}

func runSDK053IBC10HandlerForTests(
	t *testing.T,
	h *TestHarness,
	fromVM module.VersionMap,
	height int64,
) (module.VersionMap, error) {
	t.Helper()
	seedConsolidationBoundary(t, h)
	planInfo, err := zeroneapp.BuildSDK053IBC10PlanInfo(nil, nil)
	require.NoError(t, err)
	return h.App.RunUpgradeHandlerWithInfoForTests(
		h.Ctx,
		zeroneapp.UpgradeNameSDK053IBC10,
		fromVM,
		height,
		planInfo,
	)
}

func TestStandardSDKGovernanceSchedulesAndExecutesUpgradeAcrossRestart(
	t *testing.T,
) {
	h := NewTestHarness(t)
	startTime := time.Unix(1_900_000_000, 0).UTC()
	h.Ctx = h.Ctx.
		WithBlockTime(startTime).
		WithBlockHeader(cmtproto.Header{
			Height:  h.Height(),
			ChainID: testChainID,
			Time:    startTime,
		})
	require.NoError(t, h.App.UpgradeKeeper.SetModuleVersionMap(
		h.Ctx,
		h.App.CurrentModuleVersionMap(),
	))
	require.NoError(t, h.KnowledgeKeeper.WriteMigrationMarker(
		h.Ctx,
		"chain_lineage_native_sdk-0.53-ibc-10",
		"genesis",
	))
	authority := h.App.AccountKeeper.GetModuleAddress("gov")
	require.NotNil(t, authority)
	voter := sdkGovLifecycleVoter(t, h)
	const upgradeHeight int64 = 8
	plan := upgradetypes.Plan{
		Name:   zeroneapp.UpgradeNameTestnetV2,
		Height: upgradeHeight,
		Info:   `{"release_id":"governance-lifecycle"}`,
	}

	passExpeditedSDKGovProposal(t, h, voter, &upgradetypes.MsgSoftwareUpgrade{
		Authority: authority.String(),
		Plan:      plan,
	})
	scheduled, err := h.App.UpgradeKeeper.GetUpgradePlan(h.Ctx)
	require.NoError(t, err)
	require.Equal(t, plan, scheduled)

	h.currentHeight = upgradeHeight - 1
	h.Ctx = h.Ctx.
		WithBlockHeight(h.currentHeight).
		WithBlockHeader(cmtproto.Header{
			Height:  h.currentHeight,
			ChainID: testChainID,
			Time:    h.Ctx.BlockTime(),
		})
	h.CommitHMinusOne()

	h.currentHeight = upgradeHeight
	upgradeTime := h.Ctx.BlockTime().Add(time.Second)
	h.Ctx = h.App.NewContext(true).
		WithBlockHeight(upgradeHeight).
		WithChainID(testChainID).
		WithBlockTime(upgradeTime).
		WithHeaderInfo(header.Info{
			Height:  upgradeHeight,
			ChainID: testChainID,
			Time:    upgradeTime,
		}).
		WithBlockHeader(cmtproto.Header{
			Height:  upgradeHeight,
			ChainID: testChainID,
			Time:    upgradeTime,
		})
	_, err = h.App.PotPreBlocker(h.Ctx, &abci.RequestFinalizeBlock{
		Height: upgradeHeight,
		Time:   upgradeTime,
	})
	require.NoError(t, err)
	doneHeight, err := h.App.UpgradeKeeper.GetDoneHeight(
		h.Ctx,
		zeroneapp.UpgradeNameTestnetV2,
	)
	require.NoError(t, err)
	require.Equal(t, upgradeHeight, doneHeight)
	require.Equal(
		t,
		"genesis",
		h.KnowledgeKeeper.ReadMigrationMarker(
			h.Ctx,
			"chain_lineage_native_sdk-0.53-ibc-10",
		),
	)
	_, err = h.App.UpgradeKeeper.GetUpgradePlan(h.Ctx)
	require.True(t, errors.Is(err, upgradetypes.ErrNoUpgradePlanFound))

	cache, ok := h.Ctx.MultiStore().(storetypes.CacheMultiStore)
	require.True(t, ok)
	cache.Write()
	h.App.CommitMultiStore().Commit()

	restarted := zeroneapp.NewZeroneApp(
		log.NewNopLogger(),
		h.DB,
		nil,
		false,
		simtestutil.NewAppOptionsWithFlagHome(h.Home),
		baseapp.SetChainID(testChainID),
	)
	require.NoError(t, restarted.LoadLatestVersion())
	require.NoError(t, restarted.ValidateSDK053IBC10StartupCoordination())
	restartTime := upgradeTime.Add(time.Second)
	restartCtx := restarted.NewContext(true).
		WithBlockHeight(upgradeHeight + 1).
		WithChainID(testChainID).
		WithBlockTime(restartTime).
		WithHeaderInfo(header.Info{
			Height:  upgradeHeight + 1,
			ChainID: testChainID,
			Time:    restartTime,
		}).
		WithBlockHeader(cmtproto.Header{
			Height:  upgradeHeight + 1,
			ChainID: testChainID,
			Time:    restartTime,
		})
	doneHeight, err = restarted.UpgradeKeeper.GetDoneHeight(
		restartCtx,
		zeroneapp.UpgradeNameTestnetV2,
	)
	require.NoError(t, err)
	require.Equal(t, upgradeHeight, doneHeight)
	_, err = restarted.PotPreBlocker(
		restartCtx,
		&abci.RequestFinalizeBlock{
			Height: upgradeHeight + 1,
			Time:   restartTime,
		},
	)
	require.NoError(t, err, "restart must not replay the completed upgrade")
}

func TestStandardSDKGovernanceCancellationPreventsExecution(t *testing.T) {
	h := NewTestHarness(t)
	startTime := time.Unix(1_910_000_000, 0).UTC()
	h.Ctx = h.Ctx.
		WithBlockTime(startTime).
		WithBlockHeader(cmtproto.Header{
			Height:  h.Height(),
			ChainID: testChainID,
			Time:    startTime,
		})
	authority := h.App.AccountKeeper.GetModuleAddress("gov")
	require.NotNil(t, authority)
	voter := sdkGovLifecycleVoter(t, h)
	const cancelledHeight int64 = 20

	passExpeditedSDKGovProposal(t, h, voter, &upgradetypes.MsgSoftwareUpgrade{
		Authority: authority.String(),
		Plan: upgradetypes.Plan{
			Name:   zeroneapp.UpgradeNameTestnetV2,
			Height: cancelledHeight,
		},
	})
	_, err := h.App.UpgradeKeeper.GetUpgradePlan(h.Ctx)
	require.NoError(t, err)
	passExpeditedSDKGovProposal(t, h, voter, &upgradetypes.MsgCancelUpgrade{
		Authority: authority.String(),
	})
	_, err = h.App.UpgradeKeeper.GetUpgradePlan(h.Ctx)
	require.True(t, errors.Is(err, upgradetypes.ErrNoUpgradePlanFound))

	h.currentHeight = cancelledHeight - 1
	h.Ctx = h.Ctx.WithBlockHeight(h.currentHeight)
	h.CommitHMinusOne()
	h.currentHeight = cancelledHeight
	cancelledTime := h.Ctx.BlockTime().Add(time.Second)
	h.Ctx = h.App.NewContext(true).
		WithBlockHeight(cancelledHeight).
		WithChainID(testChainID).
		WithBlockTime(cancelledTime).
		WithHeaderInfo(header.Info{
			Height:  cancelledHeight,
			ChainID: testChainID,
			Time:    cancelledTime,
		}).
		WithBlockHeader(cmtproto.Header{
			Height:  cancelledHeight,
			ChainID: testChainID,
			Time:    cancelledTime,
		})
	_, err = h.App.PotPreBlocker(h.Ctx, &abci.RequestFinalizeBlock{
		Height: cancelledHeight,
		Time:   cancelledTime,
	})
	require.NoError(t, err)
	doneHeight, err := h.App.UpgradeKeeper.GetDoneHeight(
		h.Ctx,
		zeroneapp.UpgradeNameTestnetV2,
	)
	require.NoError(t, err)
	require.Zero(t, doneHeight, "cancelled plan must not execute")
}

func TestStandardSDKGovernanceMustCancelBeforeReplacingScheduledUpgrade(
	t *testing.T,
) {
	h := NewTestHarness(t)
	authority := h.App.AccountKeeper.GetModuleAddress("gov")
	require.NotNil(t, authority)

	softwareUpgradeHandler := h.App.MsgServiceRouter().Handler(
		&upgradetypes.MsgSoftwareUpgrade{},
	)
	cancelUpgradeHandler := h.App.MsgServiceRouter().Handler(
		&upgradetypes.MsgCancelUpgrade{},
	)
	require.NotNil(t, softwareUpgradeHandler)
	require.NotNil(t, cancelUpgradeHandler)

	first := &upgradetypes.MsgSoftwareUpgrade{
		Authority: authority.String(),
		Plan: upgradetypes.Plan{
			Name:   "canonical-upgrade",
			Height: h.Height() + 100,
			Info:   `{"release_id":"release-a"}`,
		},
	}
	_, err := softwareUpgradeHandler(h.Ctx, first)
	require.NoError(t, err)

	replacement := &upgradetypes.MsgSoftwareUpgrade{
		Authority: authority.String(),
		Plan: upgradetypes.Plan{
			Name:   "hostile-replacement",
			Height: h.Height() + 101,
			Info:   `{"release_id":"release-b"}`,
		},
	}
	_, err = softwareUpgradeHandler(h.Ctx, replacement)
	require.Error(t, err)
	require.Contains(t, err.Error(), "refusing to overwrite scheduled SDK upgrade")
	require.Contains(t, err.Error(), "must pass MsgCancelUpgrade")

	persisted, err := h.App.UpgradeKeeper.GetUpgradePlan(h.Ctx)
	require.NoError(t, err)
	require.Equal(t, first.Plan, persisted)

	_, err = cancelUpgradeHandler(h.Ctx, &upgradetypes.MsgCancelUpgrade{
		Authority: authority.String(),
	})
	require.NoError(t, err)
	_, err = softwareUpgradeHandler(h.Ctx, replacement)
	require.NoError(t, err)

	persisted, err = h.App.UpgradeKeeper.GetUpgradePlan(h.Ctx)
	require.NoError(t, err)
	require.Equal(t, replacement.Plan, persisted)
}

// TestUpgrade_ChainVersionReportWellFormed — the introspection surface
// returns a sorted, complete module list plus the registered upgrade
// lineage. First guard: drift between the registered handlers and the
// self-described lineage would mask a missing upgrade, so we assert
// parity.
func TestUpgrade_ChainVersionReportWellFormed(t *testing.T) {
	h := NewTestHarness(t)

	report := h.App.BuildChainVersionReport()
	require.NotEmpty(t, report.Modules, "chain has registered modules")
	require.NotEmpty(t, report.KnownUpgrades, "at least one upgrade lineage entry")

	// Modules sorted by name deterministically.
	for i := 1; i < len(report.Modules); i++ {
		require.LessOrEqual(t, report.Modules[i-1].ModuleName, report.Modules[i].ModuleName,
			"module list must be name-sorted for deterministic consumption")
	}

	// Every advertised upgrade must have a handler registered. No drift.
	names := h.App.KnownUpgradeNames()
	for _, n := range names {
		require.True(t, h.App.UpgradeKeeper.HasHandler(n),
			"handler for %q must be registered to match the lineage entry", n)
	}

	// The report binds the current consensus releases for modules whose named
	// activation boundaries are pending.
	var sawKnowledge, sawLiquidityPool, sawVestingRewards bool
	for _, m := range report.Modules {
		switch m.ModuleName {
		case "knowledge":
			sawKnowledge = true
			require.Equal(t, uint64(6), m.ConsensusVersion,
				"knowledge module advertises its current ConsensusVersion")
		case liquiditypooltypes.ModuleName:
			sawLiquidityPool = true
			require.Equal(t, uint64(5), m.ConsensusVersion,
				"liquiditypool module advertises the LP-only fee ConsensusVersion")
		case vestingrewardstypes.ModuleName:
			sawVestingRewards = true
			require.Equal(t, uint64(2), m.ConsensusVersion,
				"vesting_rewards advertises the retired automatic-tap ConsensusVersion")
		}
	}
	require.True(t, sawKnowledge, "knowledge module appears in report")
	require.True(t, sawLiquidityPool, "liquiditypool module appears in report")
	require.True(t, sawVestingRewards, "vesting_rewards module appears in report")
}

// TestUpgrade_V1ToV2MigrationPipeline — exercise the v1.0.1-testnet upgrade
// against the knowledge module downshifted from its current ConsensusVersion
// to v1. Exercises the full pipeline Migrate1to2 → Migrate2to3 → Migrate3to4
// in sequence. Other modules stay at their current version (no migration
// runs for them — we're not testing SDK-module migrations here, which have
// their own Cosmos SDK test coverage and require test fixtures this harness
// doesn't set up).
func TestUpgrade_V1ToV2MigrationPipeline(t *testing.T) {
	h := NewTestHarness(t)

	// Build fromVM: all modules at current, knowledge downshifted to v1.
	current := h.App.CurrentModuleVersionMap()
	fromVM := make(module.VersionMap, len(current))
	for name, ver := range current {
		fromVM[name] = ver
	}
	fromVM["knowledge"] = 1

	toVM, err := h.App.RunUpgradeHandlerForTests(h.Ctx, zeroneapp.UpgradeNameTestnetV2, fromVM, h.Height())
	require.NoError(t, err, "v1.0.1-testnet handler completes without error")
	require.Equal(t, uint64(6), toVM["knowledge"],
		"knowledge module advances to its current ConsensusVersion (6) via full migration chain")

	// All migrations ran in sequence — each wrote its marker.
	require.Equal(t, "true", h.KnowledgeKeeper.ReadMigrationMarker(h.Ctx, "migration_v2_complete"),
		"v1→v2 migration marker proves Migrate1to2 ran")
	require.Equal(t, "true", h.KnowledgeKeeper.ReadMigrationMarker(h.Ctx, "migration_v4_complete"),
		"v3→v4 migration marker proves Migrate3to4 ran mid-chain")
	require.Equal(t, "true", h.KnowledgeKeeper.ReadMigrationMarker(h.Ctx, "migration_v5_complete"),
		"v4→v5 migration marker proves Migrate4to5 ran")
	require.Equal(t, "true", h.KnowledgeKeeper.ReadMigrationMarker(h.Ctx, "migration_v6_complete"),
		"v5→v6 migration marker proves Migrate5to6 ran at the end of the chain")

	// The v1.0.1 handler-level marker was written by the upgrade handler itself.
	handlerMarker := h.KnowledgeKeeper.ReadMigrationMarker(h.Ctx, "upgrade_marker_v1.0.1")
	require.Equal(t, "migrated", handlerMarker,
		"handler-level marker proves the named upgrade handler executed")
}

// TestUpgrade_V3ToV4KnowledgeMigrationPipeline — exercise the
// v1.0.2-testnet upgrade, which in turn fires the knowledge v3→v4
// migration (TraceSchema backfill + v4 marker).
func TestUpgrade_V3ToV4KnowledgeMigrationPipeline(t *testing.T) {
	h := NewTestHarness(t)

	// Synthetic fromVM: knowledge at v3, everything else at current.
	current := h.App.CurrentModuleVersionMap()
	fromVM := make(module.VersionMap, len(current))
	for name, ver := range current {
		fromVM[name] = ver
	}
	fromVM["knowledge"] = 3 // downshift so v3→v4 migration fires

	// NO pre-seeded TraceSchema — the v4 migration must backfill it.
	_, seeded := h.KnowledgeKeeper.GetTraceSchema(h.Ctx)
	if seeded {
		t.Log("trace schema already present pre-upgrade; v4 migration will be a no-op on schema")
	}

	toVM, err := h.App.RunUpgradeHandlerForTests(h.Ctx, zeroneapp.UpgradeNameTestnetV3, fromVM, h.Height())
	require.NoError(t, err)
	require.Equal(t, uint64(6), toVM["knowledge"],
		"knowledge module is now at ConsensusVersion 6 post-migration (chain now extends 3→4→5→6)")

	// v4 + v5 migration markers prove Migrate3to4 and Migrate4to5 both ran.
	require.Equal(t, "true", h.KnowledgeKeeper.ReadMigrationMarker(h.Ctx, "migration_v4_complete"),
		"v4 migration marker present")
	require.Equal(t, "true", h.KnowledgeKeeper.ReadMigrationMarker(h.Ctx, "migration_v5_complete"),
		"v5 migration marker present")
	require.Equal(t, "true", h.KnowledgeKeeper.ReadMigrationMarker(h.Ctx, "migration_v6_complete"),
		"v6 migration marker present")

	// TraceSchema is present post-upgrade.
	schema, ok := h.KnowledgeKeeper.GetTraceSchema(h.Ctx)
	require.True(t, ok, "TraceSchema is backfilled by v4 migration")
	require.Equal(t, uint64(1), schema.Version)

	// v1.0.2 handler-level marker.
	require.Equal(t, "migrated",
		h.KnowledgeKeeper.ReadMigrationMarker(h.Ctx, "upgrade_marker_v1.0.2"))
}

// TestUpgrade_UnknownHandlerRejected — calling an unregistered upgrade
// name must return a clear error, not silently succeed or panic.
func TestUpgrade_UnknownHandlerRejected(t *testing.T) {
	h := NewTestHarness(t)
	_, err := h.App.RunUpgradeHandlerForTests(h.Ctx, "not-a-real-upgrade", module.VersionMap{}, h.Height())
	require.Error(t, err)
	require.Contains(t, err.Error(), "no upgrade handler registered")
}

// TestUpgrade_MigrationMarkerIdempotent — writing the same marker twice
// is a no-op (idempotent); writing a DIFFERENT value for the same key is
// rejected without overwriting (first writer wins).
// TestUpgrade_CompassionCalibrationV1RefreshesScores drives the real
// compassion-calibration-v1 handler and asserts it refreshes a stored
// calibration score under the inconclusive-excluding formula (docs/COMPASSION.md
// commitment C2). A record whose honest inconclusive attempts had dragged its
// stored score down is lifted to its true decisive-accuracy score, and the
// migration marker is written.
func TestUpgrade_CompassionCalibrationV1RefreshesScores(t *testing.T) {
	h := NewTestHarness(t)

	addr := "zerone1compassionupgrade00000000000000aa"
	// 3 accepted + 7 inconclusive. Under the OLD formula the stored score was
	// 3/10 = 300_000; seed that stale value so we can prove the handler recomputes.
	require.NoError(t, h.KnowledgeKeeper.SetAgentCalibration(h.Ctx, &knowledgetypes.AgentCalibration{
		Address:             addr,
		TotalSubmissions:    10,
		Accepted:            3,
		Inconclusive:        7,
		CalibrationScoreBps: 300_000, // stale, old-formula value
	}))

	// Run the real upgrade handler through the full pipeline.
	fromVM := h.App.CurrentModuleVersionMap()
	_, err := h.App.RunUpgradeHandlerForTests(h.Ctx, zeroneapp.UpgradeNameCompassionCalibrationV1, fromVM, h.Height())
	require.NoError(t, err)

	// The 7 inconclusive attempts leave the denominator: 3/3 decisive = BPS.
	refreshed, found := h.KnowledgeKeeper.GetAgentCalibration(h.Ctx, addr)
	require.True(t, found)
	require.Equal(t, uint64(1_000_000), refreshed.CalibrationScoreBps,
		"handler must recompute the stale score under the inconclusive-excluding formula")

	require.Equal(t, "migrated",
		h.KnowledgeKeeper.ReadMigrationMarker(h.Ctx, "upgrade_marker_compassion-calibration-v1"),
		"handler must write the compassion-calibration-v1 migration marker")
}

func TestUpgrade_MigrationMarkerIdempotent(t *testing.T) {
	h := NewTestHarness(t)
	require.NoError(t, h.KnowledgeKeeper.WriteMigrationMarker(h.Ctx, "test_marker", "alpha"))
	require.Equal(t, "alpha", h.KnowledgeKeeper.ReadMigrationMarker(h.Ctx, "test_marker"))

	// Same value again — idempotent.
	require.NoError(t, h.KnowledgeKeeper.WriteMigrationMarker(h.Ctx, "test_marker", "alpha"))
	require.Equal(t, "alpha", h.KnowledgeKeeper.ReadMigrationMarker(h.Ctx, "test_marker"))

	// Different value — preserves original and fails closed.
	require.Error(t, h.KnowledgeKeeper.WriteMigrationMarker(h.Ctx, "test_marker", "beta"))
	require.Equal(t, "alpha", h.KnowledgeKeeper.ReadMigrationMarker(h.Ctx, "test_marker"),
		"first writer wins: conflicting value does not overwrite")
}

// TestUpgrade_LineageParityWithHandlers — every entry in the lineage list
// has a registered handler, AND every registered handler appears in the
// lineage list. Parity prevents the chain from drifting into a state
// where a handler exists but is invisible to operators, or vice versa.
func TestUpgrade_LineageParityWithHandlers(t *testing.T) {
	h := NewTestHarness(t)
	lineage := h.App.BuildChainVersionReport().KnownUpgrades

	// Every advertised upgrade must have a handler.
	for _, entry := range lineage {
		require.True(t, h.App.UpgradeKeeper.HasHandler(entry.UpgradeName),
			"lineage entry %q advertises a handler; must be registered", entry.UpgradeName)
	}

	// Inverse — every UpgradeName constant known to app should be listed
	// in the lineage. Hard-coded check against the three we ship.
	lineageNames := h.App.KnownUpgradeNames()
	require.Contains(t, lineageNames, zeroneapp.UpgradeNameTestnet)
	require.Contains(t, lineageNames, zeroneapp.UpgradeNameTestnetV2)
	require.Contains(t, lineageNames, zeroneapp.UpgradeNameTestnetV3)
	require.Contains(t, lineageNames, zeroneapp.UpgradeNameDoctrineMetabolismExemptV1)
	require.Contains(t, lineageNames, zeroneapp.UpgradeNameSubstrateDedupeV1)
	require.Contains(t, lineageNames, zeroneapp.UpgradeNameConsolidationSafetyV1)
	require.Contains(t, lineageNames, zeroneapp.UpgradeNameSDK053IBC10)
	require.NotContains(t, lineageNames, "auth-ante-hardening-v1")
	require.False(t, h.App.UpgradeKeeper.HasHandler("auth-ante-hardening-v1"),
		"the retired auth-only plan must not bypass the SDK/IBC guards")
	require.NotContains(t, lineageNames, "upgrade-incident-operations-v1")
	require.False(t, h.App.UpgradeKeeper.HasHandler("upgrade-incident-operations-v1"),
		"operations safety is bundled into the singleton SDK/IBC handler")
	require.NotContains(t, lineageNames, "liquiditypool-safety-v2",
		"the atomic H1 binary must not advertise a redundant future liquidity handler")
	require.False(t, h.App.UpgradeKeeper.HasHandler("liquiditypool-safety-v2"),
		"registering the retired H2 handler would trigger x/upgrade's early-binary guard")

	var consolidationIndex, sdkIndex = -1, -1
	for i, name := range lineageNames {
		switch name {
		case zeroneapp.UpgradeNameConsolidationSafetyV1:
			consolidationIndex = i
		case zeroneapp.UpgradeNameSDK053IBC10:
			sdkIndex = i
		}
	}
	require.NotEqual(t, -1, consolidationIndex)
	require.NotEqual(t, -1, sdkIndex)
	require.Less(t, consolidationIndex, sdkIndex,
		"the coordinated SDK/IBC activation must follow the pending consolidation boundary")
}

func TestValidatorCosmovisorEntrypointPinsAndAtomicallyStagesBinaries(t *testing.T) {
	entrypoint, err := os.ReadFile("../../deploy/validator-cosmovisor-entrypoint.sh")
	require.NoError(t, err)
	body := string(entrypoint)
	require.Contains(t, body, "DAEMON_BINARY_SHA256")
	require.Contains(t, body, "DAEMON_GENESIS_BINARY_SHA256")
	require.Contains(t, body, "DAEMON_CURRENT_BINARY_SHA256")
	require.Contains(t, body, "DAEMON_CURRENT_UPGRADE_NAME")
	require.Contains(t, body, `source_digest="$(sha256sum "${source_binary}"`)
	require.Contains(t, body, `if [ -L "${destination}" ]; then`)
	require.Contains(t, body, `temporary="$(mktemp "${destination_dir}/.${DAEMON_NAME}.tmp.XXXXXX")"`)
	require.Contains(t, body, `mv -f "${temporary}" "${destination}"`)
	require.Contains(t, body, `upgrade_binary="${upgrade_dir}/bin/${DAEMON_NAME}"`)
	require.Contains(t, body, "refusing artifact replacement")
	require.Contains(t, body, "populated application data has no pinned Cosmovisor genesis binary")
	require.Contains(t, body, `current_target="$(readlink "${current_link}")"`)
	require.Contains(t, body, "current selector may be absent only while genesis is the authorized predecessor")
	require.Contains(t, body, `verify_binary_digest "${final_binary}" "${final_digest}"`)
	require.Contains(t, body, "canonical lowercase ASCII")
	require.Contains(t, body, "DAEMON_ALLOW_DOWNLOAD_BINARIES false")
	require.Contains(t, body, "DAEMON_DOWNLOAD_MUST_HAVE_CHECKSUM true")
	require.Contains(t, body, "UNSAFE_SKIP_BACKUP false")
	require.Contains(t, body, "runtime command is image-frozen")
	require.Contains(t, body, "Cosmovisor must be an executable regular non-symlink")
	require.Less(
		t,
		strings.Index(body, `source_digest="$(sha256sum "${source_binary}"`),
		strings.Index(body, `mv -f "${temporary}" "${destination}"`),
	)
	readme, err := os.ReadFile("../../cosmovisor/README.md")
	require.NoError(t, err)
	require.Contains(t, string(readme), "never a symlink")
	require.False(
		t,
		strings.Contains(string(readme), "copy or symlink"),
		"operator guide must not permit a mutable genesis binary symlink",
	)
}

func flyValidatorKeyJSON(secret string) []byte {
	privateKey := cmted25519.GenPrivKeyFromSecret([]byte(secret))
	publicKey := privateKey.PubKey().(cmted25519.PubKey)
	return []byte(fmt.Sprintf(
		`{"address":"%X","pub_key":{"type":"tendermint/PubKeyEd25519","value":"%s"},"priv_key":{"type":"tendermint/PrivKeyEd25519","value":"%s"}}`,
		[]byte(publicKey.Address()),
		base64.StdEncoding.EncodeToString(publicKey),
		base64.StdEncoding.EncodeToString(privateKey),
	))
}

func flyNodeKeyJSON(secret string) []byte {
	privateKey := cmted25519.GenPrivKeyFromSecret([]byte(secret))
	return []byte(fmt.Sprintf(
		`{"priv_key":{"type":"tendermint/PrivKeyEd25519","value":"%s"}}`,
		base64.StdEncoding.EncodeToString(privateKey),
	))
}

func TestFlyValidatorProfilesPinRuntimeKeysAndDoNotPublishAPIs(t *testing.T) {
	commonEntrypoint, err := os.ReadFile(
		"../../deploy/fly-validator-entrypoint-common.sh",
	)
	require.NoError(t, err)
	for _, network := range []string{"mainnet", "testnet"} {
		t.Run(network, func(t *testing.T) {
			entrypoint, err := os.ReadFile(
				"../../deploy/" + network + "/entrypoint.containment.sh",
			)
			require.NoError(t, err)
			body := string(entrypoint) + "\n" + string(commonEntrypoint)
			require.Contains(t, body, "PRIV_VALIDATOR_KEY_SHA256")
			require.Contains(t, body, "NODE_KEY_SHA256")
			require.Contains(t, body, "EXPECTED_VALIDATOR_ADDRESS")
			require.Contains(t, body, "EXPECTED_NODE_ID")
			require.Contains(t, body, "EXPECTED_PRIV_VALIDATOR_KEY_SHA256")
			require.Contains(t, body, "EXPECTED_NODE_KEY_SHA256")
			require.Contains(t, body, "derived P2P node ID does not match")
			require.Contains(t, body, "ZERONE_GENESIS_SHA256")
			require.Contains(t, body, "image-frozen genesis SHA-256")
			require.Contains(t, body, `actual_digest="$(sha256sum "${temporary}"`)
			require.Contains(t, body, "refusing to replace persisted identity")
			require.Contains(t, body, "validator address does not match")
			require.Contains(t, body, "public suffix does not match its seed")
			require.Contains(t, body, "persisted validator signing state")
			require.Contains(t, body, "unset PRIV_VALIDATOR_KEY_FILE")
			require.Less(
				t,
				strings.Index(body, `actual_digest="$(sha256sum "${temporary}"`),
				strings.Index(body, `mv -f "${temporary}" "${destination}"`),
				"key digest must be verified before the atomic replacement",
			)
			require.NotContains(t, body, `tcp://0.0.0.0:26657`)
			require.NotContains(t, body, `tcp://0.0.0.0:1317`)
			require.NotContains(t, body, `0.0.0.0:9090`)

			flyConfig, err := os.ReadFile(
				"../../deploy/" + network + "/fly.toml",
			)
			require.NoError(t, err)
			config := string(flyConfig)
			require.NotContains(t, config, "internal_port = 26657")
			require.NotContains(t, config, "internal_port = 1317")
			require.NotContains(t, config, "internal_port = 9090")
			require.Contains(t, config, "internal_port = 26656")
		})
	}
}

func TestFlyEntrypointPreservesPersistedSignerOnDigestMismatch(t *testing.T) {
	for _, network := range []string{"mainnet", "testnet"} {
		t.Run(network, func(t *testing.T) {
			home := t.TempDir()
			configDir := filepath.Join(home, "config")
			require.NoError(t, os.MkdirAll(configDir, 0o700))
			genesis, err := os.ReadFile(
				"../../deploy/" + network + "/artifacts/genesis.json",
			)
			require.NoError(t, err)
			require.NoError(
				t,
				os.WriteFile(
					filepath.Join(configDir, "genesis.json"),
					genesis,
					0o600,
				),
			)
			persisted := []byte("persisted-signer-must-survive")
			destination := filepath.Join(configDir, "priv_validator_key.json")
			require.NoError(t, os.WriteFile(destination, persisted, 0o600))

			candidateBytes := flyValidatorKeyJSON("digest-mismatch-candidate")
			candidate := filepath.Join(home, "candidate.json")
			require.NoError(
				t,
				os.WriteFile(
					candidate,
					candidateBytes,
					0o600,
				),
			)
			nodeBytes := flyNodeKeyJSON("digest-mismatch-node")
			nodeCandidate := filepath.Join(home, "node-candidate.json")
			require.NoError(t, os.WriteFile(nodeCandidate, nodeBytes, 0o600))
			nodeDigest := sha256.Sum256(nodeBytes)
			validatorPublic := cmted25519.GenPrivKeyFromSecret(
				[]byte("digest-mismatch-candidate"),
			).PubKey()
			nodePublic := cmted25519.GenPrivKeyFromSecret(
				[]byte("digest-mismatch-node"),
			).PubKey()

			command := exec.Command(
				"bash",
				"../../deploy/"+network+"/entrypoint.containment.sh",
			)
			command.Env = append(
				os.Environ(),
				"ZERONE_HOME="+home,
				"PRIV_VALIDATOR_KEY_FILE="+candidate,
				"PRIV_VALIDATOR_KEY_SHA256="+strings.Repeat("0", 64),
				"NODE_KEY_FILE="+nodeCandidate,
				"NODE_KEY_SHA256="+fmt.Sprintf("%x", nodeDigest),
				"EXPECTED_VALIDATOR_ADDRESS="+fmt.Sprintf(
					"%X",
					[]byte(validatorPublic.Address()),
				),
				"EXPECTED_NODE_ID="+fmt.Sprintf("%x", []byte(nodePublic.Address())),
				"EXPECTED_PRIV_VALIDATOR_KEY_SHA256="+strings.Repeat("0", 64),
				"EXPECTED_NODE_KEY_SHA256="+fmt.Sprintf("%x", nodeDigest),
			)
			output, err := command.CombinedOutput()
			require.Error(t, err)
			require.Contains(t, string(output), "SHA-256 mismatch")
			after, readErr := os.ReadFile(destination)
			require.NoError(t, readErr)
			require.Equal(t, persisted, after)
		})
	}
}

func TestFlyEntrypointRefusesDigestValidPersistedSignerDrift(t *testing.T) {
	for _, network := range []string{"mainnet", "testnet"} {
		t.Run(network, func(t *testing.T) {
			home := t.TempDir()
			configDir := filepath.Join(home, "config")
			dataDir := filepath.Join(home, "data")
			require.NoError(t, os.MkdirAll(configDir, 0o700))
			require.NoError(t, os.MkdirAll(dataDir, 0o700))
			genesis, err := os.ReadFile(
				"../../deploy/" + network + "/artifacts/genesis.json",
			)
			require.NoError(t, err)
			require.NoError(
				t,
				os.WriteFile(
					filepath.Join(configDir, "genesis.json"),
					genesis,
					0o600,
				),
			)
			require.NoError(t, os.WriteFile(
				filepath.Join(configDir, "config.toml"),
				[]byte{},
				0o600,
			))
			require.NoError(t, os.WriteFile(
				filepath.Join(configDir, "app.toml"),
				[]byte{},
				0o600,
			))
			persisted := flyValidatorKeyJSON("persisted-validator")
			destination := filepath.Join(configDir, "priv_validator_key.json")
			require.NoError(t, os.WriteFile(destination, persisted, 0o600))
			signState := []byte(`{"height":"77","round":0,"step":3}`)
			statePath := filepath.Join(dataDir, "priv_validator_state.json")
			require.NoError(t, os.WriteFile(statePath, signState, 0o600))
			nodeBytes := flyNodeKeyJSON("persisted-node")
			nodeDestination := filepath.Join(configDir, "node_key.json")
			require.NoError(t, os.WriteFile(nodeDestination, nodeBytes, 0o600))

			candidateBytes := flyValidatorKeyJSON("replacement-validator")
			candidate := filepath.Join(home, "candidate.json")
			require.NoError(t, os.WriteFile(candidate, candidateBytes, 0o600))
			candidateDigest := sha256.Sum256(candidateBytes)
			nodeCandidate := filepath.Join(home, "node-candidate.json")
			require.NoError(t, os.WriteFile(nodeCandidate, nodeBytes, 0o600))
			nodeDigest := sha256.Sum256(nodeBytes)
			validatorPublic := cmted25519.GenPrivKeyFromSecret(
				[]byte("replacement-validator"),
			).PubKey()
			nodePublic := cmted25519.GenPrivKeyFromSecret(
				[]byte("persisted-node"),
			).PubKey()

			command := exec.Command(
				"bash",
				"../../deploy/"+network+"/entrypoint.containment.sh",
			)
			command.Env = append(
				os.Environ(),
				"ZERONE_HOME="+home,
				"PRIV_VALIDATOR_KEY_FILE="+candidate,
				"PRIV_VALIDATOR_KEY_SHA256="+fmt.Sprintf(
					"%x",
					candidateDigest,
				),
				"NODE_KEY_FILE="+nodeCandidate,
				"NODE_KEY_SHA256="+fmt.Sprintf("%x", nodeDigest),
				"EXPECTED_VALIDATOR_ADDRESS="+fmt.Sprintf(
					"%X",
					[]byte(validatorPublic.Address()),
				),
				"EXPECTED_NODE_ID="+fmt.Sprintf("%x", []byte(nodePublic.Address())),
				"EXPECTED_PRIV_VALIDATOR_KEY_SHA256="+fmt.Sprintf(
					"%x",
					candidateDigest,
				),
				"EXPECTED_NODE_KEY_SHA256="+fmt.Sprintf("%x", nodeDigest),
			)
			output, err := command.CombinedOutput()
			require.Error(t, err)
			require.Contains(t, string(output), "reviewed identity manifest")

			afterKey, readErr := os.ReadFile(destination)
			require.NoError(t, readErr)
			require.Equal(t, persisted, afterKey)
			afterState, readErr := os.ReadFile(statePath)
			require.NoError(t, readErr)
			require.Equal(t, signState, afterState)
		})
	}
}

func TestUpgrade_SDK053IBC10RunsIBCStateMigrations(t *testing.T) {
	h := NewTestHarness(t)
	seedConsolidationBoundary(t, h)
	h.GovKeeper.SetLIP(h.Ctx, &zeronegovtypes.LIP{
		Id:           "LIP-sdk-plan-legacy-upgrade",
		Title:        "legacy custom upgrade",
		Description:  "must retire inside the singleton SDK activation",
		Category:     zeronegovtypes.CategoryUpgrade,
		Stage:        zeronegovtypes.StatusReview,
		StakedAmount: "0",
	})
	h.EmergencyKeeper.SetEmergencyStatus(
		h.Ctx,
		zeroneemergencytypes.StatusHaltVoting,
	)
	for _, id := range []string{"sdk-legacy-active-a", "sdk-legacy-active-b"} {
		require.NoError(t, h.EmergencyKeeper.SetCeremony(
			h.Ctx,
			&zeroneemergencytypes.EmergencyCeremony{
				Id:    id,
				Type:  string(zeroneemergencytypes.CeremonyHalt),
				Phase: string(zeroneemergencytypes.PhasePrevote),
			},
		))
	}
	h.CommitHMinusOne()

	current := activationSafetySourceVM(h)
	fromVM := make(module.VersionMap, len(current)+2)
	for name, version := range current {
		fromVM[name] = version
	}
	fromVM["ibc"] = 6
	fromVM["transfer"] = 5
	fromVM["interchainaccounts"] = 3
	fromVM["capability"] = 1
	fromVM["feeibc"] = 2

	planInfo, err := zeroneapp.BuildSDK053IBC10PlanInfo(nil, nil)
	require.NoError(t, err)
	toVM, err := h.App.RunUpgradeHandlerWithInfoForTests(
		h.Ctx,
		zeroneapp.UpgradeNameSDK053IBC10,
		fromVM,
		h.Height(),
		planInfo,
	)
	require.NoError(t, err)
	require.Equal(t, uint64(8), toVM["ibc"], "IBC core must run v6→v7→v8 migrations")
	require.Equal(t, uint64(6), toVM["transfer"], "ICS-20 must run its v5→v6 denom migration")
	require.Equal(t, uint64(3), toVM["interchainaccounts"])
	require.NotContains(t, toVM, "capability")
	require.NotContains(t, toVM, "feeibc")
	require.Equal(
		t,
		"migrated",
		h.KnowledgeKeeper.ReadMigrationMarker(h.Ctx, "upgrade_marker_auth-ante-hardening-v1"),
		"the unified guarded plan must activate and mark signer-policy hardening",
	)
	require.Equal(
		t,
		"migrated-with-loader-proof-v1",
		h.KnowledgeKeeper.ReadMigrationMarker(h.Ctx, "upgrade_marker_sdk-0.53-ibc-10"),
	)
	require.Equal(
		t,
		"migrated",
		h.KnowledgeKeeper.ReadMigrationMarker(h.Ctx, "upgrade_marker_upgrade-incident-operations-v1"),
	)
	require.Equal(t, zeroneemergencytypes.StatusNormal, h.EmergencyKeeper.GetEmergencyStatus(h.Ctx))
	_, active := h.EmergencyKeeper.GetActiveCeremony(h.Ctx)
	require.False(t, active)
	legacyUpgrade, found := h.GovKeeper.GetLIP(h.Ctx, "LIP-sdk-plan-legacy-upgrade")
	require.True(t, found)
	require.Equal(t, zeronegovtypes.StatusFailed, legacyUpgrade.Stage)

	// Both the compiled module manager and persisted x/upgrade map exclude the
	// retired modules. The handler explicitly deletes those merge-only version
	// keys so post-H startup can detect unsafe-skip aftermath.
	targetVM := h.App.CurrentModuleVersionMap()
	require.NotContains(t, targetVM, "capability")
	require.NotContains(t, targetVM, "feeibc")
}

func TestUpgrade_SDK053IBC10RefusesMissingKeysetManifestBeforeAuthMarker(t *testing.T) {
	h := NewTestHarness(t)
	seedConsolidationBoundary(t, h)

	current := activationSafetySourceVM(h)
	fromVM := make(module.VersionMap, len(current)+2)
	for name, version := range current {
		fromVM[name] = version
	}
	fromVM["ibc"] = 6
	fromVM["transfer"] = 5
	fromVM["interchainaccounts"] = 3
	fromVM["capability"] = 1
	fromVM["feeibc"] = 2

	_, err := h.App.RunUpgradeHandlerWithInfoForTests(
		h.Ctx,
		zeroneapp.UpgradeNameSDK053IBC10,
		fromVM,
		h.Height(),
		"",
	)
	require.Error(t, err)
	require.Contains(t, err.Error(), "missing mandatory legacy IBC keyset manifest")
	require.Empty(
		t,
		h.KnowledgeKeeper.ReadMigrationMarker(h.Ctx, "upgrade_marker_auth-ante-hardening-v1"),
	)
}

func TestUpgrade_SDK053IBC10RefusesUnexpectedSourceVersionBeforeAuthMarker(t *testing.T) {
	h := NewTestHarness(t)
	seedConsolidationBoundary(t, h)

	current := activationSafetySourceVM(h)
	fromVM := make(module.VersionMap, len(current)+2)
	for name, version := range current {
		fromVM[name] = version
	}
	fromVM["ibc"] = 6
	fromVM["transfer"] = 5
	fromVM["interchainaccounts"] = 3
	fromVM["capability"] = 1
	fromVM["feeibc"] = 1 // legacy ICS-29 shipped consensus version 2

	planInfo, err := zeroneapp.BuildSDK053IBC10PlanInfo(nil, nil)
	require.NoError(t, err)
	_, err = h.App.RunUpgradeHandlerWithInfoForTests(
		h.Ctx,
		zeroneapp.UpgradeNameSDK053IBC10,
		fromVM,
		h.Height(),
		planInfo,
	)
	require.Error(t, err)
	require.Contains(t, err.Error(), `requires source module "feeibc" at consensus version 2: got 1`)
	require.Empty(
		t,
		h.KnowledgeKeeper.ReadMigrationMarker(h.Ctx, "upgrade_marker_auth-ante-hardening-v1"),
		"the auth marker must remain unwritten when an IBC source-version guard fails",
	)
}

func TestSDK053IBC10RefusesMissingOrWrongConsolidationBoundaryBeforeMutation(
	t *testing.T,
) {
	planInfo, err := zeroneapp.BuildSDK053IBC10PlanInfo(nil, nil)
	require.NoError(t, err)
	prerequisites := []struct {
		name    string
		version uint64
	}{
		{knowledgetypes.ModuleName, 6},
		{claimingpottypes.ModuleName, 2},
		{liquiditypooltypes.ModuleName, 5},
		{vestingrewardstypes.ModuleName, 2},
	}

	for _, prerequisite := range prerequisites {
		for _, corruption := range []string{"missing", "wrong"} {
			t.Run(prerequisite.name+"/"+corruption, func(t *testing.T) {
				h := NewTestHarness(t)
				seedConsolidationBoundary(t, h)
				fromVM := sdk053IBC10SourceVM(h)
				if corruption == "missing" {
					delete(fromVM, prerequisite.name)
				} else {
					fromVM[prerequisite.name] = prerequisite.version - 1
				}

				_, err := h.App.RunUpgradeHandlerWithInfoForTests(
					h.Ctx,
					zeroneapp.UpgradeNameSDK053IBC10,
					fromVM,
					h.Height(),
					planInfo,
				)
				require.Error(t, err)
				require.Contains(t, err.Error(),
					`requires prerequisite module "`+prerequisite.name+`"`)
				require.Empty(t, h.KnowledgeKeeper.ReadMigrationMarker(
					h.Ctx,
					"upgrade_marker_sdk-0.53-ibc-10",
				))
				require.Empty(t, h.KnowledgeKeeper.ReadMigrationMarker(
					h.Ctx,
					"upgrade_marker_upgrade-incident-operations-v1",
				))
			})
		}
	}

	for _, marker := range []struct {
		name  string
		value string
	}{
		{name: "missing"},
		{name: "wrong", value: "unexpected"},
	} {
		t.Run("marker/"+marker.name, func(t *testing.T) {
			h := NewTestHarness(t)
			if marker.value != "" {
				require.NoError(t, h.KnowledgeKeeper.WriteMigrationMarker(
					h.Ctx,
					"upgrade_marker_consolidation-safety-v1",
					marker.value,
				))
			}
			_, err := h.App.RunUpgradeHandlerWithInfoForTests(
				h.Ctx,
				zeroneapp.UpgradeNameSDK053IBC10,
				sdk053IBC10SourceVM(h),
				h.Height(),
				planInfo,
			)
			require.Error(t, err)
			require.Contains(t, err.Error(),
				`requires prerequisite marker "upgrade_marker_consolidation-safety-v1"="migrated"`)
			require.Empty(t, h.KnowledgeKeeper.ReadMigrationMarker(
				h.Ctx,
				"upgrade_marker_sdk-0.53-ibc-10",
			))
		})
	}
}

func TestSDK053IBC10ScheduledPreflightRefusesPersistedUnconsolidatedVersion(
	t *testing.T,
) {
	h := NewTestHarness(t)
	seedConsolidationBoundary(t, h)
	versionMap := sdk053IBC10SourceVM(h)
	versionMap[knowledgetypes.ModuleName] = 5
	require.NoError(t, h.App.UpgradeKeeper.SetModuleVersionMap(
		h.Ctx,
		versionMap,
	))
	planInfo, err := zeroneapp.BuildSDK053IBC10PlanInfo(nil, nil)
	require.NoError(t, err)
	plan := upgradetypes.Plan{
		Name:   zeroneapp.UpgradeNameSDK053IBC10,
		Height: h.App.CommitMultiStore().LastCommitID().Version + 2,
		Info:   planInfo,
	}
	require.NoError(t, h.App.UpgradeKeeper.ScheduleUpgrade(h.Ctx, plan))
	h.CommitHMinusOne()
	require.Equal(t, plan.Height-1,
		h.App.CommitMultiStore().LastCommitID().Version,
		"fixture must exercise the exact scheduled H-1 verifier",
	)

	report, err := h.App.VerifyScheduledActivationPrestate()
	require.ErrorContains(t, err,
		`requires prerequisite module "knowledge" at consensus version 6: got 5`)
	require.False(t, report.ActivationReady)
	require.Empty(t, h.KnowledgeKeeper.ReadMigrationMarker(
		h.Ctx,
		"upgrade_marker_sdk-0.53-ibc-10",
	), "H-1 refusal must happen before the handler dry-run")
}

func TestSDK053IBC10RefusesMissingOrWrongSafetySourceVersionsBeforeMutation(
	t *testing.T,
) {
	safetyModules := []struct {
		name    string
		version uint64
	}{
		{zeroneemergencytypes.ModuleName, 1},
		{sdkgovtypes.ModuleName, 5},
		{zeronegovtypes.ModuleName, 2},
	}
	wrongVersion := uint64(99)
	corruptions := []struct {
		name    string
		version *uint64
	}{
		{name: "missing"},
		{name: "wrong", version: &wrongVersion},
	}

	for _, safetyModule := range safetyModules {
		for _, corruption := range corruptions {
			t.Run(
				safetyModule.name+"/"+corruption.name,
				func(t *testing.T) {
					h := NewTestHarness(t)
					legacyLIP := &zeronegovtypes.LIP{
						Id:          "LIP-source-version-guard",
						Title:       "source version guard",
						Description: "must remain untouched on guard failure",
						Category:    zeronegovtypes.CategoryUpgrade,
						Stage:       zeronegovtypes.StatusReview,
					}
					h.GovKeeper.SetLIP(h.Ctx, legacyLIP)
					rawEmergencyStore := h.Ctx.KVStore(
						h.App.GetStoreKeyForTests(
							zeroneemergencytypes.StoreKey,
						),
					)
					rawEmergencyStore.Delete(
						zeroneemergencytypes.HaltStatusKey,
					)

					vm := sdk053IBC10SourceVM(h)
					require.NoError(t, h.App.UpgradeKeeper.SetModuleVersionMap(
						h.Ctx,
						vm,
					))
					versionKey := append(
						[]byte{upgradetypes.VersionMapByte},
						[]byte(safetyModule.name)...,
					)
					upgradeStore := h.Ctx.KVStore(
						h.App.GetStoreKeyForTests(upgradetypes.StoreKey),
					)
					if corruption.version == nil {
						delete(vm, safetyModule.name)
						upgradeStore.Delete(versionKey)
					} else {
						vm[safetyModule.name] = *corruption.version
						encoded := make([]byte, 8)
						binary.BigEndian.PutUint64(encoded, *corruption.version)
						upgradeStore.Set(versionKey, encoded)
					}
					h.CommitHMinusOne()

					_, err := runSDK053IBC10HandlerForTests(
						t,
						h,
						vm,
						h.Height()+1,
					)
					require.Error(t, err)
					require.Contains(t, err.Error(), safetyModule.name)
					require.Contains(
						t,
						err.Error(),
						"authenticated source module",
					)
					postUpgradeEmergencyStore := h.Ctx.KVStore(
						h.App.GetStoreKeyForTests(
							zeroneemergencytypes.StoreKey,
						),
					)
					require.False(
						t,
						postUpgradeEmergencyStore.Has(
							zeroneemergencytypes.HaltStatusKey,
						),
						"guard failure must precede emergency normalization",
					)
					stored, found := h.GovKeeper.GetLIP(h.Ctx, legacyLIP.Id)
					require.True(t, found)
					require.Equal(t, zeronegovtypes.StatusReview, stored.Stage)
					require.Empty(
						t,
						h.KnowledgeKeeper.ReadMigrationMarker(
							h.Ctx,
							"upgrade_marker_upgrade-incident-operations-v1",
						),
					)
				},
			)
		}
	}
}

func TestActivationPreflightCommonVerifierIsReadOnly(
	t *testing.T,
) {
	h := NewTestHarness(t)
	require.NoError(t, h.App.UpgradeKeeper.SetModuleVersionMap(
		h.Ctx,
		activationSafetySourceVM(h),
	))
	legacyLIP := &zeronegovtypes.LIP{
		Id:          "LIP-preflight-custom-upgrade",
		Title:       "preflight custom upgrade",
		Description: "must be reported but not terminalized",
		Category:    zeronegovtypes.CategoryUpgrade,
		Stage:       zeronegovtypes.StatusReview,
	}
	h.GovKeeper.SetLIP(h.Ctx, legacyLIP)
	h.CommitHMinusOne()

	report, err := h.App.VerifyActivationPrestate()
	require.NoError(t, err)
	require.Equal(t, "zerone.activation-preflight/v3", report.Schema)
	require.Equal(t, "common-safety", report.Scope)
	require.False(t, report.ActivationReady)
	require.Equal(t, h.App.CommitMultiStore().LastCommitID().Version, report.Height)
	require.Len(t, report.AppHash, 64)
	require.Equal(t, uint64(1), report.SafetySourceVersions["emergency"])
	require.Equal(t, uint64(5), report.SafetySourceVersions["gov"])
	require.Equal(t, uint64(2), report.SafetySourceVersions["zerone_gov"])
	require.Equal(t, 1, report.CustomGovUpgradeLIPRecords)
	require.Positive(t, report.CustomGovUpgradeLIPBytes)
	require.Equal(
		t,
		[]string{legacyLIP.Id},
		report.CustomGovUpgradeLIPIDs,
	)
	require.Equal(t, "0", report.CustomGovUnattributedStakeUzrn)
	require.Len(t, report.CustomGovStakeManifestSHA256, 64)

	stored, found := h.GovKeeper.GetLIP(h.Ctx, legacyLIP.Id)
	require.True(t, found)
	require.Equal(
		t,
		zeronegovtypes.StatusReview,
		stored.Stage,
		"preflight must not apply retirement",
	)
}

func TestSDK053IBC10RefusesUnattributedCustomUpgradeStakeBeforeMutation(
	t *testing.T,
) {
	h := NewTestHarness(t)
	fromVM := sdk053IBC10SourceVM(h)
	require.NoError(t, h.App.UpgradeKeeper.SetModuleVersionMap(
		h.Ctx,
		fromVM,
	))
	locked := &zeronegovtypes.LIP{
		Id:           "LIP-unattributed-stake",
		Title:        "unattributed stake",
		Description:  "must block terminalization",
		Category:     zeronegovtypes.CategoryUpgrade,
		Stage:        zeronegovtypes.StatusReview,
		StakedAmount: "42",
	}
	h.GovKeeper.SetLIP(h.Ctx, locked)
	rawEmergencyStore := h.Ctx.KVStore(
		h.App.GetStoreKeyForTests(zeroneemergencytypes.StoreKey),
	)
	rawEmergencyStore.Delete(zeroneemergencytypes.HaltStatusKey)
	h.CommitHMinusOne()

	_, err := h.App.VerifyActivationPrestate()
	require.Error(t, err)
	require.Contains(t, err.Error(), "would strand 42 uzrn")
	require.Contains(t, err.Error(), "stake manifest sha256")

	_, err = runSDK053IBC10HandlerForTests(
		t,
		h,
		fromVM,
		h.Height()+1,
	)
	require.Error(t, err)
	require.Contains(t, err.Error(), "would strand 42 uzrn")
	require.Contains(t, err.Error(), "stake manifest sha256")
	require.False(
		t,
		h.Ctx.KVStore(
			h.App.GetStoreKeyForTests(zeroneemergencytypes.StoreKey),
		).Has(zeroneemergencytypes.HaltStatusKey),
		"stake guard must run before emergency normalization",
	)
	stored, found := h.GovKeeper.GetLIP(h.Ctx, locked.Id)
	require.True(t, found)
	require.Equal(t, zeronegovtypes.StatusReview, stored.Stage)
}

func TestSDK053IBC10RefusesPreseededCustomGovernanceHoldKey(
	t *testing.T,
) {
	h := NewTestHarness(t)
	fromVM := sdk053IBC10SourceVM(h)
	require.NoError(t, h.App.UpgradeKeeper.SetModuleVersionMap(
		h.Ctx,
		fromVM,
	))
	rawCustomGovStore := h.Ctx.KVStore(
		h.App.GetStoreKeyForTests(zeronegovtypes.StoreKey),
	)
	rawCustomGovStore.Set(
		zeronegovtypes.EmergencyTransitionHoldKey,
		[]byte("source-version-must-not-own-this-key"),
	)
	rawEmergencyStore := h.Ctx.KVStore(
		h.App.GetStoreKeyForTests(zeroneemergencytypes.StoreKey),
	)
	rawEmergencyStore.Delete(zeroneemergencytypes.HaltStatusKey)
	h.CommitHMinusOne()

	_, err := h.App.VerifyActivationPrestate()
	require.ErrorContains(
		t,
		err,
		"pre-seeds reserved emergency transition hold key",
	)
	_, err = runSDK053IBC10HandlerForTests(
		t,
		h,
		fromVM,
		h.Height()+1,
	)
	require.ErrorContains(
		t,
		err,
		"pre-seeds reserved emergency transition hold key",
	)
	require.Equal(
		t,
		[]byte("source-version-must-not-own-this-key"),
		rawCustomGovStore.Get(zeronegovtypes.EmergencyTransitionHoldKey),
	)
	require.Empty(
		t,
		h.KnowledgeKeeper.ReadMigrationMarker(
			h.Ctx,
			"upgrade_marker_upgrade-incident-operations-v1",
		),
	)
}

func TestUpgrade_SDK053IBC10RefusesLegacyFeeBalance(t *testing.T) {
	h := NewTestHarness(t)
	seedConsolidationBoundary(t, h)

	feeAddress := authtypes.NewModuleAddress("feeibc")
	require.NoError(t, h.FundAccount(
		feeAddress,
		sdk.NewCoins(sdk.NewCoin(zeroneapp.BondDenom, sdkmath.NewInt(1))),
	))

	current := activationSafetySourceVM(h)
	fromVM := make(module.VersionMap, len(current))
	for name, version := range current {
		fromVM[name] = version
	}
	fromVM["ibc"] = 6
	fromVM["transfer"] = 5
	fromVM["interchainaccounts"] = 3
	fromVM["capability"] = 1
	fromVM["feeibc"] = 2

	planInfo, err := zeroneapp.BuildSDK053IBC10PlanInfo(nil, nil)
	require.NoError(t, err)
	_, err = h.App.RunUpgradeHandlerWithInfoForTests(
		h.Ctx,
		zeroneapp.UpgradeNameSDK053IBC10,
		fromVM,
		h.Height(),
		planInfo,
	)
	require.Error(t, err)
	require.Contains(t, err.Error(), "cannot remove legacy IBC fee middleware")
	require.Equal(
		t,
		sdkmath.NewInt(1),
		h.BankKeeper.GetBalance(h.Ctx, feeAddress, zeroneapp.BondDenom).Amount,
		"failed upgrade must not move or burn the legacy fee balance",
	)
	require.Empty(
		t,
		h.KnowledgeKeeper.ReadMigrationMarker(h.Ctx, "upgrade_marker_auth-ante-hardening-v1"),
		"the auth marker must remain unwritten when the legacy fee-balance guard fails",
	)
}

// TestUpgrade_SubstrateDedupeV1SeedsAndArms drives the real substrate-dedupe-v1
// handler end-to-end: it must run RunMigrations + ReconcileModuleAccountPerms +
// SeedSourceRefs + WriteMigrationMarker without error, index a pre-existing
// settled attestation's source, and leave enforcement armed. Guards the whole
// wiring under the exact plan name a governance proposal would carry.
func TestUpgrade_SubstrateDedupeV1SeedsAndArms(t *testing.T) {
	h := NewTestHarness(t)

	// A settled attestation that predates the wall — the seed must index its
	// source so a post-upgrade replay is blocked.
	link := &substratebridgetypes.SubstrateLink{
		AdapterId: "agenttool-invocation-v1",
		Source: &substratebridgetypes.ExternalSource{
			AdapterId: "agenttool-invocation-v1",
			SourceId:  "inv-preupgrade",
		},
	}
	require.NoError(t, h.SubstrateBridgeKeeper.WriteAttestation(h.Ctx, &substratebridgetypes.ExternalAttestation{
		AttestationId: "att-1-1",
		AdapterId:     "agenttool-invocation-v1",
		Submitter:     "zerone1preupgradesubmitter000000000000aa",
		Status:        substratebridgetypes.AttestationStatus_ATTESTATION_STATUS_SETTLED,
		Link:          link,
	}))

	fromVM := h.App.CurrentModuleVersionMap()
	_, err := h.App.RunUpgradeHandlerForTests(h.Ctx, zeroneapp.UpgradeNameSubstrateDedupeV1, fromVM, h.Height())
	require.NoError(t, err)

	holder, taken := h.SubstrateBridgeKeeper.GetSourceRef(h.Ctx, "agenttool-invocation-v1", "inv-preupgrade")
	require.True(t, taken, "handler must seed the pre-existing attestation's source")
	require.Equal(t, "att-1-1", holder)
	require.True(t, h.SubstrateBridgeKeeper.IsDedupeArmed(h.Ctx), "handler must arm enforcement")
	require.Equal(t, "migrated",
		h.KnowledgeKeeper.ReadMigrationMarker(h.Ctx, "upgrade_marker_substrate-dedupe-v1"),
		"handler must write the substrate-dedupe-v1 migration marker")
}

// TestUpgrade_AgenttoolSeamV1DeclaresAxisBounds drives the real agenttool-seam-v1
// handler end-to-end under the exact plan name a governance proposal would carry.
// The adapter is written in the genesis shape of agenttool-invocation-v1 — active,
// AxisBounds nil — which is the state that let recursion weight through unbounded.
// After the handler, the ceiling must exist, and a weighted claim must be refused
// by the same code path a real submission takes.
func TestUpgrade_AgenttoolSeamV1DeclaresAxisBounds(t *testing.T) {
	h := NewTestHarness(t)

	require.NoError(t, h.SubstrateBridgeKeeper.WriteAdapter(h.Ctx, &substratebridgetypes.AdapterRegistration{
		AdapterId: "agenttool-invocation-v1",
		Status:    substratebridgetypes.AdapterStatus_ADAPTER_STATUS_ACTIVE,
		// AxisBounds nil — exactly how genesis seeded it.
	}))

	before, found := h.SubstrateBridgeKeeper.GetAdapter(h.Ctx, "agenttool-invocation-v1")
	require.True(t, found)
	require.Nil(t, before.AxisBounds, "precondition: the drain state must be reproduced")

	fromVM := h.App.CurrentModuleVersionMap()
	_, err := h.App.RunUpgradeHandlerForTests(h.Ctx, zeroneapp.UpgradeNameAgenttoolSeamV1, fromVM, h.Height())
	require.NoError(t, err)

	after, found := h.SubstrateBridgeKeeper.GetAdapter(h.Ctx, "agenttool-invocation-v1")
	require.True(t, found)
	require.NotNil(t, after.AxisBounds, "handler must leave the adapter declaring a ceiling")

	// The drain, attempted through the real validation path.
	drain := &substratebridgetypes.SubstrateLink{
		AdapterId: "agenttool-invocation-v1",
		Source: &substratebridgetypes.ExternalSource{
			AdapterId: "agenttool-invocation-v1",
			SourceId:  "inv-drain",
		},
		RecursionWeight: &substratebridgetypes.AxisProjection{
			AxisSubstrate: ^uint64(0), AxisVerification: ^uint64(0),
			AxisClassification: ^uint64(0), AxisAttribution: ^uint64(0),
			AxisTooling: ^uint64(0), AxisInterface: ^uint64(0),
		},
	}
	substrateParams := substratebridgetypes.DefaultParams()
	// ErrAxisOverflow, not ErrAdapterAxisBoundsUnset: after the migration this
	// adapter *does* declare a ceiling, so the claim is refused for exceeding it.
	// The two errors mark the two halves of the fix — the migration closes the
	// drain for adapters that already exist, the gate closes it for any adapter
	// registered without bounds later. This test exercises the first.
	require.ErrorIs(t, h.SubstrateBridgeKeeper.ValidateLink(h.Ctx, drain, substrateParams),
		substratebridgetypes.ErrAxisOverflow,
		"a weighted claim must not survive the upgrade")

	// The live relay's shape — source only, no weight — must still pass.
	relayShaped := &substratebridgetypes.SubstrateLink{
		AdapterId: "agenttool-invocation-v1",
		Source: &substratebridgetypes.ExternalSource{
			AdapterId:   "agenttool-invocation-v1",
			SourceId:    "inv-normal",
			ContentHash: make([]byte, 32),
		},
	}
	require.NoError(t, h.SubstrateBridgeKeeper.ValidateLink(h.Ctx, relayShaped, substrateParams),
		"the upgrade must not stall the live bridge")

	require.Equal(t, "migrated",
		h.KnowledgeKeeper.ReadMigrationMarker(h.Ctx, "upgrade_marker_agenttool-seam-v1"),
		"handler must write the agenttool-seam-v1 migration marker")
}

func TestUpgrade_ConsolidationSafetyV1RecordsActivationBoundary(t *testing.T) {
	h := NewTestHarness(t)

	current := h.App.CurrentModuleVersionMap()

	// Reproduce the published pre-H1 economic parameter shapes. H1 is the one
	// atomic boundary that advances lifecycle safety and retires control-linked
	// automatic claims.
	legacyLiquidity := h.App.LiquidityPoolKeeper.GetParams(h.Ctx)
	legacyLiquidity.ProtocolFeeBps = 450_000
	legacyLiquidity.MaxPools = 0
	legacyLiquidity.AllowedPoolDenoms = nil
	legacyLiquidity.PoolCreators = nil
	legacyLiquidity.BillingQuoteDenoms = nil
	h.App.LiquidityPoolKeeper.SetParams(h.Ctx, legacyLiquidity)

	legacyRewards := h.App.VestingRewardsKeeper.GetParams(h.Ctx)
	legacyRewards.FounderShareBps = 70_000
	legacyRewards.FounderAddress = "zrn1legacyfounder00000000000000000000000"
	legacyRewards.BlockReward = "10000000"
	legacyRewards.FloorReward = "100000"
	h.App.VestingRewardsKeeper.SetParams(h.Ctx, legacyRewards)

	fromVM := make(module.VersionMap, len(current))
	for name, version := range current {
		fromVM[name] = version
	}
	fromVM["knowledge"] = 5
	fromVM["claiming_pot"] = 1
	fromVM[liquiditypooltypes.ModuleName] = 3
	fromVM[vestingrewardstypes.ModuleName] = 1

	toVM, err := h.App.RunUpgradeHandlerForTests(
		h.Ctx,
		zeroneapp.UpgradeNameConsolidationSafetyV1,
		fromVM,
		h.Height(),
	)
	require.NoError(t, err)
	require.Equal(t, uint64(6), toVM["knowledge"])
	require.Equal(t, uint64(2), toVM["claiming_pot"])
	require.Equal(t, uint64(5), toVM[liquiditypooltypes.ModuleName])
	require.Equal(t, uint64(2), toVM[vestingrewardstypes.ModuleName])

	liquidityParams := h.App.LiquidityPoolKeeper.GetParams(h.Ctx)
	require.Zero(t, liquidityParams.ProtocolFeeBps,
		"H1 retires the protocol skim")
	require.Empty(t, liquidityParams.AllowedPoolDenoms,
		"pool asset admission remains fail-closed")
	require.Empty(t, liquidityParams.PoolCreators,
		"pool creator admission remains fail-closed")
	require.Empty(t, liquidityParams.BillingQuoteDenoms,
		"billing oracle admission remains fail-closed")

	rewardParams := h.App.VestingRewardsKeeper.GetParams(h.Ctx)
	require.Zero(t, rewardParams.FounderShareBps)
	require.Empty(t, rewardParams.FounderAddress)
	require.Equal(t, "0", rewardParams.BlockReward)
	require.Equal(t, "0", rewardParams.FloorReward)
	require.Zero(t, rewardParams.EmptyBlockRewardRate)
	require.Equal(t, "true",
		h.KnowledgeKeeper.ReadMigrationMarker(h.Ctx, "migration_v6_complete"))
	require.Equal(t, "migrated",
		h.KnowledgeKeeper.ReadMigrationMarker(h.Ctx, "upgrade_marker_consolidation-safety-v1"))
}

func TestUpgrade_ConsolidationSafetyV1CannotBeBlockedByAnUnrelatedBallot(t *testing.T) {
	h := NewTestHarness(t)
	h.GovKeeper.SetLIP(h.Ctx, &zeronegovtypes.LIP{
		Id:    "LIP-live-ballot",
		Stage: zeronegovtypes.StatusVoting,
	})

	current := h.App.CurrentModuleVersionMap()
	fromVM := make(module.VersionMap, len(current))
	for name, version := range current {
		fromVM[name] = version
	}
	fromVM["knowledge"] = 5
	fromVM["claiming_pot"] = 1

	toVM, err := h.App.RunUpgradeHandlerForTests(
		h.Ctx,
		zeroneapp.UpgradeNameConsolidationSafetyV1,
		fromVM,
		h.Height(),
	)
	require.NoError(t, err,
		"an unrelated permissionless ballot must not halt a scheduled upgrade")
	require.Equal(t, uint64(6), toVM["knowledge"])
	require.Equal(t, uint64(2), toVM["claiming_pot"])
	lip, found := h.GovKeeper.GetLIP(h.Ctx, "LIP-live-ballot")
	require.True(t, found)
	require.Equal(t, zeronegovtypes.StatusVoting, lip.Stage,
		"the upgrade must not rewrite an unrelated ballot")
}

// H1 retains the permanent module-account reconciliation step. It restores
// liquidity's LP-only mint/burn authority and strips vesting_rewards' retired
// automatic-mint permission in the same atomic transition.
func TestUpgrade_ConsolidationSafetyV1ReconcilesEconomicPermissions(t *testing.T) {
	h := NewTestHarness(t)

	liquidityAccount := h.AccountKeeper.GetModuleAccount(h.Ctx, liquiditypooltypes.ModuleName)
	require.NotNil(t, liquidityAccount)
	vestingAccount := h.AccountKeeper.GetModuleAccount(h.Ctx, vestingrewardstypes.ModuleName)
	require.NotNil(t, vestingAccount)

	// Reproduce an old liquidity account missing LP permissions.
	h.AccountKeeper.SetModuleAccount(h.Ctx, authtypes.NewModuleAccount(
		authtypes.NewBaseAccount(
			liquidityAccount.GetAddress(),
			nil,
			liquidityAccount.GetAccountNumber(),
			liquidityAccount.GetSequence(),
		),
		liquiditypooltypes.ModuleName,
	))
	// Reproduce the pre-v2 vesting account with mint/burn permissions.
	h.AccountKeeper.SetModuleAccount(h.Ctx, authtypes.NewModuleAccount(
		authtypes.NewBaseAccount(
			vestingAccount.GetAddress(),
			nil,
			vestingAccount.GetAccountNumber(),
			vestingAccount.GetSequence(),
		),
		vestingrewardstypes.ModuleName,
		authtypes.Minter,
		authtypes.Burner,
	))

	drifted := h.AccountKeeper.GetModuleAccount(h.Ctx, liquiditypooltypes.ModuleName)
	require.Empty(t, drifted.GetPermissions(), "precondition: stored permissions drifted")
	require.ElementsMatch(t, []string{authtypes.Minter, authtypes.Burner},
		h.AccountKeeper.GetModuleAccount(h.Ctx, vestingrewardstypes.ModuleName).GetPermissions())

	fromVM := h.App.CurrentModuleVersionMap()
	toVM, err := h.App.RunUpgradeHandlerForTests(
		h.Ctx,
		zeroneapp.UpgradeNameConsolidationSafetyV1,
		fromVM,
		h.Height(),
	)
	require.NoError(t, err)
	require.Equal(t, uint64(5), toVM[liquiditypooltypes.ModuleName])
	require.Equal(t, uint64(2), toVM[vestingrewardstypes.ModuleName])

	reconciled := h.AccountKeeper.GetModuleAccount(h.Ctx, liquiditypooltypes.ModuleName)
	require.ElementsMatch(t,
		[]string{authtypes.Minter, authtypes.Burner},
		reconciled.GetPermissions(),
		"H1 must restore liquidity's LP mint/burn permissions")
	require.Empty(t,
		h.AccountKeeper.GetModuleAccount(h.Ctx, vestingrewardstypes.ModuleName).GetPermissions(),
		"H1 must remove vesting_rewards' retired mint/burn permissions")
}
