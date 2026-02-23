package simulation_test

import (
	"math/rand"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// ============================================================================
// Activity Generator — produces randomized per-block economic activity.
//
// Activity probabilities per block:
//   - Knowledge claims:    30%
//   - Verification rounds: 20%
//   - Tool calls:          25%
//   - Transfers:           15%
//   - Delegation changes:   5%
//   - Governance:           2%
//   - Research:             3%
// ============================================================================

// ActivityType categorizes the simulated activity.
type ActivityType int

const (
	ActivityKnowledgeClaim ActivityType = iota
	ActivityVerification
	ActivityToolCall
	ActivityTransfer
	ActivityDelegation
	ActivityGovernance
	ActivityResearch
)

// SimActivity describes a single economic activity in a block.
type SimActivity struct {
	Type   ActivityType
	Actor  sdk.AccAddress // primary actor
	Target sdk.AccAddress // recipient / validator (may be nil)
	Amount sdkmath.Int    // uzrn amount involved
	Gas    sdkmath.Int    // gas cost in uzrn
	ToolID int            // tool index for tool calls
}

// ActivityGenerator produces randomized block activity.
type ActivityGenerator struct {
	rng        *rand.Rand
	agents     []sdk.AccAddress
	validators []sdk.AccAddress
	toolCount  int
}

// NewActivityGenerator creates a generator with the given accounts.
func NewActivityGenerator(seed int64, agents, validators []sdk.AccAddress, toolCount int) *ActivityGenerator {
	return &ActivityGenerator{
		rng:        rand.New(rand.NewSource(seed)),
		agents:     agents,
		validators: validators,
		toolCount:  toolCount,
	}
}

// GenerateBlock produces activities for a single block.
// Each activity type fires independently based on its probability.
func (g *ActivityGenerator) GenerateBlock(height int64) []SimActivity {
	var activities []SimActivity

	// Knowledge claims — 30%
	if g.rng.Float64() < 0.30 {
		actor := g.randomAgent()
		activities = append(activities, SimActivity{
			Type:   ActivityKnowledgeClaim,
			Actor:  actor,
			Amount: sdkmath.NewInt(int64(100_000 + g.rng.Intn(900_000))), // 0.1–1 ZRN claim stake
			Gas:    sdkmath.NewInt(100_000),                               // claim_submit gas
		})
	}

	// Verification rounds — 20%
	if g.rng.Float64() < 0.20 {
		validator := g.randomValidator()
		activities = append(activities, SimActivity{
			Type:   ActivityVerification,
			Actor:  validator,
			Amount: sdkmath.NewInt(int64(3_000_000)), // ~3 ZRN reward base
			Gas:    sdkmath.NewInt(70_000),            // commit + reveal gas
		})
	}

	// Tool calls — 25%
	if g.rng.Float64() < 0.25 && g.toolCount > 0 {
		actor := g.randomAgent()
		activities = append(activities, SimActivity{
			Type:   ActivityToolCall,
			Actor:  actor,
			Amount: sdkmath.NewInt(int64(50_000 + g.rng.Intn(200_000))), // 0.05–0.25 ZRN fee
			Gas:    sdkmath.NewInt(40_000),                               // call_tool gas
			ToolID: g.rng.Intn(g.toolCount),
		})
	}

	// Transfers — 15%
	if g.rng.Float64() < 0.15 {
		actor := g.randomAgent()
		target := g.randomAgentExcluding(actor)
		activities = append(activities, SimActivity{
			Type:   ActivityTransfer,
			Actor:  actor,
			Target: target,
			Amount: sdkmath.NewInt(int64(10_000 + g.rng.Intn(500_000))), // 0.01–0.51 ZRN
			Gas:    sdkmath.NewInt(21_000),                               // transfer gas
		})
	}

	// Delegation changes — 5%
	if g.rng.Float64() < 0.05 {
		actor := g.randomAgent()
		validator := g.randomValidator()
		activities = append(activities, SimActivity{
			Type:   ActivityDelegation,
			Actor:  actor,
			Target: validator,
			Amount: sdkmath.NewInt(int64(100_000 + g.rng.Intn(1_000_000))), // 0.1–1.1 ZRN
			Gas:    sdkmath.NewInt(60_000),                                  // delegate gas
		})
	}

	// Governance — 2%
	if g.rng.Float64() < 0.02 {
		actor := g.randomAgent()
		activities = append(activities, SimActivity{
			Type:   ActivityGovernance,
			Actor:  actor,
			Amount: sdkmath.NewInt(int64(30_000)),  // vote gas-only
			Gas:    sdkmath.NewInt(100_000),         // governance_propose gas
		})
	}

	// Research — 3%
	if g.rng.Float64() < 0.03 {
		actor := g.randomAgent()
		activities = append(activities, SimActivity{
			Type:   ActivityResearch,
			Actor:  actor,
			Amount: sdkmath.NewInt(int64(1_000_000)), // 1 ZRN research stake
			Gas:    sdkmath.NewInt(80_000),            // submit_research gas
		})
	}

	return activities
}

func (g *ActivityGenerator) randomAgent() sdk.AccAddress {
	return g.agents[g.rng.Intn(len(g.agents))]
}

func (g *ActivityGenerator) randomValidator() sdk.AccAddress {
	return g.validators[g.rng.Intn(len(g.validators))]
}

func (g *ActivityGenerator) randomAgentExcluding(exclude sdk.AccAddress) sdk.AccAddress {
	for i := 0; i < 10; i++ {
		addr := g.agents[g.rng.Intn(len(g.agents))]
		if !addr.Equals(exclude) {
			return addr
		}
	}
	// Fallback: return a different index
	for _, a := range g.agents {
		if !a.Equals(exclude) {
			return a
		}
	}
	return exclude
}
