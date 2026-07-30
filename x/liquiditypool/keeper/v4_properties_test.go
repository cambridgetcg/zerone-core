package keeper_test

import (
	"math/big"
	"testing"

	"github.com/zerone-chain/zerone/x/liquiditypool/keeper"
)

const v4FeeScale uint64 = 1_000_000

func v4ExpectedScaledSwapOutput(reserveIn, reserveOut, amountIn *big.Int, feePPM uint64) *big.Int {
	if reserveIn.Sign() <= 0 || reserveOut.Sign() <= 0 || amountIn.Sign() <= 0 || feePPM >= v4FeeScale {
		return new(big.Int)
	}

	scale := new(big.Int).SetUint64(v4FeeScale)
	multiplier := new(big.Int).SetUint64(v4FeeScale - feePPM)
	weightedIn := new(big.Int).Mul(amountIn, multiplier)
	numerator := new(big.Int).Mul(reserveOut, weightedIn)
	denominator := new(big.Int).Add(
		new(big.Int).Mul(reserveIn, scale),
		weightedIn,
	)
	return numerator.Quo(numerator, denominator)
}

func TestV4ScaledSwapMathPreservesFractionalFeeEffect(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		reserveIn  string
		reserveOut string
		amountIn   string
		feePPM     uint64
		wantOut    string
	}{
		{
			name:       "one indivisible counter unit still pays fee through output",
			reserveIn:  "1",
			reserveOut: "10000000000",
			amountIn:   "1",
			feePPM:     3_000,
			wantOut:    "4992488733",
		},
		{
			name:       "input below old rounded fee threshold",
			reserveIn:  "1000000000",
			reserveOut: "1000000000000",
			amountIn:   "333",
			feePPM:     3_000,
			wantOut:    "332000",
		},
		{
			name:       "zero fee remains exact constant product",
			reserveIn:  "1000000",
			reserveOut: "2000000",
			amountIn:   "10000",
			feePPM:     0,
			wantOut:    "19801",
		},
		{
			name:       "maximum approved pool fee",
			reserveIn:  "1000000000",
			reserveOut: "5000000000",
			amountIn:   "25000000",
			feePPM:     100_000,
			wantOut:    "110024449",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			reserveIn, ok := new(big.Int).SetString(tt.reserveIn, 10)
			if !ok {
				t.Fatalf("bad test reserve_in %q", tt.reserveIn)
			}
			reserveOut, ok := new(big.Int).SetString(tt.reserveOut, 10)
			if !ok {
				t.Fatalf("bad test reserve_out %q", tt.reserveOut)
			}
			amountIn, ok := new(big.Int).SetString(tt.amountIn, 10)
			if !ok {
				t.Fatalf("bad test amount_in %q", tt.amountIn)
			}

			got, fee := keeper.CalculateSwapOutput(reserveIn, reserveOut, amountIn, tt.feePPM)
			if got.String() != tt.wantOut {
				t.Fatalf("scaled swap output = %s, want %s", got, tt.wantOut)
			}

			formula := v4ExpectedScaledSwapOutput(reserveIn, reserveOut, amountIn, tt.feePPM)
			if got.Cmp(formula) != 0 {
				t.Fatalf("implementation/formula mismatch: got %s, formula %s", got, formula)
			}
			wantFee := new(big.Int).Mul(amountIn, new(big.Int).SetUint64(tt.feePPM))
			wantFee.Quo(wantFee, new(big.Int).SetUint64(v4FeeScale))
			if fee.Cmp(wantFee) != 0 {
				t.Fatalf("fee amount = %s, want %s", fee, wantFee)
			}
		})
	}
}

func TestV4SplittingSwapNeverImprovesOutput(t *testing.T) {
	t.Parallel()

	fees := []uint64{0, 1, 3_000, 100_000}
	for reserveIn := uint64(1); reserveIn <= 24; reserveIn++ {
		for reserveOut := uint64(1); reserveOut <= 24; reserveOut++ {
			for totalIn := uint64(2); totalIn <= 24; totalIn++ {
				for firstIn := uint64(1); firstIn < totalIn; firstIn++ {
					for _, feePPM := range fees {
						x := new(big.Int).SetUint64(reserveIn)
						y := new(big.Int).SetUint64(reserveOut)
						total := new(big.Int).SetUint64(totalIn)

						oneShot, _ := keeper.CalculateSwapOutput(x, y, total, feePPM)

						first := new(big.Int).SetUint64(firstIn)
						firstOut, _ := keeper.CalculateSwapOutput(x, y, first, feePPM)
						xAfterFirst := new(big.Int).Add(x, first)
						yAfterFirst := new(big.Int).Sub(y, firstOut)

						second := new(big.Int).SetUint64(totalIn - firstIn)
						secondOut, _ := keeper.CalculateSwapOutput(xAfterFirst, yAfterFirst, second, feePPM)
						splitOut := new(big.Int).Add(firstOut, secondOut)

						if splitOut.Cmp(oneShot) > 0 {
							t.Fatalf(
								"fragmentation advantage: reserves=%d/%d input=%d split=%d+%d fee=%d one=%s split=%s",
								reserveIn,
								reserveOut,
								totalIn,
								firstIn,
								totalIn-firstIn,
								feePPM,
								oneShot,
								splitOut,
							)
						}
					}
				}
			}
		}
	}
}

