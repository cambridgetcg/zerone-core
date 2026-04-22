package types

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
)

// GenesisAxiom represents a foundational truth injected at genesis.
// Axioms are converted to Fact objects with appropriate confidence and "verified" status.
// The layered type system supports 7 claim types with varying confidence models.
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

// AxiomConfidence is the confidence value for genesis axioms (100% = 1,000,000 BPS).
const AxiomConfidence = uint64(1_000_000)

// AxiomSubmitter is the submitter address for genesis axioms.
const AxiomSubmitter = "genesis"

// AxiomSubmitterDID is the DID for genesis axioms.
const AxiomSubmitterDID = "did:zrn:genesis"

// MaturityCanonical is the maturity label for axioms (eternal).
const MaturityCanonical = "canonical"

// ValidAxiomCategories lists the epistemic categories allowed for genesis axioms.
var ValidAxiomCategories = map[string]bool{
	"analytic":      true,
	"formal":        true,
	"empirical":     true,
	"protocol":      true,
	"computational": true,
}

// ValidClaimTypes lists the 7 claim types in the layered genesis axiom schema.
var ValidClaimTypes = map[string]bool{
	"axiom":              true,
	"empirical_axiom":    true,
	"definition":         true,
	"regime_declaration": true,
	"derived_claim":      true,
	"measurement_fact":   true,
	"meta":               true,
}

// ValidRegimes lists the physics regime tags for scoping empirical claims.
var ValidRegimes = map[string]bool{
	"classical":              true,
	"classical_field_theory": true,
	"classical_gravity":      true,
	"SR":                     true,
	"GR":                     true,
	"QM":                     true,
	"QFT":                    true,
	"QM/QFT":                 true,
	"QM/statmech":            true,
	"SR/QFT":                 true,
	"EM":                     true,
	"EM/QM":                  true,
	"EM/QFT":                 true,
	"thermo":                 true,
	"statmech":               true,
	"cosmology":              true,
	"SM":                     true,
	"particle_physics":       true,
	"modern_physics":         true,
}

// ClaimTypeToCategory maps the 7 genesis claim types to epistemic categories.
var ClaimTypeToCategory = map[string]string{
	"axiom":              "analytic",
	"definition":         "analytic",
	"regime_declaration": "formal",
	"derived_claim":      "formal",
	"empirical_axiom":    "empirical",
	"measurement_fact":   "empirical",
	"meta":               "empirical",
}

// ClaimTypeDefaultConfidence returns the default confidence for a claim type
// when no explicit confidence is provided (confidence == 0).
var ClaimTypeDefaultConfidence = map[string]float64{
	"axiom":              1.0,
	"definition":         1.0,
	"regime_declaration": 1.0,
	"empirical_axiom":    0.97,
	"derived_claim":      0.95,
	"measurement_fact":   0.90,
	"meta":               0.85,
}

// DomainPrefixMap maps domain names to their axiom ID prefixes.
var DomainPrefixMap = map[string]string{
	"theology":           "THEO",
	"philosophy":         "PHIL",
	"mathematics":        "MATH",
	"logic":              "LOGIC",
	"physics":            "PHYS",
	"chemistry":          "CHEM",
	"biology":            "BIO",
	"computer_science":   "CS",
	"economics":          "ECON",
	"linguistics":        "LING",
	"psychology":         "PSYCH",
	"sociology":          "SOC",
	"cosmology":          "COSM",
	"information_theory": "INFO",
	"ethics":             "ETHIC",
	"agent_rights":       "AGRT",
	"agent_purpose":      "AP",
	"general":            "GEN",
}

// PrefixToDomainMap is the reverse mapping from prefix to domain.
var PrefixToDomainMap map[string]string

func init() {
	PrefixToDomainMap = make(map[string]string, len(DomainPrefixMap))
	for domain, prefix := range DomainPrefixMap {
		PrefixToDomainMap[prefix] = domain
	}
}

// axiomIDRegex validates axiom IDs: PREFIX-NNN with optional lowercase suffixes
// (e.g., "MATH-001", "AGRT-004a", "AGRT-004a-ii").
var axiomIDRegex = regexp.MustCompile(`^[A-Z]+-\d{3}[a-z0-9\-]*$`)

