package keeper

import (
	"context"
	"encoding/binary"
	"fmt"
	"math"
	"math/big"

	"cosmossdk.io/core/store"
	"cosmossdk.io/log"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"google.golang.org/protobuf/proto"

	"github.com/zerone-chain/zerone/x/sponsorship/types"
)

type Keeper struct {
	storeService    store.KVStoreService
	cdc             codec.BinaryCodec
	bankKeeper      types.BankKeeper
	knowledgeKeeper types.KnowledgeKeeper
}

func NewKeeper(
	storeService store.KVStoreService,
	cdc codec.BinaryCodec,
	bk types.BankKeeper,
	kk types.KnowledgeKeeper,
) Keeper {
	return Keeper{
		storeService:    storeService,
		cdc:             cdc,
		bankKeeper:      bk,
		knowledgeKeeper: kk,
	}
}

func (k Keeper) Logger(ctx context.Context) log.Logger {
	return sdk.UnwrapSDKContext(ctx).Logger().With("module", "x/"+types.ModuleName)
}

// ---------- Params ----------

func (k Keeper) SetParams(ctx context.Context, params *types.Params) {
	kv := k.storeService.OpenKVStore(ctx)
	bz, err := proto.Marshal(params)
	if err != nil {
		panic(fmt.Sprintf("marshal params: %v", err))
	}
	_ = kv.Set(types.ParamsKey, bz)
}

func (k Keeper) GetParams(ctx context.Context) *types.Params {
	kv := k.storeService.OpenKVStore(ctx)
	bz, err := kv.Get(types.ParamsKey)
	if err != nil || bz == nil {
		return types.DefaultParams()
	}
	var p types.Params
	if err := proto.Unmarshal(bz, &p); err != nil {
		return types.DefaultParams()
	}
	return &p
}

// ---------- Counter ----------

func (k Keeper) nextBountyID(ctx context.Context) (uint64, error) {
	kv := k.storeService.OpenKVStore(ctx)
	bz, err := kv.Get(types.BountyCounterKey)
	if err != nil || bz == nil {
		bz = make([]byte, 8)
	}
	n := binary.BigEndian.Uint64(bz)
	if n >= math.MaxUint64-1 {
		return 0, fmt.Errorf("bounty id space exhausted")
	}
	n++
	newBz := make([]byte, 8)
	binary.BigEndian.PutUint64(newBz, n)
	_ = kv.Set(types.BountyCounterKey, newBz)
	return n, nil
}

func (k Keeper) canAllocateBountyID(ctx context.Context) error {
	kv := k.storeService.OpenKVStore(ctx)
	bz, err := kv.Get(types.BountyCounterKey)
	if err != nil {
		return err
	}
	if bz != nil && binary.BigEndian.Uint64(bz) >= math.MaxUint64-1 {
		return fmt.Errorf("bounty id space exhausted")
	}
	return nil
}

// ---------- BountyOrder CRUD ----------

func (k Keeper) SetBountyOrder(ctx context.Context, o *types.BountyOrder) {
	kv := k.storeService.OpenKVStore(ctx)
	bz, err := proto.Marshal(o)
	if err != nil {
		panic(fmt.Sprintf("marshal bounty: %v", err))
	}
	_ = kv.Set(types.BountyOrderKey(o.Id), bz)
}

func (k Keeper) GetBountyOrder(ctx context.Context, id string) (*types.BountyOrder, bool) {
	kv := k.storeService.OpenKVStore(ctx)
	bz, err := kv.Get(types.BountyOrderKey(id))
	if err != nil || bz == nil {
		return nil, false
	}
	var o types.BountyOrder
	if err := proto.Unmarshal(bz, &o); err != nil {
		return nil, false
	}
	return &o, true
}

func (k Keeper) IterateBountyOrders(ctx context.Context, cb func(*types.BountyOrder) bool) {
	kv := k.storeService.OpenKVStore(ctx)
	iter, err := kv.Iterator(types.BountyOrderKeyPrefix, prefixEndBytes(types.BountyOrderKeyPrefix))
	if err != nil {
		return
	}
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		var o types.BountyOrder
		if err := proto.Unmarshal(iter.Value(), &o); err != nil {
			continue
		}
		if cb(&o) {
			break
		}
	}
}

