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
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		printUsage(stderr)
		return 2
	}
	switch args[0] {
	case "evaluate":
		return runEvaluate(args[1:], stdout, stderr)
	case "help", "-h", "--help":
		printUsage(stdout)
		return 0
	default:
		fmt.Fprintln(stderr, "validator-recovery-gate: unknown command")
		printUsage(stderr)
		return 2
	}
}

func printUsage(output io.Writer) {
	fmt.Fprintln(output, "Usage:")
	fmt.Fprintln(output, "  validator-recovery-gate evaluate --chain-id <trusted-id> --incident-id <trusted-id> \\")
	fmt.Fprintln(output, "    --custody-policy <canonical.json> --custody-policy-sha256 <external-hex> \\")
	fmt.Fprintln(output, "    --assessment <canonical.json> --assessment-sha256 <external-hex> \\")
	fmt.Fprintln(output, "    [--controlled-policy <canonical.json> --controlled-policy-sha256 <external-hex> \\")
	fmt.Fprintln(output, "     --controlled <canonical.json> --controlled-sha256 <external-hex>] \\")
	fmt.Fprintln(output, "    [--fork-policy <canonical.json> --fork-policy-sha256 <external-hex> \\")
	fmt.Fprintln(output, "     --fork-release <canonical.json> --fork-release-sha256 <external-hex> \\")
	fmt.Fprintln(output, "     --fork-choice <canonical.json> --fork-choice-sha256 <external-hex> \\")
	fmt.Fprintln(output, "     --genesis <canonical.json> --genesis-sha256 <external-hex> \\")
	fmt.Fprintln(output, "     --compiler-report-a <canonical.json> --compiler-report-a-sha256 <external-hex> \\")
	fmt.Fprintln(output, "     --compiler-report-b <canonical.json> --compiler-report-b-sha256 <external-hex>]")
	fmt.Fprintln(output)
	fmt.Fprintln(output, "The command is offline, accepts only bounded non-symlink regular files, and emits one canonical self-hashed report.")
}