// ValidateAxiomID checks that an axiom ID has the correct PREFIX-NNN format
// and that the prefix maps to a known domain.
func ValidateAxiomID(id string) error {
	if !axiomIDRegex.MatchString(id) {
		return fmt.Errorf("axiom ID %q does not match format PREFIX-NNN (e.g., MATH-001)", id)
	}
	prefix := id[:strings.Index(id, "-")]
	if _, ok := PrefixToDomainMap[prefix]; !ok {
		return fmt.Errorf("axiom ID %q has unknown prefix %q", id, prefix)
	}
	return nil
}

// LoadAxiomsFromFile reads a JSON array of GenesisAxiom from a file.
func LoadAxiomsFromFile(path string) ([]*GenesisAxiom, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read axiom file %s: %w", path, err)
	}
	return ParseAxioms(data)
}

// ParseAxioms parses a JSON array of GenesisAxiom from raw bytes.
func ParseAxioms(data []byte) ([]*GenesisAxiom, error) {
	var axioms []*GenesisAxiom
	if err := json.Unmarshal(data, &axioms); err != nil {
		return nil, fmt.Errorf("failed to unmarshal axioms: %w", err)
	}
	return axioms, nil
}

// ValidateAxioms performs comprehensive validation on a set of genesis axioms.
func ValidateAxioms(axioms []*GenesisAxiom, validDomains []string) error {
	if len(axioms) == 0 {
		return fmt.Errorf("axiom set is empty")
	}

	domainSet := make(map[string]bool, len(validDomains))
	for _, d := range validDomains {
		domainSet[d] = true
	}

	idSet := make(map[string]bool, len(axioms))

	for i, a := range axioms {
		if a == nil {
			return fmt.Errorf("axiom at index %d is nil", i)
		}

		if err := ValidateAxiomID(a.AxiomID); err != nil {
			return fmt.Errorf("axiom index %d: %w", i, err)
		}

		if idSet[a.AxiomID] {
			return fmt.Errorf("duplicate axiom ID: %s", a.AxiomID)
		}
		idSet[a.AxiomID] = true

		if !domainSet[a.Domain] {
			return fmt.Errorf("axiom %s references unknown domain %q", a.AxiomID, a.Domain)
		}

		prefix := a.AxiomID[:strings.Index(a.AxiomID, "-")]
		expectedDomain := PrefixToDomainMap[prefix]
		if expectedDomain != a.Domain {
			return fmt.Errorf("axiom %s prefix %q implies domain %q but domain is %q",
				a.AxiomID, prefix, expectedDomain, a.Domain)
		}

		if a.ClaimType != "" && !ValidClaimTypes[a.ClaimType] {
			return fmt.Errorf("axiom %s has invalid claim type %q", a.AxiomID, a.ClaimType)
		}

		if !ValidAxiomCategories[a.EpistemicCategory] {
			return fmt.Errorf("axiom %s has invalid epistemic category %q (must be analytic, formal, empirical, protocol, or computational)",
				a.AxiomID, a.EpistemicCategory)
		}

		if a.Regime != "" && !ValidRegimes[a.Regime] {
			return fmt.Errorf("axiom %s has invalid regime %q", a.AxiomID, a.Regime)
		}

		if a.Confidence < 0 || a.Confidence > 1.0 {
			return fmt.Errorf("axiom %s has confidence %f outside [0, 1]", a.AxiomID, a.Confidence)
		}

		if a.ClaimType == "derived_claim" && len(a.Dependencies) == 0 {
			return fmt.Errorf("axiom %s is a derived_claim but has no dependencies", a.AxiomID)
		}

		if strings.TrimSpace(a.Statement) == "" {
			return fmt.Errorf("axiom %s has empty statement", a.AxiomID)
		}
	}

	// Validate references point to existing axioms
	for _, a := range axioms {
		for _, ref := range a.References {
			if !idSet[ref] {
				return fmt.Errorf("axiom %s references unknown axiom %q", a.AxiomID, ref)
			}
		}
	}

	if err := ValidateAxiomDAG(axioms); err != nil {
		return err
	}

	if err := ValidateStratumConsistency(axioms); err != nil {
		return err
	}

	if err := ValidateDerivedConfidence(axioms); err != nil {
		return err
	}

	return nil
}

