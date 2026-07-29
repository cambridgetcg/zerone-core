package cmd

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/cometbft/cometbft/crypto/ed25519"
	"github.com/cometbft/cometbft/crypto/tmhash"
	cmtjson "github.com/cometbft/cometbft/libs/json"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	coretypes "github.com/cometbft/cometbft/rpc/core/types"
	cmttypes "github.com/cometbft/cometbft/types"
)

const (
	frozenTestChainID       = "zerone-1"
	frozenTestTrustedHeight = int64(8)
	frozenTestF             = int64(10)
	frozenTestA             = int64(11)
	frozenTestH             = int64(12)
)

type frozenTerminalTestFixture struct {
	opts frozenTerminalOptions

	valSet        *cmttypes.ValidatorSet
	privByAddress map[string]ed25519.PrivKey

	trustedBlockResult coretypes.ResultBlock
	trustedCommit      *coretypes.ResultCommit
	anchorBlockResult  coretypes.ResultBlock
	anchorCommit       *coretypes.ResultCommit
	haltBlockResult    coretypes.ResultBlock
	haltCommit         *coretypes.ResultCommit
}

func TestVerifyFrozenTerminalCommand(t *testing.T) {
	fixture := newFrozenTerminalTestFixture(t)
	cmd := verifyFrozenTerminalCmd()
	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs(fixture.commandArgs())

	if err := cmd.Execute(); err != nil {
		t.Fatalf("verify-frozen-terminal failed: %v", err)
	}
	if got, want := output.String(), "frozen-terminal-crypto: MATCH\n"; got != want {
		t.Fatalf("unexpected command output %q, want %q", got, want)
	}
}

func TestVerifyFrozenTerminalThroughRootCommand(t *testing.T) {
	fixture := newFrozenTerminalTestFixture(t)
	t.Setenv("HOME", t.TempDir())
	root := NewRootCmd()
	var output bytes.Buffer
	root.SetOut(&output)
	root.SetErr(&bytes.Buffer{})
	root.SetArgs(append([]string{"verify-frozen-terminal"}, fixture.commandArgs()...))

	if err := root.Execute(); err != nil {
		t.Fatalf("root verify-frozen-terminal failed: %v", err)
	}
	if got, want := output.String(), "frozen-terminal-crypto: MATCH\n"; got != want {
		t.Fatalf("unexpected root command output %q, want %q", got, want)
	}
}

