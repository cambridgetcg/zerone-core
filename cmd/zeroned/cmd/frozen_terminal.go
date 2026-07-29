package cmd

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	cmtjson "github.com/cometbft/cometbft/libs/json"
	cmtmath "github.com/cometbft/cometbft/libs/math"
	coretypes "github.com/cometbft/cometbft/rpc/core/types"
	cmttypes "github.com/cometbft/cometbft/types"
	"github.com/cosmos/gogoproto/proto"
	"github.com/spf13/cobra"
)

const (
	frozenTerminalMaxRPCFileSize = 512 << 20
	frozenTerminalMaxHeight      = int64(^uint64(0) >> 1)
)

type frozenTerminalOptions struct {
	genesisPath               string
	trustedBlockPath          string
	trustedCommitPath         string
	trustedValidatorsPath     string
	anchorBlockPath           string
	anchorCommitPath          string
	anchorValidatorsPath      string
	anchorBlockResultsPath    string
	haltBlockPath             string
	haltCommitPath            string
	haltValidatorsPath        string
	expectedChainID           string
	trustedHeight             int64
	checkpointStateHeight     int64
	finalCommittedHeight      int64
	haltTriggerHeight         int64
	expectedTrustedBlockHash  string
	expectedTrustedAppHash    string
	expectedCheckpointAppHash string
	expectedAnchorBlockHash   string
	expectedHaltBlockHash     string
	expectedPostAnchorAppHash string
	expectedRPCGenesisSHA256  string
}

type frozenTerminalHeightEvidence struct {
	block      coretypes.ResultBlock
	commit     coretypes.ResultCommit
	validators coretypes.ResultValidators
	valSet     *cmttypes.ValidatorSet
}

func verifyFrozenTerminalCmd() *cobra.Command {
	opts := new(frozenTerminalOptions)
	cmd := &cobra.Command{
		Use:           "verify-frozen-terminal",
		Short:         "Cryptographically verify frozen zerone-1 terminal RPC evidence offline",
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		// Do not run the daemon root's config interception for this file-only
		// verifier. It must neither need nor mutate a node home.
		PersistentPreRunE: func(_ *cobra.Command, _ []string) error { return nil },
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := verifyFrozenTerminal(opts); err != nil {
				return err
			}
			_, err := fmt.Fprintln(cmd.OutOrStdout(), "frozen-terminal-crypto: MATCH")
			return err
		},
	}

	flags := cmd.Flags()
	flags.StringVar(&opts.genesisPath, "genesis", "", "raw /genesis JSON-RPC response")
	flags.StringVar(&opts.trustedBlockPath, "trusted-block", "", "raw trusted /block JSON-RPC response")
	flags.StringVar(&opts.trustedCommitPath, "trusted-commit", "", "raw trusted /commit JSON-RPC response")
	flags.StringVar(&opts.trustedValidatorsPath, "trusted-validators", "", "raw trusted full /validators JSON-RPC response")
	flags.StringVar(&opts.anchorBlockPath, "a-block", "", "raw anchor A /block JSON-RPC response")
	flags.StringVar(&opts.anchorCommitPath, "a-commit", "", "raw anchor A /commit JSON-RPC response")
	flags.StringVar(&opts.anchorValidatorsPath, "a-validators", "", "raw anchor A full /validators JSON-RPC response")
	flags.StringVar(&opts.anchorBlockResultsPath, "a-block-results", "", "raw anchor A /block_results JSON-RPC response")
	flags.StringVar(&opts.haltBlockPath, "h-block", "", "raw halt trigger H /block JSON-RPC response")
	flags.StringVar(&opts.haltCommitPath, "h-commit", "", "raw halt trigger H /commit JSON-RPC response")
	flags.StringVar(&opts.haltValidatorsPath, "h-validators", "", "raw halt trigger H full /validators JSON-RPC response")
	flags.StringVar(&opts.expectedChainID, "expected-chain-id", "", "expected predecessor chain ID")
	flags.Int64Var(&opts.trustedHeight, "trusted-height", 0, "RELEASE-pinned trusted block height")
	flags.Int64Var(&opts.checkpointStateHeight, "checkpoint-state-height", 0, "checkpoint state height F")
	flags.Int64Var(&opts.finalCommittedHeight, "final-committed-height", 0, "empty canonical anchor height A=F+1")
	flags.Int64Var(&opts.haltTriggerHeight, "halt-trigger-height", 0, "staged halt trigger height H=A+1")
	flags.StringVar(&opts.expectedTrustedBlockHash, "expected-trusted-block-hash", "", "RELEASE-pinned trusted block hash")
	flags.StringVar(&opts.expectedTrustedAppHash, "expected-trusted-app-hash", "", "RELEASE-pinned trusted block AppHash")
	flags.StringVar(&opts.expectedCheckpointAppHash, "expected-checkpoint-app-hash", "", "expected checkpoint-F AppHash B")
	flags.StringVar(&opts.expectedAnchorBlockHash, "expected-anchor-block-hash", "", "expected anchor-A block hash")
	flags.StringVar(&opts.expectedHaltBlockHash, "expected-halt-trigger-block-hash", "", "expected staged-H block hash D")
	flags.StringVar(&opts.expectedPostAnchorAppHash, "expected-post-anchor-app-hash", "", "expected post-anchor-A AppHash E")
	flags.StringVar(&opts.expectedRPCGenesisSHA256, "expected-rpc-genesis-sha256", "", "expected semantic SHA-256 of result.genesis")

	required := []string{
		"genesis",
		"trusted-block", "trusted-commit", "trusted-validators",
		"a-block", "a-commit", "a-validators", "a-block-results",
		"h-block", "h-commit", "h-validators",
		"expected-chain-id",
		"trusted-height", "expected-trusted-block-hash", "expected-trusted-app-hash",
		"checkpoint-state-height", "final-committed-height", "halt-trigger-height",
		"expected-checkpoint-app-hash", "expected-anchor-block-hash",
		"expected-halt-trigger-block-hash", "expected-post-anchor-app-hash",
		"expected-rpc-genesis-sha256",
	}
	for _, name := range required {
		if err := cmd.MarkFlagRequired(name); err != nil {
			panic(err)
		}
	}

	return cmd
}