func FuzzV4SwapOutputAndConstantProduct(f *testing.F) {
	seeds := [][4]uint64{
		{1, 10_000_000_000, 1, 3_000},
		{1_000_000, 2_000_000, 10_000, 0},
		{1_000_000_000, 5_000_000_000, 25_000_000, 100_000},
		{10_000_000_000, 1, 1_000_000_000, 3_000},
		{1<<63 - 1, 1<<63 - 2, 1<<32 + 1, 99_999},
	}
	for _, seed := range seeds {
		f.Add(seed[0], seed[1], seed[2], seed[3])
	}

	f.Fuzz(func(t *testing.T, reserveIn, reserveOut, amountIn, feeSeed uint64) {
		if reserveIn == 0 || reserveOut == 0 || amountIn == 0 {
			t.Skip()
		}
		feePPM := feeSeed % 100_001

		x := new(big.Int).SetUint64(reserveIn)
		y := new(big.Int).SetUint64(reserveOut)
		dx := new(big.Int).SetUint64(amountIn)

		got, fee := keeper.CalculateSwapOutput(x, y, dx, feePPM)
		want := v4ExpectedScaledSwapOutput(x, y, dx, feePPM)
		if got.Cmp(want) != 0 {
			t.Fatalf(
				"scaled formula mismatch: x=%d y=%d dx=%d fee=%d got=%s want=%s",
				reserveIn,
				reserveOut,
				amountIn,
				feePPM,
				got,
				want,
			)
		}
		if got.Sign() < 0 || got.Cmp(y) >= 0 {
			t.Fatalf("output outside [0,reserveOut): out=%s reserve_out=%s", got, y)
		}
		wantFee := new(big.Int).Mul(dx, new(big.Int).SetUint64(feePPM))
		wantFee.Quo(wantFee, new(big.Int).SetUint64(v4FeeScale))
		if fee.Cmp(wantFee) != 0 {
			t.Fatalf("fee mismatch: dx=%s fee_ppm=%d got=%s want=%s", dx, feePPM, fee, wantFee)
		}

		kBefore := new(big.Int).Mul(x, y)
		xAfter := new(big.Int).Add(x, dx)
		yAfter := new(big.Int).Sub(y, got)
		kAfter := new(big.Int).Mul(xAfter, yAfter)
		if kAfter.Cmp(kBefore) < 0 {
			t.Fatalf(
				"constant product decreased: x=%s y=%s dx=%s out=%s before=%s after=%s",
				x,
				y,
				dx,
				got,
				kBefore,
				kAfter,
			)
		}
	})
}

