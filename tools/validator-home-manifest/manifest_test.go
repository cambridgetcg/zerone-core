package main

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const testAppHash = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

func TestCreateAndVerifyManifest(t *testing.T) {
	fixture := makeTestManifestFixture(t, "vol_test_001")

	manifest, err := createManifest(fixture.Source, fixture.CreateOptions)
	if err != nil {
		t.Fatalf("create manifest: %v", err)
	}
	if manifest.LastHeight != 41 || manifest.AppHash != testAppHash {
		t.Fatalf("unexpected observed state: %+v", manifest)
	}
	if len(manifest.Databases) != 3 {
		t.Fatalf("database groups = %d, want 3", len(manifest.Databases))
	}
	document, err := canonicalDocument(manifest)
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range fixture.PrivateMaterial {
		if bytes.Contains(document, []byte(secret)) {
			t.Fatalf("manifest leaked private material %q", secret)
		}
	}
	copyTestHome(t, fixture.Source, fixture.Destination)
	verified, err := verifyManifest(
		document,
		fixture.Destination,
		fixture.VerifyOptions,
		true,
	)
	if err != nil {
		t.Fatalf("verify manifest: %v", err)
	}
	if verified.ManifestSHA256 != manifest.ManifestSHA256 {
		t.Fatalf("verified self-hash changed")
	}
}

func TestVerifyRejectsContentAndVolumeDrift(t *testing.T) {
	fixture := makeTestManifestFixture(t, "vol_expected")
	manifest, err := createManifest(fixture.Source, fixture.CreateOptions)
	if err != nil {
		t.Fatal(err)
	}
	document, err := canonicalDocument(manifest)
	if err != nil {
		t.Fatal(err)
	}
	copyTestHome(t, fixture.Source, fixture.Destination)
	wrongVolume := fixture.VerifyOptions
	wrongVolume.DestinationVolumeID = "vol_wrong"
	if _, err := verifyManifest(document, fixture.Destination, wrongVolume, true); err == nil ||
		!strings.Contains(err.Error(), "volume mismatch") {
		t.Fatalf("wrong volume error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(fixture.Destination, "data", "application.db", "000001.log"), []byte("changed"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := verifyManifest(
		document,
		fixture.Destination,
		fixture.VerifyOptions,
		true,
	); err == nil ||
		!strings.Contains(err.Error(), "content does not match") {
		t.Fatalf("content drift error = %v", err)
	}
}

func TestScanHomeRejectsSymlink(t *testing.T) {
	home, _ := makeTestHome(t)
	if err := os.Symlink(
		filepath.Join(home, "config", "genesis.json"),
		filepath.Join(home, "data", "linked-genesis"),
	); err != nil {
		t.Fatal(err)
	}
	if _, err := scanHome(home); err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("symlink error = %v", err)
	}
}

func TestManifestRejectsNonCanonicalAndDuplicateJSON(t *testing.T) {
	fixture := makeTestManifestFixture(t, "vol_expected")
	manifest, err := createManifest(fixture.Source, fixture.CreateOptions)
	if err != nil {
		t.Fatal(err)
	}
	document, err := canonicalDocument(manifest)
	if err != nil {
		t.Fatal(err)
	}
	nonCanonical := append([]byte(" \n"), document...)
	copyTestHome(t, fixture.Source, fixture.Destination)
	if _, err := verifyManifest(
		nonCanonical,
		fixture.Destination,
		fixture.VerifyOptions,
		true,
	); err == nil ||
		!strings.Contains(err.Error(), "not exact canonical") {
		t.Fatalf("non-canonical error = %v", err)
	}
	ambiguous, err := os.ReadFile("testdata/duplicate-keys.json")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := verifyManifest(
		ambiguous,
		fixture.Destination,
		fixture.VerifyOptions,
		true,
	); err == nil ||
		!strings.Contains(err.Error(), "duplicate JSON key") {
		t.Fatalf("duplicate-key error = %v", err)
	}
}

func TestCreateStoppedEvidenceRejectsRunningPID(t *testing.T) {
	home, _ := makeTestHome(t)
	_, err := createStoppedEvidence(
		home,
		os.Getpid(),
		"2026-07-30T11:00:00Z",
		strings.Repeat("e", 64),
		"/does/not/matter-for-running-pid.json",
		41,
		testAppHash,
		"unit test",
		"observer-test",
	)
	if err == nil || !strings.Contains(err.Error(), "still running") {
		t.Fatalf("running PID error = %v", err)
	}
}

