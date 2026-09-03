package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/zerone-chain/zerone/x/sponsorship/keeper"
	"github.com/zerone-chain/zerone/x/sponsorship/types"
)

func TestFulfillmentQueryReturnsReplayEvidence(t *testing.T) {
	k, ctx, bank, knowledge := setup(t)
	msgServer := keeper.NewMsgServerImpl(k)
	sponsor := mkAddr("query-settlement-sponsor")
	worker := mkAddr("query-settlement-worker")
	bountyID := createTestBounty(
		t,
		k,
		msgServer,
		ctx,
		bank,
		sponsor,
		"mathematics",
		"1000000",
		1,
		500,
		worker.String(),
	)
	makeVerifiedFact(t, knowledge, "query-settlement-fact", "mathematics", worker.String(), 1000)
	_, err := msgServer.FulfillBounty(ctx, &types.MsgFulfillBounty{
		Caller: worker.String(), BountyId: bountyID, FactId: "query-settlement-fact",
	})
	require.NoError(t, err)

	response, err := keeper.NewQueryServerImpl(k).Fulfillment(
		ctx,
		&types.QueryFulfillmentRequest{
			BountyId: bountyID,
			FactId:   "query-settlement-fact",
		},
	)
	require.NoError(t, err)
	require.Equal(t, bountyID, response.Fulfillment.BountyId)
	require.Equal(t, worker.String(), response.Fulfillment.Worker)
	require.Equal(t, bountyID, response.FactConsumedByBountyId)
	require.Equal(t, bountyID, response.ReceiptConsumedByBountyId)
	require.Equal(t, bountyID, response.SettlementNullifierConsumedByBountyId)
	require.Equal(t, uint64(ctx.BlockHeight()), response.SnapshotBlockHeight)
}

func TestFulfillmentQueryValidatesIdentityAndPresence(t *testing.T) {
	k, ctx, _, _ := setup(t)
	query := keeper.NewQueryServerImpl(k)

	response, err := query.Fulfillment(ctx, &types.QueryFulfillmentRequest{})
	require.Nil(t, response)
	require.Equal(t, codes.InvalidArgument, status.Code(err))

	response, err = query.Fulfillment(ctx, &types.QueryFulfillmentRequest{
		BountyId: "missing-bounty",
		FactId:   "missing-fact",
	})
	require.Nil(t, response)
	require.Equal(t, codes.NotFound, status.Code(err))
}

func TestEscrowAccountingQueryReportsSolvencyAndSurplus(t *testing.T) {
	k, ctx, bank, _ := setup(t)
	query := keeper.NewQueryServerImpl(k)
	sponsor := mkAddr("query-accounting-sponsor")
	createTestBounty(
		t,
		k,
		keeper.NewMsgServerImpl(k),
		ctx,
		bank,
		sponsor,
		"mathematics",
		"1000000",
		1,
		500,
	)

	response, err := query.EscrowAccounting(
		ctx,
		&types.QueryEscrowAccountingRequest{},
	)
	require.NoError(t, err)
	require.Equal(t, "uzrn", response.Denom)
	require.Equal(t, "1000000", response.PersistedLiability)
	require.Equal(t, "1000000", response.ModuleBalance)
	require.Equal(t, "0", response.Surplus)
	require.True(t, response.Solvent)

	bank.moduleBalances[types.ModuleName]["uzrn"] = 999999
	response, err = query.EscrowAccounting(
		ctx,
		&types.QueryEscrowAccountingRequest{},
	)
	require.NoError(t, err)
	require.Equal(t, "-1", response.Surplus)
	require.False(t, response.Solvent)
}
