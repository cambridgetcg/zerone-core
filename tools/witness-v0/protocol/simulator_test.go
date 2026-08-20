package protocol

import (
	"crypto/ed25519"
	"encoding/json"
	"strings"
	"testing"
)

func simulationDocument(t *testing.T, records ...[]byte) []byte {
	t.Helper()
	raw := make([]json.RawMessage, len(records))
	for i := range records {
		raw[i] = json.RawMessage(records[i])
	}
	encoded, err := json.Marshal(SimulationInput{Records: raw})
	if err != nil {
		t.Fatal(err)
	}
	return encoded
}

func capabilityChain(t *testing.T, maxPerUse, maxTotal, firstAmount string) (Record, []byte, Record, []byte) {
	t.Helper()
	key := testKey(1)
	capability := testRef("sim-capability")
	grantPayload := CapabilityGrantPayload{
		CapabilityRef: capability, GrantDigest: testDigest("sim-grant"), AssetRef: testDigest("sim-asset"),
		MaxPerConsumeMinor: maxPerUse, MaxTotalMinor: maxTotal,
	}
	grant, grantBytes := seal(t, testEnvelope(KindAgentToolCapability, ActionGrant, capability, key, "1", nil), grantPayload, key)
	consumeEnvelope := testEnvelope(KindAgentToolCapability, ActionConsume, capability, key, "2", &grant.Commitment)
	consumePayload := CapabilityConsumePayload{
		CapabilityRef: capability, GrantCommitment: grant.Commitment, AssetRef: grantPayload.AssetRef,
		AmountMinor: firstAmount, SourceEventDigest: testDigest("sim-source-1"),
	}
	consumePayload.Nullifier, _ = CapabilityNullifier(consumeEnvelope, consumePayload)
	consume, consumeBytes := seal(t, consumeEnvelope, consumePayload, key)
	return grant, grantBytes, consume, consumeBytes
}

func TestSimulatorCapabilityConsumptionAndPermanentNullifier(t *testing.T) {
	grant, grantBytes, consume, consumeBytes := capabilityChain(t, "10", "20", "7")
	result, err := Simulate(simulationDocument(t, grantBytes, consumeBytes))
	if err != nil {
		t.Fatal(err)
	}
	if result.PermanentNullifierCount != "1" || len(result.Capabilities) != 1 || result.Capabilities[0].SpentMinor != "7" {
		t.Fatalf("unexpected result: %#v", result)
	}

	key := testKey(1)
	capability := grant.Envelope.SubjectRef
	replayEnvelope := testEnvelope(KindAgentToolCapability, ActionConsume, capability, key, "3", &consume.Commitment)
	replayPayload := consume.Payload
	var replay CapabilityConsumePayload
	if err := json.Unmarshal(replayPayload, &replay); err != nil {
		t.Fatal(err)
	}
	// Sequence is intentionally different; the semantic source event is not.
	expected, _ := CapabilityNullifier(replayEnvelope, replay)
	if expected != replay.Nullifier {
		t.Fatalf("sequence changed nullifier: %s != %s", expected, replay.Nullifier)
	}
	_, replayBytes := seal(t, replayEnvelope, replay, key)
	if _, err := Simulate(simulationDocument(t, grantBytes, consumeBytes, replayBytes)); err == nil || !strings.Contains(err.Error(), "nullifier replay") {
		t.Fatalf("expected permanent replay rejection, got %v", err)
	}
}

