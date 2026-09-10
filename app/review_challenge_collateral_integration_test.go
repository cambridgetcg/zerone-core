package app

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	knowledgekeeper "github.com/zerone-chain/zerone/x/knowledge/keeper"
	knowledgetypes "github.com/zerone-chain/zerone/x/knowledge/types"
	"google.golang.org/protobuf/proto"
)

// Exact SDK bank/cache behavior for an inherited policy-0 obligation. The
// imported test verdict is synthetic; this does not simulate independent science.
func TestReviewNeutralityLegacyCollateralFailureRemainsRetryable(t *testing.T) {
	for _, tc := range []struct {
		name    string
		funding int64
	}{{"refund", 1499}, {"treasury-after-refund", 4499}} {
		t.Run(tc.name, func(t *testing.T) {
			app, ctx := settlementFixture(t)
			require.NoError(t, app.KnowledgeKeeper.EnableReviewNeutrality(ctx))
			challenger := settlementAddress(81)
			original := &knowledgetypes.Fact{Id: "collateral-target", ClaimId: "target-author-claim", Submitter: challenger.String(), Domain: "physics", Category: "empirical", Status: knowledgetypes.FactStatus_FACT_STATUS_CHALLENGED}
			require.NoError(t, app.KnowledgeKeeper.SetFact(ctx, original))
			claim := &knowledgetypes.Claim{Id: "legacy-collateral", Submitter: challenger.String(), Domain: "physics", Category: "empirical", Stake: "10000", ProvisionalFactId: original.Id, Status: knowledgetypes.ClaimStatus_CLAIM_STATUS_IN_VERIFICATION}
			require.NoError(t, app.KnowledgeKeeper.SetClaim(ctx, claim))
			round := &knowledgetypes.VerificationRound{Id: "legacy-collateral-round", ClaimId: claim.Id, Phase: knowledgetypes.VerificationPhase_VERIFICATION_PHASE_AGGREGATION}
			require.NoError(t, app.KnowledgeKeeper.SetVerificationRound(ctx, round))
			fundSettlementFixture(t, app, ctx, tc.funding)
			beforeRound := proto.Clone(round)
			beforeClaim := proto.Clone(claim)
			beforeFact := proto.Clone(original)
			before := app.BankKeeper.GetBalance(ctx, app.AccountKeeper.GetModuleAddress(knowledgetypes.ModuleName), "uzrn")
			events := len(ctx.EventManager().Events())
			result := &knowledgekeeper.VerificationResult{Verdict: knowledgetypes.Verdict_VERDICT_REJECT}
			require.ErrorContains(t, app.KnowledgeKeeper.CompleteRound(ctx, round, result), "challenge collateral")
			currentRound, found := app.KnowledgeKeeper.GetVerificationRound(ctx, round.Id)
			require.True(t, found)
			currentClaim, found := app.KnowledgeKeeper.GetClaim(ctx, claim.Id)
			require.True(t, found)
			currentFact, found := app.KnowledgeKeeper.GetFact(ctx, original.Id)
			require.True(t, found)
			require.True(t, proto.Equal(beforeRound, currentRound))
			require.True(t, proto.Equal(beforeClaim, currentClaim))
			require.True(t, proto.Equal(beforeFact, currentFact))
			require.Equal(t, before, app.BankKeeper.GetBalance(ctx, app.AccountKeeper.GetModuleAddress(knowledgetypes.ModuleName), "uzrn"))
			require.True(t, app.BankKeeper.GetBalance(ctx, challenger, "uzrn").IsZero())
			require.Len(t, ctx.EventManager().Events(), events)
			fundSettlementFixture(t, app, ctx, 4500-tc.funding)
			require.NoError(t, app.KnowledgeKeeper.CompleteRound(ctx, currentRound, result))
			after, found := app.KnowledgeKeeper.GetVerificationRound(ctx, round.Id)
			require.True(t, found)
			require.Equal(t, knowledgetypes.VerificationPhase_VERIFICATION_PHASE_COMPLETE, after.Phase)
			require.Zero(t, after.ReviewPolicyVersion)
			require.Equal(t, sdk.NewInt64Coin("uzrn", 1500), app.BankKeeper.GetBalance(ctx, challenger, "uzrn"))
			require.NoError(t, app.KnowledgeKeeper.CompleteRound(ctx, after, result))
			require.Equal(t, sdk.NewInt64Coin("uzrn", 1500), app.BankKeeper.GetBalance(ctx, challenger, "uzrn"))
		})
	}
}
