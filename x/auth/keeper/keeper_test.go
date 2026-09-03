package keeper_test

import (
	"bytes"
	"crypto/ed25519"
	"encoding/hex"
	"errors"
	"math"
	"strings"
	"testing"
	"time"

	"cosmossdk.io/log"
	"cosmossdk.io/store"
	storemetrics "cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"

	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"

	"github.com/zerone-chain/zerone/x/auth/keeper"
	"github.com/zerone-chain/zerone/x/auth/types"
)

// ---------- Test Setup ----------

func init() {
	config := sdk.GetConfig()
	config.SetBech32PrefixForAccount("zrn", "zrnpub")
	config.SetBech32PrefixForValidator("zrnvaloper", "zrnvaloperpub")
	config.SetBech32PrefixForConsensusNode("zrnvalcons", "zrnvalconspub")
}

func setupKeeper(t *testing.T) (keeper.Keeper, sdk.Context) {
	t.Helper()

	storeKey := storetypes.NewKVStoreKey(types.StoreKey)

	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), storemetrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	err := stateStore.LoadLatestVersion()
	if err != nil {
		t.Fatalf("failed to load latest version: %v", err)
	}

	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)

	k := keeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(storeKey),
		testAuthority,
	)

	ctx := sdk.NewContext(stateStore, cmtproto.Header{
		Height: 100,
		Time:   testBlockTime,
	}, false, log.NewNopLogger()).WithChainID(testChainID)

	// Set default params
	defaultParams := types.DefaultParams()
	if err := k.SetParams(ctx, &defaultParams); err != nil {
		t.Fatalf("failed to set params: %v", err)
	}

	return k, ctx
}

func setupRotationKeeper(t *testing.T) (keeper.Keeper, sdk.Context) {
	t.Helper()
	k, ctx := setupKeeper(t)
	return k, ctx.WithChainID(testChainID)
}

func requireExportPanic(t *testing.T, operation func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Fatal("expected export to reject invalid state")
		}
	}()
	operation()
}

const (
	testAddr1     = "zrn1m037n75vk2jhdr56y2ptzjjj02uljwnqwwzr7z"
	testAddr2     = "zrn1ur4eyeuuhrkfpcyhykfjsasftv9hn33smszt58"
	testAuthority = "zrn1q0ar3f2cswzlemcss4nu82cd40crftd9utnt0e"
	testChainID   = "zerone-test-1"
)

var (
	testBlockTime = time.Date(2026, time.September, 3, 12, 0, 0, 0, time.UTC)
	testPrivKey1  = ed25519.NewKeyFromSeed(bytes.Repeat([]byte{0x11}, ed25519.SeedSize))
	testPrivKey2  = ed25519.NewKeyFromSeed(bytes.Repeat([]byte{0x22}, ed25519.SeedSize))
	testPubKey1   = hex.EncodeToString(testPrivKey1.Public().(ed25519.PublicKey))
	testPubKey2   = hex.EncodeToString(testPrivKey2.Public().(ed25519.PublicKey))
	testDID1      = "did:zrn:" + testPubKey1
	testDID2      = "did:zrn:" + testPubKey2
)

func signedRegistration(
	t *testing.T,
	ctx sdk.Context,
	sender string,
	privateKey ed25519.PrivateKey,
	accountType string,
	metadata string,
) *types.MsgRegisterAccount {
	t.Helper()
	publicKey := privateKey.Public().(ed25519.PublicKey)
	publicKeyHex := hex.EncodeToString(publicKey)
	did := "did:zrn:" + publicKeyHex
	signBytes, err := types.AccountRegistrationProofSignBytes(
		ctx.ChainID(), sender, did, publicKey, accountType, metadata,
	)
	if err != nil {
		t.Fatalf("build registration proof bytes: %v", err)
	}
	return &types.MsgRegisterAccount{
		Sender:                 sender,
		Did:                    did,
		PublicKey:              publicKeyHex,
		AccountType:            accountType,
		Metadata:               metadata,
		IdentityProofSignature: ed25519.Sign(privateKey, signBytes),
	}
}

func signedRotation(
	t *testing.T,
	k keeper.Keeper,
	ctx sdk.Context,
	sender string,
	currentPrivateKey ed25519.PrivateKey,
	newPrivateKey ed25519.PrivateKey,
	expiresAt time.Time,
) *types.MsgRotateKey {
	t.Helper()
	account, found := k.GetAccount(ctx, sender)
	if !found {
		t.Fatal("account not found while preparing key-rotation authorization")
	}
	newPublicKey := newPrivateKey.Public().(ed25519.PublicKey)
	signBytes, err := types.KeyRotationAuthorizationSignBytes(
		ctx.ChainID(),
		sender,
		account.OperationalKeyVersion,
		newPublicKey,
		expiresAt.Unix(),
	)
	if err != nil {
		t.Fatalf("build key-rotation sign bytes: %v", err)
	}
	acceptanceBytes, err := types.KeyRotationAcceptanceSignBytes(
		ctx.ChainID(),
		sender,
		account.OperationalKeyVersion,
		newPublicKey,
		expiresAt.Unix(),
	)
	if err != nil {
		t.Fatalf("build key-rotation acceptance bytes: %v", err)
	}
	return &types.MsgRotateKey{
		Sender:                      sender,
		NewOperationalKey:           append([]byte(nil), newPublicKey...),
		AuthorizationSignature:      ed25519.Sign(currentPrivateKey, signBytes),
		AuthorizationExpiresAtUnix:  expiresAt.Unix(),
		NewKeyConfirmationSignature: ed25519.Sign(newPrivateKey, acceptanceBytes),
	}
}

// ---------- RegisterAccount Tests ----------

func TestRegisterAccount_Success(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	resp, err := ms.RegisterAccount(ctx, signedRegistration(t, ctx, testAddr1, testPrivKey1, "agent", ""))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}

	// Verify account stored
	account, found := k.GetAccount(ctx, testAddr1)
	if !found {
		t.Fatal("account not found after registration")
	}
	if account.Did != testDID1 {
		t.Errorf("expected DID %s, got %s", testDID1, account.Did)
	}
	if account.PublicKey != testPubKey1 {
		t.Errorf("expected pubkey %s, got %s", testPubKey1, account.PublicKey)
	}
	if account.AccountType != "agent" {
		t.Errorf("expected type agent, got %s", account.AccountType)
	}
	expectedKeyHash, err := types.OperationalKeyHash(testPrivKey1.Public().(ed25519.PublicKey))
	if err != nil {
		t.Fatal(err)
	}
	if account.OperationalKeyHash != expectedKeyHash {
		t.Errorf("expected opkey hash %s, got %s", expectedKeyHash, account.OperationalKeyHash)
	}
	if account.OperationalKeyVersion != 1 {
		t.Errorf("expected version 1, got %d", account.OperationalKeyVersion)
	}
	if account.ReputationScore != 500000 {
		t.Errorf("expected reputation 500000, got %d", account.ReputationScore)
	}
	if !account.Flags.CanSubmitClaims {
		t.Error("expected CanSubmitClaims true")
	}

	// Verify DID mapping stored
	mapping, found := k.GetDIDMapping(ctx, testDID1)
	if !found {
		t.Fatal("DID mapping not found after registration")
	}
	if mapping.Bech32 != testAddr1 {
		t.Errorf("expected bech32 %s, got %s", testAddr1, mapping.Bech32)
	}
	if mapping.PubKey != testPubKey1 {
		t.Errorf("expected pubkey %s, got %s", testPubKey1, mapping.PubKey)
	}
}

