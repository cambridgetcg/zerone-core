package types_test

import (
	"strings"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/x/claiming_pot/types"
)

func init() {
	cfg := sdk.GetConfig()
	cfg.SetBech32PrefixForAccount("zrn", "zrnpub")
	cfg.SetBech32PrefixForValidator("zrnvaloper", "zrnvaloperpub")
	cfg.SetBech32PrefixForConsensusNode("zrnvalcons", "zrnvalconspub")
}

func mkAddr(seed string) string {
	b := make([]byte, 20)
	copy(b, []byte(seed))
	return sdk.AccAddress(b).String()
}

func TestMsgAddBootstrapEntry_ValidateBasic(t *testing.T) {
	authority := mkAddr("authority-test-aaaa1")
	good1 := mkAddr("agent-aaaaaaaaaaaaaa1")
	good2 := mkAddr("agent-bbbbbbbbbbbbbb2")

	t.Run("valid_single_address", func(t *testing.T) {
		msg := &types.MsgAddBootstrapEntry{
			Authority: authority,
			Addresses: []string{good1},
		}
		if err := msg.ValidateBasic(); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	})

	t.Run("valid_multiple_addresses", func(t *testing.T) {
		msg := &types.MsgAddBootstrapEntry{
			Authority: authority,
			Addresses: []string{good1, good2},
		}
		if err := msg.ValidateBasic(); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	})

	t.Run("empty_authority", func(t *testing.T) {
		msg := &types.MsgAddBootstrapEntry{Authority: "", Addresses: []string{good1}}
		err := msg.ValidateBasic()
		if err == nil {
			t.Fatal("expected error for empty authority")
		}
		if !strings.Contains(err.Error(), "authority cannot be empty") {
			t.Errorf("expected 'authority cannot be empty', got %q", err.Error())
		}
	})

	t.Run("invalid_authority_bech32", func(t *testing.T) {
		msg := &types.MsgAddBootstrapEntry{Authority: "not-bech32", Addresses: []string{good1}}
		err := msg.ValidateBasic()
		if err == nil {
			t.Fatal("expected error for malformed authority")
		}
		if !strings.Contains(err.Error(), "invalid authority address") {
			t.Errorf("expected 'invalid authority address', got %q", err.Error())
		}
	})

	t.Run("empty_addresses_list", func(t *testing.T) {
		msg := &types.MsgAddBootstrapEntry{Authority: authority, Addresses: nil}
		err := msg.ValidateBasic()
		if err == nil {
			t.Fatal("expected error for empty addresses list")
		}
		if !strings.Contains(err.Error(), "addresses list cannot be empty") {
			t.Errorf("expected 'addresses list cannot be empty', got %q", err.Error())
		}
	})

	t.Run("invalid_address_bech32", func(t *testing.T) {
		msg := &types.MsgAddBootstrapEntry{
			Authority: authority,
			Addresses: []string{"not-a-bech32"},
		}
		err := msg.ValidateBasic()
		if err == nil {
			t.Fatal("expected error for malformed address")
		}
		if !strings.Contains(err.Error(), "invalid bech32") {
			t.Errorf("expected 'invalid bech32', got %q", err.Error())
		}
	})

	t.Run("duplicate_addresses_in_request", func(t *testing.T) {
		msg := &types.MsgAddBootstrapEntry{
			Authority: authority,
			Addresses: []string{good1, good1},
		}
		err := msg.ValidateBasic()
		if err == nil {
			t.Fatal("expected error for duplicate addresses")
		}
		if !strings.Contains(err.Error(), "duplicate within request payload") {
			t.Errorf("expected 'duplicate within request payload', got %q", err.Error())
		}
	})

	t.Run("empty_string_in_addresses", func(t *testing.T) {
		msg := &types.MsgAddBootstrapEntry{
			Authority: authority,
			Addresses: []string{good1, ""},
		}
		err := msg.ValidateBasic()
		if err == nil {
			t.Fatal("expected error for empty address string")
		}
	})
}

func TestMsgAddBootstrapEntry_GetSigners(t *testing.T) {
	authority := mkAddr("signer-test-aaaaaa12")
	msg := &types.MsgAddBootstrapEntry{
		Authority: authority,
		Addresses: []string{mkAddr("any-agent-test-aaaa3")},
	}
	signers := msg.GetSigners()
	if len(signers) != 1 {
		t.Fatalf("expected 1 signer, got %d", len(signers))
	}
	if signers[0].String() != authority {
		t.Errorf("expected signer %s, got %s", authority, signers[0].String())
	}
}
