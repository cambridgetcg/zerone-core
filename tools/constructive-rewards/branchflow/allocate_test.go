package branchflow

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"slices"
	"strings"
	"sync"
	"testing"
)

func TestDefaultPolicyCommitment(t *testing.T) {
	policy := DefaultPolicy()
	if policy.DirectPPM != 600_000 || policy.UpstreamPPM != 100_000 || policy.DownstreamPPM != 300_000 || policy.BaseCommonsPPM != 0 {
		t.Fatalf("unexpected reference partition: %+v", policy)
	}
	if policy.EnvelopeControllerCapPPM != uint32(PPM) {
		t.Fatalf("reference cap is binding: %d", policy.EnvelopeControllerCapPPM)
	}
	if got := CanonicalPolicyPreimage(policy); got != ReferencePolicyPreimage {
		t.Fatalf("reference preimage drifted:\n got: %s\nwant: %s", got, ReferencePolicyPreimage)
	}
	if got := ComputePolicyDigest(policy); got != ReferencePolicyDigest || policy.PolicyDigest != got {
		t.Fatalf("reference digest drifted: computed=%s policy=%s want=%s", got, policy.PolicyDigest, ReferencePolicyDigest)
	}
	invalidStringPolicy := policy
	invalidStringPolicy.DomainID = "bad\x01domain"
	if preimage := CanonicalPolicyPreimage(invalidStringPolicy); !json.Valid([]byte(preimage)) {
		t.Fatalf("public preimage helper emitted non-JSON string: %q", preimage)
	}
}

func TestReferenceAllocationConservesAndExposesEffectFences(t *testing.T) {
	request := referenceRequest()
	before := mustJSON(t, request)
	result, err := Allocate(request)
	if err != nil {
		t.Fatalf("Allocate: %v", err)
	}
	if after := mustJSON(t, request); string(after) != string(before) {
		t.Fatal("Allocate mutated its request")
	}
	if result.Assurance != Assurance || result.EconomicEffect != EconomicEffect || result.MovesFunds || result.IntegrationReady {
		t.Fatalf("effect fences not explicit: %+v", result)
	}
	if result.FundedClusterID != "root" || result.FundedMilestone != MilestoneE5 {
		t.Fatalf("funded subject drifted: cluster=%q milestone=%q", result.FundedClusterID, result.FundedMilestone)
	}
	if result.ProjectedPaidUzrn != "82500000" || result.ProjectedCommonsUzrn != "17500000" {
		t.Fatalf("unexpected totals: paid=%s commons=%s", result.ProjectedPaidUzrn, result.ProjectedCommonsUzrn)
	}
	assertConserved(t, result)
	assertAllocation(t, result, LegDirect, 0, "root-controller", "", "60000000")
	assertAllocation(t, result, LegUpstream, 1, "parent-controller", "", "5000000")
	assertAllocation(t, result, LegUpstream, 2, "grand-controller", "", "2500000")
	assertAllocation(t, result, LegDownstream, 1, "child-controller", receipt("a"), "15000000")
	if got := []Leg{result.Allocations[0].Leg, result.Allocations[1].Leg, result.Allocations[2].Leg, result.Allocations[3].Leg}; !slices.Equal(got, []Leg{LegDirect, LegUpstream, LegUpstream, LegDownstream}) {
		t.Fatalf("canonical leg order drifted: %v", got)
	}

	assertBucketAmounts(t, result.UpstreamBuckets, []string{"5000000", "2500000", "1250000", "625000", "312500", "312500"})
	assertBucketAmounts(t, result.DownstreamBuckets, []string{"15000000", "7500000", "3750000", "1875000", "937500", "937500"})
	if len(result.NewReceiptUses) != 1 || result.NewReceiptUses[0].ReceiptKey != receipt("a") {
		t.Fatalf("unexpected receipt uses: %+v", result.NewReceiptUses)
	}
}

