// ledger-proof checks saved, read-only RPC responses offline and prints one
// JSON report. It signs nothing, broadcasts nothing, queries no network and
// reads no keyring. Its only trust roots are the policies compiled into
// policy.go.
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"syscall"
	"unicode/utf8"

	cmtjson "github.com/cometbft/cometbft/libs/json"
	"github.com/cosmos/cosmos-sdk/types/bech32"
	proto "github.com/cosmos/gogoproto/proto"
)

const maxInputBytes = 8 << 20

const usageText = `usage:
  ledger-proof transfer --chain ID --tx FILE --commit FILE --validators FILE
      --txhash HEX --height H --sender ADDR --recipient ADDR --amount UZRN
      --account-number N [--memo TEXT | --pay-digest HEX]
      [--block-results FILE --next-commit FILE --next-validators FILE]
      [--require-execution]
  ledger-proof state --chain ID --query FILE --next-commit FILE
      --next-validators FILE --height H --kind balance|account --address ADDR
      [--denom DENOM]
exit status: 0 every requested proof holds, 1 one does not, 2 usage error`

const (
	statusProven        = "PROVEN"
	statusNotProven     = "NOT_PROVEN"
	statusNotApplicable = "NOT_APPLICABLE"
)

type claim struct {
	Status string `json:"status"`
	Reason string `json:"reason,omitempty"`
}

func proven() claim                     { return claim{Status: statusProven} }
func notProven(reason string) claim     { return claim{Status: statusNotProven, Reason: reason} }
func notApplicable(reason string) claim { return claim{Status: statusNotApplicable, Reason: reason} }

type usageError string

func (u usageError) Error() string { return string(u) }

func usagef(format string, a ...any) error { return usageError(fmt.Sprintf(format, a...)) }

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr, compiledPolicies()))
}

// run is the whole command. main passes the compiled policy table; tests pass
// their own. No flag, environment variable or input file reaches this parameter.
func run(args []string, stdout, stderr io.Writer, policies map[string]policy) int {
	var report any
	var ok bool
	var err error
	switch {
	case len(args) == 0:
		err = usageError("missing command")
	case args[0] == "transfer":
		report, ok, err = runTransfer(args[1:], policies)
	case args[0] == "state":
		report, ok, err = runState(args[1:], policies)
	default:
		err = usagef("unknown command %q", args[0])
	}
	var u usageError
	if errors.As(err, &u) {
		fmt.Fprintf(stderr, "ledger-proof: %s\n%s\n", u, usageText)
		return 2
	}
	if err != nil {
		fmt.Fprintln(stderr, "ledger-proof:", err)
		return 1
	}
	enc := json.NewEncoder(stdout)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	if err := enc.Encode(report); err != nil {
		fmt.Fprintln(stderr, "ledger-proof:", err)
		return 1
	}
	if !ok {
		return 1
	}
	return 0
}

// stringFlag refuses a second value, so a repeated flag cannot silently
// replace an earlier expectation.
type stringFlag struct {
	value string
	given bool
}

func (s *stringFlag) String() string { return s.value }

func (s *stringFlag) Set(v string) error {
	if s.given {
		return errors.New("given more than once")
	}
	s.value, s.given = v, true
	return nil
}

// boolFlag is a boolean flag that, like stringFlag, refuses a second value,
// so a later --require-execution=false cannot cancel an earlier one.
type boolFlag struct {
	value, given bool
}

func (b *boolFlag) String() string   { return strconv.FormatBool(b.value) }
func (b *boolFlag) IsBoolFlag() bool { return true }

func (b *boolFlag) Set(v string) error {
	if b.given {
		return errors.New("given more than once")
	}
	x, err := strconv.ParseBool(v)
	if err != nil {
		return err
	}
	b.value, b.given = x, true
	return nil
}

func parseFlags(name string, args []string, names []string, bools map[string]*boolFlag) (map[string]*stringFlag, error) {
	f := flag.NewFlagSet(name, flag.ContinueOnError)
	f.SetOutput(io.Discard)
	values := map[string]*stringFlag{}
	for _, n := range names {
		values[n] = &stringFlag{}
		f.Var(values[n], n, "")
	}
	for n, b := range bools {
		f.Var(b, n, "")
	}
	if err := f.Parse(args); err != nil {
		return nil, usagef("%s: %v", name, err)
	}
	if f.NArg() != 0 {
		return nil, usagef("%s: unexpected argument %q", name, f.Arg(0))
	}
	return values, nil
}

func requireFlags(values map[string]*stringFlag, names ...string) error {
	for _, n := range names {
		if !values[n].given || values[n].value == "" {
			return usagef("missing --%s", n)
		}
	}
	return nil
}

func lookupPolicy(policies map[string]policy, chain string) (policy, error) {
	p, ok := policies[chain]
	if !ok {
		return policy{}, usagef("no compiled trust policy for chain %q", chain)
	}
	if _, err := p.validatorSet(); err != nil {
		return policy{}, fmt.Errorf("trust policy for %q: %w", chain, err)
	}
	return p, nil
}

func canonicalUint(s, label string) (uint64, error) {
	v, err := strconv.ParseUint(s, 10, 64)
	if err != nil || strconv.FormatUint(v, 10) != s {
		return 0, usagef("--%s requires a canonical unsigned decimal", label)
	}
	return v, nil
}

// Heights stop one short of the int64 limit because H+1 must exist.
func parseHeight(s string) (int64, error) {
	v, err := canonicalUint(s, "height")
	if err != nil || v == 0 || v >= uint64(^uint64(0)>>1) {
		return 0, usageError("--height requires a positive decimal below the int64 limit")
	}
	return int64(v), nil
}

