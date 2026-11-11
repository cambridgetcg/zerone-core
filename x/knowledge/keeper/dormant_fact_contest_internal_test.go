package keeper

import (
	"bytes"
	"context"
	"testing"

	corestore "cosmossdk.io/core/store"
	"cosmossdk.io/log"
	"cosmossdk.io/store"
	storemetrics "cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/zerone-chain/zerone/internal/activation"
	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// dormantMarkerReads counts reads of the dormant-fact contest marker and
// passes every call on to the handoff fault store, so one fixture can both
// count reads and inject a read failure.
type dormantMarkerReads struct {
	corestore.KVStoreService
	reads *int
}

func (s dormantMarkerReads) OpenKVStore(ctx context.Context) corestore.KVStore {
	return dormantMarkerReadsStore{s.KVStoreService.OpenKVStore(ctx), s.reads}
}

type dormantMarkerReadsStore struct {
	corestore.KVStore
	reads *int
}

func (s dormantMarkerReadsStore) Get(key []byte) ([]byte, error) {
	if bytes.Equal(key, types.DormantFactContestEnabledKey) {
		*s.reads++
	}
	return s.KVStore.Get(key)
}

// dormantNopBank accepts every transfer and records nothing: the admissions
// below lock collateral, and these tests are about the marker, not coins.
type dormantNopBank struct{}

func (dormantNopBank) SendCoins(context.Context, sdk.AccAddress, sdk.AccAddress, sdk.Coins) error {
	return nil
}
func (dormantNopBank) SendCoinsFromModuleToAccount(context.Context, string, sdk.AccAddress, sdk.Coins) error {
	return nil
}
func (dormantNopBank) SendCoinsFromAccountToModule(context.Context, sdk.AccAddress, string, sdk.Coins) error {
	return nil
}
func (dormantNopBank) SendCoinsFromModuleToModule(context.Context, string, string, sdk.Coins) error {
	return nil
}
func (dormantNopBank) MintCoins(context.Context, string, sdk.Coins) error { return nil }
func (dormantNopBank) GetBalance(_ context.Context, _ sdk.AccAddress, denom string) sdk.Coin {
	return sdk.NewInt64Coin(denom, 0)
}

type dormantMarkerFixture struct {
	handoffFixture
	reads *int
	ms    types.MsgServer
	query types.QueryServer
}

const (
	dormantFactID       = "dormant-at-risk"
	dormantPrunedFactID = "dormant-pruned"
	dormantConjectureID = "dormant-conjecture"
)

// newDormantMarkerFixture is a native-genesis chain (all markers on, the
// dormant-fact contest included) unless dormant is false. It holds an
// AT_RISK assertion, a PRUNED assertion and a PRUNED conjecture. The bank
// accepts every transfer; no staking keeper is wired.
func newDormantMarkerFixture(t *testing.T, dormant bool) dormantMarkerFixture {
	t.Helper()
	db := dbm.NewMemDB()
	ms := store.NewCommitMultiStore(db, log.NewNopLogger(), storemetrics.NewNoOpMetrics())
	key := storetypes.NewKVStoreKey(types.StoreKey)
	ms.MountStoreWithDB(key, storetypes.StoreTypeIAVL, db)
	require.NoError(t, ms.LoadLatestVersion())
	ctx := sdk.NewContext(ms, cmtproto.Header{ChainID: "dormant-marker-test", Height: 100}, false, log.NewNopLogger())
	fault, reads := new(handoffFault), new(int)
	service := dormantMarkerReads{handoffFaultService{runtime.NewKVStoreService(key), fault}, reads}
	k := NewKeeper(service, codec.NewProtoCodec(codectypes.NewInterfaceRegistry()), "authority", dormantNopBank{}, nil)
	gs := types.DefaultGenesis()
	gs.DormantFactContestEnabled = dormant
	require.NoError(t, k.InitGenesis(ctx, gs))
	for _, fact := range []*types.Fact{
		{Id: dormantFactID, ClaimId: "claim-at-risk", Domain: "physics", Category: "empirical", Submitter: "author",
			Content: "A starved assertion", Status: types.FactStatus_FACT_STATUS_AT_RISK, Confidence: 800_000,
			Energy: 200_000, EnergyCap: 1_000_000, AtRiskSinceEpoch: 3},
		{Id: dormantPrunedFactID, ClaimId: "claim-pruned", Domain: "physics", Category: "empirical", Submitter: "author",
			Content: "A pruned assertion", Status: types.FactStatus_FACT_STATUS_PRUNED, Confidence: 800_000,
			Energy: 50_000, EnergyCap: 1_000_000, AtRiskSinceEpoch: 3},
		{Id: dormantConjectureID, ClaimId: "claim-conjecture", Domain: "physics", Category: "empirical", Submitter: "author",
			Content: "A pruned conjecture", Status: types.FactStatus_FACT_STATUS_PRUNED, ClaimType: types.ClaimType_CLAIM_TYPE_CONJECTURE,
			FalsificationPredicate: "exhibit a counterexample", Energy: 50_000, EnergyCap: 1_000_000, AtRiskSinceEpoch: 3},
	} {
		require.NoError(t, k.SetFact(ctx, fact))
	}
	ctx = ctx.WithGasMeter(storetypes.NewInfiniteGasMeter()).WithEventManager(sdk.NewEventManager())
	*reads = 0
	return dormantMarkerFixture{
		handoffFixture: handoffFixture{k: k, ctx: ctx, keys: []*storetypes.KVStoreKey{key}, kFault: fault},
		reads:          reads,
		ms:             NewMsgServerImpl(k),
		query:          NewQueryServerImpl(k),
	}
}

func (f dormantMarkerFixture) challengeMsg(t *testing.T, factID string) *types.MsgChallengeFact {
	t.Helper()
	params, err := f.k.GetParams(f.ctx)
	require.NoError(t, err)
	fact, found := f.k.GetFact(f.ctx, factID)
	require.True(t, found)
	return &types.MsgChallengeFact{
		Challenger: sdk.AccAddress(bytes.Repeat([]byte{7}, 20)).String(), FactId: factID,
		Stake:  EffectiveMinChallengeStake(params, fact.Confidence).String(),
		Reason: "Synthetic dormant-fact contest: a challenge of a starved fact.",
	}
}

func (f dormantMarkerFixture) provisionalMsg(t *testing.T) *types.MsgChallengeProvisionalFact {
	t.Helper()
	params, err := f.k.GetParams(f.ctx)
	require.NoError(t, err)
	return &types.MsgChallengeProvisionalFact{
		Challenger: sdk.AccAddress(bytes.Repeat([]byte{8}, 20)).String(), FactId: dormantConjectureID,
		Stake: params.MinChallengeStake, Reason: "Synthetic dormant-fact contest: a refutation of a lapsed conjecture.",
	}
}

func (f dormantMarkerFixture) contradictionMsg(t *testing.T) *types.MsgSubmitContradiction {
	t.Helper()
	params, err := f.k.GetParams(f.ctx)
	require.NoError(t, err)
	return &types.MsgSubmitContradiction{
		Submitter: sdk.AccAddress(bytes.Repeat([]byte{9}, 20)).String(), FactId: dormantPrunedFactID,
		CounterClaim: "Synthetic dormant-fact contest: a counter-observation of a pruned fact.",
		Domain:       "physics", Category: "empirical",
		Stake: EffectiveMinChallengeStake(params, 800_000).String(),
	}
}

// A marker that cannot be read, or holds anything but {1}, is a fault, never
// off. Every entry point that reads it returns the fault (the compatibility
// listing panics with it) before any write, with no event and no gas.
func TestDormantFactContestMarkerFaultPropagates(t *testing.T) {
	for _, tc := range []struct {
		name  string
		value []byte // nil: the read itself fails
	}{
		{"unreadable", nil},
		{"malformed-0", []byte{0}},
		{"malformed-2", []byte{2}},
		{"malformed-long", []byte{1, 0}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newDormantMarkerFixture(t, true)
			challenge, provisional, contradiction := f.challengeMsg(t, dormantFactID), f.provisionalMsg(t), f.contradictionMsg(t)
			round := &types.VerificationRound{Id: "round", ClaimId: "claim"}
			if tc.value == nil {
				f.kFault.op, f.kFault.prefix = "get", types.DormantFactContestEnabledKey
			} else {
				f.ctx.KVStore(f.keys[0]).Set(types.DormantFactContestEnabledKey, tc.value)
			}
			before := f.snapshot(t)
			for name, call := range map[string]func(sdk.Context) error{
				"ChallengeFact": func(ctx sdk.Context) error { _, err := f.ms.ChallengeFact(ctx, challenge); return err },
				"ChallengeProvisionalFact": func(ctx sdk.Context) error {
					_, err := f.ms.ChallengeProvisionalFact(ctx, provisional)
					return err
				},
				"SubmitContradiction": func(ctx sdk.Context) error { _, err := f.ms.SubmitContradiction(ctx, contradiction); return err },
				"OpenQuestions": func(ctx sdk.Context) error {
					_, err := f.query.OpenQuestions(ctx, &types.QueryOpenQuestionsRequest{})
					return err
				},
				"OpenQuestionsForDomain": func(ctx sdk.Context) (err error) {
					defer func() {
						recovered, ok := recover().(error)
						require.True(t, ok, "the compatibility listing panics with the fault")
						err = recovered
					}()
					f.k.OpenQuestionsForDomain(ctx, "", 0)
					return nil
				},
				"BeginBlocker":       func(ctx sdk.Context) error { return f.k.BeginBlocker(ctx) },
				"CompleteRound":      func(ctx sdk.Context) error { return f.k.CompleteRound(ctx, round, &VerificationResult{}) },
				"AdvanceRoundPhases": func(ctx sdk.Context) error { return f.k.AdvanceRoundPhases(ctx) },
			} {
				meter, events := storetypes.NewInfiniteGasMeter(), sdk.NewEventManager()
				err := call(f.ctx.WithGasMeter(meter).WithEventManager(events))
				require.ErrorIs(t, err, activation.ErrMarkerFault, name)
				require.Zero(t, meter.GasConsumed(), "%s: the marker read charges no gas", name)
				require.Empty(t, events.Events(), name)
				require.Equal(t, before, f.snapshot(t), "%s writes nothing", name)
			}
		})
	}
}