func TestAbsolutePathSpecificDepthFlow(t *testing.T) {
	t.Run("sparse depth is not renormalized", func(t *testing.T) {
		request := referenceRequest()
		request.Nodes = []Node{
			node("root", NodeModePayAndPropagate, credit("root-controller", "origin", 1_000_000)),
			node("pass", NodeModePassThrough),
			node("ancestor", NodeModePayAndPropagate, credit("ancestor-controller", "dependency", 1_000_000)),
			node("child", NodeModePayAndPropagate, credit("child-controller", "impact", 1_000_000)),
		}
		request.Edges = []Edge{
			{ChildClusterID: "root", ParentClusterID: "pass", RawDependencyPPM: 1_000_000},
			{ChildClusterID: "pass", ParentClusterID: "ancestor", RawDependencyPPM: 1_000_000},
			{ChildClusterID: "child", ParentClusterID: "root", RawDependencyPPM: 1_000_000},
		}
		result, err := Allocate(request)
		if err != nil {
			t.Fatalf("Allocate: %v", err)
		}
		assertAllocation(t, result, LegUpstream, 2, "ancestor-controller", "", "2500000")
		if got := allocationTotal(result, LegUpstream, 1, "ancestor-controller", ""); got.Sign() != 0 {
			t.Fatalf("empty depth one enriched depth two ancestor: %s", got)
		}
		assertConserved(t, result)
	})

	t.Run("convergent paths retain exact depths", func(t *testing.T) {
		request := referenceRequest()
		request.Nodes = []Node{
			node("root", NodeModePayAndPropagate, credit("root-controller", "origin", 1_000_000)),
			node("ancestor", NodeModePayAndPropagate, credit("ancestor-controller", "dependency", 1_000_000)),
			node("mid", NodeModePassThrough),
			node("child", NodeModePayAndPropagate, credit("child-controller", "impact", 1_000_000)),
		}
		request.Edges = []Edge{
			{ChildClusterID: "root", ParentClusterID: "ancestor", RawDependencyPPM: 500_000},
			{ChildClusterID: "root", ParentClusterID: "mid", RawDependencyPPM: 500_000},
			{ChildClusterID: "mid", ParentClusterID: "ancestor", RawDependencyPPM: 1_000_000},
			{ChildClusterID: "child", ParentClusterID: "root", RawDependencyPPM: 1_000_000},
		}
		result, err := Allocate(request)
		if err != nil {
			t.Fatalf("Allocate: %v", err)
		}
		assertAllocation(t, result, LegUpstream, 1, "ancestor-controller", "", "2500000")
		assertAllocation(t, result, LegUpstream, 2, "ancestor-controller", "", "1250000")
		assertConserved(t, result)
	})

	t.Run("blocked node terminates payment and propagation", func(t *testing.T) {
		request := referenceRequest()
		request.Nodes[1].Mode = NodeModeBlocked
		request.Nodes[1].Credits = nil
		result, err := Allocate(request)
		if err != nil {
			t.Fatalf("Allocate: %v", err)
		}
		if got := allocationTotal(result, LegUpstream, 0, "", ""); got.Sign() != 0 {
			t.Fatalf("blocked lineage received or propagated upstream value: %s", got)
		}
		if result.ProjectedPaidUzrn != "75000000" || result.ProjectedCommonsUzrn != "25000000" {
			t.Fatalf("unexpected blocked totals: paid=%s commons=%s", result.ProjectedPaidUzrn, result.ProjectedCommonsUzrn)
		}
		assertConserved(t, result)
	})
}

func TestGraphFlooringExposesZeroSharesAndRejectsRawCycle(t *testing.T) {
	request := referenceRequest()
	request.Nodes = append(request.Nodes,
		node("tiny", NodeModePayAndPropagate, credit("tiny-controller", "dependency", 1_000_000)),
		node("parent-two", NodeModePayAndPropagate, credit("parent-two-controller", "dependency", 1_000_000)),
	)
	request.Edges = []Edge{
		{ChildClusterID: "root", ParentClusterID: "tiny", RawDependencyPPM: 1},
		{ChildClusterID: "root", ParentClusterID: "parent", RawDependencyPPM: 1_000_000},
		{ChildClusterID: "root", ParentClusterID: "parent-two", RawDependencyPPM: 1_000_000},
		{ChildClusterID: "child", ParentClusterID: "root", RawDependencyPPM: 1_000_000},
	}
	result, err := Allocate(request)
	if err != nil {
		t.Fatalf("Allocate: %v", err)
	}
	var foundZero bool
	var rootShare uint64
	for _, edge := range result.NormalizedEdges {
		if edge.ChildClusterID == "root" {
			rootShare += uint64(edge.SharePPM)
		}
		if edge.ChildClusterID == "root" && edge.ParentClusterID == "tiny" && edge.SharePPM == 0 {
			foundZero = true
		}
	}
	if !foundZero {
		t.Fatal("zero normalized share was hidden from result")
	}
	if rootShare != 999_998 {
		t.Fatalf("graph flooring did not preserve precision leakage: root shares=%d", rootShare)
	}

	request.Edges = append(request.Edges, Edge{
		ChildClusterID: "tiny", ParentClusterID: "root", RawDependencyPPM: 1_000_000,
	})
	_, err = Allocate(request)
	if !errors.Is(err, ErrInvalidGraph) || validationCode(err) != CodeGraphCycle {
		t.Fatalf("zero-share raw cycle was not rejected: %v", err)
	}
}

