package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
)

const (
	exitOK            = 0
	exitCheckFailed   = 2
	exitTargetRefused = 3
)

type checkOptions struct {
	enforceCanonicalManifest bool
}

type report struct {
	Schema                         string   `json:"schema"`
	Mode                           string   `json:"mode"`
	Status                         string   `json:"status"`
	Scope                          string   `json:"scope"`
	ManifestPath                   string   `json:"manifestPath"`
	ManifestSHA256                 string   `json:"manifestSha256"`
	ManifestSchema                 string   `json:"manifestSchema"`
	ManifestStatus                 string   `json:"manifestStatus"`
	SourceDesignSHA256             string   `json:"sourceDesignSha256"`
	SourceAnchorsVerified          int      `json:"sourceAnchorsVerified"`
	StaticAuthorityGate            string   `json:"staticAuthorityGate"`
	StaticAuthoritySurfacesPassing int      `json:"staticAuthoritySurfacesPassing"`
	StaticAuthoritySurfacesTotal   int      `json:"staticAuthoritySurfacesTotal"`
	H4GatesEvidenced               int      `json:"h4GatesEvidenced"`
	H4GatesTotal                   int      `json:"h4GatesTotal"`
	H5GatesEvidenced               int      `json:"h5GatesEvidenced"`
	H5GatesTotal                   int      `json:"h5GatesTotal"`
	ReleaseAssessment              string   `json:"releaseAssessment"`
	TargetGateMustExitNonZero      bool     `json:"targetGateMustExitNonZero"`
	DiscoveredCustomStakeConsumers []string `json:"discoveredCustomStakingConsumers"`
	DetectedAuthorityConstructors  []string `json:"detectedAuthorityConstructors"`
	AbsentTargetModules            []string `json:"absentTargetModules"`
	BlockerIDs                     []string `json:"blockerIds"`
	UnevidencedActivationGateIDs   []string `json:"unevidencedActivationGateIds"`
}

type failureReport struct {
	Schema       string  `json:"schema"`
	Mode         string  `json:"mode"`
	Status       string  `json:"status"`
	ManifestPath string  `json:"manifestPath"`
	Issues       []issue `json:"issues"`
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout))
}

func run(args []string, output io.Writer) int {
	return runWithOptions(args, output, checkOptions{enforceCanonicalManifest: true})
}

func runWithOptions(args []string, output io.Writer, options checkOptions) int {
	mode, root, usageIssue := parseArguments(args)
	if usageIssue != nil {
		writeFailure(output, mode, []issue{*usageIssue})
		return exitCheckFailed
	}

	manifestBytes, reason := readRepositoryFile(root, manifestPath, maxManifestBytes)
	if reason != "" {
		writeFailure(output, mode, []issue{{ID: "MANIFEST_READ_FAILED", Detail: reason}})
		return exitCheckFailed
	}

	issues := &issueSet{}
	manifestDigest := sha256Hex(manifestBytes)
	if options.enforceCanonicalManifest && manifestDigest != canonicalManifestSHA256 {
		issues.add("MANIFEST_SHA256_MISMATCH", "manifest does not match the reviewed Authority Geometry v1 SHA-256")
	}
	m, decoded := decodeManifest(manifestBytes, issues)
	if decoded {
		validateManifest(m, issues)
		verifiedAnchors, discovery := validateSource(root, m, issues)
		if len(issues.items) == 0 {
			report := buildReport(mode, manifestDigest, m, verifiedAnchors, discovery)
			if mode == "target-gate" {
				report.Status = "TARGET_GATE_REFUSED"
				writeJSON(output, report)
				return exitTargetRefused
			}
			writeJSON(output, report)
			return exitOK
		}
	}

	writeFailure(output, mode, issues.sorted())
	return exitCheckFailed
}

