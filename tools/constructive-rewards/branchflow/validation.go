package branchflow

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math/big"
	"regexp"
	"sort"
)

var (
	idPattern     = regexp.MustCompile(`^[a-z0-9][a-z0-9._:@/-]{0,127}$`)
	domainPattern = regexp.MustCompile(`^[a-z][a-z0-9_]{0,63}$`)
	digestPattern = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)
	amountPattern = regexp.MustCompile(`^(0|[1-9][0-9]*)$`)
)

type validatedRequest struct {
	request       Request
	envelope      *big.Int
	minimumPayout *big.Int
	programCap    *big.Int
	hasProgramCap bool
	priorPaid     map[string]*big.Int
	nodes         map[string]Node
}

func validateRequest(input Request) (*validatedRequest, error) {
	if input.Schema != Schema {
		return nil, inputError(CodeUnsupportedSchema, "schema", fmt.Sprintf("must equal %q", Schema))
	}
	if err := validateID(input.EnvelopeID, "envelope_id"); err != nil {
		return nil, err
	}
	if err := validateID(input.FundedClusterID, "funded_cluster_id"); err != nil {
		return nil, err
	}
	if !validFundedMilestone(input.FundedMilestone) {
		return nil, inputError(CodeInvalidMilestone, "funded_milestone", "must be E2, E3, E4, E5, or E6")
	}
	if !input.DescendantWindowClosed {
		return nil, inputError(CodeInvalidInput, "descendant_window_closed", "must be true before deterministic evaluation")
	}
	envelope, err := parseAmount(input.EnvelopeUzrn, "envelope_uzrn")
	if err != nil {
		return nil, err
	}
	minimum, programCap, hasProgramCap, err := validatePolicy(input.Policy)
	if err != nil {
		return nil, err
	}
	if input.FundedMilestone != MilestoneE5 && input.FundedMilestone != MilestoneE6 {
		if input.Policy.DownstreamPPM != 0 {
			return nil, inputError(CodeInvalidPolicySum, "policy.downstream_ppm", "must be zero for E2, E3, and E4 envelopes")
		}
		if len(input.DescendantImpacts) != 0 {
			return nil, inputError(CodeInvalidMilestone, "descendant_impacts", "must be empty outside E5 and E6 envelopes")
		}
	}

	if len(input.Nodes) == 0 {
		return nil, inputError(CodeInvalidInput, "nodes", "must contain the funded cluster")
	}
	if len(input.Nodes) > MaxNodes {
		return nil, limitError(CodeLimitExceeded, "nodes", fmt.Sprintf("maximum is %d", MaxNodes))
	}
	if len(input.Edges) > MaxEdges {
		return nil, limitError(CodeLimitExceeded, "edges", fmt.Sprintf("maximum is %d", MaxEdges))
	}
	if len(input.DescendantImpacts) > MaxDescendantImpacts {
		return nil, limitError(CodeLimitExceeded, "descendant_impacts", fmt.Sprintf("maximum is %d", MaxDescendantImpacts))
	}
	if len(input.PriorReceiptUses) > MaxPriorReceiptUses {
		return nil, limitError(CodeLimitExceeded, "prior_receipt_uses", fmt.Sprintf("maximum is %d", MaxPriorReceiptUses))
	}
	if len(input.PriorControllerPaid) > MaxControllers {
		return nil, limitError(CodeLimitExceeded, "prior_controller_paid", fmt.Sprintf("maximum is %d", MaxControllers))
	}

	request := input
	request.Nodes = append([]Node(nil), input.Nodes...)
	request.Edges = append([]Edge(nil), input.Edges...)
	request.DescendantImpacts = append([]Impact(nil), input.DescendantImpacts...)
	request.PriorReceiptUses = append([]ReceiptUse(nil), input.PriorReceiptUses...)
	request.PriorControllerPaid = append([]ControllerAmount(nil), input.PriorControllerPaid...)

	nodes := make(map[string]Node, len(request.Nodes))
	controllers := make(map[string]struct{})
	for i := range request.Nodes {
		node := &request.Nodes[i]
		path := fmt.Sprintf("nodes[%d]", i)
		if err := validateID(node.ClusterID, path+".cluster_id"); err != nil {
			return nil, err
		}
		if _, exists := nodes[node.ClusterID]; exists {
			return nil, inputError(CodeDuplicateID, path+".cluster_id", "cluster ID is duplicated")
		}
		switch node.Mode {
		case NodeModePayAndPropagate, NodeModePassThrough, NodeModeBlocked:
		default:
			return nil, inputError(CodeInvalidNodeMode, path+".mode", "must be PAY_AND_PROPAGATE, PASS_THROUGH, or BLOCKED")
		}
		if node.Mode != NodeModePayAndPropagate && len(node.Credits) != 0 {
			return nil, inputError(CodeInvalidCreditSum, path+".credits", "PASS_THROUGH and BLOCKED nodes must not declare payment credits")
		}
		if len(node.Credits) > MaxCreditsPerNode {
			return nil, limitError(CodeLimitExceeded, path+".credits", fmt.Sprintf("maximum is %d", MaxCreditsPerNode))
		}
		node.Credits = append([]Credit(nil), node.Credits...)
		creditKeys := make(map[string]struct{}, len(node.Credits))
		var creditSum uint64
		for j, credit := range node.Credits {
			creditPath := fmt.Sprintf("%s.credits[%d]", path, j)
			if err := validateID(credit.ControllerID, creditPath+".controller_id"); err != nil {
				return nil, err
			}
			if err := validateID(credit.RoleID, creditPath+".role_id"); err != nil {
				return nil, err
			}
			if credit.WeightPPM == 0 || uint64(credit.WeightPPM) > PPM {
				return nil, inputError(CodeInvalidCreditSum, creditPath+".weight_ppm", fmt.Sprintf("must be in [1,%d]", PPM))
			}
			key := credit.ControllerID + "\x00" + credit.RoleID
			if _, exists := creditKeys[key]; exists {
				return nil, inputError(CodeDuplicateID, creditPath, "controller-role credit is duplicated")
			}
			creditKeys[key] = struct{}{}
			creditSum += uint64(credit.WeightPPM)
			controllers[credit.ControllerID] = struct{}{}
		}
		if creditSum > PPM {
			return nil, inputError(CodeInvalidCreditSum, path+".credits", fmt.Sprintf("weights sum to %d; maximum is %d", creditSum, PPM))
		}
		sort.Slice(node.Credits, func(i, j int) bool {
			if node.Credits[i].ControllerID != node.Credits[j].ControllerID {
				return node.Credits[i].ControllerID < node.Credits[j].ControllerID
			}
			return node.Credits[i].RoleID < node.Credits[j].RoleID
		})
		nodes[node.ClusterID] = *node
	}
	if _, exists := nodes[input.FundedClusterID]; !exists {
		return nil, inputError(CodeUnknownNode, "funded_cluster_id", "does not identify a supplied node")
	}
	if nodes[input.FundedClusterID].Mode != NodeModePayAndPropagate {
		return nil, inputError(CodeInvalidNodeMode, "funded_cluster_id", "funded cluster must be PAY_AND_PROPAGATE")
	}
	sort.Slice(request.Nodes, func(i, j int) bool { return request.Nodes[i].ClusterID < request.Nodes[j].ClusterID })

	edgeKeys := make(map[string]struct{}, len(request.Edges))
	parentCounts := make(map[string]int)
	for i, edge := range request.Edges {
		path := fmt.Sprintf("edges[%d]", i)
		if err := validateID(edge.ChildClusterID, path+".child_cluster_id"); err != nil {
			return nil, err
		}
		if err := validateID(edge.ParentClusterID, path+".parent_cluster_id"); err != nil {
			return nil, err
		}
		if _, exists := nodes[edge.ChildClusterID]; !exists {
			return nil, inputError(CodeUnknownNode, path+".child_cluster_id", "does not identify a supplied node")
		}
		if _, exists := nodes[edge.ParentClusterID]; !exists {
			return nil, inputError(CodeUnknownNode, path+".parent_cluster_id", "does not identify a supplied node")
		}
		if edge.ChildClusterID == edge.ParentClusterID {
			return nil, graphError(CodeSelfEdge, path, "child and parent must differ")
		}
		if edge.RawDependencyPPM == 0 {
			return nil, inputError(CodeZeroEdge, path+".raw_dependency_ppm", "must be positive")
		}
		if uint64(edge.RawDependencyPPM) > PPM {
			return nil, inputError(CodeInvalidInput, path+".raw_dependency_ppm", fmt.Sprintf("must not exceed %d", PPM))
		}
		key := edge.ChildClusterID + "\x00" + edge.ParentClusterID
		if _, exists := edgeKeys[key]; exists {
			return nil, graphError(CodeDuplicateEdge, path, "child-parent edge is duplicated")
		}
		edgeKeys[key] = struct{}{}
		parentCounts[edge.ChildClusterID]++
		if parentCounts[edge.ChildClusterID] > MaxParentsPerNode {
			return nil, limitError(CodeTooManyParents, path+".child_cluster_id", fmt.Sprintf("maximum is %d", MaxParentsPerNode))
		}
	}
	sort.Slice(request.Edges, func(i, j int) bool {
		if request.Edges[i].ChildClusterID != request.Edges[j].ChildClusterID {
			return request.Edges[i].ChildClusterID < request.Edges[j].ChildClusterID
		}
		return request.Edges[i].ParentClusterID < request.Edges[j].ParentClusterID
	})

	priorKeys := make(map[string]struct{}, len(request.PriorReceiptUses))
	priorSlots := make(map[string]struct{}, len(request.PriorReceiptUses))
	for i, use := range request.PriorReceiptUses {
		path := fmt.Sprintf("prior_receipt_uses[%d]", i)
		if err := validateDigest(use.ReceiptKey, path+".receipt_key", CodeInvalidReceiptKey); err != nil {
			return nil, err
		}
		if err := validateID(use.EconomicSlotID, path+".economic_slot_id"); err != nil {
			return nil, err
		}
		if _, exists := priorKeys[use.ReceiptKey]; exists {
			return nil, inputError(CodeDuplicateReceipt, path+".receipt_key", "prior receipt key is duplicated")
		}
		if _, exists := priorSlots[use.EconomicSlotID]; exists {
			return nil, inputError(CodeDuplicateReceipt, path+".economic_slot_id", "prior economic slot is duplicated")
		}
		priorKeys[use.ReceiptKey] = struct{}{}
		priorSlots[use.EconomicSlotID] = struct{}{}
	}
	sort.Slice(request.PriorReceiptUses, func(i, j int) bool {
		if request.PriorReceiptUses[i].ReceiptKey != request.PriorReceiptUses[j].ReceiptKey {
			return request.PriorReceiptUses[i].ReceiptKey < request.PriorReceiptUses[j].ReceiptKey
		}
		return request.PriorReceiptUses[i].EconomicSlotID < request.PriorReceiptUses[j].EconomicSlotID
	})

	currentKeys := make(map[string]struct{}, len(request.DescendantImpacts))
	currentSlots := make(map[string]struct{}, len(request.DescendantImpacts))
	for i, impact := range request.DescendantImpacts {
		path := fmt.Sprintf("descendant_impacts[%d]", i)
		if err := validateID(impact.DescendantClusterID, path+".descendant_cluster_id"); err != nil {
			return nil, err
		}
		if _, exists := nodes[impact.DescendantClusterID]; !exists {
			return nil, inputError(CodeUnknownNode, path+".descendant_cluster_id", "does not identify a supplied node")
		}
		if impact.DescendantClusterID == input.FundedClusterID {
			return nil, inputError(CodeImpactNotLinked, path+".descendant_cluster_id", "funded cluster cannot be its own descendant")
		}
		if nodes[impact.DescendantClusterID].Mode != NodeModePayAndPropagate {
			return nil, inputError(CodeImpactNotLinked, path+".descendant_cluster_id", "descendant must be PAY_AND_PROPAGATE")
		}
		if impact.Milestone != input.FundedMilestone {
			return nil, inputError(CodeInvalidMilestone, path+".milestone", "must equal funded_milestone")
		}
		switch impact.Disposition {
		case ImpactDispositionPayable, ImpactDispositionTerminal:
		default:
			return nil, inputError(
				CodeInvalidImpactDisposition,
				path+".disposition",
				"must be PAYABLE or TERMINAL",
			)
		}
		if err := validateDigest(impact.ReceiptKey, path+".receipt_key", CodeInvalidReceiptKey); err != nil {
			return nil, err
		}
		if err := validateID(impact.EconomicSlotID, path+".economic_slot_id"); err != nil {
			return nil, err
		}
		if impact.ImpactPPM == 0 || uint64(impact.ImpactPPM) > PPM {
			return nil, inputError(CodeInvalidInput, path+".impact_ppm", fmt.Sprintf("must be in [1,%d]", PPM))
		}
		if _, exists := currentKeys[impact.ReceiptKey]; exists {
			return nil, inputError(CodeDuplicateReceipt, path+".receipt_key", "receipt key is duplicated in this envelope")
		}
		if _, exists := currentSlots[impact.EconomicSlotID]; exists {
			return nil, inputError(CodeDuplicateReceipt, path+".economic_slot_id", "economic slot is duplicated in this envelope")
		}
		if _, consumed := priorKeys[impact.ReceiptKey]; consumed {
			return nil, receiptConsumedError(path+".receipt_key", "receipt key already has an economic use")
		}
		if _, consumed := priorSlots[impact.EconomicSlotID]; consumed {
			return nil, receiptConsumedError(path+".economic_slot_id", "economic slot already has a receipt use")
		}
		currentKeys[impact.ReceiptKey] = struct{}{}
		currentSlots[impact.EconomicSlotID] = struct{}{}
	}
	sort.Slice(request.DescendantImpacts, func(i, j int) bool {
		if request.DescendantImpacts[i].DescendantClusterID != request.DescendantImpacts[j].DescendantClusterID {
			return request.DescendantImpacts[i].DescendantClusterID < request.DescendantImpacts[j].DescendantClusterID
		}
		if request.DescendantImpacts[i].Milestone != request.DescendantImpacts[j].Milestone {
			return request.DescendantImpacts[i].Milestone < request.DescendantImpacts[j].Milestone
		}
		return request.DescendantImpacts[i].ReceiptKey < request.DescendantImpacts[j].ReceiptKey
	})

	priorPaid := make(map[string]*big.Int, len(request.PriorControllerPaid))
	for i, paid := range request.PriorControllerPaid {
		path := fmt.Sprintf("prior_controller_paid[%d]", i)
		if err := validateID(paid.ControllerID, path+".controller_id"); err != nil {
			return nil, err
		}
		amount, err := parseAmount(paid.AmountUzrn, path+".amount_uzrn")
		if err != nil {
			return nil, err
		}
		if _, exists := priorPaid[paid.ControllerID]; exists {
			return nil, inputError(CodeDuplicateID, path+".controller_id", "prior controller amount is duplicated")
		}
		priorPaid[paid.ControllerID] = amount
		controllers[paid.ControllerID] = struct{}{}
	}
	if len(controllers) > MaxControllers {
		return nil, limitError(CodeLimitExceeded, "nodes.credits", fmt.Sprintf("distinct controller maximum is %d", MaxControllers))
	}
	sort.Slice(request.PriorControllerPaid, func(i, j int) bool {
		return request.PriorControllerPaid[i].ControllerID < request.PriorControllerPaid[j].ControllerID
	})

	return &validatedRequest{
		request:       request,
		envelope:      envelope,
		minimumPayout: minimum,
		programCap:    programCap,
		hasProgramCap: hasProgramCap,
		priorPaid:     priorPaid,
		nodes:         nodes,
	}, nil
}