func TestRegisterAccount_DuplicateAddress(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	_, err := ms.RegisterAccount(ctx, signedRegistration(t, ctx, testAddr1, testPrivKey1, "agent", ""))
	if err != nil {
		t.Fatalf("unexpected error on first registration: %v", err)
	}

	_, err = ms.RegisterAccount(ctx, signedRegistration(t, ctx, testAddr1, testPrivKey2, "human", ""))
	if err == nil {
		t.Fatal("expected error for duplicate address")
	}
}

func TestRegisterAccount_DuplicateDID(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	_, err := ms.RegisterAccount(ctx, signedRegistration(t, ctx, testAddr1, testPrivKey1, "agent", ""))
	if err != nil {
		t.Fatalf("unexpected error on first registration: %v", err)
	}

	_, err = ms.RegisterAccount(ctx, signedRegistration(t, ctx, testAddr2, testPrivKey1, "human", ""))
	if err == nil {
		t.Fatal("expected error for duplicate DID")
	}
}

// ---------- RotateKey Tests ----------

func TestRotateKey_Success(t *testing.T) {
	k, ctx := setupRotationKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	_, err := ms.RegisterAccount(ctx, signedRegistration(t, ctx, testAddr1, testPrivKey1, "agent", ""))
	if err != nil {
		t.Fatalf("failed to register account: %v", err)
	}

	newPrivateKey := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{0x33}, ed25519.SeedSize))
	msg := signedRotation(
		t, k, ctx, testAddr1, testPrivKey1,
		newPrivateKey,
		ctx.BlockTime().Add(time.Minute),
	)
	_, err = ms.RotateKey(ctx, msg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	account, _ := k.GetAccount(ctx, testAddr1)
	expectedHex := hex.EncodeToString(newPrivateKey.Public().(ed25519.PublicKey))
	if account.OperationalPublicKey != expectedHex {
		t.Errorf("expected OperationalPublicKey %s, got %s", expectedHex, account.OperationalPublicKey)
	}
	if account.OperationalKeyVersion != 2 {
		t.Errorf("expected version 2, got %d", account.OperationalKeyVersion)
	}
}

func TestRotateKey_NotFound(t *testing.T) {
	k, ctx := setupRotationKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	_, err := ms.RotateKey(ctx, &types.MsgRotateKey{
		Sender:                     testAddr1,
		NewOperationalKey:          make([]byte, ed25519.PublicKeySize),
		AuthorizationSignature:     make([]byte, ed25519.SignatureSize),
		AuthorizationExpiresAtUnix: ctx.BlockTime().Add(time.Minute).Unix(),
	})
	if err == nil {
		t.Fatal("expected error for non-existent account")
	}
}

func TestRotateKey_Cooldown(t *testing.T) {
	k, ctx := setupRotationKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	_, _ = ms.RegisterAccount(ctx, signedRegistration(t, ctx, testAddr1, testPrivKey1, "agent", ""))

	key2 := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{0x44}, ed25519.SeedSize))
	key3 := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{0x55}, ed25519.SeedSize))
	_, err := ms.RotateKey(ctx, signedRotation(
		t, k, ctx, testAddr1, testPrivKey1, key2,
		ctx.BlockTime().Add(time.Minute),
	))
	if err != nil {
		t.Fatalf("first rotation failed: %v", err)
	}

	// Immediate second rotation should fail (cooldown)
	second := signedRotation(
		t, k, ctx, testAddr1, key2, key3,
		ctx.BlockTime().Add(time.Minute),
	)
	_, err = ms.RotateKey(ctx, second)
	if err == nil {
		t.Fatal("expected cooldown error")
	}

	// After cooldown passes
	params := k.GetParams(ctx)
	newCtx := ctx.WithBlockHeight(int64(100 + params.KeyRotationCooldown + 1))
	_, err = ms.RotateKey(newCtx, second)
	if err != nil {
		t.Fatalf("rotation after cooldown should succeed: %v", err)
	}
}

func TestRotateKey_FrozenAccount(t *testing.T) {
	k, ctx := setupRotationKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	_, _ = ms.RegisterAccount(ctx, signedRegistration(t, ctx, testAddr1, testPrivKey1, "agent", ""))
	account, _ := k.GetAccount(ctx, testAddr1)
	account.Flags.Frozen = true
	k.SetAccount(ctx, account)

	_, err := ms.RotateKey(ctx, &types.MsgRotateKey{
		Sender:                     testAddr1,
		NewOperationalKey:          make([]byte, ed25519.PublicKeySize),
		AuthorizationSignature:     make([]byte, ed25519.SignatureSize),
		AuthorizationExpiresAtUnix: ctx.BlockTime().Add(time.Minute).Unix(),
	})
	if err == nil {
		t.Fatal("expected error for frozen account")
	}
}

func TestRotateKey_InvalidKeyType(t *testing.T) {
	k, ctx := setupRotationKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	_, _ = ms.RegisterAccount(ctx, signedRegistration(t, ctx, testAddr1, testPrivKey1, "agent", ""))

	_, err := ms.RotateKey(ctx, &types.MsgRotateKey{
		Sender:                     testAddr1,
		NewOperationalKey:          nil,
		AuthorizationSignature:     make([]byte, ed25519.SignatureSize),
		AuthorizationExpiresAtUnix: ctx.BlockTime().Add(time.Minute).Unix(),
	})
	if err == nil {
		t.Fatal("expected error for empty operational key")
	}
}

func TestRotateKey_RejectsInvalidSecondarySignatureWithoutMutation(t *testing.T) {
	k, ctx := setupRotationKeeper(t)
	ms := keeper.NewMsgServerImpl(k)
	_, err := ms.RegisterAccount(ctx, signedRegistration(t, ctx, testAddr1, testPrivKey1, "agent", ""))
	if err != nil {
		t.Fatal(err)
	}

	newPrivateKey := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{0x88}, ed25519.SeedSize))
	msg := signedRotation(
		t, k, ctx, testAddr1, testPrivKey1,
		newPrivateKey,
		ctx.BlockTime().Add(time.Minute),
	)
	msg.AuthorizationSignature[0] ^= 0xff
	_, err = ms.RotateKey(ctx, msg)
	if !errors.Is(err, types.ErrInvalidAuthorizationSig) {
		t.Fatalf("expected invalid authorization signature, got %v", err)
	}
	account, _ := k.GetAccount(ctx, testAddr1)
	if account.OperationalKeyVersion != 1 || account.OperationalPublicKey != testPubKey1 {
		t.Fatal("failed authorization mutated operational-key state")
	}
}

