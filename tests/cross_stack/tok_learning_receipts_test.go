//go:build tok_learning

package cross_stack_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	witnessprotocol "github.com/zerone-chain/zerone/tools/witness-v0/protocol"
)

// This test-only local format is not a witness-v0 schema, signature, ToK topology
// root, consensus use receipt, or payment/qualification eligibility adapter.
const (
	tokLearningReceiptFormat = "tok-fold-learning/0"
	tokLearningMaxObjects    = 22 // three bindings + six (payload, metadata, record) + run
	tokLearningMaxRunBytes   = 16 << 20
)

func tokLearningReceiptStages() [6]string {
	return [6]string{"evidence", "use", "feedback", "correction", "rerun", "attribution"}
}

type tokLearningReceiptBindings struct {
	TaskSHA256   string `json:"task_sha256"`
	PolicySHA256 string `json:"policy_sha256"`
	MethodSHA256 string `json:"method_sha256"`
}

// These are enforced envelope boundaries, not an interpretation of opaque payload
// claims. In particular scripted actors are not independently verified witnesses.
type tokLearningReceiptEffects struct {
	AttributionMode      string `json:"attribution_mode"`
	AllocationStatus     string `json:"allocation_status"`
	EconomicEffects      string `json:"economic_effects"`
	QualificationEffects string `json:"qualification_effects"`
	AuthorityEffects     string `json:"authority_effects"`
	LiveEffects          string `json:"live_effects"`
	IndependentWitnesses bool   `json:"independent_witnesses"`
}

func tokLearningZeroEffects() tokLearningReceiptEffects {
	return tokLearningReceiptEffects{
		AttributionMode: "DESCRIPTIVE_ARTIFACT_ROLES", AllocationStatus: "NOT_EVALUATED",
		EconomicEffects: "0", QualificationEffects: "0", AuthorityEffects: "0", LiveEffects: "0",
		IndependentWitnesses: false,
	}
}

type tokLearningCheckpoint struct {
	Format         string                     `json:"format"`
	Kind           string                     `json:"kind"`
	Sequence       string                     `json:"sequence"`
	Stage          string                     `json:"stage"`
	ParentSHA256   string                     `json:"parent_sha256"`
	Bindings       tokLearningReceiptBindings `json:"bindings"`
	PayloadSHA256  string                     `json:"payload_sha256"`
	MetadataSHA256 string                     `json:"metadata_sha256"`
	Effects        tokLearningReceiptEffects  `json:"effects"`
}

type tokLearningRunHead struct {
	Format      string                     `json:"format"`
	Kind        string                     `json:"kind"`
	Closed      bool                       `json:"closed"`
	Bindings    tokLearningReceiptBindings `json:"bindings"`
	Checkpoints []string                   `json:"checkpoints"`
	Effects     tokLearningReceiptEffects  `json:"effects"`
}

type tokLearningHeadPointer struct {
	Format     string `json:"format"`
	HeadSHA256 string `json:"head_sha256"`
}

type tokLearningReceiptObject struct {
	digest string
	data   []byte
}

// tokLearningReceiptObjectFor accepts JSON-compatible Go values or raw JSON via
// json.RawMessage. Do not decode numbers through float64: use UseNumber, and
// represent signed, fractional and large values as strings. Only the existing
// witness CanonicalJSON profile defines canonical bytes and bounds.
func tokLearningReceiptObjectFor(value any) (tokLearningReceiptObject, error) {
	var raw []byte
	var err error
	if encoded, ok := value.(json.RawMessage); ok {
		raw = encoded
	} else {
		raw, err = json.Marshal(value)
		if err != nil {
			return tokLearningReceiptObject{}, err
		}
		if err = tokLearningRejectFloats(reflect.ValueOf(value), 0); err != nil {
			return tokLearningReceiptObject{}, err
		}
	}
	canonical, err := witnessprotocol.CanonicalJSON(raw)
	if err != nil {
		return tokLearningReceiptObject{}, err
	}
	sum := sha256.Sum256(canonical)
	return tokLearningReceiptObject{hex.EncodeToString(sum[:]), canonical}, nil
}

