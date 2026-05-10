package types

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
)

// BountyOrderKey returns the KV key for a bounty by id.
func BountyOrderKey(id string) []byte {
	return append(BountyOrderKeyPrefix, []byte(id)...)
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
