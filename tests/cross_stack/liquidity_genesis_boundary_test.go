package cross_stack_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"cosmossdk.io/log"
	sdkmath "cosmossdk.io/math"
	abci "github.com/cometbft/cometbft/abci/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/baseapp"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/stretchr/testify/require"

	zeroneapp "github.com/zerone-chain/zerone/app"
	lpkeeper "github.com/zerone-chain/zerone/x/liquiditypool/keeper"
	lptypes "github.com/zerone-chain/zerone/x/liquiditypool/types"
	tokenstypes "github.com/zerone-chain/zerone/x/tokens/types"
)

const lpGenesisChainID = "zerone-lp-genesis-test"

func lpGenesisApp(t *testing.T) *zeroneapp.ZeroneApp {
	t.Helper()
	app := zeroneapp.NewZeroneApp(log.NewNopLogger(), dbm.NewMemDB(), nil, true,
		simtestutil.NewAppOptionsWithFlagHome(t.TempDir()),
		baseapp.SetChainID(lpGenesisChainID), baseapp.SetIAVLSyncPruning(true))
	t.Cleanup(func() { require.NoError(t, app.Close()) })
	return app
}

// All pool records and LP supply are produced by the real message server and
// SDK bank. The fixture is direct handler execution, not signed transport.
func lpGenesisExport(t *testing.T) (*zeroneapp.ZeroneApp, servertypes.ExportedApp, sdk.AccAddress) {
	t.Helper()
	app := lpGenesisApp(t)
	genesis := genesisStateWithValSet(t, app, app.DefaultGenesis())
	bz, err := json.Marshal(genesis)
	require.NoError(t, err)
	_, err = app.InitChain(&abci.RequestInitChain{ChainId: lpGenesisChainID, InitialHeight: 1,
		AppStateBytes: bz, ConsensusParams: simtestutil.DefaultConsensusParams})
	require.NoError(t, err)
	ctx := app.NewContextLegacy(false, cmtproto.Header{ChainID: lpGenesisChainID, Height: 1})
	creator := sdk.AccAddress(bytes.Repeat([]byte{0x61}, 20))
	coins := sdk.NewCoins(sdk.NewInt64Coin("uzrn", 30_000_000_000),
		sdk.NewInt64Coin("uatom", 40_000_000_000), sdk.NewInt64Coin("uusdc", 50_000_000_000))
	require.NoError(t, app.BankKeeper.MintCoins(ctx, tokenstypes.ModuleName, coins))
	require.NoError(t, app.BankKeeper.SendCoinsFromModuleToAccount(ctx, tokenstypes.ModuleName, creator, coins))
	bank := bankkeeper.NewMsgServerImpl(app.BankKeeper)
	_, err = bank.SetSendEnabled(ctx, banktypes.NewMsgSetSendEnabled(app.BankKeeper.GetAuthority(),
		[]*banktypes.SendEnabled{{Denom: "uzrn", Enabled: true}, {Denom: "uatom", Enabled: true}, {Denom: "uusdc", Enabled: true}}, nil))
	require.NoError(t, err)
	server := lpkeeper.NewMsgServerImpl(app.LiquidityPoolKeeper)
	params := app.LiquidityPoolKeeper.GetParams(ctx)
	params.PoolCreators = []string{creator.String()}
	params.AllowedPoolDenoms = []string{"uatom", "uusdc"}
	_, err = server.UpdateParams(ctx, &lptypes.MsgUpdateParams{Authority: app.LiquidityPoolKeeper.GetAuthority(), Params: params})
	require.NoError(t, err)
	for i, counter := range []string{"uatom", "uusdc"} {
		amount := sdkmath.NewInt(20_000_000_000).Add(sdkmath.NewInt(int64(i)).MulRaw(10_000_000_000))
		_, err = server.CreatePool(ctx, &lptypes.MsgCreatePool{Creator: creator.String(), DenomA: "uzrn", DenomB: counter,
			AmountA: "10000000000", AmountB: amount.String()})
		require.NoError(t, err)
	}
	// Commit the source's actual finalize cache before exporting. Merely
	// seeding a CheckTx cache would not prove a committed source round-trip.
	_, err = app.FinalizeBlock(&abci.RequestFinalizeBlock{Height: 1})
	require.NoError(t, err)
	_, err = app.Commit()
	require.NoError(t, err)
	ctx = app.NewContextLegacy(true, cmtproto.Header{ChainID: lpGenesisChainID, Height: 1})
	report, broken := lpkeeper.StateConsistencyInvariant(app.LiquidityPoolKeeper)(ctx)
	require.False(t, broken, report)
	require.Equal(t, uint64(2), app.LiquidityPoolKeeper.CountOpenPools(ctx))
	exported, err := app.ExportAppStateAndValidators(false, nil, nil)
	require.NoError(t, err)
	require.Equal(t, app.LastBlockHeight()+1, exported.Height)
	return app, exported, creator
}

