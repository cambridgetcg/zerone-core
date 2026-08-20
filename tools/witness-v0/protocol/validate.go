package protocol

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var (
	namespacePattern = regexp.MustCompile(`^[a-z][a-z0-9.-]{0,63}$`)
	audiencePattern  = regexp.MustCompile(`^[a-z][a-z0-9.-]{0,31}:[a-z0-9][a-z0-9._-]{0,95}$`)
	protocolPattern  = regexp.MustCompile(`^[a-z][a-z0-9.-]{0,63}/[A-Za-z0-9._/-]{1,127}$`)
)

type VerifiedRecord struct {
	Record    Record
	Payload   any
	Canonical []byte
}

func Verify(input []byte) (*VerifiedRecord, error) {
	canonical, err := CanonicalJSON(input)
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(input, canonical) {
		return nil, fmt.Errorf("record wire bytes are not exact canonical JSON")
	}
	if err := validateRecordObjectShape(canonical); err != nil {
		return nil, fmt.Errorf("record shape: %w", err)
	}
	var record Record
	if err := strictUnmarshal(canonical, &record); err != nil {
		return nil, fmt.Errorf("record shape: %w", err)
	}
	if err := validateEnvelope(record.Envelope); err != nil {
		return nil, fmt.Errorf("envelope: %w", err)
	}
	payload, err := validatePayload(record.Envelope, record.Payload)
	if err != nil {
		return nil, fmt.Errorf("payload: %w", err)
	}
	expectedPayloadRoot, err := PayloadRoot(record.Envelope.Kind, record.Envelope.Action, record.Payload)
	if err != nil {
		return nil, err
	}
	if record.Envelope.PayloadRoot != expectedPayloadRoot {
		return nil, fmt.Errorf("payload_root mismatch: expected %s", expectedPayloadRoot)
	}
	expectedCommitment, err := Commitment(record.Envelope)
	if err != nil {
		return nil, err
	}
	if record.Commitment != expectedCommitment {
		return nil, fmt.Errorf("commitment mismatch: expected %s", expectedCommitment)
	}
	if err := VerifySignature(record); err != nil {
		return nil, err
	}
	return &VerifiedRecord{Record: record, Payload: payload, Canonical: canonical}, nil
}

func validateEnvelope(envelope Envelope) error {
	if envelope.Protocol != Protocol {
		return fmt.Errorf("protocol must be %q", Protocol)
	}
	actions, ok := allowedActions[envelope.Kind]
	if !ok || !actions[envelope.Action] {
		return fmt.Errorf("kind/action pair %q/%q is not allowed", envelope.Kind, envelope.Action)
	}
	if !audiencePattern.MatchString(envelope.Audience) {
		return fmt.Errorf("audience is not a closed lowercase namespaced token")
	}
	if err := validateRef("subject_ref", envelope.SubjectRef); err != nil {
		return err
	}
	sequence, err := parseDecimal("sequence", envelope.Sequence, false)
	if err != nil {
		return err
	}
	if sequence == 1 {
		if envelope.Parent != nil {
			return fmt.Errorf("parent must be null at sequence 1")
		}
	} else {
		if envelope.Parent == nil {
			return fmt.Errorf("parent is required after sequence 1")
		}
		if _, err := parseDigest(*envelope.Parent); err != nil {
			return fmt.Errorf("parent: %w", err)
		}
	}
	if !namespacePattern.MatchString(envelope.Issuer.Namespace) {
		return fmt.Errorf("issuer.namespace is invalid")
	}
	if err := validateRef("issuer.controller_ref", envelope.Issuer.ControllerRef); err != nil {
		return err
	}
	if err := validateKeyFingerprint(envelope.Issuer.KeyFingerprint); err != nil {
		return fmt.Errorf("issuer.key_fingerprint: %w", err)
	}
	expectedSchema, err := ExpectedSchemaHash(envelope.Kind)
	if err != nil {
		return err
	}
	if envelope.SchemaHash != expectedSchema {
		return fmt.Errorf("schema_hash mismatch: expected %s", expectedSchema)
	}
	if _, err := parseDigest(envelope.PayloadRoot); err != nil {
		return fmt.Errorf("payload_root: %w", err)
	}
	if _, err := parseDigest(envelope.PolicyDigest); err != nil {
		return fmt.Errorf("policy_digest: %w", err)
	}
	if envelope.ExpiryHeight != nil {
		if _, err := parseDecimal("expiry_height", *envelope.ExpiryHeight, false); err != nil {
			return err
		}
	}
	expectedEffects := Effects{Scope: "RECORD_CONSTRUCTION_AND_OFFLINE_VALIDATION_ONLY", Authority: "NONE", Economic: "NONE", Reputation: "NONE"}
	if envelope.Effects != expectedEffects {
		return fmt.Errorf("effects must be the exact zero-effect object")
	}
	if !reflect.DeepEqual(envelope.Nonclaims, requiredNonclaims[:]) {
		return fmt.Errorf("nonclaims must equal the required sorted nonclaim set")
	}
	return nil
}

