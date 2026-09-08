package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"cosmossdk.io/log"
	"cosmossdk.io/store/metrics"
	"cosmossdk.io/store/rootmulti"
	storetypes "cosmossdk.io/store/types"
	abci "github.com/cometbft/cometbft/abci/types"
	cmtcrypto "github.com/cometbft/cometbft/proto/tendermint/crypto"
	dbm "github.com/cosmos/cosmos-db"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	zeroneapp "github.com/zerone-chain/zerone/app"
	schedulemodule "github.com/zerone-chain/zerone/x/schedule"
	schedule "github.com/zerone-chain/zerone/x/schedule/types"
)

const fixtureChainID = "zerone-consensus-rehearsal-1"

// The app import configures/seals the actual chain prefixes for full genesis validation.

func TestSchedulerFixtureExpectations(t *testing.T) {
	gs := schedulerFixture(fixtureChainID)
	require.NoError(t, gs.ValidateForChainID(fixtureChainID))
	require.False(t, gs.Params.AcceptNewSchedules)
	require.Equal(t, uint32(2), gs.Params.MaxDueRecordsPerBlock)
	require.Len(t, gs.Receipts, 1) // Labelled synthetic history, not a runtime observation.
	for _, height := range []int64{39, 40, 41, 51, 120} {
		e := fixtureExpectation(fixtureChainID, height)
		keys, err := schedulerKeys(e, fixtureChainID)
		require.NoError(t, err)
		require.NotEmpty(t, keys)
		require.False(t, e.State.Params.AcceptNewSchedules)
	}
	atDue := fixtureExpectation(fixtureChainID, 40)
	require.Equal(t, uint32(1), atDue.State.Schedules[1].ExecutionCount)
	require.Equal(t, uint32(1), atDue.State.Schedules[2].ExecutionCount)
	require.Zero(t, atDue.State.Schedules[3].ExecutionCount)
	require.Equal(t, "300053", atDue.State.TotalEscrowUzrn)
	late := fixtureExpectation(fixtureChainID, 41)
	require.Equal(t, uint64(51), late.State.Schedules[3].NextExecutionHeight)
	require.Equal(t, "200036", late.State.TotalEscrowUzrn)
	r := late.State.Receipts[3]
	require.Equal(t, uint64(40), r.DueHeight)
	require.Equal(t, uint64(41), r.ExecutedHeight)
	finished := fixtureExpectation(fixtureChainID, 120)
	require.Equal(t, "0", finished.State.TotalEscrowUzrn)
	require.Len(t, finished.State.Receipts, 6)
	require.Equal(t, "1000084", finished.Balances[1].Amount)
}

func TestSchedulerRejectsWrongOccurrenceAndContradictoryExpectations(t *testing.T) {
	for name, mutate := range map[string]func(*schedulerExpectation){
		"wrong occurrence": func(e *schedulerExpectation) {
			e.State.Receipts[0].OccurrenceId = schedule.OccurrenceID("other-chain", e.State.Receipts[0].ScheduleId, 1, 1, 3)
		},
		"wrong absent occurrence": func(e *schedulerExpectation) { e.AbsentOccurrences[0].OccurrenceId = e.State.Receipts[0].OccurrenceId },
		"open admission":          func(e *schedulerExpectation) { e.State.Params.AcceptNewSchedules = true },
		"wrong liability":         func(e *schedulerExpectation) { e.State.TotalEscrowUzrn = "0" },
		"duplicate key":           func(e *schedulerExpectation) { e.AbsentDue = append(e.AbsentDue, e.AbsentDue[0]) },
		"future receipt":          func(e *schedulerExpectation) { e.StateHeight = 39 },
		"wrong recurrence":        func(e *schedulerExpectation) { e.State.Schedules[3].NextExecutionHeight = 50 },
		"wrong schema":            func(e *schedulerExpectation) { e.Schema = "other" },
	} {
		t.Run(name, func(t *testing.T) {
			e := fixtureExpectation(fixtureChainID, 41)
			mutate(e)
			_, err := schedulerKeys(e, fixtureChainID)
			require.Error(t, err)
		})
	}
}

