package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
)

type EvaluationInputs struct {
	CustodyPolicy          *SignerPolicy
	CustodyPolicySHA256    string
	Assessment             CustodyAssessment
	AssessmentSHA256       string
	ControlledPolicy       *SignerPolicy
	ControlledPolicySHA256 string
	Controlled             *ControlledTransition
	ControlledSHA256       string
	ForkPolicy             *ForkPolicy
	ForkPolicySHA256       string
	ForkRelease            *ForkRelease
	ForkReleaseSHA256      string
	ForkChoice             *ForkChoice
	ForkChoiceSHA256       string
	Genesis                *ForkGenesis
	GenesisSHA256          string
	CompilerReports        []ForkGenesisReport
	CompilerReportSHA256s  []string
}

func evaluate(inputs EvaluationInputs) (GateReport, error) {
	compilerReportDigests := append([]string{}, inputs.CompilerReportSHA256s...)
	sort.Strings(compilerReportDigests)
	inputDigests := GateInputDigests{
		CustodyPolicySHA256:    inputs.CustodyPolicySHA256,
		AssessmentSHA256:       inputs.AssessmentSHA256,
		ControlledPolicySHA256: inputs.ControlledPolicySHA256,
		ControlledSHA256:       inputs.ControlledSHA256,
		ForkPolicySHA256:       inputs.ForkPolicySHA256,
		ForkReleaseSHA256:      inputs.ForkReleaseSHA256,
		ForkChoiceSHA256:       inputs.ForkChoiceSHA256,
		GenesisSHA256:          inputs.GenesisSHA256,
		CompilerReportSHA256s:  compilerReportDigests,
	}
	report := GateReport{
		Schema:         gateReportSchema,
		EvaluatedAt:    "",
		IncidentID:     "",
		OldChainID:     "",
		Decision:       decisionNoGo,
		ReasonCodes:    []string{},
		InputDigests:   inputDigests,
		SelectedSHA256: "",
		ReportSHA256:   "",
	}

	if inputs.CustodyPolicy == nil || inputs.CustodyPolicySHA256 == "" {
		report.RequiredRoute = routeFork
		report.ReasonCodes = []string{"CUSTODY_POLICY_MISSING"}
		return sealGateReport(report)
	}
	if !matchesExactCanonicalDigest(
		*inputs.CustodyPolicy,
		inputs.CustodyPolicySHA256,
	) {
		report.RequiredRoute = routeFork
		report.ReasonCodes = []string{"CUSTODY_POLICY_INVALID"}
		return sealGateReport(report)
	}
	if err := validateSignerPolicy(
		*inputs.CustodyPolicy,
		signerPurposeCustody,
		inputs.Assessment.IncidentID,
		inputs.Assessment.ChainID,
		"",
		requiredCustodyRoles,
	); err != nil {
		report.RequiredRoute = routeFork
		report.ReasonCodes = []string{"CUSTODY_POLICY_INVALID"}
		return sealGateReport(report)
	}
	if !matchesExactCanonicalDigest(
		inputs.Assessment,
		inputs.AssessmentSHA256,
	) {
		report.RequiredRoute = routeFork
		report.ReasonCodes = []string{"CUSTODY_ASSESSMENT_INVALID"}
		return sealGateReport(report)
	}
	if err := validateCustodyAssessment(
		inputs.Assessment,
		*inputs.CustodyPolicy,
		inputs.CustodyPolicySHA256,
	); err != nil {
		report.RequiredRoute = routeFork
		report.ReasonCodes = []string{"CUSTODY_ASSESSMENT_INVALID"}
		return sealGateReport(report)
	}
	report.EvaluatedAt = inputs.Assessment.EvaluatedAt
	report.IncidentID = inputs.Assessment.IncidentID
	report.OldChainID = inputs.Assessment.ChainID

	route, routeReasons := custodyRoute(inputs.Assessment)
	report.RequiredRoute = route
	report.ReasonCodes = append(report.ReasonCodes, routeReasons...)
	switch route {
	case routeControlled:
		if inputs.ForkPolicy != nil || inputs.ForkPolicySHA256 != "" ||
			inputs.ForkRelease != nil || inputs.ForkReleaseSHA256 != "" ||
			inputs.ForkChoice != nil || inputs.ForkChoiceSHA256 != "" ||
			inputs.Genesis != nil || inputs.GenesisSHA256 != "" ||
			len(inputs.CompilerReports) != 0 ||
			len(inputs.CompilerReportSHA256s) != 0 {
			report.ReasonCodes = append(
				report.ReasonCodes,
				"INACTIVE_ROUTE_INPUTS_PRESENT",
			)
			break
		}
		if inputs.Controlled == nil || inputs.ControlledSHA256 == "" {
			report.ReasonCodes = append(report.ReasonCodes, "CONTROLLED_PLAN_MISSING")
			break
		}
		if inputs.ControlledPolicy == nil || inputs.ControlledPolicySHA256 == "" {
			report.ReasonCodes = append(report.ReasonCodes, "CONTROLLED_POLICY_MISSING")
			break
		}
		if !matchesExactCanonicalDigest(
			*inputs.ControlledPolicy,
			inputs.ControlledPolicySHA256,
		) {
			report.ReasonCodes = append(report.ReasonCodes, "CONTROLLED_POLICY_INVALID")
			break
		}
		if !matchesExactCanonicalDigest(
			*inputs.Controlled,
			inputs.ControlledSHA256,
		) {
			report.ReasonCodes = append(report.ReasonCodes, "CONTROLLED_PLAN_INVALID")
			break
		}
		if err := validateControlledTransition(
			*inputs.Controlled,
			inputs.Assessment,
			inputs.AssessmentSHA256,
			*inputs.ControlledPolicy,
			inputs.ControlledPolicySHA256,
		); err != nil {
			report.ReasonCodes = append(report.ReasonCodes, "CONTROLLED_PLAN_INVALID")
			break
		}
		report.Decision = decisionControlled
		report.SelectedSHA256 = inputs.ControlledSHA256
	case routeFork:
		if inputs.ControlledPolicy != nil ||
			inputs.ControlledPolicySHA256 != "" ||
			inputs.Controlled != nil ||
			inputs.ControlledSHA256 != "" {
			report.ReasonCodes = append(
				report.ReasonCodes,
				"INACTIVE_ROUTE_INPUTS_PRESENT",
			)
			break
		}
		missing := false
		if inputs.ForkPolicy == nil || inputs.ForkPolicySHA256 == "" {
			report.ReasonCodes = append(report.ReasonCodes, "FORK_POLICY_MISSING")
			missing = true
		}
		if inputs.ForkRelease == nil || inputs.ForkReleaseSHA256 == "" {
			report.ReasonCodes = append(report.ReasonCodes, "FORK_RELEASE_MISSING")
			missing = true
		}
		if inputs.ForkChoice == nil || inputs.ForkChoiceSHA256 == "" {
			report.ReasonCodes = append(report.ReasonCodes, "FORK_CHOICE_MISSING")
			missing = true
		}
		if inputs.Genesis == nil || inputs.GenesisSHA256 == "" {
			report.ReasonCodes = append(report.ReasonCodes, "FORK_GENESIS_MISSING")
			missing = true
		}
		if len(inputs.CompilerReports) != 2 ||
			len(inputs.CompilerReportSHA256s) != 2 {
			report.ReasonCodes = append(report.ReasonCodes, "COMPILER_REPORTS_MISSING")
			missing = true
		}
		if missing {
			break
		}
		if !matchesExactCanonicalDigest(
			*inputs.ForkPolicy,
			inputs.ForkPolicySHA256,
		) {
			report.ReasonCodes = append(report.ReasonCodes, "FORK_POLICY_INVALID")
			break
		}
		if !matchesExactCanonicalDigest(
			*inputs.ForkRelease,
			inputs.ForkReleaseSHA256,
		) {
			report.ReasonCodes = append(report.ReasonCodes, "FORK_RELEASE_INVALID")
			break
		}
		if !matchesExactCanonicalDigest(
			*inputs.ForkChoice,
			inputs.ForkChoiceSHA256,
		) {
			report.ReasonCodes = append(report.ReasonCodes, "FORK_CHOICE_INVALID")
			break
		}
		if err := validateForkPolicy(
			*inputs.ForkPolicy,
			inputs.Assessment,
			inputs.AssessmentSHA256,
		); err != nil {
			report.ReasonCodes = append(report.ReasonCodes, "FORK_POLICY_INVALID")
			break
		}
		if err := validateForkRelease(
			*inputs.ForkRelease,
			*inputs.ForkPolicy,
			inputs.ForkPolicySHA256,
			inputs.Assessment,
			inputs.AssessmentSHA256,
		); err != nil {
			report.ReasonCodes = append(report.ReasonCodes, "FORK_RELEASE_INVALID")
			break
		}
		if err := validateForkArtifacts(
			*inputs.Genesis,
			inputs.GenesisSHA256,
			inputs.CompilerReports,
			inputs.CompilerReportSHA256s,
			*inputs.ForkRelease,
			*inputs.ForkPolicy,
			inputs.ForkPolicySHA256,
			inputs.Assessment,
			inputs.AssessmentSHA256,
		); err != nil {
			report.ReasonCodes = append(report.ReasonCodes, "FORK_ARTIFACTS_INVALID")
			break
		}
		if err := validateForkChoice(
			*inputs.ForkChoice,
			*inputs.ForkPolicy,
			inputs.ForkPolicySHA256,
			*inputs.ForkRelease,
			inputs.ForkReleaseSHA256,
			inputs.Assessment,
			inputs.AssessmentSHA256,
		); err != nil {
			report.ReasonCodes = append(report.ReasonCodes, "FORK_CHOICE_INVALID")
			break
		}
		report.Decision = decisionFork
		report.SelectedSHA256 = inputs.ForkChoiceSHA256
	default:
		return GateReport{}, errors.New("internal recovery route is invalid")
	}
	return sealGateReport(report)
}

