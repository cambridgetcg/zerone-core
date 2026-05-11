package knowledgeclaim

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	contribkeeper "github.com/zerone-chain/zerone/x/contribution/keeper"
	contribtypes "github.com/zerone-chain/zerone/x/contribution/types"
	knowledgetypes "github.com/zerone-chain/zerone/x/knowledge/types"
)

// KnowledgeHooksAdapter implements knowledgetypes.KnowledgeHooks.
// It mirrors claim lifecycle into Contribution lifecycle.
type KnowledgeHooksAdapter struct {
	contribKeeper *contribkeeper.Keeper
	adapter       Adapter
}

// NewKnowledgeHooksAdapter constructs the hooks adapter.
func NewKnowledgeHooksAdapter(ck *contribkeeper.Keeper, a Adapter) KnowledgeHooksAdapter {
	return KnowledgeHooksAdapter{contribKeeper: ck, adapter: a}
}

var _ knowledgetypes.KnowledgeHooks = KnowledgeHooksAdapter{}

// AfterClaimSubmitted constructs the Contribution mirror in
// STATUS_SUBMITTED, runs Classify + SubstrateLink, transitions to
// STATUS_CLASSIFIED on success or STATUS_CLASSIFICATION_FAILED on error.
//
// All status writes go through TransitionStatus to enforce the
// forward-only audit invariant (truth-seeking commitment 10) — the
// snapshot builder leaves the Contribution at SUBMITTED, so every
// write here is a SUBMITTED → {CLASSIFIED, CLASSIFICATION_FAILED}
// transition, both of which are valid per status.go.
func (h KnowledgeHooksAdapter) AfterClaimSubmitted(ctx context.Context, claimID string, snap knowledgetypes.ClaimSnapshot) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	c := BuildContributionFromSnapshot(claimID, snap, sdkCtx.BlockHeight())

	// Stage ② — Classify.
	if err := h.adapter.Classify(ctx, c); err != nil {
		if tErr := h.contribKeeper.TransitionStatus(ctx, c, contribtypes.ContributionStatus_STATUS_CLASSIFICATION_FAILED); tErr != nil {
			return tErr
		}
		h.contribKeeper.EmitClassificationFailed(ctx, c.Id, err.Error())
		return nil
	}
	linkBps, err := h.adapter.SubstrateLink(ctx, c)
	if err != nil {
		if tErr := h.contribKeeper.TransitionStatus(ctx, c, contribtypes.ContributionStatus_STATUS_CLASSIFICATION_FAILED); tErr != nil {
			return tErr
		}
		h.contribKeeper.EmitClassificationFailed(ctx, c.Id, err.Error())
		return nil
	}
	c.SubstrateLinkBps = linkBps
	if err := h.contribKeeper.TransitionStatus(ctx, c, contribtypes.ContributionStatus_STATUS_CLASSIFIED); err != nil {
		return err
	}
	h.contribKeeper.EmitContributionSubmitted(ctx, c)
	h.contribKeeper.EmitContributionClassified(ctx, c)
	return nil
}

// AfterClaimVerificationFinalized sets the verification_score and
// transitions to STATUS_VERIFIED or STATUS_VERIFICATION_FAILED.
// Emits useful_work_attested + useful_work_settled + recursion_weight_computed
// only on the success path (mirrors msg_server.go's Verify gating);
// on failure emits verification_failed.
//
// Late hook firing on a Contribution already in a terminal state
// (e.g., CLASSIFICATION_FAILED) is a NO-OP — the forward-only
// invariant must not be violated by lifecycle hooks racing against
// earlier failures.
func (h KnowledgeHooksAdapter) AfterClaimVerificationFinalized(ctx context.Context, claimID string, scoreBps uint32) error {
	c, found := h.contribKeeper.GetContributionByBackRef(ctx, claimID)
	if !found {
		return nil // mirror absent — claim wasn't submitted under our hooks
	}
	if contribtypes.IsTerminal(c.Status) {
		return nil // already terminal; ignore late lifecycle hook
	}
	c.VerificationScoreBps = scoreBps
	if scoreBps >= contribtypes.MinVerificationScoreBps {
		// success path
		if err := h.contribKeeper.TransitionStatus(ctx, c, contribtypes.ContributionStatus_STATUS_VERIFIED); err != nil {
			return err
		}
		h.contribKeeper.EmitUsefulWorkAttested(ctx, c)
		h.contribKeeper.EmitUsefulWorkSettled(ctx, c)       // shape-only at Phase 1
		h.contribKeeper.EmitRecursionWeightComputed(ctx, c) // all-zero at Phase 1
		return nil
	}
	// failure path
	if err := h.contribKeeper.TransitionStatus(ctx, c, contribtypes.ContributionStatus_STATUS_VERIFICATION_FAILED); err != nil {
		return err
	}
	h.contribKeeper.EmitVerificationFailed(ctx, c, "verification score below threshold")
	return nil
}

// AfterClaimAccepted transitions to STATUS_ADMITTED.
//
// Option C for back_ref consistency: BackRef is left as claim_id
// throughout the Contribution lifecycle so GetContributionByBackRef(claimID)
// continues to resolve at admission, disprove, and any later lookup.
// The fact_id is carried on the emitted contribution_admitted event
// as an attribute (so off-chain consumers can join the two), and
// remains recoverable on-chain via x/knowledge's claim→fact link if
// callers need it later.
//
// Late hook firing on a Contribution already in a terminal state
// (e.g., VERIFICATION_FAILED) is a NO-OP.
func (h KnowledgeHooksAdapter) AfterClaimAccepted(ctx context.Context, claimID string, factID string) error {
	c, found := h.contribKeeper.GetContributionByBackRef(ctx, claimID)
	if !found {
		return nil
	}
	if contribtypes.IsTerminal(c.Status) {
		return nil // already terminal; ignore late lifecycle hook
	}
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	c.AdmittedAtBlock = uint64(sdkCtx.BlockHeight())
	if err := h.contribKeeper.TransitionStatus(ctx, c, contribtypes.ContributionStatus_STATUS_ADMITTED); err != nil {
		return err
	}
	h.contribKeeper.EmitContributionAdmitted(ctx, c, factID)
	return nil
}

// AfterClaimDisproven transitions to STATUS_REVOKED.
//
// Resolves the Contribution by claim_id (BackRef is the original
// claim_id throughout the lifecycle — see AfterClaimAccepted note).
func (h KnowledgeHooksAdapter) AfterClaimDisproven(ctx context.Context, claimID string, disproverArtifactID string) error {
	c, found := h.contribKeeper.GetContributionByBackRef(ctx, claimID)
	if !found {
		return nil
	}
	if err := h.contribKeeper.TransitionStatus(ctx, c, contribtypes.ContributionStatus_STATUS_REVOKED); err != nil {
		return err
	}
	h.contribKeeper.EmitContributionRevoked(ctx, c, disproverArtifactID)
	return nil
}
