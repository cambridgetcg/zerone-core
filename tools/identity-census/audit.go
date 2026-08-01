package main

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/cosmos/cosmos-sdk/types/bech32"
	"golang.org/x/crypto/ripemd160"
)

const (
	severityError   = "error"
	severityWarning = "warning"
)

// Finding is one deterministic, machine-readable census result.
type Finding struct {
	Severity string   `json:"severity"`
	Code     string   `json:"code"`
	Message  string   `json:"message"`
	Address  string   `json:"address,omitempty"`
	DID      string   `json:"did,omitempty"`
	Location string   `json:"location,omitempty"`
	Related  []string `json:"related,omitempty"`
}

// Coverage makes the audit boundary explicit. In particular, the current
// zerone_auth export schema does not contain the last-rotation store.
type Coverage struct {
	InputKind        string   `json:"input_kind"`
	CosmosAuth       bool     `json:"cosmos_auth_present"`
	CompleteSnapshot bool     `json:"complete_snapshot"`
	RotationState    string   `json:"rotation_state"`
	Limitations      []string `json:"limitations"`
}

type Summary struct {
	ZeroneAccounts int `json:"zerone_accounts"`
	DIDMappings    int `json:"did_mappings"`
	CosmosAccounts int `json:"cosmos_accounts"`
	Errors         int `json:"errors"`
	Warnings       int `json:"warnings"`
}

// Report is the stable JSON output of an identity census.
type Report struct {
	Source   string    `json:"source"`
	Coverage Coverage  `json:"coverage"`
	Summary  Summary   `json:"summary"`
	Findings []Finding `json:"findings"`
}

type zeroneAccount struct {
	Address               string          `json:"address"`
	DID                   string          `json:"did"`
	PublicKey             string          `json:"public_key"`
	AccountType           string          `json:"account_type"`
	OperationalPublicKey  string          `json:"operational_public_key"`
	OperationalKeyVersion json.RawMessage `json:"operational_key_version"`
}

type didMapping struct {
	DID    string `json:"did"`
	Bech32 string `json:"bech32"`
	PubKey string `json:"pub_key"`
}

type cosmosBaseAccount struct {
	Address  string
	PubKey   any
	Location string
}

type didOccurrence struct {
	Raw      string
	Address  string
	Location string
}

type auditor struct {
	report         Report
	didOccurrences map[string][]didOccurrence
}

func newAuditor(source string) *auditor {
	return &auditor{
		report: Report{
			Source: source,
			Coverage: Coverage{
				RotationState: "not exported by zerone_auth GenesisState; only rotation evidence in account fields can be checked",
				Limitations: []string{
					"signature possession and private-key control cannot be proven from exported state",
					"historical rotation events and the last-rotation cooldown height are absent from the current export schema",
					"unknown Cosmos public-key types are reported but cannot have their address invariant recomputed",
				},
			},
			Findings: []Finding{},
		},
		didOccurrences: make(map[string][]didOccurrence),
	}
}

func (a *auditor) add(f Finding) {
	a.report.Findings = append(a.report.Findings, f)
}

