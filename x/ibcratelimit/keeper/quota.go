package keeper

import (
	"context"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/x/ibcratelimit/types"
)

// maybeResetWindow resets the rate limit counters if the current window has expired.
func (k Keeper) maybeResetWindow(ctx context.Context, rl *types.RateLimit) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	currentHeight := uint64(sdkCtx.BlockHeight())

	if currentHeight >= rl.WindowStart+rl.WindowBlocks {
		rl.CurrentSend = "0"
		rl.CurrentRecv = "0"
		rl.WindowStart = currentHeight
	}
}

// CheckAndUpdateSendQuota checks and updates the send quota for a given channel/denom.
// Returns an error if the send rate limit would be exceeded.
func (k Keeper) CheckAndUpdateSendQuota(ctx context.Context, channelID, denom string, amount *big.Int) error {
	params := k.GetParams(ctx)
	if !params.Enabled {
		return nil
	}

	rl, found := k.GetRateLimit(ctx, channelID, denom)
	if !found {
		return nil // no rate limit configured — allow
	}

	k.maybeResetWindow(ctx, rl)

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
	k.SetRateLimit(ctx, rl)
	return nil
}

// CheckAndUpdateRecvQuota checks and updates the receive quota for a given channel/denom.
// Returns an error if the receive rate limit would be exceeded.
func (k Keeper) CheckAndUpdateRecvQuota(ctx context.Context, channelID, denom string, amount *big.Int) error {
	params := k.GetParams(ctx)
	if !params.Enabled {
		return nil
	}

	rl, found := k.GetRateLimit(ctx, channelID, denom)
	if !found {
		return nil // no rate limit configured — allow
	}

	k.maybeResetWindow(ctx, rl)

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
	k.SetRateLimit(ctx, rl)
	return nil
}

// ReverseSendQuota decrements the send counter when a packet is refunded (timeout or error ack).
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
	k.SetRateLimit(ctx, rl)
}

// ResetExpiredWindows iterates all rate limits and resets those whose window has expired.
// Called in BeginBlock.
func (k Keeper) ResetExpiredWindows(ctx context.Context) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	currentHeight := uint64(sdkCtx.BlockHeight())

	rateLimits := k.GetAllRateLimits(ctx)
	for _, rl := range rateLimits {
		if currentHeight >= rl.WindowStart+rl.WindowBlocks {
			rl.CurrentSend = "0"
			rl.CurrentRecv = "0"
			rl.WindowStart = currentHeight
			k.SetRateLimit(ctx, rl)
		}
	}
}