func verifyFrozenTerminal(opts *frozenTerminalOptions) error {
	if opts == nil {
		return errors.New("missing frozen terminal options")
	}
	if opts.expectedChainID == "" {
		return errors.New("expected chain ID must not be empty")
	}
	if opts.trustedHeight <= 0 || opts.checkpointStateHeight <= 0 || opts.finalCommittedHeight <= 0 || opts.haltTriggerHeight <= 0 {
		return errors.New("trusted height, F, A, and H must be positive heights")
	}
	if opts.checkpointStateHeight == frozenTerminalMaxHeight {
		return errors.New("invalid terminal heights: F has no representable successor A")
	}
	expectedAnchorHeight := opts.checkpointStateHeight + 1
	if opts.finalCommittedHeight != expectedAnchorHeight {
		return fmt.Errorf("invalid terminal heights: A=%d must equal F+1=%d", opts.finalCommittedHeight, expectedAnchorHeight)
	}
	if opts.finalCommittedHeight == frozenTerminalMaxHeight {
		return errors.New("invalid terminal heights: A has no representable successor H")
	}
	expectedHaltHeight := opts.finalCommittedHeight + 1
	if opts.haltTriggerHeight != expectedHaltHeight {
		return fmt.Errorf("invalid terminal heights: H=%d must equal A+1=%d", opts.haltTriggerHeight, expectedHaltHeight)
	}

	checkpointAppHash, err := decodeFrozenUpperHash("expected checkpoint AppHash B", opts.expectedCheckpointAppHash)
	if err != nil {
		return err
	}
	anchorBlockHash, err := decodeFrozenUpperHash("expected anchor block hash A", opts.expectedAnchorBlockHash)
	if err != nil {
		return err
	}
	haltBlockHash, err := decodeFrozenUpperHash("expected halt trigger block hash D", opts.expectedHaltBlockHash)
	if err != nil {
		return err
	}
	postAnchorAppHash, err := decodeFrozenUpperHash("expected post-anchor AppHash E", opts.expectedPostAnchorAppHash)
	if err != nil {
		return err
	}
	trustedBlockHash, err := decodeFrozenUpperHash("expected trusted block hash", opts.expectedTrustedBlockHash)
	if err != nil {
		return err
	}
	trustedAppHash, err := decodeFrozenUpperHash("expected trusted AppHash", opts.expectedTrustedAppHash)
	if err != nil {
		return err
	}

	if err := verifyFrozenGenesis(opts.genesisPath, opts.expectedChainID, opts.expectedRPCGenesisSHA256); err != nil {
		return err
	}

	trusted, err := readFrozenHeightEvidence(
		"trusted", opts.trustedBlockPath, opts.trustedCommitPath, opts.trustedValidatorsPath,
	)
	if err != nil {
		return err
	}
	if opts.trustedHeight > opts.checkpointStateHeight {
		return fmt.Errorf("trusted height %d must not exceed F=%d", opts.trustedHeight, opts.checkpointStateHeight)
	}
	if err := validateFrozenHeightEvidence("trusted", opts.expectedChainID, opts.trustedHeight, trusted); err != nil {
		return err
	}
	if !bytes.Equal(trusted.block.BlockID.Hash, trustedBlockHash) {
		return fmt.Errorf("trusted block hash %X does not match RELEASE-pinned hash %X", trusted.block.BlockID.Hash, trustedBlockHash)
	}
	if !bytes.Equal(trusted.block.Block.Header.AppHash, trustedAppHash) {
		return fmt.Errorf("trusted block AppHash %X does not match RELEASE-pinned AppHash %X", trusted.block.Block.Header.AppHash, trustedAppHash)
	}

	anchor, err := readFrozenHeightEvidence(
		"anchor A", opts.anchorBlockPath, opts.anchorCommitPath, opts.anchorValidatorsPath,
	)
	if err != nil {
		return err
	}
	if err := validateFrozenHeightEvidence("anchor A", opts.expectedChainID, opts.finalCommittedHeight, anchor); err != nil {
		return err
	}

	halt, err := readFrozenHeightEvidence(
		"halt trigger H", opts.haltBlockPath, opts.haltCommitPath, opts.haltValidatorsPath,
	)
	if err != nil {
		return err
	}
	if err := validateFrozenHeightEvidence("halt trigger H", opts.expectedChainID, opts.haltTriggerHeight, halt); err != nil {
		return err
	}

	if !anchor.commit.CanonicalCommit {
		return errors.New("anchor A commit is not canonical")
	}
	if halt.commit.CanonicalCommit {
		return errors.New("halt trigger H commit is canonical; expected the subjective noncanonical tip commit")
	}
	if len(anchor.block.Block.Data.Txs) != 0 {
		return fmt.Errorf("anchor A contains %d transactions", len(anchor.block.Block.Data.Txs))
	}
	if len(halt.block.Block.Data.Txs) != 0 {
		return fmt.Errorf("halt trigger H contains %d transactions", len(halt.block.Block.Data.Txs))
	}

	if !bytes.Equal(anchor.block.BlockID.Hash, anchorBlockHash) {
		return fmt.Errorf("anchor A block hash %X does not match expected %X", anchor.block.BlockID.Hash, anchorBlockHash)
	}
	if !bytes.Equal(halt.block.BlockID.Hash, haltBlockHash) {
		return fmt.Errorf("halt trigger H block hash %X does not match expected %X", halt.block.BlockID.Hash, haltBlockHash)
	}
	if !bytes.Equal(anchor.block.Block.Header.AppHash, checkpointAppHash) {
		return fmt.Errorf("anchor A header AppHash %X does not match checkpoint AppHash B %X", anchor.block.Block.Header.AppHash, checkpointAppHash)
	}
	if !bytes.Equal(halt.block.Block.Header.AppHash, postAnchorAppHash) {
		return fmt.Errorf("halt trigger H header AppHash %X does not match post-anchor AppHash E %X", halt.block.Block.Header.AppHash, postAnchorAppHash)
	}

	var anchorResults coretypes.ResultBlockResults
	if _, err := decodeFrozenRPCFile(opts.anchorBlockResultsPath, "anchor A block results", &anchorResults); err != nil {
		return err
	}
	if anchorResults.Height != opts.finalCommittedHeight {
		return fmt.Errorf("anchor A block results height %d does not match expected A=%d", anchorResults.Height, opts.finalCommittedHeight)
	}
	if len(anchorResults.TxsResults) != 0 {
		return fmt.Errorf("anchor A block results contain %d transaction results", len(anchorResults.TxsResults))
	}
	if !bytes.Equal(anchorResults.AppHash, postAnchorAppHash) {
		return fmt.Errorf("anchor A results AppHash %X does not match post-anchor AppHash E %X", anchorResults.AppHash, postAnchorAppHash)
	}

	if !halt.block.Block.Header.LastBlockID.Equals(anchor.block.BlockID) {
		return fmt.Errorf("halt trigger H does not link to anchor A block ID")
	}
	if !proto.Equal(halt.block.Block.LastCommit.ToProto(), anchor.commit.Commit.ToProto()) {
		return errors.New("halt trigger H last commit is not exactly the anchor A commit")
	}
	if !bytes.Equal(anchor.block.Block.Header.NextValidatorsHash, halt.valSet.Hash()) {
		return fmt.Errorf("anchor A next validator hash %X does not match halt trigger H validator set %X", anchor.block.Block.Header.NextValidatorsHash, halt.valSet.Hash())
	}
	if !trusted.block.Block.Header.Time.Before(anchor.block.Block.Header.Time) ||
		!anchor.block.Block.Header.Time.Before(halt.block.Block.Header.Time) {
		return fmt.Errorf(
			"terminal header times are not strictly ordered trusted<A<H: %s, %s, %s",
			trusted.block.Block.Header.Time,
			anchor.block.Block.Header.Time,
			halt.block.Block.Header.Time,
		)
	}
	if err := trusted.valSet.VerifyCommitLightTrustingAllSignatures(
		opts.expectedChainID,
		anchor.commit.Commit,
		cmtmath.Fraction{Numerator: 1, Denominator: 3},
	); err != nil {
		return fmt.Errorf("trusted validator continuity to anchor A failed at 1/3: %w", err)
	}

	return nil
}

