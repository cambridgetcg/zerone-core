package types_test

import (
	"encoding/hex"
	"encoding/json"
	"os"
	"strings"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"github.com/zerone-chain/zerone/x/knowledge/types"
	"google.golang.org/protobuf/proto"
)

func TestReviewCommitmentV2CrossLanguageVector(t *testing.T) {
	sdk.GetConfig().SetBech32PrefixForAccount("zrn", "zrnpub")
	var v struct {
		ChainID     string `json:"chainId"`
		RoundID     string `json:"roundId"`
		Verifier    string
		Vote        string
		SaltHex     string
		SHA256      string
		Attestation *types.ReviewAttestation
	}
	raw, err := os.ReadFile("../../../sdk/typescript/tests/fixtures/review-commitment-v2.json")
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(raw, &v))
	salt, err := hex.DecodeString(v.SaltHex)
	require.NoError(t, err)
	// encoding/json does not translate proto snake_case tags from camelCase.
	var wire struct {
		Attestation struct {
			MethodID    string `json:"methodId"`
			Reason      string
			Scope       string
			EvidenceIDs []string `json:"evidenceIds"`
		}
	}
	require.NoError(t, json.Unmarshal(raw, &wire))
	att := &types.ReviewAttestation{MethodId: wire.Attestation.MethodID, Reason: wire.Attestation.Reason, Scope: wire.Attestation.Scope, EvidenceIds: wire.Attestation.EvidenceIDs}
	hash, err := types.ComputeReviewCommitmentV2(v.ChainID, v.RoundID, v.Verifier, v.Vote, 800000, salt, att)
	require.NoError(t, err)
	require.Equal(t, v.SHA256, hex.EncodeToString(hash))
	for _, name := range []string{"chain", "round", "reviewer", "vote", "confidence", "salt", "method", "reason", "scope", "evidence", "order"} {
		t.Run(name, func(t *testing.T) {
			chain, round, reviewer, vote, confidence := v.ChainID, v.RoundID, v.Verifier, v.Vote, uint64(800000)
			s := append([]byte(nil), salt...)
			a := proto.Clone(att).(*types.ReviewAttestation)
			switch name {
			case "chain":
				chain += "-other"
			case "round":
				round += "-other"
			case "reviewer":
				reviewer = sdk.AccAddress([]byte("other-public-address")).String()
			case "vote":
				vote = "reject"
			case "confidence":
				confidence--
			case "salt":
				s[0]++
			case "method":
				a.MethodId += "-other"
			case "reason":
				a.Reason += " "
			case "scope":
				a.Scope += " "
			case "evidence":
				a.EvidenceIds[0] += "-other"
			case "order":
				a.EvidenceIds[0], a.EvidenceIds[1] = a.EvidenceIds[1], a.EvidenceIds[0]
			}
			changed, err := types.ComputeReviewCommitmentV2(chain, round, reviewer, vote, confidence, s, a)
			require.NoError(t, err)
			require.NotEqual(t, hash, changed)
		})
	}
}

func TestReviewCommitmentV2RejectsUnboundOrMalformedFields(t *testing.T) {
	sdk.GetConfig().SetBech32PrefixForAccount("zrn", "zrnpub")
	for _, name := range []string{"missing", "blank", "unknown", "reason-limit", "scope-limit", "evidence-duplicate", "evidence-limit", "salt-short", "confidence", "address", "unknown-scheme"} {
		t.Run(name, func(t *testing.T) {
			a := &types.ReviewAttestation{Reason: "I checked the cited fixture."}
			salt := make([]byte, 16)
			confidence := uint64(800000)
			address := sdk.AccAddress([]byte("public-fixture-user!")).String()
			switch name {
			case "missing":
				a = nil
			case "blank":
				a.Reason = "\u0085 \t"
			case "unknown":
				a.ProtoReflect().SetUnknown([]byte{0x28, 1})
			case "reason-limit":
				a.Reason = strings.Repeat("a", 4097)
			case "scope-limit":
				a.Scope = strings.Repeat("a", 1025)
			case "evidence-duplicate":
				a.EvidenceIds = []string{"same", "same"}
			case "evidence-limit":
				a.EvidenceIds = make([]string, 17)
			case "salt-short":
				salt = make([]byte, 15)
			case "confidence":
				confidence = 1000001
			case "address":
				address = strings.ToUpper(address)
			case "unknown-scheme":
				require.Error(t, types.ValidateVerificationRoundRecord(&types.VerificationRound{Id: "r", ClaimId: "c", Phase: types.VerificationPhase_VERIFICATION_PHASE_COMMIT, CommitmentScheme: 1}, true))
				return
			}
			_, err := types.ComputeReviewCommitmentV2("chain", "round", address, "accept", confidence, salt, a)
			require.Error(t, err)
		})
	}
}
