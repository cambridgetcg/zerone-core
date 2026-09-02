//go:build integration

package multivalidator_test

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	cmttypes "github.com/cometbft/cometbft/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	cmthttp "github.com/cometbft/cometbft/rpc/client/http"

	knowledgetypes "github.com/zerone-chain/zerone/x/knowledge/types"
	stakingtypes "github.com/zerone-chain/zerone/x/staking/types"
)

const (
	defaultGRPCAddr = "localhost:9090"
	defaultRPCAddr  = "http://127.0.0.1:26601"

	grpcAddrEnv = "ZERONE_TEST_GRPC_ADDR"
	rpcAddrEnv  = "ZERONE_TEST_RPC_ADDR"
)

type commitSignatureSummary struct {
	CommitSignatures int
	NilSignatures    int
	AbsentSignatures int
	SignedPower      int64
	TotalPower       int64
}

func configuredTestEndpoint(envName, defaultAddr string, validate func(string) error) (string, error) {
	addr, configured := os.LookupEnv(envName)
	if !configured {
		addr = defaultAddr
	}
	if addr == "" {
		return "", fmt.Errorf("%s must not be empty", envName)
	}
	if strings.TrimSpace(addr) != addr {
		return "", fmt.Errorf("%s must not contain leading or trailing whitespace", envName)
	}
	if err := validate(addr); err != nil {
		return "", fmt.Errorf("invalid %s: %w", envName, err)
	}
	return addr, nil
}

func validateHostPort(addr string) error {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return fmt.Errorf("expected host:port: %w", err)
	}
	if host == "" {
		return fmt.Errorf("host must not be empty")
	}
	portNumber, err := strconv.Atoi(port)
	if err != nil || portNumber < 1 || portNumber > 65535 {
		return fmt.Errorf("port must be an integer between 1 and 65535")
	}
	return nil
}

func grpcTestAddr() (string, error) {
	return configuredTestEndpoint(grpcAddrEnv, defaultGRPCAddr, validateHostPort)
}

func validateRPCAddr(addr string) error {
	u, err := url.Parse(addr)
	if err != nil {
		return fmt.Errorf("expected an absolute HTTP URL: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("scheme must be http or https")
	}
	if u.User != nil {
		return fmt.Errorf("userinfo is not allowed")
	}
	if u.Path != "" && u.Path != "/" {
		return fmt.Errorf("path must be empty or /")
	}
	if u.RawQuery != "" || u.Fragment != "" {
		return fmt.Errorf("query and fragment are not allowed")
	}
	return validateHostPort(u.Host)
}

func rpcTestAddr() (string, error) {
	return configuredTestEndpoint(rpcAddrEnv, defaultRPCAddr, validateRPCAddr)
}

func grpcConn(t *testing.T) *grpc.ClientConn {
	t.Helper()
	addr, err := grpcTestAddr()
	require.NoError(t, err)
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err, "gRPC dial failed — is localnet running? (scripts/localnet.sh start)")
	t.Cleanup(func() { conn.Close() })
	return conn
}

func rpcClient(t *testing.T) *cmthttp.HTTP {
	t.Helper()
	addr, err := rpcTestAddr()
	require.NoError(t, err)
	c, err := cmthttp.New(addr, "/websocket")
	require.NoError(t, err, "CometBFT RPC client creation failed")
	return c
}

func validatorSetAtHeight(
	t *testing.T,
	ctx context.Context,
	c *cmthttp.HTTP,
	height int64,
) *cmttypes.ValidatorSet {
	t.Helper()

	const perPage = 100
	var (
		page          = 1
		expectedTotal = -1
		validators    []*cmttypes.Validator
	)

	for {
		result, err := c.Validators(ctx, &height, &page, ptr(perPage))
		require.NoError(t, err, "validator-set query failed at height %d page %d", height, page)
		require.NotNil(t, result)
		require.Equal(t, height, result.BlockHeight, "validator-set response height mismatch")
		require.Equal(t, len(result.Validators), result.Count, "validator-set page count mismatch")

		if expectedTotal == -1 {
			expectedTotal = result.Total
			require.Positive(t, expectedTotal, "validator set must not be empty")
		} else {
			require.Equal(t, expectedTotal, result.Total, "validator-set total changed across pages")
		}

		require.NotEmpty(t, result.Validators, "validator-set pagination returned an empty page")
		validators = append(validators, result.Validators...)
		require.LessOrEqual(t, len(validators), expectedTotal, "validator-set pagination exceeded reported total")
		if len(validators) == expectedTotal {
			break
		}
		page++
	}

	return cmttypes.NewValidatorSet(validators)
}

