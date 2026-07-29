package main

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/cosmos/cosmos-sdk/types/bech32"
)

func TestAuditCleanSnapshot(t *testing.T) {
	identity := bytes.Repeat([]byte{0x11}, 32)
	txKey := append([]byte{0x02}, bytes.Repeat([]byte{0x22}, 32)...)
	address := addressForKey(t, "/cosmos.crypto.secp256k1.PubKey", txKey)
	document := makeDocument(t, []map[string]any{
		zeroneRecord(address, identity, "did:zrn:"+hex.EncodeToString(identity)[:32], 1),
	}, []map[string]any{
		mappingRecord(address, identity, "did:zrn:"+hex.EncodeToString(identity)[:32]),
	}, []any{
		baseAccount(address, "/cosmos.crypto.secp256k1.PubKey", txKey),
	})

	report, err := auditDocument(document, "clean.json")
	if err != nil {
		t.Fatal(err)
	}
	if report.Summary.Errors != 0 || report.Summary.Warnings != 0 {
		t.Fatalf("expected clean report, got %+v", report.Findings)
	}
	if !report.Coverage.CompleteSnapshot {
		t.Fatal("expected full genesis snapshot coverage")
	}
}

func TestAuditDetectsCaseAndLengthAliases(t *testing.T) {
	identity := bytes.Repeat([]byte{0xab}, 32)
	txKeyA := append([]byte{0x02}, bytes.Repeat([]byte{0x31}, 32)...)
	txKeyB := append([]byte{0x03}, bytes.Repeat([]byte{0x32}, 32)...)
	addressA := addressForKey(t, "/cosmos.crypto.secp256k1.PubKey", txKeyA)
	addressB := addressForKey(t, "/cosmos.crypto.secp256k1.PubKey", txKeyB)
	canonical := "did:zrn:" + hex.EncodeToString(identity)[:32]
	alias := "did:zrn:" + stringsToUpper(hex.EncodeToString(identity))

	document := makeDocument(t, []map[string]any{
		zeroneRecord(addressA, identity, canonical, 1),
		zeroneRecord(addressB, identity, alias, 1),
	}, []map[string]any{
		mappingRecord(addressA, identity, canonical),
		mappingRecord(addressB, identity, alias),
	}, []any{
		baseAccount(addressA, "/cosmos.crypto.secp256k1.PubKey", txKeyA),
		baseAccount(addressB, "/cosmos.crypto.secp256k1.PubKey", txKeyB),
	})

	report, err := auditDocument(document, "aliases.json")
	if err != nil {
		t.Fatal(err)
	}
	assertFinding(t, report, "DID_NON_CANONICAL_CASE")
	assertFinding(t, report, "DID_NON_CANONICAL_LENGTH")
	assertFinding(t, report, "DID_NORMALIZATION_ALIAS")
	assertFinding(t, report, "DID_NORMALIZED_DUPLICATE")
}

func TestAuditDetectsBaseAccountPubKeyAddressMismatchInNestedVestingAccount(t *testing.T) {
	identity := bytes.Repeat([]byte{0x41}, 32)
	storedKey := bytes.Repeat([]byte{0x42}, 32)
	otherKey := bytes.Repeat([]byte{0x43}, 32)
	address := addressForKey(t, "/cosmos.crypto.ed25519.PubKey", storedKey)
	did := "did:zrn:" + hex.EncodeToString(identity)[:32]
	nested := map[string]any{
		"@type": "/cosmos.vesting.v1beta1.PermanentLockedAccount",
		"base_vesting_account": map[string]any{
			"base_account": baseAccount(address, "/cosmos.crypto.ed25519.PubKey", otherKey),
		},
	}
	document := makeDocument(t, []map[string]any{
		zeroneRecord(address, identity, did, 1),
	}, []map[string]any{
		mappingRecord(address, identity, did),
	}, []any{nested})

	report, err := auditDocument(document, "mismatch.json")
	if err != nil {
		t.Fatal(err)
	}
	assertFinding(t, report, "BASEACCOUNT_PUBKEY_ADDRESS_MISMATCH")
}

func TestAuditDetectsKeyAnomaliesAndMissingMapping(t *testing.T) {
	identity := bytes.Repeat([]byte{0x51}, 32)
	txKey := append([]byte{0x02}, bytes.Repeat([]byte{0x52}, 32)...)
	address := addressForKey(t, "/cosmos.crypto.secp256k1.PubKey", txKey)
	did := "did:zrn:" + hex.EncodeToString(identity)[:32]
	account := zeroneRecord(address, identity, did, 1)
	account["public_key"] = "1234"
	account["operational_public_key"] = fmt.Sprintf("%064s", "not-hex")
	document := makeDocument(t, []map[string]any{account}, nil, []any{
		baseAccount(address, "/cosmos.crypto.secp256k1.PubKey", txKey),
	})

	report, err := auditDocument(document, "keys.json")
	if err != nil {
		t.Fatal(err)
	}
	assertFinding(t, report, "IDENTITY_KEY_LENGTH_INVALID")
	assertFinding(t, report, "OPERATIONAL_KEY_HEX_INVALID")
	assertFinding(t, report, "DID_MAPPING_MISSING")
}

