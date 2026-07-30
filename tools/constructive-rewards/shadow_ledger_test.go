package main

import (
	"encoding/json"
	"math"
	"reflect"
	"testing"
)

func testShadowLedger(t *testing.T) *ShadowCapacityLedger {
	t.Helper()
	ledger, err := NewShadowCapacityLedger(ShadowLedgerConfig{
		RootID:                 "semantic-root-alpha",
		PolicyDigest:           "policy-v1",
		LifetimeCap:            100,
		LifetimeReplacementCap: 60,
	})
	if err != nil {
		t.Fatal(err)
	}
	return ledger
}

func testAccrual() ShadowAccrual {
	return ShadowAccrual{
		LotID:                  "lot-alpha",
		ReceiptDigest:          "receipt-original",
		Amount:                 100,
		Deadline:               10,
		BeneficiaryControllers: []string{"originator"},
		DependencyControllers:  []string{"dependency-root"},
		EvaluatorControllers:   []string{"original-evaluator"},
	}
}

func testNamedAccrual(
	lotID string,
	receipt string,
	amount uint64,
	deadline uint64,
) ShadowAccrual {
	return ShadowAccrual{
		LotID:                  lotID,
		ReceiptDigest:          receipt,
		Amount:                 amount,
		Deadline:               deadline,
		BeneficiaryControllers: []string{"beneficiary-" + lotID},
		DependencyControllers:  []string{"dependency-" + lotID},
		EvaluatorControllers:   []string{"evaluator-" + lotID},
	}
}

func testInvalidation(target string, cleanSupport uint64) ShadowInvalidation {
	return ShadowInvalidation{
		DecisionID:             "decision-" + target,
		TargetReceiptDigest:    target,
		CleanSupportTarget:     cleanSupport,
		CulpableControllers:    []string{"originator"},
		ChallengerControllers:  []string{"challenger"},
		AdjudicatorControllers: []string{"adjudicator-a", "adjudicator-b", "adjudicator-c"},
		OrganizationRoots:      []string{"organization-a", "organization-b"},
		Final:                  true,
		AssignmentAfterFreeze:  true,
		OutcomeIndependentPay:  true,
	}
}

func testSuccessor(receipt, prior string, support uint64) ShadowSuccessor {
	return ShadowSuccessor{
		ReceiptDigest:          receipt,
		PriorReceiptDigest:     prior,
		SupportTarget:          support,
		BeneficiaryControllers: []string{"honest-successor"},
		DependencyControllers:  []string{"clean-dependency"},
		EvaluatorControllers:   []string{"clean-evaluator-a", "clean-evaluator-b", "clean-evaluator-c"},
		PolicyPassed:           true,
	}
}

func shadowEconomicTuple(snapshot ShadowLedgerSnapshot) [6]uint64 {
	return [6]uint64{
		snapshot.Accrued,
		snapshot.Funded,
		snapshot.Live,
		snapshot.Quarantined,
		snapshot.Extinguished,
		snapshot.ReplacementUsed,
	}
}

