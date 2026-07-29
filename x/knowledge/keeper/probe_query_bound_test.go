package keeper_test

import (
	"fmt"
	"math"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/zerone-chain/zerone/x/knowledge/keeper"
	"github.com/zerone-chain/zerone/x/knowledge/types"
)

func TestQueryIdleFacts_MaxUint32ClampsBeforeAllocation(t *testing.T) {
	k, ctx := setupKnowledgeTest(t)
	const domain = "idle-max-limit"

	// These IDs sort before the genesis doctrine facts, making the query's
	// bounded scan window consist entirely of this fixture set.
	for i := uint32(0); i < keeper.IdleFactsScanCap+1; i++ {
		require.NoError(t, k.SetFact(ctx, &types.Fact{
			Id:                  fmt.Sprintf("!idle-max-%04d", i),
			Domain:              domain,
			Status:              types.FactStatus_FACT_STATUS_VERIFIED,
			Confidence:          900_000,
			ProbeInvitedAtBlock: uint64(i + 1),
		}))
	}

	query := keeper.NewQueryServerImpl(k)
	resp, err := query.IdleFacts(ctx, &types.QueryIdleFactsRequest{
		Domain: domain,
		Limit:  math.MaxUint32,
	})
	require.NoError(t, err)
	require.Len(t, resp.Facts, int(keeper.IdleFactsScanCap))

	// The first entry beyond the work/result cap must not leak into the
	// response even when the caller asks for MaxUint32 entries.
	for _, fact := range resp.Facts {
		require.NotEqual(t, fmt.Sprintf("!idle-max-%04d", keeper.IdleFactsScanCap), fact.Id)
	}
}

func TestQueryIdleFacts_NoMatchesReturnsEmpty(t *testing.T) {
	k, ctx := setupKnowledgeTest(t)
	const domain = "idle-no-match"

	for i := 0; i < 8; i++ {
		require.NoError(t, k.SetFact(ctx, &types.Fact{
			Id:                  fmt.Sprintf("!idle-none-%02d", i),
			Domain:              domain,
			Status:              types.FactStatus_FACT_STATUS_PENDING,
			ProbeInvitedAtBlock: 1,
		}))
	}

	query := keeper.NewQueryServerImpl(k)
	resp, err := query.IdleFacts(ctx, &types.QueryIdleFactsRequest{
		Domain: domain,
		Limit:  1,
	})
	require.NoError(t, err)
	require.Empty(t, resp.Facts)
}

func TestQueryIdleFacts_NoMatchScanStopsBeforeEligibleFactBeyondCap(t *testing.T) {
	k, ctx := setupKnowledgeTest(t)
	const domain = "idle-beyond-scan-cap"

	// Fill the entire bounded scan window with non-matching facts.
	for i := uint32(0); i < keeper.IdleFactsScanCap; i++ {
		require.NoError(t, k.SetFact(ctx, &types.Fact{
			Id:                  fmt.Sprintf("!idle-scan-%04d", i),
			Domain:              domain,
			Status:              types.FactStatus_FACT_STATUS_PENDING,
			ProbeInvitedAtBlock: 1,
		}))
	}

	// This eligible fact sorts after both the fixtures and genesis doctrine
	// facts. An unbounded no-match scan would reach it and return it.
	require.NoError(t, k.SetFact(ctx, &types.Fact{
		Id:                  "~idle-beyond-cap",
		Domain:              domain,
		Status:              types.FactStatus_FACT_STATUS_VERIFIED,
		Confidence:          900_000,
		ProbeInvitedAtBlock: 1,
	}))

	query := keeper.NewQueryServerImpl(k)
	resp, err := query.IdleFacts(ctx, &types.QueryIdleFactsRequest{
		Domain: domain,
		Limit:  1,
	})
	require.NoError(t, err)
	require.Empty(t, resp.Facts,
		"an eligible fact beyond the scan cap proves the no-match path walked the full store")
}

func TestQueryIdleFacts_OnlyVerifiedAndActiveRemainEligible(t *testing.T) {
	k, ctx := setupKnowledgeTest(t)
	const domain = "idle-status-gate"

	statuses := []types.FactStatus{
		types.FactStatus_FACT_STATUS_UNSPECIFIED,
		types.FactStatus_FACT_STATUS_PENDING,
		types.FactStatus_FACT_STATUS_PROVISIONAL,
		types.FactStatus_FACT_STATUS_VERIFIED,
		types.FactStatus_FACT_STATUS_ACTIVE,
		types.FactStatus_FACT_STATUS_CONTESTED,
		types.FactStatus_FACT_STATUS_CHALLENGED,
		types.FactStatus_FACT_STATUS_SUPERSEDED,
		types.FactStatus_FACT_STATUS_EXPIRED,
		types.FactStatus_FACT_STATUS_DISPROVEN,
		types.FactStatus_FACT_STATUS_REVOKED,
		types.FactStatus_FACT_STATUS_AT_RISK,
		types.FactStatus_FACT_STATUS_PRUNED,
	}
	for i, status := range statuses {
		invitedAt := uint64(20)
		if status == types.FactStatus_FACT_STATUS_ACTIVE {
			invitedAt = 10
		}
		require.NoError(t, k.SetFact(ctx, &types.Fact{
			Id:                  fmt.Sprintf("!idle-status-%02d", i),
			Domain:              domain,
			Status:              status,
			Confidence:          900_000,
			ProbeInvitedAtBlock: invitedAt,
		}))
	}

	query := keeper.NewQueryServerImpl(k)
	resp, err := query.IdleFacts(ctx, &types.QueryIdleFactsRequest{Domain: domain})
	require.NoError(t, err)
	require.Len(t, resp.Facts, 2)
	require.Equal(t, "!idle-status-04", resp.Facts[0].Id,
		"oldest still-active invitation should be first")
	require.Equal(t, "!idle-status-03", resp.Facts[1].Id)
}
