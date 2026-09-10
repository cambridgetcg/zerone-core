package keeper

import (
	"bytes"
	"context"
	"fmt"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/zerone-chain/zerone/x/knowledge/types"
	vestingtypes "github.com/zerone-chain/zerone/x/vesting_rewards/types"
)

// This narrow funded-bank double measures the actual helper transfer calls.
// It is not an ante/signature or SDK bank integration fixture.
type bonusPolicyBank struct {
	types.BankKeeper
	balances      map[string]sdk.Coins
	mintCalls     int
	transferCalls int
}

func (b *bonusPolicyBank) GetBalance(_ context.Context, address sdk.AccAddress, denom string) sdk.Coin {
	return sdk.NewCoin(denom, b.balances[address.String()].AmountOf(denom))
}
func (b *bonusPolicyBank) SendCoinsFromModuleToAccount(_ context.Context, module string, recipient sdk.AccAddress, coins sdk.Coins) error {
	address := authtypes.NewModuleAddress(module).String()
	balance, negative := b.balances[address].SafeSub(coins...)
	if negative {
		return fmt.Errorf("unfunded fixture module %s", module)
	}
	b.balances[address] = balance
	b.balances[recipient.String()] = b.balances[recipient.String()].Add(coins...)
	b.transferCalls++
	return nil
}
func (b *bonusPolicyBank) SendCoinsFromModuleToModule(ctx context.Context, sender, recipient string, coins sdk.Coins) error {
	return b.SendCoinsFromModuleToAccount(ctx, sender, authtypes.NewModuleAddress(recipient), coins)
}
func (b *bonusPolicyBank) MintCoins(_ context.Context, module string, coins sdk.Coins) error {
	address := authtypes.NewModuleAddress(module).String()
	b.balances[address] = b.balances[address].Add(coins...)
	b.mintCalls++
	return nil
}
func bonusPolicyFixture(t *testing.T, active bool) (handoffFixture, *bonusPolicyBank) {
	t.Helper()
	f := setupHandoff(t)
	params := types.DefaultParams()
	require.NoError(t, f.k.SetParams(f.ctx, &params))
	if active {
		require.NoError(t, f.k.EnableRecordIntegrity(f.ctx))
		require.NoError(t, f.k.EnableReviewNeutrality(f.ctx))
	}
	bank := &bonusPolicyBank{balances: make(map[string]sdk.Coins)}
	for _, module := range []string{types.ModuleName, types.ProbeBountyPoolModuleName, protocolTreasuryModule} {
		bank.balances[authtypes.NewModuleAddress(module).String()] = sdk.NewCoins(sdk.NewInt64Coin("uzrn", 100_000_000))
	}
	f.k.bankKeeper = bank
	return f, bank
}

func TestReviewPolicyRetiresFundedBonusesPreservingLegacyRefunds(t *testing.T) {
	for _, policy := range []uint32{0, 1} {
		t.Run(fmt.Sprint(policy), func(t *testing.T) {
			f, bank := bonusPolicyFixture(t, true)
			challenger := sdk.AccAddress(bytes.Repeat([]byte{71}, 20))
			target := &types.Fact{Id: "target", ClaimId: "original", Submitter: challenger.String(), Status: types.FactStatus_FACT_STATUS_DISPROVEN, Confidence: 900_000, ProbeInvitedAtBlock: 99}
			require.NoError(t, f.k.SetFact(f.ctx, target))
			claim := &types.Claim{Id: "challenge", Submitter: challenger.String(), ProvisionalFactId: target.Id, Stake: "1100000", ReviewPolicyVersion: policy}
			params := types.DefaultParams()
			// The target author and challenger deliberately coincide: correction and
			// returning actual collateral do not depend on a claim of independence.
			require.NoError(t, f.k.settleChallengeStake(f.ctx, claim, types.Verdict_VERDICT_ACCEPT, &params))
			expected := "495000" //45% of the actual locked stake, never a new bonus.
			if policy == 0 {
				expected = "23595000"
			} //legacy23.1ZRN confidence bonus remains funded.
			require.Equal(t, expected, bank.GetBalance(f.ctx, challenger, "uzrn").Amount.String())
			before := bank.GetBalance(f.ctx, challenger, "uzrn")
			for i := 0; i < 2; i++ {
				claim.Id = fmt.Sprintf("distinct-invited-challenge-%d", i)
				require.NoError(t, f.k.payInvitationBonus(f.ctx, claim, &params))
			}
			delta := bank.GetBalance(f.ctx, challenger, "uzrn").Amount.Sub(before.Amount)
			if policy == 0 {
				require.Equal(t, "1000000", delta.String())
			} else {
				require.True(t, delta.IsZero())
			}
			require.Zero(t, bank.mintCalls)
		})
	}
}

