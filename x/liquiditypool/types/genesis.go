package types

import (
	"fmt"
	"math/big"
	"sort"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func DefaultParams() *Params {
	return &Params{
		DefaultSwapFeeBps:   3000,          // 0.3%
		MaxPools:            16,            // finite bound on open pools and per-block TWAP work
		MinInitialLiquidity: "10000000000", // 10,000 ZRN in uzrn (uzrn side of the pool)
		TwapWindowBlocks:    1000,          // ~42 minutes at 2.521s blocks
		ProtocolFeeBps:      0,             // retired wire field: all swap fees remain with LPs
		MinReserve:          "1",           // minimum reserve after swap
		BillingQuoteDenoms:  []string{},    // empty = ZRN price oracle disabled (fail-closed)
		AllowedPoolDenoms:   []string{},    // empty = pool creation frozen (fail-closed)
		PoolCreators:        []string{},    // empty = pool creation frozen (fail-closed)
	}
}

func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Params:           DefaultParams(),
		Pools:            []*Pool{},
		TwapAccumulators: []*TWAPAccumulator{},
		NextPoolId:       1,
		TwapObservations: []*TWAPObservation{},
	}
}

func (gs *GenesisState) Validate() error {
	if gs.Params == nil {
		return fmt.Errorf("params cannot be nil")
	}
	if err := gs.Params.Validate(); err != nil {
		return err
	}

	pools := make(map[string]*Pool, len(gs.Pools))
	openPairs := make(map[string]string)
	var maxID uint64
	var openCount uint64

	for i, pool := range gs.Pools {
		if pool == nil {
			return fmt.Errorf("pool %d cannot be nil", i)
		}
		id, err := ParsePoolID(pool.PoolId)
		if err != nil {
			return fmt.Errorf("pool %d: %w", i, err)
		}
		if id > MaxPoolRecordsCap {
			return fmt.Errorf(
				"pool %d ID %d exceeds lifetime record cap %d",
				i, id, MaxPoolRecordsCap,
			)
		}
		if _, duplicate := pools[pool.PoolId]; duplicate {
			return fmt.Errorf("duplicate pool ID %q", pool.PoolId)
		}
		pools[pool.PoolId] = pool
		if id > maxID {
			maxID = id
		}
		if err := validateGenesisPool(pool); err != nil {
			return fmt.Errorf("pool %s: %w", pool.PoolId, err)
		}
		if IsOpenPoolStatus(pool.Status) {
			openCount++
			pairKey := string(DenomPairKey(pool.DenomA, pool.DenomB))
			if existing, duplicate := openPairs[pairKey]; duplicate {
				return fmt.Errorf("open pools %s and %s duplicate denom pair", existing, pool.PoolId)
			}
			openPairs[pairKey] = pool.PoolId
		}
	}
	if openCount > gs.Params.MaxPools {
		return fmt.Errorf("open pool count %d exceeds max_pools %d", openCount, gs.Params.MaxPools)
	}
	if maxID == ^uint64(0) {
		return fmt.Errorf("pool ID space exhausted")
	}
	if gs.NextPoolId != 0 && gs.NextPoolId <= maxID {
		return fmt.Errorf("next_pool_id %d must be greater than maximum pool ID %d", gs.NextPoolId, maxID)
	}
	if gs.NextPoolId > MaxPoolRecordsCap+1 {
		return fmt.Errorf(
			"next_pool_id %d exceeds lifetime record boundary %d",
			gs.NextPoolId, MaxPoolRecordsCap+1,
		)
	}

	accs := make(map[string]*TWAPAccumulator, len(gs.TwapAccumulators))
	accumulatorCumulatives := make(map[string][2]*big.Int, len(gs.TwapAccumulators))
	for i, acc := range gs.TwapAccumulators {
		if acc == nil {
			return fmt.Errorf("TWAP accumulator %d cannot be nil", i)
		}
		pool, exists := pools[acc.PoolId]
		if !exists {
			return fmt.Errorf("TWAP accumulator references unknown pool %q", acc.PoolId)
		}
		if !IsOpenPoolStatus(pool.Status) {
			return fmt.Errorf("closed pool %s cannot retain a TWAP accumulator", pool.PoolId)
		}
		if _, duplicate := accs[acc.PoolId]; duplicate {
			return fmt.Errorf("duplicate TWAP accumulator for pool %s", acc.PoolId)
		}
		if acc.LastBlock < acc.StartBlock {
			return fmt.Errorf("TWAP accumulator %s last_block precedes start_block", acc.PoolId)
		}
		if acc.StartBlock < pool.CreatedAtBlock {
			return fmt.Errorf("TWAP accumulator %s start_block precedes pool creation", acc.PoolId)
		}
		cumA, err := ParseCumulativeAmount(acc.CumPriceAToB)
		if err != nil {
			return fmt.Errorf("TWAP accumulator %s cum_price_a_to_b: %w", acc.PoolId, err)
		}
		cumB, err := ParseCumulativeAmount(acc.CumPriceBToA)
		if err != nil {
			return fmt.Errorf("TWAP accumulator %s cum_price_b_to_a: %w", acc.PoolId, err)
		}
		accs[acc.PoolId] = acc
		accumulatorCumulatives[acc.PoolId] = [2]*big.Int{cumA, cumB}
	}
	for _, pool := range pools {
		if IsOpenPoolStatus(pool.Status) {
			if _, exists := accs[pool.PoolId]; !exists {
				return fmt.Errorf("open pool %s is missing a TWAP accumulator", pool.PoolId)
			}
		}
	}

	observations := make(map[string]struct{}, len(gs.TwapObservations))
	observationCount := make(map[string]uint64)
	type observationPoint struct {
		height uint64
		cumA   *big.Int
		cumB   *big.Int
	}
	observationPoints := make(map[string][]observationPoint, len(accs))
	for i, observation := range gs.TwapObservations {
		if observation == nil {
			return fmt.Errorf("TWAP observation %d cannot be nil", i)
		}
		acc, exists := accs[observation.PoolId]
		if !exists {
			return fmt.Errorf("TWAP observation references pool %q without an accumulator", observation.PoolId)
		}
		if observation.BlockHeight < acc.StartBlock || observation.BlockHeight > acc.LastBlock {
			return fmt.Errorf("TWAP observation %s/%d lies outside accumulator range", observation.PoolId, observation.BlockHeight)
		}
		if acc.LastBlock > gs.Params.TwapWindowBlocks &&
			observation.BlockHeight < acc.LastBlock-gs.Params.TwapWindowBlocks {
			return fmt.Errorf(
				"TWAP observation %s/%d lies outside retained %d-block window",
				observation.PoolId, observation.BlockHeight, gs.Params.TwapWindowBlocks,
			)
		}
		cumA, err := ParseCumulativeAmount(observation.CumPriceAToB)
		if err != nil {
			return fmt.Errorf("TWAP observation %s/%d cum_price_a_to_b: %w", observation.PoolId, observation.BlockHeight, err)
		}
		cumB, err := ParseCumulativeAmount(observation.CumPriceBToA)
		if err != nil {
			return fmt.Errorf("TWAP observation %s/%d cum_price_b_to_a: %w", observation.PoolId, observation.BlockHeight, err)
		}
		accCums := accumulatorCumulatives[observation.PoolId]
		accA, accB := accCums[0], accCums[1]
		if cumA.Cmp(accA) > 0 || cumB.Cmp(accB) > 0 {
			return fmt.Errorf("TWAP observation %s/%d exceeds current cumulative price", observation.PoolId, observation.BlockHeight)
		}
		key := fmt.Sprintf("%s/%d", observation.PoolId, observation.BlockHeight)
		if _, duplicate := observations[key]; duplicate {
			return fmt.Errorf("duplicate TWAP observation %s", key)
		}
		observations[key] = struct{}{}
		observationCount[observation.PoolId]++
		observationPoints[observation.PoolId] = append(
			observationPoints[observation.PoolId],
			observationPoint{
				height: observation.BlockHeight,
				cumA:   cumA,
				cumB:   cumB,
			},
		)
		if observationCount[observation.PoolId] > gs.Params.TwapWindowBlocks+1 {
			return fmt.Errorf(
				"TWAP observation count for %s exceeds retained window bound",
				observation.PoolId,
			)
		}
	}
	for poolID, acc := range accs {
		points := observationPoints[poolID]
		if len(points) == 0 {
			return fmt.Errorf("TWAP accumulator %s has no retained observations", poolID)
		}
		sort.Slice(points, func(i, j int) bool {
			return points[i].height < points[j].height
		})
		for i := 1; i < len(points); i++ {
			if points[i].cumA.Cmp(points[i-1].cumA) < 0 ||
				points[i].cumB.Cmp(points[i-1].cumB) < 0 {
				return fmt.Errorf(
					"TWAP observations for %s decrease between blocks %d and %d",
					poolID, points[i-1].height, points[i].height,
				)
			}
		}
		last := points[len(points)-1]
		accCums := accumulatorCumulatives[poolID]
		if last.height != acc.LastBlock ||
			last.cumA.Cmp(accCums[0]) != 0 ||
			last.cumB.Cmp(accCums[1]) != 0 {
			return fmt.Errorf(
				"TWAP observations for %s do not end at the accumulator checkpoint",
				poolID,
			)
		}
	}
	return nil
}

