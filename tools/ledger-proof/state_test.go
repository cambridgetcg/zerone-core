package main

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"cosmossdk.io/collections"
	"cosmossdk.io/log"
	sdkmath "cosmossdk.io/math"
	"cosmossdk.io/store/metrics"
	"cosmossdk.io/store/rootmulti"
	storetypes "cosmossdk.io/store/types"
	abci "github.com/cometbft/cometbft/abci/types"
	rpctypes "github.com/cometbft/cometbft/rpc/core/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
)

// The SDK's own collections codecs build the keys the stores are written
// with; the tool derives its keys by hand. A divergence fails these tests.
func sdkBalanceKey(t *testing.T, addr sdk.AccAddress, denom string) []byte {
	t.Helper()
	k, err := collections.EncodeKeyWithPrefix([]byte(banktypes.BalancesPrefix), collections.PairKeyCodec(sdk.AccAddressKey, collections.StringKey), collections.Join(addr, denom))
	if err != nil {
		t.Fatal(err)
	}
	return k
}

func sdkAccountKey(t *testing.T, addr sdk.AccAddress) []byte {
	t.Helper()
	k, err := collections.EncodeKeyWithPrefix([]byte(authtypes.AddressStoreKeyPrefix), sdk.AccAddressKey, addr)
	if err != nil {
		t.Fatal(err)
	}
	return k
}

type stateFixture struct {
	chain                 testChain
	rs                    *rootmulti.Store
	hashes                map[int64][]byte
	holder, module, empty sdk.AccAddress
	h                     int64
}

// newStateFixture commits versions 1 to 4 of an IAVL multistore; the state
// under test is version 3, whose app hash belongs in the header at height 4.
func newStateFixture(t *testing.T, chain testChain) *stateFixture {
	t.Helper()
	s := &stateFixture{chain: chain, rs: rootmulti.NewStore(dbm.NewMemDB(), log.NewNopLogger(), metrics.NewNoOpMetrics()), hashes: map[int64][]byte{}, h: 3}
	keys := map[string]*storetypes.KVStoreKey{}
	for _, name := range []string{authtypes.StoreKey, banktypes.StoreKey, "staking"} {
		keys[name] = storetypes.NewKVStoreKey(name)
		s.rs.MountStoreWithDB(keys[name], storetypes.StoreTypeIAVL, nil)
	}
	if err := s.rs.LoadLatestVersion(); err != nil {
		t.Fatal(err)
	}
	reg := codectypes.NewInterfaceRegistry()
	authtypes.RegisterInterfaces(reg)
	cryptocodec.RegisterInterfaces(reg)
	accounts := codec.CollInterfaceValue[sdk.AccountI](codec.NewProtoCodec(reg))
	key := secp256k1.GenPrivKey()
	s.holder = sdk.AccAddress(key.PubKey().Address())
	s.module = authtypes.NewModuleAddress("test-module")
	s.empty = sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address())
	put := func(name string, k []byte, v []byte) { s.rs.GetKVStore(keys[name]).Set(k, v) }
	for v := int64(1); v <= 4; v++ {
		amount, err := banktypes.BalanceValueCodec.Encode(sdkmath.NewInt(1000 * v))
		if err != nil {
			t.Fatal(err)
		}
		put(banktypes.StoreKey, sdkBalanceKey(t, s.holder, nativeDenom), amount)
		legacy := sdk.NewInt64Coin("ulegacy", 5)
		put(banktypes.StoreKey, sdkBalanceKey(t, s.holder, "ulegacy"), wire(t, &legacy))
		holder := &authtypes.BaseAccount{Address: address(t, s.holder), AccountNumber: 17, Sequence: uint64(v)}
		if err := holder.SetPubKey(key.PubKey()); err != nil {
			t.Fatal(err)
		}
		for addr, acc := range map[string]sdk.AccountI{string(s.holder): holder, string(s.module): authtypes.NewEmptyModuleAccount("test-module")} {
			b, err := accounts.Encode(acc)
			if err != nil {
				t.Fatal(err)
			}
			put(authtypes.StoreKey, sdkAccountKey(t, sdk.AccAddress(addr)), b)
		}
		put("staking", []byte{1, byte(v)}, []byte("another store"))
		id := s.rs.Commit()
		s.hashes[id.Version] = id.Hash
	}
	return s
}

