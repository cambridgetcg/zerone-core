//go:build darwin

package cli

import (
	"bytes"
	"crypto/ed25519"
	"encoding/hex"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const inheritedEveryoneACL = "everyone allow list,search,read,file_inherit,directory_inherit"

func darwinTestIdentity(address string) onboardIdentity {
	seed := bytes.Repeat([]byte{0x51}, ed25519.SeedSize)
	publicKey := ed25519.NewKeyFromSeed(seed).Public().(ed25519.PublicKey)
	publicKeyHex := hex.EncodeToString(publicKey)
	return onboardIdentity{
		Address:       address,
		Did:           "did:zrn:" + publicKeyHex,
		PublicKeyHex:  publicKeyHex,
		PrivateKeyHex: hex.EncodeToString(seed),
		Note:          "Darwin custody test identity",
	}
}

func requireDarwinACLTooling(t *testing.T) {
	t.Helper()
	probe := privateTempDir(t)
	command := exec.Command("/bin/chmod", "+a", inheritedEveryoneACL, probe)
	if output, err := command.CombinedOutput(); err != nil {
		t.Skipf("macOS ACL tooling unavailable: %v (%s)", err, output)
	}
}

func addDarwinACL(t *testing.T, path, entry string) {
	t.Helper()
	command := exec.Command("/bin/chmod", "+a", entry, path)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("add ACL to %s: %v (%s)", path, err, output)
	}
}

func TestDarwinOnboardIdentityRejectsLocalAndInheritedACLs(t *testing.T) {
	useRepositoryDarwinACLHelper(t)
	requireDarwinACLTooling(t)

	t.Run("mode 0600 file with local read ACL", func(t *testing.T) {
		directory := privateTempDir(t)
		path := filepath.Join(directory, "identity.json")
		identity := darwinTestIdentity(cliTestAddress)
		if err := persistOnboardIdentity(path, identity); err != nil {
			t.Fatal(err)
		}
		addDarwinACL(t, path, "everyone allow read")
		if err := os.Chmod(path, 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := readOnboardIdentity(path, cliTestAddress); err == nil || !strings.Contains(err.Error(), "extended ACLs are not allowed") {
			t.Fatalf("mode-0600 ACL-bearing identity was not rejected: %v", err)
		}
	})

	t.Run("inherited ACL blocks before identity creation", func(t *testing.T) {
		directory := privateTempDir(t)
		addDarwinACL(t, directory, inheritedEveryoneACL)
		path := filepath.Join(directory, "new", "nested", "identity.json")
		if err := persistOnboardIdentity(path, darwinTestIdentity(cliTestAddress)); err == nil || !strings.Contains(err.Error(), "extended ACLs are not allowed") {
			t.Fatalf("inherited directory ACL was not rejected: %v", err)
		}
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("identity bytes reached disk before parent ACL validation: %v", err)
		}
	})
}

