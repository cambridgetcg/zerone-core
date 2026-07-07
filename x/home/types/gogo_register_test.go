package types

import (
	"testing"

	gogoproto "github.com/cosmos/gogoproto/proto"
)

// TestGogoNestedMsgTypesResolvable pins the tx-decode fix: cosmos-sdk v0.50's
// RejectUnknownFields resolves nested concrete message fields through the gogo
// registry, so every message the home Msgs embed (MsgUpdateParams.Params, and
// Params' nested HomeGuardian and DeadmanConfig) must be gogo-resolvable or the
// tx fails to decode at CheckTx.
func TestGogoNestedMsgTypesResolvable(t *testing.T) {
	for _, name := range []string{
		"zerone.home.v1.Params",
		"zerone.home.v1.HomeGuardian",
		"zerone.home.v1.DeadmanConfig",
	} {
		if gogoproto.MessageType(name) == nil {
			t.Fatalf("%s not in the gogo registry — home MsgUpdateParams txs will fail to decode", name)
		}
	}
}
