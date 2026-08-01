package main

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"os"
	"sort"
	"time"
)

const maxTrustPolicyBytes = 1 << 20

func loadPinnedTrustPolicy(path, expectedSHA256 string) (TrustPolicy, string, error) {
	if path == "" {
		return TrustPolicy{}, "", errors.New("--trust-policy is required")
	}
	if err := validateSHA256("--trust-policy-sha256", expectedSHA256, false); err != nil {
		return TrustPolicy{}, "", fmt.Errorf("an externally obtained --trust-policy-sha256 is required: %w", err)
	}
	file, err := os.Open(path)
	if err != nil {
		return TrustPolicy{}, "", fmt.Errorf("open trust policy %s: %w", path, err)
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return TrustPolicy{}, "", fmt.Errorf("stat opened trust policy %s: %w", path, err)
	}
	if info.IsDir() {
		return TrustPolicy{}, "", fmt.Errorf("trust policy %s is a directory", path)
	}
	reader := io.LimitReader(file, maxTrustPolicyBytes+1)
	document, err := io.ReadAll(reader)
	if err != nil {
		return TrustPolicy{}, "", fmt.Errorf("read trust policy %s: %w", path, err)
	}
	if len(document) > maxTrustPolicyBytes {
		return TrustPolicy{}, "", fmt.Errorf("trust policy %s exceeds %d-byte limit", path, maxTrustPolicyBytes)
	}
	policy, actualSHA256, err := decodeTrustPolicy(document, true)
	if err != nil {
		return TrustPolicy{}, "", err
	}
	if actualSHA256 != expectedSHA256 {
		return TrustPolicy{}, "", fmt.Errorf(
			"trust policy SHA-256 mismatch: file has %s, externally pinned value is %s",
			actualSHA256,
			expectedSHA256,
		)
	}
	return policy, actualSHA256, nil
}

// decodeTrustPolicy returns the policy and the SHA-256 of its exact canonical
// bytes. A policy is a local trust root, so ambiguity is rejected just as it is
// for transition documents.
func decodeTrustPolicy(data []byte, requireCanonical bool) (TrustPolicy, string, error) {
	var policy TrustPolicy
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&policy); err != nil {
		return TrustPolicy{}, "", fmt.Errorf("decode trust policy JSON: %w", err)
	}
	var extra json.RawMessage
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return TrustPolicy{}, "", errors.New("decode trust policy JSON: multiple values are not allowed")
		}
		return TrustPolicy{}, "", fmt.Errorf("decode trailing trust policy JSON: %w", err)
	}
	canonical, err := json.Marshal(policy)
	if err != nil {
		return TrustPolicy{}, "", fmt.Errorf("canonicalize trust policy: %w", err)
	}
	if requireCanonical && !bytes.Equal(data, canonical) {
		return TrustPolicy{}, "", errors.New("trust policy JSON is not canonical: require exact compact field order with no whitespace or trailing newline")
	}
	if err := validateTrustPolicy(policy); err != nil {
		return TrustPolicy{}, "", err
	}
	digest := sha256.Sum256(canonical)
	return policy, hex.EncodeToString(digest[:]), nil
}

func trustPolicySHA256(policy TrustPolicy) (string, error) {
	canonical, err := json.Marshal(policy)
	if err != nil {
		return "", fmt.Errorf("canonicalize trust policy: %w", err)
	}
	digest := sha256.Sum256(canonical)
	return hex.EncodeToString(digest[:]), nil
}

