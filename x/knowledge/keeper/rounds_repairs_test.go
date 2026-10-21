package keeper

import (
	"crypto/sha256"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"math"
	"os"
	"testing"

	storetypes "cosmossdk.io/store/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/zerone-chain/zerone/internal/activation"
	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// Real in-memory SDK/IAVL adapters; q2Fixture records staking calls and uses
// no live node, keys, funds or external state. DefaultGenesis explicitly has
// both KR2 markers false; on cases use the production Enable helpers.
func newKR2Fixture(t *testing.T, calibration, floor, q2 bool) *q2Fixture {
	t.Helper()
	f := newQ2Fixture(t, q2)
	if calibration {
		require.NoError(t, f.k.EnableCalibrationSmoothing(f.ctx))
	}
	if floor {
		require.NoError(t, f.k.EnableZeroConfidenceFloor(f.ctx))
	}
	return f
}

func TestKR2LegacyBodiesPinned(t *testing.T) {
	// Hashes calculated from bc2a58fccb3e9bbb43647b0e51490fad034f34fd,
	// not from the new copies. Q2's existing pins also cover orchestration.
	for _, tc := range []struct{ file, name, digest string }{
		{"calibration_attribution_legacy.go", "AttributeContributions", "c0d937608a96679feb52edaaeaf6f20d7deafb4b85b784a441ce7c98704a7034"},
		{"calibration_smoothing_legacy.go", "recordSubmissionOutcomeLegacy", "7a7b852ce3f56d2e0a1b52333162d9ecbb05a3b2b2b2c27155cc2ccbf4ab37af"},
		{"calibration_smoothing_legacy.go", "recordCorroborationForSubmitterLegacy", "c805a3516cd1ea26cfc017246745491edde1e1fa91d86867d91cb7356ac30544"},
		{"calibration_smoothing_legacy.go", "recordDisprovalForSubmitterLegacy", "c25b5401e9ffe1d4babd52cbd5973c7d63a11050850beefc1b1c082d0d40bc8e"},
		{"calibration_smoothing_legacy.go", "recordChallengeOutcomeLegacy", "e906298c181b3f7b2739846c62f39e3a8e1cf357735c50b64d8bee42f60ff39a"},
		{"calibration_smoothing_legacy.go", "ComputeAgentCalibrationScore", "0eb816211ec70d436791027e1f0c9c20082e57f8b697596e2983d940f57f9d37"},
		{"rounds_repairs_legacy.go", "createFactFromClaimLegacy", "02b1e68237040e929c83f1ab7939769121892e8ff39703ac7a721a8f3cec2426"},
		{"rounds_repairs_legacy.go", "computeProvenance", "0e2236f54f7266e4ab449f308f6a978bb99b9aed063c9108a37da0ec7904f965"},
		{"zero_confidence_floor_legacy.go", "advanceConfidenceLegacy", "f26e4a28ccd0a7c11225315d1ed69aac1769c4fc3cf269d42332e4cd02f5d96f"},
		{"training_calibration_legacy.go", "computeTrainingValueWeightLegacy", "00e4acee2646a8dbeb37780813f78d19715ddafe87da759b1982757988c4f84e"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source, err := os.ReadFile(tc.file)
			require.NoError(t, err)
			set := token.NewFileSet()
			file, err := parser.ParseFile(set, tc.file, source, 0)
			require.NoError(t, err)
			found := false
			for _, decl := range file.Decls {
				fn, ok := decl.(*ast.FuncDecl)
				if !ok || fn.Name.Name != tc.name {
					continue
				}
				found = true
				body := source[set.Position(fn.Body.Pos()).Offset:set.Position(fn.Body.End()).Offset]
				require.Equal(t, tc.digest, fmt.Sprintf("%x", sha256.Sum256(body)))
			}
			require.True(t, found)
		})
	}
}

