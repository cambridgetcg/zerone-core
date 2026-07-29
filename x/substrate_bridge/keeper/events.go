package keeper

const (
	// Event types.
	EventTypeExternalAttestationSubmitted = "external_attestation_submitted"
	EventTypeExternalAttestationCommitted = "external_attestation_committed"
	EventTypeExternalAttestationSettled   = "external_attestation_settled"
	EventTypeExternalAttestationRejected  = "external_attestation_rejected"
	EventTypeExternalAttestationPartial   = "external_attestation_partial"

	EventTypeAdapterRegistered = "adapter_registered"
	EventTypeAdapterSuspended  = "adapter_suspended"
	EventTypeAdapterTombstoned = "adapter_tombstoned"

	EventTypeLineageEdgeCreated    = "lineage_edge_created"
	EventTypeLineageRoyaltyAccrued = "lineage_royalty_accrued"

	// EventTypeLineageRoyaltyPaid is retained as a deprecated Go symbol for
	// source compatibility. Runtime emits EventTypeLineageRoyaltyAccrued;
	// no coin payment occurs in the current lineage accountant.
	EventTypeLineageRoyaltyPaid = EventTypeLineageRoyaltyAccrued

	EventTypeWitnessRewardEscrowed  = "witness_reward_escrowed"
	EventTypeWitnessRewardReleased  = "witness_reward_released"
	EventTypeWitnessRewardCancelled = "witness_reward_cancelled"

	// Attributes.
	AttrUsefulWorkCommitment = "useful_work_commitment" // value: "UW"
	AttrMechanism            = "mechanism"              // value: "M1" | "M2,M3" | etc.
	AttrAttestationID        = "attestation_id"
	AttrRewardUzrn           = "reward_uzrn" // amount actually minted and paid (cap-clip honest)
	AttrAdapterID            = "adapter_id"
	AttrRecipient            = "recipient"
	AttrDeadline             = "deadline" // block height the challenge window closes
	AttrReason               = "reason"
)
