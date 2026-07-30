package main

import (
	"crypto/ed25519"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"slices"
	"sort"
	"strconv"
	"strings"
)

type statementBuilder func(Approval) (string, error)

func validateCustodyAssessment(
	assessment CustodyAssessment,
	policy SignerPolicy,
	policyFileSHA256 string,
) error {
	if assessment.Schema != custodyAssessmentSchema {
		return fmt.Errorf("custody assessment schema must be %q", custodyAssessmentSchema)
	}
	if assessment.SignerPolicySHA256 != policyFileSHA256 {
		return errors.New("custody assessment does not bind the pinned signer policy")
	}
	if err := validateSignerPolicy(
		policy,
		signerPurposeCustody,
		assessment.IncidentID,
		assessment.ChainID,
		"",
		requiredCustodyRoles,
	); err != nil {
		return fmt.Errorf("custody signer policy: %w", err)
	}
	for name, value := range map[string]string{
		"custody assessment ID": assessment.AssessmentID,
		"custody incident ID":   assessment.IncidentID,
		"custody chain ID":      assessment.ChainID,
	} {
		if err := validateLabel(name, value, 256); err != nil {
			return err
		}
	}
	if err := validateTimestamp("custody evaluated_at", assessment.EvaluatedAt); err != nil {
		return err
	}
	if err := validateCheckpoint(assessment.Checkpoint); err != nil {
		return err
	}
	evaluatedAt, _ := parseCanonicalUTCTime(
		"custody evaluated_at",
		assessment.EvaluatedAt,
	)
	checkpointTime, _ := parseCanonicalUTCTime(
		"checkpoint block time",
		assessment.Checkpoint.BlockTime,
	)
	if !evaluatedAt.After(checkpointTime) {
		return errors.New("custody evaluation must be strictly after the checkpoint time")
	}
	if assessment.ExposureWindow.FirstPossiblyExposedHeight == 0 ||
		assessment.ExposureWindow.FirstPossiblyExposedHeight >
			assessment.Checkpoint.Height ||
		assessment.ExposureWindow.LastReviewedHeight !=
			assessment.Checkpoint.Height {
		return errors.New("custody exposure window is invalid")
	}
	if err := validateValidatorIdentity("old validator", assessment.OldValidator); err != nil {
		return err
	}
	if err := validateCustodyFindings(assessment.Findings); err != nil {
		return err
	}
	if err := validatePrivilegedIdentityAssessments(assessment); err != nil {
		return err
	}
	if err := validateEvidence(assessment.Evidence); err != nil {
		return err
	}
	if err := validateCustodyEvidenceLinks(assessment); err != nil {
		return err
	}
	if err := validateSortedUniqueStrings(
		"prohibited consensus public keys",
		assessment.ProhibitedConsensusPublicKeys,
		func(name, value string) error { return validatePublicKey(name, value) },
	); err != nil {
		return err
	}
	if !containsString(
		assessment.ProhibitedConsensusPublicKeys,
		assessment.OldValidator.ConsensusPublicKey,
	) {
		return errors.New("prohibited consensus public keys must include the old validator key")
	}
	if err := validateSortedLabels(
		"prohibited privileged identities",
		assessment.ProhibitedPrivilegedIdentities,
	); err != nil {
		return err
	}
	if err := validateApprovalShape(assessment.Approvals); err != nil {
		return err
	}
	if err := verifySignerPolicyApprovals(
		assessment.Approvals,
		policy,
		forbiddenApprovalPublicKeys(assessment),
		assessment.ProhibitedPrivilegedIdentities,
		func(approval Approval) (string, error) {
			return custodyApprovalStatement(assessment, approval)
		},
	); err != nil {
		return fmt.Errorf("custody approvals: %w", err)
	}
	return verifyCustodySeal(assessment)
}

func validateCustodyFindings(findings []CustodyFinding) error {
	if findings == nil {
		return errors.New("custody findings must be [] rather than null")
	}
	if len(findings) > len(requiredCustodyFindings) {
		return errors.New("custody findings contain unsupported entries")
	}
	if !sort.SliceIsSorted(findings, func(i, j int) bool {
		return findings[i].ID < findings[j].ID
	}) {
		return errors.New("custody findings must be sorted by ID")
	}
	seen := make(map[string]bool, len(findings))
	for _, finding := range findings {
		if seen[finding.ID] {
			return errors.New("custody findings must be unique")
		}
		seen[finding.ID] = true
		if !containsString(requiredCustodyFindings, finding.ID) {
			return errors.New("custody finding ID is unsupported")
		}
		switch finding.Result {
		case custodyResultPass, custodyResultFail, custodyResultUnknown:
		default:
			return errors.New("custody finding result must be PASS, FAIL, or UNKNOWN")
		}
		if err := validateSHA256("custody finding evidence digest", finding.EvidenceSHA256, false); err != nil {
			return err
		}
	}
	return nil
}

