package keeper_test

import (
	"context"
	"testing"
	"time"

	"cosmossdk.io/log"
	"cosmossdk.io/store"
	storemetrics "cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"

	emergency "github.com/zerone-chain/zerone/x/emergency"
	"github.com/zerone-chain/zerone/x/emergency/keeper"
	"github.com/zerone-chain/zerone/x/emergency/types"
)

// --- Mock Staking Keeper ---

type mockValidator struct {
	Address    string
	TotalStake string
	Tier       uint32
	IsActive   bool
}

type mockStakingKeeper struct {
	validators []mockValidator
}

func (m *mockStakingKeeper) GetValidator(_ context.Context, addr string) (*types.ValidatorInfo, bool) {
	for _, v := range m.validators {
		if v.Address == addr {
			return &types.ValidatorInfo{
				Address:    v.Address,
				TotalStake: v.TotalStake,
				Tier:       v.Tier,
				IsActive:   v.IsActive,
			}, true
		}
	}
	return nil, false
}

func (m *mockStakingKeeper) GetGuardianValidators(_ context.Context) ([]types.ValidatorInfo, error) {
	var guardians []types.ValidatorInfo
	for _, v := range m.validators {
		if v.Tier == types.TierGuardian && v.IsActive {
			guardians = append(guardians, types.ValidatorInfo{
				Address:    v.Address,
				TotalStake: v.TotalStake,
				Tier:       v.Tier,
				IsActive:   v.IsActive,
			})
		}
	}
	return guardians, nil
}

func (m *mockStakingKeeper) addGuardian(addr, stake string) {
	m.validators = append(m.validators, mockValidator{
		Address:    addr,
		TotalStake: stake,
		Tier:       types.TierGuardian,
		IsActive:   true,
	})
}

func (m *mockStakingKeeper) addNonGuardian(addr, stake string) {
	m.validators = append(m.validators, mockValidator{
		Address:    addr,
		TotalStake: stake,
		Tier:       3, // Scholar
		IsActive:   true,
	})
}

// --- Test Setup ---

func setupKeeper(t *testing.T) (keeper.Keeper, *mockStakingKeeper, sdk.Context) {
	t.Helper()

	storeKey := storetypes.NewKVStoreKey(types.StoreKey)
	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), storemetrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	if err := stateStore.LoadLatestVersion(); err != nil {
		t.Fatalf("failed to load latest version: %v", err)
	}

	ctx := sdk.NewContext(stateStore, cmtproto.Header{Height: 100}, false, log.NewNopLogger()).
		WithBlockTime(time.Now())

	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)

	mock := &mockStakingKeeper{}

	k := keeper.NewKeeper(runtime.NewKVStoreService(storeKey), cdc, "authority", mock)

	// Set default params with lower thresholds for testing.
	params := types.DefaultParams()
	params.MinGuardianStake = "0"        // No min stake for tests
	params.MinDistinctVoters = 1         // Allow single voter for simplicity
	params.CooldownBlocks = 0            // No cooldown for tests
	params.MaxProposalsPerEpoch = 100    // High limit for tests
	params.MaxProposalsPerGuardianPerEpoch = 100
	k.SetParams(ctx, &params)

	return k, mock, ctx
}

// --- Tests ---

func TestGuardianCheck(t *testing.T) {
	k, mock, ctx := setupKeeper(t)

	guardian := "zrn1guardian1"
	nonGuardian := "zrn1scholar1"

	mock.addGuardian(guardian, "100000000000")
	mock.addNonGuardian(nonGuardian, "50000000000")

	if !k.IsGuardian(ctx, guardian) {
		t.Error("expected guardian to be recognized")
	}
	if k.IsGuardian(ctx, nonGuardian) {
		t.Error("expected non-guardian to NOT be recognized")
	}
	if k.IsGuardian(ctx, "zrn1unknown") {
		t.Error("expected unknown address to NOT be recognized")
	}
}

