package keeper

import (
	"fmt"
	"math/big"

	"cosmossdk.io/core/store"
	"cosmossdk.io/log"
	sdkmath "cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/protobuf/proto"

	commontypes "github.com/zerone-chain/zerone/x/common/types"
	"github.com/zerone-chain/zerone/x/vesting_rewards/types"
)

// Keeper manages the vesting_rewards module's state.
type Keeper struct {
	cdc          codec.Codec
	storeService store.KVStoreService

	bankKeeper      types.BankKeeper
	stakingKeeper   types.StakingKeeper
	distrKeeper     types.DistributionKeeper // optional; honors withdraw-address mappings for reward payouts
	knowledgeKeeper types.KnowledgeKeeper    // optional; reports survival metrics and supports legacy reward queries

	authority string
}

// NewKeeper creates a new vesting_rewards module Keeper.
func NewKeeper(
	cdc codec.Codec,
	storeService store.KVStoreService,
	bk types.BankKeeper,
	sk types.StakingKeeper,
	authority string,
) Keeper {
	return Keeper{
		cdc:           cdc,
		storeService:  storeService,
		bankKeeper:    bk,
		stakingKeeper: sk,
		authority:     authority,
	}
}

// prefixEndBytes returns the end key for a prefix scan (exclusive).
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

// SetKnowledgeKeeper wires survived/(survived+disproven) telemetry for
// vesting audits and legacy reward queries. Consensus v2 does not use it to
// drive automatic issuance.
func (k *Keeper) SetKnowledgeKeeper(kk types.KnowledgeKeeper) {
	k.knowledgeKeeper = kk
}

// SetDistributionKeeper wires x/distribution for the historical proposer
// reward API so it honors withdraw-address mappings. Consensus v2 does not
// call that API automatically. Nil-safe when unset.
func (k *Keeper) SetDistributionKeeper(dk types.DistributionKeeper) {
	k.distrKeeper = dk
}

// Logger returns a module-specific logger.
func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

// GetAuthority returns the module authority address.
func (k Keeper) GetAuthority() string {
	return k.authority
}

// GetStakingKeeper returns the staking keeper.
func (k Keeper) GetStakingKeeper() types.StakingKeeper {
	return k.stakingKeeper
}

// ResolveProposerRewardAddress preserves the historical proposer-resolution
// helper for old integrations. Consensus v2 does not pay a proposer reward:
//
//  1. Resolve the consensus address to the validator via x/staking
//     (GetValidatorByConsAddr) and take the OPERATOR account. The raw
//     consensus address is not controlled by any operator key — paying it
//     directly would make a legacy transfer unspendable.
//  2. Honor the operator's x/distribution withdraw-address mapping when the
//     distribution keeper is wired (defaults to the operator itself).
func (k Keeper) ResolveProposerRewardAddress(ctx sdk.Context, consAddr sdk.ConsAddress) (sdk.AccAddress, error) {
	if k.stakingKeeper == nil {
		return nil, fmt.Errorf("staking keeper not wired; cannot resolve proposer %s to operator account", consAddr)
	}

	validator, err := k.stakingKeeper.GetValidatorByConsAddr(ctx, consAddr)
	if err != nil {
		return nil, fmt.Errorf("resolve validator for consensus address %s: %w", consAddr, err)
	}

	valAddr, err := sdk.ValAddressFromBech32(validator.GetOperator())
	if err != nil {
		return nil, fmt.Errorf("invalid operator address %q for consensus address %s: %w", validator.GetOperator(), consAddr, err)
	}

	return k.RewardWithdrawAddress(ctx, sdk.AccAddress(valAddr)), nil
}

// RewardWithdrawAddress preserves the historical x/distribution mapping
// helper. It falls back to the rewardee itself when the
// distribution keeper is not wired or the lookup fails — x/distribution's own
// default is the delegator address, so the fallback matches its semantics.
func (k Keeper) RewardWithdrawAddress(ctx sdk.Context, rewardee sdk.AccAddress) sdk.AccAddress {
	if k.distrKeeper == nil {
		return rewardee
	}
	withdrawAddr, err := k.distrKeeper.GetDelegatorWithdrawAddr(ctx, rewardee)
	if err != nil || withdrawAddr.Empty() {
		return rewardee
	}
	return withdrawAddr
}

// GetParams returns the module parameters.
func (k Keeper) GetParams(ctx sdk.Context) *types.Params {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(types.ParamsKey)
	if err != nil || bz == nil {
		return types.DefaultParams()
	}
	var params types.Params
	if err := proto.Unmarshal(bz, &params); err != nil {
		return types.DefaultParams()
	}
	return &params
}

