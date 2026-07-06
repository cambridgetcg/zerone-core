package keeper

import (
	"context"
	"encoding/binary"
	"encoding/json"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// ─── Survival-gate escrow ─────────────────────────────────────────────────────
//
// On a witness-not-proof chain the only quality signal expensive to fake is a
// claim SURVIVING adversarial challenge. Acceptance is cheap to fake, so the
// submitter reward must not mint at accept — it is escrowed here and issued only
// when the fact survives: it wins a challenge (handleChallengeSurvival) or its
// challenge window closes unchallenged (SweepSurvivedRewards). If the fact is
// disproven, the reward is cancelled (handleChallengeDisproven) — a free clawback,
// because nothing was minted at accept. Issuance follows survival, not acceptance.

// SurvivalPendingReward is the submitter reward held until the fact survives.
// Stored as JSON under SurvivalPendingRewardPrefix (mirrors the vindication pattern).
type SurvivalPendingReward struct {
	ClaimId       string `json:"claim_id"`
	FactId        string `json:"fact_id"`
	Recipient     string `json:"recipient"`
	Amount        string `json:"amount"`         // uzrn, string for big.Int compat
	Category      string `json:"category"`
	PartnershipId string `json:"partnership_id"` // route through the partnership split on release, if set
	Deadline      uint64 `json:"deadline"`       // block height the challenge window closes
}

func survivalPendingKey(factId string) []byte {
	return append(append([]byte{}, types.SurvivalPendingRewardPrefix...), []byte(factId)...)
}

func survivalDeadlineKey(deadline uint64, factId string) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, deadline)
	key := append(append([]byte{}, types.SurvivalDeadlineIndexPrefix...), b...)
	return append(key, []byte(factId)...)
}

// SetSurvivalPendingReward stores the pending reward and its deadline-index entry.
func (k Keeper) SetSurvivalPendingReward(ctx context.Context, pr SurvivalPendingReward) {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := json.Marshal(pr)
	if err != nil {
		return
	}
	_ = store.Set(survivalPendingKey(pr.FactId), bz)
	_ = store.Set(survivalDeadlineKey(pr.Deadline, pr.FactId), []byte{0x01})
}

// GetSurvivalPendingReward returns the pending reward for a fact, if any.
func (k Keeper) GetSurvivalPendingReward(ctx context.Context, factId string) (SurvivalPendingReward, bool) {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(survivalPendingKey(factId))
	if err != nil || bz == nil {
		return SurvivalPendingReward{}, false
	}
	var pr SurvivalPendingReward
	if err := json.Unmarshal(bz, &pr); err != nil {
		return SurvivalPendingReward{}, false
	}
	return pr, true
}

func (k Keeper) deleteSurvivalPending(ctx context.Context, pr SurvivalPendingReward) {
	store := k.storeService.OpenKVStore(ctx)
	_ = store.Delete(survivalPendingKey(pr.FactId))
	_ = store.Delete(survivalDeadlineKey(pr.Deadline, pr.FactId))
}

// EscrowSubmitterReward records the submitter reward as pending (nothing minted)
// and stamps the fact's challenge window. Replaces the accept-time reward routing.
func (k Keeper) EscrowSubmitterReward(ctx context.Context, fact *types.Fact, claim *types.Claim) {
	if k.vestingRewardsKeeper == nil {
		return
	}
	params, err := k.GetParams(ctx)
	if err != nil {
		return
	}
	window := params.ChallengeDurationBlocks
	if window == 0 {
		window = 34_272 // ~1 day at 2.521s block time — defensive default
	}
	deadline := uint64(sdk.UnwrapSDKContext(ctx).BlockHeight()) + window
	k.SetSurvivalPendingReward(ctx, SurvivalPendingReward{
		ClaimId:       claim.Id,
		FactId:        fact.Id,
		Recipient:     claim.Submitter,
		Amount:        claim.Stake,
		Category:      claim.Category,
		PartnershipId: claim.PartnershipId,
		Deadline:      deadline,
	})
	fact.ChallengeWindowEnd = deadline
	_ = k.SetFact(ctx, fact)
}

