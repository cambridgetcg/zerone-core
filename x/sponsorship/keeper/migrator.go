package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/x/sponsorship/types"
)

// Migrator handles sponsorship's in-place consensus store migrations.
type Migrator struct{ keeper Keeper }

func NewMigrator(keeper Keeper) Migrator { return Migrator{keeper: keeper} }

// Migrate1to2 installs permanent fact tombstones for every historical payout.
// Legacy orders intentionally remain without WorkContract: v2 fulfillment
// refuses them, while CancelBountyOrder can still return their escrow.
func (m Migrator) Migrate1to2(ctx sdk.Context) error {
	params := m.keeper.GetParams(ctx)
	// v1 did not have a consensus hard cap. Clamp an otherwise valid legacy
	// parameter to the v2 bound, but refuse activation if live state itself
	// exceeds the bound: lazy sponsor pruning must remain O(256).
	if params.MaxActiveBountiesPerSponsor > types.MaxActiveBountiesPerSponsorHardCap {
		params.MaxActiveBountiesPerSponsor = types.MaxActiveBountiesPerSponsorHardCap
	}
	if err := params.Validate(); err != nil {
		return fmt.Errorf("invalid sponsorship params for v2: %w", err)
	}

	orders := m.keeper.GetAllBountyOrders(ctx)
	activeBySponsor := make(map[string]uint32)
	legacySponsorAliases := make(map[string]string)
	// v1 accepted any base-10 string understood by big.Int (for example
	// "001" and "+1"). Its genesis validator also admitted equivalent
	// escrow_remaining spellings. Normalize both before v2's canonical sdk.Int
	// validation becomes active.
	for _, order := range orders {
		canonicalSponsor, err := types.CanonicalAccountAddress(order.Sponsor)
		if err != nil {
			return fmt.Errorf("bounty %s has invalid sponsor %q: %w", order.Id, order.Sponsor, err)
		}
		if order.WorkContract == nil {
			legacySponsorAliases[order.Id] = order.Sponsor
			order.Sponsor = canonicalSponsor
		} else if order.Sponsor != canonicalSponsor {
			return fmt.Errorf("bound bounty %s has noncanonical sponsor %q", order.Id, order.Sponsor)
		}
		if order.WorkContract == nil {
			price, err := types.NormalizeLegacyPositiveAmount(order.PricePerArtifact)
			if err != nil {
				return fmt.Errorf("legacy bounty %s has invalid price_per_artifact %q", order.Id, order.PricePerArtifact)
			}
			remaining, err := types.NormalizeLegacyNonNegativeAmount(order.EscrowRemaining)
			if err != nil {
				return fmt.Errorf("legacy bounty %s has invalid escrow_remaining %q", order.Id, order.EscrowRemaining)
			}
			order.PricePerArtifact = price
			order.EscrowRemaining = remaining
		}
		if order.Status == types.BountyStatus_BOUNTY_STATUS_ACTIVE {
			activeBySponsor[canonicalSponsor]++
			if activeBySponsor[canonicalSponsor] > params.MaxActiveBountiesPerSponsor {
				return fmt.Errorf("legacy sponsor %s has %d active bounties, v2 max is %d",
					canonicalSponsor, activeBySponsor[canonicalSponsor], params.MaxActiveBountiesPerSponsor)
			}
		}
	}
	fulfillments := m.keeper.GetAllFulfillments(ctx)
	for _, fulfillment := range fulfillments {
		amount, err := types.NormalizeLegacyPositiveAmount(fulfillment.AmountPaid)
		if err != nil {
			return fmt.Errorf("legacy fulfillment %s/%s has invalid amount_paid %q",
				fulfillment.BountyId, fulfillment.FactId, fulfillment.AmountPaid)
		}
		fulfillment.AmountPaid = amount
	}

	// All fallible compatibility preflight above completed before store writes.
	m.keeper.SetParams(ctx, params)
	for _, order := range orders {
		if rawSponsor := legacySponsorAliases[order.Id]; rawSponsor != "" && rawSponsor != order.Sponsor {
			_ = m.keeper.storeService.OpenKVStore(ctx).Delete(types.ActiveSponsorIndexKey(rawSponsor, order.Id))
		}
		m.keeper.SetBountyOrder(ctx, order)
		m.keeper.indexActiveBounty(ctx, order)
	}
	for _, fulfillment := range fulfillments {
		// SetFulfillment both normalizes the exported record and installs the
		// permanent fact tombstone. Receipt/artifact/nullifier remain empty for v1.
		m.keeper.SetFulfillment(ctx, fulfillment)
	}
	liability, err := m.keeper.DerivedEscrowLiability(ctx)
	if err != nil {
		return fmt.Errorf("derive v2 escrow liability: %w", err)
	}
	if err := m.keeper.SetEscrowLiability(ctx, liability); err != nil {
		return fmt.Errorf("store v2 escrow liability: %w", err)
	}
	if err := m.keeper.EnsureEscrowAccounting(ctx); err != nil {
		return err
	}
	return m.keeper.EnsureEscrowSolvent(ctx)
}
