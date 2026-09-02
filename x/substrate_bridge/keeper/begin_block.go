package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/x/substrate_bridge/types"
)

// BeginBlocker scans AWAITING_RESOLUTION attestations and:
//   - times out any older than Params.MaxPendingWindowBlocks (settle-as-PARTIAL or REJECTED)
//   - drains READY attestations into SETTLED (called from settlement engine,
//     but BeginBlocker also pulls them in case OnClaimResolved didn't fire
//     for the last pending claim).
func (k Keeper) BeginBlocker(ctx context.Context) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Fail closed while enforcement is unarmed. Every mint path — settlement
	// (SettleAttestation) and witness-reward release (SweepWitnessRewards) —
	// is reached only from here, and each authorizes a mint by consulting the
	// 0x8E source index. On a plan-less restart the substrate-dedupe-v1 handler
	// has not run, so that index is empty and unarmed: a pre-existing in-flight
	// attestation whose already-settled twin minted before the index existed
	// would find its source "free" and mint a second time. Deferring settlement
	// (rather than zeroing rewards) leaves the attestations in flight, losing
	// nothing; the handler seeds the index + arms, and the very next block
	// settles them correctly — the seeded holder suppresses the duplicate. A
	// fresh chain is always armed before anything can settle, because the only
	// creator of attestations, SubmitExternalAttestation, arms on its first call.
	if !k.IsDedupeArmed(ctx) {
		return nil
	}

	currentHeight := uint64(sdkCtx.BlockHeight())
	params := k.GetParams(ctx)

	// Timeout scan.
	var timedOut []string
	k.IterateAttestationsByStatus(ctx, types.AttestationStatus_ATTESTATION_STATUS_AWAITING_RESOLUTION, func(id string) bool {
		att, found := k.GetAttestation(ctx, id)
		if !found {
			return false
		}
		if currentHeight-att.SubmittedAtBlock >= params.MaxPendingWindowBlocks {
			timedOut = append(timedOut, id)
		}
		return false
	})
	for _, id := range timedOut {
		if err := k.SettleAttestation(ctx, id); err != nil {
			k.Logger(sdkCtx).Error("timeout-settle failed", "attestation_id", id, "err", err)
		}
	}

	// READY drain.
	var ready []string
	k.IterateAttestationsByStatus(ctx, types.AttestationStatus_ATTESTATION_STATUS_READY, func(id string) bool {
		ready = append(ready, id)
		return false
	})
	for _, id := range ready {
		if err := k.SettleAttestation(ctx, id); err != nil {
			k.Logger(sdkCtx).Error("ready-settle failed", "attestation_id", id, "err", err)
		}
	}

	// Witness-reward escrows whose challenge window closed.
	k.SweepWitnessRewards(ctx)

	return nil
}