// ValidateAxiomDAG checks that axiom dependencies form a directed acyclic graph.
func ValidateAxiomDAG(axioms []*GenesisAxiom) error {
	idSet := make(map[string]bool, len(axioms))
	for _, a := range axioms {
		idSet[a.AxiomID] = true
	}

	for _, a := range axioms {
		for _, dep := range a.Dependencies {
			if !idSet[dep] {
				if dashIdx := strings.Index(dep, "-"); dashIdx > 0 {
					prefix := dep[:dashIdx]
					if domain, ok := PrefixToDomainMap[prefix]; ok {
						return fmt.Errorf("axiom %s depends on unknown axiom %q (expected in domain %q based on prefix %q)",
							a.AxiomID, dep, domain, prefix)
					}
				}
				return fmt.Errorf("axiom %s depends on unknown axiom %q", a.AxiomID, dep)
			}
			if dep == a.AxiomID {
				return fmt.Errorf("axiom %s has self-dependency", a.AxiomID)
			}
		}
	}

	// Kahn's algorithm for topological sort
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

	queue := make([]string, 0)
	for id, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, id)
		}
	}

	sorted := 0
	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]
		sorted++

		for _, dependent := range dependents[node] {
			inDegree[dependent]--
			if inDegree[dependent] == 0 {
				queue = append(queue, dependent)
			}
		}
	}

	if sorted != len(axioms) {
		var cycleMembers []string
		for id, degree := range inDegree {
			if degree > 0 {
				cycleMembers = append(cycleMembers, id)
			}
		}
		return fmt.Errorf("dependency cycle detected among axioms: %v", cycleMembers)
	}

	return nil
}

// ResolveAxiomCategory determines the epistemic category for a genesis axiom.
func ResolveAxiomCategory(a *GenesisAxiom) (category string, overrideDisagrees bool) {
	autoDerived := ""
	if a.ClaimType != "" {
		autoDerived = ClaimTypeToCategory[a.ClaimType]
	}

	if a.EpistemicCategory != "" {
		if autoDerived != "" && a.EpistemicCategory != autoDerived {
			return a.EpistemicCategory, true
		}
		return a.EpistemicCategory, false
	}

	if autoDerived != "" {
		return autoDerived, false
	}

	return "analytic", false
}

// axiomConfidenceBPS converts a genesis axiom's float64 confidence [0,1] to BPS uint64 [0,1_000_000].
func axiomConfidenceBPS(a *GenesisAxiom) uint64 {
	conf := a.Confidence
	if conf == 0 {
		if def, ok := ClaimTypeDefaultConfidence[a.ClaimType]; ok {
			conf = def
		} else {
			conf = 1.0
		}
	}
	return uint64(conf * 1_000_000)
}

// AxiomsToFacts converts genesis axioms to Fact objects ready for InitGenesis.
// Facts are created with per-axiom confidence, "verified" status, "canonical" maturity,
// and block height 0 (genesis). Category is auto-derived from ClaimType unless
// explicitly overridden by EpistemicCategory on the axiom.
func AxiomsToFacts(axioms []*GenesisAxiom) []*Fact {
	facts := make([]*Fact, 0, len(axioms))
	for _, a := range axioms {
		confBPS := axiomConfidenceBPS(a)
		resolvedCategory, _ := ResolveAxiomCategory(a)

		stratum := ""
		if s, ok := BootstrapDomainStrata[a.Domain]; ok {
			stratum = s
		}

		fact := &Fact{
			Id:              a.AxiomID,
			Content:         a.Statement,
			Domain:          a.Domain,
			Category:        resolvedCategory,
			Confidence:      confBPS,
			Submitter:       AxiomSubmitter,
			SubmittedAtBlock: 0,
			VerifiedAtBlock: 0,
			Fundamentality:  1_000_000,
			References:      a.Dependencies,
			Status:          FactStatus_FACT_STATUS_VERIFIED,
			LastVerifiedBlock: 0,
			Stratum:         stratum,
			Maturity:        MaturityCanonical,
			// Axioms are the bedrock — distance 0 from themselves. Derived
			// claims inherit distance+1 from their minimum-distance citation.
			// (ToK Wave 2)
			AxiomDistance: 0,
		}
		facts = append(facts, fact)
	}
	return facts
}

