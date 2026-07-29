package keeper_test

import (
	"crypto/sha256"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/zerone-chain/zerone/x/substrate_bridge/keeper"
	"github.com/zerone-chain/zerone/x/substrate_bridge/types"
)

func validExternalSource(adapterID string) *types.ExternalSource {
	digest := sha256.Sum256([]byte("declared-source:" + adapterID))
	return &types.ExternalSource{
		AdapterId:   adapterID,
		SourceId:    "source-1",
		ContentHash: digest[:],
	}
}

func TestComputeLinkHash_Deterministic(t *testing.T) {
	link := &types.SubstrateLink{
		AdapterId:       "wiki-v1",
		CitedFacts:      []*types.FactCitation{{FactId: "fact-1", CitationType: types.CitationType_CITATION_TYPE_SUPPORTS}},
		PendingClaims:   []*types.PendingClaim{{ClaimContent: "X is Y", Domain: "history", MethodologyId: "wiki-cite"}},
		RecursionWeight: &types.AxisProjection{AxisSubstrate: 100},
		Source:          validExternalSource("wiki-v1"),
	}
	h1 := keeper.ComputeLinkHash(link)
	h2 := keeper.ComputeLinkHash(link)
	require.Equal(t, h1, h2)
	require.Len(t, h1, sha256.Size)
}

func TestComputeLinkHash_FieldSensitivity(t *testing.T) {
	a := &types.SubstrateLink{AdapterId: "wiki-v1", CitedFacts: []*types.FactCitation{{FactId: "fact-1"}}}
	b := &types.SubstrateLink{AdapterId: "wiki-v1", CitedFacts: []*types.FactCitation{{FactId: "fact-2"}}}
	require.NotEqual(t, keeper.ComputeLinkHash(a), keeper.ComputeLinkHash(b))
}

func TestValidateLink_AdapterMustExist(t *testing.T) {
	k, ctx := setupSubstrateBridgeKeeper(t)
	link := &types.SubstrateLink{AdapterId: "unregistered"}
	err := k.ValidateLink(ctx, link, types.DefaultParams())
	require.ErrorIs(t, err, types.ErrAdapterNotFound)
}

func TestValidateLink_RequiresParams(t *testing.T) {
	k, ctx := setupSubstrateBridgeKeeper(t)
	err := k.ValidateLink(ctx, &types.SubstrateLink{AdapterId: "unregistered"}, nil)
	require.EqualError(t, err, "params required")
}

func TestValidateLink_AdapterMustBeActive(t *testing.T) {
	k, ctx := setupSubstrateBridgeKeeper(t)
	require.NoError(t, k.WriteAdapter(ctx, &types.AdapterRegistration{
		AdapterId: "wiki-v1",
		Status:    types.AdapterStatus_ADAPTER_STATUS_SUSPENDED,
	}))
	err := k.ValidateLink(ctx, &types.SubstrateLink{AdapterId: "wiki-v1"}, types.DefaultParams())
	require.ErrorIs(t, err, types.ErrAdapterNotActive)
}

func TestValidateLink_TooManyPendingClaims(t *testing.T) {
	k, ctx := setupSubstrateBridgeKeeper(t)
	require.NoError(t, k.WriteAdapter(ctx, &types.AdapterRegistration{
		AdapterId: "wiki-v1", Status: types.AdapterStatus_ADAPTER_STATUS_ACTIVE,
	}))
	p := types.DefaultParams()
	p.MaxPendingClaimsPerAttestation = 2
	link := &types.SubstrateLink{
		AdapterId: "wiki-v1",
		PendingClaims: []*types.PendingClaim{
			{ClaimContent: "a"}, {ClaimContent: "b"}, {ClaimContent: "c"},
		},
	}
	require.ErrorIs(t, k.ValidateLink(ctx, link, p), types.ErrTooManyPendingClaims)
}

func TestValidateLink_SourceRequired(t *testing.T) {
	k, ctx := setupSubstrateBridgeKeeper(t)
	require.NoError(t, k.WriteAdapter(ctx, &types.AdapterRegistration{
		AdapterId: "wiki-v1", Status: types.AdapterStatus_ADAPTER_STATUS_ACTIVE,
	}))

	require.ErrorIs(
		t,
		k.ValidateLink(ctx, &types.SubstrateLink{AdapterId: "wiki-v1"}, types.DefaultParams()),
		types.ErrSourceRequired,
	)
}

func TestValidateLink_SourceAdapterMustMatchLink(t *testing.T) {
	k, ctx := setupSubstrateBridgeKeeper(t)
	require.NoError(t, k.WriteAdapter(ctx, &types.AdapterRegistration{
		AdapterId: "wiki-v1", Status: types.AdapterStatus_ADAPTER_STATUS_ACTIVE,
	}))
	link := &types.SubstrateLink{
		AdapterId: "wiki-v1",
		Source:    validExternalSource("other-adapter"),
	}

	require.ErrorIs(t, k.ValidateLink(ctx, link, types.DefaultParams()), types.ErrSourceAdapterIdMismatch)
}

