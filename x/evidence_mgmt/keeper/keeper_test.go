package keeper_test

import (
	"context"
	"crypto/sha256"
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

	"github.com/zerone-chain/zerone/x/evidence_mgmt/keeper"
	"github.com/zerone-chain/zerone/x/evidence_mgmt/types"
)

func init() {
	config := sdk.GetConfig()
	config.SetBech32PrefixForAccount("zrn", "zrnpub")
	config.SetBech32PrefixForValidator("zrnvaloper", "zrnvaloperpub")
	config.SetBech32PrefixForConsensusNode("zrnvalcons", "zrnvalconspub")
}

// ---------- Mock StakingKeeper ----------

type mockStakingKeeper struct {
	tiers map[string]uint32
}

func newMockStakingKeeper() *mockStakingKeeper {
	return &mockStakingKeeper{tiers: make(map[string]uint32)}
}

func (m *mockStakingKeeper) GetValidatorTier(_ context.Context, addr string) (uint32, error) {
	return m.tiers[addr], nil
}

// ---------- Mock DisputesKeeper ----------

type mockDisputesKeeper struct {
	disputes map[string]string // evidenceID -> disputeID
}

func newMockDisputesKeeper() *mockDisputesKeeper {
	return &mockDisputesKeeper{disputes: make(map[string]string)}
}

func (m *mockDisputesKeeper) CreateDispute(_ context.Context, _, targetID, _, _ string) (string, error) {
	id := "dispute-" + targetID
	m.disputes[targetID] = id
	return id, nil
}

// ---------- Test Addresses ----------

func testAddr(name string) string {
	h := sha256.Sum256([]byte("test_seed:" + name))
	return sdk.AccAddress(h[:20]).String()
}

// ---------- Test Setup ----------

func setupKeeper(t *testing.T) (keeper.Keeper, sdk.Context, *mockStakingKeeper, *mockDisputesKeeper) {
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

	mockSK := newMockStakingKeeper()
	mockDK := newMockDisputesKeeper()

	storeService := runtime.NewKVStoreService(storeKey)
	k := keeper.NewKeeper(storeService, cdc, "zrn1authority", mockSK, mockDK)

	ctx := sdk.NewContext(stateStore, cmtproto.Header{Height: 100, ChainID: "zerone-test-1"}, false, log.NewNopLogger())

	return k, ctx, mockSK, mockDK
}

func setupMsgServer(t *testing.T) (types.MsgServer, keeper.Keeper, sdk.Context, *mockStakingKeeper, *mockDisputesKeeper) {
	t.Helper()
	k, ctx, sk, dk := setupKeeper(t)
	return keeper.NewMsgServerImpl(k), k, ctx, sk, dk
}

// ---------- Test 1: Submit Evidence ----------

func TestSubmitEvidence(t *testing.T) {
	msgSrv, k, ctx, _, _ := setupMsgServer(t)

	submitter := testAddr("submitter1")
	resp, err := msgSrv.SubmitEvidence(ctx, &types.MsgSubmitEvidence{
		Submitter:    submitter,
		EvidenceType: types.EvidenceType_EVIDENCE_TYPE_DOCUMENT,
		ContentHash:  "abc123hash",
		Metadata:     "test metadata",
	})
	if err != nil {
		t.Fatalf("SubmitEvidence failed: %v", err)
	}
	if resp.EvidenceId == "" {
		t.Fatal("expected non-empty evidence ID")
	}

	// Verify stored
	evidence, found := k.GetEvidence(ctx, resp.EvidenceId)
	if !found {
		t.Fatal("evidence not found after submission")
	}
	if evidence.Submitter != submitter {
		t.Errorf("expected submitter %s, got %s", submitter, evidence.Submitter)
	}
	if evidence.ContentHash != "abc123hash" {
		t.Errorf("expected content_hash abc123hash, got %s", evidence.ContentHash)
	}
	if evidence.Status != types.EvidenceStatus_EVIDENCE_STATUS_SUBMITTED {
		t.Errorf("expected SUBMITTED status, got %s", evidence.Status.String())
	}

	// Verify initial custody entry
	if len(evidence.ChainOfCustody) != 1 {
		t.Fatalf("expected 1 custody entry, got %d", len(evidence.ChainOfCustody))
	}
	if evidence.ChainOfCustody[0].Custodian != submitter {
		t.Errorf("expected custodian %s, got %s", submitter, evidence.ChainOfCustody[0].Custodian)
	}
	if evidence.ChainOfCustody[0].Action != "submit" {
		t.Errorf("expected action 'submit', got %s", evidence.ChainOfCustody[0].Action)
	}

	// Verify submitter index
	bySubmitter := k.GetEvidenceBySubmitter(ctx, submitter)
	if len(bySubmitter) != 1 {
		t.Errorf("expected 1 evidence by submitter, got %d", len(bySubmitter))
	}
}

