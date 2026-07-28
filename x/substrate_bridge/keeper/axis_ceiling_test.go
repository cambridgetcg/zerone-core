package keeper_test

import (
	"math"
	"testing"

	sdkmath "cosmossdk.io/math"
	"github.com/stretchr/testify/require"

	"github.com/zerone-chain/zerone/x/substrate_bridge/types"
)

// The protocol axis ceiling. computeReward sums six caller-supplied uint64
// axis projections straight into a mint. The bound on those projections used
// to be applied only when the adapter declared AxisBounds — and mainnet's
// only registered adapter (agenttool-invocation-v1) ships axis_bounds:null,
// so on the one adapter that exists the sum was unbounded.
//
// These tests pin BOTH halves of the fix: rejection at entry for new links,
// and clamping at settlement for records already in the store.

// An adapter cannot widen the protocol ceiling even if its own declared
// bounds are maximally permissive.
func TestValidateLink_ProtocolCeilingOverridesWiderAdapterBounds(t *testing.T) {
	k, ctx := setupSubstrateBridgeKeeper(t)
	require.NoError(t, k.WriteAdapter(ctx, &types.AdapterRegistration{
		AdapterId: "wide-bounds-v1",
		Status:    types.AdapterStatus_ADAPTER_STATUS_ACTIVE,
		AxisBounds: &types.AxisBounds{
			AxisSubstrateMax:      math.MaxUint64,
			AxisVerificationMax:   math.MaxUint64,
			AxisClassificationMax: math.MaxUint64,
			AxisAttributionMax:    math.MaxUint64,
			AxisToolingMax:        math.MaxUint64,
			AxisInterfaceMax:      math.MaxUint64,
		},
	}))

	link := &types.SubstrateLink{
		AdapterId: "wide-bounds-v1",
		// The drain: six max-uint64 projections.
		RecursionWeight: &types.AxisProjection{
			AxisSubstrate:      math.MaxUint64,
			AxisVerification:   math.MaxUint64,
			AxisClassification: math.MaxUint64,
			AxisAttribution:    math.MaxUint64,
			AxisTooling:        math.MaxUint64,
			AxisInterface:      math.MaxUint64,
		},
	}
	require.ErrorIs(t, k.ValidateLink(ctx, link, defaultSubstrateBridgeParams()), types.ErrAxisOverflow,
		"an adapter widened the protocol ceiling and restored the supply drain")
}

// One axis over the ceiling is enough to reject, even with the rest at zero.
func TestValidateLink_SingleAxisOverCeilingRejected(t *testing.T) {
	k, ctx := setupSubstrateBridgeKeeper(t)
	require.NoError(t, k.WriteAdapter(ctx, &types.AdapterRegistration{
		AdapterId: "wide-bounds-v1", Status: types.AdapterStatus_ADAPTER_STATUS_ACTIVE,
		AxisBounds: &types.AxisBounds{AxisInterfaceMax: math.MaxUint64},
	}))

	link := &types.SubstrateLink{
		AdapterId:       "wide-bounds-v1",
		RecursionWeight: &types.AxisProjection{AxisInterface: types.MaxAxisProjectionBps + 1},
	}
	require.ErrorIs(t, k.ValidateLink(ctx, link, defaultSubstrateBridgeParams()), types.ErrAxisOverflow)
}

// Legitimate values at and below the ceiling must still pass when the adapter
// has explicitly declared matching bounds. A nil adapter bound remains
// fail-closed under agenttool-seam-v1.
func TestValidateLink_AtCeilingAcceptedWithDeclaredBounds(t *testing.T) {
	k, ctx := setupSubstrateBridgeKeeper(t)
	require.NoError(t, k.WriteAdapter(ctx, &types.AdapterRegistration{
		AdapterId: "declared-bounds-v1", Status: types.AdapterStatus_ADAPTER_STATUS_ACTIVE,
		AxisBounds: &types.AxisBounds{
			AxisSubstrateMax:    types.MaxAxisProjectionBps,
			AxisVerificationMax: types.MaxAxisProjectionBps,
		},
	}))

	link := &types.SubstrateLink{
		AdapterId: "declared-bounds-v1",
		RecursionWeight: &types.AxisProjection{
			AxisSubstrate:    types.MaxAxisProjectionBps,
			AxisVerification: 500_000,
		},
	}
	require.NoError(t, k.ValidateLink(ctx, link, defaultSubstrateBridgeParams()))
}