func TestCapabilityNullifierDomainMutations(t *testing.T) {
	key := testKey(1)
	capability := testRef("nullifier-capability")
	envelope := testEnvelope(KindAgentToolCapability, ActionConsume, capability, key, "9", ptr(testDigest("parent")))
	payload := CapabilityConsumePayload{
		CapabilityRef: capability, GrantCommitment: testDigest("grant"), AssetRef: testDigest("asset-a"),
		AmountMinor: "1", SourceEventDigest: testDigest("source"),
	}
	base, err := CapabilityNullifier(envelope, payload)
	if err != nil {
		t.Fatal(err)
	}
	sequenceChanged := envelope
	sequenceChanged.Sequence = "10"
	if got, _ := CapabilityNullifier(sequenceChanged, payload); got != base {
		t.Fatal("sequence must not affect nullifier")
	}
	assetChanged := payload
	assetChanged.AssetRef = testDigest("asset-b")
	if got, _ := CapabilityNullifier(envelope, assetChanged); got == base {
		t.Fatal("asset_ref must affect nullifier")
	}
	audienceChanged := envelope
	audienceChanged.Audience = "zerone:zerone-1"
	if got, _ := CapabilityNullifier(audienceChanged, payload); got == base {
		t.Fatal("audience must affect nullifier")
	}
	sourceChanged := payload
	sourceChanged.SourceEventDigest = testDigest("source-b")
	if got, _ := CapabilityNullifier(envelope, sourceChanged); got == base {
		t.Fatal("source event must affect nullifier")
	}
}

func TestSimulatorCapabilityBoundsAndRevocation(t *testing.T) {
	grant, grantBytes, consume, consumeBytes := capabilityChain(t, "10", "15", "10")
	key := testKey(1)
	capability := grant.Envelope.SubjectRef

	secondEnvelope := testEnvelope(KindAgentToolCapability, ActionConsume, capability, key, "3", &consume.Commitment)
	secondPayload := CapabilityConsumePayload{CapabilityRef: capability, GrantCommitment: grant.Commitment, AssetRef: testDigest("sim-asset"), AmountMinor: "6", SourceEventDigest: testDigest("sim-source-2")}
	secondPayload.Nullifier, _ = CapabilityNullifier(secondEnvelope, secondPayload)
	_, secondBytes := seal(t, secondEnvelope, secondPayload, key)
	if _, err := Simulate(simulationDocument(t, grantBytes, consumeBytes, secondBytes)); err == nil || !strings.Contains(err.Error(), "cumulative") {
		t.Fatalf("expected cumulative rejection, got %v", err)
	}

	revokeEnvelope := testEnvelope(KindAgentToolCapability, ActionRevoke, capability, key, "3", &consume.Commitment)
	revokePayload := CapabilityRevokePayload{CapabilityRef: capability, GrantCommitment: grant.Commitment, ReasonDigest: testDigest("revoke")}
	revoke, revokeBytes := seal(t, revokeEnvelope, revokePayload, key)
	result, err := Simulate(simulationDocument(t, grantBytes, consumeBytes, revokeBytes))
	if err != nil {
		t.Fatal(err)
	}
	if !result.Capabilities[0].Revoked {
		t.Fatal("revocation not retained")
	}

	afterEnvelope := testEnvelope(KindAgentToolCapability, ActionConsume, capability, key, "4", &revoke.Commitment)
	afterPayload := CapabilityConsumePayload{CapabilityRef: capability, GrantCommitment: grant.Commitment, AssetRef: testDigest("sim-asset"), AmountMinor: "1", SourceEventDigest: testDigest("after-revoke")}
	afterPayload.Nullifier, _ = CapabilityNullifier(afterEnvelope, afterPayload)
	_, afterBytes := seal(t, afterEnvelope, afterPayload, key)
	if _, err := Simulate(simulationDocument(t, grantBytes, consumeBytes, revokeBytes, afterBytes)); err == nil || !strings.Contains(err.Error(), "revoked capability") {
		t.Fatalf("expected post-revocation rejection, got %v", err)
	}
}

func TestSimulatorAcceptsMaxUint64WithoutOverflow(t *testing.T) {
	_, grantBytes, _, consumeBytes := capabilityChain(t, "18446744073709551615", "18446744073709551615", "18446744073709551615")
	result, err := Simulate(simulationDocument(t, grantBytes, consumeBytes))
	if err != nil {
		t.Fatal(err)
	}
	if result.Capabilities[0].SpentMinor != "18446744073709551615" {
		t.Fatal("max uint64 spend changed")
	}
}

