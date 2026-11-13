package types

const (
	ModuleName = "capture_challenge"
	StoreKey   = ModuleName
)

var (
	ParamsKey             = []byte{0x00}
	ChallengeKeyPrefix    = []byte{0x01}
	BountyPoolKeyPrefix   = []byte{0x02}
	DomainIndexPrefix     = []byte{0x03}
	PausedDomainKeyPrefix = []byte{0x04}
	// ChallengeAccountingEnabledKey is the consensus-repairs-v1 marker for
	// zerone-core #121 (keeper/challenge_accounting_activation.go).
	ChallengeAccountingEnabledKey = []byte{0x05}
)
