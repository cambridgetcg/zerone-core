package keeper_test

import (
	"strings"
	"testing"

	"github.com/cosmos/cosmos-sdk/types/bech32"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	"github.com/zerone-chain/zerone/x/auth/keeper"
	"github.com/zerone-chain/zerone/x/auth/types"
)

func TestQueryAccountIdentifier(t *testing.T) {
	k, baseCtx := setupKeeper(t)
	ctx := baseCtx.WithChainID("zerone-2")
	ms := keeper.NewMsgServerImpl(k)
	qs := keeper.NewQueryServerImpl(k)

	_, err := ms.RegisterAccount(ctx, signedRegistration(t, ctx, testAddr1, testPrivKey1, "agent", ""))
	if err != nil {
		t.Fatalf("failed to register fixture account: %v", err)
	}

	before, found := k.GetAccount(ctx, testAddr1)
	if !found {
		t.Fatal("fixture account not found")
	}
	eventCount := len(ctx.EventManager().Events())

	resp, err := qs.AccountIdentifier(ctx, &types.QueryAccountIdentifierRequest{Address: testAddr1})
	if err != nil {
		t.Fatalf("unexpected query error: %v", err)
	}
	if resp.Identifier == nil {
		t.Fatal("expected an identifier")
	}

	identifier := resp.Identifier
	if identifier.Namespace != "cosmos" {
		t.Errorf("expected namespace cosmos, got %q", identifier.Namespace)
	}
	if identifier.Reference != "zerone-2" {
		t.Errorf("expected reference zerone-2, got %q", identifier.Reference)
	}
	if identifier.RawChainId != "zerone-2" {
		t.Errorf("expected raw chain ID zerone-2, got %q", identifier.RawChainId)
	}
	wantAccountID := "cosmos:zerone-2:" + testAddr1
	if identifier.AccountId != wantAccountID {
		t.Errorf("expected account ID %q, got %q", wantAccountID, identifier.AccountId)
	}
	if identifier.Address != testAddr1 || identifier.Did != testDID1 {
		t.Errorf("unexpected native identity projection: address=%q did=%q", identifier.Address, identifier.Did)
	}
	if identifier.AccountType != "agent" {
		t.Errorf("expected account type agent, got %q", identifier.AccountType)
	}
	if identifier.Frozen {
		t.Error("expected account not to be frozen")
	}
	if identifier.CreatedAtBlock != 100 {
		t.Errorf("expected creation block 100, got %d", identifier.CreatedAtBlock)
	}

	after, found := k.GetAccount(ctx, testAddr1)
	if !found || !proto.Equal(before, after) {
		t.Error("account identifier query mutated the stored account")
	}
	if got := len(ctx.EventManager().Events()); got != eventCount {
		t.Errorf("account identifier query emitted events: before=%d after=%d", eventCount, got)
	}
}

func TestQueryAccountIdentifierValidation(t *testing.T) {
	k, baseCtx := setupKeeper(t)
	ctx := baseCtx.WithChainID("zerone-2")
	qs := keeper.NewQueryServerImpl(k)

	_, addressBytes, err := bech32.DecodeAndConvert(testAddr1)
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
		name string
		req  *types.QueryAccountIdentifierRequest
		code codes.Code
	}{
		{name: "nil request", code: codes.InvalidArgument},
		{name: "empty address", req: &types.QueryAccountIdentifierRequest{}, code: codes.InvalidArgument},
		{name: "malformed address", req: &types.QueryAccountIdentifierRequest{Address: "invalid"}, code: codes.InvalidArgument},
		{name: "valid wrong hrp", req: &types.QueryAccountIdentifierRequest{Address: wrongHRP}, code: codes.InvalidArgument},
		{name: "overlong payload", req: &types.QueryAccountIdentifierRequest{Address: overlong}, code: codes.InvalidArgument},
		{name: "noncanonical case", req: &types.QueryAccountIdentifierRequest{Address: strings.ToUpper(testAddr1)}, code: codes.InvalidArgument},
		{name: "missing account", req: &types.QueryAccountIdentifierRequest{Address: testAddr1}, code: codes.NotFound},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			_, err := qs.AccountIdentifier(ctx, tt.req)
			if status.Code(err) != tt.code {
				t.Fatalf("expected gRPC code %s, got %s (%v)", tt.code, status.Code(err), err)
			}
		})
	}
}

func TestRegisterAccountRejectsNoncanonicalAddress(t *testing.T) {
	overlong, err := bech32.ConvertAndEncode("zrn", make([]byte, 21))
	if err != nil {
		t.Fatalf("failed to encode overlong fixture: %v", err)
	}

	tests := map[string]string{
		"uppercase":       strings.ToUpper(testAddr1),
		"21-byte payload": overlong,
	}
	for name, address := range tests {
		t.Run(name, func(t *testing.T) {
			k, baseCtx := setupKeeper(t)
			ctx := baseCtx.WithChainID("zerone-2")
			ms := keeper.NewMsgServerImpl(k)
			msg := signedRegistration(t, ctx, testAddr1, testPrivKey1, "agent", "")
			msg.Sender = address

			if _, err := ms.RegisterAccount(ctx, msg); err == nil {
				t.Fatal("expected noncanonical sender to be rejected")
			}
			if _, found := k.GetAccount(ctx, address); found {
				t.Fatal("rejected account was stored")
			}
		})
	}
}

func TestQueryAccountIdentifierRejectsEmptyChainID(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)
	qs := keeper.NewQueryServerImpl(k)

	_, err := ms.RegisterAccount(ctx, signedRegistration(t, ctx, testAddr1, testPrivKey1, "agent", ""))
	if err != nil {
		t.Fatalf("failed to register fixture account: %v", err)
	}

	_, err = qs.AccountIdentifier(ctx.WithChainID(""), &types.QueryAccountIdentifierRequest{Address: testAddr1})
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("expected FailedPrecondition, got %s (%v)", status.Code(err), err)
	}
}

func TestQueryAccountIdentifierDetectsCorruptDIDMapping(t *testing.T) {
	k, baseCtx := setupKeeper(t)
	ctx := baseCtx.WithChainID("zerone-2")
	ms := keeper.NewMsgServerImpl(k)
	qs := keeper.NewQueryServerImpl(k)

	_, err := ms.RegisterAccount(ctx, signedRegistration(t, ctx, testAddr1, testPrivKey1, "agent", ""))
	if err != nil {
		t.Fatalf("failed to register fixture account: %v", err)
	}
	k.SetDIDMapping(ctx, &types.DIDMapping{
		Did:    testDID1,
		Bech32: testAddr2,
		PubKey: testPubKey1,
	})

	_, err = qs.AccountIdentifier(ctx, &types.QueryAccountIdentifierRequest{Address: testAddr1})
	if status.Code(err) != codes.DataLoss {
		t.Fatalf("expected DataLoss, got %s (%v)", status.Code(err), err)
	}
}
