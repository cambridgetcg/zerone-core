package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"

	rpctypes "github.com/cometbft/cometbft/rpc/core/types"
	"github.com/cosmos/cosmos-sdk/types/bech32"
)

func TestCompiledPolicies(t *testing.T) {
	want := map[string]struct {
		address, key, power, source string
	}{
		"zerone-dev-1": {"E85D65FC900DEFEB9FF160F1E6EEA2CA00768420", "C7jBQ2qhJ6uORoLnFoshmLIeKOeOtDnDlCsWdpgsJBg=", "1000000", "../research-checkpoint/verify.go"},
		"zerone-1":     {"93DBC1A1D9A07E67423B193395C9EB6F1E41B844", "LA944mpk44k8Jr9EN/uOzSSf8G+Ng2LiIvf9FJBkl80=", "11111", "../../deploy/legacy-observer/checkpoint/main.go"},
	}
	ps := compiledPolicies()
	if len(ps) != len(want) {
		t.Fatalf("compiled policy table has %d entries", len(ps))
	}
	for id, w := range want {
		p := ps[id]
		a := p.attestation()
		if p.ChainID != id || p.Label != "ONE_OPERATOR" || !strings.Contains(p.Words, "not independent confirmation") || !strings.Contains(p.Words, "not a promise of finality") || !strings.Contains(p.Source, strings.TrimLeft(w.source, "./")) {
			t.Fatalf("%s policy words or source: %+v", id, p)
		}
		if len(a.Validators) != 1 || a.Validators[0].Address != w.address || a.Validators[0].PublicKey != w.key || a.Validators[0].VotingPower != w.power {
			t.Fatalf("%s pin: %+v", id, a.Validators)
		}
		// The pinned key is the one the cited repository file already holds.
		src, err := os.ReadFile(w.source)
		if err != nil || !bytes.Contains(src, []byte(w.key)) {
			t.Fatalf("%s key not found in %s: %v", id, w.source, err)
		}
	}
	legacy, _ := os.ReadFile(want["zerone-1"].source)
	if !bytes.Contains(legacy, []byte(want["zerone-1"].address)) || !bytes.Contains(legacy, []byte("observedPower     = "+want["zerone-1"].power)) {
		t.Fatal("zerone-1 address or power differs from its source")
	}
	// The zerone-dev-1 power is the one its cited validator response records.
	raw, err := readInput(filepath.Join(publishedCheckpoint, "validators.json"))
	if err != nil {
		t.Fatal(err)
	}
	var v rpctypes.ResultValidators
	if err := rpcResult(raw, &v); err != nil {
		t.Fatal(err)
	}
	dev := ps["zerone-dev-1"].Validators
	if !strings.Contains(ps["zerone-dev-1"].Source, "2026-09-14-demimetric-v1/chain/validators.json") || v.BlockHeight != 183706 || v.Count != 1 || v.Total != 1 || len(v.Validators) != 1 || len(dev) != 1 ||
		base64.StdEncoding.EncodeToString(v.Validators[0].PubKey.Bytes()) != dev[0].PublicKey || upperHex(v.Validators[0].Address) != want["zerone-dev-1"].address || v.Validators[0].VotingPower != dev[0].Power {
		t.Fatalf("zerone-dev-1 pin differs from its cited validator response: %+v", v)
	}
	for _, bad := range []policy{
		{},
		{ChainID: "x", Label: "L", Words: "W", Source: "S", Validators: []pinnedValidator{{PublicKey: "not base64", Power: 1}}},
		{ChainID: "x", Label: "L", Words: "W", Source: "S", Validators: []pinnedValidator{{PublicKey: want["zerone-1"].key, Power: 0}}},
		{ChainID: "x", Label: "L", Words: "W", Source: "S", Validators: []pinnedValidator{{PublicKey: want["zerone-1"].key, Power: 1}, {PublicKey: want["zerone-1"].key, Power: 1}}},
	} {
		if _, err := bad.validatorSet(); err == nil {
			t.Fatalf("invalid policy accepted: %+v", bad)
		}
	}
}