func TestHaltCeremonyFullLifecycle(t *testing.T) {
	k, mock, ctx := setupKeeper(t)

	g1 := "zrn1g1"
	g2 := "zrn1g2"
	mock.addGuardian(g1, "50000000000") // 50k ZRN
	mock.addGuardian(g2, "50000000000") // 50k ZRN

	msgSvr := keeper.NewMsgServerImpl(k)

	// 1. Propose halt
	resp, err := msgSvr.ProposeHalt(ctx, &types.MsgProposeHalt{
		Proposer: g1,
		Reason:   "security breach detected",
	})
	if err != nil {
		t.Fatalf("ProposeHalt failed: %v", err)
	}
	proposalId := resp.ProposalId
	if proposalId == "" {
		t.Fatal("expected non-empty proposal ID")
	}

	// Status should be halt_voting
	status := k.GetEmergencyStatus(ctx)
	if status != types.StatusHaltVoting {
		t.Fatalf("expected halt_voting, got %s", status)
	}

	// 2. Guardian 1 prevotes yes
	voteResp, err := msgSvr.VoteHalt(ctx, &types.MsgVoteHalt{
		Voter:      g1,
		ProposalId: proposalId,
		Approve:    true,
	})
	if err != nil {
		t.Fatalf("VoteHalt (g1 prevote) failed: %v", err)
	}

	// After 50% prevote, not yet quorum (need 75%)
	ceremony, _ := k.GetCeremony(ctx, proposalId)
	if ceremony.Phase != string(types.PhasePrevote) {
		t.Logf("phase after g1 prevote: %s", ceremony.Phase)
	}

	// 3. Guardian 2 prevotes yes → should reach 100% > 75% threshold → advance to precommit
	voteResp, err = msgSvr.VoteHalt(ctx, &types.MsgVoteHalt{
		Voter:      g2,
		ProposalId: proposalId,
		Approve:    true,
	})
	if err != nil {
		t.Fatalf("VoteHalt (g2 prevote) failed: %v", err)
	}

	ceremony, _ = k.GetCeremony(ctx, proposalId)
	if ceremony.Phase != string(types.PhasePrecommit) {
		t.Fatalf("expected precommit phase, got %s", ceremony.Phase)
	}
	if !voteResp.QuorumReached {
		t.Error("expected quorum_reached to be true after prevote quorum")
	}

	// 4. Guardian 1 precommits
	_, err = msgSvr.VoteHalt(ctx, &types.MsgVoteHalt{
		Voter:      g1,
		ProposalId: proposalId,
		Approve:    true,
	})
	if err != nil {
		t.Fatalf("VoteHalt (g1 precommit) failed: %v", err)
	}

	// Check if we need g2 precommit (MinDistinctVoters=1, so g1 alone is enough)
	ceremony, _ = k.GetCeremony(ctx, proposalId)
	if ceremony.Phase == string(types.PhaseFinalized) {
		// Finalized with g1's precommit (50% stake >= 75%? No...)
		// Actually with MinDistinctVoters=1, but 50% < 75% threshold.
		t.Log("ceremony finalized after g1 precommit")
	}

	// 5. Guardian 2 precommits → 100% stake > 75% → finalized
	voteResp, err = msgSvr.VoteHalt(ctx, &types.MsgVoteHalt{
		Voter:      g2,
		ProposalId: proposalId,
		Approve:    true,
	})
	if err != nil {
		t.Fatalf("VoteHalt (g2 precommit) failed: %v", err)
	}

	if !voteResp.ChainHalted {
		t.Error("expected chain_halted to be true")
	}

	// Status should now be halted
	status = k.GetEmergencyStatus(ctx)
	if status != types.StatusHalted {
		t.Fatalf("expected halted, got %s", status)
	}

	// Audit log should have entries
	auditLog := k.GetAuditLog(ctx)
	if len(auditLog) == 0 {
		t.Error("expected audit log entries")
	}
}

