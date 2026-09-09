// verify-checkpoint verifies public checkpoint bytes without opening a node home.
// Signature validity against this operator-observed pin is not independent trust.
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"syscall"
	"time"
	"unicode/utf8"

	cmtjson "github.com/cometbft/cometbft/libs/json"
	core "github.com/cometbft/cometbft/rpc/core/types"
	cmt "github.com/cometbft/cometbft/types"
)

const (
	maxInputBytes     = 1 << 20
	chainID           = "zerone-1"
	checkpointSchema  = "zerone-1-observer-checkpoint/v1"
	trustSeconds      = 604800
	observedAddress   = "93DBC1A1D9A07E67423B193395C9EB6F1E41B844"
	observedPublicKey = "LA944mpk44k8Jr9EN/uOzSSf8G+Ng2LiIvf9FJBkl80="
	observedPower     = 11111
)

type checkpoint struct {
	Schema             string          `json:"schema"`
	ChainID            string          `json:"chain_id"`
	Height             int64           `json:"height"`
	BlockHash          string          `json:"block_hash"`
	HeaderTime         string          `json:"header_time"`
	TrustPeriodSeconds int64           `json:"trust_period_seconds"`
	SignedHeader       json.RawMessage `json:"signed_header"`
	ValidatorSet       json.RawMessage `json:"validator_set"`
}

type receipt struct {
	Schema                           string `json:"schema"`
	Result                           string `json:"result"`
	CheckpointSHA256                 string `json:"checkpoint_sha256"`
	ChainID                          string `json:"chain_id"`
	Height                           int64  `json:"height"`
	BlockHash                        string `json:"block_hash"`
	HeaderTime                       string `json:"header_time"`
	VerifiedAt                       string `json:"verified_at"`
	ExpiresAt                        string `json:"expires_at"`
	Expired                          bool   `json:"expired"`
	AllowExpired                     bool   `json:"allow_expired"`
	CanonicalHeaderHashVerified      bool   `json:"canonical_header_hash_verified"`
	ValidatorSetHashVerified         bool   `json:"validator_set_hash_verified"`
	CommitSignaturesVerified         bool   `json:"commit_signatures_verified"`
	OperatorObservedValidatorMatched bool   `json:"operator_observed_validator_matched"`
	IndependentTrustAnchor           bool   `json:"independent_trust_anchor"`
	TrustDisclosure                  string `json:"trust_disclosure"`
}