// parseAddress accepts only lowercase bech32 with the zrn prefix and one of
// the given byte lengths.
func parseAddress(s string, sizes ...int) ([]byte, error) {
	prefix, b, err := bech32.DecodeAndConvert(s)
	if err != nil || prefix != "zrn" || s != strings.ToLower(s) {
		return nil, errors.New("not a lowercase zrn bech32 address")
	}
	for _, n := range sizes {
		if len(b) == n {
			return b, nil
		}
	}
	return nil, fmt.Errorf("address is %d bytes", len(b))
}

func readFiles(values map[string]*stringFlag, names ...string) (map[string][]byte, error) {
	out := map[string][]byte{}
	for _, n := range names {
		if !values[n].given {
			continue
		}
		if values[n].value == "" {
			return nil, usagef("--%s needs a file path", n)
		}
		b, err := readInput(values[n].value)
		if err != nil {
			return nil, fmt.Errorf("--%s: %w", n, err)
		}
		out[n] = b
	}
	return out, nil
}

// Duplicate keys are ambiguous evidence even when the later value looks valid.
func strictJSON(b []byte) error {
	if len(b) == 0 || len(b) > maxInputBytes || !utf8.Valid(b) {
		return errors.New("JSON input size or UTF-8 invalid")
	}
	d := json.NewDecoder(bytes.NewReader(b))
	d.UseNumber()
	var value func(int) error
	value = func(depth int) error {
		if depth > 64 {
			return errors.New("JSON nesting exceeds 64")
		}
		t, err := d.Token()
		if err != nil {
			return err
		}
		c, ok := t.(json.Delim)
		if !ok {
			return nil
		}
		switch c {
		case '{':
			seen := map[string]bool{}
			for d.More() {
				k, err := d.Token()
				if err != nil {
					return err
				}
				s, ok := k.(string)
				if !ok || seen[s] {
					return errors.New("duplicate or invalid JSON object key")
				}
				seen[s] = true
				if err := value(depth + 1); err != nil {
					return err
				}
			}
			end, err := d.Token()
			if err != nil || end != json.Delim('}') {
				return errors.New("unclosed JSON object")
			}
		case '[':
			for d.More() {
				if err := value(depth + 1); err != nil {
					return err
				}
			}
			end, err := d.Token()
			if err != nil || end != json.Delim(']') {
				return errors.New("unclosed JSON array")
			}
		default:
			return errors.New("unexpected JSON delimiter")
		}
		return nil
	}
	if err := value(0); err != nil {
		return err
	}
	if _, err := d.Token(); err != io.EOF {
		return errors.New("trailing JSON value")
	}
	return nil
}

// Nonblocking open keeps a FIFO from stalling before the regular-file check.
func readInput(path string) ([]byte, error) {
	fd, err := syscall.Open(path, syscall.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK|syscall.O_CLOEXEC, 0)
	if err != nil {
		return nil, err
	}
	f := os.NewFile(uintptr(fd), path)
	defer f.Close()
	s, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if !s.Mode().IsRegular() || s.Size() <= 0 || s.Size() > maxInputBytes {
		return nil, errors.New("expected nonempty bounded regular evidence file")
	}
	b, err := io.ReadAll(io.LimitReader(f, maxInputBytes+1))
	if err != nil {
		return nil, err
	}
	if len(b) > maxInputBytes {
		return nil, errors.New("evidence exceeds 8 MiB")
	}
	if err := strictJSON(b); err != nil {
		return nil, err
	}
	return b, nil
}

// rpcResult reads the envelope's members by their exact names. encoding/json
// matches struct fields without regard to case, so a struct would let
// "Result" or "reſult" stand in for the "result" another reader sees.
func rpcResult(raw []byte, out any) error {
	if err := strictJSON(raw); err != nil {
		return err
	}
	var e map[string]json.RawMessage
	if err := json.Unmarshal(raw, &e); err != nil {
		return err
	}
	for k := range e {
		switch k {
		case "jsonrpc", "id", "result", "error":
		default:
			return fmt.Errorf("unexpected JSON-RPC envelope member %q", k)
		}
	}
	var version string
	if err := json.Unmarshal(e["jsonrpc"], &version); err != nil || version != "2.0" ||
		len(e["result"]) == 0 || bytes.Equal(e["result"], []byte("null")) || (len(e["error"]) != 0 && !bytes.Equal(e["error"], []byte("null"))) {
		return errors.New("expected successful JSON-RPC 2.0 result")
	}
	return cmtjson.Unmarshal(e["result"], out)
}

// Canonical re-encoding refuses fields and encodings this decoder normalizes.
// Any retains unknown wire bytes, so callers check accepted Any wrappers.
func canonicalProto(raw []byte, msg proto.Message) error {
	if err := proto.Unmarshal(raw, msg); err != nil {
		return err
	}
	b, err := proto.Marshal(msg)
	if err != nil {
		return err
	}
	if !bytes.Equal(raw, b) {
		return errors.New("noncanonical or unknown protobuf fields")
	}
	return nil
}

func digest(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }

func upperHex(b []byte) string { return strings.ToUpper(hex.EncodeToString(b)) }

func inputHashes(inputs map[string][]byte) map[string]string {
	out := map[string]string{}
	for k, b := range inputs {
		if len(b) != 0 {
			out[k] = digest(b)
		}
	}
	return out
}
