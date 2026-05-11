package keeper

import (
	"context"
	"fmt"
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/x/contribution/types"
)

// TransitionStatus updates a Contribution's status if the transition
// is allowed by the forward-only audit invariant. Persists via
// WriteContribution. Caller is responsible for emitting any
// stage-specific event.
func (k Keeper) TransitionStatus(ctx context.Context, c *types.Contribution, to types.ContributionStatus) error {
	if !types.CanTransition(c.Status, to) {
		return types.ErrInvalidStatusTransition.Wrapf("from %s to %s", c.Status, to)
	}
	c.Status = to
	return k.WriteContribution(ctx, c)
}

// ── event emitters ──

// idHex hex-encodes a contribution id for event attribute display.
func idHex(id []byte) string { return fmt.Sprintf("%x", id) }

func (k Keeper) EmitContributionSubmitted(ctx context.Context, c *types.Contribution) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeContributionSubmitted,
		sdk.NewAttribute(types.AttributeKeyID, idHex(c.Id)),
		sdk.NewAttribute(types.AttributeKeyClass, c.Class.String()),
		sdk.NewAttribute(types.AttributeKeyPhase, c.Phase.String()),
		sdk.NewAttribute(types.AttributeKeyContributor, c.Contributor),
		sdk.NewAttribute(types.AttributeKeyCreedCommitment, types.CommitmentIssuance),
		sdk.NewAttribute(types.AttributeKeyUsefulWorkCommitment, types.UsefulWorkCommitmentValue),
	))
}

func (k Keeper) EmitContributionClassified(ctx context.Context, c *types.Contribution) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeContributionClassified,
		sdk.NewAttribute(types.AttributeKeyID, idHex(c.Id)),
		sdk.NewAttribute(types.AttributeKeySubstrateLinkBps, strconv.FormatUint(uint64(c.SubstrateLinkBps), 10)),
		sdk.NewAttribute(types.AttributeKeyMechanism, types.MechanismM2),
		sdk.NewAttribute(types.AttributeKeyUsefulWorkCommitment, types.UsefulWorkCommitmentValue),
	))
}

func (k Keeper) EmitUsefulWorkAttested(ctx context.Context, c *types.Contribution) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeUsefulWorkAttested,
		sdk.NewAttribute(types.AttributeKeyID, idHex(c.Id)),
		sdk.NewAttribute(types.AttributeKeyClass, c.Class.String()),
		sdk.NewAttribute(types.AttributeKeyVerificationScoreBps, strconv.FormatUint(uint64(c.VerificationScoreBps), 10)),
		sdk.NewAttribute(types.AttributeKeyMechanism, types.MechanismM3),
		sdk.NewAttribute(types.AttributeKeyUsefulWorkCommitment, types.UsefulWorkCommitmentValue),
	))
}

func (k Keeper) EmitUsefulWorkSettled(ctx context.Context, c *types.Contribution) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	// Phase 1: W=0 always (identity scorers); reward shape only.
	sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeUsefulWorkSettled,
		sdk.NewAttribute(types.AttributeKeyID, idHex(c.Id)),
		sdk.NewAttribute(types.AttributeKeyRewardShape, "base+L*W*Q"),
		sdk.NewAttribute(types.AttributeKeyLBps, strconv.FormatUint(uint64(c.SubstrateLinkBps), 10)),
		sdk.NewAttribute(types.AttributeKeyWBps, "0"),
		sdk.NewAttribute(types.AttributeKeyQBps, strconv.FormatUint(uint64(c.VerificationScoreBps), 10)),
		sdk.NewAttribute(types.AttributeKeyMechanism, types.MechanismM4),
		sdk.NewAttribute(types.AttributeKeyUsefulWorkCommitment, types.UsefulWorkCommitmentValue),
	))
}

func (k Keeper) EmitRecursionWeightComputed(ctx context.Context, c *types.Contribution) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	// Phase 1: identity scorers — all axes zero.
	sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeRecursionWeightComputed,
		sdk.NewAttribute(types.AttributeKeyID, idHex(c.Id)),
		sdk.NewAttribute(types.AttributeKeyAxisSubstrate, "0"),
		sdk.NewAttribute(types.AttributeKeyAxisVerification, "0"),
		sdk.NewAttribute(types.AttributeKeyAxisClassification, "0"),
		sdk.NewAttribute(types.AttributeKeyAxisAttribution, "0"),
		sdk.NewAttribute(types.AttributeKeyAxisTooling, "0"),
		sdk.NewAttribute(types.AttributeKeyAxisInterface, "0"),
		sdk.NewAttribute(types.AttributeKeyTotalWeight, "0"),
		sdk.NewAttribute(types.AttributeKeyMechanism, types.MechanismM5),
	))
}

func (k Keeper) EmitContributionAdmitted(ctx context.Context, c *types.Contribution) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeContributionAdmitted,
		sdk.NewAttribute(types.AttributeKeyID, idHex(c.Id)),
		sdk.NewAttribute(types.AttributeKeyClass, c.Class.String()),
		sdk.NewAttribute(types.AttributeKeyPhase, c.Phase.String()),
		sdk.NewAttribute(types.AttributeKeyAdmittedAtBlock, strconv.FormatUint(c.AdmittedAtBlock, 10)),
		sdk.NewAttribute(types.AttributeKeyBackRef, c.BackRef),
		sdk.NewAttribute(types.AttributeKeyCreedCommitment, types.CommitmentIssuance),
		sdk.NewAttribute(types.AttributeKeyUsefulWorkCommitment, types.UsefulWorkCommitmentValue),
	))
}

func (k Keeper) EmitContributionRevoked(ctx context.Context, c *types.Contribution, disproverID string) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeContributionRevoked,
		sdk.NewAttribute(types.AttributeKeyID, idHex(c.Id)),
		sdk.NewAttribute(types.AttributeKeyDisproverArtifactID, disproverID),
		sdk.NewAttribute(types.AttributeKeyCascadeFlag, types.CascadeFlagRevokedAncestor),
	))
}

func (k Keeper) EmitClassificationFailed(ctx context.Context, id []byte, reason string) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeContributionClassificationFailed,
		sdk.NewAttribute(types.AttributeKeyID, idHex(id)),
		sdk.NewAttribute(types.AttributeKeyReason, reason),
	))
}

func (k Keeper) EmitVerificationFailed(ctx context.Context, c *types.Contribution, reason string) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeContributionVerificationFailed,
		sdk.NewAttribute(types.AttributeKeyID, idHex(c.Id)),
		sdk.NewAttribute(types.AttributeKeyReason, reason),
		sdk.NewAttribute(types.AttributeKeyVerificationScoreBps, strconv.FormatUint(uint64(c.VerificationScoreBps), 10)),
	))
}
