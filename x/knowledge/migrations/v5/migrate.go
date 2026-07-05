package v5

// Knowledge module v4 → v5 migration
//
// v5 removes eleven dead params from the module's Params:
//   - citation-gaming knobs: novelty_bonus_bps, max_citations_per_claim,
//     citation_decay_per_level, self_citation_discount_bps
//   - FARM anti-gaming block: conformity_threshold_bps,
//     calibration_trivial_threshold, misbehavior_rejection_threshold,
//     min_domain_contributors_for_novelty, min_participation_rate_bps,
//     challenge_stake_ratio_min_bps
//   - malformed_claim_slash_bps
//
// Every one carried a confident default but was read by NO keeper logic — the
// chain advertised a quality gate it never enforced. Removing them is the
// honest move: no dead knobs implying control that isn't there.
//
// State effect: the removed proto fields become unknown on decode and are
// dropped when Params is re-marshalled. This migration re-saves Params to drop
// the dead bytes, then records a verifiable marker. Idempotent — safe on
// genesis-fresh chains (a no-op rewrite) and on chains upgrading from v4.
//
// Follows the reference upgrade pattern in docs/UPGRADE_PROTOCOL.md.

import (
	"context"

	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// V5MigrationKeeper is the narrow keeper interface the v5 migration requires.
// Keeping it local prevents the migrations/v5 → keeper → migrations/v5 cycle.
type V5MigrationKeeper interface {
	GetParams(ctx context.Context) (*types.Params, error)
	SetParams(ctx context.Context, params *types.Params) error
	WriteMigrationMarker(ctx context.Context, key, value string) error
}

// Migrate runs the knowledge v4 → v5 state transformation.
// Returns nil on success; any returned error rolls back the upgrade.
func Migrate(ctx context.Context, k V5MigrationKeeper) error {
	// Re-save Params so the removed (now-unknown) param fields are dropped
	// from stored state. On a genesis-fresh chain this is a no-op rewrite.
	params, err := k.GetParams(ctx)
	if err != nil {
		return err
	}
	if err := k.SetParams(ctx, params); err != nil {
		return err
	}

	// Record a verifiable marker so upgrade e2e tests can confirm the handler ran.
	return k.WriteMigrationMarker(ctx, "migration_v5_complete", "true")
}
