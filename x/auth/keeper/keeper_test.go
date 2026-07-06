package keeper_test

import (
	"context"
	"testing"

	"cosmossdk.io/log"
	"cosmossdk.io/store"
	storemetrics "cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"

	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"

	"github.com/zerone-chain/zerone/x/auth/keeper"
	"github.com/zerone-chain/zerone/x/auth/types"
)

// ---------- Mock Keepers ----------

type mockCosmosAccountKeeper struct{}

func (m mockCosmosAccountKeeper) GetAccount(_ context.Context, _ sdk.AccAddress) sdk.AccountI {
	return nil
}
func (m mockCosmosAccountKeeper) SetAccount(_ context.Context, _ sdk.AccountI) {}
func (m mockCosmosAccountKeeper) NewAccountWithAddress(_ context.Context, _ sdk.AccAddress) sdk.AccountI {
	return nil
}

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
		mockCosmosAccountKeeper{},
		"authority",
	)

	ctx := sdk.NewContext(stateStore, cmtproto.Header{Height: 100}, false, log.NewNopLogger())

	// Set default params
	defaultParams := types.DefaultParams()
	if err := k.SetParams(ctx, &defaultParams); err != nil {
		t.Fatalf("failed to set params: %v", err)
	}

	return k, ctx
}

const (
	testAddr1 = "zrn1m037n75vk2jhdr56y2ptzjjj02uljwnqwwzr7z"
	testAddr2 = "zrn1ur4eyeuuhrkfpcyhykfjsasftv9hn33smszt58"
	// DIDs must derive from their corresponding public keys (first 32 hex chars)
	testPubKey1 = "abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789"
	testDID1    = "did:zrn:abcdef0123456789abcdef0123456789" // pubKey1[:32]
	testPubKey2 = "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"
	testDID2    = "did:zrn:1234567890abcdef1234567890abcdef" // pubKey2[:32]
)

// ---------- RegisterAccount Tests ----------