func validatePolicy(policy Policy) (*big.Int, *big.Int, bool, error) {
	shares := []uint32{policy.DirectPPM, policy.UpstreamPPM, policy.DownstreamPPM, policy.BaseCommonsPPM}
	var shareSum uint64
	for _, share := range shares {
		if uint64(share) > PPM {
			return nil, nil, false, inputError(CodeInvalidPolicySum, "policy", fmt.Sprintf("each top-level share must not exceed %d", PPM))
		}
		shareSum += uint64(share)
	}
	if shareSum != PPM {
		return nil, nil, false, inputError(CodeInvalidPolicySum, "policy", fmt.Sprintf("top-level shares sum to %d; must equal %d", shareSum, PPM))
	}
	continuations := []struct {
		path  string
		value uint32
	}{
		{"policy.upstream_continuation_ppm", policy.UpstreamContinuationPPM},
		{"policy.downstream_continuation_ppm", policy.DownstreamContinuationPPM},
	}
	for _, item := range continuations {
		if uint64(item.value) >= PPM {
			return nil, nil, false, inputError(CodeInvalidContinuation, item.path, fmt.Sprintf("must be less than %d", PPM))
		}
	}
	depths := []struct {
		path  string
		value uint32
	}{
		{"policy.upstream_max_depth", policy.UpstreamMaxDepth},
		{"policy.downstream_max_depth", policy.DownstreamMaxDepth},
	}
	for _, item := range depths {
		if item.value == 0 || item.value > MaxDepth {
			return nil, nil, false, inputError(CodeInvalidDepth, item.path, fmt.Sprintf("must be in [1,%d]", MaxDepth))
		}
	}
	if policy.EnvelopeControllerCapPPM == 0 || uint64(policy.EnvelopeControllerCapPPM) > PPM {
		return nil, nil, false, inputError(CodeInvalidInput, "policy.envelope_controller_cap_ppm", fmt.Sprintf("must be in [1,%d]", PPM))
	}
	minimum, err := parseAmount(policy.MinProjectedPayoutUzrn, "policy.min_projected_payout_uzrn")
	if err != nil {
		return nil, nil, false, err
	}
	programCap := new(big.Int)
	hasProgramCap := policy.ProgramWindowCapUzrn != ""
	if hasProgramCap {
		programCap, err = parseAmount(policy.ProgramWindowCapUzrn, "policy.program_window_cap_uzrn")
		if err != nil {
			return nil, nil, false, err
		}
	}
	if !domainPattern.MatchString(policy.DomainID) {
		return nil, nil, false, inputError(CodeInvalidID, "policy.domain_id", "must be lower snake case and at most 64 bytes")
	}
	if policy.DomainRevision == 0 {
		return nil, nil, false, inputError(CodeInvalidInput, "policy.domain_revision", "must be positive")
	}
	if err := validateDigest(policy.PolicyDigest, "policy.policy_digest", CodeInvalidPolicyDigest); err != nil {
		return nil, nil, false, err
	}
	if expected := ComputePolicyDigest(policy); policy.PolicyDigest != expected {
		return nil, nil, false, inputError(CodeInvalidPolicyDigest, "policy.policy_digest", fmt.Sprintf("must equal canonical policy digest %q", expected))
	}
	return minimum, programCap, hasProgramCap, nil
}

