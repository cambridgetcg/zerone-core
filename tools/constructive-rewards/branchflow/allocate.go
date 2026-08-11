package branchflow

import (
	"fmt"
	"math/big"
	"sort"
)

const (
	topDirectKey                = "00:DIRECT"
	topUpstreamKey              = "01:UPSTREAM"
	topDownstreamKey            = "02:DOWNSTREAM"
	topBaseCommonsKey           = "03:COMMONS"
	commonsBase                 = "BASE_COMMONS"
	commonsDirectUnattributed   = "DIRECT_UNATTRIBUTED"
	commonsUpstreamUnattributed = "UPSTREAM_UNATTRIBUTED"
	commonsUpstreamTail         = "UPSTREAM_TAIL"
	commonsDownstreamUnattr     = "DOWNSTREAM_UNATTRIBUTED"
	commonsDownstreamTail       = "DOWNSTREAM_TAIL"
	commonsControllerCap        = "CONTROLLER_CAP"
	commonsMinimumPayout        = "MINIMUM_PAYOUT"
)

type rawAllocation struct {
	leg        Leg
	depth      uint32
	cluster    string
	controller string
	role       string
	receipt    string
	milestone  string
	amount     *big.Int
}

type weightedAllocation struct {
	line   rawAllocation
	weight *big.Int
}

type rawCommons struct {
	reason string
	leg    Leg
	depth  uint32
	amount *big.Int
}

type geometricBuckets struct {
	public []DepthBucket
	depths []*big.Int
	tail   *big.Int
}