func TestDownstreamVisibleIndependenceAndImpactNormalization(t *testing.T) {
	t.Run("same funded controller routes to commons", func(t *testing.T) {
		request := referenceRequest()
		request.Nodes[3].Credits = []Credit{
			credit("root-controller", "related-impact", 500_000),
			credit("child-controller", "independent-impact", 500_000),
		}
		result, err := Allocate(request)
		if err != nil {
			t.Fatalf("Allocate: %v", err)
		}
		assertAllocation(t, result, LegDownstream, 1, "child-controller", receipt("a"), "7500000")
		if got := allocationTotal(result, LegDownstream, 1, "root-controller", receipt("a")); got.Sign() != 0 {
			t.Fatalf("funded controller received descendant allocation: %s", got)
		}
		if got := commonsTotal(result, commonsDownstreamUnattr, LegDownstream, 1); got.String() != "7500000" {
			t.Fatalf("same-control mass did not remain commons: %s", got)
		}
		assertConserved(t, result)
	})

	t.Run("excluded mass cannot enrich a saturated cohort", func(t *testing.T) {
		request := threeDescendantRequest()
		baseline, err := Allocate(request)
		if err != nil {
			t.Fatalf("baseline Allocate: %v", err)
		}
		for _, controller := range []string{"child-controller", "child-b-controller", "child-c-controller"} {
			assertAllocation(t, baseline, LegDownstream, 1, controller, "", "5000000")
		}

		request.Nodes[5].Credits = []Credit{credit("root-controller", "related-impact", 1_000_000)}
		result, err := Allocate(request)
		if err != nil {
			t.Fatalf("overlap Allocate: %v", err)
		}
		assertAllocation(t, result, LegDownstream, 1, "child-controller", "", "5000000")
		assertAllocation(t, result, LegDownstream, 1, "child-b-controller", "", "5000000")
		if got := allocationTotal(result, LegDownstream, 1, "root-controller", ""); got.Sign() != 0 {
			t.Fatalf("funded controller received descendant allocation: %s", got)
		}
		if got := commonsTotal(result, commonsDownstreamUnattr, LegDownstream, 1); got.String() != "5000000" {
			t.Fatalf("excluded saturated mass did not remain commons: %s", got)
		}
		assertConserved(t, result)
	})

	t.Run("terminal tombstone preserves admitted descendant capacity", func(t *testing.T) {
		request := threeDescendantRequest()
		baseline, err := Allocate(request)
		if err != nil {
			t.Fatalf("baseline Allocate: %v", err)
		}
		for _, controller := range []string{"child-controller", "child-b-controller"} {
			assertAllocation(t, baseline, LegDownstream, 1, controller, "", "5000000")
		}

		request.DescendantImpacts[2].Disposition = ImpactDispositionTerminal
		result, err := Allocate(request)
		if err != nil {
			t.Fatalf("terminal Allocate: %v", err)
		}
		for _, controller := range []string{"child-controller", "child-b-controller"} {
			assertAllocation(t, result, LegDownstream, 1, controller, "", "5000000")
		}
		if got := allocationTotal(result, LegDownstream, 1, "child-c-controller", ""); got.Sign() != 0 {
			t.Fatalf("terminal descendant received claimant value: %s", got)
		}
		if got := commonsTotal(result, commonsDownstreamUnattr, LegDownstream, 1); got.String() != "5000000" {
			t.Fatalf("terminal descendant capacity was not reserved: %s", got)
		}
		if len(result.NewReceiptUses) != 3 {
			t.Fatalf("terminal admitted receipt was not consumed: %+v", result.NewReceiptUses)
		}
		assertConserved(t, result)
	})

	t.Run("missing roles cannot enrich a saturated cohort", func(t *testing.T) {
		request := threeDescendantRequest()
		request.Nodes[3].Credits[0].WeightPPM = 500_000
		request.Nodes[4].Credits[0].WeightPPM = 500_000
		request.Nodes[5].Credits[0].WeightPPM = 500_000
		result, err := Allocate(request)
		if err != nil {
			t.Fatalf("Allocate: %v", err)
		}
		for _, controller := range []string{"child-controller", "child-b-controller", "child-c-controller"} {
			assertAllocation(t, result, LegDownstream, 1, controller, "", "2500000")
		}
		if got := commonsTotal(result, commonsDownstreamUnattr, LegDownstream, 1); got.String() != "7500000" {
			t.Fatalf("missing role mass did not remain commons: %s", got)
		}
		assertConserved(t, result)
	})

	t.Run("sibling receipts divide one descendant impact", func(t *testing.T) {
		request := referenceRequest()
		request.DescendantImpacts = []Impact{
			impact("child", MilestoneE5, "a", "impact:e5:a", 1_000_000),
			impact("child", MilestoneE5, "b", "impact:e5:b", 1_000_000),
		}
		result, err := Allocate(request)
		if err != nil {
			t.Fatalf("Allocate: %v", err)
		}
		assertAllocation(t, result, LegDownstream, 1, "child-controller", receipt("a"), "7500000")
		assertAllocation(t, result, LegDownstream, 1, "child-controller", receipt("b"), "7500000")
		if got := allocationTotal(result, LegDownstream, 1, "child-controller", ""); got.String() != "15000000" {
			t.Fatalf("multiple receipts cloned descendant mass: %s", got)
		}
		if len(result.NewReceiptUses) != 2 {
			t.Fatalf("expected two exclusive uses: %+v", result.NewReceiptUses)
		}
		assertConserved(t, result)
	})

	t.Run("impact normalization loss remains terminal under saturation", func(t *testing.T) {
		request := referenceRequest()
		request.Nodes = append(request.Nodes,
			node("child-b", NodeModePayAndPropagate, credit("child-b-controller", "impact", 1_000_000)),
		)
		request.Edges = append(request.Edges,
			Edge{ChildClusterID: "child-b", ParentClusterID: "root", RawDependencyPPM: 1_000_000},
		)
		request.DescendantImpacts = []Impact{
			impact("child", MilestoneE5, "a", "impact:e5:a", 1_000_000),
			impact("child", MilestoneE5, "b", "impact:e5:b", 1_000_000),
			impact("child", MilestoneE5, "c", "impact:e5:c", 1_000_000),
			impact("child-b", MilestoneE5, "d", "impact:e5:d", 1_000_000),
		}
		result, err := Allocate(request)
		if err != nil {
			t.Fatalf("Allocate: %v", err)
		}
		assertAllocation(t, result, LegDownstream, 1, "child-controller", "", "7499992")
		assertAllocation(t, result, LegDownstream, 1, "child-b-controller", "", "7500000")
		if got := commonsTotal(result, commonsDownstreamUnattr, LegDownstream, 1); got.String() != "8" {
			t.Fatalf("impact normalization loss did not remain commons: %s", got)
		}
		assertConserved(t, result)
	})

	t.Run("impact and role use one conservative product", func(t *testing.T) {
		request := referenceRequest()
		request.Nodes[3].Credits = []Credit{credit("child-controller", "impact", 750_000)}
		request.Edges[2].RawDependencyPPM = 2
		request.DescendantImpacts[0].ImpactPPM = 950_000
		result, err := Allocate(request)
		if err != nil {
			t.Fatalf("Allocate: %v", err)
		}
		// floor(2 * 950,000 * 750,000 / 1,000,000^2) = 1 PPM
		// claimant weight, so the 15,000,000 depth-one bucket projects 15.
		assertAllocation(t, result, LegDownstream, 1, "child-controller", receipt("a"), "15")
		assertConserved(t, result)
	})
}

