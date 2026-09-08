package keeper_test

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"strings"
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
	"google.golang.org/protobuf/proto"

	"github.com/zerone-chain/zerone/x/knowledge/keeper"
	"github.com/zerone-chain/zerone/x/knowledge/types"
)

func feedbackConsumer(n byte) string { return sdk.AccAddress(bytes.Repeat([]byte{n}, 20)).String() }

func enableFeedback(t *testing.T, k keeper.Keeper, ctx sdk.Context, consumers ...string) {
	t.Helper()
	p, err := k.GetParams(ctx)
	require.NoError(t, err)
	p.FactUseEnabled, p.FactUseConsumers = true, consumers
	p.FitnessWeightQueryBps, p.FitnessWeightSatisfactionBps, p.MetabolismEnergyPerQuery = 0, 0, 0
	require.NoError(t, p.Validate())
	require.NoError(t, k.SetParams(ctx, p))
}

// Instrument real SDK cache-backed storage, not a fake transaction transport.
// Faults occur before a requested module operation; rejected handlers must leave
// every module key and emitted event unchanged even after earlier cached writes.
type feedbackStoreService struct {
	corestore.KVStoreService
	op                        string
	prefix                    []byte
	open                      int
	examined                  int
	iteratorError, closeError bool
}
type feedbackStore struct {
	corestore.KVStore
	s *feedbackStoreService
}

var feedbackStoreErr = errors.New("injected feedback store failure")

func (s *feedbackStoreService) OpenKVStore(ctx context.Context) corestore.KVStore {
	return feedbackStore{s.KVStoreService.OpenKVStore(ctx), s}
}
func (s *feedbackStoreService) fails(op string, key []byte) bool {
	return s.op == op && bytes.HasPrefix(key, s.prefix)
}
func (s feedbackStore) Get(key []byte) ([]byte, error) {
	if s.s.fails("get", key) {
		return nil, feedbackStoreErr
	}
	return s.KVStore.Get(key)
}
func (s feedbackStore) Set(key, value []byte) error {
	if s.s.open != 0 {
		return errors.New("write while iterator open")
	}
	if s.s.fails("set", key) {
		return feedbackStoreErr
	}
	return s.KVStore.Set(key, value)
}
func (s feedbackStore) Delete(key []byte) error {
	if s.s.open != 0 {
		return errors.New("delete while iterator open")
	}
	if s.s.fails("delete", key) {
		return feedbackStoreErr
	}
	return s.KVStore.Delete(key)
}
func (s feedbackStore) Iterator(start, end []byte) (corestore.Iterator, error) {
	if s.s.fails("iterator", start) {
		return nil, feedbackStoreErr
	}
	it, err := s.KVStore.Iterator(start, end)
	if err != nil {
		return nil, err
	}
	s.s.open++
	return &feedbackIterator{Iterator: it, s: s.s}, nil
}

type feedbackIterator struct {
	corestore.Iterator
	s *feedbackStoreService
}

func (i *feedbackIterator) Value() []byte { i.s.examined++; return i.Iterator.Value() }
func (i *feedbackIterator) Error() error {
	if i.s.iteratorError {
		return feedbackStoreErr
	}
	return i.Iterator.Error()
}
func (i *feedbackIterator) Close() error {
	i.s.open--
	err := i.Iterator.Close()
	if i.s.closeError {
		return errors.Join(err, feedbackStoreErr)
	}
	return err
}

