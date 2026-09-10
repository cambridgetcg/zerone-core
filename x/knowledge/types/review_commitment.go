package types

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"strings"
	"unicode/utf8"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	CommitmentSchemeLegacy    uint32 = 0
	CommitmentSchemeReviewV2  uint32 = 2
	MaxReviewReasonBytes             = 4096
	MaxReviewScopeBytes              = 1024
	MaxMethodIDBytes                 = 128
	MaxEvidenceReferences            = 16
	MaxEvidenceReferenceBytes        = 256
	MaxReviewAttestationBytes        = 8192
	MaxClaimReasoningBytes           = 8192
	MaxClaimReferences               = 64
	MaxCommitmentContextBytes        = 256
)

// ValidateRecordText bounds exact UTF-8 bytes without rewriting signed text.
func ValidateRecordText(name, value string, maxBytes int, required bool) error {
	if !utf8.ValidString(value) || len(value) > maxBytes {
		return fmt.Errorf("%s must be valid UTF-8 of at most %d bytes", name, maxBytes)
	}
	if required && strings.TrimSpace(value) == "" {
		return fmt.Errorf("%s must state a nonempty value", name)
	}
	return nil
}

func ValidateEvidenceReferences(refs []string) error {
	if len(refs) > MaxEvidenceReferences {
		return fmt.Errorf("at most %d evidence references are allowed", MaxEvidenceReferences)
	}
	seen := make(map[string]struct{}, len(refs))
	for _, ref := range refs {
		if err := ValidateRecordText("evidence reference", ref, MaxEvidenceReferenceBytes, true); err != nil {
			return err
		}
		if _, exists := seen[ref]; exists {
			return fmt.Errorf("duplicate evidence reference")
		}
		seen[ref] = struct{}{}
	}
	return nil
}

// ValidateReviewAttestation validates a statement of review, not its truth or quality.
func ValidateReviewAttestation(attestation *ReviewAttestation) error {
	if attestation == nil {
		return fmt.Errorf("scheme-2 review requires an attestation stating what was checked")
	}
	if len(attestation.ProtoReflect().GetUnknown()) != 0 {
		return fmt.Errorf("unsupported review attestation fields")
	}
	for _, field := range []struct {
		name, value string
		limit       int
		required    bool
	}{
		{"review reason", attestation.Reason, MaxReviewReasonBytes, true},
		{"review method", attestation.MethodId, MaxMethodIDBytes, false},
		{"review scope", attestation.Scope, MaxReviewScopeBytes, false},
	} {
		if err := ValidateRecordText(field.name, field.value, field.limit, field.required); err != nil {
			return err
		}
	}
	if err := ValidateEvidenceReferences(attestation.EvidenceIds); err != nil {
		return err
	}
	total := len(attestation.MethodId) + len(attestation.Reason) + len(attestation.Scope)
	for _, ref := range attestation.EvidenceIds {
		total += len(ref)
	}
	if total > MaxReviewAttestationBytes {
		return fmt.Errorf("review attestation exceeds %d bytes", MaxReviewAttestationBytes)
	}
	return nil
}

// CanonicalReviewAddress requires the exact SDK account-address spelling and
// returns its bytes. Ante authenticates this message signer independently.
func CanonicalReviewAddress(verifier string) ([]byte, error) {
	address, err := sdk.AccAddressFromBech32(verifier)
	if err != nil {
		return nil, fmt.Errorf("invalid reviewer address: %w", err)
	}
	if address.String() != verifier {
		return nil, fmt.Errorf("reviewer address must use canonical encoding")
	}
	return []byte(address), nil
}

// ComputeReviewCommitmentV2 hashes an unambiguous, versioned projection of all
// signed review fields. It uses the round's stored creation-chain identity;
// an imported in-flight round retains that identity, while its transaction
// is still authenticated on the importing chain by the SDK.
//
// Encoding: domain bytes "ZRN.review.commit.v2\x00", then uint32-big-endian
// length-prefixed chain, round, canonical reviewer bytes, vote; uint64-big-endian
// confidence; length-prefixed salt, method, reason, scope; uint32 evidence count
// and each length-prefixed evidence reference, preserving submitted order.
// No protobuf serialization, map iteration, delimiters, normalization or unknown
// fields participate. Legacy ComputeCommitmentHash is deliberately unchanged.
func ComputeReviewCommitmentV2(chainID, roundID, verifier, vote string, confidence uint64, salt []byte, attestation *ReviewAttestation) ([]byte, error) {
	if err := ValidateRecordText("commitment chain", chainID, MaxCommitmentContextBytes, true); err != nil {
		return nil, err
	}
	if err := ValidateRecordText("commitment round", roundID, MaxCommitmentContextBytes, true); err != nil {
		return nil, err
	}
	address, err := CanonicalReviewAddress(verifier)
	if err != nil {
		return nil, err
	}
	if vote != "accept" && vote != "reject" && vote != "malformed" {
		return nil, fmt.Errorf("invalid review vote")
	}
	if confidence > BPS {
		return nil, fmt.Errorf("review confidence must be between 0 and %d", BPS)
	}
	if len(salt) < 16 || len(salt) > 64 {
		return nil, fmt.Errorf("scheme-2 salt must contain 16 to 64 bytes")
	}
	if err := ValidateReviewAttestation(attestation); err != nil {
		return nil, err
	}
	h := sha256.New()
	_, _ = h.Write([]byte("ZRN.review.commit.v2\x00"))
	var word [8]byte
	put := func(value []byte) {
		binary.BigEndian.PutUint32(word[:4], uint32(len(value)))
		_, _ = h.Write(word[:4])
		_, _ = h.Write(value)
	}
	for _, value := range [][]byte{[]byte(chainID), []byte(roundID), address, []byte(vote)} {
		put(value)
	}
	binary.BigEndian.PutUint64(word[:], confidence)
	_, _ = h.Write(word[:])
	for _, value := range [][]byte{salt, []byte(attestation.MethodId), []byte(attestation.Reason), []byte(attestation.Scope)} {
		put(value)
	}
	binary.BigEndian.PutUint32(word[:4], uint32(len(attestation.EvidenceIds)))
	_, _ = h.Write(word[:4])
	for _, ref := range attestation.EvidenceIds {
		put([]byte(ref))
	}
	return h.Sum(nil), nil
}

