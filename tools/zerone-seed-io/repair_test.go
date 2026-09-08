package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	sdklog "cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"
	feegrant "cosmossdk.io/x/feegrant"
	feegrantkeeper "cosmossdk.io/x/feegrant/keeper"
	abci "github.com/cometbft/cometbft/abci/types"
	cmtsecp "github.com/cometbft/cometbft/crypto/secp256k1"
	cmtbytes "github.com/cometbft/cometbft/libs/bytes"
	cmtjson "github.com/cometbft/cometbft/libs/json"
	"github.com/cometbft/cometbft/libs/log"
	rpctypes "github.com/cometbft/cometbft/rpc/core/types"
	rpcserver "github.com/cometbft/cometbft/rpc/jsonrpc/server"
	jsonrpc "github.com/cometbft/cometbft/rpc/jsonrpc/types"
	cmttypes "github.com/cometbft/cometbft/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/codec"
	addresscodec "github.com/cosmos/cosmos-sdk/codec/address"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
	potkeeper "github.com/zerone-chain/zerone/x/claiming_pot/keeper"
	pottypes "github.com/zerone-chain/zerone/x/claiming_pot/types"
)

// Exercise the actual native keepers AND BaseApp's gRPC-to-ABCI error mapping.
// No daemon or copied error mapping: in-memory empty stores supply real absence.
func TestNativeKeeperAbsenceThroughBaseApp(t *testing.T) {
	f := newFake(t)
	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)
	app := baseapp.NewBaseApp("seed-absence-test", sdklog.NewNopLogger(), dbm.NewMemDB(), nil)
	keys := storetypes.NewKVStoreKeys(authtypes.StoreKey, feegrant.StoreKey, pottypes.StoreKey)
	app.MountKVStores(keys)
	app.SetInterfaceRegistry(registry)
	ak := authkeeper.NewAccountKeeper(cdc, runtime.NewKVStoreService(keys[authtypes.StoreKey]), authtypes.ProtoBaseAccount, nil, addresscodec.NewBech32Codec("zrn"), "zrn", "test-authority")
	fk := feegrantkeeper.NewKeeper(cdc, runtime.NewKVStoreService(keys[feegrant.StoreKey]), ak)
	pk := potkeeper.NewKeeper(runtime.NewKVStoreService(keys[pottypes.StoreKey]), cdc, "test-authority", nil, nil, nil, nil)
	pottypes.RegisterQueryServer(app.GRPCQueryRouter(), potkeeper.NewQueryServerImpl(pk))
	authtypes.RegisterQueryServer(app.GRPCQueryRouter(), authkeeper.NewQueryServer(ak))
	feegrant.RegisterQueryServer(app.GRPCQueryRouter(), fk)
	if err := app.LoadLatestVersion(); err != nil {
		t.Fatal(err)
	}
	app.CommitMultiStore().Commit()
	t.Cleanup(func() { app.Close() })
	// Forward only typed query calls to real BaseApp.Query. Other methods are the
	// existing bounded Comet fixture; only absence contracts are under test here.
	for _, surface := range []string{"pot", "account", "allowance"} {
		t.Run(surface, func(t *testing.T) {
			n := f.client(f.req("inspect"))
			defer n.close()
			var response *abci.ResponseQuery
			var mutate func(*abci.ResponseQuery)
			n.client.Transport = repairTransport(func(req *http.Request) (*http.Response, error) {
				var input object
				if err := json.NewDecoder(req.Body).Decode(&input); err != nil {
					return nil, err
				}
				params := obj(input["params"])
				var data []byte
				if _, err := fmt.Sscanf(str(params, "data"), "%x", &data); err != nil {
					return nil, err
				}
				var err error
				response, err = app.Query(req.Context(), &abci.RequestQuery{Path: str(params, "path"), Data: data, Height: 1})
				if err != nil {
					return nil, err
				}
				if mutate != nil {
					mutate(response)
				}
				encoded, err := cmtjson.Marshal(&rpctypes.ResultABCIQuery{Response: *response})
				if err != nil {
					return nil, err
				}
				body := canonical(object{"jsonrpc": "2.0", "id": input["id"], "result": json.RawMessage(encoded)})
				return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader(body))}, nil
			})
			read := func() {
				switch surface {
				case "pot":
					p, c := n.potAndClaim(1)
					if p != nil || c != nil {
						t.Fatal("pot absence lost")
					}
				case "account":
					if n.account(1)["status"] != "absent" {
						t.Fatal("account absence lost")
					}
				case "allowance":
					if n.allowance(1)["status"] != "absent" {
						t.Fatal("allowance absence lost")
					}
				}
			}
			read()
			t.Logf("native %s codespace=%s code=%d log=%s", surface, response.Codespace, response.Code, response.Log)
			for _, field := range []string{"code", "codespace", "log", "route", "unimplemented", "height"} {
				t.Run(field, func(t *testing.T) {
					mutate = func(r *abci.ResponseQuery) {
						switch field {
						case "code":
							r.Code = 18
						case "codespace":
							r.Codespace = "other"
						case "log":
							r.Log += " different requested coordinate"
						case "route":
							r.Code = 6
							r.Log = "unknown query path: unknown request"
						case "unimplemented":
							r.Code = 6
							r.Log = "rpc error: code = Unimplemented desc = method not implemented: unknown request"
						case "height":
							r.Height++
						}
					}
					want := nodeFailure("not_found_unproven")
					if field == "height" {
						want = nodeFailure("incoherent_height")
					}
					if got := mustFail(t, read); got != want {
						t.Fatalf("unexpected absence mutation result %v", got)
					}
				})
			}
		})
	}
}

