package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	errorsmod "cosmossdk.io/errors"
	feegrant "cosmossdk.io/x/feegrant"
	storekeyring "github.com/99designs/keyring"
	abci "github.com/cometbft/cometbft/abci/types"
	cmtbytes "github.com/cometbft/cometbft/libs/bytes"
	cmtjson "github.com/cometbft/cometbft/libs/json"
	rpctypes "github.com/cometbft/cometbft/rpc/core/types"
	cmttypes "github.com/cometbft/cometbft/types"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	pottypes "github.com/zerone-chain/zerone/x/claiming_pot/types"
	vesttypes "github.com/zerone-chain/zerone/x/vesting_rewards/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var updateVector = flag.Bool("update-seed-vector", false, "regenerate the public native test-key vector only")
var fixtureNow = time.Date(2030, 1, 2, 3, 4, 5, 0, time.UTC)

func publicTestKey() *secp256k1.PrivKey {
	b := make([]byte, 32)
	b[31] = 1
	return &secp256k1.PrivKey{Key: b}
}
func round(o object) object    { return parseJSON(canonical(o)) }
func setID(o object, k string) { o[k] = hashObject(o, k) }
func fixture(pub []byte) (object, object, object) {
	chain := "cosmos:seed-local-vector"
	claimant := sdk.AccAddress((&secp256k1.PubKey{Key: pub}).Address()).String()
	sponsor := sdk.AccAddress(bytes.Repeat([]byte{7}, 20)).String()
	p := object{"protocol": "agent-wallet-zerone.seed-profile/0.1", "chain_reference": "seed-local-vector", "chain_id": chain, "native_asset_id": chain + "/denom:uzrn", "claiming_pot_account": chain + ":" + authtypes.NewModuleAddress("claiming_pot").String(), "genesis_hash": digest([]byte("public-test-genesis")), "source_digest": digest([]byte("public-test-source")), "zerone_core_commit": "89553a0132dafa1ba9670f53b8a3195b33a6e720", "cosmos_sdk_version": "v0.53.8", "runtime_sha256": digest([]byte("public-test-runtime")), "helper_sha256": digest([]byte("public-test-helper")), "native_denom": "uzrn", "bech32_prefix": "zrn", "seed_amount_uzrn": "222000", "claim_type_url": claimURL, "claim_gas_floor": "22222", "tx_gas_cap": "11111111", "min_gas_price_uzrn": "1", "confirmation_depth": 1}
	setID(p, "profile_id")
	p = round(p)
	pol := object{"protocol": "agent-wallet-zerone.seed-policy/0.1", "profile_id": p["profile_id"], "node_trust_id": digest([]byte("public-test-node")), "claimant_account": chain + ":" + claimant, "sponsor_account": chain + ":" + sponsor, "pot_id": "bootstrap-" + claimant, "max_intents": 1, "seed_amount_uzrn": "222000", "max_fee_uzrn": "300000", "max_gas": "300000", "grant_spend_limit_uzrn": "600000", "grant_expires_at": "2030-01-02T04:00:00.000Z", "setup_fee_budget_uzrn": "200000", "not_before": "2030-01-02T03:00:00.000Z", "expires_at": "2030-01-02T03:59:00.000Z", "timeout_height": "1000", "max_observation_age_seconds": 300, "max_height_lag": 100}
	setID(pol, "policy_hash")
	pol = round(pol)
	c := object{"protocol": "agent-wallet-zerone.seed-commitment/0.1", "capability_record_id": digest([]byte("public-test-capability")), "intent_record_id": digest([]byte("public-test-intent")), "signer_key_id": digest(pub), "signer_public_key_b64u": b64(pub), "account_number": "7", "sequence": "0", "fee_amount_uzrn": "250000", "gas_limit": "250000"}
	for _, k := range []string{"profile_id", "source_digest", "genesis_hash", "chain_id", "chain_reference"} {
		c[k] = p[k]
	}
	for _, k := range []string{"policy_hash", "claimant_account", "sponsor_account", "pot_id", "timeout_height", "expires_at", "grant_spend_limit_uzrn", "grant_expires_at"} {
		c[k] = pol[k]
	}
	plan := object{"protocol": "agent-wallet-zerone.seed-plan/0.1", "commitment": c, "commitment_hash": hashObject(c, ""), "observation_hash": digest([]byte("public-test-observation"))}
	body, auth, doc, sim := buildUnsigned(c)
	for k, b := range map[string][]byte{"body": body, "auth_info": auth, "sign_doc": doc, "simulation_tx": sim} {
		plan[k+"_bytes_b64u"] = b64(b)
		plan[k+"_bytes_hash"] = digest(b)
	}
	setID(plan, "plan_id")
	return p, pol, round(plan)
}
func signedFixture(plan object) []byte {
	body, auth, doc, _ := buildUnsigned(obj(plan["commitment"]))
	sig, e := publicTestKey().Sign(doc)
	if e != nil {
		panic(e)
	}
	return marshalNative(&txtypes.TxRaw{BodyBytes: body, AuthInfoBytes: auth, Signatures: [][]byte{sig}})
}
func mustFail(t *testing.T, fn func()) any {
	t.Helper()
	var got any
	func() { defer func() { got = recover() }(); fn() }()
	if got == nil {
		t.Fatal("accepted invalid input")
	}
	return got
}
func privateDir(t *testing.T) string {
	t.Helper()
	dir, e := filepath.EvalSymlinks(t.TempDir())
	if e != nil {
		t.Fatal(e)
	}
	if e = os.Chmod(dir, 0700); e != nil {
		t.Fatal(e)
	}
	return dir
}
func request(command string, p, pol, plan object) object {
	r := object{"protocol": protocol, "request_id": "test-1", "timeout_ms": 30000, "command": command, "profile": p, "policy": pol}
	if plan != nil {
		r["plan"] = plan
	}
	return r
}

