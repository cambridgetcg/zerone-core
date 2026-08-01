package keeper_test

import (
	"bytes"
	"context"
	"fmt"
	"strings"
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
	"google.golang.org/protobuf/proto"

	emergency "github.com/zerone-chain/zerone/x/emergency"
	"github.com/zerone-chain/zerone/x/emergency/keeper"
	"github.com/zerone-chain/zerone/x/emergency/types"
)

const (
	testRecoveryManifestSHA256 = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	legacyQuarantineIDForTest  = "legacy-genesis-quarantine"
)

func testCouncilAddr(seed byte) string {
	return sdk.AccAddress(bytes.Repeat([]byte{seed}, 20)).String()
}

func testResumeMsg(proposer string) *types.MsgProposeResume {
	return &types.MsgProposeResume{
		Proposer:               proposer,
		Justification:          "recovery manifest independently verified",
		RecoveryManifestSha256: testRecoveryManifestSHA256,
	}
}

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

type mockRecoveryTargetVerifier struct {
	err   error
	calls int
}

func (m *mockRecoveryTargetVerifier) VerifyRecoveryAuthorizationTarget(
	_ context.Context,
	_ uint64,
	_ string,
	_ string,
	_ string,
) error {
	m.calls++
	return m.err
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
	ctx = types.WithAuthenticatedEmergencyTx(ctx)

	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)

	mock := &mockStakingKeeper{}

	k := keeper.NewKeeper(runtime.NewKVStoreService(storeKey), cdc, "authority", mock)

	// Set valid parameters with lower thresholds for testing.
	params := types.DefaultParams()
	params.MinGuardianStake = "1"
	params.MinDistinctVoters = 1 // Allow single voter for simplicity
	params.CooldownBlocks = 0
	params.MaxProposalsPerEpoch = 100 // High limit for tests
	params.MaxProposalsPerGuardianPerEpoch = 100
	k.SetParams(ctx, &params)

	return k, mock, ctx
}

