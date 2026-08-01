// validator-home-manifest creates and verifies byte-exact, self-hashed
// manifests for cleanly stopped Zerone validator homes. It uses only the Go
// standard library and never emits private key or signing-state contents.
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
	case "capture-stop":
		return runCaptureStop(args[1:], stdout, stderr)
	case "create":
		return runCreate(args[1:], stdout, stderr)
	case "verify":
		return runVerify(args[1:], stdout, stderr)
	case "help", "-h", "--help":
		printUsage(stdout)
		return 0
	default:
		fmt.Fprintf(stderr, "validator-home-manifest: unknown command %q\n", args[0])
		printUsage(stderr)
		return 2
	}
}

func printUsage(output io.Writer) {
	fmt.Fprintln(output, "Usage:")
	fmt.Fprintln(output, "  validator-home-manifest capture-stop --home <path> --pid <n> --process-start-time <UTC> --process-identity-sha256 <hex> --restart-inhibit-evidence <json> --last-height <n> --app-hash <64-hex> --method <text> --observer <text> --out <evidence.json>")
	fmt.Fprintln(output, "  validator-home-manifest create --home <read-only-snapshot> --destination-home <empty-mounted-path> --stopped-evidence <json> --restart-inhibit-evidence <json> --snapshot-evidence <json> --volume-evidence <json> --destination-volume-id <id> --out <manifest.json>")
	fmt.Fprintln(output, "  validator-home-manifest verify --home <destination-path> --manifest <manifest.json> --restart-inhibit-evidence <json> --snapshot-evidence <json> --volume-evidence <json> --destination-volume-id <id>")
	fmt.Fprintln(output)
	fmt.Fprintln(output, "Control documents are canonical single-line JSON with a trailing newline.")
}

func runCaptureStop(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("validator-home-manifest capture-stop", flag.ContinueOnError)
	flags.SetOutput(stderr)
	home := flags.String("home", "", "cleanly stopped validator home")
	processID := flags.Int("pid", 0, "PID of the validator process that must be absent")
	processStartTime := flags.String("process-start-time", "", "externally captured canonical UTC process start time")
	processIdentitySHA256 := flags.String("process-identity-sha256", "", "external digest of PID/start-time/executable identity")
	restartInhibitEvidence := flags.String("restart-inhibit-evidence", "", "canonical external supervisor restart-inhibit evidence")
	lastHeight := flags.Int64("last-height", 0, "trusted last committed application height")
	appHash := flags.String("app-hash", "", "trusted last application hash (64 hex)")
	method := flags.String("method", "", "stop-observation method")
	observer := flags.String("observer", "", "independent observer identity")
	output := flags.String("out", "", "output evidence path, or - for stdout")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if flags.NArg() != 0 {
		fmt.Fprintln(stderr, "capture-stop: positional arguments are not accepted")
		return 2
	}
	if err := evidenceOutputOutsideHome(*home, *output); err != nil {
		fmt.Fprintf(stderr, "capture-stop: %v\n", err)
		return 2
	}
	evidence, err := createStoppedEvidence(
		*home,
		*processID,
		*processStartTime,
		*processIdentitySHA256,
		*restartInhibitEvidence,
		*lastHeight,
		*appHash,
		*method,
		*observer,
	)
	if err != nil {
		fmt.Fprintf(stderr, "capture-stop: %v\n", err)
		return 1
	}
	document, err := canonicalDocument(evidence)
	if err != nil {
		fmt.Fprintf(stderr, "capture-stop: %v\n", err)
		return 1
	}
	if err := writeAtomic(*output, document, stdout); err != nil {
		fmt.Fprintf(stderr, "capture-stop: %v\n", err)
		return 1
	}
	if *output != "-" {
		fmt.Fprintln(stdout, stoppedEvidenceSummary(evidence))
	}
	return 0
}

