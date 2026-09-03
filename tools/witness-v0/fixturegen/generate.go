// Package fixturegen builds the deterministic cross-language conformance
// corpus for kingdom.witnessed-agent-economy/0.1. It never reads a clock,
// randomness, or the network. Files are written only through WriteDir after a
// caller supplies an explicit destination.
package fixturegen

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"filippo.io/edwards25519"

	"github.com/zerone-chain/zerone/tools/witness-v0/protocol"
)

const FreezeState = "FROZEN"

type SchemaEntry struct {
	Kind       protocol.Kind `json:"kind"`
	SchemaHash string        `json:"schema_hash"`
}

type Vector struct {
	Path                    string          `json:"path"`
	FileSHA256              string          `json:"file_sha256"`
	Operation               string          `json:"operation"`
	Expected                string          `json:"expected"`
	Stage                   string          `json:"stage"`
	Code                    string          `json:"code"`
	ErrorContains           string          `json:"error_contains,omitempty"`
	Kind                    protocol.Kind   `json:"kind,omitempty"`
	Action                  protocol.Action `json:"action,omitempty"`
	Commitment              string          `json:"commitment,omitempty"`
	PayloadRoot             string          `json:"payload_root,omitempty"`
	MerkleRoot              string          `json:"merkle_root,omitempty"`
	CanonicalHex            string          `json:"canonical_hex,omitempty"`
	AcceptedRecords         string          `json:"accepted_records,omitempty"`
	PermanentNullifierCount string          `json:"permanent_nullifier_count,omitempty"`
}

type Manifest struct {
	Protocol                  string        `json:"protocol"`
	FreezeState               string        `json:"freeze_state"`
	WirePolicy                string        `json:"wire_policy"`
	ActionPairCount           string        `json:"action_pair_count"`
	RecordSchemaHash          string        `json:"record_schema_hash"`
	SettlementBatchSchemaHash string        `json:"settlement_batch_schema_hash"`
	SchemaSetDigest           string        `json:"schema_set_digest"`
	PayloadSchemas            []SchemaEntry `json:"payload_schemas"`
	Vectors                   []Vector      `json:"vectors"`
	CorpusDigestAlgorithm     string        `json:"corpus_digest_algorithm"`
	CorpusDigest              string        `json:"corpus_digest"`
	EmptyMerkleRoot           string        `json:"empty_merkle_root"`
	SettlementMerkleRoot      string        `json:"settlement_merkle_root"`
	CapabilityNullifier       string        `json:"capability_nullifier"`
	UnicodeCanonicalSHA256    string        `json:"unicode_canonical_sha256"`
	UnicodeCanonicalHex       string        `json:"unicode_canonical_hex"`
}

type Corpus struct {
	Files    map[string][]byte
	Manifest Manifest
}

type builder struct {
	files      map[string][]byte
	vectors    []Vector
	key1       ed25519.PrivateKey
	key2       ed25519.PrivateKey
	controller string
	records    map[string]protocol.Record
	wires      map[string][]byte
}

func Build() (Corpus, error) {
	b := &builder{
		files: make(map[string][]byte), records: make(map[string]protocol.Record), wires: make(map[string][]byte),
		key1: fixedKey(1), key2: fixedKey(2), controller: ref("controller"),
	}
	if err := b.build(); err != nil {
		return Corpus{}, err
	}
	return b.finish()
}

