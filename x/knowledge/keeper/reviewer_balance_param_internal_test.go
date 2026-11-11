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
	"google.golang.org/protobuf/proto"

	"github.com/zerone-chain/zerone/internal/activation"
	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// reviewerBalanceReads counts reads of the reviewer-balance marker and of the
// params blob, and passes every call on to the handoff fault store, so one
// fixture can count reads and inject a read failure.
type reviewerBalanceReads struct {
	corestore.KVStoreService
	marker, params *int
}

func (s reviewerBalanceReads) OpenKVStore(ctx context.Context) corestore.KVStore {
	return reviewerBalanceReadsStore{s.KVStoreService.OpenKVStore(ctx), s.marker, s.params}
}

type reviewerBalanceReadsStore struct {
	corestore.KVStore
	marker, params *int
}

func (s reviewerBalanceReadsStore) Get(key []byte) ([]byte, error) {
	switch {
	case bytes.Equal(key, types.ReviewerBalanceParamEnabledKey):
		*s.marker++
	case bytes.Equal(key, types.ParamsKey):
		*s.params++
	}
	return s.KVStore.Get(key)
}

// reviewerBalanceBank holds one balance for every account and counts balance
// reads. Nothing in these tests moves coins.
type reviewerBalanceBank struct {
	types.BankKeeper
	balance int64
	reads   int
}

func (b *reviewerBalanceBank) GetBalance(_ context.Context, _ sdk.AccAddress, denom string) sdk.Coin {
	b.reads++
	return sdk.NewInt64Coin(denom, b.balance)
}

type reviewerBalanceFixture struct {
	k              Keeper
	server         types.MsgServer
	ctx            sdk.Context
	key            *storetypes.KVStoreKey
	fault          *handoffFault
	bank           *reviewerBalanceBank
	marker, params *int
}

const reviewerBalanceRound = "round-reviewer-balance"

// newReviewerBalanceInternalFixture is native DefaultGenesis with the marker
// on or off, the parameter written straight to the store (a marker-off chain
// cannot reach a configured parameter through genesis or a message), one
// legacy commit-phase round, and a reviewer holding 100 ZRN.
func newReviewerBalanceInternalFixture(t *testing.T, markerOn bool, param string) reviewerBalanceFixture {
	t.Helper()
	db := dbm.NewMemDB()
	ms := store.NewCommitMultiStore(db, log.NewNopLogger(), storemetrics.NewNoOpMetrics())
	key := storetypes.NewKVStoreKey(types.StoreKey)
	ms.MountStoreWithDB(key, storetypes.StoreTypeIAVL, db)
	require.NoError(t, ms.LoadLatestVersion())
	ctx := sdk.NewContext(ms, cmtproto.Header{Height: 100}, false, log.NewNopLogger())
	fault, marker, params := new(handoffFault), new(int), new(int)
	bank := &reviewerBalanceBank{balance: int64(types.BuiltInMinReviewerBalanceUzrn)}
	service := reviewerBalanceReads{handoffFaultService{runtime.NewKVStoreService(key), fault}, marker, params}
	k := NewKeeper(service, codec.NewProtoCodec(codectypes.NewInterfaceRegistry()), "authority", bank, nil)
	gs := types.DefaultGenesis()
	gs.ReviewerBalanceParamEnabled = markerOn
	require.NoError(t, k.InitGenesis(ctx, gs))
	p, err := k.GetParams(ctx)
	require.NoError(t, err)
	p.MinReviewerBalance = param
	require.NoError(t, k.SetParams(ctx, p))
	require.NoError(t, k.SetVerificationRound(ctx, &types.VerificationRound{
		Id: reviewerBalanceRound, ClaimId: "claim-reviewer-balance", StartedAtBlock: 98,
		Phase:          types.VerificationPhase_VERIFICATION_PHASE_COMMIT,
		CommitDeadline: 298, RevealDeadline: 498, AggregationDeadline: 548,
	}))
	ctx = ctx.WithGasMeter(storetypes.NewInfiniteGasMeter()).WithEventManager(sdk.NewEventManager())
	*marker, *params, bank.reads = 0, 0, 0
	return reviewerBalanceFixture{k: k, server: NewMsgServerImpl(k), ctx: ctx, key: key, fault: fault, bank: bank, marker: marker, params: params}
}

func (f reviewerBalanceFixture) commit(ctx sdk.Context) error {
	reviewer := sdk.AccAddress(bytes.Repeat([]byte{7}, 20)).String()
	_, err := f.server.SubmitCommitment(ctx, &types.MsgSubmitCommitment{
		Verifier: reviewer, RoundId: reviewerBalanceRound,
		CommitHash: types.ComputeCommitmentHash(reviewerBalanceRound, "accept", 0, []byte("salt")),
	})
	return err
}

