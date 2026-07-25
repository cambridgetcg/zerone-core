package app

import (
	"context"
	"fmt"
	"sort"

	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
)

const UpgradeNameTestnet = "v1.0.0-testnet"
const UpgradeNameTestnetV2 = "v1.0.1-testnet"
const UpgradeNameTestnetV3 = "v1.0.2-testnet"
const UpgradeNameTestnetV4 = "v1.0.3-testnet"
const UpgradeNameLiquidityHardeningV1 = "liquiditypool-hardening-v1"
const UpgradeNameCompassionCalibrationV1 = "compassion-calibration-v1"
const UpgradeNameDoctrineMetabolismExemptV1 = "doctrine-metabolism-exempt-v1"
const UpgradeNameSubstrateDedupeV1 = "substrate-dedupe-v1"
const UpgradeNameAgenttoolSeamV1 = "agenttool-seam-v1"

// RegisterUpgradeHandlers registers upgrade handlers for each named software upgrade.
// When a governance upgrade proposal passes, the corresponding handler here runs
// the necessary state migrations before the new binary starts producing blocks.
//
// Call this AFTER RegisterServices but BEFORE LoadLatestVersion.
func (app *ZeroneApp) RegisterUpgradeHandlers() {
	// v1.0.0-testnet — initial testnet launch.
	// Runs all module migrations from ConsensusVersion 1 → 2.
	app.UpgradeKeeper.SetUpgradeHandler(
		UpgradeNameTestnet,
		func(ctx context.Context, plan upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
			app.Logger().Info(fmt.Sprintf("applying upgrade %q at height %d", plan.Name, plan.Height))
			return app.ModuleManager.RunMigrations(ctx, app.configurator, fromVM)
		},
	)

	// v1.0.1-testnet — simulated upgrade for testing the migration pipeline.
	// Runs module migrations (knowledge v1→v2 writes a verifiable marker) and
	// writes its own upgrade marker to the knowledge store.
	app.UpgradeKeeper.SetUpgradeHandler(
		UpgradeNameTestnetV2,
		func(ctx context.Context, plan upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
			app.Logger().Info(fmt.Sprintf("applying upgrade %q at height %d", plan.Name, plan.Height))

			toVM, err := app.ModuleManager.RunMigrations(ctx, app.configurator, fromVM)
			if err != nil {
				return nil, err
			}

			// Handler-level marker (via the knowledge keeper's marker API)
			// to prove this named upgrade handler executed. Tests read it
			// via ReadMigrationMarker.
			if err := app.KnowledgeKeeper.WriteMigrationMarker(ctx, "upgrade_marker_v1.0.1", "migrated"); err != nil {
				return nil, err
			}

			return toVM, nil
		},
	)

	// v1.0.2-testnet — Wave 10 reference upgrade exercising the v3→v4
	// knowledge migration (TraceSchema backfill + v4 marker). Also used by
	// the end-to-end upgrade test to verify the full pipeline works.
	app.UpgradeKeeper.SetUpgradeHandler(
		UpgradeNameTestnetV3,
		func(ctx context.Context, plan upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
			app.Logger().Info(fmt.Sprintf("applying upgrade %q at height %d", plan.Name, plan.Height))

			toVM, err := app.ModuleManager.RunMigrations(ctx, app.configurator, fromVM)
			if err != nil {
				return nil, err
			}

			// Handler-level marker — tests assert both the per-module v4
			// marker (written by the migrator) AND this handler-level marker
			// were recorded, proving both layers ran.
			if err := app.KnowledgeKeeper.WriteMigrationMarker(ctx, "upgrade_marker_v1.0.2", "migrated"); err != nil {
				return nil, err
			}

			return toVM, nil
		},
	)

	// v1.0.3-testnet — the first upgrade the chain can actually EXECUTE at a
	// height (the PreBlocker fix landed with it). Carries the two migrations
	// stranded since v1.0.2 — knowledge v4→v5 (dead anti-slop param removal)
	// and liquiditypool v1→v2 (TWAP StartBlock backfill) — and reconciles
	// stored module-account permissions with the code's maccPerms.
	app.UpgradeKeeper.SetUpgradeHandler(
		UpgradeNameTestnetV4,
		func(ctx context.Context, plan upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
			app.Logger().Info(fmt.Sprintf("applying upgrade %q at height %d", plan.Name, plan.Height))

			toVM, err := app.ModuleManager.RunMigrations(ctx, app.configurator, fromVM)
			if err != nil {
				return nil, err
			}

			// Permanent reconcile step (keep in every future handler): bank's
			// mint/burn checks read the module account STORED in x/auth state,
			// which is frozen at whatever maccPerms said when the account was
			// first touched. Rebuild any account whose stored permissions
			// drifted from the code — the substrate_bridge-Burner class of
			// bug, fixed generically and idempotently.
			app.ReconcileModuleAccountPerms(ctx)

			if err := app.KnowledgeKeeper.WriteMigrationMarker(ctx, "upgrade_marker_v1.0.3", "migrated"); err != nil {
				return nil, err
			}

			return toVM, nil
		},
	)

	// liquiditypool-hardening-v1 — the Phase-1 gates before external liquidity
	// (docs/plans/2026-07-06-defi-liquidity-pipeline.md): fail-closed billing
	// oracle quote-denom allowlist (new Params field), real ProtocolFeeBps to
	// fee_collector, Locked flag on Add/RemoveLiquidity, ZRN-quoted pairs with
	// the floor on the uzrn side only, 10% swap-fee ceiling. Carries the
	// liquiditypool v2→v3 migration (seeds the empty allowlist explicitly).
	app.UpgradeKeeper.SetUpgradeHandler(
		UpgradeNameLiquidityHardeningV1,
		func(ctx context.Context, plan upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
			app.Logger().Info(fmt.Sprintf("applying upgrade %q at height %d", plan.Name, plan.Height))

			toVM, err := app.ModuleManager.RunMigrations(ctx, app.configurator, fromVM)
			if err != nil {
				return nil, err
			}

			// Permanent reconcile step (kept in every handler — see v1.0.3).
			app.ReconcileModuleAccountPerms(ctx)

			if err := app.KnowledgeKeeper.WriteMigrationMarker(ctx, "upgrade_marker_liquiditypool-hardening-v1", "migrated"); err != nil {
				return nil, err
			}

			return toVM, nil
		},
	)

	// compassion-calibration-v1 — commitment C2 of docs/COMPASSION.md ("error is
	// not deceit"). ComputeAgentCalibrationScore now EXCLUDES inconclusive
	// outcomes from the calibration denominator: an inconclusive is the panel
	// failing to resolve, not the agent failing to be right, so it no longer
	// drags a submitter's score the way a refuted claim does. This handler
	// recomputes every stored calibration score under the new formula (monotonic
	// — scores with inconclusive history rise, all others are unchanged). No
	// store or proto change; the score feeds x/trust_score and the training-fund
	// disbursement gate, so the refresh keeps stored state == live computation
	// on every node from the upgrade height.
	app.UpgradeKeeper.SetUpgradeHandler(
		UpgradeNameCompassionCalibrationV1,
		func(ctx context.Context, plan upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
			app.Logger().Info(fmt.Sprintf("applying upgrade %q at height %d", plan.Name, plan.Height))

			toVM, err := app.ModuleManager.RunMigrations(ctx, app.configurator, fromVM)
			if err != nil {
				return nil, err
			}

			// Permanent reconcile step (kept in every handler — see v1.0.3).
			app.ReconcileModuleAccountPerms(ctx)

			// Refresh every stored calibration score under the inconclusive-
			// excluding formula. Deterministic single pass; monotonic.
			n, err := app.KnowledgeKeeper.RecomputeAllCalibrationScores(ctx)
			if err != nil {
				return nil, err
			}
			app.Logger().Info(fmt.Sprintf("compassion-calibration-v1: recomputed %d calibration scores", n))

			if err := app.KnowledgeKeeper.WriteMigrationMarker(ctx, "upgrade_marker_compassion-calibration-v1", "migrated"); err != nil {
				return nil, err
			}

			return toVM, nil
		},
	)

	// doctrine-metabolism-exempt-v1 — the doctrine finding of
	// docs/plans/2026-07-10-framework-critique.md. Doctrine facts were born
	// starving (BuildDoctrineFact omitted Energy) and marched toward PRUNED at
	// block ~260,000 (~2026-07-16), which would display the chain's own
	// constitution as extinct-by-disuse. This upgrade makes doctrine live by
	// process, not by market: ProcessMetabolism now skips the doctrinal stratum
	// (starvation is not falsification — the same shape as C2), and this handler
	// resurrects the 47 genesis facts to VERIFIED at full energy with a cleared
	// at-risk clock. Deterministic single pass; idempotent. No store or proto
	// change. Doctrine is amended by the creed pin + amendment LIP, never starved.
	app.UpgradeKeeper.SetUpgradeHandler(
		UpgradeNameDoctrineMetabolismExemptV1,
		func(ctx context.Context, plan upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
			app.Logger().Info(fmt.Sprintf("applying upgrade %q at height %d", plan.Name, plan.Height))

			toVM, err := app.ModuleManager.RunMigrations(ctx, app.configurator, fromVM)
			if err != nil {
				return nil, err
			}

			// Permanent reconcile step (kept in every handler — see v1.0.3).
			app.ReconcileModuleAccountPerms(ctx)

			// Resurrect the doctrine corpus under the new metabolism exemption.
			n, err := app.KnowledgeKeeper.ResurrectDoctrineFacts(ctx)
			if err != nil {
				return nil, err
			}
			app.Logger().Info(fmt.Sprintf("doctrine-metabolism-exempt-v1: resurrected %d doctrine facts", n))

			if err := app.KnowledgeKeeper.WriteMigrationMarker(ctx, "upgrade_marker_doctrine-metabolism-exempt-v1", "migrated"); err != nil {
				return nil, err
			}

			return toVM, nil
		},
	)

	// substrate-dedupe-v1 — closes the external-attestation replay gap
	// (2026-07-23 integration audit): SubmitExternalAttestation now requires
	// a source reference and enforces (adapter_id, source_id) uniqueness, and
	// settlement mints only for the source's ref-holder, so one declared work
	// identity mints at most once. Rejection releases the source; a minted
	// holder keeps it forever. This handler seeds the index from every
	// attestation already on chain — pre-upgrade history becomes the wall's
	// first bricks — then arms enforcement. Seeding runs two ordered,
	// deterministic passes (minted holders before in-flight duplicates) and is
	// idempotent (an already-seeded ref is counted, never overwritten). No
	// store key or proto change; the index lives under a new prefix in the
	// existing substrate_bridge store.
	app.UpgradeKeeper.SetUpgradeHandler(
		UpgradeNameSubstrateDedupeV1,
		func(ctx context.Context, plan upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
			app.Logger().Info(fmt.Sprintf("applying upgrade %q at height %d", plan.Name, plan.Height))

			toVM, err := app.ModuleManager.RunMigrations(ctx, app.configurator, fromVM)
			if err != nil {
				return nil, err
			}

			// Permanent reconcile step (kept in every handler — see v1.0.3).
			app.ReconcileModuleAccountPerms(ctx)

			seeded, duplicates, sourceless, err := app.SubstrateBridgeKeeper.SeedSourceRefs(ctx)
			if err != nil {
				return nil, err
			}
			// Arm enforcement only after the index is seeded — the two are
			// atomic in this handler, which runs in the same PreBlock as the
			// binary swap. A plan-less restart onto the new binary skips this
			// handler, leaving enforcement disarmed, and SubmitExternalAttestation
			// fails closed rather than replay-minting against an empty index.
			app.SubstrateBridgeKeeper.SetDedupeArmed(ctx)
			app.Logger().Info(fmt.Sprintf(
				"substrate-dedupe-v1: seeded %d source refs, armed enforcement (%d pre-existing duplicate sources, %d sourceless attestations left unwalled)",
				seeded, duplicates, sourceless))

			if err := app.KnowledgeKeeper.WriteMigrationMarker(ctx, "upgrade_marker_substrate-dedupe-v1", "migrated"); err != nil {
				return nil, err
			}

			return toVM, nil
		},
	)

	// agenttool-seam-v1 — closes the axis-bounds drain on the agent-economy
	// seam (2026-07-25 incentive audit). RecursionWeight is caller-supplied and
	// multiplies the settlement reward, but ValidateLink only applied the
	// ceiling when the adapter declared one — so an adapter registered with
	// axis_bounds:null was the most permissive on chain rather than the most
	// restrictive, and six uint64s could claim the remaining supply cap in a
	// single message. mainnet's only live adapter, agenttool-invocation-v1, was
	// seeded exactly that way at genesis.
	//
	// The binary half of the fix refuses a weighted claim against an unbounded
	// adapter (ErrAdapterAxisBoundsUnset). This handler is the data half: it
	// gives every bounds-less adapter an explicit empty ceiling, so "unbounded"
	// stops existing as a reachable state rather than merely being unreachable
	// by the current caller.
	//
	// There is no governance path to do this instead: LIP dispatch for
	// CategoryAdapterRegistration is still a Phase-1 TODO (x/gov/keeper/abci.go),
	// so an upgrade handler is the only mechanism that can write adapter data.
	//
	// No proto or store-key change, and no live traffic is affected: the relay's
	// buildLink sets only `source`, never RecursionWeight.
	app.UpgradeKeeper.SetUpgradeHandler(
		UpgradeNameAgenttoolSeamV1,
		func(ctx context.Context, plan upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
			app.Logger().Info(fmt.Sprintf("applying upgrade %q at height %d", plan.Name, plan.Height))

			toVM, err := app.ModuleManager.RunMigrations(ctx, app.configurator, fromVM)
			if err != nil {
				return nil, err
			}

			// Permanent reconcile step (kept in every handler — see v1.0.3).
			app.ReconcileModuleAccountPerms(ctx)

			declared, err := app.SubstrateBridgeKeeper.DeclareMissingAxisBounds(ctx)
			if err != nil {
				return nil, err
			}
			app.Logger().Info(fmt.Sprintf(
				"agenttool-seam-v1: declared explicit zero axis bounds on %d adapter(s) that had none; weighted claims against an unbounded adapter are now refused",
				declared))

			if err := app.KnowledgeKeeper.WriteMigrationMarker(ctx, "upgrade_marker_agenttool-seam-v1", "migrated"); err != nil {
				return nil, err
			}

			return toVM, nil
		},
	)
}

