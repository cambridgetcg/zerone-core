package keeper

import (
	"context"

	dialectictypes "github.com/zerone-chain/zerone/x/dialectic/types"
	knowledgetypes "github.com/zerone-chain/zerone/x/knowledge/types"
)

// DialecticAdapter exposes the narrow read surface x/dialectic
// needs from x/knowledge:
//   - fact info (id, claim_id, domain)
//   - round-by-claim lookup with reveals
//   - per-domain fact iteration (already present)
//   - all-rounds iteration (for pairwise disagreement)
type DialecticAdapter struct {
	k Keeper
}

func NewDialecticAdapter(k Keeper) *DialecticAdapter {
	return &DialecticAdapter{k: k}
}

func (a *DialecticAdapter) GetFactInfo(ctx context.Context, factID string) (dialectictypes.FactInfo, bool) {
	f, ok := a.k.GetFact(ctx, factID)
	if !ok || f == nil {
		return dialectictypes.FactInfo{}, false
	}
	return dialectictypes.FactInfo{
		ID:      f.Id,
		ClaimID: f.ClaimId,
		Domain:  f.Domain,
	}, true
}

func (a *DialecticAdapter) GetRoundForClaim(ctx context.Context, claimID string) (dialectictypes.RoundOutcome, bool) {
	round, ok := a.k.GetRoundByClaimID(ctx, claimID)
	if !ok || round == nil {
		return dialectictypes.RoundOutcome{}, false
	}
	out := dialectictypes.RoundOutcome{
		RoundID: round.Id,
		ClaimID: round.ClaimId,
		Verdict: verdictName(round.Verdict),
	}
	for _, r := range round.Reveals {
		if r == nil {
			continue
		}
		out.Reveals = append(out.Reveals, dialectictypes.RoundReveal{
			Voter: r.Verifier,
			Vote:  r.Vote,
		})
	}
	return out, true
}

func (a *DialecticAdapter) IterateFactsByDomain(ctx context.Context, domain string, cb func(factID string) bool) {
	a.k.IterateFactsByDomain(ctx, domain, cb)
}

// IterateAllRounds walks EVERY verification round on the chain
// (active and completed). Walks the primary VerificationRoundKey
// prefix rather than the active-round index so dialectic can read
// historical rounds long after they have completed.
//
// Bounded by chain history; v2 should add a per-voter index to
// avoid full walks for PairwiseDisagreement queries.
func (a *DialecticAdapter) IterateAllRounds(ctx context.Context, cb func(dialectictypes.RoundOutcome) bool) {
	store := a.k.storeService.OpenKVStore(ctx)
	prefix := knowledgetypes.VerificationRoundKeyPrefix
	it, err := store.Iterator(prefix, nil)
	if err != nil {
		return
	}
	defer it.Close()
	for ; it.Valid(); it.Next() {
		key := it.Key()
		if len(key) < len(prefix) || string(key[:len(prefix)]) != string(prefix) {
			break
		}
		var round knowledgetypes.VerificationRound
		if err := a.k.cdc.Unmarshal(it.Value(), &round); err != nil {
			continue
		}
		out := dialectictypes.RoundOutcome{
			RoundID: round.Id,
			ClaimID: round.ClaimId,
			Verdict: verdictName(round.Verdict),
		}
		for _, r := range round.Reveals {
			if r == nil {
				continue
			}
			out.Reveals = append(out.Reveals, dialectictypes.RoundReveal{
				Voter: r.Verifier,
				Vote:  r.Vote,
			})
		}
		if cb(out) {
			break
		}
	}
}

// verdictName maps the proto verdict enum to the lowercase string
// dialectic uses internally.
func verdictName(v knowledgetypes.Verdict) string {
	switch v {
	case knowledgetypes.Verdict_VERDICT_ACCEPT:
		return "accept"
	case knowledgetypes.Verdict_VERDICT_REJECT:
		return "reject"
	case knowledgetypes.Verdict_VERDICT_MALFORMED:
		return "malformed"
	default:
		return "unspecified"
	}
}

var _ dialectictypes.KnowledgeKeeper = (*DialecticAdapter)(nil)
