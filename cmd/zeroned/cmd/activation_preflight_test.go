package cmd

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	upgradetypes "cosmossdk.io/x/upgrade/types"
)

func TestActivationPreflightCloneCannotMutateSourceTree(t *testing.T) {
	source := filepath.Join(t.TempDir(), "data")
	if err := os.Mkdir(source, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(source, "application.db"), 0o700); err != nil {
		t.Fatal(err)
	}
	sourceFile := filepath.Join(source, "application.db", "000001.ldb")
	if err := os.WriteFile(sourceFile, []byte("committed-source"), 0o640); err != nil {
		t.Fatal(err)
	}
	before, err := buildActivationSourceManifest(source)
	if err != nil {
		t.Fatal(err)
	}

	copyRoot := filepath.Join(t.TempDir(), "data")
	if err := cloneActivationSourceTree(source, copyRoot); err != nil {
		t.Fatal(err)
	}
	copied, err := buildActivationSourceManifest(copyRoot)
	if err != nil {
		t.Fatal(err)
	}
	if copied != before {
		t.Fatalf("copy manifest differs: source=%+v copy=%+v", before, copied)
	}
	if err := os.WriteFile(
		filepath.Join(copyRoot, "application.db", "000001.ldb"),
		[]byte("load-version-rebuild-write"),
		0o640,
	); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(copyRoot, "application.db", "LOCK"),
		[]byte("clone-only"),
		0o600,
	); err != nil {
		t.Fatal(err)
	}
	after, err := buildActivationSourceManifest(source)
	if err != nil {
		t.Fatal(err)
	}
	if after != before {
		t.Fatalf("verification-copy writes mutated source: before=%+v after=%+v", before, after)
	}
	sourceBytes, err := os.ReadFile(sourceFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(sourceBytes) != "committed-source" {
		t.Fatalf("source bytes changed: %q", sourceBytes)
	}
}

func TestActivationPreflightManifestRejectsSymlinks(t *testing.T) {
	source := t.TempDir()
	if err := os.Symlink("target", filepath.Join(source, "link")); err != nil {
		t.Fatal(err)
	}
	if _, err := buildActivationSourceManifest(source); err == nil {
		t.Fatal("source manifest must reject symlinks")
	}
	if err := cloneActivationSourceTree(
		source,
		filepath.Join(t.TempDir(), "copy"),
	); err == nil {
		t.Fatal("source clone must reject symlinks")
	}
}

func TestActivationPreflightBindsGenesisAndUpgradeInfo(t *testing.T) {
	root := t.TempDir()
	genesisPath := filepath.Join(root, "genesis.json")
	genesis := []byte(`{"chain_id":"zerone-test-1","app_state":{}}`)
	if err := os.WriteFile(genesisPath, genesis, 0o600); err != nil {
		t.Fatal(err)
	}
	chainID, genesisDigest, err := readActivationGenesisIdentity(genesisPath)
	if err != nil {
		t.Fatal(err)
	}
	if chainID != "zerone-test-1" {
		t.Fatalf("unexpected chain id %q", chainID)
	}
	expectedGenesisDigest := sha256.Sum256(genesis)
	if genesisDigest != fmt.Sprintf("%x", expectedGenesisDigest) {
		t.Fatalf("unexpected genesis digest %s", genesisDigest)
	}

	plan := upgradetypes.Plan{
		Name:   "sdk-0.53-ibc-10",
		Height: 22,
		Info:   `{"schema":"fixture"}`,
	}
	raw, err := json.Marshal(plan)
	if err != nil {
		t.Fatal(err)
	}
	upgradeInfoPath := filepath.Join(root, upgradetypes.UpgradeInfoFilename)
	if err := os.WriteFile(upgradeInfoPath, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	decoded, digest, err := readActivationUpgradeInfo(upgradeInfoPath)
	if err != nil {
		t.Fatal(err)
	}
	if decoded.Name != plan.Name ||
		decoded.Height != plan.Height ||
		decoded.Info != plan.Info {
		t.Fatalf("decoded plan differs: got %+v want %+v", decoded, plan)
	}
	expectedUpgradeDigest := sha256.Sum256(raw)
	if digest != fmt.Sprintf("%x", expectedUpgradeDigest) {
		t.Fatalf("unexpected upgrade-info digest %s", digest)
	}
}

func TestActivationPreflightIdentityFilesRejectSymlinks(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "target")
	if err := os.WriteFile(target, []byte(`{"chain_id":"chain"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "link")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	if _, _, err := readActivationGenesisIdentity(link); err == nil {
		t.Fatal("genesis symlink must be rejected")
	}
	if _, _, err := readActivationUpgradeInfo(link); err == nil {
		t.Fatal("upgrade-info symlink must be rejected")
	}
}