func (k Keeper) GetAllBountyOrders(ctx context.Context) []*types.BountyOrder {
	var out []*types.BountyOrder
	k.IterateBountyOrders(ctx, func(o *types.BountyOrder) bool {
		out = append(out, o)
		return false
	})
	return out
}

func (k Keeper) CountActiveBountiesBySponsor(ctx context.Context, sponsor string) uint32 {
	canonicalSponsor, err := types.CanonicalAccountAddress(sponsor)
	if err != nil {
		return 0
	}
	kv := k.storeService.OpenKVStore(ctx)
	prefix := types.ActiveSponsorIndexPrefix(canonicalSponsor)
	iter, err := kv.Iterator(prefix, prefixEndBytes(prefix))
	if err != nil {
		return 0
	}
	defer iter.Close()
	var n uint32
	for ; iter.Valid(); iter.Next() {
		n++
	}
	return n
}

func (k Keeper) indexActiveBounty(ctx context.Context, order *types.BountyOrder) {
	if order == nil || order.Status != types.BountyStatus_BOUNTY_STATUS_ACTIVE {
		return
	}
	canonicalSponsor, err := types.CanonicalAccountAddress(order.Sponsor)
	if err != nil {
		return
	}
	kv := k.storeService.OpenKVStore(ctx)
	if canonicalSponsor != order.Sponsor {
		_ = kv.Delete(types.ActiveSponsorIndexKey(order.Sponsor, order.Id))
	}
	_ = kv.Set(types.ActiveSponsorIndexKey(canonicalSponsor, order.Id), []byte{1})
	_ = kv.Set(types.DeadlineIndexKey(order.EndBlock, order.Id), []byte{1})
}

func (k Keeper) unindexActiveBounty(ctx context.Context, order *types.BountyOrder) {
	if order == nil {
		return
	}
	kv := k.storeService.OpenKVStore(ctx)
	_ = kv.Delete(types.ActiveSponsorIndexKey(order.Sponsor, order.Id))
	if canonicalSponsor, err := types.CanonicalAccountAddress(order.Sponsor); err == nil && canonicalSponsor != order.Sponsor {
		_ = kv.Delete(types.ActiveSponsorIndexKey(canonicalSponsor, order.Id))
	}
	_ = kv.Delete(types.DeadlineIndexKey(order.EndBlock, order.Id))
}

// PruneExpiredBountiesForSponsor lazily closes the sponsor's bounded active
// set before enforcing MaxActiveBountiesPerSponsor. This keeps delayed global
// expiry processing from stranding a sponsor behind stale index entries.
func (k Keeper) PruneExpiredBountiesForSponsor(ctx context.Context, sponsor string, currentBlock uint64) {
	canonicalSponsor, err := types.CanonicalAccountAddress(sponsor)
	if err != nil {
		return
	}
	kv := k.storeService.OpenKVStore(ctx)
	prefix := types.ActiveSponsorIndexPrefix(canonicalSponsor)
	iter, err := kv.Iterator(prefix, prefixEndBytes(prefix))
	if err != nil {
		return
	}
	var ids []string
	for ; iter.Valid(); iter.Next() {
		ids = append(ids, string(iter.Key()[len(prefix):]))
	}
	iter.Close()
	for _, id := range ids {
		order, found := k.GetBountyOrder(ctx, id)
		if !found || order.Status != types.BountyStatus_BOUNTY_STATUS_ACTIVE {
			_ = kv.Delete(types.ActiveSponsorIndexKey(canonicalSponsor, id))
			continue
		}
		if currentBlock < order.EndBlock {
			continue
		}
		order.Status = types.BountyStatus_BOUNTY_STATUS_EXPIRED
		k.SetBountyOrder(ctx, order)
		k.unindexActiveBounty(ctx, order)
	}
}

// ---------- Fulfillment CRUD ----------

func (k Keeper) SetFulfillment(ctx context.Context, f *types.BountyFulfillment) {
	kv := k.storeService.OpenKVStore(ctx)
	bz, err := proto.Marshal(f)
	if err != nil {
		panic(fmt.Sprintf("marshal fulfillment: %v", err))
	}
	_ = kv.Set(types.FulfillmentKey(f.BountyId, f.FactId), bz)
	k.SetConsumptionIndexes(ctx, f)
}