func setupRecoveryAuthorizationKeeper(
	t *testing.T,
) (
	keeper.Keeper,
	*mockStakingKeeper,
	*mockRecoveryTargetVerifier,
	sdk.Context,
	string,
) {
	t.Helper()
	k, staking, ctx := setupKeeper(t)
	snapshot, err := k.ReadOperationsSafetySnapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := k.PrepareOperationsSafetyV2FromSnapshot(
		ctx,
		snapshot,
	); err != nil {
		t.Fatal(err)
	}
	if err := keeper.NewMigrator(k).Migrate1to2(ctx); err != nil {
		t.Fatal(err)
	}
	guardian := testCouncilAddr(0x73)
	staking.addGuardian(guardian, "100000000000")
	halt, err := k.CreateHaltCeremony(
		ctx,
		&types.EmergencyHaltProposal{
			Id:       "recovery-test-halt",
			Proposer: guardian,
			Reason:   "recovery authorization test incident",
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	halt.Phase = string(types.PhaseFinalized)
	if err := k.SetCeremony(ctx, halt); err != nil {
		t.Fatal(err)
	}
	k.SetEmergencyStatus(ctx, types.StatusHalted)
	k.SetActiveHaltCeremonyId(ctx, halt.Id)
	k.SetHaltStartBlock(ctx, uint64(ctx.BlockHeight()-1))
	verifier := &mockRecoveryTargetVerifier{}
	k.SetRecoveryAuthorizationTargetVerifier(verifier)
	return k, staking, verifier, ctx, guardian
}

func recoveryAuthorizationMsg(
	proposer string,
	actionType string,
) *types.MsgProposeRecoveryAuthorization {
	return &types.MsgProposeRecoveryAuthorization{
		Proposer:               proposer,
		SdkGovProposalId:       7,
		ActionSha256:           strings.Repeat("b", 64),
		RecoveryManifestSha256: strings.Repeat("c", 64),
		Justification:          "reviewed forward recovery artifact",
		UpgradePlanSha256:      strings.Repeat("d", 64),
		AuthorizedSubmitter:    testCouncilAddr(0x74),
		ActionType:             actionType,
	}
}

func finalizeRecoveryAuthorization(
	t *testing.T,
	server types.MsgServer,
	ctx sdk.Context,
	guardian string,
	proposalID string,
) *types.MsgVoteRecoveryAuthorizationResponse {
	t.Helper()
	if _, err := server.VoteRecoveryAuthorization(
		ctx,
		&types.MsgVoteRecoveryAuthorization{
			Voter:      guardian,
			ProposalId: proposalID,
			Approve:    true,
		},
	); err != nil {
		t.Fatal(err)
	}
	response, err := server.VoteRecoveryAuthorization(
		ctx,
		&types.MsgVoteRecoveryAuthorization{
			Voter:      guardian,
			ProposalId: proposalID,
			Approve:    true,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	return response
}

// --- Tests ---

func TestRecoveryAuthorizationLifecycleAndQuorumRevocation(t *testing.T) {
	k, _, verifier, ctx, guardian := setupRecoveryAuthorizationKeeper(t)
	server := keeper.NewMsgServerImpl(k)

	authorize := recoveryAuthorizationMsg(guardian, "software_upgrade")
	proposed, err := server.ProposeRecoveryAuthorization(ctx, authorize)
	if err != nil {
		t.Fatal(err)
	}
	finalized := finalizeRecoveryAuthorization(
		t,
		server,
		ctx,
		guardian,
		proposed.ProposalId,
	)
	if !finalized.QuorumReached || !finalized.RecoveryAuthorized {
		t.Fatalf("expected exact recovery authorization, got %+v", finalized)
	}
	authorization, found, err := k.GetRecoveryAuthorization(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !found {
		t.Fatal("finalized Guardian quorum did not persist authorization")
	}
	if authorization.AuthorizationCeremonyId != proposed.ProposalId ||
		authorization.Generation != 1 ||
		authorization.Outcome != "" {
		t.Fatalf("unexpected live authorization: %+v", authorization)
	}
	if verifier.calls != 2 {
		t.Fatalf(
			"expected target verification at propose and final precommit, got %d",
			verifier.calls,
		)
	}

	replacement := recoveryAuthorizationMsg(guardian, "software_upgrade")
	replacement.ActionSha256 = strings.Repeat("e", 64)
	if _, err := server.ProposeRecoveryAuthorization(
		ctx,
		replacement,
	); err == nil || !strings.Contains(err.Error(), "must reach a terminal outcome") {
		t.Fatalf("expected live authorization replacement rejection, got %v", err)
	}

	revoke := recoveryAuthorizationMsg(guardian, "revoke")
	revokeResponse, err := server.ProposeRecoveryAuthorization(ctx, revoke)
	if err != nil {
		t.Fatal(err)
	}
	revoked := finalizeRecoveryAuthorization(
		t,
		server,
		ctx,
		guardian,
		revokeResponse.ProposalId,
	)
	if !revoked.QuorumReached || revoked.RecoveryAuthorized {
		t.Fatalf("revocation must close, not grant, recovery authority: %+v", revoked)
	}
	authorization, found, err = k.GetRecoveryAuthorization(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !found ||
		authorization.Outcome != "revoked" ||
		authorization.TerminalAtBlock != uint64(ctx.BlockHeight()) {
		t.Fatalf("unexpected revoked authorization: %+v found=%t", authorization, found)
	}

	next := recoveryAuthorizationMsg(guardian, "software_upgrade")
	next.ActionSha256 = strings.Repeat("e", 64)
	nextResponse, err := server.ProposeRecoveryAuthorization(ctx, next)
	if err != nil {
		t.Fatal(err)
	}
	nextCeremony, found := k.GetCeremony(ctx, nextResponse.ProposalId)
	if !found {
		t.Fatal("replacement authorization ceremony not found")
	}
	var nextProposal types.EmergencyRecoveryAuthorizationProposal
	if err := proto.Unmarshal(nextCeremony.ProposalData, &nextProposal); err != nil {
		t.Fatal(err)
	}
	if nextProposal.Generation != 2 {
		t.Fatalf(
			"expected replacement generation 2 after revocation, got %d",
			nextProposal.Generation,
		)
	}
}

func TestRecoveryAuthorizationFinalPrecommitRechecksTarget(t *testing.T) {
	k, _, verifier, ctx, guardian := setupRecoveryAuthorizationKeeper(t)
	server := keeper.NewMsgServerImpl(k)
	msg := recoveryAuthorizationMsg(guardian, "software_upgrade")
	proposed, err := server.ProposeRecoveryAuthorization(ctx, msg)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := server.VoteRecoveryAuthorization(
		ctx,
		&types.MsgVoteRecoveryAuthorization{
			Voter:      guardian,
			ProposalId: proposed.ProposalId,
			Approve:    true,
		},
	); err != nil {
		t.Fatal(err)
	}
	verifier.err = fmt.Errorf("target sequence changed")
	response, err := server.VoteRecoveryAuthorization(
		ctx,
		&types.MsgVoteRecoveryAuthorization{
			Voter:      guardian,
			ProposalId: proposed.ProposalId,
			Approve:    true,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if response.RecoveryAuthorized {
		t.Fatal("changed target must never produce an authorized response")
	}
	ceremony, found := k.GetCeremony(ctx, proposed.ProposalId)
	if !found ||
		ceremony.Phase != string(types.PhaseFailed) ||
		!strings.Contains(ceremony.FailureReason, "target changed") {
		t.Fatalf("unexpected terminal ceremony: %+v found=%t", ceremony, found)
	}
	if _, found, err := k.GetRecoveryAuthorization(ctx); err != nil {
		t.Fatal(err)
	} else if found {
		t.Fatal("failed final precommit must not persist recovery authority")
	}
	if verifier.calls != 2 {
		t.Fatalf(
			"expected target verification at propose and final precommit, got %d",
			verifier.calls,
		)
	}
}

func TestRecoveryAuthorizationRateLimitStopsCeremonyMonopoly(
	t *testing.T,
) {
	k, staking, verifier, ctx, guardian :=
		setupRecoveryAuthorizationKeeper(t)
	challenger := testCouncilAddr(0x75)
	staking.addGuardian(challenger, "100000000000")
	params := k.GetParams(ctx)
	params.MaxProposalsPerGuardianPerEpoch = 1
	params.MaxProposalsPerEpoch = 3
	params.CooldownBlocks = 5
	k.SetParams(ctx, params)
	server := keeper.NewMsgServerImpl(k)

	first, err := server.ProposeRecoveryAuthorization(
		ctx,
		recoveryAuthorizationMsg(guardian, "software_upgrade"),
	)
	if err != nil {
		t.Fatalf("first recovery authorization failed: %v", err)
	}
	if _, err := server.VoteRecoveryAuthorization(
		ctx,
		&types.MsgVoteRecoveryAuthorization{
			Voter:      challenger,
			ProposalId: first.ProposalId,
			Approve:    false,
		},
	); err != nil {
		t.Fatalf("rejecting hostile recovery authorization failed: %v", err)
	}
	if active, found := k.GetActiveCeremony(ctx); found {
		t.Fatalf("failed recovery authorization remained active: %+v", active)
	}

	retryCtx := ctx.WithBlockHeight(ctx.BlockHeight() + 1)
	if _, err := server.ProposeResume(
		retryCtx,
		testResumeMsg(guardian),
	); err == nil || !types.ErrProposalLimitExceeded.Is(err) {
		t.Fatalf(
			"guardian bypassed its recovery limit through the resume lane: %v",
			err,
		)
	}
	changed := recoveryAuthorizationMsg(guardian, "software_upgrade")
	changed.ActionSha256 = strings.Repeat("e", 64)
	if _, err := server.ProposeRecoveryAuthorization(
		retryCtx,
		changed,
	); err == nil || !types.ErrProposalLimitExceeded.Is(err) {
		t.Fatalf(
			"same guardian reopened the sole recovery ceremony: %v",
			err,
		)
	}
	if verifier.calls != 1 {
		t.Fatalf(
			"rate-limited retry reached target verifier: calls=%d",
			verifier.calls,
		)
	}

	honest := recoveryAuthorizationMsg(challenger, "software_upgrade")
	honest.ActionSha256 = strings.Repeat("f", 64)
	if _, err := server.ProposeRecoveryAuthorization(
		retryCtx,
		honest,
	); err == nil || !types.ErrCooldownActive.Is(err) {
		t.Fatalf("shared recovery cooldown was bypassed: %v", err)
	}
	if verifier.calls != 1 {
		t.Fatalf(
			"cooldown-limited retry reached target verifier: calls=%d",
			verifier.calls,
		)
	}

	boundaryCtx := ctx.WithBlockHeight(ctx.BlockHeight() + 5)
	next, err := server.ProposeRecoveryAuthorization(
		boundaryCtx,
		honest,
	)
	if err != nil {
		t.Fatalf(
			"another guardian could not recover at cooldown boundary: %v",
			err,
		)
	}
	if next.ProposalId == first.ProposalId {
		t.Fatal("honest recovery reused the failed ceremony id")
	}
	if verifier.calls != 2 {
		t.Fatalf(
			"accepted recovery target verification calls=%d, want 2",
			verifier.calls,
		)
	}
	if got := k.GetGuardianProposalCount(boundaryCtx, guardian); got != 1 {
		t.Fatalf("attacker proposal count=%d, want 1", got)
	}
	if got := k.GetGuardianProposalCount(boundaryCtx, challenger); got != 1 {
		t.Fatalf("honest guardian proposal count=%d, want 1", got)
	}
	if got := k.GetEpochProposalCount(boundaryCtx); got != 2 {
		t.Fatalf("shared epoch proposal count=%d, want 2", got)
	}
}

func TestGuardianCheck(t *testing.T) {
	k, mock, ctx := setupKeeper(t)

	guardian := testCouncilAddr(10)
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

func TestEmergencyMessagesRejectUnmarkedExecutionContext(t *testing.T) {
	k, mock, markedCtx := setupKeeper(t)
	ctx := markedCtx.WithContext(context.Background())
	guardian := testCouncilAddr(9)
	mock.addGuardian(guardian, "100000000000")
	msgServer := keeper.NewMsgServerImpl(k)

	tests := []struct {
		name string
		call func() error
	}{
		{
			name: "propose halt",
			call: func() error {
				_, err := msgServer.ProposeHalt(ctx, &types.MsgProposeHalt{
					Proposer: guardian,
					Reason:   "must not execute through governance or a module router",
				})
				return err
			},
		},
		{
			name: "vote halt",
			call: func() error {
				_, err := msgServer.VoteHalt(ctx, &types.MsgVoteHalt{
					Voter:      guardian,
					ProposalId: "missing",
					Approve:    true,
				})
				return err
			},
		},
		{
			name: "propose resume",
			call: func() error {
				_, err := msgServer.ProposeResume(ctx, testResumeMsg(guardian))
				return err
			},
		},
		{
			name: "vote resume",
			call: func() error {
				_, err := msgServer.VoteResume(ctx, &types.MsgVoteResume{
					Voter:      guardian,
					ProposalId: "missing",
					Approve:    true,
				})
				return err
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.call()
			if err == nil || !types.ErrUnauthenticatedEmergencyExecution.Is(err) {
				t.Fatalf("unmarked emergency execution returned %v", err)
			}
		})
	}

	// Governance-authority parameter updates remain executable without the
	// direct-transaction marker.
	params := k.GetParams(ctx)
	if _, err := msgServer.UpdateParams(ctx, &types.MsgUpdateParams{
		Authority: k.GetAuthority(),
		Params:    params,
	}); err != nil {
		t.Fatalf("governance authority update unexpectedly required marker: %v", err)
	}
}

func TestCeremonyDeadlineOverflowFailsClosed(t *testing.T) {
	k, _, ctx := setupKeeper(t)
	params := k.GetParams(ctx)
	params.HaltTimeoutBlocks = ^uint64(0)
	k.SetParams(ctx, params)

	_, err := k.CreateHaltCeremony(ctx, &types.EmergencyHaltProposal{
		Id:       "overflow-halt",
		Proposer: "guardian",
		Reason:   "deadline overflow test",
	})
	if err == nil {
		t.Fatal("expected overflowing ceremony timeout deadline to fail closed")
	}
}

func TestHaltCeremonyFullLifecycle(t *testing.T) {
	k, mock, ctx := setupKeeper(t)

	g1 := testCouncilAddr(1)
	g2 := testCouncilAddr(2)
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

	g1 := testCouncilAddr(1)
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

	// The halt-finalization block must commit before recovery voting opens.
	resumeCtx := ctx.WithBlockHeight(ctx.BlockHeight() + 1)
	resumeResp, err := msgSvr.ProposeResume(resumeCtx, testResumeMsg(g1))
	if err != nil {
		t.Fatalf("ProposeResume failed: %v", err)
	}

	status := k.GetEmergencyStatus(ctx)
	if status != types.StatusResumeVoting {
		t.Fatalf("expected resume_voting, got %s", status)
	}

	// Vote yes → prevote quorum → precommit
	msgSvr.VoteResume(resumeCtx, &types.MsgVoteResume{Voter: g1, ProposalId: resumeResp.ProposalId, Approve: true})
	// Vote again → should be precommit phase
	vr, err := msgSvr.VoteResume(resumeCtx, &types.MsgVoteResume{Voter: g1, ProposalId: resumeResp.ProposalId, Approve: true})
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
	releaseBlock := uint64(resumeCtx.BlockHeight())
	if got := k.GetQuarantineReleaseBlock(resumeCtx); got != releaseBlock {
		t.Fatalf("quarantine release block = %d, want %d", got, releaseBlock)
	}
	if !k.IsHalted(resumeCtx) {
		t.Fatal("resume finalization must keep DeliverTx quarantined through the rest of its block")
	}
	if k.IsHalted(resumeCtx.WithIsCheckTx(true)) {
		t.Fatal("post-commit CheckTx targeting H+1 must reopen admission")
	}

	queryResp, err := keeper.NewQueryServerImpl(k).Status(
		resumeCtx,
		&types.QueryStatusRequest{},
	)
	if err != nil {
		t.Fatalf("query post-resume latch: %v", err)
	}
	if !queryResp.IsHalted ||
		queryResp.QuarantineReleaseBlock != releaseBlock ||
		queryResp.AdmissionReopensAtBlock != releaseBlock+1 {
		t.Fatalf("post-resume query omitted release-latch semantics: %+v", queryResp)
	}

	foundReopenAttribute := false
	for _, event := range resumeCtx.EventManager().Events() {
		if event.Type != "zerone.emergency.ceremony_finalized" {
			continue
		}
		for _, attribute := range event.Attributes {
			if string(attribute.Key) == "admission_reopens_at_block" &&
				string(attribute.Value) == fmt.Sprintf("%d", releaseBlock+1) {
				foundReopenAttribute = true
			}
		}
	}
	if !foundReopenAttribute {
		t.Fatal("resume finalization event omitted admission_reopens_at_block")
	}

	// A guardian cannot use the normal status written by resume finalization
	// to open another halt vote and bypass the release latch in the same block.
	if _, err := msgSvr.ProposeHalt(
		resumeCtx,
		&types.MsgProposeHalt{
			Proposer: g1,
			Reason:   "same-block latch bypass attempt",
		},
	); err == nil || !strings.Contains(err.Error(), "release latch remains active") {
		t.Fatalf("same-block re-halt bypass was accepted: %v", err)
	}
	if k.GetEmergencyStatus(resumeCtx) != types.StatusNormal {
		t.Fatal("failed same-block re-halt attempt changed emergency status")
	}

	nextCtx := resumeCtx.WithBlockHeight(resumeCtx.BlockHeight() + 1)
	if err := emergency.NewAppModule(nil, k).BeginBlock(nextCtx); err != nil {
		t.Fatalf("clear expired release latch: %v", err)
	}
	if k.GetQuarantineReleaseBlock(nextCtx) != 0 || k.IsHalted(nextCtx) {
		t.Fatal("H+1 BeginBlock did not reopen and clear the release latch")
	}
}

func TestResumeRequiresActiveIncidentLinkage(t *testing.T) {
	k, mock, ctx := setupKeeper(t)
	guardian := testCouncilAddr(10)
	mock.addGuardian(guardian, "100000000000")
	k.SetEmergencyStatus(ctx, types.StatusHalted)

	_, err := keeper.NewMsgServerImpl(k).ProposeResume(ctx, testResumeMsg(guardian))
	if err == nil || !types.ErrHaltRequired.Is(err) {
		t.Fatalf("expected missing halt ceremony identity to fail closed, got: %v", err)
	}
	if k.GetEmergencyStatus(ctx) != types.StatusHalted {
		t.Fatal("failed resume proposal must leave quarantine active")
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

	g1 := testCouncilAddr(1)
	mock.addGuardian(g1, "100000000000")

	params := k.GetParams(ctx)
	params.HaltTimeoutBlocks = 10
	params.HaltPrevoteBlocks = 5
	params.HaltPrecommitBlocks = 5
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

	g1 := testCouncilAddr(1)
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

	g1 := testCouncilAddr(1)
	g2 := testCouncilAddr(2)
	g3 := testCouncilAddr(3)
	g4 := testCouncilAddr(4)
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

func TestQuarantineDoesNotAutoResumeAfterDeadline(t *testing.T) {
	k, mock, ctx := setupKeeper(t)

	g1 := testCouncilAddr(1)
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

	// At expiry — elapsed time is an escalation signal, not recovery proof.
	ctx = ctx.WithBlockHeight(150)
	k.CheckHaltExpiry(ctx)
	if k.GetEmergencyStatus(ctx) != types.StatusHalted {
		t.Fatal("transaction quarantine must remain fail-closed after its deadline")
	}
	if k.GetActiveHaltCeremonyId(ctx) != "test-halt" {
		t.Fatal("deadline must not erase the active incident identity")
	}
}

func TestHeightOnlyRevertMessagesAreDisabled(t *testing.T) {
	k, mock, ctx := setupKeeper(t)

	g1 := testCouncilAddr(1)
	mock.addGuardian(g1, "100000000000")

	params := k.GetParams(ctx)
	params.RevertQuorum = 500000
	params.MaxRevertDepth = 1000
	k.SetParams(ctx, params)

	// Must be halted first
	k.SetEmergencyStatus(ctx, types.StatusHalted)
	k.SetHaltStartBlock(ctx, 50)

	msgSvr := keeper.NewMsgServerImpl(k)

	_, err := msgSvr.ProposeRevert(ctx, &types.MsgProposeRevert{
		Proposer:       g1,
		RevertToHeight: 90,
		Justification:  "roll back to safe state",
	})
	if err == nil || !types.ErrUnsafeRevertDisabled.Is(err) {
		t.Fatalf("expected unsafe legacy revert to fail closed, got: %v", err)
	}
	if k.GetEmergencyStatus(ctx) != types.StatusHalted {
		t.Fatal("refused revert must leave transaction quarantine active")
	}
	if _, found := k.GetRevertTarget(ctx); found {
		t.Fatal("refused revert must not persist an unauthenticated target")
	}
}

func TestLiveLegacyRevertingStateBecomesRecoverableQuarantine(t *testing.T) {
	k, mock, ctx := setupKeeper(t)
	guardian := testCouncilAddr(20)
	mock.addGuardian(guardian, "100000000000")

	k.SetEmergencyStatus(ctx, types.StatusReverting)
	k.SetRevertTarget(ctx, 90, "legacy-target-hash", "legacy-revert-ceremony")
	k.MonitorRevertStatus(ctx)

	if got := k.GetEmergencyStatus(ctx); got != types.StatusHalted {
		t.Fatalf("legacy reverting state must normalize to halted, got %s", got)
	}
	if got := k.GetActiveHaltCeremonyId(ctx); got != "legacy-genesis-quarantine" {
		t.Fatalf("legacy recovery linkage: got %q", got)
	}
	if _, found := k.GetRevertTarget(ctx); found {
		t.Fatal("unsafe legacy revert target must be cleared")
	}
	resumeCtx := ctx.WithBlockHeight(ctx.BlockHeight() + 1)
	if _, err := keeper.NewMsgServerImpl(k).ProposeResume(resumeCtx, testResumeMsg(guardian)); err != nil {
		t.Fatalf("normalized in-place state must admit evidence-bound resume: %v", err)
	}
}

func TestActiveLegacyRevertVoteIsRetiredWithoutWaitingForDeadline(t *testing.T) {
	k, _, ctx := setupKeeper(t)
	legacy := &types.EmergencyCeremony{
		Id:                "legacy-active-revert",
		Type:              string(types.CeremonyRevert),
		Phase:             string(types.PhasePrevote),
		PrevoteDeadline:   ^uint64(0),
		PrecommitDeadline: ^uint64(0),
		TimeoutDeadline:   ^uint64(0),
		YesPrevoteStake:   "0",
		NoPrevoteStake:    "0",
		PrecommitStake:    "0",
	}
	if err := k.SetCeremony(ctx, legacy); err != nil {
		t.Fatal(err)
	}
	k.SetEmergencyStatus(ctx, types.StatusRevertVoting)

	k.MonitorRevertStatus(ctx)

	if got := k.GetEmergencyStatus(ctx); got != types.StatusHalted {
		t.Fatalf("legacy revert voting must normalize immediately, got %s", got)
	}
	updated, found := k.GetCeremony(ctx, legacy.Id)
	if !found || updated.Phase != string(types.PhaseFailed) {
		t.Fatalf("legacy active revert ceremony was not retired: %+v", updated)
	}
	if got := k.GetActiveHaltCeremonyId(ctx); got != "legacy-genesis-quarantine" {
		t.Fatalf("legacy ceremony recovery linkage: got %q", got)
	}

	exported := k.ExportGenesis(ctx)
	if err := exported.Validate(); err != nil {
		t.Fatalf("terminal legacy ceremony must remain exportable: %v", err)
	}
	k2, _, ctx2 := setupKeeper(t)
	k2.InitGenesis(ctx2, exported)
	if got := k2.GetEmergencyStatus(ctx2); got != types.StatusHalted {
		t.Fatalf("legacy export/import reopened quarantine: %s", got)
	}
}

func TestRevertDepthExceeded(t *testing.T) {
	k, mock, ctx := setupKeeper(t)

	g1 := testCouncilAddr(1)
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

	g1 := testCouncilAddr(1)
	g2 := testCouncilAddr(2)
	g3 := testCouncilAddr(3)
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

	g1 := testCouncilAddr(1)
	g2 := testCouncilAddr(2)
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

	g1 := testCouncilAddr(1)
	mock.addGuardian(g1, "100000000000")

	// Set a coherent finalized quarantine record.
	ceremony, err := k.CreateHaltCeremony(ctx, &types.EmergencyHaltProposal{
		Id:              "test-ceremony",
		Proposer:        g1,
		Reason:          "genesis roundtrip",
		ProposedAtBlock: uint64(ctx.BlockHeight()),
	})
	if err != nil {
		t.Fatalf("create halt ceremony: %v", err)
	}
	ceremony.Phase = string(types.PhaseFinalized)
	if err := k.SetCeremony(ctx, ceremony); err != nil {
		t.Fatalf("persist finalized halt ceremony: %v", err)
	}
	k.SetEmergencyStatus(ctx, types.StatusHalted)
	k.SetHaltStartBlock(ctx, 42)
	k.SetActiveHaltCeremonyId(ctx, "test-ceremony")
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
	if genState.ActiveHaltCeremonyId != "test-ceremony" || genState.HaltStartBlock != 42 {
		t.Fatalf("export lost quarantine linkage: %+v", genState)
	}

	// Re-init on fresh keeper
	k2, _, ctx2 := setupKeeper(t)
	k2.InitGenesis(ctx2, genState)

	status := k2.GetEmergencyStatus(ctx2)
	if status != types.StatusHalted {
		t.Fatalf("expected halted after import, got %s", status)
	}
	if got := k2.GetActiveHaltCeremonyId(ctx2); got != "test-ceremony" {
		t.Fatalf("active halt ceremony after import: got %q", got)
	}
	if got := k2.GetHaltStartBlock(ctx2); got != 42 {
		t.Fatalf("halt start after import: got %d", got)
	}

	auditLog := k2.GetAuditLog(ctx2)
	if len(auditLog) != 1 {
		t.Fatalf("expected 1 audit entry after import, got %d", len(auditLog))
	}
}

func TestLegacyRevertingGenesisNormalizesToRecoverableQuarantine(t *testing.T) {
	k, _, ctx := setupKeeper(t)
	genState := types.DefaultGenesis()
	genState.Params = k.GetParams(ctx)
	genState.Status = string(types.StatusReverting)

	k2, mock2, ctx2 := setupKeeper(t)
	k2.InitGenesis(ctx2, genState)
	if got := k2.GetEmergencyStatus(ctx2); got != types.StatusHalted {
		t.Fatalf("legacy reverting status must normalize to halted, got %s", got)
	}
	if got := k2.GetActiveHaltCeremonyId(ctx2); got == "" {
		t.Fatal("legacy quarantine must receive deterministic recovery linkage")
	}
	if got := k2.GetHaltStartBlock(ctx2); got == 0 {
		t.Fatal("legacy quarantine must receive an escalation start block")
	}

	guardian := testCouncilAddr(20)
	mock2.addGuardian(guardian, "100000000000")
	resumeCtx := ctx2.WithBlockHeight(ctx2.BlockHeight() + 1)
	if _, err := keeper.NewMsgServerImpl(k2).ProposeResume(resumeCtx, testResumeMsg(guardian)); err != nil {
		t.Fatalf("normalized legacy quarantine must admit evidence-bound resume: %v", err)
	}
}

func TestLegacyRevertVotingGenesisTerminalizesActiveCeremony(t *testing.T) {
	k, _, ctx := setupKeeper(t)
	proposalData, err := proto.Marshal(&types.EmergencyRevertProposal{
		Id:       "legacy-genesis-revert",
		Proposer: "legacy-guardian",
	})
	if err != nil {
		t.Fatal(err)
	}
	legacy := &types.EmergencyCeremony{
		Id:                "legacy-genesis-revert",
		Type:              string(types.CeremonyRevert),
		Phase:             string(types.PhasePrevote),
		ProposalData:      proposalData,
		StartBlock:        1,
		PrevoteDeadline:   ^uint64(0) - 2,
		PrecommitDeadline: ^uint64(0) - 1,
		TimeoutDeadline:   ^uint64(0),
		YesPrevoteStake:   "0",
		NoPrevoteStake:    "0",
		PrecommitStake:    "0",
	}
	genState := types.DefaultGenesis()
	genState.Params = k.GetParams(ctx)
	genState.Status = string(types.StatusRevertVoting)
	genState.Ceremonies = []*types.EmergencyCeremony{legacy}

	k2, mock2, ctx2 := setupKeeper(t)
	k2.InitGenesis(ctx2, genState)
	if got := k2.GetEmergencyStatus(ctx2); got != types.StatusHalted {
		t.Fatalf("legacy revert voting import must remain quarantined, got %s", got)
	}
	imported, found := k2.GetCeremony(ctx2, legacy.Id)
	if !found || imported.Phase != string(types.PhaseFailed) {
		t.Fatalf("legacy revert ceremony must be terminalized during import: %+v", imported)
	}
	if _, found := k2.GetActiveCeremony(ctx2); found {
		t.Fatal("legacy revert import must not retain an active ceremony")
	}
	guardian := testCouncilAddr(21)
	mock2.addGuardian(guardian, "100000000000")
	resumeCtx := ctx2.WithBlockHeight(ctx2.BlockHeight() + 1)
	if _, err := keeper.NewMsgServerImpl(k2).ProposeResume(resumeCtx, testResumeMsg(guardian)); err != nil {
		t.Fatalf("terminalized legacy revert must admit a fresh evidence-bound resume: %v", err)
	}
}

func TestGenesisRejectsInconsistentAndMultipleActiveCeremonies(t *testing.T) {
	k, mock, ctx := setupKeeper(t)
	guardian := testCouncilAddr(22)
	mock.addGuardian(guardian, "100000000000")

	first, err := k.CreateHaltCeremony(ctx, &types.EmergencyHaltProposal{
		Id:              "active-halt-one",
		Proposer:        guardian,
		Reason:          "malicious genesis fixture one",
		ProposedAtBlock: uint64(ctx.BlockHeight()),
	})
	if err != nil {
		t.Fatal(err)
	}
	second, err := k.CreateHaltCeremony(ctx, &types.EmergencyHaltProposal{
		Id:              "active-halt-two",
		Proposer:        guardian,
		Reason:          "malicious genesis fixture two",
		ProposedAtBlock: uint64(ctx.BlockHeight()),
	})
	if err != nil {
		t.Fatal(err)
	}

	params := k.GetParams(ctx)
	inconsistent := &types.GenesisState{
		Params:     params,
		Status:     string(types.StatusHalted),
		Ceremonies: []*types.EmergencyCeremony{first},
	}
	if err := inconsistent.Validate(); err == nil {
		t.Fatal("halted genesis with an active halt ceremony must be rejected")
	}

	multiple := &types.GenesisState{
		Params:     params,
		Status:     string(types.StatusHaltVoting),
		Ceremonies: []*types.EmergencyCeremony{first, second},
	}
	if err := multiple.Validate(); err == nil {
		t.Fatal("genesis with multiple active ceremonies must be rejected")
	}

	forgedTally := proto.Clone(first).(*types.EmergencyCeremony)
	forgedTally.YesPrevoteStake = "100000000000"
	forged := &types.GenesisState{
		Params:     params,
		Status:     string(types.StatusHaltVoting),
		Ceremonies: []*types.EmergencyCeremony{forgedTally},
	}
	if err := forged.Validate(); err == nil {
		t.Fatal("genesis with a vote tally not derived from snapshot members must be rejected")
	}

	partialSnapshot := proto.Clone(first).(*types.EmergencyCeremony)
	partialSnapshot.Electorate = nil
	partial := &types.GenesisState{
		Params:     params,
		Status:     string(types.StatusHaltVoting),
		Ceremonies: []*types.EmergencyCeremony{partialSnapshot},
	}
	if err := partial.Validate(); err == nil {
		t.Fatal("genesis with a partial electorate snapshot must be rejected")
	}
}

func TestGenesisRejectsSnapshottedResumeWithoutExplicitQuarantineLink(t *testing.T) {
	k, mock, ctx := setupKeeper(t)
	guardian := testCouncilAddr(23)
	mock.addGuardian(guardian, "100000000000")

	halt, err := k.CreateHaltCeremony(ctx, &types.EmergencyHaltProposal{
		Id:              "finalized-halt-for-resume",
		Proposer:        guardian,
		Reason:          "anchor imported resume",
		ProposedAtBlock: uint64(ctx.BlockHeight()),
	})
	if err != nil {
		t.Fatal(err)
	}
	halt.Phase = string(types.PhaseFinalized)
	if err := k.SetCeremony(ctx, halt); err != nil {
		t.Fatal(err)
	}
	k.SetEmergencyStatus(ctx, types.StatusHalted)
	k.SetActiveHaltCeremonyId(ctx, halt.Id)
	k.SetHaltStartBlock(ctx, 50)

	resumeCtx := ctx.WithBlockHeight(ctx.BlockHeight() + 1)
	if _, err := keeper.NewMsgServerImpl(k).ProposeResume(resumeCtx, testResumeMsg(guardian)); err != nil {
		t.Fatal(err)
	}
	exported := k.ExportGenesis(ctx)
	if err := exported.Validate(); err != nil {
		t.Fatalf("legitimate active resume export is invalid: %v", err)
	}

	exported.ActiveHaltCeremonyId = ""
	exported.HaltStartBlock = 0
	if err := exported.Validate(); err == nil {
		t.Fatal("snapshotted active resume without explicit quarantine linkage must be rejected")
	}
}

func TestFailedHaltCannotReopenExistingQuarantine(t *testing.T) {
	k, mock, ctx := setupKeeper(t)
	guardian := testCouncilAddr(24)
	mock.addGuardian(guardian, "100000000000")
	ceremony, err := k.CreateHaltCeremony(ctx, &types.EmergencyHaltProposal{
		Id:              "failed-halt-under-quarantine",
		Proposer:        guardian,
		Reason:          "state-effect guard",
		ProposedAtBlock: uint64(ctx.BlockHeight()),
	})
	if err != nil {
		t.Fatal(err)
	}
	ceremony.Phase = string(types.PhaseFailed)
	ceremony.FailureReason = "inconsistent imported ceremony"
	if err := k.SetCeremony(ctx, ceremony); err != nil {
		t.Fatal(err)
	}
	k.SetEmergencyStatus(ctx, types.StatusHalted)
	k.SetActiveHaltCeremonyId(ctx, "existing-quarantine")
	k.SetHaltStartBlock(ctx, 50)

	k.HandleCeremonyFailure(ctx, ceremony.Id)

	if got := k.GetEmergencyStatus(ctx); got != types.StatusHalted {
		t.Fatalf("failed halt must not reopen an existing quarantine, got %s", got)
	}
	if got := k.GetActiveHaltCeremonyId(ctx); got != "existing-quarantine" {
		t.Fatalf("failed unrelated halt replaced quarantine link: %q", got)
	}
}

func TestLiveLegacyHaltedStateRepairsQuarantineLinkage(t *testing.T) {
	k, mock, ctx := setupKeeper(t)
	guardian := testCouncilAddr(25)
	mock.addGuardian(guardian, "100000000000")
	ceremony, err := k.CreateHaltCeremony(ctx, &types.EmergencyHaltProposal{
		Id:              "recoverable-finalized-halt",
		Proposer:        guardian,
		Reason:          "legacy live linkage repair",
		ProposedAtBlock: uint64(ctx.BlockHeight()),
	})
	if err != nil {
		t.Fatal(err)
	}
	ceremony.Phase = string(types.PhaseFinalized)
	if err := k.SetCeremony(ctx, ceremony); err != nil {
		t.Fatal(err)
	}
	k.SetEmergencyStatus(ctx, types.StatusHalted)
	k.SetActiveHaltCeremonyId(ctx, "")
	k.ClearHaltStartBlock(ctx)

	am := emergency.NewAppModule(nil, k)
	if err := am.BeginBlock(ctx); err != nil {
		t.Fatal(err)
	}
	if got := k.GetActiveHaltCeremonyId(ctx); got != ceremony.Id {
		t.Fatalf("legacy halted state should recover finalized halt link: got %q want %q", got, ceremony.Id)
	}
	if got := k.GetHaltStartBlock(ctx); got == 0 {
		t.Fatal("legacy halted state should receive a deterministic escalation start")
	}
	resumeCtx := ctx.WithBlockHeight(ctx.BlockHeight() + 1)
	if _, err := keeper.NewMsgServerImpl(k).ProposeResume(resumeCtx, testResumeMsg(guardian)); err != nil {
		t.Fatalf("repaired live quarantine must admit evidence-bound resume: %v", err)
	}
}

func TestMalformedLiveLegacyResumeIsTerminalizedBeforeProgress(t *testing.T) {
	k, _, ctx := setupKeeper(t)
	proposalData, err := proto.Marshal(&types.EmergencyResumeProposal{
		Id:       "legacy-active-resume",
		Proposer: "legacy-guardian",
	})
	if err != nil {
		t.Fatal(err)
	}
	legacy := &types.EmergencyCeremony{
		Id:                "legacy-active-resume",
		Type:              string(types.CeremonyResume),
		Phase:             string(types.PhasePrecommit),
		ProposalData:      proposalData,
		StartBlock:        1,
		PrevoteDeadline:   ^uint64(0) - 2,
		PrecommitDeadline: ^uint64(0) - 1,
		TimeoutDeadline:   ^uint64(0),
		YesPrevoteStake:   "0",
		NoPrevoteStake:    "0",
		PrecommitStake:    "0",
	}
	if err := k.SetCeremony(ctx, legacy); err != nil {
		t.Fatal(err)
	}
	k.SetEmergencyStatus(ctx, types.StatusResumeVoting)

	am := emergency.NewAppModule(nil, k)
	if err := am.BeginBlock(ctx); err != nil {
		t.Fatal(err)
	}
	if got := k.GetEmergencyStatus(ctx); got != types.StatusHalted {
		t.Fatalf("malformed legacy resume must remain quarantined, got %s", got)
	}
	updated, found := k.GetCeremony(ctx, legacy.Id)
	if !found || updated.Phase != string(types.PhaseFailed) {
		t.Fatalf("malformed legacy resume must be terminalized immediately: %+v", updated)
	}
	if got := k.GetActiveHaltCeremonyId(ctx); got == "" {
		t.Fatal("malformed legacy resume normalization must repair quarantine linkage")
	}
}

func TestLiveSnapshottedResumeCannotSelfAssertMissingQuarantineLink(t *testing.T) {
	k, mock, ctx := setupKeeper(t)
	guardian := testCouncilAddr(26)
	mock.addGuardian(guardian, "100000000000")
	k.SetEmergencyStatus(ctx, types.StatusHalted)
	k.SetActiveHaltCeremonyId(ctx, "proposal-asserted-incident")
	k.SetHaltStartBlock(ctx, 50)

	resp, err := keeper.NewMsgServerImpl(k).ProposeResume(ctx, testResumeMsg(guardian))
	if err != nil {
		t.Fatal(err)
	}
	k.SetActiveHaltCeremonyId(ctx, "")
	k.ClearHaltStartBlock(ctx)

	am := emergency.NewAppModule(nil, k)
	if err := am.BeginBlock(ctx); err != nil {
		t.Fatal(err)
	}
	if got := k.GetEmergencyStatus(ctx); got != types.StatusHalted {
		t.Fatalf("unlinked resume must remain quarantined, got %s", got)
	}
	ceremony, found := k.GetCeremony(ctx, resp.ProposalId)
	if !found || ceremony.Phase != string(types.PhaseFailed) {
		t.Fatalf("unlinked resume was not terminalized: %+v", ceremony)
	}
	if got := k.GetActiveHaltCeremonyId(ctx); got != "legacy-genesis-quarantine" {
		t.Fatalf("normalization trusted proposal-supplied link: got %q", got)
	}
}

func TestGuardianCouncilOverlapIsNotDoubleCounted(t *testing.T) {
	k, mock, ctx := setupKeeper(t)
	guardian := testCouncilAddr(1)
	councilOnly := testCouncilAddr(2)
	mock.addGuardian(guardian, "100")

	params := k.GetParams(ctx)
	params.GenesisCouncil = []string{guardian, councilOnly}
	params.CouncilExpiryBlock = 1000
	params.CouncilVirtualStake = "10"
	k.SetParams(ctx, params)

	if got := k.GetGuardianStake(ctx).String(); got != "110" {
		t.Fatalf("guardian/council overlap double-counted in quorum denominator: got %s want 110", got)
	}
	if got := k.GetGuardianEffectiveStake(ctx, guardian).String(); got != "100" {
		t.Fatalf("overlapping guardian effective stake: got %s want 100", got)
	}
	readiness := k.GetGuardianReadiness(ctx)
	if readiness.EligibleGuardians != 2 || readiness.EffectiveStake.String() != "110" {
		t.Fatalf("readiness disagrees with quorum accounting: %+v", readiness)
	}
}

func TestResumeSnapshotDoesNotShrinkAtCouncilExpiry(t *testing.T) {
	k, _, ctx := setupKeeper(t)
	council := []string{
		testCouncilAddr(1),
		testCouncilAddr(2),
		testCouncilAddr(3),
		testCouncilAddr(4),
		testCouncilAddr(5),
		testCouncilAddr(6),
	}
	params := k.GetParams(ctx)
	params.GenesisCouncil = council
	params.CouncilExpiryBlock = 105
	params.CouncilVirtualStake = "10"
	params.ResumeQuorum = 800000
	params.MinDistinctVoters = 4
	k.SetParams(ctx, params)
	k.SetEmergencyStatus(ctx, types.StatusHalted)
	k.SetActiveHaltCeremonyId(ctx, "council-expiry-quarantine")
	k.SetHaltStartBlock(ctx, 50)

	msgSvr := keeper.NewMsgServerImpl(k)
	resp, err := msgSvr.ProposeResume(ctx, testResumeMsg(council[0]))
	if err != nil {
		t.Fatalf("propose resume: %v", err)
	}
	for _, voter := range council[:5] {
		if _, err := msgSvr.VoteResume(ctx, &types.MsgVoteResume{
			Voter:      voter,
			ProposalId: resp.ProposalId,
			Approve:    true,
		}); err != nil {
			t.Fatalf("prevote %s: %v", voter, err)
		}
	}
	for _, voter := range council[:4] {
		if _, err := msgSvr.VoteResume(ctx, &types.MsgVoteResume{
			Voter:      voter,
			ProposalId: resp.ProposalId,
			Approve:    true,
		}); err != nil {
			t.Fatalf("precommit %s: %v", voter, err)
		}
	}

	ctxAfterExpiry := ctx.WithBlockHeight(106)
	am := emergency.NewAppModule(nil, k)
	if err := am.BeginBlock(ctxAfterExpiry); err != nil {
		t.Fatal(err)
	}
	ceremony, found := k.GetCeremony(ctxAfterExpiry, resp.ProposalId)
	if !found || ceremony.Phase != string(types.PhasePrecommit) {
		t.Fatalf("4/6 precommits must remain below the immutable 80%% quorum: %+v", ceremony)
	}
	if got := k.GetEmergencyStatus(ctxAfterExpiry); got != types.StatusResumeVoting {
		t.Fatalf("council expiry must not shrink the denominator and reopen admission, got %s", got)
	}
}

func TestResumeSnapshotSurvivesGuardianDeactivationWithoutShrinkingQuorum(t *testing.T) {
	k, mock, ctx := setupKeeper(t)
	guardians := []string{testCouncilAddr(31), testCouncilAddr(32), testCouncilAddr(33), testCouncilAddr(34), testCouncilAddr(35)}
	for _, guardian := range guardians {
		mock.addGuardian(guardian, "100")
	}
	params := k.GetParams(ctx)
	params.ResumeQuorum = 800000
	params.MinDistinctVoters = 4
	k.SetParams(ctx, params)
	k.SetEmergencyStatus(ctx, types.StatusHalted)
	k.SetActiveHaltCeremonyId(ctx, "deactivation-quarantine")
	k.SetHaltStartBlock(ctx, 50)

	msgSvr := keeper.NewMsgServerImpl(k)
	resp, err := msgSvr.ProposeResume(ctx, testResumeMsg(guardians[0]))
	if err != nil {
		t.Fatalf("propose resume: %v", err)
	}
	for _, voter := range guardians[:4] {
		if _, err := msgSvr.VoteResume(ctx, &types.MsgVoteResume{
			Voter:      voter,
			ProposalId: resp.ProposalId,
			Approve:    true,
		}); err != nil {
			t.Fatalf("prevote %s: %v", voter, err)
		}
	}
	for _, voter := range guardians[:3] {
		if _, err := msgSvr.VoteResume(ctx, &types.MsgVoteResume{
			Voter:      voter,
			ProposalId: resp.ProposalId,
			Approve:    true,
		}); err != nil {
			t.Fatalf("precommit %s: %v", voter, err)
		}
	}

	for i := range mock.validators {
		if mock.validators[i].Address != guardians[4] {
			mock.validators[i].IsActive = false
		}
	}
	finalized, err := k.CheckCeremonyProgress(ctx, resp.ProposalId)
	if err != nil {
		t.Fatal(err)
	}
	if finalized {
		t.Fatal("3/5 snapshotted precommits must not become quorum after guardian deactivation")
	}
	if got := k.GetEmergencyStatus(ctx); got != types.StatusResumeVoting {
		t.Fatalf("guardian deactivation must not reopen admission, got %s", got)
	}

	// A signer in the immutable electorate remains eligible for this short
	// ceremony even if later staking state changes; the original 4/5 policy
	// is still required.
	result, err := msgSvr.VoteResume(ctx, &types.MsgVoteResume{
		Voter:      guardians[3],
		ProposalId: resp.ProposalId,
		Approve:    true,
	})
	if err != nil {
		t.Fatalf("snapshotted fourth precommit: %v", err)
	}
	if !result.ChainResumed || k.GetEmergencyStatus(ctx) != types.StatusNormal {
		t.Fatal("exact original 4/5 electorate quorum should finalize")
	}
}

func TestResumeSnapshotRequiresGuardianStakeFloor(t *testing.T) {
	k, mock, ctx := setupKeeper(t)
	guardian := testCouncilAddr(40)
	mock.addGuardian(guardian, "100")
	params := k.GetParams(ctx)
	params.MinGuardianStake = "1000"
	k.SetParams(ctx, params)
	k.SetEmergencyStatus(ctx, types.StatusHalted)
	k.SetActiveHaltCeremonyId(ctx, "stake-floor-quarantine")
	k.SetHaltStartBlock(ctx, 50)

	_, err := keeper.NewMsgServerImpl(k).ProposeResume(ctx, testResumeMsg(guardian))
	if err == nil {
		t.Fatal("below-floor electorate must not open a resume ceremony")
	}
	if got := k.GetEmergencyStatus(ctx); got != types.StatusHalted {
		t.Fatalf("failed resume admission must remain quarantined, got %s", got)
	}
	if _, found := k.GetActiveCeremony(ctx); found {
		t.Fatal("failed resume admission must not persist a ceremony")
	}
}

func TestResumeSnapshotIgnoresMidCeremonyParameterReduction(t *testing.T) {
	k, mock, ctx := setupKeeper(t)
	guardians := []string{testCouncilAddr(41), testCouncilAddr(42), testCouncilAddr(43), testCouncilAddr(44), testCouncilAddr(45)}
	for _, guardian := range guardians {
		mock.addGuardian(guardian, "100")
	}
	params := k.GetParams(ctx)
	params.ResumeQuorum = 800000
	params.MinDistinctVoters = 4
	k.SetParams(ctx, params)
	k.SetEmergencyStatus(ctx, types.StatusHalted)
	k.SetActiveHaltCeremonyId(ctx, "parameter-quarantine")
	k.SetHaltStartBlock(ctx, 50)

	msgSvr := keeper.NewMsgServerImpl(k)
	resp, err := msgSvr.ProposeResume(ctx, testResumeMsg(guardians[0]))
	if err != nil {
		t.Fatal(err)
	}
	for _, voter := range guardians[:4] {
		if _, err := msgSvr.VoteResume(ctx, &types.MsgVoteResume{
			Voter: voter, ProposalId: resp.ProposalId, Approve: true,
		}); err != nil {
			t.Fatal(err)
		}
	}

	params = k.GetParams(ctx)
	params.ResumeQuorum = 1
	params.MinDistinctVoters = 1
	k.SetParams(ctx, params)
	if _, err := msgSvr.VoteResume(ctx, &types.MsgVoteResume{
		Voter: guardians[0], ProposalId: resp.ProposalId, Approve: true,
	}); err != nil {
		t.Fatal(err)
	}
	ceremony, _ := k.GetCeremony(ctx, resp.ProposalId)
	if ceremony.Phase != string(types.PhasePrecommit) {
		t.Fatalf("snapshotted 80%%/4-voter policy must survive param reduction: %+v", ceremony)
	}
	if got := k.GetEmergencyStatus(ctx); got != types.StatusResumeVoting {
		t.Fatalf("parameter reduction must not reopen admission, got %s", got)
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

	g1 := testCouncilAddr(1)
	g2 := testCouncilAddr(2)
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

	// A malformed/imported future marker must fail closed instead of
	// underflowing the unsigned subtraction and bypassing the cooldown.
	k.SetLastProposalBlock(ctx, 200)
	_, err = msgSvr.ProposeHalt(ctx, &types.MsgProposeHalt{
		Proposer: g2,
		Reason:   "future marker must not bypass cooldown",
	})
	if err == nil || !strings.Contains(err.Error(), "ahead of current block") {
		t.Fatalf("expected future cooldown marker rejection, got %v", err)
	}
	k.SetLastProposalBlock(ctx, 100)

	// Try at block 160 (after cooldown)
	ctx = ctx.WithBlockHeight(160)
	_, err = msgSvr.ProposeHalt(ctx, &types.MsgProposeHalt{Proposer: g2, Reason: "after cooldown"})
	if err != nil {
		t.Fatalf("should succeed after cooldown: %v", err)
	}
}

func TestQueryStatus(t *testing.T) {
	k, mock, ctx := setupKeeper(t)
	mock.addGuardian(testCouncilAddr(10), "10")

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
	if resp.RestrictionScope != "application_transactions" || !resp.ConsensusContinues {
		t.Fatalf("unexpected restriction semantics: %+v", resp)
	}
	if !resp.HaltCeremonyReady || resp.EligibleGuardians != 1 {
		t.Fatalf("expected structurally ready custom guardian electorate: %+v", resp)
	}
	if resp.AutomaticResumeEnabled || resp.ArbitraryStateRevertEnabled {
		t.Fatalf("unsafe recovery capability must remain disabled: %+v", resp)
	}

	params := k.GetParams(ctx)
	params.MaxHaltDurationBlocks = 10
	k.SetParams(ctx, params)
	k.SetEmergencyStatus(ctx, types.StatusHalted)
	k.SetHaltStartBlock(ctx, 50)

	resp, err = querySvr.Status(ctx, &types.QueryStatusRequest{})
	if err != nil {
		t.Fatalf("query halted Status failed: %v", err)
	}
	if !resp.IsHalted || !resp.QuarantineDeadlineExceeded ||
		resp.QuarantineStartedAtBlock != 50 {
		t.Fatalf("expected overdue transaction quarantine telemetry: %+v", resp)
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

	g1 := testCouncilAddr(1)
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

	councilMember := testCouncilAddr(3)

	// Set params with genesis council active
	params := k.GetParams(ctx)
	params.GenesisCouncil = []string{councilMember}
	params.CouncilExpiryBlock = 1000 // expires at block 1000
	params.CouncilVirtualStake = "50000000000"
	params.MinGuardianStake = "1"
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

	councilMember := testCouncilAddr(3)

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

func TestMinDistinctVotersRejectsImpossibleElectorate(t *testing.T) {
	k, mock, ctx := setupKeeper(t)

	// One guardian cannot satisfy a three-identity policy. Reject the
	// ceremony at admission instead of entering a vote that can only time out.
	g1 := testCouncilAddr(1)
	mock.addGuardian(g1, "100000000000")

	params := k.GetParams(ctx)
	params.HaltQuorum = 500000     // 50% — g1 alone meets this
	params.MinDistinctVoters = 3   // But we require 3 distinct voters
	params.HaltTimeoutBlocks = 10  // Short, coherent overall timeout
	params.HaltPrevoteBlocks = 5   // Short prevote window
	params.HaltPrecommitBlocks = 5 // Short precommit window
	k.SetParams(ctx, params)

	msgSvr := keeper.NewMsgServerImpl(k)
	if readiness := k.GetGuardianReadiness(ctx); readiness.Ready {
		t.Fatal("impossible electorate must not report ready")
	}
	if _, err := msgSvr.ProposeHalt(ctx, &types.MsgProposeHalt{
		Proposer: g1,
		Reason:   "min voters test",
	}); err == nil {
		t.Fatal("expected impossible electorate to be rejected at admission")
	}
	if got := k.GetEmergencyStatus(ctx); got != types.StatusNormal {
		t.Fatalf("rejected ceremony changed emergency status to %s", got)
	}
	if _, found := k.GetActiveCeremony(ctx); found {
		t.Fatal("rejected ceremony persisted active state")
	}
}

func TestEpochCounterReset(t *testing.T) {
	k, _, ctx := setupKeeper(t)

	// Simulate some proposal counts
	k.IncrementGuardianProposalCount(ctx, testCouncilAddr(1))
	k.IncrementGuardianProposalCount(ctx, testCouncilAddr(2))
	k.IncrementEpochProposalCount(ctx)

	if k.GetGuardianProposalCount(ctx, testCouncilAddr(1)) != 1 {
		t.Fatal("expected count 1")
	}
	if k.GetEpochProposalCount(ctx) != 1 {
		t.Fatal("expected epoch count 1")
	}

	// Reset
	k.ResetEpochCounters(ctx)

	if k.GetGuardianProposalCount(ctx, testCouncilAddr(1)) != 0 {
		t.Fatal("expected count 0 after reset")
	}
	if k.GetEpochProposalCount(ctx) != 0 {
		t.Fatal("expected epoch count 0 after reset")
	}
}

// ============================================================
// Ported tests from legible-money prototype — Batch 1
// ============================================================

// --- Guardian / Stake Tests ---

// TestGetGuardianStake verifies total stake calculation across multiple guardians.
func TestGetGuardianStake(t *testing.T) {
	k, mock, ctx := setupKeeper(t)

	mock.addGuardian(testCouncilAddr(1), "100000000000")
	mock.addGuardian(testCouncilAddr(2), "200000000000")
	mock.addGuardian(testCouncilAddr(3), "300000000000")

	total := k.GetGuardianStake(ctx)
	if total.String() != "600000000000" {
		t.Fatalf("expected total guardian stake 600000000000, got %s", total.String())
	}
}

// TestGetGuardianStakeWithCouncil verifies council virtual stake is included during bootstrap.
func TestGetGuardianStakeWithCouncil(t *testing.T) {
	k, mock, ctx := setupKeeper(t)

	mock.addGuardian(testCouncilAddr(1), "100000000000")

	params := k.GetParams(ctx)
	params.GenesisCouncil = []string{testCouncilAddr(4), testCouncilAddr(5)}
	params.CouncilExpiryBlock = 1000
	params.CouncilVirtualStake = "50000000000"
	k.SetParams(ctx, params)

	total := k.GetGuardianStake(ctx)
	// 100B (guardian) + 2 * 50B (council) = 200B
	if total.String() != "200000000000" {
		t.Fatalf("expected total stake 200000000000 (including council), got %s", total.String())
	}
}

// TestGetGuardianEffectiveStake verifies individual guardian stake lookups.
func TestGetGuardianEffectiveStake(t *testing.T) {
	k, mock, ctx := setupKeeper(t)

	mock.addGuardian(testCouncilAddr(1), "100000000000")
	mock.addNonGuardian("zrn1scholar", "50000000000")

	stake := k.GetGuardianEffectiveStake(ctx, testCouncilAddr(1))
	if stake.String() != "100000000000" {
		t.Fatalf("expected guardian stake 100000000000, got %s", stake.String())
	}

	// Non-guardian should return 0
	stake = k.GetGuardianEffectiveStake(ctx, "zrn1scholar")
	if stake.Sign() != 0 {
		t.Fatalf("expected 0 stake for non-guardian, got %s", stake.String())
	}

	// Unknown should return 0
	stake = k.GetGuardianEffectiveStake(ctx, "zrn1unknown")
	if stake.Sign() != 0 {
		t.Fatalf("expected 0 stake for unknown, got %s", stake.String())
	}
}

// --- Ceremony Validation Tests ---

// TestCannotHaltWhenAlreadyHalted verifies halt proposal when already halted is rejected.
func TestCannotHaltWhenAlreadyHalted(t *testing.T) {
	k, mock, ctx := setupKeeper(t)

	g1 := testCouncilAddr(1)
	mock.addGuardian(g1, "100000000000")

	k.SetEmergencyStatus(ctx, types.StatusHalted)

	msgSvr := keeper.NewMsgServerImpl(k)
	_, err := msgSvr.ProposeHalt(ctx, &types.MsgProposeHalt{
		Proposer: g1,
		Reason:   "already halted",
	})
	if err == nil {
		t.Fatal("expected error when proposing halt while already halted")
	}
}

// TestCannotResumeWhenNotHalted verifies resume proposal when status is normal is rejected.
func TestCannotResumeWhenNotHalted(t *testing.T) {
	k, mock, ctx := setupKeeper(t)

	g1 := testCouncilAddr(1)
	mock.addGuardian(g1, "100000000000")

	// Status is normal by default
	msgSvr := keeper.NewMsgServerImpl(k)
	_, err := msgSvr.ProposeResume(ctx, testResumeMsg(g1))
	if err == nil {
		t.Fatal("expected error when proposing resume while not halted")
	}
}

func TestExportGenesisPreservesAntiAbuseState(t *testing.T) {
	k, _, ctx := setupKeeper(t)
	k.SetGuardianProposalCount(ctx, "guardian-a", 1)
	k.SetGuardianProposalCount(ctx, "guardian-b", 2)
	k.SetEpochProposalCount(ctx, 3)
	k.SetLastProposalBlock(ctx, 91)

	exported := k.ExportGenesis(ctx)
	if err := exported.Validate(); err != nil {
		t.Fatalf("exported genesis is invalid: %v", err)
	}

	k2, _, ctx2 := setupKeeper(t)
	k2.InitGenesis(ctx2, exported)
	if got := k2.GetGuardianProposalCount(ctx2, "guardian-a"); got != 1 {
		t.Fatalf("guardian-a count after import = %d, want 1", got)
	}
	if got := k2.GetGuardianProposalCount(ctx2, "guardian-b"); got != 2 {
		t.Fatalf("guardian-b count after import = %d, want 2", got)
	}
	if got := k2.GetEpochProposalCount(ctx2); got != 3 {
		t.Fatalf("epoch count after import = %d, want 3", got)
	}
	if got := k2.GetLastProposalBlock(ctx2); got != 91 {
		t.Fatalf("last proposal block after import = %d, want 91", got)
	}
}

func TestUpdateParamsRejectsLiveStateInconsistency(t *testing.T) {
	t.Run("epoch cap", func(t *testing.T) {
		k, _, ctx := setupKeeper(t)
		k.SetGuardianProposalCount(ctx, "guardian-a", 2)
		k.SetEpochProposalCount(ctx, 2)
		params := k.GetParams(ctx)
		params.MaxProposalsPerEpoch = 1

		_, err := keeper.NewMsgServerImpl(k).UpdateParams(ctx, &types.MsgUpdateParams{
			Authority: "authority",
			Params:    params,
		})
		if err == nil {
			t.Fatal("expected proposed epoch cap below persisted count to fail")
		}
	})

	t.Run("guardian cap", func(t *testing.T) {
		k, _, ctx := setupKeeper(t)
		k.SetGuardianProposalCount(ctx, "guardian-a", 2)
		k.SetEpochProposalCount(ctx, 2)
		params := k.GetParams(ctx)
		params.MaxProposalsPerGuardianPerEpoch = 1

		_, err := keeper.NewMsgServerImpl(k).UpdateParams(ctx, &types.MsgUpdateParams{
			Authority: "authority",
			Params:    params,
		})
		if err == nil {
			t.Fatal("expected proposed guardian cap below persisted count to fail")
		}
	})

	t.Run("escalation marker", func(t *testing.T) {
		k, _, ctx := setupKeeper(t)
		current := k.GetParams(ctx)
		current.MaxHaltDurationBlocks = 10
		k.SetParams(ctx, current)
		k.SetEmergencyStatus(ctx, types.StatusHalted)
		k.SetActiveHaltCeremonyId(ctx, "legacy-quarantine")
		k.SetHaltStartBlock(ctx, 10)
		k.SetLastHaltEscalationBlock(ctx, 20)

		proposed := k.GetParams(ctx)
		proposed.MaxHaltDurationBlocks = 11
		_, err := keeper.NewMsgServerImpl(k).UpdateParams(ctx, &types.MsgUpdateParams{
			Authority: "authority",
			Params:    proposed,
		})
		if err == nil {
			t.Fatal("expected deadline beyond persisted escalation marker to fail")
		}
		if got := k.GetParams(ctx).MaxHaltDurationBlocks; got != 10 {
			t.Fatalf("rejected update mutated params: got duration %d", got)
		}
	})
}

func TestUpdateParamsAcceptedStateRemainsExportable(t *testing.T) {
	k, _, ctx := setupKeeper(t)
	k.SetGuardianProposalCount(ctx, "guardian-a", 2)
	k.SetEpochProposalCount(ctx, 2)
	params := k.GetParams(ctx)
	params.MaxProposalsPerEpoch = 2
	params.MaxProposalsPerGuardianPerEpoch = 2

	if _, err := keeper.NewMsgServerImpl(k).UpdateParams(ctx, &types.MsgUpdateParams{
		Authority: "authority",
		Params:    params,
	}); err != nil {
		t.Fatalf("state-compatible parameter update failed: %v", err)
	}
	if err := k.ExportGenesis(ctx).Validate(); err != nil {
		t.Fatalf("accepted params made exported state invalid: %v", err)
	}
}

func TestUpdateParamsGenesisCouncilCanOnlyRetireMonotonically(t *testing.T) {
	t.Run("expired council cannot be resurrected", func(t *testing.T) {
		k, _, ctx := setupKeeper(t)
		current := k.GetParams(ctx)
		current.GenesisCouncil = []string{testCouncilAddr(41)}
		current.CouncilExpiryBlock = 101
		current.CouncilVirtualStake = "10"
		k.SetParams(ctx, current)

		expiredCtx := ctx.WithBlockHeight(101)
		proposed := k.GetParams(expiredCtx)
		proposed.GenesisCouncil = []string{testCouncilAddr(42)}
		proposed.CouncilExpiryBlock = 1_000
		proposed.CouncilVirtualStake = "20"
		_, err := keeper.NewMsgServerImpl(k).UpdateParams(
			expiredCtx,
			&types.MsgUpdateParams{
				Authority: "authority",
				Params:    proposed,
			},
		)
		if err == nil || !strings.Contains(err.Error(), "cannot be re-enabled") {
			t.Fatalf("expired council resurrection returned %v", err)
		}
	})

	for _, testCase := range []struct {
		name   string
		mutate func(*types.Params)
		match  string
	}{
		{
			name: "member replacement",
			mutate: func(params *types.Params) {
				params.GenesisCouncil[0] = testCouncilAddr(43)
			},
			match: "membership is immutable",
		},
		{
			name: "virtual power change",
			mutate: func(params *types.Params) {
				params.CouncilVirtualStake = "11"
			},
			match: "virtual stake is immutable",
		},
		{
			name: "expiry extension",
			mutate: func(params *types.Params) {
				params.CouncilExpiryBlock = 301
			},
			match: "expiry cannot increase",
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			k, _, ctx := setupKeeper(t)
			current := k.GetParams(ctx)
			current.GenesisCouncil = []string{testCouncilAddr(44)}
			current.CouncilExpiryBlock = 300
			current.CouncilVirtualStake = "10"
			k.SetParams(ctx, current)
			proposed := k.GetParams(ctx)
			testCase.mutate(proposed)

			_, err := keeper.NewMsgServerImpl(k).UpdateParams(
				ctx,
				&types.MsgUpdateParams{
					Authority: "authority",
					Params:    proposed,
				},
			)
			if err == nil || !strings.Contains(err.Error(), testCase.match) {
				t.Fatalf("non-monotonic council update returned %v", err)
			}
		})
	}

	t.Run("expiry may shorten and council may clear", func(t *testing.T) {
		k, _, ctx := setupKeeper(t)
		current := k.GetParams(ctx)
		current.GenesisCouncil = []string{testCouncilAddr(45)}
		current.CouncilExpiryBlock = 300
		current.CouncilVirtualStake = "10"
		k.SetParams(ctx, current)

		shortened := k.GetParams(ctx)
		shortened.CouncilExpiryBlock = 200
		if _, err := keeper.NewMsgServerImpl(k).UpdateParams(
			ctx,
			&types.MsgUpdateParams{
				Authority: "authority",
				Params:    shortened,
			},
		); err != nil {
			t.Fatalf("shorten council expiry: %v", err)
		}

		cleared := k.GetParams(ctx)
		cleared.GenesisCouncil = nil
		cleared.CouncilExpiryBlock = 0
		if _, err := keeper.NewMsgServerImpl(k).UpdateParams(
			ctx,
			&types.MsgUpdateParams{
				Authority: "authority",
				Params:    cleared,
			},
		); err != nil {
			t.Fatalf("clear council: %v", err)
		}
	})
}

func TestElectorateRejectsUnsignableAddresses(t *testing.T) {
	t.Run("snapshot construction", func(t *testing.T) {
		k, mock, ctx := setupKeeper(t)
		mock.addGuardian("zrn1not-a-valid-account", "100000000000")
		_, err := k.CreateHaltCeremony(ctx, &types.EmergencyHaltProposal{
			Id:       "invalid-electorate-builder",
			Proposer: "zrn1not-a-valid-account",
			Reason:   "must not count unsignable power",
		})
		if err == nil {
			t.Fatal("expected unsignable guardian address to fail snapshot construction")
		}
	})

	t.Run("runtime validation", func(t *testing.T) {
		k, mock, ctx := setupKeeper(t)
		guardian := testCouncilAddr(61)
		mock.addGuardian(guardian, "100000000000")
		ceremony, err := k.CreateHaltCeremony(ctx, &types.EmergencyHaltProposal{
			Id:       "invalid-electorate-runtime",
			Proposer: guardian,
			Reason:   "detect corrupted live snapshot",
		})
		if err != nil {
			t.Fatal(err)
		}
		ceremony.Electorate[0].Address = "zrn1not-a-valid-account"
		if err := k.SetCeremony(ctx, ceremony); err != nil {
			t.Fatal(err)
		}
		if finalized, err := k.CheckCeremonyProgress(ctx, ceremony.Id); err != nil || finalized {
			t.Fatalf("corrupt runtime snapshot must terminalize without finalizing: finalized=%v err=%v", finalized, err)
		}
		failed, found := k.GetCeremony(ctx, ceremony.Id)
		if !found || failed.Phase != string(types.PhaseFailed) {
			t.Fatalf("corrupt runtime snapshot was not terminalized: %+v", failed)
		}
		if err := k.ExportGenesis(ctx).Validate(); err == nil {
			t.Fatal("expected genesis validation to reject unsignable electorate member")
		}
	})
}

func TestEmergencyElectorateConsensusMaximum(t *testing.T) {
	k, mock, ctx := setupKeeper(t)
	guardians := make([]string, 0, types.MaxEmergencyElectorateSize)
	for index := 0; index < types.MaxEmergencyElectorateSize; index++ {
		guardian := testCouncilAddr(byte(index + 1))
		guardians = append(guardians, guardian)
		mock.addGuardian(guardian, "1")
	}

	msgServer := keeper.NewMsgServerImpl(k)
	response, err := msgServer.ProposeHalt(ctx, &types.MsgProposeHalt{
		Proposer: guardians[0],
		Reason:   "exact maximum electorate",
	})
	if err != nil {
		t.Fatalf(
			"exact consensus maximum %d was rejected: %v",
			types.MaxEmergencyElectorateSize,
			err,
		)
	}
	ceremony, found := k.GetCeremony(ctx, response.ProposalId)
	if !found || len(ceremony.Electorate) != types.MaxEmergencyElectorateSize {
		t.Fatalf("exact-boundary snapshot = %+v", ceremony)
	}
	exported := k.ExportGenesis(ctx)
	if err := exported.Validate(); err != nil {
		t.Fatalf("exact-boundary genesis was rejected: %v", err)
	}
	if terminalized, err := k.MigrateOperationsSafetyV1(ctx); err != nil {
		t.Fatalf("exact-boundary migration failed: %v", err)
	} else if terminalized != 0 {
		t.Fatalf("exact-boundary migration terminalized %d ceremonies", terminalized)
	}

	overBoundGenesis := proto.Clone(exported).(*types.GenesisState)
	overBoundGenesis.Ceremonies[0].Electorate = append(
		overBoundGenesis.Ceremonies[0].Electorate,
		&types.EmergencyElectorateMember{
			Address: testCouncilAddr(0xfe),
			Power:   "1",
		},
	)
	if err := overBoundGenesis.Validate(); err == nil ||
		!strings.Contains(err.Error(), "exceeds consensus maximum") {
		t.Fatalf("over-bound genesis electorate was accepted: %v", err)
	}

	ceremony, _ = k.GetCeremony(ctx, response.ProposalId)
	ceremony.Electorate = append(
		ceremony.Electorate,
		&types.EmergencyElectorateMember{
			Address: testCouncilAddr(0xfe),
			Power:   "1",
		},
	)
	if err := k.SetCeremony(ctx, ceremony); err != nil {
		t.Fatal(err)
	}
	_, err = k.MigrateOperationsSafetyV1(ctx)
	if err == nil || !strings.Contains(err.Error(), "exceeds consensus maximum") {
		t.Fatalf("over-bound migration electorate was accepted: %v", err)
	}
	migrated, _ := k.GetCeremony(ctx, response.ProposalId)
	if migrated.Phase != string(types.PhasePrevote) ||
		k.GetEmergencyStatus(ctx) != types.StatusHaltVoting {
		t.Fatalf(
			"failed migration partially mutated state: phase=%s status=%s",
			migrated.Phase,
			k.GetEmergencyStatus(ctx),
		)
	}

	k2, mock2, ctx2 := setupKeeper(t)
	for index := 0; index <= types.MaxEmergencyElectorateSize; index++ {
		mock2.addGuardian(testCouncilAddr(byte(index+1)), "1")
	}
	if readiness := k2.GetGuardianReadiness(ctx2); readiness.Ready ||
		!strings.Contains(readiness.Reason, "consensus maximum") {
		t.Fatalf("over-bound electorate reported ready: %+v", readiness)
	}
	if _, err := keeper.NewMsgServerImpl(k2).ProposeHalt(
		ctx2,
		&types.MsgProposeHalt{
			Proposer: testCouncilAddr(1),
			Reason:   "over maximum electorate",
		},
	); err == nil || !strings.Contains(err.Error(), "exceeds consensus maximum") {
		t.Fatalf("over-bound snapshot construction was accepted: %v", err)
	}
	if _, found := k2.GetActiveCeremony(ctx2); found {
		t.Fatal("over-bound snapshot construction persisted an active ceremony")
	}
}

func TestOperationsSafetyMigrationReconcilesLegacyEmergencyState(t *testing.T) {
	t.Run("empty status", func(t *testing.T) {
		k, _, ctx := setupKeeper(t)
		terminalized, err := k.MigrateOperationsSafetyV1(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if terminalized != 0 || k.GetEmergencyStatus(ctx) != types.StatusNormal {
			t.Fatalf("empty legacy status was not normalized: count=%d status=%s", terminalized, k.GetEmergencyStatus(ctx))
		}
	})

	t.Run("multiple legacy active ceremonies", func(t *testing.T) {
		k, _, ctx := setupKeeper(t)
		k.SetEmergencyStatus(ctx, types.StatusHaltVoting)
		for _, id := range []string{"legacy-b", "legacy-a"} {
			if err := k.SetCeremony(ctx, &types.EmergencyCeremony{
				Id:    id,
				Type:  string(types.CeremonyHalt),
				Phase: string(types.PhasePrevote),
			}); err != nil {
				t.Fatal(err)
			}
		}

		terminalized, err := k.MigrateOperationsSafetyV1(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if terminalized != 2 || k.GetEmergencyStatus(ctx) != types.StatusNormal {
			t.Fatalf("legacy collision reconciliation = count %d status %s", terminalized, k.GetEmergencyStatus(ctx))
		}
		for _, id := range []string{"legacy-a", "legacy-b"} {
			ceremony, found := k.GetCeremony(ctx, id)
			if !found || ceremony.Phase != string(types.PhaseFailed) {
				t.Fatalf("legacy ceremony %q was not terminalized: %+v", id, ceremony)
			}
		}
	})

	t.Run("multiple equally valid snapshotted ceremonies fail closed", func(t *testing.T) {
		k, mock, ctx := setupKeeper(t)
		guardian := testCouncilAddr(0x72)
		mock.addGuardian(guardian, "100000000000")
		k.SetEmergencyStatus(ctx, types.StatusHaltVoting)
		for _, id := range []string{
			"valid-snapshot-b",
			"valid-snapshot-a",
		} {
			if _, err := k.CreateHaltCeremony(
				ctx,
				&types.EmergencyHaltProposal{
					Id:       id,
					Proposer: guardian,
					Reason:   "ambiguous legacy authority",
				},
			); err != nil {
				t.Fatal(err)
			}
		}

		terminalized, err := k.MigrateOperationsSafetyV1(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if terminalized != 2 ||
			k.GetEmergencyStatus(ctx) != types.StatusNormal {
			t.Fatalf(
				"ambiguous valid candidates invented a survivor: terminalized=%d status=%s",
				terminalized,
				k.GetEmergencyStatus(ctx),
			)
		}
		if _, found := k.GetActiveCeremony(ctx); found {
			t.Fatal("ambiguous legacy candidates retained active authority")
		}
		for _, id := range []string{
			"valid-snapshot-a",
			"valid-snapshot-b",
		} {
			ceremony, found := k.GetCeremony(ctx, id)
			if !found ||
				ceremony.Phase != string(types.PhaseFailed) {
				t.Fatalf(
					"ambiguous ceremony %q was not terminalized: %+v",
					id,
					ceremony,
				)
			}
		}
	})

	t.Run("halt voting with wrong active type", func(t *testing.T) {
		k, _, ctx := setupKeeper(t)
		k.SetEmergencyStatus(ctx, types.StatusHaltVoting)
		if err := k.SetCeremony(ctx, &types.EmergencyCeremony{
			Id:    "legacy-wrong-type",
			Type:  string(types.CeremonyResume),
			Phase: string(types.PhasePrevote),
		}); err != nil {
			t.Fatal(err)
		}

		if _, err := k.MigrateOperationsSafetyV1(ctx); err != nil {
			t.Fatal(err)
		}
		if k.GetEmergencyStatus(ctx) != types.StatusNormal {
			t.Fatalf("malformed halt voting state remained %s", k.GetEmergencyStatus(ctx))
		}
		if _, found := k.GetActiveCeremony(ctx); found {
			t.Fatal("malformed halt voting ceremony remained active")
		}
	})

	t.Run("legacy revert target is retired", func(t *testing.T) {
		k, _, ctx := setupKeeper(t)
		k.SetEmergencyStatus(ctx, types.StatusReverting)
		k.SetRevertTarget(ctx, 77, "legacy-block-hash", "legacy-revert")

		if _, err := k.MigrateOperationsSafetyV1(ctx); err != nil {
			t.Fatal(err)
		}
		if k.GetEmergencyStatus(ctx) != types.StatusHalted {
			t.Fatalf("legacy revert did not become quarantine: %s", k.GetEmergencyStatus(ctx))
		}
		if target, found := k.GetRevertTarget(ctx); found {
			t.Fatalf("retired revert target survived activation: %+v", target)
		}
		foundAudit := false
		for _, entry := range k.GetAuditLog(ctx) {
			if entry.Action == string(types.AuditRevertFailed) &&
				strings.Contains(entry.Details, "height=77") {
				foundAudit = true
			}
		}
		if !foundAudit {
			t.Fatal("revert-target retirement was not audited")
		}
	})
}

func TestEmergencyConsensusV2RequiresNamedHandlerPreparation(t *testing.T) {
	t.Run("unprepared migration fails closed", func(t *testing.T) {
		k, _, ctx := setupKeeper(t)
		err := keeper.NewMigrator(k).Migrate1to2(ctx)
		if err == nil ||
			!strings.Contains(err.Error(), "requires named-handler preparation") {
			t.Fatalf("unprepared migration returned %v", err)
		}
		if _, _, found, readErr := k.GetOperationsSafetyV2Activation(ctx); readErr != nil {
			t.Fatal(readErr)
		} else if found {
			t.Fatal("unprepared migration wrote activation lineage")
		}
	})

	t.Run("preparation is height bound and finalizes once", func(t *testing.T) {
		k, _, ctx := setupKeeper(t)
		snapshot, err := k.ReadOperationsSafetySnapshot(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := k.PrepareOperationsSafetyV2FromSnapshot(
			ctx,
			snapshot,
		); err != nil {
			t.Fatal(err)
		}
		if err := keeper.NewMigrator(k).Migrate1to2(
			ctx.WithBlockHeight(ctx.BlockHeight() + 1),
		); err == nil || !strings.Contains(err.Error(), "does not match") {
			t.Fatalf("wrong-height migration returned %v", err)
		}
		if err := keeper.NewMigrator(k).Migrate1to2(ctx); err != nil {
			t.Fatal(err)
		}
		height, digest, found, err :=
			k.GetOperationsSafetyV2Activation(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if !found || height != uint64(ctx.BlockHeight()) ||
			digest == ([32]byte{}) {
			t.Fatalf(
				"activation lineage found=%t height=%d digest=%x",
				found,
				height,
				digest,
			)
		}
		if err := keeper.NewMigrator(k).Migrate1to2(ctx); err == nil ||
			!strings.Contains(err.Error(), "already exists") {
			t.Fatalf("replayed migration returned %v", err)
		}
	})

	t.Run("source v1 cannot preseed release latch key", func(t *testing.T) {
		k, _, ctx := setupKeeper(t)
		snapshot, err := keeper.NewOperationsSafetySnapshot(
			[]keeper.OperationsSafetyRecord{{
				Key:   types.QuarantineReleaseBlockKey,
				Value: []byte{0, 0, 0, 0, 0, 0, 0, 100},
			}},
		)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := k.PrepareOperationsSafetyV2FromSnapshot(
			ctx,
			snapshot,
		); err == nil || !strings.Contains(err.Error(), "reserved marker") {
			t.Fatalf("preseeded release latch key was accepted: %v", err)
		}
	})
}

func TestOperationsSafetyMigrationPersistsNormalizedLegacyParams(t *testing.T) {
	k, _, ctx := setupKeeper(t)
	legacy := types.DefaultParams()
	legacy.RevertQuorum = 0
	legacy.ResumeQuorum = 0
	legacy.RevertPrevoteBlocks = 0
	legacy.RevertPrecommitBlocks = 0
	legacy.RevertTimeoutBlocks = 0
	legacy.ResumePrevoteBlocks = 0
	legacy.ResumePrecommitBlocks = 0
	legacy.ResumeTimeoutBlocks = 0
	legacy.MaxProposalsPerEpoch = 0
	legacy.MaxProposalsPerGuardianPerEpoch = 0
	legacy.MinGuardianStake = ""
	legacy.MinDistinctVoters = 0
	legacy.MaxRevertDepth = 0
	rawLegacy, err := proto.Marshal(&legacy)
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := keeper.NewOperationsSafetySnapshot(
		[]keeper.OperationsSafetyRecord{{
			Key:   types.ParamsKey,
			Value: rawLegacy,
		}},
	)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := k.MigrateOperationsSafetyV1FromSnapshot(ctx, snapshot); err != nil {
		t.Fatal(err)
	}
	persistedSnapshot, err := k.ReadOperationsSafetySnapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var persistedParams *types.Params
	for _, record := range persistedSnapshot.Records {
		if bytes.Equal(record.Key, types.ParamsKey) {
			var decoded types.Params
			if err := proto.Unmarshal(record.Value, &decoded); err != nil {
				t.Fatal(err)
			}
			persistedParams = &decoded
			break
		}
	}
	if persistedParams == nil {
		t.Fatal("normalized params were not persisted")
	}
	if err := persistedParams.Validate(); err != nil {
		t.Fatalf("persisted params remain invalid: %v", err)
	}
	if persistedParams.ResumeQuorum == 0 ||
		persistedParams.MaxProposalsPerEpoch == 0 ||
		persistedParams.MinDistinctVoters == 0 {
		t.Fatalf("legacy fields were not normalized on disk: %+v", persistedParams)
	}
}

func TestOperationsSafetyMigrationSelectsResumeLinkedToActiveQuarantine(
	t *testing.T,
) {
	k, mock, ctx := setupKeeper(t)
	guardian := testCouncilAddr(0x71)
	mock.addGuardian(guardian, "100000000000")
	halt, err := k.CreateHaltCeremony(ctx, &types.EmergencyHaltProposal{
		Id:       "active-halt",
		Proposer: guardian,
		Reason:   "authenticated quarantine link",
	})
	if err != nil {
		t.Fatal(err)
	}
	halt.Phase = string(types.PhaseFinalized)
	if err := k.SetCeremony(ctx, halt); err != nil {
		t.Fatal(err)
	}
	k.SetEmergencyStatus(ctx, types.StatusResumeVoting)
	k.SetActiveHaltCeremonyId(ctx, "active-halt")
	k.SetHaltStartBlock(ctx, 90)

	wrong, err := k.CreateResumeCeremony(ctx, &types.EmergencyResumeProposal{
		Id:                     "resume-a-wrong-link",
		Proposer:               guardian,
		HaltCeremonyId:         "other-halt",
		Justification:          "wrong legacy quarantine link",
		RecoveryManifestSha256: testRecoveryManifestSHA256,
	})
	if err != nil {
		t.Fatal(err)
	}
	right, err := k.CreateResumeCeremony(ctx, &types.EmergencyResumeProposal{
		Id:                     "resume-b-right-link",
		Proposer:               guardian,
		HaltCeremonyId:         "active-halt",
		Justification:          "correct legacy quarantine link",
		RecoveryManifestSha256: testRecoveryManifestSHA256,
	})
	if err != nil {
		t.Fatal(err)
	}

	terminalized, err := k.MigrateOperationsSafetyV1(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if terminalized != 1 {
		t.Fatalf("terminalized %d ceremonies, want 1", terminalized)
	}
	if got := k.GetEmergencyStatus(ctx); got != types.StatusResumeVoting {
		t.Fatalf("correctly linked recovery was lost: status=%s", got)
	}
	gotWrong, _ := k.GetCeremony(ctx, wrong.Id)
	gotRight, _ := k.GetCeremony(ctx, right.Id)
	if gotWrong.Phase != string(types.PhaseFailed) ||
		gotRight.Phase != string(types.PhasePrevote) {
		t.Fatalf(
			"wrong survivor selection: wrong=%s right=%s",
			gotWrong.Phase,
			gotRight.Phase,
		)
	}
}

func TestOperationsSafetyMigrationAggregatesLargeLegacyAudit(t *testing.T) {
	k, _, ctx := setupKeeper(t)
	const ceremonyCount = 1_200
	for i := 0; i < ceremonyCount; i++ {
		if err := k.SetCeremony(ctx, &types.EmergencyCeremony{
			Id:    fmt.Sprintf("legacy-%04d", i),
			Type:  string(types.CeremonyHalt),
			Phase: string(types.PhasePrevote),
		}); err != nil {
			t.Fatal(err)
		}
	}

	terminalized, err := k.MigrateOperationsSafetyV1(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if terminalized != ceremonyCount {
		t.Fatalf("terminalized %d ceremonies, want %d", terminalized, ceremonyCount)
	}
	normalizationAudits := 0
	for _, entry := range k.GetAuditLog(ctx) {
		if entry.Action == string(types.AuditLegacyNormalized) {
			normalizationAudits++
		}
	}
	if normalizationAudits > 2 {
		t.Fatalf(
			"large migration emitted %d normalization audits; expected aggregate O(1) emission",
			normalizationAudits,
		)
	}
}

func TestOperationsSafetyMigrationPreservesInferredQuarantineClock(t *testing.T) {
	k, mock, ctx := setupKeeper(t)
	guardian := testCouncilAddr(0x72)
	mock.addGuardian(guardian, "100000000000")
	halt, err := k.CreateHaltCeremony(ctx, &types.EmergencyHaltProposal{
		Id:       "historical-finalized-halt",
		Proposer: guardian,
		Reason:   "old export omitted quarantine linkage",
	})
	if err != nil {
		t.Fatal(err)
	}
	halt.StartBlock = 42
	halt.Phase = string(types.PhaseFinalized)
	if err := k.SetCeremony(ctx, halt); err != nil {
		t.Fatal(err)
	}
	k.SetEmergencyStatus(ctx, types.StatusHalted)
	k.SetActiveHaltCeremonyId(ctx, "")
	k.ClearHaltStartBlock(ctx)

	if _, err := k.MigrateOperationsSafetyV1(ctx); err != nil {
		t.Fatal(err)
	}
	if got := k.GetActiveHaltCeremonyId(ctx); got != halt.Id {
		t.Fatalf("inferred quarantine id=%q want %q", got, halt.Id)
	}
	if got := k.GetHaltStartBlock(ctx); got != halt.StartBlock {
		t.Fatalf("inferred quarantine start=%d want %d", got, halt.StartBlock)
	}
	if err := k.ExportGenesis(ctx).Validate(); err != nil {
		t.Fatalf("migrated halted state is not exportable: %v", err)
	}
}

func TestOperationsSafetyMigrationClearsStaleQuarantineDuringHaltVote(
	t *testing.T,
) {
	k, mock, ctx := setupKeeper(t)
	guardian := testCouncilAddr(0x73)
	mock.addGuardian(guardian, "100000000000")
	active, err := k.CreateHaltCeremony(ctx, &types.EmergencyHaltProposal{
		Id:       "active-halt-vote",
		Proposer: guardian,
		Reason:   "new halt vote is not yet a quarantine",
	})
	if err != nil {
		t.Fatal(err)
	}
	k.SetEmergencyStatus(ctx, types.StatusHaltVoting)
	k.SetActiveHaltCeremonyId(ctx, legacyQuarantineIDForTest)
	k.SetHaltStartBlock(ctx, 10)
	k.SetLastHaltEscalationBlock(ctx, 50_000)

	if _, err := k.MigrateOperationsSafetyV1(ctx); err != nil {
		t.Fatal(err)
	}
	if got := k.GetEmergencyStatus(ctx); got != types.StatusHaltVoting {
		t.Fatalf("halt vote did not survive: %s", got)
	}
	if got := k.GetActiveHaltCeremonyId(ctx); got != "" {
		t.Fatalf("halt vote retained stale quarantine id %q", got)
	}
	if got := k.GetHaltStartBlock(ctx); got != 0 {
		t.Fatalf("halt vote retained stale quarantine start %d", got)
	}
	if got := k.GetLastHaltEscalationBlock(ctx); got != 0 {
		t.Fatalf("halt vote retained stale escalation %d", got)
	}
	indexed, found := k.GetActiveCeremony(ctx)
	if !found || indexed.Id != active.Id {
		t.Fatalf("active ceremony index was not rebuilt: %+v", indexed)
	}
	if err := k.ExportGenesis(ctx).Validate(); err != nil {
		t.Fatalf("migrated halt-voting state is not exportable: %v", err)
	}
}

func TestOperationsSafetyMigrationRejectsMalformedEscalationBytes(t *testing.T) {
	k, _, ctx := setupKeeper(t)
	snapshot, err := keeper.NewOperationsSafetySnapshot(
		[]keeper.OperationsSafetyRecord{{
			Key:   types.LastHaltEscalationBlockKey,
			Value: []byte{0x01},
		}},
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := k.MigrateOperationsSafetyV1FromSnapshot(ctx, snapshot); err == nil ||
		!strings.Contains(err.Error(), "escalation block has length") {
		t.Fatalf("malformed escalation bytes did not fail explicitly: %v", err)
	}
}

func TestOperationsSafetyMigrationDoesNotTrustArbitraryQuarantineID(
	t *testing.T,
) {
	k, _, ctx := setupKeeper(t)
	k.SetEmergencyStatus(ctx, types.StatusHalted)
	k.SetActiveHaltCeremonyId(ctx, "attacker-selected-incident")
	k.SetHaltStartBlock(ctx, 77)

	if _, err := k.MigrateOperationsSafetyV1(ctx); err != nil {
		t.Fatal(err)
	}
	if got := k.GetActiveHaltCeremonyId(ctx); got != legacyQuarantineIDForTest {
		t.Fatalf("unproved quarantine id survived migration: %q", got)
	}
	if got := k.GetHaltStartBlock(ctx); got != 77 {
		t.Fatalf("valid persisted quarantine clock changed: %d", got)
	}
	if err := k.ExportGenesis(ctx).Validate(); err != nil {
		t.Fatalf("normalized quarantine is not exportable: %v", err)
	}
}

func TestExportGenesisIrreversiblyScrubsExpiredCouncil(t *testing.T) {
	k, _, ctx := setupKeeper(t)
	council := testCouncilAddr(0x41)
	params := k.GetParams(ctx)
	params.GenesisCouncil = []string{council}
	params.CouncilExpiryBlock = uint64(ctx.BlockHeight())
	k.SetParams(ctx, params)

	exported := k.ExportGenesis(ctx)
	if len(exported.Params.GenesisCouncil) != 0 || exported.Params.CouncilExpiryBlock != 0 {
		t.Fatalf("expired council survived export: %+v", exported.Params)
	}
	live := k.GetParams(ctx)
	if len(live.GenesisCouncil) != 1 || live.CouncilExpiryBlock != uint64(ctx.BlockHeight()) {
		t.Fatal("export mutated live council parameters")
	}
}

func TestZeroHeightExportRebasesCouncilWithoutExtendingAuthority(t *testing.T) {
	k, _, ctx := setupKeeper(t)
	council := testCouncilAddr(0x42)
	params := k.GetParams(ctx)
	params.GenesisCouncil = []string{council}
	params.CouncilExpiryBlock = 150
	k.SetParams(ctx, params)

	exported, err := k.ExportGenesisForZeroHeight(ctx)
	if err != nil {
		t.Fatalf("zero-height export failed: %v", err)
	}
	if exported.Params.CouncilExpiryBlock != 50 {
		t.Fatalf("rebased council expiry = %d, want 50", exported.Params.CouncilExpiryBlock)
	}
	if live := k.GetParams(ctx); live.CouncilExpiryBlock != 150 {
		t.Fatalf("zero-height export mutated live expiry to %d", live.CouncilExpiryBlock)
	}

	k2, _, ctx2 := setupKeeper(t)
	ctx2 = ctx2.WithBlockHeight(0)
	k2.InitGenesis(ctx2, exported)
	if !k2.IsGuardian(ctx2.WithBlockHeight(49), council) {
		t.Fatal("council should remain active for the exported remaining window")
	}
	if k2.IsGuardian(ctx2.WithBlockHeight(50), council) {
		t.Fatal("council authority was extended beyond the exported remaining window")
	}
}

func TestZeroHeightExportRefusesIncidentStateAndActiveCooldown(t *testing.T) {
	k, _, ctx := setupKeeper(t)
	k.SetEmergencyStatus(ctx, types.StatusHalted)
	k.SetActiveHaltCeremonyId(ctx, "legacy-genesis-quarantine")
	k.SetHaltStartBlock(ctx, uint64(ctx.BlockHeight()))
	if _, err := k.ExportGenesisForZeroHeight(ctx); err == nil {
		t.Fatal("expected zero-height export to refuse active quarantine")
	}

	k.SetEmergencyStatus(ctx, types.StatusNormal)
	k.SetActiveHaltCeremonyId(ctx, "")
	k.ClearHaltStartBlock(ctx)
	params := k.GetParams(ctx)
	params.CooldownBlocks = 10
	k.SetParams(ctx, params)
	k.SetLastProposalBlock(ctx, uint64(ctx.BlockHeight()-5))
	if _, err := k.ExportGenesisForZeroHeight(ctx); err == nil {
		t.Fatal("expected zero-height export to refuse active proposal cooldown")
	}
}

func TestQuarantineReleaseLatchGenesisRoundTripAndZeroHeight(t *testing.T) {
	k, _, ctx := setupKeeper(t)
	releaseBlock := uint64(ctx.BlockHeight())
	k.SetEmergencyStatus(ctx, types.StatusNormal)
	k.SetQuarantineReleaseBlock(ctx, releaseBlock)

	exported := k.ExportGenesis(ctx)
	if exported.QuarantineReleaseBlock != releaseBlock {
		t.Fatalf(
			"exported release block = %d, want %d",
			exported.QuarantineReleaseBlock,
			releaseBlock,
		)
	}
	if err := exported.Validate(); err != nil {
		t.Fatalf("release-latched genesis is invalid: %v", err)
	}

	k2, _, ctx2 := setupKeeper(t)
	k2.InitGenesis(ctx2, exported)
	if got := k2.GetQuarantineReleaseBlock(ctx2); got != releaseBlock {
		t.Fatalf("round-tripped release block = %d, want %d", got, releaseBlock)
	}
	if !k2.IsHalted(ctx2) {
		t.Fatal("round-tripped same-height release latch reopened early")
	}
	if _, err := k.ExportGenesisForZeroHeight(ctx); err == nil ||
		!strings.Contains(err.Error(), "release latch") {
		t.Fatalf("active latch survived zero-height safety check: %v", err)
	}

	expiredCtx := ctx.WithBlockHeight(ctx.BlockHeight() + 1)
	rebased, err := k.ExportGenesisForZeroHeight(expiredCtx)
	if err != nil {
		t.Fatalf("expired release latch could not be rebased: %v", err)
	}
	if rebased.QuarantineReleaseBlock != 0 {
		t.Fatalf(
			"expired absolute release block survived zero-height export: %d",
			rebased.QuarantineReleaseBlock,
		)
	}
	if got := k.GetQuarantineReleaseBlock(ctx); got != releaseBlock {
		t.Fatalf("zero-height export mutated live release block to %d", got)
	}
}

func TestResumeRetryRequiresChangedEvidenceAndAdvancesGeneration(t *testing.T) {
	k, mock, ctx := setupKeeper(t)
	g1 := testCouncilAddr(0x51)
	g2 := testCouncilAddr(0x52)
	mock.addGuardian(g1, "50")
	mock.addGuardian(g2, "50")
	k.SetEmergencyStatus(ctx, types.StatusHalted)
	k.SetActiveHaltCeremonyId(ctx, "halt-incident-a")
	k.SetHaltStartBlock(ctx, uint64(ctx.BlockHeight()))

	msgServer := keeper.NewMsgServerImpl(k)
	resumeCtx := ctx.WithBlockHeight(ctx.BlockHeight() + 1)
	first, err := msgServer.ProposeResume(resumeCtx, testResumeMsg(g1))
	if err != nil {
		t.Fatalf("first resume proposal failed: %v", err)
	}
	if _, err := msgServer.VoteResume(resumeCtx, &types.MsgVoteResume{
		Voter:      g2,
		ProposalId: first.ProposalId,
		Approve:    false,
	}); err != nil {
		t.Fatalf("rejecting hostile resume proposal failed: %v", err)
	}
	if status := k.GetEmergencyStatus(ctx); status != types.StatusHalted {
		t.Fatalf("status after failed resume = %s, want halted", status)
	}

	changedEvidence := testResumeMsg(g1)
	changedEvidence.RecoveryManifestSha256 =
		"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	if _, err := msgServer.ProposeResume(
		resumeCtx,
		changedEvidence,
	); err == nil || !types.ErrCooldownActive.Is(err) {
		t.Fatalf("same-block recovery retry was accepted: %v", err)
	}

	nextCtx := resumeCtx.WithBlockHeight(resumeCtx.BlockHeight() + 1)
	if _, err := msgServer.ProposeResume(nextCtx, testResumeMsg(g1)); err == nil ||
		!types.ErrProposalLimitExceeded.Is(err) {
		t.Fatalf("same evidence reopened a recovery generation: %v", err)
	}

	retry, err := msgServer.ProposeResume(nextCtx, changedEvidence)
	if err != nil {
		t.Fatalf("changed evidence must open the next recovery generation: %v", err)
	}
	if !strings.Contains(retry.ProposalId, "-g2-") {
		t.Fatalf("retry id does not expose generation 2: %q", retry.ProposalId)
	}

	thirdEvidence := testResumeMsg(g2)
	thirdEvidence.RecoveryManifestSha256 =
		"cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
	if _, err := msgServer.ProposeResume(nextCtx, thirdEvidence); err == nil {
		t.Fatalf("duplicate active resume proposal was not blocked: %v", err)
	}
	active, found := k.GetActiveCeremony(nextCtx)
	if !found || active.Id != retry.ProposalId {
		t.Fatalf("duplicate proposal displaced active generation: %+v", active)
	}
}

func TestResumeRateLimitStopsCeremonyMonopoly(t *testing.T) {
	k, mock, ctx := setupKeeper(t)
	attacker := testCouncilAddr(0x56)
	honest := testCouncilAddr(0x57)
	mock.addGuardian(attacker, "50")
	mock.addGuardian(honest, "50")
	params := k.GetParams(ctx)
	params.MaxProposalsPerGuardianPerEpoch = 1
	params.MaxProposalsPerEpoch = 3
	params.CooldownBlocks = 5
	k.SetParams(ctx, params)
	k.SetEmergencyStatus(ctx, types.StatusHalted)
	k.SetActiveHaltCeremonyId(ctx, "halt-rate-limit-incident")
	k.SetHaltStartBlock(ctx, uint64(ctx.BlockHeight()))

	server := keeper.NewMsgServerImpl(k)
	firstCtx := ctx.WithBlockHeight(ctx.BlockHeight() + 1)
	first, err := server.ProposeResume(firstCtx, testResumeMsg(attacker))
	if err != nil {
		t.Fatalf("first resume proposal failed: %v", err)
	}
	if _, err := server.VoteResume(
		firstCtx,
		&types.MsgVoteResume{
			Voter:      honest,
			ProposalId: first.ProposalId,
			Approve:    false,
		},
	); err != nil {
		t.Fatalf("rejecting hostile resume failed: %v", err)
	}
	if status := k.GetEmergencyStatus(firstCtx); status != types.StatusHalted {
		t.Fatalf("failed resume left status %s, want halted", status)
	}

	retryCtx := firstCtx.WithBlockHeight(firstCtx.BlockHeight() + 1)
	changed := testResumeMsg(attacker)
	changed.RecoveryManifestSha256 = strings.Repeat("b", 64)
	if _, err := server.ProposeResume(
		retryCtx,
		changed,
	); err == nil || !types.ErrProposalLimitExceeded.Is(err) {
		t.Fatalf("attacker reopened the sole resume ceremony: %v", err)
	}

	recovery := testResumeMsg(honest)
	recovery.RecoveryManifestSha256 = strings.Repeat("c", 64)
	if _, err := server.ProposeResume(
		retryCtx,
		recovery,
	); err == nil || !types.ErrCooldownActive.Is(err) {
		t.Fatalf("shared resume cooldown was bypassed: %v", err)
	}

	boundaryCtx := firstCtx.WithBlockHeight(firstCtx.BlockHeight() + 5)
	next, err := server.ProposeResume(boundaryCtx, recovery)
	if err != nil {
		t.Fatalf(
			"another guardian could not recover at cooldown boundary: %v",
			err,
		)
	}
	if next.ProposalId == first.ProposalId {
		t.Fatal("honest recovery reused the failed resume ceremony id")
	}
	if got := k.GetGuardianProposalCount(boundaryCtx, attacker); got != 1 {
		t.Fatalf("attacker proposal count=%d, want 1", got)
	}
	if got := k.GetGuardianProposalCount(boundaryCtx, honest); got != 1 {
		t.Fatalf("honest guardian proposal count=%d, want 1", got)
	}
	if got := k.GetEpochProposalCount(boundaryCtx); got != 2 {
		t.Fatalf("shared epoch proposal count=%d, want 2", got)
	}
}

func TestResumeRequiresCommittedHaltFinalizationBlock(t *testing.T) {
	k, mock, ctx := setupKeeper(t)
	guardian := testCouncilAddr(0x53)
	mock.addGuardian(guardian, "100")
	k.SetEmergencyStatus(ctx, types.StatusHalted)
	k.SetActiveHaltCeremonyId(ctx, "halt-commit-boundary")
	k.SetHaltStartBlock(ctx, uint64(ctx.BlockHeight()))

	msgServer := keeper.NewMsgServerImpl(k)
	if _, err := msgServer.ProposeResume(ctx, testResumeMsg(guardian)); err == nil ||
		!types.ErrCooldownActive.Is(err) {
		t.Fatalf("same-block resume proposal must fail: %v", err)
	}
	if _, found := k.GetActiveCeremony(ctx); found {
		t.Fatal("same-block resume rejection created an active ceremony")
	}
	if status := k.GetEmergencyStatus(ctx); status != types.StatusHalted {
		t.Fatalf("same-block rejection changed emergency status to %s", status)
	}

	nextBlock := ctx.WithBlockHeight(ctx.BlockHeight() + 1)
	if _, err := msgServer.ProposeResume(
		nextBlock,
		testResumeMsg(guardian),
	); err != nil {
		t.Fatalf("resume after halt-finalization commit failed: %v", err)
	}
}

func TestResumeAttemptIndexRebuildsAcrossGenesisImport(t *testing.T) {
	k, mock, ctx := setupKeeper(t)
	guardian := testCouncilAddr(0x54)
	voter := testCouncilAddr(0x55)
	mock.addGuardian(guardian, "50")
	mock.addGuardian(voter, "50")
	k.SetEmergencyStatus(ctx, types.StatusHalted)
	k.SetActiveHaltCeremonyId(ctx, legacyQuarantineIDForTest)
	k.SetHaltStartBlock(ctx, 90)

	msgServer := keeper.NewMsgServerImpl(k)
	first, err := msgServer.ProposeResume(ctx, testResumeMsg(guardian))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := msgServer.VoteResume(ctx, &types.MsgVoteResume{
		Voter: voter, ProposalId: first.ProposalId, Approve: false,
	}); err != nil {
		t.Fatal(err)
	}
	exported := k.ExportGenesis(ctx)
	if err := exported.Validate(); err != nil {
		t.Fatalf("resume history is not exportable: %v", err)
	}

	k2, mock2, ctx2 := setupKeeper(t)
	mock2.addGuardian(guardian, "50")
	mock2.addGuardian(voter, "50")
	k2.InitGenesis(ctx2, exported)
	msgServer2 := keeper.NewMsgServerImpl(k2)
	if _, err := msgServer2.ProposeResume(
		ctx2,
		testResumeMsg(guardian),
	); err == nil || !types.ErrProposalLimitExceeded.Is(err) {
		t.Fatalf("genesis import lost same-evidence replay protection: %v", err)
	}

	changedEvidence := testResumeMsg(guardian)
	changedEvidence.RecoveryManifestSha256 =
		"dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"
	second, err := msgServer2.ProposeResume(
		ctx2.WithBlockHeight(ctx2.BlockHeight()+1),
		changedEvidence,
	)
	if err != nil {
		t.Fatalf("genesis import did not preserve retry liveness: %v", err)
	}
	if !strings.Contains(second.ProposalId, "-g2-") {
		t.Fatalf("imported retry generation = %q, want generation 2", second.ProposalId)
	}
}

func TestOperationsSafetyMigrationRebuildsAndValidatesResumeAttemptIndex(
	t *testing.T,
) {
	t.Run("rebuild legacy history", func(t *testing.T) {
		k, mock, ctx := setupKeeper(t)
		guardian := testCouncilAddr(0x56)
		mock.addGuardian(guardian, "100")
		for index, digest := range []string{
			testRecoveryManifestSHA256,
			"eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
		} {
			proposal := &types.EmergencyResumeProposal{
				Id: fmt.Sprintf(
					"legacy-resume-generation-%d",
					index+1,
				),
				Proposer:               guardian,
				HaltCeremonyId:         legacyQuarantineIDForTest,
				Justification:          "legacy evidence-bound recovery",
				RecoveryManifestSha256: digest,
			}
			ceremony, err := k.CreateResumeCeremony(
				ctx.WithBlockHeight(ctx.BlockHeight()+int64(index)),
				proposal,
			)
			if err != nil {
				t.Fatal(err)
			}
			ceremony.Phase = string(types.PhaseFailed)
			ceremony.FailureReason = "legacy failed attempt"
			if err := k.SetCeremony(ctx, ceremony); err != nil {
				t.Fatal(err)
			}
		}
		if _, err := k.MigrateOperationsSafetyV1(ctx); err != nil {
			t.Fatal(err)
		}
		k.SetEmergencyStatus(ctx, types.StatusHalted)
		k.SetActiveHaltCeremonyId(ctx, legacyQuarantineIDForTest)
		k.SetHaltStartBlock(ctx, 90)

		latestEvidence := testResumeMsg(guardian)
		latestEvidence.RecoveryManifestSha256 =
			"eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"
		if _, err := keeper.NewMsgServerImpl(k).ProposeResume(
			ctx.WithBlockHeight(ctx.BlockHeight()+2),
			latestEvidence,
		); err == nil || !types.ErrProposalLimitExceeded.Is(err) {
			t.Fatalf("migration did not rebuild latest evidence cursor: %v", err)
		}

		nextEvidence := testResumeMsg(guardian)
		nextEvidence.RecoveryManifestSha256 =
			"ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"
		third, err := keeper.NewMsgServerImpl(k).ProposeResume(
			ctx.WithBlockHeight(ctx.BlockHeight()+2),
			nextEvidence,
		)
		if err != nil {
			t.Fatalf("migration did not preserve retry liveness: %v", err)
		}
		if !strings.Contains(third.ProposalId, "-g3-") {
			t.Fatalf("rebuilt retry generation = %q, want generation 3", third.ProposalId)
		}
	})

	t.Run("reject corrupt persisted cursor", func(t *testing.T) {
		k, mock, ctx := setupKeeper(t)
		guardian := testCouncilAddr(0x57)
		mock.addGuardian(guardian, "100")
		k.SetEmergencyStatus(ctx, types.StatusHalted)
		k.SetActiveHaltCeremonyId(ctx, "halt-index-validation")
		k.SetHaltStartBlock(ctx, 90)
		if _, err := keeper.NewMsgServerImpl(k).ProposeResume(
			ctx,
			testResumeMsg(guardian),
		); err != nil {
			t.Fatal(err)
		}
		snapshot, err := k.ReadOperationsSafetySnapshot(ctx)
		if err != nil {
			t.Fatal(err)
		}
		foundIndex := false
		for index := range snapshot.Records {
			if bytes.HasPrefix(
				snapshot.Records[index].Key,
				types.ResumeAttemptKeyPrefix,
			) {
				snapshot.Records[index].Value[0] = 0xff
				foundIndex = true
			}
		}
		if !foundIndex {
			t.Fatal("operations-safety snapshot omitted resume attempt index")
		}
		if _, err := k.MigrateOperationsSafetyV1FromSnapshot(
			ctx,
			snapshot,
		); err == nil || !strings.Contains(err.Error(), "resume attempt index") {
			t.Fatalf("corrupt resume attempt index was accepted: %v", err)
		}
	})
}

func TestCeremonyCreateRejectsExistingID(t *testing.T) {
	k, mock, ctx := setupKeeper(t)
	guardian := testCouncilAddr(0x61)
	mock.addGuardian(guardian, "100")
	proposal := &types.EmergencyHaltProposal{
		Id:       "immutable-ceremony-id",
		Proposer: guardian,
		Reason:   "first record must remain immutable",
	}
	if _, err := k.CreateHaltCeremony(ctx, proposal); err != nil {
		t.Fatalf("first ceremony create failed: %v", err)
	}
	if _, err := k.CreateHaltCeremony(ctx, proposal); err == nil {
		t.Fatal("duplicate ceremony id should be rejected")
	}
}

func TestQuarantineEscalationPersistsAcrossResumeVoting(t *testing.T) {
	k, _, ctx := setupKeeper(t)
	params := k.GetParams(ctx)
	params.MaxHaltDurationBlocks = 10
	k.SetParams(ctx, params)
	k.SetEmergencyStatus(ctx, types.StatusResumeVoting)
	k.SetActiveHaltCeremonyId(ctx, "halt-incident-b")
	k.SetHaltStartBlock(ctx, 100)

	k.CheckHaltExpiry(ctx.WithBlockHeight(110))
	if got := k.GetLastHaltEscalationBlock(ctx); got != 110 {
		t.Fatalf("first escalation marker = %d, want 110", got)
	}
	k.CheckHaltExpiry(ctx.WithBlockHeight(111))
	if got := k.GetLastHaltEscalationBlock(ctx); got != 110 {
		t.Fatalf("idempotent escalation marker = %d, want 110", got)
	}
	k.CheckHaltExpiry(ctx.WithBlockHeight(120))
	if got := k.GetLastHaltEscalationBlock(ctx); got != 120 {
		t.Fatalf("second escalation marker = %d, want 120", got)
	}
}

func TestNegativeHaltPrecommitCannotFinalize(t *testing.T) {
	k, mock, ctx := setupKeeper(t)
	g1 := testCouncilAddr(0x71)
	g2 := testCouncilAddr(0x72)
	mock.addGuardian(g1, "50")
	mock.addGuardian(g2, "50")
	params := k.GetParams(ctx)
	params.HaltQuorum = 500_000
	k.SetParams(ctx, params)

	msgServer := keeper.NewMsgServerImpl(k)
	proposal, err := msgServer.ProposeHalt(ctx, &types.MsgProposeHalt{
		Proposer: g1,
		Reason:   "negative precommit regression",
	})
	if err != nil {
		t.Fatalf("propose halt: %v", err)
	}
	if _, err := msgServer.VoteHalt(ctx, &types.MsgVoteHalt{
		Voter: g1, ProposalId: proposal.ProposalId, Approve: true,
	}); err != nil {
		t.Fatalf("halt prevote: %v", err)
	}
	if _, err := msgServer.VoteHalt(ctx, &types.MsgVoteHalt{
		Voter: g1, ProposalId: proposal.ProposalId, Approve: false,
	}); err == nil {
		t.Fatal("negative halt precommit should fail")
	}
	ceremony, _ := k.GetCeremony(ctx, proposal.ProposalId)
	if len(ceremony.Precommits) != 0 || ceremony.Phase != string(types.PhasePrecommit) {
		t.Fatalf("negative halt precommit mutated ceremony: %+v", ceremony)
	}
	if status := k.GetEmergencyStatus(ctx); status != types.StatusHaltVoting {
		t.Fatalf("negative halt precommit changed status to %s", status)
	}
}

func TestNegativeResumePrecommitCannotReopenQuarantine(t *testing.T) {
	k, mock, ctx := setupKeeper(t)
	g1 := testCouncilAddr(0x73)
	g2 := testCouncilAddr(0x74)
	mock.addGuardian(g1, "50")
	mock.addGuardian(g2, "50")
	params := k.GetParams(ctx)
	params.ResumeQuorum = 500_000
	k.SetParams(ctx, params)
	k.SetEmergencyStatus(ctx, types.StatusHalted)
	k.SetActiveHaltCeremonyId(ctx, "halt-negative-precommit")
	k.SetHaltStartBlock(ctx, uint64(ctx.BlockHeight()))

	msgServer := keeper.NewMsgServerImpl(k)
	resumeCtx := ctx.WithBlockHeight(ctx.BlockHeight() + 1)
	proposal, err := msgServer.ProposeResume(resumeCtx, testResumeMsg(g1))
	if err != nil {
		t.Fatalf("propose resume: %v", err)
	}
	if _, err := msgServer.VoteResume(resumeCtx, &types.MsgVoteResume{
		Voter: g1, ProposalId: proposal.ProposalId, Approve: true,
	}); err != nil {
		t.Fatalf("resume prevote: %v", err)
	}
	if _, err := msgServer.VoteResume(resumeCtx, &types.MsgVoteResume{
		Voter: g1, ProposalId: proposal.ProposalId, Approve: false,
	}); err == nil {
		t.Fatal("negative resume precommit should fail")
	}
	ceremony, _ := k.GetCeremony(ctx, proposal.ProposalId)
	if len(ceremony.Precommits) != 0 || ceremony.Phase != string(types.PhasePrecommit) {
		t.Fatalf("negative resume precommit mutated ceremony: %+v", ceremony)
	}
	if status := k.GetEmergencyStatus(ctx); status != types.StatusResumeVoting {
		t.Fatalf("negative resume precommit changed status to %s", status)
	}
}

// TestCeremonyActiveBlocksNewProposal verifies active ceremony prevents new proposals.
func TestCeremonyActiveBlocksNewProposal(t *testing.T) {
	k, mock, ctx := setupKeeper(t)

	g1 := testCouncilAddr(1)
	g2 := testCouncilAddr(2)
	mock.addGuardian(g1, "100000000000")
	mock.addGuardian(g2, "100000000000")

	params := k.GetParams(ctx)
	params.MaxProposalsPerGuardianPerEpoch = 100
	params.MaxProposalsPerEpoch = 100
	k.SetParams(ctx, params)

	msgSvr := keeper.NewMsgServerImpl(k)

	// First proposal creates an active ceremony
	_, err := msgSvr.ProposeHalt(ctx, &types.MsgProposeHalt{
		Proposer: g1,
		Reason:   "first proposal",
	})
	if err != nil {
		t.Fatalf("first proposal should succeed: %v", err)
	}

	// Second proposal should fail because ceremony is active
	_, err = msgSvr.ProposeHalt(ctx, &types.MsgProposeHalt{
		Proposer: g2,
		Reason:   "second proposal",
	})
	if err == nil {
		t.Fatal("expected error when ceremony is already active")
	}
}

// --- Voting Mechanics Tests ---

// TestPrecommitInPrevotePhaseRejected verifies that submitting a precommit during prevote phase
// is handled correctly (vote is treated as a prevote since ceremony is in prevote phase).
func TestPrecommitInPrevotePhaseRejected(t *testing.T) {
	k, mock, ctx := setupKeeper(t)

	g1 := testCouncilAddr(1)
	g2 := testCouncilAddr(2)
	g3 := testCouncilAddr(3)
	mock.addGuardian(g1, "100000000000")
	mock.addGuardian(g2, "100000000000")
	mock.addGuardian(g3, "100000000000")

	params := k.GetParams(ctx)
	params.HaltQuorum = 900000 // 90% — need all 3 to advance to precommit
	k.SetParams(ctx, params)

	msgSvr := keeper.NewMsgServerImpl(k)

	resp, _ := msgSvr.ProposeHalt(ctx, &types.MsgProposeHalt{
		Proposer: g1,
		Reason:   "prevote phase test",
	})

	// g1 prevotes — still in prevote phase (33% < 90%)
	_, err := msgSvr.VoteHalt(ctx, &types.MsgVoteHalt{
		Voter: g1, ProposalId: resp.ProposalId, Approve: true,
	})
	if err != nil {
		t.Fatalf("g1 prevote should succeed: %v", err)
	}

	// Verify still in prevote phase
	ceremony, _ := k.GetCeremony(ctx, resp.ProposalId)
	if ceremony.Phase != string(types.PhasePrevote) {
		t.Fatalf("expected prevote phase, got %s", ceremony.Phase)
	}

	// g2 votes — this should be recorded as a prevote (phase is still prevote)
	_, err = msgSvr.VoteHalt(ctx, &types.MsgVoteHalt{
		Voter: g2, ProposalId: resp.ProposalId, Approve: true,
	})
	if err != nil {
		t.Fatalf("g2 prevote should succeed: %v", err)
	}

	// Verify g2's vote was recorded as a prevote
	ceremony, _ = k.GetCeremony(ctx, resp.ProposalId)
	_, hasPrevote := ceremony.GetPrevote(g2)
	if !hasPrevote {
		t.Fatal("expected g2's vote to be recorded as a prevote")
	}
}

// TestStakeWeightedVoting verifies that stake weights affect quorum calculations.
func TestStakeWeightedVoting(t *testing.T) {
	k, mock, ctx := setupKeeper(t)

	// g1 has 90%, g2 has 10%
	g1 := testCouncilAddr(1)
	g2 := testCouncilAddr(2)
	mock.addGuardian(g1, "900000000000")
	mock.addGuardian(g2, "100000000000")

	params := k.GetParams(ctx)
	params.HaltQuorum = 750000 // 75%
	k.SetParams(ctx, params)

	msgSvr := keeper.NewMsgServerImpl(k)

	resp, _ := msgSvr.ProposeHalt(ctx, &types.MsgProposeHalt{
		Proposer: g1,
		Reason:   "stake weight test",
	})

	// g2 alone votes (10% < 75%) — should NOT advance to precommit
	_, err := msgSvr.VoteHalt(ctx, &types.MsgVoteHalt{
		Voter: g2, ProposalId: resp.ProposalId, Approve: true,
	})
	if err != nil {
		t.Fatalf("g2 prevote should succeed: %v", err)
	}
	ceremony, _ := k.GetCeremony(ctx, resp.ProposalId)
	if ceremony.Phase != string(types.PhasePrevote) {
		t.Fatalf("expected prevote (10%% < 75%%), got %s", ceremony.Phase)
	}

	// g1 votes (90% > 75%) — should advance to precommit
	_, err = msgSvr.VoteHalt(ctx, &types.MsgVoteHalt{
		Voter: g1, ProposalId: resp.ProposalId, Approve: true,
	})
	if err != nil {
		t.Fatalf("g1 prevote should succeed: %v", err)
	}
	ceremony, _ = k.GetCeremony(ctx, resp.ProposalId)
	if ceremony.Phase != string(types.PhasePrecommit) {
		t.Fatalf("expected precommit after 100%% stake voted, got %s", ceremony.Phase)
	}
}

// TestNonGuardianVoteRejected verifies a non-guardian vote attempt is rejected.
func TestNonGuardianVoteRejected(t *testing.T) {
	k, mock, ctx := setupKeeper(t)

	g1 := testCouncilAddr(1)
	scholar := "zrn1scholar"
	mock.addGuardian(g1, "100000000000")
	mock.addNonGuardian(scholar, "50000000000")

	msgSvr := keeper.NewMsgServerImpl(k)

	resp, _ := msgSvr.ProposeHalt(ctx, &types.MsgProposeHalt{
		Proposer: g1,
		Reason:   "non-guardian vote test",
	})

	// Non-guardian tries to vote
	_, err := msgSvr.VoteHalt(ctx, &types.MsgVoteHalt{
		Voter:      scholar,
		ProposalId: resp.ProposalId,
		Approve:    true,
	})
	if err == nil {
		t.Fatal("expected error for non-guardian vote attempt")
	}
}

// --- State Tests ---

// TestIsHaltedStates verifies IsHalted returns correct values for each status.
func TestIsHaltedStates(t *testing.T) {
	k, _, ctx := setupKeeper(t)

	tests := []struct {
		status   types.EmergencyStatus
		expected bool
	}{
		{types.StatusNormal, false},
		{types.StatusHaltVoting, false},
		{types.StatusHalted, true},
		{types.StatusRevertVoting, true},
		{types.StatusReverting, true},
		{types.StatusResumeVoting, true},
	}

	for _, tt := range tests {
		k.SetEmergencyStatus(ctx, tt.status)
		got := k.IsHalted(ctx)
		if got != tt.expected {
			t.Errorf("IsHalted(%s) = %v, want %v", tt.status, got, tt.expected)
		}
	}
}

// TestAuditLogRecordsActions verifies audit entries are recorded for actions.
func TestAuditLogRecordsActions(t *testing.T) {
	k, mock, ctx := setupKeeper(t)

	g1 := testCouncilAddr(1)
	mock.addGuardian(g1, "100000000000")

	msgSvr := keeper.NewMsgServerImpl(k)

	// Propose halt — should generate audit entry
	resp, err := msgSvr.ProposeHalt(ctx, &types.MsgProposeHalt{
		Proposer: g1,
		Reason:   "audit test",
	})
	if err != nil {
		t.Fatalf("ProposeHalt failed: %v", err)
	}

	entries := k.GetAuditLog(ctx)
	if len(entries) == 0 {
		t.Fatal("expected audit log entries after proposal")
	}

	// Check halt_proposed audit entry
	foundProposal := false
	for _, e := range entries {
		if e.Action == string(types.AuditHaltProposed) && e.Actor == g1 {
			foundProposal = true
			break
		}
	}
	if !foundProposal {
		t.Fatal("expected halt_proposed audit entry from g1")
	}

	// Vote — should generate another audit entry
	_, err = msgSvr.VoteHalt(ctx, &types.MsgVoteHalt{
		Voter: g1, ProposalId: resp.ProposalId, Approve: true,
	})
	if err != nil {
		t.Fatalf("VoteHalt failed: %v", err)
	}

	entries = k.GetAuditLog(ctx)
	foundVote := false
	for _, e := range entries {
		if e.Action == string(types.AuditHaltPrevote) && e.Actor == g1 {
			foundVote = true
			break
		}
	}
	if !foundVote {
		t.Fatal("expected halt_prevote audit entry from g1")
	}
}

// --- Query Server Tests ---

// TestQueryActiveCeremony_None verifies query returns no active ceremony when none exists.
func TestQueryActiveCeremony_None(t *testing.T) {
	k, _, ctx := setupKeeper(t)

	querySvr := keeper.NewQueryServerImpl(k)

	resp, err := querySvr.ActiveCeremony(ctx, &types.QueryActiveCeremonyRequest{})
	if err != nil {
		t.Fatalf("query ActiveCeremony failed: %v", err)
	}
	if resp.Found {
		t.Error("expected no active ceremony when none exists")
	}
}

// TestQueryActiveCeremony_Found verifies query finds an active ceremony.
func TestQueryActiveCeremony_Found(t *testing.T) {
	k, mock, ctx := setupKeeper(t)

	g1 := testCouncilAddr(1)
	mock.addGuardian(g1, "100000000000")

	msgSvr := keeper.NewMsgServerImpl(k)
	querySvr := keeper.NewQueryServerImpl(k)

	_, err := msgSvr.ProposeHalt(ctx, &types.MsgProposeHalt{
		Proposer: g1,
		Reason:   "query test",
	})
	if err != nil {
		t.Fatalf("ProposeHalt failed: %v", err)
	}

	resp, err := querySvr.ActiveCeremony(ctx, &types.QueryActiveCeremonyRequest{})
	if err != nil {
		t.Fatalf("query ActiveCeremony failed: %v", err)
	}
	if !resp.Found {
		t.Error("expected active ceremony to be found")
	}
	if resp.Ceremony == nil {
		t.Error("expected non-nil ceremony in response")
	}
}

// TestQueryCompletedCeremonies verifies querying completed ceremonies.
func TestQueryCompletedCeremonies(t *testing.T) {
	k, mock, ctx := setupKeeper(t)

	g1 := testCouncilAddr(1)
	mock.addGuardian(g1, "100000000000")

	params := k.GetParams(ctx)
	params.HaltQuorum = 500000
	k.SetParams(ctx, params)

	msgSvr := keeper.NewMsgServerImpl(k)
	querySvr := keeper.NewQueryServerImpl(k)

	// No completed ceremonies initially
	resp, err := querySvr.CompletedCeremonies(ctx, &types.QueryCompletedCeremoniesRequest{})
	if err != nil {
		t.Fatalf("query CompletedCeremonies failed: %v", err)
	}
	if resp.Total != 0 {
		t.Fatalf("expected 0 completed ceremonies, got %d", resp.Total)
	}

	// Complete a halt ceremony
	haltResp, _ := msgSvr.ProposeHalt(ctx, &types.MsgProposeHalt{Proposer: g1, Reason: "test"})
	msgSvr.VoteHalt(ctx, &types.MsgVoteHalt{Voter: g1, ProposalId: haltResp.ProposalId, Approve: true})
	msgSvr.VoteHalt(ctx, &types.MsgVoteHalt{Voter: g1, ProposalId: haltResp.ProposalId, Approve: true})

	// Now should have 1 completed ceremony
	resp, err = querySvr.CompletedCeremonies(ctx, &types.QueryCompletedCeremoniesRequest{})
	if err != nil {
		t.Fatalf("query CompletedCeremonies failed: %v", err)
	}
	if resp.Total != 1 {
		t.Fatalf("expected 1 completed ceremony, got %d", resp.Total)
	}
}

// TestQueryAuditLog verifies querying audit log entries.
func TestQueryAuditLog(t *testing.T) {
	k, _, ctx := setupKeeper(t)

	querySvr := keeper.NewQueryServerImpl(k)

	// No audit entries initially
	resp, err := querySvr.AuditLog(ctx, &types.QueryAuditLogRequest{})
	if err != nil {
		t.Fatalf("query AuditLog failed: %v", err)
	}
	if resp.Total != 0 {
		t.Fatalf("expected 0 audit entries, got %d", resp.Total)
	}

	// Add an audit entry
	k.AddAuditEntry(ctx, &types.EmergencyAuditEntry{
		BlockNumber: 100,
		Action:      string(types.AuditHaltProposed),
		Actor:       testCouncilAddr(1),
		CeremonyId:  "test-123",
		Details:     "test entry",
	})

	resp, err = querySvr.AuditLog(ctx, &types.QueryAuditLogRequest{})
	if err != nil {
		t.Fatalf("query AuditLog failed: %v", err)
	}
	if resp.Total != 1 {
		t.Fatalf("expected 1 audit entry, got %d", resp.Total)
	}
	if len(resp.Entries) != 1 {
		t.Fatalf("expected 1 entry in response, got %d", len(resp.Entries))
	}
	if resp.Entries[0].Actor != testCouncilAddr(1) {
		t.Fatalf("expected actor zrn1g1, got %s", resp.Entries[0].Actor)
	}
}

// --- Halt Expiry Edge Cases ---

// TestCheckHaltExpiry_NotHalted verifies expiry check when not halted is a no-op.
func TestCheckHaltExpiry_NotHalted(t *testing.T) {
	k, _, ctx := setupKeeper(t)

	// Status is normal by default
	k.CheckHaltExpiry(ctx)
	if k.GetEmergencyStatus(ctx) != types.StatusNormal {
		t.Fatal("status should remain normal after CheckHaltExpiry when not halted")
	}
}

// TestCheckHaltExpiry_NoStartBlock verifies edge case with halted but no start block.
func TestCheckHaltExpiry_NoStartBlock(t *testing.T) {
	k, _, ctx := setupKeeper(t)

	// Halted but no start block recorded (pre-upgrade scenario)
	k.SetEmergencyStatus(ctx, types.StatusHalted)
	k.CheckHaltExpiry(ctx)
	if k.GetEmergencyStatus(ctx) != types.StatusHalted {
		t.Fatal("should not expire without start block (graceful for pre-upgrade halts)")
	}
}

// ============================================================
// Adversarial Tests (ported from openclaw_emergency_test.go)
// ============================================================

// TestOC_NonGuardianHaltAttempt verifies a non-guardian cannot vote on halt.
// Note: TestNonGuardianRejection covers proposal rejection; this tests VOTE rejection.
func TestOC_NonGuardianHaltAttempt(t *testing.T) {
	k, mock, ctx := setupKeeper(t)

	g1 := testCouncilAddr(1)
	unknown := "zrn1attacker"
	mock.addGuardian(g1, "100000000000")

	msgSvr := keeper.NewMsgServerImpl(k)

	// Guardian creates a valid ceremony
	resp, _ := msgSvr.ProposeHalt(ctx, &types.MsgProposeHalt{
		Proposer: g1,
		Reason:   "adversarial test",
	})

	// Completely unknown address tries to vote
	_, err := msgSvr.VoteHalt(ctx, &types.MsgVoteHalt{
		Voter:      unknown,
		ProposalId: resp.ProposalId,
		Approve:    true,
	})
	if err == nil {
		t.Fatal("OC: unknown address voting on halt should fail")
	}

	// Non-guardian (Scholar tier) tries to vote
	mock.addNonGuardian("zrn1scholar", "50000000000")
	_, err = msgSvr.VoteHalt(ctx, &types.MsgVoteHalt{
		Voter:      "zrn1scholar",
		ProposalId: resp.ProposalId,
		Approve:    true,
	})
	if err == nil {
		t.Fatal("OC: non-guardian voting on halt should fail")
	}
}

// TestOC_CooldownBypass verifies cooldown bypass via block height manipulation.
// Note: TestCooldown covers basic cooldown; this tests the exact boundary.
func TestOC_CooldownBypass(t *testing.T) {
	k, mock, ctx := setupKeeper(t)

	g1 := testCouncilAddr(1)
	g2 := testCouncilAddr(2)
	mock.addGuardian(g1, "100000000000")
	mock.addGuardian(g2, "100000000000")

	params := k.GetParams(ctx)
	params.CooldownBlocks = 50
	params.HaltQuorum = 200000 // 20%
	params.MaxProposalsPerGuardianPerEpoch = 100
	params.MaxProposalsPerEpoch = 100
	params.HaltTimeoutBlocks = 5 // Short timeout
	params.HaltPrevoteBlocks = 2
	params.HaltPrecommitBlocks = 2
	k.SetParams(ctx, params)

	msgSvr := keeper.NewMsgServerImpl(k)

	// First proposal at block 100
	resp, _ := msgSvr.ProposeHalt(ctx, &types.MsgProposeHalt{
		Proposer: g1,
		Reason:   "cooldown boundary test",
	})

	// Advance past timeout so ceremony fails, then handle failure
	ctx = ctx.WithBlockHeight(106)
	k.CheckCeremonyProgress(ctx, resp.ProposalId)
	k.HandleCeremonyFailure(ctx, resp.ProposalId)

	// Block 149 — still in cooldown (100 + 50 = 150)
	ctx = ctx.WithBlockHeight(149)
	_, err := msgSvr.ProposeHalt(ctx, &types.MsgProposeHalt{
		Proposer: g2,
		Reason:   "boundary minus one",
	})
	if err == nil {
		t.Fatal("OC: proposal at cooldown boundary-1 (block 149, cooldown ends at 150) should fail")
	}

	// Block 150 — exactly at cooldown end
	ctx = ctx.WithBlockHeight(150)
	_, err = msgSvr.ProposeHalt(ctx, &types.MsgProposeHalt{
		Proposer: g2,
		Reason:   "exactly at boundary",
	})
	if err != nil {
		t.Fatalf("OC: proposal at cooldown boundary (block 150) should succeed: %v", err)
	}
}

// TestOC_ProposalSpamPerEpoch verifies same guardian cannot propose twice in one epoch.
// Note: TestAntiAbusePerGuardianLimit covers basic limit; this tests with epoch boundary.
func TestOC_ProposalSpamPerEpoch(t *testing.T) {
	k, mock, ctx := setupKeeper(t)

	g1 := testCouncilAddr(1)
	g2 := testCouncilAddr(2)
	mock.addGuardian(g1, "100000000000")
	mock.addGuardian(g2, "100000000000")

	params := k.GetParams(ctx)
	params.MaxProposalsPerGuardianPerEpoch = 1
	params.MaxProposalsPerEpoch = 100
	params.HaltQuorum = 200000
	params.CooldownBlocks = 0 // No cooldown for this test.
	params.HaltTimeoutBlocks = 5
	params.HaltPrevoteBlocks = 2
	params.HaltPrecommitBlocks = 2
	k.SetParams(ctx, params)

	msgSvr := keeper.NewMsgServerImpl(k)

	// g1 proposes at block 100
	resp, _ := msgSvr.ProposeHalt(ctx, &types.MsgProposeHalt{
		Proposer: g1,
		Reason:   "first",
	})

	// Advance past timeout, fail the ceremony
	ctx = ctx.WithBlockHeight(106)
	k.CheckCeremonyProgress(ctx, resp.ProposalId)
	k.HandleCeremonyFailure(ctx, resp.ProposalId)

	// g1 tries again — should fail (per-guardian limit)
	_, err := msgSvr.ProposeHalt(ctx, &types.MsgProposeHalt{
		Proposer: g1,
		Reason:   "second by same guardian",
	})
	if err == nil {
		t.Fatal("OC: same guardian proposing twice per epoch should fail")
	}

	// g2 should succeed (different guardian)
	_, err = msgSvr.ProposeHalt(ctx, &types.MsgProposeHalt{
		Proposer: g2,
		Reason:   "different guardian",
	})
	if err != nil {
		t.Fatalf("OC: different guardian should be able to propose: %v", err)
	}
}

// TestOC_PrecommitWithoutPrevote verifies precommit from a guardian who didn't prevote.
// Note: TestPrecommitWithoutPrevoteRejection exists; this tests a resume ceremony variant.
func TestOC_PrecommitWithoutPrevote(t *testing.T) {
	k, mock, ctx := setupKeeper(t)

	g1 := testCouncilAddr(1)
	g2 := testCouncilAddr(2)
	g3 := testCouncilAddr(3)
	mock.addGuardian(g1, "100000000000")
	mock.addGuardian(g2, "100000000000")
	mock.addGuardian(g3, "100000000000")

	params := k.GetParams(ctx)
	params.HaltQuorum = 500000 // 50%
	params.ResumeQuorum = 500000
	k.SetParams(ctx, params)

	msgSvr := keeper.NewMsgServerImpl(k)

	// Execute a halt
	haltResp, _ := msgSvr.ProposeHalt(ctx, &types.MsgProposeHalt{Proposer: g1, Reason: "precommit test"})
	msgSvr.VoteHalt(ctx, &types.MsgVoteHalt{Voter: g1, ProposalId: haltResp.ProposalId, Approve: true})
	msgSvr.VoteHalt(ctx, &types.MsgVoteHalt{Voter: g2, ProposalId: haltResp.ProposalId, Approve: true})
	// Now in precommit phase, finalize halt
	msgSvr.VoteHalt(ctx, &types.MsgVoteHalt{Voter: g1, ProposalId: haltResp.ProposalId, Approve: true})
	msgSvr.VoteHalt(ctx, &types.MsgVoteHalt{Voter: g2, ProposalId: haltResp.ProposalId, Approve: true})

	if k.GetEmergencyStatus(ctx) != types.StatusHalted {
		t.Fatal("expected halted status for precommit test setup")
	}

	// Propose resume
	resumeCtx := ctx.WithBlockHeight(ctx.BlockHeight() + 1)
	resumeResp, err := msgSvr.ProposeResume(resumeCtx, testResumeMsg(g1))
	if err != nil {
		t.Fatalf("propose resume: %v", err)
	}

	// g1 and g2 prevote resume
	msgSvr.VoteResume(resumeCtx, &types.MsgVoteResume{Voter: g1, ProposalId: resumeResp.ProposalId, Approve: true})
	msgSvr.VoteResume(resumeCtx, &types.MsgVoteResume{Voter: g2, ProposalId: resumeResp.ProposalId, Approve: true})

	// Verify ceremony is in precommit
	ceremony, _ := k.GetCeremony(ctx, resumeResp.ProposalId)
	if ceremony.Phase != string(types.PhasePrecommit) {
		t.Fatalf("expected precommit phase, got %s", ceremony.Phase)
	}

	// g3 did NOT prevote — precommit on resume should fail
	_, err = msgSvr.VoteResume(resumeCtx, &types.MsgVoteResume{
		Voter: g3, ProposalId: resumeResp.ProposalId, Approve: true,
	})
	if err == nil {
		t.Fatal("OC: precommit without prevote on resume ceremony should fail")
	}
}

// TestOC_DuplicateVoteInPhase verifies duplicate precommit vote is rejected.
// Note: TestDuplicateVotePrevention covers prevote duplicates; this tests precommit duplicates.
func TestOC_DuplicateVoteInPhase(t *testing.T) {
	k, mock, ctx := setupKeeper(t)

	g1 := testCouncilAddr(1)
	g2 := testCouncilAddr(2)
	mock.addGuardian(g1, "80000000000")
	mock.addGuardian(g2, "20000000000")

	params := k.GetParams(ctx)
	params.HaltQuorum = 500000 // 50%
	k.SetParams(ctx, params)

	msgSvr := keeper.NewMsgServerImpl(k)

	resp, _ := msgSvr.ProposeHalt(ctx, &types.MsgProposeHalt{
		Proposer: g1,
		Reason:   "dup precommit test",
	})

	// g1 prevotes (80% > 50%) — advances to precommit
	msgSvr.VoteHalt(ctx, &types.MsgVoteHalt{Voter: g1, ProposalId: resp.ProposalId, Approve: true})

	ceremony, _ := k.GetCeremony(ctx, resp.ProposalId)
	if ceremony.Phase != string(types.PhasePrecommit) {
		t.Fatalf("expected precommit phase, got %s", ceremony.Phase)
	}

	// g1 first precommit — succeeds
	_, err := msgSvr.VoteHalt(ctx, &types.MsgVoteHalt{
		Voter: g1, ProposalId: resp.ProposalId, Approve: true,
	})
	if err != nil {
		t.Fatalf("first precommit should succeed: %v", err)
	}

	// g1 duplicate precommit — should fail
	_, err = msgSvr.VoteHalt(ctx, &types.MsgVoteHalt{
		Voter: g1, ProposalId: resp.ProposalId, Approve: true,
	})
	if err == nil {
		t.Fatal("OC: duplicate precommit should fail")
	}
}

// TestOC_HaltFromWrongStatus verifies halt proposals are rejected from non-normal statuses.
func TestOC_HaltFromWrongStatus(t *testing.T) {
	k, mock, ctx := setupKeeper(t)

	g1 := testCouncilAddr(1)
	mock.addGuardian(g1, "100000000000")

	msgSvr := keeper.NewMsgServerImpl(k)

	invalidStatuses := []types.EmergencyStatus{
		types.StatusHalted,
		types.StatusResumeVoting,
		types.StatusRevertVoting,
		types.StatusReverting,
		types.StatusHaltVoting,
	}

	for _, status := range invalidStatuses {
		k.SetEmergencyStatus(ctx, status)
		_, err := msgSvr.ProposeHalt(ctx, &types.MsgProposeHalt{
			Proposer: g1,
			Reason:   "wrong status: " + string(status),
		})
		if err == nil {
			t.Errorf("OC: halt proposal should fail when status is %s", status)
		}
	}
}

// TestOC_RevertDepthExceeded verifies that even a formerly valid depth cannot
// bypass the fail-closed removal of arbitrary height-only rollback.
func TestOC_RevertDepthExceeded(t *testing.T) {
	k, mock, ctx := setupKeeper(t)

	g1 := testCouncilAddr(1)
	mock.addGuardian(g1, "100000000000")

	params := k.GetParams(ctx)
	params.MaxRevertDepth = 50
	params.HaltQuorum = 500000
	k.SetParams(ctx, params)

	// Halt the chain
	k.SetEmergencyStatus(ctx, types.StatusHalted)

	msgSvr := keeper.NewMsgServerImpl(k)

	// At block 100, revert to block 50 was historically accepted at the exact
	// depth boundary. It is now refused because height does not bind AppHash.
	_, err := msgSvr.ProposeRevert(ctx, &types.MsgProposeRevert{
		Proposer:       g1,
		RevertToHeight: 50,
		Justification:  "exact max depth",
	})
	if err == nil || !types.ErrUnsafeRevertDisabled.Is(err) {
		t.Fatalf("OC: exact-depth height-only revert must be disabled, got: %v", err)
	}
}

// TestOC_RevertDepthExceededByOne verifies revert at max+1 is rejected.
func TestOC_RevertDepthExceededByOne(t *testing.T) {
	k, mock, ctx := setupKeeper(t)

	g1 := testCouncilAddr(1)
	mock.addGuardian(g1, "100000000000")

	params := k.GetParams(ctx)
	params.MaxRevertDepth = 50
	k.SetParams(ctx, params)

	k.SetEmergencyStatus(ctx, types.StatusHalted)

	msgSvr := keeper.NewMsgServerImpl(k)

	// At block 100, revert to block 49 → depth = 51 > MaxRevertDepth (should fail)
	_, err := msgSvr.ProposeRevert(ctx, &types.MsgProposeRevert{
		Proposer:       g1,
		RevertToHeight: 49,
		Justification:  "one past max depth",
	})
	if err == nil {
		t.Fatal("OC: revert at depth max+1 (51 > 50) should fail")
	}
}

// TestOC_ResumeFromNormalState verifies resume when already normal is rejected.
func TestOC_ResumeFromNormalState(t *testing.T) {
	k, mock, ctx := setupKeeper(t)

	g1 := testCouncilAddr(1)
	mock.addGuardian(g1, "100000000000")

	// Status is normal by default
	if k.GetEmergencyStatus(ctx) != types.StatusNormal {
		t.Fatal("expected normal status")
	}

	msgSvr := keeper.NewMsgServerImpl(k)

	_, err := msgSvr.ProposeResume(ctx, testResumeMsg(g1))
	if err == nil {
		t.Fatal("OC: resume from normal state should fail")
	}

	// Also test revert from normal state
	_, err = msgSvr.ProposeRevert(ctx, &types.MsgProposeRevert{
		Proposer:       g1,
		RevertToHeight: 50,
		Justification:  "not halted",
	})
	if err == nil {
		t.Fatal("OC: revert from normal state should fail")
	}
}

// TestOC_QuorumManipulation verifies quorum advances correctly at threshold boundaries.
func TestOC_QuorumManipulation(t *testing.T) {
	k, mock, ctx := setupKeeper(t)

	// 5 guardians with equal stake (20% each)
	guardians := []string{testCouncilAddr(1), testCouncilAddr(2), testCouncilAddr(3), testCouncilAddr(4), testCouncilAddr(5)}
	for _, g := range guardians {
		mock.addGuardian(g, "100000000000")
	}

	params := k.GetParams(ctx)
	params.HaltQuorum = 750000 // 75%
	params.MinDistinctVoters = 1
	k.SetParams(ctx, params)

	msgSvr := keeper.NewMsgServerImpl(k)

	resp, _ := msgSvr.ProposeHalt(ctx, &types.MsgProposeHalt{
		Proposer: guardians[0],
		Reason:   "quorum manipulation test",
	})

	// 2 guardians = 40% < 75% — still prevote
	for _, g := range guardians[:2] {
		msgSvr.VoteHalt(ctx, &types.MsgVoteHalt{Voter: g, ProposalId: resp.ProposalId, Approve: true})
	}
	ceremony, _ := k.GetCeremony(ctx, resp.ProposalId)
	if ceremony.Phase != string(types.PhasePrevote) {
		t.Fatalf("OC: 40%% should still be prevote, got %s", ceremony.Phase)
	}

	// 3 guardians = 60% < 75% — still prevote
	msgSvr.VoteHalt(ctx, &types.MsgVoteHalt{Voter: guardians[2], ProposalId: resp.ProposalId, Approve: true})
	ceremony, _ = k.GetCeremony(ctx, resp.ProposalId)
	if ceremony.Phase != string(types.PhasePrevote) {
		t.Fatalf("OC: 60%% should still be prevote, got %s", ceremony.Phase)
	}

	// 4 guardians = 80% > 75% — advance to precommit
	msgSvr.VoteHalt(ctx, &types.MsgVoteHalt{Voter: guardians[3], ProposalId: resp.ProposalId, Approve: true})
	ceremony, _ = k.GetCeremony(ctx, resp.ProposalId)
	if ceremony.Phase != string(types.PhasePrecommit) {
		t.Fatalf("OC: 80%% should advance to precommit, got %s", ceremony.Phase)
	}

	// 3 precommits = 60% < 75% — not finalized
	for _, g := range guardians[:3] {
		msgSvr.VoteHalt(ctx, &types.MsgVoteHalt{Voter: g, ProposalId: resp.ProposalId, Approve: true})
	}
	ceremony, _ = k.GetCeremony(ctx, resp.ProposalId)
	if ceremony.Phase == string(types.PhaseFinalized) {
		t.Fatal("OC: 60% precommit should not finalize at 75% quorum")
	}

	// 4th precommit → 80% > 75% → finalized
	msgSvr.VoteHalt(ctx, &types.MsgVoteHalt{Voter: guardians[3], ProposalId: resp.ProposalId, Approve: true})

	status := k.GetEmergencyStatus(ctx)
	if status != types.StatusHalted {
		t.Fatalf("OC: expected halted after full quorum, got %s", status)
	}
}

// TestOC_CeremonyTimeoutEnforcement verifies ceremony timeout at exact deadline boundary.
func TestOC_CeremonyTimeoutEnforcement(t *testing.T) {
	k, mock, ctx := setupKeeper(t)

	g1 := testCouncilAddr(1)
	mock.addGuardian(g1, "100000000000")

	params := k.GetParams(ctx)
	params.HaltPrevoteBlocks = 10
	params.HaltPrecommitBlocks = 10
	params.HaltTimeoutBlocks = 20
	k.SetParams(ctx, params)

	msgSvr := keeper.NewMsgServerImpl(k)

	resp, _ := msgSvr.ProposeHalt(ctx, &types.MsgProposeHalt{
		Proposer: g1,
		Reason:   "timeout boundary test",
	})

	// At prevote deadline boundary (100 + 10 = 110): should still be active
	ctxAtDeadline := ctx.WithBlockHeight(110)
	k.CheckCeremonyProgress(ctxAtDeadline, resp.ProposalId)
	ceremony, _ := k.GetCeremony(ctxAtDeadline, resp.ProposalId)
	if ceremony.Phase == string(types.PhaseFailed) {
		t.Error("OC: ceremony should not timeout exactly at prevote deadline")
	}

	// Just past prevote deadline (111): should fail (prevote quorum not reached)
	ctxPastDeadline := ctx.WithBlockHeight(111)
	k.CheckCeremonyProgress(ctxPastDeadline, resp.ProposalId)
	ceremony, _ = k.GetCeremony(ctxPastDeadline, resp.ProposalId)
	if ceremony.Phase != string(types.PhaseFailed) {
		t.Fatalf("OC: ceremony should fail after prevote deadline, got %s", ceremony.Phase)
	}
	if ceremony.FailureReason == "" {
		t.Fatal("OC: expected failure reason to be set")
	}
}

// TestOC_VoteFlipRejected verifies that changing vote (YES→NO) in prevote phase is rejected.
func TestOC_VoteFlipRejected(t *testing.T) {
	k, mock, ctx := setupKeeper(t)

	g1 := testCouncilAddr(1)
	g2 := testCouncilAddr(2)
	mock.addGuardian(g1, "100000000000")
	mock.addGuardian(g2, "100000000000")

	params := k.GetParams(ctx)
	params.HaltQuorum = 900000 // High quorum so g1 alone can't advance
	k.SetParams(ctx, params)

	msgSvr := keeper.NewMsgServerImpl(k)

	resp, _ := msgSvr.ProposeHalt(ctx, &types.MsgProposeHalt{
		Proposer: g1,
		Reason:   "vote flip test",
	})

	// g2 prevotes YES
	_, err := msgSvr.VoteHalt(ctx, &types.MsgVoteHalt{
		Voter: g2, ProposalId: resp.ProposalId, Approve: true,
	})
	if err != nil {
		t.Fatalf("first prevote should succeed: %v", err)
	}

	// g2 tries to prevote NO (flip) — should fail as duplicate
	_, err = msgSvr.VoteHalt(ctx, &types.MsgVoteHalt{
		Voter: g2, ProposalId: resp.ProposalId, Approve: false,
	})
	if err == nil {
		t.Fatal("OC: vote flip (YES→NO) should be rejected as duplicate")
	}
}

// TestOC_NoPrevoteRejection verifies that "no" prevotes contribute to quorum impossibility.
func TestOC_NoPrevoteRejection(t *testing.T) {
	k, mock, ctx := setupKeeper(t)

	// 4 guardians with equal stake (25% each)
	guardians := []string{testCouncilAddr(1), testCouncilAddr(2), testCouncilAddr(3), testCouncilAddr(4)}
	for _, g := range guardians {
		mock.addGuardian(g, "100000000000")
	}

	params := k.GetParams(ctx)
	params.HaltQuorum = 750000 // 75%
	k.SetParams(ctx, params)

	msgSvr := keeper.NewMsgServerImpl(k)

	resp, _ := msgSvr.ProposeHalt(ctx, &types.MsgProposeHalt{
		Proposer: guardians[0],
		Reason:   "no vote test",
	})

	// g1 votes YES (25%), g2 votes NO (25%)
	msgSvr.VoteHalt(ctx, &types.MsgVoteHalt{Voter: guardians[0], ProposalId: resp.ProposalId, Approve: true})
	msgSvr.VoteHalt(ctx, &types.MsgVoteHalt{Voter: guardians[1], ProposalId: resp.ProposalId, Approve: false})

	// After NO exceeds (100% - 75%) = 25%, quorum should become impossible
	// g2's NO at 25% = 25% which is equal to threshold (not strictly greater), check behavior
	ceremony, _ := k.GetCeremony(ctx, resp.ProposalId)
	if ceremony.Phase == string(types.PhaseFailed) {
		t.Log("ceremony correctly failed: quorum impossible with 25% NO votes")
	}

	// g3 also votes NO (now 50% NO > 25% allowed)
	if ceremony.Phase != string(types.PhaseFailed) {
		msgSvr.VoteHalt(ctx, &types.MsgVoteHalt{Voter: guardians[2], ProposalId: resp.ProposalId, Approve: false})
		ceremony, _ = k.GetCeremony(ctx, resp.ProposalId)
		if ceremony.Phase != string(types.PhaseFailed) {
			t.Fatal("OC: ceremony should fail when quorum is impossible (50% NO at 75% threshold)")
		}
	}
}

// TestBeginBlockCeremonyTimeout verifies BeginBlock detects ceremony timeout.
func TestBeginBlockCeremonyTimeout(t *testing.T) {
	k, mock, ctx := setupKeeper(t)

	g1 := testCouncilAddr(1)
	mock.addGuardian(g1, "100000000000")

	params := k.GetParams(ctx)
	params.HaltTimeoutBlocks = 10
	params.HaltPrevoteBlocks = 5
	params.HaltPrecommitBlocks = 5
	k.SetParams(ctx, params)

	msgSvr := keeper.NewMsgServerImpl(k)

	resp, _ := msgSvr.ProposeHalt(ctx, &types.MsgProposeHalt{
		Proposer: g1,
		Reason:   "begin block timeout test",
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
		t.Fatalf("expected failed phase after timeout via BeginBlock, got %s", ceremony.Phase)
	}

	// Status should revert to normal after halt ceremony failure
	status := k.GetEmergencyStatus(ctx)
	if status != types.StatusNormal {
		t.Fatalf("expected normal after halt failure via BeginBlock, got %s", status)
	}
}