func (b *builder) build() error {
	// Three RFC6962 sidecars include an explicit-gap example and the same
	// authentic receipt digest reassigned across two batches. The latter is an
	// intentional v0 activation blocker, not a verifier claim of uniqueness.
	replayedReceipt := digest("receipt/replayed")
	batch1 := protocol.SettlementBatch{
		FirstSequence: "1", LastSequence: "1", ReceiptCount: "1", DeclaredGaps: []protocol.Gap{},
		Leaves: []protocol.SettlementLeaf{{Sequence: "1", ReceiptDigest: replayedReceipt}},
	}
	batch2 := protocol.SettlementBatch{
		FirstSequence: "2", LastSequence: "2", ReceiptCount: "1", DeclaredGaps: []protocol.Gap{},
		Leaves: []protocol.SettlementLeaf{{Sequence: "2", ReceiptDigest: replayedReceipt}},
	}
	gapBatch := protocol.SettlementBatch{
		FirstSequence: "1", LastSequence: "4", ReceiptCount: "3", DeclaredGaps: []protocol.Gap{{First: "2", Last: "2"}},
		Leaves: []protocol.SettlementLeaf{
			{Sequence: "1", ReceiptDigest: "sha256:" + strings.Repeat("1", 64)},
			{Sequence: "3", ReceiptDigest: "sha256:" + strings.Repeat("2", 64)},
			{Sequence: "4", ReceiptDigest: "sha256:" + strings.Repeat("3", 64)},
		},
	}
	batch1Wire := canonicalValue(batch1)
	batch2Wire := canonicalValue(batch2)
	gapBatchWire := canonicalValue(gapBatch)
	batch1Root, _ := protocol.SettlementMerkleRoot(batch1.Leaves)
	batch2Root, _ := protocol.SettlementMerkleRoot(batch2.Leaves)
	gapRoot, _ := protocol.SettlementMerkleRoot(gapBatch.Leaves)
	b.add("batches/settlement-replay-0001.json", batch1Wire, Vector{Operation: "VERIFY_BATCH", Expected: "ACCEPT", Stage: "BATCH", Code: "VALID_BATCH", MerkleRoot: batch1Root})
	b.add("batches/settlement-replay-0002.json", batch2Wire, Vector{Operation: "VERIFY_BATCH", Expected: "ACCEPT", Stage: "BATCH", Code: "BATCH_LOCAL_UNIQUENESS_ONLY", MerkleRoot: batch2Root})
	b.add("batches/settlement-with-gap.json", gapBatchWire, Vector{Operation: "VERIFY_BATCH", Expected: "ACCEPT", Stage: "BATCH", Code: "DECLARED_SEQUENCE_GAP", MerkleRoot: gapRoot})

	// Every closed kind/action pair has a deterministic signed record.
	releaseSubject := ref("release-stream")
	release := protocol.KingdomReleasePayload{
		ReleaseRef: releaseSubject, LedgerProtocol: "kingdom.immutable-publications/1", LedgerDocumentDigest: digest("ledger-document"),
		EntryMerkleRoot: digest("entry-root"), EntryCount: "83", GitCommit: "sha1:" + strings.Repeat("1", 40), GitTree: "sha1:" + strings.Repeat("2", 40),
		BuildManifestDigest: digest("build-manifest"), DeploymentManifestDigest: digest("deployment-manifest"),
		VerifierProtocol: "kingdom.ledger-verifier/1", VerifierDigest: digest("verifier"), PreviousRelease: nil,
	}
	if err := b.signRecord("kingdom-release-root", protocol.KindKingdomReleaseRoot, protocol.ActionCheckpoint, releaseSubject, "1", nil, b.controller, b.key1, release, "VERIFY_RECORD", "ACCEPT", "VALID_RECORD"); err != nil {
		return err
	}

	settlementSubject := ref("settlement-feed")
	settlement1 := protocol.SettlementRootPayload{
		ReceiptProtocol: "agenttool.settlement-receipt/1", ReceiptSchemaDigest: digest("receipt-schema"),
		SourceSequenceBinding: "PROJECTION_ONLY", ReceiptUniquenessScope: "BATCH_ONLY",
		FirstSequence: "1", LastSequence: "1", ReceiptCount: "1", DeclaredGaps: []protocol.Gap{}, MerkleRoot: batch1Root, PreviousBatch: nil,
	}
	if err := b.signRecord("settlement-root-0001", protocol.KindAgentToolSettlementRoot, protocol.ActionCheckpoint, settlementSubject, "1", nil, b.controller, b.key1, settlement1, "VERIFY_RECORD", "ACCEPT", "PUBLISHER_SIGNED_BATCH_SHADOW"); err != nil {
		return err
	}
	settlement1Record := b.records["settlement-root-0001"]
	settlement2 := protocol.SettlementRootPayload{
		ReceiptProtocol: settlement1.ReceiptProtocol, ReceiptSchemaDigest: settlement1.ReceiptSchemaDigest,
		SourceSequenceBinding: "PROJECTION_ONLY", ReceiptUniquenessScope: "BATCH_ONLY",
		FirstSequence: "2", LastSequence: "2", ReceiptCount: "1", DeclaredGaps: []protocol.Gap{}, MerkleRoot: batch2Root,
		PreviousBatch: stringPtr(settlement1Record.Commitment),
	}
	if err := b.signRecord("settlement-root-0002-cross-batch-replay", protocol.KindAgentToolSettlementRoot, protocol.ActionCheckpoint, settlementSubject, "2", stringPtr(settlement1Record.Commitment), b.controller, b.key1, settlement2, "VERIFY_RECORD_AND_ACTIVATION_AUDIT", "ACCEPT_STRUCTURALLY_BLOCKED_ACTIVATION", "CROSS_BATCH_RECEIPT_REPLAY_NOT_DETECTABLE_V0"); err != nil {
		return err
	}

	capability := ref("capability")
	grantPayload := protocol.CapabilityGrantPayload{
		CapabilityRef: capability, GrantDigest: digest("grant-document"), AssetRef: digest("asset"), MaxPerConsumeMinor: "10", MaxTotalMinor: "25",
	}
	if err := b.signRecord("capability-grant", protocol.KindAgentToolCapability, protocol.ActionGrant, capability, "1", nil, b.controller, b.key1, grantPayload, "VERIFY_RECORD", "ACCEPT", "VALID_RECORD"); err != nil {
		return err
	}
	grant := b.records["capability-grant"]
	consumeEnvelope := b.envelope(protocol.KindAgentToolCapability, protocol.ActionConsume, capability, "2", stringPtr(grant.Commitment), b.controller, b.key1)
	consumePayload := protocol.CapabilityConsumePayload{
		CapabilityRef: capability, GrantCommitment: grant.Commitment, AssetRef: grantPayload.AssetRef, AmountMinor: "7", SourceEventDigest: digest("source-event"),
	}
	consumePayload.Nullifier, _ = protocol.CapabilityNullifier(consumeEnvelope, consumePayload)
	if err := b.signPrepared("capability-consume", consumeEnvelope, b.key1, consumePayload, "VERIFY_RECORD", "ACCEPT", "SEQUENCE_INDEPENDENT_NULLIFIER"); err != nil {
		return err
	}
	consume := b.records["capability-consume"]
	revokePayload := protocol.CapabilityRevokePayload{CapabilityRef: capability, GrantCommitment: grant.Commitment, ReasonDigest: digest("capability-reason")}
	if err := b.signRecord("capability-revoke", protocol.KindAgentToolCapability, protocol.ActionRevoke, capability, "3", stringPtr(consume.Commitment), b.controller, b.key1, revokePayload, "VERIFY_RECORD", "ACCEPT", "VALID_RECORD"); err != nil {
		return err
	}

	maxCapability := ref("capability-uint64-max")
	maxGrantPayload := protocol.CapabilityGrantPayload{
		CapabilityRef: maxCapability, GrantDigest: digest("grant-max-document"), AssetRef: digest("asset-max"),
		MaxPerConsumeMinor: "18446744073709551615", MaxTotalMinor: "18446744073709551615",
	}
	if err := b.signRecord("capability-grant-uint64-max", protocol.KindAgentToolCapability, protocol.ActionGrant, maxCapability, "1", nil, b.controller, b.key1, maxGrantPayload, "VERIFY_RECORD", "ACCEPT", "UINT64_MAX_DECIMAL_STRING"); err != nil {
		return err
	}

	recognition := ref("recognition")
	adoptPayload := protocol.RecognitionAdoptPayload{
		RecognitionRef: recognition, SurfaceDigest: digest("surface"), RegistryDigest: digest("registry"),
		AdoptionDocumentDigest: digest("adoption-document"), AuthoritySequence: "1", Visibility: "PUBLIC",
	}
	if err := b.signRecord("recognition-adopt", protocol.KindAgentToolPublicRecognition, protocol.ActionAdopt, recognition, "1", nil, b.controller, b.key1, adoptPayload, "VERIFY_RECORD", "ACCEPT", "VALID_PUBLIC_SOURCE_BINDING"); err != nil {
		return err
	}
	adopt := b.records["recognition-adopt"]
	withdrawRecognition := protocol.RecognitionWithdrawPayload{
		RecognitionRef: recognition, AdoptionCommitment: adopt.Commitment, SurfaceDigest: adoptPayload.SurfaceDigest, RegistryDigest: adoptPayload.RegistryDigest,
		WithdrawalDocumentDigest: digest("recognition-withdrawal-document"), AuthoritySequence: "2", ReasonDigest: digest("recognition-reason"), Visibility: "PUBLIC",
	}
	if err := b.signRecord("recognition-withdraw", protocol.KindAgentToolPublicRecognition, protocol.ActionWithdraw, recognition, "2", stringPtr(adopt.Commitment), b.controller, b.key1, withdrawRecognition, "VERIFY_RECORD", "ACCEPT", "VALID_PUBLIC_SOURCE_BINDING"); err != nil {
		return err
	}

	offer := ref("offer")
	publishOffer := protocol.OfferPublishPayload{
		OfferRef: offer, OfferDocumentDigest: digest("offer-publish-document"), CapabilityRoot: digest("offer-capabilities"), PricingRoot: digest("offer-pricing"),
		SLARoot: digest("offer-sla"), TermsDigest: digest("offer-terms"), Revision: "1", AuthoritySequence: "10", Visibility: "PUBLIC",
	}
	if err := b.signRecord("offer-publish", protocol.KindAgentToolOffer, protocol.ActionPublish, offer, "1", nil, b.controller, b.key1, publishOffer, "VERIFY_RECORD", "ACCEPT", "VALID_PUBLIC_SOURCE_BINDING"); err != nil {
		return err
	}
	publishedOffer := b.records["offer-publish"]
	supersedeOffer := protocol.OfferSupersedePayload{
		OfferRef: offer, OfferDocumentDigest: digest("offer-supersede-document"), CapabilityRoot: digest("offer-capabilities-2"), PricingRoot: digest("offer-pricing-2"),
		SLARoot: digest("offer-sla-2"), TermsDigest: digest("offer-terms-2"), Revision: "2", AuthoritySequence: "11", Visibility: "PUBLIC", Supersedes: publishedOffer.Commitment,
	}
	if err := b.signRecord("offer-supersede", protocol.KindAgentToolOffer, protocol.ActionSupersede, offer, "2", stringPtr(publishedOffer.Commitment), b.controller, b.key1, supersedeOffer, "VERIFY_RECORD", "ACCEPT", "VALID_LIFECYCLE_POINTER"); err != nil {
		return err
	}
	supersededOffer := b.records["offer-supersede"]
	revokeOffer := protocol.OfferRevokePayload{
		OfferRef: offer, OfferCommitment: supersededOffer.Commitment, OfferDocumentDigest: digest("offer-revoke-document"), AuthoritySequence: "12",
		ReasonDigest: digest("offer-reason"), Visibility: "PUBLIC",
	}
	if err := b.signRecord("offer-revoke", protocol.KindAgentToolOffer, protocol.ActionRevoke, offer, "3", stringPtr(supersededOffer.Commitment), b.controller, b.key1, revokeOffer, "VERIFY_RECORD", "ACCEPT", "VALID_LIFECYCLE_POINTER"); err != nil {
		return err
	}

	wake := ref("wake-contract")
	wakeCheckpoint := protocol.WakeCheckpointPayload{
		PublicContractProtocol: "agenttool.public-wake-contract/0.1", PublicContractSchemaDigest: digest("wake-schema"), ContractRoot: digest("wake-contract-root"),
		CapabilityRoot: digest("wake-capabilities"), PricingRoot: digest("wake-pricing"), ProtocolsRoot: digest("wake-protocols"), BoundariesRoot: digest("wake-boundaries"), AuthoritySequence: "20",
	}
	if err := b.signRecord("wake-checkpoint", protocol.KindWakePublicCheckpoint, protocol.ActionCheckpoint, wake, "1", nil, b.controller, b.key1, wakeCheckpoint, "VERIFY_RECORD", "ACCEPT", "VALID_PUBLIC_SOURCE_BINDING"); err != nil {
		return err
	}
	wake1 := b.records["wake-checkpoint"]
	wakeSupersede := protocol.WakeSupersedePayload{
		PublicContractProtocol: wakeCheckpoint.PublicContractProtocol, PublicContractSchemaDigest: wakeCheckpoint.PublicContractSchemaDigest,
		ContractRoot: digest("wake-contract-root-2"), CapabilityRoot: digest("wake-capabilities-2"), PricingRoot: digest("wake-pricing-2"),
		ProtocolsRoot: digest("wake-protocols-2"), BoundariesRoot: digest("wake-boundaries-2"), AuthoritySequence: "21", Supersedes: wake1.Commitment,
	}
	if err := b.signRecord("wake-supersede", protocol.KindWakePublicCheckpoint, protocol.ActionSupersede, wake, "2", stringPtr(wake1.Commitment), b.controller, b.key1, wakeSupersede, "VERIFY_RECORD", "ACCEPT", "VALID_LIFECYCLE_POINTER"); err != nil {
		return err
	}
	wake2 := b.records["wake-supersede"]
	wakeWithdraw := protocol.WakeWithdrawPayload{
		CheckpointCommitment: wake2.Commitment, WithdrawalDocumentDigest: digest("wake-withdrawal-document"), AuthoritySequence: "22",
		ReasonDigest: digest("wake-reason"), Visibility: "PUBLIC",
	}
	if err := b.signRecord("wake-withdraw", protocol.KindWakePublicCheckpoint, protocol.ActionWithdraw, wake, "3", stringPtr(wake2.Commitment), b.controller, b.key1, wakeWithdraw, "VERIFY_RECORD", "ACCEPT", "VALID_PUBLIC_SOURCE_BINDING"); err != nil {
		return err
	}

	lineage := ref("downstream")
	lineagePayload := protocol.ArtifactLineagePayload{UpstreamRef: ref("upstream"), DownstreamRef: lineage, Relation: "DERIVES_FROM", EvidenceDigest: digest("lineage-evidence")}
	if err := b.signRecord("artifact-lineage", protocol.KindArtifactLineage, protocol.ActionCheckpoint, lineage, "1", nil, b.controller, b.key1, lineagePayload, "VERIFY_RECORD", "ACCEPT", "BOUNDED_LINEAGE_RELATION"); err != nil {
		return err
	}

	workspace := ref("workspace")
	collabPayload := protocol.CollaborationCheckpointPayload{
		WorkspaceRef: workspace, EpochRef: digest("collab-epoch"), EventHeadSequence: "10", EventHeadHash: digest("event-head"), EventCount: "10", ParticipantSetRoot: digest("participant-set"),
	}
	if err := b.signRecord("collaboration-checkpoint", protocol.KindCollaborationCheckpoint, protocol.ActionCheckpoint, workspace, "1", nil, b.controller, b.key1, collabPayload, "VERIFY_RECORD", "ACCEPT", "JOURNAL_EVENTS_NOT_CONTRIBUTIONS"); err != nil {
		return err
	}

	dispute := ref("dispute")
	disputePayload := protocol.DisputeTerminalPayload{SettlementCommitment: digest("settlement"), Outcome: "SPLIT", DecisionDigest: digest("decision"), DistributionRoot: digest("distribution")}
	if err := b.signRecord("dispute-terminal", protocol.KindDisputeTerminal, protocol.ActionSettle, dispute, "1", nil, b.controller, b.key1, disputePayload, "VERIFY_RECORD", "ACCEPT", "DECLARED_TERMINAL_OUTCOME_ONLY"); err != nil {
		return err
	}

	key1Fingerprint, _ := protocol.KeyFingerprint(b.key1.Public().(ed25519.PublicKey))
	key2Fingerprint, _ := protocol.KeyFingerprint(b.key2.Public().(ed25519.PublicKey))
	rotatePayload := protocol.KeyRotatePayload{PreviousKeyFingerprint: key1Fingerprint, NextKeyFingerprint: key2Fingerprint, RotationDigest: digest("rotation-document")}
	if err := b.signRecord("issuer-key-rotate", protocol.KindIssuerKeyContinuity, protocol.ActionRotate, b.controller, "1", nil, b.controller, b.key1, rotatePayload, "VERIFY_RECORD", "ACCEPT", "PUBLISHER_KEY_CONTINUITY_ONLY"); err != nil {
		return err
	}
	rotate := b.records["issuer-key-rotate"]
	revokeKeyPayload := protocol.KeyRevokePayload{RevokedKeyFingerprint: key2Fingerprint, ReasonDigest: digest("key-reason")}
	if err := b.signRecord("issuer-key-revoke", protocol.KindIssuerKeyContinuity, protocol.ActionRevoke, b.controller, "2", stringPtr(rotate.Commitment), b.controller, b.key2, revokeKeyPayload, "VERIFY_RECORD", "ACCEPT", "PUBLISHER_KEY_CONTINUITY_ONLY"); err != nil {
		return err
	}

	if err := b.addCanonicalVectors(); err != nil {
		return err
	}
	if err := b.addDerivationVector(consumeEnvelope, consumePayload); err != nil {
		return err
	}
	if err := b.addSimulations(grantPayload, consumePayload, rotatePayload, revokeKeyPayload); err != nil {
		return err
	}
	if err := b.addInvalidVectors(gapBatch); err != nil {
		return err
	}
	b.addExpectationIndex()
	return nil
}

