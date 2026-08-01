package contentid

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"

	cid "github.com/ipfs/go-cid"
	multihash "github.com/multiformats/go-multihash"
	"github.com/stretchr/testify/require"
)

const canonicalCIDv1 = "bafzbeigai3eoy2ccc7ybwjfz5r3rdxqrinwi4rwytly24tdbh6yk7zslrm"

type cidVector struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Parse  bool   `json:"parse"`
	Memory bool   `json:"memory"`
}

func TestParseMemoryV1ByteBoundary(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name        string
		digestBytes int
		textBytes   int
		wantMemory  bool
	}{
		{name: "256 bytes", digestBytes: 154, textBytes: 256, wantMemory: true},
		{name: "257 bytes", digestBytes: 155, textBytes: 257, wantMemory: false},
	} {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			digest, err := multihash.Encode(bytes.Repeat([]byte{'a'}, test.digestBytes), multihash.IDENTITY)
			require.NoError(t, err)
			value := cid.NewCidV1(cid.Raw, digest).String()
			require.Len(t, value, test.textBytes)

			_, err = ParseCanonicalV1(value)
			require.NoError(t, err)
			_, err = ParseMemoryV1(value)
			if test.wantMemory {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, "exceeds")
			}
		})
	}
}

func loadCIDVectors(t *testing.T) []cidVector {
	t.Helper()

	data, err := os.ReadFile("../../../testdata/cid-v1-vectors.json")
	require.NoError(t, err)

	var vectors []cidVector
	require.NoError(t, json.Unmarshal(data, &vectors))
	require.NotEmpty(t, vectors)
	return vectors
}

func TestParseCanonicalV1(t *testing.T) {
	t.Parallel()

	parsed, err := ParseCanonicalV1(canonicalCIDv1)
	require.NoError(t, err)
	require.Equal(t, uint64(1), parsed.Version())
	require.Equal(t, canonicalCIDv1, parsed.String())
}

func TestParseCanonicalV1SharedVectors(t *testing.T) {
	t.Parallel()

	for _, vector := range loadCIDVectors(t) {
		vector := vector
		t.Run(vector.Name, func(t *testing.T) {
			t.Parallel()
			_, err := ParseCanonicalV1(vector.Value)
			if vector.Parse {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
			require.Equal(t, vector.Memory, vector.Parse && len(vector.Value) <= 256)
		})
	}
}
