// zerone-ops verifies canonical, append-only operations transition journals.
// It is deliberately offline and uses only the Go standard library.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

const maxTransitionBytes = 1 << 20

type powerSnapshotPins map[uint64]string

func (pins powerSnapshotPins) String() string {
	return ""
}

func (pins powerSnapshotPins) Set(value string) error {
	sequenceText, digest, found := strings.Cut(value, "=")
	if !found {
		return fmt.Errorf("power snapshot pin must use <sequence>=<sha256>")
	}
	sequence, err := strconv.ParseUint(sequenceText, 10, 64)
	if err != nil || sequence == 0 {
		return fmt.Errorf("power snapshot pin sequence %q must be a positive integer", sequenceText)
	}
	if err := validateSHA256("power snapshot pin", digest, false); err != nil {
		return err
	}
	if existing, duplicate := pins[sequence]; duplicate {
		return fmt.Errorf(
			"power snapshot sequence %d was pinned more than once (%s and %s)",
			sequence,
			existing,
			digest,
		)
	}
	pins[sequence] = digest
	return nil
}

type repeatedPaths []string

func (paths *repeatedPaths) String() string {
	return strings.Join(*paths, ",")
}

func (paths *repeatedPaths) Set(value string) error {
	if value == "" || strings.TrimSpace(value) != value {
		return fmt.Errorf("path must be non-empty and trimmed")
	}
	*paths = append(*paths, value)
	return nil
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		printUsage(stderr)
		return 2
	}
	switch args[0] {
	case "verify":
		return runVerify(args[1:], stdin, stdout, stderr)
	case "seal":
		return runSeal(args[1:], stdin, stdout, stderr)
	case "approval-statement":
		return runApprovalStatement(args[1:], stdin, stdout, stderr)
	case "verify-supersession":
		return runVerifySupersession(args[1:], stdin, stdout, stderr)
	case "verify-replacement":
		return runVerifyReplacement(args[1:], stdin, stdout, stderr)
	case "seal-supersession":
		return runSealSupersession(args[1:], stdin, stdout, stderr)
	case "supersession-approval-statement":
		return runSupersessionApprovalStatement(args[1:], stdin, stdout, stderr)
	case "help", "-h", "--help":
		printUsage(stdout)
		return 0
	default:
		fmt.Fprintf(stderr, "zerone-ops: unknown command %q\n", args[0])
		printUsage(stderr)
		return 2
	}
}

func printUsage(output io.Writer) {
	fmt.Fprintln(output, "Usage:")
	fmt.Fprintln(output, "  zerone-ops verify --trust-policy <policy.json> --trust-policy-sha256 <hex> --head-sha256 <hex> [flags] <transition-1.json> [transition-2.json ...]")
	fmt.Fprintln(output, "  zerone-ops approval-statement --trust-policy <policy.json> --trust-policy-sha256 <hex> --input <draft.json>")
	fmt.Fprintln(output, "  zerone-ops seal --trust-policy <policy.json> --trust-policy-sha256 <hex> --input <approved-draft.json>")
	fmt.Fprintln(output, "  zerone-ops verify-supersession --old-trust-policy <old.json> --old-trust-policy-sha256 <hex> --new-trust-policy <new.json> --new-trust-policy-sha256 <hex> --old-head-sha256 <hex> --input <sealed.json>")
	fmt.Fprintln(output, "  zerone-ops verify-replacement [supersession policy flags] --old-head-sha256 <hex> --new-head-sha256 <hex> --old-journal <path> --input <sidecar.json> --new-journal <path>")
	fmt.Fprintln(output, "  zerone-ops supersession-approval-statement [supersession policy flags] --input <draft.json> --role <role> --identity <id> --public-key <hex>")
	fmt.Fprintln(output, "  zerone-ops seal-supersession [supersession policy flags] --input <approved-draft.json>")
	fmt.Fprintln(output)
	fmt.Fprintln(output, "All commands are offline. `verify` requires exact canonical JSON bytes.")
}