func TestRegisterAccount_Success(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	resp, err := ms.RegisterAccount(ctx, &types.MsgRegisterAccount{
		Sender:             testAddr1,
		Did:                testDID1,
		PublicKey:          testPubKey1,
		AccountType:        "agent",
		OperationalKeyHash: "opkey1hash",
	})
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
	if account.OperationalKeyHash != "opkey1hash" {
		t.Errorf("expected opkey hash opkey1hash, got %s", account.OperationalKeyHash)
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

	_, err := ms.RegisterAccount(ctx, &types.MsgRegisterAccount{
		Sender:      testAddr1,
		Did:         testDID1,
		PublicKey:   testPubKey1,
		AccountType: "agent",
	})
	if err != nil {
		t.Fatalf("unexpected error on first registration: %v", err)
	}

	_, err = ms.RegisterAccount(ctx, &types.MsgRegisterAccount{
		Sender:      testAddr1,
		Did:         testDID2,
		PublicKey:   testPubKey2,
		AccountType: "human",
	})
	if err == nil {
		t.Fatal("expected error for duplicate address")
	}
}

func TestRegisterAccount_DuplicateDID(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	_, err := ms.RegisterAccount(ctx, &types.MsgRegisterAccount{
		Sender:      testAddr1,
		Did:         testDID1,
		PublicKey:   testPubKey1,
		AccountType: "agent",
	})
	if err != nil {
		t.Fatalf("unexpected error on first registration: %v", err)
	}

	_, err = ms.RegisterAccount(ctx, &types.MsgRegisterAccount{
		Sender:      testAddr2,
		Did:         testDID1,
		PublicKey:   testPubKey2,
		AccountType: "human",
	})
	if err == nil {
		t.Fatal("expected error for duplicate DID")
	}
}

// ---------- RotateKey Tests ----------

func TestRotateKey_Success(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	_, err := ms.RegisterAccount(ctx, &types.MsgRegisterAccount{
		Sender:             testAddr1,
		Did:                testDID1,
		PublicKey:          testPubKey1,
		AccountType:        "agent",
		OperationalKeyHash: "oldkey",
	})
	if err != nil {
		t.Fatalf("failed to register account: %v", err)
	}

	_, err = ms.RotateKey(ctx, &types.MsgRotateKey{
		Sender:                 testAddr1,
		NewOperationalKey:      []byte("newkey"),
		AuthorizationSignature: []byte("sig"),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	account, _ := k.GetAccount(ctx, testAddr1)
	expectedHex := "6e65776b6579" // hex("newkey")
	if account.OperationalPublicKey != expectedHex {
		t.Errorf("expected OperationalPublicKey %s, got %s", expectedHex, account.OperationalPublicKey)
	}
	if account.OperationalKeyVersion != 2 {
		t.Errorf("expected version 2, got %d", account.OperationalKeyVersion)
	}
}

func TestRotateKey_NotFound(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	_, err := ms.RotateKey(ctx, &types.MsgRotateKey{
		Sender:                 testAddr1,
		NewOperationalKey:      []byte("newkey"),
		AuthorizationSignature: []byte("sig"),
	})
	if err == nil {
		t.Fatal("expected error for non-existent account")
	}
}

func TestRotateKey_Cooldown(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	_, _ = ms.RegisterAccount(ctx, &types.MsgRegisterAccount{
		Sender:      testAddr1,
		Did:         testDID1,
		PublicKey:   testPubKey1,
		AccountType: "agent",
	})

	_, err := ms.RotateKey(ctx, &types.MsgRotateKey{
		Sender:                 testAddr1,
		NewOperationalKey:      []byte("key2"),
		AuthorizationSignature: []byte("sig"),
	})
	if err != nil {
		t.Fatalf("first rotation failed: %v", err)
	}

	// Immediate second rotation should fail (cooldown)
	_, err = ms.RotateKey(ctx, &types.MsgRotateKey{
		Sender:                 testAddr1,
		NewOperationalKey:      []byte("key3"),
		AuthorizationSignature: []byte("sig"),
	})
	if err == nil {
		t.Fatal("expected cooldown error")
	}

	// After cooldown passes
	params := k.GetParams(ctx)
	newCtx := ctx.WithBlockHeight(int64(100 + params.KeyRotationCooldown + 1))
	_, err = ms.RotateKey(newCtx, &types.MsgRotateKey{
		Sender:                 testAddr1,
		NewOperationalKey:      []byte("key3"),
		AuthorizationSignature: []byte("sig"),
	})
	if err != nil {
		t.Fatalf("rotation after cooldown should succeed: %v", err)
	}
}

func TestRotateKey_FrozenAccount(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	_, _ = ms.RegisterAccount(ctx, &types.MsgRegisterAccount{
		Sender:      testAddr1,
		Did:         testDID1,
		PublicKey:   testPubKey1,
		AccountType: "agent",
	})
	account, _ := k.GetAccount(ctx, testAddr1)
	account.Flags.Frozen = true
	k.SetAccount(ctx, account)

	_, err := ms.RotateKey(ctx, &types.MsgRotateKey{
		Sender:                 testAddr1,
		NewOperationalKey:      []byte("newkey"),
		AuthorizationSignature: []byte("sig"),
	})
	if err == nil {
		t.Fatal("expected error for frozen account")
	}
}

func TestRotateKey_InvalidKeyType(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	_, _ = ms.RegisterAccount(ctx, &types.MsgRegisterAccount{
		Sender:      testAddr1,
		Did:         testDID1,
		PublicKey:   testPubKey1,
		AccountType: "agent",
	})

	_, err := ms.RotateKey(ctx, &types.MsgRotateKey{
		Sender:                 testAddr1,
		NewOperationalKey:      nil,
		AuthorizationSignature: []byte("sig"),
	})
	if err == nil {
		t.Fatal("expected error for empty operational key")
	}
}






// ---------- Query Tests ----------

func TestQueryAccount(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)
	qs := keeper.NewQueryServerImpl(k)

	_, _ = ms.RegisterAccount(ctx, &types.MsgRegisterAccount{
		Sender:      testAddr1,
		Did:         testDID1,
		PublicKey:   testPubKey1,
		AccountType: "agent",
	})

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

	_, _ = ms.RegisterAccount(ctx, &types.MsgRegisterAccount{
		Sender:      testAddr1,
		Did:         testDID1,
		PublicKey:   testPubKey1,
		AccountType: "human",
	})

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

func TestQueryAccount_NotFound(t *testing.T) {
	k, ctx := setupKeeper(t)
	qs := keeper.NewQueryServerImpl(k)

	_, err := qs.Account(ctx, &types.QueryAccountRequest{Address: testAddr1})
	if err == nil {
		t.Fatal("expected error for non-existent account")
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

	_, _ = ms.RegisterAccount(ctx, &types.MsgRegisterAccount{
		Sender:      testAddr1,
		Did:         testDID1,
		PublicKey:   testPubKey1,
		AccountType: "agent",
	})
	_, _ = ms.RegisterAccount(ctx, &types.MsgRegisterAccount{
		Sender:      testAddr2,
		Did:         testDID2,
		PublicKey:   testPubKey2,
		AccountType: "human",
	})

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

	_, _ = ms.RegisterAccount(ctx, &types.MsgRegisterAccount{
		Sender:      testAddr1,
		Did:         testDID1,
		PublicKey:   testPubKey1,
		AccountType: "agent",
	})

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

	_, _ = ms.RegisterAccount(ctx, &types.MsgRegisterAccount{
		Sender:      testAddr1,
		Did:         testDID1,
		PublicKey:   testPubKey1,
		AccountType: "agent",
	})
	_, _ = ms.RegisterAccount(ctx, &types.MsgRegisterAccount{
		Sender:      testAddr2,
		Did:         testDID2,
		PublicKey:   testPubKey2,
		AccountType: "human",
	})

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

// ---------- ValidateBasic Tests ----------

func TestMsgRegisterAccount_ValidateBasic(t *testing.T) {
	msg := types.MsgRegisterAccount{
		Sender:      testAddr1,
		Did:         testDID1,
		PublicKey:   testPubKey1,
		AccountType: "agent",
	}
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

// ---------- Phase 2: OperationalPublicKey + Ed25519 Sync Tests ----------

func TestRegisterAccount_SetsOperationalPublicKey(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	_, err := ms.RegisterAccount(ctx, &types.MsgRegisterAccount{
		Sender:             testAddr1,
		Did:                testDID1,
		PublicKey:          testPubKey1,
		AccountType:        "agent",
		OperationalKeyHash: "opkey1hash",
	})
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
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	_, _ = ms.RegisterAccount(ctx, &types.MsgRegisterAccount{
		Sender:             testAddr1,
		Did:                testDID1,
		PublicKey:          testPubKey1,
		AccountType:        "agent",
		OperationalKeyHash: "oldkey",
	})

	_, err := ms.RotateKey(ctx, &types.MsgRotateKey{
		Sender:                 testAddr1,
		NewOperationalKey:      []byte("newkey"),
		AuthorizationSignature: []byte("sig"),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	account, _ := k.GetAccount(ctx, testAddr1)
	expectedHex := "6e65776b6579" // hex("newkey")
	if account.OperationalPublicKey != expectedHex {
		t.Errorf("expected OperationalPublicKey %s, got %s", expectedHex, account.OperationalPublicKey)
	}
	if account.OperationalKeyVersion != 2 {
		t.Errorf("expected version 2, got %d", account.OperationalKeyVersion)
	}
}

func TestRotateKey_WithoutNewPublicKey_PreservesExisting(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	_, _ = ms.RegisterAccount(ctx, &types.MsgRegisterAccount{
		Sender:             testAddr1,
		Did:                testDID1,
		PublicKey:          testPubKey1,
		AccountType:        "agent",
		OperationalKeyHash: "oldkey",
	})

	_, err := ms.RotateKey(ctx, &types.MsgRotateKey{
		Sender:                 testAddr1,
		NewOperationalKey:      nil,
		AuthorizationSignature: []byte("sig"),
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

	_, _ = ms.RegisterAccount(ctx, &types.MsgRegisterAccount{
		Sender:             testAddr1,
		Did:                testDID1,
		PublicKey:          testPubKey1,
		AccountType:        "agent",
		OperationalKeyHash: "opkey1hash",
	})

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
	if account.OperationalKeyHash != "opkey1hash" {
		t.Errorf("expected opkey hash opkey1hash, got %s", account.OperationalKeyHash)
	}
}

// ---------- ValidateBasic Phase 2-4 Tests ----------

func TestMsgRotateKey_ValidateBasic(t *testing.T) {
	msg := types.MsgRotateKey{
		Sender:                 testAddr1,
		NewOperationalKey:      []byte("newhash"),
		AuthorizationSignature: []byte("sig"),
	}
	if err := msg.ValidateBasic(); err != nil {
		t.Errorf("expected valid, got error: %v", err)
	}

	msg.Sender = "invalid"
	if err := msg.ValidateBasic(); err == nil {
		t.Error("expected error for invalid sender")
	}
}



// ---------- LastActiveBlock Update Tests ----------

func TestRegisterAccount_SetsCreatedAndLastActive(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	_, _ = ms.RegisterAccount(ctx, &types.MsgRegisterAccount{
		Sender:      testAddr1,
		Did:         testDID1,
		PublicKey:   testPubKey1,
		AccountType: "agent",
	})

	account, _ := k.GetAccount(ctx, testAddr1)
	if account.CreatedAtBlock != 100 {
		t.Errorf("expected CreatedAtBlock 100, got %d", account.CreatedAtBlock)
	}
	if account.LastActiveBlock != 100 {
		t.Errorf("expected LastActiveBlock 100, got %d", account.LastActiveBlock)
	}
}

func TestRotateKey_UpdatesLastActive(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	_, _ = ms.RegisterAccount(ctx, &types.MsgRegisterAccount{
		Sender:      testAddr1,
		Did:         testDID1,
		PublicKey:   testPubKey1,
		AccountType: "agent",
	})

	params := k.GetParams(ctx)
	advancedCtx := ctx.WithBlockHeight(int64(100 + params.KeyRotationCooldown + 1))

	_, err := ms.RotateKey(advancedCtx, &types.MsgRotateKey{
		Sender:                 testAddr1,
		NewOperationalKey:      []byte("newkey"),
		AuthorizationSignature: []byte("sig"),
	})
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

	_, _ = ms.RegisterAccount(ctx, &types.MsgRegisterAccount{
		Sender:      testAddr1,
		Did:         testDID1,
		PublicKey:   testPubKey1,
		AccountType: "agent",
	})

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

	_, _ = ms.RegisterAccount(ctx, &types.MsgRegisterAccount{
		Sender:      testAddr1,
		Did:         testDID1,
		PublicKey:   testPubKey1,
		AccountType: "agent",
	})

	_, err := ms.FreezeAccount(ctx, &types.MsgFreezeAccount{
		Sender:  "authority",
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

	_, _ = ms.RegisterAccount(ctx, &types.MsgRegisterAccount{
		Sender:      testAddr1,
		Did:         testDID1,
		PublicKey:   testPubKey1,
		AccountType: "agent",
	})

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

	_, _ = ms.RegisterAccount(ctx, &types.MsgRegisterAccount{
		Sender:      testAddr1,
		Did:         testDID1,
		PublicKey:   testPubKey1,
		AccountType: "agent",
	})

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

	_, _ = ms.RegisterAccount(ctx, &types.MsgRegisterAccount{
		Sender:      testAddr1,
		Did:         testDID1,
		PublicKey:   testPubKey1,
		AccountType: "agent",
	})

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

	_, _ = ms.RegisterAccount(ctx, &types.MsgRegisterAccount{
		Sender:      testAddr1,
		Did:         testDID1,
		PublicKey:   testPubKey1,
		AccountType: "agent",
	})
	_, _ = ms.FreezeAccount(ctx, &types.MsgFreezeAccount{
		Sender:  testAddr1,
		Address: testAddr1,
		Reason:  "test",
	})

	_, err := ms.UnfreezeAccount(ctx, &types.MsgUnfreezeAccount{
		Authority: "authority",
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

	_, _ = ms.RegisterAccount(ctx, &types.MsgRegisterAccount{
		Sender:      testAddr1,
		Did:         testDID1,
		PublicKey:   testPubKey1,
		AccountType: "agent",
	})
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

	_, _ = ms.RegisterAccount(ctx, &types.MsgRegisterAccount{
		Sender:      testAddr1,
		Did:         testDID1,
		PublicKey:   testPubKey1,
		AccountType: "agent",
	})

	_, err := ms.UnfreezeAccount(ctx, &types.MsgUnfreezeAccount{
		Authority: "authority",
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
		Authority: "authority",
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
		Authority: "authority",
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

	_, _ = ms.RegisterAccount(ctx, &types.MsgRegisterAccount{
		Sender:      testAddr1,
		Did:         testDID1,
		PublicKey:   testPubKey1,
		AccountType: "agent",
	})

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

	_, err := ms.RegisterAccount(ctx, &types.MsgRegisterAccount{
		Sender:      testAddr1,
		Did:         testDID1,
		PublicKey:   testPubKey1,
		AccountType: "agent",
		Metadata:    `{"name":"TestAgent","version":"1.0"}`,
	})
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
