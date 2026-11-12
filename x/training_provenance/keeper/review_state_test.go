package keeper

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// The review state and its sentence are fixed by predicate v2. The SDK
// recomputes both from the counts, so any change here is a new predicate.
func TestComputeReviewState(t *testing.T) {
	for _, tc := range []struct {
		name        string
		coverage    reviewCoverage
		state       string
		explanation string
	}{
		{
			name:        "no facts",
			coverage:    reviewCoverage{},
			state:       ReviewStateEmpty,
			explanation: "fact_count=0: no fact IDs are included directly in this manifest, so there is no review to count",
		},
		{
			name:        "every fact reviewed",
			coverage:    reviewCoverage{reviewed: 3},
			state:       ReviewStateReviewed,
			explanation: "reviewed_fact_count=3 of fact_count=3 have a retained verification round with at least one revealed vote; unreviewed_fact_count=0, including doctrine_fact_count=0 and missing_fact_count=0",
		},
		{
			name:        "doctrine only",
			coverage:    reviewCoverage{unreviewed: 47, doctrine: 47},
			state:       ReviewStateUnreviewed,
			explanation: "reviewed_fact_count=0 of fact_count=47 have a retained verification round with at least one revealed vote; unreviewed_fact_count=47, including doctrine_fact_count=47 and missing_fact_count=0",
		},
		{
			name:        "missing only",
			coverage:    reviewCoverage{unreviewed: 1, missing: 1},
			state:       ReviewStateUnreviewed,
			explanation: "reviewed_fact_count=0 of fact_count=1 have a retained verification round with at least one revealed vote; unreviewed_fact_count=1, including doctrine_fact_count=0 and missing_fact_count=1",
		},
		{
			name:        "mixed",
			coverage:    reviewCoverage{reviewed: 1, unreviewed: 5, doctrine: 1, missing: 1},
			state:       ReviewStatePartlyReviewed,
			explanation: "reviewed_fact_count=1 of fact_count=6 have a retained verification round with at least one revealed vote; unreviewed_fact_count=5, including doctrine_fact_count=1 and missing_fact_count=1",
		},
		{
			name:        "one unreviewed among many reviewed",
			coverage:    reviewCoverage{reviewed: 999, unreviewed: 1},
			state:       ReviewStatePartlyReviewed,
			explanation: "reviewed_fact_count=999 of fact_count=1000 have a retained verification round with at least one revealed vote; unreviewed_fact_count=1, including doctrine_fact_count=0 and missing_fact_count=0",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			state, explanation := computeReviewState(tc.coverage)
			require.Equal(t, tc.state, state)
			require.Equal(t, tc.explanation, explanation)
		})
	}
}
