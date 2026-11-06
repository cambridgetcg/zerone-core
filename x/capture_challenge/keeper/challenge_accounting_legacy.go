package keeper

import (
	"context"
	"fmt"
	"math/big"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	"github.com/zerone-chain/zerone/x/capture_challenge/types"
)

// Legacy bodies below are verbatim from 77521a0daf14420836483ae9d8ea3fbcc1ae0d40
// (the capture_challenge subtree also matches 2e37c4c), with only function names
// changed. Keep their historical writes, errors, events and gas independent of
// the repaired paths until the consensus-repairs-v1 marker selects the repair.

func (k Keeper) advanceChallengePhasesLegacy(ctx sdk.Context, height uint64) {
	k.IterateChallenges(ctx, func(ch *types.CaptureChallenge) bool {
		switch ch.Status {
		case types.ChallengeStatus_CHALLENGE_STATUS_OPEN,
			types.ChallengeStatus_CHALLENGE_STATUS_EVIDENCE:
			// Transition to under_review at evidence deadline
			if height >= ch.EvidenceDeadline {
				ch.Status = types.ChallengeStatus_CHALLENGE_STATUS_UNDER_REVIEW
				k.SetChallenge(ctx, ch)
			}
		case types.ChallengeStatus_CHALLENGE_STATUS_UNDER_REVIEW:
			// Auto-expire at review deadline
			if height >= ch.ReviewDeadline {
				ch.Status = types.ChallengeStatus_CHALLENGE_STATUS_EXPIRED
				// Return stake on expiry
				challengerAddr, err := sdk.AccAddressFromBech32(ch.Challenger)
				if err == nil {
					stakeAmt, ok := new(big.Int).SetString(ch.Stake, 10)
					if ok && stakeAmt.Sign() > 0 {
						coins := sdk.NewCoins(sdk.NewCoin("uzrn", sdkmath.NewIntFromBigInt(stakeAmt)))
						_ = k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, challengerAddr, coins)
					}
				}
				k.SetChallenge(ctx, ch)
			}
		}
		return false
	})
}

func (k Keeper) autoSubmitChallengeLegacy(ctx context.Context, domain string, riskScore, hhi uint64, evidence string) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Check if there's already an active challenge for this domain
	existingIDs := k.GetChallengesByDomain(sdkCtx, domain)
	for _, id := range existingIDs {
		ch, found := k.GetChallenge(sdkCtx, id)
		if !found {
			continue
		}
		if ch.Status == types.ChallengeStatus_CHALLENGE_STATUS_EVIDENCE ||
			ch.Status == types.ChallengeStatus_CHALLENGE_STATUS_OPEN ||
			ch.Status == types.ChallengeStatus_CHALLENGE_STATUS_UNDER_REVIEW {
			return nil // Already has an active challenge
		}
	}

	// Use module account as auto-challenger
	moduleAddr := authtypes.NewModuleAddress(types.ModuleName).String()

	challengeID := GenerateChallengeID(moduleAddr, domain, sdkCtx.BlockHeight())

	params := k.GetParams(sdkCtx)
	height := uint64(sdkCtx.BlockHeight())

	challenge := &types.CaptureChallenge{
		Id:                challengeID,
		Challenger:        moduleAddr,
		Domain:            domain,
		AccusedValidators: []string{},
		Status:            types.ChallengeStatus_CHALLENGE_STATUS_UNDER_REVIEW,
		Stake:             "0",
		EvidenceDeadline:  height,
		ReviewDeadline:    height + params.ReviewPeriodBlocks,
		CreatedBlock:      height,
		Evidence: []*types.CaptureEvidence{
			{
				Description:    evidence,
				DataHash:       fmt.Sprintf("auto:%s:%d", domain, height),
				SubmittedBlock: height,
			},
		},
	}
	k.SetChallenge(sdkCtx, challenge)

	// Maintain domain index for lookups
	k.SetDomainIndex(sdkCtx, domain, challengeID)

	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			"zerone.capture_challenge.auto_challenge_submitted",
			sdk.NewAttribute("challenge_id", challengeID),
			sdk.NewAttribute("domain", domain),
			sdk.NewAttribute("risk_score", fmt.Sprintf("%d", riskScore)),
			sdk.NewAttribute("hhi", fmt.Sprintf("%d", hhi)),
		),
	)

	return nil
}

