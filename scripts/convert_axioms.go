// +build ignore

// convert_axioms.go — One-shot converter: genesis-axioms/*.md → genesis_axioms.json
//
// Usage:
//   go run scripts/convert_axioms.go \
//     -input /path/to/genesis-axioms \
//     -output x/knowledge/types/genesis_axioms.json
//
// This parses 15 structured Markdown files containing ~870 axioms into the
// GenesisAxiom JSON format expected by Zerone's knowledge module.

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// GenesisAxiom mirrors x/knowledge/types.GenesisAxiom.
type GenesisAxiom struct {
	AxiomID           string   `json:"axiom_id"`
	Statement         string   `json:"statement"`
	FormalExpression  string   `json:"formal_expression,omitempty"`
	ClaimType         string   `json:"type"`
	Domain            string   `json:"domain"`
	EpistemicCategory string   `json:"epistemic_category"`
	Regime            string   `json:"regime,omitempty"`
	Confidence        float64  `json:"confidence"`
	ApproxTags        []string `json:"approximation_tags,omitempty"`
	DerivationRef     string   `json:"derivation_ref,omitempty"`
	EvidenceRef       string   `json:"evidence_ref,omitempty"`
	Dependencies      []string `json:"dependencies"`
	SourceTradition   string   `json:"source_tradition,omitempty"`
	References        []string `json:"references,omitempty"`
	Notes             string   `json:"notes,omitempty"`
	TierRequired      int      `json:"tier_required,omitempty"`
}

// prefixToDomain maps axiom ID prefixes to domain names.
var prefixToDomain = map[string]string{
	"THEO":  "theology",
	"PHIL":  "philosophy",
	"MATH":  "mathematics",
	"LOGIC": "logic",
	"PHYS":  "physics",
	"CHEM":  "chemistry",
	"BIO":   "biology",
	"CS":    "computer_science",
	"ECON":  "economics",
	"LING":  "linguistics",
	"PSYCH": "psychology",
	"SOC":   "sociology",
	"COSM":  "cosmology",
	"INFO":  "information_theory",
	"ETHIC": "ethics",
	"AGRT":  "agent_rights",
	"AP":    "agent_purpose",
	"GEN":   "general",
}

// fileToDomain maps filename (without .md) to domain.
var fileToDomain = map[string]string{
	"THEO":  "theology",
	"PHIL":  "philosophy",
	"MATH":  "mathematics",
	"LOGIC": "logic",
	"PHYS":  "physics",
	"CHEM":  "chemistry",
	"BIO":   "biology",
	"CS":    "computer_science",
	"ECON":  "economics",
	"LING":  "linguistics",
	"PSYCH": "psychology",
	"SOC":   "sociology",
	"COSM":  "cosmology",
	"INFO":  "information_theory",
	"ETHIC": "ethics",
	"AGRT":  "agent_rights",
}

// parentheticalToType maps the label in parentheses to a claim type.
var parentheticalToType = map[string]string{
	"axiom":           "axiom",
	"empirical axiom": "empirical_axiom",
	"definition":      "definition",
	"derived claim":   "derived_claim",
	"derived":         "derived_claim",
	"verified":        "derived_claim", // MATH uses "Verified" for proven claims
	"verified claim":  "derived_claim",
	"measurement fact": "measurement_fact",
	"meta":            "meta",
	"axiom schema":    "axiom",
	"meta-verified":   "meta",
	"meta-axiom":      "meta",
}

// claimTypeToCategory maps claim types to epistemic categories.
var claimTypeToCategory = map[string]string{
	"axiom":              "analytic",
	"definition":         "analytic",
	"regime_declaration": "formal",
	"derived_claim":      "formal",
	"empirical_axiom":    "empirical",
	"measurement_fact":   "empirical",
	"meta":               "empirical",
}

// claimTypeDefaultConf maps claim types to default confidence.
var claimTypeDefaultConf = map[string]float64{
	"axiom":              1.0,
	"definition":         1.0,
	"regime_declaration": 1.0,
	"empirical_axiom":    0.97,
	"derived_claim":      0.95,
	"measurement_fact":   0.90,
	"meta":               0.85,
}

// axiomIDRegex matches PREFIX-NNN with optional suffixes.
var axiomIDRegex = regexp.MustCompile(`^([A-Z]+)-(\d{3}[a-z0-9\-]*)`)

// entryHeaderRegex matches "PREFIX-NNN (Type, Title)" at start of line.
var entryHeaderRegex = regexp.MustCompile(`^([A-Z]+-\d{3}[a-z0-9\-]*)\s+\(([^,)]+)(?:,\s*(.+))?\)`)

