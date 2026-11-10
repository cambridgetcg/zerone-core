package app

import (
	"encoding/json"
	"fmt"
	"testing"

	"cosmossdk.io/core/appmodule"
	"cosmossdk.io/core/header"
	sdkmath "cosmossdk.io/math"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	alignment "github.com/zerone-chain/zerone/x/alignment/types"
)

// dev1CensusInputs are zerone-dev-1's alignment inputs as its 2026-10-28 copied
// census measured them at h3,828,519 (C-dev1 drill census.json): bank supply
// 3,000,000,000,000 uzrn, mint ledger "0", an empty zerone_staking registry and
// 7 of 7 domains active. That census refused consensus-repairs-v1 with the Q1
// blocker; this source must pass on the same inputs.
const dev1CensusInputs = "bank_supply=3000000000000 mint_ledger=0 registry=0 active=0 active_stake=0 domains=7 active_domains=7 legacy_staking_ratio_bps=500000 fixed_staking_ratio_bps=0"

// setDev1AlignmentInputs gives a consensus-repairs fixture dev-1's measured
// alignment inputs. The bank supply row is synthetic (no balances back it); the
// census and the alignment sensors read only the supply total.
func setDev1AlignmentInputs(t *testing.T, app *ZeroneApp, ctx sdk.Context) {
	t.Helper()
	registry := ctx.KVStore(app.keys["zerone_staking"])
	iterator := registry.Iterator([]byte{1}, []byte{2})
	var keys [][]byte
	for ; iterator.Valid(); iterator.Next() {
		keys = append(keys, append([]byte(nil), iterator.Key()...))
	}
	require.NoError(t, iterator.Close())
	for _, key := range keys {
		registry.Delete(key)
	}
	ctx.KVStore(app.keys["vesting_rewards"]).Set([]byte{0x0b}, []byte("0"))
	raw, err := sdk.IntValue.Encode(sdkmath.NewInt(3_000_000_000_000))
	require.NoError(t, err)
	ctx.KVStore(app.keys["bank"]).Set([]byte("\x00uzrn"), raw)
	// dev-1 holds no consensus-repairs-v2 marker (Q1 0x09, bank supply 0x0c, ...);
	// native genesis in these fixtures selects them all.
	for _, loc := range consensusRepairsV2MarkerLocations() {
		ctx.KVStore(app.keys[loc.store]).Delete(loc.key)
	}
}

func TestConsensusRepairsV1PassesOnDev1CensusInputs(t *testing.T) {
	app, ctx, db := consensusRepairsFixture(t)
	setDev1AlignmentInputs(t, app, ctx)
	census, err := app.CensusConsensusRepairsV1(ctx)
	require.NoError(t, err, "the 2026-10-28 dev-1 census blocker must not recur")
	raw, err := json.Marshal(census)
	require.NoError(t, err)
	require.Contains(t, string(raw), dev1CensusInputs)
	for _, finding := range census.Findings {
		require.Equal(t, "held", finding.Outcome, finding.Reason)
	}
	legacy := app.AlignmentKeeper.ObserveAll(ctx)
	require.Equal(t, alignment.NeutralBPS, legacy.EconomicStability)
	require.Equal(t, alignment.NeutralBPS, legacy.StakingRatio)

	// The carrier's startup validation at H-1, then the upgrade at H.
	latest := app.CommitMultiStore().Commit().Version
	plan := upgradetypes.Plan{Name: UpgradeNameConsensusRepairsV1, Height: latest + 1}
	require.NoError(t, app.UpgradeKeeper.ScheduleUpgrade(ctx, plan))
	require.NoError(t, app.UpgradeKeeper.DumpUpgradeInfoToDisk(plan.Height, plan))
	require.NoError(t, app.ValidateAccountingAuthorityStartup(), "carrier startup at H-1")
	ctx = ctx.WithBlockHeight(plan.Height).WithHeaderInfo(header.Info{Height: plan.Height, ChainID: ctx.ChainID()})
	require.NoError(t, app.UpgradeKeeper.ApplyUpgrade(ctx, plan))
	for _, marker := range app.ConsensusRepairsV1Markers() {
		on, err := marker.Enabled(ctx)
		require.NoError(t, err)
		require.True(t, on, marker.Module+"."+marker.Name)
	}
	on, err := app.VestingRewardsKeeper.AlignmentBankSupplyEnabled(ctx)
	require.NoError(t, err)
	require.False(t, on, "v1 must not switch the alignment denominator")

	// Every alignment input reads as before: neutral stays neutral.
	after := app.AlignmentKeeper.ObserveAll(ctx)
	require.Equal(t, legacy.EconomicStability, after.EconomicStability)
	require.Equal(t, legacy.StakingRatio, after.StakingRatio)
	require.Equal(t, legacy.KnowledgeQuality, after.KnowledgeQuality)
	require.Equal(t, legacy.GovernanceParticipation, after.GovernanceParticipation)
	require.Equal(t, legacy.NetworkSecurity, after.NetworkSecurity)

	// The applied target restarts.
	app.CommitMultiStore().Commit()
	require.NoError(t, app.ValidateAccountingAuthorityStartup())
	_ = db
}

// The bank-supply denominator is a consensus-repairs-v2 selection, listed
// directly after Q1, so native genesis can select it only together with Q1.
func TestConsensusRepairsV2SelectsBankSupplyWithQ1(t *testing.T) {
	require.Empty(t, ConsensusRepairsDeferredGenesisSelections())
	selections := ConsensusRepairsV2GenesisSelections()
	require.Equal(t, ActivationSelection{"alignment", "unobserved_health_enabled"}, selections[0])
	require.Equal(t, ActivationSelection{"vesting_rewards", "alignment_bank_supply_enabled"}, selections[1])
	for _, selection := range ConsensusRepairsV1GenesisSelections() {
		require.NotEqual(t, "alignment_bank_supply_enabled", selection.Name)
	}
}