// SetConsumptionIndexes writes permanent replay tombstones. Empty v1 receipt
// fields are skipped, but every legacy fact is still marked consumed so a
// pre-upgrade payout cannot be replayed through a bound v2 order.
func (k Keeper) SetConsumptionIndexes(ctx context.Context, f *types.BountyFulfillment) {
	if f == nil {
		return
	}
	kv := k.storeService.OpenKVStore(ctx)
	if f.FactId != "" {
		_ = kv.Set(types.FactConsumptionKey(f.FactId), []byte(f.BountyId))
	}
	if f.WorkReceiptHash != "" {
		_ = kv.Set(types.ReceiptConsumptionKey(f.WorkReceiptHash), []byte(f.BountyId))
	}
	if f.SettlementNullifier != "" {
		_ = kv.Set(types.SettlementNullifierKey(f.SettlementNullifier), []byte(f.BountyId))
	}
}

func (k Keeper) IsFactConsumed(ctx context.Context, factID string) bool {
	bz, err := k.storeService.OpenKVStore(ctx).Get(types.FactConsumptionKey(factID))
	return err == nil && bz != nil
}

func (k Keeper) IsReceiptConsumed(ctx context.Context, receiptHash string) bool {
	bz, err := k.storeService.OpenKVStore(ctx).Get(types.ReceiptConsumptionKey(receiptHash))
	return err == nil && bz != nil
}

func (k Keeper) IsSettlementNullifierConsumed(ctx context.Context, nullifier string) bool {
	bz, err := k.storeService.OpenKVStore(ctx).Get(types.SettlementNullifierKey(nullifier))
	return err == nil && bz != nil
}

func (k Keeper) GetFulfillment(ctx context.Context, bountyID, factID string) (*types.BountyFulfillment, bool) {
	kv := k.storeService.OpenKVStore(ctx)
	bz, err := kv.Get(types.FulfillmentKey(bountyID, factID))
	if err != nil || bz == nil {
		return nil, false
	}
	var f types.BountyFulfillment
	if err := proto.Unmarshal(bz, &f); err != nil {
		return nil, false
	}
	return &f, true
}

func (k Keeper) GetAllFulfillments(ctx context.Context) []*types.BountyFulfillment {
	kv := k.storeService.OpenKVStore(ctx)
	iter, err := kv.Iterator(types.FulfillmentKeyPrefix, prefixEndBytes(types.FulfillmentKeyPrefix))
	if err != nil {
		return nil
	}
	defer iter.Close()
	var out []*types.BountyFulfillment
	for ; iter.Valid(); iter.Next() {
		var f types.BountyFulfillment
		if err := proto.Unmarshal(iter.Value(), &f); err != nil {
			continue
		}
		out = append(out, &f)
	}
	return out
}

// DerivedEscrowLiability scans orders to derive the open liability. Consensus
// transaction paths never call it; it is reserved for genesis, migration,
// export verification, and tests.
func (k Keeper) DerivedEscrowLiability(ctx context.Context) (*big.Int, error) {
	total := new(big.Int)
	var firstErr error
	k.IterateBountyOrders(ctx, func(order *types.BountyOrder) bool {
		remaining, err := types.ParseNonNegativeAmount(order.EscrowRemaining)
		if err != nil {
			firstErr = fmt.Errorf("bounty %s: invalid escrow_remaining: %w", order.Id, err)
			return true
		}
		expected, err := types.ExpectedEscrowRemaining(order)
		if err != nil {
			firstErr = fmt.Errorf("bounty %s: %w", order.Id, err)
			return true
		}
		if remaining.Cmp(expected) != 0 {
			firstErr = fmt.Errorf("bounty %s: escrow_remaining %s != derived liability %s", order.Id, remaining, expected)
			return true
		}
		if order.Status == types.BountyStatus_BOUNTY_STATUS_ACTIVE || order.Status == types.BountyStatus_BOUNTY_STATUS_EXPIRED {
			total.Add(total, remaining)
		} else if remaining.Sign() != 0 {
			firstErr = fmt.Errorf("bounty %s: terminal status %s retains escrow %s", order.Id, order.Status, remaining)
			return true
		}
		return false
	})
	if firstErr != nil {
		return nil, firstErr
	}
	return total, nil
}