func runEvaluate(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("validator-recovery-gate evaluate", flag.ContinueOnError)
	flags.SetOutput(stderr)
	chainID := flags.String("chain-id", "", "old chain ID obtained through a trusted channel")
	incidentID := flags.String("incident-id", "", "incident ID obtained through a trusted channel")
	custodyPolicyPath := flags.String("custody-policy", "", "canonical independently provisioned custody signer policy")
	custodyPolicyPin := flags.String("custody-policy-sha256", "", "separately obtained custody signer policy SHA-256")
	assessmentPath := flags.String("assessment", "", "canonical custody assessment")
	assessmentPin := flags.String("assessment-sha256", "", "separately obtained assessment file SHA-256")
	controlledPath := flags.String("controlled", "", "canonical controlled-transition plan")
	controlledPin := flags.String("controlled-sha256", "", "separately obtained controlled plan file SHA-256")
	controlledPolicyPath := flags.String("controlled-policy", "", "canonical independently provisioned controlled signer policy")
	controlledPolicyPin := flags.String("controlled-policy-sha256", "", "separately obtained controlled signer policy SHA-256")
	forkPolicyPath := flags.String("fork-policy", "", "canonical separately provisioned fork policy")
	forkPolicyPin := flags.String("fork-policy-sha256", "", "separately obtained fork policy file SHA-256")
	forkReleasePath := flags.String("fork-release", "", "canonical fork release")
	forkReleasePin := flags.String("fork-release-sha256", "", "separately obtained fork release file SHA-256")
	forkChoicePath := flags.String("fork-choice", "", "canonical signed fork choice")
	forkChoicePin := flags.String("fork-choice-sha256", "", "separately obtained fork choice file SHA-256")
	genesisPath := flags.String("genesis", "", "exact canonical fork genesis")
	genesisPin := flags.String("genesis-sha256", "", "separately obtained exact fork genesis SHA-256")
	compilerReportAPath := flags.String("compiler-report-a", "", "first exact canonical fork-genesis compiler report")
	compilerReportAPin := flags.String("compiler-report-a-sha256", "", "separately obtained first compiler report SHA-256")
	compilerReportBPath := flags.String("compiler-report-b", "", "second exact canonical fork-genesis compiler report")
	compilerReportBPin := flags.String("compiler-report-b-sha256", "", "separately obtained second compiler report SHA-256")
	flags.Usage = func() {
		printUsage(stderr)
		flags.PrintDefaults()
	}
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if flags.NArg() != 0 {
		fmt.Fprintln(stderr, "validator-recovery-gate evaluate: positional arguments are not accepted")
		return 2
	}
	if err := validateLabel("--chain-id", *chainID, 256); err != nil {
		fmt.Fprintln(stderr, "validator-recovery-gate evaluate: --chain-id is required and invalid")
		return 2
	}
	if err := validateLabel("--incident-id", *incidentID, 256); err != nil {
		fmt.Fprintln(stderr, "validator-recovery-gate evaluate: --incident-id is required and invalid")
		return 2
	}

	assessmentData, assessmentSHA256, err := readPinnedRegularFile(
		*assessmentPath,
		*assessmentPin,
		"custody assessment",
	)
	if err != nil {
		fmt.Fprintf(stderr, "validator-recovery-gate evaluate: %v\n", err)
		return 2
	}
	assessment, err := decodeExactJSON[CustodyAssessment](
		assessmentData,
		"custody assessment",
	)
	if err != nil {
		fmt.Fprintf(stderr, "validator-recovery-gate evaluate: %v\n", err)
		return 2
	}
	if assessment.ChainID != *chainID || assessment.IncidentID != *incidentID {
		fmt.Fprintln(stderr, "validator-recovery-gate evaluate: assessment does not match the independently trusted chain and incident IDs")
		return 1
	}

	inputs := EvaluationInputs{
		Assessment:       assessment,
		AssessmentSHA256: assessmentSHA256,
	}
	var loadErr error
	inputs.CustodyPolicy, inputs.CustodyPolicySHA256, loadErr =
		loadOptionalPinned[SignerPolicy](
			*custodyPolicyPath,
			*custodyPolicyPin,
			"custody signer policy",
		)
	if loadErr != nil {
		fmt.Fprintf(stderr, "validator-recovery-gate evaluate: %v\n", loadErr)
		return 2
	}
	if inputs.CustodyPolicy != nil &&
		validateCustodyAssessment(
			assessment,
			*inputs.CustodyPolicy,
			inputs.CustodyPolicySHA256,
		) == nil {
		route, _ := custodyRoute(assessment)
		switch route {
		case routeControlled:
			if *forkPolicyPath != "" || *forkPolicyPin != "" ||
				*forkReleasePath != "" || *forkReleasePin != "" ||
				*forkChoicePath != "" || *forkChoicePin != "" ||
				*genesisPath != "" || *genesisPin != "" ||
				*compilerReportAPath != "" || *compilerReportAPin != "" ||
				*compilerReportBPath != "" || *compilerReportBPin != "" {
				fmt.Fprintln(stderr, "validator-recovery-gate evaluate: fork inputs are not accepted for a controlled-required assessment")
				return 2
			}
			inputs.ControlledPolicy, inputs.ControlledPolicySHA256, loadErr =
				loadOptionalPinned[SignerPolicy](
					*controlledPolicyPath,
					*controlledPolicyPin,
					"controlled signer policy",
				)
			if loadErr != nil {
				fmt.Fprintf(stderr, "validator-recovery-gate evaluate: %v\n", loadErr)
				return 2
			}
			if *controlledPath != "" || *controlledPin != "" {
				if *controlledPath == "" || *controlledPin == "" {
					fmt.Fprintln(stderr, "validator-recovery-gate evaluate: controlled path and pin must be supplied together")
					return 2
				}
				data, digest, readErr := readPinnedRegularFile(
					*controlledPath,
					*controlledPin,
					"controlled transition",
				)
				if readErr != nil {
					fmt.Fprintf(stderr, "validator-recovery-gate evaluate: %v\n", readErr)
					return 2
				}
				document, decodeErr := decodeExactJSON[ControlledTransition](
					data,
					"controlled transition",
				)
				if decodeErr != nil {
					fmt.Fprintf(stderr, "validator-recovery-gate evaluate: %v\n", decodeErr)
					return 2
				}
				inputs.Controlled = &document
				inputs.ControlledSHA256 = digest
			}
		case routeFork:
			if *controlledPath != "" || *controlledPin != "" ||
				*controlledPolicyPath != "" || *controlledPolicyPin != "" {
				fmt.Fprintln(stderr, "validator-recovery-gate evaluate: controlled inputs are prohibited for a fork-required assessment")
				return 2
			}
			inputs.ForkPolicy, inputs.ForkPolicySHA256, loadErr =
				loadOptionalPinned[ForkPolicy](
					*forkPolicyPath,
					*forkPolicyPin,
					"fork policy",
				)
			if loadErr == nil {
				inputs.ForkRelease, inputs.ForkReleaseSHA256, loadErr =
					loadOptionalPinned[ForkRelease](
						*forkReleasePath,
						*forkReleasePin,
						"fork release",
					)
			}
			if loadErr == nil {
				inputs.Genesis, inputs.GenesisSHA256, loadErr =
					loadOptionalForkGenesis(*genesisPath, *genesisPin)
			}
			if loadErr == nil {
				var report *ForkGenesisReport
				var digest string
				report, digest, loadErr =
					loadOptionalPinnedBounded[ForkGenesisReport](
						*compilerReportAPath,
						*compilerReportAPin,
						"first compiler report",
						maxCompilerReportBytes,
					)
				if report != nil {
					inputs.CompilerReports = append(inputs.CompilerReports, *report)
					inputs.CompilerReportSHA256s = append(
						inputs.CompilerReportSHA256s,
						digest,
					)
				}
			}
			if loadErr == nil {
				var report *ForkGenesisReport
				var digest string
				report, digest, loadErr =
					loadOptionalPinnedBounded[ForkGenesisReport](
						*compilerReportBPath,
						*compilerReportBPin,
						"second compiler report",
						maxCompilerReportBytes,
					)
				if report != nil {
					inputs.CompilerReports = append(inputs.CompilerReports, *report)
					inputs.CompilerReportSHA256s = append(
						inputs.CompilerReportSHA256s,
						digest,
					)
				}
			}
			if loadErr == nil {
				inputs.ForkChoice, inputs.ForkChoiceSHA256, loadErr =
					loadOptionalPinned[ForkChoice](
						*forkChoicePath,
						*forkChoicePin,
						"fork choice",
					)
			}
			if loadErr != nil {
				fmt.Fprintf(stderr, "validator-recovery-gate evaluate: %v\n", loadErr)
				return 2
			}
		}
	}

	report, err := evaluate(inputs)
	if err != nil {
		fmt.Fprintln(stderr, "validator-recovery-gate evaluate: report generation failed")
		return 2
	}
	canonical, err := json.Marshal(report)
	if err != nil {
		fmt.Fprintln(stderr, "validator-recovery-gate evaluate: report encoding failed")
		return 2
	}
	if _, err := stdout.Write(canonical); err != nil {
		fmt.Fprintln(stderr, "validator-recovery-gate evaluate: report write failed")
		return 2
	}
	if report.Decision == decisionNoGo {
		return 1
	}
	return 0
}