func TestReceiptConsumeOnEvaluationEvenWhenProjectedValueIsZero(t *testing.T) {
	tests := []struct {
		name string
		edit func(*Request)
	}{
		{
			name: "visible controller overlap",
			edit: func(request *Request) {
				request.Nodes[3].Credits = []Credit{credit("root-controller", "same-control", 1_000_000)}
			},
		},
		{
			name: "program cap",
			edit: func(request *Request) {
				request.Policy.ProgramWindowCapUzrn = "0"
				request.Policy = rehash(request.Policy)
			},
		},
		{
			name: "minimum payout dust",
			edit: func(request *Request) {
				request.Policy.MinProjectedPayoutUzrn = "100000001"
				request.Policy = rehash(request.Policy)
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := referenceRequest()
			test.edit(&request)
			result, err := Allocate(request)
			if err != nil {
				t.Fatalf("Allocate: %v", err)
			}
			if got := allocationTotal(result, LegDownstream, 0, "", receipt("a")); got.Sign() != 0 {
				t.Fatalf("zero-value receipt unexpectedly paid: %s", got)
			}
			if len(result.NewReceiptUses) != 1 || result.NewReceiptUses[0].ReceiptKey != receipt("a") {
				t.Fatalf("zero-value evaluated receipt was not consumed: %+v", result.NewReceiptUses)
			}
			assertConserved(t, result)
		})
	}
}

func TestControllerCapsAndMinimumPayout(t *testing.T) {
	t.Run("envelope cap overflow does not enrich competitors", func(t *testing.T) {
		request := referenceRequest()
		request.Policy.EnvelopeControllerCapPPM = 500_000
		request.Policy = rehash(request.Policy)
		result, err := Allocate(request)
		if err != nil {
			t.Fatalf("Allocate: %v", err)
		}
		if got := controllerTotal(result, "root-controller"); got.String() != "50000000" {
			t.Fatalf("envelope controller cap failed: %s", got)
		}
		if got := commonsTotal(result, commonsControllerCap, LegDirect, 0); got.String() != "10000000" {
			t.Fatalf("cap overflow was not terminal commons: %s", got)
		}
		assertConserved(t, result)
	})

	t.Run("program cap uses prior exposure", func(t *testing.T) {
		request := referenceRequest()
		request.Policy.ProgramWindowCapUzrn = "30000000"
		request.Policy = rehash(request.Policy)
		request.PriorControllerPaid = []ControllerAmount{{ControllerID: "root-controller", AmountUzrn: "25000000"}}
		result, err := Allocate(request)
		if err != nil {
			t.Fatalf("Allocate: %v", err)
		}
		if got := controllerTotal(result, "root-controller"); got.String() != "5000000" {
			t.Fatalf("program remaining cap failed: %s", got)
		}
		assertConserved(t, result)
	})

	t.Run("minimum applies after controller aggregation", func(t *testing.T) {
		request := referenceRequest()
		request.Policy.MinProjectedPayoutUzrn = "20000000"
		request.Policy = rehash(request.Policy)
		result, err := Allocate(request)
		if err != nil {
			t.Fatalf("Allocate: %v", err)
		}
		if result.ProjectedPaidUzrn != "60000000" {
			t.Fatalf("unexpected post-minimum paid total: %s", result.ProjectedPaidUzrn)
		}
		if got := controllerTotal(result, "root-controller"); got.String() != "60000000" {
			t.Fatalf("eligible controller changed: %s", got)
		}
		assertConserved(t, result)
	})
}

func TestControllerAggregationPreventsRoleSplittingGain(t *testing.T) {
	policy := DefaultPolicy()
	policy.DirectPPM = 1_000_000
	policy.UpstreamPPM = 0
	policy.DownstreamPPM = 0
	policy = rehash(policy)
	request := Request{
		Schema:                 Schema,
		EnvelopeID:             "envelope:e3:split-test",
		FundedClusterID:        "root",
		FundedMilestone:        MilestoneE3,
		EnvelopeUzrn:           "2",
		DescendantWindowClosed: true,
		Policy:                 policy,
		Nodes: []Node{
			node("root", NodeModePayAndPropagate,
				credit("controller-a", "one-role", 510_000),
				credit("controller-b", "one-role", 250_000),
			),
		},
	}
	unsplit, err := Allocate(request)
	if err != nil {
		t.Fatalf("unsplit: %v", err)
	}
	request.Nodes[0].Credits = []Credit{
		credit("controller-a", "role-a", 250_000),
		credit("controller-a", "role-b", 260_000),
		credit("controller-b", "one-role", 250_000),
	}
	split, err := Allocate(request)
	if err != nil {
		t.Fatalf("split: %v", err)
	}
	if got, want := controllerTotal(split, "controller-a"), controllerTotal(unsplit, "controller-a"); got.Cmp(want) != 0 || got.String() != "1" {
		t.Fatalf("role splitting changed controller allocation: split=%s unsplit=%s", got, want)
	}
	assertConserved(t, unsplit)
	assertConserved(t, split)
}

