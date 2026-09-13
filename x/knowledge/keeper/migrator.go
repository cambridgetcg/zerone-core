package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/internal/claimrecordmigration"
	"github.com/zerone-chain/zerone/internal/fundsettlementmigration"
	"github.com/zerone-chain/zerone/internal/recordmigration"
	"github.com/zerone-chain/zerone/internal/reviewmigration"
	"github.com/zerone-chain/zerone/internal/survivalmigration"
	v3 "github.com/zerone-chain/zerone/x/knowledge/migrations/v3"
	v4 "github.com/zerone-chain/zerone/x/knowledge/migrations/v4"
	v5 "github.com/zerone-chain/zerone/x/knowledge/migrations/v5"
)

// Migrator handles in-place store migrations.
type Migrator struct {
	keeper Keeper
}

// NewMigrator returns a new Migrator.
func NewMigrator(keeper Keeper) Migrator {
	return Migrator{keeper: keeper}
}

// Migrate1to2 migrates from version 1 to version 2.
// Writes a verifiable marker to confirm the migration ran successfully.
func (m Migrator) Migrate1to2(ctx sdk.Context) error {
	return m.keeper.WriteMigrationMarker(ctx, "migration_v2_complete", "true")
}

// Migrate2to3 migrates from version 2 to version 3.
// Backfills R29 param defaults for zero-valued fields after upgrade.
func (m Migrator) Migrate2to3(ctx sdk.Context) error {
	return v3.Migrate(ctx, m.keeper)
}

// Migrate3to4 migrates from version 3 to version 4 (Route B Wave 10).
// Backfills TraceSchema if missing; records a verifiable marker.
// See docs/UPGRADE_PROTOCOL.md for the canonical pattern this follows.
func (m Migrator) Migrate3to4(ctx sdk.Context) error {
	return v4.Migrate(ctx, m.keeper)
}

// Migrate4to5 migrates from version 4 to version 5.
// Drops eleven dead anti-slop / FARM / citation-gaming params from Params
// (defined but never read by any keeper path) and records a verifiable marker.
func (m Migrator) Migrate4to5(ctx sdk.Context) error {
	return v5.Migrate(ctx, m.keeper)
}

// Migrate5to6 marks the coordinated activation boundary for the consolidation
// safety release. The release adds consensus behavior and message surfaces but
// does not require an in-place rewrite of existing knowledge records.
func (m Migrator) Migrate5to6(ctx sdk.Context) error {
	return m.keeper.WriteMigrationMarker(ctx, "migration_v6_complete", "true")
}

// Migrate6to7 repairs only the derived survival deadline index. Pending reward
// bytes, economic parameters, balances, and existing schedules remain intact.
func (m Migrator) Migrate6to7(ctx sdk.Context) error {
	if err := survivalmigration.Require(ctx); err != nil {
		return err
	}
	cache, write := ctx.CacheContext()
	if err := m.keeper.RebuildSurvivalDeadlineIndex(cache); err != nil {
		return err
	}
	if err := m.keeper.WriteMigrationMarker(cache, "migration_v7_complete", "true"); err != nil {
		return err
	}
	write()
	return nil
}

// Migrate7to8 enables record integrity only under its named application owner.
// Existing claims, rounds and history retain their recorded legacy semantics.
func (m Migrator) Migrate7to8(ctx sdk.Context) error {
	if err := recordmigration.Require(ctx); err != nil {
		return err
	}
	cache, write := ctx.CacheContext()
	if err := m.keeper.ValidateRecordIntegrityActivation(cache); err != nil {
		return err
	}
	if err := m.keeper.ValidateKnowledgeHistoryState(cache); err != nil {
		return err
	}
	if err := m.keeper.EnableRecordIntegrity(cache); err != nil {
		return err
	}
	if err := m.keeper.WriteMigrationMarker(cache, "migration_v8_complete", "true"); err != nil {
		return err
	}
	write()
	return nil
}

// Migrate8to9 changes future admission terms without rewriting old obligations.
func (m Migrator) Migrate8to9(ctx sdk.Context) error {
	if err := reviewmigration.Require(ctx); err != nil {
		return err
	}
	cache, write := ctx.CacheContext()
	if err := m.keeper.ValidateReviewNeutralityActivation(cache); err != nil {
		return err
	}
	if err := m.keeper.EnableReviewNeutrality(cache); err != nil {
		return err
	}
	if err := m.keeper.WriteMigrationMarker(cache, "migration_v9_complete", "true"); err != nil {
		return err
	}
	write()
	return nil
}

// Migrate9to10 enables retention only for future contradiction submissions. No
// missing historical argument/evidence is inferred or copied from other records.
func (m Migrator) Migrate9to10(ctx sdk.Context) error {
	if err := claimrecordmigration.Require(ctx); err != nil {
		return err
	}
	cache, write := ctx.CacheContext()
	enabled, err := m.keeper.ClaimRecordsEnabled(cache)
	if err != nil || enabled {
		return fmt.Errorf("claim records requires unenabled predecessor: %v", err)
	}
	_, marked, err := m.keeper.ReadMigrationMarkerPresenceChecked(cache, "migration_v10_complete")
	if err != nil || marked {
		return fmt.Errorf("claim records predecessor already marked or unreadable: %v", err)
	}
	if err := m.keeper.EnableClaimRecords(cache); err != nil {
		return err
	}
	if err := m.keeper.WriteMigrationMarker(cache, "migration_v10_complete", "true"); err != nil {
		return err
	}
	write()
	return nil
}

// Migrate10to11 selects explicit funding terms for future admissions. Existing
// claims, payment plans and balances are inventoried but never rewritten.
func (m Migrator) Migrate10to11(ctx sdk.Context) error {
	if err := fundsettlementmigration.Require(ctx); err != nil {
		return err
	}
	cache, write := ctx.CacheContext()
	if err := m.keeper.ValidateFundSettlementActivation(cache); err != nil {
		return err
	}
	_, marked, err := m.keeper.ReadMigrationMarkerPresenceChecked(cache, "migration_v11_complete")
	if err != nil || marked {
		return fmt.Errorf("fund settlement predecessor already marked or unreadable: %v", err)
	}
	if err := m.keeper.EnableFundSettlement(cache); err != nil {
		return err
	}
	if err := m.keeper.WriteMigrationMarker(cache, "migration_v11_complete", "true"); err != nil {
		return err
	}
	write()
	return nil
}
