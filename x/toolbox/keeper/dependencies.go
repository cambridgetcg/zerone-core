package keeper

import (
	"context"
	"math/big"
	"strconv"

	"github.com/zerone-chain/zerone/x/toolbox/types"
)

// CheckDependencyCycles performs DFS cycle detection bounded by MaxDependencyDepth.
// Returns an error if adding depID as a dependency of toolID would create a cycle.
func (k Keeper) CheckDependencyCycles(ctx context.Context, toolID, depID string, maxDepth uint32) error {
	visited := make(map[string]bool)
	return k.dfsCycleCheck(ctx, depID, toolID, visited, 0, maxDepth)
}

func (k Keeper) dfsCycleCheck(ctx context.Context, current, target string, visited map[string]bool, depth uint32, maxDepth uint32) error {
	if depth > maxDepth {
		return types.ErrDependencyDepthExceeded
	}
	if current == target {
		return types.ErrDependencyCycle
	}
	if visited[current] {
		return nil
	}
	visited[current] = true

	var depErr error
	k.IterateDependencyEdgesFrom(ctx, current, func(edge *types.DependencyEdge) bool {
		if err := k.dfsCycleCheck(ctx, edge.ToToolId, target, visited, depth+1, maxDepth); err != nil {
			depErr = err
			return true
		}
		return false
	})
	return depErr
}

// ValidateDependencies checks that all dependency IDs exist, are active/testing, and trust-eligible.
func (k Keeper) ValidateDependencies(ctx context.Context, toolID string, depIDs []string) error {
	params := k.GetParams(ctx)
	if uint32(len(depIDs)) > params.MaxDependencies {
		return types.ErrTooManyDependencies
	}

	for _, depID := range depIDs {
		dep, ok := k.GetTool(ctx, depID)
		if !ok {
			return types.ErrDependencyNotFound.Wrapf("dependency %s not found", depID)
		}
		if dep.Status == types.ToolStatusRetired {
			return types.ErrToolRetired.Wrapf("dependency %s is retired", depID)
		}
		if dep.Status == types.ToolStatusDeprecated {
			return types.ErrToolDeprecated.Wrapf("dependency %s is deprecated", depID)
		}
		if !types.IsDependencyEligible(dep.TrustScore) {
			return types.ErrIneligibleDependency.Wrapf("dependency %s has insufficient trust", depID)
		}
		// Cycle check.
		if err := k.CheckDependencyCycles(ctx, toolID, depID, params.MaxDependencyDepth); err != nil {
			return err
		}
	}
	return nil
}

// FlattenDependencies returns all transitive dependencies in DFS post-order.
func (k Keeper) FlattenDependencies(ctx context.Context, toolID string) []string {
	visited := make(map[string]bool)
	var result []string
	k.flattenDFS(ctx, toolID, visited, &result)
	return result
}

func (k Keeper) flattenDFS(ctx context.Context, toolID string, visited map[string]bool, result *[]string) {
	if visited[toolID] {
		return
	}
	visited[toolID] = true
	k.IterateDependencyEdgesFrom(ctx, toolID, func(edge *types.DependencyEdge) bool {
		k.flattenDFS(ctx, edge.ToToolId, visited, result)
		return false
	})
	*result = append(*result, toolID)
}

// TopologicalSort returns a topological ordering of the direct dependencies using Kahn's algorithm.
func (k Keeper) TopologicalSort(ctx context.Context, depIDs []string) []string {
	if len(depIDs) == 0 {
		return nil
	}

	// Build adjacency for the subset.
	inDegree := make(map[string]int)
	adj := make(map[string][]string)
	depSet := make(map[string]bool)
	for _, id := range depIDs {
		depSet[id] = true
		inDegree[id] = 0
	}

	for _, id := range depIDs {
		k.IterateDependencyEdgesFrom(ctx, id, func(edge *types.DependencyEdge) bool {
			if depSet[edge.ToToolId] {
				adj[id] = append(adj[id], edge.ToToolId)
				inDegree[edge.ToToolId]++
			}
			return false
		})
	}

	var queue []string
	for _, id := range depIDs {
		if inDegree[id] == 0 {
			queue = append(queue, id)
		}
	}

	var sorted []string
	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]
		sorted = append(sorted, node)
		for _, neighbor := range adj[node] {
			inDegree[neighbor]--
			if inDegree[neighbor] == 0 {
				queue = append(queue, neighbor)
			}
		}
	}
	return sorted
}