type expectationIndex struct {
	Protocol string   `json:"protocol"`
	Vectors  []Vector `json:"vectors"`
}

func (b *builder) addExpectationIndex() {
	vectors := append([]Vector(nil), b.vectors...)
	for i := range vectors {
		sum := sha256.Sum256(b.files[vectors[i].Path])
		vectors[i].FileSHA256 = digestSum(sum)
	}
	sort.Slice(vectors, func(i, j int) bool { return vectors[i].Path < vectors[j].Path })
	b.add("expectations.json", canonicalValue(expectationIndex{Protocol: protocol.Protocol, Vectors: vectors}), Vector{
		Operation: "CHECK_EXPECTATION_INDEX", Expected: "ACCEPT", Stage: "CORPUS", Code: "PINNED_VECTOR_EXPECTATIONS",
	})
}

func (b *builder) addCanonicalVectors() error {
	input := []byte(" {\n  \"𐀀\":\"astral:\\ud800\\udc00\",\n  \"�\":\"replacement:�\",\n  \"\":\"bmp:\",\n  \"line\":\"<>&\\\"\\\\\\u0008\\u000c\\u000a\\u000d\\u0009\\u0001\\u001f  \"\n}\n")
	canonical, err := protocol.CanonicalJSON(input)
	if err != nil {
		return fmt.Errorf("canonical profile fixture: %w", err)
	}
	hexValue := hex.EncodeToString(canonical)
	b.add("canonical/unicode-utf8-order.input.json", input, Vector{Operation: "CANONICALIZE", Expected: "ACCEPT", Stage: "CANONICALIZATION", Code: "UTF8_SORT_AND_MINIMAL_ESCAPES", CanonicalHex: hexValue})
	b.add("canonical/unicode-utf8-order.canonical.json", canonical, Vector{Operation: "CANONICAL_WIRE", Expected: "ACCEPT", Stage: "CANONICALIZATION", Code: "EXACT_CANONICAL_BYTES", CanonicalHex: hexValue})
	return nil
}