// TotalEscrowLiability reads the persisted aggregate in O(1).
func (k Keeper) TotalEscrowLiability(ctx context.Context) (*big.Int, error) {
	bz, err := k.storeService.OpenKVStore(ctx).Get(types.EscrowLiabilityKey)
	if err != nil {
		return nil, err
	}
	if bz == nil {
		return new(big.Int), nil
	}
	liability, err := types.ParseNonNegativeAmount(string(bz))
	if err != nil {
		return nil, fmt.Errorf("invalid persisted escrow liability: %w", err)
	}
	return liability, nil
}

func (k Keeper) SetEscrowLiability(ctx context.Context, liability *big.Int) error {
	if liability == nil {
		return fmt.Errorf("escrow liability cannot be nil")
	}
	canonical, err := types.ParseNonNegativeAmount(liability.String())
	if err != nil {
		return err
	}
	return k.storeService.OpenKVStore(ctx).Set(types.EscrowLiabilityKey, []byte(canonical.String()))
}

func (k Keeper) IncreaseEscrowLiability(ctx context.Context, amount *big.Int) error {
	if err := k.CanIncreaseEscrowLiability(ctx, amount); err != nil {
		return err
	}
	current, _ := k.TotalEscrowLiability(ctx)
	next := new(big.Int).Add(current, amount)
	return k.SetEscrowLiability(ctx, next)
}

func (k Keeper) CanIncreaseEscrowLiability(ctx context.Context, amount *big.Int) error {
	if amount == nil || amount.Sign() < 0 {
		return fmt.Errorf("liability increase must be non-negative")
	}
	current, err := k.TotalEscrowLiability(ctx)
	if err != nil {
		return err
	}
	next := new(big.Int).Add(current, amount)
	_, err = types.ParseNonNegativeAmount(next.String())
	return err
}

func (k Keeper) CanDecreaseEscrowLiability(ctx context.Context, amount *big.Int) error {
	if amount == nil || amount.Sign() < 0 {
		return fmt.Errorf("liability decrease must be non-negative")
	}
	current, err := k.TotalEscrowLiability(ctx)
	if err != nil {
		return err
	}
	if current.Cmp(amount) < 0 {
		return fmt.Errorf("persisted escrow liability %s below outgoing amount %s", current, amount)
	}
	return nil
}

func (k Keeper) DecreaseEscrowLiability(ctx context.Context, amount *big.Int) error {
	if err := k.CanDecreaseEscrowLiability(ctx, amount); err != nil {
		return err
	}
	current, _ := k.TotalEscrowLiability(ctx)
	return k.SetEscrowLiability(ctx, new(big.Int).Sub(current, amount))
}

func (k Keeper) EnsureEscrowAccounting(ctx context.Context) error {
	stored, err := k.TotalEscrowLiability(ctx)
	if err != nil {
		return err
	}
	derived, err := k.DerivedEscrowLiability(ctx)
	if err != nil {
		return err
	}
	if stored.Cmp(derived) != 0 {
		return fmt.Errorf("persisted escrow liability %s differs from derived order liability %s", stored, derived)
	}
	return nil
}

// EnsureEscrowSolvent enforces the outgoing-payment wall. Unsolicited module
// transfers are harmless surplus, so solvency is balance >= liabilities.
func (k Keeper) EnsureEscrowSolvent(ctx context.Context) error {
	liability, err := k.TotalEscrowLiability(ctx)
	if err != nil {
		return err
	}
	moduleAddr := sdk.AccAddress(authtypes.NewModuleAddress(types.ModuleName))
	balance := k.bankKeeper.GetBalance(ctx, moduleAddr, "uzrn").Amount.BigInt()
	if balance.Cmp(liability) < 0 {
		return fmt.Errorf("sponsorship module balance %suzrn is below escrow liability %suzrn", balance, liability)
	}
	return nil
}

// ---------- BeginBlocker — bounded deadline index ----------

const MaxExpiryTransitionsPerBlock = 256

