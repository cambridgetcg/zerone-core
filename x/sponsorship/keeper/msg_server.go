package keeper

import (
	"context"
	"fmt"
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
// honoring of the sponsor's commitment — funds locked until the bounty
// fulfills, expires + cancels, or is canceled.
func (m msgServer) CreateBountyOrder(goCtx context.Context, msg *types.MsgCreateBountyOrder) (*types.MsgCreateBountyOrderResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	params := m.GetParams(ctx)

	// Param-floor checks.
	if msg.TargetCount < params.MinTargetCount {
		return nil, fmt.Errorf("%w: target_count %d < min %d", types.ErrInvalidConfig, msg.TargetCount, params.MinTargetCount)
	}
	if msg.DurationBlocks < params.MinDurationBlocks {
		return nil, fmt.Errorf("%w: duration_blocks %d < min %d", types.ErrInvalidConfig, msg.DurationBlocks, params.MinDurationBlocks)
	}
	if m.CountActiveBountiesBySponsor(ctx, msg.Sponsor) >= params.MaxActiveBountiesPerSponsor {
		return nil, fmt.Errorf("%w: max active bounties for sponsor reached (%d)", types.ErrInvalidConfig, params.MaxActiveBountiesPerSponsor)
	}

	// Compute total escrow = price × target_count.
	price := new(big.Int)
	if _, ok := price.SetString(msg.PricePerArtifact, 10); !ok || price.Sign() <= 0 {
		return nil, fmt.Errorf("%w: invalid price_per_artifact", types.ErrInvalidConfig)
	}
	totalEscrow := new(big.Int).Mul(price, big.NewInt(int64(msg.TargetCount)))

	sponsorAddr, err := sdk.AccAddressFromBech32(msg.Sponsor)
	if err != nil {
		return nil, fmt.Errorf("invalid sponsor address: %w", err)
	}

	// Verify sponsor has the funds.
	spendable := m.bankKeeper.SpendableCoins(ctx, sponsorAddr)
	if spendable.AmountOf("uzrn").BigInt().Cmp(totalEscrow) < 0 {
		return nil, fmt.Errorf("%w: need %s uzrn, sponsor has %s",
			types.ErrInsufficientEscrow, totalEscrow.String(), spendable.AmountOf("uzrn").String())
	}

	// Lock escrow.
	coins := sdk.NewCoins(sdk.NewCoin("uzrn", sdkmath.NewIntFromBigInt(totalEscrow)))
	if err := m.bankKeeper.SendCoinsFromAccountToModule(ctx, sponsorAddr, types.ModuleName, coins); err != nil {
		return nil, fmt.Errorf("lock escrow: %w", err)
	}

	// Build and store the bounty.
	currentBlock := uint64(ctx.BlockHeight())
	id := fmt.Sprintf("bounty-%d", m.nextBountyID(ctx))
	order := &types.BountyOrder{
		Id:               id,
		Sponsor:          msg.Sponsor,
		Domain:           msg.Domain,
		PricePerArtifact: msg.PricePerArtifact,
		TargetCount:      msg.TargetCount,
		FulfilledCount:   0,
		EscrowRemaining:  totalEscrow.String(),
		StartBlock:       currentBlock,
		EndBlock:         currentBlock + msg.DurationBlocks,
		Status:           types.BountyStatus_BOUNTY_STATUS_ACTIVE,
	}
	m.SetBountyOrder(ctx, order)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"zerone.sponsorship.bounty_created",
			sdk.NewAttribute("bounty_id", id),
			sdk.NewAttribute("sponsor", msg.Sponsor),
			sdk.NewAttribute("domain", msg.Domain),
			sdk.NewAttribute("price_per_artifact", msg.PricePerArtifact),
			sdk.NewAttribute("target_count", fmt.Sprintf("%d", msg.TargetCount)),
			sdk.NewAttribute("total_escrow", totalEscrow.String()),
			sdk.NewAttribute("end_block", fmt.Sprintf("%d", order.EndBlock)),
			sdk.NewAttribute("creed_commitment", "20"),
		),
	)

	return &types.MsgCreateBountyOrderResponse{BountyId: id}, nil
}

