package types

import (
	"testing"

	gogoproto "github.com/cosmos/gogoproto/proto"
)

// TestNestedMsgTypesGogoResolvable pins the tx-decode fix: cosmos-sdk v0.50's
// RejectUnknownFields resolves nested concrete message fields through the
// gogo registry, so every message gov's MsgUpdateParams embeds (Params and
// its nested PinnedCreed, CreedCouncilMember, CommitmentEntry) must be
// gogo-resolvable or the tx fails to decode at CheckTx.
func TestNestedMsgTypesGogoResolvable(t *testing.T) {
	for _, name := range []string{
		"zerone.creed.v1.Params",
		"zerone.creed.v1.PinnedCreed",
		"zerone.creed.v1.CreedCouncilMember",
		"zerone.creed.v1.CommitmentEntry",
	} {
		if gogoproto.MessageType(name) == nil {
			t.Fatalf("%s not in the gogo registry — MsgUpdateParams txs will fail to decode", name)
		}
	}
}
