package keeper

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// Exhaust every candidate traversal, not random map samples. In particular,
// [1,1,3] must clear an earlier tie in both lower-tie-first permutations.
func TestReformulationSelectionEveryPermutation(t *testing.T) {
	for _, tc := range []struct {
		weights   []uint64
		threshold uint64
		want      types.AugmentationVerdict
		tied      bool
	}{
		{[]uint64{1, 1, 3}, 200_000, 3, false},
		{[]uint64{3, 3, 1}, 200_000, 0, true},
		{[]uint64{3, 3, 1, 4}, 200_000, 4, false},
		{[]uint64{1, 1, 1, 1}, 250_000, 0, true},
		{[]uint64{1, 1}, 666_000, 0, true},
		{[]uint64{0, 0}, 1, 0, false},
		{[]uint64{200_000_000_000_000, 100_000_000_000_000}, 666_000, 1, false},
		{[]uint64{2, 1}, 1_000_001, 0, false},
	} {
		t.Run(fmt.Sprint(tc.weights, tc.threshold), func(t *testing.T) {
			var total uint64
			candidates := make([]reformulationCandidate, len(tc.weights))
			for i, weight := range tc.weights {
				total += weight
				candidates[i] = reformulationCandidate{types.AugmentationVerdict(i + 1), weight}
			}
			count := 0
			var permute func(int)
			permute = func(i int) {
				if i == len(candidates) {
					winner, tied := selectReformulationWinner(candidates, total, tc.threshold)
					require.Equal(t, tc.want, winner, "order=%v", candidates)
					require.Equal(t, tc.tied, tied, "order=%v", candidates)
					count++
					return
				}
				for j := i; j < len(candidates); j++ {
					candidates[i], candidates[j] = candidates[j], candidates[i]
					permute(i + 1)
					candidates[i], candidates[j] = candidates[j], candidates[i]
				}
			}
			permute(0)
			wantCount := 1
			for n := 2; n <= len(candidates); n++ {
				wantCount *= n
			}
			require.Equal(t, wantCount, count)
		})
	}
}
