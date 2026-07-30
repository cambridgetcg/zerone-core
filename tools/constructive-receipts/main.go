// Command constructive-receipts creates an offline, zero-value candidate or
// refusal receipt from one exact constructive-intelligence tree and locally
// re-evaluated PoCA profile/evidence inputs.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/zerone-chain/zerone/tools/constructive-receipts/bridge"
	poca "github.com/zerone-chain/zerone/tools/poca-shadow/evaluate"
)

const maxInputBytes = 1 << 20

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "constructive-receipts: %v\n", err)
		os.Exit(1)
	}
}

func run(arguments []string, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("constructive-receipts", flag.ContinueOnError)
	flags.SetOutput(stderr)
	var requestPath string
	var treePath string
	var profilePath string
	var evidencePath string
	flags.StringVar(&requestPath, "request", "", "path to a local ConstructiveReceiptRequest v0 JSON file (required)")
	flags.StringVar(&treePath, "tree", "", "path to the exact local constructive-intelligence tree v1 JSON file (required)")
	flags.StringVar(&profilePath, "profile", "", "path to a local PoCA StandardProfile v0 JSON file (required)")
	flags.StringVar(&evidencePath, "evidence", "", "path to a local PoCA EvidenceBundle v0 JSON file (required)")
	if err := flags.Parse(arguments); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return errors.New("positional arguments are not accepted; use named flags")
	}
	if requestPath == "" || treePath == "" || profilePath == "" || evidencePath == "" {
		return errors.New("--request, --tree, --profile, and --evidence are required")
	}

	requestBytes, err := readBoundedRegularFile(requestPath)
	if err != nil {
		return fmt.Errorf("read request: %w", err)
	}
	treeBytes, err := readBoundedRegularFile(treePath)
	if err != nil {
		return fmt.Errorf("read tree: %w", err)
	}
	profileBytes, err := readBoundedRegularFile(profilePath)
	if err != nil {
		return fmt.Errorf("read profile: %w", err)
	}
	evidenceBytes, err := readBoundedRegularFile(evidencePath)
	if err != nil {
		return fmt.Errorf("read evidence: %w", err)
	}

	request, err := bridge.ParseRequest(requestBytes)
	if err != nil {
		return err
	}
	profile, err := poca.ParseProfile(profileBytes)
	if err != nil {
		return err
	}
	evidence, err := poca.ParseEvidence(evidenceBytes)
	if err != nil {
		return err
	}
	receipt, err := bridge.Evaluate(request, treeBytes, profile, evidence)
	if err != nil {
		return err
	}

	encoder := json.NewEncoder(stdout)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(receipt); err != nil {
		return fmt.Errorf("encode output: %w", err)
	}
	return nil
}

func readBoundedRegularFile(path string) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("%s is not a regular file", path)
	}
	if info.Size() > maxInputBytes {
		return nil, fmt.Errorf("%s exceeds %d-byte limit", path, maxInputBytes)
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	data, err := io.ReadAll(io.LimitReader(file, maxInputBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxInputBytes {
		return nil, fmt.Errorf("%s exceeds %d-byte limit", path, maxInputBytes)
	}
	return data, nil
}
