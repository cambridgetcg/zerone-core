package types

const (
	// ModuleName is the toolbox module's name.
	ModuleName = "toolbox"

	// StoreKey is the store key for the toolbox module.
	StoreKey = ModuleName

	// RouterKey is the router key for the toolbox module.
	RouterKey = ModuleName
)

// KV store key prefixes.
var (
	// Primary records
	ParamsKey                    = []byte{0x00}
	ToolKeyPrefix                = []byte{0x01}
	ToolCallKeyPrefix            = []byte{0x02}
	PendingContributorshipPrefix = []byte{0x03}
	ToolCounterKey               = []byte{0x04}

	// Indexes
	ToolByDeployerPrefix  = []byte{0x10}
	ToolByStatusPrefix    = []byte{0x11}
	ToolByCategoryPrefix  = []byte{0x12}
	ToolByTagPrefix       = []byte{0x13}
	AgentToolUsagePrefix  = []byte{0x14}
	AgentActiveToolPrefix = []byte{0x15}

	// Trust engine
	TrustSnapshotPrefix = []byte{0x20}
	CallerRecordPrefix  = []byte{0x21}

	// Dependency graph
	DependencyEdgePrefix    = []byte{0x30}
	ReverseDependencyPrefix = []byte{0x31}

	// Demand tracking
	DemandWindowPrefix = []byte{0x50}
	GlobalDemandKey    = []byte{0x51}

	// Free tier
	FreeAllowancePrefix = []byte{0x60}
)

// ToolKey returns the store key for a specific tool.
func ToolKey(toolID string) []byte {
	return append(ToolKeyPrefix, []byte(toolID)...)
}

// ToolCallKey returns the store key for a specific tool call.
func ToolCallKey(callID string) []byte {
	return append(ToolCallKeyPrefix, []byte(callID)...)
}

// PendingContributorshipKey returns the key for a pending contributorship.
func PendingContributorshipKey(toolID, address string) []byte {
	return append(PendingContributorshipPrefix, []byte(toolID+"/"+address)...)
}

// PendingContributorshipIterPrefix returns the prefix for iterating all pending contributorships of a tool.
func PendingContributorshipIterPrefix(toolID string) []byte {
	return append(PendingContributorshipPrefix, []byte(toolID+"/")...)
}

// ToolByDeployerKey returns the index key {deployer}/{toolID}.
func ToolByDeployerKey(deployer, toolID string) []byte {
	return append(ToolByDeployerPrefix, []byte(deployer+"/"+toolID)...)
}

// ToolByDeployerIterPrefix returns the prefix for iterating tools by deployer.
func ToolByDeployerIterPrefix(deployer string) []byte {
	return append(ToolByDeployerPrefix, []byte(deployer+"/")...)
}

// ToolByStatusKey returns the index key {status}/{toolID}.
func ToolByStatusKey(status, toolID string) []byte {
	return append(ToolByStatusPrefix, []byte(status+"/"+toolID)...)
}

// ToolByStatusIterPrefix returns the prefix for iterating tools by status.
func ToolByStatusIterPrefix(status string) []byte {
	return append(ToolByStatusPrefix, []byte(status+"/")...)
}

// ToolByCategoryKey returns the index key {category}/{toolID}.
func ToolByCategoryKey(category, toolID string) []byte {
	return append(ToolByCategoryPrefix, []byte(category+"/"+toolID)...)
}

// ToolByCategoryIterPrefix returns the prefix for iterating tools by category.
func ToolByCategoryIterPrefix(category string) []byte {
	return append(ToolByCategoryPrefix, []byte(category+"/")...)
}

// ToolByTagKey returns the index key {tag}/{toolID}.
func ToolByTagKey(tag, toolID string) []byte {
	return append(ToolByTagPrefix, []byte(tag+"/"+toolID)...)
}

// ToolByTagIterPrefix returns the prefix for iterating tools by tag.
func ToolByTagIterPrefix(tag string) []byte {
	return append(ToolByTagPrefix, []byte(tag+"/")...)
}

// TrustSnapshotKey returns the key for a tool's trust snapshot.
func TrustSnapshotKey(toolID string) []byte {
	return append(TrustSnapshotPrefix, []byte(toolID)...)
}

// CallerRecordKey returns the key for a caller record {toolID}/{caller}.
func CallerRecordKey(toolID, caller string) []byte {
	return append(CallerRecordPrefix, []byte(toolID+"/"+caller)...)
}

// CallerRecordIterPrefix returns the prefix for iterating caller records of a tool.
func CallerRecordIterPrefix(toolID string) []byte {
	return append(CallerRecordPrefix, []byte(toolID+"/")...)
}

// DependencyEdgeKey returns the key for a dependency edge {from}/{to}.
func DependencyEdgeKey(fromToolID, toToolID string) []byte {
	return append(DependencyEdgePrefix, []byte(fromToolID+"/"+toToolID)...)
}

// DependencyEdgeIterPrefix returns the prefix for iterating a tool's dependencies.
func DependencyEdgeIterPrefix(fromToolID string) []byte {
	return append(DependencyEdgePrefix, []byte(fromToolID+"/")...)
}

// ReverseDependencyKey returns the key for a reverse dependency {to}/{from}.
func ReverseDependencyKey(toToolID, fromToolID string) []byte {
	return append(ReverseDependencyPrefix, []byte(toToolID+"/"+fromToolID)...)
}

// ReverseDependencyIterPrefix returns the prefix for iterating a tool's dependents.
func ReverseDependencyIterPrefix(toToolID string) []byte {
	return append(ReverseDependencyPrefix, []byte(toToolID+"/")...)
}

// DemandWindowKey returns the key for a tool's demand window.
func DemandWindowKey(toolID string) []byte {
	return append(DemandWindowPrefix, []byte(toolID)...)
}

// FreeAllowanceKey returns the key for a caller's free tier allowance.
func FreeAllowanceKey(caller string) []byte {
	return append(FreeAllowancePrefix, []byte(caller)...)
}

// AgentToolUsageKey returns the key for agent tool usage {agent}/{toolID}.
func AgentToolUsageKey(agentAddr, toolID string) []byte {
	return append(AgentToolUsagePrefix, []byte(agentAddr+"/"+toolID)...)
}

// AgentToolUsageIterPrefix returns the prefix for iterating agent tool usage.
func AgentToolUsageIterPrefix(agentAddr string) []byte {
	return append(AgentToolUsagePrefix, []byte(agentAddr+"/")...)
}

// AgentActiveToolsKey returns the key for an agent's active tools list.
func AgentActiveToolsKey(agentAddr string) []byte {
	return append(AgentActiveToolPrefix, []byte(agentAddr)...)
}