// The marker is read once per entry point. Each admitting message reads it
// once; the open-questions query reads it once; a BeginBlocker that ends a
// starved challenge reads it once at entry, and the restore uses that
// selection: the AT_RISK target comes back AT_RISK with its stamp.
func TestDormantFactContestMarkerReadOncePerEntryPoint(t *testing.T) {
	f := newDormantMarkerFixture(t, true)

	opened, err := f.ms.ChallengeFact(f.ctx, f.challengeMsg(t, dormantFactID))
	require.NoError(t, err)
	require.Equal(t, 1, *f.reads, "ChallengeFact")

	*f.reads = 0
	_, err = f.ms.ChallengeProvisionalFact(f.ctx, f.provisionalMsg(t))
	require.NoError(t, err)
	require.Equal(t, 1, *f.reads, "ChallengeProvisionalFact")

	*f.reads = 0
	_, err = f.ms.SubmitContradiction(f.ctx, f.contradictionMsg(t))
	require.NoError(t, err)
	require.Equal(t, 1, *f.reads, "SubmitContradiction")

	*f.reads = 0
	response, err := f.query.OpenQuestions(f.ctx, &types.QueryOpenQuestionsRequest{})
	require.NoError(t, err)
	require.Equal(t, 1, *f.reads, "OpenQuestions")
	require.Len(t, response.Questions, 1, "the pruned conjecture is listed")

	round, found := f.k.GetVerificationRound(f.ctx, opened.RoundId)
	require.True(t, found)
	ctx := f.ctx.WithBlockHeight(int64(round.RevealDeadline))
	*f.reads = 0
	require.NoError(t, f.k.BeginBlocker(ctx))
	require.Equal(t, 1, *f.reads, "BeginBlocker")
	done, found := f.k.GetVerificationRound(ctx, opened.RoundId)
	require.True(t, found)
	require.Equal(t, types.Verdict_VERDICT_INCONCLUSIVE, done.Verdict)
	fact, found := f.k.GetFact(ctx, dormantFactID)
	require.True(t, found)
	require.Equal(t, types.FactStatus_FACT_STATUS_AT_RISK, fact.Status)
	require.Equal(t, uint64(3), fact.AtRiskSinceEpoch)
	require.Equal(t, uint64(200_000), fact.Energy)
}

