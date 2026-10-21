//go:build ignore

// Synthetic fixture ONLY. Compile against OLD_COMMIT, not H1 or trunk.
// Creates an application DB with a fixed public validator key; no private key.
package main

import (
	"encoding/hex"
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"time"

	"cosmossdk.io/log"
	sdkmath "cosmossdk.io/math"
	abci "github.com/cometbft/cometbft/abci/types"
	cmted25519 "github.com/cometbft/cometbft/crypto/ed25519"
	cmttypes "github.com/cometbft/cometbft/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/baseapp"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	zeroneapp "github.com/zerone-chain/zerone/app"
)

type options map[string]interface{}

func (o options) Get(k string) interface{} { return o[k] }
func check(err error) {
	if err != nil {
		panic(err)
	}
}
func main() {
	output := flag.String("output", "", "NEW synthetic fixture directory")
	flag.Parse()
	if *output == "" {
		panic("output required")
	}
	_, err := os.Lstat(*output)
	if !os.IsNotExist(err) {
		panic("output must not exist")
	}
	check(os.Mkdir(*output, 0700))
	// App home is a separate disposable directory, not included in the copy.
	home, err := os.MkdirTemp(filepath.Dir(*output), "fixture-home-")
	check(err)
	db, err := dbm.NewDB("application", dbm.GoLevelDBBackend, *output)
	check(err)
	const chain = "zerone-h1-fixture-1"
	a := zeroneapp.NewZeroneApp(log.NewNopLogger(), db, nil, true,
		options{"home": home, "oracle.enabled": false}, baseapp.SetChainID(chain))
	genesis := a.DefaultGenesis()
	// A public key only; this fixture can never run a consensus signer.
	pub := make(cmted25519.PubKey, 32)
	copy(pub, []byte("synthetic public validator key"))
	publicKey, err := cryptocodec.FromCmtPubKeyInterface(pub)
	check(err)
	keyAny, err := codectypes.NewAnyWithValue(publicKey)
	check(err)
	operator := sdk.ValAddress(pub.Address()).String()
	account := authtypes.NewBaseAccount(sdk.AccAddress(pub.Address()), nil, 0, 0)
	bond := sdk.DefaultPowerReduction
	genesis[authtypes.ModuleName] = a.AppCodec().MustMarshalJSON(
		authtypes.NewGenesisState(authtypes.DefaultParams(), []authtypes.GenesisAccount{account}))
	validator := stakingtypes.Validator{OperatorAddress: operator, ConsensusPubkey: keyAny,
		Status: stakingtypes.Bonded, Tokens: bond, DelegatorShares: sdkmath.LegacyOneDec(),
		Description: stakingtypes.Description{}, MinSelfDelegation: sdkmath.ZeroInt(),
		Commission: stakingtypes.NewCommission(sdkmath.LegacyZeroDec(), sdkmath.LegacyZeroDec(), sdkmath.LegacyZeroDec())}
	genesis[stakingtypes.ModuleName] = a.AppCodec().MustMarshalJSON(stakingtypes.NewGenesisState(
		stakingtypes.DefaultParams(), []stakingtypes.Validator{validator},
		[]stakingtypes.Delegation{stakingtypes.NewDelegation(account.GetAddress().String(), operator, sdkmath.LegacyOneDec())}))
	coins := sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, bond))
	genesis[banktypes.ModuleName] = a.AppCodec().MustMarshalJSON(banktypes.NewGenesisState(
		banktypes.DefaultGenesisState().Params,
		[]banktypes.Balance{{Address: authtypes.NewModuleAddress(stakingtypes.BondedPoolName).String(), Coins: coins}},
		coins, nil, nil))
	body, err := json.Marshal(genesis)
	check(err)
	at := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	params := cmttypes.DefaultConsensusParams().ToProto()
	_, err = a.InitChain(&abci.RequestInitChain{ChainId: chain, Time: at, AppStateBytes: body, ConsensusParams: &params})
	check(err)
	_, err = a.FinalizeBlock(&abci.RequestFinalizeBlock{Height: 1, Time: at})
	check(err)
	_, err = a.Commit()
	check(err)
	manifest := map[string]interface{}{"schema": "zerone-h1-scrubbed-copy/v1",
		"chain_id": chain, "height": a.LastBlockHeight(), "app_hash": hex.EncodeToString(a.LastCommitID().Hash),
		"block_time": at.Format(time.RFC3339), "backend": "goleveldb", "key_scrubbed": true,
		"stopped_consistent_copy": true, "evidence_class": "synthetic-fixture",
		"provenance": "Old-source synthetic genesis, fixed public key only; not a testnet snapshot"}
	check(a.Close())
	body, err = json.MarshalIndent(manifest, "", "  ")
	check(err)
	check(os.WriteFile(filepath.Join(*output, "copy.json"), append(body, '\n'), 0600))
	check(json.NewEncoder(os.Stdout).Encode(manifest))
}
