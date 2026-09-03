package protocol

import (
	"crypto/ed25519"
	"encoding/json"
	"reflect"
	"sort"
	"strings"
	"testing"
)

func TestAllClosedKindActionPairsVerify(t *testing.T) {
	key := testKey(1)
	seen := make(map[Kind]map[Action]bool)
	for _, tc := range allPayloadCases(t) {
		t.Run(tc.name, func(t *testing.T) {
			envelope := envelopeForCase(t, tc, key)
			_, encoded := seal(t, envelope, tc.payload, key)
			if _, err := Verify(encoded); err != nil {
				t.Fatal(err)
			}
		})
		if seen[tc.kind] == nil {
			seen[tc.kind] = make(map[Action]bool)
		}
		seen[tc.kind][tc.action] = true
	}
	if !reflect.DeepEqual(seen, allowedActions) {
		t.Fatalf("payload cases do not cover closed action matrix\nseen=%v\nwant=%v", seen, allowedActions)
	}
}

func TestRecordRequiresEveryClosedObjectField(t *testing.T) {
	tc := allPayloadCases(t)[0]
	_, encoded := seal(t, envelopeForCase(t, tc, testKey(1)), tc.payload, testKey(1))

	paths := [][]string{
		{"envelope"}, {"payload"}, {"commitment"}, {"signature"},
		{"envelope", "protocol"}, {"envelope", "kind"}, {"envelope", "action"}, {"envelope", "audience"},
		{"envelope", "subject_ref"}, {"envelope", "sequence"}, {"envelope", "parent"}, {"envelope", "issuer"},
		{"envelope", "schema_hash"}, {"envelope", "payload_root"}, {"envelope", "policy_digest"},
		{"envelope", "expiry_height"}, {"envelope", "effects"}, {"envelope", "nonclaims"},
		{"envelope", "issuer", "namespace"}, {"envelope", "issuer", "controller_ref"}, {"envelope", "issuer", "key_fingerprint"},
		{"envelope", "effects", "authority"}, {"envelope", "effects", "economic"}, {"envelope", "effects", "reputation"},
		{"envelope", "effects", "scope"},
		{"envelope", "effects", "network_requests"}, {"envelope", "effects", "storage_writes"},
		{"envelope", "effects", "zerone_transaction"}, {"envelope", "effects", "external_receipt"},
		{"envelope", "effects", "nen_invocation"}, {"envelope", "effects", "score"},
		{"signature", "algorithm"}, {"signature", "public_key"}, {"signature", "value"},
	}
	for _, path := range paths {
		name := strings.Join(path, ".")
		t.Run("delete-"+name, func(t *testing.T) {
			mutated := deleteJSONPath(t, encoded, path)
			if _, err := Verify(mutated); err == nil || !strings.Contains(err.Error(), "missing required") {
				t.Fatalf("expected missing-field rejection, got %v", err)
			}
		})
	}

	for _, path := range [][]string{{}, {"envelope"}, {"envelope", "issuer"}, {"envelope", "effects"}, {"signature"}} {
		name := "record"
		if len(path) > 0 {
			name = strings.Join(path, ".")
		}
		t.Run("extra-"+name, func(t *testing.T) {
			mutated := addJSONPath(t, encoded, path, "unexpected", "value")
			if _, err := Verify(mutated); err == nil || !strings.Contains(err.Error(), "unknown field") {
				t.Fatalf("expected unknown-field rejection, got %v", err)
			}
		})
	}
}

