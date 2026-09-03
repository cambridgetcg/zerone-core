package types

import gogoproto "github.com/cosmos/gogoproto/proto"

// Cosmos SDK v0.50's unknown-field checker resolves nested concrete message
// fields through the gogo registry even though this repository generates
// protobuf-v2 types. Register WorkContract so MsgCreateBountyOrder decodes.
func init() {
	if gogoproto.MessageType("zerone.sponsorship.v1.WorkContract") == nil {
		gogoproto.RegisterType((*WorkContract)(nil), "zerone.sponsorship.v1.WorkContract")
	}
}
