package keeper

import (
	"context"
	"math"
	"math/big"

	"github.com/zerone-chain/zerone/x/billing/types"
)

// bpsBasis is 1,000,000 (basis points denominator, 1M = 100%).
var bpsBasis = big.NewInt(1000000)

// CalculateQueryPrice computes the price for querying a set of facts.
//
// Per-fact formula (Marginal Information Value pricing):
//   - Confidence bell curve: medium-high = peak value, low = surcharge, high = discount
//   - Novelty (inverted citations): few = premium, many = discount
//   - Freshness: recent facts = premium (not in LLM training data)
func (k Keeper) CalculateQueryPrice(ctx context.Context, factIds []string, currentBlock uint64) (*big.Int, []*types.FactPriceBreakdown) {
	params := k.GetParams(ctx)

	baseCost := new(big.Int)

	// Dynamic pricing: use oracle-derived base cost when available
	dynamicBase := k.calculateDynamicBaseCost(ctx)
	if dynamicBase.Sign() > 0 {
		baseCost.Set(dynamicBase)
	} else {
		baseCost.SetString(params.BaseQueryPrice, 10)
	}

	dpCfg := params.DynamicPricing
	if dpCfg == nil {
		dpCfg = types.DefaultDynamicPricingConfig()
	}

	totalPrice := new(big.Int)
	var breakdown []*types.FactPriceBreakdown

	for _, factId := range factIds {
		factPrice := new(big.Int).Set(baseCost)
		confidencePremium := new(big.Int)
		noveltyBonus := new(big.Int)
		freshnessDiscount := new(big.Int)

		// 1. Confidence bell curve
		confidence, found := k.knowledgeKeeper.GetFactConfidence(ctx, factId)
		if found {
			adj := new(big.Int).Mul(baseCost, big.NewInt(int64(params.ConfidenceWeightBps)))
			adj.Div(adj, bpsBasis)
			if confidence < params.ConfidenceThreshold {
				// Low confidence: surcharge
				confidencePremium.Set(adj)
				factPrice.Add(factPrice, confidencePremium)
			} else if confidence > 850000 {
				// High confidence: discount (already in LLM training data)
				confidencePremium.Neg(adj)
				factPrice.Sub(factPrice, adj)
				if factPrice.Sign() < 0 {
					factPrice.SetInt64(0)
				}
			}
			// Threshold to 850k: peak value zone, no adjustment
		}

		// 2. Novelty (inverted citations): highly cited = well-known = discount
		citations, citFound := k.knowledgeKeeper.GetFactCitationCount(ctx, factId)
		if citFound && citations > 0 {
			log2Val := uint64(math.Log2(float64(citations + 1)))
			if log2Val > 0 {
				noveltyBonus.Mul(baseCost, big.NewInt(int64(log2Val)))
				noveltyBonus.Div(noveltyBonus, big.NewInt(10))
				// Cap at 50% of base
				halfBase := new(big.Int).Div(baseCost, big.NewInt(2))
				if noveltyBonus.Cmp(halfBase) > 0 {
					noveltyBonus.Set(halfBase)
				}
				noveltyBonus.Neg(noveltyBonus) // negative = discount
				factPrice.Sub(factPrice, new(big.Int).Neg(noveltyBonus))
				if factPrice.Sign() < 0 {
					factPrice.SetInt64(0)
				}
			}
		}

		// 3. Freshness premium: recent facts = NOT in training data = premium
		createdBlock, blockFound := k.knowledgeKeeper.GetFactCreatedBlock(ctx, factId)
		if blockFound && currentBlock > createdBlock {
			age := currentBlock - createdBlock
			if age < params.FreshnessWindowBlocks {
				freshnessDiscount.Mul(baseCost, big.NewInt(int64(params.FreshnessWeightBps)))
				freshnessDiscount.Div(freshnessDiscount, bpsBasis)
				freshnessDiscount.Neg(freshnessDiscount) // negative = premium
				factPrice.Add(factPrice, new(big.Int).Neg(freshnessDiscount))
			}
		}

		// Enforce per-fact floor/ceiling when dynamic pricing is active
		if dpCfg.Enabled && dynamicBase.Sign() > 0 {
			minFact := new(big.Int)
			minFact.SetString(dpCfg.MinCostPerFact, 10)
			maxFact := new(big.Int)
			maxFact.SetString(dpCfg.MaxCostPerFact, 10)
			if factPrice.Cmp(minFact) < 0 {
				factPrice.Set(minFact)
			}
			if factPrice.Cmp(maxFact) > 0 {
				factPrice.Set(maxFact)
			}
		}

		totalPrice.Add(totalPrice, factPrice)

		breakdown = append(breakdown, &types.FactPriceBreakdown{
			FactId:            factId,
			BaseCost:          baseCost.String(),
			ConfidencePremium: confidencePremium.String(),
			NoveltyBonus:      noveltyBonus.String(),
			FreshnessDiscount: freshnessDiscount.String(),
			TotalPrice:        factPrice.String(),
		})
	}

	return totalPrice, breakdown
}