// Move a single coin inside a complete exported bank genesis without changing
// total supply. This models an inconsistent imported liability, not a public
// transaction that can take module reserves.
func lpGenesisMove(t *testing.T, bank *banktypes.GenesisState, from, to, denom string) {
	t.Helper()
	fromFound, toFound := false, false
	one := sdk.NewInt64Coin(denom, 1)
	for i := range bank.Balances {
		balance := &bank.Balances[i]
		if balance.Address == from {
			require.True(t, balance.Coins.AmountOf(denom).IsPositive())
			balance.Coins = balance.Coins.Sub(one)
			fromFound = true
		}
		if balance.Address == to {
			balance.Coins = balance.Coins.Add(one)
			toFound = true
		}
	}
	require.True(t, fromFound)
	require.True(t, toFound)
}

// Characterized v5 behavior, not a new genesis-admission rule: the ordinary
// full-app import accepts structurally valid LP records with inconsistent bank
// backing/supply. The explicit custody audit is the discriminator. Keeping the
// passing predecessor test must not be mistaken for repairing this boundary.
func TestLiquidityGenesisCustodyAuditBoundary(t *testing.T) {
	source, exported, creator := lpGenesisExport(t)
	moduleAddress := authtypes.NewModuleAddress(lptypes.ModuleName)
	for _, tc := range []struct {
		name   string
		broken bool
		detail string
	}{
		{name: "healthy_shared_denom"},
		{name: "shared_denom_reserve_deficit", broken: true, detail: "below aggregate reserve liability"},
		{name: "lp_supply_short", broken: true, detail: "records LP supply"},
		{name: "lp_supply_excess", broken: true, detail: "records LP supply"},
		{name: "underlying_surplus"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var genesis zeroneapp.GenesisState
			require.NoError(t, json.Unmarshal(exported.AppState, &genesis))
			var bank banktypes.GenesisState
			source.AppCodec().MustUnmarshalJSON(genesis[banktypes.ModuleName], &bank)
			var pools lptypes.GenesisState
			require.NoError(t, json.Unmarshal(genesis[lptypes.ModuleName], &pools))
			require.NoError(t, pools.Validate())
			require.Len(t, pools.Pools, 2)
			switch tc.name {
			case "shared_denom_reserve_deficit":
				lpGenesisMove(t, &bank, moduleAddress.String(), creator.String(), "uzrn")
			case "underlying_surplus":
				lpGenesisMove(t, &bank, creator.String(), moduleAddress.String(), "uzrn")
			case "lp_supply_short", "lp_supply_excess":
				one := sdk.NewInt64Coin(pools.Pools[0].LpDenom, 1)
				changed := false
				for i := range bank.Balances {
					if bank.Balances[i].Address != creator.String() {
						continue
					}
					if tc.name == "lp_supply_short" {
						bank.Balances[i].Coins = bank.Balances[i].Coins.Sub(one)
						bank.Supply = bank.Supply.Sub(one)
					} else {
						bank.Balances[i].Coins = bank.Balances[i].Coins.Add(one)
						bank.Supply = bank.Supply.Add(one)
					}
					changed = true
				}
				require.True(t, changed)
			}
			// Each bank mutation is internally supply-consistent. Only the
			// relationship to unchanged LP obligations differs.
			total := sdk.NewCoins()
			for _, balance := range bank.Balances {
				total = total.Add(balance.Coins...)
			}
			require.Equal(t, bank.Supply.String(), total.String())
			genesis[banktypes.ModuleName] = source.AppCodec().MustMarshalJSON(&bank)
			bz, err := json.Marshal(genesis)
			require.NoError(t, err)
			imported := lpGenesisApp(t)
			_, err = imported.InitChain(&abci.RequestInitChain{ChainId: lpGenesisChainID, InitialHeight: exported.Height,
				AppStateBytes: bz, ConsensusParams: &exported.ConsensusParams})
			require.NoError(t, err, "characterized predecessor import; not an assertion that underbacking is safe")
			ctx := imported.NewContextLegacy(false, cmtproto.Header{ChainID: lpGenesisChainID, Height: exported.Height})
			// Read the real InitChain finalize cache, not a default genesis or
			// unrelated CheckTx cache. Check all supplied bank balances/supply
			// and the entire exported LP projection before block hooks run.
			for _, balance := range bank.Balances {
				address, err := sdk.AccAddressFromBech32(balance.Address)
				require.NoError(t, err)
				require.Equal(t, balance.Coins.String(), imported.BankKeeper.GetAllBalances(ctx, address).String())
			}
			for _, coin := range bank.Supply {
				require.Equal(t, coin.String(), imported.BankKeeper.GetSupply(ctx, coin.Denom).String())
			}
			require.JSONEq(t, string(genesis[lptypes.ModuleName]), string(imported.LiquidityPoolKeeper.ExportGenesisJSON(ctx)))
			lpGenesisAssertAudit(t, imported, ctx, tc.broken, tc.detail)
			if tc.name == "shared_denom_reserve_deficit" {
				backing := imported.BankKeeper.GetBalance(ctx, moduleAddress, "uzrn").Amount
				for _, pool := range pools.Pools {
					reserve, ok := sdkmath.NewIntFromString(pool.ReserveA)
					require.True(t, ok)
					require.Equal(t, "uzrn", pool.DenomA)
					require.True(t, backing.GT(reserve), "checking each pool alone would miss this aggregate deficit")
				}
			}
			// Continue with the exported height and actual ABCI lifecycle;
			// unlike AdvanceBlocks this path preserves returned hook errors.
			_, err = imported.FinalizeBlock(&abci.RequestFinalizeBlock{Height: exported.Height})
			require.NoError(t, err)
			_, err = imported.Commit()
			require.NoError(t, err)
			ctx = imported.NewContextLegacy(true, cmtproto.Header{ChainID: lpGenesisChainID, Height: exported.Height})
			lpGenesisAssertAudit(t, imported, ctx, tc.broken, tc.detail)
			require.Equal(t, pools.NextPoolId, imported.LiquidityPoolKeeper.GetNextPoolId(ctx))
			require.Equal(t, uint64(len(pools.Pools)), imported.LiquidityPoolKeeper.CountOpenPools(ctx))
			for _, pool := range pools.Pools {
				restored, found := imported.LiquidityPoolKeeper.GetPool(ctx, pool.PoolId)
				require.True(t, found)
				expected, err := json.Marshal(pool)
				require.NoError(t, err)
				actual, err := json.Marshal(restored)
				require.NoError(t, err)
				require.JSONEq(t, string(expected), string(actual))
				require.Equal(t, pool.PoolId, imported.LiquidityPoolKeeper.GetPoolByDenoms(ctx, pool.DenomA, pool.DenomB).PoolId)
			}
			// Normal export also remains separate from explicit custody audit.
			again, err := imported.ExportAppStateAndValidators(false, nil, nil)
			require.NoError(t, err)
			require.Equal(t, exported.Height+1, again.Height)
			var reexport zeroneapp.GenesisState
			require.NoError(t, json.Unmarshal(again.AppState, &reexport))
			var reexportedLP lptypes.GenesisState
			require.NoError(t, json.Unmarshal(reexport[lptypes.ModuleName], &reexportedLP))
			require.NoError(t, reexportedLP.Validate())
			require.Equal(t, pools.NextPoolId, reexportedLP.NextPoolId)
			var reexportedBank banktypes.GenesisState
			imported.AppCodec().MustUnmarshalJSON(reexport[banktypes.ModuleName], &reexportedBank)
			for _, address := range []string{moduleAddress.String(), creator.String()} {
				require.Equal(t, lpGenesisBalance(t, bank, address).String(), lpGenesisBalance(t, reexportedBank, address).String())
			}
		})
	}
}

