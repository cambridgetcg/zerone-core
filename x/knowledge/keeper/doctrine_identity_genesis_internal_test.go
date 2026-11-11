package keeper

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/zerone-chain/zerone/internal/activation"
	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// The consensus-repairs-v3 marker is imported and exported outside the pinned
// legacy genesis bodies, so it must follow both InitGenesis paths: a file
// without an accounting inventory (initGenesisLegacy alone) and a file with one
// (the atomic accounting path).
func TestDoctrineIdentityMarkerFollowsBothImportPaths(t *testing.T) {
	for _, accounting := range []bool{false, true} {
		for _, on := range []bool{false, true} {
			t.Run(fmt.Sprintf("accounting=%t/on=%t", accounting, on), func(t *testing.T) {
				f := setupRelationGenesis(t, nil)
				f.k.bankKeeper = accountingZeroBank{}
				gs := types.DefaultGenesis()
				if accounting {
					gs = accountingEmptyGenesis()
				}
				gs.DoctrineIdentityEnabled = on
				require.NoError(t, f.k.InitGenesis(f.ctx, gs))
				stored := f.ctx.KVStore(f.keys[0]).Get(types.DoctrineIdentityEnabledKey)
				if on {
					require.Equal(t, []byte{1}, stored)
				} else {
					require.Nil(t, stored, "off writes no marker")
				}
				require.Equal(t, on, f.k.ExportGenesis(f.ctx).DoctrineIdentityEnabled)
			})
		}
	}
}

// A malformed v3 marker panics on export with the activation.ErrMarkerFault
// chain, like the markers that exportGenesisLegacy reads.
func TestDoctrineIdentityMarkerFaultRefusesExport(t *testing.T) {
	f := setupRelationGenesis(t, types.DefaultGenesis())
	f.ctx.KVStore(f.keys[0]).Set(types.DoctrineIdentityEnabledKey, []byte{2})
	var recovered any
	func() { defer func() { recovered = recover() }(); f.k.ExportGenesis(f.ctx) }()
	require.ErrorIs(t, recovered.(error), activation.ErrMarkerFault)
}