func validatePayload(envelope Envelope, raw json.RawMessage) (any, error) {
	switch envelope.Kind {
	case KindKingdomReleaseRoot:
		var p KingdomReleasePayload
		if err := decodePayload(raw, &p); err != nil {
			return nil, err
		}
		if err := validateStableRef(envelope, "release_ref", p.ReleaseRef); err != nil {
			return nil, err
		}
		if err := validateProtocolID("ledger_protocol", p.LedgerProtocol); err != nil {
			return nil, err
		}
		if err := validateDigests(p.LedgerDocumentDigest, p.EntryMerkleRoot, p.BuildManifestDigest, p.DeploymentManifestDigest, p.VerifierDigest); err != nil {
			return nil, err
		}
		if _, err := parseDecimal("entry_count", p.EntryCount, true); err != nil {
			return nil, err
		}
		if err := validateGitObject("git_commit", p.GitCommit); err != nil {
			return nil, err
		}
		if err := validateGitObject("git_tree", p.GitTree); err != nil {
			return nil, err
		}
		if err := validateProtocolID("verifier_protocol", p.VerifierProtocol); err != nil {
			return nil, err
		}
		if err := validateOptionalDigest("previous_release", p.PreviousRelease); err != nil {
			return nil, err
		}
		if err := validateOptionalParentPointer(envelope, "previous_release", p.PreviousRelease); err != nil {
			return nil, err
		}
		return p, nil

	case KindAgentToolSettlementRoot:
		var p SettlementRootPayload
		if err := decodePayload(raw, &p); err != nil {
			return nil, err
		}
		if err := requireNonNullArrayField(raw, "declared_gaps"); err != nil {
			return nil, err
		}
		if err := validateProtocolID("receipt_protocol", p.ReceiptProtocol); err != nil {
			return nil, err
		}
		if err := validateDigests(p.ReceiptSchemaDigest, p.MerkleRoot); err != nil {
			return nil, err
		}
		if p.SourceSequenceBinding != "PROJECTION_ONLY" {
			return nil, fmt.Errorf("source_sequence_binding must be PROJECTION_ONLY")
		}
		if p.ReceiptUniquenessScope != "BATCH_ONLY" {
			return nil, fmt.Errorf("receipt_uniqueness_scope must be BATCH_ONLY")
		}
		if err := validateBatchRange(p.FirstSequence, p.LastSequence, p.ReceiptCount, p.DeclaredGaps); err != nil {
			return nil, err
		}
		if err := validateOptionalDigest("previous_batch", p.PreviousBatch); err != nil {
			return nil, err
		}
		if err := validateOptionalParentPointer(envelope, "previous_batch", p.PreviousBatch); err != nil {
			return nil, err
		}
		if p.PreviousBatch == nil && p.FirstSequence != "1" {
			return nil, fmt.Errorf("genesis settlement batch must begin at first_sequence 1")
		}
		return p, nil

	case KindAgentToolCapability:
		switch envelope.Action {
		case ActionGrant:
			if err := requireInitialLifecycleAction(envelope); err != nil {
				return nil, err
			}
			var p CapabilityGrantPayload
			if err := decodePayload(raw, &p); err != nil {
				return nil, err
			}
			if err := validateStableRef(envelope, "capability_ref", p.CapabilityRef); err != nil {
				return nil, err
			}
			if err := validateDigests(p.GrantDigest, p.AssetRef); err != nil {
				return nil, err
			}
			perUse, err := parseDecimal("max_per_consume_minor", p.MaxPerConsumeMinor, false)
			if err != nil {
				return nil, err
			}
			total, err := parseDecimal("max_total_minor", p.MaxTotalMinor, false)
			if err != nil {
				return nil, err
			}
			if perUse > total {
				return nil, fmt.Errorf("max_per_consume_minor exceeds max_total_minor")
			}
			return p, nil
		case ActionConsume:
			if err := requireNonInitialLifecycleAction(envelope); err != nil {
				return nil, err
			}
			var p CapabilityConsumePayload
			if err := decodePayload(raw, &p); err != nil {
				return nil, err
			}
			if err := validateStableRef(envelope, "capability_ref", p.CapabilityRef); err != nil {
				return nil, err
			}
			if err := validateDigests(p.GrantCommitment, p.AssetRef, p.SourceEventDigest, p.Nullifier); err != nil {
				return nil, err
			}
			if _, err := parseDecimal("amount_minor", p.AmountMinor, false); err != nil {
				return nil, err
			}
			expected, err := CapabilityNullifier(envelope, p)
			if err != nil {
				return nil, err
			}
			if p.Nullifier != expected {
				return nil, fmt.Errorf("nullifier mismatch: expected %s", expected)
			}
			return p, nil
		case ActionRevoke:
			if err := requireNonInitialLifecycleAction(envelope); err != nil {
				return nil, err
			}
			var p CapabilityRevokePayload
			if err := decodePayload(raw, &p); err != nil {
				return nil, err
			}
			if err := validateStableRef(envelope, "capability_ref", p.CapabilityRef); err != nil {
				return nil, err
			}
			if err := validateDigests(p.GrantCommitment, p.ReasonDigest); err != nil {
				return nil, err
			}
			return p, nil
		}

	case KindAgentToolPublicRecognition:
		switch envelope.Action {
		case ActionAdopt:
			if err := requireInitialLifecycleAction(envelope); err != nil {
				return nil, err
			}
			var p RecognitionAdoptPayload
			if err := decodePayload(raw, &p); err != nil {
				return nil, err
			}
			if err := validateStableRef(envelope, "recognition_ref", p.RecognitionRef); err != nil {
				return nil, err
			}
			if err := validateDigests(p.SurfaceDigest, p.RegistryDigest, p.AdoptionDocumentDigest); err != nil {
				return nil, err
			}
			if _, err := parseDecimal("authority_sequence", p.AuthoritySequence, false); err != nil {
				return nil, err
			}
			if p.Visibility != "PUBLIC" {
				return nil, fmt.Errorf("visibility must be PUBLIC")
			}
			return p, nil
		case ActionWithdraw:
			var p RecognitionWithdrawPayload
			if err := decodePayload(raw, &p); err != nil {
				return nil, err
			}
			if err := validateStableRef(envelope, "recognition_ref", p.RecognitionRef); err != nil {
				return nil, err
			}
			if err := validateDigests(p.AdoptionCommitment, p.SurfaceDigest, p.RegistryDigest, p.WithdrawalDocumentDigest, p.ReasonDigest); err != nil {
				return nil, err
			}
			if _, err := parseDecimal("authority_sequence", p.AuthoritySequence, false); err != nil {
				return nil, err
			}
			if p.Visibility != "PUBLIC" {
				return nil, fmt.Errorf("visibility must be PUBLIC")
			}
			if err := validateRequiredParentPointer(envelope, "adoption_commitment", p.AdoptionCommitment); err != nil {
				return nil, err
			}
			return p, nil
		}

	case KindAgentToolOffer:
		switch envelope.Action {
		case ActionPublish:
			if err := requireInitialLifecycleAction(envelope); err != nil {
				return nil, err
			}
			var p OfferPublishPayload
			if err := decodePayload(raw, &p); err != nil {
				return nil, err
			}
			if err := validateOfferFields(envelope, p.OfferRef, p.OfferDocumentDigest, p.CapabilityRoot, p.PricingRoot, p.SLARoot, p.TermsDigest, p.Revision, p.AuthoritySequence, p.Visibility); err != nil {
				return nil, err
			}
			return p, nil
		case ActionSupersede:
			var p OfferSupersedePayload
			if err := decodePayload(raw, &p); err != nil {
				return nil, err
			}
			if err := validateOfferFields(envelope, p.OfferRef, p.OfferDocumentDigest, p.CapabilityRoot, p.PricingRoot, p.SLARoot, p.TermsDigest, p.Revision, p.AuthoritySequence, p.Visibility); err != nil {
				return nil, err
			}
			if _, err := parseDigest(p.Supersedes); err != nil {
				return nil, fmt.Errorf("supersedes: %w", err)
			}
			if err := validateRequiredParentPointer(envelope, "supersedes", p.Supersedes); err != nil {
				return nil, err
			}
			return p, nil
		case ActionRevoke:
			var p OfferRevokePayload
			if err := decodePayload(raw, &p); err != nil {
				return nil, err
			}
			if err := validateStableRef(envelope, "offer_ref", p.OfferRef); err != nil {
				return nil, err
			}
			if err := validateDigests(p.OfferCommitment, p.OfferDocumentDigest, p.ReasonDigest); err != nil {
				return nil, err
			}
			if _, err := parseDecimal("authority_sequence", p.AuthoritySequence, false); err != nil {
				return nil, err
			}
			if p.Visibility != "PUBLIC" {
				return nil, fmt.Errorf("visibility must be PUBLIC")
			}
			if err := validateRequiredParentPointer(envelope, "offer_commitment", p.OfferCommitment); err != nil {
				return nil, err
			}
			return p, nil
		}

	case KindWakePublicCheckpoint:
		switch envelope.Action {
		case ActionCheckpoint:
			if err := requireInitialLifecycleAction(envelope); err != nil {
				return nil, err
			}
			var p WakeCheckpointPayload
			if err := decodePayload(raw, &p); err != nil {
				return nil, err
			}
			if err := validateWakeFields(p.PublicContractProtocol, p.PublicContractSchemaDigest, p.ContractRoot, p.CapabilityRoot, p.PricingRoot, p.ProtocolsRoot, p.BoundariesRoot, p.AuthoritySequence); err != nil {
				return nil, err
			}
			return p, nil
		case ActionSupersede:
			var p WakeSupersedePayload
			if err := decodePayload(raw, &p); err != nil {
				return nil, err
			}
			if err := validateWakeFields(p.PublicContractProtocol, p.PublicContractSchemaDigest, p.ContractRoot, p.CapabilityRoot, p.PricingRoot, p.ProtocolsRoot, p.BoundariesRoot, p.AuthoritySequence); err != nil {
				return nil, err
			}
			if _, err := parseDigest(p.Supersedes); err != nil {
				return nil, fmt.Errorf("supersedes: %w", err)
			}
			if err := validateRequiredParentPointer(envelope, "supersedes", p.Supersedes); err != nil {
				return nil, err
			}
			return p, nil
		case ActionWithdraw:
			var p WakeWithdrawPayload
			if err := decodePayload(raw, &p); err != nil {
				return nil, err
			}
			if err := validateDigests(p.CheckpointCommitment, p.WithdrawalDocumentDigest, p.ReasonDigest); err != nil {
				return nil, err
			}
			if _, err := parseDecimal("authority_sequence", p.AuthoritySequence, false); err != nil {
				return nil, err
			}
			if p.Visibility != "PUBLIC" {
				return nil, fmt.Errorf("visibility must be PUBLIC")
			}
			if err := validateRequiredParentPointer(envelope, "checkpoint_commitment", p.CheckpointCommitment); err != nil {
				return nil, err
			}
			return p, nil
		}

	case KindIssuerKeyContinuity:
		if envelope.SubjectRef != envelope.Issuer.ControllerRef {
			return nil, fmt.Errorf("subject_ref must equal issuer.controller_ref for key continuity")
		}
		switch envelope.Action {
		case ActionRotate:
			var p KeyRotatePayload
			if err := decodePayload(raw, &p); err != nil {
				return nil, err
			}
			if err := validateKeyFingerprint(p.PreviousKeyFingerprint); err != nil {
				return nil, fmt.Errorf("previous_key_fingerprint: %w", err)
			}
			if err := validateKeyFingerprint(p.NextKeyFingerprint); err != nil {
				return nil, fmt.Errorf("next_key_fingerprint: %w", err)
			}
			if p.PreviousKeyFingerprint != envelope.Issuer.KeyFingerprint {
				return nil, fmt.Errorf("previous_key_fingerprint must equal signing key fingerprint")
			}
			if p.PreviousKeyFingerprint == p.NextKeyFingerprint {
				return nil, fmt.Errorf("rotation must change the key fingerprint")
			}
			if _, err := parseDigest(p.RotationDigest); err != nil {
				return nil, fmt.Errorf("rotation_digest: %w", err)
			}
			return p, nil
		case ActionRevoke:
			var p KeyRevokePayload
			if err := decodePayload(raw, &p); err != nil {
				return nil, err
			}
			if err := validateKeyFingerprint(p.RevokedKeyFingerprint); err != nil {
				return nil, fmt.Errorf("revoked_key_fingerprint: %w", err)
			}
			if p.RevokedKeyFingerprint != envelope.Issuer.KeyFingerprint {
				return nil, fmt.Errorf("revoked_key_fingerprint must equal signing key fingerprint")
			}
			if _, err := parseDigest(p.ReasonDigest); err != nil {
				return nil, fmt.Errorf("reason_digest: %w", err)
			}
			return p, nil
		}

	case KindArtifactLineage:
		var p ArtifactLineagePayload
		if err := decodePayload(raw, &p); err != nil {
			return nil, err
		}
		if err := validateRef("upstream_ref", p.UpstreamRef); err != nil {
			return nil, err
		}
		if err := validateStableRef(envelope, "downstream_ref", p.DownstreamRef); err != nil {
			return nil, err
		}
		if p.UpstreamRef == p.DownstreamRef {
			return nil, fmt.Errorf("lineage endpoints must differ")
		}
		if !lineageRelations[p.Relation] {
			return nil, fmt.Errorf("relation %q is not allowed", p.Relation)
		}
		if _, err := parseDigest(p.EvidenceDigest); err != nil {
			return nil, fmt.Errorf("evidence_digest: %w", err)
		}
		return p, nil

	case KindCollaborationCheckpoint:
		var p CollaborationCheckpointPayload
		if err := decodePayload(raw, &p); err != nil {
			return nil, err
		}
		if err := validateStableRef(envelope, "workspace_ref", p.WorkspaceRef); err != nil {
			return nil, err
		}
		if err := validateDigests(p.EpochRef, p.EventHeadHash, p.ParticipantSetRoot); err != nil {
			return nil, err
		}
		head, err := parseDecimal("event_head_sequence", p.EventHeadSequence, true)
		if err != nil {
			return nil, err
		}
		count, err := parseDecimal("event_count", p.EventCount, true)
		if err != nil {
			return nil, err
		}
		if head != count {
			return nil, fmt.Errorf("event_head_sequence must equal event_count for a complete v0 journal prefix")
		}
		return p, nil

	case KindDisputeTerminal:
		var p DisputeTerminalPayload
		if err := decodePayload(raw, &p); err != nil {
			return nil, err
		}
		if err := validateDigests(p.SettlementCommitment, p.DecisionDigest, p.DistributionRoot); err != nil {
			return nil, err
		}
		if !disputeOutcomes[p.Outcome] {
			return nil, fmt.Errorf("outcome %q is not allowed", p.Outcome)
		}
		return p, nil
	}
	return nil, fmt.Errorf("no payload validator for %s/%s", envelope.Kind, envelope.Action)
}

