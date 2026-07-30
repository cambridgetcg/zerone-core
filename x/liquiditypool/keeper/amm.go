package keeper

import "math/big"

// bpsBasis is the basis points scale (1,000,000 = 100%).
var bpsBasis = big.NewInt(1_000_000)

// CalculateSwapOutput computes the output amount for a constant-product swap.
// Formula uses a scaled denominator so fractional fees still affect price
// even when the whole-coin fee_amount rounds to zero:
//
//	weightedIn = tokenIn * (1_000_000 - feeBps)
//	tokenOut = reserveOut * weightedIn /
//	           (reserveIn * 1_000_000 + weightedIn)
func CalculateSwapOutput(reserveIn, reserveOut, tokenIn *big.Int, feeBps uint64) (tokenOut, feeAmount *big.Int) {
	if reserveIn.Sign() <= 0 || reserveOut.Sign() <= 0 || tokenIn.Sign() <= 0 || feeBps > 1_000_000 {
		return new(big.Int), new(big.Int)
	}

	feeAmount = new(big.Int).Mul(tokenIn, big.NewInt(int64(feeBps)))
	feeAmount.Div(feeAmount, bpsBasis)

	feeMultiplier := new(big.Int).Sub(bpsBasis, new(big.Int).SetUint64(feeBps))
	weightedIn := new(big.Int).Mul(tokenIn, feeMultiplier)
	numerator := new(big.Int).Mul(reserveOut, weightedIn)
	denominator := new(big.Int).Mul(reserveIn, bpsBasis)
	denominator.Add(denominator, weightedIn)

	tokenOut = new(big.Int).Div(numerator, denominator)
	return tokenOut, feeAmount
}

// CalculateLPTokensForDeposit computes LP tokens to mint for a liquidity deposit.
// For initial deposit: LP = sqrt(amountA * amountB)
// For subsequent deposits: LP = totalSupply * min(amountA/reserveA, amountB/reserveB)
func CalculateLPTokensForDeposit(reserveA, reserveB, amountA, amountB, totalSupply *big.Int) *big.Int {
	if totalSupply.Sign() == 0 {
		// Initial deposit: LP = sqrt(amountA * amountB)
		product := new(big.Int).Mul(amountA, amountB)
		return new(big.Int).Sqrt(product)
	}

	// Subsequent deposit: min(amountA/reserveA, amountB/reserveB) * totalSupply
	// Use cross-multiplication to avoid division: compare amountA * reserveB vs amountB * reserveA
	ratioAcross := new(big.Int).Mul(amountA, reserveB)
	ratioBcross := new(big.Int).Mul(amountB, reserveA)

	var minRatioNum, minRatioDen *big.Int
	if ratioAcross.Cmp(ratioBcross) <= 0 {
		minRatioNum = amountA
		minRatioDen = reserveA
	} else {
		minRatioNum = amountB
		minRatioDen = reserveB
	}

	lpTokens := new(big.Int).Mul(totalSupply, minRatioNum)
	lpTokens.Div(lpTokens, minRatioDen)
	return lpTokens
}

// CalculateWithdrawalAmounts computes the underlying assets returned for LP token redemption.
// amountA = reserveA * lpTokens / totalSupply
// amountB = reserveB * lpTokens / totalSupply
func CalculateWithdrawalAmounts(reserveA, reserveB, lpTokens, totalSupply *big.Int) (amountA, amountB *big.Int) {
	amountA = new(big.Int).Mul(reserveA, lpTokens)
	amountA.Div(amountA, totalSupply)

	amountB = new(big.Int).Mul(reserveB, lpTokens)
	amountB.Div(amountB, totalSupply)

	return amountA, amountB
}

