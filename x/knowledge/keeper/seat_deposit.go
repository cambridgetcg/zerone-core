package keeper

import (
	"context"
	"fmt"
	"strconv"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// Reviewer seat deposit: step B of the reviewer-entry design draft
// (docs/specs/knowledge-reviewer-entry-v1.md, pull request #190), rules D1 to
// D7. Everything here is selected by the seat-deposit marker
// (seat_deposit_activation.go), read once per entry point without gas. With
// the marker off no round records seat terms, so no code below runs past its
// first check and every message, record, event and gas figure is what
// 3a844c48 produces.

// setRoundSeatTerms records a new round's seat terms (D1): while the marker is
// on, a round under review policy 1 with commitment scheme 2 copies the
// counted-seat deposit from min_reviewer_balance as it stands at creation. The
// figure is ParseMinReviewerBalance's, not EffectiveMinReviewerBalance's: a
// figure below 100 ZRN is what a locked deposit allows. An empty parameter
// locks the built-in 100 ZRN. A later parameter change does not alter the
// round: ValidateSeatDepositUpdate keeps terms immutable. Policy-0 rounds,
// and every round created while the marker is off, get no terms and keep the
// balance check (A5, D7).
//
// params is the set CreateVerificationRound already loaded, so this adds no
// param read. GetParams returns DefaultParams when the stored set fails to
// decode, so such a chain locks 100 ZRN, not the governed figure (the same
// caveat as reviewerMinBalance).
func (k Keeper) setRoundSeatTerms(ctx context.Context, round *types.VerificationRound, params *types.Params) error {
	enabled, err := k.SeatDepositEnabled(ctx)
	if err != nil {
		return err
	}
	if !enabled || !types.RoundSeatTermsAllowed(round) {
		return nil
	}
	// The deposit is the governable figure, so its marker must be on too;
	// an inconsistent activation fails closed instead of locking a figure
	// that governance cannot set.
	paramMarker, err := k.ReviewerBalanceParamEnabled(ctx)
	if err != nil {
		return err
	}
	if !paramMarker {
		return fmt.Errorf("seat deposits require the reviewer-balance-param activation")
	}
	if params == nil {
		return fmt.Errorf("seat terms require the knowledge params")
	}
	amount, err := types.ParseMinReviewerBalance(params.MinReviewerBalance)
	if err != nil {
		return fmt.Errorf("stored params: %w", err)
	}
	round.SeatTerms = &types.ReviewSeatTerms{CountedSeatDeposit: types.FormatSeatDeposit(amount)}
	return nil
}

// submitDepositedCommitment seats a reviewer in a round with seat terms (D2,
// D3). SubmitCommitment has already checked the round's phase, deadline and
// seat cap, and read the marker. In such a round:
//
//   - the handler checks that the signer is registered with zerone_auth, may
//     review and is not frozen, as the ante does, since simulation and the
//     interchain-accounts host skip the ante (D3). The signer is msg.Verifier,
//     so the seat is the signer's own;
//   - the deposit moves from the signer to SeatDepositModuleName after every
//     other check passes, and is recorded with the seat. The transfer needs
//     spendable coins, where the balance check it replaces read the whole
//     balance. If it fails, the commitment is refused and nothing is stored
//     (D2);
//   - no balance check and no param read: the locked deposit replaces them.
func (m *msgServer) submitDepositedCommitment(ctx context.Context, msg *types.MsgSubmitCommitment, height uint64) (*types.MsgSubmitCommitmentResponse, error) {
	k := m.keeper
	if k.bankKeeper == nil {
		return nil, types.ErrSeatDepositRequired.Wrap("the bank keeper is unavailable to lock a seat deposit")
	}
	if k.zeroneAuthKeeper == nil {
		return nil, types.ErrReviewerNotAdmitted.Wrap("zerone_auth is unavailable to admit the reviewer")
	}
	if err := k.zeroneAuthKeeper.CheckReviewer(ctx, msg.Verifier); err != nil {
		return nil, err
	}
	var locked string
	lock := func(cache context.Context, round *types.VerificationRound) (string, error) {
		amount := round.SeatTerms.CountedSeatDeposit
		if err := k.lockSeatDeposit(cache, msg.Verifier, amount); err != nil {
			return "", err
		}
		locked = amount
		return amount, nil
	}
	if err := k.storeCommitmentInRound(ctx, msg.RoundId, &types.CommitEntry{
		Verifier:         msg.Verifier,
		CommitHash:       msg.CommitHash,
		CommittedAtBlock: height,
	}, lock); err != nil {
		return nil, err
	}
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent("zerone.knowledge.submit_commitment",
			sdk.NewAttribute("round_id", msg.RoundId),
			sdk.NewAttribute("verifier", msg.Verifier),
			sdk.NewAttribute("committed_at_block", fmt.Sprintf("%d", height)),
		),
	)
	sdkCtx.EventManager().EmitEvent(sdk.NewEvent("zerone.knowledge.seat_deposit_locked",
		sdk.NewAttribute("round_id", msg.RoundId),
		sdk.NewAttribute("depositor", msg.Verifier),
		sdk.NewAttribute("amount_uzrn", locked),
		sdk.NewAttribute("committed_at_block", strconv.FormatUint(height, 10)),
	))
	return &types.MsgSubmitCommitmentResponse{}, nil
}