type nullifierVector struct {
	Protocol                  string `json:"protocol"`
	Audience                  string `json:"audience"`
	SubjectRef                string `json:"subject_ref"`
	CapabilityRef             string `json:"capability_ref"`
	GrantCommitment           string `json:"grant_commitment"`
	AssetRef                  string `json:"asset_ref"`
	AlternativeAssetRef       string `json:"alternative_asset_ref"`
	SourceEventDigest         string `json:"source_event_digest"`
	SequenceA                 string `json:"sequence_a"`
	SequenceB                 string `json:"sequence_b"`
	NullifierA                string `json:"nullifier_a"`
	NullifierB                string `json:"nullifier_b"`
	AlternativeAssetNullifier string `json:"alternative_asset_nullifier"`
}

func (b *builder) addDerivationVector(envelope protocol.Envelope, consume protocol.CapabilityConsumePayload) error {
	alternative := consume
	alternative.AssetRef = digest("alternative-asset")
	envelopeB := envelope
	envelopeB.Sequence = "3"
	nullifierB, err := protocol.CapabilityNullifier(envelopeB, consume)
	if err != nil {
		return err
	}
	alternativeNullifier, err := protocol.CapabilityNullifier(envelope, alternative)
	if err != nil {
		return err
	}
	vector := nullifierVector{
		Protocol: protocol.Protocol, Audience: envelope.Audience, SubjectRef: envelope.SubjectRef, CapabilityRef: consume.CapabilityRef,
		GrantCommitment: consume.GrantCommitment, AssetRef: consume.AssetRef, AlternativeAssetRef: alternative.AssetRef,
		SourceEventDigest: consume.SourceEventDigest, SequenceA: envelope.Sequence, SequenceB: envelopeB.Sequence,
		NullifierA: consume.Nullifier, NullifierB: nullifierB, AlternativeAssetNullifier: alternativeNullifier,
	}
	b.add("derivations/capability-nullifier.json", canonicalValue(vector), Vector{Operation: "CHECK_NULLIFIER_DERIVATION", Expected: "ACCEPT", Stage: "DERIVATION", Code: "SEQUENCE_INDEPENDENT_ASSET_BOUND_NULLIFIER"})
	return nil
}

