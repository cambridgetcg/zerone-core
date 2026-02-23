package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	ktypes "github.com/zerone-chain/zerone/x/knowledge/types"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	cmd := os.Args[1]
	args := os.Args[2:]

	var err error
	switch cmd {
	case "validate":
		err = runValidate(args)
	case "inject":
		err = runInject(args)
	case "stats":
		err = runStats(args)
	case "edges":
		err = runEdges(args)
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", cmd)
		printUsage()
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `Usage: axiom-loader <command> [args]

Commands:
  validate <axioms.json>                    Validate axiom DAG
  inject   <axioms.json> <genesis.json>     Inject axioms into genesis
  stats    <axioms.json>                    Print axiom statistics
  edges    <axioms.json> [-o output.csv]    Export dependency edges as CSV
`)
}

func runValidate(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: axiom-loader validate <axioms.json>")
	}

	axioms, err := ktypes.LoadAxiomsFromFile(args[0])
	if err != nil {
		return err
	}
	fmt.Printf("✓ %d axioms loaded\n", len(axioms))

	// Check duplicates
	idSet := make(map[string]bool, len(axioms))
	dupes := 0
	for _, a := range axioms {
		if idSet[a.AxiomID] {
			dupes++
			fmt.Fprintf(os.Stderr, "  duplicate: %s\n", a.AxiomID)
		}
		idSet[a.AxiomID] = true
	}
	fmt.Printf("✓ %d duplicate IDs\n", dupes)

	// Check missing deps
	missing := 0
	for _, a := range axioms {
		for _, dep := range a.Dependencies {
			if !idSet[dep] {
				missing++
				fmt.Fprintf(os.Stderr, "  %s → missing %s\n", a.AxiomID, dep)
			}
		}
	}
	fmt.Printf("✓ %d missing dependencies\n", missing)

	// Full validation (includes DAG cycle check)
	domainNames := ktypes.AxiomDomainNames()
	valErr := ktypes.ValidateAxioms(axioms, domainNames)
	if valErr != nil {
		if strings.Contains(valErr.Error(), "cycle") {
			fmt.Printf("✗ Cycle detected\n")
		}
		return valErr
	}
	fmt.Printf("✓ 0 cycles detected\n")
	fmt.Printf("✓ DAG validation passed\n")

	// Summary stats
	dagStats, err := ktypes.ComputeDAGStats(axioms)
	if err != nil {
		return err
	}

	domains := make(map[string]bool)
	typeSet := make(map[string]bool)
	for _, a := range axioms {
		domains[a.Domain] = true
		typeSet[a.ClaimType] = true
	}

	// Count leaves: axioms with no dependents
	dependedOn := make(map[string]bool)
	for _, a := range axioms {
		for _, dep := range a.Dependencies {
			dependedOn[dep] = true
		}
	}
	leafCount := 0
	for _, a := range axioms {
		if !dependedOn[a.AxiomID] {
			leafCount++
		}
	}

	fmt.Printf("\nDomains:   %d\n", len(domains))
	fmt.Printf("Types:     %d distinct\n", len(typeSet))
	fmt.Printf("Roots:     %d (no dependencies)\n", dagStats.RootCount)
	fmt.Printf("Leaves:    %d (no dependents)\n", leafCount)
	fmt.Printf("Max depth: %d\n", dagStats.MaxDepth)

	return nil
}
func runInject(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: axiom-loader inject <axioms.json> <genesis.json>")
	}

	axiomPath := args[0]
	genesisPath := args[1]

	axioms, err := ktypes.LoadAxiomsFromFile(axiomPath)
	if err != nil {
		return err
	}

	// Validate before injecting
	domainNames := ktypes.AxiomDomainNames()
	if err := ktypes.ValidateAxioms(axioms, domainNames); err != nil {
		return fmt.Errorf("axiom validation failed: %w", err)
	}

	// Convert axioms to facts
	facts := ktypes.AxiomsToFacts(axioms)

	// Read genesis.json
	genesisData, err := os.ReadFile(genesisPath)
	if err != nil {
		return fmt.Errorf("failed to read genesis: %w", err)
	}

	var genesis map[string]json.RawMessage
	if err := json.Unmarshal(genesisData, &genesis); err != nil {
		return fmt.Errorf("failed to parse genesis: %w", err)
	}

	var appState map[string]json.RawMessage
	if err := json.Unmarshal(genesis["app_state"], &appState); err != nil {
		return fmt.Errorf("failed to parse app_state: %w", err)
	}

	var knowledgeState map[string]json.RawMessage
	if err := json.Unmarshal(appState["knowledge"], &knowledgeState); err != nil {
		return fmt.Errorf("failed to parse knowledge state: %w", err)
	}

	// Marshal facts and inject
	factsJSON, err := json.Marshal(facts)
	if err != nil {
		return fmt.Errorf("failed to marshal facts: %w", err)
	}
	knowledgeState["facts"] = factsJSON

	// Reassemble
	knowledgeJSON, err := json.Marshal(knowledgeState)
	if err != nil {
		return fmt.Errorf("failed to marshal knowledge: %w", err)
	}
	appState["knowledge"] = knowledgeJSON

	appStateJSON, err := json.Marshal(appState)
	if err != nil {
		return fmt.Errorf("failed to marshal app_state: %w", err)
	}
	genesis["app_state"] = appStateJSON

	result, err := json.MarshalIndent(genesis, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal genesis: %w", err)
	}

	if err := os.WriteFile(genesisPath, result, 0644); err != nil {
		return fmt.Errorf("failed to write genesis: %w", err)
	}

	fmt.Printf("✓ Injected %d axioms into %s\n", len(facts), genesisPath)
	return nil
}
func runStats(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: axiom-loader stats <axioms.json>")
	}

	axioms, err := ktypes.LoadAxiomsFromFile(args[0])
	if err != nil {
		return err
	}

	dagStats, err := ktypes.ComputeDAGStats(axioms)
	if err != nil {
		return err
	}

	// Group by domain
	type domainInfo struct {
		count    int
		roots    int
		types    map[string]bool
		maxDepth int
	}

	depthOf := computeDepthMap(axioms)

	domainMap := make(map[string]*domainInfo)
	for _, a := range axioms {
		di, ok := domainMap[a.Domain]
		if !ok {
			di = &domainInfo{types: make(map[string]bool)}
			domainMap[a.Domain] = di
		}
		di.count++
		di.types[a.ClaimType] = true
		if len(a.Dependencies) == 0 {
			di.roots++
		}
		d := depthOf[a.AxiomID]
		if d > di.maxDepth {
			di.maxDepth = d
		}
	}

	var domainNames []string
	for d := range domainMap {
		domainNames = append(domainNames, d)
	}
	sort.Strings(domainNames)

	fmt.Printf("%-20s %5s %5s %9s   %s\n", "Domain", "Count", "Roots", "Max Depth", "Types")
	fmt.Println(strings.Repeat("─", 78))
	for _, d := range domainNames {
		di := domainMap[d]
		var typeNames []string
		for t := range di.types {
			typeNames = append(typeNames, t)
		}
		sort.Strings(typeNames)
		fmt.Printf("%-20s %5d %5d %9d   %s\n", d, di.count, di.roots, di.maxDepth, strings.Join(typeNames, ", "))
	}

	crossMatrix := ktypes.ComputeCrossDomainMatrix(axioms)
	totalCross := 0
	for _, e := range crossMatrix.Entries {
		totalCross += e.Count
	}

	fmt.Printf("\n%-20s %5d %5d %9d\n", "Total:", len(axioms), dagStats.RootCount, dagStats.MaxDepth)
	fmt.Printf("Cross-domain deps: %d\n", totalCross)

	return nil
}

