package keeper

import (
	"context"
	"fmt"
	"math/big"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/x/inquiry/types"
)

const denomZRN = "uzrn"

type msgServer struct {
	types.UnimplementedMsgServer
	keeper Keeper
}

func NewMsgServerImpl(k Keeper) types.MsgServer { return &msgServer{keeper: k} }

var _ types.MsgServer = &msgServer{}

func (m *msgServer) SubmitInquiry(ctx context.Context, msg *types.MsgSubmitInquiry) (*types.MsgSubmitInquiryResponse, error) {
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}
	params := m.keeper.GetParams(ctx)
	if !params.SubmissionsEnabled {
		return nil, fmt.Errorf("%w (commitment 16: the chain pays for exploration of the unknown — disabling inquiry submissions silences the demand side of the exploration market and must be a deliberate, governance-witnessed pause, not a silent default)", types.ErrSubmissionsDisabled)
	}
	if uint32(len(msg.Question)) > params.MaxQuestionBytes {
		return nil, fmt.Errorf("%w: question %d > %d", types.ErrTextTooLong, len(msg.Question), params.MaxQuestionBytes)
	}
	if uint32(len(msg.Context)) > params.MaxContextBytes {
		return nil, fmt.Errorf("%w: context %d > %d", types.ErrTextTooLong, len(msg.Context), params.MaxContextBytes)
	}
	bountyAmt, err := types.ParseBounty(msg.Bounty)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", types.ErrInvalidBounty, err)
	}
	minBounty, _ := types.ParseBounty(params.MinBounty)
	if bountyAmt.Cmp(minBounty) < 0 {
		return nil, fmt.Errorf("%w: %s < %s (commitment 16: the bounty is the chain's price signal for exploration — bounties below the minimum would let the unknown be claimed for nothing, eroding the exploration market the chain pays to maintain)", types.ErrBountyTooLow, msg.Bounty, params.MinBounty)
	}

	expiry := msg.ExpiryBlocks
	if expiry == 0 {
		expiry = params.DefaultExpiryBlocks
	}
	if expiry > params.MaxExpiryBlocks {
		return nil, fmt.Errorf("%w: %d > max %d", types.ErrInvalidExpiry, expiry, params.MaxExpiryBlocks)
	}

	askerAddr, err := sdk.AccAddressFromBech32(msg.Asker)
	if err != nil {
		return nil, err
	}
	// Escrow the bounty into the inquiry-bounty-pool module account.
	coins := sdk.NewCoins(sdk.NewCoin(denomZRN, sdkmath.NewIntFromBigInt(bountyAmt)))
	if err := m.keeper.bankKeeper.SendCoinsFromAccountToModule(ctx, askerAddr, types.BountyPoolModuleName, coins); err != nil {
		return nil, fmt.Errorf("escrow bounty: %w", err)
	}

	id, err := m.keeper.NextInquiryID(ctx)
	if err != nil {
		return nil, err
	}
	now := currentBlock(ctx)
	q := &types.Inquiry{
		Id:               id,
		Asker:            msg.Asker,
		Question:         msg.Question,
		Domain:           msg.Domain,
		Context:          msg.Context,
		Bounty:           msg.Bounty,
		SubmittedAtBlock: now,
		ExpiresAtBlock:   now + expiry,
		Status:           types.InquiryStatus_INQUIRY_STATUS_OPEN,
	}
	if err := m.keeper.SetInquiry(ctx, q, nil); err != nil {
		return nil, err
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	// Commitment 16 (chain pays for exploration of the unknown):
	// publishing a new inquiry IS the chain manufacturing demand for
	// unmapped territory.
	sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
		"zerone.inquiry.submitted",
		sdk.NewAttribute("inquiry_id", q.Id),
		sdk.NewAttribute("asker", q.Asker),
		sdk.NewAttribute("domain", q.Domain),
		sdk.NewAttribute("bounty", q.Bounty),
		sdk.NewAttribute("expires_at_block", fmt.Sprintf("%d", q.ExpiresAtBlock)),
		sdk.NewAttribute("creed_commitment", "16"),
	))
	return &types.MsgSubmitInquiryResponse{InquiryId: q.Id, ExpiresAtBlock: q.ExpiresAtBlock}, nil
}

