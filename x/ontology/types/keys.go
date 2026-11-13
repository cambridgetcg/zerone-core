package types

const (
	ModuleName  = "zerone_ontology"
	StoreKey    = ModuleName
	RouterKey   = ModuleName
	MemStoreKey = "mem_" + ModuleName
	QuerierRoute = ModuleName
)

var (
	ParamsKey                  = []byte{0x00}
	StratumKeyPrefix           = []byte{0x01}
	DomainKeyPrefix            = []byte{0x02}
	ProposalKeyPrefix          = []byte{0x03}
	LinkKeyPrefix              = []byte{0x04}
	DomainByStratumPrefix      = []byte{0x05}
	LogicZoneKeyPrefix         = []byte{0x10}
	IncompletenessAckKeyPrefix = []byte{0x11}

	// ActiveDomainCountEnabledKey is the consensus-repairs-v1 marker for
	// zerone-core #124 (keeper/active_domain_count_activation.go). 0x06-0x0F
	// and 0x12+ are unused; 0x06 follows the last contiguous prefix.
	ActiveDomainCountEnabledKey = []byte{0x06}
)