func runCreate(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("validator-home-manifest create", flag.ContinueOnError)
	flags.SetOutput(stderr)
	home := flags.String("home", "", "read-only isolated snapshot of the stopped source home")
	destinationHome := flags.String("destination-home", "", "empty mounted destination home on a distinct filesystem device")
	stoppedEvidence := flags.String("stopped-evidence", "", "canonical stopped-process evidence")
	restartInhibitEvidence := flags.String("restart-inhibit-evidence", "", "canonical external restart-inhibit evidence")
	snapshotEvidence := flags.String("snapshot-evidence", "", "canonical external read-only snapshot evidence")
	volumeEvidence := flags.String("volume-evidence", "", "canonical external volume-control evidence")
	destinationVolumeID := flags.String("destination-volume-id", "", "exact destination volume identifier")
	output := flags.String("out", "", "output manifest path, or - for stdout")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if flags.NArg() != 0 {
		fmt.Fprintln(stderr, "create: positional arguments are not accepted")
		return 2
	}
	secureHome, err := secureAbsoluteDirectory(*home)
	if err != nil {
		fmt.Fprintf(stderr, "create: %v\n", err)
		return 2
	}
	if err := ensureOutsideHome(secureHome, *output); err != nil {
		fmt.Fprintf(stderr, "create: %v\n", err)
		return 2
	}
	secureDestination, err := secureAbsoluteDirectory(*destinationHome)
	if err != nil {
		fmt.Fprintf(stderr, "create: %v\n", err)
		return 2
	}
	if err := ensureOutsideHome(secureDestination, *output); err != nil {
		fmt.Fprintf(stderr, "create: %v\n", err)
		return 2
	}
	manifest, err := createManifest(secureHome, CreateManifestOptions{
		DestinationHome:            secureDestination,
		DestinationVolumeID:        *destinationVolumeID,
		StoppedEvidencePath:        *stoppedEvidence,
		RestartInhibitEvidencePath: *restartInhibitEvidence,
		SnapshotEvidencePath:       *snapshotEvidence,
		VolumeEvidencePath:         *volumeEvidence,
	})
	if err != nil {
		fmt.Fprintf(stderr, "create: %v\n", err)
		return 1
	}
	document, err := canonicalDocument(manifest)
	if err != nil {
		fmt.Fprintf(stderr, "create: %v\n", err)
		return 1
	}
	if err := writeAtomic(*output, document, stdout); err != nil {
		fmt.Fprintf(stderr, "create: %v\n", err)
		return 1
	}
	if *output != "-" {
		fmt.Fprintln(stdout, manifestSummary(manifest))
	}
	return 0
}

func runVerify(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("validator-home-manifest verify", flag.ContinueOnError)
	flags.SetOutput(stderr)
	home := flags.String("home", "", "restored destination validator home")
	manifestPath := flags.String("manifest", "", "canonical validator-home manifest")
	destinationVolumeID := flags.String("destination-volume-id", "", "externally observed destination volume identifier")
	restartInhibitEvidence := flags.String("restart-inhibit-evidence", "", "canonical external restart-inhibit evidence")
	snapshotEvidence := flags.String("snapshot-evidence", "", "canonical external read-only snapshot evidence")
	volumeEvidence := flags.String("volume-evidence", "", "canonical external volume-control evidence")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if flags.NArg() != 0 {
		fmt.Fprintln(stderr, "verify: positional arguments are not accepted")
		return 2
	}
	data, err := loadManifestFile(*manifestPath)
	if err != nil {
		fmt.Fprintf(stderr, "verify: %v\n", err)
		return 2
	}
	manifest, err := verifyManifest(data, *home, VerifyManifestOptions{
		DestinationVolumeID:        *destinationVolumeID,
		RestartInhibitEvidencePath: *restartInhibitEvidence,
		SnapshotEvidencePath:       *snapshotEvidence,
		VolumeEvidencePath:         *volumeEvidence,
	}, true)
	if err != nil {
		fmt.Fprintf(stderr, "verify: INVALID_LOCAL_MANIFEST: %v\n", err)
		return 1
	}
	fmt.Fprintln(stdout, manifestSummary(manifest))
	return 0
}
