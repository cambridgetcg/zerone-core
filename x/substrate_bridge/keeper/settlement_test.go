package keeper_test

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	sdkmath "cosmossdk.io/math"

	"github.com/zerone-chain/zerone/x/substrate_bridge/types"
)

func TestSettleAttestation_FullVerified(t *testing.T) {
	k, ctx, bk, vk := setupSubstrateBridgeKeeperFull(t)
	require.NoError(t, k.WriteAdapter(ctx, &types.AdapterRegistration{
		AdapterId: "wiki-v1", Status: types.AdapterStatus_ADAPTER_STATUS_ACTIVE,
		AxisBounds: &types.AxisBounds{AxisSubstrateMax: 1_000_000},
	}))
	submitter := testSubmitter("alice")
	att := &types.ExternalAttestation{
		AttestationId: "att-1", AdapterId: "wiki-v1", Submitter: submitter,
		BondUzrn: "1000000",
		Status:   types.AttestationStatus_ATTESTATION_STATUS_READY,
		Link: &types.SubstrateLink{
			RecursionWeight: &types.AxisProjection{AxisSubstrate: 500_000},
			PendingClaims:   []*types.PendingClaim{{ClaimContent: "a"}, {ClaimContent: "b"}},
		},
		VerifiedCount: 2, RejectedCount: 0,
	}
	require.NoError(t, k.WriteAttestation(ctx, att))

	require.NoError(t, k.SettleAttestation(ctx, "att-1"))

	settled, _ := k.GetAttestation(ctx, "att-1")
	require.Equal(t, types.AttestationStatus_ATTESTATION_STATUS_SETTLED, settled.Status)

	// Reward was minted through the cap (never paid from escrow) and
	// recorded as actually paid.
	reward, _ := sdkmath.NewIntFromString(settled.RewardUzrn)
	require.True(t, reward.GT(sdkmath.ZeroInt()))
	require.NotNil(t, vk.minted[types.AuditBountyPoolModuleName])
	require.Equal(t, reward.BigInt().String(), vk.minted[types.AuditBountyPoolModuleName].String())

	// Submitter received bond back + reward.
	bond, _ := sdkmath.NewIntFromString(att.BondUzrn)
	require.Equal(t, bond.Add(reward).String(), bk.payments[submitter].String())
}

func TestSettleAttestation_PartialVerified(t *testing.T) {
	k, ctx, bk, _ := setupSubstrateBridgeKeeperFull(t)
	require.NoError(t, k.WriteAdapter(ctx, &types.AdapterRegistration{
		AdapterId: "wiki-v1", Status: types.AdapterStatus_ADAPTER_STATUS_ACTIVE,
	}))
	// 1 of 4 pending claims verified, 3 rejected (75% rejection).
	// But threshold is 50% — this should trigger REJECTED path.
	att := &types.ExternalAttestation{
		AttestationId: "att-r", AdapterId: "wiki-v1", Submitter: testSubmitter("bob"),
		BondUzrn: "1000000",
		Status:   types.AttestationStatus_ATTESTATION_STATUS_READY,
		Link: &types.SubstrateLink{
			PendingClaims: []*types.PendingClaim{{}, {}, {}, {}},
		},
		VerifiedCount: 1, RejectedCount: 3,
	}
	require.NoError(t, k.WriteAttestation(ctx, att))

	require.NoError(t, k.SettleAttestation(ctx, "att-r"))

	settled, _ := k.GetAttestation(ctx, "att-r")
	require.Equal(t, types.AttestationStatus_ATTESTATION_STATUS_REJECTED, settled.Status)
	// Slashed bond is burned, not kept in escrow: dishonesty frees cap
	// headroom instead of accumulating as module dead weight.
	require.Equal(t, att.BondUzrn, settled.SlashUzrn)
	require.Equal(t, "1000000", bk.burned.String())
}