func readFrozenHeightEvidence(label, blockPath, commitPath, validatorsPath string) (*frozenTerminalHeightEvidence, error) {
	evidence := new(frozenTerminalHeightEvidence)
	if _, err := decodeFrozenRPCFile(blockPath, label+" block", &evidence.block); err != nil {
		return nil, err
	}
	commitObject, err := decodeFrozenRPCFile(commitPath, label+" commit", &evidence.commit)
	if err != nil {
		return nil, err
	}
	canonical, ok := commitObject["canonical"].(bool)
	if !ok {
		return nil, fmt.Errorf("%s commit result must contain a boolean canonical field", label)
	}
	if canonical != evidence.commit.CanonicalCommit {
		return nil, fmt.Errorf("%s commit canonical field did not decode consistently", label)
	}
	if _, err := decodeFrozenRPCFile(validatorsPath, label+" validators", &evidence.validators); err != nil {
		return nil, err
	}
	return evidence, nil
}

func validateFrozenHeightEvidence(label, chainID string, expectedHeight int64, evidence *frozenTerminalHeightEvidence) error {
	if evidence == nil || evidence.block.Block == nil {
		return fmt.Errorf("%s block result is missing its block", label)
	}
	block := evidence.block.Block
	if block.Header.Height != expectedHeight {
		return fmt.Errorf("%s block height %d does not match expected %d", label, block.Header.Height, expectedHeight)
	}
	if evidence.validators.BlockHeight != expectedHeight {
		return fmt.Errorf("%s validator height %d does not match expected %d", label, evidence.validators.BlockHeight, expectedHeight)
	}
	if evidence.validators.Count <= 0 ||
		evidence.validators.Count != evidence.validators.Total ||
		evidence.validators.Count != len(evidence.validators.Validators) {
		return fmt.Errorf(
			"%s validators are not one full non-paginated set: count=%d total=%d validators=%d",
			label, evidence.validators.Count, evidence.validators.Total, len(evidence.validators.Validators),
		)
	}

	valSet, err := frozenValidatorSet(label, evidence.validators.Validators)
	if err != nil {
		return err
	}
	evidence.valSet = valSet

	if err := block.ValidateBasic(); err != nil {
		return fmt.Errorf("%s block ValidateBasic failed: %w", label, err)
	}
	if err := evidence.block.BlockID.ValidateBasic(); err != nil {
		return fmt.Errorf("%s block ID ValidateBasic failed: %w", label, err)
	}
	if !evidence.block.BlockID.IsComplete() {
		return fmt.Errorf("%s block ID is incomplete", label)
	}
	if !bytes.Equal(block.Hash(), evidence.block.BlockID.Hash) {
		return fmt.Errorf("%s block hash %X does not match RPC block ID %X", label, block.Hash(), evidence.block.BlockID.Hash)
	}
	parts, err := block.MakePartSet(cmttypes.BlockPartSizeBytes)
	if err != nil {
		return fmt.Errorf("%s block part-set calculation failed: %w", label, err)
	}
	if !parts.Header().Equals(evidence.block.BlockID.PartSetHeader) {
		return fmt.Errorf("%s block part-set header does not match RPC block ID", label)
	}

	if err := evidence.commit.SignedHeader.ValidateBasic(chainID); err != nil {
		return fmt.Errorf("%s signed header ValidateBasic failed: %w", label, err)
	}
	if !proto.Equal(block.Header.ToProto(), evidence.commit.Header.ToProto()) {
		return fmt.Errorf("%s block and commit endpoint headers are not exactly equal", label)
	}
	if !bytes.Equal(block.Header.ValidatorsHash, valSet.Hash()) {
		return fmt.Errorf("%s header validator hash %X does not match validator set %X", label, block.Header.ValidatorsHash, valSet.Hash())
	}
	if err := valSet.VerifyCommit(chainID, evidence.block.BlockID, expectedHeight, evidence.commit.Commit); err != nil {
		return fmt.Errorf("%s validator set did not cryptographically verify >2/3 commit power: %w", label, err)
	}

	return nil
}

