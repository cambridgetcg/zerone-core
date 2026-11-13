package main

import (
	"bytes"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"time"

	"cosmossdk.io/store/rootmulti"
	storetypes "cosmossdk.io/store/types"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/crypto/merkle"
	rpctypes "github.com/cometbft/cometbft/rpc/core/types"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
)

const (
	stateSchema = "zerone.ledger-proof/state/v1"
	// Key layouts, the same in cosmos-sdk v0.50.15 and v0.53.8:
	//   x/bank balance: 0x02 | len(address) | address | denom  (store "bank")
	//   x/auth account: 0x01 | address                        (store "acc")
	balancesPrefix = 0x02
	accountsPrefix = 0x01
	baseAccountURL = "/cosmos.auth.v1beta1.BaseAccount"
)

// x/bank stores a balance as math.Int text and deletes a zero balance.
var balanceText = regexp.MustCompile(`\A(0|[1-9][0-9]{0,77})\z`)

type stateWant struct {
	Height  int64
	Kind    string // "balance" or "account"
	Address string
	Denom   string // balance only
}

type stateEvidence struct {
	Query, NextCommit, NextValidators []byte
}

type stateClaims struct {
	AppHash       claim  `json:"app_hash"`
	StoredValue   claim  `json:"stored_value"`
	Balance       *claim `json:"balance,omitempty"`
	Account       *claim `json:"account,omitempty"`
	AccountNumber *claim `json:"account_number,omitempty"`
	Sequence      *claim `json:"sequence,omitempty"`
}

type stateReport struct {
	Schema      string            `json:"schema"`
	ChainID     string            `json:"chain_id"`
	Height      string            `json:"height"`
	AttestedBy  attestation       `json:"attested_by"`
	Claims      stateClaims       `json:"claims"`
	Expected    map[string]any    `json:"expected"`
	Store       string            `json:"store"`
	KeyHex      string            `json:"key_hex"`
	Header      map[string]any    `json:"header,omitempty"`
	Balance     map[string]any    `json:"balance,omitempty"`
	Account     map[string]any    `json:"account,omitempty"`
	InputSHA256 map[string]string `json:"input_sha256"`
}

func runState(args []string, policies map[string]policy) (any, bool, error) {
	files := []string{"query", "next-commit", "next-validators"}
	v, err := parseFlags("state", args, append([]string{"chain", "height", "kind", "address", "denom"}, files...), nil)
	if err != nil {
		return nil, false, err
	}
	if err := requireFlags(v, "chain", "query", "next-commit", "next-validators", "height", "kind", "address"); err != nil {
		return nil, false, err
	}
	p, err := lookupPolicy(policies, v["chain"].value)
	if err != nil {
		return nil, false, err
	}
	w := stateWant{Kind: v["kind"].value, Address: v["address"].value, Denom: v["denom"].value}
	if w.Height, err = parseHeight(v["height"].value); err != nil {
		return nil, false, err
	}
	switch w.Kind {
	case "balance":
		if !v["denom"].given {
			w.Denom = nativeDenom
		}
		if err := sdk.ValidateDenom(w.Denom); err != nil {
			return nil, false, usagef("--denom: %v", err)
		}
	case "account":
		if v["denom"].given {
			return nil, false, usageError("--denom applies only to --kind balance")
		}
	default:
		return nil, false, usageError("--kind must be balance or account")
	}
	if _, _, err := stateKey(w); err != nil {
		return nil, false, usagef("--address: %v", err)
	}
	in, err := readFiles(v, files...)
	if err != nil {
		return nil, false, err
	}
	r, ok := verifyState(p, stateEvidence{Query: in["query"], NextCommit: in["next-commit"], NextValidators: in["next-validators"]}, w)
	return r, ok, nil
}

// stateKey derives the store and key from the kind and address; the caller
// never supplies them.
func stateKey(w stateWant) (string, []byte, error) {
	addr, err := parseAddress(w.Address, 20, 32)
	if err != nil {
		return "", nil, err
	}
	switch w.Kind {
	case "balance":
		key := append([]byte{balancesPrefix, byte(len(addr))}, addr...)
		return "bank", append(key, w.Denom...), nil
	case "account":
		return "acc", append([]byte{accountsPrefix}, addr...), nil
	}
	return "", nil, errors.New("unknown kind")
}

