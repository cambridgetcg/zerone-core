package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestCapturePinsHeightAndBalancesSupply(t *testing.T) {
	t.Helper()
	const (
		checkpointHeight  = "41"
		anchorHeight      = "42"
		triggerHeight     = "43"
		previousHash      = "0011223300112233001122330011223300112233001122330011223300112233"
		anchorHash        = "1122334411223344112233441122334411223344112233441122334411223344"
		triggerHash       = "2233445522334455223344552233445522334455223344552233445522334455"
		checkpointAppHash = "aabbccddaabbccddaabbccddaabbccddaabbccddaabbccddaabbccddaabbccdd"
		postAnchorAppHash = "ffeeddccffeeddccffeeddccffeeddccffeeddccffeeddccffeeddccffeeddcc"
	)
	responses := map[string]string{
		"http://rpc/status": `{
          "result": {
            "node_info": {"network":"zerone-1"},
            "sync_info": {
		              "latest_block_height":"` + triggerHeight + `",
		              "latest_block_hash":"` + triggerHash + `",
		              "latest_app_hash":"` + postAnchorAppHash + `",
              "catching_up":false
            }
          }
        }`,
		"http://rpc/block?height=" + anchorHeight: `{
		          "result": {
		            "block_id":{"hash":"` + anchorHash + `"},
		            "block":{"header":{
		              "chain_id":"zerone-1",
		              "height":"` + anchorHeight + `",
		              "time":"2026-07-12T13:00:00+01:00",
		              "app_hash":"` + checkpointAppHash + `",
		              "last_block_id":{"hash":"` + previousHash + `"}
		            },"data":{"txs":null}}
		          }
		        }`,
		"http://rpc/block?height=" + triggerHeight: `{
		          "result": {
		            "block_id":{"hash":"` + triggerHash + `"},
		            "block":{"header":{
		              "chain_id":"zerone-1",
		              "height":"` + triggerHeight + `",
		              "time":"2026-07-12T12:00:01Z",
		              "app_hash":"` + postAnchorAppHash + `",
		              "last_block_id":{"hash":"` + anchorHash + `"}
		            },"data":{"txs":[]}}
		          }
		        }`,
		"http://rpc/commit?height=" + anchorHeight: `{
		  "result":{"canonical":true,"signed_header":{
		    "header":{"chain_id":"zerone-1","height":"` + anchorHeight + `","app_hash":"` + checkpointAppHash + `"},
		    "commit":{"height":"` + anchorHeight + `","block_id":{"hash":"` + anchorHash + `"},"signatures":[{}]}
		  }}
		}`,
		"http://rpc/commit?height=" + triggerHeight: `{
		  "result":{"canonical":false,"signed_header":{
		    "header":{"chain_id":"zerone-1","height":"` + triggerHeight + `","app_hash":"` + postAnchorAppHash + `"},
		    "commit":{"height":"` + triggerHeight + `","block_id":{"hash":"` + triggerHash + `"},"signatures":[{}]}
		  }}
		}`,
		"http://rpc/block_results?height=" + anchorHeight:  `{"result":{"height":"` + anchorHeight + `"}}`,
		"http://rpc/block_results?height=" + triggerHeight: `{"error":{"message":"Internal error","data":"could not find results for height #` + triggerHeight + `"}}`,
		"http://rpc/abci_info": `{
          "result":{"response":{
		            "last_block_height":"` + anchorHeight + `",
		            "last_block_app_hash":"` + postAnchorAppHash + `"
          }}
        }`,
		"http://rpc/genesis": `{
          "result":{"genesis":{"chain_id":"zerone-1","app_state":{"bank":{}}}}
        }`,
		"http://rest/cosmos/auth/v1beta1/accounts?pagination.limit=500": `{
          "accounts":[
            {"@type":"/cosmos.auth.v1beta1.BaseAccount","address":"zrn1user"},
            {"@type":"/cosmos.auth.v1beta1.ModuleAccount","base_account":{"address":"zrn1module"},"name":"research_fund"}
          ],
          "pagination":{"next_key":null}
        }`,
		"http://rest/cosmos/bank/v1beta1/denom_owners/uzrn?pagination.limit=500": `{
          "denom_owners":[
            {"address":"zrn1user","balance":{"denom":"uzrn","amount":"60"}},
            {"address":"zrn1module","balance":{"denom":"uzrn","amount":"40"}}
          ],
          "pagination":{"next_key":null}
        }`,
		"http://rest/cosmos/bank/v1beta1/supply/by_denom?denom=uzrn": `{
          "amount":{"denom":"uzrn","amount":"100"}
        }`,
		"http://rest/cosmos/staking/v1beta1/validators?status=BOND_STATUS_BONDED&pagination.limit=500": `{
          "validators":[{
            "operator_address":"zrnvaloper1validator",
            "consensus_pubkey":{"@type":"/cosmos.crypto.ed25519.PubKey","key":"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="},
            "jailed":false,
            "status":"BOND_STATUS_BONDED",
            "tokens":"50"
          }],
          "pagination":{"next_key":null}
        }`,
	}

	requestCounts := make(map[string]int)
	client := &http.Client{
		Timeout: time.Second,
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			requestCounts[req.URL.String()]++
			body, ok := responses[req.URL.String()]
			if !ok {
				t.Fatalf("unexpected request: %s", req.URL)
			}
			if strings.HasPrefix(req.URL.String(), "http://rest/") {
				if got := req.Header.Get("x-cosmos-block-height"); got != checkpointHeight {
					t.Fatalf("REST request height header = %q, want %q", got, checkpointHeight)
				}
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"X-Cosmos-Block-Height": []string{checkpointHeight}},
				Body:       io.NopCloser(strings.NewReader(body)),
				Request:    req,
			}, nil
		}),
	}
	a := &api{
		client: client, rpc: "http://rpc", rest: "http://rest",
		checkpointStateHeight: 41, finalCommittedBlockHeight: 42, haltTriggerHeight: 43,
	}

	got, err := a.capture(context.Background(), "zerone-1", strings.Repeat("a", 64), "uzrn")
	if err != nil {
		t.Fatalf("capture: %v", err)
	}
	if got.Schema != schema || got.Source.CheckpointStateHeight != 41 ||
		got.Source.FinalCommittedBlockHeight != 42 || got.Source.HaltTriggerHeight != 43 ||
		got.Supply != "100" {
		t.Fatalf("unexpected snapshot header: %+v", got)
	}
	if len(got.Owners) != 2 || got.Owners[0].Address != "zrn1module" {
		t.Fatalf("owners were not deterministically sorted: %+v", got.Owners)
	}
	if got.Owners[0].ModuleName != "research_fund" {
		t.Fatalf("module classification lost: %+v", got.Owners[0])
	}
	if len(got.Validators) != 1 || got.Validators[0].Tokens != "50" {
		t.Fatalf("validator snapshot lost: %+v", got.Validators)
	}
	if got.Source.RPCGenesisCanonicalSHA256 == "" {
		t.Fatal("canonical genesis hash missing")
	}
	if got.Source.CheckpointAppHash != strings.ToUpper(checkpointAppHash) ||
		got.Source.ExcludedPostAnchorAppHash != strings.ToUpper(postAnchorAppHash) {
		t.Fatalf("app hash boundary is wrong: %+v", got.Source)
	}
	if got.Source.FinalCommittedBlockTime != "2026-07-12T12:00:00Z" {
		t.Fatalf("block time was not normalized: %q", got.Source.FinalCommittedBlockTime)
	}
	if got.Source.FinalCommittedBlockTxs != 0 || got.Source.StagedHaltTriggerBlockTxs != 0 ||
		!got.Source.FinalCommittedBlockCanonical || got.Source.FinalCommittedBlockHasResults != true ||
		got.Source.StagedHaltTriggerCommitCanonical || got.Source.StagedHaltTriggerHasBlockResults ||
		got.Source.RPCBlockstoreHeight != 43 || got.Source.RESTTrustModel != restTrustModel {
		t.Fatalf("anchor/trust metadata is wrong: %+v", got.Source)
	}
	for _, endpoint := range []string{
		"http://rpc/status", "http://rpc/block?height=42", "http://rpc/block?height=43",
		"http://rpc/commit?height=42", "http://rpc/commit?height=43",
		"http://rpc/block_results?height=42", "http://rpc/block_results?height=43",
		"http://rpc/abci_info",
	} {
		if requestCounts[endpoint] != 2 {
			t.Fatalf("%s requests = %d, want initial + final recheck", endpoint, requestCounts[endpoint])
		}
	}
}