func TestValidateLink_SourceContentHashMustHaveSHA256Width(t *testing.T) {
	k, ctx := setupSubstrateBridgeKeeper(t)
	require.NoError(t, k.WriteAdapter(ctx, &types.AdapterRegistration{
		AdapterId: "wiki-v1", Status: types.AdapterStatus_ADAPTER_STATUS_ACTIVE,
	}))

	for _, length := range []int{0, sha256.Size - 1, sha256.Size, sha256.Size + 1} {
		t.Run(fmt.Sprintf("%d_bytes", length), func(t *testing.T) {
			link := &types.SubstrateLink{
				AdapterId: "wiki-v1",
				Source:    validExternalSource("wiki-v1"),
			}
			link.Source.ContentHash = make([]byte, length)

			err := k.ValidateLink(ctx, link, types.DefaultParams())
			if length == sha256.Size {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, types.ErrInvalidSourceContentHash)
		})
	}
}

func TestValidateLink_AxisOverflow(t *testing.T) {
	k, ctx := setupSubstrateBridgeKeeper(t)
	require.NoError(t, k.WriteAdapter(ctx, &types.AdapterRegistration{
		AdapterId:  "wiki-v1",
		Status:     types.AdapterStatus_ADAPTER_STATUS_ACTIVE,
		AxisBounds: &types.AxisBounds{AxisSubstrateMax: 100},
	}))
	link := &types.SubstrateLink{
		AdapterId:       "wiki-v1",
		RecursionWeight: &types.AxisProjection{AxisSubstrate: 200},
	}
	require.ErrorIs(t, k.ValidateLink(ctx, link, types.DefaultParams()), types.ErrAxisOverflow)
}

// An adapter that declares no AxisBounds used to be the most permissive on
// chain instead of the most restrictive: the ceiling check was skipped entirely,
// and recursion weight multiplies the settlement reward, so six caller-supplied
// uint64s could claim the remaining supply cap in one message. mainnet's only
// live adapter was seeded exactly that way.
func TestValidateLink_WeightAgainstUnboundedAdapterIsRefused(t *testing.T) {
	k, ctx := setupSubstrateBridgeKeeper(t)
	require.NoError(t, k.WriteAdapter(ctx, &types.AdapterRegistration{
		AdapterId: "unbounded-v1",
		Status:    types.AdapterStatus_ADAPTER_STATUS_ACTIVE,
		// AxisBounds deliberately nil — the genesis shape of agenttool-invocation-v1.
	}))
	link := &types.SubstrateLink{
		AdapterId: "unbounded-v1",
		RecursionWeight: &types.AxisProjection{
			AxisSubstrate:      ^uint64(0),
			AxisVerification:   ^uint64(0),
			AxisClassification: ^uint64(0),
			AxisAttribution:    ^uint64(0),
			AxisTooling:        ^uint64(0),
			AxisInterface:      ^uint64(0),
		},
	}
	require.ErrorIs(t, k.ValidateLink(ctx, link, defaultSubstrateBridgeParams()), types.ErrAdapterAxisBoundsUnset)
}

// The other half of the same gate, and the one that matters operationally: the
// live agenttool relay's buildLink sets only `source` and never RecursionWeight.
// Refusing unweighted links against a bounds-less adapter would have stalled the
// bridge instead of protecting it, so this pins that they still pass.
func TestValidateLink_UnweightedLinkStillPassesUnboundedAdapter(t *testing.T) {
	k, ctx := setupSubstrateBridgeKeeper(t)
	require.NoError(t, k.WriteAdapter(ctx, &types.AdapterRegistration{
		AdapterId: "unbounded-v1",
		Status:    types.AdapterStatus_ADAPTER_STATUS_ACTIVE,
	}))
	link := &types.SubstrateLink{
		AdapterId: "unbounded-v1",
		Source:    validExternalSource("unbounded-v1"),
	} // no RecursionWeight
	require.NoError(t, k.ValidateLink(ctx, link, defaultSubstrateBridgeParams()))
}

// An explicit all-zero ceiling is a real answer, not a missing one: it means
// "this adapter accepts no weighted claim", which is what the agenttool-seam-v1
// migration writes. Zero weight is still admissible against it.
func TestValidateLink_ExplicitZeroBoundsRefuseWeightButAllowNone(t *testing.T) {
	k, ctx := setupSubstrateBridgeKeeper(t)
	require.NoError(t, k.WriteAdapter(ctx, &types.AdapterRegistration{
		AdapterId:  "declared-v1",
		Status:     types.AdapterStatus_ADAPTER_STATUS_ACTIVE,
		AxisBounds: &types.AxisBounds{},
	}))
	over := &types.SubstrateLink{
		AdapterId:       "declared-v1",
		RecursionWeight: &types.AxisProjection{AxisSubstrate: 1},
	}
	require.ErrorIs(t, k.ValidateLink(ctx, over, defaultSubstrateBridgeParams()), types.ErrAxisOverflow)

	zero := &types.SubstrateLink{
		AdapterId:       "declared-v1",
		RecursionWeight: &types.AxisProjection{},
		Source:          validExternalSource("declared-v1"),
	}
	require.NoError(t, k.ValidateLink(ctx, zero, defaultSubstrateBridgeParams()))
}
