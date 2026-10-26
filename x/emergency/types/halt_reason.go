package types

// HaltReasonResearchFund is the reserved reason for research-fund misuse.
// Free text remains valid for halt proposals, but is unclassified evidence:
// after activation it blocks forward phase progress and cannot justify rollback.
const HaltReasonResearchFund = "research_fund"