func TestVerifyFrozenTerminalRejectsCryptographicAndLinkageTampering(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*testing.T, *frozenTerminalTestFixture)
		wantErr string
	}{
		{
			name: "forged signature",
			mutate: func(t *testing.T, fixture *frozenTerminalTestFixture) {
				fixture.anchorCommit.Commit.Signatures[0].Signature[0] ^= 0xff
				fixture.writeRPCResult(t, fixture.opts.anchorCommitPath, fixture.anchorCommit)
			},
			wantErr: "cryptographically verify >2/3 commit power",
		},
		{
			name: "wrong block hash",
			mutate: func(t *testing.T, fixture *frozenTerminalTestFixture) {
				fixture.anchorBlockResult.BlockID.Hash = tmhash.Sum([]byte("wrong-anchor-block"))
				fixture.writeRPCResult(t, fixture.opts.anchorBlockPath, &fixture.anchorBlockResult)
			},
			wantErr: "does not match RPC block ID",
		},
		{
			name: "less than two thirds commit power",
			mutate: func(t *testing.T, fixture *frozenTerminalTestFixture) {
				fixture.anchorCommit.Commit.Signatures[2] = cmttypes.NewCommitSigAbsent()
				fixture.anchorCommit.Commit.Signatures[3] = cmttypes.NewCommitSigAbsent()
				fixture.writeRPCResult(t, fixture.opts.anchorCommitPath, fixture.anchorCommit)
			},
			wantErr: "insufficient voting power",
		},
		{
			name: "bad expected anchor",
			mutate: func(_ *testing.T, fixture *frozenTerminalTestFixture) {
				fixture.opts.expectedAnchorBlockHash = strings.Repeat("0", sha256.Size*2)
			},
			wantErr: "does not match expected",
		},
		{
			name: "header drift between endpoints",
			mutate: func(t *testing.T, fixture *frozenTerminalTestFixture) {
				driftedHeader := fixture.anchorBlockResult.Block.Header
				driftedHeader.Time = driftedHeader.Time.Add(time.Millisecond)
				driftedID := fixture.anchorBlockResult.BlockID
				driftedID.Hash = driftedHeader.Hash()
				driftedCommit := signFrozenTestCommit(
					t,
					frozenTestChainID,
					frozenTestA,
					driftedID,
					fixture.valSet,
					fixture.privByAddress,
					fixture.valSet.Size(),
					driftedHeader.Time,
				)
				result := coretypes.NewResultCommit(&driftedHeader, driftedCommit, true)
				fixture.writeRPCResult(t, fixture.opts.anchorCommitPath, result)
			},
			wantErr: "headers are not exactly equal",
		},
		{
			name: "wrong RELEASE trusted block anchor",
			mutate: func(_ *testing.T, fixture *frozenTerminalTestFixture) {
				fixture.opts.expectedTrustedBlockHash = strings.Repeat("F", sha256.Size*2)
			},
			wantErr: "does not match RELEASE-pinned hash",
		},
		{
			name: "reversed header chronology",
			mutate: func(t *testing.T, fixture *frozenTerminalTestFixture) {
				header := fixture.trustedBlockResult.Block.Header
				header.Time = fixture.anchorBlockResult.Block.Header.Time.Add(time.Second)
				block := rebuildFrozenTestBlock(t, fixture.trustedBlockResult.Block, header)
				blockID := frozenTestBlockID(t, block)
				commit := signFrozenTestCommit(
					t,
					frozenTestChainID,
					frozenTestTrustedHeight,
					blockID,
					fixture.valSet,
					fixture.privByAddress,
					fixture.valSet.Size(),
					header.Time,
				)
				fixture.trustedBlockResult = coretypes.ResultBlock{BlockID: blockID, Block: block}
				fixture.trustedCommit = coretypes.NewResultCommit(&block.Header, commit, true)
				fixture.opts.expectedTrustedBlockHash = strings.ToUpper(hex.EncodeToString(blockID.Hash))
				fixture.writeRPCResult(t, fixture.opts.trustedBlockPath, &fixture.trustedBlockResult)
				fixture.writeRPCResult(t, fixture.opts.trustedCommitPath, fixture.trustedCommit)
			},
			wantErr: "not strictly ordered trusted<A<H",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fixture := newFrozenTerminalTestFixture(t)
			test.mutate(t, fixture)
			err := verifyFrozenTerminal(&fixture.opts)
			if err == nil || !strings.Contains(err.Error(), test.wantErr) {
				t.Fatalf("expected error containing %q, got %v", test.wantErr, err)
			}
		})
	}
}

func TestVerifyFrozenTerminalRejectsGenesisMismatchAndDuplicates(t *testing.T) {
	t.Run("semantic hash mismatch", func(t *testing.T) {
		fixture := newFrozenTerminalTestFixture(t)
		fixture.opts.expectedRPCGenesisSHA256 = strings.Repeat("0", sha256.Size*2)
		err := verifyFrozenTerminal(&fixture.opts)
		if err == nil || !strings.Contains(err.Error(), "semantic SHA-256") {
			t.Fatalf("expected semantic genesis hash rejection, got %v", err)
		}
	})

	t.Run("duplicate nested key", func(t *testing.T) {
		fixture := newFrozenTerminalTestFixture(t)
		raw := []byte(`{"jsonrpc":"2.0","id":1,"result":{"genesis":{"chain_id":"zerone-1","chain_id":"zerone-1"}}}`)
		if err := os.WriteFile(fixture.opts.genesisPath, raw, 0o600); err != nil {
			t.Fatal(err)
		}
		err := verifyFrozenTerminal(&fixture.opts)
		if err == nil || !strings.Contains(err.Error(), `duplicate JSON object key "chain_id"`) {
			t.Fatalf("expected duplicate genesis key rejection, got %v", err)
		}
	})
}