func (f reviewerBalanceFixture) snapshot(t *testing.T) map[string][]byte {
	t.Helper()
	return handoffFixture{ctx: f.ctx, keys: []*storetypes.KVStoreKey{f.key}}.snapshot(t)
}

func (f reviewerBalanceFixture) updateReviewerBalance(t *testing.T, value string) error {
	t.Helper()
	p, err := f.k.GetParams(f.ctx)
	require.NoError(t, err)
	next := proto.Clone(p).(*types.Params)
	next.MinReviewerBalance = value
	_, err = f.server.UpdateParams(f.ctx, &types.MsgUpdateParams{Authority: "authority", Params: next})
	return err
}

// A marker that cannot be read, or holds anything but {1}, is a fault, never
// off. SubmitCommitment returns it before it reads the reviewer's balance or
// writes anything, and an update that changes min_reviewer_balance returns it
// before storing params. An update that leaves the field alone never reads the
// marker, so it still passes.
func TestReviewerBalanceMarkerFaultPropagates(t *testing.T) {
	for _, tc := range []struct {
		name  string
		value []byte // nil: the read itself fails
	}{
		{"unreadable", nil},
		{"malformed-0", []byte{0}},
		{"malformed-2", []byte{2}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newReviewerBalanceInternalFixture(t, true, "")
			if tc.value == nil {
				f.fault.op, f.fault.prefix = "get", types.ReviewerBalanceParamEnabledKey
			} else {
				f.ctx.KVStore(f.key).Set(types.ReviewerBalanceParamEnabledKey, tc.value)
			}
			before := f.snapshot(t)

			require.ErrorIs(t, f.commit(f.ctx), activation.ErrMarkerFault)
			require.Zero(t, f.bank.reads, "the fault returns before the balance read")
			require.ErrorIs(t, f.updateReviewerBalance(t, "222000"), activation.ErrMarkerFault)
			require.Empty(t, f.ctx.EventManager().Events())
			require.Equal(t, before, f.snapshot(t), "no commitment and no params are stored")

			p, err := f.k.GetParams(f.ctx)
			require.NoError(t, err)
			next := proto.Clone(p).(*types.Params)
			next.MinVerifiers++
			reads := *f.marker
			_, err = f.server.UpdateParams(f.ctx, &types.MsgUpdateParams{Authority: "authority", Params: next})
			require.NoError(t, err)
			require.Equal(t, reads, *f.marker, "an update that leaves the field alone does not read the marker")
		})
	}
}

// SubmitCommitment reads the marker once. With it off it reads no params;
// with it on it reads them once.
func TestReviewerBalanceMarkerReadOncePerCommitment(t *testing.T) {
	for _, tc := range []struct {
		name     string
		markerOn bool
		params   int
	}{
		{"off", false, 0},
		{"on", true, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newReviewerBalanceInternalFixture(t, tc.markerOn, "222000")
			require.NoError(t, f.commit(f.ctx))
			require.Equal(t, 1, *f.marker)
			require.Equal(t, tc.params, *f.params)
			require.Equal(t, 1, f.bank.reads)
		})
	}
}

// With the marker off the figure costs no gas at all: the marker read is
// gas-free and no param is read, so SubmitCommitment spends exactly what it
// spent before, whatever the parameter holds. With the marker on the param
// read is charged as any param read is.
func TestReviewerBalanceMarkerOffGasUnchanged(t *testing.T) {
	gasOf := func(markerOn bool, param string) (figureGas, commitGas uint64) {
		f := newReviewerBalanceInternalFixture(t, markerOn, param)
		meter := storetypes.NewInfiniteGasMeter()
		_, err := f.k.reviewerMinBalance(f.ctx.WithGasMeter(meter))
		require.NoError(t, err)
		figureGas = meter.GasConsumed()
		meter = storetypes.NewInfiniteGasMeter()
		require.NoError(t, f.commit(f.ctx.WithGasMeter(meter)))
		return figureGas, meter.GasConsumed()
	}
	offEmptyFigure, offEmpty := gasOf(false, "")
	offSetFigure, offSet := gasOf(false, "1")
	require.Zero(t, offEmptyFigure, "with the marker off the figure is free")
	require.Zero(t, offSetFigure)
	require.Equal(t, offEmpty, offSet, "with the marker off a configured parameter changes no gas")

	onEmptyFigure, onEmpty := gasOf(true, "")
	_, onSet := gasOf(true, "1")
	require.Positive(t, onEmptyFigure, "with the marker on the param read is charged")
	require.Equal(t, offEmpty+onEmptyFigure, onEmpty, "the param read is the only gas the marker adds")
	require.NotEqual(t, onEmpty, onSet, "the comparison above would see a param read")
}
