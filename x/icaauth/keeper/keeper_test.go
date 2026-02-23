package keeper_test

import (
	"fmt"
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

	capabilitytypes "github.com/cosmos/ibc-go/modules/capability/types"
	icatypes "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/types"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/zerone-chain/zerone/x/icaauth/keeper"
	"github.com/zerone-chain/zerone/x/icaauth/types"
)

func init() {
	config := sdk.GetConfig()
	config.SetBech32PrefixForAccount("zrn", "zrnpub")
	config.SetBech32PrefixForValidator("zrnvaloper", "zrnvaloperpub")
	config.SetBech32PrefixForConsensusNode("zrnvalcons", "zrnvalconspub")
}

// ---- Mock ICA Controller ----

type mockICAController struct {
	accounts     map[string]string // connectionID/owner -> address
	openChannels map[string]bool   // connectionID/portID -> open
	registerErr  error
	sendErr      error
}

func newMockICAController() *mockICAController {
	return &mockICAController{
		accounts:     make(map[string]string),
		openChannels: make(map[string]bool),
	}
}

func (m *mockICAController) RegisterInterchainAccount(ctx sdk.Context, connectionID, owner, version string) error {
	if m.registerErr != nil {
		return m.registerErr
	}
	m.accounts[connectionID+"/"+owner] = ""
	return nil
}

func (m *mockICAController) GetInterchainAccountAddress(ctx sdk.Context, connectionID, portID string) (string, bool) {
	addr, ok := m.accounts[connectionID+"/"+portID]
	return addr, ok
}

func (m *mockICAController) GetOpenActiveChannel(ctx sdk.Context, connectionID, portID string) (string, bool) {
	open, ok := m.openChannels[connectionID+"/"+portID]
	return "channel-0", open && ok
}

func (m *mockICAController) SendTx(ctx sdk.Context, chanCap *capabilitytypes.Capability, connectionID, portID string, icaPacketData icatypes.InterchainAccountPacketData, timeoutTimestamp uint64) (uint64, error) {
	if m.sendErr != nil {
		return 0, m.sendErr
	}
	return 1, nil
}

// ---- Setup ----

func setupKeeper(t *testing.T) (keeper.Keeper, sdk.Context, *mockICAController) {
	t.Helper()

	storeKey := storetypes.NewKVStoreKey(types.StoreKey)
	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), storemetrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	if err := stateStore.LoadLatestVersion(); err != nil {
		t.Fatalf("failed to load latest version: %v", err)
	}

	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)

	mock := newMockICAController()
	authority := sdk.AccAddress([]byte("authority-addr------")).String()

	k := keeper.NewKeeper(cdc, runtime.NewKVStoreService(storeKey), mock, authority)

	ctx := sdk.NewContext(stateStore, cmtproto.Header{Height: 100}, false, log.NewNopLogger())

	return k, ctx, mock
}

func testAddr(i int) string {
	return sdk.AccAddress([]byte(fmt.Sprintf("test-addr-%010d", i))).String()
}

// ---- Tests ----

func TestRecordCRUD(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	owner := testAddr(1)

	// No records initially
	accounts := k.GetRemoteAccounts(ctx, owner)
	if len(accounts) != 0 {
		t.Fatalf("expected 0 accounts, got %d", len(accounts))
	}

	// Add a remote account
	acct := &types.RemoteAccount{
		ConnectionId: "connection-0",
		PortId:       "icacontroller-" + owner,
		OwnerAddress: owner,
		Active:       false,
	}
	k.AddRemoteAccount(ctx, owner, acct)

	// Get by owner
	accounts = k.GetRemoteAccounts(ctx, owner)
	if len(accounts) != 1 {
		t.Fatalf("expected 1 account, got %d", len(accounts))
	}
	if accounts[0].ConnectionId != "connection-0" {
		t.Fatalf("expected connection-0, got %s", accounts[0].ConnectionId)
	}

	// Get by connection
	found, ok := k.GetRemoteAccountByConnection(ctx, owner, "connection-0")
	if !ok {
		t.Fatal("expected to find account by connection")
	}
	if found.PortId != acct.PortId {
		t.Fatalf("expected port %s, got %s", acct.PortId, found.PortId)
	}

	// Not found for different connection
	_, ok = k.GetRemoteAccountByConnection(ctx, owner, "connection-99")
	if ok {
		t.Fatal("should not find account for non-existent connection")
	}
}