func validateTrustPolicy(policy TrustPolicy) error {
	if policy.Schema != trustPolicySchema {
		return fmt.Errorf("trust policy schema must be %q, got %q", trustPolicySchema, policy.Schema)
	}
	if err := validateLabel("trust policy policy_id", policy.PolicyID, 128); err != nil {
		return err
	}
	if err := validateLabel("trust policy chain_id", policy.ChainID, 128); err != nil {
		return err
	}
	if policy.IncidentID == "" && policy.ReleaseID == "" {
		return errors.New("trust policy requires incident_id, release_id, or both")
	}
	if policy.IncidentID != "" {
		if err := validateLabel("trust policy incident_id", policy.IncidentID, 128); err != nil {
			return err
		}
	}
	if policy.ReleaseID != "" {
		if err := validateLabel("trust policy release_id", policy.ReleaseID, 128); err != nil {
			return err
		}
	}
	if policy.ApprovalPolicies == nil {
		return errors.New("trust policy approval_policies must be [] rather than null")
	}
	if !sort.SliceIsSorted(policy.ApprovalPolicies, func(i, j int) bool {
		return edgeApprovalPolicyLess(policy.ApprovalPolicies[i], policy.ApprovalPolicies[j])
	}) {
		return errors.New("trust policy approval_policies must be sorted by lane, from, then to")
	}

	expectedEdges := make(map[string]bool)
	if policy.IncidentID != "" {
		addExpectedLaneEdges(expectedEdges, laneIncident, incidentTransitions)
	}
	if policy.ReleaseID != "" {
		addExpectedLaneEdges(expectedEdges, laneRelease, releaseTransitions)
	}
	seenEdges := make(map[string]bool, len(policy.ApprovalPolicies))
	allSeparatedPairs := make(map[string]RolePair)
	for i, edgePolicy := range policy.ApprovalPolicies {
		prefix := fmt.Sprintf("trust policy approval_policies[%d]", i)
		key := edgeApprovalPolicyKey(edgePolicy.Lane, edgePolicy.From, edgePolicy.To)
		if seenEdges[key] {
			return fmt.Errorf("%s duplicates edge %s/%s->%s", prefix, edgePolicy.Lane, edgePolicy.From, edgePolicy.To)
		}
		seenEdges[key] = true
		if !expectedEdges[key] {
			return fmt.Errorf(
				"%s edge %s/%s->%s is not an allowed edge in an authorized lane",
				prefix,
				edgePolicy.Lane,
				edgePolicy.From,
				edgePolicy.To,
			)
		}
		if err := validateApprovalPolicyShape(edgePolicy.ApprovalPolicy); err != nil {
			return fmt.Errorf("%s: %w", prefix, err)
		}
		if edgePolicy.ApprovalPolicy.MinimumApprovals == 0 {
			return fmt.Errorf("%s approval_policy.minimum_approvals must be greater than zero", prefix)
		}
		if edgePolicy.ApprovalPolicy.MinimumDistinctIdentities == 0 {
			return fmt.Errorf("%s approval_policy.minimum_distinct_identities must be greater than zero", prefix)
		}
		if len(edgePolicy.ApprovalPolicy.RequiredRoles) == 0 {
			return fmt.Errorf("%s approval_policy.required_roles must contain at least one role", prefix)
		}
		mandatoryRoles := mandatoryRolesForEdge(edgePolicy)
		for _, required := range mandatoryRoles {
			position := sort.SearchStrings(
				edgePolicy.ApprovalPolicy.RequiredRoles,
				required,
			)
			if position == len(edgePolicy.ApprovalPolicy.RequiredRoles) ||
				edgePolicy.ApprovalPolicy.RequiredRoles[position] != required {
				return fmt.Errorf(
					"%s must require operational role %q",
					prefix,
					required,
				)
			}
		}
		if edgePolicy.ApprovalPolicy.MinimumApprovals <
			uint64(len(mandatoryRoles)) {
			return fmt.Errorf(
				"%s approval_policy.minimum_approvals must be at least %d for its mandatory operational roles",
				prefix,
				len(mandatoryRoles),
			)
		}
		if edgePolicy.ApprovalPolicy.MinimumDistinctIdentities <
			uint64(len(mandatoryRoles)) {
			return fmt.Errorf(
				"%s approval_policy.minimum_distinct_identities must be at least %d for its mandatory operational roles",
				prefix,
				len(mandatoryRoles),
			)
		}
		for _, requiredPair := range mandatorySeparatedRolePairsForEdge(edgePolicy) {
			found := false
			for _, actualPair := range edgePolicy.ApprovalPolicy.SeparatedRolePairs {
				if actualPair == requiredPair {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf(
					"%s must separate operational roles %q and %q",
					prefix,
					requiredPair.RoleA,
					requiredPair.RoleB,
				)
			}
		}
		if edgeRequiresStrictOperatorSupermajority(edgePolicy) {
			quorum := edgePolicy.ApprovalPolicy.PowerQuorum
			if quorum.Role != validatorOperatorRole ||
				!quorum.Strict ||
				!powerThresholdAtLeastTwoThirds(quorum) {
				return fmt.Errorf(
					"%s requires role %q with a strict power quorum greater than or equal to 2/3 for activation readiness",
					prefix,
					validatorOperatorRole,
				)
			}
		}
		for _, pair := range edgePolicy.ApprovalPolicy.SeparatedRolePairs {
			allSeparatedPairs[pair.RoleA+"\x00"+pair.RoleB] = pair
		}
	}
	for expected := range expectedEdges {
		if !seenEdges[expected] {
			return fmt.Errorf("trust policy approval_policies is missing required edge %s", expected)
		}
	}
	if policy.Signers == nil {
		return errors.New("trust policy signers must be [] rather than null")
	}
	if len(policy.Signers) == 0 {
		return errors.New("trust policy requires at least one signer")
	}
	if !sort.SliceIsSorted(policy.Signers, func(i, j int) bool {
		return trustedSignerLess(policy.Signers[i], policy.Signers[j])
	}) {
		return errors.New("trust policy signers must be sorted by role, identity, then public_key")
	}

	identityToKey := make(map[string]string)
	keyToIdentity := make(map[string]string)
	identityRoles := make(map[string]map[string]bool)
	keyRoles := make(map[string]map[string]bool)
	seenTuple := make(map[string]bool)
	roleCounts := make(map[string]int)

	for i, signer := range policy.Signers {
		prefix := fmt.Sprintf("trust policy signers[%d]", i)
		if err := validateLabel(prefix+".role", signer.Role, 128); err != nil {
			return err
		}
		if err := validateLabel(prefix+".identity", signer.Identity, 256); err != nil {
			return err
		}
		if _, err := decodeExactHex(prefix+".public_key", signer.PublicKey, ed25519.PublicKeySize); err != nil {
			return err
		}

		if existing, ok := identityToKey[signer.Identity]; ok && existing != signer.PublicKey {
			return fmt.Errorf("trust policy identity %q uses multiple public keys", signer.Identity)
		}
		if existing, ok := keyToIdentity[signer.PublicKey]; ok && existing != signer.Identity {
			return fmt.Errorf("trust policy public key %q uses multiple identities", signer.PublicKey)
		}
		identityToKey[signer.Identity] = signer.PublicKey
		keyToIdentity[signer.PublicKey] = signer.Identity

		tuple := signer.Role + "\x00" + signer.Identity + "\x00" + signer.PublicKey
		if seenTuple[tuple] {
			return fmt.Errorf("trust policy signer tuple %q/%q is duplicated", signer.Role, signer.Identity)
		}
		seenTuple[tuple] = true
		roleCounts[signer.Role]++
		if identityRoles[signer.Identity] == nil {
			identityRoles[signer.Identity] = make(map[string]bool)
		}
		identityRoles[signer.Identity][signer.Role] = true
		if keyRoles[signer.PublicKey] == nil {
			keyRoles[signer.PublicKey] = make(map[string]bool)
		}
		keyRoles[signer.PublicKey][signer.Role] = true
	}

	for _, required := range []string{
		evidenceCustodianRole,
		policyRotationAuthorityRole,
	} {
		if roleCounts[required] == 0 {
			return fmt.Errorf(
				"trust policy requires preprovisioned offline role %q for signed policy supersession",
				required,
			)
		}
	}
	for identity, roles := range identityRoles {
		if (roles[evidenceCustodianRole] ||
			roles[policyRotationAuthorityRole]) &&
			len(roles) != 1 {
			return fmt.Errorf(
				"trust policy offline supersession identity %q must not fill routine operational roles",
				identity,
			)
		}
	}
	for publicKey, roles := range keyRoles {
		if (roles[evidenceCustodianRole] ||
			roles[policyRotationAuthorityRole]) &&
			len(roles) != 1 {
			return fmt.Errorf(
				"trust policy offline supersession public key %q must not fill routine operational roles",
				publicKey,
			)
		}
	}

	for i, edgePolicy := range policy.ApprovalPolicies {
		approvalPolicy := edgePolicy.ApprovalPolicy
		prefix := fmt.Sprintf("trust policy approval_policies[%d]", i)
		if approvalPolicy.MinimumApprovals > uint64(len(policy.Signers)) {
			return fmt.Errorf(
				"%s requires %d approvals but declares only %d signer tuples",
				prefix,
				approvalPolicy.MinimumApprovals,
				len(policy.Signers),
			)
		}
		if approvalPolicy.MinimumDistinctIdentities > uint64(len(identityToKey)) {
			return fmt.Errorf(
				"%s requires %d distinct identities but declares only %d",
				prefix,
				approvalPolicy.MinimumDistinctIdentities,
				len(identityToKey),
			)
		}
		for _, role := range approvalPolicy.RequiredRoles {
			if roleCounts[role] == 0 {
				return fmt.Errorf("%s required role %q has no signer", prefix, role)
			}
		}
		if quorumRole := approvalPolicy.PowerQuorum.Role; quorumRole != "" &&
			roleCounts[quorumRole] == 0 {
			return fmt.Errorf("%s power quorum role %q has no signer", prefix, quorumRole)
		}
	}
	for _, pair := range allSeparatedPairs {
		for identity, roles := range identityRoles {
			if roles[pair.RoleA] && roles[pair.RoleB] {
				return fmt.Errorf("trust policy identity %q fills separated roles %q and %q", identity, pair.RoleA, pair.RoleB)
			}
		}
		for publicKey, roles := range keyRoles {
			if roles[pair.RoleA] && roles[pair.RoleB] {
				return fmt.Errorf("trust policy public key %q fills separated roles %q and %q", publicKey, pair.RoleA, pair.RoleB)
			}
		}
	}
	return nil
}

func edgeApprovalPolicyLess(a, b EdgeApprovalPolicy) bool {
	if a.Lane != b.Lane {
		return a.Lane < b.Lane
	}
	if a.From != b.From {
		return a.From < b.From
	}
	return a.To < b.To
}

func edgeApprovalPolicyKey(journalLane lane, from, to State) string {
	return string(journalLane) + "/" + string(from) + "->" + string(to)
}

func addExpectedLaneEdges(
	expected map[string]bool,
	journalLane lane,
	transitions map[State]map[State]bool,
) {
	for from, destinations := range transitions {
		for to, allowed := range destinations {
			if allowed {
				expected[edgeApprovalPolicyKey(journalLane, from, to)] = true
			}
		}
	}
}

func edgeRequiresStrictOperatorSupermajority(policy EdgeApprovalPolicy) bool {
	return (policy.Lane == laneRelease &&
		policy.From == StateScheduled &&
		policy.To == StateStaged) ||
		(policy.Lane == laneIncident &&
			policy.From == StateRecoveryReady &&
			policy.To == StateActivating) ||
		(policy.Lane == laneIncident && policy.To == StateForkChoice)
}

func mandatoryRolesForEdge(policy EdgeApprovalPolicy) []string {
	roles := []string{}
	switch {
	case policy.Lane == laneRelease &&
		policy.From == StatePreparing &&
		policy.To == StateReleaseFrozen:
		roles = []string{
			releaseAuthorRole,
			releaseVerifierRole,
		}
	case policy.Lane == laneRelease &&
		policy.From == StateReleaseFrozen &&
		policy.To == StateScheduled:
		roles = []string{
			governanceCoordinatorRole,
			releaseVerifierRole,
		}
	case policy.Lane == laneRelease &&
		policy.From == StateScheduled &&
		policy.To == StateStaged:
		roles = []string{releaseVerifierRole, validatorOperatorRole}
	case policy.Lane == laneRelease &&
		(policy.From == StateScheduled || policy.From == StateStaged) &&
		policy.To == StateCancelled:
		roles = []string{
			governanceCoordinatorRole,
			releaseVerifierRole,
		}
	case policy.Lane == laneIncident &&
		policy.From == StateRecoveryDesign &&
		policy.To == StateRecoveryReady:
		roles = []string{
			ibcLeadRole,
			incidentCommanderRole,
			releaseVerifierRole,
			supplyVerifierRole,
		}
	case policy.Lane == laneIncident &&
		policy.From == StateRecoveryReady &&
		policy.To == StateActivating:
		roles = []string{
			ibcLeadRole,
			incidentCommanderRole,
			releaseVerifierRole,
			supplyVerifierRole,
			validatorOperatorRole,
		}
	case policy.Lane == laneIncident && policy.To == StateForkChoice:
		roles = []string{
			governanceCoordinatorRole,
			ibcLeadRole,
			incidentCommanderRole,
			releaseVerifierRole,
			supplyVerifierRole,
			validatorOperatorRole,
		}
	}
	sort.Strings(roles)
	return roles
}

func mandatorySeparatedRolePairsForEdge(policy EdgeApprovalPolicy) []RolePair {
	roles := mandatoryRolesForEdge(policy)
	pairs := make([]RolePair, 0, len(roles)*(len(roles)-1)/2)
	for i := 0; i < len(roles); i++ {
		for j := i + 1; j < len(roles); j++ {
			pairs = append(pairs, RolePair{
				RoleA: roles[i],
				RoleB: roles[j],
			})
		}
	}
	return pairs
}

func powerThresholdAtLeastTwoThirds(quorum PowerQuorum) bool {
	left := new(big.Int).Mul(
		new(big.Int).SetUint64(quorum.Numerator),
		big.NewInt(3),
	)
	right := new(big.Int).Mul(
		new(big.Int).SetUint64(quorum.Denominator),
		big.NewInt(2),
	)
	return left.Cmp(right) >= 0
}

func trustedSignerLess(a, b TrustedSigner) bool {
	if a.Role != b.Role {
		return a.Role < b.Role
	}
	if a.Identity != b.Identity {
		return a.Identity < b.Identity
	}
	return a.PublicKey < b.PublicKey
}

func validateTransitionAgainstTrustPolicy(
	transition Transition,
	policy TrustPolicy,
	policySHA256 string,
	expectedPowerSnapshotSHA256 string,
) error {
	if err := validateTransitionTrustBinding(transition, policy, policySHA256); err != nil {
		return err
	}
	for i, approval := range transition.Approvals {
		if err := validateApprovalAuthorized(approval, policy); err != nil {
			return fmt.Errorf("approvals[%d]: %w", i, err)
		}
	}
	approvalPolicy, err := validateTransitionPowerSnapshotAgainstPolicy(
		transition,
		policy,
		expectedPowerSnapshotSHA256,
	)
	if err != nil {
		return err
	}
	return validateApprovalQuorum(
		transition.Approvals,
		approvalPolicy,
		transition.PowerSnapshot,
	)
}

func validateTransitionPowerSnapshotAgainstPolicy(
	transition Transition,
	policy TrustPolicy,
	expectedPowerSnapshotSHA256 string,
) (ApprovalPolicy, error) {
	approvalPolicy, found := approvalPolicyForEdge(
		policy,
		transition.Lane,
		transition.From,
		transition.To,
	)
	if !found {
		return ApprovalPolicy{}, fmt.Errorf(
			"externally pinned trust policy has no approval policy for edge %s",
			edgeApprovalPolicyKey(transition.Lane, transition.From, transition.To),
		)
	}
	if err := validatePowerSnapshotForTransition(
		transition,
		policy,
		approvalPolicy.PowerQuorum,
		expectedPowerSnapshotSHA256,
	); err != nil {
		return ApprovalPolicy{}, err
	}
	return approvalPolicy, nil
}

func validatePowerSnapshotForTransition(
	transition Transition,
	policy TrustPolicy,
	quorum PowerQuorum,
	expectedPowerSnapshotSHA256 string,
) error {
	if quorum.Role == "" {
		if expectedPowerSnapshotSHA256 != "" {
			return errors.New("externally pinned power snapshot was supplied for an edge with no power quorum")
		}
		if !isEmptyPowerSnapshot(transition.PowerSnapshot) {
			return errors.New("transition power_snapshot must be empty when its edge has no power quorum")
		}
		return nil
	}
	snapshot := transition.PowerSnapshot
	if snapshot.Schema == "" {
		return errors.New("power-gated transition requires power_snapshot")
	}
	if err := validateSHA256(
		"externally pinned power snapshot SHA-256",
		expectedPowerSnapshotSHA256,
		false,
	); err != nil {
		return fmt.Errorf(
			"power-gated transition requires an externally obtained snapshot digest: %w",
			err,
		)
	}
	if snapshot.SnapshotSHA256 != expectedPowerSnapshotSHA256 {
		return fmt.Errorf(
			"power snapshot SHA-256 %s does not match externally pinned digest %s",
			snapshot.SnapshotSHA256,
			expectedPowerSnapshotSHA256,
		)
	}
	if snapshot.Role != quorum.Role {
		return fmt.Errorf(
			"power snapshot role %q does not match edge quorum role %q",
			snapshot.Role,
			quorum.Role,
		)
	}
	if snapshot.ChainID != transition.ChainID {
		return fmt.Errorf(
			"power snapshot chain_id %q does not match transition chain_id %q",
			snapshot.ChainID,
			transition.ChainID,
		)
	}
	if snapshot.Height != transition.Checkpoint.Height ||
		snapshot.BlockIDSHA256 != transition.Checkpoint.BlockIDSHA256 ||
		snapshot.AppHashSHA256 != transition.Checkpoint.AppHashSHA256 {
		return fmt.Errorf(
			"power snapshot must bind the exact transition checkpoint: snapshot=%d/%s/%s transition=%d/%s/%s",
			snapshot.Height,
			snapshot.BlockIDSHA256,
			snapshot.AppHashSHA256,
			transition.Checkpoint.Height,
			transition.Checkpoint.BlockIDSHA256,
			transition.Checkpoint.AppHashSHA256,
		)
	}
	occurredAt, err := time.Parse(time.RFC3339Nano, transition.OccurredAt)
	if err != nil {
		return fmt.Errorf("parse transition occurred_at for power snapshot: %w", err)
	}
	capturedAt, _ := time.Parse(time.RFC3339Nano, snapshot.CapturedAt)
	validUntil, _ := time.Parse(time.RFC3339Nano, snapshot.ValidUntil)
	if occurredAt.Before(capturedAt) {
		return fmt.Errorf(
			"transition occurred_at %s predates power snapshot capture %s",
			transition.OccurredAt,
			snapshot.CapturedAt,
		)
	}
	if occurredAt.After(validUntil) {
		return fmt.Errorf(
			"power snapshot expired at %s before transition occurred_at %s",
			snapshot.ValidUntil,
			transition.OccurredAt,
		)
	}
	for i, member := range snapshot.Members {
		if !trustedSignerAuthorized(
			policy,
			snapshot.Role,
			member.Identity,
			member.PublicKey,
		) {
			return fmt.Errorf(
				"power_snapshot.members[%d] role/identity/public_key is not authorized by the externally pinned trust policy",
				i,
			)
		}
	}
	foundSnapshotEvidence := false
	for _, item := range transition.Evidence {
		if item.Type == "validator-power-snapshot" &&
			item.SHA256 == snapshot.SnapshotSHA256 {
			foundSnapshotEvidence = true
			break
		}
	}
	if !foundSnapshotEvidence {
		return fmt.Errorf(
			"transition evidence does not bind validator-power snapshot %s",
			snapshot.SnapshotSHA256,
		)
	}
	return nil
}

func approvalPolicyForEdge(
	policy TrustPolicy,
	journalLane lane,
	from, to State,
) (ApprovalPolicy, bool) {
	for _, candidate := range policy.ApprovalPolicies {
		if candidate.Lane == journalLane && candidate.From == from && candidate.To == to {
			return candidate.ApprovalPolicy, true
		}
	}
	return ApprovalPolicy{}, false
}

func validateTransitionTrustBinding(transition Transition, policy TrustPolicy, policySHA256 string) error {
	if err := validateSHA256("trusted policy SHA-256", policySHA256, false); err != nil {
		return err
	}
	if transition.TrustPolicySHA256 != policySHA256 {
		return fmt.Errorf(
			"trust_policy_sha256 mismatch: transition has %q, externally pinned policy has %q",
			transition.TrustPolicySHA256,
			policySHA256,
		)
	}
	if transition.ChainID != policy.ChainID {
		return fmt.Errorf("trust policy chain_id %q does not authorize %q", policy.ChainID, transition.ChainID)
	}
	switch transition.Lane {
	case laneIncident:
		if transition.IncidentID != policy.IncidentID {
			return fmt.Errorf("trust policy incident_id %q does not authorize %q", policy.IncidentID, transition.IncidentID)
		}
	case laneRelease:
		if transition.ReleaseID != policy.ReleaseID {
			return fmt.Errorf("trust policy release_id %q does not authorize %q", policy.ReleaseID, transition.ReleaseID)
		}
	default:
		return fmt.Errorf("trust policy cannot authorize unknown lane %q", transition.Lane)
	}
	return nil
}

func validateApprovalAuthorized(approval Approval, policy TrustPolicy) error {
	if trustedSignerAuthorized(
		policy,
		approval.Role,
		approval.Identity,
		approval.PublicKey,
	) {
		return nil
	}
	return errors.New("role/identity/public_key tuple is not authorized by the externally pinned trust policy")
}

func trustedSignerAuthorized(
	policy TrustPolicy,
	role, identity, publicKey string,
) bool {
	for _, signer := range policy.Signers {
		if trustedSignerKey(signer.Role, signer.Identity, signer.PublicKey) ==
			trustedSignerKey(role, identity, publicKey) {
			return true
		}
	}
	return false
}

func trustedSignerKey(role, identity, publicKey string) string {
	return role + "\x00" + identity + "\x00" + publicKey
}