func sdkGenesisBytes(t *testing.T, g *cmttypes.GenesisDoc) []byte {
	t.Helper()
	// The same encoding used by SDK AppGenesis.SaveAs / the v0.53 CLI.
	ag := &genutiltypes.AppGenesis{
		AppName: "zerone-public-test", AppVersion: "fixture", ChainID: g.ChainID,
		GenesisTime: g.GenesisTime, InitialHeight: g.InitialHeight,
		AppHash: g.AppHash, AppState: g.AppState,
		Consensus: &genutiltypes.ConsensusGenesis{Params: g.ConsensusParams, Validators: g.Validators},
	}
	b, err := json.MarshalIndent(ag, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(b, []byte(`"initial_height": 1`)) || !bytes.Contains(b, []byte(`"consensus": {`)) || !bytes.Contains(b, []byte(`"params": {`)) {
		t.Fatal("fixture is not SDK-native genesis encoding")
	}
	return b
}

func trustFixture(t *testing.T, f *fakeNode, dir string, genesis []byte) (object, options) {
	t.Helper()
	source := canonical(object{"kind": "public-repair-fixture"})
	for name, b := range map[string][]byte{"genesis.json": genesis, "source.json": source, "runtime.fixture": []byte("public marker"), "signed.bin": f.signed} {
		if err := os.WriteFile(filepath.Join(dir, name), b, 0600); err != nil {
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
	f.p["runtime_sha256"] = digest([]byte("public marker"))
	setID(f.p, "profile_id")
	repinPlan(f)
	trust := object{"protocol": "zerone-seed-trust/0.1", "profile_id": f.p["profile_id"], "node": f.node(), "genesis_file": filepath.Join(dir, "genesis.json"), "runtime_file": filepath.Join(dir, "runtime.fixture"), "source_manifest_file": filepath.Join(dir, "source.json"), "disposable_test": true}
	opts := options{trustFile: filepath.Join(dir, "trust.json"), disposable: true}
	if err := os.WriteFile(opts.trustFile, canonical(trust), 0600); err != nil {
		t.Fatal(err)
	}
	return trust, opts
}
func repinPlan(f *fakeNode) {
	f.pol["profile_id"] = f.p["profile_id"]
	setID(f.pol, "policy_hash")
	c := obj(f.plan["commitment"])
	for _, k := range []string{"profile_id", "genesis_hash", "source_digest"} {
		c[k] = f.p[k]
	}
	c["policy_hash"] = f.pol["policy_hash"]
	f.plan["commitment_hash"] = hashObject(c, "")
	setID(f.plan, "plan_id")
}

func TestSDKNativeGenesisTrust(t *testing.T) {
	f := newFake(t)
	genesis := sdkGenesisBytes(t, f.genesis)
	if _, err := cmttypes.GenesisDocFromJSON(genesis); err == nil {
		t.Fatal("obsolete Comet-only parser unexpectedly accepted numeric SDK height")
	}
	dir := privateDir(t)
	trust, opts := trustFixture(t, f, dir, genesis)
	r := f.req("inspect")
	got := loadTrust(context.Background(), r, opts)
	if !sameGenesis(got.genesis, f.genesis) {
		t.Fatal("SDK conversion changed semantic genesis trust")
	}
	// Pins bind exact original bytes, even semantically equivalent whitespace.
	if err := os.WriteFile(str(trust, "genesis_file"), append(genesis, '\n'), 0600); err != nil {
		t.Fatal(err)
	}
	if got := mustFail(t, func() { loadTrust(context.Background(), r, opts) }); got != failure("profile_mismatch") {
		t.Fatalf("unexpected pin failure %v", got)
	}
	// Remote genesis comparison must still bind consensus, app state/hash, chain,
	// time, height and validators, not merely successful SDK decoding.
	for _, field := range []string{"chain", "time", "height", "params", "state", "app_hash", "validators"} {
		t.Run(field, func(t *testing.T) {
			b, err := cmtjson.Marshal(f.genesis)
			if err != nil {
				t.Fatal(err)
			}
			var other cmttypes.GenesisDoc
			if err := cmtjson.Unmarshal(b, &other); err != nil {
				t.Fatal(err)
			}
			switch field {
			case "chain":
				other.ChainID = "seed-local-other"
			case "time":
				other.GenesisTime = other.GenesisTime.Add(time.Second)
			case "height":
				other.InitialHeight++
			case "params":
				other.ConsensusParams.Block.MaxGas++
			case "state":
				other.AppState = json.RawMessage(`{"changed":true}`)
			case "app_hash":
				other.AppHash = []byte{1}
			case "validators":
				other.Validators = []cmttypes.GenesisValidator{{PubKey: cmtsecp.PubKey(publicTestKey().PubKey().Bytes()), Power: 1}}
			}
			if sameGenesis(got.genesis, &other) {
				t.Fatal("accepted changed remote genesis")
			}
		})
	}
}

func TestSDKGenesisValidationIsNotBypassed(t *testing.T) {
	for _, field := range []string{"height", "consensus", "params", "chain", "validator"} {
		t.Run(field, func(t *testing.T) {
			f := newFake(t)
			var ag genutiltypes.AppGenesis
			if err := json.Unmarshal(sdkGenesisBytes(t, f.genesis), &ag); err != nil {
				t.Fatal(err)
			}
			switch field {
			case "height":
				ag.InitialHeight = -1
			case "consensus":
				ag.Consensus = nil
			case "params":
				ag.Consensus.Params.Block.MaxBytes = 0
			case "chain":
				ag.ChainID = "seed-local-other"
			case "validator":
				ag.Consensus.Validators = []cmttypes.GenesisValidator{{PubKey: cmtsecp.PubKey(publicTestKey().PubKey().Bytes()), Power: 0}}
			}
			genesis, err := json.Marshal(ag)
			if err != nil {
				t.Fatal(err)
			}
			_, opts := trustFixture(t, f, privateDir(t), genesis)
			if got := mustFail(t, func() { loadTrust(context.Background(), f.req("inspect"), opts) }); got != failure("profile_mismatch") {
				t.Fatalf("invalid SDK genesis error %v", got)
			}
		})
	}
}

func TestPinnedCometJSONArgumentCodec(t *testing.T) {
	f := newFake(t)
	f.server.Close()
	mux := http.NewServeMux()
	queryCalls, txCalls, broadcasts := 0, 0, 0
	// Signatures match Comet v0.38.25 Environment.ABCIQuery/Tx/BroadcastTxSync.
	// RegisterRPCFuncs runs the real jsonParamsToArgs codec before these handlers.
	routes := map[string]*rpcserver.RPCFunc{
		"abci_query": rpcserver.NewRPCFunc(func(_ *jsonrpc.Context, path string, data cmtbytes.HexBytes, height int64, prove bool) (*rpctypes.ResultABCIQuery, error) {
			f.mu.Lock()
			defer f.mu.Unlock()
			queryCalls++
			if prove || (height != 10 && height != 11) {
				t.Error("query height/proof changed")
			}
			return &rpctypes.ResultABCIQuery{Response: *f.query(path, data, height)}, nil
		}, "path,data,height,prove"),
		"tx": rpcserver.NewRPCFunc(func(_ *jsonrpc.Context, hash []byte, prove bool) (*rpctypes.ResultTx, error) {
			f.mu.Lock()
			defer f.mu.Unlock()
			txCalls++
			if prove || !bytes.Equal(hash, cmttypes.Tx(f.signed).Hash()) {
				t.Error("tx lookup changed digest bytes")
			}
			return &rpctypes.ResultTx{Hash: hash, Height: 10, Index: 0, Tx: f.signed, TxResult: *f.results[0]}, nil
		}, "hash,prove"),
		"broadcast_tx_sync": rpcserver.NewRPCFunc(func(_ *jsonrpc.Context, tx cmttypes.Tx) (*rpctypes.ResultBroadcastTx, error) {
			f.mu.Lock()
			defer f.mu.Unlock()
			broadcasts++
			if !bytes.Equal(tx, f.signed) {
				t.Error("broadcast changed TxRaw bytes")
			}
			return &rpctypes.ResultBroadcastTx{Hash: tx.Hash()}, nil
		}, "tx"),
	}
	rpcserver.RegisterRPCFuncs(mux, routes, log.NewNopLogger())
	f.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		b, err := io.ReadAll(req.Body)
		if err != nil {
			t.Error(err)
			w.WriteHeader(400)
			return
		}
		req.Body.Close()
		req.Body = io.NopCloser(bytes.NewReader(b))
		var envelope struct {
			Method string `json:"method"`
		}
		if err := json.Unmarshal(b, &envelope); err != nil {
			t.Error(err)
			w.WriteHeader(400)
			return
		}
		if routes[envelope.Method] != nil {
			mux.ServeHTTP(w, req)
		} else {
			f.serve(w, req)
		}
	}))
	t.Cleanup(f.server.Close)
	r := f.req("inspect")
	n := f.client(r)
	defer n.close()
	if got := n.inspect(nil, trustedArtifacts{f.genesis}); str(got, "status") != "observed" {
		t.Fatalf("real argument codec inspection %s", canonical(got))
	}
	if got := n.simulate(f.req("simulate"), trustedArtifacts{f.genesis}); str(got, "gas_wanted") != "18446744073709551615" {
		t.Fatal("native simulation gas altered")
	}
	submit := f.req("submit")
	submit["signed_tx_path"] = filepath.Join(privateDir(t), "signed.bin")
	writePrivateExclusive(str(submit, "signed_tx_path"), f.signed)
	submit["expected_tx_hash"] = signedSummary(f.plan, f.signed)["tx_hash"]
	if got := n.submit(submit, trustedArtifacts{f.genesis}); str(got, "status") != "accepted" {
		t.Fatalf("broadcast %s", canonical(got))
	}
	lookup := f.req("lookup")
	lookup["tx_hash"] = submit["expected_tx_hash"]
	if got := n.lookup(lookup, trustedArtifacts{f.genesis}); str(got, "status") != "included" {
		t.Fatalf("real argument codec lookup %s", canonical(got))
	}
	// The formerly accepted fake formats must be rejected by the native codec,
	// before invoking query/lookup handlers (not coerced by a test-only decoder).
	for _, bad := range []struct {
		method string
		params object
	}{
		{"abci_query", object{"path": "test", "data": "0x0102", "height": "11", "prove": false}},
		{"tx", object{"hash": "0x" + str(lookup, "tx_hash"), "prove": false}},
	} {
		var out rpctypes.ResultABCIQuery
		if err := n.rpc(bad.method, bad.params, &out); err == nil || err.Code != -32602 {
			t.Fatal("native codec accepted obsolete format")
		}
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if queryCalls == 0 || txCalls != 1 || broadcasts != 1 {
		t.Fatalf("unexpected codec call counts %d/%d/%d", queryCalls, txCalls, broadcasts)
	}
}

type repairTransport func(*http.Request) (*http.Response, error)

func (f repairTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func changeStatusAt(n *nodeClient, at int, field string) {
	original := n.client.Transport
	calls := 0
	n.client.Transport = repairTransport(func(req *http.Request) (*http.Response, error) {
		body, err := req.GetBody()
		if err != nil {
			return nil, err
		}
		var input object
		err = json.NewDecoder(body).Decode(&input)
		body.Close()
		if err != nil {
			return nil, err
		}
		res, err := original.RoundTrip(req)
		if err != nil || input["method"] != "status" {
			return res, err
		}
		calls++
		if calls != at {
			return res, nil
		}
		b, err := io.ReadAll(res.Body)
		res.Body.Close()
		if err != nil {
			return nil, err
		}
		var envelope object
		if err := json.Unmarshal(b, &envelope); err != nil {
			return nil, err
		}
		result := obj(envelope["result"])
		if field == "hash" {
			obj(result["sync_info"])["latest_block_hash"] = strings.Repeat("2A", 32)
		} else {
			obj(result["node_info"])["network"] = "seed-local-other"
		}
		b = canonical(envelope)
		res.Body = io.NopCloser(bytes.NewReader(b))
		res.ContentLength = int64(len(b))
		return res, nil
	})
}
func TestEqualHeightFinalStatusContradictionIsUnknown(t *testing.T) {
	for _, command := range []string{"inspect", "simulate", "lookup"} {
		for _, field := range []string{"hash", "chain"} {
			t.Run(command+"/"+field, func(t *testing.T) {
				f := newFake(t)
				r := f.req(command)
				if command == "lookup" {
					r["tx_hash"] = signedSummary(f.plan, f.signed)["tx_hash"]
				}
				n := f.client(r)
				defer n.close()
				at := 2
				if command == "simulate" {
					at = 3
				}
				changeStatusAt(n, at, field)
				if command == "simulate" {
					got := mustFail(t, func() { n.simulate(r, trustedArtifacts{f.genesis}) })
					want := nodeFailure("incoherent_height")
					if field == "chain" {
						want = nodeFailure("chain_mismatch")
					}
					if got != want {
						t.Fatalf("simulation contradiction %v", got)
					}
				} else {
					var got object
					if command == "inspect" {
						got = n.inspect(nil, trustedArtifacts{f.genesis})
					} else {
						got = n.lookup(r, trustedArtifacts{f.genesis})
					}
					if str(got, "status") != "unknown" || got["evidence"] != nil {
						t.Fatalf("contradiction retained positive evidence %s", canonical(got))
					}
				}
			})
		}
	}
}
func TestSimulationPrequeryHashContradiction(t *testing.T) {
	f := newFake(t)
	r := f.req("simulate")
	n := f.client(r)
	defer n.close()
	changeStatusAt(n, 2, "hash")
	if got := mustFail(t, func() { n.simulate(r, trustedArtifacts{f.genesis}) }); got != nodeFailure("incoherent_height") {
		t.Fatalf("unexpected error %v", got)
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.queryCalls != 0 {
		t.Fatal("simulated after contradictory status")
	}
}

type cancelingReader struct {
	cancel         context.CancelFunc
	calls, largest int
}

func (r *cancelingReader) Read(p []byte) (int, error) {
	r.calls++
	r.largest = max(r.largest, len(p))
	clear(p)
	r.cancel()
	return len(p), nil
}
func TestContextReadsStopBetweenBoundedChunks(t *testing.T) {
	for _, copyToHash := range []bool{false, true} {
		ctx, cancel := context.WithCancel(context.Background())
		r := &cancelingReader{cancel: cancel}
		bounded := contextReader{ctx, r}
		var err error
		if copyToHash {
			_, err = io.CopyBuffer(io.Discard, bounded, make([]byte, 1<<20))
		} else {
			_, err = io.ReadAll(bounded)
		}
		if err != context.Canceled || r.calls != 1 || r.largest > fileReadChunk {
			t.Fatalf("cancellation/chunk failure %v calls=%d chunk=%d", err, r.calls, r.largest)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	// Cancellation is checked before opening even an invalid path.
	for _, read := range []func(){func() { hashArtifact(ctx, "/not-opened") }, func() { readPrivate(ctx, "/not-opened", maxProto) }, func() { readPublic(ctx, "/not-opened", maxJSON) }} {
		if got := mustFail(t, read); got != failure("limit_exceeded") {
			t.Fatalf("late cancellation %v", got)
		}
	}
}

type lateTimerContext struct{ context.Context }

func (lateTimerContext) Deadline() (time.Time, bool) { return time.Unix(1, 0), true }

func TestBudgetDoesNotWaitForTimerDelivery(t *testing.T) {
	ctx := lateTimerContext{context.Background()}
	if ctx.Err() != nil || budgetError(ctx) != context.DeadlineExceeded {
		t.Fatal("past deadline did not independently exhaust budget")
	}
	if got := mustFail(t, func() { hashArtifact(ctx, "/not-opened") }); got != failure("limit_exceeded") {
		t.Fatal("hashing waited for timer cancellation")
	}
	buf := make([]byte, 1)
	if n, err := (contextReader{ctx, strings.NewReader("x")}).Read(buf); n != 0 || err != context.DeadlineExceeded {
		t.Fatal("read started past an undelivered deadline")
	}
}

func TestVerifyIncludesTrustHashingInTotalDeadline(t *testing.T) {
	f := newFake(t)
	dir := privateDir(t)
	trust, opts := trustFixture(t, f, dir, sdkGenesisBytes(t, f.genesis))
	// Sparse public bytes, not a daemon or runtime key. Large enough that hashing
	// cannot finish in 10ms; the deterministic chunk test above owns stop mechanics.
	file, err := os.OpenFile(str(trust, "runtime_file"), os.O_WRONLY, 0600)
	if err != nil {
		t.Fatal(err)
	}
	if err := file.Truncate(512 << 20); err != nil {
		file.Close()
		t.Fatal(err)
	}
	file.Close()
	f.p["runtime_sha256"] = hashArtifact(context.Background(), str(trust, "runtime_file"))
	setID(f.p, "profile_id")
	repinPlan(f)
	trust["profile_id"] = f.p["profile_id"]
	if err := os.WriteFile(opts.trustFile, canonical(trust), 0600); err != nil {
		t.Fatal(err)
	}
	r := request("verify", f.p, f.pol, f.plan)
	r["signed_tx_path"] = filepath.Join(dir, "signed.bin")
	r["timeout_ms"] = 10
	var output bytes.Buffer
	start := time.Now()
	rc := run([]string{"verify", "--trust-file", opts.trustFile, "--disposable-test"}, bytes.NewReader(canonical(r)), &output)
	response := parseJSON(output.Bytes())
	t.Logf("10ms total budget elapsed=%s exit=%d status=%s", time.Since(start), rc, str(response, "status"))
	if rc != 1 || response["code"] != "limit_exceeded" {
		t.Fatalf("verify accepted expired budget %s", output.Bytes())
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.queryCalls != 0 || f.broadcasts != 0 {
		t.Fatal("file-only verify touched network")
	}
}
func TestDeadlineResponsePreservesEffectAmbiguity(t *testing.T) {
	for _, command := range []string{"inspect", "simulate", "verify", "lookup", "operator-admit", "sign", "submit"} {
		response := object{"protocol": protocol, "request_id": "test", "command": command, "status": "ok", "result": object{"status": "accepted", "tx_hash": strings.Repeat("AB", 32)}}
		got := deadlineResponse(response)
		if command == "submit" {
			result := obj(got["result"])
			if got["status"] != "ok" || result["status"] != "submission_unknown" || result["tx_hash"] != strings.Repeat("AB", 32) {
				t.Fatal("submission ambiguity lost")
			}
		} else {
			code := "limit_exceeded"
			if command == "sign" {
				code = "signing_unknown"
			}
			if got["status"] != "error" || got["code"] != code || got["result"] != nil {
				t.Fatalf("expired result leaked %s", canonical(got))
			}
		}
	}
}
func TestSubmissionDeadlineAfterTransportIsUnknown(t *testing.T) {
	f := newFake(t)
	r := f.req("submit")
	r["signed_tx_path"] = filepath.Join(privateDir(t), "signed.bin")
	writePrivateExclusive(str(r, "signed_tx_path"), f.signed)
	r["expected_tx_hash"] = signedSummary(f.plan, f.signed)["tx_hash"]
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	n := newNode(ctx, r, func() time.Time { return fixtureNow })
	defer n.close()
	original := n.client.Transport
	n.client.Transport = repairTransport(func(req *http.Request) (*http.Response, error) {
		body, _ := req.GetBody()
		var input object
		err := json.NewDecoder(body).Decode(&input)
		body.Close()
		if err != nil {
			return nil, err
		}
		res, err := original.RoundTrip(req)
		if err != nil || input["method"] != "broadcast_tx_sync" {
			return res, err
		}
		b, err := io.ReadAll(res.Body)
		res.Body.Close()
		if err != nil {
			return nil, err
		}
		res.Body = io.NopCloser(bytes.NewReader(b))
		cancel() // A complete successful response exists, but the command budget ended.
		return res, nil
	})
	got := n.submit(r, trustedArtifacts{f.genesis})
	if got["status"] != "submission_unknown" || got["tx_hash"] != r["expected_tx_hash"] {
		t.Fatalf("expired submission %s", canonical(got))
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.broadcasts != 1 {
		t.Fatal("submission retried or never reached boundary")
	}
}

type cancelOnFileContext struct {
	context.Context
	cancel context.CancelFunc
	path   string
}

func (c cancelOnFileContext) Err() error {
	if _, err := os.Stat(c.path); err == nil {
		c.cancel()
	}
	return c.Context.Err()
}
func TestSigningDeadlineAfterPersistenceStaysUnknown(t *testing.T) {
	home := privateDir(t)
	kr, err := keyring.New(sdk.KeyringServiceName(), keyring.BackendTest, home, bytes.NewReader(nil), keyCodec())
	if err != nil {
		t.Fatal(err)
	}
	if err := kr.ImportPrivKeyHex("claimant", fmt.Sprintf("%x", publicTestKey().Key), "secp256k1"); err != nil {
		t.Fatal(err)
	}
	p, pol, plan := fixture(publicTestKey().PubKey().Bytes())
	r := request("sign", p, pol, plan)
	r["signed_tx_path"] = filepath.Join(home, "signed.bin")
	r["keyring"] = object{"backend": "test", "home": home, "key_name": "claimant"}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	// Deterministic cancellation exactly after exclusive persistence, without a
	// mutable production clock/hook or a scheduling-sensitive watcher goroutine.
	watched := cancelOnFileContext{ctx, cancel, str(r, "signed_tx_path")}
	if got := mustFail(t, func() { signExisting(watched, r, options{disposable: true}) }); got != failure("signing_unknown") {
		t.Fatalf("lost possible signature %v", got)
	}
	b := readPrivate(context.Background(), str(r, "signed_tx_path"), maxProto)
	signedSummary(plan, b)
	if len(b) == 0 {
		t.Fatal("signed evidence deleted on timeout")
	}
	if got := mustFail(t, func() { signExisting(context.Background(), r, options{disposable: true}) }); got != failure("already_exists") {
		t.Fatal("timeout reopened exclusive output")
	}
}