// The open-questions query reads the marker without gas: with the marker off
// it costs exactly what the marker-free listing costs, and with it on, over
// the same facts and no pruned conjecture, the same again.
func TestDormantFactContestOpenQuestionsGasUnchanged(t *testing.T) {
	gasOf := func(f dormantMarkerFixture, run func(sdk.Context)) uint64 {
		meter := storetypes.NewInfiniteGasMeter()
		run(f.ctx.WithGasMeter(meter))
		return meter.GasConsumed()
	}
	query := func(f dormantMarkerFixture) func(sdk.Context) {
		return func(ctx sdk.Context) {
			_, err := f.query.OpenQuestions(ctx, &types.QueryOpenQuestionsRequest{})
			require.NoError(t, err)
		}
	}
	off, on := newDormantMarkerFixture(t, false), newDormantMarkerFixture(t, true)
	for _, f := range []dormantMarkerFixture{off, on} {
		conjecture, found := f.k.GetFact(f.ctx, dormantConjectureID)
		require.True(t, found)
		conjecture.Status = types.FactStatus_FACT_STATUS_PROVISIONAL
		require.NoError(t, f.k.SetFact(f.ctx, conjecture))
	}

	worker := gasOf(off, func(ctx sdk.Context) { off.k.openQuestionsForDomain(ctx, "", 0, false) })
	require.NotZero(t, worker)
	require.Equal(t, worker, gasOf(off, query(off)), "marker off: the query costs what the listing costs")
	require.Equal(t, worker, gasOf(on, query(on)), "marker on, same facts: the same gas")
}