func TestUpdateRemoteAccountAddress(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	owner := testAddr(1)

	acct := &types.RemoteAccount{
		ConnectionId: "connection-0",
		OwnerAddress: owner,
		Active:       false,
	}
	k.AddRemoteAccount(ctx, owner, acct)

	k.UpdateRemoteAccountAddress(ctx, owner, "connection-0", "cosmos1remote...")

	found, ok := k.GetRemoteAccountByConnection(ctx, owner, "connection-0")
	if !ok {
		t.Fatal("expected to find account")
	}
	if found.RemoteAddress != "cosmos1remote..." {
		t.Fatalf("expected remote address cosmos1remote..., got %s", found.RemoteAddress)
	}
	if !found.Active {
		t.Fatal("expected account to be active after address update")
	}
}

func TestParamsCRUD(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	// Default params
	params := k.GetParams(ctx)
	if params.MaxRemoteAccountsPerOwner != 5 {
		t.Fatalf("expected default max=5, got %d", params.MaxRemoteAccountsPerOwner)
	}
	if params.MaxMessagesPerTx != 5 {
		t.Fatalf("expected default max_messages=5, got %d", params.MaxMessagesPerTx)
	}

	// Set custom
	k.SetParams(ctx, &types.Params{
		MaxRemoteAccountsPerOwner: 10,
		AllowedHostMsgTypes:       []string{"/cosmos.bank.v1beta1.MsgSend"},
		RegistrationCooldown:      50,
		MaxMessagesPerTx:          3,
	})

	params = k.GetParams(ctx)
	if params.MaxRemoteAccountsPerOwner != 10 {
		t.Fatalf("expected max=10, got %d", params.MaxRemoteAccountsPerOwner)
	}
	if params.RegistrationCooldown != 50 {
		t.Fatalf("expected cooldown=50, got %d", params.RegistrationCooldown)
	}
}

func TestGenesisRoundTrip(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	// Init with some state
	genState := &types.GenesisState{
		Params: &types.Params{
			MaxRemoteAccountsPerOwner: 3,
			AllowedHostMsgTypes:       []string{"/cosmos.bank.v1beta1.MsgSend"},
			RegistrationCooldown:      200,
			MaxMessagesPerTx:          10,
		},
		Records: []*types.InterchainAccountRecord{
			{
				Owner: testAddr(1),
				Accounts: []*types.RemoteAccount{
					{ConnectionId: "connection-0", OwnerAddress: testAddr(1), Active: true},
				},
			},
		},
	}
	k.InitGenesis(ctx, genState)

	exported := k.ExportGenesis(ctx)
	if exported.Params.MaxRemoteAccountsPerOwner != 3 {
		t.Fatalf("expected max=3, got %d", exported.Params.MaxRemoteAccountsPerOwner)
	}
	if len(exported.Records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(exported.Records))
	}
	if exported.Records[0].Owner != testAddr(1) {
		t.Fatalf("expected owner %s, got %s", testAddr(1), exported.Records[0].Owner)
	}
}

