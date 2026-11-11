package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/zerone-chain/zerone/x/knowledge/types"
)

func TestParseMinReviewerBalance(t *testing.T) {
	require.Equal(t, uint64(100_000_000), types.BuiltInMinReviewerBalanceUzrn)
	for value, want := range map[string]uint64{
		"":                     100_000_000,
		"1":                    1,
		"222000":               222_000,
		"100000000":            100_000_000,
		"18446744073709551615": 18_446_744_073_709_551_615,
	} {
		got, err := types.ParseMinReviewerBalance(value)
		require.NoError(t, err, value)
		require.Equal(t, want, got, value)
	}
	for _, bad := range []string{
		"0", "00", "000", "01", "0222000", "+1", "-1", " 1", "1 ", "\t1", "1\n",
		"1.0", "2.5", "1e6", "0x10", "1_000", "1,000", "18446744073709551616",
		"99999999999999999999999", "１", "uzrn", "100uzrn",
	} {
		_, err := types.ParseMinReviewerBalance(bad)
		require.ErrorContains(t, err, "min_reviewer_balance must be empty or a canonical positive base-10 integer", "%q", bad)
	}
}

func TestMinReviewerBalanceParamsValidation(t *testing.T) {
	p := types.DefaultParams()
	require.Empty(t, p.MinReviewerBalance, "DefaultParams leaves the field empty: default genesis behaves as before")
	require.NoError(t, p.Validate())

	p.MinReviewerBalance = "222000"
	require.NoError(t, p.Validate())
	for _, bad := range []string{"0", "007", "-5", "1e3", "18446744073709551616"} {
		p.MinReviewerBalance = bad
		require.ErrorContains(t, p.Validate(), "min_reviewer_balance", bad)
	}
}

func TestMinReviewerBalanceGenesisValidation(t *testing.T) {
	gs := types.DefaultGenesis()
	require.True(t, gs.ReviewerBalanceParamEnabled)
	require.NoError(t, gs.Validate())

	gs.Params.MinReviewerBalance = "222000"
	require.NoError(t, gs.Validate(), "configured with the marker on")

	gs.ReviewerBalanceParamEnabled = false
	require.ErrorContains(t, gs.Validate(), "min_reviewer_balance requires reviewer_balance_param_enabled")

	gs.Params.MinReviewerBalance = ""
	require.NoError(t, gs.Validate(), "marker off with the field empty is today's genesis")

	gs.ReviewerBalanceParamEnabled = true
	gs.Params.MinReviewerBalance = "0"
	require.ErrorContains(t, gs.Validate(), "min_reviewer_balance must be empty or a canonical positive base-10 integer")
}