func FuzzV4DepositAndImmediateWithdrawalCannotExtractValue(f *testing.F) {
	seeds := [][5]uint64{
		{10_000_000_000, 1, 100_000, 1, 1},
		{10_000_000_000, 5_000_000_000, 7_071_067_811, 1_000_000, 500_000},
		{1, 10_000_000_000, 100_000, 1, 1_000_000},
		{1<<63 - 1, 1<<62 + 1, 1<<32 + 1, 1<<31 + 1, 1<<30 + 1},
	}
	for _, seed := range seeds {
		f.Add(seed[0], seed[1], seed[2], seed[3], seed[4])
	}

	f.Fuzz(func(t *testing.T, reserveA, reserveB, supply, desiredA, desiredB uint64) {
		if reserveA == 0 || reserveB == 0 || supply == 0 || desiredA == 0 || desiredB == 0 {
			t.Skip()
		}

		rA := new(big.Int).SetUint64(reserveA)
		rB := new(big.Int).SetUint64(reserveB)
		total := new(big.Int).SetUint64(supply)
		dA := new(big.Int).SetUint64(desiredA)
		dB := new(big.Int).SetUint64(desiredB)

		actualA, actualB, minted := keeper.CalculateDepositForShares(rA, rB, dA, dB, total)
		if minted.Sign() == 0 {
			if actualA.Sign() != 0 || actualB.Sign() != 0 {
				t.Fatalf("zero-share deposit returned non-zero assets %s/%s", actualA, actualB)
			}
			return
		}
		if actualA.Sign() <= 0 || actualB.Sign() <= 0 {
			t.Fatalf("positive shares %s are not backed by two positive assets: %s/%s", minted, actualA, actualB)
		}
		if actualA.Cmp(dA) > 0 || actualB.Cmp(dB) > 0 {
			t.Fatalf("actual deposit %s/%s exceeds desired %s/%s", actualA, actualB, dA, dB)
		}

		// Each side independently backs the minted ownership fraction.
		if new(big.Int).Mul(actualA, total).Cmp(new(big.Int).Mul(minted, rA)) < 0 {
			t.Fatalf("asset A under-backs minted shares: reserve=%s supply=%s actual=%s minted=%s", rA, total, actualA, minted)
		}
		if new(big.Int).Mul(actualB, total).Cmp(new(big.Int).Mul(minted, rB)) < 0 {
			t.Fatalf("asset B under-backs minted shares: reserve=%s supply=%s actual=%s minted=%s", rB, total, actualB, minted)
		}

		newReserveA := new(big.Int).Add(rA, actualA)
		newReserveB := new(big.Int).Add(rB, actualB)
		newSupply := new(big.Int).Add(total, minted)
		withdrawnA, withdrawnB := keeper.CalculateWithdrawalAmounts(
			newReserveA,
			newReserveB,
			minted,
			newSupply,
		)
		if withdrawnA.Cmp(actualA) > 0 || withdrawnB.Cmp(actualB) > 0 {
			t.Fatalf(
				"deposit/withdraw cycle extracted value: deposit=%s/%s withdrawal=%s/%s minted=%s",
				actualA,
				actualB,
				withdrawnA,
				withdrawnB,
				minted,
			)
		}
	})
}

func FuzzV4WithdrawalConservesPoolAssets(f *testing.F) {
	f.Add(uint64(10_000_000_000), uint64(5_000_000_000), uint64(7_071_067_811), uint64(1))
	f.Add(uint64(1), uint64(1), uint64(1), uint64(0))
	f.Add(uint64(1<<63-1), uint64(1<<62+1), uint64(1<<32+1), uint64(1<<31))

	f.Fuzz(func(t *testing.T, reserveA, reserveB, supply, shareSeed uint64) {
		if reserveA == 0 || reserveB == 0 || supply == 0 {
			t.Skip()
		}
		shares := shareSeed%supply + 1
		rA := new(big.Int).SetUint64(reserveA)
		rB := new(big.Int).SetUint64(reserveB)
		lp := new(big.Int).SetUint64(shares)
		total := new(big.Int).SetUint64(supply)

		amountA, amountB := keeper.CalculateWithdrawalAmounts(rA, rB, lp, total)
		if amountA.Sign() < 0 || amountA.Cmp(rA) > 0 || amountB.Sign() < 0 || amountB.Cmp(rB) > 0 {
			t.Fatalf("withdrawal exceeds reserves: reserves=%s/%s shares=%s/%s output=%s/%s", rA, rB, lp, total, amountA, amountB)
		}
		if shares == supply && (amountA.Cmp(rA) != 0 || amountB.Cmp(rB) != 0) {
			t.Fatalf("full withdrawal did not return all reserves: reserves=%s/%s output=%s/%s", rA, rB, amountA, amountB)
		}
	})
}