func (m msgServer) resolveChallengeLegacy(goCtx context.Context, msg *types.MsgResolveChallenge) (*types.MsgResolveChallengeResponse, error) {
	if m.GetAuthority() != msg.Authority {
		return nil, fmt.Errorf("%w: expected %s, got %s", types.ErrUnauthorized, m.GetAuthority(), msg.Authority)
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	challenge, found := m.GetChallenge(ctx, msg.ChallengeId)
	if !found {
		return nil, fmt.Errorf("%w: %s", types.ErrChallengeNotFound, msg.ChallengeId)
	}

	if challenge.Status != types.ChallengeStatus_CHALLENGE_STATUS_UNDER_REVIEW {
		return nil, fmt.Errorf("%w: expected UNDER_REVIEW, got %s", types.ErrChallengeNotOpen, challenge.Status.String())
	}

	currentBlock := uint64(ctx.BlockHeight())
	scale := new(big.Int).SetUint64(1_000_000)
	params := m.GetParams(ctx)

	stakeAmt := new(big.Int)
	stakeAmt.SetString(challenge.Stake, 10)

	challengerAddr, err := sdk.AccAddressFromBech32(challenge.Challenger)
	if err != nil {
		return nil, fmt.Errorf("invalid challenger address: %w", err)
	}

	var rewardAmount string
	var slashAmount string

	switch msg.Outcome {
	case types.ChallengeOutcome_CHALLENGE_OUTCOME_UPHELD:
		// Reward challenger from bounty pool
		pool, poolFound := m.GetBountyPool(ctx, challenge.Domain)
		reward := new(big.Int)
		if poolFound {
			poolBal := new(big.Int)
			poolBal.SetString(pool.Balance, 10)
			// reward = pool_balance * reward_rate_bps / 1_000_000
			reward.Mul(poolBal, new(big.Int).SetUint64(params.RewardRateBps))
			reward.Div(reward, scale)
			if reward.Sign() > 0 {
				// Send reward from module to challenger
				rewardCoins := sdk.NewCoins(sdk.NewCoin("uzrn", sdkmath.NewIntFromBigInt(reward)))
				if err := m.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, challengerAddr, rewardCoins); err != nil {
					m.Logger(ctx).Error("failed to send reward", "error", err)
					reward.SetInt64(0)
				} else {
					// Update bounty pool balance
					poolBal.Sub(poolBal, reward)
					pool.Balance = poolBal.String()
					m.SetBountyPool(ctx, pool)
				}
			}
		}
		rewardAmount = reward.String()

		// Track slash amounts for accused validators
		var slashes []*types.ValidatorSlash
		totalSlash := new(big.Int)
		for _, validator := range challenge.AccusedValidators {
			// slash_amount = reported_stake * slash_rate_bps / 1_000_000
			// Here we track the slash record; actual slashing is deferred to staking module
			slashAmt := new(big.Int).Mul(stakeAmt, new(big.Int).SetUint64(params.SlashRateBps))
			slashAmt.Div(slashAmt, scale)
			totalSlash.Add(totalSlash, slashAmt)
			slashes = append(slashes, &types.ValidatorSlash{
				Validator:   validator,
				SlashAmount: slashAmt.String(),
				Reason:      msg.Reason,
			})
		}
		challenge.Slashes = slashes
		slashAmount = totalSlash.String()

		// Return challenger stake
		if stakeAmt.Sign() > 0 {
			stakeCoins := sdk.NewCoins(sdk.NewCoin("uzrn", sdkmath.NewIntFromBigInt(stakeAmt)))
			if err := m.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, challengerAddr, stakeCoins); err != nil {
				return nil, fmt.Errorf("failed to return stake: %w", err)
			}
		}

		// Apply capture consequences (R28-8)
		expiryHeight := uint64(ctx.BlockHeight()) + 50000 // ~35 hours temporary

		// Reduce qualification weight for accused validators
		if m.qualificationKeeper != nil {
			for _, accused := range challenge.AccusedValidators {
				if err := m.qualificationKeeper.ReduceQualificationWeight(ctx, accused, challenge.Domain, 500000, expiryHeight); err != nil {
					m.Logger(ctx).Error("failed to reduce qualification weight", "validator", accused, "err", err)
				}
			}
		}

		// Increase verification threshold for the domain temporarily
		if m.knowledgeKeeper != nil {
			if err := m.knowledgeKeeper.IncreaseVerificationThreshold(ctx, challenge.Domain, 2, expiryHeight); err != nil {
				m.Logger(ctx).Error("failed to increase verification threshold", "domain", challenge.Domain, "err", err)
			}

			// Record role impact for domain elasticity (R29-3)
			// NOTE: CaptureChallenge is domain-level (no FactId field), so challenge.Id
			// won't resolve to a fact. This is a forward-compatible placeholder — role impact
			// from capture challenges will activate when fact-level linking is added.
			if err := m.knowledgeKeeper.RecordChallengeRoleImpact(ctx, challenge.Id, challenge.Domain, true); err != nil {
				m.Logger(ctx).Error("failed to record challenge role impact", "domain", challenge.Domain, "err", err)
			}
		}

		// Cartel detection has consequence: this event announces the
		// uphold verdict that downstream consumers (qualification
		// penalty, threshold raise, stake slash) read. Without the
		// announcement, the consequence flow is invisible. See
		// TRUTH_SEEKING.md commitment 9.
		ctx.EventManager().EmitEvent(
			sdk.NewEvent(
				"zerone.capture_challenge.capture_confirmed",
				sdk.NewAttribute("domain", challenge.Domain),
				sdk.NewAttribute("challenge_id", challenge.Id),
				sdk.NewAttribute("creed_commitment", "9"),
			),
		)

	case types.ChallengeOutcome_CHALLENGE_OUTCOME_REJECTED:
		// Route rejected challenger stake to development fund
		if stakeAmt.Sign() > 0 {
			stakeCoins := sdk.NewCoins(sdk.NewCoin("uzrn", sdkmath.NewIntFromBigInt(stakeAmt)))
			if err := m.bankKeeper.SendCoinsFromModuleToModule(ctx, types.ModuleName, "development_fund", stakeCoins); err != nil {
				return nil, fmt.Errorf("failed to route stake to development fund: %w", err)
			}
		}

		// Add slashed stake to bounty pool
		pool, poolFound := m.GetBountyPool(ctx, challenge.Domain)
		if !poolFound {
			pool = &types.DomainBountyPool{
				Domain:  challenge.Domain,
				Balance: "0",
			}
		}
		poolBal := new(big.Int)
		poolBal.SetString(pool.Balance, 10)
		poolBal.Add(poolBal, stakeAmt)
		pool.Balance = poolBal.String()
		m.SetBountyPool(ctx, pool)

		rewardAmount = "0"
		slashAmount = "0"

		// Clear capture flag on rejected challenge (R28-8)
		if m.captureDefenseKeeper != nil {
			m.captureDefenseKeeper.ClearCaptureFlag(ctx, challenge.Domain)
		}

		// Record role impact — challenge rejected means original verifiers were right (R29-3)
		// NOTE: See UPHELD note — challenge.Id is not a fact ID, so this is a no-op placeholder.
		if m.knowledgeKeeper != nil {
			if err := m.knowledgeKeeper.RecordChallengeRoleImpact(ctx, challenge.Id, challenge.Domain, false); err != nil {
				m.Logger(ctx).Error("failed to record challenge role impact", "domain", challenge.Domain, "err", err)
			}
		}

	case types.ChallengeOutcome_CHALLENGE_OUTCOME_PARTIAL:
		// Return half the stake, route the other half to development fund
		halfStake := new(big.Int).Div(stakeAmt, big.NewInt(2))
		remainder := new(big.Int).Sub(stakeAmt, halfStake)

		if halfStake.Sign() > 0 {
			returnCoins := sdk.NewCoins(sdk.NewCoin("uzrn", sdkmath.NewIntFromBigInt(halfStake)))
			if err := m.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, challengerAddr, returnCoins); err != nil {
				return nil, fmt.Errorf("failed to return partial stake: %w", err)
			}
		}
		if remainder.Sign() > 0 {
			devCoins := sdk.NewCoins(sdk.NewCoin("uzrn", sdkmath.NewIntFromBigInt(remainder)))
			if err := m.bankKeeper.SendCoinsFromModuleToModule(ctx, types.ModuleName, "development_fund", devCoins); err != nil {
				return nil, fmt.Errorf("failed to route partial stake to development fund: %w", err)
			}
		}

		// Partial reward from bounty pool
		pool, poolFound := m.GetBountyPool(ctx, challenge.Domain)
		reward := new(big.Int)
		if poolFound {
			poolBal := new(big.Int)
			poolBal.SetString(pool.Balance, 10)
			// Half of the normal reward rate
			reward.Mul(poolBal, new(big.Int).SetUint64(params.RewardRateBps))
			reward.Div(reward, scale)
			reward.Div(reward, big.NewInt(2))
			if reward.Sign() > 0 {
				rewardCoins := sdk.NewCoins(sdk.NewCoin("uzrn", sdkmath.NewIntFromBigInt(reward)))
				if err := m.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, challengerAddr, rewardCoins); err != nil {
					m.Logger(ctx).Error("failed to send partial reward", "error", err)
					reward.SetInt64(0)
				} else {
					poolBal.Sub(poolBal, reward)
					pool.Balance = poolBal.String()
					m.SetBountyPool(ctx, pool)
				}
			}
		}
		rewardAmount = reward.String()
		slashAmount = "0"

	default:
		return nil, fmt.Errorf("%w: %s", types.ErrInvalidOutcome, msg.Outcome.String())
	}

	// Set resolution record
	challenge.Resolution = &types.CaptureResolution{
		Outcome:       msg.Outcome,
		Resolver:      msg.Authority,
		Reason:        msg.Reason,
		ResolvedBlock: currentBlock,
		RewardAmount:  rewardAmount,
		SlashAmount:   slashAmount,
	}
	challenge.Status = types.ChallengeStatus_CHALLENGE_STATUS_RESOLVED
	m.SetChallenge(ctx, challenge)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"zerone.capture_challenge.challenge_resolved",
			sdk.NewAttribute("challenge_id", msg.ChallengeId),
			sdk.NewAttribute("outcome", msg.Outcome.String()),
			sdk.NewAttribute("reward_amount", rewardAmount),
			sdk.NewAttribute("slash_amount", slashAmount),
		),
	)

	return &types.MsgResolveChallengeResponse{}, nil
}
