// Command poca-shadow evaluates a local Proof of Constructive Adaptation v0
// profile and evidence bundle. It is an offline, read-only, zero-reward
// projection: it performs no network access, chain query, transaction, or
// signature verification.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/zerone-chain/zerone/tools/poca-shadow/evaluate"
)

const maxInputBytes = 1 << 20

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "poca-shadow: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("poca-shadow", flag.ContinueOnError)
	flags.SetOutput(stderr)
	var profilePath string
	var evidencePath string
	var requireCrown bool
	var outputFormat string
	var expectedProfileDigest string
	flags.StringVar(&profilePath, "profile", "", "path to a local StandardProfile v0 JSON file (required)")
	flags.StringVar(&evidencePath, "evidence", "", "path to a local EvidenceBundle v0 JSON file (required)")
	flags.BoolVar(&requireCrown, "require-crown", false, "fail unless the declared shadow evidence unlocks the crown")
	flags.StringVar(&outputFormat, "format", "certificate", "output format: certificate or in-toto")
	flags.StringVar(&expectedProfileDigest, "expect-profile-digest", "", "required canonical profile digest for a crown gate")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return errors.New("positional arguments are not accepted; use named flags")
	}
	if profilePath == "" || evidencePath == "" {
		return errors.New("--profile and --evidence are required")
	}
	if outputFormat != "certificate" && outputFormat != "in-toto" {
		return errors.New("--format must be certificate or in-toto")
	}
	if requireCrown && expectedProfileDigest == "" {
		return errors.New("--require-crown also requires --expect-profile-digest")
	}

	profileBytes, err := readBoundedRegularFile(profilePath)
	if err != nil {
		return fmt.Errorf("read profile: %w", err)
	}
	evidenceBytes, err := readBoundedRegularFile(evidencePath)
	if err != nil {
		return fmt.Errorf("read evidence: %w", err)
	}
	profile, err := evaluate.ParseProfile(profileBytes)
	if err != nil {
		return err
	}
	evidence, err := evaluate.ParseEvidence(evidenceBytes)
	if err != nil {
		return err
	}
	certificate, err := evaluate.Evaluate(profile, evidence)
	if err != nil {
		return err
	}
	if expectedProfileDigest != "" && certificate.Profile.Digest != expectedProfileDigest {
		return fmt.Errorf(
			"profile digest mismatch: expected %s, got %s",
			expectedProfileDigest,
			certificate.Profile.Digest,
		)
	}
	if requireCrown && certificate.CrownStatus != "DECLARED_PASS" {
		return fmt.Errorf("crown %q is %s", profile.CrownNodeID, certificate.CrownStatus)
	}

	var output any = certificate
	if outputFormat == "in-toto" {
		statement, err := evaluate.EvaluateInToto(profile, evidence)
		if err != nil {
			return err
		}
		output = statement
	}

	encoder := json.NewEncoder(stdout)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(output); err != nil {
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
