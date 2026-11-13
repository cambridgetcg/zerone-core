package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/crypto/ed25519"
	rpctypes "github.com/cometbft/cometbft/rpc/core/types"
	cmttypes "github.com/cometbft/cometbft/types"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	kmultisig "github.com/cosmos/cosmos-sdk/crypto/keys/multisig"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	proto "github.com/cosmos/gogoproto/proto"
)

func TestTransferProven(t *testing.T) {
	chain := newTestChain("test-a", 100)
	f := newTransfer(t, chain, txSpec{})
	r, ok := checkTransfer(t, chain.pol, f.e, f.w)
	c := r.Claims
	if !ok || c.Inclusion.Status != statusProven || c.Authorisation.Status != statusProven || c.Execution.Status != statusProven || c.TransferEffect.Status != statusProven {
		t.Fatalf("valid transfer not proven: %+v", c)
	}
	x := r.Transaction
	if x["memo_class"] != "none" || x["amount"] != "250uzrn" || x["gas_limit"] != "200000" || x["sequence"] != "4" || x["timeout_height"] != "0" || x["fee_payer"] != "" {
		t.Fatalf("unexpected facts: %+v", x)
	}
	if r.Execution["code"] != uint32(0) || r.Execution["gas_used"] != "61234" || r.Execution["tx_index"] != uint32(1) || r.Execution["tx_count"] != int64(3) || r.Endpoint != nil {
		t.Fatalf("unexpected execution facts: %+v %+v", r.Execution, r.Endpoint)
	}
	b, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	// The endpoint's log, codespace and events are not committed; none reaches the report.
	if bytes.Contains(b, []byte("uncommitted")) || bytes.Contains(b, []byte("999uzrn")) {
		t.Fatal("uncommitted result fields reported")
	}
	if !bytes.Contains(b, []byte(`"schema":"zerone.ledger-proof/transfer/v1"`)) {
		t.Fatal("schema missing")
	}
}

func TestTransferWithoutExecutionEvidence(t *testing.T) {
	chain := newTestChain("test-a", 100)
	f := newTransfer(t, chain, txSpec{})
	f.e.BlockResults, f.e.NextCommit, f.e.NextValidators = nil, nil, nil
	f.w.RequireExecution = false
	r, ok := checkTransfer(t, chain.pol, f.e, f.w)
	if !ok || r.Claims.Execution.Status != statusNotProven || r.Claims.TransferEffect.Status != statusNotProven || r.Execution != nil ||
		r.Endpoint["code"] != uint32(0) || r.Endpoint["status"] != "endpoint_observation_only" || r.Block["tx_index"] != nil || r.Block["tx_count"] != nil {
		t.Fatalf("inclusion-only report wrong: ok=%v %+v %+v %+v", ok, r.Claims, r.Endpoint, r.Block)
	}
	f.w.RequireExecution = true
	if _, ok := checkTransfer(t, chain.pol, f.e, f.w); ok {
		t.Fatal("required execution proof omitted but accepted")
	}
	// The /tx response's own report of failure is not dropped.
	g := newTransfer(t, chain, txSpec{code: 5})
	g.e.BlockResults, g.e.NextCommit, g.e.NextValidators = nil, nil, nil
	g.w.RequireExecution = false
	r, ok = checkTransfer(t, chain.pol, g.e, g.w)
	if ok || r.Claims.Inclusion.Status != statusProven || r.Claims.Authorisation.Status != statusProven || r.Claims.TransferEffect.Status != statusNotProven ||
		!strings.Contains(r.Claims.TransferEffect.Reason, "reports code 5") || r.Endpoint["code"] != uint32(5) {
		t.Fatalf("endpoint failure accepted: ok=%v %+v %+v", ok, r.Claims, r.Endpoint)
	}
}

// moveToLast rebuilds the fixture's block with its transaction last of three.
func (f *transferFixture) moveToLast(t *testing.T) {
	t.Helper()
	txs := cmttypes.Txs{cmttypes.Tx("first test transaction"), cmttypes.Tx("third test transaction"), f.tx.Tx}
	f.tx.Index, f.tx.Proof = 2, txs.Proof(2)
	f.header.DataHash = txs.Hash()
	x := f.results.TxsResults
	f.results.TxsResults = []*abci.ExecTxResult{x[0], x[2], x[1]}
	f.seal(t)
}

