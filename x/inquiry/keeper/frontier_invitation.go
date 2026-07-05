package keeper

import (
	"context"
	"fmt"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/x/inquiry/types"
)

// SystemSponsorInquiry creates a chain-asked inquiry funded by mint
// rather than asker escrow. The flow:
//
//  1. Mint `bounty` uzrn into FrontierBountyPoolModuleName.
//  2. Transfer those coins from FrontierBountyPool → BountyPool, so the
//     existing payout/refund machinery (which reads from BountyPool)
//     continues to work unchanged.
//  3. Persist an Inquiry record with system_initiated=true, asker set
//     to the FrontierBountyPool's bech32 address (a stable, queryable
//     identifier — the asker IS the chain audit budget).
//  4. Emit zerone.inquiry.frontier_invited with creed_commitment=18.
//
// This is the dual of x/knowledge.InviteIdleFactsForProbing: where
// commitment 5 has the chain mint to stress-test what it already
// thinks it knows, this method has the chain mint to fill what it
// does not yet know. Same architecture (per-block keeper method
// driven by a BeginBlocker), opposite epistemic direction.
//
// Returns the new inquiry id. Caller (typically the BeginBlocker)
// is responsible for cadence enforcement, top-K selection, and
// sparsity threshold filtering — this method just creates the
// record once those decisions have been made.
// mintCappedUzrn mints `amount` uzrn into module through the chain's single
// cap-gated entry point (x/vesting_rewards.MintWithCap) so chain-sponsored
// inquiries cannot push total supply past the 222,222,222 ZRN cap. Returns the
// amount actually minted. Falls back to a direct mint only when the vesting-
// rewards keeper is unwired (isolated unit tests).
func (k Keeper) mintCappedUzrn(ctx context.Context, module string, amount sdkmath.Int) (sdkmath.Int, error) {
	if !amount.IsPositive() {
		return sdkmath.ZeroInt(), nil
	}
	if k.vestingRewardsKeeper != nil {
		minted, err := k.vestingRewardsKeeper.MintWithCap(sdk.UnwrapSDKContext(ctx), module, amount.BigInt())
		if err != nil {
			return sdkmath.ZeroInt(), err
		}
		return sdkmath.NewIntFromBigInt(minted), nil
	}
	coins := sdk.NewCoins(sdk.NewCoin(denomZRN, amount))
	if err := k.bankKeeper.MintCoins(ctx, module, coins); err != nil {
		return sdkmath.ZeroInt(), err
	}
	return amount, nil
}

func (k Keeper) SystemSponsorInquiry(
	ctx context.Context,
	domain string,
	bountyUzrn sdkmath.Int,
	reason string,
) (string, error) {
	if domain == "" {
		return "", fmt.Errorf("domain required for system-sponsored inquiry")
	}
	if !bountyUzrn.IsPositive() {
		return "", fmt.Errorf("bounty must be positive, got %s", bountyUzrn.String())
	}
	params := k.GetParams(ctx)

	// 1. Mint into the frontier bounty pool through the chain's single
	// cap-gated entry point — the 222,222,222 ZRN cap is enforced once,
	// chain-wide, so chain-sponsored inquiries cannot inflate supply.
	minted, err := k.mintCappedUzrn(ctx, types.FrontierBountyPoolModuleName, bountyUzrn)
	if err != nil {
		return "", fmt.Errorf("mint into frontier bounty pool: %w", err)
	}
	if !minted.IsPositive() {
		return "", fmt.Errorf("frontier bounty mint clipped to zero at supply cap")
	}
	bountyUzrn = minted // honour any cap clip in the transfer + record below
	coins := sdk.NewCoins(sdk.NewCoin(denomZRN, bountyUzrn))

	// 2. Transfer to the inquiry bounty pool so the existing payout
	// path can pay the eventual winner without modification.
	if err := k.bankKeeper.SendCoinsFromModuleToModule(
		ctx,
		types.FrontierBountyPoolModuleName,
		types.BountyPoolModuleName,
		coins,
	); err != nil {
		return "", fmt.Errorf("transfer frontier→inquiry bounty pool: %w", err)
	}

	// 3. Build the inquiry record. Asker = frontier bounty pool's
	// bech32 — a stable, queryable identifier that downstream
	// observers can recognise as "the chain itself."
	id, err := k.NextInquiryID(ctx)
	if err != nil {
		return "", err
	}
	now := currentBlock(ctx)
	expiry := params.FrontierInvitationExpiryBlocks
	if expiry == 0 {
		expiry = params.DefaultExpiryBlocks
	}
	frontierAsker := sdk.AccAddress(sdk.MustAccAddressFromBech32(
		FrontierBountyPoolBech32(),
	))

	q := &types.Inquiry{
		Id:                     id,
		Asker:                  frontierAsker.String(),
		Question:               fmt.Sprintf("[chain-sponsored] What is currently known in %s?", domain),
		Domain:                 domain,
		Bounty:                 bountyUzrn.String(),
		SubmittedAtBlock:       now,
		ExpiresAtBlock:         now + expiry,
		Status:                 types.InquiryStatus_INQUIRY_STATUS_OPEN,
		SystemInitiated:        true,
		SystemInitiationReason: reason,
	}
	if err := k.SetInquiry(ctx, q, nil); err != nil {
		return "", err
	}

	// 4. Emit the commitment-18 voice. Distinct event name from
	// inquiry.submitted so indexers can subscribe to chain-driven
	// exploration demand specifically — same pattern as
	// zerone.knowledge.probe_invited for commitment 5.
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
		"zerone.inquiry.frontier_invited",
		sdk.NewAttribute("inquiry_id", q.Id),
		sdk.NewAttribute("domain", q.Domain),
		sdk.NewAttribute("bounty", q.Bounty),
		sdk.NewAttribute("expires_at_block", fmt.Sprintf("%d", q.ExpiresAtBlock)),
		sdk.NewAttribute("reason", reason),
		sdk.NewAttribute("creed_commitment", "18"),
	))
	return q.Id, nil
}