// lockSeatDeposit moves amount uzrn from depositor into the seat-deposit
// account. The account is created by the bank on the first deposit.
func (k Keeper) lockSeatDeposit(ctx context.Context, depositor, amount string) error {
	value, err := types.SeatDepositAmount(amount)
	if err != nil {
		return err
	}
	if err := k.requireSeatDepositAccount(ctx); err != nil {
		return err
	}
	addr, err := sdk.AccAddressFromBech32(depositor)
	if err != nil {
		return fmt.Errorf("invalid depositor address: %w", err)
	}
	coins := sdk.NewCoins(sdk.NewCoin("uzrn", sdkmath.NewIntFromUint64(value)))
	if err := k.bankKeeper.SendCoinsFromAccountToModule(ctx, addr, types.SeatDepositModuleName, coins); err != nil {
		return types.ErrSeatDepositRequired.Wrapf("the seat deposit of %s uzrn could not be locked: %v", amount, err)
	}
	return nil
}

// requireSeatDepositAccount refuses, with an error, an address that holds an
// account other than the seat-deposit module account. Before this binary the
// address was an ordinary one, so a send could have created a plain account
// there; the bank would then panic on every deposit. Returns are paid by
// address and are unaffected. A future carrier must check or convert this
// address on an existing chain; this draft only refuses.
func (k Keeper) requireSeatDepositAccount(ctx context.Context) error {
	if k.accountKeeper == nil {
		return fmt.Errorf("seat deposits require the account keeper")
	}
	account := k.accountKeeper.GetAccount(ctx, authtypes.NewModuleAddress(types.SeatDepositModuleName))
	if account == nil {
		return nil
	}
	module, ok := account.(sdk.ModuleAccountI)
	if !ok || module.GetName() != types.SeatDepositModuleName {
		return fmt.Errorf("the %s address holds an account that is not its module account", types.SeatDepositModuleName)
	}
	return nil
}

// buildClaimRefundSettlement shadows the Keeper method for completeRound,
// whose body is pinned to its foundation (TestQ2LegacyBodiesPinnedToFoundation)
// and calls it on roundsExecution once per completion, inside the
// completion's cache, after the verdict, phase and verdict block are set and
// beside the reviewer payment plan. It builds the refund exactly as before,
// then freezes the round's seat-deposit returns (D4, D5). A round without
// seat terms reads nothing more. The accrued event joins the completion's
// cache, so it is emitted only if the completion is written.
func (k roundsExecution) buildClaimRefundSettlement(ctx context.Context, claim *types.Claim, round *types.VerificationRound, result *VerificationResult) error {
	if err := k.Keeper.buildClaimRefundSettlement(ctx, claim, round, result); err != nil {
		return err
	}
	frozen, err := k.freezeSeatDepositReturns(round, round.VerdictBlock)
	if err != nil {
		return err
	}
	if frozen {
		emitSeatDepositReturnsAccrued(sdk.UnwrapSDKContext(ctx), round, "completion")
	}
	return nil
}