func (m *msgServer) SubmitAnswer(ctx context.Context, msg *types.MsgSubmitAnswer) (*types.MsgSubmitAnswerResponse, error) {
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}
	params := m.keeper.GetParams(ctx)

	q, ok := m.keeper.GetInquiry(ctx, msg.InquiryId)
	if !ok {
		return nil, fmt.Errorf("%w: %s", types.ErrInquiryNotFound, msg.InquiryId)
	}
	if q.Status != types.InquiryStatus_INQUIRY_STATUS_OPEN &&
		q.Status != types.InquiryStatus_INQUIRY_STATUS_ANSWERED {
		return nil, fmt.Errorf("%w (commitment 16: an inquiry's resolution is the corpus's record of which exploration the chain paid for — answers cannot be appended to a closed call)", types.ErrInquiryNotOpen)
	}
	if m.keeper.CountAnswers(ctx, q.Id) >= params.MaxAnswersPerInquiry {
		return nil, fmt.Errorf("%w (commitment 16: the answer cap protects the resolution from being flooded; the chain pays the first whose claim accepts, not whoever spams the most attempts)", types.ErrTooManyAnswers)
	}
	if m.keeper.ClaimAlreadyLinked(ctx, msg.ClaimId) {
		return nil, types.ErrClaimAlreadyLinked
	}

	// If we have a knowledge keeper wired, refuse if the claim
	// doesn't exist or doesn't belong to the answerer. Otherwise
	// (unit tests in isolation) accept the link.
	if m.keeper.knowledgeKeeper != nil {
		submitter, ok := m.keeper.knowledgeKeeper.ClaimSubmitter(ctx, msg.ClaimId)
		if !ok {
			return nil, fmt.Errorf("%w: %s", types.ErrClaimNotFound, msg.ClaimId)
		}
		if submitter != msg.Answerer {
			return nil, types.ErrClaimNotOwned
		}
	}

	id, err := m.keeper.NextAnswerID(ctx)
	if err != nil {
		return nil, err
	}
	a := &types.Answer{
		Id:               id,
		InquiryId:        q.Id,
		Answerer:         msg.Answerer,
		ClaimId:          msg.ClaimId,
		SubmittedAtBlock: currentBlock(ctx),
		Status:           types.AnswerStatus_ANSWER_STATUS_PENDING,
	}
	if err := m.keeper.SetAnswer(ctx, a); err != nil {
		return nil, err
	}

	// Promote the inquiry to ANSWERED on first answer.
	if q.Status == types.InquiryStatus_INQUIRY_STATUS_OPEN {
		prev := *q
		q.Status = types.InquiryStatus_INQUIRY_STATUS_ANSWERED
		if err := m.keeper.SetInquiry(ctx, q, &prev); err != nil {
			return nil, err
		}
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
		"zerone.inquiry.answer_submitted",
		sdk.NewAttribute("answer_id", fmt.Sprintf("%d", a.Id)),
		sdk.NewAttribute("inquiry_id", q.Id),
		sdk.NewAttribute("answerer", a.Answerer),
		sdk.NewAttribute("claim_id", a.ClaimId),
		sdk.NewAttribute("creed_commitment", "16"),
	))
	return &types.MsgSubmitAnswerResponse{AnswerId: a.Id}, nil
}

func (m *msgServer) ResolveInquiry(ctx context.Context, msg *types.MsgResolveInquiry) (*types.MsgResolveInquiryResponse, error) {
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}
	q, ok := m.keeper.GetInquiry(ctx, msg.InquiryId)
	if !ok {
		return nil, fmt.Errorf("%w: %s", types.ErrInquiryNotFound, msg.InquiryId)
	}
	if q.Status == types.InquiryStatus_INQUIRY_STATUS_RESOLVED ||
		q.Status == types.InquiryStatus_INQUIRY_STATUS_EXPIRED ||
		q.Status == types.InquiryStatus_INQUIRY_STATUS_CANCELLED {
		return nil, fmt.Errorf("%w (commitment 16: an inquiry resolves once — the resolution is the corpus's permanent record of which exploration the chain bought, not a draft to be re-attempted)", types.ErrInquiryAlreadyResolved)
	}
	if err := m.keeper.tryResolveInquiry(ctx, q); err != nil {
		return nil, err
	}
	updated, _ := m.keeper.GetInquiry(ctx, q.Id)
	return &types.MsgResolveInquiryResponse{
		Status:        updated.Status,
		WinningFactId: updated.WinningFactId,
	}, nil
}