func TestReceiptReplayAndMilestoneBindingFailClosed(t *testing.T) {
	t.Run("prior receipt key represents child own economic use", func(t *testing.T) {
		request := referenceRequest()
		request.PriorReceiptUses = []ReceiptUse{{ReceiptKey: receipt("a"), EconomicSlotID: "child:e5:own"}}
		_, err := Allocate(request)
		if !errors.Is(err, ErrReceiptAlreadyConsumed) || validationCode(err) != CodeReceiptAlreadyConsumed {
			t.Fatalf("reused child receipt did not fail closed: %v", err)
		}
	})

	t.Run("prior economic slot cannot reset with a new key", func(t *testing.T) {
		request := referenceRequest()
		request.PriorReceiptUses = []ReceiptUse{{ReceiptKey: receipt("b"), EconomicSlotID: "impact:e5:a"}}
		_, err := Allocate(request)
		if !errors.Is(err, ErrReceiptAlreadyConsumed) {
			t.Fatalf("reused economic slot did not fail closed: %v", err)
		}
	})

	t.Run("impact milestone must match envelope", func(t *testing.T) {
		request := referenceRequest()
		request.DescendantImpacts[0].Milestone = MilestoneE6
		_, err := Allocate(request)
		if validationCode(err) != CodeInvalidMilestone {
			t.Fatalf("mixed E5/E6 envelope accepted: %v", err)
		}
	})

	t.Run("non-consequence milestone has no downstream compartment", func(t *testing.T) {
		request := referenceRequest()
		request.FundedMilestone = MilestoneE3
		request.DescendantImpacts = nil
		_, err := Allocate(request)
		if validationCode(err) != CodeInvalidPolicySum {
			t.Fatalf("E3 downstream compartment accepted: %v", err)
		}
	})

	t.Run("non-consequence milestone rejects impacts even with zero downstream", func(t *testing.T) {
		request := referenceRequest()
		request.FundedMilestone = MilestoneE3
		request.Policy.DirectPPM += request.Policy.DownstreamPPM
		request.Policy.DownstreamPPM = 0
		request.Policy = rehash(request.Policy)
		_, err := Allocate(request)
		if validationCode(err) != CodeInvalidMilestone {
			t.Fatalf("E3 descendant impact accepted: %v", err)
		}
	})

	t.Run("impact must reach the funded cluster", func(t *testing.T) {
		request := referenceRequest()
		request.Edges = request.Edges[:2]
		_, err := Allocate(request)
		if validationCode(err) != CodeImpactNotLinked {
			t.Fatalf("unlinked descendant impact accepted: %v", err)
		}
	})
}

func TestValidationAndLimits(t *testing.T) {
	tests := []struct {
		name string
		edit func(*Request)
		code string
		kind error
	}{
		{
			name: "non canonical amount",
			edit: func(request *Request) { request.EnvelopeUzrn = "01" },
			code: CodeInvalidAmount,
			kind: ErrInvalidInput,
		},
		{
			name: "amount exceeds 256 bits",
			edit: func(request *Request) {
				request.EnvelopeUzrn = new(big.Int).Lsh(big.NewInt(1), MaxAmountBits).String()
			},
			code: CodeInvalidAmount,
			kind: ErrLimitExceeded,
		},
		{
			name: "amount text is bounded before parsing",
			edit: func(request *Request) {
				request.EnvelopeUzrn = strings.Repeat("9", MaxAmountDecimalDigits+1)
			},
			code: CodeInvalidAmount,
			kind: ErrLimitExceeded,
		},
		{
			name: "digest does not commit to policy",
			edit: func(request *Request) { request.Policy.DomainRevision++ },
			code: CodeInvalidPolicyDigest,
			kind: ErrInvalidInput,
		},
		{
			name: "non pay node declares credits",
			edit: func(request *Request) { request.Nodes[1].Mode = NodeModePassThrough },
			code: CodeInvalidCreditSum,
			kind: ErrInvalidInput,
		},
		{
			name: "funded node must pay and propagate",
			edit: func(request *Request) {
				request.Nodes[0].Mode = NodeModePassThrough
				request.Nodes[0].Credits = nil
			},
			code: CodeInvalidNodeMode,
			kind: ErrInvalidInput,
		},
		{
			name: "descendant id syntax checked before lookup",
			edit: func(request *Request) { request.DescendantImpacts[0].DescendantClusterID = "BAD ID" },
			code: CodeInvalidID,
			kind: ErrInvalidInput,
		},
		{
			name: "impact disposition must be explicit",
			edit: func(request *Request) { request.DescendantImpacts[0].Disposition = "" },
			code: CodeInvalidImpactDisposition,
			kind: ErrInvalidInput,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := referenceRequest()
			test.edit(&request)
			_, err := Allocate(request)
			if !errors.Is(err, test.kind) || validationCode(err) != test.code {
				t.Fatalf("got %v (code %q), want kind %v code %q", err, validationCode(err), test.kind, test.code)
			}
		})
	}

	request := referenceRequest()
	request.PriorControllerPaid = make([]ControllerAmount, MaxControllers+1)
	for index := range request.PriorControllerPaid {
		request.PriorControllerPaid[index] = ControllerAmount{
			ControllerID: "prior-" + leftPad(index, 3), AmountUzrn: "0",
		}
	}
	_, err := Allocate(request)
	if !errors.Is(err, ErrLimitExceeded) {
		t.Fatalf("unbounded prior paid slice accepted: %v", err)
	}
}