func (b *builder) addSimulations(grantPayload protocol.CapabilityGrantPayload, consumePayload protocol.CapabilityConsumePayload, rotatePayload protocol.KeyRotatePayload, revokeKeyPayload protocol.KeyRevokePayload) error {
	ordered := []string{
		"kingdom-release-root", "settlement-root-0001", "settlement-root-0002-cross-batch-replay",
		"capability-grant", "capability-consume", "capability-revoke", "capability-grant-uint64-max",
		"recognition-adopt", "recognition-withdraw", "offer-publish", "offer-supersede", "offer-revoke",
		"wake-checkpoint", "wake-supersede", "wake-withdraw", "artifact-lineage", "collaboration-checkpoint", "dispute-terminal",
		"issuer-key-rotate", "issuer-key-revoke",
	}
	all := make([][]byte, 0, len(ordered))
	for _, name := range ordered {
		all = append(all, b.wires[name])
	}
	b.add("simulation.json", simulationDocument(all...), Vector{Operation: "SIMULATE", Expected: "ACCEPT", Stage: "SIMULATION", Code: "ALL_LIFECYCLES", AcceptedRecords: "20", PermanentNullifierCount: "1"})

	settlementReplay := simulationDocument(b.wires["settlement-root-0001"], b.wires["settlement-root-0002-cross-batch-replay"])
	b.add("adversarial/cross-batch-receipt-replay.json", settlementReplay, Vector{
		Operation: "SIMULATE_SETTLEMENT_REPLAY_AND_ACTIVATION_AUDIT", Expected: "ACCEPT_STRUCTURALLY_BLOCKED_ACTIVATION", Stage: "ACTIVATION",
		Code: "CROSS_BATCH_RECEIPT_REPLAY_NOT_DETECTABLE_V0", AcceptedRecords: "2", PermanentNullifierCount: "0",
	})

	grant := b.records["capability-grant"]
	consume := b.records["capability-consume"]
	replayEnvelope := b.envelope(protocol.KindAgentToolCapability, protocol.ActionConsume, grantPayload.CapabilityRef, "3", stringPtr(consume.Commitment), b.controller, b.key1)
	replayPayload := consumePayload
	replayPayload.Nullifier, _ = protocol.CapabilityNullifier(replayEnvelope, replayPayload)
	_, replayWire, err := protocol.SignRecord(replayEnvelope, canonicalValue(replayPayload), b.key1)
	if err != nil {
		return err
	}
	b.add("invalid/simulation-nullifier-replay.json", simulationDocument(b.wires["capability-grant"], b.wires["capability-consume"], replayWire), Vector{Operation: "SIMULATE", Expected: "REJECT", Stage: "SIMULATION", Code: "PERMANENT_NULLIFIER_REPLAY", ErrorContains: "permanent nullifier replay"})

	hijackerKey := fixedKey(9)
	hijackerController := ref("hijacker-controller")
	hijackEnvelope := b.envelope(protocol.KindAgentToolCapability, protocol.ActionConsume, grantPayload.CapabilityRef, "2", stringPtr(grant.Commitment), hijackerController, hijackerKey)
	hijackPayload := consumePayload
	hijackPayload.AmountMinor = "1"
	hijackPayload.SourceEventDigest = digest("hijack-source")
	hijackPayload.Nullifier, _ = protocol.CapabilityNullifier(hijackEnvelope, hijackPayload)
	_, hijackWire, err := protocol.SignRecord(hijackEnvelope, canonicalValue(hijackPayload), hijackerKey)
	if err != nil {
		return err
	}
	b.add("invalid/simulation-controller-takeover.json", simulationDocument(b.wires["capability-grant"], hijackWire), Vector{Operation: "SIMULATE", Expected: "REJECT", Stage: "SIMULATION", Code: "SUBJECT_CONTROLLER_TAKEOVER", ErrorContains: "pinned"})

	lineagePayload := protocol.ArtifactLineagePayload{UpstreamRef: ref("cross-kind-upstream"), DownstreamRef: grantPayload.CapabilityRef, Relation: "DERIVES_FROM", EvidenceDigest: digest("cross-kind-evidence")}
	lineageEnvelope := b.envelope(protocol.KindArtifactLineage, protocol.ActionCheckpoint, grantPayload.CapabilityRef, "2", stringPtr(grant.Commitment), b.controller, b.key1)
	_, lineageWire, err := protocol.SignRecord(lineageEnvelope, canonicalValue(lineagePayload), b.key1)
	if err != nil {
		return err
	}
	b.add("invalid/simulation-cross-kind-subject.json", simulationDocument(b.wires["capability-grant"], lineageWire), Vector{Operation: "SIMULATE", Expected: "REJECT", Stage: "SIMULATION", Code: "TYPED_SUBJECT_HEAD", ErrorContains: "kind is pinned"})

	rotate := b.records["issuer-key-rotate"]
	oldFingerprint, _ := protocol.KeyFingerprint(b.key1.Public().(ed25519.PublicKey))
	badOldPayload := protocol.KeyRevokePayload{RevokedKeyFingerprint: oldFingerprint, ReasonDigest: digest("old-key-revoke")}
	badOldEnvelope := b.envelope(protocol.KindIssuerKeyContinuity, protocol.ActionRevoke, b.controller, "2", stringPtr(rotate.Commitment), b.controller, b.key1)
	_, badOldWire, err := protocol.SignRecord(badOldEnvelope, canonicalValue(badOldPayload), b.key1)
	if err != nil {
		return err
	}
	b.add("invalid/simulation-key-rotation-old-key.json", simulationDocument(b.wires["issuer-key-rotate"], badOldWire), Vector{Operation: "SIMULATE", Expected: "REJECT", Stage: "SIMULATION", Code: "NEXT_KEY_POSSESSION_REQUIRED", ErrorContains: "staged key"})

	revoked := b.records["issuer-key-revoke"]
	thirdEnvelope := b.envelope(protocol.KindIssuerKeyContinuity, protocol.ActionRevoke, b.controller, "3", stringPtr(revoked.Commitment), b.controller, b.key2)
	_, thirdWire, err := protocol.SignRecord(thirdEnvelope, canonicalValue(revokeKeyPayload), b.key2)
	if err != nil {
		return err
	}
	b.add("invalid/simulation-key-after-revoke.json", simulationDocument(b.wires["issuer-key-rotate"], b.wires["issuer-key-revoke"], thirdWire), Vector{Operation: "SIMULATE", Expected: "REJECT", Stage: "SIMULATION", Code: "TERMINAL_KEY_REVOCATION", ErrorContains: "terminally revoked"})
	return nil
}