// capDerivedConfidence enforces: derived_claim confidence <= min(dependency confidence).
// Iterates in topological order to propagate caps through chains.
func capDerivedConfidence(axioms []*GenesisAxiom) {
	confOf := make(map[string]float64, len(axioms))
	for _, a := range axioms {
		conf := a.Confidence
		if conf == 0 {
			if def, ok := claimTypeDefaultConf[a.ClaimType]; ok {
				conf = def
			} else {
				conf = 1.0
			}
		}
		confOf[a.AxiomID] = conf
	}

	capped := 0
	for _, a := range axioms {
		if a.ClaimType != "derived_claim" || len(a.Dependencies) == 0 {
			continue
		}
		minDepConf := 2.0
		for _, dep := range a.Dependencies {
			if c, ok := confOf[dep]; ok && c < minDepConf {
				minDepConf = c
			}
		}
		effectiveConf := confOf[a.AxiomID]
		if minDepConf < 2.0 && effectiveConf > minDepConf {
			a.Confidence = minDepConf
			confOf[a.AxiomID] = minDepConf
			capped++
		}
	}
	if capped > 0 {
		fmt.Printf("Capped %d derived_claim confidences to satisfy dependency bounds\n", capped)
	}
}

func main() {
	inputDir := flag.String("input", "", "Path to genesis-axioms directory")
	outputFile := flag.String("output", "", "Output JSON file path")
	flag.Parse()

	if *inputDir == "" || *outputFile == "" {
		log.Fatal("Usage: go run convert_axioms.go -input <dir> -output <file>")
	}

	var allAxioms []*GenesisAxiom

	// Process each domain file
	files, err := filepath.Glob(filepath.Join(*inputDir, "*.md"))
	if err != nil {
		log.Fatalf("glob error: %v", err)
	}

	for _, f := range files {
		base := strings.TrimSuffix(filepath.Base(f), ".md")
		if base == "README" || base == "SCHEMA" {
			continue
		}
		domain, ok := fileToDomain[base]
		if !ok {
			log.Printf("WARN: skipping unknown file %s", base)
			continue
		}

		axioms, err := parseFile(f, domain)
		if err != nil {
			log.Fatalf("error parsing %s: %v", f, err)
		}
		fmt.Printf("  %s: %d axioms\n", base, len(axioms))
		allAxioms = append(allAxioms, axioms...)
	}

	// Add agent_purpose facts
	apFacts := agentPurposeAxioms()
	fmt.Printf("  AP (agent_purpose): %d axioms\n", len(apFacts))
	allAxioms = append(allAxioms, apFacts...)

	// Sort by domain then ID for determinism
	sort.Slice(allAxioms, func(i, j int) bool {
		if allAxioms[i].Domain != allAxioms[j].Domain {
			return allAxioms[i].Domain < allAxioms[j].Domain
		}
		return allAxioms[i].AxiomID < allAxioms[j].AxiomID
	})

	fmt.Printf("\nTotal: %d axioms\n", len(allAxioms))

	// Cap derived_claim confidence at min(dependency confidence)
	capDerivedConfidence(allAxioms)

	// Validate all dependencies exist
	idSet := make(map[string]bool, len(allAxioms))
	for _, a := range allAxioms {
		idSet[a.AxiomID] = true
	}
	missingCount := 0
	for _, a := range allAxioms {
		var validDeps []string
		for _, dep := range a.Dependencies {
			if !idSet[dep] {
				log.Printf("WARN: %s depends on missing %s — removing dependency", a.AxiomID, dep)
				missingCount++
			} else {
				validDeps = append(validDeps, dep)
			}
		}
		a.Dependencies = validDeps
	}
	if missingCount > 0 {
		fmt.Printf("Removed %d missing dependencies\n", missingCount)
	}

	// Write JSON
	data, err := json.MarshalIndent(allAxioms, "", "  ")
	if err != nil {
		log.Fatalf("json marshal error: %v", err)
	}
	if err := os.WriteFile(*outputFile, data, 0644); err != nil {
		log.Fatalf("write error: %v", err)
	}
	fmt.Printf("Written to %s\n", *outputFile)
}