func TestPermutationAndConcurrentReplayAreByteIdentical(t *testing.T) {
	request := referenceRequest()
	request.Nodes[0].Credits = []Credit{
		credit("root-controller", "origin", 700_000),
		credit("second-root-controller", "review", 300_000),
	}
	request.DescendantImpacts = append(request.DescendantImpacts,
		impact("child", MilestoneE5, "b", "impact:e5:b", 250_000),
	)
	want, err := Allocate(request)
	if err != nil {
		t.Fatalf("Allocate baseline: %v", err)
	}
	wantJSON := mustJSON(t, want)

	permuted := cloneRequest(t, request)
	slices.Reverse(permuted.Nodes)
	slices.Reverse(permuted.Edges)
	slices.Reverse(permuted.DescendantImpacts)
	for index := range permuted.Nodes {
		slices.Reverse(permuted.Nodes[index].Credits)
	}
	got, err := Allocate(permuted)
	if err != nil {
		t.Fatalf("Allocate permuted: %v", err)
	}
	if gotJSON := mustJSON(t, got); string(gotJSON) != string(wantJSON) {
		t.Fatalf("permutation changed output:\nwant %s\n got %s", wantJSON, gotJSON)
	}

	const workers = 16
	errorsFromWorkers := make(chan error, workers)
	var wait sync.WaitGroup
	for worker := 0; worker < workers; worker++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			result, allocateErr := Allocate(request)
			if allocateErr != nil {
				errorsFromWorkers <- allocateErr
				return
			}
			encoded, marshalErr := json.Marshal(result)
			if marshalErr != nil {
				errorsFromWorkers <- marshalErr
				return
			}
			if string(encoded) != string(wantJSON) {
				errorsFromWorkers <- errors.New("concurrent replay changed output")
			}
		}()
	}
	wait.Wait()
	close(errorsFromWorkers)
	for workerErr := range errorsFromWorkers {
		t.Fatal(workerErr)
	}
}

func TestMaximumAmountAndHamiltonCommonsTie(t *testing.T) {
	request := referenceRequest()
	request.EnvelopeUzrn = new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), MaxAmountBits), big.NewInt(1)).String()
	result, err := Allocate(request)
	if err != nil {
		t.Fatalf("maximum amount: %v", err)
	}
	assertConserved(t, result)

	values, err := apportion(big.NewInt(1), big.NewInt(2), []weightedKey{
		{key: "claimant", weight: big.NewInt(1)},
		{key: "commons", weight: big.NewInt(1), isCommons: true},
	})
	if err != nil {
		t.Fatalf("tie apportion: %v", err)
	}
	if findApportioned(values, "commons").Cmp(big.NewInt(1)) != 0 {
		t.Fatalf("exact tie did not prefer commons: %+v", values)
	}
}

func TestCrossControllerRoundingResidualCannotEnrichCompetitors(t *testing.T) {
	line := func(controller string) weightedAllocation {
		return weightedAllocation{
			line: rawAllocation{
				leg: LegDownstream, depth: 1, cluster: "child-" + controller,
				controller: controller, role: "impact",
			},
			weight: ppmInt(200_000),
		}
	}
	baselineWeighted := []weightedAllocation{
		line("controller-a"),
		line("controller-b"),
		{
			line: rawAllocation{
				leg: LegDownstream, depth: 1, cluster: "child-x",
				controller: "controller-x", role: "impact",
			},
			weight: ppmInt(400_000),
		},
	}
	baseline, baselineCommons, err := apportionAllocationWeights(
		big.NewInt(2), ppmInt(PPM), baselineWeighted, ppmInt(200_000),
	)
	if err != nil {
		t.Fatalf("baseline apportion: %v", err)
	}

	// Invalidating controller-x turns its weight into terminal mass. Neither
	// remaining controller may capture a unit that previously belonged to the
	// invalid claimant through a cross-controller Hamilton remainder.
	withoutX, withoutXCommons, err := apportionAllocationWeights(
		big.NewInt(2), ppmInt(PPM), baselineWeighted[:2], ppmInt(600_000),
	)
	if err != nil {
		t.Fatalf("removed claimant apportion: %v", err)
	}
	for _, controller := range []string{"controller-a", "controller-b"} {
		if before, after := rawControllerTotal(baseline, controller), rawControllerTotal(withoutX, controller); before.Cmp(after) != 0 {
			t.Fatalf("removing controller-x enriched %s: before=%s after=%s", controller, before, after)
		}
	}
	if baselineCommons.String() != "2" || withoutXCommons.String() != "2" {
		t.Fatalf("rounding residual escaped terminal route: before=%s after=%s", baselineCommons, withoutXCommons)
	}
}

