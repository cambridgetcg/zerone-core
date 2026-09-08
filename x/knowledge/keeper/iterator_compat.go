package keeper

import corestore "cosmossdk.io/core/store"

// feedbackIteratorError is shared by checked feedback/history and bounded reads.
// Pinned cosmossdk.io/store@v1.1.2 cachekv/internal/mergeiterator.go:148-154
// returns this unexported diagnostic at ordinary EOF instead of nil. Recognize
// ONLY that exact diagnostic while exhausted; propagate all other exposed errors.
// Callers must still close the iterator and preserve their checked-close policy.
//
// The pinned cacheMergeIterator never forwards parent/cache Error() results;
// Valid() cannot distinguish their failures from normal exhaustion. This helper
// cannot recover those hidden errors or certify underlying storage health. It
// only adapts the exposed SDK contract, without weakening caller scan budgets.
func feedbackIteratorError(it corestore.Iterator) error {
	err := it.Error()
	if err != nil && !it.Valid() && err.Error() == "invalid cacheMergeIterator" {
		return nil
	}
	return err
}