func (m *msgServer) CancelInquiry(ctx context.Context, msg *types.MsgCancelInquiry) (*types.MsgCancelInquiryResponse, error) {
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}
	q, ok := m.keeper.GetInquiry(ctx, msg.InquiryId)
	if !ok {
		return nil, fmt.Errorf("%w: %s", types.ErrInquiryNotFound, msg.InquiryId)
	}
	// Commitment 18: system-sponsored inquiries cannot be cancelled.
	// The chain does not withdraw its own asks — that withdrawal
	// would silently retract a publicly-posted exploration commitment
	// and let the chain pretend it never asked. Refuse structurally.
	if q.SystemInitiated {
		return nil, fmt.Errorf("%w (commitment 18: the chain manufactures exploration demand and does not withdraw its own asks — letting the chain quietly retract its frontier invitations would let observation become silence again, and silence is what the commitment refuses)", types.ErrSystemInitiated)
	}
	if q.Asker != msg.Asker {
		return nil, types.ErrNotAsker
	}
	if q.Status != types.InquiryStatus_INQUIRY_STATUS_OPEN {
		return nil, fmt.Errorf("%w (commitment 16: once answers are in flight, withdrawing the call would silently retract a publicly-posted exploration commitment — the chain honours the work already underway)", types.ErrAnswersInFlight)
	}

	if err := m.keeper.refundBounty(ctx, q); err != nil {
		return nil, err
	}
	prev := *q
	q.Status = types.InquiryStatus_INQUIRY_STATUS_CANCELLED
	q.ResolvedAtBlock = currentBlock(ctx)
	if err := m.keeper.SetInquiry(ctx, q, &prev); err != nil {
		return nil, err
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
		"zerone.inquiry.cancelled",
		sdk.NewAttribute("inquiry_id", q.Id),
		sdk.NewAttribute("asker", q.Asker),
		sdk.NewAttribute("creed_commitment", "16"),
	))
	return &types.MsgCancelInquiryResponse{}, nil
}

func (m *msgServer) UpdateParams(ctx context.Context, msg *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}
	if msg.Authority != m.keeper.Authority() {
		return nil, fmt.Errorf("%w: expected %s, got %s", types.ErrInvalidAuthority, m.keeper.Authority(), msg.Authority)
	}
	if msg.Params == nil {
		return nil, fmt.Errorf("params required")
	}
	if err := m.keeper.SetParams(ctx, *msg.Params); err != nil {
		return nil, err
	}
	return &types.MsgUpdateParamsResponse{}, nil
}

// ─── Resolution mechanics ────────────────────────────────────────────

// tryResolveInquiry checks the inquiry's answers for an accepted
// claim. If found, pay the winner and mark RESOLVED. If expired
// without resolution, refund and mark EXPIRED. Otherwise leave
// untouched. Used by both MsgResolveInquiry (manual) and BeginBlocker
// (auto).
func (k Keeper) tryResolveInquiry(ctx context.Context, q *types.Inquiry) error {
	if q.Status == types.InquiryStatus_INQUIRY_STATUS_RESOLVED ||
		q.Status == types.InquiryStatus_INQUIRY_STATUS_EXPIRED ||
		q.Status == types.InquiryStatus_INQUIRY_STATUS_CANCELLED {
		return nil
	}

	// Look for a winning answer.
	if k.knowledgeKeeper != nil {
		var winner *types.Answer
		var winningFactID string
		_ = k.IterateAnswersByInquiry(ctx, q.Id, func(a *types.Answer) bool {
			if a.Status != types.AnswerStatus_ANSWER_STATUS_PENDING {
				return false
			}
			factID, accepted := k.knowledgeKeeper.AcceptedFactForClaim(ctx, a.ClaimId)
			if accepted {
				winner = a
				winningFactID = factID
				return true
			}
			return false
		})
		if winner != nil {
			return k.payWinner(ctx, q, winner, winningFactID)
		}
	}

	// Check for expiry.
	if currentBlock(ctx) >= q.ExpiresAtBlock {
		return k.expireInquiry(ctx, q)
	}
	return nil
}

