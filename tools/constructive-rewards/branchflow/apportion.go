package branchflow

import (
	"fmt"
	"math/big"
	"sort"
)

type weightedKey struct {
	key       string
	weight    *big.Int
	isCommons bool
}

type apportionedValue struct {
	key    string
	amount *big.Int
}

type remainderRank struct {
	index     int
	remainder *big.Int
	commons   bool
	key       string
}

// apportion applies Hamilton's largest-remainder method using exact integers.
// Weights must sum exactly to denominator. Exact ties prefer commons, then the
// canonical key, so caller iteration order never affects a unit of output.
func apportion(total, denominator *big.Int, weights []weightedKey) ([]apportionedValue, error) {
	if total == nil || total.Sign() < 0 {
		return nil, inputError(CodeInvalidAmount, "apportion.total", "must be non-negative")
	}
	if denominator == nil || denominator.Sign() <= 0 {
		return nil, inputError(CodeInvalidAmount, "apportion.denominator", "must be positive")
	}
	if len(weights) == 0 {
		return nil, inputError(CodeInvalidInput, "apportion.weights", "must be non-empty")
	}

	sortedWeights := make([]weightedKey, len(weights))
	seen := make(map[string]struct{}, len(weights))
	sum := new(big.Int)
	for i, item := range weights {
		if item.key == "" {
			return nil, inputError(CodeDuplicateID, fmt.Sprintf("apportion.weights[%d].key", i), "must be non-empty")
		}
		if _, exists := seen[item.key]; exists {
			return nil, inputError(CodeDuplicateID, fmt.Sprintf("apportion.weights[%d].key", i), "duplicate key")
		}
		seen[item.key] = struct{}{}
		if item.weight == nil || item.weight.Sign() < 0 {
			return nil, inputError(CodeInvalidAmount, fmt.Sprintf("apportion.weights[%d].weight", i), "must be non-negative")
		}
		sortedWeights[i] = weightedKey{
			key: item.key, weight: new(big.Int).Set(item.weight), isCommons: item.isCommons,
		}
		sum.Add(sum, item.weight)
	}
	if sum.Cmp(denominator) != 0 {
		return nil, invariantError(fmt.Sprintf("apportion weights sum %s, denominator %s", sum, denominator))
	}
	sort.Slice(sortedWeights, func(i, j int) bool { return sortedWeights[i].key < sortedWeights[j].key })

	result := make([]apportionedValue, len(sortedWeights))
	ranks := make([]remainderRank, len(sortedWeights))
	allocated := new(big.Int)
	for i, item := range sortedWeights {
		numerator := new(big.Int).Mul(total, item.weight)
		quotient, remainder := new(big.Int), new(big.Int)
		quotient.QuoRem(numerator, denominator, remainder)
		result[i] = apportionedValue{key: item.key, amount: quotient}
		ranks[i] = remainderRank{
			index: i, remainder: remainder, commons: item.isCommons, key: item.key,
		}
		allocated.Add(allocated, quotient)
	}

	leftover := new(big.Int).Sub(new(big.Int).Set(total), allocated)
	if !leftover.IsInt64() || leftover.Int64() < 0 || leftover.Int64() >= int64(len(weights)+1) {
		return nil, invariantError(fmt.Sprintf("invalid Hamilton remainder %s for %d weights", leftover, len(weights)))
	}
	sort.Slice(ranks, func(i, j int) bool {
		if cmp := ranks[i].remainder.Cmp(ranks[j].remainder); cmp != 0 {
			return cmp > 0
		}
		if ranks[i].commons != ranks[j].commons {
			return ranks[i].commons
		}
		return ranks[i].key < ranks[j].key
	})
	for i := int64(0); i < leftover.Int64(); i++ {
		result[ranks[i].index].amount.Add(result[ranks[i].index].amount, big.NewInt(1))
	}

	check := new(big.Int)
	for _, item := range result {
		check.Add(check, item.amount)
	}
	if check.Cmp(total) != 0 {
		return nil, invariantError(fmt.Sprintf("Hamilton output %s, total %s", check, total))
	}
	return result, nil
}

func findApportioned(items []apportionedValue, key string) *big.Int {
	for _, item := range items {
		if item.key == key {
			return new(big.Int).Set(item.amount)
		}
	}
	return new(big.Int)
}

func indexApportioned(items []apportionedValue) map[string]*big.Int {
	indexed := make(map[string]*big.Int, len(items))
	for _, item := range items {
		indexed[item.key] = new(big.Int).Set(item.amount)
	}
	return indexed
}

func ppmInt(value uint64) *big.Int {
	return new(big.Int).SetUint64(value)
}
