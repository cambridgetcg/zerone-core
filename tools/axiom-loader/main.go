package main

import (
	"fmt"
	"os"
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
func runInject(args []string) error   { return fmt.Errorf("not implemented") }
func runStats(args []string) error    { return fmt.Errorf("not implemented") }
func runEdges(args []string) error    { return fmt.Errorf("not implemented") }