func matchesExactCanonicalDigest(value any, expected string) bool {
	if validateSHA256("exact canonical input", expected, false) != nil {
		return false
	}
	actual, err := canonicalDigest(value)
	return err == nil && actual == expected
}

func sealGateReport(report GateReport) (GateReport, error) {
	report.ReasonCodes = sortedUnique(report.ReasonCodes)
	report.ReportSHA256 = ""
	report.GateID = ""
	gateID, err := expectedGateID(report.RequiredRoute, report.InputDigests)
	if err != nil {
		return GateReport{}, err
	}
	report.GateID = gateID
	reportDigest, err := canonicalDigest(report)
	if err != nil {
		return GateReport{}, err
	}
	report.ReportSHA256 = reportDigest
	return report, nil
}

func expectedGateID(
	requiredRoute string,
	inputDigests GateInputDigests,
) (string, error) {
	idMaterial := struct {
		Schema        string           `json:"schema"`
		RequiredRoute string           `json:"required_route"`
		InputDigests  GateInputDigests `json:"input_digests"`
	}{
		Schema:        gateReportSchema,
		RequiredRoute: requiredRoute,
		InputDigests:  inputDigests,
	}
	idDigest, err := canonicalDigest(idMaterial)
	if err != nil {
		return "", err
	}
	return "gate-" + idDigest[:32], nil
}