// auditDocument audits a full genesis document, an app_state object, or a
// standalone zerone_auth module object. A full census needs both auth modules.
func auditDocument(data []byte, source string) (Report, error) {
	var root map[string]json.RawMessage
	if err := json.Unmarshal(data, &root); err != nil {
		return Report{}, fmt.Errorf("decode input JSON: %w", err)
	}

	a := newAuditor(source)
	modules := root
	if appStateRaw, ok := root["app_state"]; ok {
		if err := json.Unmarshal(appStateRaw, &modules); err != nil {
			return Report{}, fmt.Errorf("decode app_state: %w", err)
		}
		a.report.Coverage.InputKind = "genesis"
	} else if _, hasZeroneAuth := root["zerone_auth"]; hasZeroneAuth {
		a.report.Coverage.InputKind = "app_state"
	} else if _, hasAccounts := root["accounts"]; hasAccounts {
		a.report.Coverage.InputKind = "zerone_auth_module"
		moduleRaw, err := json.Marshal(root)
		if err != nil {
			return Report{}, fmt.Errorf("re-encode zerone_auth module: %w", err)
		}
		modules = map[string]json.RawMessage{"zerone_auth": moduleRaw}
	} else {
		return Report{}, fmt.Errorf("zerone_auth module not found")
	}

	zeroneRaw, ok := modules["zerone_auth"]
	if !ok {
		return Report{}, fmt.Errorf("zerone_auth module not found")
	}
	accounts, mappings, err := parseZeroneAuth(zeroneRaw)
	if err != nil {
		return Report{}, err
	}
	a.report.Summary.ZeroneAccounts = len(accounts)
	a.report.Summary.DIDMappings = len(mappings)

	var cosmosAccounts []cosmosBaseAccount
	if authRaw, exists := modules["auth"]; exists {
		a.report.Coverage.CosmosAuth = true
		cosmosAccounts, err = parseCosmosAuth(authRaw)
		if err != nil {
			return Report{}, err
		}
	} else {
		a.add(Finding{
			Severity: severityError,
			Code:     "COSMOS_AUTH_STATE_MISSING",
			Message:  "Cosmos auth state is absent; BaseAccount public-key/address invariants cannot be audited",
			Location: "auth",
		})
	}
	a.report.Coverage.CompleteSnapshot = a.report.Coverage.CosmosAuth
	a.report.Summary.CosmosAccounts = len(cosmosAccounts)

	a.auditZeroneAccounts(accounts)
	a.auditMappings(mappings)
	a.auditParity(accounts, mappings)
	a.auditDIDGroups()
	a.auditCosmosAccounts(cosmosAccounts, accounts)
	a.finish()
	return a.report, nil
}

func parseZeroneAuth(raw json.RawMessage) ([]zeroneAccount, []didMapping, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return nil, nil, fmt.Errorf("decode zerone_auth state: %w", err)
	}

	var accounts []zeroneAccount
	if accountRaw, ok := fields["accounts"]; ok && string(accountRaw) != "null" {
		if err := json.Unmarshal(accountRaw, &accounts); err != nil {
			return nil, nil, fmt.Errorf("decode zerone_auth.accounts: %w", err)
		}
	}
	var mappings []didMapping
	if mappingRaw, ok := fields["did_mappings"]; ok && string(mappingRaw) != "null" {
		if err := json.Unmarshal(mappingRaw, &mappings); err != nil {
			return nil, nil, fmt.Errorf("decode zerone_auth.did_mappings: %w", err)
		}
	}
	return accounts, mappings, nil
}

func parseCosmosAuth(raw json.RawMessage) ([]cosmosBaseAccount, error) {
	var state map[string]json.RawMessage
	if err := json.Unmarshal(raw, &state); err != nil {
		return nil, fmt.Errorf("decode auth state: %w", err)
	}
	accountRaw, exists := state["accounts"]
	if !exists || string(accountRaw) == "null" {
		return nil, nil
	}
	var entries []any
	if err := json.Unmarshal(accountRaw, &entries); err != nil {
		return nil, fmt.Errorf("decode auth.accounts: %w", err)
	}

	var result []cosmosBaseAccount
	for i, entry := range entries {
		location := fmt.Sprintf("auth.accounts[%d]", i)
		var candidates []cosmosBaseAccount
		findBaseAccounts(entry, location, &candidates)
		if len(candidates) == 0 {
			result = append(result, cosmosBaseAccount{Location: location})
			continue
		}
		result = append(result, candidates...)
	}
	return result, nil
}

// findBaseAccounts recognizes BaseAccount JSON without coupling the tool to
// every vesting/module-account wrapper. A BaseAccount always has address,
// account_number, and sequence fields; pub_key may be null or omitted.
func findBaseAccounts(value any, location string, out *[]cosmosBaseAccount) {
	switch v := value.(type) {
	case map[string]any:
		_, hasAddress := v["address"]
		_, hasAccountNumber := v["account_number"]
		_, hasSequence := v["sequence"]
		if hasAddress && hasAccountNumber && hasSequence {
			address, _ := v["address"].(string)
			*out = append(*out, cosmosBaseAccount{
				Address:  address,
				PubKey:   v["pub_key"],
				Location: location,
			})
			return
		}
		keys := make([]string, 0, len(v))
		for key := range v {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			findBaseAccounts(v[key], location+"."+key, out)
		}
	case []any:
		for i, child := range v {
			findBaseAccounts(child, fmt.Sprintf("%s[%d]", location, i), out)
		}
	}
}

