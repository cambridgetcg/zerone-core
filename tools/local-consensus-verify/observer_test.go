package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"cosmossdk.io/collections"
	"cosmossdk.io/log"
	"cosmossdk.io/store/metrics"
	"cosmossdk.io/store/rootmulti"
	storetypes "cosmossdk.io/store/types"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/crypto/ed25519"
	cmtbytes "github.com/cometbft/cometbft/libs/bytes"
	"github.com/cometbft/cometbft/p2p"
	cmtcrypto "github.com/cometbft/cometbft/proto/tendermint/crypto"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	rpcclient "github.com/cometbft/cometbft/rpc/client"
	coretypes "github.com/cometbft/cometbft/rpc/core/types"
	cmttypes "github.com/cometbft/cometbft/types"
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
)

// Public-key-shaped bytes suffice for identity/exclusion tests. No custody or
// signing key is generated or opened by these fixtures.
func observerFixture(t *testing.T) (*observerExpectation, []byte, []*cmttypes.Validator) {
	t.Helper()
	e := &observerExpectation{Schema: observerSchema, Phase: "replay", ChainID: fixtureChainID,
		RPC: "http://127.0.0.1:40009", NodeID: strings.Repeat("09", 20),
		ConsensusPubKey: bytes.Repeat([]byte{9}, 32), BinarySHA256: strings.Repeat("ab", 32),
		HistoryBeforeJoin: 120, StateHeight: 125, AppHash: strings.Repeat("cd", 32)}
	var txs []any
	var validators []*cmttypes.Validator
	for i := byte(1); i <= 4; i++ {
		pub := ed25519.PubKey(bytes.Repeat([]byte{i}, 32))
		validators = append(validators, cmttypes.NewValidator(pub, 1000))
		txs = append(txs, map[string]any{"body": map[string]any{"messages": []any{map[string]any{
			"@type":  "/cosmos.staking.v1beta1.MsgCreateValidator",
			"pubkey": map[string]any{"@type": "/cosmos.crypto.ed25519.PubKey", "key": []byte(pub)},
		}}}})
	}
	bz, err := json.Marshal(map[string]any{"chain_id": fixtureChainID, "initial_height": 10,
		"consensus": map[string]any{"validators": []any{}},
		"app_state": map[string]any{"genutil": map[string]any{"gen_txs": txs},
			"bank": map[string]any{"params": banktypes.DefaultParams()}}})
	require.NoError(t, err)
	hash := sha256.Sum256(bz)
	e.GenesisSHA256 = hex.EncodeToString(hash[:])
	return e, bz, validators
}

func TestObserverGenesisBindingsAndExclusion(t *testing.T) {
	e, genesis, validators := observerFixture(t)
	members, initial, err := verifyObserverGenesis(genesis, e)
	require.NoError(t, err)
	require.Len(t, members, 4)
	require.Equal(t, int64(10), initial)
	for name, mutate := range map[string]func(*observerExpectation){
		"chain":               func(e *observerExpectation) { e.ChainID = "wrong-chain" },
		"hash":                func(e *observerExpectation) { e.GenesisSHA256 = strings.Repeat("00", 32) },
		"excluded membership": func(e *observerExpectation) { e.ConsensusPubKey = validators[0].PubKey.Bytes() },
		"no past history":     func(e *observerExpectation) { e.HistoryBeforeJoin = 10 },
	} {
		t.Run(name, func(t *testing.T) {
			bad := *e
			mutate(&bad)
			_, _, err := verifyObserverGenesis(genesis, &bad)
			require.Error(t, err)
		})
	}
	badGenesis := bytes.Replace(genesis, []byte(`"gen_txs"`), []byte(`"not_gen_txs"`), 1)
	hash := sha256.Sum256(badGenesis)
	e.GenesisSHA256 = hex.EncodeToString(hash[:])
	_, _, err = verifyObserverGenesis(badGenesis, e)
	require.ErrorContains(t, err, "exactly four")
}

