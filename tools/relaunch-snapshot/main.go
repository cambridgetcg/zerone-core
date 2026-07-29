// relaunch-snapshot captures the public, supply-bearing boundary of a frozen
// Zerone chain. It is intentionally read-only: the output becomes an input to
// a successor-chain migration ceremony, never a transaction broadcaster.
package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	schema           = "zerone-relaunch-snapshot-v3"
	maxResponseBytes = 64 << 20
	restTrustModel   = "trusted height-pinned REST responses; no Merkle proof binds inventory to checkpoint_app_hash"
)

type coin struct {
	Denom  string `json:"denom"`
	Amount string `json:"amount"`
}

type owner struct {
	Address     string `json:"address"`
	AccountType string `json:"account_type"`
	ModuleName  string `json:"module_name,omitempty"`
	Amount      string `json:"amount_uzrn"`
}

type validator struct {
	OperatorAddress string          `json:"operator_address"`
	ConsensusPubKey consensusPubKey `json:"consensus_pubkey"`
	Jailed          bool            `json:"jailed"`
	Status          string          `json:"status"`
	Tokens          string          `json:"tokens"`
}

type consensusPubKey struct {
	Type string `json:"@type"`
	Key  string `json:"key"`
}

type sourceCheckpoint struct {
	ChainID string `json:"chain_id"`
	// CheckpointStateHeight F is the state queried through REST. Empty canonical
	// block A=F+1 carries CheckpointAppHash in its signed header. Comet stages
	// H=A+1 before the SDK rejects FinalizeBlock(H), so the block-store tip is H
	// while ABCI remains at A.
	CheckpointStateHeight              int64  `json:"checkpoint_state_height"`
	CheckpointAppHash                  string `json:"checkpoint_app_hash"`
	FinalCommittedBlockHeight          int64  `json:"final_committed_block_height"`
	FinalCommittedBlockHash            string `json:"final_committed_block_hash"`
	FinalCommittedBlockTime            string `json:"final_committed_block_time"`
	FinalCommittedBlockTxs             int    `json:"final_committed_block_txs"`
	FinalCommittedBlockCanonical       bool   `json:"final_committed_block_canonical"`
	FinalCommittedBlockHasResults      bool   `json:"final_committed_block_has_results"`
	HaltTriggerHeight                  int64  `json:"halt_trigger_height"`
	RPCBlockstoreHeight                int64  `json:"rpc_blockstore_height"`
	StagedHaltTriggerBlockHash         string `json:"staged_halt_trigger_block_hash"`
	StagedHaltTriggerBlockTime         string `json:"staged_halt_trigger_block_time"`
	StagedHaltTriggerBlockTxs          int    `json:"staged_halt_trigger_block_txs"`
	StagedHaltTriggerPreviousBlockHash string `json:"staged_halt_trigger_previous_block_hash"`
	StagedHaltTriggerHeaderAppHash     string `json:"staged_halt_trigger_header_app_hash"`
	StagedHaltTriggerCommitCanonical   bool   `json:"staged_halt_trigger_commit_canonical"`
	StagedHaltTriggerHasBlockResults   bool   `json:"staged_halt_trigger_has_block_results"`
	ABCILastAppliedHeight              int64  `json:"abci_last_applied_height"`
	// State after A is recorded only to prove the split. Successor eligibility
	// remains pinned to F, even though staged application-unapplied H names it.
	ExcludedPostAnchorAppHash string `json:"excluded_post_anchor_app_hash"`
	RPCGenesisCanonicalSHA256 string `json:"rpc_genesis_canonical_sha256"`
	DeclaredGenesisFileSHA256 string `json:"declared_genesis_file_sha256,omitempty"`
	RESTTrustModel            string `json:"rest_trust_model"`
	RPC                       string `json:"rpc"`
	REST                      string `json:"rest"`
}

type snapshot struct {
	Schema     string           `json:"schema"`
	Source     sourceCheckpoint `json:"source"`
	Denom      string           `json:"denom"`
	Supply     string           `json:"supply_uzrn"`
	Owners     []owner          `json:"owners"`
	Validators []validator      `json:"bonded_validators"`
}

type page struct {
	NextKey json.RawMessage `json:"next_key"`
}

type api struct {
	client                    *http.Client
	rpc                       string
	rest                      string
	height                    int64
	checkpointStateHeight     int64
	finalCommittedBlockHeight int64
	haltTriggerHeight         int64
}

