package protocol

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"
)

type payloadCase struct {
	name    string
	kind    Kind
	action  Action
	subject string
	payload any
}

func testKey(seedByte byte) ed25519.PrivateKey {
	seed := make([]byte, ed25519.SeedSize)
	for i := range seed {
		seed[i] = seedByte
	}
	return ed25519.NewKeyFromSeed(seed)
}

func testRef(label string) string {
	sum := sha256.Sum256([]byte("ref/" + label))
	return hex.EncodeToString(sum[:])
}

func testDigest(label string) string {
	sum := sha256.Sum256([]byte("digest/" + label))
	return "sha256:" + hex.EncodeToString(sum[:])
}

func testEnvelope(kind Kind, action Action, subject string, key ed25519.PrivateKey, sequence string, parent *string) Envelope {
	publicKey := key.Public().(ed25519.PublicKey)
	fingerprint, _ := KeyFingerprint(publicKey)
	return Envelope{
		Kind: kind, Action: action, Audience: "kingdom:offline-shadow", SubjectRef: subject,
		Sequence: sequence, Parent: parent,
		Issuer:       Issuer{Namespace: "witness-fixture", ControllerRef: testRef("controller"), KeyFingerprint: fingerprint},
		PolicyDigest: testDigest("policy-v0"), ExpiryHeight: nil,
		Effects:   Effects{Scope: "RECORD_CONSTRUCTION_AND_OFFLINE_VALIDATION_ONLY", Authority: "NONE", Economic: "NONE", Reputation: "NONE"},
		Nonclaims: RequiredNonclaims(),
	}
}

func marshalPayload(t *testing.T, payload any) []byte {
	t.Helper()
	encoded, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	return encoded
}

func seal(t *testing.T, envelope Envelope, payload any, key ed25519.PrivateKey) (Record, []byte) {
	t.Helper()
	record, encoded, err := SignRecord(envelope, marshalPayload(t, payload), key)
	if err != nil {
		t.Fatal(err)
	}
	return record, encoded
}

func envelopeForCase(t *testing.T, tc payloadCase, key ed25519.PrivateKey) Envelope {
	t.Helper()
	sequence := "1"
	var parent *string
	switch p := tc.payload.(type) {
	case CapabilityConsumePayload:
		sequence, parent = "2", &p.GrantCommitment
	case CapabilityRevokePayload:
		sequence, parent = "2", &p.GrantCommitment
	case RecognitionWithdrawPayload:
		sequence, parent = "2", &p.AdoptionCommitment
	case OfferSupersedePayload:
		sequence, parent = "2", &p.Supersedes
	case OfferRevokePayload:
		sequence, parent = "2", &p.OfferCommitment
	case WakeSupersedePayload:
		sequence, parent = "2", &p.Supersedes
	case WakeWithdrawPayload:
		sequence, parent = "2", &p.CheckpointCommitment
	}
	return testEnvelope(tc.kind, tc.action, tc.subject, key, sequence, parent)
}