func setupFeedbackStore(t *testing.T) (keeper.Keeper, sdk.Context, *feedbackStoreService) {
	t.Helper()
	key := storetypes.NewKVStoreKey(types.StoreKey)
	db := dbm.NewMemDB()
	ms := store.NewCommitMultiStore(db, log.NewNopLogger(), storemetrics.NewNoOpMetrics())
	ms.MountStoreWithDB(key, storetypes.StoreTypeIAVL, db)
	require.NoError(t, ms.LoadLatestVersion())
	svc := &feedbackStoreService{KVStoreService: runtime.NewKVStoreService(key)}
	k := keeper.NewKeeper(svc, codec.NewProtoCodec(codectypes.NewInterfaceRegistry()), "authority", nil, nil)
	ctx := sdk.NewContext(ms, cmtproto.Header{Height: 9}, false, log.NewNopLogger())
	p := types.DefaultParams()
	p.FitnessEpochBlocks = 10
	require.NoError(t, k.SetParams(ctx, &p))
	enableFeedback(t, k, ctx, feedbackConsumer(1), feedbackConsumer(2))
	return k, ctx, svc
}
func feedbackSnapshot(t *testing.T, s *feedbackStoreService, ctx sdk.Context) map[string]string {
	t.Helper()
	raw := s.KVStoreService.OpenKVStore(ctx)
	it, err := raw.Iterator(nil, nil)
	require.NoError(t, err)
	out := map[string]string{}
	for ; it.Valid(); it.Next() {
		out[string(it.Key())] = string(it.Value())
	}
	require.NoError(t, it.Error())
	require.NoError(t, it.Close())
	return out
}
func feedbackFact(t *testing.T, k keeper.Keeper, ctx sdk.Context, id string) *types.Fact {
	t.Helper()
	f := &types.Fact{Id: id, Status: types.FactStatus_FACT_STATUS_VERIFIED, Energy: 100, Confidence: 800000}
	require.NoError(t, k.SetFact(ctx, f))
	return f
}

func TestFactUseReportRatingAndCurrentEpochRPC(t *testing.T) {
	k, ctx, s := setupFeedbackStore(t)
	f := feedbackFact(t, k, ctx, "fact")
	m := keeper.NewMsgServerImpl(k)
	q := keeper.NewQueryServerImpl(k)
	consumer := feedbackConsumer(1)
	before := feedbackSnapshot(t, s, ctx)
	missing, err := q.FactUseReceipt(ctx, &types.QueryFactUseReceiptRequest{Consumer: consumer, FactId: f.Id})
	require.NoError(t, err)
	require.False(t, missing.Found)
	require.Zero(t, missing.Epoch)
	require.Equal(t, uint64(9), missing.SnapshotBlockHeight)
	require.Equal(t, before, feedbackSnapshot(t, s, ctx))
	resp, err := m.ReportFactUse(ctx, &types.MsgReportFactUse{Consumer: consumer, FactId: f.Id})
	require.NoError(t, err)
	require.Equal(t, uint64(20), resp.Receipt.ExpiryHeight)
	state, err := k.GetFactUsePruningState(ctx)
	require.NoError(t, err)
	require.True(t, state.EverReported)
	afterReport := feedbackSnapshot(t, s, ctx)
	read, err := q.FactUseReceipt(ctx, &types.QueryFactUseReceiptRequest{Consumer: consumer, FactId: f.Id})
	require.NoError(t, err)
	require.True(t, read.Found)
	require.True(t, proto.Equal(resp.Receipt, read.Receipt))
	require.Equal(t, afterReport, feedbackSnapshot(t, s, ctx))
	_, err = m.RateFact(ctx, &types.MsgRateFact{Rater: consumer, FactId: f.Id, Useful: true})
	require.NoError(t, err)
	r, found, err := k.GetFactUseReceipt(ctx, 0, consumer, f.Id)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, types.FactUseRating_FACT_USE_RATING_USEFUL, r.Rating)
	afterRating := feedbackSnapshot(t, s, ctx)
	events := len(ctx.EventManager().Events())
	// A new message allocation models semantic replay with a fresh tx nonce;
	// this keeper test does not claim to test account sequences or signatures.
	_, err = m.ReportFactUse(ctx, &types.MsgReportFactUse{Consumer: consumer, FactId: f.Id})
	require.ErrorContains(t, err, "already reported")
	_, err = m.RateFact(ctx, &types.MsgRateFact{Rater: consumer, FactId: f.Id})
	require.ErrorContains(t, err, "already rated")
	require.Equal(t, afterRating, feedbackSnapshot(t, s, ctx))
	require.Len(t, ctx.EventManager().Events(), events)
	updated, found := k.GetFact(ctx, f.Id)
	require.True(t, found)
	require.Equal(t, f.Status, updated.Status)
	require.Equal(t, f.Energy, updated.Energy)
	require.Equal(t, uint64(1), updated.QueryCount)
	require.Equal(t, uint64(1), updated.SatisfactionUp)
	for _, height := range []int64{10, 11} {
		at := ctx.WithBlockHeight(height)
		read, err := q.FactUseReceipt(at, &types.QueryFactUseReceiptRequest{Consumer: consumer, FactId: f.Id})
		require.NoError(t, err)
		require.Equal(t, uint64(1), read.Epoch)
		require.False(t, read.Found)
		_, err = m.RateFact(at, &types.MsgRateFact{Rater: consumer, FactId: f.Id})
		require.ErrorContains(t, err, "current-epoch")
	}
	_, err = m.ReportFactUse(ctx.WithBlockHeight(10), &types.MsgReportFactUse{Consumer: consumer, FactId: f.Id})
	require.NoError(t, err)
	_, err = m.RateFact(ctx.WithBlockHeight(11), &types.MsgRateFact{Rater: consumer, FactId: f.Id})
	require.NoError(t, err)
}

