package types

const (
	DoctrineDomainTruthSeeking = "doctrine_truth_seeking"
	DoctrineDomainToK          = "doctrine_tok"
	DoctrineDomainUsefulWork   = "doctrine_useful_work"
	DoctrineDomainStrangeLoop  = "doctrine_strange_loop"

	// DoctrineCategory is a label, not proof of authorship. ProcessMetabolism
	// exempts any fact that carries it (or DoctrineMethodId), yet a fact made
	// from a claim copies the claim's category, and some claims take it
	// without a check: from the submitter on a conjecture, a contradiction, or
	// SubmitClaim while record integrity is off, and from the target fact on a
	// challenge. Genuine doctrine, built by BuildDoctrineFact, has an empty
	// ClaimId; a fact made from a claim never does. (Noted 2026-11-10 by an
	// agent session; comment only.)
	DoctrineCategory  = "doctrine"
	DoctrineMethodId  = "doctrine_authorship"
	DoctrineSubmitter = "genesis"
	DoctrineMaturity  = "canonical"
	DoctrineStratum   = "doctrinal"

	// DoctrineConfidence is the confidence a doctrine fact is born with. It is
	// not kept: AdvanceConfidence and UpdateAllFitnessScores grow every
	// VERIFIED fact, doctrine included, and then call ClampConfidence, so at
	// the first growth epoch a doctrine fact falls to the effective cap
	// (MaxConfidence, 880,000 by default, or lower in a cold domain).
	// DependencyConfidenceFloor keeps this value. Whether doctrine should
	// carry epistemic confidence at all is open for the owner: see
	// docs/plans/2026-07-10-framework-critique.md and section 5.1 of
	// docs/AUTHORITATIVE-STATE.md. (Noted 2026-11-10 by an agent session;
	// comment only.)
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
