package cross_stack_test

import (
	"crypto/sha256"
	"encoding/hex"

	knowledgetypes "github.com/zerone-chain/zerone/x/knowledge/types"
	sponsorshiptypes "github.com/zerone-chain/zerone/x/sponsorship/types"
)

func sponsorshipV2Digest(label string) string {
	sum := sha256.Sum256([]byte(label))
	return hex.EncodeToString(sum[:])
}

func sponsorshipV2WorkContract(workerAddress ...string) *sponsorshiptypes.WorkContract {
	assignedWorker := "zrn1v3jkzervd9hx2ttfdejx27pdwdcx7m3dwexsmf"
	if len(workerAddress) > 0 {
		assignedWorker = workerAddress[0]
	}
	return &sponsorshiptypes.WorkContract{
		WorkSpecHash: sponsorshipV2Digest("cross-stack/work-spec"), AcceptanceHash: sponsorshipV2Digest("cross-stack/acceptance"),
		InputRoot: sponsorshipV2Digest("cross-stack/input"), EnvironmentRoot: sponsorshipV2Digest("cross-stack/environment"),
		MinCorroborations: 0,
		WorkerAddress:     assignedWorker,
	}
}

func sponsorshipV2Fact(id, domain, submitter string, submittedAt uint64) *knowledgetypes.Fact {
	contract := sponsorshipV2WorkContract(submitter)
	commitment := &knowledgetypes.ComputationalCommitment{
		WorkSpecHash: contract.WorkSpecHash, AcceptanceHash: contract.AcceptanceHash,
		InputRoot: contract.InputRoot, EnvironmentRoot: contract.EnvironmentRoot,
		ArtifactRoot: sponsorshipV2Digest("artifact/" + id), EvidenceRoot: sponsorshipV2Digest("evidence/" + id),
	}
	commitment.WorkReceiptHash = knowledgetypes.ComputeWorkReceiptHash(commitment, submitter)
	return &knowledgetypes.Fact{
		Id: id, Domain: domain, Submitter: submitter, SubmittedAtBlock: submittedAt,
		VerifiedAtBlock: submittedAt, Status: knowledgetypes.FactStatus_FACT_STATUS_VERIFIED,
		ClaimType: knowledgetypes.ClaimType_CLAIM_TYPE_COMPUTATIONAL, MethodId: knowledgetypes.MethodologyComputational,
		ChallengeWindowEnd: submittedAt, ComputationalCommitment: commitment,
	}
}
