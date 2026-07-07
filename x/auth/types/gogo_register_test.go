package types

import (
	"testing"

	gogoproto "github.com/cosmos/gogoproto/proto"
)

// TestGogoRegisteredParamsResolvable pins the tx-decode fix: cosmos-sdk v0.50's
// RejectUnknownFields resolves nested concrete message fields through the gogo
// registry, so the Params message that gov MsgUpdateParams embeds must be
// gogo-resolvable or the params-update tx fails to decode at CheckTx.
func TestGogoRegisteredParamsResolvable(t *testing.T) {
	for _, name := range []string{
		"zerone.auth.v1.Params",
	} {
		if gogoproto.MessageType(name) == nil {
			t.Fatalf("%s not in the gogo registry — MsgUpdateParams txs will fail to decode", name)
		}
	}
}