// ---------- Test 2: Transfer Custody ----------

func TestTransferCustody(t *testing.T) {
	msgSrv, k, ctx, _, _ := setupMsgServer(t)

	submitter := testAddr("submitter1")
	newCustodian := testAddr("custodian2")

	// Submit evidence first
	resp, _ := msgSrv.SubmitEvidence(ctx, &types.MsgSubmitEvidence{
		Submitter:    submitter,
		EvidenceType: types.EvidenceType_EVIDENCE_TYPE_DOCUMENT,
		ContentHash:  "hash1",
		Metadata:     "test",
	})

	// Transfer custody
	_, err := msgSrv.TransferCustody(ctx, &types.MsgTransferCustody{
		EvidenceId:       resp.EvidenceId,
		CurrentCustodian: submitter,
		NewCustodian:     newCustodian,
		Notes:            "Transferring for analysis",
	})
	if err != nil {
		t.Fatalf("TransferCustody failed: %v", err)
	}

	// Verify chain of custody
	evidence, _ := k.GetEvidence(ctx, resp.EvidenceId)
	if len(evidence.ChainOfCustody) != 2 {
		t.Fatalf("expected 2 custody entries, got %d", len(evidence.ChainOfCustody))
	}
	if evidence.ChainOfCustody[1].Custodian != newCustodian {
		t.Errorf("expected new custodian %s, got %s", newCustodian, evidence.ChainOfCustody[1].Custodian)
	}
	if evidence.ChainOfCustody[1].Action != "transfer" {
		t.Errorf("expected action 'transfer', got %s", evidence.ChainOfCustody[1].Action)
	}

	// Non-custodian cannot transfer
	outsider := testAddr("outsider")
	_, err = msgSrv.TransferCustody(ctx, &types.MsgTransferCustody{
		EvidenceId:       resp.EvidenceId,
		CurrentCustodian: outsider,
		NewCustodian:     testAddr("someone"),
		Notes:            "Should fail",
	})
	if err == nil {
		t.Fatal("expected error for non-custodian transfer")
	}
}

// ---------- Test 3: Verify Evidence (tier check + quorum) ----------

func TestVerifyEvidence(t *testing.T) {
	msgSrv, k, ctx, sk, _ := setupMsgServer(t)

	submitter := testAddr("submitter1")
	verifier1 := testAddr("verifier1")
	verifier2 := testAddr("verifier2")
	verifier3 := testAddr("verifier3")

	// Set verifier tiers (need >= 2 per default params)
	sk.tiers[verifier1] = 2
	sk.tiers[verifier2] = 3
	sk.tiers[verifier3] = 2

	// Submit evidence
	resp, _ := msgSrv.SubmitEvidence(ctx, &types.MsgSubmitEvidence{
		Submitter:    submitter,
		EvidenceType: types.EvidenceType_EVIDENCE_TYPE_DOCUMENT,
		ContentHash:  "hash1",
		Metadata:     "test",
	})

	// First verification (quorum not reached yet)
	_, err := msgSrv.VerifyEvidence(ctx, &types.MsgVerifyEvidence{
		EvidenceId: resp.EvidenceId,
		Verifier:   verifier1,
		Outcome:    true,
		Confidence: 900000,
		Method:     "manual_review",
	})
	if err != nil {
		t.Fatalf("VerifyEvidence #1 failed: %v", err)
	}

	// Check status not yet changed (quorum = 3)
	evidence, _ := k.GetEvidence(ctx, resp.EvidenceId)
	if evidence.Status != types.EvidenceStatus_EVIDENCE_STATUS_SUBMITTED {
		t.Errorf("expected SUBMITTED before quorum, got %s", evidence.Status.String())
	}

	// Second verification
	_, err = msgSrv.VerifyEvidence(ctx, &types.MsgVerifyEvidence{
		EvidenceId: resp.EvidenceId,
		Verifier:   verifier2,
		Outcome:    true,
		Confidence: 800000,
		Method:     "manual_review",
	})
	if err != nil {
		t.Fatalf("VerifyEvidence #2 failed: %v", err)
	}

	// Third verification reaches quorum (3 verifications, 3 positive → VERIFIED)
	_, err = msgSrv.VerifyEvidence(ctx, &types.MsgVerifyEvidence{
		EvidenceId: resp.EvidenceId,
		Verifier:   verifier3,
		Outcome:    true,
		Confidence: 700000,
		Method:     "automated_check",
	})
	if err != nil {
		t.Fatalf("VerifyEvidence #3 failed: %v", err)
	}

	evidence, _ = k.GetEvidence(ctx, resp.EvidenceId)
	if evidence.Status != types.EvidenceStatus_EVIDENCE_STATUS_VERIFIED {
		t.Errorf("expected VERIFIED after positive quorum, got %s", evidence.Status.String())
	}
}