func main() {
	var (
		rpc                   = flag.String("rpc", "", "CometBFT RPC base URL")
		rest                  = flag.String("rest", "", "Cosmos REST base URL")
		expectedChain         = flag.String("expected-chain-id", "", "required source chain ID")
		checkpointStateHeight = flag.Int64("checkpoint-state-height", 0, "state height F selected for inventory")
		finalCommittedHeight  = flag.Int64("final-committed-height", 0, "empty anchor block height A=F+1")
		haltTriggerHeight     = flag.Int64("halt-trigger-height", 0, "SDK halt trigger H=A+1")
		declaredGenesis       = flag.String("declared-genesis-sha256", "", "required published raw genesis.json SHA-256")
		denom                 = flag.String("denom", "uzrn", "supply denomination")
		out                   = flag.String("out", "", "output path (default stdout)")
		timeout               = flag.Duration("timeout", 15*time.Second, "per-request timeout")
	)
	flag.Parse()

	if *rpc == "" || *rest == "" || *expectedChain == "" || *declaredGenesis == "" {
		fmt.Fprintln(os.Stderr, "usage: relaunch-snapshot --rpc URL --rest URL --expected-chain-id ID --checkpoint-state-height F --final-committed-height A --halt-trigger-height H --declared-genesis-sha256 SHA [--out FILE]")
		os.Exit(2)
	}
	if !isSHA256(*declaredGenesis) {
		fmt.Fprintln(os.Stderr, "declared genesis SHA-256 must be 64 lowercase hex characters")
		os.Exit(2)
	}
	if err := validateHeightPlan(*checkpointStateHeight, *finalCommittedHeight, *haltTriggerHeight); err != nil {
		fmt.Fprintf(os.Stderr, "invalid checkpoint height plan: %v\n", err)
		os.Exit(2)
	}
	if *timeout <= 0 {
		fmt.Fprintln(os.Stderr, "timeout must be positive")
		os.Exit(2)
	}
	normalizedRPC, err := normalizeBaseURL(*rpc)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid RPC URL: %v\n", err)
		os.Exit(2)
	}
	normalizedREST, err := normalizeBaseURL(*rest)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid REST URL: %v\n", err)
		os.Exit(2)
	}

	ctx := context.Background()
	a := &api{
		client: &http.Client{
			Timeout: *timeout,
			CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		rpc:                       normalizedRPC,
		rest:                      normalizedREST,
		checkpointStateHeight:     *checkpointStateHeight,
		finalCommittedBlockHeight: *finalCommittedHeight,
		haltTriggerHeight:         *haltTriggerHeight,
	}
	s, err := a.capture(ctx, *expectedChain, *declaredGenesis, *denom)
	if err != nil {
		fmt.Fprintf(os.Stderr, "relaunch snapshot failed: %v\n", err)
		os.Exit(1)
	}

	bz, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "marshal snapshot: %v\n", err)
		os.Exit(1)
	}
	bz = append(bz, '\n')
	if *out == "" {
		if _, err := os.Stdout.Write(bz); err != nil {
			fmt.Fprintf(os.Stderr, "write snapshot: %v\n", err)
			os.Exit(1)
		}
		return
	}
	if err := writeAtomic(*out, bz); err != nil {
		fmt.Fprintf(os.Stderr, "write snapshot: %v\n", err)
		os.Exit(1)
	}
}

func normalizeBaseURL(raw string) (string, error) {
	parsed, err := url.Parse(raw)
	if err != nil {
		return "", err
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fmt.Errorf("scheme %q is not http or https", parsed.Scheme)
	}
	if parsed.Host == "" {
		return "", errors.New("host is empty")
	}
	if parsed.User != nil {
		return "", errors.New("embedded credentials are forbidden because endpoint URLs are written to the snapshot")
	}
	if parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", errors.New("query strings and fragments are not allowed on a base URL")
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/")
	parsed.RawPath = strings.TrimRight(parsed.RawPath, "/")
	return parsed.String(), nil
}

func validateHeightPlan(checkpointStateHeight, finalCommittedBlockHeight, haltTriggerHeight int64) error {
	if checkpointStateHeight <= 0 || finalCommittedBlockHeight <= 0 || haltTriggerHeight <= 0 {
		return errors.New("checkpoint, final-committed, and halt-trigger heights must all be positive")
	}
	if checkpointStateHeight > 9223372036854775805 {
		return errors.New("checkpoint state height is too large to leave room for anchor and halt heights")
	}
	if finalCommittedBlockHeight != checkpointStateHeight+1 {
		return fmt.Errorf("final committed block %d must equal checkpoint state %d + 1", finalCommittedBlockHeight, checkpointStateHeight)
	}
	if haltTriggerHeight != finalCommittedBlockHeight+1 {
		return fmt.Errorf("halt trigger %d must equal final committed block %d + 1", haltTriggerHeight, finalCommittedBlockHeight)
	}
	return nil
}

func (a *api) capture(ctx context.Context, expectedChainID, declaredGenesisSHA, denom string) (*snapshot, error) {
	boundary, err := a.captureHaltBoundary(ctx, expectedChainID)
	if err != nil {
		return nil, err
	}
	a.height = a.checkpointStateHeight
	genesisSHA, err := a.genesisCanonicalSHA(ctx, expectedChainID)
	if err != nil {
		return nil, err
	}
	accountTypes, err := a.accountTypes(ctx)
	if err != nil {
		return nil, err
	}
	owners, err := a.denomOwners(ctx, denom, accountTypes)
	if err != nil {
		return nil, err
	}
	supply, err := a.supply(ctx, denom)
	if err != nil {
		return nil, err
	}
	if err := verifySupply(owners, supply); err != nil {
		return nil, err
	}
	validators, err := a.bondedValidators(ctx)
	if err != nil {
		return nil, err
	}
	if err := a.verifySourceStable(ctx, expectedChainID, boundary); err != nil {
		return nil, err
	}

	return &snapshot{
		Schema: schema,
		Source: sourceCheckpoint{
			ChainID:                            boundary.Status.ChainID,
			CheckpointStateHeight:              a.checkpointStateHeight,
			CheckpointAppHash:                  boundary.Anchor.HeaderAppHash,
			FinalCommittedBlockHeight:          a.finalCommittedBlockHeight,
			FinalCommittedBlockHash:            boundary.Anchor.Hash,
			FinalCommittedBlockTime:            boundary.Anchor.Time,
			FinalCommittedBlockTxs:             boundary.Anchor.TxCount,
			FinalCommittedBlockCanonical:       boundary.AnchorCommit.Canonical,
			FinalCommittedBlockHasResults:      true,
			HaltTriggerHeight:                  a.haltTriggerHeight,
			RPCBlockstoreHeight:                boundary.Status.Height,
			StagedHaltTriggerBlockHash:         boundary.Trigger.Hash,
			StagedHaltTriggerBlockTime:         boundary.Trigger.Time,
			StagedHaltTriggerBlockTxs:          boundary.Trigger.TxCount,
			StagedHaltTriggerPreviousBlockHash: boundary.Trigger.LastBlockHash,
			StagedHaltTriggerHeaderAppHash:     boundary.Trigger.HeaderAppHash,
			StagedHaltTriggerCommitCanonical:   boundary.TriggerCommit.Canonical,
			StagedHaltTriggerHasBlockResults:   false,
			ABCILastAppliedHeight:              a.finalCommittedBlockHeight,
			ExcludedPostAnchorAppHash:          boundary.PostAnchorAppHash,
			RPCGenesisCanonicalSHA256:          genesisSHA,
			DeclaredGenesisFileSHA256:          declaredGenesisSHA,
			RESTTrustModel:                     restTrustModel,
			RPC:                                a.rpc,
			REST:                               a.rest,
		},
		Denom:      denom,
		Supply:     supply,
		Owners:     owners,
		Validators: validators,
	}, nil
}

