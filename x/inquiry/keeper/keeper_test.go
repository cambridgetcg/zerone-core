package keeper_test

import (
	"context"
	"crypto/sha256"
	"fmt"
	"testing"

	dbm "github.com/cosmos/cosmos-db"

	sdkmath "cosmossdk.io/math"
	"cosmossdk.io/log"
	"cosmossdk.io/store"
	storemetrics "cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/x/inquiry/keeper"
	"github.com/zerone-chain/zerone/x/inquiry/types"
)

func init() {
	cfg := sdk.GetConfig()
	cfg.SetBech32PrefixForAccount("zrn", "zrnpub")
	cfg.SetBech32PrefixForValidator("zrnvaloper", "zrnvaloperpub")
	cfg.SetBech32PrefixForConsensusNode("zrnvalcons", "zrnvalconspub")
}

func testAddr(name string) string {
	h := sha256.Sum256([]byte("inquiry_test:" + name))
	return sdk.AccAddress(h[:20]).String()
}

// ─── Stub bank keeper (tracks module-account flows in memory) ─────────

type stubBank struct {
	accounts map[string]int64 // bech32 → uzrn
	modules  map[string]int64 // module name → uzrn
}

func newStubBank() *stubBank {
	return &stubBank{accounts: map[string]int64{}, modules: map[string]int64{}}
}

func (s *stubBank) fund(addr sdk.AccAddress, amt int64) {
	s.accounts[addr.String()] += amt
}

func (s *stubBank) SendCoinsFromAccountToModule(_ context.Context, sender sdk.AccAddress, recipientModule string, amt sdk.Coins) error {
	val := amt.AmountOf("uzrn").Int64()
	if s.accounts[sender.String()] < val {
		return fmt.Errorf("insufficient funds: %s has %d, needs %d", sender.String(), s.accounts[sender.String()], val)
	}
	s.accounts[sender.String()] -= val
	s.modules[recipientModule] += val
	return nil
}

func (s *stubBank) SendCoinsFromModuleToAccount(_ context.Context, senderModule string, recipient sdk.AccAddress, amt sdk.Coins) error {
	val := amt.AmountOf("uzrn").Int64()
	if s.modules[senderModule] < val {
		return fmt.Errorf("insufficient module funds: %s has %d, needs %d", senderModule, s.modules[senderModule], val)
	}
	s.modules[senderModule] -= val
	s.accounts[recipient.String()] += val
	return nil
}

func (s *stubBank) GetBalance(_ context.Context, addr sdk.AccAddress, _ string) sdk.Coin {
	return sdk.NewCoin("uzrn", sdkmath.NewInt(s.accounts[addr.String()]))
}

// ─── Stub knowledge keeper ────────────────────────────────────────────

type stubKnowledge struct {
	claims      map[string]string // claim_id → submitter
	acceptedFor map[string]string // claim_id → fact_id (for accepted only)
}

func newStubKnowledge() *stubKnowledge {
	return &stubKnowledge{
		claims:      map[string]string{},
		acceptedFor: map[string]string{},
	}
}

func (s *stubKnowledge) ClaimSubmitter(_ context.Context, claimID string) (string, bool) {
	v, ok := s.claims[claimID]
	return v, ok
}

func (s *stubKnowledge) AcceptedFactForClaim(_ context.Context, claimID string) (string, bool) {
	v, ok := s.acceptedFor[claimID]
	return v, ok
}

// ─── Setup ───────────────────────────────────────────────────────────

func setup(t *testing.T) (keeper.Keeper, types.MsgServer, *stubBank, *stubKnowledge, sdk.Context) {
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

	bank := newStubBank()
	kk := newStubKnowledge()
	storeService := runtime.NewKVStoreService(storeKey)
	k := keeper.NewKeeper(storeService, cdc, testAddr("authority"), bank)
	k.SetKnowledgeKeeper(kk)
	ctx := sdk.NewContext(stateStore, cmtproto.Header{Height: 100, ChainID: "zerone-test"}, false, log.NewNopLogger())
	return k, keeper.NewMsgServerImpl(k), bank, kk, ctx
}

// ─── Tests ───────────────────────────────────────────────────────────