func parseArguments(args []string) (mode, root string, problem *issue) {
	mode = "unknown"
	root = "."
	if len(args) == 0 {
		return mode, root, &issue{ID: "USAGE_INVALID", Detail: "usage: authority-graph (report|target-gate) [--root PATH]"}
	}
	mode = args[0]
	if mode != "report" && mode != "target-gate" {
		return mode, root, &issue{ID: "USAGE_INVALID", Detail: "mode must be report or target-gate"}
	}
	flags := flag.NewFlagSet("authority-graph "+mode, flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	flags.StringVar(&root, "root", ".", "repository root")
	if err := flags.Parse(args[1:]); err != nil || flags.NArg() != 0 || root == "" {
		return mode, ".", &issue{ID: "USAGE_INVALID", Detail: "usage: authority-graph (report|target-gate) [--root PATH]"}
	}
	return mode, root, nil
}

func buildReport(mode, manifestDigest string, m manifest, verifiedAnchors int, discovery sourceDiscovery) report {
	status := "CURRENT_SOURCE_COMPLETELY_CLASSIFIED"
	blockers := make([]string, 0, len(m.CurrentFindings))
	for _, finding := range m.CurrentFindings {
		if finding.Status == "OPEN" {
			blockers = append(blockers, finding.ID)
		}
	}
	sort.Strings(blockers)

	unevidenced := make([]string, 0, len(m.ActivationGates.H4)+len(m.ActivationGates.H5))
	for _, gate := range append(append([]activationGate(nil), m.ActivationGates.H4...), m.ActivationGates.H5...) {
		if gate.Status == "NOT_EVIDENCED" {
			unevidenced = append(unevidenced, gate.ID)
		}
	}
	targetMustRefuse := m.ReleaseAssessment.TargetGateMustExitNonZero != nil && *m.ReleaseAssessment.TargetGateMustExitNonZero
	return report{
		Schema:                         checkerSchema,
		Mode:                           mode,
		Status:                         status,
		Scope:                          "SEVEN_H4_02_STATIC_AUTHORITY_SURFACES",
		ManifestPath:                   manifestPath,
		ManifestSHA256:                 manifestDigest,
		ManifestSchema:                 m.Schema,
		ManifestStatus:                 m.Status,
		SourceDesignSHA256:             m.SourceDesign.SHA256,
		SourceAnchorsVerified:          verifiedAnchors,
		StaticAuthorityGate:            m.StaticAuthorityGate.Status,
		StaticAuthoritySurfacesPassing: m.ReleaseAssessment.StaticAuthoritySurfacesPassing,
		StaticAuthoritySurfacesTotal:   m.ReleaseAssessment.StaticAuthoritySurfacesTotal,
		H4GatesEvidenced:               m.ReleaseAssessment.H4GatesEvidenced,
		H4GatesTotal:                   m.ReleaseAssessment.H4GatesTotal,
		H5GatesEvidenced:               m.ReleaseAssessment.H5GatesEvidenced,
		H5GatesTotal:                   m.ReleaseAssessment.H5GatesTotal,
		ReleaseAssessment:              m.ReleaseAssessment.Overall,
		TargetGateMustExitNonZero:      targetMustRefuse,
		DiscoveredCustomStakeConsumers: append([]string(nil), discovery.CustomStakingConsumers...),
		DetectedAuthorityConstructors:  append([]string(nil), discovery.Constructors...),
		AbsentTargetModules:            append([]string(nil), discovery.AbsentTargetModules...),
		BlockerIDs:                     blockers,
		UnevidencedActivationGateIDs:   unevidenced,
	}
}

func writeFailure(output io.Writer, mode string, issues []issue) {
	if len(issues) == 0 {
		issues = []issue{{ID: "CHECK_FAILED", Detail: "authority graph verification failed closed"}}
	}
	writeJSON(output, failureReport{
		Schema:       checkerSchema,
		Mode:         mode,
		Status:       "CHECK_FAILED",
		ManifestPath: manifestPath,
		Issues:       issues,
	})
}

func writeJSON(output io.Writer, value any) {
	encoder := json.NewEncoder(output)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(value); err != nil {
		_, _ = fmt.Fprintln(io.Discard, err)
	}
}