func runVerify(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("zerone-ops verify", flag.ContinueOnError)
	flags.SetOutput(stderr)
	chainID := flags.String("chain-id", "", "expected chain ID from a trusted channel")
	incidentID := flags.String("incident-id", "", "expected incident ID from a trusted channel")
	releaseID := flags.String("release-id", "", "expected release ID from a trusted channel")
	binarySHA256 := flags.String("binary-sha256", "", "expected release binary SHA-256")
	headSHA256 := flags.String("head-sha256", "", "expected journal head SHA-256")
	trustPolicyPath := flags.String("trust-policy", "", "canonical trust policy path (required)")
	trustPolicySHA256 := flags.String("trust-policy-sha256", "", "expected trust policy SHA-256 from a separate trusted channel (required)")
	snapshotPins := make(powerSnapshotPins)
	flags.Var(
		snapshotPins,
		"power-snapshot-pin",
		"externally obtained <transition-sequence>=<snapshot-sha256>; repeat for every power-gated transition",
	)
	flags.Usage = func() {
		fmt.Fprintln(stderr, "Usage: zerone-ops verify --trust-policy <policy.json> --trust-policy-sha256 <hex> --head-sha256 <hex> [flags] <transition-1.json> [transition-2.json ...]")
		flags.PrintDefaults()
	}
	if err := flags.Parse(args); err != nil {
		return 2
	}
	paths := flags.Args()
	if len(paths) == 0 {
		fmt.Fprintln(stderr, "zerone-ops verify: at least one transition path is required")
		flags.Usage()
		return 2
	}
	if len(paths) > 1 {
		for _, path := range paths {
			if path == "-" {
				fmt.Fprintln(stderr, "zerone-ops verify: stdin (-) can only be used for a single transition")
				return 2
			}
		}
	}
	trustPolicy, pinnedTrustPolicySHA256, err := loadPinnedTrustPolicy(
		*trustPolicyPath,
		*trustPolicySHA256,
	)
	if err != nil {
		fmt.Fprintf(stderr, "zerone-ops verify: %v\n", err)
		return 2
	}
	if err := validateSHA256(
		"externally pinned journal head SHA-256",
		*headSHA256,
		false,
	); err != nil {
		fmt.Fprintf(
			stderr,
			"zerone-ops verify: --head-sha256 is required and must be exact: %v\n",
			err,
		)
		return 2
	}

	documents := make([][]byte, 0, len(paths))
	for _, path := range paths {
		document, err := readTransition(path, stdin)
		if err != nil {
			fmt.Fprintf(stderr, "zerone-ops verify: %v\n", err)
			return 2
		}
		documents = append(documents, document)
	}
	result, err := verifyDocuments(documents, VerifyOptions{
		ExpectedChainID:             *chainID,
		ExpectedIncidentID:          *incidentID,
		ExpectedReleaseID:           *releaseID,
		ExpectedBinarySHA256:        *binarySHA256,
		ExpectedHeadSHA256:          *headSHA256,
		ExpectedPowerSnapshotSHA256: map[uint64]string(snapshotPins),
		TrustPolicy:                 &trustPolicy,
		TrustPolicySHA256:           pinnedTrustPolicySHA256,
	})
	if err != nil {
		fmt.Fprintf(stderr, "zerone-ops verify: INVALID: %v\n", err)
		return 1
	}
	fmt.Fprintf(
		stdout,
		"VALID transitions=%d lane=%s chain_id=%s incident_id=%s release_id=%s state=%s head_sha256=%s\n",
		result.Transitions,
		result.Lane,
		result.ChainID,
		result.IncidentID,
		result.ReleaseID,
		result.State,
		result.HeadSHA256,
	)
	return 0
}