func TestRotateKey_BindsChainIDAndCurrentVersion(t *testing.T) {
	k, ctx := setupRotationKeeper(t)
	ms := keeper.NewMsgServerImpl(k)
	_, err := ms.RegisterAccount(ctx, signedRegistration(t, ctx, testAddr1, testPrivKey1, "agent", ""))
	if err != nil {
		t.Fatal(err)
	}
	newPrivateKey := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{0x99}, ed25519.SeedSize))
	msg := signedRotation(
		t, k, ctx, testAddr1, testPrivKey1,
		newPrivateKey,
		ctx.BlockTime().Add(time.Minute),
	)

	_, err = ms.RotateKey(ctx.WithChainID("zerone-other-1"), msg)
	if !errors.Is(err, types.ErrInvalidAuthorizationSig) {
		t.Fatalf("expected cross-chain replay rejection, got %v", err)
	}

	account, _ := k.GetAccount(ctx, testAddr1)
	account.OperationalKeyVersion = 2
	k.SetAccount(ctx, account)
	_, err = ms.RotateKey(ctx, msg)
	if !errors.Is(err, types.ErrInvalidAuthorizationSig) {
		t.Fatalf("expected key-version replay rejection, got %v", err)
	}
}

func TestRotateKey_UsesConsensusTimeAndBoundsAuthorizationTTL(t *testing.T) {
	tests := []struct {
		name    string
		expires time.Time
		wantErr error
	}{
		{name: "expired", expires: testBlockTime, wantErr: types.ErrKeyAuthorizationExpired},
		{
			name:    "beyond maximum TTL",
			expires: testBlockTime.Add(types.KeyRotationAuthorizationMaxTTL + time.Second),
			wantErr: types.ErrKeyAuthorizationTooFar,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			k, ctx := setupRotationKeeper(t)
			ms := keeper.NewMsgServerImpl(k)
			_, err := ms.RegisterAccount(ctx, signedRegistration(t, ctx, testAddr1, testPrivKey1, "agent", ""))
			if err != nil {
				t.Fatal(err)
			}
			newPrivateKey := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{0xaa}, ed25519.SeedSize))
			msg := signedRotation(
				t, k, ctx, testAddr1, testPrivKey1,
				newPrivateKey, test.expires,
			)
			_, err = ms.RotateKey(ctx, msg)
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("expected %v, got %v", test.wantErr, err)
			}
		})
	}
}

func TestRegisterAccount_RejectsFalseOperationalKeyHash(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)
	msg := signedRegistration(t, ctx, testAddr1, testPrivKey1, "agent", "")
	msg.OperationalKeyHash = "0" + strings.Repeat("1", 63)
	_, err := ms.RegisterAccount(ctx, msg)
	if !errors.Is(err, types.ErrInvalidPublicKey) {
		t.Fatalf("expected false operational-key hash rejection, got %v", err)
	}
	if _, found := k.GetAccount(ctx, testAddr1); found {
		t.Fatal("rejected registration wrote account state")
	}
}

// ---------- Query Tests ----------