func loadOptionalPinned[T any](
	path,
	pin,
	documentName string,
) (*T, string, error) {
	if path == "" && pin == "" {
		return nil, "", nil
	}
	if path == "" || pin == "" {
		return nil, "", fmt.Errorf("%s path and pin must be supplied together", documentName)
	}
	data, digest, err := readPinnedRegularFile(path, pin, documentName)
	if err != nil {
		return nil, "", err
	}
	document, err := decodeExactJSON[T](data, documentName)
	if err != nil {
		return nil, "", err
	}
	return &document, digest, nil
}

func loadOptionalPinnedBounded[T any](
	path,
	pin,
	documentName string,
	maximumBytes int64,
) (*T, string, error) {
	if path == "" && pin == "" {
		return nil, "", nil
	}
	if path == "" || pin == "" {
		return nil, "", fmt.Errorf("%s path and pin must be supplied together", documentName)
	}
	data, digest, err := readPinnedRegularFileBounded(
		path,
		pin,
		documentName,
		maximumBytes,
	)
	if err != nil {
		return nil, "", err
	}
	document, err := decodeExactJSON[T](data, documentName)
	if err != nil {
		return nil, "", err
	}
	return &document, digest, nil
}

func loadOptionalForkGenesis(
	path,
	pin string,
) (*ForkGenesis, string, error) {
	if path == "" && pin == "" {
		return nil, "", nil
	}
	if path == "" || pin == "" {
		return nil, "", errors.New("fork genesis path and pin must be supplied together")
	}
	data, digest, err := readPinnedRegularFileBounded(
		path,
		pin,
		"fork genesis",
		maxGenesisBytes,
	)
	if err != nil {
		return nil, "", err
	}
	genesis, err := decodeForkGenesis(data)
	if err != nil {
		return nil, "", err
	}
	return &genesis, digest, nil
}
