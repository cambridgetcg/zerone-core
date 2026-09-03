package main

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

const missingFinding = "MISSING"

type testSigner struct {
	role          string
	identity      string
	controlDomain string
	publicKey     ed25519.PublicKey
	privateKey    ed25519.PrivateKey
}

type recoveryFixture struct {
	custodySigners         []testSigner
	controlledSigners      []testSigner
	forkSigners            []testSigner
	reproducerSigners      []testSigner
	custodyPolicy          SignerPolicy
	custodyPolicyDigest    string
	assessment             CustodyAssessment
	assessmentDigest       string
	controlledPolicy       SignerPolicy
	controlledPolicyDigest string
	controlled             ControlledTransition
	controlledDigest       string
	policy                 ForkPolicy
	policyDigest           string
	genesis                ForkGenesis
	genesisDigest          string
	reports                []ForkGenesisReport
	reportDigests          []string
	release                ForkRelease
	releaseDigest          string
	choice                 ForkChoice
	choiceDigest           string
}

func TestEvaluateControlledTransitionGO(t *testing.T) {
	fixture := makeRecoveryFixture(t, nil)
	inputs := fixture.controlledInputs()
	report, err := evaluate(inputs)
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	if report.Decision != decisionControlled ||
		report.RequiredRoute != routeControlled ||
		report.SelectedSHA256 != fixture.controlledDigest {
		t.Fatalf("unexpected controlled report: %#v", report)
	}
	if err := validateGateReportEnvelope(report); err != nil {
		t.Fatalf("envelope: %v", err)
	}
	if err := verifyGateReportWithInputs(report, inputs); err != nil {
		t.Fatalf("authoritative verification: %v", err)
	}

	again, err := evaluate(inputs)
	if err != nil {
		t.Fatalf("evaluate again: %v", err)
	}
	first, _ := json.Marshal(report)
	second, _ := json.Marshal(again)
	if !bytes.Equal(first, second) {
		t.Fatal("controlled evaluation is not deterministic")
	}
}

func TestEvaluateForkRegenesisGO(t *testing.T) {
	fixture := makeRecoveryFixture(t, map[string]string{
		"consensus-key-exclusive-control": custodyResultUnknown,
	})
	inputs := fixture.forkInputs()
	report, err := evaluate(inputs)
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	if report.Decision != decisionFork ||
		report.RequiredRoute != routeFork ||
		report.SelectedSHA256 != fixture.choiceDigest {
		t.Fatalf("unexpected fork report: %#v", report)
	}
	if !containsString(report.ReasonCodes, "CUSTODY_FINDING_UNKNOWN") {
		t.Fatalf("fork reasons = %v", report.ReasonCodes)
	}
	if err := validateGateReportEnvelope(report); err != nil {
		t.Fatalf("envelope: %v", err)
	}
	if err := verifyGateReportWithInputs(report, inputs); err != nil {
		t.Fatalf("authoritative verification: %v", err)
	}
}

func TestCurrentV10ForkReportNeedsOnlyScheduleAdditionMigration(t *testing.T) {
	fixture := forkFixture(t)
	for reportIndex := range fixture.reports {
		report := &fixture.reports[reportIndex]
		report.SchemaMigrations = []string{messageScheduleGenesisMigration}
		for digestIndex := range report.ModuleDigests {
			module := &report.ModuleDigests[digestIndex]
			if module.Module == "ibc" || module.Module == "transfer" {
				module.BeforeSHA256 = module.AfterSHA256
				module.Changed = false
			}
		}
		sealCompilerReport(t, report)
		fixture.reportDigests[reportIndex] = exactDigest(t, *report)
	}
	fixture.rebindReleaseToCurrentArtifacts(t)
	report, err := evaluate(fixture.forkInputs())
	if err != nil {
		t.Fatal(err)
	}
	if report.Decision != decisionFork {
		t.Fatalf("v10 no-migration decision = %q, reasons = %v", report.Decision, report.ReasonCodes)
	}
}

func TestConsensusOnlyForkRefusesUnsafeRetainedAuthorityFacts(t *testing.T) {
	for _, finding := range []string{
		"canonical-history-single",
		"old-governance-authority-safe",
		"old-sdk-operator-key-safe",
	} {
		t.Run(finding, func(t *testing.T) {
			overrides := map[string]string{
				finding: custodyResultUnknown,
			}
			if finding == "old-sdk-operator-key-safe" {
				// Keep the typed SDK assessment consistent so the
				// assessment remains valid and the profile check is
				// what closes the route.
				fixture := makeRecoveryFixture(t, nil)
				fixture.assessment.PrivilegedIdentityAssessments[0].Result =
					custodyResultUnknown
				fixture.assessment.PrivilegedIdentityAssessments[0].Disposition =
					privilegedDispositionRetire
				fixture.assessment.ProhibitedPrivilegedIdentities =
					[]string{fixture.assessment.OldValidator.SDKOperatorAddress}
				for index := range fixture.assessment.Findings {
					if fixture.assessment.Findings[index].ID == finding {
						fixture.assessment.Findings[index].Result =
							custodyResultUnknown
					}
				}
				signCustody(t, &fixture.assessment, fixture.custodySigners)
				fixture.assessmentDigest = exactDigest(t, fixture.assessment)
				inputs := EvaluationInputs{
					CustodyPolicy:         &fixture.custodyPolicy,
					CustodyPolicySHA256:   fixture.custodyPolicyDigest,
					Assessment:            fixture.assessment,
					AssessmentSHA256:      fixture.assessmentDigest,
					ForkPolicy:            &fixture.policy,
					ForkPolicySHA256:      fixture.policyDigest,
					ForkRelease:           &fixture.release,
					ForkReleaseSHA256:     fixture.releaseDigest,
					ForkChoice:            &fixture.choice,
					ForkChoiceSHA256:      fixture.choiceDigest,
					Genesis:               &fixture.genesis,
					GenesisSHA256:         fixture.genesisDigest,
					CompilerReports:       fixture.reports,
					CompilerReportSHA256s: fixture.reportDigests,
				}
				report, err := evaluate(inputs)
				if err != nil {
					t.Fatal(err)
				}
				assertNoGoReason(t, report, "FORK_POLICY_INVALID")
				return
			}
			fixture := makeRecoveryFixture(t, overrides)
			report, err := evaluate(fixture.forkInputs())
			if err != nil {
				t.Fatal(err)
			}
			assertNoGoReason(t, report, "FORK_POLICY_INVALID")
		})
	}
}

func TestPassRetirePrivilegedIdentityCannotUseControlledRoute(t *testing.T) {
	fixture := makeRecoveryFixture(t, nil)
	entry := &fixture.assessment.PrivilegedIdentityAssessments[0]
	entry.Disposition = privilegedDispositionRetire
	fixture.assessment.ProhibitedPrivilegedIdentities =
		[]string{entry.Identity}
	signCustody(t, &fixture.assessment, fixture.custodySigners)
	fixture.assessmentDigest = exactDigest(t, fixture.assessment)
	report, err := evaluate(EvaluationInputs{
		CustodyPolicy:       &fixture.custodyPolicy,
		CustodyPolicySHA256: fixture.custodyPolicyDigest,
		Assessment:          fixture.assessment,
		AssessmentSHA256:    fixture.assessmentDigest,
	})
	if err != nil {
		t.Fatal(err)
	}
	if report.RequiredRoute != routeFork ||
		!containsString(
			report.ReasonCodes,
			"PRIVILEGED_IDENTITY_RETIRE_REQUIRED",
		) {
		t.Fatalf("retire route = %q, reasons = %v", report.RequiredRoute, report.ReasonCodes)
	}
}

func TestCustodyFactsSelectFork(t *testing.T) {
	tests := []struct {
		name   string
		result string
		reason string
	}{
		{"unknown", custodyResultUnknown, "CUSTODY_FINDING_UNKNOWN"},
		{"failed", custodyResultFail, "CUSTODY_FINDING_FAILED"},
		{"missing", missingFinding, "CUSTODY_FINDING_MISSING"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fixture := makeRecoveryFixture(t, map[string]string{
				"historical-image-access-accounted": test.result,
			})
			report, err := evaluate(fixture.forkInputs())
			if err != nil {
				t.Fatalf("evaluate: %v", err)
			}
			if report.Decision != decisionFork {
				t.Fatalf("decision = %q, reasons = %v", report.Decision, report.ReasonCodes)
			}
			if !containsString(report.ReasonCodes, test.reason) {
				t.Fatalf("reasons %v do not contain %q", report.ReasonCodes, test.reason)
			}
		})
	}
}

