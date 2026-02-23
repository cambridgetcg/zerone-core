package keeper_test

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"testing"

	dbm "github.com/cosmos/cosmos-db"

	"cosmossdk.io/log"
	"cosmossdk.io/store"
	storemetrics "cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"

	"github.com/zerone-chain/zerone/x/qualification/keeper"
	"github.com/zerone-chain/zerone/x/qualification/types"
)

func init() {
	config := sdk.GetConfig()
	config.SetBech32PrefixForAccount("zrn", "zrnpub")
	config.SetBech32PrefixForValidator("zrnvaloper", "zrnvaloperpub")
	config.SetBech32PrefixForConsensusNode("zrnvalcons", "zrnvalconspub")
}

func testAddr(name string) string {
	h := sha256.Sum256([]byte("test_seed:" + name))
	return sdk.AccAddress(h[:20]).String()
}

// ---------- Mock Keepers ----------

type mockBankKeeper struct {
	balances       map[string]map[string]int64
	moduleBalances map[string]map[string]int64
}

func newMockBankKeeper() *mockBankKeeper {
	return &mockBankKeeper{
		balances:       make(map[string]map[string]int64),
		moduleBalances: make(map[string]map[string]int64),
	}
}

func (m *mockBankKeeper) setBalance(addr string, denom string, amount int64) {
	if m.balances[addr] == nil {
		m.balances[addr] = make(map[string]int64)
	}
	m.balances[addr][denom] = amount
}

