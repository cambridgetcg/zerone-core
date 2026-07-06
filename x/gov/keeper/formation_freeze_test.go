package keeper_test

import (
	"testing"

	"github.com/zerone-chain/zerone/x/gov/keeper"
	"github.com/zerone-chain/zerone/x/gov/types"
)

// DomainFormationFreeze became witness-only in the slim cut: x/partnerships
// (the enforcement surface) was removed from consensus, so the handler now
// records the authority decree as a dated, signed event and nothing else.
// These tests witness that shape.

func TestDomainFormationFreeze_AuthorityOnly(t *testing.T) {
	k, ctx, _ := setupWithStaking(t, "1000000")

	ms := keeper.NewMsgServerImpl(k)

	_, err := ms.DomainFormationFreeze(ctx, &types.MsgDomainFormationFreeze{
		Authority:      testAddr("random"),
		Domain:         "physics",
		DurationBlocks: 1000,
		Reason:         "governance review",
	})
	if err == nil {
		t.Fatal("expected unauthorized error, got nil")
	}
}

func TestDomainFormationFreeze_WitnessOnlyDecree(t *testing.T) {
	k, ctx, _ := setupWithStaking(t, "1000000")

	ms := keeper.NewMsgServerImpl(k)

	_, err := ms.DomainFormationFreeze(ctx, &types.MsgDomainFormationFreeze{
		Authority:      k.GetAuthority(),
		Domain:         "physics",
		DurationBlocks: 1000,
		Reason:         "governance review",
	})
	if err != nil {
		t.Fatalf("expected witness-only decree to succeed, got: %v", err)
	}

	// The decree's only on-chain effect is the emitted event.
	found := false
	for _, ev := range ctx.EventManager().Events() {
		if ev.Type == "zerone.gov.domain_formation_freeze" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected zerone.gov.domain_formation_freeze event")
	}
}

func TestDomainFormationFreeze_Validation(t *testing.T) {
	k, ctx, _ := setupWithStaking(t, "1000000")

	ms := keeper.NewMsgServerImpl(k)

	if _, err := ms.DomainFormationFreeze(ctx, &types.MsgDomainFormationFreeze{
		Authority:      k.GetAuthority(),
		Domain:         "",
		DurationBlocks: 1000,
	}); err == nil {
		t.Fatal("expected error for empty domain")
	}

	if _, err := ms.DomainFormationFreeze(ctx, &types.MsgDomainFormationFreeze{
		Authority:      k.GetAuthority(),
		Domain:         "physics",
		DurationBlocks: 0,
	}); err == nil {
		t.Fatal("expected error for zero duration")
	}
}
