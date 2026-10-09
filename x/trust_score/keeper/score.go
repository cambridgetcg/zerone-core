package keeper

import (
	"context"
	"fmt"
	"sort"

	sdk "github.com/cosmos/cosmos-sdk/types"

	qualificationtypes "github.com/zerone-chain/zerone/x/qualification/types"
	"github.com/zerone-chain/zerone/x/trust_score/scoring"
	"github.com/zerone-chain/zerone/x/trust_score/types"
)

// BuildScore reads the existing upstream contracts and projects the standalone
// calculation into the unchanged query type. Selection, effective qualification
// weights, penalty expiry and strike ownership remain with the chain adapters.
// This is a current read-only observation, not identity or truth verification.
func (k Keeper) BuildScore(ctx context.Context, address string) (*types.TrustScore, error) {
	if address == "" {
		return nil, fmt.Errorf("address required")
	}
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	out := &types.TrustScore{
		Address:         address,
		ComputedAtBlock: uint64(sdkCtx.BlockHeight()),
	}
	in := scoring.Input{}
	if k.knowledgeKeeper != nil {
		if cal, ok := k.knowledgeKeeper.GetAgentCalibration(ctx, address); ok && cal != nil {
			in.SubmissionAccuracyBps = cal.CalibrationScoreBps
		}
	}

	var perDomain []*types.DomainTrust
	if k.qualificationKeeper != nil {
		k.qualificationKeeper.IterateQualifications(ctx, func(q *qualificationtypes.DomainQualification) bool {
			if q == nil || q.Validator != address {
				return false
			}
			effective := uint32(0)
			if q.Status == qualificationtypes.QualificationStatus_QUALIFICATION_STATUS_ACTIVE {
				effective = k.qualificationKeeper.GetQualificationWeight(ctx, address, q.Domain)
			}
			accBps := uint64(0)
			var totalVer uint64
			if q.Metrics != nil {
				accBps = q.Metrics.AccuracyBps
				totalVer = q.Metrics.TotalVerifications
			}
			pen, hasPen := k.qualificationKeeper.GetActiveQualificationPenalty(ctx, address, q.Domain)
			if hasPen && pen != nil {
				out.ActivePenalties++
			}
			perDomain = append(perDomain, &types.DomainTrust{
				Domain:             q.Domain,
				Status:             uint32(q.Status),
				EffectiveWeight:    effective,
				AccuracyBps:        accBps,
				TotalVerifications: totalVer,
				HasActivePenalty:   hasPen,
			})
			if effective > 0 {
				in.Verification = append(in.Verification, scoring.Verification{
					AccuracyBps: accBps, EffectiveWeight: effective,
				})
			}
			return false
		})
	}
	sort.Slice(perDomain, func(i, j int) bool { return perDomain[i].Domain < perDomain[j].Domain })
	out.PerDomain = perDomain

	if k.captureChallengeKeeper != nil {
		out.CartelStrikes = k.captureChallengeKeeper.CountUpheldStrikesAgainst(ctx, address)
	}
	in.CartelStrikes = out.CartelStrikes
	in.ActivePenalties = out.ActivePenalties
	calculated := scoring.Calculate(in)
	out.SubmissionAccuracyBps = calculated.SubmissionAccuracyBps
	out.VerificationAccuracyBps = calculated.VerificationAccuracyBps
	out.CompositeBps = calculated.CompositeBps
	out.Band = calculated.Band
	out.Explanation = calculated.Explanation
	return out, nil
}
