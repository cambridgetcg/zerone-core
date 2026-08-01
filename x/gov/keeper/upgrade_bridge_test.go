package keeper_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/zerone-chain/zerone/x/gov/keeper"
	"github.com/zerone-chain/zerone/x/gov/types"
)

// ---------- Mock Upgrade Keeper (bridge-specific) ----------

type bridgeMockUpgradeKeeper struct {
	called bool
	plan   *types.UpgradePlan
	err    error
}

func (m *bridgeMockUpgradeKeeper) ScheduleUpgrade(_ context.Context, plan *types.UpgradePlan) error {
	m.called = true
	m.plan = plan
	return m.err
}

// ---------- Upgrade Bridge Integration Tests ----------

func TestUpgradeBridge_FullLifecycle(t *testing.T) {
	k, ctx, _ := setupWithStaking(t, "1000000")

	mockUK := &bridgeMockUpgradeKeeper{}
	k.SetUpgradeKeeper(mockUK)

	lip := seedLegacyUpgradeLIP(k, ctx, "LEGACY-BRIDGE", types.StatusVoting)
	lip.VotingEndBlock = uint64(ctx.BlockHeight())
	lip.YesStake = "1000000"
	lip.UniqueVoters = 1
	k.SetLIP(ctx, lip)
	k.SetUpgradePlan(ctx, lip.Id, &types.UpgradePlan{
		Name:   "v2.0.0",
		Height: 40000,
		Info:   "https://github.com/zerone-chain/zerone/releases/v2.0.0",
	})
	k.BeginBlocker(ctx)

	lip, _ = k.GetLIP(ctx, lip.Id)
	if lip.Stage != types.StatusFailed {
		t.Errorf("expected failed, got %s", lip.Stage)
	}

	if mockUK.called {
		t.Fatal("retired custom authority must not call ScheduleUpgrade")
	}

	// Assert the compatibility scheduling boundary failed closed, never
	// upgrade_scheduled. Aggregate authority-retirement events are emitted by
	// the named activation migration, not this legacy BeginBlock path.
	events := ctx.EventManager().Events()
	foundFailure := false
	for _, e := range events {
		if e.Type == "zerone.gov.upgrade_schedule_failed" {
			foundFailure = true
		}
		if e.Type == "zerone.gov.upgrade_scheduled" {
			t.Fatal("retired custom authority emitted upgrade_scheduled")
		}
	}
	if !foundFailure {
		t.Error("expected zerone.gov.upgrade_schedule_failed event")
	}
}

func TestRetireCustomUpgradeLIPsEmitsOneBoundedAggregateEvent(t *testing.T) {
	k, ctx := setupKeeper(t)
	const recordCount = 1_200
	lips := make([]*types.LIP, 0, recordCount)
	for i := 0; i < recordCount; i++ {
		lip := &types.LIP{
			Id:           fmt.Sprintf("LEGACY-ACTIVATION-%04d", i),
			Category:     types.CategoryUpgrade,
			Stage:        types.StatusReview,
			StakedAmount: "0",
		}
		k.SetLIP(ctx, lip)
		lips = append(lips, lip)
	}

	retired, err := k.RetireCustomUpgradeLIPs(ctx, lips)
	if err != nil {
		t.Fatal(err)
	}
	if retired != recordCount {
		t.Fatalf("retired=%d want %d", retired, recordCount)
	}

	var retirementEvents int
	for _, event := range ctx.EventManager().Events() {
		if event.Type != "zerone.gov.custom_upgrade_authority_retired" {
			continue
		}
		retirementEvents++
		attributes := make(map[string]string, len(event.Attributes))
		for _, attribute := range event.Attributes {
			attributes[attribute.Key] = attribute.Value
		}
		if attributes["retired_count"] != "1200" {
			t.Fatalf("unexpected retired_count: %q", attributes["retired_count"])
		}
	}
	if retirementEvents != 1 {
		t.Fatalf("retirement event count=%d want 1", retirementEvents)
	}

	for _, lip := range lips {
		stored, found := k.GetLIP(ctx, lip.Id)
		if !found || stored.Stage != types.StatusFailed {
			t.Fatalf("LIP %q was not terminalized: %+v", lip.Id, stored)
		}
	}
}

func TestRetireCustomUpgradeLIPsRefusesUnattributedStakeBeforeMutation(
	t *testing.T,
) {
	k, ctx := setupKeeper(t)
	first := &types.LIP{
		Id:           "LEGACY-ZERO",
		Category:     types.CategoryUpgrade,
		Stage:        types.StatusReview,
		StakedAmount: "0",
	}
	second := &types.LIP{
		Id:           "LEGACY-LOCKED",
		Category:     types.CategoryUpgrade,
		Stage:        types.StatusVoting,
		StakedAmount: "77",
	}
	k.SetLIP(ctx, first)
	k.SetLIP(ctx, second)

	_, err := k.RetireCustomUpgradeLIPs(
		ctx,
		[]*types.LIP{first, second},
	)
	if err == nil {
		t.Fatal("expected unattributed stake to block authority retirement")
	}
	if !strings.Contains(err.Error(), "reconcile it before authority retirement") {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, id := range []string{first.Id, second.Id} {
		stored, found := k.GetLIP(ctx, id)
		if !found || types.IsTerminal(stored.Stage) {
			t.Fatalf("guard failure partially mutated %q: %+v", id, stored)
		}
	}
}

func TestUpgradeBridge_NonUpgradeLIP_NoSchedule(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	// Submit a parameter-category LIP
	ms.SubmitLIP(ctx, &types.MsgSubmitLIP{
		Proposer:     testAddr("alice"),
		Title:        "Param Change",
		Description:  "Change voting period",
		Category:     types.CategoryParameter,
		InitialStake: "1000000",
	})

	// Attach upgrade plan → should fail (wrong category)
	_, err := ms.AttachUpgradePlan(ctx, &types.MsgAttachUpgradePlan{
		Proposer:    testAddr("alice"),
		LipId:       "LIP-1",
		UpgradeName: "v2.0.0",
		Height:      500,
		Info:        "release manifest",
	})
	if err == nil {
		t.Error("expected error when attaching upgrade plan to non-upgrade LIP")
	}
}

func TestUpgradeBridge_FailedLIP_NoSchedule(t *testing.T) {
	k, ctx, _ := setupWithStaking(t, "1000000")

	mockUK := &bridgeMockUpgradeKeeper{}
	k.SetUpgradeKeeper(mockUK)

	lip := seedLegacyUpgradeLIP(k, ctx, "LEGACY-FAILED", types.StatusFailed)
	k.SetUpgradePlan(ctx, lip.Id, &types.UpgradePlan{
		Name: "v2.0.0", Height: 1000, Info: "release manifest",
	})
	k.BeginBlocker(ctx)

	lip, _ = k.GetLIP(ctx, lip.Id)
	if lip.Stage != types.StatusFailed {
		t.Errorf("expected failed, got %s", lip.Stage)
	}

	// Assert ScheduleUpgrade was NOT called
	if mockUK.called {
		t.Error("ScheduleUpgrade should not be called for a failed LIP")
	}
}