func recommitShadowSnapshot(t *testing.T, snapshot ShadowLedgerSnapshot) ShadowLedgerSnapshot {
	t.Helper()
	commitment, err := shadowSnapshotCommitment(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	snapshot.StateCommitment = commitment
	return snapshot
}

func cloneShadowSnapshotForTest(
	t *testing.T,
	snapshot ShadowLedgerSnapshot,
) ShadowLedgerSnapshot {
	t.Helper()
	encoded, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	var clone ShadowLedgerSnapshot
	if err := json.Unmarshal(encoded, &clone); err != nil {
		t.Fatal(err)
	}
	return clone
}

func testReplacementHistorySnapshot(t *testing.T) ShadowLedgerSnapshot {
	t.Helper()
	ledger, err := NewShadowCapacityLedger(ShadowLedgerConfig{
		RootID:                 "multi-lot-root",
		PolicyDigest:           "multi-lot-policy",
		LifetimeCap:            100,
		LifetimeReplacementCap: 100,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := ledger.Accrue(
		"accrue-a",
		1,
		testNamedAccrual("lot-a", "receipt-a", 50, 10),
	); err != nil {
		t.Fatal(err)
	}
	if err := ledger.Accrue(
		"accrue-b",
		2,
		testNamedAccrual("lot-b", "receipt-b", 50, 20),
	); err != nil {
		t.Fatal(err)
	}
	if err := ledger.FinalizeInvalidation(
		"invalidate-a",
		3,
		testInvalidation("receipt-a", 100),
	); err != nil {
		t.Fatal(err)
	}
	if moved, err := ledger.Reattribute(
		"reattribute-a",
		4,
		testSuccessor("successor-a", "receipt-a", 100),
	); err != nil || moved != 50 {
		t.Fatalf("first successor attribution = %d, %v; want 50", moved, err)
	}
	if funded, err := ledger.Fund("fund-a", 5, 50); err != nil || funded != 50 {
		t.Fatalf("first successor funding = %d, %v; want 50", funded, err)
	}
	if err := ledger.FinalizeInvalidation(
		"invalidate-b",
		6,
		testInvalidation("receipt-b", 100),
	); err != nil {
		t.Fatal(err)
	}
	if moved, err := ledger.Reattribute(
		"reattribute-b",
		7,
		testSuccessor("successor-b", "receipt-b", 100),
	); err != nil || moved != 50 {
		t.Fatalf("second successor attribution = %d, %v; want 50", moved, err)
	}
	return ledger.Snapshot()
}

func TestShadowReattributesOnlyUnpaidSupportedCapacity(t *testing.T) {
	ledger := testShadowLedger(t)
	if err := ledger.Accrue("accrue", 1, testAccrual()); err != nil {
		t.Fatal(err)
	}
	funded, err := ledger.Fund("fund", 2, 30)
	if err != nil {
		t.Fatal(err)
	}
	if funded != 30 {
		t.Fatalf("funded %d, want 30", funded)
	}
	if err := ledger.FinalizeInvalidation(
		"invalidate",
		3,
		testInvalidation("receipt-original", 0),
	); err != nil {
		t.Fatal(err)
	}
	invalidated := ledger.Snapshot()
	if got, want := shadowEconomicTuple(invalidated), [6]uint64{100, 30, 0, 60, 10, 0}; got != want {
		t.Fatalf("post-invalidation state %v, want %v", got, want)
	}

	moved, err := ledger.Reattribute(
		"reattribute",
		4,
		testSuccessor("receipt-successor", "receipt-original", 80),
	)
	if err != nil {
		t.Fatal(err)
	}
	if moved != 50 {
		t.Fatalf("moved %d, want deterministic maximum 50", moved)
	}
	replaced := ledger.Snapshot()
	if got, want := shadowEconomicTuple(replaced), [6]uint64{100, 30, 50, 10, 10, 50}; got != want {
		t.Fatalf("post-replacement state %v, want %v", got, want)
	}
	if replaced.Lots[0].Deadline != 10 {
		t.Fatalf("replacement deadline = %d, want inherited 10", replaced.Lots[0].Deadline)
	}
}

func TestShadowScenarioIsExactDeterministicAndZeroValue(t *testing.T) {
	first, err := RunShadowScenario()
	if err != nil {
		t.Fatal(err)
	}
	second, err := RunShadowScenario()
	if err != nil {
		t.Fatal(err)
	}
	firstJSON, err := json.Marshal(first)
	if err != nil {
		t.Fatal(err)
	}
	secondJSON, err := json.Marshal(second)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(firstJSON, secondJSON) {
		t.Fatal("same shadow scenario produced different JSON")
	}
	if first.Authoritative ||
		first.NetworkObserved ||
		first.RewardBearing ||
		first.TransferableValue ||
		first.MovesFunds ||
		first.SettlementZRN != 0 ||
		first.IntegrationReady {
		t.Fatalf("shadow scenario crossed a zero-value boundary: %+v", first)
	}
	if !first.Checks.Passed || len(first.Trace) != 4 {
		t.Fatalf("shadow scenario checks/trace failed: %+v", first)
	}
	final := first.Trace[len(first.Trace)-1].Snapshot
	if got, want := shadowEconomicTuple(final), [6]uint64{100, 30, 50, 10, 10, 50}; got != want {
		t.Fatalf("final shadow vector %v, want %v", got, want)
	}
}

func TestFundedCapacityNeverReopens(t *testing.T) {
	ledger := testShadowLedger(t)
	if err := ledger.Accrue("accrue", 1, testAccrual()); err != nil {
		t.Fatal(err)
	}
	if _, err := ledger.Fund("fund", 2, 30); err != nil {
		t.Fatal(err)
	}
	if err := ledger.FinalizeInvalidation(
		"invalidate",
		3,
		testInvalidation("receipt-original", 0),
	); err != nil {
		t.Fatal(err)
	}
	before := ledger.Snapshot()
	if _, err := ledger.Reattribute(
		"reattribute",
		4,
		testSuccessor("receipt-successor", "receipt-original", 80),
	); err != nil {
		t.Fatal(err)
	}
	after := ledger.Snapshot()
	if after.Funded != before.Funded || after.Accrued != before.Accrued {
		t.Fatalf("reattribution changed terminal/accrued counters: before %+v after %+v", before, after)
	}
	if err := ledger.FinalizeInvalidation(
		"invalidate-replacement",
		5,
		testInvalidation("receipt-successor", 30),
	); err != nil {
		t.Fatal(err)
	}
	failedReplacement := ledger.Snapshot()
	if failedReplacement.Funded != 30 ||
		failedReplacement.Accrued != 100 ||
		failedReplacement.Live != 0 ||
		failedReplacement.Quarantined != 10 ||
		failedReplacement.Extinguished != 60 {
		t.Fatalf("replacement failure reopened funded capacity: %+v", failedReplacement)
	}
}

func TestReplacementIsOneShotAndLifetimeBounded(t *testing.T) {
	ledger := testShadowLedger(t)
	if err := ledger.Accrue("accrue", 1, testAccrual()); err != nil {
		t.Fatal(err)
	}
	if _, err := ledger.Fund("fund", 2, 30); err != nil {
		t.Fatal(err)
	}
	if err := ledger.FinalizeInvalidation(
		"invalidate",
		3,
		testInvalidation("receipt-original", 0),
	); err != nil {
		t.Fatal(err)
	}
	first := testSuccessor("receipt-successor-one", "receipt-original", 70)
	first.BeneficiaryControllers = []string{"successor-one"}
	moved, err := ledger.Reattribute("replace-one", 4, first)
	if err != nil {
		t.Fatal(err)
	}
	if moved != 40 {
		t.Fatalf("first replacement moved %d, want 40", moved)
	}
	if err := ledger.FinalizeInvalidation(
		"invalidate-replacement",
		5,
		testInvalidation("receipt-successor-one", 30),
	); err != nil {
		t.Fatal(err)
	}
	afterFailure := ledger.Snapshot()
	if got, want := shadowEconomicTuple(afterFailure), [6]uint64{100, 30, 0, 20, 50, 40}; got != want {
		t.Fatalf("one-shot failure state %v, want %v", got, want)
	}

	restored, err := RestoreShadowCapacityLedger(
		afterFailure,
		afterFailure.StateCommitment,
	)
	if err != nil {
		t.Fatal(err)
	}
	second := testSuccessor("receipt-successor-two", "receipt-original", 100)
	second.BeneficiaryControllers = []string{"successor-two"}
	second.DependencyControllers = []string{"clean-dependency-two"}
	second.EvaluatorControllers = []string{
		"clean-evaluator-two-a",
		"clean-evaluator-two-b",
		"clean-evaluator-two-c",
	}
	moved, err = restored.Reattribute("replace-two", 6, second)
	if err != nil {
		t.Fatal(err)
	}
	if moved != 20 {
		t.Fatalf("second replacement moved %d, want untouched/capped 20", moved)
	}
	final := restored.Snapshot()
	if final.ReplacementUsed != 60 || final.Quarantined != 0 || final.Live != 20 {
		t.Fatalf("replacement lifetime cap did not survive restore: %+v", final)
	}
}

func TestHistoricalFundedSuccessorRemainsInvalidatable(t *testing.T) {
	ledger := testShadowLedger(t)
	if err := ledger.Accrue("accrue", 1, testAccrual()); err != nil {
		t.Fatal(err)
	}
	if _, err := ledger.Fund("fund-original", 2, 30); err != nil {
		t.Fatal(err)
	}
	if err := ledger.FinalizeInvalidation(
		"invalidate-original",
		3,
		testInvalidation("receipt-original", 0),
	); err != nil {
		t.Fatal(err)
	}

	first := testSuccessor("successor-first", "receipt-original", 50)
	first.BeneficiaryControllers = []string{"first-beneficiary"}
	first.DependencyControllers = []string{"first-dependency"}
	first.EvaluatorControllers = []string{
		"first-evaluator-a",
		"first-evaluator-b",
		"first-evaluator-c",
	}
	if moved, err := ledger.Reattribute("replace-first", 4, first); err != nil || moved != 20 {
		t.Fatalf("first replacement = %d, %v; want 20", moved, err)
	}
	if funded, err := ledger.Fund("fund-first", 5, 20); err != nil || funded != 20 {
		t.Fatalf("first replacement funding = %d, %v; want 20", funded, err)
	}

	second := testSuccessor("successor-second", "receipt-original", 80)
	second.BeneficiaryControllers = []string{"second-beneficiary"}
	second.DependencyControllers = []string{"second-dependency"}
	second.EvaluatorControllers = []string{
		"second-evaluator-a",
		"second-evaluator-b",
		"second-evaluator-c",
	}
	if moved, err := ledger.Reattribute("replace-second", 6, second); err != nil || moved != 30 {
		t.Fatalf("second replacement = %d, %v; want 30", moved, err)
	}

	oldDecision := testInvalidation("successor-first", 80)
	oldDecision.CulpableControllers = []string{"first-beneficiary"}
	if err := ledger.FinalizeInvalidation(
		"invalidate-funded-first",
		7,
		oldDecision,
	); err != nil {
		t.Fatal(err)
	}
	snapshot := ledger.Snapshot()
	if snapshot.Funded != 50 ||
		snapshot.Live != 30 ||
		snapshot.Quarantined != 10 ||
		snapshot.Extinguished != 10 {
		t.Fatalf("historical invalidation reopened or stole capacity: %+v", snapshot)
	}
	lot := snapshot.Lots[0]
	if len(lot.ReplacementClaims) != 2 ||
		lot.ReplacementClaims[0].ReceiptDigest != "successor-first" ||
		lot.ReplacementClaims[0].Funded != 20 ||
		lot.ReplacementClaims[0].Live != 0 ||
		lot.ReplacementClaims[1].ReceiptDigest != "successor-second" ||
		lot.ReplacementClaims[1].Live != 30 {
		t.Fatalf("successor claim history was not preserved: %+v", lot.ReplacementClaims)
	}
}

func TestReceiptDigestsCannotAccrueOrReenterTwice(t *testing.T) {
	ledger, err := NewShadowCapacityLedger(ShadowLedgerConfig{
		RootID:                 "receipt-root",
		PolicyDigest:           "receipt-policy",
		LifetimeCap:            200,
		LifetimeReplacementCap: 100,
	})
	if err != nil {
		t.Fatal(err)
	}
	first := testAccrual()
	first.Amount = 50
	if err := ledger.Accrue("accrue-first", 1, first); err != nil {
		t.Fatal(err)
	}
	duplicate := first
	duplicate.LotID = "lot-duplicate"
	before := ledger.Snapshot()
	if err := ledger.Accrue("accrue-duplicate", 2, duplicate); err == nil {
		t.Fatal("the same receipt digest should not create a second accrual lot")
	}
	if !reflect.DeepEqual(ledger.Snapshot(), before) {
		t.Fatal("rejected receipt replay changed state")
	}
}

func TestChallengeCannotQuarantineOrSteal(t *testing.T) {
	ledger := testShadowLedger(t)
	if err := ledger.Accrue("accrue", 1, testAccrual()); err != nil {
		t.Fatal(err)
	}
	before := ledger.Snapshot()
	if err := ledger.ObserveChallenge("raw-challenge", 2); err != nil {
		t.Fatal(err)
	}
	afterChallenge := ledger.Snapshot()
	if shadowEconomicTuple(afterChallenge) != shadowEconomicTuple(before) {
		t.Fatalf("raw challenge changed economic state: before %+v after %+v", before, afterChallenge)
	}

	unsigned := testInvalidation("receipt-original", 0)
	unsigned.Final = false
	if err := ledger.FinalizeInvalidation("unsigned", 3, unsigned); err == nil {
		t.Fatal("unsigned/raw challenge should not quarantine")
	}
	if shadowEconomicTuple(ledger.Snapshot()) != shadowEconomicTuple(before) {
		t.Fatal("rejected invalidation changed economic state")
	}

	if err := ledger.FinalizeInvalidation(
		"final-invalidation",
		3,
		testInvalidation("receipt-original", 0),
	); err != nil {
		t.Fatal(err)
	}
	thief := testSuccessor("thief", "receipt-original", 100)
	thief.BeneficiaryControllers = []string{"challenger"}
	beforeTheft := ledger.Snapshot()
	if _, err := ledger.Reattribute("steal", 4, thief); err == nil {
		t.Fatal("challenger should not collect quarantined capacity")
	}
	afterTheft := ledger.Snapshot()
	if shadowEconomicTuple(afterTheft) != shadowEconomicTuple(beforeTheft) ||
		!reflect.DeepEqual(afterTheft.ConsumedEventIDs, beforeTheft.ConsumedEventIDs) {
		t.Fatal("rejected replacement was not atomic")
	}
}

func TestExpiryPrecedesReplacementAndDeadlineIsInherited(t *testing.T) {
	ledger := testShadowLedger(t)
	if err := ledger.Accrue("accrue", 1, testAccrual()); err != nil {
		t.Fatal(err)
	}
	if _, err := ledger.Fund("fund", 2, 30); err != nil {
		t.Fatal(err)
	}
	if err := ledger.FinalizeInvalidation(
		"invalidate",
		3,
		testInvalidation("receipt-original", 0),
	); err != nil {
		t.Fatal(err)
	}
	if _, err := ledger.Reattribute(
		"late-replacement",
		10,
		testSuccessor("late", "receipt-original", 100),
	); err == nil {
		t.Fatal("replacement at the inherited deadline should fail")
	}
	expired := ledger.Snapshot()
	if got, want := shadowEconomicTuple(expired), [6]uint64{100, 30, 0, 0, 70, 0}; got != want {
		t.Fatalf("expiry-first state %v, want %v", got, want)
	}

	early := testShadowLedger(t)
	if err := early.Accrue("accrue", 1, testAccrual()); err != nil {
		t.Fatal(err)
	}
	if _, err := early.Fund("fund", 2, 30); err != nil {
		t.Fatal(err)
	}
	if err := early.FinalizeInvalidation(
		"invalidate",
		3,
		testInvalidation("receipt-original", 0),
	); err != nil {
		t.Fatal(err)
	}
	if _, err := early.Reattribute(
		"early-replacement",
		9,
		testSuccessor("early", "receipt-original", 80),
	); err != nil {
		t.Fatal(err)
	}
	if early.Snapshot().Lots[0].Deadline != 10 {
		t.Fatal("early replacement renewed its source deadline")
	}
	if funded, err := early.Fund("fund-at-deadline", 10, 100); err != nil || funded != 0 {
		t.Fatalf("deadline funding = %d, %v; want expiry before zero funding", funded, err)
	}
	if early.Snapshot().Live != 0 {
		t.Fatal("replacement remained live at its inherited deadline")
	}
}

func TestControllerMergePropagatesExclusion(t *testing.T) {
	ledger := testShadowLedger(t)
	if err := ledger.Accrue("accrue", 1, testAccrual()); err != nil {
		t.Fatal(err)
	}
	if _, err := ledger.Fund("fund", 2, 30); err != nil {
		t.Fatal(err)
	}
	if err := ledger.FinalizeInvalidation(
		"invalidate",
		3,
		testInvalidation("receipt-original", 0),
	); err != nil {
		t.Fatal(err)
	}
	successor := testSuccessor("replacement", "receipt-original", 80)
	successor.BeneficiaryControllers = []string{"apparently-independent"}
	if _, err := ledger.Reattribute("replace", 4, successor); err != nil {
		t.Fatal(err)
	}
	if err := ledger.LinkControllers(
		"merge",
		5,
		[]string{"apparently-independent", "challenger"},
	); err != nil {
		t.Fatal(err)
	}
	merged := ledger.Snapshot()
	if merged.Live != 0 || merged.Extinguished != 60 || merged.Quarantined != 10 {
		t.Fatalf("controller merge did not extinguish tainted live replacement: %+v", merged)
	}

	before := merged
	if err := ledger.LinkControllers(
		"split-shaped-link",
		6,
		[]string{"apparently-independent", "new-alias"},
	); err != nil {
		t.Fatal(err)
	}
	after := ledger.Snapshot()
	if len(after.ExcludedControllers) <= len(before.ExcludedControllers) {
		t.Fatal("later alias link cleared rather than expanded exclusion")
	}
}

func TestKnownControllerAliasesCannotOccupyDisjointRoles(t *testing.T) {
	ledger := testShadowLedger(t)
	if err := ledger.LinkControllers(
		"prelink-origin",
		1,
		[]string{"alias-beneficiary", "alias-evaluator"},
	); err != nil {
		t.Fatal(err)
	}
	aliasedAccrual := testAccrual()
	aliasedAccrual.BeneficiaryControllers = []string{"alias-beneficiary"}
	aliasedAccrual.EvaluatorControllers = []string{"alias-evaluator"}
	beforeAccrual := ledger.Snapshot()
	if err := ledger.Accrue("aliased-accrual", 2, aliasedAccrual); err == nil {
		t.Fatal("effective controller aliases should not fill disjoint accrual roles")
	}
	if !reflect.DeepEqual(ledger.Snapshot(), beforeAccrual) {
		t.Fatal("rejected aliased accrual changed state")
	}

	invalidationLedger := testShadowLedger(t)
	if err := invalidationLedger.Accrue("accrue", 1, testAccrual()); err != nil {
		t.Fatal(err)
	}
	if err := invalidationLedger.LinkControllers(
		"prelink-adjudicators",
		2,
		[]string{"adjudicator-a", "adjudicator-b"},
	); err != nil {
		t.Fatal(err)
	}
	beforeInvalidation := invalidationLedger.Snapshot()
	if err := invalidationLedger.FinalizeInvalidation(
		"aliased-invalidation",
		3,
		testInvalidation("receipt-original", 0),
	); err == nil {
		t.Fatal("three labels under fewer than three effective adjudicator roots should fail")
	}
	if !reflect.DeepEqual(invalidationLedger.Snapshot(), beforeInvalidation) {
		t.Fatal("rejected aliased invalidation changed state")
	}

	organizationLedger := testShadowLedger(t)
	if err := organizationLedger.Accrue("accrue", 1, testAccrual()); err != nil {
		t.Fatal(err)
	}
	if err := organizationLedger.LinkControllers(
		"prelink-organizations",
		2,
		[]string{"organization-a", "organization-b"},
	); err != nil {
		t.Fatal(err)
	}
	beforeOrganizationDecision := organizationLedger.Snapshot()
	if err := organizationLedger.FinalizeInvalidation(
		"aliased-organizations",
		3,
		testInvalidation("receipt-original", 0),
	); err == nil {
		t.Fatal("two organization labels under one effective root should fail")
	}
	if !reflect.DeepEqual(
		organizationLedger.Snapshot(),
		beforeOrganizationDecision,
	) {
		t.Fatal("rejected organization-root alias changed state")
	}

	fundingLedger := testShadowLedger(t)
	if err := fundingLedger.Accrue("accrue", 1, testAccrual()); err != nil {
		t.Fatal(err)
	}
	if err := fundingLedger.LinkControllers(
		"late-role-collapse",
		2,
		[]string{"original-evaluator", "originator"},
	); err != nil {
		t.Fatal(err)
	}
	beforeFunding := fundingLedger.Snapshot()
	if _, err := fundingLedger.Fund("fund-after-collapse", 3, 10); err == nil {
		t.Fatal("funding should recheck effective role separation after a controller merge")
	}
	if !reflect.DeepEqual(fundingLedger.Snapshot(), beforeFunding) {
		t.Fatal("rejected funding after role collapse changed state")
	}

	excludedFundingLedger, err := NewShadowCapacityLedger(ShadowLedgerConfig{
		RootID:                       "late-exclusion-root",
		PolicyDigest:                 "late-exclusion-policy",
		LifetimeCap:                  100,
		LifetimeReplacementCap:       60,
		InitiallyExcludedControllers: []string{"barred-controller"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := excludedFundingLedger.Accrue("accrue", 1, testAccrual()); err != nil {
		t.Fatal(err)
	}
	if err := excludedFundingLedger.LinkControllers(
		"late-exclusion",
		2,
		[]string{"barred-controller", "originator"},
	); err != nil {
		t.Fatal(err)
	}
	beforeExcludedFunding := excludedFundingLedger.Snapshot()
	if _, err := excludedFundingLedger.Fund(
		"fund-excluded-original",
		3,
		10,
	); err == nil {
		t.Fatal("funding should recheck original controllers against later exclusions")
	}
	if !reflect.DeepEqual(
		excludedFundingLedger.Snapshot(),
		beforeExcludedFunding,
	) {
		t.Fatal("rejected funding for an excluded original changed state")
	}
}

func TestShadowSnapshotRejectsCounterAndPartitionTampering(t *testing.T) {
	ledger := testShadowLedger(t)
	if err := ledger.Accrue("accrue", 1, testAccrual()); err != nil {
		t.Fatal(err)
	}
	snapshot := ledger.Snapshot()
	encodedSnapshot, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	var roundTripped ShadowLedgerSnapshot
	if err := json.Unmarshal(encodedSnapshot, &roundTripped); err != nil {
		t.Fatal(err)
	}
	if _, err := RestoreShadowCapacityLedger(
		roundTripped,
		snapshot.StateCommitment,
	); err != nil {
		t.Fatalf("valid JSON snapshot failed committed restore: %v", err)
	}

	tamperedCounter := snapshot
	tamperedCounter.ReplacementUsed = 1
	tamperedCounter = recommitShadowSnapshot(t, tamperedCounter)
	if _, err := RestoreShadowCapacityLedger(
		tamperedCounter,
		tamperedCounter.StateCommitment,
	); err == nil {
		t.Fatal("tampered replacement counter should fail restore")
	}

	tamperedLot := snapshot
	tamperedLot.Lots = append([]ShadowCapacityLot(nil), snapshot.Lots...)
	tamperedLot.Lots[0].OriginalLive--
	tamperedLot = recommitShadowSnapshot(t, tamperedLot)
	if _, err := RestoreShadowCapacityLedger(
		tamperedLot,
		tamperedLot.StateCommitment,
	); err == nil {
		t.Fatal("tampered lot partition should fail restore")
	}

	resurrectedExpiredLot := snapshot
	resurrectedExpiredLot.CurrentEpoch = snapshot.Lots[0].Deadline
	resurrectedExpiredLot = recommitShadowSnapshot(t, resurrectedExpiredLot)
	if _, err := RestoreShadowCapacityLedger(
		resurrectedExpiredLot,
		resurrectedExpiredLot.StateCommitment,
	); err == nil {
		t.Fatal("snapshot must not resurrect live capacity at its deadline")
	}

	cyclicControllers := snapshot
	cyclicControllers.ControllerLinks = map[string]string{
		"controller-a": "controller-b",
		"controller-b": "controller-a",
	}
	cyclicControllers = recommitShadowSnapshot(t, cyclicControllers)
	if _, err := RestoreShadowCapacityLedger(
		cyclicControllers,
		cyclicControllers.StateCommitment,
	); err == nil {
		t.Fatal("cyclic controller snapshot should fail restore")
	}

	danglingController := snapshot
	danglingController.ControllerLinks = map[string]string{
		"controller-a": "missing-parent",
	}
	danglingController = recommitShadowSnapshot(t, danglingController)
	if _, err := RestoreShadowCapacityLedger(
		danglingController,
		danglingController.StateCommitment,
	); err == nil {
		t.Fatal("dangling controller snapshot should fail restore")
	}

	incompleteReplacement := snapshot
	incompleteReplacement.Lots = append([]ShadowCapacityLot(nil), snapshot.Lots...)
	incompleteReplacement.Lots[0].ReplacementAttributed = 1
	incompleteReplacement.ReplacementUsed = 1
	incompleteReplacement = recommitShadowSnapshot(t, incompleteReplacement)
	if _, err := RestoreShadowCapacityLedger(
		incompleteReplacement,
		incompleteReplacement.StateCommitment,
	); err == nil {
		t.Fatal("incomplete replacement metadata should fail restore")
	}

	missingReplayID := snapshot
	missingReplayID.ConsumedEventIDs = nil
	missingReplayID = recommitShadowSnapshot(t, missingReplayID)
	if _, err := RestoreShadowCapacityLedger(
		missingReplayID,
		missingReplayID.StateCommitment,
	); err == nil {
		t.Fatal("snapshot must not drop replay IDs while retaining its event counter")
	}

	rewrittenReplayIdentity := snapshot
	rewrittenReplayIdentity.ConsumedEventIDs = []string{"fabricated-accrual"}
	rewrittenReplayIdentity = recommitShadowSnapshot(t, rewrittenReplayIdentity)
	if _, err := RestoreShadowCapacityLedger(
		rewrittenReplayIdentity,
		snapshot.StateCommitment,
	); err == nil {
		t.Fatal("rewritten replay identities must fail the separately trusted commitment")
	}
	if _, err := RestoreShadowCapacityLedger(snapshot, ""); err == nil {
		t.Fatal("restore must require a separately trusted state commitment")
	}

	invalidUTF8 := snapshot
	invalidUTF8.ConsumedEventIDs = []string{string([]byte{0xff})}
	invalidUTF8 = recommitShadowSnapshot(t, invalidUTF8)
	if _, err := RestoreShadowCapacityLedger(
		invalidUTF8,
		invalidUTF8.StateCommitment,
	); err == nil {
		t.Fatal("non-UTF-8 identifiers must fail before JSON commitment ambiguity")
	}

	excludedLedger, err := NewShadowCapacityLedger(ShadowLedgerConfig{
		RootID:                       "excluded-root",
		PolicyDigest:                 "excluded-policy",
		LifetimeCap:                  10,
		LifetimeReplacementCap:       5,
		InitiallyExcludedControllers: []string{"barred-controller"},
	})
	if err != nil {
		t.Fatal(err)
	}
	missingInitialExclusion := excludedLedger.Snapshot()
	missingInitialExclusion.ControllerLinks = map[string]string{}
	missingInitialExclusion.ExcludedControllers = nil
	missingInitialExclusion = recommitShadowSnapshot(t, missingInitialExclusion)
	if _, err := RestoreShadowCapacityLedger(
		missingInitialExclusion,
		missingInitialExclusion.StateCommitment,
	); err == nil {
		t.Fatal("snapshot must not clear a policy-snapshotted controller exclusion")
	}
}

func TestShadowSnapshotRejectsUnreachableReplacementStates(t *testing.T) {
	snapshot := testReplacementHistorySnapshot(t)
	if _, err := RestoreShadowCapacityLedger(
		snapshot,
		snapshot.StateCommitment,
	); err != nil {
		t.Fatalf("valid multi-lot replacement history failed restore: %v", err)
	}

	t.Run("concurrent live successors", func(t *testing.T) {
		mutated := cloneShadowSnapshotForTest(t, snapshot)
		firstLot := &mutated.Lots[0]
		firstClaim := &firstLot.ReplacementClaims[0]
		if firstClaim.Funded != 50 || firstClaim.Live != 0 {
			t.Fatalf("unexpected first claim fixture: %+v", *firstClaim)
		}
		firstClaim.Funded = 0
		firstClaim.Live = 50
		firstLot.Funded = 0
		firstLot.ReplacementLive = 50
		firstLot.ReplacementReceiptDigest = firstClaim.ReceiptDigest
		firstLot.ReplacementControllers = append(
			[]string(nil),
			firstClaim.Controllers...,
		)
		mutated.Funded = 0
		mutated.Live += 50
		mutated = recommitShadowSnapshot(t, mutated)
		if _, err := RestoreShadowCapacityLedger(
			mutated,
			mutated.StateCommitment,
		); err == nil {
			t.Fatal("snapshot with two globally live successors should fail restore")
		}
	})

	t.Run("successor receipt reused across lots", func(t *testing.T) {
		mutated := cloneShadowSnapshotForTest(t, snapshot)
		reusedReceipt := mutated.Lots[0].ReplacementClaims[0].ReceiptDigest
		secondLot := &mutated.Lots[1]
		secondLot.ReplacementClaims[0].ReceiptDigest = reusedReceipt
		secondLot.ReplacementReceiptHistory[0] = reusedReceipt
		secondLot.ReplacementReceiptDigest = reusedReceipt
		mutated = recommitShadowSnapshot(t, mutated)
		if _, err := RestoreShadowCapacityLedger(
			mutated,
			mutated.StateCommitment,
		); err == nil {
			t.Fatal("snapshot reusing one successor receipt across lots should fail restore")
		}
	})

	t.Run("replacement source was never invalidated", func(t *testing.T) {
		mutated := cloneShadowSnapshotForTest(t, snapshot)
		mutated.ConsumedDecisionIDs = []string{"decision-receipt-a"}
		mutated.InvalidatedReceipts = []string{"receipt-a"}
		mutated = recommitShadowSnapshot(t, mutated)
		if _, err := RestoreShadowCapacityLedger(
			mutated,
			mutated.StateCommitment,
		); err == nil {
			t.Fatal("replacement attribution without source invalidation should fail restore")
		}
	})

	t.Run("quarantine source was never invalidated", func(t *testing.T) {
		ledger := testShadowLedger(t)
		if err := ledger.Accrue("accrue", 1, testAccrual()); err != nil {
			t.Fatal(err)
		}
		mutated := ledger.Snapshot()
		mutated.Lots[0].OriginalLive -= 10
		mutated.Lots[0].Quarantined = 10
		mutated.Live -= 10
		mutated.Quarantined = 10
		mutated = recommitShadowSnapshot(t, mutated)
		if _, err := RestoreShadowCapacityLedger(
			mutated,
			mutated.StateCommitment,
		); err == nil {
			t.Fatal("quarantine without source invalidation should fail restore")
		}
	})

	t.Run("live successor is not last in chronological history", func(t *testing.T) {
		ledger := testShadowLedger(t)
		if err := ledger.Accrue("accrue", 1, testAccrual()); err != nil {
			t.Fatal(err)
		}
		if err := ledger.FinalizeInvalidation(
			"invalidate",
			2,
			testInvalidation("receipt-original", 0),
		); err != nil {
			t.Fatal(err)
		}
		if moved, err := ledger.Reattribute(
			"reattribute-first",
			3,
			testSuccessor("successor-first", "receipt-original", 20),
		); err != nil || moved != 20 {
			t.Fatalf("first successor attribution = %d, %v; want 20", moved, err)
		}
		if funded, err := ledger.Fund(
			"fund-first",
			4,
			20,
		); err != nil || funded != 20 {
			t.Fatalf("first successor funding = %d, %v; want 20", funded, err)
		}
		if moved, err := ledger.Reattribute(
			"reattribute-second",
			5,
			testSuccessor("successor-second", "receipt-original", 60),
		); err != nil || moved != 40 {
			t.Fatalf("second successor attribution = %d, %v; want 40", moved, err)
		}
		mutated := ledger.Snapshot()
		lot := &mutated.Lots[0]
		lot.ReplacementClaims[0], lot.ReplacementClaims[1] =
			lot.ReplacementClaims[1], lot.ReplacementClaims[0]
		lot.ReplacementReceiptHistory[0], lot.ReplacementReceiptHistory[1] =
			lot.ReplacementReceiptHistory[1], lot.ReplacementReceiptHistory[0]
		mutated = recommitShadowSnapshot(t, mutated)
		if _, err := RestoreShadowCapacityLedger(
			mutated,
			mutated.StateCommitment,
		); err == nil {
			t.Fatal("live successor before a terminal history entry should fail restore")
		}
	})
}

func TestShadowSnapshotRestoreRequiresCanonicalRepresentation(t *testing.T) {
	snapshot := testReplacementHistorySnapshot(t)
	restored, err := RestoreShadowCapacityLedger(
		snapshot,
		snapshot.StateCommitment,
	)
	if err != nil {
		t.Fatal(err)
	}
	if got := restored.Snapshot().StateCommitment; got != snapshot.StateCommitment {
		t.Fatalf(
			"canonical restore changed commitment from %s to %s",
			snapshot.StateCommitment,
			got,
		)
	}

	tests := []struct {
		name   string
		mutate func(*ShadowLedgerSnapshot)
	}{
		{
			name: "reordered lots",
			mutate: func(mutated *ShadowLedgerSnapshot) {
				mutated.Lots[0], mutated.Lots[1] = mutated.Lots[1], mutated.Lots[0]
			},
		},
		{
			name: "reordered replay IDs",
			mutate: func(mutated *ShadowLedgerSnapshot) {
				last := len(mutated.ConsumedEventIDs) - 1
				mutated.ConsumedEventIDs[0], mutated.ConsumedEventIDs[last] =
					mutated.ConsumedEventIDs[last], mutated.ConsumedEventIDs[0]
			},
		},
		{
			name: "duplicate exclusion",
			mutate: func(mutated *ShadowLedgerSnapshot) {
				mutated.ExcludedControllers = append(
					mutated.ExcludedControllers,
					mutated.ExcludedControllers[0],
				)
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mutated := cloneShadowSnapshotForTest(t, snapshot)
			test.mutate(&mutated)
			mutated = recommitShadowSnapshot(t, mutated)
			if _, err := RestoreShadowCapacityLedger(
				mutated,
				mutated.StateCommitment,
			); err == nil {
				t.Fatal("noncanonical committed snapshot should fail restore")
			}
		})
	}

	t.Run("null set instead of canonical empty array", func(t *testing.T) {
		emptyLedger := testShadowLedger(t)
		mutated := emptyLedger.Snapshot()
		if mutated.ConsumedEventIDs == nil {
			t.Fatal("empty snapshot fixture did not emit a canonical empty replay array")
		}
		mutated.ConsumedEventIDs = nil
		mutated = recommitShadowSnapshot(t, mutated)
		if _, err := RestoreShadowCapacityLedger(
			mutated,
			mutated.StateCommitment,
		); err == nil {
			t.Fatal("noncanonical null replay set should fail restore")
		}
	})
}

func TestShadowSnapshotRejectsUnreachableControllerGraphs(t *testing.T) {
	snapshot := testShadowLedger(t).Snapshot()
	tests := []struct {
		name  string
		links map[string]string
	}{
		{
			name: "non-flat chain",
			links: map[string]string{
				"controller-a": "controller-b",
				"controller-b": "controller-c",
				"controller-c": "controller-c",
			},
		},
		{
			name: "root is not lexical minimum",
			links: map[string]string{
				"controller-a": "controller-b",
				"controller-b": "controller-b",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mutated := cloneShadowSnapshotForTest(t, snapshot)
			mutated.ControllerLinks = test.links
			mutated = recommitShadowSnapshot(t, mutated)
			if _, err := RestoreShadowCapacityLedger(
				mutated,
				mutated.StateCommitment,
			); err == nil {
				t.Fatal("unreachable controller-link graph should fail restore")
			}
		})
	}
}

func TestShadowFundingUsesDeadlineThenLotIDOrdering(t *testing.T) {
	ledger, err := NewShadowCapacityLedger(ShadowLedgerConfig{
		RootID:                 "ordering-root",
		PolicyDigest:           "ordering-policy",
		LifetimeCap:            30,
		LifetimeReplacementCap: 0,
	})
	if err != nil {
		t.Fatal(err)
	}
	for index, accrual := range []ShadowAccrual{
		testNamedAccrual("lot-z", "receipt-z", 10, 10),
		testNamedAccrual("lot-b", "receipt-b", 10, 5),
		testNamedAccrual("lot-a", "receipt-a", 10, 5),
	} {
		if err := ledger.Accrue(
			"accrue-order-"+accrual.LotID,
			uint64(index+1),
			accrual,
		); err != nil {
			t.Fatal(err)
		}
	}
	if funded, err := ledger.Fund("fund-ordered", 4, 15); err != nil || funded != 15 {
		t.Fatalf("ordered funding = %d, %v; want 15", funded, err)
	}
	fundedByLot := make(map[string]uint64)
	for _, lot := range ledger.Snapshot().Lots {
		fundedByLot[lot.ID] = lot.Funded
	}
	if got, want := fundedByLot["lot-a"], uint64(10); got != want {
		t.Fatalf("lot-a funded %d, want %d", got, want)
	}
	if got, want := fundedByLot["lot-b"], uint64(5); got != want {
		t.Fatalf("lot-b funded %d, want %d", got, want)
	}
	if got, want := fundedByLot["lot-z"], uint64(0); got != want {
		t.Fatalf("lot-z funded %d, want %d", got, want)
	}
}

func TestShadowUint64CapAndDeadlineBoundaries(t *testing.T) {
	t.Run("capacity maximum and overflow", func(t *testing.T) {
		ledger, err := NewShadowCapacityLedger(ShadowLedgerConfig{
			RootID:                 "maximum-cap-root",
			PolicyDigest:           "maximum-cap-policy",
			LifetimeCap:            math.MaxUint64,
			LifetimeReplacementCap: math.MaxUint64,
		})
		if err != nil {
			t.Fatal(err)
		}
		if err := ledger.Accrue(
			"accrue-maximum",
			0,
			testNamedAccrual(
				"lot-maximum",
				"receipt-maximum",
				math.MaxUint64,
				math.MaxUint64,
			),
		); err != nil {
			t.Fatal(err)
		}
		beforeOverflow := ledger.Snapshot()
		if err := ledger.Accrue(
			"accrue-overflow",
			0,
			testNamedAccrual("lot-overflow", "receipt-overflow", 1, math.MaxUint64),
		); err == nil {
			t.Fatal("accrual beyond MaxUint64 should fail")
		}
		if !reflect.DeepEqual(ledger.Snapshot(), beforeOverflow) {
			t.Fatal("overflowing accrual changed state")
		}
		funded, err := ledger.Fund("fund-maximum", 1, math.MaxUint64)
		if err != nil {
			t.Fatal(err)
		}
		if funded != math.MaxUint64 {
			t.Fatalf("funded %d, want MaxUint64", funded)
		}
		final := ledger.Snapshot()
		if final.Funded != math.MaxUint64 || final.Live != 0 {
			t.Fatalf("maximum-cap terminal state is inconsistent: %+v", final)
		}
		aggregateOverflow := cloneShadowSnapshotForTest(t, final)
		extra := testNamedAccrual(
			"lot-overflow-restore",
			"receipt-overflow-restore",
			1,
			math.MaxUint64,
		)
		aggregateOverflow.Lots = append(
			aggregateOverflow.Lots,
			ShadowCapacityLot{
				ID:                     extra.LotID,
				ReceiptDigest:          extra.ReceiptDigest,
				Amount:                 extra.Amount,
				Extinguished:           extra.Amount,
				Deadline:               extra.Deadline,
				BeneficiaryControllers: extra.BeneficiaryControllers,
				DependencyControllers:  extra.DependencyControllers,
				EvaluatorControllers:   extra.EvaluatorControllers,
			},
		)
		aggregateOverflow.Extinguished = 1
		aggregateOverflow = recommitShadowSnapshot(t, aggregateOverflow)
		if _, err := RestoreShadowCapacityLedger(
			aggregateOverflow,
			aggregateOverflow.StateCommitment,
		); err == nil {
			t.Fatal("snapshot lot aggregation beyond MaxUint64 should fail restore")
		}
	})

	t.Run("deadline maximum", func(t *testing.T) {
		ledger, err := NewShadowCapacityLedger(ShadowLedgerConfig{
			RootID:                 "maximum-deadline-root",
			PolicyDigest:           "maximum-deadline-policy",
			LifetimeCap:            1,
			LifetimeReplacementCap: 1,
		})
		if err != nil {
			t.Fatal(err)
		}
		if err := ledger.ObserveChallenge("advance-near-maximum", math.MaxUint64-1); err != nil {
			t.Fatal(err)
		}
		if err := ledger.Accrue(
			"accrue-at-maximum-deadline",
			math.MaxUint64-1,
			testNamedAccrual(
				"lot-maximum-deadline",
				"receipt-maximum-deadline",
				1,
				math.MaxUint64,
			),
		); err != nil {
			t.Fatal(err)
		}
		if err := ledger.ObserveChallenge("reach-maximum-deadline", math.MaxUint64); err != nil {
			t.Fatal(err)
		}
		final := ledger.Snapshot()
		if final.CurrentEpoch != math.MaxUint64 ||
			final.Live != 0 ||
			final.Extinguished != 1 {
			t.Fatalf("maximum deadline did not expire exactly: %+v", final)
		}
	})
}