func (m *mockBankKeeper) SendCoinsFromAccountToModule(_ context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error {
	sender := senderAddr.String()
	for _, coin := range amt {
		if m.balances[sender] == nil || m.balances[sender][coin.Denom] < coin.Amount.Int64() {
			return fmt.Errorf("insufficient funds")
		}
		m.balances[sender][coin.Denom] -= coin.Amount.Int64()
		if m.moduleBalances[recipientModule] == nil {
			m.moduleBalances[recipientModule] = make(map[string]int64)
		}
		m.moduleBalances[recipientModule][coin.Denom] += coin.Amount.Int64()
	}
	return nil
}

func (m *mockBankKeeper) SendCoinsFromModuleToAccount(_ context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error {
	recipient := recipientAddr.String()
	for _, coin := range amt {
		if m.moduleBalances[senderModule] == nil || m.moduleBalances[senderModule][coin.Denom] < coin.Amount.Int64() {
			return fmt.Errorf("insufficient module funds")
		}
		m.moduleBalances[senderModule][coin.Denom] -= coin.Amount.Int64()
		if m.balances[recipient] == nil {
			m.balances[recipient] = make(map[string]int64)
		}
		m.balances[recipient][coin.Denom] += coin.Amount.Int64()
	}
	return nil
}

type mockStakingKeeper struct {
	validators map[string]bool
}

func newMockStakingKeeper() *mockStakingKeeper {
	return &mockStakingKeeper{validators: make(map[string]bool)}
}

func (m *mockStakingKeeper) addValidator(addr string) {
	m.validators[addr] = true
}

func (m *mockStakingKeeper) IsValidator(_ context.Context, addr string) bool {
	return m.validators[addr]
}

type mockCaptureDefenseKeeper struct {
	reputations map[string]*types.DomainReputation
}

func newMockCaptureDefenseKeeper() *mockCaptureDefenseKeeper {
	return &mockCaptureDefenseKeeper{reputations: make(map[string]*types.DomainReputation)}
}

func (m *mockCaptureDefenseKeeper) setReputation(validator, domain string, score uint64) {
	m.reputations[validator+"/"+domain] = &types.DomainReputation{Score: score, TotalStake: "0"}
}

func (m *mockCaptureDefenseKeeper) GetDomainReputation(_ context.Context, validator string, domain string) (*types.DomainReputation, bool) {
	rep, ok := m.reputations[validator+"/"+domain]
	return rep, ok
}

// ---------- Setup Helpers ----------

func setupKeeper(t *testing.T) (keeper.Keeper, sdk.Context, *mockBankKeeper, *mockStakingKeeper) {
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

	mockBK := newMockBankKeeper()
	mockSK := newMockStakingKeeper()

	storeService := runtime.NewKVStoreService(storeKey)
	k := keeper.NewKeeper(storeService, cdc, "zrn1authority", mockBK, mockSK)

	ctx := sdk.NewContext(stateStore, cmtproto.Header{Height: 100, ChainID: "zerone-test-1"}, false, log.NewNopLogger())

	return k, ctx, mockBK, mockSK
}

func setupMsgServer(t *testing.T) (types.MsgServer, keeper.Keeper, sdk.Context, *mockBankKeeper, *mockStakingKeeper) {
	t.Helper()
	k, ctx, bk, sk := setupKeeper(t)
	return keeper.NewMsgServerImpl(k), k, ctx, bk, sk
}

// ---------- State CRUD Tests ----------

func TestSetGetQualification(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	q := &types.DomainQualification{
		Validator: testAddr("val1"),
		Domain:    "mathematics",
		Pathway:   types.QualificationPathway_QUALIFICATION_PATHWAY_STAKE,
		Status:    types.QualificationStatus_QUALIFICATION_STATUS_ACTIVE,
		Weight:    50,
		GrantedAt: 100,
		ExpiresAt: 2000,
	}
	k.SetQualification(ctx, q)

	got, found := k.GetQualification(ctx, q.Validator, q.Domain)
	if !found {
		t.Fatal("qualification not found")
	}
	if got.Validator != q.Validator || got.Domain != q.Domain {
		t.Fatalf("expected %s/%s, got %s/%s", q.Validator, q.Domain, got.Validator, got.Domain)
	}
	if got.Weight != 50 {
		t.Fatalf("expected weight 50, got %d", got.Weight)
	}
	if got.Status != types.QualificationStatus_QUALIFICATION_STATUS_ACTIVE {
		t.Fatalf("expected ACTIVE status, got %d", got.Status)
	}
}

func TestDeleteQualification(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	q := &types.DomainQualification{
		Validator: testAddr("val1"),
		Domain:    "physics",
		Status:    types.QualificationStatus_QUALIFICATION_STATUS_ACTIVE,
		Weight:    40,
	}
	k.SetQualification(ctx, q)

	k.DeleteQualification(ctx, q.Validator, q.Domain)

	_, found := k.GetQualification(ctx, q.Validator, q.Domain)
	if found {
		t.Fatal("qualification should have been deleted")
	}
}

func TestGetAllQualifications(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	for i := 0; i < 5; i++ {
		q := &types.DomainQualification{
			Validator: testAddr("val1"),
			Domain:    fmt.Sprintf("domain-%d", i),
			Status:    types.QualificationStatus_QUALIFICATION_STATUS_ACTIVE,
			Weight:    uint32(30 + i),
		}
		k.SetQualification(ctx, q)
	}

	all := k.GetAllQualifications(ctx)
	if len(all) != 5 {
		t.Fatalf("expected 5 qualifications, got %d", len(all))
	}
}

func TestDomainValidatorIndex(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	val1 := testAddr("val1")
	val2 := testAddr("val2")

	k.SetQualification(ctx, &types.DomainQualification{
		Validator: val1,
		Domain:    "chemistry",
		Status:    types.QualificationStatus_QUALIFICATION_STATUS_ACTIVE,
	})
	k.SetQualification(ctx, &types.DomainQualification{
		Validator: val2,
		Domain:    "chemistry",
		Status:    types.QualificationStatus_QUALIFICATION_STATUS_ACTIVE,
	})

	validators := k.GetValidatorsByDomain(ctx, "chemistry")
	if len(validators) != 2 {
		t.Fatalf("expected 2 validators in domain, got %d", len(validators))
	}
}

func TestEndorsementCRUD(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	val := testAddr("val1")
	endorser := testAddr("endorser1")

	// Set up qualification first.
	k.SetQualification(ctx, &types.DomainQualification{
		Validator: val,
		Domain:    "biology",
		Status:    types.QualificationStatus_QUALIFICATION_STATUS_ACTIVE,
	})

	id := k.GetNextEndorsementID(ctx)
	e := &types.QualificationEndorsement{
		Id:                     id,
		QualificationValidator: val,
		QualificationDomain:    "biology",
		Endorser:               endorser,
		Reason:                 "excellent work",
		Weight:                 75,
		CreatedAt:              100,
		ExpiresAt:              2000,
	}
	k.SetEndorsement(ctx, e)

	got, found := k.GetEndorsement(ctx, id)
	if !found {
		t.Fatal("endorsement not found")
	}
	if got.Endorser != endorser {
		t.Fatalf("expected endorser %s, got %s", endorser, got.Endorser)
	}

	// Test GetEndorsementsByTarget.
	endorsements := k.GetEndorsementsByTarget(ctx, val, "biology")
	if len(endorsements) != 1 {
		t.Fatalf("expected 1 endorsement, got %d", len(endorsements))
	}

	// Delete.
	k.DeleteEndorsement(ctx, e)
	_, found = k.GetEndorsement(ctx, id)
	if found {
		t.Fatal("endorsement should have been deleted")
	}
}

func TestEndorsementCounter(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	id1 := k.GetNextEndorsementID(ctx)
	id2 := k.GetNextEndorsementID(ctx)
	id3 := k.GetNextEndorsementID(ctx)

	if id1 != 1 || id2 != 2 || id3 != 3 {
		t.Fatalf("expected IDs 1,2,3, got %d,%d,%d", id1, id2, id3)
	}
}

// ---------- Params Tests ----------

func TestParamsSetGet(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	params := types.DefaultParams()
	params.MinStakeAmount = "999999"
	k.SetParams(ctx, params)

	got := k.GetParams(ctx)
	if got.MinStakeAmount != "999999" {
		t.Fatalf("expected min_stake 999999, got %s", got.MinStakeAmount)
	}
}

// ---------- Stake Pathway Tests ----------

func TestQualifyByStakeHappyPath(t *testing.T) {
	k, ctx, bk, _ := setupKeeper(t)

	val := testAddr("val1")
	bk.setBalance(val, "uzrn", 200_000_000)

	err := k.QualifyByStake(ctx, val, "mathematics", "100000000")
	if err != nil {
		t.Fatalf("QualifyByStake failed: %v", err)
	}

	q, found := k.GetQualification(ctx, val, "mathematics")
	if !found {
		t.Fatal("qualification not found after stake")
	}
	if q.Pathway != types.QualificationPathway_QUALIFICATION_PATHWAY_STAKE {
		t.Fatalf("expected STAKE pathway, got %d", q.Pathway)
	}
	if q.Status != types.QualificationStatus_QUALIFICATION_STATUS_ACTIVE {
		t.Fatalf("expected ACTIVE status, got %d", q.Status)
	}
	if q.Weight != 50 {
		t.Fatalf("expected weight 50, got %d", q.Weight)
	}
	if q.StakedAmount != "100000000" {
		t.Fatalf("expected staked amount 100000000, got %s", q.StakedAmount)
	}

	// Verify bank deduction.
	if bk.balances[val]["uzrn"] != 100_000_000 {
		t.Fatalf("expected 100M remaining, got %d", bk.balances[val]["uzrn"])
	}
}

func TestQualifyByStakeInsufficientFunds(t *testing.T) {
	k, ctx, bk, _ := setupKeeper(t)

	val := testAddr("val1")
	bk.setBalance(val, "uzrn", 50_000_000) // less than minimum 100M

	err := k.QualifyByStake(ctx, val, "mathematics", "100000000")
	if err == nil {
		t.Fatal("expected error for insufficient balance")
	}
}

func TestQualifyByStakeBelowMinimum(t *testing.T) {
	k, ctx, bk, _ := setupKeeper(t)

	val := testAddr("val1")
	bk.setBalance(val, "uzrn", 200_000_000)

	err := k.QualifyByStake(ctx, val, "mathematics", "50000000")
	if err == nil {
		t.Fatal("expected error for stake below minimum")
	}
}

func TestQualifyByStakeDuplicate(t *testing.T) {
	k, ctx, bk, _ := setupKeeper(t)

	val := testAddr("val1")
	bk.setBalance(val, "uzrn", 400_000_000)

	err := k.QualifyByStake(ctx, val, "mathematics", "100000000")
	if err != nil {
		t.Fatalf("first QualifyByStake failed: %v", err)
	}

	err = k.QualifyByStake(ctx, val, "mathematics", "100000000")
	if err == nil {
		t.Fatal("expected error for duplicate qualification")
	}
}

// ---------- Track Record Pathway Tests ----------

func TestQualifyByTrackRecordInsufficientVerifications(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	val := testAddr("val1")

	// No prior metrics: should fail.
	err := k.QualifyByTrackRecord(ctx, val, "physics")
	if err == nil {
		t.Fatal("expected error for insufficient track record")
	}
}

// ---------- Cross-Reference Pathway Tests ----------

func TestQualifyByCrossReferenceHappyPath(t *testing.T) {
	k, ctx, bk, _ := setupKeeper(t)

	val := testAddr("val1")
	bk.setBalance(val, "uzrn", 200_000_000)

	// First, qualify in source domain.
	err := k.QualifyByStake(ctx, val, "mathematics", "100000000")
	if err != nil {
		t.Fatalf("QualifyByStake failed: %v", err)
	}

	// Cross-reference to physics.
	err = k.QualifyByCrossReference(ctx, val, "physics", "mathematics")
	if err != nil {
		t.Fatalf("QualifyByCrossReference failed: %v", err)
	}

	q, found := k.GetQualification(ctx, val, "physics")
	if !found {
		t.Fatal("cross-ref qualification not found")
	}
	if q.Pathway != types.QualificationPathway_QUALIFICATION_PATHWAY_CROSS_REFERENCE {
		t.Fatalf("expected CROSS_REFERENCE pathway, got %d", q.Pathway)
	}
	if q.CrossRefDomain != "mathematics" {
		t.Fatalf("expected cross_ref_domain mathematics, got %s", q.CrossRefDomain)
	}
	// Weight should be discounted: 50 * (1000000 - 200000) / 1000000 = 40
	if q.Weight != 40 {
		t.Fatalf("expected discounted weight 40, got %d", q.Weight)
	}
}

func TestQualifyByCrossReferenceNoSourceQualification(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	val := testAddr("val1")

	err := k.QualifyByCrossReference(ctx, val, "physics", "mathematics")
	if err == nil {
		t.Fatal("expected error for no source qualification")
	}
}

func TestQualifyByCrossReferenceWeightTooLow(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	val := testAddr("val1")

	// Manually set a low-weight qualification.
	k.SetQualification(ctx, &types.DomainQualification{
		Validator: val,
		Domain:    "mathematics",
		Status:    types.QualificationStatus_QUALIFICATION_STATUS_ACTIVE,
		Weight:    10, // Below CrossRefMinWeight of 30
	})

	err := k.QualifyByCrossReference(ctx, val, "physics", "mathematics")
	if err == nil {
		t.Fatal("expected error for weight too low")
	}
}

// ---------- Inheritance Pathway Tests ----------

func TestQualifyByInheritanceHappyPath(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	val := testAddr("val1")

	// Set parent domain qualification with lower stratum.
	k.SetQualification(ctx, &types.DomainQualification{
		Validator: val,
		Domain:    "science",
		Pathway:   types.QualificationPathway_QUALIFICATION_PATHWAY_STAKE,
		Status:    types.QualificationStatus_QUALIFICATION_STATUS_ACTIVE,
		Weight:    70,
		Stratum:   0, // More foundational
	})

	err := k.QualifyByInheritance(ctx, val, "physics", "science")
	if err != nil {
		t.Fatalf("QualifyByInheritance failed: %v", err)
	}

	q, found := k.GetQualification(ctx, val, "physics")
	if !found {
		t.Fatal("inheritance qualification not found")
	}
	if q.Pathway != types.QualificationPathway_QUALIFICATION_PATHWAY_INHERITANCE {
		t.Fatalf("expected INHERITANCE pathway, got %d", q.Pathway)
	}
	if q.ParentDomain != "science" {
		t.Fatalf("expected parent_domain science, got %s", q.ParentDomain)
	}
	// Weight should be discounted: 70 * (1000000 - 300000) / 1000000 = 49
	if q.Weight != 49 {
		t.Fatalf("expected discounted weight 49, got %d", q.Weight)
	}
}

func TestQualifyByInheritanceNoParent(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	val := testAddr("val1")

	err := k.QualifyByInheritance(ctx, val, "physics", "science")
	if err == nil {
		t.Fatal("expected error for no parent qualification")
	}
}

// ---------- Status Transition Tests ----------

func TestStatusTransitionExpiry(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	val := testAddr("val1")

	// Create a qualification that expires at block 200.
	k.SetQualification(ctx, &types.DomainQualification{
		Validator: val,
		Domain:    "biology",
		Pathway:   types.QualificationPathway_QUALIFICATION_PATHWAY_STAKE,
		Status:    types.QualificationStatus_QUALIFICATION_STATUS_ACTIVE,
		Weight:    50,
		GrantedAt: 50,
		ExpiresAt: 200,
	})

	// Run BeginBlocker at block 200.
	expireCtx := ctx.WithBlockHeight(200)
	err := k.BeginBlocker(expireCtx)
	if err != nil {
		t.Fatalf("BeginBlocker failed: %v", err)
	}

	q, found := k.GetQualification(expireCtx, val, "biology")
	if !found {
		t.Fatal("qualification should still exist after expiry")
	}
	if q.Status != types.QualificationStatus_QUALIFICATION_STATUS_EXPIRED {
		t.Fatalf("expected EXPIRED status, got %d", q.Status)
	}
}

func TestStatusTransitionProbationToActive(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	val := testAddr("val1")
	params := k.GetParams(ctx)

	// Create a probationary qualification with good metrics.
	k.SetQualification(ctx, &types.DomainQualification{
		Validator:      val,
		Domain:         "biology",
		Pathway:        types.QualificationPathway_QUALIFICATION_PATHWAY_TRACK_RECORD,
		Status:         types.QualificationStatus_QUALIFICATION_STATUS_PROBATIONARY,
		Weight:         40,
		GrantedAt:      50,
		ExpiresAt:      50000,
		ProbationUntil: 200,
		Metrics: &types.QualificationMetrics{
			TotalVerifications:   200,
			CorrectVerifications: 180,
			AccuracyBps:         900000,
		},
	})

	// Run BeginBlocker at block 200 (probation ends).
	promoteCtx := ctx.WithBlockHeight(200)
	_ = params
	err := k.BeginBlocker(promoteCtx)
	if err != nil {
		t.Fatalf("BeginBlocker failed: %v", err)
	}

	q, found := k.GetQualification(promoteCtx, val, "biology")
	if !found {
		t.Fatal("qualification not found")
	}
	if q.Status != types.QualificationStatus_QUALIFICATION_STATUS_ACTIVE {
		t.Fatalf("expected ACTIVE after promotion, got %d", q.Status)
	}
	if q.ProbationUntil != 0 {
		t.Fatalf("expected probation_until reset to 0, got %d", q.ProbationUntil)
	}
}

func TestStatusTransitionProbationToSuspended(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	val := testAddr("val1")

	// Probationary with bad metrics (below min accuracy).
	k.SetQualification(ctx, &types.DomainQualification{
		Validator:      val,
		Domain:         "biology",
		Status:         types.QualificationStatus_QUALIFICATION_STATUS_PROBATIONARY,
		Weight:         40,
		GrantedAt:      50,
		ExpiresAt:      50000,
		ProbationUntil: 200,
		Metrics: &types.QualificationMetrics{
			TotalVerifications:   200,
			CorrectVerifications: 100,
			AccuracyBps:         500000, // Below min 800000
		},
	})

	promoteCtx := ctx.WithBlockHeight(200)
	err := k.BeginBlocker(promoteCtx)
	if err != nil {
		t.Fatalf("BeginBlocker failed: %v", err)
	}

	q, found := k.GetQualification(promoteCtx, val, "biology")
	if !found {
		t.Fatal("qualification not found")
	}
	if q.Status != types.QualificationStatus_QUALIFICATION_STATUS_SUSPENDED {
		t.Fatalf("expected SUSPENDED, got %d", q.Status)
	}
}

// ---------- BeginBlocker Tests ----------

func TestBeginBlockerEndorsementExpiry(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	val := testAddr("val1")
	endorser := testAddr("endorser1")

	k.SetQualification(ctx, &types.DomainQualification{
		Validator:        val,
		Domain:           "biology",
		Status:           types.QualificationStatus_QUALIFICATION_STATUS_ACTIVE,
		EndorsementCount: 1,
		ExpiresAt:        50000,
	})

	k.SetEndorsement(ctx, &types.QualificationEndorsement{
		Id:                     1,
		QualificationValidator: val,
		QualificationDomain:    "biology",
		Endorser:               endorser,
		Weight:                 50,
		CreatedAt:              100,
		ExpiresAt:              200, // Expires at block 200
	})

	// Manually set counter so GetNextEndorsementID doesn't reuse ID 1.
	// (SetEndorsement doesn't auto-increment counter.)

	expireCtx := ctx.WithBlockHeight(200)
	err := k.BeginBlocker(expireCtx)
	if err != nil {
		t.Fatalf("BeginBlocker failed: %v", err)
	}

	_, found := k.GetEndorsement(expireCtx, 1)
	if found {
		t.Fatal("endorsement should have expired")
	}

	q, found := k.GetQualification(expireCtx, val, "biology")
	if !found {
		t.Fatal("qualification not found")
	}
	if q.EndorsementCount != 0 {
		t.Fatalf("expected endorsement count 0 after expiry, got %d", q.EndorsementCount)
	}
}

func TestBeginBlockerStakeUnlockOnExpiry(t *testing.T) {
	k, ctx, bk, _ := setupKeeper(t)

	val := testAddr("val1")
	bk.moduleBalances[types.ModuleName] = map[string]int64{"uzrn": 100_000_000}

	params := k.GetParams(ctx)
	grantedAt := uint64(50)
	expiresAt := grantedAt + params.QualificationPeriod

	k.SetQualification(ctx, &types.DomainQualification{
		Validator:    val,
		Domain:       "mathematics",
		Pathway:      types.QualificationPathway_QUALIFICATION_PATHWAY_STAKE,
		Status:       types.QualificationStatus_QUALIFICATION_STATUS_ACTIVE,
		Weight:       50,
		StakedAmount: "100000000",
		GrantedAt:    grantedAt,
		ExpiresAt:    expiresAt,
	})

	// Run at expiry block, after stake lock period.
	expireBlock := int64(expiresAt)
	if uint64(expireBlock) < grantedAt+params.StakeLockPeriod {
		expireBlock = int64(grantedAt + params.StakeLockPeriod)
	}
	expireCtx := ctx.WithBlockHeight(expireBlock)
	err := k.BeginBlocker(expireCtx)
	if err != nil {
		t.Fatalf("BeginBlocker failed: %v", err)
	}

	// Verify stake returned to validator.
	if bk.balances[val]["uzrn"] != 100_000_000 {
		t.Fatalf("expected 100M returned to validator, got %d", bk.balances[val]["uzrn"])
	}
}

// ---------- Renewal Tests ----------

func TestRenewQualificationHappyPath(t *testing.T) {
	msgSrv, k, ctx, bk, _ := setupMsgServer(t)

	val := testAddr("val1")
	bk.setBalance(val, "uzrn", 200_000_000)

	err := k.QualifyByStake(ctx, val, "mathematics", "100000000")
	if err != nil {
		t.Fatalf("QualifyByStake failed: %v", err)
	}

	params := k.GetParams(ctx)
	q, _ := k.GetQualification(ctx, val, "mathematics")

	// Advance to renewal window.
	renewBlock := int64(q.ExpiresAt - params.RenewalWindow)
	renewCtx := ctx.WithBlockHeight(renewBlock)

	_, err = msgSrv.RenewQualification(renewCtx, &types.MsgRenewQualification{
		Validator: val,
		Domain:    "mathematics",
	})
	if err != nil {
		t.Fatalf("RenewQualification failed: %v", err)
	}

	renewed, _ := k.GetQualification(renewCtx, val, "mathematics")
	if renewed.ExpiresAt != uint64(renewBlock)+params.QualificationPeriod {
		t.Fatalf("expected new expiry %d, got %d", uint64(renewBlock)+params.QualificationPeriod, renewed.ExpiresAt)
	}
	if renewed.RenewedAt != uint64(renewBlock) {
		t.Fatalf("expected renewed_at %d, got %d", renewBlock, renewed.RenewedAt)
	}
}

func TestRenewQualificationTooEarly(t *testing.T) {
	msgSrv, k, ctx, bk, _ := setupMsgServer(t)

	val := testAddr("val1")
	bk.setBalance(val, "uzrn", 200_000_000)

	err := k.QualifyByStake(ctx, val, "mathematics", "100000000")
	if err != nil {
		t.Fatalf("QualifyByStake failed: %v", err)
	}

	// Try to renew immediately (too early).
	_, err = msgSrv.RenewQualification(ctx, &types.MsgRenewQualification{
		Validator: val,
		Domain:    "mathematics",
	})
	if err == nil {
		t.Fatal("expected error for too early renewal")
	}
}

// ---------- Withdraw Tests ----------

func TestWithdrawQualificationHappyPath(t *testing.T) {
	msgSrv, k, ctx, bk, _ := setupMsgServer(t)

	val := testAddr("val1")
	bk.setBalance(val, "uzrn", 200_000_000)

	err := k.QualifyByStake(ctx, val, "mathematics", "100000000")
	if err != nil {
		t.Fatalf("QualifyByStake failed: %v", err)
	}

	params := k.GetParams(ctx)
	q, _ := k.GetQualification(ctx, val, "mathematics")

	// Advance past stake lock period.
	unlockBlock := int64(q.GrantedAt + params.StakeLockPeriod + 1)
	withdrawCtx := ctx.WithBlockHeight(unlockBlock)

	_, err = msgSrv.WithdrawQualification(withdrawCtx, &types.MsgWithdrawQualification{
		Validator: val,
		Domain:    "mathematics",
	})
	if err != nil {
		t.Fatalf("WithdrawQualification failed: %v", err)
	}

	_, found := k.GetQualification(withdrawCtx, val, "mathematics")
	if found {
		t.Fatal("qualification should be deleted after withdrawal")
	}

	// Verify stake returned.
	if bk.balances[val]["uzrn"] != 200_000_000 {
		t.Fatalf("expected all 200M returned, got %d", bk.balances[val]["uzrn"])
	}
}

func TestWithdrawQualificationStakeLocked(t *testing.T) {
	msgSrv, k, ctx, bk, _ := setupMsgServer(t)

	val := testAddr("val1")
	bk.setBalance(val, "uzrn", 200_000_000)

	err := k.QualifyByStake(ctx, val, "mathematics", "100000000")
	if err != nil {
		t.Fatalf("QualifyByStake failed: %v", err)
	}

	// Try to withdraw immediately (stake locked).
	_, err = msgSrv.WithdrawQualification(ctx, &types.MsgWithdrawQualification{
		Validator: val,
		Domain:    "mathematics",
	})
	if err == nil {
		t.Fatal("expected error for locked stake")
	}
}

// ---------- Cross-Module Interface Tests ----------

func TestIsQualified(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	val := testAddr("val1")

	// Not qualified yet.
	if k.IsQualified(ctx, val, "mathematics") {
		t.Fatal("should not be qualified")
	}

	k.SetQualification(ctx, &types.DomainQualification{
		Validator: val,
		Domain:    "mathematics",
		Status:    types.QualificationStatus_QUALIFICATION_STATUS_ACTIVE,
	})

	if !k.IsQualified(ctx, val, "mathematics") {
		t.Fatal("should be qualified")
	}

	// Suspended qualification should not count.
	k.SetQualification(ctx, &types.DomainQualification{
		Validator: val,
		Domain:    "physics",
		Status:    types.QualificationStatus_QUALIFICATION_STATUS_SUSPENDED,
	})

	if k.IsQualified(ctx, val, "physics") {
		t.Fatal("suspended qualification should not count as qualified")
	}
}

func TestGetQualificationWeight(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	val := testAddr("val1")

	if w := k.GetQualificationWeight(ctx, val, "math"); w != 0 {
		t.Fatalf("expected 0 weight for non-existent, got %d", w)
	}

	k.SetQualification(ctx, &types.DomainQualification{
		Validator: val,
		Domain:    "math",
		Status:    types.QualificationStatus_QUALIFICATION_STATUS_ACTIVE,
		Weight:    75,
	})

	if w := k.GetQualificationWeight(ctx, val, "math"); w != 75 {
		t.Fatalf("expected weight 75, got %d", w)
	}
}

func TestRecordVerificationOutcome(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	val := testAddr("val1")

	k.SetQualification(ctx, &types.DomainQualification{
		Validator: val,
		Domain:    "biology",
		Status:    types.QualificationStatus_QUALIFICATION_STATUS_ACTIVE,
	})

	// Record 10 outcomes: 8 correct, 2 incorrect.
	for i := 0; i < 8; i++ {
		err := k.RecordVerificationOutcome(ctx, val, "biology", true)
		if err != nil {
			t.Fatalf("RecordVerificationOutcome correct failed: %v", err)
		}
	}
	for i := 0; i < 2; i++ {
		err := k.RecordVerificationOutcome(ctx, val, "biology", false)
		if err != nil {
			t.Fatalf("RecordVerificationOutcome incorrect failed: %v", err)
		}
	}

	q, _ := k.GetQualification(ctx, val, "biology")
	if q.Metrics.TotalVerifications != 10 {
		t.Fatalf("expected 10 total, got %d", q.Metrics.TotalVerifications)
	}
	if q.Metrics.CorrectVerifications != 8 {
		t.Fatalf("expected 8 correct, got %d", q.Metrics.CorrectVerifications)
	}
	// 8/10 = 0.8 = 800000 BPS
	if q.Metrics.AccuracyBps != 800000 {
		t.Fatalf("expected accuracy 800000, got %d", q.Metrics.AccuracyBps)
	}
}

func TestRecordVerificationOutcomeNotFound(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	err := k.RecordVerificationOutcome(ctx, testAddr("val1"), "nonexistent", true)
	if err == nil {
		t.Fatal("expected error for non-existent qualification")
	}
}

func TestGetQualifiedValidators(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	val1 := testAddr("val1")
	val2 := testAddr("val2")
	val3 := testAddr("val3")

	k.SetQualification(ctx, &types.DomainQualification{
		Validator: val1, Domain: "math",
		Status: types.QualificationStatus_QUALIFICATION_STATUS_ACTIVE,
	})
	k.SetQualification(ctx, &types.DomainQualification{
		Validator: val2, Domain: "math",
		Status: types.QualificationStatus_QUALIFICATION_STATUS_ACTIVE,
	})
	k.SetQualification(ctx, &types.DomainQualification{
		Validator: val3, Domain: "math",
		Status: types.QualificationStatus_QUALIFICATION_STATUS_SUSPENDED,
	})

	qualified := k.GetQualifiedValidators(ctx, "math")
	if len(qualified) != 2 {
		t.Fatalf("expected 2 qualified validators, got %d", len(qualified))
	}
}

// ---------- MsgServer Tests ----------

func TestMsgEndorseQualification(t *testing.T) {
	msgSrv, k, ctx, _, _ := setupMsgServer(t)

	val := testAddr("val1")
	endorser := testAddr("endorser1")

	k.SetQualification(ctx, &types.DomainQualification{
		Validator:        val,
		Domain:           "biology",
		Status:           types.QualificationStatus_QUALIFICATION_STATUS_ACTIVE,
		EndorsementCount: 0,
		ExpiresAt:        50000,
	})

	resp, err := msgSrv.EndorseQualification(ctx, &types.MsgEndorseQualification{
		Endorser:  endorser,
		Validator: val,
		Domain:    "biology",
		Weight:    75,
		Reason:    "great validator",
	})
	if err != nil {
		t.Fatalf("EndorseQualification failed: %v", err)
	}
	if resp.EndorsementId == 0 {
		t.Fatal("expected non-zero endorsement ID")
	}

	q, _ := k.GetQualification(ctx, val, "biology")
	if q.EndorsementCount != 1 {
		t.Fatalf("expected endorsement count 1, got %d", q.EndorsementCount)
	}
}

func TestMsgEndorseSelfRejected(t *testing.T) {
	msgSrv, k, ctx, _, _ := setupMsgServer(t)

	val := testAddr("val1")

	k.SetQualification(ctx, &types.DomainQualification{
		Validator: val,
		Domain:    "biology",
		Status:    types.QualificationStatus_QUALIFICATION_STATUS_ACTIVE,
	})

	_, err := msgSrv.EndorseQualification(ctx, &types.MsgEndorseQualification{
		Endorser:  val,
		Validator: val,
		Domain:    "biology",
		Weight:    50,
		Reason:    "self",
	})
	if err == nil {
		t.Fatal("expected error for self-endorsement")
	}
}

func TestMsgUpdateParams(t *testing.T) {
	msgSrv, k, ctx, _, _ := setupMsgServer(t)

	params := types.DefaultParams()
	params.MinStakeAmount = "500000000"

	_, err := msgSrv.UpdateParams(ctx, &types.MsgUpdateParams{
		Authority: "zrn1authority",
		Params:    params,
	})
	if err != nil {
		t.Fatalf("UpdateParams failed: %v", err)
	}

	got := k.GetParams(ctx)
	if got.MinStakeAmount != "500000000" {
		t.Fatalf("expected 500000000, got %s", got.MinStakeAmount)
	}
}

func TestMsgUpdateParamsUnauthorized(t *testing.T) {
	msgSrv, _, ctx, _, _ := setupMsgServer(t)

	_, err := msgSrv.UpdateParams(ctx, &types.MsgUpdateParams{
		Authority: testAddr("imposter"),
		Params:    types.DefaultParams(),
	})
	if err == nil {
		t.Fatal("expected error for unauthorized UpdateParams")
	}
}

// ---------- QueryServer Tests ----------

func TestQueryQualification(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)
	qSrv := keeper.NewQueryServerImpl(k)

	val := testAddr("val1")
	k.SetQualification(ctx, &types.DomainQualification{
		Validator: val,
		Domain:    "chemistry",
		Status:    types.QualificationStatus_QUALIFICATION_STATUS_ACTIVE,
		Weight:    60,
	})

	resp, err := qSrv.Qualification(ctx, &types.QueryQualificationRequest{Validator: val, Domain: "chemistry"})
	if err != nil {
		t.Fatalf("Query Qualification failed: %v", err)
	}
	if resp.Qualification.Weight != 60 {
		t.Fatalf("expected weight 60, got %d", resp.Qualification.Weight)
	}
}

func TestQueryQualificationsByDomain(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)
	qSrv := keeper.NewQueryServerImpl(k)

	for i := 0; i < 3; i++ {
		k.SetQualification(ctx, &types.DomainQualification{
			Validator: testAddr(fmt.Sprintf("val%d", i)),
			Domain:    "ai",
			Status:    types.QualificationStatus_QUALIFICATION_STATUS_ACTIVE,
		})
	}

	resp, err := qSrv.QualificationsByDomain(ctx, &types.QueryByDomainRequest{Domain: "ai"})
	if err != nil {
		t.Fatalf("Query QualificationsByDomain failed: %v", err)
	}
	if len(resp.Qualifications) != 3 {
		t.Fatalf("expected 3 qualifications, got %d", len(resp.Qualifications))
	}
}

