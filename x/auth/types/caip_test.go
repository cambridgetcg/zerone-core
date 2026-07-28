package types_test

import (
	"crypto/sha256"
	"fmt"
	"strings"
	"testing"

	"github.com/cosmos/cosmos-sdk/types/bech32"

	"github.com/zerone-chain/zerone/x/auth/types"
)

const caipTestAddress = "zrn1m037n75vk2jhdr56y2ptzjjj02uljwnqwwzr7z"

func TestCosmosChainReference(t *testing.T) {
	t.Parallel()

	longChainID := "123456789012345678901234567890123456789012345678"
	tests := []struct {
		name    string
		chainID string
		want    string
		wantErr bool
	}{
		{name: "zerone direct", chainID: "zerone-2", want: "zerone-2"},
		{name: "maximum direct length", chainID: strings.Repeat("a", 32), want: strings.Repeat("a", 32)},
		{name: "hashed without dash is direct", chainID: "hashed", want: "hashed"},
		{name: "hash dash is direct", chainID: "hash-", want: "hash-"},
		{name: "reserved prefix official vector", chainID: "hashed-", want: "hashed-c904589232422def"},
		{name: "reserved prefix with suffix official vector", chainID: "hashed-123", want: "hashed-99df5cd68192b33e"},
		{name: "reserved hashed prefix", chainID: "hashed-zerone", want: hashedReference("hashed-zerone")},
		{name: "unsupported character", chainID: "zerone_2", want: hashedReference("zerone_2")},
		{name: "long official vector", chainID: longChainID, want: "hashed-0204c92a0388779d"},
		{name: "space official vector", chainID: " ", want: "hashed-36a9e7f1c95b82ff"},
		{name: "empty", wantErr: true},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := types.CosmosChainReference(tt.chainID)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected an error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, got)
			}
		})
	}
}

func TestCAIPIdentifiers(t *testing.T) {
	t.Parallel()

	chainID, err := types.CAIP2ChainID("zerone-2")
	if err != nil {
		t.Fatalf("unexpected CAIP-2 error: %v", err)
	}
	if chainID != "cosmos:zerone-2" {
		t.Fatalf("expected cosmos:zerone-2, got %q", chainID)
	}

	accountID, err := types.CAIP10AccountID("zerone-2", caipTestAddress)
	if err != nil {
		t.Fatalf("unexpected CAIP-10 error: %v", err)
	}
	want := "cosmos:zerone-2:" + caipTestAddress
	if accountID != want {
		t.Fatalf("expected %q, got %q", want, accountID)
	}
}

func TestCAIP10AccountIDRejectsInvalidAddresses(t *testing.T) {
	_, addressBytes, err := bech32.DecodeAndConvert(caipTestAddress)
	if err != nil {
		t.Fatalf("failed to decode fixture address: %v", err)
	}
	wrongHRP, err := bech32.ConvertAndEncode("cosmos", addressBytes)
	if err != nil {
		t.Fatalf("failed to encode wrong-HRP fixture: %v", err)
	}
	overlong, err := bech32.ConvertAndEncode("zrn", make([]byte, 21))
	if err != nil {
		t.Fatalf("failed to encode overlong fixture: %v", err)
	}

	tests := []struct {
		name    string
		address string
	}{
		{name: "empty"},
		{name: "malformed", address: "not-an-address"},
		{name: "valid wrong hrp", address: wrongHRP},
		{name: "overlong payload", address: overlong},
		{name: "noncanonical case", address: strings.ToUpper(caipTestAddress)},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if _, err := types.CAIP10AccountID("zerone-2", tt.address); err == nil {
				t.Fatalf("expected %q to be rejected", tt.address)
			}
		})
	}
}

func hashedReference(chainID string) string {
	sum := sha256.Sum256([]byte(chainID))
	return fmt.Sprintf("hashed-%x", sum[:8])
}