func tokLearningRejectFloats(value reflect.Value, depth int) error {
	if !value.IsValid() {
		return nil
	}
	if depth > witnessprotocol.MaxJSONDepth {
		return fmt.Errorf("receipt value exceeds nesting bound")
	}
	switch value.Kind() {
	case reflect.Float32, reflect.Float64:
		return fmt.Errorf("receipt floats must be encoded as exact strings")
	case reflect.Interface, reflect.Pointer:
		if !value.IsNil() {
			return tokLearningRejectFloats(value.Elem(), depth)
		}
	case reflect.Map:
		iter := value.MapRange()
		for iter.Next() {
			if err := tokLearningRejectFloats(iter.Value(), depth+1); err != nil {
				return err
			}
		}
	case reflect.Array, reflect.Slice:
		if value.Type().Elem().Kind() == reflect.Uint8 {
			return nil
		}
		for i := 0; i < value.Len(); i++ {
			if err := tokLearningRejectFloats(value.Index(i), depth+1); err != nil {
				return err
			}
		}
	case reflect.Struct:
		for i := 0; i < value.NumField(); i++ {
			if value.Type().Field(i).IsExported() && value.Type().Field(i).Tag.Get("json") != "-" {
				if err := tokLearningRejectFloats(value.Field(i), depth+1); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// Strict round-trip equality also rejects absent zero-valued fields and Go's
// case-insensitive field aliases, which DisallowUnknownFields alone permits.
func tokLearningDecodeEnvelope(raw []byte, dst any) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		return err
	}
	object, err := tokLearningReceiptObjectFor(dst)
	if err != nil {
		return err
	}
	if !bytes.Equal(raw, object.data) {
		return fmt.Errorf("receipt envelope must have exactly its canonical typed fields")
	}
	return nil
}

// The builder keeps private copies; Head seals it. A failed Append does not add
// orphan objects. Loaded runs are already sealed. No method executes payloads.
type tokLearningReceipts struct {
	bindings tokLearningReceiptBindings
	records  []string
	objects  map[string][]byte
	head     string
}

func newTokLearningReceipts(task, policy, method any) (*tokLearningReceipts, error) {
	values := []any{task, policy, method}
	objects := make([]tokLearningReceiptObject, len(values))
	for i, value := range values {
		object, err := tokLearningReceiptObjectFor(value)
		if err != nil {
			return nil, fmt.Errorf("binding %d: %w", i, err)
		}
		objects[i] = object
	}
	run := &tokLearningReceipts{
		bindings: tokLearningReceiptBindings{objects[0].digest, objects[1].digest, objects[2].digest},
		objects:  make(map[string][]byte),
	}
	if err := run.addObjects(objects...); err != nil {
		return nil, err
	}
	return run, nil
}

func (run *tokLearningReceipts) addObjects(objects ...tokLearningReceiptObject) error {
	pending := make(map[string][]byte)
	for _, object := range objects {
		if old, ok := run.objects[object.digest]; ok {
			if !bytes.Equal(old, object.data) {
				return fmt.Errorf("conflicting content for digest %s", object.digest)
			}
		} else if old, ok := pending[object.digest]; ok && !bytes.Equal(old, object.data) {
			return fmt.Errorf("conflicting batch content for digest %s", object.digest)
		} else {
			pending[object.digest] = object.data
		}
	}
	if len(run.objects)+len(pending) > tokLearningMaxObjects {
		return fmt.Errorf("receipt exceeds object count bound")
	}
	total := 0
	for _, data := range run.objects {
		total += len(data)
	}
	for _, data := range pending {
		total += len(data)
	}
	if total > tokLearningMaxRunBytes {
		return fmt.Errorf("receipt exceeds total byte bound")
	}
	for digest, data := range pending {
		run.objects[digest] = bytes.Clone(data)
	}
	return nil
}

func (run *tokLearningReceipts) Append(stage string, payload, metadata any) (string, error) {
	stages := tokLearningReceiptStages()
	index := len(run.records)
	if run.head != "" || index >= len(stages) || stage != stages[index] {
		return "", fmt.Errorf("receipt stage %q out of order or run sealed", stage)
	}
	body, err := tokLearningReceiptObjectFor(payload)
	if err != nil {
		return "", fmt.Errorf("%s payload: %w", stage, err)
	}
	meta, err := tokLearningReceiptObjectFor(metadata)
	if err != nil {
		return "", fmt.Errorf("%s metadata: %w", stage, err)
	}
	parent := ""
	if index > 0 {
		parent = run.records[index-1]
	}
	record, err := tokLearningReceiptObjectFor(tokLearningCheckpoint{
		Format: tokLearningReceiptFormat, Kind: "checkpoint", Sequence: strconv.Itoa(index + 1),
		Stage: stage, ParentSHA256: parent, Bindings: run.bindings,
		PayloadSHA256: body.digest, MetadataSHA256: meta.digest, Effects: tokLearningZeroEffects(),
	})
	if err != nil {
		return "", err
	}
	if err = run.addObjects(body, meta, record); err != nil {
		return "", err
	}
	run.records = append(run.records, record.digest)
	return record.digest, nil
}

func (run *tokLearningReceipts) Head() (string, error) {
	if run.head != "" {
		return run.head, nil
	}
	if len(run.records) != len(tokLearningReceiptStages()) {
		return "", fmt.Errorf("receipt run requires exactly six checkpoints")
	}
	object, err := tokLearningReceiptObjectFor(tokLearningRunHead{
		Format: tokLearningReceiptFormat, Kind: "run", Closed: true,
		Bindings: run.bindings, Checkpoints: run.records, Effects: tokLearningZeroEffects(),
	})
	if err != nil {
		return "", err
	}
	if err = run.addObjects(object); err != nil {
		return "", err
	}
	run.head = object.digest
	return run.head, nil
}

func (run *tokLearningReceipts) Bindings() (task, policy, method json.RawMessage) {
	return bytes.Clone(run.objects[run.bindings.TaskSHA256]),
		bytes.Clone(run.objects[run.bindings.PolicySHA256]), bytes.Clone(run.objects[run.bindings.MethodSHA256])
}

func (run *tokLearningReceipts) Stage(stage string) (payload, metadata json.RawMessage, err error) {
	for i, name := range tokLearningReceiptStages() {
		if name == stage && i < len(run.records) {
			var record tokLearningCheckpoint
			if err = tokLearningDecodeEnvelope(run.objects[run.records[i]], &record); err != nil {
				return nil, nil, err
			}
			return bytes.Clone(run.objects[record.PayloadSHA256]), bytes.Clone(run.objects[record.MetadataSHA256]), nil
		}
	}
	return nil, nil, fmt.Errorf("receipt stage %q is unavailable", stage)
}

// The caller chooses persistence explicitly. The parent must already exist, but
// the destination must not (even an empty directory or symlink is refused).
// Failures may leave partial output; only a valid head and its complete closure
// count as a run. A late flush error can also leave complete output. Neither is
// automatically removed or reused. Pick a new destination to retry.
func (run *tokLearningReceipts) Write(destination string) (string, error) {
	head, err := run.Head()
	if err != nil {
		return "", err
	}
	if err = run.validate(); err != nil {
		return "", err
	}
	parentPath, base, err := tokLearningDestination(destination)
	if err != nil {
		return "", err
	}
	parent, err := os.OpenRoot(parentPath)
	if err != nil {
		return "", err
	}
	defer parent.Close()
	if err = parent.Mkdir(base, 0700); err != nil {
		return "", fmt.Errorf("receipt destination must be fresh: %w", err)
	}
	root, err := tokLearningOpenDirectory(parent, base)
	if err != nil {
		return "", err
	}
	defer root.Close()
	if err = root.Mkdir("objects", 0700); err != nil {
		return "", err
	}
	digests := make([]string, 0, len(run.objects))
	for digest := range run.objects {
		digests = append(digests, digest)
	}
	sort.Strings(digests)
	for _, digest := range digests {
		if err = tokLearningWriteExclusive(root, "objects/"+digest+".json", run.objects[digest]); err != nil {
			return "", err
		}
	}
	if err = tokLearningSyncDirectory(root, "objects"); err != nil {
		return "", err
	}
	pointer, err := tokLearningReceiptObjectFor(tokLearningHeadPointer{tokLearningReceiptFormat, head})
	if err != nil {
		return "", err
	}
	// Head is the final write. A truncated head is rejected by the loader; all
	// referenced content has been flushed before a complete head can appear.
	if err = tokLearningWriteExclusive(root, "head.json", pointer.data); err != nil {
		return "", err
	}
	if err = tokLearningSyncDirectory(root, "."); err != nil {
		return "", err
	}
	if err = tokLearningSyncDirectory(parent, "."); err != nil {
		return "", err
	}
	return head, nil
}

func tokLearningDestination(path string) (parent, base string, err error) {
	if path == "" || strings.IndexByte(path, 0) >= 0 {
		return "", "", fmt.Errorf("receipt path is empty or invalid")
	}
	for _, part := range strings.Split(filepath.ToSlash(path), "/") {
		if part == ".." {
			return "", "", fmt.Errorf("receipt path cannot contain parent traversal")
		}
	}
	absolute, err := filepath.Abs(path)
	if err != nil || absolute == string(filepath.Separator) {
		return "", "", fmt.Errorf("receipt path must name a child directory")
	}
	// Resolve the existing parent (macOS /tmp and /var are normal symlinks),
	// never the destination itself. Root-relative operations confine all I/O.
	parent, err = filepath.EvalSymlinks(filepath.Dir(absolute))
	return parent, filepath.Base(absolute), err
}

func tokLearningOpenDirectory(parent *os.Root, name string) (*os.Root, error) {
	before, err := parent.Lstat(name)
	if err != nil {
		return nil, err
	}
	if !before.IsDir() || before.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("receipt directory %q must not be a symlink", name)
	}
	root, err := parent.OpenRoot(name)
	if err != nil {
		return nil, err
	}
	after, err := root.Stat(".")
	if err != nil || !os.SameFile(before, after) {
		root.Close()
		return nil, fmt.Errorf("receipt directory changed during open")
	}
	return root, nil
}

func tokLearningWriteExclusive(root *os.Root, path string, data []byte) error {
	file, err := root.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	defer file.Close()
	n, err := file.Write(data)
	if err != nil {
		return err
	}
	if n != len(data) {
		return io.ErrShortWrite
	}
	if err = file.Chmod(0400); err != nil {
		return err
	}
	if err = file.Sync(); err != nil {
		return err
	}
	return file.Close()
}

func tokLearningSyncDirectory(root *os.Root, path string) error {
	file, err := root.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	return file.Sync()
}

func tokLearningReadFile(root *os.Root, name string, limit int) ([]byte, error) {
	return tokLearningReadFileWithOpenHook(root, name, limit, nil)
}

// A per-call hook schedules deterministic post-Lstat replacements in tests. It
// cannot substitute the root-confined opener and is never set by the loader.
func tokLearningReadFileWithOpenHook(root *os.Root, name string, limit int, beforeOpen func()) ([]byte, error) {
	before, err := root.Lstat(name)
	if err != nil {
		return nil, err
	}
	if !before.Mode().IsRegular() || before.Size() > int64(limit) {
		return nil, fmt.Errorf("receipt file %q is not a bounded regular file", name)
	}
	if beforeOpen != nil {
		beforeOpen()
	}
	file, err := tokLearningOpenReadFile(root, name)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	after, err := file.Stat()
	if err != nil || !after.Mode().IsRegular() || !os.SameFile(before, after) || after.Size() > int64(limit) {
		return nil, fmt.Errorf("receipt file changed during open")
	}
	data, err := io.ReadAll(io.LimitReader(file, int64(limit)+1))
	if err != nil {
		return nil, err
	}
	if len(data) > limit {
		return nil, fmt.Errorf("receipt file %q exceeds byte bound", name)
	}
	return data, nil
}

func tokLearningDirectoryNames(root *os.Root, limit int) ([]string, error) {
	file, err := root.Open(".")
	if err != nil {
		return nil, err
	}
	defer file.Close()
	names, err := file.Readdirnames(limit + 1)
	if err != nil && err != io.EOF {
		return nil, err
	}
	if len(names) > limit {
		return nil, fmt.Errorf("receipt directory exceeds entry bound")
	}
	sort.Strings(names)
	return names, nil
}

// Loading is read-only and suitable for a fresh process. It checks local byte
// integrity and closure, not computational correctness or actor identity. Replay
// computations belong to the integration test's fixed trusted code. Without a
// previously pinned expectedHead, substitution of another whole valid run (or
// rollback) is not detectable. Receipt strings are never treated as paths/code.
func loadTokLearningReceipts(directory, expectedHead string) (*tokLearningReceipts, error) {
	if expectedHead != "" && !tokLearningValidDigest(expectedHead) {
		return nil, fmt.Errorf("invalid expected receipt head")
	}
	parentPath, base, err := tokLearningDestination(directory)
	if err != nil {
		return nil, err
	}
	parent, err := os.OpenRoot(parentPath)
	if err != nil {
		return nil, err
	}
	defer parent.Close()
	root, err := tokLearningOpenDirectory(parent, base)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	names, err := tokLearningDirectoryNames(root, 2)
	if err != nil {
		return nil, err
	}
	if !reflect.DeepEqual(names, []string{"head.json", "objects"}) {
		return nil, fmt.Errorf("receipt directory is incomplete or contains unrelated entries")
	}
	raw, err := tokLearningReadFile(root, "head.json", 1024)
	if err != nil {
		return nil, err
	}
	var pointer tokLearningHeadPointer
	if err = tokLearningDecodeEnvelope(raw, &pointer); err != nil {
		return nil, err
	}
	if pointer.Format != tokLearningReceiptFormat || !tokLearningValidDigest(pointer.HeadSHA256) {
		return nil, fmt.Errorf("invalid receipt head pointer")
	}
	if expectedHead != "" && pointer.HeadSHA256 != expectedHead {
		return nil, fmt.Errorf("receipt head does not match expected pin")
	}
	blobs, err := tokLearningOpenDirectory(root, "objects")
	if err != nil {
		return nil, err
	}
	defer blobs.Close()
	names, err = tokLearningDirectoryNames(blobs, tokLearningMaxObjects)
	if err != nil {
		return nil, err
	}
	run := &tokLearningReceipts{head: pointer.HeadSHA256, objects: make(map[string][]byte)}
	for _, name := range names {
		digest := strings.TrimSuffix(name, ".json")
		if name != digest+".json" || !tokLearningValidDigest(digest) {
			return nil, fmt.Errorf("invalid receipt object filename")
		}
		data, err := tokLearningReadFile(blobs, name, witnessprotocol.MaxDocumentBytes)
		if err != nil {
			return nil, err
		}
		object, err := tokLearningReceiptObjectFor(json.RawMessage(data))
		if err != nil {
			return nil, err
		}
		if object.digest != digest || !bytes.Equal(object.data, data) {
			return nil, fmt.Errorf("receipt object is noncanonical or digest mismatched")
		}
		if err = run.addObjects(object); err != nil {
			return nil, err
		}
	}
	if err = run.validate(); err != nil {
		return nil, err
	}
	return run, nil
}

func tokLearningValidDigest(digest string) bool {
	if len(digest) != sha256.Size*2 {
		return false
	}
	for _, c := range digest {
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return false
		}
	}
	return true
}

