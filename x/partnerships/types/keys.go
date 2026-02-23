package types

const (
	ModuleName = "partnerships"
	StoreKey   = ModuleName
	RouterKey  = ModuleName
)

// Store key prefixes.
var (
	PartnershipKeyPrefix       = []byte{0x01}
	FormationKeyPrefix         = []byte{0x02}
	ByHumanIndexPrefix         = []byte{0x03}
	ByAgentIndexPrefix         = []byte{0x04}
	ConsensusOpKeyPrefix       = []byte{0x06}
	CoercionSignalKeyPrefix    = []byte{0x07}
	SafetyFreezeKeyPrefix      = []byte{0x08}
	RejectionCooldownKeyPrefix = []byte{0x09}
	ParamsKey                  = []byte{0x0a}
	SequenceKey                = []byte{0x0b}
	SeedPartnershipKeyPrefix   = []byte{0x10}
	PoolEntryKeyPrefix         = []byte{0x11}
	MentorshipKeyPrefix        = []byte{0x13}
	ByDIDSeedIndexPrefix       = []byte{0x14}
)