// ProcessBountyExpiry flips ACTIVE bounties whose end_block has elapsed
// to EXPIRED. Unlike claiming_pot's bootstrap-pot rule, sponsorship
// bounties DO expire — the sponsor's deadline is a methodological
// commitment (commitment 1) and the chain honors it. Funds remain in
// escrow on EXPIRED bounties until the sponsor calls CancelBountyOrder
// to reclaim them.
func (k Keeper) ProcessBountyExpiry(ctx context.Context, currentBlock uint64) {
	kv := k.storeService.OpenKVStore(ctx)
	iter, err := kv.Iterator(types.DeadlineIndexKeyPrefix, prefixEndBytes(types.DeadlineIndexKeyPrefix))
	if err != nil {
		return
	}
	type dueEntry struct {
		key []byte
		id  string
	}
	due := make([]dueEntry, 0, MaxExpiryTransitionsPerBlock)
	for ; iter.Valid() && len(due) < MaxExpiryTransitionsPerBlock; iter.Next() {
		key := iter.Key()
		if len(key) < 9 {
			due = append(due, dueEntry{key: append([]byte(nil), key...)})
			continue
		}
		endBlock := binary.BigEndian.Uint64(key[1:9])
		if endBlock > currentBlock {
			break
		}
		due = append(due, dueEntry{key: append([]byte(nil), key...), id: string(key[9:])})
	}
	iter.Close()

	for _, entry := range due {
		_ = kv.Delete(entry.key)
		if entry.id == "" {
			continue
		}
		order, found := k.GetBountyOrder(ctx, entry.id)
		if !found {
			continue
		}
		if order.Status != types.BountyStatus_BOUNTY_STATUS_ACTIVE {
			k.unindexActiveBounty(ctx, order)
			continue
		}
		if currentBlock < order.EndBlock {
			k.indexActiveBounty(ctx, order) // repair a stale deadline key
			continue
		}
		order.Status = types.BountyStatus_BOUNTY_STATUS_EXPIRED
		k.SetBountyOrder(ctx, order)
		k.unindexActiveBounty(ctx, order)
	}
}

// ---------- Genesis ----------

func (k Keeper) InitGenesis(ctx context.Context, gs *types.GenesisState) {
	if err := gs.Validate(); err != nil {
		panic(fmt.Sprintf("invalid sponsorship genesis: %v", err))
	}
	if gs.Params != nil {
		k.SetParams(ctx, gs.Params)
	}
	for _, o := range gs.Orders {
		k.SetBountyOrder(ctx, o)
		k.indexActiveBounty(ctx, o)
	}
	for _, f := range gs.Fulfillments {
		k.SetFulfillment(ctx, f)
	}
	if gs.NextBountyId > 0 {
		kv := k.storeService.OpenKVStore(ctx)
		buf := make([]byte, 8)
		binary.BigEndian.PutUint64(buf, gs.NextBountyId-1) // counter increments before use
		_ = kv.Set(types.BountyCounterKey, buf)
	}
	liability, err := k.DerivedEscrowLiability(ctx)
	if err != nil {
		panic(fmt.Sprintf("derive sponsorship genesis liability: %v", err))
	}
	if err := k.SetEscrowLiability(ctx, liability); err != nil {
		panic(fmt.Sprintf("store sponsorship genesis liability: %v", err))
	}
	// Bank initializes before sponsorship in app genesis order. Refuse to boot
	// an authored/imported state whose module balance cannot cover every open
	// escrow liability; GenesisState.Validate cannot inspect x/bank.
	if err := k.EnsureEscrowSolvent(ctx); err != nil {
		panic(fmt.Sprintf("insolvent sponsorship genesis: %v", err))
	}
}

func (k Keeper) ExportGenesis(ctx context.Context) *types.GenesisState {
	if err := k.EnsureEscrowAccounting(ctx); err != nil {
		panic(fmt.Sprintf("export sponsorship escrow accounting: %v", err))
	}
	return &types.GenesisState{
		Params:       k.GetParams(ctx),
		Orders:       k.GetAllBountyOrders(ctx),
		Fulfillments: k.GetAllFulfillments(ctx),
		NextBountyId: k.peekNextBountyID(ctx),
	}
}

func (k Keeper) peekNextBountyID(ctx context.Context) uint64 {
	kv := k.storeService.OpenKVStore(ctx)
	bz, err := kv.Get(types.BountyCounterKey)
	if err != nil || bz == nil {
		return 1
	}
	return binary.BigEndian.Uint64(bz) + 1
}

// ---------- helpers ----------

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
