package main

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	cmthttp "github.com/cometbft/cometbft/rpc/client/http"
	coretypes "github.com/cometbft/cometbft/rpc/core/types"
	cmttypes "github.com/cometbft/cometbft/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type nodeReport struct {
	RPC             string `json:"rpc"`
	CommitVotes     int    `json:"commit_votes"`
	NilVotes        int    `json:"nil_votes"`
	AbsentVotes     int    `json:"absent_votes"`
	SignedPower     int64  `json:"signed_power"`
	TotalPower      int64  `json:"total_power"`
	CanonicalCommit bool   `json:"canonical_commit"`
}

type txReport struct {
	Hash   string `json:"hash"`
	Height int64  `json:"height"`
	Index  uint32 `json:"index"`
	Code   uint32 `json:"code"`
	Bytes  int    `json:"bytes"`
}

type report struct {
	ChainID         string           `json:"chain_id"`
	Height          int64            `json:"height"`
	BlockID         string           `json:"block_id"`
	PreStateAppHash string           `json:"pre_state_app_hash"`
	PostTxHeight    int64            `json:"post_tx_height,omitempty"`
	PostTxBlockID   string           `json:"post_tx_block_id,omitempty"`
	PostTxAppHash   string           `json:"post_tx_app_hash,omitempty"`
	Nodes           []nodeReport     `json:"nodes"`
	Tx              *txReport        `json:"tx,omitempty"`
	Scheduler       *schedulerReport `json:"scheduler,omitempty"`
	Observer        *observerReport  `json:"observer,omitempty"`
}

func failf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "local consensus verification failed: "+format+"\n", args...)
	os.Exit(1)
}

func parseRPCs(raw string) []string {
	parts := strings.Split(raw, ",")
	result := make([]string, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))
	for _, part := range parts {
		endpoint := strings.TrimSpace(part)
		if endpoint == "" {
			failf("empty RPC endpoint")
		}
		parsed, err := url.Parse(endpoint)
		if err != nil || parsed.Scheme != "http" || parsed.User != nil || parsed.Path != "" || parsed.RawQuery != "" || parsed.Fragment != "" {
			failf("RPC endpoint must be a plain loopback HTTP origin: %q", endpoint)
		}
		host := parsed.Hostname()
		if host != "127.0.0.1" && host != "localhost" && host != "::1" {
			failf("non-loopback RPC endpoint refused: %q", endpoint)
		}
		if parsed.Port() == "" {
			failf("RPC endpoint requires an explicit port: %q", endpoint)
		}
		if _, exists := seen[endpoint]; exists {
			failf("duplicate RPC endpoint: %q", endpoint)
		}
		seen[endpoint] = struct{}{}
		result = append(result, endpoint)
	}
	if len(result) == 0 {
		failf("at least one RPC endpoint is required")
	}
	return result
}

func parseHash(raw string) []byte {
	const sha256Size = 32

	raw = strings.TrimPrefix(strings.TrimPrefix(strings.TrimSpace(raw), "0x"), "0X")
	if raw == "" {
		return nil
	}
	digest, err := hex.DecodeString(raw)
	if err != nil || len(digest) != sha256Size {
		failf("transaction hash must be exactly %d hex bytes", sha256Size)
	}
	return digest
}

func validateExpectedChainID(chainID string) error {
	if chainID == "" {
		return fmt.Errorf("-expect-chain-id is required")
	}
	if strings.TrimSpace(chainID) != chainID {
		return fmt.Errorf("-expect-chain-id must not contain leading or trailing whitespace")
	}
	return nil
}

func verifyExpectedChainID(expected, actual string) error {
	if actual == "" {
		return fmt.Errorf("RPC returned an empty chain ID")
	}
	if actual != expected {
		return fmt.Errorf("chain ID %q does not match required %q", actual, expected)
	}
	return nil
}

func rpcContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 8*time.Second)
}