// ---------- Test 4: Verify rejects low-tier verifier & self-verification ----------

func TestVerifyEvidenceTierCheck(t *testing.T) {
	msgSrv, _, ctx, sk, _ := setupMsgServer(t)

	submitter := testAddr("submitter1")
	lowTier := testAddr("lowtier")
	sk.tiers[lowTier] = 1 // below min of 2

	// Submit evidence
	resp, _ := msgSrv.SubmitEvidence(ctx, &types.MsgSubmitEvidence{
		Submitter:    submitter,
		EvidenceType: types.EvidenceType_EVIDENCE_TYPE_DOCUMENT,
		ContentHash:  "hash1",
		Metadata:     "test",
	})

	// Low tier should be rejected
	_, err := msgSrv.VerifyEvidence(ctx, &types.MsgVerifyEvidence{
		EvidenceId: resp.EvidenceId,
		Verifier:   lowTier,
		Outcome:    true,
		Confidence: 500000,
		Method:     "review",
	})
	if err == nil {
		t.Fatal("expected error for low-tier verifier")
	}

	// Self-verification should be rejected
	sk.tiers[submitter] = 3 // high tier but same person
	_, err = msgSrv.VerifyEvidence(ctx, &types.MsgVerifyEvidence{
		EvidenceId: resp.EvidenceId,
		Verifier:   submitter,
		Outcome:    true,
		Confidence: 900000,
		Method:     "review",
	})
	if err == nil {
		t.Fatal("expected error for self-verification")
	}
}

// ---------- Test 5: Challenge Evidence → Dispute bridge ----------

func TestChallengeEvidence(t *testing.T) {
	msgSrv, k, ctx, _, dk := setupMsgServer(t)

	submitter := testAddr("submitter1")
	challenger := testAddr("challenger1")

	// Submit evidence
	resp, _ := msgSrv.SubmitEvidence(ctx, &types.MsgSubmitEvidence{
		Submitter:    submitter,
		EvidenceType: types.EvidenceType_EVIDENCE_TYPE_DOCUMENT,
		ContentHash:  "hash1",
		Metadata:     "test",
	})

	// Challenge (within window — created at block 100, window is 50000)
	challengeResp, err := msgSrv.ChallengeEvidence(ctx, &types.MsgChallengeEvidence{
		EvidenceId: resp.EvidenceId,
		Challenger: challenger,
		Reason:     "Evidence is fabricated",
		Bond:       "500000",
	})
	if err != nil {
		t.Fatalf("ChallengeEvidence failed: %v", err)
	}
	if challengeResp.DisputeId == "" {
		t.Fatal("expected non-empty dispute ID")
	}

	// Verify dispute was created via mock
	if _, found := dk.disputes[resp.EvidenceId]; !found {
		t.Error("expected dispute keeper to have created dispute")
	}

	// Verify evidence status updated to challenged
	evidence, _ := k.GetEvidence(ctx, resp.EvidenceId)
	if evidence.Status != types.EvidenceStatus_EVIDENCE_STATUS_CHALLENGED {
		t.Errorf("expected CHALLENGED status, got %s", evidence.Status.String())
	}

	// Verify challenge custody entry added
	lastEntry := evidence.ChainOfCustody[len(evidence.ChainOfCustody)-1]
	if lastEntry.Action != "challenge" {
		t.Errorf("expected action 'challenge', got %s", lastEntry.Action)
	}
}

