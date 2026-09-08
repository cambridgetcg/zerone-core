package main

import (
	"bytes"
	"testing"

	"cosmossdk.io/collections"
	"cosmossdk.io/log"
	sdkmath "cosmossdk.io/math"
	"cosmossdk.io/store/metrics"
	"cosmossdk.io/store/rootmulti"
	storetypes "cosmossdk.io/store/types"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/crypto/merkle"
	cmtcrypto "github.com/cometbft/cometbft/proto/tendermint/crypto"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/address"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	banktestutil "github.com/cosmos/cosmos-sdk/x/bank/testutil"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	ics23 "github.com/cosmos/ics23/go"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	schedule "github.com/zerone-chain/zerone/x/schedule/types"
)

// Account lookups are stubbed, but balance writes, reverse indexes, transfers,
// commits and both proof layers use the actual SDK. No app, signer or disk DB.
func indexedBankProofFixture(t *testing.T) (*rootmulti.Store, sdk.Context, bankkeeper.BaseSendKeeper, map[string]*storetypes.KVStoreKey) {
	t.Helper()
	db := dbm.NewMemDB()
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	store := rootmulti.NewStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	mounts := map[string]*storetypes.KVStoreKey{}
	for _, name := range []string{banktypes.StoreKey, schedule.StoreKey} {
		mounts[name] = storetypes.NewKVStoreKey(name)
		store.MountStoreWithDB(mounts[name], storetypes.StoreTypeIAVL, nil)
	}
	require.NoError(t, store.LoadVersion(0))
	require.NoError(t, store.SetInitialVersion(120))
	ctx := sdk.NewContext(store, cmtproto.Header{Height: 120}, false, log.NewNopLogger())
	ak := banktestutil.NewMockAccountKeeper(gomock.NewController(t))
	ak.EXPECT().AddressCodec().Return(address.NewBech32Codec("zrn")).AnyTimes()
	ak.EXPECT().GetAccount(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	ak.EXPECT().HasAccount(gomock.Any(), gomock.Any()).Return(true).AnyTimes()
	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	bank := bankkeeper.NewBaseSendKeeper(cdc, runtime.NewKVStoreService(mounts[banktypes.StoreKey]), ak, nil, sdk.AccAddress(bytes.Repeat([]byte{1}, 20)).String(), log.NewNopLogger())
	return store, ctx, bank, mounts
}

func bankProofExpectation(t *testing.T, addr sdk.AccAddress, amount string) stateKeyExpectation {
	t.Helper()
	e := fixtureExpectation(fixtureChainID, 120)
	e.Balances = []balanceExpectation{{Address: addr.String(), Amount: amount}}
	keys, err := schedulerKeys(e, fixtureChainID)
	require.NoError(t, err)
	for _, key := range keys {
		if key.Store == banktypes.StoreKey {
			return key
		}
	}
	t.Fatal("missing bank expectation")
	return stateKeyExpectation{}
}

func queryBankProofFixture(t *testing.T, store *rootmulti.Store, expected stateKeyExpectation) abci.ResponseQuery {
	t.Helper()
	res, err := store.Query(&storetypes.RequestQuery{Path: "/" + expected.Store + "/key", Data: expected.Key, Height: 120, Prove: true})
	require.NoError(t, err)
	require.Zero(t, res.Code)
	require.Len(t, res.ProofOps.Ops, 2)
	require.Equal(t, storetypes.ProofOpIAVLCommitment, res.ProofOps.Ops[0].Type)
	require.Equal(t, storetypes.ProofOpSimpleMerkleCommitment, res.ProofOps.Ops[1].Type)
	return abci.ResponseQuery{Code: res.Code, Height: res.Height, Key: res.Key, Value: res.Value, ProofOps: res.ProofOps}
}

func TestSchedulerBankZeroCodecAndStrictExpectations(t *testing.T) {
	zero, err := banktypes.BalanceValueCodec.Encode(sdkmath.ZeroInt())
	require.NoError(t, err)
	require.Equal(t, []byte("0"), zero) // SDK zero is not an empty value.
	decoded, err := banktypes.BalanceValueCodec.Decode(zero)
	require.NoError(t, err)
	require.True(t, decoded.IsZero())
	empty, err := banktypes.BalanceValueCodec.Decode([]byte{})
	require.NoError(t, err)
	require.True(t, empty.IsNil()) // An uninitialized Int is not encoded zero.

	addr := sdk.AccAddress(bytes.Repeat([]byte{2}, 20))
	zeroKey := bankProofExpectation(t, addr, "0")
	require.True(t, zeroKey.Absent)
	require.Equal(t, zero, zeroKey.Value)
	nonzeroKey := bankProofExpectation(t, addr, "7")
	require.False(t, nonzeroKey.Absent)
	require.Equal(t, []byte("7"), nonzeroKey.Value)
}

func TestSchedulerBankZeroRepresentations(t *testing.T) {
	for _, representation := range []string{"absent", "included codec zero", "included empty", "nonzero"} {
		t.Run(representation, func(t *testing.T) {
			store, ctx, bank, mounts := indexedBankProofFixture(t)
			addr := sdk.AccAddress(bytes.Repeat([]byte{2}, 20))
			expected := bankProofExpectation(t, addr, "0")
			// Nonempty balance neighbors isolate representation from the known
			// upstream empty reverse-index neighbor failure tested separately.
			for _, b := range []byte{1, 3} {
				require.NoError(t, bank.Balances.Set(ctx, collections.Join(sdk.AccAddress(bytes.Repeat([]byte{b}, 20)), schedule.Denom), sdkmath.NewInt(5)))
			}
			switch representation {
			case "included codec zero":
				// Direct collection insertion is deliberately not a valid bank
				// transfer: SDK setBalance deletes zero instead of persisting it.
				require.NoError(t, bank.Balances.Set(ctx, collections.Join(addr, schedule.Denom), sdkmath.ZeroInt()))
			case "included empty":
				store.GetKVStore(mounts[banktypes.StoreKey]).Set(expected.Key, []byte{})
			case "nonzero":
				require.NoError(t, bank.Balances.Set(ctx, collections.Join(addr, schedule.Denom), sdkmath.NewInt(7)))
			}
			commit := store.Commit()
			response := queryBankProofFixture(t, store, expected)
			proof := new(ics23.CommitmentProof)
			require.NoError(t, proof.Unmarshal(response.ProofOps.Ops[0].Data))
			if representation == "absent" {
				require.Nil(t, response.Value)
				require.NotNil(t, proof.GetNonexist())
				require.NoError(t, verifyStateProof(response, expected, 120, commit.Hash))
				return
			}
			// Even a genuine membership proof cannot be used as strict absence.
			require.NotNil(t, proof.GetExist())
			require.Error(t, verifyStateProof(response, expected, 120, commit.Hash))
			member := expected
			member.Absent = false
			member.Value = response.Value
			if representation == "included empty" {
				require.Empty(t, response.Value)
				require.ErrorContains(t, verifyStateProof(response, member, 120, commit.Hash), "leaf op needs value")
			} else {
				require.NoError(t, verifyStateProof(response, member, 120, commit.Hash))
			}
		})
	}
}

func TestSchedulerSDKZeroTransferEmptyIndexNeighbor(t *testing.T) {
	store, ctx, bank, _ := indexedBankProofFixture(t)
	recipient := sdk.AccAddress(bytes.Repeat([]byte{1}, 20))
	sender := sdk.AccAddress(bytes.Repeat([]byte{0xfe}, 20))
	require.NoError(t, bank.Balances.Set(ctx, collections.Join(recipient, schedule.Denom), sdkmath.NewInt(5)))
	require.NoError(t, bank.Balances.Set(ctx, collections.Join(sender, schedule.Denom), sdkmath.NewInt(7)))
	require.NoError(t, bank.SendCoins(ctx, sender, recipient, sdk.NewCoins(sdk.NewInt64Coin(schedule.Denom, 7))))
	_, err := bank.Balances.Get(ctx, collections.Join(sender, schedule.Denom))
	require.ErrorIs(t, err, collections.ErrNotFound)
	require.True(t, bank.GetBalance(ctx, sender, schedule.Denom).IsZero())
	require.Equal(t, "12", bank.GetBalance(ctx, recipient, schedule.Denom).Amount.String())
	commit := store.Commit()
	expected := bankProofExpectation(t, sender, "0")
	response := queryBankProofFixture(t, store, expected)
	require.Nil(t, response.Value)
	proof := new(ics23.CommitmentProof)
	require.NoError(t, proof.Unmarshal(response.ProofOps.Ops[0].Data))
	nonexist := proof.GetNonexist()
	require.NotNil(t, nonexist)
	require.NotNil(t, nonexist.Left)
	require.NotNil(t, nonexist.Right)
	require.True(t, bytes.HasPrefix(nonexist.Right.Key, banktypes.DenomAddressPrefix.Bytes()))
	require.Empty(t, nonexist.Right.Value) // Written by SDK ReversePair, not this test.
	bankRoot, err := proof.Calculate()
	require.NoError(t, err) // Calculate uses the nonempty left neighbor only.
	require.NoError(t, nonexist.Left.Verify(ics23.IavlSpec, bankRoot, nonexist.Left.Key, nonexist.Left.Value))
	require.ErrorContains(t, nonexist.Verify(ics23.IavlSpec, bankRoot, expected.Key), "right proof, error calculating root, leaf, leaf op needs value")
	storePath := merkle.KeyPath{}.AppendKey([]byte(banktypes.StoreKey), merkle.KeyEncodingURL).String()
	require.NoError(t, rootmulti.DefaultProofRuntime().VerifyValue(&cmtcrypto.ProofOps{Ops: response.ProofOps.Ops[1:]}, commit.Hash, storePath, bankRoot))
	// Regression for the attempt5 blocker, NOT a repair: ICS23 v0.11.0
	// cannot verify the empty-valued index neighbor. Keep failing closed;
	// neither a balance representation fallback nor the valid outer layer
	// can substitute for verification of the complete absence proof.
	require.ErrorContains(t, verifyStateProof(response, expected, 120, commit.Hash), "proof did not verify absence of key")
}

func TestSchedulerBankProofBindingsRemainMandatory(t *testing.T) {
	for _, amount := range []string{"0", "7"} {
		t.Run(amount, func(t *testing.T) {
			store, ctx, bank, _ := indexedBankProofFixture(t)
			addr := sdk.AccAddress(bytes.Repeat([]byte{2}, 20))
			expected := bankProofExpectation(t, addr, amount)
			for _, b := range []byte{1, 3} {
				require.NoError(t, bank.Balances.Set(ctx, collections.Join(sdk.AccAddress(bytes.Repeat([]byte{b}, 20)), schedule.Denom), sdkmath.NewInt(5)))
			}
			if amount != "0" {
				require.NoError(t, bank.Balances.Set(ctx, collections.Join(addr, schedule.Denom), sdkmath.NewInt(7)))
			}
			commit := store.Commit()
			response := queryBankProofFixture(t, store, expected)
			require.NoError(t, verifyStateProof(response, expected, 120, commit.Hash))
			for name, mutate := range map[string]func(*abci.ResponseQuery, *stateKeyExpectation, *[]byte){
				"wrong root":         func(_ *abci.ResponseQuery, _ *stateKeyExpectation, root *[]byte) { (*root)[0] ^= 1 },
				"short root":         func(_ *abci.ResponseQuery, _ *stateKeyExpectation, root *[]byte) { *root = (*root)[:31] },
				"wrong height":       func(r *abci.ResponseQuery, _ *stateKeyExpectation, _ *[]byte) { r.Height++ },
				"wrong response key": func(r *abci.ResponseQuery, _ *stateKeyExpectation, _ *[]byte) { r.Key = []byte("wrong") },
				"relabelled key":     func(r *abci.ResponseQuery, e *stateKeyExpectation, _ *[]byte) { e.Key = []byte("wrong"); r.Key = e.Key },
				"wrong store":        func(_ *abci.ResponseQuery, e *stateKeyExpectation, _ *[]byte) { e.Store = schedule.StoreKey },
				"wrong value":        func(r *abci.ResponseQuery, _ *stateKeyExpectation, _ *[]byte) { r.Value = []byte("9") },
				"relabelled value": func(r *abci.ResponseQuery, e *stateKeyExpectation, _ *[]byte) {
					r.Value = []byte("9")
					e.Value = r.Value
				},
				"query error":         func(r *abci.ResponseQuery, _ *stateKeyExpectation, _ *[]byte) { r.Code = 1 },
				"missing proof":       func(r *abci.ResponseQuery, _ *stateKeyExpectation, _ *[]byte) { r.ProofOps = nil },
				"empty proof":         func(r *abci.ResponseQuery, _ *stateKeyExpectation, _ *[]byte) { r.ProofOps.Ops = nil },
				"missing inner layer": func(r *abci.ResponseQuery, _ *stateKeyExpectation, _ *[]byte) { r.ProofOps.Ops = r.ProofOps.Ops[1:] },
				"missing outer layer": func(r *abci.ResponseQuery, _ *stateKeyExpectation, _ *[]byte) { r.ProofOps.Ops = r.ProofOps.Ops[:1] },
				"malformed inner":     func(r *abci.ResponseQuery, _ *stateKeyExpectation, _ *[]byte) { r.ProofOps.Ops[0].Data = []byte{0xff} },
				"malformed outer":     func(r *abci.ResponseQuery, _ *stateKeyExpectation, _ *[]byte) { r.ProofOps.Ops[1].Data = []byte{0xff} },
				"wrong inner key": func(r *abci.ResponseQuery, _ *stateKeyExpectation, _ *[]byte) {
					r.ProofOps.Ops[0].Key = []byte("wrong")
				},
				"wrong outer store": func(r *abci.ResponseQuery, _ *stateKeyExpectation, _ *[]byte) {
					r.ProofOps.Ops[1].Key = []byte(schedule.StoreKey)
				},
				"unknown operator": func(r *abci.ResponseQuery, _ *stateKeyExpectation, _ *[]byte) { r.ProofOps.Ops[0].Type = "unknown" },
			} {
				t.Run(name, func(t *testing.T) {
					r, e, root := response, expected, bytes.Clone(commit.Hash)
					r.ProofOps = &cmtcrypto.ProofOps{Ops: append([]cmtcrypto.ProofOp(nil), response.ProofOps.Ops...)}
					mutate(&r, &e, &root)
					require.Error(t, verifyStateProof(r, e, 120, root))
				})
			}
		})
	}
}

func TestSchedulerDueAndReceiptAbsenceStayStrict(t *testing.T) {
	for name, key := range map[string][]byte{
		"due":     schedule.DueKey(120, fixtureExpectation(fixtureChainID, 120).State.Schedules[0].Id),
		"receipt": schedule.ReceiptKey(fixtureExpectation(fixtureChainID, 120).State.Schedules[0].Id, 2),
	} {
		for _, value := range [][]byte{[]byte("0"), {}} {
			t.Run(name+"/value="+string(value), func(t *testing.T) {
				store, _, _, mounts := indexedBankProofFixture(t)
				store.GetKVStore(mounts[schedule.StoreKey]).Set(key, value)
				commit := store.Commit()
				expected := stateKeyExpectation{Store: schedule.StoreKey, Key: key, Absent: true}
				response := queryBankProofFixture(t, store, expected)
				require.Error(t, verifyStateProof(response, expected, 120, commit.Hash))
			})
		}
	}
}