type haltBoundary struct {
	Status            statusResult
	Anchor            blockResult
	Trigger           blockResult
	AnchorCommit      commitResult
	TriggerCommit     commitResult
	PostAnchorAppHash string
}

func (a *api) captureHaltBoundary(ctx context.Context, expectedChainID string) (haltBoundary, error) {
	status, err := a.status(ctx)
	if err != nil {
		return haltBoundary{}, err
	}
	if status.ChainID != expectedChainID {
		return haltBoundary{}, fmt.Errorf("RPC chain ID %q, expected %q", status.ChainID, expectedChainID)
	}
	if status.CatchingUp {
		return haltBoundary{}, errors.New("source RPC is still catching up")
	}
	if status.Height != a.haltTriggerHeight {
		return haltBoundary{}, fmt.Errorf("RPC block-store height %d, expected staged halt trigger %d", status.Height, a.haltTriggerHeight)
	}

	anchor, err := a.block(ctx, a.finalCommittedBlockHeight)
	if err != nil {
		return haltBoundary{}, err
	}
	trigger, err := a.block(ctx, a.haltTriggerHeight)
	if err != nil {
		return haltBoundary{}, err
	}
	if anchor.ChainID != status.ChainID || trigger.ChainID != status.ChainID {
		return haltBoundary{}, fmt.Errorf("boundary block chain IDs %q/%q, status reported %q", anchor.ChainID, trigger.ChainID, status.ChainID)
	}
	if trigger.Hash != status.BlockHash || trigger.HeaderAppHash != status.HeaderAppHash {
		return haltBoundary{}, errors.New("status does not identify the staged halt-trigger block")
	}
	if trigger.LastBlockHash != anchor.Hash {
		return haltBoundary{}, fmt.Errorf("staged halt-trigger block links %s, expected anchor %s", trigger.LastBlockHash, anchor.Hash)
	}
	if anchor.TxCount != 0 {
		return haltBoundary{}, fmt.Errorf("final committed anchor block %d contains %d transaction(s); expected exactly zero", a.finalCommittedBlockHeight, anchor.TxCount)
	}
	if trigger.TxCount != 0 {
		return haltBoundary{}, fmt.Errorf("staged halt-trigger block %d contains %d transaction(s); expected exactly zero", a.haltTriggerHeight, trigger.TxCount)
	}

	anchorCommit, err := a.commit(ctx, a.finalCommittedBlockHeight)
	if err != nil {
		return haltBoundary{}, err
	}
	triggerCommit, err := a.commit(ctx, a.haltTriggerHeight)
	if err != nil {
		return haltBoundary{}, err
	}
	if !anchorCommit.Canonical || anchorCommit.BlockHash != anchor.Hash ||
		anchorCommit.HeaderAppHash != anchor.HeaderAppHash || anchorCommit.ChainID != status.ChainID {
		return haltBoundary{}, errors.New("final anchor does not have the expected canonical commit")
	}
	if triggerCommit.Canonical || triggerCommit.BlockHash != trigger.Hash ||
		triggerCommit.HeaderAppHash != trigger.HeaderAppHash || triggerCommit.ChainID != status.ChainID {
		return haltBoundary{}, errors.New("halt-trigger tip does not have the expected canonical=false subjective seen commit")
	}

	postAnchorAppHash, err := a.stateAppHash(ctx, a.finalCommittedBlockHeight)
	if err != nil {
		return haltBoundary{}, err
	}
	if postAnchorAppHash != trigger.HeaderAppHash {
		return haltBoundary{}, fmt.Errorf("ABCI app hash %s does not match staged trigger header %s", postAnchorAppHash, trigger.HeaderAppHash)
	}
	if err := a.requireBlockResults(ctx, a.finalCommittedBlockHeight); err != nil {
		return haltBoundary{}, err
	}
	if err := a.requireNoBlockResults(ctx, a.haltTriggerHeight); err != nil {
		return haltBoundary{}, err
	}
	return haltBoundary{
		Status: status, Anchor: anchor, Trigger: trigger,
		AnchorCommit: anchorCommit, TriggerCommit: triggerCommit,
		PostAnchorAppHash: postAnchorAppHash,
	}, nil

}

