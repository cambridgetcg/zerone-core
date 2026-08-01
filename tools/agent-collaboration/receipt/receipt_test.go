package receipt

import (
	"bytes"
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/oasisprotocol/curve25519-voi/curve"
)

type signedTestContext struct {
	manifest    Manifest
	alpha       PrivateKeyFile
	beta        PrivateKeyFile
	alphaPublic PublicKeyFile
	betaPublic  PublicKeyFile
	terms       ConsentTerms
	termsDigest string
}

func TestSignedHistoryAlphaBetaEndToEnd(t *testing.T) {
	context := newSignedTestContext(t)
	receipts := signedHappyHistory(t, context)
	report, err := VerifyHistory(context.manifest, receipts)
	if err != nil {
		t.Fatalf("VerifyHistory: %v", err)
	}
	if !report.Valid || report.EventCount != "5" || len(report.Tasks) != 1 {
		t.Fatalf("verification report: %#v", report)
	}
	if report.Tasks[0].Status != StatusAccepted {
		t.Fatalf("task status = %q, want %q", report.Tasks[0].Status, StatusAccepted)
	}
	if !reflect.DeepEqual(report.Effects, ZeroEffects()) {
		t.Fatalf("report effects drifted: %#v", report.Effects)
	}
	if report.Assurance != AssuranceEventKeyPossession {
		t.Fatalf("assurance = %q", report.Assurance)
	}
	wantLimitations := []string{
		"NO_CHAIN_NETWORK_ECONOMIC_REWARD_KARMA_OR_GOVERNANCE_EFFECT",
		"SIGNATURES_PROVE_EXACT_LOCAL_KEY_POSSESSION_ONLY",
		"TASK_STATUS_IS_A_SIGNED_PROTOCOL_DECLARATION_NOT_TRUTH_QUALITY_OR_LEGAL_AUTHORITY",
		"FREE_TEXT_AND_CONTENT_DIGESTS_ARE_UNINTERPRETED_DECLARATIONS",
		"VERIFICATION_REPORT_IS_UNSIGNED_REVERIFY_PINNED_JOURNAL_BYTES",
		"UNSIGNED_MANIFEST_DOES_NOT_PROVE_POSSESSION_OF_NON_SIGNING_ROSTER_KEYS",
	}
	if !reflect.DeepEqual(report.Limitations, wantLimitations) {
		t.Fatalf("report limitations = %#v", report.Limitations)
	}

	for index, original := range receipts {
		encoded, err := MarshalDocument(original)
		if err != nil {
			t.Fatal(err)
		}
		parsed, err := ParseSignedReceipt(encoded)
		if err != nil {
			t.Fatalf("parse receipt %d: %v", index+1, err)
		}
		if !reflect.DeepEqual(parsed, original) {
			t.Fatalf("receipt %d parse drift", index+1)
		}
		if strings.Contains(string(encoded), "ed25519-seed:") {
			t.Fatalf("receipt %d leaked private key material", index+1)
		}
	}

	// Fixed keys, times, payloads, and nonce readers make this a cross-change
	// transcript vector. Update only with a deliberate protocol-version change.
	if got, want := receipts[0].EventID, "sha256:623923f6a3991405febab2f6607b288c26416bb0dba72273c5a7a193178ece84"; got != want {
		t.Fatalf("proposal event vector = %s, want %s", got, want)
	}
	if got, want := receipts[0].ReceiptSHA256, "sha256:781d17ef5ba41e086ef5f161c3557ec34034fca0e4cd648479f4ffa3a15348cd"; got != want {
		t.Fatalf("proposal receipt vector = %s, want %s", got, want)
	}
}

