package keeper_test

// #118, review of 479e07e5 (finding 2). With the marker on, a passing verdict
// deferred while a sibling held its bounty could finalize after the bounty
// expired. Expiry had already returned the remaining escrow to the sponsor, so
// the payout came from the pooled training fund and left another bounty short.
// The same happens on base when a vote simply arrives after expiry. With the
// marker on, both now archive the verdict without payout; with it off, the base
// behaviour is unchanged. Adapted from the reviewer's probe.

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

func TestReformulationVerdictAfterBountyExpiryIsNotPaidFromThePool(t *testing.T) {
	for _, tc := range []struct {
		name       string
		enabled    bool
		siblingTie bool // b's passing tally is deferred by a's hold, then a's tie breaks
		paid       bool
	}{
		{"marker on, deferred by a held sibling", true, true, false},
		{"marker on, late vote", true, false, false},
		{"marker off, same deferred sequence", false, true, true},
		{"marker off, late vote (base behaviour)", false, false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			k, ctx, bank, staking := newKnowledgeOrderFixture(t)
			params, err := k.GetParams(ctx)
			require.NoError(t, err)
			params.ReformulationMinPanelVotes = 2
			params.ReformulationConsensusBps = 500_000
			require.NoError(t, k.SetParams(ctx, params))
			fund := authtypes.NewModuleAddress(types.TrainingFundModuleName)
			author := sdk.AccAddress([]byte("expired-bounty-author"))
			for _, id := range []string{"p1", "p2"} {
				sponsor := sdk.AccAddress([]byte(fmt.Sprintf("expired-bounty-spon-%s", id)))
				coins := sdk.NewCoins(sdk.NewInt64Coin("uzrn", 1500))
				require.NoError(t, bank.MintCoins(ctx, orderTestMint, coins))
				require.NoError(t, bank.SendCoinsFromModuleToAccount(ctx, orderTestMint, sponsor, coins))
				require.NoError(t, k.EscrowAugmentationBounty(ctx, sponsor.String(), id, sdkmath.NewInt(1500)))
				expires := map[string]uint64{"p1": 200, "p2": 10_000}[id]
				require.NoError(t, k.SetAugmentationBounty(ctx, &types.AugmentationBounty{Id: id, SponsorAddress: sponsor.String(), Active: true, RewardPerVariant: 1000, MaxVariants: 2, ExpiresAtBlock: expires}))
			}
			if tc.siblingTie {
				require.NoError(t, k.SetAugmentation(ctx, &types.Augmentation{Id: "a", BountyId: "p1", Submitter: author.String(), VerdictVoters: []string{"t1", "t2"}, VerdictVotes: []types.AugmentationVerdict{1, 4}, VerdictVoteStakes: []uint64{100_000, 100_000}}))
			}
			require.NoError(t, k.SetAugmentation(ctx, &types.Augmentation{Id: "b", BountyId: "p1", Submitter: author.String(), VerdictVoters: []string{"w1"}, VerdictVotes: []types.AugmentationVerdict{1}, VerdictVoteStakes: []uint64{100_000}}))
			if tc.enabled {
				require.NoError(t, k.EnableReformulationVindicationOrdering(ctx))
			}
			staking.addValidator("x", 100_000, "bonded")
			staking.addValidator("y", 200_000, "bonded")
			server := keeper.NewMsgServerImpl(k)
			vote := func(h int64, id, voter string, v types.AugmentationVerdict) bool {
				resp, err := server.VoteOnAugmentation(ctx.WithBlockHeight(h), &types.MsgVoteOnAugmentation{AugmentationId: id, Verifier: voter, Vote: v})
				require.NoError(t, err)
				return resp.VerdictFinalized
			}
			if tc.siblingTie {
				finalized := vote(150, "b", "x", 1)
				require.Equal(t, !tc.enabled, finalized, "b's passing tally is deferred only while a holds the bounty with the marker on")
				vote(150, "a", "y", 3) // breaks a's tie as INFERIOR, releasing the hold
			}
			require.NoError(t, k.ProcessRouteBLifecycle(ctx.WithBlockHeight(201)))
			p1, _ := k.GetAugmentationBounty(ctx, "p1")
			require.False(t, p1.Active, "p1 expired at 200")

			// A re-tally after expiry: a zero-weight voter for the deferred
			// case, a late staked vote otherwise.
			staking.addValidator("z", 0, "bonded")
			voter := "x"
			if tc.siblingTie {
				voter = "z"
			}
			if b, _ := k.GetAugmentation(ctx, "b"); b.Verdict == types.AugmentationVerdict_AUGMENTATION_VERDICT_PENDING {
				require.True(t, vote(202, "b", voter, 1), "b finalizes after expiry")
			}
			b, _ := k.GetAugmentation(ctx, "b")
			fundBalance := bank.GetBalance(ctx, fund, "uzrn").Amount
			p2Locked := k.GetEscrowLocked(ctx, "p2")
			if tc.paid {
				require.Equal(t, int64(1000), bank.GetBalance(ctx, author, "uzrn").Amount.Int64())
				require.True(t, b.Accepted)
				if !tc.siblingTie {
					// The base defect, kept with the marker off: the payout drew on
					// p2's backing.
					require.True(t, fundBalance.LT(p2Locked), "fund %s, p2 locked %s", fundBalance, p2Locked)
				}
				return
			}
			require.True(t, bank.GetBalance(ctx, author, "uzrn").Amount.IsZero(), "no payout from the pool")
			require.False(t, b.Accepted)
			require.Equal(t, "bounty expired before finalize", b.AcceptanceNote)
			require.Equal(t, types.AugmentationVerdict_AUGMENTATION_VERDICT_EQUIVALENT, b.Verdict)
			require.True(t, fundBalance.GTE(p2Locked), "p2 stays backed: fund %s, p2 locked %s", fundBalance, p2Locked)
		})
	}
}