func (a *api) verifySourceStable(ctx context.Context, expectedChainID string, initial haltBoundary) error {
	final, err := a.captureHaltBoundary(ctx, expectedChainID)
	if err != nil {
		return fmt.Errorf("final halted-boundary recheck: %w", err)
	}
	if final != initial {
		return fmt.Errorf("source changed during capture: initial %+v, final %+v", initial, final)
	}
	return nil
}

type statusResult struct {
	ChainID       string
	Height        int64
	BlockHash     string
	HeaderAppHash string
	CatchingUp    bool
}

func (a *api) status(ctx context.Context) (statusResult, error) {
	var response struct {
		Result struct {
			NodeInfo struct {
				Network string `json:"network"`
			} `json:"node_info"`
			SyncInfo struct {
				LatestHeight    string `json:"latest_block_height"`
				LatestBlockHash string `json:"latest_block_hash"`
				LatestAppHash   string `json:"latest_app_hash"`
				CatchingUp      bool   `json:"catching_up"`
			} `json:"sync_info"`
		} `json:"result"`
	}
	if err := a.getJSON(ctx, a.rpc+"/status", false, &response); err != nil {
		return statusResult{}, fmt.Errorf("query status: %w", err)
	}
	height, err := strconv.ParseInt(response.Result.SyncInfo.LatestHeight, 10, 64)
	if err != nil || height < 1 {
		return statusResult{}, fmt.Errorf("invalid latest height %q", response.Result.SyncInfo.LatestHeight)
	}
	if !isCometHash(response.Result.SyncInfo.LatestBlockHash) {
		return statusResult{}, fmt.Errorf("invalid latest block hash %q", response.Result.SyncInfo.LatestBlockHash)
	}
	if !isCometHash(response.Result.SyncInfo.LatestAppHash) {
		return statusResult{}, fmt.Errorf("invalid latest app hash %q", response.Result.SyncInfo.LatestAppHash)
	}
	return statusResult{
		ChainID:       response.Result.NodeInfo.Network,
		Height:        height,
		BlockHash:     strings.ToUpper(response.Result.SyncInfo.LatestBlockHash),
		HeaderAppHash: strings.ToUpper(response.Result.SyncInfo.LatestAppHash),
		CatchingUp:    response.Result.SyncInfo.CatchingUp,
	}, nil
}

type blockResult struct {
	Hash          string
	LastBlockHash string
	HeaderAppHash string
	ChainID       string
	Time          string
	TxCount       int
}

func (a *api) block(ctx context.Context, height int64) (blockResult, error) {
	var response struct {
		Result struct {
			BlockID struct {
				Hash string `json:"hash"`
			} `json:"block_id"`
			Block struct {
				Header struct {
					ChainID     string `json:"chain_id"`
					Height      string `json:"height"`
					Time        string `json:"time"`
					AppHash     string `json:"app_hash"`
					LastBlockID struct {
						Hash string `json:"hash"`
					} `json:"last_block_id"`
				} `json:"header"`
				Data *struct {
					Txs json.RawMessage `json:"txs"`
				} `json:"data"`
			} `json:"block"`
		} `json:"result"`
	}
	endpoint := fmt.Sprintf("%s/block?height=%d", a.rpc, height)
	if err := a.getJSON(ctx, endpoint, false, &response); err != nil {
		return blockResult{}, fmt.Errorf("query block %d: %w", height, err)
	}
	if response.Result.Block.Header.Height != strconv.FormatInt(height, 10) {
		return blockResult{}, fmt.Errorf("RPC returned block height %q, expected %d", response.Result.Block.Header.Height, height)
	}
	if response.Result.Block.Header.ChainID == "" {
		return blockResult{}, errors.New("RPC returned block with empty chain ID")
	}
	if !isCometHash(response.Result.BlockID.Hash) {
		return blockResult{}, fmt.Errorf("invalid block hash %q", response.Result.BlockID.Hash)
	}
	if !isCometHash(response.Result.Block.Header.AppHash) {
		return blockResult{}, fmt.Errorf("invalid block header app hash %q", response.Result.Block.Header.AppHash)
	}
	if !isCometHash(response.Result.Block.Header.LastBlockID.Hash) {
		return blockResult{}, fmt.Errorf("invalid previous block hash %q", response.Result.Block.Header.LastBlockID.Hash)
	}
	blockTime, err := time.Parse(time.RFC3339Nano, response.Result.Block.Header.Time)
	if err != nil {
		return blockResult{}, fmt.Errorf("invalid block time: %w", err)
	}
	txCount, err := blockTxCount(response.Result.Block.Data)
	if err != nil {
		return blockResult{}, fmt.Errorf("invalid block data at height %d: %w", height, err)
	}
	return blockResult{
		Hash:          strings.ToUpper(response.Result.BlockID.Hash),
		LastBlockHash: strings.ToUpper(response.Result.Block.Header.LastBlockID.Hash),
		HeaderAppHash: strings.ToUpper(response.Result.Block.Header.AppHash),
		ChainID:       response.Result.Block.Header.ChainID,
		Time:          blockTime.UTC().Format(time.RFC3339Nano),
		TxCount:       txCount,
	}, nil
}

type commitResult struct {
	Height        int64
	ChainID       string
	BlockHash     string
	HeaderAppHash string
	Canonical     bool
	Signatures    int
}

