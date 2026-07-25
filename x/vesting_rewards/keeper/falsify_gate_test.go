package keeper_test

import (
	"testing"

	"github.com/zerone-chain/zerone/x/vesting_rewards/types"
)

// The falsification gate. MsgFalsifyVesting's proto signer is `challenger`,
// not `authority` — any address can reach FalsifyClaim by naming itself — and
// before this gate the handler consulted nothing but the schedule's
// existence. One tx fee permanently zeroed any honest submitter's payout.
//
// These tests pin the GATE, not the symptom: each asserts both that the call
// is refused and that the schedule is left untouched, because a gate that
// returns an error after already mutating state would satisfy a weaker test.

func TestFalsifyClaim_RefusedWhenFactNotDisproven(t *testing.T) {
	k, ctx := setupKeeper(t)
	k.SetKnowledgeKeeper(&stubKnowledgeKeeper{disproven: map[string]bool{}})

	const claimID, factID, recipient = "claim-honest", "fact-honest", "zrn1recipient"
	sched, err := k.CreateVestingSchedule(ctx, claimID, factID, recipient,
		"1000000", types.CategoryComputational, types.SourceVerification)
	if err != nil {
		t.Fatalf("CreateVestingSchedule: %v", err)
	}
	before := *sched

	// The attack: any address names itself challenger and calls.
	_, err = k.FalsifyClaim(ctx, claimID, "zrn1attacker")
	if err == nil {
		t.Fatal("FalsifyClaim succeeded against a fact that was never disproven — " +
			"this is the live unauthenticated destruction primitive")
	}
	if !types.ErrFactNotDisproven.Is(err) {
		t.Fatalf("expected ErrFactNotDisproven, got %v", err)
	}

	after, found := k.GetVestingByClaimId(ctx, claimID)
	if !found {
		t.Fatal("schedule vanished after a refused falsification")
	}
	if after.Status != before.Status {
		t.Fatalf("refused falsification still changed status: %q -> %q", before.Status, after.Status)
	}
	if after.TotalAmount != before.TotalAmount {
		t.Fatalf("refused falsification still changed total: %q -> %q", before.TotalAmount, after.TotalAmount)
	}
}

func TestFalsifyClaim_AllowedWhenFactIsDisproven(t *testing.T) {
	k, ctx := setupKeeper(t)
	const claimID, factID = "claim-false", "fact-false"
	k.SetKnowledgeKeeper(&stubKnowledgeKeeper{disproven: map[string]bool{factID: true}})

	if _, err := k.CreateVestingSchedule(ctx, claimID, factID, "zrn1recipient",
		"1000000", types.CategoryComputational, types.SourceVerification); err != nil {
		t.Fatalf("CreateVestingSchedule: %v", err)
	}

	rec, err := k.FalsifyClaim(ctx, claimID, "zrn1challenger")
	if err != nil {
		t.Fatalf("FalsifyClaim refused an adjudicated-false fact: %v", err)
	}
	if rec == nil {
		t.Fatal("expected a clawback record")
	}
	after, found := k.GetVestingByClaimId(ctx, claimID)
	if !found {
		t.Fatal("schedule missing after falsification")
	}
	if after.Status != string(types.VestingStatusFalsified) {
		t.Fatalf("expected status %q, got %q", types.VestingStatusFalsified, after.Status)
	}
}

// The gate must FAIL CLOSED. If the adjudication source is unavailable we
// refuse to destroy rewards rather than allowing the destruction — the
// opposite of the fail-open pattern the audit found elsewhere on this chain.
func TestFalsifyClaim_FailsClosedWithoutKnowledgeKeeper(t *testing.T) {
	k, ctx := setupKeeper(t)
	// deliberately do NOT call SetKnowledgeKeeper

	const claimID = "claim-orphan"
	if _, err := k.CreateVestingSchedule(ctx, claimID, "fact-orphan", "zrn1recipient",
		"1000000", types.CategoryComputational, types.SourceVerification); err != nil {
		t.Fatalf("CreateVestingSchedule: %v", err)
	}

	_, err := k.FalsifyClaim(ctx, claimID, "zrn1attacker")
	if err == nil {
		t.Fatal("FalsifyClaim succeeded with no adjudication source wired — gate fails OPEN")
	}
	if !types.ErrAdjudicationUnavailable.Is(err) {
		t.Fatalf("expected ErrAdjudicationUnavailable, got %v", err)
	}
}

// An id naming no fact at all must not authorise a clawback.
func TestFalsifyClaim_UnknownFactIsNotDisproven(t *testing.T) {
	k, ctx := setupKeeper(t)
	k.SetKnowledgeKeeper(&stubKnowledgeKeeper{disproven: map[string]bool{"some-other-fact": true}})

	const claimID = "claim-unknown-fact"
	if _, err := k.CreateVestingSchedule(ctx, claimID, "", "zrn1recipient",
		"1000000", types.CategoryComputational, types.SourceVerification); err != nil {
		t.Fatalf("CreateVestingSchedule: %v", err)
	}

	if _, err := k.FalsifyClaim(ctx, claimID, "zrn1attacker"); err == nil {
		t.Fatal("an empty fact id authorised a clawback")
	}
}