func TestReviewPolicyRejectKeepsOwnStakeReturnWithoutNewBonus(t *testing.T) {
	f, bank := bonusPolicyFixture(t, true)
	recipient := sdk.AccAddress(bytes.Repeat([]byte{72}, 20))
	claim := &types.Claim{Id: "rejected-challenge", Submitter: recipient.String(), Stake: "1100000", ReviewPolicyVersion: 1}
	params := types.DefaultParams()
	require.NoError(t, f.k.settleChallengeStake(f.ctx, claim, types.Verdict_VERDICT_REJECT, &params))
	require.Equal(t, "165000", bank.GetBalance(f.ctx, recipient, "uzrn").Amount.String())
	require.Equal(t, 2, bank.transferCalls) //15%return +30%treasury; reviewer55%is separate.
}

func TestReviewPolicySurvivalPreservesAccruedHandoffWithoutNewStanding(t *testing.T) {
	for _, policy := range []uint32{0, 1} {
		t.Run(fmt.Sprint(policy), func(t *testing.T) {
			f, bank := bonusPolicyFixture(t, true)
			fact, found := f.k.GetFact(f.ctx, f.pending.FactId)
			require.True(t, found)
			fact.Status = types.FactStatus_FACT_STATUS_CHALLENGED
			fact.Energy = 123
			fact.CorroborationCount = 4
			fact.LastCorroboratedBlock = 10
			require.NoError(t, f.k.SetFact(f.ctx, fact))
			claim := &types.Claim{Id: "challenge", Submitter: fact.Submitter, ProvisionalFactId: fact.Id, ReviewPolicyVersion: policy}
			require.NoError(t, f.k.handleChallengeSurvival(f.ctx, claim))
			restored, found := f.k.GetFact(f.ctx, fact.Id)
			require.True(t, found)
			require.Equal(t, types.FactStatus_FACT_STATUS_ACTIVE, restored.Status)
			require.Equal(t, uint64(123), restored.Energy)
			require.Equal(t, uint64(4), restored.CorroborationCount)
			require.Equal(t, uint64(10), restored.LastCorroboratedBlock)
			_, found, err := f.k.getSurvivalPendingReward(f.ctx, fact.Id)
			require.NoError(t, err)
			require.False(t, found)
			// Real knowledge/vesting stores: the pre-existing pending amount becomes
			// one unchanged schedule. Retrying the correction does not create another.
			key := append(bytes.Clone(vestingtypes.ClaimRecordKeyPrefix), []byte(f.pending.ClaimId)...)
			scheduleID := bytes.Clone(f.ctx.KVStore(f.keys[1]).Get(key))
			require.NotEmpty(t, scheduleID)
			snapshot := f.snapshot(t)
			require.NoError(t, f.k.releaseSurvivalReward(f.ctx, fact.Id))
			require.Equal(t, snapshot, f.snapshot(t))
			require.Zero(t, bank.transferCalls)
			require.Zero(t, bank.mintCalls)
			_, calibration := f.k.GetAgentCalibration(f.ctx, fact.Submitter)
			require.False(t, calibration)
		})
	}
}

