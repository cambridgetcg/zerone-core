package types

import (
	"testing"

	gogoproto "github.com/cosmos/gogoproto/proto"
)

// TestNestedMsgTypesGogoResolvable pins the tx-decode fix: cosmos-sdk v0.50's
// RejectUnknownFields resolves nested concrete message fields through the
// gogo registry, so every message a Msg embeds (MsgUpdateParams.Params,
// MsgRegisterLogicZone.ZoneProperties) must be gogo-resolvable or the tx
// fails to decode at CheckTx.
func TestNestedMsgTypesGogoResolvable(t *testing.T) {
	for _, name := range []string{
		"zerone.ontology.v1.Params",
		"zerone.ontology.v1.LogicZoneProperties",
	} {
		if gogoproto.MessageType(name) == nil {
			t.Fatalf("%s not in the gogo registry — txs carrying it will fail to decode", name)
		}
	}
}