func TestObserverSetMustBeExactHeightPinnedFourAndExcludeObserver(t *testing.T) {
	e, genesis, validators := observerFixture(t)
	members, _, err := verifyObserverGenesis(genesis, e)
	require.NoError(t, err)
	set := &coretypes.ResultValidators{BlockHeight: 126, Total: 4, Count: 4, Validators: validators}
	require.NoError(t, verifyObserverSet(e, members, set, 126))
	for name, mutate := range map[string]func(*coretypes.ResultValidators){
		"height":    func(s *coretypes.ResultValidators) { s.BlockHeight-- },
		"count":     func(s *coretypes.ResultValidators) { s.Total = 5 },
		"missing":   func(s *coretypes.ResultValidators) { s.Validators = s.Validators[:3] },
		"duplicate": func(s *coretypes.ResultValidators) { s.Validators[1] = s.Validators[0] },
		"observer included": func(s *coretypes.ResultValidators) {
			s.Validators[0] = cmttypes.NewValidator(ed25519.PubKey(e.ConsensusPubKey), 1000)
		},
		"wrong member": func(s *coretypes.ResultValidators) {
			s.Validators[0] = cmttypes.NewValidator(ed25519.PubKey(bytes.Repeat([]byte{33}, 32)), 1000)
		},
	} {
		t.Run(name, func(t *testing.T) {
			bad := *set
			bad.Validators = append([]*cmttypes.Validator(nil), validators...)
			mutate(&bad)
			require.Error(t, verifyObserverSet(e, members, &bad, 126))
		})
	}
}

func TestObserverStatusIdentityAndRestartMismatch(t *testing.T) {
	e, genesis, validators := observerFixture(t)
	members, _, err := verifyObserverGenesis(genesis, e)
	require.NoError(t, err)
	var rpcs []string
	var statuses []*coretypes.ResultStatus
	for i, v := range append(validators, cmttypes.NewValidator(ed25519.PubKey(e.ConsensusPubKey), 0)) {
		id := fmt.Sprintf("%040x", i+1)
		endpoint := fmt.Sprintf("http://127.0.0.1:%d", 40001+i*2)
		if i == 4 {
			id = e.NodeID
			endpoint = e.RPC
		}
		rpcs = append(rpcs, endpoint)
		statuses = append(statuses, &coretypes.ResultStatus{
			NodeInfo:      p2p.DefaultNodeInfo{DefaultNodeID: p2p.ID(id), Network: e.ChainID},
			ValidatorInfo: coretypes.ValidatorInfo{Address: v.Address, PubKey: v.PubKey, VotingPower: v.VotingPower},
			SyncInfo:      coretypes.SyncInfo{LatestBlockHeight: e.StateHeight + 3},
		})
	}
	index, err := verifyObserverStatuses(e, rpcs, statuses, members)
	require.NoError(t, err)
	require.Equal(t, 4, index)
	for name, mutate := range map[string]func(*coretypes.ResultStatus){
		"chain":                func(s *coretypes.ResultStatus) { s.NodeInfo.Network = "other" },
		"changed P2P identity": func(s *coretypes.ResultStatus) { s.NodeInfo.DefaultNodeID = p2p.ID(strings.Repeat("aa", 20)) },
		"changed consensus identity": func(s *coretypes.ResultStatus) {
			s.ValidatorInfo.PubKey = validators[0].PubKey
			s.ValidatorInfo.Address = validators[0].Address
		},
		"nonzero power":     func(s *coretypes.ResultStatus) { s.ValidatorInfo.VotingPower = 1 },
		"catching up":       func(s *coretypes.ResultStatus) { s.SyncInfo.CatchingUp = true },
		"behind checkpoint": func(s *coretypes.ResultStatus) { s.SyncInfo.LatestBlockHeight = e.StateHeight },
	} {
		t.Run(name, func(t *testing.T) {
			bad := *statuses[4]
			mutate(&bad)
			copyStatuses := append([]*coretypes.ResultStatus(nil), statuses...)
			copyStatuses[4] = &bad
			_, err := verifyObserverStatuses(e, rpcs, copyStatuses, members)
			require.Error(t, err)
		})
	}
	_, err = verifyObserverStatuses(e, rpcs[:4], statuses[:4], members)
	require.Error(t, err)
}

func TestObserverZeroSigningState(t *testing.T) {
	require.NoError(t, verifyObserverZeroState([]byte(`{"height":"0","round":0,"step":0}`)))
	for _, bad := range []string{
		`{}`, `{"height":"0"}`, `{"height":"1","round":0,"step":0}`, `{"height":"0","round":1,"step":0}`, `{"height":"0","round":0,"step":1}`,
		`{"height":"0","round":0,"step":0,"signature":"AQ=="}`, `{"height":"0","round":0,"step":0,"signbytes":"AQ=="}`,
		`{"height":"0","private_key":"sentinel-private-input"}`, `{"height":"0"} {}`,
	} {
		err := verifyObserverZeroState([]byte(bad))
		require.Error(t, err)
		require.NotContains(t, err.Error(), "sentinel-private-input")
	}
}

