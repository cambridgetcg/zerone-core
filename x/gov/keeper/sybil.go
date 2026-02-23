package keeper

import (
	"encoding/json"
	"fmt"
	"math/big"

	storetypes "cosmossdk.io/store/types"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/x/gov/types"
)

// --- Sybil Detection Types (hand-written, JSON-serialized) ---

// FundingRecord tracks a sender->recipient funding relationship for sybil detection.
type FundingRecord struct {
	Sender        string `json:"sender"`
	Recipient     string `json:"recipient"`
	TotalAmount   string `json:"total_amount"`
	FirstBlock    uint64 `json:"first_block"`
	LastBlock     uint64 `json:"last_block"`
	TransferCount uint32 `json:"transfer_count"`
}

// SybilParams configures the sybil vote-weight decay system.
type SybilParams struct {
	Enabled           bool   `json:"enabled"`
	CorrelationWindow uint64 `json:"correlation_window"`   // blocks to look back (default: 480000 ~14 days)
	DecayPerSourceBPS uint64 `json:"decay_per_source_bps"` // decay per shared source (default: 2000 = 20%)
	MinPowerBPS       uint64 `json:"min_power_bps"`        // floor (default: 1000 = 10%)
}

// DefaultSybilParams returns the default sybil detection parameters.
func DefaultSybilParams() SybilParams {
	return SybilParams{
		Enabled:           true,
		CorrelationWindow: 480000, // ~14 days at 2521ms blocks
		DecayPerSourceBPS: 2000,   // 20% per shared funding source
		MinPowerBPS:       1000,   // 10% floor
	}
}

// SybilGenesis holds the supplementary genesis state for sybil detection.
type SybilGenesis struct {
	Params         SybilParams     `json:"params"`
	FundingRecords []FundingRecord `json:"funding_records"`
}

// --- Key Helpers ---

// fundingRecordKey returns the store key for a specific sender->recipient funding record.
// Format: 0x0A | recipient | "/" | sender
func fundingRecordKey(recipient, sender string) []byte {
	return append(types.FundingRecordKeyPrefix, []byte(fmt.Sprintf("%s/%s", recipient, sender))...)
}

// fundingRecordRecipientPrefix returns the prefix for all funding records for a recipient.
// Format: 0x0A | recipient | "/"
func fundingRecordRecipientPrefix(recipient string) []byte {
	return append(types.FundingRecordKeyPrefix, []byte(recipient+"/")...)
}

// --- FundingRecord CRUD ---

// RecordFunding records or updates a sender->recipient funding relationship.
// Satisfies the types.FundingRecorder interface.
func (k Keeper) RecordFunding(ctx sdk.Context, sender, recipient, amount string, blockHeight uint64) {
	store := ctx.KVStore(k.storeKey)
	key := fundingRecordKey(recipient, sender)

	var record FundingRecord
	bz := store.Get(key)
	if bz != nil {
		if err := json.Unmarshal(bz, &record); err == nil {
			record.TotalAmount = addBigIntStrings(record.TotalAmount, amount)
			record.LastBlock = blockHeight
			record.TransferCount++
		} else {
			record = FundingRecord{
				Sender:        sender,
				Recipient:     recipient,
				TotalAmount:   amount,
				FirstBlock:    blockHeight,
				LastBlock:     blockHeight,
				TransferCount: 1,
			}
		}
	} else {
		record = FundingRecord{
			Sender:        sender,
			Recipient:     recipient,
			TotalAmount:   amount,
			FirstBlock:    blockHeight,
			LastBlock:     blockHeight,
			TransferCount: 1,
		}
	}

	bz, err := json.Marshal(record)
	if err != nil {
		panic("failed to marshal funding record: " + err.Error())
	}
	store.Set(key, bz)
}

// GetFundingSources returns all funding sources for a recipient within the correlation window.
func (k Keeper) GetFundingSources(ctx sdk.Context, recipient string, currentBlock, windowBlocks uint64) []FundingRecord {
	store := ctx.KVStore(k.storeKey)
	prefix := fundingRecordRecipientPrefix(recipient)
	iter := storetypes.KVStorePrefixIterator(store, prefix)
	defer iter.Close()

	var records []FundingRecord
	for ; iter.Valid(); iter.Next() {
		var record FundingRecord
		if err := json.Unmarshal(iter.Value(), &record); err != nil {
			continue
		}
		if currentBlock > windowBlocks && record.LastBlock < currentBlock-windowBlocks {
			continue
		}
		records = append(records, record)
	}
	return records
}