func (s *stateFixture) query(t *testing.T, store string, key []byte, height int64) abci.ResponseQuery {
	t.Helper()
	res, err := s.rs.Query(&storetypes.RequestQuery{Path: "/" + store + "/key", Data: key, Height: height, Prove: true})
	if err != nil {
		t.Fatal(err)
	}
	res.Height = height // as baseapp sets it
	return abci.ResponseQuery(*res)
}

func (s *stateFixture) evidence(t *testing.T, res abci.ResponseQuery, headerHeight int64, appHash []byte) stateEvidence {
	t.Helper()
	h := s.chain.header(t, headerHeight)
	h.AppHash = appHash
	return stateEvidence{Query: envelope(t, rpctypes.ResultABCIQuery{Response: res}), NextCommit: envelope(t, s.chain.commit(t, h)), NextValidators: s.chain.validators(t, headerHeight)}
}

// valid returns honest evidence for w at the fixture's height.
func (s *stateFixture) valid(t *testing.T, w stateWant) stateEvidence {
	t.Helper()
	store, key, err := stateKey(w)
	if err != nil {
		t.Fatal(err)
	}
	return s.evidence(t, s.query(t, store, key, s.h), s.h+1, s.hashes[s.h])
}

func TestStateKeysMatchSDK(t *testing.T) {
	for _, n := range []int{20, 32} {
		addr := sdk.AccAddress(bytes.Repeat([]byte{byte(n)}, n))
		store, key, err := stateKey(stateWant{Kind: "balance", Address: address(t, addr), Denom: "ibc/ABC"})
		if err != nil || store != banktypes.StoreKey || !bytes.Equal(key, sdkBalanceKey(t, addr, "ibc/ABC")) {
			t.Fatalf("balance key differs from the SDK's: %s %X", store, key)
		}
		store, key, err = stateKey(stateWant{Kind: "account", Address: address(t, addr)})
		if err != nil || store != authtypes.StoreKey || !bytes.Equal(key, sdkAccountKey(t, addr)) {
			t.Fatalf("account key differs from the SDK's: %s %X", store, key)
		}
	}
}

func TestStateProven(t *testing.T) {
	chain := newTestChain("test-a", 100)
	s := newStateFixture(t, chain)
	holder, empty, module := address(t, s.holder), address(t, s.empty), address(t, s.module)
	check := func(t *testing.T, w stateWant) (stateReport, bool) {
		t.Helper()
		w.Height = s.h
		r, ok := verifyState(chain.pol, s.valid(t, w), w)
		if r.Claims.AppHash.Status != statusProven || r.Claims.StoredValue.Status != statusProven || r.AttestedBy.Label != chain.pol.Label || r.Header["height"] != "4" {
			t.Fatalf("store proof not proven: %+v", r.Claims)
		}
		return r, ok
	}
	t.Run("balance present", func(t *testing.T) {
		r, ok := check(t, stateWant{Kind: "balance", Address: holder, Denom: nativeDenom})
		if !ok || r.Claims.Balance.Status != statusProven || r.Balance["amount"] != "3000" || r.Balance["stored"] != true || r.Store != "bank" {
			t.Fatalf("balance: ok=%v %+v %+v", ok, r.Claims, r.Balance)
		}
	})
	for name, w := range map[string]stateWant{"absent denom": {Kind: "balance", Address: holder, Denom: "uabsent"}, "absent address": {Kind: "balance", Address: empty, Denom: nativeDenom}} {
		t.Run(name, func(t *testing.T) {
			r, ok := check(t, w)
			if !ok || r.Claims.Balance.Status != statusProven || r.Balance["amount"] != "0" || r.Balance["stored"] != false {
				t.Fatalf("proven absence not reported as zero: %+v %+v", r.Claims, r.Balance)
			}
		})
	}
	t.Run("balance in a legacy encoding", func(t *testing.T) {
		r, ok := check(t, stateWant{Kind: "balance", Address: holder, Denom: "ulegacy"})
		if ok || r.Claims.Balance.Status != statusNotProven || r.Balance != nil {
			t.Fatalf("noncanonical balance reported: %+v %+v", r.Claims, r.Balance)
		}
	})
	t.Run("base account", func(t *testing.T) {
		r, ok := check(t, stateWant{Kind: "account", Address: holder})
		if !ok || r.Claims.Sequence.Status != statusProven || r.Account["sequence"] != "3" || r.Account["account_number"] != "17" || r.Account["address"] != holder || r.Store != "acc" {
			t.Fatalf("account: ok=%v %+v %+v", ok, r.Claims, r.Account)
		}
	})
	t.Run("absent account", func(t *testing.T) {
		r, ok := check(t, stateWant{Kind: "account", Address: empty})
		if !ok || r.Claims.Account.Status != statusProven || r.Account["exists"] != false || r.Claims.Sequence.Status != statusNotApplicable || r.Claims.AccountNumber.Status != statusNotApplicable {
			t.Fatalf("absent account: ok=%v %+v %+v", ok, r.Claims, r.Account)
		}
	})
	t.Run("module account", func(t *testing.T) {
		r, ok := check(t, stateWant{Kind: "account", Address: module})
		if ok || r.Claims.Account.Status != statusProven || r.Account["type_url"] != "/cosmos.auth.v1beta1.ModuleAccount" || r.Claims.Sequence.Status != statusNotProven || r.Claims.AccountNumber.Status != statusNotProven || r.Account["sequence"] != nil {
			t.Fatalf("non-base account: ok=%v %+v %+v", ok, r.Claims, r.Account)
		}
	})
}

