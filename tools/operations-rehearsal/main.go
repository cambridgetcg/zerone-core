// operations-rehearsal compiles and verifies canonical, hash-bound evidence
// indexes for Zerone upgrade, emergency, and fresh-volume rehearsals. It
// deliberately uses only the Go standard library.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		printUsage(stderr)
		return 2
	}
	switch args[0] {
	case "seal-evidence":
		return runSealEvidence(args[1:], stdout, stderr)
	case "compile":
		return runCompile(args[1:], stdout, stderr)
	case "verify":
		return runVerify(args[1:], stdout, stderr)
	case "digest":
		return runDigest(args[1:], stdout, stderr)
	case "fault-matrix":
		return runFaultMatrix(args[1:], stdout, stderr)
	case "help", "-h", "--help":
		printUsage(stdout)
		return 0
	default:
		fmt.Fprintf(stderr, "operations-rehearsal: unknown command %q\n", args[0])
		printUsage(stderr)
		return 2
	}
}

func printUsage(output io.Writer) {
	fmt.Fprintln(output, "Usage:")
	fmt.Fprintln(output, "  operations-rehearsal seal-evidence --draft <envelope-draft.json> --evidence-root <dir> --out <envelope.json>")
	fmt.Fprintln(output, "  operations-rehearsal digest --evidence-root <dir> --path <relative> --kind <kind> --media-type <type>")
	fmt.Fprintln(output, "  operations-rehearsal fault-matrix")
	fmt.Fprintln(output, "  operations-rehearsal compile --draft <draft.json> --evidence-root <dir> --out <report.json>")
	fmt.Fprintln(output, "  operations-rehearsal verify --report <report.json> --evidence-root <dir>")
	fmt.Fprintln(output)
	fmt.Fprintln(output, "Compile verifies a typed evidence index. It never claims that offline evidence proves execution.")
}

func runSealEvidence(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("operations-rehearsal seal-evidence", flag.ContinueOnError)
	flags.SetOutput(stderr)
	draftPath := flags.String("draft", "", "typed evidence-envelope draft with empty envelope_sha256")
	evidenceRootPath := flags.String("evidence-root", "", "root containing referenced raw execution artifacts")
	output := flags.String("out", "", "canonical sealed envelope output path, or - for stdout")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if flags.NArg() != 0 {
		fmt.Fprintln(stderr, "seal-evidence: positional arguments are not accepted")
		return 2
	}
	draft, err := readControlFile(*draftPath)
	if err != nil {
		fmt.Fprintf(stderr, "seal-evidence: %v\n", err)
		return 2
	}
	root, err := secureEvidenceRoot(*evidenceRootPath)
	if err != nil {
		fmt.Fprintf(stderr, "seal-evidence: %v\n", err)
		return 2
	}
	envelope, document, err := sealEvidenceEnvelope(draft, root)
	if err != nil {
		fmt.Fprintf(stderr, "seal-evidence: INVALID_EVIDENCE: %v\n", err)
		return 1
	}
	if err := ensureEnvelopeOutputDoesNotReplaceArtifact(*output, root, envelope); err != nil {
		fmt.Fprintf(stderr, "seal-evidence: %v\n", err)
		return 1
	}
	if err := writeAtomic(*output, document, stdout); err != nil {
		fmt.Fprintf(stderr, "seal-evidence: %v\n", err)
		return 1
	}
	if *output != "-" {
		fmt.Fprintf(
			stdout,
			"SEALED_SELF_ATTESTED_EVIDENCE kind=%s envelope_sha256=%s\n",
			envelope.Kind,
			envelope.EnvelopeSHA256,
		)
	}
	return 0
}