func TestVerifyRejectsTamperWrongKeyReplayAndReorder(t *testing.T) {
	context := newSignedTestContext(t)
	receipts := signedHappyHistory(t, context)

	tests := []struct {
		name   string
		mutate func([]SignedReceipt)
		want   string
	}{
		{"event field", func(changed []SignedReceipt) { changed[1].Event.OccurredAt = "2026-08-01T12:00:09Z" }, "event_id"},
		{"event id", func(changed []SignedReceipt) { changed[1].EventID = signedTestDigest("forged-event") }, "event_id"},
		{"signature", func(changed []SignedReceipt) { changed[1].Signature.Value = "ed25519:" + strings.Repeat("00", 64) }, "signature"},
		{"receipt hash", func(changed []SignedReceipt) { changed[1].ReceiptSHA256 = signedTestDigest("forged-receipt") }, "receipt_sha256"},
		{"wrong actor key", func(changed []SignedReceipt) { changed[1].Event.ActorKeyID = context.alpha.KeyID }, "actor_key_id"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			changed := cloneReceipts(t, receipts)
			test.mutate(changed)
			if _, err := VerifyHistory(context.manifest, changed); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("VerifyHistory error = %v, want substring %q", err, test.want)
			}
		})
	}

	reordered := []SignedReceipt{receipts[1], receipts[0]}
	if _, err := VerifyHistory(context.manifest, reordered); err == nil || !strings.Contains(err.Error(), "sequence") {
		t.Fatalf("reordered history error = %v", err)
	}
	replayed := append(cloneReceipts(t, receipts[:1]), receipts[0])
	if _, err := VerifyHistory(context.manifest, replayed); err == nil {
		t.Fatal("exact replay unexpectedly verified")
	}

	secondProposal := signedProposal(t, context, "parallel", "2026-08-01T12:00:01Z")
	secondRequest := signedRequest(t, EventTaskProposed, context.alpha, "2026-08-01T12:00:01Z", secondProposal)
	sameNonce, err := buildReceiptWithRandom(context.manifest, 2, receipts[0].ReceiptSHA256, secondRequest, context.alpha, bytes.NewReader(bytes.Repeat([]byte{0x21}, 32)))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := VerifyHistory(context.manifest, []SignedReceipt{receipts[0], sameNonce}); err == nil || !strings.Contains(err.Error(), "reuses an actor nonce") {
		t.Fatalf("same actor nonce error = %v", err)
	}
}

func TestEveryEffectMutationFailsClosed(t *testing.T) {
	context := newSignedTestContext(t)
	root := signedHappyHistory(t, context)[0]
	mutations := map[string]func(*Effects){
		"network":       func(e *Effects) { e.Network = "WRITE" },
		"chain":         func(e *Effects) { e.Chain = "WRITE" },
		"economic":      func(e *Effects) { e.Economic = "VALUE" },
		"fiat":          func(e *Effects) { e.Fiat = "VALUE" },
		"zrn":           func(e *Effects) { e.ZRN = "VALUE" },
		"reward":        func(e *Effects) { e.Reward = "VALUE" },
		"karma":         func(e *Effects) { e.Karma = "VALUE" },
		"governance":    func(e *Effects) { e.Governance = "VALUE" },
		"ownership":     func(e *Effects) { e.Ownership = "VALUE" },
		"qualification": func(e *Effects) { e.Qualification = "VALUE" },
		"membership":    func(e *Effects) { e.Membership = "VALUE" },
		"endorsement":   func(e *Effects) { e.Endorsement = "VALUE" },
		"authority":     func(e *Effects) { e.Authority = "VALUE" },
		"attribution":   func(e *Effects) { e.Attribution = "VALUE" },
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			changed := cloneReceipts(t, []SignedReceipt{root})
			mutate(&changed[0].Event.Effects)
			if _, err := VerifyHistory(context.manifest, changed); err == nil || !strings.Contains(err.Error(), "zero-effects") {
				t.Fatalf("effect mutation error = %v", err)
			}
		})
	}
}