func newFrozenTerminalTestFixture(t *testing.T) *frozenTerminalTestFixture {
	t.Helper()
	dir := t.TempDir()
	fixture := &frozenTerminalTestFixture{
		privByAddress: make(map[string]ed25519.PrivKey),
	}

	validators := make([]*cmttypes.Validator, 0, 4)
	for i := 0; i < 4; i++ {
		privKey := ed25519.GenPrivKeyFromSecret([]byte(fmt.Sprintf("frozen-terminal-validator-%d", i)))
		validator := cmttypes.NewValidator(privKey.PubKey(), 10)
		fixture.privByAddress[string(validator.Address)] = privKey
		validators = append(validators, validator)
	}
	fixture.valSet = cmttypes.NewValidatorSet(validators)

	trustedAppHash := tmhash.Sum([]byte("trusted-app-hash"))
	checkpointAppHash := tmhash.Sum([]byte("checkpoint-app-hash-B"))
	postAnchorAppHash := tmhash.Sum([]byte("post-anchor-app-hash-E"))
	baseTime := time.Date(2026, time.July, 13, 12, 0, 0, 0, time.UTC)

	trustedLastCommit := frozenTestDummyCommit(frozenTestTrustedHeight-1, fixture.valSet.Size())
	trustedBlock := frozenTestBlock(
		frozenTestTrustedHeight,
		trustedLastCommit.BlockID,
		trustedLastCommit,
		fixture.valSet,
		trustedAppHash,
		baseTime,
	)
	trustedBlockID := frozenTestBlockID(t, trustedBlock)
	trustedCommit := signFrozenTestCommit(
		t,
		frozenTestChainID,
		frozenTestTrustedHeight,
		trustedBlockID,
		fixture.valSet,
		fixture.privByAddress,
		fixture.valSet.Size(),
		baseTime,
	)

	anchorLastCommit := frozenTestDummyCommit(frozenTestA-1, fixture.valSet.Size())
	anchorBlock := frozenTestBlock(
		frozenTestA,
		anchorLastCommit.BlockID,
		anchorLastCommit,
		fixture.valSet,
		checkpointAppHash,
		baseTime.Add(time.Minute),
	)
	anchorBlockID := frozenTestBlockID(t, anchorBlock)
	anchorCommit := signFrozenTestCommit(
		t,
		frozenTestChainID,
		frozenTestA,
		anchorBlockID,
		fixture.valSet,
		fixture.privByAddress,
		fixture.valSet.Size(),
		anchorBlock.Header.Time,
	)

	haltBlock := frozenTestBlock(
		frozenTestH,
		anchorBlockID,
		anchorCommit,
		fixture.valSet,
		postAnchorAppHash,
		baseTime.Add(2*time.Minute),
	)
	haltBlockID := frozenTestBlockID(t, haltBlock)
	haltCommit := signFrozenTestCommit(
		t,
		frozenTestChainID,
		frozenTestH,
		haltBlockID,
		fixture.valSet,
		fixture.privByAddress,
		fixture.valSet.Size(),
		haltBlock.Header.Time,
	)

	fixture.trustedBlockResult = coretypes.ResultBlock{BlockID: trustedBlockID, Block: trustedBlock}
	fixture.trustedCommit = coretypes.NewResultCommit(&trustedBlock.Header, trustedCommit, true)
	fixture.anchorBlockResult = coretypes.ResultBlock{BlockID: anchorBlockID, Block: anchorBlock}
	fixture.anchorCommit = coretypes.NewResultCommit(&anchorBlock.Header, anchorCommit, true)
	fixture.haltBlockResult = coretypes.ResultBlock{BlockID: haltBlockID, Block: haltBlock}
	fixture.haltCommit = coretypes.NewResultCommit(&haltBlock.Header, haltCommit, false)

	genesisPath := filepath.Join(dir, "genesis.json.raw")
	trustedBlockPath := filepath.Join(dir, "trusted-block.json.raw")
	trustedCommitPath := filepath.Join(dir, "trusted-commit.json.raw")
	trustedValidatorsPath := filepath.Join(dir, "trusted-validators.json.raw")
	anchorBlockPath := filepath.Join(dir, "a-block.json.raw")
	anchorCommitPath := filepath.Join(dir, "a-commit.json.raw")
	anchorValidatorsPath := filepath.Join(dir, "a-validators.json.raw")
	anchorBlockResultsPath := filepath.Join(dir, "a-block-results.json.raw")
	haltBlockPath := filepath.Join(dir, "h-block.json.raw")
	haltCommitPath := filepath.Join(dir, "h-commit.json.raw")
	haltValidatorsPath := filepath.Join(dir, "h-validators.json.raw")

	genesis := map[string]any{
		"chain_id":       frozenTestChainID,
		"initial_height": "1",
		"app_state": map[string]any{
			"auth": map[string]any{},
			"bank": map[string]any{},
		},
	}
	canonicalGenesis, err := json.Marshal(genesis)
	if err != nil {
		t.Fatal(err)
	}
	genesisDigest := sha256.Sum256(canonicalGenesis)
	fixture.writeStandardRPCResult(t, genesisPath, map[string]any{"genesis": genesis})
	fixture.writeRPCResult(t, trustedBlockPath, &fixture.trustedBlockResult)
	fixture.writeRPCResult(t, trustedCommitPath, fixture.trustedCommit)
	fixture.writeRPCResult(t, trustedValidatorsPath, frozenTestValidatorsResult(frozenTestTrustedHeight, fixture.valSet))
	fixture.writeRPCResult(t, anchorBlockPath, &fixture.anchorBlockResult)
	fixture.writeRPCResult(t, anchorCommitPath, fixture.anchorCommit)
	fixture.writeRPCResult(t, anchorValidatorsPath, frozenTestValidatorsResult(frozenTestA, fixture.valSet))
	fixture.writeRPCResult(t, anchorBlockResultsPath, &coretypes.ResultBlockResults{
		Height:     frozenTestA,
		AppHash:    postAnchorAppHash,
		TxsResults: nil,
	})
	fixture.writeRPCResult(t, haltBlockPath, &fixture.haltBlockResult)
	fixture.writeRPCResult(t, haltCommitPath, fixture.haltCommit)
	fixture.writeRPCResult(t, haltValidatorsPath, frozenTestValidatorsResult(frozenTestH, fixture.valSet))

	fixture.opts = frozenTerminalOptions{
		genesisPath:               genesisPath,
		trustedBlockPath:          trustedBlockPath,
		trustedCommitPath:         trustedCommitPath,
		trustedValidatorsPath:     trustedValidatorsPath,
		anchorBlockPath:           anchorBlockPath,
		anchorCommitPath:          anchorCommitPath,
		anchorValidatorsPath:      anchorValidatorsPath,
		anchorBlockResultsPath:    anchorBlockResultsPath,
		haltBlockPath:             haltBlockPath,
		haltCommitPath:            haltCommitPath,
		haltValidatorsPath:        haltValidatorsPath,
		expectedChainID:           frozenTestChainID,
		trustedHeight:             frozenTestTrustedHeight,
		checkpointStateHeight:     frozenTestF,
		finalCommittedHeight:      frozenTestA,
		haltTriggerHeight:         frozenTestH,
		expectedTrustedBlockHash:  strings.ToUpper(hex.EncodeToString(trustedBlockID.Hash)),
		expectedTrustedAppHash:    strings.ToUpper(hex.EncodeToString(trustedAppHash)),
		expectedCheckpointAppHash: strings.ToUpper(hex.EncodeToString(checkpointAppHash)),
		expectedAnchorBlockHash:   strings.ToUpper(hex.EncodeToString(anchorBlockID.Hash)),
		expectedHaltBlockHash:     strings.ToUpper(hex.EncodeToString(haltBlockID.Hash)),
		expectedPostAnchorAppHash: strings.ToUpper(hex.EncodeToString(postAnchorAppHash)),
		expectedRPCGenesisSHA256:  hex.EncodeToString(genesisDigest[:]),
	}

	return fixture
}

