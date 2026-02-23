package keeper

import (
	"context"
	"fmt"
	"math/big"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/x/compute_pool/types"
)

// BeginBlocker runs at the beginning of each block to manage provider lifecycle.
func (k Keeper) BeginBlocker(ctx context.Context) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	currentBlock := uint64(sdkCtx.BlockHeight())
	params := k.GetParams(ctx)

	// Collect providers that need modification to avoid mutating during iteration.
	var toUpdate []*types.ComputeProvider
	var toDelete []string
	var toRefund []struct {
		address string
		stake   string
	}

	k.IterateProviders(ctx, func(p *types.ComputeProvider) bool {
		switch p.Status {
		case "active":
			// 1. Check liveness: jail providers that missed heartbeat.
			if p.LastHeartbeat+params.HeartbeatIntervalBlocks < currentBlock {
				p.Status = "jailed"
				toUpdate = append(toUpdate, p)
				k.Logger(ctx).Info("provider jailed for missed heartbeat",
					"address", p.Address,
					"last_heartbeat", p.LastHeartbeat,
					"current_block", currentBlock,
				)
			} else {
				// 2. Apply pending price changes for active providers.
				if p.PendingPrice != "" && p.PriceChangeAt <= currentBlock {
					p.PricePerCu = p.PendingPrice
					p.PendingPrice = ""
					p.PriceChangeAt = 0
					toUpdate = append(toUpdate, p)
					k.Logger(ctx).Info("provider price updated",
						"address", p.Address,
						"new_price", p.PricePerCu,
					)
				}
			}

		case "unbonding":
			// 3. Complete unbondings: refund stake and remove provider.
			if p.UnbondingAt+params.ProviderUnbondingBlocks <= currentBlock {
				toDelete = append(toDelete, p.Address)
				toRefund = append(toRefund, struct {
					address string
					stake   string
				}{address: p.Address, stake: p.Stake})
			}
		}
		return false
	})

	// Apply updates and emit events.
	for _, p := range toUpdate {
		k.SetProvider(ctx, p)
		if p.Status == "jailed" {
			sdkCtx.EventManager().EmitEvent(
				sdk.NewEvent("zerone.compute_pool.provider_jailed",
					sdk.NewAttribute("address", p.Address),
					sdk.NewAttribute("last_heartbeat", fmt.Sprintf("%d", p.LastHeartbeat)),
					sdk.NewAttribute("block_height", fmt.Sprintf("%d", currentBlock)),
				),
			)
		} else if p.PendingPrice == "" { // price was just applied
			sdkCtx.EventManager().EmitEvent(
				sdk.NewEvent("zerone.compute_pool.price_applied",
					sdk.NewAttribute("address", p.Address),
					sdk.NewAttribute("new_price", p.PricePerCu),
				),
			)
		}
	}

	// Process unbonding completions: refund stake and delete provider.
	for i, addr := range toDelete {
		refund := toRefund[i]
		stakeAmt := new(big.Int)
		if _, ok := stakeAmt.SetString(refund.stake, 10); ok && stakeAmt.Sign() > 0 {
			providerAddr, err := sdk.AccAddressFromBech32(refund.address)
			if err == nil {
				coins := sdk.NewCoins(sdk.NewCoin("uzrn", sdkmath.NewIntFromBigInt(stakeAmt)))
				if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, providerAddr, coins); err != nil {
					k.Logger(ctx).Error("failed to refund stake on unbonding",
						"address", refund.address,
						"error", err,
					)
					continue
				}
			}
		}
		k.DeleteProvider(ctx, addr)

		sdkCtx.EventManager().EmitEvent(
			sdk.NewEvent(
				"zerone.compute_pool.provider_removed",
				sdk.NewAttribute("address", addr),
				sdk.NewAttribute("stake_refunded", fmt.Sprintf("%s", refund.stake)),
			),
		)
	}

	return nil
}
