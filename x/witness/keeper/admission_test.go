package keeper_test

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/zerone-chain/zerone/tools/witness-v0/protocol"
	"github.com/zerone-chain/zerone/x/witness/keeper"
	witnesstypes "github.com/zerone-chain/zerone/x/witness/types"
)

const testChainID = "witness-test-1"

type verifierSpy struct {
	record witnesstypes.VerifiedRecord
	err    error
	calls  int
	mutate bool
}

func (spy *verifierSpy) Verify(record []byte) (witnesstypes.VerifiedRecord, error) {
	spy.calls++
	if spy.mutate && len(record) != 0 {
		record[0] ^= 0xff
	}
	return spy.record, spy.err
}

type sharedProtocolVerifier struct{ calls int }

func (verifier *sharedProtocolVerifier) Verify(record []byte) (witnesstypes.VerifiedRecord, error) {
	verifier.calls++
	return keeper.VerifyFrozenCore(record)
}

type controllerPolicySpy struct{ calls int }

func (policy *controllerPolicySpy) Authorize(context.Context, witnesstypes.VerifiedRecord) error {
	policy.calls++
	return nil
}

type stateMutatorSpy struct {
	calls              int
	heads              map[string]string
	controllerPolicies map[string]string
	nullifiers         map[string]string
	score              int64
	reputation         int64
	economicMinor      uint64
}

func newStateMutatorSpy() *stateMutatorSpy {
	return &stateMutatorSpy{
		heads:              map[string]string{"sentinel": "unchanged"},
		controllerPolicies: map[string]string{"sentinel": "unchanged"},
		nullifiers:         map[string]string{"sentinel": "unchanged"},
		score:              7,
		reputation:         11,
		economicMinor:      13,
	}
}

func (state *stateMutatorSpy) Apply(_ context.Context, record witnesstypes.VerifiedRecord) error {
	state.calls++
	state.heads[record.SubjectRef] = record.Commitment
	state.controllerPolicies[record.ControllerRef] = record.Commitment
	state.nullifiers[record.Commitment] = record.SubjectRef
	state.score++
	state.reputation++
	state.economicMinor++
	return nil
}

func (state *stateMutatorSpy) snapshot() stateMutatorSpy {
	clone := *state
	clone.heads = cloneMap(state.heads)
	clone.controllerPolicies = cloneMap(state.controllerPolicies)
	clone.nullifiers = cloneMap(state.nullifiers)
	return clone
}

func cloneMap(source map[string]string) map[string]string {
	result := make(map[string]string, len(source))
	for key, value := range source {
		result[key] = value
	}
	return result
}

func mustAdmitter(t *testing.T, config witnesstypes.AdmissionConfig, verify keeper.VerifyFunc, policy keeper.ControllerPolicyFunc, mutate keeper.StateMutatorFunc) keeper.Admitter {
	t.Helper()
	admitter, err := keeper.NewAdmitter(config, verify, policy, mutate)
	if err != nil {
		t.Fatal(err)
	}
	return admitter
}

func enabledConfig(t *testing.T) witnesstypes.AdmissionConfig {
	t.Helper()
	audience, err := witnesstypes.ExpectedAudience(testChainID)
	if err != nil {
		t.Fatal(err)
	}
	return witnesstypes.AdmissionConfig{Enabled: true, ChainID: testChainID, Audience: audience}
}

func knownRecord(config witnesstypes.AdmissionConfig) witnesstypes.VerifiedRecord {
	return witnesstypes.VerifiedRecord{
		Protocol: witnesstypes.Protocol, Kind: witnesstypes.KindAgentToolCapability, Action: witnesstypes.ActionGrant,
		Audience: config.Audience, SubjectRef: strings.Repeat("1", 64), ControllerRef: strings.Repeat("2", 64),
		Commitment: "sha256:" + strings.Repeat("3", 64),
	}
}