// computeDepthMap returns the DAG depth for each axiom ID via topological sort.
func computeDepthMap(axioms []*ktypes.GenesisAxiom) map[string]int {
	inDegree := make(map[string]int, len(axioms))
	dependents := make(map[string][]string, len(axioms))

	for _, a := range axioms {
		if _, ok := inDegree[a.AxiomID]; !ok {
			inDegree[a.AxiomID] = 0
		}
		for _, dep := range a.Dependencies {
			inDegree[a.AxiomID]++
			dependents[dep] = append(dependents[dep], a.AxiomID)
		}
	}

	depth := make(map[string]int, len(axioms))
	queue := make([]string, 0)
	for id, deg := range inDegree {
		if deg == 0 {
			queue = append(queue, id)
			depth[id] = 0
		}
	}

	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]
		for _, dependent := range dependents[node] {
			candidate := depth[node] + 1
			if candidate > depth[dependent] {
				depth[dependent] = candidate
			}
			inDegree[dependent]--
			if inDegree[dependent] == 0 {
				queue = append(queue, dependent)
			}
		}
	}

	return depth
}
func runEdges(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: axiom-loader edges <axioms.json> [-o output.csv]")
	}

	axiomPath := args[0]
	outputPath := ""
	for i := 1; i < len(args); i++ {
		if args[i] == "-o" && i+1 < len(args) {
			outputPath = args[i+1]
			i++
		}
	}

	axioms, err := ktypes.LoadAxiomsFromFile(axiomPath)
	if err != nil {
		return err
	}

	idToDomain := make(map[string]string, len(axioms))
	for _, a := range axioms {
		idToDomain[a.AxiomID] = a.Domain
	}

	out := os.Stdout
	if outputPath != "" {
		f, err := os.Create(outputPath)
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}
		defer f.Close()
		out = f
	}

	fmt.Fprintln(out, "source,target,source_domain,target_domain")
	edgeCount := 0
	for _, a := range axioms {
		for _, dep := range a.Dependencies {
			fmt.Fprintf(out, "%s,%s,%s,%s\n", a.AxiomID, dep, a.Domain, idToDomain[dep])
			edgeCount++
		}
	}

	if outputPath != "" {
		fmt.Printf("✓ Exported %d edges to %s\n", edgeCount, outputPath)
	}

	return nil
}