func runCompile(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("operations-rehearsal compile", flag.ContinueOnError)
	flags.SetOutput(stderr)
	draftPath := flags.String("draft", "", "strict JSON draft with empty computed digest fields")
	evidenceRootPath := flags.String("evidence-root", "", "root containing every referenced artifact")
	output := flags.String("out", "", "canonical report output path, or - for stdout")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if flags.NArg() != 0 {
		fmt.Fprintln(stderr, "compile: positional arguments are not accepted")
		return 2
	}
	draft, err := readControlFile(*draftPath)
	if err != nil {
		fmt.Fprintf(stderr, "compile: %v\n", err)
		return 2
	}
	evidenceRoot, err := secureEvidenceRoot(*evidenceRootPath)
	if err != nil {
		fmt.Fprintf(stderr, "compile: %v\n", err)
		return 2
	}
	report, document, err := compileReport(draft, evidenceRoot)
	if err != nil {
		fmt.Fprintf(stderr, "compile: INVALID_EVIDENCE_INDEX: %v\n", err)
		return 1
	}
	if err := ensureOutputDoesNotReplaceEvidence(*output, evidenceRoot, report); err != nil {
		fmt.Fprintf(stderr, "compile: %v\n", err)
		return 1
	}
	if err := writeAtomic(*output, document, stdout); err != nil {
		fmt.Fprintf(stderr, "compile: %v\n", err)
		return 1
	}
	if *output != "-" {
		fmt.Fprintln(stdout, reportSummary(report))
	}
	return 0
}

func runVerify(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("operations-rehearsal verify", flag.ContinueOnError)
	flags.SetOutput(stderr)
	reportPath := flags.String("report", "", "canonical compiled report")
	evidenceRootPath := flags.String("evidence-root", "", "root containing every referenced artifact")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if flags.NArg() != 0 {
		fmt.Fprintln(stderr, "verify: positional arguments are not accepted")
		return 2
	}
	document, err := readControlFile(*reportPath)
	if err != nil {
		fmt.Fprintf(stderr, "verify: %v\n", err)
		return 2
	}
	evidenceRoot, err := secureEvidenceRoot(*evidenceRootPath)
	if err != nil {
		fmt.Fprintf(stderr, "verify: %v\n", err)
		return 2
	}
	report, err := verifyReport(document, evidenceRoot)
	if err != nil {
		fmt.Fprintf(stderr, "verify: INVALID_EVIDENCE_INDEX: %v\n", err)
		return 1
	}
	fmt.Fprintln(stdout, reportSummary(report))
	return 0
}

func runDigest(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("operations-rehearsal digest", flag.ContinueOnError)
	flags.SetOutput(stderr)
	evidenceRootPath := flags.String("evidence-root", "", "artifact root")
	path := flags.String("path", "", "artifact path relative to the evidence root")
	kind := flags.String("kind", "", "stable evidence kind")
	mediaType := flags.String("media-type", "", "application/json, text/plain, or application/octet-stream")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if flags.NArg() != 0 {
		fmt.Fprintln(stderr, "digest: positional arguments are not accepted")
		return 2
	}
	root, err := secureEvidenceRoot(*evidenceRootPath)
	if err != nil {
		fmt.Fprintf(stderr, "digest: %v\n", err)
		return 2
	}
	reference, err := makeEvidenceRef(root, *path, *kind, *mediaType)
	if err != nil {
		fmt.Fprintf(stderr, "digest: INVALID: %v\n", err)
		return 1
	}
	document, err := canonicalDocument(reference)
	if err != nil {
		fmt.Fprintf(stderr, "digest: %v\n", err)
		return 1
	}
	if _, err := stdout.Write(document); err != nil {
		fmt.Fprintf(stderr, "digest: %v\n", err)
		return 1
	}
	return 0
}

func runFaultMatrix(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("operations-rehearsal fault-matrix", flag.ContinueOnError)
	flags.SetOutput(stderr)
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if flags.NArg() != 0 {
		fmt.Fprintln(stderr, "fault-matrix: positional arguments are not accepted")
		return 2
	}
	matrix, err := buildFaultMatrix()
	if err != nil {
		fmt.Fprintf(stderr, "fault-matrix: %v\n", err)
		return 1
	}
	document, err := canonicalDocument(matrix)
	if err != nil {
		fmt.Fprintf(stderr, "fault-matrix: %v\n", err)
		return 1
	}
	if _, err := stdout.Write(document); err != nil {
		fmt.Fprintf(stderr, "fault-matrix: %v\n", err)
		return 1
	}
	return 0
}

func reportSummary(report Report) string {
	return fmt.Sprintf(
		"VERIFIED_EVIDENCE_INDEX provenance=self_attested external_controls=unverified release_decision=none schema=%s run_id=%s mode=%s chain_id=%s upgrade_height=%d faults=%d evidence_sha256=%s report_sha256=%s",
		report.Schema,
		report.RunID,
		report.Mode,
		report.ChainID,
		report.Upgrade.UpgradeHeight,
		len(report.Faults),
		report.EvidenceManifestSHA256,
		report.ReportSHA256,
	)
}