func TestFactUseAdmissionAndQuotaRejectionsAreAtomic(t *testing.T) {
	k, ctx, s := setupFeedbackStore(t)
	m := keeper.NewMsgServerImpl(k)
	consumer := feedbackConsumer(1)
	for _, id := range []string{"one", "two", "three", "four"} {
		feedbackFact(t, k, ctx, id)
	}
	p, err := k.GetParams(ctx)
	require.NoError(t, err)
	p.FactUseMaxPerConsumerEpoch = 2
	p.FactUseMaxPerEpoch = 3
	require.NoError(t, k.SetParams(ctx, p))
	for _, msg := range []*types.MsgReportFactUse{nil, {Consumer: strings.ToUpper(consumer), FactId: "one"}, {Consumer: feedbackConsumer(3), FactId: "one"}, {Consumer: consumer, FactId: "missing"}, {Consumer: consumer, FactId: "bad/id"}} {
		before := feedbackSnapshot(t, s, ctx)
		_, err := m.ReportFactUse(ctx, msg)
		require.Error(t, err)
		require.Equal(t, before, feedbackSnapshot(t, s, ctx))
	}
	for _, id := range []string{"one", "two"} {
		_, err := m.ReportFactUse(ctx, &types.MsgReportFactUse{Consumer: consumer, FactId: id})
		require.NoError(t, err)
	}
	before := feedbackSnapshot(t, s, ctx)
	_, err = m.ReportFactUse(ctx, &types.MsgReportFactUse{Consumer: consumer, FactId: "three"})
	require.ErrorContains(t, err, "consumer")
	require.Equal(t, before, feedbackSnapshot(t, s, ctx))
	_, err = m.ReportFactUse(ctx, &types.MsgReportFactUse{Consumer: feedbackConsumer(2), FactId: "three"})
	require.NoError(t, err)
	before = feedbackSnapshot(t, s, ctx)
	_, err = m.ReportFactUse(ctx, &types.MsgReportFactUse{Consumer: feedbackConsumer(2), FactId: "four"})
	require.ErrorContains(t, err, "global")
	require.Equal(t, before, feedbackSnapshot(t, s, ctx))
	p.FactUseEnabled = false
	require.NoError(t, k.SetParams(ctx, p))
	before = feedbackSnapshot(t, s, ctx)
	_, err = m.ReportFactUse(ctx, &types.MsgReportFactUse{Consumer: consumer, FactId: "four"})
	require.ErrorContains(t, err, "disabled")
	require.Equal(t, before, feedbackSnapshot(t, s, ctx))
	_, err = m.RateFact(ctx, &types.MsgRateFact{Rater: consumer, FactId: "one"})
	require.ErrorContains(t, err, "disabled")
}