// ComputeTransitiveCost computes the total cost of all transitive dependencies.
func (k Keeper) ComputeTransitiveCost(ctx context.Context, toolID string) *big.Int {
	deps := k.FlattenDependencies(ctx, toolID)
	total := new(big.Int)
	for _, depID := range deps {
		if depID == toolID {
			continue // Skip self.
		}
		tool, ok := k.GetTool(ctx, depID)
		if !ok {
			continue
		}
		price, _ := new(big.Int).SetString(tool.PricePerCall, 10)
		if price != nil {
			total.Add(total, price)
		}
	}
	return total
}

// ComputeDependencyCost computes the cost of direct dependencies only.
func (k Keeper) ComputeDependencyCost(ctx context.Context, depIDs []string) *big.Int {
	total := new(big.Int)
	for _, depID := range depIDs {
		tool, ok := k.GetTool(ctx, depID)
		if !ok {
			continue
		}
		price := new(big.Int)
		if p, ok := price.SetString(tool.PricePerCall, 10); ok && p.Sign() > 0 {
			total.Add(total, p)
		}
	}
	return total
}

// ComputeDependencyCostUint64 returns the uint64 value of dependency cost.
func (k Keeper) ComputeDependencyCostUint64(ctx context.Context, depIDs []string) uint64 {
	cost := k.ComputeDependencyCost(ctx, depIDs)
	if !cost.IsUint64() {
		return ^uint64(0)
	}
	v, _ := strconv.ParseUint(cost.String(), 10, 64)
	return v
}

// ---------- Dependency Tree ----------

// DependencyNode represents a node in the dependency tree.
type DependencyNode struct {
	ToolID       string            `json:"tool_id"`
	Name         string            `json:"name"`
	Version      string            `json:"version"`
	PricePerCall string            `json:"price_per_call"`
	Status       string            `json:"status"`
	TrustScore   uint64            `json:"trust_score"`
	Children     []*DependencyNode `json:"children,omitempty"`
}

// GetDependencyTree builds a full dependency tree rooted at toolID, bounded by maxDepth.
func (k Keeper) GetDependencyTree(ctx context.Context, toolID string, maxDepth uint32) *DependencyNode {
	visited := make(map[string]bool)
	return k.buildDepTree(ctx, toolID, visited, 0, maxDepth)
}

// buildDepTree recursively constructs the dependency tree. Cycle-safe via visited map.
func (k Keeper) buildDepTree(ctx context.Context, toolID string, visited map[string]bool, depth, maxDepth uint32) *DependencyNode {
	tool, ok := k.GetTool(ctx, toolID)
	if !ok {
		return &DependencyNode{ToolID: toolID, Name: "(not found)", Status: "unknown"}
	}

	node := &DependencyNode{
		ToolID:       tool.Id,
		Name:         tool.Name,
		Version:      tool.Version,
		PricePerCall: tool.PricePerCall,
		Status:       tool.Status,
		TrustScore:   tool.TrustScore,
	}

	// Stop recursion at max depth or if already visited (cycle).
	if depth >= maxDepth || visited[toolID] {
		return node
	}
	visited[toolID] = true

	for _, depID := range tool.DependencyIds {
		child := k.buildDepTree(ctx, depID, visited, depth+1, maxDepth)
		node.Children = append(node.Children, child)
	}

	// Unmark for sibling subtree exploration (allows diamond deps, prevents cycles).
	delete(visited, toolID)

	return node
}
