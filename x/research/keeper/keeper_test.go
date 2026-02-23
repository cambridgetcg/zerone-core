package keeper_test

import (
	"context"
	"fmt"
	"testing"

	"cosmossdk.io/log"
	sdkmath "cosmossdk.io/math"
	"cosmossdk.io/store"
	storemetrics "cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/x/research/keeper"
	"github.com/zerone-chain/zerone/x/research/types"
)

// -----------------------------------------------------------------------
// Mock BankKeeper
// -----------------------------------------------------------------------

type mockBankKeeper struct {
	balances map[string]sdkmath.Int
	burned   sdkmath.Int
}

func newMockBankKeeper() *mockBankKeeper {
	return &mockBankKeeper{
		balances: make(map[string]sdkmath.Int),
		burned:   sdkmath.ZeroInt(),
	}
}

func (m *mockBankKeeper) setBalance(addr sdk.AccAddress, denom string, amount sdkmath.Int) {
	m.balances[addr.String()+"/"+denom] = amount
}

func (m *mockBankKeeper) SendCoins(_ context.Context, fromAddr sdk.AccAddress, toAddr sdk.AccAddress, amt sdk.Coins) error {
	for _, coin := range amt {
		fromKey := fromAddr.String() + "/" + coin.Denom
		toKey := toAddr.String() + "/" + coin.Denom

		fromBal, ok := m.balances[fromKey]
		if !ok {
			fromBal = sdkmath.ZeroInt()
		}
		if fromBal.LT(coin.Amount) {
			return fmt.Errorf("insufficient balance")
		}
		m.balances[fromKey] = fromBal.Sub(coin.Amount)

		toBal, ok := m.balances[toKey]
		if !ok {
			toBal = sdkmath.ZeroInt()
		}
		m.balances[toKey] = toBal.Add(coin.Amount)
	}
	return nil
}