func TestFactUseParamUpdateChecksStoredStateErrors(t *testing.T) {
	for _, tc := range []struct {
		op     string
		prefix []byte
	}{{"get", types.ParamsKey}, {"get", types.FactUsePruningStateKey}, {"iterator", types.FactUseReceiptPrefix}} {
		t.Run(fmt.Sprintf("%s-%x", tc.op, tc.prefix), func(t *testing.T) {
			k, ctx, s := setupFeedbackStore(t)
			p, err := k.GetParams(ctx)
			require.NoError(t, err)
			p.FactUseEnabled = false
			before := feedbackSnapshot(t, s, ctx)
			s.op, s.prefix = tc.op, tc.prefix
			_, err = keeper.NewMsgServerImpl(k).UpdateParams(ctx, &types.MsgUpdateParams{Authority: k.GetAuthority(), Params: p})
			require.ErrorIs(t, err, feedbackStoreErr)
			require.Equal(t, before, feedbackSnapshot(t, s, ctx))
		})
	}
}

func TestFactUseCounterCorruptionAndOverflow(t *testing.T) {
	for _, name := range []string{"missing-global", "short-global", "zero-global", "overflow-global", "missing-account", "overflow-account", "stat-lifetime", "stat-epoch", "corrupt-params", "corrupt-fact", "corrupt-pruning"} {
		t.Run(name, func(t *testing.T) {
			k, ctx, s := setupFeedbackStore(t)
			m := keeper.NewMsgServerImpl(k)
			consumer := feedbackConsumer(1)
			feedbackFact(t, k, ctx, "one")
			f := feedbackFact(t, k, ctx, "two")
			_, err := m.ReportFactUse(ctx, &types.MsgReportFactUse{Consumer: consumer, FactId: "one"})
			require.NoError(t, err)
			raw := s.KVStoreService.OpenKVStore(ctx)
			key := types.FactUseEpochCountKey(0)
			switch name {
			case "missing-global":
				require.NoError(t, raw.Delete(key))
			case "short-global":
				require.NoError(t, raw.Set(key, []byte{1}))
			case "zero-global":
				require.NoError(t, raw.Set(key, make([]byte, 8)))
			case "overflow-global":
				require.NoError(t, raw.Set(key, binary.BigEndian.AppendUint64(nil, math.MaxUint64)))
			case "missing-account":
				require.NoError(t, raw.Delete(types.FactUseConsumerCountKey(0, consumer)))
			case "overflow-account":
				require.NoError(t, raw.Set(types.FactUseConsumerCountKey(0, consumer), binary.BigEndian.AppendUint64(nil, math.MaxUint64)))
			case "stat-lifetime":
				f.QueryCount = math.MaxUint64
				require.NoError(t, k.SetFact(ctx, f))
			case "stat-epoch":
				f.QueryCountEpoch = math.MaxUint64
				require.NoError(t, k.SetFact(ctx, f))
			case "corrupt-params":
				require.NoError(t, raw.Set(types.ParamsKey, []byte{255}))
			case "corrupt-fact":
				require.NoError(t, raw.Set(types.FactKey(f.Id), []byte{255}))
			case "corrupt-pruning":
				require.NoError(t, raw.Set(types.FactUsePruningStateKey, []byte{255}))
			}
			before := feedbackSnapshot(t, s, ctx)
			_, err = m.ReportFactUse(ctx, &types.MsgReportFactUse{Consumer: consumer, FactId: f.Id})
			require.Error(t, err)
			require.Equal(t, before, feedbackSnapshot(t, s, ctx))
		})
	}
}

func TestFactUseStoreFailuresDiscardAllModuleEffects(t *testing.T) {
	for _, op := range []string{"get", "set", "iterator", "iterator-error", "close-error"} {
		for _, prefix := range [][]byte{types.ParamsKey, types.FactKeyPrefix, types.FactUseReceiptPrefix, types.FactUseEpochCountPrefix, types.FactUseConsumerCountPrefix, types.FactUsePruningStateKey} {
			if op == "set" && bytes.Equal(prefix, types.ParamsKey) {
				continue
			}
			if strings.HasPrefix(op, "iterator") || op == "close-error" {
				if !bytes.Equal(prefix, types.FactUseReceiptPrefix) {
					continue
				}
			}
			t.Run(fmt.Sprintf("%s-%x", op, prefix), func(t *testing.T) {
				k, ctx, s := setupFeedbackStore(t)
				feedbackFact(t, k, ctx, "fact")
				before := feedbackSnapshot(t, s, ctx)
				events := len(ctx.EventManager().Events())
				s.op, s.prefix = op, prefix
				s.iteratorError = op == "iterator-error"
				s.closeError = op == "close-error"
				_, err := keeper.NewMsgServerImpl(k).ReportFactUse(ctx, &types.MsgReportFactUse{Consumer: feedbackConsumer(1), FactId: "fact"})
				require.Error(t, err)
				require.Equal(t, before, feedbackSnapshot(t, s, ctx))
				require.Len(t, ctx.EventManager().Events(), events)
				require.Zero(t, s.open)
			})
		}
	}
}