func frozenValidatorSet(label string, validators []*cmttypes.Validator) (*cmttypes.ValidatorSet, error) {
	copied := make([]*cmttypes.Validator, len(validators))
	seen := make(map[string]struct{}, len(validators))
	for i, validator := range validators {
		if validator == nil {
			return nil, fmt.Errorf("%s validator #%d is nil", label, i)
		}
		if err := validator.ValidateBasic(); err != nil {
			return nil, fmt.Errorf("%s validator #%d is invalid: %w", label, i, err)
		}
		if validator.VotingPower <= 0 {
			return nil, fmt.Errorf("%s validator #%d has non-positive voting power %d", label, i, validator.VotingPower)
		}
		key := string(validator.Address)
		if _, exists := seen[key]; exists {
			return nil, fmt.Errorf("%s validator set contains duplicate address %X", label, validator.Address)
		}
		seen[key] = struct{}{}
		copied[i] = validator.Copy()
	}

	valSet, err := cmttypes.ValidatorSetFromExistingValidators(copied)
	if err != nil {
		return nil, fmt.Errorf("%s validator set is invalid: %w", label, err)
	}
	if err := valSet.ValidateBasic(); err != nil {
		return nil, fmt.Errorf("%s validator set ValidateBasic failed: %w", label, err)
	}
	return valSet, nil
}

