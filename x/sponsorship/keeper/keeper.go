package keeper

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
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

func (k Keeper) setParamsChecked(ctx context.Context, params *types.Params) error {
	bz, err := proto.Marshal(params)
	if err != nil {
		return fmt.Errorf("marshal params: %w", err)
	}
	if err := k.storeService.OpenKVStore(ctx).Set(types.ParamsKey, bz); err != nil {
		return fmt.Errorf("write params: %w", err)
	}
	return nil
}

func (k Keeper) GetParams(ctx context.Context) *types.Params {
	params, err := k.getParamsChecked(ctx)
	if err != nil {
		return types.DefaultParams()
	}
	return params
}

func (k Keeper) getParamsChecked(ctx context.Context) (*types.Params, error) {
	kv := k.storeService.OpenKVStore(ctx)
	bz, err := kv.Get(types.ParamsKey)
	if err != nil {
		return nil, fmt.Errorf("read params: %w", err)
	}
	if bz == nil {
		return nil, fmt.Errorf("sponsorship params are absent")
	}
	var p types.Params
	if err := proto.Unmarshal(bz, &p); err != nil {
		return nil, fmt.Errorf("decode params: %w", err)
	}
	return &p, nil
}

// ---------- Counter ----------

func (k Keeper) nextBountyID(ctx context.Context) (uint64, error) {
	kv := k.storeService.OpenKVStore(ctx)
	bz, err := kv.Get(types.BountyCounterKey)
	if err != nil {
		return 0, fmt.Errorf("read bounty counter: %w", err)
	}
	if bz == nil {
		bz = make([]byte, 8)
	}
	if len(bz) != 8 {
		return 0, fmt.Errorf("invalid bounty counter length %d", len(bz))
	}
	n := binary.BigEndian.Uint64(bz)
	if n >= math.MaxUint64-1 {
		return 0, fmt.Errorf("bounty id space exhausted")
	}
	n++
	newBz := make([]byte, 8)
	binary.BigEndian.PutUint64(newBz, n)
	if err := kv.Set(types.BountyCounterKey, newBz); err != nil {
		return 0, fmt.Errorf("write bounty counter: %w", err)
	}
	return n, nil
}