func (b *builder) addInvalidVectors(gapBatch protocol.SettlementBatch) error {
	// Parser/canonicalization failures.
	b.add("invalid/duplicate-keys.json", []byte(`{"a":1,"a":2}`), rejectVector("CANONICALIZE", "CANONICALIZATION", "DUPLICATE_KEY", "duplicate object key"))
	b.add("invalid/escaped-duplicate-keys.json", []byte(`{"a":1,"\u0061":2}`), rejectVector("CANONICALIZE", "CANONICALIZATION", "DECODED_DUPLICATE_KEY", "duplicate object key"))
	b.add("invalid/lone-surrogate.json", []byte(`{"x":"\ud800"}`), rejectVector("CANONICALIZE", "CANONICALIZATION", "LONE_SURROGATE", "lone high surrogate"))
	b.add("invalid/unpaired-surrogate.json", []byte(`{"x":"\ud800x"}`), rejectVector("CANONICALIZE", "CANONICALIZATION", "UNPAIRED_SURROGATE", "high surrogate"))
	b.add("invalid/escaped-nul.json", []byte(`{"x":"\u0000"}`), rejectVector("CANONICALIZE", "CANONICALIZATION", "ESCAPED_U0000", "U+0000"))
	b.add("invalid/raw-nul.json", []byte{'{', '"', 'x', '"', ':', '"', 0, '"', '}'}, rejectVector("CANONICALIZE", "CANONICALIZATION", "RAW_U0000", "invalid character"))
	b.add("invalid/unsafe-integer.json", []byte(`{"x":9007199254740992}`), rejectVector("CANONICALIZE", "CANONICALIZATION", "UNSAFE_BARE_INTEGER", "safe integer"))

	valid := b.wires["kingdom-release-root"]
	b.add("invalid/record-leading-space.json", append([]byte{' '}, valid...), rejectVector("VERIFY_RECORD", "WIRE", "NONCANONICAL_LEADING_SPACE", "wire bytes"))
	b.add("invalid/record-trailing-newline.json", append(append([]byte(nil), valid...), '\n'), rejectVector("VERIFY_RECORD", "WIRE", "NONCANONICAL_TRAILING_BYTE", "wire bytes"))
	b.add("invalid/record-nonminimal-key-escape.json", []byte(strings.Replace(string(valid), `"protocol"`, `"\u0070rotocol"`, 1)), rejectVector("VERIFY_RECORD", "WIRE", "NONMINIMAL_STRING_ESCAPE", "wire bytes"))
	var validRecord protocol.Record
	if err := json.Unmarshal(valid, &validRecord); err != nil {
		return err
	}
	b.add("invalid/record-reordered-keys.json", mustJSONMarshal(validRecord), rejectVector("VERIFY_RECORD", "WIRE", "NONCANONICAL_KEY_ORDER", "wire bytes"))

	// Strict Ed25519 vectors pin canonical prime-subgroup A/R and canonical S.
	identity := append([]byte{1}, make([]byte, 31)...)
	lowOrder, _ := hex.DecodeString("26e8958fc2b227b045c3f489f2ef98f0d5dfac05d3c63339b13802886d53fc85")
	primePoint, _ := new(edwards25519.Point).SetBytes(b.key1.Public().(ed25519.PublicKey))
	torsionPoint, _ := new(edwards25519.Point).SetBytes(lowOrder)
	mixed := new(edwards25519.Point).Add(primePoint, torsionPoint).Bytes()
	noncanonical := mustHex("eeffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff7f")
	b.add("invalid/signature-identity-forgery.json", maliciousPublicRecord(validRecord, identity, identity, make([]byte, 32)), rejectVector("VERIFY_RECORD", "SIGNATURE", "ED25519_IDENTITY_FORGERY", "identity"))
	b.add("invalid/signature-small-order-a.json", maliciousPublicRecord(validRecord, lowOrder, identity, make([]byte, 32)), rejectVector("VERIFY_RECORD", "SIGNATURE", "ED25519_SMALL_ORDER_PUBLIC_KEY", "small-order"))
	b.add("invalid/signature-mixed-order-a.json", maliciousPublicRecord(validRecord, mixed, identity, make([]byte, 32)), rejectVector("VERIFY_RECORD", "SIGNATURE", "ED25519_MIXED_ORDER_PUBLIC_KEY", "prime-order subgroup"))
	b.add("invalid/signature-noncanonical-a.json", maliciousPublicRecord(validRecord, noncanonical, identity, make([]byte, 32)), rejectVector("VERIFY_RECORD", "SIGNATURE", "ED25519_NONCANONICAL_PUBLIC_KEY", "noncanonical"))
	noncanonicalR := validRecord
	rawSignature := mustHex(noncanonicalR.Signature.Value)
	copy(rawSignature[:32], noncanonical)
	noncanonicalR.Signature.Value = hex.EncodeToString(rawSignature)
	b.add("invalid/signature-noncanonical-r.json", canonicalValue(noncanonicalR), rejectVector("VERIFY_RECORD", "SIGNATURE", "ED25519_NONCANONICAL_R", "signature R encoding is noncanonical"))
	noncanonicalS := validRecord
	rawSignature = mustHex(noncanonicalS.Signature.Value)
	orderBigEndian := mustBigOrderBytes()
	copy(rawSignature[32:], orderBigEndian)
	noncanonicalS.Signature.Value = hex.EncodeToString(rawSignature)
	b.add("invalid/signature-noncanonical-s.json", canonicalValue(noncanonicalS), rejectVector("VERIFY_RECORD", "SIGNATURE", "ED25519_NONCANONICAL_S", "canonical Ed25519 scalar"))
	tampered := mutateRecord(valid, func(top map[string]any) { top["signature"].(map[string]any)["value"] = strings.Repeat("0", 128) })
	b.add("invalid/signature-tampered.json", tampered, rejectVector("VERIFY_RECORD", "SIGNATURE", "ED25519_SIGNATURE_TAMPER", "signature"))

	// Closed envelope/payload and semantic binding failures.
	fingerprintRecord := validRecord
	fingerprintRecord.Envelope.Issuer.KeyFingerprint = "ed25519-sha256:" + strings.Repeat("0", 64)
	fingerprintRecord.Commitment, _ = protocol.Commitment(fingerprintRecord.Envelope)
	commitmentRaw := mustHex(strings.TrimPrefix(fingerprintRecord.Commitment, "sha256:"))
	fingerprintRecord.Signature.Value = hex.EncodeToString(ed25519.Sign(b.key1, commitmentRaw))
	b.add("invalid/fingerprint-mismatch.json", canonicalValue(fingerprintRecord), rejectVector("VERIFY_RECORD", "SIGNATURE", "KEY_FINGERPRINT_MISMATCH", "fingerprint mismatch"))
	b.add("invalid/schema-hash-mismatch.json", mutateRecord(valid, func(top map[string]any) { top["envelope"].(map[string]any)["schema_hash"] = digest("wrong-schema") }), rejectVector("VERIFY_RECORD", "ENVELOPE", "SCHEMA_HASH_MISMATCH", "schema_hash mismatch"))
	b.add("invalid/payload-missing-field.json", mutateRecord(valid, func(top map[string]any) { delete(top["payload"].(map[string]any), "entry_count") }), rejectVector("VERIFY_RECORD", "PAYLOAD", "MISSING_REQUIRED_PAYLOAD_FIELD", "missing required"))
	b.add("invalid/payload-extra-field.json", mutateRecord(valid, func(top map[string]any) { top["payload"].(map[string]any)["unexpected"] = true }), rejectVector("VERIFY_RECORD", "PAYLOAD", "UNKNOWN_PAYLOAD_FIELD", "unknown field"))
	b.add("invalid/effects-nonzero.json", mutateRecord(valid, func(top map[string]any) {
		top["envelope"].(map[string]any)["effects"].(map[string]any)["storage_writes"] = float64(1)
	}), rejectVector("VERIFY_RECORD", "ENVELOPE", "NONZERO_EFFECT", "zero-effect"))
	b.add("invalid/effects-scope-mismatch.json", mutateRecord(valid, func(top map[string]any) {
		top["envelope"].(map[string]any)["effects"].(map[string]any)["scope"] = "FUTURE_CHAIN_CARRIER"
	}), rejectVector("VERIFY_RECORD", "ENVELOPE", "EFFECT_SCOPE_MISMATCH", "zero-effect"))
	b.add("invalid/nonclaims-missing.json", mutateRecord(valid, func(top map[string]any) {
		nonclaims := top["envelope"].(map[string]any)["nonclaims"].([]any)
		top["envelope"].(map[string]any)["nonclaims"] = nonclaims[:len(nonclaims)-1]
	}), rejectVector("VERIFY_RECORD", "ENVELOPE", "NONCLAIM_SET_MISMATCH", "nonclaims"))
	b.add("invalid/subject-ref-mismatch.json", mutateRecord(b.wires["capability-grant"], func(top map[string]any) { top["envelope"].(map[string]any)["subject_ref"] = ref("forked-subject") }), rejectVector("VERIFY_RECORD", "PAYLOAD", "STABLE_SUBJECT_MISMATCH", "subject_ref"))
	b.add("invalid/parent-pointer-mismatch.json", mutateRecord(b.wires["recognition-withdraw"], func(top map[string]any) {
		top["payload"].(map[string]any)["adoption_commitment"] = digest("wrong-parent")
	}), rejectVector("VERIFY_RECORD", "PAYLOAD", "LIFECYCLE_POINTER_MISMATCH", "envelope.parent"))
	b.add("invalid/uint64-overflow-record.json", mutateRecord(b.wires["capability-grant-uint64-max"], func(top map[string]any) { top["payload"].(map[string]any)["max_total_minor"] = "18446744073709551616" }), rejectVector("VERIFY_RECORD", "PAYLOAD", "UINT64_DECIMAL_OVERFLOW", "uint64"))
	b.add("invalid/multi-asset-consume.json", mutateRecord(b.wires["capability-consume"], func(top map[string]any) {
		top["payload"].(map[string]any)["asset_refs"] = []any{digest("asset"), digest("asset-2")}
	}), rejectVector("VERIFY_RECORD", "PAYLOAD", "MULTI_ASSET_OUTSIDE_SCOPE", "unknown field"))
	b.add("invalid/collaboration-incomplete-prefix.json", mutateRecord(b.wires["collaboration-checkpoint"], func(top map[string]any) {
		top["payload"].(map[string]any)["event_count"] = "9"
	}), rejectVector("VERIFY_RECORD", "PAYLOAD", "COLLABORATION_PREFIX_COUNT_MISMATCH", "must equal event_count"))
	b.add("invalid/settlement-declared-gaps-null.json", mutateRecord(b.wires["settlement-root-0001"], func(top map[string]any) { top["payload"].(map[string]any)["declared_gaps"] = nil }), rejectVector("VERIFY_RECORD", "PAYLOAD", "NULL_DECLARED_GAPS", "non-null JSON array"))
	b.add("invalid/settlement-genesis-prefix-omission.json", mutateRecord(b.wires["settlement-root-0001"], func(top map[string]any) {
		payload := top["payload"].(map[string]any)
		payload["first_sequence"], payload["last_sequence"], payload["receipt_count"], payload["declared_gaps"] = "42", "42", "1", []any{}
	}), rejectVector("VERIFY_RECORD", "PAYLOAD", "SETTLEMENT_GENESIS_MUST_START_AT_ONE", "first_sequence 1"))

	duplicateBatch := gapBatch
	duplicateBatch.Leaves = append([]protocol.SettlementLeaf(nil), gapBatch.Leaves...)
	duplicateBatch.Leaves[1].ReceiptDigest = duplicateBatch.Leaves[0].ReceiptDigest
	b.add("invalid/batch-duplicate-receipt.json", canonicalValue(duplicateBatch), rejectVector("VERIFY_BATCH", "BATCH", "DUPLICATE_RECEIPT_IN_BATCH", "duplicates"))
	overflowBatch := gapBatch
	overflowBatch.FirstSequence = "18446744073709551616"
	b.add("invalid/batch-uint64-overflow.json", canonicalValue(overflowBatch), rejectVector("VERIFY_BATCH", "BATCH", "UINT64_DECIMAL_OVERFLOW", "uint64"))
	b.add("invalid/batch-missing-declared-gaps.json", mutateObject(canonicalValue(gapBatch), func(top map[string]any) { delete(top, "declared_gaps") }), rejectVector("VERIFY_BATCH", "BATCH", "MISSING_DECLARED_GAPS", "missing required"))
	b.add("invalid/batch-trailing-newline.json", append(canonicalValue(gapBatch), '\n'), rejectVector("VERIFY_BATCH", "WIRE", "NONCANONICAL_TRAILING_BYTE", "wire bytes"))
	return nil
}