// CanonicalPolicyPreimage returns the exact JSON byte string committed to by a
// PolicyDigest. The digest field itself is excluded to avoid self-reference.
func CanonicalPolicyPreimage(policy Policy) string {
	return fmt.Sprintf(
		`{"base_commons_ppm":%d,"direct_ppm":%d,"domain_id":%s,"domain_revision":%d,"downstream_continuation_ppm":%d,"downstream_max_depth":%d,"downstream_ppm":%d,"envelope_controller_cap_ppm":%d,"min_projected_payout_uzrn":%s,"program_window_cap_uzrn":%s,"upstream_continuation_ppm":%d,"upstream_max_depth":%d,"upstream_ppm":%d}`,
		policy.BaseCommonsPPM,
		policy.DirectPPM,
		jsonString(policy.DomainID),
		policy.DomainRevision,
		policy.DownstreamContinuationPPM,
		policy.DownstreamMaxDepth,
		policy.DownstreamPPM,
		policy.EnvelopeControllerCapPPM,
		jsonString(policy.MinProjectedPayoutUzrn),
		jsonString(policy.ProgramWindowCapUzrn),
		policy.UpstreamContinuationPPM,
		policy.UpstreamMaxDepth,
		policy.UpstreamPPM,
	)
}

func jsonString(value string) string {
	encoded, _ := json.Marshal(value)
	return string(encoded)
}

