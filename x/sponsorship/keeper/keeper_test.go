package keeper_test

import (
	"context"
	"errors"
	"testing"

	"cosmossdk.io/log"
	sdkmath "cosmossdk.io/math"
	"cosmossdk.io/store"
	storemetrics "cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"

	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"

	knowledgetypes "github.com/zerone-chain/zerone/x/knowledge/types"
	"github.com/zerone-chain/zerone/x/sponsorship/keeper"
	"github.com/zerone-chain/zerone/x/sponsorship/types"
)

func init() {
	cfg := sdk.GetConfig()
	cfg.SetBech32PrefixForAccount("zrn", "zrnpub")
	cfg.SetBech32PrefixForValidator("zrnvaloper", "zrnvaloperpub")
	cfg.SetBech32PrefixForConsensusNode("zrnvalcons", "zrnvalconspub")
}

// ---------- Mocks ----------

type mockBankKeeper struct {
	balances       map[string]map[string]int64
	moduleBalances map[string]map[string]int64
}

func newMockBank() *mockBankKeeper {
	return &mockBankKeeper{
		balances:       map[string]map[string]int64{},
		moduleBalances: map[string]map[string]int64{},
	}
}

func (m *mockBankKeeper) setBalance(addr, denom string, amount int64) {
	if m.balances[addr] == nil {
		m.balances[addr] = map[string]int64{}
	}
	m.balances[addr][denom] = amount
}

func (m *mockBankKeeper) SpendableCoins(_ context.Context, addr sdk.AccAddress) sdk.Coins {
	out := sdk.Coins{}
	for denom, amt := range m.balances[addr.String()] {
		if amt > 0 {
			out = out.Add(sdk.NewCoin(denom, sdkmath.NewInt(amt)))
		}
	}
	return out
}

func (m *mockBankKeeper) SendCoinsFromAccountToModule(_ context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error {
	for _, coin := range amt {
		if m.balances[senderAddr.String()] == nil || m.balances[senderAddr.String()][coin.Denom] < coin.Amount.Int64() {
			return errors.New("insufficient funds")
		}
		m.balances[senderAddr.String()][coin.Denom] -= coin.Amount.Int64()
		if m.moduleBalances[recipientModule] == nil {
			m.moduleBalances[recipientModule] = map[string]int64{}
		}
		m.moduleBalances[recipientModule][coin.Denom] += coin.Amount.Int64()
	}
	return nil
}

func (m *mockBankKeeper) SendCoinsFromModuleToAccount(_ context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error {
	for _, coin := range amt {
		if m.moduleBalances[senderModule] == nil || m.moduleBalances[senderModule][coin.Denom] < coin.Amount.Int64() {
			return errors.New("insufficient module balance")
		}
		m.moduleBalances[senderModule][coin.Denom] -= coin.Amount.Int64()
		if m.balances[recipientAddr.String()] == nil {
			m.balances[recipientAddr.String()] = map[string]int64{}
		}
		m.balances[recipientAddr.String()][coin.Denom] += coin.Amount.Int64()
	}
	return nil
}

type mockKnowledgeKeeper struct {
	facts map[string]*knowledgetypes.Fact
}

func newMockKnowledge() *mockKnowledgeKeeper {
	return &mockKnowledgeKeeper{facts: map[string]*knowledgetypes.Fact{}}
}

func (m *mockKnowledgeKeeper) GetFact(_ context.Context, id string) (*knowledgetypes.Fact, bool) {
	f, ok := m.facts[id]
	return f, ok
}

// ---------- Setup ----------

func setup(t *testing.T) (keeper.Keeper, sdk.Context, *mockBankKeeper, *mockKnowledgeKeeper) {
	t.Helper()
	storeKey := storetypes.NewKVStoreKey(types.StoreKey)
	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), storemetrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	if err := stateStore.LoadLatestVersion(); err != nil {
		t.Fatalf("load store: %v", err)
	}
	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)
	bk := newMockBank()
	kk := newMockKnowledge()
	storeService := runtime.NewKVStoreService(storeKey)
	k := keeper.NewKeeper(storeService, cdc, bk, kk)
	ctx := sdk.NewContext(stateStore, cmtproto.Header{Height: 1000, ChainID: "zerone-test"}, false, log.NewNopLogger())
	return k, ctx, bk, kk
}

func mkAddr(seed string) sdk.AccAddress {
	b := make([]byte, 20)
	copy(b, []byte(seed))
	return sdk.AccAddress(b)
}

