package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	// ZRNDenom is the native staking denom every pool must include on one
	// side: zerone pools are ZRN-quoted by design.
	ZRNDenom = "uzrn"

	// MaxSwapFeeBps caps the governance-controlled default swap fee at 10%
	// on the 1M bps scale. Pool creation must pass zero and inherit that
	// default so creators cannot override governance policy.
	MaxSwapFeeBps = 100_000

	// MaxPoolsCap bounds all per-block and invariant work over open pools.
	MaxPoolsCap = 64

	// MaxPoolRecordsCap bounds immutable tombstones, export/invariant scans and
	// pagination over the chain's full pool lifetime. IDs are never reused.
	MaxPoolRecordsCap = 10_000

	// MaxTWAPWindowBlocks bounds retained observations per open pool.
	MaxTWAPWindowBlocks = 10_000

	// MaxAdmissionEntries bounds governance allowlist lookup/query size.
	MaxAdmissionEntries = 32
)

func ValidatePoolDenom(denom string) error {
	if err := sdk.ValidateDenom(denom); err != nil {
		return ErrInvalidDenom.Wrapf("%q: %v", denom, err)
	}
	return nil
}

func IsOpenPoolStatus(status PoolStatus) bool {
	switch status {
	case PoolStatus_POOL_STATUS_ACTIVE,
		PoolStatus_POOL_STATUS_SWAPS_PAUSED,
		PoolStatus_POOL_STATUS_EXIT_ONLY:
		return true
	default:
		return false
	}
}

func CanSwap(status PoolStatus) bool {
	return status == PoolStatus_POOL_STATUS_ACTIVE
}

func CanAddLiquidity(status PoolStatus) bool {
	return status == PoolStatus_POOL_STATUS_ACTIVE ||
		status == PoolStatus_POOL_STATUS_SWAPS_PAUSED
}

func CanRemoveLiquidity(status PoolStatus) bool {
	return IsOpenPoolStatus(status)
}

// ValidateBasic performs stateless validation for MsgCreatePool.
func (m *MsgCreatePool) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Creator); err != nil {
		return fmt.Errorf("invalid creator address: %w", err)
	}
	if err := ValidatePoolDenom(m.DenomA); err != nil {
		return err
	}
	if err := ValidatePoolDenom(m.DenomB); err != nil {
		return err
	}
	if m.DenomA == m.DenomB {
		return ErrSameDenom
	}
	if m.DenomA != ZRNDenom && m.DenomB != ZRNDenom {
		return ErrMissingZRNSide
	}
	if _, err := ParsePositiveAmount(m.AmountA); err != nil {
		return err
	}
	if _, err := ParsePositiveAmount(m.AmountB); err != nil {
		return err
	}
	if m.SwapFeeBps != 0 {
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
	if _, err := ParsePoolID(m.PoolId); err != nil {
		return ErrPoolNotFound.Wrap(err.Error())
	}
	if err := ValidatePoolDenom(m.TokenInDenom); err != nil {
		return err
	}
	if _, err := ParsePositiveAmount(m.TokenInAmount); err != nil {
		return err
	}
	if _, err := ParseOptionalPositiveAmount(m.MinTokenOut); err != nil {
		return fmt.Errorf("invalid min_token_out: %w", err)
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
	if _, err := ParsePoolID(m.PoolId); err != nil {
		return ErrPoolNotFound.Wrap(err.Error())
	}
	if _, err := ParsePositiveAmount(m.AmountA); err != nil {
		return err
	}
	if _, err := ParsePositiveAmount(m.AmountB); err != nil {
		return err
	}
	if _, err := ParseOptionalPositiveAmount(m.MinLpTokens); err != nil {
		return fmt.Errorf("invalid min_lp_tokens: %w", err)
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
	if _, err := ParsePoolID(m.PoolId); err != nil {
		return ErrPoolNotFound.Wrap(err.Error())
	}
	if _, err := ParsePositiveAmount(m.LpTokens); err != nil {
		return err
	}
	if _, err := ParseOptionalPositiveAmount(m.MinAmountA); err != nil {
		return fmt.Errorf("invalid min_amount_a: %w", err)
	}
	if _, err := ParseOptionalPositiveAmount(m.MinAmountB); err != nil {
		return fmt.Errorf("invalid min_amount_b: %w", err)
	}
	return nil
}

func (m *MsgUpdateParams) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Authority); err != nil {
		return fmt.Errorf("invalid authority address: %w", err)
	}
	if m.Params == nil {
		return fmt.Errorf("params cannot be nil")
	}
	return m.Params.Validate()
}

func (m *MsgSetPoolStatus) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Authority); err != nil {
		return fmt.Errorf("invalid authority address: %w", err)
	}
	if _, err := ParsePoolID(m.PoolId); err != nil {
		return ErrPoolNotFound.Wrap(err.Error())
	}
	switch m.Status {
	case PoolStatus_POOL_STATUS_ACTIVE,
		PoolStatus_POOL_STATUS_SWAPS_PAUSED,
		PoolStatus_POOL_STATUS_EXIT_ONLY:
		return nil
	case PoolStatus_POOL_STATUS_CLOSED:
		return ErrInvalidPoolStatus.Wrap("CLOSED is produced only by the final LP exit")
	default:
		return ErrInvalidPoolStatus
	}
}