func TestKR2CalibrationSmoothingC2(t *testing.T) {
	require.Zero(t, ComputeSmoothedAgentCalibrationScore(nil))
	require.Zero(t, ComputeSmoothedAgentCalibrationScore(&types.AgentCalibration{}))
	require.EqualValues(t, 500_000, ComputeSmoothedAgentCalibrationScore(&types.AgentCalibration{TotalSubmissions: 1, Accepted: 1, DisprovenCount: 1}))
	require.EqualValues(t, 1, ComputeSmoothedAgentCalibrationScore(&types.AgentCalibration{TotalSubmissions: math.MaxUint64, Accepted: math.MaxUint64, DisprovenCount: math.MaxUint64}))
	for accepted := uint64(0); accepted <= 20; accepted++ {
		for rejected := uint64(0); rejected <= 20; rejected++ {
			for disproven := uint64(0); disproven <= accepted; disproven++ {
				for _, corroborations := range []uint64{0, 1, 10, 100} {
					c := &types.AgentCalibration{TotalSubmissions: accepted + rejected, Accepted: accepted, Rejected: rejected, DisprovenCount: disproven, CorroborationsEarned: corroborations}
					score := ComputeSmoothedAgentCalibrationScore(c)
					require.GreaterOrEqual(t, score, ComputeAgentCalibrationScore(c), "smoothing cannot worsen the same valid bounded record")
					require.LessOrEqual(t, score, uint64(BPS))
					c.TotalSubmissions += 7
					c.Inconclusive += 7
					if accepted+rejected > 0 {
						require.Equal(t, score, ComputeSmoothedAgentCalibrationScore(c), "unresolved attempts do not penalize decisive history")
					} else {
						require.Equal(t, CalibrationReachingCreditBps, ComputeSmoothedAgentCalibrationScore(c))
					}
				}
			}
		}
	}
}

func TestKR2CalibrationProducersAndPredecessorReplay(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		f := newKR2Fixture(t, enabled, false, false)
		require.NoError(t, f.k.RecordSubmissionOutcome(f.ctx, "a", "method", types.Verdict_VERDICT_ACCEPT))
		require.NoError(t, f.k.RecordDisprovalForSubmitter(f.ctx, "a", "method"))
		c, found := f.k.GetAgentCalibration(f.ctx, "a")
		require.True(t, found)
		want := uint64(0)
		if enabled {
			want = 500_000
		}
		require.Equal(t, want, c.CalibrationScoreBps)
		require.EqualValues(t, 1, c.PerMethod["method"].Disproven)
		require.NoError(t, f.k.RecordCorroborationForSubmitter(f.ctx, "a", "method"))
		require.NoError(t, f.k.RecordChallengeOutcome(f.ctx, "a", true))
		c, _ = f.k.GetAgentCalibration(f.ctx, "a")
		if enabled {
			require.EqualValues(t, 600_000, c.CalibrationScoreBps)
		}
		before := proto.Clone(c).(*types.AgentCalibration)
		n, err := f.k.RecomputeAllCalibrationScores(f.ctx)
		require.NoError(t, err)
		require.Equal(t, 1, n)
		c, _ = f.k.GetAgentCalibration(f.ctx, "a")
		require.Equal(t, ComputeAgentCalibrationScore(c), c.CalibrationScoreBps, "predecessor helper stays historical even when Q5 enabled")
		before.CalibrationScoreBps = c.CalibrationScoreBps
		require.True(t, proto.Equal(before, c))
	}
}

func kr2AssertParity(t *testing.T, old, actual *q2Fixture, legacy, current func() error) {
	t.Helper()
	old.resetObservation()
	actual.resetObservation()
	wantErr, wantPanic := runQ2Block(legacy)
	gotErr, gotPanic := runQ2Block(current)
	require.Equal(t, fmt.Sprint(wantErr), fmt.Sprint(gotErr))
	require.Equal(t, wantPanic, gotPanic)
	require.Equal(t, old.ctx.GasMeter().GasConsumed(), actual.ctx.GasMeter().GasConsumed())
	require.Equal(t, old.ctx.EventManager().Events(), actual.ctx.EventManager().Events())
	require.Equal(t, old.snapshot(t), actual.snapshot(t))
	require.Equal(t, old.staking.slashes, actual.staking.slashes)
}