// The transaction Merkle proof does not commit to its index or total: the
// same leaf and aunts reach the data hash as 1 of 3 and 1 of 4, and as 2 of 3
// and 1 of 2. Inclusion stays proven, the report states no position, and the
// committed results refuse the relabelled count.
func TestInclusionDoesNotProvePosition(t *testing.T) {
	chain := newTestChain("test-a", 100)
	for _, tc := range []struct {
		name  string
		last  bool
		index uint32
		total int64
	}{
		{"index 1 of 3 relabelled 1 of 4", false, 1, 4},
		{"index 2 of 3 relabelled 1 of 2", true, 1, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newTransfer(t, chain, txSpec{})
			if tc.last {
				f.moveToLast(t)
			}
			honest := f.tx.Index
			if r, ok := checkTransfer(t, chain.pol, f.e, f.w); !ok || r.Execution["tx_index"] != honest || r.Execution["tx_count"] != int64(3) {
				t.Fatalf("honest position not proven: ok=%v %+v", ok, r.Execution)
			}
			f.tx.Index, f.tx.Proof.Proof.Index, f.tx.Proof.Proof.Total = tc.index, int64(tc.index), tc.total
			f.e.Tx = envelope(t, f.tx)
			r, ok := checkTransfer(t, chain.pol, f.e, f.w)
			if ok || r.Claims.Inclusion.Status != statusProven || !strings.Contains(r.Claims.Execution.Reason, "count differs") || r.Execution != nil {
				t.Fatalf("relabelled position accepted with execution: ok=%v %+v", ok, r.Claims)
			}
			f.w.RequireExecution = false
			f.e.BlockResults, f.e.NextCommit, f.e.NextValidators = nil, nil, nil
			r, ok = checkTransfer(t, chain.pol, f.e, f.w)
			b, err := json.Marshal(r)
			if err != nil {
				t.Fatal(err)
			}
			if !ok || r.Claims.Inclusion.Status != statusProven || bytes.Contains(b, []byte("tx_index")) || bytes.Contains(b, []byte("tx_count")) {
				t.Fatalf("inclusion-only report states an unproven position: ok=%v\n%s", ok, b)
			}
		})
	}
}

func TestVotingPowerThreshold(t *testing.T) {
	chain := newTestChain("test-a", 40, 30, 20, 10) // set order is by power: 40, 30, 20, 10
	for _, tc := range []struct {
		absent []int
		ok     bool
	}{{nil, true}, {[]int{3}, true}, {[]int{2, 3}, true}, {[]int{0}, false}, {[]int{1, 3}, false}} {
		f := newTransfer(t, chain, txSpec{})
		f.w.RequireExecution = false
		f.e.BlockResults, f.e.NextCommit, f.e.NextValidators = nil, nil, nil
		f.e.Commit = envelope(t, chain.commit(t, f.header, tc.absent...))
		r, ok := checkTransfer(t, chain.pol, f.e, f.w)
		if ok != tc.ok || (r.Claims.Inclusion.Status == statusProven) != tc.ok {
			t.Fatalf("absent %v: ok=%v %+v", tc.absent, ok, r.Claims.Inclusion)
		}
	}
}

// A commit that makes CometBFT's verification panic must come back as a
// refusal. trustedHeader refuses the one known route first (a signature
// timestamp no protobuf timestamp can carry); this calls verifyCommit with
// that same commit directly, so the recover is exercised too.
func TestVerifyCommitNeverPanics(t *testing.T) {
	chain := newTestChain("test-a", 100)
	f := newTransfer(t, chain, txSpec{})
	c := chain.commit(t, f.header)
	c.Commit.Signatures[0].Timestamp = time.Date(0, 1, 1, 0, 0, 0, 0, time.UTC)
	err := verifyCommit(chain.set(t), "test-a", f.header.Height, c.Commit)
	if err == nil || !strings.Contains(err.Error(), "could not be verified") {
		t.Fatalf("want a refusal, got %v", err)
	}
}