// dev-1's live alignment state (GET abci_query at h4,559,478 on 2026-11-05):
// previous_category degraded, degraded_frequency_active false, latest health
// index composite 249,800, pacing 750,000/1,500,000.
func setDev1AlignmentState(t *testing.T, app *ZeroneApp, ctx sdk.Context) {
	t.Helper()
	state := app.AlignmentKeeper.GetState(ctx)
	state.PreviousCategory = alignment.CategoryDegraded
	state.DegradedFrequencyActive = false
	app.AlignmentKeeper.SetState(ctx, state)
	app.AlignmentKeeper.SetHealthIndex(ctx, &alignment.AlignmentHealthIndex{Height: uint64(ctx.BlockHeight()), CompositeScore: 249_800, Category: alignment.CategoryDegraded})
	creation, analysis := app.AlignmentKeeper.GetGlobalPacingMultiplier(ctx)
	require.Equal(t, [2]uint64{750_000, 1_500_000}, [2]uint64{creation, analysis})
}

// F5: on dev-1's inputs v2 switches Q1 and the bank-supply denominator
// together. Its census must say what that does to pacing, fees and gov voting,
// and the chain must then do exactly that at its first scheduled observation.
func TestConsensusRepairsV2ReportsAndCausesDev1PacingShift(t *testing.T) {
	a, ctx, _ := consensusRepairsV2Fixture(t)
	setDev1AlignmentInputs(t, a, ctx)
	setDev1AlignmentState(t, a, ctx)
	params, err := a.KnowledgeKeeper.GetParams(ctx)
	require.NoError(t, err)
	baseFee := params.MinReviewFee
	degradedFee := a.KnowledgeKeeper.GetEffectiveMinReviewFee(ctx)
	require.NotEqual(t, baseFee, degradedFee, "degraded pacing raises the fee today")
	voting := a.ZeroneGovKeeper.GetParams(ctx).VotingPeriodBlocks

	ctx, plan := scheduleRepairsV2(t, a, ctx)
	for _, strict := range []bool{false, true} {
		report, _, err := a.censusConsensusRepairsV2(ctx, strict)
		require.NoError(t, err)
		var shift string
		for _, finding := range report.Findings {
			if finding.Census == "q1_health" && finding.Module == "alignment" {
				require.Equal(t, "held", finding.Outcome)
				shift = finding.Reason
			}
		}
		t.Logf("q1_health: %s", shift)
		for _, want := range []string{
			"category degraded->not_observed composite 249800->",
			"creation_pacing_bps 750000->1000000 analysis_pacing_bps 1500000->1000000",
			fmt.Sprintf("min_review_fee %s->%s", degradedFee, baseFee),
			fmt.Sprintf("voting_period_blocks %d->%d (expedited true->false; gov read degraded before)", max(voting/2, 1), voting),
			"inputs registered=0 active=0 active_stake=0 mint_ledger=0 bank_supply=3000000000000 legacy_staking_ratio_bps=500000 fixed_staking_ratio_bps=0",
			"scheduled=true",
		} {
			require.Contains(t, shift, want)
		}
	}
	require.NoError(t, a.ValidateAccountingAuthorityStartup(), "carrier startup at H-1")
	require.NoError(t, a.UpgradeKeeper.ApplyUpgrade(ctx, plan))
	for _, marker := range a.ConsensusRepairsV2Markers() {
		on, err := marker.Enabled(ctx)
		require.NoError(t, err)
		require.True(t, on, marker.Module+"."+marker.Name)
	}
	obs := a.AlignmentKeeper.ObserveAll(ctx)
	require.Zero(t, obs.EconomicStability, "bank supply is the denominator now")
	require.Zero(t, obs.StakingRatio)

	// The first scheduled observation does what the census said.
	module := a.ModuleManager.Modules["alignment"].(appmodule.HasEndBlocker)
	for h := plan.Height + 1; ; h++ {
		require.Less(t, h, plan.Height+1000, "no observation in 1000 blocks")
		c := ctx.WithBlockHeight(h).WithHeaderInfo(header.Info{Height: h, ChainID: ctx.ChainID()}).WithEventManager(sdk.NewEventManager())
		require.NoError(t, module.EndBlock(c))
		if a.AlignmentKeeper.GetState(c).LastObservationHeight != uint64(h) {
			continue
		}
		index, found := a.AlignmentKeeper.GetHealthIndex(c, uint64(h))
		require.True(t, found)
		require.Equal(t, alignment.CategoryNotObserved, index.Category)
		creation, analysis := a.AlignmentKeeper.GetGlobalPacingMultiplier(c)
		require.Equal(t, [2]uint64{alignment.BPS, alignment.BPS}, [2]uint64{creation, analysis})
		require.False(t, a.AlignmentKeeper.GetState(c).DegradedFrequencyActive)
		require.Equal(t, baseFee, a.KnowledgeKeeper.GetEffectiveMinReviewFee(c))
		require.Equal(t, alignment.CategoryNotObserved, a.AlignmentKeeper.GetHealthCategory(c), "gov no longer expedites")
		for _, event := range c.EventManager().Events() {
			require.NotEqual(t, "zerone.alignment.network_health_critical", event.Type)
		}
		break
	}
}
