package keeper_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	knowledgemodule "github.com/zerone-chain/zerone/x/knowledge"
	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// Reviewer-balance parameter (consensus-repairs-v3, marker 0x96). With the
// marker on, MsgSubmitCommitment compares the reviewer's bank balance with
// Params.MinReviewerBalance, empty meaning the built-in 100 ZRN. With it off,
// the built-in 100 ZRN applies whatever the parameter holds. The internal
// tests (reviewer_balance_param_internal_test.go) cover marker faults, read
// counts and gas.

// agentDoorUzrn is the seed a newcomer admitted through the agents' door
// receives: 0.222 ZRN (tools/zerone-door/src/policy.ts SEED_UZRN), below the
// built-in figure.
const agentDoorUzrn = 222_000

// newReviewerBalanceFixture is native DefaultGenesis, every selection on,
// with the reviewer-balance marker on or off and the parameter as given. A
// parameter with the marker off cannot pass genesis, so it is written after
// genesis, straight to the store: the state is unreachable through messages,
// which is what the marker-off cases prove harmless.
func newReviewerBalanceFixture(t *testing.T, markerOn bool, param string) *v11Fixture {
	t.Helper()
	gs := types.DefaultGenesis()
	gs.ReviewerBalanceParamEnabled = markerOn
	if markerOn {
		gs.Params.MinReviewerBalance = param
	}
	f := newClaimIntegrityGenesisFixture(t, "reviewer-balance", gs)
	enabled, err := f.k.ReviewerBalanceParamEnabled(f.ctx)
	require.NoError(t, err)
	require.Equal(t, markerOn, enabled)
	if !markerOn && param != "" {
		params, err := f.k.GetParams(f.ctx)
		require.NoError(t, err)
		params.MinReviewerBalance = param
		require.NoError(t, f.k.SetParams(f.ctx, params))
	}
	return f
}

// commit opens a fresh legacy commit-phase round and submits one commitment
// from a reviewer holding exactly balance uzrn.
func (f *v11Fixture) reviewerCommit(t *testing.T, name string, balance int64) error {
	t.Helper()
	reviewer := makeValidBech32Addr("reviewer-" + name)
	f.bank.balances[reviewer] = sdk.NewCoins(sdk.NewInt64Coin("uzrn", balance))
	roundID := "round-" + name
	require.NoError(t, f.k.SetVerificationRound(f.ctx, makeRoundInPhase(roundID, "claim-"+name, types.VerificationPhase_VERIFICATION_PHASE_COMMIT, 98)))
	_, err := f.server.SubmitCommitment(f.ctx, &types.MsgSubmitCommitment{
		Verifier:   reviewer,
		RoundId:    roundID,
		CommitHash: computeMsgServerCommitHash(roundID, "accept", 0, []byte("salt")),
	})
	round, found := f.k.GetVerificationRound(f.ctx, roundID)
	require.True(t, found)
	if err != nil {
		require.Empty(t, round.Commits, "a refused commitment stores nothing")
	} else {
		require.Len(t, round.Commits, 1)
	}
	return err
}

func requireBalanceRefusal(t *testing.T, err error) {
	t.Helper()
	require.Error(t, err)
	require.Contains(t, err.Error(), "verifier does not meet minimum balance requirement")
}

// Just below the figure is refused and exactly at it is accepted, for each
// marker state and parameter value. With the marker off a configured
// parameter is ignored: the built-in 100 ZRN holds.
func TestReviewerBalanceCommitThreshold(t *testing.T) {
	for _, tc := range []struct {
		name     string
		markerOn bool
		param    string
		figure   int64
	}{
		{"on/configured-lower", true, "222000", agentDoorUzrn},
		{"on/configured-one", true, "1", 1},
		{"on/configured-higher", true, "150000000", 150_000_000},
		{"on/empty-is-100-ZRN", true, "", 100_000_000},
		{"off/empty", false, "", 100_000_000},
		{"off/configured-lower-ignored", false, "222000", 100_000_000},
		{"off/configured-higher-ignored", false, "150000000", 100_000_000},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newReviewerBalanceFixture(t, tc.markerOn, tc.param)
			requireBalanceRefusal(t, f.reviewerCommit(t, "below", tc.figure-1))
			require.NoError(t, f.reviewerCommit(t, "at", tc.figure))
		})
	}
}

// The agents' door case: with the marker off a newcomer holding 0.222 ZRN
// cannot commit, even with the parameter written to 0.222 ZRN.
func TestReviewerBalanceMarkerOffIgnoresParameter(t *testing.T) {
	f := newReviewerBalanceFixture(t, false, fmt.Sprint(agentDoorUzrn))
	requireBalanceRefusal(t, f.reviewerCommit(t, "door", agentDoorUzrn))
	requireBalanceRefusal(t, f.reviewerCommit(t, "almost", 99_999_999))
	require.NoError(t, f.reviewerCommit(t, "hundred", 100_000_000))
}

