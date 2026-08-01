package keeper

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"

	"cosmossdk.io/core/store"
	"cosmossdk.io/log"
	sdkmath "cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/protobuf/proto"

	"github.com/zerone-chain/zerone/x/liquiditypool/types"
)

// Keeper manages the liquiditypool module's state.
type Keeper struct {
	cdc                codec.Codec
	storeService       store.KVStoreService
	bankKeeper         types.BankKeeper
	authority          string
	activationEvidence ActivationEvidenceReader
}

// ActivationEvidenceReader is the deliberately narrow cross-module surface
// needed to prove an accepted migrated-H1 or native-v5 lineage. A retired
// liquidity parameter is not sufficient evidence: zero was valid legacy state
// and could historically be written through MsgUpdateParams.
type ActivationEvidenceReader interface {
	ReadMigrationMarkerPresenceChecked(context.Context, string) (string, bool, error)
	GetDoneHeight(context.Context, string) (int64, error)
}

// NewKeeper creates a new liquiditypool module Keeper.
func NewKeeper(
	cdc codec.Codec,
	storeService store.KVStoreService,
	bk types.BankKeeper,
	authority string,
	activationEvidence ...ActivationEvidenceReader,
) Keeper {
	if len(activationEvidence) > 1 {
		panic("liquiditypool keeper accepts at most one activation evidence reader")
	}
	var evidence ActivationEvidenceReader
	if len(activationEvidence) == 1 {
		evidence = activationEvidence[0]
	}
	return Keeper{
		cdc:                cdc,
		storeService:       storeService,
		bankKeeper:         bk,
		authority:          authority,
		activationEvidence: evidence,
	}
}

func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

func (k Keeper) GetAuthority() string {
	return k.authority
}

// --- Params ---

func (k Keeper) GetParams(ctx sdk.Context) *types.Params {
	params, err := k.getParamsChecked(ctx)
	if err != nil {
		panic(err)
	}
	return params
}

// getParamsChecked preserves the ordinary genesis-compatible default while
// allowing storage and decoding errors to be handled explicitly by callers.
func (k Keeper) getParamsChecked(ctx context.Context) (*types.Params, error) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := kvStore.Get(types.ParamsKey)
	if err != nil {
		return nil, fmt.Errorf("read liquiditypool params: %w", err)
	}
	if bz == nil {
		return types.DefaultParams(), nil
	}
	var params types.Params
	if err := proto.Unmarshal(bz, &params); err != nil {
		return nil, fmt.Errorf("decode liquiditypool params: %w", err)
	}
	return &params, nil
}

// getStoredParamsChecked is stricter than the ordinary genesis-compatible
// getter: activation evidence must include an actual, decodable, valid Params
// record. A deleted key must not silently become DefaultParams and therefore a
// counterfeit zero sentinel.
func (k Keeper) getStoredParamsChecked(ctx context.Context) (*types.Params, error) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := kvStore.Get(types.ParamsKey)
	if err != nil {
		return nil, fmt.Errorf("read liquiditypool params: %w", err)
	}
	if bz == nil {
		return nil, fmt.Errorf("liquiditypool params key is absent")
	}
	var params types.Params
	if err := proto.Unmarshal(bz, &params); err != nil {
		return nil, fmt.Errorf("decode liquiditypool params: %w", err)
	}
	if err := params.Validate(); err != nil {
		return nil, fmt.Errorf("validate liquiditypool params: %w", err)
	}
	return &params, nil
}

func (k Keeper) SetParams(ctx sdk.Context, params *types.Params) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := proto.Marshal(params)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal params: %v", err))
	}
	if err := kvStore.Set(types.ParamsKey, bz); err != nil {
		panic(fmt.Sprintf("failed to set params: %v", err))
	}
}

// --- Genesis ---

func (k Keeper) InitGenesis(ctx sdk.Context, gs *types.GenesisState) {
	if err := gs.Validate(); err != nil {
		panic(fmt.Sprintf("invalid liquiditypool genesis: %v", err))
	}
	k.SetParams(ctx, gs.Params)
	k.DeleteSecondaryIndexes(ctx)
	for _, pool := range gs.Pools {
		k.SetPool(ctx, pool)
	}
	for _, acc := range gs.TwapAccumulators {
		k.SetTWAPAccumulator(ctx, acc)
	}
	for _, observation := range gs.TwapObservations {
		k.SetTWAPObservation(ctx, observation)
	}

	nextPoolID := gs.NextPoolId
	var derived uint64 = 1
	for _, pool := range gs.Pools {
		id, _ := types.ParsePoolID(pool.PoolId)
		if id >= derived {
			derived = id + 1
		}
	}
	if nextPoolID < derived {
		nextPoolID = derived
	}
	k.SetNextPoolId(ctx, nextPoolID)
}

func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	var pools []*types.Pool
	k.IteratePools(ctx, func(p *types.Pool) bool {
		pools = append(pools, p)
		return false
	})
	var accs []*types.TWAPAccumulator
	k.IterateTWAPAccumulators(ctx, func(a *types.TWAPAccumulator) bool {
		accs = append(accs, a)
		return false
	})
	var observations []*types.TWAPObservation
	accumulatorPools := make(map[string]struct{}, len(accs))
	for _, acc := range accs {
		accumulatorPools[acc.PoolId] = struct{}{}
	}
	k.IterateTWAPObservations(ctx, "", func(observation *types.TWAPObservation) bool {
		// Closed-pool checkpoints queued for bounded garbage collection are
		// not live state and must not leak into an exported v4 genesis.
		if _, live := accumulatorPools[observation.PoolId]; live {
			observations = append(observations, observation)
		}
		return false
	})
	return &types.GenesisState{
		Params:           k.GetParams(ctx),
		Pools:            pools,
		TwapAccumulators: accs,
		NextPoolId:       k.GetNextPoolId(ctx),
		TwapObservations: observations,
	}
}