func TestAuditFlagsUnexportedRotationState(t *testing.T) {
	identity := bytes.Repeat([]byte{0x61}, 32)
	rotated := bytes.Repeat([]byte{0x62}, 32)
	txKey := append([]byte{0x02}, bytes.Repeat([]byte{0x63}, 32)...)
	address := addressForKey(t, "/cosmos.crypto.secp256k1.PubKey", txKey)
	did := "did:zrn:" + hex.EncodeToString(identity)[:32]
	account := zeroneRecord(address, identity, did, 2)
	account["operational_public_key"] = hex.EncodeToString(rotated)
	document := makeDocument(t, []map[string]any{account}, []map[string]any{
		mappingRecord(address, identity, did),
	}, []any{
		baseAccount(address, "/cosmos.crypto.secp256k1.PubKey", txKey),
	})

	report, err := auditDocument(document, "rotation.json")
	if err != nil {
		t.Fatal(err)
	}
	assertFinding(t, report, "ROTATION_STATE_NOT_EXPORTED")
}

func TestAuditReportsUnknownCosmosKeyTypeWithoutGuessing(t *testing.T) {
	identity := bytes.Repeat([]byte{0x68}, 32)
	address, err := bech32.ConvertAndEncode("zrn", bytes.Repeat([]byte{0x69}, 20))
	if err != nil {
		t.Fatal(err)
	}
	did := "did:zrn:" + hex.EncodeToString(identity)[:32]
	unknown := baseAccount(address, "/cosmos.crypto.multisig.LegacyAminoPubKey", nil)
	unknown["pub_key"] = map[string]any{
		"@type":       "/cosmos.crypto.multisig.LegacyAminoPubKey",
		"threshold":   2,
		"public_keys": []any{},
	}
	document := makeDocument(t, []map[string]any{
		zeroneRecord(address, identity, did, 1),
	}, []map[string]any{
		mappingRecord(address, identity, did),
	}, []any{unknown})

	report, err := auditDocument(document, "unknown-key.json")
	if err != nil {
		t.Fatal(err)
	}
	assertFinding(t, report, "COSMOS_PUBKEY_UNSUPPORTED")
}

func TestStandaloneModuleIsExplicitlyIncomplete(t *testing.T) {
	identity := bytes.Repeat([]byte{0x71}, 32)
	address, err := bech32.ConvertAndEncode("zrn", bytes.Repeat([]byte{0x72}, 20))
	if err != nil {
		t.Fatal(err)
	}
	did := "did:zrn:" + hex.EncodeToString(identity)[:32]
	module := map[string]any{
		"accounts":     []map[string]any{zeroneRecord(address, identity, did, 1)},
		"did_mappings": []map[string]any{mappingRecord(address, identity, did)},
	}
	data, err := json.Marshal(module)
	if err != nil {
		t.Fatal(err)
	}

	report, err := auditDocument(data, "module.json")
	if err != nil {
		t.Fatal(err)
	}
	if report.Coverage.CompleteSnapshot {
		t.Fatal("module-only input must not claim complete snapshot coverage")
	}
	assertFinding(t, report, "COSMOS_AUTH_STATE_MISSING")
}

func TestRunJSONAndExitPolicy(t *testing.T) {
	module := []byte(`{"accounts":[],"did_mappings":[]}`)
	var stdout, stderr bytes.Buffer
	exitCode := run(
		[]string{"--input", "-", "--format", "json", "--fail-on", "error"},
		bytes.NewReader(module),
		&stdout,
		&stderr,
	)
	if exitCode != 1 {
		t.Fatalf("expected error-policy exit 1 for incomplete module input, got %d (stderr %s)", exitCode, stderr.String())
	}
	var report Report
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("decode JSON output: %v", err)
	}
	assertFinding(t, report, "COSMOS_AUTH_STATE_MISSING")
}

func makeDocument(
	t *testing.T,
	accounts []map[string]any,
	mappings []map[string]any,
	cosmosAccounts []any,
) []byte {
	t.Helper()
	document := map[string]any{
		"chain_id": "zerone-test-1",
		"app_state": map[string]any{
			"zerone_auth": map[string]any{
				"accounts":     accounts,
				"did_mappings": mappings,
			},
			"auth": map[string]any{
				"accounts": cosmosAccounts,
			},
		},
	}
	data, err := json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func zeroneRecord(address string, identity []byte, did string, version uint32) map[string]any {
	return map[string]any{
		"address":                 address,
		"did":                     did,
		"public_key":              hex.EncodeToString(identity),
		"account_type":            "agent",
		"operational_public_key":  hex.EncodeToString(identity),
		"operational_key_version": version,
	}
}

func mappingRecord(address string, identity []byte, did string) map[string]any {
	return map[string]any{
		"did":     did,
		"bech32":  address,
		"pub_key": hex.EncodeToString(identity),
	}
}

func baseAccount(address, pubKeyType string, key []byte) map[string]any {
	return map[string]any{
		"@type":          "/cosmos.auth.v1beta1.BaseAccount",
		"address":        address,
		"pub_key":        map[string]any{"@type": pubKeyType, "key": base64.StdEncoding.EncodeToString(key)},
		"account_number": "0",
		"sequence":       "0",
	}
}

func addressForKey(t *testing.T, pubKeyType string, key []byte) string {
	t.Helper()
	payload, err := cosmosAddressBytes(pubKeyType, key)
	if err != nil {
		t.Fatal(err)
	}
	address, err := bech32.ConvertAndEncode("zrn", payload)
	if err != nil {
		t.Fatal(err)
	}
	return address
}

func assertFinding(t *testing.T, report Report, code string) {
	t.Helper()
	for _, finding := range report.Findings {
		if finding.Code == code {
			return
		}
	}
	t.Fatalf("finding %s missing from %+v", code, report.Findings)
}

func stringsToUpper(value string) string {
	result := make([]byte, len(value))
	for i := range value {
		if value[i] >= 'a' && value[i] <= 'f' {
			result[i] = value[i] - ('a' - 'A')
		} else {
			result[i] = value[i]
		}
	}
	return string(result)
}