type lpGenesisInvariantRegistrar func(string, string, sdk.Invariant)

func (register lpGenesisInvariantRegistrar) RegisterRoute(module, route string, invariant sdk.Invariant) {
	register(module, route, invariant)
}

func lpGenesisAssertAudit(t *testing.T, app *zeroneapp.ZeroneApp, ctx sdk.Context, wantBroken bool, detail string) {
	t.Helper()
	// Invoke the actual module's registration contract explicitly. This test
	// checkpoint is not a claim that app InitChain/BeginBlock dispatches it.
	owner, ok := app.ModuleManager.Modules[lptypes.ModuleName].(interface {
		RegisterInvariants(sdk.InvariantRegistry)
	})
	require.True(t, ok)
	registered := 0
	owner.RegisterInvariants(lpGenesisInvariantRegistrar(func(module, route string, invariant sdk.Invariant) {
		require.Equal(t, lptypes.ModuleName, module)
		require.Equal(t, "state-consistency", route)
		registered++
		report, broken := invariant(ctx)
		require.Equal(t, wantBroken, broken, report)
		if detail != "" {
			require.Contains(t, report, detail)
		}
	}))
	require.Equal(t, 1, registered)
}

func lpGenesisBalance(t *testing.T, bank banktypes.GenesisState, address string) sdk.Coins {
	t.Helper()
	for _, balance := range bank.Balances {
		if balance.Address == address {
			return balance.Coins
		}
	}
	t.Fatalf("missing expected exported bank address %s", address)
	return nil
}
