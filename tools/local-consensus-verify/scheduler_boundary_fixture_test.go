package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"cosmossdk.io/collections"
	sdkmath "cosmossdk.io/math"
	abci "github.com/cometbft/cometbft/abci/types"
	cmtcrypto "github.com/cometbft/cometbft/proto/tendermint/crypto"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	ics23 "github.com/cosmos/ics23/go"
	"github.com/stretchr/testify/require"

	zeroneapp "github.com/zerone-chain/zerone/app"
	schedule "github.com/zerone-chain/zerone/x/schedule/types"
)

func TestSchedulerBoundaryFixtureGenesisPreservesExistingState(t *testing.T) {
	for _, mode := range []string{"existing public account", "boundary auth collision", "boundary balance collision"} {
		t.Run(mode, func(t *testing.T) {
			encoding := zeroneapp.MakeEncodingConfig()
			cdc := encoding.Codec
			appState := zeroneapp.ModuleBasics.DefaultGenesis(cdc)
			address := fixtureAddress("existing-public-account")
			account := authtypes.NewBaseAccount(address, nil, 2, 0) // Preserve nonzero gentx account numbering.
			auth := authtypes.DefaultGenesisState()
			bankGenesis := banktypes.DefaultGenesisState()
			coins := sdk.NewCoins(sdk.NewInt64Coin("uother", 17), sdk.NewInt64Coin(schedule.Denom, 11))
			bankGenesis.Balances = []banktypes.Balance{{Address: address.String(), Coins: coins}}
			bankGenesis.Supply = coins
			if mode == "boundary auth collision" {
				account = authtypes.NewBaseAccount(fixtureBankBoundaryAddress(), nil, 2, 0)
			}
			if mode == "boundary balance collision" {
				bankGenesis.Balances[0].Address = fixtureBankBoundaryAddress().String()
			}
			var err error
			auth.Accounts, err = authtypes.PackAccounts(authtypes.GenesisAccounts{account})
			require.NoError(t, err)
			appState[authtypes.ModuleName] = cdc.MustMarshalJSON(auth)
			appState[banktypes.ModuleName] = cdc.MustMarshalJSON(bankGenesis)
			root := filepath.Join(t.TempDir(), "zerone-consensus-rehearsal.boundary")
			require.NoError(t, os.MkdirAll(filepath.Join(root, "coordinator", "config"), 0700))
			require.NoError(t, os.Mkdir(filepath.Join(root, "reports"), 0700))
			require.NoError(t, os.WriteFile(filepath.Join(root, ".zerone-consensus-rehearsal-owned"), nil, 0600))
			path := filepath.Join(root, "coordinator", "config", "genesis.json")
			require.NoError(t, writeFixtureJSON(path, map[string]any{"chain_id": fixtureChainID, "app_state": appState}))
			before, err := os.ReadFile(path)
			require.NoError(t, err)
			err = writeSchedulerFixture(root, fixtureChainID)
			if mode != "existing public account" {
				if mode == "boundary auth collision" {
					require.ErrorContains(t, err, "already has an auth account")
				} else {
					require.ErrorContains(t, err, "already funded")
				}
				after, err := os.ReadFile(path)
				require.NoError(t, err)
				require.Equal(t, before, after) // Reject without partially writing genesis or evidence.
				reports, err := os.ReadDir(filepath.Join(root, "reports"))
				require.NoError(t, err)
				require.Empty(t, reports)
				return
			}
			require.NoError(t, err)
			bz, err := os.ReadFile(path)
			require.NoError(t, err)
			var doc struct {
				AppState map[string]json.RawMessage `json:"app_state"`
			}
			require.NoError(t, json.Unmarshal(bz, &doc))
			require.NoError(t, zeroneapp.ModuleBasics.ValidateGenesis(cdc, encoding.TxConfig, doc.AppState))
			require.NoError(t, cdc.UnmarshalJSON(doc.AppState[authtypes.ModuleName], auth))
			accounts, err := authtypes.UnpackAccounts(auth.Accounts)
			require.NoError(t, err)
			require.Len(t, accounts, 6)
			require.Equal(t, account, accounts[0])
			require.NoError(t, cdc.UnmarshalJSON(doc.AppState[banktypes.ModuleName], bankGenesis))
			require.Len(t, bankGenesis.Balances, 6)
			require.Equal(t, banktypes.Balance{Address: address.String(), Coins: coins}, bankGenesis.Balances[0])
			require.Equal(t, "17uother,2600096uzrn", bankGenesis.Supply.String())
			sum := sdk.NewCoins()
			for _, b := range bankGenesis.Balances {
				sum = sum.Add(b.Coins...)
			}
			require.True(t, sum.Equal(bankGenesis.Supply))

			// Replay only bank sends from the GENERATED balances with actual indexed
			// collections, checking every checkpoint's balances. Not scheduler execution.
			store, ctx, bank, _ := indexedBankProofFixture(t)
			for _, b := range bankGenesis.Balances {
				addr, err := sdk.AccAddressFromBech32(b.Address)
				require.NoError(t, err)
				for _, c := range b.Coins {
					require.NoError(t, bank.Balances.Set(ctx, collections.Join(addr, c.Denom), c.Amount))
				}
			}
			escrow := authtypes.NewModuleAddress(schedule.ModuleName)
			fees := authtypes.NewModuleAddress(authtypes.FeeCollectorName)
			for _, height := range []int64{39, 40, 41, 51, 120} {
				for _, occurrence := range fixtureOccurrences {
					if occurrence.Executed != uint64(height) {
						continue
					}
					s := schedulerFixture(fixtureChainID).Schedules[occurrence.ID-1]
					amount, ok := sdkmath.NewIntFromString(s.AmountPerExecutionUzrn)
					require.True(t, ok)
					require.NoError(t, bank.SendCoins(ctx, escrow, fixtureAddress("recipient"), sdk.NewCoins(sdk.NewCoin(schedule.Denom, amount))))
					require.NoError(t, bank.SendCoins(ctx, escrow, fees, sdk.NewCoins(sdk.NewInt64Coin(schedule.Denom, 100000))))
				}
				for _, b := range fixtureExpectation(fixtureChainID, height).Balances {
					addr, err := sdk.AccAddressFromBech32(b.Address)
					require.NoError(t, err)
					require.Equal(t, b.Amount, bank.GetBalance(ctx, addr, schedule.Denom).Amount.String(), "checkpoint %d account %s", height, b.Address)
				}
			}
			require.Equal(t, "11", bank.GetBalance(ctx, address, schedule.Denom).Amount.String())
			require.Equal(t, "17", bank.GetBalance(ctx, address, "uother").Amount.String())
			commit := store.Commit()
			keys, err := schedulerKeys(fixtureExpectation(fixtureChainID, 120), fixtureChainID)
			require.NoError(t, err)
			for _, key := range keys {
				if key.Store == banktypes.StoreKey {
					response := queryBankProofFixture(t, store, key)
					require.NoError(t, verifyStateProof(response, key, 120, commit.Hash))
				}
			}
		})
	}
}

