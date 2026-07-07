package keeper

import (
	"context"
	"encoding/binary"
	"fmt"
	"math/big"
	"strings"

	"cosmossdk.io/core/store"
	"cosmossdk.io/log"
	"cosmossdk.io/store/prefix"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"google.golang.org/protobuf/proto"

	"github.com/zerone-chain/zerone/x/claiming_pot/types"
)

type Keeper struct {
	storeService         store.KVStoreService
	cdc                  codec.BinaryCodec
	authority            string
	stakingKeeper        types.StakingKeeper
	authKeeper           types.AuthKeeper
	bankKeeper           types.BankKeeper
	vestingRewardsKeeper types.VestingRewardsKeeper
}

func NewKeeper(
	storeService store.KVStoreService,
	cdc codec.BinaryCodec,
	authority string,
	sk types.StakingKeeper,
	ak types.AuthKeeper,
	bk types.BankKeeper,
	vrk types.VestingRewardsKeeper,
) Keeper {
	return Keeper{
		storeService:         storeService,
		cdc:                  cdc,
		authority:            authority,
		stakingKeeper:        sk,
		authKeeper:           ak,
		bankKeeper:           bk,
		vestingRewardsKeeper: vrk,
	}
}

func (k Keeper) Logger(ctx context.Context) log.Logger {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	return sdkCtx.Logger().With("module", "x/"+types.ModuleName)
}

func (k Keeper) GetAuthority() string {
	return k.authority
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

// ---------- Params ----------

func (k Keeper) SetParams(ctx context.Context, params *types.Params) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := proto.Marshal(params)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal params: %v", err))
	}
	_ = kvStore.Set(types.ParamsKey, bz)
}

func (k Keeper) GetParams(ctx context.Context) *types.Params {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := kvStore.Get(types.ParamsKey)
	if err != nil || bz == nil {
		return types.DefaultParams()
	}
	var params types.Params
	if err := proto.Unmarshal(bz, &params); err != nil {
		return types.DefaultParams()
	}
	return &params
}

// ---------- Counter ----------

func (k Keeper) GetNextPotID(ctx context.Context) uint64 {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := kvStore.Get(types.PotCounterKey)
	if err != nil || bz == nil {
		bz = make([]byte, 8)
	}
	counter := binary.BigEndian.Uint64(bz)
	counter++
	newBz := make([]byte, 8)
	binary.BigEndian.PutUint64(newBz, counter)
	_ = kvStore.Set(types.PotCounterKey, newBz)
	return counter
}

// ---------- Bootstrap emission accounting ----------

// GetBootstrapMintedEntries returns the number of bootstrap entries ever
// created (genesis + admissions). Monotonic: pots are never deleted and
// DEPLETED pots stay in state, so this never decreases.
func (k Keeper) GetBootstrapMintedEntries(ctx context.Context) uint64 {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := kvStore.Get(types.BootstrapMintedEntriesKey)
	if err != nil || len(bz) != 8 {
		return 0
	}
	return binary.BigEndian.Uint64(bz)
}

func (k Keeper) SetBootstrapMintedEntries(ctx context.Context, count uint64) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, count)
	_ = kvStore.Set(types.BootstrapMintedEntriesKey, bz)
}

// GetBootstrapWindowCount returns the registrar admission count for the
// given window index. A stored record for a different window reads as 0 —
// the window has rolled and the counter implicitly resets.
func (k Keeper) GetBootstrapWindowCount(ctx context.Context, windowIndex uint64) uint64 {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := kvStore.Get(types.BootstrapAdmissionWindowKey)
	if err != nil || len(bz) != 16 {
		return 0
	}
	if binary.BigEndian.Uint64(bz[:8]) != windowIndex {
		return 0
	}
	return binary.BigEndian.Uint64(bz[8:])
}

func (k Keeper) SetBootstrapWindowCount(ctx context.Context, windowIndex, count uint64) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz := make([]byte, 16)
	binary.BigEndian.PutUint64(bz[:8], windowIndex)
	binary.BigEndian.PutUint64(bz[8:], count)
	_ = kvStore.Set(types.BootstrapAdmissionWindowKey, bz)
}

// ---------- ClaimingPot CRUD ----------

func (k Keeper) SetPot(ctx context.Context, pot *types.ClaimingPot) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := proto.Marshal(pot)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal pot: %v", err))
	}
	_ = kvStore.Set(types.PotKey(pot.Id), bz)

	// Active index
	if pot.Status == types.PotStatus_POT_STATUS_ACTIVE {
		_ = kvStore.Set(types.ActivePotKey(pot.Id), []byte{1})
	} else {
		_ = kvStore.Delete(types.ActivePotKey(pot.Id))
	}
}

func (k Keeper) GetPot(ctx context.Context, id string) (*types.ClaimingPot, bool) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := kvStore.Get(types.PotKey(id))
	if err != nil || bz == nil {
		return nil, false
	}
	var pot types.ClaimingPot
	if err := proto.Unmarshal(bz, &pot); err != nil {
		return nil, false
	}
	return &pot, true
}

