package main

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
)

// decodeSupersession preserves the same canonical-byte discipline as v1
// transition documents while evolving through an independent schema.
func decodeSupersession(data []byte, requireCanonical bool) (Supersession, error) {
	var supersession Supersession
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&supersession); err != nil {
		return Supersession{}, fmt.Errorf("decode supersession JSON: %w", err)
	}
	var extra json.RawMessage
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return Supersession{}, errors.New("decode supersession JSON: multiple values are not allowed")
		}
		return Supersession{}, fmt.Errorf("decode trailing supersession JSON: %w", err)
	}
	canonical, err := json.Marshal(supersession)
	if err != nil {
		return Supersession{}, fmt.Errorf("canonicalize supersession: %w", err)
	}
	if requireCanonical && !bytes.Equal(data, canonical) {
		return Supersession{}, errors.New(
			"supersession JSON is not canonical: require exact compact field order with no whitespace or trailing newline",
		)
	}
	return supersession, nil
}

func supersessionHash(supersession Supersession) (string, error) {
	supersession.SupersessionSHA256 = ""
	canonical, err := json.Marshal(supersession)
	if err != nil {
		return "", fmt.Errorf("canonicalize supersession: %w", err)
	}
	digest := sha256.Sum256(canonical)
	return hex.EncodeToString(digest[:]), nil
}

type supersessionApprovalStatement struct {
	Supersession Supersession  `json:"supersession"`
	Approval     approvalClaim `json:"approval"`
}

func supersessionApprovalStatementDigest(
	supersession Supersession,
	approval Approval,
) ([sha256.Size]byte, error) {
	supersession.Approvals = []Approval{}
	supersession.SupersessionSHA256 = ""
	statement := supersessionApprovalStatement{
		Supersession: supersession,
		Approval: approvalClaim{
			Role:      approval.Role,
			Identity:  approval.Identity,
			PublicKey: approval.PublicKey,
			Power:     approval.Power,
		},
	}
	canonical, err := json.Marshal(statement)
	if err != nil {
		return [sha256.Size]byte{}, err
	}
	message := make([]byte, 0, len(supersessionApprovalDomain)+len(canonical))
	message = append(message, supersessionApprovalDomain...)
	message = append(message, canonical...)
	return sha256.Sum256(message), nil
}

func validateSupersessionCore(supersession Supersession) error {
	if supersession.Schema != supersessionSchema {
		return fmt.Errorf(
			"supersession schema must be %q, got %q",
			supersessionSchema,
			supersession.Schema,
		)
	}
	if err := validateLabel("supersession chain_id", supersession.ChainID, 128); err != nil {
		return err
	}
	for _, hash := range []struct {
		name  string
		value string
	}{
		{"supersession old_journal_head_sha256", supersession.OldJournalHeadSHA256},
		{"supersession old_trust_policy_sha256", supersession.OldTrustPolicySHA256},
		{"supersession new_trust_policy_sha256", supersession.NewTrustPolicySHA256},
	} {
		if err := validateSHA256(hash.name, hash.value, false); err != nil {
			return err
		}
	}
	if supersession.OldTrustPolicySHA256 == supersession.NewTrustPolicySHA256 {
		return errors.New("supersession requires a different replacement trust policy")
	}
	hasIncident := supersession.ReplacementIncidentID != ""
	hasRelease := supersession.ReplacementReleaseID != ""
	if hasIncident == hasRelease {
		return errors.New(
			"supersession requires exactly one replacement incident_id or release_id",
		)
	}
	if supersession.ReplacementIncidentID != "" {
		if err := validateLabel(
			"supersession replacement_incident_id",
			supersession.ReplacementIncidentID,
			128,
		); err != nil {
			return err
		}
	}
	if supersession.ReplacementReleaseID != "" {
		if err := validateLabel(
			"supersession replacement_release_id",
			supersession.ReplacementReleaseID,
			128,
		); err != nil {
			return err
		}
	}
	if err := validateCanonicalTime(supersession.OccurredAt); err != nil {
		return fmt.Errorf("supersession occurred_at: %w", err)
	}
	if supersession.Reason == "" ||
		strings.TrimSpace(supersession.Reason) != supersession.Reason ||
		len(supersession.Reason) > 4096 {
		return errors.New("supersession reason must be non-empty, trimmed, and at most 4096 bytes")
	}
	if err := validateEvidence(supersession.Evidence); err != nil {
		return fmt.Errorf("supersession: %w", err)
	}
	presentEvidence := make(map[string]int, len(supersession.Evidence))
	for _, item := range supersession.Evidence {
		presentEvidence[item.Type]++
	}
	for _, required := range []string{
		"replacement-policy-ceremony",
		"trust-policy-compromise-assessment",
	} {
		if presentEvidence[required] != 1 {
			return fmt.Errorf(
				"supersession requires exactly one evidence item of type %q",
				required,
			)
		}
	}
	if supersession.Approvals == nil {
		return errors.New("supersession approvals must be [] rather than null")
	}
	return nil
}

