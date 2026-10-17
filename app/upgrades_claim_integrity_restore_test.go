package app

import (
	"encoding/binary"
	"testing"

	"cosmossdk.io/core/header"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	knowledgekeeper "github.com/zerone-chain/zerone/x/knowledge/keeper"
	knowledgetypes "github.com/zerone-chain/zerone/x/knowledge/types"
)

// knowledge-claim-integrity-v1 restores, at the upgrade height, a fact edge
// that an equal-ID augmentation replaced before activation (approved by Yu on
// 2026-10-17; spec §1.5). F was accepted from a claim naming D; an
// augmentation of D whose ID is F's then passed through the real verdict
// writer with the marker off and replaced F's edge. Across ApplyUpgrade every
// byte of every store is unchanged except 0x87, its receipt, x/upgrade's own
// bookkeeping and the two entries of that edge, which hold F's own edge again.
// The keeper tests prove the restored bytes equal those the real round writer
// (createFactFromClaim) produced; here F's edge is written with the fields
// that writer sets.
func TestClaimIntegrityUpgradeRestoresEdgeReplacedByEqualIDAugmentation(t *testing.T) {
	app, ctx, _ := claimIntegritySourceFixture(t)
	k := app.KnowledgeKeeper
	const created, verdict = 3, 5
	submitter := settlementAddress(215).String()
	factD := &knowledgetypes.Fact{Id: "app-restore-d", Domain: "physics", Category: "empirical", Status: knowledgetypes.FactStatus_FACT_STATUS_VERIFIED, Content: "Synthetic fact D, which fact F requires", Submitter: settlementAddress(216).String()}
	claim := &knowledgetypes.Claim{Id: "app-restore-claim", VerificationRoundId: "app-restore-round", FactContent: "Synthetic fact F, which requires fact D", Domain: "physics", Stake: "100", Submitter: submitter,
		Relations: []*knowledgetypes.ClaimRelation{{Relation: knowledgetypes.RelationType_RELATION_TYPE_REQUIRES, TargetFactId: factD.Id}}}
	factF := &knowledgetypes.Fact{Id: knowledgekeeper.GenerateFactID(claim.Id, created), ClaimId: claim.Id, VerifiedAtBlock: created, Domain: "physics", Category: "empirical", Status: knowledgetypes.FactStatus_FACT_STATUS_VERIFIED, Content: claim.FactContent, Submitter: submitter}
	require.NoError(t, k.SetClaim(ctx, claim))
	require.NoError(t, k.SetFact(ctx, factD))
	require.NoError(t, k.SetFact(ctx, factF))
	require.NoError(t, k.SetFactRelation(ctx, &knowledgetypes.FactRelation{SourceFactId: factF.Id, TargetFactId: factD.Id, Relation: knowledgetypes.RelationType_RELATION_TYPE_REQUIRES, CreatedAtBlock: created, Creator: submitter}))
	store := ctx.KVStore(app.keys[knowledgetypes.StoreKey])
	forwardKey, reverseKey := knowledgetypes.FactRelationKey(factF.Id, factD.Id), knowledgetypes.FactRelationReverseKey(factD.Id, factF.Id)
	own := store.Get(forwardKey)
	require.NotEmpty(t, own)

	require.NoError(t, k.SetAugmentation(ctx, &knowledgetypes.Augmentation{Id: factF.Id, OriginalFactId: factD.Id, VariantContent: "Synthetic variant of fact D"}))
	require.NoError(t, k.ApplyFinalizedAugmentationVerdict(ctx.WithBlockHeight(verdict), factF.Id, knowledgetypes.AugmentationVerdict_AUGMENTATION_VERDICT_EQUIVALENT))
	replaced := store.Get(forwardKey)
	require.NotEqual(t, own, replaced, "the verdict replaced F's own edge")
	require.Equal(t, replaced, store.Get(reverseKey))

	latest := app.CommitMultiStore().Commit().Version
	plan := upgradetypes.Plan{Name: UpgradeNameKnowledgeClaimIntegrityV1, Height: latest + 2}
	require.NoError(t, app.UpgradeKeeper.ScheduleUpgrade(ctx, plan))
	require.NoError(t, app.UpgradeKeeper.DumpUpgradeInfoToDisk(plan.Height, plan))
	require.Equal(t, plan.Height-1, app.CommitMultiStore().Commit().Version)
	require.NoError(t, app.ValidateAccountingAuthorityStartup())
	before := claimIntegrityAllStores(t, app, ctx)
	ctx = ctx.WithBlockHeight(plan.Height).WithHeaderInfo(header.Info{Height: plan.Height, ChainID: ctx.ChainID()}).WithEventManager(sdk.NewEventManager())
	cache, write := ctx.CacheContext()
	require.NoError(t, app.UpgradeKeeper.ApplyUpgrade(cache, plan))
	require.Equal(t, before, claimIntegrityAllStores(t, app, ctx), "outer block cache owns the upgrade")
	write()

	after := claimIntegrityAllStores(t, app, ctx)
	row := func(key []byte) string { return knowledgetypes.StoreKey + ":" + string(key) }
	require.Equal(t, own, after[row(forwardKey)], "F's own edge again")
	require.Equal(t, own, after[row(reverseKey)])
	require.Equal(t, []byte{1}, after[claimIntegrityMarkerRow()])
	require.Equal(t, []byte("true"), after[claimIntegrityReceiptRow()])
	upgradeRow := func(key []byte) string { return upgradetypes.StoreKey + ":" + string(key) }
	doneKey := make([]byte, 9+len(plan.Name))
	doneKey[0] = upgradetypes.DoneByte
	binary.BigEndian.PutUint64(doneKey[1:9], uint64(plan.Height))
	copy(doneKey[9:], plan.Name)
	for _, rows := range []map[string][]byte{before, after} {
		for _, key := range []string{
			row(forwardKey), row(reverseKey), claimIntegrityMarkerRow(), claimIntegrityReceiptRow(),
			upgradeRow(upgradetypes.PlanKey()), upgradeRow(doneKey),
			upgradeRow(append([]byte{upgradetypes.VersionMapByte}, "knowledge"...)), upgradeRow([]byte{upgradetypes.ProtocolVersionByte}),
		} {
			delete(rows, key)
		}
	}
	require.Equal(t, before, after, "nothing else changes in any store")

	var restored []map[string]string
	for _, event := range ctx.EventManager().Events() {
		if event.Type != "zerone.knowledge.augmentation_edge_restored" {
			continue
		}
		attrs := map[string]string{}
		for _, attr := range event.Attributes {
			attrs[attr.Key] = attr.Value
		}
		restored = append(restored, attrs)
	}
	require.Equal(t, []map[string]string{{"fact_id": factF.Id, "target_id": factD.Id, "augmentation_id": factF.Id}}, restored, "the upgrade block reports the restored pair")
	require.Equal(t, plan.Height, app.CommitMultiStore().Commit().Version)
	require.NoError(t, app.ValidateAccountingAuthorityStartup(), "restart at the applied target is accepted")
}