func TestStrictParsingRejectsAmbiguousJSON(t *testing.T) {
	context := newSignedTestContext(t)
	manifestJSON, err := MarshalDocument(context.manifest)
	if err != nil {
		t.Fatal(err)
	}
	request := signedRequest(t, EventTaskProposed, context.alpha, "2026-08-01T12:00:00Z", signedProposal(t, context, "root", "2026-08-01T12:00:00Z"))
	requestJSON, err := MarshalDocument(request)
	if err != nil {
		t.Fatal(err)
	}
	receiptJSON, err := MarshalDocument(signedHappyHistory(t, context)[0])
	if err != nil {
		t.Fatal(err)
	}

	manifestCases := []struct {
		name string
		raw  []byte
		want string
	}{
		{"duplicate", []byte(strings.Replace(string(manifestJSON), `"schema":`, `"schema":"zerone.agent-collaboration-manifest/v0","schema":`, 1)), "duplicate JSON object key"},
		{"case alias", []byte(strings.Replace(string(manifestJSON), `"schema":`, `"Schema":"zerone.agent-collaboration-manifest/v0","schema":`, 1)), "unexpected exact field"},
		{"nested effect alias", []byte(strings.Replace(string(manifestJSON), `"network":`, `"Network":"NONE","network":`, 1)), "unexpected exact field"},
		{"participant alias", []byte(strings.Replace(string(manifestJSON), `"actor_id":`, `"Actor_ID":"ignored","actor_id":`, 1)), "unexpected exact field"},
		{"unknown", []byte(strings.Replace(string(manifestJSON), `"mode":`, `"unknown":"x","mode":`, 1)), "unknown field"},
		{"null", []byte(strings.Replace(string(manifestJSON), `"mode":"INTERNAL_LOCAL_ONLY"`, `"mode":null`, 1)), "null is not allowed"},
		{"number", []byte(strings.Replace(string(manifestJSON), `"mode":"INTERNAL_LOCAL_ONLY"`, `"mode":1`, 1)), "numbers are not allowed"},
		{"escaped error path", []byte(strings.Replace(string(manifestJSON), `"mode":`, `"\u001b[31m\nINJECT":null,"mode":`, 1)), "null is not allowed"},
		{"trailing", append(append([]byte(nil), manifestJSON...), []byte(` {}`)...), "multiple JSON values"},
		{"oversize", make([]byte, MaxManifestBytes+1), "exceeds"},
	}
	for _, test := range manifestCases {
		t.Run("manifest "+test.name, func(t *testing.T) {
			if _, err := ParseManifest(test.raw); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("ParseManifest error = %v, want %q", err, test.want)
			}
		})
	}
	injected := manifestCases[7].raw
	if _, err := ParseManifest(injected); err == nil || strings.ContainsRune(err.Error(), '\x1b') || strings.Contains(err.Error(), "\nINJECT") {
		t.Fatalf("unsafe parser error rendering = %q", err)
	}

	requestCases := []struct {
		name string
		raw  []byte
		want string
	}{
		{"nested duplicate", []byte(strings.Replace(string(requestJSON), `"task_id":`, `"task_id":"root","task_id":`, 1)), "duplicate JSON object key"},
		{"nested case alias", []byte(strings.Replace(string(requestJSON), `"task_id":`, `"TASK_ID":"root","task_id":`, 1)), "unexpected exact field"},
		{"consent case alias", []byte(strings.Replace(string(requestJSON), `"role":`, `"Role":"ignored","role":`, 1)), "unexpected exact field"},
		{"nested unknown", []byte(strings.Replace(string(requestJSON), `"task_id":`, `"unexpected":"x","task_id":`, 1)), "unknown field"},
		{"omitted", []byte(strings.Replace(string(requestJSON), `"acceptance_required":true,`, "", 1)), "missing required field"},
		{"unpaired surrogate", []byte(strings.Replace(string(requestJSON), `"objective":"exercise one signed local collaboration loop"`, `"objective":"\ud800"`, 1)), "unpaired UTF-16"},
	}
	for _, test := range requestCases {
		t.Run("request "+test.name, func(t *testing.T) {
			if _, err := ParseEventRequest(test.raw); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("ParseEventRequest error = %v, want %q", err, test.want)
			}
		})
	}

	receiptCases := []struct {
		name string
		raw  []byte
	}{
		{"root case alias", []byte(strings.Replace(string(receiptJSON), `"schema":`, `"Schema":"zerone.agent-collaboration-receipt/v0","schema":`, 1))},
		{"event case alias", []byte(strings.Replace(string(receiptJSON), `"actor_id":`, `"Actor_ID":"ignored","actor_id":`, 1))},
		{"signature case alias", []byte(strings.Replace(string(receiptJSON), `"algorithm":`, `"Algorithm":"ED25519","algorithm":`, 1))},
		{"effects case alias", []byte(strings.Replace(string(receiptJSON), `"network":`, `"Network":"NONE","network":`, 1))},
	}
	for _, test := range receiptCases {
		t.Run("receipt "+test.name, func(t *testing.T) {
			if _, err := ParseSignedReceipt(test.raw); err == nil || !strings.Contains(err.Error(), "unexpected exact field") {
				t.Fatalf("ParseSignedReceipt error = %v", err)
			}
		})
	}
}