func TestEveryPayloadRequiresExactKeys(t *testing.T) {
	key := testKey(1)
	for _, tc := range allPayloadCases(t) {
		t.Run(tc.name, func(t *testing.T) {
			_, encoded := seal(t, envelopeForCase(t, tc, key), tc.payload, key)
			var top map[string]any
			if err := json.Unmarshal(encoded, &top); err != nil {
				t.Fatal(err)
			}
			payload := top["payload"].(map[string]any)
			keys := make([]string, 0, len(payload))
			for field := range payload {
				keys = append(keys, field)
			}
			sort.Strings(keys)
			for _, field := range keys {
				mutated := cloneJSONMap(t, top)
				delete(mutated["payload"].(map[string]any), field)
				bytes, _ := json.Marshal(mutated)
				if _, err := Verify(bytes); err == nil || !strings.Contains(err.Error(), "missing required") {
					t.Fatalf("deleting %s was not rejected as missing: %v", field, err)
				}
			}
			mutated := cloneJSONMap(t, top)
			mutated["payload"].(map[string]any)["unexpected"] = "value"
			bytes, _ := json.Marshal(mutated)
			if _, err := Verify(bytes); err == nil || !strings.Contains(err.Error(), "unknown field") {
				t.Fatalf("extra payload key was not rejected: %v", err)
			}
		})
	}
}

func TestStableSubjectReferencesCannotFork(t *testing.T) {
	key := testKey(1)
	for _, tc := range allPayloadCases(t) {
		switch tc.kind {
		case KindKingdomReleaseRoot, KindAgentToolCapability, KindAgentToolPublicRecognition,
			KindAgentToolOffer, KindIssuerKeyContinuity, KindArtifactLineage, KindCollaborationCheckpoint:
		default:
			continue
		}
		t.Run(tc.name, func(t *testing.T) {
			envelope := envelopeForCase(t, tc, key)
			envelope.SubjectRef = testRef("fork")
			if _, _, err := SignRecord(envelope, marshalPayload(t, tc.payload), key); err == nil || !strings.Contains(err.Error(), "subject_ref") {
				t.Fatalf("expected stable subject rejection, got %v", err)
			}
		})
	}
}

func TestSequenceParentRulesAndSignature(t *testing.T) {
	tc := allPayloadCases(t)[0]
	key := testKey(1)
	parent := testDigest("parent")
	for _, tt := range []struct {
		name   string
		seq    string
		parent *string
	}{
		{"one-with-parent", "1", &parent},
		{"two-without-parent", "2", nil},
		{"zero", "0", nil},
		{"overflow", "18446744073709551616", nil},
	} {
		t.Run(tt.name, func(t *testing.T) {
			envelope := testEnvelope(tc.kind, tc.action, tc.subject, key, tt.seq, tt.parent)
			if _, _, err := SignRecord(envelope, marshalPayload(t, tc.payload), key); err == nil {
				t.Fatal("expected rejection")
			}
		})
	}

	_, encoded := seal(t, envelopeForCase(t, tc, key), tc.payload, key)
	mutated := addJSONPath(t, encoded, []string{"signature"}, "value", strings.Repeat("0", 128))
	if _, err := Verify(mutated); err == nil || !strings.Contains(err.Error(), "signature") {
		t.Fatalf("expected signature rejection: %v", err)
	}
}

func TestZeroEffectsAndNonclaimsAreExact(t *testing.T) {
	tc := allPayloadCases(t)[0]
	key := testKey(1)
	envelope := testEnvelope(tc.kind, tc.action, tc.subject, key, "1", nil)
	envelope.Effects.NetworkRequests = 1
	if _, _, err := SignRecord(envelope, marshalPayload(t, tc.payload), key); err == nil {
		t.Fatal("nonzero effect accepted")
	}
	envelope = testEnvelope(tc.kind, tc.action, tc.subject, key, "1", nil)
	envelope.Nonclaims = envelope.Nonclaims[:len(envelope.Nonclaims)-1]
	if _, _, err := SignRecord(envelope, marshalPayload(t, tc.payload), key); err == nil {
		t.Fatal("missing nonclaim accepted")
	}
}