func TestQueryAccount(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)
	qs := keeper.NewQueryServerImpl(k)

	_, _ = ms.RegisterAccount(ctx, signedRegistration(t, ctx, testAddr1, testPrivKey1, "agent", ""))

	resp, err := qs.Account(ctx, &types.QueryAccountRequest{Address: testAddr1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Account.Did != testDID1 {
		t.Errorf("expected DID %s, got %s", testDID1, resp.Account.Did)
	}
}

func TestQueryAccountByDID(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)
	qs := keeper.NewQueryServerImpl(k)

	_, _ = ms.RegisterAccount(ctx, signedRegistration(t, ctx, testAddr1, testPrivKey1, "human", ""))

	resp, err := qs.AccountByDID(ctx, &types.QueryAccountByDIDRequest{Did: testDID1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Account.Address != testAddr1 {
		t.Errorf("expected address %s, got %s", testAddr1, resp.Account.Address)
	}
	if resp.Account.AccountType != "human" {
		t.Errorf("expected type human, got %s", resp.Account.AccountType)
	}
}

func TestQueryAccountByDIDPreservesHistoricalZerone1Forms(t *testing.T) {
	tests := []struct {
		name string
		did  string
	}{
		{name: "32 lowercase hex", did: "did:zrn:" + testPubKey1[:32]},
		{name: "32 uppercase hex", did: "did:zrn:" + strings.ToUpper(testPubKey1[:32])},
		{name: "64 uppercase hex", did: "did:zrn:" + strings.ToUpper(testPubKey1)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k, baseCtx := setupKeeper(t)
			ctx := baseCtx.WithChainID("zerone-1")
			qs := keeper.NewQueryServerImpl(k)
			account := &types.Account{
				Address:   testAddr1,
				Did:       tt.did,
				PublicKey: testPubKey1,
			}
			k.SetAccount(ctx, account)
			k.SetDIDMapping(ctx, &types.DIDMapping{
				Did:    tt.did,
				Bech32: testAddr1,
				PubKey: testPubKey1,
			})

			resp, err := qs.AccountByDID(ctx, &types.QueryAccountByDIDRequest{Did: tt.did})
			if err != nil {
				t.Fatalf("historical DID lookup failed: %v", err)
			}
			if resp.Account.Address != testAddr1 || resp.Account.Did != tt.did {
				t.Fatalf("unexpected historical account: %+v", resp.Account)
			}
		})
	}

	t.Run("lookup remains byte exact", func(t *testing.T) {
		k, baseCtx := setupKeeper(t)
		ctx := baseCtx.WithChainID("zerone-1")
		qs := keeper.NewQueryServerImpl(k)
		storedDID := "did:zrn:" + testPubKey1[:32]
		k.SetAccount(ctx, &types.Account{
			Address: testAddr1, Did: storedDID, PublicKey: testPubKey1,
		})
		k.SetDIDMapping(ctx, &types.DIDMapping{
			Did: storedDID, Bech32: testAddr1, PubKey: testPubKey1,
		})

		_, err := qs.AccountByDID(ctx, &types.QueryAccountByDIDRequest{
			Did: "did:zrn:" + strings.ToUpper(testPubKey1[:32]),
		})
		if !errors.Is(err, types.ErrAccountNotFound) {
			t.Fatalf("case variant aliased a byte-distinct historical DID: %v", err)
		}
	})
}

func TestQueryAccountByDIDRejectsHistoricalFormsOutsideZerone1(t *testing.T) {
	tests := []string{
		"did:zrn:" + testPubKey1[:32],
		"did:zrn:" + strings.ToUpper(testPubKey1[:32]),
		"did:zrn:" + strings.ToUpper(testPubKey1),
	}

	for _, did := range tests {
		k, baseCtx := setupKeeper(t)
		ctx := baseCtx.WithChainID("zerone-1-copy")
		qs := keeper.NewQueryServerImpl(k)
		_, err := qs.AccountByDID(ctx, &types.QueryAccountByDIDRequest{Did: did})
		if status.Code(err) != codes.InvalidArgument {
			t.Fatalf("historical DID %q escaped exact zerone-1 boundary: %v", did, err)
		}
	}
}

func TestQueryAccount_NotFound(t *testing.T) {
	k, ctx := setupKeeper(t)
	qs := keeper.NewQueryServerImpl(k)

	_, err := qs.Account(ctx, &types.QueryAccountRequest{Address: testAddr1})
	if err == nil {
		t.Fatal("expected error for non-existent account")
	}
}

func TestQueryAccountRejectsMalformedRequest(t *testing.T) {
	k, ctx := setupKeeper(t)
	qs := keeper.NewQueryServerImpl(k)
	if _, err := qs.Account(ctx, nil); err == nil {
		t.Fatal("nil account query was accepted")
	}
	if _, err := qs.Account(ctx, &types.QueryAccountRequest{Address: strings.ToUpper(testAddr1)}); err == nil {
		t.Fatal("noncanonical account query address was accepted")
	}
	if _, err := qs.AccountByDID(ctx, nil); err == nil {
		t.Fatal("nil DID query was accepted")
	}
	if _, err := qs.AccountByDID(ctx, &types.QueryAccountByDIDRequest{Did: testDID1[:len(testDID1)-2]}); err == nil {
		t.Fatal("noncanonical DID query was accepted")
	}
	if _, err := qs.AccountByDID(ctx.WithChainID("zerone-1"), &types.QueryAccountByDIDRequest{Did: "did:zrn:" + strings.Repeat("g", 32)}); err == nil {
		t.Fatal("non-hex historical DID query was accepted")
	}
}

func TestQueryParams(t *testing.T) {
	k, ctx := setupKeeper(t)
	qs := keeper.NewQueryServerImpl(k)

	resp, err := qs.Params(ctx, &types.QueryParamsRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Params.KeyRotationCooldown != 111 {
		t.Errorf("expected key rotation cooldown 111, got %d", resp.Params.KeyRotationCooldown)
	}
}

func TestQueryFrozenAccounts(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)
	qs := keeper.NewQueryServerImpl(k)

	_, _ = ms.RegisterAccount(ctx, signedRegistration(t, ctx, testAddr1, testPrivKey1, "agent", ""))
	_, _ = ms.RegisterAccount(ctx, signedRegistration(t, ctx, testAddr2, testPrivKey2, "human", ""))

	// Freeze first account
	_, _ = ms.FreezeAccount(ctx, &types.MsgFreezeAccount{
		Sender:  testAddr1,
		Address: testAddr1,
		Reason:  "test freeze",
	})

	resp, err := qs.FrozenAccounts(ctx, &types.QueryFrozenAccountsRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Accounts) != 1 {
		t.Errorf("expected 1 frozen account, got %d", len(resp.Accounts))
	}
	if resp.Accounts[0].Address != testAddr1 {
		t.Errorf("expected frozen account %s, got %s", testAddr1, resp.Accounts[0].Address)
	}
}

// ---------- DID Lookup Tests ----------

func TestGetAccountByDID(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	_, _ = ms.RegisterAccount(ctx, signedRegistration(t, ctx, testAddr1, testPrivKey1, "agent", ""))

	account, found := k.GetAccountByDID(ctx, testDID1)
	if !found {
		t.Fatal("expected to find account by DID")
	}
	if account.Address != testAddr1 {
		t.Errorf("expected address %s, got %s", testAddr1, account.Address)
	}

	addr, found := k.GetAddressForDID(ctx, testDID1)
	if !found {
		t.Fatal("expected to find address for DID")
	}
	if addr != testAddr1 {
		t.Errorf("expected %s, got %s", testAddr1, addr)
	}
}

// ---------- Genesis Tests ----------

func TestInitExportGenesis(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	_, _ = ms.RegisterAccount(ctx, signedRegistration(t, ctx, testAddr1, testPrivKey1, "agent", ""))
	_, _ = ms.RegisterAccount(ctx, signedRegistration(t, ctx, testAddr2, testPrivKey2, "human", ""))

	gs := k.ExportGenesis(ctx)
	if len(gs.Accounts) != 2 {
		t.Errorf("expected 2 accounts in genesis, got %d", len(gs.Accounts))
	}
	if len(gs.DidMappings) != 2 {
		t.Errorf("expected 2 DID mappings in genesis, got %d", len(gs.DidMappings))
	}

	if err := gs.Validate(); err != nil {
		t.Fatalf("genesis validation failed: %v", err)
	}
}

func TestExportGenesisPreservesExactZerone1LegacyAccount(t *testing.T) {
	k, ctx := setupKeeper(t)
	legacy := types.Account{
		Address:               testAddr1,
		Did:                   testDID1,
		PublicKey:             testPubKey1,
		AccountType:           "agent",
		OperationalKeyHash:    "",
		OperationalPublicKey:  testPubKey1,
		OperationalKeyVersion: 1,
		ReputationScore:       500_000,
		CreatedAtBlock:        uint64(ctx.BlockHeight()),
		LastActiveBlock:       uint64(ctx.BlockHeight()),
		Flags: &types.AccountFlags{
			CanSubmitClaims: true,
			CanChallenge:    true,
		},
	}
	k.SetAccount(ctx, &legacy)
	k.SetDIDMapping(ctx, &types.DIDMapping{
		Did:    legacy.Did,
		Bech32: legacy.Address,
		PubKey: legacy.PublicKey,
	})

	exported := k.ExportGenesis(ctx.WithChainID("zerone-1"))
	if len(exported.Accounts) != 1 || exported.Accounts[0].OperationalKeyHash != "" {
		t.Fatalf("legacy account was not preserved exactly: %+v", exported.Accounts)
	}
	if err := exported.Validate(); err == nil {
		t.Fatal("legacy export unexpectedly passed strict import validation")
	}

	requireExportPanic(t, func() {
		k.ExportGenesis(ctx.WithChainID("zerone-2"))
	})
	requireExportPanic(t, func() {
		k.ExportGenesis(ctx.WithChainID(""))
	})
}

func TestNewZerone1RegistrationStillWritesOperationalKeyHash(t *testing.T) {
	k, ctx := setupKeeper(t)
	ctx = ctx.WithChainID("zerone-1")
	ms := keeper.NewMsgServerImpl(k)
	if _, err := ms.RegisterAccount(
		ctx,
		signedRegistration(t, ctx, testAddr1, testPrivKey1, "agent", ""),
	); err != nil {
		t.Fatalf("register account: %v", err)
	}

	account, found := k.GetAccount(ctx, testAddr1)
	if !found {
		t.Fatal("newly registered account not found")
	}
	expected, err := types.OperationalKeyHash(testPrivKey1.Public().(ed25519.PublicKey))
	if err != nil {
		t.Fatal(err)
	}
	if account.OperationalKeyHash != expected {
		t.Fatalf("new zerone-1 write stored operational key hash %q, want %q", account.OperationalKeyHash, expected)
	}
}

// ---------- ValidateBasic Tests ----------

func TestMsgRegisterAccount_ValidateBasic(t *testing.T) {
	_, ctx := setupKeeper(t)
	msg := *signedRegistration(t, ctx, testAddr1, testPrivKey1, "agent", "")
	if err := msg.ValidateBasic(); err != nil {
		t.Errorf("expected valid, got error: %v", err)
	}

	msg.AccountType = "invalid"
	if err := msg.ValidateBasic(); err == nil {
		t.Error("expected error for invalid account type")
	}

	msg.AccountType = "agent"
	msg.PublicKey = ""
	if err := msg.ValidateBasic(); err == nil {
		t.Error("expected error for empty public key")
	}

	msg.PublicKey = "short"
	if err := msg.ValidateBasic(); err == nil {
		t.Error("expected error for short public key")
	}

	msg.PublicKey = testPubKey1
	msg.Did = "invalid"
	if err := msg.ValidateBasic(); err == nil {
		t.Error("expected error for invalid DID")
	}

	missingProof := *signedRegistration(t, ctx, testAddr1, testPrivKey1, "agent", "")
	missingProof.IdentityProofSignature = nil
	if err := missingProof.ValidateBasic(); err == nil {
		t.Error("expected error for missing identity proof signature")
	}

	legacy := *signedRegistration(t, ctx.WithChainID("zerone-1"), testAddr1, testPrivKey1, "agent", "")
	legacy.Did = "did:zrn:" + testPubKey1[:32]
	if err := legacy.ValidateBasic(); err == nil {
		t.Error("new zerone-1 registration accepted the export/query-only legacy DID form")
	}
}

func TestValidateDID(t *testing.T) {
	if err := types.ValidateDID(testDID1); err != nil {
		t.Errorf("expected valid DID: %v", err)
	}

	if err := types.ValidateDID("0000000000000000000000000000000000000000000000000000000000000001"); err == nil {
		t.Error("expected error for missing did:zrn: prefix")
	}

	if err := types.ValidateDID("did:zrn:short"); err == nil {
		t.Error("expected error for short suffix")
	}
}

// ---------- Operational key separation and authorization tests ----------

func TestRegisterAccount_SetsOperationalPublicKey(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	_, err := ms.RegisterAccount(ctx, signedRegistration(t, ctx, testAddr1, testPrivKey1, "agent", ""))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	account, found := k.GetAccount(ctx, testAddr1)
	if !found {
		t.Fatal("account not found")
	}
	if account.OperationalPublicKey != testPubKey1 {
		t.Errorf("expected OperationalPublicKey %s, got %s", testPubKey1, account.OperationalPublicKey)
	}
}

func TestRotateKey_StoresNewPublicKey(t *testing.T) {
	k, ctx := setupRotationKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	_, _ = ms.RegisterAccount(ctx, signedRegistration(t, ctx, testAddr1, testPrivKey1, "agent", ""))

	newPrivateKey := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{0x66}, ed25519.SeedSize))
	_, err := ms.RotateKey(ctx, signedRotation(
		t, k, ctx, testAddr1, testPrivKey1,
		newPrivateKey,
		ctx.BlockTime().Add(time.Minute),
	))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	account, _ := k.GetAccount(ctx, testAddr1)
	expectedHex := hex.EncodeToString(newPrivateKey.Public().(ed25519.PublicKey))
	if account.OperationalPublicKey != expectedHex {
		t.Errorf("expected OperationalPublicKey %s, got %s", expectedHex, account.OperationalPublicKey)
	}
	if account.OperationalKeyVersion != 2 {
		t.Errorf("expected version 2, got %d", account.OperationalKeyVersion)
	}
	expectedHash, err := types.OperationalKeyHash(newPrivateKey.Public().(ed25519.PublicKey))
	if err != nil {
		t.Fatal(err)
	}
	if account.OperationalKeyHash != expectedHash {
		t.Errorf("expected operational key hash %s, got %s", expectedHash, account.OperationalKeyHash)
	}
}