// CalculateProportionalDeposit computes the actual deposit amounts to maintain pool ratio.
// Given desired amountA and amountB, returns the actual amounts that maintain the ratio.
func CalculateProportionalDeposit(reserveA, reserveB, desiredA, desiredB *big.Int) (actualA, actualB *big.Int) {
	if reserveA.Sign() == 0 || reserveB.Sign() == 0 {
		return new(big.Int).Set(desiredA), new(big.Int).Set(desiredB)
	}

	// Try using full desiredA, calculate required B
	requiredB := new(big.Int).Mul(desiredA, reserveB)
	requiredB.Div(requiredB, reserveA)

	if requiredB.Cmp(desiredB) <= 0 {
		return new(big.Int).Set(desiredA), requiredB
	}

	// B is the binding constraint; calculate required A
	requiredA := new(big.Int).Mul(desiredB, reserveA)
	requiredA.Div(requiredA, reserveB)

	return requiredA, new(big.Int).Set(desiredB)
}

// CalculateDepositForShares derives both the LP shares and the exact
// proportional assets backing those shares. Asset requirements use ceiling
// division, which prevents a positive one-sided donation from minting zero LP
// tokens when reserves are highly asymmetric.
func CalculateDepositForShares(
	reserveA, reserveB, desiredA, desiredB, totalSupply *big.Int,
) (actualA, actualB, lpTokens *big.Int) {
	zero := func() (*big.Int, *big.Int, *big.Int) {
		return new(big.Int), new(big.Int), new(big.Int)
	}
	if reserveA.Sign() <= 0 || reserveB.Sign() <= 0 ||
		desiredA.Sign() <= 0 || desiredB.Sign() <= 0 || totalSupply.Sign() <= 0 {
		return zero()
	}

	lpTokens = CalculateLPTokensForDeposit(reserveA, reserveB, desiredA, desiredB, totalSupply)
	if lpTokens.Sign() <= 0 {
		return zero()
	}
	actualA = ceilDiv(new(big.Int).Mul(lpTokens, reserveA), totalSupply)
	actualB = ceilDiv(new(big.Int).Mul(lpTokens, reserveB), totalSupply)
	if actualA.Sign() <= 0 || actualB.Sign() <= 0 ||
		actualA.Cmp(desiredA) > 0 || actualB.Cmp(desiredB) > 0 {
		return zero()
	}
	return actualA, actualB, lpTokens
}

func ceilDiv(numerator, denominator *big.Int) *big.Int {
	if numerator.Sign() <= 0 || denominator.Sign() <= 0 {
		return new(big.Int)
	}
	adjusted := new(big.Int).Sub(denominator, big.NewInt(1))
	adjusted.Add(adjusted, numerator)
	return adjusted.Div(adjusted, denominator)
}

// CalculatePriceImpactBps computes the price impact in basis points (1M scale).
// priceImpact = 1 - (tokenOut * reserveIn) / (tokenIn * reserveOut)
func CalculatePriceImpactBps(reserveIn, reserveOut, tokenIn, tokenOut *big.Int) uint64 {
	return CalculatePriceImpactBpsWithFee(reserveIn, reserveOut, tokenIn, tokenOut, 0)
}

// CalculatePriceImpactBpsWithFee measures curve impact against the
// post-fee infinitesimal quote, so the configured spread is not mislabeled as
// market impact. The returned scale remains 1,000,000 for wire compatibility.
func CalculatePriceImpactBpsWithFee(
	reserveIn, reserveOut, tokenIn, tokenOut *big.Int,
	feeBps uint64,
) uint64 {
	if reserveIn.Sign() <= 0 || tokenIn.Sign() <= 0 || reserveOut.Sign() <= 0 ||
		tokenOut.Sign() < 0 || feeBps > 1_000_000 {
		return 0
	}
	feeMultiplier := new(big.Int).Sub(bpsBasis, new(big.Int).SetUint64(feeBps))
	idealNumerator := new(big.Int).Mul(tokenIn, feeMultiplier)
	idealNumerator.Mul(idealNumerator, reserveOut)
	if idealNumerator.Sign() <= 0 {
		return 0
	}
	idealDenominator := new(big.Int).Mul(reserveIn, bpsBasis)
	actualNumerator := new(big.Int).Mul(tokenOut, idealDenominator)
	diff := new(big.Int).Sub(idealNumerator, actualNumerator)
	if diff.Sign() <= 0 {
		return 0
	}
	impact := new(big.Int).Mul(diff, bpsBasis)
	impact.Div(impact, idealNumerator)

	return impact.Uint64()
}