// Allocate evaluates one closed outcome-reward envelope without reading or
// mutating chain state. The returned amounts are shadow projections only;
// MovesFunds and IntegrationReady are always false.
func Allocate(input Request) (Result, error) {
	validated, err := validateRequest(input)
	if err != nil {
		return Result{}, err
	}
	operations := &operationCounter{}
	graph, err := buildGraph(validated, operations)
	if err != nil {
		return Result{}, err
	}

	top, err := apportion(validated.envelope, ppmInt(PPM), []weightedKey{
		{key: topDirectKey, weight: ppmInt(uint64(validated.request.Policy.DirectPPM))},
		{key: topUpstreamKey, weight: ppmInt(uint64(validated.request.Policy.UpstreamPPM))},
		{key: topDownstreamKey, weight: ppmInt(uint64(validated.request.Policy.DownstreamPPM))},
		{key: topBaseCommonsKey, weight: ppmInt(uint64(validated.request.Policy.BaseCommonsPPM)), isCommons: true},
	})
	if err != nil {
		return Result{}, err
	}
	directTotal := findApportioned(top, topDirectKey)
	upstreamTotal := findApportioned(top, topUpstreamKey)
	downstreamTotal := findApportioned(top, topDownstreamKey)
	baseCommons := findApportioned(top, topBaseCommonsKey)

	allocations := make([]rawAllocation, 0)
	commons := make([]rawCommons, 0)
	appendCommons(&commons, commonsBase, LegCommons, 0, baseCommons)

	directAllocations, directUnattributed, err := allocateNodeCredits(
		directTotal, validated.request.FundedClusterID, validated.nodes[validated.request.FundedClusterID], LegDirect, 0,
	)
	if err != nil {
		return Result{}, err
	}
	if err := appendAllocationLines(&allocations, directAllocations); err != nil {
		return Result{}, err
	}
	appendCommons(&commons, commonsDirectUnattributed, LegDirect, 0, directUnattributed)

	upstreamBuckets, err := buildGeometricBuckets(
		upstreamTotal,
		LegUpstream,
		validated.request.Policy.UpstreamContinuationPPM,
		validated.request.Policy.UpstreamMaxDepth,
	)
	if err != nil {
		return Result{}, err
	}
	upstreamLayers, err := graph.trace(
		validated.request.FundedClusterID,
		validated.request.Policy.UpstreamMaxDepth,
		operations,
	)
	if err != nil {
		return Result{}, err
	}
	for index, bucket := range upstreamBuckets.depths {
		lines, unattributed, allocateErr := allocateUpstreamDepth(bucket, uint32(index+1), upstreamLayers[index], validated.nodes)
		if allocateErr != nil {
			return Result{}, allocateErr
		}
		if err := appendAllocationLines(&allocations, lines); err != nil {
			return Result{}, err
		}
		appendCommons(&commons, commonsUpstreamUnattributed, LegUpstream, uint32(index+1), unattributed)
	}
	appendCommons(&commons, commonsUpstreamTail, LegUpstream, 0, upstreamBuckets.tail)

	downstreamBuckets, err := buildGeometricBuckets(
		downstreamTotal,
		LegDownstream,
		validated.request.Policy.DownstreamContinuationPPM,
		validated.request.Policy.DownstreamMaxDepth,
	)
	if err != nil {
		return Result{}, err
	}
	downstreamLines, downstreamCommons, err := allocateDownstream(
		downstreamBuckets.depths, validated, graph, operations,
	)
	if err != nil {
		return Result{}, err
	}
	if err := appendAllocationLines(&allocations, downstreamLines); err != nil {
		return Result{}, err
	}
	commons = append(commons, downstreamCommons...)
	appendCommons(&commons, commonsDownstreamTail, LegDownstream, 0, downstreamBuckets.tail)

	allocations = coalesceAllocations(allocations)
	allocations, capCommons, err := applyControllerCaps(allocations, validated)
	if err != nil {
		return Result{}, err
	}
	commons = append(commons, capCommons...)
	allocations, minimumCommons := applyMinimumPayout(allocations, validated.minimumPayout)
	commons = append(commons, minimumCommons...)
	allocations = coalesceAllocations(allocations)
	commons = coalesceCommons(commons)

	paid := new(big.Int)
	publicAllocations := make([]Allocation, 0, len(allocations))
	for _, line := range allocations {
		if line.amount.Sign() == 0 {
			continue
		}
		paid.Add(paid, line.amount)
		publicAllocations = append(publicAllocations, Allocation{
			Leg:           line.leg,
			Depth:         line.depth,
			ClusterID:     line.cluster,
			ControllerID:  line.controller,
			RoleID:        line.role,
			ReceiptKey:    line.receipt,
			Milestone:     line.milestone,
			ProjectedUzrn: line.amount.String(),
		})
	}
	commonsTotal := new(big.Int)
	publicCommons := make([]CommonsAllocation, 0, len(commons))
	for _, line := range commons {
		if line.amount.Sign() == 0 {
			continue
		}
		commonsTotal.Add(commonsTotal, line.amount)
		publicCommons = append(publicCommons, CommonsAllocation{
			Reason: line.reason, SourceLeg: line.leg, Depth: line.depth, ProjectedUzrn: line.amount.String(),
		})
	}
	if new(big.Int).Add(new(big.Int).Set(paid), commonsTotal).Cmp(validated.envelope) != 0 {
		return Result{}, invariantError(fmt.Sprintf(
			"projected paid %s plus commons %s does not equal envelope %s",
			paid, commonsTotal, validated.envelope,
		))
	}

	receiptUses := make([]ReceiptUse, 0, len(validated.request.DescendantImpacts))
	for _, impact := range validated.request.DescendantImpacts {
		receiptUses = append(receiptUses, ReceiptUse{
			ReceiptKey: impact.ReceiptKey, EconomicSlotID: impact.EconomicSlotID,
		})
	}
	sort.Slice(receiptUses, func(i, j int) bool {
		if receiptUses[i].ReceiptKey != receiptUses[j].ReceiptKey {
			return receiptUses[i].ReceiptKey < receiptUses[j].ReceiptKey
		}
		return receiptUses[i].EconomicSlotID < receiptUses[j].EconomicSlotID
	})

	return Result{
		Schema:               Schema,
		AlgorithmVersion:     AlgorithmVersion,
		Assurance:            Assurance,
		EconomicEffect:       EconomicEffect,
		MovesFunds:           false,
		IntegrationReady:     false,
		EnvelopeID:           validated.request.EnvelopeID,
		FundedClusterID:      validated.request.FundedClusterID,
		FundedMilestone:      validated.request.FundedMilestone,
		EnvelopeUzrn:         validated.envelope.String(),
		Policy:               validated.request.Policy,
		NormalizedEdges:      append([]NormalizedEdge{}, graph.normalized...),
		UpstreamBuckets:      upstreamBuckets.public,
		DownstreamBuckets:    downstreamBuckets.public,
		Allocations:          publicAllocations,
		Commons:              publicCommons,
		NewReceiptUses:       receiptUses,
		ProjectedPaidUzrn:    paid.String(),
		ProjectedCommonsUzrn: commonsTotal.String(),
		ConservationCheck:    ConservationOK,
	}, nil
}

