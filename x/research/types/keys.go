package types

const (
	// ModuleName defines the module name.
	ModuleName = "research"

	// StoreKey defines the primary module store key.
	StoreKey = ModuleName

	// RouterKey defines the routing key.
	RouterKey = ModuleName

	// QuerierRoute defines the querier route.
	QuerierRoute = ModuleName
)

// Store key prefixes.
var (
	SubmissionKeyPrefix = []byte{0x01}
	PeerReviewKeyPrefix = []byte{0x02}
	BountyKeyPrefix     = []byte{0x03}
	TreasuryKeyPrefix   = []byte{0x04}
	ParamsKey           = []byte{0x05}
)
