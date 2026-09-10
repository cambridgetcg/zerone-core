package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// These bounds apply to offline exports and the one-time activation inventory,
// not ordinary claim admission. Refusal is explicit; records are never skipped.
const recordInventoryMaxRecords = 100_000
const recordInventoryMaxBytes = 64 << 20

func hasUnknownRecordFields(message protoreflect.Message) bool {
	if len(message.GetUnknown()) != 0 {
		return true
	}
	unknown := false
	message.Range(func(field protoreflect.FieldDescriptor, value protoreflect.Value) bool {
		if field.IsList() && field.Message() != nil {
			list := value.List()
			for i := 0; i < list.Len(); i++ {
				if hasUnknownRecordFields(list.Get(i).Message()) {
					unknown = true
					return false
				}
			}
		} else if field.IsMap() && field.MapValue().Message() != nil {
			value.Map().Range(func(_ protoreflect.MapKey, item protoreflect.Value) bool {
				unknown = hasUnknownRecordFields(item.Message())
				return !unknown
			})
		} else if !field.IsMap() && !field.IsList() && field.Message() != nil {
			unknown = hasUnknownRecordFields(value.Message())
		}
		return !unknown
	})
	return unknown
}

func (k Keeper) scanRecordPrimary(ctx context.Context, prefix []byte, makeRecord func() proto.Message, visit func(proto.Message, string) error) error {
	iterator, err := k.storeService.OpenKVStore(ctx).Iterator(prefix, prefixEndBytes(prefix))
	if err != nil {
		return err
	}
	count, size := 0, 0
	for ; iterator.Valid(); iterator.Next() {
		key, value := iterator.Key(), iterator.Value()
		count++
		size += len(key) + len(value)
		if count > recordInventoryMaxRecords || size > recordInventoryMaxBytes || len(key) <= len(prefix) {
			_ = iterator.Close()
			return fmt.Errorf("record inventory exceeds bounds or has empty primary identity")
		}
		record := makeRecord()
		if err := proto.Unmarshal(value, record); err != nil {
			_ = iterator.Close()
			return fmt.Errorf("decode record primary %x: %w", key, err)
		}
		if hasUnknownRecordFields(record.ProtoReflect()) {
			_ = iterator.Close()
			return fmt.Errorf("record primary %x has unsupported fields", key)
		}
		if err := visit(record, string(key[len(prefix):])); err != nil {
			_ = iterator.Close()
			return err
		}
	}
	// Also handles only the exact SDK cache-iterator exhaustion sentinels.
	return closeSurvivalIterator(iterator)
}

func (k Keeper) GetAllClaimsChecked(ctx context.Context) ([]*types.Claim, error) {
	var claims []*types.Claim
	err := k.scanRecordPrimary(ctx, types.ClaimKeyPrefix, func() proto.Message { return &types.Claim{} }, func(record proto.Message, id string) error {
		claim := record.(*types.Claim)
		if claim.Id != id {
			return fmt.Errorf("claim primary key does not match payload")
		}
		claims = append(claims, claim)
		return nil
	})
	return claims, err
}

func (k Keeper) GetAllVerificationRoundsChecked(ctx context.Context) ([]*types.VerificationRound, error) {
	enabled, err := k.RecordIntegrityEnabled(ctx)
	if err != nil {
		return nil, err
	}
	var rounds []*types.VerificationRound
	err = k.scanRecordPrimary(ctx, types.VerificationRoundKeyPrefix, func() proto.Message { return &types.VerificationRound{} }, func(record proto.Message, id string) error {
		round := record.(*types.VerificationRound)
		if round.Id != id {
			return fmt.Errorf("round primary key does not match payload")
		}
		if err := types.ValidateVerificationRoundRecord(round, enabled); err != nil {
			return fmt.Errorf("round %s: %w", id, err)
		}
		rounds = append(rounds, round)
		return nil
	})
	return rounds, err
}

func (k Keeper) ValidateRecordIntegrityActivation(ctx context.Context) error {
	enabled, err := k.RecordIntegrityEnabled(ctx)
	if err != nil {
		return err
	}
	if enabled {
		return fmt.Errorf("record integrity activation requires unenabled predecessor")
	}
	// Under the absent marker, typed validation rejects any preseeded scheme-2
	// round or new payout plan. No historical record is rewritten or relabeled.
	_, err = k.GetAllVerificationRoundsChecked(ctx)
	return err
}

// importVerificationRounds restores all supplied primary records. It does not
// run the runtime create path, which properly forbids replacing a claim's
// selected round. Historical non-selected rounds remain available as history.
func (k Keeper) importVerificationRounds(ctx context.Context, gs *types.GenesisState) error {
	if err := types.ValidateGenesisRounds(gs); err != nil {
		return err
	}
	cache, write := sdk.UnwrapSDKContext(ctx).CacheContext()
	store := k.storeService.OpenKVStore(cache)
	byID := make(map[string]*types.VerificationRound)
	uniqueByClaim := make(map[string]*types.VerificationRound)
	for _, collection := range [][]*types.VerificationRound{gs.ActiveRounds, gs.CompletedRounds} {
		for _, round := range collection {
			bz, err := marshalOpts.Marshal(round)
			if err != nil {
				return err
			}
			if err := store.Set(types.RoundKey(round.Id), bz); err != nil {
				return err
			}
			if round.Phase != types.VerificationPhase_VERIFICATION_PHASE_COMPLETE && round.Phase != types.VerificationPhase_VERIFICATION_PHASE_EXPIRED {
				if err := store.Set(activeRoundKey(round.Id), []byte{1}); err != nil {
					return err
				}
			}
			byID[round.Id] = round
			uniqueByClaim[round.ClaimId] = round
		}
	}
	for _, claim := range gs.PendingClaims {
		selected := byID[claim.VerificationRoundId]
		if selected == nil && claim.VerificationRoundId == "" {
			// Validation proves this is the sole provided round for the claim.
			selected = uniqueByClaim[claim.Id]
		}
		if selected != nil {
			if err := store.Set(types.ClaimRoundIndexKey(claim.Id), []byte(selected.Id)); err != nil {
				return err
			}
		}
	}
	write()
	return nil
}
