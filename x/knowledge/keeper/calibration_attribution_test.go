package keeper

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/zerone-chain/zerone/x/knowledge/types"
)

func TestKR2AttributionOffParity(t *testing.T) {
	for _, neutral := range []bool{false, true} {
		for _, fault := range []string{"", "nil", "missing", "owner", "set"} {
			t.Run(fmt.Sprintf("neutral=%t/fault=%s", neutral, fault), func(t *testing.T) {
				old, actual := newKR2Fixture(t, false, false, false), newKR2Fixture(t, false, false, false)
				msg := &types.MsgAttributeContributions{ModelId: "model", Owner: "owner", FactIds: []string{"zero", "unknown", "positive", "zero", "missing"}, TotalWeight: 7}
				for _, f := range []*q2Fixture{old, actual} {
					require.NoError(t, f.k.SetModelCard(f.ctx, &types.ModelCard{Id: "model", OwnerAddress: "owner"}))
					for _, fact := range []*types.Fact{
						{Id: "zero", Status: types.FactStatus_FACT_STATUS_VERIFIED, HasCalibrationSnapshot: true},
						{Id: "unknown", Status: types.FactStatus_FACT_STATUS_VERIFIED},
						{Id: "positive", Status: types.FactStatus_FACT_STATUS_VERIFIED, SubmitterCalibrationSnapshotBps: 900_000},
					} {
						require.NoError(t, f.k.SetFact(f.ctx, fact))
					}
					if neutral {
						require.NoError(t, f.k.EnableReviewNeutrality(f.ctx))
					}
					if fault == "set" {
						f.probe.failOp, f.probe.failPrefix = "set", types.ContributionByModelKeyPrefix
					}
				}
				switch fault {
				case "nil":
					msg = nil
				case "missing":
					msg.ModelId = "missing"
				case "owner":
					msg.Owner = "other"
				}
				var want, got *types.MsgAttributeContributionsResponse
				kr2AssertParity(t, old, actual, func() (err error) {
					want, err = (calibrationMsgLegacy{calibrationKeeperLegacy{old.k}}).AttributeContributions(old.ctx, msg)
					return
				}, func() (err error) {
					got, err = NewMsgServerImpl(actual.k).AttributeContributions(actual.ctx, msg)
					return
				})
				require.Equal(t, want, got)
				require.Equal(t, 1, actual.probe.calibrationReads)
				require.Zero(t, old.probe.calibrationReads)
			})
		}
	}
}

func TestKR2BeginBlockGrowthUsesSnapshot(t *testing.T) {
	for _, q2 := range []bool{false, true} {
		for _, pr37 := range []bool{false, true} {
			f := newKR2Fixture(t, true, pr37, q2)
			params, err := f.k.GetParams(f.ctx)
			require.NoError(t, err)
			params.ConfidenceGrowthEpoch = 1
			params.ConfidenceGrowthPerEpochBps = 1
			require.NoError(t, f.k.SetParams(f.ctx, params))
			for _, initial := range []uint64{0, 50} {
				require.NoError(t, f.k.SetFact(f.ctx, &types.Fact{Id: fmt.Sprint(initial), Confidence: initial, Status: types.FactStatus_FACT_STATUS_ACTIVE}))
			}
			f.resetObservation()
			require.NoError(t, f.k.BeginBlocker(f.ctx))
			require.Equal(t, 1, f.probe.calibrationReads)
			require.Equal(t, 1, f.probe.floorReads)
			zero, _ := f.k.GetFact(f.ctx, "0")
			positive, _ := f.k.GetFact(f.ctx, "50")
			if pr37 {
				require.Zero(t, zero.Confidence)
			} else {
				require.EqualValues(t, 1, zero.Confidence)
			}
			require.EqualValues(t, 51, positive.Confidence)
		}
	}
}