func TestRotateKey_WithoutNewPublicKey_PreservesExisting(t *testing.T) {
	k, ctx := setupRotationKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	_, _ = ms.RegisterAccount(ctx, signedRegistration(t, ctx, testAddr1, testPrivKey1, "agent", ""))

	_, err := ms.RotateKey(ctx, &types.MsgRotateKey{
		Sender:                     testAddr1,
		NewOperationalKey:          nil,
		AuthorizationSignature:     make([]byte, ed25519.SignatureSize),
		AuthorizationExpiresAtUnix: ctx.BlockTime().Add(time.Minute).Unix(),
	})
	if err == nil {
		t.Fatal("expected error when NewOperationalKey is nil")
	}

	account, _ := k.GetAccount(ctx, testAddr1)
	if account.OperationalPublicKey != testPubKey1 {
		t.Errorf("expected OperationalPublicKey preserved as %s, got %s", testPubKey1, account.OperationalPublicKey)
	}
}

// ---------- Phase 4: DID Resolution Tests ----------

func TestGetAddressForDID_NotRegistered(t *testing.T) {
	k, ctx := setupKeeper(t)

	_, found := k.GetAddressForDID(ctx, testDID1)
	if found {
		t.Fatal("expected DID not found for unregistered DID")
	}
}

func TestGetAccountByDID_ReturnsFullAccount(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	_, _ = ms.RegisterAccount(ctx, signedRegistration(t, ctx, testAddr1, testPrivKey1, "agent", ""))

	account, found := k.GetAccountByDID(ctx, testDID1)
	if !found {
		t.Fatal("expected to find account by DID")
	}
	if account.Address != testAddr1 {
		t.Errorf("expected address %s, got %s", testAddr1, account.Address)
	}
	if account.OperationalPublicKey != testPubKey1 {
		t.Errorf("expected OperationalPublicKey %s, got %s", testPubKey1, account.OperationalPublicKey)
	}
	expectedKeyHash, err := types.OperationalKeyHash(testPrivKey1.Public().(ed25519.PublicKey))
	if err != nil {
		t.Fatal(err)
	}
	if account.OperationalKeyHash != expectedKeyHash {
		t.Errorf("expected opkey hash %s, got %s", expectedKeyHash, account.OperationalKeyHash)
	}
}

// ---------- ValidateBasic Phase 2-4 Tests ----------

func TestMsgRotateKey_ValidateBasic(t *testing.T) {
	msg := types.MsgRotateKey{
		Sender:                      testAddr1,
		NewOperationalKey:           append([]byte(nil), testPrivKey2.Public().(ed25519.PublicKey)...),
		AuthorizationSignature:      make([]byte, ed25519.SignatureSize),
		AuthorizationExpiresAtUnix:  testBlockTime.Add(time.Minute).Unix(),
		NewKeyConfirmationSignature: make([]byte, ed25519.SignatureSize),
	}
	if err := msg.ValidateBasic(); err != nil {
		t.Errorf("expected valid, got error: %v", err)
	}

	msg.Sender = "invalid"
	if err := msg.ValidateBasic(); err == nil {
		t.Error("expected error for invalid sender")
	}

	msg.Sender = testAddr1
	msg.NewKeyConfirmationSignature = nil
	if err := msg.ValidateBasic(); err == nil {
		t.Error("expected error for missing new-key confirmation signature")
	}
}

// ---------- LastActiveBlock Update Tests ----------

