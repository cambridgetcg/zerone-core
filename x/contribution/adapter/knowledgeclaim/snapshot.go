package knowledgeclaim

import (
	"crypto/sha256"
	"encoding/binary"

	contribtypes "github.com/zerone-chain/zerone/x/contribution/types"
	knowledgetypes "github.com/zerone-chain/zerone/x/knowledge/types"
)

// BuildContributionFromSnapshot constructs a Contribution mirror from
// a ClaimSnapshot. The resulting Contribution is in STATUS_SUBMITTED;
// the caller (KnowledgeHooksAdapter) is responsible for running
// Classify+SubstrateLink and transitioning to CLASSIFIED.
//
// The id is derived from class+phase+claim_id+contributor for stability:
// re-mirroring the same claim produces the same id, supporting idempotent
// hook invocations.
func BuildContributionFromSnapshot(claimID string, snap knowledgetypes.ClaimSnapshot, blockHeight int64) *contribtypes.Contribution {
	return &contribtypes.Contribution{
		Id:              computeMirrorID(claimID, snap.Submitter),
		Contributor:     snap.Submitter,
		Class:           contribtypes.ContributionClass_KNOWLEDGE_CLAIM,
		Phase:           contribtypes.LifecyclePhase_PHASE_KNOWLEDGE,
		ManifestCid:     snap.TokManifestCID,
		Status:          contribtypes.ContributionStatus_STATUS_SUBMITTED,
		ClaimsAboutSelf: snap.MethodologyTrace, // Treat methodology trace as testable claims-about-self at Phase 1.
		CreatedAtBlock:  uint64(blockHeight),
		BackRef:         claimID,
		Payload: &contribtypes.ContributionPayload{
			Payload: &contribtypes.ContributionPayload_Knowledge{
				Knowledge: &contribtypes.KnowledgeClaim{
					ClaimId:          claimID,
					Domain:           snap.Domain,
					StatementHash:    snap.StatementHash,
					MethodologyTrace: snap.MethodologyTrace,
					AxiomRefs:        snap.AxiomRefs,
					TokManifestCid:   snap.TokManifestCID,
				},
			},
		},
		Recursion: &contribtypes.RecursionImpact{
			Type:          contribtypes.RecursionType_RECURSION_TYPE_NONE,
			MultiplierBps: 10_000, // 1× at Phase 1
			Revocable:     true,
			Axes:          &contribtypes.RecursionAxisScores{},
		},
	}
}

func computeMirrorID(claimID, contributor string) []byte {
	h := sha256.New()
	classBz := make([]byte, 4)
	binary.BigEndian.PutUint32(classBz, uint32(contribtypes.ContributionClass_KNOWLEDGE_CLAIM))
	h.Write(classBz)
	phaseBz := make([]byte, 4)
	binary.BigEndian.PutUint32(phaseBz, uint32(contribtypes.LifecyclePhase_PHASE_KNOWLEDGE))
	h.Write(phaseBz)
	h.Write([]byte(claimID))
	h.Write([]byte(contributor))
	out := h.Sum(nil)
	return out[:]
}