func TestSchedulerProofHeightBinding(t *testing.T) {
	e := fixtureExpectation(fixtureChainID, 41)
	anchor, err := schedulerAnchorHeight(e, 0, 43)
	require.NoError(t, err)
	require.Equal(t, int64(42), anchor)
	_, err = schedulerAnchorHeight(e, 41, 43)
	require.ErrorContains(t, err, "requires header 42")
	_, err = schedulerAnchorHeight(e, 42, 42)
	require.ErrorContains(t, err, "height 43")
	e.StateHeight = int64(^uint64(0) >> 1)
	_, err = schedulerAnchorHeight(e, 0, 43)
	require.Error(t, err)
}

// Uses real SDK IAVL+multistore proofs, not a stub proof runtime. This checks
// verifier mechanics; it does not claim daemon/application execution occurred.
func TestSchedulerMembershipAndAbsenceProofs(t *testing.T) {
	for _, height := range []int64{40, 41, 120} {
		e := fixtureExpectation(fixtureChainID, height)
		keys, err := schedulerKeys(e, fixtureChainID)
		require.NoError(t, err)
		db := dbm.NewMemDB()
		t.Cleanup(func() { require.NoError(t, db.Close()) })
		store := rootmulti.NewStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
		mounts := map[string]*storetypes.KVStoreKey{}
		for _, name := range []string{schedule.StoreKey, banktypes.StoreKey} {
			mounts[name] = storetypes.NewKVStoreKey(name)
			store.MountStoreWithDB(mounts[name], storetypes.StoreTypeIAVL, nil)
		}
		require.NoError(t, store.LoadVersion(0))
		require.NoError(t, store.SetInitialVersion(height))
		for _, key := range keys {
			if !key.Absent {
				store.GetKVStore(mounts[key.Store]).Set(key.Key, key.Value)
			}
		}
		commit := store.Commit()
		require.Equal(t, height, commit.Version)
		for _, key := range keys {
			res, err := store.Query(&storetypes.RequestQuery{Path: "/" + key.Store + "/key", Data: key.Key, Height: height, Prove: true})
			require.NoError(t, err)
			response := abci.ResponseQuery{Code: res.Code, Height: res.Height, Key: res.Key, Value: res.Value, ProofOps: res.ProofOps}
			require.NoError(t, verifyStateProof(response, key, height, commit.Hash), "store=%s key=%x", key.Store, key.Key)
			require.Error(t, verifyStateProof(response, key, height, bytes.Repeat([]byte{9}, 32)))
			require.Error(t, verifyStateProof(response, key, height+1, commit.Hash))
			wrongKey := key
			wrongKey.Key = append(bytes.Clone(key.Key), 9)
			require.ErrorContains(t, verifyStateProof(response, wrongKey, height, commit.Hash), "key mismatch")
			malformed := response
			malformed.ProofOps = nil
			require.ErrorContains(t, verifyStateProof(malformed, key, height, commit.Hash), "missing")
			malformed.ProofOps = &cmtcrypto.ProofOps{Ops: []cmtcrypto.ProofOp{{Type: "ics23:iavl", Key: key.Key, Data: []byte{0xff}}}}
			require.Error(t, verifyStateProof(malformed, key, height, commit.Hash))
			malformed = response
			malformed.Code = 1
			require.Error(t, verifyStateProof(malformed, key, height, commit.Hash))
			malformed = response
			malformed.Value = []byte("wrong value")
			require.Error(t, verifyStateProof(malformed, key, height, commit.Hash))
		}
	}
}

