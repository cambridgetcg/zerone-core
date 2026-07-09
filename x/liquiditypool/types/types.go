package types

import (
	"fmt"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	// ZRNDenom is the native staking denom every pool must include on one
	// side: zerone pools are ZRN-quoted by design.
	ZRNDenom = "uzrn"

	// MaxSwapFeeBps caps the per-pool swap fee at creation: 10% on the 1M
	// bps scale. Higher fees are griefing pools, not markets. Zero is
	// allowed (= use the module default).
	MaxSwapFeeBps = 100_000
)

// ValidateBasic performs stateless validation for MsgCreatePool.
func (m *MsgCreatePool) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Creator); err != nil {
		return fmt.Errorf("invalid creator address: %w", err)
	}
	if m.DenomA == "" || m.DenomB == "" {
		return ErrInvalidDenom
	}
	if m.DenomA == m.DenomB {
		return ErrSameDenom
	}
	if m.DenomA != ZRNDenom && m.DenomB != ZRNDenom {
		return ErrMissingZRNSide
	}
	amtA := new(big.Int)
	if _, ok := amtA.SetString(m.AmountA, 10); !ok || amtA.Sign() <= 0 {
		return ErrZeroAmount
	}
	amtB := new(big.Int)
	if _, ok := amtB.SetString(m.AmountB, 10); !ok || amtB.Sign() <= 0 {
		return ErrZeroAmount
	}
	if m.SwapFeeBps > MaxSwapFeeBps {
		return ErrInvalidSwapFee
	}
	return nil
}

// ValidateBasic performs stateless validation for MsgSwap.
func (m *MsgSwap) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Sender); err != nil {
		return fmt.Errorf("invalid sender address: %w", err)
	}
	if m.PoolId == "" {
		return ErrPoolNotFound
	}
	if m.TokenInDenom == "" {
		return ErrInvalidDenom
	}
	amt := new(big.Int)
	if _, ok := amt.SetString(m.TokenInAmount, 10); !ok || amt.Sign() <= 0 {
		return ErrZeroAmount
	}
	return nil
}

// ValidateBasic performs stateless validation for MsgAddLiquidity.
func (m *MsgAddLiquidity) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Sender); err != nil {
		return fmt.Errorf("invalid sender address: %w", err)
	}
	if m.PoolId == "" {
		return ErrPoolNotFound
	}
	amtA := new(big.Int)
	if _, ok := amtA.SetString(m.AmountA, 10); !ok || amtA.Sign() <= 0 {
		return ErrZeroAmount
	}
	amtB := new(big.Int)
	if _, ok := amtB.SetString(m.AmountB, 10); !ok || amtB.Sign() <= 0 {
		return ErrZeroAmount
	}
	return nil
}

// ValidateBasic performs stateless validation for MsgRemoveLiquidity.
func (m *MsgRemoveLiquidity) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Sender); err != nil {
		return fmt.Errorf("invalid sender address: %w", err)
	}
	if m.PoolId == "" {
		return ErrPoolNotFound
	}
	lp := new(big.Int)
	if _, ok := lp.SetString(m.LpTokens, 10); !ok || lp.Sign() <= 0 {
		return ErrZeroAmount
	}
	return nil
}