func TestValidateHeightPlan(t *testing.T) {
	tests := []struct {
		name       string
		checkpoint int64
		anchor     int64
		halt       int64
		wantErr    string
	}{
		{name: "valid", checkpoint: 41, anchor: 42, halt: 43},
		{name: "missing", checkpoint: 0, anchor: 0, halt: 0, wantErr: "must all be positive"},
		{name: "anchor gap", checkpoint: 41, anchor: 43, halt: 44, wantErr: "must equal checkpoint state"},
		{name: "halt gap", checkpoint: 41, anchor: 42, halt: 44, wantErr: "must equal final committed block"},
		{name: "overflow", checkpoint: 9223372036854775806, anchor: 9223372036854775807, halt: 1, wantErr: "too large"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validateHeightPlan(tc.checkpoint, tc.anchor, tc.halt)
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("validateHeightPlan: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("error = %v, want substring %q", err, tc.wantErr)
			}
		})
	}
}

func TestNormalizeAppHashAcceptsABCIBase64(t *testing.T) {
	const (
		hexHash    = "D2467D4C7CDC5C12C8F8C55B2CDAC61F3A443920D5435FBED25FFEB1AB8E30D6"
		base64Hash = "0kZ9THzcXBLI+MVbLNrGHzpEOSDVQ1++0l/+sauOMNY="
	)
	got, err := normalizeAppHash(base64Hash)
	if err != nil || got != hexHash {
		t.Fatalf("normalizeAppHash = %q, %v; want %q", got, err, hexHash)
	}
	if _, err := normalizeAppHash(base64Hash + "="); err == nil {
		t.Fatal("accepted noncanonical base64 app hash")
	}
}