func TestRegisterAccount_SetsCreatedAndLastActive(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	_, _ = ms.RegisterAccount(ctx, signedRegistration(t, ctx, testAddr1, testPrivKey1, "agent", ""))

	account, _ := k.GetAccount(ctx, testAddr1)
	if account.CreatedAtBlock != 100 {
		t.Errorf("expected CreatedAtBlock 100, got %d", account.CreatedAtBlock)
	}
	if account.LastActiveBlock != 100 {
		t.Errorf("expected LastActiveBlock 100, got %d", account.LastActiveBlock)
	}
}

func TestRotateKey_UpdatesLastActive(t *testing.T) {
	k, ctx := setupRotationKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	_, _ = ms.RegisterAccount(ctx, signedRegistration(t, ctx, testAddr1, testPrivKey1, "agent", ""))

	params := k.GetParams(ctx)
	advancedCtx := ctx.WithBlockHeight(int64(100 + params.KeyRotationCooldown + 1))

	newPrivateKey := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{0x77}, ed25519.SeedSize))
	_, err := ms.RotateKey(advancedCtx, signedRotation(
		t, k, advancedCtx, testAddr1, testPrivKey1,
		newPrivateKey,
		advancedCtx.BlockTime().Add(time.Minute),
	))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	account, _ := k.GetAccount(advancedCtx, testAddr1)
	expected := uint64(100 + params.KeyRotationCooldown + 1)
	if account.LastActiveBlock != expected {
		t.Errorf("expected LastActiveBlock %d, got %d", expected, account.LastActiveBlock)
	}
}

// ---------- FreezeAccount Tests ----------

