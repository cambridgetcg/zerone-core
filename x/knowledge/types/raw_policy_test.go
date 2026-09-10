package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/zerone-chain/zerone/x/knowledge/types"
	"google.golang.org/protobuf/encoding/protowire"
)

func TestRawReviewPolicyRefusesLossyOrAmbiguousWireValues(t *testing.T) {
	for _, field := range []protowire.Number{types.ClaimReviewPolicyField, types.RoundReviewPolicyField, 9} {
		encode := func(v uint64) []byte {
			return protowire.AppendVarint(protowire.AppendTag(nil, field, protowire.VarintType), v)
		}
		for _, v := range []uint64{0, 1} {
			present, err := types.RawPolicyFieldPresent(encode(v), field)
			require.NoError(t, err)
			require.True(t, present)
		}
		present, err := types.RawPolicyFieldPresent(protowire.AppendString(protowire.AppendTag(nil, 1, protowire.BytesType), "ordinary-record"), field)
		require.NoError(t, err)
		require.False(t, present)
		for _, bad := range [][]byte{
			encode(2), encode(1 << 32), encode((1 << 32) + 1),
			append(encode(1), encode(0)...),
			protowire.AppendString(protowire.AppendTag(nil, field, protowire.BytesType), "1"),
			append(protowire.AppendTag(nil, field, protowire.VarintType), 0x80, 0),
			append(protowire.AppendTag(nil, field, protowire.VarintType), 0x80),
			{0xff}, protowire.AppendTag(nil, 3, protowire.StartGroupType),
		} {
			require.Error(t, types.ValidateRawPolicyField(bad, field))
		}
	}
}