func TestCaptureRejectsNonEmptyAnchorBlock(t *testing.T) {
	const (
		priorHash   = "0011223300112233001122330011223300112233001122330011223300112233"
		anchorHash  = "1122334411223344112233441122334411223344112233441122334411223344"
		triggerHash = "2233445522334455223344552233445522334455223344552233445522334455"
		appHash     = "aabbccddaabbccddaabbccddaabbccddaabbccddaabbccddaabbccddaabbccdd"
	)
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		var body string
		switch req.URL.String() {
		case "http://rpc/status":
			body = `{"result":{"node_info":{"network":"zerone-1"},"sync_info":{"latest_block_height":"43","latest_block_hash":"` + triggerHash + `","latest_app_hash":"` + appHash + `","catching_up":false}}}`
		case "http://rpc/block?height=42":
			body = `{"result":{"block_id":{"hash":"` + anchorHash + `"},"block":{"header":{"chain_id":"zerone-1","height":"42","time":"2026-07-12T12:00:00Z","app_hash":"` + appHash + `","last_block_id":{"hash":"` + priorHash + `"}},"data":{"txs":["AQ=="]}}}}`
		case "http://rpc/block?height=43":
			body = `{"result":{"block_id":{"hash":"` + triggerHash + `"},"block":{"header":{"chain_id":"zerone-1","height":"43","time":"2026-07-12T12:00:01Z","app_hash":"` + appHash + `","last_block_id":{"hash":"` + anchorHash + `"}},"data":{"txs":null}}}}`
		default:
			t.Fatalf("unexpected request: %s", req.URL)
		}
		return &http.Response{StatusCode: http.StatusOK, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(body)), Request: req}, nil
	})}
	a := &api{
		client: client, rpc: "http://rpc", rest: "http://rest",
		checkpointStateHeight: 41, finalCommittedBlockHeight: 42, haltTriggerHeight: 43,
	}
	if _, err := a.capture(context.Background(), "zerone-1", "", "uzrn"); err == nil || !strings.Contains(err.Error(), "contains 1 transaction") {
		t.Fatalf("expected non-empty anchor rejection, got %v", err)
	}
}