func (a *auditor) auditZeroneAccounts(accounts []zeroneAccount) {
	seenAddresses := make(map[string]int)
	for i, account := range accounts {
		location := fmt.Sprintf("zerone_auth.accounts[%d]", i)
		seenAddresses[account.Address]++
		a.validateAddress(account.Address, location+".address")

		normalized, valid := a.auditDID(account.DID, account.Address, location+".did")
		if valid {
			a.didOccurrences[normalized] = append(a.didOccurrences[normalized], didOccurrence{
				Raw: account.DID, Address: account.Address, Location: location + ".did",
			})
		}

		identityKey, identityValid := a.auditHexKey(
			account.PublicKey,
			account.Address,
			account.DID,
			location+".public_key",
			"IDENTITY",
		)
		operationalKey, operationalValid := a.auditHexKey(
			account.OperationalPublicKey,
			account.Address,
			account.DID,
			location+".operational_public_key",
			"OPERATIONAL",
		)

		if valid && identityValid && !didMatchesKey(account.DID, identityKey) {
			a.add(Finding{
				Severity: severityError,
				Code:     "DID_KEY_DERIVATION_MISMATCH",
				Message:  "DID does not derive from the stored identity public key",
				Address:  account.Address,
				DID:      account.DID,
				Location: location,
			})
		}

		version, versionOK := parseUint32(account.OperationalKeyVersion)
		if !versionOK || version == 0 {
			a.add(Finding{
				Severity: severityError,
				Code:     "OPERATIONAL_KEY_VERSION_INVALID",
				Message:  "operational_key_version must be a positive uint32",
				Address:  account.Address,
				DID:      account.DID,
				Location: location + ".operational_key_version",
			})
		} else {
			if version == 1 && identityValid && operationalValid && !equalBytes(identityKey, operationalKey) {
				a.add(Finding{
					Severity: severityError,
					Code:     "UNRECORDED_OPERATIONAL_KEY_CHANGE",
					Message:  "operational key differs from the identity key while its version is still 1",
					Address:  account.Address,
					DID:      account.DID,
					Location: location,
				})
			}
			if version > 1 || (identityValid && operationalValid && !equalBytes(identityKey, operationalKey)) {
				a.add(Finding{
					Severity: severityWarning,
					Code:     "ROTATION_STATE_NOT_EXPORTED",
					Message:  "account shows rotation evidence, but last-rotation height/history is absent from exported genesis and cannot be preserved or audited",
					Address:  account.Address,
					DID:      account.DID,
					Location: location,
				})
			}
		}
	}

	for address, count := range seenAddresses {
		if count > 1 {
			a.add(Finding{
				Severity: severityError,
				Code:     "DUPLICATE_ZERONE_ADDRESS",
				Message:  fmt.Sprintf("address appears in %d zerone_auth account records", count),
				Address:  address,
			})
		}
	}
}

func (a *auditor) auditMappings(mappings []didMapping) {
	for i, mapping := range mappings {
		location := fmt.Sprintf("zerone_auth.did_mappings[%d]", i)
		a.validateAddress(mapping.Bech32, location+".bech32")
		normalized, valid := a.auditDID(mapping.DID, mapping.Bech32, location+".did")
		if valid {
			a.didOccurrences[normalized] = append(a.didOccurrences[normalized], didOccurrence{
				Raw: mapping.DID, Address: mapping.Bech32, Location: location + ".did",
			})
		}
		key, keyValid := a.auditHexKey(
			mapping.PubKey,
			mapping.Bech32,
			mapping.DID,
			location+".pub_key",
			"MAPPING",
		)
		if valid && keyValid && !didMatchesKey(mapping.DID, key) {
			a.add(Finding{
				Severity: severityError,
				Code:     "MAPPING_DID_KEY_DERIVATION_MISMATCH",
				Message:  "mapping DID does not derive from the mapping public key",
				Address:  mapping.Bech32,
				DID:      mapping.DID,
				Location: location,
			})
		}
	}
}

