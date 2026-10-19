package keeper_test

import (
	"fmt"
	"testing"

	"cosmossdk.io/log"
	"cosmossdk.io/store"
	storemetrics "cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/codec"
	addresscodec "github.com/cosmos/cosmos-sdk/codec/address"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/stretchr/testify/require"

	"github.com/zerone-chain/zerone/x/knowledge/keeper"
	"github.com/zerone-chain/zerone/x/knowledge/types"
)

const orderTestMint = "knowledge_order_test_mint"

// These are in-memory keeper/message tests with real SDK bank/auth stores,
// synthetic issuance and a supplied staking adapter, not signed transactions,
// application block replay or observations of a live escrow deficit.
func newKnowledgeOrderFixture(t *testing.T) (keeper.Keeper, sdk.Context, bankkeeper.BaseKeeper, *trackingStakingKeeper) {
	t.Helper()
	keys := storetypes.NewKVStoreKeys(types.StoreKey, authtypes.StoreKey, banktypes.StoreKey)
	db := dbm.NewMemDB()
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	ms := store.NewCommitMultiStore(db, log.NewNopLogger(), storemetrics.NewNoOpMetrics())
	ms.SetIAVLSyncPruning(true)
	for _, name := range []string{types.StoreKey, authtypes.StoreKey, banktypes.StoreKey} {
		ms.MountStoreWithDB(keys[name], storetypes.StoreTypeIAVL, db)
	}
	require.NoError(t, ms.LoadLatestVersion())
	ctx := sdk.NewContext(ms, cmtproto.Header{Height: 100}, false, log.NewNopLogger())
	registry := codectypes.NewInterfaceRegistry()
	authtypes.RegisterInterfaces(registry)
	banktypes.RegisterInterfaces(registry)
	cdc := codec.NewProtoCodec(registry)
	authority := authtypes.NewModuleAddress("gov").String()
	perms := map[string][]string{
		orderTestMint: {authtypes.Minter}, types.TrainingFundModuleName: nil,
		types.VindicationEscrowModuleName: nil, "development_fund": nil,
	}
	prefix := sdk.GetConfig().GetBech32AccountAddrPrefix()
	ak := authkeeper.NewAccountKeeper(cdc, runtime.NewKVStoreService(keys[authtypes.StoreKey]), authtypes.ProtoBaseAccount, perms, addresscodec.NewBech32Codec(prefix), prefix, authority)
	ak.InitGenesis(ctx, *authtypes.DefaultGenesisState())
	bank := bankkeeper.NewBaseKeeper(cdc, runtime.NewKVStoreService(keys[banktypes.StoreKey]), ak, map[string]bool{}, authority, log.NewNopLogger())
	bank.InitGenesis(ctx, banktypes.DefaultGenesisState())
	for _, name := range []string{orderTestMint, types.TrainingFundModuleName, types.VindicationEscrowModuleName, "development_fund"} {
		ak.GetModuleAccount(ctx, name)
	}
	staking := newTrackingStakingKeeper()
	k := keeper.NewKeeper(runtime.NewKVStoreService(keys[types.StoreKey]), cdc, authority, bank, staking)
	params := types.DefaultParams()
	require.NoError(t, k.SetParams(ctx, &params))
	return k, ctx.WithEventManager(sdk.NewEventManager()), bank, staking
}

func TestVindicationOrderTransfers(t *testing.T) {
	for _, funding := range []int64{5, 7, 0} {
		for _, reverseInsert := range []bool{false, true} {
			t.Run(fmt.Sprintf("funding=%d/reverse-insert=%t", funding, reverseInsert), func(t *testing.T) {
				k, ctx, bank, _ := newKnowledgeOrderFixture(t)
				if funding > 0 {
					coins := sdk.NewCoins(sdk.NewInt64Coin("uzrn", funding))
					require.NoError(t, bank.MintCoins(ctx, orderTestMint, coins))
					require.NoError(t, bank.SendCoinsFromModuleToModule(ctx, orderTestMint, types.VindicationEscrowModuleName, coins))
				}
				ids := []string{"fact-a", "fact-b"}
				if reverseInsert {
					ids[0], ids[1] = ids[1], ids[0]
				}
				for _, id := range ids {
					amount := "3"
					if id == "fact-b" {
						amount = "4"
					}
					require.NoError(t, k.SetVindicationPending(ctx, id, []types.VindicationEntry{{FactId: id, Verifier: "minority", SlashAmount: amount, Height: 100}}))
				}
				// Boundary/future/empty entries remain untouched. Expiry is strictly
				// beyond the window, and an insufficient transfer still deletes the
				// expired obligation: this ordering repair does NOT fix that policy.
				for _, entry := range []types.VindicationEntry{
					{FactId: "fact-boundary", SlashAmount: "9", Height: 190},
					{FactId: "fact-future", SlashAmount: "9", Height: 201},
				} {
					require.NoError(t, k.SetVindicationPending(ctx, entry.FactId, []types.VindicationEntry{entry}))
				}
				require.NoError(t, k.SetVindicationPending(ctx, "fact-empty", []types.VindicationEntry{}))
				ctx = ctx.WithBlockHeight(200).WithEventManager(sdk.NewEventManager())
				k.PruneExpiredVindications(ctx, 200, 10)

				wantPaid := funding
				if funding == 5 {
					wantPaid = 3 // fact-a wins access to the shared, underbacked pool.
				}
				require.Equal(t, wantPaid, bank.GetBalance(ctx, authtypes.NewModuleAddress("development_fund"), "uzrn").Amount.Int64())
				require.Equal(t, funding-wantPaid, bank.GetBalance(ctx, authtypes.NewModuleAddress(types.VindicationEscrowModuleName), "uzrn").Amount.Int64())
				require.Equal(t, funding, bank.GetSupply(ctx, "uzrn").Amount.Int64())
				require.Nil(t, k.GetVindicationPending(ctx, "fact-a"))
				require.Nil(t, k.GetVindicationPending(ctx, "fact-b"))
				pending := k.GetAllVindicationPending(ctx)
				require.Len(t, pending, 3)
				require.EqualValues(t, 190, pending["fact-boundary"][0].Height)
				require.EqualValues(t, 201, pending["fact-future"][0].Height)
				require.Contains(t, pending, "fact-empty")
				var expired []string
				for _, event := range ctx.EventManager().Events() {
					if event.Type == "zerone.knowledge.vindication_expired" {
						for _, attr := range event.Attributes {
							if attr.Key == "fact_id" {
								expired = append(expired, attr.Value)
							}
						}
					}
				}
				require.Equal(t, []string{"fact-a", "fact-b"}, expired)
				ctx = ctx.WithEventManager(sdk.NewEventManager())
				k.PruneExpiredVindications(ctx, 200, 10)
				require.Empty(t, ctx.EventManager().Events(), "deleted obligations must not transfer twice")
				require.Equal(t, wantPaid, bank.GetBalance(ctx, authtypes.NewModuleAddress("development_fund"), "uzrn").Amount.Int64())
				t.Logf("funded=%d treasury=%d escrow=%d expired=%v", funding, wantPaid, funding-wantPaid, expired)
			})
		}
	}
}