// ValidateVerificationRoundRecord preserves the legacy scheme explicitly and
// refuses unknown scheme versions instead of selecting a fallback hash.
// Imported scheme-2 rounds keep their original chain context.
func ValidateVerificationRoundRecord(round *VerificationRound, recordIntegrityEnabled bool) error {
	if round == nil || round.Id == "" || round.ClaimId == "" {
		return fmt.Errorf("verification round requires id and claim_id")
	}
	if len(round.ProtoReflect().GetUnknown()) != 0 {
		return fmt.Errorf("unsupported verification round fields")
	}
	if round.Phase < VerificationPhase_VERIFICATION_PHASE_COMMIT || round.Phase > VerificationPhase_VERIFICATION_PHASE_EXPIRED {
		return fmt.Errorf("invalid verification round phase")
	}
	switch round.CommitmentScheme {
	case CommitmentSchemeLegacy:
		if round.CommitmentChainId != "" {
			return fmt.Errorf("legacy round cannot declare scheme-2 chain context")
		}
		for _, reveal := range round.Reveals {
			if reveal == nil || reveal.Confidence != 0 || reveal.Attestation != nil {
				return fmt.Errorf("legacy round cannot contain unbound review fields")
			}
		}
	case CommitmentSchemeReviewV2:
		if !recordIntegrityEnabled {
			return fmt.Errorf("scheme-2 round requires record-integrity activation")
		}
		if err := ValidateRecordText("commitment chain", round.CommitmentChainId, MaxCommitmentContextBytes, true); err != nil {
			return err
		}
		if err := ValidateRecordText("commitment round", round.Id, MaxCommitmentContextBytes, true); err != nil {
			return err
		}
		if len(round.Commits) > CommitSeatHardCap || len(round.Reveals) > CommitSeatHardCap || len(round.SelectedVerifiers) > CommitSeatHardCap {
			return fmt.Errorf("scheme-2 round exceeds verifier bound")
		}
		commits := make(map[string]*CommitEntry, len(round.Commits))
		for _, commit := range round.Commits {
			if commit == nil || len(commit.ProtoReflect().GetUnknown()) != 0 || len(commit.CommitHash) != sha256.Size {
				return fmt.Errorf("invalid scheme-2 commitment")
			}
			if _, err := CanonicalReviewAddress(commit.Verifier); err != nil {
				return err
			}
			if _, exists := commits[commit.Verifier]; exists {
				return fmt.Errorf("duplicate scheme-2 commitment")
			}
			commits[commit.Verifier] = commit
		}
		selected := make(map[string]bool, len(round.SelectedVerifiers))
		for _, verifier := range round.SelectedVerifiers {
			if selected[verifier] || commits[verifier] == nil {
				return fmt.Errorf("scheme-2 selected reviewers must match distinct commitments")
			}
			selected[verifier] = true
		}
		if len(selected) != len(commits) {
			return fmt.Errorf("scheme-2 selected reviewers omit a commitment")
		}
		seen := make(map[string]bool, len(round.Reveals))
		for _, reveal := range round.Reveals {
			if reveal == nil || len(reveal.ProtoReflect().GetUnknown()) != 0 {
				return fmt.Errorf("invalid scheme-2 reveal")
			}
			if seen[reveal.Verifier] {
				return fmt.Errorf("duplicate scheme-2 reveal")
			}
			seen[reveal.Verifier] = true
			commit := commits[reveal.Verifier]
			if commit == nil {
				return fmt.Errorf("scheme-2 reveal has no commitment")
			}
			hash, err := ComputeReviewCommitmentV2(round.CommitmentChainId, round.Id, reveal.Verifier, reveal.Vote, reveal.Confidence, reveal.Salt, reveal.Attestation)
			if err != nil {
				return err
			}
			if !bytes.Equal(commit.CommitHash, hash) {
				return fmt.Errorf("scheme-2 reveal does not match commitment")
			}
		}
	default:
		return fmt.Errorf("unsupported commitment scheme %d", round.CommitmentScheme)
	}
	if round.VerifierRewardSettlement != nil && !recordIntegrityEnabled {
		return fmt.Errorf("verifier settlement requires record-integrity activation")
	}
	return ValidateVerifierRewardSettlement(round)
}
