package keeper_test

import (
	"fmt"
	"go/scanner"
	"go/token"
	"os"
	"slices"
	"strings"
	"testing"

	corestore "cosmossdk.io/core/store"
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
	return newKnowledgeOrderFixtureWithStore(t, func(s corestore.KVStoreService) corestore.KVStoreService { return s })
}

func newKnowledgeOrderFixtureWithStore(t *testing.T, wrap func(corestore.KVStoreService) corestore.KVStoreService) (keeper.Keeper, sdk.Context, bankkeeper.BaseKeeper, *trackingStakingKeeper) {
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
	k := keeper.NewKeeper(wrap(runtime.NewKVStoreService(keys[types.StoreKey])), cdc, authority, bank, staking)
	params := types.DefaultParams()
	require.NoError(t, k.SetParams(ctx, &params))
	return k, ctx.WithEventManager(sdk.NewEventManager()), bank, staking
}

func TestVindicationOrderTransfers(t *testing.T) {
	ids := []string{"fact-a", "fact-b", "fact-c", "fact-d", "fact-e", "fact-f", "fact-g", "fact-h", "fact-i"}
	amounts := make(map[string]int64, len(ids))
	var fullyFunded int64
	for i, id := range ids {
		amounts[id] = int64(i + 3)
		fullyFunded += amounts[id]
	}
	for _, funding := range []int64{5, fullyFunded, 0} {
		for _, reverseInsert := range []bool{false, true} {
			t.Run(fmt.Sprintf("funding=%d/reverse-insert=%t", funding, reverseInsert), func(t *testing.T) {
				k, ctx, bank, _ := newKnowledgeOrderFixture(t)
				require.NoError(t, k.EnableReformulationVindicationOrdering(ctx))
				if funding > 0 {
					coins := sdk.NewCoins(sdk.NewInt64Coin("uzrn", funding))
					require.NoError(t, bank.MintCoins(ctx, orderTestMint, coins))
					require.NoError(t, bank.SendCoinsFromModuleToModule(ctx, orderTestMint, types.VindicationEscrowModuleName, coins))
				}
				insertion := slices.Clone(ids)
				if reverseInsert {
					slices.Reverse(insertion)
				}
				for _, id := range insertion {
					require.NoError(t, k.SetVindicationPending(ctx, id, []types.VindicationEntry{{FactId: id, Verifier: "minority", SlashAmount: fmt.Sprint(amounts[id]), Height: 100}}))
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
				for _, id := range ids {
					require.Nil(t, k.GetVindicationPending(ctx, id))
				}
				pending := k.GetAllVindicationPending(ctx)
				require.Len(t, pending, 3)
				require.EqualValues(t, 190, pending["fact-boundary"][0].Height)
				require.EqualValues(t, 201, pending["fact-future"][0].Height)
				require.Contains(t, pending, "fact-empty")
				var wantEvents sdk.Events
				for _, id := range ids {
					if funding == fullyFunded || (funding == 5 && id == "fact-a") {
						from := authtypes.NewModuleAddress(types.VindicationEscrowModuleName)
						to := authtypes.NewModuleAddress("development_fund")
						coins := sdk.NewCoins(sdk.NewInt64Coin("uzrn", amounts[id]))
						wantEvents = append(wantEvents, banktypes.NewCoinSpentEvent(from, coins), banktypes.NewCoinReceivedEvent(to, coins),
							sdk.NewEvent(banktypes.EventTypeTransfer, sdk.NewAttribute(banktypes.AttributeKeyRecipient, to.String()), sdk.NewAttribute(banktypes.AttributeKeySender, from.String()), sdk.NewAttribute(sdk.AttributeKeyAmount, coins.String())),
							sdk.NewEvent(sdk.EventTypeMessage, sdk.NewAttribute(banktypes.AttributeKeySender, from.String())))
					}
					wantEvents = append(wantEvents, sdk.NewEvent("zerone.knowledge.vindication_expired",
						sdk.NewAttribute("fact_id", id), sdk.NewAttribute("entry_count", "1"), sdk.NewAttribute("creed_commitment", "10")))
				}
				require.Equal(t, wantEvents, ctx.EventManager().Events(), "complete bank and nine expiry events, in key order")
				ctx = ctx.WithEventManager(sdk.NewEventManager())
				k.PruneExpiredVindications(ctx, 200, 10)
				require.Empty(t, ctx.EventManager().Events(), "deleted obligations must not transfer twice")
				require.Equal(t, wantPaid, bank.GetBalance(ctx, authtypes.NewModuleAddress("development_fund"), "uzrn").Amount.Int64())
				t.Logf("funded=%d treasury=%d escrow=%d expired=%v", funding, wantPaid, funding-wantPaid, ids)
			})
		}
	}
}

// Nine facts make accidental ordered map traversal unlikely, not impossible.
// This narrow source guard makes a reverted sort fail on EVERY plain go test
// run, without relying on map randomness or an external overlay runner.
// The bank-backed test above establishes behavior; this is only a backstop for
// the reviewed enabled-sort-before-processing structure, not a general proof.
// Its brittleness is intentional: a renamed variable or slices.Sort refactor
// must update this structural backstop as well as pass the behavioral test.
func TestVindicationOrderSortGuard(t *testing.T) {
	source, err := os.ReadFile("vindication.go")
	require.NoError(t, err)
	hasSort := func(source []byte) bool {
		var scan scanner.Scanner
		scan.Init(token.NewFileSet().AddFile("vindication.go", -1, len(source)), source, nil, 0)
		var tokens strings.Builder
		for {
			_, tok, literal := scan.Scan()
			if tok == token.EOF {
				break
			}
			if tok == token.SEMICOLON {
				continue
			}
			if literal != "" {
				tokens.WriteString(literal)
			} else {
				tokens.WriteString(tok.String())
			}
		}
		return strings.Contains(tokens.String(), "ifenabled{sort.Strings(factIDs)}for_,factId:=rangefactIDs{")
	}
	require.True(t, hasSort(source), "enabled pruning must sort before processing")
	for _, replacement := range []string{"", "// sort.Strings(factIDs)", "slices.Reverse(factIDs)"} {
		mutated := strings.Replace(string(source), "sort.Strings(factIDs)", replacement, 1)
		require.NotEqual(t, string(source), mutated)
		require.False(t, hasSort([]byte(mutated)), "guard must reject removed, commented or reversed sort")
	}
}
