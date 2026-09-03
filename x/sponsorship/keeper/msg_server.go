package keeper

import (
	"context"
	"fmt"
	"math"
	"math/big"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	knowledgetypes "github.com/zerone-chain/zerone/x/knowledge/types"
	"github.com/zerone-chain/zerone/x/sponsorship/types"
)

type msgServer struct {
	types.UnimplementedMsgServer
	Keeper
}

func NewMsgServerImpl(k Keeper) types.MsgServer { return &msgServer{Keeper: k} }

var _ types.MsgServer = msgServer{}

// CreateBountyOrder escrows price_per_artifact × target_count uzrn from
// the sponsor's account to the sponsorship module account and records
// the bounty with ACTIVE status. The escrow is the chain's mechanical
// honoring of the sponsor's commitment — funds remain locked until the
// bounty fulfills or expires and is canceled.
func (m msgServer) CreateBountyOrder(goCtx context.Context, msg *types.MsgCreateBountyOrder) (*types.MsgCreateBountyOrderResponse, error) {
	active, err := m.knowledgeKeeper.AgentEconomyActivated(goCtx)
	if err != nil {
		return nil, types.ErrAgentEconomyDisabled.Wrap(err.Error())
	}
	if !active {
		return nil, types.ErrAgentEconomyDisabled
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	if err := msg.ValidateBasic(); err != nil {
		return nil, fmt.Errorf("%w: %v", types.ErrInvalidConfig, err)
	}
	canonicalSponsor, err := types.CanonicalAccountAddress(msg.Sponsor)
	if err != nil || canonicalSponsor != msg.Sponsor {
		return nil, fmt.Errorf("%w: sponsor must use canonical lowercase bech32 encoding", types.ErrInvalidConfig)
	}
	sponsorAddr, err := sdk.AccAddressFromBech32(canonicalSponsor)
	if err != nil {
		return nil, fmt.Errorf("invalid sponsor address: %w", err)
	}
	params, err := m.getParamsChecked(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: read params: %v", types.ErrInvalidConfig, err)
	}

	// Param-floor checks.
	if msg.TargetCount < params.MinTargetCount {
		return nil, fmt.Errorf("%w: target_count %d < min %d", types.ErrInvalidConfig, msg.TargetCount, params.MinTargetCount)
	}
	if msg.DurationBlocks < params.MinDurationBlocks {
		return nil, fmt.Errorf("%w: duration_blocks %d < min %d", types.ErrInvalidConfig, msg.DurationBlocks, params.MinDurationBlocks)
	}
	currentBlock := uint64(ctx.BlockHeight())
	if err := m.PruneExpiredBountiesForSponsor(ctx, canonicalSponsor, currentBlock); err != nil {
		return nil, fmt.Errorf("%w: prune sponsor orders: %v", types.ErrEscrowInvariant, err)
	}
	activeBounties, err := m.countActiveBountiesBySponsorChecked(ctx, canonicalSponsor)
	if err != nil {
		return nil, fmt.Errorf("%w: count sponsor orders: %v", types.ErrEscrowInvariant, err)
	}
	if activeBounties >= params.MaxActiveBountiesPerSponsor {
		return nil, fmt.Errorf("%w: max active bounties for sponsor reached (%d)", types.ErrInvalidConfig, params.MaxActiveBountiesPerSponsor)
	}

	// Compute total escrow = price × target_count.
	price, err := types.ParsePositiveAmount(msg.PricePerArtifact)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid price_per_artifact: %v", types.ErrInvalidConfig, err)
	}
	totalEscrow := new(big.Int).Mul(price, big.NewInt(int64(msg.TargetCount)))
	if totalEscrow.BitLen() > sdkmath.MaxBitLen {
		return nil, fmt.Errorf("%w: total escrow exceeds %d-bit sdk.Int limit", types.ErrInvalidConfig, sdkmath.MaxBitLen)
	}
	if msg.DurationBlocks > math.MaxUint64-currentBlock {
		return nil, fmt.Errorf("%w: end_block overflows uint64", types.ErrInvalidConfig)
	}
	if err := m.canAllocateBountyID(ctx); err != nil {
		return nil, fmt.Errorf("%w: %v", types.ErrInvalidConfig, err)
	}

	// Verify sponsor has the funds.
	spendable := m.bankKeeper.SpendableCoins(ctx, sponsorAddr)
	if spendable.AmountOf("uzrn").BigInt().Cmp(totalEscrow) < 0 {
		return nil, fmt.Errorf("%w: need %s uzrn, sponsor has %s",
			types.ErrInsufficientEscrow, totalEscrow.String(), spendable.AmountOf("uzrn").String())
	}
	// A new deposit must never paper over a pre-existing sponsorship shortfall.
	// Once the old state is solvent, adding the same amount to both balance and
	// liability preserves the invariant mechanically.
	if err := m.EnsureEscrowSolvent(ctx); err != nil {
		return nil, fmt.Errorf("%w: %v", types.ErrEscrowInvariant, err)
	}
	if err := m.CanIncreaseEscrowLiability(ctx, totalEscrow); err != nil {
		return nil, fmt.Errorf("%w: %v", types.ErrEscrowInvariant, err)
	}

	// Lock escrow.
	coins := sdk.NewCoins(sdk.NewCoin("uzrn", sdkmath.NewIntFromBigInt(totalEscrow)))
	if err := m.bankKeeper.SendCoinsFromAccountToModule(ctx, sponsorAddr, types.ModuleName, coins); err != nil {
		return nil, fmt.Errorf("lock escrow: %w", err)
	}

	// Build and store the bounty.
	nextID, err := m.nextBountyID(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", types.ErrInvalidConfig, err)
	}
	id := fmt.Sprintf("bounty-%d", nextID)
	order := &types.BountyOrder{
		Id:               id,
		Sponsor:          canonicalSponsor,
		Domain:           msg.Domain,
		PricePerArtifact: msg.PricePerArtifact,
		TargetCount:      msg.TargetCount,
		FulfilledCount:   0,
		EscrowRemaining:  totalEscrow.String(),
		StartBlock:       currentBlock,
		EndBlock:         currentBlock + msg.DurationBlocks,
		Status:           types.BountyStatus_BOUNTY_STATUS_ACTIVE,
		WorkContract:     msg.WorkContract,
	}
	if err := m.setBountyOrderChecked(ctx, order); err != nil {
		return nil, fmt.Errorf("store bounty: %w", err)
	}
	if err := m.IncreaseEscrowLiability(ctx, totalEscrow); err != nil {
		return nil, fmt.Errorf("%w: increase liability: %v", types.ErrEscrowInvariant, err)
	}
	if err := m.indexActiveBountyChecked(ctx, order); err != nil {
		return nil, fmt.Errorf("index bounty: %w", err)
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"zerone.sponsorship.bounty_created",
			sdk.NewAttribute("bounty_id", id),
			sdk.NewAttribute("sponsor", canonicalSponsor),
			sdk.NewAttribute("domain", msg.Domain),
			sdk.NewAttribute("price_per_artifact", msg.PricePerArtifact),
			sdk.NewAttribute("target_count", fmt.Sprintf("%d", msg.TargetCount)),
			sdk.NewAttribute("total_escrow", totalEscrow.String()),
			sdk.NewAttribute("end_block", fmt.Sprintf("%d", order.EndBlock)),
			sdk.NewAttribute("work_spec_hash", order.WorkContract.WorkSpecHash),
			sdk.NewAttribute("acceptance_hash", order.WorkContract.AcceptanceHash),
			sdk.NewAttribute("input_root", order.WorkContract.InputRoot),
			sdk.NewAttribute("environment_root", order.WorkContract.EnvironmentRoot),
			sdk.NewAttribute("min_corroborations", fmt.Sprintf("%d", order.WorkContract.MinCorroborations)),
			sdk.NewAttribute("worker_address", order.WorkContract.WorkerAddress),
			sdk.NewAttribute("creed_commitment", "20"),
		),
	)

	return &types.MsgCreateBountyOrderResponse{BountyId: id}, nil
}

