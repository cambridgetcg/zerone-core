package keeper

import (
	"context"
	"math/big"

	"cosmossdk.io/core/store"
	"cosmossdk.io/log"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/x/emergency/types"
)

// Keeper manages the emergency module's state.
type Keeper struct {
	storeService  store.KVStoreService
	cdc           codec.BinaryCodec
	authority     string
	stakingKeeper types.StakingKeeper
}

// NewKeeper creates a new emergency module Keeper.
func NewKeeper(
	storeService store.KVStoreService,
	cdc codec.BinaryCodec,
	authority string,
	stakingKeeper types.StakingKeeper,
) Keeper {
	return Keeper{
		storeService:  storeService,
		cdc:           cdc,
		authority:     authority,
		stakingKeeper: stakingKeeper,
	}
}

// prefixEndBytes returns the end key for prefix iteration (exclusive).
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

// Logger returns a module-scoped logger.
func (k Keeper) Logger(ctx context.Context) log.Logger {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	return sdkCtx.Logger().With("module", "x/"+types.ModuleName)
}

// GetAuthority returns the module authority address.
func (k Keeper) GetAuthority() string {
	return k.authority
}

// IsGuardian checks if an address is a Guardian-tier validator or genesis council member.
func (k Keeper) IsGuardian(ctx context.Context, operatorAddr string) bool {
	val, found := k.stakingKeeper.GetValidator(ctx, operatorAddr)
	if found && val.Tier == types.TierGuardian && val.IsActive {
		return true
	}
	// H-5: Check genesis emergency council.
	params := k.GetParams(ctx)
	if k.isCouncilActive(ctx, params) {
		return k.isCouncilMember(operatorAddr, params)
	}
	return false
}

// GetGuardianStake returns the total effective stake of all active Guardians.
func (k Keeper) GetGuardianStake(ctx context.Context) *big.Int {
	total := new(big.Int)
	guardians, err := k.stakingKeeper.GetGuardianValidators(ctx)
	if err != nil {
		return total
	}
	for _, v := range guardians {
		if v.IsActive {
			stake, ok := new(big.Int).SetString(v.TotalStake, 10)
			if ok {
				total.Add(total, stake)
			}
		}
	}
	// H-5: Add council virtual stake during bootstrap.
	params := k.GetParams(ctx)
	if k.isCouncilActive(ctx, params) && len(params.GenesisCouncil) > 0 {
		virtualPerMember, ok := new(big.Int).SetString(params.CouncilVirtualStake, 10)
		if ok {
			councilTotal := new(big.Int).Mul(virtualPerMember, big.NewInt(int64(len(params.GenesisCouncil))))
			total.Add(total, councilTotal)
		}
	}
	return total
}

// GetGuardianEffectiveStake returns a single guardian's effective stake.
func (k Keeper) GetGuardianEffectiveStake(ctx context.Context, operatorAddr string) *big.Int {
	val, found := k.stakingKeeper.GetValidator(ctx, operatorAddr)
	if found && val.Tier == types.TierGuardian && val.IsActive {
		stake, ok := new(big.Int).SetString(val.TotalStake, 10)
		if ok {
			return stake
		}
	}
	// H-5: Council member virtual stake.
	params := k.GetParams(ctx)
	if k.isCouncilActive(ctx, params) && k.isCouncilMember(operatorAddr, params) {
		virtualStake, ok := new(big.Int).SetString(params.CouncilVirtualStake, 10)
		if ok {
			return virtualStake
		}
	}
	return new(big.Int)
}

// isCouncilMember checks if an address is in the genesis emergency council.
func (k Keeper) isCouncilMember(operatorAddr string, params *types.Params) bool {
	for _, member := range params.GenesisCouncil {
		if member == operatorAddr {
			return true
		}
	}
	return false
}

// isCouncilActive checks if the genesis council is still active.
func (k Keeper) isCouncilActive(ctx context.Context, params *types.Params) bool {
	if params.CouncilExpiryBlock == 0 || len(params.GenesisCouncil) == 0 {
		return false
	}
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	return uint64(sdkCtx.BlockHeight()) < params.CouncilExpiryBlock
}

// --- Genesis ---

// InitGenesis initializes the module's state from genesis.
func (k Keeper) InitGenesis(ctx context.Context, genState *types.GenesisState) {
	if genState.Params != nil {
		k.SetParams(ctx, genState.Params)
	}
	k.SetEmergencyStatus(ctx, types.EmergencyStatus(genState.Status))
	for _, ceremony := range genState.Ceremonies {
		if ceremony == nil {
			continue
		}
		if err := k.SetCeremony(ctx, ceremony); err != nil {
			panic("failed to init genesis ceremony: " + err.Error())
		}
	}
	for _, entry := range genState.AuditLog {
		if entry == nil {
			continue
		}
		k.AddAuditEntry(ctx, entry)
	}
}

// ExportGenesis exports the module's state.
func (k Keeper) ExportGenesis(ctx context.Context) *types.GenesisState {
	ceremonies := k.GetAllCeremonies(ctx)
	auditLog := k.GetAuditLog(ctx)
	params := k.GetParams(ctx)
	return &types.GenesisState{
		Params:     params,
		Status:     string(k.GetEmergencyStatus(ctx)),
		Ceremonies: ceremonies,
		AuditLog:   auditLog,
	}
}
