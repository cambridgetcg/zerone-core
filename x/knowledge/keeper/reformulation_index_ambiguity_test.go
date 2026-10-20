package keeper_test

import (
	"fmt"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/zerone-chain/zerone/x/knowledge/keeper"
	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// BLOCKER for the indexed candidate: the existing 0x65 key encoding has no
// escape/length boundary for bounty IDs. These IDs are accepted by the real
// message handlers, even after the 0x90 marker is enabled. This desired-behavior
// regression deliberately remains RED until the identifier/index contract has
// an authorized solution. Do not remove the assertion merely to publish.
func TestReformulationIndexBountyDelimiterIsolation(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		t.Run(fmt.Sprint(enabled), func(t *testing.T) {
			k, ctx, bank, staking := newKnowledgeOrderFixture(t)
			params, err := k.GetParams(ctx)
			require.NoError(t, err)
			params.ReformulationMinPanelVotes = 1
			require.NoError(t, k.SetParams(ctx, params))
			if enabled {
				require.NoError(t, k.EnableReformulationVindicationOrdering(ctx))
			}
			sponsor := sdk.AccAddress([]byte("nul-sponsor-000000001"))
			author := sdk.AccAddress([]byte("nul-author-0000000010"))
			coins := sdk.NewCoins(sdk.NewInt64Coin("uzrn", 3000))
			require.NoError(t, bank.MintCoins(ctx, orderTestMint, coins))
			require.NoError(t, bank.SendCoinsFromModuleToAccount(ctx, orderTestMint, sponsor, coins))
			require.NoError(t, k.SetFact(ctx, &types.Fact{Id: "original", Submitter: author.String()}))
			server := keeper.NewMsgServerImpl(k)
			for _, id := range []string{"pool", "pool\x00evil"} {
				_, err := server.CreateAugmentationBounty(ctx, &types.MsgCreateAugmentationBounty{Id: id, Sponsor: sponsor.String(), TargetFactId: "original", RewardPerVariant: 1000, MaxVariants: 1})
				require.NoError(t, err, "existing admission permits the exact bounty identifier %q", id)
			}
			for _, pair := range [][2]string{{"target", "pool"}, {"variant", "pool\x00evil"}} {
				_, err := server.SubmitAugmentation(ctx, &types.MsgSubmitAugmentation{Id: pair[0], BountyId: pair[1], OriginalFactId: "original", Submitter: author.String(), VariantContent: "synthetic variant"})
				require.NoError(t, err)
			}
			holds, err := k.CensusReformulationHolds(ctx)
			require.NoError(t, err, "the candidate's full census and index-completeness check accept this state")
			require.Empty(t, holds)
			staking.addValidator("voter", 100_000, "bonded")
			cached, write := ctx.CacheContext()
			response, err := server.VoteOnAugmentation(cached, &types.MsgVoteOnAugmentation{AugmentationId: "target", Verifier: "voter", Vote: 1})
			t.Logf("AW118b_DELIMITER enabled=%t holds=%d vote_error=%v", enabled, len(holds), err)
			require.NoError(t, err, "an unrelated bounty must not freeze the target pool")
			require.True(t, response.VerdictFinalized)
			write()
			require.EqualValues(t, 1000, bank.GetBalance(ctx, author, "uzrn").Amount.Int64())
		})
	}
}
