package types

import "cosmossdk.io/errors"

const codespace = ModuleName

var (
	// Adapter errors (M3).
	ErrAdapterNotFound      = errors.Register(codespace, 1, "adapter not found (UW + M3: adapter must be gov-registered before use)")
	ErrAdapterNotActive     = errors.Register(codespace, 2, "adapter not in ACTIVE status (UW + M3)")
	ErrAdapterAlreadyExists = errors.Register(codespace, 3, "adapter id already registered (UW + M3: forward-only registry)")
	ErrAdapterTombstoned    = errors.Register(codespace, 4, "adapter is tombstoned; id cannot be reused (UW + M3 + commitment 10)")
	ErrAdapterAuthority     = errors.Register(codespace, 5, "adapter mutation requires gov authority (UW + M3)")

	// Submission errors (M1, M2, M3, M5).
	ErrInsufficientQualification = errors.Register(codespace, 10, "submitter lacks required qualification (UW + M3)")
	ErrInsufficientBond          = errors.Register(codespace, 11, "bond below adapter or chain minimum (UW + M1)")
	ErrWorkClassNotAllowed       = errors.Register(codespace, 12, "adapter does not permit this work class (UW + M3)")
	ErrCitedFactNotFound         = errors.Register(codespace, 13, "substrate-link cites non-existent fact_id (UW + M2: cited_facts must exist at commit)")
	ErrTooManyPendingClaims      = errors.Register(codespace, 14, "pending_claims count exceeds max_pending_claims_per_attestation (UW + M2)")
	ErrAxisOverflow              = errors.Register(codespace, 15, "axis_projection exceeds adapter AxisBounds (UW + M5)")
	ErrLinkHashMismatch          = errors.Register(codespace, 16, "substrate-link hash does not match recomputed canonical form (UW + M2: re-derivability is the link)")
	ErrInvalidCitationType       = errors.Register(codespace, 17, "citation_type unspecified or unknown (UW + M6)")
	ErrContributionSharesInvalid = errors.Register(codespace, 18, "contribution_share_bps does not sum to 10000 across cites (UW + M6)")
	ErrPendingClaimsNotSupported = errors.Register(codespace, 19, "pending_claims are not accepted yet: translation into x/knowledge is unwired (ToK Plan 4), so an accepted pending claim could never resolve and the bond would slash on timeout — submit cited_facts only")
	ErrSourceRequired            = errors.Register(codespace, 22, "substrate-link source with a non-empty source_id is required (UW + M2: the source reference is the identity of the work)")
	ErrDuplicateSource           = errors.Register(codespace, 23, "source already attested for this adapter (UW + M1: one work, one attestation — replay is closed on-chain)")
	ErrAdapterIdMismatch         = errors.Register(codespace, 24, "msg.adapter_id and link.adapter_id differ — one submission, one adapter")
	ErrDedupeNotArmed            = errors.Register(codespace, 25, "external-attestation dedupe is not armed on this node: the substrate-dedupe-v1 migration has not run (plan-less deploy?) — submissions are refused fail-closed rather than risk a replay-mint until the source-ref index is seeded")
	ErrAdapterAxisBoundsUnset    = errors.Register(codespace, 26, "recursion_weight claimed against an adapter that declares no axis_bounds: weight multiplies the settlement reward, so an unbounded adapter accepts no weighted claim at all (UW + M5) — register bounds before claiming weight")
	ErrTooManyCitedFacts         = errors.Register(codespace, 27, "cited_facts count exceeds MaxCitedFactsPerLink (UW + M2: each cite is a knowledge read at admission and a settlement-time event in the BeginBlocker drain — bounded, reject-not-truncate)")
	ErrSourceAdapterIdMismatch   = errors.Register(codespace, 28, "source.adapter_id and link.adapter_id differ — declared provenance must be bound to the submitted adapter")
	ErrInvalidSourceContentHash  = errors.Register(codespace, 29, "source.content_hash must be exactly 32 bytes — consensus validates declared SHA-256 width, not external content")

	// State machine errors.
	ErrAttestationNotFound    = errors.Register(codespace, 20, "attestation not found")
	ErrAttestationWrongStatus = errors.Register(codespace, 21, "attestation status does not permit this transition")

	// Lineage errors (M6).
	ErrLineageCycle            = errors.Register(codespace, 30, "lineage cycle: upstream.created_at_block >= downstream.created_at_block (UW + M6)")
	ErrSelfCitationCapExceeded = errors.Register(codespace, 31, "self-citation contribution_share_bps exceeds self_citation_cap_bps (UW + M6)")
)
