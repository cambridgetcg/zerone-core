package main

import (
	"fmt"
	"reflect"
	"testing"
)

func FuzzShadowLedgerConservation(f *testing.F) {
	f.Add(
		uint16(100),
		uint16(60),
		[]byte{0, 2, 3, 0, 4, 3, 5, 1, 0},
	)
	f.Add(
		uint16(1),
		uint16(0),
		[]byte{2, 3, 4, 5, 0, 1},
	)

	f.Fuzz(func(t *testing.T, capRaw, replacementRaw uint16, operations []byte) {
		if len(operations) > 128 {
			operations = operations[:128]
		}
		capacity := uint64(capRaw%10_000) + 1
		replacementCap := uint64(replacementRaw) % (capacity + 1)
		ledger, err := NewShadowCapacityLedger(ShadowLedgerConfig{
			RootID:                 "fuzz-root",
			PolicyDigest:           "fuzz-policy",
			LifetimeCap:            capacity,
			LifetimeReplacementCap: replacementCap,
		})
		if err != nil {
			t.Fatal(err)
		}
		if err := ledger.Accrue("initial-accrual", 1, ShadowAccrual{
			LotID:                  "fuzz-lot",
			ReceiptDigest:          "fuzz-original",
			Amount:                 capacity,
			Deadline:               1_000,
			BeneficiaryControllers: []string{"fuzz-originator"},
			DependencyControllers:  []string{"fuzz-dependency"},
			EvaluatorControllers:   []string{"fuzz-original-evaluator"},
		}); err != nil {
			t.Fatal(err)
		}

		for index, operation := range operations {
			now := uint64(index) + 2
			eventID := fmt.Sprintf("fuzz-event-%03d", index)
			before := ledger.Snapshot()
			var eventErr error
			switch operation % 6 {
			case 0:
				_, eventErr = ledger.Fund(
					eventID,
					now,
					uint64(operation)%capacity+1,
				)
			case 1:
				eventErr = ledger.ObserveChallenge(eventID, now)
			case 2:
				eventErr = ledger.FinalizeInvalidation(
					eventID,
					now,
					ShadowInvalidation{
						DecisionID:             "fuzz-original-decision",
						TargetReceiptDigest:    "fuzz-original",
						CleanSupportTarget:     before.Funded,
						CulpableControllers:    []string{"fuzz-originator"},
						ChallengerControllers:  []string{"fuzz-challenger"},
						AdjudicatorControllers: []string{"fuzz-adjudicator-a", "fuzz-adjudicator-b", "fuzz-adjudicator-c"},
						OrganizationRoots:      []string{"fuzz-organization-a", "fuzz-organization-b"},
						Final:                  true,
						AssignmentAfterFreeze:  true,
						OutcomeIndependentPay:  true,
					},
				)
			case 3:
				receipt := fmt.Sprintf("fuzz-successor-%03d", index)
				_, eventErr = ledger.Reattribute(
					eventID,
					now,
					ShadowSuccessor{
						ReceiptDigest:          receipt,
						PriorReceiptDigest:     "fuzz-original",
						SupportTarget:          before.Accrued,
						BeneficiaryControllers: []string{fmt.Sprintf("fuzz-beneficiary-%03d", index)},
						DependencyControllers:  []string{fmt.Sprintf("fuzz-clean-dependency-%03d", index)},
						EvaluatorControllers: []string{
							fmt.Sprintf("fuzz-clean-evaluator-%03d-a", index),
							fmt.Sprintf("fuzz-clean-evaluator-%03d-b", index),
							fmt.Sprintf("fuzz-clean-evaluator-%03d-c", index),
						},
						PolicyPassed: true,
					},
				)
			case 4:
				target := ""
				if len(before.Lots) > 0 {
					target = before.Lots[0].ReplacementReceiptDigest
				}
				eventErr = ledger.FinalizeInvalidation(
					eventID,
					now,
					ShadowInvalidation{
						DecisionID:             "decision-" + target,
						TargetReceiptDigest:    target,
						CleanSupportTarget:     before.Funded,
						CulpableControllers:    []string{"fuzz-originator"},
						ChallengerControllers:  []string{"fuzz-second-challenger"},
						AdjudicatorControllers: []string{"fuzz-second-adjudicator-a", "fuzz-second-adjudicator-b", "fuzz-second-adjudicator-c"},
						OrganizationRoots:      []string{"fuzz-second-organization-a", "fuzz-second-organization-b"},
						Final:                  true,
						AssignmentAfterFreeze:  true,
						OutcomeIndependentPay:  true,
					},
				)
			case 5:
				controller := fmt.Sprintf("fuzz-unlinked-%03d", index)
				if len(before.Lots) > 0 &&
					len(before.Lots[0].ReplacementControllers) > 0 {
					controller = before.Lots[0].ReplacementControllers[0]
				}
				linked := []string{"fuzz-challenger", controller}
				if linked[0] > linked[1] {
					linked[0], linked[1] = linked[1], linked[0]
				}
				eventErr = ledger.LinkControllers(eventID, now, linked)
			}

			after := ledger.Snapshot()
			if eventErr != nil && !reflect.DeepEqual(after, before) {
				t.Fatalf(
					"rejected operation %d mutated state: before %+v after %+v: %v",
					operation%6,
					before,
					after,
					eventErr,
				)
			}
			if after.Accrued < before.Accrued ||
				after.Funded < before.Funded ||
				after.Extinguished < before.Extinguished ||
				after.ReplacementUsed < before.ReplacementUsed {
				t.Fatalf("monotone counter regressed: before %+v after %+v", before, after)
			}
			if after.Accrued != after.Funded+after.Live+after.Quarantined+after.Extinguished {
				t.Fatalf("partition failed: %+v", after)
			}
			if after.ReplacementUsed+after.Quarantined > replacementCap {
				t.Fatalf("replacement exposure exceeded cap: %+v", after)
			}
			for _, lot := range after.Lots {
				if lot.ReplacementGeneration > 1 || lot.Deadline != 1_000 {
					t.Fatalf("generation/deadline invariant failed: %+v", lot)
				}
			}
		}

		beforeExpiry := ledger.Snapshot()
		if err := ledger.ObserveChallenge("fuzz-expiry", 1_000); err != nil {
			t.Fatal(err)
		}
		expired := ledger.Snapshot()
		if expired.Accrued != expired.Funded+expired.Extinguished ||
			expired.Live != 0 ||
			expired.Quarantined != 0 ||
			expired.Accrued < beforeExpiry.Accrued ||
			expired.Funded < beforeExpiry.Funded ||
			expired.Extinguished < beforeExpiry.Extinguished ||
			expired.ReplacementUsed < beforeExpiry.ReplacementUsed {
			t.Fatalf("terminal expiry invariant failed: before %+v after %+v", beforeExpiry, expired)
		}
		if _, err := RestoreShadowCapacityLedger(
			expired,
			expired.StateCommitment,
		); err != nil {
			t.Fatalf("valid terminal snapshot failed restore: %v", err)
		}
	})
}