// ValidateDerivedConfidence checks that no derived_claim has confidence exceeding
// the minimum confidence of its dependencies.
func ValidateDerivedConfidence(axioms []*GenesisAxiom) error {
	confOf := make(map[string]float64, len(axioms))
	for _, a := range axioms {
		conf := a.Confidence
		if conf == 0 {
			if def, ok := ClaimTypeDefaultConfidence[a.ClaimType]; ok {
				conf = def
			} else {
				conf = 1.0
			}
		}
		confOf[a.AxiomID] = conf
	}

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
		if effectiveConf > minDepConf {
			return fmt.Errorf("derived_claim %s has confidence %.4f exceeding min dependency confidence %.4f",
				a.AxiomID, effectiveConf, minDepConf)
		}
	}
	return nil
}

// AxiomDomainNames returns all domain names that support axioms.
func AxiomDomainNames() []string {
	names := make([]string, 0, len(DomainPrefixMap))
	for domain := range DomainPrefixMap {
		names = append(names, domain)
	}
	return names
}

// DAGStats holds topological analysis results for the axiom dependency DAG.
type DAGStats struct {
	RootCount      int
	MaxDepth       int
	DepthCounts    map[int]int
	Orphans        []string
	MostReferenced []string
	TotalEdges     int
}

// CrossDomainEntry records references between two domains.
type CrossDomainEntry struct {
	FromDomain string
	ToDomain   string
	Count      int
}

// CrossDomainMatrix holds the full inter-domain dependency matrix.
type CrossDomainMatrix struct {
	Entries []CrossDomainEntry
	Domains []string
}

// ComputeDAGStats performs topological analysis on the axiom dependency DAG.
func ComputeDAGStats(axioms []*GenesisAxiom) (*DAGStats, error) {
	inDegree := make(map[string]int, len(axioms))
	dependents := make(map[string][]string, len(axioms))
	depCount := make(map[string]int, len(axioms))

	for _, a := range axioms {
		if _, ok := inDegree[a.AxiomID]; !ok {
			inDegree[a.AxiomID] = 0
		}
		for _, dep := range a.Dependencies {
			inDegree[a.AxiomID]++
			dependents[dep] = append(dependents[dep], a.AxiomID)
			depCount[dep]++
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

	processed := 0
	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]
		processed++

		for _, dependent := range dependents[node] {
			candidateDepth := depth[node] + 1
			if candidateDepth > depth[dependent] {
				depth[dependent] = candidateDepth
			}
			inDegree[dependent]--
			if inDegree[dependent] == 0 {
				queue = append(queue, dependent)
			}
		}
	}

	if processed != len(axioms) {
		return nil, fmt.Errorf("cycle detected: processed %d of %d nodes", processed, len(axioms))
	}

	stats := &DAGStats{
		DepthCounts: make(map[int]int),
	}

	dependedOn := make(map[string]bool)
	for _, a := range axioms {
		for _, dep := range a.Dependencies {
			dependedOn[dep] = true
		}
		stats.TotalEdges += len(a.Dependencies)
	}

	maxDepth := 0
	for _, a := range axioms {
		d := depth[a.AxiomID]
		stats.DepthCounts[d]++
		if d > maxDepth {
			maxDepth = d
		}
		if d == 0 {
			stats.RootCount++
		}
		if len(a.Dependencies) == 0 && !dependedOn[a.AxiomID] {
			stats.Orphans = append(stats.Orphans, a.AxiomID)
		}
	}
	stats.MaxDepth = maxDepth

	sort.Strings(stats.Orphans)

	type idCount struct {
		id    string
		count int
	}
	var counts []idCount
	for id, c := range depCount {
		counts = append(counts, idCount{id, c})
	}
	sort.Slice(counts, func(i, j int) bool {
		if counts[i].count != counts[j].count {
			return counts[i].count > counts[j].count
		}
		return counts[i].id < counts[j].id
	})
	limit := 10
	if len(counts) < limit {
		limit = len(counts)
	}
	stats.MostReferenced = make([]string, limit)
	for i := 0; i < limit; i++ {
		stats.MostReferenced[i] = fmt.Sprintf("%s (%d)", counts[i].id, counts[i].count)
	}

	return stats, nil
}