func TestStoppedEvidenceRejectsHeightSuccessorOverflow(t *testing.T) {
	evidence := StoppedEvidence{
		Schema:                       stoppedEvidenceSchema,
		CapturedAt:                   "2026-07-30T12:00:00Z",
		Method:                       "unit test",
		Observer:                     "observer-test",
		SourceHome:                   "/srv/zerone-source",
		ProcessID:                    999999,
		ProcessStartTime:             "2026-07-30T11:00:00Z",
		ProcessIdentitySHA256:        strings.Repeat("a", 64),
		ProcessAbsent:                true,
		RestartInhibitEvidenceSHA256: strings.Repeat("b", 64),
		LastHeight:                   math.MaxInt64,
		AppHash:                      testAppHash,
		EvidenceSHA256:               strings.Repeat("c", 64),
	}
	if err := validateStoppedEvidence(evidence); err == nil ||
		!strings.Contains(err.Error(), "successor") {
		t.Fatalf("height-overflow error = %v", err)
	}
}

func TestCreateRejectsMismatchedValidatorPrivateKey(t *testing.T) {
	home, _ := makeTestHome(t)
	path := filepath.Join(home, "config", "priv_validator_key.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var document validatorKeyDocument
	if err := decodeStrict(data, &document); err != nil {
		t.Fatal(err)
	}
	otherPrivate := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{0x44}, ed25519.SeedSize))
	document.PrivKey.Value = base64.StdEncoding.EncodeToString(otherPrivate)
	writeJSONFile(t, path, document, 0o600)
	if _, err := inspectConsensusIdentity(home); err == nil ||
		!strings.Contains(err.Error(), "does not match") {
		t.Fatalf("mismatched validator key error = %v", err)
	}
}

func TestRejectsForgedEd25519PublicSuffix(t *testing.T) {
	home, _ := makeTestHome(t)
	path := filepath.Join(home, "config", "priv_validator_key.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var document validatorKeyDocument
	if err := decodeStrict(data, &document); err != nil {
		t.Fatal(err)
	}
	privateKey, err := base64.StdEncoding.DecodeString(document.PrivKey.Value)
	if err != nil {
		t.Fatal(err)
	}
	privateKey[len(privateKey)-1] ^= 0xff
	document.PrivKey.Value = base64.StdEncoding.EncodeToString(privateKey)
	writeJSONFile(t, path, document, 0o600)
	if _, err := inspectConsensusIdentity(home); err == nil ||
		!strings.Contains(err.Error(), "does not derive from its seed") {
		t.Fatalf("forged public suffix error = %v", err)
	}
}

func TestRejectsLookalikeEd25519KeyType(t *testing.T) {
	home, _ := makeTestHome(t)
	path := filepath.Join(home, "config", "priv_validator_key.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var document validatorKeyDocument
	if err := decodeStrict(data, &document); err != nil {
		t.Fatal(err)
	}
	document.PubKey.Type = "untrusted/PubKeyEd25519"
	writeJSONFile(t, path, document, 0o600)
	if _, err := inspectConsensusIdentity(home); err == nil ||
		!strings.Contains(err.Error(), "public key must be") {
		t.Fatalf("lookalike key-type error = %v", err)
	}
}

func TestRejectsHardLinkedPrivateKey(t *testing.T) {
	home, _ := makeTestHome(t)
	keyPath := filepath.Join(home, "config", "priv_validator_key.json")
	if err := os.Link(keyPath, filepath.Join(home, "config", "validator-key-alias")); err != nil {
		t.Fatal(err)
	}
	if _, err := inspectConsensusIdentity(home); err == nil ||
		!strings.Contains(err.Error(), "exactly one hard link") {
		t.Fatalf("hard-linked private key error = %v", err)
	}
}

func TestRejectsInvalidCometSigningState(t *testing.T) {
	tests := []struct {
		name      string
		state     signingStateDocument
		wantError string
	}{
		{
			name: "short signature",
			state: func() signingStateDocument {
				signature := base64.StdEncoding.EncodeToString([]byte("short"))
				signBytes := base64.StdEncoding.EncodeToString([]byte("sign bytes"))
				return signingStateDocument{"41", 0, 3, &signature, &signBytes}
			}(),
			wantError: "exactly 64 bytes",
		},
		{
			name: "invalid step",
			state: func() signingStateDocument {
				signature := base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{1}, 64))
				signBytes := base64.StdEncoding.EncodeToString([]byte("sign bytes"))
				return signingStateDocument{"41", 0, 4, &signature, &signBytes}
			}(),
			wantError: "outside Comet bounds",
		},
		{
			name: "genesis with signature",
			state: func() signingStateDocument {
				signature := base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{1}, 64))
				signBytes := base64.StdEncoding.EncodeToString([]byte("sign bytes"))
				return signingStateDocument{"0", 0, 0, &signature, &signBytes}
			}(),
			wantError: "genesis signing state",
		},
		{
			name: "non-genesis unsigned step",
			state: signingStateDocument{
				Height: "41",
				Round:  0,
				Step:   0,
			},
			wantError: "non-genesis signing state",
		},
		{
			name: "signature without sign bytes",
			state: func() signingStateDocument {
				signature := base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{1}, 64))
				return signingStateDocument{"41", 0, 3, &signature, nil}
			}(),
			wantError: "present together",
		},
		{
			name: "empty sign bytes",
			state: func() signingStateDocument {
				signature := base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{1}, 64))
				signBytes := base64.StdEncoding.EncodeToString(nil)
				return signingStateDocument{"41", 0, 3, &signature, &signBytes}
			}(),
			wantError: "must not be empty",
		},
		{
			name: "genesis nonzero round",
			state: signingStateDocument{
				Height: "0",
				Round:  1,
				Step:   0,
			},
			wantError: "genesis signing state",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			home, _ := makeTestHome(t)
			writeJSONFile(
				t,
				filepath.Join(home, "data", "priv_validator_state.json"),
				test.state,
				0o600,
			)
			if _, err := inspectSigningState(home); err == nil ||
				!strings.Contains(err.Error(), test.wantError) {
				t.Fatalf("signing-state error = %v", err)
			}
		})
	}
}

