package branchflow

import (
	"errors"
	"fmt"
	"math/big"
	"testing"
)

func TestBoundsNodes(t *testing.T) {
	t.Run("accepts maximum", func(t *testing.T) {
		request := boundsNodeRequest(MaxNodes)
		result, err := Allocate(request)
		if err != nil {
			t.Fatalf("Allocate at MaxNodes: %v", err)
		}
		boundsAssertConserved(t, result)
	})

	t.Run("rejects maximum plus one", func(t *testing.T) {
		_, err := Allocate(boundsNodeRequest(MaxNodes + 1))
		boundsRequireError(t, err, ErrLimitExceeded, CodeLimitExceeded)
	})
}

func TestBoundsEdges(t *testing.T) {
	t.Run("accepts maximum", func(t *testing.T) {
		request := boundsEdgeRequest(MaxEdges)
		result, err := Allocate(request)
		if err != nil {
			t.Fatalf("Allocate at MaxEdges: %v", err)
		}
		if len(result.NormalizedEdges) != MaxEdges {
			t.Fatalf("normalized edges=%d, want %d", len(result.NormalizedEdges), MaxEdges)
		}
		boundsAssertConserved(t, result)
	})

	t.Run("rejects maximum plus one", func(t *testing.T) {
		_, err := Allocate(boundsEdgeRequest(MaxEdges + 1))
		boundsRequireError(t, err, ErrLimitExceeded, CodeLimitExceeded)
	})
}

func TestBoundsDescendantImpacts(t *testing.T) {
	t.Run("accepts maximum", func(t *testing.T) {
		request := boundsImpactRequest(MaxDescendantImpacts)
		result, err := Allocate(request)
		if err != nil {
			t.Fatalf("Allocate at MaxDescendantImpacts: %v", err)
		}
		if len(result.NewReceiptUses) != MaxDescendantImpacts {
			t.Fatalf("new receipt uses=%d, want %d", len(result.NewReceiptUses), MaxDescendantImpacts)
		}
		boundsAssertConserved(t, result)
	})

	t.Run("rejects maximum plus one", func(t *testing.T) {
		_, err := Allocate(boundsImpactRequest(MaxDescendantImpacts + 1))
		boundsRequireError(t, err, ErrLimitExceeded, CodeLimitExceeded)
	})
}

func TestBoundsDepth(t *testing.T) {
	t.Run("accepts maximum in both directions", func(t *testing.T) {
		request := boundsBaseRequest("depth-max")
		request.Policy.UpstreamMaxDepth = MaxDepth
		request.Policy.DownstreamMaxDepth = MaxDepth
		request.Policy.PolicyDigest = ComputePolicyDigest(request.Policy)
		result, err := Allocate(request)
		if err != nil {
			t.Fatalf("Allocate at MaxDepth: %v", err)
		}
		if len(result.UpstreamBuckets) != MaxDepth+1 || len(result.DownstreamBuckets) != MaxDepth+1 {
			t.Fatalf(
				"bucket lengths upstream=%d downstream=%d, want %d each",
				len(result.UpstreamBuckets),
				len(result.DownstreamBuckets),
				MaxDepth+1,
			)
		}
		boundsAssertConserved(t, result)
	})

	for _, direction := range []string{"upstream", "downstream"} {
		t.Run("rejects "+direction+" maximum plus one", func(t *testing.T) {
			request := boundsBaseRequest("depth-over-" + direction)
			if direction == "upstream" {
				request.Policy.UpstreamMaxDepth = MaxDepth + 1
			} else {
				request.Policy.DownstreamMaxDepth = MaxDepth + 1
			}
			request.Policy.PolicyDigest = ComputePolicyDigest(request.Policy)
			_, err := Allocate(request)
			boundsRequireError(t, err, ErrInvalidInput, CodeInvalidDepth)
		})
	}
}

func TestBoundsTraversalOperations(t *testing.T) {
	counter := operationCounter{used: MaxTraversalOperations - 1}
	if err := counter.add(1); err != nil {
		t.Fatalf("last permitted traversal operation failed: %v", err)
	}
	if counter.used != MaxTraversalOperations {
		t.Fatalf("operation count=%d, want %d", counter.used, MaxTraversalOperations)
	}
	err := counter.add(1)
	boundsRequireError(t, err, ErrLimitExceeded, CodeTraversalLimit)
	if counter.used != MaxTraversalOperations {
		t.Fatalf("refused operation changed count to %d", counter.used)
	}
}

func boundsBaseRequest(suffix string) Request {
	return Request{
		Schema:                 Schema,
		EnvelopeID:             "bounds:" + suffix,
		FundedClusterID:        "root",
		FundedMilestone:        MilestoneE5,
		EnvelopeUzrn:           "100000000",
		DescendantWindowClosed: true,
		Policy:                 DefaultPolicy(),
		Nodes: []Node{{
			ClusterID: "root",
			Mode:      NodeModePayAndPropagate,
			Credits: []Credit{{
				ControllerID: "root-controller",
				RoleID:       "origin",
				WeightPPM:    1_000_000,
			}},
		}},
	}
}

