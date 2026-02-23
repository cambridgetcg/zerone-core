package types

const (
	ModuleName = "claiming_pot"
	StoreKey   = ModuleName
)

// KV store key prefixes.
var (
	ParamsKey         = []byte{0x01}
	PotKeyPrefix      = []byte{0x02}
	ClaimKeyPrefix    = []byte{0x03}
	PotCounterKey     = []byte{0x04}
	ActivePotPrefix   = []byte{0x05}
)

func PotKey(id string) []byte {
	return append(PotKeyPrefix, []byte(id)...)
}

func ClaimKey(potID, claimant string) []byte {
	key := append(ClaimKeyPrefix, []byte(potID)...)
	key = append(key, '/')
	key = append(key, []byte(claimant)...)
	return key
}

func ClaimByPotPrefix(potID string) []byte {
	key := append(ClaimKeyPrefix, []byte(potID)...)
	key = append(key, '/')
	return key
}

func ActivePotKey(id string) []byte {
	return append(ActivePotPrefix, []byte(id)...)
}