func TestHeaderTrust(t *testing.T) {
	chain := newTestChain("test-a", 100)
	cases := []struct {
		name, reason string
		change       func(*transferFixture)
	}{
		{"self-consistent chain signed by a key outside the policy", "not in the pinned set", func(f *transferFixture) {
			other := newTestChain("test-a", 100)
			g := newTransfer(t, other, txSpec{})
			f.e, f.w = g.e, g.w
		}},
		{"pinned validator response with a commit by another key", "commit signatures", func(f *transferFixture) {
			other := newTestChain("test-a", 100)
			f.e.Commit = envelope(t, other.commit(t, f.header))
		}},
		{"extra validator", "not exactly the pinned set", func(f *transferFixture) {
			vals := append(chain.set(t).Validators, cmttypes.NewValidator(ed25519.GenPrivKey().PubKey(), 100))
			f.e.Validators = envelope(t, rpctypes.ResultValidators{BlockHeight: f.w.Height, Validators: vals, Count: 2, Total: 2})
		}},
		{"re-weighted validator", "with its pinned power", func(f *transferFixture) {
			v := chain.set(t).Validators[0].Copy()
			v.VotingPower = 101
			f.e.Validators = envelope(t, rpctypes.ResultValidators{BlockHeight: f.w.Height, Validators: []*cmttypes.Validator{v}, Count: 1, Total: 1})
		}},
		{"replacement validator", "not in the pinned set", func(f *transferFixture) {
			v := cmttypes.NewValidator(ed25519.GenPrivKey().PubKey(), 100)
			f.e.Validators = envelope(t, rpctypes.ResultValidators{BlockHeight: f.w.Height, Validators: []*cmttypes.Validator{v}, Count: 1, Total: 1})
		}},
		{"validator response with more validators on another page", "not exactly the pinned set", func(f *transferFixture) {
			f.e.Validators = envelope(t, rpctypes.ResultValidators{BlockHeight: f.w.Height, Validators: chain.set(t).Validators, Count: 1, Total: 2})
		}},
		{"header names a larger validator set", "validators hash", func(f *transferFixture) {
			larger := cmttypes.NewValidatorSet(append(chain.set(t).Validators, cmttypes.NewValidator(ed25519.GenPrivKey().PubKey(), 1000)))
			f.header.ValidatorsHash = larger.Hash()
			f.e.Commit = envelope(t, chain.commit(t, f.header))
		}},
		{"commit block ID with a malformed part-set hash", "commit block ID", func(f *transferFixture) {
			c := chain.commit(t, f.header)
			c.Commit.BlockID.PartSetHeader.Hash = []byte{1, 2, 3, 4, 5}
			f.e.Commit = envelope(t, c)
		}},
		{"commit signature with a timestamp before year 1", "timestamp is outside years", func(f *transferFixture) {
			c := chain.commit(t, f.header)
			c.Commit.Signatures[0].Timestamp = time.Date(0, 1, 1, 0, 0, 0, 0, time.UTC)
			f.e.Commit = envelope(t, c)
		}},
		{"validator response for another height", "validator response is for height", func(f *transferFixture) { f.e.Validators = chain.validators(t, f.w.Height+1) }},
		{"header changed after signing", "commit signs block", func(f *transferFixture) {
			c := chain.commit(t, f.header)
			c.Header.AppHash = bytes.Repeat([]byte{4}, 32)
			f.e.Commit = envelope(t, c)
		}},
		{"commit signature altered", "wrong signature", func(f *transferFixture) {
			c := chain.commit(t, f.header)
			c.Commit.Signatures[0].Signature[0] ^= 1
			f.e.Commit = envelope(t, c)
		}},
		{"commit signature marked absent", "voting power", func(f *transferFixture) { f.e.Commit = envelope(t, chain.commit(t, f.header, 0)) }},
		{"proposer outside the set", "proposer", func(f *transferFixture) {
			f.header.ProposerAddress = ed25519.GenPrivKey().PubKey().Address()
			f.e.Commit = envelope(t, chain.commit(t, f.header))
		}},
		{"header signed for another chain ID", "another chain", func(f *transferFixture) {
			f.header.ChainID = "test-b"
			f.e.Commit = envelope(t, chain.commit(t, f.header))
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newTransfer(t, chain, txSpec{})
			tc.change(f)
			r, ok := checkTransfer(t, chain.pol, f.e, f.w)
			if ok || r.Claims.Inclusion.Status != statusNotProven || !strings.Contains(r.Claims.Inclusion.Reason, tc.reason) || r.Claims.TransferEffect.Status == statusProven {
				t.Fatalf("want refusal %q: %+v", tc.reason, r.Claims)
			}
		})
	}
	t.Run("missing validator", func(t *testing.T) {
		multi := newTestChain("test-a", 40, 30, 20, 10)
		f := newTransfer(t, multi, txSpec{})
		vals := multi.set(t).Validators[:3]
		f.e.Validators = envelope(t, rpctypes.ResultValidators{BlockHeight: f.w.Height, Validators: vals, Count: 3, Total: 3})
		if r, ok := checkTransfer(t, multi.pol, f.e, f.w); ok || !strings.Contains(r.Claims.Inclusion.Reason, "not exactly the pinned set") {
			t.Fatalf("incomplete validator response accepted: %+v", r.Claims)
		}
	})
	t.Run("one invalid signature beside enough valid power", func(t *testing.T) {
		multi := newTestChain("test-a", 40, 30, 20, 10)
		f := newTransfer(t, multi, txSpec{})
		c := multi.commit(t, f.header)
		c.Commit.Signatures[3].Signature[0] ^= 1 // the 10-power validator
		f.e.Commit = envelope(t, c)
		if r, ok := checkTransfer(t, multi.pol, f.e, f.w); ok || !strings.Contains(r.Claims.Inclusion.Reason, "wrong signature") {
			t.Fatalf("commit with an invalid signature accepted: %+v", r.Claims)
		}
	})
}

