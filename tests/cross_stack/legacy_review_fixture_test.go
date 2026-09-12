package cross_stack_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	knowledgekeeper "github.com/zerone-chain/zerone/x/knowledge/keeper"
	knowledgetypes "github.com/zerone-chain/zerone/x/knowledge/types"
)

// useLegacyReviewPolicy selects historical feedback rules only in a fresh
// isolated test context. It is not a runnable downgrade or an application
// startup fixture; current native genesis remains enabled everywhere else.
func useLegacyReviewPolicy(t *testing.T, h *TestHarness) {
	t.Helper()
	store := h.Ctx.KVStore(h.App.GetStoreKeyForTests(knowledgetypes.StoreKey))
	// Input-retention activation depends on neutral review; this historical
	// fixture must clear the later selection as well as its predecessor.
	require.Equal(t, []byte{1}, store.Get([]byte(knowledgekeeper.FundSettlementEnabledStoreKey)))
	store.Delete([]byte(knowledgekeeper.FundSettlementEnabledStoreKey))
	require.Equal(t, []byte{1}, store.Get([]byte(knowledgekeeper.ClaimRecordsEnabledStoreKey)))
	store.Delete([]byte(knowledgekeeper.ClaimRecordsEnabledStoreKey))
	require.Equal(t, []byte{1}, store.Get([]byte(knowledgekeeper.ReviewNeutralityEnabledStoreKey)))
	store.Delete([]byte(knowledgekeeper.ReviewNeutralityEnabledStoreKey))
	enabled, err := h.KnowledgeKeeper.ReviewNeutralityEnabled(h.Ctx)
	require.NoError(t, err)
	require.False(t, enabled)
}
