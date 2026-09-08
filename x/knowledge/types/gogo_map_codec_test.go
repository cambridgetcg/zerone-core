package types_test

import (
	"bytes"
	"testing"

	"github.com/cosmos/cosmos-sdk/codec/unknownproto"
	gogo "github.com/cosmos/gogoproto/proto"
	"github.com/stretchr/testify/require"
	"github.com/zerone-chain/zerone/x/knowledge/types"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"
)

func mapBytes(tag protowire.Number, value []byte) []byte {
	return protowire.AppendBytes(protowire.AppendTag(nil, tag, protowire.BytesType), value)
}

func TestGeneratedGogoMapValidation(t *testing.T) {
	for _, tc := range []struct {
		name string
		msg  interface {
			proto.Message
			gogo.Message
		}
		tag   protowire.Number
		value []byte
	}{
		{"params", &types.Params{MethodologyNormalizationBps: map[string]uint64{"x": 1}}, 140, []byte{0x10, 1}},
		{"methodology", &types.Methodology{CrossMethodDiscountBps: map[string]uint64{"x": 1}}, 6, []byte{0x10, 1}},
		{"message-value", &types.AgentCalibration{PerMethod: map[string]*types.AgentMethodStats{"x": {Submissions: 1}}}, 15, mapBytes(2, []byte{8, 1})},
	} {
		t.Run(tc.name, func(t *testing.T) {
			valid, err := proto.Marshal(tc.msg)
			require.NoError(t, err)
			require.NoError(t, unknownproto.RejectUnknownFieldsStrict(valid, tc.msg, unknownproto.DefaultAnyResolver{}))
			entry := append(mapBytes(1, []byte("x")), tc.value...)
			require.True(t, bytes.Equal(valid, mapBytes(tc.tag, entry)))
			// Validation must recurse into synthetic entries; their fields are not
			// an opaque bytes escape hatch. Noncritical semantics remain SDK-owned.
			bad := append(bytes.Clone(entry), 0x18, 1)
			require.Error(t, unknownproto.RejectUnknownFieldsStrict(mapBytes(tc.tag, bad), tc.msg, unknownproto.DefaultAnyResolver{}))
			bad = protowire.AppendVarint(protowire.AppendTag(bytes.Clone(entry), 1024, protowire.VarintType), 1)
			require.Error(t, unknownproto.RejectUnknownFieldsStrict(mapBytes(tc.tag, bad), tc.msg, unknownproto.DefaultAnyResolver{}))
			noncritical, err := unknownproto.RejectUnknownFields(mapBytes(tc.tag, bad), tc.msg, true, unknownproto.DefaultAnyResolver{})
			require.NoError(t, err)
			require.True(t, noncritical)
			bad = append(mapBytes(1, []byte("x")), protowire.AppendFixed64(protowire.AppendTag(nil, 2, protowire.Fixed64Type), 1)...)
			require.Error(t, unknownproto.RejectUnknownFieldsStrict(mapBytes(tc.tag, bad), tc.msg, unknownproto.DefaultAnyResolver{}))
			require.Error(t, unknownproto.RejectUnknownFieldsStrict(protowire.AppendVarint(protowire.AppendTag(nil, tc.tag, protowire.VarintType), 1), tc.msg, unknownproto.DefaultAnyResolver{}))
			require.Error(t, unknownproto.RejectUnknownFieldsStrict(valid[:len(valid)-1], tc.msg, unknownproto.DefaultAnyResolver{}))
		})
	}
	// A message-valued map must validate the value's own descriptor as well.
	badValue := mapBytes(15, append(mapBytes(1, []byte("x")), mapBytes(2, []byte{0x30, 1})...))
	require.Error(t, unknownproto.RejectUnknownFieldsStrict(badValue, &types.AgentCalibration{}, unknownproto.DefaultAnyResolver{}))
}