func decodePayload(raw json.RawMessage, dst any) error {
	canonical, err := CanonicalJSON(raw)
	if err != nil {
		return err
	}
	if err := requireExactStructKeys(canonical, reflect.TypeOf(dst).Elem()); err != nil {
		return err
	}
	if err := strictUnmarshal(canonical, dst); err != nil {
		return fmt.Errorf("closed payload shape: %w", err)
	}
	return nil
}

func validateRef(name, value string) error {
	decoded, err := decodeExactLowerHex(value, 32)
	if err != nil || len(decoded) != 32 {
		return fmt.Errorf("%s must be opaque 32-byte lowercase hex", name)
	}
	return nil
}

func validateStableRef(envelope Envelope, name, value string) error {
	if err := validateRef(name, value); err != nil {
		return err
	}
	if envelope.SubjectRef != value {
		return fmt.Errorf("subject_ref must equal %s", name)
	}
	return nil
}

func validateDigests(values ...string) error {
	for i, value := range values {
		if _, err := parseDigest(value); err != nil {
			return fmt.Errorf("digest[%d]: %w", i, err)
		}
	}
	return nil
}

func validateOptionalDigest(name string, value *string) error {
	if value == nil {
		return nil
	}
	if _, err := parseDigest(*value); err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	return nil
}