func verifyState(p policy, e stateEvidence, w stateWant) (stateReport, bool) {
	r := stateReport{Schema: stateSchema, ChainID: p.ChainID, Height: strconv.FormatInt(w.Height, 10), AttestedBy: p.attestation(),
		Expected:    map[string]any{"kind": w.Kind, "address": w.Address},
		InputSHA256: inputHashes(map[string][]byte{"query": e.Query, "next_commit": e.NextCommit, "next_validators": e.NextValidators})}
	unproven := notProven("the stored value is not proven")
	if w.Kind == "balance" {
		r.Expected["denom"] = w.Denom
		r.Claims.Balance = &unproven
	} else {
		a, n, s := unproven, unproven, unproven
		r.Claims.Account, r.Claims.AccountNumber, r.Claims.Sequence = &a, &n, &s
	}
	store, key, err := stateKey(w)
	if err != nil {
		r.Claims.AppHash, r.Claims.StoredValue = notProven(err.Error()), notProven(err.Error())
		return r, false
	}
	r.Store, r.KeyHex = store, upperHex(key)

	next, err := trustedHeader(p, e.NextCommit, e.NextValidators, w.Height+1)
	if err != nil {
		r.Claims.AppHash = notProven("header at H+1: " + err.Error())
	} else {
		r.Claims.AppHash = proven()
		r.Header = map[string]any{"height": strconv.FormatInt(next.Height, 10), "header_hash": upperHex(next.Hash()), "app_hash": upperHex(next.AppHash), "time": next.Time.UTC().Format(time.RFC3339Nano)}
	}
	res, err := parseQuery(e.Query, w.Height, store, key)
	switch {
	case err != nil:
		r.Claims.StoredValue = notProven(err.Error())
	case next == nil:
		r.Claims.StoredValue = notProven("the app hash at H+1 is not proven")
	default:
		if err := verifyStoreProof(res, store, key, next.AppHash); err != nil {
			r.Claims.StoredValue = notProven("store proof: " + err.Error())
		} else {
			r.Claims.StoredValue = proven()
		}
	}
	if r.Claims.StoredValue.Status == statusProven {
		if w.Kind == "balance" {
			readBalance(&r, w, res.Value)
		} else {
			readAccount(&r, w, res.Value)
		}
	}
	ok := r.Claims.AppHash.Status == statusProven && r.Claims.StoredValue.Status == statusProven
	for _, c := range []*claim{r.Claims.Balance, r.Claims.Account, r.Claims.AccountNumber, r.Claims.Sequence} {
		if c != nil && c.Status == statusNotProven {
			ok = false
		}
	}
	return r, ok
}

func parseQuery(raw []byte, height int64, store string, key []byte) (*abci.ResponseQuery, error) {
	var q rpctypes.ResultABCIQuery
	if err := rpcResult(raw, &q); err != nil {
		return nil, fmt.Errorf("query response: %w", err)
	}
	res := q.Response
	if res.Code != 0 {
		return nil, fmt.Errorf("query response has code %d", res.Code)
	}
	if res.Height != height {
		return nil, fmt.Errorf("query response is for height %d, not %d", res.Height, height)
	}
	if !bytes.Equal(res.Key, key) {
		return nil, errors.New("query response is for another key than this kind and address")
	}
	if res.ProofOps == nil || len(res.ProofOps.Ops) != 2 {
		return nil, errors.New("query response does not carry exactly two proof operations; fetch it with prove=true")
	}
	if op := res.ProofOps.Ops[0]; op.Type != storetypes.ProofOpIAVLCommitment || !bytes.Equal(op.Key, key) {
		return nil, errors.New("first proof operation is not an IAVL proof for this key")
	}
	if op := res.ProofOps.Ops[1]; op.Type != storetypes.ProofOpSimpleMerkleCommitment || string(op.Key) != store {
		return nil, fmt.Errorf("second proof operation is not a multistore proof for store %q", store)
	}
	return &res, nil
}

// verifyStoreProof runs the store's standard ICS-23 proof runtime: an IAVL
// existence or absence proof, then the multistore proof to the app hash.
func verifyStoreProof(res *abci.ResponseQuery, store string, key, appHash []byte) (err error) {
	defer func() {
		if recover() != nil {
			err = errors.New("proof could not be evaluated")
		}
	}()
	path := merkle.KeyPath{}.AppendKey([]byte(store), merkle.KeyEncodingURL).AppendKey(key, merkle.KeyEncodingHex).String()
	prt := rootmulti.DefaultProofRuntime()
	if len(res.Value) == 0 {
		return prt.VerifyAbsence(res.ProofOps, appHash, path)
	}
	return prt.VerifyValue(res.ProofOps, appHash, path, res.Value)
}

func readBalance(r *stateReport, w stateWant, value []byte) {
	switch {
	case len(value) == 0:
		r.Balance = map[string]any{"denom": w.Denom, "amount": "0", "stored": false}
	case balanceText.Match(value):
		r.Balance = map[string]any{"denom": w.Denom, "amount": string(value), "stored": true}
	default:
		c := notProven("stored value is not a canonical decimal amount")
		r.Claims.Balance = &c
		return
	}
	c := proven()
	r.Claims.Balance = &c
}

func readAccount(r *stateReport, w stateWant, value []byte) {
	set := func(account, number, sequence claim) {
		r.Claims.Account, r.Claims.AccountNumber, r.Claims.Sequence = &account, &number, &sequence
	}
	if len(value) == 0 {
		r.Account = map[string]any{"exists": false}
		none := notApplicable("no account is stored at this address")
		set(proven(), none, none)
		return
	}
	var a codectypes.Any
	if err := canonicalProto(value, &a); err != nil || len(a.XXX_unrecognized) != 0 {
		bad := notProven("stored account is not a canonical Any")
		set(bad, bad, bad)
		return
	}
	r.Account = map[string]any{"exists": true, "type_url": a.TypeUrl}
	if a.TypeUrl != baseAccountURL {
		other := notProven(fmt.Sprintf("account type %s is not decoded by this tool", a.TypeUrl))
		set(proven(), other, other)
		return
	}
	var b authtypes.BaseAccount
	if err := canonicalProto(a.Value, &b); err != nil {
		bad := notProven("stored base account: " + err.Error())
		set(bad, bad, bad)
		return
	}
	if b.Address != w.Address {
		bad := notProven("stored account names another address")
		set(bad, bad, bad)
		return
	}
	r.Account["address"] = b.Address
	r.Account["account_number"] = strconv.FormatUint(b.AccountNumber, 10)
	r.Account["sequence"] = strconv.FormatUint(b.Sequence, 10)
	set(proven(), proven(), proven())
}
