package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	sdkmath "cosmossdk.io/math"

	"github.com/zerone-chain/zerone/x/substrate_bridge/types"
)

func TestSettleAttestation_FullVerified(t *testing.T) {
	k, ctx := setupSubstrateBridgeKeeperWithBank(t)
	require.NoError(t, k.WriteAdapter(ctx, &types.AdapterRegistration{
		AdapterId: "wiki-v1", Status: types.AdapterStatus_ADAPTER_STATUS_ACTIVE,
		AxisBounds: &types.AxisBounds{AxisSubstrateMax: 1_000_000},
	}))
	att := &types.ExternalAttestation{
		AttestationId: "att-1", AdapterId: "wiki-v1", Submitter: "alice",
		Status: types.AttestationStatus_ATTESTATION_STATUS_READY,
		Link: &types.SubstrateLink{
			RecursionWeight: &types.AxisProjection{AxisSubstrate: 500_000},
			PendingClaims: []*types.PendingClaim{{ClaimContent: "a"}, {ClaimContent: "b"}},
		},
		VerifiedCount: 2, RejectedCount: 0,
	}
	require.NoError(t, k.WriteAttestation(ctx, att))

	require.NoError(t, k.SettleAttestation(ctx, "att-1"))

	settled, _ := k.GetAttestation(ctx, "att-1")
	require.Equal(t, types.AttestationStatus_ATTESTATION_STATUS_SETTLED, settled.Status)
	reward, _ := sdkmath.NewIntFromString(settled.RewardUzrn)
	require.True(t, reward.GT(sdkmath.ZeroInt()))
}

func TestSettleAttestation_PartialVerified(t *testing.T) {
	k, ctx := setupSubstrateBridgeKeeperWithBank(t)
	require.NoError(t, k.WriteAdapter(ctx, &types.AdapterRegistration{
		AdapterId: "wiki-v1", Status: types.AdapterStatus_ADAPTER_STATUS_ACTIVE,
	}))
	// 1 of 4 pending claims verified, 3 rejected (75% rejection).
	// But threshold is 50% — this should trigger REJECTED path.
	att := &types.ExternalAttestation{
		AttestationId: "att-r", AdapterId: "wiki-v1", Submitter: "bob",
		Status: types.AttestationStatus_ATTESTATION_STATUS_READY,
		Link: &types.SubstrateLink{
			PendingClaims: []*types.PendingClaim{{}, {}, {}, {}},
		},
		VerifiedCount: 1, RejectedCount: 3,
	}
	require.NoError(t, k.WriteAttestation(ctx, att))

	require.NoError(t, k.SettleAttestation(ctx, "att-r"))

	settled, _ := k.GetAttestation(ctx, "att-r")
	require.Equal(t, types.AttestationStatus_ATTESTATION_STATUS_REJECTED, settled.Status)
}

func TestSettleAttestation_PartialButAboveThreshold(t *testing.T) {
	k, ctx := setupSubstrateBridgeKeeperWithBank(t)
	require.NoError(t, k.WriteAdapter(ctx, &types.AdapterRegistration{
		AdapterId: "wiki-v1", Status: types.AdapterStatus_ADAPTER_STATUS_ACTIVE,
	}))
	// 3 verified, 1 rejected (25% rejection, 75% verified) → PARTIAL settle.
	att := &types.ExternalAttestation{
		AttestationId: "att-p", AdapterId: "wiki-v1", Submitter: "carol",
		Status: types.AttestationStatus_ATTESTATION_STATUS_READY,
		Link: &types.SubstrateLink{
			RecursionWeight: &types.AxisProjection{AxisSubstrate: 100_000},
			PendingClaims:   []*types.PendingClaim{{}, {}, {}, {}},
		},
		VerifiedCount: 3, RejectedCount: 1,
	}
	require.NoError(t, k.WriteAttestation(ctx, att))

	require.NoError(t, k.SettleAttestation(ctx, "att-p"))

	settled, _ := k.GetAttestation(ctx, "att-p")
	require.Equal(t, types.AttestationStatus_ATTESTATION_STATUS_PARTIAL, settled.Status)
}