// RegisterStoreUpgrades configures store loaders for upgrades that add or remove
// module store keys. Call this BEFORE LoadLatestVersion.
func (app *ZeroneApp) RegisterStoreUpgrades() {
	upgradeInfo, err := app.UpgradeKeeper.ReadUpgradeInfoFromDisk()
	if err != nil {
		// No pending upgrade — nothing to do.
		return
	}

	switch upgradeInfo.Name {
	case UpgradeNameTestnet:
		storeUpgrades := storetypes.StoreUpgrades{
			Added: []string{
				// Add new module store keys here when the upgrade introduces them.
			},
		}
		app.SetStoreLoader(upgradetypes.UpgradeStoreLoader(upgradeInfo.Height, &storeUpgrades))

	case UpgradeNameTestnetV2:
		// No new store keys for v1.0.1-testnet — migration-only upgrade.
		storeUpgrades := storetypes.StoreUpgrades{}
		app.SetStoreLoader(upgradetypes.UpgradeStoreLoader(upgradeInfo.Height, &storeUpgrades))

	case UpgradeNameTestnetV3:
		// v1.0.2-testnet — Wave 10 reference upgrade. No new store keys;
		// knowledge v3→v4 migration only touches existing prefixes.
		storeUpgrades := storetypes.StoreUpgrades{}
		app.SetStoreLoader(upgradetypes.UpgradeStoreLoader(upgradeInfo.Height, &storeUpgrades))

	case UpgradeNameTestnetV4:
		// v1.0.3-testnet — migration-only (knowledge v5, liquiditypool v2,
		// module-account perms reconcile). No store keys added or removed.
		storeUpgrades := storetypes.StoreUpgrades{}
		app.SetStoreLoader(upgradetypes.UpgradeStoreLoader(upgradeInfo.Height, &storeUpgrades))

	case UpgradeNameLiquidityHardeningV1:
		// Migration-only (liquiditypool v2→v3 — new Params field lives in the
		// existing store). No store keys added or removed.
		storeUpgrades := storetypes.StoreUpgrades{}
		app.SetStoreLoader(upgradetypes.UpgradeStoreLoader(upgradeInfo.Height, &storeUpgrades))

	case UpgradeNameCompassionCalibrationV1:
		// Migration-only — no store keys added or removed. The calibration score
		// is an existing field recomputed in the handler.
		storeUpgrades := storetypes.StoreUpgrades{}
		app.SetStoreLoader(upgradetypes.UpgradeStoreLoader(upgradeInfo.Height, &storeUpgrades))

	case UpgradeNameDoctrineMetabolismExemptV1:
		// Migration-only — no store keys. Doctrine facts are existing records
		// rewritten in the handler; the metabolism exemption is pure code.
		storeUpgrades := storetypes.StoreUpgrades{}
		app.SetStoreLoader(upgradetypes.UpgradeStoreLoader(upgradeInfo.Height, &storeUpgrades))

	case UpgradeNameSubstrateDedupeV1:
		// Migration-only — the source-ref index is a new prefix inside the
		// existing substrate_bridge store; no store keys added or removed.
		storeUpgrades := storetypes.StoreUpgrades{}
		app.SetStoreLoader(upgradetypes.UpgradeStoreLoader(upgradeInfo.Height, &storeUpgrades))
	}
}