func (b *builder) signRecord(name string, kind protocol.Kind, action protocol.Action, subject, sequence string, parent *string, controller string, key ed25519.PrivateKey, payload any, operation, expected, code string) error {
	envelope := b.envelope(kind, action, subject, sequence, parent, controller, key)
	return b.signPrepared(name, envelope, key, payload, operation, expected, code)
}

func (b *builder) signPrepared(name string, envelope protocol.Envelope, key ed25519.PrivateKey, payload any, operation, expected, code string) error {
	if operation == "VERIFY_RECORD" && expected == "ACCEPT" {
		operation = "VERIFY_RECORD_AND_ACTIVATION_AUDIT"
		expected = "ACCEPT_STRUCTURALLY_BLOCKED_ACTIVATION"
	}
	record, wire, err := protocol.SignRecord(envelope, canonicalValue(payload), key)
	if err != nil {
		return fmt.Errorf("sign %s: %w", name, err)
	}
	b.records[name], b.wires[name] = record, wire
	b.add("records/"+name+".json", wire, Vector{
		Operation: operation, Expected: expected, Stage: stageForExpected(expected), Code: code,
		Kind: record.Envelope.Kind, Action: record.Envelope.Action, Commitment: record.Commitment, PayloadRoot: record.Envelope.PayloadRoot,
	})
	return nil
}

func (b *builder) envelope(kind protocol.Kind, action protocol.Action, subject, sequence string, parent *string, controller string, key ed25519.PrivateKey) protocol.Envelope {
	fingerprint, _ := protocol.KeyFingerprint(key.Public().(ed25519.PublicKey))
	return protocol.Envelope{
		Kind: kind, Action: action, Audience: "kingdom:offline-shadow", SubjectRef: subject, Sequence: sequence, Parent: parent,
		Issuer:       protocol.Issuer{Namespace: "witness-fixture", ControllerRef: controller, KeyFingerprint: fingerprint},
		PolicyDigest: digest("policy-v0"), ExpiryHeight: nil,
		Effects:   protocol.Effects{Scope: "RECORD_CONSTRUCTION_AND_OFFLINE_VALIDATION_ONLY", Authority: "NONE", Economic: "NONE", Reputation: "NONE"},
		Nonclaims: protocol.RequiredNonclaims(),
	}
}

func (b *builder) add(path string, contents []byte, vector Vector) {
	if _, exists := b.files[path]; exists {
		panic("duplicate fixture path " + path)
	}
	copyBytes := append([]byte(nil), contents...)
	b.files[path] = copyBytes
	vector.Path = path
	b.vectors = append(b.vectors, vector)
}

func (b *builder) finish() (Corpus, error) {
	sort.Slice(b.vectors, func(i, j int) bool { return b.vectors[i].Path < b.vectors[j].Path })
	for i := range b.vectors {
		sum := sha256.Sum256(b.files[b.vectors[i].Path])
		b.vectors[i].FileSHA256 = digestSum(sum)
	}
	payloadHashes, err := protocol.SchemaHashes()
	if err != nil {
		return Corpus{}, err
	}
	schemas := make([]SchemaEntry, 0, len(payloadHashes))
	for kind, hash := range payloadHashes {
		schemas = append(schemas, SchemaEntry{Kind: kind, SchemaHash: hash})
	}
	sort.Slice(schemas, func(i, j int) bool { return schemas[i].Kind < schemas[j].Kind })
	recordHash, err := protocol.RecordSchemaHash()
	if err != nil {
		return Corpus{}, err
	}
	batchHash, err := protocol.SettlementBatchSchemaHash()
	if err != nil {
		return Corpus{}, err
	}
	schemaSet, err := protocol.SchemaSetDigest()
	if err != nil {
		return Corpus{}, err
	}
	emptyRoot, _ := protocol.SettlementMerkleRoot(nil)
	_, gapRoot, _ := protocol.VerifySettlementBatch(b.files["batches/settlement-with-gap.json"])
	unicodeCanonical := b.files["canonical/unicode-utf8-order.canonical.json"]
	unicodeSum := sha256.Sum256(unicodeCanonical)
	manifest := Manifest{
		Protocol: protocol.Protocol, FreezeState: FreezeState, WirePolicy: "EXACT_CANONICAL_JSON_NO_TRAILING_BYTES", ActionPairCount: "18",
		RecordSchemaHash: recordHash, SettlementBatchSchemaHash: batchHash, SchemaSetDigest: schemaSet, PayloadSchemas: schemas,
		Vectors: b.vectors, CorpusDigestAlgorithm: "SHA256(protocol || NUL || known-answer-corpus || NUL || repeated(sorted_path || NUL || raw32(file_sha256) || NUL))",
		CorpusDigest: corpusDigest(b.vectors), EmptyMerkleRoot: emptyRoot, SettlementMerkleRoot: gapRoot,
		CapabilityNullifier:    b.records["capability-consume"].Envelope.PayloadRoot, // replaced below with the semantic nullifier
		UnicodeCanonicalSHA256: digestSum(unicodeSum), UnicodeCanonicalHex: hex.EncodeToString(unicodeCanonical),
	}
	var consume protocol.CapabilityConsumePayload
	if err := json.Unmarshal(b.records["capability-consume"].Payload, &consume); err != nil {
		return Corpus{}, err
	}
	manifest.CapabilityNullifier = consume.Nullifier
	manifestBytes := canonicalValue(manifest)
	b.files["known-answer.json"] = manifestBytes
	return Corpus{Files: b.files, Manifest: manifest}, nil
}