func makeVerifiedFact(t *testing.T, kk *mockKnowledgeKeeper, factID, domain, submitter string, submittedAt uint64) {
	t.Helper()
	kk.facts[factID] = &knowledgetypes.Fact{
		Id:               factID,
		Domain:           domain,
		Submitter:        submitter,
		SubmittedAtBlock: submittedAt,
		Status:           knowledgetypes.FactStatus_FACT_STATUS_VERIFIED,
	}
}

func createTestBounty(t *testing.T, k keeper.Keeper, srv types.MsgServer, ctx sdk.Context, bk *mockBankKeeper, sponsor sdk.AccAddress, domain, price string, target uint32, duration uint64) string {
	t.Helper()
	bk.setBalance(sponsor.String(), "uzrn", 1_000_000_000_000)
	resp, err := srv.CreateBountyOrder(ctx, &types.MsgCreateBountyOrder{
		Sponsor: sponsor.String(), Domain: domain, PricePerArtifact: price,
		TargetCount: target, DurationBlocks: duration,
	})
	if err != nil {
		t.Fatalf("create bounty: %v", err)
	}
	return resp.BountyId
}

// ---------- CreateBountyOrder ----------

func TestCreateBountyOrder_HappyPath(t *testing.T) {
	k, ctx, bk, _ := setup(t)
	srv := keeper.NewMsgServerImpl(k)

	sponsor := mkAddr("sponsor-happy-aaaaa1")
	bk.setBalance(sponsor.String(), "uzrn", 100_000_000)

	resp, err := srv.CreateBountyOrder(ctx, &types.MsgCreateBountyOrder{
		Sponsor:          sponsor.String(),
		Domain:           "mathematics",
		PricePerArtifact: "1000000",
		TargetCount:      10,
		DurationBlocks:   500,
	})
	if err != nil {
		t.Fatalf("CreateBountyOrder: %v", err)
	}
	if resp.BountyId == "" {
		t.Fatal("expected non-empty bounty_id")
	}

	order, found := k.GetBountyOrder(ctx, resp.BountyId)
	if !found {
		t.Fatal("bounty not stored")
	}
	if order.Status != types.BountyStatus_BOUNTY_STATUS_ACTIVE {
		t.Errorf("status: want ACTIVE, got %s", order.Status)
	}
	if order.EscrowRemaining != "10000000" {
		t.Errorf("escrow: want 10000000, got %s", order.EscrowRemaining)
	}

	if bk.balances[sponsor.String()]["uzrn"] != 100_000_000-10_000_000 {
		t.Errorf("sponsor balance: want %d, got %d", 100_000_000-10_000_000, bk.balances[sponsor.String()]["uzrn"])
	}
	if bk.moduleBalances[types.ModuleName]["uzrn"] != 10_000_000 {
		t.Errorf("module balance: want 10000000, got %d", bk.moduleBalances[types.ModuleName]["uzrn"])
	}
}

func TestCreateBountyOrder_InsufficientFunds(t *testing.T) {
	k, ctx, bk, _ := setup(t)
	srv := keeper.NewMsgServerImpl(k)

	sponsor := mkAddr("sponsor-poor-aaaaaa2")
	bk.setBalance(sponsor.String(), "uzrn", 1_000_000)

	_, err := srv.CreateBountyOrder(ctx, &types.MsgCreateBountyOrder{
		Sponsor:          sponsor.String(),
		Domain:           "mathematics",
		PricePerArtifact: "1000000",
		TargetCount:      10,
		DurationBlocks:   500,
	})
	if err == nil {
		t.Fatal("expected ErrInsufficientEscrow")
	}
	if !errors.Is(err, types.ErrInsufficientEscrow) {
		t.Errorf("wrong error: %v", err)
	}
}

func TestCreateBountyOrder_BelowMinTargetCount(t *testing.T) {
	k, ctx, bk, _ := setup(t)
	srv := keeper.NewMsgServerImpl(k)
	k.SetParams(ctx, &types.Params{MinTargetCount: 5, MinDurationBlocks: 100, MaxActiveBountiesPerSponsor: 16})

	sponsor := mkAddr("sponsor-target-aaaaa3")
	bk.setBalance(sponsor.String(), "uzrn", 100_000_000)

	_, err := srv.CreateBountyOrder(ctx, &types.MsgCreateBountyOrder{
		Sponsor: sponsor.String(), Domain: "m", PricePerArtifact: "1000000",
		TargetCount: 1, DurationBlocks: 500,
	})
	if err == nil {
		t.Fatal("expected error for target_count below min")
	}
}