func TestNativeVector(t *testing.T) {
	p, pol, plan := fixture(publicTestKey().PubKey().Bytes())
	validateProfile(p)
	validatePolicy(p, pol)
	validatePlan(p, pol, plan)
	tx := signedFixture(plan)
	summary := signedSummary(plan, tx)
	c := obj(plan["commitment"])
	claim := marshalNative(&pottypes.MsgClaim{Claimant: rawAccount(str(c, "claimant_account"), str(c, "chain_id")), PotId: str(c, "pot_id")})
	v := object{"fixture": "public-disposable-test-key-only", "cosmos_sdk_version": "v0.53.8", "feegrant_version": "v0.2.0", "profile": p, "policy": pol, "plan": plan, "claim_value_b64u": b64(claim), "signed_tx_b64u": b64(tx), "summary": summary}
	got := canonical(v)
	if *updateVector {
		if e := os.MkdirAll("testdata", 0755); e != nil {
			t.Fatal(e)
		}
		if e := os.WriteFile("testdata/claim-vector.json", got, 0644); e != nil {
			t.Fatal(e)
		}
	}
	want, e := os.ReadFile("testdata/claim-vector.json")
	if e != nil {
		t.Fatal(e)
	}
	if !bytes.Equal(want, got) {
		t.Fatal("native Claim/granter/SignDoc/TxRaw golden mismatch")
	}
	var ai txtypes.AuthInfo
	if ai.Unmarshal(unb64(str(plan, "auth_info_bytes_b64u"))) != nil || ai.Fee.Payer != "" || ai.Fee.Granter != rawAccount(str(pol, "sponsor_account"), str(p, "chain_id")) {
		t.Fatal("native granter mismatch")
	}
}
func TestClosedCanonicalWireAndPlan(t *testing.T) {
	for _, s := range []string{`{"a":1,"a":2}`, `{"a":-0}`, `{"a":1.0}`, `{"a":9007199254740992}`, `{"a":"\ud800"}`, ` {"a":1}`, `{"b":1,"a":2}`} {
		t.Run(s, func(t *testing.T) { mustFail(t, func() { parseJSON([]byte(s)) }) })
	}
	p, pol, plan := fixture(publicTestKey().PubKey().Bytes())
	for _, field := range []string{"pot_id", "sponsor_account", "account_number", "sequence", "fee_amount_uzrn", "gas_limit", "timeout_height", "policy_hash", "signer_public_key_b64u", "expires_at"} {
		t.Run(field, func(t *testing.T) {
			bad := round(plan)
			obj(bad["commitment"])[field] = "1"
			bad["commitment_hash"] = hashObject(obj(bad["commitment"]), "")
			setID(bad, "plan_id")
			mustFail(t, func() { validatePlan(p, pol, bad) })
		})
	}
	bad := round(plan)
	bad["extra"] = true
	mustFail(t, func() { validatePlan(p, pol, bad) })
	for _, field := range []string{"body", "auth_info", "sign_doc", "simulation_tx"} {
		bad := round(plan)
		b := append(unb64(str(bad, field+"_bytes_b64u")), 0x78, 0)
		bad[field+"_bytes_b64u"] = b64(b)
		bad[field+"_bytes_hash"] = digest(b)
		setID(bad, "plan_id")
		mustFail(t, func() { validatePlan(p, pol, bad) })
	}
	p["cosmos_sdk_version"] = "v0.50.15"
	setID(p, "profile_id")
	mustFail(t, func() { validateProfile(p) })
}
func TestNarrowedIntentExpiry(t *testing.T) {
	p, pol, plan := fixture(publicTestKey().PubKey().Bytes())
	for _, test := range []struct {
		expiry string
		valid  bool
	}{
		{"2030-01-02T03:30:00.000Z", true},
		{str(pol, "expires_at"), true},
		{str(pol, "not_before"), false},
		{"2030-01-02T04:00:00.000Z", false},
	} {
		candidate := round(plan)
		c := obj(candidate["commitment"])
		c["expires_at"] = test.expiry
		candidate["commitment_hash"] = hashObject(c, "")
		setID(candidate, "plan_id")
		if test.valid {
			validatePlan(p, pol, candidate)
		} else {
			mustFail(t, func() { validatePlan(p, pol, candidate) })
		}
	}
}