func allPayloadCases(t *testing.T) []payloadCase {
	t.Helper()
	key1 := testKey(1)
	key2 := testKey(2)
	nextFingerprint, _ := KeyFingerprint(key2.Public().(ed25519.PublicKey))
	controller := testRef("controller")
	capability := testRef("capability")
	consumeEnvelope := testEnvelope(KindAgentToolCapability, ActionConsume, capability, key1, "1", nil)
	consume := CapabilityConsumePayload{
		CapabilityRef: capability, GrantCommitment: testDigest("grant-commitment"), AssetRef: testDigest("asset"),
		AmountMinor: "7", SourceEventDigest: testDigest("source-event"),
	}
	consume.Nullifier, _ = CapabilityNullifier(consumeEnvelope, consume)

	offer := testRef("offer")
	recognition := testRef("recognition")
	workspace := testRef("workspace")
	downstream := testRef("downstream")
	release := testRef("release-stream")

	return []payloadCase{
		{"kingdom-release-checkpoint", KindKingdomReleaseRoot, ActionCheckpoint, release, KingdomReleasePayload{
			ReleaseRef: release, LedgerProtocol: "kingdom.immutable-publications/1", LedgerDocumentDigest: testDigest("ledger-document"),
			EntryMerkleRoot: testDigest("entry-root"), EntryCount: "83", GitCommit: "sha1:" + strings.Repeat("1", 40),
			GitTree: "sha1:" + strings.Repeat("2", 40), BuildManifestDigest: testDigest("build-manifest"),
			DeploymentManifestDigest: testDigest("deployment-manifest"), VerifierProtocol: "kingdom.ledger-verifier/1",
			VerifierDigest: testDigest("verifier"), PreviousRelease: nil,
		}},
		{"settlement-checkpoint", KindAgentToolSettlementRoot, ActionCheckpoint, testRef("settlement-feed"), SettlementRootPayload{
			ReceiptProtocol: "agenttool.settlement-receipt/1", ReceiptSchemaDigest: testDigest("receipt-schema"),
			SourceSequenceBinding: "PROJECTION_ONLY", ReceiptUniquenessScope: "BATCH_ONLY",
			FirstSequence: "1", LastSequence: "3", ReceiptCount: "2", DeclaredGaps: []Gap{{First: "2", Last: "2"}},
			MerkleRoot: testDigest("settlement-merkle"), PreviousBatch: nil,
		}},
		{"capability-grant", KindAgentToolCapability, ActionGrant, capability, CapabilityGrantPayload{
			CapabilityRef: capability, GrantDigest: testDigest("grant-document"), AssetRef: testDigest("asset"),
			MaxPerConsumeMinor: "10", MaxTotalMinor: "25",
		}},
		{"capability-consume", KindAgentToolCapability, ActionConsume, capability, consume},
		{"capability-revoke", KindAgentToolCapability, ActionRevoke, capability, CapabilityRevokePayload{
			CapabilityRef: capability, GrantCommitment: testDigest("grant-commitment"), ReasonDigest: testDigest("capability-reason"),
		}},
		{"recognition-adopt", KindAgentToolPublicRecognition, ActionAdopt, recognition, RecognitionAdoptPayload{
			RecognitionRef: recognition, SurfaceDigest: testDigest("surface"), RegistryDigest: testDigest("registry"),
			AdoptionDocumentDigest: testDigest("adoption-document"), AuthoritySequence: "1", Visibility: "PUBLIC",
		}},
		{"recognition-withdraw", KindAgentToolPublicRecognition, ActionWithdraw, recognition, RecognitionWithdrawPayload{
			RecognitionRef: recognition, AdoptionCommitment: testDigest("adoption"), SurfaceDigest: testDigest("surface"),
			RegistryDigest: testDigest("registry"), WithdrawalDocumentDigest: testDigest("recognition-withdrawal-document"),
			AuthoritySequence: "2", ReasonDigest: testDigest("recognition-reason"), Visibility: "PUBLIC",
		}},
		{"offer-publish", KindAgentToolOffer, ActionPublish, offer, OfferPublishPayload{
			OfferRef: offer, OfferDocumentDigest: testDigest("offer-publish-document"), CapabilityRoot: testDigest("offer-capabilities"),
			PricingRoot: testDigest("offer-pricing"), SLARoot: testDigest("offer-sla"), TermsDigest: testDigest("offer-terms"),
			Revision: "1", AuthoritySequence: "1", Visibility: "PUBLIC",
		}},
		{"offer-supersede", KindAgentToolOffer, ActionSupersede, offer, OfferSupersedePayload{
			OfferRef: offer, OfferDocumentDigest: testDigest("offer-supersede-document"), CapabilityRoot: testDigest("offer-capabilities-2"),
			PricingRoot: testDigest("offer-pricing-2"), SLARoot: testDigest("offer-sla-2"), TermsDigest: testDigest("offer-terms-2"),
			Revision: "2", AuthoritySequence: "2", Visibility: "PUBLIC", Supersedes: testDigest("offer-prior"),
		}},
		{"offer-revoke", KindAgentToolOffer, ActionRevoke, offer, OfferRevokePayload{
			OfferRef: offer, OfferCommitment: testDigest("offer-prior"), OfferDocumentDigest: testDigest("offer-revoke-document"),
			AuthoritySequence: "3", ReasonDigest: testDigest("offer-reason"), Visibility: "PUBLIC",
		}},
		{"wake-checkpoint", KindWakePublicCheckpoint, ActionCheckpoint, testRef("wake-contract"), WakeCheckpointPayload{
			PublicContractProtocol: "agenttool.public-wake-contract/0.1", PublicContractSchemaDigest: testDigest("wake-schema"),
			ContractRoot: testDigest("wake-contract-root"), CapabilityRoot: testDigest("wake-capabilities"), PricingRoot: testDigest("wake-pricing"),
			ProtocolsRoot: testDigest("wake-protocols"), BoundariesRoot: testDigest("wake-boundaries"), AuthoritySequence: "1",
		}},
		{"wake-supersede", KindWakePublicCheckpoint, ActionSupersede, testRef("wake-contract"), WakeSupersedePayload{
			PublicContractProtocol: "agenttool.public-wake-contract/0.1", PublicContractSchemaDigest: testDigest("wake-schema"),
			ContractRoot: testDigest("wake-contract-root-2"), CapabilityRoot: testDigest("wake-capabilities-2"), PricingRoot: testDigest("wake-pricing-2"),
			ProtocolsRoot: testDigest("wake-protocols-2"), BoundariesRoot: testDigest("wake-boundaries-2"), AuthoritySequence: "2",
			Supersedes: testDigest("wake-prior"),
		}},
		{"wake-withdraw", KindWakePublicCheckpoint, ActionWithdraw, testRef("wake-contract"), WakeWithdrawPayload{
			CheckpointCommitment: testDigest("wake-prior"), WithdrawalDocumentDigest: testDigest("wake-withdrawal-document"),
			AuthoritySequence: "3", ReasonDigest: testDigest("wake-reason"), Visibility: "PUBLIC",
		}},
		{"key-rotate", KindIssuerKeyContinuity, ActionRotate, controller, KeyRotatePayload{
			PreviousKeyFingerprint: testEnvelope(KindIssuerKeyContinuity, ActionRotate, controller, key1, "1", nil).Issuer.KeyFingerprint,
			NextKeyFingerprint:     nextFingerprint, RotationDigest: testDigest("rotation-document"),
		}},
		{"key-revoke", KindIssuerKeyContinuity, ActionRevoke, controller, KeyRevokePayload{
			RevokedKeyFingerprint: testEnvelope(KindIssuerKeyContinuity, ActionRevoke, controller, key1, "1", nil).Issuer.KeyFingerprint,
			ReasonDigest:          testDigest("key-reason"),
		}},
		{"artifact-lineage", KindArtifactLineage, ActionCheckpoint, downstream, ArtifactLineagePayload{
			UpstreamRef: testRef("upstream"), DownstreamRef: downstream, Relation: "DERIVES_FROM", EvidenceDigest: testDigest("lineage-evidence"),
		}},
		{"collaboration-checkpoint", KindCollaborationCheckpoint, ActionCheckpoint, workspace, CollaborationCheckpointPayload{
			WorkspaceRef: workspace, EpochRef: testDigest("collab-epoch"), EventHeadSequence: "10", EventHeadHash: testDigest("event-head"),
			EventCount: "10", ParticipantSetRoot: testDigest("participant-set"),
		}},
		{"dispute-terminal", KindDisputeTerminal, ActionSettle, testRef("dispute"), DisputeTerminalPayload{
			SettlementCommitment: testDigest("settlement"), Outcome: "SPLIT", DecisionDigest: testDigest("decision"), DistributionRoot: testDigest("distribution"),
		}},
	}
}