func TestKR2MarkerOffCompleteRoundParity(t *testing.T) {
	for _, verdict := range []types.Verdict{types.Verdict_VERDICT_ACCEPT, types.Verdict_VERDICT_REJECT, types.Verdict_VERDICT_MALFORMED, types.Verdict_VERDICT_INCONCLUSIVE} {
		for _, fault := range []string{"", "get-params", "set-fact", "missing-claim", "low-gas"} {
			t.Run(fmt.Sprintf("%s/%s", verdict, fault), func(t *testing.T) {
				old, actual := newKR2Fixture(t, false, false, false), newKR2Fixture(t, false, false, false)
				var rounds []*types.VerificationRound
				for _, f := range []*q2Fixture{old, actual} {
					r := f.seed(t, "parity", nil)
					rounds = append(rounds, r)
					claim, _ := f.k.GetClaim(f.ctx, r.ClaimId)
					claim.Relations = []*types.ClaimRelation{{TargetFactId: "target", Relation: types.RelationType_RELATION_TYPE_SUPPORTS, InferenceStrengthBps: 1}}
					require.NoError(t, f.k.SetClaim(f.ctx, claim))
					require.NoError(t, f.k.SetFact(f.ctx, &types.Fact{Id: "target", Submitter: "other", Confidence: 700_000, Status: types.FactStatus_FACT_STATUS_VERIFIED}))
					switch fault {
					case "get-params":
						f.probe.failOp, f.probe.failPrefix = "get", types.ParamsKey
					case "set-fact":
						f.probe.failOp, f.probe.failPrefix = "set", types.FactKeyPrefix
					case "missing-claim":
						r.ClaimId = "missing"
					}
				}
				legacy := func() error {
					if fault == "low-gas" {
						old.ctx = old.ctx.WithGasMeter(storetypes.NewGasMeter(100))
					}
					return (roundsExecution{Keeper: old.k}).CompleteRound(old.ctx, rounds[0], &VerificationResult{Verdict: verdict, Confidence: 850_000})
				}
				current := func() error {
					if fault == "low-gas" {
						actual.ctx = actual.ctx.WithGasMeter(storetypes.NewGasMeter(100))
					}
					return actual.k.CompleteRound(actual.ctx, rounds[1], &VerificationResult{Verdict: verdict, Confidence: 850_000})
				}
				kr2AssertParity(t, old, actual, legacy, current)
				require.Equal(t, 1, actual.probe.calibrationReads)
				require.Equal(t, 1, actual.probe.floorReads)
				require.Zero(t, old.probe.calibrationReads)
				require.Zero(t, old.probe.floorReads)
			})
		}
	}
}

func TestKR2SnapshotPresenceAndFreezing(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		for _, floor := range []bool{false, true} {
			for _, neutral := range []bool{false, true} {
				for _, score := range []int64{-1, 0, 1, 500_000} {
					t.Run(fmt.Sprintf("q5=%t/pr37=%t/neutral=%t/score=%d", enabled, floor, neutral, score), func(t *testing.T) {
						f := newKR2Fixture(t, enabled, floor, false)
						if score >= 0 {
							require.NoError(t, f.k.SetAgentCalibration(f.ctx, &types.AgentCalibration{Address: "author", CalibrationScoreBps: uint64(score)}))
						}
						round := &types.VerificationRound{}
						if neutral {
							round.ReviewPolicyVersion = types.ReviewPolicyNeutral
						}
						id, err := f.k.createFactFromClaim(f.ctx, &types.Claim{Id: "claim", Submitter: "author", ClaimType: types.ClaimType_CLAIM_TYPE_ASSERTION}, round, 850_000)
						require.NoError(t, err)
						fact, found := f.k.GetFact(f.ctx, id)
						require.True(t, found)
						require.Equal(t, enabled && !neutral && score >= 0, fact.HasCalibrationSnapshot)
						want := uint64(500_000)
						if neutral {
							want = 0
						} else if score > 0 || (enabled && score == 0) {
							want = uint64(score)
						}
						require.Equal(t, want, fact.SubmitterCalibrationSnapshotBps)
						before := f.k.ComputeTrainingValueWeight(f.ctx, id)
						if enabled && !neutral && score == 0 {
							require.Zero(t, before.Final)
							require.Zero(t, before.SubmitterCalibration)
						}
						require.NoError(t, f.k.SetAgentCalibration(f.ctx, &types.AgentCalibration{Address: "author", CalibrationScoreBps: 999_999}))
						require.Equal(t, before, f.k.ComputeTrainingValueWeight(f.ctx, id), "snapshot cannot follow current reputation")
						bz, err := proto.Marshal(fact)
						require.NoError(t, err)
						var decoded types.Fact
						require.NoError(t, proto.Unmarshal(bz, &decoded))
						require.True(t, proto.Equal(fact, &decoded))
						require.EqualValues(t, 77, fact.ProtoReflect().Descriptor().Fields().ByName("has_calibration_snapshot").Number())
					})
				}
			}
		}
	}
}