func allocateNodeCredits(total *big.Int, clusterID string, node Node, leg Leg, depth uint32) ([]rawAllocation, *big.Int, error) {
	weighted := make([]weightedAllocation, 0, len(node.Credits))
	var creditSum uint64
	if node.Mode == NodeModePayAndPropagate {
		for _, credit := range node.Credits {
			creditSum += uint64(credit.WeightPPM)
			weighted = append(weighted, weightedAllocation{
				line: rawAllocation{
					leg: leg, depth: depth, cluster: clusterID,
					controller: credit.ControllerID, role: credit.RoleID,
				},
				weight: ppmInt(uint64(credit.WeightPPM)),
			})
		}
	}
	return apportionAllocationWeights(total, ppmInt(PPM), weighted, ppmInt(PPM-creditSum))
}

func allocateUpstreamDepth(total *big.Int, depth uint32, flow map[string]*big.Int, nodes map[string]Node) ([]rawAllocation, *big.Int, error) {
	weighted := make([]weightedAllocation, 0)
	lineWeightTotal := new(big.Int)
	for _, clusterID := range sortedAmountKeys(flow) {
		node := nodes[clusterID]
		if node.Mode != NodeModePayAndPropagate {
			continue
		}
		for _, credit := range node.Credits {
			weight := floorRatio(flow[clusterID], ppmInt(uint64(credit.WeightPPM)), ppmInt(PPM))
			if weight.Sign() == 0 {
				continue
			}
			lineWeightTotal.Add(lineWeightTotal, weight)
			weighted = append(weighted, weightedAllocation{
				line: rawAllocation{
					leg: LegUpstream, depth: depth, cluster: clusterID,
					controller: credit.ControllerID, role: credit.RoleID,
				},
				weight: weight,
			})
		}
	}
	if lineWeightTotal.Cmp(ppmInt(PPM)) > 0 {
		return nil, nil, invariantError("upstream claimant weight exceeds PPM")
	}
	commonsWeight := new(big.Int).Sub(ppmInt(PPM), lineWeightTotal)
	return apportionAllocationWeights(total, ppmInt(PPM), weighted, commonsWeight)
}

