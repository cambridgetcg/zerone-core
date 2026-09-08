// Command zerone-seed-io is the narrow external native implementation of
// zerone-seed-io/0.1. Build from the repository root with its existing go.mod:
//
//	go build -mod=readonly -o /explicit/path/zerone-seed-io ./tools/zerone-seed-io
//
// No command creates keys, exports keys, retries broadcasts, or admits Wallet
// authority. The caller must durably reserve before possible signing/submission.
package main

import (
	"context"
	"flag"
	"io"
	"os"
	"strings"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

type options struct {
	trustFile      string
	disposable     bool
	unlockTerminal string
}

func init() {
	c := sdk.GetConfig()
	c.SetBech32PrefixForAccount("zrn", "zrnpub")
	c.SetBech32PrefixForValidator("zrnvaloper", "zrnvaloperpub")
	c.SetBech32PrefixForConsensusNode("zrnvalcons", "zrnvalconspub")
}

const help = `zerone-seed-io <command> [--trust-file ABSOLUTE] [--disposable-test] [--unlock-terminal ABSOLUTE]
Commands: inspect simulate sign verify submit lookup operator-grant operator-revoke operator-admit
One canonical zerone-seed-io/0.1 JSON request on stdin, one digest-only JSON response.
No arguments or --help reads this help only. No endpoints, keys, homes, or budgets default.
Every claimant command requires an explicit owner-only --trust-file containing:
  protocol="zerone-seed-trust/0.1", profile_id, node (exact SeedNodeConfig),
  genesis_file, runtime_file, source_manifest_file, disposable_test (boolean).
The host independently approves that mapping. Local hashing does not attest remote runtime bytes.
All native queries use typed protobuf over the configured RPC's ABCI query service;
the explicit grpc_address is validated trust configuration, never a fallback endpoint.
The source manifest is exact canonical JSON; its digest and the local runtime,
self executable and exact genesis bytes must equal profile pins. No pin is inferred.
Existing file keyrings need a separately attached --unlock-terminal; existing test
keyrings require --disposable-test and seed-local-* chain reference. os/pass are
unsupported rather than silently discovering or creating an ambient key store.
Paths must be absolute and symlink-free; private parents 0700 and files 0600.
Simulation is latest-only with an empty mempool and unchanged height around the
call: SDK Simulate uses checkState and cannot honor arbitrary historical height.
A helper result is not Wallet authorization, release acceptance, or effect consent.
`

// Used only for an otherwise completed command whose total budget was crossed.
// Earlier failures retain their restrictive error; no signed output is deleted.
func deadlineResponse(response object) object {
	command := str(response, "command")
	if command == "submit" {
		response["result"] = object{"status": "submission_unknown", "tx_hash": obj(response["result"])["tx_hash"]}
		return response
	}
	code := "limit_exceeded"
	if command == "sign" {
		code = "signing_unknown"
	}
	return object{"protocol": protocol, "request_id": response["request_id"], "status": "error", "command": command, "code": code}
}

func main() {
	if run(os.Args[1:], os.Stdin, os.Stdout) != 0 {
		os.Exit(1)
	}
}
func run(args []string, input io.Reader, output io.Writer) (exit int) {
	start := time.Now()
	command := "inspect"
	id := "invalid"
	var response object
	var deadline time.Time
	defer func() {
		if e := recover(); e != nil {
			code := "internal_error"
			switch x := e.(type) {
			case failure:
				code = string(x)
			case nodeFailure:
				code = "observation_unknown"
			}
			allowed := map[string]bool{"invalid_request": true, "unsupported_command": true, "limit_exceeded": true, "profile_mismatch": true, "policy_mismatch": true, "plan_mismatch": true, "node_unavailable": true, "observation_unknown": true, "key_mismatch": true, "unsafe_path": true, "already_exists": true, "signature_invalid": true, "signing_unknown": true, "internal_error": true}
			if !allowed[code] {
				code = "internal_error"
			}
			response = object{"protocol": protocol, "request_id": id, "status": "error", "command": command, "code": code}
			exit = 1
		}
		if response != nil {
			b := canonical(response)
			// Include trust hashing, file-only verification and response encoding in
			// the total budget, without turning a possible effect into no-effect.
			// Use the deadline itself: normal deferred context cleanup has run.
			if !deadline.IsZero() && !time.Now().Before(deadline) && response["status"] == "ok" {
				response = deadlineResponse(response)
				b = canonical(response)
				if response["status"] == "error" {
					exit = 1
				}
			}
			if len(b) > maxJSON {
				b = canonical(object{"protocol": protocol, "request_id": id, "status": "error", "command": command, "code": "limit_exceeded"})
				exit = 1
			}
			if _, e := output.Write(append(b, '\n')); e != nil {
				exit = 1
			}
		}
	}()
	if len(args) == 0 || args[0] == "--help" || args[0] == "help" || args[0] == "-h" {
		_, e := io.WriteString(output, help)
		if e != nil {
			return 1
		}
		return 0
	}
	require(!strings.ContainsAny(args[0], " \t\r\n") && strings.Contains(" inspect simulate sign verify submit lookup operator-grant operator-revoke operator-admit ", " "+args[0]+" "), "unsupported_command")
	command = args[0]
	flags := flag.NewFlagSet("zerone-seed-io", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	opts := options{}
	flags.StringVar(&opts.trustFile, "trust-file", "", "explicit independently approved host trust mapping")
	flags.BoolVar(&opts.disposable, "disposable-test", false, "opt in to an existing disposable seed-local test keyring")
	flags.StringVar(&opts.unlockTerminal, "unlock-terminal", "", "explicit terminal for an existing file keyring")
	require(flags.Parse(args[1:]) == nil && flags.NArg() == 0, "invalid_request")
	if command != "sign" {
		require(opts.unlockTerminal == "", "invalid_request")
	}
	type readResult struct {
		b []byte
		e error
	}
	ch := make(chan readResult, 1)
	go func() { b, e := io.ReadAll(io.LimitReader(input, maxJSON+2)); ch <- readResult{b, e} }()
	var data []byte
	select {
	case read := <-ch:
		require(read.e == nil, "invalid_request")
		data = read.b
	case <-time.After(30 * time.Second):
		panic(failure("limit_exceeded"))
	}
	r := parseJSON(data)
	validateRequest(r, command)
	id = str(r, "request_id")
	deadline = start.Add(time.Duration(num(r, "timeout_ms")) * time.Millisecond)
	ctx, cancel := context.WithDeadline(context.Background(), deadline)
	defer cancel()
	checkDeadline(ctx)
	var result object
	if strings.HasPrefix(command, "operator-") {
		result = operatorMessage(r, time.Now())
	} else {
		trust := loadTrust(ctx, r, opts)
		checkDeadline(ctx)
		if command == "sign" || command == "submit" {
			pol := obj(r["policy"])
			now := time.Now()
			require(!now.Before(timestamp(str(pol, "not_before"))) && now.Before(timestamp(str(pol, "expires_at"))), "policy_mismatch")
			require(now.Before(timestamp(str(obj(obj(r["plan"])["commitment"]), "expires_at"))), "plan_mismatch")
		}
		switch command {
		case "sign":
			result = signExisting(ctx, r, opts)
		case "verify":
			result = signedSummary(obj(r["plan"]), readPrivate(ctx, str(r, "signed_tx_path"), maxProto))
		default:
			n := newNode(ctx, r, time.Now)
			defer n.close()
			switch command {
			case "inspect":
				result = n.inspect(r["height"], trust)
			case "simulate":
				result = n.simulate(r, trust)
			case "submit":
				result = n.submit(r, trust)
			case "lookup":
				result = n.lookup(r, trust)
			}
		}
	}
	response = object{"protocol": protocol, "request_id": id, "status": "ok", "command": command, "result": result}
	return 0
}