func TestSettleAttestation_PartialButAboveThreshold(t *testing.T) {
	k, ctx, bk, _ := setupSubstrateBridgeKeeperFull(t)
	require.NoError(t, k.WriteAdapter(ctx, &types.AdapterRegistration{
		AdapterId: "wiki-v1", Status: types.AdapterStatus_ADAPTER_STATUS_ACTIVE,
	}))
	submitter := testSubmitter("carol")
	// 3 verified, 1 rejected (25% rejection, 75% verified) → PARTIAL settle.
	att := &types.ExternalAttestation{
		AttestationId: "att-p", AdapterId: "wiki-v1", Submitter: submitter,
		BondUzrn: "2000000",
		Status:   types.AttestationStatus_ATTESTATION_STATUS_READY,
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
	// PARTIAL still returns the full bond — it was collateral, not payment.
	paid := bk.payments[submitter]
	require.True(t, paid.GTE(sdkmath.NewInt(2000000)))
}

// TestSettleAttestation_WitnessOnlyReturnsBond pins the un-stranding fix:
// a link with no cited facts and no pending claims earns nothing, but its
// bond comes back whole.
func TestSettleAttestation_WitnessOnlyReturnsBond(t *testing.T) {
	k, ctx, bk, vk := setupSubstrateBridgeKeeperFull(t)
	require.NoError(t, k.WriteAdapter(ctx, &types.AdapterRegistration{
		AdapterId: "agenttool-invocation-v1", Status: types.AdapterStatus_ADAPTER_STATUS_ACTIVE,
	}))
	submitter := testSubmitter("witness")
	att := &types.ExternalAttestation{
		AttestationId: "att-w", AdapterId: "agenttool-invocation-v1", Submitter: submitter,
		BondUzrn: "1000000",
		Status:   types.AttestationStatus_ATTESTATION_STATUS_READY,
		Link:     &types.SubstrateLink{},
	}
	require.NoError(t, k.WriteAttestation(ctx, att))

	require.NoError(t, k.SettleAttestation(ctx, "att-w"))

	settled, _ := k.GetAttestation(ctx, "att-w")
	require.Equal(t, types.AttestationStatus_ATTESTATION_STATUS_SETTLED, settled.Status)
	require.Equal(t, "0", settled.RewardUzrn)
	require.Equal(t, "1000000", bk.payments[submitter].String())
	require.Nil(t, vk.minted[types.AuditBountyPoolModuleName]) // nothing minted
}

// TestSettleAttestation_CapClippedReward pins cap-clip honesty: when the
// supply cap can only cover part of the computed reward, the attestation
// records what was actually minted and paid, not the formula value.
func TestSettleAttestation_CapClippedReward(t *testing.T) {
	k, ctx, bk, vk := setupSubstrateBridgeKeeperFull(t)
	vk.capRemaining = big.NewInt(1000) // almost exhausted cap
	require.NoError(t, k.WriteAdapter(ctx, &types.AdapterRegistration{
		AdapterId: "wiki-v1", Status: types.AdapterStatus_ADAPTER_STATUS_ACTIVE,
		AxisBounds: &types.AxisBounds{AxisSubstrateMax: 1_000_000},
	}))
	submitter := testSubmitter("dave")
	att := &types.ExternalAttestation{
		AttestationId: "att-c", AdapterId: "wiki-v1", Submitter: submitter,
		BondUzrn: "1000000",
		Status:   types.AttestationStatus_ATTESTATION_STATUS_READY,
		Link: &types.SubstrateLink{
			RecursionWeight: &types.AxisProjection{AxisSubstrate: 500_000},
			PendingClaims:   []*types.PendingClaim{{ClaimContent: "a"}},
		},
		VerifiedCount: 1,
	}
	require.NoError(t, k.WriteAttestation(ctx, att))

	require.NoError(t, k.SettleAttestation(ctx, "att-c"))

	settled, _ := k.GetAttestation(ctx, "att-c")
	require.Equal(t, "1000", settled.RewardUzrn) // clipped, honestly recorded
	// bond (1000000) + clipped reward (1000)
	require.Equal(t, "1001000", bk.payments[submitter].String())
}
