package keeper_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"strings"
	"testing"

	corestore "cosmossdk.io/core/store"
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
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

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
	failModuleSend bool
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

func (m *mockBankKeeper) GetBalance(_ context.Context, addr sdk.AccAddress, denom string) sdk.Coin {
	moduleAddr := sdk.AccAddress(authtypes.NewModuleAddress(types.ModuleName))
	if addr.Equals(moduleAddr) {
		return sdk.NewInt64Coin(denom, m.moduleBalances[types.ModuleName][denom])
	}
	return sdk.NewInt64Coin(denom, m.balances[addr.String()][denom])
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
	if m.failModuleSend {
		return errors.New("injected module send failure")
	}
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
	k, ctx, bk, kk, _ := setupWithStoreService(t)
	return k, ctx, bk, kk
}

func setupWithStoreService(t *testing.T) (keeper.Keeper, sdk.Context, *mockBankKeeper, *mockKnowledgeKeeper, corestore.KVStoreService) {
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
	return k, ctx, bk, kk, storeService
}

func storeHas(t *testing.T, storeService corestore.KVStoreService, ctx sdk.Context, key []byte) bool {
	t.Helper()
	bz, err := storeService.OpenKVStore(ctx).Get(key)
	if err != nil {
		t.Fatalf("read store key %x: %v", key, err)
	}
	return bz != nil
}

func mkAddr(seed string) sdk.AccAddress {
	b := make([]byte, 20)
	copy(b, []byte(seed))
	return sdk.AccAddress(b)
}

func testDigest(label string) string {
	sum := sha256.Sum256([]byte(label))
	return hex.EncodeToString(sum[:])
}

func testWorkContract(workerAddress ...string) *types.WorkContract {
	assignedWorker := mkAddr("default-assigned-work").String()
	if len(workerAddress) > 0 {
		assignedWorker = workerAddress[0]
	}
	return &types.WorkContract{
		WorkSpecHash:      testDigest("work-spec"),
		AcceptanceHash:    testDigest("acceptance"),
		InputRoot:         testDigest("input"),
		EnvironmentRoot:   testDigest("environment"),
		MinCorroborations: 0,
		WorkerAddress:     assignedWorker,
	}
}

func makeVerifiedFact(t *testing.T, kk *mockKnowledgeKeeper, factID, domain, submitter string, submittedAt uint64) {
	t.Helper()
	contract := testWorkContract()
	work := &knowledgetypes.ComputationalCommitment{
		WorkSpecHash:    contract.WorkSpecHash,
		AcceptanceHash:  contract.AcceptanceHash,
		InputRoot:       contract.InputRoot,
		EnvironmentRoot: contract.EnvironmentRoot,
		ArtifactRoot:    testDigest("artifact:" + factID),
		EvidenceRoot:    testDigest("evidence:" + factID),
	}
	work.WorkReceiptHash = knowledgetypes.ComputeWorkReceiptHash(work, submitter)
	kk.facts[factID] = &knowledgetypes.Fact{
		Id:                      factID,
		Domain:                  domain,
		Submitter:               submitter,
		SubmittedAtBlock:        submittedAt,
		Status:                  knowledgetypes.FactStatus_FACT_STATUS_VERIFIED,
		ClaimType:               knowledgetypes.ClaimType_CLAIM_TYPE_COMPUTATIONAL,
		ChallengeWindowEnd:      1000,
		CorroborationCount:      0,
		ComputationalCommitment: work,
	}
}

func createTestBounty(t *testing.T, k keeper.Keeper, srv types.MsgServer, ctx sdk.Context, bk *mockBankKeeper, sponsor sdk.AccAddress, domain, price string, target uint32, duration uint64, workerAddress ...string) string {
	return createTestBountyWithContract(t, k, srv, ctx, bk, sponsor, domain, price, target, duration, testWorkContract(workerAddress...))
}

