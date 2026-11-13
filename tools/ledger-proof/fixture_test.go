package main

import (
	"bytes"
	"encoding/base64"
	"strings"
	"testing"
	"time"

	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/crypto/ed25519"
	cmtjson "github.com/cometbft/cometbft/libs/json"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	cmtversion "github.com/cometbft/cometbft/proto/tendermint/version"
	rpctypes "github.com/cometbft/cometbft/rpc/core/types"
	cmttypes "github.com/cometbft/cometbft/types"
	"github.com/cometbft/cometbft/version"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/bech32"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	proto "github.com/cosmos/gogoproto/proto"
)

// Every key, block and store in these tests is disposable and made in memory.

func wire(t *testing.T, p proto.Message) []byte {
	t.Helper()
	b, err := proto.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func envelope(t *testing.T, v any) []byte {
	t.Helper()
	b, err := cmtjson.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return append(append([]byte(`{"jsonrpc":"2.0","id":1,"result":`), b...), '}')
}

func address(t *testing.T, b []byte) string {
	t.Helper()
	s, err := bech32.ConvertAndEncode("zrn", b)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// testChain is a policy over fresh Ed25519 keys and the keys to sign with.
type testChain struct {
	pol  policy
	keys map[string]ed25519.PrivKey
}

func newTestChain(chainID string, powers ...int64) testChain {
	c := testChain{pol: policy{ChainID: chainID, Label: "TEST_ONLY", Words: "Disposable keys made by a test.", Source: "tools/ledger-proof tests"}, keys: map[string]ed25519.PrivKey{}}
	for _, p := range powers {
		k := ed25519.GenPrivKey()
		c.keys[string(k.PubKey().Address())] = k
		c.pol.Validators = append(c.pol.Validators, pinnedValidator{PublicKey: base64.StdEncoding.EncodeToString(k.PubKey().Bytes()), Power: p})
	}
	return c
}

func (c testChain) set(t *testing.T) *cmttypes.ValidatorSet {
	t.Helper()
	s, err := c.pol.validatorSet()
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func (c testChain) header(t *testing.T, height int64) *cmttypes.Header {
	s := c.set(t)
	return &cmttypes.Header{Version: cmtversion.Consensus{Block: version.BlockProtocol}, ChainID: c.pol.ChainID, Height: height,
		Time: time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC).Add(time.Duration(height) * time.Second), ValidatorsHash: s.Hash(), NextValidatorsHash: s.Hash(),
		ConsensusHash: bytes.Repeat([]byte{2}, 32), AppHash: bytes.Repeat([]byte{3}, 32), ProposerAddress: s.Validators[0].Address}
}

// commit signs h with every validator of the set except those at the absent
// indices (in validator-set order).
func (c testChain) commit(t *testing.T, h *cmttypes.Header, absent ...int) *rpctypes.ResultCommit {
	t.Helper()
	s := c.set(t)
	bid := cmttypes.BlockID{Hash: h.Hash(), PartSetHeader: cmttypes.PartSetHeader{Total: 1, Hash: bytes.Repeat([]byte{9}, 32)}}
	sigs := make([]cmttypes.CommitSig, s.Size())
	for i, v := range s.Validators {
		skip := false
		for _, a := range absent {
			skip = skip || a == i
		}
		if skip {
			sigs[i] = cmttypes.NewCommitSigAbsent()
			continue
		}
		vote := &cmttypes.Vote{Type: cmtproto.PrecommitType, Height: h.Height, BlockID: bid, Timestamp: h.Time.Add(time.Second), ValidatorAddress: v.Address, ValidatorIndex: int32(i)}
		sig, err := c.keys[string(v.Address)].Sign(cmttypes.VoteSignBytes(h.ChainID, vote.ToProto()))
		if err != nil {
			t.Fatal(err)
		}
		vote.Signature = sig
		sigs[i] = vote.CommitSig()
	}
	return &rpctypes.ResultCommit{SignedHeader: cmttypes.SignedHeader{Header: h, Commit: &cmttypes.Commit{Height: h.Height, BlockID: bid, Signatures: sigs}}, CanonicalCommit: true}
}

func (c testChain) validators(t *testing.T, height int64) []byte {
	s := c.set(t)
	return envelope(t, rpctypes.ResultValidators{BlockHeight: height, Validators: s.Validators, Count: s.Size(), Total: s.Size()})
}

type txSpec struct {
	signer    *secp256k1.PrivKey // defaults to the sender's key
	recipient string             // defaults to a fresh address
	amount    int64
	memo      string
	code      uint32
	alter     func(*txtypes.TxBody, *txtypes.AuthInfo)
}

type transferFixture struct {
	chain   testChain
	sender  *secp256k1.PrivKey
	w       transferWant
	tx      rpctypes.ResultTx
	header  *cmttypes.Header
	results rpctypes.ResultBlockResults
	next    *cmttypes.Header
	e       transferEvidence
}

func newTransfer(t *testing.T, chain testChain, s txSpec) *transferFixture {
	t.Helper()
	f := &transferFixture{chain: chain, sender: secp256k1.GenPrivKey()}
	from := address(t, f.sender.PubKey().Address())
	if s.recipient == "" {
		s.recipient = address(t, secp256k1.GenPrivKey().PubKey().Address())
	}
	if s.amount == 0 {
		s.amount = 250
	}
	if s.signer == nil {
		s.signer = f.sender
	}
	const height, accountNumber = 40, 9
	msg := &banktypes.MsgSend{FromAddress: from, ToAddress: s.recipient, Amount: sdk.NewCoins(sdk.NewInt64Coin(nativeDenom, s.amount))}
	body := &txtypes.TxBody{Messages: []*codectypes.Any{{TypeUrl: msgSendURL, Value: wire(t, msg)}}, Memo: s.memo}
	auth := &txtypes.AuthInfo{SignerInfos: []*txtypes.SignerInfo{{PublicKey: &codectypes.Any{TypeUrl: secp256k1URL, Value: wire(t, s.signer.PubKey())}, Sequence: 4,
		ModeInfo: &txtypes.ModeInfo{Sum: &txtypes.ModeInfo_Single_{Single: &txtypes.ModeInfo_Single{Mode: signing.SignMode_SIGN_MODE_DIRECT}}}}},
		Fee: &txtypes.Fee{GasLimit: 200000, Amount: sdk.NewCoins(sdk.NewInt64Coin(nativeDenom, 200000))}}
	if s.alter != nil {
		s.alter(body, auth)
	}
	b, a := wire(t, body), wire(t, auth)
	sig, err := s.signer.Sign(wire(t, &txtypes.SignDoc{BodyBytes: b, AuthInfoBytes: a, ChainId: chain.pol.ChainID, AccountNumber: accountNumber}))
	if err != nil {
		t.Fatal(err)
	}
	raw := wire(t, &txtypes.TxRaw{BodyBytes: b, AuthInfoBytes: a, Signatures: [][]byte{sig}})
	txs := cmttypes.Txs{cmttypes.Tx("first test transaction"), cmttypes.Tx(raw), cmttypes.Tx("third test transaction")}
	result := abci.ExecTxResult{Code: s.code, GasWanted: 200000, GasUsed: 61234, Data: []byte("result data"), Log: "uncommitted log", Codespace: "uncommitted", Events: []abci.Event{{Type: "transfer", Attributes: []abci.EventAttribute{{Key: "amount", Value: "999uzrn"}}}}}
	f.tx = rpctypes.ResultTx{Hash: txs[1].Hash(), Height: height, Index: 1, Tx: raw, Proof: txs.Proof(1), TxResult: result}
	f.header = chain.header(t, height)
	f.header.DataHash = txs.Hash()
	second := result
	f.results = rpctypes.ResultBlockResults{Height: height, TxsResults: []*abci.ExecTxResult{{Code: 0, GasUsed: 1}, &second, {Code: 7, Data: []byte("x")}}}
	f.w = transferWant{TxHash: strings.ToLower(upperHex(txs[1].Hash())), Height: height, Sender: from, Recipient: s.recipient, Amount: msg.Amount[0].Amount.String(), AccountNumber: accountNumber, RequireExecution: true}
	f.seal(t)
	return f
}

// seal rebuilds the header at H+1 and every envelope from the fixture's fields.
func (f *transferFixture) seal(t *testing.T) {
	t.Helper()
	commit := f.chain.commit(t, f.header)
	f.next = f.chain.header(t, f.header.Height+1)
	f.next.LastBlockID = commit.Commit.BlockID
	f.next.LastResultsHash = cmttypes.NewResults(f.results.TxsResults).Hash()
	f.e = transferEvidence{Tx: envelope(t, f.tx), Commit: envelope(t, commit), Validators: f.chain.validators(t, f.header.Height),
		BlockResults: envelope(t, f.results), NextCommit: envelope(t, f.chain.commit(t, f.next)), NextValidators: f.chain.validators(t, f.next.Height)}
}

// checkTransfer runs the verifier and asserts the invariants every report
// must keep, whatever the evidence: events and logs are never proven, and
// transfer_effect is proven only on top of the three proofs it needs.
func checkTransfer(t *testing.T, p policy, e transferEvidence, w transferWant) (transferReport, bool) {
	t.Helper()
	r, ok := verifyTransfer(p, e, w)
	c := r.Claims
	if c.Events.Status != statusNotProven || c.Logs.Status != statusNotProven || c.Events.Reason == "" {
		t.Fatalf("events or logs not reported as NOT_PROVEN: %+v", c)
	}
	if c.TransferEffect.Status == statusProven && (c.Inclusion.Status != statusProven || c.Authorisation.Status != statusProven || c.Execution.Status != statusProven) {
		t.Fatalf("effect proven without its prerequisites: %+v", c)
	}
	for _, x := range []claim{c.Inclusion, c.Authorisation, c.Execution, c.TransferEffect} {
		if x.Status != statusProven && x.Reason == "" {
			t.Fatalf("unproven claim without a reason: %+v", c)
		}
	}
	if r.AttestedBy.Label != p.Label || r.AttestedBy.Words != p.Words || r.AttestedBy.PinSource != p.Source || len(r.AttestedBy.Validators) != len(p.Validators) {
		t.Fatalf("attestation missing: %+v", r.AttestedBy)
	}
	return r, ok
}
