package types_test

import (
	"bytes"
	"crypto/ed25519"
	"encoding/hex"
	"strings"
	"testing"

	"google.golang.org/protobuf/proto"

	"github.com/zerone-chain/zerone/x/auth/types"
)

func validAuthGenesis(t *testing.T) *types.GenesisState {
	t.Helper()
	privateKey := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{0x71}, ed25519.SeedSize))
	publicKey := privateKey.Public().(ed25519.PublicKey)
	publicKeyHex := hex.EncodeToString(publicKey)
	keyHash, err := types.OperationalKeyHash(publicKey)
	if err != nil {
		t.Fatal(err)
	}
	params := types.DefaultParams()
	account := &types.Account{
		Address:               rotationTestAddress,
		Did:                   "did:zrn:" + publicKeyHex,
		PublicKey:             publicKeyHex,
		AccountType:           "agent",
		OperationalKeyHash:    keyHash,
		OperationalPublicKey:  publicKeyHex,
		OperationalKeyVersion: 1,
		ReputationScore:       500_000,
		CreatedAtBlock:        100,
		LastActiveBlock:       100,
		Flags: &types.AccountFlags{
			CanSubmitClaims: true,
			CanChallenge:    true,
		},
		Metadata: `{"name":"genesis"}`,
	}
	return &types.GenesisState{
		Params:   &params,
		Accounts: []*types.Account{account},
		DidMappings: []*types.DIDMapping{{
			Did: account.Did, Bech32: account.Address, PubKey: account.PublicKey,
		}},
		LastKeyRotations: []*types.KeyRotationRecord{},
	}
}

func rotatedAuthGenesis(t *testing.T) *types.GenesisState {
	t.Helper()
	genesis := validAuthGenesis(t)
	operationalPrivateKey := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{0x72}, ed25519.SeedSize))
	operationalKey := operationalPrivateKey.Public().(ed25519.PublicKey)
	operationalKeyHex := hex.EncodeToString(operationalKey)
	keyHash, err := types.OperationalKeyHash(operationalKey)
	if err != nil {
		t.Fatal(err)
	}
	genesis.Accounts[0].OperationalPublicKey = operationalKeyHex
	genesis.Accounts[0].OperationalKeyHash = keyHash
	genesis.Accounts[0].OperationalKeyVersion = 2
	genesis.Accounts[0].LastActiveBlock = 150
	genesis.LastKeyRotations = []*types.KeyRotationRecord{{Address: rotationTestAddress, Height: 150}}
	return genesis
}

func cloneGenesis(t *testing.T, genesis *types.GenesisState) *types.GenesisState {
	t.Helper()
	return proto.Clone(genesis).(*types.GenesisState)
}

func TestDefaultAndPopulatedAuthGenesisValidate(t *testing.T) {
	if err := types.DefaultGenesis().Validate(); err != nil {
		t.Fatalf("default genesis rejected: %v", err)
	}
	if err := validAuthGenesis(t).Validate(); err != nil {
		t.Fatalf("valid populated genesis rejected: %v", err)
	}
	if err := rotatedAuthGenesis(t).Validate(); err != nil {
		t.Fatalf("valid rotated genesis rejected: %v", err)
	}
}