func TestVerifyRequiresExactCanonicalWireBytes(t *testing.T) {
	tc := allPayloadCases(t)[0]
	key := testKey(1)
	_, encoded := seal(t, envelopeForCase(t, tc, key), tc.payload, key)
	for name, mutated := range map[string][]byte{
		"trailing-newline":      append(append([]byte(nil), encoded...), '\n'),
		"leading-space":         append([]byte{' '}, encoded...),
		"nonminimal-key-escape": []byte(strings.Replace(string(encoded), `"protocol"`, `"\u0070rotocol"`, 1)),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := Verify(mutated); err == nil || !strings.Contains(err.Error(), "wire bytes") {
				t.Fatalf("expected exact-wire rejection, got %v", err)
			}
		})
	}
}

func TestSettlementPayloadDeclaredGapsCannotBeNull(t *testing.T) {
	tc := allPayloadCases(t)[1]
	key := testKey(1)
	_, encoded := seal(t, envelopeForCase(t, tc, key), tc.payload, key)
	var root map[string]any
	if err := json.Unmarshal(encoded, &root); err != nil {
		t.Fatal(err)
	}
	root["payload"].(map[string]any)["declared_gaps"] = nil
	mutated, _ := json.Marshal(root)
	if _, err := Verify(mutated); err == nil || !strings.Contains(err.Error(), "non-null JSON array") {
		t.Fatalf("expected null-array rejection, got %v", err)
	}
}

func TestCollaborationCheckpointRequiresCompleteJournalPrefix(t *testing.T) {
	key := testKey(1)
	var tc payloadCase
	for _, candidate := range allPayloadCases(t) {
		if candidate.kind == KindCollaborationCheckpoint {
			tc = candidate
			break
		}
	}
	payload := tc.payload.(CollaborationCheckpointPayload)
	payload.EventCount = "9"
	if _, _, err := SignRecord(envelopeForCase(t, tc, key), marshalPayload(t, payload), key); err == nil || !strings.Contains(err.Error(), "must equal event_count") {
		t.Fatalf("expected full-prefix equality rejection, got %v", err)
	}
}

var hostileMarshalCalled bool

type hostilePayload []byte

func (hostilePayload) MarshalJSON() ([]byte, error) {
	hostileMarshalCalled = true
	panic("must never be called")
}

func TestSignRecordConsumesBytesWithoutInvokingMarshalers(t *testing.T) {
	tc := allPayloadCases(t)[0]
	key := testKey(1)
	hostileMarshalCalled = false
	payload := hostilePayload(marshalPayload(t, tc.payload))
	if _, _, err := SignRecord(envelopeForCase(t, tc, key), []byte(payload), key); err != nil {
		t.Fatal(err)
	}
	if hostileMarshalCalled {
		t.Fatal("caller-defined MarshalJSON was invoked")
	}
}

func TestUint64PayloadBoundary(t *testing.T) {
	key := testKey(1)
	capability := testRef("max-capability")
	envelope := testEnvelope(KindAgentToolCapability, ActionGrant, capability, key, "1", nil)
	payload := CapabilityGrantPayload{CapabilityRef: capability, GrantDigest: testDigest("grant"), AssetRef: testDigest("asset"), MaxPerConsumeMinor: "18446744073709551615", MaxTotalMinor: "18446744073709551615"}
	if _, _, err := SignRecord(envelope, marshalPayload(t, payload), key); err != nil {
		t.Fatalf("max uint64 rejected: %v", err)
	}
	payload.MaxTotalMinor = "18446744073709551616"
	if _, _, err := SignRecord(envelope, marshalPayload(t, payload), key); err == nil {
		t.Fatal("uint64+1 accepted")
	}
}

func TestLifecyclePointersEqualEnvelopeParent(t *testing.T) {
	key := testKey(1)
	for _, tc := range allPayloadCases(t) {
		applies := false
		switch tc.kind {
		case KindKingdomReleaseRoot, KindAgentToolSettlementRoot:
			applies = true
		case KindAgentToolPublicRecognition:
			applies = tc.action == ActionWithdraw
		case KindAgentToolOffer:
			applies = tc.action == ActionSupersede || tc.action == ActionRevoke
		case KindWakePublicCheckpoint:
			applies = tc.action == ActionSupersede || tc.action == ActionWithdraw
		}
		if !applies {
			continue
		}
		t.Run(tc.name, func(t *testing.T) {
			envelope := envelopeForCase(t, tc, key)
			envelope.Sequence = "2"
			wrong := testDigest("wrong-parent")
			envelope.Parent = &wrong
			if _, _, err := SignRecord(envelope, marshalPayload(t, tc.payload), key); err == nil || !strings.Contains(err.Error(), "envelope.parent") {
				t.Fatalf("expected pointer coherence rejection, got %v", err)
			}
		})
	}
}

