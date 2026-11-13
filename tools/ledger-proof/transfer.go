package main

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	abci "github.com/cometbft/cometbft/abci/types"
	rpctypes "github.com/cometbft/cometbft/rpc/core/types"
	cmttypes "github.com/cometbft/cometbft/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	proto "github.com/cosmos/gogoproto/proto"
	curve "github.com/decred/dcrd/dcrec/secp256k1/v4"
)

const (
	transferSchema = "zerone.ledger-proof/transfer/v1"
	msgSendURL     = "/cosmos.bank.v1beta1.MsgSend"
	secp256k1URL   = "/cosmos.crypto.secp256k1.PubKey"
	nativeDenom    = "uzrn"
	maxBlockTxs    = 100000
	notCommitted   = "CometBFT 0.38 does not commit events or logs to any hash a header carries"
)

var (
	hashPattern   = regexp.MustCompile(`\A[0-9a-fA-F]{64}\z`)
	digestPattern = regexp.MustCompile(`\A[0-9a-f]{64}\z`)
	amountPattern = regexp.MustCompile(`\A[1-9][0-9]{0,77}\z`)
	payMemo       = regexp.MustCompile(`\Azr1:pay:([0-9a-f]{64})\z`)
	ancMemo       = regexp.MustCompile(`\Azr1:anc:([0-9a-f]{32}):([1-9][0-9]*):([0-9a-f]{64})\z`)
)

type transferWant struct {
	TxHash            string
	Height            int64
	Sender, Recipient string
	Amount            string
	AccountNumber     uint64
	Memo              *string // nil: no memo expected
	PayDigest         string  // "": no payment digest expected
	RequireExecution  bool
}

type transferEvidence struct {
	Tx, Commit, Validators, BlockResults, NextCommit, NextValidators []byte
}

type transferClaims struct {
	Inclusion      claim `json:"inclusion"`
	Authorisation  claim `json:"authorisation"`
	Execution      claim `json:"execution"`
	TransferEffect claim `json:"transfer_effect"`
	Events         claim `json:"events"`
	Logs           claim `json:"logs"`
}

type transferReport struct {
	Schema      string            `json:"schema"`
	ChainID     string            `json:"chain_id"`
	Height      string            `json:"height"`
	AttestedBy  attestation       `json:"attested_by"`
	Claims      transferClaims    `json:"claims"`
	Expected    map[string]any    `json:"expected"`
	Transaction map[string]any    `json:"transaction,omitempty"`
	Block       map[string]any    `json:"block,omitempty"`
	Execution   map[string]any    `json:"execution,omitempty"`
	Endpoint    map[string]any    `json:"endpoint_result,omitempty"`
	InputSHA256 map[string]string `json:"input_sha256"`
}

func runTransfer(args []string, policies map[string]policy) (any, bool, error) {
	var requireExecution boolFlag
	files := []string{"tx", "commit", "validators", "block-results", "next-commit", "next-validators"}
	names := append([]string{"chain", "txhash", "height", "sender", "recipient", "amount", "account-number", "memo", "pay-digest"}, files...)
	v, err := parseFlags("transfer", args, names, map[string]*boolFlag{"require-execution": &requireExecution})
	if err != nil {
		return nil, false, err
	}
	if err := requireFlags(v, "chain", "tx", "commit", "validators", "txhash", "height", "sender", "recipient", "amount", "account-number"); err != nil {
		return nil, false, err
	}
	p, err := lookupPolicy(policies, v["chain"].value)
	if err != nil {
		return nil, false, err
	}
	w := transferWant{TxHash: v["txhash"].value, Sender: v["sender"].value, Recipient: v["recipient"].value, Amount: v["amount"].value, RequireExecution: requireExecution.value}
	if w.Height, err = parseHeight(v["height"].value); err != nil {
		return nil, false, err
	}
	if !hashPattern.MatchString(w.TxHash) {
		return nil, false, usageError("--txhash requires 64 hex characters")
	}
	if _, err := parseAddress(w.Sender, 20); err != nil {
		return nil, false, usagef("--sender: %v", err)
	}
	if _, err := parseAddress(w.Recipient, 20, 32); err != nil {
		return nil, false, usagef("--recipient: %v", err)
	}
	if !amountPattern.MatchString(w.Amount) {
		return nil, false, usageError("--amount requires a positive integer of uzrn without leading zeros")
	}
	if w.AccountNumber, err = canonicalUint(v["account-number"].value, "account-number"); err != nil {
		return nil, false, err
	}
	if v["memo"].given && v["pay-digest"].given {
		return nil, false, usageError("give at most one of --memo and --pay-digest")
	}
	if v["memo"].given {
		m := v["memo"].value
		w.Memo = &m
	}
	if v["pay-digest"].given {
		if !digestPattern.MatchString(v["pay-digest"].value) {
			return nil, false, usageError("--pay-digest requires 64 lowercase hex characters")
		}
		w.PayDigest = v["pay-digest"].value
	}
	group := 0
	for _, n := range files[3:] {
		if v[n].given {
			group++
		}
	}
	if group != 0 && group != 3 {
		return nil, false, usageError("--block-results, --next-commit and --next-validators go together")
	}
	if w.RequireExecution && group == 0 {
		return nil, false, usageError("--require-execution needs --block-results, --next-commit and --next-validators")
	}
	in, err := readFiles(v, files...)
	if err != nil {
		return nil, false, err
	}
	r, ok := verifyTransfer(p, transferEvidence{Tx: in["tx"], Commit: in["commit"], Validators: in["validators"], BlockResults: in["block-results"], NextCommit: in["next-commit"], NextValidators: in["next-validators"]}, w)
	return r, ok, nil
}

