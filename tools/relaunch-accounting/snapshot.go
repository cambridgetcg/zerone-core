package main

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"math/big"
	"net/url"
	"strings"
	"time"
	"unicode"
)

const (
	snapshotSchema  = "zerone-relaunch-snapshot-v3"
	restTrustModel  = "trusted height-pinned REST responses; no Merkle proof binds inventory to checkpoint_app_hash"
	maxInputBytes   = 64 << 20
	maxOwners       = 100000
	maxValidators   = 10000
	maxDepth        = 8
	maxStringBytes  = 2048
	maxAmountDigits = 78
	moduleAccount   = "/cosmos.auth.v1beta1.ModuleAccount"
	baseAccount     = "/cosmos.auth.v1beta1.BaseAccount"
)

// This is the snapshot-v3 wire schema in relaunch-snapshot/main.go:39-100.
// It is deliberately local to this offline tool, not a shared custody API.
// The capture tool can serialize empty owner/validator slices as null.
type snapshot struct {
	Schema     string           `json:"schema"`
	Source     sourceCheckpoint `json:"source"`
	Denom      string           `json:"denom"`
	Supply     string           `json:"supply_uzrn"`
	Owners     []owner          `json:"owners"`
	Validators []validator      `json:"bonded_validators"`
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
	ChainID                            string `json:"chain_id"`
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
	ExcludedPostAnchorAppHash          string `json:"excluded_post_anchor_app_hash"`
	RPCGenesisCanonicalSHA256          string `json:"rpc_genesis_canonical_sha256"`
	DeclaredGenesisFileSHA256          string `json:"declared_genesis_file_sha256,omitempty"`
	RESTTrustModel                     string `json:"rest_trust_model"`
	RPC                                string `json:"rpc"`
	REST                               string `json:"rest"`
}

func validateSnapshot(s snapshot) error {
	if s.Schema != snapshotSchema {
		return errors.New("unsupported snapshot schema; only snapshot-v3 is accepted")
	}
	if s.Denom != "uzrn" {
		return errors.New("snapshot denomination must be uzrn")
	}
	if err := validateSource(s.Source); err != nil {
		return err
	}
	supply, err := parseAmount(s.Supply)
	if err != nil {
		return fmt.Errorf("supply: %w", err)
	}
	if len(s.Owners) > maxOwners || len(s.Validators) > maxValidators {
		return errors.New("snapshot exceeds row limit")
	}
	total := new(big.Int)
	seenOwners := make(map[string]bool, len(s.Owners))
	seenModules := make(map[string]bool)
	for i, row := range s.Owners {
		if !validLabel(row.Address, 256) || !validLabel(row.AccountType, 256) {
			return fmt.Errorf("owner %d has invalid address or account type label", i)
		}
		if seenOwners[row.Address] {
			return fmt.Errorf("duplicate owner at row %d", i)
		}
		seenOwners[row.Address] = true
		if row.AccountType == moduleAccount {
			if !validLabel(row.ModuleName, 128) || seenModules[row.ModuleName] {
				return fmt.Errorf("owner %d has missing, invalid or duplicate module name", i)
			}
			seenModules[row.ModuleName] = true
		} else if row.ModuleName != "" {
			return fmt.Errorf("owner %d has a module name without ModuleAccount evidence", i)
		}
		amount, err := parseAmount(row.Amount)
		if err != nil || amount.Sign() <= 0 {
			return fmt.Errorf("owner %d amount must be a positive bounded canonical decimal", i)
		}
		total.Add(total, amount)
	}
	if total.Cmp(supply) != 0 {
		return errors.New("owner balance sum does not equal supplied supply_uzrn")
	}
	seenValidators := make(map[string]bool, len(s.Validators))
	bonded := new(big.Int)
	for i, row := range s.Validators {
		if !validLabel(row.OperatorAddress, 256) || row.Status != "BOND_STATUS_BONDED" {
			return fmt.Errorf("bonded validator %d has invalid operator or status", i)
		}
		if seenValidators[row.OperatorAddress] {
			return fmt.Errorf("duplicate bonded validator at row %d", i)
		}
		seenValidators[row.OperatorAddress] = true
		if !validLabel(row.ConsensusPubKey.Type, 256) {
			return fmt.Errorf("bonded validator %d has invalid key type", i)
		}
		key, err := base64.StdEncoding.Strict().DecodeString(row.ConsensusPubKey.Key)
		if err != nil || len(key) == 0 || base64.StdEncoding.EncodeToString(key) != row.ConsensusPubKey.Key ||
			(row.ConsensusPubKey.Type == "/cosmos.crypto.ed25519.PubKey" && len(key) != 32) {
			return fmt.Errorf("bonded validator %d has invalid public key encoding", i)
		}
		amount, err := parseAmount(row.Tokens)
		if err != nil || amount.Sign() <= 0 {
			return fmt.Errorf("bonded validator %d tokens must be a positive bounded canonical decimal", i)
		}
		bonded.Add(bonded, amount)
	}
	// These are backing-associated metadata, never a second source of supply.
	if bonded.Cmp(supply) > 0 {
		return errors.New("bonded validator token sum exceeds bank supply")
	}
	return nil
}

