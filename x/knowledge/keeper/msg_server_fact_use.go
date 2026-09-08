package keeper

import (
	"context"
	"fmt"
	"math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// Admission is separate from ReportDemand. The generated message signer is the
// consumer; signature/account/fee/capability checks belong to SDK ante, not a
// caller-supplied receipt or a keeper-level impersonation of signature checking.
func (k Keeper) admitFactUse(ctx context.Context, consumer string) (*types.Params, *types.FactUsePruningState, uint64, error) {
	if _, err := types.CanonicalFactUseConsumer(consumer); err != nil {
		return nil, nil, 0, err
	}
	p, err := k.getFactUseParams(ctx)
	if err != nil {
		return nil, nil, 0, err
	}
	if err := p.ValidateFactUsePolicy(); err != nil {
		return nil, nil, 0, err
	}
	if !p.FactUseEnabled {
		return nil, nil, 0, fmt.Errorf("fact-use feedback is disabled")
	}
	admitted := false
	for _, account := range p.FactUseConsumers {
		if account == consumer {
			admitted = true
			break
		}
	}
	if !admitted {
		return nil, nil, 0, fmt.Errorf("consumer is not in the fact-use cohort")
	}
	state, err := k.GetFactUsePruningState(ctx)
	if err != nil {
		return nil, nil, 0, err
	}
	if err := types.ValidateFactUseNonEconomicState(p, state, false); err != nil {
		return nil, nil, 0, err
	}
	height := sdk.UnwrapSDKContext(ctx).BlockHeight()
	if height <= 0 {
		return nil, nil, 0, fmt.Errorf("fact use requires a positive committed block height")
	}
	epoch, err := types.FactUseEpoch(height, p.FitnessEpochBlocks)
	return p, state, epoch, err
}

func (m *msgServer) ReportFactUse(ctx context.Context, msg *types.MsgReportFactUse) (*types.MsgReportFactUseResponse, error) {
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}
	cache, write := sdk.UnwrapSDKContext(ctx).CacheContext()
	p, state, epoch, err := m.keeper.admitFactUse(cache, msg.Consumer)
	if err != nil {
		return nil, err
	}
	fact, found, err := m.keeper.getFeedbackFact(cache, msg.FactId)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, fmt.Errorf("fact not found: %s", msg.FactId)
	}
	if _, found, err := m.keeper.GetFactUseReceipt(cache, epoch, msg.Consumer, msg.FactId); err != nil {
		return nil, err
	} else if found {
		return nil, fmt.Errorf("fact use already reported in this epoch")
	}
	total, global, account, err := m.keeper.factUseAdmissionCounts(cache, epoch, msg.Consumer)
	if err != nil {
		return nil, err
	}
	if err := types.ValidateFactUseNonEconomicState(p, state, total > 0); err != nil {
		return nil, err
	}
	if total >= types.MaxFactUseRetainedReceipts {
		return nil, fmt.Errorf("fact-use retained capacity reached")
	}
	if global >= p.FactUseMaxPerEpoch {
		return nil, fmt.Errorf("global fact-use epoch quota reached")
	}
	if account >= p.FactUseMaxPerConsumerEpoch {
		return nil, fmt.Errorf("consumer fact-use epoch quota reached")
	}
	if fact.QueryCount == math.MaxUint64 || fact.QueryCountEpoch == math.MaxUint64 {
		return nil, fmt.Errorf("fact-use statistic overflow")
	}
	expiry, err := types.FactUseExpiryHeight(epoch, p.FitnessEpochBlocks)
	if err != nil {
		return nil, err
	}
	r := &types.FactUseReceipt{
		Version: types.FactUseReceiptVersion, Epoch: epoch, Consumer: msg.Consumer,
		FactId: msg.FactId, UseHeight: uint64(cache.BlockHeight()), ExpiryHeight: expiry,
		Rating: types.FactUseRating_FACT_USE_RATING_UNRATED,
	}
	fact.QueryCount++
	fact.QueryCountEpoch++
	if err := m.keeper.writeFeedbackFact(cache, fact); err != nil {
		return nil, err
	}
	if err := m.keeper.setFactUseReceipt(cache, r); err != nil {
		return nil, err
	}
	store := m.keeper.storeService.OpenKVStore(cache)
	if err := writeFactUseCount(store, types.FactUseEpochCountKey(epoch), global+1); err != nil {
		return nil, err
	}
	if err := writeFactUseCount(store, types.FactUseConsumerCountKey(epoch, msg.Consumer), account+1); err != nil {
		return nil, err
	}
	state.EverReported = true // Permanent, including after disable and complete pruning.
	if err := m.keeper.setFactUsePruningState(cache, state); err != nil {
		return nil, err
	}
	cache.EventManager().EmitEvent(sdk.NewEvent("zerone.knowledge.fact_use_reported",
		sdk.NewAttribute("consumer", msg.Consumer), sdk.NewAttribute("fact_id", msg.FactId),
		sdk.NewAttribute("epoch", fmt.Sprint(epoch)), sdk.NewAttribute("use_height", fmt.Sprint(r.UseHeight)),
		sdk.NewAttribute("expiry_height", fmt.Sprint(expiry)), sdk.NewAttribute("provenance", "signed_self_report"),
		sdk.NewAttribute("economic_credit", "none"),
	))
	write()
	return &types.MsgReportFactUseResponse{Receipt: r}, nil
}