func TestSimulatorKeyRotationProofAndTerminalRevocation(t *testing.T) {
	key1, key2 := testKey(1), testKey(2)
	controller := testRef("controller")
	fp1, _ := KeyFingerprint(key1.Public().(ed25519.PublicKey))
	fp2, _ := KeyFingerprint(key2.Public().(ed25519.PublicKey))
	rotatePayload := KeyRotatePayload{PreviousKeyFingerprint: fp1, NextKeyFingerprint: fp2, RotationDigest: testDigest("rotation")}
	rotate, rotateBytes := seal(t, testEnvelope(KindIssuerKeyContinuity, ActionRotate, controller, key1, "1", nil), rotatePayload, key1)

	revokePayload := KeyRevokePayload{RevokedKeyFingerprint: fp2, ReasonDigest: testDigest("key-revoke")}
	revokeEnvelope := testEnvelope(KindIssuerKeyContinuity, ActionRevoke, controller, key2, "2", &rotate.Commitment)
	revoke, revokeBytes := seal(t, revokeEnvelope, revokePayload, key2)
	result, err := Simulate(simulationDocument(t, rotateBytes, revokeBytes))
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Controllers) != 1 || !result.Controllers[0].Revoked || result.Controllers[0].ActiveKeyFingerprint != fp2 {
		t.Fatalf("unexpected controller result: %#v", result.Controllers)
	}

	badRevokePayload := KeyRevokePayload{RevokedKeyFingerprint: fp1, ReasonDigest: testDigest("old-key-revoke")}
	badEnvelope := testEnvelope(KindIssuerKeyContinuity, ActionRevoke, controller, key1, "2", &rotate.Commitment)
	_, badBytes := seal(t, badEnvelope, badRevokePayload, key1)
	if _, err := Simulate(simulationDocument(t, rotateBytes, badBytes)); err == nil || !strings.Contains(err.Error(), "staged key") {
		t.Fatalf("expected next-key proof rejection, got %v", err)
	}

	thirdEnvelope := testEnvelope(KindIssuerKeyContinuity, ActionRevoke, controller, key2, "3", &revoke.Commitment)
	_, thirdBytes := seal(t, thirdEnvelope, revokePayload, key2)
	if _, err := Simulate(simulationDocument(t, rotateBytes, revokeBytes, thirdBytes)); err == nil || !strings.Contains(err.Error(), "terminally revoked") {
		t.Fatalf("expected terminal revocation, got %v", err)
	}
}

func TestSimulatorPinsIssuerNamespace(t *testing.T) {
	key := testKey(1)
	firstCase := allPayloadCases(t)[16]
	_, first := seal(t, testEnvelope(firstCase.kind, firstCase.action, firstCase.subject, key, "1", nil), firstCase.payload, key)
	secondCase := allPayloadCases(t)[17]
	secondEnvelope := testEnvelope(secondCase.kind, secondCase.action, secondCase.subject, key, "1", nil)
	secondEnvelope.Issuer.Namespace = "relabeled"
	_, second := seal(t, secondEnvelope, secondCase.payload, key)
	if _, err := Simulate(simulationDocument(t, first, second)); err == nil || !strings.Contains(err.Error(), "namespace changed") {
		t.Fatalf("expected namespace pinning rejection, got %v", err)
	}
}

func TestSimulatorRecognitionStableFieldsAndAuthoritySequence(t *testing.T) {
	key := testKey(1)
	recognition := testRef("sim-recognition")
	adoptPayload := RecognitionAdoptPayload{
		RecognitionRef: recognition, SurfaceDigest: testDigest("surface"), RegistryDigest: testDigest("registry"),
		AdoptionDocumentDigest: testDigest("adoption-doc"), AuthoritySequence: "5", Visibility: "PUBLIC",
	}
	adopt, adoptBytes := seal(t, testEnvelope(KindAgentToolPublicRecognition, ActionAdopt, recognition, key, "1", nil), adoptPayload, key)
	baseWithdraw := RecognitionWithdrawPayload{
		RecognitionRef: recognition, AdoptionCommitment: adopt.Commitment, SurfaceDigest: adoptPayload.SurfaceDigest,
		RegistryDigest: adoptPayload.RegistryDigest, WithdrawalDocumentDigest: testDigest("withdraw-doc"),
		AuthoritySequence: "6", ReasonDigest: testDigest("reason"), Visibility: "PUBLIC",
	}
	withdrawEnvelope := testEnvelope(KindAgentToolPublicRecognition, ActionWithdraw, recognition, key, "2", &adopt.Commitment)
	_, withdrawBytes := seal(t, withdrawEnvelope, baseWithdraw, key)
	if _, err := Simulate(simulationDocument(t, adoptBytes, withdrawBytes)); err != nil {
		t.Fatal(err)
	}

	for _, mutate := range []func(*RecognitionWithdrawPayload){
		func(p *RecognitionWithdrawPayload) { p.SurfaceDigest = testDigest("changed-surface") },
		func(p *RecognitionWithdrawPayload) { p.RegistryDigest = testDigest("changed-registry") },
		func(p *RecognitionWithdrawPayload) { p.AuthoritySequence = "5" },
	} {
		payload := baseWithdraw
		mutate(&payload)
		_, bytes := seal(t, withdrawEnvelope, payload, key)
		if _, err := Simulate(simulationDocument(t, adoptBytes, bytes)); err == nil {
			t.Fatal("expected recognition continuity rejection")
		}
	}
}