func TestCustodyAssessmentFailsClosed(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*recoveryFixture)
	}{
		{
			name: "lying consensus address",
			mutate: func(f *recoveryFixture) {
				f.assessment.OldValidator.ConsensusAddress =
					strings.Repeat("A", cometAddressBytes*2)
			},
		},
		{
			name: "finding evidence unlinked",
			mutate: func(f *recoveryFixture) {
				digest := f.assessment.Findings[0].EvidenceSHA256
				f.assessment.Evidence = removeEvidenceDigest(
					f.assessment.Evidence,
					digest,
				)
			},
		},
		{
			name: "privileged evidence unlinked",
			mutate: func(f *recoveryFixture) {
				digest := f.assessment.
					PrivilegedIdentityAssessments[0].
					EvidenceSHA256
				f.assessment.Evidence = removeEvidenceDigest(
					f.assessment.Evidence,
					digest,
				)
			},
		},
		{
			name: "checkpoint evidence unlinked",
			mutate: func(f *recoveryFixture) {
				f.assessment.Evidence = removeEvidenceDigest(
					f.assessment.Evidence,
					f.assessment.Checkpoint.SignedCommitSHA256,
				)
			},
		},
		{
			name: "policy digest substitution",
			mutate: func(f *recoveryFixture) {
				f.assessment.SignerPolicySHA256 =
					fixtureHash("different-custody-policy")
			},
		},
		{
			name: "evaluation before checkpoint",
			mutate: func(f *recoveryFixture) {
				f.assessment.EvaluatedAt =
					f.assessment.Checkpoint.BlockTime
			},
		},
		{
			name: "review does not end at checkpoint",
			mutate: func(f *recoveryFixture) {
				f.assessment.ExposureWindow.LastReviewedHeight--
			},
		},
		{
			name: "cross-role old public keys",
			mutate: func(f *recoveryFixture) {
				f.assessment.OldValidator.NodePublicKey =
					f.assessment.OldValidator.ConsensusPublicKey
				f.assessment.OldValidator.NodeID, _ =
					nodeIDFromPublicKeyHex(
						f.assessment.OldValidator.NodePublicKey,
					)
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fixture := makeRecoveryFixture(t, nil)
			test.mutate(&fixture)
			signCustody(t, &fixture.assessment, fixture.custodySigners)
			fixture.assessmentDigest = exactDigest(t, fixture.assessment)
			inputs := fixture.controlledInputs()
			inputs.Controlled = nil
			inputs.ControlledSHA256 = ""
			report, err := evaluate(inputs)
			if err != nil {
				t.Fatalf("evaluate: %v", err)
			}
			assertNoGoReason(t, report, "CUSTODY_ASSESSMENT_INVALID")
		})
	}
}

func TestCustodyAndControlledApprovalsRequirePinnedSignerPolicies(t *testing.T) {
	t.Run("custody", func(t *testing.T) {
		fixture := makeRecoveryFixture(t, nil)
		rogue := signerSet(requiredCustodyRoles, 180)
		signCustody(t, &fixture.assessment, rogue)
		fixture.assessmentDigest = exactDigest(t, fixture.assessment)
		report, err := evaluate(EvaluationInputs{
			CustodyPolicy:       &fixture.custodyPolicy,
			CustodyPolicySHA256: fixture.custodyPolicyDigest,
			Assessment:          fixture.assessment,
			AssessmentSHA256:    fixture.assessmentDigest,
		})
		if err != nil {
			t.Fatalf("evaluate: %v", err)
		}
		assertNoGoReason(t, report, "CUSTODY_ASSESSMENT_INVALID")
	})

	t.Run("controlled", func(t *testing.T) {
		fixture := makeRecoveryFixture(t, nil)
		rogue := signerSet(requiredControlledRoles, 190)
		signControlled(t, &fixture.controlled, rogue)
		fixture.controlledDigest = exactDigest(t, fixture.controlled)
		report, err := evaluate(fixture.controlledInputs())
		if err != nil {
			t.Fatalf("evaluate: %v", err)
		}
		assertNoGoReason(t, report, "CONTROLLED_PLAN_INVALID")
	})
}

func TestControlledTransitionFailsClosed(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*recoveryFixture)
	}{
		{
			name: "two thirds equality",
			mutate: func(f *recoveryFixture) {
				f.controlled.ConsentingPower = "2"
				f.controlled.TotalBondedPower = "3"
			},
		},
		{
			name: "admission closed",
			mutate: func(f *recoveryFixture) {
				f.controlled.AdmissionState = "CLOSED"
			},
		},
		{
			name: "activation B plus one",
			mutate: func(f *recoveryFixture) {
				f.controlled.ExpectedActivationHeight =
					f.controlled.BondTransactionHeight + 1
			},
		},
		{
			name: "bond before checkpoint",
			mutate: func(f *recoveryFixture) {
				f.controlled.BondTransactionHeight =
					f.controlled.Checkpoint.Height
				f.controlled.ExpectedActivationHeight =
					f.controlled.BondTransactionHeight + 2
			},
		},
		{
			name: "incomplete pagination",
			mutate: func(f *recoveryFixture) {
				f.controlled.StakeInventory.Redelegations.Complete = false
				f.controlled.StakeInventory.Redelegations.NextKey = "more"
			},
		},
		{
			name: "lying consensus address",
			mutate: func(f *recoveryFixture) {
				f.controlled.NewValidator.ConsensusAddress =
					strings.Repeat("B", cometAddressBytes*2)
			},
		},
		{
			name: "old node key reused as consensus key",
			mutate: func(f *recoveryFixture) {
				f.controlled.NewValidator.ConsensusPublicKey =
					f.assessment.OldValidator.NodePublicKey
				f.controlled.NewValidator.ConsensusAddress, _ =
					consensusAddressFromPublicKeyHex(
						f.controlled.NewValidator.ConsensusPublicKey,
					)
			},
		},
		{
			name: "old node digest reused as validator digest",
			mutate: func(f *recoveryFixture) {
				f.controlled.NewValidator.ValidatorKeySHA256 =
					f.assessment.OldValidator.NodeKeySHA256
			},
		},
		{
			name: "new key digests not distinct",
			mutate: func(f *recoveryFixture) {
				f.controlled.NewValidator.NodeKeySHA256 =
					f.controlled.NewValidator.ValidatorKeySHA256
			},
		},
		{
			name: "binary evidence unlinked",
			mutate: func(f *recoveryFixture) {
				f.controlled.Evidence = removeEvidenceDigest(
					f.controlled.Evidence,
					f.controlled.BinarySHA256,
				)
			},
		},
		{
			name: "new signing state evidence unlinked",
			mutate: func(f *recoveryFixture) {
				f.controlled.Evidence = removeEvidenceDigest(
					f.controlled.Evidence,
					f.controlled.NewValidator.SigningStateSHA256,
				)
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fixture := makeRecoveryFixture(t, nil)
			test.mutate(&fixture)
			signControlled(
				t,
				&fixture.controlled,
				fixture.controlledSigners,
			)
			fixture.controlledDigest = exactDigest(t, fixture.controlled)
			report, err := evaluate(fixture.controlledInputs())
			if err != nil {
				t.Fatalf("evaluate: %v", err)
			}
			assertNoGoReason(t, report, "CONTROLLED_PLAN_INVALID")
		})
	}
}

func TestForkReleaseRequiresAuditedConsensusKeyOnlyProfile(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*recoveryFixture)
	}{
		{
			name: "unsupported rewrite profile",
			mutate: func(f *recoveryFixture) {
				f.release.RewriteProfile = "full-identity"
			},
		},
		{
			name: "fresh operator unsupported",
			mutate: func(f *recoveryFixture) {
				f.release.NewValidators[0].SDKOperatorAddress =
					operatorAddress(t, 0x91)
			},
		},
		{
			name: "old consensus key",
			mutate: func(f *recoveryFixture) {
				f.release.NewValidators[0].ConsensusPublicKey =
					f.assessment.OldValidator.ConsensusPublicKey
				f.release.NewValidators[0].ConsensusAddress =
					f.assessment.OldValidator.ConsensusAddress
			},
		},
		{
			name: "release artifact evidence unlinked",
			mutate: func(f *recoveryFixture) {
				f.release.Evidence = removeEvidenceDigest(
					f.release.Evidence,
					f.release.BinarySHA256,
				)
			},
		},
		{
			name: "release key evidence unlinked",
			mutate: func(f *recoveryFixture) {
				f.release.Evidence = removeEvidenceDigest(
					f.release.Evidence,
					f.release.NewValidators[0].NodeKeySHA256,
				)
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fixture := forkFixture(t)
			test.mutate(&fixture)
			fixture.resignReleaseAndChoice(t, true)
			report, err := evaluate(fixture.forkInputs())
			if err != nil {
				t.Fatalf("evaluate: %v", err)
			}
			assertNoGoReason(t, report, "FORK_RELEASE_INVALID")
		})
	}
}

func TestGenesisReproductionSignaturesFailClosed(t *testing.T) {
	t.Run("unsigned", func(t *testing.T) {
		fixture := forkFixture(t)
		fixture.release.GenesisReproductions[0].Signature =
			strings.Repeat("0", ed25519.SignatureSize*2)
		fixture.resignReleaseAndChoice(t, false)
		report, err := evaluate(fixture.forkInputs())
		if err != nil {
			t.Fatalf("evaluate: %v", err)
		}
		assertNoGoReason(t, report, "FORK_RELEASE_INVALID")
	})

	t.Run("same key", func(t *testing.T) {
		fixture := forkFixture(t)
		first := fixture.release.GenesisReproductions[0]
		second := &fixture.release.GenesisReproductions[1]
		second.PublicKey = first.PublicKey
		statement, err := genesisReproductionStatement(
			fixture.release,
			*second,
		)
		if err != nil {
			t.Fatal(err)
		}
		statementBytes, _ := hex.DecodeString(statement)
		second.StatementSHA256 = statement
		second.Signature = hex.EncodeToString(ed25519.Sign(
			fixture.reproducerSigners[0].privateKey,
			statementBytes,
		))
		fixture.resignReleaseAndChoice(t, false)
		report, err := evaluate(fixture.forkInputs())
		if err != nil {
			t.Fatalf("evaluate: %v", err)
		}
		assertNoGoReason(t, report, "FORK_RELEASE_INVALID")
	})

	t.Run("report digest not evidence linked", func(t *testing.T) {
		fixture := forkFixture(t)
		fixture.release.Evidence = removeEvidenceDigest(
			fixture.release.Evidence,
			fixture.reportDigests[0],
		)
		fixture.resignReleaseAndChoice(t, true)
		report, err := evaluate(fixture.forkInputs())
		if err != nil {
			t.Fatalf("evaluate: %v", err)
		}
		assertNoGoReason(t, report, "FORK_RELEASE_INVALID")
	})
}