func validatePrivilegedIdentityAssessments(assessment CustodyAssessment) error {
	entries := assessment.PrivilegedIdentityAssessments
	if entries == nil || len(entries) == 0 {
		return errors.New("privileged identity assessments must be a non-empty array")
	}
	if len(entries) > maxCollectionEntries {
		return errors.New("privileged identity assessments contains too many entries")
	}
	if !sort.SliceIsSorted(entries, func(i, j int) bool {
		if entries[i].Kind != entries[j].Kind {
			return entries[i].Kind < entries[j].Kind
		}
		return entries[i].Identity < entries[j].Identity
	}) {
		return errors.New("privileged identity assessments must be sorted by kind then identity")
	}

	seenTuples := make(map[string]bool, len(entries))
	dispositionByIdentity := make(map[string]string, len(entries))
	retired := make(map[string]bool)
	sdkOperatorCount := 0
	var sdkOperator PrivilegedIdentityAssessment
	for _, entry := range entries {
		if err := validateLabel("privileged identity kind", entry.Kind, 128); err != nil {
			return err
		}
		if err := validateLabel("privileged identity", entry.Identity, 256); err != nil {
			return err
		}
		switch entry.Result {
		case custodyResultPass, custodyResultFail, custodyResultUnknown:
		default:
			return errors.New("privileged identity result must be PASS, FAIL, or UNKNOWN")
		}
		if err := validateSHA256(
			"privileged identity evidence digest",
			entry.EvidenceSHA256,
			false,
		); err != nil {
			return err
		}
		switch entry.Disposition {
		case privilegedDispositionRetain:
			if entry.Result != custodyResultPass {
				return errors.New("privileged identity RETAIN requires PASS")
			}
		case privilegedDispositionRetire:
			retired[entry.Identity] = true
		default:
			return errors.New("privileged identity disposition must be RETAIN or RETIRE")
		}
		if entry.Result != custodyResultPass &&
			entry.Disposition != privilegedDispositionRetire {
			return errors.New("privileged identity FAIL or UNKNOWN must RETIRE")
		}

		tuple := entry.Kind + "\x00" + entry.Identity
		if seenTuples[tuple] {
			return errors.New("privileged identity assessment tuples must be unique")
		}
		seenTuples[tuple] = true
		if existing, present := dispositionByIdentity[entry.Identity]; present &&
			existing != entry.Disposition {
			return errors.New("one privileged identity cannot be both retained and retired")
		}
		dispositionByIdentity[entry.Identity] = entry.Disposition

		if entry.Kind == privilegedKindSDKOperator {
			sdkOperatorCount++
			sdkOperator = entry
		}
	}
	if sdkOperatorCount != 1 ||
		sdkOperator.Identity != assessment.OldValidator.SDKOperatorAddress {
		return errors.New("exactly one sdk-operator assessment must match the old validator")
	}
	for _, finding := range assessment.Findings {
		if finding.ID == "old-sdk-operator-key-safe" &&
			finding.Result != sdkOperator.Result {
			return errors.New("sdk-operator assessment and custody finding results must match")
		}
	}

	derived := make([]string, 0, len(retired))
	for identity := range retired {
		derived = append(derived, identity)
	}
	sort.Strings(derived)
	if !slices.Equal(derived, assessment.ProhibitedPrivilegedIdentities) {
		return errors.New("prohibited privileged identities must exactly equal RETIRE assessments")
	}
	return nil
}

func custodyRoute(assessment CustodyAssessment) (string, []string) {
	results := make(map[string]string, len(assessment.Findings))
	for _, finding := range assessment.Findings {
		results[finding.ID] = finding.Result
	}
	reasons := make([]string, 0, 3)
	for _, required := range requiredCustodyFindings {
		switch results[required] {
		case custodyResultPass:
		case custodyResultFail:
			reasons = append(reasons, "CUSTODY_FINDING_FAILED")
		case custodyResultUnknown:
			reasons = append(reasons, "CUSTODY_FINDING_UNKNOWN")
		default:
			reasons = append(reasons, "CUSTODY_FINDING_MISSING")
		}
	}
	for _, privileged := range assessment.PrivilegedIdentityAssessments {
		switch privileged.Result {
		case custodyResultPass:
		case custodyResultFail:
			reasons = append(reasons, "PRIVILEGED_IDENTITY_FAILED")
		case custodyResultUnknown:
			reasons = append(reasons, "PRIVILEGED_IDENTITY_UNKNOWN")
		default:
			reasons = append(reasons, "PRIVILEGED_IDENTITY_INVALID")
		}
		if privileged.Disposition == privilegedDispositionRetire {
			reasons = append(
				reasons,
				"PRIVILEGED_IDENTITY_RETIRE_REQUIRED",
			)
		}
	}
	reasons = sortedUnique(reasons)
	if len(reasons) != 0 {
		return routeFork, reasons
	}
	return routeControlled, []string{"CUSTODY_ALL_FINDINGS_PASS"}
}

func forbiddenApprovalPublicKeys(
	assessment CustodyAssessment,
) []string {
	result := append(
		[]string{},
		assessment.ProhibitedConsensusPublicKeys...,
	)
	result = append(result, assessment.OldValidator.NodePublicKey)
	return sortedUnique(result)
}

func validateControlledTransition(
	plan ControlledTransition,
	assessment CustodyAssessment,
	assessmentFileSHA256 string,
	policy SignerPolicy,
	policyFileSHA256 string,
) error {
	if plan.Schema != controlledTransitionSchema {
		return fmt.Errorf("controlled transition schema must be %q", controlledTransitionSchema)
	}
	for name, value := range map[string]string{
		"controlled plan ID":     plan.PlanID,
		"controlled incident ID": plan.IncidentID,
		"controlled chain ID":    plan.ChainID,
	} {
		if err := validateLabel(name, value, 256); err != nil {
			return err
		}
	}
	if plan.AssessmentSHA256 != assessmentFileSHA256 ||
		plan.SignerPolicySHA256 != policyFileSHA256 ||
		plan.IncidentID != assessment.IncidentID ||
		plan.ChainID != assessment.ChainID ||
		plan.Checkpoint != assessment.Checkpoint {
		return errors.New("controlled transition binding does not match custody assessment")
	}
	if err := validateSignerPolicy(
		policy,
		signerPurposeControlled,
		assessment.IncidentID,
		assessment.ChainID,
		assessmentFileSHA256,
		requiredControlledRoles,
	); err != nil {
		return fmt.Errorf("controlled signer policy: %w", err)
	}
	if plan.AdmissionState != requiredAdmissionState {
		return errors.New("controlled admission state must be OPEN")
	}
	if plan.BondTransactionHeight == 0 ||
		plan.BondTransactionHeight <= plan.Checkpoint.Height ||
		plan.ValidatorUpdateWaitBlocks != requiredValidatorUpdateWait ||
		plan.BondTransactionHeight > ^uint64(0)-requiredValidatorUpdateWait ||
		plan.ExpectedActivationHeight !=
			plan.BondTransactionHeight+requiredValidatorUpdateWait {
		return errors.New("controlled activation must be exactly B+2")
	}
	consenting, err := parseCanonicalPositiveInteger("controlled consenting power", plan.ConsentingPower)
	if err != nil {
		return err
	}
	total, err := parseCanonicalPositiveInteger("controlled total bonded power", plan.TotalBondedPower)
	if err != nil {
		return err
	}
	if consenting.Cmp(total) > 0 ||
		new(big.Int).Mul(consenting, big.NewInt(3)).Cmp(
			new(big.Int).Mul(total, big.NewInt(2)),
		) <= 0 {
		return errors.New("controlled consenting power must be strictly greater than two thirds")
	}
	if err := validateSHA256("controlled power snapshot", plan.PowerSnapshotSHA256, false); err != nil {
		return err
	}
	if err := validateStakeInventory(plan.StakeInventory); err != nil {
		return err
	}
	if err := validateValidatorIdentity("new validator", plan.NewValidator); err != nil {
		return err
	}
	if err := validateFreshIdentity(
		plan.NewValidator,
		assessment,
		assessment.ProhibitedConsensusPublicKeys,
		assessment.ProhibitedPrivilegedIdentities,
	); err != nil {
		return err
	}
	if err := validateRecoveryArtifactDigests(
		plan.BinarySHA256,
		plan.ImageSHA256,
		plan.ProvenanceSHA256,
		plan.SBOMSHA256,
		plan.RehearsalSHA256,
		plan.TopologySHA256,
		plan.JournalHeadSHA256,
	); err != nil {
		return err
	}
	if err := validateEvidence(plan.Evidence); err != nil {
		return err
	}
	if err := validateControlledEvidenceLinks(plan); err != nil {
		return err
	}
	if err := validateApprovalShape(plan.Approvals); err != nil {
		return err
	}
	if err := verifySignerPolicyApprovals(
		plan.Approvals,
		policy,
		forbiddenApprovalPublicKeys(assessment),
		assessment.ProhibitedPrivilegedIdentities,
		func(approval Approval) (string, error) {
			return controlledApprovalStatement(plan, approval)
		},
	); err != nil {
		return fmt.Errorf("controlled approvals: %w", err)
	}
	return verifyControlledSeal(plan)
}