func TestQueryParams(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)
	qSrv := keeper.NewQueryServerImpl(k)

	k.SetParams(ctx, types.DefaultParams())

	resp, err := qSrv.Params(ctx, &types.QueryParamsRequest{})
	if err != nil {
		t.Fatalf("Query Params failed: %v", err)
	}
	if resp.Params.MinStakeAmount != "100000000" {
		t.Fatalf("expected default min stake, got %s", resp.Params.MinStakeAmount)
	}
}

// ---------- Max Endorsements Test ----------

func TestMaxEndorsementsExceeded(t *testing.T) {
	msgSrv, k, ctx, _, _ := setupMsgServer(t)

	val := testAddr("val1")

	// Set max endorsements to 2
	params := k.GetParams(ctx)
	params.MaxEndorsements = 2
	k.SetParams(ctx, params)

	k.SetQualification(ctx, &types.DomainQualification{
		Validator:        val,
		Domain:           "biology",
		Status:           types.QualificationStatus_QUALIFICATION_STATUS_ACTIVE,
		EndorsementCount: 0,
		ExpiresAt:        50000,
	})

	// First two endorsements should succeed
	for i := 0; i < 2; i++ {
		endorser := testAddr(fmt.Sprintf("endorser%d", i))
		_, err := msgSrv.EndorseQualification(ctx, &types.MsgEndorseQualification{
			Endorser:  endorser,
			Validator: val,
			Domain:    "biology",
			Weight:    50,
			Reason:    fmt.Sprintf("endorsement %d", i),
		})
		if err != nil {
			t.Fatalf("endorsement %d should succeed: %v", i, err)
		}
	}

	// Third endorsement should fail (max = 2)
	_, err := msgSrv.EndorseQualification(ctx, &types.MsgEndorseQualification{
		Endorser:  testAddr("endorser-extra"),
		Validator: val,
		Domain:    "biology",
		Weight:    50,
		Reason:    "one too many",
	})
	if err == nil {
		t.Fatal("expected error for exceeding max endorsements")
	}
}

