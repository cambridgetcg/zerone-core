package app

import (
	"context"
	"encoding/hex"
	"testing"

	corestore "cosmossdk.io/core/store"
	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

// Exercise the census cache through a keeper API actually used by v2, not a
// bank getter (which masks malformed raw balances and is intentionally avoided).
type writingV2CensusStoreService struct {
	corestore.KVStoreService
	calls *int
}

func (s writingV2CensusStoreService) OpenKVStore(ctx context.Context) corestore.KVStore {
	*s.calls++
	store := s.KVStoreService.OpenKVStore(ctx)
	if err := store.Set([]byte("unwanted-census-write"), []byte{1}); err != nil {
		panic(err)
	}
	sdk.UnwrapSDKContext(ctx).EventManager().EmitEvent(sdk.NewEvent("unwanted-census-event"))
	return store
}

func TestConsensusRepairsV2RawBankBalanceClassification(t *testing.T) {
	coin := sdk.NewCoin("uzrn", sdkmath.NewInt(17))
	legacy, err := coin.Marshal()
	require.NoError(t, err)
	for _, tc := range []struct {
		name  string
		value []byte
		bad   bool
	}{
		{"absent", nil, false}, {"zero", []byte("0"), false}, {"positive", []byte("17"), false},
		{"legacy-coin", legacy, false}, {"malformed", []byte{0xff}, true},
		{"empty", []byte{}, true}, {"negative", []byte("-1"), true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, ctx, _ := consensusRepairsV2Fixture(t)
			reads, err := consensusRepairsV2Reads()
			require.NoError(t, err)
			key := reads["fx7_accounting"][4].key
			if tc.value != nil {
				ctx.KVStore(a.keys["bank"]).Set(key, tc.value)
			}
			ctx, plan := scheduleRepairsV2(t, a, ctx)
			before := claimIntegrityAllStores(t, a, ctx)
			offline, err := a.CensusConsensusRepairsV2(ctx)
			if tc.bad {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			held, _, err := a.censusConsensusRepairsV2(ctx, false)
			require.NoError(t, err)
			found := false
			for _, f := range held.Findings {
				if f.Census == "fx7_accounting" && f.Module == "bank" && f.Key == hex.EncodeToString(key) {
					found = true
					require.Equal(t, "held", f.Outcome)
					require.Contains(t, f.Reason, "backing unknown")
					require.NotContains(t, f.Reason, "deficit_uzrn=", "never infer a deficit from unreadable backing")
				}
			}
			require.Equal(t, tc.bad, found)
			require.Len(t, offline.Findings, len(held.Findings))
			require.Equal(t, before, claimIntegrityAllStores(t, a, ctx))
			require.NoError(t, a.ValidateAccountingAuthorityStartup())
			require.NoError(t, a.UpgradeKeeper.ApplyUpgrade(ctx, plan))
			if tc.value != nil {
				require.Equal(t, tc.value, ctx.KVStore(a.keys["bank"]).Get(key))
			}
		})
	}
}
