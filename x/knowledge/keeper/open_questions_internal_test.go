package keeper

import (
	"math"
	"testing"
)

func TestBoundedOpenQuestionsLimit_ClampsBeforeAllocation(t *testing.T) {
	if got := boundedOpenQuestionsLimit(math.MaxUint32); got != OpenQuestionsScanCap {
		t.Fatalf("huge requested limit was not clamped: got %d, want %d", got, OpenQuestionsScanCap)
	}
}