func TestReadInputRefusals(t *testing.T) {
	d := t.TempDir()
	write := func(name string, b []byte) string {
		p := filepath.Join(d, name)
		if err := os.WriteFile(p, b, 0600); err != nil {
			t.Fatal(err)
		}
		return p
	}
	ordinary := write("ok.json", []byte(`{"ok":true}`))
	if _, err := readInput(ordinary); err != nil {
		t.Fatal(err)
	}
	symlink := filepath.Join(d, "link.json")
	if err := os.Symlink(ordinary, symlink); err != nil {
		t.Fatal(err)
	}
	fifo := filepath.Join(d, "fifo")
	if err := syscall.Mkfifo(fifo, 0600); err != nil {
		t.Fatal(err)
	}
	big := filepath.Join(d, "big.json")
	f, err := os.Create(big)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Truncate(maxInputBytes + 1); err != nil {
		t.Fatal(err)
	}
	f.Close()
	for name, p := range map[string]string{
		"symlink": symlink, "FIFO": fifo, "directory": d, "device": os.DevNull, "missing": filepath.Join(d, "missing"), "oversized": big,
		"empty":           write("empty.json", nil),
		"invalid UTF-8":   write("utf8.json", []byte("{\"a\":\"\xff\"}")),
		"duplicate key":   write("dup.json", []byte(`{"a":1,"a":2}`)),
		"nested dup":      write("nested.json", []byte(`{"x":{"a":1,"a":2}}`)),
		"trailing value":  write("trailing.json", []byte(`{} {}`)),
		"deep nesting":    write("deep.json", []byte(strings.Repeat("[", 66)+"0"+strings.Repeat("]", 66))),
		"not JSON":        write("text.json", []byte("hello")),
		"unclosed object": write("open.json", []byte(`{"a":1`)),
	} {
		if _, err := readInput(p); err == nil {
			t.Fatalf("%s accepted", name)
		}
	}
	var out map[string]any
	if err := rpcResult([]byte(`{"jsonrpc":"2.0","id":1,"result":{},"error":null}`), &out); err != nil {
		t.Fatal(err)
	}
	for name, raw := range map[string]string{
		"JSON-RPC 1.0":  `{"jsonrpc":"1.0","result":{}}`,
		"error member":  `{"jsonrpc":"2.0","result":{},"error":{"code":1}}`,
		"null result":   `{"jsonrpc":"2.0","result":null}`,
		"no result":     `{"jsonrpc":"2.0","id":1}`,
		"no version":    `{"result":{}}`,
		"not an object": `[{"jsonrpc":"2.0","result":{}}]`,
		// encoding/json would match these to result, error and jsonrpc.
		"Result beside result":   `{"jsonrpc":"2.0","result":{"a":1},"Result":{"a":2}}`,
		"long-s result":          `{"jsonrpc":"2.0","result":{"a":1},"reſult":{"a":2}}`,
		"ERROR beside error":     `{"jsonrpc":"2.0","result":{},"ERROR":{"code":1},"error":null}`,
		"JSONRPC beside jsonrpc": `{"jsonrpc":"1.0","JSONRPC":"2.0","result":{}}`,
		"unknown member":         `{"jsonrpc":"2.0","result":{},"extra":1}`,
	} {
		if err := rpcResult([]byte(raw), &out); err == nil {
			t.Fatalf("%s envelope accepted", name)
		}
	}
}

// publishedCheckpoint holds evidence this repository already publishes for
// tools/research-checkpoint. The tests read it offline; no key is in it.
var publishedCheckpoint = filepath.Join("..", "..", "dashboard", "public", "research", "checkpoints", "2026-09-14-demimetric-v1", "chain")

// TestPublishedTransferProof runs saved CometBFT RPC responses, not ones this
// package encoded, through the compiled zerone-dev-1 pin.
func TestPublishedTransferProof(t *testing.T) {
	read := func(name string) []byte {
		t.Helper()
		b, err := readInput(filepath.Join(publishedCheckpoint, name))
		if err != nil {
			t.Fatal(err)
		}
		return b
	}
	e := transferEvidence{Tx: read("tx-rpc.json"), Commit: read("commit.json"), Validators: read("validators.json"),
		BlockResults: read("block-results.json"), NextCommit: read("next-commit.json"), NextValidators: read("next-validators.json")}
	sender := "zrn1q36wcvumlxa3qrmsu8cmkrjqsa4jqfchtzkuj8"
	memo := "zerone:research:v1:ce6e799a-6713-41a9-b95f-57c50a223074:81:86c8163599c0098aa43b4f3599a81b02a03557553fc595b67b026fdfe108704e:23baf5b0a2e7f4a48fe2b86775ae2ef92e607f2d793270ff10219e61d2563847"
	w := transferWant{TxHash: "8DFC740C5326E76B482B6D370F527311CB518934D6B64A4BF5EFDBD2806D0B8E", Height: 183706, Sender: sender, Recipient: sender, Amount: "1", AccountNumber: 7, Memo: &memo, RequireExecution: true}
	ps := compiledPolicies()
	r, ok := checkTransfer(t, ps["zerone-dev-1"], e, w)
	c := r.Claims
	if !ok || c.Inclusion.Status != statusProven || c.Authorisation.Status != statusProven || c.Execution.Status != statusProven || c.TransferEffect.Status != statusProven ||
		r.Execution["gas_used"] != "100584" || r.Execution["tx_count"] != int64(1) || r.Transaction["byte_length"] != 501 || r.Transaction["memo_class"] != "other" {
		t.Fatalf("published evidence not proven: ok=%v %+v %+v %+v", ok, c, r.Execution, r.Transaction)
	}
	if r, ok := checkTransfer(t, ps["zerone-1"], e, w); ok || !strings.Contains(r.Claims.Inclusion.Reason, "another chain") {
		t.Fatalf("zerone-dev-1 evidence accepted under the zerone-1 pin: %+v", r.Claims)
	}
}

