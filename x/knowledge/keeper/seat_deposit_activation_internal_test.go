package keeper

import (
	"fmt"
	"testing"

	storetypes "cosmossdk.io/store/types"
	"github.com/stretchr/testify/require"

	"github.com/zerone-chain/zerone/internal/activation"
	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// The seat-deposit marker (consensus-repairs-v3, 0x97) follows the v3 marker
// contract: a pinned key, a gas-free read, a genesis field written on both
// import paths and exported exactly, and a fault that is never read as off.
// Native genesis selects it, like every other v3 marker. Selecting it needs its
// predecessor markers, so it has its own table here instead of a row in
// v3GenesisMarkers (whose rows select one marker alone); the module-side table
// knowledgeV3Markers has its row.

func TestSeatDepositMarkerKeyAndNativeGenesis(t *testing.T) {
	require.Equal(t, []byte{0x97}, types.SeatDepositEnabledKey)
	for _, other := range [][]byte{types.DoctrineIdentityEnabledKey, types.DormantFactContestEnabledKey, types.ReviewerBalanceParamEnabledKey} {
		require.NotEqual(t, other, types.SeatDepositEnabledKey)
	}
	gs := types.DefaultGenesis()
	require.True(t, gs.SeatDepositEnabled, "a new chain starts with the seat deposit selected (the design's section 7)")
	require.NoError(t, gs.Validate())
}

func TestSeatDepositMarkerFollowsBothImportPaths(t *testing.T) {
	for _, accounting := range []bool{false, true} {
		for _, selected := range []bool{false, true} {
			t.Run(fmt.Sprintf("accounting=%t/selected=%t", accounting, selected), func(t *testing.T) {
				f := setupRelationGenesis(t, nil)
				f.k.bankKeeper = accountingZeroBank{}
				gs := types.DefaultGenesis()
				if accounting {
					gs = accountingEmptyGenesis()
				}
				gs.SeatDepositEnabled = selected
				require.NoError(t, f.k.InitGenesis(f.ctx, gs))
				stored := f.ctx.KVStore(f.keys[0]).Get(types.SeatDepositEnabledKey)
				if selected {
					require.Equal(t, []byte{1}, stored)
				} else {
					require.Nil(t, stored, "off writes no marker")
				}
				enabled, err := f.k.SeatDepositEnabled(f.ctx)
				require.NoError(t, err)
				require.Equal(t, selected, enabled)
				exported := f.k.ExportGenesis(f.ctx)
				require.Equal(t, selected, exported.SeatDepositEnabled)
				// The other v3 markers are untouched by this field.
				require.True(t, exported.ReviewerBalanceParamEnabled)
				require.True(t, exported.DormantFactContestEnabled)
				require.True(t, exported.DoctrineIdentityEnabled)
			})
		}
	}
}

func TestSeatDepositMarkerFaultIsNeverOff(t *testing.T) {
	f := setupRelationGenesis(t, types.DefaultGenesis())
	f.ctx.KVStore(f.keys[0]).Set(types.SeatDepositEnabledKey, []byte{2})
	_, err := f.k.SeatDepositEnabled(f.ctx)
	require.ErrorIs(t, err, activation.ErrMarkerFault)
	require.ErrorContains(t, err, "malformed seat-deposit marker")
	require.ErrorIs(t, f.k.EnableSeatDeposit(f.ctx), activation.ErrMarkerFault, "enable never repairs a fault")
	var recovered any
	func() { defer func() { recovered = recover() }(); f.k.ExportGenesis(f.ctx) }()
	panicErr, ok := recovered.(error)
	require.True(t, ok, "export panics with an error, got %v", recovered)
	require.ErrorIs(t, panicErr, activation.ErrMarkerFault)
}

func TestSeatDepositMarkerReadIsGasFree(t *testing.T) {
	gs := types.DefaultGenesis()
	gs.SeatDepositEnabled = false
	f := setupRelationGenesis(t, gs)
	meter := storetypes.NewGasMeter(1_000_000)
	ctx := f.ctx.WithGasMeter(meter)
	enabled, err := f.k.SeatDepositEnabled(ctx)
	require.NoError(t, err)
	require.False(t, enabled)
	require.Zero(t, meter.GasConsumed())
	require.NoError(t, f.k.EnableSeatDeposit(f.ctx))
	enabled, err = f.k.SeatDepositEnabled(ctx)
	require.NoError(t, err)
	require.True(t, enabled)
	require.Zero(t, meter.GasConsumed())
}