func TestCreateBountyOrder_BelowMinDuration(t *testing.T) {
	k, ctx, bk, _ := setup(t)
	srv := keeper.NewMsgServerImpl(k)

	sponsor := mkAddr("sponsor-dur-aaaaaaaa4")
	bk.setBalance(sponsor.String(), "uzrn", 100_000_000)

	_, err := srv.CreateBountyOrder(ctx, &types.MsgCreateBountyOrder{
		Sponsor: sponsor.String(), Domain: "m", PricePerArtifact: "1000000",
		TargetCount: 1, DurationBlocks: 10,
	})
	if err == nil {
		t.Fatal("expected error for duration below min")
	}
}

func TestCreateBountyOrder_MaxActivePerSponsor(t *testing.T) {
	k, ctx, bk, _ := setup(t)
	srv := keeper.NewMsgServerImpl(k)
	k.SetParams(ctx, &types.Params{MinTargetCount: 1, MinDurationBlocks: 100, MaxActiveBountiesPerSponsor: 2})

	sponsor := mkAddr("sponsor-cap-aaaaaaa5")
	bk.setBalance(sponsor.String(), "uzrn", 100_000_000)
	msg := &types.MsgCreateBountyOrder{
		Sponsor: sponsor.String(), Domain: "m", PricePerArtifact: "1000",
		TargetCount: 1, DurationBlocks: 500,
	}

	if _, err := srv.CreateBountyOrder(ctx, msg); err != nil {
		t.Fatalf("first create: %v", err)
	}
	if _, err := srv.CreateBountyOrder(ctx, msg); err != nil {
		t.Fatalf("second create: %v", err)
	}
	if _, err := srv.CreateBountyOrder(ctx, msg); err == nil {
		t.Fatal("expected error on third active bounty")
	}
}

func TestCreateBountyOrder_EmitsEvent(t *testing.T) {
	k, ctx, bk, _ := setup(t)
	srv := keeper.NewMsgServerImpl(k)

	sponsor := mkAddr("sponsor-event-aaaaaa6")
	bk.setBalance(sponsor.String(), "uzrn", 100_000_000)

	_, err := srv.CreateBountyOrder(ctx, &types.MsgCreateBountyOrder{
		Sponsor: sponsor.String(), Domain: "m", PricePerArtifact: "1000",
		TargetCount: 1, DurationBlocks: 500,
	})
	if err != nil {
		t.Fatalf("CreateBountyOrder: %v", err)
	}

	var found bool
	for _, e := range ctx.EventManager().Events() {
		if e.Type == "zerone.sponsorship.bounty_created" {
			for _, attr := range e.Attributes {
				if attr.Key == "creed_commitment" && attr.Value == "20" {
					found = true
				}
			}
		}
	}
	if !found {
		t.Fatal("expected bounty_created event with creed_commitment=20")
	}
}

// ---------- FulfillBounty ----------

func TestFulfillBounty_HappyPath(t *testing.T) {
	k, ctx, bk, kk := setup(t)
	srv := keeper.NewMsgServerImpl(k)

	sponsor := mkAddr("sponsor-fhappy-aaaa7")
	worker := mkAddr("worker-fhappy-aaaaa1")
	bountyID := createTestBounty(t, k, srv, ctx, bk, sponsor, "math", "1000000", 5, 500)
	makeVerifiedFact(t, kk, "fact-1", "math", worker.String(), 1000)

	caller := mkAddr("caller-aaaaaaaaaaaaaa")
	resp, err := srv.FulfillBounty(ctx, &types.MsgFulfillBounty{
		Caller: caller.String(), BountyId: bountyID, FactId: "fact-1",
	})
	if err != nil {
		t.Fatalf("FulfillBounty: %v", err)
	}
	if resp.Worker != worker.String() {
		t.Errorf("worker: want %s, got %s", worker, resp.Worker)
	}
	if resp.AmountPaid != "1000000" {
		t.Errorf("amount: want 1000000, got %s", resp.AmountPaid)
	}
	if resp.BountyNowFulfilled {
		t.Error("bounty should not be fulfilled after 1 of 5")
	}

	if bk.balances[worker.String()]["uzrn"] != 1_000_000 {
		t.Errorf("worker balance: want 1000000, got %d", bk.balances[worker.String()]["uzrn"])
	}
	if bk.moduleBalances[types.ModuleName]["uzrn"] != 5_000_000-1_000_000 {
		t.Errorf("module balance: want %d, got %d", 5_000_000-1_000_000, bk.moduleBalances[types.ModuleName]["uzrn"])
	}
	order, _ := k.GetBountyOrder(ctx, bountyID)
	if order.FulfilledCount != 1 {
		t.Errorf("fulfilled_count: want 1, got %d", order.FulfilledCount)
	}
	if order.EscrowRemaining != "4000000" {
		t.Errorf("escrow_remaining: want 4000000, got %s", order.EscrowRemaining)
	}
	if _, exists := k.GetFulfillment(ctx, bountyID, "fact-1"); !exists {
		t.Error("fulfillment record missing")
	}
}

