//go:build ignore

// Run from the repository root:
// go run -mod=readonly sdk/typescript/tests/fixtures/claim-history-response.go
// The fixture is synthetic and contains no account credentials or live records.
package main

import (
	"encoding/hex"
	"encoding/json"
	"os"

	"google.golang.org/protobuf/proto"

	"github.com/zerone-chain/zerone/x/knowledge/types"
)

func main() {
	const height = uint64(9007199254740993)
	edge := &types.FactRelation{SourceFactId: "fact-root", TargetFactId: "fact-neighbor", Relation: types.RelationType_RELATION_TYPE_SUPPORTS, CreatedAtBlock: height - 1, Creator: "test-author", Inference: types.InferenceType_INFERENCE_TYPE_DEDUCTIVE, InferenceStrengthBps: 654321, MethodId: "test-method"}
	response := &types.QueryClaimHistoryResponse{
		ChainId: "zerone-local-history", BlockHeight: height,
		Record: &types.ClaimHistoryRecord{
			ClaimId:         "claim/root",
			Claim:           &types.Claim{Id: "claim/root", FactContent: "Synthetic date-parser claim", Submitter: "test-author", ReasoningTrace: "Checked leap-year boundary", ArgumentText: "Reason retained verbatim.", EvidenceIds: []string{"test-evidence"}},
			Rounds:          []*types.VerificationRound{{Id: "round-root", ClaimId: "claim/root", StartedAtBlock: height - 2, Reveals: []*types.RevealEntry{{Verifier: "test-reviewer", Vote: "reject", Salt: []byte{1, 2, 3}, Confidence: 765432, Attestation: &types.ReviewAttestation{MethodId: "test-method", Reason: "February 30 is invalid.", Scope: "Calendar validity", EvidenceIds: []string{"test-counterexample"}}}}}},
			Facts:           []*types.ClaimHistoryFact{{Fact: &types.Fact{Id: "fact-root", ClaimId: "claim/root", Content: "Retained derived fact", VerifiedAtBlock: height - 1}, OutgoingRelations: []*types.FactRelation{edge}}},
			MissingRoundIds: []string{"historical-missing-round"},
		},
		RelatedClaims: []*types.RelatedClaimHistory{{Record: &types.ClaimHistoryRecord{ClaimId: "claim-contradiction", Claim: &types.Claim{Id: "claim-contradiction", ArgumentText: "Contradiction reason", EvidenceIds: []string{"test-counterexample"}, Relations: []*types.ClaimRelation{{TargetFactId: "fact-root", Relation: types.RelationType_RELATION_TYPE_CONTRADICTS}}}}, Links: []*types.ClaimHistoryLink{{Field: "relations.contradicts", TargetId: "fact-root"}}}},
	}
	bz, err := proto.MarshalOptions{Deterministic: true}.Marshal(response)
	if err != nil {
		panic(err)
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(map[string]string{"generator": "claim-history-response.go; google.golang.org/protobuf/proto deterministic encoding", "chainId": response.ChainId, "claimId": response.Record.ClaimId, "blockHeight": "9007199254740993", "protobufHex": hex.EncodeToString(bz)}); err != nil {
		panic(err)
	}
}