func TestFactUseRatingFailuresRetainUnratedMarker(t *testing.T) {
	for _, name := range []string{"memo", "utf8", "overflow-up", "overflow-down", "write-fact", "write-receipt", "read-receipt"} {
		t.Run(name, func(t *testing.T) {
			k, ctx, s := setupFeedbackStore(t)
			f := feedbackFact(t, k, ctx, "fact")
			m := keeper.NewMsgServerImpl(k)
			consumer := feedbackConsumer(1)
			_, err := m.ReportFactUse(ctx, &types.MsgReportFactUse{Consumer: consumer, FactId: f.Id})
			require.NoError(t, err)
			msg := &types.MsgRateFact{Rater: consumer, FactId: f.Id, Useful: true}
			switch name {
			case "memo":
				msg.Memo = strings.Repeat("a", 257)
			case "utf8":
				msg.Memo = string([]byte{255})
			case "overflow-up":
				f.SatisfactionUp = math.MaxUint64
				require.NoError(t, k.SetFact(ctx, f))
			case "overflow-down":
				msg.Useful = false
				f.SatisfactionDownEpoch = math.MaxUint64
				require.NoError(t, k.SetFact(ctx, f))
			case "write-fact":
				s.op, s.prefix = "set", types.FactKeyPrefix
			case "write-receipt":
				s.op, s.prefix = "set", types.FactUseReceiptPrefix
			case "read-receipt":
				s.op, s.prefix = "get", types.FactUseReceiptPrefix
			}
			before := feedbackSnapshot(t, s, ctx)
			_, err = m.RateFact(ctx, msg)
			require.Error(t, err)
			require.Equal(t, before, feedbackSnapshot(t, s, ctx))
		})
	}
}

func feedbackReceipt(epoch uint64, consumer, fact string) *types.FactUseReceipt {
	height := epoch * 10
	if height == 0 {
		height = 1
	}
	return &types.FactUseReceipt{Version: 1, Epoch: epoch, Consumer: consumer, FactId: fact, UseHeight: height, ExpiryHeight: (epoch + 2) * 10, Rating: types.FactUseRating_FACT_USE_RATING_UNRATED}
}