// ComputePolicyDigest returns the lowercase SHA-256 commitment expected in a
// policy's PolicyDigest field.
func ComputePolicyDigest(policy Policy) string {
	digest := sha256.Sum256([]byte(CanonicalPolicyPreimage(policy)))
	return fmt.Sprintf("sha256:%x", digest)
}

func parseAmount(value, path string) (*big.Int, error) {
	// Bound work before the regular expression and big.Int parser. A 256-bit
	// unsigned integer needs at most 78 canonical decimal digits.
	if len(value) > MaxAmountDecimalDigits {
		return nil, limitError(CodeInvalidAmount, path, fmt.Sprintf("maximum is %d decimal digits", MaxAmountDecimalDigits))
	}
	if !amountPattern.MatchString(value) {
		return nil, inputError(CodeInvalidAmount, path, "must be a canonical non-negative decimal integer")
	}
	amount, ok := new(big.Int).SetString(value, 10)
	if !ok || amount.Sign() < 0 {
		return nil, inputError(CodeInvalidAmount, path, "cannot parse amount")
	}
	if amount.BitLen() > MaxAmountBits {
		return nil, limitError(CodeInvalidAmount, path, fmt.Sprintf("maximum is %d bits", MaxAmountBits))
	}
	return amount, nil
}

func validateID(value, path string) error {
	if len(value) > MaxIDBytes || !idPattern.MatchString(value) {
		return inputError(CodeInvalidID, path, "must be canonical lowercase ASCII and at most 128 bytes")
	}
	return nil
}

func validateDigest(value, path, code string) error {
	if !digestPattern.MatchString(value) {
		return inputError(code, path, "must be lowercase sha256:<64 hex characters>")
	}
	return nil
}

func validFundedMilestone(value string) bool {
	switch value {
	case MilestoneE2, MilestoneE3, MilestoneE4, MilestoneE5, MilestoneE6:
		return true
	default:
		return false
	}
}
