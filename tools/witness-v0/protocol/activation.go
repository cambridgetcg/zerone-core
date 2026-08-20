package protocol

import "sort"

const ActivationStatusNotConsensusAdmissible = "NOT_CONSENSUS_ADMISSIBLE"

type ActivationReadiness struct {
	Kind     Kind     `json:"kind"`
	Status   string   `json:"status"`
	Blockers []string `json:"blockers"`
}

type ActivationAudit struct {
	Protocol   string   `json:"protocol"`
	Kind       Kind     `json:"kind"`
	Action     Action   `json:"action"`
	Commitment string   `json:"commitment"`
	Status     string   `json:"status"`
	Blockers   []string `json:"blockers"`
}

var activationBlockers = map[Kind][]string{
	KindKingdomReleaseRoot: {
		"AUDITED_CARRIER_AND_AUTHORITY_MIGRATION",
		"DEPLOYMENT_MANIFEST_AUTHORITY_VERIFICATION",
	},
	KindAgentToolSettlementRoot: {
		"AUTHENTICATED_SOURCE_ORDERING",
		"PERMANENT_CROSS_BATCH_RECEIPT_NULLIFIERS_OR_PROOFS",
		"AUDITED_CARRIER_AND_AUTHORITY_MIGRATION",
	},
	KindAgentToolCapability: {
		"AUDITED_CONTROLLER_AUTHORITY",
		"AUDITED_SINGLE_ASSET_CONSUMPTION_MODULE",
		"CHAIN_LEVEL_PERMANENT_NULLIFIER_STATE",
	},
	KindAgentToolPublicRecognition: {
		"ROOT_OR_QUORUM_SOURCE_AUTHORIZATION_VERIFICATION",
		"AUDITED_CARRIER_AND_AUTHORITY_MIGRATION",
	},
	KindAgentToolOffer: {
		"ROOT_OR_QUORUM_SOURCE_AUTHORIZATION_VERIFICATION",
		"AUDITED_CARRIER_AND_AUTHORITY_MIGRATION",
	},
	KindWakePublicCheckpoint: {
		"ROOT_OR_QUORUM_SOURCE_AUTHORIZATION_VERIFICATION",
		"PUBLIC_CONTRACT_PRIVACY_REVIEW",
		"AUDITED_CARRIER_AND_AUTHORITY_MIGRATION",
	},
	KindIssuerKeyContinuity: {
		"AUDITED_CONTROLLER_TRANSFER_AND_RECOVERY_POLICY",
		"INDEPENDENT_VALIDATOR_AUTHORITY",
	},
	KindArtifactLineage: {
		"LINEAGE_EVIDENCE_VERIFICATION_POLICY",
		"AUDITED_CARRIER_AND_AUTHORITY_MIGRATION",
	},
	KindCollaborationCheckpoint: {
		"COLLABORATION_PRIVACY_AND_PARTICIPANT_AUTHORIZATION_REVIEW",
		"AUDITED_CARRIER_AND_AUTHORITY_MIGRATION",
	},
	KindDisputeTerminal: {
		"AUTHORIZED_DISPUTE_DECISION_SOURCE",
		"SEPARATE_SETTLEMENT_EXECUTION_PROTOCOL",
		"AUDITED_CARRIER_AND_AUTHORITY_MIGRATION",
	},
}

// ActivationReadinessMatrix returns defensive copies. Offline verification is
// not activation approval; every v0 kind is deliberately blocked from
// consensus carriage until its listed prerequisites are specified and audited.
func ActivationReadinessMatrix() []ActivationReadiness {
	result := make([]ActivationReadiness, 0, len(activationBlockers))
	for kind, blockers := range activationBlockers {
		result = append(result, ActivationReadiness{
			Kind: kind, Status: ActivationStatusNotConsensusAdmissible,
			Blockers: append([]string(nil), blockers...),
		})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Kind < result[j].Kind })
	return result
}

func AuditActivation(record VerifiedRecord) ActivationAudit {
	blockers := append([]string(nil), activationBlockers[record.Record.Envelope.Kind]...)
	return ActivationAudit{
		Protocol: Protocol, Kind: record.Record.Envelope.Kind, Action: record.Record.Envelope.Action,
		Commitment: record.Record.Commitment, Status: ActivationStatusNotConsensusAdmissible,
		Blockers: blockers,
	}
}