// GetAllFundingRecords returns all stored funding records.
func (k Keeper) GetAllFundingRecords(ctx sdk.Context) []FundingRecord {
	store := ctx.KVStore(k.storeKey)
	iter := storetypes.KVStorePrefixIterator(store, types.FundingRecordKeyPrefix)
	defer iter.Close()

	var records []FundingRecord
	for ; iter.Valid(); iter.Next() {
		var record FundingRecord
		if err := json.Unmarshal(iter.Value(), &record); err != nil {
			continue
		}
		records = append(records, record)
	}
	return records
}

// SetFundingRecord stores a funding record directly (used for genesis import).
func (k Keeper) SetFundingRecord(ctx sdk.Context, record FundingRecord) {
	store := ctx.KVStore(k.storeKey)
	key := fundingRecordKey(record.Recipient, record.Sender)
	bz, err := json.Marshal(record)
	if err != nil {
		panic("failed to marshal funding record: " + err.Error())
	}
	store.Set(key, bz)
}

// --- SybilParams CRUD ---

// GetSybilParams returns the sybil detection parameters.
func (k Keeper) GetSybilParams(ctx sdk.Context) SybilParams {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.SybilParamsKey)
	if bz == nil {
		return DefaultSybilParams()
	}
	var params SybilParams
	if err := json.Unmarshal(bz, &params); err != nil {
		return DefaultSybilParams()
	}
	return params
}

// SetSybilParams stores the sybil detection parameters.
func (k Keeper) SetSybilParams(ctx sdk.Context, params SybilParams) {
	store := ctx.KVStore(k.storeKey)
	bz, err := json.Marshal(params)
	if err != nil {
		panic("failed to marshal sybil params: " + err.Error())
	}
	store.Set(types.SybilParamsKey, bz)
}

// --- Correlation Engine ---

// ComputeSybilDecayBPS computes the sybil decay multiplier in basis points (10000 = 100%).
// Returns a value in [MinPowerBPS, 10000] to multiply against quadratic voting power.
func (k Keeper) ComputeSybilDecayBPS(ctx sdk.Context, voter string, lipID string) uint64 {
	params := k.GetSybilParams(ctx)
	if !params.Enabled {
		return 10000
	}

	currentBlock := uint64(ctx.BlockHeight())

	voterSources := k.GetFundingSources(ctx, voter, currentBlock, params.CorrelationWindow)
	if len(voterSources) == 0 {
		return 10000
	}

	voterSourceSet := make(map[string]struct{}, len(voterSources))
	for _, src := range voterSources {
		voterSourceSet[src.Sender] = struct{}{}
	}

	existingVotes := k.GetVotesForLIP(ctx, lipID)
	if len(existingVotes) == 0 {
		return 10000
	}

	var maxShared uint64
	for _, vote := range existingVotes {
		otherSources := k.GetFundingSources(ctx, vote.Voter, currentBlock, params.CorrelationWindow)
		var shared uint64
		for _, src := range otherSources {
			if _, ok := voterSourceSet[src.Sender]; ok {
				shared++
			}
		}
		if shared > maxShared {
			maxShared = shared
		}
	}

	if maxShared == 0 {
		return 10000
	}

	decay := maxShared * params.DecayPerSourceBPS
	if decay >= 10000 {
		return params.MinPowerBPS
	}

	result := 10000 - decay
	if result < params.MinPowerBPS {
		return params.MinPowerBPS
	}
	return result
}

// --- Sybil Genesis ---

// ExportSybilGenesis exports the sybil detection state as JSON.
func (k Keeper) ExportSybilGenesis(ctx sdk.Context) json.RawMessage {
	sg := SybilGenesis{
		Params:         k.GetSybilParams(ctx),
		FundingRecords: k.GetAllFundingRecords(ctx),
	}
	if sg.FundingRecords == nil {
		sg.FundingRecords = []FundingRecord{}
	}
	bz, err := json.Marshal(sg)
	if err != nil {
		panic("failed to marshal sybil genesis: " + err.Error())
	}
	return bz
}

// InitSybilGenesis restores sybil detection state from JSON.
func (k Keeper) InitSybilGenesis(ctx sdk.Context, bz json.RawMessage) {
	if bz == nil || len(bz) == 0 {
		k.SetSybilParams(ctx, DefaultSybilParams())
		return
	}
	var sg SybilGenesis
	if err := json.Unmarshal(bz, &sg); err != nil {
		k.SetSybilParams(ctx, DefaultSybilParams())
		return
	}
	k.SetSybilParams(ctx, sg.Params)
	for _, record := range sg.FundingRecords {
		k.SetFundingRecord(ctx, record)
	}
}

// --- Helpers ---

// addBigIntStrings adds two big integer strings. Returns "0" on parse failure.
func addBigIntStrings(a, b string) string {
	ai := new(big.Int)
	bi := new(big.Int)
	ai.SetString(a, 10)
	bi.SetString(b, 10)
	return new(big.Int).Add(ai, bi).String()
}