func TestCaptureHaltBoundaryRejectsInvalidSplit(t *testing.T) {
	const (
		priorHash   = "0011223300112233001122330011223300112233001122330011223300112233"
		anchorHash  = "1122334411223344112233441122334411223344112233441122334411223344"
		triggerHash = "2233445522334455223344552233445522334455223344552233445522334455"
		checkpoint  = "aabbccddaabbccddaabbccddaabbccddaabbccddaabbccddaabbccddaabbccdd"
		postAnchor  = "ffeeddccffeeddccffeeddccffeeddccffeeddccffeeddccffeeddccffeeddcc"
	)
	base := func() map[string]string {
		return map[string]string{
			"http://rpc/status":                  `{"result":{"node_info":{"network":"zerone-1"},"sync_info":{"latest_block_height":"43","latest_block_hash":"` + triggerHash + `","latest_app_hash":"` + postAnchor + `","catching_up":false}}}`,
			"http://rpc/block?height=42":         `{"result":{"block_id":{"hash":"` + anchorHash + `"},"block":{"header":{"chain_id":"zerone-1","height":"42","time":"2026-07-12T12:00:00Z","app_hash":"` + checkpoint + `","last_block_id":{"hash":"` + priorHash + `"}},"data":{"txs":null}}}}`,
			"http://rpc/block?height=43":         `{"result":{"block_id":{"hash":"` + triggerHash + `"},"block":{"header":{"chain_id":"zerone-1","height":"43","time":"2026-07-12T12:00:01Z","app_hash":"` + postAnchor + `","last_block_id":{"hash":"` + anchorHash + `"}},"data":{"txs":null}}}}`,
			"http://rpc/commit?height=42":        `{"result":{"canonical":true,"signed_header":{"header":{"chain_id":"zerone-1","height":"42","app_hash":"` + checkpoint + `"},"commit":{"height":"42","block_id":{"hash":"` + anchorHash + `"},"signatures":[{}]}}}}`,
			"http://rpc/commit?height=43":        `{"result":{"canonical":false,"signed_header":{"header":{"chain_id":"zerone-1","height":"43","app_hash":"` + postAnchor + `"},"commit":{"height":"43","block_id":{"hash":"` + triggerHash + `"},"signatures":[{}]}}}}`,
			"http://rpc/abci_info":               `{"result":{"response":{"last_block_height":"42","last_block_app_hash":"` + postAnchor + `"}}}`,
			"http://rpc/block_results?height=42": `{"result":{"height":"42"}}`,
			"http://rpc/block_results?height=43": `{"error":{"message":"Internal error","data":"could not find results for height #43"}}`,
		}
	}
	tests := []struct {
		name    string
		mutate  func(map[string]string)
		wantErr string
	}{
		{name: "wrong blockstore tip", wantErr: "expected staged halt trigger", mutate: func(r map[string]string) {
			r["http://rpc/status"] = strings.Replace(r["http://rpc/status"], `"latest_block_height":"43"`, `"latest_block_height":"42"`, 1)
		}},
		{name: "trigger link mismatch", wantErr: "links", mutate: func(r map[string]string) {
			r["http://rpc/block?height=43"] = strings.Replace(r["http://rpc/block?height=43"], anchorHash, priorHash, 1)
		}},
		{name: "anchor noncanonical", wantErr: "expected canonical commit", mutate: func(r map[string]string) {
			r["http://rpc/commit?height=42"] = strings.Replace(r["http://rpc/commit?height=42"], `"canonical":true`, `"canonical":false`, 1)
		}},
		{name: "trigger reports canonical true", wantErr: "canonical=false subjective", mutate: func(r map[string]string) {
			r["http://rpc/commit?height=43"] = strings.Replace(r["http://rpc/commit?height=43"], `"canonical":false`, `"canonical":true`, 1)
		}},
		{name: "trigger has results", wantErr: "unexpectedly has application block results", mutate: func(r map[string]string) {
			r["http://rpc/block_results?height=43"] = `{"result":{"height":"43"}}`
		}},
		{name: "trigger nonempty", wantErr: "contains 1 transaction", mutate: func(r map[string]string) {
			r["http://rpc/block?height=43"] = strings.Replace(r["http://rpc/block?height=43"], `"txs":null`, `"txs":["AQ=="]`, 1)
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			responses := base()
			tc.mutate(responses)
			client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				body, ok := responses[req.URL.String()]
				if !ok {
					t.Fatalf("unexpected request: %s", req.URL)
				}
				return &http.Response{StatusCode: http.StatusOK, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(body)), Request: req}, nil
			})}
			a := &api{client: client, rpc: "http://rpc", finalCommittedBlockHeight: 42, haltTriggerHeight: 43}
			_, err := a.captureHaltBoundary(context.Background(), "zerone-1")
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("error = %v, want substring %q", err, tc.wantErr)
			}
		})
	}
}

