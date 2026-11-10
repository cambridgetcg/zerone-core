package keeper

import (
	"testing"

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

// newDoctrineReservedFixture is newDoctrineMarkerFixture with the record
// markers off, so the legacy SubmitClaim path is reachable. It holds a
// verified fact to challenge or contradict and a conjecture's fact to
// challenge. No bank or staking keeper is wired: these tests stop at the
// marker.
func newDoctrineReservedFixture(t *testing.T) doctrineMarkerFixture {
	t.Helper()
	db := dbm.NewMemDB()
	ms := store.NewCommitMultiStore(db, log.NewNopLogger(), storemetrics.NewNoOpMetrics())
	key := storetypes.NewKVStoreKey(types.StoreKey)
	ms.MountStoreWithDB(key, storetypes.StoreTypeIAVL, db)
	require.NoError(t, ms.LoadLatestVersion())
	ctx := sdk.NewContext(ms, cmtproto.Header{Height: 100}, false, log.NewNopLogger())
	fault, reads := new(handoffFault), new(int)
	service := doctrineMarkerReads{handoffFaultService{runtime.NewKVStoreService(key), fault}, reads}
	k := NewKeeper(service, codec.NewProtoCodec(codectypes.NewInterfaceRegistry()), "authority", nil, nil)
	gs := types.DefaultGenesis()
	gs.RecordIntegrityEnabled = false
	gs.ReviewNeutralityEnabled = false
	gs.ClaimRecordsEnabled = false
	gs.FundSettlementEnabled = false
	gs.ClaimIntegrityEnabled = false
	require.NoError(t, k.InitGenesis(ctx, gs))
	require.NoError(t, k.SetFact(ctx, &types.Fact{
		Id: "fact-verified", ClaimId: "claim-verified", Category: "empirical", Domain: "physics",
		Content: "An ordinary verified fact", Submitter: "author", Confidence: 500_000,
		Status: types.FactStatus_FACT_STATUS_VERIFIED, Energy: 500_000, EnergyCap: 1_000_000,
	}))
	require.NoError(t, k.SetFact(ctx, &types.Fact{
		Id: "fact-conjecture", ClaimId: "claim-conjecture", Category: types.DoctrineCategory, Domain: "physics",
		Content: "An open conjecture", Submitter: "author", ClaimType: types.ClaimType_CLAIM_TYPE_CONJECTURE,
		Status: types.FactStatus_FACT_STATUS_PROVISIONAL, Energy: 500_000, EnergyCap: 1_000_000,
	}))
	ctx = ctx.WithGasMeter(storetypes.NewInfiniteGasMeter()).WithEventManager(sdk.NewEventManager())
	*reads = 0
	return doctrineMarkerFixture{
		handoffFixture: handoffFixture{k: k, ctx: ctx, keys: []*storetypes.KVStoreKey{key}, kFault: fault},
		reads:          reads,
		epoch:          ctx,
	}
}

// doctrineReservedEntries are the entry points this repair makes read the
// doctrine-identity marker, and BeginBlocker at the confidence growth epoch and
// at the fitness epoch. ProcessMetabolism is covered in
// doctrine_identity_internal_test.go. Each runs with category "doctrine" where
// it takes one.
var doctrineReservedEntries = []string{
	"conjecture", "contradiction", "claim/legacy", "challenge", "provisional-challenge",
	"fitness", "growth", "rescue", "begin/growth", "begin/fitness",
}

func runDoctrineReservedEntry(t *testing.T, k Keeper, ctx sdk.Context, entry string) error {
	t.Helper()
	ms := NewMsgServerImpl(k)
	// Read on a separate meter, so the caller measures only the entry point.
	params, err := k.GetParams(ctx.WithGasMeter(storetypes.NewInfiniteGasMeter()))
	require.NoError(t, err)
	switch entry {
	case "conjecture":
		_, err = ms.PostConjecture(ctx, &types.MsgPostConjecture{
			Proposer: "proposer", Statement: "A conjecture labelled doctrine for the marker test",
			FalsificationPredicate: "A reproducible counterexample under the stated condition.",
			Domain:                 "physics", Category: types.DoctrineCategory, Stake: "11000000",
		})
	case "contradiction":
		_, err = ms.SubmitContradiction(ctx, &types.MsgSubmitContradiction{
			Submitter: "submitter", FactId: "fact-verified", CounterClaim: "A counter-claim labelled doctrine",
			Category: types.DoctrineCategory, Stake: "11000000",
		})
	case "claim/legacy":
		_, err = ms.SubmitClaim(ctx, &types.MsgSubmitClaim{
			Submitter: "submitter", FactContent: "A claim labelled doctrine for the marker test",
			Domain: "physics", Category: types.DoctrineCategory, Stake: "11000000",
		})
	case "challenge":
		_, err = ms.ChallengeFact(ctx, &types.MsgChallengeFact{
			Challenger: "challenger", FactId: "commitment-1", Reason: "A challenge of genesis doctrine",
			Stake: EffectiveMinChallengeStake(params, types.DoctrineConfidence).String(),
		})
	case "provisional-challenge":
		_, err = ms.ChallengeProvisionalFact(ctx, &types.MsgChallengeProvisionalFact{
			Challenger: "challenger", FactId: "fact-conjecture", Reason: "A refutation of the conjecture",
			Stake: "11000000",
		})
	case "fitness":
		return k.UpdateAllFitnessScores(ctx)
	case "growth":
		return k.AdvanceConfidence(ctx)
	case "rescue":
		_, err = k.ResurrectDoctrineFacts(ctx)
		return err
	case "begin/growth":
		return k.BeginBlocker(ctx.WithBlockHeader(cmtproto.Header{Height: int64(params.ConfidenceGrowthEpoch)}))
	case "begin/fitness":
		return k.BeginBlocker(ctx.WithBlockHeader(cmtproto.Header{Height: int64(params.FitnessEpochBlocks)}))
	default:
		t.Fatalf("unknown entry %q", entry)
	}
	return err
}

// gasBeforeDoctrineMarker is what an entry point spends before it reads the
// doctrine-identity marker: the earlier, gas-metered marker reads of the
// message handlers. The doctrine-identity read itself is gas-free.
func gasBeforeDoctrineMarker(t *testing.T, k Keeper, ctx sdk.Context, entry string) storetypes.Gas {
	t.Helper()
	meter := storetypes.NewInfiniteGasMeter()
	ctx = ctx.WithGasMeter(meter)
	var err error
	switch entry {
	case "conjecture":
		_, err = k.FundSettlementEnabled(ctx)
	case "contradiction":
		_, err = k.ClaimRecordsEnabled(ctx)
	case "claim/legacy", "challenge", "provisional-challenge":
		_, err = k.RecordIntegrityEnabled(ctx)
	}
	require.NoError(t, err)
	return meter.GasConsumed()
}

// An unreadable or malformed doctrine-identity marker is a fault, never off.
// Every entry point that this repair makes read it returns the fault before
// any write or event, and the marker read charges no gas.
func TestDoctrineReservedMarkerFaultPropagates(t *testing.T) {
	for _, entry := range doctrineReservedEntries {
		for _, tc := range []struct {
			name  string
			value []byte // nil: the read itself fails
		}{
			{"unreadable", nil},
			{"malformed-0", []byte{0}},
			{"malformed-2", []byte{2}},
		} {
			t.Run(entry+"/"+tc.name, func(t *testing.T) {
				f := newDoctrineReservedFixture(t)
				if tc.value == nil {
					f.kFault.op, f.kFault.prefix = "get", types.DoctrineIdentityEnabledKey
				} else {
					f.ctx.KVStore(f.keys[0]).Set(types.DoctrineIdentityEnabledKey, tc.value)
				}
				before := f.snapshot(t)
				wantGas := gasBeforeDoctrineMarker(t, f.k, f.ctx, entry)
				meter, events := storetypes.NewInfiniteGasMeter(), sdk.NewEventManager()
				ctx := f.ctx.WithGasMeter(meter).WithEventManager(events)

				require.ErrorIs(t, runDoctrineReservedEntry(t, f.k, ctx, entry), activation.ErrMarkerFault)
				require.Equal(t, wantGas, meter.GasConsumed(), "the doctrine-identity marker read charges no gas")
				require.Empty(t, events.Events())
				require.Equal(t, before, f.snapshot(t), "the fault precedes every write")
			})
		}
	}
}

// Each of these entry points reads the marker exactly once. With the marker on
// the admissions refuse the doctrine label, while the challenges (no bank is
// wired, so no coins move), the epoch steps, the rescue and both BeginBlocker
// heights run to the end; none reads the marker a second time.
func TestDoctrineReservedMarkerReadOncePerEntryPoint(t *testing.T) {
	for _, entry := range doctrineReservedEntries {
		t.Run(entry, func(t *testing.T) {
			f := newDoctrineReservedFixture(t)
			err := runDoctrineReservedEntry(t, f.k, f.ctx, entry)
			switch entry {
			case "conjecture", "contradiction", "claim/legacy":
				require.ErrorIs(t, err, ErrDoctrineCategoryReserved)
			default:
				require.NoError(t, err)
			}
			require.Equal(t, 1, *f.reads)
		})
	}
}