func TestDarwinACLHelperInspectsInheritedDescriptorAfterPathReplacement(t *testing.T) {
	useRepositoryDarwinACLHelper(t)
	requireDarwinACLTooling(t)
	directory := privateTempDir(t)
	originalPath := filepath.Join(directory, "identity.json")
	movedPath := filepath.Join(directory, "identity-with-acl.json")
	if err := os.WriteFile(originalPath, []byte("secret"), 0o600); err != nil {
		t.Fatal(err)
	}
	addDarwinACL(t, originalPath, "everyone allow read")
	target, err := os.Open(originalPath)
	if err != nil {
		t.Fatal(err)
	}
	defer target.Close()
	if err := os.Rename(originalPath, movedPath); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(originalPath, []byte("clean replacement"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := validateNoDarwinExtendedACL(target, originalPath, "file"); err == nil || !strings.Contains(err.Error(), "extended ACLs are not allowed") {
		t.Fatalf("helper followed the replaced pathname instead of fd 0: %v", err)
	}
}

func TestDarwinACLHelperFailsClosedWhenMissingOrProtocolInvalid(t *testing.T) {
	useRepositoryDarwinACLHelper(t)
	requireDarwinACLTooling(t)
	trustedHelper := darwinACLHelperPathOverride

	t.Run("missing", func(t *testing.T) {
		directory := privateTempDir(t)
		previous := darwinACLHelperPathOverride
		darwinACLHelperPathOverride = filepath.Join(directory, "missing-helper")
		t.Cleanup(func() { darwinACLHelperPathOverride = previous })
		identityPath := filepath.Join(directory, "identity.json")
		err := persistOnboardIdentity(identityPath, darwinTestIdentity(cliTestAddress))
		if err == nil || !strings.Contains(err.Error(), "descriptor ACL inspection is unavailable or unsafe") {
			t.Fatalf("missing helper did not fail closed: %v", err)
		}
		if _, statErr := os.Stat(identityPath); !os.IsNotExist(statErr) {
			t.Fatalf("identity was created with missing helper: %v", statErr)
		}
	})

	t.Run("special mode bits", func(t *testing.T) {
		directory := privateTempDir(t)
		helperPath := filepath.Join(directory, darwinACLHelperName)
		helperBytes, err := os.ReadFile(trustedHelper)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(helperPath, helperBytes, 0o555); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(helperPath, 0o555|os.ModeSetuid); err != nil {
			t.Fatal(err)
		}
		previous := darwinACLHelperPathOverride
		darwinACLHelperPathOverride = helperPath
		t.Cleanup(func() { darwinACLHelperPathOverride = previous })
		identityPath := filepath.Join(privateTempDir(t), "identity.json")
		err = persistOnboardIdentity(identityPath, darwinTestIdentity(cliTestAddress))
		if err == nil || !strings.Contains(err.Error(), "must not have setuid, setgid, or sticky bits") {
			t.Fatalf("helper with special mode bits did not fail closed: %v", err)
		}
		if _, statErr := os.Stat(identityPath); !os.IsNotExist(statErr) {
			t.Fatalf("identity was created with unsafe helper mode: %v", statErr)
		}
	})

	t.Run("bad protocol", func(t *testing.T) {
		directory := privateTempDir(t)
		helperPath := filepath.Join(directory, darwinACLHelperName)
		sourcePath := filepath.Join(directory, "bad-helper.c")
		const source = `#include <unistd.h>
int main(void) {
  static const char bad[] = "zerone-darwin-acl-v1 maybe\n";
  return write(1, bad, sizeof(bad) - 1) < 0 ? 70 : 0;
}
`
		if err := os.WriteFile(sourcePath, []byte(source), 0o600); err != nil {
			t.Fatal(err)
		}
		compile := exec.Command(
			"/usr/bin/xcrun", "--sdk", "macosx", "clang", sourcePath,
			"-arch", "arm64", "-arch", "x86_64", "-Wl,-no_uuid", "-o", helperPath,
		)
		if output, err := compile.CombinedOutput(); err != nil {
			t.Fatalf("compile invalid-protocol helper: %v (%s)", err, output)
		}
		if err := os.Chmod(helperPath, 0o555); err != nil {
			t.Fatal(err)
		}
		previous := darwinACLHelperPathOverride
		darwinACLHelperPathOverride = helperPath
		t.Cleanup(func() { darwinACLHelperPathOverride = previous })
		identityPath := filepath.Join(directory, "identity.json")
		err := persistOnboardIdentity(identityPath, darwinTestIdentity(cliTestAddress))
		if err == nil || !strings.Contains(err.Error(), "invalid result") {
			t.Fatalf("invalid helper protocol did not fail closed: %v", err)
		}
		if _, statErr := os.Stat(identityPath); !os.IsNotExist(statErr) {
			t.Fatalf("identity was created after invalid helper protocol: %v", statErr)
		}
	})
}

func TestDarwinACLHelperBootstrapRejectsItsOwnACLs(t *testing.T) {
	useRepositoryDarwinACLHelper(t)
	requireDarwinACLTooling(t)
	trustedHelper := darwinACLHelperPathOverride
	helperBytes, err := os.ReadFile(trustedHelper)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("helper file", func(t *testing.T) {
		helperDirectory := privateTempDir(t)
		helperPath := filepath.Join(helperDirectory, darwinACLHelperName)
		if err := os.WriteFile(helperPath, helperBytes, 0o555); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(helperPath, 0o555); err != nil {
			t.Fatal(err)
		}
		addDarwinACL(t, helperPath, "everyone allow read")
		previous := darwinACLHelperPathOverride
		darwinACLHelperPathOverride = helperPath
		t.Cleanup(func() { darwinACLHelperPathOverride = previous })

		identityPath := filepath.Join(privateTempDir(t), "identity.json")
		err := persistOnboardIdentity(identityPath, darwinTestIdentity(cliTestAddress))
		if err == nil || !strings.Contains(err.Error(), "bootstrap helper has an extended ACL") {
			t.Fatalf("ACL-bearing helper was trusted: %v", err)
		}
		if _, statErr := os.Stat(identityPath); !os.IsNotExist(statErr) {
			t.Fatalf("identity was created after unsafe helper bootstrap: %v", statErr)
		}
	})

	t.Run("helper directory", func(t *testing.T) {
		helperDirectory := privateTempDir(t)
		helperPath := filepath.Join(helperDirectory, darwinACLHelperName)
		if err := os.WriteFile(helperPath, helperBytes, 0o555); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(helperPath, 0o555); err != nil {
			t.Fatal(err)
		}
		addDarwinACL(t, helperDirectory, "everyone allow list,search,read")
		previous := darwinACLHelperPathOverride
		darwinACLHelperPathOverride = helperPath
		t.Cleanup(func() { darwinACLHelperPathOverride = previous })

		identityPath := filepath.Join(privateTempDir(t), "identity.json")
		err := persistOnboardIdentity(identityPath, darwinTestIdentity(cliTestAddress))
		if err == nil || !strings.Contains(err.Error(), "bootstrap helper directory has an extended ACL") {
			t.Fatalf("ACL-bearing helper directory was trusted: %v", err)
		}
		if _, statErr := os.Stat(identityPath); !os.IsNotExist(statErr) {
			t.Fatalf("identity was created after unsafe helper-directory bootstrap: %v", statErr)
		}
	})
}

func TestDarwinOnboardCreatesPrivateComponents(t *testing.T) {
	useRepositoryDarwinACLHelper(t)
	directory := privateTempDir(t)
	identityPath := filepath.Join(directory, "one", "two", "identity.json")
	if err := persistOnboardIdentity(identityPath, darwinTestIdentity(cliTestAddress)); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{filepath.Join(directory, "one"), filepath.Join(directory, "one", "two")} {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != 0o700 {
			t.Fatalf("new identity directory %s mode = %04o, want 0700", path, info.Mode().Perm())
		}
	}
}