func TestRejectsDegenerateNoncanonicalAndMixedTorsionPublicKeys(t *testing.T) {
	identity := make([]byte, ed25519.PublicKeySize)
	identity[0] = 1
	identityParticipant := Participant{
		ActorID:   computeActorID(identity),
		Label:     "Identity",
		KeyID:     computeKeyID(identity),
		Algorithm: AlgorithmEd25519,
		PublicKey: "ed25519:" + hex.EncodeToString(identity),
	}
	if err := validateParticipant(identityParticipant); err == nil || !strings.Contains(err.Error(), "small-order") {
		t.Fatalf("identity public key error = %v", err)
	}

	// This degenerate signature is accepted by Go's compatibility verifier for
	// every message when A is the identity. Strict v0 verification must reject.
	forged := make([]byte, ed25519.SignatureSize)
	forged[0] = 0x58
	for index := 1; index < 32; index++ {
		forged[index] = 0x66
	}
	forged[32] = 1
	message := []byte("one message cannot prove identity-key possession")
	if !ed25519.Verify(ed25519.PublicKey(identity), message, forged) {
		t.Fatal("test vector no longer demonstrates standard-library identity-key acceptance")
	}
	if strictVerifyEd25519(ed25519.PublicKey(identity), message, forged) {
		t.Fatal("strict verifier accepted universal identity-key forgery")
	}

	offCurve := bytes.Repeat([]byte{0xff}, ed25519.PublicKeySize)
	if err := validateEd25519PublicKey(offCurve); err == nil {
		t.Fatal("off-curve/noncanonical public key unexpectedly accepted")
	}

	var mixed curve.EdwardsPoint
	mixed.Add(curve.ED25519_BASEPOINT_POINT, curve.EIGHT_TORSION[1])
	mixedBytes, err := mixed.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	if err := validateEd25519PublicKey(mixedBytes); err == nil || !strings.Contains(err.Error(), "prime-order") {
		t.Fatalf("mixed-torsion public key error = %v", err)
	}
}

func TestInspectableTextRejectsBidiFormattingButAllowsUnicode(t *testing.T) {
	if err := validateText("label", "研究🧬", 1, 64); err != nil {
		t.Fatalf("ordinary Unicode text was rejected: %v", err)
	}
	if err := validateText("label", "Alpha\u202eNOSREP", 1, 64); err == nil || !strings.Contains(err.Error(), "bidirectional") {
		t.Fatalf("bidi formatting error = %v", err)
	}
}