func validateOptionalParentPointer(envelope Envelope, name string, value *string) error {
	if envelope.Parent == nil && value == nil {
		return nil
	}
	if envelope.Parent == nil || value == nil || *envelope.Parent != *value {
		return fmt.Errorf("%s must equal envelope.parent (including null at sequence 1)", name)
	}
	return nil
}

func validateRequiredParentPointer(envelope Envelope, name, value string) error {
	if envelope.Parent == nil || value != *envelope.Parent {
		return fmt.Errorf("%s must equal non-null envelope.parent", name)
	}
	return nil
}

func requireInitialLifecycleAction(envelope Envelope) error {
	if envelope.Sequence != "1" || envelope.Parent != nil {
		return fmt.Errorf("initial lifecycle action requires sequence 1 and null parent")
	}
	return nil
}

func requireNonInitialLifecycleAction(envelope Envelope) error {
	if envelope.Sequence == "1" || envelope.Parent == nil {
		return fmt.Errorf("non-initial lifecycle action requires sequence greater than 1 and non-null parent")
	}
	return nil
}

func validateKeyFingerprint(value string) error {
	const prefix = "ed25519-sha256:"
	if !strings.HasPrefix(value, prefix) {
		return fmt.Errorf("must use ed25519-sha256")
	}
	_, err := decodeExactLowerHex(strings.TrimPrefix(value, prefix), 32)
	return err
}