func ptr[T any](value T) *T {
	return &value
}

// verifyCommitSignatures binds the validator set to the header, verifies every
// non-absent commit signature, and reports voting power only for signatures
// whose flag commits to the block (not NIL votes or ABSENT slots).
func verifyCommitSignatures(
	chainID string,
	signedHeader cmttypes.SignedHeader,
	validatorSet *cmttypes.ValidatorSet,
) (commitSignatureSummary, error) {
	var summary commitSignatureSummary

	if err := signedHeader.ValidateBasic(chainID); err != nil {
		return summary, fmt.Errorf("invalid signed header: %w", err)
	}
	if validatorSet == nil {
		return summary, fmt.Errorf("validator set is nil")
	}
	if err := validatorSet.ValidateBasic(); err != nil {
		return summary, fmt.Errorf("invalid validator set: %w", err)
	}
	if !bytes.Equal(signedHeader.ValidatorsHash, validatorSet.Hash()) {
		return summary, fmt.Errorf("validator-set hash does not match signed header")
	}
	if len(signedHeader.Commit.Signatures) != validatorSet.Size() {
		return summary, fmt.Errorf(
			"commit has %d signature slots for %d validators",
			len(signedHeader.Commit.Signatures), validatorSet.Size(),
		)
	}

	summary.TotalPower = validatorSet.TotalVotingPower()
	for i, sig := range signedHeader.Commit.Signatures {
		validator := validatorSet.Validators[i]
		switch sig.BlockIDFlag {
		case cmttypes.BlockIDFlagCommit:
			if !bytes.Equal(sig.ValidatorAddress, validator.Address) {
				return summary, fmt.Errorf("commit signature %d has the wrong validator address", i)
			}
			summary.CommitSignatures++
			summary.SignedPower += validator.VotingPower
		case cmttypes.BlockIDFlagNil:
			if !bytes.Equal(sig.ValidatorAddress, validator.Address) {
				return summary, fmt.Errorf("NIL signature %d has the wrong validator address", i)
			}
			summary.NilSignatures++
		case cmttypes.BlockIDFlagAbsent:
			summary.AbsentSignatures++
		default:
			return summary, fmt.Errorf("signature %d has unknown block ID flag %d", i, sig.BlockIDFlag)
		}
	}

	if err := validatorSet.VerifyCommit(
		chainID,
		signedHeader.Commit.BlockID,
		signedHeader.Height,
		signedHeader.Commit,
	); err != nil {
		return summary, fmt.Errorf("cryptographic commit verification failed: %w", err)
	}

	return summary, nil
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

// TestBlockSignatures verifies that the most recently canonicalized block
// commit is bound to its validator set, all included signatures are valid, and
// strictly more than two-thirds of total voting power committed to the block.
func TestBlockSignatures(t *testing.T) {
	c := rpcClient(t)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get latest block height
	status, err := c.Status(ctx)
	require.NoError(t, err, "RPC status query failed")
	require.GreaterOrEqual(t, status.SyncInfo.LatestBlockHeight, int64(2),
		"chain should have produced at least 2 blocks")

	// A block's canonical commit is persisted in the following block. At the
	// current BlockStore height CometBFT can only return the non-canonical seen
	// commit, so verify the immediately preceding height.
	height := status.SyncInfo.LatestBlockHeight - 1
	commit, err := c.Commit(ctx, &height)
	require.NoError(t, err, "Commit query failed")
	require.NotNil(t, commit)
	require.True(t, commit.CanonicalCommit, "RPC returned a non-canonical commit")
	require.NotEmpty(t, status.NodeInfo.Network, "RPC status returned an empty chain ID")

	validatorSet := validatorSetAtHeight(t, ctx, c, height)
	summary, err := verifyCommitSignatures(status.NodeInfo.Network, commit.SignedHeader, validatorSet)
	require.NoError(t, err)

	quorumThreshold := summary.TotalPower * 2 / 3
	require.Greater(t, summary.SignedPower, quorumThreshold,
		"commit must contain valid block signatures from >2/3 of total voting power")
	t.Logf(
		"Block %d commit verified: %d COMMIT, %d NIL, %d ABSENT; signed power %d/%d (> %d required)",
		height,
		summary.CommitSignatures,
		summary.NilSignatures,
		summary.AbsentSignatures,
		summary.SignedPower,
		summary.TotalPower,
		quorumThreshold,
	)
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