// Adapter bounds may only TIGHTEN the protocol ceiling.
func TestValidateLink_AdapterBoundsStillTighten(t *testing.T) {
	k, ctx := setupSubstrateBridgeKeeper(t)
	require.NoError(t, k.WriteAdapter(ctx, &types.AdapterRegistration{
		AdapterId: "tight-v1", Status: types.AdapterStatus_ADAPTER_STATUS_ACTIVE,
		AxisBounds: &types.AxisBounds{AxisSubstrateMax: 100},
	}))

	link := &types.SubstrateLink{
		AdapterId:       "tight-v1",
		RecursionWeight: &types.AxisProjection{AxisSubstrate: 200}, // under protocol, over adapter
	}
	require.ErrorIs(t, k.ValidateLink(ctx, link, defaultSubstrateBridgeParams()), types.ErrAxisOverflow)
}

// Entry validation only guards NEW links. An attestation already in the store
// from before the ceiling existed must still mint a bounded amount, so the
// reward computation clamps as well. Without the clamp this settlement mints
// an astronomically large reward.
func TestSettleAttestation_ClampsPreExistingUnboundedAxes(t *testing.T) {
	k, ctx, _, vk := setupSubstrateBridgeKeeperFull(t)
	require.NoError(t, k.WriteAdapter(ctx, &types.AdapterRegistration{
		AdapterId: "nullbounds-v1", Status: types.AdapterStatus_ADAPTER_STATUS_ACTIVE,
	}))

	att := &types.ExternalAttestation{
		AttestationId: "att-legacy", AdapterId: "nullbounds-v1",
		Submitter: testSubmitter("alice"), BondUzrn: "1000000",
		Status: types.AttestationStatus_ATTESTATION_STATUS_READY,
		Link: &types.SubstrateLink{
			// Written directly to the store, bypassing ValidateLink — this is
			// the shape a record could already have on mainnet.
			RecursionWeight: &types.AxisProjection{
				AxisSubstrate:      math.MaxUint64,
				AxisVerification:   math.MaxUint64,
				AxisClassification: math.MaxUint64,
				AxisAttribution:    math.MaxUint64,
				AxisTooling:        math.MaxUint64,
				AxisInterface:      math.MaxUint64,
			},
			PendingClaims: []*types.PendingClaim{{ClaimContent: "a"}},
		},
		VerifiedCount: 1, RejectedCount: 0,
	}
	require.NoError(t, k.WriteAttestation(ctx, att))
	require.NoError(t, k.SettleAttestation(ctx, "att-legacy"))

	settled, found := k.GetAttestation(ctx, "att-legacy")
	require.True(t, found)
	reward, ok := sdkmath.NewIntFromString(settled.RewardUzrn)
	require.True(t, ok)

	// Ceiling: base × (1 + 6,000,000/20,000) = base × 301. Assert the mint is
	// within that bound rather than asserting an exact figure, so the test
	// keeps pinning "finite" if the base param is ever retuned.
	base, ok := sdkmath.NewIntFromString(types.DefaultParams().AttestationMinBondUzrn)
	require.True(t, ok)
	maxReward := base.Mul(sdkmath.NewInt(301))
	require.True(t, reward.LTE(maxReward),
		"settlement minted %s uzrn, above the clamped ceiling of %s — the clamp is not applied",
		reward, maxReward)

	minted := vk.minted[types.AuditBountyPoolModuleName]
	require.NotNil(t, minted)
	require.True(t, sdkmath.NewIntFromBigInt(minted).LTE(maxReward))
}