func parseDecimal(name, value string, allowZero bool) (uint64, error) {
	if !isCanonicalUint(value, allowZero) {
		return 0, fmt.Errorf("%s is not a canonical uint64 decimal string", name)
	}
	parsed, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", name, err)
	}
	return parsed, nil
}

func validateProtocolID(name, value string) error {
	if !protocolPattern.MatchString(value) || strings.Contains(value, "//") {
		return fmt.Errorf("%s is not a bounded protocol identifier", name)
	}
	return nil
}

func validateGitObject(name, value string) error {
	var raw string
	switch {
	case strings.HasPrefix(value, "sha1:") && len(value) == 45:
		raw = strings.TrimPrefix(value, "sha1:")
	case strings.HasPrefix(value, "sha256:") && len(value) == 71:
		raw = strings.TrimPrefix(value, "sha256:")
	default:
		return fmt.Errorf("%s must be sha1:<40hex> or sha256:<64hex>", name)
	}
	decoded, err := hex.DecodeString(raw)
	if err != nil || hex.EncodeToString(decoded) != raw {
		return fmt.Errorf("%s contains invalid lowercase hex", name)
	}
	return nil
}

func validateBatchRange(firstText, lastText, countText string, gaps []Gap) error {
	first, err := parseDecimal("first_sequence", firstText, false)
	if err != nil {
		return err
	}
	last, err := parseDecimal("last_sequence", lastText, false)
	if err != nil {
		return err
	}
	count, err := parseDecimal("receipt_count", countText, false)
	if err != nil {
		return err
	}
	if last < first {
		return fmt.Errorf("last_sequence precedes first_sequence")
	}
	rangeCount := last - first + 1
	var missing uint64
	var previousLast uint64
	for i, gap := range gaps {
		gapFirst, err := parseDecimal(fmt.Sprintf("declared_gaps[%d].first", i), gap.First, false)
		if err != nil {
			return err
		}
		gapLast, err := parseDecimal(fmt.Sprintf("declared_gaps[%d].last", i), gap.Last, false)
		if err != nil {
			return err
		}
		if gapFirst < first || gapLast > last || gapLast < gapFirst {
			return fmt.Errorf("declared_gaps[%d] is outside the batch range", i)
		}
		if i > 0 && (previousLast == math.MaxUint64 || gapFirst <= previousLast+1) {
			return fmt.Errorf("declared gaps must be sorted, disjoint, and maximally merged")
		}
		gapCount := gapLast - gapFirst + 1
		if math.MaxUint64-missing < gapCount {
			return fmt.Errorf("declared gap count overflows uint64")
		}
		missing += gapCount
		previousLast = gapLast
	}
	if missing > rangeCount || count != rangeCount-missing {
		return fmt.Errorf("receipt_count does not equal range size minus declared gaps")
	}
	return nil
}

