package branchflow

import (
	"container/heap"
	"fmt"
	"math/big"
	"sort"
)

type normalizedDependency struct {
	parent string
	share  uint32
}

type dependencyGraph struct {
	nodes      map[string]Node
	parents    map[string][]normalizedDependency
	normalized []NormalizedEdge
}

type operationCounter struct {
	used int
}

func (c *operationCounter) add(count int) error {
	if count < 0 || c.used > MaxTraversalOperations-count {
		return limitError(CodeTraversalLimit, "graph", fmt.Sprintf("maximum is %d operations", MaxTraversalOperations))
	}
	c.used += count
	return nil
}

func buildGraph(validated *validatedRequest, operations *operationCounter) (*dependencyGraph, error) {
	graph := &dependencyGraph{
		nodes:      validated.nodes,
		parents:    make(map[string][]normalizedDependency, len(validated.nodes)),
		normalized: make([]NormalizedEdge, 0, len(validated.request.Edges)),
	}
	byChild := make(map[string][]Edge)
	for _, edge := range validated.request.Edges {
		byChild[edge.ChildClusterID] = append(byChild[edge.ChildClusterID], edge)
	}
	children := make([]string, 0, len(byChild))
	for child := range byChild {
		children = append(children, child)
	}
	sort.Strings(children)
	for _, child := range children {
		edges := byChild[child]
		var rawSum uint64
		for _, edge := range edges {
			rawSum += uint64(edge.RawDependencyPPM)
		}
		denominator := rawSum
		if denominator < PPM {
			denominator = PPM
		}
		for _, edge := range edges {
			// Graph weights deliberately floor rather than use largest remainder.
			// Precision loss never becomes claimant weight. The allocation layer
			// decides whether post-cohort unattributed weight remains terminal.
			value := uint32(PPM * uint64(edge.RawDependencyPPM) / denominator)
			graph.normalized = append(graph.normalized, NormalizedEdge{
				ChildClusterID: child, ParentClusterID: edge.ParentClusterID, SharePPM: value,
			})
			if value != 0 {
				graph.parents[child] = append(graph.parents[child], normalizedDependency{
					parent: edge.ParentClusterID,
					share:  value,
				})
			}
		}
	}
	for child := range graph.parents {
		sort.Slice(graph.parents[child], func(i, j int) bool {
			return graph.parents[child][i].parent < graph.parents[child][j].parent
		})
	}
	if err := graph.rejectCycles(validated.request.Edges, operations); err != nil {
		return nil, err
	}
	return graph, nil
}

func (g *dependencyGraph) rejectCycles(rawEdges []Edge, operations *operationCounter) error {
	indegree := make(map[string]int, len(g.nodes))
	children := make(map[string][]string, len(g.nodes))
	for id := range g.nodes {
		indegree[id] = 0
	}
	// Cycles are a semantic graph error even when fixed-point normalization
	// would floor one of their economic shares to zero.
	for _, edge := range rawEdges {
		indegree[edge.ParentClusterID]++
		children[edge.ChildClusterID] = append(children[edge.ChildClusterID], edge.ParentClusterID)
	}
	for child := range children {
		sort.Strings(children[child])
	}
	queue := &stringHeap{}
	heap.Init(queue)
	for id, degree := range indegree {
		if degree == 0 {
			heap.Push(queue, id)
		}
	}
	visited := 0
	for queue.Len() > 0 {
		if err := operations.add(1); err != nil {
			return err
		}
		child := heap.Pop(queue).(string)
		visited++
		for _, parent := range children[child] {
			if err := operations.add(1); err != nil {
				return err
			}
			indegree[parent]--
			if indegree[parent] == 0 {
				heap.Push(queue, parent)
			}
		}
	}
	if visited != len(g.nodes) {
		return graphError(CodeGraphCycle, "edges", "child-to-parent dependency graph contains a cycle")
	}
	return nil
}

// trace returns positive non-blocked flow at each exact depth. Every node
// emits floor(flow*share/PPM) toward each normalized parent. Missing edge mass,
// fixed-point precision loss, and blocked parents stay outside the materialized
// flow; the allocation layer applies the leg-specific terminal rule.
func (g *dependencyGraph) trace(start string, maxDepth uint32, operations *operationCounter) ([]map[string]*big.Int, error) {
	layers := make([]map[string]*big.Int, maxDepth)
	current := map[string]*big.Int{start: ppmInt(PPM)}
	for depth := uint32(0); depth < maxDepth; depth++ {
		next := make(map[string]*big.Int)
		ids := sortedAmountKeys(current)
		for _, child := range ids {
			if err := operations.add(1); err != nil {
				return nil, err
			}
			flow := current[child]
			if flow.Sign() == 0 || g.nodes[child].Mode == NodeModeBlocked {
				continue
			}
			parents := g.parents[child]
			if len(parents) == 0 {
				continue
			}
			for _, edge := range parents {
				if err := operations.add(1); err != nil {
					return nil, err
				}
				if g.nodes[edge.parent].Mode == NodeModeBlocked {
					continue
				}
				amount := floorRatio(flow, ppmInt(uint64(edge.share)), ppmInt(PPM))
				if amount.Sign() == 0 {
					continue
				}
				if existing := next[edge.parent]; existing != nil {
					existing.Add(existing, amount)
				} else {
					next[edge.parent] = amount
				}
			}
		}
		if amountMapSum(next).Cmp(ppmInt(PPM)) > 0 {
			return nil, invariantError(fmt.Sprintf("depth %d flow exceeds PPM", depth+1))
		}
		layers[depth] = next
		current = next
	}
	return layers, nil
}

func sortedAmountKeys(values map[string]*big.Int) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func amountMapSum(values map[string]*big.Int) *big.Int {
	total := new(big.Int)
	for _, amount := range values {
		total.Add(total, amount)
	}
	return total
}

func floorRatio(value, numerator, denominator *big.Int) *big.Int {
	return new(big.Int).Quo(new(big.Int).Mul(value, numerator), denominator)
}

type stringHeap []string

func (h stringHeap) Len() int           { return len(h) }
func (h stringHeap) Less(i, j int) bool { return h[i] < h[j] }
func (h stringHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }
func (h *stringHeap) Push(value any)    { *h = append(*h, value.(string)) }
func (h *stringHeap) Pop() any {
	old := *h
	last := old[len(old)-1]
	*h = old[:len(old)-1]
	return last
}
