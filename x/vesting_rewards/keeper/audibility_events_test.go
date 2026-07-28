package keeper_test

import (
	"math/big"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/x/vesting_rewards/types"
)

// ---------- Audibility Event Tests (schedule_created / claimed) ----------

func eventsOfType(ctx sdk.Context, eventType string) []sdk.Event {
	var out []sdk.Event
	for _, ev := range ctx.EventManager().Events() {
		if ev.Type == eventType {
			out = append(out, ev)
		}
	}
	return out
}

func eventAttr(t *testing.T, ev sdk.Event, key string) string {
	t.Helper()
	for _, attr := range ev.Attributes {
		if attr.Key == key {
			return attr.Value
		}
	}
	t.Fatalf("event %s missing attribute %q", ev.Type, key)
	return ""
}

func TestScheduleCreated_EmitsAudibilityEvent(t *testing.T) {
	k, ctx := setupKeeper(t)

	recipient := sdk.AccAddress("recipient1__________").String()
	total := "1000000000000000000"

	schedule, err := k.CreateVestingSchedule(ctx, "claim-1", "fact-1", recipient,
		total, types.CategoryFormalProof, types.SourceVerification)
	if err != nil {
		t.Fatalf("create vesting failed: %v", err)
	}

	events := eventsOfType(ctx, "zerone.vesting_rewards.schedule_created")
	if len(events) != 1 {
		t.Fatalf("expected 1 schedule_created event, got %d", len(events))
	}
	ev := events[0]
	if got := eventAttr(t, ev, "beneficiary"); got != recipient {
		t.Errorf("beneficiary: expected %s, got %s", recipient, got)
	}
	if got := eventAttr(t, ev, "schedule_id"); got != schedule.Id {
		t.Errorf("schedule_id: expected %s, got %s", schedule.Id, got)
	}
	if got := eventAttr(t, ev, "total_uzrn"); got != total {
		t.Errorf("total_uzrn: expected %s, got %s", total, got)
	}
	if got := eventAttr(t, ev, "category"); got != string(types.CategoryFormalProof) {
		t.Errorf("category: expected %s, got %s", types.CategoryFormalProof, got)
	}
	if got := eventAttr(t, ev, "source"); got != string(types.SourceVerification) {
		t.Errorf("source: expected %s, got %s", types.SourceVerification, got)
	}
}

// The event lives in the core constructor, so callers that do not build the
// schedule themselves (falsification rewards, the knowledge adapter) are
// covered without their own emission site.
func TestScheduleCreated_EmittedViaFalsificationRewardPath(t *testing.T) {
	k, ctx := setupKeeper(t)

	recipient := sdk.AccAddress("challenger1_________").String()

	if err := k.DistributeFalsificationReward(ctx, "counter-1", "target-1", recipient, "5000000"); err != nil {
		t.Fatalf("distribute falsification reward failed: %v", err)
	}

	events := eventsOfType(ctx, "zerone.vesting_rewards.schedule_created")
	if len(events) != 1 {
		t.Fatalf("expected 1 schedule_created event, got %d", len(events))
	}
	ev := events[0]
	if got := eventAttr(t, ev, "beneficiary"); got != recipient {
		t.Errorf("beneficiary: expected %s, got %s", recipient, got)
	}
	if got := eventAttr(t, ev, "total_uzrn"); got != "5000000" {
		t.Errorf("total_uzrn: expected 5000000, got %s", got)
	}
	if got := eventAttr(t, ev, "category"); got != string(types.CategoryComputational) {
		t.Errorf("category: expected %s, got %s", types.CategoryComputational, got)
	}
	if got := eventAttr(t, ev, "source"); got != string(types.SourceFalsification) {
		t.Errorf("source: expected %s, got %s", types.SourceFalsification, got)
	}
}

