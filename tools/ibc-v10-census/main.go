// ibc-v10-census performs a read-only, offline preflight of the IBC-Go v8
// state that must be understood before Zerone removes ICS-29 and upgrades to
// IBC-Go v10.
package main

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

const maxInputBytes int64 = 128 << 20

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("ibc-v10-census", flag.ContinueOnError)
	flags.SetOutput(stderr)
	input := flags.String("input", "-", "full zeroned export/app_state JSON path, or - for stdin")
	heightText := flags.String("export-height", "", "trusted export height (required, positive uint64)")
	appHashText := flags.String("app-hash", "", "trusted 32-byte app hash as 64 hexadecimal characters (required)")
	format := flags.String("format", "text", "output format: text or json")
	failOn := flags.String("fail-on", "error", "exit 1 on: error, warning, or never")
	flags.Usage = func() {
		fmt.Fprintln(
			stderr,
			"Usage: go run ./tools/ibc-v10-census --input <export.json> --export-height <height> --app-hash <64-hex> [--format text|json] [--fail-on error|warning|never]",
		)
		flags.PrintDefaults()
	}
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if flags.NArg() != 0 {
		fmt.Fprintln(stderr, "ibc-v10-census: positional arguments are not accepted")
		flags.Usage()
		return 2
	}
	if *format != "text" && *format != "json" {
		fmt.Fprintf(stderr, "ibc-v10-census: invalid --format %q\n", *format)
		return 2
	}
	if *failOn != severityError && *failOn != severityWarning && *failOn != "never" {
		fmt.Fprintf(stderr, "ibc-v10-census: invalid --fail-on %q\n", *failOn)
		return 2
	}

	height, err := parseExportHeight(*heightText)
	if err != nil {
		fmt.Fprintf(stderr, "ibc-v10-census: %v\n", err)
		return 2
	}
	appHash, err := parseAppHash(*appHashText)
	if err != nil {
		fmt.Fprintf(stderr, "ibc-v10-census: %v\n", err)
		return 2
	}

	data, source, err := readInput(*input, stdin)
	if err != nil {
		fmt.Fprintf(stderr, "ibc-v10-census: %v\n", err)
		return 2
	}
	report, err := auditDocument(data, source, Evidence{
		ExportHeight: strconv.FormatUint(height, 10),
		AppHash:      appHash,
	})
	if err != nil {
		fmt.Fprintf(stderr, "ibc-v10-census: %v\n", err)
		return 2
	}

	switch *format {
	case "json":
		encoder := json.NewEncoder(stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(report); err != nil {
			fmt.Fprintf(stderr, "ibc-v10-census: encode report: %v\n", err)
			return 2
		}
	default:
		if err := printText(stdout, report); err != nil {
			fmt.Fprintf(stderr, "ibc-v10-census: write report: %v\n", err)
			return 2
		}
	}

	switch *failOn {
	case severityError:
		if report.Summary.Errors > 0 {
			return 1
		}
	case severityWarning:
		if report.Summary.Errors > 0 || report.Summary.Warnings > 0 {
			return 1
		}
	}
	return 0
}

func parseExportHeight(value string) (uint64, error) {
	if value == "" {
		return 0, errors.New("--export-height is required")
	}
	if value != strings.TrimSpace(value) || strings.HasPrefix(value, "+") {
		return 0, errors.New("--export-height must be a canonical positive uint64")
	}
	height, err := strconv.ParseUint(value, 10, 64)
	if err != nil || height == 0 || strconv.FormatUint(height, 10) != value {
		return 0, errors.New("--export-height must be a canonical positive uint64")
	}
	return height, nil
}

func parseAppHash(value string) (string, error) {
	if len(value) != 64 {
		return "", errors.New("--app-hash must contain exactly 64 hexadecimal characters")
	}
	if _, err := hex.DecodeString(value); err != nil {
		return "", errors.New("--app-hash must contain exactly 64 hexadecimal characters")
	}
	return strings.ToUpper(value), nil
}

func readInput(path string, stdin io.Reader) ([]byte, string, error) {
	if path == "-" {
		data, err := readBounded(stdin)
		if err != nil {
			return nil, "", fmt.Errorf("read stdin: %w", err)
		}
		return data, "stdin", nil
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, "", fmt.Errorf("open %s: %w", path, err)
	}
	defer file.Close()
	data, err := readBounded(file)
	if err != nil {
		return nil, "", fmt.Errorf("read %s: %w", path, err)
	}
	return data, path, nil
}

func readBounded(reader io.Reader) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(reader, maxInputBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > maxInputBytes {
		return nil, fmt.Errorf("input exceeds the %d-byte safety limit", maxInputBytes)
	}
	return data, nil
}
