package cross_stack_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	researchtypes "github.com/zerone-chain/zerone/x/research/types"
	emtypes "github.com/zerone-chain/zerone/x/evidence_mgmt/types"
	disputestypes "github.com/zerone-chain/zerone/x/disputes/types"
)

// TestScenario3_ResearchPeerReviewFlow verifies the research submission and
// peer review lifecycle: submit → review → status progression.
func TestScenario3_ResearchPeerReviewFlow(t *testing.T) {
	h := NewTestHarness(t)

	submitter := testAddr("researcher_1")

	// 1. Create a research submission.
	research := &researchtypes.Research{
		Id:           "research-001",
		Submitter:    submitter.String(),
		Title:        "Replication Study: Agent Consensus Accuracy",
		Description:  "Replicating consensus accuracy claims from fact-42",
		Domain:       "consensus",
		ResearchType: string(researchtypes.ResearchTypeReplication),
		TargetFactId: "fact-42",
		Stake:        "1000000",
		Status:       string(researchtypes.ResearchStatusSubmitted),
		ReviewCount:  0,
		ApproveCount: 0,
		RejectCount:  0,
		CreatedAt:    uint64(h.Height()),
		UpdatedAt:    uint64(h.Height()),
	}
	h.ResearchKeeper.SetResearch(h.Ctx, research)

	// 2. Verify submission is stored and retrievable.
	retrieved, found := h.ResearchKeeper.GetResearch(h.Ctx, "research-001")
	require.True(t, found, "research must be retrievable")
	require.Equal(t, string(researchtypes.ResearchStatusSubmitted), retrieved.Status)
	require.Equal(t, submitter.String(), retrieved.Submitter)

	// 3. Create peer reviews with APPROVE verdict.
	for i := 1; i <= 3; i++ {
		reviewer := testAddr(fmt.Sprintf("reviewer_%d", i))
		review := &researchtypes.PeerReview{
			Id:         fmt.Sprintf("review-%d", i),
			ResearchId: "research-001",
			Reviewer:   reviewer.String(),
			Verdict:    "approve",
			Score:      80,
			Reasoning:  "Methodology is sound, results replicate.",
			ReviewedAt: uint64(h.Height()),
		}
		h.ResearchKeeper.SetPeerReview(h.Ctx, review)
	}

	// 4. Verify reviews are stored.
	reviews := h.ResearchKeeper.GetReviewsForResearch(h.Ctx, "research-001")
	require.Len(t, reviews, 3, "all 3 reviews must be stored")

	// 5. Verify HasReviewerReviewed.
	reviewer1 := testAddr("reviewer_1")
	require.True(t, h.ResearchKeeper.HasReviewerReviewed(h.Ctx, "research-001", reviewer1.String()))

	nonReviewer := testAddr("non_reviewer")
	require.False(t, h.ResearchKeeper.HasReviewerReviewed(h.Ctx, "research-001", nonReviewer.String()))

	// 6. Progress research to accepted (simulated — in production, msg_server
	// would tally votes and transition status).
	retrieved.Status = string(researchtypes.ResearchStatusAccepted)
	retrieved.ReviewCount = 3
	retrieved.ApproveCount = 3
	retrieved.AggregateScore = 80
	retrieved.UpdatedAt = uint64(h.Height())
	h.ResearchKeeper.SetResearch(h.Ctx, retrieved)

	accepted, found := h.ResearchKeeper.GetResearch(h.Ctx, "research-001")
	require.True(t, found)
	require.Equal(t, string(researchtypes.ResearchStatusAccepted), accepted.Status)
	require.Equal(t, uint32(3), accepted.ApproveCount)

	// 7. Verify filtering by status.
	byStatus := h.ResearchKeeper.GetResearchesByStatus(h.Ctx, researchtypes.ResearchStatusAccepted)
	require.Len(t, byStatus, 1)
	require.Equal(t, "research-001", byStatus[0].Id)
}

