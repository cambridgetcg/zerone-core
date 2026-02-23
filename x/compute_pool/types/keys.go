package types

const (
	// ModuleName is the compute_pool module's name.
	ModuleName = "compute_pool"

	// StoreKey is the store key for the compute_pool module.
	StoreKey = ModuleName
)

// KV store key prefixes.
var (
	ParamsKey         = []byte{0x00}
	ProviderKeyPrefix = []byte{0x01}
	CreditKeyPrefix   = []byte{0x02}
)