func TestVerifySourceStableRejectsAdvancingNode(t *testing.T) {
	const (
		priorHash   = "0011223300112233001122330011223300112233001122330011223300112233"
		anchorHash  = "1122334411223344112233441122334411223344112233441122334411223344"
		triggerHash = "2233445522334455223344552233445522334455223344552233445522334455"
		checkpoint  = "aabbccddaabbccddaabbccddaabbccddaabbccddaabbccddaabbccddaabbccdd"
		postAnchor  = "ffeeddccffeeddccffeeddccffeeddccffeeddccffeeddccffeeddccffeeddcc"
	)
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		var body string
		switch req.URL.String() {
		case "http://rpc/status":
			body = `{"result":{"node_info":{"network":"zerone-1"},"sync_info":{"latest_block_height":"43","latest_block_hash":"` + triggerHash + `","latest_app_hash":"` + postAnchor + `","catching_up":false}}}`
		case "http://rpc/block?height=42":
			body = `{"result":{"block_id":{"hash":"` + anchorHash + `"},"block":{"header":{"chain_id":"zerone-1","height":"42","time":"2026-07-12T12:00:00Z","app_hash":"` + checkpoint + `","last_block_id":{"hash":"` + priorHash + `"}},"data":{"txs":null}}}}`
		case "http://rpc/block?height=43":
			body = `{"result":{"block_id":{"hash":"` + triggerHash + `"},"block":{"header":{"chain_id":"zerone-1","height":"43","time":"2026-07-12T12:00:01Z","app_hash":"` + postAnchor + `","last_block_id":{"hash":"` + anchorHash + `"}},"data":{"txs":null}}}}`
		case "http://rpc/commit?height=42":
			body = `{"result":{"canonical":true,"signed_header":{"header":{"chain_id":"zerone-1","height":"42","app_hash":"` + checkpoint + `"},"commit":{"height":"42","block_id":{"hash":"` + anchorHash + `"},"signatures":[{}]}}}}`
		case "http://rpc/commit?height=43":
			body = `{"result":{"canonical":false,"signed_header":{"header":{"chain_id":"zerone-1","height":"43","app_hash":"` + postAnchor + `"},"commit":{"height":"43","block_id":{"hash":"` + triggerHash + `"},"signatures":[{}]}}}}`
		case "http://rpc/abci_info":
			body = `{"result":{"response":{"last_block_height":"42","last_block_app_hash":"` + postAnchor + `"}}}`
		case "http://rpc/block_results?height=42":
			body = `{"result":{"height":"42"}}`
		case "http://rpc/block_results?height=43":
			body = `{"error":{"message":"Internal error","data":"could not find results for height #43"}}`
		default:
			t.Fatalf("unexpected request: %s", req.URL)
		}
		return &http.Response{StatusCode: http.StatusOK, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(body)), Request: req}, nil
	})}
	a := &api{client: client, rpc: "http://rpc", finalCommittedBlockHeight: 42, haltTriggerHeight: 43}
	initial := haltBoundary{
		Status: statusResult{
			ChainID: "zerone-1", Height: 43,
			BlockHash: strings.Repeat("A", 64), HeaderAppHash: strings.ToUpper(postAnchor),
		},
	}
	if err := a.verifySourceStable(context.Background(), "zerone-1", initial); err == nil || !strings.Contains(err.Error(), "source changed during capture") {
		t.Fatalf("expected advancing-source rejection, got %v", err)
	}
}

func TestVerifySupplyRejectsMismatch(t *testing.T) {
	err := verifySupply([]owner{
		{Address: "a", Amount: "40"},
		{Address: "b", Amount: "59"},
	}, "100")
	if err == nil || !strings.Contains(err.Error(), "does not equal supply") {
		t.Fatalf("expected supply mismatch, got %v", err)
	}
}

func TestParseAccountSupportsVestingAndModuleAccounts(t *testing.T) {
	tests := []struct {
		name       string
		raw        string
		address    string
		moduleName string
	}{
		{
			name:       "module",
			raw:        `{"@type":"/cosmos.auth.v1beta1.ModuleAccount","base_account":{"address":"zrn1module"},"name":"gov"}`,
			address:    "zrn1module",
			moduleName: "gov",
		},
		{
			name:    "permanent locked",
			raw:     `{"@type":"/cosmos.vesting.v1beta1.PermanentLockedAccount","base_vesting_account":{"base_account":{"address":"zrn1locked"}}}`,
			address: "zrn1locked",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			address, metadata, err := parseAccount(json.RawMessage(tc.raw))
			if err != nil {
				t.Fatalf("parseAccount: %v", err)
			}
			if address != tc.address || metadata.ModuleName != tc.moduleName {
				t.Fatalf("got address=%q module=%q", address, metadata.ModuleName)
			}
		})
	}
}