// ReconcileModuleAccountPerms rebuilds every EXISTING module account whose
// STORED permission list differs from the code's maccPerms. Idempotent and
// cheap; run it in every named upgrade handler so permission drift can
// never strand funds again (bank checks stored perms, not code).
//
// Two determinism rules, learned from a live three-way AppHash divergence
// on the localnet upgrade drill:
//   - iterate maccPerms in SORTED order — Go map order differs per process;
//   - never call GetModuleAccount here — it lazily CREATES missing accounts,
//     consuming account numbers in iteration order. Accounts that don't
//     exist yet are skipped; lazy creation on first real use already applies
//     the current code's perms, so there is nothing to reconcile.
func (app *ZeroneApp) ReconcileModuleAccountPerms(ctx context.Context) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	names := make([]string, 0, len(maccPerms))
	for name := range maccPerms {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		perms := maccPerms[name]
		existing := app.AccountKeeper.GetAccount(sdkCtx, authtypes.NewModuleAddress(name))
		if existing == nil {
			continue // not yet created — lazy creation will apply current perms
		}
		acc, ok := existing.(sdk.ModuleAccountI)
		if !ok {
			continue
		}
		if equalStringSets(acc.GetPermissions(), perms) {
			continue
		}
		rebuilt := authtypes.NewModuleAccount(
			authtypes.NewBaseAccount(acc.GetAddress(), nil, acc.GetAccountNumber(), acc.GetSequence()),
			name, perms...,
		)
		app.AccountKeeper.SetModuleAccount(sdkCtx, rebuilt)
		app.Logger().Info("reconciled module account permissions",
			"account", name, "was", acc.GetPermissions(), "now", perms)
	}
}

func equalStringSets(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	seen := make(map[string]int, len(a))
	for _, s := range a {
		seen[s]++
	}
	for _, s := range b {
		if seen[s] == 0 {
			return false
		}
		seen[s]--
	}
	return true
}
