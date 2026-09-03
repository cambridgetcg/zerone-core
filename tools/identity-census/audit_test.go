package main

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/cosmos/cosmos-sdk/types/bech32"
)

func TestAuditCleanSnapshot(t *testing.T) {
	identity := identityKey(0x11)
	txKey := append([]byte{0x02}, bytes.Repeat([]byte{0x22}, 32)...)
	address := addressForKey(t, "/cosmos.crypto.secp256k1.PubKey", txKey)
	document := makeDocument(t, []map[string]any{
		zeroneRecord(address, identity, canonicalDID(identity), 1),
	}, []map[string]any{
		mappingRecord(address, identity, canonicalDID(identity)),
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
	if !report.Coverage.RotationStatePresent || report.Summary.KeyRotations != 0 {
		t.Fatalf("unexpected rotation coverage: %+v", report.Coverage)
	}
	if report.Coverage.ValidationProfile != "source-canonical" {
		t.Fatalf("unexpected validation profile %q", report.Coverage.ValidationProfile)
	}
}

func TestAuditZerone1LegacyProfilePreservesAndLabelsHistoricalFields(t *testing.T) {
	identity := identityKey(0x12)
	txKey := append([]byte{0x02}, bytes.Repeat([]byte{0x23}, 32)...)
	address := addressForKey(t, "/cosmos.crypto.secp256k1.PubKey", txKey)
	legacyDID := "did:zrn:" + stringsToUpper(hex.EncodeToString(identity)[:32])
	account := zeroneRecord(address, identity, legacyDID, 1)
	delete(account, "operational_key_hash")
	document := makeDocumentForChain(
		t,
		"zerone-1",
		[]map[string]any{account},
		[]map[string]any{mappingRecord(address, identity, legacyDID)},
		[]any{baseAccount(address, "/cosmos.crypto.secp256k1.PubKey", txKey)},
		nil,
		false,
	)

	report, err := auditDocument(document, "zerone-1-export.json")
	if err != nil {
		t.Fatal(err)
	}
	if report.Summary.Errors != 0 {
		t.Fatalf("legacy-but-internally-consistent snapshot produced corruption errors: %+v", report.Findings)
	}
	assertFindingSeverity(t, report, "DID_LEGACY_SHORT_FORM", severityWarning)
	assertFindingSeverity(t, report, "DID_NON_CANONICAL_CASE", severityWarning)
	assertFindingSeverity(t, report, "OPERATIONAL_KEY_HASH_MISSING_LEGACY", severityWarning)
	assertNoFinding(t, report, "OPERATIONAL_KEY_HASH_MISSING")
	if report.Coverage.ValidationProfile != "deployed-zerone-1-legacy-audit" ||
		report.Coverage.RotationStatePresent {
		t.Fatalf("legacy boundary was not explicit: %+v", report.Coverage)
	}
}

func TestAuditDoesNotGrantLegacyExceptionsOutsideExactZerone1Document(t *testing.T) {
	identity := identityKey(0x13)
	txKey := append([]byte{0x03}, bytes.Repeat([]byte{0x24}, 32)...)
	address := addressForKey(t, "/cosmos.crypto.secp256k1.PubKey", txKey)
	legacyDID := "did:zrn:" + stringsToUpper(hex.EncodeToString(identity)[:32])
	account := zeroneRecord(address, identity, legacyDID, 1)
	delete(account, "operational_key_hash")
	document := makeDocumentForChain(
		t,
		"zerone-2",
		[]map[string]any{account},
		[]map[string]any{mappingRecord(address, identity, legacyDID)},
		[]any{baseAccount(address, "/cosmos.crypto.secp256k1.PubKey", txKey)},
		[]any{},
		true,
	)

	report, err := auditDocument(document, "zerone-2.json")
	if err != nil {
		t.Fatal(err)
	}
	assertFindingSeverity(t, report, "DID_LEGACY_SHORT_FORM", severityError)
	assertFindingSeverity(t, report, "DID_NON_CANONICAL_CASE", severityError)
	assertFindingSeverity(t, report, "OPERATIONAL_KEY_HASH_MISSING", severityError)
	assertNoFinding(t, report, "OPERATIONAL_KEY_HASH_MISSING_LEGACY")
}

func TestAuditDetectsCaseAndLengthAliases(t *testing.T) {
	identity := identityKey(0xab)
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
	assertFinding(t, report, "DID_LEGACY_SHORT_FORM")
	assertFinding(t, report, "DID_NORMALIZATION_ALIAS")
	assertFinding(t, report, "DID_NORMALIZED_DUPLICATE")
}

func TestAuditDetectsBaseAccountPubKeyAddressMismatchInNestedVestingAccount(t *testing.T) {
	identity := identityKey(0x41)
	storedKey := identityKey(0x42)
	otherKey := identityKey(0x43)
	address := addressForKey(t, "/cosmos.crypto.ed25519.PubKey", storedKey)
	did := canonicalDID(identity)
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

func TestAuditRejectsNonSubgroupCosmosEd25519PublicKey(t *testing.T) {
	identity := identityKey(0x44)
	invalidTxKey := make([]byte, ed25519.PublicKeySize)
	address := addressForKey(t, "/cosmos.crypto.ed25519.PubKey", invalidTxKey)
	did := canonicalDID(identity)
	document := makeDocument(t, []map[string]any{
		zeroneRecord(address, identity, did, 1),
	}, []map[string]any{
		mappingRecord(address, identity, did),
	}, []any{
		baseAccount(address, "/cosmos.crypto.ed25519.PubKey", invalidTxKey),
	})

	report, err := auditDocument(document, "invalid-cosmos-ed25519.json")
	if err != nil {
		t.Fatal(err)
	}
	assertFindingSeverity(t, report, "COSMOS_PUBKEY_INVALID", severityError)
}

func TestAuditDetectsKeyAnomaliesAndMissingMapping(t *testing.T) {
	identity := identityKey(0x51)
	txKey := append([]byte{0x02}, bytes.Repeat([]byte{0x52}, 32)...)
	address := addressForKey(t, "/cosmos.crypto.secp256k1.PubKey", txKey)
	did := canonicalDID(identity)
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

func TestAuditOperationalKeyHashMismatchIsAlwaysAnError(t *testing.T) {
	identity := identityKey(0x53)
	txKey := append([]byte{0x02}, bytes.Repeat([]byte{0x54}, 32)...)
	address := addressForKey(t, "/cosmos.crypto.secp256k1.PubKey", txKey)
	account := zeroneRecord(address, identity, canonicalDID(identity), 1)
	account["operational_key_hash"] = "A" + fmt.Sprintf("%063x", 1)
	document := makeDocumentForChain(
		t,
		"zerone-1",
		[]map[string]any{account},
		[]map[string]any{mappingRecord(address, identity, canonicalDID(identity))},
		[]any{baseAccount(address, "/cosmos.crypto.secp256k1.PubKey", txKey)},
		[]any{},
		true,
	)

	report, err := auditDocument(document, "bad-hash.json")
	if err != nil {
		t.Fatal(err)
	}
	assertFindingSeverity(t, report, "OPERATIONAL_KEY_HASH_INVALID", severityError)

	account["operational_key_hash"] = fmt.Sprintf("%064x", 1)
	document = makeDocumentForChain(
		t,
		"zerone-1",
		[]map[string]any{account},
		[]map[string]any{mappingRecord(address, identity, canonicalDID(identity))},
		[]any{baseAccount(address, "/cosmos.crypto.secp256k1.PubKey", txKey)},
		[]any{},
		true,
	)
	report, err = auditDocument(document, "wrong-hash.json")
	if err != nil {
		t.Fatal(err)
	}
	assertFindingSeverity(t, report, "OPERATIONAL_KEY_HASH_MISMATCH", severityError)
}

func TestAuditRejectsNonSubgroupEd25519IdentityMaterial(t *testing.T) {
	identity := make([]byte, ed25519.PublicKeySize)
	txKey := append([]byte{0x02}, bytes.Repeat([]byte{0x55}, 32)...)
	address := addressForKey(t, "/cosmos.crypto.secp256k1.PubKey", txKey)
	did := canonicalDID(identity)
	document := makeDocument(t, []map[string]any{
		zeroneRecord(address, identity, did, 1),
	}, []map[string]any{
		mappingRecord(address, identity, did),
	}, []any{
		baseAccount(address, "/cosmos.crypto.secp256k1.PubKey", txKey),
	})

	report, err := auditDocument(document, "invalid-points.json")
	if err != nil {
		t.Fatal(err)
	}
	assertFinding(t, report, "IDENTITY_KEY_ED25519_INVALID")
	assertFinding(t, report, "OPERATIONAL_KEY_ED25519_INVALID")
	assertFinding(t, report, "MAPPING_KEY_ED25519_INVALID")
}

func TestAuditFlagsMissingCurrentRotationState(t *testing.T) {
	identity := identityKey(0x61)
	rotated := identityKey(0x62)
	txKey := append([]byte{0x02}, bytes.Repeat([]byte{0x63}, 32)...)
	address := addressForKey(t, "/cosmos.crypto.secp256k1.PubKey", txKey)
	did := canonicalDID(identity)
	account := zeroneRecord(address, identity, did, 2)
	setOperationalKey(account, rotated)
	document := makeDocument(t, []map[string]any{account}, []map[string]any{
		mappingRecord(address, identity, did),
	}, []any{
		baseAccount(address, "/cosmos.crypto.secp256k1.PubKey", txKey),
	})

	report, err := auditDocument(document, "rotation.json")
	if err != nil {
		t.Fatal(err)
	}
	assertFinding(t, report, "KEY_ROTATION_RECORD_MISSING")
}

func TestAuditAcceptsExportedRotationAnchorForRotatedAccount(t *testing.T) {
	identity := identityKey(0x64)
	rotated := identityKey(0x65)
	txKey := append([]byte{0x03}, bytes.Repeat([]byte{0x66}, 32)...)
	address := addressForKey(t, "/cosmos.crypto.secp256k1.PubKey", txKey)
	account := zeroneRecord(address, identity, canonicalDID(identity), 2)
	setOperationalKey(account, rotated)
	document := makeDocumentForChain(
		t,
		"zerone-2",
		[]map[string]any{account},
		[]map[string]any{mappingRecord(address, identity, canonicalDID(identity))},
		[]any{baseAccount(address, "/cosmos.crypto.secp256k1.PubKey", txKey)},
		[]any{map[string]any{"address": address, "height": "42"}},
		true,
	)

	report, err := auditDocument(document, "rotation-present.json")
	if err != nil {
		t.Fatal(err)
	}
	if report.Summary.Errors != 0 || report.Summary.Warnings != 0 {
		t.Fatalf("expected consistent rotation state, got %+v", report.Findings)
	}
	if !report.Coverage.RotationStatePresent || report.Summary.KeyRotations != 1 {
		t.Fatalf("rotation field was not reflected in report: %+v", report)
	}
}

func TestAuditLabelsMissingLegacyRotationAnchor(t *testing.T) {
	identity := identityKey(0x67)
	rotated := identityKey(0x69)
	txKey := append([]byte{0x02}, bytes.Repeat([]byte{0x6a}, 32)...)
	address := addressForKey(t, "/cosmos.crypto.secp256k1.PubKey", txKey)
	account := zeroneRecord(address, identity, canonicalDID(identity), 2)
	setOperationalKey(account, rotated)
	document := makeDocumentForChain(
		t,
		"zerone-1",
		[]map[string]any{account},
		[]map[string]any{mappingRecord(address, identity, canonicalDID(identity))},
		[]any{baseAccount(address, "/cosmos.crypto.secp256k1.PubKey", txKey)},
		nil,
		false,
	)

	report, err := auditDocument(document, "legacy-rotation.json")
	if err != nil {
		t.Fatal(err)
	}
	assertFindingSeverity(t, report, "KEY_ROTATION_RECORD_MISSING_LEGACY", severityWarning)
	assertNoFinding(t, report, "KEY_ROTATION_RECORD_MISSING")
}

func TestAuditRejectsRotationRecordVersionAndShapeAnomalies(t *testing.T) {
	identity := identityKey(0x6b)
	rotated := identityKey(0x6c)
	txKey := append([]byte{0x03}, bytes.Repeat([]byte{0x6d}, 32)...)
	address := addressForKey(t, "/cosmos.crypto.secp256k1.PubKey", txKey)
	orphanAddress, err := bech32.ConvertAndEncode("zrn", bytes.Repeat([]byte{0x6e}, 20))
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name      string
		version   uint32
		rotations []any
		code      string
	}{
		{
			name:      "version one cannot have anchor",
			version:   1,
			rotations: []any{map[string]any{"address": address, "height": 42}},
			code:      "KEY_ROTATION_RECORD_UNEXPECTED",
		},
		{
			name:    "null record",
			version: 1,
			rotations: []any{
				nil,
			},
			code: "KEY_ROTATION_RECORD_INVALID",
		},
		{
			name:      "zero height",
			version:   2,
			rotations: []any{map[string]any{"address": address, "height": 0}},
			code:      "KEY_ROTATION_HEIGHT_INVALID",
		},
		{
			name:      "orphan",
			version:   1,
			rotations: []any{map[string]any{"address": orphanAddress, "height": 42}},
			code:      "KEY_ROTATION_RECORD_ORPHANED",
		},
		{
			name:    "duplicate",
			version: 2,
			rotations: []any{
				map[string]any{"address": address, "height": 42},
				map[string]any{"address": address, "height": "43"},
			},
			code: "KEY_ROTATION_RECORD_DUPLICATE",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			account := zeroneRecord(address, identity, canonicalDID(identity), test.version)
			if test.version > 1 {
				setOperationalKey(account, rotated)
			}
			document := makeDocumentForChain(
				t,
				"zerone-2",
				[]map[string]any{account},
				[]map[string]any{mappingRecord(address, identity, canonicalDID(identity))},
				[]any{baseAccount(address, "/cosmos.crypto.secp256k1.PubKey", txKey)},
				test.rotations,
				true,
			)
			report, err := auditDocument(document, test.name+".json")
			if err != nil {
				t.Fatal(err)
			}
			assertFindingSeverity(t, report, test.code, severityError)
		})
	}
}

func TestAuditReportsUnknownCosmosKeyTypeWithoutGuessing(t *testing.T) {
	identity := identityKey(0x68)
	address, err := bech32.ConvertAndEncode("zrn", bytes.Repeat([]byte{0x69}, 20))
	if err != nil {
		t.Fatal(err)
	}
	did := canonicalDID(identity)
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
	identity := identityKey(0x71)
	address, err := bech32.ConvertAndEncode("zrn", bytes.Repeat([]byte{0x72}, 20))
	if err != nil {
		t.Fatal(err)
	}
	did := canonicalDID(identity)
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

func TestStandaloneModuleCannotSelfAssertLegacyProfile(t *testing.T) {
	identity := identityKey(0x72)
	address, err := bech32.ConvertAndEncode("zrn", bytes.Repeat([]byte{0x73}, 20))
	if err != nil {
		t.Fatal(err)
	}
	legacyDID := "did:zrn:" + hex.EncodeToString(identity)[:32]
	account := zeroneRecord(address, identity, legacyDID, 1)
	delete(account, "operational_key_hash")
	module := map[string]any{
		"chain_id":     "zerone-1",
		"accounts":     []map[string]any{account},
		"did_mappings": []map[string]any{mappingRecord(address, identity, legacyDID)},
	}
	data, err := json.Marshal(module)
	if err != nil {
		t.Fatal(err)
	}

	report, err := auditDocument(data, "unscoped-module.json")
	if err != nil {
		t.Fatal(err)
	}
	if report.Coverage.ChainID != "" || report.Coverage.ValidationProfile != "source-canonical" {
		t.Fatalf("standalone module asserted a legacy profile: %+v", report.Coverage)
	}
	assertFindingSeverity(t, report, "DID_LEGACY_SHORT_FORM", severityError)
	assertFindingSeverity(t, report, "OPERATIONAL_KEY_HASH_MISSING", severityError)
	assertNoFinding(t, report, "OPERATIONAL_KEY_HASH_MISSING_LEGACY")
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

func TestAuditOutputIsDeterministic(t *testing.T) {
	identityA := identityKey(0x73)
	identityB := identityKey(0x74)
	txKeyA := append([]byte{0x02}, bytes.Repeat([]byte{0x75}, 32)...)
	txKeyB := append([]byte{0x03}, bytes.Repeat([]byte{0x76}, 32)...)
	addressA := addressForKey(t, "/cosmos.crypto.secp256k1.PubKey", txKeyA)
	addressB := addressForKey(t, "/cosmos.crypto.secp256k1.PubKey", txKeyB)
	accountA := zeroneRecord(addressA, identityA, canonicalDID(identityA), 2)
	setOperationalKey(accountA, identityKey(0x77))
	accountB := zeroneRecord(addressB, identityB, canonicalDID(identityB), 1)
	document := makeDocumentForChain(
		t,
		"zerone-2",
		[]map[string]any{accountA, accountB},
		[]map[string]any{
			mappingRecord(addressA, identityA, canonicalDID(identityA)),
			mappingRecord(addressB, identityB, canonicalDID(identityB)),
		},
		[]any{
			baseAccount(addressA, "/cosmos.crypto.secp256k1.PubKey", txKeyA),
			baseAccount(addressB, "/cosmos.crypto.secp256k1.PubKey", txKeyB),
		},
		[]any{
			map[string]any{"address": addressA, "height": 42},
			map[string]any{"address": addressA, "height": 43},
			map[string]any{"address": addressB, "height": 44},
		},
		true,
	)

	var expected []byte
	for iteration := 0; iteration < 20; iteration++ {
		report, err := auditDocument(document, "deterministic.json")
		if err != nil {
			t.Fatal(err)
		}
		encoded, err := json.Marshal(report)
		if err != nil {
			t.Fatal(err)
		}
		if iteration == 0 {
			expected = encoded
			continue
		}
		if !bytes.Equal(encoded, expected) {
			t.Fatalf("report changed between runs\nfirst: %s\nlater: %s", expected, encoded)
		}
	}
}

func makeDocument(
	t *testing.T,
	accounts []map[string]any,
	mappings []map[string]any,
	cosmosAccounts []any,
) []byte {
	return makeDocumentForChain(t, "zerone-test-1", accounts, mappings, cosmosAccounts, []any{}, true)
}

func makeDocumentForChain(
	t *testing.T,
	chainID string,
	accounts []map[string]any,
	mappings []map[string]any,
	cosmosAccounts []any,
	rotations []any,
	includeRotationState bool,
) []byte {
	t.Helper()
	zeroneAuth := map[string]any{
		"accounts":     accounts,
		"did_mappings": mappings,
	}
	if includeRotationState {
		zeroneAuth["last_key_rotations"] = rotations
	}
	document := map[string]any{
		"chain_id": chainID,
		"app_state": map[string]any{
			"zerone_auth": zeroneAuth,
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
	digest := sha256.Sum256(identity)
	return map[string]any{
		"address":                 address,
		"did":                     did,
		"public_key":              hex.EncodeToString(identity),
		"account_type":            "agent",
		"operational_key_hash":    hex.EncodeToString(digest[:]),
		"operational_public_key":  hex.EncodeToString(identity),
		"operational_key_version": version,
	}
}

func setOperationalKey(account map[string]any, key []byte) {
	digest := sha256.Sum256(key)
	account["operational_public_key"] = hex.EncodeToString(key)
	account["operational_key_hash"] = hex.EncodeToString(digest[:])
}

func identityKey(seedByte byte) []byte {
	privateKey := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{seedByte}, ed25519.SeedSize))
	publicKey := privateKey.Public().(ed25519.PublicKey)
	return append([]byte(nil), publicKey...)
}

func canonicalDID(identity []byte) string {
	return "did:zrn:" + hex.EncodeToString(identity)
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

func assertFindingSeverity(t *testing.T, report Report, code, severity string) {
	t.Helper()
	for _, finding := range report.Findings {
		if finding.Code == code && finding.Severity == severity {
			return
		}
	}
	t.Fatalf("finding %s with severity %s missing from %+v", code, severity, report.Findings)
}

func assertNoFinding(t *testing.T, report Report, code string) {
	t.Helper()
	for _, finding := range report.Findings {
		if finding.Code == code {
			t.Fatalf("unexpected finding %s in %+v", code, report.Findings)
		}
	}
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