func TestFactUseBoundedPruningRestartAndInsertion(t *testing.T) {
	k, ctx, s := setupFeedbackStore(t)
	var receipts []*types.FactUseReceipt
	for i := 0; i < 201; i++ {
		id := fmt.Sprintf("f%03d", i)
		feedbackFact(t, k, ctx, id)
		receipts = append(receipts, feedbackReceipt(0, feedbackConsumer(byte(i/100+1)), id))
	}
	require.NoError(t, k.InitFactUseReceipts(ctx, receipts, &types.FactUsePruningState{EverReported: true}))
	s.examined = 0
	require.NoError(t, k.PruneFactUseReceipts(ctx))
	require.LessOrEqual(t, s.examined, 100)
	require.Equal(t, 100, s.examined)
	pruning, err := k.GetFactUsePruningState(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, pruning.NextKey)
	// Save/restore actual cursor and records through the public helpers.
	exported, err := k.ExportFactUseReceipts(ctx)
	require.NoError(t, err)
	k2, ctx2, s2 := setupFeedbackStore(t)
	for _, r := range exported {
		feedbackFact(t, k2, ctx2, r.FactId)
	}
	require.NoError(t, k2.InitFactUseReceipts(ctx2, exported, pruning))
	p2, err := k2.GetFactUsePruningState(ctx2)
	require.NoError(t, err)
	require.True(t, proto.Equal(pruning, p2))
	// A new current-epoch key sorts before the cursor. It must eventually be
	// visited after wrap even though it did not exist during the previous scan.
	feedbackFact(t, k2, ctx2, "a-new")
	m := keeper.NewMsgServerImpl(k2)
	// Find a fresh canonical cohort member whose new key is strictly BEFORE
	// the saved cursor (bech32 lexical order is not the numeric account order).
	var insertedConsumer string
	for n := byte(4); n < 64; n++ {
		candidate := feedbackConsumer(n)
		if bytes.Compare(types.FactUseReceiptKey(0, candidate, "a-new"), pruning.NextKey) < 0 {
			insertedConsumer = candidate
			break
		}
	}
	require.NotEmpty(t, insertedConsumer)
	enableFeedback(t, k2, ctx2, insertedConsumer)
	_, err = m.ReportFactUse(ctx2, &types.MsgReportFactUse{Consumer: insertedConsumer, FactId: "a-new"})
	require.NoError(t, err)
	for i := 0; i < 8; i++ {
		s2.examined = 0
		require.NoError(t, k2.PruneFactUseReceipts(ctx2.WithBlockHeight(20+int64(i))))
		require.LessOrEqual(t, s2.examined, 100)
	}
	retained, err := k2.HasRetainedFactUseReceipts(ctx2)
	require.NoError(t, err)
	require.False(t, retained)
	state, err := k2.GetFactUsePruningState(ctx2)
	require.NoError(t, err)
	require.True(t, state.EverReported)
	require.Empty(t, state.NextKey)
	raw := s2.KVStoreService.OpenKVStore(ctx2)
	for _, p := range [][]byte{types.FactUseEpochCountPrefix, types.FactUseConsumerCountPrefix} {
		it, err := raw.Iterator(p, []byte{p[0] + 1})
		require.NoError(t, err)
		require.False(t, it.Valid())
		require.NoError(t, it.Close())
	}
}

func TestFactUseExpiryAndPruneFailureAtomicity(t *testing.T) {
	for _, failure := range []string{"none", "delete", "counter-get", "counter-set", "state-set", "missing-count", "iterator-error", "close-error"} {
		t.Run(failure, func(t *testing.T) {
			k, ctx, s := setupFeedbackStore(t)
			feedbackFact(t, k, ctx, "fact")
			m := keeper.NewMsgServerImpl(k)
			_, err := m.ReportFactUse(ctx, &types.MsgReportFactUse{Consumer: feedbackConsumer(1), FactId: "fact"})
			require.NoError(t, err)
			require.NoError(t, k.PruneFactUseReceipts(ctx.WithBlockHeight(19)))
			found, err := k.HasRetainedFactUseReceipts(ctx)
			require.NoError(t, err)
			require.True(t, found)
			switch failure {
			case "delete":
				s.op, s.prefix = "delete", types.FactUseReceiptPrefix
			case "counter-get":
				s.op, s.prefix = "get", types.FactUseEpochCountPrefix
			case "counter-set":
				s.op, s.prefix = "delete", types.FactUseEpochCountPrefix
			case "state-set":
				s.op, s.prefix = "set", types.FactUsePruningStateKey
			case "missing-count":
				require.NoError(t, s.KVStoreService.OpenKVStore(ctx).Delete(types.FactUseEpochCountKey(0)))
			case "iterator-error":
				s.iteratorError = true
			case "close-error":
				s.closeError = true
			}
			before := feedbackSnapshot(t, s, ctx)
			err = k.PruneFactUseReceipts(ctx.WithBlockHeight(20))
			if failure == "none" {
				require.NoError(t, err)
				found, err := k.HasRetainedFactUseReceipts(ctx)
				require.NoError(t, err)
				require.False(t, found)
			} else {
				require.Error(t, err)
				require.Equal(t, before, feedbackSnapshot(t, s, ctx))
			}
		})
	}
}

