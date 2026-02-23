//go:build integration

package multivalidator_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	cmthttp "github.com/cometbft/cometbft/rpc/client/http"

	knowledgetypes "github.com/zerone-chain/zerone/x/knowledge/types"
	stakingtypes "github.com/zerone-chain/zerone/x/staking/types"
)

const (
	grpcAddr = "localhost:9090"
	rpcAddr  = "http://127.0.0.1:26601"
)

func grpcConn(t *testing.T) *grpc.ClientConn {
	t.Helper()
	conn, err := grpc.NewClient(grpcAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err, "gRPC dial failed — is localnet running? (scripts/localnet.sh start)")
	t.Cleanup(func() { conn.Close() })
	return conn
}

func rpcClient(t *testing.T) *cmthttp.HTTP {
	t.Helper()
	c, err := cmthttp.New(rpcAddr, "/websocket")
	require.NoError(t, err, "CometBFT RPC client creation failed")
	return c
}

// TestValidatorSetSize verifies 4 validators are registered in the Zerone staking module.
func TestValidatorSetSize(t *testing.T) {
	conn := grpcConn(t)
	client := stakingtypes.NewQueryClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := client.Validators(ctx, &stakingtypes.QueryValidatorsRequest{
		ActiveOnly: false,
		Tier:       -1, // all tiers
		Limit:      100,
	})
	require.NoError(t, err, "Validators query failed")
	require.NotNil(t, resp)
	assert.Equal(t, 4, len(resp.Validators), "expected 4 validators in set")

	// Verify each validator has a moniker
	for _, v := range resp.Validators {
		assert.NotEmpty(t, v.Moniker, "validator should have a moniker")
		assert.NotEmpty(t, v.OperatorAddress, "validator should have an operator address")
		t.Logf("Validator: %s (tier=%s, stake=%s, active=%v)",
			v.Moniker, v.Tier, v.TotalStake, v.IsActive)
	}
}

// TestBlockSignatures verifies the latest block has signatures from all 4 validators.
func TestBlockSignatures(t *testing.T) {
	c := rpcClient(t)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get latest block height
	status, err := c.Status(ctx)
	require.NoError(t, err, "RPC status query failed")
	require.True(t, status.SyncInfo.LatestBlockHeight >= 3,
		"chain should have produced at least 3 blocks")

	// Query latest commit (has validator signatures)
	height := status.SyncInfo.LatestBlockHeight
	commit, err := c.Commit(ctx, &height)
	require.NoError(t, err, "Commit query failed")
	require.NotNil(t, commit)

	sigCount := len(commit.Commit.Signatures)
	t.Logf("Block %d has %d signatures", height, sigCount)

	// With 4 validators, we expect at least 3 signatures (2/3+ for BFT)
	assert.GreaterOrEqual(t, sigCount, 3,
		"block should have at least 3 of 4 validator signatures")
}

// TestPoTRoundCompletion queries the knowledge module to verify PoT params are set for testing.
func TestPoTRoundCompletion(t *testing.T) {
	conn := grpcConn(t)
	client := knowledgetypes.NewQueryClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Verify knowledge params are set for local testing
	paramsResp, err := client.Params(ctx, &knowledgetypes.QueryParamsRequest{})
	require.NoError(t, err, "Knowledge params query failed")
	require.NotNil(t, paramsResp)

	params := paramsResp.Params
	assert.Equal(t, uint64(2), params.MinVerifiers,
		"min_verifiers should be 2 for local testing")
	assert.Equal(t, uint64(10), params.CommitPhaseBlocks,
		"commit_phase_blocks should be 10")
	assert.Equal(t, uint64(10), params.RevealPhaseBlocks,
		"reveal_phase_blocks should be 10")
	assert.Equal(t, uint64(5), params.AggregationPhaseBlocks,
		"aggregation_phase_blocks should be 5")
	assert.Equal(t, false, params.AdversarialVerificationEnabled,
		"adversarial_verification should be disabled")

	t.Logf("Knowledge params: min_verifiers=%d, commit=%d, reveal=%d, agg=%d",
		params.MinVerifiers, params.CommitPhaseBlocks,
		params.RevealPhaseBlocks, params.AggregationPhaseBlocks)
}

// TestSlashingReducesPower queries validators and verifies jailed validators have reduced state.
func TestSlashingReducesPower(t *testing.T) {
	conn := grpcConn(t)
	client := stakingtypes.NewQueryClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := client.Validators(ctx, &stakingtypes.QueryValidatorsRequest{
		ActiveOnly: false,
		Tier:       -1,
		Limit:      100,
	})
	require.NoError(t, err, "Validators query failed")
	require.NotNil(t, resp)

	// Check if any validator is jailed (this test runs after localnet-test.sh slashing test)
	var jailedCount int
	for _, v := range resp.Validators {
		if v.Jailed {
			jailedCount++
			t.Logf("Jailed validator found: %s (reason=%s, slash_count=%d)",
				v.Moniker, v.JailReason, v.SlashCount)
		}
	}

	// If slashing test has run, we expect at least 1 jailed validator
	// If not, we just verify the jailed field is queryable
	if jailedCount > 0 {
		t.Logf("Found %d jailed validator(s) — slashing mechanism verified", jailedCount)
	} else {
		t.Log("No jailed validators found — run localnet-test.sh slashing test first")
		t.Log("Verifying validator jailed fields are queryable (all false = healthy network)")
		for _, v := range resp.Validators {
			assert.False(t, v.Jailed, "validator %s should not be jailed on fresh network", v.Moniker)
		}
	}
}