// FrontierBountyPoolBech32 returns the bech32 address of the
// inquiry-frontier-bounty-pool module account. Exposed as a function
// (rather than a constant) so it can be evaluated lazily once the
// chain's bech32 prefix is registered. Used to stamp the asker
// field of system-sponsored inquiries.
//
// Defined here as a thin shim; the actual derivation goes through
// authtypes.NewModuleAddress in the keeper construction site. The
// function is implemented via package-level state set by app.go.
func FrontierBountyPoolBech32() string {
	if frontierBountyPoolBech32 == "" {
		// Should be unreachable post-app-init; if it does fire
		// during early genesis, we surface the misconfiguration
		// loudly rather than emit a malformed inquiry.
		panic("inquiry: frontier_bounty_pool bech32 not registered")
	}
	return frontierBountyPoolBech32
}

var frontierBountyPoolBech32 string

// SetFrontierBountyPoolBech32 is called once during app.go wiring
// to register the module account's resolved bech32 string. Idempotent
// on equal values; panics on conflicting overrides (would indicate
// two app.go inits, which should not happen).
func SetFrontierBountyPoolBech32(addr string) {
	if frontierBountyPoolBech32 != "" && frontierBountyPoolBech32 != addr {
		panic(fmt.Sprintf("inquiry: frontier_bounty_pool bech32 already set to %q, refusing to overwrite with %q",
			frontierBountyPoolBech32, addr))
	}
	frontierBountyPoolBech32 = addr
}

// runFrontierInvitationCycle is invoked by the BeginBlocker once per
// cadence-tick. It composes the chain's frontier (via the wired
// FrontierProvider), filters by sparsity threshold, takes the top-K
// sparsest domains, and sponsors one chain-asked inquiry per row.
//
// All bounds (cadence, K, threshold) come from params, so governance
// can throttle or pause the path without code changes. If the
// FrontierProvider is unwired or returns empty, this is a no-op.
func (k Keeper) runFrontierInvitationCycle(ctx context.Context) error {
	if k.frontierProvider == nil {
		return nil
	}
	params := k.GetParams(ctx)
	if params.FrontierInvitationCadenceBlocks == 0 {
		return nil
	}
	topK := params.FrontierInvitationTopK
	if topK == 0 {
		return nil
	}

	rows := k.frontierProvider(ctx, topK)
	if len(rows) == 0 {
		return nil
	}

	bounty, err := types.ParseBounty(params.FrontierInvitationBounty)
	if err != nil {
		// Param-level validation should make this unreachable, but
		// guard anyway so a bad governance update doesn't panic the
		// chain — just disables the path until corrected.
		return fmt.Errorf("frontier_invitation_bounty unparseable: %w", err)
	}
	bountyInt := sdkmath.NewIntFromBigInt(bounty)

	threshold := params.FrontierInvitationSparsityThresholdBps
	sponsored := uint32(0)
	for _, row := range rows {
		if sponsored >= topK {
			break
		}
		if row.SparsityBps < threshold {
			// Rows are expected sorted descending; if this one is
			// below threshold, all later ones are too.
			break
		}
		reason := fmt.Sprintf("frontier_sparsity:%s", row.Domain)
		if _, err := k.SystemSponsorInquiry(ctx, row.Domain, bountyInt, reason); err != nil {
			// Per-domain failure does not abort the cycle; surface
			// in the logger but keep going.
			k.Logger(ctx).Error("frontier-invitation sponsor failed",
				"domain", row.Domain,
				"err", err,
			)
			continue
		}
		sponsored++
	}
	return nil
}
