package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

var testNow = time.Date(2026, 9, 9, 22, 20, 0, 0, time.UTC)

func fixture(t *testing.T) []byte {
	t.Helper()
	raw, err := os.ReadFile("testdata/checkpoint.json")
	if err != nil {
		t.Fatal(err)
	}
	return raw
}
func mutate(t *testing.T, fn func(map[string]any)) []byte {
	t.Helper()
	var v map[string]any
	if err := json.Unmarshal(fixture(t), &v); err != nil {
		t.Fatal(err)
	}
	fn(v)
	raw, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}
func header(v map[string]any) map[string]any {
	return v["signed_header"].(map[string]any)["header"].(map[string]any)
}
func commit(v map[string]any) map[string]any {
	return v["signed_header"].(map[string]any)["commit"].(map[string]any)
}
func validator(v map[string]any) map[string]any {
	return v["validator_set"].(map[string]any)["validators"].([]any)[0].(map[string]any)
}

func TestCapturedCheckpoint(t *testing.T) {
	out, err := verify(fixture(t), testNow, false)
	if err != nil {
		t.Fatal(err)
	}
	if out.Result != "PASS" || out.Height != 1262000 || out.IndependentTrustAnchor || out.Expired || out.AllowExpired {
		t.Fatalf("invalid receipt: %+v", out)
	}
}

func TestRejectsTamperingAndIncompleteInputs(t *testing.T) {
	cases := map[string]func(map[string]any){
		"schema":              func(v map[string]any) { v["schema"] = "other" },
		"chain":               func(v map[string]any) { v["chain_id"] = "other" },
		"height":              func(v map[string]any) { v["height"] = 1262001 },
		"fractional-height":   func(v map[string]any) { v["height"] = 1262000.5 },
		"hash":                func(v map[string]any) { v["block_hash"] = strings.Repeat("A", 64) },
		"lowercase-hash":      func(v map[string]any) { v["block_hash"] = strings.ToLower(v["block_hash"].(string)) },
		"header-time":         func(v map[string]any) { v["header_time"] = "2026-09-09T19:24:14Z" },
		"longer-trust":        func(v map[string]any) { v["trust_period_seconds"] = 1814400 },
		"unknown-top":         func(v map[string]any) { v["approved"] = true },
		"alias-top":           func(v map[string]any) { v["Schema"] = v["schema"]; delete(v, "schema") },
		"unknown-header":      func(v map[string]any) { header(v)["other"] = true },
		"wrong-header-chain":  func(v map[string]any) { header(v)["chain_id"] = "other" },
		"tampered-app-hash":   func(v map[string]any) { header(v)["app_hash"] = strings.Repeat("A", 64) },
		"wrong-commit-height": func(v map[string]any) { commit(v)["height"] = "1262001" },
		"tampered-signature": func(v map[string]any) {
			s := commit(v)["signatures"].([]any)[0].(map[string]any)
			sig, err := base64.StdEncoding.DecodeString(s["signature"].(string))
			if err != nil {
				t.Fatal(err)
			}
			sig[0] ^= 1
			s["signature"] = base64.StdEncoding.EncodeToString(sig)
		},
		"absent-signature":       func(v map[string]any) { commit(v)["signatures"] = []any{} },
		"partial-validators":     func(v map[string]any) { v["validator_set"].(map[string]any)["total"] = "2" },
		"wrong-validator-height": func(v map[string]any) { v["validator_set"].(map[string]any)["block_height"] = "1262001" },
		"wrong-voting-power":     func(v map[string]any) { validator(v)["voting_power"] = "11112" },
		"wrong-public-key": func(v map[string]any) {
			validator(v)["pub_key"].(map[string]any)["value"] = base64.StdEncoding.EncodeToString(make([]byte, 32))
		},
		"nil-validator": func(v map[string]any) { v["validator_set"].(map[string]any)["validators"] = []any{nil} },
		"nil-header":    func(v map[string]any) { v["signed_header"].(map[string]any)["header"] = nil },
	}
	for name, fn := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := verify(mutate(t, fn), testNow, false); err == nil {
				t.Fatal("accepted invalid checkpoint")
			}
		})
	}
}