func TestFreezeAccount_SelfFreeze(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	_, _ = ms.RegisterAccount(ctx, signedRegistration(t, ctx, testAddr1, testPrivKey1, "agent", ""))

	_, err := ms.FreezeAccount(ctx, &types.MsgFreezeAccount{
		Sender:  testAddr1,
		Address: testAddr1,
		Reason:  "compromised key",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	account, _ := k.GetAccount(ctx, testAddr1)
	if !account.Flags.Frozen {
		t.Fatal("expected account to be frozen")
	}
}

func TestFreezeAccount_AuthorityFreeze(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	_, _ = ms.RegisterAccount(ctx, signedRegistration(t, ctx, testAddr1, testPrivKey1, "agent", ""))

	_, err := ms.FreezeAccount(ctx, &types.MsgFreezeAccount{
		Sender:  testAuthority,
		Address: testAddr1,
		Reason:  "malicious activity",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	account, _ := k.GetAccount(ctx, testAddr1)
	if !account.Flags.Frozen {
		t.Fatal("expected account to be frozen by authority")
	}
}

func TestFreezeAccount_UnauthorizedThirdParty(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	_, _ = ms.RegisterAccount(ctx, signedRegistration(t, ctx, testAddr1, testPrivKey1, "agent", ""))

	_, err := ms.FreezeAccount(ctx, &types.MsgFreezeAccount{
		Sender:  testAddr2,
		Address: testAddr1,
		Reason:  "attack",
	})
	if err == nil {
		t.Fatal("expected error for unauthorized freeze")
	}
}

func TestFreezeAccount_AlreadyFrozen(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	_, _ = ms.RegisterAccount(ctx, signedRegistration(t, ctx, testAddr1, testPrivKey1, "agent", ""))

	_, _ = ms.FreezeAccount(ctx, &types.MsgFreezeAccount{
		Sender:  testAddr1,
		Address: testAddr1,
		Reason:  "first freeze",
	})

	_, err := ms.FreezeAccount(ctx, &types.MsgFreezeAccount{
		Sender:  testAddr1,
		Address: testAddr1,
		Reason:  "second freeze",
	})
	if err == nil {
		t.Fatal("expected error for already frozen account")
	}
}

func TestFreezeAccount_NotFound(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	_, err := ms.FreezeAccount(ctx, &types.MsgFreezeAccount{
		Sender:  testAddr1,
		Address: testAddr1,
		Reason:  "test",
	})
	if err == nil {
		t.Fatal("expected error for non-existent account")
	}
}

func TestFreezeAccount_StoresReason(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	_, _ = ms.RegisterAccount(ctx, signedRegistration(t, ctx, testAddr1, testPrivKey1, "agent", ""))

	_, err := ms.FreezeAccount(ctx, &types.MsgFreezeAccount{
		Sender:  testAddr1,
		Address: testAddr1,
		Reason:  "suspected breach",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	account, _ := k.GetAccount(ctx, testAddr1)
	if account.Flags.FreezeReason != "suspected breach" {
		t.Errorf("expected freeze reason 'suspected breach', got '%s'", account.Flags.FreezeReason)
	}
}

// ---------- UnfreezeAccount Tests ----------

func TestUnfreezeAccount_Success(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	_, _ = ms.RegisterAccount(ctx, signedRegistration(t, ctx, testAddr1, testPrivKey1, "agent", ""))
	_, _ = ms.FreezeAccount(ctx, &types.MsgFreezeAccount{
		Sender:  testAddr1,
		Address: testAddr1,
		Reason:  "test",
	})

	_, err := ms.UnfreezeAccount(ctx, &types.MsgUnfreezeAccount{
		Authority: testAuthority,
		Address:   testAddr1,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	account, _ := k.GetAccount(ctx, testAddr1)
	if account.Flags.Frozen {
		t.Fatal("expected account to be unfrozen")
	}
}

func TestUnfreezeAccount_NonAuthority(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	_, _ = ms.RegisterAccount(ctx, signedRegistration(t, ctx, testAddr1, testPrivKey1, "agent", ""))
	_, _ = ms.FreezeAccount(ctx, &types.MsgFreezeAccount{
		Sender:  testAddr1,
		Address: testAddr1,
		Reason:  "test",
	})

	_, err := ms.UnfreezeAccount(ctx, &types.MsgUnfreezeAccount{
		Authority: testAddr2,
		Address:   testAddr1,
	})
	if err == nil {
		t.Fatal("expected error for non-authority unfreeze")
	}
}

func TestUnfreezeAccount_NotFrozen(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	_, _ = ms.RegisterAccount(ctx, signedRegistration(t, ctx, testAddr1, testPrivKey1, "agent", ""))

	_, err := ms.UnfreezeAccount(ctx, &types.MsgUnfreezeAccount{
		Authority: testAuthority,
		Address:   testAddr1,
	})
	if err == nil {
		t.Fatal("expected error for unfreezing non-frozen account")
	}
}

// ---------- ValidateBasic Tests for Msg Types ----------

func TestMsgFreezeAccount_ValidateBasic(t *testing.T) {
	msg := types.MsgFreezeAccount{
		Sender:  testAddr1,
		Address: testAddr1,
		Reason:  "test",
	}
	if err := msg.ValidateBasic(); err != nil {
		t.Errorf("expected valid, got error: %v", err)
	}

	msg.Sender = ""
	if err := msg.ValidateBasic(); err == nil {
		t.Error("expected error for empty sender")
	}

	msg.Sender = testAddr1
	msg.Address = ""
	if err := msg.ValidateBasic(); err == nil {
		t.Error("expected error for empty address")
	}
}

func TestMsgUnfreezeAccount_ValidateBasic(t *testing.T) {
	msg := types.MsgUnfreezeAccount{
		Authority: testAddr2,
		Address:   testAddr1,
	}
	if err := msg.ValidateBasic(); err != nil {
		t.Errorf("expected valid, got error: %v", err)
	}

	msg.Authority = ""
	if err := msg.ValidateBasic(); err == nil {
		t.Error("expected error for empty authority")
	}
}

// ---------- UpdateParams Tests ----------

func TestUpdateParams_AuthoritySuccess(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	newParams := types.Params{
		KeyRotationCooldown: 222,
		MaxMetadataLength:   2048,
		RequireDid:          true,
	}

	_, err := ms.UpdateParams(ctx, &types.MsgUpdateParams{
		Authority: testAuthority,
		Params:    &newParams,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	stored := k.GetParams(ctx)
	if stored.KeyRotationCooldown != 222 {
		t.Errorf("expected KeyRotationCooldown 222, got %d", stored.KeyRotationCooldown)
	}
	if stored.MaxMetadataLength != 2048 {
		t.Errorf("expected MaxMetadataLength 2048, got %d", stored.MaxMetadataLength)
	}
}

func TestUpdateParams_NonAuthorityRejected(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	defaultParams := types.DefaultParams()
	_, err := ms.UpdateParams(ctx, &types.MsgUpdateParams{
		Authority: testAddr1,
		Params:    &defaultParams,
	})
	if err == nil {
		t.Fatal("expected error for non-authority UpdateParams")
	}
}

func TestUpdateParams_InvalidParamsRejected(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	invalidParams := types.Params{
		MaxMetadataLength: 0,
	}
	_, err := ms.UpdateParams(ctx, &types.MsgUpdateParams{
		Authority: testAuthority,
		Params:    &invalidParams,
	})
	if err == nil {
		t.Fatal("expected error for invalid params")
	}
}

func TestMsgUpdateParams_ValidateBasic(t *testing.T) {
	dp := types.DefaultParams()
	msg := types.MsgUpdateParams{
		Authority: testAddr1,
		Params:    &dp,
	}
	if err := msg.ValidateBasic(); err != nil {
		t.Errorf("expected valid, got error: %v", err)
	}

	msg.Authority = "invalid"
	if err := msg.ValidateBasic(); err == nil {
		t.Error("expected error for invalid authority")
	}
}

// ---------- Invariant Tests ----------

func TestAccountDIDParityInvariant_Passes(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	_, _ = ms.RegisterAccount(ctx, signedRegistration(t, ctx, testAddr1, testPrivKey1, "agent", ""))

	inv := keeper.AccountDIDParityInvariant(k)
	msg, broken := inv(ctx)
	if broken {
		t.Errorf("invariant should pass: %s", msg)
	}
}

func TestAccountDIDParityInvariant_DetectsOrphanedAccount(t *testing.T) {
	k, ctx := setupKeeper(t)

	account := types.Account{
		Address:     testAddr1,
		Did:         testDID1,
		PublicKey:   testPubKey1,
		AccountType: "agent",
	}
	k.SetAccount(ctx, &account)

	inv := keeper.AccountDIDParityInvariant(k)
	_, broken := inv(ctx)
	if !broken {
		t.Error("invariant should detect orphaned account without DID mapping")
	}
}

func TestParamsValidInvariant_Passes(t *testing.T) {
	k, ctx := setupKeeper(t)

	inv := keeper.ParamsValidInvariant(k)
	msg, broken := inv(ctx)
	if broken {
		t.Errorf("invariant should pass with default params: %s", msg)
	}
}

// ---------- Metadata Tests (Zerone-specific) ----------

func TestRegisterAccount_WithMetadata(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	_, err := ms.RegisterAccount(ctx, signedRegistration(
		t, ctx, testAddr1, testPrivKey1, "agent", `{"name":"TestAgent","version":"1.0"}`,
	))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	account, found := k.GetAccount(ctx, testAddr1)
	if !found {
		t.Fatal("account not found")
	}
	if account.Metadata != `{"name":"TestAgent","version":"1.0"}` {
		t.Errorf("expected metadata preserved, got %s", account.Metadata)
	}
}

func TestRegisterAccount_RejectsInvalidIdentityProofWithoutMutation(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)
	msg := signedRegistration(t, ctx, testAddr1, testPrivKey1, "agent", `{"name":"bound"}`)
	msg.IdentityProofSignature[0] ^= 0xff

	_, err := ms.RegisterAccount(ctx, msg)
	if !errors.Is(err, types.ErrInvalidIdentityProof) {
		t.Fatalf("expected invalid identity proof, got %v", err)
	}
	if _, found := k.GetAccount(ctx, testAddr1); found {
		t.Fatal("invalid identity proof wrote account state")
	}
	if _, found := k.GetDIDMapping(ctx, testDID1); found {
		t.Fatal("invalid identity proof wrote DID mapping state")
	}
}

func TestRegisterAccount_IdentityProofBindsChainID(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)
	msg := signedRegistration(t, ctx, testAddr1, testPrivKey1, "agent", "")

	_, err := ms.RegisterAccount(ctx.WithChainID("zerone-other-1"), msg)
	if !errors.Is(err, types.ErrInvalidIdentityProof) {
		t.Fatalf("expected cross-chain identity proof rejection, got %v", err)
	}
	if _, found := k.GetAccount(ctx, testAddr1); found {
		t.Fatal("cross-chain identity proof wrote account state")
	}
}

func TestRegisterAccount_EnforcesMetadataLimit(t *testing.T) {
	k, ctx := setupKeeper(t)
	params := types.DefaultParams()
	params.MaxMetadataLength = 3
	if err := k.SetParams(ctx, &params); err != nil {
		t.Fatal(err)
	}
	ms := keeper.NewMsgServerImpl(k)
	msg := signedRegistration(t, ctx, testAddr1, testPrivKey1, "agent", "four")

	if _, err := ms.RegisterAccount(ctx, msg); err == nil {
		t.Fatal("expected oversized metadata rejection")
	}
	if _, found := k.GetAccount(ctx, testAddr1); found {
		t.Fatal("oversized metadata registration wrote account state")
	}
}

func TestRotateKey_RejectsInvalidNewKeyConfirmationWithoutMutation(t *testing.T) {
	k, ctx := setupRotationKeeper(t)
	ms := keeper.NewMsgServerImpl(k)
	if _, err := ms.RegisterAccount(ctx, signedRegistration(t, ctx, testAddr1, testPrivKey1, "agent", "")); err != nil {
		t.Fatal(err)
	}
	newPrivateKey := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{0x91}, ed25519.SeedSize))
	msg := signedRotation(t, k, ctx, testAddr1, testPrivKey1, newPrivateKey, ctx.BlockTime().Add(time.Minute))
	msg.NewKeyConfirmationSignature[0] ^= 0xff

	_, err := ms.RotateKey(ctx, msg)
	if !errors.Is(err, types.ErrInvalidNewKeyProof) {
		t.Fatalf("expected invalid new-key proof, got %v", err)
	}
	account, found := k.GetAccount(ctx, testAddr1)
	if !found {
		t.Fatal("account disappeared after rejected rotation")
	}
	if account.OperationalKeyVersion != 1 || account.OperationalPublicKey != testPubKey1 {
		t.Fatal("invalid new-key proof mutated operational-key state")
	}
	if got := k.GetLastRotation(ctx, testAddr1); got != 0 {
		t.Fatalf("invalid new-key proof wrote cooldown anchor %d", got)
	}
}

func TestRotateKey_CooldownArithmeticCannotWrap(t *testing.T) {
	k, ctx := setupRotationKeeper(t)
	ms := keeper.NewMsgServerImpl(k)
	if _, err := ms.RegisterAccount(ctx, signedRegistration(t, ctx, testAddr1, testPrivKey1, "agent", "")); err != nil {
		t.Fatal(err)
	}
	k.SetLastRotation(ctx, testAddr1, math.MaxUint64-5)
	newPrivateKey := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{0x92}, ed25519.SeedSize))
	msg := signedRotation(t, k, ctx, testAddr1, testPrivKey1, newPrivateKey, ctx.BlockTime().Add(time.Minute))

	_, err := ms.RotateKey(ctx, msg)
	if !errors.Is(err, types.ErrKeyRotationCooldown) {
		t.Fatalf("expected corrupt/future cooldown anchor rejection, got %v", err)
	}
	account, _ := k.GetAccount(ctx, testAddr1)
	if account.OperationalKeyVersion != 1 {
		t.Fatal("cooldown arithmetic wrap mutated the account")
	}
}

func TestUpdateParams_RejectsMetadataLimitBelowExistingState(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)
	if _, err := ms.RegisterAccount(ctx, signedRegistration(t, ctx, testAddr1, testPrivKey1, "agent", "four")); err != nil {
		t.Fatal(err)
	}
	params := types.DefaultParams()
	params.MaxMetadataLength = 3

	if _, err := ms.UpdateParams(ctx, &types.MsgUpdateParams{Authority: testAuthority, Params: &params}); err == nil {
		t.Fatal("expected metadata limit reduction to be rejected")
	}
	if got := k.GetParams(ctx).MaxMetadataLength; got != types.DefaultParams().MaxMetadataLength {
		t.Fatalf("rejected params update changed max metadata length to %d", got)
	}
}

func TestGenesisRoundTripPreservesRotationCooldown(t *testing.T) {
	sourceKeeper, sourceCtx := setupRotationKeeper(t)
	ms := keeper.NewMsgServerImpl(sourceKeeper)
	if _, err := ms.RegisterAccount(sourceCtx, signedRegistration(t, sourceCtx, testAddr1, testPrivKey1, "agent", "")); err != nil {
		t.Fatal(err)
	}
	secondPrivateKey := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{0x93}, ed25519.SeedSize))
	if _, err := ms.RotateKey(sourceCtx, signedRotation(
		t, sourceKeeper, sourceCtx, testAddr1, testPrivKey1, secondPrivateKey,
		sourceCtx.BlockTime().Add(time.Minute),
	)); err != nil {
		t.Fatal(err)
	}

	exported := sourceKeeper.ExportGenesis(sourceCtx)
	if len(exported.LastKeyRotations) != 1 || exported.LastKeyRotations[0].Height != uint64(sourceCtx.BlockHeight()) {
		t.Fatalf("rotation cooldown anchor missing from export: %+v", exported.LastKeyRotations)
	}
	if err := exported.Validate(); err != nil {
		t.Fatalf("exported genesis is invalid: %v", err)
	}

	restoredKeeper, restoredCtx := setupRotationKeeper(t)
	if err := restoredKeeper.InitGenesis(restoredCtx, exported); err != nil {
		t.Fatalf("restore genesis: %v", err)
	}
	if got := restoredKeeper.GetLastRotation(restoredCtx, testAddr1); got != uint64(sourceCtx.BlockHeight()) {
		t.Fatalf("restored cooldown anchor = %d, want %d", got, sourceCtx.BlockHeight())
	}

	params := restoredKeeper.GetParams(restoredCtx)
	thirdPrivateKey := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{0x94}, ed25519.SeedSize))
	beforeBoundary := restoredCtx.WithBlockHeight(sourceCtx.BlockHeight() + int64(params.KeyRotationCooldown) - 1)
	thirdRotation := signedRotation(
		t, restoredKeeper, beforeBoundary, testAddr1, secondPrivateKey, thirdPrivateKey,
		beforeBoundary.BlockTime().Add(time.Minute),
	)
	restoredMS := keeper.NewMsgServerImpl(restoredKeeper)
	if _, err := restoredMS.RotateKey(beforeBoundary, thirdRotation); !errors.Is(err, types.ErrKeyRotationCooldown) {
		t.Fatalf("expected restored cooldown rejection, got %v", err)
	}

	atBoundary := restoredCtx.WithBlockHeight(sourceCtx.BlockHeight() + int64(params.KeyRotationCooldown))
	if _, err := restoredMS.RotateKey(atBoundary, thirdRotation); err != nil {
		t.Fatalf("rotation at restored cooldown boundary failed: %v", err)
	}
}