func TestReviewPolicyNoNewNominalRewardButLegacyClaimRetainsTerms(t *testing.T) {
	f, _ := bonusPolicyFixture(t, true)
	for _, policy := range []uint32{0, 1} {
		claim := &types.Claim{Id: fmt.Sprintf("accepted-%d", policy), Submitter: "recipient", Stake: "200000", Category: "empirical", ReviewPolicyVersion: policy}
		fact := &types.Fact{Id: fmt.Sprintf("accepted-fact-%d", policy), ClaimId: claim.Id, Submitter: claim.Submitter, Status: types.FactStatus_FACT_STATUS_VERIFIED}
		require.NoError(t, f.k.SetFact(f.ctx, fact))
		require.NoError(t, f.k.EscrowSubmitterReward(f.ctx, fact, claim))
		pending, found, err := f.k.getSurvivalPendingReward(f.ctx, fact.Id)
		require.NoError(t, err)
		require.Equal(t, policy == 0, found)
		if policy == 0 {
			require.Equal(t, "200000", pending.Amount)
		} else {
			require.Zero(t, fact.ChallengeWindowEnd)
		}
	}
}

func TestReviewPolicyProbeMintRetiredWithoutSweeping(t *testing.T) {
	for _, active := range []bool{false, true} {
		f, bank := bonusPolicyFixture(t, active)
		f.k.vestingRewardsKeeper = nil //explicit test-only mint fallback, never production wiring.
		params := types.DefaultParams()
		params.ProbeBountyMintPerBlock = "1000"
		before := f.k.ProbeBountyPoolBalance(f.ctx)
		require.NoError(t, f.k.MintToProbeBountyPool(f.ctx, &params))
		after := f.k.ProbeBountyPoolBalance(f.ctx)
		if active {
			require.Equal(t, before, after)
			require.Zero(t, bank.mintCalls)
		} else {
			require.Equal(t, "100001000", after.String())
			require.Equal(t, 1, bank.mintCalls)
		}
	}
}

func TestReviewPolicyInvalidGateOrClaimCannotTriggerBonusEffects(t *testing.T) {
	for _, fault := range []string{"corrupt-marker", "marker-read", "unknown-policy", "nil-claim"} {
		t.Run(fault, func(t *testing.T) {
			f, bank := bonusPolicyFixture(t, true)
			claim := &types.Claim{Id: "invalid", Submitter: sdk.AccAddress(bytes.Repeat([]byte{73}, 20)).String(), Stake: "1100000", ProvisionalFactId: f.pending.FactId, ReviewPolicyVersion: 1}
			switch fault {
			case "corrupt-marker":
				require.NoError(t, f.k.storeService.OpenKVStore(f.ctx).Set([]byte(ReviewNeutralityEnabledStoreKey), []byte{2}))
			case "marker-read":
				*f.kFault = handoffFault{op: "get", prefix: []byte(ReviewNeutralityEnabledStoreKey)}
			case "unknown-policy":
				claim.ReviewPolicyVersion = 2
			case "nil-claim":
				claim = nil
			}
			params := types.DefaultParams()
			before := f.snapshot(t)
			fact, _ := f.k.GetFact(f.ctx, f.pending.FactId)
			require.Error(t, f.k.payInvitationBonus(f.ctx, claim, &params))
			require.Error(t, f.k.settleChallengeStake(f.ctx, claim, types.Verdict_VERDICT_ACCEPT, &params))
			require.Error(t, f.k.handleChallengeSurvival(f.ctx, claim))
			require.Error(t, f.k.EscrowSubmitterReward(f.ctx, fact, claim))
			require.Equal(t, before, f.snapshot(t))
			require.Zero(t, bank.transferCalls)
			require.Zero(t, bank.mintCalls)
		})
	}
}