func TestClaimed_EmitsAudibilityEventWithArithmetic(t *testing.T) {
	k, ctx := setupKeeper(t)

	recipient := sdk.AccAddress("recipient1__________").String()
	total := "1000000000000000000"

	schedule, err := k.CreateVestingSchedule(ctx, "claim-1", "fact-1", recipient,
		total, types.CategoryFormalProof, types.SourceVerification)
	if err != nil {
		t.Fatalf("create vesting failed: %v", err)
	}

	ctx = ctx.WithBlockHeight(20000)

	claimed, err := k.ClaimRewards(ctx, recipient, schedule.Id)
	if err != nil {
		t.Fatalf("claim rewards failed: %v", err)
	}
	if claimed == "0" {
		t.Fatal("expected non-zero claimed amount after cliff")
	}

	events := eventsOfType(ctx, "zerone.vesting_rewards.claimed")
	if len(events) != 1 {
		t.Fatalf("expected 1 claimed event, got %d", len(events))
	}
	ev := events[0]
	if got := eventAttr(t, ev, "beneficiary"); got != recipient {
		t.Errorf("beneficiary: expected %s, got %s", recipient, got)
	}
	if got := eventAttr(t, ev, "schedule_id"); got != schedule.Id {
		t.Errorf("schedule_id: expected %s, got %s", schedule.Id, got)
	}
	if got := eventAttr(t, ev, "amount_uzrn"); got != claimed {
		t.Errorf("amount_uzrn: expected %s (returned claim), got %s", claimed, got)
	}

	// remaining_uzrn must equal TotalAmount - ReleasedAmount after the claim.
	stored, found := k.GetVestingSchedule(ctx, schedule.Id)
	if !found {
		t.Fatal("schedule not found after claim")
	}
	totalBig, _ := new(big.Int).SetString(total, 10)
	releasedBig, _ := new(big.Int).SetString(stored.ReleasedAmount, 10)
	wantRemaining := new(big.Int).Sub(totalBig, releasedBig)

	gotRemaining, ok := new(big.Int).SetString(eventAttr(t, ev, "remaining_uzrn"), 10)
	if !ok {
		t.Fatal("remaining_uzrn is not a valid integer")
	}
	if gotRemaining.Cmp(wantRemaining) != 0 {
		t.Errorf("remaining_uzrn: expected %s, got %s", wantRemaining, gotRemaining)
	}

	// First claim on a fresh schedule: amount + remaining must equal total.
	amountBig, _ := new(big.Int).SetString(claimed, 10)
	sum := new(big.Int).Add(amountBig, gotRemaining)
	if sum.Cmp(totalBig) != 0 {
		t.Errorf("amount + remaining: expected %s, got %s", totalBig, sum)
	}
}

func TestClaimed_EmitsOneEventPerScheduleOnBatchClaim(t *testing.T) {
	k, ctx := setupKeeper(t)

	recipient := sdk.AccAddress("recipient1__________").String()

	s1, err := k.CreateVestingSchedule(ctx, "claim-1", "fact-1", recipient,
		"1000000000000000000", types.CategoryFormalProof, types.SourceVerification)
	if err != nil {
		t.Fatalf("create vesting 1 failed: %v", err)
	}
	s2, err := k.CreateVestingSchedule(ctx, "claim-2", "fact-2", recipient,
		"2000000000000000000", types.CategoryFormalProof, types.SourceVerification)
	if err != nil {
		t.Fatalf("create vesting 2 failed: %v", err)
	}

	ctx = ctx.WithBlockHeight(20000)

	claimed, err := k.ClaimRewards(ctx, recipient, "")
	if err != nil {
		t.Fatalf("batch claim failed: %v", err)
	}

	events := eventsOfType(ctx, "zerone.vesting_rewards.claimed")
	if len(events) != 2 {
		t.Fatalf("expected 2 claimed events, got %d", len(events))
	}

	amounts := make(map[string]*big.Int)
	for _, ev := range events {
		id := eventAttr(t, ev, "schedule_id")
		amount, ok := new(big.Int).SetString(eventAttr(t, ev, "amount_uzrn"), 10)
		if !ok {
			t.Fatalf("amount_uzrn is not a valid integer for schedule %s", id)
		}
		amounts[id] = amount
	}
	if _, ok := amounts[s1.Id]; !ok {
		t.Errorf("no claimed event for schedule %s", s1.Id)
	}
	if _, ok := amounts[s2.Id]; !ok {
		t.Errorf("no claimed event for schedule %s", s2.Id)
	}

	// Per-schedule amounts must sum to the returned total.
	sum := new(big.Int)
	for _, amount := range amounts {
		sum.Add(sum, amount)
	}
	claimedBig, _ := new(big.Int).SetString(claimed, 10)
	if sum.Cmp(claimedBig) != 0 {
		t.Errorf("sum of per-schedule amounts: expected %s, got %s", claimedBig, sum)
	}
}