func TestInitGenesisValidatesBeforeWriting(t *testing.T) {
	k, ctx := setupKeeper(t)
	original := k.GetParams(ctx)
	invalid := types.DefaultGenesis()
	invalid.Params = nil
	if err := k.InitGenesis(ctx, invalid); err == nil {
		t.Fatal("invalid genesis was accepted")
	}
	if got := k.GetParams(ctx); got.MaxMetadataLength != original.MaxMetadataLength || got.KeyRotationCooldown != original.KeyRotationCooldown {
		t.Fatal("invalid genesis mutated params before validation")
	}
}

func TestInitGenesisRejectsLegacyExceptionEvenOnZerone1(t *testing.T) {
	k, ctx := setupKeeper(t)
	invalid := types.DefaultGenesis()
	identityKey := testPrivKey1.Public().(ed25519.PublicKey)
	invalid.Accounts = []*types.Account{{
		Address:               testAddr1,
		Did:                   testDID1,
		PublicKey:             testPubKey1,
		AccountType:           "agent",
		OperationalKeyHash:    "",
		OperationalPublicKey:  testPubKey1,
		OperationalKeyVersion: 1,
		ReputationScore:       500_000,
		CreatedAtBlock:        1,
		LastActiveBlock:       1,
		Flags:                 &types.AccountFlags{CanSubmitClaims: true, CanChallenge: true},
	}}
	invalid.DidMappings = []*types.DIDMapping{{
		Did:    testDID1,
		Bech32: testAddr1,
		PubKey: hex.EncodeToString(identityKey),
	}}

	if err := k.InitGenesis(ctx.WithChainID("zerone-1"), invalid); err == nil {
		t.Fatal("fresh zerone-1 InitGenesis accepted the export-only legacy exception")
	}
	if _, found := k.GetAccount(ctx, testAddr1); found {
		t.Fatal("rejected legacy genesis wrote account state")
	}
}