func TestPayloadKeyReorderingDoesNotChangeVerification(t *testing.T) {
	context := newSignedTestContext(t)
	receipt := signedHappyHistory(t, context)[0]
	var payload map[string]any
	if err := json.Unmarshal(receipt.Event.Payload, &payload); err != nil {
		t.Fatal(err)
	}
	reordered, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(reordered, receipt.Event.Payload) {
		t.Fatal("test payload map unexpectedly retained struct field order")
	}
	receipt.Event.Payload = reordered
	encoded, err := json.Marshal(receipt)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := ParseSignedReceipt(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := VerifyHistory(context.manifest, []SignedReceipt{parsed}); err != nil {
		t.Fatalf("reordered payload failed canonical verification: %v", err)
	}
}

func TestKeyAndManifestIdentityPins(t *testing.T) {
	context := newSignedTestContext(t)
	changedKey := context.alpha
	changedKey.PrivateKey = "ed25519-seed:" + strings.Repeat("ff", 32)
	if err := ValidatePrivateKeyFile(changedKey); err == nil || !strings.Contains(err.Error(), "does not match") {
		t.Fatalf("wrong private seed error = %v", err)
	}

	changedManifest := context.manifest
	changedManifest.Participants = append([]Participant(nil), context.manifest.Participants...)
	changedManifest.Participants[0].Label += " changed"
	if err := ValidateManifest(changedManifest); err == nil || !strings.Contains(err.Error(), "collaboration_id mismatch") {
		t.Fatalf("manifest roster tamper error = %v", err)
	}
}

func TestBuildNextReceiptRequiresCallerPinnedCollaborationAndHead(t *testing.T) {
	context := newSignedTestContext(t)
	request := signedRequest(t, EventTaskProposed, context.alpha, "2026-08-01T12:00:00Z", signedProposal(t, context, "pinned", "2026-08-01T12:00:00Z"))

	created, report, err := BuildNextReceipt(context.manifest, nil, signedTestDigest("wrong-collaboration"), None, request, context.alpha)
	if err == nil || !strings.Contains(err.Error(), "collaboration ID") {
		t.Fatalf("wrong collaboration pin error = %v", err)
	}
	if !reflect.DeepEqual(created, SignedReceipt{}) || !reflect.DeepEqual(report, VerificationReport{}) {
		t.Fatalf("wrong collaboration pin returned candidate material: %#v %#v", created, report)
	}

	created, report, err = BuildNextReceipt(context.manifest, nil, context.manifest.CollaborationID, signedTestDigest("wrong-head"), request, context.alpha)
	if err == nil || !strings.Contains(err.Error(), "history head") {
		t.Fatalf("wrong head pin error = %v", err)
	}
	if !reflect.DeepEqual(created, SignedReceipt{}) || !reflect.DeepEqual(report, VerificationReport{}) {
		t.Fatalf("wrong head pin returned candidate material: %#v %#v", created, report)
	}

	created, report, err = BuildNextReceipt(context.manifest, nil, context.manifest.CollaborationID, None, request, context.alpha)
	if err != nil {
		t.Fatalf("correct pins: %v", err)
	}
	if created.Event.CollaborationID != context.manifest.CollaborationID || report.HeadReceiptSHA256 != created.ReceiptSHA256 {
		t.Fatalf("correctly pinned candidate/report mismatch: %#v %#v", created, report)
	}

	if _, _, err := BuildNextReceipt(context.manifest, nil, "\x1b[31m", None, request, context.alpha); err == nil || strings.ContainsRune(err.Error(), '\x1b') {
		t.Fatalf("malformed pin error = %q", err)
	}
}

func TestVerifyHistoryRejectsReceiptCountBeyondBound(t *testing.T) {
	context := newSignedTestContext(t)
	if _, err := VerifyHistory(context.manifest, make([]SignedReceipt, MaxHistoryReceipts+1)); err == nil || !strings.Contains(err.Error(), "receipt limit") {
		t.Fatalf("history bound error = %v", err)
	}
	if _, err := DecodeDocuments(make([][]byte, MaxHistoryReceipts+1)); err == nil || !strings.Contains(err.Error(), "receipt limit") {
		t.Fatalf("document bound error = %v", err)
	}
}

func TestManifestConstructionPrevalidatesBoundedInputs(t *testing.T) {
	context := newSignedTestContext(t)
	participants := append([]Participant(nil), context.manifest.Participants...)
	participants[0].Label = strings.Repeat("x", 65)
	if _, err := newManifestWithNonce(participants, context.manifest.CreatedAt, context.manifest.Nonce); err == nil || !strings.Contains(err.Error(), "label") {
		t.Fatalf("oversized participant error = %v", err)
	}
	if _, err := NewManifest(make([]Participant, 17), context.manifest.CreatedAt); err == nil || !strings.Contains(err.Error(), "between 2 and 16") {
		t.Fatalf("oversized roster error = %v", err)
	}
}

func newSignedTestContext(t *testing.T) signedTestContext {
	t.Helper()
	alpha, alphaPublic, err := generateKeyWithRandom("Alpha", bytes.NewReader(bytes.Repeat([]byte{0x11}, 32)))
	if err != nil {
		t.Fatal(err)
	}
	beta, betaPublic, err := generateKeyWithRandom("Beta", bytes.NewReader(bytes.Repeat([]byte{0x12}, 32)))
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := newManifestWithNonce(
		[]Participant{alphaPublic.Participant, betaPublic.Participant},
		"2026-08-01T12:00:00Z",
		"hex:"+strings.Repeat("13", 32),
	)
	if err != nil {
		t.Fatal(err)
	}
	terms := reducerTestTerms()
	termsDigest, err := ConsentTermsDigest(terms)
	if err != nil {
		t.Fatal(err)
	}
	return signedTestContext{manifest: manifest, alpha: alpha, beta: beta, alphaPublic: alphaPublic, betaPublic: betaPublic, terms: terms, termsDigest: termsDigest}
}

func signedHappyHistory(t *testing.T, context signedTestContext) []SignedReceipt {
	t.Helper()
	receipts := make([]SignedReceipt, 0, 5)
	head := None
	appendReceipt := func(kind string, actor PrivateKeyFile, at string, payload any, marker byte) SignedReceipt {
		t.Helper()
		request := signedRequest(t, kind, actor, at, payload)
		created, err := buildReceiptWithRandom(context.manifest, uint64(len(receipts)+1), head, request, actor, bytes.NewReader(bytes.Repeat([]byte{marker}, 32)))
		if err != nil {
			t.Fatalf("buildReceiptWithRandom %s: %v", kind, err)
		}
		receipts = append(receipts, created)
		head = created.ReceiptSHA256
		return created
	}
	proposalPayload := signedProposal(t, context, "root", "2026-08-01T12:00:00Z")
	proposal := appendReceipt(EventTaskProposed, context.alpha, "2026-08-01T12:00:00Z", proposalPayload, 0x21)
	acceptance := appendReceipt(EventTaskDecision, context.beta, "2026-08-01T12:00:01Z", TaskDecision{
		TaskID:             "root",
		OfferEventID:       proposal.EventID,
		Decision:           DecisionAccept,
		Affirmative:        true,
		ConsentTermsSHA256: context.termsDigest,
		ReasonCodes:        []string{},
	}, 0x22)
	artifact := signedTestDigest("artifact")
	contribution := appendReceipt(EventContribution, context.beta, "2026-08-01T12:00:02Z", ContributionSubmitted{
		TaskID:            "root",
		AcceptanceEventID: acceptance.EventID,
		Summary:           "one bounded internal artifact",
		ArtifactSHA256:    []string{artifact},
		EvidenceSHA256:    []string{},
		LimitationCodes:   []string{},
	}, 0x23)
	completion := appendReceipt(EventCompletionClaimed, context.beta, "2026-08-01T12:00:03Z", CompletionClaimed{
		TaskID:               "root",
		AcceptanceEventID:    acceptance.EventID,
		ContributionEventIDs: []string{contribution.EventID},
		DeliverableSHA256:    []string{artifact},
		LimitationCodes:      []string{},
	}, 0x24)
	appendReceipt(EventCompletionReview, context.alpha, "2026-08-01T12:00:04Z", CompletionReviewed{
		TaskID:            "root",
		CompletionEventID: completion.EventID,
		Decision:          ReviewAccept,
		ReasonCodes:       []string{},
		EvidenceSHA256:    []string{artifact},
	}, 0x25)
	return receipts
}

func signedProposal(t *testing.T, context signedTestContext, taskID, _ string) TaskProposed {
	t.Helper()
	criteria := []string{"artifact-is-locally-inspectable", "effects-remain-none"}
	sort.Strings(criteria)
	return TaskProposed{
		TaskID:                 taskID,
		ParentTaskID:           None,
		Objective:              "exercise one signed local collaboration loop",
		OfferedToActorID:       context.beta.ActorID,
		OfferedToActorKeyID:    context.beta.KeyID,
		AcceptanceRequired:     true,
		ConsentTerms:           context.terms,
		ConsentTermsSHA256:     context.termsDigest,
		AcceptanceCriteria:     criteria,
		RequiredArtifactSHA256: []string{},
	}
}

func signedRequest(t *testing.T, kind string, actor PrivateKeyFile, at string, payload any) EventRequest {
	t.Helper()
	raw, err := MarshalDocument(payload)
	if err != nil {
		t.Fatal(err)
	}
	return EventRequest{Schema: EventRequestSchema, Kind: kind, ActorID: actor.ActorID, OccurredAt: at, Payload: raw}
}

func signedTestDigest(label string) string {
	return digestText(domainDigest("zerone.agent-collaboration.signed-test/v0", []byte(label)))
}

func cloneReceipts(t *testing.T, receipts []SignedReceipt) []SignedReceipt {
	t.Helper()
	encoded, err := json.Marshal(receipts)
	if err != nil {
		t.Fatal(err)
	}
	var clone []SignedReceipt
	if err := json.Unmarshal(encoded, &clone); err != nil {
		t.Fatal(err)
	}
	return clone
}