func verifyFrozenGenesis(path, expectedChainID, expectedSHA256 string) error {
	parsed, err := decodeFrozenJSONFile(path, "genesis")
	if err != nil {
		return err
	}
	envelope, ok := parsed.(map[string]any)
	if !ok {
		return errors.New("genesis JSON-RPC response must be an object")
	}
	if err := validateFrozenRPCEnvelope(envelope, "genesis"); err != nil {
		return err
	}
	result, ok := envelope["result"].(map[string]any)
	if !ok {
		return errors.New("genesis JSON-RPC result must be an object")
	}
	genesis, ok := result["genesis"].(map[string]any)
	if !ok {
		return errors.New("genesis JSON-RPC result.genesis must be an object")
	}
	chainID, ok := genesis["chain_id"].(string)
	if !ok || chainID != expectedChainID {
		return fmt.Errorf("RPC genesis chain ID %q does not match expected %q", chainID, expectedChainID)
	}
	if len(expectedSHA256) != sha256.Size*2 || strings.ToLower(expectedSHA256) != expectedSHA256 {
		return errors.New("expected RPC genesis SHA-256 must be 64 lowercase hexadecimal characters")
	}
	if _, err := hex.DecodeString(expectedSHA256); err != nil {
		return errors.New("expected RPC genesis SHA-256 must be 64 lowercase hexadecimal characters")
	}
	canonical, err := json.Marshal(genesis)
	if err != nil {
		return fmt.Errorf("canonicalize RPC genesis: %w", err)
	}
	digest := sha256.Sum256(canonical)
	actual := hex.EncodeToString(digest[:])
	if actual != expectedSHA256 {
		return fmt.Errorf("RPC genesis semantic SHA-256 %s does not match expected %s", actual, expectedSHA256)
	}
	return nil
}

func decodeFrozenRPCFile(path, label string, dst any) (map[string]any, error) {
	parsed, raw, err := readFrozenJSONFile(path, label)
	if err != nil {
		return nil, err
	}
	envelope, ok := parsed.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("%s JSON-RPC response must be an object", label)
	}
	if err := validateFrozenRPCEnvelope(envelope, label); err != nil {
		return nil, err
	}
	resultObject, ok := envelope["result"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("%s JSON-RPC result must be an object", label)
	}

	var rawEnvelope struct {
		Result json.RawMessage `json:"result"`
	}
	if err := json.Unmarshal(raw, &rawEnvelope); err != nil {
		return nil, fmt.Errorf("decode %s JSON-RPC envelope: %w", label, err)
	}
	if len(rawEnvelope.Result) == 0 || bytes.Equal(bytes.TrimSpace(rawEnvelope.Result), []byte("null")) {
		return nil, fmt.Errorf("%s JSON-RPC response has no result", label)
	}
	if err := cmtjson.Unmarshal(rawEnvelope.Result, dst); err != nil {
		return nil, fmt.Errorf("decode %s Comet result: %w", label, err)
	}
	return resultObject, nil
}

