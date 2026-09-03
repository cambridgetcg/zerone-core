package protocol

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

type knownAnswerSchema struct {
	Kind       Kind   `json:"kind"`
	SchemaHash string `json:"schema_hash"`
}

type knownAnswerVector struct {
	Path                    string `json:"path"`
	FileSHA256              string `json:"file_sha256"`
	Operation               string `json:"operation"`
	Expected                string `json:"expected"`
	Stage                   string `json:"stage"`
	Code                    string `json:"code"`
	ErrorContains           string `json:"error_contains,omitempty"`
	Kind                    Kind   `json:"kind,omitempty"`
	Action                  Action `json:"action,omitempty"`
	Commitment              string `json:"commitment,omitempty"`
	PayloadRoot             string `json:"payload_root,omitempty"`
	MerkleRoot              string `json:"merkle_root,omitempty"`
	CanonicalHex            string `json:"canonical_hex,omitempty"`
	AcceptedRecords         string `json:"accepted_records,omitempty"`
	PermanentNullifierCount string `json:"permanent_nullifier_count,omitempty"`
}

type knownAnswerManifest struct {
	Protocol                  string              `json:"protocol"`
	FreezeState               string              `json:"freeze_state"`
	WirePolicy                string              `json:"wire_policy"`
	ActionPairCount           string              `json:"action_pair_count"`
	RecordSchemaHash          string              `json:"record_schema_hash"`
	SettlementBatchSchemaHash string              `json:"settlement_batch_schema_hash"`
	SchemaSetDigest           string              `json:"schema_set_digest"`
	PayloadSchemas            []knownAnswerSchema `json:"payload_schemas"`
	Vectors                   []knownAnswerVector `json:"vectors"`
	CorpusDigestAlgorithm     string              `json:"corpus_digest_algorithm"`
	CorpusDigest              string              `json:"corpus_digest"`
	EmptyMerkleRoot           string              `json:"empty_merkle_root"`
	SettlementMerkleRoot      string              `json:"settlement_merkle_root"`
	CapabilityNullifier       string              `json:"capability_nullifier"`
	UnicodeCanonicalSHA256    string              `json:"unicode_canonical_sha256"`
	UnicodeCanonicalHex       string              `json:"unicode_canonical_hex"`
}