func allocateDownstream(
	buckets []*big.Int,
	validated *validatedRequest,
	graph *dependencyGraph,
	operations *operationCounter,
) ([]rawAllocation, []rawCommons, error) {
	type impactWeight struct {
		impact Impact
		share  uint32
	}
	type descendantGroup struct {
		cluster          string
		capacitySharePPM uint32
		items            []impactWeight
		layers           []map[string]*big.Int
	}
	fundedControllers := make(map[string]struct{})
	for _, credit := range validated.nodes[validated.request.FundedClusterID].Credits {
		fundedControllers[credit.ControllerID] = struct{}{}
	}

	byDescendant := make(map[string][]Impact)
	for _, impact := range validated.request.DescendantImpacts {
		byDescendant[impact.DescendantClusterID] = append(byDescendant[impact.DescendantClusterID], impact)
	}
	descendants := make([]string, 0, len(byDescendant))
	for descendant := range byDescendant {
		descendants = append(descendants, descendant)
	}
	sort.Strings(descendants)
	groups := make([]descendantGroup, 0, len(descendants))
	for _, descendant := range descendants {
		impacts := byDescendant[descendant]
		var rawSum uint64
		for _, impact := range impacts {
			rawSum += uint64(impact.ImpactPPM)
		}
		denominator := rawSum
		if denominator < PPM {
			denominator = PPM
		}
		capacityShare := rawSum
		if capacityShare > PPM {
			capacityShare = PPM
		}
		items := make([]impactWeight, 0, len(impacts))
		for _, impact := range impacts {
			items = append(items, impactWeight{
				impact: impact,
				share:  uint32(PPM * uint64(impact.ImpactPPM) / denominator),
			})
		}
		layers, err := graph.trace(descendant, validated.request.Policy.DownstreamMaxDepth, operations)
		if err != nil {
			return nil, nil, err
		}
		linked := false
		for _, layer := range layers {
			if amount := layer[validated.request.FundedClusterID]; amount != nil && amount.Sign() > 0 {
				linked = true
				break
			}
		}
		if !linked {
			return nil, nil, inputError(
				CodeImpactNotLinked,
				"descendant_impacts",
				fmt.Sprintf("descendant %q has no positive dependency flow to funded cluster %q within depth %d", descendant, validated.request.FundedClusterID, validated.request.Policy.DownstreamMaxDepth),
			)
		}
		groups = append(groups, descendantGroup{
			cluster: descendant, capacitySharePPM: uint32(capacityShare), items: items, layers: layers,
		})
	}

	allLines := make([]rawAllocation, 0)
	commons := make([]rawCommons, 0, len(buckets))
	for index, bucket := range buckets {
		depth := uint32(index + 1)
		effective := make([]weightedAllocation, 0)
		terminalWeight := new(big.Int)
		for _, group := range groups {
			reliance := group.layers[index][validated.request.FundedClusterID]
			if reliance == nil || reliance.Sign() == 0 {
				continue
			}
			node := validated.nodes[group.cluster]
			// Compute one pre-role capacity per semantic descendant before
			// receipt-share floors. The difference between this aggregate
			// capacity and the receipt lines makes impact-normalization and
			// per-receipt precision loss terminal rather than competitor value.
			capacity := floorRatio(
				reliance,
				ppmInt(uint64(group.capacitySharePPM)),
				ppmInt(PPM),
			)
			terminalWeight.Add(terminalWeight, capacity)
			for _, item := range group.items {
				// A terminal disposition is an admitted cohort tombstone. Its
				// normalized impact remains in capacity but creates no claimant
				// line, so later invalidation cannot enrich a competitor.
				if item.impact.Disposition == ImpactDispositionTerminal {
					continue
				}
				for _, credit := range node.Credits {
					lineWeight := floorPPMProduct(reliance, item.share, credit.WeightPPM)
					// The supplied canonical controller identity lets the pure
					// calculator enforce the obvious independence boundary. Hidden
					// or correlated control still belongs to external adjudication.
					if _, sameController := fundedControllers[credit.ControllerID]; sameController {
						continue
					}
					if lineWeight.Sign() == 0 {
						continue
					}
					terminalWeight.Sub(terminalWeight, lineWeight)
					if len(effective) >= MaxAllocationLines {
						return nil, nil, limitError(CodeAllocationLineLimit, "downstream.effective", fmt.Sprintf("maximum is %d", MaxAllocationLines))
					}
					if err := operations.add(1); err != nil {
						return nil, nil, err
					}
					effective = append(effective, weightedAllocation{
						line: rawAllocation{
							leg: LegDownstream, depth: depth, cluster: group.cluster,
							controller: credit.ControllerID, role: credit.RoleID,
							receipt: item.impact.ReceiptKey, milestone: item.impact.Milestone,
						},
						weight: lineWeight,
					})
				}
			}
		}
		effective, err := saturateDownstreamControllers(effective)
		if err != nil {
			return nil, nil, err
		}
		if terminalWeight.Sign() < 0 {
			return nil, nil, invariantError("downstream terminal weight became negative")
		}
		totalWeight := new(big.Int)
		for _, item := range effective {
			totalWeight.Add(totalWeight, item.weight)
		}
		denominator := new(big.Int).Add(new(big.Int).Set(totalWeight), terminalWeight)
		if denominator.Cmp(ppmInt(PPM)) < 0 {
			denominator.SetUint64(PPM)
		}
		commonsWeight := new(big.Int).Sub(new(big.Int).Set(denominator), totalWeight)
		lines, unattributed, err := apportionAllocationWeights(bucket, denominator, effective, commonsWeight)
		if err != nil {
			return nil, nil, err
		}
		if err := appendAllocationLines(&allLines, lines); err != nil {
			return nil, nil, err
		}
		appendCommons(&commons, commonsDownstreamUnattr, LegDownstream, depth, unattributed)
	}
	return allLines, commons, nil
}