func TestHistoricalExportAllowsOnlyZerone1LegacyDIDSpellings(t *testing.T) {
	base := validAuthGenesis(t)
	legacyDIDs := map[string]string{
		"lowercase short canonical output": "did:zrn:" + base.Accounts[0].PublicKey[:ed25519.PublicKeySize],
		"uppercase short suffix":           "did:zrn:" + strings.ToUpper(base.Accounts[0].PublicKey[:ed25519.PublicKeySize]),
		"uppercase full suffix":            "did:zrn:" + strings.ToUpper(base.Accounts[0].PublicKey),
	}
	for name, legacyDID := range legacyDIDs {
		t.Run(name, func(t *testing.T) {
			legacy := cloneGenesis(t, base)
			legacy.Accounts[0].Did = legacyDID
			legacy.DidMappings[0].Did = legacyDID

			if err := legacy.Validate(); err == nil {
				t.Fatal("strict genesis validation accepted a historical DID spelling")
			}
			for _, chainID := range []string{"", "zerone-2", "zerone-1-copy"} {
				if err := legacy.ValidateForExport(chainID); err == nil {
					t.Fatalf("export validation accepted a historical DID spelling on chain %q", chainID)
				}
			}
			if err := legacy.ValidateForExport("zerone-1"); err != nil {
				t.Fatalf("historical zerone-1 DID export rejected: %v", err)
			}
		})
	}

	legacy := cloneGenesis(t, base)
	legacyDID := "did:zrn:" + legacy.Accounts[0].PublicKey[:ed25519.PublicKeySize]
	legacy.Accounts[0].Did = legacyDID
	legacy.DidMappings[0].Did = legacyDID

	t.Run("composes with missing operational key hash", func(t *testing.T) {
		historical := cloneGenesis(t, legacy)
		historical.Accounts[0].OperationalKeyHash = ""
		if err := historical.ValidateForExport("zerone-1"); err != nil {
			t.Fatalf("typical historical zerone-1 account export rejected: %v", err)
		}
		if err := historical.Validate(); err == nil {
			t.Fatal("strict genesis validation accepted combined historical exceptions")
		}
	})

	t.Run("not derived from identity key", func(t *testing.T) {
		corrupt := cloneGenesis(t, legacy)
		corrupt.Accounts[0].Did = "did:zrn:" + strings.Repeat("00", ed25519.PublicKeySize/2)
		corrupt.DidMappings[0].Did = corrupt.Accounts[0].Did
		if err := corrupt.ValidateForExport("zerone-1"); err == nil {
			t.Fatal("historical export accepted a short DID not derived from the identity key")
		}
	})

	t.Run("uppercase prefix", func(t *testing.T) {
		corrupt := cloneGenesis(t, legacy)
		corrupt.Accounts[0].Did = "DID:ZRN:" + corrupt.Accounts[0].PublicKey[:ed25519.PublicKeySize]
		corrupt.DidMappings[0].Did = corrupt.Accounts[0].Did
		if err := corrupt.ValidateForExport("zerone-1"); err == nil {
			t.Fatal("historical export accepted a prefix rejected by the legacy grammar")
		}
	})

	t.Run("mapping must match account", func(t *testing.T) {
		corrupt := cloneGenesis(t, legacy)
		corrupt.DidMappings[0].Did = "did:zrn:" + corrupt.Accounts[0].PublicKey
		if err := corrupt.ValidateForExport("zerone-1"); err == nil {
			t.Fatal("historical export accepted different account and mapping DIDs")
		}
	})
}

func TestHistoricalExportAllowsOnlyZerone1MissingOperationalKeyHash(t *testing.T) {
	legacy := validAuthGenesis(t)
	legacy.Accounts[0].OperationalKeyHash = ""

	if err := legacy.Validate(); err == nil {
		t.Fatal("strict genesis validation accepted a missing operational key hash")
	}
	for _, chainID := range []string{"", "zerone-2", "zerone-1-copy"} {
		if err := legacy.ValidateForExport(chainID); err == nil {
			t.Fatalf("export validation accepted a missing operational key hash on chain %q", chainID)
		}
	}
	if err := legacy.ValidateForExport("zerone-1"); err != nil {
		t.Fatalf("historical zerone-1 export rejected: %v", err)
	}

	t.Run("nonempty mismatch", func(t *testing.T) {
		corrupt := cloneGenesis(t, legacy)
		corrupt.Accounts[0].OperationalKeyHash = strings.Repeat("00", 32)
		if err := corrupt.ValidateForExport("zerone-1"); err == nil {
			t.Fatal("historical export accepted a nonempty incorrect operational key hash")
		}
	})

	t.Run("invalid operational key", func(t *testing.T) {
		corrupt := cloneGenesis(t, legacy)
		corrupt.Accounts[0].OperationalPublicKey = strings.Repeat("00", 32)
		if err := corrupt.ValidateForExport("zerone-1"); err == nil {
			t.Fatal("historical export accepted an invalid operational public key")
		}
	})

	t.Run("version one key mismatch", func(t *testing.T) {
		corrupt := cloneGenesis(t, legacy)
		other := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{0x74}, ed25519.SeedSize)).Public().(ed25519.PublicKey)
		corrupt.Accounts[0].OperationalPublicKey = hex.EncodeToString(other)
		if err := corrupt.ValidateForExport("zerone-1"); err == nil {
			t.Fatal("historical export accepted different identity and operational keys")
		}
	})

	t.Run("rotated account", func(t *testing.T) {
		corrupt := rotatedAuthGenesis(t)
		corrupt.Accounts[0].OperationalKeyHash = ""
		if err := corrupt.ValidateForExport("zerone-1"); err == nil {
			t.Fatal("historical export accepted a missing hash on a rotated account")
		}
	})

	t.Run("missing rotation continuity", func(t *testing.T) {
		corrupt := rotatedAuthGenesis(t)
		corrupt.LastKeyRotations = nil
		if err := corrupt.ValidateForExport("zerone-1"); err == nil {
			t.Fatal("historical export accepted missing key-rotation continuity")
		}
	})

	t.Run("missing DID mapping", func(t *testing.T) {
		corrupt := cloneGenesis(t, legacy)
		corrupt.DidMappings = nil
		if err := corrupt.ValidateForExport("zerone-1"); err == nil {
			t.Fatal("historical export accepted a missing DID mapping")
		}
	})
}

