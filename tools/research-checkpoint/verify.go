// research-checkpoint verifies public evidence offline. It never signs or sends.
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	abci "github.com/cometbft/cometbft/abci/types"
	cmcrypto "github.com/cometbft/cometbft/crypto"
	"github.com/cometbft/cometbft/crypto/ed25519"
	cmtjson "github.com/cometbft/cometbft/libs/json"
	rpctypes "github.com/cometbft/cometbft/rpc/core/types"
	cmttypes "github.com/cometbft/cometbft/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	"github.com/cosmos/cosmos-sdk/types/bech32"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	proto "github.com/cosmos/gogoproto/proto"
	curve "github.com/decred/dcrd/dcrec/secp256k1/v4"
)

const chainID = "zerone-dev-1"
const originalGenesisSHA256 = "f8b4570b37fd41d5b63c02fbae1f89225c197c75315e8629650716f6e5f79256"
const originalValidatorKey = "C7jBQ2qhJ6uORoLnFoshmLIeKOeOtDnDlCsWdpgsJBg="

var memoPattern = regexp.MustCompile(`\Azerone:research:v1:([0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}):([1-9][0-9]*):([0-9a-f]{64}):([0-9a-f]{64})\z`)
var hashPattern = regexp.MustCompile(`\A[0-9a-fA-F]{64}\z`)

type Expected struct {
	Height                int64
	TxHash, Memo, Sender  string
	AccountNumber, Gas    uint64
	Fee                   string
	RequireExecutionProof bool
}

type Evidence struct {
	Genesis, Tx, Commit, Validators          []byte
	BlockResults, NextCommit, NextValidators []byte
}

type Receipt struct {
	Schema      string            `json:"schema"`
	Verified    bool              `json:"verified"`
	ChainID     string            `json:"chain_id"`
	Height      string            `json:"height"`
	InputSHA256 map[string]string `json:"input_sha256"`
	Trust       map[string]string `json:"trust"`
	Transaction map[string]any    `json:"transaction"`
	Checkpoint  map[string]any    `json:"checkpoint"`
	Execution   map[string]any    `json:"execution"`
	Limits      []string          `json:"limits"`
}

func digest(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }

// The public verifier accepts only the original, independently pinned bootstrap
// genesis. A freshly fetched validator response is evidence, never a trust root.
func originalAnchor(raw []byte) (cmcrypto.PubKey, error) {
	if digest(raw) != originalGenesisSHA256 {
		return nil, errors.New("original genesis SHA-256 mismatch")
	}
	var g struct {
		ChainID  string `json:"chain_id"`
		AppState struct {
			Genutil struct {
				GenTxs []struct {
					Body struct {
						Messages []struct {
							Type   string `json:"@type"`
							Pubkey struct {
								Type string `json:"@type"`
								Key  string `json:"key"`
							} `json:"pubkey"`
						} `json:"messages"`
					} `json:"body"`
				} `json:"gen_txs"`
			} `json:"genutil"`
		} `json:"app_state"`
	}
	if err := json.Unmarshal(raw, &g); err != nil {
		return nil, err
	}
	if g.ChainID != chainID || len(g.AppState.Genutil.GenTxs) != 1 || len(g.AppState.Genutil.GenTxs[0].Body.Messages) != 1 {
		return nil, errors.New("unexpected original genesis validator layout")
	}
	m := g.AppState.Genutil.GenTxs[0].Body.Messages[0]
	if m.Type != "/cosmos.staking.v1beta1.MsgCreateValidator" || m.Pubkey.Type != "/cosmos.crypto.ed25519.PubKey" || m.Pubkey.Key != originalValidatorKey {
		return nil, errors.New("original validator key mismatch")
	}
	b, err := base64.StdEncoding.DecodeString(m.Pubkey.Key)
	if err != nil || len(b) != ed25519.PubKeySize {
		return nil, errors.New("invalid original public key")
	}
	return ed25519.PubKey(b), nil
}

