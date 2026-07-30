package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/protobuf/proto"

	"github.com/zerone-chain/zerone/x/liquiditypool/types"
)

// Migrator handles in-place store migrations.
type Migrator struct {
	keeper Keeper
}

// NewMigrator returns a new Migrator.
func NewMigrator(keeper Keeper) Migrator {
	return Migrator{keeper: keeper}
}

// Migrate1to2 backfills TWAPAccumulator.StartBlock for accumulators created
// before the field existed. Without it, GetTWAP's divisor (LastBlock -
// StartBlock) would treat StartBlock=0 as "accumulating since genesis" and
// dilute the average for every pool created later. The pool's creation
// height is exact: CreatePool seeds the accumulator in the same block.
func (m Migrator) Migrate1to2(ctx sdk.Context) error {
	m.keeper.IteratePools(ctx, func(pool *types.Pool) bool {
		acc, found := m.keeper.GetTWAPAccumulator(ctx, pool.PoolId)
		if !found || acc.StartBlock != 0 {
			return false
		}
		acc.StartBlock = pool.CreatedAtBlock
		m.keeper.SetTWAPAccumulator(ctx, acc)
		return false
	})
	return nil
}

// Migrate2to3 introduces Params.BillingQuoteDenoms (the ZRN price oracle's
// quote-denom allowlist). Existing state decodes the absent field as an
// empty list already — which IS the intended default: the oracle is
// fail-closed until governance allowlists a stable quote denom. The
// migration re-persists params explicitly so the stored bytes carry the
// v3 shape and the fail-closed default is a deliberate write, not an
// accident of proto3 decoding.
func (m Migrator) Migrate2to3(ctx sdk.Context) error {
	params := m.keeper.GetParams(ctx)
	if params.BillingQuoteDenoms == nil {
		params.BillingQuoteDenoms = []string{}
	}
	m.keeper.SetParams(ctx, params)
	return nil
}

// Migrate3to4 establishes the v4 lifecycle and bounded-index invariants.
// Positive legacy pools become EXIT_ONLY so the first generic RunMigrations
// cannot silently activate trading before the named readiness checkpoint. A
// legacy 0/0/0 pool becomes an immutable CLOSED tombstone. Partial-zero or
// otherwise inconsistent state fails the upgrade rather than guessing
// ownership.
//
// Legacy TWAP accumulators cannot answer bounded-window queries because they
// have no historical checkpoints. The migration deliberately retires that
// history and starts a truthful observation window at the upgrade height.
func (m Migrator) Migrate3to4(ctx sdk.Context) error {
	params := m.keeper.GetParams(ctx)
	if params.MaxPools == 0 {
		params.MaxPools = types.DefaultParams().MaxPools
	} else if params.MaxPools > types.MaxPoolsCap {
		// v3 admitted unbounded/custom limits. v4 preserves existing pools up
		// to its hard cap, but never carries a policy value beyond that cap.
		params.MaxPools = types.MaxPoolsCap
	}
	if params.AllowedPoolDenoms == nil {
		params.AllowedPoolDenoms = []string{}
	}
	if params.PoolCreators == nil {
		params.PoolCreators = []string{}
	}
	if params.BillingQuoteDenoms == nil {
		params.BillingQuoteDenoms = []string{}
	}

	var pools []*types.Pool
	var maxPoolID uint64
	var openCount uint64
	m.keeper.IteratePools(ctx, func(pool *types.Pool) bool {
		pools = append(pools, proto.Clone(pool).(*types.Pool))
		return false
	})
	for _, pool := range pools {
		id, err := types.ParsePoolID(pool.PoolId)
		if err != nil {
			return fmt.Errorf("invalid legacy pool ID %q: %w", pool.PoolId, err)
		}
		if id > maxPoolID {
			maxPoolID = id
		}
		if id > types.MaxPoolRecordsCap {
			return fmt.Errorf(
				"legacy pool ID %d exceeds lifetime record cap %d",
				id, types.MaxPoolRecordsCap,
			)
		}
		reserveA, err := types.ParseNonNegativeAmount(pool.ReserveA)
		if err != nil {
			return fmt.Errorf("pool %s reserve_a: %w", pool.PoolId, err)
		}
		reserveB, err := types.ParseNonNegativeAmount(pool.ReserveB)
		if err != nil {
			return fmt.Errorf("pool %s reserve_b: %w", pool.PoolId, err)
		}
		supply, err := types.ParseNonNegativeAmount(pool.LpTokenSupply)
		if err != nil {
			return fmt.Errorf("pool %s lp_token_supply: %w", pool.PoolId, err)
		}
		allPositive := reserveA.Sign() > 0 && reserveB.Sign() > 0 && supply.Sign() > 0
		allZero := reserveA.Sign() == 0 && reserveB.Sign() == 0 && supply.Sign() == 0
		switch {
		case allPositive:
			pool.Status = types.PoolStatus_POOL_STATUS_EXIT_ONLY
			pool.ClosedAtBlock = 0
			openCount++
		case allZero:
			pool.Status = types.PoolStatus_POOL_STATUS_CLOSED
			if ctx.BlockHeight() < 0 {
				return fmt.Errorf("cannot close legacy pool at negative upgrade height")
			}
			pool.ClosedAtBlock = uint64(ctx.BlockHeight())
		default:
			return fmt.Errorf(
				"pool %s has inconsistent reserve/supply zero state (%s/%s/%s)",
				pool.PoolId, pool.ReserveA, pool.ReserveB, pool.LpTokenSupply,
			)
		}
	}
	if openCount > types.MaxPoolsCap {
		return fmt.Errorf("legacy open pool count %d exceeds hard cap %d", openCount, types.MaxPoolsCap)
	}
	if params.MaxPools < openCount {
		params.MaxPools = openCount
	}
	if err := params.Validate(); err != nil {
		return fmt.Errorf("v4 params migration: %w", err)
	}
	m.keeper.SetParams(ctx, params)

	m.keeper.DeleteSecondaryIndexes(ctx)
	for _, pool := range pools {
		m.keeper.DeleteTWAPHistory(ctx, pool.PoolId)
		m.keeper.SetPool(ctx, pool)
		if types.IsOpenPoolStatus(pool.Status) {
			m.keeper.UpdateTWAPAccumulator(ctx, pool)
		}
	}
	nextPoolID := m.keeper.GetNextPoolId(ctx)
	if maxPoolID == ^uint64(0) {
		return fmt.Errorf("pool ID space exhausted")
	}
	derived := maxPoolID + 1
	if nextPoolID < derived {
		nextPoolID = derived
	}
	m.keeper.SetNextPoolId(ctx, nextPoolID)

	if err := m.keeper.ExportGenesis(ctx).Validate(); err != nil {
		return fmt.Errorf("v4 post-migration state validation failed: %w", err)
	}
	if report, broken := StateConsistencyInvariant(m.keeper)(ctx); broken {
		return fmt.Errorf("v4 post-migration bank/state audit failed: %s", report)
	}
	return nil
}