func TestMsgRegisterAccount(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	owner := testAddr(1)
	_, err := ms.RegisterAccount(ctx, &types.MsgRegisterAccount{
		Owner:        owner,
		ConnectionId: "connection-0",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	accounts := k.GetRemoteAccounts(ctx, owner)
	if len(accounts) != 1 {
		t.Fatalf("expected 1 account, got %d", len(accounts))
	}
}

func TestMsgRegisterAccountMaxAccounts(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	// Set max to 2
	k.SetParams(ctx, &types.Params{
		MaxRemoteAccountsPerOwner: 2,
		AllowedHostMsgTypes:       []string{},
		RegistrationCooldown:      0, // disable cooldown for this test
		MaxMessagesPerTx:          5,
	})

	owner := testAddr(1)

	_, err := ms.RegisterAccount(ctx, &types.MsgRegisterAccount{Owner: owner, ConnectionId: "connection-0"})
	if err != nil {
		t.Fatalf("first registration should succeed: %v", err)
	}

	_, err = ms.RegisterAccount(ctx, &types.MsgRegisterAccount{Owner: owner, ConnectionId: "connection-1"})
	if err != nil {
		t.Fatalf("second registration should succeed: %v", err)
	}

	_, err = ms.RegisterAccount(ctx, &types.MsgRegisterAccount{Owner: owner, ConnectionId: "connection-2"})
	if err == nil {
		t.Fatal("third registration should fail (max=2)")
	}
}

func TestMsgRegisterAccountDuplicate(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	k.SetParams(ctx, &types.Params{
		MaxRemoteAccountsPerOwner: 10,
		AllowedHostMsgTypes:       []string{},
		RegistrationCooldown:      0,
		MaxMessagesPerTx:          5,
	})

	owner := testAddr(1)

	_, err := ms.RegisterAccount(ctx, &types.MsgRegisterAccount{Owner: owner, ConnectionId: "connection-0"})
	if err != nil {
		t.Fatalf("first registration should succeed: %v", err)
	}

	_, err = ms.RegisterAccount(ctx, &types.MsgRegisterAccount{Owner: owner, ConnectionId: "connection-0"})
	if err == nil {
		t.Fatal("duplicate registration should fail")
	}
}

func TestMsgRegisterAccountCooldown(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	k.SetParams(ctx, &types.Params{
		MaxRemoteAccountsPerOwner: 10,
		AllowedHostMsgTypes:       []string{},
		RegistrationCooldown:      100,
		MaxMessagesPerTx:          5,
	})

	owner := testAddr(1)

	// Register at height 100
	_, err := ms.RegisterAccount(ctx, &types.MsgRegisterAccount{Owner: owner, ConnectionId: "connection-0"})
	if err != nil {
		t.Fatalf("first registration should succeed: %v", err)
	}

	// Try at height 150 (100 + 100 = 200 is the cooldown expiry)
	ctx150 := ctx.WithBlockHeight(150)
	_, err = ms.RegisterAccount(ctx150, &types.MsgRegisterAccount{Owner: owner, ConnectionId: "connection-1"})
	if err == nil {
		t.Fatal("registration during cooldown should fail")
	}

	// Try at height 200 (cooldown expired)
	ctx200 := ctx.WithBlockHeight(200)
	_, err = ms.RegisterAccount(ctx200, &types.MsgRegisterAccount{Owner: owner, ConnectionId: "connection-1"})
	if err != nil {
		t.Fatalf("registration after cooldown should succeed: %v", err)
	}
}

func TestMsgRegisterAccountICAFailure(t *testing.T) {
	k, ctx, mock := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	mock.registerErr = fmt.Errorf("ICA registration failed")

	_, err := ms.RegisterAccount(ctx, &types.MsgRegisterAccount{
		Owner:        testAddr(1),
		ConnectionId: "connection-0",
	})
	if err == nil {
		t.Fatal("should propagate ICA controller error")
	}
}

func TestMsgSubmitTxNotRegistered(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	_, err := ms.SubmitTx(ctx, &types.MsgSubmitTx{
		Owner:        testAddr(1),
		ConnectionId: "connection-0",
		Msgs:         nil,
		TimeoutNs:    uint64(120 * time.Second),
	})
	if err == nil {
		t.Fatal("should fail when not registered")
	}
}

func TestDefaultParamsMsgTransferExcluded(t *testing.T) {
	params := types.DefaultParams()
	for _, msgType := range params.AllowedHostMsgTypes {
		if msgType == "/ibc.applications.transfer.v1.MsgTransfer" {
			t.Fatal("SECURITY: MsgTransfer must NOT be in DefaultParams AllowedHostMsgTypes (P0-6)")
		}
	}
}

func TestTimeoutValidation(t *testing.T) {
	// Zero timeout — rejected
	msg := &types.MsgSubmitTx{
		Owner:        testAddr(1),
		ConnectionId: "connection-0",
		Msgs:         nil,
		TimeoutNs:    0,
	}
	if err := msg.ValidateBasic(); err == nil {
		t.Fatal("timeout_ns=0 should be rejected")
	}

	// Under 60s — rejected
	msg.TimeoutNs = uint64(30 * time.Second)
	msg.Msgs = nil
	if err := msg.ValidateBasic(); err == nil {
		t.Fatal("timeout_ns < 60s should be rejected")
	}

	// Exactly 60s — but msgs is empty so still fails on msgs check
	msg.TimeoutNs = uint64(60 * time.Second)
	msg.Msgs = nil
	err := msg.ValidateBasic()
	if err == nil {
		t.Fatal("empty msgs should be rejected")
	}
	// Make sure the error is about msgs, not timeout
	if err.Error() == "timeout_ns cannot be zero" || err.Error() == fmt.Sprintf("timeout_ns must be at least %d (60s)", types.MinTimeout) {
		t.Fatal("60s timeout should pass validation, error should be about empty msgs")
	}
}

func TestMsgUpdateParams(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	authority := sdk.AccAddress([]byte("authority-addr------")).String()

	// Success
	_, err := ms.UpdateParams(ctx, &types.MsgUpdateParams{
		Authority: authority,
		Params: &types.Params{
			MaxRemoteAccountsPerOwner: 20,
			AllowedHostMsgTypes:       []string{"/cosmos.bank.v1beta1.MsgSend"},
			RegistrationCooldown:      50,
			MaxMessagesPerTx:          10,
		},
	})
	if err != nil {
		t.Fatalf("update params should succeed: %v", err)
	}

	params := k.GetParams(ctx)
	if params.MaxRemoteAccountsPerOwner != 20 {
		t.Fatalf("expected max=20, got %d", params.MaxRemoteAccountsPerOwner)
	}

	// Wrong authority
	_, err = ms.UpdateParams(ctx, &types.MsgUpdateParams{
		Authority: testAddr(99),
		Params: &types.Params{
			MaxRemoteAccountsPerOwner: 1,
			AllowedHostMsgTypes:       nil,
			RegistrationCooldown:      0,
			MaxMessagesPerTx:          1,
		},
	})
	if err == nil {
		t.Fatal("wrong authority should fail")
	}
}

// ---------- R6-6 Security Tests ----------

// setupActiveAccount registers an account, marks it active, and sets up the
// mock ICA controller to report an open channel for it.
func setupActiveAccount(t *testing.T, k keeper.Keeper, ctx sdk.Context, mock *mockICAController, owner string, connectionID string) {
	t.Helper()
	portID := "icacontroller-" + owner
	acct := &types.RemoteAccount{
		ConnectionId:    connectionID,
		PortId:          portID,
		OwnerAddress:    owner,
		Active:          true,
		RegisteredBlock: uint64(ctx.BlockHeight()),
		RemoteAddress:   "cosmos1remote...",
	}
	k.AddRemoteAccount(ctx, owner, acct)
	mock.openChannels[connectionID+"/"+portID] = true
}

func TestSubmitTxMaxMessages(t *testing.T) {
	k, ctx, mock := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	k.SetParams(ctx, &types.Params{
		MaxRemoteAccountsPerOwner: 10,
		AllowedHostMsgTypes:       []string{"/cosmos.bank.v1beta1.MsgSend"},
		RegistrationCooldown:      0,
		MaxMessagesPerTx:          2,
	})

	owner := testAddr(1)
	setupActiveAccount(t, k, ctx, mock, owner, "connection-0")

	// Create 3 messages (exceeds MaxMessagesPerTx=2)
	msgs := make([]*anypb.Any, 3)
	for i := range msgs {
		msgs[i] = &anypb.Any{TypeUrl: "/cosmos.bank.v1beta1.MsgSend", Value: []byte{}}
	}

	_, err := ms.SubmitTx(ctx, &types.MsgSubmitTx{
		Owner:        owner,
		ConnectionId: "connection-0",
		Msgs:         msgs,
		TimeoutNs:    uint64(120 * time.Second),
	})
	if err == nil {
		t.Fatal("should reject tx exceeding MaxMessagesPerTx")
	}
	if !strings.Contains(err.Error(), "max is 2") {
		t.Fatalf("error should mention max limit, got: %v", err)
	}
}

func TestSubmitTxDisallowedMessage(t *testing.T) {
	k, ctx, mock := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	// Only allow MsgSend in global allowlist
	k.SetParams(ctx, &types.Params{
		MaxRemoteAccountsPerOwner: 10,
		AllowedHostMsgTypes:       []string{"/cosmos.bank.v1beta1.MsgSend"},
		RegistrationCooldown:      0,
		MaxMessagesPerTx:          5,
	})

	owner := testAddr(1)
	setupActiveAccount(t, k, ctx, mock, owner, "connection-0")

	// Submit with MsgTransfer (explicitly excluded / not in allowlist)
	msgs := []*anypb.Any{
		{TypeUrl: "/ibc.applications.transfer.v1.MsgTransfer", Value: []byte{}},
	}

	_, err := ms.SubmitTx(ctx, &types.MsgSubmitTx{
		Owner:        owner,
		ConnectionId: "connection-0",
		Msgs:         msgs,
		TimeoutNs:    uint64(120 * time.Second),
	})
	if err == nil {
		t.Fatal("SECURITY: disallowed message type must be rejected")
	}
	if !strings.Contains(err.Error(), "not in global allowlist") {
		t.Fatalf("error should mention global allowlist, got: %v", err)
	}
}

func TestSubmitTxPerAccountAllowlistRestriction(t *testing.T) {
	k, ctx, mock := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	// Global allowlist permits MsgSend + MsgDelegate
	k.SetParams(ctx, &types.Params{
		MaxRemoteAccountsPerOwner: 10,
		AllowedHostMsgTypes: []string{
			"/cosmos.bank.v1beta1.MsgSend",
			"/cosmos.staking.v1beta1.MsgDelegate",
		},
		RegistrationCooldown: 0,
		MaxMessagesPerTx:     5,
	})

	owner := testAddr(1)
	portID := "icacontroller-" + owner
	// Register with per-account allowlist that only permits MsgSend
	acct := &types.RemoteAccount{
		ConnectionId:    "connection-0",
		PortId:          portID,
		OwnerAddress:    owner,
		Active:          true,
		RegisteredBlock: uint64(ctx.BlockHeight()),
		RemoteAddress:   "cosmos1remote...",
		AllowedMsgTypes: []string{"/cosmos.bank.v1beta1.MsgSend"},
	}
	k.AddRemoteAccount(ctx, owner, acct)
	mock.openChannels["connection-0/"+portID] = true

	// MsgDelegate passes global but fails per-account
	msgs := []*anypb.Any{
		{TypeUrl: "/cosmos.staking.v1beta1.MsgDelegate", Value: []byte{}},
	}

	_, err := ms.SubmitTx(ctx, &types.MsgSubmitTx{
		Owner:        owner,
		ConnectionId: "connection-0",
		Msgs:         msgs,
		TimeoutNs:    uint64(120 * time.Second),
	})
	if err == nil {
		t.Fatal("SECURITY: per-account allowlist must restrict beyond global")
	}
	if !strings.Contains(err.Error(), "account allowlist") {
		t.Fatalf("error should mention account allowlist, got: %v", err)
	}
}

func TestSubmitTxInactiveAccount(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	owner := testAddr(1)
	// Register but do NOT activate
	acct := &types.RemoteAccount{
		ConnectionId:    "connection-0",
		PortId:          "icacontroller-" + owner,
		OwnerAddress:    owner,
		Active:          false,
		RegisteredBlock: uint64(ctx.BlockHeight()),
	}
	k.AddRemoteAccount(ctx, owner, acct)

	msgs := []*anypb.Any{
		{TypeUrl: "/cosmos.bank.v1beta1.MsgSend", Value: []byte{}},
	}

	_, err := ms.SubmitTx(ctx, &types.MsgSubmitTx{
		Owner:        owner,
		ConnectionId: "connection-0",
		Msgs:         msgs,
		TimeoutNs:    uint64(120 * time.Second),
	})
	if err == nil {
		t.Fatal("should reject tx on inactive account")
	}
	if !strings.Contains(err.Error(), "not active") {
		t.Fatalf("error should mention inactive, got: %v", err)
	}
}

func TestSubmitTxChannelClosedMarksInactive(t *testing.T) {
	k, ctx, mock := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	owner := testAddr(1)
	portID := "icacontroller-" + owner
	acct := &types.RemoteAccount{
		ConnectionId:    "connection-0",
		PortId:          portID,
		OwnerAddress:    owner,
		Active:          true,
		RegisteredBlock: uint64(ctx.BlockHeight()),
		RemoteAddress:   "cosmos1remote...",
	}
	k.AddRemoteAccount(ctx, owner, acct)
	// NOTE: do NOT add to mock.openChannels — channel is closed

	msgs := []*anypb.Any{
		{TypeUrl: "/cosmos.bank.v1beta1.MsgSend", Value: []byte{}},
	}

	_, err := ms.SubmitTx(ctx, &types.MsgSubmitTx{
		Owner:        owner,
		ConnectionId: "connection-0",
		Msgs:         msgs,
		TimeoutNs:    uint64(120 * time.Second),
	})
	if err == nil {
		t.Fatal("should reject tx when channel is closed")
	}

	// Verify account was marked inactive
	found, ok := k.GetRemoteAccountByConnection(ctx, owner, "connection-0")
	if !ok {
		t.Fatal("account record should still exist")
	}
	if found.Active {
		t.Fatal("account should be marked inactive after closed channel detection")
	}

	// Suppress unused variable warnings for mock
	_ = mock
}
