package keeper

import (
	"context"
	"testing"

	"github.com/zerone-chain/zerone/internal/activation"
	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// SetContestScanMaxRecordsForTest lowers the record ceiling of one contest
// check for the rest of t, so a test reaches the bound with a few records.
// Keeper tests do not run in parallel.
func SetContestScanMaxRecordsForTest(t testing.TB, n int) {
	t.Helper()
	previous := contestScanMaxRecords
	contestScanMaxRecords = n
	t.Cleanup(func() { contestScanMaxRecords = previous })
}

// SetContestScanMaxBytesForTest lowers the byte ceiling of one contest check
// for the rest of t, so a test reaches the bound with a few long records.
// Keeper tests do not run in parallel.
func SetContestScanMaxBytesForTest(t testing.TB, n int) {
	t.Helper()
	previous := contestScanMaxBytes
	contestScanMaxBytes = n
	t.Cleanup(func() { contestScanMaxBytes = previous })
}

// SetAugmentationRestoreMaxRecordsForTest lowers the record ceiling of the
// augmentation inventory RestoreAugmentationReplacedEdges reads, for the rest
// of t. Keeper tests do not run in parallel.
func SetAugmentationRestoreMaxRecordsForTest(t testing.TB, n int) {
	t.Helper()
	previous := augmentationRestoreMaxRecords
	augmentationRestoreMaxRecords = n
	t.Cleanup(func() { augmentationRestoreMaxRecords = previous })
}

// SetAugmentationRestoreMaxBytesForTest lowers the byte ceiling of that
// inventory for the rest of t. Keeper tests do not run in parallel.
func SetAugmentationRestoreMaxBytesForTest(t testing.TB, n int) {
	t.Helper()
	previous := augmentationRestoreMaxBytes
	augmentationRestoreMaxBytes = n
	t.Cleanup(func() { augmentationRestoreMaxBytes = previous })
}

// WriteSeatDepositMarkerForTest writes the seat-deposit marker without
// EnableSeatDeposit's checks, for tests of a state that Enable refuses, such
// as the marker on without review neutrality.
func WriteSeatDepositMarkerForTest(ctx context.Context, k Keeper) error {
	return activation.Enable(ctx, k.storeService, types.SeatDepositEnabledKey, "seat-deposit")
}
