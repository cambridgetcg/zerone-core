package main

import (
	"bytes"
	"encoding/json"
	"strconv"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"github.com/zerone-chain/zerone/app"
	alignment "github.com/zerone-chain/zerone/x/alignment/types"
)

func TestReadOnlyCopiedDBV2Census(t *testing.T) {
	for _, fault := range []bool{false, true} {
		home, height, hash := sourceCopy(t, fault, false, true)
		before := fileDigests(t, home)
		var out, diagnostics bytes.Buffer
		code := run([]string{"--repairs", "v2", "--copied-db", "--home", home, "--chain-id", "zerone-rehearsal-census", "--expected-height", strconv.FormatInt(height, 10), "--expected-app-hash", hash}, &out, &diagnostics)
		want := 0
		if fault {
			want = 2
		}
		require.Equal(t, want, code, diagnostics.String()+out.String())
		var value struct {
			Schema string `json:"schema"`
			Passed bool   `json:"passed"`
		}
		require.NoError(t, json.Unmarshal(out.Bytes(), &value))
		require.Equal(t, "zerone.consensus-repairs-census/v2", value.Schema)
		require.Equal(t, !fault, value.Passed)
		require.Contains(t, out.String(), "fx7_accounting")
		require.Contains(t, out.String(), "fx8_retained")
		require.Contains(t, out.String(), "q5_calibration")
		require.Equal(t, before, fileDigests(t, home), "every copied source file remains byte-identical")
	}
}

// F5: on dev-1's measured inputs and live alignment state (previous category
// degraded, pacing 750,000/1,500,000), the v2 census tool states the shift v2
// causes: not_observed, neutral pacing, base fees and unexpedited gov voting.
func TestDev1ShapedCopiedDBV2CensusReportsPacingShift(t *testing.T) {
	home, height, hash := sourceCopyMutated(t, false, []bool{false, true}, func(a *app.ZeroneApp, ctx sdk.Context) {
		dev1CensusShape(a, ctx)
		for _, location := range []struct {
			store string
			key   byte
		}{{"alignment", 0x09}, {"vesting_rewards", 0x0c}} {
			ctx.KVStore(a.GetStoreKeyForTests(location.store)).Delete([]byte{location.key})
		}
		state := a.AlignmentKeeper.GetState(ctx)
		state.PreviousCategory, state.DegradedFrequencyActive = alignment.CategoryDegraded, false
		a.AlignmentKeeper.SetState(ctx, state)
		a.AlignmentKeeper.SetHealthIndex(ctx, &alignment.AlignmentHealthIndex{Height: uint64(ctx.BlockHeight()), CompositeScore: 249_800, Category: alignment.CategoryDegraded})
	})
	before := fileDigests(t, home)
	var out, diagnostics bytes.Buffer
	code := run([]string{"--repairs", "v2", "--copied-db", "--home", home, "--chain-id", "zerone-rehearsal-census", "--expected-height", strconv.FormatInt(height, 10), "--expected-app-hash", hash}, &out, &diagnostics)
	require.Equal(t, 0, code, diagnostics.String()+out.String())
	var report struct {
		Passed bool                       `json:"passed"`
		Census app.ConsensusRepairsCensus `json:"census"`
	}
	require.NoError(t, json.Unmarshal(out.Bytes(), &report))
	require.True(t, report.Passed)
	shift := ""
	for _, finding := range report.Census.Findings {
		if finding.Census == "q1_health" && finding.Module == "alignment" {
			require.Equal(t, "held", finding.Outcome)
			shift = finding.Reason
		}
	}
	t.Logf("q1_health: %s", shift)
	for _, want := range []string{
		"category degraded->not_observed",
		"creation_pacing_bps 750000->1000000 analysis_pacing_bps 1500000->1000000",
		"(expedited true->false; gov read degraded before)",
		"inputs registered=0 active=0 active_stake=0 mint_ledger=0 bank_supply=3000000000000 legacy_staking_ratio_bps=500000 fixed_staking_ratio_bps=0",
	} {
		require.Contains(t, shift, want)
	}
	require.Equal(t, before, fileDigests(t, home), "the copied database is only read")
}
