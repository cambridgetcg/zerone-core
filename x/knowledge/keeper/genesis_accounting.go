package keeper

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"

	corestore "cosmossdk.io/core/store"
	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/protobuf/proto"

	"github.com/zerone-chain/zerone/x/knowledge/types"
)

var genesisAccountingPrefixes = [][]byte{
	types.TrainingFundEscrowLockedKeyPrefix,
	types.BootstrapClaimCountPrefix,
	types.BootstrapEpochCountPrefix,
}

// walkGenesisAccountingPrefix is export/import-only. Do not tighten live
// iterators as a side effect of a genesis repair. In particular, empty scans,
// iteration failures and Close failures must never become partial inventories.
func walkGenesisAccountingPrefix(store corestore.KVStore, prefix []byte, visit func([]byte, []byte) error) (err error) {
	it, err := store.Iterator(prefix, prefixEndBytes(prefix))
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, it.Close()) }()
	for ; it.Valid(); it.Next() {
		if err := visit(it.Key(), it.Value()); err != nil {
			return err
		}
	}
	return historyIteratorError(it)
}

func (k Keeper) validateGenesisAccountingBacking(ctx context.Context, gs *types.GenesisState) error {
	if k.bankKeeper == nil {
		return fmt.Errorf("accounting_state requires the bank keeper")
	}
	// ValidateGenesisAccounting has already checked this canonical amount.
	allocation, _ := sdkmath.NewIntFromString(gs.TrainingFundAllocation)
	addr := sdk.AccAddress(authtypesNewModuleAddress(types.TrainingFundModuleName))
	balance := k.bankKeeper.GetBalance(ctx, addr, "uzrn").Amount
	if balance.IsNil() || !balance.Equal(allocation) {
		return fmt.Errorf("accounting_state training allocation must equal the imported bank balance")
	}
	return nil
}

func (k Keeper) exportGenesisAccounting(ctx context.Context, gs *types.GenesisState) error {
	store := k.storeService.OpenKVStore(ctx)
	state := &types.KnowledgeGenesisAccounting{}
	for _, prefix := range genesisAccountingPrefixes {
		err := walkGenesisAccountingPrefix(store, prefix, func(key, value []byte) error {
			suffix := key[len(prefix):]
			switch {
			case bytes.Equal(prefix, types.TrainingFundEscrowLockedKeyPrefix):
				state.AugmentationEscrows = append(state.AugmentationEscrows, &types.AugmentationEscrowBalance{
					BountyId: string(suffix), RemainingUzrn: string(value),
				})
			case bytes.Equal(prefix, types.BootstrapClaimCountPrefix):
				if len(value) != 8 {
					return fmt.Errorf("bootstrap address count must have exactly eight value bytes")
				}
				state.BootstrapAddressCounts = append(state.BootstrapAddressCounts, &types.BootstrapAddressCount{
					Address: string(suffix), Count: binary.BigEndian.Uint64(value),
				})
			case bytes.Equal(prefix, types.BootstrapEpochCountPrefix):
				if len(suffix) != 8 || len(value) != 8 {
					return fmt.Errorf("bootstrap epoch count must have exactly eight key and value bytes")
				}
				state.BootstrapEpochCounts = append(state.BootstrapEpochCounts, &types.BootstrapEpochCount{
					Epoch: binary.BigEndian.Uint64(suffix), Count: binary.BigEndian.Uint64(value),
				})
			}
			return nil
		})
		if err != nil {
			return err
		}
	}
	if k.bankKeeper == nil {
		// Dependency-less keepers cannot attest a bank inventory. Preserve the
		// old absence only when all three primary inventories are empty; never
		// drop a stored counter/escrow or label an unknown balance as zero.
		// The application always supplies bank; a present import requires it.
		if len(state.AugmentationEscrows)+len(state.BootstrapAddressCounts)+len(state.BootstrapEpochCounts) != 0 {
			return fmt.Errorf("cannot export accounting inventory without bank keeper")
		}
		return nil
	}
	// The old bounty iterator skips malformed records and errors. An accounting
	// inventory must carry every owner, including inactive retained escrows.
	var bounties []*types.AugmentationBounty
	if err := walkGenesisAccountingPrefix(store, types.AugmentationBountyKeyPrefix, func(key, value []byte) error {
		bounty := new(types.AugmentationBounty)
		if err := proto.Unmarshal(value, bounty); err != nil {
			return fmt.Errorf("decode accounting bounty: %w", err)
		}
		canonical, err := marshalOpts.Marshal(bounty)
		if err != nil {
			return err
		}
		if len(bounty.ProtoReflect().GetUnknown()) != 0 || !bytes.Equal(value, canonical) || !bytes.Equal(key, types.AugmentationBountyKey(bounty.Id)) {
			return fmt.Errorf("noncanonical accounting bounty or key/payload mismatch")
		}
		bounties = append(bounties, bounty)
		return nil
	}); err != nil {
		return err
	}
	balance, ok := sdkmath.NewIntFromString(gs.TrainingFundAllocation)
	if !ok || balance.IsNegative() {
		return fmt.Errorf("invalid training fund balance")
	}
	nonEscrow := balance
	for _, entry := range state.AugmentationEscrows {
		amount, ok := sdkmath.NewIntFromString(entry.RemainingUzrn)
		if !ok || amount.IsNegative() || amount.String() != entry.RemainingUzrn {
			return fmt.Errorf("noncanonical accounting escrow amount")
		}
		if amount.GT(nonEscrow) {
			return fmt.Errorf("accounting escrow exceeds training fund bank balance")
		}
		nonEscrow = nonEscrow.Sub(amount)
	}
	state.TrainingFundNonEscrowBalance = nonEscrow.String()
	gs.AccountingState = state
	gs.AugmentationBounties = bounties
	return types.ValidateGenesisAccounting(gs)
}

// importGenesisAccounting replaces only these primary inventories. The caller
// supplies an outer SDK cache covering all genesis writes (including bank).
func (k Keeper) importGenesisAccounting(ctx context.Context, state *types.KnowledgeGenesisAccounting) error {
	store := k.storeService.OpenKVStore(ctx)
	for _, prefix := range genesisAccountingPrefixes {
		var keys [][]byte
		if err := walkGenesisAccountingPrefix(store, prefix, func(key, _ []byte) error {
			keys = append(keys, bytes.Clone(key))
			return nil
		}); err != nil {
			return err
		}
		for _, key := range keys {
			if err := store.Delete(key); err != nil {
				return err
			}
		}
	}
	for _, entry := range state.AugmentationEscrows {
		if err := store.Set(types.TrainingFundEscrowLockedKey(entry.BountyId), []byte(entry.RemainingUzrn)); err != nil {
			return err
		}
	}
	for _, entry := range state.BootstrapAddressCounts {
		key := append(bytes.Clone(types.BootstrapClaimCountPrefix), []byte(entry.Address)...)
		if err := store.Set(key, binary.BigEndian.AppendUint64(nil, entry.Count)); err != nil {
			return err
		}
	}
	for _, entry := range state.BootstrapEpochCounts {
		key := binary.BigEndian.AppendUint64(bytes.Clone(types.BootstrapEpochCountPrefix), entry.Epoch)
		if err := store.Set(key, binary.BigEndian.AppendUint64(nil, entry.Count)); err != nil {
			return err
		}
	}
	return nil
}
