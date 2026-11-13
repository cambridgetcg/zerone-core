package keeper_test

import (
	"context"
	"fmt"
	"testing"

	sdkmath "cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	qualificationkeeper "github.com/zerone-chain/zerone/x/qualification/keeper"
	qualificationtypes "github.com/zerone-chain/zerone/x/qualification/types"
	"github.com/zerone-chain/zerone/x/substrate_bridge/keeper"
	"github.com/zerone-chain/zerone/x/substrate_bridge/types"
)

func TestQualificationSemanticsNativeGenesisAdmitsActive(t *testing.T) {
	f := newQualificationFixture(t)
	require.NoError(t, f.bridge.InitGenesis(f.ctx, types.DefaultGenesis()))
	msg := f.message(t, "physics", types.QualificationStatus_QUALIFICATION_STATUS_ACTIVE)
	f.native.SetQualification(f.ctx, &qualificationtypes.DomainQualification{
		Validator: msg.Submitter, Domain: "physics", Status: qualificationtypes.QualificationStatus_QUALIFICATION_STATUS_ACTIVE,
	})
	f.assertSubmission(t, msg, true)
}

// The native legacy decay path actually demotes the record. The bridge does
// not distinguish that retained PROBATIONARY state from any other probation.
// This is not evidence that decay remains enabled under review neutrality.
func TestQualificationSemanticsAccuracyDemotionFollowsFloorMeaning(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		for _, minimum := range []types.QualificationStatus{0, 1, 2} {
			t.Run(fmt.Sprintf("on=%t/min=%d", enabled, minimum), func(t *testing.T) {
				f := newQualificationFixture(t)
				msg := f.message(t, "physics", minimum)
				params := qualificationtypes.DefaultParams()
				f.native.SetQualification(f.ctx, &qualificationtypes.DomainQualification{
					Validator: msg.Submitter, Domain: "physics", Status: qualificationtypes.QualificationStatus_QUALIFICATION_STATUS_ACTIVE,
					Metrics: &qualificationtypes.QualificationMetrics{TotalVerifications: params.DecayMinSamples, AccuracyBps: 0},
				})
				require.NoError(t, f.native.RunAccuracyDecay(f.ctx, uint64(f.ctx.BlockHeight()), params))
				q, found := f.native.GetQualification(f.ctx, msg.Submitter, "physics")
				require.True(t, found)
				require.Equal(t, qualificationtypes.QualificationStatus_QUALIFICATION_STATUS_PROBATIONARY, q.Status)
				f.ctx = f.ctx.WithEventManager(sdk.NewEventManager())
				if enabled {
					require.NoError(t, f.bridge.EnableQualificationSemantics(f.ctx))
				}
				f.assertSubmission(t, msg, !enabled || minimum != types.QualificationStatus_QUALIFICATION_STATUS_ACTIVE)
			})
		}
	}
}

func TestQualificationSemanticsRetainsLegacyUnsupportedAdapters(t *testing.T) {
	for _, minimum := range []types.QualificationStatus{3, -1, 4, 2147483647} {
		t.Run(fmt.Sprint(minimum), func(t *testing.T) {
			f := newQualificationFixture(t)
			msg := f.message(t, "physics", minimum)
			a, found := f.bridge.GetAdapter(f.ctx, msg.AdapterId)
			require.True(t, found)
			_, err := keeper.NewMsgServerImpl(f.bridge).RegisterAdapter(f.ctx, &types.MsgRegisterAdapter{Authority: "authority-addr", Adapter: a})
			require.NoError(t, err, "legacy registration must stay permissive")
			before := qualificationStoreSnapshot(t, f.ctx, f.bridgeKey)
			require.NoError(t, f.bridge.EnableQualificationSemantics(f.ctx))
			before[string(types.QualificationSemanticsEnabledKey)] = []byte{1}
			require.Equal(t, before, qualificationStoreSnapshot(t, f.ctx, f.bridgeKey))
			gs := f.bridge.ExportGenesis(f.ctx)
			require.True(t, gs.QualificationSemanticsEnabled)
			require.NoError(t, gs.Validate(), "prospective validation must not strand historical exports")
			imported := newQualificationFixture(t)
			require.NoError(t, imported.bridge.InitGenesis(imported.ctx, gs))
			require.Equal(t, before, qualificationStoreSnapshot(t, imported.ctx, imported.bridgeKey))
			require.True(t, proto.Equal(gs, imported.bridge.ExportGenesis(imported.ctx)))
			imported.native.SetQualification(imported.ctx, &qualificationtypes.DomainQualification{
				Validator: msg.Submitter, Domain: "physics", Status: qualificationtypes.QualificationStatus_QUALIFICATION_STATUS_ACTIVE,
			})
			imported.assertSubmission(t, msg, false)
		})
	}
}

type qualificationBankSpy struct {
	*stubBankKeeper
	paymentCalls int
}

func (b *qualificationBankSpy) SendCoinsFromModuleToAccount(ctx context.Context, mod string, to sdk.AccAddress, coins sdk.Coins) error {
	b.paymentCalls++
	return b.stubBankKeeper.SendCoinsFromModuleToAccount(ctx, mod, to, coins)
}

