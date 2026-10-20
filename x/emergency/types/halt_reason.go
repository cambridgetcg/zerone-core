package types

import "errors"

// HaltReasonResearchFund is the reserved reason for research-fund misuse.
// Free text remains valid for halt proposals, but is unclassified evidence:
// after activation it blocks forward phase progress and cannot justify rollback.
const HaltReasonResearchFund = "research_fund"

// ErrHaltEvidence denotes unclassified or malformed in-window evidence, not an
// activation marker fault. It must not halt block processing.
var ErrHaltEvidence = errors.New("research-fund halt evidence is unknown")