func frozenTestBlock(
	height int64,
	lastBlockID cmttypes.BlockID,
	lastCommit *cmttypes.Commit,
	valSet *cmttypes.ValidatorSet,
	appHash []byte,
	blockTime time.Time,
) *cmttypes.Block {
	block := cmttypes.MakeBlock(height, nil, lastCommit, nil)
	block.Header.ChainID = frozenTestChainID
	block.Header.Time = blockTime
	block.Header.LastBlockID = lastBlockID
	block.Header.ValidatorsHash = valSet.Hash()
	block.Header.NextValidatorsHash = valSet.Hash()
	block.Header.ConsensusHash = tmhash.Sum([]byte(fmt.Sprintf("consensus-%d", height)))
	block.Header.AppHash = append([]byte(nil), appHash...)
	block.Header.LastResultsHash = tmhash.Sum([]byte(fmt.Sprintf("results-%d", height-1)))
	block.Header.ProposerAddress = append([]byte(nil), valSet.GetProposer().Address...)
	return block
}

func rebuildFrozenTestBlock(t *testing.T, original *cmttypes.Block, header cmttypes.Header) *cmttypes.Block {
	t.Helper()
	block := cmttypes.MakeBlock(original.Height, original.Data.Txs, original.LastCommit, original.Evidence.Evidence)
	block.Header = header
	if err := block.ValidateBasic(); err != nil {
		t.Fatalf("rebuilt test block is invalid: %v", err)
	}
	return block
}