func (a *api) commit(ctx context.Context, height int64) (commitResult, error) {
	var response struct {
		Result struct {
			Canonical    *bool `json:"canonical"`
			SignedHeader struct {
				Header struct {
					ChainID string `json:"chain_id"`
					Height  string `json:"height"`
					AppHash string `json:"app_hash"`
				} `json:"header"`
				Commit struct {
					Height  string `json:"height"`
					BlockID struct {
						Hash string `json:"hash"`
					} `json:"block_id"`
					Signatures []json.RawMessage `json:"signatures"`
				} `json:"commit"`
			} `json:"signed_header"`
		} `json:"result"`
	}
	endpoint := fmt.Sprintf("%s/commit?height=%d", a.rpc, height)
	if err := a.getJSON(ctx, endpoint, false, &response); err != nil {
		return commitResult{}, fmt.Errorf("query commit %d: %w", height, err)
	}
	want := strconv.FormatInt(height, 10)
	if response.Result.SignedHeader.Header.Height != want ||
		response.Result.SignedHeader.Commit.Height != want {
		return commitResult{}, fmt.Errorf("commit endpoint returned header/commit heights %q/%q, expected %d",
			response.Result.SignedHeader.Header.Height,
			response.Result.SignedHeader.Commit.Height, height)
	}
	if response.Result.Canonical == nil {
		return commitResult{}, fmt.Errorf("commit %d omitted canonical flag", height)
	}
	if response.Result.SignedHeader.Header.ChainID == "" {
		return commitResult{}, fmt.Errorf("commit %d has empty chain ID", height)
	}
	if !isCometHash(response.Result.SignedHeader.Commit.BlockID.Hash) {
		return commitResult{}, fmt.Errorf("commit %d has invalid block hash %q", height, response.Result.SignedHeader.Commit.BlockID.Hash)
	}
	if !isCometHash(response.Result.SignedHeader.Header.AppHash) {
		return commitResult{}, fmt.Errorf("commit %d has invalid header app hash %q", height, response.Result.SignedHeader.Header.AppHash)
	}
	if len(response.Result.SignedHeader.Commit.Signatures) == 0 {
		return commitResult{}, fmt.Errorf("commit %d has no signatures", height)
	}
	return commitResult{
		Height:        height,
		ChainID:       response.Result.SignedHeader.Header.ChainID,
		BlockHash:     strings.ToUpper(response.Result.SignedHeader.Commit.BlockID.Hash),
		HeaderAppHash: strings.ToUpper(response.Result.SignedHeader.Header.AppHash),
		Canonical:     *response.Result.Canonical,
		Signatures:    len(response.Result.SignedHeader.Commit.Signatures),
	}, nil
}

func (a *api) requireBlockResults(ctx context.Context, height int64) error {
	var response struct {
		Result struct {
			Height string `json:"height"`
		} `json:"result"`
	}
	endpoint := fmt.Sprintf("%s/block_results?height=%d", a.rpc, height)
	if err := a.getJSON(ctx, endpoint, false, &response); err != nil {
		return fmt.Errorf("query block results %d: %w", height, err)
	}
	if response.Result.Height != strconv.FormatInt(height, 10) {
		return fmt.Errorf("block results returned height %q, expected %d", response.Result.Height, height)
	}
	return nil
}

func (a *api) requireNoBlockResults(ctx context.Context, height int64) error {
	endpoint := fmt.Sprintf("%s/block_results?height=%d", a.rpc, height)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	resp, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("query absent block results %d: %w", height, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return fmt.Errorf("read absent block results %d: %w", height, err)
	}
	if len(body) == 1<<20 {
		return fmt.Errorf("absent block-results response for %d is too large", height)
	}
	var envelope struct {
		Result json.RawMessage `json:"result"`
		Error  *struct {
			Message string          `json:"message"`
			Data    json.RawMessage `json:"data"`
		} `json:"error"`
	}
	if err := decodeSingleJSON(body, &envelope); err != nil {
		return fmt.Errorf("decode absent block-results response for %d: %w", height, err)
	}
	if len(bytes.TrimSpace(envelope.Result)) > 0 && !bytes.Equal(bytes.TrimSpace(envelope.Result), []byte("null")) {
		return fmt.Errorf("halt-trigger height %d unexpectedly has application block results", height)
	}
	want := fmt.Sprintf("could not find results for height #%d", height)
	if envelope.Error == nil ||
		!strings.Contains(envelope.Error.Message+" "+string(envelope.Error.Data), want) {
		return fmt.Errorf("block results %d returned HTTP %d without the expected missing-results proof", height, resp.StatusCode)
	}
	return nil
}

func blockTxCount(data *struct {
	Txs json.RawMessage `json:"txs"`
}) (int, error) {
	if data == nil {
		return 0, errors.New("missing block.data")
	}
	raw := bytes.TrimSpace(data.Txs)
	if len(raw) == 0 {
		return 0, errors.New("missing block.data.txs")
	}
	if bytes.Equal(raw, []byte("null")) {
		return 0, nil
	}
	var txs []json.RawMessage
	if err := decodeSingleJSON(raw, &txs); err != nil {
		return 0, err
	}
	return len(txs), nil
}

func (a *api) stateAppHash(ctx context.Context, expectedHeight int64) (string, error) {
	var response struct {
		Result struct {
			Response struct {
				LastBlockHeight  string `json:"last_block_height"`
				LastBlockAppHash string `json:"last_block_app_hash"`
			} `json:"response"`
		} `json:"result"`
	}
	if err := a.getJSON(ctx, a.rpc+"/abci_info", false, &response); err != nil {
		return "", fmt.Errorf("query ABCI info: %w", err)
	}
	if response.Result.Response.LastBlockHeight != strconv.FormatInt(expectedHeight, 10) {
		return "", fmt.Errorf("ABCI last block height %q, expected %d", response.Result.Response.LastBlockHeight, expectedHeight)
	}
	normalized, err := normalizeAppHash(response.Result.Response.LastBlockAppHash)
	if err != nil {
		return "", fmt.Errorf("invalid ABCI last block app hash: %w", err)
	}
	return normalized, nil
}