func TestSchedulerBankBoundaryOrderingAndCheckpoints(t *testing.T) {
	boundary := fixtureBankBoundaryAddress()
	require.Equal(t, sdk.AccAddress(bytes.Repeat([]byte{0xff}, 20)), boundary)
	roundTrip, err := sdk.AccAddressFromBech32(boundary.String())
	require.NoError(t, err)
	require.Equal(t, boundary, roundTrip)
	boundaryKey := bankProofExpectation(t, boundary, "1")
	// Pin exact SDK pair encoding as well as ordering: balance prefix, one-byte
	// address length, 20 address bytes, then the terminal unprefixed denom.
	want := append([]byte{0x02, 0x14}, bytes.Repeat([]byte{0xff}, 20)...)
	want = append(want, []byte("uzrn")...)
	require.Equal(t, want, boundaryKey.Key)
	require.True(t, bytes.Compare(boundaryKey.Key, banktypes.DenomAddressPrefix.Bytes()) < 0)
	for _, addr := range []sdk.AccAddress{
		fixtureAddress("creator"), fixtureAddress("recipient"),
		authtypes.NewModuleAddress(schedule.ModuleName), authtypes.NewModuleAddress(authtypes.FeeCollectorName),
	} {
		require.Len(t, addr, 20)
		require.True(t, bytes.Compare(bankProofExpectation(t, addr, "1").Key, boundaryKey.Key) < 0)
	}
	for _, height := range []int64{39, 40, 41, 51, 120} {
		e := fixtureExpectation(fixtureChainID, height)
		require.Contains(t, e.Balances, balanceExpectation{boundary.String(), "1"})
		keys, err := schedulerKeys(e, fixtureChainID)
		require.NoError(t, err)
		require.Contains(t, keys, boundaryKey) // Exact nonempty membership, never absence.
		require.Len(t, e.State.Schedules, 5)
		for _, s := range e.State.Schedules {
			require.NotEqual(t, boundary.String(), s.Creator)
			require.NotEqual(t, boundary.String(), s.Recipient)
		}
		for _, r := range e.State.Receipts {
			require.NotEqual(t, boundary.String(), r.Recipient)
		}
	}
}