func TestInitialLifecycleActionsRequireGenesisHead(t *testing.T) {
	key := testKey(1)
	for _, tc := range allPayloadCases(t) {
		if !((tc.kind == KindAgentToolCapability && tc.action == ActionGrant) ||
			(tc.kind == KindAgentToolPublicRecognition && tc.action == ActionAdopt) ||
			(tc.kind == KindAgentToolOffer && tc.action == ActionPublish) ||
			(tc.kind == KindWakePublicCheckpoint && tc.action == ActionCheckpoint)) {
			continue
		}
		t.Run(tc.name, func(t *testing.T) {
			parent := testDigest("parent")
			envelope := testEnvelope(tc.kind, tc.action, tc.subject, key, "2", &parent)
			if _, _, err := SignRecord(envelope, marshalPayload(t, tc.payload), key); err == nil || !strings.Contains(err.Error(), "initial lifecycle") {
				t.Fatalf("expected initial-action rejection, got %v", err)
			}
		})
	}
}

func TestCapabilityUseAndRevokeRequireExistingHeadShape(t *testing.T) {
	key := testKey(1)
	for _, tc := range allPayloadCases(t) {
		if tc.kind != KindAgentToolCapability || (tc.action != ActionConsume && tc.action != ActionRevoke) {
			continue
		}
		t.Run(tc.name, func(t *testing.T) {
			envelope := testEnvelope(tc.kind, tc.action, tc.subject, key, "1", nil)
			if _, _, err := SignRecord(envelope, marshalPayload(t, tc.payload), key); err == nil || !strings.Contains(err.Error(), "non-initial lifecycle") {
				t.Fatalf("expected non-initial rejection, got %v", err)
			}
		})
	}
}

func TestGenesisSettlementBeginsAtSourceSequenceOne(t *testing.T) {
	key := testKey(1)
	tc := allPayloadCases(t)[1]
	payload := tc.payload.(SettlementRootPayload)
	payload.FirstSequence = "42"
	payload.LastSequence = "42"
	payload.ReceiptCount = "1"
	payload.DeclaredGaps = []Gap{}
	if _, _, err := SignRecord(envelopeForCase(t, tc, key), marshalPayload(t, payload), key); err == nil || !strings.Contains(err.Error(), "first_sequence 1") {
		t.Fatalf("expected genesis-prefix rejection, got %v", err)
	}
}

func deleteJSONPath(t *testing.T, input []byte, path []string) []byte {
	t.Helper()
	var root map[string]any
	if err := json.Unmarshal(input, &root); err != nil {
		t.Fatal(err)
	}
	object := root
	for _, part := range path[:len(path)-1] {
		object = object[part].(map[string]any)
	}
	delete(object, path[len(path)-1])
	result, _ := json.Marshal(root)
	return result
}

func addJSONPath(t *testing.T, input []byte, path []string, key string, value any) []byte {
	t.Helper()
	var root map[string]any
	if err := json.Unmarshal(input, &root); err != nil {
		t.Fatal(err)
	}
	object := root
	for _, part := range path {
		object = object[part].(map[string]any)
	}
	object[key] = value
	result, _ := json.Marshal(root)
	return result
}

func cloneJSONMap(t *testing.T, source map[string]any) map[string]any {
	t.Helper()
	encoded, err := json.Marshal(source)
	if err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(encoded, &result); err != nil {
		t.Fatal(err)
	}
	return result
}

var _ func(Envelope, []byte, ed25519.PrivateKey) (Record, []byte, error) = SignRecord