func TestGetJSONRequiresExactHeightResponseHeader(t *testing.T) {
	tests := []struct {
		name    string
		header  http.Header
		wantErr string
	}{
		{
			name:   "direct SDK header",
			header: http.Header{"X-Cosmos-Block-Height": []string{"42"}},
		},
		{
			name:   "gRPC gateway metadata header",
			header: http.Header{"Grpc-Metadata-X-Cosmos-Block-Height": []string{"42"}},
		},
		{
			name:    "missing",
			header:  http.Header{},
			wantErr: "missing Cosmos block-height",
		},
		{
			name:    "wrong height",
			header:  http.Header{"X-Cosmos-Block-Height": []string{"43"}},
			wantErr: "returned height 43, expected 42",
		},
		{
			name: "conflicting",
			header: http.Header{
				"X-Cosmos-Block-Height":               []string{"42"},
				"Grpc-Metadata-X-Cosmos-Block-Height": []string{"43"},
			},
			wantErr: "conflicting Cosmos block-height",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     tc.header,
					Body:       io.NopCloser(strings.NewReader(`{"ok":true}`)),
					Request:    req,
				}, nil
			})}
			a := &api{client: client, height: 42}
			var target map[string]any
			err := a.getJSON(context.Background(), "http://rest/test", true, &target)
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("getJSON: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("error = %v, want substring %q", err, tc.wantErr)
			}
		})
	}
}

func TestPaginationFailsClosed(t *testing.T) {
	tests := []struct {
		name     string
		page     *page
		wantErr  string
		wantDone bool
	}{
		{name: "missing object", wantErr: "missing pagination object"},
		{name: "missing next key", page: &page{}, wantErr: "has no next_key"},
		{name: "malformed next key", page: &page{NextKey: json.RawMessage(`{}`)}, wantErr: "invalid next_key"},
		{name: "non-base64 next key", page: &page{NextKey: json.RawMessage(`"not base64"`)}, wantErr: "canonical non-empty base64"},
		{name: "null terminates", page: &page{NextKey: json.RawMessage(`null`)}, wantDone: true},
		{name: "empty string terminates", page: &page{NextKey: json.RawMessage(`""`)}, wantDone: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, done, err := advancePage(tc.page, make(map[string]struct{}))
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("error = %v, want substring %q", err, tc.wantErr)
				}
				return
			}
			if err != nil || done != tc.wantDone {
				t.Fatalf("done=%v err=%v, want done=%v", done, err, tc.wantDone)
			}
		})
	}

	seen := make(map[string]struct{})
	keyPage := &page{NextKey: json.RawMessage(`"c2FtZQ=="`)}
	if _, done, err := advancePage(keyPage, seen); err != nil || done {
		t.Fatalf("first key: done=%v err=%v", done, err)
	}
	if _, _, err := advancePage(keyPage, seen); err == nil || !strings.Contains(err.Error(), "repeated next_key") {
		t.Fatalf("expected repeated-key rejection, got %v", err)
	}
}

func TestAccountTypesConsumesEncodedPaginationKeys(t *testing.T) {
	requests := 0
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		requests++
		var body string
		switch requests {
		case 1:
			if got := req.URL.Query().Get("pagination.key"); got != "" {
				t.Fatalf("first pagination key = %q", got)
			}
			body = `{
              "accounts":[{"@type":"/cosmos.auth.v1beta1.BaseAccount","address":"zrn1first"}],
              "pagination":{"next_key":"+/8="}
            }`
		case 2:
			if got := req.URL.Query().Get("pagination.key"); got != "+/8=" {
				t.Fatalf("second pagination key = %q", got)
			}
			body = `{
              "accounts":[{"@type":"/cosmos.auth.v1beta1.BaseAccount","address":"zrn1second"}],
              "pagination":{"next_key":null}
            }`
		default:
			t.Fatalf("unexpected third pagination request")
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Grpc-Metadata-X-Cosmos-Block-Height": []string{"42"}},
			Body:       io.NopCloser(strings.NewReader(body)),
			Request:    req,
		}, nil
	})}
	a := &api{client: client, rest: "http://rest", height: 42}
	accounts, err := a.accountTypes(context.Background())
	if err != nil {
		t.Fatalf("accountTypes: %v", err)
	}
	if requests != 2 || len(accounts) != 2 {
		t.Fatalf("requests=%d accounts=%v", requests, accounts)
	}
}

