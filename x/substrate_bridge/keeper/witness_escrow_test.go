package keeper_test

import (
	"math/big"
	"testing"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/zerone-chain/zerone/x/substrate_bridge/keeper"
	"github.com/zerone-chain/zerone/x/substrate_bridge/types"
)

// witnessEscrowFixture: a reward-bearing adapter and a READY witness-only
// attestation, with a short challenge window for sweep tests.
func witnessEscrowFixture(t *testing.T) (keeper.Keeper, sdk.Context, *stubBankKeeper, *stubVestingKeeper, string) {
	t.Helper()
	k, ctx, bk, vk := setupSubstrateBridgeKeeperFull(t)

	params := types.DefaultParams()
	params.WitnessRewardChallengeWindowBlocks = 10
	require.NoError(t, k.SetParams(ctx, params))

	require.NoError(t, k.WriteAdapter(ctx, &types.AdapterRegistration{
		AdapterId:         "agenttool-invocation-v1",
		Status:            types.AdapterStatus_ADAPTER_STATUS_ACTIVE,
		WitnessRewardUzrn: "222000",
	}))
	submitter := testSubmitter("agent")
	require.NoError(t, k.WriteAttestation(ctx, &types.ExternalAttestation{
		AttestationId: "att-w1", AdapterId: "agenttool-invocation-v1", Submitter: submitter,
		BondUzrn: "1000000",
		Status:   types.AttestationStatus_ATTESTATION_STATUS_READY,
		Link:     &types.SubstrateLink{},
	}))
	return k, ctx, bk, vk, submitter
}

// TestWitnessReward_EscrowedAtSettle pins the survival gate: settlement of a
// witness-only attestation through a reward-bearing adapter returns the bond
// but mints NOTHING — the reward is escrowed under the challenge window.
func TestWitnessReward_EscrowedAtSettle(t *testing.T) {
	k, ctx, bk, vk, submitter := witnessEscrowFixture(t)

	require.NoError(t, k.SettleAttestation(ctx, "att-w1"))

	settled, _ := k.GetAttestation(ctx, "att-w1")
	require.Equal(t, types.AttestationStatus_ATTESTATION_STATUS_SETTLED, settled.Status)
	require.Equal(t, "0", settled.RewardUzrn)                       // nothing paid at settle
	require.Equal(t, "1000000", bk.payments[submitter].String())    // bond back whole
	require.Nil(t, vk.minted[types.AuditBountyPoolModuleName])      // nothing minted

	pr, found := k.GetWitnessPendingReward(ctx, "att-w1")
	require.True(t, found)
	require.Equal(t, "222000", pr.Amount)
	require.Equal(t, submitter, pr.Recipient)
	require.Equal(t, uint64(ctx.BlockHeight())+10, pr.Deadline)
}

// TestWitnessReward_ReleasedAfterWindow pins survival issuance: once the
// window closes with the adapter still ACTIVE, the sweep mints (cap-gated)
// and pays exactly once, recording the paid amount on the attestation.
func TestWitnessReward_ReleasedAfterWindow(t *testing.T) {
	k, ctx, bk, vk, submitter := witnessEscrowFixture(t)
	require.NoError(t, k.SettleAttestation(ctx, "att-w1"))

	// Before the deadline: sweep is a no-op.
	k.SweepWitnessRewards(ctx.WithBlockHeight(ctx.BlockHeight() + 5))
	require.Nil(t, vk.minted[types.AuditBountyPoolModuleName])

	// Past the deadline: released.
	late := ctx.WithBlockHeight(ctx.BlockHeight() + 11)
	k.SweepWitnessRewards(late)

	require.Equal(t, big.NewInt(222000), vk.minted[types.AuditBountyPoolModuleName])
	require.Equal(t, "1222000", bk.payments[submitter].String()) // bond + reward
	settled, _ := k.GetAttestation(ctx, "att-w1")
	require.Equal(t, "222000", settled.RewardUzrn)
	_, found := k.GetWitnessPendingReward(ctx, "att-w1")
	require.False(t, found)

	// Exactly-once: a second sweep pays nothing more.
	k.SweepWitnessRewards(late)
	require.Equal(t, big.NewInt(222000), vk.minted[types.AuditBountyPoolModuleName])
	require.Equal(t, "1222000", bk.payments[submitter].String())
}

