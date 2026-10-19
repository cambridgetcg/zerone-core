package keeper_test

import (
	"fmt"
	"testing"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/require"

	"github.com/zerone-chain/zerone/x/knowledge/keeper"
	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// The companion deterministic_order_test.py forces both map traversal orders
// through compiler overlays. Reversing ballot insertion alone does not force
// Go map iteration and is not relied on as the regression's RED proof.
func TestReformulationOrderTies(t *testing.T) {
	verdicts := []types.AugmentationVerdict{
		types.AugmentationVerdict_AUGMENTATION_VERDICT_EQUIVALENT,
		types.AugmentationVerdict_AUGMENTATION_VERDICT_SUPERIOR,
		types.AugmentationVerdict_AUGMENTATION_VERDICT_INFERIOR,
		types.AugmentationVerdict_AUGMENTATION_VERDICT_DRIFT,
	}
	for i, first := range verdicts {
		for _, second := range verdicts[i+1:] {
			for _, reverseBallots := range []bool{false, true} {
				for _, neutral := range []bool{false, true} {
					t.Run(fmt.Sprintf("%s-%s/reverse=%t/neutral=%t", first, second, reverseBallots, neutral), func(t *testing.T) {
						votes := []types.AugmentationVerdict{first, second}
						if reverseBallots {
							votes[0], votes[1] = votes[1], votes[0]
						}
						checkReformulationOrder(t, votes, []uint64{100_000, 100_000}, 500_000, neutral, first)
					})
				}
			}
		}
	}
}

func TestReformulationOrderControls(t *testing.T) {
	equivalent := types.AugmentationVerdict_AUGMENTATION_VERDICT_EQUIVALENT
	drift := types.AugmentationVerdict_AUGMENTATION_VERDICT_DRIFT
	pending := types.AugmentationVerdict_AUGMENTATION_VERDICT_PENDING
	for _, tc := range []struct {
		name      string
		threshold uint64
		stakes    []uint64
		want      types.AugmentationVerdict
	}{
		{"default-threshold-tie", 666_000, []uint64{100_000, 100_000}, pending},
		{"zero-threshold-fallback", 0, []uint64{100_000, 100_000}, pending},
		{"strict-majority-tie", 500_001, []uint64{100_000, 100_000}, pending},
		{"larger-weight-beats-enum-priority", 250_000, []uint64{100_000, 300_000}, drift},
		{"no-stake-no-finalization", 500_000, []uint64{0, 0}, pending},
	} {
		t.Run(tc.name, func(t *testing.T) {
			checkReformulationOrder(t, []types.AugmentationVerdict{equivalent, drift}, tc.stakes, tc.threshold, false, tc.want)
		})
	}
	// Below 50%, more than two verdicts may qualify. Use the same numeric
	// priority for every tie, not a special case for EQUIVALENT versus DRIFT.
	checkReformulationOrder(t, []types.AugmentationVerdict{drift, types.AugmentationVerdict_AUGMENTATION_VERDICT_INFERIOR, equivalent}, []uint64{100_000, 100_000, 100_000}, 333_333, false, equivalent)
}

func checkReformulationOrder(t *testing.T, votes []types.AugmentationVerdict, stakes []uint64, threshold uint64, neutral bool, want types.AugmentationVerdict) {
	t.Helper()
	k, ctx, bank, staking := newKnowledgeOrderFixture(t)
	params, err := k.GetParams(ctx)
	require.NoError(t, err)
	params.ReformulationConsensusBps = threshold
	params.ReformulationMinPanelVotes = uint64(len(votes) + 1)
	require.NoError(t, params.Validate(), "keep existing accepted parameter configurations")
	require.NoError(t, k.SetParams(ctx, params))
	sponsor := sdk.AccAddress([]byte("order-sponsor-000001")).String()
	submitter := sdk.AccAddress([]byte("order-submitter-001")).String()
	sponsorAddr, err := sdk.AccAddressFromBech32(sponsor)
	require.NoError(t, err)
	submitterAddr, err := sdk.AccAddressFromBech32(submitter)
	require.NoError(t, err)
	coins := sdk.NewCoins(sdk.NewInt64Coin("uzrn", 1500))
	require.NoError(t, bank.MintCoins(ctx, orderTestMint, coins))
	require.NoError(t, bank.SendCoinsFromModuleToAccount(ctx, orderTestMint, sponsorAddr, coins))
	require.NoError(t, k.EscrowAugmentationBounty(ctx, sponsor, "bounty", sdkmath.NewInt(1500)))
	require.NoError(t, k.SetFact(ctx, &types.Fact{Id: "original", Status: types.FactStatus_FACT_STATUS_VERIFIED}))
	require.NoError(t, k.SetAugmentationBounty(ctx, &types.AugmentationBounty{Id: "bounty", SponsorAddress: sponsor, TargetFactId: "original", RewardPerVariant: 1000, MaxVariants: 1, Active: true}))
	// A retained pending augmentation remains votable after review neutrality.
	// No new-augmentation admission or activation policy is changed here.
	require.NoError(t, k.SetAugmentation(ctx, &types.Augmentation{Id: "variant", OriginalFactId: "original", BountyId: "bounty", Submitter: submitter}))
	if neutral {
		require.NoError(t, k.EnableReviewNeutrality(ctx))
	}
	server := keeper.NewMsgServerImpl(k)
	for i, vote := range votes {
		voter := fmt.Sprintf("order-voter-%d", i)
		staking.addValidator(voter, stakes[i], "bonded")
		response, err := server.VoteOnAugmentation(ctx, &types.MsgVoteOnAugmentation{AugmentationId: "variant", Verifier: voter, Vote: vote})
		require.NoError(t, err)
		require.False(t, response.VerdictFinalized, "panel count must be reached first")
	}
	staking.addValidator("zero-weight-final-voter", 0, "bonded")
	response, err := server.VoteOnAugmentation(ctx, &types.MsgVoteOnAugmentation{AugmentationId: "variant", Verifier: "zero-weight-final-voter", Vote: votes[0]})
	require.NoError(t, err)
	require.Equal(t, want != types.AugmentationVerdict_AUGMENTATION_VERDICT_PENDING, response.VerdictFinalized)
	require.Equal(t, want, response.FinalizedVerdict, "equal qualifying weight must prefer the lowest numeric verdict")
	aug, found := k.GetAugmentation(ctx, "variant")
	require.True(t, found)
	require.Equal(t, want, aug.Verdict, "message handler must persist the selected verdict")
	require.Len(t, aug.VerdictVoters, len(votes)+1)
	require.Equal(t, append(append([]uint64(nil), stakes...), 0), aug.VerdictVoteStakes)
	wantPayout := int64(0)
	if want == types.AugmentationVerdict_AUGMENTATION_VERDICT_EQUIVALENT {
		wantPayout = 1000
	} else if want == types.AugmentationVerdict_AUGMENTATION_VERDICT_SUPERIOR {
		wantPayout = 1500
	}
	require.Equal(t, wantPayout > 0, aug.Accepted)
	if wantPayout > 0 {
		require.Equal(t, fmt.Sprint(wantPayout), aug.PayoutAmount)
	} else {
		require.Empty(t, aug.PayoutAmount)
	}
	require.Equal(t, wantPayout, bank.GetBalance(ctx, submitterAddr, "uzrn").Amount.Int64())
	require.Equal(t, 1500-wantPayout, bank.GetBalance(ctx, authtypes.NewModuleAddress(types.TrainingFundModuleName), "uzrn").Amount.Int64())
	require.Equal(t, 1500-wantPayout, k.GetEscrowLocked(ctx, "bounty").Int64())
	require.EqualValues(t, 1500, bank.GetSupply(ctx, "uzrn").Amount.Int64())
	bounty, found := k.GetAugmentationBounty(ctx, "bounty")
	require.True(t, found)
	if wantPayout > 0 {
		require.EqualValues(t, 1, bounty.AcceptedVariants)
		require.False(t, bounty.Active)
	} else {
		require.Zero(t, bounty.AcceptedVariants)
		require.True(t, bounty.Active)
	}
	t.Logf("threshold=%d verdict=%s paid=%d locked=%d", threshold, aug.Verdict, wantPayout, 1500-wantPayout)
}