func (k Keeper) canAllocateBountyID(ctx context.Context) error {
	kv := k.storeService.OpenKVStore(ctx)
	bz, err := kv.Get(types.BountyCounterKey)
	if err != nil {
		return err
	}
	if bz != nil && len(bz) != 8 {
		return fmt.Errorf("invalid bounty counter length %d", len(bz))
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

func (k Keeper) setBountyOrderChecked(ctx context.Context, order *types.BountyOrder) error {
	if order == nil || order.Id == "" {
		return fmt.Errorf("cannot write nil or id-less bounty order")
	}
	bz, err := proto.Marshal(order)
	if err != nil {
		return fmt.Errorf("marshal bounty %q: %w", order.Id, err)
	}
	if err := k.storeService.OpenKVStore(ctx).Set(types.BountyOrderKey(order.Id), bz); err != nil {
		return fmt.Errorf("write bounty %q: %w", order.Id, err)
	}
	return nil
}

func (k Keeper) GetBountyOrder(ctx context.Context, id string) (*types.BountyOrder, bool) {
	order, found, err := k.getBountyOrderChecked(ctx, id)
	if err != nil {
		return nil, false
	}
	return order, found
}

func (k Keeper) getBountyOrderChecked(
	ctx context.Context,
	id string,
) (*types.BountyOrder, bool, error) {
	kv := k.storeService.OpenKVStore(ctx)
	bz, err := kv.Get(types.BountyOrderKey(id))
	if err != nil {
		return nil, false, fmt.Errorf("read bounty %q: %w", id, err)
	}
	if bz == nil {
		return nil, false, nil
	}
	var o types.BountyOrder
	if err := proto.Unmarshal(bz, &o); err != nil {
		return nil, false, fmt.Errorf("decode bounty %q: %w", id, err)
	}
	if o.Id != id {
		return nil, false, fmt.Errorf(
			"bounty key id %q does not match encoded id %q",
			id,
			o.Id,
		)
	}
	return &o, true, nil
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

// getAllBountyOrdersChecked is the migration/accounting census. Unlike the
// permissive query helper above, it refuses any unreadable, malformed, or
// key-mismatched row and checks close errors. Cosmos SDK's cacheMergeIterator
// defines Error as "iterator is no longer valid", so natural exhaustion always
// returns an error and cannot be used as an underlying-storage error channel.
func (k Keeper) getAllBountyOrdersChecked(
	ctx context.Context,
) (orders []*types.BountyOrder, err error) {
	kv := k.storeService.OpenKVStore(ctx)
	iter, err := kv.Iterator(
		types.BountyOrderKeyPrefix,
		prefixEndBytes(types.BountyOrderKeyPrefix),
	)
	if err != nil {
		return nil, fmt.Errorf("open bounty-order census: %w", err)
	}
	defer func() {
		closeErr := iter.Close()
		if closeErr != nil {
			closeErr = fmt.Errorf("close bounty-order census: %w", closeErr)
		}
		if joined := errors.Join(err, closeErr); joined != nil {
			orders = nil
			err = joined
		}
	}()

	for ; iter.Valid(); iter.Next() {
		var order types.BountyOrder
		if decodeErr := proto.Unmarshal(iter.Value(), &order); decodeErr != nil {
			return nil, fmt.Errorf(
				"decode bounty-order row at key %x: %w",
				iter.Key(),
				decodeErr,
			)
		}
		if order.Id == "" || !bytes.Equal(iter.Key(), types.BountyOrderKey(order.Id)) {
			return nil, fmt.Errorf(
				"bounty-order row key %x does not match encoded id %q",
				iter.Key(),
				order.Id,
			)
		}
		orders = append(orders, &order)
	}
	return orders, nil
}

func (k Keeper) CountActiveBountiesBySponsor(ctx context.Context, sponsor string) uint32 {
	count, err := k.countActiveBountiesBySponsorChecked(ctx, sponsor)
	if err != nil {
		return 0
	}
	return count
}

func (k Keeper) countActiveBountiesBySponsorChecked(
	ctx context.Context,
	sponsor string,
) (count uint32, err error) {
	canonicalSponsor, err := types.CanonicalAccountAddress(sponsor)
	if err != nil {
		return 0, err
	}
	kv := k.storeService.OpenKVStore(ctx)
	prefix := types.ActiveSponsorIndexPrefix(canonicalSponsor)
	iter, err := kv.Iterator(prefix, prefixEndBytes(prefix))
	if err != nil {
		return 0, fmt.Errorf("open active-bounty index for %q: %w", canonicalSponsor, err)
	}
	defer func() {
		if closeErr := iter.Close(); closeErr != nil {
			count = 0
			err = errors.Join(err, fmt.Errorf(
				"close active-bounty index for %q: %w",
				canonicalSponsor,
				closeErr,
			))
		}
	}()
	for ; iter.Valid(); iter.Next() {
		if len(iter.Key()) <= len(prefix) {
			return 0, fmt.Errorf("empty bounty id in active index for %q", canonicalSponsor)
		}
		if count == math.MaxUint32 {
			return 0, fmt.Errorf("active-bounty count overflow for %q", canonicalSponsor)
		}
		count++
	}
	return count, nil
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

func (k Keeper) indexActiveBountyChecked(ctx context.Context, order *types.BountyOrder) error {
	if order == nil {
		return fmt.Errorf("cannot index nil bounty")
	}
	if order.Status != types.BountyStatus_BOUNTY_STATUS_ACTIVE {
		return nil
	}
	canonicalSponsor, err := types.CanonicalAccountAddress(order.Sponsor)
	if err != nil {
		return fmt.Errorf("canonicalize active bounty %q sponsor: %w", order.Id, err)
	}
	kv := k.storeService.OpenKVStore(ctx)
	if canonicalSponsor != order.Sponsor {
		if err := kv.Delete(types.ActiveSponsorIndexKey(order.Sponsor, order.Id)); err != nil {
			return fmt.Errorf("delete legacy sponsor index for bounty %q: %w", order.Id, err)
		}
	}
	if err := kv.Set(types.ActiveSponsorIndexKey(canonicalSponsor, order.Id), []byte{1}); err != nil {
		return fmt.Errorf("write active sponsor index for bounty %q: %w", order.Id, err)
	}
	if err := kv.Set(types.DeadlineIndexKey(order.EndBlock, order.Id), []byte{1}); err != nil {
		return fmt.Errorf("write deadline index for bounty %q: %w", order.Id, err)
	}
	return nil
}

func (k Keeper) unindexActiveBounty(ctx context.Context, order *types.BountyOrder) {
	_ = k.unindexActiveBountyChecked(ctx, order)
}

func (k Keeper) unindexActiveBountyChecked(
	ctx context.Context,
	order *types.BountyOrder,
) error {
	if order == nil {
		return nil
	}
	canonicalSponsor, err := types.CanonicalAccountAddress(order.Sponsor)
	if err != nil {
		return fmt.Errorf("canonicalize bounty %q sponsor before unindex: %w", order.Id, err)
	}
	kv := k.storeService.OpenKVStore(ctx)
	if err := kv.Delete(types.ActiveSponsorIndexKey(order.Sponsor, order.Id)); err != nil {
		return fmt.Errorf("delete active sponsor index for bounty %q: %w", order.Id, err)
	}
	if canonicalSponsor != order.Sponsor {
		if err := kv.Delete(types.ActiveSponsorIndexKey(canonicalSponsor, order.Id)); err != nil {
			return fmt.Errorf("delete canonical active sponsor index for bounty %q: %w", order.Id, err)
		}
	}
	if err := kv.Delete(types.DeadlineIndexKey(order.EndBlock, order.Id)); err != nil {
		return fmt.Errorf("delete deadline index for bounty %q: %w", order.Id, err)
	}
	return nil
}

// PruneExpiredBountiesForSponsor lazily closes the sponsor's bounded active
// set before enforcing MaxActiveBountiesPerSponsor. This keeps delayed global
// expiry processing from stranding a sponsor behind stale index entries.
func (k Keeper) PruneExpiredBountiesForSponsor(
	ctx context.Context,
	sponsor string,
	currentBlock uint64,
) error {
	canonicalSponsor, err := types.CanonicalAccountAddress(sponsor)
	if err != nil {
		return err
	}
	kv := k.storeService.OpenKVStore(ctx)
	prefix := types.ActiveSponsorIndexPrefix(canonicalSponsor)
	iter, err := kv.Iterator(prefix, prefixEndBytes(prefix))
	if err != nil {
		return fmt.Errorf("open active-bounty pruning index for %q: %w", canonicalSponsor, err)
	}
	ids := make([]string, 0, types.MaxActiveBountiesPerSponsorHardCap)
	for ; iter.Valid(); iter.Next() {
		if len(iter.Key()) <= len(prefix) {
			closeErr := iter.Close()
			return errors.Join(
				fmt.Errorf("empty bounty id in active pruning index for %q", canonicalSponsor),
				closeErr,
			)
		}
		if uint32(len(ids)) >= types.MaxActiveBountiesPerSponsorHardCap {
			closeErr := iter.Close()
			return errors.Join(
				fmt.Errorf(
					"active pruning index for %q exceeds hard cap %d",
					canonicalSponsor,
					types.MaxActiveBountiesPerSponsorHardCap,
				),
				closeErr,
			)
		}
		ids = append(ids, string(iter.Key()[len(prefix):]))
	}
	if err := iter.Close(); err != nil {
		return fmt.Errorf("close active-bounty pruning index for %q: %w", canonicalSponsor, err)
	}
	for _, id := range ids {
		order, found, err := k.getBountyOrderChecked(ctx, id)
		if err != nil {
			return err
		}
		if !found || order.Status != types.BountyStatus_BOUNTY_STATUS_ACTIVE {
			if err := kv.Delete(types.ActiveSponsorIndexKey(canonicalSponsor, id)); err != nil {
				return fmt.Errorf("delete stale active index for bounty %q: %w", id, err)
			}
			continue
		}
		if currentBlock < order.EndBlock {
			continue
		}
		order.Status = types.BountyStatus_BOUNTY_STATUS_EXPIRED
		if err := k.setBountyOrderChecked(ctx, order); err != nil {
			return err
		}
		if err := k.unindexActiveBountyChecked(ctx, order); err != nil {
			return err
		}
	}
	return nil
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

func (k Keeper) setFulfillmentChecked(ctx context.Context, fulfillment *types.BountyFulfillment) error {
	if fulfillment == nil || fulfillment.BountyId == "" || fulfillment.FactId == "" {
		return fmt.Errorf("cannot write nil or unbound fulfillment")
	}
	bz, err := proto.Marshal(fulfillment)
	if err != nil {
		return fmt.Errorf(
			"marshal fulfillment %q/%q: %w",
			fulfillment.BountyId,
			fulfillment.FactId,
			err,
		)
	}
	kv := k.storeService.OpenKVStore(ctx)
	if err := kv.Set(
		types.FulfillmentKey(fulfillment.BountyId, fulfillment.FactId),
		bz,
	); err != nil {
		return fmt.Errorf(
			"write fulfillment %q/%q: %w",
			fulfillment.BountyId,
			fulfillment.FactId,
			err,
		)
	}
	return k.setConsumptionIndexesChecked(ctx, fulfillment)
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

func (k Keeper) setConsumptionIndexesChecked(
	ctx context.Context,
	fulfillment *types.BountyFulfillment,
) error {
	if fulfillment == nil {
		return fmt.Errorf("cannot index nil fulfillment")
	}
	kv := k.storeService.OpenKVStore(ctx)
	entries := []struct {
		key   []byte
		value string
		name  string
	}{
		{types.FactConsumptionKey(fulfillment.FactId), fulfillment.BountyId, "fact"},
	}
	if fulfillment.WorkReceiptHash != "" {
		entries = append(entries, struct {
			key   []byte
			value string
			name  string
		}{types.ReceiptConsumptionKey(fulfillment.WorkReceiptHash), fulfillment.BountyId, "receipt"})
	}
	if fulfillment.SettlementNullifier != "" {
		entries = append(entries, struct {
			key   []byte
			value string
			name  string
		}{types.SettlementNullifierKey(fulfillment.SettlementNullifier), fulfillment.BountyId, "settlement nullifier"})
	}
	for _, entry := range entries {
		if err := kv.Set(entry.key, []byte(entry.value)); err != nil {
			return fmt.Errorf(
				"write %s consumption index for fulfillment %q/%q: %w",
				entry.name,
				fulfillment.BountyId,
				fulfillment.FactId,
				err,
			)
		}
	}
	return nil
}

func (k Keeper) IsFactConsumed(ctx context.Context, factID string) bool {
	_, found, err := k.consumptionOwnerChecked(ctx, types.FactConsumptionKey(factID), "fact")
	return err == nil && found
}

func (k Keeper) IsReceiptConsumed(ctx context.Context, receiptHash string) bool {
	_, found, err := k.consumptionOwnerChecked(ctx, types.ReceiptConsumptionKey(receiptHash), "receipt")
	return err == nil && found
}

func (k Keeper) IsSettlementNullifierConsumed(ctx context.Context, nullifier string) bool {
	_, found, err := k.consumptionOwnerChecked(
		ctx,
		types.SettlementNullifierKey(nullifier),
		"settlement nullifier",
	)
	return err == nil && found
}

func (k Keeper) GetFulfillment(ctx context.Context, bountyID, factID string) (*types.BountyFulfillment, bool) {
	fulfillment, found, err := k.getFulfillmentChecked(ctx, bountyID, factID)
	if err != nil {
		return nil, false
	}
	return fulfillment, found
}

func (k Keeper) getFulfillmentChecked(
	ctx context.Context,
	bountyID string,
	factID string,
) (*types.BountyFulfillment, bool, error) {
	kv := k.storeService.OpenKVStore(ctx)
	bz, err := kv.Get(types.FulfillmentKey(bountyID, factID))
	if err != nil {
		return nil, false, fmt.Errorf(
			"read fulfillment %q/%q: %w",
			bountyID,
			factID,
			err,
		)
	}
	if bz == nil {
		return nil, false, nil
	}
	var f types.BountyFulfillment
	if err := proto.Unmarshal(bz, &f); err != nil {
		return nil, false, fmt.Errorf(
			"decode fulfillment %q/%q: %w",
			bountyID,
			factID,
			err,
		)
	}
	if f.BountyId != bountyID || f.FactId != factID {
		return nil, false, fmt.Errorf(
			"fulfillment key %q/%q does not match encoded pair %q/%q",
			bountyID,
			factID,
			f.BountyId,
			f.FactId,
		)
	}
	return &f, true, nil
}

func (k Keeper) consumptionOwnerChecked(
	ctx context.Context,
	key []byte,
	kind string,
) (string, bool, error) {
	bz, err := k.storeService.OpenKVStore(ctx).Get(key)
	if err != nil {
		return "", false, fmt.Errorf("read %s consumption index: %w", kind, err)
	}
	if bz == nil {
		return "", false, nil
	}
	if len(bz) == 0 {
		return "", false, fmt.Errorf("%s consumption index has empty bounty id", kind)
	}
	return string(bz), true, nil
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

func (k Keeper) getAllFulfillmentsChecked(
	ctx context.Context,
) (fulfillments []*types.BountyFulfillment, err error) {
	kv := k.storeService.OpenKVStore(ctx)
	iter, err := kv.Iterator(
		types.FulfillmentKeyPrefix,
		prefixEndBytes(types.FulfillmentKeyPrefix),
	)
	if err != nil {
		return nil, fmt.Errorf("open fulfillment census: %w", err)
	}
	defer func() {
		closeErr := iter.Close()
		if closeErr != nil {
			closeErr = fmt.Errorf("close fulfillment census: %w", closeErr)
		}
		if joined := errors.Join(err, closeErr); joined != nil {
			fulfillments = nil
			err = joined
		}
	}()

	for ; iter.Valid(); iter.Next() {
		var fulfillment types.BountyFulfillment
		if decodeErr := proto.Unmarshal(iter.Value(), &fulfillment); decodeErr != nil {
			return nil, fmt.Errorf(
				"decode fulfillment row at key %x: %w",
				iter.Key(),
				decodeErr,
			)
		}
		if fulfillment.BountyId == "" || fulfillment.FactId == "" ||
			!bytes.Equal(
				iter.Key(),
				types.FulfillmentKey(fulfillment.BountyId, fulfillment.FactId),
			) {
			return nil, fmt.Errorf(
				"fulfillment row key %x does not match encoded pair %q/%q",
				iter.Key(),
				fulfillment.BountyId,
				fulfillment.FactId,
			)
		}
		fulfillments = append(fulfillments, &fulfillment)
	}
	return fulfillments, nil
}

// DerivedEscrowLiability scans orders to derive the open liability. Consensus
// transaction paths never call it; it is reserved for genesis, migration,
// export verification, and tests.
func (k Keeper) DerivedEscrowLiability(ctx context.Context) (*big.Int, error) {
	orders, err := k.getAllBountyOrdersChecked(ctx)
	if err != nil {
		return nil, err
	}
	return derivedEscrowLiabilityFromOrders(orders)
}

func derivedEscrowLiabilityFromOrders(
	orders []*types.BountyOrder,
) (*big.Int, error) {
	total := new(big.Int)
	for _, order := range orders {
		if order == nil {
			return nil, fmt.Errorf("nil bounty order in liability census")
		}
		remaining, err := types.ParseNonNegativeAmount(order.EscrowRemaining)
		if err != nil {
			return nil, fmt.Errorf("bounty %s: invalid escrow_remaining: %w", order.Id, err)
		}
		expected, err := types.ExpectedEscrowRemaining(order)
		if err != nil {
			return nil, fmt.Errorf("bounty %s: %w", order.Id, err)
		}
		if remaining.Cmp(expected) != 0 {
			return nil, fmt.Errorf("bounty %s: escrow_remaining %s != derived liability %s", order.Id, remaining, expected)
		}
		if order.Status == types.BountyStatus_BOUNTY_STATUS_ACTIVE || order.Status == types.BountyStatus_BOUNTY_STATUS_EXPIRED {
			total.Add(total, remaining)
		} else if remaining.Sign() != 0 {
			return nil, fmt.Errorf("bounty %s: terminal status %s retains escrow %s", order.Id, order.Status, remaining)
		}
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
		return nil, fmt.Errorf("persisted escrow liability is absent")
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
	if amount == nil || amount.Sign() < 0 {
		return fmt.Errorf("liability increase must be non-negative")
	}
	current, err := k.TotalEscrowLiability(ctx)
	if err != nil {
		return err
	}
	next := new(big.Int).Add(current, amount)
	if _, err := types.ParseNonNegativeAmount(next.String()); err != nil {
		return err
	}
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
func (k Keeper) ProcessBountyExpiry(ctx context.Context, currentBlock uint64) error {
	kv := k.storeService.OpenKVStore(ctx)
	iter, err := kv.Iterator(types.DeadlineIndexKeyPrefix, prefixEndBytes(types.DeadlineIndexKeyPrefix))
	if err != nil {
		return fmt.Errorf("open deadline expiry index: %w", err)
	}
	type dueEntry struct {
		key []byte
		id  string
	}
	due := make([]dueEntry, 0, MaxExpiryTransitionsPerBlock)
	for ; iter.Valid() && len(due) < MaxExpiryTransitionsPerBlock; iter.Next() {
		key := bytes.Clone(iter.Key())
		if len(key) <= 9 {
			closeErr := iter.Close()
			return errors.Join(
				fmt.Errorf("malformed deadline index key %x", key),
				closeErr,
			)
		}
		endBlock := binary.BigEndian.Uint64(key[1:9])
		if endBlock > currentBlock {
			break
		}
		due = append(due, dueEntry{key: key, id: string(key[9:])})
	}
	if err := iter.Close(); err != nil {
		return fmt.Errorf("close deadline expiry index: %w", err)
	}

	for _, entry := range due {
		if err := kv.Delete(entry.key); err != nil {
			return fmt.Errorf("delete deadline index for bounty %q: %w", entry.id, err)
		}
		order, found, err := k.getBountyOrderChecked(ctx, entry.id)
		if err != nil {
			return err
		}
		if !found {
			continue
		}
		if order.Status != types.BountyStatus_BOUNTY_STATUS_ACTIVE {
			if err := k.unindexActiveBountyChecked(ctx, order); err != nil {
				return err
			}
			continue
		}
		if currentBlock < order.EndBlock {
			if err := k.indexActiveBountyChecked(ctx, order); err != nil {
				return err
			}
			continue
		}
		order.Status = types.BountyStatus_BOUNTY_STATUS_EXPIRED
		if err := k.setBountyOrderChecked(ctx, order); err != nil {
			return err
		}
		if err := k.unindexActiveBountyChecked(ctx, order); err != nil {
			return err
		}
	}
	return nil
}

// ---------- Genesis ----------

func (k Keeper) InitGenesis(ctx context.Context, gs *types.GenesisState) {
	if err := gs.Validate(); err != nil {
		panic(fmt.Sprintf("invalid sponsorship genesis: %v", err))
	}
	if gs.Params != nil {
		if err := k.setParamsChecked(ctx, gs.Params); err != nil {
			panic(fmt.Sprintf("store sponsorship genesis params: %v", err))
		}
	}
	for _, o := range gs.Orders {
		if err := k.setBountyOrderChecked(ctx, o); err != nil {
			panic(fmt.Sprintf("store sponsorship genesis order: %v", err))
		}
		if err := k.indexActiveBountyChecked(ctx, o); err != nil {
			panic(fmt.Sprintf("index sponsorship genesis order: %v", err))
		}
	}
	for _, f := range gs.Fulfillments {
		if err := k.setFulfillmentChecked(ctx, f); err != nil {
			panic(fmt.Sprintf("store sponsorship genesis fulfillment: %v", err))
		}
	}
	if gs.NextBountyId > 0 {
		kv := k.storeService.OpenKVStore(ctx)
		buf := make([]byte, 8)
		binary.BigEndian.PutUint64(buf, gs.NextBountyId-1) // counter increments before use
		if err := kv.Set(types.BountyCounterKey, buf); err != nil {
			panic(fmt.Sprintf("store sponsorship genesis counter: %v", err))
		}
	}
	liability, err := derivedEscrowLiabilityFromOrders(gs.Orders)
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
	orders, err := k.getAllBountyOrdersChecked(ctx)
	if err != nil {
		panic(fmt.Sprintf("export sponsorship bounty orders: %v", err))
	}
	fulfillments, err := k.getAllFulfillmentsChecked(ctx)
	if err != nil {
		panic(fmt.Sprintf("export sponsorship fulfillments: %v", err))
	}
	params, err := k.getParamsChecked(ctx)
	if err != nil {
		panic(fmt.Sprintf("export sponsorship params: %v", err))
	}
	nextBountyID, err := k.peekNextBountyIDChecked(ctx)
	if err != nil {
		panic(fmt.Sprintf("export sponsorship counter: %v", err))
	}
	return &types.GenesisState{
		Params:       params,
		Orders:       orders,
		Fulfillments: fulfillments,
		NextBountyId: nextBountyID,
	}
}

func (k Keeper) peekNextBountyID(ctx context.Context) uint64 {
	next, err := k.peekNextBountyIDChecked(ctx)
	if err != nil {
		return 1
	}
	return next
}

func (k Keeper) peekNextBountyIDChecked(ctx context.Context) (uint64, error) {
	kv := k.storeService.OpenKVStore(ctx)
	bz, err := kv.Get(types.BountyCounterKey)
	if err != nil {
		return 0, fmt.Errorf("read bounty counter: %w", err)
	}
	if bz == nil {
		return 1, nil
	}
	if len(bz) != 8 {
		return 0, fmt.Errorf("invalid bounty counter length %d", len(bz))
	}
	counter := binary.BigEndian.Uint64(bz)
	if counter == math.MaxUint64 {
		return 0, fmt.Errorf("bounty id space exhausted")
	}
	return counter + 1, nil
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