func updateReviewerBalance(t *testing.T, f *v11Fixture, value string) error {
	t.Helper()
	params, err := f.k.GetParams(f.ctx)
	require.NoError(t, err)
	next := proto.Clone(params).(*types.Params)
	next.MinReviewerBalance = value
	_, err = f.server.UpdateParams(f.ctx, &types.MsgUpdateParams{Authority: "authority", Params: next})
	return err
}

func storedReviewerBalance(t *testing.T, f *v11Fixture) string {
	t.Helper()
	params, err := f.k.GetParams(f.ctx)
	require.NoError(t, err)
	return params.MinReviewerBalance
}

// With the marker on, the module's governance parameter update carries the
// new field and it takes effect at the next commitment; setting it back to
// empty restores 100 ZRN.
func TestReviewerBalanceParamUpdateTakesEffect(t *testing.T) {
	f := newReviewerBalanceFixture(t, true, "")
	requireBalanceRefusal(t, f.reviewerCommit(t, "before", agentDoorUzrn))

	require.NoError(t, updateReviewerBalance(t, f, "222000"))
	require.Equal(t, "222000", storedReviewerBalance(t, f))
	requireBalanceRefusal(t, f.reviewerCommit(t, "below", agentDoorUzrn-1))
	require.NoError(t, f.reviewerCommit(t, "after", agentDoorUzrn))

	require.NoError(t, updateReviewerBalance(t, f, ""))
	require.Empty(t, storedReviewerBalance(t, f))
	requireBalanceRefusal(t, f.reviewerCommit(t, "restored", agentDoorUzrn))
	require.NoError(t, f.reviewerCommit(t, "restored-100", 100_000_000))
}

// With the marker off, an update that sets the field is refused and stores
// nothing, as before the field existed. An update that leaves it untouched
// still passes.
func TestReviewerBalanceParamUpdateRefusedWithMarkerOff(t *testing.T) {
	f := newReviewerBalanceFixture(t, false, "")
	err := updateReviewerBalance(t, f, "222000")
	require.ErrorContains(t, err, "min_reviewer_balance requires the reviewer-balance-param activation")
	require.Empty(t, storedReviewerBalance(t, f))

	params, err := f.k.GetParams(f.ctx)
	require.NoError(t, err)
	next := proto.Clone(params).(*types.Params)
	next.MinVerifiers = params.MinVerifiers + 1
	_, err = f.server.UpdateParams(f.ctx, &types.MsgUpdateParams{Authority: "authority", Params: next})
	require.NoError(t, err)
	requireBalanceRefusal(t, f.reviewerCommit(t, "door", agentDoorUzrn))
}

// Validation refuses a malformed value in an update, whatever the marker.
func TestReviewerBalanceParamUpdateRefusesMalformed(t *testing.T) {
	for _, markerOn := range []bool{true, false} {
		t.Run(fmt.Sprintf("marker-%t", markerOn), func(t *testing.T) {
			f := newReviewerBalanceFixture(t, markerOn, "")
			for _, bad := range []string{"0", "00", "0222000", "+222000", "-1", " 222000", "222000 ", "2.5", "1e6", "18446744073709551616", "222_000"} {
				err := updateReviewerBalance(t, f, bad)
				require.ErrorContains(t, err, "min_reviewer_balance must be empty or a canonical positive base-10 integer", bad)
				require.Empty(t, storedReviewerBalance(t, f), bad)
			}
		})
	}
}

// reviewerBalanceModule is the knowledge AppModule over a fresh native store,
// for genesis JSON round trips.
func reviewerBalanceModule(t *testing.T) (*v11Fixture, knowledgemodule.AppModule, codec.JSONCodec) {
	t.Helper()
	f := newKnowledgeStoreFixture(t, "reviewer-balance-genesis")
	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	return f, knowledgemodule.NewAppModule(cdc, f.k), cdc
}

