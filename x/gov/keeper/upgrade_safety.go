package keeper

import (
	"fmt"
	"math/big"

	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/x/gov/types"
)

func customUpgradeAuthorityRetiredError() error {
	return errorsmod.Wrap(
		types.ErrUpgradeScheduleFailed,
		"custom x/gov software-upgrade authority is retired; submit the plan through standard SDK governance",
	)
}

// RetireCustomUpgradeLIPs terminalizes a caller-supplied complete snapshot.
// The coordinated app upgrade obtains this list from a complete committed IAVL
// export whose reconstructed subroot is bound to the H-1 app hash.
func (k Keeper) RetireCustomUpgradeLIPs(
	ctx sdk.Context,
	lips []*types.LIP,
) (uint64, error) {
	seen := make(map[string]struct{}, len(lips))
	candidates := make([]*types.LIP, 0, len(lips))
	for _, lip := range lips {
		if lip == nil || lip.Id == "" {
			return 0, errorsmod.Wrap(
				types.ErrUpgradeScheduleFailed,
				"custom upgrade LIP snapshot contains a nil or empty-id record",
			)
		}
		if _, duplicate := seen[lip.Id]; duplicate {
			return 0, errorsmod.Wrapf(
				types.ErrUpgradeScheduleFailed,
				"custom upgrade LIP snapshot contains duplicate id %q",
				lip.Id,
			)
		}
		seen[lip.Id] = struct{}{}
		if lip.Category != types.CategoryUpgrade || types.IsTerminal(lip.Stage) {
			continue
		}
		stakeText := lip.StakedAmount
		if stakeText == "" {
			stakeText = "0"
		}
		stake, ok := new(big.Int).SetString(stakeText, 10)
		if !ok || stake.Sign() < 0 {
			return 0, errorsmod.Wrapf(
				types.ErrUpgradeScheduleFailed,
				"custom upgrade LIP %q has invalid aggregate stake %q",
				lip.Id,
				lip.StakedAmount,
			)
		}
		if stake.Sign() != 0 {
			return 0, errorsmod.Wrapf(
				types.ErrUpgradeScheduleFailed,
				"custom upgrade LIP %q retains %s uzrn without a claimant ledger; reconcile it before authority retirement",
				lip.Id,
				stake,
			)
		}
		candidates = append(candidates, lip)
	}

	var retired uint64
	for _, lip := range candidates {
		lip.Stage = types.StatusFailed
		k.SetLIP(ctx, lip)
		retired++
	}
	if retired != 0 {
		ctx.EventManager().EmitEvent(
			sdk.NewEvent(
				"zerone.gov.custom_upgrade_authority_retired",
				sdk.NewAttribute("retired_count", fmt.Sprintf("%d", retired)),
				sdk.NewAttribute("new_stage", types.StatusFailed),
				sdk.NewAttribute("stake_reconciliation", "manual: legacy state has no per-staker escrow ledger"),
				sdk.NewAttribute("canonical_authority", "cosmos.gov.v1"),
			),
		)
	}
	return retired, nil
}

// validateUpgradePlanForVoting enforces one software-upgrade authority.
// Standard SDK governance owns x/upgrade; custom LIPs remain queryable but
// cannot enter voting as executable software-upgrade proposals.
func (k Keeper) validateUpgradePlanForVoting(
	ctx sdk.Context,
	lip *types.LIP,
	votingEnd uint64,
) error {
	if lip.Category != types.CategoryUpgrade {
		return nil
	}
	_ = ctx
	_ = votingEnd
	return customUpgradeAuthorityRetiredError()
}

// scheduleApprovedUpgrade is a second fail-closed boundary for legacy custom
// upgrade LIPs that were already in voting when this rule activated.
func (k Keeper) scheduleApprovedUpgrade(ctx sdk.Context, lip *types.LIP) (*types.UpgradePlan, error) {
	_ = ctx
	_ = lip
	return nil, customUpgradeAuthorityRetiredError()
}
