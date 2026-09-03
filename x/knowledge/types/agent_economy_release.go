package types

// AgentEconomyUpgradeMarker is the app-owned activation receipt for the
// reserved knowledge v6->v7 and sponsorship v1->v2 transition. Module
// migrations deliberately do not write this marker: only the future,
// separately reviewed agent-economy upgrade handler may do so.
const AgentEconomyUpgradeMarker = "upgrade_marker_agent-economy-v1"

// AgentEconomyNativeMarker is reserved for an explicitly reviewed native
// genesis profile. Default/native InitGenesis does not write it.
const AgentEconomyNativeMarker = "chain_lineage_native_agent-economy-v1"

const AgentEconomyActivationValue = "migrated"

const (
	AgentEconomyLineageNone    = "none"
	AgentEconomyLineageUpgrade = "upgrade"
	AgentEconomyLineageNative  = "native"
)