func TestFulfillBounty_BountyNotFound(t *testing.T) {
	k, ctx, _, _ := setup(t)
	srv := keeper.NewMsgServerImpl(k)
	caller := mkAddr("caller-nf-aaaaaaaaa1")
	_, err := srv.FulfillBounty(ctx, &types.MsgFulfillBounty{
		Caller: caller.String(), BountyId: "doesnotexist", FactId: "fact-1",
	})
	if err == nil || !errors.Is(err, types.ErrBountyNotFound) {
		t.Fatalf("expected ErrBountyNotFound, got %v", err)
	}
}

func TestFulfillBounty_FactNotFound(t *testing.T) {
	k, ctx, bk, _ := setup(t)
	srv := keeper.NewMsgServerImpl(k)
	sponsor := mkAddr("sponsor-fnf-aaaaaa9")
	bountyID := createTestBounty(t, k, srv, ctx, bk, sponsor, "math", "1000", 5, 500)
	caller := mkAddr("caller-fnf-aaaaaaaa")
	_, err := srv.FulfillBounty(ctx, &types.MsgFulfillBounty{
		Caller: caller.String(), BountyId: bountyID, FactId: "no-such-fact",
	})
	if err == nil || !errors.Is(err, types.ErrFactNotEligible) {
		t.Fatalf("expected ErrFactNotEligible, got %v", err)
	}
}

func TestFulfillBounty_FactNotVerified(t *testing.T) {
	k, ctx, bk, kk := setup(t)
	srv := keeper.NewMsgServerImpl(k)
	sponsor := mkAddr("sponsor-fnv-aaaaaa1")
	worker := mkAddr("worker-fnv-aaaaaaaa1")
	bountyID := createTestBounty(t, k, srv, ctx, bk, sponsor, "math", "1000", 5, 500)
	kk.facts["fact-pending"] = &knowledgetypes.Fact{
		Id: "fact-pending", Domain: "math", Submitter: worker.String(),
		SubmittedAtBlock: 1000, Status: knowledgetypes.FactStatus_FACT_STATUS_PENDING,
	}
	caller := mkAddr("caller-fnv-aaaaaaaa1")
	_, err := srv.FulfillBounty(ctx, &types.MsgFulfillBounty{
		Caller: caller.String(), BountyId: bountyID, FactId: "fact-pending",
	})
	if err == nil || !errors.Is(err, types.ErrFactNotEligible) {
		t.Fatalf("expected ErrFactNotEligible, got %v", err)
	}
}

func TestFulfillBounty_DomainMismatch(t *testing.T) {
	k, ctx, bk, kk := setup(t)
	srv := keeper.NewMsgServerImpl(k)
	sponsor := mkAddr("sponsor-dm-aaaaaaaa1")
	worker := mkAddr("worker-dm-aaaaaaaaa1")
	bountyID := createTestBounty(t, k, srv, ctx, bk, sponsor, "math", "1000", 5, 500)
	makeVerifiedFact(t, kk, "fact-bio", "biology", worker.String(), 1000)
	caller := mkAddr("caller-dm-aaaaaaaaa1")
	_, err := srv.FulfillBounty(ctx, &types.MsgFulfillBounty{
		Caller: caller.String(), BountyId: bountyID, FactId: "fact-bio",
	})
	if err == nil || !errors.Is(err, types.ErrFactNotEligible) {
		t.Fatalf("expected ErrFactNotEligible (domain mismatch), got %v", err)
	}
}