func validateSignerPolicy(
	policy SignerPolicy,
	purpose,
	incidentID,
	chainID,
	assessmentSHA256 string,
	mandatoryRoles []string,
) error {
	if policy.Schema != signerPolicySchema {
		return fmt.Errorf("signer policy schema must be %q", signerPolicySchema)
	}
	if err := validateLabel("signer policy ID", policy.PolicyID, 256); err != nil {
		return err
	}
	if policy.Purpose != purpose ||
		policy.IncidentID != incidentID ||
		policy.ChainID != chainID ||
		policy.AssessmentSHA256 != assessmentSHA256 {
		return errors.New("signer policy subject binding is invalid")
	}
	if !slices.Equal(policy.RequiredRoles, mandatoryRoles) {
		return errors.New("signer policy required roles must exactly match the mandatory role set")
	}
	if err := validateSortedLabels("signer policy required roles", policy.RequiredRoles); err != nil {
		return err
	}
	minimum := uint64(len(mandatoryRoles))
	if policy.MinimumApprovals < minimum ||
		policy.MinimumDistinctIdentities < minimum ||
		policy.MinimumDistinctControlDomains < minimum {
		return errors.New("signer policy thresholds must cover every mandatory role")
	}
	if policy.Signers == nil || len(policy.Signers) == 0 ||
		len(policy.Signers) > maxCollectionEntries {
		return errors.New("signer policy signers must be a bounded non-empty array")
	}
	if !sort.SliceIsSorted(policy.Signers, func(i, j int) bool {
		return trustedSignerLess(policy.Signers[i], policy.Signers[j])
	}) {
		return errors.New("signer policy signers must be sorted by role, identity, control domain, then public key")
	}
	tuples := make(map[string]bool, len(policy.Signers))
	identities := make(map[string]bool)
	domains := make(map[string]bool)
	for _, signer := range policy.Signers {
		if err := validateTrustedSigner(signer); err != nil {
			return err
		}
		if !containsString(policy.RequiredRoles, signer.Role) {
			return errors.New("signer policy contains a signer for an unauthorized role")
		}
		key := trustedSignerKey(
			signer.Role,
			signer.Identity,
			signer.ControlDomain,
			signer.PublicKey,
		)
		if tuples[key] {
			return errors.New("signer policy tuples must be unique")
		}
		tuples[key] = true
		identities[signer.Identity] = true
		domains[signer.ControlDomain] = true
	}
	if uint64(len(policy.Signers)) < policy.MinimumApprovals ||
		uint64(len(identities)) < policy.MinimumDistinctIdentities ||
		uint64(len(domains)) < policy.MinimumDistinctControlDomains {
		return errors.New("signer policy signer set cannot satisfy its thresholds")
	}
	for _, role := range policy.RequiredRoles {
		found := false
		for _, signer := range policy.Signers {
			if signer.Role == role {
				found = true
				break
			}
		}
		if !found {
			return errors.New("signer policy required role has no trusted signer")
		}
	}
	return nil
}

func validateTrustedReproducers(
	reproducers []TrustedReproducer,
	prohibitedPublicKeys []string,
	prohibitedIdentities []string,
) error {
	if len(reproducers) != 2 {
		return errors.New("fork policy requires exactly two trusted independent reproducers")
	}
	if !sort.SliceIsSorted(reproducers, func(i, j int) bool {
		return trustedReproducerLess(reproducers[i], reproducers[j])
	}) {
		return errors.New("trusted reproducers must be sorted by identity, control domain, then public key")
	}
	identities := make(map[string]bool)
	domains := make(map[string]bool)
	publicKeys := make(map[string]bool)
	for _, reproducer := range reproducers {
		if err := validateLabel("trusted reproducer identity", reproducer.Identity, 256); err != nil {
			return err
		}
		if err := validateLabel("trusted reproducer control domain", reproducer.ControlDomain, 256); err != nil {
			return err
		}
		if err := validatePublicKey("trusted reproducer public key", reproducer.PublicKey); err != nil {
			return err
		}
		if containsString(prohibitedPublicKeys, reproducer.PublicKey) {
			return errors.New("trusted reproducer uses a prohibited old consensus public key")
		}
		if containsString(prohibitedIdentities, reproducer.Identity) {
			return errors.New("trusted reproducer uses a prohibited privileged identity")
		}
		if identities[reproducer.Identity] ||
			domains[reproducer.ControlDomain] ||
			publicKeys[reproducer.PublicKey] {
			return errors.New("trusted reproducers must use distinct identities, control domains, and public keys")
		}
		identities[reproducer.Identity] = true
		domains[reproducer.ControlDomain] = true
		publicKeys[reproducer.PublicKey] = true
	}
	return nil
}

func trustedReproducerLess(left, right TrustedReproducer) bool {
	if left.Identity != right.Identity {
		return left.Identity < right.Identity
	}
	if left.ControlDomain != right.ControlDomain {
		return left.ControlDomain < right.ControlDomain
	}
	return left.PublicKey < right.PublicKey
}

