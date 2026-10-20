package keeper_test

import (
	"fmt"
	"testing"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/zerone-chain/zerone/internal/activation"
	"github.com/zerone-chain/zerone/x/knowledge/keeper"
	"github.com/zerone-chain/zerone/x/knowledge/types"
)

func TestReformulationHeldEscrowLifecycle(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		for _, invalid := range []bool{false, true} {
			t.Run(fmt.Sprintf("enabled=%t/invalid=%t", enabled, invalid), func(t *testing.T) {
				k, ctx, bank, staking := newKnowledgeOrderFixture(t)
				params, err := k.GetParams(ctx)
				require.NoError(t, err)
				params.ReformulationMinPanelVotes = 2
				params.ReformulationConsensusBps = 500_000
				require.NoError(t, k.SetParams(ctx, params))
				sponsor := sdk.AccAddress([]byte("held-sponsor-00000001"))
				submitter := sdk.AccAddress([]byte("held-author-000000001"))
				coins := sdk.NewCoins(sdk.NewInt64Coin("uzrn", 1500))
				require.NoError(t, bank.MintCoins(ctx, orderTestMint, coins))
				require.NoError(t, bank.SendCoinsFromModuleToAccount(ctx, orderTestMint, sponsor, coins))
				require.NoError(t, k.EscrowAugmentationBounty(ctx, sponsor.String(), "bounty", sdkmath.NewInt(1500)))
				require.NoError(t, k.SetAugmentationBounty(ctx, &types.AugmentationBounty{Id: "bounty", SponsorAddress: sponsor.String(), TargetFactId: "original", Active: true, RewardPerVariant: 1000, MaxVariants: 1, ExpiresAtBlock: 101}))
				aug := &types.Augmentation{Id: "retained", OriginalFactId: "original", BountyId: "bounty", Submitter: submitter.String(), VerdictVoters: []string{"a", "b"}, VerdictVotes: []types.AugmentationVerdict{1, 4}, VerdictVoteStakes: []uint64{100_000, 100_000}}
				reason := "tied_top_weight"
				if invalid {
					aug.VerdictVotes[0] = 7
					reason = "invalid_retained_vote"
				}
				require.NoError(t, k.SetAugmentation(ctx, aug))
				if enabled {
					require.NoError(t, k.EnableReformulationVindicationOrdering(ctx))
				}
				holds, err := k.CensusReformulationHolds(ctx)
				require.NoError(t, err)
				require.Equal(t, []keeper.ReformulationHold{{AugmentationID: "retained", OriginalFactID: "original", BountyID: "bounty", Reason: reason}}, holds)
				ctx = ctx.WithBlockHeight(102).WithEventManager(sdk.NewEventManager())
				require.NoError(t, k.ProcessRouteBLifecycle(ctx))
				stored, found := k.GetAugmentation(ctx, aug.Id)
				require.True(t, found)
				require.True(t, proto.Equal(aug, stored), "heartbeat never retallies into a verdict or rewrites retained ballots")
				bounty, found := k.GetAugmentationBounty(ctx, "bounty")
				require.True(t, found)
				require.Equal(t, enabled, bounty.Active)
				if !enabled {
					require.Zero(t, k.GetEscrowLocked(ctx, "bounty").Int64())
					require.EqualValues(t, 1455, bank.GetBalance(ctx, sponsor, "uzrn").Amount.Int64(), "the predecessor DOES have automatic bounty expiry")
					for _, event := range ctx.EventManager().Events() {
						require.NotEqual(t, "zerone.knowledge.reformulation_held", event.Type)
					}
					return
				}
				require.EqualValues(t, 1500, k.GetEscrowLocked(ctx, "bounty").Int64())
				require.Zero(t, bank.GetBalance(ctx, sponsor, "uzrn").Amount.Int64())
				require.EqualValues(t, 1500, bank.GetBalance(ctx, authtypes.NewModuleAddress(types.TrainingFundModuleName), "uzrn").Amount.Int64())
				require.Empty(t, ctx.EventManager().Events(), "continuing holds never re-emit on a block")
				require.NoError(t, k.ProcessRouteBLifecycle(ctx.WithBlockHeight(103)))
				require.Empty(t, ctx.EventManager().Events(), "nor on the following block")
				// Pooled backing cannot escape via a sibling payout or sponsor veto.
				require.ErrorContains(t, k.ReleaseAugmentationPayout(ctx, "bounty", submitter.String(), sdkmath.NewInt(1)), "escrow held")
				require.ErrorContains(t, k.ForfeitAugmentationDeposit(ctx, "bounty", sdkmath.NewInt(1)), "escrow held")
				staking.addValidator("new-voter", 200_000, "bonded")
				ctx = ctx.WithEventManager(sdk.NewEventManager())
				response, err := keeper.NewMsgServerImpl(k).VoteOnAugmentation(ctx, &types.MsgVoteOnAugmentation{AugmentationId: "retained", Verifier: "new-voter", Vote: 1})
				if invalid {
					require.ErrorContains(t, err, "invalid retained vote")
					require.Nil(t, response)
					require.Empty(t, ctx.EventManager().Events(), "failed transactions do not supply durable hold events")
					require.EqualValues(t, 1500, k.GetEscrowLocked(ctx, "bounty").Int64())
				} else {
					require.NoError(t, err)
					require.True(t, response.VerdictFinalized)
					require.EqualValues(t, 1000, bank.GetBalance(ctx, submitter, "uzrn").Amount.Int64())
					holds, err = k.CensusReformulationHolds(ctx)
					require.NoError(t, err)
					require.Empty(t, holds, "new weight can break a retained tie")
				}
				require.EqualValues(t, 1500, bank.GetSupply(ctx, "uzrn").Amount.Int64())
			})
		}
	}
}