func normalizeAppHash(raw string) (string, error) {
	if isCometHash(raw) {
		return strings.ToUpper(raw), nil
	}
	decoded, err := base64.StdEncoding.DecodeString(raw)
	if err != nil || len(decoded) != sha256.Size || base64.StdEncoding.EncodeToString(decoded) != raw {
		return "", fmt.Errorf("%q is neither a 32-byte hex nor canonical base64 hash", raw)
	}
	return strings.ToUpper(hex.EncodeToString(decoded)), nil
}

func (a *api) genesisCanonicalSHA(ctx context.Context, expectedChainID string) (string, error) {
	var response struct {
		Result struct {
			Genesis json.RawMessage `json:"genesis"`
		} `json:"result"`
	}
	if err := a.getJSON(ctx, a.rpc+"/genesis", false, &response); err != nil {
		return "", fmt.Errorf("query genesis: %w", err)
	}
	var canonical map[string]any
	if err := decodeSingleJSON(response.Result.Genesis, &canonical); err != nil {
		return "", fmt.Errorf("decode RPC genesis: %w", err)
	}
	chainID, _ := canonical["chain_id"].(string)
	if chainID != expectedChainID {
		return "", fmt.Errorf("RPC genesis chain ID %q, expected %q", chainID, expectedChainID)
	}
	// encoding/json sorts map keys. Combined with UseNumber above, this gives
	// the schema's deterministic semantic hash; it is deliberately distinct
	// from the byte-for-byte genesis file hash supplied by the operator.
	bz, err := json.Marshal(canonical)
	if err != nil {
		return "", fmt.Errorf("canonicalize RPC genesis: %w", err)
	}
	sum := sha256.Sum256(bz)
	return hex.EncodeToString(sum[:]), nil
}

type accountMetadata struct {
	Type       string
	ModuleName string
}

func (a *api) accountTypes(ctx context.Context) (map[string]accountMetadata, error) {
	result := make(map[string]accountMetadata)
	next := ""
	seenPageKeys := make(map[string]struct{})
	for {
		endpoint := a.rest + "/cosmos/auth/v1beta1/accounts?pagination.limit=500"
		if next != "" {
			endpoint += "&pagination.key=" + url.QueryEscape(next)
		}
		var response struct {
			Accounts   []json.RawMessage `json:"accounts"`
			Pagination *page             `json:"pagination"`
		}
		if err := a.getJSON(ctx, endpoint, true, &response); err != nil {
			return nil, fmt.Errorf("query auth accounts: %w", err)
		}
		for _, raw := range response.Accounts {
			address, metadata, err := parseAccount(raw)
			if err != nil {
				return nil, err
			}
			if _, duplicate := result[address]; duplicate {
				return nil, fmt.Errorf("duplicate auth account %s", address)
			}
			result[address] = metadata
		}
		pageNext, done, err := advancePage(response.Pagination, seenPageKeys)
		if err != nil {
			return nil, fmt.Errorf("auth accounts pagination: %w", err)
		}
		if done {
			break
		}
		next = pageNext
	}
	return result, nil
}

func (a *api) denomOwners(ctx context.Context, denom string, metadata map[string]accountMetadata) ([]owner, error) {
	var owners []owner
	next := ""
	seenPageKeys := make(map[string]struct{})
	for {
		endpoint := fmt.Sprintf("%s/cosmos/bank/v1beta1/denom_owners/%s?pagination.limit=500", a.rest, url.PathEscape(denom))
		if next != "" {
			endpoint += "&pagination.key=" + url.QueryEscape(next)
		}
		var response struct {
			Owners []struct {
				Address string `json:"address"`
				Balance coin   `json:"balance"`
			} `json:"denom_owners"`
			Pagination *page `json:"pagination"`
		}
		if err := a.getJSON(ctx, endpoint, true, &response); err != nil {
			return nil, fmt.Errorf("query %s owners: %w", denom, err)
		}
		for _, entry := range response.Owners {
			if entry.Address == "" || entry.Balance.Denom != denom {
				return nil, fmt.Errorf("malformed denom owner response for %s", entry.Address)
			}
			amount, err := parseDecimalAmount(entry.Balance.Amount)
			if err != nil || amount.Sign() <= 0 {
				return nil, fmt.Errorf("invalid balance %q for %s", entry.Balance.Amount, entry.Address)
			}
			m, hasAccount := metadata[entry.Address]
			if m.Type == "" {
				if hasAccount {
					return nil, fmt.Errorf("auth account %s has empty account type", entry.Address)
				}
				m.Type = "bank_only"
			}
			owners = append(owners, owner{
				Address: entry.Address, AccountType: m.Type,
				ModuleName: m.ModuleName, Amount: entry.Balance.Amount,
			})
		}
		pageNext, done, err := advancePage(response.Pagination, seenPageKeys)
		if err != nil {
			return nil, fmt.Errorf("%s owners pagination: %w", denom, err)
		}
		if done {
			break
		}
		next = pageNext
	}
	sort.Slice(owners, func(i, j int) bool { return owners[i].Address < owners[j].Address })
	for i := 1; i < len(owners); i++ {
		if owners[i-1].Address == owners[i].Address {
			return nil, fmt.Errorf("duplicate denom owner %s", owners[i].Address)
		}
	}
	return owners, nil
}

