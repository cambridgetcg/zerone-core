package keeper

import (
	"crypto/sha256"
	"fmt"
	"math/big"

	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	"github.com/zerone-chain/zerone/x/vesting_rewards/types"
)

// DistributeRevenue computes the governance-adjustable 4-way allocation for a
// caller-supplied reward. This compatibility helper does not mint or transfer
// value by itself.
//
// The split is driven by RevenueSplit from params (not constants):
//   - ContributorBps: assigned to the named reward recipient
//   - ProtocolBps:    split further via ProtocolSubSplit (citation/verification/treasury)
//   - ResearchBps:    assigned in full to the research fund
//   - DevelopmentBps:    development fund (bug bounties, truth discovery, protocol development)
//
// The retired automatic block-reward path is not a caller in consensus v2.
func (k Keeper) DistributeRevenue(
	ctx sdk.Context,
	source types.RewardSource,
	amount string,
	recipient string,
	factId string,
) (*types.RewardRouting, error) {
	amountBig := new(big.Int)
	if _, ok := amountBig.SetString(amount, 10); !ok {
		return nil, types.ErrInvalidRewardAmount
	}

	if amountBig.Sign() <= 0 {
		return nil, types.ErrInvalidRewardAmount
	}

	split := k.GetRevenueSplit(ctx)
	subSplit := k.GetProtocolSubSplit(ctx)
	bps := big.NewInt(1000000)

	// 4-way split
	contributorAmount := new(big.Int).Mul(amountBig, big.NewInt(int64(split.ContributorBps)))
	contributorAmount.Div(contributorAmount, bps)

	protocolAmount := new(big.Int).Mul(amountBig, big.NewInt(int64(split.ProtocolBps)))
	protocolAmount.Div(protocolAmount, bps)

	researchAmount := new(big.Int).Mul(amountBig, big.NewInt(int64(split.ResearchBps)))
	researchAmount.Div(researchAmount, bps)

	// Development = remainder to avoid rounding leaks
	devAmount := new(big.Int).Set(amountBig)
	devAmount.Sub(devAmount, contributorAmount)
	devAmount.Sub(devAmount, protocolAmount)
	devAmount.Sub(devAmount, researchAmount)
	if devAmount.Sign() < 0 {
		devAmount.SetInt64(0)
	}

	// Protocol sub-split
	citationPool := new(big.Int).Mul(protocolAmount, big.NewInt(int64(subSplit.CitationBps)))
	citationPool.Div(citationPool, bps)

	verificationPool := new(big.Int).Mul(protocolAmount, big.NewInt(int64(subSplit.VerificationBps)))
	verificationPool.Div(verificationPool, bps)

	// Treasury = remainder of protocol share
	treasuryShare := new(big.Int).Set(protocolAmount)
	treasuryShare.Sub(treasuryShare, citationPool)
	treasuryShare.Sub(treasuryShare, verificationPool)
	if treasuryShare.Sign() < 0 {
		treasuryShare.SetInt64(0)
	}

	// Retired wire output. Research is never reduced by an identity-based tap.
	founderShare := new(big.Int)

	routing := &types.RewardRouting{
		Source:            string(source),
		OriginalAmount:    amount,
		ContributorShare:  contributorAmount.String(),
		ProtocolShare:     protocolAmount.String(),
		ResearchShare:     researchAmount.String(),
		DevelopmentAmount: devAmount.String(),
		Recipient:         recipient,
		FactId:            factId,
		BlockNumber:       uint64(ctx.BlockHeight()),
		FounderShare:      founderShare.String(),
		CitationPool:      citationPool.String(),
		VerificationPool:  verificationPool.String(),
		TreasuryShare:     treasuryShare.String(),
	}

	return routing, nil
}

