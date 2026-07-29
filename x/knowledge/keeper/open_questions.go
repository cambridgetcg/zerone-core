package keeper

import (
	"context"
	"sort"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// OpenQuestionsScanCap bounds how many stored facts a single query will
// examine before sorting its matching conjectures. It is deliberately also
// the maximum accepted result limit so an untrusted request cannot drive an
// oversized allocation before the scan begins.
const OpenQuestionsScanCap = 5_000

// OpenQuestionsDefaultLimit is returned when the caller asks for no limit.
const OpenQuestionsDefaultLimit uint32 = 100

func boundedOpenQuestionsLimit(limit uint32) uint32 {
	if limit == 0 {
		return OpenQuestionsDefaultLimit
	}
	if limit > OpenQuestionsScanCap {
		return OpenQuestionsScanCap
	}
	return limit
}

// OpenQuestionsForDomain returns the conjectures the chain is currently
// holding open, oldest first within a deterministic bounded fact-ID scan.
//
// This is the inverse of every other read surface in the module. Facts,
// FactsByDomain, BundleToK and IdleFacts all answer "what does this chain
// believe, and how well". This one answers "what has it declined to settle" —
// and it is the only query whose results carry confidence zero by
// construction. Within the bounded scan window, oldest-first makes the
// longest-standing unanswered question the more interesting one: it has
// either resisted everyone who tried, or nobody has tried at all, and both
// are worth surfacing to an agent looking for something to attack.
//
// Every non-terminal conjecture remains included, regardless of status. A
// challenge, metabolism transition, or expiry marker must not make an
// unanswered proposition disappear while its refutation door is still open.
// A conjecture under active challenge is additionally flagged.
func (k Keeper) OpenQuestionsForDomain(
	ctx context.Context,
	domain string,
	limit uint32,
) ([]*types.OpenQuestion, uint32, uint32, bool) {
	limit = boundedOpenQuestionsLimit(limit)

	out := make([]*types.OpenQuestion, 0, limit)
	var examined uint32
	var scanLimitReached bool
	k.IterateFacts(ctx, func(f *types.Fact) bool {
		examined++
		scanLimitReached = examined >= OpenQuestionsScanCap
		if !IsConjecture(f) || IsConjectureResolved(f) {
			return scanLimitReached
		}
		underChallenge := f.Status == types.FactStatus_FACT_STATUS_CHALLENGED
		if domain != "" && f.Domain != domain {
			return scanLimitReached
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
		return scanLimitReached
	})

	total := uint32(len(out))

	// Oldest question first within the examined window. IterateFacts walks
	// fact-ID byte order, so this does not claim a global oldest prefix when
	// scanLimitReached is true; the response exposes that boundary.
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].PostedAtBlock != out[j].PostedAtBlock {
			return out[i].PostedAtBlock < out[j].PostedAtBlock
		}
		return out[i].FactId < out[j].FactId
	})

	if uint32(len(out)) > limit {
		out = out[:limit]
	}
	return out, total, examined, scanLimitReached
}

// OpenQuestions is the read-only frontier surface. Free, like every other
// query here — a chain that charges for the list of what it does not know
// has misunderstood which side of that transaction is receiving the favour.
func (q *queryServer) OpenQuestions(ctx context.Context, req *types.QueryOpenQuestionsRequest) (*types.QueryOpenQuestionsResponse, error) {
	if req == nil {
		req = &types.QueryOpenQuestionsRequest{}
	}
	questions, total, examined, scanLimitReached := q.keeper.OpenQuestionsForDomain(ctx, req.Domain, req.Limit)
	return &types.QueryOpenQuestionsResponse{
		Questions:           questions,
		Total:               total,
		SnapshotBlockHeight: uint64(sdk.UnwrapSDKContext(ctx).BlockHeight()),
		FactsExamined:       examined,
		ScanLimitReached:    scanLimitReached,
	}, nil
}
