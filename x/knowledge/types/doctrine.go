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

	// DoctrineEnergy is the energy every doctrine fact is born with. Doctrine
	// is exempt from metabolism (ProcessMetabolism skips it) and lives by the
	// creed pin + amendment LIP, so this is the "fully alive" reading rather
	// than a decaying balance. Matches the default MetabolismEnergyCap so a
	// canonical fact never displays as starved. (Before 2026-07-12 the field
	// was omitted, so all 47 genesis facts were born at energy 0 and marched
	// toward PRUNED — the born-starving bug.)
	DoctrineEnergy uint64 = 1_000_000
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
		Energy:                    DoctrineEnergy,
	}
}