func TestFulfillBounty_RetroactiveRejected(t *testing.T) {
	k, ctx, bk, kk := setup(t)
	srv := keeper.NewMsgServerImpl(k)
	sponsor := mkAddr("sponsor-retro-aaaa1")
	worker := mkAddr("worker-retro-aaaaa1")
	bountyID := createTestBounty(t, k, srv, ctx, bk, sponsor, "math", "1000", 5, 500)
	makeVerifiedFact(t, kk, "fact-retro", "math", worker.String(), 999)
	caller := mkAddr("caller-retro-aaaaa1")
	_, err := srv.FulfillBounty(ctx, &types.MsgFulfillBounty{
		Caller: caller.String(), BountyId: bountyID, FactId: "fact-retro",
	})
	if err == nil || !errors.Is(err, types.ErrFactNotEligible) {
		t.Fatalf("expected ErrFactNotEligible (retroactive), got %v", err)
	}
}

func TestFulfillBounty_DoubleFulfillRejected(t *testing.T) {
	k, ctx, bk, kk := setup(t)
	srv := keeper.NewMsgServerImpl(k)
	sponsor := mkAddr("sponsor-double-aaaa1")
	worker := mkAddr("worker-double-aaaaa1")
	bountyID := createTestBounty(t, k, srv, ctx, bk, sponsor, "math", "1000", 5, 500)
	makeVerifiedFact(t, kk, "fact-1", "math", worker.String(), 1000)
	caller := mkAddr("caller-double-aaaaa1")

	if _, err := srv.FulfillBounty(ctx, &types.MsgFulfillBounty{
		Caller: caller.String(), BountyId: bountyID, FactId: "fact-1",
	}); err != nil {
		t.Fatalf("first fulfill: %v", err)
	}
	if _, err := srv.FulfillBounty(ctx, &types.MsgFulfillBounty{
		Caller: caller.String(), BountyId: bountyID, FactId: "fact-1",
	}); !errors.Is(err, types.ErrAlreadyFulfilled) {
		t.Fatalf("expected ErrAlreadyFulfilled, got %v", err)
	}
}

func TestFulfillBounty_TransitionsToFulfilled(t *testing.T) {
	k, ctx, bk, kk := setup(t)
	srv := keeper.NewMsgServerImpl(k)
	sponsor := mkAddr("sponsor-trans-aaaaa1")
	worker := mkAddr("worker-trans-aaaaaa1")
	bountyID := createTestBounty(t, k, srv, ctx, bk, sponsor, "math", "1000", 2, 500)
	makeVerifiedFact(t, kk, "fact-1", "math", worker.String(), 1000)
	makeVerifiedFact(t, kk, "fact-2", "math", worker.String(), 1000)
	caller := mkAddr("caller-trans-aaaaaa1")

	resp1, _ := srv.FulfillBounty(ctx, &types.MsgFulfillBounty{Caller: caller.String(), BountyId: bountyID, FactId: "fact-1"})
	if resp1.BountyNowFulfilled {
		t.Error("after 1 of 2, should not be fulfilled")
	}
	resp2, _ := srv.FulfillBounty(ctx, &types.MsgFulfillBounty{Caller: caller.String(), BountyId: bountyID, FactId: "fact-2"})
	if !resp2.BountyNowFulfilled {
		t.Error("after 2 of 2, should be fulfilled")
	}

	order, _ := k.GetBountyOrder(ctx, bountyID)
	if order.Status != types.BountyStatus_BOUNTY_STATUS_FULFILLED {
		t.Errorf("status: want FULFILLED, got %s", order.Status)
	}
}

func TestFulfillBounty_ExpiredRejected(t *testing.T) {
	k, ctx, bk, kk := setup(t)
	srv := keeper.NewMsgServerImpl(k)
	sponsor := mkAddr("sponsor-exp-aaaaaaa1")
	worker := mkAddr("worker-exp-aaaaaaaa1")
	bountyID := createTestBounty(t, k, srv, ctx, bk, sponsor, "math", "1000", 5, 100)
	makeVerifiedFact(t, kk, "fact-1", "math", worker.String(), 1000)

	ctx2 := ctx.WithBlockHeight(int64(1101))
	k.ProcessBountyExpiry(ctx2, 1101)

	caller := mkAddr("caller-exp-aaaaaaaaa1")
	_, err := srv.FulfillBounty(ctx2, &types.MsgFulfillBounty{
		Caller: caller.String(), BountyId: bountyID, FactId: "fact-1",
	})
	if err == nil || (!errors.Is(err, types.ErrBountyNotActive) && !errors.Is(err, types.ErrBountyExpired)) {
		t.Fatalf("expected ErrBountyNotActive or ErrBountyExpired, got %v", err)
	}
}