// RouteFees intercepts transaction fees before x/distribution sweeps them to validators.
// Applies the full 4-way revenue split to accumulated fees in fee_collector.
// Must run in BeginBlock BEFORE x/distribution's BeginBlocker.
func (k Keeper) RouteFees(ctx sdk.Context) error {
	if k.bankKeeper == nil {
		return nil
	}

	split := k.GetRevenueSplit(ctx)
	// If all non-contributor shares are zero, nothing to route
	if split.ProtocolBps == 0 && split.ResearchBps == 0 && split.DevelopmentBps == 0 {
		return nil
	}

	feeCollectorBalances := k.bankKeeper.GetAllBalances(ctx, authtypes.NewModuleAddress(authtypes.FeeCollectorName))
	if feeCollectorBalances.IsZero() {
		return nil
	}

	bps := int64(1000000)

	for _, coin := range feeCollectorBalances {
		if coin.Denom != "uzrn" {
			continue
		}

		totalAmount := coin.Amount

		// Research share
		researchTotal := totalAmount.MulRaw(int64(split.ResearchBps)).QuoRaw(bps)
		if researchTotal.IsPositive() {
			researchCoins := sdk.NewCoins(sdk.NewCoin(coin.Denom, researchTotal))
			// Escrow from fee_collector to vesting_rewards module
			if err := k.bankKeeper.SendCoinsFromModuleToModule(ctx, authtypes.FeeCollectorName, types.ModuleName, researchCoins); err != nil {
				k.Logger(ctx).Warn("failed to escrow fee research share", "err", err)
				continue
			}
			// Route through the canonical research-fund depositor.
			if err := k.DepositToResearchFund(ctx, types.ModuleName, researchCoins); err != nil {
				k.Logger(ctx).Warn("failed to deposit fee research share", "err", err)
			}
		}

		// Development fund share
		devTotal := totalAmount.MulRaw(int64(split.DevelopmentBps)).QuoRaw(bps)
		if devTotal.IsPositive() {
			devCoins := sdk.NewCoins(sdk.NewCoin(coin.Denom, devTotal))
			if err := k.bankKeeper.SendCoinsFromModuleToModule(ctx, authtypes.FeeCollectorName, types.DevelopmentFundModuleName, devCoins); err != nil {
				k.Logger(ctx).Warn("failed to route fee development share", "err", err)
				continue
			}
		}

		// Protocol share stays in fee_collector for x/distribution to sweep to validators.
		// Contributor share is irrelevant for fees (fees come from tx senders, not contributors).
	}

	return nil
}

// DepositToResearchFund routes a deposit in full to the research fund.
// sourceModule must hold the funds in its module account before calling this
// method. The zero-valued founder event attribute is retained for indexer
// compatibility; consensus version 2 has no founder payout path.
func (k Keeper) DepositToResearchFund(ctx sdk.Context, sourceModule string, amount sdk.Coins) error {
	if amount.IsZero() {
		return nil
	}

	// Escrow to vesting_rewards if source is a different module.
	if sourceModule != types.ModuleName {
		if err := k.bankKeeper.SendCoinsFromModuleToModule(ctx, sourceModule, types.ModuleName, amount); err != nil {
			return fmt.Errorf("research fund escrow to vesting_rewards failed: %w", err)
		}
	}

	for _, coin := range amount {
		if coin.Amount.IsZero() {
			continue
		}

		founderAmount := sdkmath.ZeroInt()
		researchAmount := coin.Amount

		// Send research portion to research_fund
		if researchAmount.IsPositive() {
			researchCoins := sdk.NewCoins(sdk.NewCoin(coin.Denom, researchAmount))
			if err := k.bankKeeper.SendCoinsFromModuleToModule(ctx, types.ModuleName, types.ResearchFundModuleName, researchCoins); err != nil {
				return fmt.Errorf("research fund deposit failed: %w", err)
			}
		}

		ctx.EventManager().EmitEvent(sdk.NewEvent(
			"zerone.vesting_rewards.research_fund_deposit",
			sdk.NewAttribute("source_module", sourceModule),
			sdk.NewAttribute("denom", coin.Denom),
			sdk.NewAttribute("total", coin.Amount.String()),
			sdk.NewAttribute("research", researchAmount.String()),
			sdk.NewAttribute("founder", founderAmount.String()),
		))
	}

	return nil
}

// DisburseFromResearchFund sends coins from the research fund module account to a recipient.
func (k Keeper) DisburseFromResearchFund(ctx sdk.Context, recipient sdk.AccAddress, amount sdk.Coins) error {
	if k.bankKeeper == nil {
		return fmt.Errorf("bank keeper not available")
	}
	return k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ResearchFundModuleName, recipient, amount)
}

// DisburseFromDevelopmentFund sends coins from the development fund module account to a recipient.
// Called by governance proposals for bug bounties and development grants.
func (k Keeper) DisburseFromDevelopmentFund(ctx sdk.Context, recipient sdk.AccAddress, amount sdk.Coins) error {
	if k.bankKeeper == nil {
		return fmt.Errorf("bank keeper not available")
	}
	return k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.DevelopmentFundModuleName, recipient, amount)
}

// GetEpochBlockRewardPool preserves the legacy integration boundary and always
// reports zero. Consensus v2 has no automatic per-block or per-epoch pool,
// even if an internal caller bypasses parameter validation.
func (k Keeper) GetEpochBlockRewardPool(_ sdk.Context, _ uint64) uint64 { return 0 }

