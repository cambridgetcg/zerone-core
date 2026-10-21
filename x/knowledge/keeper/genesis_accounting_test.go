package keeper_test

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"testing"

	corestore "cosmossdk.io/core/store"
	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// In-memory SDK auth/bank keepers, synthetic issuance, no node or signed tx.
// The target imports bank genesis BEFORE knowledge, as the application does.
func TestGenesisAccountingBankCoupledRoundTrip(t *testing.T) {
	for _, payout := range []int64{0, 1500} {
		for _, ordering := range []bool{false, true} {
			t.Run(fmt.Sprintf("paid=%d/ordering=%t", payout, ordering), func(t *testing.T) {
				var sourceService corestore.KVStoreService
				k, ctx, bank, _ := newKnowledgeOrderFixtureWithStore(t, func(s corestore.KVStoreService) corestore.KVStoreService {
					sourceService = s
					return s
				})
				gs := types.DefaultGenesis()
				gs.ReformulationVindicationOrderingEnabled = ordering
				require.NoError(t, k.InitGenesis(ctx, gs))
				sponsor := sdk.AccAddress(bytes.Repeat([]byte{0x31}, 20))
				worker := sdk.AccAddress(bytes.Repeat([]byte{0x32}, 20))
				coins := sdk.NewCoins(sdk.NewInt64Coin("uzrn", 10000))
				require.NoError(t, bank.MintCoins(ctx, orderTestMint, coins))
				require.NoError(t, bank.SendCoinsFromModuleToAccount(ctx, orderTestMint, sponsor, coins))
				for _, b := range []*types.AugmentationBounty{
					{Id: "untouched-or-paid", Active: true, SponsorAddress: sponsor.String(), EscrowLocked: "4500", ExpiresAtBlock: 200},
					{Id: "inactive-retained", Active: false, SponsorAddress: sponsor.String(), EscrowLocked: "500", ExpiresAtBlock: 90},
				} {
					require.NoError(t, k.SetAugmentationBounty(ctx, b))
					amount, ok := sdkmath.NewIntFromString(b.EscrowLocked)
					require.True(t, ok)
					require.NoError(t, k.EscrowAugmentationBounty(ctx, sponsor.String(), b.Id, amount))
				}
				reserve := sdk.NewCoins(sdk.NewInt64Coin("uzrn", 700))
				require.NoError(t, bank.SendCoinsFromAccountToModule(ctx, sponsor, types.TrainingFundModuleName, reserve))
				if payout > 0 {
					require.NoError(t, k.ReleaseAugmentationPayout(ctx, "untouched-or-paid", worker.String(), sdkmath.NewInt(payout)))
				}
				for i := 0; i < 10; i++ {
					require.NoError(t, k.IncrementBootstrapClaimCount(ctx, sponsor.String()))
					require.NoError(t, k.IncrementBootstrapEpochCount(ctx, 0))
				}
				require.NoError(t, k.IncrementBootstrapClaimCount(ctx, worker.String()))
				require.NoError(t, k.IncrementBootstrapEpochCount(ctx, 77))
				// Explicit zero keys are also part of the primary inventory.
				store := sourceService.OpenKVStore(ctx)
				require.NoError(t, store.Set(append(bytes.Clone(types.BootstrapClaimCountPrefix), []byte("ExactCase")...), make([]byte, 8)))
				require.NoError(t, store.Set(binary.BigEndian.AppendUint64(bytes.Clone(types.BootstrapEpochCountPrefix), ^uint64(0)), make([]byte, 8)))
				exported := k.ExportGenesis(ctx)
				require.NotNil(t, exported.AccountingState)
				require.Equal(t, "700", exported.AccountingState.TrainingFundNonEscrowBalance)
				require.Len(t, exported.AccountingState.AugmentationEscrows, 2)
				encoded, err := protojson.Marshal(exported)
				require.NoError(t, err)
				var imported types.GenesisState
				require.NoError(t, protojson.Unmarshal(encoded, &imported))
				require.NoError(t, imported.Validate())

				var targetService corestore.KVStoreService
				fresh, freshCtx, freshBank, _ := newKnowledgeOrderFixtureWithStore(t, func(s corestore.KVStoreService) corestore.KVStoreService {
					targetService = s
					return s
				})
				bankGenesis := bank.ExportGenesis(ctx)
				freshBank.InitGenesis(freshCtx, bankGenesis)
				supplyBefore := freshBank.GetSupply(freshCtx, "uzrn")
				require.NoError(t, fresh.InitGenesis(freshCtx, &imported))
				require.Equal(t, supplyBefore, freshBank.GetSupply(freshCtx, "uzrn"), "knowledge import must not mint")
				require.True(t, proto.Equal(exported.AccountingState, fresh.ExportGenesis(freshCtx).AccountingState))
				require.Equal(t, accountingPrimarySnapshot(t, store), accountingPrimarySnapshot(t, targetService.OpenKVStore(freshCtx)))
				require.EqualValues(t, 10, fresh.GetBootstrapClaimCount(freshCtx, sponsor.String()))
				require.EqualValues(t, 10, fresh.GetBootstrapEpochCount(freshCtx, 0))
				require.EqualValues(t, 1, fresh.GetBootstrapEpochCount(freshCtx, 77))
				require.Equal(t, "500", fresh.GetEscrowLocked(freshCtx, "inactive-retained").String())
				require.Equal(t, sdkmath.NewInt(4500-payout), fresh.GetEscrowLocked(freshCtx, "untouched-or-paid"))

				ctx = ctx.WithBlockHeight(201).WithEventManager(sdk.NewEventManager())
				freshCtx = freshCtx.WithBlockHeight(201).WithEventManager(sdk.NewEventManager())
				beforeRefund := bank.GetBalance(ctx, sponsor, "uzrn").Amount
				require.NoError(t, k.ProcessRouteBLifecycle(ctx))
				require.NoError(t, fresh.ProcessRouteBLifecycle(freshCtx))
				remaining := sdkmath.NewInt(4500 - payout)
				refund := remaining.Sub(remaining.MulRaw(30000).QuoRaw(1000000))
				require.Equal(t, refund, bank.GetBalance(ctx, sponsor, "uzrn").Amount.Sub(beforeRefund))
				require.Equal(t, bank.GetBalance(ctx, sponsor, "uzrn"), freshBank.GetBalance(freshCtx, sponsor, "uzrn"))
				require.Equal(t, bank.GetSupply(ctx, "uzrn"), freshBank.GetSupply(freshCtx, "uzrn"))
				require.Equal(t, ctx.EventManager().Events(), freshCtx.EventManager().Events())
				require.True(t, proto.Equal(k.ExportGenesis(ctx).AccountingState, fresh.ExportGenesis(freshCtx).AccountingState))
				require.True(t, fresh.GetEscrowLocked(freshCtx, "untouched-or-paid").IsZero())
				balanceAfter := freshBank.GetBalance(freshCtx, sponsor, "uzrn")
				freshCtx = freshCtx.WithEventManager(sdk.NewEventManager())
				require.NoError(t, fresh.ProcessRouteBLifecycle(freshCtx))
				require.Equal(t, balanceAfter, freshBank.GetBalance(freshCtx, sponsor, "uzrn"))
				require.Empty(t, freshCtx.EventManager().Events(), "no repeated refund")
				t.Logf("payout=%d refund=%s preserved lifetime=10 epoch0=10 inactive=500; bank supply=%s", payout, refund, freshBank.GetSupply(freshCtx, "uzrn"))
			})
		}
	}
}

