package types

import (
	"testing"

	gogoproto "github.com/cosmos/gogoproto/proto"
)

// TestNestedMsgTypesGogoResolvable pins the tx-decode fix: cosmos-sdk v0.50's
// RejectUnknownFields resolves nested concrete message fields through the
// gogo registry, so every message a Msg embeds (MsgUpdateParams.Params) must
// be gogo-resolvable or the tx fails to decode at CheckTx.
func TestNestedMsgTypesGogoResolvable(t *testing.T) {
	if gogoproto.MessageType("zerone.qualification.v1.Params") == nil {
		t.Fatal("zerone.qualification.v1.Params not in the gogo registry — MsgUpdateParams txs will fail to decode")
	}
}
