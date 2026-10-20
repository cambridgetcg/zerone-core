package cross_stack_test

import (
	"fmt"
	"testing"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	knowledgekeeper "github.com/zerone-chain/zerone/x/knowledge/keeper"
	knowledgetypes "github.com/zerone-chain/zerone/x/knowledge/types"
)

// Real application-registered BeginBlock, cached SDK stores and real bank;
// retained records/issuance are synthetic, not a full replay or signed tx.
func TestReformulationHoldAppLifecycle(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		t.Run(fmt.Sprint(enabled), func(t *testing.T) {
			h := NewTestHarness(t)
			if !enabled {
				useLegacyReviewPolicy(t, h)
			}
			on, err := h.KnowledgeKeeper.ReformulationVindicationOrderingEnabled(h.Ctx)
			require.NoError(t, err)
			require.Equal(t, enabled, on)
			sponsor := testAddr("reformulation-hold-sponsor")
			require.NoError(t, h.FundAccount(sponsor, sdk.NewCoins(sdk.NewInt64Coin("uzrn", 1500))))
			require.NoError(t, h.KnowledgeKeeper.EscrowAugmentationBounty(h.Ctx, sponsor.String(), "held-pool", sdkmath.NewInt(1500)))
			require.NoError(t, h.KnowledgeKeeper.SetAugmentationBounty(h.Ctx, &knowledgetypes.AugmentationBounty{Id: "held-pool", SponsorAddress: sponsor.String(), Active: true, ExpiresAtBlock: 2}))
			retained := &knowledgetypes.Augmentation{Id: "legacy-invalid", BountyId: "held-pool", Submitter: testAddr("author").String(), VerdictVoters: []string{"retained-voter"}, VerdictVotes: []knowledgetypes.AugmentationVerdict{7}}
			require.NoError(t, h.KnowledgeKeeper.SetAugmentation(h.Ctx, retained))
			h.AdvanceBlocks(1)
			bounty, found := h.KnowledgeKeeper.GetAugmentationBounty(h.Ctx, "held-pool")
			require.True(t, found)
			require.Equal(t, enabled, bounty.Active)
			stored, found := h.KnowledgeKeeper.GetAugmentation(h.Ctx, "legacy-invalid")
			require.True(t, found)
			require.True(t, proto.Equal(retained, stored))
			if enabled {
				require.EqualValues(t, 1500, h.KnowledgeKeeper.GetEscrowLocked(h.Ctx, "held-pool").Int64())
				require.Zero(t, h.GetBalance(sponsor, "uzrn").Amount.Int64())
				cached, _ := h.Ctx.CacheContext()
				_, err := knowledgekeeper.NewMsgServerImpl(h.KnowledgeKeeper).VoteOnAugmentation(cached, &knowledgetypes.MsgVoteOnAugmentation{AugmentationId: retained.Id, Verifier: "new-voter", Vote: 1})
				require.ErrorContains(t, err, "invalid retained vote")
			} else {
				require.Zero(t, h.KnowledgeKeeper.GetEscrowLocked(h.Ctx, "held-pool").Int64())
				require.EqualValues(t, 1455, h.GetBalance(sponsor, "uzrn").Amount.Int64())
			}
		})
	}
}