func boundsNodeRequest(count int) Request {
	request := boundsBaseRequest("nodes")
	request.FundedClusterID = "node-0000"
	request.Nodes = make([]Node, count)
	for index := range request.Nodes {
		request.Nodes[index] = Node{
			ClusterID: fmt.Sprintf("node-%04d", index),
			Mode:      NodeModePayAndPropagate,
		}
	}
	request.Nodes[0].Credits = []Credit{{
		ControllerID: "root-controller",
		RoleID:       "origin",
		WeightPPM:    1_000_000,
	}}
	return request
}

func boundsEdgeRequest(count int) Request {
	const (
		parentCount = MaxParentsPerNode
		childCount  = MaxEdges/MaxParentsPerNode + 1
	)
	request := boundsBaseRequest("edges")
	request.FundedClusterID = "child-000"
	request.Nodes = make([]Node, 0, parentCount+childCount)
	for parent := 0; parent < parentCount; parent++ {
		request.Nodes = append(request.Nodes, Node{
			ClusterID: fmt.Sprintf("parent-%02d", parent),
			Mode:      NodeModePayAndPropagate,
		})
	}
	for child := 0; child < childCount; child++ {
		node := Node{
			ClusterID: fmt.Sprintf("child-%03d", child),
			Mode:      NodeModePayAndPropagate,
		}
		if child == 0 {
			node.Credits = []Credit{{
				ControllerID: "root-controller",
				RoleID:       "origin",
				WeightPPM:    1_000_000,
			}}
		}
		request.Nodes = append(request.Nodes, node)
	}
	request.Edges = make([]Edge, 0, count)
	for child := 0; child < childCount && len(request.Edges) < count; child++ {
		for parent := 0; parent < parentCount && len(request.Edges) < count; parent++ {
			request.Edges = append(request.Edges, Edge{
				ChildClusterID:   fmt.Sprintf("child-%03d", child),
				ParentClusterID:  fmt.Sprintf("parent-%02d", parent),
				RawDependencyPPM: uint32(PPM / parentCount),
			})
		}
	}
	return request
}

func boundsImpactRequest(count int) Request {
	request := boundsBaseRequest("impacts")
	request.Nodes = append(request.Nodes, Node{
		ClusterID: "child",
		Mode:      NodeModePayAndPropagate,
		Credits: []Credit{{
			ControllerID: "child-controller",
			RoleID:       "impact",
			WeightPPM:    1_000_000,
		}},
	})
	request.Edges = []Edge{{
		ChildClusterID:   "child",
		ParentClusterID:  "root",
		RawDependencyPPM: 1_000_000,
	}}
	request.DescendantImpacts = make([]Impact, count)
	for index := range request.DescendantImpacts {
		request.DescendantImpacts[index] = Impact{
			DescendantClusterID: "child",
			Milestone:           MilestoneE5,
			ReceiptKey:          fmt.Sprintf("sha256:%064x", index+1),
			EconomicSlotID:      fmt.Sprintf("impact:e5:%04d", index),
			Disposition:         ImpactDispositionPayable,
			ImpactPPM:           1,
		}
	}
	return request
}

func boundsRequireError(t *testing.T, err, kind error, code string) {
	t.Helper()
	if !errors.Is(err, kind) {
		t.Fatalf("error=%v, want errors.Is(..., %v)", err, kind)
	}
	var validation *ValidationError
	if !errors.As(err, &validation) {
		t.Fatalf("error=%v, want ValidationError", err)
	}
	if validation.Code != code {
		t.Fatalf("error code=%q, want %q: %v", validation.Code, code, err)
	}
}

func boundsAssertConserved(t *testing.T, result Result) {
	t.Helper()
	envelope, ok := new(big.Int).SetString(result.EnvelopeUzrn, 10)
	if !ok {
		t.Fatalf("invalid envelope output %q", result.EnvelopeUzrn)
	}
	paid, ok := new(big.Int).SetString(result.ProjectedPaidUzrn, 10)
	if !ok {
		t.Fatalf("invalid paid output %q", result.ProjectedPaidUzrn)
	}
	commons, ok := new(big.Int).SetString(result.ProjectedCommonsUzrn, 10)
	if !ok {
		t.Fatalf("invalid commons output %q", result.ProjectedCommonsUzrn)
	}
	if paid.Add(paid, commons).Cmp(envelope) != 0 {
		t.Fatalf(
			"not conserved: envelope=%s paid+commons=%s",
			envelope,
			paid,
		)
	}
}