// floorPPMProduct implements one conservative downstream term:
// floor(reliance * impactShare * creditShare / PPM^2). It intentionally uses
// one quotient, matching the public arithmetic version, rather than flooring
// between the impact and credit factors.
func floorPPMProduct(reliance *big.Int, impactShare, creditShare uint32) *big.Int {
	numerator := new(big.Int).Mul(reliance, ppmInt(uint64(impactShare)))
	numerator.Mul(numerator, ppmInt(uint64(creditShare)))
	denominator := new(big.Int).Mul(ppmInt(PPM), ppmInt(PPM))
	return numerator.Quo(numerator, denominator)
}

func saturateDownstreamControllers(input []weightedAllocation) ([]weightedAllocation, error) {
	byController := make(map[string][]int)
	for index, item := range input {
		byController[item.line.controller] = append(byController[item.line.controller], index)
	}
	controllers := make([]string, 0, len(byController))
	for controller := range byController {
		controllers = append(controllers, controller)
	}
	sort.Strings(controllers)
	output := make([]weightedAllocation, len(input))
	for i, item := range input {
		output[i] = weightedAllocation{line: item.line, weight: new(big.Int).Set(item.weight)}
	}
	for _, controller := range controllers {
		indices := byController[controller]
		total := new(big.Int)
		for _, index := range indices {
			total.Add(total, output[index].weight)
		}
		if total.Cmp(ppmInt(PPM)) <= 0 {
			continue
		}
		weights := make([]weightedKey, 0, len(indices))
		for _, index := range indices {
			weights = append(weights, weightedKey{
				key: rawAllocationKey(output[index].line), weight: output[index].weight,
			})
		}
		shares, err := apportion(ppmInt(PPM), total, weights)
		if err != nil {
			return nil, err
		}
		indexed := indexApportioned(shares)
		for _, index := range indices {
			output[index].weight = indexed[rawAllocationKey(output[index].line)]
		}
	}
	return output, nil
}

func buildGeometricBuckets(total *big.Int, direction Leg, continuation uint32, maxDepth uint32) (geometricBuckets, error) {
	scale := ppmInt(PPM)
	q := ppmInt(uint64(continuation))
	denominator := new(big.Int).Exp(scale, ppmInt(uint64(maxDepth)), nil)
	weights := make([]weightedKey, 0, maxDepth+1)
	for depth := uint32(1); depth <= maxDepth; depth++ {
		weight := new(big.Int).Sub(scale, q)
		weight.Mul(weight, new(big.Int).Exp(q, ppmInt(uint64(depth-1)), nil))
		weight.Mul(weight, new(big.Int).Exp(scale, ppmInt(uint64(maxDepth-depth)), nil))
		weights = append(weights, weightedKey{
			key: fmt.Sprintf("depth:%02d", depth), weight: weight,
		})
	}
	tailKey := "tail"
	weights = append(weights, weightedKey{
		key: tailKey, weight: new(big.Int).Exp(q, ppmInt(uint64(maxDepth)), nil), isCommons: true,
	})
	amounts, err := apportion(total, denominator, weights)
	if err != nil {
		return geometricBuckets{}, err
	}
	result := geometricBuckets{
		public: make([]DepthBucket, 0, maxDepth+1),
		depths: make([]*big.Int, 0, maxDepth),
		tail:   findApportioned(amounts, tailKey),
	}
	for depth := uint32(1); depth <= maxDepth; depth++ {
		amount := findApportioned(amounts, fmt.Sprintf("depth:%02d", depth))
		result.depths = append(result.depths, amount)
		result.public = append(result.public, DepthBucket{
			Direction: direction, Depth: depth, ProjectedUzrn: amount.String(),
		})
	}
	result.public = append(result.public, DepthBucket{
		Direction: direction, Tail: true, ProjectedUzrn: result.tail.String(),
	})
	return result, nil
}

