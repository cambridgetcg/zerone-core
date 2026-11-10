package keeper

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/zerone-chain/zerone/internal/activation"
	"github.com/zerone-chain/zerone/x/knowledge/types"
)

func TestKR2TVWPresenceMatrixAndLegacyParity(t *testing.T) {
	for _, present := range []bool{false, true} {
		for _, score := range []uint64{0, 1, 500_000} {
			for _, enabled := range []bool{false, true} {
				f := newKR2Fixture(t, enabled, false, false)
				require.NoError(t, f.k.SetFact(f.ctx, &types.Fact{Id: "fact", Status: types.FactStatus_FACT_STATUS_VERIFIED, CorroborationCount: 10, HasCalibrationSnapshot: present, SubmitterCalibrationSnapshotBps: score}))
				f.resetObservation()
				got, err := f.k.ComputeTrainingValueWeightChecked(f.ctx, "fact")
				require.NoError(t, err)
				require.Equal(t, 1, f.probe.calibrationReads)
				want := score
				if score == 0 && (!enabled || !present) {
					want = 500_000
				}
				require.Equal(t, want, got.SubmitterCalibration)
				if enabled && present && score == 0 {
					require.Zero(t, got.Final)
				}
				if !enabled {
					gas := f.ctx.GasMeter().GasConsumed()
					f.resetObservation()
					require.Equal(t, got, f.k.computeTrainingValueWeightLegacy(f.ctx, "fact"))
					require.Equal(t, gas, f.ctx.GasMeter().GasConsumed())
				}
			}
		}
	}
}

func TestKR2ValuationEntryPointsReadOnce(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		f := newKR2Fixture(t, enabled, false, false)
		for _, id := range []string{"a", "b", "c"} {
			require.NoError(t, f.k.SetFact(f.ctx, &types.Fact{Id: id, MethodId: "kr2-method", Status: types.FactStatus_FACT_STATUS_VERIFIED, HasCalibrationSnapshot: true}))
		}
		require.NoError(t, f.k.SetModelCard(f.ctx, &types.ModelCard{Id: "model", OwnerAddress: "owner"}))
		require.NoError(t, f.k.SetTrainingManifest(f.ctx, &types.TrainingManifest{ManifestId: "manifest", IncludedFactIds: []string{"a", "b", "c"}}))
		f.resetObservation()
		_, err := NewMsgServerImpl(f.k).AttributeContributions(f.ctx, &types.MsgAttributeContributions{ModelId: "model", Owner: "owner", FactIds: []string{"a", "b", "c"}})
		require.NoError(t, err)
		require.Equal(t, 1, f.probe.calibrationReads)
		record, found := f.k.GetContributionRecord(f.ctx, "model")
		require.True(t, found)
		if enabled {
			require.Zero(t, record.ComputedTvw)
		} else {
			require.Positive(t, record.ComputedTvw)
		}
		for _, entry := range []string{"tvw", "trace", "traces", "bundle"} {
			f.resetObservation()
			q := NewQueryServerImpl(f.k)
			switch entry {
			case "tvw":
				_, err = q.TrainingValueWeight(f.ctx, &types.QueryTrainingValueWeightRequest{FactId: "a"})
			case "trace":
				_, err = q.MethodologyApplicationTrace(f.ctx, &types.QueryMethodologyApplicationTraceRequest{FactId: "a"})
			case "traces":
				var r *types.QueryMethodologyApplicationTracesResponse
				r, err = q.MethodologyApplicationTraces(f.ctx, &types.QueryMethodologyApplicationTracesRequest{MethodId: "kr2-method"})
				require.Len(t, r.Traces, 3)
				if enabled {
					for _, trace := range r.Traces {
						require.Zero(t, trace.TrainingValueWeightBps)
					}
				}
			case "bundle":
				var r *types.QueryTrainingManifestBundleResponse
				r, err = f.k.AssembleManifestBundle(f.ctx, "manifest")
				require.Len(t, r.Traces, 3)
			}
			require.NoError(t, err, entry)
			require.Equal(t, 1, f.probe.calibrationReads, entry)
		}
	}
}

