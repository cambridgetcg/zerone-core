package keeper

const (
	// Event types.
	EventTypeExternalAttestationSubmitted = "external_attestation_submitted"
	EventTypeAdapterRegistered            = "adapter_registered"
	EventTypeAdapterSuspended             = "adapter_suspended"
	EventTypeAdapterTombstoned            = "adapter_tombstoned"

	// Attributes.
	AttrUsefulWorkCommitment = "useful_work_commitment" // value: "UW"
	AttrMechanism            = "mechanism"              // value: "M1" | "M2,M3" | etc.
)
