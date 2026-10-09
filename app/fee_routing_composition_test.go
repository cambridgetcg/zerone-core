package app

import (
	"encoding/json"
	"testing"

	"cosmossdk.io/core/header"
	"cosmossdk.io/log"
	abci "github.com/cometbft/cometbft/abci/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/baseapp"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	"github.com/stretchr/testify/require"

	knowledgetypes "github.com/zerone-chain/zerone/x/knowledge/types"
	vestingtypes "github.com/zerone-chain/zerone/x/vesting_rewards/types"
)

// TestFeeRoutingCompositionActualBank follows the registered BeginBlock order,
// not a mock transfer log or a second fee calculator. The zero-vote fixture
// deliberately exercises SDK community-pool accounting, not delegator payouts,
// transaction admission, independent validators, or a deployed network.
func TestFeeRoutingCompositionActualBank(t *testing.T) {
	db := dbm.NewMemDB()
	application := NewZeroneApp(log.NewNopLogger(), db, nil, true,
		simtestutil.NewAppOptionsWithFlagHome(t.TempDir()),
		baseapp.SetChainID("fee-handoff-fixture"), baseapp.SetIAVLSyncPruning(true))
	t.Cleanup(func() { require.NoError(t, application.Close()) })
	genesis, err := json.Marshal(sdk053IBC10GenesisWithValidator(t, application))
	require.NoError(t, err)
	ctx := application.NewUncachedContext(false, cmtproto.Header{ChainID: "fee-handoff-fixture"})
	_, err = application.InitChainer(ctx, &abci.RequestInitChain{ChainId: ctx.ChainID(), AppStateBytes: genesis})
	require.NoError(t, err)

	balance := func(module string) sdk.Coins {
		return application.BankKeeper.GetAllBalances(ctx, authtypes.NewModuleAddress(module))
	}
	seedFees := func(coins sdk.Coins) {
		// Synthetic test issuance only, through the existing permitted module.
		require.NoError(t, application.BankKeeper.MintCoins(ctx, knowledgetypes.BootstrapFundModuleName, coins))
		require.NoError(t, application.BankKeeper.SendCoinsFromModuleToModule(ctx,
			knowledgetypes.BootstrapFundModuleName, authtypes.FeeCollectorName, coins))
	}
	begin := func(height int64) {
		ctx = ctx.WithBlockHeight(height).WithHeaderInfo(header.Info{Height: height, ChainID: ctx.ChainID()})
		_, err := application.BeginBlocker(ctx)
		require.NoError(t, err)
	}
	assertCustody := func(research, development int64, distributed sdk.Coins) {
		require.Equal(t, sdk.NewCoins(sdk.NewInt64Coin("uzrn", research)).String(), balance(vestingtypes.ResearchFundModuleName).String())
		require.Equal(t, sdk.NewCoins(sdk.NewInt64Coin("uzrn", development)).String(), balance(vestingtypes.DevelopmentFundModuleName).String())
		require.True(t, balance(vestingtypes.ModuleName).IsZero(), "research escrow must finish at its existing owner")
		require.True(t, balance(authtypes.FeeCollectorName).IsZero(), "SDK distribution must consume the entire remainder")
		require.Equal(t, distributed.String(), balance(distrtypes.ModuleName).String())
		pool, err := application.DistrKeeper.FeePool.Get(ctx)
		require.NoError(t, err)
		require.Equal(t, sdk.NewDecCoinsFromCoins(distributed...).String(), pool.CommunityPool.String())
	}

	fees := sdk.NewCoins(sdk.NewInt64Coin("atom", 7), sdk.NewInt64Coin("uzrn", 1000003))
	seedFees(fees)
	nativeSupply := application.BankKeeper.GetSupply(ctx, "uzrn")
	otherSupply := application.BankKeeper.GetSupply(ctx, "atom")

	// Neither router consumes fees at genesis height.
	begin(1)
	require.Equal(t, fees.String(), balance(authtypes.FeeCollectorName).String())
	for _, module := range []string{vestingtypes.ModuleName, vestingtypes.ResearchFundModuleName,
		vestingtypes.DevelopmentFundModuleName, distrtypes.ModuleName} {
		require.True(t, balance(module).IsZero(), module)
	}

	// Default native slices floor independently; dust and other denominations
	// stay in the collector until the real SDK module sweeps them.
	begin(2)
	remainder := sdk.NewCoins(sdk.NewInt64Coin("atom", 7), sdk.NewInt64Coin("uzrn", 770003))
	assertCustody(33300, 196700, remainder)
	require.Equal(t, nativeSupply, application.BankKeeper.GetSupply(ctx, "uzrn"))
	require.Equal(t, otherSupply, application.BankKeeper.GetSupply(ctx, "atom"))

	// An empty next block must not split the retained SDK balance again.
	begin(3)
	assertCustody(33300, 196700, remainder)
	require.Equal(t, nativeSupply, application.BankKeeper.GetSupply(ctx, "uzrn"))
	require.Equal(t, otherSupply, application.BankKeeper.GetSupply(ctx, "atom"))

	// A sub-share fee survives integer rounding instead of disappearing.
	dust := sdk.NewCoins(sdk.NewInt64Coin("uzrn", 1))
	seedFees(dust)
	nativeSupply = application.BankKeeper.GetSupply(ctx, "uzrn")
	begin(4)
	assertCustody(33300, 196700, remainder.Add(dust...))
	require.Equal(t, nativeSupply, application.BankKeeper.GetSupply(ctx, "uzrn"))
	require.Equal(t, otherSupply, application.BankKeeper.GetSupply(ctx, "atom"))
}
