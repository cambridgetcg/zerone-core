package types

import (
	"testing"

	gogoproto "github.com/cosmos/gogoproto/proto"
)

// TestGogoNestedMsgTypesResolvable pins the tx-decode fix: cosmos-sdk v0.50's
// RejectUnknownFields resolves nested concrete message fields through the gogo
// registry, so every message gov's MsgUpdateParams embeds (Params and its
// nested RevenueSplit, ProtocolSubSplit, CategoryRewardConfig) must be
// gogo-resolvable or the tx fails to decode at CheckTx.
func TestGogoNestedMsgTypesResolvable(t *testing.T) {
	for _, name := range []string{
		"zerone.vesting_rewards.v1.Params",
		"zerone.common.v1.RevenueSplit",
		"zerone.common.v1.ProtocolSubSplit",
		"zerone.vesting_rewards.v1.CategoryRewardConfig",
	} {
		if gogoproto.MessageType(name) == nil {
			t.Fatalf("%s not in the gogo registry — MsgUpdateParams txs will fail to decode", name)
		}
	}
}