func (m *mockBankKeeper) SendCoinsFromAccountToModule(_ context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error {
	for _, coin := range amt {
		key := senderAddr.String() + "/" + coin.Denom
		bal, ok := m.balances[key]
		if !ok {
			bal = sdkmath.ZeroInt()
		}
		if bal.LT(coin.Amount) {
			return fmt.Errorf("insufficient balance for send to module")
		}
		m.balances[key] = bal.Sub(coin.Amount)
		modKey := recipientModule + "/" + coin.Denom
		modBal, ok := m.balances[modKey]
		if !ok {
			modBal = sdkmath.ZeroInt()
		}
		m.balances[modKey] = modBal.Add(coin.Amount)
	}
	return nil
}

func (m *mockBankKeeper) SendCoinsFromModuleToAccount(_ context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error {
	for _, coin := range amt {
		modKey := senderModule + "/" + coin.Denom
		modBal, ok := m.balances[modKey]
		if !ok {
			modBal = sdkmath.ZeroInt()
		}
		if modBal.LT(coin.Amount) {
			return fmt.Errorf("insufficient module balance")
		}
		m.balances[modKey] = modBal.Sub(coin.Amount)

		key := recipientAddr.String() + "/" + coin.Denom
		bal, ok := m.balances[key]
		if !ok {
			bal = sdkmath.ZeroInt()
		}
		m.balances[key] = bal.Add(coin.Amount)
	}
	return nil
}

func (m *mockBankKeeper) BurnCoins(_ context.Context, moduleName string, amt sdk.Coins) error {
	for _, coin := range amt {
		key := moduleName + "/" + coin.Denom
		bal, ok := m.balances[key]
		if !ok {
			bal = sdkmath.ZeroInt()
		}
		if bal.LT(coin.Amount) {
			return fmt.Errorf("insufficient balance to burn")
		}
		m.balances[key] = bal.Sub(coin.Amount)
		m.burned = m.burned.Add(coin.Amount)
	}
	return nil
}

// -----------------------------------------------------------------------
// Setup
// -----------------------------------------------------------------------

var testAuthority = sdk.AccAddress([]byte("authority-addr------")).String()

func setupKeeper(t *testing.T) (keeper.Keeper, sdk.Context, *mockBankKeeper) {
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

	bk := newMockBankKeeper()

	k := keeper.NewKeeper(runtime.NewKVStoreService(storeKey), cdc, testAuthority, bk)

	ctx := sdk.NewContext(stateStore, cmtproto.Header{Height: 100}, false, log.NewNopLogger())

	return k, ctx, bk
}

func testAddr(i int) sdk.AccAddress {
	return sdk.AccAddress([]byte(fmt.Sprintf("test-addr-%010d", i)))
}

func testAddrStr(i int) string {
	return testAddr(i).String()
}

// -----------------------------------------------------------------------
// Tests: Research Submission
// -----------------------------------------------------------------------

func TestSubmitResearch(t *testing.T) {
	k, ctx, bk := setupKeeper(t)

	submitter := testAddr(1)
	bk.setBalance(submitter, "uzrn", sdkmath.NewInt(5000000))

	msgServer := keeper.NewMsgServerImpl(k)
	resp, err := msgServer.SubmitResearch(ctx, &types.MsgSubmitResearch{
		Submitter:   submitter.String(),
		Title:       "Test Research",
		Description: "Testing research submission",
		Domain:      "physics",
		Stake:       "1000000",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ResearchId == "" {
		t.Fatal("expected non-empty research_id")
	}

	// Verify stored
	research, found := k.GetResearch(ctx, resp.ResearchId)
	if !found {
		t.Fatal("research not found after submission")
	}
	if research.Status != "submitted" {
		t.Errorf("expected status 'submitted', got %q", research.Status)
	}
	if research.Submitter != submitter.String() {
		t.Errorf("expected submitter %s, got %s", submitter.String(), research.Submitter)
	}
	if research.Stake != "1000000" {
		t.Errorf("expected stake 1000000, got %s", research.Stake)
	}

	// Verify stake was deducted
	remaining := bk.balances[submitter.String()+"/uzrn"]
	if !remaining.Equal(sdkmath.NewInt(4000000)) {
		t.Errorf("expected remaining balance 4000000, got %s", remaining.String())
	}
}

func TestSubmitResearchInsufficientStake(t *testing.T) {
	k, ctx, bk := setupKeeper(t)

	submitter := testAddr(1)
	bk.setBalance(submitter, "uzrn", sdkmath.NewInt(500000)) // less than min stake

	msgServer := keeper.NewMsgServerImpl(k)
	_, err := msgServer.SubmitResearch(ctx, &types.MsgSubmitResearch{
		Submitter:   submitter.String(),
		Title:       "Test Research",
		Description: "Testing",
		Domain:      "physics",
		Stake:       "500000", // below 1000000 minimum
	})
	if err == nil {
		t.Fatal("expected error for insufficient stake, got nil")
	}
}

// -----------------------------------------------------------------------
// Tests: Review
// -----------------------------------------------------------------------

func TestReviewResearch(t *testing.T) {
	k, ctx, bk := setupKeeper(t)

	// Submit research
	submitter := testAddr(1)
	bk.setBalance(submitter, "uzrn", sdkmath.NewInt(5000000))

	msgServer := keeper.NewMsgServerImpl(k)
	resp, _ := msgServer.SubmitResearch(ctx, &types.MsgSubmitResearch{
		Submitter:   submitter.String(),
		Title:       "Test Research",
		Description: "Testing",
		Domain:      "physics",
		Stake:       "1000000",
	})

	// Review
	reviewer := testAddrStr(2)
	_, err := msgServer.ReviewResearch(ctx, &types.MsgReviewResearch{
		Reviewer:     reviewer,
		ResearchId:   resp.ResearchId,
		Verdict:      types.ReviewVerdict_REVIEW_VERDICT_APPROVE,
		Reasoning:    "Good work",
		QualityScore: 80,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify research updated
	research, _ := k.GetResearch(ctx, resp.ResearchId)
	if research.ReviewCount != 1 {
		t.Errorf("expected review_count 1, got %d", research.ReviewCount)
	}
	if research.ApproveCount != 1 {
		t.Errorf("expected approve_count 1, got %d", research.ApproveCount)
	}
	if research.AggregateScore != 80 {
		t.Errorf("expected aggregate_score 80, got %d", research.AggregateScore)
	}
	if research.Status != "under_review" {
		t.Errorf("expected status 'under_review', got %q", research.Status)
	}
}

func TestReviewResearchDuplicate(t *testing.T) {
	k, ctx, bk := setupKeeper(t)

	submitter := testAddr(1)
	bk.setBalance(submitter, "uzrn", sdkmath.NewInt(5000000))

	msgServer := keeper.NewMsgServerImpl(k)
	resp, _ := msgServer.SubmitResearch(ctx, &types.MsgSubmitResearch{
		Submitter:   submitter.String(),
		Title:       "Test",
		Description: "Testing",
		Domain:      "physics",
		Stake:       "1000000",
	})

	reviewer := testAddrStr(2)
	msgServer.ReviewResearch(ctx, &types.MsgReviewResearch{
		Reviewer:     reviewer,
		ResearchId:   resp.ResearchId,
		Verdict:      types.ReviewVerdict_REVIEW_VERDICT_APPROVE,
		Reasoning:    "OK",
		QualityScore: 70,
	})

	// Second review by same reviewer should fail
	_, err := msgServer.ReviewResearch(ctx, &types.MsgReviewResearch{
		Reviewer:     reviewer,
		ResearchId:   resp.ResearchId,
		Verdict:      types.ReviewVerdict_REVIEW_VERDICT_REJECT,
		Reasoning:    "Changed my mind",
		QualityScore: 30,
	})
	if err == nil {
		t.Fatal("expected error for duplicate review, got nil")
	}
}

// -----------------------------------------------------------------------
// Tests: Resolution
// -----------------------------------------------------------------------

func submitAndReview(t *testing.T, k keeper.Keeper, ctx sdk.Context, bk *mockBankKeeper, scores []uint32) string {
	t.Helper()
	submitter := testAddr(1)
	bk.setBalance(submitter, "uzrn", sdkmath.NewInt(10000000))

	msgServer := keeper.NewMsgServerImpl(k)
	resp, err := msgServer.SubmitResearch(ctx, &types.MsgSubmitResearch{
		Submitter:   submitter.String(),
		Title:       "Research for Resolution",
		Description: "Testing resolution",
		Domain:      "math",
		Stake:       "1000000",
	})
	if err != nil {
		t.Fatalf("submit error: %v", err)
	}

	for i, score := range scores {
		reviewer := testAddrStr(100 + i)
		_, err := msgServer.ReviewResearch(ctx, &types.MsgReviewResearch{
			Reviewer:     reviewer,
			ResearchId:   resp.ResearchId,
			Verdict:      types.ReviewVerdict_REVIEW_VERDICT_APPROVE,
			Reasoning:    "Review",
			QualityScore: score,
		})
		if err != nil {
			t.Fatalf("review %d error: %v", i, err)
		}
	}
	return resp.ResearchId
}

func TestResolveResearchAccepted(t *testing.T) {
	k, ctx, bk := setupKeeper(t)

	// 3 reviews with scores averaging >= 70 (threshold)
	researchId := submitAndReview(t, k, ctx, bk, []uint32{80, 75, 70})

	// Pre-resolution: check module has stake
	moduleBal := bk.balances[types.ModuleName+"/uzrn"]
	if moduleBal.LT(sdkmath.NewInt(1000000)) {
		t.Fatal("module should hold stake before resolution")
	}

	submitterBefore := bk.balances[testAddr(1).String()+"/uzrn"]

	msgServer := keeper.NewMsgServerImpl(k)
	resolveResp, err := msgServer.ResolveResearch(ctx, &types.MsgResolveResearch{
		Authority:  testAuthority,
		ResearchId: researchId,
	})
	if err != nil {
		t.Fatalf("resolve error: %v", err)
	}
	if resolveResp.Outcome != types.ResearchOutcome_RESEARCH_OUTCOME_ACCEPTED {
		t.Errorf("expected ACCEPTED outcome, got %v", resolveResp.Outcome)
	}

	// Check research status
	research, _ := k.GetResearch(ctx, researchId)
	if research.Status != "accepted" {
		t.Errorf("expected status 'accepted', got %q", research.Status)
	}

	// Stake returned to submitter
	submitterAfter := bk.balances[testAddr(1).String()+"/uzrn"]
	returned := submitterAfter.Sub(submitterBefore)
	if !returned.Equal(sdkmath.NewInt(1000000)) {
		t.Errorf("expected 1000000 returned, got %s", returned.String())
	}
}

func TestResolveResearchRejected(t *testing.T) {
	k, ctx, bk := setupKeeper(t)

	// 3 reviews with scores averaging < 70 (threshold)
	researchId := submitAndReview(t, k, ctx, bk, []uint32{50, 60, 55})

	bk.burned = sdkmath.ZeroInt()

	msgServer := keeper.NewMsgServerImpl(k)
	resolveResp, err := msgServer.ResolveResearch(ctx, &types.MsgResolveResearch{
		Authority:  testAuthority,
		ResearchId: researchId,
	})
	if err != nil {
		t.Fatalf("resolve error: %v", err)
	}
	if resolveResp.Outcome != types.ResearchOutcome_RESEARCH_OUTCOME_REJECTED {
		t.Errorf("expected REJECTED outcome, got %v", resolveResp.Outcome)
	}

	// Check research status
	research, _ := k.GetResearch(ctx, researchId)
	if research.Status != "rejected" {
		t.Errorf("expected status 'rejected', got %q", research.Status)
	}

	// Some amount burned (33% of 1000000 = 330000)
	if bk.burned.IsZero() {
		t.Error("expected some tokens to be burned on rejection")
	}
}

// -----------------------------------------------------------------------
// Tests: Bounty
// -----------------------------------------------------------------------

func TestCreateBounty(t *testing.T) {
	k, ctx, bk := setupKeeper(t)

	creator := testAddr(1)
	bk.setBalance(creator, "uzrn", sdkmath.NewInt(100000000))

	msgServer := keeper.NewMsgServerImpl(k)
	resp, err := msgServer.CreateBounty(ctx, &types.MsgCreateBounty{
		Creator:        creator.String(),
		Title:          "Find error in paper X",
		Description:    "Investigate",
		Reward:         "5000000",
		DeadlineHeight: uint64(ctx.BlockHeight()) + 50000, // well past minimum
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.BountyId == "" {
		t.Fatal("expected non-empty bounty_id")
	}

	bounty, found := k.GetBounty(ctx, resp.BountyId)
	if !found {
		t.Fatal("bounty not found after creation")
	}
	if bounty.Status != "open" {
		t.Errorf("expected status 'open', got %q", bounty.Status)
	}
	if bounty.Reward != "5000000" {
		t.Errorf("expected reward 5000000, got %s", bounty.Reward)
	}

	// Reward locked from creator
	remaining := bk.balances[creator.String()+"/uzrn"]
	if !remaining.Equal(sdkmath.NewInt(95000000)) {
		t.Errorf("expected 95000000 remaining, got %s", remaining.String())
	}
}

func TestClaimBounty(t *testing.T) {
	k, ctx, bk := setupKeeper(t)

	creator := testAddr(1)
	bk.setBalance(creator, "uzrn", sdkmath.NewInt(100000000))

	msgServer := keeper.NewMsgServerImpl(k)
	createResp, _ := msgServer.CreateBounty(ctx, &types.MsgCreateBounty{
		Creator:        creator.String(),
		Title:          "Bounty",
		Description:    "Test",
		Reward:         "5000000",
		DeadlineHeight: uint64(ctx.BlockHeight()) + 50000,
	})

	claimer := testAddrStr(2)
	_, err := msgServer.ClaimBounty(ctx, &types.MsgClaimBounty{
		Claimer:  claimer,
		BountyId: createResp.BountyId,
	})
	if err != nil {
		t.Fatalf("claim error: %v", err)
	}

	bounty, _ := k.GetBounty(ctx, createResp.BountyId)
	if bounty.Status != "claimed" {
		t.Errorf("expected status 'claimed', got %q", bounty.Status)
	}
	if bounty.ClaimedBy != claimer {
		t.Errorf("expected claimed_by %s, got %s", claimer, bounty.ClaimedBy)
	}
}

func TestFulfillBounty(t *testing.T) {
	k, ctx, bk := setupKeeper(t)

	creator := testAddr(1)
	bk.setBalance(creator, "uzrn", sdkmath.NewInt(100000000))

	msgServer := keeper.NewMsgServerImpl(k)
	createResp, _ := msgServer.CreateBounty(ctx, &types.MsgCreateBounty{
		Creator:        creator.String(),
		Title:          "Bounty",
		Description:    "Test",
		Reward:         "5000000",
		DeadlineHeight: uint64(ctx.BlockHeight()) + 50000,
	})

	claimer := testAddr(2)
	msgServer.ClaimBounty(ctx, &types.MsgClaimBounty{
		Claimer:  claimer.String(),
		BountyId: createResp.BountyId,
	})

	_, err := msgServer.FulfillBounty(ctx, &types.MsgFulfillBounty{
		Authority: testAuthority,
		BountyId:  createResp.BountyId,
		Claimer:   claimer.String(),
	})
	if err != nil {
		t.Fatalf("fulfill error: %v", err)
	}

	bounty, _ := k.GetBounty(ctx, createResp.BountyId)
	if bounty.Status != "fulfilled" {
		t.Errorf("expected status 'fulfilled', got %q", bounty.Status)
	}

	// Reward paid to claimer
	claimerBal := bk.balances[claimer.String()+"/uzrn"]
	if !claimerBal.Equal(sdkmath.NewInt(5000000)) {
		t.Errorf("expected claimer to have 5000000, got %s", claimerBal.String())
	}
}

// -----------------------------------------------------------------------
// Tests: Bounty Expiry
// -----------------------------------------------------------------------

func TestBountyExpiry(t *testing.T) {
	k, ctx, bk := setupKeeper(t)

	creator := testAddr(1)
	bk.setBalance(creator, "uzrn", sdkmath.NewInt(100000000))

	msgServer := keeper.NewMsgServerImpl(k)
	createResp, _ := msgServer.CreateBounty(ctx, &types.MsgCreateBounty{
		Creator:        creator.String(),
		Title:          "Bounty",
		Description:    "Test",
		Reward:         "5000000",
		DeadlineHeight: uint64(ctx.BlockHeight()) + 50000,
	})

	// Also create a claimed bounty
	createResp2, _ := msgServer.CreateBounty(ctx, &types.MsgCreateBounty{
		Creator:        creator.String(),
		Title:          "Bounty 2",
		Description:    "Test 2",
		Reward:         "3000000",
		DeadlineHeight: uint64(ctx.BlockHeight()) + 50000,
	})

	claimer := testAddrStr(2)
	msgServer.ClaimBounty(ctx, &types.MsgClaimBounty{
		Claimer:  claimer,
		BountyId: createResp2.BountyId,
	})

	creatorBalBefore := bk.balances[creator.String()+"/uzrn"]

	// Advance past deadline
	ctx = ctx.WithBlockHeight(int64(uint64(ctx.BlockHeight()) + 50001))
	k.ExpireBounties(ctx)

	// Open bounty should be expired, reward returned
	bounty1, _ := k.GetBounty(ctx, createResp.BountyId)
	if bounty1.Status != "expired" {
		t.Errorf("expected status 'expired', got %q", bounty1.Status)
	}

	creatorBalAfter := bk.balances[creator.String()+"/uzrn"]
	returned := creatorBalAfter.Sub(creatorBalBefore)
	if !returned.Equal(sdkmath.NewInt(5000000)) {
		t.Errorf("expected 5000000 returned to creator, got %s", returned.String())
	}

	// Claimed bounty should be reopened
	bounty2, _ := k.GetBounty(ctx, createResp2.BountyId)
	if bounty2.Status != "open" {
		t.Errorf("expected claimed bounty reopened to 'open', got %q", bounty2.Status)
	}
	if bounty2.ClaimedBy != "" {
		t.Errorf("expected claimed_by cleared, got %q", bounty2.ClaimedBy)
	}
}

// -----------------------------------------------------------------------
// Tests: Genesis Import/Export
// -----------------------------------------------------------------------

func TestGenesisImportExport(t *testing.T) {
	k, ctx, bk := setupKeeper(t)

	// Create some state
	submitter := testAddr(1)
	bk.setBalance(submitter, "uzrn", sdkmath.NewInt(100000000))

	msgServer := keeper.NewMsgServerImpl(k)

	// Submit research
	resp, _ := msgServer.SubmitResearch(ctx, &types.MsgSubmitResearch{
		Submitter:   submitter.String(),
		Title:       "Genesis Test",
		Description: "Testing genesis",
		Domain:      "math",
		Stake:       "1000000",
	})

	// Review it
	msgServer.ReviewResearch(ctx, &types.MsgReviewResearch{
		Reviewer:     testAddrStr(2),
		ResearchId:   resp.ResearchId,
		Verdict:      types.ReviewVerdict_REVIEW_VERDICT_APPROVE,
		Reasoning:    "Good",
		QualityScore: 85,
	})

	// Create a bounty
	msgServer.CreateBounty(ctx, &types.MsgCreateBounty{
		Creator:        submitter.String(),
		Title:          "Test Bounty",
		Description:    "Testing",
		Reward:         "2000000",
		DeadlineHeight: uint64(ctx.BlockHeight()) + 50000,
	})

	// Fund treasury
	bk.setBalance(testAddr(3), "uzrn", sdkmath.NewInt(50000000))
	msgServer.FundResearch(ctx, &types.MsgFundResearch{
		Funder: testAddrStr(3),
		Amount: "5000000",
	})

	// Export genesis
	exportedState := k.ExportGenesis(ctx)

	if exportedState.Params == nil {
		t.Fatal("exported params is nil")
	}
	if len(exportedState.Researches) != 1 {
		t.Errorf("expected 1 research, got %d", len(exportedState.Researches))
	}
	if len(exportedState.Bounties) != 1 {
		t.Errorf("expected 1 bounty, got %d", len(exportedState.Bounties))
	}
	if len(exportedState.PeerReviews) != 1 {
		t.Errorf("expected 1 peer review, got %d", len(exportedState.PeerReviews))
	}
	if exportedState.TreasuryBalance == nil || exportedState.TreasuryBalance.Balance != "5000000" {
		t.Errorf("expected treasury balance 5000000, got %v", exportedState.TreasuryBalance)
	}

	// Re-import into fresh keeper
	k2, ctx2, _ := setupKeeper(t)
	k2.InitGenesis(ctx2, exportedState)

	// Verify imported state
	research, found := k2.GetResearch(ctx2, resp.ResearchId)
	if !found {
		t.Fatal("imported research not found")
	}
	if research.Title != "Genesis Test" {
		t.Errorf("expected title 'Genesis Test', got %q", research.Title)
	}

	treasury := k2.GetTreasuryBalance(ctx2)
	if treasury != "5000000" {
		t.Errorf("expected imported treasury 5000000, got %s", treasury)
	}
}