func (w transferWant) expected() map[string]any {
	m := map[string]any{"txhash": strings.ToUpper(w.TxHash), "height": strconv.FormatInt(w.Height, 10), "sender": w.Sender, "recipient": w.Recipient, "amount_uzrn": w.Amount, "account_number": strconv.FormatUint(w.AccountNumber, 10), "require_execution": w.RequireExecution}
	if w.Memo != nil {
		m["memo"] = *w.Memo
	}
	if w.PayDigest != "" {
		m["pay_digest"] = w.PayDigest
	}
	return m
}

// verifyTransfer forms a report whenever it can. ok is true only when every
// proof the caller asked for holds: inclusion and authorisation always, and
// execution with its effect when the H+1 evidence is supplied or required.
// Without that evidence, a /tx response that itself reports failure is not ok.
func verifyTransfer(p policy, e transferEvidence, w transferWant) (transferReport, bool) {
	r := transferReport{Schema: transferSchema, ChainID: p.ChainID, Height: strconv.FormatInt(w.Height, 10), AttestedBy: p.attestation(), Expected: w.expected(),
		InputSHA256: inputHashes(map[string][]byte{"tx": e.Tx, "commit": e.Commit, "validators": e.Validators, "block_results": e.BlockResults, "next_commit": e.NextCommit, "next_validators": e.NextValidators})}
	r.Claims.Events, r.Claims.Logs = notProven(notCommitted), notProven(notCommitted)
	r.Claims.Execution = notProven("block results and the signed header at H+1 were not supplied")
	r.Claims.TransferEffect = notProven("execution is not proven")
	askedExecution := w.RequireExecution || len(e.BlockResults) != 0 || len(e.NextCommit) != 0 || len(e.NextValidators) != 0

	var tx rpctypes.ResultTx
	if err := rpcResult(e.Tx, &tx); err != nil {
		reason := "transaction response: " + err.Error()
		r.Claims.Inclusion, r.Claims.Authorisation = notProven(reason), notProven(reason)
		if askedExecution {
			r.Claims.Execution = notProven("inclusion is not proven, so no committed result can be bound to this transaction")
		}
		r.Claims.TransferEffect = notProven("needs inclusion, authorisation and execution all proven")
		return r, false
	}
	header, err := proveInclusion(p, e, w, &tx)
	if err != nil {
		r.Claims.Inclusion = notProven(err.Error())
	} else {
		r.Claims.Inclusion = proven()
		// The Merkle proof does not commit to its index or total, so the
		// transaction's position is reported only with execution.
		r.Block = map[string]any{"header_hash": upperHex(header.Hash()), "data_hash": upperHex(header.DataHash), "time": header.Time.UTC().Format(time.RFC3339Nano)}
	}
	// Authorisation speaks of the --txhash transaction, so other bytes are not judged.
	r.Transaction = map[string]any{"txhash": upperHex(tx.Tx.Hash()), "byte_length": len(tx.Tx)}
	if !strings.EqualFold(digest(tx.Tx), w.TxHash) {
		r.Claims.Authorisation = notProven("the --tx bytes are not the --txhash transaction")
	} else if facts, err := verifyTransaction(tx.Tx, p.ChainID, w); err != nil {
		r.Claims.Authorisation = notProven(err.Error())
	} else {
		r.Claims.Authorisation, r.Transaction = proven(), facts
	}

	var code uint32
	if askedExecution {
		switch {
		case len(e.BlockResults) == 0 || len(e.NextCommit) == 0 || len(e.NextValidators) == 0:
			r.Claims.Execution = notProven("block results, the commit at H+1 and the validators at H+1 must all be supplied")
		case header == nil:
			r.Claims.Execution = notProven("inclusion is not proven, so no committed result can be bound to this transaction")
		default:
			r.Execution, code, err = proveExecution(p, e, w, header, &tx)
			if err != nil {
				r.Claims.Execution = notProven(err.Error())
			} else {
				r.Claims.Execution = proven()
			}
		}
	}
	switch {
	case r.Claims.Inclusion.Status != statusProven || r.Claims.Authorisation.Status != statusProven || r.Claims.Execution.Status != statusProven:
		r.Claims.TransferEffect = notProven("needs inclusion, authorisation and execution all proven")
	case code != 0:
		r.Claims.TransferEffect = notProven(fmt.Sprintf("committed result code is %d, so the chain did not apply the transfer", code))
	default:
		r.Claims.TransferEffect = proven()
	}
	if !askedExecution {
		r.Endpoint = map[string]any{"code": tx.TxResult.Code, "status": "endpoint_observation_only"}
		if tx.TxResult.Code != 0 {
			r.Claims.TransferEffect = notProven(fmt.Sprintf("the /tx response reports code %d, an endpoint observation; supply the H+1 inputs to prove execution", tx.TxResult.Code))
		}
	}
	ok := r.Claims.Inclusion.Status == statusProven && r.Claims.Authorisation.Status == statusProven &&
		(!askedExecution || r.Claims.TransferEffect.Status == statusProven) && (askedExecution || tx.TxResult.Code == 0)
	return r, ok
}

