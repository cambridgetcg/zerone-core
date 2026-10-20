package keeper_test

import (
	"fmt"
	"testing"

	"cosmossdk.io/store/rootmulti"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/zerone-chain/zerone/x/knowledge/keeper"
	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// zc-reformulation-id-delimiter retains OFF admission and requires NUL-free
// prospective IDs ON. Readable foreign legacy index entries do not hold this
// bounty. The carrier separately refuses any retained NUL ID before activation;
// this fixture is not permission to enable the marker over ambiguous state.
func TestReformulationIndexBountyDelimiterIsolation(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		t.Run(fmt.Sprint(enabled), func(t *testing.T) {
			k, ctx, bank, staking := newKnowledgeOrderFixture(t)
			initReformulationAdmissionGenesis(t, k, ctx)
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
			_, err = server.CreateAugmentationBounty(ctx, &types.MsgCreateAugmentationBounty{Id: "pool", Sponsor: sponsor.String(), TargetFactId: "original", RewardPerVariant: 1000, MaxVariants: 1})
			require.NoError(t, err)
			before := reformulationAdmissionSnapshot(t, ctx)
			ctx = ctx.WithEventManager(sdk.NewEventManager())
			_, err = server.CreateAugmentationBounty(ctx, &types.MsgCreateAugmentationBounty{Id: "pool\x00evil", Sponsor: sponsor.String(), TargetFactId: "original", RewardPerVariant: 1000, MaxVariants: 1})
			if enabled {
				require.EqualError(t, err, "bounty id must not contain NUL")
				require.Equal(t, before, reformulationAdmissionSnapshot(t, ctx))
				require.Empty(t, ctx.EventManager().Events())
			} else {
				require.NoError(t, err, "OFF retains NUL bounty admission")
			}
			_, err = server.SubmitAugmentation(ctx, &types.MsgSubmitAugmentation{Id: "target", BountyId: "pool", OriginalFactId: "original", Submitter: author.String(), VariantContent: "synthetic variant"})
			require.NoError(t, err)
			if enabled {
				// A readable record belongs to another pool, even when a retained
				// index entry places it under this prefix. Its invalid ballot must
				// neither hold nor error the target's finalization/payout.
				require.NoError(t, k.SetAugmentation(ctx, &types.Augmentation{Id: "foreign", BountyId: "other", VerdictVotes: []types.AugmentationVerdict{7}}))
				store := ctx.MultiStore().(*rootmulti.Store).GetStoreByName(types.StoreKey).(storetypes.KVStore)
				store.Set(types.AugmentationByBountyKey("pool", "foreign"), []byte{1})
			} else {
				_, err = server.SubmitAugmentation(ctx, &types.MsgSubmitAugmentation{Id: "variant", BountyId: "pool\x00evil", OriginalFactId: "original", Submitter: author.String(), VariantContent: "synthetic variant"})
				require.NoError(t, err)
			}
			holds, err := k.CensusReformulationHolds(ctx)
			if enabled {
				require.NoError(t, err)
				require.Equal(t, []keeper.ReformulationHold{{AugmentationID: "foreign", BountyID: "other", Reason: "invalid_retained_vote"}}, holds)
			} else {
				require.ErrorContains(t, err, `bounty id "pool\x00evil" contains NUL`)
				require.Nil(t, holds, "the prospective census holds activation, not legacy transactions")
			}
			staking.addValidator("voter", 100_000, "bonded")
			cached, write := ctx.CacheContext()
			response, err := server.VoteOnAugmentation(cached, &types.MsgVoteOnAugmentation{AugmentationId: "target", Verifier: "voter", Vote: 1})
			t.Logf("AW118c_DELIMITER enabled=%t vote_error=%v", enabled, err)
			require.NoError(t, err, "an unrelated bounty must not freeze the target pool")
			require.True(t, response.VerdictFinalized)
			write()
			require.EqualValues(t, 1000, bank.GetBalance(ctx, author, "uzrn").Amount.Int64())
		})
	}
}
