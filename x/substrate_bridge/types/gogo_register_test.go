package types

import (
	"testing"

	gogoproto "github.com/cosmos/gogoproto/proto"
)

// TestGogoRegistration guards the fix in gogo_register.go. Every substrate_bridge
// message that can appear as a NESTED concrete field of a tx message must be
// resolvable via gogoproto.MessageType — otherwise cosmos-sdk's unknownproto
// rejects the whole tx at decode time ("failed to retrieve the message of
// type ..."), which is what left this module's attestation path unusable.
func TestGogoRegistration(t *testing.T) {
	mustResolve := []string{
		"zerone.substrate_bridge.v1.SubstrateLink",       // MsgSubmitExternalAttestation.link
		"zerone.substrate_bridge.v1.AdapterRegistration", // MsgRegisterAdapter.adapter
		"zerone.substrate_bridge.v1.AxisBounds",          // AdapterRegistration.axis_bounds
		"zerone.substrate_bridge.v1.SlashGradient",       // AdapterRegistration.slash_gradient
	}
	for _, n := range mustResolve {
		if gogoproto.MessageType(n) == nil {
			t.Errorf("%s is not in the gogo registry — a tx carrying it will fail to decode", n)
		}
	}
}