func (k Keeper) payWinner(ctx context.Context, q *types.Inquiry, winner *types.Answer, factID string) error {
	bountyAmt, err := types.ParseBounty(q.Bounty)
	if err != nil {
		return err
	}
	winnerAddr, err := sdk.AccAddressFromBech32(winner.Answerer)
	if err != nil {
		return err
	}
	coins := sdk.NewCoins(sdk.NewCoin(denomZRN, sdkmath.NewIntFromBigInt(bountyAmt)))
	if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.BountyPoolModuleName, winnerAddr, coins); err != nil {
		return fmt.Errorf("pay winner: %w", err)
	}

	prev := *q
	q.Status = types.InquiryStatus_INQUIRY_STATUS_RESOLVED
	q.WinningFactId = factID
	q.Winner = winner.Answerer
	q.ResolvedAtBlock = currentBlock(ctx)
	if err := k.SetInquiry(ctx, q, &prev); err != nil {
		return err
	}
	winner.Status = types.AnswerStatus_ANSWER_STATUS_WON
	winner.FactId = factID
	if err := k.SetAnswer(ctx, winner); err != nil {
		return err
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
		"zerone.inquiry.resolved",
		sdk.NewAttribute("inquiry_id", q.Id),
		sdk.NewAttribute("winner", winner.Answerer),
		sdk.NewAttribute("winning_fact_id", factID),
		sdk.NewAttribute("bounty", q.Bounty),
		sdk.NewAttribute("resolved_at_block", fmt.Sprintf("%d", q.ResolvedAtBlock)),
		sdk.NewAttribute("creed_commitment", "16"),
	))
	return nil
}

func (k Keeper) expireInquiry(ctx context.Context, q *types.Inquiry) error {
	if err := k.refundBounty(ctx, q); err != nil {
		return err
	}
	prev := *q
	q.Status = types.InquiryStatus_INQUIRY_STATUS_EXPIRED
	q.ResolvedAtBlock = currentBlock(ctx)
	if err := k.SetInquiry(ctx, q, &prev); err != nil {
		return err
	}
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
		"zerone.inquiry.expired",
		sdk.NewAttribute("inquiry_id", q.Id),
		sdk.NewAttribute("asker", q.Asker),
		sdk.NewAttribute("resolved_at_block", fmt.Sprintf("%d", q.ResolvedAtBlock)),
		sdk.NewAttribute("creed_commitment", "16"),
	))
	return nil
}

// refundBounty returns an unspent bounty to its source. For
// user-asked inquiries (commitment 16), the source is the asker's
// account. For system-sponsored inquiries (commitment 18), the
// source is the FrontierBountyPool — the chain's exploration audit
// budget conserves itself across unanswered cycles, so an expired
// system inquiry replenishes the pool rather than leaking into
// general circulation.
func (k Keeper) refundBounty(ctx context.Context, q *types.Inquiry) error {
	bountyAmt, err := types.ParseBounty(q.Bounty)
	if err != nil {
		return err
	}
	coins := sdk.NewCoins(sdk.NewCoin(denomZRN, sdkmath.NewIntFromBigInt(bountyAmt)))

	if q.SystemInitiated {
		// Round-trip back to the frontier pool. Preserves the
		// "the chain pays for its own audit" commitment (12) at the
		// audit-budget layer: mints in, mints out, balance reflects
		// outstanding chain-driven exploration spend.
		if err := k.bankKeeper.SendCoinsFromModuleToModule(
			ctx,
			types.BountyPoolModuleName,
			types.FrontierBountyPoolModuleName,
			coins,
		); err != nil {
			return fmt.Errorf("return system-initiated bounty to frontier pool: %w", err)
		}
		return nil
	}

	askerAddr, err := sdk.AccAddressFromBech32(q.Asker)
	if err != nil {
		return err
	}
	if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.BountyPoolModuleName, askerAddr, coins); err != nil {
		return fmt.Errorf("refund bounty: %w", err)
	}
	return nil
}

// keep the math import alive when bankKeeper is nil-checked elsewhere.
var _ = big.NewInt