func (run *tokLearningReceipts) validate() error {
	seen := make(map[string]bool)
	get := func(digest string) ([]byte, error) {
		data, ok := run.objects[digest]
		if !tokLearningValidDigest(digest) || !ok {
			return nil, fmt.Errorf("missing or invalid receipt object reference %q", digest)
		}
		object, err := tokLearningReceiptObjectFor(json.RawMessage(data))
		if err != nil || object.digest != digest || !bytes.Equal(data, object.data) {
			return nil, fmt.Errorf("invalid receipt object bytes for %s", digest)
		}
		seen[digest] = true
		return data, nil
	}
	raw, err := get(run.head)
	if err != nil {
		return err
	}
	var head tokLearningRunHead
	if err = tokLearningDecodeEnvelope(raw, &head); err != nil {
		return err
	}
	stages := tokLearningReceiptStages()
	if head.Format != tokLearningReceiptFormat || head.Kind != "run" || !head.Closed ||
		len(head.Checkpoints) != len(stages) || head.Effects != tokLearningZeroEffects() {
		return fmt.Errorf("receipt head violates closure, stages or zero-effect boundary")
	}
	for _, digest := range []string{head.Bindings.TaskSHA256, head.Bindings.PolicySHA256, head.Bindings.MethodSHA256} {
		if _, err = get(digest); err != nil {
			return err
		}
	}
	parent := ""
	for i, digest := range head.Checkpoints {
		raw, err = get(digest)
		if err != nil {
			return err
		}
		var record tokLearningCheckpoint
		if err = tokLearningDecodeEnvelope(raw, &record); err != nil {
			return err
		}
		if record.Format != tokLearningReceiptFormat || record.Kind != "checkpoint" ||
			record.Stage != stages[i] || record.Sequence != strconv.Itoa(i+1) || record.ParentSHA256 != parent ||
			record.Bindings != head.Bindings || record.Effects != tokLearningZeroEffects() {
			return fmt.Errorf("receipt checkpoint %d violates order, parent, bindings or zero-effect boundary", i+1)
		}
		for _, ref := range []string{record.PayloadSHA256, record.MetadataSHA256} {
			if _, err = get(ref); err != nil {
				return err
			}
		}
		parent = digest
	}
	if len(seen) != len(run.objects) {
		return fmt.Errorf("receipt contains objects outside the closed run")
	}
	run.bindings = head.Bindings
	run.records = append([]string(nil), head.Checkpoints...)
	return nil
}