// DistributeBlockReward preserves the pre-v2 Go method shape while making the
// retired path structurally inert. It never mints, transfers, writes a reward
// record, or emits an event, even if a caller bypasses parameter validation and
// injects legacy non-zero values. Historical records remain queryable from
// state; new issuance must begin with independently witnessed successful work.
func (k Keeper) DistributeBlockReward(
	ctx sdk.Context,
	_ string,
	activeValidatorCount uint32,
	_ bool,
) (*types.BlockRewardDistribution, error) {
	return &types.BlockRewardDistribution{
		BlockHeight:       uint64(ctx.BlockHeight()),
		ProducerReward:    "0",
		ResearchShare:     "0",
		TotalMinted:       "0",
		ValidatorCount:    activeValidatorCount,
		FundBalanceAfter:  k.GetTotalMinted(ctx).String(),
		FounderShare:      "0",
		DevelopmentAmount: "0",
		ProtocolShare:     "0",
	}, nil
}

// FalsifyClaim handles clawback when a claim is proven false.
//
// Clawback logic:
//   - 33% of already-released rewards are clawed back
//   - All unvested amount is forfeited
//   - Reserve goes to challenger as bonus
//
// ADJUDICATION GATE. MsgFalsifyVesting's proto signer is `challenger`, not
// `authority` (proto/zerone/vesting_rewards/v1/tx.proto), so any address can
// reach this function by naming itself. Without the gate below, one tx fee
// permanently destroyed any honest submitter's payout stream: the handler
// checked only that the schedule existed, and never consulted the fact the
// schedule was paying for. Falsification is a verdict the PoT layer reaches,
// never a claim the caller makes — so the linked fact must already be
// FACT_STATUS_DISPROVEN.
//
// The gate FAILS CLOSED. If the knowledge keeper is not wired we refuse the
// clawback rather than allow it, because the failure mode of allowing it is
// irreversible destruction of someone else's rewards.
func (k Keeper) FalsifyClaim(
	ctx sdk.Context,
	claimId string,
	challenger string,
) (*types.ClawbackRecord, error) {
	schedule, found := k.GetVestingByClaimId(ctx, claimId)
	if !found {
		return nil, types.ErrScheduleNotFound
	}

	if schedule.Status == string(types.VestingStatusFalsified) {
		return nil, types.ErrAlreadyFalsified
	}

	if k.knowledgeKeeper == nil {
		return nil, types.ErrAdjudicationUnavailable
	}
	if !k.knowledgeKeeper.IsFactDisproven(ctx, schedule.FactId) {
		return nil, types.ErrFactNotDisproven.Wrapf(
			"fact %q linked to claim %q is not FACT_STATUS_DISPROVEN", schedule.FactId, claimId)
	}

	params := k.GetParams(ctx)
	height := uint64(ctx.BlockHeight())

	releasedBig := new(big.Int)
	releasedBig.SetString(schedule.ReleasedAmount, 10)

	// Released clawback: 33% of already released
	releasedClawback := new(big.Int).Mul(releasedBig, big.NewInt(int64(params.ReleasedClawbackRate)))
	releasedClawback.Div(releasedClawback, big.NewInt(10000))

	// Unvested = total - released - reserve
	totalBig := new(big.Int)
	totalBig.SetString(schedule.TotalAmount, 10)
	reserveBig := new(big.Int)
	reserveBig.SetString(schedule.ReserveAmount, 10)

	unvested := new(big.Int).Sub(totalBig, releasedBig)
	unvested.Sub(unvested, reserveBig)
	if unvested.Sign() < 0 {
		unvested = big.NewInt(0)
	}

	// Challenger reward = released clawback + unvested + reserve
	challengerReward := new(big.Int).Add(releasedClawback, unvested)
	challengerReward.Add(challengerReward, reserveBig)

	idInput := fmt.Sprintf("clawback:%s:%d", claimId, height)
	hash := sha256.Sum256([]byte(idInput))
	recordId := fmt.Sprintf("%x", hash[:16])

	record := &types.ClawbackRecord{
		Id:                recordId,
		VestingId:         schedule.Id,
		ReleasedClawback:  releasedClawback.String(),
		UnvestedForfeited: unvested.String(),
		ReserveForfeited:  reserveBig.String(),
		ChallengerReward:  challengerReward.String(),
		BlockHeight:       height,
	}

	schedule.Status = string(types.VestingStatusFalsified)
	schedule.UpdatedAt = height
	k.SetVestingSchedule(ctx, schedule)

	k.SetClawbackRecord(ctx, record)

	k.Logger(ctx).Info("falsified claim, clawback executed",
		"claim_id", claimId,
		"vesting_id", schedule.Id,
		"challenger", challenger,
		"challenger_reward", challengerReward.String(),
	)

	return record, nil
}