func (k Keeper) GetAllPots(ctx context.Context) []*types.ClaimingPot {
	var pots []*types.ClaimingPot
	k.IteratePots(ctx, func(p *types.ClaimingPot) bool {
		pots = append(pots, p)
		return false
	})
	return pots
}

func (k Keeper) IteratePots(ctx context.Context, cb func(*types.ClaimingPot) bool) {
	kvStore := k.storeService.OpenKVStore(ctx)
	iter, err := kvStore.Iterator(types.PotKeyPrefix, prefixEndBytes(types.PotKeyPrefix))
	if err != nil {
		return
	}
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		var pot types.ClaimingPot
		if err := proto.Unmarshal(iter.Value(), &pot); err != nil {
			continue
		}
		if cb(&pot) {
			break
		}
	}
}

// GetPotsPaginated returns one page of pots using standard SDK pagination
// (nil pageReq → default limit). GetAllPots stays unpaginated for genesis
// export, which must be exhaustive; queries go through this instead.
func (k Keeper) GetPotsPaginated(ctx context.Context, pageReq *query.PageRequest) ([]*types.ClaimingPot, *query.PageResponse, error) {
	adapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	potStore := prefix.NewStore(adapter, types.PotKeyPrefix)

	var pots []*types.ClaimingPot
	pageRes, err := query.Paginate(potStore, pageReq, func(_, value []byte) error {
		var pot types.ClaimingPot
		if err := proto.Unmarshal(value, &pot); err != nil {
			return err
		}
		pots = append(pots, &pot)
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	return pots, pageRes, nil
}

// IterateActivePotIDs walks the ActivePotPrefix index only — O(active
// pots), never touching (or unmarshalling) DEPLETED/EXPIRED pots. The
// callback returns true to stop.
func (k Keeper) IterateActivePotIDs(ctx context.Context, cb func(id string) bool) {
	kvStore := k.storeService.OpenKVStore(ctx)
	iter, err := kvStore.Iterator(types.ActivePotPrefix, prefixEndBytes(types.ActivePotPrefix))
	if err != nil {
		return
	}
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		id := string(iter.Key()[len(types.ActivePotPrefix):])
		if cb(id) {
			break
		}
	}
}

func (k Keeper) CountActivePots(ctx context.Context) int {
	kvStore := k.storeService.OpenKVStore(ctx)
	iter, err := kvStore.Iterator(types.ActivePotPrefix, prefixEndBytes(types.ActivePotPrefix))
	if err != nil {
		return 0
	}
	defer iter.Close()
	count := 0
	for ; iter.Valid(); iter.Next() {
		count++
	}
	return count
}

// ---------- Claim CRUD ----------

func (k Keeper) SetClaim(ctx context.Context, claim *types.Claim) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := proto.Marshal(claim)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal claim: %v", err))
	}
	_ = kvStore.Set(types.ClaimKey(claim.PotId, claim.Claimant), bz)
}

func (k Keeper) GetClaim(ctx context.Context, potID, claimant string) (*types.Claim, bool) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := kvStore.Get(types.ClaimKey(potID, claimant))
	if err != nil || bz == nil {
		return nil, false
	}
	var claim types.Claim
	if err := proto.Unmarshal(bz, &claim); err != nil {
		return nil, false
	}
	return &claim, true
}

func (k Keeper) GetClaimsByPot(ctx context.Context, potID string) []*types.Claim {
	kvStore := k.storeService.OpenKVStore(ctx)
	prefix := types.ClaimByPotPrefix(potID)
	iter, err := kvStore.Iterator(prefix, prefixEndBytes(prefix))
	if err != nil {
		return nil
	}
	defer iter.Close()

	var claims []*types.Claim
	for ; iter.Valid(); iter.Next() {
		var claim types.Claim
		if err := proto.Unmarshal(iter.Value(), &claim); err != nil {
			continue
		}
		claims = append(claims, &claim)
	}
	return claims
}

func (k Keeper) GetAllClaims(ctx context.Context) []*types.Claim {
	kvStore := k.storeService.OpenKVStore(ctx)
	iter, err := kvStore.Iterator(types.ClaimKeyPrefix, prefixEndBytes(types.ClaimKeyPrefix))
	if err != nil {
		return nil
	}
	defer iter.Close()

	var claims []*types.Claim
	for ; iter.Valid(); iter.Next() {
		var claim types.Claim
		if err := proto.Unmarshal(iter.Value(), &claim); err != nil {
			continue
		}
		claims = append(claims, &claim)
	}
	return claims
}

// ---------- Vesting Math ----------