func apportionAllocationWeights(total, denominator *big.Int, weighted []weightedAllocation, commonsWeight *big.Int) ([]rawAllocation, *big.Int, error) {
	byController := make(map[string][]weightedAllocation)
	controllerWeights := make(map[string]*big.Int)
	for _, item := range weighted {
		byController[item.line.controller] = append(byController[item.line.controller], item)
		if controllerWeights[item.line.controller] == nil {
			controllerWeights[item.line.controller] = new(big.Int)
		}
		controllerWeights[item.line.controller].Add(controllerWeights[item.line.controller], item.weight)
	}
	controllers := make([]string, 0, len(byController))
	for controller := range byController {
		controllers = append(controllers, controller)
	}
	sort.Strings(controllers)

	if commonsWeight == nil || commonsWeight.Sign() < 0 {
		return nil, nil, invariantError("allocation commons weight must be non-negative")
	}
	partitionWeight := new(big.Int).Set(commonsWeight)
	for _, controller := range controllers {
		partitionWeight.Add(partitionWeight, controllerWeights[controller])
	}
	if partitionWeight.Cmp(denominator) != 0 {
		return nil, nil, invariantError(fmt.Sprintf(
			"allocation weights sum %s, denominator %s", partitionWeight, denominator,
		))
	}

	// Controller claimants receive only their exact floor. All cross-controller
	// monetary residual stays commons, so removing or invalidating one claimant
	// can never move a remainder unit to a competitor. Hamilton is used only to
	// divide an already fixed controller amount among that controller's lines.
	lines := make([]rawAllocation, 0, len(weighted))
	claimed := new(big.Int)
	for _, controller := range controllers {
		controllerAmount := floorRatio(total, controllerWeights[controller], denominator)
		if controllerAmount.Sign() == 0 {
			continue
		}
		claimed.Add(claimed, controllerAmount)
		items := byController[controller]
		innerWeights := make([]weightedKey, 0, len(items))
		for _, item := range items {
			innerWeights = append(innerWeights, weightedKey{
				key: rawAllocationKey(item.line), weight: item.weight,
			})
		}
		innerAmounts, err := apportion(controllerAmount, controllerWeights[controller], innerWeights)
		if err != nil {
			return nil, nil, err
		}
		inner := indexApportioned(innerAmounts)
		for _, item := range items {
			amount := inner[rawAllocationKey(item.line)]
			if amount.Sign() == 0 {
				continue
			}
			line := item.line
			line.amount = amount
			lines = append(lines, line)
		}
	}
	return lines, new(big.Int).Sub(new(big.Int).Set(total), claimed), nil
}

func applyControllerCaps(input []rawAllocation, validated *validatedRequest) ([]rawAllocation, []rawCommons, error) {
	byController := make(map[string][]int)
	for index, line := range input {
		byController[line.controller] = append(byController[line.controller], index)
	}
	controllers := make([]string, 0, len(byController))
	for controller := range byController {
		controllers = append(controllers, controller)
	}
	sort.Strings(controllers)
	output := cloneRawAllocations(input)
	commons := make([]rawCommons, 0)
	envelopeCap := floorRatio(
		validated.envelope,
		ppmInt(uint64(validated.request.Policy.EnvelopeControllerCapPPM)),
		ppmInt(PPM),
	)
	for _, controller := range controllers {
		indices := byController[controller]
		rawTotal := new(big.Int)
		for _, index := range indices {
			rawTotal.Add(rawTotal, output[index].amount)
		}
		allowed := new(big.Int).Set(rawTotal)
		if allowed.Cmp(envelopeCap) > 0 {
			allowed.Set(envelopeCap)
		}
		if validated.hasProgramCap {
			remaining := new(big.Int).Set(validated.programCap)
			if prior := validated.priorPaid[controller]; prior != nil {
				remaining.Sub(remaining, prior)
				if remaining.Sign() < 0 {
					remaining.SetInt64(0)
				}
			}
			if allowed.Cmp(remaining) > 0 {
				allowed.Set(remaining)
			}
		}
		if allowed.Cmp(rawTotal) == 0 {
			continue
		}
		capped := make(map[string]*big.Int, len(indices))
		if allowed.Sign() > 0 {
			weights := make([]weightedKey, 0, len(indices))
			for _, index := range indices {
				weights = append(weights, weightedKey{
					key: rawAllocationKey(output[index]), weight: output[index].amount,
				})
			}
			amounts, err := apportion(allowed, rawTotal, weights)
			if err != nil {
				return nil, nil, err
			}
			capped = indexApportioned(amounts)
		}
		for _, index := range indices {
			oldAmount := new(big.Int).Set(output[index].amount)
			newAmount := new(big.Int)
			if value := capped[rawAllocationKey(output[index])]; value != nil {
				newAmount.Set(value)
			}
			overflow := new(big.Int).Sub(oldAmount, newAmount)
			appendCommons(&commons, commonsControllerCap, output[index].leg, output[index].depth, overflow)
			output[index].amount = newAmount
		}
	}
	return output, commons, nil
}