func (p *Params) Validate() error {
	if p.DefaultSwapFeeBps > MaxSwapFeeBps {
		return fmt.Errorf("default_swap_fee_bps cannot exceed %d (10%%)", MaxSwapFeeBps)
	}
	if p.MaxPools == 0 || p.MaxPools > MaxPoolsCap {
		return fmt.Errorf("max_pools must be between 1 and %d", MaxPoolsCap)
	}
	if p.TwapWindowBlocks == 0 || p.TwapWindowBlocks > MaxTWAPWindowBlocks {
		return fmt.Errorf("twap_window_blocks must be between 1 and %d", MaxTWAPWindowBlocks)
	}
	if p.ProtocolFeeBps != 0 {
		return fmt.Errorf("protocol_fee_bps is retired and must be zero")
	}
	// MinInitialLiquidity and MinReserve are bigint strings consumed via
	// SetString(_, 10) in the msg server, which silently keeps a partial-parse
	// prefix on failure — a malformed gov value ("1e10", "10_000") would
	// quietly collapse the floor. Validate them here (the only gate on the
	// UpdateParams path) so a typo is rejected, not silently applied.
	if _, err := ParsePositiveAmount(p.MinInitialLiquidity); err != nil {
		return fmt.Errorf("min_initial_liquidity: %w", err)
	}
	if _, err := ParseNonNegativeAmount(p.MinReserve); err != nil {
		return fmt.Errorf("min_reserve: %w", err)
	}
	if len(p.BillingQuoteDenoms) > MaxAdmissionEntries {
		return fmt.Errorf("billing_quote_denoms cannot exceed %d entries", MaxAdmissionEntries)
	}
	seen := make(map[string]struct{}, len(p.BillingQuoteDenoms))
	for _, denom := range p.BillingQuoteDenoms {
		if err := sdk.ValidateDenom(denom); err != nil {
			return fmt.Errorf("invalid billing quote denom %q: %w", denom, err)
		}
		if denom == ZRNDenom {
			return fmt.Errorf("billing_quote_denoms cannot contain %s itself", ZRNDenom)
		}
		if _, dup := seen[denom]; dup {
			return fmt.Errorf("duplicate billing quote denom %q", denom)
		}
		seen[denom] = struct{}{}
	}
	if len(p.AllowedPoolDenoms) > MaxAdmissionEntries {
		return fmt.Errorf("allowed_pool_denoms cannot exceed %d entries", MaxAdmissionEntries)
	}
	seen = make(map[string]struct{}, len(p.AllowedPoolDenoms))
	for _, denom := range p.AllowedPoolDenoms {
		if err := sdk.ValidateDenom(denom); err != nil {
			return fmt.Errorf("invalid allowed pool denom %q: %w", denom, err)
		}
		if denom == ZRNDenom {
			return fmt.Errorf("allowed_pool_denoms cannot contain %s itself", ZRNDenom)
		}
		if _, duplicate := seen[denom]; duplicate {
			return fmt.Errorf("duplicate allowed pool denom %q", denom)
		}
		seen[denom] = struct{}{}
	}
	if len(p.PoolCreators) > MaxAdmissionEntries {
		return fmt.Errorf("pool_creators cannot exceed %d entries", MaxAdmissionEntries)
	}
	seen = make(map[string]struct{}, len(p.PoolCreators))
	for _, creator := range p.PoolCreators {
		if _, err := sdk.AccAddressFromBech32(creator); err != nil {
			return fmt.Errorf("invalid pool creator %q: %w", creator, err)
		}
		if _, duplicate := seen[creator]; duplicate {
			return fmt.Errorf("duplicate pool creator %q", creator)
		}
		seen[creator] = struct{}{}
	}
	return nil
}

