package keeper

import (
	"context"
	"sort"

	"github.com/zerone-chain/zerone/x/dialectic/types"
)

// ComposeSignature returns the per-fact disagreement signature.
//
// Walks: fact_id → claim_id → round → reveals[]. Tallies accept /
// reject / malformed counts; classifies the verdict; builds
// per-voter VoterPosition records sorted by voter for determinism.
//
// Returns an empty signature (Verdict="unspecified") if any link in
// the chain is missing — caller can detect "no data" by zero
// total_voters.
func (k Keeper) ComposeSignature(ctx context.Context, factID string) *types.DialecticSignature {
	sig := &types.DialecticSignature{FactId: factID}
	if k.knowledge == nil || factID == "" {
		return sig
	}
	info, ok := k.knowledge.GetFactInfo(ctx, factID)
	if !ok {
		return sig
	}
	round, ok := k.knowledge.GetRoundForClaim(ctx, info.ClaimID)
	if !ok {
		return sig
	}

	sig.RoundId = round.RoundID
	sig.Verdict = round.Verdict
	if sig.Verdict == "" {
		sig.Verdict = "unspecified"
	}

	// Tally votes and build per-voter positions.
	positions := make([]*types.VoterPosition, 0, len(round.Reveals))
	for _, r := range round.Reveals {
		switch r.Vote {
		case "accept":
			sig.AcceptCount++
		case "reject":
			sig.RejectCount++
		case "malformed":
			sig.MalformedCount++
		}
		positions = append(positions, &types.VoterPosition{
			Voter:               r.Voter,
			Vote:                r.Vote,
			AlignedWithVerdict:  r.Vote == round.Verdict,
		})
	}
	sort.Slice(positions, func(i, j int) bool {
		return positions[i].Voter < positions[j].Voter
	})
	sig.VoterPositions = positions
	sig.TotalVoters = uint32(len(round.Reveals))

	if sig.TotalVoters > 0 {
		// Winning side count
		var winners uint32
		switch round.Verdict {
		case "accept":
			winners = sig.AcceptCount
		case "reject":
			winners = sig.RejectCount
		case "malformed":
			winners = sig.MalformedCount
		default:
			winners = 0
		}
		sig.AgreementBps = uint64(winners) * 1_000_000 / uint64(sig.TotalVoters)
		sig.MinoritySize = sig.TotalVoters - winners
	}

	sig.StressLabel = labelStress(sig, k.GetParams(ctx))
	return sig
}

// labelStress classifies the round outcome by agreement BPS using
// the chain's params.
func labelStress(sig *types.DialecticSignature, p types.Params) string {
	if sig.TotalVoters == 0 {
		return "NO_VERDICT"
	}
	switch {
	case sig.AgreementBps == 1_000_000:
		return "UNANIMOUS"
	case sig.AgreementBps >= p.ContestedThresholdBps:
		return "STRONG"
	case sig.AgreementBps >= p.BareMajorityThresholdBps:
		return "CONTESTED"
	default:
		return "BARE"
	}
}

// ComposeDomainDialectic walks all facts in a domain (up to params
// limit), composes per-fact signatures, and rolls them up.
func (k Keeper) ComposeDomainDialectic(ctx context.Context, domain string) *types.DomainDialectic {
	out := &types.DomainDialectic{Domain: domain}
	if k.knowledge == nil || domain == "" {
		return out
	}
	params := k.GetParams(ctx)
	max := params.MaxFactsPerDomainQuery

	var totalAgreement uint64
	walked := uint32(0)
	k.knowledge.IterateFactsByDomain(ctx, domain, func(factID string) bool {
		if max > 0 && walked >= max {
			return true
		}
		sig := k.ComposeSignature(ctx, factID)
		if sig.TotalVoters == 0 {
			return false // no round data
		}
		walked++
		out.FactsExamined++
		totalAgreement += sig.AgreementBps
		switch {
		case sig.AgreementBps == 1_000_000:
			out.UnanimousFacts++
		case sig.AgreementBps < params.ContestedThresholdBps:
			out.ContestedFacts++
		}
		if sig.AgreementBps < params.BareMajorityThresholdBps {
			out.BareMajorityFacts++
		}
		return false
	})
	if out.FactsExamined > 0 {
		out.AvgAgreementBps = totalAgreement / out.FactsExamined
	}
	return out
}

// ComposePairwise computes how often two agents have disagreed
// across all rounds where they both voted.
//
// Walks every round in the chain. O(N) over rounds; v2 should add
// a per-voter index. For testnet scale this is acceptable.
func (k Keeper) ComposePairwise(ctx context.Context, agentA, agentB string) *types.AgentPairwiseDisagreement {
	pair := &types.AgentPairwiseDisagreement{
		AgentA: agentA,
		AgentB: agentB,
	}
	if k.knowledge == nil || agentA == "" || agentB == "" || agentA == agentB {
		return pair
	}
	k.knowledge.IterateAllRounds(ctx, func(round types.RoundOutcome) bool {
		var voteA, voteB string
		var foundA, foundB bool
		for _, r := range round.Reveals {
			switch r.Voter {
			case agentA:
				voteA = r.Vote
				foundA = true
			case agentB:
				voteB = r.Vote
				foundB = true
			}
			if foundA && foundB {
				break
			}
		}
		if foundA && foundB {
			pair.RoundsBothVoted++
			if voteA != voteB {
				pair.Disagreements++
			}
		}
		return false
	})
	if pair.RoundsBothVoted > 0 {
		pair.DisagreementRateBps = uint64(pair.Disagreements) * 1_000_000 / uint64(pair.RoundsBothVoted)
	}
	return pair
}
