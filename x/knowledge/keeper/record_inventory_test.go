package keeper

import (
	"github.com/stretchr/testify/require"
	"github.com/zerone-chain/zerone/x/knowledge/types"
	"google.golang.org/protobuf/proto"
	"testing"
)

func TestRecordIntegrityMarkerRefusesCorruptionAndReadFailure(t *testing.T) {
	f := setupHandoff(t)
	enabled, err := f.k.RecordIntegrityEnabled(f.ctx)
	require.NoError(t, err)
	require.False(t, enabled)
	*f.kFault = handoffFault{"get", []byte(RecordIntegrityEnabledStoreKey), false}
	enabled, err = f.k.RecordIntegrityEnabled(f.ctx)
	require.Error(t, err)
	require.False(t, enabled)
	require.Error(t, f.k.EnableRecordIntegrity(f.ctx))
	*f.kFault = handoffFault{}
	store := f.ctx.KVStore(f.keys[0])
	require.False(t, store.Has([]byte(RecordIntegrityEnabledStoreKey)))
	for _, invalid := range [][]byte{{0}, {1, 0}, {2}} {
		store.Set([]byte(RecordIntegrityEnabledStoreKey), invalid)
		enabled, err = f.k.RecordIntegrityEnabled(f.ctx)
		require.Error(t, err)
		require.False(t, enabled)
		require.Error(t, f.k.EnableRecordIntegrity(f.ctx))
		require.Equal(t, invalid, store.Get([]byte(RecordIntegrityEnabledStoreKey)))
	}
	store.Delete([]byte(RecordIntegrityEnabledStoreKey))
	require.NoError(t, f.k.EnableRecordIntegrity(f.ctx))
	before := f.snapshot(t)
	require.NoError(t, f.k.EnableRecordIntegrity(f.ctx))
	require.Equal(t, before, f.snapshot(t))
}

func TestRecordPrimaryInventoriesRefuseMalformedAndPreserveRawLegacy(t *testing.T) {
	for _, kind := range []string{"round-key", "round-wire", "unknown-fields", "unknown-scheme", "preseeded-v2", "claim-key"} {
		t.Run(kind, func(t *testing.T) {
			f := setupHandoff(t)
			store := f.ctx.KVStore(f.keys[0])
			round := &types.VerificationRound{Id: "round", ClaimId: "claim", Phase: types.VerificationPhase_VERIFICATION_PHASE_COMMIT}
			key := types.RoundKey(round.Id)
			if kind == "round-key" {
				key = types.RoundKey("wrong")
			}
			if kind == "unknown-scheme" {
				round.CommitmentScheme = 1
			}
			if kind == "preseeded-v2" {
				round.CommitmentScheme = 2
				round.CommitmentChainId = "original"
			}
			raw, err := proto.Marshal(round)
			require.NoError(t, err)
			if kind == "round-wire" {
				raw = []byte{0xff}
			}
			if kind == "unknown-fields" {
				raw = append(raw, 0xa0, 0x06, 0x01)
			}
			store.Set(key, raw)
			before := f.snapshot(t)
			if kind == "claim-key" {
				claimRaw, err := proto.Marshal(&types.Claim{Id: "different"})
				require.NoError(t, err)
				store.Set(types.ClaimKey("claim"), claimRaw)
				before = f.snapshot(t)
				_, err = f.k.GetAllClaimsChecked(f.ctx)
				require.Error(t, err)
			} else {
				require.Error(t, f.k.ValidateRecordIntegrityActivation(f.ctx))
			}
			require.Equal(t, before, f.snapshot(t))
		})
	}
	f := setupHandoff(t)
	legacy := &types.VerificationRound{Id: "old", ClaimId: "old-claim", Phase: types.VerificationPhase_VERIFICATION_PHASE_COMPLETE}
	raw, err := proto.Marshal(legacy)
	require.NoError(t, err)
	f.ctx.KVStore(f.keys[0]).Set(types.RoundKey(legacy.Id), raw)
	cache, _ := f.ctx.CacheContext()
	require.NoError(t, f.k.ValidateRecordIntegrityActivation(cache), "normal nested cache iterator exhaustion is not corruption")
	require.Equal(t, raw, f.ctx.KVStore(f.keys[0]).Get(types.RoundKey(legacy.Id)))
}