// Exercise both an in-flight legacy admission and an already escrowed legacy
// reward. Bank/mint seams are tracked stubs, not economic backing evidence.
func TestQualificationSemanticsLegacyWitnessRewardSurvivesActivation(t *testing.T) {
	for _, alreadySettled := range []bool{false, true} {
		t.Run(fmt.Sprintf("settled=%t", alreadySettled), func(t *testing.T) {
			f := newQualificationFixture(t)
			bank := &qualificationBankSpy{stubBankKeeper: f.bank}
			vesting := &stubVestingKeeper{}
			registry := codectypes.NewInterfaceRegistry()
			types.RegisterInterfaces(registry)
			f.bridge = keeper.NewKeeper(codec.NewProtoCodec(registry), f.bridgeKey, "authority-addr", nil,
				qualificationkeeper.NewSubstrateBridgeQualificationAdapter(f.native), bank, vesting)
			msg := f.message(t, "physics", types.QualificationStatus_QUALIFICATION_STATUS_ACTIVE)
			a, _ := f.bridge.GetAdapter(f.ctx, msg.AdapterId)
			a.WitnessRewardUzrn = "222000"
			require.NoError(t, f.bridge.WriteAdapter(f.ctx, a))
			f.native.SetQualification(f.ctx, &qualificationtypes.DomainQualification{
				Validator: msg.Submitter, Domain: "physics", Status: qualificationtypes.QualificationStatus_QUALIFICATION_STATUS_REVOKED,
			})
			f.assertSubmission(t, msg, true)
			id, found := f.bridge.GetSourceRef(f.ctx, msg.AdapterId, msg.Link.Source.SourceId)
			require.True(t, found)
			if alreadySettled {
				require.NoError(t, f.bridge.BeginBlocker(f.ctx))
				require.Equal(t, 1, bank.paymentCalls)
				_, found = f.bridge.GetWitnessPendingReward(f.ctx, id)
				require.True(t, found)
			}
			before := qualificationStoreSnapshot(t, f.ctx, f.bridgeKey)
			beforeNative := qualificationStoreSnapshot(t, f.ctx, f.nativeKey)
			require.NoError(t, f.bridge.EnableQualificationSemantics(f.ctx))
			before[string(types.QualificationSemanticsEnabledKey)] = []byte{1}
			require.Equal(t, before, qualificationStoreSnapshot(t, f.ctx, f.bridgeKey))
			require.Equal(t, beforeNative, qualificationStoreSnapshot(t, f.ctx, f.nativeKey))
			gs := f.bridge.ExportGenesis(f.ctx)
			require.NoError(t, gs.Validate())
			imported := newQualificationFixture(t)
			require.NoError(t, imported.bridge.InitGenesis(imported.ctx, gs))
			require.Equal(t, before, qualificationStoreSnapshot(t, imported.ctx, imported.bridgeKey))
			require.True(t, proto.Equal(gs, imported.bridge.ExportGenesis(imported.ctx)))

			msg.Link.Source.SourceId = "after-activation"
			msg.Link.LinkHash = keeper.ComputeLinkHash(msg.Link)
			f.ctx = f.ctx.WithEventManager(sdk.NewEventManager())
			f.bank.deposits = 0
			f.assertSubmission(t, msg, false)
			require.NoError(t, f.bridge.BeginBlocker(f.ctx))
			require.Equal(t, 1, bank.paymentCalls, "one bond refund, whether settled before or after activation")
			require.Equal(t, "2000000", bank.payments[msg.Submitter].String())
			pending, found := f.bridge.GetWitnessPendingReward(f.ctx, id)
			require.True(t, found)
			require.Empty(t, vesting.minted)
			f.ctx = f.ctx.WithBlockHeight(int64(pending.Deadline))
			require.NoError(t, f.bridge.BeginBlocker(f.ctx))
			require.Equal(t, 2, bank.paymentCalls, "legacy reward still pays once after the challenge window")
			want := sdkmath.NewInt(2000000).Add(sdkmath.NewInt(222000))
			require.Equal(t, want.String(), bank.payments[msg.Submitter].String())
			require.Equal(t, "222000", vesting.minted[types.AuditBountyPoolModuleName].String())
			_, found = f.bridge.GetWitnessPendingReward(f.ctx, id)
			require.False(t, found)
			att, found := f.bridge.GetAttestation(f.ctx, id)
			require.True(t, found)
			require.Equal(t, types.AttestationStatus_ATTESTATION_STATUS_SETTLED, att.Status)
			require.Equal(t, "222000", att.RewardUzrn)
			require.NoError(t, f.bridge.BeginBlocker(f.ctx.WithBlockHeight(f.ctx.BlockHeight()+1)))
			require.Equal(t, 2, bank.paymentCalls, "neither refund nor reward repeats")
			require.Equal(t, want.String(), bank.payments[msg.Submitter].String())
			require.Equal(t, beforeNative, qualificationStoreSnapshot(t, f.ctx, f.nativeKey))
		})
	}
}
