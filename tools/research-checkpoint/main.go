package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"syscall"
	"unicode/utf8"
)

const maxInputBytes = 8 << 20

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

func readInput(path string) ([]byte, error) {
	fd, err := syscall.Open(path, syscall.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
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

func writeReceipt(path string, r *Receipt) error {
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	if _, err := f.Write(b); err != nil {
		f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

func canonicalUint(s, label string) (uint64, error) {
	v, err := strconv.ParseUint(s, 10, 64)
	if err != nil || strconv.FormatUint(v, 10) != s {
		return 0, fmt.Errorf("%s requires canonical unsigned decimal", label)
	}
	return v, nil
}

func run(args []string, stdout io.Writer) error {
	f := flag.NewFlagSet("research-checkpoint", flag.ContinueOnError)
	f.SetOutput(stdout)
	paths := map[string]*string{}
	for _, name := range []string{"genesis", "tx", "commit", "validators", "block-results", "next-commit", "next-validators"} {
		paths[name] = f.String(name, "", "saved public JSON evidence file (offline)")
	}
	h := f.String("height", "", "expected committed transaction height")
	hash := f.String("txhash", "", "expected SHA-256 transaction hash")
	memo := f.String("memo", "", "exact expected zerone:research:v1 checkpoint memo")
	sender := f.String("sender", "", "expected self-send signer zrn address")
	account := f.String("account-number", "", "independently recorded signing account number")
	gas := f.String("gas", "", "exact signed gas limit")
	fee := f.String("fee-uzrn", "", "exact signed positive fee in uzrn")
	required := f.Bool("require-execution-proof", false, "refuse unless H+1 signed result-root proof is supplied")
	out := f.String("output", "", "new receipt path; existing files are never overwritten")
	if err := f.Parse(args); err != nil {
		return err
	}
	if f.NArg() != 0 || *out == "" || *sender == "" || *memo == "" {
		return errors.New("supply all expected values and --output; no positional arguments")
	}
	height, err := canonicalUint(*h, "height")
	if err != nil || height == 0 || height > uint64(^uint64(0)>>1) {
		return errors.New("height requires positive int64 decimal")
	}
	accountNumber, err := canonicalUint(*account, "account-number")
	if err != nil {
		return err
	}
	gasLimit, err := canonicalUint(*gas, "gas")
	if err != nil {
		return err
	}
	input := map[string][]byte{}
	for _, name := range []string{"genesis", "tx", "commit", "validators", "block-results", "next-commit", "next-validators"} {
		if *paths[name] == "" {
			if name == "genesis" || name == "tx" || name == "commit" || name == "validators" {
				return fmt.Errorf("missing --%s", name)
			}
			continue
		}
		b, err := readInput(*paths[name])
		if err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
		input[name] = b
	}
	r, err := verify(Evidence{Genesis: input["genesis"], Tx: input["tx"], Commit: input["commit"], Validators: input["validators"], BlockResults: input["block-results"], NextCommit: input["next-commit"], NextValidators: input["next-validators"]}, Expected{Height: int64(height), TxHash: *hash, Memo: *memo, Sender: *sender, AccountNumber: accountNumber, Gas: gasLimit, Fee: *fee, RequireExecutionProof: *required})
	if err != nil {
		return err
	}
	if err := writeReceipt(*out, r); err != nil {
		return fmt.Errorf("receipt: %w", err)
	}
	fmt.Fprintf(stdout, "Verified checkpoint %s at %s height %s; execution: %s\n", r.Transaction["txhash"], chainID, r.Height, r.Execution["status"])
	return nil
}

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "research-checkpoint:", err)
		os.Exit(1)
	}
}