func TestSchedulerExpectationFileStrictness(t *testing.T) {
	path := filepath.Join(t.TempDir(), "expect.json")
	require.NoError(t, writeFixtureJSON(path, fixtureExpectation(fixtureChainID, 41)))
	e, digest, err := loadSchedulerExpectation(path, fixtureChainID)
	require.NoError(t, err)
	require.Equal(t, int64(41), e.StateHeight)
	require.Len(t, digest, 64)
	bz, err := os.ReadFile(path)
	require.NoError(t, err)
	for _, malformed := range [][]byte{append(bytes.Clone(bz), []byte(" {}")...), []byte(`{"schema":"x","unknown":true}`), bytes.Repeat([]byte(" "), 1<<20+1)} {
		require.NoError(t, os.WriteFile(path, malformed, 0600))
		_, _, err = loadSchedulerExpectation(path, fixtureChainID)
		require.Error(t, err)
	}
	legacy, err := json.Marshal(report{ChainID: fixtureChainID, Height: 12})
	require.NoError(t, err)
	require.NotContains(t, string(legacy), "scheduler")
}

func TestWriteSchedulerFixtureClosedBackedGenesis(t *testing.T) {
	root := filepath.Join(t.TempDir(), "zerone-consensus-rehearsal.test")
	require.NoError(t, os.MkdirAll(filepath.Join(root, "coordinator", "config"), 0700))
	require.NoError(t, os.Mkdir(filepath.Join(root, "reports"), 0700))
	genesisPath := filepath.Join(root, "coordinator", "config", "genesis.json")
	basic := schedulemodule.AppModuleBasic{}
	encoding := zeroneapp.MakeEncodingConfig()
	cdc := encoding.Codec
	appState := zeroneapp.ModuleBasics.DefaultGenesis(cdc)
	require.NoError(t, zeroneapp.ModuleBasics.ValidateGenesis(cdc, encoding.TxConfig, appState))
	genesis := map[string]any{"chain_id": fixtureChainID, "app_state": appState}
	require.NoError(t, writeFixtureJSON(genesisPath, genesis))
	require.ErrorContains(t, writeSchedulerFixture(root, fixtureChainID), "ownership marker")
	require.NoError(t, os.WriteFile(filepath.Join(root, ".zerone-consensus-rehearsal-owned"), nil, 0600))
	require.NoError(t, writeSchedulerFixture(root, fixtureChainID))
	bz, err := os.ReadFile(genesisPath)
	require.NoError(t, err)
	var doc struct {
		InitialHeight string                     `json:"initial_height"`
		AppState      map[string]json.RawMessage `json:"app_state"`
	}
	require.NoError(t, json.Unmarshal(bz, &doc))
	require.Equal(t, "10", doc.InitialHeight)
	// Exercise all app module validators, not only scheduler JSON decoding. This
	// is offline validation, not InitChain, a gentx-signing test or a node start.
	require.NoError(t, zeroneapp.ModuleBasics.ValidateGenesis(cdc, encoding.TxConfig, doc.AppState))
	require.Len(t, doc.AppState, len(appState))
	for name, original := range appState {
		if name != schedule.ModuleName && name != authtypes.ModuleName && name != banktypes.ModuleName {
			require.JSONEq(t, string(original), string(doc.AppState[name]), "unrelated module %s changed", name)
		}
	}
	raw := doc.AppState[schedule.ModuleName]
	require.NoError(t, basic.ValidateGenesis(nil, nil, raw))
	var gs schedule.GenesisState
	require.NoError(t, json.Unmarshal(raw, &gs))
	require.True(t, proto.Equal(schedulerFixture(fixtureChainID), &gs))
	var encoded struct {
		Params          map[string]json.RawMessage   `json:"params"`
		NextScheduleID  json.RawMessage              `json:"next_schedule_id"`
		TotalEscrowUzrn json.RawMessage              `json:"total_escrow_uzrn"`
		Schedules       []map[string]json.RawMessage `json:"schedules"`
		Receipts        []map[string]json.RawMessage `json:"receipts"`
	}
	require.NoError(t, json.Unmarshal(raw, &encoded))
	// These are exact JSON tokens, not values coerced by a protobuf decoder.
	require.Equal(t, "false", string(encoded.Params["accept_new_schedules"]))
	require.Equal(t, "2", string(encoded.Params["min_schedule_delay_blocks"]))
	require.Equal(t, "10", string(encoded.Params["min_interval_blocks"]))
	require.Equal(t, "6", string(encoded.NextScheduleID))
	require.Equal(t, `"500077"`, string(encoded.TotalEscrowUzrn))
	for i, record := range encoded.Schedules {
		require.Equal(t, "1", string(record["revision"]))
		status := "1"
		if i == 0 {
			status = "2"
		}
		require.Equal(t, status, string(record["status"]))
		for _, field := range []string{"amount_per_execution_uzrn", "execution_fee_uzrn", "principal_remaining_uzrn", "fee_remaining_uzrn"} {
			require.True(t, bytes.HasPrefix(record[field], []byte(`"`)), "monetary field %s must remain a string", field)
		}
	}
	require.Equal(t, "1", string(encoded.Receipts[0]["outcome"]))
	require.Equal(t, `"7"`, string(encoded.Receipts[0]["amount_uzrn"]))
	require.Equal(t, `"100000"`, string(encoded.Receipts[0]["fee_uzrn"]))

	for _, test := range []struct{ name, before, after, message string }{
		{"quoted uint64", `"min_schedule_delay_blocks": 2`, `"min_schedule_delay_blocks": "2"`, "cannot unmarshal string"},
		{"named schedule enum", `"status": 2`, `"status": "SCHEDULE_STATUS_COMPLETED"`, "cannot unmarshal string"},
		{"named outcome enum", `"outcome": 1`, `"outcome": "EXECUTION_OUTCOME_SUCCEEDED"`, "cannot unmarshal string"},
		{"invalid delay", `"min_schedule_delay_blocks": 2`, `"min_schedule_delay_blocks": 0`, "min_schedule_delay_blocks must be positive"},
	} {
		t.Run(test.name, func(t *testing.T) {
			require.Contains(t, string(raw), test.before)
			invalid := bytes.Replace(raw, []byte(test.before), []byte(test.after), 1)
			require.ErrorContains(t, basic.ValidateGenesis(nil, nil, invalid), test.message)
		})
	}

	var auth authtypes.GenesisState
	require.NoError(t, cdc.UnmarshalJSON(doc.AppState["auth"], &auth))
	require.NoError(t, authtypes.ValidateGenesis(auth))
	accounts, err := authtypes.UnpackAccounts(auth.Accounts)
	require.NoError(t, err)
	require.Len(t, accounts, 5)
	accountAddresses := map[string]bool{}
	for _, a := range accounts {
		require.NoError(t, a.Validate())
		require.Nil(t, a.GetPubKey())
		require.False(t, accountAddresses[a.GetAddress().String()], "duplicate auth account")
		accountAddresses[a.GetAddress().String()] = true
		if a.GetAddress().Equals(fixtureBankBoundaryAddress()) {
			require.IsType(t, &authtypes.BaseAccount{}, a) // No module permissions or signer.
			require.Zero(t, a.GetAccountNumber())
			require.Zero(t, a.GetSequence())
		}
	}
	require.True(t, accountAddresses[fixtureBankBoundaryAddress().String()])
	var bank banktypes.GenesisState
	require.NoError(t, cdc.UnmarshalJSON(doc.AppState["bank"], &bank))
	require.NoError(t, bank.Validate())
	supply := sdk.NewCoins()
	balanceAmounts := map[string]string{}
	for _, b := range bank.Balances {
		supply = supply.Add(b.Coins...)
		require.NotContains(t, balanceAmounts, b.Address, "duplicate bank account")
		require.True(t, accountAddresses[b.Address], "balance without auth account")
		require.Len(t, b.Coins, 1)
		balanceAmounts[b.Address] = b.Coins.String()
	}
	require.Equal(t, map[string]string{
		fixtureAddress("creator").String():                              "1000000uzrn",
		fixtureAddress("recipient").String():                            "1000007uzrn",
		authtypes.NewModuleAddress(schedule.ModuleName).String():        "500077uzrn",
		authtypes.NewModuleAddress(authtypes.FeeCollectorName).String(): "100000uzrn",
		fixtureBankBoundaryAddress().String():                           "1uzrn",
	}, balanceAmounts)
	require.True(t, bank.Supply.Equal(supply))
	require.Equal(t, "2600085uzrn", supply.String()) // Original 2600084 plus exactly one test-only coin.
	for _, height := range []string{"39", "40", "41", "51", "120"} {
		e, _, err := loadSchedulerExpectation(filepath.Join(root, "reports", "scheduler-expect-"+height+".json"), fixtureChainID)
		require.NoError(t, err)
		require.Contains(t, e.Balances, balanceExpectation{fixtureBankBoundaryAddress().String(), "1"})
		require.Len(t, e.Balances, 4)
	}
	manifestJSON, err := os.ReadFile(filepath.Join(root, "reports", "scheduler-test-manifest.json"))
	require.NoError(t, err)
	var manifest struct {
		TestOnly           bool `json:"test_only"`
		AcceptNewSchedules bool `json:"accept_new_schedules"`
		BankProofBoundary  struct {
			TestOnly   bool   `json:"test_only"`
			Address    string `json:"address"`
			Amount     string `json:"amount_uzrn"`
			Keyless    bool   `json:"keyless"`
			NeverSpent bool   `json:"never_spent"`
			Reason     string `json:"reason"`
			Limit      string `json:"limit"`
		} `json:"bank_proof_boundary"`
	}
	require.NoError(t, json.Unmarshal(manifestJSON, &manifest))
	require.True(t, manifest.TestOnly)
	require.False(t, manifest.AcceptNewSchedules)
	require.True(t, manifest.BankProofBoundary.TestOnly)
	require.Equal(t, fixtureBankBoundaryAddress().String(), manifest.BankProofBoundary.Address)
	require.Equal(t, "1", manifest.BankProofBoundary.Amount)
	require.True(t, manifest.BankProofBoundary.Keyless)
	require.True(t, manifest.BankProofBoundary.NeverSpent)
	require.Contains(t, manifest.BankProofBoundary.Reason, "included in supply")
	require.Contains(t, manifest.BankProofBoundary.Limit, "NOT general proof compatibility or a production solution")
	require.Contains(t, manifest.BankProofBoundary.Limit, "arbitrary layouts with an empty index neighbor")
	require.ErrorContains(t, writeSchedulerFixture(root, fixtureChainID), "non-fresh")
	require.ErrorContains(t, writeSchedulerFixture(root, "zerone-1"), "local rehearsal chain ID")
}