func TestSimulatorOfferRevisionAndAuthoritySequence(t *testing.T) {
	key := testKey(1)
	offer := testRef("sim-offer")
	publishPayload := OfferPublishPayload{
		OfferRef: offer, OfferDocumentDigest: testDigest("offer-doc-1"), CapabilityRoot: testDigest("cap-root-1"),
		PricingRoot: testDigest("price-root-1"), SLARoot: testDigest("sla-root-1"), TermsDigest: testDigest("terms-1"),
		Revision: "5", AuthoritySequence: "10", Visibility: "PUBLIC",
	}
	publish, publishBytes := seal(t, testEnvelope(KindAgentToolOffer, ActionPublish, offer, key, "1", nil), publishPayload, key)
	supersedePayload := OfferSupersedePayload{
		OfferRef: offer, OfferDocumentDigest: testDigest("offer-doc-2"), CapabilityRoot: testDigest("cap-root-2"),
		PricingRoot: testDigest("price-root-2"), SLARoot: testDigest("sla-root-2"), TermsDigest: testDigest("terms-2"),
		Revision: "6", AuthoritySequence: "11", Visibility: "PUBLIC", Supersedes: publish.Commitment,
	}
	supersedeEnvelope := testEnvelope(KindAgentToolOffer, ActionSupersede, offer, key, "2", &publish.Commitment)
	supersede, supersedeBytes := seal(t, supersedeEnvelope, supersedePayload, key)
	if _, err := Simulate(simulationDocument(t, publishBytes, supersedeBytes)); err != nil {
		t.Fatal(err)
	}

	for _, mutate := range []func(*OfferSupersedePayload){
		func(p *OfferSupersedePayload) { p.Revision = "5" },
		func(p *OfferSupersedePayload) { p.AuthoritySequence = "10" },
	} {
		payload := supersedePayload
		mutate(&payload)
		_, bytes := seal(t, supersedeEnvelope, payload, key)
		if _, err := Simulate(simulationDocument(t, publishBytes, bytes)); err == nil {
			t.Fatal("expected offer monotonicity rejection")
		}
	}

	revokePayload := OfferRevokePayload{
		OfferRef: offer, OfferCommitment: supersede.Commitment, OfferDocumentDigest: testDigest("offer-revoke-doc"),
		AuthoritySequence: "12", ReasonDigest: testDigest("offer-revoke-reason"), Visibility: "PUBLIC",
	}
	revokeEnvelope := testEnvelope(KindAgentToolOffer, ActionRevoke, offer, key, "3", &supersede.Commitment)
	_, revokeBytes := seal(t, revokeEnvelope, revokePayload, key)
	if _, err := Simulate(simulationDocument(t, publishBytes, supersedeBytes, revokeBytes)); err != nil {
		t.Fatal(err)
	}
	revokePayload.AuthoritySequence = "11"
	_, badRevoke := seal(t, revokeEnvelope, revokePayload, key)
	if _, err := Simulate(simulationDocument(t, publishBytes, supersedeBytes, badRevoke)); err == nil {
		t.Fatal("expected revoke authority-sequence rejection")
	}
}