func TestResumeLifecycle(t *testing.T) {
	k, mock, ctx := setupKeeper(t)

	g1 := "zrn1g1"
	mock.addGuardian(g1, "100000000000")

	// Set low thresholds and 1 voter minimum
	params := k.GetParams(ctx)
	params.HaltQuorum = 500000   // 50%
	params.ResumeQuorum = 500000 // 50%
	k.SetParams(ctx, params)

	msgSvr := keeper.NewMsgServerImpl(k)

	// First halt the chain
	resp, _ := msgSvr.ProposeHalt(ctx, &types.MsgProposeHalt{
		Proposer: g1,
		Reason:   "test halt",
	})
	msgSvr.VoteHalt(ctx, &types.MsgVoteHalt{Voter: g1, ProposalId: resp.ProposalId, Approve: true})
	msgSvr.VoteHalt(ctx, &types.MsgVoteHalt{Voter: g1, ProposalId: resp.ProposalId, Approve: true})

	if k.GetEmergencyStatus(ctx) != types.StatusHalted {
		t.Fatal("expected halted status for resume test setup")
	}

	// Now propose resume
	resumeResp, err := msgSvr.ProposeResume(ctx, &types.MsgProposeResume{
		Proposer: g1,
	})
	if err != nil {
		t.Fatalf("ProposeResume failed: %v", err)
	}

	status := k.GetEmergencyStatus(ctx)
	if status != types.StatusResumeVoting {
		t.Fatalf("expected resume_voting, got %s", status)
	}

	// Vote yes → prevote quorum → precommit
	msgSvr.VoteResume(ctx, &types.MsgVoteResume{Voter: g1, ProposalId: resumeResp.ProposalId, Approve: true})
	// Vote again → should be precommit phase
	vr, err := msgSvr.VoteResume(ctx, &types.MsgVoteResume{Voter: g1, ProposalId: resumeResp.ProposalId, Approve: true})
	if err != nil {
		t.Fatalf("VoteResume (precommit) failed: %v", err)
	}

	if !vr.ChainResumed {
		t.Error("expected chain_resumed to be true")
	}

	status = k.GetEmergencyStatus(ctx)
	if status != types.StatusNormal {
		t.Fatalf("expected normal after resume, got %s", status)
	}
}

func TestNonGuardianRejection(t *testing.T) {
	k, mock, ctx := setupKeeper(t)

	scholar := "zrn1scholar"
	mock.addNonGuardian(scholar, "50000000000")

	msgSvr := keeper.NewMsgServerImpl(k)

	_, err := msgSvr.ProposeHalt(ctx, &types.MsgProposeHalt{
		Proposer: scholar,
		Reason:   "should fail",
	})
	if err == nil {
		t.Fatal("expected error for non-guardian proposer")
	}
}

func TestCeremonyTimeout(t *testing.T) {
	k, mock, ctx := setupKeeper(t)

	g1 := "zrn1g1"
	mock.addGuardian(g1, "100000000000")

	params := k.GetParams(ctx)
	params.HaltTimeoutBlocks = 10
	params.HaltPrevoteBlocks = 5
	k.SetParams(ctx, params)

	msgSvr := keeper.NewMsgServerImpl(k)

	resp, _ := msgSvr.ProposeHalt(ctx, &types.MsgProposeHalt{
		Proposer: g1,
		Reason:   "timeout test",
	})

	// Advance past timeout deadline
	ctx = ctx.WithBlockHeight(int64(100 + params.HaltTimeoutBlocks + 1))

	// BeginBlock should detect timeout
	am := emergency.NewAppModule(nil, k)
	am.BeginBlock(ctx)

	ceremony, found := k.GetCeremony(ctx, resp.ProposalId)
	if !found {
		t.Fatal("ceremony not found after timeout")
	}
	if ceremony.Phase != string(types.PhaseFailed) {
		t.Fatalf("expected failed phase after timeout, got %s", ceremony.Phase)
	}

	// Status should revert to normal after halt failure
	status := k.GetEmergencyStatus(ctx)
	if status != types.StatusNormal {
		t.Fatalf("expected normal after halt failure, got %s", status)
	}
}

