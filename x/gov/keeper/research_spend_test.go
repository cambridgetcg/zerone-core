package keeper_test

import (
	"fmt"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/x/gov/types"
)

// ---------- Mock VestingRewardsKeeper ----------

type mockVestingKeeper struct {
	disburseCalled bool
	disburseErr    error
}

func (m *mockVestingKeeper) DisburseFromResearchFund(_ sdk.Context, _ sdk.AccAddress, _ sdk.Coins) error {
	m.disburseCalled = true
	return m.disburseErr
}

// ---------- Helpers ----------

// setupResearchVoters configures voter1 and voter2 on the keeper and returns their addresses.
func setupResearchVoters(t *testing.T, k interface {
	SetResearchFundVoters(ctx sdk.Context, voters *types.ResearchFundVoters)
}, ctx sdk.Context) (voter1, voter2 string) {
	t.Helper()
	voter1 = testAddr("rfv1")
	voter2 = testAddr("rfv2")
	k.SetResearchFundVoters(ctx, &types.ResearchFundVoters{
		Voter1: voter1,
		Voter2: voter2,
	})
	return voter1, voter2
}

// ---------- Research Spend Tests ----------

func TestResearchSpend_SubmitByVoter1(t *testing.T) {
	k, ctx := setupKeeper(t)
	voter1, _ := setupResearchVoters(t, k, ctx)

	resp, err := k.SubmitResearchSpend(ctx, &types.MsgSubmitResearchSpend{
		Proposer:    voter1,
		Title:       "Fund AI safety research",
		Description: "Grant for adversarial testing framework",
		Recipient:   testAddr("researcher"),
		Amount:      "1000000000",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ProposalId != 1 {
		t.Errorf("expected proposal_id=1, got %d", resp.ProposalId)
	}

	prop, found := k.GetResearchSpendProposal(ctx, 1)
	if !found {
		t.Fatal("proposal not found")
	}
	if prop.Stage != string(types.ResearchStageDiscussion) {
		t.Errorf("expected discussion stage, got %s", prop.Stage)
	}
	if prop.Proposer != voter1 {
		t.Errorf("wrong proposer")
	}
}

func TestResearchSpend_SubmitByVoter2(t *testing.T) {
	k, ctx := setupKeeper(t)
	_, voter2 := setupResearchVoters(t, k, ctx)

	resp, err := k.SubmitResearchSpend(ctx, &types.MsgSubmitResearchSpend{
		Proposer:    voter2,
		Title:       "Fund protocol audit",
		Description: "Security audit grant",
		Recipient:   testAddr("auditor"),
		Amount:      "500000000",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ProposalId != 1 {
		t.Errorf("expected proposal_id=1, got %d", resp.ProposalId)
	}

	prop, found := k.GetResearchSpendProposal(ctx, 1)
	if !found {
		t.Fatal("proposal not found")
	}
	if prop.Proposer != voter2 {
		t.Errorf("wrong proposer: got %s, want %s", prop.Proposer, voter2)
	}
}

func TestResearchSpend_SubmitByNonVoter(t *testing.T) {
	k, ctx := setupKeeper(t)
	setupResearchVoters(t, k, ctx)

	_, err := k.SubmitResearchSpend(ctx, &types.MsgSubmitResearchSpend{
		Proposer:    testAddr("outsider"),
		Title:       "Sneaky proposal",
		Description: "Should fail",
		Recipient:   testAddr("me"),
		Amount:      "999999999",
	})
	if err == nil {
		t.Error("expected error for non-voter submit")
	}
	if err != types.ErrNotDesignatedVoter {
		t.Errorf("expected ErrNotDesignatedVoter, got %v", err)
	}
}

func TestResearchSpend_BothYes(t *testing.T) {
	k, ctx := setupKeeper(t)
	voter1, voter2 := setupResearchVoters(t, k, ctx)

	// Wire mock vesting keeper.
	mock := &mockVestingKeeper{}
	k.SetVestingKeeper(mock)

	// Submit proposal.
	k.SubmitResearchSpend(ctx, &types.MsgSubmitResearchSpend{
		Proposer:  voter1,
		Title:     "Both yes test",
		Recipient: testAddr("recipient"),
		Amount:    "100000000",
	})

	// Advance to voting by moving past discussion period.
	prop, _ := k.GetResearchSpendProposal(ctx, 1)
	prop.Stage = string(types.ResearchStageVoting)
	k.SetResearchSpendProposal(ctx, prop)

	// Voter 1 votes yes.
	_, err := k.VoteResearchSpend(ctx, &types.MsgVoteResearchSpend{
		Voter: voter1, ProposalId: 1, Vote: "yes", Reasoning: "Great research",
	})
	if err != nil {
		t.Fatalf("voter1 vote failed: %v", err)
	}

	// Voter 2 votes yes.
	_, err = k.VoteResearchSpend(ctx, &types.MsgVoteResearchSpend{
		Voter: voter2, ProposalId: 1, Vote: "yes", Reasoning: "I concur",
	})
	if err != nil {
		t.Fatalf("voter2 vote failed: %v", err)
	}

	prop, _ = k.GetResearchSpendProposal(ctx, 1)
	if prop.Stage != string(types.ResearchStageExecuted) {
		t.Errorf("expected executed, got %s", prop.Stage)
	}
	if !mock.disburseCalled {
		t.Error("expected DisburseFromResearchFund to be called")
	}
}

func TestResearchSpend_OneNo(t *testing.T) {
	k, ctx := setupKeeper(t)
	voter1, voter2 := setupResearchVoters(t, k, ctx)

	k.SubmitResearchSpend(ctx, &types.MsgSubmitResearchSpend{
		Proposer:  voter1,
		Title:     "One no test",
		Recipient: testAddr("recipient"),
		Amount:    "100000000",
	})

	// Advance to voting.
	prop, _ := k.GetResearchSpendProposal(ctx, 1)
	prop.Stage = string(types.ResearchStageVoting)
	k.SetResearchSpendProposal(ctx, prop)

	// Voter 1 votes yes.
	k.VoteResearchSpend(ctx, &types.MsgVoteResearchSpend{
		Voter: voter1, ProposalId: 1, Vote: "yes",
	})

	// Voter 2 votes no.
	k.VoteResearchSpend(ctx, &types.MsgVoteResearchSpend{
		Voter: voter2, ProposalId: 1, Vote: "no", Reasoning: "Not convinced",
	})

	prop, _ = k.GetResearchSpendProposal(ctx, 1)
	if prop.Stage != string(types.ResearchStageRejected) {
		t.Errorf("expected rejected, got %s", prop.Stage)
	}
}

func TestResearchSpend_Timeout(t *testing.T) {
	k, ctx := setupKeeper(t)
	voter1, _ := setupResearchVoters(t, k, ctx)

	// Override params for short periods.
	params := k.GetParams(ctx)
	params.ResearchDiscussionBlocks = 5
	params.ResearchVotingBlocks = 10
	k.SetParams(ctx, params)

	k.SubmitResearchSpend(ctx, &types.MsgSubmitResearchSpend{
		Proposer:  voter1,
		Title:     "Timeout test",
		Recipient: testAddr("recipient"),
		Amount:    "100000000",
	})

	// Discussion period ends at height 105, voting ends at 115.
	// Advance past discussion → voting via BeginBlocker.
	ctx2 := ctx.WithBlockHeight(106)
	k.BeginBlocker(ctx2)

	prop, _ := k.GetResearchSpendProposal(ctx2, 1)
	if prop.Stage != string(types.ResearchStageVoting) {
		t.Errorf("expected voting after discussion period, got %s", prop.Stage)
	}

	// Advance past voting → expired.
	ctx3 := ctx.WithBlockHeight(116)
	k.BeginBlocker(ctx3)

	prop, _ = k.GetResearchSpendProposal(ctx3, 1)
	if prop.Stage != string(types.ResearchStageExpired) {
		t.Errorf("expected expired after voting period, got %s", prop.Stage)
	}
}

func TestResearchSpend_DoubleVote(t *testing.T) {
	k, ctx := setupKeeper(t)
	voter1, _ := setupResearchVoters(t, k, ctx)

	k.SubmitResearchSpend(ctx, &types.MsgSubmitResearchSpend{
		Proposer:  voter1,
		Title:     "Double vote test",
		Recipient: testAddr("recipient"),
		Amount:    "100000000",
	})

	// Advance to voting.
	prop, _ := k.GetResearchSpendProposal(ctx, 1)
	prop.Stage = string(types.ResearchStageVoting)
	k.SetResearchSpendProposal(ctx, prop)

	// First vote.
	k.VoteResearchSpend(ctx, &types.MsgVoteResearchSpend{
		Voter: voter1, ProposalId: 1, Vote: "yes",
	})

	// Second vote (same voter) should fail.
	_, err := k.VoteResearchSpend(ctx, &types.MsgVoteResearchSpend{
		Voter: voter1, ProposalId: 1, Vote: "no",
	})
	if err == nil {
		t.Error("expected error for double vote")
	}
	if err != types.ErrResearchAlreadyVoted {
		t.Errorf("expected ErrResearchAlreadyVoted, got %v", err)
	}
}

func TestResearchSpend_NonVoterVote(t *testing.T) {
	k, ctx := setupKeeper(t)
	voter1, _ := setupResearchVoters(t, k, ctx)

	k.SubmitResearchSpend(ctx, &types.MsgSubmitResearchSpend{
		Proposer:  voter1,
		Title:     "Non-voter vote test",
		Recipient: testAddr("recipient"),
		Amount:    "100000000",
	})

	// Advance to voting.
	prop, _ := k.GetResearchSpendProposal(ctx, 1)
	prop.Stage = string(types.ResearchStageVoting)
	k.SetResearchSpendProposal(ctx, prop)

	_, err := k.VoteResearchSpend(ctx, &types.MsgVoteResearchSpend{
		Voter: testAddr("outsider"), ProposalId: 1, Vote: "yes",
	})
	if err == nil {
		t.Error("expected error for non-voter vote")
	}
	if err != types.ErrNotDesignatedVoter {
		t.Errorf("expected ErrNotDesignatedVoter, got %v", err)
	}
}

func TestResearchSpend_InsufficientFunds(t *testing.T) {
	k, ctx := setupKeeper(t)
	voter1, voter2 := setupResearchVoters(t, k, ctx)

	mock := &mockVestingKeeper{
		disburseErr: fmt.Errorf("insufficient funds in research_fund"),
	}
	k.SetVestingKeeper(mock)

	k.SubmitResearchSpend(ctx, &types.MsgSubmitResearchSpend{
		Proposer:  voter1,
		Title:     "Insufficient funds test",
		Recipient: testAddr("recipient"),
		Amount:    "999999999999999",
	})

	// Advance to voting.
	prop, _ := k.GetResearchSpendProposal(ctx, 1)
	prop.Stage = string(types.ResearchStageVoting)
	k.SetResearchSpendProposal(ctx, prop)

	// Both vote yes — execution should fail.
	k.VoteResearchSpend(ctx, &types.MsgVoteResearchSpend{
		Voter: voter1, ProposalId: 1, Vote: "yes",
	})
	k.VoteResearchSpend(ctx, &types.MsgVoteResearchSpend{
		Voter: voter2, ProposalId: 1, Vote: "yes",
	})

	prop, _ = k.GetResearchSpendProposal(ctx, 1)
	// Should NOT be in executed stage — execution error should be recorded.
	if prop.Stage == string(types.ResearchStageExecuted) {
		t.Error("should not be executed with insufficient funds")
	}
	if prop.ExecutionErr == "" {
		t.Error("expected execution_err to be set")
	}
}

func TestResearchSpend_DiscussionPeriod(t *testing.T) {
	k, ctx := setupKeeper(t)
	voter1, _ := setupResearchVoters(t, k, ctx)

	k.SubmitResearchSpend(ctx, &types.MsgSubmitResearchSpend{
		Proposer:  voter1,
		Title:     "Discussion period test",
		Recipient: testAddr("recipient"),
		Amount:    "100000000",
	})

	// Proposal is in discussion stage. Try to vote (should fail).
	_, err := k.VoteResearchSpend(ctx, &types.MsgVoteResearchSpend{
		Voter: voter1, ProposalId: 1, Vote: "yes",
	})
	if err == nil {
		t.Error("expected error for voting during discussion period")
	}
	if err != types.ErrDiscussionPeriodActive {
		t.Errorf("expected ErrDiscussionPeriodActive, got %v", err)
	}
}

func TestResearchSpend_AuditTrail(t *testing.T) {
	k, ctx := setupKeeper(t)
	voter1, voter2 := setupResearchVoters(t, k, ctx)
	mock := &mockVestingKeeper{}
	k.SetVestingKeeper(mock)

	k.SubmitResearchSpend(ctx, &types.MsgSubmitResearchSpend{
		Proposer:      voter1,
		Title:         "Audit trail test",
		Description:   "Testing vote storage",
		Recipient:     testAddr("recipient"),
		Amount:        "100000000",
		Justification: "Critical research need",
	})

	// Advance to voting.
	prop, _ := k.GetResearchSpendProposal(ctx, 1)
	prop.Stage = string(types.ResearchStageVoting)
	k.SetResearchSpendProposal(ctx, prop)

	// Vote with reasoning.
	k.VoteResearchSpend(ctx, &types.MsgVoteResearchSpend{
		Voter: voter1, ProposalId: 1, Vote: "yes", Reasoning: "Aligns with roadmap",
	})
	k.VoteResearchSpend(ctx, &types.MsgVoteResearchSpend{
		Voter: voter2, ProposalId: 1, Vote: "yes", Reasoning: "Verified budget",
	})

	prop, _ = k.GetResearchSpendProposal(ctx, 1)

	// Check voter 1 audit.
	if prop.Voter1Vote != "yes" {
		t.Errorf("voter1 vote: got %q, want yes", prop.Voter1Vote)
	}
	if prop.Voter1Reason != "Aligns with roadmap" {
		t.Errorf("voter1 reason: got %q", prop.Voter1Reason)
	}
	if prop.Voter1VotedAt != 100 { // block height from setupKeeper
		t.Errorf("voter1 voted_at: got %d, want 100", prop.Voter1VotedAt)
	}

	// Check voter 2 audit.
	if prop.Voter2Vote != "yes" {
		t.Errorf("voter2 vote: got %q, want yes", prop.Voter2Vote)
	}
	if prop.Voter2Reason != "Verified budget" {
		t.Errorf("voter2 reason: got %q", prop.Voter2Reason)
	}
	if prop.Voter2VotedAt != 100 {
		t.Errorf("voter2 voted_at: got %d, want 100", prop.Voter2VotedAt)
	}

	// Check justification preserved.
	if prop.Justification != "Critical research need" {
		t.Errorf("justification: got %q", prop.Justification)
	}
}

func TestResearchSpend_EmptyVoters(t *testing.T) {
	k, ctx := setupKeeper(t)

	// No voters configured — submit should fail.
	_, err := k.SubmitResearchSpend(ctx, &types.MsgSubmitResearchSpend{
		Proposer:  testAddr("anyone"),
		Title:     "No voters",
		Recipient: testAddr("recipient"),
		Amount:    "100000000",
	})
	if err == nil {
		t.Error("expected error when no voters configured")
	}
	if err != types.ErrResearchVotersNotSet {
		t.Errorf("expected ErrResearchVotersNotSet, got %v", err)
	}
}

func TestResearchVoters_GovernanceOnly(t *testing.T) {
	k, ctx := setupKeeper(t)

	// Non-authority should fail.
	_, err := k.SetResearchVoters(ctx, &types.MsgSetResearchVoters{
		Authority: testAddr("random"),
		Voters: &types.ResearchFundVoters{
			Voter1: testAddr("v1"),
			Voter2: testAddr("v2"),
		},
	})
	if err == nil {
		t.Error("expected error for non-authority")
	}
	if err != types.ErrUnauthorized {
		t.Errorf("expected ErrUnauthorized, got %v", err)
	}

	// Authority should succeed.
	_, err = k.SetResearchVoters(ctx, &types.MsgSetResearchVoters{
		Authority: "authority", // matches setupKeeper authority
		Voters: &types.ResearchFundVoters{
			Voter1: testAddr("v1"),
			Voter2: testAddr("v2"),
		},
	})
	if err != nil {
		t.Fatalf("authority call failed: %v", err)
	}

	voters := k.GetResearchFundVoters(ctx)
	if voters == nil {
		t.Fatal("voters not set")
	}
	if voters.Voter1 != testAddr("v1") {
		t.Errorf("voter1: got %s, want %s", voters.Voter1, testAddr("v1"))
	}
	if voters.Voter2 != testAddr("v2") {
		t.Errorf("voter2: got %s, want %s", voters.Voter2, testAddr("v2"))
	}
}