func TestKR2ValuationAndProducerMarkerFaults(t *testing.T) {
	for _, entry := range []string{"attribute", "tvw", "trace", "bundle", "submission", "corroboration", "disproval", "challenge", "growth"} {
		t.Run(entry, func(t *testing.T) {
			f := newKR2Fixture(t, false, false, false)
			key := types.CalibrationSmoothingEnabledKey
			if entry == "growth" {
				key = types.ZeroConfidenceFloorEnabledKey
			}
			f.ctx.KVStore(f.keys[0]).Set(key, []byte{2})
			before := f.snapshot(t)
			f.resetObservation()
			var err error
			switch entry {
			case "attribute":
				_, err = NewMsgServerImpl(f.k).AttributeContributions(f.ctx, nil)
			case "tvw":
				_, err = f.k.ComputeTrainingValueWeightChecked(f.ctx, "missing")
			case "trace":
				_, _, err = f.k.BuildMethodologyApplicationTraceChecked(f.ctx, "missing")
			case "bundle":
				_, err = f.k.AssembleManifestBundle(f.ctx, "missing")
			case "submission":
				err = f.k.RecordSubmissionOutcome(f.ctx, "a", "m", types.Verdict_VERDICT_ACCEPT)
			case "corroboration":
				err = f.k.RecordCorroborationForSubmitter(f.ctx, "a", "m")
			case "disproval":
				err = f.k.RecordDisprovalForSubmitter(f.ctx, "a", "m")
			case "challenge":
				err = f.k.RecordChallengeOutcome(f.ctx, "a", true)
			case "growth":
				err = f.k.AdvanceConfidence(f.ctx)
			}
			require.ErrorIs(t, err, activation.ErrMarkerFault)
			require.Zero(t, f.probe.writes)
			require.Zero(t, f.ctx.GasMeter().GasConsumed())
			require.Empty(t, f.ctx.EventManager().Events())
			require.Equal(t, before, f.snapshot(t))
		})
	}
}

func TestKR2StandaloneOffProducerAndGrowthParity(t *testing.T) {
	for _, op := range []string{"accept", "inconclusive", "corroborate", "disprove", "challenge", "growth"} {
		for _, fail := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/fail=%t", op, fail), func(t *testing.T) {
				old, actual := newKR2Fixture(t, false, false, false), newKR2Fixture(t, false, false, false)
				for _, f := range []*q2Fixture{old, actual} {
					require.NoError(t, f.k.SetAgentCalibration(f.ctx, &types.AgentCalibration{Address: "a", TotalSubmissions: 1, Accepted: 1, DisprovenCount: 1}))
					require.NoError(t, f.k.SetFact(f.ctx, &types.Fact{Id: "zero", Status: types.FactStatus_FACT_STATUS_VERIFIED}))
					if fail {
						f.probe.failOp, f.probe.failPrefix = "set", types.AgentCalibrationKeyPrefix
					}
				}
				legacy := func() error {
					switch op {
					case "accept":
						return old.k.recordSubmissionOutcomeLegacy(old.ctx, "a", "m", types.Verdict_VERDICT_ACCEPT)
					case "inconclusive":
						return old.k.recordSubmissionOutcomeLegacy(old.ctx, "a", "m", types.Verdict_VERDICT_INCONCLUSIVE)
					case "corroborate":
						return old.k.recordCorroborationForSubmitterLegacy(old.ctx, "a", "m")
					case "disprove":
						return old.k.recordDisprovalForSubmitterLegacy(old.ctx, "a", "m")
					case "challenge":
						return old.k.recordChallengeOutcomeLegacy(old.ctx, "a", false)
					default:
						return old.k.advanceConfidenceLegacy(old.ctx)
					}
				}
				current := func() error {
					switch op {
					case "accept":
						return actual.k.RecordSubmissionOutcome(actual.ctx, "a", "m", types.Verdict_VERDICT_ACCEPT)
					case "inconclusive":
						return actual.k.RecordSubmissionOutcome(actual.ctx, "a", "m", types.Verdict_VERDICT_INCONCLUSIVE)
					case "corroborate":
						return actual.k.RecordCorroborationForSubmitter(actual.ctx, "a", "m")
					case "disprove":
						return actual.k.RecordDisprovalForSubmitter(actual.ctx, "a", "m")
					case "challenge":
						return actual.k.RecordChallengeOutcome(actual.ctx, "a", false)
					default:
						return actual.k.AdvanceConfidence(actual.ctx)
					}
				}
				kr2AssertParity(t, old, actual, legacy, current)
			})
		}
	}
}

func TestKR2PresentZeroGenesisRoundTrip(t *testing.T) {
	f := newKR2Fixture(t, true, true, false)
	f.k.bankKeeper = accountingZeroBank{}
	fact := &types.Fact{Id: "present-zero", Status: types.FactStatus_FACT_STATUS_VERIFIED, HasCalibrationSnapshot: true}
	require.NoError(t, f.k.SetFact(f.ctx, fact))
	gs := f.k.ExportGenesis(f.ctx)
	bz, err := proto.Marshal(gs)
	require.NoError(t, err)
	var decoded types.GenesisState
	require.NoError(t, proto.Unmarshal(bz, &decoded))
	target := newKR2Fixture(t, false, false, false)
	target.k.bankKeeper = accountingZeroBank{}
	require.NoError(t, target.k.InitGenesis(target.ctx, &decoded))
	got, found := target.k.GetFact(target.ctx, fact.Id)
	require.True(t, found)
	require.True(t, proto.Equal(fact, got))
	tvw, err := target.k.ComputeTrainingValueWeightChecked(target.ctx, fact.Id)
	require.NoError(t, err)
	require.Zero(t, tvw.SubmitterCalibration)
	require.Zero(t, tvw.Final)
}