func TestSimulatorWakeAuthoritySequence(t *testing.T) {
	key := testKey(1)
	subject := testRef("sim-wake")
	checkpointPayload := WakeCheckpointPayload{
		PublicContractProtocol: "agenttool.public-wake-contract/0.1", PublicContractSchemaDigest: testDigest("wake-schema"),
		ContractRoot: testDigest("wake-contract-1"), CapabilityRoot: testDigest("wake-cap-1"), PricingRoot: testDigest("wake-price-1"),
		ProtocolsRoot: testDigest("wake-protocol-1"), BoundariesRoot: testDigest("wake-boundary-1"), AuthoritySequence: "20",
	}
	checkpoint, checkpointBytes := seal(t, testEnvelope(KindWakePublicCheckpoint, ActionCheckpoint, subject, key, "1", nil), checkpointPayload, key)
	supersedePayload := WakeSupersedePayload{
		PublicContractProtocol: checkpointPayload.PublicContractProtocol, PublicContractSchemaDigest: checkpointPayload.PublicContractSchemaDigest,
		ContractRoot: testDigest("wake-contract-2"), CapabilityRoot: testDigest("wake-cap-2"), PricingRoot: testDigest("wake-price-2"),
		ProtocolsRoot: testDigest("wake-protocol-2"), BoundariesRoot: testDigest("wake-boundary-2"), AuthoritySequence: "21", Supersedes: checkpoint.Commitment,
	}
	supersedeEnvelope := testEnvelope(KindWakePublicCheckpoint, ActionSupersede, subject, key, "2", &checkpoint.Commitment)
	supersede, supersedeBytes := seal(t, supersedeEnvelope, supersedePayload, key)
	if _, err := Simulate(simulationDocument(t, checkpointBytes, supersedeBytes)); err != nil {
		t.Fatal(err)
	}
	supersedePayload.AuthoritySequence = "20"
	_, badSupersede := seal(t, supersedeEnvelope, supersedePayload, key)
	if _, err := Simulate(simulationDocument(t, checkpointBytes, badSupersede)); err == nil {
		t.Fatal("expected WAKE authority-sequence rejection")
	}

	withdrawPayload := WakeWithdrawPayload{
		CheckpointCommitment: supersede.Commitment, WithdrawalDocumentDigest: testDigest("wake-withdraw-doc"),
		AuthoritySequence: "22", ReasonDigest: testDigest("wake-withdraw-reason"), Visibility: "PUBLIC",
	}
	withdrawEnvelope := testEnvelope(KindWakePublicCheckpoint, ActionWithdraw, subject, key, "3", &supersede.Commitment)
	_, withdrawBytes := seal(t, withdrawEnvelope, withdrawPayload, key)
	if _, err := Simulate(simulationDocument(t, checkpointBytes, supersedeBytes, withdrawBytes)); err != nil {
		t.Fatal(err)
	}
}

func TestSimulatorSettlementRangesAndDeclaredGaps(t *testing.T) {
	key := testKey(1)
	subject := testRef("sim-settlement")
	firstPayload := SettlementRootPayload{
		ReceiptProtocol: "agenttool.settlement-receipt/1", ReceiptSchemaDigest: testDigest("receipt-schema"),
		SourceSequenceBinding: "PROJECTION_ONLY", ReceiptUniquenessScope: "BATCH_ONLY",
		FirstSequence: "1", LastSequence: "3", ReceiptCount: "2", DeclaredGaps: []Gap{{First: "2", Last: "2"}},
		MerkleRoot: testDigest("batch-1"), PreviousBatch: nil,
	}
	first, firstBytes := seal(t, testEnvelope(KindAgentToolSettlementRoot, ActionCheckpoint, subject, key, "1", nil), firstPayload, key)
	secondPayload := SettlementRootPayload{
		ReceiptProtocol: firstPayload.ReceiptProtocol, ReceiptSchemaDigest: firstPayload.ReceiptSchemaDigest,
		SourceSequenceBinding: "PROJECTION_ONLY", ReceiptUniquenessScope: "BATCH_ONLY",
		FirstSequence: "4", LastSequence: "5", ReceiptCount: "2", DeclaredGaps: []Gap{},
		MerkleRoot: testDigest("batch-2"), PreviousBatch: &first.Commitment,
	}
	secondEnvelope := testEnvelope(KindAgentToolSettlementRoot, ActionCheckpoint, subject, key, "2", &first.Commitment)
	_, secondBytes := seal(t, secondEnvelope, secondPayload, key)
	if _, err := Simulate(simulationDocument(t, firstBytes, secondBytes)); err != nil {
		t.Fatal(err)
	}
	secondPayload.FirstSequence = "5"
	secondPayload.ReceiptCount = "1"
	_, badSecond := seal(t, secondEnvelope, secondPayload, key)
	if _, err := Simulate(simulationDocument(t, firstBytes, badSecond)); err == nil || !strings.Contains(err.Error(), "contiguous") {
		t.Fatalf("expected range continuity rejection, got %v", err)
	}
}