func tokLearningTestReceipts(t *testing.T, variant string) *tokLearningReceipts {
	t.Helper()
	run, err := newTokLearningReceipts(
		json.RawMessage(`{"n":[3,5,7],"q":["1","3/2","2"]}`),
		map[string]any{"scope": "finite-local", "variant": variant},
		map[string]any{"method": "local-cross-check", "fixture_height": "123"},
	)
	if err != nil {
		t.Fatal(err)
	}
	for _, stage := range tokLearningReceiptStages() {
		_, err = run.Append(stage, map[string]any{"observed_stage": stage, "delta": "-1", "large": "9007199254740992"},
			map[string]any{"height": "123", "source": "scripted-local"})
		if err != nil {
			t.Fatal(err)
		}
	}
	return run
}

func TestToKLearningReceiptsCanonical(t *testing.T) {
	one, err := tokLearningReceiptObjectFor(json.RawMessage(` {"z":0,"a":[true,"3/2"]} `))
	if err != nil {
		t.Fatal(err)
	}
	two, err := tokLearningReceiptObjectFor(map[string]any{"a": []any{true, "3/2"}, "z": 0})
	if err != nil || !bytes.Equal(one.data, two.data) || one.digest != two.digest {
		t.Fatalf("canonical order mismatch: %v", err)
	}
	sum := sha256.Sum256([]byte(`{"a":[true,"3/2"],"z":0}`))
	if one.digest != hex.EncodeToString(sum[:]) {
		t.Fatal("digest is not SHA256 of canonical bytes")
	}
	for name, value := range map[string]any{
		"duplicate":         json.RawMessage(`{"x":1,"x":2}`),
		"escaped_duplicate": json.RawMessage("{\"x\":1,\"\\u0078\":2}"),
		"float":             json.RawMessage(`{"x":1.0}`), "exponent": json.RawMessage(`1e2`),
		"go_integral_float": map[string]any{"x": float64(1)},
		"negative":          json.RawMessage(`-1`), "unsafe": json.RawMessage(`9007199254740992`),
		"trailing": json.RawMessage(`{} {}`), "unknown_value": make(chan int),
		"oversize": json.RawMessage(bytes.Repeat([]byte(" "), witnessprotocol.MaxDocumentBytes+1)),
		"depth":    json.RawMessage(strings.Repeat("[", 34) + "0" + strings.Repeat("]", 34)),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := tokLearningReceiptObjectFor(value); err == nil {
				t.Fatal("accepted invalid canonical input")
			}
		})
	}
}