func TestEvidenceUnderAnotherChainsPolicy(t *testing.T) {
	a := newTestChain("test-a", 100)
	f := newTransfer(t, a, txSpec{})
	b := a
	b.pol.ChainID = "test-b" // same keys, another chain
	r, ok := checkTransfer(t, b.pol, f.e, f.w)
	if ok || !strings.Contains(r.Claims.Inclusion.Reason, "another chain") || !strings.Contains(r.Claims.Authorisation.Reason, "signature does not verify") {
		t.Fatalf("evidence from test-a accepted under test-b: %+v", r.Claims)
	}
	// The compiled policies refuse disposable keys whatever chain ID they claim.
	for id, p := range compiledPolicies() {
		g := newTransfer(t, newTestChain(id, 100), txSpec{})
		if r, ok := checkTransfer(t, p, g.e, g.w); ok || !strings.Contains(r.Claims.Inclusion.Reason, "not in the pinned set") {
			t.Fatalf("test key trusted under compiled %s policy: %+v", id, r.Claims)
		}
	}
}

func TestInclusionTampering(t *testing.T) {
	chain := newTestChain("test-a", 100)
	cases := []struct {
		name, reason string
		change       func(*transferFixture)
	}{
		{"transaction bytes", "do not hash to --txhash", func(f *transferFixture) { f.tx.Tx[len(f.tx.Tx)-1] ^= 1; f.e.Tx = envelope(t, f.tx) }},
		{"expected hash", "do not hash to --txhash", func(f *transferFixture) { f.w.TxHash = strings.Repeat("f", 64) }},
		{"proof total over the bound", "proof data, index or total", func(f *transferFixture) {
			f.tx.Proof.Proof.Total = maxBlockTxs + 1
			f.e.Tx = envelope(t, f.tx)
		}},
		{"response hash", "do not hash to --txhash", func(f *transferFixture) { f.tx.Hash = bytes.Repeat([]byte{1}, 32); f.e.Tx = envelope(t, f.tx) }},
		{"proof data", "proof data, index or total", func(f *transferFixture) { f.tx.Proof.Data = cmttypes.Tx("other"); f.e.Tx = envelope(t, f.tx) }},
		{"proof root", "Merkle proof", func(f *transferFixture) { f.tx.Proof.RootHash[0] ^= 1; f.e.Tx = envelope(t, f.tx) }},
		{"proof aunt", "Merkle proof", func(f *transferFixture) { f.tx.Proof.Proof.Aunts[0][0] ^= 1; f.e.Tx = envelope(t, f.tx) }},
		{"index", "proof data, index or total", func(f *transferFixture) { f.tx.Index = 0; f.e.Tx = envelope(t, f.tx) }},
		{"index and proof index", "Merkle proof", func(f *transferFixture) {
			f.tx.Index, f.tx.Proof.Proof.Index = 2, 2
			f.e.Tx = envelope(t, f.tx)
		}},
		{"proof total", "proof data, index or total", func(f *transferFixture) { f.tx.Proof.Proof.Total = -1; f.e.Tx = envelope(t, f.tx) }},
		{"no proof", "proof data, index or total", func(f *transferFixture) { f.tx.Proof = cmttypes.TxProof{}; f.e.Tx = envelope(t, f.tx) }},
		{"response height", "transaction response is for height", func(f *transferFixture) { f.tx.Height++; f.e.Tx = envelope(t, f.tx) }},
		{"expected height", "transaction response is for height", func(f *transferFixture) { f.w.Height++ }},
		{"commit for another height", "signed header is for height", func(f *transferFixture) {
			f.header.Height++
			f.e.Commit = envelope(t, chain.commit(t, f.header))
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newTransfer(t, chain, txSpec{})
			tc.change(f)
			r, ok := checkTransfer(t, chain.pol, f.e, f.w)
			if ok || !strings.Contains(r.Claims.Inclusion.Reason, tc.reason) || r.Claims.Execution.Status == statusProven {
				t.Fatalf("want refusal %q: %+v", tc.reason, r.Claims)
			}
			// Authorisation is about the --txhash transaction, never other bytes.
			if tc.reason == "do not hash to --txhash" && tc.name != "response hash" &&
				(r.Claims.Authorisation.Status != statusNotProven || !strings.Contains(r.Claims.Authorisation.Reason, "not the --txhash transaction")) {
				t.Fatalf("other bytes authorised: %+v", r.Claims.Authorisation)
			}
		})
	}
}