func validateStakeInventory(inventory StakeInventory) error {
	for name, pagination := range map[string]PaginatedInventory{
		"delegations":   inventory.Delegations,
		"unbondings":    inventory.Unbondings,
		"redelegations": inventory.Redelegations,
	} {
		if pagination.PageSHA256s == nil || len(pagination.PageSHA256s) == 0 {
			return fmt.Errorf("%s pagination must include at least one response page", name)
		}
		if len(pagination.PageSHA256s) > maxCollectionEntries {
			return fmt.Errorf("%s pagination contains too many pages", name)
		}
		seen := make(map[string]bool, len(pagination.PageSHA256s))
		for _, digest := range pagination.PageSHA256s {
			if err := validateSHA256(name+" page digest", digest, false); err != nil {
				return err
			}
			if seen[digest] {
				return fmt.Errorf("%s pagination page digests must be unique", name)
			}
			seen[digest] = true
		}
		if !pagination.Complete || pagination.NextKey != "" {
			return fmt.Errorf("%s pagination must be complete with an empty next key", name)
		}
	}
	return nil
}

func validateRecoveryArtifactDigests(values ...string) error {
	names := []string{
		"binary", "image", "provenance", "SBOM", "rehearsal", "topology", "journal head",
	}
	for index, value := range values {
		if err := validateSHA256(names[index]+" digest", value, false); err != nil {
			return err
		}
	}
	return nil
}

func requireEvidenceLink(
	evidence []Evidence,
	digest,
	expectedType string,
) error {
	matches := 0
	for _, item := range evidence {
		if item.SHA256 != digest {
			continue
		}
		if expectedType != "" && item.Type != expectedType {
			return fmt.Errorf("evidence digest is linked with the wrong type; expected %s", expectedType)
		}
		matches++
	}
	if matches != 1 {
		return errors.New("required digest must resolve to exactly one evidence entry")
	}
	return nil
}

func validateCustodyEvidenceLinks(assessment CustodyAssessment) error {
	for _, link := range []struct {
		digest       string
		evidenceType string
	}{
		{assessment.Checkpoint.BlockIDSHA256, "checkpoint-block-id"},
		{assessment.Checkpoint.AppHashSHA256, "checkpoint-app-hash"},
		{assessment.Checkpoint.SignedCommitSHA256, "checkpoint-signed-commit"},
		{assessment.Checkpoint.ValidatorSetSHA256, "checkpoint-validator-set"},
		{assessment.OldValidator.ValidatorKeySHA256, "old-validator-key-file"},
		{assessment.OldValidator.NodeKeySHA256, "old-node-key-file"},
		{assessment.OldValidator.SigningStateSHA256, "old-signing-state"},
	} {
		if err := requireEvidenceLink(
			assessment.Evidence,
			link.digest,
			link.evidenceType,
		); err != nil {
			return err
		}
	}
	for _, finding := range assessment.Findings {
		if err := requireEvidenceLink(
			assessment.Evidence,
			finding.EvidenceSHA256,
			"custody-finding",
		); err != nil {
			return err
		}
	}
	for _, privileged := range assessment.PrivilegedIdentityAssessments {
		if err := requireEvidenceLink(
			assessment.Evidence,
			privileged.EvidenceSHA256,
			"privileged-identity-assessment",
		); err != nil {
			return err
		}
	}
	return nil
}