func (a *api) supply(ctx context.Context, denom string) (string, error) {
	var response struct {
		Amount coin `json:"amount"`
	}
	endpoint := fmt.Sprintf("%s/cosmos/bank/v1beta1/supply/by_denom?denom=%s", a.rest, url.QueryEscape(denom))
	if err := a.getJSON(ctx, endpoint, true, &response); err != nil {
		return "", fmt.Errorf("query %s supply: %w", denom, err)
	}
	if response.Amount.Denom != denom {
		return "", fmt.Errorf("supply response denom %q, expected %q", response.Amount.Denom, denom)
	}
	if _, err := parseDecimalAmount(response.Amount.Amount); err != nil {
		return "", fmt.Errorf("invalid supply %q", response.Amount.Amount)
	}
	return response.Amount.Amount, nil
}

func (a *api) bondedValidators(ctx context.Context) ([]validator, error) {
	var validators []validator
	next := ""
	seenPageKeys := make(map[string]struct{})
	for {
		endpoint := a.rest + "/cosmos/staking/v1beta1/validators?status=BOND_STATUS_BONDED&pagination.limit=500"
		if next != "" {
			endpoint += "&pagination.key=" + url.QueryEscape(next)
		}
		var response struct {
			Validators []struct {
				OperatorAddress string          `json:"operator_address"`
				ConsensusPubKey consensusPubKey `json:"consensus_pubkey"`
				Jailed          bool            `json:"jailed"`
				Status          string          `json:"status"`
				Tokens          string          `json:"tokens"`
			} `json:"validators"`
			Pagination *page `json:"pagination"`
		}
		if err := a.getJSON(ctx, endpoint, true, &response); err != nil {
			return nil, fmt.Errorf("query bonded validators: %w", err)
		}
		for _, v := range response.Validators {
			if v.OperatorAddress == "" || v.Status != "BOND_STATUS_BONDED" {
				return nil, fmt.Errorf("malformed bonded validator %q", v.OperatorAddress)
			}
			if err := validateConsensusPubKey(v.ConsensusPubKey); err != nil {
				return nil, fmt.Errorf("validator %s consensus pubkey: %w", v.OperatorAddress, err)
			}
			tokens, err := parseDecimalAmount(v.Tokens)
			if err != nil || tokens.Sign() <= 0 {
				return nil, fmt.Errorf("validator %s has invalid bonded tokens %q", v.OperatorAddress, v.Tokens)
			}
			validators = append(validators, validator(v))
		}
		pageNext, done, err := advancePage(response.Pagination, seenPageKeys)
		if err != nil {
			return nil, fmt.Errorf("bonded validators pagination: %w", err)
		}
		if done {
			break
		}
		next = pageNext
	}
	sort.Slice(validators, func(i, j int) bool { return validators[i].OperatorAddress < validators[j].OperatorAddress })
	for i := 1; i < len(validators); i++ {
		if validators[i-1].OperatorAddress == validators[i].OperatorAddress {
			return nil, fmt.Errorf("duplicate bonded validator %s", validators[i].OperatorAddress)
		}
	}
	return validators, nil
}

func (a *api) getJSON(ctx context.Context, endpoint string, atHeight bool, target any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	if atHeight {
		req.Header.Set("x-cosmos-block-height", strconv.FormatInt(a.height, 10))
	}
	resp, err := a.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("GET %s: HTTP %d: %s", endpoint, resp.StatusCode, strings.TrimSpace(string(body)))
	}
	if atHeight {
		got, err := cosmosResponseHeight(resp.Header)
		if err != nil {
			return fmt.Errorf("GET %s: %w", endpoint, err)
		}
		if got != strconv.FormatInt(a.height, 10) {
			return fmt.Errorf("GET %s returned height %s, expected %d", endpoint, got, a.height)
		}
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes+1))
	if err != nil {
		return fmt.Errorf("read GET %s: %w", endpoint, err)
	}
	if len(body) > maxResponseBytes {
		return fmt.Errorf("GET %s response exceeds %d bytes", endpoint, maxResponseBytes)
	}
	if err := decodeSingleJSON(body, target); err != nil {
		return fmt.Errorf("decode GET %s: %w", endpoint, err)
	}
	return nil
}

func cosmosResponseHeight(header http.Header) (string, error) {
	direct := strings.TrimSpace(header.Get("x-cosmos-block-height"))
	gateway := strings.TrimSpace(header.Get("grpc-metadata-x-cosmos-block-height"))
	if direct == "" && gateway == "" {
		return "", errors.New("missing Cosmos block-height response header")
	}
	if direct != "" && gateway != "" && direct != gateway {
		return "", fmt.Errorf("conflicting Cosmos block-height response headers %q and %q", direct, gateway)
	}
	if direct != "" {
		return direct, nil
	}
	return gateway, nil
}

func decodeSingleJSON(data []byte, target any) error {
	if err := rejectDuplicateJSONKeys(data); err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("multiple JSON values")
		}
		return fmt.Errorf("trailing JSON data: %w", err)
	}
	return nil
}

func rejectDuplicateJSONKeys(data []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := walkJSONValue(decoder); err != nil {
		return err
	}
	if _, err := decoder.Token(); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("multiple JSON values")
		}
		return fmt.Errorf("trailing JSON data: %w", err)
	}
	return nil
}