// CalculateClaimable computes the vested-but-unclaimed amount for a pot at a given block.
func CalculateClaimable(pot *types.ClaimingPot, currentBlock uint64) *big.Int {
	if pot.Schedule == nil {
		return new(big.Int)
	}

	schedule := pot.Schedule

	// Before start: nothing vested
	if currentBlock < schedule.StartBlock {
		return new(big.Int)
	}

	// Before cliff: nothing vested
	cliffBlock := schedule.StartBlock + schedule.CliffBlocks
	if currentBlock < cliffBlock {
		return new(big.Int)
	}

	totalAmount := new(big.Int)
	totalAmount.SetString(pot.TotalAmount, 10)
	if totalAmount.Sign() <= 0 {
		return new(big.Int)
	}

	claimedAmount := new(big.Int)
	claimedAmount.SetString(pot.ClaimedAmount, 10)

	// After end: fully vested
	if currentBlock >= schedule.EndBlock {
		remaining := new(big.Int).Sub(totalAmount, claimedAmount)
		if remaining.Sign() < 0 {
			return new(big.Int)
		}
		return remaining
	}

	// Linear vesting between cliff and end
	elapsed := currentBlock - schedule.StartBlock
	totalDuration := schedule.EndBlock - schedule.StartBlock
	if totalDuration == 0 {
		return new(big.Int)
	}

	// vested = totalAmount * elapsed / totalDuration
	vested := new(big.Int).Mul(totalAmount, new(big.Int).SetUint64(elapsed))
	vested.Div(vested, new(big.Int).SetUint64(totalDuration))

	// claimable = vested - already claimed
	claimable := new(big.Int).Sub(vested, claimedAmount)
	if claimable.Sign() < 0 {
		return new(big.Int)
	}
	return claimable
}

// ---------- BeginBlocker: Pot Expiry ----------

// ProcessPotExpiry checks active pots and marks expired ones.
//
// Bootstrap pots (ID prefix BootstrapPotIDPrefix) are participation seeds
// — they never auto-expire. Their only terminal state is DEPLETED, set by
// successful claim. This preserves the doctrine of commitment 20
// (issuance follows participation): a participation seed must remain
// claimable for the participant who shows up, regardless of how late.
// Without this carve-out, the genesis bootstrap pathway is structurally
// unclaimable — at the start of block 1, BeginBlocker would flip every
// bootstrap pot to EXPIRED before any MsgClaim tx in block 1 could run.
// Scale: this runs every block, so it iterates the ActivePotPrefix index
// ONLY — cost is O(active pots), independent of how many DEPLETED/EXPIRED
// pots have accumulated. Terminal pots are deliberately KEPT in state:
// claim idempotency (and AddBootstrapEntry's skip-if-exists) depends on
// their presence. Bootstrap pots are skipped by ID prefix before any store
// read of the pot record itself.
func (k Keeper) ProcessPotExpiry(ctx context.Context, currentBlock uint64) {
	// Collect first, mutate after: SetPot deletes ActivePotPrefix entries,
	// and deleting under a live iterator over that same prefix is unsafe.
	var expired []*types.ClaimingPot
	k.IterateActivePotIDs(ctx, func(id string) bool {
		if strings.HasPrefix(id, types.BootstrapPotIDPrefix) {
			return false // participation seeds never auto-expire (see doctrine above)
		}
		pot, found := k.GetPot(ctx, id)
		if !found {
			return false // tolerate a stale index entry
		}
		if pot.Status == types.PotStatus_POT_STATUS_ACTIVE && pot.Schedule != nil && currentBlock >= pot.Schedule.EndBlock {
			expired = append(expired, pot)
		}
		return false
	})
	for _, pot := range expired {
		pot.Status = types.PotStatus_POT_STATUS_EXPIRED
		k.SetPot(ctx, pot) // also removes the pot from the active index
	}
}

// ---------- Genesis ----------

func (k Keeper) InitGenesis(ctx context.Context, genState *types.GenesisState) {
	if genState.Params != nil {
		k.SetParams(ctx, genState.Params)
	}
	// Genesis bootstrap pots consume the same lifetime emission budget as
	// post-genesis admissions, so seed the minted-entries counter from
	// them. The counter is fully derivable (pots are never deleted), which
	// keeps export → import round-trips consistent without a genesis field.
	bootstrapEntries := uint64(0)
	for _, pot := range genState.Pots {
		k.SetPot(ctx, pot)
		if strings.HasPrefix(pot.Id, types.BootstrapPotIDPrefix) {
			bootstrapEntries++
		}
	}
	if bootstrapEntries > 0 {
		k.SetBootstrapMintedEntries(ctx, bootstrapEntries)
	}
	for _, claim := range genState.Claims {
		k.SetClaim(ctx, claim)
	}
}

func (k Keeper) ExportGenesis(ctx context.Context) *types.GenesisState {
	params := k.GetParams(ctx)
	pots := k.GetAllPots(ctx)
	claims := k.GetAllClaims(ctx)
	return &types.GenesisState{
		Params: params,
		Pots:   pots,
		Claims: claims,
	}
}