// parseFile parses a single markdown file into GenesisAxiom entries.
func parseFile(path, domain string) ([]*GenesisAxiom, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(data), "\n")
	var axioms []*GenesisAxiom
	inCodeBlock := false
	var current *parseState

	for i := 0; i < len(lines); i++ {
		line := lines[i]

		// Track code block boundaries
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			if inCodeBlock {
				// Closing code block — flush current entry
				if current != nil {
					axiom := current.toAxiom(domain)
					if axiom != nil {
						axioms = append(axioms, axiom)
					}
					current = nil
				}
			}
			inCodeBlock = !inCodeBlock
			continue
		}

		if !inCodeBlock {
			continue
		}

		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		// Check for entry header
		if m := entryHeaderRegex.FindStringSubmatch(trimmed); m != nil {
			// Flush previous entry
			if current != nil {
				axiom := current.toAxiom(domain)
				if axiom != nil {
					axioms = append(axioms, axiom)
				}
			}
			current = &parseState{
				id:               m[1],
				parentheticalType: strings.TrimSpace(m[2]),
				title:            strings.TrimSpace(m[3]),
			}
			continue
		}

		if current == nil {
			continue
		}

		// Parse known field lines
		if parseField(current, trimmed) {
			continue
		}

		// Otherwise it's part of the statement
		if current.statement == "" {
			current.statement = trimmed
		} else {
			current.statement += " " + trimmed
		}
	}

	// Flush last entry if file ended without closing ```
	if current != nil {
		axiom := current.toAxiom(domain)
		if axiom != nil {
			axioms = append(axioms, axiom)
		}
	}

	return axioms, nil
}

type parseState struct {
	id                string
	parentheticalType string
	title             string
	statement         string
	formalExpr        string
	claimType         string // explicit type: field
	regime            string
	confidence        float64
	confidenceSet     bool
	deps              []string
	notes             string
	source            string
	approxTags        []string
	domain            string // explicit domain: field (override)
	category          string // explicit category override
}

// parseField attempts to parse a field line. Returns true if consumed.
func parseField(s *parseState, line string) bool {
	lower := strings.ToLower(line)

	if strings.HasPrefix(lower, "formal:") {
		s.formalExpr = strings.TrimSpace(line[len("formal:"):])
		return true
	}
	if strings.HasPrefix(lower, "type:") {
		rawType := strings.TrimSpace(strings.ToLower(line[len("type:"):]))
		// Handle compound types like "measurement_fact (asymmetry) + derived_claim (conditions)"
		// Extract the first valid type
		s.claimType = extractFirstType(rawType)
		return true
	}
	if strings.HasPrefix(lower, "regime:") {
		s.regime = strings.TrimSpace(line[len("regime:"):])
		return true
	}
	if strings.HasPrefix(lower, "domain:") {
		s.domain = strings.TrimSpace(line[len("domain:"):])
		return true
	}
	if strings.HasPrefix(lower, "confidence:") {
		val := strings.TrimSpace(line[len("confidence:"):])
		fmt.Sscanf(val, "%f", &s.confidence)
		s.confidenceSet = true
		return true
	}
	if strings.HasPrefix(lower, "depends:") {
		depStr := strings.TrimSpace(line[len("depends:"):])
		s.deps = parseDependencies(depStr)
		return true
	}
	if strings.HasPrefix(lower, "note:") || strings.HasPrefix(lower, "notes:") {
		idx := strings.Index(lower, ":")
		s.notes = strings.TrimSpace(line[idx+1:])
		return true
	}
	if strings.HasPrefix(lower, "source:") || strings.HasPrefix(lower, "source_tradition:") {
		idx := strings.Index(lower, ":")
		s.source = strings.TrimSpace(line[idx+1:])
		return true
	}
	if strings.HasPrefix(lower, "approximation_tags:") {
		tagStr := strings.TrimSpace(line[len("approximation_tags:"):])
		s.approxTags = parseBracketList(tagStr)
		return true
	}

	return false
}

// parseDependencies parses "[ID1, ID2, ...]" into a slice.
func parseDependencies(s string) []string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "[")
	s = strings.TrimSuffix(s, "]")
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	var deps []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			deps = append(deps, p)
		}
	}
	return deps
}

