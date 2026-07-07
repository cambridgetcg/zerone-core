package types

import (
	"testing"

	gogoproto "github.com/cosmos/gogoproto/proto"
)

// TestGogoNestedMsgTypesResolvable pins the tx-decode fix: cosmos-sdk v0.50's
// RejectUnknownFields resolves nested concrete message fields through the gogo
// registry, so every message gov's MsgUpdateParams embeds (Params, and its
// nested TierConfig via `repeated TierConfig tier_configs`) must be
// gogo-resolvable or the params-update tx fails to decode at CheckTx.
func TestGogoNestedMsgTypesResolvable(t *testing.T) {
	for _, name := range []string{
		"zerone.staking.v1.Params",
		"zerone.staking.v1.TierConfig",
	} {
		if gogoproto.MessageType(name) == nil {
			t.Fatalf("%s not in the gogo registry — MsgUpdateParams txs will fail to decode", name)
		}
	}
}