func validateSupersessionBindings(
	supersession Supersession,
	oldPolicy TrustPolicy,
	oldPolicySHA256 string,
	newPolicy TrustPolicy,
	newPolicySHA256 string,
	expectedOldHeadSHA256 string,
) error {
	if err := validateTrustPolicy(oldPolicy); err != nil {
		return fmt.Errorf("invalid old trust policy: %w", err)
	}
	if err := validateTrustPolicy(newPolicy); err != nil {
		return fmt.Errorf("invalid new trust policy: %w", err)
	}
	actualOldPolicySHA256, err := trustPolicySHA256(oldPolicy)
	if err != nil {
		return err
	}
	if actualOldPolicySHA256 != oldPolicySHA256 {
		return fmt.Errorf(
			"old trust policy object has SHA-256 %s, caller pinned %s",
			actualOldPolicySHA256,
			oldPolicySHA256,
		)
	}
	actualNewPolicySHA256, err := trustPolicySHA256(newPolicy)
	if err != nil {
		return err
	}
	if actualNewPolicySHA256 != newPolicySHA256 {
		return fmt.Errorf(
			"new trust policy object has SHA-256 %s, caller pinned %s",
			actualNewPolicySHA256,
			newPolicySHA256,
		)
	}
	if err := validateSHA256(
		"externally pinned old journal head SHA-256",
		expectedOldHeadSHA256,
		false,
	); err != nil {
		return fmt.Errorf("an externally obtained old journal head is required: %w", err)
	}
	if supersession.OldJournalHeadSHA256 != expectedOldHeadSHA256 {
		return fmt.Errorf(
			"old journal head mismatch: sidecar has %s, externally pinned value is %s",
			supersession.OldJournalHeadSHA256,
			expectedOldHeadSHA256,
		)
	}
	if supersession.OldTrustPolicySHA256 != oldPolicySHA256 {
		return fmt.Errorf(
			"old trust policy mismatch: sidecar has %s, externally pinned value is %s",
			supersession.OldTrustPolicySHA256,
			oldPolicySHA256,
		)
	}
	if supersession.NewTrustPolicySHA256 != newPolicySHA256 {
		return fmt.Errorf(
			"new trust policy mismatch: sidecar has %s, externally pinned value is %s",
			supersession.NewTrustPolicySHA256,
			newPolicySHA256,
		)
	}
	if supersession.ChainID != oldPolicy.ChainID ||
		supersession.ChainID != newPolicy.ChainID {
		return fmt.Errorf(
			"supersession chain_id %q must match old policy %q and new policy %q",
			supersession.ChainID,
			oldPolicy.ChainID,
			newPolicy.ChainID,
		)
	}
	if supersession.ReplacementIncidentID != newPolicy.IncidentID ||
		supersession.ReplacementReleaseID != newPolicy.ReleaseID {
		return errors.New(
			"replacement incident_id/release_id must exactly match the externally pinned new trust policy",
		)
	}
	return nil
}