func TestFactUsePermanentNonEconomicLockAndEpochFreeze(t *testing.T) {
	k, ctx, _ := setupFeedbackStore(t)
	feedbackFact(t, k, ctx, "fact")
	m := keeper.NewMsgServerImpl(k)
	_, err := m.ReportFactUse(ctx, &types.MsgReportFactUse{Consumer: feedbackConsumer(1), FactId: "fact"})
	require.NoError(t, err)
	p, err := k.GetParams(ctx)
	require.NoError(t, err)
	change := func(p *types.Params) error {
		_, err := m.UpdateParams(ctx, &types.MsgUpdateParams{Authority: k.GetAuthority(), Params: p})
		return err
	}
	p.FitnessEpochBlocks = 11
	require.ErrorContains(t, change(p), "frozen")
	p.FitnessEpochBlocks = 10
	p.FactUseEnabled = false
	require.NoError(t, change(p))
	for _, pruned := range []bool{false, true} {
		if pruned {
			require.NoError(t, k.PruneFactUseReceipts(ctx.WithBlockHeight(20)))
		}
		for _, weight := range []string{"query", "satisfaction", "energy"} {
			bad := proto.Clone(p).(*types.Params)
			switch weight {
			case "query":
				bad.FitnessWeightQueryBps = 1
			case "satisfaction":
				bad.FitnessWeightSatisfactionBps = 1
			case "energy":
				bad.MetabolismEnergyPerQuery = 1
			}
			require.ErrorContains(t, change(bad), "ever_reported")
		}
	}
	p.FitnessEpochBlocks = 11
	require.NoError(t, change(p)) // Pruned+disabled releases ONLY epoch freeze.
	require.ErrorContains(t, k.InitFactUseReceipts(ctx, nil, nil), "ever_reported")
}

func TestFactUseRetainedCapacityAndImportValidation(t *testing.T) {
	k, ctx, s := setupFeedbackStore(t)
	var receipts []*types.FactUseReceipt
	for i := 0; i < 100; i++ {
		feedbackFact(t, k, ctx, fmt.Sprintf("f%03d", i))
	}
	for epoch := uint64(0); epoch < 2; epoch++ {
		for consumer := byte(1); consumer <= 10; consumer++ {
			for i := 0; i < 100; i++ {
				receipts = append(receipts, feedbackReceipt(epoch, feedbackConsumer(consumer), fmt.Sprintf("f%03d", i)))
			}
		}
	}
	require.NoError(t, k.InitFactUseReceipts(ctx, receipts, &types.FactUsePruningState{EverReported: true}))
	before := feedbackSnapshot(t, s, ctx)
	_, err := keeper.NewMsgServerImpl(k).ReportFactUse(ctx.WithBlockHeight(20), &types.MsgReportFactUse{Consumer: feedbackConsumer(1), FactId: "f000"})
	require.ErrorContains(t, err, "capacity")
	require.Equal(t, before, feedbackSnapshot(t, s, ctx))
	for _, name := range []string{"duplicate", "no-latch", "missing-fact", "bad-expiry", "too-many", "bad-cursor"} {
		t.Run(name, func(t *testing.T) {
			k, ctx, s := setupFeedbackStore(t)
			feedbackFact(t, k, ctx, "fact")
			r := feedbackReceipt(0, feedbackConsumer(1), "fact")
			rs := []*types.FactUseReceipt{r}
			p := &types.FactUsePruningState{EverReported: true}
			switch name {
			case "duplicate":
				rs = append(rs, proto.Clone(r).(*types.FactUseReceipt))
			case "no-latch":
				p.EverReported = false
			case "missing-fact":
				r.FactId = "missing"
			case "bad-expiry":
				r.ExpiryHeight = 19
			case "too-many":
				rs = make([]*types.FactUseReceipt, 2001)
			case "bad-cursor":
				p.NextKey = []byte{255}
			}
			before := feedbackSnapshot(t, s, ctx)
			require.Error(t, k.InitFactUseReceipts(ctx, rs, p))
			require.Equal(t, before, feedbackSnapshot(t, s, ctx))
		})
	}
}
