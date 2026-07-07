package app

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	vestingrewardstypes "github.com/zerone-chain/zerone/x/vesting_rewards/types"
)

var researchFundModuleAddr = authtypes.NewModuleAddress(ResearchFundName)

// allowedResearchFundDepositors holds the raw module-address BYTES permitted to
// deposit into the research fund. Stored as AccAddress (never stringified at
// package scope): calling .String() here would bech32-encode BEFORE app.init()
// seals the "zrn" prefix, poisoning the SDK's process-wide address cache with a
// "cosmos1…" form for these bytes — which corrupts the vesting_rewards module
// account address and sends all producer rewards to an unspendable void.
var allowedResearchFundDepositors = []sdk.AccAddress{
	authtypes.NewModuleAddress(vestingrewardstypes.ModuleName),
}

// ResearchFundRestriction is a bank SendRestriction that rejects any send TO the
// research_fund module account unless it originates from vesting_rewards.
// This enforces the invariant that all research fund deposits flow through
// DepositToResearchFund (which handles the founder 7% split).
// Genesis (block height 0) is exempt to allow initial fund seeding.
func ResearchFundRestriction(ctx context.Context, fromAddr, toAddr sdk.AccAddress, amt sdk.Coins) (sdk.AccAddress, error) {
	if toAddr.Equals(researchFundModuleAddr) {
		sdkCtx := sdk.UnwrapSDKContext(ctx)
		if sdkCtx.BlockHeight() == 0 {
			return toAddr, nil // genesis bypass
		}
		allowed := false
		for _, a := range allowedResearchFundDepositors {
			if fromAddr.Equals(a) {
				allowed = true
				break
			}
		}
		if !allowed {
			return toAddr, fmt.Errorf(
				"unauthorized deposit to research_fund from %s: must route through DepositToResearchFund",
				fromAddr.String(),
			)
		}
	}
	return toAddr, nil
}