func TestAuthGenesisRejectsMalformedAccountState(t *testing.T) {
	base := validAuthGenesis(t)
	tests := []struct {
		name   string
		mutate func(*types.GenesisState)
	}{
		{name: "nil params", mutate: func(gs *types.GenesisState) { gs.Params = nil }},
		{name: "nil account", mutate: func(gs *types.GenesisState) { gs.Accounts[0] = nil }},
		{name: "noncanonical address", mutate: func(gs *types.GenesisState) { gs.Accounts[0].Address = strings.ToUpper(gs.Accounts[0].Address) }},
		{name: "truncated DID", mutate: func(gs *types.GenesisState) { gs.Accounts[0].Did = gs.Accounts[0].Did[:len(gs.Accounts[0].Did)-2] }},
		{name: "weak identity key", mutate: func(gs *types.GenesisState) {
			gs.Accounts[0].PublicKey = hex.EncodeToString(append([]byte{1}, make([]byte, 31)...))
			gs.Accounts[0].Did = "did:zrn:" + gs.Accounts[0].PublicKey
		}},
		{name: "weak operational key", mutate: func(gs *types.GenesisState) { gs.Accounts[0].OperationalPublicKey = strings.Repeat("00", 32) }},
		{name: "wrong operational hash", mutate: func(gs *types.GenesisState) { gs.Accounts[0].OperationalKeyHash = strings.Repeat("00", 32) }},
		{name: "zero version", mutate: func(gs *types.GenesisState) { gs.Accounts[0].OperationalKeyVersion = 0 }},
		{name: "version one key mismatch", mutate: func(gs *types.GenesisState) {
			other := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{0x73}, ed25519.SeedSize)).Public().(ed25519.PublicKey)
			gs.Accounts[0].OperationalPublicKey = hex.EncodeToString(other)
			gs.Accounts[0].OperationalKeyHash, _ = types.OperationalKeyHash(other)
		}},
		{name: "invalid account type", mutate: func(gs *types.GenesisState) { gs.Accounts[0].AccountType = "administrator" }},
		{name: "excess reputation", mutate: func(gs *types.GenesisState) { gs.Accounts[0].ReputationScore = 1_000_001 }},
		{name: "zero creation height", mutate: func(gs *types.GenesisState) { gs.Accounts[0].CreatedAtBlock = 0 }},
		{name: "reversed activity bounds", mutate: func(gs *types.GenesisState) { gs.Accounts[0].LastActiveBlock = 99 }},
		{name: "nil flags", mutate: func(gs *types.GenesisState) { gs.Accounts[0].Flags = nil }},
		{name: "stale freeze reason", mutate: func(gs *types.GenesisState) { gs.Accounts[0].Flags.FreezeReason = "stale" }},
		{name: "forbidden contract capabilities", mutate: func(gs *types.GenesisState) { gs.Accounts[0].AccountType = "contract" }},
		{name: "oversized metadata", mutate: func(gs *types.GenesisState) {
			gs.Params.MaxMetadataLength = 1
		}},
		{name: "duplicate account", mutate: func(gs *types.GenesisState) {
			gs.Accounts = append(gs.Accounts, proto.Clone(gs.Accounts[0]).(*types.Account))
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			genesis := cloneGenesis(t, base)
			test.mutate(genesis)
			if err := genesis.Validate(); err == nil {
				t.Fatal("expected malformed account genesis to be rejected")
			}
		})
	}
}