const observerTestConfig = `priv_validator_laddr = ""
[rpc]
unsafe = false
pprof_laddr = ""
[statesync]
enable = false
[instrumentation]
prometheus = false
`

func TestObserverHomeRejectsKeysDatabaseAndUnsafeConfig(t *testing.T) {
	e, genesis, _ := observerFixture(t)
	home := t.TempDir()
	for _, dir := range []string{"config", "data"} {
		require.NoError(t, os.Mkdir(filepath.Join(home, dir), 0700))
	}
	for name, bz := range map[string][]byte{"config/genesis.json": genesis, "config/config.toml": []byte(observerTestConfig), "data/priv_validator_state.json": []byte(`{"height":"0","round":0,"step":0}`)} {
		require.NoError(t, os.WriteFile(filepath.Join(home, name), bz, 0600))
	}
	// Deliberately no private identity files: these helpers must not open them.
	_, _, err := verifyObserverHome(home, e)
	require.NoError(t, err)
	require.NoError(t, verifyObserverFreshHome(home))
	require.NoError(t, os.Mkdir(filepath.Join(home, "keyring-test"), 0700))
	_, _, err = verifyObserverHome(home, e)
	require.ErrorContains(t, err, "keyring")
	require.NoError(t, os.Remove(filepath.Join(home, "keyring-test")))
	require.NoError(t, os.Mkdir(filepath.Join(home, "data", "application.db"), 0700))
	require.ErrorContains(t, verifyObserverFreshHome(home), "database copying")
	for _, mutation := range []string{
		strings.Replace(observerTestConfig, "enable = false", "enable = true", 1),
		strings.Replace(observerTestConfig, `priv_validator_laddr = ""`, `priv_validator_laddr = "tcp://127.0.0.1:1234"`, 1),
		strings.Replace(observerTestConfig, "unsafe = false", "unsafe = true", 1),
		strings.Replace(observerTestConfig, `pprof_laddr = ""`, `pprof_laddr = ":6060"`, 1),
		strings.Replace(observerTestConfig, "prometheus = false", "prometheus = true", 1),
		"[statesync]\n", // Missing booleans cannot masquerade as disabled.
	} {
		require.NoError(t, os.WriteFile(filepath.Join(home, "config", "config.toml"), []byte(mutation), 0600))
		_, _, err = verifyObserverHome(home, e)
		require.Error(t, err)
	}
}

func TestObserverInputBoundsAndLinks(t *testing.T) {
	e, _, _ := observerFixture(t)
	path := filepath.Join(t.TempDir(), "expect.json")
	require.NoError(t, writeFixtureJSON(path, e))
	_, err := loadObserverExpectation(path, e.ChainID)
	require.NoError(t, err)
	link := filepath.Join(t.TempDir(), "link")
	require.NoError(t, os.Symlink(path, link))
	_, err = loadObserverExpectation(link, e.ChainID)
	require.Error(t, err)
	require.NoError(t, os.WriteFile(path, bytes.Repeat([]byte(" "), 16<<10+1), 0600))
	_, err = loadObserverExpectation(path, e.ChainID)
	require.Error(t, err)
	for _, mutate := range []func(*observerExpectation){
		func(e *observerExpectation) { e.StateHeight = 119 },
		func(e *observerExpectation) { e.StateHeight = int64(^uint64(0) >> 1) },
		func(e *observerExpectation) { e.AppHash = "00" },
		func(e *observerExpectation) { e.BinarySHA256 = "" },
	} {
		bad := *e
		mutate(&bad)
		require.Error(t, validateObserverExpectation(&bad, e.ChainID))
	}
}

func TestObserverCheckpointUsesNextHeaderNotSameHeight(t *testing.T) {
	e, _, _ := observerFixture(t)
	root, err := hex.DecodeString(e.AppHash)
	require.NoError(t, err)
	block := &cmttypes.Block{Header: cmttypes.Header{ChainID: e.ChainID, Height: e.StateHeight + 1, AppHash: root}}
	require.NoError(t, verifyObserverAnchor(e, block))
	block.Height--
	require.ErrorContains(t, verifyObserverAnchor(e, block), "H+1")
	block.Height++
	block.AppHash = bytes.Repeat([]byte{1}, 32)
	require.ErrorContains(t, verifyObserverAnchor(e, block), "pre-state AppHash")
	block.AppHash = root
	block.ChainID = "other"
	require.Error(t, verifyObserverAnchor(e, block))
}

