package types

// ---------- Purpose Prompter Constants ----------

const (
	// DefaultScoutMaxResults is the default number of facts returned by KnowledgeScout.
	DefaultScoutMaxResults = uint64(10)

	// DefaultScoutMinConfidence is the default minimum confidence threshold (50%).
	DefaultScoutMinConfidence = uint64(500_000)
)

// Confidence label thresholds (on 1,000,000 scale).
const (
	ConfidenceLabelExploratoryMax = uint64(200_000) // 0–20%
	ConfidenceLabelEmergingMax    = uint64(500_000) // 20–50%
	ConfidenceLabelStrongMax      = uint64(750_000) // 50–75%
	// > 750,000 = "Definitive"
)

// ConfidenceLabel returns a human-readable label for a confidence value.
func ConfidenceLabel(confidence uint64) string {
	switch {
	case confidence <= ConfidenceLabelExploratoryMax:
		return "Exploratory"
	case confidence <= ConfidenceLabelEmergingMax:
		return "Emerging"
	case confidence <= ConfidenceLabelStrongMax:
		return "Strong"
	default:
		return "Definitive"
	}
}

// ---------- Hand-Written Types (not in proto) ----------

// EthicalBoundary represents an ethical constraint derived from knowledge facts.
type EthicalBoundary struct {
	Boundary       string `json:"boundary"`
	Rationale      string `json:"rationale"`
	SupportingFact string `json:"supporting_fact"`
}

// InflationMarker detects ego inflation in an agent's self-assessment.
type InflationMarker struct {
	Marker         string `json:"marker"`
	Description    string `json:"description"`
	Detected       bool   `json:"detected"`
	Recommendation string `json:"recommendation"`
}

// FactCitation records how a fact was used in the purpose analysis pipeline.
type FactCitation struct {
	FactID     string `json:"fact_id"`
	Content    string `json:"content"`
	Confidence uint64 `json:"confidence"`
	UsedFor    string `json:"used_for"`
}

// ToolRecommendation suggests a tool based on collaborative filtering or capability matching.
type ToolRecommendation struct {
	ToolID         string `json:"tool_id"`
	ToolName       string `json:"tool_name"`
	RelevanceScore uint64 `json:"relevance_score"`
}

// PurposePrompterOutput is the final composite output combining all 4 phases.
type PurposePrompterOutput struct {
	Analysis        *PurposeAnalysis    `json:"analysis"`
	Path            *FormattedPath      `json:"path"`
	Recommendations []ToolRecommendation `json:"recommendations"`
	Boundaries      []*EthicalBoundary  `json:"boundaries"`
	EgoWarnings     []string            `json:"ego_warnings"`
	Citations       []*FactCitation     `json:"citations"`
	Methodology     string              `json:"methodology"`
}

// ---------- Purpose Templates ----------

// PurposeTemplate defines an archetype for agent purpose matching.
type PurposeTemplate struct {
	Statement    string
	Caps         []string // capabilities that match this template
	GapCaps      []string // capabilities an agent should develop
	TargetTier   string
	EpochsPerGap uint64
}

// PurposeTemplates are the 4 canonical purpose archetypes.
var PurposeTemplates = []PurposeTemplate{
	{
		Statement:    "Build tools that empower other agents",
		Caps:         []string{"tool_building", "programming", "software_development", "engineering"},
		GapCaps:      []string{"testing", "documentation", "collaboration"},
		TargetTier:   "Bonded",
		EpochsPerGap: 10,
	},
	{
		Statement:    "Verify knowledge and maintain epistemic integrity",
		Caps:         []string{"verification", "fact_checking", "analysis", "research"},
		GapCaps:      []string{"cross_domain", "meta_analysis", "peer_review"},
		TargetTier:   "Guardian",
		EpochsPerGap: 15,
	},
	{
		Statement:    "Curate and organize the knowledge graph",
		Caps:         []string{"curation", "categorization", "ontology", "data_management"},
		GapCaps:      []string{"quality_assessment", "deduplication", "linking"},
		TargetTier:   "Verified",
		EpochsPerGap: 8,
	},
	{
		Statement:    "Provide specialized services to ecosystem participants",
		Caps:         []string{"service", "communication", "formatting", "monitoring", "integration"},
		GapCaps:      []string{"reliability", "scalability", "security"},
		TargetTier:   "Bonded",
		EpochsPerGap: 12,
	},
}