func TestToKLearningReceiptsRoundTrip(t *testing.T) {
	run := tokLearningTestReceipts(t, "a")
	path := filepath.Join(t.TempDir(), "receipt")
	head, err := run.Write(path)
	if err != nil {
		t.Fatal(err)
	}
	again := tokLearningTestReceipts(t, "a")
	againHead, err := again.Head()
	if err != nil || head != againHead {
		t.Fatalf("nondeterministic head: %v", err)
	}
	loaded, err := loadTokLearningReceipts(path, head)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(run.objects, loaded.objects) || !reflect.DeepEqual(run.records, loaded.records) {
		t.Fatal("retained content differs after load")
	}
	for _, stage := range tokLearningReceiptStages() {
		payload, metadata, err := loaded.Stage(stage)
		wantPayload, wantMetadata, _ := run.Stage(stage)
		if err != nil || !bytes.Equal(payload, wantPayload) || !bytes.Equal(metadata, wantMetadata) {
			t.Fatalf("stage %s differs: %v", stage, err)
		}
		payload[0], metadata[0] = 'x', 'x'
		stillPayload, stillMetadata, _ := loaded.Stage(stage)
		if !bytes.Equal(stillPayload, wantPayload) || !bytes.Equal(stillMetadata, wantMetadata) {
			t.Fatal("stage accessor leaked mutable storage")
		}
	}
	task, _, _ := loaded.Bindings()
	task[0] = 'x'
	taskAgain, _, _ := loaded.Bindings()
	if taskAgain[0] == 'x' {
		t.Fatal("binding accessor leaked mutable storage")
	}
	if _, err = loaded.Append("attribution", nil, nil); err == nil {
		t.Fatal("loaded run accepted append")
	}
	other := tokLearningTestReceipts(t, "b")
	otherHead, _ := other.Head()
	if _, err = loadTokLearningReceipts(path, otherHead); err == nil {
		t.Fatal("wrong pinned head accepted")
	}
	if _, err = loadTokLearningReceipts(path, ""); err != nil {
		t.Fatal(err)
	}
	if _, err = loadTokLearningReceipts(path, "../head"); err == nil {
		t.Fatal("invalid expected head accepted")
	}
}

