package keeper

import (
	"fmt"
	"math/big"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	"github.com/zerone-chain/zerone/x/liquiditypool/types"
)

func RegisterInvariants(ir sdk.InvariantRegistry, k Keeper) {
	ir.RegisterRoute(types.ModuleName, "state-consistency", StateConsistencyInvariant(k))
}

// StateConsistencyInvariant ties liquiditypool's records to bank custody,
// bank LP supply, every secondary index, TWAP history and the monotonic pool
// counter. Underlying module-account surplus is permitted; insolvency is not.
func StateConsistencyInvariant(k Keeper) sdk.Invariant {
	return func(ctx sdk.Context) (formatted string, broken bool) {
		var details strings.Builder
		defer func() {
			if recovered := recover(); recovered != nil {
				fmt.Fprintf(&details, "panic while checking state: %v\n", recovered)
				broken = true
			}
			formatted = sdk.FormatInvariant(types.ModuleName, "state-consistency", details.String())
		}()

		genesis := k.ExportGenesis(ctx)
		if err := genesis.Validate(); err != nil {
			fmt.Fprintf(&details, "structural validation failed: %v\n", err)
			broken = true
		}

		moduleAddress := authtypes.NewModuleAddress(types.ModuleName)
		store := k.storeService.OpenKVStore(ctx)
		liabilities := make(map[string]*big.Int)
		poolsByID := make(map[string]*types.Pool, len(genesis.Pools))
		accumulatorPools := make(map[string]struct{}, len(genesis.TwapAccumulators))
		for _, acc := range genesis.TwapAccumulators {
			accumulatorPools[acc.PoolId] = struct{}{}
		}
		var openCount uint64
		for _, pool := range genesis.Pools {
			poolsByID[pool.PoolId] = pool
			reserveA, errA := types.ParseNonNegativeAmount(pool.ReserveA)
			reserveB, errB := types.ParseNonNegativeAmount(pool.ReserveB)
			supply, errS := types.ParseNonNegativeAmount(pool.LpTokenSupply)
			if errA != nil || errB != nil || errS != nil {
				continue // structural validation already reported the exact field
			}
			addLiability(liabilities, pool.DenomA, reserveA)
			addLiability(liabilities, pool.DenomB, reserveB)

			bankSupply := k.bankKeeper.GetSupply(ctx, pool.LpDenom).Amount.BigInt()
			if bankSupply.Cmp(supply) != 0 {
				fmt.Fprintf(
					&details,
					"pool %s records LP supply %s but bank supply for %s is %s\n",
					pool.PoolId, supply, pool.LpDenom, bankSupply,
				)
				broken = true
			}
			if types.IsOpenPoolStatus(pool.Status) {
				openCount++
				indexed := k.GetPoolByDenoms(ctx, pool.DenomA, pool.DenomB)
				if indexed == nil || indexed.PoolId != pool.PoolId {
					fmt.Fprintf(&details, "pool %s denom index does not round-trip\n", pool.PoolId)
					broken = true
				}
				indexedID, err := store.Get(types.OpenPoolIndexKey(pool.PoolId))
				if err != nil || string(indexedID) != pool.PoolId {
					fmt.Fprintf(&details, "pool %s open-pool index does not round-trip\n", pool.PoolId)
					broken = true
				}
			}
		}
		for denom, liability := range liabilities {
			custody := k.bankKeeper.GetBalance(ctx, moduleAddress, denom).Amount.BigInt()
			if custody.Cmp(liability) < 0 {
				fmt.Fprintf(
					&details,
					"module custody %s%s is below aggregate reserve liability %s%s\n",
					custody, denom, liability, denom,
				)
				broken = true
			}
		}
		queuedPools := make(map[string]struct{})
		k.IterateTWAPGarbageCollection(ctx, func(poolID string) bool {
			queuedPools[poolID] = struct{}{}
			pool, found := poolsByID[poolID]
			if !found || pool.Status != types.PoolStatus_POOL_STATUS_CLOSED {
				fmt.Fprintf(&details, "TWAP cleanup queue references non-closed pool %s\n", poolID)
				broken = true
			}
			if _, live := accumulatorPools[poolID]; live {
				fmt.Fprintf(&details, "TWAP cleanup queue references live accumulator %s\n", poolID)
				broken = true
			}
			return false
		})
		k.IterateTWAPObservations(ctx, "", func(observation *types.TWAPObservation) bool {
			if _, live := accumulatorPools[observation.PoolId]; live {
				return false
			}
			if _, queued := queuedPools[observation.PoolId]; !queued {
				fmt.Fprintf(
					&details,
					"orphan TWAP observation %s/%d is not queued for cleanup\n",
					observation.PoolId, observation.BlockHeight,
				)
				broken = true
			}
			return false
		})

		denomIndexes, err := k.countPrefix(ctx, types.DenomIndexPrefix)
		if err != nil {
			fmt.Fprintf(&details, "failed to count denom indexes: %v\n", err)
			broken = true
		} else if denomIndexes != openCount {
			fmt.Fprintf(&details, "denom index count %d != open pool count %d\n", denomIndexes, openCount)
			broken = true
		}
		openIndexes, err := k.countPrefix(ctx, types.OpenPoolIndexPrefix)
		if err != nil {
			fmt.Fprintf(&details, "failed to count open-pool indexes: %v\n", err)
			broken = true
		} else if openIndexes != openCount {
			fmt.Fprintf(&details, "open-pool index count %d != open pool count %d\n", openIndexes, openCount)
			broken = true
		}

		return "", broken
	}
}

func addLiability(liabilities map[string]*big.Int, denom string, amount *big.Int) {
	if liabilities[denom] == nil {
		liabilities[denom] = new(big.Int)
	}
	liabilities[denom].Add(liabilities[denom], amount)
}

func (k Keeper) countPrefix(ctx sdk.Context, prefix []byte) (uint64, error) {
	store := k.storeService.OpenKVStore(ctx)
	iter, err := store.Iterator(prefix, prefixEndBytes(prefix))
	if err != nil {
		return 0, err
	}
	defer iter.Close()
	var count uint64
	for ; iter.Valid(); iter.Next() {
		count++
	}
	return count, nil
}