func frozenTestBlockID(t *testing.T, block *cmttypes.Block) cmttypes.BlockID {
	t.Helper()
	parts, err := block.MakePartSet(cmttypes.BlockPartSizeBytes)
	if err != nil {
		t.Fatal(err)
	}
	return cmttypes.BlockID{
		Hash:          append([]byte(nil), block.Hash()...),
		PartSetHeader: parts.Header(),
	}
}

func frozenTestDummyCommit(height int64, validatorCount int) *cmttypes.Commit {
	signatures := make([]cmttypes.CommitSig, validatorCount)
	for i := range signatures {
		signatures[i] = cmttypes.NewCommitSigAbsent()
	}
	return &cmttypes.Commit{
		Height:     height,
		Round:      0,
		BlockID:    frozenTestSyntheticBlockID(height),
		Signatures: signatures,
	}
}

func frozenTestSyntheticBlockID(height int64) cmttypes.BlockID {
	return cmttypes.BlockID{
		Hash: tmhash.Sum([]byte(fmt.Sprintf("synthetic-block-%d", height))),
		PartSetHeader: cmttypes.PartSetHeader{
			Total: 1,
			Hash:  tmhash.Sum([]byte(fmt.Sprintf("synthetic-parts-%d", height))),
		},
	}
}

func signFrozenTestCommit(
	t *testing.T,
	chainID string,
	height int64,
	blockID cmttypes.BlockID,
	valSet *cmttypes.ValidatorSet,
	privByAddress map[string]ed25519.PrivKey,
	signedCount int,
	commitTime time.Time,
) *cmttypes.Commit {
	t.Helper()
	signatures := make([]cmttypes.CommitSig, valSet.Size())
	for i, validator := range valSet.Validators {
		if i >= signedCount {
			signatures[i] = cmttypes.NewCommitSigAbsent()
			continue
		}
		vote := &cmttypes.Vote{
			Type:             cmtproto.PrecommitType,
			Height:           height,
			Round:            0,
			BlockID:          blockID,
			Timestamp:        commitTime.Add(time.Duration(i) * time.Nanosecond),
			ValidatorAddress: append([]byte(nil), validator.Address...),
			ValidatorIndex:   int32(i),
		}
		privKey, ok := privByAddress[string(validator.Address)]
		if !ok {
			t.Fatalf("missing private key for validator %X", validator.Address)
		}
		signature, err := privKey.Sign(cmttypes.VoteSignBytes(chainID, vote.ToProto()))
		if err != nil {
			t.Fatal(err)
		}
		vote.Signature = signature
		signatures[i] = vote.CommitSig()
	}
	return &cmttypes.Commit{
		Height:     height,
		Round:      0,
		BlockID:    blockID,
		Signatures: signatures,
	}
}