func validateControlledEvidenceLinks(plan ControlledTransition) error {
	typed := []struct {
		digest       string
		evidenceType string
	}{
		{plan.PowerSnapshotSHA256, "power-snapshot"},
		{plan.BinarySHA256, "binary"},
		{plan.ImageSHA256, "image"},
		{plan.ProvenanceSHA256, "provenance"},
		{plan.SBOMSHA256, "sbom"},
		{plan.RehearsalSHA256, "rehearsal"},
		{plan.TopologySHA256, "topology"},
		{plan.JournalHeadSHA256, "journal-head"},
		{plan.NewValidator.ValidatorKeySHA256, "validator-key-file"},
		{plan.NewValidator.NodeKeySHA256, "node-key-file"},
		{plan.NewValidator.SigningStateSHA256, "signing-state"},
	}
	for _, link := range typed {
		if err := requireEvidenceLink(plan.Evidence, link.digest, link.evidenceType); err != nil {
			return err
		}
	}
	for _, inventory := range []struct {
		pagination   PaginatedInventory
		evidenceType string
	}{
		{plan.StakeInventory.Delegations, "delegations-page"},
		{plan.StakeInventory.Unbondings, "unbondings-page"},
		{plan.StakeInventory.Redelegations, "redelegations-page"},
	} {
		for _, digest := range inventory.pagination.PageSHA256s {
			if err := requireEvidenceLink(plan.Evidence, digest, inventory.evidenceType); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateForkReleaseEvidenceLinks(release ForkRelease) error {
	typed := []struct {
		digest       string
		evidenceType string
	}{
		{release.SourceExportSHA256, "source-export"},
		{release.RewriteToolSHA256, "rewrite-tool"},
		{release.RewritePolicyFileSHA256, "rewrite-policy"},
		{release.GenesisSHA256, "genesis"},
		{release.SupplyReconciliationSHA256, "supply-reconciliation"},
		{release.IBCReconciliationSHA256, "ibc-reconciliation"},
		{release.ModuleReconciliationSHA256, "module-reconciliation"},
		{release.BinarySHA256, "binary"},
		{release.ImageSHA256, "image"},
		{release.ProvenanceSHA256, "provenance"},
		{release.SBOMSHA256, "sbom"},
		{release.RehearsalSHA256, "rehearsal"},
		{release.TopologySHA256, "topology"},
		{release.JournalHeadSHA256, "journal-head"},
	}
	for _, link := range typed {
		if err := requireEvidenceLink(release.Evidence, link.digest, link.evidenceType); err != nil {
			return err
		}
	}
	for _, validator := range release.NewValidators {
		for _, link := range []struct {
			digest       string
			evidenceType string
		}{
			{validator.ValidatorKeySHA256, "validator-key-file"},
			{validator.NodeKeySHA256, "node-key-file"},
			{validator.SigningStateSHA256, "signing-state"},
		} {
			if err := requireEvidenceLink(
				release.Evidence,
				link.digest,
				link.evidenceType,
			); err != nil {
				return err
			}
		}
	}
	for _, reproduction := range release.GenesisReproductions {
		if err := requireEvidenceLink(
			release.Evidence,
			reproduction.CompilerReportFileSHA256,
			"fork-genesis-compiler-report",
		); err != nil {
			return err
		}
	}
	return nil
}

func validateForkPolicy(
	policy ForkPolicy,
	assessment CustodyAssessment,
	assessmentFileSHA256 string,
) error {
	if policy.Schema != forkPolicySchema {
		return fmt.Errorf("fork policy schema must be %q", forkPolicySchema)
	}
	if err := validateLabel("fork policy ID", policy.PolicyID, 256); err != nil {
		return err
	}
	if policy.AssessmentSHA256 != assessmentFileSHA256 ||
		policy.IncidentID != assessment.IncidentID ||
		policy.OldChainID != assessment.ChainID {
		return errors.New("fork policy binding does not match custody assessment")
	}
	findings := make(map[string]string, len(assessment.Findings))
	for _, finding := range assessment.Findings {
		findings[finding.ID] = finding.Result
	}
	for _, required := range requiredConsensusOnlySafeFindings {
		if findings[required] != custodyResultPass {
			return errors.New("consensus-key-only policy requires retained history and authority findings to PASS")
		}
	}
	if err := validateSortedLabels("fork policy required roles", policy.RequiredRoles); err != nil {
		return err
	}
	for _, required := range requiredForkRoles {
		if !containsString(policy.RequiredRoles, required) {
			return errors.New("fork policy is missing a mandatory role")
		}
	}
	requiredCount := uint64(len(requiredForkRoles))
	if policy.MinimumApprovals < requiredCount ||
		policy.MinimumDistinctIdentities < requiredCount ||
		policy.MinimumDistinctControlDomains < requiredCount {
		return errors.New("fork policy thresholds must cover every mandatory independent role")
	}
	if !slices.Equal(
		policy.ProhibitedConsensusPublicKeys,
		assessment.ProhibitedConsensusPublicKeys,
	) || !slices.Equal(
		policy.ProhibitedPrivilegedIdentities,
		assessment.ProhibitedPrivilegedIdentities,
	) {
		return errors.New("fork policy prohibited identities must exactly match custody assessment")
	}
	if len(policy.ProhibitedConsensusPublicKeys) != 1 ||
		policy.ProhibitedConsensusPublicKeys[0] !=
			assessment.OldValidator.ConsensusPublicKey ||
		policy.ProhibitedPrivilegedIdentities == nil ||
		len(policy.ProhibitedPrivilegedIdentities) != 0 {
		return errors.New("consensus-key-only policy requires only the old consensus key prohibited and no retired privileged identity")
	}
	if err := validateSortedUniqueStrings(
		"fork policy prohibited consensus public keys",
		policy.ProhibitedConsensusPublicKeys,
		func(name, value string) error { return validatePublicKey(name, value) },
	); err != nil {
		return err
	}
	if err := validateSortedLabels(
		"fork policy prohibited privileged identities",
		policy.ProhibitedPrivilegedIdentities,
	); err != nil {
		return err
	}
	if policy.Signers == nil || len(policy.Signers) == 0 ||
		len(policy.Signers) > maxCollectionEntries {
		return errors.New("fork policy signers must be a bounded non-empty array")
	}
	if !sort.SliceIsSorted(policy.Signers, func(i, j int) bool {
		return trustedSignerLess(policy.Signers[i], policy.Signers[j])
	}) {
		return errors.New("fork policy signers must be sorted by role, identity, control domain, then public key")
	}
	tuples := make(map[string]bool, len(policy.Signers))
	identities := make(map[string]bool)
	controlDomains := make(map[string]bool)
	for _, signer := range policy.Signers {
		if err := validateTrustedSigner(signer); err != nil {
			return err
		}
		if !containsString(policy.RequiredRoles, signer.Role) {
			return errors.New("fork policy signer role is not required by the policy")
		}
		if containsString(policy.ProhibitedConsensusPublicKeys, signer.PublicKey) ||
			signer.PublicKey == assessment.OldValidator.NodePublicKey ||
			containsString(policy.ProhibitedPrivilegedIdentities, signer.Identity) {
			return errors.New("fork policy signer reuses a prohibited old identity")
		}
		key := signer.Role + "\x00" + signer.Identity + "\x00" +
			signer.ControlDomain + "\x00" + signer.PublicKey
		if tuples[key] {
			return errors.New("fork policy signer tuples must be unique")
		}
		tuples[key] = true
		identities[signer.Identity] = true
		controlDomains[signer.ControlDomain] = true
	}
	if uint64(len(policy.Signers)) < policy.MinimumApprovals ||
		uint64(len(identities)) < policy.MinimumDistinctIdentities ||
		uint64(len(controlDomains)) < policy.MinimumDistinctControlDomains {
		return errors.New("fork policy signer set cannot satisfy its thresholds")
	}
	for _, role := range policy.RequiredRoles {
		found := false
		for _, signer := range policy.Signers {
			if signer.Role == role {
				found = true
				break
			}
		}
		if !found {
			return errors.New("fork policy required role has no trusted signer")
		}
	}
	if err := validateTrustedReproducers(
		policy.IndependentReproducers,
		forbiddenApprovalPublicKeys(assessment),
		policy.ProhibitedPrivilegedIdentities,
	); err != nil {
		return err
	}
	return nil
}

func validateTrustedSigner(signer TrustedSigner) error {
	if err := validateLabel("trusted signer role", signer.Role, 128); err != nil {
		return err
	}
	if err := validateLabel("trusted signer identity", signer.Identity, 256); err != nil {
		return err
	}
	if err := validateLabel("trusted signer control domain", signer.ControlDomain, 256); err != nil {
		return err
	}
	return validatePublicKey("trusted signer public key", signer.PublicKey)
}

func trustedSignerLess(left, right TrustedSigner) bool {
	if left.Role != right.Role {
		return left.Role < right.Role
	}
	if left.Identity != right.Identity {
		return left.Identity < right.Identity
	}
	if left.ControlDomain != right.ControlDomain {
		return left.ControlDomain < right.ControlDomain
	}
	return left.PublicKey < right.PublicKey
}

func validateForkRelease(
	release ForkRelease,
	policy ForkPolicy,
	policyFileSHA256 string,
	assessment CustodyAssessment,
	assessmentFileSHA256 string,
) error {
	if release.Schema != forkReleaseSchema {
		return fmt.Errorf("fork release schema must be %q", forkReleaseSchema)
	}
	if err := validateLabel("fork release ID", release.ReleaseID, 256); err != nil {
		return err
	}
	if release.AssessmentSHA256 != assessmentFileSHA256 ||
		release.ForkPolicySHA256 != policyFileSHA256 ||
		release.IncidentID != assessment.IncidentID ||
		release.OldChainID != assessment.ChainID ||
		release.Checkpoint != assessment.Checkpoint {
		return errors.New("fork release binding does not match its pinned inputs")
	}
	if err := validateLabel("fork release new chain ID", release.NewChainID, 256); err != nil {
		return err
	}
	if release.NewChainID == release.OldChainID {
		return errors.New("fork release requires a unique new chain ID")
	}
	sourceRevision, err := chainRevision(release.OldChainID)
	if err != nil {
		return fmt.Errorf("fork release old chain ID: %w", err)
	}
	targetRevision, err := chainRevision(release.NewChainID)
	if err != nil || targetRevision <= sourceRevision {
		return errors.New("fork release target chain revision must exceed the old chain revision")
	}
	if release.Checkpoint.Height == ^uint64(0) ||
		release.InitialHeight != release.Checkpoint.Height+1 {
		return errors.New("fork release initial height must continue at checkpoint height plus one")
	}
	if release.RewriteProfile != rewriteProfileConsensusOnly {
		return errors.New("fork release rewrite profile is not supported by an audited compiler")
	}
	for name, value := range map[string]string{
		"source export":         release.SourceExportSHA256,
		"rewrite tool":          release.RewriteToolSHA256,
		"rewrite policy file":   release.RewritePolicyFileSHA256,
		"rewrite policy self":   release.RewritePolicySelfSHA256,
		"genesis":               release.GenesisSHA256,
		"supply reconciliation": release.SupplyReconciliationSHA256,
		"IBC reconciliation":    release.IBCReconciliationSHA256,
		"module reconciliation": release.ModuleReconciliationSHA256,
	} {
		if err := validateSHA256(name+" digest", value, false); err != nil {
			return err
		}
	}
	if err := validateGenesisReproductions(
		release.GenesisReproductions,
		release.GenesisSHA256,
		release,
		policy,
	); err != nil {
		return err
	}
	if err := validateForkValidators(release.NewValidators, assessment, policy); err != nil {
		return err
	}
	if len(release.NewValidators) != 1 ||
		release.NewValidators[0].SDKOperatorAddress !=
			assessment.OldValidator.SDKOperatorAddress {
		return errors.New("consensus-key-only release must retain exactly the independently proven-safe old SDK operator")
	}
	if !slices.Equal(
		release.RetiredPrivilegedIdentities,
		policy.ProhibitedPrivilegedIdentities,
	) {
		return errors.New("fork release must retire every prohibited old privileged identity")
	}
	if err := validateSortedLabels(
		"retired privileged identities",
		release.RetiredPrivilegedIdentities,
	); err != nil {
		return err
	}
	if err := validateRecoveryArtifactDigests(
		release.BinarySHA256,
		release.ImageSHA256,
		release.ProvenanceSHA256,
		release.SBOMSHA256,
		release.RehearsalSHA256,
		release.TopologySHA256,
		release.JournalHeadSHA256,
	); err != nil {
		return err
	}
	if err := validateEvidence(release.Evidence); err != nil {
		return err
	}
	if err := validateForkReleaseEvidenceLinks(release); err != nil {
		return err
	}
	if err := validateApprovalShape(release.Approvals); err != nil {
		return err
	}
	if err := verifyPolicyApprovals(
		release.Approvals,
		policy,
		func(approval Approval) (string, error) {
			return forkReleaseApprovalStatement(release, approval)
		},
	); err != nil {
		return fmt.Errorf("fork release approvals: %w", err)
	}
	return verifyForkReleaseSeal(release)
}

func chainRevision(chainID string) (uint64, error) {
	index := strings.LastIndexByte(chainID, '-')
	if index <= 0 || index == len(chainID)-1 {
		return 0, errors.New("chain ID must end in a positive revision")
	}
	revisionText := chainID[index+1:]
	if revisionText[0] == '0' {
		return 0, errors.New("chain revision must be canonical")
	}
	revision, err := strconv.ParseUint(revisionText, 10, 64)
	if err != nil || revision == 0 {
		return 0, errors.New("chain revision must be a positive integer")
	}
	return revision, nil
}

func validateGenesisReproductions(
	reproductions []GenesisReproduction,
	genesisSHA256 string,
	release ForkRelease,
	policy ForkPolicy,
) error {
	if len(reproductions) != 2 {
		return errors.New("fork release requires exactly two independent genesis reproductions")
	}
	if !sort.SliceIsSorted(reproductions, func(i, j int) bool {
		if reproductions[i].Identity != reproductions[j].Identity {
			return reproductions[i].Identity < reproductions[j].Identity
		}
		if reproductions[i].ControlDomain != reproductions[j].ControlDomain {
			return reproductions[i].ControlDomain < reproductions[j].ControlDomain
		}
		return reproductions[i].PublicKey < reproductions[j].PublicKey
	}) {
		return errors.New("genesis reproductions must be sorted by identity, control domain, then public key")
	}
	identities := make(map[string]bool)
	domains := make(map[string]bool)
	publicKeys := make(map[string]bool)
	reports := make(map[string]bool)
	trusted := make(map[string]bool, len(policy.IndependentReproducers))
	for _, reproducer := range policy.IndependentReproducers {
		trusted[reproducer.Identity+"\x00"+reproducer.ControlDomain+"\x00"+reproducer.PublicKey] = true
	}
	for _, reproduction := range reproductions {
		if err := validateLabel("genesis reproducer identity", reproduction.Identity, 256); err != nil {
			return err
		}
		if err := validateLabel("genesis reproducer control domain", reproduction.ControlDomain, 256); err != nil {
			return err
		}
		if err := validatePublicKey("genesis reproducer public key", reproduction.PublicKey); err != nil {
			return err
		}
		if !trusted[reproduction.Identity+"\x00"+
			reproduction.ControlDomain+"\x00"+
			reproduction.PublicKey] {
			return errors.New("genesis reproducer is absent from the pinned fork policy")
		}
		if reproduction.GenesisSHA256 != genesisSHA256 {
			return errors.New("genesis reproductions must exactly match the selected genesis digest")
		}
		if err := validateSHA256(
			"genesis reproduction compiler report file",
			reproduction.CompilerReportFileSHA256,
			false,
		); err != nil {
			return err
		}
		if err := validateSHA256(
			"genesis reproduction statement",
			reproduction.StatementSHA256,
			false,
		); err != nil {
			return err
		}
		if err := validateSignature(
			"genesis reproduction signature",
			reproduction.Signature,
		); err != nil {
			return err
		}
		if identities[reproduction.Identity] ||
			domains[reproduction.ControlDomain] ||
			publicKeys[reproduction.PublicKey] ||
			reports[reproduction.CompilerReportFileSHA256] {
			return errors.New("genesis reproductions must have independent identities, control domains, keys, and reports")
		}
		statement, err := genesisReproductionStatement(release, reproduction)
		if err != nil {
			return err
		}
		if reproduction.StatementSHA256 != statement {
			return errors.New("genesis reproduction statement digest mismatch")
		}
		publicKey, _ := hex.DecodeString(reproduction.PublicKey)
		signature, _ := hex.DecodeString(reproduction.Signature)
		statementBytes, _ := hex.DecodeString(statement)
		if !ed25519.Verify(
			ed25519.PublicKey(publicKey),
			statementBytes,
			signature,
		) {
			return errors.New("genesis reproduction signature verification failed")
		}
		identities[reproduction.Identity] = true
		domains[reproduction.ControlDomain] = true
		publicKeys[reproduction.PublicKey] = true
		reports[reproduction.CompilerReportFileSHA256] = true
	}
	return nil
}

func validateForkValidators(
	validators []ValidatorIdentity,
	assessment CustodyAssessment,
	policy ForkPolicy,
) error {
	if validators == nil || len(validators) == 0 ||
		len(validators) > maxCollectionEntries {
		return errors.New("fork release new validators must be a bounded non-empty array")
	}
	if !sort.SliceIsSorted(validators, func(i, j int) bool {
		if validators[i].SDKOperatorAddress != validators[j].SDKOperatorAddress {
			return validators[i].SDKOperatorAddress < validators[j].SDKOperatorAddress
		}
		return validators[i].ConsensusPublicKey < validators[j].ConsensusPublicKey
	}) {
		return errors.New("fork validators must be sorted by operator address then consensus public key")
	}
	operators := make(map[string]bool)
	publicKeys := make(map[string]bool)
	consensusAddresses := make(map[string]bool)
	nodeIDs := make(map[string]bool)
	keyDigests := make(map[string]bool)
	for _, validator := range validators {
		if err := validateValidatorIdentity("fork validator", validator); err != nil {
			return err
		}
		if err := validateFreshIdentity(
			validator,
			assessment,
			policy.ProhibitedConsensusPublicKeys,
			policy.ProhibitedPrivilegedIdentities,
		); err != nil {
			return err
		}
		if operators[validator.SDKOperatorAddress] ||
			publicKeys[validator.ConsensusPublicKey] ||
			publicKeys[validator.NodePublicKey] ||
			consensusAddresses[validator.ConsensusAddress] ||
			nodeIDs[validator.NodeID] {
			return errors.New("fork validator public identities must be unique")
		}
		for _, digest := range []string{
			validator.ValidatorKeySHA256,
			validator.NodeKeySHA256,
			validator.SigningStateSHA256,
		} {
			if keyDigests[digest] {
				return errors.New("fork validator key-file digests must be unique")
			}
			keyDigests[digest] = true
		}
		operators[validator.SDKOperatorAddress] = true
		publicKeys[validator.ConsensusPublicKey] = true
		publicKeys[validator.NodePublicKey] = true
		consensusAddresses[validator.ConsensusAddress] = true
		nodeIDs[validator.NodeID] = true
	}
	return nil
}

func validateFreshIdentity(
	identity ValidatorIdentity,
	assessment CustodyAssessment,
	prohibitedPublicKeys []string,
	prohibitedPrivilegedIdentities []string,
) error {
	if containsString(prohibitedPublicKeys, identity.ConsensusPublicKey) ||
		containsString(prohibitedPrivilegedIdentities, identity.SDKOperatorAddress) {
		return errors.New("new identity reuses a prohibited old public identity")
	}
	old := assessment.OldValidator
	if identity.SDKOperatorAddress == old.SDKOperatorAddress {
		sdkOperator, found := sdkOperatorAssessment(assessment)
		if !found ||
			sdkOperator.Result != custodyResultPass ||
			sdkOperator.Disposition != privilegedDispositionRetain {
			return errors.New("retaining the old SDK operator requires an independently approved PASS/RETAIN assessment")
		}
	}
	for _, publicKey := range []string{
		identity.ConsensusPublicKey,
		identity.NodePublicKey,
	} {
		if publicKey == old.ConsensusPublicKey ||
			publicKey == old.NodePublicKey ||
			containsString(prohibitedPublicKeys, publicKey) {
			return errors.New("new identity reuses an old or prohibited public key across roles")
		}
	}
	oldFileDigests := map[string]bool{
		old.ValidatorKeySHA256: true,
		old.NodeKeySHA256:      true,
		old.SigningStateSHA256: true,
	}
	for _, digest := range []string{
		identity.ValidatorKeySHA256,
		identity.NodeKeySHA256,
		identity.SigningStateSHA256,
	} {
		if oldFileDigests[digest] {
			return errors.New("new identity reuses an old key-file digest across roles")
		}
	}
	if identity.ConsensusPublicKey == old.ConsensusPublicKey ||
		identity.ConsensusAddress == old.ConsensusAddress ||
		identity.NodeID == old.NodeID ||
		identity.ValidatorKeySHA256 == old.ValidatorKeySHA256 ||
		identity.NodeKeySHA256 == old.NodeKeySHA256 ||
		identity.SigningStateSHA256 == old.SigningStateSHA256 {
		return errors.New("new identity is not fresh relative to the old validator")
	}
	return nil
}

func sdkOperatorAssessment(
	assessment CustodyAssessment,
) (PrivilegedIdentityAssessment, bool) {
	for _, entry := range assessment.PrivilegedIdentityAssessments {
		if entry.Kind == privilegedKindSDKOperator &&
			entry.Identity == assessment.OldValidator.SDKOperatorAddress {
			return entry, true
		}
	}
	return PrivilegedIdentityAssessment{}, false
}

func validateForkChoice(
	choice ForkChoice,
	policy ForkPolicy,
	policyFileSHA256 string,
	release ForkRelease,
	releaseFileSHA256 string,
	assessment CustodyAssessment,
	assessmentFileSHA256 string,
) error {
	if choice.Schema != forkChoiceSchema {
		return fmt.Errorf("fork choice schema must be %q", forkChoiceSchema)
	}
	if err := validateLabel("fork choice ID", choice.ChoiceID, 256); err != nil {
		return err
	}
	if choice.AssessmentSHA256 != assessmentFileSHA256 ||
		choice.ForkPolicySHA256 != policyFileSHA256 ||
		choice.ForkReleaseSHA256 != releaseFileSHA256 ||
		choice.IncidentID != assessment.IncidentID ||
		choice.OldChainID != assessment.ChainID ||
		choice.NewChainID != release.NewChainID {
		return errors.New("fork choice binding does not match its pinned inputs")
	}
	if choice.NewChainID == choice.OldChainID {
		return errors.New("fork choice requires a unique new chain ID")
	}
	if choice.ReasonCode != requiredForkReason {
		return errors.New("fork choice reason must be SUSPECT_SIGNER_CUSTODY")
	}
	if err := validateApprovalShape(choice.Approvals); err != nil {
		return err
	}
	if err := verifyPolicyApprovals(
		choice.Approvals,
		policy,
		func(approval Approval) (string, error) {
			return forkChoiceApprovalStatement(choice, approval)
		},
	); err != nil {
		return fmt.Errorf("fork choice approvals: %w", err)
	}
	return verifyForkChoiceSeal(choice)
}

func trustedSignerKey(
	role,
	identity,
	controlDomain,
	publicKey string,
) string {
	return role + "\x00" + identity + "\x00" + controlDomain + "\x00" + publicKey
}

func verifySignerPolicyApprovals(
	approvals []Approval,
	policy SignerPolicy,
	prohibitedPublicKeys []string,
	prohibitedIdentities []string,
	statement statementBuilder,
) error {
	if uint64(len(approvals)) < policy.MinimumApprovals {
		return errors.New("signer policy minimum approval threshold is not met")
	}
	trusted := make(map[string]bool, len(policy.Signers))
	for _, signer := range policy.Signers {
		trusted[trustedSignerKey(
			signer.Role,
			signer.Identity,
			signer.ControlDomain,
			signer.PublicKey,
		)] = true
	}
	identities := make(map[string]bool)
	controlDomains := make(map[string]bool)
	publicKeys := make(map[string]bool)
	roles := make(map[string]bool)
	for _, approval := range approvals {
		if !trusted[trustedSignerKey(
			approval.Role,
			approval.Identity,
			approval.ControlDomain,
			approval.PublicKey,
		)] {
			return errors.New("approval signer is absent from the pinned signer policy")
		}
		if containsString(prohibitedPublicKeys, approval.PublicKey) {
			return errors.New("approval uses a prohibited old consensus public key")
		}
		if containsString(prohibitedIdentities, approval.Identity) {
			return errors.New("approval uses a prohibited privileged identity")
		}
		if identities[approval.Identity] ||
			controlDomains[approval.ControlDomain] ||
			publicKeys[approval.PublicKey] {
			return errors.New("approvals must use independent identities, control domains, and keys")
		}
		identities[approval.Identity] = true
		controlDomains[approval.ControlDomain] = true
		publicKeys[approval.PublicKey] = true
		roles[approval.Role] = true
		expected, err := statement(approval)
		if err != nil {
			return err
		}
		if err := verifyApprovalSignature(approval, expected); err != nil {
			return err
		}
	}
	if uint64(len(identities)) < policy.MinimumDistinctIdentities ||
		uint64(len(controlDomains)) < policy.MinimumDistinctControlDomains {
		return errors.New("signer policy approval independence threshold is not met")
	}
	for _, required := range policy.RequiredRoles {
		if !roles[required] {
			return errors.New("signer policy mandatory approval role is missing")
		}
	}
	return nil
}

func verifyPolicyApprovals(
	approvals []Approval,
	policy ForkPolicy,
	statement statementBuilder,
) error {
	if uint64(len(approvals)) < policy.MinimumApprovals {
		return errors.New("fork policy minimum approval threshold is not met")
	}
	trusted := make(map[string]bool, len(policy.Signers))
	for _, signer := range policy.Signers {
		key := signer.Role + "\x00" + signer.Identity + "\x00" +
			signer.ControlDomain + "\x00" + signer.PublicKey
		trusted[key] = true
	}
	identities := make(map[string]bool)
	controlDomains := make(map[string]bool)
	publicKeys := make(map[string]bool)
	roles := make(map[string]bool)
	for _, approval := range approvals {
		key := approval.Role + "\x00" + approval.Identity + "\x00" +
			approval.ControlDomain + "\x00" + approval.PublicKey
		if !trusted[key] {
			return errors.New("fork approval signer is absent from the pinned policy")
		}
		if containsString(policy.ProhibitedConsensusPublicKeys, approval.PublicKey) {
			return errors.New("fork approval uses a prohibited old consensus public key")
		}
		if identities[approval.Identity] ||
			controlDomains[approval.ControlDomain] ||
			publicKeys[approval.PublicKey] {
			return errors.New("fork approvals must use independent identities, control domains, and keys")
		}
		identities[approval.Identity] = true
		controlDomains[approval.ControlDomain] = true
		publicKeys[approval.PublicKey] = true
		roles[approval.Role] = true
		expected, err := statement(approval)
		if err != nil {
			return err
		}
		if err := verifyApprovalSignature(approval, expected); err != nil {
			return err
		}
	}
	if uint64(len(identities)) < policy.MinimumDistinctIdentities ||
		uint64(len(controlDomains)) < policy.MinimumDistinctControlDomains {
		return errors.New("fork approval independence threshold is not met")
	}
	for _, role := range policy.RequiredRoles {
		if !roles[role] {
			return errors.New("fork approval is missing a policy-required role")
		}
	}
	return nil
}

func sortedUnique(values []string) []string {
	sort.Strings(values)
	if len(values) < 2 {
		return values
	}
	result := values[:1]
	for _, value := range values[1:] {
		if value != result[len(result)-1] {
			result = append(result, value)
		}
	}
	return result
}
