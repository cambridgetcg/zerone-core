package types

import (
	"fmt"
	"math/big"
	"strconv"
	"strings"

	sdkmath "cosmossdk.io/math"
)

// ParsePositiveAmount parses a canonical, positive base-10 amount that fits
// in sdkmath.Int. Keeping this check at every message/query boundary prevents
// sdkmath.NewIntFromBigInt from panicking on oversized user input.
func ParsePositiveAmount(value string) (*big.Int, error) {
	return parseCanonicalAmount(value, false)
}

// ParseNonNegativeAmount parses a canonical, non-negative base-10 amount that
// fits in sdkmath.Int. It is intended for stored state and parameters.
func ParseNonNegativeAmount(value string) (*big.Int, error) {
	return parseCanonicalAmount(value, true)
}

// ParseCumulativeAmount parses internal TWAP cumulative values. Their
// deterministic upper bound is wider than an sdkmath.Int because a 256-bit
// reserve ratio is multiplied by the price scale and retained block window.
func ParseCumulativeAmount(value string) (*big.Int, error) {
	return parseCanonicalBigInt(value, true, 512)
}

// ParseOptionalPositiveAmount accepts an omitted protection value, but any
// supplied value must be canonical and positive. Malformed, negative and zero
// strings never silently disable slippage protection.
func ParseOptionalPositiveAmount(value string) (*big.Int, error) {
	if value == "" {
		return nil, nil
	}
	return ParsePositiveAmount(value)
}

func parseCanonicalAmount(value string, allowZero bool) (*big.Int, error) {
	return parseCanonicalBigInt(value, allowZero, sdkmath.MaxBitLen)
}

func parseCanonicalBigInt(value string, allowZero bool, maxBitLen int) (*big.Int, error) {
	if value == "" {
		return nil, ErrInvalidAmount.Wrap("amount cannot be empty")
	}
	v, ok := new(big.Int).SetString(value, 10)
	if !ok || v.String() != value {
		return nil, ErrInvalidAmount.Wrapf("%q is not a canonical base-10 integer", value)
	}
	if v.Sign() < 0 || (!allowZero && v.Sign() == 0) {
		if allowZero {
			return nil, ErrInvalidAmount.Wrap("amount cannot be negative")
		}
		return nil, ErrZeroAmount
	}
	if v.BitLen() > maxBitLen {
		return nil, ErrInvalidAmount.Wrapf("amount exceeds %d-bit integer limit", maxBitLen)
	}
	return v, nil
}

// ParsePoolID validates and extracts the monotonically increasing numeric
// suffix from the canonical "pool-N" identifier.
func ParsePoolID(poolID string) (uint64, error) {
	raw, ok := strings.CutPrefix(poolID, "pool-")
	if !ok || raw == "" {
		return 0, fmt.Errorf("invalid pool ID %q", poolID)
	}
	id, err := strconv.ParseUint(raw, 10, 64)
	if err != nil || id == 0 || fmt.Sprintf("pool-%d", id) != poolID {
		return 0, fmt.Errorf("invalid pool ID %q", poolID)
	}
	return id, nil
}

// ParseLPDenom returns the pool ID encoded by the canonical LP denom.
func ParseLPDenom(denom string) (string, error) {
	poolID, ok := strings.CutPrefix(denom, "lp/")
	if !ok {
		return "", fmt.Errorf("invalid liquiditypool LP denom %q", denom)
	}
	if _, err := ParsePoolID(poolID); err != nil {
		return "", fmt.Errorf("invalid liquiditypool LP denom %q: %w", denom, err)
	}
	if LPDenom(poolID) != denom {
		return "", fmt.Errorf("invalid liquiditypool LP denom %q", denom)
	}
	return poolID, nil
}