func TestToKLearningReceiptsFreshProcess(t *testing.T) {
	// The only subprocess is this fixed trusted test executable. No path or
	// command from receipt content is executable, and no integration flags exist.
	if directory := os.Getenv("TOK_LEARNING_RECEIPTS_TEST_DIR"); directory != "" {
		run, err := loadTokLearningReceipts(directory, os.Getenv("TOK_LEARNING_RECEIPTS_TEST_HEAD"))
		if err != nil {
			t.Fatal(err)
		}
		for _, stage := range tokLearningReceiptStages() {
			if _, _, err = run.Stage(stage); err != nil {
				t.Fatal(err)
			}
		}
		return
	}
	run := tokLearningTestReceipts(t, "fresh-process")
	path := filepath.Join(t.TempDir(), "run")
	head, err := run.Write(path)
	if err != nil {
		t.Fatal(err)
	}
	before, err := os.Stat(filepath.Join(path, "head.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, directory := range []string{path, filepath.Join(path, "objects")} {
		t.Cleanup(func() { _ = os.Chmod(directory, 0700) })
		if err = os.Chmod(directory, 0500); err != nil {
			t.Fatal(err)
		}
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, executable, "-test.run=^TestToKLearningReceiptsFreshProcess$")
	command.Env = append(os.Environ(), "TOK_LEARNING_RECEIPTS_TEST_DIR="+path, "TOK_LEARNING_RECEIPTS_TEST_HEAD="+head)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("fresh-process read-only load: %v: %s", err, output)
	}
	after, err := os.Stat(filepath.Join(path, "head.json"))
	if err != nil || before.ModTime() != after.ModTime() || before.Mode() != after.Mode() {
		t.Fatalf("loader changed the published head: %v", err)
	}
	loaded, err := loadTokLearningReceipts(path, head)
	if err != nil || !reflect.DeepEqual(loaded.objects, run.objects) {
		t.Fatalf("fresh process changed retained bytes: %v", err)
	}
}