func TestExactForkArtifactsFailClosed(t *testing.T) {
	t.Run("genesis file mismatch", func(t *testing.T) {
		fixture := forkFixture(t)
		fixture.genesis.AppVersion = "v-mismatched"
		fixture.genesisDigest = exactDigest(t, fixture.genesis)
		report, err := evaluate(fixture.forkInputs())
		if err != nil {
			t.Fatalf("evaluate: %v", err)
		}
		assertNoGoReason(t, report, "FORK_ARTIFACTS_INVALID")
	})

	t.Run("compiler report output mismatch", func(t *testing.T) {
		fixture := forkFixture(t)
		fixture.reports[0].OutputGenesisSHA256 =
			fixtureHash("different-genesis")
		sealCompilerReport(t, &fixture.reports[0])
		fixture.reportDigests[0] =
			exactDigest(t, fixture.reports[0])
		fixture.rebindReleaseToCurrentArtifacts(t)
		report, err := evaluate(fixture.forkInputs())
		if err != nil {
			t.Fatalf("evaluate: %v", err)
		}
		assertNoGoReason(t, report, "FORK_ARTIFACTS_INVALID")
	})

	t.Run("random report file digest", func(t *testing.T) {
		fixture := forkFixture(t)
		fixture.reportDigests[0] = fixtureHash("random-label")
		fixture.rebindReleaseToCurrentArtifacts(t)
		report, err := evaluate(fixture.forkInputs())
		if err != nil {
			t.Fatalf("evaluate: %v", err)
		}
		assertNoGoReason(t, report, "FORK_ARTIFACTS_INVALID")
	})

	t.Run("reports disagree", func(t *testing.T) {
		fixture := forkFixture(t)
		fixture.reports[0].ModuleDigests[0].BeforeSHA256 =
			fixtureHash("different-before")
		fixture.reports[0].ModuleDigests[0].Changed =
			fixture.reports[0].ModuleDigests[0].BeforeSHA256 !=
				fixture.reports[0].ModuleDigests[0].AfterSHA256
		sealCompilerReport(t, &fixture.reports[0])
		fixture.reportDigests[0] =
			exactDigest(t, fixture.reports[0])
		fixture.rebindReleaseToCurrentArtifacts(t)
		report, err := evaluate(fixture.forkInputs())
		if err != nil {
			t.Fatalf("evaluate: %v", err)
		}
		assertNoGoReason(t, report, "FORK_ARTIFACTS_INVALID")
	})

	t.Run("message schedule addition migration omitted", func(t *testing.T) {
		fixture := forkFixture(t)
		for index := range fixture.reports {
			report := &fixture.reports[index]
			report.SchemaMigrations = report.SchemaMigrations[:len(report.SchemaMigrations)-1]
			sealCompilerReport(t, report)
			fixture.reportDigests[index] = exactDigest(t, *report)
		}
		fixture.rebindReleaseToCurrentArtifacts(t)
		report, err := evaluate(fixture.forkInputs())
		if err != nil {
			t.Fatalf("evaluate: %v", err)
		}
		assertNoGoReason(t, report, "FORK_ARTIFACTS_INVALID")
	})

	t.Run("message schedule absence sentinel forged", func(t *testing.T) {
		fixture := forkFixture(t)
		for reportIndex := range fixture.reports {
			report := &fixture.reports[reportIndex]
			for digestIndex := range report.ModuleDigests {
				module := &report.ModuleDigests[digestIndex]
				if module.Module == "message_schedule" {
					module.BeforeSHA256 = fixtureHash("forged-message-schedule-source")
					module.Changed = module.BeforeSHA256 != module.AfterSHA256
				}
			}
			sealCompilerReport(t, report)
			fixture.reportDigests[reportIndex] = exactDigest(t, *report)
		}
		fixture.rebindReleaseToCurrentArtifacts(t)
		report, err := evaluate(fixture.forkInputs())
		if err != nil {
			t.Fatalf("evaluate: %v", err)
		}
		assertNoGoReason(t, report, "FORK_ARTIFACTS_INVALID")
	})
}

func TestForkGenesisHostileStateIsNoGoEvenWhenReSigned(t *testing.T) {
	tests := []struct {
		name   string
		module string
		mutate func(map[string]any, *recoveryFixture)
	}{
		{
			name:   "not halted",
			module: "emergency",
			mutate: func(value map[string]any, _ *recoveryFixture) {
				value["status"] = "normal"
			},
		},
		{
			name:   "pending upgrade",
			module: "upgrade",
			mutate: func(value map[string]any, _ *recoveryFixture) {
				value["plan"] = map[string]any{"name": "hostile"}
			},
		},
		{
			name:   "genesis transaction",
			module: "genutil",
			mutate: func(value map[string]any, _ *recoveryFixture) {
				value["gen_txs"] = []any{"unexpected"}
			},
		},
		{
			name:   "live IBC channel",
			module: "ibc",
			mutate: func(value map[string]any, _ *recoveryFixture) {
				channel := value["channel_genesis"].(map[string]any)
				channel["channels"] = []any{map[string]any{"id": "channel-0"}}
			},
		},
		{
			name:   "jailed retained validator",
			module: "staking",
			mutate: func(value map[string]any, _ *recoveryFixture) {
				validators := value["validators"].([]any)
				validators[0].(map[string]any)["jailed"] = true
			},
		},
		{
			name:   "old consensus key remains",
			module: "slashing",
			mutate: func(value map[string]any, f *recoveryFixture) {
				oldKey, _ := hex.DecodeString(
					f.assessment.OldValidator.ConsensusPublicKey,
				)
				value["stale_key"] =
					base64.StdEncoding.EncodeToString(oldKey)
			},
		},
		{
			name:   "message schedule is not the fresh closed default",
			module: "message_schedule",
			mutate: func(value map[string]any, _ *recoveryFixture) {
				value["next_schedule_id"] = json.Number("2")
			},
		},
		{
			name:   "retired scheduler module account has coins",
			module: "bank",
			mutate: func(value map[string]any, _ *recoveryFixture) {
				value["balances"] = []any{map[string]any{
					"address": schedulerModuleAccountAddress("schedule"),
					"coins": []any{map[string]any{
						"denom": "uretired", "amount": "1",
					}},
				}}
			},
		},
		{
			name:   "fresh scheduler module account has coins",
			module: "bank",
			mutate: func(value map[string]any, _ *recoveryFixture) {
				value["balances"] = []any{map[string]any{
					"address": schedulerModuleAccountAddress("message_schedule"),
					"coins": []any{map[string]any{
						"denom": "ufresh", "amount": "1",
					}},
				}}
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fixture := forkFixture(t)
			mutateGenesisModule(
				t,
				&fixture,
				test.module,
				func(value map[string]any) {
					test.mutate(value, &fixture)
				},
			)
			fixture.resealArtifactsAfterGenesisChange(t)
			report, err := evaluate(fixture.forkInputs())
			if err != nil {
				t.Fatalf("evaluate: %v", err)
			}
			assertNoGoReason(t, report, "FORK_ARTIFACTS_INVALID")
		})
	}
}

func TestCompilerModuleDigestsMustMatchExactGenesis(t *testing.T) {
	fixture := forkFixture(t)
	for index := range fixture.reports {
		for moduleIndex := range fixture.reports[index].ModuleDigests {
			if fixture.reports[index].ModuleDigests[moduleIndex].Module == "bank" {
				fixture.reports[index].ModuleDigests[moduleIndex].AfterSHA256 =
					fixtureHash("random-bank-after")
				fixture.reports[index].ModuleDigests[moduleIndex].BeforeSHA256 =
					fixture.reports[index].ModuleDigests[moduleIndex].AfterSHA256
				fixture.reports[index].ModuleDigests[moduleIndex].Changed = false
			}
		}
		sealCompilerReport(t, &fixture.reports[index])
		fixture.reportDigests[index] =
			exactDigest(t, fixture.reports[index])
	}
	fixture.rebindReleaseToCurrentArtifacts(t)
	report, err := evaluate(fixture.forkInputs())
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	assertNoGoReason(t, report, "FORK_ARTIFACTS_INVALID")
}

func TestSelfHashedForgedGORequiresAuthoritativeReevaluation(t *testing.T) {
	fixture := makeRecoveryFixture(t, nil)
	inputs := fixture.controlledInputs()
	valid, err := evaluate(inputs)
	if err != nil {
		t.Fatal(err)
	}
	forged := valid
	forged.InputDigests.ControlledSHA256 = fixtureHash("forged-plan")
	forged.SelectedSHA256 = forged.InputDigests.ControlledSHA256
	forged, err = sealGateReport(forged)
	if err != nil {
		t.Fatal(err)
	}
	if err := validateGateReportEnvelope(forged); err != nil {
		t.Fatalf("forged report should be a structurally valid envelope: %v", err)
	}
	if err := verifyGateReportWithInputs(forged, inputs); err == nil {
		t.Fatal("self-hashed forged GO passed authoritative re-evaluation")
	}
}