func createTestBountyWithContract(t *testing.T, k keeper.Keeper, srv types.MsgServer, ctx sdk.Context, bk *mockBankKeeper, sponsor sdk.AccAddress, domain, price string, target uint32, duration uint64, contract *types.WorkContract) string {
	t.Helper()
	bk.setBalance(sponsor.String(), "uzrn", 1_000_000_000_000)
	resp, err := srv.CreateBountyOrder(ctx, &types.MsgCreateBountyOrder{
		Sponsor: sponsor.String(), Domain: domain, PricePerArtifact: price,
		TargetCount: target, DurationBlocks: duration,
		WorkContract: contract,
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
		WorkContract:     testWorkContract(),
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
		WorkContract:     testWorkContract(),
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
		WorkContract: testWorkContract(),
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
		WorkContract: testWorkContract(),
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
		WorkContract: testWorkContract(),
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

func TestCreateBountyOrder_SponsorAliasCannotEvadeCanonicalCap(t *testing.T) {
	k, ctx, bk, _, storeService := setupWithStoreService(t)
	srv := keeper.NewMsgServerImpl(k)
	k.SetParams(ctx, &types.Params{MinTargetCount: 1, MinDurationBlocks: 100, MaxActiveBountiesPerSponsor: 1})
	sponsor := mkAddr("sponsor-alias-cap-123")
	canonical := sponsor.String()
	alias := strings.ToUpper(canonical)
	bk.setBalance(canonical, "uzrn", 10)

	first, err := srv.CreateBountyOrder(ctx, &types.MsgCreateBountyOrder{
		Sponsor: canonical, Domain: "math", PricePerArtifact: "1",
		TargetCount: 1, DurationBlocks: 500, WorkContract: testWorkContract(),
	})
	if err != nil {
		t.Fatalf("canonical create: %v", err)
	}
	sponsorBefore := bk.balances[canonical]["uzrn"]
	moduleBefore := bk.moduleBalances[types.ModuleName]["uzrn"]
	_, err = srv.CreateBountyOrder(ctx, &types.MsgCreateBountyOrder{
		Sponsor: alias, Domain: "math", PricePerArtifact: "1",
		TargetCount: 1, DurationBlocks: 500, WorkContract: testWorkContract(),
	})
	if err == nil || !strings.Contains(err.Error(), "canonical lowercase") {
		t.Fatalf("uppercase alias must fail before cap/escrow admission: %v", err)
	}
	if bk.balances[canonical]["uzrn"] != sponsorBefore || bk.moduleBalances[types.ModuleName]["uzrn"] != moduleBefore {
		t.Fatal("rejected address alias moved escrow")
	}
	if got := k.CountActiveBountiesBySponsor(ctx, alias); got != 1 {
		t.Fatalf("alias lookup must resolve the canonical cap bucket: got %d", got)
	}
	if !storeHas(t, storeService, ctx, types.ActiveSponsorIndexKey(canonical, first.BountyId)) {
		t.Fatal("canonical active-sponsor key missing")
	}
	if storeHas(t, storeService, ctx, types.ActiveSponsorIndexKey(alias, first.BountyId)) {
		t.Fatal("noncanonical active-sponsor key must never be written")
	}
	order, found := k.GetBountyOrder(ctx, first.BountyId)
	if !found || order.Sponsor != canonical {
		t.Fatalf("new order did not store canonical sponsor: %+v", order)
	}
}

func TestCreateBountyOrder_LazilyPrunesExpiredSponsorSlot(t *testing.T) {
	k, ctx, bk, _ := setup(t)
	srv := keeper.NewMsgServerImpl(k)
	k.SetParams(ctx, &types.Params{MinTargetCount: 1, MinDurationBlocks: 100, MaxActiveBountiesPerSponsor: 1})
	sponsor := mkAddr("sponsor-lazy-prune1")

	first := createTestBounty(t, k, srv, ctx, bk, sponsor, "math", "1", 1, 100)
	ctx = ctx.WithBlockHeight(1100)
	second := createTestBounty(t, k, srv, ctx, bk, sponsor, "math", "1", 1, 100)

	firstOrder, _ := k.GetBountyOrder(ctx, first)
	secondOrder, _ := k.GetBountyOrder(ctx, second)
	if firstOrder.Status != types.BountyStatus_BOUNTY_STATUS_EXPIRED || secondOrder.Status != types.BountyStatus_BOUNTY_STATUS_ACTIVE {
		t.Fatalf("lazy expiry did not free sponsor slot: first=%s second=%s", firstOrder.Status, secondOrder.Status)
	}
	if got := k.CountActiveBountiesBySponsor(ctx, sponsor.String()); got != 1 {
		t.Fatalf("active sponsor index: got %d want 1", got)
	}
	liability, err := k.TotalEscrowLiability(ctx)
	if err != nil || liability.String() != "2" {
		t.Fatalf("expired plus active escrow remains liable: got %v err=%v", liability, err)
	}
	if err := k.EnsureEscrowAccounting(ctx); err != nil {
		t.Fatalf("lazy prune changed liability accounting: %v", err)
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
		WorkContract: testWorkContract(),
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
	bountyID := createTestBounty(t, k, srv, ctx, bk, sponsor, "math", "1000000", 5, 500, worker.String())
	makeVerifiedFact(t, kk, "fact-1", "math", worker.String(), 1000)

	caller := worker
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
	if kk.facts["fact-1"].CorroborationCount != 0 {
		t.Fatal("happy path must exercise the zero-formal-challenge baseline")
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
	bountyID := createTestBounty(t, k, srv, ctx, bk, sponsor, "math", "1000", 5, 500, worker.String())
	kk.facts["fact-pending"] = &knowledgetypes.Fact{
		Id: "fact-pending", Domain: "math", Submitter: worker.String(),
		SubmittedAtBlock: 1000, Status: knowledgetypes.FactStatus_FACT_STATUS_PENDING,
	}
	caller := worker
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
	bountyID := createTestBounty(t, k, srv, ctx, bk, sponsor, "math", "1000", 5, 500, worker.String())
	makeVerifiedFact(t, kk, "fact-bio", "biology", worker.String(), 1000)
	caller := worker
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
	bountyID := createTestBounty(t, k, srv, ctx, bk, sponsor, "math", "1000", 5, 500, worker.String())
	makeVerifiedFact(t, kk, "fact-retro", "math", worker.String(), 999)
	caller := worker
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
	bountyID := createTestBounty(t, k, srv, ctx, bk, sponsor, "math", "1000", 5, 500, worker.String())
	makeVerifiedFact(t, kk, "fact-1", "math", worker.String(), 1000)
	caller := worker

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
	bountyID := createTestBounty(t, k, srv, ctx, bk, sponsor, "math", "1000", 2, 500, worker.String())
	makeVerifiedFact(t, kk, "fact-1", "math", worker.String(), 1000)
	makeVerifiedFact(t, kk, "fact-2", "math", worker.String(), 1000)
	caller := worker

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
	bountyID := createTestBounty(t, k, srv, ctx, bk, sponsor, "math", "1000", 5, 100, worker.String())
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

func TestProcessBountyExpiry_BoundsTransitionsAndUsesDeadlineIndex(t *testing.T) {
	k, ctx, bk, _ := setup(t)
	const orderCount = keeper.MaxExpiryTransitionsPerBlock + 1
	sponsorA := mkAddr("a-deadline-index-spon")
	sponsorB := mkAddr("b-deadline-index-spon")
	orders := make([]*types.BountyOrder, 0, orderCount)
	for i := 1; i <= orderCount; i++ {
		sponsor := sponsorA
		if i == orderCount {
			sponsor = sponsorB
		}
		orders = append(orders, &types.BountyOrder{
			Id: fmt.Sprintf("bounty-%d", i), Sponsor: sponsor.String(), Domain: "math",
			PricePerArtifact: "1", TargetCount: 1, EscrowRemaining: "1",
			StartBlock: 1, EndBlock: 900, Status: types.BountyStatus_BOUNTY_STATUS_ACTIVE,
			WorkContract: testWorkContract(),
		})
	}
	bk.moduleBalances[types.ModuleName] = map[string]int64{"uzrn": orderCount}
	k.InitGenesis(ctx, &types.GenesisState{
		Params: &types.Params{
			MinTargetCount: 1, MinDurationBlocks: 100,
			MaxActiveBountiesPerSponsor: types.MaxActiveBountiesPerSponsorHardCap,
		},
		Orders: orders, NextBountyId: orderCount + 1,
	})

	k.ProcessBountyExpiry(ctx, 1000)
	active, expired := 0, 0
	for _, order := range k.GetAllBountyOrders(ctx) {
		switch order.Status {
		case types.BountyStatus_BOUNTY_STATUS_ACTIVE:
			active++
		case types.BountyStatus_BOUNTY_STATUS_EXPIRED:
			expired++
		}
	}
	if expired != keeper.MaxExpiryTransitionsPerBlock || active != 1 {
		t.Fatalf("first bounded expiry pass: expired=%d active=%d", expired, active)
	}

	k.ProcessBountyExpiry(ctx, 1000)
	for _, order := range k.GetAllBountyOrders(ctx) {
		if order.Status != types.BountyStatus_BOUNTY_STATUS_EXPIRED {
			t.Fatalf("second expiry pass left %s active", order.Id)
		}
	}
	if got := k.CountActiveBountiesBySponsor(ctx, sponsorA.String()) + k.CountActiveBountiesBySponsor(ctx, sponsorB.String()); got != 0 {
		t.Fatalf("terminal orders remain in active sponsor index: %d", got)
	}
	liability, err := k.TotalEscrowLiability(ctx)
	if err != nil || liability.Int64() != orderCount {
		t.Fatalf("expiry must preserve refundable liability: got %v err=%v", liability, err)
	}
}

func TestTerminalTransitionsDeleteSponsorAndDeadlineIndexes(t *testing.T) {
	t.Run("fulfillment", func(t *testing.T) {
		k, ctx, bk, kk, storeService := setupWithStoreService(t)
		srv := keeper.NewMsgServerImpl(k)
		sponsor := mkAddr("terminal-index-ful-sp")
		worker := mkAddr("terminal-index-ful-wk")
		bountyID := createTestBounty(t, k, srv, ctx, bk, sponsor, "math", "1", 1, 500, worker.String())
		order, _ := k.GetBountyOrder(ctx, bountyID)
		activeKey := types.ActiveSponsorIndexKey(sponsor.String(), bountyID)
		deadlineKey := types.DeadlineIndexKey(order.EndBlock, bountyID)
		if !storeHas(t, storeService, ctx, activeKey) || !storeHas(t, storeService, ctx, deadlineKey) {
			t.Fatal("created bounty missing active indexes")
		}

		makeVerifiedFact(t, kk, "fact-terminal-index", "math", worker.String(), 1000)
		if _, err := srv.FulfillBounty(ctx, &types.MsgFulfillBounty{
			Caller: worker.String(), BountyId: bountyID, FactId: "fact-terminal-index",
		}); err != nil {
			t.Fatalf("fulfill terminal bounty: %v", err)
		}
		if storeHas(t, storeService, ctx, activeKey) || storeHas(t, storeService, ctx, deadlineKey) {
			t.Fatal("fulfilled bounty retained active index")
		}
	})

	t.Run("legacy_active_cancel", func(t *testing.T) {
		k, ctx, bk, _, storeService := setupWithStoreService(t)
		srv := keeper.NewMsgServerImpl(k)
		sponsor := mkAddr("terminal-index-can-sp")
		order := &types.BountyOrder{
			Id: "bounty-1", Sponsor: sponsor.String(), Domain: "math", PricePerArtifact: "100",
			TargetCount: 1, EscrowRemaining: "100", StartBlock: 900, EndBlock: 1500,
			Status: types.BountyStatus_BOUNTY_STATUS_ACTIVE,
		}
		bk.moduleBalances[types.ModuleName] = map[string]int64{"uzrn": 100}
		k.InitGenesis(ctx, &types.GenesisState{
			Params: types.DefaultParams(), Orders: []*types.BountyOrder{order}, NextBountyId: 2,
		})
		activeKey := types.ActiveSponsorIndexKey(sponsor.String(), order.Id)
		deadlineKey := types.DeadlineIndexKey(order.EndBlock, order.Id)
		if !storeHas(t, storeService, ctx, activeKey) || !storeHas(t, storeService, ctx, deadlineKey) {
			t.Fatal("imported legacy bounty missing active indexes")
		}

		if _, err := srv.CancelBountyOrder(ctx, &types.MsgCancelBountyOrder{
			Sponsor: sponsor.String(), BountyId: order.Id,
		}); err != nil {
			t.Fatalf("cancel legacy active bounty: %v", err)
		}
		if storeHas(t, storeService, ctx, activeKey) || storeHas(t, storeService, ctx, deadlineKey) {
			t.Fatal("canceled bounty retained active index")
		}
	})
}

// ---------- CancelBountyOrder ----------

func TestCancelBountyOrder_HappyPath(t *testing.T) {
	k, ctx, bk, _ := setup(t)
	srv := keeper.NewMsgServerImpl(k)
	sponsor := mkAddr("sponsor-cancel-h-aa1")
	bountyID := createTestBounty(t, k, srv, ctx, bk, sponsor, "math", "1000000", 5, 500)
	_, err := srv.CancelBountyOrder(ctx, &types.MsgCancelBountyOrder{
		Sponsor: sponsor.String(), BountyId: bountyID,
	})
	if !errors.Is(err, types.ErrBountyNotCancelable) {
		t.Fatalf("bound active bounty must not be cancelable, got %v", err)
	}
	ctx = ctx.WithBlockHeight(1500)
	k.ProcessBountyExpiry(ctx, 1500)

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
	ctx = ctx.WithBlockHeight(1500)
	k.ProcessBountyExpiry(ctx, 1500)

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
	bountyID := createTestBounty(t, k, srv, ctx, bk, sponsor, "math", "1000000", 5, 500, worker.String())
	makeVerifiedFact(t, kk, "fact-1", "math", worker.String(), 1000)
	caller := worker

	if _, err := srv.FulfillBounty(ctx, &types.MsgFulfillBounty{Caller: caller.String(), BountyId: bountyID, FactId: "fact-1"}); err != nil {
		t.Fatalf("fulfill: %v", err)
	}
	ctx = ctx.WithBlockHeight(1500)
	k.ProcessBountyExpiry(ctx, 1500)
	resp, err := srv.CancelBountyOrder(ctx, &types.MsgCancelBountyOrder{Sponsor: sponsor.String(), BountyId: bountyID})
	if err != nil {
		t.Fatalf("cancel: %v", err)
	}
	if resp.RefundedAmount != "4000000" {
		t.Errorf("refund: want 4000000, got %s", resp.RefundedAmount)
	}
}

func TestFulfillBounty_CrossBountyFactReplayRejected(t *testing.T) {
	k, ctx, bk, kk := setup(t)
	srv := keeper.NewMsgServerImpl(k)
	sponsor := mkAddr("sponsor-replay-fact1")
	worker := mkAddr("worker-replay-fact12")
	bounty1 := createTestBounty(t, k, srv, ctx, bk, sponsor, "math", "1000", 1, 500, worker.String())
	bounty2 := createTestBounty(t, k, srv, ctx, bk, sponsor, "math", "1000", 1, 500, worker.String())
	makeVerifiedFact(t, kk, "fact-global-replay", "math", worker.String(), 1000)
	caller := worker

	_, err := srv.FulfillBounty(ctx, &types.MsgFulfillBounty{Caller: caller.String(), BountyId: bounty1, FactId: "fact-global-replay"})
	if err != nil {
		t.Fatalf("first fulfillment: %v", err)
	}
	_, err = srv.FulfillBounty(ctx, &types.MsgFulfillBounty{Caller: caller.String(), BountyId: bounty2, FactId: "fact-global-replay"})
	if !errors.Is(err, types.ErrSettlementReplay) {
		t.Fatalf("expected cross-bounty fact replay refusal, got %v", err)
	}
}

func TestFulfillBounty_CrossBountyReceiptReplayRejected(t *testing.T) {
	k, ctx, bk, kk := setup(t)
	srv := keeper.NewMsgServerImpl(k)
	sponsor := mkAddr("sponsor-replay-receipt")
	worker := mkAddr("worker-replay-receip")
	bounty1 := createTestBounty(t, k, srv, ctx, bk, sponsor, "math", "1000", 1, 500, worker.String())
	bounty2 := createTestBounty(t, k, srv, ctx, bk, sponsor, "math", "1000", 1, 500, worker.String())
	makeVerifiedFact(t, kk, "fact-receipt-1", "math", worker.String(), 1000)
	makeVerifiedFact(t, kk, "fact-receipt-2", "math", worker.String(), 1000)
	// A fresh fact wrapper around the same stored work receipt must not create
	// a second economic act.
	kk.facts["fact-receipt-2"].ComputationalCommitment = kk.facts["fact-receipt-1"].ComputationalCommitment
	caller := worker

	_, err := srv.FulfillBounty(ctx, &types.MsgFulfillBounty{Caller: caller.String(), BountyId: bounty1, FactId: "fact-receipt-1"})
	if err != nil {
		t.Fatalf("first fulfillment: %v", err)
	}
	_, err = srv.FulfillBounty(ctx, &types.MsgFulfillBounty{Caller: caller.String(), BountyId: bounty2, FactId: "fact-receipt-2"})
	if !errors.Is(err, types.ErrSettlementReplay) {
		t.Fatalf("expected cross-bounty receipt replay refusal, got %v", err)
	}
}

func TestFulfillBounty_SameArtifactWithFreshEvidenceAndReceiptRejected(t *testing.T) {
	k, ctx, bk, kk := setup(t)
	srv := keeper.NewMsgServerImpl(k)
	sponsor := mkAddr("sponsor-artifact-replay")
	worker := mkAddr("worker-artifact-replay1")
	bounty1 := createTestBounty(t, k, srv, ctx, bk, sponsor, "math", "1000", 1, 500, worker.String())
	bounty2 := createTestBounty(t, k, srv, ctx, bk, sponsor, "math", "1000", 1, 500, worker.String())
	makeVerifiedFact(t, kk, "fact-artifact-1", "math", worker.String(), 1000)
	makeVerifiedFact(t, kk, "fact-artifact-2", "math", worker.String(), 1000)
	firstWork := kk.facts["fact-artifact-1"].ComputationalCommitment
	secondWork := kk.facts["fact-artifact-2"].ComputationalCommitment
	secondWork.ArtifactRoot = firstWork.ArtifactRoot
	secondWork.EvidenceRoot = testDigest("fresh-evidence-wrapper")
	secondWork.WorkReceiptHash = knowledgetypes.ComputeWorkReceiptHash(secondWork, worker.String())
	if secondWork.WorkReceiptHash == firstWork.WorkReceiptHash {
		t.Fatal("fixture requires distinct receipts")
	}

	_, err := srv.FulfillBounty(ctx, &types.MsgFulfillBounty{
		Caller: worker.String(), BountyId: bounty1, FactId: "fact-artifact-1",
	})
	if err != nil {
		t.Fatalf("first artifact payout: %v", err)
	}
	_, err = srv.FulfillBounty(ctx, &types.MsgFulfillBounty{
		Caller: worker.String(), BountyId: bounty2, FactId: "fact-artifact-2",
	})
	if !errors.Is(err, types.ErrSettlementReplay) {
		t.Fatalf("same work-spec/artifact with fresh evidence and receipt must replay-fail, got %v", err)
	}
}

func TestFulfillBounty_SameArtifactUnderDifferentInputContractIsDistinct(t *testing.T) {
	k, ctx, bk, kk := setup(t)
	srv := keeper.NewMsgServerImpl(k)
	sponsor := mkAddr("sponsor-distinct-input")
	worker := mkAddr("worker-distinct-input1")
	firstContract := testWorkContract(worker.String())
	secondContract := testWorkContract(worker.String())
	secondContract.InputRoot = testDigest("different-input-manifest")
	bounty1 := createTestBountyWithContract(t, k, srv, ctx, bk, sponsor, "math", "1000", 1, 500, firstContract)
	bounty2 := createTestBountyWithContract(t, k, srv, ctx, bk, sponsor, "math", "1000", 1, 500, secondContract)
	makeVerifiedFact(t, kk, "fact-input-1", "math", worker.String(), 1000)
	makeVerifiedFact(t, kk, "fact-input-2", "math", worker.String(), 1000)
	firstWork := kk.facts["fact-input-1"].ComputationalCommitment
	secondWork := kk.facts["fact-input-2"].ComputationalCommitment
	secondWork.InputRoot = secondContract.InputRoot
	secondWork.ArtifactRoot = firstWork.ArtifactRoot
	secondWork.WorkReceiptHash = knowledgetypes.ComputeWorkReceiptHash(secondWork, worker.String())

	_, err := srv.FulfillBounty(ctx, &types.MsgFulfillBounty{
		Caller: worker.String(), BountyId: bounty1, FactId: "fact-input-1",
	})
	if err != nil {
		t.Fatalf("first input contract payout: %v", err)
	}
	_, err = srv.FulfillBounty(ctx, &types.MsgFulfillBounty{
		Caller: worker.String(), BountyId: bounty2, FactId: "fact-input-2",
	})
	if err != nil {
		t.Fatalf("same artifact bytes under a different input contract must remain payable: %v", err)
	}
}

func TestFulfillBounty_AssignedWorkerBlocksAlternateClaimWrapper(t *testing.T) {
	k, ctx, bk, kk := setup(t)
	srv := keeper.NewMsgServerImpl(k)
	sponsor := mkAddr("sponsor-worker-binding")
	worker := mkAddr("assigned-worker-bind1")
	attacker := mkAddr("alternate-claim-wrap1")

	originalContract := testWorkContract(worker.String())
	attackerContract := testWorkContract(attacker.String())
	originalBounty := createTestBountyWithContract(t, k, srv, ctx, bk, sponsor, "math", "1000", 1, 500, originalContract)
	attackerBounty := createTestBountyWithContract(t, k, srv, ctx, bk, sponsor, "math", "1", 1, 500, attackerContract)
	makeVerifiedFact(t, kk, "fact-assigned-worker", "math", worker.String(), 1000)
	makeVerifiedFact(t, kk, "fact-alternate-wrapper", "math", attacker.String(), 1000)

	originalWork := kk.facts["fact-assigned-worker"].ComputationalCommitment
	attackerWork := kk.facts["fact-alternate-wrapper"].ComputationalCommitment
	attackerWork.WorkSpecHash = originalWork.WorkSpecHash
	attackerWork.AcceptanceHash = originalWork.AcceptanceHash
	attackerWork.InputRoot = originalWork.InputRoot
	attackerWork.EnvironmentRoot = originalWork.EnvironmentRoot
	attackerWork.ArtifactRoot = originalWork.ArtifactRoot
	attackerWork.EvidenceRoot = testDigest("attacker-evidence-wrapper")
	attackerWork.WorkReceiptHash = knowledgetypes.ComputeWorkReceiptHash(attackerWork, attacker.String())

	originalNullifier := types.ComputeSettlementNullifier(
		originalWork.WorkSpecHash, originalWork.AcceptanceHash, originalWork.InputRoot,
		originalWork.EnvironmentRoot, originalWork.ArtifactRoot, worker.String(),
	)
	attackerNullifier := types.ComputeSettlementNullifier(
		attackerWork.WorkSpecHash, attackerWork.AcceptanceHash, attackerWork.InputRoot,
		attackerWork.EnvironmentRoot, attackerWork.ArtifactRoot, attacker.String(),
	)
	if originalNullifier == attackerNullifier {
		t.Fatal("different worker assignments must derive distinct economic units")
	}

	_, err := srv.FulfillBounty(ctx, &types.MsgFulfillBounty{
		Caller: attacker.String(), BountyId: originalBounty, FactId: "fact-alternate-wrapper",
	})
	if !errors.Is(err, types.ErrUnauthorized) {
		t.Fatalf("alternate Claim submitter must not consume original assignment: %v", err)
	}
	if k.IsFactConsumed(ctx, "fact-alternate-wrapper") || k.IsReceiptConsumed(ctx, attackerWork.WorkReceiptHash) ||
		k.IsSettlementNullifierConsumed(ctx, originalNullifier) {
		t.Fatal("assignment refusal wrote settlement state")
	}

	if _, err := srv.FulfillBounty(ctx, &types.MsgFulfillBounty{
		Caller: attacker.String(), BountyId: attackerBounty, FactId: "fact-alternate-wrapper",
	}); err != nil {
		t.Fatalf("separately funded attacker assignment should remain a distinct contract: %v", err)
	}
	if !k.IsSettlementNullifierConsumed(ctx, attackerNullifier) {
		t.Fatal("attacker-assigned clone did not consume its worker-bound nullifier")
	}
	if _, err := srv.FulfillBounty(ctx, &types.MsgFulfillBounty{
		Caller: worker.String(), BountyId: originalBounty, FactId: "fact-assigned-worker",
	}); err != nil {
		t.Fatalf("attacker's distinct assignment must not grief original worker settlement: %v", err)
	}
	if !k.IsSettlementNullifierConsumed(ctx, originalNullifier) {
		t.Fatal("original worker settlement did not consume its nullifier")
	}
}

func TestFulfillBounty_ThirdPartyCannotFrontRunClonedCheapOrder(t *testing.T) {
	k, ctx, bk, kk := setup(t)
	srv := keeper.NewMsgServerImpl(k)
	sponsor := mkAddr("sponsor-cloned-orders")
	worker := mkAddr("worker-chooses-order12")
	attacker := mkAddr("attacker-cheap-order1")
	cheapBounty := createTestBounty(t, k, srv, ctx, bk, sponsor, "math", "1", 1, 500, worker.String())
	expensiveBounty := createTestBounty(t, k, srv, ctx, bk, sponsor, "math", "1000", 1, 500, worker.String())
	makeVerifiedFact(t, kk, "fact-worker-choice", "math", worker.String(), 1000)
	work := kk.facts["fact-worker-choice"].ComputationalCommitment
	nullifier := types.ComputeSettlementNullifier(work.WorkSpecHash, work.AcceptanceHash, work.InputRoot, work.EnvironmentRoot, work.ArtifactRoot, worker.String())
	cheapOrder, _ := k.GetBountyOrder(ctx, cheapBounty)
	expensiveOrder, _ := k.GetBountyOrder(ctx, expensiveBounty)
	cheapNullifier := types.ComputeSettlementNullifier(
		work.WorkSpecHash, work.AcceptanceHash, work.InputRoot, work.EnvironmentRoot,
		work.ArtifactRoot, cheapOrder.WorkContract.WorkerAddress,
	)
	expensiveNullifier := types.ComputeSettlementNullifier(
		work.WorkSpecHash, work.AcceptanceHash, work.InputRoot, work.EnvironmentRoot,
		work.ArtifactRoot, expensiveOrder.WorkContract.WorkerAddress,
	)
	if cheapNullifier != expensiveNullifier || cheapNullifier != nullifier {
		t.Fatal("same-worker cloned offers must share the worker-controlled single-use nullifier")
	}
	moduleBefore := bk.moduleBalances[types.ModuleName]["uzrn"]

	_, err := srv.FulfillBounty(ctx, &types.MsgFulfillBounty{
		Caller: attacker.String(), BountyId: cheapBounty, FactId: "fact-worker-choice",
	})
	if !errors.Is(err, types.ErrUnauthorized) {
		t.Fatalf("third-party cheap-order front-run must be refused, got %v", err)
	}
	if k.IsFactConsumed(ctx, "fact-worker-choice") || k.IsReceiptConsumed(ctx, work.WorkReceiptHash) || k.IsSettlementNullifierConsumed(ctx, nullifier) {
		t.Fatal("refused front-run consumed a settlement key")
	}
	if _, found := k.GetFulfillment(ctx, cheapBounty, "fact-worker-choice"); found {
		t.Fatal("refused front-run wrote a fulfillment")
	}
	if got := bk.moduleBalances[types.ModuleName]["uzrn"]; got != moduleBefore {
		t.Fatalf("refused front-run moved escrow: before=%d after=%d", moduleBefore, got)
	}

	resp, err := srv.FulfillBounty(ctx, &types.MsgFulfillBounty{
		Caller: worker.String(), BountyId: expensiveBounty, FactId: "fact-worker-choice",
	})
	if err != nil || resp.AmountPaid != "1000" {
		t.Fatalf("worker could not choose the higher-paying matching order: resp=%+v err=%v", resp, err)
	}
}

func TestFulfillBounty_ChallengeMaturityBoundary(t *testing.T) {
	for _, tc := range []struct {
		name    string
		height  int64
		wantErr bool
	}{
		{"end_minus_one", 1099, true},
		{"at_end", 1100, false},
		{"end_plus_one", 1101, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			k, ctx, bk, kk := setup(t)
			srv := keeper.NewMsgServerImpl(k)
			sponsor := mkAddr("sponsor-maturity-123")
			worker := mkAddr("worker-maturity-1234")
			bountyID := createTestBounty(t, k, srv, ctx, bk, sponsor, "math", "1000", 1, 500, worker.String())
			makeVerifiedFact(t, kk, "fact-maturity", "math", worker.String(), 1000)
			kk.facts["fact-maturity"].ChallengeWindowEnd = 1100
			caller := worker
			_, err := srv.FulfillBounty(ctx.WithBlockHeight(tc.height), &types.MsgFulfillBounty{
				Caller: caller.String(), BountyId: bountyID, FactId: "fact-maturity",
			})
			if tc.wantErr && !errors.Is(err, types.ErrFactNotMature) {
				t.Fatalf("expected maturity refusal, got %v", err)
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("expected mature fulfillment, got %v", err)
			}
		})
	}
}

func TestFulfillBounty_StatusAndCorroborationAreANDGates(t *testing.T) {
	statuses := []knowledgetypes.FactStatus{
		knowledgetypes.FactStatus_FACT_STATUS_PENDING,
		knowledgetypes.FactStatus_FACT_STATUS_PROVISIONAL,
		knowledgetypes.FactStatus_FACT_STATUS_CONTESTED,
		knowledgetypes.FactStatus_FACT_STATUS_CHALLENGED,
		knowledgetypes.FactStatus_FACT_STATUS_SUPERSEDED,
		knowledgetypes.FactStatus_FACT_STATUS_EXPIRED,
		knowledgetypes.FactStatus_FACT_STATUS_DISPROVEN,
		knowledgetypes.FactStatus_FACT_STATUS_REVOKED,
		knowledgetypes.FactStatus_FACT_STATUS_AT_RISK,
		knowledgetypes.FactStatus_FACT_STATUS_PRUNED,
	}
	for _, status := range statuses {
		t.Run(status.String(), func(t *testing.T) {
			k, ctx, bk, kk := setup(t)
			srv := keeper.NewMsgServerImpl(k)
			sponsor := mkAddr("sponsor-status-gate12")
			worker := mkAddr("worker-status-gate123")
			bountyID := createTestBounty(t, k, srv, ctx, bk, sponsor, "math", "1000", 1, 500, worker.String())
			makeVerifiedFact(t, kk, "fact-status", "math", worker.String(), 1000)
			kk.facts["fact-status"].Status = status
			caller := worker
			_, err := srv.FulfillBounty(ctx, &types.MsgFulfillBounty{Caller: caller.String(), BountyId: bountyID, FactId: "fact-status"})
			if !errors.Is(err, types.ErrFactNotEligible) {
				t.Fatalf("status %s should be refused, got %v", status, err)
			}
		})
	}

	k, ctx, bk, kk := setup(t)
	srv := keeper.NewMsgServerImpl(k)
	sponsor := mkAddr("sponsor-corroboration1")
	worker := mkAddr("worker-corroboration12")
	bountyID := createTestBounty(t, k, srv, ctx, bk, sponsor, "math", "1000", 1, 500, worker.String())
	order, _ := k.GetBountyOrder(ctx, bountyID)
	order.WorkContract.MinCorroborations = 2
	k.SetBountyOrder(ctx, order)
	makeVerifiedFact(t, kk, "fact-corroboration", "math", worker.String(), 1000)
	kk.facts["fact-corroboration"].Status = knowledgetypes.FactStatus_FACT_STATUS_ACTIVE
	kk.facts["fact-corroboration"].CorroborationCount = 1
	caller := worker
	_, err := srv.FulfillBounty(ctx, &types.MsgFulfillBounty{Caller: caller.String(), BountyId: bountyID, FactId: "fact-corroboration"})
	if !errors.Is(err, types.ErrFactNotMature) {
		t.Fatalf("expected min corroboration refusal, got %v", err)
	}
	kk.facts["fact-corroboration"].CorroborationCount = 2
	if _, err := srv.FulfillBounty(ctx, &types.MsgFulfillBounty{Caller: caller.String(), BountyId: bountyID, FactId: "fact-corroboration"}); err != nil {
		t.Fatalf("mature active fact with required corroborations: %v", err)
	}
}

func TestLegacyUnboundBounty_IsRefundOnly(t *testing.T) {
	k, ctx, bk, kk := setup(t)
	srv := keeper.NewMsgServerImpl(k)
	sponsor := mkAddr("legacy-sponsor-refund1")
	worker := mkAddr("legacy-worker-refund12")
	bk.moduleBalances[types.ModuleName] = map[string]int64{"uzrn": 1000}
	k.SetBountyOrder(ctx, &types.BountyOrder{
		Id: "bounty-1", Sponsor: sponsor.String(), Domain: "math", PricePerArtifact: "1000",
		TargetCount: 1, EscrowRemaining: "1000", StartBlock: 900, EndBlock: 1500,
		Status: types.BountyStatus_BOUNTY_STATUS_ACTIVE,
	})
	liability, _ := types.ParseNonNegativeAmount("1000")
	if err := k.SetEscrowLiability(ctx, liability); err != nil {
		t.Fatalf("seed legacy liability: %v", err)
	}
	makeVerifiedFact(t, kk, "fact-legacy", "math", worker.String(), 1000)
	caller := mkAddr("legacy-caller-refund12")
	_, err := srv.FulfillBounty(ctx, &types.MsgFulfillBounty{Caller: caller.String(), BountyId: "bounty-1", FactId: "fact-legacy"})
	if !errors.Is(err, types.ErrWorkContractRequired) {
		t.Fatalf("legacy order must be refund-only, got %v", err)
	}
	resp, err := srv.CancelBountyOrder(ctx, &types.MsgCancelBountyOrder{Sponsor: sponsor.String(), BountyId: "bounty-1"})
	if err != nil || resp.RefundedAmount != "1000" {
		t.Fatalf("legacy refund failed: resp=%+v err=%v", resp, err)
	}
}

func TestEscrowLiability_ConservedAcrossLifecycle(t *testing.T) {
	k, ctx, bk, kk := setup(t)
	srv := keeper.NewMsgServerImpl(k)
	sponsor := mkAddr("sponsor-liability-123")
	worker := mkAddr("worker-liability-1234")
	bountyID := createTestBounty(t, k, srv, ctx, bk, sponsor, "math", "100", 3, 500, worker.String())

	assertLiability := func(want int64) {
		t.Helper()
		liability, err := k.TotalEscrowLiability(ctx)
		if err != nil || liability.Int64() != want {
			t.Fatalf("liability: got %v err=%v want=%d", liability, err, want)
		}
		if bk.moduleBalances[types.ModuleName]["uzrn"] < want {
			t.Fatalf("module balance %d below liability %d", bk.moduleBalances[types.ModuleName]["uzrn"], want)
		}
		if err := k.EnsureEscrowAccounting(ctx); err != nil {
			t.Fatalf("persisted/derived liability mismatch: %v", err)
		}
	}
	assertLiability(300)
	makeVerifiedFact(t, kk, "fact-liability", "math", worker.String(), 1000)
	caller := worker
	if _, err := srv.FulfillBounty(ctx, &types.MsgFulfillBounty{Caller: caller.String(), BountyId: bountyID, FactId: "fact-liability"}); err != nil {
		t.Fatalf("fulfill: %v", err)
	}
	assertLiability(200)
	// Unsolicited module sends are surplus, not liabilities; the invariant is
	// balance >= liability rather than equality.
	bk.moduleBalances[types.ModuleName]["uzrn"] += 7
	assertLiability(200)
	ctx = ctx.WithBlockHeight(1500)
	k.ProcessBountyExpiry(ctx, 1500)
	if _, err := srv.CancelBountyOrder(ctx, &types.MsgCancelBountyOrder{Sponsor: sponsor.String(), BountyId: bountyID}); err != nil {
		t.Fatalf("cancel: %v", err)
	}
	assertLiability(0)
	if got := bk.moduleBalances[types.ModuleName]["uzrn"]; got != 7 {
		t.Fatalf("unsolicited surplus should remain in module: got %d", got)
	}
}

func TestFulfillBounty_EscrowShortageLeavesNoSettlementState(t *testing.T) {
	k, ctx, bk, kk := setup(t)
	srv := keeper.NewMsgServerImpl(k)
	sponsor := mkAddr("sponsor-bank-failure1")
	worker := mkAddr("worker-bank-failure12")
	bountyID := createTestBounty(t, k, srv, ctx, bk, sponsor, "math", "100", 1, 500, worker.String())
	makeVerifiedFact(t, kk, "fact-bank-failure", "math", worker.String(), 1000)
	work := kk.facts["fact-bank-failure"].ComputationalCommitment
	nullifier := types.ComputeSettlementNullifier(work.WorkSpecHash, work.AcceptanceHash, work.InputRoot, work.EnvironmentRoot, work.ArtifactRoot, worker.String())
	// Simulate a module-balance shortfall. The aggregate liability wall fails
	// before any order/index/fulfillment mutation.
	bk.moduleBalances[types.ModuleName]["uzrn"] = 0
	caller := worker
	_, err := srv.FulfillBounty(ctx, &types.MsgFulfillBounty{Caller: caller.String(), BountyId: bountyID, FactId: "fact-bank-failure"})
	if err == nil {
		t.Fatal("expected bank payout failure")
	}
	order, _ := k.GetBountyOrder(ctx, bountyID)
	if order.FulfilledCount != 0 || order.EscrowRemaining != "100" || order.Status != types.BountyStatus_BOUNTY_STATUS_ACTIVE {
		t.Fatalf("bank failure mutated order: %+v", order)
	}
	if _, found := k.GetFulfillment(ctx, bountyID, "fact-bank-failure"); found {
		t.Fatal("bank failure wrote fulfillment")
	}
	if k.IsFactConsumed(ctx, "fact-bank-failure") || k.IsReceiptConsumed(ctx, work.WorkReceiptHash) || k.IsSettlementNullifierConsumed(ctx, nullifier) {
		t.Fatal("bank failure wrote a replay tombstone")
	}
}

func TestFulfillBounty_BankFailureLeavesNoSettlementState(t *testing.T) {
	k, ctx, bk, kk := setup(t)
	srv := keeper.NewMsgServerImpl(k)
	sponsor := mkAddr("sponsor-send-failure12")
	worker := mkAddr("worker-send-failure123")
	bountyID := createTestBounty(t, k, srv, ctx, bk, sponsor, "math", "100", 1, 500, worker.String())
	makeVerifiedFact(t, kk, "fact-send-failure", "math", worker.String(), 1000)
	work := kk.facts["fact-send-failure"].ComputationalCommitment
	nullifier := types.ComputeSettlementNullifier(work.WorkSpecHash, work.AcceptanceHash, work.InputRoot, work.EnvironmentRoot, work.ArtifactRoot, worker.String())
	bk.failModuleSend = true

	_, err := srv.FulfillBounty(ctx, &types.MsgFulfillBounty{
		Caller: worker.String(), BountyId: bountyID, FactId: "fact-send-failure",
	})
	if err == nil {
		t.Fatal("expected injected bank payout failure")
	}
	order, _ := k.GetBountyOrder(ctx, bountyID)
	if order.FulfilledCount != 0 || order.EscrowRemaining != "100" || order.Status != types.BountyStatus_BOUNTY_STATUS_ACTIVE {
		t.Fatalf("bank failure mutated order: %+v", order)
	}
	if _, found := k.GetFulfillment(ctx, bountyID, "fact-send-failure"); found {
		t.Fatal("bank failure wrote fulfillment")
	}
	if k.IsFactConsumed(ctx, "fact-send-failure") || k.IsReceiptConsumed(ctx, work.WorkReceiptHash) || k.IsSettlementNullifierConsumed(ctx, nullifier) {
		t.Fatal("bank failure wrote a replay tombstone")
	}
	if got := bk.balances[worker.String()]["uzrn"]; got != 0 {
		t.Fatalf("bank failure paid worker %d", got)
	}
}

func TestMigration1To2_NormalizesLegacyPricesAndBuildsTombstones(t *testing.T) {
	k, ctx, bk, _ := setup(t)
	sponsor := mkAddr("legacy-migration-spon1")
	worker := mkAddr("legacy-migration-work1")
	k.SetBountyOrder(ctx, &types.BountyOrder{
		Id: "bounty-1", Sponsor: strings.ToUpper(sponsor.String()), Domain: "math", PricePerArtifact: "001",
		TargetCount: 2, EscrowRemaining: "+002", StartBlock: 1, EndBlock: 2000,
		Status: types.BountyStatus_BOUNTY_STATUS_ACTIVE,
	})
	k.SetBountyOrder(ctx, &types.BountyOrder{
		Id: "bounty-2", Sponsor: sponsor.String(), Domain: "math", PricePerArtifact: "+1",
		TargetCount: 1, FulfilledCount: 1, EscrowRemaining: "+0", StartBlock: 1, EndBlock: 2000,
		Status: types.BountyStatus_BOUNTY_STATUS_FULFILLED,
	})
	k.SetFulfillment(ctx, &types.BountyFulfillment{
		BountyId: "bounty-2", FactId: "legacy-paid-fact", Worker: worker.String(),
		AmountPaid: "+1", FulfilledAtBlock: 100,
	})
	bk.moduleBalances[types.ModuleName] = map[string]int64{"uzrn": 2}

	if err := keeper.NewMigrator(k).Migrate1to2(ctx); err != nil {
		t.Fatalf("migrate v1 prices: %v", err)
	}
	order1, _ := k.GetBountyOrder(ctx, "bounty-1")
	order2, _ := k.GetBountyOrder(ctx, "bounty-2")
	if order1.PricePerArtifact != "1" || order2.PricePerArtifact != "1" ||
		order1.EscrowRemaining != "2" || order2.EscrowRemaining != "0" {
		t.Fatalf("legacy amounts not normalized: %+v %+v", order1, order2)
	}
	if order1.Sponsor != sponsor.String() || order2.Sponsor != sponsor.String() {
		t.Fatalf("legacy sponsor aliases not normalized: %+v %+v", order1, order2)
	}
	if got := k.CountActiveBountiesBySponsor(ctx, strings.ToUpper(sponsor.String())); got != 1 {
		t.Fatalf("canonical migration index not shared by alias lookup: %d", got)
	}
	if !k.IsFactConsumed(ctx, "legacy-paid-fact") {
		t.Fatal("historical payout did not create permanent fact tombstone")
	}
	fulfillments := k.GetAllFulfillments(ctx)
	if len(fulfillments) != 1 || fulfillments[0].AmountPaid != "1" {
		t.Fatalf("legacy fulfillment amount not normalized: %+v", fulfillments)
	}
	liability, err := k.TotalEscrowLiability(ctx)
	if err != nil || liability.String() != "2" {
		t.Fatalf("migration did not persist exact aggregate liability: got %v err=%v", liability, err)
	}
	if err := k.EnsureEscrowAccounting(ctx); err != nil {
		t.Fatalf("migration persisted/derived liability mismatch: %v", err)
	}
}

func TestMigration1To2_SponsorAliasesShareCapBeforeWrites(t *testing.T) {
	k, ctx, bk, _ := setup(t)
	sponsor := mkAddr("legacy-alias-cap-spon")
	k.SetParams(ctx, &types.Params{
		MinTargetCount: 1, MinDurationBlocks: 100, MaxActiveBountiesPerSponsor: 1,
	})
	for i, address := range []string{sponsor.String(), strings.ToUpper(sponsor.String())} {
		k.SetBountyOrder(ctx, &types.BountyOrder{
			Id: fmt.Sprintf("bounty-%d", i+1), Sponsor: address, Domain: "math", PricePerArtifact: "1",
			TargetCount: 1, EscrowRemaining: "1", StartBlock: 1, EndBlock: 2000,
			Status: types.BountyStatus_BOUNTY_STATUS_ACTIVE,
		})
	}
	bk.moduleBalances[types.ModuleName] = map[string]int64{"uzrn": 2}

	err := keeper.NewMigrator(k).Migrate1to2(ctx)
	if err == nil || !strings.Contains(err.Error(), "2 active bounties") {
		t.Fatalf("address aliases must fail the single-account migration cap: %v", err)
	}
	if got := k.CountActiveBountiesBySponsor(ctx, sponsor.String()); got != 0 {
		t.Fatalf("failed migration wrote a canonical active index: %d", got)
	}
	storedAlias, found := k.GetBountyOrder(ctx, "bounty-2")
	if !found || storedAlias.Sponsor != strings.ToUpper(sponsor.String()) {
		t.Fatalf("failed preflight mutated stored legacy state: %+v", storedAlias)
	}
}

func TestMigration1To2_ClampsLegacyParamWithinAuditedActiveCardinality(t *testing.T) {
	k, ctx, bk, _ := setup(t)
	sponsor := mkAddr("legacy-cap-migration1")
	k.SetParams(ctx, &types.Params{
		MinTargetCount: 1, MinDurationBlocks: 100,
		MaxActiveBountiesPerSponsor: types.MaxActiveBountiesPerSponsorHardCap + 50,
	})
	k.SetBountyOrder(ctx, &types.BountyOrder{
		Id: "bounty-1", Sponsor: sponsor.String(), Domain: "math", PricePerArtifact: "1",
		TargetCount: 1, EscrowRemaining: "1", StartBlock: 1, EndBlock: 2000,
		Status: types.BountyStatus_BOUNTY_STATUS_ACTIVE,
	})
	bk.moduleBalances[types.ModuleName] = map[string]int64{"uzrn": 1}

	if err := keeper.NewMigrator(k).Migrate1to2(ctx); err != nil {
		t.Fatalf("migrate bounded legacy state: %v", err)
	}
	if got := k.GetParams(ctx).MaxActiveBountiesPerSponsor; got != types.MaxActiveBountiesPerSponsorHardCap {
		t.Fatalf("legacy max active not clamped: got %d", got)
	}
	if got := k.CountActiveBountiesBySponsor(ctx, sponsor.String()); got != 1 {
		t.Fatalf("active sponsor index not rebuilt: got %d", got)
	}
	liability, err := k.TotalEscrowLiability(ctx)
	if err != nil || liability.String() != "1" {
		t.Fatalf("persisted migration liability: got %v err=%v", liability, err)
	}
	k.ProcessBountyExpiry(ctx, 2000)
	order, _ := k.GetBountyOrder(ctx, "bounty-1")
	if order.Status != types.BountyStatus_BOUNTY_STATUS_EXPIRED || k.CountActiveBountiesBySponsor(ctx, sponsor.String()) != 0 {
		t.Fatalf("migration did not rebuild deadline/sponsor indexes: %+v", order)
	}
}

func TestMigration1To2_RefusesSponsorAboveHardCapBeforeWrites(t *testing.T) {
	k, ctx, bk, _ := setup(t)
	sponsor := mkAddr("legacy-over-cap-spon")
	k.SetParams(ctx, &types.Params{
		MinTargetCount: 1, MinDurationBlocks: 100,
		MaxActiveBountiesPerSponsor: types.MaxActiveBountiesPerSponsorHardCap + 50,
	})
	for i := uint32(1); i <= types.MaxActiveBountiesPerSponsorHardCap+1; i++ {
		k.SetBountyOrder(ctx, &types.BountyOrder{
			Id: fmt.Sprintf("bounty-%d", i), Sponsor: sponsor.String(), Domain: "math", PricePerArtifact: "1",
			TargetCount: 1, EscrowRemaining: "1", StartBlock: 1, EndBlock: 2000,
			Status: types.BountyStatus_BOUNTY_STATUS_ACTIVE,
		})
	}
	bk.moduleBalances[types.ModuleName] = map[string]int64{"uzrn": int64(types.MaxActiveBountiesPerSponsorHardCap + 1)}

	err := keeper.NewMigrator(k).Migrate1to2(ctx)
	if err == nil || !strings.Contains(err.Error(), "257 active bounties") {
		t.Fatalf("expected explicit hard-cap migration failure, got %v", err)
	}
	if got := k.GetParams(ctx).MaxActiveBountiesPerSponsor; got != types.MaxActiveBountiesPerSponsorHardCap+50 {
		t.Fatalf("failed migration wrote clamped params: got %d", got)
	}
	if got := k.CountActiveBountiesBySponsor(ctx, sponsor.String()); got != 0 {
		t.Fatalf("failed migration wrote active indexes: got %d", got)
	}
}

func TestMigration1To2_RefusesUndercollateralizedState(t *testing.T) {
	k, ctx, _, _ := setup(t)
	sponsor := mkAddr("legacy-short-migrate1")
	k.SetBountyOrder(ctx, &types.BountyOrder{
		Id: "bounty-1", Sponsor: sponsor.String(), Domain: "math", PricePerArtifact: "+1",
		TargetCount: 1, EscrowRemaining: "1", StartBlock: 1, EndBlock: 2000,
		Status: types.BountyStatus_BOUNTY_STATUS_ACTIVE,
	})
	if err := keeper.NewMigrator(k).Migrate1to2(ctx); err == nil {
		t.Fatal("undercollateralized legacy state must fail migration")
	}
}

func TestInitGenesis_RefusesUndercollateralizedEscrow(t *testing.T) {
	k, ctx, bk, _ := setup(t)
	sponsor := mkAddr("genesis-short-sponsor1")
	bk.moduleBalances[types.ModuleName] = map[string]int64{"uzrn": 99}
	gs := types.DefaultGenesis()
	gs.Orders = []*types.BountyOrder{{
		Id: "bounty-1", Sponsor: sponsor.String(), Domain: "math", PricePerArtifact: "100",
		TargetCount: 1, EscrowRemaining: "100", StartBlock: 1, EndBlock: 2000,
		Status: types.BountyStatus_BOUNTY_STATUS_ACTIVE, WorkContract: testWorkContract(),
	}}
	gs.NextBountyId = 2
	defer func() {
		if recover() == nil {
			t.Fatal("undercollateralized genesis must panic before chain startup")
		}
	}()
	k.InitGenesis(ctx, gs)
}

func TestInitExportGenesis_NormalizesLegacyV1Shapes(t *testing.T) {
	k, ctx, bk, _ := setup(t)
	sponsor := mkAddr("legacy-genesis-sponsor")
	worker := mkAddr("legacy-genesis-worker1")
	bk.moduleBalances[types.ModuleName] = map[string]int64{"uzrn": 2}
	gs := &types.GenesisState{
		Params: types.DefaultParams(),
		Orders: []*types.BountyOrder{
			{
				Id: "bounty-1", Sponsor: sponsor.String(), Domain: "math", PricePerArtifact: "001",
				TargetCount: 1, FulfilledCount: 1, EscrowRemaining: "+0", StartBlock: 100, EndBlock: 50,
				Status: types.BountyStatus_BOUNTY_STATUS_FULFILLED,
			},
			{
				Id: "bounty-2", Sponsor: sponsor.String(), Domain: "math", PricePerArtifact: "+1",
				TargetCount: 2, EscrowRemaining: "002", StartBlock: 100, EndBlock: 50,
				Status: types.BountyStatus_BOUNTY_STATUS_ACTIVE,
			},
		},
		Fulfillments: []*types.BountyFulfillment{{
			BountyId: "bounty-1", FactId: "legacy-overflow-paid", Worker: worker.String(),
			AmountPaid: "+1", FulfilledAtBlock: 101,
		}},
		NextBountyId: 3,
	}
	if err := gs.Validate(); err != nil {
		t.Fatalf("legitimate v1 export rejected: %v", err)
	}
	if gs.Orders[0].PricePerArtifact != "1" || gs.Orders[1].PricePerArtifact != "1" ||
		gs.Orders[0].EscrowRemaining != "0" || gs.Orders[1].EscrowRemaining != "2" ||
		gs.Fulfillments[0].AmountPaid != "1" {
		t.Fatalf("legacy values not normalized in place: %+v %+v", gs.Orders, gs.Fulfillments)
	}
	k.InitGenesis(ctx, gs)
	exported := k.ExportGenesis(ctx)
	if err := exported.Validate(); err != nil {
		t.Fatalf("normalized v2 export invalid: %v", err)
	}
	if exported.Orders[0].EndBlock != 50 || exported.Orders[1].EndBlock != 50 {
		t.Fatal("legacy wrapped deadlines must round-trip byte-semantically")
	}
	liability, err := k.TotalEscrowLiability(ctx)
	if err != nil || liability.String() != "2" {
		t.Fatalf("normalized liability: got %v err=%v", liability, err)
	}
}

func TestInitGenesis_LegacySponsorAliasCanCancelAndRefund(t *testing.T) {
	k, ctx, bk, _ := setup(t)
	sponsor := mkAddr("legacy-genesis-alias")
	canonical := sponsor.String()
	alias := strings.ToUpper(canonical)
	bk.moduleBalances[types.ModuleName] = map[string]int64{"uzrn": 10}
	gs := &types.GenesisState{
		Params: types.DefaultParams(),
		Orders: []*types.BountyOrder{{
			Id: "bounty-1", Sponsor: alias, Domain: "math", PricePerArtifact: "10",
			TargetCount: 1, EscrowRemaining: "10", StartBlock: 1, EndBlock: 2000,
			Status: types.BountyStatus_BOUNTY_STATUS_ACTIVE,
		}},
		NextBountyId: 2,
	}
	k.InitGenesis(ctx, gs)
	order, found := k.GetBountyOrder(ctx, "bounty-1")
	if !found || order.Sponsor != canonical {
		t.Fatalf("legacy genesis sponsor not normalized: %+v", order)
	}
	if got := k.CountActiveBountiesBySponsor(ctx, alias); got != 1 {
		t.Fatalf("legacy alias did not resolve canonical active index: %d", got)
	}

	resp, err := keeper.NewMsgServerImpl(k).CancelBountyOrder(ctx, &types.MsgCancelBountyOrder{
		Sponsor: alias, BountyId: order.Id,
	})
	if err != nil || resp.RefundedAmount != "10" {
		t.Fatalf("address-semantic legacy cancellation failed: resp=%+v err=%v", resp, err)
	}
	if got := bk.balances[canonical]["uzrn"]; got != 10 {
		t.Fatalf("legacy refund did not reach canonical account: %d", got)
	}
	order, _ = k.GetBountyOrder(ctx, order.Id)
	if order.Sponsor != canonical || order.Status != types.BountyStatus_BOUNTY_STATUS_CANCELED {
		t.Fatalf("canceled legacy order is not canonical and terminal: %+v", order)
	}
	if got := k.CountActiveBountiesBySponsor(ctx, alias); got != 0 {
		t.Fatalf("legacy cancellation left canonical active index: %d", got)
	}
}

func TestCreateBountyOrder_CounterExhaustionFailsBeforeEscrow(t *testing.T) {
	k, ctx, bk, _ := setup(t)
	gs := types.DefaultGenesis()
	gs.NextBountyId = math.MaxUint64 - 1
	k.InitGenesis(ctx, gs)
	srv := keeper.NewMsgServerImpl(k)
	sponsor := mkAddr("counter-overflow-spon1")
	bk.setBalance(sponsor.String(), "uzrn", 1000)
	msg := &types.MsgCreateBountyOrder{
		Sponsor: sponsor.String(), Domain: "math", PricePerArtifact: "1",
		TargetCount: 1, DurationBlocks: 100, WorkContract: testWorkContract(),
	}
	resp, err := srv.CreateBountyOrder(ctx, msg)
	if err != nil || resp.BountyId != "bounty-18446744073709551614" {
		t.Fatalf("last allocatable ID failed: resp=%+v err=%v", resp, err)
	}
	exported := k.ExportGenesis(ctx)
	if exported.NextBountyId != math.MaxUint64 {
		t.Fatalf("exhausted sentinel: got %d", exported.NextBountyId)
	}
	if err := exported.Validate(); err != nil {
		t.Fatalf("exhausted sentinel must round-trip through genesis: %v", err)
	}
	sponsorBefore := bk.balances[sponsor.String()]["uzrn"]
	moduleBefore := bk.moduleBalances[types.ModuleName]["uzrn"]
	if _, err := srv.CreateBountyOrder(ctx, msg); err == nil {
		t.Fatal("exhausted bounty counter must refuse creation")
	}
	if bk.balances[sponsor.String()]["uzrn"] != sponsorBefore || bk.moduleBalances[types.ModuleName]["uzrn"] != moduleBefore {
		t.Fatal("counter exhaustion moved escrow before failing")
	}
}
