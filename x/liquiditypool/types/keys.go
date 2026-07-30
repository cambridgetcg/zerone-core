package types

import "encoding/binary"

const (
	ModuleName   = "liquiditypool"
	StoreKey     = ModuleName
	RouterKey    = ModuleName
	MemStoreKey  = "mem_" + ModuleName
	QuerierRoute = ModuleName
)

var (
	PoolKeyPrefix            = []byte{0x01}
	TWAPKeyPrefix            = []byte{0x02}
	ParamsKey                = []byte{0x03}
	PoolCounterKey           = []byte{0x04}
	TWAPObservationKeyPrefix = []byte{0x05}
	DenomIndexPrefix         = []byte{0x10} // index: denom pair -> pool ID
	OpenPoolIndexPrefix      = []byte{0x11} // bounded index: non-closed pool ID -> pool ID
	TWAPGarbageCollectPrefix = []byte{0x12} // queue: closed pool ID -> deferred observation cleanup
)

func PoolKey(poolId string) []byte {
	return append(PoolKeyPrefix, []byte(poolId)...)
}

func TWAPKey(poolId string) []byte {
	return append(TWAPKeyPrefix, []byte(poolId)...)
}

// DenomPairKey returns the index key for a denom pair (sorted lexicographically).
func DenomPairKey(denomA, denomB string) []byte {
	if denomA > denomB {
		denomA, denomB = denomB, denomA
	}
	key := make([]byte, 1+4+len(denomA)+len(denomB))
	key[0] = DenomIndexPrefix[0]
	binary.BigEndian.PutUint32(key[1:5], uint32(len(denomA)))
	copy(key[5:], denomA)
	copy(key[5+len(denomA):], denomB)
	return key
}

func OpenPoolIndexKey(poolID string) []byte {
	return append(OpenPoolIndexPrefix, []byte(poolID)...)
}

func TWAPObservationPoolPrefix(poolID string) []byte {
	key := make([]byte, 0, 1+len(poolID)+1)
	key = append(key, TWAPObservationKeyPrefix...)
	key = append(key, poolID...)
	return append(key, 0)
}

func TWAPObservationKey(poolID string, height uint64) []byte {
	key := TWAPObservationPoolPrefix(poolID)
	heightBz := make([]byte, 8)
	binary.BigEndian.PutUint64(heightBz, height)
	return append(key, heightBz...)
}

func TWAPObservationHeight(key []byte) (uint64, bool) {
	if len(key) < 9 {
		return 0, false
	}
	return binary.BigEndian.Uint64(key[len(key)-8:]), true
}

func TWAPGarbageCollectionKey(poolID string) []byte {
	return append(TWAPGarbageCollectPrefix, []byte(poolID)...)
}

// LPDenom returns the LP token denom for a pool.
func LPDenom(poolId string) string {
	return "lp/" + poolId
}
