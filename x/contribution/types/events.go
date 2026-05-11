package types

// Event type names emitted by the x/contribution module. Names follow
// the `zerone.<module>.<action>` convention used elsewhere in the
// chain (e.g., x/creed, x/knowledge), so off-chain indexers can route
// by module prefix.
const (
	EventTypeContributionSubmitted            = "zerone.contribution.submitted"
	EventTypeContributionClassified           = "zerone.contribution.classified"
	EventTypeUsefulWorkAttested               = "zerone.contribution.useful_work_attested"
	EventTypeUsefulWorkSettled                = "zerone.contribution.useful_work_settled"
	EventTypeRecursionWeightComputed          = "zerone.contribution.recursion_weight_computed"
	EventTypeContributionAdmitted             = "zerone.contribution.admitted"
	EventTypeContributionRevoked              = "zerone.contribution.revoked"
	EventTypeContributionClassificationFailed = "zerone.contribution.classification_failed"
	EventTypeContributionVerificationFailed   = "zerone.contribution.verification_failed"
)

// Attribute keys used across events.
const (
	AttributeKeyID                   = "id"
	AttributeKeyClass                = "class"
	AttributeKeyPhase                = "phase"
	AttributeKeyContributor          = "contributor"
	AttributeKeySubstrateLinkBps     = "substrate_link_bps"
	AttributeKeyVerificationScoreBps = "verification_score_bps"
	AttributeKeyAdmittedAtBlock      = "admitted_at_block"
	AttributeKeyBackRef              = "back_ref"
	AttributeKeyFactRef              = "fact_ref"
	AttributeKeyDisproverArtifactID  = "disprover_artifact_id"
	AttributeKeyCascadeFlag          = "cascade_flag"
	AttributeKeyReason               = "reason"
	AttributeKeyMechanism            = "mechanism"
	AttributeKeyRewardShape          = "reward_uzrn_shape"
	AttributeKeyLBps                 = "L_bps"
	AttributeKeyWBps                 = "W_bps"
	AttributeKeyQBps                 = "Q_bps"
	AttributeKeyAxisSubstrate        = "axis_substrate"
	AttributeKeyAxisVerification     = "axis_verification"
	AttributeKeyAxisClassification   = "axis_classification"
	AttributeKeyAxisAttribution      = "axis_attribution"
	AttributeKeyAxisTooling          = "axis_tooling"
	AttributeKeyAxisInterface        = "axis_interface"
	AttributeKeyTotalWeight          = "total_weight"
	AttributeKeyCreedCommitment      = "creed_commitment"
	AttributeKeyUsefulWorkCommitment = "useful_work_commitment"
)

// Constant values for tagging events with commitments.
const (
	UsefulWorkCommitmentValue  = "UW"
	CommitmentIssuance         = "20" // truth-seeking commitment 20: issuance follows participation
	CascadeFlagRevokedAncestor = "provenance_revoked_ancestor"
	MechanismM2                = "M2"
	MechanismM3                = "M3"
	MechanismM4                = "M4"
	MechanismM5                = "M5"
)