type observerProofClient struct {
	t           *testing.T
	store       *rootmulti.Store
	height      int64
	hash        []byte
	queryHeight int64
	mutate      func(*coretypes.ResultABCIQuery) *coretypes.ResultABCIQuery
}

func (c observerProofClient) ABCIInfo(context.Context) (*coretypes.ResultABCIInfo, error) {
	return &coretypes.ResultABCIInfo{Response: abci.ResponseInfo{LastBlockHeight: c.height, LastBlockAppHash: c.hash}}, nil
}
func (c observerProofClient) ABCIQueryWithOptions(_ context.Context, path string, key cmtbytes.HexBytes, options rpcclient.ABCIQueryOptions) (*coretypes.ResultABCIQuery, error) {
	require.Equal(c.t, "/store/bank/key", path)
	require.Equal(c.t, []byte{0x05}, []byte(key))
	require.Equal(c.t, c.queryHeight, options.Height)
	require.True(c.t, options.Prove)
	res, err := c.store.Query(&storetypes.RequestQuery{Path: "/bank/key", Data: key, Height: options.Height, Prove: true})
	if err != nil {
		return nil, err
	}
	result := &coretypes.ResultABCIQuery{Response: abci.ResponseQuery{Code: res.Code, Height: res.Height, Key: res.Key, Value: res.Value, ProofOps: res.ProofOps}}
	if c.mutate != nil {
		result = c.mutate(result)
	}
	return result, nil
}