func TestInactiveRouteInputsAreRejected(t *testing.T) {
	t.Run("controlled with fork input", func(t *testing.T) {
		fixture := makeRecoveryFixture(t, nil)
		inputs := fixture.controlledInputs()
		inputs.ForkPolicy = &fixture.policy
		inputs.ForkPolicySHA256 = fixture.policyDigest
		report, err := evaluate(inputs)
		if err != nil {
			t.Fatal(err)
		}
		assertNoGoReason(t, report, "INACTIVE_ROUTE_INPUTS_PRESENT")
	})

	t.Run("fork with controlled input", func(t *testing.T) {
		fixture := forkFixture(t)
		inputs := fixture.forkInputs()
		inputs.Controlled = &fixture.controlled
		inputs.ControlledSHA256 = fixture.controlledDigest
		report, err := evaluate(inputs)
		if err != nil {
			t.Fatal(err)
		}
		assertNoGoReason(t, report, "INACTIVE_ROUTE_INPUTS_PRESENT")
	})
}

func TestExactInputDigestSubstitutionIsRejected(t *testing.T) {
	fixture := makeRecoveryFixture(t, nil)
	inputs := fixture.controlledInputs()
	inputs.ControlledSHA256 = fixtureHash("wrong-plan-pin")
	report, err := evaluate(inputs)
	if err != nil {
		t.Fatal(err)
	}
	assertNoGoReason(t, report, "CONTROLLED_PLAN_INVALID")
}

func TestDecodeRejectsNonCanonicalUnknownSecretAndDuplicateFields(t *testing.T) {
	fixture := makeRecoveryFixture(t, nil)
	canonical, _ := json.Marshal(fixture.assessment)
	if _, err := decodeExactJSON[CustodyAssessment](
		append(canonical, '\n'),
		"custody assessment",
	); err == nil {
		t.Fatal("trailing newline was accepted")
	}

	unknown := append([]byte{}, canonical[:len(canonical)-1]...)
	unknown = append(unknown, []byte(`,"unknown":"value"}`)...)
	if _, err := decodeExactJSON[CustodyAssessment](
		unknown,
		"custody assessment",
	); err == nil {
		t.Fatal("unknown field was accepted")
	}

	secretValue := "never-echo-this"
	secret := append([]byte{}, canonical[:len(canonical)-1]...)
	secret = append(secret, []byte(`,"mnemonic":"`+secretValue+`"}`)...)
	_, err := decodeExactJSON[CustodyAssessment](
		secret,
		"custody assessment",
	)
	if err == nil || strings.Contains(err.Error(), secretValue) {
		t.Fatal("secret-bearing field was accepted or echoed")
	}

	genesis, _ := json.Marshal(fixture.genesis)
	if _, err := decodeForkGenesis(genesis); err != nil {
		t.Fatalf("compiler-ordered AppGenesis rejected: %v", err)
	}
	duplicate := bytes.Replace(
		genesis,
		[]byte(`"app_name":"zeroned"`),
		[]byte(`"app_name":"zeroned","app_name":"zeroned"`),
		1,
	)
	if _, err := decodeForkGenesis(duplicate); err == nil {
		t.Fatal("duplicate AppGenesis key was accepted")
	}
}