func TestAuthorisation(t *testing.T) {
	chain := newTestChain("test-a", 100)
	other := secp256k1.GenPrivKey()
	otherAddress := address(t, other.PubKey().Address())
	send := func(t *testing.T, b *txtypes.TxBody, edit func(*banktypes.MsgSend)) {
		var m banktypes.MsgSend
		if err := proto.Unmarshal(b.Messages[0].Value, &m); err != nil {
			t.Fatal(err)
		}
		edit(&m)
		b.Messages[0].Value = wire(t, &m)
	}
	single := func(mode signing.SignMode) *txtypes.ModeInfo {
		return &txtypes.ModeInfo{Sum: &txtypes.ModeInfo_Single_{Single: &txtypes.ModeInfo_Single{Mode: mode}}}
	}
	cases := []struct {
		name   string
		spec   txSpec
		want   func(*transferWant)
		reason string
	}{
		{"two messages", txSpec{alter: func(b *txtypes.TxBody, a *txtypes.AuthInfo) { b.Messages = append(b.Messages, b.Messages[0]) }}, nil, "2 messages"},
		{"another message type", txSpec{alter: func(b *txtypes.TxBody, a *txtypes.AuthInfo) {
			b.Messages[0] = &codectypes.Any{TypeUrl: "/cosmos.bank.v1beta1.MsgMultiSend", Value: wire(t, &banktypes.MsgMultiSend{})}
		}}, nil, "not supported"},
		{"two coins", txSpec{alter: func(b *txtypes.TxBody, a *txtypes.AuthInfo) {
			send(t, b, func(m *banktypes.MsgSend) {
				m.Amount = sdk.NewCoins(sdk.NewInt64Coin("uother", 1), sdk.NewInt64Coin(nativeDenom, 250))
			})
		}}, nil, "2 coins"},
		{"another denom", txSpec{alter: func(b *txtypes.TxBody, a *txtypes.AuthInfo) {
			send(t, b, func(m *banktypes.MsgSend) { m.Amount = sdk.NewCoins(sdk.NewInt64Coin("uother", 250)) })
		}}, nil, "denom"},
		{"extension options", txSpec{alter: func(b *txtypes.TxBody, a *txtypes.AuthInfo) {
			b.ExtensionOptions = []*codectypes.Any{{TypeUrl: "/test.Extension"}}
		}}, nil, "extension"},
		{"non-critical extension options", txSpec{alter: func(b *txtypes.TxBody, a *txtypes.AuthInfo) {
			b.NonCriticalExtensionOptions = []*codectypes.Any{{TypeUrl: "/test.Extension"}}
		}}, nil, "extension"},
		{"unordered", txSpec{alter: func(b *txtypes.TxBody, a *txtypes.AuthInfo) { b.Unordered = true }}, nil, "unordered"},
		{"timeout timestamp", txSpec{alter: func(b *txtypes.TxBody, a *txtypes.AuthInfo) {
			ts := time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC)
			b.TimeoutTimestamp = &ts
		}}, nil, "timeout timestamps"},
		{"memo not UTF-8", txSpec{memo: "zr1:pay:\xff"}, nil, "UTF-8"},
		{"unknown field in the message wrapper", txSpec{alter: func(b *txtypes.TxBody, a *txtypes.AuthInfo) {
			b.Messages[0].XXX_unrecognized = []byte{0x18, 0x01}
		}}, nil, "unknown fields"},
		{"signer other than the stated sender", txSpec{signer: other}, nil, "not --sender"},
		{"message from another sender", txSpec{signer: other, alter: func(b *txtypes.TxBody, a *txtypes.AuthInfo) {
			send(t, b, func(m *banktypes.MsgSend) { m.FromAddress = otherAddress })
		}}, nil, "message sender"},
		{"multisig signer", txSpec{alter: func(b *txtypes.TxBody, a *txtypes.AuthInfo) {
			pk := kmultisig.NewLegacyAminoPubKey(2, []cryptotypes.PubKey{secp256k1.GenPrivKey().PubKey(), secp256k1.GenPrivKey().PubKey()})
			any, err := codectypes.NewAnyWithValue(pk)
			if err != nil {
				t.Fatal(err)
			}
			a.SignerInfos[0].PublicKey = any
			a.SignerInfos[0].ModeInfo = &txtypes.ModeInfo{Sum: &txtypes.ModeInfo_Multi_{Multi: &txtypes.ModeInfo_Multi{Bitarray: cryptotypes.NewCompactBitArray(2), ModeInfos: []*txtypes.ModeInfo{single(signing.SignMode_SIGN_MODE_DIRECT), single(signing.SignMode_SIGN_MODE_DIRECT)}}}}
		}}, nil, "multisig"},
		{"amino-json signer", txSpec{alter: func(b *txtypes.TxBody, a *txtypes.AuthInfo) {
			a.SignerInfos[0].ModeInfo = single(signing.SignMode_SIGN_MODE_LEGACY_AMINO_JSON)
		}}, nil, "SIGN_MODE_LEGACY_AMINO_JSON"},
		{"ed25519 signer key", txSpec{alter: func(b *txtypes.TxBody, a *txtypes.AuthInfo) {
			a.SignerInfos[0].PublicKey.TypeUrl = "/cosmos.crypto.ed25519.PubKey"
		}}, nil, "key type"},
		{"two signers", txSpec{alter: func(b *txtypes.TxBody, a *txtypes.AuthInfo) { a.SignerInfos = append(a.SignerInfos, a.SignerInfos[0]) }}, nil, "2 signers"},
		{"tip", txSpec{alter: func(b *txtypes.TxBody, a *txtypes.AuthInfo) { a.Tip = &txtypes.Tip{Tipper: otherAddress} }}, nil, "tips"},
		{"wrong account number", txSpec{}, func(w *transferWant) { w.AccountNumber++ }, "signature does not verify"},
		{"wrong expected recipient", txSpec{}, func(w *transferWant) { w.Recipient = otherAddress }, "recipient"},
		{"wrong expected amount", txSpec{}, func(w *transferWant) { w.Amount = "251" }, "amount"},
		{"wrong expected sender", txSpec{}, func(w *transferWant) { w.Sender = otherAddress }, "message sender"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newTransfer(t, chain, tc.spec)
			if tc.want != nil {
				tc.want(&f.w)
			}
			r, ok := checkTransfer(t, chain.pol, f.e, f.w)
			if ok || r.Claims.Inclusion.Status != statusProven || r.Claims.Authorisation.Status != statusNotProven || !strings.Contains(r.Claims.Authorisation.Reason, tc.reason) {
				t.Fatalf("want authorisation refused for %q: ok=%v %+v", tc.reason, ok, r.Claims)
			}
			if r.Claims.TransferEffect.Status == statusProven {
				t.Fatal("effect proven without authorisation")
			}
		})
	}
}