// TestScenario5_EvidenceDisputesChain verifies the evidence submission and
// dispute resolution lifecycle.
func TestScenario5_EvidenceDisputesChain(t *testing.T) {
	h := NewTestHarness(t)

	submitter := testAddr("evidence_submitter")
	challenger := testAddr("evidence_challenger")

	// 1. Submit evidence.
	evidence := &emtypes.Evidence{
		Id:        "ev-001",
		Submitter: submitter.String(),
		EvidenceType: emtypes.EvidenceType_EVIDENCE_TYPE_DOCUMENT,
		ContentHash:  "sha256:abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
		Metadata:     `{"source": "research-lab-1", "format": "pdf"}`,
		ChainOfCustody: []*emtypes.ChainOfCustodyEntry{
			{
				Custodian: submitter.String(),
				Action:    "submit",
				Timestamp: uint64(h.Height()),
				Notes:     "Initial submission",
			},
		},
		Status:         emtypes.EvidenceStatus_EVIDENCE_STATUS_SUBMITTED,
		CreatedAtBlock: uint64(h.Height()),
		UpdatedAtBlock: uint64(h.Height()),
	}
	h.EvidenceMgmtKeeper.SetEvidence(h.Ctx, evidence)

	// 2. Verify evidence is stored and retrievable.
	retrieved, found := h.EvidenceMgmtKeeper.GetEvidence(h.Ctx, "ev-001")
	require.True(t, found, "evidence must be retrievable")
	require.Equal(t, emtypes.EvidenceStatus_EVIDENCE_STATUS_SUBMITTED, retrieved.Status)
	require.Len(t, retrieved.ChainOfCustody, 1)

	// 3. Verify evidence by submitter.
	bySubmitter := h.EvidenceMgmtKeeper.GetEvidenceBySubmitter(h.Ctx, submitter.String())
	require.Len(t, bySubmitter, 1)
	require.Equal(t, "ev-001", bySubmitter[0].Id)

	// 4. Challenge evidence — create a dispute.
	dispute := &disputestypes.Dispute{
		Id:         "dispute-001",
		TargetId:   "ev-001",
		TargetType: disputestypes.DisputeTargetType_DISPUTE_TARGET_TYPE_EVIDENCE,
		Challenger: challenger.String(),
		Defender:   submitter.String(),
		Reason:     "Content hash does not match claimed document",
		Bond:       "1000000",
		Tier:       1,
		Phase:      disputestypes.DisputePhase_DISPUTE_PHASE_EVIDENCE_COMMIT,
		CreatedAt:  uint64(h.Height()),
	}
	h.DisputesKeeper.SetDispute(h.Ctx, dispute)

	// 5. Verify dispute is stored.
	retrievedDispute, found := h.DisputesKeeper.GetDispute(h.Ctx, "dispute-001")
	require.True(t, found, "dispute must be retrievable")
	require.Equal(t, "ev-001", retrievedDispute.TargetId)
	require.Equal(t, disputestypes.DisputeTargetType_DISPUTE_TARGET_TYPE_EVIDENCE, retrievedDispute.TargetType)

	// 6. Verify disputes by target.
	byTarget := h.DisputesKeeper.GetDisputesByTarget(h.Ctx, "ev-001")
	require.Len(t, byTarget, 1)

	// 7. Settle the dispute (simulated resolution).
	retrievedDispute.Phase = disputestypes.DisputePhase_DISPUTE_PHASE_SETTLED
	retrievedDispute.Outcome = disputestypes.DisputeOutcome_DISPUTE_OUTCOME_CHALLENGER_WINS
	h.DisputesKeeper.SetDispute(h.Ctx, retrievedDispute)

	// 8. Update evidence status based on dispute outcome.
	retrieved.Status = emtypes.EvidenceStatus_EVIDENCE_STATUS_CHALLENGED
	retrieved.ChainOfCustody = append(retrieved.ChainOfCustody, &emtypes.ChainOfCustodyEntry{
		Custodian: challenger.String(),
		Action:    "challenge",
		Timestamp: uint64(h.Height()),
		Notes:     "Dispute dispute-001 settled: challenger wins",
	})
	retrieved.UpdatedAtBlock = uint64(h.Height())
	h.EvidenceMgmtKeeper.SetEvidence(h.Ctx, retrieved)

	// 9. Verify final states.
	finalEvidence, found := h.EvidenceMgmtKeeper.GetEvidence(h.Ctx, "ev-001")
	require.True(t, found)
	require.Equal(t, emtypes.EvidenceStatus_EVIDENCE_STATUS_CHALLENGED, finalEvidence.Status)
	require.Len(t, finalEvidence.ChainOfCustody, 2)

	finalDispute, found := h.DisputesKeeper.GetDispute(h.Ctx, "dispute-001")
	require.True(t, found)
	require.Equal(t, disputestypes.DisputePhase_DISPUTE_PHASE_SETTLED, finalDispute.Phase)
	require.Equal(t, disputestypes.DisputeOutcome_DISPUTE_OUTCOME_CHALLENGER_WINS, finalDispute.Outcome)
}