func TestZeroValueIsDisabledBeforeParsingOrDependencies(t *testing.T) {
	verifier := &verifierSpy{err: errors.New("must not run")}
	policy := &controllerPolicySpy{}
	state := newStateMutatorSpy()
	before := state.snapshot()
	admitter := keeper.Admitter{}
	decision, err := admitter.Admit(context.Background(), bytes.Repeat([]byte{'x'}, witnesstypes.MaxRecordBytes+1))
	if !errors.Is(err, keeper.ErrDisabled) || decision.Admitted {
		t.Fatalf("zero-value config did not fail disabled: %#v, %v", decision, err)
	}
	if verifier.calls != 0 || policy.calls != 0 || state.calls != 0 || !reflect.DeepEqual(before, state.snapshot()) {
		t.Fatalf("disabled admission had effects: verifier=%d policy=%d state=%#v", verifier.calls, policy.calls, state)
	}
}

func TestRecordSizeBoundRunsBeforeVerification(t *testing.T) {
	config := enabledConfig(t)
	verifier := &verifierSpy{record: knownRecord(config)}
	policy := &controllerPolicySpy{}
	state := newStateMutatorSpy()
	admitter := mustAdmitter(t, config, verifier.Verify, policy.Authorize, state.Apply)

	if _, err := admitter.Admit(context.Background(), nil); !errors.Is(err, keeper.ErrEmptyRecord) {
		t.Fatalf("empty record: %v", err)
	}
	if _, err := admitter.Admit(context.Background(), bytes.Repeat([]byte{'x'}, witnesstypes.MaxRecordBytes+1)); !errors.Is(err, keeper.ErrRecordTooLarge) {
		t.Fatalf("oversize record: %v", err)
	}
	if verifier.calls != 0 {
		t.Fatalf("verifier saw preflight-refused bytes %d times", verifier.calls)
	}

	exact := bytes.Repeat([]byte{'x'}, witnesstypes.MaxRecordBytes)
	decision, err := admitter.Admit(context.Background(), exact)
	if !errors.Is(err, keeper.ErrNotConsensusAdmissible) || decision.Status != witnesstypes.ActivationStatusNotConsensusAdmissible || verifier.calls != 1 {
		t.Fatalf("exact 32 KiB boundary: %#v, %v, verifier=%d", decision, err, verifier.calls)
	}
	if policy.calls != 0 || state.calls != 0 {
		t.Fatalf("blocked boundary reached policy/state: policy=%d state=%d", policy.calls, state.calls)
	}
}

func TestCallerMustSupplyExactZeroneAudienceBinding(t *testing.T) {
	valid := enabledConfig(t)
	for _, config := range []witnesstypes.AdmissionConfig{
		{Enabled: true},
		{Enabled: true, ChainID: testChainID, Audience: "kingdom:offline-shadow"},
		{Enabled: true, ChainID: testChainID, Audience: "zerone:other-chain"},
		{Enabled: true, ChainID: "UPPER", Audience: "zerone:UPPER"},
		{Enabled: true, ChainID: "bad/chain", Audience: "zerone:bad/chain"},
	} {
		verifier := &verifierSpy{record: knownRecord(valid)}
		policy := &controllerPolicySpy{}
		state := newStateMutatorSpy()
		admitter := mustAdmitter(t, config, verifier.Verify, policy.Authorize, state.Apply)
		if _, err := admitter.Admit(context.Background(), []byte(`{}`)); !errors.Is(err, keeper.ErrAudienceConfiguration) {
			t.Fatalf("config %#v: got %v", config, err)
		}
		if verifier.calls != 0 {
			t.Fatalf("invalid audience config reached verifier: %#v", config)
		}
	}

	wrongRecord := knownRecord(valid)
	wrongRecord.Audience = "zerone:other-chain"
	verifier := &verifierSpy{record: wrongRecord}
	policy := &controllerPolicySpy{}
	state := newStateMutatorSpy()
	decision, err := mustAdmitter(t, valid, verifier.Verify, policy.Authorize, state.Apply).Admit(context.Background(), []byte(`{}`))
	if !errors.Is(err, keeper.ErrAudienceMismatch) || decision.Audience != wrongRecord.Audience {
		t.Fatalf("verified audience mismatch: %#v, %v", decision, err)
	}
	if policy.calls != 0 || state.calls != 0 {
		t.Fatal("audience mismatch reached policy/state")
	}
}