func (a *auditor) auditParity(accounts []zeroneAccount, mappings []didMapping) {
	accountsByAddress := make(map[string][]zeroneAccount)
	mappingsByDID := make(map[string][]didMapping)
	mappingsByAddress := make(map[string][]didMapping)
	for _, account := range accounts {
		accountsByAddress[account.Address] = append(accountsByAddress[account.Address], account)
	}
	for _, mapping := range mappings {
		mappingsByDID[mapping.DID] = append(mappingsByDID[mapping.DID], mapping)
		mappingsByAddress[mapping.Bech32] = append(mappingsByAddress[mapping.Bech32], mapping)
	}

	for _, account := range accounts {
		exact := mappingsByDID[account.DID]
		if len(exact) == 0 {
			a.add(Finding{
				Severity: severityError,
				Code:     "DID_MAPPING_MISSING",
				Message:  "account has no DID mapping with the same stored DID",
				Address:  account.Address,
				DID:      account.DID,
			})
			continue
		}
		if len(exact) > 1 {
			a.add(Finding{
				Severity: severityError,
				Code:     "DID_MAPPING_DUPLICATE",
				Message:  fmt.Sprintf("DID has %d mapping records", len(exact)),
				Address:  account.Address,
				DID:      account.DID,
			})
		}
		matched := false
		for _, mapping := range exact {
			if mapping.Bech32 == account.Address && mapping.PubKey == account.PublicKey {
				matched = true
				break
			}
		}
		if !matched {
			a.add(Finding{
				Severity: severityError,
				Code:     "DID_MAPPING_MISMATCH",
				Message:  "DID mapping does not match the account address and identity public key",
				Address:  account.Address,
				DID:      account.DID,
			})
		}
	}

	for _, mapping := range mappings {
		if len(accountsByAddress[mapping.Bech32]) == 0 {
			a.add(Finding{
				Severity: severityError,
				Code:     "DID_MAPPING_ORPHANED",
				Message:  "DID mapping points to an address with no zerone_auth account",
				Address:  mapping.Bech32,
				DID:      mapping.DID,
			})
		}
	}

	for address, addressMappings := range mappingsByAddress {
		rawDIDs := make(map[string]struct{})
		for _, mapping := range addressMappings {
			rawDIDs[mapping.DID] = struct{}{}
		}
		if len(rawDIDs) > 1 {
			a.add(Finding{
				Severity: severityError,
				Code:     "ADDRESS_HAS_MULTIPLE_DID_MAPPINGS",
				Message:  fmt.Sprintf("address has %d distinct DID mappings", len(rawDIDs)),
				Address:  address,
				Related:  sortedKeys(rawDIDs),
			})
		}
	}
}

func (a *auditor) auditDIDGroups() {
	normalizedDIDs := make([]string, 0, len(a.didOccurrences))
	for normalized := range a.didOccurrences {
		normalizedDIDs = append(normalizedDIDs, normalized)
	}
	sort.Strings(normalizedDIDs)

	for _, normalized := range normalizedDIDs {
		occurrences := a.didOccurrences[normalized]
		rawDIDs := make(map[string]struct{})
		addresses := make(map[string]struct{})
		for _, occurrence := range occurrences {
			rawDIDs[occurrence.Raw] = struct{}{}
			if occurrence.Address != "" {
				addresses[occurrence.Address] = struct{}{}
			}
		}
		if len(rawDIDs) > 1 {
			a.add(Finding{
				Severity: severityError,
				Code:     "DID_NORMALIZATION_ALIAS",
				Message:  "distinct stored DIDs collapse to the same lowercase 32-hex migration identifier",
				DID:      normalized,
				Related:  sortedKeys(rawDIDs),
			})
		}
		if len(addresses) > 1 {
			a.add(Finding{
				Severity: severityError,
				Code:     "DID_NORMALIZED_DUPLICATE",
				Message:  "one normalized DID is associated with multiple addresses",
				DID:      normalized,
				Related:  sortedKeys(addresses),
			})
		}
	}
}