func validateGenesisPool(pool *Pool) error {
	if err := ValidatePoolDenom(pool.DenomA); err != nil {
		return err
	}
	if err := ValidatePoolDenom(pool.DenomB); err != nil {
		return err
	}
	if pool.DenomA == pool.DenomB {
		return ErrSameDenom
	}
	if pool.DenomA != ZRNDenom && pool.DenomB != ZRNDenom {
		return ErrMissingZRNSide
	}
	if pool.SwapFeeBps > MaxSwapFeeBps {
		return ErrInvalidSwapFee
	}
	if pool.LpDenom != LPDenom(pool.PoolId) {
		return fmt.Errorf("lp_denom must be %q", LPDenom(pool.PoolId))
	}
	if _, err := sdk.AccAddressFromBech32(pool.Creator); err != nil {
		return fmt.Errorf("invalid creator address: %w", err)
	}
	if pool.Locked {
		return fmt.Errorf("transient lock cannot be set in genesis")
	}
	reserveA, err := ParseNonNegativeAmount(pool.ReserveA)
	if err != nil {
		return fmt.Errorf("reserve_a: %w", err)
	}
	reserveB, err := ParseNonNegativeAmount(pool.ReserveB)
	if err != nil {
		return fmt.Errorf("reserve_b: %w", err)
	}
	supply, err := ParseNonNegativeAmount(pool.LpTokenSupply)
	if err != nil {
		return fmt.Errorf("lp_token_supply: %w", err)
	}
	switch pool.Status {
	case PoolStatus_POOL_STATUS_ACTIVE,
		PoolStatus_POOL_STATUS_SWAPS_PAUSED,
		PoolStatus_POOL_STATUS_EXIT_ONLY:
		if reserveA.Sign() <= 0 || reserveB.Sign() <= 0 || supply.Sign() <= 0 {
			return fmt.Errorf("open pool reserves and LP supply must all be positive")
		}
		if pool.ClosedAtBlock != 0 {
			return fmt.Errorf("open pool cannot have closed_at_block")
		}
	case PoolStatus_POOL_STATUS_CLOSED:
		if reserveA.Sign() != 0 || reserveB.Sign() != 0 || supply.Sign() != 0 {
			return fmt.Errorf("closed pool reserves and LP supply must all be zero")
		}
		if pool.ClosedAtBlock == 0 {
			return fmt.Errorf("closed pool must record a positive closed_at_block")
		}
		if pool.ClosedAtBlock < pool.CreatedAtBlock {
			return fmt.Errorf("closed_at_block cannot precede created_at_block")
		}
	default:
		return ErrInvalidPoolStatus
	}
	return nil
}