// TestWitnessReward_CancelledOnTombstone pins the falsification clawback:
// tombstoning the adapter inside the window cancels the pending reward
// eagerly, and nothing ever mints — a free clawback.
func TestWitnessReward_CancelledOnTombstone(t *testing.T) {
	k, ctx, bk, vk, submitter := witnessEscrowFixture(t)
	require.NoError(t, k.SettleAttestation(ctx, "att-w1"))

	require.NoError(t, k.TombstoneAdapter(ctx, "agenttool-invocation-v1"))

	_, found := k.GetWitnessPendingReward(ctx, "att-w1")
	require.False(t, found) // cancelled eagerly at tombstone

	k.SweepWitnessRewards(ctx.WithBlockHeight(ctx.BlockHeight() + 11))
	require.Nil(t, vk.minted[types.AuditBountyPoolModuleName])
	require.Equal(t, "1000000", bk.payments[submitter].String()) // only the bond, ever
}

// TestWitnessReward_DeferredWhileSuspended pins the two-stage falsification:
// SUSPENDED at the deadline defers the escrow one window (investigation
// state); reinstatement then releases it, tombstoning would cancel.
func TestWitnessReward_DeferredWhileSuspended(t *testing.T) {
	k, ctx, bk, vk, submitter := witnessEscrowFixture(t)
	require.NoError(t, k.SettleAttestation(ctx, "att-w1"))
	require.NoError(t, k.SuspendAdapter(ctx, "agenttool-invocation-v1", "under investigation"))

	afterFirst := ctx.WithBlockHeight(ctx.BlockHeight() + 11)
	k.SweepWitnessRewards(afterFirst)

	pr, found := k.GetWitnessPendingReward(ctx, "att-w1")
	require.True(t, found) // deferred, not decided
	require.Equal(t, uint64(afterFirst.BlockHeight())+10, pr.Deadline)
	require.Nil(t, vk.minted[types.AuditBountyPoolModuleName])

	// Reinstate and let the deferred window close: released.
	adapter, _ := k.GetAdapter(ctx, "agenttool-invocation-v1")
	adapter.Status = types.AdapterStatus_ADAPTER_STATUS_ACTIVE
	require.NoError(t, k.WriteAdapter(ctx, adapter))

	k.SweepWitnessRewards(ctx.WithBlockHeight(afterFirst.BlockHeight() + 11))
	require.Equal(t, big.NewInt(222000), vk.minted[types.AuditBountyPoolModuleName])
	require.Equal(t, "1222000", bk.payments[submitter].String())
}

// TestWitnessReward_CapClippedAtRelease pins cap-clip honesty at release
// time: the attestation records what was actually minted and paid.
func TestWitnessReward_CapClippedAtRelease(t *testing.T) {
	k, ctx, bk, vk, submitter := witnessEscrowFixture(t)
	require.NoError(t, k.SettleAttestation(ctx, "att-w1"))

	vk.capRemaining = big.NewInt(1000) // cap nearly exhausted before release
	k.SweepWitnessRewards(ctx.WithBlockHeight(ctx.BlockHeight() + 11))

	settled, _ := k.GetAttestation(ctx, "att-w1")
	require.Equal(t, "1000", settled.RewardUzrn)                 // clipped, honestly recorded
	require.Equal(t, "1001000", bk.payments[submitter].String()) // bond + clipped reward
}

// TestWitnessReward_NoRewardAdapterNoEscrow pins backward compatibility: an
// adapter without witness_reward_uzrn settles exactly as before — bond back,
// nothing minted, nothing escrowed.
func TestWitnessReward_NoRewardAdapterNoEscrow(t *testing.T) {
	k, ctx, _, vk := setupSubstrateBridgeKeeperFull(t)
	require.NoError(t, k.WriteAdapter(ctx, &types.AdapterRegistration{
		AdapterId: "plain-witness", Status: types.AdapterStatus_ADAPTER_STATUS_ACTIVE,
	}))
	require.NoError(t, k.WriteAttestation(ctx, &types.ExternalAttestation{
		AttestationId: "att-plain", AdapterId: "plain-witness", Submitter: testSubmitter("nobody"),
		BondUzrn: "1000000",
		Status:   types.AttestationStatus_ATTESTATION_STATUS_READY,
		Link:     &types.SubstrateLink{},
	}))

	require.NoError(t, k.SettleAttestation(ctx, "att-plain"))

	_, found := k.GetWitnessPendingReward(ctx, "att-plain")
	require.False(t, found)
	require.Nil(t, vk.minted[types.AuditBountyPoolModuleName])
}

// TestWitnessReward_SweepInBeginBlocker pins the wiring: the module
// BeginBlocker itself releases due escrows.
func TestWitnessReward_SweepInBeginBlocker(t *testing.T) {
	k, ctx, bk, _, submitter := witnessEscrowFixture(t)
	require.NoError(t, k.SettleAttestation(ctx, "att-w1"))

	require.NoError(t, k.BeginBlocker(ctx.WithBlockHeight(ctx.BlockHeight()+11)))
	require.Equal(t, sdkmath.NewInt(1222000).String(), bk.payments[submitter].String())
}
