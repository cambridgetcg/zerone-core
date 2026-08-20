package types

import "encoding/binary"

const (
	// ModuleName is the canonical identifier for x/sponsorship.
	ModuleName = "sponsorship"

	// StoreKey is the primary store key under which the module's KV
	// store is mounted.
	StoreKey = ModuleName

	// RouterKey is the message routing key.
	RouterKey = ModuleName

	// QuerierRoute is the query routing key.
	QuerierRoute = ModuleName
)

var (
	// ParamsKey holds the module's serialized Params.
	ParamsKey = []byte{0x00}

	// BountyOrderKeyPrefix is the prefix for BountyOrder records.
	// Layout: BountyOrderKeyPrefix || id
	BountyOrderKeyPrefix = []byte{0x01}

	// FulfillmentKeyPrefix is the prefix for BountyFulfillment records.
	// Layout: FulfillmentKeyPrefix || bounty_id || "/" || fact_id
	FulfillmentKeyPrefix = []byte{0x02}

	// BountyCounterKey holds the monotonically-incrementing next-id counter.
	BountyCounterKey = []byte{0x03}

	// FactConsumptionKeyPrefix permanently marks facts already paid by any
	// sponsorship bounty.
	FactConsumptionKeyPrefix = []byte{0x04}

	// ReceiptConsumptionKeyPrefix permanently marks work receipts already paid
	// by any sponsorship bounty.
	ReceiptConsumptionKeyPrefix = []byte{0x05}

	// SettlementNullifierKeyPrefix indexes the domain-separated immutable
	// work-contract+artifact nullifier for independent audit and replay refusal.
	SettlementNullifierKeyPrefix = []byte{0x06}

	// EscrowLiabilityKey stores the canonical decimal sum of every ACTIVE or
	// EXPIRED order's escrow_remaining. Transaction paths read it in O(1).
	EscrowLiabilityKey = []byte{0x07}

	// DeadlineIndexKeyPrefix indexes ACTIVE orders by big-endian end height.
	DeadlineIndexKeyPrefix = []byte{0x08}

	// ActiveSponsorIndexKeyPrefix indexes a sponsor's bounded active set.
	ActiveSponsorIndexKeyPrefix = []byte{0x09}
)

// BountyOrderKey returns the KV key for a bounty by id.
func BountyOrderKey(id string) []byte {
	return append(BountyOrderKeyPrefix, []byte(id)...)
}

func FactConsumptionKey(factID string) []byte {
	return append(append([]byte{}, FactConsumptionKeyPrefix...), []byte(factID)...)
}

func ReceiptConsumptionKey(receiptHash string) []byte {
	return append(append([]byte{}, ReceiptConsumptionKeyPrefix...), []byte(receiptHash)...)
}

func SettlementNullifierKey(nullifier string) []byte {
	return append(append([]byte{}, SettlementNullifierKeyPrefix...), []byte(nullifier)...)
}

func DeadlineIndexKey(endBlock uint64, bountyID string) []byte {
	key := make([]byte, 1+8+len(bountyID))
	key[0] = DeadlineIndexKeyPrefix[0]
	binary.BigEndian.PutUint64(key[1:9], endBlock)
	copy(key[9:], bountyID)
	return key
}

func ActiveSponsorIndexPrefix(sponsor string) []byte {
	key := append(append([]byte{}, ActiveSponsorIndexKeyPrefix...), []byte(sponsor)...)
	return append(key, 0)
}

func ActiveSponsorIndexKey(sponsor, bountyID string) []byte {
	return append(ActiveSponsorIndexPrefix(sponsor), []byte(bountyID)...)
}

// FulfillmentKey returns the KV key for a (bounty_id, fact_id) fulfillment.
func FulfillmentKey(bountyID, factID string) []byte {
	key := append([]byte{}, FulfillmentKeyPrefix...)
	key = append(key, []byte(bountyID)...)
	key = append(key, '/')
	key = append(key, []byte(factID)...)
	return key
}

// FulfillmentByBountyPrefix returns the iteration prefix for all fulfillments of a bounty.
func FulfillmentByBountyPrefix(bountyID string) []byte {
	prefix := append([]byte{}, FulfillmentKeyPrefix...)
	prefix = append(prefix, []byte(bountyID)...)
	prefix = append(prefix, '/')
	return prefix
}
