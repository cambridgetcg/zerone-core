package caip

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCosmosChainID(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		chainID string
		want    string
		wantErr bool
	}{
		"mainnet":             {chainID: "zerone-2", want: "cosmos:zerone-2"},
		"testnet":             {chainID: "zerone-testnet-1", want: "cosmos:zerone-testnet-1"},
		"direct hash prefix":  {chainID: "hash-", want: "cosmos:hash-"},
		"direct hashed word":  {chainID: "hashed", want: "cosmos:hashed"},
		"reserved prefix":     {chainID: "hashed-", want: "cosmos:hashed-c904589232422def"},
		"underscore fallback": {chainID: "local_dev-1", want: "cosmos:hashed-556aa1eb6beeff49"},
		"space fallback":      {chainID: " ", want: "cosmos:hashed-36a9e7f1c95b82ff"},
		"unicode fallback":    {chainID: "wonderland🧝‍♂️", want: "cosmos:hashed-843d2fc87f40eeb9"},
		"long fallback": {
			chainID: "123456789012345678901234567890123456789012345678",
			want:    "cosmos:hashed-0204c92a0388779d",
		},
		"empty": {chainID: "", wantErr: true},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got, err := CosmosChainID(tc.chainID)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}
