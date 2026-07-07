package types

const (
	ModuleName = "claiming_pot"
	StoreKey   = ModuleName
)

// KV store key prefixes.
var (
	ParamsKey       = []byte{0x01}
	PotKeyPrefix    = []byte{0x02}
	ClaimKeyPrefix  = []byte{0x03}
	PotCounterKey   = []byte{0x04}
	ActivePotPrefix = []byte{0x05}

	// BootstrapMintedEntriesKey stores a monotonic uint64 (big-endian):
	// the number of bootstrap entries EVER created (genesis + every
	// MsgAddBootstrapEntry admission). It only increases — pots are never
	// deleted and DEPLETED pots stay in state, so this counter times
	// PerAgentBootstrapUzrn is the lifetime bootstrap emission commitment,
	// enforced against Params.BootstrapEmissionCapUzrn.
	BootstrapMintedEntriesKey = []byte{0x06}

	// BootstrapAdmissionWindowKey stores 16 bytes (big-endian):
	// window index (uint64) || admissions counted in that window (uint64).
	// The window index is block_height / BootstrapAdmissionWindowBlocks; a
	// read for a different (newer) window resets the count to zero.
	BootstrapAdmissionWindowKey = []byte{0x07}
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