// SetParams sets the module parameters.
func (k Keeper) SetParams(ctx sdk.Context, params *types.Params) {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := proto.Marshal(params)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal params: %v", err))
	}
	if err := store.Set(types.ParamsKey, bz); err != nil {
		panic(fmt.Sprintf("failed to set params: %v", err))
	}
}

// GetRevenueSplit returns the revenue split from params, falling back to defaults.
func (k Keeper) GetRevenueSplit(ctx sdk.Context) *commontypes.RevenueSplit {
	params := k.GetParams(ctx)
	if params.RevenueSplit != nil {
		return params.RevenueSplit
	}
	return types.DefaultRevenueSplit()
}

// GetProtocolSubSplit returns the protocol sub-split from params, falling back to defaults.
func (k Keeper) GetProtocolSubSplit(ctx sdk.Context) *commontypes.ProtocolSubSplit {
	params := k.GetParams(ctx)
	if params.ProtocolSubSplit != nil {
		return params.ProtocolSubSplit
	}
	return types.DefaultProtocolSubSplit()
}

// isFounderShareActive is retained for the compatibility query surface.
// Consensus version 2 permanently retires the founder auto-split.
func (k Keeper) isFounderShareActive(_ sdk.Context, _ *types.Params) bool {
	return false
}

// GetCategoryConfig returns the release curve config for a vesting category.
func (k Keeper) GetCategoryConfig(ctx sdk.Context, category types.VestingCategoryStr) (*types.CategoryConfig, bool) {
	store := k.storeService.OpenKVStore(ctx)
	key := append(types.CategoryConfigKeyPrefix, []byte(category)...)
	bz, err := store.Get(key)
	if err != nil || bz == nil {
		for _, cfg := range types.DefaultCategoryConfigs() {
			if cfg.Category == string(category) {
				return cfg, true
			}
		}
		return nil, false
	}
	var cfg types.CategoryConfig
	if err := proto.Unmarshal(bz, &cfg); err != nil {
		return nil, false
	}
	return &cfg, true
}

// SetCategoryConfig stores a category config.
func (k Keeper) SetCategoryConfig(ctx sdk.Context, cfg *types.CategoryConfig) {
	store := k.storeService.OpenKVStore(ctx)
	key := append(types.CategoryConfigKeyPrefix, []byte(cfg.Category)...)
	bz, err := proto.Marshal(cfg)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal category config: %v", err))
	}
	if err := store.Set(key, bz); err != nil {
		panic(fmt.Sprintf("failed to set category config: %v", err))
	}
}

// GetTotalMinted returns the shared MintWithCap accounting ledger. It begins
// from imported InitialFundBalance and excludes direct bank/genesis minting.
func (k Keeper) GetTotalMinted(ctx sdk.Context) *big.Int {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(types.TotalMintedKey)
	if err != nil || bz == nil {
		return new(big.Int)
	}
	total := new(big.Int)
	if _, ok := total.SetString(string(bz), 10); !ok {
		return new(big.Int)
	}
	return total
}

// SetTotalMinted stores the shared MintWithCap accounting ledger (in uzrn).
func (k Keeper) SetTotalMinted(ctx sdk.Context, amount *big.Int) {
	store := k.storeService.OpenKVStore(ctx)
	if err := store.Set(types.TotalMintedKey, []byte(amount.String())); err != nil {
		panic(fmt.Sprintf("failed to set total minted: %v", err))
	}
}

// MintWithCap mints new ZRN tokens up to the supply cap (222,222,222 ZRN)
// into the specified module account. The cap is enforced against current
// bank supply (not cumulative totalMinted) so burned tokens free headroom
// for future minting.
//
// This is the wired application's single cap-gated native mint entry point.
// Current callers include:
//
//   - bootstrap and legacy general-pot claims from x/claiming_pot;
//   - External-work attestations: x/substrate_bridge calls MintWithCap with
//     its audit-bounty pool module, then settles the reward to the submitter.
//   - the default-disabled x/knowledge probe pool and x/tokens emission
//     periods when governance activates their nonzero latches.
//
// The pre-v2 transaction-presence proposer mint is permanently retired and
// is not a current caller.
//
// The function exists so the post-genesis cap is enforced once across native
// mint callers; InitChainer independently rejects over-cap genesis supply.
func (k Keeper) MintWithCap(ctx sdk.Context, recipientModule string, amount *big.Int) (*big.Int, error) {
	if amount.Sign() <= 0 {
		return new(big.Int), nil
	}

	maxSupply := new(big.Int)
	maxSupply.SetString(types.MaxSupplyUzrn, 10)

	var currentSupply *big.Int
	if k.bankKeeper != nil {
		supply := k.bankKeeper.GetSupply(ctx, "uzrn")
		currentSupply = supply.Amount.BigInt()
	} else {
		currentSupply = k.GetTotalMinted(ctx)
	}

	remaining := new(big.Int).Sub(maxSupply, currentSupply)
	if remaining.Sign() <= 0 {
		return new(big.Int), nil
	}

	actual := new(big.Int).Set(amount)
	if actual.Cmp(remaining) > 0 {
		actual.Set(remaining)
	}

	if k.bankKeeper != nil {
		mintCoins := sdk.NewCoins(sdk.NewCoin("uzrn", sdkmath.NewIntFromBigInt(actual)))
		if err := k.bankKeeper.MintCoins(ctx, recipientModule, mintCoins); err != nil {
			return nil, fmt.Errorf("mint into module %s: %w", recipientModule, err)
		}
	}

	totalMinted := k.GetTotalMinted(ctx)
	newTotal := new(big.Int).Add(totalMinted, actual)
	k.SetTotalMinted(ctx, newTotal)

	return actual, nil
}