func TestAntiAbusePerGuardianLimit(t *testing.T) {
	k, mock, ctx := setupKeeper(t)

	g1 := "zrn1g1"
	mock.addGuardian(g1, "200000000000")

	params := k.GetParams(ctx)
	params.MaxProposalsPerGuardianPerEpoch = 1
	params.MaxProposalsPerEpoch = 100
	params.HaltQuorum = 500000
	k.SetParams(ctx, params)

	msgSvr := keeper.NewMsgServerImpl(k)

	// First proposal succeeds
	resp1, err := msgSvr.ProposeHalt(ctx, &types.MsgProposeHalt{
		Proposer: g1,
		Reason:   "first proposal",
	})
	if err != nil {
		t.Fatalf("first proposal should succeed: %v", err)
	}

	// Complete the ceremony so another can be proposed
	msgSvr.VoteHalt(ctx, &types.MsgVoteHalt{Voter: g1, ProposalId: resp1.ProposalId, Approve: true})
	msgSvr.VoteHalt(ctx, &types.MsgVoteHalt{Voter: g1, ProposalId: resp1.ProposalId, Approve: true})

	// Second proposal should fail (per-guardian limit)
	_, err = msgSvr.ProposeHalt(ctx, &types.MsgProposeHalt{
		Proposer: g1,
		Reason:   "second proposal",
	})
	if err == nil {
		t.Fatal("second proposal should fail due to per-guardian limit")
	}
}

func TestAntiAbuseEpochLimit(t *testing.T) {
	k, mock, ctx := setupKeeper(t)

	g1 := "zrn1g1"
	g2 := "zrn1g2"
	g3 := "zrn1g3"
	g4 := "zrn1g4"
	mock.addGuardian(g1, "100000000000")
	mock.addGuardian(g2, "100000000000")
	mock.addGuardian(g3, "100000000000")
	mock.addGuardian(g4, "100000000000")

	params := k.GetParams(ctx)
	params.MaxProposalsPerEpoch = 2
	params.MaxProposalsPerGuardianPerEpoch = 5
	params.HaltQuorum = 200000 // 20% so each guardian alone reaches quorum
	k.SetParams(ctx, params)

	msgSvr := keeper.NewMsgServerImpl(k)

	// Proposal 1 by g1 → succeed + complete
	r1, _ := msgSvr.ProposeHalt(ctx, &types.MsgProposeHalt{Proposer: g1, Reason: "p1"})
	msgSvr.VoteHalt(ctx, &types.MsgVoteHalt{Voter: g1, ProposalId: r1.ProposalId, Approve: true})
	msgSvr.VoteHalt(ctx, &types.MsgVoteHalt{Voter: g1, ProposalId: r1.ProposalId, Approve: true})

	// Must resume before next halt proposal
	k.SetEmergencyStatus(ctx, types.StatusNormal)

	// Proposal 2 by g2 → succeed + complete
	r2, _ := msgSvr.ProposeHalt(ctx, &types.MsgProposeHalt{Proposer: g2, Reason: "p2"})
	msgSvr.VoteHalt(ctx, &types.MsgVoteHalt{Voter: g2, ProposalId: r2.ProposalId, Approve: true})
	msgSvr.VoteHalt(ctx, &types.MsgVoteHalt{Voter: g2, ProposalId: r2.ProposalId, Approve: true})

	k.SetEmergencyStatus(ctx, types.StatusNormal)

	// Proposal 3 by g3 → should fail (epoch limit = 2)
	_, err := msgSvr.ProposeHalt(ctx, &types.MsgProposeHalt{Proposer: g3, Reason: "p3"})
	if err == nil {
		t.Fatal("third proposal should fail due to epoch limit")
	}
}

func TestAutoResumeAfterMaxHaltDuration(t *testing.T) {
	k, mock, ctx := setupKeeper(t)

	g1 := "zrn1g1"
	mock.addGuardian(g1, "100000000000")

	params := k.GetParams(ctx)
	params.MaxHaltDurationBlocks = 50
	params.HaltQuorum = 500000
	k.SetParams(ctx, params)

	// Set up halted state
	k.SetEmergencyStatus(ctx, types.StatusHalted)
	k.SetHaltStartBlock(ctx, 100)
	k.SetActiveHaltCeremonyId(ctx, "test-halt")

	// Before expiry — should stay halted
	ctx = ctx.WithBlockHeight(149)
	k.CheckHaltExpiry(ctx)
	if k.GetEmergencyStatus(ctx) != types.StatusHalted {
		t.Fatal("should still be halted before max duration")
	}

	// At expiry — should auto-resume
	ctx = ctx.WithBlockHeight(150)
	k.CheckHaltExpiry(ctx)
	if k.GetEmergencyStatus(ctx) != types.StatusNormal {
		t.Fatal("should be auto-resumed after max duration")
	}
}