func proveInclusion(p policy, e transferEvidence, w transferWant, tx *rpctypes.ResultTx) (*cmttypes.SignedHeader, error) {
	if !strings.EqualFold(digest(tx.Tx), w.TxHash) || !bytes.Equal(tx.Hash, tx.Tx.Hash()) {
		return nil, errors.New("transaction bytes do not hash to --txhash")
	}
	if tx.Height != w.Height {
		return nil, fmt.Errorf("transaction response is for height %d, not %d", tx.Height, w.Height)
	}
	pr := tx.Proof.Proof
	if !bytes.Equal(tx.Proof.Data, tx.Tx) || pr.Index != int64(tx.Index) || pr.Total <= 0 || pr.Total > maxBlockTxs || pr.Index < 0 || pr.Index >= pr.Total {
		return nil, errors.New("transaction proof data, index or total does not match the transaction; fetch it with prove=true")
	}
	header, err := trustedHeader(p, e.Commit, e.Validators, w.Height)
	if err != nil {
		return nil, fmt.Errorf("header at H: %w", err)
	}
	if err := tx.Proof.Validate(header.DataHash); err != nil {
		return nil, fmt.Errorf("transaction Merkle proof: %w", err)
	}
	return header, nil
}

// verifyTransaction checks the signed intent. It refuses what it cannot read
// unambiguously and reports, without judging, the fee and the other terms.
func verifyTransaction(raw []byte, chainID string, w transferWant) (map[string]any, error) {
	var tx txtypes.TxRaw
	var body txtypes.TxBody
	var auth txtypes.AuthInfo
	if err := canonicalProto(raw, &tx); err != nil {
		return nil, fmt.Errorf("TxRaw: %w", err)
	}
	if err := canonicalProto(tx.BodyBytes, &body); err != nil {
		return nil, fmt.Errorf("TxBody: %w", err)
	}
	if err := canonicalProto(tx.AuthInfoBytes, &auth); err != nil {
		return nil, fmt.Errorf("AuthInfo: %w", err)
	}
	if len(body.Messages) != 1 {
		return nil, fmt.Errorf("transaction carries %d messages; exactly one is supported", len(body.Messages))
	}
	m := body.Messages[0]
	if m == nil || len(m.XXX_unrecognized) != 0 {
		return nil, errors.New("message wrapper is empty or has unknown fields")
	}
	if m.TypeUrl != msgSendURL {
		return nil, fmt.Errorf("message type %s is not supported; only %s", m.TypeUrl, msgSendURL)
	}
	if len(body.ExtensionOptions) != 0 || len(body.NonCriticalExtensionOptions) != 0 {
		return nil, errors.New("extension options are not supported")
	}
	if body.Unordered || body.TimeoutTimestamp != nil {
		return nil, errors.New("unordered transactions and timeout timestamps are not supported")
	}
	var send banktypes.MsgSend
	if err := canonicalProto(m.Value, &send); err != nil {
		return nil, fmt.Errorf("MsgSend: %w", err)
	}
	if send.FromAddress != w.Sender {
		return nil, fmt.Errorf("message sender is %q, not --sender", send.FromAddress)
	}
	if send.ToAddress != w.Recipient {
		return nil, fmt.Errorf("message recipient is %q, not --recipient", send.ToAddress)
	}
	if len(send.Amount) != 1 {
		return nil, fmt.Errorf("message moves %d coins; exactly one is supported", len(send.Amount))
	}
	coin := send.Amount[0]
	if coin.Denom != nativeDenom {
		return nil, fmt.Errorf("message moves denom %q, not %s", coin.Denom, nativeDenom)
	}
	if coin.Amount.IsNil() || coin.Amount.String() != w.Amount {
		return nil, errors.New("message amount differs from --amount")
	}
	if !utf8.ValidString(body.Memo) {
		return nil, errors.New("memo is not valid UTF-8")
	}
	if w.Memo != nil && body.Memo != *w.Memo {
		return nil, errors.New("memo differs from --memo")
	}
	if w.PayDigest != "" && body.Memo != "zr1:pay:"+w.PayDigest {
		return nil, errors.New("memo is not zr1:pay:<--pay-digest>")
	}
	if auth.Tip != nil {
		return nil, errors.New("tips are not supported")
	}
	if len(auth.SignerInfos) != 1 || len(tx.Signatures) != 1 || auth.SignerInfos[0] == nil {
		return nil, fmt.Errorf("transaction has %d signers and %d signatures; exactly one of each is supported", len(auth.SignerInfos), len(tx.Signatures))
	}
	si := auth.SignerInfos[0]
	if si.ModeInfo == nil {
		return nil, errors.New("signer has no sign mode")
	}
	if si.ModeInfo.GetMulti() != nil {
		return nil, errors.New("multisig signers are not supported")
	}
	if si.ModeInfo.GetSingle() == nil {
		return nil, errors.New("signer has no sign mode")
	}
	if mode := si.ModeInfo.GetSingle().Mode; mode != signing.SignMode_SIGN_MODE_DIRECT {
		return nil, fmt.Errorf("sign mode %s is not supported; only SIGN_MODE_DIRECT", mode)
	}
	if si.PublicKey == nil || len(si.PublicKey.XXX_unrecognized) != 0 {
		return nil, errors.New("signer public key is missing or has unknown fields")
	}
	if si.PublicKey.TypeUrl != secp256k1URL {
		return nil, fmt.Errorf("signer key type %s is not supported; only %s", si.PublicKey.TypeUrl, secp256k1URL)
	}
	var pk secp256k1.PubKey
	if err := canonicalProto(si.PublicKey.Value, &pk); err != nil {
		return nil, fmt.Errorf("signer public key: %w", err)
	}
	if len(pk.Key) != 33 {
		return nil, errors.New("signer public key is not a compressed secp256k1 key")
	}
	if _, err := curve.ParsePubKey(pk.Key); err != nil {
		return nil, errors.New("signer public key is not on the secp256k1 curve")
	}
	sender, err := parseAddress(w.Sender, 20)
	if err != nil {
		return nil, fmt.Errorf("--sender: %w", err)
	}
	if !bytes.Equal(pk.Address(), sender) {
		return nil, errors.New("signing key's address is not --sender")
	}
	if len(tx.Signatures[0]) != 64 {
		return nil, errors.New("signature is not 64 bytes")
	}
	doc, err := proto.Marshal(&txtypes.SignDoc{BodyBytes: tx.BodyBytes, AuthInfoBytes: tx.AuthInfoBytes, ChainId: chainID, AccountNumber: w.AccountNumber})
	if err != nil {
		return nil, err
	}
	if !pk.VerifySignature(doc, tx.Signatures[0]) {
		return nil, errors.New("signature does not verify for this chain ID and --account-number")
	}
	fee := auth.Fee
	if fee == nil {
		fee = &txtypes.Fee{}
	}
	feeCoins := []string{}
	for _, c := range fee.Amount {
		feeCoins = append(feeCoins, coinText(c))
	}
	class, fields := classifyMemo(body.Memo, send.FromAddress, send.ToAddress, coin.Amount.String())
	facts := map[string]any{
		"txhash": upperHex(cmttypes.Tx(raw).Hash()), "byte_length": len(raw),
		"sender": send.FromAddress, "recipient": send.ToAddress, "amount": coinText(coin),
		"memo": body.Memo, "memo_class": class, "timeout_height": strconv.FormatUint(body.TimeoutHeight, 10),
		"fee": feeCoins, "gas_limit": strconv.FormatUint(fee.GasLimit, 10), "fee_payer": fee.Payer, "fee_granter": fee.Granter,
		"account_number": strconv.FormatUint(w.AccountNumber, 10), "sequence": strconv.FormatUint(si.Sequence, 10),
		"signer_public_key_base64": base64.StdEncoding.EncodeToString(pk.Key), "sign_doc_sha256": digest(doc),
	}
	if fields != nil {
		facts["memo_fields"] = fields
	}
	return facts, nil
}