func TestDenomOwnersLabelsAddressesWithoutAuthRecords(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"X-Cosmos-Block-Height": []string{"42"}},
			Body: io.NopCloser(strings.NewReader(`{
              "denom_owners":[{"address":"zrn1bankonly","balance":{"denom":"uzrn","amount":"7"}}],
              "pagination":{"next_key":null}
            }`)),
			Request: req,
		}, nil
	})}
	a := &api{client: client, rest: "http://rest", height: 42}
	owners, err := a.denomOwners(context.Background(), "uzrn", map[string]accountMetadata{})
	if err != nil {
		t.Fatalf("denomOwners: %v", err)
	}
	if len(owners) != 1 || owners[0].AccountType != "bank_only" {
		t.Fatalf("owners = %+v", owners)
	}
}

func TestDecodeSingleJSONRejectsAmbiguity(t *testing.T) {
	for _, input := range []string{
		`{"a":1}{"b":2}`,
		`{"a":1} trailing`,
		`{"a":1,"a":2}`,
		`{"nested":{"a":1,"a":2}}`,
	} {
		var target map[string]any
		if err := decodeSingleJSON([]byte(input), &target); err == nil {
			t.Fatalf("expected rejection for %q", input)
		}
	}
}

func TestGenesisHashIsSemanticAndChainBound(t *testing.T) {
	hashFor := func(t *testing.T, body, expectedChainID string) (string, error) {
		t.Helper()
		client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{},
				Body:       io.NopCloser(strings.NewReader(body)),
				Request:    req,
			}, nil
		})}
		a := &api{client: client, rpc: "http://rpc"}
		return a.genesisCanonicalSHA(context.Background(), expectedChainID)
	}

	first, err := hashFor(t, `{"result":{"genesis":{"chain_id":"zerone-1","app_state":{"bank":{},"auth":{}},"initial_height":"1"}}}`, "zerone-1")
	if err != nil {
		t.Fatal(err)
	}
	second, err := hashFor(t, `{"result":{"genesis":{"initial_height":"1","app_state":{"auth":{},"bank":{}},"chain_id":"zerone-1"}}}`, "zerone-1")
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatalf("semantic genesis hashes differ: %s != %s", first, second)
	}
	if _, err := hashFor(t, `{"result":{"genesis":{"chain_id":"other"}}}`, "zerone-1"); err == nil || !strings.Contains(err.Error(), "genesis chain ID") {
		t.Fatalf("expected chain-bound genesis rejection, got %v", err)
	}
}

func TestDecimalAmountsAreCanonical(t *testing.T) {
	for _, valid := range []string{"0", "1", "100000000000000000000000000000000000000"} {
		if _, err := parseDecimalAmount(valid); err != nil {
			t.Fatalf("valid amount %q rejected: %v", valid, err)
		}
	}
	for _, invalid := range []string{"", "00", "01", "+1", "-1", "1.0", " 1"} {
		if _, err := parseDecimalAmount(invalid); err == nil {
			t.Fatalf("invalid amount %q accepted", invalid)
		}
	}
}

func TestWriteAtomicReplacesTargetWithoutFixedTempFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "snapshot.json")
	if err := os.WriteFile(path, []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := writeAtomic(path, []byte("new\n")); err != nil {
		t.Fatalf("writeAtomic: %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "new\n" {
		t.Fatalf("content = %q", got)
	}
	matches, err := filepath.Glob(filepath.Join(dir, ".snapshot.json.tmp-*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 0 {
		t.Fatalf("temporary files remain: %v", matches)
	}
}

func TestNormalizeBaseURLRejectsAmbiguousOrSecretProvenance(t *testing.T) {
	got, err := normalizeBaseURL("https://rpc.example.test/cosmos///")
	if err != nil {
		t.Fatal(err)
	}
	if got != "https://rpc.example.test/cosmos" {
		t.Fatalf("normalized URL = %q", got)
	}
	for _, invalid := range []string{
		"ftp://rpc.example.test",
		"https://user:secret@rpc.example.test",
		"https://rpc.example.test?token=secret",
		"https://rpc.example.test#fragment",
		"/relative/path",
	} {
		if _, err := normalizeBaseURL(invalid); err == nil {
			t.Fatalf("invalid base URL %q accepted", invalid)
		}
	}
}