func validateSupersessionApprovals(
	supersession Supersession,
	oldPolicy TrustPolicy,
) error {
	if len(supersession.Approvals) != 2 {
		return errors.New(
			"supersession requires exactly two approvals: evidence-custodian and policy-rotation-authority",
		)
	}
	if !sort.SliceIsSorted(supersession.Approvals, func(i, j int) bool {
		return approvalLess(supersession.Approvals[i], supersession.Approvals[j])
	}) {
		return errors.New("supersession approvals must be sorted by role, identity, then public_key")
	}

	requiredRoles := map[string]bool{
		evidenceCustodianRole:       true,
		policyRotationAuthorityRole: true,
	}
	seenRoles := make(map[string]bool, len(requiredRoles))
	var firstIdentity, firstKey string
	for i, approval := range supersession.Approvals {
		prefix := fmt.Sprintf("supersession approvals[%d]", i)
		if !requiredRoles[approval.Role] {
			return fmt.Errorf("%s has unauthorized role %q", prefix, approval.Role)
		}
		if seenRoles[approval.Role] {
			return fmt.Errorf("%s duplicates role %q", prefix, approval.Role)
		}
		seenRoles[approval.Role] = true
		if err := validateLabel(prefix+".identity", approval.Identity, 256); err != nil {
			return err
		}
		publicKey, err := decodeExactHex(
			prefix+".public_key",
			approval.PublicKey,
			ed25519.PublicKeySize,
		)
		if err != nil {
			return err
		}
		signature, err := decodeExactHex(
			prefix+".signature",
			approval.Signature,
			ed25519.SignatureSize,
		)
		if err != nil {
			return err
		}
		if approval.Power != "0" {
			return fmt.Errorf("%s.power must be \"0\"", prefix)
		}
		if !trustedSignerAuthorized(
			oldPolicy,
			approval.Role,
			approval.Identity,
			approval.PublicKey,
		) {
			return fmt.Errorf(
				"%s role/identity/public_key is not authorized by the old trust policy",
				prefix,
			)
		}
		statementDigest, err := supersessionApprovalStatementDigest(
			supersession,
			approval,
		)
		if err != nil {
			return fmt.Errorf("compute %s statement: %w", prefix, err)
		}
		statementHex := hex.EncodeToString(statementDigest[:])
		if approval.StatementSHA256 != statementHex {
			return fmt.Errorf("%s.statement_sha256 does not bind this supersession", prefix)
		}
		if !ed25519.Verify(
			ed25519.PublicKey(publicKey),
			statementDigest[:],
			signature,
		) {
			return fmt.Errorf("%s.signature is invalid", prefix)
		}
		if i == 0 {
			firstIdentity = approval.Identity
			firstKey = approval.PublicKey
		} else if approval.Identity == firstIdentity ||
			approval.PublicKey == firstKey {
			return errors.New(
				"supersession authority roles require distinct identities and public keys",
			)
		}
	}
	for required := range requiredRoles {
		if !seenRoles[required] {
			return fmt.Errorf("supersession is missing approval role %q", required)
		}
	}
	return nil
}

func sealSupersession(
	supersession Supersession,
	oldPolicy TrustPolicy,
	oldPolicySHA256 string,
	newPolicy TrustPolicy,
	newPolicySHA256 string,
	expectedOldHeadSHA256 string,
) (Supersession, error) {
	if supersession.SupersessionSHA256 != "" {
		return Supersession{}, errors.New("supersession_sha256 must be empty before sealing")
	}
	if err := validateSupersessionCore(supersession); err != nil {
		return Supersession{}, err
	}
	if err := validateSupersessionBindings(
		supersession,
		oldPolicy,
		oldPolicySHA256,
		newPolicy,
		newPolicySHA256,
		expectedOldHeadSHA256,
	); err != nil {
		return Supersession{}, err
	}
	if err := validateSupersessionApprovals(supersession, oldPolicy); err != nil {
		return Supersession{}, err
	}
	digest, err := supersessionHash(supersession)
	if err != nil {
		return Supersession{}, err
	}
	supersession.SupersessionSHA256 = digest
	return supersession, nil
}

func verifySupersession(
	document []byte,
	oldPolicy TrustPolicy,
	oldPolicySHA256 string,
	newPolicy TrustPolicy,
	newPolicySHA256 string,
	expectedOldHeadSHA256 string,
) (SupersessionVerificationResult, error) {
	supersession, err := decodeSupersession(document, true)
	if err != nil {
		return SupersessionVerificationResult{}, err
	}
	if err := validateSupersessionCore(supersession); err != nil {
		return SupersessionVerificationResult{}, err
	}
	if err := validateSupersessionBindings(
		supersession,
		oldPolicy,
		oldPolicySHA256,
		newPolicy,
		newPolicySHA256,
		expectedOldHeadSHA256,
	); err != nil {
		return SupersessionVerificationResult{}, err
	}
	if err := validateSupersessionApprovals(supersession, oldPolicy); err != nil {
		return SupersessionVerificationResult{}, err
	}
	if err := validateSHA256(
		"supersession_sha256",
		supersession.SupersessionSHA256,
		false,
	); err != nil {
		return SupersessionVerificationResult{}, err
	}
	expected, err := supersessionHash(supersession)
	if err != nil {
		return SupersessionVerificationResult{}, err
	}
	if supersession.SupersessionSHA256 != expected {
		return SupersessionVerificationResult{}, fmt.Errorf(
			"supersession_sha256 mismatch: got %s, expected %s",
			supersession.SupersessionSHA256,
			expected,
		)
	}
	return SupersessionVerificationResult{
		ChainID:               supersession.ChainID,
		OldJournalHeadSHA256:  supersession.OldJournalHeadSHA256,
		OldTrustPolicySHA256:  supersession.OldTrustPolicySHA256,
		NewTrustPolicySHA256:  supersession.NewTrustPolicySHA256,
		ReplacementIncidentID: supersession.ReplacementIncidentID,
		ReplacementReleaseID:  supersession.ReplacementReleaseID,
		SupersessionSHA256:    supersession.SupersessionSHA256,
	}, nil
}
