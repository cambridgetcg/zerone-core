package types

const (
	ModuleName = "ibcratelimit"
	// StoreKey must NOT use ModuleName because "ibcratelimit" starts with "ibc",
	// which collides with the IBC module's store key prefix.
	StoreKey = "zerone_ibcrl"
)

var (
	ParamsKey          = []byte{0x00}
	RateLimitKeyPrefix = []byte{0x01}
	PacketFlowPrefix   = []byte{0x02}
)