func applyMinimumPayout(input []rawAllocation, minimum *big.Int) ([]rawAllocation, []rawCommons) {
	if minimum.Sign() == 0 {
		return cloneRawAllocations(input), nil
	}
	totals := make(map[string]*big.Int)
	for _, line := range input {
		if totals[line.controller] == nil {
			totals[line.controller] = new(big.Int)
		}
		totals[line.controller].Add(totals[line.controller], line.amount)
	}
	output := cloneRawAllocations(input)
	commons := make([]rawCommons, 0)
	for index := range output {
		if totals[output[index].controller].Cmp(minimum) >= 0 {
			continue
		}
		appendCommons(&commons, commonsMinimumPayout, output[index].leg, output[index].depth, output[index].amount)
		output[index].amount = new(big.Int)
	}
	return output, commons
}

func appendAllocationLines(destination *[]rawAllocation, lines []rawAllocation) error {
	if len(*destination) > MaxAllocationLines-len(lines) {
		return limitError(CodeAllocationLineLimit, "allocations", fmt.Sprintf("maximum is %d", MaxAllocationLines))
	}
	*destination = append(*destination, lines...)
	return nil
}

func appendCommons(destination *[]rawCommons, reason string, leg Leg, depth uint32, amount *big.Int) {
	if amount == nil || amount.Sign() == 0 {
		return
	}
	*destination = append(*destination, rawCommons{
		reason: reason, leg: leg, depth: depth, amount: new(big.Int).Set(amount),
	})
}

func coalesceAllocations(input []rawAllocation) []rawAllocation {
	byKey := make(map[string]rawAllocation, len(input))
	for _, line := range input {
		if line.amount == nil || line.amount.Sign() == 0 {
			continue
		}
		key := rawAllocationKey(line)
		if existing, found := byKey[key]; found {
			existing.amount.Add(existing.amount, line.amount)
			byKey[key] = existing
		} else {
			line.amount = new(big.Int).Set(line.amount)
			byKey[key] = line
		}
	}
	keys := make([]string, 0, len(byKey))
	for key := range byKey {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	output := make([]rawAllocation, 0, len(keys))
	for _, key := range keys {
		output = append(output, byKey[key])
	}
	return output
}

func coalesceCommons(input []rawCommons) []rawCommons {
	byKey := make(map[string]rawCommons, len(input))
	for _, line := range input {
		if line.amount == nil || line.amount.Sign() == 0 {
			continue
		}
		key := fmt.Sprintf("%s\x00%02d\x00%s\x00%08d", line.reason, canonicalLegRank(line.leg), line.leg, line.depth)
		if existing, found := byKey[key]; found {
			existing.amount.Add(existing.amount, line.amount)
			byKey[key] = existing
		} else {
			line.amount = new(big.Int).Set(line.amount)
			byKey[key] = line
		}
	}
	keys := make([]string, 0, len(byKey))
	for key := range byKey {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	output := make([]rawCommons, 0, len(keys))
	for _, key := range keys {
		output = append(output, byKey[key])
	}
	return output
}

func cloneRawAllocations(input []rawAllocation) []rawAllocation {
	output := make([]rawAllocation, len(input))
	for index, line := range input {
		output[index] = line
		output[index].amount = new(big.Int).Set(line.amount)
	}
	return output
}

func rawAllocationKey(line rawAllocation) string {
	return fmt.Sprintf(
		"%02d\x00%s\x00%08d\x00%s\x00%s\x00%s\x00%s\x00%s",
		canonicalLegRank(line.leg), line.leg, line.depth, line.cluster, line.controller, line.role, line.receipt, line.milestone,
	)
}

func canonicalLegRank(leg Leg) int {
	switch leg {
	case LegDirect:
		return 0
	case LegUpstream:
		return 1
	case LegDownstream:
		return 2
	case LegCommons:
		return 3
	default:
		return 99
	}
}
