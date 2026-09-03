package types

import (
	"testing"

	gogoproto "github.com/cosmos/gogoproto/proto"
)

func TestWorkContractGogoResolvable(t *testing.T) {
	const name = "zerone.sponsorship.v1.WorkContract"
	if gogoproto.MessageType(name) == nil {
		t.Fatalf("%s not in gogo registry; MsgCreateBountyOrder tx decoding will fail", name)
	}
}