func TestAmbiguousEncodingsRefused(t *testing.T) {
	chain := newTestChain("test-a", 100)
	f := newTransfer(t, chain, txSpec{})
	var tx txtypes.TxRaw
	if err := proto.Unmarshal(f.tx.Tx, &tx); err != nil {
		t.Fatal(err)
	}
	badSig := tx
	badSig.Signatures = [][]byte{append([]byte{}, tx.Signatures[0]...)}
	badSig.Signatures[0][1] ^= 1
	for name, raw := range map[string][]byte{
		"unknown TxRaw field":  append(append([]byte{}, f.tx.Tx...), 0x20, 0x01),
		"duplicate body field": append(append([]byte{}, f.tx.Tx...), 0x0a, 0x00),
		"altered signature":    wire(t, &badSig),
	} {
		if _, err := verifyTransaction(raw, chain.pol.ChainID, f.w); err == nil {
			t.Fatalf("%s accepted", name)
		}
	}
	// A signed public-key wrapper with unknown fields is canonical on re-encoding
	// because Any keeps the bytes; it must still be refused.
	g := newTransfer(t, chain, txSpec{alter: func(b *txtypes.TxBody, a *txtypes.AuthInfo) {
		a.SignerInfos[0].PublicKey.XXX_unrecognized = []byte{0x18, 0x01}
	}})
	if _, err := verifyTransaction(g.tx.Tx, chain.pol.ChainID, g.w); err == nil || !strings.Contains(err.Error(), "unknown fields") {
		t.Fatalf("signed unknown public-key fields accepted: %v", err)
	}
}

func TestReportedTermsAreNotRefused(t *testing.T) {
	chain := newTestChain("test-a", 100)
	granter := address(t, secp256k1.GenPrivKey().PubKey().Address())
	f := newTransfer(t, chain, txSpec{alter: func(b *txtypes.TxBody, a *txtypes.AuthInfo) {
		b.TimeoutHeight = 4000
		a.Fee.Granter = granter
		a.Fee.Payer = granter
		a.Fee.GasLimit = 1
		a.Fee.Amount = sdk.NewCoins(sdk.NewInt64Coin("uother", 3), sdk.NewInt64Coin(nativeDenom, 5))
	}})
	f.w.RequireExecution = false
	f.e.BlockResults, f.e.NextCommit, f.e.NextValidators = nil, nil, nil
	r, ok := checkTransfer(t, chain.pol, f.e, f.w)
	if !ok || r.Claims.Authorisation.Status != statusProven {
		t.Fatalf("reported terms refused: %+v", r.Claims)
	}
	x := r.Transaction
	fee, _ := x["fee"].([]string)
	if x["timeout_height"] != "4000" || x["fee_granter"] != granter || x["fee_payer"] != granter || x["gas_limit"] != "1" || len(fee) != 2 || fee[0] != "3uother" || fee[1] != "5uzrn" {
		t.Fatalf("terms not reported: %+v", x)
	}
}