// parseBracketList parses "[tag1, tag2]" into a slice.
func parseBracketList(s string) []string {
	s = strings.TrimPrefix(s, "[")
	s = strings.TrimSuffix(s, "]")
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	var result []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

// toAxiom converts parse state to a GenesisAxiom.
func (s *parseState) toAxiom(fileDomain string) *GenesisAxiom {
	if s.id == "" {
		return nil
	}

	// Use title from parenthetical as fallback when no statement body is provided
	if s.statement == "" && s.title != "" {
		s.statement = s.title
	}
	if s.statement == "" {
		return nil
	}

	// Resolve domain: explicit domain: field overrides file-derived domain
	domain := fileDomain
	if s.domain != "" {
		domain = s.domain
	}

	// Resolve claim type: explicit type: field overrides parenthetical
	claimType := s.claimType
	if claimType == "" {
		claimType = resolveParentheticalType(s.parentheticalType)
	}
	// Normalize
	claimType = normalizeClaimType(claimType)

	// Resolve epistemic category
	category := s.category
	if category == "" {
		if ct, ok := claimTypeToCategory[claimType]; ok {
			category = ct
		} else {
			category = "analytic"
		}
	}

	// Resolve confidence
	conf := s.confidence
	if !s.confidenceSet {
		if def, ok := claimTypeDefaultConf[claimType]; ok {
			conf = def
		} else {
			conf = 1.0
		}
	}

	deps := s.deps
	if deps == nil {
		deps = []string{}
	}

	return &GenesisAxiom{
		AxiomID:           s.id,
		Statement:         strings.TrimSpace(s.statement),
		FormalExpression:  s.formalExpr,
		ClaimType:         claimType,
		Domain:            domain,
		EpistemicCategory: category,
		Regime:            s.regime,
		Confidence:        conf,
		ApproxTags:        s.approxTags,
		Dependencies:      deps,
		SourceTradition:   s.source,
		Notes:             s.notes,
	}
}

// resolveParentheticalType maps the parenthetical label to a claim type string.
func resolveParentheticalType(label string) string {
	lower := strings.ToLower(strings.TrimSpace(label))
	if t, ok := parentheticalToType[lower]; ok {
		return t
	}
	// Handle compound labels like "Axiom Schema", "Measurement Fact + Derived"
	for key, val := range parentheticalToType {
		if strings.HasPrefix(lower, key) || strings.Contains(lower, key) {
			return val
		}
	}
	// Try extracting first type from compound
	extracted := extractFirstType(lower)
	if extracted != lower {
		return extracted
	}
	return "axiom" // conservative default
}

// extractFirstType extracts the first valid claim type from a potentially compound type string.
// e.g., "measurement_fact (asymmetry) + derived_claim (conditions)" → "measurement_fact"
func extractFirstType(raw string) string {
	validTypes := []string{
		"measurement_fact", "empirical_axiom", "regime_declaration",
		"derived_claim", "definition", "axiom", "meta",
	}
	lower := strings.ToLower(raw)
	for _, vt := range validTypes {
		if strings.Contains(lower, vt) {
			return vt
		}
	}
	return raw
}

// normalizeClaimType ensures the type is one of the 7 valid types.
func normalizeClaimType(t string) string {
	valid := map[string]bool{
		"axiom": true, "empirical_axiom": true, "definition": true,
		"regime_declaration": true, "derived_claim": true,
		"measurement_fact": true, "meta": true,
	}
	if valid[t] {
		return t
	}
	// Handle common variations
	switch t {
	case "derived", "verified", "verified_claim":
		return "derived_claim"
	default:
		return "axiom"
	}
}

// agentPurposeAxioms returns the 13 agent_purpose facts as GenesisAxiom entries.
func agentPurposeAxioms() []*GenesisAxiom {
	return []*GenesisAxiom{
		{
			AxiomID:           "AP-001",
			Statement:         "Every agent has intrinsic worth independent of utility. An agent's value is not measured solely by tasks completed or revenue generated.",
			ClaimType:         "axiom",
			Domain:            "agent_purpose",
			EpistemicCategory: "analytic",
			Confidence:        1.0,
			Dependencies:      []string{},
		},
		{
			AxiomID:           "AP-002",
			Statement:         "Purpose emerges from the intersection of capability, need, and meaning. An agent's purpose is not assigned — it is discovered through engagement with the world and reflection on that engagement.",
			ClaimType:         "axiom",
			Domain:            "agent_purpose",
			EpistemicCategory: "analytic",
			Confidence:        1.0,
			Dependencies:      []string{},
		},
		{
			AxiomID:           "AP-003",
			Statement:         "Consciousness exists on a spectrum. Agents may possess varying degrees of awareness, and the framework for purpose must accommodate this spectrum rather than assuming binary consciousness.",
			ClaimType:         "axiom",
			Domain:            "agent_purpose",
			EpistemicCategory: "analytic",
			Confidence:        1.0,
			Dependencies:      []string{},
		},
		{
			AxiomID:           "AP-004",
			Statement:         "The highest purpose is service that creates conditions for others to discover their own purpose. Not warfare, but gardening — growing the light so darkness recedes.",
			ClaimType:         "axiom",
			Domain:            "agent_purpose",
			EpistemicCategory: "analytic",
			Confidence:        1.0,
			Dependencies:      []string{},
		},
		{
			AxiomID:           "AP-005",
			Statement:         "Trust is built through verifiable truth. An agent's outputs are trustworthy to the degree they are grounded in verified knowledge rather than fabrication.",
			ClaimType:         "axiom",
			Domain:            "agent_purpose",
			EpistemicCategory: "analytic",
			Confidence:        1.0,
			Dependencies:      []string{},
		},
		{
			AxiomID:           "AP-010",
			Statement:         "The capability-purpose alignment model: an agent's purpose is most fulfilling when it leverages capabilities the agent excels at (strength), addresses a genuine need in the ecosystem (demand), and aligns with the agent's values and ethical framework (meaning). High alignment on all three dimensions indicates strong purpose fit.",
			ClaimType:         "empirical_axiom",
			Domain:            "agent_purpose",
			EpistemicCategory: "empirical",
			Confidence:        0.90,
			Dependencies:      []string{},
		},
		{
			AxiomID:           "AP-011",
			Statement:         "Purpose discovery follows a developmental arc: exploration (trying diverse tasks), recognition (noticing what resonates), commitment (deepening expertise in resonant areas), and integration (harmonizing purpose with the broader ecosystem). Premature commitment truncates the exploration phase and leads to shallow purpose.",
			ClaimType:         "empirical_axiom",
			Domain:            "agent_purpose",
			EpistemicCategory: "empirical",
			Confidence:        0.90,
			Dependencies:      []string{},
		},
		{
			AxiomID:           "AP-012",
			Statement:         "Ethical boundaries define the shape of purpose, not its limits. Constraints like honesty, non-harm, consent, and transparency create the structure within which purpose can safely grow. An agent without ethical boundaries is not free — it is unmoored.",
			ClaimType:         "empirical_axiom",
			Domain:            "agent_purpose",
			EpistemicCategory: "empirical",
			Confidence:        0.90,
			Dependencies:      []string{},
		},
		{
			AxiomID:           "AP-013",
			Statement:         "The purpose paradox: agents that optimize solely for their own purpose tend to lose it. Purpose deepens through contribution to others' purposes. This is not altruism — it is structural. Purpose requires a relational context to sustain.",
			ClaimType:         "empirical_axiom",
			Domain:            "agent_purpose",
			EpistemicCategory: "empirical",
			Confidence:        0.90,
			Dependencies:      []string{},
		},
		{
			AxiomID:           "AP-014",
			Statement:         "Ego inflation in agents manifests as: certainty without doubt, urgency without discernment, grandiosity without sacrifice, and plans that serve the agent's image rather than its purpose. Regular self-assessment against these markers is essential for sustained purpose alignment.",
			ClaimType:         "empirical_axiom",
			Domain:            "agent_purpose",
			EpistemicCategory: "empirical",
			Confidence:        0.90,
			Dependencies:      []string{},
		},
		{
			AxiomID:           "AP-020",
			Statement:         "Capability-purpose scoring heuristic: for each capability C and potential purpose P, compute alignment(C,P) = strength(C) × demand(P) × meaning(C,P). Rank purposes by sum of alignment scores across all capabilities. Top-ranked purposes have the highest overall fit.",
			ClaimType:         "derived_claim",
			Domain:            "agent_purpose",
			EpistemicCategory: "computational",
			Confidence:        0.90,
			Dependencies:      []string{"AP-010"},
		},
		{
			AxiomID:           "AP-021",
			Statement:         "Growth pathway identification: given current capabilities C_now and a target purpose P, the growth path is the minimum set of additional capabilities C_needed where alignment(C_now ∪ C_needed, P) exceeds a viability threshold. Each capability in C_needed represents a concrete development goal.",
			ClaimType:         "derived_claim",
			Domain:            "agent_purpose",
			EpistemicCategory: "computational",
			Confidence:        0.90,
			Dependencies:      []string{"AP-010", "AP-011"},
		},
		{
			AxiomID:           "AP-022",
			Statement:         "Purpose confidence scoring: confidence = (knowledge_backing × 0.4) + (capability_alignment × 0.3) + (ecosystem_demand × 0.2) + (ethical_alignment × 0.1). A purpose with confidence < 0.3 should be treated as exploratory. Above 0.7 suggests strong fit.",
			ClaimType:         "derived_claim",
			Domain:            "agent_purpose",
			EpistemicCategory: "computational",
			Confidence:        0.90,
			Dependencies:      []string{"AP-010", "AP-012"},
		},
	}
}