func TestStateRefusals(t *testing.T) {
	chain := newTestChain("test-a", 100)
	s := newStateFixture(t, chain)
	w := stateWant{Height: s.h, Kind: "balance", Address: address(t, s.holder), Denom: nativeDenom}
	key := sdkBalanceKey(t, s.holder, nativeDenom)
	honest := func() abci.ResponseQuery { return s.query(t, banktypes.StoreKey, key, s.h) }
	cases := []struct {
		name, reason string
		e            func() stateEvidence
	}{
		{"key for another address", "another key", func() stateEvidence {
			return s.evidence(t, s.query(t, banktypes.StoreKey, sdkBalanceKey(t, s.empty, nativeDenom), s.h), s.h+1, s.hashes[s.h])
		}},
		{"key for another denom", "another key", func() stateEvidence {
			return s.evidence(t, s.query(t, banktypes.StoreKey, sdkBalanceKey(t, s.holder, "uabsent"), s.h), s.h+1, s.hashes[s.h])
		}},
		{"absence proven in another store", "multistore proof for store \"bank\"", func() stateEvidence {
			return s.evidence(t, s.query(t, authtypes.StoreKey, key, s.h), s.h+1, s.hashes[s.h])
		}},
		{"altered value", "store proof", func() stateEvidence {
			res := honest()
			res.Value = []byte("999999")
			return s.evidence(t, res, s.h+1, s.hashes[s.h])
		}},
		{"existence proof presented as absence", "did not verify absence", func() stateEvidence {
			res := honest()
			res.Value = nil
			return s.evidence(t, res, s.h+1, s.hashes[s.h])
		}},
		{"altered proof", "store proof", func() stateEvidence {
			res := honest()
			res.ProofOps.Ops[0].Data = append([]byte{}, res.ProofOps.Ops[0].Data...)
			res.ProofOps.Ops[0].Data[len(res.ProofOps.Ops[0].Data)-1] ^= 1
			return s.evidence(t, res, s.h+1, s.hashes[s.h])
		}},
		{"third proof operation", "exactly two proof operations", func() stateEvidence {
			res := honest()
			res.ProofOps.Ops = append(res.ProofOps.Ops, res.ProofOps.Ops[1])
			return s.evidence(t, res, s.h+1, s.hashes[s.h])
		}},
		{"commit signature at H+1 with a timestamp before year 1", "timestamp is outside years", func() stateEvidence {
			e := s.evidence(t, honest(), s.h+1, s.hashes[s.h])
			h := chain.header(t, s.h+1)
			h.AppHash = s.hashes[s.h]
			c := chain.commit(t, h)
			c.Commit.Signatures[0].Timestamp = time.Date(0, 1, 1, 0, 0, 0, 0, time.UTC)
			e.NextCommit = envelope(t, c)
			return e
		}},
		{"commit block ID at H+1 with a malformed part-set hash", "commit block ID", func() stateEvidence {
			e := s.evidence(t, honest(), s.h+1, s.hashes[s.h])
			h := chain.header(t, s.h+1)
			h.AppHash = s.hashes[s.h]
			c := chain.commit(t, h)
			c.Commit.BlockID.PartSetHeader.Hash = []byte{1, 2, 3, 4, 5}
			e.NextCommit = envelope(t, c)
			return e
		}},
		{"multistore proof dropped", "exactly two proof operations", func() stateEvidence {
			res := honest()
			res.ProofOps.Ops = res.ProofOps.Ops[:1]
			return s.evidence(t, res, s.h+1, s.hashes[s.h])
		}},
		{"proof operations swapped", "first proof operation", func() stateEvidence {
			res := honest()
			res.ProofOps.Ops[0], res.ProofOps.Ops[1] = res.ProofOps.Ops[1], res.ProofOps.Ops[0]
			return s.evidence(t, res, s.h+1, s.hashes[s.h])
		}},
		{"query without proof", "exactly two proof operations", func() stateEvidence {
			res := honest()
			res.ProofOps = nil
			return s.evidence(t, res, s.h+1, s.hashes[s.h])
		}},
		{"query response with an error code", "code 1", func() stateEvidence {
			res := honest()
			res.Code = 1
			return s.evidence(t, res, s.h+1, s.hashes[s.h])
		}},
		{"wrong app hash", "calculated root hash is invalid", func() stateEvidence { return s.evidence(t, honest(), s.h+1, s.hashes[s.h-1]) }},
		{"header at H instead of H+1", "signed header is for height 3, not 4", func() stateEvidence { return s.evidence(t, honest(), s.h, s.hashes[s.h-1]) }},
		{"query from H-1 relabelled as H", "calculated root hash is invalid", func() stateEvidence {
			res := s.query(t, banktypes.StoreKey, key, s.h-1)
			res.Height = s.h
			return s.evidence(t, res, s.h+1, s.hashes[s.h])
		}},
		{"query for another height", "query response is for height 2", func() stateEvidence {
			return s.evidence(t, s.query(t, banktypes.StoreKey, key, s.h-1), s.h+1, s.hashes[s.h])
		}},
		{"header at H+1 signed by a key outside the policy", "commit signatures", func() stateEvidence {
			h := chain.header(t, s.h+1)
			h.AppHash = s.hashes[s.h]
			e := s.evidence(t, honest(), s.h+1, s.hashes[s.h])
			e.NextCommit = envelope(t, newTestChain("test-a", 100).commit(t, h))
			return e
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r, ok := verifyState(chain.pol, tc.e(), w)
			if ok || r.Claims.StoredValue.Status == statusProven || r.Claims.Balance.Status == statusProven || r.Balance != nil ||
				!strings.Contains(r.Claims.AppHash.Reason+" "+r.Claims.StoredValue.Reason, tc.reason) {
				t.Fatalf("want refusal %q: %+v %+v", tc.reason, r.Claims, r.Balance)
			}
		})
	}
	t.Run("evidence under another chain's policy", func(t *testing.T) {
		other := chain.pol
		other.ChainID = "test-b"
		r, ok := verifyState(other, s.valid(t, w), w)
		if ok || r.Claims.AppHash.Status == statusProven || r.Claims.Balance.Status == statusProven {
			t.Fatalf("test-a header accepted under test-b: %+v", r.Claims)
		}
	})
	t.Run("account stored under another address", func(t *testing.T) {
		// A proven value is still refused when the stored account names another address.
		r := stateReport{}
		holder := &authtypes.BaseAccount{Address: address(t, s.empty), AccountNumber: 1}
		any, err := codectypes.NewAnyWithValue(holder)
		if err != nil {
			t.Fatal(err)
		}
		readAccount(&r, stateWant{Kind: "account", Address: address(t, s.holder)}, wire(t, any))
		if r.Claims.Account.Status != statusNotProven || r.Claims.Sequence.Status != statusNotProven {
			t.Fatalf("mismatched stored address accepted: %+v", r.Claims)
		}
		// Any keeps unknown wire bytes and re-encodes them, so they pass the
		// canonical check and must be refused on their own.
		r = stateReport{}
		holder.Address = address(t, s.holder)
		if any, err = codectypes.NewAnyWithValue(holder); err != nil {
			t.Fatal(err)
		}
		any.XXX_unrecognized = []byte{0x18, 0x01}
		readAccount(&r, stateWant{Kind: "account", Address: holder.Address}, wire(t, any))
		if r.Claims.Account.Status != statusNotProven || r.Account != nil {
			t.Fatalf("stored account with unknown fields accepted: %+v %+v", r.Claims, r.Account)
		}
	})
}
