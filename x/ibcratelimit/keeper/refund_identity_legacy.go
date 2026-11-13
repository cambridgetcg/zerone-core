package keeper

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/x/ibcratelimit/types"
)

// Frozen marker-off bodies from 77521a0daf14420836483ae9d8ea3fbcc1ae0d40.
// Only names/calls to other frozen bodies differ. Do not route these through
// the identity-aware setters: ignored write errors and gas are consensus too.

// SetRateLimit preserves the pre-activation API for legacy consumers.
// Deprecated: legacy dispatch/tests only. Production entry points must read
// RefundIdentityEnabled and pass it to SetRateLimitForActivation.
func (k Keeper) SetRateLimit(ctx context.Context, rl *types.RateLimit) {
	k.SetRateLimitLegacy(ctx, rl)
}

// DeletePacketFlow preserves pre-activation ignored cleanup errors.
// Deprecated: legacy dispatch/tests only; use DeletePacketFlowForActivation.
func (k Keeper) DeletePacketFlow(ctx context.Context, channelID string, sequence uint64) {
	k.DeletePacketFlowLegacy(ctx, channelID, sequence)
}

// CheckAndUpdateRecvQuota preserves the pre-activation counter API.
// Deprecated: legacy dispatch/tests only; use CheckAndUpdateRecvQuotaForActivation.
func (k Keeper) CheckAndUpdateRecvQuota(ctx context.Context, channelID, denom string, amount *big.Int) error {
	return k.CheckAndUpdateRecvQuotaLegacy(ctx, channelID, denom, amount)
}

// ResetExpiredWindows preserves the pre-activation block helper API.
// Deprecated: legacy dispatch/tests only; use ResetExpiredWindowsForActivation.
func (k Keeper) ResetExpiredWindows(ctx context.Context) {
	k.ResetExpiredWindowsLegacy(ctx)
}

func (k Keeper) SetRateLimitLegacy(ctx context.Context, rl *types.RateLimit) {
	kvStore := k.storeService.OpenKVStore(ctx)
	key := rateLimitKey(rl.ChannelId, rl.Denom)
	bz, err := json.Marshal(rl)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal rate limit: %v", err))
	}
	_ = kvStore.Set(key, bz)
}

func (k Keeper) DeletePacketFlowLegacy(ctx context.Context, channelID string, sequence uint64) {
	kvStore := k.storeService.OpenKVStore(ctx)
	_ = kvStore.Delete(packetFlowKey(channelID, sequence))
}

func (k Keeper) maybeResetWindowLegacy(ctx context.Context, rl *types.RateLimit) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	currentHeight := uint64(sdkCtx.BlockHeight())

	if currentHeight >= rl.WindowStart+rl.WindowBlocks {
		rl.CurrentSend = "0"
		rl.CurrentRecv = "0"
		rl.WindowStart = currentHeight
	}
}

// CheckAndUpdateSendQuota is the pre-activation counter primitive.
// Deprecated: legacy dispatch/tests only. Activated packet senders must retain
// the receipt from ChargeSendQuota, not combine this with SetPacketFlow.
func (k Keeper) CheckAndUpdateSendQuota(ctx context.Context, channelID, denom string, amount *big.Int) error {
	params := k.GetParams(ctx)
	if !params.Enabled {
		return nil
	}

	rl, found := k.GetRateLimit(ctx, channelID, denom)
	if !found {
		return nil // no rate limit configured — allow
	}

	k.maybeResetWindowLegacy(ctx, rl)

	currentSend := new(big.Int)
	if rl.CurrentSend != "" {
		currentSend.SetString(rl.CurrentSend, 10)
	}

	maxSend := new(big.Int)
	maxSend.SetString(rl.MaxSend, 10)

	newSend := new(big.Int).Add(currentSend, amount)
	if newSend.Cmp(maxSend) > 0 {
		return types.ErrSendRateExceeded.Wrapf(
			"channel %s denom %s: sending %s would bring total to %s, max is %s",
			channelID, denom, amount.String(), newSend.String(), maxSend.String(),
		)
	}

	rl.CurrentSend = newSend.String()
	k.SetRateLimitLegacy(ctx, rl)
	return nil
}

func (k Keeper) CheckAndUpdateRecvQuotaLegacy(ctx context.Context, channelID, denom string, amount *big.Int) error {
	params := k.GetParams(ctx)
	if !params.Enabled {
		return nil
	}

	rl, found := k.GetRateLimit(ctx, channelID, denom)
	if !found {
		return nil // no rate limit configured — allow
	}

	k.maybeResetWindowLegacy(ctx, rl)

	currentRecv := new(big.Int)
	if rl.CurrentRecv != "" {
		currentRecv.SetString(rl.CurrentRecv, 10)
	}

	maxRecv := new(big.Int)
	maxRecv.SetString(rl.MaxRecv, 10)

	newRecv := new(big.Int).Add(currentRecv, amount)
	if newRecv.Cmp(maxRecv) > 0 {
		return types.ErrRecvRateExceeded.Wrapf(
			"channel %s denom %s: receiving %s would bring total to %s, max is %s",
			channelID, denom, amount.String(), newRecv.String(), maxRecv.String(),
		)
	}

	rl.CurrentRecv = newRecv.String()
	k.SetRateLimitLegacy(ctx, rl)
	return nil
}

// ReverseSendQuota is the pre-activation raw counter primitive.
// Deprecated: legacy dispatch/tests only. Activated refunds must prove their
// identity through ReverseSendCharge or ReversePacketFlow.
func (k Keeper) ReverseSendQuota(ctx context.Context, channelID, denom string, amount *big.Int) {
	rl, found := k.GetRateLimit(ctx, channelID, denom)
	if !found {
		return
	}

	currentSend := new(big.Int)
	if rl.CurrentSend != "" {
		currentSend.SetString(rl.CurrentSend, 10)
	}

	newSend := new(big.Int).Sub(currentSend, amount)
	if newSend.Sign() < 0 {
		newSend.SetInt64(0)
	}

	rl.CurrentSend = newSend.String()
	k.SetRateLimitLegacy(ctx, rl)
}

func (k Keeper) ResetExpiredWindowsLegacy(ctx context.Context) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	currentHeight := uint64(sdkCtx.BlockHeight())

	rateLimits := k.GetAllRateLimits(ctx)
	for _, rl := range rateLimits {
		if currentHeight >= rl.WindowStart+rl.WindowBlocks {
			rl.CurrentSend = "0"
			rl.CurrentRecv = "0"
			rl.WindowStart = currentHeight
			k.SetRateLimitLegacy(ctx, rl)
		}
	}
}
