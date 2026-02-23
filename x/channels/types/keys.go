package types

const (
	ModuleName = "channels"
	StoreKey   = ModuleName
	RouterKey  = ModuleName
)

// KV store key prefixes.
var (
	ParamsKey           = []byte{0x00}
	ChannelKeyPrefix    = []byte{0x01}
	DisputeKeyPrefix    = []byte{0x02}
	PayerIndexPrefix    = []byte{0x10}
	ReceiverIndexPrefix = []byte{0x11}
	ChannelCounterKey   = []byte{0x20}
)

// ChannelKey returns the store key for a specific channel.
func ChannelKey(channelId string) []byte {
	return append(ChannelKeyPrefix, []byte(channelId)...)
}

// DisputeKey returns the store key for a dispute.
func DisputeKey(channelId string) []byte {
	return append(DisputeKeyPrefix, []byte(channelId)...)
}

// PayerChannelKey returns the index key for a payer's channel.
func PayerChannelKey(payer, channelId string) []byte {
	return append(PayerIndexPrefix, []byte(payer+"/"+channelId)...)
}

// PayerChannelPrefix returns the prefix for all of a payer's channels.
func PayerChannelPrefix(payer string) []byte {
	return append(PayerIndexPrefix, []byte(payer+"/")...)
}

// ReceiverChannelKey returns the index key for a receiver's channel.
func ReceiverChannelKey(receiver, channelId string) []byte {
	return append(ReceiverIndexPrefix, []byte(receiver+"/"+channelId)...)
}

// ReceiverChannelPrefix returns the prefix for all of a receiver's channels.
func ReceiverChannelPrefix(receiver string) []byte {
	return append(ReceiverIndexPrefix, []byte(receiver+"/")...)
}