type nullifierKnownAnswer struct {
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

type expectationKnownAnswer struct {
	Protocol string              `json:"protocol"`
	Vectors  []knownAnswerVector `json:"vectors"`
}

const (
	expectedRecordSchema = "sha256:71401ebb962d8909206b77acb6a07616727bd17663f5028e5d2745d911199005"
	expectedBatchSchema  = "sha256:4dfb561b0d395d556d5549e45301bb07b79beb089c3fd73e7fc643edcc7f02ec"
	expectedSchemaSet    = "sha256:d62e44643c8e1986336416237df26b76663728403d417a5ee9e83b6aa5baaaa5"
	expectedCorpus       = "sha256:b26b5cce4899aa62d6dee03e25471e2c80810008fbd07c2c3ac9170164e5352a"
	expectedMerkle       = "sha256:8f0995e7dcee1737603969d8e03ecccc1d7dbe5777c5bc4dc1d42d4b20025b52"
	expectedNullifier    = "sha256:462b53e09e2f35ed8009fa4d118dde8d87d04a64dfe9ca7279425e9ec73da899"
)

func loadKnownAnswerManifest(t *testing.T) (string, knownAnswerManifest) {
	t.Helper()
	root := filepath.Join("..", "testdata")
	manifestBytes, err := os.ReadFile(filepath.Join(root, "known-answer.json"))
	if err != nil {
		t.Fatal(err)
	}
	canonical, err := CanonicalJSON(manifestBytes)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(manifestBytes, canonical) {
		t.Fatal("known-answer manifest is not exact canonical JSON without trailing bytes")
	}
	var manifest knownAnswerManifest
	if err := strictUnmarshal(canonical, &manifest); err != nil {
		t.Fatal(err)
	}
	return root, manifest
}

func TestKnownAnswerIdentityAndSchemaSetArePinned(t *testing.T) {
	_, manifest := loadKnownAnswerManifest(t)
	if manifest.Protocol != Protocol || manifest.FreezeState != "FROZEN" || manifest.WirePolicy != "EXACT_CANONICAL_JSON_NO_TRAILING_BYTES" || manifest.ActionPairCount != "18" {
		t.Fatalf("unexpected manifest identity: %#v", manifest)
	}
	if manifest.RecordSchemaHash != expectedRecordSchema || manifest.SettlementBatchSchemaHash != expectedBatchSchema ||
		manifest.SchemaSetDigest != expectedSchemaSet || manifest.CorpusDigest != expectedCorpus ||
		manifest.SettlementMerkleRoot != expectedMerkle || manifest.CapabilityNullifier != expectedNullifier {
		t.Fatalf("known-answer identity drift: %#v", manifest)
	}
	if got, _ := RecordSchemaHash(); got != expectedRecordSchema {
		t.Fatalf("record schema drift: %s", got)
	}
	if got, _ := SettlementBatchSchemaHash(); got != expectedBatchSchema {
		t.Fatalf("batch schema drift: %s", got)
	}
	if got, _ := SchemaSetDigest(); got != expectedSchemaSet {
		t.Fatalf("schema-set drift: %s", got)
	}

	expectedPayloadSchemas := map[Kind]string{
		KindAgentToolCapability:        "sha256:0ee2f39395842e43b76a89b04a18c5f25a125f220a3aeac2694d9972c44a5148",
		KindAgentToolOffer:             "sha256:7eae4752f652a1d1e7fac391a74a0285dae2267ecac36cc6483dc4e53fab1a94",
		KindAgentToolPublicRecognition: "sha256:a3ece9072d305b86568abf275e53dbc6e31e025edad007fbc721f0082141fc8a",
		KindAgentToolSettlementRoot:    "sha256:34dfb9cc5add4301ccb9bb80038416b2ef843b89b48ef19d6c039a19575f7d59",
		KindArtifactLineage:            "sha256:60bd4992d30dfc6b699b2814ab3e8d152852866eeb40c9b97041f2a8070b4436",
		KindCollaborationCheckpoint:    "sha256:637307571a43ee9a593499bc87219bb2eb29cff5a3136fcb224e7327ccff3d53",
		KindDisputeTerminal:            "sha256:afa816d536e8321404a2c71c81d9478fdb6374b9b7acee436d07e05d5a0d54bb",
		KindIssuerKeyContinuity:        "sha256:13bd04c9e3c882c1bd3b5061a2dbfc867a96cc5d4e777ffe4978f22f56b72cef",
		KindKingdomReleaseRoot:         "sha256:15edb7e1726b1b23c117fc956f810a77437563062ae78040ad9ed367f1c120a9",
		KindWakePublicCheckpoint:       "sha256:0b9b5c63ea1760dadfce186b494633cf5f12247402dc207214e0174403fd9457",
	}
	if len(manifest.PayloadSchemas) != len(expectedPayloadSchemas) {
		t.Fatalf("payload schema count: got %d, want %d", len(manifest.PayloadSchemas), len(expectedPayloadSchemas))
	}
	for _, entry := range manifest.PayloadSchemas {
		if expectedPayloadSchemas[entry.Kind] != entry.SchemaHash {
			t.Fatalf("manifest schema drift for %s", entry.Kind)
		}
		if got, _ := ExpectedSchemaHash(entry.Kind); got != entry.SchemaHash {
			t.Fatalf("embedded schema drift for %s: %s", entry.Kind, got)
		}
	}
}

func TestEveryKnownAnswerVectorIsPinnedAndConforms(t *testing.T) {
	root, manifest := loadKnownAnswerManifest(t)
	paths := make([]string, 0, len(manifest.Vectors))
	fileSums := make(map[string][32]byte)
	seenPaths := make(map[string]bool)
	seenActions := make(map[Kind]map[Action]bool)
	for _, vector := range manifest.Vectors {
		vector := vector
		t.Run(strings.ReplaceAll(vector.Path, "/", "_"), func(t *testing.T) {
			if seenPaths[vector.Path] {
				t.Fatalf("duplicate manifest path %s", vector.Path)
			}
			seenPaths[vector.Path] = true
			contents, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(vector.Path)))
			if err != nil {
				t.Fatal(err)
			}
			sum := sha256.Sum256(contents)
			if digestString(sum) != vector.FileSHA256 {
				t.Fatalf("file digest drift: got %s, want %s", digestString(sum), vector.FileSHA256)
			}
			paths = append(paths, vector.Path)
			fileSums[vector.Path] = sum
			runKnownAnswerVector(t, vector, contents)
			if (vector.Operation == "VERIFY_RECORD" && vector.Expected == "ACCEPT") || vector.Operation == "VERIFY_RECORD_AND_ACTIVATION_AUDIT" {
				if seenActions[vector.Kind] == nil {
					seenActions[vector.Kind] = make(map[Action]bool)
				}
				seenActions[vector.Kind][vector.Action] = true
			}
		})
	}
	if !reflect.DeepEqual(seenActions, allowedActions) {
		t.Fatalf("shared corpus does not cover every closed kind/action pair\ngot=%v\nwant=%v", seenActions, allowedActions)
	}
	assertExpectationIndexPinsManifest(t, root, manifest)
	if countFiles(t, root)-1 != len(manifest.Vectors) {
		t.Fatalf("every corpus file except known-answer.json must be pinned: got %d files and %d vectors", countFiles(t, root), len(manifest.Vectors))
	}

	sort.Strings(paths)
	h := sha256.New()
	h.Write([]byte(Protocol))
	h.Write([]byte{0})
	h.Write([]byte("known-answer-corpus"))
	h.Write([]byte{0})
	for _, path := range paths {
		sum := fileSums[path]
		h.Write([]byte(path))
		h.Write([]byte{0})
		h.Write(sum[:])
		h.Write([]byte{0})
	}
	var corpus [32]byte
	copy(corpus[:], h.Sum(nil))
	if got := digestString(corpus); got != manifest.CorpusDigest || got != expectedCorpus {
		t.Fatalf("corpus digest drift: %s", got)
	}
}