// ---------- Revocation Status Test ----------

func TestRevocationStatus(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	val := testAddr("val1")

	// Set qualification with REVOKED status
	k.SetQualification(ctx, &types.DomainQualification{
		Validator: val,
		Domain:    "mathematics",
		Status:    types.QualificationStatus_QUALIFICATION_STATUS_REVOKED,
		Weight:    50,
	})

	// Revoked qualification should NOT count as qualified
	if k.IsQualified(ctx, val, "mathematics") {
		t.Fatal("revoked qualification should not be considered qualified")
	}

	// Weight should be 0 for revoked
	if w := k.GetQualificationWeight(ctx, val, "mathematics"); w != 0 {
		t.Fatalf("expected weight 0 for revoked qualification, got %d", w)
	}

	// Verify the status is correctly persisted
	q, found := k.GetQualification(ctx, val, "mathematics")
	if !found {
		t.Fatal("qualification not found")
	}
	if q.Status != types.QualificationStatus_QUALIFICATION_STATUS_REVOKED {
		t.Fatalf("expected REVOKED status, got %d", q.Status)
	}
}

// ---------- Max Qualifications Per Validator ----------

func TestMultiDomainQualification(t *testing.T) {
	k, ctx, bk, _ := setupKeeper(t)

	val := testAddr("val1")
	bk.setBalance(val, "uzrn", 1_000_000_000)

	// Qualify in multiple domains
	domains := []string{"math", "physics", "chemistry", "biology", "cs"}
	for _, domain := range domains {
		err := k.QualifyByStake(ctx, val, domain, "100000000")
		if err != nil {
			t.Fatalf("QualifyByStake for %s failed: %v", domain, err)
		}
	}

	// Verify all qualifications exist
	for _, domain := range domains {
		if !k.IsQualified(ctx, val, domain) {
			t.Fatalf("expected qualified for %s", domain)
		}
	}

	all := k.GetAllQualifications(ctx)
	if len(all) != 5 {
		t.Fatalf("expected 5 qualifications, got %d", len(all))
	}
}