func (k Keeper) ExportGenesisJSON(ctx sdk.Context) json.RawMessage {
	gs := k.ExportGenesis(ctx)
	bz, err := json.Marshal(gs)
	if err != nil {
		panic("failed to marshal genesis: " + err.Error())
	}
	return bz
}

// --- Pool Counter ---

func (k Keeper) GetNextPoolId(ctx sdk.Context) uint64 {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := kvStore.Get(types.PoolCounterKey)
	if err != nil {
		panic(fmt.Sprintf("failed to read pool counter: %v", err))
	}
	if bz == nil {
		return 1
	}
	counter := new(big.Int).SetBytes(bz)
	if !counter.IsUint64() || counter.Sign() <= 0 {
		panic("invalid liquiditypool next pool ID")
	}
	return counter.Uint64()
}

func (k Keeper) SetNextPoolId(ctx sdk.Context, next uint64) {
	if next == 0 {
		panic("liquiditypool next pool ID must be positive")
	}
	kvStore := k.storeService.OpenKVStore(ctx)
	bz := new(big.Int).SetUint64(next).Bytes()
	if err := kvStore.Set(types.PoolCounterKey, bz); err != nil {
		panic(fmt.Sprintf("failed to set pool counter: %v", err))
	}
}

func (k Keeper) IncrementPoolCounter(ctx sdk.Context) uint64 {
	current := k.GetNextPoolId(ctx)
	if current == ^uint64(0) {
		panic("liquiditypool pool ID space exhausted")
	}
	next := current + 1
	k.SetNextPoolId(ctx, next)
	return current
}

// --- Cross-module price oracle ---

// GetZRNPrice returns the retained-window ZRN arithmetic price in the quote
// denom, scaled by 1e6. It requires a complete configured observation window.
//
// Only quote denoms allowlisted in params.BillingQuoteDenoms are priced:
// any other ZRN pair could poison chain-wide dynamic pricing (a worthless
// counter-denom pool would be reported as the ZRN price). An empty
// allowlist — the default — selects NO pool: fail-closed, callers get the
// same ErrNoPool they get when no pool exists, and fall back to their own
// manual override / fallback tier.
func (k Keeper) GetZRNPrice(ctx sdk.Context, quoteDenom string) (sdkmath.Int, error) {
	allowed := false
	for _, d := range k.GetParams(ctx).BillingQuoteDenoms {
		if d == quoteDenom {
			allowed = true
			break
		}
	}
	if !allowed {
		return sdkmath.ZeroInt(), types.ErrNoPool.Wrap("quote denom not in billing_quote_denoms (oracle fail-closed)")
	}

	pool := k.GetPoolByDenoms(ctx, types.ZRNDenom, quoteDenom)
	if pool == nil {
		return sdkmath.ZeroInt(), types.ErrNoPool
	}
	if pool.Status != types.PoolStatus_POOL_STATUS_ACTIVE {
		return sdkmath.ZeroInt(), types.ErrPoolNotActive
	}
	if k.bankKeeper != nil {
		sendProbe := sdk.NewCoins(
			sdk.NewCoin(pool.DenomA, sdkmath.OneInt()),
			sdk.NewCoin(pool.DenomB, sdkmath.OneInt()),
		)
		if err := k.bankKeeper.IsSendEnabledCoins(ctx, sendProbe...); err != nil {
			return sdkmath.ZeroInt(), types.ErrNoPool.Wrapf(
				"oracle pool contains a send-disabled denom: %v", err,
			)
		}
	}
	reserveA, err := types.ParsePositiveAmount(pool.ReserveA)
	if err != nil {
		return sdkmath.ZeroInt(), types.ErrInvalidPoolState.Wrapf("reserve_a: %v", err)
	}
	reserveB, err := types.ParsePositiveAmount(pool.ReserveB)
	if err != nil {
		return sdkmath.ZeroInt(), types.ErrInvalidPoolState.Wrapf("reserve_b: %v", err)
	}
	minReserve, err := types.ParseNonNegativeAmount(k.GetParams(ctx).MinReserve)
	if err != nil {
		return sdkmath.ZeroInt(), types.ErrInvalidPoolState.Wrapf("min_reserve: %v", err)
	}
	if reserveA.Cmp(minReserve) < 0 || reserveB.Cmp(minReserve) < 0 {
		return sdkmath.ZeroInt(), types.ErrReserveBelowMinimum
	}
	params := k.GetParams(ctx)
	price, windowUsed, err := k.GetTWAP(ctx, pool.PoolId, types.ZRNDenom, params.TwapWindowBlocks)
	if err != nil {
		return sdkmath.ZeroInt(), err
	}
	if windowUsed < params.TwapWindowBlocks {
		return sdkmath.ZeroInt(), types.ErrTWAPWindowUnavailable.Wrapf(
			"pool has %d of %d required blocks", windowUsed, params.TwapWindowBlocks,
		)
	}
	if price.Sign() <= 0 || price.BitLen() > sdkmath.MaxBitLen {
		return sdkmath.ZeroInt(), types.ErrInvalidPoolState.Wrap("TWAP price is zero or out of range")
	}
	return sdkmath.NewIntFromBigInt(price), nil
}

// prefixEndBytes returns the end key for a prefix scan.
func prefixEndBytes(prefix []byte) []byte {
	if len(prefix) == 0 {
		return nil
	}
	end := make([]byte, len(prefix))
	copy(end, prefix)
	for i := len(end) - 1; i >= 0; i-- {
		end[i]++
		if end[i] != 0 {
			return end
		}
	}
	return nil
}