// genesisWithReviewerBalance is the module's default genesis JSON with the
// marker field and Params.min_reviewer_balance set as given; nil removes the
// field, as in a file written before it existed.
func genesisWithReviewerBalance(t *testing.T, am knowledgemodule.AppModule, cdc codec.JSONCodec, marker bool, param *string) json.RawMessage {
	t.Helper()
	var fields map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(am.DefaultGenesis(cdc), &fields))
	var params map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(fields["params"], &params))
	require.JSONEq(t, `""`, string(params["min_reviewer_balance"]), "default genesis leaves the parameter empty")
	if param == nil {
		delete(params, "min_reviewer_balance")
	} else {
		bz, err := json.Marshal(*param)
		require.NoError(t, err)
		params["min_reviewer_balance"] = bz
	}
	bz, err := json.Marshal(params)
	require.NoError(t, err)
	fields["params"] = bz
	fields["reviewer_balance_param_enabled"] = json.RawMessage(fmt.Sprint(marker))
	raw, err := json.Marshal(fields)
	require.NoError(t, err)
	return raw
}

func exportedReviewerBalance(t *testing.T, raw json.RawMessage) (string, bool) {
	t.Helper()
	var gs struct {
		Params struct {
			MinReviewerBalance string `json:"min_reviewer_balance"`
		} `json:"params"`
		Marker bool `json:"reviewer_balance_param_enabled"`
	}
	require.NoError(t, json.Unmarshal(raw, &gs))
	return gs.Params.MinReviewerBalance, gs.Marker
}

// Genesis export and import carry the parameter beside its marker, through
// the module's JSON codec, and the imported chain applies it.
func TestReviewerBalanceParamGenesisRoundTrip(t *testing.T) {
	door := fmt.Sprint(agentDoorUzrn)
	for _, tc := range []struct {
		name   string
		marker bool
		param  *string
		want   string
		figure int64
	}{
		{"on/configured", true, &door, door, agentDoorUzrn},
		{"on/empty", true, nil, "", 100_000_000},
		{"off/empty", false, nil, "", 100_000_000},
	} {
		t.Run(tc.name, func(t *testing.T) {
			first, am, cdc := reviewerBalanceModule(t)
			raw := genesisWithReviewerBalance(t, am, cdc, tc.marker, tc.param)
			require.NoError(t, am.ValidateGenesis(cdc, nil, raw))
			am.InitGenesis(first.ctx, cdc, raw)

			exported := am.ExportGenesis(first.ctx, cdc)
			param, marker := exportedReviewerBalance(t, exported)
			require.Equal(t, tc.want, param)
			require.Equal(t, tc.marker, marker)

			second, am2, cdc2 := reviewerBalanceModule(t)
			require.NoError(t, am2.ValidateGenesis(cdc2, nil, exported))
			am2.InitGenesis(second.ctx, cdc2, exported)
			require.Equal(t, tc.want, storedReviewerBalance(t, second))
			enabled, err := second.k.ReviewerBalanceParamEnabled(second.ctx)
			require.NoError(t, err)
			require.Equal(t, tc.marker, enabled)
			again, _ := exportedReviewerBalance(t, am2.ExportGenesis(second.ctx, cdc2))
			require.Equal(t, tc.want, again)

			// The imported chain applies what it imported. (These test rounds
			// have no claim, so no export follows them.)
			requireBalanceRefusal(t, second.reviewerCommit(t, "below", tc.figure-1))
			require.NoError(t, second.reviewerCommit(t, "at", tc.figure))
		})
	}
}

// A genesis may not configure the parameter with the marker off, nor carry a
// malformed value. ValidateGenesis refuses it, and so does InitGenesis (which
// InitChain runs without ValidateGenesis) before any write.
func TestReviewerBalanceParamGenesisRefusals(t *testing.T) {
	door, zero, padded := fmt.Sprint(agentDoorUzrn), "0", "0222000"
	for _, tc := range []struct {
		name   string
		marker bool
		param  string
		want   string
	}{
		{"off/configured", false, door, "min_reviewer_balance requires reviewer_balance_param_enabled"},
		{"on/zero", true, zero, "min_reviewer_balance must be empty or a canonical positive base-10 integer"},
		{"on/leading-zero", true, padded, "min_reviewer_balance must be empty or a canonical positive base-10 integer"},
		{"off/zero", false, zero, "min_reviewer_balance must be empty or a canonical positive base-10 integer"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f, am, cdc := reviewerBalanceModule(t)
			param := tc.param
			raw := genesisWithReviewerBalance(t, am, cdc, tc.marker, &param)
			require.ErrorContains(t, am.ValidateGenesis(cdc, nil, raw), tc.want)

			gs := types.DefaultGenesis()
			gs.ReviewerBalanceParamEnabled = tc.marker
			gs.Params.MinReviewerBalance = tc.param
			require.ErrorContains(t, f.k.InitGenesis(f.ctx, gs), tc.want)
			iter := f.ctx.KVStore(f.key).Iterator(nil, nil)
			defer iter.Close()
			require.False(t, iter.Valid(), "InitGenesis refused before any write")
		})
	}
}
