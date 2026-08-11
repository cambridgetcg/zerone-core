package branchflow

import (
	"encoding/json"
	"slices"
	"strconv"
	"testing"
)

func FuzzAllocateConservation(f *testing.F) {
	f.Add(uint64(100_000_000), uint32(600_000), uint32(100_000), uint32(500_000), uint32(500_000), uint32(1_000_000))
	f.Add(uint64(1), uint32(1), uint32(999_999), uint32(0), uint32(1), uint32(1))
	f.Add(^uint64(0), uint32(999_999), uint32(999_999), uint32(999_999), uint32(999_999), uint32(999_999))
	f.Fuzz(func(t *testing.T, envelope uint64, directSeed, upstreamSeed, continuationSeed, dependencySeed, capSeed uint32) {
		direct := uint64(directSeed) % (PPM + 1)
		remaining := PPM - direct
		upstream := uint64(upstreamSeed) % (remaining + 1)
		downstream := remaining - upstream
		policy := DefaultPolicy()
		policy.DirectPPM = uint32(direct)
		policy.UpstreamPPM = uint32(upstream)
		policy.DownstreamPPM = uint32(downstream)
		policy.UpstreamContinuationPPM = uint32(uint64(continuationSeed) % PPM)
		policy.DownstreamContinuationPPM = uint32((uint64(continuationSeed) * 7919) % PPM)
		policy.UpstreamMaxDepth = 1 + continuationSeed%5
		policy.DownstreamMaxDepth = 1 + (continuationSeed/5)%5
		policy.EnvelopeControllerCapPPM = 1 + capSeed%uint32(PPM)
		policy = rehash(policy)

		request := referenceRequest()
		request.EnvelopeUzrn = strconv.FormatUint(envelope, 10)
		request.Policy = policy
		request.Edges[0].RawDependencyPPM = 1 + dependencySeed%uint32(PPM)
		request.Edges[1].RawDependencyPPM = 1 + (dependencySeed*31)%uint32(PPM)
		request.Edges[2].RawDependencyPPM = 1 + (dependencySeed*131)%uint32(PPM)
		request.DescendantImpacts[0].ImpactPPM = 1 + (dependencySeed*8191)%uint32(PPM)

		result, err := Allocate(request)
		if err != nil {
			t.Fatalf("valid generated request failed: %v", err)
		}
		assertConserved(t, result)
		if result.MovesFunds || result.IntegrationReady || result.EconomicEffect != EconomicEffect {
			t.Fatalf("effect fence changed: %+v", result)
		}
	})
}

func FuzzAllocatePermutation(f *testing.F) {
	f.Add(uint64(0))
	f.Add(uint64(0xff))
	f.Fuzz(func(t *testing.T, mask uint64) {
		request := referenceRequest()
		request.Nodes[0].Credits = []Credit{
			credit("root-controller", "origin", 700_000),
			credit("second-root-controller", "review", 300_000),
		}
		request.DescendantImpacts = append(request.DescendantImpacts,
			impact("child", MilestoneE5, "b", "impact:e5:b", 250_000),
		)
		baseline, err := Allocate(request)
		if err != nil {
			t.Fatalf("baseline: %v", err)
		}
		permuted := cloneRequest(t, request)
		if mask&1 != 0 {
			slices.Reverse(permuted.Nodes)
		}
		if mask&2 != 0 {
			slices.Reverse(permuted.Edges)
		}
		if mask&4 != 0 {
			slices.Reverse(permuted.DescendantImpacts)
		}
		if mask&8 != 0 {
			for index := range permuted.Nodes {
				slices.Reverse(permuted.Nodes[index].Credits)
			}
		}
		got, err := Allocate(permuted)
		if err != nil {
			t.Fatalf("permuted: %v", err)
		}
		baselineJSON, err := json.Marshal(baseline)
		if err != nil {
			t.Fatal(err)
		}
		gotJSON, err := json.Marshal(got)
		if err != nil {
			t.Fatal(err)
		}
		if string(gotJSON) != string(baselineJSON) {
			t.Fatalf("mask %#x changed output:\nbaseline %s\npermuted %s", mask, baselineJSON, gotJSON)
		}
	})
}
