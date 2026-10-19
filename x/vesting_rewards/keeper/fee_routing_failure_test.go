package keeper_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/require"

	"github.com/zerone-chain/zerone/x/vesting_rewards/types"
)

// This is a characterization of CURRENT behavior, not the desired atomicity
// contract. Real SDK bank/auth stores hold all balances. A test-only SDK send
// restriction refuses one exact leg before that leg moves money; it is not a
// balance-counting mock or a production-installed restriction. An injected
// dependency-contract failure is not evidence of observed production loss.
func TestRouteFeesActualBankFailureCharacterization(t *testing.T) {
	genesis := types.DefaultGenesis()
	fees := sdk.NewCoins(sdk.NewInt64Coin("atom", 7), sdk.NewInt64Coin("uzrn", 1000003))
	feeAmount := fees.AmountOf("uzrn")
	split := genesis.Params.RevenueSplit
	// RevenueSplit uses millionths; each routed share truncates independently.
	researchShare := feeAmount.MulRaw(int64(split.ResearchBps)).QuoRaw(1_000_000).Int64()
	developmentShare := feeAmount.MulRaw(int64(split.DevelopmentBps)).QuoRaw(1_000_000).Int64()
	collectorRemainder := feeAmount.Int64() - researchShare - developmentShare
	researchTransfer := sdk.NewInt64Coin("uzrn", researchShare).String()
	developmentTransfer := sdk.NewInt64Coin("uzrn", developmentShare).String()
	for _, tc := range []struct {
		name, refuseFrom, refuseTo               string
		collector, escrow, research, development int64
		wantAttempts                             int
	}{
		{"success-control", "", "", collectorRemainder, 0, researchShare, developmentShare, 3},
		{"deposit-failure-retains-escrow", types.ModuleName, types.ResearchFundModuleName, collectorRemainder, researchShare, 0, developmentShare, 3},
		{"escrow-failure-skips-development", authtypes.FeeCollectorName, types.ModuleName, feeAmount.Int64(), 0, 0, 0, 1},
	} {
		for _, nested := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/caller-cache-%t", tc.name, nested), func(t *testing.T) {
				k, ctx, bank, modules := setupFeeRoutingFixture(t, genesis, fees)
				supply := bank.GetSupply(ctx, "uzrn")
				otherSupply := bank.GetSupply(ctx, "atom")

				type attempt struct{ from, to, amount string }
				var attempts []attempt
				fault := errors.New("injected fee routing send refusal")
				faults, deposits := 0, 0
				bank.AppendSendRestriction(func(callCtx context.Context, from, to sdk.AccAddress, amount sdk.Coins) (sdk.AccAddress, error) {
					attempts = append(attempts, attempt{from.String(), to.String(), amount.String()})
					if from.Equals(authtypes.NewModuleAddress(types.ModuleName)) && to.Equals(authtypes.NewModuleAddress(types.ResearchFundModuleName)) {
						deposits++
						require.Equal(t, researchTransfer, amount.String())
						require.Equal(t, amount.String(), bank.GetAllBalances(callCtx, from).String(), "actual escrow transfer must have succeeded before the deposit attempt")
					}
					if tc.refuseFrom != "" && from.Equals(authtypes.NewModuleAddress(tc.refuseFrom)) && to.Equals(authtypes.NewModuleAddress(tc.refuseTo)) {
						faults++
						return to, fault
					}
					return to, nil
				})

				callCtx, commit := ctx, func() {}
				if nested {
					callCtx, commit = ctx.CacheContext()
				}
				err := k.RouteFees(callCtx)
				require.NoError(t, err, "CURRENT behavior swallows either research-leg error")
				require.Len(t, attempts, tc.wantAttempts)
				if tc.refuseFrom == "" {
					require.Zero(t, faults)
				} else {
					require.Equal(t, 1, faults, "the exact fault must fire")
				}
				if tc.refuseTo == types.ModuleName {
					require.Zero(t, deposits)
					require.Equal(t, []attempt{{authtypes.NewModuleAddress(authtypes.FeeCollectorName).String(), authtypes.NewModuleAddress(types.ModuleName).String(), researchTransfer}}, attempts,
						"CURRENT continue skips this coin's development share as well as deposit")
				} else {
					require.Equal(t, 1, deposits)
					message := "development is transferred after a successful research deposit"
					if tc.refuseTo == types.ResearchFundModuleName {
						message = "development is still transferred after a deposit refusal"
					}
					require.Equal(t, attempt{authtypes.NewModuleAddress(authtypes.FeeCollectorName).String(), authtypes.NewModuleAddress(types.DevelopmentFundModuleName).String(), developmentTransfer}, attempts[2], message)
				}
				assertBalances := func(observed sdk.Context) {
					t.Helper()
					for module, want := range map[string]int64{
						authtypes.FeeCollectorName: tc.collector, types.ModuleName: tc.escrow,
						types.ResearchFundModuleName: tc.research, types.DevelopmentFundModuleName: tc.development,
					} {
						require.Equal(t, want, bank.GetBalance(observed, authtypes.NewModuleAddress(module), "uzrn").Amount.Int64(), module)
					}
					require.EqualValues(t, 7, bank.GetBalance(observed, authtypes.NewModuleAddress(authtypes.FeeCollectorName), "atom").Amount.Int64())
					var held sdk.Coins
					for _, module := range modules {
						held = held.Add(bank.GetAllBalances(observed, authtypes.NewModuleAddress(module))...)
					}
					require.Equal(t, fees.String(), held.String(), "coins are retained, not burned or lost from bank supply")
					require.Equal(t, supply, bank.GetSupply(observed, "uzrn"))
					require.Equal(t, otherSupply, bank.GetSupply(observed, "atom"))
				}
				assertBalances(callCtx)
				depositEvents := 0
				for _, event := range callCtx.EventManager().Events() {
					if event.Type == "zerone.vesting_rewards.research_fund_deposit" {
						depositEvents++
					}
				}
				if tc.refuseFrom == "" {
					require.Equal(t, 1, depositEvents)
				} else {
					require.Zero(t, depositEvents)
				}
				if nested {
					require.Equal(t, fees.String(), bank.GetAllBalances(ctx, authtypes.NewModuleAddress(authtypes.FeeCollectorName)).String())
					require.True(t, bank.GetAllBalances(ctx, authtypes.NewModuleAddress(types.ModuleName)).IsZero())
					require.Empty(t, ctx.EventManager().Events())
				}
				commit() // A caller committing the nil result retains the partial route.
				assertBalances(ctx)
				t.Logf("RouteFees=%v faults=%d collector=%s escrow=%s research=%s development=%s supply=%s",
					err, faults,
					bank.GetAllBalances(ctx, authtypes.NewModuleAddress(authtypes.FeeCollectorName)),
					bank.GetAllBalances(ctx, authtypes.NewModuleAddress(types.ModuleName)),
					bank.GetAllBalances(ctx, authtypes.NewModuleAddress(types.ResearchFundModuleName)),
					bank.GetAllBalances(ctx, authtypes.NewModuleAddress(types.DevelopmentFundModuleName)), supply)
			})
		}
	}
}