func runCLI(t *testing.T, policies map[string]policy, args ...string) (int, string, string) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	code := run(args, &stdout, &stderr, policies)
	return code, stdout.String(), stderr.String()
}

func TestCommandLine(t *testing.T) {
	chain := newTestChain("test-a", 100)
	policies := map[string]policy{"test-a": chain.pol}
	d := t.TempDir()
	save := func(name string, b []byte) string {
		p := filepath.Join(d, name)
		if err := os.WriteFile(p, b, 0600); err != nil {
			t.Fatal(err)
		}
		return p
	}
	transferArgs := func(f *transferFixture, prefix string) []string {
		return []string{"transfer", "--chain", "test-a", "--tx", save(prefix+"tx.json", f.e.Tx), "--commit", save(prefix+"commit.json", f.e.Commit), "--validators", save(prefix+"validators.json", f.e.Validators),
			"--txhash", f.w.TxHash, "--height", strconv.FormatInt(f.w.Height, 10), "--sender", f.w.Sender, "--recipient", f.w.Recipient, "--amount", f.w.Amount, "--account-number", "9"}
	}
	f := newTransfer(t, chain, txSpec{memo: "zr1:pay:" + strings.Repeat("ab", 32)})
	transfer := transferArgs(f, "")
	group := []string{"--block-results", save("block-results.json", f.e.BlockResults), "--next-commit", save("next-commit.json", f.e.NextCommit), "--next-validators", save("next-validators.json", f.e.NextValidators)}
	with := func(base []string, extra ...string) []string { return append(append([]string{}, base...), extra...) }
	otherPrefix, err := bech32.ConvertAndEncode("cosmos", make([]byte, 20))
	if err != nil {
		t.Fatal(err)
	}

	code, out, _ := runCLI(t, policies, with(transfer, append(group, "--require-execution", "--pay-digest", strings.Repeat("ab", 32))...)...)
	var r transferReport
	if code != 0 || json.Unmarshal([]byte(out), &r) != nil || r.Schema != transferSchema || r.Claims.TransferEffect.Status != statusProven || r.Claims.Events.Status != statusNotProven || r.AttestedBy.Label != "TEST_ONLY" {
		t.Fatalf("transfer: exit %d\n%s", code, out)
	}
	if code, out, _ = runCLI(t, policies, with(transfer, "--memo", "other text")...); code != 1 || !strings.Contains(out, `"memo differs from --memo"`) {
		t.Fatalf("failed proof must print its report and exit 1: %d\n%s", code, out)
	}
	// Without the H+1 inputs, the /tx response's code is an observation, and a
	// reported failure exits 1.
	if code, out, _ = runCLI(t, policies, transfer...); code != 0 || !strings.Contains(out, `"status": "endpoint_observation_only"`) || strings.Contains(out, "tx_index") {
		t.Fatalf("inclusion-only transfer: exit %d\n%s", code, out)
	}
	if code, out, _ = runCLI(t, policies, transferArgs(newTransfer(t, chain, txSpec{code: 4}), "failed-")...); code != 1 || !strings.Contains(out, `"code": 4`) || !strings.Contains(out, "reports code 4") {
		t.Fatalf("transfer the endpoint reports as failed: exit %d\n%s", code, out)
	}
	// The compiled table knows no test chain, and its pins refuse test keys.
	if code, out, _ = runCLI(t, compiledPolicies(), transfer...); code != 2 || out != "" {
		t.Fatalf("compiled table accepted a test chain: %d", code)
	}
	if code, out, _ = runCLI(t, compiledPolicies(), replace(transfer, "--chain", "zerone-1")...); code != 1 || !strings.Contains(out, `"label": "ONE_OPERATOR"`) || strings.Contains(out, `"status": "PROVEN"`) {
		t.Fatalf("test evidence under the zerone-1 pin: exit %d\n%s", code, out)
	}

	s := newStateFixture(t, chain)
	w := stateWant{Height: s.h, Kind: "account", Address: address(t, s.holder)}
	e := s.valid(t, w)
	state := []string{"state", "--chain", "test-a", "--query", save("query.json", e.Query), "--next-commit", save("state-next-commit.json", e.NextCommit), "--next-validators", save("state-next-validators.json", e.NextValidators),
		"--height", "3", "--kind", "account", "--address", w.Address}
	var sr stateReport
	if code, out, _ = runCLI(t, policies, state...); code != 0 || json.Unmarshal([]byte(out), &sr) != nil || sr.Schema != stateSchema || sr.Account["sequence"] != "3" {
		t.Fatalf("state: exit %d\n%s", code, out)
	}

	usage := map[string][]string{
		"no command":                       nil,
		"unknown command":                  {"balance"},
		"unknown flag":                     with(transfer, "--policy", "p.json"),
		"positional argument":              with(transfer, "extra"),
		"repeated flag":                    with(transfer, "--amount", f.w.Amount),
		"repeated boolean flag":            with(transfer, append(group, "--require-execution", "--require-execution")...),
		"boolean flag cancelled":           with(transfer, "--require-execution", "--require-execution=false"),
		"boolean flag with a bad value":    with(transfer, append(group, "--require-execution=maybe")...),
		"missing --tx":                     cut(transfer, "--tx"),
		"missing --chain":                  cut(transfer, "--chain"),
		"unknown chain":                    replace(transfer, "--chain", "test-b"),
		"memo and pay digest":              with(transfer, "--memo", "", "--pay-digest", strings.Repeat("ab", 32)),
		"uppercase pay digest":             with(transfer, "--pay-digest", strings.Repeat("AB", 32)),
		"amount with leading zero":         replace(transfer, "--amount", "0250"),
		"zero amount":                      replace(transfer, "--amount", "0"),
		"negative amount":                  replace(transfer, "--amount", "-250"),
		"decimal amount":                   replace(transfer, "--amount", "2.5"),
		"uppercase sender":                 replace(transfer, "--sender", strings.ToUpper(f.w.Sender)),
		"sender with another prefix":       replace(transfer, "--sender", otherPrefix),
		"recipient of 21 bytes":            replace(transfer, "--recipient", address(t, make([]byte, 21))),
		"short txhash":                     replace(transfer, "--txhash", "abc"),
		"zero height":                      replace(transfer, "--height", "0"),
		"height with leading zero":         replace(transfer, "--height", "040"),
		"account number not decimal":       replace(transfer, "--account-number", "0x9"),
		"partial execution group":          with(transfer, group[:2]...),
		"required execution without group": with(transfer, "--require-execution"),
		"empty file path":                  replace(transfer, "--tx", ""),
		"unknown kind":                     replace(state, "--kind", "sequence"),
		"denom with account":               with(state, "--denom", "uzrn"),
		"invalid denom":                    with(replace(state, "--kind", "balance"), "--denom", "1"),
		"state height missing":             cut(state, "--height"),
	}
	for name, args := range usage {
		if code, out, errText := runCLI(t, policies, args...); code != 2 || out != "" || !strings.Contains(errText, "usage:") {
			t.Fatalf("%s: exit %d, stdout %q", name, code, out)
		}
	}

	// Refused evidence files exit 1 without a report.
	link := filepath.Join(d, "link.json")
	if err := os.Symlink(filepath.Join(d, "tx.json"), link); err != nil {
		t.Fatal(err)
	}
	for name, p := range map[string]string{"symlink": link, "device": os.DevNull, "duplicate key": save("dup.json", []byte(`{"jsonrpc":"2.0","jsonrpc":"2.0"}`))} {
		if code, out, _ := runCLI(t, policies, replace(transfer, "--tx", p)...); code != 1 || out != "" {
			t.Fatalf("%s: exit %d, stdout %q", name, code, out)
		}
	}
	// An invalid compiled policy is an error, not a usage mistake.
	broken := map[string]policy{"test-a": {ChainID: "test-a"}}
	if code, _, _ := runCLI(t, broken, transfer...); code != 1 {
		t.Fatalf("invalid policy: exit %d", code)
	}
}

func replace(args []string, flagName, value string) []string {
	out := append([]string{}, args...)
	for i := range out {
		if out[i] == flagName {
			out[i+1] = value
		}
	}
	return out
}

func cut(args []string, flagName string) []string {
	out := []string{}
	for i := 0; i < len(args); i++ {
		if args[i] == flagName {
			i++
			continue
		}
		out = append(out, args[i])
	}
	return out
}