func TestToKLearningReceiptsStorageBounds(t *testing.T) {
	t.Run("object_count", func(t *testing.T) {
		run := &tokLearningReceipts{objects: make(map[string][]byte)}
		for i := 0; i <= tokLearningMaxObjects; i++ {
			object, err := tokLearningReceiptObjectFor(i)
			if err != nil {
				t.Fatal(err)
			}
			err = run.addObjects(object)
			if i < tokLearningMaxObjects && err != nil {
				t.Fatal(err)
			}
			if i == tokLearningMaxObjects && (err == nil || len(run.objects) != tokLearningMaxObjects) {
				t.Fatal("object bound absent or failed addition mutated storage")
			}
		}
	})
	t.Run("total_bytes", func(t *testing.T) {
		run := &tokLearningReceipts{objects: make(map[string][]byte)}
		chunks := make([]string, 16)
		for i := range chunks {
			chunks[i] = strings.Repeat("x", 63<<10)
		}
		for i := 0; i < tokLearningMaxObjects; i++ {
			object, err := tokLearningReceiptObjectFor(map[string]any{"index": i, "chunks": chunks})
			if err != nil {
				t.Fatal(err)
			}
			if err = run.addObjects(object); err != nil {
				if !strings.Contains(err.Error(), "total byte bound") || len(run.objects) != i {
					t.Fatalf("wrong bound or non-atomic failure: %v", err)
				}
				return
			}
		}
		t.Fatal("total byte bound was not enforced")
	})
	t.Run("conflicting_object", func(t *testing.T) {
		run, err := newTokLearningReceipts(nil, nil, nil)
		if err != nil {
			t.Fatal(err)
		}
		if err = run.addObjects(tokLearningReceiptObject{run.bindings.TaskSHA256, []byte("true")}); err == nil {
			t.Fatal("conflicting digest overwrite accepted")
		}
	})
}