func coinText(c sdk.Coin) string {
	if c.Amount.IsNil() {
		return "nil" + c.Denom
	}
	return c.Amount.String() + c.Denom
}

// classifyMemo applies the closed memo grammar. An anchor is a self-transfer
// of exactly 1 uzrn; the same text on any other transfer is "other".
func classifyMemo(memo, sender, recipient, amount string) (string, map[string]any) {
	if memo == "" {
		return "none", nil
	}
	if m := payMemo.FindStringSubmatch(memo); m != nil {
		return "pay", map[string]any{"digest": m[1]}
	}
	if m := ancMemo.FindStringSubmatch(memo); m != nil && sender == recipient && amount == "1" {
		if _, err := strconv.ParseUint(m[2], 10, 64); err == nil {
			return "anc", map[string]any{"stream": m[1], "counter": m[2], "head": m[3]}
		}
	}
	return "other", nil
}

// proveExecution binds the block's results to the trusted header at H+1 and
// reads the committed result at the transaction's index. The results root
// fixes the block's transaction count, and with it the proof's index.
func proveExecution(p policy, e transferEvidence, w transferWant, header *cmttypes.SignedHeader, tx *rpctypes.ResultTx) (map[string]any, uint32, error) {
	next, err := trustedHeader(p, e.NextCommit, e.NextValidators, w.Height+1)
	if err != nil {
		return nil, 0, fmt.Errorf("header at H+1: %w", err)
	}
	if !next.LastBlockID.Equals(header.Commit.BlockID) {
		return nil, 0, errors.New("header at H+1 does not name the block at H as its last block")
	}
	if !bytes.Equal(header.NextValidatorsHash, next.ValidatorsHash) {
		return nil, 0, errors.New("next validators hash at H differs from the validators hash at H+1")
	}
	var results rpctypes.ResultBlockResults
	if err := rpcResult(e.BlockResults, &results); err != nil {
		return nil, 0, fmt.Errorf("block results response: %w", err)
	}
	if results.Height != w.Height {
		return nil, 0, fmt.Errorf("block results are for height %d, not %d", results.Height, w.Height)
	}
	if int64(len(results.TxsResults)) != tx.Proof.Proof.Total {
		return nil, 0, errors.New("block results count differs from the block's transaction count")
	}
	for _, x := range results.TxsResults {
		if x == nil {
			return nil, 0, errors.New("block results contain an empty result")
		}
	}
	committed := cmttypes.NewResults(results.TxsResults)
	if !bytes.Equal(committed.Hash(), next.LastResultsHash) {
		return nil, 0, errors.New("block results do not rebuild to the last results hash at H+1")
	}
	res := committed[tx.Index]
	if !proto.Equal(res, cmttypes.NewResults([]*abci.ExecTxResult{&tx.TxResult})[0]) {
		return nil, 0, errors.New("the transaction response's result differs from the committed result")
	}
	return map[string]any{
		"tx_index": tx.Index, "tx_count": tx.Proof.Proof.Total,
		"code": res.Code, "data_base64": base64.StdEncoding.EncodeToString(res.Data),
		"gas_wanted": strconv.FormatInt(res.GasWanted, 10), "gas_used": strconv.FormatInt(res.GasUsed, 10),
		"next_height": strconv.FormatInt(w.Height+1, 10), "next_header_hash": upperHex(next.Hash()), "last_results_hash": upperHex(next.LastResultsHash),
	}, res.Code, nil
}