// ---------- Test 6: Challenge outside window fails ----------

func TestChallengeEvidenceWindowClosed(t *testing.T) {
	msgSrv, _, ctx, _, _ := setupMsgServer(t)

	submitter := testAddr("submitter1")
	challenger := testAddr("challenger1")

	// Submit evidence at block 100
	resp, _ := msgSrv.SubmitEvidence(ctx, &types.MsgSubmitEvidence{
		Submitter:    submitter,
		EvidenceType: types.EvidenceType_EVIDENCE_TYPE_DOCUMENT,
		ContentHash:  "hash1",
		Metadata:     "test",
	})

	// Advance past window: created at 100, window is 50000 → deadline at 50100
	lateCtx := ctx.WithBlockHeight(60000)

	_, err := msgSrv.ChallengeEvidence(lateCtx, &types.MsgChallengeEvidence{
		EvidenceId: resp.EvidenceId,
		Challenger: challenger,
		Reason:     "Too late",
		Bond:       "500000",
	})
	if err == nil {
		t.Fatal("expected error for challenge outside window")
	}
}

// ---------- Test 7: Genesis round-trip ----------

func TestGenesisRoundTrip(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	genState := &types.GenesisState{
		Params: &types.Params{
			MinVerifierTier:       3,
			VerificationQuorum:    5,
			ChallengeBond:         "1000000",
			ChallengeWindowBlocks: 100000,
		},
		Evidences: []*types.Evidence{
			{
				Id:           "evid-gen-1",
				Submitter:    testAddr("submitter1"),
				EvidenceType: types.EvidenceType_EVIDENCE_TYPE_DOCUMENT,
				ContentHash:  "genhash1",
				Status:       types.EvidenceStatus_EVIDENCE_STATUS_SUBMITTED,
				ChainOfCustody: []*types.ChainOfCustodyEntry{
					{Custodian: testAddr("submitter1"), Action: "submit", Timestamp: 50},
				},
				CreatedAtBlock: 50,
				UpdatedAtBlock: 50,
			},
		},
		Verifications: []*types.VerificationResult{
			{
				Id:         "ver-gen-1",
				EvidenceId: "evid-gen-1",
				Verifier:   testAddr("verifier1"),
				Outcome:    true,
				Confidence: 900000,
				Method:     "manual",
			},
		},
		NextEvidenceId:     5,
		NextVerificationId: 3,
	}

	k.InitGenesis(ctx, genState)

	exported := k.ExportGenesis(ctx)

	// Check params
	if exported.Params.MinVerifierTier != 3 {
		t.Errorf("expected min_verifier_tier 3, got %d", exported.Params.MinVerifierTier)
	}
	if exported.Params.VerificationQuorum != 5 {
		t.Errorf("expected verification_quorum 5, got %d", exported.Params.VerificationQuorum)
	}

	// Check evidences
	if len(exported.Evidences) != 1 {
		t.Fatalf("expected 1 evidence, got %d", len(exported.Evidences))
	}
	if exported.Evidences[0].Id != "evid-gen-1" {
		t.Errorf("expected evidence ID evid-gen-1, got %s", exported.Evidences[0].Id)
	}

	// Check verifications
	if len(exported.Verifications) != 1 {
		t.Fatalf("expected 1 verification, got %d", len(exported.Verifications))
	}
	if exported.Verifications[0].Id != "ver-gen-1" {
		t.Errorf("expected verification ID ver-gen-1, got %s", exported.Verifications[0].Id)
	}

	// Check counters
	if exported.NextEvidenceId != 5 {
		t.Errorf("expected next_evidence_id 5, got %d", exported.NextEvidenceId)
	}
	if exported.NextVerificationId != 3 {
		t.Errorf("expected next_verification_id 3, got %d", exported.NextVerificationId)
	}
}

