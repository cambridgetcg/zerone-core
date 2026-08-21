// Command zerone-supabase-observatory verifies the source-only
// zerone-agenttool-supabase-observatory/0.1 contract and its closed fixtures.
// It has no network, database, RPC, chain, wallet, or signing dependency.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
)

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "zerone-supabase-observatory: %v\n", err)
		os.Exit(1)
	}
}

func run(arguments []string, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("zerone-supabase-observatory", flag.ContinueOnError)
	flags.SetOutput(stderr)
	var repositoryRoot string
	flags.StringVar(&repositoryRoot, "repository-root", "", "path to the exact local Zerone repository root (required)")
	if err := flags.Parse(arguments); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return errors.New("positional arguments are not accepted; use --repository-root")
	}
	if repositoryRoot == "" {
		return errors.New("--repository-root is required")
	}
	report, err := verifyRepository(repositoryRoot)
	if err != nil {
		return err
	}
	encoder := json.NewEncoder(stdout)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(report); err != nil {
		return fmt.Errorf("encode verification report: %w", err)
	}
	return nil
}