func TestReviewPolicyNeutralDisprovalRetainsHistoryWithoutCalibration(t *testing.T) {
	f, _ := bonusPolicyFixture(t, true)
	original, found := f.k.GetFact(f.ctx, f.pending.FactId)
	require.True(t, found)
	original.Status = types.FactStatus_FACT_STATUS_CHALLENGED
	require.NoError(t, f.k.SetFact(f.ctx, original))
	claim := &types.Claim{Id: "neutral-disproof", Submitter: original.Submitter, ProvisionalFactId: original.Id, ReviewPolicyVersion: 1}
	before := proto.Clone(original).(*types.Fact)
	require.NoError(t, f.k.handleChallengeDisproven(f.ctx, claim, "counter-fact"))
	after, found := f.k.GetFact(f.ctx, original.Id)
	require.True(t, found)
	before.Status = types.FactStatus_FACT_STATUS_DISPROVEN
	require.True(t, proto.Equal(before, after))
	_, found, err := f.k.getSurvivalPendingReward(f.ctx, original.Id)
	require.NoError(t, err)
	require.False(t, found)
	_, found = f.k.GetAgentCalibration(f.ctx, original.Submitter)
	require.False(t, found)
}

func TestReviewPolicyInvitationRetirementPreservesExistingStamps(t *testing.T) {
	for _, active := range []bool{false, true} {
		f, _ := bonusPolicyFixture(t, active)
		fact := &types.Fact{Id: "idle-invited", Status: types.FactStatus_FACT_STATUS_VERIFIED, Confidence: 900000, VerifiedAtBlock: 1, ProbeInvitedAtBlock: 10}
		require.NoError(t, f.k.SetFact(f.ctx, fact))
		params := types.DefaultParams()
		before := f.snapshot(t)
		events := len(f.ctx.EventManager().Events())
		require.NoError(t, f.k.InviteIdleFactsForProbing(f.ctx, 200000, &params))
		after, found := f.k.GetFact(f.ctx, fact.Id)
		require.True(t, found)
		if active {
			require.Equal(t, uint64(10), after.ProbeInvitedAtBlock)
			require.Equal(t, before, f.snapshot(t))
			require.Len(t, f.ctx.EventManager().Events(), events)
		} else {
			require.Equal(t, uint64(200000), after.ProbeInvitedAtBlock)
		}
	}
}

func TestReviewPolicyUnreadableGateRefusesAllAdmissionBeforeEffects(t *testing.T) {
	f, bank := bonusPolicyFixture(t, true)
	*f.kFault = handoffFault{op: "get", prefix: []byte(ReviewNeutralityEnabledStoreKey)}
	ms := NewMsgServerImpl(f.k)
	before := f.snapshot(t)
	events := len(f.ctx.EventManager().Events())
	_, err := ms.SubmitClaim(f.ctx, &types.MsgSubmitClaim{})
	require.ErrorContains(t, err, "injected read failure")
	_, err = ms.ChallengeFact(f.ctx, &types.MsgChallengeFact{})
	require.ErrorContains(t, err, "injected read failure")
	_, err = ms.ChallengeProvisionalFact(f.ctx, &types.MsgChallengeProvisionalFact{})
	require.ErrorContains(t, err, "injected read failure")
	_, err = ms.SubmitContradiction(f.ctx, &types.MsgSubmitContradiction{})
	require.ErrorContains(t, err, "injected read failure")
	_, err = ms.PostConjecture(f.ctx, &types.MsgPostConjecture{})
	require.ErrorContains(t, err, "injected read failure")
	params := types.DefaultParams()
	params.ProbeBountyMintPerBlock = "1000"
	require.ErrorContains(t, f.k.MintToProbeBountyPool(f.ctx, &params), "injected read failure")
	require.ErrorContains(t, f.k.InviteIdleFactsForProbing(f.ctx, 200000, &params), "injected read failure")
	require.Equal(t, before, f.snapshot(t))
	require.Len(t, f.ctx.EventManager().Events(), events)
	require.Zero(t, bank.transferCalls)
	require.Zero(t, bank.mintCalls)
}
