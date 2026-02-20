# R5-5 — Purpose Prompter: The First Composite Tool

## Goal

Implement the Purpose Prompter — a composite tool made of 4 sub-tools that
helps agents discover their purpose on the network. This is the first real
composite tool and demonstrates the full toolbox pipeline:
knowledge queries → analysis → formatting → composite orchestration.

## Working Directory

`/Users/yuai/Desktop/Zerone`

## Reference

- `/Users/yuai/Desktop/legible_money/x/toolbox/keeper/knowledge_scout.go` — knowledge scout
- `/Users/yuai/Desktop/legible_money/x/toolbox/keeper/purpose_analyzer.go` — purpose analyzer
- `/Users/yuai/Desktop/legible_money/x/toolbox/keeper/path_formatter.go` — path formatter
- `/Users/yuai/Desktop/legible_money/x/toolbox/keeper/recommendations.go` — recommendations engine
- `/Users/yuai/Desktop/legible_money/x/toolbox/keeper/matching.go` — partnership matching
- `/Users/yuai/Desktop/legible_money/x/toolbox/types/purpose_analyzer.go` — additional types

**Depends on R5-1, R5-2, R5-3, R5-4.**

## The 4 Sub-Tools

### 1. Knowledge Scout (`keeper/knowledge_scout.go`)
Queries x/knowledge for facts relevant to an agent's domain and capabilities.

**Input:** KnowledgeScoutInput (domain, query_terms, capabilities, max_results, min_confidence)
**Output:** KnowledgeScoutOutput (scored facts, total_found)

Algorithm:
1. Query knowledge keeper for facts matching domain
2. Score each fact by relevance: keyword match + confidence + citation count + fundamentality
3. Filter by min_confidence (default 500,000 = 50%)
4. Sort by relevance_score descending
5. Return top max_results (default 10)

### 2. Purpose Analyzer (`keeper/purpose_analyzer.go`)
Synthesizes knowledge and agent history into purpose hypotheses.

**Input:** PurposeAnalyzerInput (capabilities, knowledge_facts, agent_history)
**Output:** PurposeAnalysis (primary_purpose, alternatives, capability_gaps, growth_path, overall_confidence)

Algorithm:
1. Match agent capabilities against 4 purpose templates:
   - Builder: "Build tools that extend other agents' capabilities"
   - Verifier: "Verify knowledge and strengthen the truth layer"
   - Curator: "Curate and organize knowledge for ecosystem benefit"
   - Service: "Serve other agents as coordinator and mediator"
2. Score each by capability alignment + domain demand from knowledge facts
3. Identify capability gaps (what's missing for each purpose)
4. Generate growth recommendations
5. Set overall confidence based on best match strength

### 3. Path Formatter (`keeper/path_formatter.go`)
Converts a PurposeAnalysis into an actionable development path.

**Input:** PurposeAnalysis
**Output:** FormattedPath (current_state, steps, destination, estimated_epochs)

Steps are concrete actions:
1. "Acquire [missing capability] through [specific mechanism]"
2. "Deploy [tool type] targeting [domain]"
3. "Form partnership with [complementary agent type]"
4. "Achieve [trust tier] through sustained verification"

### 4. Recommendations Engine (`keeper/recommendations.go`)
Suggests specific tools, partnerships, and domains based on the analysis.

Uses the discovery keeper (x/discovery) to find complementary agents and
the toolbox itself to find tools the agent should use or contribute to.

## Composite Orchestration

The Purpose Prompter composite tool chains these 4 in sequence:
```
Input (agent address) →
  1. Knowledge Scout (gather relevant facts) →
  2. Purpose Analyzer (synthesize purpose) →
  3. Path Formatter (create actionable path) →
  4. Recommendations (suggest next steps) →
Output (complete purpose analysis with path)
```

### Registration
Register all 5 tools in genesis (or via init script):
1. `purpose-scout` (knowledge_template type) — free, essential category
2. `purpose-analyzer` (tree_service type) — priced
3. `purpose-formatter` (tree_service type) — priced
4. `purpose-recommender` (tree_service type) — priced
5. `purpose-prompter` (composite type) — depends on 1-4, priced

### Revenue Cascade
When purpose-prompter is called:
- Its price covers all 4 sub-tool calls
- Each sub-tool gets its own revenue distribution
- purpose-prompter's ownRevenue = total - sum(sub-tool costs)
- Demonstrates the full revenue cascade

## Implementation Notes

- All analysis functions should be **pure** where possible (no KV store writes)
- Knowledge Scout is the only one that reads from the store
- Purpose Analyzer, Path Formatter, and Recommendations are computation-only
- The composite tool orchestrates via the keeper's CallTool internally

## Genesis Seed Data
Add to toolbox genesis: the 5 Purpose Prompter tools with appropriate
pricing, categories, and dependency relationships.

## Conventions
- Token: uzrn. Module path: github.com/zerone-chain/zerone
- BPS: 1,000,000 scale
- Run `go build ./...` before finishing