// Check duplicates before decoding: Go's ordinary decoder accepts last-write wins.
func validateJSON(raw []byte) error {
	if len(raw) == 0 || len(raw) > maxInputBytes || !utf8.Valid(raw) {
		return errors.New("checkpoint size or UTF-8 invalid")
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	tokens := 0
	var value func(int) error
	value = func(depth int) error {
		if depth > 32 {
			return errors.New("JSON nesting bound exceeded")
		}
		tokens++
		if tokens > 32768 {
			return errors.New("JSON token bound exceeded")
		}
		t, err := d.Token()
		if err != nil {
			return err
		}
		delim, ok := t.(json.Delim)
		if !ok {
			return nil
		}
		switch delim {
		case '{':
			seen := map[string]bool{}
			for d.More() {
				k, err := d.Token()
				if err != nil {
					return err
				}
				key, ok := k.(string)
				if !ok || seen[key] {
					return errors.New("duplicate or invalid JSON key")
				}
				seen[key] = true
				if err := value(depth + 1); err != nil {
					return err
				}
			}
		case '[':
			for d.More() {
				if err := value(depth + 1); err != nil {
					return err
				}
			}
		default:
			return errors.New("unexpected JSON delimiter")
		}
		_, err = d.Token()
		return err
	}
	if err := value(0); err != nil {
		return err
	}
	if _, err := d.Token(); err != io.EOF {
		return errors.New("trailing JSON input")
	}
	return nil
}

func object(raw json.RawMessage, fields string) (map[string]json.RawMessage, error) {
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(raw, &obj); err != nil {
		return nil, err
	}
	if obj == nil {
		return nil, errors.New("expected JSON object")
	}
	allowed := map[string]bool{}
	for _, s := range strings.Fields(fields) {
		allowed[s] = true
	}
	if len(obj) != len(allowed) {
		return nil, errors.New("missing or unknown checkpoint field")
	}
	for k := range obj {
		if !allowed[k] {
			return nil, fmt.Errorf("unknown or aliased checkpoint field %q", k)
		}
	}
	return obj, nil
}

func blockIDShape(raw json.RawMessage) error {
	o, err := object(raw, "hash parts")
	if err != nil {
		return err
	}
	_, err = object(o["parts"], "total hash")
	return err
}

func validateShape(raw []byte) error {
	top, err := object(raw, "schema chain_id height block_hash header_time trust_period_seconds signed_header validator_set")
	if err != nil {
		return err
	}
	sh, err := object(top["signed_header"], "header commit")
	if err != nil {
		return err
	}
	h, err := object(sh["header"], "version chain_id height time last_block_id last_commit_hash data_hash validators_hash next_validators_hash consensus_hash app_hash last_results_hash evidence_hash proposer_address")
	if err != nil {
		return err
	}
	// Comet omits the zero application protocol version in its canonical JSON.
	var version map[string]json.RawMessage
	if err = json.Unmarshal(h["version"], &version); err != nil {
		return err
	}
	versionFields := "block"
	if _, present := version["app"]; present {
		versionFields = "block app"
	}
	if _, err = object(h["version"], versionFields); err != nil {
		return err
	}
	if err = blockIDShape(h["last_block_id"]); err != nil {
		return err
	}
	c, err := object(sh["commit"], "height round block_id signatures")
	if err != nil {
		return err
	}
	if err = blockIDShape(c["block_id"]); err != nil {
		return err
	}
	var signatures []json.RawMessage
	if err = json.Unmarshal(c["signatures"], &signatures); err != nil {
		return err
	}
	if len(signatures) != 1 {
		return errors.New("checkpoint requires sole observed validator signature")
	}
	for _, s := range signatures {
		if _, err = object(s, "block_id_flag validator_address timestamp signature"); err != nil {
			return err
		}
	}
	v, err := object(top["validator_set"], "block_height validators count total")
	if err != nil {
		return err
	}
	var validators []json.RawMessage
	if err = json.Unmarshal(v["validators"], &validators); err != nil {
		return err
	}
	if len(validators) != 1 {
		return errors.New("checkpoint requires sole observed validator")
	}
	for _, v := range validators {
		obj, err := object(v, "address pub_key voting_power proposer_priority")
		if err != nil {
			return err
		}
		if _, err = object(obj["pub_key"], "type value"); err != nil {
			return err
		}
	}
	return nil
}

func verify(raw []byte, now time.Time, allowExpired bool) (receipt, error) {
	var out receipt
	if err := validateJSON(raw); err != nil {
		return out, err
	}
	if err := validateShape(raw); err != nil {
		return out, err
	}
	var cp checkpoint
	if err := json.Unmarshal(raw, &cp); err != nil {
		return out, err
	}
	if cp.Schema != checkpointSchema || cp.ChainID != chainID || cp.Height <= 0 || cp.TrustPeriodSeconds != trustSeconds {
		return out, errors.New("checkpoint contract mismatch")
	}
	if !regexp.MustCompile(`^[0-9A-F]{64}$`).MatchString(cp.BlockHash) {
		return out, errors.New("block_hash must be uppercase SHA256 hex")
	}
	var sh cmt.SignedHeader
	var vals core.ResultValidators
	if err := cmtjson.Unmarshal(cp.SignedHeader, &sh); err != nil {
		return out, err
	}
	if err := cmtjson.Unmarshal(cp.ValidatorSet, &vals); err != nil {
		return out, err
	}
	if sh.Header == nil || sh.Commit == nil {
		return out, errors.New("missing signed header")
	}
	if err := sh.ValidateBasic(chainID); err != nil {
		return out, err
	}
	if sh.Height != cp.Height || sh.Commit.Height != cp.Height {
		return out, errors.New("header or commit height mismatch")
	}
	if strings.ToUpper(hex.EncodeToString(sh.Header.Hash())) != cp.BlockHash || !bytes.Equal(sh.Header.Hash(), sh.Commit.BlockID.Hash) {
		return out, errors.New("canonical header/block hash mismatch")
	}
	if sh.Time.UTC().Format(time.RFC3339Nano) != cp.HeaderTime {
		return out, errors.New("header_time must exactly match canonical header timestamp")
	}
	if sh.Time.After(now.Add(10 * time.Second)) {
		return out, errors.New("checkpoint header is in the future")
	}
	expires := sh.Time.Add(trustSeconds * time.Second)
	expired := !now.Before(expires)
	if expired && !allowExpired {
		return out, errors.New("checkpoint trust period has expired")
	}
	if vals.BlockHeight != cp.Height || vals.Count != 1 || vals.Total != 1 || len(vals.Validators) != 1 {
		return out, errors.New("incomplete or wrong-height validator set")
	}
	v := vals.Validators[0]
	if v == nil {
		return out, errors.New("nil validator")
	}
	if err := v.ValidateBasic(); err != nil {
		return out, err
	}
	expectedKey, _ := base64.StdEncoding.DecodeString(observedPublicKey)
	if v.PubKey.Type() != "ed25519" || fmt.Sprintf("%X", v.Address) != observedAddress || !bytes.Equal(v.PubKey.Bytes(), expectedKey) || !bytes.Equal(v.PubKey.Address(), v.Address) || v.VotingPower != observedPower {
		return out, errors.New("operator-observed validator pin mismatch")
	}
	// Canonical vote sign bytes omit ValidatorAddress. Pin the signature's
	// declared address as well as the validator set; do not inherit a library's
	// assumption that index lookup alone authenticates this unsigned field.
	if len(sh.Commit.Signatures) != 1 || sh.Commit.Signatures[0].BlockIDFlag != cmt.BlockIDFlagCommit ||
		!bytes.Equal(sh.Commit.Signatures[0].ValidatorAddress, v.Address) {
		return out, errors.New("commit signature does not identify the pinned sole validator")
	}
	set := cmt.NewValidatorSet(vals.Validators)
	if err := set.ValidateBasic(); err != nil {
		return out, err
	}
	if !bytes.Equal(set.Hash(), sh.ValidatorsHash) {
		return out, errors.New("validator-set hash mismatch")
	}
	if err := set.VerifyCommit(chainID, sh.Commit.BlockID, cp.Height, sh.Commit); err != nil {
		return out, err
	}
	hash := sha256.Sum256(raw)
	out = receipt{Schema: "zerone-1-observer-checkpoint-verification/v1", Result: "PASS", CheckpointSHA256: hex.EncodeToString(hash[:]), ChainID: chainID, Height: cp.Height, BlockHash: cp.BlockHash, HeaderTime: cp.HeaderTime, VerifiedAt: now.UTC().Format(time.RFC3339Nano), ExpiresAt: expires.UTC().Format(time.RFC3339Nano), Expired: expired, AllowExpired: allowExpired, CanonicalHeaderHashVerified: true, ValidatorSetHashVerified: true, CommitSignaturesVerified: true, OperatorObservedValidatorMatched: true, IndependentTrustAnchor: false, TrustDisclosure: "Signatures are verified against the explicitly pinned operator-observed sole validator. Same-upstream RPC aliases are not independent witnesses; this does not establish independent trust or repair signer custody."}
	return out, nil
}

func run(args []string, stdout io.Writer) error {
	flags := flag.NewFlagSet("verify-checkpoint", flag.ContinueOnError)
	path := flags.String("checkpoint", "", "public checkpoint JSON file")
	allowExpired := flags.Bool("allow-expired", false, "verify expired evidence for existing-state resume; never authorize fresh bootstrap")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *path == "" || flags.NArg() != 0 {
		return errors.New("usage: verify-checkpoint --checkpoint FILE [--allow-expired]")
	}
	// This executable targets Linux; Darwin is also supported for local tests.
	// Nonblocking prevents a FIFO from stalling before the regular-file check.
	fd, err := syscall.Open(*path, syscall.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK|syscall.O_CLOEXEC, 0)
	if err != nil {
		return err
	}
	f := os.NewFile(uintptr(fd), *path)
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() || info.Size() > maxInputBytes {
		return errors.New("checkpoint must be a bounded regular file")
	}
	raw, err := io.ReadAll(io.LimitReader(f, maxInputBytes+1))
	if err != nil {
		return err
	}
	after, err := f.Stat()
	if err != nil {
		return err
	}
	pathAfter, err := os.Lstat(*path)
	if err != nil {
		return err
	}
	if !sameFileState(info, after) || !sameFileState(info, pathAfter) {
		return errors.New("checkpoint file changed while reading")
	}
	out, err := verify(raw, time.Now().UTC(), *allowExpired)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

func sameFileState(before, after os.FileInfo) bool {
	return after.Mode().IsRegular() && os.SameFile(before, after) && before.Mode() == after.Mode() &&
		before.Size() == after.Size() && before.ModTime().Equal(after.ModTime())
}

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "checkpoint verification failed:", err)
		os.Exit(1)
	}
}
