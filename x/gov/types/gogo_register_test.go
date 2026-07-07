package types

import (
	"testing"

	gogoproto "github.com/cosmos/gogoproto/proto"
)

// TestGogoNestedMsgTypesResolvable pins the tx-decode fix: cosmos-sdk v0.50's
// RejectUnknownFields resolves nested concrete message fields through the
// gogo registry, so every message a Msg embeds (MsgUpdateParams.Params and
// the concrete messages Params embeds) must be gogo-resolvable or the tx
// fails to decode at CheckTx.
func TestGogoNestedMsgTypesResolvable(t *testing.T) {
	for _, name := range []string{
		"zerone.gov.v1.Params",
		"zerone.gov.v1.ParamChange",
		"zerone.gov.v1.ResearchFundVoters",
		"zerone.gov.v1.CategoryConfig",
	} {
		if gogoproto.MessageType(name) == nil {
			t.Fatalf("%s not in the gogo registry — MsgUpdateParams txs will fail to decode", name)
		}
	}
}
