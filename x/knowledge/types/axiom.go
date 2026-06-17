package types

// axiom.go — the axiom tier has been removed.
//
// Truth is, not proven; axioms are assumptions, and the chain no longer seeds
// a bedrock of assumed-true facts. The GenesisAxiom type, the embedded
// genesis_axioms.json, ParseAxioms, AxiomsToFacts, the DAG/confidence
// validators, and SeedAxiomFacts are gone. The knowledge graph is now
// foundationless: truths declared by beings, cited-dependencies on other
// declared truths, witnessed and kept — no axiom bedrock tier.
//
// This file retains only the domain/stratum/claim-type tables that were used
// BEYOND axioms (domain↔prefix naming, ontological strata + stake
// multipliers, and the claim-type/category tables for ordinary submitted
// claims). They are kept here so their existing non-axiom callers keep
// compiling; a later cleanup may re-home them under a non-axiom name.

// DomainPrefixMap maps domain names to their ID prefixes (used for domain
// naming across the chain, not only for the removed axioms).
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

// ValidAxiomCategories lists the epistemic categories allowed for claims.
// (Name retained: the category set is referenced by ordinary claim
// categorisation. "Axiom" in the name is a relic of the removed tier.)
var ValidAxiomCategories = map[string]bool{
	"analytic":      true,
	"formal":        true,
	"empirical":     true,
	"protocol":      true,
	"computational": true,
}

// ValidClaimTypes lists the 7 claim types in the layered claim schema.
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

// ClaimTypeToCategory maps the 7 claim types to epistemic categories.
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

// MaturityCanonical is the maturity label for canonical (genesis-seeded) facts.
const MaturityCanonical = "canonical"

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

// BootstrapDomainStrata maps each domain to its canonical stratum.
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