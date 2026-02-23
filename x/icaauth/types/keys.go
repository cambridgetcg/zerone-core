package types

const (
	ModuleName = "icaauth"
	StoreKey   = ModuleName
)

var (
	ParamsKey          = []byte{0x00}
	RecordKeyPrefix    = []byte{0x01} // records by owner
)
