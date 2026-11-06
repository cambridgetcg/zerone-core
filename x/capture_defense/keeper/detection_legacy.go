package keeper

import (
	"math"

	"github.com/zerone-chain/zerone/x/capture_defense/types"
)

// timingCorrelationLegacy preserves the predecessor calculation, including its
// overflow and floating-point rounding, until IntegerVarianceEnabled is set.
// Do not simplify or repair this path: pre-activation replay needs the original
// operations. The body is copied from 2e37c4c (detection.go blob 3085b4f8).
func timingCorrelationLegacy(history []*types.VerificationHistoryEntry) uint64 {
	if len(history) == 0 {
		return 0
	}

	var correlated uint64
	var total uint64

	for _, entry := range history {
		if len(entry.SubmitBlocks) < 2 {
			continue
		}
		total++

		// Compute standard deviation
		var sum uint64
		for _, b := range entry.SubmitBlocks {
			sum += b
		}
		mean := float64(sum) / float64(len(entry.SubmitBlocks))

		var varianceSum float64
		for _, b := range entry.SubmitBlocks {
			diff := float64(b) - mean
			varianceSum += diff * diff
		}
		stdDev := math.Sqrt(varianceSum / float64(len(entry.SubmitBlocks)))

		if stdDev < 2.0 {
			correlated++
		}
	}

	if total == 0 {
		return 0
	}
	return correlated * types.BPSScale / total
}
