package keeper

import "context"

import "github.com/zerone-chain/zerone/x/inquiry/types"

// BeginBlocker performs two responsibilities in a fixed order:
//
//  1. RESOLUTION SCAN (commitment 16) — scan OPEN and ANSWERED
//     inquiries for resolution.
//
//     For each inquiry up to params.begin_blocker_scan_limit:
//       - If any linked answer's claim has been accepted, resolve to
//         RESOLVED and pay the bounty to the answerer.
//       - Else, if expiry block reached, resolve to EXPIRED and
//         refund the asker (or return the bounty to the frontier
//         pool, for system-sponsored inquiries).
//       - Otherwise leave untouched.
//
//     The scan is bounded so a runaway pile of unresolved inquiries
//     cannot consume unbounded gas. Older inquiries that fall
//     outside the scan window can still be resolved via
//     MsgResolveInquiry.
//
//  2. FRONTIER-INVITATION CYCLE (commitment 18) — once per
//     params.frontier_invitation_cadence_blocks, walk the chain's
//     frontier and SPONSOR open inquiries in the sparsest domains.
//     Funded by mint into the frontier-bounty pool; bounded by
//     params.frontier_invitation_top_k.
//
//     Disabled when cadence == 0 OR FrontierProvider is unwired.
//     Per-domain failures inside the cycle do not abort BeginBlocker.
func (k Keeper) BeginBlocker(ctx context.Context) error {
	params := k.GetParams(ctx)
	limit := params.BeginBlockerScanLimit

	// 1. Resolution scan.
	scanned := uint32(0)
	// Scan ANSWERED first (most likely to resolve to RESOLVED), then
	// OPEN (resolvable only via expiry).
	for _, status := range []types.InquiryStatus{
		types.InquiryStatus_INQUIRY_STATUS_ANSWERED,
		types.InquiryStatus_INQUIRY_STATUS_OPEN,
	} {
		if scanned >= limit {
			break
		}
		err := k.IterateInquiriesByStatus(ctx, status, limit-scanned, func(q *types.Inquiry) bool {
			scanned++
			_ = k.tryResolveInquiry(ctx, q)
			return false
		})
		if err != nil {
			return err
		}
	}

	// 2. Frontier-invitation cycle (commitment 18).
	cadence := params.FrontierInvitationCadenceBlocks
	if cadence > 0 {
		height := currentBlock(ctx)
		if height%cadence == 0 {
			if err := k.runFrontierInvitationCycle(ctx); err != nil {
				// Surface the error at the logger but do not abort
				// the block — the resolution scan above already
				// committed its writes, and frontier sponsorship is
				// best-effort exploration funding, not a consensus-
				// critical hot path.
				k.Logger(ctx).Error("frontier-invitation cycle failed",
					"err", err,
					"height", height,
				)
			}
		}
	}
	return nil
}