func TestMarshalSchedulerFixtureGenesisPreservesUint64Precision(t *testing.T) {
	for _, test := range []struct {
		name  string
		value uint64
		token string
	}{
		{"above float64 precision", 1<<53 + 1, "9007199254740993"},
		{"exhausted allocator", ^uint64(0), "18446744073709551615"},
	} {
		t.Run(test.name, func(t *testing.T) {
			gs := schedulerFixture(fixtureChainID)
			gs.NextScheduleId = test.value
			gs.Params.MinScheduleDelayBlocks = 1<<53 + 1
			require.NoError(t, gs.ValidateForChainID(fixtureChainID))
			raw, err := marshalSchedulerFixtureGenesis(gs)
			require.NoError(t, err)
			require.Contains(t, string(raw), `"next_schedule_id":`+test.token)
			require.Contains(t, string(raw), `"min_schedule_delay_blocks":9007199254740993`)
			require.Contains(t, string(raw), `"accept_new_schedules":false`)
			require.NoError(t, (schedulemodule.AppModuleBasic{}).ValidateGenesis(nil, nil, raw))
			var decoded schedule.GenesisState
			require.NoError(t, json.Unmarshal(raw, &decoded))
			require.True(t, proto.Equal(gs, &decoded))
		})
	}
}

func TestFixtureEvidenceNeverOverwritesExistingFilesOrSymlinks(t *testing.T) {
	path := filepath.Join(t.TempDir(), "fixture.json")
	require.NoError(t, writeFixtureJSON(path, "original"))
	require.Error(t, writeFixtureJSON(path, "replacement"))
	link := filepath.Join(t.TempDir(), "symlink.json")
	require.NoError(t, os.Symlink(path, link))
	require.Error(t, writeFixtureJSON(link, "replacement"))
	bz, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, "\"original\"\n", string(bz))
}
