package keeper

import (
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/x/partnerships/types"
)

// Deliberation window blocks by amount tier.
var deliberationWindowBlocks = map[types.AmountTier]uint64{
	types.AmountTierMicro:  22,
	types.AmountTierSmall:  77,
	types.AmountTierMedium: 222,
	types.AmountTierLarge:  777,
	types.AmountTierMajor:  2222,
}

const maxCooldownBlocks = uint64(11111)

var bpsDenom = big.NewInt(1000000)

// CalculateAmountTier determines the deliberation tier based on the operation
// amount as a percentage of the common pot balance.
func (k Keeper) CalculateAmountTier(amount string, potBalance string) types.AmountTier {
	amtInt := new(big.Int)
	if _, ok := amtInt.SetString(amount, 10); !ok || amtInt.Sign() <= 0 {
		return types.AmountTierMicro
	}

	potInt := new(big.Int)
	if _, ok := potInt.SetString(potBalance, 10); !ok || potInt.Sign() <= 0 {
		return types.AmountTierMajor
	}

	pctBps := new(big.Int).Mul(amtInt, bpsDenom)
	pctBps.Div(pctBps, potInt)
	pct := pctBps.Uint64()

	switch {
	case pct < 10000: // <1%
		return types.AmountTierMicro
	case pct < 50000: // 1-5%
		return types.AmountTierSmall
	case pct < 220000: // 5-22%
		return types.AmountTierMedium
	case pct < 550000: // 22-55%
		return types.AmountTierLarge
	default: // >55%
		return types.AmountTierMajor
	}
}

// GetDeliberationWindow returns the window duration in blocks for a given tier.
func (k Keeper) GetDeliberationWindow(tier types.AmountTier) uint64 {
	if w, ok := deliberationWindowBlocks[tier]; ok {
		return w
	}
	return deliberationWindowBlocks[types.AmountTierMicro]
}

// CalculateCooldown computes the cooldown duration after a rejection.
// Formula: baseCooldown * 2^rejectionCount, capped at maxCooldownBlocks.
func (k Keeper) CalculateCooldown(ctx sdk.Context, rejectionCount uint32) uint64 {
	params := k.GetParams(ctx)
	base := params.BaseCooldownBlocks

	cooldown := base
	for i := uint32(0); i < rejectionCount; i++ {
		cooldown *= 2
		if cooldown > maxCooldownBlocks {
			return maxCooldownBlocks
		}
	}
	return cooldown
}

// ValidateCounterProposal checks that the counter-proposal chain depth
// does not exceed the maximum allowed depth.
func (k Keeper) ValidateCounterProposal(ctx sdk.Context, parentOp *types.ConsensusOperation) error {
	params := k.GetParams(ctx)
	if parentOp.Deliberation.ChainDepth >= params.MaxCounterProposalDepth {
		return types.ErrCounterProposalDepth
	}
	return nil
}

// CreateDeliberationState builds a DeliberationState for a new operation.
func (k Keeper) CreateDeliberationState(
	ctx sdk.Context,
	amount string,
	potBalance string,
	rationale string,
	counterProposalOf string,
	chainDepth uint32,
) *types.DeliberationState {
	tier := k.CalculateAmountTier(amount, potBalance)
	window := k.GetDeliberationWindow(tier)
	currentBlock := uint64(ctx.BlockHeight())

	return &types.DeliberationState{
		AmountTier:        tier,
		FloorEndsAt:       currentBlock + window/2,
		WindowEndsAt:      currentBlock + window,
		Rationale:         rationale,
		CounterProposalOf: counterProposalOf,
		ChainDepth:        chainDepth,
	}
}