// Real SDK InitGenesis/SendCoins create the bank records and empty reverse index;
// only account lookups are mocked. There is no manually simplified bank tree,
// synthetic boundary address, daemon, custody key or retained live database.
func observerSDKBankFixture(t *testing.T, height int64) (*rootmulti.Store, dbm.DB, bankkeeper.BaseKeeper, sdk.Context, *storetypes.KVStoreKey) {
	t.Helper()
	db := dbm.NewMemDB()
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	store, key := observerBankStore(db)
	require.NoError(t, store.LoadVersion(0))
	require.NoError(t, store.SetInitialVersion(height))
	ctx := sdk.NewContext(store, cmtproto.Header{Height: height}, false, log.NewNopLogger())
	ak := banktestutil.NewMockAccountKeeper(gomock.NewController(t))
	ak.EXPECT().AddressCodec().Return(address.NewBech32Codec("zrn")).AnyTimes()
	ak.EXPECT().GetAccount(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	ak.EXPECT().HasAccount(gomock.Any(), gomock.Any()).Return(true).AnyTimes()
	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	recipient := sdk.AccAddress(bytes.Repeat([]byte{1}, 20))
	sender := sdk.AccAddress(bytes.Repeat([]byte{0xfe}, 20))
	authority, err := ak.AddressCodec().BytesToString(recipient)
	require.NoError(t, err)
	bank := bankkeeper.NewBaseKeeper(cdc, runtime.NewKVStoreService(key), ak, nil, authority, log.NewNopLogger())
	genesis := banktypes.DefaultGenesisState()
	genesis.Balances = []banktypes.Balance{
		{Address: recipient.String(), Coins: sdk.NewCoins(sdk.NewInt64Coin("uzrn", 5))},
		{Address: sender.String(), Coins: sdk.NewCoins(sdk.NewInt64Coin("uzrn", 7))},
	}
	bank.InitGenesis(ctx, genesis)
	expected, err := codec.CollValue[banktypes.Params](cdc).Encode(genesis.Params)
	require.NoError(t, err)
	require.Equal(t, []byte{0x10, 0x01}, expected)
	require.Equal(t, expected, store.GetKVStore(key).Get(banktypes.ParamsKey.Bytes()))
	require.NoError(t, bank.SendCoins(ctx, sender, recipient, sdk.NewCoins(sdk.NewInt64Coin("uzrn", 7))))
	_, err = bank.Balances.Get(ctx, collections.Join(sender, "uzrn"))
	require.ErrorIs(t, err, collections.ErrNotFound)
	iterator := storetypes.KVStorePrefixIterator(store.GetKVStore(key), banktypes.DenomAddressPrefix.Bytes())
	defer iterator.Close()
	require.True(t, iterator.Valid(), "SDK must actually write a bank reverse-index record")
	require.Empty(t, iterator.Value(), "exercise the real empty-valued 0x03 bank index")
	return store, db, bank, ctx, key
}

func observerBankStore(db dbm.DB) (*rootmulti.Store, *storetypes.KVStoreKey) {
	store := rootmulti.NewStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	key := storetypes.NewKVStoreKey(banktypes.StoreKey)
	store.MountStoreWithDB(key, storetypes.StoreTypeIAVL, nil)
	store.MountStoreWithDB(storetypes.NewKVStoreKey("other"), storetypes.StoreTypeIAVL, nil)
	return store, key
}

func TestObserverRetainedCheckpointProofAndRestartRegression(t *testing.T) {
	e, _, _ := observerFixture(t)
	store, db, bank, ctx, _ := observerSDKBankFixture(t, e.StateHeight)
	checkpoint := store.Commit()
	e.AppHash = hex.EncodeToString(checkpoint.Hash)
	client := observerProofClient{t: t, store: store, height: checkpoint.Version, hash: checkpoint.Hash, queryHeight: e.StateHeight}
	r, err := verifyObserverCheckpoint(client, e, 10)
	require.NoError(t, err)
	require.False(t, r.CheckpointProof.Absent)
	require.Equal(t, "bank", r.CheckpointProof.Store)
	require.Equal(t, "05", r.CheckpointProof.KeyHex)
	require.Equal(t, []byte{0x10, 0x01}, r.CheckpointProof.Value)
	require.Equal(t, "cosmos.bank.v1beta1.Params", r.CheckpointRecord)
	require.Contains(t, r.Scope, "not completeness or accounting")
	proof := new(ics23.CommitmentProof)
	require.NoError(t, proof.Unmarshal(r.CheckpointProof.ProofOps.Ops[0].Data))
	require.NotNil(t, proof.GetExist())
	require.Nil(t, proof.GetNonexist(), "observer must never request absence")

	// Change the latest parameter record through SDK SetParams. It is empty at
	// H+1; after reopening the DB we must still prove H's nonempty exact record.
	require.NoError(t, bank.SetParams(ctx, banktypes.NewParams(false)))
	latest := store.Commit()
	reopened, _ := observerBankStore(db)
	require.NoError(t, reopened.LoadLatestVersion())
	client.store, client.height, client.hash = reopened, latest.Version, latest.Hash
	e.Phase = "restart"
	r, err = verifyObserverCheckpoint(client, e, 10)
	require.NoError(t, err)
	require.Equal(t, e.StateHeight+1, r.AppliedHeight)
	require.Equal(t, []byte{0x10, 0x01}, r.CheckpointProof.Value)
	anchor := &cmttypes.Block{Header: cmttypes.Header{ChainID: e.ChainID, Height: e.StateHeight + 1, AppHash: checkpoint.Hash}}
	require.NoError(t, verifyObserverAnchor(e, anchor))
	anchor.AppHash = latest.Hash
	require.ErrorContains(t, verifyObserverAnchor(e, anchor), "pre-state AppHash")
	client.height = e.StateHeight
	_, err = verifyObserverCheckpoint(client, e, 10)
	require.ErrorContains(t, err, "applied height")
	encoded, err := json.Marshal(r)
	require.NoError(t, err)
	for _, forbidden := range []string{"priv_validator_key", "signbytes", "signature\"", "home\""} {
		require.NotContains(t, string(encoded), forbidden)
	}
}

func TestObserverMembershipProofRejectsWrongBindingsAndMalformedResponses(t *testing.T) {
	e, _, _ := observerFixture(t)
	store, _, _, _, _ := observerSDKBankFixture(t, e.StateHeight)
	checkpoint := store.Commit()
	e.AppHash = hex.EncodeToString(checkpoint.Hash)
	client := observerProofClient{t: t, store: store, height: checkpoint.Version, hash: checkpoint.Hash, queryHeight: e.StateHeight}
	for name, mutate := range map[string]func(*abci.ResponseQuery){
		"height":             func(r *abci.ResponseQuery) { r.Height++ },
		"query error":        func(r *abci.ResponseQuery) { r.Code = 1 },
		"wrong key":          func(r *abci.ResponseQuery) { r.Key = []byte{0x06} },
		"empty value":        func(r *abci.ResponseQuery) { r.Value = nil },
		"wrong value":        func(r *abci.ResponseQuery) { r.Value = []byte{0x10, 0x00} },
		"malformed value":    func(r *abci.ResponseQuery) { r.Value = []byte{0xff} },
		"noncanonical value": func(r *abci.ResponseQuery) { r.Value = []byte{0x10, 0x01, 0x10, 0x01} },
		"missing proof":      func(r *abci.ResponseQuery) { r.ProofOps = nil },
		"empty proof":        func(r *abci.ResponseQuery) { r.ProofOps.Ops = nil },
		"inner only":         func(r *abci.ResponseQuery) { r.ProofOps.Ops = r.ProofOps.Ops[:1] },
		"outer only":         func(r *abci.ResponseQuery) { r.ProofOps.Ops = r.ProofOps.Ops[1:] },
		"malformed inner":    func(r *abci.ResponseQuery) { r.ProofOps.Ops[0].Data = []byte{0xff} },
		"malformed outer":    func(r *abci.ResponseQuery) { r.ProofOps.Ops[1].Data = []byte{0xff} },
		"wrong inner key":    func(r *abci.ResponseQuery) { r.ProofOps.Ops[0].Key = []byte{0x06} },
		"wrong store":        func(r *abci.ResponseQuery) { r.ProofOps.Ops[1].Key = []byte("other") },
		"unknown operator":   func(r *abci.ResponseQuery) { r.ProofOps.Ops[0].Type = "unknown" },
		"forged proof value": func(r *abci.ResponseQuery) {
			proof := new(ics23.CommitmentProof)
			require.NoError(t, proof.Unmarshal(r.ProofOps.Ops[0].Data))
			proof.GetExist().Value = []byte("forged")
			encoded, err := proof.Marshal()
			require.NoError(t, err)
			r.ProofOps.Ops[0].Data = encoded
		},
	} {
		t.Run(name, func(t *testing.T) {
			bad := client
			bad.mutate = func(r *coretypes.ResultABCIQuery) *coretypes.ResultABCIQuery {
				r.Response.ProofOps = &cmtcrypto.ProofOps{Ops: append([]cmtcrypto.ProofOp(nil), r.Response.ProofOps.Ops...)}
				mutate(&r.Response)
				return r
			}
			_, err := verifyObserverCheckpoint(bad, e, 10)
			require.Error(t, err)
		})
	}
	bad := *e
	bad.AppHash = strings.Repeat("aa", 32)
	_, err := verifyObserverCheckpoint(client, &bad, 10)
	require.Error(t, err)
	bad.AppHash = "aa"
	_, err = verifyObserverCheckpoint(client, &bad, 10)
	require.ErrorContains(t, err, "root length")
	client.mutate = func(*coretypes.ResultABCIQuery) *coretypes.ResultABCIQuery { return nil }
	_, err = verifyObserverCheckpoint(client, e, 10)
	require.ErrorContains(t, err, "missing or empty")
}

func TestObserverAnchorRequiresNonemptyInitializedParamsWithoutFallback(t *testing.T) {
	for _, name := range []string{"missing genesis params", "disabled genesis sends"} {
		t.Run(name, func(t *testing.T) {
			e, genesis, _ := observerFixture(t)
			if name == "missing genesis params" {
				genesis = bytes.Replace(genesis, []byte(`"params"`), []byte(`"missing_params"`), 1)
			} else {
				genesis = bytes.Replace(genesis, []byte(`"default_send_enabled":true`), []byte(`"default_send_enabled":false`), 1)
			}
			hash := sha256.Sum256(genesis)
			e.GenesisSHA256 = hex.EncodeToString(hash[:])
			_, _, err := verifyObserverGenesis(genesis, e)
			require.ErrorContains(t, err, "nonempty bank/05 membership anchor")
		})
	}
	for _, name := range []string{"absent retained params", "empty retained params"} {
		t.Run(name, func(t *testing.T) {
			e, _, _ := observerFixture(t)
			store, _, bank, ctx, key := observerSDKBankFixture(t, e.StateHeight)
			if name == "absent retained params" {
				store.GetKVStore(key).Delete(banktypes.ParamsKey.Bytes())
			} else {
				require.NoError(t, bank.SetParams(ctx, banktypes.NewParams(false)))
			}
			checkpoint := store.Commit()
			e.AppHash = hex.EncodeToString(checkpoint.Hash)
			client := observerProofClient{t: t, store: store, height: checkpoint.Version, hash: checkpoint.Hash, queryHeight: e.StateHeight}
			_, err := verifyObserverCheckpoint(client, e, 10)
			require.ErrorContains(t, err, "missing or empty; no fallback")
		})
	}
}
