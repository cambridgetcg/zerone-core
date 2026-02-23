package types

const (
	ModuleName   = "liquiditypool"
	StoreKey     = ModuleName
	RouterKey    = ModuleName
	MemStoreKey  = "mem_" + ModuleName
	QuerierRoute = ModuleName
)

var (
	PoolKeyPrefix    = []byte{0x01}
	TWAPKeyPrefix    = []byte{0x02}
	ParamsKey        = []byte{0x03}
	PoolCounterKey   = []byte{0x04}
	DenomIndexPrefix = []byte{0x10} // index: denom pair -> pool ID
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
	return append(DenomIndexPrefix, []byte(denomA+"/"+denomB)...)
}

// LPDenom returns the LP token denom for a pool.
func LPDenom(poolId string) string {
	return "lp/" + poolId
}