func rpcResult(raw []byte, out any) error {
	if err := strictJSON(raw); err != nil {
		return err
	}
	var e struct {
		JSONRPC string          `json:"jsonrpc"`
		Result  json.RawMessage `json:"result"`
		Error   json.RawMessage `json:"error"`
	}
	if err := json.Unmarshal(raw, &e); err != nil {
		return err
	}
	if e.JSONRPC != "2.0" || len(e.Result) == 0 || bytes.Equal(e.Result, []byte("null")) || (len(e.Error) != 0 && !bytes.Equal(e.Error, []byte("null"))) {
		return errors.New("expected successful JSON-RPC 2.0 result")
	}
	return cmtjson.Unmarshal(e.Result, out)
}

func signedHeader(raw, validators []byte, anchor cmcrypto.PubKey, height int64) (*rpctypes.ResultCommit, error) {
	var c rpctypes.ResultCommit
	var v rpctypes.ResultValidators
	if err := rpcResult(raw, &c); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}
	if err := rpcResult(validators, &v); err != nil {
		return nil, fmt.Errorf("validators: %w", err)
	}
	if c.Header == nil || c.Commit == nil {
		return nil, errors.New("missing signed header")
	}
	if err := c.Header.ValidateBasic(); err != nil {
		return nil, fmt.Errorf("header: %w", err)
	}
	if err := c.Commit.ValidateBasic(); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}
	if c.Header.ChainID != chainID || c.Header.Height != height || c.Commit.Height != height || height <= 0 {
		return nil, errors.New("signed header chain or height mismatch")
	}
	if !bytes.Equal(c.Header.Hash(), c.Commit.BlockID.Hash) {
		return nil, errors.New("header hash does not match commit block ID")
	}
	if v.BlockHeight != height || v.Count != 1 || v.Total != 1 || len(v.Validators) != 1 || v.Validators[0] == nil {
		return nil, errors.New("expected complete original single-validator set")
	}
	x := v.Validators[0]
	if x.PubKey == nil || !x.PubKey.Equals(anchor) || !bytes.Equal(x.Address, anchor.Address()) || x.VotingPower <= 0 || x.VotingPower > cmttypes.MaxTotalVotingPower {
		return nil, errors.New("validator differs from original trust anchor")
	}
	vs := cmttypes.NewValidatorSet(v.Validators)
	if !bytes.Equal(vs.Hash(), c.Header.ValidatorsHash) || !bytes.Equal(c.Header.ProposerAddress, anchor.Address()) {
		return nil, errors.New("signed validator set or proposer mismatch")
	}
	if len(c.Commit.Signatures) != 1 || c.Commit.Signatures[0].BlockIDFlag != cmttypes.BlockIDFlagCommit || !bytes.Equal(c.Commit.Signatures[0].ValidatorAddress, anchor.Address()) {
		return nil, errors.New("missing original validator block signature")
	}
	if err := vs.VerifyCommitLightAllSignatures(chainID, c.Commit.BlockID, height, c.Commit); err != nil {
		return nil, fmt.Errorf("commit signature: %w", err)
	}
	return &c, nil
}

// Canonical re-encoding refuses fields and encodings this decoder normalizes.
// Any retains unknown wire bytes, so accepted Any wrappers are checked below.
func canonicalProto(raw []byte, msg proto.Message) error {
	if err := proto.Unmarshal(raw, msg); err != nil {
		return err
	}
	b, err := proto.Marshal(msg)
	if err != nil {
		return err
	}
	if !bytes.Equal(raw, b) {
		return errors.New("noncanonical or unknown protobuf fields")
	}
	return nil
}