// freezeSeatDepositReturns freezes a round's deposit returns (D4, D5): one
// return per seat, in commit order, to the address that locked it, for the
// amount it locked, whatever the verdict, the seat's vote or whether it
// revealed. Nothing is forfeited. It reports whether it froze a plan. A round
// without seat terms reads nothing; a plan already frozen (at the aggregation
// deadline) is kept; a round with no seat freezes none.
func (k roundsExecution) freezeSeatDepositReturns(round *types.VerificationRound, height uint64) (bool, error) {
	if round.SeatTerms == nil {
		return false, nil
	}
	if !k.seatDeposit {
		return false, fmt.Errorf("round %s has seat terms but the seat-deposit activation is off", round.Id)
	}
	if round.SeatDepositReturns != nil || len(round.Commits) == 0 {
		return false, nil
	}
	round.SeatDepositReturns = types.BuildSeatDepositReturns(round, height)
	return true, nil
}

// freezeOverdueSeatDepositReturns is the aggregation-deadline fallback (D4):
// a policy-1 round whose completion keeps failing stays open and is retried,
// so from its aggregation deadline its seats' returns are frozen anyway and
// the pending queue pays them. Completion later keeps that plan. Each round
// is frozen in its own cache; a failure is logged and leaves that round to
// the next block, and never fails the block.
//
// The limit: a round whose own record, or whose claim record, cannot be read
// or written holds its deposits until that is repaired, since the freeze
// writes the round through SetVerificationRound like completion does.
func (k roundsExecution) freezeOverdueSeatDepositReturns(ctx context.Context) error {
	if !k.seatDeposit {
		return nil
	}
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	height := uint64(sdkCtx.BlockHeight())
	var overdue []*types.VerificationRound
	k.IterateActiveRounds(ctx, func(round *types.VerificationRound) bool {
		if round.SeatTerms != nil && round.SeatDepositReturns == nil && len(round.Commits) > 0 && height >= round.AggregationDeadline {
			overdue = append(overdue, round)
		}
		return false
	})
	for _, round := range overdue {
		cache, write := sdkCtx.CacheContext()
		frozen, err := k.freezeSeatDepositReturns(round, height)
		if err == nil && frozen {
			err = k.SetVerificationRound(cache, round)
		}
		if err != nil {
			k.Logger(ctx).Error("seat deposit returns remain unfrozen", "round_id", round.Id, "error", err)
			continue
		}
		if frozen {
			emitSeatDepositReturnsAccrued(cache, round, "aggregation_deadline")
			write()
		}
	}
	return nil
}

// emitSeatDepositReturnsAccrued records a frozen returns plan. The returns are
// an obligation here; seat_deposit_returned records each transfer.
func emitSeatDepositReturnsAccrued(ctx sdk.Context, round *types.VerificationRound, trigger string) {
	// The plan was validated before it was stored, so the total parses; a
	// total that does not is reported as such, never as a number.
	totalText := "unreadable"
	if total, err := types.SeatDepositReturnTotal(round.SeatDepositReturns); err == nil {
		totalText = total.String()
	}
	ctx.EventManager().EmitEvent(sdk.NewEvent("zerone.knowledge.seat_deposit_returns_accrued",
		sdk.NewAttribute("round_id", round.Id),
		sdk.NewAttribute("seats", strconv.Itoa(len(round.SeatDepositReturns.GetReturns()))),
		sdk.NewAttribute("total_uzrn", totalText),
		sdk.NewAttribute("trigger", trigger),
		sdk.NewAttribute("payment_status", "pending"),
	))
}