func TestSignatureAndFileSubstitution(t *testing.T) {
	_, _, plan := fixture(publicTestKey().PubKey().Bytes())
	tx := signedFixture(plan)
	signedSummary(plan, tx)
	bad := append(append([]byte{}, tx...), 0x20, 1)
	mustFail(t, func() { signedSummary(plan, bad) })
	var raw txtypes.TxRaw
	raw.Unmarshal(tx)
	order, _ := new(big.Int).SetString("FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFEBAAEDCE6AF48A03BBFD25E8CD0364141", 16)
	s := new(big.Int).SetBytes(raw.Signatures[0][32:])
	order.Sub(order, s).FillBytes(raw.Signatures[0][32:])
	mustFail(t, func() { signedSummary(plan, marshalNative(&raw)) })
	dir := privateDir(t)
	path := filepath.Join(dir, "signed.bin")
	writePrivateExclusive(path, tx)
	signedSummary(plan, readPrivate(context.Background(), path, maxProto))
	mustFail(t, func() { writePrivateExclusive(path, tx) })
	os.Chmod(path, 0644)
	mustFail(t, func() { readPrivate(context.Background(), path, maxProto) })
	os.Chmod(path, 0600)
	link := filepath.Join(dir, "link")
	os.Symlink(path, link)
	mustFail(t, func() { readPrivate(context.Background(), link, maxProto) })
	hard := filepath.Join(dir, "hard")
	os.Link(path, hard)
	mustFail(t, func() { readPrivate(context.Background(), path, maxProto) })
	os.Remove(hard)
	os.Chmod(dir, 0755)
	mustFail(t, func() { readPrivate(context.Background(), path, maxProto) })
}
func TestSelectedKeyringCannotMigrateOrWrite(t *testing.T) {
	source := storekeyring.NewArrayKeyring([]storekeyring.Item{{Key: "claimant.info", Data: []byte("intentionally unsupported legacy record")}, {Key: "other.info", Data: []byte("other")}})
	bounded := selectedReadOnlyKeyring{source: source, selected: "claimant.info"}
	if _, err := bounded.Get("other.info"); err == nil {
		t.Fatal("other key exposed")
	}
	if err := bounded.Set(storekeyring.Item{Key: "claimant.info", Data: []byte("replacement")}); err == nil {
		t.Fatal("write allowed")
	}
	if err := bounded.Remove("claimant.info"); err == nil {
		t.Fatal("delete allowed")
	}
	if _, err := bounded.Keys(); err == nil {
		t.Fatal("list allowed")
	}
	kr := keyring.NewInMemoryWithKeyring(bounded, keyCodec())
	if err := kr.ImportPrivKeyHex("new", hex.EncodeToString(publicTestKey().Key), "secp256k1"); err == nil {
		t.Fatal("SDK import escaped read-only seam")
	}
	if _, err := kr.Key("claimant"); err == nil {
		t.Fatal("unsupported record migrated")
	}
	item, err := source.Get("claimant.info")
	if err != nil || string(item.Data) != "intentionally unsupported legacy record" {
		t.Fatal("legacy record changed")
	}
}

func TestInputAndResponseBounds(t *testing.T) {
	for _, b := range [][]byte{bytes.Repeat([]byte("x"), maxJSON+2), []byte(strings.Repeat("[", 33) + "0" + strings.Repeat("]", 33)), canonical(object{"value": strings.Repeat("x", 4097)})} {
		mustFail(t, func() { parseJSON(b) })
	}
	for _, b := range [][]byte{[]byte(`{"result":1,"result":2}`), []byte(strings.Repeat("[", 33) + "0" + strings.Repeat("]", 33)), canonical(object{"value": strings.Repeat("x", 4097)})} {
		mustFail(t, func() { validateNodeJSON(b) })
	}
	validateNodeJSON([]byte(` { "result": 1 } `))
	got := parseJSON([]byte("{\"value\":\"<>&\"}\n"))
	if str(got, "value") != "<>&" {
		t.Fatal("canonical HTML characters changed")
	}
	f := newFake(t)
	r := f.req("inspect")
	for _, host := range []string{"localhost", "192.0.2.1"} {
		bad := round(r)
		obj(bad["node"])["rpc_url"] = "http://" + host + ":26657"
		mustFail(t, func() { newNode(context.Background(), bad, time.Now) })
	}
}

func TestDisposableExistingCosmosKeyringSignsActualBytes(t *testing.T) {
	home := privateDir(t)
	kr, e := keyring.New(sdk.KeyringServiceName(), keyring.BackendTest, home, bytes.NewReader(nil), keyCodec())
	if e != nil {
		t.Fatal(e)
	}
	// Key creation is test-only, never a production helper command. Public fixed
	// scalar fixture is not used here: exercise real SDK key creation in TempDir.
	rec, _, e := kr.NewMnemonic("claimant", keyring.English, sdk.FullFundraiserPath, "", hd.Secp256k1)
	if e != nil {
		t.Fatal(e)
	}
	pub, e := rec.GetPubKey()
	if e != nil {
		t.Fatal(e)
	}
	p, pol, plan := fixture(pub.Bytes())
	r := request("sign", p, pol, plan)
	r["keyring"] = object{"backend": "test", "home": home, "key_name": "claimant"}
	r["signed_tx_path"] = filepath.Join(home, "signed.bin")
	r = round(r)
	validateRequest(r, "sign")
	mustFail(t, func() { signExisting(context.Background(), r, options{}) })
	summary := signExisting(context.Background(), r, options{disposable: true})
	actual := readPrivate(context.Background(), str(r, "signed_tx_path"), maxProto)
	if !bytes.Equal(canonical(summary), canonical(signedSummary(plan, actual))) {
		t.Fatal("keyring exact-byte verification mismatch")
	}
	entries, e := os.ReadDir(filepath.Join(home, "keyring-test"))
	if e != nil {
		t.Fatal(e)
	}
	before := len(entries)
	bad := round(r)
	obj(bad["keyring"])["key_name"] = "missing"
	bad["signed_tx_path"] = filepath.Join(home, "missing.bin")
	mustFail(t, func() { signExisting(context.Background(), bad, options{disposable: true}) })
	entries, _ = os.ReadDir(filepath.Join(home, "keyring-test"))
	if len(entries) != before {
		t.Fatal("missing key request mutated keyring")
	}
	for _, backend := range []string{"os", "pass"} {
		bad := round(r)
		obj(bad["keyring"])["backend"] = backend
		mustFail(t, func() { signExisting(context.Background(), bad, options{}) })
	}
}