func TestConstructorRequiresEveryExplicitDependency(t *testing.T) {
	config := enabledConfig(t)
	verifier := &verifierSpy{record: knownRecord(config)}
	policy := &controllerPolicySpy{}
	state := newStateMutatorSpy()

	if _, err := keeper.NewAdmitter(config, nil, policy.Authorize, state.Apply); !errors.Is(err, keeper.ErrVerifierRequired) {
		t.Fatalf("nil verifier: %v", err)
	}
	if _, err := keeper.NewAdmitter(config, verifier.Verify, nil, state.Apply); !errors.Is(err, keeper.ErrControllerPolicyRequired) {
		t.Fatalf("nil policy: %v", err)
	}
	if _, err := keeper.NewAdmitter(config, verifier.Verify, policy.Authorize, nil); !errors.Is(err, keeper.ErrStateMutatorRequired) {
		t.Fatalf("nil state mutator: %v", err)
	}
}

func TestClosedMatrixAndReadinessExactlyMirrorFrozenProtocol(t *testing.T) {
	corePairs := make(map[string]bool)
	for kind, actions := range protocol.KindActionMatrix() {
		for _, action := range actions {
			corePairs[string(kind)+"/"+string(action)] = true
		}
	}
	scaffoldPairs := make(map[string]bool)
	for _, pair := range witnesstypes.ClosedActionPairs() {
		key := string(pair.Kind) + "/" + string(pair.Action)
		scaffoldPairs[key] = true
		if !witnesstypes.IsAllowedAction(pair.Kind, pair.Action) {
			t.Fatalf("listed pair is not allowed: %s", key)
		}
	}
	if len(scaffoldPairs) != 18 || !reflect.DeepEqual(scaffoldPairs, corePairs) {
		t.Fatalf("kind/action matrix drift\nscaffold=%v\ncore=%v", scaffoldPairs, corePairs)
	}

	coreReadiness := make(map[string]protocol.ActivationReadiness)
	for _, readiness := range protocol.ActivationReadinessMatrix() {
		coreReadiness[string(readiness.Kind)] = readiness
	}
	for _, readiness := range witnesstypes.CurrentActivationReadinessMatrix() {
		core, ok := coreReadiness[string(readiness.Kind)]
		if !ok || readiness.Status != core.Status || !reflect.DeepEqual(readiness.Blockers, core.Blockers) {
			t.Fatalf("readiness drift for %s: scaffold=%#v core=%#v", readiness.Kind, readiness, core)
		}
		if readiness.Status != witnesstypes.ActivationStatusNotConsensusAdmissible || len(readiness.Blockers) == 0 {
			t.Fatalf("kind unexpectedly admissible: %#v", readiness)
		}
	}
}

func TestEveryFrozenKindActionRejectsBeforePolicyAndState(t *testing.T) {
	config := enabledConfig(t)
	records := chainAudienceRecords(t, config.Audience)
	if len(records) != 18 {
		t.Fatalf("got %d unique signed action pairs, want 18", len(records))
	}
	verifier := &sharedProtocolVerifier{}
	policy := &controllerPolicySpy{}
	state := newStateMutatorSpy()
	before := state.snapshot()
	admitter := mustAdmitter(t, config, verifier.Verify, policy.Authorize, state.Apply)

	keys := make([]string, 0, len(records))
	for key := range records {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		decision, err := admitter.Admit(context.Background(), records[key])
		if !errors.Is(err, keeper.ErrNotConsensusAdmissible) {
			t.Fatalf("%s: expected readiness rejection, got %#v, %v", key, decision, err)
		}
		if decision.Admitted || decision.Status != witnesstypes.ActivationStatusNotConsensusAdmissible || len(decision.Blockers) == 0 {
			t.Fatalf("%s: unsafe decision %#v", key, decision)
		}
	}
	if verifier.calls != 18 {
		t.Fatalf("verified %d records, want 18", verifier.calls)
	}
	if policy.calls != 0 || state.calls != 0 || !reflect.DeepEqual(before, state.snapshot()) {
		t.Fatalf("blocked actions mutated effects: policy=%d before=%#v after=%#v", policy.calls, before, state.snapshot())
	}
}