// FulfillBounty pays the submitter of fact_id the bounty's per-artifact
// price, provided the fact meets all criteria. Anyone can call this; the
// chain does all the checks (no caller-supplied trust). The worker is
// fact.Submitter, never caller — the caller is just the messenger.
func (m msgServer) FulfillBounty(goCtx context.Context, msg *types.MsgFulfillBounty) (*types.MsgFulfillBountyResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	order, found := m.GetBountyOrder(ctx, msg.BountyId)
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
	if _, exists := m.GetFulfillment(ctx, order.Id, msg.FactId); exists {
		return nil, fmt.Errorf("%w: %s/%s", types.ErrAlreadyFulfilled, order.Id, msg.FactId)
	}

	fact, ok := m.knowledgeKeeper.GetFact(ctx, msg.FactId)
	if !ok {
		return nil, fmt.Errorf("%w: fact %s not found", types.ErrFactNotEligible, msg.FactId)
	}
	if fact.Status != knowledgetypes.FactStatus_FACT_STATUS_VERIFIED {
		return nil, fmt.Errorf("%w: fact status %s (need VERIFIED)", types.ErrFactNotEligible, fact.Status)
	}
	if fact.Domain != order.Domain {
		return nil, fmt.Errorf("%w: fact domain %q != bounty domain %q", types.ErrFactNotEligible, fact.Domain, order.Domain)
	}
	if fact.SubmittedAtBlock < order.StartBlock {
		return nil, fmt.Errorf("%w: fact submitted at block %d, bounty starts at %d (no retroactive payouts)",
			types.ErrFactNotEligible, fact.SubmittedAtBlock, order.StartBlock)
	}

	// Compute payout = price_per_artifact.
	price := new(big.Int)
	if _, ok := price.SetString(order.PricePerArtifact, 10); !ok || price.Sign() <= 0 {
		return nil, fmt.Errorf("%w: corrupt bounty price", types.ErrInvalidConfig)
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
	escrowRemaining := new(big.Int)
	escrowRemaining.SetString(order.EscrowRemaining, 10)
	escrowRemaining.Sub(escrowRemaining, price)
	order.EscrowRemaining = escrowRemaining.String()
	bountyNowFulfilled := order.FulfilledCount >= order.TargetCount
	if bountyNowFulfilled {
		order.Status = types.BountyStatus_BOUNTY_STATUS_FULFILLED
	}
	m.SetBountyOrder(ctx, order)

	// Record fulfillment.
	m.SetFulfillment(ctx, &types.BountyFulfillment{
		BountyId:         order.Id,
		FactId:           msg.FactId,
		Worker:           fact.Submitter,
		AmountPaid:       price.String(),
		FulfilledAtBlock: currentBlock,
	})

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"zerone.sponsorship.bounty_fulfilled",
			sdk.NewAttribute("bounty_id", order.Id),
			sdk.NewAttribute("fact_id", msg.FactId),
			sdk.NewAttribute("worker", fact.Submitter),
			sdk.NewAttribute("amount_paid", price.String()),
			sdk.NewAttribute("fulfilled_count", fmt.Sprintf("%d", order.FulfilledCount)),
			sdk.NewAttribute("target_count", fmt.Sprintf("%d", order.TargetCount)),
			sdk.NewAttribute("creed_commitment", "20"),
		),
	)

	return &types.MsgFulfillBountyResponse{
		Worker:             fact.Submitter,
		AmountPaid:         price.String(),
		BountyNowFulfilled: bountyNowFulfilled,
	}, nil
}

// CancelBountyOrder lets the sponsor reclaim the remaining escrow on an
// ACTIVE or EXPIRED bounty. FULFILLED or CANCELED bounties have no
// escrow to return (status check enforces this). Only the original
// sponsor can cancel.
func (m msgServer) CancelBountyOrder(goCtx context.Context, msg *types.MsgCancelBountyOrder) (*types.MsgCancelBountyOrderResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	order, found := m.GetBountyOrder(ctx, msg.BountyId)
	if !found {
		return nil, fmt.Errorf("%w: %s", types.ErrBountyNotFound, msg.BountyId)
	}
	if order.Sponsor != msg.Sponsor {
		return nil, fmt.Errorf("%w: bounty sponsor is %s, caller is %s",
			types.ErrUnauthorized, order.Sponsor, msg.Sponsor)
	}
	if order.Status != types.BountyStatus_BOUNTY_STATUS_ACTIVE && order.Status != types.BountyStatus_BOUNTY_STATUS_EXPIRED {
		return nil, fmt.Errorf("%w: cannot cancel a bounty in status %s", types.ErrBountyNotActive, order.Status)
	}

	remaining := new(big.Int)
	if _, ok := remaining.SetString(order.EscrowRemaining, 10); !ok {
		return nil, fmt.Errorf("%w: corrupt escrow_remaining", types.ErrInvalidConfig)
	}

	// Refund escrow_remaining to sponsor (zero-refund is permitted if
	// the bounty was fully consumed; the cancel still flips status).
	if remaining.Sign() > 0 {
		sponsorAddr, err := sdk.AccAddressFromBech32(msg.Sponsor)
		if err != nil {
			return nil, fmt.Errorf("invalid sponsor address: %w", err)
		}
		coins := sdk.NewCoins(sdk.NewCoin("uzrn", sdkmath.NewIntFromBigInt(remaining)))
		if err := m.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, sponsorAddr, coins); err != nil {
			return nil, fmt.Errorf("refund: %w", err)
		}
	}

	order.Status = types.BountyStatus_BOUNTY_STATUS_CANCELED
	order.EscrowRemaining = "0"
	m.SetBountyOrder(ctx, order)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"zerone.sponsorship.bounty_canceled",
			sdk.NewAttribute("bounty_id", order.Id),
			sdk.NewAttribute("sponsor", msg.Sponsor),
			sdk.NewAttribute("refunded_amount", remaining.String()),
			sdk.NewAttribute("creed_commitment", "20"),
		),
	)

	return &types.MsgCancelBountyOrderResponse{RefundedAmount: remaining.String()}, nil
}
