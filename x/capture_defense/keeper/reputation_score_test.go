package keeper

import (
	"testing"

	"github.com/zerone-chain/zerone/x/capture_defense/types"
)

func TestNextReputationScore(t *testing.T) {
	for _, tc := range []struct {
		score    uint64
		approved bool
		want     uint64
	}{
		{0, true, 10000}, {0, false, 0},
		{999, false, 0}, {1000, false, 0}, {1001, false, 1},
		{500000, true, 505000}, {500000, false, 490000},
		{999000, true, 1000000}, {1000000, true, 1000000}, {1000000, false, 980000},
	} {
		if got := nextReputationScore(tc.score, tc.approved); got != tc.want {
			t.Errorf("score=%d approved=%t: got %d, want %d", tc.score, tc.approved, got, tc.want)
		}
	}
}

// The prior implementation copied this arithmetic into all three reputation
// layers. Freeze it here to check extraction, not to introduce a new policy.
func legacyReputationScore(score uint64, approved bool) uint64 {
	if approved {
		delta := (types.BPSScale - score) / 100
		if delta < 1000 {
			delta = 1000
		}
		score += delta
		if score > types.BPSScale {
			score = types.BPSScale
		}
	} else {
		delta := score / 50
		if delta < 1000 {
			delta = 1000
		}
		if score > delta {
			score -= delta
		} else {
			score = 0
		}
	}
	return score
}

func TestNextReputationScorePreservesLegacyArithmetic(t *testing.T) {
	check := func(score uint64) {
		for _, approved := range []bool{false, true} {
			if got, want := nextReputationScore(score, approved), legacyReputationScore(score, approved); got != want {
				t.Fatalf("score=%d approved=%t: got %d, legacy %d", score, approved, got, want)
			}
		}
	}
	for score := uint64(0); score <= types.BPSScale; score++ {
		check(score)
	}
	// Even out-of-range retained values must not acquire new overflow semantics
	// merely because duplicated arithmetic was extracted.
	for _, score := range []uint64{types.BPSScale + 1, 2 * types.BPSScale, 1 << 63, ^uint64(0) - 1000, ^uint64(0)} {
		check(score)
	}
}