func TestUnsupportedPairAndProtocolRejectBeforePolicyAndState(t *testing.T) {
	config := enabledConfig(t)
	policy := &controllerPolicySpy{}
	state := newStateMutatorSpy()
	for _, test := range []struct {
		name   string
		record witnesstypes.VerifiedRecord
		want   error
	}{
		{"protocol", func() witnesstypes.VerifiedRecord {
			record := knownRecord(config)
			record.Protocol = "other/1"
			return record
		}(), keeper.ErrProtocolMismatch},
		{"pair", func() witnesstypes.VerifiedRecord {
			record := knownRecord(config)
			record.Action = witnesstypes.ActionSettle
			return record
		}(), keeper.ErrUnsupportedKindAction},
		{"kind", func() witnesstypes.VerifiedRecord {
			record := knownRecord(config)
			record.Kind = "UNKNOWN"
			return record
		}(), keeper.ErrUnsupportedKindAction},
	} {
		t.Run(test.name, func(t *testing.T) {
			verifier := &verifierSpy{record: test.record}
			admitter := mustAdmitter(t, config, verifier.Verify, policy.Authorize, state.Apply)
			if _, err := admitter.Admit(context.Background(), []byte(`{}`)); !errors.Is(err, test.want) {
				t.Fatalf("got %v, want %v", err, test.want)
			}
		})
	}
	if policy.calls != 0 || state.calls != 0 {
		t.Fatalf("invalid protocol/pairs reached policy/state: policy=%d state=%d", policy.calls, state.calls)
	}
}

func TestVerifierReceivesDefensiveRecordCopy(t *testing.T) {
	config := enabledConfig(t)
	verifier := &verifierSpy{record: knownRecord(config), mutate: true}
	original := []byte(`{"immutable":true}`)
	want := append([]byte(nil), original...)
	policy := &controllerPolicySpy{}
	state := newStateMutatorSpy()
	_, err := mustAdmitter(t, config, verifier.Verify, policy.Authorize, state.Apply).Admit(context.Background(), original)
	if !errors.Is(err, keeper.ErrNotConsensusAdmissible) || !bytes.Equal(original, want) {
		t.Fatalf("caller bytes were mutated: %q, %v", original, err)
	}
}

func chainAudienceRecords(t *testing.T, audience string) map[string][]byte {
	t.Helper()
	directory := filepath.Join("..", "..", "..", "tools", "witness-v0", "testdata", "records")
	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatal(err)
	}
	result := make(map[string][]byte)
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(directory, entry.Name()))
		if err != nil {
			t.Fatal(err)
		}
		verified, err := protocol.Verify(raw)
		if err != nil {
			t.Fatalf("%s: %v", entry.Name(), err)
		}
		key := string(verified.Record.Envelope.Kind) + "/" + string(verified.Record.Envelope.Action)
		if _, exists := result[key]; exists {
			continue
		}
		result[key] = resignForAudience(t, *verified, audience)
	}
	return result
}

func resignForAudience(t *testing.T, verified protocol.VerifiedRecord, audience string) []byte {
	t.Helper()
	privateKey := fixturePrivateKey(t, verified.Record.Signature.PublicKey)
	envelope := verified.Record.Envelope
	envelope.Audience = audience
	payload := append([]byte(nil), verified.Record.Payload...)
	if envelope.Kind == protocol.KindAgentToolCapability && envelope.Action == protocol.ActionConsume {
		var consume protocol.CapabilityConsumePayload
		if err := json.Unmarshal(payload, &consume); err != nil {
			t.Fatal(err)
		}
		consume.Nullifier, _ = protocol.CapabilityNullifier(envelope, consume)
		payload, _ = json.Marshal(consume)
	}
	_, encoded, err := protocol.SignRecord(envelope, payload, privateKey)
	if err != nil {
		t.Fatal(err)
	}
	if len(encoded) > witnesstypes.MaxRecordBytes {
		t.Fatalf("fixture exceeds carrier cap: %d", len(encoded))
	}
	return encoded
}

func fixturePrivateKey(t *testing.T, publicKeyHex string) ed25519.PrivateKey {
	t.Helper()
	for _, fill := range []byte{1, 2} {
		key := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{fill}, ed25519.SeedSize))
		if hex.EncodeToString(key.Public().(ed25519.PublicKey)) == publicKeyHex {
			return key
		}
	}
	t.Fatalf("unknown fixture public key %s", publicKeyHex)
	return nil
}