func TestKR2FloorCasesAndGrowth(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		for _, scenario := range []string{"none", "missing", "full", "strength-one", "zero-reference", "contradiction-only", "mixed-zero"} {
			t.Run(fmt.Sprintf("enabled=%t/%s", enabled, scenario), func(t *testing.T) {
				f := newKR2Fixture(t, false, enabled, false)
				round := f.seed(t, scenario, nil)
				claim, _ := f.k.GetClaim(f.ctx, round.ClaimId)
				require.NoError(t, f.k.SetFact(f.ctx, &types.Fact{Id: "positive", Confidence: 700_000, Status: types.FactStatus_FACT_STATUS_VERIFIED}))
				require.NoError(t, f.k.SetFact(f.ctx, &types.Fact{Id: "zero", Status: types.FactStatus_FACT_STATUS_VERIFIED}))
				switch scenario {
				case "missing":
					claim.References = []string{"missing"}
				case "full":
					claim.References = []string{"positive"}
				case "strength-one":
					claim.Relations = []*types.ClaimRelation{{TargetFactId: "positive", Relation: types.RelationType_RELATION_TYPE_SUPPORTS, InferenceStrengthBps: 1}}
				case "zero-reference":
					claim.References = []string{"zero"}
				case "contradiction-only":
					claim.Relations = []*types.ClaimRelation{{TargetFactId: "zero", Relation: types.RelationType_RELATION_TYPE_CONTRADICTS}}
				case "mixed-zero":
					claim.References = []string{"positive", "zero"}
				}
				require.NoError(t, f.k.SetClaim(f.ctx, claim))
				f.resetObservation()
				require.NoError(t, f.k.CompleteRound(f.ctx, round, &VerificationResult{Verdict: types.Verdict_VERDICT_ACCEPT, Confidence: 850_000}))
				fact, found := f.k.GetFact(f.ctx, GenerateFactID(claim.Id, uint64(f.ctx.BlockHeight())))
				require.True(t, found)
				want := uint64(850_000)
				if scenario == "full" {
					want = 700_000
				}
				zeroFloor := scenario == "strength-one" || scenario == "zero-reference" || scenario == "mixed-zero"
				if enabled && zeroFloor {
					want = 0
				}
				require.Equal(t, want, fact.Confidence)
				clamped := false
				for _, e := range f.ctx.EventManager().Events() {
					if e.Type == "zerone.knowledge.confidence_clamped_to_floor" {
						clamped = true
					}
				}
				require.Equal(t, scenario == "full" || enabled && zeroFloor, clamped)
				if zeroFloor {
					require.Zero(t, fact.DependencyConfidenceFloor)
					require.EqualValues(t, 1, fact.AxiomDistance)
				}
			})
		}
		for _, status := range []types.FactStatus{types.FactStatus_FACT_STATUS_VERIFIED, types.FactStatus_FACT_STATUS_ACTIVE, types.FactStatus_FACT_STATUS_PROVISIONAL} {
			for _, initial := range []uint64{0, 50} {
				f := newKR2Fixture(t, false, enabled, false)
				require.NoError(t, f.k.SetFact(f.ctx, &types.Fact{Id: "growth", Status: status, Confidence: initial}))
				for i := 0; i < 3; i++ {
					require.NoError(t, f.k.AdvanceConfidence(f.ctx))
				}
				fact, _ := f.k.GetFact(f.ctx, "growth")
				want := initial
				if status != types.FactStatus_FACT_STATUS_PROVISIONAL && (!enabled || initial > 0) {
					want += 3
				}
				require.Equal(t, want, fact.Confidence)
			}
		}
	}
}