func TestCreateRejectsSameDeviceAndNonFreshDestination(t *testing.T) {
	fixture := makeTestManifestFixture(t, "vol_expected")
	oldReader := readFilesystemIdentity
	readFilesystemIdentity = func(path string) (FilesystemIdentity, error) {
		identity, err := oldReader(path)
		if err != nil {
			return FilesystemIdentity{}, err
		}
		if filepath.Clean(path) == fixture.Destination {
			identity.DeviceID = 101
		}
		return identity, nil
	}
	if _, err := createManifest(fixture.Source, fixture.CreateOptions); err == nil ||
		!strings.Contains(err.Error(), "different filesystem device") {
		t.Fatalf("same-device error = %v", err)
	}
	readFilesystemIdentity = oldReader

	if err := os.WriteFile(filepath.Join(fixture.Destination, "unexpected"), []byte("data"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := createManifest(fixture.Source, fixture.CreateOptions); err == nil ||
		!strings.Contains(err.Error(), "not empty") {
		t.Fatalf("non-fresh destination error = %v", err)
	}
}

func TestCreateRejectsStaleSnapshotAndVolumeEvidence(t *testing.T) {
	t.Run("snapshot predates stop", func(t *testing.T) {
		fixture := makeTestManifestFixture(t, "vol_expected")
		data, err := os.ReadFile(fixture.CreateOptions.SnapshotEvidencePath)
		if err != nil {
			t.Fatal(err)
		}
		var evidence SnapshotEvidence
		if err := decodeStrict(data, &evidence); err != nil {
			t.Fatal(err)
		}
		evidence.CapturedAt = "2026-07-30T11:30:00Z"
		forHash := evidence
		forHash.EvidenceSHA256 = ""
		evidence.EvidenceSHA256, err = hashCanonical(forHash)
		if err != nil {
			t.Fatal(err)
		}
		writeCanonicalTestDocument(t, fixture.CreateOptions.SnapshotEvidencePath, evidence)
		if _, err := createManifest(fixture.Source, fixture.CreateOptions); err == nil ||
			!strings.Contains(err.Error(), "snapshot evidence predates") {
			t.Fatalf("stale snapshot error = %v", err)
		}
	})

	t.Run("volume observation predates stop", func(t *testing.T) {
		fixture := makeTestManifestFixture(t, "vol_expected")
		data, err := os.ReadFile(fixture.CreateOptions.VolumeEvidencePath)
		if err != nil {
			t.Fatal(err)
		}
		var evidence VolumeEvidence
		if err := decodeStrict(data, &evidence); err != nil {
			t.Fatal(err)
		}
		evidence.CapturedAt = "2026-07-30T11:30:00Z"
		forHash := evidence
		forHash.EvidenceSHA256 = ""
		evidence.EvidenceSHA256, err = hashCanonical(forHash)
		if err != nil {
			t.Fatal(err)
		}
		writeCanonicalTestDocument(t, fixture.CreateOptions.VolumeEvidencePath, evidence)
		if _, err := createManifest(fixture.Source, fixture.CreateOptions); err == nil ||
			!strings.Contains(err.Error(), "freshness evidence predates") {
			t.Fatalf("stale volume error = %v", err)
		}
	})
}

func TestCreateRechecksSourceSignerAbsence(t *testing.T) {
	fixture := makeTestManifestFixture(t, "vol_expected")
	data, err := os.ReadFile(fixture.CreateOptions.StoppedEvidencePath)
	if err != nil {
		t.Fatal(err)
	}
	var stopped StoppedEvidence
	if err := decodeStrict(data, &stopped); err != nil {
		t.Fatal(err)
	}
	stopped.ProcessID = os.Getpid()
	forHash := stopped
	forHash.EvidenceSHA256 = ""
	stopped.EvidenceSHA256, err = hashCanonical(forHash)
	if err != nil {
		t.Fatal(err)
	}
	writeCanonicalTestDocument(t, fixture.CreateOptions.StoppedEvidencePath, stopped)
	if _, err := createManifest(fixture.Source, fixture.CreateOptions); err == nil ||
		!strings.Contains(err.Error(), "still running") {
		t.Fatalf("source signer re-check error = %v", err)
	}
}

func TestCreateRechecksDestinationFreshness(t *testing.T) {
	fixture := makeTestManifestFixture(t, "vol_expected")
	baseReader := readFilesystemIdentity
	destinationReads := 0
	readFilesystemIdentity = func(path string) (FilesystemIdentity, error) {
		identity, err := baseReader(path)
		if err != nil {
			return FilesystemIdentity{}, err
		}
		if filepath.Clean(path) == fixture.Destination {
			destinationReads++
			if destinationReads == 2 {
				if err := os.WriteFile(
					filepath.Join(fixture.Destination, "late-file"),
					[]byte("late"),
					0o600,
				); err != nil {
					return FilesystemIdentity{}, err
				}
			}
		}
		return identity, nil
	}
	t.Cleanup(func() {
		readFilesystemIdentity = baseReader
	})
	if _, err := createManifest(fixture.Source, fixture.CreateOptions); err == nil ||
		!strings.Contains(err.Error(), "destination freshness re-check") {
		t.Fatalf("destination freshness re-check error = %v", err)
	}
}

func TestVerifyRechecksSourceSignerAbsence(t *testing.T) {
	fixture := makeTestManifestFixture(t, "vol_expected")
	manifest, err := createManifest(fixture.Source, fixture.CreateOptions)
	if err != nil {
		t.Fatal(err)
	}
	document, err := canonicalDocument(manifest)
	if err != nil {
		t.Fatal(err)
	}
	copyTestHome(t, fixture.Source, fixture.Destination)

	baseCheck := checkProcessAbsent
	checks := 0
	checkProcessAbsent = func(processID int) error {
		checks++
		if checks == 2 {
			return fmt.Errorf("process %d is still running", processID)
		}
		return nil
	}
	t.Cleanup(func() {
		checkProcessAbsent = baseCheck
	})
	if _, err := verifyManifest(
		document,
		fixture.Destination,
		fixture.VerifyOptions,
		true,
	); err == nil || !strings.Contains(err.Error(), "restarted during verification") {
		t.Fatalf("verification signer re-check error = %v", err)
	}
}

func TestOutputInstallIsNoReplaceAndRejectsSymlinkParent(t *testing.T) {
	directory, err := secureAbsoluteDirectory(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	existing := filepath.Join(directory, "existing.json")
	if err := os.WriteFile(existing, []byte("original"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := writeAtomic(existing, []byte("replacement"), &bytes.Buffer{}); err == nil ||
		!strings.Contains(err.Error(), "already exists") {
		t.Fatalf("replace refusal error = %v", err)
	}
	content, err := os.ReadFile(existing)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "original" {
		t.Fatalf("existing output was modified")
	}

	realParent := filepath.Join(directory, "real-parent")
	if err := os.Mkdir(realParent, 0o700); err != nil {
		t.Fatal(err)
	}
	symlinkParent := filepath.Join(directory, "linked-parent")
	if err := os.Symlink(realParent, symlinkParent); err != nil {
		t.Fatal(err)
	}
	if err := writeAtomic(
		filepath.Join(symlinkParent, "output.json"),
		[]byte("{}\n"),
		&bytes.Buffer{},
	); err == nil || !strings.Contains(err.Error(), "real directory") {
		t.Fatalf("symlink-parent refusal error = %v", err)
	}
}

func TestVerifyRejectsManifestSelfHashTampering(t *testing.T) {
	fixture := makeTestManifestFixture(t, "vol_expected")
	manifest, err := createManifest(fixture.Source, fixture.CreateOptions)
	if err != nil {
		t.Fatal(err)
	}
	manifest.LastHeight++
	document, err := canonicalDocument(manifest)
	if err != nil {
		t.Fatal(err)
	}
	copyTestHome(t, fixture.Source, fixture.Destination)
	if _, err := verifyManifest(
		document,
		fixture.Destination,
		fixture.VerifyOptions,
		true,
	); err == nil ||
		!strings.Contains(err.Error(), "self-hash mismatch") {
		t.Fatalf("self-hash tamper error = %v", err)
	}
}

func TestManifestRejectsRehashedStoppedEvidenceDrift(t *testing.T) {
	fixture := makeTestManifestFixture(t, "vol_expected")
	manifest, err := createManifest(fixture.Source, fixture.CreateOptions)
	if err != nil {
		t.Fatal(err)
	}
	manifest.StoppedEvidence.Method = "forged method"
	forHash := manifest
	forHash.ManifestSHA256 = ""
	manifest.ManifestSHA256, err = hashCanonical(forHash)
	if err != nil {
		t.Fatal(err)
	}
	if err := validateManifestShape(manifest); err == nil ||
		!strings.Contains(err.Error(), "stopped-evidence self-hash mismatch") {
		t.Fatalf("stopped-evidence drift error = %v", err)
	}
}

func TestRunCreateAndVerify(t *testing.T) {
	fixture := makeTestManifestFixture(t, "vol_cli")
	outputDirectory, err := secureAbsoluteDirectory(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	outputPath := filepath.Join(outputDirectory, "manifest.json")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if code := run([]string{
		"create",
		"--home", fixture.Source,
		"--destination-home", fixture.Destination,
		"--stopped-evidence", fixture.CreateOptions.StoppedEvidencePath,
		"--restart-inhibit-evidence", fixture.CreateOptions.RestartInhibitEvidencePath,
		"--snapshot-evidence", fixture.CreateOptions.SnapshotEvidencePath,
		"--volume-evidence", fixture.CreateOptions.VolumeEvidencePath,
		"--destination-volume-id", "vol_cli",
		"--out", outputPath,
	}, &stdout, &stderr); code != 0 {
		t.Fatalf("create exit %d: %s", code, stderr.String())
	}
	copyTestHome(t, fixture.Source, fixture.Destination)
	stdout.Reset()
	stderr.Reset()
	if code := run([]string{
		"verify",
		"--home", fixture.Destination,
		"--manifest", outputPath,
		"--restart-inhibit-evidence", fixture.VerifyOptions.RestartInhibitEvidencePath,
		"--snapshot-evidence", fixture.VerifyOptions.SnapshotEvidencePath,
		"--volume-evidence", fixture.VerifyOptions.VolumeEvidencePath,
		"--destination-volume-id", "vol_cli",
	}, &stdout, &stderr); code != 0 {
		t.Fatalf("verify exit %d: %s", code, stderr.String())
	}
	if !strings.HasPrefix(stdout.String(), "VERIFIED_LOCAL_MANIFEST ") {
		t.Fatalf("unexpected verify output: %s", stdout.String())
	}
}

func makeTestHome(t *testing.T) (string, []string) {
	t.Helper()
	home := t.TempDir()
	for _, directory := range []string{
		"config",
		"data/application.db",
		"data/blockstore.db",
		"data/state.db",
	} {
		if err := os.MkdirAll(filepath.Join(home, directory), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	writeJSONFile(t, filepath.Join(home, "config", "genesis.json"), map[string]any{
		"genesis_time":   "2026-01-01T00:00:00Z",
		"chain_id":       "zerone-rehearsal-1",
		"initial_height": "1",
		"app_state":      map[string]any{},
	}, 0o644)

	validatorSeed := bytes.Repeat([]byte{0x11}, ed25519.SeedSize)
	validatorPrivate := ed25519.NewKeyFromSeed(validatorSeed)
	validatorPublic := validatorPrivate.Public().(ed25519.PublicKey)
	addressDigest := sha256.Sum256(validatorPublic)
	validatorPrivateText := base64.StdEncoding.EncodeToString(validatorPrivate)
	writeJSONFile(t, filepath.Join(home, "config", "priv_validator_key.json"), validatorKeyDocument{
		Address: strings.ToUpper(hex.EncodeToString(addressDigest[:20])),
		PubKey: encodedKey{
			Type:  "tendermint/PubKeyEd25519",
			Value: base64.StdEncoding.EncodeToString(validatorPublic),
		},
		PrivKey: encodedKey{
			Type:  "tendermint/PrivKeyEd25519",
			Value: validatorPrivateText,
		},
	}, 0o600)

	nodeSeed := bytes.Repeat([]byte{0x22}, ed25519.SeedSize)
	nodePrivate := ed25519.NewKeyFromSeed(nodeSeed)
	nodePrivateText := base64.StdEncoding.EncodeToString(nodePrivate)
	writeJSONFile(t, filepath.Join(home, "config", "node_key.json"), nodeKeyDocument{
		PrivKey: encodedKey{
			Type:  "tendermint/PrivKeyEd25519",
			Value: nodePrivateText,
		},
	}, 0o600)

	signature := base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{0x33}, ed25519.SignatureSize))
	signBytes := base64.StdEncoding.EncodeToString([]byte("public consensus sign bytes"))
	writeJSONFile(t, filepath.Join(home, "data", "priv_validator_state.json"), signingStateDocument{
		Height:    "41",
		Round:     0,
		Step:      3,
		Signature: &signature,
		SignBytes: &signBytes,
	}, 0o600)
	for path, content := range map[string]string{
		"data/application.db/000001.log": "application database bytes",
		"data/blockstore.db/000001.log":  "blockstore database bytes",
		"data/state.db/000001.log":       "comet state database bytes",
		"data/cs.wal":                    "consensus wal bytes",
		"config/config.toml":             "proxy_app = \"tcp://127.0.0.1:26658\"\n",
	} {
		if err := os.WriteFile(filepath.Join(home, path), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return home, []string{validatorPrivateText, nodePrivateText, signature, signBytes}
}

type testManifestFixture struct {
	Source          string
	Destination     string
	PrivateMaterial []string
	CreateOptions   CreateManifestOptions
	VerifyOptions   VerifyManifestOptions
}

func makeTestManifestFixture(t *testing.T, volumeID string) testManifestFixture {
	t.Helper()
	source, privateMaterial := makeTestHome(t)
	secureSource, err := secureAbsoluteDirectory(source)
	if err != nil {
		t.Fatal(err)
	}
	destination, err := secureAbsoluteDirectory(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	oldIdentityReader := readFilesystemIdentity
	readFilesystemIdentity = func(path string) (FilesystemIdentity, error) {
		identity, err := filesystemIdentity(path)
		if err != nil {
			return FilesystemIdentity{}, err
		}
		switch filepath.Clean(path) {
		case secureSource:
			identity.DeviceID = 101
			identity.ReadOnly = true
		case destination:
			identity.DeviceID = 202
			identity.ReadOnly = false
		}
		return identity, nil
	}
	t.Cleanup(func() {
		readFilesystemIdentity = oldIdentityReader
	})

	evidenceDirectory, err := secureAbsoluteDirectory(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	processStartTime := time.Date(2026, 7, 30, 11, 0, 0, 0, time.UTC).Format(time.RFC3339Nano)
	processIdentity := strings.Repeat("e", 64)
	restart := RestartInhibitEvidence{
		Schema:                restartEvidenceSchema,
		CapturedAt:            "2026-07-30T11:59:00Z",
		SourceHome:            secureSource,
		ProcessStartTime:      processStartTime,
		ProcessIdentitySHA256: processIdentity,
		Supervisor:            "launchd-test",
		Unit:                  "zeroned-test",
		RestartDisabled:       true,
		StartBlocked:          true,
	}
	restartForHash := restart
	restartForHash.EvidenceSHA256 = ""
	restart.EvidenceSHA256, err = hashCanonical(restartForHash)
	if err != nil {
		t.Fatal(err)
	}
	restartPath := filepath.Join(evidenceDirectory, "restart-inhibit.json")
	writeCanonicalTestDocument(t, restartPath, restart)

	snapshot := SnapshotEvidence{
		Schema:         snapshotEvidenceSchema,
		CapturedAt:     "2026-07-30T12:01:00Z",
		SourceHome:     secureSource,
		SnapshotID:     "snapshot-test-001",
		Method:         "read-only isolated unit-test fixture",
		ReadOnly:       true,
		Isolated:       true,
		IncludesACLs:   true,
		IncludesXAttrs: true,
	}
	snapshotForHash := snapshot
	snapshotForHash.EvidenceSHA256 = ""
	snapshot.EvidenceSHA256, err = hashCanonical(snapshotForHash)
	if err != nil {
		t.Fatal(err)
	}
	snapshotPath := filepath.Join(evidenceDirectory, "snapshot.json")
	writeCanonicalTestDocument(t, snapshotPath, snapshot)

	volume := VolumeEvidence{
		Schema:              volumeEvidenceSchema,
		CapturedAt:          "2026-07-30T12:02:00Z",
		DestinationVolumeID: volumeID,
		Provider:            "unit-test-provider",
		ImmutableID:         true,
		EncryptionAtRest:    true,
		Fresh:               true,
		EmptyBeforeRestore:  true,
	}
	volumeForHash := volume
	volumeForHash.EvidenceSHA256 = ""
	volume.EvidenceSHA256, err = hashCanonical(volumeForHash)
	if err != nil {
		t.Fatal(err)
	}
	volumePath := filepath.Join(evidenceDirectory, "volume.json")
	writeCanonicalTestDocument(t, volumePath, volume)

	stopped := StoppedEvidence{
		Schema:                       stoppedEvidenceSchema,
		CapturedAt:                   "2026-07-30T12:00:00Z",
		Method:                       "unit-test stopped process",
		Observer:                     "observer-test",
		SourceHome:                   secureSource,
		ProcessID:                    999999,
		ProcessStartTime:             processStartTime,
		ProcessIdentitySHA256:        processIdentity,
		ProcessAbsent:                true,
		RestartInhibitEvidenceSHA256: restart.EvidenceSHA256,
		LastHeight:                   41,
		AppHash:                      testAppHash,
	}
	stoppedForHash := stopped
	stoppedForHash.EvidenceSHA256 = ""
	stopped.EvidenceSHA256, err = hashCanonical(stoppedForHash)
	if err != nil {
		t.Fatal(err)
	}
	stoppedPath := filepath.Join(evidenceDirectory, "stopped.json")
	writeCanonicalTestDocument(t, stoppedPath, stopped)

	return testManifestFixture{
		Source:          secureSource,
		Destination:     destination,
		PrivateMaterial: privateMaterial,
		CreateOptions: CreateManifestOptions{
			DestinationHome:            destination,
			DestinationVolumeID:        volumeID,
			StoppedEvidencePath:        stoppedPath,
			RestartInhibitEvidencePath: restartPath,
			SnapshotEvidencePath:       snapshotPath,
			VolumeEvidencePath:         volumePath,
		},
		VerifyOptions: VerifyManifestOptions{
			DestinationVolumeID:        volumeID,
			RestartInhibitEvidencePath: restartPath,
			SnapshotEvidencePath:       snapshotPath,
			VolumeEvidencePath:         volumePath,
		},
	}
}

func copyTestHome(t *testing.T, source, destination string) {
	t.Helper()
	err := filepath.Walk(source, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		if relative == "." {
			return nil
		}
		target := filepath.Join(destination, relative)
		if info.IsDir() {
			return os.Mkdir(target, info.Mode().Perm())
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, info.Mode().Perm())
	})
	if err != nil {
		t.Fatalf("copy validator home: %v", err)
	}
}

func writeCanonicalTestDocument(t *testing.T, path string, value any) {
	t.Helper()
	document, err := canonicalDocument(value)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, document, 0o600); err != nil {
		t.Fatal(err)
	}
}

func writeJSONFile(t *testing.T, path string, value any, mode os.FileMode) {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, mode); err != nil {
		t.Fatal(err)
	}
}
