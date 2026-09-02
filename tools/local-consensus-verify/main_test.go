package main

import (
	"bytes"
	"testing"
	"time"

	cmtversion "github.com/cometbft/cometbft/proto/tendermint/version"
	cmttypes "github.com/cometbft/cometbft/types"
	"github.com/cometbft/cometbft/version"
	"github.com/stretchr/testify/require"
)

func TestVerifyBlockHashBindingRejectsMutatedRPCBlock(t *testing.T) {
	hash := func(value byte) []byte { return bytes.Repeat([]byte{value}, 32) }
	block := &cmttypes.Block{Header: cmttypes.Header{
		Version:            cmtversion.Consensus{Block: version.BlockProtocol, App: 1},
		ChainID:            "binding-test",
		Height:             7,
		Time:               time.Unix(1, 0).UTC(),
		LastBlockID:        cmttypes.BlockID{Hash: hash(0x01), PartSetHeader: cmttypes.PartSetHeader{Total: 1, Hash: hash(0x02)}},
		LastCommitHash:     hash(0x03),
		DataHash:           hash(0x04),
		ValidatorsHash:     hash(0x05),
		NextValidatorsHash: hash(0x06),
		ConsensusHash:      hash(0x07),
		AppHash:            hash(0x08),
		LastResultsHash:    hash(0x09),
		EvidenceHash:       hash(0x0a),
		ProposerAddress:    bytes.Repeat([]byte{0x0b}, 20),
	}, LastCommit: &cmttypes.Commit{}}
	blockID := cmttypes.BlockID{Hash: bytes.Clone(block.Header.Hash())}
	require.NoError(t, verifyBlockHashBinding(blockID, block))

	mutatedBlock := &cmttypes.Block{Header: block.Header, LastCommit: &cmttypes.Commit{}}
	mutatedBlock.Header.AppHash = bytes.Repeat([]byte{0x22}, 32)
	err := verifyBlockHashBinding(blockID, mutatedBlock)
	require.ErrorContains(t, err, "does not match RPC block ID")
}

func TestVerifyBlockHashBindingRejectsNilBlock(t *testing.T) {
	err := verifyBlockHashBinding(cmttypes.BlockID{}, nil)
	require.ErrorContains(t, err, "block is nil")
}