func TestRevertCeremony(t *testing.T) {
	k, mock, ctx := setupKeeper(t)

	g1 := "zrn1g1"
	mock.addGuardian(g1, "100000000000")

	params := k.GetParams(ctx)
	params.RevertQuorum = 500000
	params.MaxRevertDepth = 1000
	k.SetParams(ctx, params)

	// Must be halted first
	k.SetEmergencyStatus(ctx, types.StatusHalted)
	k.SetHaltStartBlock(ctx, 50)

	msgSvr := keeper.NewMsgServerImpl(k)

	resp, err := msgSvr.ProposeRevert(ctx, &types.MsgProposeRevert{
		Proposer:       g1,
		RevertToHeight: 90,
		Justification:  "roll back to safe state",
	})
	if err != nil {
		t.Fatalf("ProposeRevert failed: %v", err)
	}

	status := k.GetEmergencyStatus(ctx)
	if status != types.StatusRevertVoting {
		t.Fatalf("expected revert_voting, got %s", status)
	}

	// Vote to finalize
	msgSvr.VoteRevert(ctx, &types.MsgVoteRevert{Voter: g1, ProposalId: resp.ProposalId, Approve: true})
	vr, err := msgSvr.VoteRevert(ctx, &types.MsgVoteRevert{Voter: g1, ProposalId: resp.ProposalId, Approve: true})
	if err != nil {
		t.Fatalf("VoteRevert precommit failed: %v", err)
	}

	if !vr.RevertExecuted {
		t.Error("expected revert_executed to be true")
	}

	status = k.GetEmergencyStatus(ctx)
	if status != types.StatusReverting {
		t.Fatalf("expected reverting, got %s", status)
	}

	target, found := k.GetRevertTarget(ctx)
	if !found {
		t.Fatal("expected revert target to be set")
	}
	if target.Height != 90 {
		t.Fatalf("expected revert height 90, got %d", target.Height)
	}
}

func TestRevertDepthExceeded(t *testing.T) {
	k, mock, ctx := setupKeeper(t)

	g1 := "zrn1g1"
	mock.addGuardian(g1, "100000000000")

	params := k.GetParams(ctx)
	params.MaxRevertDepth = 10
	k.SetParams(ctx, params)

	k.SetEmergencyStatus(ctx, types.StatusHalted)

	msgSvr := keeper.NewMsgServerImpl(k)

	// Try to revert too far back
	_, err := msgSvr.ProposeRevert(ctx, &types.MsgProposeRevert{
		Proposer:       g1,
		RevertToHeight: 50, // current=100, depth=50 > max=10
		Justification:  "too deep",
	})
	if err == nil {
		t.Fatal("expected error for revert depth exceeded")
	}
}

func TestDuplicateVotePrevention(t *testing.T) {
	k, mock, ctx := setupKeeper(t)

	g1 := "zrn1g1"
	g2 := "zrn1g2"
	g3 := "zrn1g3"
	// 3 guardians: each has 33% of stake, so one vote alone (33%) < 90% threshold.
	mock.addGuardian(g1, "100000000000")
	mock.addGuardian(g2, "100000000000")
	mock.addGuardian(g3, "100000000000")

	params := k.GetParams(ctx)
	params.HaltQuorum = 900000 // 90% — one guardian's 33% won't reach quorum
	k.SetParams(ctx, params)

	msgSvr := keeper.NewMsgServerImpl(k)

	resp, _ := msgSvr.ProposeHalt(ctx, &types.MsgProposeHalt{
		Proposer: g1,
		Reason:   "dup test",
	})

	// First prevote succeeds
	_, err := msgSvr.VoteHalt(ctx, &types.MsgVoteHalt{
		Voter: g1, ProposalId: resp.ProposalId, Approve: true,
	})
	if err != nil {
		t.Fatalf("first prevote should succeed: %v", err)
	}

	// Verify still in prevote phase
	ceremony, _ := k.GetCeremony(ctx, resp.ProposalId)
	if ceremony.Phase != string(types.PhasePrevote) {
		t.Fatalf("expected prevote phase, got %s", ceremony.Phase)
	}

	// Second prevote from same guardian should fail (duplicate)
	_, err = msgSvr.VoteHalt(ctx, &types.MsgVoteHalt{
		Voter: g1, ProposalId: resp.ProposalId, Approve: true,
	})
	if err == nil {
		t.Fatal("duplicate prevote should fail")
	}
}