// ---------- Genesis Roundtrip ----------

func TestGenesisRoundtrip(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	val := testAddr("val1")
	endorser := testAddr("endorser1")

	// Set params and state.
	params := types.DefaultParams()
	params.MinStakeAmount = "777777"
	k.SetParams(ctx, params)

	k.SetQualification(ctx, &types.DomainQualification{
		Validator: val,
		Domain:    "chemistry",
		Pathway:   types.QualificationPathway_QUALIFICATION_PATHWAY_STAKE,
		Status:    types.QualificationStatus_QUALIFICATION_STATUS_ACTIVE,
		Weight:    55,
		GrantedAt: 100,
		ExpiresAt: 2000,
	})

	id := k.GetNextEndorsementID(ctx)
	k.SetEndorsement(ctx, &types.QualificationEndorsement{
		Id:                     id,
		QualificationValidator: val,
		QualificationDomain:    "chemistry",
		Endorser:               endorser,
		Weight:                 80,
		CreatedAt:              100,
		ExpiresAt:              2000,
	})

	// Export.
	genState := k.ExportGenesis(ctx)

	// Validate.
	if err := genState.Validate(); err != nil {
		t.Fatalf("genesis validation failed: %v", err)
	}

	// Re-marshal and check.
	bz, err := json.Marshal(genState)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var restored types.GenesisState
	if err := json.Unmarshal(bz, &restored); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	// Create new keeper and import.
	k2, ctx2, _, _ := setupKeeper(t)
	k2.InitGenesis(ctx2, &restored)

	// Verify state restored.
	q, found := k2.GetQualification(ctx2, val, "chemistry")
	if !found {
		t.Fatal("qualification not found after genesis import")
	}
	if q.Weight != 55 {
		t.Fatalf("expected weight 55, got %d", q.Weight)
	}

	e, found := k2.GetEndorsement(ctx2, id)
	if !found {
		t.Fatal("endorsement not found after genesis import")
	}
	if e.Weight != 80 {
		t.Fatalf("expected endorsement weight 80, got %d", e.Weight)
	}

	restoredParams := k2.GetParams(ctx2)
	if restoredParams.MinStakeAmount != "777777" {
		t.Fatalf("expected min_stake 777777, got %s", restoredParams.MinStakeAmount)
	}
}