func walkJSONValue(decoder *json.Decoder) error {
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	delim, isDelimiter := token.(json.Delim)
	if !isDelimiter {
		return nil
	}
	switch delim {
	case '{':
		seen := make(map[string]struct{})
		for decoder.More() {
			keyToken, err := decoder.Token()
			if err != nil {
				return err
			}
			key, ok := keyToken.(string)
			if !ok {
				return errors.New("JSON object key is not a string")
			}
			if _, duplicate := seen[key]; duplicate {
				return fmt.Errorf("duplicate JSON object key %q", key)
			}
			seen[key] = struct{}{}
			if err := walkJSONValue(decoder); err != nil {
				return err
			}
		}
		closing, err := decoder.Token()
		if err != nil {
			return err
		}
		if closing != json.Delim('}') {
			return errors.New("malformed JSON object")
		}
	case '[':
		for decoder.More() {
			if err := walkJSONValue(decoder); err != nil {
				return err
			}
		}
		closing, err := decoder.Token()
		if err != nil {
			return err
		}
		if closing != json.Delim(']') {
			return errors.New("malformed JSON array")
		}
	default:
		return fmt.Errorf("unexpected JSON delimiter %q", delim)
	}
	return nil
}

func parseAccount(raw json.RawMessage) (string, accountMetadata, error) {
	var value map[string]any
	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	decoder.UseNumber()
	if err := decoder.Decode(&value); err != nil {
		return "", accountMetadata{}, fmt.Errorf("decode auth account: %w", err)
	}
	typeURL, _ := value["@type"].(string)
	if typeURL == "" {
		return "", accountMetadata{}, errors.New("auth account has no @type")
	}
	address := firstStringPath(value,
		[]string{"address"},
		[]string{"base_account", "address"},
		[]string{"base_vesting_account", "base_account", "address"},
	)
	if address == "" {
		return "", accountMetadata{}, fmt.Errorf("auth account %s has no address", typeURL)
	}
	moduleName := ""
	if typeURL == "/cosmos.auth.v1beta1.ModuleAccount" {
		moduleName, _ = value["name"].(string)
		if moduleName == "" {
			return "", accountMetadata{}, fmt.Errorf("module account %s has no name", address)
		}
	}
	return address, accountMetadata{Type: typeURL, ModuleName: moduleName}, nil
}

func firstStringPath(value map[string]any, paths ...[]string) string {
	for _, path := range paths {
		var current any = value
		for _, key := range path {
			m, ok := current.(map[string]any)
			if !ok {
				current = nil
				break
			}
			current = m[key]
		}
		if s, ok := current.(string); ok && s != "" {
			return s
		}
	}
	return ""
}

func advancePage(p *page, seen map[string]struct{}) (string, bool, error) {
	if p == nil {
		return "", false, errors.New("missing pagination object")
	}
	next, done, err := nextPage(*p)
	if err != nil || done {
		return next, done, err
	}
	if _, duplicate := seen[next]; duplicate {
		return "", false, fmt.Errorf("repeated next_key %q", next)
	}
	seen[next] = struct{}{}
	return next, false, nil
}

func nextPage(p page) (string, bool, error) {
	raw := bytes.TrimSpace(p.NextKey)
	if len(raw) == 0 {
		return "", false, errors.New("pagination object has no next_key")
	}
	if bytes.Equal(raw, []byte("null")) || bytes.Equal(raw, []byte(`""`)) {
		return "", true, nil
	}
	var key string
	if err := json.Unmarshal(raw, &key); err != nil {
		return "", false, fmt.Errorf("invalid next_key: %w", err)
	}
	if key == "" {
		return "", true, nil
	}
	decoded, err := base64.StdEncoding.Strict().DecodeString(key)
	if err != nil || len(decoded) == 0 {
		return "", false, errors.New("next_key is not canonical non-empty base64")
	}
	return key, false, nil
}

func verifySupply(owners []owner, expected string) error {
	total := new(big.Int)
	for _, entry := range owners {
		amount, err := parseDecimalAmount(entry.Amount)
		if err != nil || amount.Sign() <= 0 {
			return fmt.Errorf("invalid owner balance %q for %s", entry.Amount, entry.Address)
		}
		total.Add(total, amount)
	}
	expectedAmount, err := parseDecimalAmount(expected)
	if err != nil {
		return fmt.Errorf("invalid supply %q", expected)
	}
	if total.Cmp(expectedAmount) != 0 {
		return fmt.Errorf("owner balance sum %s does not equal supply %s", total, expected)
	}
	return nil
}

func parseDecimalAmount(value string) (*big.Int, error) {
	if value == "" || (len(value) > 1 && value[0] == '0') {
		return nil, errors.New("amount is not canonical decimal")
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return nil, errors.New("amount is not canonical decimal")
		}
	}
	amount, ok := new(big.Int).SetString(value, 10)
	if !ok {
		return nil, errors.New("invalid decimal amount")
	}
	return amount, nil
}

func validateConsensusPubKey(pubKey consensusPubKey) error {
	if pubKey.Type == "" || pubKey.Key == "" {
		return errors.New("missing @type or key")
	}
	decoded, err := base64.StdEncoding.Strict().DecodeString(pubKey.Key)
	if err != nil {
		return fmt.Errorf("invalid base64 key: %w", err)
	}
	if len(decoded) == 0 {
		return errors.New("empty key")
	}
	if pubKey.Type == "/cosmos.crypto.ed25519.PubKey" && len(decoded) != 32 {
		return fmt.Errorf("Ed25519 key is %d bytes, expected 32", len(decoded))
	}
	return nil
}

func isSHA256(value string) bool {
	if len(value) != 64 || strings.ToLower(value) != value {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func isCometHash(value string) bool {
	if len(value) != sha256.Size*2 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func writeAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()
	if err := tmp.Chmod(0o644); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		return err
	}
	directory, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer directory.Close()
	return directory.Sync()
}