func TestJSONBoundsAndDuplicateKeys(t *testing.T) {
	raw := fixture(t)
	for name, bad := range map[string][]byte{
		"duplicate-top":    bytes.Replace(raw, []byte(`"chain_id": "zerone-1",`), []byte(`"chain_id": "zerone-1", "chain_id": "zerone-1",`), 1),
		"duplicate-nested": bytes.Replace(raw, []byte(`"voting_power": "11111"`), []byte(`"voting_power": "11111", "voting_power": "11111"`), 1),
		"trailing":         append(append([]byte{}, raw...), []byte(`{}`)...),
		"too-large":        bytes.Repeat([]byte(" "), maxInputBytes+1),
		"too-deep":         []byte(strings.Repeat("[", 34) + "0" + strings.Repeat("]", 34)),
		"invalid-utf8":     []byte{'"', 0xff, '"'},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := verify(bad, testNow, false); err == nil {
				t.Fatal("accepted invalid JSON")
			}
		})
	}
}

func TestTrustWindowAndResume(t *testing.T) {
	raw := fixture(t)
	expiry := time.Date(2026, 9, 16, 19, 24, 13, 254302055, time.UTC)
	if _, err := verify(raw, expiry.Add(-time.Nanosecond), false); err != nil {
		t.Fatal(err)
	}
	if _, err := verify(raw, expiry, false); err == nil {
		t.Fatal("accepted expired bootstrap")
	}
	out, err := verify(raw, expiry, true)
	if err != nil || !out.Expired || !out.AllowExpired {
		t.Fatalf("resume verification: %+v %v", out, err)
	}
	bad := mutate(t, func(v map[string]any) { v["block_hash"] = strings.Repeat("A", 64) })
	if _, err := verify(bad, expiry, true); err == nil {
		t.Fatal("expiry override bypassed proof")
	}
	if _, err := verify(raw, time.Date(2026, 9, 9, 19, 0, 0, 0, time.UTC), true); err == nil {
		t.Fatal("expiry override accepted future header")
	}
}

func TestRejectsUnsignedCommitAddressSubstitution(t *testing.T) {
	// ValidatorAddress is excluded from canonical vote sign bytes. The original
	// signature therefore remains cryptographically valid when this field is
	// substituted; address/index binding must independently reject the packet.
	raw := mutate(t, func(v map[string]any) {
		commit(v)["signatures"].([]any)[0].(map[string]any)["validator_address"] = strings.Repeat("A", 40)
	})
	for _, allowExpired := range []bool{false, true} {
		if _, err := verify(raw, testNow, allowExpired); err == nil || !strings.Contains(err.Error(), "commit signature") {
			t.Fatalf("address substitution was not specifically refused: %v", err)
		}
	}
}

func TestCLIRefusesMissingPathAndExtraArguments(t *testing.T) {
	for _, args := range [][]string{nil, {"--checkpoint", "testdata/checkpoint.json", "extra"}} {
		if err := run(args, &bytes.Buffer{}); err == nil {
			t.Fatal("invalid CLI accepted")
		}
	}
}

func TestCLIRejectsSymlinkAndFIFO(t *testing.T) {
	dir := t.TempDir()
	real := filepath.Join(dir, "checkpoint.json")
	if err := os.WriteFile(real, fixture(t), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "link.json")
	if err := os.Symlink(real, link); err != nil {
		t.Fatal(err)
	}
	fifo := filepath.Join(dir, "fifo")
	if err := syscall.Mkfifo(fifo, 0o600); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{link, fifo, dir} {
		if err := run([]string{"--checkpoint", path}, &bytes.Buffer{}); err == nil {
			t.Fatalf("accepted nonregular input: %s", path)
		}
	}
}

func TestFileIdentityAndSizeDrift(t *testing.T) {
	path := filepath.Join(t.TempDir(), "checkpoint.json")
	if err := os.WriteFile(path, fixture(t), 0o600); err != nil {
		t.Fatal(err)
	}
	before, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if !sameFileState(before, before) {
		t.Fatal("unchanged file refused")
	}
	if err := os.WriteFile(path, []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	after, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if sameFileState(before, after) {
		t.Fatal("size drift accepted")
	}
	other := filepath.Join(t.TempDir(), "other.json")
	if err := os.WriteFile(other, fixture(t), 0o600); err != nil {
		t.Fatal(err)
	}
	replacement, err := os.Stat(other)
	if err != nil {
		t.Fatal(err)
	}
	if sameFileState(before, replacement) {
		t.Fatal("file replacement accepted")
	}
}