func TestMemoGrammar(t *testing.T) {
	hex64, hex32 := strings.Repeat("ab", 32), strings.Repeat("cd", 16)
	self := "zrn1self"
	anchor := "zr1:anc:" + hex32 + ":7:" + hex64
	cases := []struct {
		memo, sender, recipient, amount, class string
	}{
		{"", "a", "b", "5", "none"},
		{"zr1:pay:" + hex64, "a", "b", "5", "pay"},
		{anchor, self, self, "1", "anc"},
		{"zr1:anc:" + hex32 + ":1:" + hex64, self, self, "1", "anc"},
		{"zr1:anc:" + hex32 + ":18446744073709551615:" + hex64, self, self, "1", "anc"},
		{anchor, self, "zrn1other", "1", "other"},
		{anchor, self, self, "2", "other"},
		{"zr1:anc:" + hex32 + ":0:" + hex64, self, self, "1", "other"},
		{"zr1:anc:" + hex32 + ":07:" + hex64, self, self, "1", "other"},
		{"zr1:anc:" + hex32 + ":18446744073709551616:" + hex64, self, self, "1", "other"},
		{"zr1:anc:" + hex32[1:] + ":7:" + hex64, self, self, "1", "other"},
		{"zr1:anc:" + strings.ToUpper(hex32) + ":7:" + hex64, self, self, "1", "other"},
		{"zr1:anc:" + hex32 + ":7:" + hex64 + ":", self, self, "1", "other"},
		{"zr1:pay:" + hex64[1:], "a", "b", "5", "other"},
		{"zr1:pay:" + strings.ToUpper(hex64), "a", "b", "5", "other"},
		{"zr1:pay:" + hex64 + "\n", "a", "b", "5", "other"},
		{" zr1:pay:" + hex64, "a", "b", "5", "other"},
		{"zr1:PAY:" + hex64, "a", "b", "5", "other"},
		{"zr1:pay:" + hex64[:63] + "g", "a", "b", "5", "other"},
		{"hello", "a", "b", "5", "other"},
	}
	for _, tc := range cases {
		if got, _ := classifyMemo(tc.memo, tc.sender, tc.recipient, tc.amount); got != tc.class {
			t.Fatalf("memo %q (%s->%s, %s) classified %s, want %s", tc.memo, tc.sender, tc.recipient, tc.amount, got, tc.class)
		}
	}
}

func TestMemoExpectations(t *testing.T) {
	chain := newTestChain("test-a", 100)
	digestText := strings.Repeat("0f", 32)
	empty, exact, wrong := "", "zr1:pay:"+digestText, "zr1:pay:"+strings.Repeat("1f", 32)
	cases := []struct {
		name    string
		memo    string
		want    *string
		pay     string
		ok      bool
		class   string
		fieldOK bool
	}{
		{"no expectation", "free text", nil, "", true, "other", false},
		{"pay digest", exact, nil, digestText, true, "pay", true},
		{"wrong pay digest", wrong, nil, digestText, false, "", false},
		{"exact memo", exact, &exact, "", true, "pay", true},
		{"memo differs", exact + " ", &exact, "", false, "", false},
		{"empty memo expected", "", &empty, "", true, "none", false},
		{"empty memo expected but present", "x", &empty, "", false, "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newTransfer(t, chain, txSpec{memo: tc.memo})
			f.w.Memo, f.w.PayDigest = tc.want, tc.pay
			r, ok := checkTransfer(t, chain.pol, f.e, f.w)
			if ok != tc.ok || (r.Claims.Authorisation.Status == statusProven) != tc.ok {
				t.Fatalf("ok=%v %+v", ok, r.Claims.Authorisation)
			}
			if tc.ok && (r.Transaction["memo_class"] != tc.class || r.Transaction["memo"] != tc.memo || (r.Transaction["memo_fields"] != nil) != tc.fieldOK) {
				t.Fatalf("memo facts: %+v", r.Transaction)
			}
		})
	}
	t.Run("anchor memo", func(t *testing.T) {
		memo := "zr1:anc:" + strings.Repeat("cd", 16) + ":3:" + strings.Repeat("ab", 32)
		for _, tc := range []struct {
			name  string
			f     *transferFixture
			class string
		}{
			{"1 uzrn to itself", newSelfTransfer(t, chain, memo, 1), "anc"},
			{"2 uzrn to itself", newSelfTransfer(t, chain, memo, 2), "other"},
			{"1 uzrn to another account", newTransfer(t, chain, txSpec{memo: memo, amount: 1}), "other"},
		} {
			r, ok := checkTransfer(t, chain.pol, tc.f.e, tc.f.w)
			if !ok || r.Transaction["memo_class"] != tc.class {
				t.Fatalf("%s: ok=%v class %v, want %s", tc.name, ok, r.Transaction["memo_class"], tc.class)
			}
		}
	})
}

