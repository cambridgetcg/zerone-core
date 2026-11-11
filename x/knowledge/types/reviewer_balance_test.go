package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// floorRefusal is the sentence a figure below the built-in 100 ZRN is refused
// with, wherever it is refused.
const floorRefusal = "is below the built-in minimum of 100000000 uzrn: a lower figure needs a locked seat deposit, which this source does not have"

// One below, exactly at and one above the built-in 100 ZRN, in uzrn.
const (
	oneBelow = "99999999"
	atFloor  = "100000000"
	oneAbove = "100000001"
)

// ParseMinReviewerBalance checks the form only: a canonical positive uint64,
// or empty for the built-in figure. The floor is ValidateMinReviewerBalance's.
func TestParseMinReviewerBalance(t *testing.T) {
	require.Equal(t, uint64(100_000_000), types.BuiltInMinReviewerBalanceUzrn)
	for value, want := range map[string]uint64{
		"":                     100_000_000,
		"1":                    1,
		"222000":               222_000,
		oneBelow:               99_999_999,
		atFloor:                100_000_000,
		oneAbove:               100_000_001,
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

// A3: a figure below the built-in minimum is refused, one at or above it
// passes, and a malformed value keeps its own refusal.
func TestValidateMinReviewerBalanceFloor(t *testing.T) {
	for _, ok := range []string{"", atFloor, oneAbove, "150000000", "18446744073709551615"} {
		require.NoError(t, types.ValidateMinReviewerBalance(ok), ok)
	}
	for _, low := range []string{oneBelow, "222000", "1"} {
		err := types.ValidateMinReviewerBalance(low)
		require.ErrorContains(t, err, "min_reviewer_balance "+low+" "+floorRefusal, low)
	}
	for _, bad := range []string{"0", "007", "-5", "1e3", "18446744073709551616"} {
		require.ErrorContains(t, types.ValidateMinReviewerBalance(bad), "min_reviewer_balance must be empty or a canonical positive base-10 integer", bad)
	}
}

// A5: the figure a commitment is compared with is the larger of the stored
// value and the built-in minimum, so a stored figure below it reads as 100 ZRN.
func TestEffectiveMinReviewerBalance(t *testing.T) {
	for value, want := range map[string]uint64{
		"":                     100_000_000,
		"1":                    100_000_000,
		"222000":               100_000_000,
		oneBelow:               100_000_000,
		atFloor:                100_000_000,
		oneAbove:               100_000_001,
		"150000000":            150_000_000,
		"18446744073709551615": 18_446_744_073_709_551_615,
	} {
		got, err := types.EffectiveMinReviewerBalance(value)
		require.NoError(t, err, value)
		require.Equal(t, want, got, value)
	}
	for _, bad := range []string{"0", "007", "abc", "18446744073709551616"} {
		_, err := types.EffectiveMinReviewerBalance(bad)
		require.ErrorContains(t, err, "min_reviewer_balance must be empty or a canonical positive base-10 integer", bad)
	}
}

// Params.Validate, which genesis validation and MsgUpdateParams both run,
// applies the floor. Params carries no marker, so the floor holds whatever
// the marker's state.
func TestMinReviewerBalanceParamsValidation(t *testing.T) {
	p := types.DefaultParams()
	require.Empty(t, p.MinReviewerBalance, "DefaultParams leaves the field empty: default genesis behaves as before")
	require.NoError(t, p.Validate())

	for _, ok := range []string{atFloor, oneAbove} {
		p.MinReviewerBalance = ok
		require.NoError(t, p.Validate(), ok)
	}
	for _, low := range []string{oneBelow, "222000"} {
		p.MinReviewerBalance = low
		require.ErrorContains(t, p.Validate(), floorRefusal, low)
	}
	for _, bad := range []string{"0", "007", "-5", "1e3", "18446744073709551616"} {
		p.MinReviewerBalance = bad
		require.ErrorContains(t, p.Validate(), "min_reviewer_balance", bad)
	}
}

// Genesis validation, in both marker states: one below the floor is refused
// for the floor; with the marker on, at and one above pass; with it off, any
// configured figure is refused for the marker.
func TestMinReviewerBalanceGenesisValidation(t *testing.T) {
	gs := types.DefaultGenesis()
	require.True(t, gs.ReviewerBalanceParamEnabled)
	require.NoError(t, gs.Validate())

	const markerRefusal = "min_reviewer_balance requires reviewer_balance_param_enabled"
	for _, tc := range []struct {
		marker bool
		param  string
		want   string // empty: accepted
	}{
		{true, oneBelow, floorRefusal},
		{true, atFloor, ""},
		{true, oneAbove, ""},
		{true, "222000", floorRefusal},
		{true, "0", "min_reviewer_balance must be empty or a canonical positive base-10 integer"},
		{false, oneBelow, floorRefusal},
		{false, atFloor, markerRefusal},
		{false, oneAbove, markerRefusal},
		{false, "", ""},
	} {
		gs := types.DefaultGenesis()
		gs.ReviewerBalanceParamEnabled = tc.marker
		gs.Params.MinReviewerBalance = tc.param
		if tc.want == "" {
			require.NoError(t, gs.Validate(), "marker %t, %q", tc.marker, tc.param)
			require.NoError(t, types.ValidateReviewerBalanceGenesis(gs), "marker %t, %q", tc.marker, tc.param)
			continue
		}
		require.ErrorContains(t, gs.Validate(), tc.want, "marker %t, %q", tc.marker, tc.param)
		require.ErrorContains(t, types.ValidateReviewerBalanceGenesis(gs), tc.want, "marker %t, %q", tc.marker, tc.param)
	}
}