func TestReformulationSiblingTiesCanMakeProgress(t *testing.T) {
	k, ctx, bank, staking := newKnowledgeOrderFixture(t)
	params, err := k.GetParams(ctx)
	require.NoError(t, err)
	params.ReformulationMinPanelVotes = 2
	params.ReformulationConsensusBps = 500_000
	require.NoError(t, k.SetParams(ctx, params))
	sponsor := sdk.AccAddress([]byte("sibling-sponsor-00001"))
	author := sdk.AccAddress([]byte("sibling-author-000001"))
	coins := sdk.NewCoins(sdk.NewInt64Coin("uzrn", 3000))
	require.NoError(t, bank.MintCoins(ctx, orderTestMint, coins))
	require.NoError(t, bank.SendCoinsFromModuleToAccount(ctx, orderTestMint, sponsor, coins))
	require.NoError(t, k.EscrowAugmentationBounty(ctx, sponsor.String(), "pool", sdkmath.NewInt(3000)))
	require.NoError(t, k.SetAugmentationBounty(ctx, &types.AugmentationBounty{Id: "pool", SponsorAddress: sponsor.String(), Active: true, RewardPerVariant: 1000, MaxVariants: 2}))
	for _, id := range []string{"a", "b"} {
		require.NoError(t, k.SetAugmentation(ctx, &types.Augmentation{Id: id, BountyId: "pool", Submitter: author.String(), VerdictVoters: []string{"v1", "v2"}, VerdictVotes: []types.AugmentationVerdict{1, 2}, VerdictVoteStakes: []uint64{100_000, 100_000}}))
	}
	require.NoError(t, k.EnableReformulationVindicationOrdering(ctx))
	staking.addValidator("breaker", 200_000, "bonded")
	server := keeper.NewMsgServerImpl(k)
	vote := func(id, voter string, wantFinal bool) {
		cached, write := ctx.CacheContext()
		response, err := server.VoteOnAugmentation(cached, &types.MsgVoteOnAugmentation{AugmentationId: id, Verifier: voter, Vote: 1})
		require.NoError(t, err, "a peer hold must not roll back the ballot that breaks this tie")
		require.Equal(t, wantFinal, response.VerdictFinalized)
		write()
	}
	vote("a", "breaker", false)
	stored, found := k.GetAugmentation(ctx, "a")
	require.True(t, found)
	require.Len(t, stored.VerdictVoters, 3)
	require.Empty(t, keeper.ReformulationHoldReason(stored, params))
	require.EqualValues(t, 3000, k.GetEscrowLocked(ctx, "pool").Int64())
	vote("b", "breaker", true)
	// A new zero-weight voter re-tallies the now-unblocked first record.
	vote("a", "after-peer-resolves", true)
	require.EqualValues(t, 2000, bank.GetBalance(ctx, author, "uzrn").Amount.Int64())
	require.EqualValues(t, 1000, k.GetEscrowLocked(ctx, "pool").Int64())
}

func TestReformulationHoldCensusRefusesCorruption(t *testing.T) {
	for _, corruptMarker := range []bool{false, true} {
		t.Run(fmt.Sprint(corruptMarker), func(t *testing.T) {
			f := newKnowledgeStoreFixture(t, "hold-census")
			require.NoError(t, f.k.EnableReformulationVindicationOrdering(f.ctx))
			if corruptMarker {
				f.ctx.KVStore(f.key).Set([]byte(keeper.ReformulationVindicationOrderingEnabledStoreKey), []byte{2})
			} else {
				require.NoError(t, f.k.SetAugmentationBounty(f.ctx, &types.AugmentationBounty{Id: "pool", Active: true, ExpiresAtBlock: 1}))
				require.NoError(t, f.k.SetAugmentation(f.ctx, &types.Augmentation{Id: "bad", BountyId: "pool"}))
				f.ctx.KVStore(f.key).Set(types.AugmentationKey("bad"), []byte{0xff})
			}
			before := relationContractSnapshot(t, f.ctx, f.key)
			f.ctx = f.ctx.WithEventManager(sdk.NewEventManager())
			err := f.k.ProcessRouteBLifecycle(f.ctx)
			require.Equal(t, before, relationContractSnapshot(t, f.ctx, f.key))
			if corruptMarker {
				require.ErrorIs(t, err, activation.ErrMarkerFault)
				require.Empty(t, f.ctx.EventManager().Events(), "a marker fault propagates, not log-and-skip")
			} else {
				require.NoError(t, err, "ordinary lifecycle errors are not new halts")
				require.Len(t, f.ctx.EventManager().Events(), 1)
				require.Equal(t, "zerone.knowledge.reformulation_processing_refused", f.ctx.EventManager().Events()[0].Type)
			}
		})
	}
}
