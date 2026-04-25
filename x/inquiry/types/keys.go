package types

const (
	ModuleName = "inquiry"
	StoreKey   = ModuleName

	// BountyPoolModuleName is the module account that escrows
	// inquiry bounties between submission and resolution. Funded
	// by askers, drained by winning answerers (or refunded to
	// askers on expiry/cancel).
	BountyPoolModuleName = "inquiry_bounty_pool"
)

var (
	ParamsKey            = []byte{0x00}
	InquiryKeyPrefix     = []byte{0x01} // id → Inquiry
	AnswerKeyPrefix      = []byte{0x02} // id (uint64 BE) → Answer
	NextInquirySeqKey    = []byte{0x03}
	NextAnswerIDKey      = []byte{0x04}

	// Indexes.
	ByDomainPrefix       = []byte{0x10} // domain/inquiry_id → 1
	ByAskerPrefix        = []byte{0x11} // asker/inquiry_id → 1
	ByStatusPrefix       = []byte{0x12} // status_byte/inquiry_id → 1 (for BeginBlocker scan)
	AnswersByInquiryPrefix = []byte{0x13} // inquiry_id/answer_id → 1
	AnswersByClaimPrefix = []byte{0x14}   // claim_id → answer_id (one answer per claim)
)