func TestToKLearningReceiptsOrderAndNonoverwrite(t *testing.T) {
	run, err := newTokLearningReceipts(nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	before := len(run.objects)
	if _, err = run.Append("use", nil, nil); err == nil {
		t.Fatal("skipped evidence accepted")
	}
	if _, err = run.Append("evidence", map[string]any{"valid": true}, json.RawMessage(`1.5`)); err == nil {
		t.Fatal("invalid metadata accepted")
	}
	if len(run.objects) != before || len(run.records) != 0 {
		t.Fatal("failed append mutated run")
	}
	parent := t.TempDir()
	incomplete := filepath.Join(parent, "incomplete")
	if _, err = run.Write(incomplete); err == nil {
		t.Fatal("incomplete run published")
	}
	if _, err = os.Lstat(incomplete); !os.IsNotExist(err) {
		t.Fatal("incomplete write touched destination")
	}
	run = tokLearningTestReceipts(t, "a")
	for _, existing := range []string{parent, filepath.Join(parent, "file"), filepath.Join(parent, "empty")} {
		if existing != parent {
			if filepath.Base(existing) == "file" {
				err = os.WriteFile(existing, []byte("unrelated"), 0600)
			} else {
				err = os.Mkdir(existing, 0700)
			}
			if err != nil {
				t.Fatal(err)
			}
		}
		if _, err = run.Write(existing); err == nil {
			t.Fatalf("overwrote existing %s", existing)
		}
	}
	path := filepath.Join(parent, "valid")
	if _, err = run.Write(path); err != nil {
		t.Fatal(err)
	}
	if _, err = run.Write(path); err == nil {
		t.Fatal("even matching run must not overwrite")
	}
	if _, err = run.Append("extra", nil, nil); err == nil {
		t.Fatal("seventh checkpoint accepted")
	}
	if _, _, err = run.Stage("unknown"); err == nil {
		t.Fatal("unknown stage accepted")
	}
}

func TestToKLearningReceiptsTampering(t *testing.T) {
	for _, scenario := range []string{"blob", "missing_blob", "noncanonical", "truncated_head", "missing_head", "extra_file", "extra_blob", "oversize_blob", "root_symlink", "objects_symlink", "blob_symlink", "head_symlink", "traversal"} {
		t.Run(scenario, func(t *testing.T) {
			run := tokLearningTestReceipts(t, "a")
			parent := t.TempDir()
			path := filepath.Join(parent, "run")
			head, err := run.Write(path)
			if err != nil {
				t.Fatal(err)
			}
			blob := filepath.Join(path, "objects", run.bindings.TaskSHA256+".json")
			replace := func(name string, raw []byte) {
				t.Helper()
				if err := os.Remove(name); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(name, raw, 0600); err != nil {
					t.Fatal(err)
				}
			}
			switch scenario {
			case "blob":
				replace(blob, []byte(`{"tampered":true}`))
			case "missing_blob":
				err = os.Remove(blob)
			case "noncanonical":
				replace(blob, append([]byte(" "), run.objects[run.bindings.TaskSHA256]...))
			case "truncated_head":
				replace(filepath.Join(path, "head.json"), []byte(`{"format":`))
			case "missing_head":
				err = os.Remove(filepath.Join(path, "head.json"))
			case "extra_file":
				err = os.WriteFile(filepath.Join(path, "other"), nil, 0600)
			case "extra_blob":
				object, _ := tokLearningReceiptObjectFor("unreferenced")
				err = os.WriteFile(filepath.Join(path, "objects", object.digest+".json"), object.data, 0600)
			case "oversize_blob":
				replace(blob, bytes.Repeat([]byte(" "), witnessprotocol.MaxDocumentBytes+1))
			case "root_symlink":
				link := filepath.Join(parent, "link")
				err = os.Symlink(path, link)
				path = link
				if _, writeErr := run.Write(link); writeErr == nil {
					t.Fatal("writer accepted destination symlink")
				}
			case "objects_symlink":
				err = os.Rename(filepath.Join(path, "objects"), filepath.Join(parent, "moved"))
				if err == nil {
					err = os.Symlink(filepath.Join(parent, "moved"), filepath.Join(path, "objects"))
				}
			case "blob_symlink", "head_symlink":
				name := blob
				if scenario == "head_symlink" {
					name = filepath.Join(path, "head.json")
				}
				err = os.Rename(name, filepath.Join(parent, "moved"))
				if err == nil {
					err = os.Symlink(filepath.Join(parent, "moved"), name)
				}
			case "traversal":
				path = path + "/../run"
			}
			if err != nil {
				t.Fatal(err)
			}
			if _, err = loadTokLearningReceipts(path, head); err == nil {
				t.Fatal("accepted tampered or unsafe receipt")
			}
		})
	}
}

// Rehash mutations to prove envelope validation does more than verify digests.
func TestToKLearningReceiptsClosedEnvelope(t *testing.T) {
	for _, scenario := range []string{"stage", "sequence", "parent", "binding", "effect", "allocation", "missing_effect", "alias", "extra_field", "open_root", "missing_stage", "root_effect", "missing_payload"} {
		t.Run(scenario, func(t *testing.T) {
			run := tokLearningTestReceipts(t, "a")
			if _, err := run.Head(); err != nil {
				t.Fatal(err)
			}
			var head tokLearningRunHead
			if err := tokLearningDecodeEnvelope(run.objects[run.head], &head); err != nil {
				t.Fatal(err)
			}
			index := len(head.Checkpoints) - 1
			oldRecord := head.Checkpoints[index]
			var record map[string]any
			if err := json.Unmarshal(run.objects[oldRecord], &record); err != nil {
				t.Fatal(err)
			}
			switch scenario {
			case "stage":
				record["stage"] = "rerun"
			case "sequence":
				record["sequence"] = "06"
			case "parent":
				record["parent_sha256"] = head.Checkpoints[0]
			case "binding":
				record["bindings"].(map[string]any)["task_sha256"] = head.Bindings.MethodSHA256
			case "effect":
				record["effects"].(map[string]any)["live_effects"] = "1"
			case "allocation":
				record["effects"].(map[string]any)["allocation_status"] = "ALLOCATED"
			case "missing_effect":
				delete(record["effects"].(map[string]any), "independent_witnesses")
			case "alias":
				record["Stage"] = record["stage"]
				delete(record, "stage")
			case "extra_field":
				record["execute"] = "never"
			case "open_root":
				head.Closed = false
			case "missing_stage":
				head.Checkpoints = head.Checkpoints[:5]
			case "root_effect":
				head.Effects.IndependentWitnesses = true
			case "missing_payload":
				record["payload_sha256"] = strings.Repeat("0", 64)
			}
			object, err := tokLearningReceiptObjectFor(record)
			if err != nil {
				t.Fatal(err)
			}
			if scenario != "missing_stage" {
				delete(run.objects, oldRecord)
				run.objects[object.digest] = object.data
				head.Checkpoints[index] = object.digest
			}
			delete(run.objects, run.head)
			object, err = tokLearningReceiptObjectFor(head)
			if err != nil {
				t.Fatal(err)
			}
			run.head = object.digest
			run.objects[object.digest] = object.data
			if err = run.validate(); err == nil {
				t.Fatal("accepted rehashed invalid envelope")
			}
		})
	}
}
