package types

const (
	// ModuleName is the discovery module's name.
	ModuleName = "discovery"

	// StoreKey is the store key for the discovery module.
	StoreKey = ModuleName
)

// KV store key prefixes.
var (
	ParamsKey               = []byte{0x00}
	ProfileKeyPrefix        = []byte{0x01}
	ByDomainIndexPrefix     = []byte{0x02}
	ByCapabilityIndexPrefix = []byte{0x03}
)
