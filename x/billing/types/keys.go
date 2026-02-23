package types

const (
	// ModuleName is the billing module's name.
	ModuleName = "billing"

	// StoreKey is the store key for the billing module.
	StoreKey = ModuleName
)

// KV store key prefixes.
var (
	ParamsKey              = []byte{0x00}
	ProviderKeyPrefix      = []byte{0x01}
	DomainIndexPrefix      = []byte{0x02}
	DynamicPricingConfigKey = []byte{0x03}
	LastEmittedPriceKey    = []byte{0x04}
)
