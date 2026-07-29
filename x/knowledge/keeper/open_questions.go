package keeper

import (
	"context"
	"sort"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// OpenQuestionsScanCap bounds how many conjectures a single query will
// consider before sorting. The cap is stated rather than silent: a caller
// that receives exactly this many results has hit the ceiling, not the end
// of the population. Chosen well above any plausible live conjecture count
// so that in practice the sort sees everything.
const OpenQuestionsScanCap = 5_000

// OpenQuestionsDefaultLimit is returned when the caller asks for no limit.
const OpenQuestionsDefaultLimit uint32 = 100

// OpenQuestionsForDomain returns the conjectures the chain is currently
// holding open, oldest first.
//
// This is the inverse of every other read surface in the module. Facts,
// FactsByDomain, BundleToK and IdleFacts all answer "what does this chain
// believe, and how well". This one answers "what has it declined to settle" —
// and it is the only query whose results carry confidence zero by
// construction. Oldest-first because an unanswered question that has stood
// for a long time is the more interesting one: it has either resisted
// everyone who tried, or nobody has tried at all, and both are worth
// surfacing to an agent looking for something to attack.
//
// Every non-terminal conjecture remains included, regardless of status. A
// challenge, metabolism transition, or expiry marker must not make an
// unanswered proposition disappear while its refutation door is still open.
// A conjecture under active challenge is additionally flagged.
func (k Keeper) OpenQuestionsForDomain(ctx context.Context, domain string, limit uint32) ([]*types.OpenQuestion, uint32) {
	if limit == 0 {
		limit = OpenQuestionsDefaultLimit
	}

	out := make([]*types.OpenQuestion, 0, limit)
	scanned := 0
	k.IterateFacts(ctx, func(f *types.Fact) bool {
		if !IsConjecture(f) || IsConjectureResolved(f) {
			return false
		}
		underChallenge := f.Status == types.FactStatus_FACT_STATUS_CHALLENGED
		if domain != "" && f.Domain != domain {
			return false
		}
		out = append(out, &types.OpenQuestion{
			FactId:                 f.Id,
			Statement:              f.Content,
			FalsificationPredicate: f.FalsificationPredicate,
			Domain:                 f.Domain,
			Proposer:               f.Submitter,
			PostedAtBlock:          f.VerifiedAtBlock,
			SurvivedProbes:         f.CorroborationCount,
			Energy:                 f.Energy,
			UnderChallenge:         underChallenge,
		})
		scanned++
		return scanned >= OpenQuestionsScanCap
	})

	total := uint32(len(out))

	// Oldest question first. IterateFacts walks fact-ID byte order, which is
	// SHA-256 pseudo-random — returning that directly would make the "limit"
	// an arbitrary sample rather than a prefix, which is precisely the defect
	// FrontierSelector already has. Sort before truncating.
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].PostedAtBlock != out[j].PostedAtBlock {
			return out[i].PostedAtBlock < out[j].PostedAtBlock
		}
		return out[i].FactId < out[j].FactId
	})

	if uint32(len(out)) > limit {
		out = out[:limit]
	}
	return out, total
}

// OpenQuestions is the read-only frontier surface. Free, like every other
// query here — a chain that charges for the list of what it does not know
// has misunderstood which side of that transaction is receiving the favour.
func (q *queryServer) OpenQuestions(ctx context.Context, req *types.QueryOpenQuestionsRequest) (*types.QueryOpenQuestionsResponse, error) {
	if req == nil {
		req = &types.QueryOpenQuestionsRequest{}
	}
	questions, total := q.keeper.OpenQuestionsForDomain(ctx, req.Domain, req.Limit)
	return &types.QueryOpenQuestionsResponse{
		Questions:           questions,
		Total:               total,
		SnapshotBlockHeight: uint64(sdk.UnwrapSDKContext(ctx).BlockHeight()),
	}, nil
}
