package types

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGenesisValidateRejectsNilAdapter(t *testing.T) {
	genesis := DefaultGenesis()
	genesis.Adapters = []*AdapterRegistration{
		{AdapterId: "valid-adapter"},
		nil,
	}

	require.EqualError(t, genesis.Validate(), "adapter at index 1 must not be nil")
}