func runKnownAnswerVector(t *testing.T, vector knownAnswerVector, contents []byte) {
	t.Helper()
	switch vector.Operation {
	case "CANONICALIZE":
		got, err := CanonicalJSON(contents)
		if vector.Expected == "REJECT" {
			requireRejected(t, err, vector)
			return
		}
		if err != nil || hex.EncodeToString(got) != vector.CanonicalHex {
			t.Fatalf("canonical result mismatch: %x, %v", got, err)
		}
	case "CANONICAL_WIRE":
		got, err := CanonicalJSON(contents)
		if err != nil || !bytes.Equal(got, contents) || hex.EncodeToString(got) != vector.CanonicalHex {
			t.Fatalf("canonical wire mismatch: %x, %v", got, err)
		}
	case "VERIFY_RECORD", "VERIFY_RECORD_AND_ACTIVATION_AUDIT":
		verified, err := Verify(contents)
		if vector.Expected == "REJECT" {
			requireRejected(t, err, vector)
			return
		}
		if err != nil {
			t.Fatal(err)
		}
		if verified.Record.Commitment != vector.Commitment || verified.Record.Envelope.PayloadRoot != vector.PayloadRoot ||
			verified.Record.Envelope.Kind != vector.Kind || verified.Record.Envelope.Action != vector.Action {
			t.Fatalf("record metadata drift: %#v", verified.Record)
		}
		if vector.Operation == "VERIFY_RECORD_AND_ACTIVATION_AUDIT" {
			audit := AuditActivation(*verified)
			if audit.Status != ActivationStatusNotConsensusAdmissible || len(audit.Blockers) == 0 {
				t.Fatalf("record became activation-admissible: %#v", audit)
			}
			if vector.Kind == KindAgentToolSettlementRoot && !containsString(audit.Blockers, "PERMANENT_CROSS_BATCH_RECEIPT_NULLIFIERS_OR_PROOFS") {
				t.Fatalf("settlement cross-batch blocker disappeared: %#v", audit)
			}
		}
	case "VERIFY_BATCH":
		_, root, err := VerifySettlementBatch(contents)
		if vector.Expected == "REJECT" {
			requireRejected(t, err, vector)
			return
		}
		if err != nil || root != vector.MerkleRoot {
			t.Fatalf("batch result: root=%s err=%v", root, err)
		}
	case "SIMULATE":
		result, err := Simulate(contents)
		if vector.Expected == "REJECT" {
			requireRejected(t, err, vector)
			return
		}
		if err != nil || result.AcceptedRecords != vector.AcceptedRecords || result.PermanentNullifierCount != vector.PermanentNullifierCount {
			t.Fatalf("simulation result: %#v err=%v", result, err)
		}
	case "SIMULATE_SETTLEMENT_REPLAY_AND_ACTIVATION_AUDIT":
		result, err := Simulate(contents)
		if err != nil || result.AcceptedRecords != vector.AcceptedRecords {
			t.Fatalf("structural replay demonstration changed: %#v err=%v", result, err)
		}
		var input SimulationInput
		if err := strictUnmarshal(contents, &input); err != nil {
			t.Fatal(err)
		}
		for _, raw := range input.Records {
			verified, err := Verify(raw)
			if err != nil {
				t.Fatal(err)
			}
			audit := AuditActivation(*verified)
			if audit.Status != ActivationStatusNotConsensusAdmissible || !containsString(audit.Blockers, "PERMANENT_CROSS_BATCH_RECEIPT_NULLIFIERS_OR_PROOFS") {
				t.Fatalf("replayed settlement became activation-admissible: %#v", audit)
			}
		}
	case "CHECK_NULLIFIER_DERIVATION":
		var known nullifierKnownAnswer
		if err := strictUnmarshal(contents, &known); err != nil {
			t.Fatal(err)
		}
		baseEnvelope := Envelope{Protocol: known.Protocol, Audience: known.Audience, SubjectRef: known.SubjectRef, Sequence: known.SequenceA}
		basePayload := CapabilityConsumePayload{CapabilityRef: known.CapabilityRef, GrantCommitment: known.GrantCommitment, AssetRef: known.AssetRef, SourceEventDigest: known.SourceEventDigest}
		gotA, err := CapabilityNullifier(baseEnvelope, basePayload)
		if err != nil || gotA != known.NullifierA {
			t.Fatalf("base nullifier: %s, %v", gotA, err)
		}
		baseEnvelope.Sequence = known.SequenceB
		gotB, _ := CapabilityNullifier(baseEnvelope, basePayload)
		basePayload.AssetRef = known.AlternativeAssetRef
		gotAsset, _ := CapabilityNullifier(baseEnvelope, basePayload)
		if gotB != known.NullifierB || gotA != gotB || gotAsset != known.AlternativeAssetNullifier || gotAsset == gotA {
			t.Fatalf("nullifier mutation semantics drift: a=%s b=%s asset=%s", gotA, gotB, gotAsset)
		}
	case "CHECK_EXPECTATION_INDEX":
		var index expectationKnownAnswer
		if err := strictUnmarshal(contents, &index); err != nil {
			t.Fatal(err)
		}
		if index.Protocol != Protocol || len(index.Vectors) == 0 {
			t.Fatalf("invalid expectation index identity: %#v", index)
		}
	default:
		t.Fatalf("unknown vector operation %q", vector.Operation)
	}
}

