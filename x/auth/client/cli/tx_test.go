package cli

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/cosmos/cosmos-sdk/client"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/x/auth/types"
)

const cliTestAddress = "zrn1m037n75vk2jhdr56y2ptzjjj02uljwnqwwzr7z"

func init() {
	sdk.GetConfig().SetBech32PrefixForAccount("zrn", "zrnpub")
}

func privateTempDir(t *testing.T) string {
	t.Helper()
	directory := t.TempDir()
	if err := os.Chmod(directory, 0o700); err != nil {
		t.Fatal(err)
	}
	return directory
}

func useRepositoryDarwinACLHelper(t *testing.T) {
	t.Helper()
	if runtime.GOOS != "darwin" {
		return
	}
	_, source, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve test source path")
	}
	helperPath := filepath.Clean(filepath.Join(filepath.Dir(source), "../../../..", "build", darwinACLHelperName))
	if _, err := os.Stat(helperPath); err != nil {
		t.Fatalf("Darwin ACL helper is not built at %s (run make darwin-acl-helper): %v", helperPath, err)
	}
	previous := darwinACLHelperPathOverride
	darwinACLHelperPathOverride = helperPath
	t.Cleanup(func() { darwinACLHelperPathOverride = previous })
}

func TestOnboardRegistrationProofSigning(t *testing.T) {
	seed := bytes.Repeat([]byte{0x31}, ed25519.SeedSize)
	privateKey := ed25519.NewKeyFromSeed(seed)
	publicKey := privateKey.Public().(ed25519.PublicKey)
	did := "did:zrn:" + hex.EncodeToString(publicKey)

	signature, err := signAccountRegistrationProof(
		seed, "zerone-2", cliTestAddress, did, "agent", "",
	)
	if err != nil {
		t.Fatal(err)
	}
	proofBytes, err := types.AccountRegistrationProofSignBytes(
		"zerone-2", cliTestAddress, did, publicKey, "agent", "",
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := types.VerifyEd25519Signature(publicKey, proofBytes, signature); err != nil {
		t.Fatalf("onboard helper produced an invalid proof: %v", err)
	}
	if _, err := signAccountRegistrationProof(seed[:31], "zerone-2", cliTestAddress, did, "agent", ""); err == nil {
		t.Fatal("short identity seed was accepted")
	}
}

func TestOnboardIdentityFileCustodyBoundary(t *testing.T) {
	useRepositoryDarwinACLHelper(t)
	identityFor := func(address string) onboardIdentity {
		seed := bytes.Repeat([]byte{0x41}, ed25519.SeedSize)
		publicKey := ed25519.NewKeyFromSeed(seed).Public().(ed25519.PublicKey)
		publicKeyHex := hex.EncodeToString(publicKey)
		return onboardIdentity{
			Address:       address,
			Did:           "did:zrn:" + publicKeyHex,
			PublicKeyHex:  publicKeyHex,
			PrivateKeyHex: hex.EncodeToString(seed),
			Note:          "test identity",
		}
	}

	t.Run("durable owner-only file round trips", func(t *testing.T) {
		path := filepath.Join(privateTempDir(t), "identity.json")
		want := identityFor(cliTestAddress)
		if err := persistOnboardIdentity(path, want); err != nil {
			t.Fatal(err)
		}
		info, err := os.Lstat(path)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != 0o600 {
			t.Fatalf("identity mode = %04o, want 0600", info.Mode().Perm())
		}
		got, err := readOnboardIdentity(path, cliTestAddress)
		if err != nil {
			t.Fatal(err)
		}
		if got != want {
			t.Fatalf("identity round trip mismatch:\nwant %#v\n got %#v", want, got)
		}
	})

	t.Run("rejects group or world access", func(t *testing.T) {
		path := filepath.Join(privateTempDir(t), "identity.json")
		if err := persistOnboardIdentity(path, identityFor(cliTestAddress)); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(path, 0o640); err != nil {
			t.Fatal(err)
		}
		if _, err := readOnboardIdentity(path, cliTestAddress); err == nil || !strings.Contains(err.Error(), "group or world") {
			t.Fatalf("permissive identity file was not rejected: %v", err)
		}
	})

	t.Run("rejects symlink", func(t *testing.T) {
		directory := privateTempDir(t)
		target := filepath.Join(directory, "target.json")
		if err := persistOnboardIdentity(target, identityFor(cliTestAddress)); err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(directory, "identity.json")
		if err := os.Symlink(target, path); err != nil {
			t.Fatal(err)
		}
		if _, err := readOnboardIdentity(path, cliTestAddress); err == nil || !strings.Contains(err.Error(), "non-symlink") {
			t.Fatalf("identity symlink was not rejected: %v", err)
		}
	})

	t.Run("requires exact stored address", func(t *testing.T) {
		for name, storedAddress := range map[string]string{
			"missing":   "",
			"different": "zrn1ur4eyeuuhrkfpcyhykfjsasftv9hn33smszt58",
		} {
			t.Run(name, func(t *testing.T) {
				path := filepath.Join(privateTempDir(t), "identity.json")
				if err := persistOnboardIdentity(path, identityFor(storedAddress)); err != nil {
					t.Fatal(err)
				}
				if _, err := readOnboardIdentity(path, cliTestAddress); err == nil || !strings.Contains(err.Error(), "identity belongs") {
					t.Fatalf("wrong identity address was not rejected: %v", err)
				}
			})
		}
	})

	t.Run("rejects non-regular and oversized inputs", func(t *testing.T) {
		directory := privateTempDir(t)
		if _, err := readOnboardIdentity(directory, cliTestAddress); err == nil || !strings.Contains(err.Error(), "regular") {
			t.Fatalf("identity directory was not rejected: %v", err)
		}
		path := filepath.Join(directory, "oversized.json")
		if err := os.WriteFile(path, bytes.Repeat([]byte{'x'}, maxOnboardIdentityBytes+1), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := readOnboardIdentity(path, cliTestAddress); err == nil || !strings.Contains(err.Error(), "byte limit") {
			t.Fatalf("oversized identity file was not rejected: %v", err)
		}
	})

	t.Run("detects an orphaned final directory before proof use", func(t *testing.T) {
		root := privateTempDir(t)
		custody := filepath.Join(root, "custody")
		if err := os.Mkdir(custody, 0o700); err != nil {
			t.Fatal(err)
		}
		location, err := secureOnboardIdentityLocation(filepath.Join(custody, "identity.json"), false)
		if err != nil {
			t.Fatal(err)
		}
		defer location.close()
		if err := os.Rename(custody, filepath.Join(root, "orphaned-custody")); err != nil {
			t.Fatal(err)
		}
		if err := os.Mkdir(custody, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := location.assertReachable(); err == nil || !strings.Contains(err.Error(), "changed or was orphaned") {
			t.Fatalf("orphaned custody directory was not detected: %v", err)
		}
	})
}

func TestRegistrationSignBytesCommand(t *testing.T) {
	privateKey := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{0x32}, ed25519.SeedSize))
	publicKey := privateKey.Public().(ed25519.PublicKey)
	publicKeyHex := hex.EncodeToString(publicKey)
	did := "did:zrn:" + publicKeyHex
	want, err := types.AccountRegistrationProofSignBytes(
		"zerone-2", cliTestAddress, did, publicKey, "agent", `{"v":1}`,
	)
	if err != nil {
		t.Fatal(err)
	}

	cmd := CmdRegistrationSignBytes()
	cmd.SetArgs([]string{did, publicKeyHex, "agent", "--metadata", `{"v":1}`})
	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetContext(context.Background())
	clientCtx := client.Context{}.
		WithChainID("zerone-2").
		WithFrom(cliTestAddress).
		WithFromAddress(sdk.MustAccAddressFromBech32(cliTestAddress))
	if err := client.SetCmdClientContext(cmd, clientCtx); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(output.String()) != hex.EncodeToString(want) {
		t.Fatalf("registration-sign-bytes output mismatch:\nwant %x\n got %s", want, output.String())
	}
}

func TestVerifyRegistrationProofCommand(t *testing.T) {
	privateKey := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{0x33}, ed25519.SeedSize))
	publicKey := privateKey.Public().(ed25519.PublicKey)
	publicKeyHex := hex.EncodeToString(publicKey)
	did := "did:zrn:" + publicKeyHex
	metadata := `{"v":1}`
	proofBytes, err := types.AccountRegistrationProofSignBytes(
		"zerone-2", cliTestAddress, did, publicKey, "agent", metadata,
	)
	if err != nil {
		t.Fatal(err)
	}
	signatureHex := hex.EncodeToString(ed25519.Sign(privateKey, proofBytes))

	commandArgs := []string{
		cliTestAddress, did, publicKeyHex, "agent", signatureHex,
		"--chain-id", "zerone-2", "--metadata", metadata,
	}
	cmd := CmdVerifyRegistrationProof()
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	cmd.SetArgs(commandArgs)
	var output bytes.Buffer
	cmd.SetOut(&output)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("valid proof rejected: %v", err)
	}
	if strings.TrimSpace(output.String()) != "registration proof valid" {
		t.Fatalf("unexpected verification output: %q", output.String())
	}

	tests := map[string]func([]string){
		"different metadata": func(args []string) { args[len(args)-1] = `{"v":2}` },
		"different chain ID": func(args []string) { args[6] = "zerone-other-1" },
		"different sender":   func(args []string) { args[0] = "zrn1ur4eyeuuhrkfpcyhykfjsasftv9hn33smszt58" },
		"corrupt signature": func(args []string) {
			signature := []byte(args[4])
			if signature[0] == '0' {
				signature[0] = '1'
			} else {
				signature[0] = '0'
			}
			args[4] = string(signature)
		},
		"short signature": func(args []string) { args[4] = "00" },
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			args := append([]string(nil), commandArgs...)
			mutate(args)
			cmd := CmdVerifyRegistrationProof()
			cmd.SilenceErrors = true
			cmd.SilenceUsage = true
			cmd.SetArgs(args)
			if err := cmd.Execute(); err == nil {
				t.Fatal("invalid proof was accepted")
			}
		})
	}
}
