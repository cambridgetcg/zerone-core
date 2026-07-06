package types

import (
	"encoding/binary"
)

const (
	ModuleName  = "zerone_auth"
	StoreKey    = ModuleName
	RouterKey   = ModuleName
	MemStoreKey = "mem_" + ModuleName
)

var (
	// AccountKeyPrefix is the prefix for account storage.
	AccountKeyPrefix = []byte{0x01}

	// DIDMappingPrefix is the prefix for DID -> bech32 mapping.
	DIDMappingPrefix = []byte{0x02}

	// 0x03 was SessionKeyPrefix (sessions removed in the 2026-07 slim cut;
	// do not reuse).

	// ParamsKey is the key for module parameters.
	ParamsKey = []byte{0x04}

	// LastRotationPrefix is the prefix for tracking last key rotation.
	LastRotationPrefix = []byte{0x05}

	// 0x06-0x09 were RecoveryRequestPrefix, RecoveryConfigPrefix,
	// BootstrapClaimPrefix, and RecoveryShardPrefix (social recovery and
	// the dormant bootstrap auto-claim removed in the 2026-07 slim cut;
	// do not reuse).
)

// AccountKey returns the store key for an account by bech32 address.
func AccountKey(address string) []byte {
	return append(AccountKeyPrefix, []byte(address)...)
}

// DIDMappingKey returns the store key for DID mapping.
func DIDMappingKey(did string) []byte {
	return append(DIDMappingPrefix, []byte(did)...)
}

// LastRotationKey returns the store key for last rotation timestamp.
func LastRotationKey(address string) []byte {
	return append(LastRotationPrefix, []byte(address)...)
}

// Uint32ToBytes converts uint32 to 4 bytes (big-endian).
func Uint32ToBytes(n uint32) []byte {
	b := make([]byte, 4)
	b[0] = byte(n >> 24)
	b[1] = byte(n >> 16)
	b[2] = byte(n >> 8)
	b[3] = byte(n)
	return b
}

// Uint64ToBytes converts uint64 to bytes for storage.
func Uint64ToBytes(n uint64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, n)
	return b
}

// BytesToUint64 converts bytes to uint64 from storage.
func BytesToUint64(b []byte) uint64 {
	if len(b) != 8 {
		return 0
	}
	return binary.BigEndian.Uint64(b)
}