// ---------- Test: Duplicate verification rejected ----------

func TestDuplicateVerificationRejected(t *testing.T) {
	msgSrv, _, ctx, sk, _ := setupMsgServer(t)

	submitter := testAddr("submitter1")
	verifier := testAddr("verifier1")
	sk.tiers[verifier] = 3

	resp, _ := msgSrv.SubmitEvidence(ctx, &types.MsgSubmitEvidence{
		Submitter:    submitter,
		EvidenceType: types.EvidenceType_EVIDENCE_TYPE_DOCUMENT,
		ContentHash:  "hash1",
		Metadata:     "test",
	})

	// First verification
	_, err := msgSrv.VerifyEvidence(ctx, &types.MsgVerifyEvidence{
		EvidenceId: resp.EvidenceId,
		Verifier:   verifier,
		Outcome:    true,
		Confidence: 900000,
		Method:     "review",
	})
	if err != nil {
		t.Fatalf("first VerifyEvidence failed: %v", err)
	}

	// Duplicate should fail
	_, err = msgSrv.VerifyEvidence(ctx, &types.MsgVerifyEvidence{
		EvidenceId: resp.EvidenceId,
		Verifier:   verifier,
		Outcome:    false,
		Confidence: 100000,
		Method:     "review",
	})
	if err == nil {
		t.Fatal("expected error for duplicate verification")
	}
}

// ---------- Test: UpdateParams authority check ----------

func TestUpdateParamsAuthority(t *testing.T) {
	msgSrv, k, ctx, _, _ := setupMsgServer(t)

	// Authorized update
	_, err := msgSrv.UpdateParams(ctx, &types.MsgUpdateParams{
		Authority: "zrn1authority",
		Params: &types.Params{
			MinVerifierTier:       4,
			VerificationQuorum:    7,
			ChallengeBond:         "2000000",
			ChallengeWindowBlocks: 100000,
		},
	})
	if err != nil {
		t.Fatalf("UpdateParams failed: %v", err)
	}

	params := k.GetParams(ctx)
	if params.MinVerifierTier != 4 {
		t.Errorf("expected min_verifier_tier 4, got %d", params.MinVerifierTier)
	}

	// Unauthorized should fail
	_, err = msgSrv.UpdateParams(ctx, &types.MsgUpdateParams{
		Authority: testAddr("not-authority"),
		Params:    types.DefaultParams(),
	})
	if err == nil {
		t.Fatal("expected error for unauthorized UpdateParams")
	}
}

// ---------- Test: Query server ----------

func TestQueryEvidence(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)
	qs := keeper.NewQueryServerImpl(k)

	// Store evidence directly
	k.SetEvidence(ctx, &types.Evidence{
		Id:          "q-evid-1",
		Submitter:   testAddr("submitter1"),
		ContentHash: "qhash1",
		Status:      types.EvidenceStatus_EVIDENCE_STATUS_SUBMITTED,
	})

	resp, err := qs.QueryEvidence(ctx, &types.QueryEvidenceRequest{Id: "q-evid-1"})
	if err != nil {
		t.Fatalf("QueryEvidence failed: %v", err)
	}
	if resp.Evidence.ContentHash != "qhash1" {
		t.Errorf("expected content_hash qhash1, got %s", resp.Evidence.ContentHash)
	}

	// Not found
	_, err = qs.QueryEvidence(ctx, &types.QueryEvidenceRequest{Id: "nonexistent"})
	if err == nil {
		t.Fatal("expected error for nonexistent evidence")
	}
}

func TestQueryParams(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)
	qs := keeper.NewQueryServerImpl(k)

	resp, err := qs.QueryParams(ctx, &types.QueryParamsRequest{})
	if err != nil {
		t.Fatalf("QueryParams failed: %v", err)
	}
	if resp.Params.MinVerifierTier != 2 {
		t.Errorf("expected min_verifier_tier 2, got %d", resp.Params.MinVerifierTier)
	}
}

// ---------- Test: Default genesis validates ----------

func TestDefaultGenesisValidation(t *testing.T) {
	gs := types.DefaultGenesis()
	if err := gs.Validate(); err != nil {
		t.Errorf("default genesis should be valid: %v", err)
	}
}