func (m *msgServer) rateFactUse(ctx context.Context, msg *types.MsgRateFact) (*types.MsgRateFactResponse, error) {
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}
	cache, write := sdk.UnwrapSDKContext(ctx).CacheContext()
	_, state, epoch, err := m.keeper.admitFactUse(cache, msg.Rater)
	if err != nil {
		return nil, err
	}
	r, found, err := m.keeper.GetFactUseReceipt(cache, epoch, msg.Rater, msg.FactId)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, fmt.Errorf("no current-epoch signed fact-use receipt")
	}
	if !state.EverReported {
		return nil, fmt.Errorf("fact-use receipt without ever_reported")
	}
	height := uint64(cache.BlockHeight())
	if r.UseHeight > height || r.ExpiryHeight <= height {
		return nil, fmt.Errorf("fact-use receipt outside rating window")
	}
	if r.Rating != types.FactUseRating_FACT_USE_RATING_UNRATED {
		return nil, fmt.Errorf("fact-use receipt already rated")
	}
	fact, found, err := m.keeper.getFeedbackFact(cache, msg.FactId)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, fmt.Errorf("fact not found: %s", msg.FactId)
	}
	if msg.Useful {
		if fact.SatisfactionUp == math.MaxUint64 || fact.SatisfactionUpEpoch == math.MaxUint64 {
			return nil, fmt.Errorf("rating statistic overflow")
		}
		fact.SatisfactionUp++
		fact.SatisfactionUpEpoch++
		r.Rating = types.FactUseRating_FACT_USE_RATING_USEFUL
	} else {
		if fact.SatisfactionDown == math.MaxUint64 || fact.SatisfactionDownEpoch == math.MaxUint64 {
			return nil, fmt.Errorf("rating statistic overflow")
		}
		fact.SatisfactionDown++
		fact.SatisfactionDownEpoch++
		r.Rating = types.FactUseRating_FACT_USE_RATING_NOT_USEFUL
	}
	r.RatingHeight = height
	if err := m.keeper.writeFeedbackFact(cache, fact); err != nil {
		return nil, err
	}
	// Retain the rated semantic-dedup marker. A fresh nonce cannot re-report it.
	if err := m.keeper.setFactUseReceipt(cache, r); err != nil {
		return nil, err
	}
	cache.EventManager().EmitEvent(sdk.NewEvent("zerone.knowledge.fact_rated",
		sdk.NewAttribute("fact_id", msg.FactId), sdk.NewAttribute("rater", msg.Rater),
		sdk.NewAttribute("useful", fmt.Sprint(msg.Useful)), sdk.NewAttribute("epoch", fmt.Sprint(epoch)),
		sdk.NewAttribute("provenance", "signed_self_report"), sdk.NewAttribute("economic_credit", "none"),
	))
	write()
	return &types.MsgRateFactResponse{}, nil
}

// Cache the complete SDK module effects even when invoked directly by another
// keeper. Bank keepers must honor the supplied SDK context as in normal ante/tx.
func (m *msgServer) ChallengeFact(ctx context.Context, msg *types.MsgChallengeFact) (*types.MsgChallengeFactResponse, error) {
	cache, write := sdk.UnwrapSDKContext(ctx).CacheContext()
	resp, err := m.challengeFact(cache, msg)
	if err != nil {
		return nil, err
	}
	write()
	return resp, nil
}

func (m *msgServer) ChallengeProvisionalFact(ctx context.Context, msg *types.MsgChallengeProvisionalFact) (*types.MsgChallengeProvisionalFactResponse, error) {
	cache, write := sdk.UnwrapSDKContext(ctx).CacheContext()
	resp, err := m.challengeProvisionalFact(cache, msg)
	if err != nil {
		return nil, err
	}
	write()
	return resp, nil
}

func (k Keeper) setChallengeStatus(ctx context.Context, fact *types.Fact, next types.FactStatus, cause, causeID string) error {
	if err := k.RecordStatusTransition(ctx, &types.StatusTransition{
		FactId: fact.Id, PriorStatus: fact.Status, NewStatus: next,
		BlockHeight: uint64(sdk.UnwrapSDKContext(ctx).BlockHeight()), CauseEventType: cause, CauseId: causeID,
	}); err != nil {
		return err
	}
	fact.Status = next
	return k.writeFeedbackFact(ctx, fact)
}

func (k Keeper) validateExistingChallengeEvidence(ctx context.Context, ids []string) error {
	if err := types.ValidateChallengeEvidenceIDs(ids); err != nil {
		return err
	}
	for _, id := range ids {
		if _, found, err := k.getFeedbackFact(ctx, id); err != nil {
			return err
		} else if !found {
			return fmt.Errorf("challenge evidence fact %s not found", id)
		}
	}
	return nil
}
