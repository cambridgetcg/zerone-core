package app

import (
	"encoding/json"
	"testing"

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

func TestConsensusRepairsV1RefusesDeferredBankSupplyGenesisSelection(t *testing.T) {
	app, _, _ := newAccountingAuthorityFixture(t)
	base := sdk053IBC10GenesisWithValidator(t, app)
	require.NoError(t, ValidateConsensusRepairsGenesisSelection(base), "default native genesis leaves the deferred marker off")
	for name, raw := range map[string]string{
		"true":       `true`,
		"camelCase":  `false`,
		"uppercase":  `false`,
		"not a bool": `"true"`,
	} {
		t.Run(name, func(t *testing.T) {
			genesis := GenesisState{}
			for module, value := range base {
				genesis[module] = value
			}
			var fields map[string]json.RawMessage
			require.NoError(t, json.Unmarshal(genesis["vesting_rewards"], &fields))
			delete(fields, "alignment_bank_supply_enabled")
			field := "alignment_bank_supply_enabled"
			switch name {
			case "camelCase":
				field = "alignmentBankSupplyEnabled"
			case "uppercase":
				field = "ALIGNMENT_BANK_SUPPLY_ENABLED"
			}
			fields[field] = json.RawMessage(raw)
			encoded, err := json.Marshal(fields)
			require.NoError(t, err)
			genesis["vesting_rewards"] = encoded
			require.ErrorContains(t, ValidateConsensusRepairsGenesisSelection(genesis), "absent or false")
		})
	}
}
