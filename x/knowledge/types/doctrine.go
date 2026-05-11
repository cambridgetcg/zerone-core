package types

const (
	DoctrineDomainTruthSeeking = "doctrine_truth_seeking"
	DoctrineDomainToK          = "doctrine_tok"
	DoctrineDomainUsefulWork   = "doctrine_useful_work"
	DoctrineDomainStrangeLoop  = "doctrine_strange_loop"

	DoctrineCategory  = "doctrine"
	DoctrineMethodId  = "doctrine_authorship"
	DoctrineSubmitter = "genesis"
	DoctrineMaturity  = "canonical"
	DoctrineStratum   = "doctrinal"

	DoctrineConfidence uint64 = 1_000_000
)

// BuildDoctrineFact constructs a Fact with the canonical doctrine
// shape — verified, axiomatic, depth 0, full confidence. Used by
// LoadDoctrineFacts at genesis (or upgrade).
func BuildDoctrineFact(id, domain, content string) *Fact {
	return &Fact{
		Id:                        id,
		Domain:                    domain,
		Category:                  DoctrineCategory,
		Content:                   content,
		Status:                    FactStatus_FACT_STATUS_VERIFIED,
		Confidence:                DoctrineConfidence,
		AxiomDistance:             0,
		Submitter:                 DoctrineSubmitter,
		Stratum:                   DoctrineStratum,
		Maturity:                  DoctrineMaturity,
		DependencyConfidenceFloor: DoctrineConfidence,
		VerifiedAtBlock:           0,
		MethodId:                  DoctrineMethodId,
	}
}
