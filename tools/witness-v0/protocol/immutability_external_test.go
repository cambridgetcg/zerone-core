package protocol_test

import (
	"testing"

	"github.com/zerone-chain/zerone/tools/witness-v0/protocol"
)

func TestNormativeTablesAreDefensiveCopies(t *testing.T) {
	nonclaims := protocol.RequiredNonclaims()
	nonclaims[0] = "MUTATED"
	if protocol.RequiredNonclaims()[0] == "MUTATED" {
		t.Fatal("nonclaim mutation escaped defensive copy")
	}

	matrix := protocol.KindActionMatrix()
	matrix[protocol.KindKingdomReleaseRoot][0] = protocol.ActionGrant
	matrix["NEW_KIND"] = []protocol.Action{protocol.ActionSettle}
	fresh := protocol.KindActionMatrix()
	if len(fresh) != 10 || len(fresh[protocol.KindKingdomReleaseRoot]) != 1 || fresh[protocol.KindKingdomReleaseRoot][0] != protocol.ActionCheckpoint {
		t.Fatalf("action matrix mutation escaped defensive copy: %#v", fresh)
	}

	relations := protocol.AllowedLineageRelations()
	relations[0] = "MUTATED"
	for _, relation := range protocol.AllowedLineageRelations() {
		if relation == "MUTATED" {
			t.Fatal("lineage relation mutation escaped defensive copy")
		}
	}

	readiness := protocol.ActivationReadinessMatrix()
	readiness[0].Status = "ACTIVATED"
	readiness[0].Blockers[0] = "REMOVED"
	freshReadiness := protocol.ActivationReadinessMatrix()
	if len(freshReadiness) != 10 || freshReadiness[0].Status != protocol.ActivationStatusNotConsensusAdmissible || freshReadiness[0].Blockers[0] == "REMOVED" {
		t.Fatalf("activation matrix mutation escaped defensive copy: %#v", freshReadiness)
	}
}
