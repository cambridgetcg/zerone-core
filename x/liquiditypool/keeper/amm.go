package keeper

import "math/big"

// bpsBasis is the basis points scale (1,000,000 = 100%).
var bpsBasis = big.NewInt(1_000_000)

// CalculateSwapOutput computes the output amount for a constant-product swap.
// Formula: dy = y * dx / (x + dx)  (after fee deduction from dx)
//
// Fee is deducted from the input before the swap:
//
//	effectiveIn = tokenIn * (1_000_000 - feeBps) / 1_000_000
//	tokenOut = reserveOut * effectiveIn / (reserveIn + effectiveIn)
func CalculateSwapOutput(reserveIn, reserveOut, tokenIn *big.Int, feeBps uint64) (tokenOut, feeAmount *big.Int) {
	if reserveIn.Sign() <= 0 || reserveOut.Sign() <= 0 || tokenIn.Sign() <= 0 {
		return new(big.Int), new(big.Int)
	}

	// Fee deduction
	feeAmount = new(big.Int).Mul(tokenIn, big.NewInt(int64(feeBps)))
	feeAmount.Div(feeAmount, bpsBasis)

	effectiveIn := new(big.Int).Sub(tokenIn, feeAmount)

	// dy = y * dx / (x + dx)
	numerator := new(big.Int).Mul(reserveOut, effectiveIn)
	denominator := new(big.Int).Add(reserveIn, effectiveIn)

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

// CalculatePriceImpactBps computes the price impact in basis points (1M scale).
// priceImpact = 1 - (tokenOut * reserveIn) / (tokenIn * reserveOut)
func CalculatePriceImpactBps(reserveIn, reserveOut, tokenIn, tokenOut *big.Int) uint64 {
	if tokenIn.Sign() <= 0 || reserveOut.Sign() <= 0 {
		return 0
	}
	// ideal = tokenIn * reserveOut / reserveIn
	ideal := new(big.Int).Mul(tokenIn, reserveOut)
	ideal.Div(ideal, reserveIn)

	if ideal.Sign() <= 0 {
		return 0
	}

	// impact = (ideal - tokenOut) * 1M / ideal
	diff := new(big.Int).Sub(ideal, tokenOut)
	if diff.Sign() <= 0 {
		return 0
	}
	impact := new(big.Int).Mul(diff, bpsBasis)
	impact.Div(impact, ideal)

	return impact.Uint64()
}
