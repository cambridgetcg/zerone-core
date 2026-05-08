package types

import "fmt"

// DefaultParams returns the default inquiry module parameters.
//
// These values express commitment 16 (chain pays for exploration of
// the unknown). The defaults make inquiry economically viable: low
// enough min_bounty that any agent can afford to ask, sane expiry
// windows so bounties are not locked indefinitely, and a per-inquiry
// answer cap that bounds griefing.
func DefaultParams() *Params {
	return &Params{
		// 1 ZRN minimum bounty. Above zero so spam is costly;
		// modest enough that asking is accessible.
		MinBounty: "1000000",

		// Question and context size limits. Inquiries should be
		// concrete; 4 KB / 8 KB are paragraph-scale.
		MaxQuestionBytes: 4096,
		MaxContextBytes:  8192,

		// Default expiry: ~3 days at 2.5s blocks. Asker can specify
		// a shorter or longer window up to max.
		DefaultExpiryBlocks: 100_000,

		// Maximum expiry: ~30 days. Past this point a bounty is
		// considered abandoned; bounties cannot be locked
		// indefinitely.
		MaxExpiryBlocks: 1_000_000,

		// 32 answers per inquiry caps griefing. The chain doesn't
		// need to evaluate 1000 answers to one question — the first
		// accepted wins.
		MaxAnswersPerInquiry: 32,

		// 100 inquiries per BeginBlocker scan. Older un-resolved
		// inquiries can still be resolved manually via
		// MsgResolveInquiry.
		BeginBlockerScanLimit: 100,

		SubmissionsEnabled: true,

		// ── Commitment 18: chain-manufactured exploration demand ──
		//
		// 600-block cadence ≈ 25 minutes at 2.5s blocks. Frequent
		// enough that the frontier voice is actually heard; slow
		// enough that one cadence-tick's worth of work stays
		// finite and bounded.
		FrontierInvitationCadenceBlocks: 600,
		// Three system-sponsored inquiries per tick caps the chain's
		// exploration spend at FrontierInvitationBounty × 3 per
		// cadence. Larger K means broader coverage; smaller K means
		// concentrated demand on the very sparsest domains. 3 is
		// the smallest value that still covers a meaningful breadth.
		FrontierInvitationTopK: 3,
		// 700_000 BPS = top tercile of the 0..1M sparsity score.
		// A domain must clear this bar to attract chain sponsorship —
		// the chain refuses to spend its exploration budget on
		// already-mapped territory.
		FrontierInvitationSparsityThresholdBps: 700_000,
		// 5 ZRN per system-sponsored inquiry. Five times min_bounty
		// so frontier questions attract attention without distorting
		// the wider bounty market. The choice expresses a value: the
		// chain pays meaningfully more for exploration than the
		// minimum it requires of asker-driven inquiries.
		FrontierInvitationBounty: "5000000",
		// 600_000 blocks ≈ 17 days. System inquiries sit longer than
		// user-asked ones because the corpus does not need to give
		// up on a sparse domain after 3 days; persistence IS part of
		// commitment 18.
		FrontierInvitationExpiryBlocks: 600_000,
	}
}

// DefaultGenesis returns the default genesis state.
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Params:         DefaultParams(),
		Inquiries:      []*Inquiry{},
		Answers:        []*Answer{},
		NextInquirySeq: 1,
		NextAnswerId:   1,
	}
}

func (gs *GenesisState) Validate() error {
	if gs.Params == nil {
		return fmt.Errorf("params required")
	}
	if err := gs.Params.Validate(); err != nil {
		return err
	}
	seenInq := map[string]bool{}
	for _, q := range gs.Inquiries {
		if q == nil {
			continue
		}
		if seenInq[q.Id] {
			return fmt.Errorf("duplicate inquiry id: %s", q.Id)
		}
		seenInq[q.Id] = true
	}
	highest := uint64(0)
	for _, a := range gs.Answers {
		if a == nil {
			continue
		}
		if !seenInq[a.InquiryId] {
			return fmt.Errorf("answer references unknown inquiry %s", a.InquiryId)
		}
		if a.Id > highest {
			highest = a.Id
		}
	}
	if gs.NextAnswerId != 0 && gs.NextAnswerId <= highest {
		return fmt.Errorf("next_answer_id (%d) must be > highest (%d)", gs.NextAnswerId, highest)
	}
	return nil
}

func (p *Params) Validate() error {
	if _, err := ParseBounty(p.MinBounty); err != nil {
		return fmt.Errorf("invalid min_bounty: %w", err)
	}
	if p.MaxQuestionBytes == 0 {
		return fmt.Errorf("max_question_bytes must be > 0")
	}
	if p.MaxContextBytes == 0 {
		return fmt.Errorf("max_context_bytes must be > 0")
	}
	if p.DefaultExpiryBlocks == 0 {
		return fmt.Errorf("default_expiry_blocks must be > 0")
	}
	if p.MaxExpiryBlocks == 0 || p.MaxExpiryBlocks < p.DefaultExpiryBlocks {
		return fmt.Errorf("max_expiry_blocks must be >= default_expiry_blocks")
	}
	if p.MaxAnswersPerInquiry == 0 {
		return fmt.Errorf("max_answers_per_inquiry must be > 0")
	}
	if p.BeginBlockerScanLimit == 0 {
		return fmt.Errorf("begin_blocker_scan_limit must be > 0")
	}

	// Commitment 18 params. Validate only when the path is enabled
	// (cadence > 0) so governance can disable frontier sponsorship
	// without forcing the rest of the params into a particular shape.
	if p.FrontierInvitationCadenceBlocks > 0 {
		if p.FrontierInvitationTopK == 0 {
			return fmt.Errorf("frontier_invitation_top_k must be > 0 when cadence enabled")
		}
		if p.FrontierInvitationSparsityThresholdBps > 1_000_000 {
			return fmt.Errorf("frontier_invitation_sparsity_threshold_bps must be <= 1_000_000")
		}
		if _, err := ParseBounty(p.FrontierInvitationBounty); err != nil {
			return fmt.Errorf("invalid frontier_invitation_bounty: %w", err)
		}
		if p.FrontierInvitationExpiryBlocks == 0 {
			return fmt.Errorf("frontier_invitation_expiry_blocks must be > 0 when cadence enabled")
		}
		if p.FrontierInvitationExpiryBlocks > p.MaxExpiryBlocks {
			return fmt.Errorf("frontier_invitation_expiry_blocks (%d) must be <= max_expiry_blocks (%d)",
				p.FrontierInvitationExpiryBlocks, p.MaxExpiryBlocks)
		}
	}
	return nil
}