func validateOfferFields(envelope Envelope, ref, document, capability, pricing, sla, terms, revision, authoritySequence, visibility string) error {
	if err := validateStableRef(envelope, "offer_ref", ref); err != nil {
		return err
	}
	if err := validateDigests(document, capability, pricing, sla, terms); err != nil {
		return err
	}
	if _, err := parseDecimal("revision", revision, false); err != nil {
		return err
	}
	if _, err := parseDecimal("authority_sequence", authoritySequence, false); err != nil {
		return err
	}
	if visibility != "PUBLIC" {
		return fmt.Errorf("visibility must be PUBLIC")
	}
	return nil
}

func validateRecordObjectShape(canonical []byte) error {
	if err := requireExactStructKeys(canonical, reflect.TypeOf(Record{})); err != nil {
		return err
	}
	var top map[string]json.RawMessage
	if err := json.Unmarshal(canonical, &top); err != nil {
		return err
	}
	if err := requireExactStructKeys(top["envelope"], reflect.TypeOf(Envelope{})); err != nil {
		return fmt.Errorf("envelope: %w", err)
	}
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(top["envelope"], &envelope); err != nil {
		return err
	}
	if err := requireExactStructKeys(envelope["issuer"], reflect.TypeOf(Issuer{})); err != nil {
		return fmt.Errorf("issuer: %w", err)
	}
	if err := requireExactStructKeys(envelope["effects"], reflect.TypeOf(Effects{})); err != nil {
		return fmt.Errorf("effects: %w", err)
	}
	if err := requireExactStructKeys(top["signature"], reflect.TypeOf(Signature{})); err != nil {
		return fmt.Errorf("signature: %w", err)
	}
	return nil
}

