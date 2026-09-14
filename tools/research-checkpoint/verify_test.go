package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	mathsdk "cosmossdk.io/math"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/crypto/ed25519"
	cmtjson "github.com/cometbft/cometbft/libs/json"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	cmtversion "github.com/cometbft/cometbft/proto/tendermint/version"
	rpctypes "github.com/cometbft/cometbft/rpc/core/types"
	cmttypes "github.com/cometbft/cometbft/types"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/bech32"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	proto "github.com/cosmos/gogoproto/proto"
)

func wire(t *testing.T, p proto.Message) []byte {
	t.Helper()
	b, e := proto.Marshal(p)
	if e != nil {
		t.Fatal(e)
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

type fixture struct {
	e         Evidence
	w         Expected
	validator ed25519.PrivKey
	sender    *secp256k1.PrivKey
	tx        rpctypes.ResultTx
	commit    *rpctypes.ResultCommit
	results   rpctypes.ResultBlockResults
}

func signTransaction(t *testing.T, key *secp256k1.PrivKey, w Expected, alter func(*txtypes.TxBody, *txtypes.AuthInfo)) []byte {
	t.Helper()
	msg := &banktypes.MsgSend{FromAddress: w.Sender, ToAddress: w.Sender, Amount: sdk.NewCoins(sdk.NewInt64Coin("uzrn", 1))}
	body := &txtypes.TxBody{Messages: []*codectypes.Any{{TypeUrl: "/cosmos.bank.v1beta1.MsgSend", Value: wire(t, msg)}}, Memo: w.Memo}
	auth := &txtypes.AuthInfo{SignerInfos: []*txtypes.SignerInfo{{PublicKey: &codectypes.Any{TypeUrl: "/cosmos.crypto.secp256k1.PubKey", Value: wire(t, key.PubKey())}, Sequence: 2, ModeInfo: &txtypes.ModeInfo{Sum: &txtypes.ModeInfo_Single_{Single: &txtypes.ModeInfo_Single{Mode: signing.SignMode_SIGN_MODE_DIRECT}}}}}, Fee: &txtypes.Fee{GasLimit: w.Gas, Amount: sdk.NewCoins(sdk.NewCoin("uzrn", mathsdk.NewInt(2000000)))}}
	if alter != nil {
		alter(body, auth)
	}
	b, a := wire(t, body), wire(t, auth)
	sig, err := key.Sign(wire(t, &txtypes.SignDoc{BodyBytes: b, AuthInfoBytes: a, ChainId: chainID, AccountNumber: w.AccountNumber}))
	if err != nil {
		t.Fatal(err)
	}
	return wire(t, &txtypes.TxRaw{BodyBytes: b, AuthInfoBytes: a, Signatures: [][]byte{sig}})
}

func signed(t *testing.T, h *cmttypes.Header, key ed25519.PrivKey) *rpctypes.ResultCommit {
	t.Helper()
	bid := cmttypes.BlockID{Hash: h.Hash(), PartSetHeader: cmttypes.PartSetHeader{Total: 1, Hash: bytes.Repeat([]byte{9}, 32)}}
	vote := &cmttypes.Vote{Type: cmtproto.PrecommitType, Height: h.Height, Round: 0, BlockID: bid, Timestamp: h.Time.Add(time.Second), ValidatorAddress: key.PubKey().Address(), ValidatorIndex: 0}
	sig, err := key.Sign(cmttypes.VoteSignBytes(h.ChainID, vote.ToProto()))
	if err != nil {
		t.Fatal(err)
	}
	vote.Signature = sig
	return &rpctypes.ResultCommit{SignedHeader: cmttypes.SignedHeader{Header: h, Commit: &cmttypes.Commit{Height: h.Height, Round: 0, BlockID: bid, Signatures: []cmttypes.CommitSig{vote.CommitSig()}}}, CanonicalCommit: true}
}

func newFixture(t *testing.T) fixture {
	t.Helper()
	f := fixture{validator: ed25519.GenPrivKey(), sender: secp256k1.GenPrivKey()}
	address, err := bech32.ConvertAndEncode("zrn", f.sender.PubKey().Address())
	if err != nil {
		t.Fatal(err)
	}
	f.w = Expected{Height: 12, Memo: "zerone:research:v1:01234567-89ab-cdef-0123-456789abcdef:81:" + strings.Repeat("a", 64) + ":" + strings.Repeat("b", 64), Sender: address, AccountNumber: 7, Gas: 2000000, Fee: "2000000", RequireExecutionProof: true}
	b := signTransaction(t, f.sender, f.w, nil)
	txs := cmttypes.Txs{cmttypes.Tx("other test transaction"), cmttypes.Tx(b)}
	f.w.TxHash = digest(b)
	f.tx = rpctypes.ResultTx{Hash: cmttypes.Tx(b).Hash(), Height: f.w.Height, Index: 1, Tx: b, Proof: txs.Proof(1), TxResult: abci.ExecTxResult{Code: 0, GasWanted: 2000000, GasUsed: 12345, Data: []byte("public result")}}
	vs := cmttypes.NewValidatorSet([]*cmttypes.Validator{cmttypes.NewValidator(f.validator.PubKey(), 100)})
	h := &cmttypes.Header{Version: cmtversion.Consensus{Block: 11}, ChainID: chainID, Height: f.w.Height, Time: time.Date(2026, 9, 14, 0, 0, 0, 0, time.UTC), DataHash: txs.Hash(), ValidatorsHash: vs.Hash(), NextValidatorsHash: vs.Hash(), ConsensusHash: bytes.Repeat([]byte{2}, 32), AppHash: bytes.Repeat([]byte{3}, 32), ProposerAddress: f.validator.PubKey().Address()}
	f.commit = signed(t, h, f.validator)
	f.results = rpctypes.ResultBlockResults{Height: f.w.Height, TxsResults: []*abci.ExecTxResult{{Code: 3, GasWanted: 2, GasUsed: 1}, &f.tx.TxResult}}
	next := *h
	next.Height++
	next.Time = next.Time.Add(time.Second)
	next.LastBlockID = f.commit.Commit.BlockID
	next.LastResultsHash = cmttypes.NewResults(f.results.TxsResults).Hash()
	f.e = Evidence{Genesis: []byte("synthetic in-memory trust fixture; main does not accept this genesis"), Tx: envelope(t, f.tx), Commit: envelope(t, f.commit), Validators: envelope(t, rpctypes.ResultValidators{BlockHeight: f.w.Height, Validators: vs.Validators, Count: 1, Total: 1}), BlockResults: envelope(t, f.results), NextCommit: envelope(t, signed(t, &next, f.validator)), NextValidators: envelope(t, rpctypes.ResultValidators{BlockHeight: next.Height, Validators: vs.Validators, Count: 1, Total: 1})}
	return f
}

func TestOfflineCheckpointAndExecutionProof(t *testing.T) {
	f := newFixture(t)
	r, err := verifyAnchored(f.e, f.w, f.validator.PubKey())
	if err != nil {
		t.Fatal(err)
	}
	if !r.Verified || r.Execution["status"] != "code_data_gas_bound_to_signed_next_header" || r.Checkpoint["entry_count"] != uint64(81) || r.Transaction["signature_verified"] != true {
		t.Fatalf("unexpected receipt: %+v", r)
	}
	// Main cannot substitute an arbitrary fetched/self-signed validator genesis.
	if _, err := verify(f.e, f.w); err == nil {
		t.Fatal("untrusted genesis accepted")
	}
	f.e.BlockResults, f.e.NextCommit, f.e.NextValidators = nil, nil, nil
	if _, err := verifyAnchored(f.e, f.w, f.validator.PubKey()); err == nil {
		t.Fatal("required execution proof omitted")
	}
	f.w.RequireExecutionProof = false
	r, err = verifyAnchored(f.e, f.w, f.validator.PubKey())
	if err != nil {
		t.Fatal(err)
	}
	if r.Execution["status"] != "endpoint_observation_only" {
		t.Fatal("unproved execution overstated")
	}
}

func TestAdversarialCheckpointEvidence(t *testing.T) {
	cases := []struct {
		name   string
		change func(*fixture)
	}{
		{"raw bytes", func(f *fixture) { f.tx.Tx[0] ^= 1; f.e.Tx = envelope(t, f.tx) }},
		{"explicit hash", func(f *fixture) { f.w.TxHash = strings.Repeat("f", 64) }},
		{"wrong expected height", func(f *fixture) { f.w.Height++ }},
		{"proof data", func(f *fixture) { f.tx.Proof.Data = []byte("different"); f.e.Tx = envelope(t, f.tx) }},
		{"proof root", func(f *fixture) { f.tx.Proof.RootHash[0] ^= 1; f.e.Tx = envelope(t, f.tx) }},
		{"proof aunt", func(f *fixture) { f.tx.Proof.Proof.Aunts[0][0] ^= 1; f.e.Tx = envelope(t, f.tx) }},
		{"proof index", func(f *fixture) { f.tx.Index = 0; f.e.Tx = envelope(t, f.tx) }},
		{"proof total", func(f *fixture) { f.tx.Proof.Proof.Total = -1; f.e.Tx = envelope(t, f.tx) }},
		{"header changed", func(f *fixture) { f.commit.Header.AppHash[0] ^= 1; f.e.Commit = envelope(t, f.commit) }},
		{"commit signature", func(f *fixture) { f.commit.Commit.Signatures[0].Signature[0] ^= 1; f.e.Commit = envelope(t, f.commit) }},
		{"signed wrong chain", func(f *fixture) {
			f.commit.Header.ChainID = "zerone-1"
			f.e.Commit = envelope(t, signed(t, f.commit.Header, f.validator))
		}},
		{"fetched replacement validator", func(f *fixture) {
			v := cmttypes.NewValidator(ed25519.GenPrivKey().PubKey(), 100)
			f.e.Validators = envelope(t, rpctypes.ResultValidators{BlockHeight: f.w.Height, Validators: []*cmttypes.Validator{v}, Count: 1, Total: 1})
		}},
		{"wrong account number", func(f *fixture) { f.w.AccountNumber++ }},
		{"wrong memo", func(f *fixture) { f.w.Memo = strings.Replace(f.w.Memo, ":81:", ":82:", 1) }},
		{"wrong fee", func(f *fixture) { f.w.Fee = "1" }},
		{"wrong gas", func(f *fixture) { f.w.Gas++ }},
		{"failed observed execution", func(f *fixture) { f.tx.TxResult.Code = 3; f.e.Tx = envelope(t, f.tx) }},
		{"result changed", func(f *fixture) { f.results.TxsResults[1].GasUsed++; f.e.BlockResults = envelope(t, f.results) }},
		{"observed result changed", func(f *fixture) { f.tx.TxResult.GasUsed++; f.e.Tx = envelope(t, f.tx) }},
		{"missing result", func(f *fixture) {
			f.results.TxsResults = f.results.TxsResults[:1]
			f.e.BlockResults = envelope(t, f.results)
		}},
		{"null result", func(f *fixture) { f.results.TxsResults[0] = nil; f.e.BlockResults = envelope(t, f.results) }},
		{"partial optional proof", func(f *fixture) { f.e.NextValidators = nil }},
		{"duplicate JSON field", func(f *fixture) {
			f.e.Tx = bytes.Replace(f.e.Tx, []byte(`"jsonrpc":"2.0"`), []byte(`"jsonrpc":"1.0","jsonrpc":"2.0"`), 1)
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newFixture(t)
			tc.change(&f)
			if _, err := verifyAnchored(f.e, f.w, f.validator.PubKey()); err == nil {
				t.Fatal("tampered evidence accepted")
			}
		})
	}
	// Even a complete, correctly signed replacement chain cannot bootstrap trust.
	f, other := newFixture(t), newFixture(t)
	if _, err := verifyAnchored(other.e, other.w, f.validator.PubKey()); err == nil {
		t.Fatal("self-consistent wrong signer trusted")
	}
}

func TestSignedIntentAndUnknownWireRefused(t *testing.T) {
	f := newFixture(t)
	mutations := []struct {
		name  string
		alter func(*txtypes.TxBody, *txtypes.AuthInfo)
	}{
		{"extra message", func(b *txtypes.TxBody, a *txtypes.AuthInfo) { b.Messages = append(b.Messages, b.Messages[0]) }},
		{"timeout", func(b *txtypes.TxBody, a *txtypes.AuthInfo) { b.TimeoutHeight = 100 }},
		{"different memo", func(b *txtypes.TxBody, a *txtypes.AuthInfo) { b.Memo += "x" }},
		{"delegated payer", func(b *txtypes.TxBody, a *txtypes.AuthInfo) { a.Fee.Payer = f.w.Sender }},
		{"second denom", func(b *txtypes.TxBody, a *txtypes.AuthInfo) {
			a.Fee.Amount = append(a.Fee.Amount, sdk.NewInt64Coin("other", 1))
		}},
		{"not self send", func(b *txtypes.TxBody, a *txtypes.AuthInfo) {
			var m banktypes.MsgSend
			_ = proto.Unmarshal(b.Messages[0].Value, &m)
			m.ToAddress = "zrn1different"
			b.Messages[0].Value = wire(t, &m)
		}},
		{"wrong principal", func(b *txtypes.TxBody, a *txtypes.AuthInfo) {
			var m banktypes.MsgSend
			_ = proto.Unmarshal(b.Messages[0].Value, &m)
			m.Amount = sdk.NewCoins(sdk.NewInt64Coin("uzrn", 2))
			b.Messages[0].Value = wire(t, &m)
		}},
	}
	for _, tc := range mutations {
		t.Run(tc.name, func(t *testing.T) {
			raw := signTransaction(t, f.sender, f.w, tc.alter)
			if _, _, err := verifyTransaction(raw, f.w); err == nil {
				t.Fatal("signed different intent accepted")
			}
		})
	}
	// Unknown TxRaw field and duplicate body field are normalized by protobuf;
	// canonical checking must reject both before treating them as the intended tx.
	for _, raw := range [][]byte{append(append([]byte{}, f.tx.Tx...), 0x20, 0x01), append(append([]byte{}, f.tx.Tx...), 0x0a, 0x00)} {
		if _, _, err := verifyTransaction(raw, f.w); err == nil {
			t.Fatal("ambiguous wire accepted")
		}
	}
	var tx txtypes.TxRaw
	_ = proto.Unmarshal(f.tx.Tx, &tx)
	tx.Signatures[0][1] ^= 1
	if _, _, err := verifyTransaction(wire(t, &tx), f.w); err == nil {
		t.Fatal("bad sender signature accepted")
	}
}

func TestNonCommittedResultFieldsAreNotClaimed(t *testing.T) {
	f := newFixture(t)
	f.results.TxsResults[1].Log = "different uncommitted log"
	f.results.TxsResults[1].Codespace = "not a committed field"
	f.e.BlockResults = envelope(t, f.results)
	r, err := verifyAnchored(f.e, f.w, f.validator.PubKey())
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := r.Execution["log"]; ok {
		t.Fatal("uncommitted log represented as verified")
	}
}

func TestSignedAnyUnknownFieldsRefused(t *testing.T) {
	f := newFixture(t)
	for _, target := range []string{"message", "signer public key"} {
		t.Run(target, func(t *testing.T) {
			raw := signTransaction(t, f.sender, f.w, func(b *txtypes.TxBody, a *txtypes.AuthInfo) {
				if target == "message" {
					b.Messages[0].XXX_unrecognized = []byte{0x18, 0x01}
				} else {
					a.SignerInfos[0].PublicKey.XXX_unrecognized = []byte{0x18, 0x01}
				}
			})
			// These are actually signed, canonical-looking nested wrappers: unlike
			// TxBody's own unknown fields, Any preserves the bytes on re-encoding.
			var tx txtypes.TxRaw
			var body txtypes.TxBody
			var auth txtypes.AuthInfo
			if err := canonicalProto(raw, &tx); err != nil {
				t.Fatal(err)
			}
			if err := canonicalProto(tx.BodyBytes, &body); err != nil {
				t.Fatal(err)
			}
			if err := canonicalProto(tx.AuthInfoBytes, &auth); err != nil {
				t.Fatal(err)
			}
			if !f.sender.PubKey().VerifySignature(wire(t, &txtypes.SignDoc{BodyBytes: tx.BodyBytes, AuthInfoBytes: tx.AuthInfoBytes, ChainId: chainID, AccountNumber: f.w.AccountNumber}), tx.Signatures[0]) {
				t.Fatal("fixture is not a valid sender signature")
			}
			if _, _, err := verifyTransaction(raw, f.w); err == nil {
				t.Fatal("signed nested unknown Any fields accepted")
			}
		})
	}
}

func TestMemoBoundsAndCanonicalText(t *testing.T) {
	f := newFixture(t)
	for _, count := range []string{"0", "01", "1001", "18446744073709551616"} {
		w := f.w
		w.Memo = strings.Replace(w.Memo, ":81:", ":"+count+":", 1)
		if _, _, err := verifyTransaction(signTransaction(t, f.sender, w, nil), w); err == nil {
			t.Fatalf("invalid count %s accepted", count)
		}
	}
	for _, suffix := range []string{"\n", " ", "Ω"} {
		w := f.w
		w.Memo += suffix
		if _, _, err := verifyTransaction(signTransaction(t, f.sender, w, nil), w); err == nil {
			t.Fatal("noncanonical memo accepted")
		}
	}
	w := f.w
	w.Memo = strings.Replace(w.Memo, ":81:", ":1000:", 1)
	if _, _, err := verifyTransaction(signTransaction(t, f.sender, w, nil), w); err != nil {
		t.Fatal(err)
	}
}

func TestBoundedFilesAndNoOverwrite(t *testing.T) {
	d := t.TempDir()
	ordinary := filepath.Join(d, "evidence.json")
	if err := os.WriteFile(ordinary, []byte(`{"ok":true}`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := readInput(ordinary); err != nil {
		t.Fatal(err)
	}
	symlink := filepath.Join(d, "link")
	if err := os.Symlink(ordinary, symlink); err != nil {
		t.Fatal(err)
	}
	if _, err := readInput(symlink); err == nil {
		t.Fatal("symlink accepted")
	}
	fifo := filepath.Join(d, "fifo")
	if err := syscall.Mkfifo(fifo, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := readInput(fifo); err == nil {
		t.Fatal("FIFO accepted")
	}
	for _, bad := range []string{`{"a":1,"a":2}`, `{"x":{"a":1,"a":2}}`, `{} {}`, strings.Repeat("[", 66) + "0" + strings.Repeat("]", 66)} {
		if err := strictJSON([]byte(bad)); err == nil {
			t.Fatalf("bad JSON accepted: %.80s", bad)
		}
	}
	f := newFixture(t)
	r, err := verifyAnchored(f.e, f.w, f.validator.PubKey())
	if err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(d, "receipt.json")
	if err := writeReceipt(out, r); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(out)
	if err := writeReceipt(out, r); err == nil {
		t.Fatal("existing receipt overwritten")
	}
	after, _ := os.ReadFile(out)
	if !bytes.Equal(before, after) {
		t.Fatal("existing bytes changed")
	}
	var parsed Receipt
	if err := json.Unmarshal(after, &parsed); err != nil {
		t.Fatal(err)
	}
	if s, err := os.Stat(out); err != nil || s.Mode().Perm() != 0600 {
		t.Fatal("receipt mode differs")
	}
}

// This is the public, already committed development checkpoint. It performs no
// network access and holds no signing keys. Its anchor remains the built-in
// original genesis/key, not a validator key selected by the fixture.
func TestPublishedCheckpointProof(t *testing.T) {
	dir := filepath.Join("..", "..", "dashboard", "public", "research", "checkpoints", "2026-09-14-demimetric-v1", "chain")
	read := func(name string) []byte {
		t.Helper()
		b, err := readInput(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		return b
	}
	e := Evidence{Genesis: read("genesis.json"), Tx: read("tx-rpc.json"), Commit: read("commit.json"), Validators: read("validators.json"), BlockResults: read("block-results.json"), NextCommit: read("next-commit.json"), NextValidators: read("next-validators.json")}
	w := Expected{Height: 183706, TxHash: "8DFC740C5326E76B482B6D370F527311CB518934D6B64A4BF5EFDBD2806D0B8E", Memo: "zerone:research:v1:ce6e799a-6713-41a9-b95f-57c50a223074:81:86c8163599c0098aa43b4f3599a81b02a03557553fc595b67b026fdfe108704e:23baf5b0a2e7f4a48fe2b86775ae2ef92e607f2d793270ff10219e61d2563847", Sender: "zrn1q36wcvumlxa3qrmsu8cmkrjqsa4jqfchtzkuj8", AccountNumber: 7, Gas: 2000000, Fee: "2000000", RequireExecutionProof: true}
	r, err := verify(e, w)
	if err != nil {
		t.Fatal(err)
	}
	if r.Execution["status"] != "code_data_gas_bound_to_signed_next_header" || r.Execution["gas_used"] != "100584" || r.Transaction["byte_length"] != 501 {
		t.Fatal("published receipt scope changed")
	}
	actual, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(append(actual, '\n'), read("verification.json")) {
		t.Fatal("published receipt differs from recomputed verification")
	}
}