func validateSource(s sourceCheckpoint) error {
	if !validLabel(s.ChainID, 128) {
		return errors.New("source chain ID is invalid")
	}
	f, a, h := s.CheckpointStateHeight, s.FinalCommittedBlockHeight, s.HaltTriggerHeight
	if f <= 0 || f > math.MaxInt64-2 || a != f+1 || h != a+1 || s.RPCBlockstoreHeight != h || s.ABCILastAppliedHeight != a {
		return errors.New("source must satisfy F>0, A=F+1, H=A+1, BlockStore=H and ABCI=A")
	}
	if s.FinalCommittedBlockTxs != 0 || s.StagedHaltTriggerBlockTxs != 0 || !s.FinalCommittedBlockCanonical ||
		!s.FinalCommittedBlockHasResults || s.StagedHaltTriggerCommitCanonical || s.StagedHaltTriggerHasBlockResults {
		return errors.New("source canonical, empty-block or results flags contradict snapshot-v3 F/A/H contract")
	}
	for _, hash := range []string{s.CheckpointAppHash, s.FinalCommittedBlockHash, s.StagedHaltTriggerBlockHash,
		s.StagedHaltTriggerPreviousBlockHash, s.StagedHaltTriggerHeaderAppHash, s.ExcludedPostAnchorAppHash} {
		if !isHash(hash) {
			return errors.New("source Comet hashes must each encode 32 bytes in hexadecimal")
		}
	}
	if !strings.EqualFold(s.StagedHaltTriggerPreviousBlockHash, s.FinalCommittedBlockHash) ||
		!strings.EqualFold(s.StagedHaltTriggerHeaderAppHash, s.ExcludedPostAnchorAppHash) ||
		strings.EqualFold(s.StagedHaltTriggerBlockHash, s.FinalCommittedBlockHash) {
		return errors.New("source block links or excluded post-anchor hash are inconsistent")
	}
	anchorTime, err := time.Parse(time.RFC3339Nano, s.FinalCommittedBlockTime)
	if err != nil {
		return errors.New("source anchor time must be RFC3339")
	}
	triggerTime, err := time.Parse(time.RFC3339Nano, s.StagedHaltTriggerBlockTime)
	if err != nil || !triggerTime.After(anchorTime) {
		return errors.New("source trigger time must be RFC3339 and later than anchor time")
	}
	if !isSHA256(s.RPCGenesisCanonicalSHA256) || (s.DeclaredGenesisFileSHA256 != "" && !isSHA256(s.DeclaredGenesisFileSHA256)) {
		return errors.New("source genesis digests must be lowercase SHA-256")
	}
	if s.RESTTrustModel != restTrustModel {
		return errors.New("source must retain the snapshot-v3 trusted REST limitation")
	}
	for _, endpoint := range []string{s.RPC, s.REST} {
		u, err := url.Parse(endpoint)
		if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Hostname() == "" || u.User != nil ||
			u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || !validLabel(endpoint, maxStringBytes) {
			return errors.New("source endpoints must be HTTP(S) base URLs without credentials, query or fragment")
		}
	}
	return nil
}

func parseAmount(value string) (*big.Int, error) {
	if value == "" || len(value) > maxAmountDigits || (len(value) > 1 && value[0] == '0') {
		return nil, errors.New("amount must be a bounded canonical nonnegative decimal integer")
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return nil, errors.New("amount must be a bounded canonical nonnegative decimal integer")
		}
	}
	amount, ok := new(big.Int).SetString(value, 10)
	if !ok {
		return nil, errors.New("invalid decimal integer")
	}
	return amount, nil
}

func validLabel(value string, maximum int) bool {
	if value == "" || len(value) > maximum {
		return false
	}
	for _, r := range value {
		if unicode.IsSpace(r) || unicode.IsControl(r) || r == unicode.ReplacementChar {
			return false
		}
	}
	return true
}

func isHash(value string) bool {
	if len(value) != sha256.Size*2 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func isSHA256(value string) bool {
	return isHash(value) && strings.ToLower(value) == value
}

func digest(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