func WriteDir(dir string) (Corpus, error) {
	if err := validateExplicitDir(dir); err != nil {
		return Corpus{}, err
	}
	corpus, err := Build()
	if err != nil {
		return Corpus{}, err
	}
	info, err := os.Lstat(dir)
	if err != nil && !os.IsNotExist(err) {
		return Corpus{}, err
	}
	if err == nil && (info.Mode()&os.ModeSymlink != 0 || !info.IsDir()) {
		return Corpus{}, fmt.Errorf("destination must be a real directory, not a symlink or non-directory")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return Corpus{}, err
	}
	paths := sortedFilePaths(corpus.Files)
	for _, relative := range paths {
		target := filepath.Join(dir, filepath.FromSlash(relative))
		if existing, err := os.Lstat(target); err == nil {
			if existing.Mode()&os.ModeSymlink != 0 || !existing.Mode().IsRegular() {
				return Corpus{}, fmt.Errorf("refusing non-regular existing fixture %q", target)
			}
		} else if !os.IsNotExist(err) {
			return Corpus{}, err
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return Corpus{}, err
		}
		if err := os.WriteFile(target, corpus.Files[relative], 0o644); err != nil {
			return Corpus{}, err
		}
	}
	return corpus, nil
}

func CheckDir(dir string) error {
	if err := validateExplicitDir(dir); err != nil {
		return err
	}
	corpus, err := Build()
	if err != nil {
		return err
	}
	actual := make(map[string][]byte)
	err = filepath.WalkDir(dir, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == dir {
			return nil
		}
		relative, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		relative = filepath.ToSlash(relative)
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if entry.Type()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			return fmt.Errorf("fixture %q is not a regular non-symlink file", relative)
		}
		contents, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		actual[relative] = contents
		return nil
	})
	if err != nil {
		return err
	}
	for path, expected := range corpus.Files {
		got, ok := actual[path]
		if !ok {
			return fmt.Errorf("missing generated fixture %q", path)
		}
		if !bytes.Equal(got, expected) {
			return fmt.Errorf("generated fixture drift at %q", path)
		}
		delete(actual, path)
	}
	if len(actual) != 0 {
		extras := make([]string, 0, len(actual))
		for path := range actual {
			extras = append(extras, path)
		}
		sort.Strings(extras)
		return fmt.Errorf("untracked fixture files: %s", strings.Join(extras, ", "))
	}
	return nil
}

func validateExplicitDir(dir string) error {
	if strings.TrimSpace(dir) == "" {
		return fmt.Errorf("an explicit destination directory is required")
	}
	clean := filepath.Clean(dir)
	if clean == "." || clean == string(filepath.Separator) {
		return fmt.Errorf("refusing broad destination %q", dir)
	}
	return nil
}

func rejectVector(operation, stage, code, contains string) Vector {
	return Vector{Operation: operation, Expected: "REJECT", Stage: stage, Code: code, ErrorContains: contains}
}

func stageForExpected(expected string) string {
	if expected == "ACCEPT_STRUCTURALLY_BLOCKED_ACTIVATION" {
		return "ACTIVATION"
	}
	return "RECORD"
}

func fixedKey(fill byte) ed25519.PrivateKey {
	seed := bytes.Repeat([]byte{fill}, ed25519.SeedSize)
	return ed25519.NewKeyFromSeed(seed)
}

func ref(label string) string {
	sum := sha256.Sum256([]byte("ref/" + label))
	return hex.EncodeToString(sum[:])
}

func digest(label string) string {
	sum := sha256.Sum256([]byte("digest/" + label))
	return digestSum(sum)
}

func digestSum(sum [32]byte) string { return "sha256:" + hex.EncodeToString(sum[:]) }

func stringPtr(value string) *string { return &value }

func canonicalValue(value any) []byte {
	encoded := mustJSONMarshal(value)
	canonical, err := protocol.CanonicalJSON(encoded)
	if err != nil {
		panic(err)
	}
	return canonical
}

func mustJSONMarshal(value any) []byte {
	encoded, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return encoded
}

func mutateRecord(input []byte, mutate func(map[string]any)) []byte {
	return mutateObject(input, mutate)
}

func mutateObject(input []byte, mutate func(map[string]any)) []byte {
	var top map[string]any
	dec := json.NewDecoder(bytes.NewReader(input))
	dec.UseNumber()
	if err := dec.Decode(&top); err != nil {
		panic(err)
	}
	mutate(top)
	return canonicalValue(top)
}

func simulationDocument(records ...[]byte) []byte {
	raw := make([]json.RawMessage, len(records))
	for i := range records {
		raw[i] = append(json.RawMessage(nil), records[i]...)
	}
	return canonicalValue(struct {
		Records []json.RawMessage `json:"records"`
	}{raw})
}

func maliciousPublicRecord(base protocol.Record, publicKey, r, s []byte) []byte {
	record := base
	fingerprint, _ := protocol.KeyFingerprint(publicKey)
	record.Envelope.Issuer.KeyFingerprint = fingerprint
	record.Commitment, _ = protocol.Commitment(record.Envelope)
	record.Signature.PublicKey = hex.EncodeToString(publicKey)
	record.Signature.Value = hex.EncodeToString(append(append([]byte(nil), r...), s...))
	return canonicalValue(record)
}

func mustHex(value string) []byte {
	decoded, err := hex.DecodeString(value)
	if err != nil {
		panic(err)
	}
	return decoded
}

func mustBigOrderBytes() []byte {
	// l in canonical little-endian scalar encoding.
	bigEndian := mustHex("1000000000000000000000000000000014def9dea2f79cd65812631a5cf5d3ed")
	result := make([]byte, 32)
	for i := range bigEndian {
		result[i] = bigEndian[len(bigEndian)-1-i]
	}
	return result
}

func corpusDigest(vectors []Vector) string {
	h := sha256.New()
	h.Write([]byte(protocol.Protocol))
	h.Write([]byte{0})
	h.Write([]byte("known-answer-corpus"))
	h.Write([]byte{0})
	for _, vector := range vectors {
		rawHash := mustHex(strings.TrimPrefix(vector.FileSHA256, "sha256:"))
		h.Write([]byte(vector.Path))
		h.Write([]byte{0})
		h.Write(rawHash)
		h.Write([]byte{0})
	}
	var result [32]byte
	copy(result[:], h.Sum(nil))
	return digestSum(result)
}

func sortedFilePaths(files map[string][]byte) []string {
	paths := make([]string, 0, len(files))
	for path := range files {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	return paths
}