func TestPrecommitWithoutPrevoteRejection(t *testing.T) {
	k, mock, ctx := setupKeeper(t)

	g1 := "zrn1g1"
	g2 := "zrn1g2"
	mock.addGuardian(g1, "80000000000")
	mock.addGuardian(g2, "20000000000")

	params := k.GetParams(ctx)
	params.HaltQuorum = 500000
	k.SetParams(ctx, params)

	msgSvr := keeper.NewMsgServerImpl(k)

	resp, _ := msgSvr.ProposeHalt(ctx, &types.MsgProposeHalt{
		Proposer: g1,
		Reason:   "precommit test",
	})

	// g1 prevotes yes → advances to precommit (80% > 50%)
	msgSvr.VoteHalt(ctx, &types.MsgVoteHalt{Voter: g1, ProposalId: resp.ProposalId, Approve: true})

	// g2 tries to precommit without having prevoted
	_, err := msgSvr.VoteHalt(ctx, &types.MsgVoteHalt{
		Voter: g2, ProposalId: resp.ProposalId, Approve: true,
	})
	if err == nil {
		t.Fatal("precommit without prevote should fail")
	}
}

func TestGenesisRoundtrip(t *testing.T) {
	k, mock, ctx := setupKeeper(t)

	g1 := "zrn1g1"
	mock.addGuardian(g1, "100000000000")

	// Set some state
	k.SetEmergencyStatus(ctx, types.StatusHalted)
	k.SetHaltStartBlock(ctx, 42)
	k.AddAuditEntry(ctx, &types.EmergencyAuditEntry{
		BlockNumber: 42,
		Action:      string(types.AuditHaltExecuted),
		Actor:       "system",
		CeremonyId:  "test-ceremony",
		Details:     "test audit",
	})

	// Export
	genState := k.ExportGenesis(ctx)
	if genState.Status != string(types.StatusHalted) {
		t.Fatalf("expected halted in export, got %s", genState.Status)
	}
	if len(genState.AuditLog) != 1 {
		t.Fatalf("expected 1 audit entry, got %d", len(genState.AuditLog))
	}

	// Re-init on fresh keeper
	k2, _, ctx2 := setupKeeper(t)
	k2.InitGenesis(ctx2, genState)

	status := k2.GetEmergencyStatus(ctx2)
	if status != types.StatusHalted {
		t.Fatalf("expected halted after import, got %s", status)
	}

	auditLog := k2.GetAuditLog(ctx2)
	if len(auditLog) != 1 {
		t.Fatalf("expected 1 audit entry after import, got %d", len(auditLog))
	}
}

func TestIsEmergencyMsg(t *testing.T) {
	tests := []struct {
		name     string
		msg      sdk.Msg
		expected bool
	}{
		{"ProposeHalt", &types.MsgProposeHalt{}, true},
		{"VoteHalt", &types.MsgVoteHalt{}, true},
		{"ProposeRevert", &types.MsgProposeRevert{}, true},
		{"VoteRevert", &types.MsgVoteRevert{}, true},
		{"ProposeResume", &types.MsgProposeResume{}, true},
		{"VoteResume", &types.MsgVoteResume{}, true},
		{"UpdateParams", &types.MsgUpdateParams{}, false}, // UpdateParams is not an emergency msg
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := types.IsEmergencyMsg(tt.msg); got != tt.expected {
				t.Errorf("IsEmergencyMsg(%s) = %v, want %v", tt.name, got, tt.expected)
			}
		})
	}
}

