package types

const (
	// ModuleName is the schedule module's name.
	ModuleName = "schedule"

	// StoreKey is the store key for the schedule module.
	StoreKey = ModuleName
)

// KV store key prefixes.
var (
	ParamsKey            = []byte{0x00}
	ProcessKeyPrefix     = []byte{0x01}
	ByTimeIndexPrefix    = []byte{0x02}
	ByAccountIndexPrefix = []byte{0x03}
	ByStatusIndexPrefix  = []byte{0x04}
	SequenceKey          = []byte{0x05}
)