func FuzzV4SwapFragmentationCannotIncreaseOutput(f *testing.F) {
	f.Add(uint64(1), uint64(10_000_000_000), uint64(2), uint64(1), uint64(3_000))
	f.Add(uint64(1_000_000), uint64(2_000_000), uint64(10_000), uint64(3_333), uint64(0))
	f.Add(uint64(5_000_000_000), uint64(9_000_000_000), uint64(25_000_000), uint64(1), uint64(100_000))

	f.Fuzz(func(t *testing.T, reserveIn, reserveOut, totalIn, splitSeed, feeSeed uint64) {
		if reserveIn == 0 || reserveOut == 0 || totalIn < 2 {
			t.Skip()
		}

		firstIn := splitSeed%(totalIn-1) + 1
		secondIn := totalIn - firstIn
		feePPM := feeSeed % 100_001

		x := new(big.Int).SetUint64(reserveIn)
		y := new(big.Int).SetUint64(reserveOut)
		total := new(big.Int).SetUint64(totalIn)
		oneShot, _ := keeper.CalculateSwapOutput(x, y, total, feePPM)

		first := new(big.Int).SetUint64(firstIn)
		firstOut, _ := keeper.CalculateSwapOutput(x, y, first, feePPM)
		xAfterFirst := new(big.Int).Add(x, first)
		yAfterFirst := new(big.Int).Sub(y, firstOut)

		second := new(big.Int).SetUint64(secondIn)
		secondOut, _ := keeper.CalculateSwapOutput(xAfterFirst, yAfterFirst, second, feePPM)
		splitOut := new(big.Int).Add(firstOut, secondOut)
		if splitOut.Cmp(oneShot) > 0 {
			t.Fatalf(
				"fragmentation advantage: x=%d y=%d total=%d split=%d+%d fee=%d one=%s split_out=%s",
				reserveIn,
				reserveOut,
				totalIn,
				firstIn,
				secondIn,
				feePPM,
				oneShot,
				splitOut,
			)
		}
	})
}

func FuzzV4ProtocolFeeSkimPreservesKAndNoFragmentationAdvantage(f *testing.F) {
	f.Add(
		uint64(10_000_000_000),
		uint64(5_000_000_000),
		uint64(1_000_000_000),
		uint64(400_000_000),
		uint64(3_000),
		uint64(450_000),
	)
	f.Add(
		uint64(1_000_000),
		uint64(2_000_000),
		uint64(10_000),
		uint64(1),
		uint64(100_000),
		uint64(1_000_000),
	)

	f.Fuzz(func(
		t *testing.T,
		reserveIn,
		reserveOut,
		totalIn,
		splitSeed,
		feeSeed,
		protocolFeeSeed uint64,
	) {
		if reserveIn == 0 || reserveOut == 0 || totalIn < 2 {
			t.Skip()
		}
		feePPM := feeSeed % 100_001
		protocolFeePPM := protocolFeeSeed % 1_000_001
		firstIn := splitSeed%(totalIn-1) + 1
		secondIn := totalIn - firstIn

		x := new(big.Int).SetUint64(reserveIn)
		y := new(big.Int).SetUint64(reserveOut)
		total := new(big.Int).SetUint64(totalIn)
		oneShot, oneFee := keeper.CalculateSwapOutput(x, y, total, feePPM)
		oneProtocolFee := new(big.Int).Mul(oneFee, new(big.Int).SetUint64(protocolFeePPM))
		oneProtocolFee.Quo(oneProtocolFee, new(big.Int).SetUint64(v4FeeScale))

		kBefore := new(big.Int).Mul(x, y)
		xAfterOne := new(big.Int).Add(x, total)
		xAfterOne.Sub(xAfterOne, oneProtocolFee)
		yAfterOne := new(big.Int).Sub(y, oneShot)
		if new(big.Int).Mul(xAfterOne, yAfterOne).Cmp(kBefore) < 0 {
			t.Fatalf(
				"protocol skim decreased k: x=%s y=%s input=%s fee=%d protocol=%d out=%s skim=%s",
				x,
				y,
				total,
				feePPM,
				protocolFeePPM,
				oneShot,
				oneProtocolFee,
			)
		}

		first := new(big.Int).SetUint64(firstIn)
		firstOut, firstFee := keeper.CalculateSwapOutput(x, y, first, feePPM)
		firstProtocolFee := new(big.Int).Mul(firstFee, new(big.Int).SetUint64(protocolFeePPM))
		firstProtocolFee.Quo(firstProtocolFee, new(big.Int).SetUint64(v4FeeScale))
		xAfterFirst := new(big.Int).Add(x, first)
		xAfterFirst.Sub(xAfterFirst, firstProtocolFee)
		yAfterFirst := new(big.Int).Sub(y, firstOut)

		second := new(big.Int).SetUint64(secondIn)
		secondOut, _ := keeper.CalculateSwapOutput(xAfterFirst, yAfterFirst, second, feePPM)
		splitOut := new(big.Int).Add(firstOut, secondOut)
		if splitOut.Cmp(oneShot) > 0 {
			t.Fatalf(
				"fragmentation advantage with protocol skim: x=%d y=%d total=%d split=%d+%d fee=%d protocol=%d one=%s split=%s",
				reserveIn,
				reserveOut,
				totalIn,
				firstIn,
				secondIn,
				feePPM,
				protocolFeePPM,
				oneShot,
				splitOut,
			)
		}
	})
}