func accountingPrimarySnapshot(t *testing.T, store corestore.KVStore) map[string][]byte {
	t.Helper()
	result := map[string][]byte{}

	for _, prefix := range [][]byte{types.TrainingFundEscrowLockedKeyPrefix, types.BootstrapClaimCountPrefix, types.BootstrapEpochCountPrefix} {
		it, err := store.Iterator(prefix, []byte{prefix[0] + 1})
		require.NoError(t, err)
		for ; it.Valid(); it.Next() {
			result[string(it.Key())] = bytes.Clone(it.Value())
		}
		require.NoError(t, it.Close())
	}
	return result
}

func TestGenesisAccountingRejectsBankMismatchBeforeWrites(t *testing.T) {
	for _, balance := range []int64{0, 4999, 5001} {
		t.Run(fmt.Sprint(balance), func(t *testing.T) {
			k, ctx, bank, _ := newKnowledgeOrderFixture(t)
			if balance > 0 {
				coins := sdk.NewCoins(sdk.NewInt64Coin("uzrn", balance))
				require.NoError(t, bank.MintCoins(ctx, orderTestMint, coins))
				require.NoError(t, bank.SendCoinsFromModuleToModule(ctx, orderTestMint, types.TrainingFundModuleName, coins))
			}
			gs := types.DefaultGenesis()
			gs.TrainingFundAllocation = "5000"
			gs.BootstrapFundAllocation = "999" // Would mint if validation ran too late.
			gs.AccountingState = &types.KnowledgeGenesisAccounting{TrainingFundNonEscrowBalance: "5000"}
			beforeSupply := bank.GetSupply(ctx, "uzrn")
			beforeParams, err := k.GetParams(ctx)
			require.NoError(t, err)
			gs.Params.MinVerifiers++
			ctx = ctx.WithEventManager(sdk.NewEventManager())
			require.ErrorContains(t, k.InitGenesis(ctx, gs), "imported bank balance")
			require.Equal(t, beforeSupply, bank.GetSupply(ctx, "uzrn"))
			params, err := k.GetParams(ctx)
			require.NoError(t, err)
			require.True(t, proto.Equal(beforeParams, params))
			require.Empty(t, ctx.EventManager().Events())
		})
	}
}