func (a *auditor) auditCosmosAccounts(cosmosAccounts []cosmosBaseAccount, zeroneAccounts []zeroneAccount) {
	byAddress := make(map[string][]cosmosBaseAccount)
	for _, account := range cosmosAccounts {
		if account.Address == "" {
			a.add(Finding{
				Severity: severityError,
				Code:     "COSMOS_BASEACCOUNT_UNREADABLE",
				Message:  "auth account entry does not contain a recognizable BaseAccount",
				Location: account.Location,
			})
			continue
		}
		byAddress[account.Address] = append(byAddress[account.Address], account)
		a.validateAddress(account.Address, account.Location+".address")
		a.auditBaseAccountKey(account)
	}

	for address, records := range byAddress {
		if len(records) > 1 {
			a.add(Finding{
				Severity: severityError,
				Code:     "DUPLICATE_COSMOS_BASEACCOUNT",
				Message:  fmt.Sprintf("address appears in %d Cosmos BaseAccount records", len(records)),
				Address:  address,
			})
		}
	}

	for _, account := range zeroneAccounts {
		records := byAddress[account.Address]
		if len(records) == 0 {
			a.add(Finding{
				Severity: severityError,
				Code:     "COSMOS_BASEACCOUNT_MISSING",
				Message:  "registered zerone_auth account has no matching Cosmos BaseAccount",
				Address:  account.Address,
				DID:      account.DID,
			})
			continue
		}
		if len(records) == 1 && records[0].PubKey == nil {
			a.add(Finding{
				Severity: severityWarning,
				Code:     "REGISTERED_BASEACCOUNT_PUBKEY_MISSING",
				Message:  "registered account has no exported Cosmos public key, so signer/address continuity cannot be checked",
				Address:  account.Address,
				DID:      account.DID,
				Location: records[0].Location + ".pub_key",
			})
		}
	}
}

func (a *auditor) auditBaseAccountKey(account cosmosBaseAccount) {
	if account.PubKey == nil {
		return
	}
	pubKeyType, keyBytes, supported, err := decodeCosmosPubKey(account.PubKey)
	if err != nil {
		a.add(Finding{
			Severity: severityError,
			Code:     "COSMOS_PUBKEY_INVALID",
			Message:  err.Error(),
			Address:  account.Address,
			Location: account.Location + ".pub_key",
		})
		return
	}
	if !supported {
		a.add(Finding{
			Severity: severityWarning,
			Code:     "COSMOS_PUBKEY_UNSUPPORTED",
			Message:  fmt.Sprintf("public-key type %q is not supported by this offline invariant checker", pubKeyType),
			Address:  account.Address,
			Location: account.Location + ".pub_key",
		})
		return
	}

	hrp, _, err := bech32.DecodeAndConvert(account.Address)
	if err != nil {
		return // validateAddress emits the actionable address finding.
	}
	derived, err := cosmosAddressBytes(pubKeyType, keyBytes)
	if err != nil {
		a.add(Finding{
			Severity: severityError,
			Code:     "COSMOS_PUBKEY_INVALID",
			Message:  err.Error(),
			Address:  account.Address,
			Location: account.Location + ".pub_key",
		})
		return
	}
	expected, err := bech32.ConvertAndEncode(hrp, derived)
	if err != nil {
		a.add(Finding{
			Severity: severityError,
			Code:     "COSMOS_ADDRESS_DERIVATION_FAILED",
			Message:  err.Error(),
			Address:  account.Address,
			Location: account.Location,
		})
		return
	}
	if expected != strings.ToLower(account.Address) {
		a.add(Finding{
			Severity: severityError,
			Code:     "BASEACCOUNT_PUBKEY_ADDRESS_MISMATCH",
			Message:  fmt.Sprintf("stored public key derives %s, not the BaseAccount address", expected),
			Address:  account.Address,
			Location: account.Location,
		})
	}
}