// newSelfTransfer builds a transfer whose recipient is its own signer.
func newSelfTransfer(t *testing.T, chain testChain, memo string, amount int64) *transferFixture {
	t.Helper()
	key := secp256k1.GenPrivKey()
	self := address(t, key.PubKey().Address())
	f := newTransfer(t, chain, txSpec{signer: key, recipient: self, memo: memo, amount: amount, alter: func(b *txtypes.TxBody, a *txtypes.AuthInfo) {
		var m banktypes.MsgSend
		if err := proto.Unmarshal(b.Messages[0].Value, &m); err != nil {
			t.Fatal(err)
		}
		m.FromAddress = self
		b.Messages[0].Value = wire(t, &m)
	}})
	f.w.Sender = self
	return f
}

func TestExecution(t *testing.T) {
	chain := newTestChain("test-a", 100)
	t.Run("non-zero result code", func(t *testing.T) {
		f := newTransfer(t, chain, txSpec{code: 5})
		r, ok := checkTransfer(t, chain.pol, f.e, f.w)
		if ok || r.Claims.Execution.Status != statusProven || r.Claims.TransferEffect.Status != statusNotProven || r.Execution["code"] != uint32(5) {
			t.Fatalf("failed transaction reported as effective: ok=%v %+v", ok, r.Claims)
		}
	})
	t.Run("uncommitted fields changed", func(t *testing.T) {
		f := newTransfer(t, chain, txSpec{})
		f.results.TxsResults[1].Log = "another log"
		f.results.TxsResults[1].Events = nil
		f.e.BlockResults = envelope(t, f.results)
		if _, ok := checkTransfer(t, chain.pol, f.e, f.w); !ok {
			t.Fatal("uncommitted fields changed the committed result")
		}
	})
	cases := []struct {
		name, reason string
		change       func(*transferFixture)
	}{
		{"block results do not rebuild", "do not rebuild", func(f *transferFixture) {
			f.results.TxsResults[1].GasUsed++
			f.e.BlockResults = envelope(t, f.results)
		}},
		{"another transaction's code changed", "do not rebuild", func(f *transferFixture) {
			f.results.TxsResults[0].Code = 1
			f.e.BlockResults = envelope(t, f.results)
		}},
		{"block results missing one", "count differs", func(f *transferFixture) {
			f.results.TxsResults = f.results.TxsResults[:2]
			f.e.BlockResults = envelope(t, f.results)
		}},
		{"block results with a null entry", "empty result", func(f *transferFixture) {
			f.results.TxsResults[0] = nil
			f.e.BlockResults = envelope(t, f.results)
		}},
		{"block results for another height", "block results are for height", func(f *transferFixture) {
			f.results.Height++
			f.e.BlockResults = envelope(t, f.results)
		}},
		{"endpoint result differs from the committed one", "differs from the committed result", func(f *transferFixture) {
			f.tx.TxResult.GasUsed++
			f.e.Tx = envelope(t, f.tx)
		}},
		{"H names other next validators", "next validators hash at H", func(f *transferFixture) {
			f.header.NextValidatorsHash = bytes.Repeat([]byte{5}, 32)
			f.seal(t)
		}},
		{"H+1 does not name H", "does not name the block at H", func(f *transferFixture) {
			f.next.LastBlockID.Hash = bytes.Repeat([]byte{7}, 32)
			f.e.NextCommit = envelope(t, f.chain.commit(t, f.next))
		}},
		{"H+2 supplied as H+1", "header at H+1: signed header is for height", func(f *transferFixture) {
			f.next.Height++
			f.e.NextCommit = envelope(t, f.chain.commit(t, f.next))
			f.e.NextValidators = f.chain.validators(t, f.next.Height)
		}},
		{"H supplied as H+1", "header at H+1: signed header is for height", func(f *transferFixture) {
			f.e.NextCommit, f.e.NextValidators = f.e.Commit, f.e.Validators
		}},
		{"H+1 signed by a key outside the policy", "header at H+1: commit signatures", func(f *transferFixture) {
			other := newTestChain("test-a", 100)
			f.e.NextCommit = envelope(t, other.commit(t, f.next))
		}},
		{"partial execution evidence", "must all be supplied", func(f *transferFixture) { f.e.NextValidators = nil }},
		{"inclusion not proven", "inclusion is not proven", func(f *transferFixture) { f.w.TxHash = strings.Repeat("0", 64) }},
		{"transaction response unreadable", "inclusion is not proven", func(f *transferFixture) { f.e.Tx = []byte(`{"jsonrpc":"2.0","result":{"tx":7}}`) }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newTransfer(t, chain, txSpec{})
			tc.change(f)
			r, ok := checkTransfer(t, chain.pol, f.e, f.w)
			if ok || !strings.Contains(r.Claims.Execution.Reason, tc.reason) || r.Claims.TransferEffect.Status == statusProven || r.Execution != nil {
				t.Fatalf("want refusal %q: %+v", tc.reason, r.Claims)
			}
		})
	}
}
