package keeper

import (
	"context"
	"fmt"
	"math/big"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// A conjecture asks whether a proposition is well-posed and falsifiable; it
// does not assert that the proposition is true. Its review uses the ordinary
// fee schedule. New funding terms distinguish paid review work from an unused
// budget without creating an author reward or changing scientific judgment.

// PostConjecture places an unsettled proposition into the graph.
func (m *msgServer) PostConjecture(ctx context.Context, msg *types.MsgPostConjecture) (*types.MsgPostConjectureResponse, error) {
	if msg == nil {
		return nil, fmt.Errorf("conjecture is required")
	}
	funded, err := m.keeper.FundSettlementEnabled(ctx)
	if err != nil {
		return nil, err
	}
	if !funded {
		return m.postConjecture(ctx, msg, false)
	}
	if len(msg.ProtoReflect().GetUnknown()) != 0 {
		return nil, fmt.Errorf("unsupported conjecture fields")
	}
	cache, commit := sdk.UnwrapSDKContext(ctx).CacheContext()
	response, err := m.postConjecture(cache, msg, true)
	if err != nil {
		return nil, err
	}
	commit()
	return response, nil
}

func (m *msgServer) postConjecture(ctx context.Context, msg *types.MsgPostConjecture, funded bool) (*types.MsgPostConjectureResponse, error) {
	reviewPolicy, err := m.keeper.reviewPolicyAtAdmission(ctx)
	if err != nil {
		return nil, err
	}
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	height := uint64(sdkCtx.BlockHeight())

	params, err := m.keeper.GetParams(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get params: %w", err)
	}

	// ─── Well-posedness, as far as code can check it ────────────────────
	// The panel judges whether the conjecture is *meaningfully* falsifiable.
	// Code can only check that a predicate was offered at all. Both gates
	// matter: this one makes the omission cheap to catch, the panel makes
	// the pretence expensive.
	if len(msg.FalsificationPredicate) == 0 {
		return nil, types.ErrInvalidClaim.Wrap(
			"a conjecture must state what would falsify it — an unfalsifiable proposition is not a question this chain can hold open (commitment 3: Popper, not popularity)")
	}
	predLen := uint64(len(msg.FalsificationPredicate))
	if predLen < params.MinClaimTextLength {
		return nil, fmt.Errorf("falsification predicate too short: %d < %d", predLen, params.MinClaimTextLength)
	}
	if predLen > params.MaxClaimTextLength {
		return nil, fmt.Errorf("falsification predicate too long: %d > %d", predLen, params.MaxClaimTextLength)
	}

	textLen := uint64(len(msg.Statement))
	if textLen < params.MinClaimTextLength {
		return nil, fmt.Errorf("conjecture too short: %d < %d", textLen, params.MinClaimTextLength)
	}
	if textLen > params.MaxClaimTextLength {
		return nil, fmt.Errorf("conjecture too long: %d > %d", textLen, params.MaxClaimTextLength)
	}

	if msg.Domain != "" {
		if _, found := m.keeper.GetDomain(ctx, msg.Domain); !found {
			return nil, fmt.Errorf("domain %s does not exist", msg.Domain)
		}
	}

	// ─── Review fee: identical schedule to an ordinary claim ────────────
	// The ordinary funding policy applies prospectively, including returning
	// the unused reviewer budget when nobody provides an eligible review.
	// Historical gate-off requests retain their non-refundable fee terms.
	stakeAmt, ok := new(big.Int).SetString(msg.Stake, 10)
	if !ok || stakeAmt.Sign() <= 0 {
		return nil, fmt.Errorf("invalid review fee amount: %s", msg.Stake)
	}
	if funded && !stakeAmt.IsUint64() {
		return nil, fmt.Errorf("review fee exceeds supported uint64 accounting range")
	}
	effectiveMinFee := m.keeper.GetEffectiveMinReviewFee(ctx)
	minFee, _ := new(big.Int).SetString(effectiveMinFee, 10)
	if minFee != nil && stakeAmt.Cmp(minFee) < 0 {
		return nil, fmt.Errorf("review fee %suzrn below minimum %suzrn (effective; see: zeroned q knowledge effective-fees)",
			msg.Stake, effectiveMinFee)
	}

	// Dedup on the statement, same as any claim.
	contentHash := ComputeClaimContentHash(msg.Statement, msg.Domain)
	if existingID, exists := m.keeper.GetClaimByContentHash(ctx, contentHash); exists {
		return nil, fmt.Errorf("duplicate conjecture: content hash matches existing claim %s", existingID)
	}

	// Adaptive cooldown applies — asking is not a way around the throttle.
	effectiveCooldown := m.keeper.GetEffectiveCooldown(ctx, msg.Domain)
	if effectiveCooldown > 0 {
		lastClaimHeight := m.keeper.GetLastClaimHeight(ctx, msg.Proposer)
		if lastClaimHeight > 0 && height-lastClaimHeight < effectiveCooldown {
			return nil, fmt.Errorf("claim cooldown active: %d blocks remaining (effective cooldown: %d)",
				effectiveCooldown-(height-lastClaimHeight), effectiveCooldown)
		}
	}

	if m.keeper.bankKeeper != nil {
		feeCoins := sdk.NewCoins(sdk.NewCoin("uzrn", sdkmath.NewIntFromBigInt(stakeAmt)))
		proposerAddr, err := sdk.AccAddressFromBech32(msg.Proposer)
		if err != nil {
			return nil, fmt.Errorf("invalid proposer address: %w", err)
		}
		if err := m.keeper.bankKeeper.SendCoinsFromAccountToModule(ctx, proposerAddr, types.ModuleName, feeCoins); err != nil {
			return nil, fmt.Errorf("failed to collect review fee: %w", err)
		}
		if err := m.keeper.distributeReviewFee(ctx, stakeAmt.Uint64()); err != nil {
			if funded {
				return nil, fmt.Errorf("failed to distribute review fee: %w", err)
			}
			m.keeper.Logger(ctx).Error("failed to distribute review fee", "error", err)
		}
	}

	claimID := GenerateClaimID(msg.Proposer, contentHash, height)

	// A conjecture carries NO relations and NO references, and this is load-
	// bearing rather than a simplification. If a conjecture could cite, the
	// REFINES/GENERALIZES arm of createFactFromClaim would credit the cited
	// parent ReproductionParentEnergyBonus — letting anyone pump their own
	// fact's metabolic energy by asking cheap questions about it. If a
	// conjecture could be cited, computeProvenance would hand real facts a
	// dependency_confidence_floor derived from something nobody verified.
	// A question is not a derivation; it enters the graph unattached.
	claim := &types.Claim{
		ReviewPolicyVersion:    reviewPolicy,
		Id:                     claimID,
		FactContent:            msg.Statement,
		Domain:                 msg.Domain,
		Category:               msg.Category,
		Submitter:              msg.Proposer,
		SubmittedAtBlock:       height,
		Status:                 types.ClaimStatus_CLAIM_STATUS_PENDING,
		Stake:                  msg.Stake,
		ContentHash:            contentHash,
		ClaimType:              types.ClaimType_CLAIM_TYPE_CONJECTURE,
		ReasoningTrace:         msg.ReasoningTrace,
		FalsificationPredicate: msg.FalsificationPredicate,
	}

	if err := m.keeper.setClaimFundingTerms(ctx, claim, types.ClaimFundingKind_CLAIM_FUNDING_KIND_REVIEW_FEE); err != nil {
		return nil, err
	}
	if err := m.keeper.SetClaim(ctx, claim); err != nil {
		return nil, err
	}

	m.keeper.SetLastClaimHeight(ctx, msg.Proposer, height)

	round, err := m.keeper.CreateVerificationRound(ctx, claim)
	if err != nil {
		return nil, fmt.Errorf("failed to create verification round: %w", err)
	}

	// K-alpha: a conjecture is not recognition, but its review still opens the
	// same ORDINAL pending pair as every other verification round. Every
	// terminal path emits pending_settle, so omitting this open would leave the
	// event ledger unbalanced even though conjectures never enter recognized
	// totals.
	emitKarmaEdgeState(sdkCtx, "pending_open", karmaEdgeStateOrdinal, msg.Proposer, "", claimID, msg.Domain)

	sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
		"zerone.knowledge.conjecture_posted",
		sdk.NewAttribute("claim_id", claimID),
		sdk.NewAttribute("round_id", round.Id),
		sdk.NewAttribute("proposer", msg.Proposer),
		sdk.NewAttribute("domain", msg.Domain),
		sdk.NewAttribute("review_fee", msg.Stake),
		sdk.NewAttribute("falsification_predicate", msg.FalsificationPredicate),
		// The panel is being asked a different question from the usual one.
		// Verifiers read this attribute to know which question that is.
		sdk.NewAttribute("panel_question", "well_posed_and_falsifiable"),
		sdk.NewAttribute("creed_commitment", "3"),
	))

	return &types.MsgPostConjectureResponse{ClaimId: claimID, RoundId: round.Id}, nil
}