func verifyTransaction(raw []byte, e Expected) (map[string]any, map[string]any, error) {
	fail := func(s string) (map[string]any, map[string]any, error) { return nil, nil, errors.New(s) }
	m := memoPattern.FindStringSubmatch(e.Memo)
	if m == nil || len(e.Memo) > 256 {
		return fail("invalid checkpoint memo")
	}
	count, err := strconv.ParseUint(m[2], 10, 64)
	if err != nil || count > 1000 {
		return fail("checkpoint entry count outside journal bound")
	}
	if e.Gas == 0 || !regexp.MustCompile(`\A[1-9][0-9]*\z`).MatchString(e.Fee) || len(e.Fee) > 78 {
		return fail("invalid expected gas or fee")
	}
	var tx txtypes.TxRaw
	var body txtypes.TxBody
	var auth txtypes.AuthInfo
	if err := canonicalProto(raw, &tx); err != nil {
		return nil, nil, fmt.Errorf("TxRaw: %w", err)
	}
	if err := canonicalProto(tx.BodyBytes, &body); err != nil {
		return nil, nil, fmt.Errorf("TxBody: %w", err)
	}
	if err := canonicalProto(tx.AuthInfoBytes, &auth); err != nil {
		return nil, nil, fmt.Errorf("AuthInfo: %w", err)
	}
	if len(body.Messages) != 1 || body.Messages[0] == nil || len(body.Messages[0].XXX_unrecognized) != 0 || body.Messages[0].TypeUrl != "/cosmos.bank.v1beta1.MsgSend" || body.Memo != e.Memo || body.TimeoutHeight != 0 || body.Unordered || body.TimeoutTimestamp != nil || len(body.ExtensionOptions) != 0 || len(body.NonCriticalExtensionOptions) != 0 {
		return fail("transaction body differs from checkpoint intent")
	}
	var send banktypes.MsgSend
	if err := canonicalProto(body.Messages[0].Value, &send); err != nil {
		return nil, nil, fmt.Errorf("MsgSend: %w", err)
	}
	if send.FromAddress != e.Sender || send.ToAddress != e.Sender || len(send.Amount) != 1 || send.Amount[0].Denom != "uzrn" || send.Amount[0].Amount.IsNil() || send.Amount[0].Amount.String() != "1" {
		return fail("expected exactly one self-send of 1uzrn")
	}
	prefix, address, err := bech32.DecodeAndConvert(e.Sender)
	if err != nil || prefix != "zrn" || len(address) != 20 || e.Sender != strings.ToLower(e.Sender) {
		return fail("invalid expected sender")
	}
	if len(auth.SignerInfos) != 1 || auth.SignerInfos[0] == nil || len(tx.Signatures) != 1 || len(tx.Signatures[0]) != 64 || auth.Fee == nil || auth.Tip != nil {
		return fail("expected one ordinary direct signer and fee")
	}
	si := auth.SignerInfos[0]
	if si.ModeInfo == nil || si.ModeInfo.GetSingle() == nil || si.ModeInfo.GetSingle().Mode != signing.SignMode_SIGN_MODE_DIRECT || si.PublicKey == nil || len(si.PublicKey.XXX_unrecognized) != 0 || si.PublicKey.TypeUrl != "/cosmos.crypto.secp256k1.PubKey" {
		return fail("expected direct secp256k1 signer")
	}
	var pk secp256k1.PubKey
	if err := canonicalProto(si.PublicKey.Value, &pk); err != nil {
		return nil, nil, fmt.Errorf("sender public key: %w", err)
	}
	if len(pk.Key) != 33 {
		return fail("invalid compressed sender public key")
	}
	if _, err := curve.ParsePubKey(pk.Key); err != nil {
		return fail("invalid sender curve point")
	}
	if !bytes.Equal(pk.Address(), address) {
		return fail("sender address does not derive from signing key")
	}
	f := auth.Fee
	if f.GasLimit != e.Gas || f.Payer != "" || f.Granter != "" || len(f.Amount) != 1 || f.Amount[0].Denom != "uzrn" || f.Amount[0].Amount.IsNil() || f.Amount[0].Amount.String() != e.Fee {
		return fail("fee differs from explicit expected fee")
	}
	doc, err := proto.Marshal(&txtypes.SignDoc{BodyBytes: tx.BodyBytes, AuthInfoBytes: tx.AuthInfoBytes, ChainId: chainID, AccountNumber: e.AccountNumber})
	if err != nil {
		return nil, nil, err
	}
	if !pk.VerifySignature(doc, tx.Signatures[0]) {
		return fail("sender signature does not verify for supplied chain and account number")
	}
	return map[string]any{
		"txhash": strings.ToUpper(digest(raw)), "byte_length": len(raw), "sender": e.Sender, "recipient": e.Sender,
		"amount_uzrn": "1", "fee_uzrn": e.Fee, "gas_limit": strconv.FormatUint(e.Gas, 10),
		"account_number": strconv.FormatUint(e.AccountNumber, 10), "sequence": strconv.FormatUint(si.Sequence, 10),
		"sign_doc_sha256": digest(doc), "public_key_base64": base64.StdEncoding.EncodeToString(pk.Key), "signature_base64": base64.StdEncoding.EncodeToString(tx.Signatures[0]), "signature_verified": true,
	}, map[string]any{"collection_id": m[1], "entry_count": count, "export_sha256": m[3], "head_sha256": m[4], "memo": e.Memo}, nil
}