func (a *auditor) auditDID(did, address, location string) (string, bool) {
	if !strings.HasPrefix(did, "did:zrn:") {
		a.add(Finding{
			Severity: severityError,
			Code:     "DID_FORMAT_INVALID",
			Message:  "DID must start with did:zrn:",
			Address:  address,
			DID:      did,
			Location: location,
		})
		return "", false
	}
	suffix := strings.TrimPrefix(did, "did:zrn:")
	if len(suffix) != 32 && len(suffix) != 64 {
		a.add(Finding{
			Severity: severityError,
			Code:     "DID_LENGTH_INVALID",
			Message:  fmt.Sprintf("DID suffix has %d hex characters; expected 32 or 64", len(suffix)),
			Address:  address,
			DID:      did,
			Location: location,
		})
		return "", false
	}
	if _, err := hex.DecodeString(suffix); err != nil {
		a.add(Finding{
			Severity: severityError,
			Code:     "DID_HEX_INVALID",
			Message:  "DID suffix is not valid hexadecimal",
			Address:  address,
			DID:      did,
			Location: location,
		})
		return "", false
	}
	if suffix != strings.ToLower(suffix) {
		a.add(Finding{
			Severity: severityWarning,
			Code:     "DID_NON_CANONICAL_CASE",
			Message:  "DID suffix is not lowercase",
			Address:  address,
			DID:      did,
			Location: location,
		})
	}
	if len(suffix) == 64 {
		a.add(Finding{
			Severity: severityWarning,
			Code:     "DID_NON_CANONICAL_LENGTH",
			Message:  "64-hex DID form aliases the canonical lowercase 32-hex migration identifier",
			Address:  address,
			DID:      did,
			Location: location,
		})
	}
	return "did:zrn:" + strings.ToLower(suffix[:32]), true
}

func (a *auditor) auditHexKey(value, address, did, location, kind string) ([]byte, bool) {
	if len(value) != 64 {
		a.add(Finding{
			Severity: severityError,
			Code:     kind + "_KEY_LENGTH_INVALID",
			Message:  fmt.Sprintf("hex public key has %d characters; expected 64 (32 bytes)", len(value)),
			Address:  address,
			DID:      did,
			Location: location,
		})
		return nil, false
	}
	key, err := hex.DecodeString(value)
	if err != nil {
		a.add(Finding{
			Severity: severityError,
			Code:     kind + "_KEY_HEX_INVALID",
			Message:  "public key is not valid hexadecimal",
			Address:  address,
			DID:      did,
			Location: location,
		})
		return nil, false
	}
	if value != strings.ToLower(value) {
		a.add(Finding{
			Severity: severityWarning,
			Code:     kind + "_KEY_NON_CANONICAL_CASE",
			Message:  "hex public key is not lowercase",
			Address:  address,
			DID:      did,
			Location: location,
		})
	}
	return key, true
}

func (a *auditor) validateAddress(address, location string) {
	hrp, payload, err := bech32.DecodeAndConvert(address)
	if err != nil || hrp != "zrn" || len(payload) != 20 {
		a.add(Finding{
			Severity: severityError,
			Code:     "ZERONE_ADDRESS_INVALID",
			Message:  "address must be lowercase zrn Bech32 carrying exactly 20 bytes",
			Address:  address,
			Location: location,
		})
		return
	}
	canonical, err := bech32.ConvertAndEncode(hrp, payload)
	if err != nil || canonical != address {
		a.add(Finding{
			Severity: severityError,
			Code:     "ZERONE_ADDRESS_NON_CANONICAL",
			Message:  "address is valid Bech32 but is not its canonical lowercase encoding",
			Address:  address,
			Location: location,
		})
	}
}

func (a *auditor) finish() {
	sort.SliceStable(a.report.Findings, func(i, j int) bool {
		left, right := a.report.Findings[i], a.report.Findings[j]
		leftKey := left.Severity + "\x00" + left.Code + "\x00" + left.Address + "\x00" + left.DID + "\x00" + left.Location
		rightKey := right.Severity + "\x00" + right.Code + "\x00" + right.Address + "\x00" + right.DID + "\x00" + right.Location
		return leftKey < rightKey
	})
	for _, finding := range a.report.Findings {
		switch finding.Severity {
		case severityError:
			a.report.Summary.Errors++
		case severityWarning:
			a.report.Summary.Warnings++
		}
	}
}