// FulfillBounty pays the submitter of fact_id the bounty's per-artifact
// price, provided the fact meets all criteria. The stored submitter must equal
// the contract's preassigned worker and sign settlement. The payee is always
// derived from stored state.
func (m msgServer) FulfillBounty(goCtx context.Context, msg *types.MsgFulfillBounty) (*types.MsgFulfillBountyResponse, error) {
	active, err := m.knowledgeKeeper.AgentEconomyActivated(goCtx)
	if err != nil {
		return nil, types.ErrAgentEconomyDisabled.Wrap(err.Error())
	}
	if !active {
		return nil, types.ErrAgentEconomyDisabled
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	if err := msg.ValidateBasic(); err != nil {
		return nil, fmt.Errorf("%w: %v", types.ErrInvalidConfig, err)
	}

	order, found, err := m.getBountyOrderChecked(ctx, msg.BountyId)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", types.ErrEscrowInvariant, err)
	}
	if !found {
		return nil, fmt.Errorf("%w: %s", types.ErrBountyNotFound, msg.BountyId)
	}
	if order.Status != types.BountyStatus_BOUNTY_STATUS_ACTIVE {
		return nil, fmt.Errorf("%w: status %s", types.ErrBountyNotActive, order.Status)
	}
	currentBlock := uint64(ctx.BlockHeight())
	if currentBlock >= order.EndBlock {
		return nil, types.ErrBountyExpired
	}
	if order.WorkContract == nil {
		return nil, fmt.Errorf("%w: legacy bounty %s is refund-only", types.ErrWorkContractRequired, order.Id)
	}
	if err := order.WorkContract.Validate(); err != nil {
		return nil, fmt.Errorf("%w: corrupt work contract: %v", types.ErrInvalidConfig, err)
	}
	if _, exists, err := m.getFulfillmentChecked(ctx, order.Id, msg.FactId); err != nil {
		return nil, fmt.Errorf("%w: %v", types.ErrEscrowInvariant, err)
	} else if exists {
		return nil, fmt.Errorf("%w: %s/%s", types.ErrAlreadyFulfilled, order.Id, msg.FactId)
	}
	if _, consumed, err := m.consumptionOwnerChecked(
		ctx,
		types.FactConsumptionKey(msg.FactId),
		"fact",
	); err != nil {
		return nil, fmt.Errorf("%w: %v", types.ErrEscrowInvariant, err)
	} else if consumed {
		return nil, fmt.Errorf("%w: fact %s", types.ErrSettlementReplay, msg.FactId)
	}

	fact, ok := m.knowledgeKeeper.GetFact(ctx, msg.FactId)
	if !ok {
		return nil, fmt.Errorf("%w: fact %s not found", types.ErrFactNotEligible, msg.FactId)
	}
	if msg.Caller != fact.Submitter {
		return nil, fmt.Errorf("%w: fulfillment caller %s is not fact submitter %s",
			types.ErrUnauthorized, msg.Caller, fact.Submitter)
	}
	if fact.Status != knowledgetypes.FactStatus_FACT_STATUS_VERIFIED &&
		fact.Status != knowledgetypes.FactStatus_FACT_STATUS_ACTIVE {
		return nil, fmt.Errorf("%w: fact status %s (need VERIFIED or ACTIVE with no open challenge)", types.ErrFactNotEligible, fact.Status)
	}
	if fact.Domain != order.Domain {
		return nil, fmt.Errorf("%w: fact domain %q != bounty domain %q", types.ErrFactNotEligible, fact.Domain, order.Domain)
	}
	if fact.SubmittedAtBlock < order.StartBlock {
		return nil, fmt.Errorf("%w: fact submitted at block %d, bounty starts at %d (no retroactive payouts)",
			types.ErrFactNotEligible, fact.SubmittedAtBlock, order.StartBlock)
	}
	if fact.ClaimType != knowledgetypes.ClaimType_CLAIM_TYPE_COMPUTATIONAL || fact.ComputationalCommitment == nil {
		return nil, fmt.Errorf("%w: fact is not a bound computational result", types.ErrFactNotEligible)
	}
	work := fact.ComputationalCommitment
	if err := work.Validate(); err != nil {
		return nil, fmt.Errorf("%w: invalid computational commitment: %v", types.ErrFactNotEligible, err)
	}
	if err := knowledgetypes.ValidateWorkReceiptBinding(work, fact.Submitter); err != nil {
		return nil, fmt.Errorf("%w: %v", types.ErrFactNotEligible, err)
	}
	if !order.WorkContract.MatchesComputationalCommitment(work) {
		return nil, fmt.Errorf("%w: fact does not match bounty work contract", types.ErrFactNotEligible)
	}
	if fact.Submitter != order.WorkContract.WorkerAddress {
		return nil, fmt.Errorf("%w: fact submitter %s is not assigned worker %s",
			types.ErrUnauthorized, fact.Submitter, order.WorkContract.WorkerAddress)
	}
	// Settlement maturity is an AND gate. ACTIVE alone and an early challenge
	// win do not bypass the original window; CHALLENGED and every degraded
	// status were refused above.
	if fact.ChallengeWindowEnd == 0 || currentBlock < fact.ChallengeWindowEnd ||
		fact.CorroborationCount < order.WorkContract.MinCorroborations {
		return nil, fmt.Errorf("%w: challenge_window_end=%d current_block=%d corroborations=%d required=%d",
			types.ErrFactNotMature, fact.ChallengeWindowEnd, currentBlock,
			fact.CorroborationCount, order.WorkContract.MinCorroborations)
	}
	nullifier := types.ComputeSettlementNullifier(
		work.WorkSpecHash, work.AcceptanceHash, work.InputRoot, work.EnvironmentRoot, work.ArtifactRoot,
		order.WorkContract.WorkerAddress,
	)
	if _, consumed, err := m.consumptionOwnerChecked(
		ctx,
		types.ReceiptConsumptionKey(work.WorkReceiptHash),
		"receipt",
	); err != nil {
		return nil, fmt.Errorf("%w: %v", types.ErrEscrowInvariant, err)
	} else if consumed {
		return nil, fmt.Errorf("%w: receipt %s", types.ErrSettlementReplay, work.WorkReceiptHash)
	}
	if _, consumed, err := m.consumptionOwnerChecked(
		ctx,
		types.SettlementNullifierKey(nullifier),
		"settlement nullifier",
	); err != nil {
		return nil, fmt.Errorf("%w: %v", types.ErrEscrowInvariant, err)
	} else if consumed {
		return nil, fmt.Errorf("%w: nullifier %s", types.ErrSettlementReplay, nullifier)
	}

	expectedRemaining, err := types.ExpectedEscrowRemaining(order)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", types.ErrEscrowInvariant, err)
	}
	storedRemaining, err := types.ParseNonNegativeAmount(order.EscrowRemaining)
	if err != nil || storedRemaining.Cmp(expectedRemaining) != 0 {
		return nil, fmt.Errorf("%w: stored=%s expected=%s", types.ErrEscrowInvariant, order.EscrowRemaining, expectedRemaining)
	}
	if err := m.EnsureEscrowSolvent(ctx); err != nil {
		return nil, fmt.Errorf("%w: %v", types.ErrEscrowInvariant, err)
	}

	// Compute payout = price_per_artifact.
	price, err := types.ParsePositiveAmount(order.PricePerArtifact)
	if err != nil {
		return nil, fmt.Errorf("%w: corrupt bounty price: %v", types.ErrInvalidConfig, err)
	}
	if err := m.CanDecreaseEscrowLiability(ctx, price); err != nil {
		return nil, fmt.Errorf("%w: %v", types.ErrEscrowInvariant, err)
	}

	// Send price from module account to fact submitter.
	workerAddr, err := sdk.AccAddressFromBech32(fact.Submitter)
	if err != nil {
		return nil, fmt.Errorf("invalid fact submitter address: %w", err)
	}
	coins := sdk.NewCoins(sdk.NewCoin("uzrn", sdkmath.NewIntFromBigInt(price)))
	if err := m.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, workerAddr, coins); err != nil {
		return nil, fmt.Errorf("payout: %w", err)
	}

	// Update bounty: fulfilled_count, escrow_remaining, status.
	order.FulfilledCount++
	escrowRemaining := new(big.Int).Set(storedRemaining)
	escrowRemaining.Sub(escrowRemaining, price)
	order.EscrowRemaining = escrowRemaining.String()
	bountyNowFulfilled := order.FulfilledCount >= order.TargetCount
	if bountyNowFulfilled {
		order.Status = types.BountyStatus_BOUNTY_STATUS_FULFILLED
	}
	if err := m.setBountyOrderChecked(ctx, order); err != nil {
		return nil, fmt.Errorf("store fulfilled bounty: %w", err)
	}
	if bountyNowFulfilled {
		if err := m.unindexActiveBountyChecked(ctx, order); err != nil {
			return nil, fmt.Errorf("unindex fulfilled bounty: %w", err)
		}
	}
	if err := m.DecreaseEscrowLiability(ctx, price); err != nil {
		return nil, fmt.Errorf("%w: decrease liability: %v", types.ErrEscrowInvariant, err)
	}

	// Record fulfillment.
	if err := m.setFulfillmentChecked(ctx, &types.BountyFulfillment{
		BountyId:            order.Id,
		FactId:              msg.FactId,
		Worker:              fact.Submitter,
		AmountPaid:          price.String(),
		FulfilledAtBlock:    currentBlock,
		WorkReceiptHash:     work.WorkReceiptHash,
		SettlementNullifier: nullifier,
		ArtifactRoot:        work.ArtifactRoot,
	}); err != nil {
		return nil, fmt.Errorf("store fulfillment: %w", err)
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"zerone.sponsorship.bounty_fulfilled",
			sdk.NewAttribute("bounty_id", order.Id),
			sdk.NewAttribute("fact_id", msg.FactId),
			sdk.NewAttribute("worker", fact.Submitter),
			sdk.NewAttribute("worker_address", order.WorkContract.WorkerAddress),
			sdk.NewAttribute("settlement_signer", msg.Caller),
			sdk.NewAttribute("amount_paid", price.String()),
			sdk.NewAttribute("fulfilled_count", fmt.Sprintf("%d", order.FulfilledCount)),
			sdk.NewAttribute("target_count", fmt.Sprintf("%d", order.TargetCount)),
			sdk.NewAttribute("work_spec_hash", work.WorkSpecHash),
			sdk.NewAttribute("acceptance_hash", work.AcceptanceHash),
			sdk.NewAttribute("input_root", work.InputRoot),
			sdk.NewAttribute("artifact_root", work.ArtifactRoot),
			sdk.NewAttribute("evidence_root", work.EvidenceRoot),
			sdk.NewAttribute("environment_root", work.EnvironmentRoot),
			sdk.NewAttribute("work_receipt_hash", work.WorkReceiptHash),
			sdk.NewAttribute("settlement_nullifier", nullifier),
			sdk.NewAttribute("challenge_window_end", fmt.Sprintf("%d", fact.ChallengeWindowEnd)),
			sdk.NewAttribute("corroboration_count", fmt.Sprintf("%d", fact.CorroborationCount)),
			sdk.NewAttribute("creed_commitment", "20"),
		),
	)

	return &types.MsgFulfillBountyResponse{
		Worker:             fact.Submitter,
		AmountPaid:         price.String(),
		BountyNowFulfilled: bountyNowFulfilled,
	}, nil
}