func verifyBlockHashBinding(blockID cmttypes.BlockID, block *cmttypes.Block) error {
	if block == nil {
		return fmt.Errorf("block is nil")
	}
	blockHash := block.Hash()
	if len(blockHash) == 0 {
		return fmt.Errorf("block header hash is empty")
	}
	if !bytes.Equal(blockHash, blockID.Hash) {
		return fmt.Errorf("block body/header hash %X does not match RPC block ID %X", blockHash, blockID.Hash)
	}
	return nil
}

func main() {
	var (
		rpcList           string
		height            int64
		txHashRaw         string
		expectChainID     string
		expectValidators  int
		expectEqualPower  bool
		schedulerPath     string
		fixtureRoot       string
		observerPath      string
		observerHome      string
		observerPreflight bool
	)
	flag.StringVar(&rpcList, "rpcs", "", "comma-separated loopback RPC origins")
	flag.Int64Var(&height, "height", 0, "height to verify; zero selects a common committed height")
	flag.StringVar(&txHashRaw, "tx", "", "optional transaction hash whose inclusion proof must verify")
	flag.StringVar(&expectChainID, "expect-chain-id", "", "required exact chain ID")
	flag.IntVar(&expectValidators, "expect-validators", 0, "required validator-set size; zero disables the count check")
	flag.BoolVar(&expectEqualPower, "expect-equal-power", false, "require identical voting power for every validator")
	flag.StringVar(&schedulerPath, "scheduler-expect", "", "optional known-key scheduler expectation JSON; verifies post-D state at header D+1 after D+2")
	flag.StringVar(&fixtureRoot, "scheduler-fixture-root", "", "write test-only closed scheduler genesis/expectations inside a fresh marked rehearsal scratch root (no RPC)")
	flag.StringVar(&observerPath, "observer-expect", "", "public observer checkpoint expectation; verifies H+1 across four validators and one observer")
	flag.StringVar(&observerHome, "observer-home", "", "explicit local observer home; reads public genesis/config and zero signing state, never private keys")
	flag.BoolVar(&observerPreflight, "observer-preflight", false, "check fresh observer public genesis/config/zero state without RPC; no replay claim")
	flag.Parse()
	if observerPreflight && (observerPath == "" || rpcList != "" || height != 0) {
		failf("observer preflight requires expectation and no RPC/height")
	}

	var observer *observerExpectation
	var observerMembers map[string]bool
	var observerInitialHeight int64
	if observerPath != "" || observerHome != "" {
		if observerPath == "" || observerHome == "" || schedulerPath != "" || fixtureRoot != "" || txHashRaw != "" || expectValidators != 4 || !expectEqualPower {
			failf("observer mode requires both observer arguments, four equal-power validators and no scheduler/transaction mode")
		}
		var err error
		observer, err = loadObserverExpectation(observerPath, expectChainID)
		if err != nil {
			failf("observer expectation: %v", err)
		}
		observerMembers, observerInitialHeight, err = verifyObserverHome(observerHome, observer)
		if err != nil {
			failf("observer home: %v", err)
		}
		if observerPreflight {
			if err := verifyObserverFreshHome(observerHome); err != nil {
				failf("observer fresh home: %v", err)
			}
			fmt.Println(`{"schema":"zerone.local-observer-preflight/v1","genesis_validators":4,"observer_excluded":true,"last_sign_height":0,"account_keyring_present":false,"state_sync_enabled":false,"scope":"local configuration only; replay not yet observed"}`)
			return
		}
		if height != 0 && height != observer.StateHeight+1 {
			failf("observer checkpoint requires height H+1")
		}
		height = observer.StateHeight + 1
	}

	if schedulerPath != "" || fixtureRoot != "" {
		sdk.GetConfig().SetBech32PrefixForAccount("zrn", "zrnpub")
		if err := validateExpectedChainID(expectChainID); err != nil {
			failf("%v", err)
		}
		if txHashRaw != "" {
			failf("scheduler occurrences are not transactions; -tx cannot be combined with scheduler modes")
		}
	}
	if fixtureRoot != "" {
		if schedulerPath != "" || rpcList != "" || height != 0 {
			failf("fixture writing cannot be combined with RPC verification")
		}
		if err := writeSchedulerFixture(fixtureRoot, expectChainID); err != nil {
			failf("scheduler fixture: %v", err)
		}
		return
	}
	var scheduler *schedulerExpectation
	var schedulerKeysExpected []stateKeyExpectation
	var schedulerDigest string
	if schedulerPath != "" {
		var err error
		scheduler, schedulerDigest, err = loadSchedulerExpectation(schedulerPath, expectChainID)
		if err != nil {
			failf("scheduler expectations: %v", err)
		}
		schedulerKeysExpected, err = schedulerKeys(scheduler, expectChainID)
		if err != nil {
			failf("scheduler expected keys: %v", err)
		}
	}

	if height < 0 {
		failf("height cannot be negative")
	}
	if err := validateExpectedChainID(expectChainID); err != nil {
		failf("%v", err)
	}
	rpcs := parseRPCs(rpcList)
	txHash := parseHash(txHashRaw)
	clients := make([]*cmthttp.HTTP, 0, len(rpcs))
	for _, endpoint := range rpcs {
		client, err := cmthttp.New(endpoint, "/websocket")
		if err != nil {
			failf("create RPC client for %s: %v", endpoint, err)
		}
		clients = append(clients, client)
	}

	minimumHeight := int64(^uint64(0) >> 1)
	statuses := make([]*coretypes.ResultStatus, 0, len(clients))
	for i, client := range clients {
		ctx, cancel := rpcContext()
		status, err := client.Status(ctx)
		cancel()
		if err != nil {
			failf("status from %s: %v", rpcs[i], err)
		}
		if status.SyncInfo.CatchingUp {
			failf("%s is still catching up", rpcs[i])
		}
		chainID := status.NodeInfo.Network
		if err := verifyExpectedChainID(expectChainID, chainID); err != nil {
			failf("%s: %v", rpcs[i], err)
		}
		statuses = append(statuses, status)
		if status.SyncInfo.LatestBlockHeight < minimumHeight {
			minimumHeight = status.SyncInfo.LatestBlockHeight
		}
	}

	observerIndex := -1
	if observer != nil {
		var err error
		observerIndex, err = verifyObserverStatuses(observer, rpcs, statuses, observerMembers)
		if err != nil {
			failf("observer statuses: %v", err)
		}
		if minimumHeight < observer.StateHeight+2 {
			failf("observer checkpoint requires every endpoint past H+1")
		}
	}

	if len(txHash) != 0 {
		ctx, cancel := rpcContext()
		tx, err := clients[0].Tx(ctx, txHash, true)
		cancel()
		if err != nil {
			failf("locate transaction %X through %s: %v", txHash, rpcs[0], err)
		}
		if height != 0 && height != tx.Height {
			failf("requested height %d does not contain transaction at height %d", height, tx.Height)
		}
		if minimumHeight <= tx.Height+1 {
			failf("transaction and post-state heights %d/%d are not canonical on every endpoint yet (minimum latest height %d; wait for height %d)", tx.Height, tx.Height+1, minimumHeight, tx.Height+2)
		}
		height = tx.Height
	}
	if scheduler != nil {
		var err error
		height, err = schedulerAnchorHeight(scheduler, height, minimumHeight)
		if err != nil {
			failf("%v", err)
		}
	}
	if height == 0 {
		// Using the preceding height avoids racing the newest block's canonical
		// commit while independently queried nodes cross a height boundary.
		height = minimumHeight - 1
	}
	if height < 1 {
		failf("no common committed height is available (minimum latest height %d)", minimumHeight)
	}

	result := report{ChainID: expectChainID, Height: height}
	if scheduler != nil {
		result.Scheduler = &schedulerReport{StateHeight: scheduler.StateHeight, AnchorHeight: height, ExpectationSHA256: schedulerDigest, Scope: "known fixture keys only; no range/subspace completeness claim", EmptyOrdinaryBlock: scheduler.RequireEmptyBlock}
	}
	var (
		expectedBlockID cmttypes.BlockID
		expectedAppHash []byte
		expectedTx      []byte
	)

	for i, client := range clients {
		ctx, cancel := rpcContext()
		block, err := client.Block(ctx, &height)
		cancel()
		if err != nil {
			failf("block %d from %s: %v", height, rpcs[i], err)
		}
		if block.Block == nil || block.Block.Height != height || block.Block.ChainID != expectChainID {
			failf("%s returned the wrong block/header at height %d", rpcs[i], height)
		}
		if err := block.Block.ValidateBasic(); err != nil {
			failf("%s returned an internally inconsistent block at height %d: %v", rpcs[i], height, err)
		}
		if err := verifyBlockHashBinding(block.BlockID, block.Block); err != nil {
			failf("%s block binding at height %d: %v", rpcs[i], height, err)
		}

		ctx, cancel = rpcContext()
		commit, err := client.Commit(ctx, &height)
		cancel()
		if err != nil {
			failf("commit %d from %s: %v", height, rpcs[i], err)
		}
		if commit.Commit == nil || commit.Header == nil || commit.Commit.Height != height {
			failf("%s returned an incomplete commit at height %d", rpcs[i], height)
		}
		if err := commit.SignedHeader.ValidateBasic(expectChainID); err != nil {
			failf("%s returned an invalid signed header at height %d: %v", rpcs[i], height, err)
		}
		if !commit.CanonicalCommit {
			failf("%s returned a non-canonical commit at height %d", rpcs[i], height)
		}
		if !commit.Commit.BlockID.Equals(block.BlockID) || !bytes.Equal(commit.Header.Hash(), block.BlockID.Hash) {
			failf("%s commit/header is not bound to block %X", rpcs[i], block.BlockID.Hash)
		}

		page, perPage := 1, 100
		ctx, cancel = rpcContext()
		validators, err := client.Validators(ctx, &height, &page, &perPage)
		cancel()
		if err != nil {
			failf("validator set %d from %s: %v", height, rpcs[i], err)
		}
		if validators.Total != len(validators.Validators) {
			failf("%s validator response was truncated: received %d of %d", rpcs[i], len(validators.Validators), validators.Total)
		}
		if expectValidators != 0 && validators.Total != expectValidators {
			failf("%s validator count %d, expected %d", rpcs[i], validators.Total, expectValidators)
		}
		if observer != nil {
			if err := verifyObserverSet(observer, observerMembers, validators, height); err != nil {
				failf("%s: %v", rpcs[i], err)
			}
			if err := verifyObserverAnchor(observer, block.Block); err != nil {
				failf("%s: %v", rpcs[i], err)
			}
		}
		validatorSet := cmttypes.NewValidatorSet(validators.Validators)
		if !bytes.Equal(validatorSet.Hash(), block.Block.ValidatorsHash) {
			failf("%s validator-set hash does not match block header", rpcs[i])
		}
		if expectEqualPower && len(validators.Validators) > 0 {
			power := validators.Validators[0].VotingPower
			for _, validator := range validators.Validators[1:] {
				if validator.VotingPower != power {
					failf("%s validator set is not equal-power", rpcs[i])
				}
			}
		}
		if err := validatorSet.VerifyCommit(expectChainID, block.BlockID, height, commit.Commit); err != nil {
			failf("cryptographic commit verification from %s: %v", rpcs[i], err)
		}

		powerByAddress := make(map[string]int64, len(validators.Validators))
		var totalPower int64
		for _, validator := range validators.Validators {
			powerByAddress[string(validator.Address)] = validator.VotingPower
			totalPower += validator.VotingPower
		}
		node := nodeReport{RPC: rpcs[i], TotalPower: totalPower, CanonicalCommit: true}
		for _, signature := range commit.Commit.Signatures {
			switch signature.BlockIDFlag {
			case cmttypes.BlockIDFlagCommit:
				node.CommitVotes++
				power, ok := powerByAddress[string(signature.ValidatorAddress)]
				if !ok {
					failf("%s commit contains an unknown validator address", rpcs[i])
				}
				node.SignedPower += power
			case cmttypes.BlockIDFlagNil:
				node.NilVotes++
			case cmttypes.BlockIDFlagAbsent:
				node.AbsentVotes++
			default:
				failf("%s commit contains unknown block-ID flag %d", rpcs[i], signature.BlockIDFlag)
			}
		}
		if node.SignedPower*3 <= node.TotalPower*2 {
			failf("%s commit power %d/%d is not strictly greater than two thirds", rpcs[i], node.SignedPower, node.TotalPower)
		}

		if i == 0 {
			expectedBlockID = block.BlockID
			expectedAppHash = bytes.Clone(block.Block.AppHash)
			result.BlockID = strings.ToUpper(hex.EncodeToString(block.BlockID.Hash))
			result.PreStateAppHash = strings.ToUpper(hex.EncodeToString(block.Block.AppHash))
		} else {
			if !block.BlockID.Equals(expectedBlockID) {
				failf("cross-node block-ID mismatch at height %d: %s differs", height, rpcs[i])
			}
			if !bytes.Equal(block.Block.AppHash, expectedAppHash) {
				failf("cross-node app-hash mismatch at height %d: %s differs", height, rpcs[i])
			}
		}

		if len(txHash) != 0 {
			ctx, cancel = rpcContext()
			tx, err := client.Tx(ctx, txHash, true)
			cancel()
			if err != nil {
				failf("transaction %X from %s: %v", txHash, rpcs[i], err)
			}
			if tx.Height != height || tx.TxResult.Code != 0 || !bytes.Equal(tx.Hash, txHash) {
				failf("%s returned a failed or mismatched transaction result", rpcs[i])
			}
			if int(tx.Index) >= len(block.Block.Data.Txs) || !bytes.Equal(block.Block.Data.Txs[tx.Index], tx.Tx) {
				failf("%s transaction bytes are not at the reported block index", rpcs[i])
			}
			if !bytes.Equal(tx.Tx.Hash(), txHash) || !bytes.Equal(tx.Proof.Data, tx.Tx) {
				failf("%s transaction hash/proof data mismatch", rpcs[i])
			}
			if err := tx.Proof.Validate(block.Block.DataHash); err != nil {
				failf("transaction Merkle proof from %s: %v", rpcs[i], err)
			}
			if i == 0 {
				expectedTx = bytes.Clone(tx.Tx)
				result.Tx = &txReport{Hash: strings.ToUpper(hex.EncodeToString(tx.Hash)), Height: tx.Height, Index: tx.Index, Code: tx.TxResult.Code, Bytes: len(tx.Tx)}
			} else if !bytes.Equal(tx.Tx, expectedTx) {
				failf("cross-node transaction-byte mismatch: %s differs", rpcs[i])
			}
		}

		if scheduler != nil {
			proofs, err := verifySchedulerNode(client, rpcs[i], scheduler, schedulerKeysExpected, block.Block)
			if err != nil {
				failf("scheduler state from %s: %v", rpcs[i], err)
			}
			result.Scheduler.Nodes = append(result.Scheduler.Nodes, proofs)
		}
		result.Nodes = append(result.Nodes, node)
	}

	if len(txHash) != 0 {
		postHeight := height + 1
		var (
			expectedPostBlockID cmttypes.BlockID
			expectedPostAppHash []byte
		)
		for i, client := range clients {
			ctx, cancel := rpcContext()
			postBlock, err := client.Block(ctx, &postHeight)
			cancel()
			if err != nil {
				failf("post-state block %d from %s: %v", postHeight, rpcs[i], err)
			}
			if postBlock.Block == nil || postBlock.Block.Height != postHeight || postBlock.Block.ChainID != expectChainID {
				failf("%s returned the wrong post-state block/header at height %d", rpcs[i], postHeight)
			}
			if err := postBlock.Block.ValidateBasic(); err != nil {
				failf("%s returned an internally inconsistent post-state block at height %d: %v", rpcs[i], postHeight, err)
			}
			if err := verifyBlockHashBinding(postBlock.BlockID, postBlock.Block); err != nil {
				failf("%s post-state block binding at height %d: %v", rpcs[i], postHeight, err)
			}
			if !postBlock.Block.LastBlockID.Equals(expectedBlockID) {
				failf("%s post-state block does not link to transaction block %X", rpcs[i], expectedBlockID.Hash)
			}

			ctx, cancel = rpcContext()
			postCommit, err := client.Commit(ctx, &postHeight)
			cancel()
			if err != nil {
				failf("post-state commit %d from %s: %v", postHeight, rpcs[i], err)
			}
			if postCommit.Commit == nil || postCommit.Header == nil || postCommit.Commit.Height != postHeight || !postCommit.CanonicalCommit {
				failf("%s returned an incomplete or non-canonical post-state commit at height %d", rpcs[i], postHeight)
			}
			if err := postCommit.SignedHeader.ValidateBasic(expectChainID); err != nil {
				failf("%s returned an invalid post-state signed header at height %d: %v", rpcs[i], postHeight, err)
			}
			if !postCommit.Commit.BlockID.Equals(postBlock.BlockID) || !bytes.Equal(postCommit.Header.Hash(), postBlock.BlockID.Hash) {
				failf("%s post-state commit/header is not bound to block %X", rpcs[i], postBlock.BlockID.Hash)
			}

			page, perPage := 1, 100
			ctx, cancel = rpcContext()
			postValidators, err := client.Validators(ctx, &postHeight, &page, &perPage)
			cancel()
			if err != nil {
				failf("post-state validator set %d from %s: %v", postHeight, rpcs[i], err)
			}
			if postValidators.Total != len(postValidators.Validators) {
				failf("%s post-state validator response was truncated: received %d of %d", rpcs[i], len(postValidators.Validators), postValidators.Total)
			}
			if expectValidators != 0 && postValidators.Total != expectValidators {
				failf("%s post-state validator count %d, expected %d", rpcs[i], postValidators.Total, expectValidators)
			}
			postValidatorSet := cmttypes.NewValidatorSet(postValidators.Validators)
			if !bytes.Equal(postValidatorSet.Hash(), postBlock.Block.ValidatorsHash) {
				failf("%s post-state validator-set hash does not match block header", rpcs[i])
			}
			if expectEqualPower && len(postValidators.Validators) > 0 {
				power := postValidators.Validators[0].VotingPower
				for _, validator := range postValidators.Validators[1:] {
					if validator.VotingPower != power {
						failf("%s post-state validator set is not equal-power", rpcs[i])
					}
				}
			}
			if err := postValidatorSet.VerifyCommit(expectChainID, postBlock.BlockID, postHeight, postCommit.Commit); err != nil {
				failf("cryptographic post-state commit verification from %s: %v", rpcs[i], err)
			}

			if i == 0 {
				expectedPostBlockID = postBlock.BlockID
				expectedPostAppHash = bytes.Clone(postBlock.Block.AppHash)
				result.PostTxHeight = postHeight
				result.PostTxBlockID = strings.ToUpper(hex.EncodeToString(postBlock.BlockID.Hash))
				result.PostTxAppHash = strings.ToUpper(hex.EncodeToString(postBlock.Block.AppHash))
			} else {
				if !postBlock.BlockID.Equals(expectedPostBlockID) {
					failf("cross-node post-state block-ID mismatch at height %d: %s differs", postHeight, rpcs[i])
				}
				if !bytes.Equal(postBlock.Block.AppHash, expectedPostAppHash) {
					failf("cross-node post-state app-hash mismatch at height %d: %s differs", postHeight, rpcs[i])
				}
			}
		}
	}

	if observer != nil {
		var err error
		result.Observer, err = verifyObserverCheckpoint(clients[observerIndex], observer, observerInitialHeight)
		if err != nil {
			failf("observer checkpoint: %v", err)
		}
		// Recheck the local boundary after RPC reads; no account keyring or vote
		// may have appeared while verification was in flight.
		if _, _, err := verifyObserverHome(observerHome, observer); err != nil {
			failf("observer home after verification: %v", err)
		}
	}

	encoded, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		failf("encode report: %v", err)
	}
	fmt.Println(string(encoded))
}