// InitGenesis initializes the module's state from genesis.
func (k Keeper) InitGenesis(ctx sdk.Context, gs *types.GenesisState) {
	if gs.Params != nil {
		k.SetParams(ctx, gs.Params)
	}

	for _, cfg := range gs.CategoryConfigs {
		if cfg != nil {
			k.SetCategoryConfig(ctx, cfg)
		}
	}

	for _, schedule := range gs.VestingSchedules {
		if schedule != nil {
			k.SetVestingSchedule(ctx, schedule)
		}
	}
	// Backward-compatible genesis without explicit indexes derives them from
	// schedule replay. New exports restore the exact live mapping afterward.
	for _, index := range gs.ClaimScheduleIndexes {
		if index != nil {
			k.SetClaimScheduleIndex(ctx, index.ClaimId, index.VestingId)
		}
	}
	for _, record := range gs.ClawbackRecords {
		if record != nil {
			k.SetClawbackRecord(ctx, record)
		}
	}
	for _, distribution := range gs.BlockRewardDistributions {
		if distribution != nil {
			k.SetBlockRewardDistribution(ctx, distribution)
		}
	}

	totalMinted := new(big.Int)
	if gs.Params != nil && gs.Params.InitialFundBalance != "" && gs.Params.InitialFundBalance != "0" {
		if _, ok := totalMinted.SetString(gs.Params.InitialFundBalance, 10); !ok {
			totalMinted = new(big.Int)
		}
	}
	k.SetTotalMinted(ctx, totalMinted)
}

// ExportGenesis exports the module's state.
func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	params := k.GetParams(ctx)
	totalMinted := k.GetTotalMinted(ctx)
	params.InitialFundBalance = totalMinted.String()
	return &types.GenesisState{
		Params:                   params,
		CategoryConfigs:          k.GetAllCategoryConfigs(ctx),
		VestingSchedules:         k.GetAllVestingSchedules(ctx),
		ClawbackRecords:          k.GetAllClawbackRecords(ctx),
		BlockRewardDistributions: k.GetAllBlockRewardDistributions(ctx),
		ClaimScheduleIndexes:     k.GetAllClaimScheduleIndexes(ctx),
	}
}

// GetAllCategoryConfigs returns all stored category configs, falling back to defaults.
func (k Keeper) GetAllCategoryConfigs(ctx sdk.Context) []*types.CategoryConfig {
	store := k.storeService.OpenKVStore(ctx)
	iter, err := store.Iterator(types.CategoryConfigKeyPrefix, prefixEndBytes(types.CategoryConfigKeyPrefix))
	if err != nil {
		return types.DefaultCategoryConfigs()
	}
	defer iter.Close()

	var configs []*types.CategoryConfig
	for ; iter.Valid(); iter.Next() {
		var cfg types.CategoryConfig
		if err := proto.Unmarshal(iter.Value(), &cfg); err != nil {
			continue
		}
		configs = append(configs, &cfg)
	}

	if len(configs) == 0 {
		return types.DefaultCategoryConfigs()
	}
	return configs
}

// GetDecaySchedule returns the epoch-based decay parameters.
func (k Keeper) GetDecaySchedule(ctx sdk.Context) (uint64, uint64, string) {
	params := k.GetParams(ctx)
	return params.BlocksPerRewardEpoch, params.RewardDecayBps, params.FloorReward
}

// GetBlockRewardDistribution retrieves the block reward distribution for a specific height.
func (k Keeper) GetBlockRewardDistribution(ctx sdk.Context, blockHeight uint64) (*types.BlockRewardDistribution, bool) {
	store := k.storeService.OpenKVStore(ctx)
	key := append(types.BlockRewardKeyPrefix, sdk.Uint64ToBigEndian(blockHeight)...)
	bz, err := store.Get(key)
	if err != nil || bz == nil {
		return nil, false
	}
	var dist types.BlockRewardDistribution
	if err := proto.Unmarshal(bz, &dist); err != nil {
		return nil, false
	}
	return &dist, true
}
