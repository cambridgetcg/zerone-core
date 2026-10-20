package keeper

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/x/ibcratelimit/types"
)

// Charge identity is selected only by RefundIdentityEnabled. The entry point
// reads that marker once; these repair helpers never read it. The named
// consensus-repairs-v1 carrier owns activation and the version bump.
//
// SendCharge is an opaque receipt for a successful quota debit. A nil receipt
// means no debit occurred. Its identity is captured before the packet is sent.
// The caller must use the same SDK transaction cache for charging and sending.
type SendCharge struct {
	channelID, denom, amount string
	windowStart              uint64
	ruleGeneration           uint64
}

// packetChargeRecord extends the JSON value at the existing PacketFlow key;
// the public PacketFlow protobuf and its four original fields are unchanged.
// Pointers distinguish a valid zero identity from missing legacy metadata.
type packetChargeRecord struct {
	ChannelID      string  `json:"channel_id"`
	Sequence       uint64  `json:"sequence"`
	Denom          string  `json:"denom"`
	Amount         string  `json:"amount"`
	WindowStart    *uint64 `json:"window_start"`
	RuleGeneration *uint64 `json:"rule_generation"`
}

// Rule generations use 0x03 | channelID | 0x00 | denom and an 8-byte BE value.
// Missing generations denote existing, pre-upgrade rules (generation zero).
// Tombstones survive deletion, so identical same-height re-adds cannot alias.
func ruleGenerationKey(channelID, denom string) []byte {
	key := rateLimitKey(channelID, denom)
	key[0] = types.RuleGenerationPrefix[0]
	return key
}

func (k Keeper) ruleGeneration(ctx context.Context, channelID, denom string) uint64 {
	generation, err := k.readRuleGeneration(ctx, channelID, denom)
	if err != nil {
		panic(err)
	}
	return generation
}

// Refunds skip unusable limiter metadata; charges and installs still fail
// closed through ruleGeneration. Missing legacy keys are valid generation 0.
func (k Keeper) readRuleGeneration(ctx context.Context, channelID, denom string) (uint64, error) {
	bz, err := k.storeService.OpenKVStore(ctx).Get(ruleGenerationKey(channelID, denom))
	if err != nil {
		return 0, fmt.Errorf("failed to read rule generation: %w", err)
	}
	if bz == nil {
		return 0, nil
	}
	if len(bz) != 8 {
		return 0, fmt.Errorf("invalid rule generation encoding")
	}
	return binary.BigEndian.Uint64(bz), nil
}

func (k Keeper) advanceRuleGeneration(ctx context.Context, channelID, denom string) {
	generation := k.ruleGeneration(ctx, channelID, denom)
	if generation == math.MaxUint64 {
		panic("rule generation exhausted")
	}
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, generation+1)
	if err := k.storeService.OpenKVStore(ctx).Set(ruleGenerationKey(channelID, denom), bz); err != nil {
		panic(fmt.Sprintf("failed to write rule generation: %v", err))
	}
}

// RecordPacketCharge persists only actual charges. Legacy SetPacketFlow records
// remain readable but cannot authorize a refund without both identity fields.
func (k Keeper) RecordPacketCharge(ctx context.Context, sequence uint64, charge *SendCharge) {
	if charge == nil {
		return
	}
	record := packetChargeRecord{
		ChannelID: charge.channelID, Sequence: sequence, Denom: charge.denom, Amount: charge.amount,
		WindowStart: &charge.windowStart, RuleGeneration: &charge.ruleGeneration,
	}
	bz, err := json.Marshal(record)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal packet charge: %v", err))
	}
	if err := k.storeService.OpenKVStore(ctx).Set(packetFlowKey(charge.channelID, sequence), bz); err != nil {
		panic(fmt.Sprintf("failed to write packet charge: %v", err))
	}
}

// ReverseSendCharge undoes an immediate failed send only if its original rule
// and unexpired window still own the usage. It deliberately ignores Enabled:
// disabling the limiter after a real charge does not invalidate that charge.
func (k Keeper) ReverseSendCharge(ctx context.Context, charge *SendCharge) {
	if charge == nil {
		return
	}
	rl, found := k.GetRateLimit(ctx, charge.channelID, charge.denom)
	if !found || rl.WindowStart != charge.windowStart {
		return
	}
	generation, err := k.readRuleGeneration(ctx, charge.channelID, charge.denom)
	if err != nil || generation != charge.ruleGeneration {
		return
	}
	height := sdk.UnwrapSDKContext(ctx).BlockHeight()
	// Subtraction avoids overflow at WindowStart + WindowBlocks. Do not reset
	// a window just to refund it; ordinary send/receive/BeginBlock owns resets.
	if height < 0 || uint64(height) < rl.WindowStart || uint64(height)-rl.WindowStart >= rl.WindowBlocks {
		return
	}
	amount, ok := new(big.Int).SetString(charge.amount, 10)
	if !ok || amount.Sign() <= 0 {
		return
	}
	k.reverseSendQuota(ctx, charge.channelID, charge.denom, amount)
}

// ReversePacketFlow checks persisted identity before adjusting quota. Records
// written by the old binary lack this identity and are conservatively ignored:
// they cannot prove a charge or its window/rule. The terminal middleware
// callback still deletes them and forwards the token refund to the IBC app.
// As with all flow records, the caller owns terminal cleanup and SDK rollback.
func (k Keeper) ReversePacketFlow(ctx context.Context, channelID string, sequence uint64) {
	bz, err := k.storeService.OpenKVStore(ctx).Get(packetFlowKey(channelID, sequence))
	if err != nil {
		panic(fmt.Sprintf("failed to read packet charge: %v", err))
	}
	if bz == nil {
		return
	}
	var record packetChargeRecord
	if err := json.Unmarshal(bz, &record); err != nil || record.WindowStart == nil || record.RuleGeneration == nil {
		return
	}
	if record.ChannelID != channelID || record.Sequence != sequence {
		return
	}
	k.ReverseSendCharge(ctx, &SendCharge{
		channelID: record.ChannelID, denom: record.Denom, amount: record.Amount,
		windowStart: *record.WindowStart, ruleGeneration: *record.RuleGeneration,
	})
}