func assertExpectationIndexPinsManifest(t *testing.T, root string, manifest knownAnswerManifest) {
	t.Helper()
	contents, err := os.ReadFile(filepath.Join(root, "expectations.json"))
	if err != nil {
		t.Fatal(err)
	}
	var index expectationKnownAnswer
	if err := strictUnmarshal(contents, &index); err != nil {
		t.Fatal(err)
	}
	manifestByPath := make(map[string]knownAnswerVector, len(manifest.Vectors))
	for _, vector := range manifest.Vectors {
		if vector.Path != "expectations.json" {
			manifestByPath[vector.Path] = vector
		}
	}
	if len(index.Vectors) != len(manifestByPath) {
		t.Fatalf("expectation index has %d vectors, manifest has %d non-index vectors", len(index.Vectors), len(manifestByPath))
	}
	for _, expected := range index.Vectors {
		if got, ok := manifestByPath[expected.Path]; !ok || !reflect.DeepEqual(got, expected) {
			t.Fatalf("manifest expectation is not corpus-pinned for %s", expected.Path)
		}
	}
}

func requireRejected(t *testing.T, err error, vector knownAnswerVector) {
	t.Helper()
	if err == nil {
		t.Fatalf("%s/%s vector was accepted", vector.Stage, vector.Code)
	}
	if vector.ErrorContains != "" && !strings.Contains(err.Error(), vector.ErrorContains) {
		t.Fatalf("%s/%s rejected at wrong seam: %v", vector.Stage, vector.Code, err)
	}
}

func countFiles(t *testing.T, root string) int {
	t.Helper()
	count := 0
	err := filepath.WalkDir(root, func(_ string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() {
			count++
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return count
}

func containsString(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}
