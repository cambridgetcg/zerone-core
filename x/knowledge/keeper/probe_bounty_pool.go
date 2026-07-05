package keeper

import (
	"context"
	"fmt"
	"math/big"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// mintCappedUzrn issues `amount` uzrn into module through the chain's single
// cap-gated mint entry point (x/vesting_rewards.MintWithCap), so no knowledge
// emission path can push total supply past the 222,222,222 ZRN cap. Returns the
// amount actually minted (clipped to remaining cap headroom). Falls back to a
// direct mint only when the vesting-rewards keeper is unwired — which happens
// only in isolated unit tests; production always routes through the cap.
func (k Keeper) mintCappedUzrn(ctx context.Context, module string, amount *big.Int) (*big.Int, error) {
	if amount == nil || amount.Sign() <= 0 {
		return new(big.Int), nil
	}
	if k.vestingRewardsKeeper != nil {
		return k.vestingRewardsKeeper.MintWithCap(ctx, module, amount)
	}
	if k.bankKeeper != nil {
		coins := sdk.NewCoins(sdk.NewCoin("uzrn", sdkmath.NewIntFromBigInt(amount)))
		if err := k.bankKeeper.MintCoins(ctx, module, coins); err != nil {
			return nil, err
		}
	}
	return amount, nil
}

// MintToProbeBountyPool issues the per-block allocation into the probe
// bounty pool (Wave 15), respecting the max-pool-size cap. Called from
// BeginBlocker. Failure is logged and non-fatal — pool can refill next
// block; fallback paths (protocol treasury) still cover bonuses even if
// the pool is temporarily empty.
func (k Keeper) MintToProbeBountyPool(ctx context.Context, params *types.Params) {
	if k.bankKeeper == nil || params == nil {
		return
	}
	mintStr := params.ProbeBountyMintPerBlock
	if mintStr == "" || mintStr == "0" {
		return
	}
	mintAmt, ok := new(big.Int).SetString(mintStr, 10)
	if !ok || mintAmt.Sign() <= 0 {
		return
	}

	// Enforce the cap: don't mint if the pool would exceed ProbeBountyMaxPoolSize.
	current := k.ProbeBountyPoolBalance(ctx)
	maxStr := params.ProbeBountyMaxPoolSize
	if maxStr != "" && maxStr != "0" {
		maxAmt, ok := new(big.Int).SetString(maxStr, 10)
		if ok && maxAmt.Sign() > 0 {
			projected := new(big.Int).Add(current, mintAmt)
			if projected.Cmp(maxAmt) > 0 {
				// Mint only up to the cap.
				room := new(big.Int).Sub(maxAmt, current)
				if room.Sign() <= 0 {
					return // already at cap
				}
				mintAmt = room
			}
		}
	}

	minted, err := k.mintCappedUzrn(ctx, types.ProbeBountyPoolModuleName, mintAmt)
	if err != nil {
		k.Logger(ctx).Error("probe bounty pool mint failed", "amount", mintAmt.String(), "err", err)
		return
	}
	if minted.Sign() <= 0 {
		return // supply cap reached — nothing minted this block
	}
	mintAmt = minted // event + accounting reflect the actually-minted amount
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
		"zerone.knowledge.probe_bounty_minted",
		sdk.NewAttribute("amount", mintAmt.String()),
		sdk.NewAttribute("block", fmt.Sprintf("%d", sdkCtx.BlockHeight())),
	))
}

// ProbeBountyPoolBalance returns the current uzrn balance of the pool.
func (k Keeper) ProbeBountyPoolBalance(ctx context.Context) *big.Int {
	if k.bankKeeper == nil {
		return new(big.Int)
	}
	addr := authtypes.NewModuleAddress(types.ProbeBountyPoolModuleName)
	bal := k.bankKeeper.GetBalance(ctx, addr, "uzrn")
	if bal.Amount.IsNil() {
		return new(big.Int)
	}
	return bal.Amount.BigInt()
}

// PayProbeBountyFromPool attempts to pay the challenger `amount` from
// the probe bounty pool. Returns the actually-paid amount (may be less
// than requested if the pool is underfunded). Used by settleChallengeStake
// to draw successful-probe bonuses from the purpose-built pool before
// falling back to the protocol treasury.
func (k Keeper) PayProbeBountyFromPool(ctx context.Context, challenger sdk.AccAddress, amount *big.Int) *big.Int {
	if k.bankKeeper == nil || amount == nil || amount.Sign() <= 0 {
		return new(big.Int)
	}
	available := k.ProbeBountyPoolBalance(ctx)
	paying := new(big.Int).Set(amount)
	if paying.Cmp(available) > 0 {
		paying = available
	}
	if paying.Sign() <= 0 {
		return new(big.Int)
	}
	coins := sdk.NewCoins(sdk.NewCoin("uzrn", sdkmath.NewIntFromBigInt(paying)))
	if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ProbeBountyPoolModuleName, challenger, coins); err != nil {
		k.Logger(ctx).Error("probe bounty payout failed", "err", err)
		return new(big.Int)
	}
	return paying
}