// An actual typed Comet JSON-RPC endpoint, not made-up REST. Every expected
// protobuf query is decoded and its exact pot/parties/denom are asserted.
type fakeNode struct {
	t            *testing.T
	mu           sync.Mutex
	server       *httptest.Server
	p, pol, plan object
	genesis      *cmttypes.GenesisDoc
	blocks       map[int64]*rpctypes.ResultBlock
	latest       int64
	signed       []byte
	results      []*abci.ExecTxResult
	broadcasts   int
	queries      []string
	fault        string
	queryCalls   int
}

func newFake(t *testing.T) *fakeNode {
	p, pol, plan := fixture(publicTestKey().PubKey().Bytes())
	f := &fakeNode{t: t, p: p, pol: pol, plan: plan, blocks: map[int64]*rpctypes.ResultBlock{}, latest: 11, signed: signedFixture(plan)}
	f.genesis = &cmttypes.GenesisDoc{ChainID: str(p, "chain_reference"), GenesisTime: fixtureNow.Add(-time.Hour), InitialHeight: 1, ConsensusParams: cmttypes.DefaultConsensusParams(), AppState: json.RawMessage(`{}`)}
	f.results = []*abci.ExecTxResult{{Code: 0, GasWanted: 250000, GasUsed: 71331, Events: []abci.Event{{Type: "zerone.claiming_pot.pot_claimed", Attributes: []abci.EventAttribute{{Key: "pot_id", Value: str(pol, "pot_id")}, {Key: "claimant", Value: rawAccount(str(pol, "claimant_account"), str(p, "chain_id"))}, {Key: "amount", Value: "222000"}}}}}}
	for h := int64(9); h <= 11; h++ {
		txs := []cmttypes.Tx{}
		if h == 10 {
			txs = append(txs, f.signed)
		}
		b := cmttypes.MakeBlock(h, txs, &cmttypes.Commit{}, nil)
		b.ChainID = str(p, "chain_reference")
		b.Time = fixtureNow.Add(time.Duration(h-12) * time.Second)
		b.ValidatorsHash = bytes.Repeat([]byte{3}, 32)
		if h > 9 {
			b.LastBlockID = f.blocks[h-1].BlockID
		}
		if h == 11 {
			b.LastResultsHash = cmttypes.NewResults(f.results).Hash()
		}
		f.blocks[h] = &rpctypes.ResultBlock{Block: b, BlockID: cmttypes.BlockID{Hash: b.Hash()}}
	}
	f.server = httptest.NewServer(http.HandlerFunc(f.serve))
	t.Cleanup(f.server.Close)
	return f
}
func (f *fakeNode) node() object {
	return object{"node_trust_id": f.pol["node_trust_id"], "rpc_url": f.server.URL, "grpc_address": "127.0.0.1:9090", "mode": "local"}
}
func (f *fakeNode) req(command string) object {
	r := request(command, f.p, f.pol, nil)
	r["node"] = f.node()
	if command != "inspect" {
		r["plan"] = f.plan
	}
	if command == "inspect" {
		r["height"] = nil
	}
	if command == "simulate" {
		r["height"] = strconv.FormatInt(f.latest, 10)
	}
	return round(r)
}
func (f *fakeNode) client(r object) *nodeClient {
	return newNode(context.Background(), r, func() time.Time { return fixtureNow })
}
func (f *fakeNode) serve(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var req struct {
		ID     int    `json:"id"`
		Method string `json:"method"`
		Params object `json:"params"`
	}
	if json.NewDecoder(r.Body).Decode(&req) != nil {
		f.t.Error("RPC decode")
		w.WriteHeader(400)
		return
	}
	var result any
	var rpcErr *rpcFailure
	switch req.Method {
	case "genesis":
		g := *f.genesis
		if f.fault == "genesis" {
			g.ChainID = "wrong"
		}
		result = &rpctypes.ResultGenesis{Genesis: &g}
	case "status":
		s := &rpctypes.ResultStatus{}
		s.NodeInfo.Network = str(f.p, "chain_reference")
		if f.fault == "chain" {
			s.NodeInfo.Network = "wrong"
		}
		s.SyncInfo.LatestBlockHeight = f.latest
		s.SyncInfo.LatestBlockHash = f.blocks[f.latest].BlockID.Hash
		s.SyncInfo.LatestBlockTime = f.blocks[f.latest].Block.Time
		s.SyncInfo.CatchingUp = f.fault == "catching_up"
		result = s
	case "block":
		h, _ := strconv.ParseInt(req.Params["height"].(string), 10, 64)
		result = f.blocks[h]
	case "num_unconfirmed_txs":
		res := &rpctypes.ResultUnconfirmedTxs{}
		if f.fault == "mempool" {
			res.Count = 1
			res.Total = 1
		}
		result = res
	case "abci_query":
		f.queryCalls++
		path := req.Params["path"].(string)
		f.queries = append(f.queries, path)
		h, _ := strconv.ParseInt(req.Params["height"].(string), 10, 64)
		var b cmtbytes.HexBytes
		e := cmtjson.Unmarshal(canonical(req.Params["data"]), &b)
		if e != nil {
			f.t.Error(e)
		}
		res := f.query(path, b, h)
		if f.fault == "height" {
			res.Height++
		}
		result = &rpctypes.ResultABCIQuery{Response: *res}
	case "broadcast_tx_sync":
		f.broadcasts++
		b, e := base64.StdEncoding.DecodeString(req.Params["tx"].(string))
		if e != nil || !bytes.Equal(b, f.signed) {
			f.t.Error("broadcast bytes differed")
		}
		code := uint32(0)
		if f.fault == "checktx" {
			code = 32
		}
		result = &rpctypes.ResultBroadcastTx{Code: code, Hash: cmttypes.Tx(b).Hash()}
		if f.fault == "lost" {
			w.Write([]byte(`{"malformed":true}`))
			return
		}
	case "tx":
		hash := str(signedSummary(f.plan, f.signed), "tx_hash")
		var hashBytes []byte
		if e := cmtjson.Unmarshal(canonical(req.Params["hash"]), &hashBytes); e != nil || !bytes.Equal(hashBytes, txHashBytes(hash)) {
			f.t.Error("lookup hash differed or failed native JSON codec")
		}
		if f.fault == "absent" {
			rpcErr = &rpcFailure{Code: -32603, Message: "Internal error", Data: fmt.Sprintf("tx (%s) not found", hash)}
		} else {
			result = &rpctypes.ResultTx{Hash: cmttypes.Tx(f.signed).Hash(), Height: 10, Index: 0, Tx: f.signed, TxResult: *f.results[0]}
		}
	case "block_results":
		result = &rpctypes.ResultBlockResults{Height: 10, TxsResults: f.results}
	default:
		f.t.Errorf("unexpected RPC %s", req.Method)
		w.WriteHeader(400)
		return
	}
	envelope := object{"jsonrpc": "2.0", "id": req.ID}
	if rpcErr != nil {
		envelope["error"] = rpcErr
	} else {
		b, e := cmtjson.Marshal(result)
		if e != nil {
			f.t.Error(e)
		}
		envelope["result"] = json.RawMessage(b)
	}
	b, e := json.Marshal(envelope)
	if e != nil {
		f.t.Error(e)
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(b)
}
func (f *fakeNode) query(path string, b []byte, h int64) *abci.ResponseQuery {
	decode := func(v any) {
		if e := unmarshalNative(b, v); e != nil {
			f.t.Error(e)
		}
	}
	a := rawAccount(str(f.pol, "claimant_account"), str(f.p, "chain_id"))
	s := rawAccount(str(f.pol, "sponsor_account"), str(f.p, "chain_id"))
	id := str(f.pol, "pot_id")
	var result any
	switch path {
	case "/zerone.claiming_pot.v1.Query/QueryPot":
		var req pottypes.QueryPotRequest
		decode(&req)
		if req.Id != id {
			f.t.Error("pot query")
		}
		pot := &pottypes.ClaimingPot{Id: id, TotalAmount: "222000", ClaimedAmount: "0", Schedule: &pottypes.VestingSchedule{StartBlock: 1, EndBlock: 2}, Eligibility: &pottypes.EligibilityCriteria{Whitelist: []string{a}}, Status: pottypes.PotStatus_POT_STATUS_ACTIVE}
		if f.fault == "wide_pot" {
			pot.Eligibility.Whitelist = append(pot.Eligibility.Whitelist, s)
		}
		if f.fault == "missing_pot" {
			err := errorsmod.Wrap(sdkerrors.ErrUnknownRequest, fmt.Errorf("%w: %s", pottypes.ErrPotNotFound, id).Error())
			space, code, log := errorsmod.ABCIInfo(err, false)
			return &abci.ResponseQuery{Height: h, Codespace: space, Code: code, Log: log}
		}
		result = &pottypes.QueryPotResponse{Pot: pot}
	case "/zerone.claiming_pot.v1.Query/QueryClaims":
		var req pottypes.QueryClaimsRequest
		decode(&req)
		if req.PotId != id {
			f.t.Error("claims query")
		}
		claims := []*pottypes.Claim{}
		if h >= 10 {
			claims = append(claims, &pottypes.Claim{PotId: id, Claimant: a, Amount: "222000", ClaimedAt: 10})
		}
		if f.fault == "wide_claims" {
			claims = append(claims, claims[0])
		}
		result = &pottypes.QueryClaimsResponse{Claims: claims}
	case "/zerone.claiming_pot.v1.Query/QueryParams":
		var req pottypes.QueryParamsRequest
		decode(&req)
		result = &pottypes.QueryParamsResponse{Params: &pottypes.Params{MinClaimAmount: "1000"}}
	case "/cosmos.auth.v1beta1.Query/Account":
		var req authtypes.QueryAccountRequest
		decode(&req)
		if req.Address != a {
			f.t.Error("account query")
		}
		if f.fault == "missing_account" {
			err := errorsmod.Wrap(sdkerrors.ErrKeyNotFound, status.Errorf(codes.NotFound, "account %s not found", a).Error())
			space, code, log := errorsmod.ABCIInfo(err, false)
			return &abci.ResponseQuery{Height: h, Codespace: space, Code: code, Log: log}
		}
		acc := &authtypes.BaseAccount{Address: a, AccountNumber: 7, Sequence: 0}
		if h >= 10 {
			acc.Sequence = 1
		}
		result = &authtypes.QueryAccountResponse{Account: anyNative("/cosmos.auth.v1beta1.BaseAccount", acc)}
	case "/cosmos.bank.v1beta1.Query/Balance":
		var req banktypes.QueryBalanceRequest
		decode(&req)
		if req.Address != s || req.Denom != "uzrn" {
			f.t.Error("balance query")
		}
		c := coin("1000000")[0]
		result = &banktypes.QueryBalanceResponse{Balance: &c}
	case "/cosmos.feegrant.v1beta1.Query/Allowance":
		var req feegrant.QueryAllowanceRequest
		decode(&req)
		if req.Granter != s || req.Grantee != a {
			f.t.Error("allowance query")
		}
		if f.fault == "missing_allowance" || f.fault == "unknown_allowance" {
			err := errorsmod.Wrap(sdkerrors.ErrUnknownRequest, status.Error(codes.Internal, sdkerrors.ErrNotFound.Wrap("fee-grant not found").Error()).Error())
			space, code, log := errorsmod.ABCIInfo(err, false)
			if f.fault == "unknown_allowance" {
				log = "provider unavailable"
			}
			return &abci.ResponseQuery{Height: h, Codespace: space, Code: code, Log: log}
		}
		exp := timestamp(str(f.pol, "grant_expires_at"))
		allowed := &feegrant.AllowedMsgAllowance{Allowance: anyNative("/cosmos.feegrant.v1beta1.BasicAllowance", &feegrant.BasicAllowance{SpendLimit: coin("600000"), Expiration: &exp}), AllowedMessages: []string{claimURL}}
		result = &feegrant.QueryAllowanceResponse{Allowance: &feegrant.Grant{Granter: s, Grantee: a, Allowance: anyNative("/cosmos.feegrant.v1beta1.AllowedMsgAllowance", allowed)}}
	case "/zerone.vesting_rewards.v1.Query/SupplyCouplingAudit":
		var req vesttypes.QuerySupplyCouplingAuditRequest
		decode(&req)
		result = &vesttypes.QuerySupplyCouplingAuditResponse{TotalMinted: "3000000", CurrentSupply: "2000000", MaxSupply: "222222222000000"}
	case "/cosmos.tx.v1beta1.Service/Simulate":
		var req txtypes.SimulateRequest
		decode(&req)
		if !bytes.Equal(req.TxBytes, unb64(str(f.plan, "simulation_tx_bytes_b64u"))) {
			f.t.Error("simulation substitution")
		}
		result = &txtypes.SimulateResponse{GasInfo: &sdk.GasInfo{GasWanted: ^uint64(0), GasUsed: 71331}}
	default:
		f.t.Errorf("unsupported query %s", path)
	}
	return &abci.ResponseQuery{Height: h, Value: marshalNative(result)}
}
func TestBoundedTypedInspection(t *testing.T) {
	for _, fault := range []string{"", "chain", "genesis", "catching_up", "height", "wide_pot", "wide_claims", "missing_pot", "missing_account", "missing_allowance", "unknown_allowance"} {
		t.Run(fault, func(t *testing.T) {
			f := newFake(t)
			f.fault = fault
			n := f.client(f.req("inspect"))
			defer n.close()
			res := n.inspect(nil, trustedArtifacts{f.genesis})
			expectedObserved := fault == "" || strings.HasPrefix(fault, "missing_")
			if (str(res, "status") == "observed") != expectedObserved {
				t.Fatalf("unexpected observation %s", canonical(res))
			}
			if fault == "wide_pot" {
				for _, q := range f.queries {
					if strings.HasSuffix(q, "/QueryClaims") {
						t.Fatal("queried unbounded claims before shape check")
					}
				}
			}
			if f.broadcasts != 0 {
				t.Fatal("inspection mutated chain")
			}
		})
	}
}
func TestSimulationAndSingleSubmission(t *testing.T) {
	f := newFake(t)
	r := f.req("simulate")
	n := f.client(r)
	defer n.close()
	res := n.simulate(r, trustedArtifacts{f.genesis})
	if str(res, "gas_used") != "71331" || str(res, "gas_wanted") != "18446744073709551615" {
		t.Fatal("measured simulation gas or native infinite-meter GasWanted lost")
	}
	f.fault = "mempool"
	mustFail(t, func() { n.simulate(r, trustedArtifacts{f.genesis}) })
	f.fault = ""
	r["height"] = "10"
	mustFail(t, func() { n.simulate(r, trustedArtifacts{f.genesis}) })
	for _, fault := range []string{"", "checktx", "lost"} {
		t.Run(fault, func(t *testing.T) {
			f := newFake(t)
			f.fault = fault
			r := f.req("submit")
			path := filepath.Join(privateDir(t), "signed.bin")
			writePrivateExclusive(path, f.signed)
			r["signed_tx_path"] = path
			r["expected_tx_hash"] = signedSummary(f.plan, f.signed)["tx_hash"]
			n := f.client(r)
			defer n.close()
			result := n.submit(r, trustedArtifacts{f.genesis})
			want := "accepted"
			if fault != "" {
				want = "submission_unknown"
			}
			if str(result, "status") != want || f.broadcasts != 1 {
				t.Fatalf("submission %s calls=%d", canonical(result), f.broadcasts)
			}
		})
	}
}
func TestPositiveCanonicalLookupAndUnknownNeverRetries(t *testing.T) {
	for _, fault := range []string{"", "absent", "unknown_allowance"} {
		t.Run(fault, func(t *testing.T) {
			f := newFake(t)
			f.fault = fault
			r := f.req("lookup")
			r["tx_hash"] = signedSummary(f.plan, f.signed)["tx_hash"]
			n := f.client(r)
			defer n.close()
			result := n.lookup(r, trustedArtifacts{f.genesis})
			want := "included"
			if fault == "absent" {
				want = "absent"
			}
			if fault == "unknown_allowance" {
				want = "unknown"
			}
			if str(result, "status") != want {
				t.Fatalf("lookup %s", canonical(result))
			}
			if want == "included" && (str(result, "credited_amount_uzrn") != "222000" || str(result, "claimant_sequence") != "1") {
				t.Fatal("lost native credited evidence")
			}
			if f.broadcasts != 0 {
				t.Fatal("lookup broadcast")
			}
		})
	}
	f := newFake(t)
	r := f.req("lookup")
	r["tx_hash"] = signedSummary(f.plan, f.signed)["tx_hash"]
	f.results[0].Code = 12
	n := f.client(r)
	defer n.close()
	if str(n.lookup(r, trustedArtifacts{f.genesis}), "status") != "unknown" {
		t.Fatal("accepted modified result against committed successor hash")
	}
	// With the failed native result genuinely committed by the successor, this
	// is positive failed inclusion, not unknown and never a credited seed.
	f.blocks[11].Block.LastResultsHash = cmttypes.NewResults(f.results).Hash()
	f.blocks[11].BlockID.Hash = f.blocks[11].Block.Header.Hash()
	failed := n.lookup(r, trustedArtifacts{f.genesis})
	if str(failed, "status") != "included" || failed["code"] != uint32(12) || failed["credited_amount_uzrn"] != nil {
		t.Fatalf("failed inclusion not preserved %s", canonical(failed))
	}
	f.blocks[10].Block.Data.Txs = nil
	if str(n.lookup(r, trustedArtifacts{f.genesis}), "status") != "unknown" {
		t.Fatal("accepted tx list inconsistent with canonical data root")
	}
}
func TestWirePipelineAndHostPins(t *testing.T) {
	f := newFake(t)
	dir := privateDir(t)
	now := time.Now().UTC().Truncate(time.Millisecond)
	// This fixture is an explicit fake node plus local public marker artifacts,
	// not a real chain runtime or a release-provenance claim.
	f.genesis.GenesisTime = now.Add(-time.Hour)
	for h := int64(9); h <= 11; h++ {
		b := f.blocks[h]
		b.Block.Time = now.Add(time.Duration(h-12) * time.Second)
		if h > 9 {
			b.Block.LastBlockID = f.blocks[h-1].BlockID
		}
		b.BlockID.Hash = b.Block.Header.Hash()
		encoded, err := cmtjson.Marshal(b)
		if err != nil {
			t.Fatal(err)
		}
		var decoded rpctypes.ResultBlock
		if err := cmtjson.Unmarshal(encoded, &decoded); err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(decoded.Block.Hash(), b.BlockID.Hash) {
			t.Fatalf("fixture roundtrip block %d hash mismatch, times %s %s", h, b.Block.Time, decoded.Block.Time)
		}
	}
	genesis := sdkGenesisBytes(t, f.genesis)
	source := canonical(object{"kind": "public-test-marker", "zerone_core_commit": f.p["zerone_core_commit"]})
	runtime := []byte("explicit public disposable runtime marker, not a zeroned binary")
	for name, data := range map[string][]byte{"genesis.json": genesis, "source.json": source, "runtime.fixture": runtime} {
		if err := os.WriteFile(filepath.Join(dir, name), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	executable, err = filepath.EvalSymlinks(executable)
	if err != nil {
		t.Fatal(err)
	}
	f.p["helper_sha256"] = hashArtifact(context.Background(), executable)
	f.p["genesis_hash"] = digest(genesis)
	f.p["source_digest"] = digest(source)
	f.p["runtime_sha256"] = digest(runtime)
	setID(f.p, "profile_id")
	f.pol["profile_id"] = f.p["profile_id"]
	f.pol["not_before"] = nowStamp(now.Add(-time.Minute))
	f.pol["expires_at"] = nowStamp(now.Add(time.Hour))
	f.pol["grant_expires_at"] = nowStamp(now.Add(2 * time.Hour))
	setID(f.pol, "policy_hash")
	c := obj(f.plan["commitment"])
	for _, k := range []string{"profile_id", "genesis_hash", "source_digest"} {
		c[k] = f.p[k]
	}
	for _, k := range []string{"policy_hash", "expires_at", "grant_expires_at"} {
		c[k] = f.pol[k]
	}
	f.plan["commitment_hash"] = hashObject(c, "")
	setID(f.plan, "plan_id")
	trust := object{"protocol": "zerone-seed-trust/0.1", "profile_id": f.p["profile_id"], "node": f.node(), "genesis_file": filepath.Join(dir, "genesis.json"), "runtime_file": filepath.Join(dir, "runtime.fixture"), "source_manifest_file": filepath.Join(dir, "source.json"), "disposable_test": true}
	trustPath := filepath.Join(dir, "trust.json")
	if err := os.WriteFile(trustPath, canonical(trust), 0600); err != nil {
		t.Fatal(err)
	}
	kr, err := keyring.New(sdk.KeyringServiceName(), keyring.BackendTest, dir, bytes.NewReader(nil), keyCodec())
	if err != nil {
		t.Fatal(err)
	}
	if err := kr.ImportPrivKeyHex("claimant", hex.EncodeToString(publicTestKey().Key), "secp256k1"); err != nil {
		t.Fatal(err)
	}
	invoke := func(command string, r object) object {
		t.Helper()
		var out bytes.Buffer
		args := []string{command, "--trust-file", trustPath, "--disposable-test"}
		if rc := run(args, bytes.NewReader(canonical(r)), &out); rc != 0 {
			t.Fatalf("wire %s rejected: %s", command, out.String())
		}
		response := parseJSON(out.Bytes())
		if str(response, "status") != "ok" {
			t.Fatalf("wire error %s", out.String())
		}
		return obj(response["result"])
	}
	obs := invoke("inspect", f.req("inspect"))
	if str(obs, "status") != "observed" {
		t.Fatalf("wire inspect %s", canonical(obs))
	}
	invoke("simulate", f.req("simulate"))
	r := request("sign", f.p, f.pol, f.plan)
	r["keyring"] = object{"backend": "test", "home": dir, "key_name": "claimant"}
	r["signed_tx_path"] = filepath.Join(dir, "signed.bin")
	signed := invoke("sign", r)
	if _, ok := signed["signed_tx_b64u"]; ok {
		t.Fatal("production wire leaked signed bytes")
	}
	verify := request("verify", f.p, f.pol, f.plan)
	verify["signed_tx_path"] = r["signed_tx_path"]
	invoke("verify", verify)
	submit := f.req("submit")
	submit["signed_tx_path"] = r["signed_tx_path"]
	submit["expected_tx_hash"] = signed["tx_hash"]
	invoke("submit", submit)
	lookup := f.req("lookup")
	lookup["tx_hash"] = signed["tx_hash"]
	included := invoke("lookup", lookup)
	if str(included, "status") != "included" || str(included, "credited_amount_uzrn") != "222000" || f.broadcasts != 1 {
		t.Fatalf("wire pipeline did not confirm %s", canonical(included))
	}
	// Exact trust artifacts do not repair an unknown runtime marker automatically.
	if err := os.WriteFile(filepath.Join(dir, "runtime.fixture"), []byte("changed"), 0600); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if run([]string{"inspect", "--trust-file", trustPath, "--disposable-test"}, bytes.NewReader(canonical(f.req("inspect"))), &out) == 0 {
		t.Fatal("accepted runtime pin drift")
	}
	if strings.Contains(out.String(), dir) {
		t.Fatal("private trust path leaked")
	}
}

func TestOperatorEncodersAndReadOnlyDefaults(t *testing.T) {
	p, pol, _ := fixture(publicTestKey().PubKey().Bytes())
	base := object{"protocol": protocol, "request_id": "operator-1", "timeout_ms": 1000, "profile": p, "granter": rawAccount(str(pol, "sponsor_account"), str(p, "chain_id")), "grantee": rawAccount(str(pol, "claimant_account"), str(p, "chain_id"))}
	r := round(base)
	r["command"] = "operator-grant"
	r["spend_limit_uzrn"] = "600000"
	r["expires_at"] = "2030-01-02T04:00:00.000Z"
	r["now"] = "2030-01-02T03:04:05.000Z"
	validateRequest(r, "operator-grant")
	res := operatorMessage(r, fixtureNow)
	var grant feegrant.MsgGrantAllowance
	if grant.Unmarshal(unb64(str(res, "value_b64u"))) != nil || grant.Allowance.TypeUrl != "/cosmos.feegrant.v1beta1.AllowedMsgAllowance" {
		t.Fatal("grant native codec")
	}
	r = round(base)
	r["command"] = "operator-revoke"
	validateRequest(r, "operator-revoke")
	res = operatorMessage(r, fixtureNow)
	var revoke feegrant.MsgRevokeAllowance
	if revoke.Unmarshal(unb64(str(res, "value_b64u"))) != nil || revoke.Grantee != base["grantee"] {
		t.Fatal("revoke native codec")
	}
	admit := object{"protocol": protocol, "request_id": "operator-1", "timeout_ms": 1000, "profile": p, "command": "operator-admit", "authority": base["granter"], "address": base["grantee"]}
	admit = round(admit)
	validateRequest(admit, "operator-admit")
	res = operatorMessage(admit, fixtureNow)
	var msg pottypes.MsgAddBootstrapEntry
	if unmarshalNative(unb64(str(res, "value_b64u")), &msg) != nil || len(msg.Addresses) != 1 {
		t.Fatal("admit native codec")
	}
	var out bytes.Buffer
	if run(nil, bytes.NewReader(nil), &out) != 0 || !strings.Contains(out.String(), "No arguments or --help") {
		t.Fatal("default is not help-only")
	}
	out.Reset()
	if run([]string{"inspect"}, bytes.NewReader([]byte(`{"duplicate":1,"duplicate":2}`)), &out) == 0 {
		t.Fatal("malformed input accepted")
	}
	if strings.Contains(out.String(), "duplicate") {
		t.Fatal("request leaked")
	}
}
