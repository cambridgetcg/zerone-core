//go:build integration

package multivalidator_test

import (
	"testing"
	"time"

	"github.com/cometbft/cometbft/crypto/tmhash"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	cmttypes "github.com/cometbft/cometbft/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeSignedCommitFixture(
	t *testing.T,
	powers []int64,
	flags []cmttypes.BlockIDFlag,
) (cmttypes.SignedHeader, *cmttypes.ValidatorSet) {
	t.Helper()
	require.Len(t, flags, len(powers))

	const (
		chainID = "commit-verification-test"
		height  = int64(11)
	)
	now := time.Date(2026, time.September, 2, 12, 0, 0, 0, time.UTC)

	validators := make([]*cmttypes.Validator, 0, len(powers))
	signersByAddress := make(map[string]cmttypes.MockPV, len(powers))
	for _, power := range powers {
		signer := cmttypes.NewMockPV()
		pubKey, err := signer.GetPubKey()
		require.NoError(t, err)
		validators = append(validators, cmttypes.NewValidator(pubKey, power))
		signersByAddress[string(pubKey.Address())] = signer
	}
	validatorSet := cmttypes.NewValidatorSet(validators)

	block := cmttypes.MakeBlock(height, nil, nil, nil)
	block.Header.ChainID = chainID
	block.Header.Time = now
	block.Header.ValidatorsHash = validatorSet.Hash()
	block.Header.NextValidatorsHash = validatorSet.Hash()
	block.Header.ProposerAddress = validatorSet.GetProposer().Address
	blockID := cmttypes.BlockID{
		Hash: block.Header.Hash(),
		PartSetHeader: cmttypes.PartSetHeader{
			Total: 1,
			Hash:  tmhash.Sum([]byte("commit-verification-test-parts")),
		},
	}

	signatures := make([]cmttypes.CommitSig, validatorSet.Size())
	for i, flag := range flags {
		if flag == cmttypes.BlockIDFlagAbsent {
			signatures[i] = cmttypes.NewCommitSigAbsent()
			continue
		}

		voteBlockID := blockID
		if flag == cmttypes.BlockIDFlagNil {
			voteBlockID = cmttypes.BlockID{}
		} else {
			require.Equal(t, cmttypes.BlockIDFlagCommit, flag, "unsupported fixture flag")
		}

		validator := validatorSet.Validators[i]
		vote := &cmttypes.Vote{
			ValidatorAddress: validator.Address,
			ValidatorIndex:   int32(i),
			Height:           height,
			Round:            0,
			Type:             cmtproto.PrecommitType,
			BlockID:          voteBlockID,
			Timestamp:        now,
		}
		protoVote := vote.ToProto()
		signer, ok := signersByAddress[string(validator.Address)]
		require.True(t, ok, "missing signer for validator %X", validator.Address)
		require.NoError(t, signer.SignVote(chainID, protoVote))
		vote.Signature = protoVote.Signature
		signatures[i] = vote.CommitSig()
	}

	return cmttypes.SignedHeader{
		Header: &block.Header,
		Commit: &cmttypes.Commit{
			Height:     height,
			Round:      0,
			BlockID:    blockID,
			Signatures: signatures,
		},
	}, validatorSet
}

func TestCommitVerificationCountsOnlyBlockVotes(t *testing.T) {
	signedHeader, validatorSet := makeSignedCommitFixture(
		t,
		[]int64{90, 4, 3, 3},
		[]cmttypes.BlockIDFlag{
			cmttypes.BlockIDFlagCommit,
			cmttypes.BlockIDFlagNil,
			cmttypes.BlockIDFlagAbsent,
			cmttypes.BlockIDFlagAbsent,
		},
	)

	summary, err := verifyCommitSignatures(signedHeader.ChainID, signedHeader, validatorSet)
	require.NoError(t, err)
	assert.Equal(t, 1, summary.CommitSignatures)
	assert.Equal(t, 1, summary.NilSignatures)
	assert.Equal(t, 2, summary.AbsentSignatures)
	assert.Equal(t, int64(90), summary.SignedPower)
	assert.Equal(t, int64(100), summary.TotalPower)
	assert.Greater(t, summary.SignedPower, summary.TotalPower*2/3)
}

func TestCommitVerificationRejectsSignatureCountWithoutQuorumPower(t *testing.T) {
	signedHeader, validatorSet := makeSignedCommitFixture(
		t,
		[]int64{90, 4, 3, 3},
		[]cmttypes.BlockIDFlag{
			cmttypes.BlockIDFlagAbsent,
			cmttypes.BlockIDFlagCommit,
			cmttypes.BlockIDFlagCommit,
			cmttypes.BlockIDFlagCommit,
		},
	)

	summary, err := verifyCommitSignatures(signedHeader.ChainID, signedHeader, validatorSet)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "insufficient voting power")
	assert.Equal(t, 3, summary.CommitSignatures)
	assert.Equal(t, 1, summary.AbsentSignatures)
	assert.Equal(t, int64(10), summary.SignedPower)
	assert.Equal(t, int64(100), summary.TotalPower)
}

func TestCommitVerificationRejectsInvalidSignature(t *testing.T) {
	signedHeader, validatorSet := makeSignedCommitFixture(
		t,
		[]int64{90, 4, 3, 3},
		[]cmttypes.BlockIDFlag{
			cmttypes.BlockIDFlagCommit,
			cmttypes.BlockIDFlagNil,
			cmttypes.BlockIDFlagAbsent,
			cmttypes.BlockIDFlagAbsent,
		},
	)
	signedHeader.Commit.Signatures[0].Signature[0] ^= 0xff

	_, err := verifyCommitSignatures(signedHeader.ChainID, signedHeader, validatorSet)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cryptographic commit verification failed")
}

func TestConfiguredTestEndpoints(t *testing.T) {
	t.Run("RPC override", func(t *testing.T) {
		t.Setenv(rpcAddrEnv, "http://127.0.0.1:28657")
		addr, err := rpcTestAddr()
		require.NoError(t, err)
		assert.Equal(t, "http://127.0.0.1:28657", addr)
	})

	t.Run("gRPC override", func(t *testing.T) {
		t.Setenv(grpcAddrEnv, "[::1]:29090")
		addr, err := grpcTestAddr()
		require.NoError(t, err)
		assert.Equal(t, "[::1]:29090", addr)
	})

	for name, addr := range map[string]string{
		"empty":        "",
		"missing port": "http://127.0.0.1",
		"wrong scheme": "ftp://127.0.0.1:26657",
		"path":         "http://127.0.0.1:26657/status",
		"whitespace":   " http://127.0.0.1:26657",
	} {
		t.Run("invalid RPC "+name, func(t *testing.T) {
			t.Setenv(rpcAddrEnv, addr)
			_, err := rpcTestAddr()
			require.Error(t, err)
		})
	}

	for name, addr := range map[string]string{
		"empty":        "",
		"missing port": "127.0.0.1",
		"scheme":       "http://127.0.0.1:9090",
		"zero port":    "127.0.0.1:0",
		"whitespace":   "127.0.0.1:9090 ",
	} {
		t.Run("invalid gRPC "+name, func(t *testing.T) {
			t.Setenv(grpcAddrEnv, addr)
			_, err := grpcTestAddr()
			require.Error(t, err)
		})
	}
}