func TestKR2BeginBlockMixedSelectionsReadOnce(t *testing.T) {
	for _, q2 := range []bool{false, true} {
		for _, q5 := range []bool{false, true} {
			for _, pr37 := range []bool{false, true} {
				t.Run(fmt.Sprintf("q2=%t/q5=%t/pr37=%t", q2, q5, pr37), func(t *testing.T) {
					f := newKR2Fixture(t, q5, pr37, q2)
					for _, id := range []string{"a", "b"} {
						round := f.seed(t, id, []string{"accept", "accept", "accept", "accept"})
						claim, _ := f.k.GetClaim(f.ctx, round.ClaimId)
						claim.References = []string{"zero"}
						require.NoError(t, f.k.SetClaim(f.ctx, claim))
					}
					require.NoError(t, f.k.SetFact(f.ctx, &types.Fact{Id: "zero", Status: types.FactStatus_FACT_STATUS_VERIFIED}))
					require.NoError(t, f.k.SetAgentCalibration(f.ctx, &types.AgentCalibration{Address: "submitter", TotalSubmissions: 1, Accepted: 1, DisprovenCount: 1}))
					f.resetObservation()
					require.NoError(t, f.k.BeginBlocker(f.ctx))
					require.Equal(t, 1, f.probe.calibrationReads)
					require.Equal(t, 1, f.probe.floorReads)
					require.Equal(t, 1, f.probe.reads)
					for _, id := range []string{"a", "b"} {
						fact, found := f.k.GetFact(f.ctx, GenerateFactID(id, 100))
						require.True(t, found)
						require.Equal(t, q5, fact.HasCalibrationSnapshot)
						if pr37 {
							require.Zero(t, fact.Confidence)
						} else {
							require.Positive(t, fact.Confidence)
						}
					}
					cal, _ := f.k.GetAgentCalibration(f.ctx, "submitter")
					want := ComputeAgentCalibrationScore(cal)
					if q5 {
						want = ComputeSmoothedAgentCalibrationScore(cal)
					}
					require.Equal(t, want, cal.CalibrationScoreBps)
				})
			}
		}
	}
}

func TestKR2MarkerFaultsPrecedeEffects(t *testing.T) {
	for _, key := range [][]byte{types.CalibrationSmoothingEnabledKey, types.ZeroConfidenceFloorEnabledKey} {
		for _, value := range [][]byte{{0}, {2}, {1, 0}, nil} {
			for _, entry := range []string{"begin", "complete", "advance"} {
				t.Run(fmt.Sprintf("%x/%v/%s", key, value, entry), func(t *testing.T) {
					f := newKR2Fixture(t, false, false, true)
					round := f.seed(t, "fault", []string{"accept", "accept", "accept", "accept"})
					if value == nil {
						f.probe.failOp, f.probe.failPrefix = "get", key
					} else {
						f.ctx.KVStore(f.keys[0]).Set(key, value)
					}
					before := f.snapshot(t)
					f.resetObservation()
					var err error
					switch entry {
					case "begin":
						err = f.k.BeginBlocker(f.ctx)
					case "complete":
						err = f.k.CompleteRound(f.ctx, round, &VerificationResult{})
					case "advance":
						err = f.k.AdvanceRoundPhases(f.ctx)
					}
					require.ErrorIs(t, err, activation.ErrMarkerFault)
					require.Zero(t, f.probe.writes)
					require.Zero(t, f.ctx.GasMeter().GasConsumed())
					require.Empty(t, f.ctx.EventManager().Events())
					require.Equal(t, before, f.snapshot(t))
				})
			}
		}
	}
}