func TestSubmitInquiry_EscrowsBounty(t *testing.T) {
	_, ms, bank, _, ctx := setup(t)
	asker, _ := sdk.AccAddressFromBech32(testAddr("asker"))
	bank.fund(asker, 5_000_000)
	resp, err := ms.SubmitInquiry(ctx, &types.MsgSubmitInquiry{
		Asker:    asker.String(),
		Question: "what?",
		Domain:   "physics",
		Bounty:   "2000000",
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.InquiryId == "" {
		t.Fatal("empty inquiry id")
	}
	if bank.accounts[asker.String()] != 3_000_000 {
		t.Fatalf("asker balance after escrow: %d (want 3_000_000)", bank.accounts[asker.String()])
	}
	if bank.modules[types.BountyPoolModuleName] != 2_000_000 {
		t.Fatalf("pool balance after escrow: %d (want 2_000_000)", bank.modules[types.BountyPoolModuleName])
	}
}

func TestSubmitInquiry_RejectsBelowMinBounty(t *testing.T) {
	_, ms, bank, _, ctx := setup(t)
	asker, _ := sdk.AccAddressFromBech32(testAddr("asker"))
	bank.fund(asker, 5_000_000)
	_, err := ms.SubmitInquiry(ctx, &types.MsgSubmitInquiry{
		Asker:    asker.String(),
		Question: "what?",
		Domain:   "physics",
		Bounty:   "100", // way below default min_bounty (1 ZRN)
	})
	if err == nil {
		t.Fatal("expected min-bounty rejection")
	}
}

func TestCancelInquiry_RefundsAsker(t *testing.T) {
	_, ms, bank, _, ctx := setup(t)
	asker, _ := sdk.AccAddressFromBech32(testAddr("asker"))
	bank.fund(asker, 5_000_000)
	resp, err := ms.SubmitInquiry(ctx, &types.MsgSubmitInquiry{
		Asker: asker.String(), Question: "q", Domain: "physics", Bounty: "2000000",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ms.CancelInquiry(ctx, &types.MsgCancelInquiry{
		Asker: asker.String(), InquiryId: resp.InquiryId,
	}); err != nil {
		t.Fatal(err)
	}
	if bank.accounts[asker.String()] != 5_000_000 {
		t.Fatalf("asker not refunded: balance %d", bank.accounts[asker.String()])
	}
	if bank.modules[types.BountyPoolModuleName] != 0 {
		t.Fatalf("pool not drained: %d", bank.modules[types.BountyPoolModuleName])
	}
}

func TestCancelInquiry_RejectsAfterAnswerLinked(t *testing.T) {
	_, ms, bank, kk, ctx := setup(t)
	asker, _ := sdk.AccAddressFromBech32(testAddr("asker"))
	answerer, _ := sdk.AccAddressFromBech32(testAddr("answerer"))
	bank.fund(asker, 5_000_000)
	resp, err := ms.SubmitInquiry(ctx, &types.MsgSubmitInquiry{
		Asker: asker.String(), Question: "q", Domain: "physics", Bounty: "2000000",
	})
	if err != nil {
		t.Fatal(err)
	}
	kk.claims["claim-1"] = answerer.String()
	if _, err := ms.SubmitAnswer(ctx, &types.MsgSubmitAnswer{
		Answerer: answerer.String(), InquiryId: resp.InquiryId, ClaimId: "claim-1",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := ms.CancelInquiry(ctx, &types.MsgCancelInquiry{
		Asker: asker.String(), InquiryId: resp.InquiryId,
	}); err == nil {
		t.Fatal("expected cancel to fail when answer in flight")
	}
}

func TestSubmitAnswer_RejectsCrossAuthor(t *testing.T) {
	_, ms, bank, kk, ctx := setup(t)
	asker, _ := sdk.AccAddressFromBech32(testAddr("asker"))
	owner, _ := sdk.AccAddressFromBech32(testAddr("claim_owner"))
	other, _ := sdk.AccAddressFromBech32(testAddr("other"))
	bank.fund(asker, 5_000_000)
	resp, err := ms.SubmitInquiry(ctx, &types.MsgSubmitInquiry{
		Asker: asker.String(), Question: "q", Domain: "physics", Bounty: "2000000",
	})
	if err != nil {
		t.Fatal(err)
	}
	kk.claims["claim-1"] = owner.String()
	if _, err := ms.SubmitAnswer(ctx, &types.MsgSubmitAnswer{
		Answerer: other.String(), InquiryId: resp.InquiryId, ClaimId: "claim-1",
	}); err == nil {
		t.Fatal("expected cross-author answer rejection")
	}
}

func TestSubmitAnswer_RejectsDuplicateClaimLink(t *testing.T) {
	_, ms, bank, kk, ctx := setup(t)
	asker, _ := sdk.AccAddressFromBech32(testAddr("asker"))
	answerer, _ := sdk.AccAddressFromBech32(testAddr("answerer"))
	bank.fund(asker, 10_000_000)
	r1, _ := ms.SubmitInquiry(ctx, &types.MsgSubmitInquiry{
		Asker: asker.String(), Question: "q1", Domain: "physics", Bounty: "2000000",
	})
	r2, _ := ms.SubmitInquiry(ctx, &types.MsgSubmitInquiry{
		Asker: asker.String(), Question: "q2", Domain: "physics", Bounty: "2000000",
	})
	kk.claims["claim-1"] = answerer.String()
	if _, err := ms.SubmitAnswer(ctx, &types.MsgSubmitAnswer{
		Answerer: answerer.String(), InquiryId: r1.InquiryId, ClaimId: "claim-1",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := ms.SubmitAnswer(ctx, &types.MsgSubmitAnswer{
		Answerer: answerer.String(), InquiryId: r2.InquiryId, ClaimId: "claim-1",
	}); err == nil {
		t.Fatal("expected duplicate claim-link rejection")
	}
}

func TestResolveInquiry_PaysWinnerWhenClaimAccepted(t *testing.T) {
	k, ms, bank, kk, ctx := setup(t)
	asker, _ := sdk.AccAddressFromBech32(testAddr("asker"))
	answerer, _ := sdk.AccAddressFromBech32(testAddr("answerer"))
	bank.fund(asker, 5_000_000)
	resp, err := ms.SubmitInquiry(ctx, &types.MsgSubmitInquiry{
		Asker: asker.String(), Question: "q", Domain: "physics", Bounty: "2000000",
	})
	if err != nil {
		t.Fatal(err)
	}
	kk.claims["claim-1"] = answerer.String()
	if _, err := ms.SubmitAnswer(ctx, &types.MsgSubmitAnswer{
		Answerer: answerer.String(), InquiryId: resp.InquiryId, ClaimId: "claim-1",
	}); err != nil {
		t.Fatal(err)
	}
	// Mark claim-1 as having produced an accepted fact.
	kk.acceptedFor["claim-1"] = "fact-42"

	r, err := ms.ResolveInquiry(ctx, &types.MsgResolveInquiry{
		Caller: answerer.String(), InquiryId: resp.InquiryId,
	})
	if err != nil {
		t.Fatal(err)
	}
	if r.Status != types.InquiryStatus_INQUIRY_STATUS_RESOLVED {
		t.Fatalf("status=%s want RESOLVED", r.Status)
	}
	if r.WinningFactId != "fact-42" {
		t.Fatalf("winning_fact_id=%s want fact-42", r.WinningFactId)
	}
	if bank.accounts[answerer.String()] != 2_000_000 {
		t.Fatalf("answerer not paid: balance %d", bank.accounts[answerer.String()])
	}
	// Idempotent: re-resolving fails.
	if _, err := ms.ResolveInquiry(ctx, &types.MsgResolveInquiry{
		Caller: answerer.String(), InquiryId: resp.InquiryId,
	}); err == nil {
		t.Fatal("expected re-resolve to fail")
	}
	_ = k
}

func TestResolveInquiry_RefundsOnExpiry(t *testing.T) {
	k, ms, bank, _, ctx := setup(t)
	asker, _ := sdk.AccAddressFromBech32(testAddr("asker"))
	bank.fund(asker, 5_000_000)
	resp, err := ms.SubmitInquiry(ctx, &types.MsgSubmitInquiry{
		Asker: asker.String(), Question: "q", Domain: "physics",
		Bounty: "2000000", ExpiryBlocks: 1, // expires next block
	})
	if err != nil {
		t.Fatal(err)
	}
	// Advance past expiry.
	expiredCtx := ctx.WithBlockHeight(int64(currentBlockUint(ctx) + 10))

	r, err := ms.ResolveInquiry(expiredCtx, &types.MsgResolveInquiry{
		Caller: asker.String(), InquiryId: resp.InquiryId,
	})
	if err != nil {
		t.Fatal(err)
	}
	if r.Status != types.InquiryStatus_INQUIRY_STATUS_EXPIRED {
		t.Fatalf("status=%s want EXPIRED", r.Status)
	}
	if bank.accounts[asker.String()] != 5_000_000 {
		t.Fatalf("asker not refunded on expiry: balance %d", bank.accounts[asker.String()])
	}
	_ = k
}

func currentBlockUint(ctx sdk.Context) uint64 {
	h := ctx.BlockHeight()
	if h < 0 {
		return 0
	}
	return uint64(h)
}