// This is a known LOCAL fixture layout, not an upstream ICS23 repair. The
// unguarded case must still reject the SDK's empty-valued reverse-index neighbor.
// Reuse the diagnostic's real indexed bank/rootmulti/IAVL helpers, not raw leaves.
func TestSchedulerBoundaryFixtureSDKProofLayout(t *testing.T) {
	for _, guarded := range []bool{false, true} {
		name := "original empty-index neighbor rejects"
		if guarded {
			name = "one-coin balance neighbor verifies"
		}
		t.Run(name, func(t *testing.T) {
			store, ctx, bank, _ := indexedBankProofFixture(t)
			escrow := authtypes.NewModuleAddress(schedule.ModuleName)
			creator, recipient := fixtureAddress("creator"), fixtureAddress("recipient")
			fees := authtypes.NewModuleAddress(authtypes.FeeCollectorName)
			boundary := fixtureBankBoundaryAddress()
			for _, initial := range []struct {
				address sdk.AccAddress
				amount  int64
			}{
				{creator, 1000000}, {recipient, fixtureRecipientInitial},
				{escrow, 500077}, {fees, 100000},
			} {
				require.NoError(t, bank.Balances.Set(ctx, collections.Join(initial.address, schedule.Denom), sdkmath.NewInt(initial.amount)))
			}
			if guarded {
				require.NoError(t, bank.Balances.Set(ctx, collections.Join(boundary, schedule.Denom), sdkmath.OneInt()))
			}
			// Actual SDK sends remove the escrow balance and its reverse index.
			// No scheduler execution is claimed by this bank-only layout test.
			for _, occurrence := range fixtureOccurrences {
				s := schedulerFixture(fixtureChainID).Schedules[occurrence.ID-1]
				amount, ok := sdkmath.NewIntFromString(s.AmountPerExecutionUzrn)
				require.True(t, ok)
				require.NoError(t, bank.SendCoins(ctx, escrow, recipient, sdk.NewCoins(sdk.NewCoin(schedule.Denom, amount))))
				require.NoError(t, bank.SendCoins(ctx, escrow, fees, sdk.NewCoins(sdk.NewInt64Coin(schedule.Denom, 100000))))
			}
			_, err := bank.Balances.Get(ctx, collections.Join(escrow, schedule.Denom))
			require.ErrorIs(t, err, collections.ErrNotFound)
			require.True(t, bank.GetBalance(ctx, escrow, schedule.Denom).IsZero())
			require.Equal(t, "1000084", bank.GetBalance(ctx, recipient, schedule.Denom).Amount.String())
			require.Equal(t, "600000", bank.GetBalance(ctx, fees, schedule.Denom).Amount.String())
			commit := store.Commit()
			require.Equal(t, int64(120), commit.Version)
			expected := bankProofExpectation(t, escrow, "0")
			response := queryBankProofFixture(t, store, expected)
			require.Nil(t, response.Value)
			proof := new(ics23.CommitmentProof)
			require.NoError(t, proof.Unmarshal(response.ProofOps.Ops[0].Data))
			nonexist := proof.GetNonexist()
			require.NotNil(t, nonexist)
			require.NotNil(t, nonexist.Left)
			require.NotNil(t, nonexist.Right)
			if !guarded {
				require.True(t, bytes.HasPrefix(nonexist.Right.Key, banktypes.DenomAddressPrefix.Bytes()))
				require.Empty(t, nonexist.Right.Value)
				bankRoot, err := proof.Calculate()
				require.NoError(t, err)
				require.ErrorContains(t, nonexist.Verify(ics23.IavlSpec, bankRoot, expected.Key), "leaf op needs value")
				require.ErrorContains(t, verifyStateProof(response, expected, 120, commit.Hash), "proof did not verify absence of key")
				return
			}
			boundaryKey := bankProofExpectation(t, boundary, "1")
			require.Equal(t, boundaryKey.Key, nonexist.Right.Key)
			require.Equal(t, []byte("1"), nonexist.Right.Value)
			require.NotEmpty(t, nonexist.Left.Value)
			require.NoError(t, verifyStateProof(response, expected, 120, commit.Hash))
			member := queryBankProofFixture(t, store, boundaryKey)
			require.NoError(t, verifyStateProof(member, boundaryKey, 120, commit.Hash))
			for _, binding := range []struct {
				name     string
				response abci.ResponseQuery
				expected stateKeyExpectation
			}{{"escrow absence", response, expected}, {"boundary membership", member, boundaryKey}} {
				t.Run(binding.name, func(t *testing.T) {
					for name, mutate := range map[string]func(*abci.ResponseQuery, *stateKeyExpectation, *[]byte){
						"wrong root":      func(_ *abci.ResponseQuery, _ *stateKeyExpectation, root *[]byte) { (*root)[0] ^= 1 },
						"wrong key":       func(r *abci.ResponseQuery, e *stateKeyExpectation, _ *[]byte) { e.Key = []byte("wrong"); r.Key = e.Key },
						"wrong height":    func(r *abci.ResponseQuery, _ *stateKeyExpectation, _ *[]byte) { r.Height++ },
						"wrong value":     func(r *abci.ResponseQuery, _ *stateKeyExpectation, _ *[]byte) { r.Value = []byte("9") },
						"malformed inner": func(r *abci.ResponseQuery, _ *stateKeyExpectation, _ *[]byte) { r.ProofOps.Ops[0].Data = []byte{0xff} },
						"malformed outer": func(r *abci.ResponseQuery, _ *stateKeyExpectation, _ *[]byte) { r.ProofOps.Ops[1].Data = []byte{0xff} },
					} {
						t.Run(name, func(t *testing.T) {
							r, e, root := binding.response, binding.expected, bytes.Clone(commit.Hash)
							r.ProofOps = &cmtcrypto.ProofOps{Ops: append([]cmtcrypto.ProofOp(nil), binding.response.ProofOps.Ops...)}
							mutate(&r, &e, &root)
							require.Error(t, verifyStateProof(r, e, 120, root))
						})
					}
				})
			}
		})
	}
}
