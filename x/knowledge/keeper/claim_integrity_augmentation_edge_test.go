package keeper_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/zerone-chain/zerone/x/knowledge/keeper"
	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// Review G1R: the augmentation edge writer stays out of the fact ID space.
// A passing augmentation verdict (ApplyFinalizedAugmentationVerdict) writes a
// REFORMULATES edge whose source is the augmentation's user-chosen ID. With
// the marker on it writes none when that ID contains '/' (it would land under
// another ID's forward-index prefix), is longer than
// keeper.AugmentationEdgeSourceMaxBytes, or is the ID of a stored fact (it
// would overwrite that fact's own edge to the same target). The verdict is
// otherwise applied unchanged, and augmentation_edge_withheld says why. With
// the marker off every passing verdict writes its edge byte for byte as
// knowledge 11 does, including the overwrite.
func TestClaimIntegrity_AugmentationEdgeStaysOutOfTheFactIDSpace(t *testing.T) {
	const withheldEvent = "zerone.knowledge.augmentation_edge_withheld"
	require.Equal(t, 128, keeper.AugmentationEdgeSourceMaxBytes)
	for _, tc := range []struct {
		name   string
		id     func(factF string) string
		reason string // "" when the edge is written with the marker on
	}{
		{"ordinary ID", func(string) string { return "v12-g1r-variant-1" }, ""},
		{"ID at the length cap", func(string) string { return strings.Repeat("a", keeper.AugmentationEdgeSourceMaxBytes) }, ""},
		{"ID extending a fact's ID", func(factF string) string { return factF + "/0" }, "augmentation id contains '/'"},
		{"ID with a slash elsewhere", func(string) string { return "v12/g1r" }, "augmentation id contains '/'"},
		{"ID over the length cap", func(string) string { return strings.Repeat("a", keeper.AugmentationEdgeSourceMaxBytes+1) }, "augmentation id is longer than 128 bytes"},
		{"ID equal to a fact's ID", func(factF string) string { return factF }, "augmentation id is the id of a fact"},
	} {
		for _, sel := range dedupSelections {
			t.Run(sel.name+"/"+tc.name, func(t *testing.T) {
				f := sel.setup(t)
				_, factF, factD := f.vdDependent(t, "ae")
				augID := tc.id(factF)
				ownEdge := f.ctx.KVStore(f.key).Get(types.FactRelationKey(factF, factD))
				require.NotEmpty(t, ownEdge)
				f.freshEvents()
				f.vdPassAugmentation(t, augID, factD)

				aug, found := f.k.GetAugmentation(f.ctx, augID)
				require.True(t, found)
				require.Equal(t, types.AugmentationVerdict_AUGMENTATION_VERDICT_EQUIVALENT, aug.Verdict)
				require.True(t, aug.Accepted, "the verdict is applied either way")
				require.Equal(t, uint64(f.ctx.BlockHeight()), aug.VerdictBlock)

				forward := f.ctx.KVStore(f.key).Get(types.FactRelationKey(augID, factD))
				reverse := f.ctx.KVStore(f.key).Get(types.FactRelationReverseKey(factD, augID))
				withheld := v11Event(f.ctx, withheldEvent)
				if sel.integrity && tc.reason != "" {
					if augID == factF {
						require.Equal(t, ownEdge, forward, "F's own edge to D is not overwritten")
						require.Equal(t, ownEdge, reverse)
					} else {
						require.Nil(t, forward, "no edge is written")
						require.Nil(t, reverse)
					}
					require.Equal(t, []map[string]string{{"augmentation_id": augID, "original_fact_id": factD, "reason": tc.reason}}, withheld)
					return
				}
				want, err := proto.Marshal(&types.FactRelation{
					SourceFactId: augID, TargetFactId: factD, Relation: types.RelationType_RELATION_TYPE_REFORMULATES, CreatedAtBlock: aug.VerdictBlock,
				})
				require.NoError(t, err)
				require.Equal(t, want, forward, "the edge is written as knowledge 11 writes it")
				require.Equal(t, want, reverse)
				require.Empty(t, withheld)
			})
		}
	}
}

// The gate reads the marker before any write of the verdict, the bounty
// payout included. A malformed marker is an error, never knowledge 11: the
// verdict is refused, no coins move and nothing is written. A failing verdict
// reads no marker.
func TestClaimIntegrity_AugmentationEdgeGateFailsClosed(t *testing.T) {
	f := setupClaimIntegrityTest(t)
	_, factF, factD := f.vdDependent(t, "af")
	augID, bountyID := factF+"/0", "v12-g1r-bounty"
	require.NoError(t, f.k.SetAugmentationBounty(f.ctx, &types.AugmentationBounty{
		Id: bountyID, TargetFactId: factD, RewardPerVariant: 1000, MaxVariants: 2, Active: true,
	}))
	f.ctx.KVStore(f.key).Set(types.TrainingFundEscrowLockedKey(bountyID), []byte("2000"))
	require.NoError(t, f.k.SetAugmentation(f.ctx, &types.Augmentation{
		Id: augID, BountyId: bountyID, OriginalFactId: factD, Submitter: v11Addr("v12-g1r-aug-author"),
		VariantContent: "Synthetic v12 review G1R: a variant of fact D.",
	}))
	f.ctx.KVStore(f.key).Set([]byte(keeper.ClaimIntegrityEnabledStoreKey), []byte{2})
	before, sends := relationContractSnapshot(t, f.ctx, f.key), len(f.bank.sendCalls)
	f.freshEvents()
	err := f.k.ApplyFinalizedAugmentationVerdict(f.ctx, augID, types.AugmentationVerdict_AUGMENTATION_VERDICT_EQUIVALENT)
	require.ErrorContains(t, err, "malformed claim integrity marker")
	require.Equal(t, before, relationContractSnapshot(t, f.ctx, f.key), "nothing is written")
	require.Len(t, f.bank.sendCalls, sends, "no coins move")
	require.Empty(t, f.ctx.EventManager().Events())

	require.NoError(t, f.k.ApplyFinalizedAugmentationVerdict(f.ctx, augID, types.AugmentationVerdict_AUGMENTATION_VERDICT_DRIFT))
	aug, found := f.k.GetAugmentation(f.ctx, augID)
	require.True(t, found)
	require.Equal(t, types.AugmentationVerdict_AUGMENTATION_VERDICT_DRIFT, aug.Verdict)
}