func frozenTestValidatorsResult(height int64, valSet *cmttypes.ValidatorSet) *coretypes.ResultValidators {
	validators := make([]*cmttypes.Validator, len(valSet.Validators))
	for i, validator := range valSet.Validators {
		validators[i] = validator.Copy()
	}
	return &coretypes.ResultValidators{
		BlockHeight: height,
		Validators:  validators,
		Count:       len(validators),
		Total:       len(validators),
	}
}

func (fixture *frozenTerminalTestFixture) writeRPCResult(t *testing.T, path string, result any) {
	t.Helper()
	resultJSON, err := cmtjson.Marshal(result)
	if err != nil {
		t.Fatalf("marshal RPC result for %s: %v", path, err)
	}
	raw := make([]byte, 0, len(resultJSON)+64)
	raw = append(raw, `{"jsonrpc":"2.0","id":1,"result":`...)
	raw = append(raw, resultJSON...)
	raw = append(raw, '}', '\n')
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
}

func (fixture *frozenTerminalTestFixture) writeStandardRPCResult(t *testing.T, path string, result any) {
	t.Helper()
	resultJSON, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal standard RPC result for %s: %v", path, err)
	}
	raw := make([]byte, 0, len(resultJSON)+64)
	raw = append(raw, `{"jsonrpc":"2.0","id":1,"result":`...)
	raw = append(raw, resultJSON...)
	raw = append(raw, '}', '\n')
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
}

func (fixture *frozenTerminalTestFixture) commandArgs() []string {
	return []string{
		"--genesis", fixture.opts.genesisPath,
		"--trusted-block", fixture.opts.trustedBlockPath,
		"--trusted-commit", fixture.opts.trustedCommitPath,
		"--trusted-validators", fixture.opts.trustedValidatorsPath,
		"--a-block", fixture.opts.anchorBlockPath,
		"--a-commit", fixture.opts.anchorCommitPath,
		"--a-validators", fixture.opts.anchorValidatorsPath,
		"--a-block-results", fixture.opts.anchorBlockResultsPath,
		"--h-block", fixture.opts.haltBlockPath,
		"--h-commit", fixture.opts.haltCommitPath,
		"--h-validators", fixture.opts.haltValidatorsPath,
		"--expected-chain-id", fixture.opts.expectedChainID,
		"--trusted-height", strconv.FormatInt(fixture.opts.trustedHeight, 10),
		"--checkpoint-state-height", strconv.FormatInt(fixture.opts.checkpointStateHeight, 10),
		"--final-committed-height", strconv.FormatInt(fixture.opts.finalCommittedHeight, 10),
		"--halt-trigger-height", strconv.FormatInt(fixture.opts.haltTriggerHeight, 10),
		"--expected-trusted-block-hash", fixture.opts.expectedTrustedBlockHash,
		"--expected-trusted-app-hash", fixture.opts.expectedTrustedAppHash,
		"--expected-checkpoint-app-hash", fixture.opts.expectedCheckpointAppHash,
		"--expected-anchor-block-hash", fixture.opts.expectedAnchorBlockHash,
		"--expected-halt-trigger-block-hash", fixture.opts.expectedHaltBlockHash,
		"--expected-post-anchor-app-hash", fixture.opts.expectedPostAnchorAppHash,
		"--expected-rpc-genesis-sha256", fixture.opts.expectedRPCGenesisSHA256,
	}
}
