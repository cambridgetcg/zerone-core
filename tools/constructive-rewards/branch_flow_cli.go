package main

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/zerone-chain/zerone/tools/constructive-rewards/branchflow"
)

func runBranchFlow(envelope, format string, stdout, stderr io.Writer) int {
	result, err := branchflow.Allocate(referenceBranchFlowRequest(envelope))
	if err != nil {
		fmt.Fprintf(stderr, "constructive-rewards branch-flow: %v\n", err)
		return 2
	}
	if format == "json" {
		encoder := json.NewEncoder(stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(result); err != nil {
			fmt.Fprintf(stderr, "constructive-rewards branch-flow: encode result: %v\n", err)
			return 2
		}
		return 0
	}

	fmt.Fprintln(stdout, "ZERONE CONSTRUCTIVE-INTELLIGENCE BRANCH FLOW")
	fmt.Fprintln(stdout, "closed-window reference projection; no chain state read or changed")
	fmt.Fprintf(
		stdout,
		"assurance=%s economic_effect=%s moves_funds=%t integration_ready=%t\n",
		result.Assurance,
		result.EconomicEffect,
		result.MovesFunds,
		result.IntegrationReady,
	)
	fmt.Fprintf(
		stdout,
		"envelope=%s funded_cluster=%s funded_milestone=%s envelope_uzrn=%s\n",
		result.EnvelopeID,
		result.FundedClusterID,
		result.FundedMilestone,
		result.EnvelopeUzrn,
	)
	fmt.Fprintf(stdout, "policy_digest=%s\n", result.Policy.PolicyDigest)
	for _, allocation := range result.Allocations {
		fmt.Fprintf(
			stdout,
			"allocation leg=%s depth=%d cluster=%s controller=%s role=%s projected_uzrn=%s",
			allocation.Leg,
			allocation.Depth,
			allocation.ClusterID,
			allocation.ControllerID,
			allocation.RoleID,
			allocation.ProjectedUzrn,
		)
		if allocation.Milestone != "" {
			fmt.Fprintf(stdout, " milestone=%s receipt=%s", allocation.Milestone, allocation.ReceiptKey)
		}
		fmt.Fprintln(stdout)
	}
	for _, use := range result.NewReceiptUses {
		fmt.Fprintf(
			stdout,
			"receipt_use key=%s economic_slot=%s disposition=CONSUMED_ON_SUCCESSFUL_EVALUATION\n",
			use.ReceiptKey,
			use.EconomicSlotID,
		)
	}
	for _, terminal := range result.Commons {
		fmt.Fprintf(
			stdout,
			"terminal reason=%s source_leg=%s depth=%d projected_uzrn=%s\n",
			terminal.Reason,
			terminal.SourceLeg,
			terminal.Depth,
			terminal.ProjectedUzrn,
		)
	}
	fmt.Fprintf(
		stdout,
		"projected_paid_uzrn=%s projected_commons_uzrn=%s conservation=%s\n",
		result.ProjectedPaidUzrn,
		result.ProjectedCommonsUzrn,
		result.ConservationCheck,
	)
	return 0
}

func referenceBranchFlowRequest(envelope string) branchflow.Request {
	return branchflow.Request{
		Schema:                 branchflow.Schema,
		EnvelopeID:             "envelope:e5:reference",
		FundedClusterID:        "root",
		FundedMilestone:        branchflow.MilestoneE5,
		EnvelopeUzrn:           envelope,
		DescendantWindowClosed: true,
		Policy:                 branchflow.DefaultPolicy(),
		Nodes: []branchflow.Node{
			{
				ClusterID: "root",
				Mode:      branchflow.NodeModePayAndPropagate,
				Credits: []branchflow.Credit{{
					ControllerID: "root-controller",
					RoleID:       "origin",
					WeightPPM:    1_000_000,
				}},
			},
			{
				ClusterID: "parent",
				Mode:      branchflow.NodeModePayAndPropagate,
				Credits: []branchflow.Credit{{
					ControllerID: "parent-controller",
					RoleID:       "dependency",
					WeightPPM:    1_000_000,
				}},
			},
			{
				ClusterID: "grandparent",
				Mode:      branchflow.NodeModePayAndPropagate,
				Credits: []branchflow.Credit{{
					ControllerID: "grand-controller",
					RoleID:       "dependency",
					WeightPPM:    1_000_000,
				}},
			},
			{
				ClusterID: "child",
				Mode:      branchflow.NodeModePayAndPropagate,
				Credits: []branchflow.Credit{{
					ControllerID: "child-controller",
					RoleID:       "impact",
					WeightPPM:    1_000_000,
				}},
			},
		},
		Edges: []branchflow.Edge{
			{ChildClusterID: "root", ParentClusterID: "parent", RawDependencyPPM: 1_000_000},
			{ChildClusterID: "parent", ParentClusterID: "grandparent", RawDependencyPPM: 1_000_000},
			{ChildClusterID: "child", ParentClusterID: "root", RawDependencyPPM: 1_000_000},
		},
		DescendantImpacts: []branchflow.Impact{{
			DescendantClusterID: "child",
			Milestone:           branchflow.MilestoneE5,
			ReceiptKey:          "sha256:" + strings.Repeat("a", 64),
			EconomicSlotID:      "impact:e5:a",
			Disposition:         branchflow.ImpactDispositionPayable,
			ImpactPPM:           1_000_000,
		}},
	}
}
