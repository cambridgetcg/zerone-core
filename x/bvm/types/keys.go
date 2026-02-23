package types

import "encoding/binary"

const (
	ModuleName   = "bvm"
	StoreKey     = ModuleName
	RouterKey    = ModuleName
	MemStoreKey  = "mem_" + ModuleName
	QuerierRoute = ModuleName
)

// Store key prefixes.
var (
	ContractKeyPrefix = []byte{0x01}
	CodeKeyPrefix     = []byte{0x02}
	StateKeyPrefix    = []byte{0x03}
	ScheduleKeyPrefix = []byte{0x04}
	ParamsKey         = []byte{0x05}
	GasScheduleKey    = []byte{0x06}

	// Index prefixes
	ContractCreatorIndexPrefix  = []byte{0x10}
	ScheduleBlockIndexPrefix    = []byte{0x11}
	ScheduleCapabilityKeyPrefix = []byte{0x12}

	// Counter keys
	ContractCounterKey = []byte{0x20}
	ScheduleCounterKey = []byte{0x21}
)

func ContractKey(address string) []byte {
	return append(ContractKeyPrefix, []byte(address)...)
}

func CodeKey(codeHash string) []byte {
	return append(CodeKeyPrefix, []byte(codeHash)...)
}

func ContractStateKey(contractAddress, key string) []byte {
	return lengthPrefixedKey(StateKeyPrefix, contractAddress, key)
}

func ContractStatePrefix(contractAddress string) []byte {
	return lengthPrefixedPrefix(StateKeyPrefix, contractAddress)
}

func ScheduleKey(scheduleId string) []byte {
	return append(ScheduleKeyPrefix, []byte(scheduleId)...)
}

func ContractCreatorIndexKey(creator, address string) []byte {
	return lengthPrefixedKey(ContractCreatorIndexPrefix, creator, address)
}

func ContractCreatorPrefix(creator string) []byte {
	return lengthPrefixedPrefix(ContractCreatorIndexPrefix, creator)
}

func ScheduleBlockIndexKey(block uint64, scheduleId string) []byte {
	key := make([]byte, 0, len(ScheduleBlockIndexPrefix)+8+len(scheduleId))
	key = append(key, ScheduleBlockIndexPrefix...)
	key = append(key, Uint64ToBytes(block)...)
	key = append(key, []byte(scheduleId)...)
	return key
}

func ScheduleCapabilityKey(scheduleId string) []byte {
	return append(ScheduleCapabilityKeyPrefix, []byte(scheduleId)...)
}

func Uint64ToBytes(v uint64) []byte {
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, v)
	return bz
}

func lengthPrefixedKey(prefix []byte, a, b string) []byte {
	aBytes := []byte(a)
	key := make([]byte, 0, len(prefix)+1+len(aBytes)+len(b))
	key = append(key, prefix...)
	key = append(key, byte(len(aBytes)))
	key = append(key, aBytes...)
	key = append(key, []byte(b)...)
	return key
}

func lengthPrefixedPrefix(prefix []byte, a string) []byte {
	aBytes := []byte(a)
	key := make([]byte, 0, len(prefix)+1+len(aBytes))
	key = append(key, prefix...)
	key = append(key, byte(len(aBytes)))
	key = append(key, aBytes...)
	return key
}