func TestCooldown(t *testing.T) {
	k, mock, ctx := setupKeeper(t)

	g1 := "zrn1g1"
	g2 := "zrn1g2"
	mock.addGuardian(g1, "100000000000")
	mock.addGuardian(g2, "100000000000")

	params := k.GetParams(ctx)
	params.CooldownBlocks = 50
	params.HaltQuorum = 200000
	params.MaxProposalsPerGuardianPerEpoch = 100
	params.MaxProposalsPerEpoch = 100
	k.SetParams(ctx, params)

	msgSvr := keeper.NewMsgServerImpl(k)

	// First proposal at block 100
	r1, _ := msgSvr.ProposeHalt(ctx, &types.MsgProposeHalt{Proposer: g1, Reason: "first"})

	// Complete it
	msgSvr.VoteHalt(ctx, &types.MsgVoteHalt{Voter: g1, ProposalId: r1.ProposalId, Approve: true})
	msgSvr.VoteHalt(ctx, &types.MsgVoteHalt{Voter: g1, ProposalId: r1.ProposalId, Approve: true})
	k.SetEmergencyStatus(ctx, types.StatusNormal)

	// Try another at block 110 (within cooldown)
	ctx = ctx.WithBlockHeight(110)
	_, err := msgSvr.ProposeHalt(ctx, &types.MsgProposeHalt{Proposer: g2, Reason: "too soon"})
	if err == nil {
		t.Fatal("expected cooldown error")
	}

	// Try at block 160 (after cooldown)
	ctx = ctx.WithBlockHeight(160)
	_, err = msgSvr.ProposeHalt(ctx, &types.MsgProposeHalt{Proposer: g2, Reason: "after cooldown"})
	if err != nil {
		t.Fatalf("should succeed after cooldown: %v", err)
	}
}

func TestQueryStatus(t *testing.T) {
	k, _, ctx := setupKeeper(t)

	querySvr := keeper.NewQueryServerImpl(k)

	resp, err := querySvr.Status(ctx, &types.QueryStatusRequest{})
	if err != nil {
		t.Fatalf("query Status failed: %v", err)
	}
	if resp.Status != string(types.StatusNormal) {
		t.Fatalf("expected normal, got %s", resp.Status)
	}
	if resp.IsHalted {
		t.Error("expected is_halted=false")
	}
}

func TestQueryParams(t *testing.T) {
	k, _, ctx := setupKeeper(t)

	querySvr := keeper.NewQueryServerImpl(k)

	resp, err := querySvr.Params(ctx, &types.QueryParamsRequest{})
	if err != nil {
		t.Fatalf("query Params failed: %v", err)
	}
	if resp.Params == nil {
		t.Fatal("expected non-nil params")
	}
}

func TestProposeRevertWhenNotHalted(t *testing.T) {
	k, mock, ctx := setupKeeper(t)

	g1 := "zrn1g1"
	mock.addGuardian(g1, "100000000000")

	params := k.GetParams(ctx)
	params.MaxRevertDepth = 1000
	k.SetParams(ctx, params)

	// Status is normal (not halted) — propose revert should fail
	msgSvr := keeper.NewMsgServerImpl(k)
	_, err := msgSvr.ProposeRevert(ctx, &types.MsgProposeRevert{
		Proposer:       g1,
		RevertToHeight: 90,
		Justification:  "trying to revert while not halted",
	})
	if err == nil {
		t.Fatal("expected error when proposing revert while not halted")
	}
}

func TestGenesisCouncilCanPropose(t *testing.T) {
	k, _, ctx := setupKeeper(t)

	councilMember := "zrn1council1"

	// Set params with genesis council active
	params := k.GetParams(ctx)
	params.GenesisCouncil = []string{councilMember}
	params.CouncilExpiryBlock = 1000 // expires at block 1000
	params.CouncilVirtualStake = "50000000000"
	params.MinGuardianStake = "0"
	params.HaltQuorum = 500000
	k.SetParams(ctx, params)

	// Council member at block 100 (< 1000) should be treated as guardian
	if !k.IsGuardian(ctx, councilMember) {
		t.Fatal("council member should be recognized as guardian during bootstrap")
	}

	// Council member should be able to propose halt
	msgSvr := keeper.NewMsgServerImpl(k)
	resp, err := msgSvr.ProposeHalt(ctx, &types.MsgProposeHalt{
		Proposer: councilMember,
		Reason:   "bootstrap emergency",
	})
	if err != nil {
		t.Fatalf("council member should be able to propose halt: %v", err)
	}
	if resp.ProposalId == "" {
		t.Fatal("expected non-empty proposal ID")
	}
}