func TestCanonicalLegRankBreaksExactTopLevelTie(t *testing.T) {
	request := referenceRequest()
	request.EnvelopeUzrn = "1"
	request.Policy.DirectPPM = 0
	request.Policy.UpstreamPPM = 500_000
	request.Policy.DownstreamPPM = 500_000
	request.Policy.UpstreamContinuationPPM = 0
	request.Policy.UpstreamMaxDepth = 1
	request.Policy = rehash(request.Policy)
	result, err := Allocate(request)
	if err != nil {
		t.Fatalf("Allocate: %v", err)
	}
	if result.ProjectedPaidUzrn != "1" {
		t.Fatalf("fixed upstream-before-downstream tie order not applied: %+v", result)
	}
	assertAllocation(t, result, LegUpstream, 1, "parent-controller", "", "1")
	assertConserved(t, result)
}

func TestModeratelyLargeCohortConservesWithinLineBound(t *testing.T) {
	const (
		descendants          = 512
		creditsPerDescendant = 8
	)
	request := Request{
		Schema:                 Schema,
		EnvelopeID:             "envelope:e5:large-cohort",
		FundedClusterID:        "root",
		FundedMilestone:        MilestoneE5,
		EnvelopeUzrn:           "100000000",
		DescendantWindowClosed: true,
		Policy:                 DefaultPolicy(),
		Nodes: []Node{
			node("root", NodeModePayAndPropagate, credit("root-controller", "origin", 1_000_000)),
		},
	}
	for descendant := 0; descendant < descendants; descendant++ {
		clusterID := fmt.Sprintf("child-%03d", descendant)
		credits := make([]Credit, 0, creditsPerDescendant)
		for controller := 0; controller < creditsPerDescendant; controller++ {
			credits = append(credits, credit(
				fmt.Sprintf("independent-%02d", controller),
				"impact",
				uint32(PPM/creditsPerDescendant),
			))
		}
		request.Nodes = append(request.Nodes, node(clusterID, NodeModePayAndPropagate, credits...))
		request.Edges = append(request.Edges, Edge{
			ChildClusterID: clusterID, ParentClusterID: "root", RawDependencyPPM: 1_000_000,
		})
		request.DescendantImpacts = append(request.DescendantImpacts, Impact{
			DescendantClusterID: clusterID,
			Milestone:           MilestoneE5,
			ReceiptKey:          fmt.Sprintf("sha256:%064x", descendant+1),
			EconomicSlotID:      fmt.Sprintf("impact:e5:%03d", descendant),
			Disposition:         ImpactDispositionPayable,
			ImpactPPM:           1_000_000,
		})
	}
	result, err := Allocate(request)
	if err != nil {
		t.Fatalf("large cohort: %v", err)
	}
	if len(result.Allocations) != descendants*creditsPerDescendant+1 {
		t.Fatalf("allocation lines=%d, want %d", len(result.Allocations), descendants*creditsPerDescendant+1)
	}
	if len(result.NewReceiptUses) != descendants {
		t.Fatalf("receipt uses=%d, want %d", len(result.NewReceiptUses), descendants)
	}
	assertConserved(t, result)
}

func referenceRequest() Request {
	return Request{
		Schema:                 Schema,
		EnvelopeID:             "envelope:e5:reference",
		FundedClusterID:        "root",
		FundedMilestone:        MilestoneE5,
		EnvelopeUzrn:           "100000000",
		DescendantWindowClosed: true,
		Policy:                 DefaultPolicy(),
		Nodes: []Node{
			node("root", NodeModePayAndPropagate, credit("root-controller", "origin", 1_000_000)),
			node("parent", NodeModePayAndPropagate, credit("parent-controller", "dependency", 1_000_000)),
			node("grandparent", NodeModePayAndPropagate, credit("grand-controller", "dependency", 1_000_000)),
			node("child", NodeModePayAndPropagate, credit("child-controller", "impact", 1_000_000)),
		},
		Edges: []Edge{
			{ChildClusterID: "root", ParentClusterID: "parent", RawDependencyPPM: 1_000_000},
			{ChildClusterID: "parent", ParentClusterID: "grandparent", RawDependencyPPM: 1_000_000},
			{ChildClusterID: "child", ParentClusterID: "root", RawDependencyPPM: 1_000_000},
		},
		DescendantImpacts: []Impact{
			impact("child", MilestoneE5, "a", "impact:e5:a", 1_000_000),
		},
	}
}

func threeDescendantRequest() Request {
	request := referenceRequest()
	request.Nodes = append(request.Nodes,
		node("child-b", NodeModePayAndPropagate, credit("child-b-controller", "impact", 1_000_000)),
		node("child-c", NodeModePayAndPropagate, credit("child-c-controller", "impact", 1_000_000)),
	)
	request.Edges = append(request.Edges,
		Edge{ChildClusterID: "child-b", ParentClusterID: "root", RawDependencyPPM: 1_000_000},
		Edge{ChildClusterID: "child-c", ParentClusterID: "root", RawDependencyPPM: 1_000_000},
	)
	request.DescendantImpacts = append(request.DescendantImpacts,
		impact("child-b", MilestoneE5, "b", "impact:e5:b", 1_000_000),
		impact("child-c", MilestoneE5, "c", "impact:e5:c", 1_000_000),
	)
	return request
}

