package keeper

import (
	"fmt"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/x/claiming_pot/types"
)

// Migrator handles in-place claiming-pot store migrations.
type Migrator struct {
	keeper Keeper
}

func NewMigrator(keeper Keeper) Migrator {
	return Migrator{keeper: keeper}
}

// Migrate1to2 charges every pre-existing pot against the shared lifetime
// commitment budget. Version 1 counted bootstrap pots only, so legacy general
// pots would otherwise remain uncharged after an in-place binary upgrade. It
// also raises PotCounter to the highest stored `pot-N` suffix, preventing an
// imported or upgraded store from overwriting an existing general pot.
func (m Migrator) Migrate1to2(ctx sdk.Context) error {
	derivedUnits := uint64(0)
	maxPotSequence := uint64(0)
	var deriveErr error
	m.keeper.IteratePots(ctx, func(pot *types.ClaimingPot) bool {
		units, err := types.PotCommitmentUnits(pot.TotalAmount)
		if err != nil {
			deriveErr = fmt.Errorf("pot %s: %w", pot.Id, err)
			return true
		}
		if ^uint64(0)-derivedUnits < units {
			deriveErr = fmt.Errorf("lifetime commitment units overflow")
			return true
		}
		derivedUnits += units
		if sequence, ok := types.GeneralPotSequence(pot.Id); ok {
			if sequence == ^uint64(0) {
				deriveErr = fmt.Errorf("pot %s leaves no next general-pot id", pot.Id)
				return true
			}
			if sequence > maxPotSequence {
				maxPotSequence = sequence
			}
		}
		return false
	})
	if deriveErr != nil {
		return deriveErr
	}

	committedUnits := m.keeper.GetBootstrapMintedEntries(ctx)
	if derivedUnits > committedUnits {
		committedUnits = derivedUnits
	}
	perUnit, _ := new(big.Int).SetString(types.PerAgentBootstrapUzrn, 10)
	committedAmount := new(big.Int).Mul(new(big.Int).SetUint64(committedUnits), perUnit)
	if committedAmount.Cmp(m.keeper.GetParams(ctx).BootstrapEmissionCap()) > 0 {
		return fmt.Errorf(
			"existing pots commit %s uzrn above bootstrap_emission_cap_uzrn",
			committedAmount,
		)
	}
	m.keeper.SetBootstrapMintedEntries(ctx, committedUnits)

	counter := m.keeper.GetPotCounter(ctx)
	if counter == ^uint64(0) {
		return fmt.Errorf("pot counter exhausted")
	}
	if maxPotSequence > counter {
		m.keeper.SetPotCounter(ctx, maxPotSequence)
	}
	return nil
}