// validateGateReportEnvelope validates only the deterministic report
// envelope. It is deliberately not an authorization check: a GO decision is
// authoritative only after verifyGateReportWithInputs re-evaluates the exact
// pinned source documents.
func validateGateReportEnvelope(report GateReport) error {
	if report.Schema != gateReportSchema {
		return errors.New("gate report schema is invalid")
	}
	if report.RequiredRoute != routeControlled && report.RequiredRoute != routeFork {
		return errors.New("gate report route is invalid")
	}
	switch report.Decision {
	case decisionControlled, decisionFork, decisionNoGo:
	default:
		return errors.New("gate report decision is invalid")
	}
	if report.ReasonCodes == nil ||
		len(report.ReasonCodes) > maxCollectionEntries ||
		!sort.StringsAreSorted(report.ReasonCodes) {
		return errors.New("gate report reason codes are not canonical")
	}
	for index, reason := range report.ReasonCodes {
		if err := validateLabel("gate report reason code", reason, 128); err != nil {
			return err
		}
		if index == 0 {
			continue
		}
		if report.ReasonCodes[index] == report.ReasonCodes[index-1] {
			return errors.New("gate report reason codes are not unique")
		}
	}
	if err := validateGateInputDigests(report.InputDigests); err != nil {
		return err
	}
	if report.Decision == decisionNoGo {
		if report.SelectedSHA256 != "" {
			return errors.New("NO_GO report must not select an artifact")
		}
		if len(report.ReasonCodes) == 0 {
			return errors.New("NO_GO report must contain a reason")
		}
	} else {
		if report.EvaluatedAt == "" || report.IncidentID == "" ||
			report.OldChainID == "" {
			return errors.New("GO report subject fields are required")
		}
		if err := validateTimestamp("gate report evaluated_at", report.EvaluatedAt); err != nil {
			return err
		}
		if err := validateLabel("gate report incident ID", report.IncidentID, 256); err != nil {
			return err
		}
		if err := validateLabel("gate report old chain ID", report.OldChainID, 256); err != nil {
			return err
		}
		if len(report.ReasonCodes) == 0 {
			return errors.New("GO report must preserve its route reason")
		}
	}
	switch report.Decision {
	case decisionControlled:
		if report.RequiredRoute != routeControlled ||
			report.SelectedSHA256 == "" ||
			report.SelectedSHA256 != report.InputDigests.ControlledSHA256 {
			return errors.New("controlled GO report selection is inconsistent")
		}
		if report.InputDigests.CustodyPolicySHA256 == "" ||
			report.InputDigests.AssessmentSHA256 == "" ||
			report.InputDigests.ControlledPolicySHA256 == "" ||
			report.InputDigests.ControlledSHA256 == "" {
			return errors.New("controlled GO report is missing a required input digest")
		}
		if report.InputDigests.ForkPolicySHA256 != "" ||
			report.InputDigests.ForkReleaseSHA256 != "" ||
			report.InputDigests.ForkChoiceSHA256 != "" ||
			report.InputDigests.GenesisSHA256 != "" ||
			len(report.InputDigests.CompilerReportSHA256s) != 0 {
			return errors.New("controlled GO report contains inactive fork inputs")
		}
	case decisionFork:
		if report.RequiredRoute != routeFork ||
			report.SelectedSHA256 == "" ||
			report.SelectedSHA256 != report.InputDigests.ForkChoiceSHA256 {
			return errors.New("fork GO report selection is inconsistent")
		}
		if report.InputDigests.CustodyPolicySHA256 == "" ||
			report.InputDigests.AssessmentSHA256 == "" ||
			report.InputDigests.ForkPolicySHA256 == "" ||
			report.InputDigests.ForkReleaseSHA256 == "" ||
			report.InputDigests.ForkChoiceSHA256 == "" ||
			report.InputDigests.GenesisSHA256 == "" ||
			len(report.InputDigests.CompilerReportSHA256s) != 2 {
			return errors.New("fork GO report is missing a required input digest")
		}
		if report.InputDigests.ControlledPolicySHA256 != "" ||
			report.InputDigests.ControlledSHA256 != "" {
			return errors.New("fork GO report contains inactive controlled inputs")
		}
	}
	expectedID, err := expectedGateID(report.RequiredRoute, report.InputDigests)
	if err != nil || report.GateID != expectedID {
		return errors.New("gate report ID mismatch")
	}
	actual := report.ReportSHA256
	report.ReportSHA256 = ""
	canonical, err := json.Marshal(report)
	if err != nil || actual == "" || digestBytes(canonical) != actual {
		return errors.New("gate report self hash mismatch")
	}
	return nil
}