func node(id string, mode NodeMode, credits ...Credit) Node {
	return Node{ClusterID: id, Mode: mode, Credits: credits}
}

func credit(controller, role string, weight uint32) Credit {
	return Credit{ControllerID: controller, RoleID: role, WeightPPM: weight}
}

func impact(descendant, milestone, receiptByte, slot string, weight uint32) Impact {
	return Impact{
		DescendantClusterID: descendant,
		Milestone:           milestone,
		ReceiptKey:          receipt(receiptByte),
		EconomicSlotID:      slot,
		Disposition:         ImpactDispositionPayable,
		ImpactPPM:           weight,
	}
}

func receipt(hexCharacter string) string {
	return "sha256:" + strings.Repeat(hexCharacter, 64)
}

func rehash(policy Policy) Policy {
	policy.PolicyDigest = ComputePolicyDigest(policy)
	return policy
}

func assertBucketAmounts(t *testing.T, buckets []DepthBucket, expected []string) {
	t.Helper()
	if len(buckets) != len(expected) {
		t.Fatalf("bucket count=%d, want %d: %+v", len(buckets), len(expected), buckets)
	}
	for index, want := range expected {
		if buckets[index].ProjectedUzrn != want {
			t.Fatalf("bucket[%d]=%s, want %s", index, buckets[index].ProjectedUzrn, want)
		}
	}
}

func assertAllocation(t *testing.T, result Result, leg Leg, depth uint32, controller, receiptKey, expected string) {
	t.Helper()
	if got := allocationTotal(result, leg, depth, controller, receiptKey); got.String() != expected {
		t.Fatalf("allocation %s depth=%d controller=%q receipt=%q: got %s want %s\nall=%+v", leg, depth, controller, receiptKey, got, expected, result.Allocations)
	}
}

func allocationTotal(result Result, leg Leg, depth uint32, controller, receiptKey string) *big.Int {
	total := new(big.Int)
	for _, allocation := range result.Allocations {
		if allocation.Leg != leg {
			continue
		}
		if depth != 0 && allocation.Depth != depth {
			continue
		}
		if controller != "" && allocation.ControllerID != controller {
			continue
		}
		if receiptKey != "" && allocation.ReceiptKey != receiptKey {
			continue
		}
		amount, _ := new(big.Int).SetString(allocation.ProjectedUzrn, 10)
		total.Add(total, amount)
	}
	return total
}

func controllerTotal(result Result, controller string) *big.Int {
	total := new(big.Int)
	for _, allocation := range result.Allocations {
		if allocation.ControllerID != controller {
			continue
		}
		amount, _ := new(big.Int).SetString(allocation.ProjectedUzrn, 10)
		total.Add(total, amount)
	}
	return total
}

func rawControllerTotal(lines []rawAllocation, controller string) *big.Int {
	total := new(big.Int)
	for _, line := range lines {
		if line.controller == controller {
			total.Add(total, line.amount)
		}
	}
	return total
}

func commonsTotal(result Result, reason string, leg Leg, depth uint32) *big.Int {
	total := new(big.Int)
	for _, commons := range result.Commons {
		if commons.Reason != reason || commons.SourceLeg != leg {
			continue
		}
		if depth != 0 && commons.Depth != depth {
			continue
		}
		amount, _ := new(big.Int).SetString(commons.ProjectedUzrn, 10)
		total.Add(total, amount)
	}
	return total
}

func assertConserved(t *testing.T, result Result) {
	t.Helper()
	envelope, ok := new(big.Int).SetString(result.EnvelopeUzrn, 10)
	if !ok {
		t.Fatalf("bad envelope: %q", result.EnvelopeUzrn)
	}
	paid, ok := new(big.Int).SetString(result.ProjectedPaidUzrn, 10)
	if !ok {
		t.Fatalf("bad paid amount: %q", result.ProjectedPaidUzrn)
	}
	commons, ok := new(big.Int).SetString(result.ProjectedCommonsUzrn, 10)
	if !ok {
		t.Fatalf("bad commons amount: %q", result.ProjectedCommonsUzrn)
	}
	if new(big.Int).Add(paid, commons).Cmp(envelope) != 0 || result.ConservationCheck != ConservationOK {
		t.Fatalf("not conserved: envelope=%s paid=%s commons=%s check=%s", envelope, paid, commons, result.ConservationCheck)
	}
}

func validationCode(err error) string {
	var validation *ValidationError
	if errors.As(err, &validation) {
		return validation.Code
	}
	return ""
}

func cloneRequest(t *testing.T, input Request) Request {
	t.Helper()
	data := mustJSON(t, input)
	var output Request
	if err := json.Unmarshal(data, &output); err != nil {
		t.Fatalf("clone request: %v", err)
	}
	return output
}

func mustJSON(t *testing.T, value any) []byte {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal JSON: %v", err)
	}
	return data
}

func leftPad(value, width int) string {
	text := new(big.Int).SetInt64(int64(value)).String()
	return strings.Repeat("0", width-len(text)) + text
}
