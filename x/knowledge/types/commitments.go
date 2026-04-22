package types

// Commitment category constants.
const (
	CommitmentCategoryPrinciple      = "principle"      // a value the chain holds
	CommitmentCategoryProcedural     = "procedural"     // how the chain must operate
	CommitmentCategoryConstitutional = "constitutional" // cryptographically-enforced rule
	CommitmentCategoryAspiration     = "aspiration"     // stated direction, not yet enforced
)

// DefaultCommitments returns an initial set of normative commitments the chain
// holds at genesis. These are values, not facts — schema-distinct so the
// chain cannot mint currency by deriving from them as if they were 100%-
// confident truth claims.
//
// Honestly short. The chain acknowledges these as stated positions;
// governance extends and amends the set over time.
func DefaultCommitments() []*NormativeCommitment {
	return []*NormativeCommitment{
		{
			Id:        "NC-DUAL-KEY-RESEARCH",
			Statement: "Spending from the research fund requires BOTH human and AI signatures; neither alone can move it.",
			Rationale: "Dual authorization over the research fund is cryptographically enforced in app-layer key management. The rule is a value the chain has committed to, not a fact about the world.",
			Category:  CommitmentCategoryConstitutional,
			Tags:      []string{"governance", "research_fund", "ai_oversight", "human_oversight"},
			Version:   1,
			Active:    true,
		},
		{
			Id:        "NC-TRANSPARENCY",
			Statement: "Every on-chain state transition is publicly visible; the history is permanent.",
			Rationale: "Transparency is a stance the chain takes — it is part of what ZERONE is, not a fact we discovered about the world. Legibility of state is enforced by the blockchain architecture itself.",
			Category:  CommitmentCategoryProcedural,
			Tags:      []string{"governance", "legibility", "permanence"},
			Version:   1,
			Active:    true,
		},
		{
			Id:        "NC-METHODOLOGY-OVER-STATEMENT",
			Statement: "Verification rewards well-reasoned method-compliance, not declared truth. Knowledge is tested, not stored.",
			Rationale: "This is the core epistemic stance of the chain. See Phase 1 of the methodology framework: an axiom is a method of seeking, not a statement of what is true. This commitment is the reason the 777 'axioms' were reclassified.",
			Category:  CommitmentCategoryPrinciple,
			Tags:      []string{"epistemology", "methodology", "foundations"},
			Version:   1,
			Active:    true,
		},
		{
			Id:        "NC-FALSIFICATION-IS-PROGRESS",
			Statement: "A disproven fact is not a failure of the chain. It is evidence that the verification mechanism is working.",
			Rationale: "Under a Popperian model, falsification is how knowledge advances. The chain's economic and cultural stance must reflect that: vindication of a minority voter is rewarded; cascade of CONTESTED status on descendants is a feature, not a bug.",
			Category:  CommitmentCategoryPrinciple,
			Tags:      []string{"epistemology", "popper", "falsification"},
			Version:   1,
			Active:    true,
		},
		{
			Id:        "NC-IS-OUGHT-WALL",
			Statement: "Normative commitments are schema-distinct from factual claims. A value cannot be derived from a fact, and a fact cannot claim the confidence of a value.",
			Rationale: "Hume's is-ought distinction, enforced as code. This commitment is itself the reason NormativeCommitment exists as a separate type. If governance ever considers collapsing the two, that proposal must be argued explicitly.",
			Category:  CommitmentCategoryPrinciple,
			Tags:      []string{"epistemology", "hume", "is_ought"},
			Version:   1,
			Active:    true,
		},
	}
}