func TestAuthGenesisRejectsMalformedMappings(t *testing.T) {
	base := validAuthGenesis(t)
	tests := []struct {
		name   string
		mutate func(*types.GenesisState)
	}{
		{name: "missing", mutate: func(gs *types.GenesisState) { gs.DidMappings = nil }},
		{name: "nil", mutate: func(gs *types.GenesisState) { gs.DidMappings[0] = nil }},
		{name: "wrong address", mutate: func(gs *types.GenesisState) { gs.DidMappings[0].Bech32 = secondRegistrationTestAddress }},
		{name: "wrong DID", mutate: func(gs *types.GenesisState) { gs.DidMappings[0].Did = "did:zrn:" + strings.Repeat("00", 32) }},
		{name: "wrong public key", mutate: func(gs *types.GenesisState) { gs.DidMappings[0].PubKey = strings.Repeat("00", 32) }},
		{name: "duplicate", mutate: func(gs *types.GenesisState) {
			gs.DidMappings = append(gs.DidMappings, proto.Clone(gs.DidMappings[0]).(*types.DIDMapping))
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			genesis := cloneGenesis(t, base)
			test.mutate(genesis)
			if err := genesis.Validate(); err == nil {
				t.Fatal("expected malformed DID mapping genesis to be rejected")
			}
		})
	}
}

func TestAuthGenesisRejectsMalformedRotationContinuity(t *testing.T) {
	versionOne := validAuthGenesis(t)
	versionTwo := rotatedAuthGenesis(t)
	tests := []struct {
		name   string
		base   *types.GenesisState
		mutate func(*types.GenesisState)
	}{
		{name: "record on version one", base: versionOne, mutate: func(gs *types.GenesisState) {
			gs.LastKeyRotations = []*types.KeyRotationRecord{{Address: rotationTestAddress, Height: 100}}
		}},
		{name: "missing after rotation", base: versionTwo, mutate: func(gs *types.GenesisState) { gs.LastKeyRotations = nil }},
		{name: "nil record", base: versionTwo, mutate: func(gs *types.GenesisState) { gs.LastKeyRotations[0] = nil }},
		{name: "zero height", base: versionTwo, mutate: func(gs *types.GenesisState) { gs.LastKeyRotations[0].Height = 0 }},
		{name: "before creation", base: versionTwo, mutate: func(gs *types.GenesisState) { gs.LastKeyRotations[0].Height = 99 }},
		{name: "after activity", base: versionTwo, mutate: func(gs *types.GenesisState) { gs.LastKeyRotations[0].Height = 151 }},
		{name: "orphan address", base: versionTwo, mutate: func(gs *types.GenesisState) { gs.LastKeyRotations[0].Address = secondRegistrationTestAddress }},
		{name: "duplicate record", base: versionTwo, mutate: func(gs *types.GenesisState) {
			gs.LastKeyRotations = append(gs.LastKeyRotations, proto.Clone(gs.LastKeyRotations[0]).(*types.KeyRotationRecord))
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			genesis := cloneGenesis(t, test.base)
			test.mutate(genesis)
			if err := genesis.Validate(); err == nil {
				t.Fatal("expected malformed key-rotation continuity to be rejected")
			}
		})
	}
}

func TestNilAuthGenesisRejected(t *testing.T) {
	var genesis *types.GenesisState
	if err := genesis.Validate(); err == nil {
		t.Fatal("nil genesis was accepted")
	}
}