// CancelBountyOrder lets the sponsor reclaim remaining escrow after a bound
// order expires. Legacy nil-contract orders retain v1 ACTIVE cancellation so
// stranded escrow stays recoverable. Only the original sponsor can cancel.
func (m msgServer) CancelBountyOrder(goCtx context.Context, msg *types.MsgCancelBountyOrder) (*types.MsgCancelBountyOrderResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	if err := msg.ValidateBasic(); err != nil {
		return nil, fmt.Errorf("%w: %v", types.ErrInvalidConfig, err)
	}

	order, found, err := m.getBountyOrderChecked(ctx, msg.BountyId)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", types.ErrEscrowInvariant, err)
	}
	if !found {
		return nil, fmt.Errorf("%w: %s", types.ErrBountyNotFound, msg.BountyId)
	}
	orderSponsorAddr, err := sdk.AccAddressFromBech32(order.Sponsor)
	if err != nil {
		return nil, fmt.Errorf("%w: bounty has invalid stored sponsor: %v", types.ErrInvalidConfig, err)
	}
	callerSponsorAddr, err := sdk.AccAddressFromBech32(msg.Sponsor)
	if err != nil {
		return nil, fmt.Errorf("invalid sponsor address: %w", err)
	}
	if !orderSponsorAddr.Equals(callerSponsorAddr) {
		return nil, fmt.Errorf("%w: bounty sponsor is %s, caller is %s",
			types.ErrUnauthorized, order.Sponsor, msg.Sponsor)
	}
	if order.Status != types.BountyStatus_BOUNTY_STATUS_ACTIVE && order.Status != types.BountyStatus_BOUNTY_STATUS_EXPIRED {
		return nil, fmt.Errorf("%w: cannot cancel a bounty in status %s", types.ErrBountyNotActive, order.Status)
	}
	// A bound offer is an economic commitment, not a revocable advertisement.
	// Once published, its sponsor cannot rug a worker who has already spent
	// compute/review/challenge costs. Legacy v1 orders retain their old active
	// cancellation behavior solely so stranded unbound escrow remains recoverable.
	currentBlock := uint64(ctx.BlockHeight())
	if order.WorkContract != nil && order.Status == types.BountyStatus_BOUNTY_STATUS_ACTIVE && currentBlock < order.EndBlock {
		return nil, fmt.Errorf("%w: bounty %s ends at block %d (current %d)",
			types.ErrBountyNotCancelable, order.Id, order.EndBlock, currentBlock)
	}

	remaining, err := types.ParseNonNegativeAmount(order.EscrowRemaining)
	if err != nil {
		return nil, fmt.Errorf("%w: corrupt escrow_remaining: %v", types.ErrInvalidConfig, err)
	}
	expectedRemaining, err := types.ExpectedEscrowRemaining(order)
	if err != nil || remaining.Cmp(expectedRemaining) != 0 {
		return nil, fmt.Errorf("%w: stored=%s expected=%v", types.ErrEscrowInvariant, order.EscrowRemaining, expectedRemaining)
	}
	if err := m.EnsureEscrowSolvent(ctx); err != nil {
		return nil, fmt.Errorf("%w: %v", types.ErrEscrowInvariant, err)
	}
	if err := m.CanDecreaseEscrowLiability(ctx, remaining); err != nil {
		return nil, fmt.Errorf("%w: %v", types.ErrEscrowInvariant, err)
	}

	// Refund escrow_remaining to sponsor (zero-refund is permitted if
	// the bounty was fully consumed; the cancel still flips status).
	if remaining.Sign() > 0 {
		coins := sdk.NewCoins(sdk.NewCoin("uzrn", sdkmath.NewIntFromBigInt(remaining)))
		if err := m.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, orderSponsorAddr, coins); err != nil {
			return nil, fmt.Errorf("refund: %w", err)
		}
	}

	// Remove both canonical and any historical raw alias key before rewriting
	// a terminal legacy record in canonical form.
	if err := m.unindexActiveBountyChecked(ctx, order); err != nil {
		return nil, fmt.Errorf("unindex canceled bounty: %w", err)
	}
	order.Sponsor = orderSponsorAddr.String()
	order.Status = types.BountyStatus_BOUNTY_STATUS_CANCELED
	order.EscrowRemaining = "0"
	if err := m.setBountyOrderChecked(ctx, order); err != nil {
		return nil, fmt.Errorf("store canceled bounty: %w", err)
	}
	if err := m.DecreaseEscrowLiability(ctx, remaining); err != nil {
		return nil, fmt.Errorf("%w: decrease liability: %v", types.ErrEscrowInvariant, err)
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"zerone.sponsorship.bounty_canceled",
			sdk.NewAttribute("bounty_id", order.Id),
			sdk.NewAttribute("sponsor", order.Sponsor),
			sdk.NewAttribute("refunded_amount", remaining.String()),
			sdk.NewAttribute("creed_commitment", "20"),
		),
	)

	return &types.MsgCancelBountyOrderResponse{RefundedAmount: remaining.String()}, nil
}
