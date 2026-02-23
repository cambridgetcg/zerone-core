package types

const (
	ModuleName = "autopoiesis"
	StoreKey   = ModuleName
)

// KV store key prefixes.
var (
	ParamsKey          = []byte{0x01}
	StateKey           = []byte{0x02}
	MultiplierPrefix   = []byte{0x03}
	FrozenPrefix       = []byte{0x04}
	SnapshotPrefix     = []byte{0x05}
	SSIKey             = []byte{0x06}
)

// MultiplierKey returns the store key for a multiplier by path.
func MultiplierKey(path string) []byte {
	return append(MultiplierPrefix, []byte(path)...)
}

// FrozenKey returns the store key for a frozen flag by path.
func FrozenKey(path string) []byte {
	return append(FrozenPrefix, []byte(path)...)
}

// SnapshotKey returns the store key for an epoch snapshot.
// Uses big-endian encoding for ordered iteration.
func SnapshotKey(epoch uint64) []byte {
	key := make([]byte, len(SnapshotPrefix)+8)
	copy(key, SnapshotPrefix)
	key[len(SnapshotPrefix)] = byte(epoch >> 56)
	key[len(SnapshotPrefix)+1] = byte(epoch >> 48)
	key[len(SnapshotPrefix)+2] = byte(epoch >> 40)
	key[len(SnapshotPrefix)+3] = byte(epoch >> 32)
	key[len(SnapshotPrefix)+4] = byte(epoch >> 24)
	key[len(SnapshotPrefix)+5] = byte(epoch >> 16)
	key[len(SnapshotPrefix)+6] = byte(epoch >> 8)
	key[len(SnapshotPrefix)+7] = byte(epoch)
	return key
}
