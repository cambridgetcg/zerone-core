package keeper

import (
	"context"
	"fmt"
	"math/big"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// ─── Conjecture: the chain holds a question open ────────────────────────────
//
// Every other earning path in this module terminates in "a verification panel
// said ACCEPT". That makes expected revenue monotone in how already-known a
// claim is, which is the correct incentive for settled knowledge and exactly
// the wrong one for a frontier: a proposition that cannot be settled yet has
// negative expected value under every existing message, because the review fee
// is spent at submission and no payout can follow.
//
// A conjecture severs that link. It asserts nothing, so there is nothing for a
// panel to accept as true; the panel is asked only whether the proposition is
// WELL-POSED AND FALSIFIABLE. It earns its proposer nothing at all. The single
// paid act against a live conjecture is MsgChallengeProvisionalFact — which has
// existed, fully implemented and CLI-exposed, since before this file: its
// status gate requires FACT_STATUS_PROVISIONAL, and until now no code path in
// the module ever wrote that status. This handler is the ignition for machinery
// that was already built.
//
// The asymmetry is the point. You cannot be paid for asserting; you can only be
// paid for destroying. And because handleChallengeSurvival already increments
// CorroborationCount on the provisional-challenge path, the price of destroying
// a conjecture rises with how many probes it has already survived.

// errIfProvisionalCitation refuses a claim that rests any part of its proof
// chain on a conjecture.
//
// This is not tidiness. computeProvenance derives a claim's
// dependency_confidence_floor as the MINIMUM effective confidence across all
// its cited edges, and a conjecture's confidence is zero by construction. So a
// claim citing one solid fact and one conjecture computes a floor of zero —
// and a zero floor is read everywhere as "no floor at all", because the clamp
// in createFactFromClaim is guarded by `floor > 0`. Citing a question would
// therefore not weaken a proof chain, it would silently exempt it from the
// ceiling its real foundations impose. A question is not evidence, and it may
// not appear anywhere that evidence is counted.
func errIfProvisionalCitation(target *types.Fact) error {
	if target == nil {
		return nil
	}
	if target.Status == types.FactStatus_FACT_STATUS_PROVISIONAL {
		return types.ErrInvalidClaim.Wrapf(
			"fact %s is a conjecture (provisional) and cannot be cited — an unsettled proposition is not support for anything; refute it with `tx knowledge challenge-provisional` instead",
			target.Id)
	}
	return nil
}

// PostConjecture places an unsettled proposition into the graph.
func (m *msgServer) PostConjecture(ctx context.Context, msg *types.MsgPostConjecture) (*types.MsgPostConjectureResponse, error) {
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
	// A conjecture costs exactly what a claim costs and refunds exactly as
	// much: nothing. Charging less would make conjectures the cheap way to
	// occupy graph space; charging more would tax asking questions.
	// Deliberately NOT sponsorable — the bootstrap fund exists to seed
	// verifiable knowledge, and a fund that pays for unfalsified
	// propositions is a fund that can be drained by asking.
	stakeAmt, ok := new(big.Int).SetString(msg.Stake, 10)
	if !ok || stakeAmt.Sign() <= 0 {
		return nil, fmt.Errorf("invalid review fee amount: %s", msg.Stake)
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

	if err := m.keeper.SetClaim(ctx, claim); err != nil {
		return nil, err
	}

	m.keeper.SetLastClaimHeight(ctx, msg.Proposer, height)

	round, err := m.keeper.CreateVerificationRound(ctx, claim)
	if err != nil {
		return nil, fmt.Errorf("failed to create verification round: %w", err)
	}

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
