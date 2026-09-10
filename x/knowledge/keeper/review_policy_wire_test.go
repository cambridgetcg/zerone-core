package keeper

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/zerone-chain/zerone/x/knowledge/types"
	"google.golang.org/protobuf/encoding/protowire"
)

func TestReviewPolicyRawOverflowCannotBecomeLegacyInRuntimeOrExport(t *testing.T) {
	f := setupHandoff(t)
	require.NoError(t, f.k.EnableRecordIntegrity(f.ctx))
	claim := &types.Claim{Id: "wire-claim"}
	round := &types.VerificationRound{Id: "wire-round", ClaimId: claim.Id, Phase: types.VerificationPhase_VERIFICATION_PHASE_COMMIT}
	require.NoError(t, f.k.SetClaim(f.ctx, claim))
	require.NoError(t, f.k.SetVerificationRound(f.ctx, round))
	store := f.ctx.KVStore(f.keys[0])
	claimKey, roundKey := types.ClaimKey(claim.Id), types.RoundKey(round.Id)
	goodClaim, goodRound := bytes.Clone(store.Get(claimKey)), bytes.Clone(store.Get(roundKey))
	badClaim := protowire.AppendVarint(protowire.AppendTag(bytes.Clone(goodClaim), types.ClaimReviewPolicyField, protowire.VarintType), 1<<32)
	store.Set(claimKey, badClaim)
	_, found := f.k.GetClaim(f.ctx, claim.Id)
	require.False(t, found)
	_, err := f.k.GetAllClaimsChecked(f.ctx)
	require.Error(t, err)
	_, err = decodeClaim(badClaim)
	require.Error(t, err)
	require.Error(t, validateToKStoredRecord(claimKey, badClaim))
	require.Error(t, f.k.SetClaim(f.ctx, claim))
	require.Error(t, f.k.SetVerificationRound(f.ctx, round))
	require.Equal(t, badClaim, store.Get(claimKey))
	store.Set(claimKey, goodClaim)
	badRound := protowire.AppendVarint(protowire.AppendTag(bytes.Clone(goodRound), types.RoundReviewPolicyField, protowire.VarintType), 1<<32)
	store.Set(roundKey, badRound)
	_, found = f.k.GetVerificationRound(f.ctx, round.Id)
	require.False(t, found)
	_, err = f.k.getVerificationRoundChecked(f.ctx, round.Id)
	require.Error(t, err)
	_, err = f.k.GetAllVerificationRoundsChecked(f.ctx)
	require.Error(t, err)
	require.Error(t, validateToKStoredRecord(roundKey, badRound))
	require.Error(t, f.k.SetVerificationRound(f.ctx, round))
	require.Equal(t, badRound, store.Get(roundKey))
}