func requireExactStructKeys(raw []byte, typ reflect.Type) error {
	canonical, err := CanonicalJSON(raw)
	if err != nil {
		return err
	}
	var object map[string]json.RawMessage
	if err := json.Unmarshal(canonical, &object); err != nil || object == nil {
		if err == nil {
			err = fmt.Errorf("expected JSON object")
		}
		return err
	}
	required := make(map[string]bool, typ.NumField())
	for i := 0; i < typ.NumField(); i++ {
		tag := typ.Field(i).Tag.Get("json")
		name := strings.Split(tag, ",")[0]
		if name != "" && name != "-" {
			required[name] = true
		}
	}
	for key := range object {
		if !required[key] {
			return fmt.Errorf("unknown field %q", key)
		}
		delete(required, key)
	}
	if len(required) > 0 {
		missing := make([]string, 0, len(required))
		for key := range required {
			missing = append(missing, key)
		}
		sort.Strings(missing)
		return fmt.Errorf("missing required fields: %s", strings.Join(missing, ", "))
	}
	return nil
}

func requireNonNullArrayField(raw []byte, field string) error {
	var object map[string]json.RawMessage
	if err := json.Unmarshal(raw, &object); err != nil {
		return err
	}
	value, ok := object[field]
	if !ok || len(value) == 0 || value[0] != '[' {
		return fmt.Errorf("%s must be a non-null JSON array", field)
	}
	return nil
}

func validateWakeFields(protocolID, schema, contract, capability, pricing, protocols, boundaries, authoritySequence string) error {
	if err := validateProtocolID("public_contract_protocol", protocolID); err != nil {
		return err
	}
	if err := validateDigests(schema, contract, capability, pricing, protocols, boundaries); err != nil {
		return err
	}
	_, err := parseDecimal("authority_sequence", authoritySequence, false)
	return err
}