func runApprovalStatement(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("zerone-ops approval-statement", flag.ContinueOnError)
	flags.SetOutput(stderr)
	input := flags.String("input", "-", "draft transition path, or - for stdin")
	role := flags.String("role", "", "approval role")
	identity := flags.String("identity", "", "approval identity")
	publicKey := flags.String("public-key", "", "Ed25519 public key (64 lowercase hex characters)")
	power := flags.String("power", "0", "canonical decimal power for the declared quorum role")
	trustPolicyPath := flags.String("trust-policy", "", "canonical trust policy path (required)")
	trustPolicySHA256 := flags.String("trust-policy-sha256", "", "expected trust policy SHA-256 from a separate trusted channel (required)")
	powerSnapshotSHA256 := flags.String("power-snapshot-sha256", "", "externally obtained power snapshot SHA-256 (required for a power-gated edge)")
	flags.Usage = func() {
		fmt.Fprintln(stderr, "Usage: zerone-ops approval-statement --trust-policy <policy.json> --trust-policy-sha256 <hex> --input <draft.json> --role <role> --identity <id> --public-key <hex> [--power <n>]")
		flags.PrintDefaults()
	}
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if flags.NArg() != 0 {
		fmt.Fprintln(stderr, "zerone-ops approval-statement: positional arguments are not accepted")
		return 2
	}
	trustPolicy, pinnedTrustPolicySHA256, err := loadPinnedTrustPolicy(
		*trustPolicyPath,
		*trustPolicySHA256,
	)
	if err != nil {
		fmt.Fprintf(stderr, "zerone-ops approval-statement: %v\n", err)
		return 2
	}
	document, err := readTransition(*input, stdin)
	if err != nil {
		fmt.Fprintf(stderr, "zerone-ops approval-statement: %v\n", err)
		return 2
	}
	transition, err := decodeTransition(document, false)
	if err != nil {
		fmt.Fprintf(stderr, "zerone-ops approval-statement: %v\n", err)
		return 1
	}
	if transition.Schema != transitionSchema {
		fmt.Fprintf(stderr, "zerone-ops approval-statement: schema must be %q\n", transitionSchema)
		return 1
	}
	if err := validateTransitionBody(transition); err != nil {
		fmt.Fprintf(stderr, "zerone-ops approval-statement: %v\n", err)
		return 1
	}
	approval := Approval{
		Role:      *role,
		Identity:  *identity,
		PublicKey: *publicKey,
		Power:     *power,
	}
	if err := validateLabel("role", approval.Role, 128); err != nil {
		fmt.Fprintf(stderr, "zerone-ops approval-statement: %v\n", err)
		return 1
	}
	if err := validateLabel("identity", approval.Identity, 256); err != nil {
		fmt.Fprintf(stderr, "zerone-ops approval-statement: %v\n", err)
		return 1
	}
	if _, err := decodeExactHex("public-key", approval.PublicKey, 32); err != nil {
		fmt.Fprintf(stderr, "zerone-ops approval-statement: %v\n", err)
		return 1
	}
	if _, err := parseCanonicalDecimal("power", approval.Power); err != nil {
		fmt.Fprintf(stderr, "zerone-ops approval-statement: %v\n", err)
		return 1
	}
	if err := validateTransitionTrustBinding(transition, trustPolicy, pinnedTrustPolicySHA256); err != nil {
		fmt.Fprintf(stderr, "zerone-ops approval-statement: %v\n", err)
		return 1
	}
	if _, err := validateTransitionPowerSnapshotAgainstPolicy(
		transition,
		trustPolicy,
		*powerSnapshotSHA256,
	); err != nil {
		fmt.Fprintf(stderr, "zerone-ops approval-statement: %v\n", err)
		return 1
	}
	if err := validateApprovalAuthorized(approval, trustPolicy); err != nil {
		fmt.Fprintf(stderr, "zerone-ops approval-statement: %v\n", err)
		return 1
	}
	digest, err := approvalStatementDigest(transition, approval)
	if err != nil {
		fmt.Fprintf(stderr, "zerone-ops approval-statement: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "%x\n", digest)
	return 0
}

func runSeal(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("zerone-ops seal", flag.ContinueOnError)
	flags.SetOutput(stderr)
	input := flags.String("input", "-", "approved draft transition path, or - for stdin")
	trustPolicyPath := flags.String("trust-policy", "", "canonical trust policy path (required)")
	trustPolicySHA256 := flags.String("trust-policy-sha256", "", "expected trust policy SHA-256 from a separate trusted channel (required)")
	powerSnapshotSHA256 := flags.String("power-snapshot-sha256", "", "externally obtained power snapshot SHA-256 (required for a power-gated edge)")
	flags.Usage = func() {
		fmt.Fprintln(stderr, "Usage: zerone-ops seal --trust-policy <policy.json> --trust-policy-sha256 <hex> --input <approved-draft.json>")
		flags.PrintDefaults()
	}
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if flags.NArg() != 0 {
		fmt.Fprintln(stderr, "zerone-ops seal: positional arguments are not accepted")
		return 2
	}
	trustPolicy, pinnedTrustPolicySHA256, err := loadPinnedTrustPolicy(
		*trustPolicyPath,
		*trustPolicySHA256,
	)
	if err != nil {
		fmt.Fprintf(stderr, "zerone-ops seal: %v\n", err)
		return 2
	}
	document, err := readTransition(*input, stdin)
	if err != nil {
		fmt.Fprintf(stderr, "zerone-ops seal: %v\n", err)
		return 2
	}
	transition, err := decodeTransition(document, false)
	if err != nil {
		fmt.Fprintf(stderr, "zerone-ops seal: %v\n", err)
		return 1
	}
	if err := validateTransitionAgainstTrustPolicy(
		transition,
		trustPolicy,
		pinnedTrustPolicySHA256,
		*powerSnapshotSHA256,
	); err != nil {
		fmt.Fprintf(stderr, "zerone-ops seal: %v\n", err)
		return 1
	}
	sealed, err := sealTransition(transition)
	if err != nil {
		fmt.Fprintf(stderr, "zerone-ops seal: %v\n", err)
		return 1
	}
	canonical, err := canonicalTransition(sealed)
	if err != nil {
		fmt.Fprintf(stderr, "zerone-ops seal: canonicalize: %v\n", err)
		return 1
	}
	if _, err := stdout.Write(canonical); err != nil {
		fmt.Fprintf(stderr, "zerone-ops seal: write output: %v\n", err)
		return 2
	}
	return 0
}

type supersessionFlagSet struct {
	input                *string
	oldTrustPolicyPath   *string
	oldTrustPolicySHA256 *string
	newTrustPolicyPath   *string
	newTrustPolicySHA256 *string
	oldHeadSHA256        *string
}

func addSupersessionFlags(flags *flag.FlagSet) supersessionFlagSet {
	return supersessionFlagSet{
		input: flags.String(
			"input",
			"-",
			"supersession document path, or - for stdin",
		),
		oldTrustPolicyPath: flags.String(
			"old-trust-policy",
			"",
			"canonical old trust policy path (required)",
		),
		oldTrustPolicySHA256: flags.String(
			"old-trust-policy-sha256",
			"",
			"old trust policy SHA-256 from a separate trusted channel (required)",
		),
		newTrustPolicyPath: flags.String(
			"new-trust-policy",
			"",
			"canonical replacement trust policy path (required)",
		),
		newTrustPolicySHA256: flags.String(
			"new-trust-policy-sha256",
			"",
			"replacement trust policy SHA-256 from a separate trusted channel (required)",
		),
		oldHeadSHA256: flags.String(
			"old-head-sha256",
			"",
			"old journal head SHA-256 from a separate trusted channel (required)",
		),
	}
}

func loadSupersessionPolicies(
	command string,
	config supersessionFlagSet,
) (TrustPolicy, string, TrustPolicy, string, error) {
	oldPolicy, oldPolicySHA256, err := loadPinnedTrustPolicy(
		*config.oldTrustPolicyPath,
		*config.oldTrustPolicySHA256,
	)
	if err != nil {
		return TrustPolicy{}, "", TrustPolicy{}, "", fmt.Errorf(
			"%s old policy: %w",
			command,
			err,
		)
	}
	newPolicy, newPolicySHA256, err := loadPinnedTrustPolicy(
		*config.newTrustPolicyPath,
		*config.newTrustPolicySHA256,
	)
	if err != nil {
		return TrustPolicy{}, "", TrustPolicy{}, "", fmt.Errorf(
			"%s new policy: %w",
			command,
			err,
		)
	}
	return oldPolicy, oldPolicySHA256, newPolicy, newPolicySHA256, nil
}

func runVerifySupersession(
	args []string,
	stdin io.Reader,
	stdout, stderr io.Writer,
) int {
	flags := flag.NewFlagSet("zerone-ops verify-supersession", flag.ContinueOnError)
	flags.SetOutput(stderr)
	config := addSupersessionFlags(flags)
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if flags.NArg() != 0 {
		fmt.Fprintln(stderr, "zerone-ops verify-supersession: positional arguments are not accepted")
		return 2
	}
	oldPolicy, oldPolicySHA256, newPolicy, newPolicySHA256, err :=
		loadSupersessionPolicies("verify-supersession", config)
	if err != nil {
		fmt.Fprintf(stderr, "zerone-ops: %v\n", err)
		return 2
	}
	document, err := readTransition(*config.input, stdin)
	if err != nil {
		fmt.Fprintf(stderr, "zerone-ops verify-supersession: %v\n", err)
		return 2
	}
	result, err := verifySupersession(
		document,
		oldPolicy,
		oldPolicySHA256,
		newPolicy,
		newPolicySHA256,
		*config.oldHeadSHA256,
	)
	if err != nil {
		fmt.Fprintf(stderr, "zerone-ops verify-supersession: INVALID: %v\n", err)
		return 1
	}
	fmt.Fprintf(
		stdout,
		"VALID-SUPERSESSION chain_id=%s old_head_sha256=%s old_policy_sha256=%s new_policy_sha256=%s replacement_incident_id=%s replacement_release_id=%s supersession_sha256=%s\n",
		result.ChainID,
		result.OldJournalHeadSHA256,
		result.OldTrustPolicySHA256,
		result.NewTrustPolicySHA256,
		result.ReplacementIncidentID,
		result.ReplacementReleaseID,
		result.SupersessionSHA256,
	)
	return 0
}

func runVerifyReplacement(
	args []string,
	stdin io.Reader,
	stdout, stderr io.Writer,
) int {
	flags := flag.NewFlagSet("zerone-ops verify-replacement", flag.ContinueOnError)
	flags.SetOutput(stderr)
	config := addSupersessionFlags(flags)
	newHeadSHA256 := flags.String(
		"new-head-sha256",
		"",
		"replacement journal head SHA-256 from a separate trusted channel (required)",
	)
	oldPins := make(powerSnapshotPins)
	newPins := make(powerSnapshotPins)
	var oldPaths, newPaths repeatedPaths
	flags.Var(
		&oldPaths,
		"old-journal",
		"old transition path in sequence order; repeat for every transition",
	)
	flags.Var(
		&newPaths,
		"new-journal",
		"replacement transition path in sequence order; repeat for every transition",
	)
	flags.Var(
		oldPins,
		"old-power-snapshot-pin",
		"old journal <sequence>=<snapshot-sha256>; repeat for every power-gated transition",
	)
	flags.Var(
		newPins,
		"new-power-snapshot-pin",
		"replacement journal <sequence>=<snapshot-sha256>; repeat for every power-gated transition",
	)
	flags.Usage = func() {
		fmt.Fprintln(
			stderr,
			"Usage: zerone-ops verify-replacement --old-trust-policy <old.json> --old-trust-policy-sha256 <hex> --new-trust-policy <new.json> --new-trust-policy-sha256 <hex> --old-head-sha256 <hex> --new-head-sha256 <hex> --old-journal <path> --input <sidecar.json> --new-journal <path>",
		)
		flags.PrintDefaults()
	}
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if flags.NArg() != 0 {
		fmt.Fprintln(
			stderr,
			"zerone-ops verify-replacement: positional arguments are not accepted",
		)
		return 2
	}
	if len(oldPaths) == 0 || len(newPaths) == 0 {
		fmt.Fprintln(
			stderr,
			"zerone-ops verify-replacement: at least one --old-journal and one --new-journal are required",
		)
		return 2
	}
	for _, path := range append(append([]string{}, oldPaths...), newPaths...) {
		if path == "-" {
			fmt.Fprintln(
				stderr,
				"zerone-ops verify-replacement: journal paths cannot use stdin (-); reserve stdin for --input",
			)
			return 2
		}
	}
	if err := validateSHA256(
		"externally pinned old journal head SHA-256",
		*config.oldHeadSHA256,
		false,
	); err != nil {
		fmt.Fprintf(
			stderr,
			"zerone-ops verify-replacement: --old-head-sha256 is required and must be exact: %v\n",
			err,
		)
		return 2
	}
	if err := validateSHA256(
		"externally pinned replacement journal head SHA-256",
		*newHeadSHA256,
		false,
	); err != nil {
		fmt.Fprintf(
			stderr,
			"zerone-ops verify-replacement: --new-head-sha256 is required and must be exact: %v\n",
			err,
		)
		return 2
	}
	oldPolicy, oldPolicySHA256, newPolicy, newPolicySHA256, err :=
		loadSupersessionPolicies("verify-replacement", config)
	if err != nil {
		fmt.Fprintf(stderr, "zerone-ops: %v\n", err)
		return 2
	}

	readDocuments := func(paths []string) ([][]byte, error) {
		documents := make([][]byte, 0, len(paths))
		for _, path := range paths {
			document, err := readTransition(path, stdin)
			if err != nil {
				return nil, err
			}
			documents = append(documents, document)
		}
		return documents, nil
	}
	oldDocuments, err := readDocuments(oldPaths)
	if err != nil {
		fmt.Fprintf(stderr, "zerone-ops verify-replacement: old journal: %v\n", err)
		return 2
	}
	supersessionDocument, err := readTransition(*config.input, stdin)
	if err != nil {
		fmt.Fprintf(stderr, "zerone-ops verify-replacement: supersession: %v\n", err)
		return 2
	}
	newDocuments, err := readDocuments(newPaths)
	if err != nil {
		fmt.Fprintf(stderr, "zerone-ops verify-replacement: replacement journal: %v\n", err)
		return 2
	}

	result, err := verifyReplacement(
		oldDocuments,
		supersessionDocument,
		newDocuments,
		VerifyOptions{
			ExpectedHeadSHA256:          *config.oldHeadSHA256,
			ExpectedPowerSnapshotSHA256: map[uint64]string(oldPins),
			TrustPolicy:                 &oldPolicy,
			TrustPolicySHA256:           oldPolicySHA256,
		},
		VerifyOptions{
			ExpectedHeadSHA256:          *newHeadSHA256,
			ExpectedPowerSnapshotSHA256: map[uint64]string(newPins),
			TrustPolicy:                 &newPolicy,
			TrustPolicySHA256:           newPolicySHA256,
		},
	)
	if err != nil {
		fmt.Fprintf(stderr, "zerone-ops verify-replacement: INVALID: %v\n", err)
		return 1
	}
	fmt.Fprintf(
		stdout,
		"VALID-REPLACEMENT chain_id=%s old_head_sha256=%s supersession_sha256=%s new_lane=%s replacement_incident_id=%s replacement_release_id=%s new_head_sha256=%s\n",
		result.Supersession.ChainID,
		result.OldJournal.HeadSHA256,
		result.Supersession.SupersessionSHA256,
		result.NewJournal.Lane,
		result.NewJournal.IncidentID,
		result.NewJournal.ReleaseID,
		result.NewJournal.HeadSHA256,
	)
	return 0
}

func runSupersessionApprovalStatement(
	args []string,
	stdin io.Reader,
	stdout, stderr io.Writer,
) int {
	flags := flag.NewFlagSet(
		"zerone-ops supersession-approval-statement",
		flag.ContinueOnError,
	)
	flags.SetOutput(stderr)
	config := addSupersessionFlags(flags)
	role := flags.String("role", "", "approval role")
	identity := flags.String("identity", "", "approval identity")
	publicKey := flags.String(
		"public-key",
		"",
		"Ed25519 public key (64 lowercase hex characters)",
	)
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if flags.NArg() != 0 {
		fmt.Fprintln(
			stderr,
			"zerone-ops supersession-approval-statement: positional arguments are not accepted",
		)
		return 2
	}
	oldPolicy, oldPolicySHA256, newPolicy, newPolicySHA256, err :=
		loadSupersessionPolicies("supersession-approval-statement", config)
	if err != nil {
		fmt.Fprintf(stderr, "zerone-ops: %v\n", err)
		return 2
	}
	document, err := readTransition(*config.input, stdin)
	if err != nil {
		fmt.Fprintf(stderr, "zerone-ops supersession-approval-statement: %v\n", err)
		return 2
	}
	supersession, err := decodeSupersession(document, false)
	if err != nil {
		fmt.Fprintf(stderr, "zerone-ops supersession-approval-statement: %v\n", err)
		return 1
	}
	if supersession.SupersessionSHA256 != "" {
		fmt.Fprintln(
			stderr,
			"zerone-ops supersession-approval-statement: supersession_sha256 must be empty in a draft",
		)
		return 1
	}
	if err := validateSupersessionCore(supersession); err != nil {
		fmt.Fprintf(stderr, "zerone-ops supersession-approval-statement: %v\n", err)
		return 1
	}
	if err := validateSupersessionBindings(
		supersession,
		oldPolicy,
		oldPolicySHA256,
		newPolicy,
		newPolicySHA256,
		*config.oldHeadSHA256,
	); err != nil {
		fmt.Fprintf(stderr, "zerone-ops supersession-approval-statement: %v\n", err)
		return 1
	}
	approval := Approval{
		Role:      *role,
		Identity:  *identity,
		PublicKey: *publicKey,
		Power:     "0",
	}
	if approval.Role != evidenceCustodianRole &&
		approval.Role != policyRotationAuthorityRole {
		fmt.Fprintf(
			stderr,
			"zerone-ops supersession-approval-statement: role must be %q or %q\n",
			evidenceCustodianRole,
			policyRotationAuthorityRole,
		)
		return 1
	}
	if err := validateLabel("identity", approval.Identity, 256); err != nil {
		fmt.Fprintf(stderr, "zerone-ops supersession-approval-statement: %v\n", err)
		return 1
	}
	if _, err := decodeExactHex("public-key", approval.PublicKey, 32); err != nil {
		fmt.Fprintf(stderr, "zerone-ops supersession-approval-statement: %v\n", err)
		return 1
	}
	if err := validateApprovalAuthorized(approval, oldPolicy); err != nil {
		fmt.Fprintf(stderr, "zerone-ops supersession-approval-statement: %v\n", err)
		return 1
	}
	digest, err := supersessionApprovalStatementDigest(supersession, approval)
	if err != nil {
		fmt.Fprintf(stderr, "zerone-ops supersession-approval-statement: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "%x\n", digest)
	return 0
}

func runSealSupersession(
	args []string,
	stdin io.Reader,
	stdout, stderr io.Writer,
) int {
	flags := flag.NewFlagSet("zerone-ops seal-supersession", flag.ContinueOnError)
	flags.SetOutput(stderr)
	config := addSupersessionFlags(flags)
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if flags.NArg() != 0 {
		fmt.Fprintln(stderr, "zerone-ops seal-supersession: positional arguments are not accepted")
		return 2
	}
	oldPolicy, oldPolicySHA256, newPolicy, newPolicySHA256, err :=
		loadSupersessionPolicies("seal-supersession", config)
	if err != nil {
		fmt.Fprintf(stderr, "zerone-ops: %v\n", err)
		return 2
	}
	document, err := readTransition(*config.input, stdin)
	if err != nil {
		fmt.Fprintf(stderr, "zerone-ops seal-supersession: %v\n", err)
		return 2
	}
	supersession, err := decodeSupersession(document, false)
	if err != nil {
		fmt.Fprintf(stderr, "zerone-ops seal-supersession: %v\n", err)
		return 1
	}
	sealed, err := sealSupersession(
		supersession,
		oldPolicy,
		oldPolicySHA256,
		newPolicy,
		newPolicySHA256,
		*config.oldHeadSHA256,
	)
	if err != nil {
		fmt.Fprintf(stderr, "zerone-ops seal-supersession: %v\n", err)
		return 1
	}
	canonical, err := json.Marshal(sealed)
	if err != nil {
		fmt.Fprintf(stderr, "zerone-ops seal-supersession: canonicalize: %v\n", err)
		return 1
	}
	if _, err := stdout.Write(canonical); err != nil {
		fmt.Fprintf(stderr, "zerone-ops seal-supersession: write output: %v\n", err)
		return 2
	}
	return 0
}

func readTransition(path string, stdin io.Reader) ([]byte, error) {
	if path == "-" {
		reader := io.LimitReader(stdin, maxTransitionBytes+1)
		document, err := io.ReadAll(reader)
		if err != nil {
			return nil, fmt.Errorf("read stdin: %w", err)
		}
		if len(document) > maxTransitionBytes {
			return nil, fmt.Errorf("stdin exceeds %d-byte limit", maxTransitionBytes)
		}
		return document, nil
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("stat opened %s: %w", path, err)
	}
	if info.IsDir() {
		return nil, fmt.Errorf("%s is a directory", path)
	}
	reader := io.LimitReader(file, maxTransitionBytes+1)
	document, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	if len(document) > maxTransitionBytes {
		return nil, fmt.Errorf("%s exceeds %d-byte limit", path, maxTransitionBytes)
	}
	return document, nil
}