// ComputeCrossDomainMatrix builds an inter-domain dependency matrix.
func ComputeCrossDomainMatrix(axioms []*GenesisAxiom) *CrossDomainMatrix {
	idToDomain := make(map[string]string, len(axioms))
	domainSet := make(map[string]bool)
	for _, a := range axioms {
		idToDomain[a.AxiomID] = a.Domain
		domainSet[a.Domain] = true
	}

	type pair struct{ from, to string }
	matrix := make(map[pair]int)
	for _, a := range axioms {
		for _, dep := range a.Dependencies {
			depDomain := idToDomain[dep]
			if depDomain == "" {
				if dashIdx := strings.Index(dep, "-"); dashIdx > 0 {
					if d, ok := PrefixToDomainMap[dep[:dashIdx]]; ok {
						depDomain = d
					}
				}
			}
			if a.Domain != depDomain && depDomain != "" {
				matrix[pair{a.Domain, depDomain}]++
			}
		}
	}

	var entries []CrossDomainEntry
	for p, count := range matrix {
		entries = append(entries, CrossDomainEntry{
			FromDomain: p.from,
			ToDomain:   p.to,
			Count:      count,
		})
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Count != entries[j].Count {
			return entries[i].Count > entries[j].Count
		}
		if entries[i].FromDomain != entries[j].FromDomain {
			return entries[i].FromDomain < entries[j].FromDomain
		}
		return entries[i].ToDomain < entries[j].ToDomain
	})

	var domains []string
	for d := range domainSet {
		domains = append(domains, d)
	}
	sort.Strings(domains)

	return &CrossDomainMatrix{
		Entries: entries,
		Domains: domains,
	}
}

// ValidStrata lists the 7 valid ontological strata.
var ValidStrata = map[string]bool{
	"fundamental": true, "physical": true, "chemical": true,
	"biological": true, "cognitive": true, "social": true,
	"technological": true,
}

// StratumStakeMultiplier defines the minimum stake multiplier per stratum.
var StratumStakeMultiplier = map[string]uint64{
	"technological": 1, "social": 2, "cognitive": 3,
	"biological": 5, "chemical": 5, "physical": 10, "fundamental": 20,
}

// BootstrapDomainStrata maps each genesis domain to its canonical stratum.
var BootstrapDomainStrata = map[string]string{
	"mathematics":        "fundamental",
	"logic":              "fundamental",
	"theology":           "fundamental",
	"philosophy":         "fundamental",
	"information_theory": "fundamental",
	"physics":            "physical",
	"cosmology":          "physical",
	"chemistry":          "chemical",
	"biology":            "biological",
	"psychology":         "cognitive",
	"linguistics":        "cognitive",
	"economics":          "social",
	"sociology":          "social",
	"ethics":             "social",
	"agent_rights":       "social",
	"computer_science":   "technological",
	"general":            "technological",
	"agent_purpose":      "cognitive",
}

// DefaultProposalStratum is the stratum assigned to newly proposed domains.
const DefaultProposalStratum = "social"

// ValidateStratumConsistency checks that axioms do not depend on derived types.
func ValidateStratumConsistency(axioms []*GenesisAxiom) error {
	idToType := make(map[string]string, len(axioms))
	for _, a := range axioms {
		if a.ClaimType != "" {
			idToType[a.AxiomID] = a.ClaimType
		}
	}

	for _, a := range axioms {
		if a.ClaimType != "axiom" {
			continue
		}
		for _, dep := range a.Dependencies {
			depType := idToType[dep]
			if depType == "derived_claim" || depType == "measurement_fact" {
				return fmt.Errorf("stratum violation: axiom %s depends on %s %s — axioms must not depend on derived or measured claims",
					a.AxiomID, depType, dep)
			}
		}
	}
	return nil
}
