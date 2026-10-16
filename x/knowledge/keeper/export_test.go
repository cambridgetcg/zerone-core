package keeper

import "testing"

// SetContestScanMaxRecordsForTest lowers the record ceiling of one contest
// check for the rest of t, so a test reaches the bound with a few records.
// Keeper tests do not run in parallel.
func SetContestScanMaxRecordsForTest(t testing.TB, n int) {
	t.Helper()
	previous := contestScanMaxRecords
	contestScanMaxRecords = n
	t.Cleanup(func() { contestScanMaxRecords = previous })
}
