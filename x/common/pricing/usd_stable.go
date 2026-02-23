package pricing

// CalculateUSDStablePrice converts a USD target price to uZRN given the current
// ZRN/USD price. All values are in micro-units (6-decimal fixed-point).
//
// Returns 0 when zrnPriceMicroUSD == 0 (oracle unavailable signal).
// Result is clamped to [minUZRN, maxUZRN].
func CalculateUSDStablePrice(targetMicroUSD, zrnPriceMicroUSD, minUZRN, maxUZRN uint64) uint64 {
	if zrnPriceMicroUSD == 0 {
		return 0
	}

	// uZRN = targetMicroUSD * 1_000_000 / zrnPriceMicroUSD
	result := targetMicroUSD * 1_000_000 / zrnPriceMicroUSD

	if result < minUZRN {
		result = minUZRN
	}
	if result > maxUZRN {
		result = maxUZRN
	}

	return result
}