func verify(e Evidence, want Expected) (*Receipt, error) {
	anchor, err := originalAnchor(e.Genesis)
	if err != nil {
		return nil, err
	}
	return verifyAnchored(e, want, anchor)
}

// Tests use fresh disposable keys through this internal boundary; main always
// derives its anchor from the immutable original genesis above.
func verifyAnchored(e Evidence, want Expected, anchor cmcrypto.PubKey) (*Receipt, error) {
	if want.Height <= 0 || !hashPattern.MatchString(want.TxHash) {
		return nil, errors.New("explicit positive height and 64-hex transaction hash required")
	}
	c, err := signedHeader(e.Commit, e.Validators, anchor, want.Height)
	if err != nil {
		return nil, err
	}
	var tx rpctypes.ResultTx
	if err := rpcResult(e.Tx, &tx); err != nil {
		return nil, fmt.Errorf("transaction RPC: %w", err)
	}
	if tx.Height != want.Height || !strings.EqualFold(digest(tx.Tx), want.TxHash) || !bytes.Equal(tx.Hash, tx.Tx.Hash()) {
		return nil, errors.New("transaction hash or height mismatch")
	}
	if !bytes.Equal(tx.Proof.Data, tx.Tx) || tx.Proof.Proof.Index != int64(tx.Index) || tx.Proof.Proof.Total <= 0 || tx.Proof.Proof.Total > 100000 || tx.Proof.Proof.Index < 0 || tx.Proof.Proof.Index >= tx.Proof.Proof.Total {
		return nil, errors.New("transaction proof data, index or total mismatch")
	}
	if err := tx.Proof.Validate(c.Header.DataHash); err != nil {
		return nil, fmt.Errorf("transaction inclusion: %w", err)
	}
	transaction, checkpoint, err := verifyTransaction(tx.Tx, want)
	if err != nil {
		return nil, err
	}
	transaction["index"] = tx.Index
	transaction["header_hash"] = strings.ToUpper(hex.EncodeToString(c.Header.Hash()))
	transaction["data_hash"] = strings.ToUpper(hex.EncodeToString(c.Header.DataHash))
	if tx.TxResult.Code != 0 {
		return nil, fmt.Errorf("endpoint reports unsuccessful execution code %d", tx.TxResult.Code)
	}
	execution := map[string]any{"code": tx.TxResult.Code, "status": "endpoint_observation_only", "gas_wanted": strconv.FormatInt(tx.TxResult.GasWanted, 10), "gas_used": strconv.FormatInt(tx.TxResult.GasUsed, 10)}
	optional := len(e.BlockResults) != 0 || len(e.NextCommit) != 0 || len(e.NextValidators) != 0
	if want.RequireExecutionProof && !optional {
		return nil, errors.New("execution proof required: supply block results and next signed header/validators")
	}
	if optional {
		if len(e.BlockResults) == 0 || len(e.NextCommit) == 0 || len(e.NextValidators) == 0 {
			return nil, errors.New("execution proof inputs must be supplied together")
		}
		if want.Height == int64(^uint64(0)>>1) {
			return nil, errors.New("height overflow")
		}
		next, err := signedHeader(e.NextCommit, e.NextValidators, anchor, want.Height+1)
		if err != nil {
			return nil, fmt.Errorf("next signed header: %w", err)
		}
		if !next.Header.LastBlockID.Equals(c.Commit.BlockID) || !bytes.Equal(c.Header.NextValidatorsHash, next.Header.ValidatorsHash) {
			return nil, errors.New("next header does not link to checkpoint block/set")
		}
		var results rpctypes.ResultBlockResults
		if err := rpcResult(e.BlockResults, &results); err != nil {
			return nil, fmt.Errorf("block results: %w", err)
		}
		if results.Height != want.Height || int64(len(results.TxsResults)) != tx.Proof.Proof.Total {
			return nil, errors.New("block result height/count differs from transaction proof")
		}
		for _, r := range results.TxsResults {
			if r == nil {
				return nil, errors.New("null transaction result")
			}
		}
		deterministic := cmttypes.NewResults(results.TxsResults)
		if !bytes.Equal(deterministic.Hash(), next.Header.LastResultsHash) {
			return nil, errors.New("block results do not match signed next-header result root")
		}
		observed := cmttypes.NewResults([]*abci.ExecTxResult{&tx.TxResult})
		if !proto.Equal(deterministic[tx.Index], observed[0]) {
			return nil, errors.New("endpoint transaction result differs from committed result")
		}
		execution["status"] = "code_data_gas_bound_to_signed_next_header"
		execution["next_height"] = strconv.FormatInt(want.Height+1, 10)
		execution["next_header_hash"] = strings.ToUpper(hex.EncodeToString(next.Header.Hash()))
		execution["last_results_hash"] = strings.ToUpper(hex.EncodeToString(next.Header.LastResultsHash))
		execution["data_base64"] = base64.StdEncoding.EncodeToString(deterministic[tx.Index].Data)
	}
	inputs := map[string][]byte{"genesis": e.Genesis, "tx": e.Tx, "commit": e.Commit, "validators": e.Validators, "block_results": e.BlockResults, "next_commit": e.NextCommit, "next_validators": e.NextValidators}
	hashes := map[string]string{}
	for k, b := range inputs {
		if len(b) != 0 {
			hashes[k] = digest(b)
		}
	}
	return &Receipt{Schema: "zerone-research-checkpoint-verification/v1", Verified: true, ChainID: chainID, Height: strconv.FormatInt(want.Height, 10), InputSHA256: hashes,
		Trust:       map[string]string{"genesis_sha256": digest(e.Genesis), "original_validator_public_key_base64": base64.StdEncoding.EncodeToString(anchor.Bytes()), "original_validator_address_hex": strings.ToUpper(hex.EncodeToString(anchor.Address())), "model": "explicit original single-validator trust anchor"},
		Transaction: transaction, Checkpoint: checkpoint, Execution: execution,
		Limits: []string{"Offline verification against the original development validator key; not independent validator diversity, custody assurance, or complete chain replay.", "Account number is supplied signing context; no account-state membership proof is claimed.", "Memo commits to hashes; the journal export must be separately obtained and validated. Inclusion does not establish scientific truth or review agreement.", "Execution logs, codespace and events are not committed by the Comet result root. Without the optional next-header proof, execution code and gas remain endpoint observations.", "This receipt does not promise permanent RPC, block or application-history availability."}}, nil
}