func TestCouncilExpiryBlock(t *testing.T) {
	k, _, ctx := setupKeeper(t)

	councilMember := "zrn1council1"

	// Set council that expires at block 200
	params := k.GetParams(ctx)
	params.GenesisCouncil = []string{councilMember}
	params.CouncilExpiryBlock = 200
	params.CouncilVirtualStake = "50000000000"
	k.SetParams(ctx, params)

	// Before expiry — council is active
	if !k.IsGuardian(ctx, councilMember) {
		t.Fatal("council member should be active at block 100")
	}

	// After expiry — council is no longer active
	ctxExpired := ctx.WithBlockHeight(200)
	if k.IsGuardian(ctxExpired, councilMember) {
		t.Fatal("council member should NOT be active at or after expiry block 200")
	}

	// Proposing halt should fail after expiry
	msgSvr := keeper.NewMsgServerImpl(k)
	_, err := msgSvr.ProposeHalt(ctxExpired, &types.MsgProposeHalt{
		Proposer: councilMember,
		Reason:   "should fail after expiry",
	})
	if err == nil {
		t.Fatal("expected error for expired council member proposing halt")
	}
}

func TestMinDistinctVotersEnforcement(t *testing.T) {
	k, mock, ctx := setupKeeper(t)

	// One guardian with 100% of stake
	g1 := "zrn1g1"
	mock.addGuardian(g1, "100000000000")

	params := k.GetParams(ctx)
	params.HaltQuorum = 500000       // 50% — g1 alone meets this
	params.MinDistinctVoters = 3     // But we require 3 distinct voters
	params.HaltTimeoutBlocks = 5     // Short timeout
	params.HaltPrevoteBlocks = 5000  // Long prevote window
	params.HaltPrecommitBlocks = 5   // Short precommit window
	k.SetParams(ctx, params)

	msgSvr := keeper.NewMsgServerImpl(k)

	// Propose + prevote → moves to precommit (g1 has 100% stake > 50%)
	resp, _ := msgSvr.ProposeHalt(ctx, &types.MsgProposeHalt{Proposer: g1, Reason: "min voters test"})
	msgSvr.VoteHalt(ctx, &types.MsgVoteHalt{Voter: g1, ProposalId: resp.ProposalId, Approve: true})

	// g1 precommits — has 100% stake but only 1 distinct voter < 3 required
	msgSvr.VoteHalt(ctx, &types.MsgVoteHalt{Voter: g1, ProposalId: resp.ProposalId, Approve: true})

	// Ceremony should NOT be finalized due to insufficient distinct voters
	ceremony, found := k.GetCeremony(ctx, resp.ProposalId)
	if !found {
		t.Fatal("ceremony not found")
	}
	if ceremony.Phase == string(types.PhaseFinalized) {
		t.Fatal("ceremony should NOT be finalized with insufficient distinct voters")
	}

	// Advance past precommit deadline — should fail due to distinct voter check
	ctx = ctx.WithBlockHeight(int64(100 + params.HaltTimeoutBlocks + 1))
	am := emergency.NewAppModule(nil, k)
	am.BeginBlock(ctx)

	ceremony, _ = k.GetCeremony(ctx, resp.ProposalId)
	if ceremony.Phase != string(types.PhaseFailed) {
		t.Fatalf("expected failed phase, got %s", ceremony.Phase)
	}
	if ceremony.FailureReason == "" {
		t.Fatal("expected failure reason to be set")
	}
}

func TestEpochCounterReset(t *testing.T) {
	k, _, ctx := setupKeeper(t)

	// Simulate some proposal counts
	k.IncrementGuardianProposalCount(ctx, "zrn1g1")
	k.IncrementGuardianProposalCount(ctx, "zrn1g2")
	k.IncrementEpochProposalCount(ctx)

	if k.GetGuardianProposalCount(ctx, "zrn1g1") != 1 {
		t.Fatal("expected count 1")
	}
	if k.GetEpochProposalCount(ctx) != 1 {
		t.Fatal("expected epoch count 1")
	}

	// Reset
	k.ResetEpochCounters(ctx)

	if k.GetGuardianProposalCount(ctx, "zrn1g1") != 0 {
		t.Fatal("expected count 0 after reset")
	}
	if k.GetEpochProposalCount(ctx) != 0 {
		t.Fatal("expected epoch count 0 after reset")
	}
}
