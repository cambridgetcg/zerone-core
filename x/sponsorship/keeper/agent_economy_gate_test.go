package keeper_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/zerone-chain/zerone/x/sponsorship/keeper"
	"github.com/zerone-chain/zerone/x/sponsorship/types"
)

func TestAgentEconomyGate_CreateBountyRejectsClosedOrUnreadableLineage(t *testing.T) {
	tests := []struct {
		name       string
		active     bool
		readErr    error
		wantDetail string
	}{
		{
			name:       "marker absent",
			wantDetail: "agent economy is not activated",
		},
		{
			name:       "lineage markers conflict",
			readErr:    errors.New("conflicting agent-economy lineages"),
			wantDetail: "conflicting agent-economy lineages",
		},
		{
			name:       "marker read fails",
			readErr:    errors.New("read agent-economy marker: injected store failure"),
			wantDetail: "injected store failure",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			k, ctx, bank, knowledge := setup(t)
			knowledge.agentEconomyActive = test.active
			knowledge.agentEconomyReadErr = test.readErr
			server := keeper.NewMsgServerImpl(k)
			sponsor := mkAddr("sealed-bounty-sponsor")
			bank.setBalance(sponsor.String(), "uzrn", 100_000_000)
			sponsorBefore := bank.balances[sponsor.String()]["uzrn"]
			moduleBefore := bank.moduleBalances[types.ModuleName]["uzrn"]

			response, err := server.CreateBountyOrder(ctx, &types.MsgCreateBountyOrder{
				Sponsor:          sponsor.String(),
				Domain:           "mathematics",
				PricePerArtifact: "1000000",
				TargetCount:      1,
				DurationBlocks:   500,
				WorkContract:     testWorkContract(),
			})

			if response != nil {
				t.Fatalf("lineage refusal returned a response: %+v", response)
			}
			if !errors.Is(err, types.ErrAgentEconomyDisabled) {
				t.Fatalf("expected ErrAgentEconomyDisabled, got %v", err)
			}
			if err == nil || !strings.Contains(err.Error(), test.wantDetail) {
				t.Fatalf("error %q does not contain %q", err, test.wantDetail)
			}
			if bank.balances[sponsor.String()]["uzrn"] != sponsorBefore ||
				bank.moduleBalances[types.ModuleName]["uzrn"] != moduleBefore {
				t.Fatal("lineage refusal moved escrow")
			}
			if orders := k.GetAllBountyOrders(ctx); len(orders) != 0 {
				t.Fatalf("lineage refusal created bounty orders: %+v", orders)
			}
		})
	}
}

func TestAgentEconomyGate_FulfillBountyRejectsClosedOrUnreadableLineage(t *testing.T) {
	tests := []struct {
		name       string
		active     bool
		readErr    error
		wantDetail string
	}{
		{
			name:       "marker absent",
			wantDetail: "agent economy is not activated",
		},
		{
			name:       "lineage markers conflict",
			readErr:    errors.New("conflicting agent-economy lineages"),
			wantDetail: "conflicting agent-economy lineages",
		},
		{
			name:       "marker read fails",
			readErr:    errors.New("read agent-economy marker: injected store failure"),
			wantDetail: "injected store failure",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			k, ctx, bank, knowledge := setup(t)
			server := keeper.NewMsgServerImpl(k)
			sponsor := mkAddr("sealed-fulfill-sponsor")
			worker := mkAddr("sealed-fulfill-worker")
			bountyID := createTestBounty(
				t,
				k,
				server,
				ctx,
				bank,
				sponsor,
				"mathematics",
				"1000000",
				1,
				500,
				worker.String(),
			)
			makeVerifiedFact(t, knowledge, "sealed-fact", "mathematics", worker.String(), 1000)
			knowledge.agentEconomyActive = test.active
			knowledge.agentEconomyReadErr = test.readErr
			workerBefore := bank.balances[worker.String()]["uzrn"]
			moduleBefore := bank.moduleBalances[types.ModuleName]["uzrn"]
			orderBefore, found := k.GetBountyOrder(ctx, bountyID)
			if !found {
				t.Fatal("setup bounty missing")
			}

			response, err := server.FulfillBounty(ctx, &types.MsgFulfillBounty{
				Caller:   worker.String(),
				BountyId: bountyID,
				FactId:   "sealed-fact",
			})

			if response != nil {
				t.Fatalf("lineage refusal returned a response: %+v", response)
			}
			if !errors.Is(err, types.ErrAgentEconomyDisabled) {
				t.Fatalf("expected ErrAgentEconomyDisabled, got %v", err)
			}
			if err == nil || !strings.Contains(err.Error(), test.wantDetail) {
				t.Fatalf("error %q does not contain %q", err, test.wantDetail)
			}
			if bank.balances[worker.String()]["uzrn"] != workerBefore ||
				bank.moduleBalances[types.ModuleName]["uzrn"] != moduleBefore {
				t.Fatal("lineage refusal paid from escrow")
			}
			orderAfter, found := k.GetBountyOrder(ctx, bountyID)
			if !found || orderAfter.FulfilledCount != orderBefore.FulfilledCount ||
				orderAfter.EscrowRemaining != orderBefore.EscrowRemaining ||
				orderAfter.Status != orderBefore.Status {
				t.Fatalf("lineage refusal mutated bounty: before=%+v after=%+v", orderBefore, orderAfter)
			}
			if _, found := k.GetFulfillment(ctx, bountyID, "sealed-fact"); found {
				t.Fatal("lineage refusal stored a fulfillment")
			}
		})
	}
}

func TestAgentEconomyGate_CancelRefundRemainsOpenWhenLineageIsClosedOrUnreadable(t *testing.T) {
	tests := []struct {
		name    string
		readErr error
	}{
		{name: "marker absent"},
		{name: "marker read fails", readErr: errors.New("injected marker store failure")},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			k, ctx, bank, knowledge := setup(t)
			server := keeper.NewMsgServerImpl(k)
			sponsor := mkAddr("sealed-cancel-sponsor")
			bountyID := createTestBounty(
				t,
				k,
				server,
				ctx,
				bank,
				sponsor,
				"mathematics",
				"1000000",
				1,
				500,
			)
			knowledge.agentEconomyActive = false
			knowledge.agentEconomyReadErr = test.readErr
			ctx = ctx.WithBlockHeight(1500)
			sponsorBefore := bank.balances[sponsor.String()]["uzrn"]

			response, err := server.CancelBountyOrder(ctx, &types.MsgCancelBountyOrder{
				Sponsor: sponsor.String(), BountyId: bountyID,
			})

			if err != nil {
				t.Fatalf("cancel must remain available to recover sealed escrow: %v", err)
			}
			if response.RefundedAmount != "1000000" {
				t.Fatalf("unexpected refund: %s", response.RefundedAmount)
			}
			if delta := bank.balances[sponsor.String()]["uzrn"] - sponsorBefore; delta != 1_000_000 {
				t.Fatalf("refund delta: got %d want 1000000", delta)
			}
			order, found := k.GetBountyOrder(ctx, bountyID)
			if !found || order.Status != types.BountyStatus_BOUNTY_STATUS_CANCELED || order.EscrowRemaining != "0" {
				t.Fatalf("cancel did not terminally close the sealed order: %+v", order)
			}
		})
	}
}