func TestSimulatorRejectsSettlementGenesisPrefixOmission(t *testing.T) {
	key := testKey(1)
	tc := allPayloadCases(t)[1]
	_, valid := seal(t, envelopeForCase(t, tc, key), tc.payload, key)
	var record map[string]any
	if err := json.Unmarshal(valid, &record); err != nil {
		t.Fatal(err)
	}
	payload := record["payload"].(map[string]any)
	payload["first_sequence"] = "42"
	payload["last_sequence"] = "42"
	payload["receipt_count"] = "1"
	payload["declared_gaps"] = []any{}
	mutated, _ := json.Marshal(record)
	if _, err := Simulate(simulationDocument(t, mutated)); err == nil || !strings.Contains(err.Error(), "first_sequence 1") {
		t.Fatalf("expected simulator genesis-prefix rejection, got %v", err)
	}
}

func TestSimulatorPinsSubjectControllerAndKind(t *testing.T) {
	grant, grantBytes, _, _ := capabilityChain(t, "10", "20", "1")
	capability := grant.Envelope.SubjectRef

	hijackerKey := testKey(9)
	hijackEnvelope := testEnvelope(KindAgentToolCapability, ActionConsume, capability, hijackerKey, "2", &grant.Commitment)
	hijackEnvelope.Issuer.ControllerRef = testRef("hijacker-controller")
	hijackPayload := CapabilityConsumePayload{
		CapabilityRef: capability, GrantCommitment: grant.Commitment, AssetRef: testDigest("sim-asset"),
		AmountMinor: "1", SourceEventDigest: testDigest("hijack-source"),
	}
	hijackPayload.Nullifier, _ = CapabilityNullifier(hijackEnvelope, hijackPayload)
	_, hijackBytes := seal(t, hijackEnvelope, hijackPayload, hijackerKey)
	if _, err := Simulate(simulationDocument(t, grantBytes, hijackBytes)); err == nil || !strings.Contains(err.Error(), "pinned") {
		t.Fatalf("expected controller takeover rejection, got %v", err)
	}

	key := testKey(1)
	lineagePayload := ArtifactLineagePayload{
		UpstreamRef: testRef("cross-kind-upstream"), DownstreamRef: capability,
		Relation: "DERIVES_FROM", EvidenceDigest: testDigest("cross-kind-evidence"),
	}
	lineageEnvelope := testEnvelope(KindArtifactLineage, ActionCheckpoint, capability, key, "2", &grant.Commitment)
	_, lineageBytes := seal(t, lineageEnvelope, lineagePayload, key)
	if _, err := Simulate(simulationDocument(t, grantBytes, lineageBytes)); err == nil || !strings.Contains(err.Error(), "kind is pinned") {
		t.Fatalf("expected cross-kind head rejection, got %v", err)
	}
}

func TestSimulatorRequiresExactCanonicalWireBytes(t *testing.T) {
	_, grantBytes, _, consumeBytes := capabilityChain(t, "10", "20", "1")
	document := simulationDocument(t, grantBytes, consumeBytes)
	if _, err := Simulate(append(document, '\n')); err == nil || !strings.Contains(err.Error(), "wire bytes") {
		t.Fatalf("expected exact-wire rejection, got %v", err)
	}
}

func ptr(value string) *string { return &value }