// releaseSurvivalReward issues the escrowed reward exactly once (no-op if already
// released or cancelled). Called when a fact survives.
func (k Keeper) releaseSurvivalReward(ctx context.Context, factId string) {
	pr, found := k.GetSurvivalPendingReward(ctx, factId)
	if !found {
		return
	}
	k.routeSubmitterReward(ctx, pr)
	k.deleteSurvivalPending(ctx, pr)
	sdk.UnwrapSDKContext(ctx).EventManager().EmitEvent(sdk.NewEvent(
		"zerone.knowledge.survival_reward_released",
		sdk.NewAttribute("fact_id", factId),
		sdk.NewAttribute("recipient", pr.Recipient),
		sdk.NewAttribute("amount", pr.Amount),
	))
}

// cancelSurvivalReward drops a pending reward without issuing it — the fact fell to
// DISPROVEN or decayed before surviving. The clawback is free: nothing was minted.
func (k Keeper) cancelSurvivalReward(ctx context.Context, factId string) {
	pr, found := k.GetSurvivalPendingReward(ctx, factId)
	if !found {
		return
	}
	k.deleteSurvivalPending(ctx, pr)
	sdk.UnwrapSDKContext(ctx).EventManager().EmitEvent(sdk.NewEvent(
		"zerone.knowledge.survival_reward_cancelled",
		sdk.NewAttribute("fact_id", factId),
		sdk.NewAttribute("recipient", pr.Recipient),
	))
}

// routeSubmitterReward runs the original accept-time routing (direct vesting) at
// RELEASE time — survivors are paid exactly as before, just after they survive
// rather than on acceptance. The partnership-split branch retired with
// x/partnerships (slim cut); payment splits between collaborators are an
// agenttool-escrow concern, not consensus.
func (k Keeper) routeSubmitterReward(ctx context.Context, pr SurvivalPendingReward) {
	if k.vestingRewardsKeeper != nil {
		_ = k.vestingRewardsKeeper.CreateVestingScheduleFromKnowledge(ctx, pr.ClaimId, pr.FactId, pr.Recipient, pr.Amount, pr.Category)
	}
}

// SweepSurvivedRewards issues escrowed rewards for facts whose challenge window
// closed while still VERIFIED (survived unchallenged). Called from BeginBlocker.
// The pending entry is the source of truth for exactly-once issuance; the deadline
// index is an ordered scan hint, so only due entries (deadline <= height) are read.
func (k Keeper) SweepSurvivedRewards(ctx context.Context) {
	height := uint64(sdk.UnwrapSDKContext(ctx).BlockHeight())
	store := k.storeService.OpenKVStore(ctx)

	end := make([]byte, 8)
	binary.BigEndian.PutUint64(end, height+1) // exclusive: captures deadline <= height
	upper := append(append([]byte{}, types.SurvivalDeadlineIndexPrefix...), end...)
	iter, err := store.Iterator(types.SurvivalDeadlineIndexPrefix, upper)
	if err != nil {
		return
	}
	// Collect due factIDs first; do not mutate the store during iteration.
	var due []string
	prefixLen := len(types.SurvivalDeadlineIndexPrefix) + 8
	for ; iter.Valid(); iter.Next() {
		key := iter.Key()
		if len(key) <= prefixLen {
			continue
		}
		due = append(due, string(key[prefixLen:]))
	}
	iter.Close()

	for _, factId := range due {
		pr, found := k.GetSurvivalPendingReward(ctx, factId)
		if !found {
			continue // already released via a challenge-win
		}
		fact, ok := k.GetFact(ctx, factId)
		if !ok {
			k.cancelSurvivalReward(ctx, factId)
			continue
		}
		switch fact.Status {
		case types.FactStatus_FACT_STATUS_VERIFIED:
			k.releaseSurvivalReward(ctx, factId) // survived its window unchallenged
		case types.FactStatus_FACT_STATUS_CHALLENGED:
			// Mid-challenge at the deadline: leave the pending for the challenge to
			// resolve (win → release, disproven → cancel); drop only the stale index.
			_ = store.Delete(survivalDeadlineKey(pr.Deadline, factId))
		default:
			k.cancelSurvivalReward(ctx, factId) // disproven / expired / superseded — no reward
		}
	}
}
