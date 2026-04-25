package keeper

import "context"

import "github.com/zerone-chain/zerone/x/inquiry/types"

// BeginBlocker scans OPEN and ANSWERED inquiries for resolution.
//
// For each inquiry up to params.begin_blocker_scan_limit:
//   - If any linked answer's claim has been accepted, resolve to
//     RESOLVED and pay the bounty to the answerer.
//   - Else, if expiry block reached, resolve to EXPIRED and refund
//     the asker.
//   - Otherwise leave untouched.
//
// The scan is bounded so a runaway pile of unresolved inquiries
// cannot consume unbounded gas. Older inquiries that fall outside
// the scan window can still be resolved via MsgResolveInquiry.
func (k Keeper) BeginBlocker(ctx context.Context) error {
	params := k.GetParams(ctx)
	limit := params.BeginBlockerScanLimit

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
	return nil
}
