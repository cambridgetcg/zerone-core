// identity-census performs a read-only source/deployed-boundary audit of
// Zerone identity records in an exported genesis/app-state JSON document.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("identity-census", flag.ContinueOnError)
	flags.SetOutput(stderr)
	input := flags.String("input", "-", "exported genesis/app-state JSON path, or - for stdin")
	format := flags.String("format", "text", "output format: text or json")
	failOn := flags.String("fail-on", "error", "exit 1 on: error, warning, or never")
	flags.Usage = func() {
		fmt.Fprintln(stderr, "Usage: go run ./tools/identity-census --input <export.json> [--format text|json] [--fail-on error|warning|never]")
		flags.PrintDefaults()
	}
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if flags.NArg() != 0 {
		fmt.Fprintln(stderr, "identity-census: positional arguments are not accepted")
		flags.Usage()
		return 2
	}
	if *format != "text" && *format != "json" {
		fmt.Fprintf(stderr, "identity-census: invalid --format %q\n", *format)
		return 2
	}
	if *failOn != "error" && *failOn != "warning" && *failOn != "never" {
		fmt.Fprintf(stderr, "identity-census: invalid --fail-on %q\n", *failOn)
		return 2
	}

	data, source, err := readInput(*input, stdin)
	if err != nil {
		fmt.Fprintf(stderr, "identity-census: %v\n", err)
		return 2
	}
	report, err := auditDocument(data, source)
	if err != nil {
		fmt.Fprintf(stderr, "identity-census: %v\n", err)
		return 2
	}

	switch *format {
	case "json":
		encoder := json.NewEncoder(stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(report); err != nil {
			fmt.Fprintf(stderr, "identity-census: encode report: %v\n", err)
			return 2
		}
	default:
		printText(stdout, report)
	}

	switch *failOn {
	case "error":
		if report.Summary.Errors > 0 {
			return 1
		}
	case "warning":
		if report.Summary.Errors > 0 || report.Summary.Warnings > 0 {
			return 1
		}
	}
	return 0
}

func readInput(path string, stdin io.Reader) ([]byte, string, error) {
	if path == "-" {
		data, err := io.ReadAll(stdin)
		if err != nil {
			return nil, "", fmt.Errorf("read stdin: %w", err)
		}
		return data, "stdin", nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, "", fmt.Errorf("read %s: %w", path, err)
	}
	return data, path, nil
}

func printText(output io.Writer, report Report) {
	fmt.Fprintf(output, "Zerone identity census: %s\n", report.Source)
	fmt.Fprintf(output, "Input: %s; complete snapshot: %t\n", report.Coverage.InputKind, report.Coverage.CompleteSnapshot)
	chainID := report.Coverage.ChainID
	if chainID == "" {
		chainID = "unavailable"
	}
	fmt.Fprintf(output, "Chain/profile: %s; %s\n", chainID, report.Coverage.ValidationProfile)
	fmt.Fprintf(
		output,
		"Records: %d zerone accounts, %d DID mappings, %d last-key rotations, %d Cosmos BaseAccounts\n",
		report.Summary.ZeroneAccounts,
		report.Summary.DIDMappings,
		report.Summary.KeyRotations,
		report.Summary.CosmosAccounts,
	)
	fmt.Fprintf(output, "Rotation coverage: %s\n", report.Coverage.RotationState)
	for _, finding := range report.Findings {
		context := make([]string, 0, 3)
		if finding.Address != "" {
			context = append(context, "address="+finding.Address)
		}
		if finding.DID != "" {
			context = append(context, "did="+finding.DID)
		}
		if finding.Location != "" {
			context = append(context, "at="+finding.Location)
		}
		suffix := ""
		if len(context) > 0 {
			suffix = " (" + strings.Join(context, ", ") + ")"
		}
		fmt.Fprintf(output, "%s %s: %s%s\n", strings.ToUpper(finding.Severity), finding.Code, finding.Message, suffix)
		if len(finding.Related) > 0 {
			fmt.Fprintf(output, "  related: %s\n", strings.Join(finding.Related, ", "))
		}
	}
	fmt.Fprintf(output, "Result: %d errors, %d warnings\n", report.Summary.Errors, report.Summary.Warnings)
}