func validateGateInputDigests(digests GateInputDigests) error {
	for name, value := range map[string]string{
		"gate custody policy":    digests.CustodyPolicySHA256,
		"gate assessment":        digests.AssessmentSHA256,
		"gate controlled policy": digests.ControlledPolicySHA256,
		"gate controlled plan":   digests.ControlledSHA256,
		"gate fork policy":       digests.ForkPolicySHA256,
		"gate fork release":      digests.ForkReleaseSHA256,
		"gate fork choice":       digests.ForkChoiceSHA256,
		"gate fork genesis":      digests.GenesisSHA256,
	} {
		if err := validateSHA256(name+" digest", value, true); err != nil {
			return err
		}
	}
	if err := validateSortedHashes(
		"gate compiler report digests",
		digests.CompilerReportSHA256s,
	); err != nil {
		return err
	}
	return nil
}

// verifyGateReportWithInputs is the only authoritative report verifier. It
// re-runs the fail-closed decision over the exact pinned inputs and requires a
// byte-for-byte identical report.
func verifyGateReportWithInputs(
	report GateReport,
	inputs EvaluationInputs,
) error {
	if err := validateGateReportEnvelope(report); err != nil {
		return err
	}
	expected, err := evaluate(inputs)
	if err != nil {
		return fmt.Errorf("gate report re-evaluation failed: %w", err)
	}
	expectedJSON, err := json.Marshal(expected)
	if err != nil {
		return errors.New("expected gate report cannot be encoded")
	}
	actualJSON, err := json.Marshal(report)
	if err != nil {
		return errors.New("gate report cannot be encoded")
	}
	if !bytes.Equal(expectedJSON, actualJSON) {
		return errors.New("gate report does not match authoritative re-evaluation")
	}
	return nil
}