// ---------- CancelBountyOrder ----------

func TestCancelBountyOrder_HappyPath(t *testing.T) {
	k, ctx, bk, _ := setup(t)
	srv := keeper.NewMsgServerImpl(k)
	sponsor := mkAddr("sponsor-cancel-h-aa1")
	bountyID := createTestBounty(t, k, srv, ctx, bk, sponsor, "math", "1000000", 5, 500)

	preBalance := bk.balances[sponsor.String()]["uzrn"]
	resp, err := srv.CancelBountyOrder(ctx, &types.MsgCancelBountyOrder{
		Sponsor: sponsor.String(), BountyId: bountyID,
	})
	if err != nil {
		t.Fatalf("Cancel: %v", err)
	}
	if resp.RefundedAmount != "5000000" {
		t.Errorf("refund: want 5000000, got %s", resp.RefundedAmount)
	}
	postBalance := bk.balances[sponsor.String()]["uzrn"]
	if postBalance-preBalance != 5_000_000 {
		t.Errorf("sponsor balance delta: want 5000000, got %d", postBalance-preBalance)
	}
	order, _ := k.GetBountyOrder(ctx, bountyID)
	if order.Status != types.BountyStatus_BOUNTY_STATUS_CANCELED {
		t.Errorf("status: want CANCELED, got %s", order.Status)
	}
	if order.EscrowRemaining != "0" {
		t.Errorf("escrow_remaining: want 0, got %s", order.EscrowRemaining)
	}
}

func TestCancelBountyOrder_NonSponsorRejected(t *testing.T) {
	k, ctx, bk, _ := setup(t)
	srv := keeper.NewMsgServerImpl(k)
	sponsor := mkAddr("sponsor-real-aaaaaaa1")
	bountyID := createTestBounty(t, k, srv, ctx, bk, sponsor, "math", "1000000", 5, 500)
	other := mkAddr("not-the-sponsor-aaaa1")
	_, err := srv.CancelBountyOrder(ctx, &types.MsgCancelBountyOrder{
		Sponsor: other.String(), BountyId: bountyID,
	})
	if err == nil || !errors.Is(err, types.ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized, got %v", err)
	}
}

func TestCancelBountyOrder_AlreadyCanceledRejected(t *testing.T) {
	k, ctx, bk, _ := setup(t)
	srv := keeper.NewMsgServerImpl(k)
	sponsor := mkAddr("sponsor-ac-aaaaaaaa1")
	bountyID := createTestBounty(t, k, srv, ctx, bk, sponsor, "math", "1000000", 5, 500)

	if _, err := srv.CancelBountyOrder(ctx, &types.MsgCancelBountyOrder{Sponsor: sponsor.String(), BountyId: bountyID}); err != nil {
		t.Fatalf("first cancel: %v", err)
	}
	_, err := srv.CancelBountyOrder(ctx, &types.MsgCancelBountyOrder{Sponsor: sponsor.String(), BountyId: bountyID})
	if err == nil || !errors.Is(err, types.ErrBountyNotActive) {
		t.Fatalf("expected ErrBountyNotActive on re-cancel, got %v", err)
	}
}

func TestCancelBountyOrder_PartialFulfillmentRefund(t *testing.T) {
	k, ctx, bk, kk := setup(t)
	srv := keeper.NewMsgServerImpl(k)
	sponsor := mkAddr("sponsor-partial-aaa1")
	worker := mkAddr("worker-partial-aaaa1")
	bountyID := createTestBounty(t, k, srv, ctx, bk, sponsor, "math", "1000000", 5, 500)
	makeVerifiedFact(t, kk, "fact-1", "math", worker.String(), 1000)
	caller := mkAddr("caller-partial-aaaaa")

	if _, err := srv.FulfillBounty(ctx, &types.MsgFulfillBounty{Caller: caller.String(), BountyId: bountyID, FactId: "fact-1"}); err != nil {
		t.Fatalf("fulfill: %v", err)
	}
	resp, err := srv.CancelBountyOrder(ctx, &types.MsgCancelBountyOrder{Sponsor: sponsor.String(), BountyId: bountyID})
	if err != nil {
		t.Fatalf("cancel: %v", err)
	}
	if resp.RefundedAmount != "4000000" {
		t.Errorf("refund: want 4000000, got %s", resp.RefundedAmount)
	}
}
