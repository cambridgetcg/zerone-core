package cross_stack_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	knowledgetypes "github.com/zerone-chain/zerone/x/knowledge/types"
)

// TestKnowledgeUpdateParamsCarriesReviewerBalance drives a knowledge
// MsgUpdateParams that sets min_reviewer_balance (Params field 161) through the
// app's TxEncoder and TxDecoder, which run unknownproto.RejectUnknownFields: the
// new field is part of the registered Params descriptor and is not refused as
// unknown.
//
// The methodology map is emptied on purpose. With it non-empty, as in
// DefaultParams, the same decode fails with "failed to retrieve the message of
// type zerone.knowledge.v1.Params.MethodologyNormalizationBpsEntry": the map's
// entry type is not in the gogo registry. That defect predates this field and is
// not repaired here; until it is, a real governance update of knowledge params
// on a chain whose params carry the map cannot be decoded, whatever it sets.
func TestKnowledgeUpdateParamsCarriesReviewerBalance(t *testing.T) {
	h := NewTestHarness(t)
	txConfig := h.App.TxConfig()

	p := knowledgetypes.DefaultParams()
	p.MinReviewerBalance = "222000"
	p.MethodologyNormalizationBps = nil
	builder := txConfig.NewTxBuilder()
	require.NoError(t, builder.SetMsgs(&knowledgetypes.MsgUpdateParams{
		Authority: "zerone1updateparamsauthority00000000001",
		Params:    &p,
	}))
	bz, err := txConfig.TxEncoder()(builder.GetTx())
	require.NoError(t, err)

	decoded, err := txConfig.TxDecoder()(bz)
	require.NoError(t, err)
	msgs := decoded.GetMsgs()
	require.Len(t, msgs, 1)
	update, ok := msgs[0].(*knowledgetypes.MsgUpdateParams)
	require.True(t, ok)
	require.Equal(t, "222000", update.Params.MinReviewerBalance)
}