func TestPinnedReaderRejectsSymlinkMismatchedPinAndOversize(t *testing.T) {
	directory := t.TempDir()
	target := filepath.Join(directory, "assessment.json")
	if err := os.WriteFile(target, []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(directory, "assessment-link.json")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	if _, _, err := readPinnedRegularFile(
		link,
		digestBytes([]byte("{}")),
		"custody assessment",
	); err == nil {
		t.Fatal("symlink input was accepted")
	}
	if _, _, err := readPinnedRegularFile(
		target,
		fixtureHash("wrong-pin"),
		"custody assessment",
	); err == nil {
		t.Fatal("mismatched pin was accepted")
	}

	oversize := filepath.Join(directory, "oversize.json")
	content := bytes.Repeat([]byte("x"), maxInputBytes+1)
	if err := os.WriteFile(oversize, content, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := readPinnedRegularFile(
		oversize,
		digestBytes(content),
		"custody assessment",
	); err == nil {
		t.Fatal("oversize input was accepted")
	}
}

func TestRunEvaluateControlledUsesPinnedPolicies(t *testing.T) {
	fixture := makeRecoveryFixture(t, nil)
	directory := t.TempDir()
	custodyPolicyPath := writeCanonical(
		t,
		directory,
		"custody-policy.json",
		fixture.custodyPolicy,
	)
	assessmentPath := writeCanonical(
		t,
		directory,
		"assessment.json",
		fixture.assessment,
	)
	controlledPolicyPath := writeCanonical(
		t,
		directory,
		"controlled-policy.json",
		fixture.controlledPolicy,
	)
	controlledPath := writeCanonical(
		t,
		directory,
		"controlled.json",
		fixture.controlled,
	)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := run([]string{
		"evaluate",
		"--chain-id", fixture.assessment.ChainID,
		"--incident-id", fixture.assessment.IncidentID,
		"--custody-policy", custodyPolicyPath,
		"--custody-policy-sha256", fixture.custodyPolicyDigest,
		"--assessment", assessmentPath,
		"--assessment-sha256", fixture.assessmentDigest,
		"--controlled-policy", controlledPolicyPath,
		"--controlled-policy-sha256", fixture.controlledPolicyDigest,
		"--controlled", controlledPath,
		"--controlled-sha256", fixture.controlledDigest,
	}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("exit = %d, stderr = %s", exitCode, stderr.String())
	}
	var report GateReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	canonical, _ := json.Marshal(report)
	if !bytes.Equal(stdout.Bytes(), canonical) {
		t.Fatal("CLI output is not exact compact canonical JSON")
	}
	if err := verifyGateReportWithInputs(
		report,
		fixture.controlledInputs(),
	); err != nil {
		t.Fatalf("CLI report verification: %v", err)
	}
}

func TestRunEvaluateForkConsumesExactGenesisAndReports(t *testing.T) {
	fixture := forkFixture(t)
	directory := t.TempDir()
	paths := map[string]string{
		"custody-policy": writeCanonical(
			t, directory, "custody-policy.json", fixture.custodyPolicy,
		),
		"assessment": writeCanonical(
			t, directory, "assessment.json", fixture.assessment,
		),
		"fork-policy": writeCanonical(
			t, directory, "fork-policy.json", fixture.policy,
		),
		"fork-release": writeCanonical(
			t, directory, "fork-release.json", fixture.release,
		),
		"fork-choice": writeCanonical(
			t, directory, "fork-choice.json", fixture.choice,
		),
		"genesis": writeCanonical(
			t, directory, "genesis.json", fixture.genesis,
		),
		"report-a": writeCanonical(
			t, directory, "report-a.json", fixture.reports[0],
		),
		"report-b": writeCanonical(
			t, directory, "report-b.json", fixture.reports[1],
		),
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := run([]string{
		"evaluate",
		"--chain-id", fixture.assessment.ChainID,
		"--incident-id", fixture.assessment.IncidentID,
		"--custody-policy", paths["custody-policy"],
		"--custody-policy-sha256", fixture.custodyPolicyDigest,
		"--assessment", paths["assessment"],
		"--assessment-sha256", fixture.assessmentDigest,
		"--fork-policy", paths["fork-policy"],
		"--fork-policy-sha256", fixture.policyDigest,
		"--fork-release", paths["fork-release"],
		"--fork-release-sha256", fixture.releaseDigest,
		"--fork-choice", paths["fork-choice"],
		"--fork-choice-sha256", fixture.choiceDigest,
		"--genesis", paths["genesis"],
		"--genesis-sha256", fixture.genesisDigest,
		"--compiler-report-a", paths["report-a"],
		"--compiler-report-a-sha256", fixture.reportDigests[0],
		"--compiler-report-b", paths["report-b"],
		"--compiler-report-b-sha256", fixture.reportDigests[1],
	}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("exit = %d, stderr = %s", exitCode, stderr.String())
	}
	var report GateReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	if report.Decision != decisionFork {
		t.Fatalf("CLI decision = %q, reasons = %v", report.Decision, report.ReasonCodes)
	}
	if err := verifyGateReportWithInputs(
		report,
		fixture.forkInputs(),
	); err != nil {
		t.Fatalf("CLI report verification: %v", err)
	}
}

func makeRecoveryFixture(
	t *testing.T,
	findingOverrides map[string]string,
) recoveryFixture {
	t.Helper()
	custodySigners := signerSet(requiredCustodyRoles, 1)
	controlledSigners := signerSet(requiredControlledRoles, 10)
	forkSigners := signerSet(requiredForkRoles, 30)
	reproducerSigners := []testSigner{
		testReproducer("reproducer-a", "reproducer-domain-a", 60),
		testReproducer("reproducer-b", "reproducer-domain-b", 61),
	}

	oldOperator := operatorAddress(t, 0x11)
	oldValidator := validatorIdentity(t, 90, 91, oldOperator, "old")
	custodyPolicy := signerPolicy(
		signerPurposeCustody,
		"incident-2026-001",
		"zerone-1",
		"",
		requiredCustodyRoles,
		custodySigners,
	)
	custodyPolicyDigest := exactDigest(t, custodyPolicy)

	assessment := CustodyAssessment{
		Schema:             custodyAssessmentSchema,
		AssessmentID:       "assessment-2026-001",
		SignerPolicySHA256: custodyPolicyDigest,
		IncidentID:         "incident-2026-001",
		ChainID:            "zerone-1",
		EvaluatedAt:        "2026-07-30T12:00:00Z",
		Checkpoint: Checkpoint{
			Height:             100,
			BlockTime:          "2026-07-30T11:59:00Z",
			BlockIDSHA256:      fixtureHash("checkpoint-block"),
			AppHashSHA256:      fixtureHash("checkpoint-app"),
			SignedCommitSHA256: fixtureHash("checkpoint-commit"),
			ValidatorSetSHA256: fixtureHash("checkpoint-validator-set"),
		},
		ExposureWindow: ExposureWindow{
			FirstPossiblyExposedHeight: 1,
			LastReviewedHeight:         100,
		},
		OldValidator: oldValidator,
		PrivilegedIdentityAssessments: []PrivilegedIdentityAssessment{
			{
				Kind:           privilegedKindSDKOperator,
				Identity:       oldOperator,
				Result:         custodyResultPass,
				EvidenceSHA256: fixtureHash("old-sdk-operator-assessment"),
				Disposition:    privilegedDispositionRetain,
			},
		},
		Findings:                       []CustodyFinding{},
		Evidence:                       []Evidence{},
		ProhibitedConsensusPublicKeys:  []string{oldValidator.ConsensusPublicKey},
		ProhibitedPrivilegedIdentities: []string{},
		Approvals:                      []Approval{},
	}
	for _, id := range requiredCustodyFindings {
		result := custodyResultPass
		if override, present := findingOverrides[id]; present {
			if override == missingFinding {
				continue
			}
			result = override
		}
		assessment.Findings = append(assessment.Findings, CustodyFinding{
			ID:             id,
			Result:         result,
			EvidenceSHA256: fixtureHash("finding-" + id),
		})
	}
	assessment.Evidence = custodyEvidence(assessment)
	signCustody(t, &assessment, custodySigners)
	assessmentDigest := exactDigest(t, assessment)

	controlledPolicy := signerPolicy(
		signerPurposeControlled,
		assessment.IncidentID,
		assessment.ChainID,
		assessmentDigest,
		requiredControlledRoles,
		controlledSigners,
	)
	controlledPolicyDigest := exactDigest(t, controlledPolicy)
	controlled := ControlledTransition{
		Schema:                    controlledTransitionSchema,
		PlanID:                    "controlled-2026-001",
		AssessmentSHA256:          assessmentDigest,
		SignerPolicySHA256:        controlledPolicyDigest,
		IncidentID:                assessment.IncidentID,
		ChainID:                   assessment.ChainID,
		Checkpoint:                assessment.Checkpoint,
		AdmissionState:            requiredAdmissionState,
		BondTransactionHeight:     110,
		ExpectedActivationHeight:  112,
		ValidatorUpdateWaitBlocks: requiredValidatorUpdateWait,
		ConsentingPower:           "67",
		TotalBondedPower:          "100",
		PowerSnapshotSHA256:       fixtureHash("power-snapshot"),
		StakeInventory: StakeInventory{
			Delegations: PaginatedInventory{
				PageSHA256s: []string{fixtureHash("delegations-page-1")},
				NextKey:     "",
				Complete:    true,
			},
			Unbondings: PaginatedInventory{
				PageSHA256s: []string{fixtureHash("unbondings-page-1")},
				NextKey:     "",
				Complete:    true,
			},
			Redelegations: PaginatedInventory{
				PageSHA256s: []string{fixtureHash("redelegations-page-1")},
				NextKey:     "",
				Complete:    true,
			},
		},
		NewValidator:      validatorIdentity(t, 100, 101, oldOperator, "controlled-new"),
		BinarySHA256:      fixtureHash("controlled-binary"),
		ImageSHA256:       fixtureHash("controlled-image"),
		ProvenanceSHA256:  fixtureHash("controlled-provenance"),
		SBOMSHA256:        fixtureHash("controlled-sbom"),
		RehearsalSHA256:   fixtureHash("controlled-rehearsal"),
		TopologySHA256:    fixtureHash("controlled-topology"),
		JournalHeadSHA256: fixtureHash("controlled-journal"),
		Evidence:          []Evidence{},
		Approvals:         []Approval{},
	}
	controlled.Evidence = controlledEvidence(controlled)
	signControlled(t, &controlled, controlledSigners)
	controlledDigest := exactDigest(t, controlled)

	policy := ForkPolicy{
		Schema:                         forkPolicySchema,
		PolicyID:                       "fork-policy-2026-001",
		AssessmentSHA256:               assessmentDigest,
		IncidentID:                     assessment.IncidentID,
		OldChainID:                     assessment.ChainID,
		MinimumApprovals:               uint64(len(requiredForkRoles)),
		MinimumDistinctIdentities:      uint64(len(requiredForkRoles)),
		MinimumDistinctControlDomains:  uint64(len(requiredForkRoles)),
		RequiredRoles:                  append([]string{}, requiredForkRoles...),
		Signers:                        trustedSigners(forkSigners),
		IndependentReproducers:         trustedReproducers(reproducerSigners),
		ProhibitedConsensusPublicKeys:  append([]string{}, assessment.ProhibitedConsensusPublicKeys...),
		ProhibitedPrivilegedIdentities: append([]string{}, assessment.ProhibitedPrivilegedIdentities...),
	}
	policyDigest := exactDigest(t, policy)

	forkValidator := validatorIdentity(t, 110, 111, oldOperator, "fork-new")
	genesis := makeForkGenesis(
		t,
		forkValidator,
		assessment.Checkpoint.Height+1,
		"zerone-2",
	)
	genesisDigest := exactDigest(t, genesis)
	release := ForkRelease{
		Schema:                      forkReleaseSchema,
		ReleaseID:                   "fork-release-2026-001",
		AssessmentSHA256:            assessmentDigest,
		ForkPolicySHA256:            policyDigest,
		IncidentID:                  assessment.IncidentID,
		OldChainID:                  assessment.ChainID,
		NewChainID:                  "zerone-2",
		Checkpoint:                  assessment.Checkpoint,
		InitialHeight:               assessment.Checkpoint.Height + 1,
		RewriteProfile:              rewriteProfileConsensusOnly,
		SourceExportSHA256:          fixtureHash("source-export"),
		RewriteToolSHA256:           fixtureHash("rewrite-tool"),
		RewritePolicyFileSHA256:     fixtureHash("rewrite-policy-file"),
		RewritePolicySelfSHA256:     fixtureHash("rewrite-policy-self"),
		GenesisSHA256:               genesisDigest,
		GenesisReproductions:        []GenesisReproduction{},
		NewValidators:               []ValidatorIdentity{forkValidator},
		RetiredPrivilegedIdentities: []string{},
		SupplyReconciliationSHA256:  fixtureHash("supply-reconciliation"),
		IBCReconciliationSHA256:     fixtureHash("ibc-reconciliation"),
		ModuleReconciliationSHA256:  fixtureHash("module-reconciliation"),
		BinarySHA256:                fixtureHash("fork-binary"),
		ImageSHA256:                 fixtureHash("fork-image"),
		ProvenanceSHA256:            fixtureHash("fork-provenance"),
		SBOMSHA256:                  fixtureHash("fork-sbom"),
		RehearsalSHA256:             fixtureHash("fork-rehearsal"),
		TopologySHA256:              fixtureHash("fork-topology"),
		JournalHeadSHA256:           fixtureHash("fork-journal"),
		Evidence:                    []Evidence{},
		Approvals:                   []Approval{},
	}
	reports := []ForkGenesisReport{
		makeCompilerReport(
			t,
			reproducerSigners[0],
			release,
			assessment,
			assessmentDigest,
			policyDigest,
			genesis,
		),
		makeCompilerReport(
			t,
			reproducerSigners[1],
			release,
			assessment,
			assessmentDigest,
			policyDigest,
			genesis,
		),
	}
	reportDigests := []string{
		exactDigest(t, reports[0]),
		exactDigest(t, reports[1]),
	}
	release.GenesisReproductions = makeGenesisReproductions(
		reproducerSigners,
		genesisDigest,
		reportDigests,
	)
	release.Evidence = releaseEvidence(release)
	signGenesisReproductions(t, &release, reproducerSigners)
	signForkRelease(t, &release, forkSigners)
	releaseDigest := exactDigest(t, release)

	choice := ForkChoice{
		Schema:            forkChoiceSchema,
		ChoiceID:          "fork-choice-2026-001",
		AssessmentSHA256:  assessmentDigest,
		ForkPolicySHA256:  policyDigest,
		ForkReleaseSHA256: releaseDigest,
		IncidentID:        assessment.IncidentID,
		OldChainID:        assessment.ChainID,
		NewChainID:        release.NewChainID,
		ReasonCode:        requiredForkReason,
		Approvals:         []Approval{},
	}
	signForkChoice(t, &choice, forkSigners)
	choiceDigest := exactDigest(t, choice)

	return recoveryFixture{
		custodySigners:         custodySigners,
		controlledSigners:      controlledSigners,
		forkSigners:            forkSigners,
		reproducerSigners:      reproducerSigners,
		custodyPolicy:          custodyPolicy,
		custodyPolicyDigest:    custodyPolicyDigest,
		assessment:             assessment,
		assessmentDigest:       assessmentDigest,
		controlledPolicy:       controlledPolicy,
		controlledPolicyDigest: controlledPolicyDigest,
		controlled:             controlled,
		controlledDigest:       controlledDigest,
		policy:                 policy,
		policyDigest:           policyDigest,
		genesis:                genesis,
		genesisDigest:          genesisDigest,
		reports:                reports,
		reportDigests:          reportDigests,
		release:                release,
		releaseDigest:          releaseDigest,
		choice:                 choice,
		choiceDigest:           choiceDigest,
	}
}

func forkFixture(t *testing.T) recoveryFixture {
	t.Helper()
	return makeRecoveryFixture(t, map[string]string{
		"consensus-key-exclusive-control": custodyResultUnknown,
	})
}

func (fixture recoveryFixture) controlledInputs() EvaluationInputs {
	return EvaluationInputs{
		CustodyPolicy:          &fixture.custodyPolicy,
		CustodyPolicySHA256:    fixture.custodyPolicyDigest,
		Assessment:             fixture.assessment,
		AssessmentSHA256:       fixture.assessmentDigest,
		ControlledPolicy:       &fixture.controlledPolicy,
		ControlledPolicySHA256: fixture.controlledPolicyDigest,
		Controlled:             &fixture.controlled,
		ControlledSHA256:       fixture.controlledDigest,
	}
}

func (fixture recoveryFixture) forkInputs() EvaluationInputs {
	return EvaluationInputs{
		CustodyPolicy:         &fixture.custodyPolicy,
		CustodyPolicySHA256:   fixture.custodyPolicyDigest,
		Assessment:            fixture.assessment,
		AssessmentSHA256:      fixture.assessmentDigest,
		ForkPolicy:            &fixture.policy,
		ForkPolicySHA256:      fixture.policyDigest,
		ForkRelease:           &fixture.release,
		ForkReleaseSHA256:     fixture.releaseDigest,
		ForkChoice:            &fixture.choice,
		ForkChoiceSHA256:      fixture.choiceDigest,
		Genesis:               &fixture.genesis,
		GenesisSHA256:         fixture.genesisDigest,
		CompilerReports:       append([]ForkGenesisReport{}, fixture.reports...),
		CompilerReportSHA256s: append([]string{}, fixture.reportDigests...),
	}
}

func (fixture *recoveryFixture) resignReleaseAndChoice(
	t *testing.T,
	resignReproductions bool,
) {
	t.Helper()
	if resignReproductions {
		signGenesisReproductions(
			t,
			&fixture.release,
			fixture.reproducerSigners,
		)
	}
	signForkRelease(t, &fixture.release, fixture.forkSigners)
	fixture.releaseDigest = exactDigest(t, fixture.release)
	fixture.choice.ForkReleaseSHA256 = fixture.releaseDigest
	fixture.choice.NewChainID = fixture.release.NewChainID
	signForkChoice(t, &fixture.choice, fixture.forkSigners)
	fixture.choiceDigest = exactDigest(t, fixture.choice)
}

func (fixture *recoveryFixture) rebindReleaseToCurrentArtifacts(
	t *testing.T,
) {
	t.Helper()
	fixture.release.GenesisSHA256 = fixture.genesisDigest
	fixture.release.GenesisReproductions = makeGenesisReproductions(
		fixture.reproducerSigners,
		fixture.genesisDigest,
		fixture.reportDigests,
	)
	fixture.release.Evidence = releaseEvidence(fixture.release)
	fixture.resignReleaseAndChoice(t, true)
}

func (fixture *recoveryFixture) resealArtifactsAfterGenesisChange(
	t *testing.T,
) {
	t.Helper()
	fixture.genesisDigest = exactDigest(t, fixture.genesis)
	moduleAfter, err := forkGenesisModuleDigests(fixture.genesis.AppState)
	if err != nil {
		t.Fatal(err)
	}
	mandatoryChanged := map[string]bool{
		"emergency": true,
		"ibc":       true,
		"slashing":  true,
		"staking":   true,
		"transfer":  true,
	}
	for reportIndex := range fixture.reports {
		report := &fixture.reports[reportIndex]
		report.OutputGenesisSHA256 = fixture.genesisDigest
		for digestIndex := range report.ModuleDigests {
			moduleDigest := &report.ModuleDigests[digestIndex]
			moduleDigest.AfterSHA256 = moduleAfter[moduleDigest.Module]
			if moduleDigest.Module == "message_schedule" {
				moduleDigest.BeforeSHA256 = absentModuleSHA256
			} else if !mandatoryChanged[moduleDigest.Module] &&
				moduleDigest.Module != "zerone_staking" {
				moduleDigest.BeforeSHA256 = moduleDigest.AfterSHA256
			}
			moduleDigest.Changed =
				moduleDigest.BeforeSHA256 != moduleDigest.AfterSHA256
		}
		sealCompilerReport(t, report)
		fixture.reportDigests[reportIndex] = exactDigest(t, *report)
	}
	fixture.rebindReleaseToCurrentArtifacts(t)
}

func signerPolicy(
	purpose,
	incidentID,
	chainID,
	assessmentSHA256 string,
	roles []string,
	signers []testSigner,
) SignerPolicy {
	return SignerPolicy{
		Schema:                        signerPolicySchema,
		PolicyID:                      "policy-" + strings.ToLower(purpose),
		Purpose:                       purpose,
		IncidentID:                    incidentID,
		ChainID:                       chainID,
		AssessmentSHA256:              assessmentSHA256,
		MinimumApprovals:              uint64(len(roles)),
		MinimumDistinctIdentities:     uint64(len(roles)),
		MinimumDistinctControlDomains: uint64(len(roles)),
		RequiredRoles:                 append([]string{}, roles...),
		Signers:                       trustedSigners(signers),
	}
}

func makeForkGenesis(
	t *testing.T,
	validator ValidatorIdentity,
	initialHeight uint64,
	chainID string,
) ForkGenesis {
	t.Helper()
	keyBytes, _ := hex.DecodeString(validator.ConsensusPublicKey)
	keyBase64 := base64.StdEncoding.EncodeToString(keyBytes)
	modules := make(map[string]json.RawMessage)
	for _, module := range append(
		append([]string{}, requiredForkGenesisModules...),
		"mint",
	) {
		modules[module] = json.RawMessage(`{}`)
	}
	modules["staking"] = mustJSON(t, map[string]any{
		"validators": []any{
			map[string]any{
				"consensus_pubkey": map[string]any{
					"@type": "/cosmos.crypto.ed25519.PubKey",
					"key":   keyBase64,
				},
				"operator_address": validator.SDKOperatorAddress,
				"status":           "BOND_STATUS_BONDED",
				"tokens":           "100000000",
				"jailed":           false,
			},
		},
	})
	modules["emergency"] = mustJSON(t, map[string]any{
		"status":                  "halted",
		"active_halt_ceremony_id": "legacy-genesis-quarantine",
		"halt_start_block":        initialHeight,
	})
	modules["genutil"] = json.RawMessage(`{"gen_txs":[]}`)
	modules["evidence"] = json.RawMessage(`{"evidence":[]}`)
	modules["gov"] = json.RawMessage(
		`{"deposits":[],"proposals":[],"votes":[]}`,
	)
	modules["upgrade"] = json.RawMessage(`{}`)
	modules["zerone_gov"] = json.RawMessage(
		`{"creed_amendment_pins":[],"lips":[],"next_lip_number":1,"next_seat_election_number":1,"params":{},"research_fund_governance":{},"seat_election_votes":[],"seat_elections":[],"upgrade_plans":[],"votes":[]}`,
	)
	modules["ibc"] = json.RawMessage(
		`{"channel_genesis":{"ack_sequences":[],"acknowledgements":[],"channels":[],"commitments":[],"receipts":[],"recv_sequences":[],"send_sequences":[]},"channel_v2_genesis":{"acknowledgements":[],"async_packets":[],"commitments":[],"receipts":[],"send_sequences":[]},"client_genesis":{"clients":[],"clients_consensus":[],"clients_metadata":[]},"client_v2_genesis":{"counterparty_infos":[]},"connection_genesis":{"client_connection_paths":[],"connections":[]}}`,
	)
	modules["transfer"] = json.RawMessage(
		`{"denoms":[],"total_escrowed":[]}`,
	)
	modules["interchainaccounts"] = json.RawMessage(
		`{"controller_genesis_state":{"active_channels":[],"interchain_accounts":[],"ports":[]},"host_genesis_state":{"active_channels":[],"interchain_accounts":[]}}`,
	)
	modules["feeibc"] = json.RawMessage(
		`{"fee_enabled_channels":[],"forward_relayers":[],"identified_fees":[],"registered_counterparty_payees":[],"registered_payees":[]}`,
	)
	modules["ibcratelimit"] = json.RawMessage(`{}`)
	modules["bank"] = json.RawMessage(`{"balances":[]}`)
	modules["message_schedule"] = append(json.RawMessage(nil), freshMessageScheduleGenesis...)
	appState, err := json.Marshal(modules)
	if err != nil {
		t.Fatal(err)
	}

	type publicKey struct {
		Type  string `json:"type"`
		Value string `json:"value"`
	}
	type consensusValidator struct {
		Address string    `json:"address"`
		PubKey  publicKey `json:"pub_key"`
		Power   string    `json:"power"`
		Name    string    `json:"name"`
	}
	consensus, err := json.Marshal(struct {
		Validators []consensusValidator `json:"validators"`
		Params     json.RawMessage      `json:"params"`
	}{
		Validators: []consensusValidator{
			{
				Address: validator.ConsensusAddress,
				PubKey: publicKey{
					Type:  "tendermint/PubKeyEd25519",
					Value: keyBase64,
				},
				Power: "100",
				Name:  "zerone-recovery-validator",
			},
		},
		Params: json.RawMessage(`{}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	return ForkGenesis{
		AppName:       "zeroned",
		AppVersion:    "v0.53.8-recovery",
		GenesisTime:   "2026-07-30T12:00:01Z",
		ChainID:       chainID,
		InitialHeight: int64(initialHeight),
		AppHash:       json.RawMessage(`null`),
		AppState:      appState,
		Consensus:     consensus,
	}
}

func makeCompilerReport(
	t *testing.T,
	reproducer testSigner,
	release ForkRelease,
	assessment CustodyAssessment,
	assessmentDigest,
	policyDigest string,
	genesis ForkGenesis,
) ForkGenesisReport {
	t.Helper()
	oldKey, _ := hex.DecodeString(
		assessment.OldValidator.ConsensusPublicKey,
	)
	newKey, _ := hex.DecodeString(
		release.NewValidators[0].ConsensusPublicKey,
	)
	oldAddress, _ := hex.DecodeString(
		strings.ToLower(assessment.OldValidator.ConsensusAddress),
	)
	newAddress, _ := hex.DecodeString(
		strings.ToLower(release.NewValidators[0].ConsensusAddress),
	)
	oldBech32, _ := encodeBech32("zrnvalcons", oldAddress)
	newBech32, _ := encodeBech32("zrnvalcons", newAddress)
	moduleAfter, err := forkGenesisModuleDigests(genesis.AppState)
	if err != nil {
		t.Fatal(err)
	}
	moduleNames := make([]string, 0, len(moduleAfter))
	for module := range moduleAfter {
		moduleNames = append(moduleNames, module)
	}
	sort.Strings(moduleNames)
	mandatoryChanged := map[string]bool{
		"emergency": true,
		"ibc":       true,
		"slashing":  true,
		"staking":   true,
		"transfer":  true,
	}
	moduleDigests := make([]ForkGenesisModuleDigest, 0, len(moduleNames))
	for _, module := range moduleNames {
		before := moduleAfter[module]
		if module == "message_schedule" {
			before = absentModuleSHA256
		} else if mandatoryChanged[module] {
			before = fixtureHash("before-" + module)
		}
		moduleDigests = append(moduleDigests, ForkGenesisModuleDigest{
			Module:       module,
			BeforeSHA256: before,
			AfterSHA256:  moduleAfter[module],
			Changed:      before != moduleAfter[module],
		})
	}
	report := ForkGenesisReport{
		Schema:                   forkGenesisReportSchema,
		Profile:                  rewriteProfileConsensusOnly,
		ReproducerID:             reproducer.identity,
		ReproducerControlDomain:  reproducer.controlDomain,
		ReproducerPublicKey:      hex.EncodeToString(reproducer.publicKey),
		IncidentID:               release.IncidentID,
		SourceGenesisSHA256:      release.SourceExportSHA256,
		PolicyFileSHA256:         release.RewritePolicyFileSHA256,
		PolicySHA256:             release.RewritePolicySelfSHA256,
		SourceChainID:            release.OldChainID,
		TargetChainID:            release.NewChainID,
		InitialHeight:            release.InitialHeight,
		SourceBlockIDSHA256:      release.Checkpoint.BlockIDSHA256,
		SourceAppHashSHA256:      release.Checkpoint.AppHashSHA256,
		SourceLastBlockTime:      release.Checkpoint.BlockTime,
		SourceSignedCommitSHA256: release.Checkpoint.SignedCommitSHA256,
		SourceValidatorSetSHA256: release.Checkpoint.ValidatorSetSHA256,
		OldConsensusAddress:      oldBech32,
		NewConsensusAddress:      newBech32,
		OldConsensusPublicKey:    base64.StdEncoding.EncodeToString(oldKey),
		NewConsensusPublicKey:    base64.StdEncoding.EncodeToString(newKey),
		OperatorDisposition:      operatorRetainProvenSafe,
		CustodyAssessmentSHA256:  assessmentDigest,
		ForkPolicySHA256:         policyDigest,
		RewriteToolSHA256:        release.RewriteToolSHA256,
		IBCDisposition:           ibcDispositionRequireEmpty,
		EmergencyStartMode:       emergencyConsensusQuarantine,
		SchemaMigrations: append(
			[]string{},
			requiredForkSchemaMigrations...,
		),
		ModuleDigests:       moduleDigests,
		OutputGenesisSHA256: exactDigest(t, genesis),
	}
	sealCompilerReport(t, &report)
	return report
}

func sealCompilerReport(t *testing.T, report *ForkGenesisReport) {
	t.Helper()
	report.ReportSHA256 = ""
	digest, err := canonicalDigest(*report)
	if err != nil {
		t.Fatal(err)
	}
	report.ReportSHA256 = digest
}

func makeGenesisReproductions(
	signers []testSigner,
	genesisDigest string,
	reportDigests []string,
) []GenesisReproduction {
	result := make([]GenesisReproduction, 0, len(signers))
	for index, signer := range signers {
		result = append(result, GenesisReproduction{
			Identity:                 signer.identity,
			ControlDomain:            signer.controlDomain,
			PublicKey:                hex.EncodeToString(signer.publicKey),
			GenesisSHA256:            genesisDigest,
			CompilerReportFileSHA256: reportDigests[index],
		})
	}
	sort.Slice(result, func(i, j int) bool {
		return trustedReproducerLess(
			TrustedReproducer{
				Identity:      result[i].Identity,
				ControlDomain: result[i].ControlDomain,
				PublicKey:     result[i].PublicKey,
			},
			TrustedReproducer{
				Identity:      result[j].Identity,
				ControlDomain: result[j].ControlDomain,
				PublicKey:     result[j].PublicKey,
			},
		)
	})
	return result
}

func mutateGenesisModule(
	t *testing.T,
	fixture *recoveryFixture,
	module string,
	mutate func(map[string]any),
) {
	t.Helper()
	var modules map[string]json.RawMessage
	if err := json.Unmarshal(fixture.genesis.AppState, &modules); err != nil {
		t.Fatal(err)
	}
	value, err := decodeJSONAny(modules[module])
	if err != nil {
		t.Fatal(err)
	}
	object, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("module %s is not an object", module)
	}
	mutate(object)
	modules[module] = mustJSON(t, object)
	fixture.genesis.AppState = mustJSON(t, modules)
}

func signCustody(
	t *testing.T,
	assessment *CustodyAssessment,
	signers []testSigner,
) {
	t.Helper()
	assessment.Approvals = signApprovals(
		t,
		signers,
		func(approval Approval) (string, error) {
			return custodyApprovalStatement(*assessment, approval)
		},
	)
	var err error
	*assessment, err = sealCustody(*assessment)
	if err != nil {
		t.Fatal(err)
	}
}

func signControlled(
	t *testing.T,
	plan *ControlledTransition,
	signers []testSigner,
) {
	t.Helper()
	plan.Approvals = signApprovals(
		t,
		signers,
		func(approval Approval) (string, error) {
			return controlledApprovalStatement(*plan, approval)
		},
	)
	var err error
	*plan, err = sealControlled(*plan)
	if err != nil {
		t.Fatal(err)
	}
}

func signGenesisReproductions(
	t *testing.T,
	release *ForkRelease,
	signers []testSigner,
) {
	t.Helper()
	byIdentity := make(map[string]testSigner, len(signers))
	for _, signer := range signers {
		byIdentity[signer.identity] = signer
	}
	for index := range release.GenesisReproductions {
		reproduction := &release.GenesisReproductions[index]
		signer, found := byIdentity[reproduction.Identity]
		if !found {
			t.Fatalf("reproducer signer %s is missing", reproduction.Identity)
		}
		statement, err := genesisReproductionStatement(
			*release,
			*reproduction,
		)
		if err != nil {
			t.Fatal(err)
		}
		statementBytes, _ := hex.DecodeString(statement)
		reproduction.StatementSHA256 = statement
		reproduction.Signature = hex.EncodeToString(
			ed25519.Sign(signer.privateKey, statementBytes),
		)
	}
}

func signForkRelease(
	t *testing.T,
	release *ForkRelease,
	signers []testSigner,
) {
	t.Helper()
	release.Approvals = signApprovals(
		t,
		signers,
		func(approval Approval) (string, error) {
			return forkReleaseApprovalStatement(*release, approval)
		},
	)
	var err error
	*release, err = sealForkRelease(*release)
	if err != nil {
		t.Fatal(err)
	}
}

func signForkChoice(
	t *testing.T,
	choice *ForkChoice,
	signers []testSigner,
) {
	t.Helper()
	choice.Approvals = signApprovals(
		t,
		signers,
		func(approval Approval) (string, error) {
			return forkChoiceApprovalStatement(*choice, approval)
		},
	)
	var err error
	*choice, err = sealForkChoice(*choice)
	if err != nil {
		t.Fatal(err)
	}
}

func signApprovals(
	t *testing.T,
	signers []testSigner,
	statement func(Approval) (string, error),
) []Approval {
	t.Helper()
	approvals := make([]Approval, 0, len(signers))
	for _, signer := range signers {
		approval := Approval{
			Role:          signer.role,
			Identity:      signer.identity,
			ControlDomain: signer.controlDomain,
			PublicKey:     hex.EncodeToString(signer.publicKey),
		}
		statementDigest, err := statement(approval)
		if err != nil {
			t.Fatal(err)
		}
		statementBytes, _ := hex.DecodeString(statementDigest)
		approval.StatementSHA256 = statementDigest
		approval.Signature = hex.EncodeToString(
			ed25519.Sign(signer.privateKey, statementBytes),
		)
		approvals = append(approvals, approval)
	}
	sort.Slice(approvals, func(i, j int) bool {
		return approvalLess(approvals[i], approvals[j])
	})
	return approvals
}

func signerSet(roles []string, firstSeed byte) []testSigner {
	signers := make([]testSigner, 0, len(roles))
	for index, role := range roles {
		privateKey := deterministicPrivateKey(firstSeed + byte(index))
		signers = append(signers, testSigner{
			role:          role,
			identity:      "identity-" + role,
			controlDomain: "domain-" + role,
			publicKey:     privateKey.Public().(ed25519.PublicKey),
			privateKey:    privateKey,
		})
	}
	return signers
}

func testReproducer(
	identity,
	controlDomain string,
	seed byte,
) testSigner {
	privateKey := deterministicPrivateKey(seed)
	return testSigner{
		identity:      identity,
		controlDomain: controlDomain,
		publicKey:     privateKey.Public().(ed25519.PublicKey),
		privateKey:    privateKey,
	}
}

func trustedSigners(signers []testSigner) []TrustedSigner {
	result := make([]TrustedSigner, 0, len(signers))
	for _, signer := range signers {
		result = append(result, TrustedSigner{
			Role:          signer.role,
			Identity:      signer.identity,
			ControlDomain: signer.controlDomain,
			PublicKey:     hex.EncodeToString(signer.publicKey),
		})
	}
	sort.Slice(result, func(i, j int) bool {
		return trustedSignerLess(result[i], result[j])
	})
	return result
}

func trustedReproducers(signers []testSigner) []TrustedReproducer {
	result := make([]TrustedReproducer, 0, len(signers))
	for _, signer := range signers {
		result = append(result, TrustedReproducer{
			Identity:      signer.identity,
			ControlDomain: signer.controlDomain,
			PublicKey:     hex.EncodeToString(signer.publicKey),
		})
	}
	sort.Slice(result, func(i, j int) bool {
		return trustedReproducerLess(result[i], result[j])
	})
	return result
}

func deterministicPrivateKey(seedByte byte) ed25519.PrivateKey {
	seed := bytes.Repeat([]byte{seedByte}, ed25519.SeedSize)
	return ed25519.NewKeyFromSeed(seed)
}

func operatorAddress(t *testing.T, value byte) string {
	t.Helper()
	address, err := encodeZRNValoper(bytes.Repeat([]byte{value}, 20))
	if err != nil {
		t.Fatal(err)
	}
	return address
}

func validatorIdentity(
	t *testing.T,
	consensusSeed,
	nodeSeed byte,
	operator,
	prefix string,
) ValidatorIdentity {
	t.Helper()
	consensusPrivate := deterministicPrivateKey(consensusSeed)
	nodePrivate := deterministicPrivateKey(nodeSeed)
	consensusPublic := hex.EncodeToString(
		consensusPrivate.Public().(ed25519.PublicKey),
	)
	nodePublic := hex.EncodeToString(
		nodePrivate.Public().(ed25519.PublicKey),
	)
	consensusAddress, err := consensusAddressFromPublicKeyHex(consensusPublic)
	if err != nil {
		t.Fatal(err)
	}
	nodeID, err := nodeIDFromPublicKeyHex(nodePublic)
	if err != nil {
		t.Fatal(err)
	}
	return ValidatorIdentity{
		SDKOperatorAddress: operator,
		ConsensusPublicKey: consensusPublic,
		ConsensusAddress:   consensusAddress,
		NodePublicKey:      nodePublic,
		NodeID:             nodeID,
		ValidatorKeySHA256: fixtureHash(prefix + "-validator-key"),
		NodeKeySHA256:      fixtureHash(prefix + "-node-key"),
		SigningStateSHA256: fixtureHash(prefix + "-signing-state"),
	}
}

func custodyEvidence(assessment CustodyAssessment) []Evidence {
	result := []Evidence{
		evidence(
			"checkpoint-block-id",
			assessment.Checkpoint.BlockIDSHA256,
		),
		evidence(
			"checkpoint-app-hash",
			assessment.Checkpoint.AppHashSHA256,
		),
		evidence(
			"checkpoint-signed-commit",
			assessment.Checkpoint.SignedCommitSHA256,
		),
		evidence(
			"checkpoint-validator-set",
			assessment.Checkpoint.ValidatorSetSHA256,
		),
		evidence(
			"old-validator-key-file",
			assessment.OldValidator.ValidatorKeySHA256,
		),
		evidence(
			"old-node-key-file",
			assessment.OldValidator.NodeKeySHA256,
		),
		evidence(
			"old-signing-state",
			assessment.OldValidator.SigningStateSHA256,
		),
	}
	for _, finding := range assessment.Findings {
		result = append(
			result,
			evidence("custody-finding", finding.EvidenceSHA256),
		)
	}
	for _, privileged := range assessment.PrivilegedIdentityAssessments {
		result = append(
			result,
			evidence(
				"privileged-identity-assessment",
				privileged.EvidenceSHA256,
			),
		)
	}
	sortEvidence(result)
	return result
}

func controlledEvidence(plan ControlledTransition) []Evidence {
	result := []Evidence{
		evidence("power-snapshot", plan.PowerSnapshotSHA256),
		evidence("binary", plan.BinarySHA256),
		evidence("image", plan.ImageSHA256),
		evidence("provenance", plan.ProvenanceSHA256),
		evidence("sbom", plan.SBOMSHA256),
		evidence("rehearsal", plan.RehearsalSHA256),
		evidence("topology", plan.TopologySHA256),
		evidence("journal-head", plan.JournalHeadSHA256),
		evidence(
			"validator-key-file",
			plan.NewValidator.ValidatorKeySHA256,
		),
		evidence("node-key-file", plan.NewValidator.NodeKeySHA256),
		evidence(
			"signing-state",
			plan.NewValidator.SigningStateSHA256,
		),
	}
	for _, page := range plan.StakeInventory.Delegations.PageSHA256s {
		result = append(result, evidence("delegations-page", page))
	}
	for _, page := range plan.StakeInventory.Unbondings.PageSHA256s {
		result = append(result, evidence("unbondings-page", page))
	}
	for _, page := range plan.StakeInventory.Redelegations.PageSHA256s {
		result = append(result, evidence("redelegations-page", page))
	}
	sortEvidence(result)
	return result
}

func releaseEvidence(release ForkRelease) []Evidence {
	result := []Evidence{
		evidence("source-export", release.SourceExportSHA256),
		evidence("rewrite-tool", release.RewriteToolSHA256),
		evidence("rewrite-policy", release.RewritePolicyFileSHA256),
		evidence("genesis", release.GenesisSHA256),
		evidence(
			"supply-reconciliation",
			release.SupplyReconciliationSHA256,
		),
		evidence(
			"ibc-reconciliation",
			release.IBCReconciliationSHA256,
		),
		evidence(
			"module-reconciliation",
			release.ModuleReconciliationSHA256,
		),
		evidence("binary", release.BinarySHA256),
		evidence("image", release.ImageSHA256),
		evidence("provenance", release.ProvenanceSHA256),
		evidence("sbom", release.SBOMSHA256),
		evidence("rehearsal", release.RehearsalSHA256),
		evidence("topology", release.TopologySHA256),
		evidence("journal-head", release.JournalHeadSHA256),
	}
	for _, validator := range release.NewValidators {
		result = append(
			result,
			evidence(
				"validator-key-file",
				validator.ValidatorKeySHA256,
			),
			evidence("node-key-file", validator.NodeKeySHA256),
			evidence(
				"signing-state",
				validator.SigningStateSHA256,
			),
		)
	}
	for _, reproduction := range release.GenesisReproductions {
		result = append(
			result,
			evidence(
				"fork-genesis-compiler-report",
				reproduction.CompilerReportFileSHA256,
			),
		)
	}
	sortEvidence(result)
	return result
}

func evidence(evidenceType, digest string) Evidence {
	return Evidence{
		Type:   evidenceType,
		SHA256: digest,
		URI:    "vault://incident-2026-001/" + evidenceType + "/" + digest[:12],
	}
}

func sortEvidence(values []Evidence) {
	sort.Slice(values, func(i, j int) bool {
		if values[i].Type != values[j].Type {
			return values[i].Type < values[j].Type
		}
		if values[i].SHA256 != values[j].SHA256 {
			return values[i].SHA256 < values[j].SHA256
		}
		return values[i].URI < values[j].URI
	})
}

func removeEvidenceDigest(values []Evidence, digest string) []Evidence {
	result := make([]Evidence, 0, len(values))
	for _, value := range values {
		if value.SHA256 != digest {
			result = append(result, value)
		}
	}
	return result
}

func fixtureHash(label string) string {
	sum := sha256.Sum256([]byte(label))
	return hex.EncodeToString(sum[:])
}

func exactDigest(t *testing.T, value any) string {
	t.Helper()
	canonical, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return digestBytes(canonical)
}

func mustJSON(t *testing.T, value any) json.RawMessage {
	t.Helper()
	canonical, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return canonical
}

func writeCanonical(
	t *testing.T,
	directory,
	name string,
	value any,
) string {
	t.Helper()
	canonical, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(directory, name)
	if err := os.WriteFile(path, canonical, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func assertNoGoReason(
	t *testing.T,
	report GateReport,
	reason string,
) {
	t.Helper()
	if report.Decision != decisionNoGo {
		t.Fatalf("decision = %q, want %q", report.Decision, decisionNoGo)
	}
	if !containsString(report.ReasonCodes, reason) {
		t.Fatalf("reasons %v do not contain %q", report.ReasonCodes, reason)
	}
	if err := validateGateReportEnvelope(report); err != nil {
		t.Fatalf("invalid NO_GO envelope: %v", err)
	}
}