func validateFrozenRPCEnvelope(envelope map[string]any, label string) error {
	if version, _ := envelope["jsonrpc"].(string); version != "2.0" {
		return fmt.Errorf("%s JSON-RPC version is %q, expected 2.0", label, version)
	}
	if rpcError, exists := envelope["error"]; exists && rpcError != nil {
		return fmt.Errorf("%s JSON-RPC response contains an error", label)
	}
	if result, exists := envelope["result"]; !exists || result == nil {
		return fmt.Errorf("%s JSON-RPC response has no result", label)
	}
	return nil
}

func decodeFrozenJSONFile(path, label string) (any, error) {
	parsed, _, err := readFrozenJSONFile(path, label)
	return parsed, err
}

func readFrozenJSONFile(path, label string) (any, []byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, nil, fmt.Errorf("open %s file: %w", label, err)
	}
	defer file.Close()

	raw, err := io.ReadAll(io.LimitReader(file, frozenTerminalMaxRPCFileSize+1))
	if err != nil {
		return nil, nil, fmt.Errorf("read %s file: %w", label, err)
	}
	if len(raw) > frozenTerminalMaxRPCFileSize {
		return nil, nil, fmt.Errorf("%s file exceeds %d bytes", label, frozenTerminalMaxRPCFileSize)
	}
	parsed, err := decodeFrozenUniqueJSON(raw)
	if err != nil {
		return nil, nil, fmt.Errorf("decode duplicate-free %s JSON: %w", label, err)
	}
	return parsed, raw, nil
}

func decodeFrozenUniqueJSON(raw []byte) (any, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	value, err := decodeFrozenUniqueJSONValue(decoder)
	if err != nil {
		return nil, err
	}
	if _, err := decoder.Token(); err != io.EOF {
		if err == nil {
			return nil, errors.New("multiple JSON values")
		}
		return nil, err
	}
	return value, nil
}

func decodeFrozenUniqueJSONValue(decoder *json.Decoder) (any, error) {
	token, err := decoder.Token()
	if err != nil {
		return nil, err
	}
	delimiter, isDelimiter := token.(json.Delim)
	if !isDelimiter {
		return token, nil
	}

	switch delimiter {
	case '{':
		object := make(map[string]any)
		for decoder.More() {
			keyToken, err := decoder.Token()
			if err != nil {
				return nil, err
			}
			key, ok := keyToken.(string)
			if !ok {
				return nil, errors.New("JSON object key is not a string")
			}
			if _, exists := object[key]; exists {
				return nil, fmt.Errorf("duplicate JSON object key %q", key)
			}
			value, err := decodeFrozenUniqueJSONValue(decoder)
			if err != nil {
				return nil, err
			}
			object[key] = value
		}
		closing, err := decoder.Token()
		if err != nil {
			return nil, err
		}
		if closing != json.Delim('}') {
			return nil, errors.New("unterminated JSON object")
		}
		return object, nil
	case '[':
		array := make([]any, 0)
		for decoder.More() {
			value, err := decodeFrozenUniqueJSONValue(decoder)
			if err != nil {
				return nil, err
			}
			array = append(array, value)
		}
		closing, err := decoder.Token()
		if err != nil {
			return nil, err
		}
		if closing != json.Delim(']') {
			return nil, errors.New("unterminated JSON array")
		}
		return array, nil
	default:
		return nil, fmt.Errorf("unexpected JSON delimiter %q", delimiter)
	}
}

func decodeFrozenUpperHash(label, raw string) ([]byte, error) {
	if len(raw) != sha256.Size*2 || strings.ToUpper(raw) != raw {
		return nil, fmt.Errorf("%s must be 64 uppercase hexadecimal characters", label)
	}
	decoded, err := hex.DecodeString(raw)
	if err != nil {
		return nil, fmt.Errorf("%s must be 64 uppercase hexadecimal characters", label)
	}
	return decoded, nil
}
