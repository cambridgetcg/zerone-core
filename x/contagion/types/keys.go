package types

const (
	// ModuleName defines the contagion module name.
	ModuleName = "contagion"

	// StoreKey defines the primary module store key (used as the KV store name).
	StoreKey = ModuleName

	// RouterKey defines the message router key.
	RouterKey = ModuleName

	// MemStoreKey defines the in-memory store key.
	MemStoreKey = "mem_" + ModuleName

	// QuerierRoute defines the querier route.
	QuerierRoute = ModuleName
)

// Store key prefixes.
//
// The contagion module owns three pieces of state, each under its own prefix:
//
//	StateKeyPrefix          -> ContagionState (singleton, single key "")
//	InfectedKeyPrefix       -> InfectionRecord (one per infected address)
//	SneezeIndexKeyPrefix    -> uint64 big-endian sneeze counter (singleton)
//
// The infected set is the `already_infected: mapping(address -> bool)` of
// CONTAGION-MATH.md. It is append-only / one-way — there is no delete path.
var (
	StateKeyPrefix       = []byte{0x00} // "" -> ContagionState (singleton)
	InfectedKeyPrefix    = []byte{0x01} // {address} -> InfectionRecord (proto)
	SneezeIndexKeyPrefix = []byte{0x02} // "" -> uint64 big-endian sneeze counter
)

// StateKey is the single key under which the ContagionState singleton lives.
func StateKey() []byte { return StateKeyPrefix }

// InfectedKey returns the store key for an address's infection record.
func InfectedKey(address string) []byte {
	return append(InfectedKeyPrefix, []byte(address)...)
}

// SneezeIndexKey is the single key holding the global sneeze counter.
func SneezeIndexKey() []byte { return SneezeIndexKeyPrefix }