func didMatchesKey(did string, key []byte) bool {
	suffix := strings.TrimPrefix(did, "did:zrn:")
	keyHex := hex.EncodeToString(key)
	switch len(suffix) {
	case 32:
		return strings.EqualFold(suffix, keyHex[:32])
	case 64:
		return strings.EqualFold(suffix, keyHex)
	default:
		return false
	}
}

func parseUint32(raw json.RawMessage) (uint32, bool) {
	if len(raw) == 0 || string(raw) == "null" {
		return 0, false
	}
	var number json.Number
	if err := json.Unmarshal(raw, &number); err == nil {
		if value, err := strconv.ParseUint(string(number), 10, 32); err == nil {
			return uint32(value), true
		}
	}
	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		if value, err := strconv.ParseUint(text, 10, 32); err == nil {
			return uint32(value), true
		}
	}
	return 0, false
}

func decodeCosmosPubKey(value any) (string, []byte, bool, error) {
	object, ok := value.(map[string]any)
	if !ok {
		return "", nil, false, fmt.Errorf("public key must be an object or null")
	}
	pubKeyType, _ := object["@type"].(string)
	if pubKeyType == "" {
		pubKeyType, _ = object["type"].(string)
	}
	if pubKeyType == "" {
		return "", nil, false, fmt.Errorf("public key has no @type/type discriminator")
	}
	switch pubKeyType {
	case "/cosmos.crypto.ed25519.PubKey",
		"tendermint/PubKeyEd25519",
		"/cosmos.crypto.secp256k1.PubKey",
		"tendermint/PubKeySecp256k1":
		// These are the two address algorithms this tool can recompute.
	default:
		return pubKeyType, nil, false, nil
	}
	encoded, _ := object["key"].(string)
	if encoded == "" {
		encoded, _ = object["value"].(string)
	}
	if encoded == "" {
		return pubKeyType, nil, false, fmt.Errorf("public key %q has no base64 key/value", pubKeyType)
	}
	key, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return pubKeyType, nil, false, fmt.Errorf("public key %q is not valid base64: %w", pubKeyType, err)
	}

	switch pubKeyType {
	case "/cosmos.crypto.ed25519.PubKey", "tendermint/PubKeyEd25519":
		if len(key) != 32 {
			return pubKeyType, nil, true, fmt.Errorf("Ed25519 public key has %d bytes; expected 32", len(key))
		}
		return pubKeyType, key, true, nil
	case "/cosmos.crypto.secp256k1.PubKey", "tendermint/PubKeySecp256k1":
		if len(key) != 33 {
			return pubKeyType, nil, true, fmt.Errorf("secp256k1 public key has %d bytes; expected 33", len(key))
		}
		if key[0] != 0x02 && key[0] != 0x03 {
			return pubKeyType, nil, true, fmt.Errorf("secp256k1 public key is not compressed (prefix 0x%02x)", key[0])
		}
		return pubKeyType, key, true, nil
	}
	return pubKeyType, key, false, nil
}

func cosmosAddressBytes(pubKeyType string, key []byte) ([]byte, error) {
	switch pubKeyType {
	case "/cosmos.crypto.ed25519.PubKey", "tendermint/PubKeyEd25519":
		sum := sha256.Sum256(key)
		return sum[:20], nil
	case "/cosmos.crypto.secp256k1.PubKey", "tendermint/PubKeySecp256k1":
		sum := sha256.Sum256(key)
		hasher := ripemd160.New()
		_, _ = hasher.Write(sum[:])
		return hasher.Sum(nil), nil
	default:
		return nil, fmt.Errorf("unsupported Cosmos public-key type %q", pubKeyType)
	}
}

func sortedKeys[V any](values map[string]V) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func equalBytes(left, right []byte) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}
