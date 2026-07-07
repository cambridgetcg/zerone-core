package types

import (
	"testing"

	gogoproto "github.com/cosmos/gogoproto/proto"
)

// TestNestedMsgTypesGogoResolvable pins the tx-decode fix: cosmos-sdk v0.50's
// RejectUnknownFields resolves nested concrete message fields through the gogo
// registry, so every message the gov MsgUpdateParams embeds (Params and the
// concrete messages Params carries) must be gogo-resolvable or the proposal
// tx fails to decode at CheckTx.
func TestNestedMsgTypesGogoResolvable(t *testing.T) {
	for _, name := range []string{
		"zerone.knowledge.v1.Params",
		"zerone.knowledge.v1.ClaimStructure",
		"zerone.knowledge.v1.ClaimRelation",
		"zerone.knowledge.v1.DemandReport",
		"zerone.knowledge.v1.TokenizerSpec",
		"zerone.knowledge.v1.TraceSchema",
		"zerone.knowledge.v1.CorpusSelector",
	} {
		if gogoproto.MessageType(name) == nil {
			t.Fatalf("%s not in the gogo registry — knowledge MsgUpdateParams txs will fail to decode", name)
		}
	}
}
