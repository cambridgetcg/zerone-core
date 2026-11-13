package keeper

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strconv"

	sdkmath "cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
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

// requireSeatDepositAccount makes the seat-deposit address one the bank can
// pay into before a deposit moves (D2). Before this binary the address was an
// ordinary one, which anyone could create with a send of one uzrn; the bank
// panics on a deposit to a module address that holds a plain account, so
// leaving it would deny every deposited seat on the chain. A plain account
// that no key has signed for (no public key, sequence 0; no key exists for a
// module address) is converted here into the seat-deposit module account,
// keeping its account number, once per chain at most. Whatever it held stays
// in the account as a surplus, reported and never swept (D6). Any other
// account there is refused with an error, and nothing moves. Returns are paid
// by address and need no account check.
func (k Keeper) requireSeatDepositAccount(ctx context.Context) error {
	squatter, err := k.seatDepositSquatter(ctx)
	if err != nil || squatter == nil {
		return err
	}
	address := squatter.GetAddress()
	module := authtypes.NewModuleAccount(authtypes.NewBaseAccount(address, nil, squatter.GetAccountNumber(), 0), types.SeatDepositModuleName)
	k.accountKeeper.SetModuleAccount(ctx, module)
	if k.bankKeeper != nil {
		k.Logger(ctx).Info("seat-deposit address converted into its module account; any balance it held is surplus, reported and not swept",
			"account_number", squatter.GetAccountNumber(), "held_uzrn", k.bankKeeper.GetBalance(ctx, address, "uzrn").Amount.String())
	}
	return nil
}

// seatDepositSquatter reads the account at the seat-deposit address. It
// returns nil when the address holds nothing or its own module account, the
// plain account when one that requireSeatDepositAccount can convert is there,
// and an error for any other account. It writes nothing.
func (k Keeper) seatDepositSquatter(ctx context.Context) (*authtypes.BaseAccount, error) {
	if k.accountKeeper == nil {
		return nil, fmt.Errorf("seat deposits require the account keeper")
	}
	address := authtypes.NewModuleAddress(types.SeatDepositModuleName)
	account := k.accountKeeper.GetAccount(ctx, address)
	if account == nil {
		return nil, nil
	}
	if module, ok := account.(sdk.ModuleAccountI); ok && module.GetName() == types.SeatDepositModuleName {
		return nil, nil
	}
	// Only the exact plain type: a vesting or other account embeds a base
	// account but carries terms a conversion would drop.
	plain, ok := account.(*authtypes.BaseAccount)
	if !ok || !address.Equals(plain.GetAddress()) || plain.GetPubKey() != nil || plain.GetSequence() != 0 {
		return nil, fmt.Errorf("the %s address holds an account that cannot become its module account", types.SeatDepositModuleName)
	}
	return plain, nil
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

// processPendingRoundObligations is the pending queue with the seat-deposit
// marker on (D5). It is processPendingVerifierRewardsLegacy with three
// changes, and nothing else:
//
//   - a round's pending obligations include its unpaid seat-deposit returns,
//     both in the index consistency check and in the transfer count, and
//     the returns are validated with the round's other plans;
//   - a round is settled by settleRoundObligations: the reviewer payment and
//     refund leg exactly as before, then each return on its own;
//   - a round whose pending transfers exceed the whole per-block budget is
//     paid alone, in a block of its own, when it comes first. The budget
//     (PendingVerifierRewardTransferBudget, 513) covers the largest reviewer
//     plan with its refund or withholding, but a round with seat terms can
//     also owe up to 512 returns; the legacy body would leave the cursor on
//     such a round for ever and stall every round after it.
//
// Per block it does at most one batch of 16 rounds within 513 transfers, or
// one round alone, at most 1,025 transfers. No round is skipped for ever.
func (k roundsExecution) processPendingRoundObligations(ctx context.Context) error {
	enabled, err := k.RecordIntegrityEnabled(ctx)
	if err != nil || !enabled {
		return err
	}
	store := k.storeService.OpenKVStore(ctx)
	cursor, err := store.Get(pendingVerifierRewardCursor)
	if err != nil {
		return err
	}
	start := bytes.Clone(pendingVerifierRewardPrefix)
	if len(cursor) > 0 {
		if !bytes.HasPrefix(cursor, pendingVerifierRewardPrefix) || len(cursor) == len(pendingVerifierRewardPrefix) {
			return fmt.Errorf("invalid pending verifier reward cursor")
		}
		start = append(bytes.Clone(cursor), 0)
	}
	it, err := store.Iterator(start, storetypes.PrefixEndBytes(pendingVerifierRewardPrefix))
	if err != nil {
		return err
	}
	var keys [][]byte
	for ; it.Valid() && len(keys) < PendingVerifierRewardBatchSize; it.Next() {
		if !bytes.Equal(it.Value(), []byte{1}) || len(it.Key()) <= len(pendingVerifierRewardPrefix) {
			_ = it.Close()
			return fmt.Errorf("invalid pending verifier reward index")
		}
		keys = append(keys, bytes.Clone(it.Key()))
	}
	exhausted := !it.Valid()
	if err := closeSurvivalIterator(it); err != nil {
		return err
	}
	transferBudget := PendingVerifierRewardTransferBudget
	var lastProcessed []byte
	for _, key := range keys {
		id := string(key[len(pendingVerifierRewardPrefix):])
		round, err := k.getVerificationRoundChecked(ctx, id)
		if err != nil {
			return err
		}
		if err := types.ValidateVerifierRewardSettlement(round); err != nil {
			return err
		}
		if err := types.ValidateClaimRefundSettlement(round); err != nil {
			return err
		}
		if err := types.ValidateRoundSeatDeposits(round); err != nil {
			return err
		}
		if !roundHasPendingObligations(round) {
			return fmt.Errorf("pending verifier reward index does not match primary round")
		}
		legs := pendingObligationTransferCount(round)
		if legs > transferBudget {
			exhausted = false
			if lastProcessed != nil {
				// Leave the cursor before this round so it gets the full
				// budget next block, including when earlier rounds remain
				// unfunded.
				break
			}
			// First in this block and larger than the whole budget: pay it
			// alone, so it never holds the queue (D5).
			lastProcessed = key
			if err := k.settleRoundObligations(ctx, id); err != nil {
				k.Logger(ctx).Error("round obligations remain pending", "round_id", id, "error", err)
			}
			break
		}
		transferBudget -= legs
		lastProcessed = key
		if err := k.settleRoundObligations(ctx, id); err != nil {
			k.Logger(ctx).Error("round obligations remain pending", "round_id", id, "error", err)
		}
	}
	if exhausted {
		return store.Delete(pendingVerifierRewardCursor)
	}
	if lastProcessed != nil {
		return store.Set(pendingVerifierRewardCursor, lastProcessed)
	}
	return nil
}

// settleRoundObligations settles a round's two independent legs (D4): the
// reviewer payment and claim refund, through the unchanged
// TrySettleVerifierRewards (all or nothing, as before; it does nothing when
// only returns are pending), and then the seat-deposit returns, each on its
// own. A failure in one leg never blocks the other; both errors are returned.
func (k roundsExecution) settleRoundObligations(ctx context.Context, roundID string) error {
	fundsErr := k.TrySettleVerifierRewards(ctx, roundID)
	returnsErr := k.settleSeatDepositReturns(ctx, roundID)
	return errors.Join(fundsErr, returnsErr)
}

// settleSeatDepositReturns pays a round's unpaid seat-deposit returns from the
// seat-deposit account only (D4, D5). Each return is its own transfer in its
// own cache: a transfer that succeeds is recorded paid at this height and
// emits seat_deposit_returned; one that fails is discarded, logged and left
// pending, and repeats nothing already paid. The round record is written once
// with every return paid in this call, and no transfer survives without its
// paid record.
func (k Keeper) settleSeatDepositReturns(ctx context.Context, roundID string) error {
	round, err := k.getVerificationRoundChecked(ctx, roundID)
	if err != nil {
		return err
	}
	if err := types.ValidateRoundSeatDeposits(round); err != nil {
		return err
	}
	plan := round.SeatDepositReturns
	if types.PendingSeatDepositReturns(round) == 0 {
		return nil
	}
	if k.bankKeeper == nil {
		return fmt.Errorf("bank keeper unavailable; seat deposits remain unreturned")
	}
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	if sdkCtx.BlockHeight() <= 0 || uint64(sdkCtx.BlockHeight()) < plan.CreatedAtBlock {
		return fmt.Errorf("invalid seat deposit return height")
	}
	height := uint64(sdkCtx.BlockHeight())
	cache, write := sdkCtx.CacheContext()
	var failures []error
	var paid []*types.SeatDepositReturn
	for _, ret := range plan.Returns {
		if ret.PaidAtBlock != 0 {
			continue
		}
		amount, err := types.SeatDepositAmount(ret.Amount)
		if err != nil {
			return err
		}
		recipient, err := sdk.AccAddressFromBech32(ret.Recipient)
		if err != nil {
			return err
		}
		transfer, keep := cache.CacheContext()
		coins := sdk.NewCoins(sdk.NewCoin("uzrn", sdkmath.NewIntFromUint64(amount)))
		if err := k.bankKeeper.SendCoinsFromModuleToAccount(transfer, types.SeatDepositModuleName, recipient, coins); err != nil {
			failures = append(failures, fmt.Errorf("seat deposit return to %s remains unpaid: %w", ret.Recipient, err))
			continue
		}
		keep()
		ret.PaidAtBlock = height
		paid = append(paid, ret)
	}
	if len(paid) == 0 {
		return errors.Join(failures...)
	}
	if err := k.SetVerificationRound(cache, round); err != nil {
		return err
	}
	for _, ret := range paid {
		cache.EventManager().EmitEvent(sdk.NewEvent("zerone.knowledge.seat_deposit_returned",
			sdk.NewAttribute("round_id", roundID),
			sdk.NewAttribute("recipient", ret.Recipient),
			sdk.NewAttribute("amount_uzrn", ret.Amount),
			sdk.NewAttribute("payment_status", "paid"),
			sdk.NewAttribute("paid_at_block", strconv.FormatUint(ret.PaidAtBlock, 10)),
		))
	}
	write()
	return errors.Join(failures...)
}

// SeatDepositCustody returns what the seat-deposit account owes, the sum of
// locked and unreturned deposits over every stored round, and what it holds,
// its uzrn bank balance (D6).
func (k Keeper) SeatDepositCustody(ctx context.Context) (owed, held sdkmath.Int, err error) {
	if k.bankKeeper == nil {
		return sdkmath.Int{}, sdkmath.Int{}, fmt.Errorf("seat-deposit custody requires the bank keeper")
	}
	rounds, err := k.GetAllVerificationRoundsChecked(ctx)
	if err != nil {
		return sdkmath.Int{}, sdkmath.Int{}, err
	}
	owed = sdkmath.ZeroInt()
	for _, round := range rounds {
		amount, err := types.SeatDepositsHeld(round)
		if err != nil {
			return sdkmath.Int{}, sdkmath.Int{}, err
		}
		owed = owed.Add(amount)
	}
	held = k.bankKeeper.GetBalance(ctx, authtypes.NewModuleAddress(types.SeatDepositModuleName), "uzrn").Amount
	return owed, held, nil
}

// ValidateSeatDepositCustody refuses state where the seat-deposit account
// holds less than the locked, unreturned deposits it owes (D6). A surplus is
// allowed: it is reported, never swept. Tests use it; a future carrier that
// activates seat deposits in place must run it. It is not a condition that
// halts a chain: at run time a shortfall only makes a return fail, which is
// logged and retried.
func (k Keeper) ValidateSeatDepositCustody(ctx context.Context) error {
	owed, held, err := k.SeatDepositCustody(ctx)
	if err != nil {
		return err
	}
	if held.LT(owed) {
		return fmt.Errorf("seat-deposit account holds %s uzrn, less than the %s uzrn of locked, unreturned deposits", held, owed)
	}
	return nil
}

// validateSeatDepositGenesisCustody is D6 at import. With the marker off
// nothing is read (ValidateSeatDepositGenesis has already refused any seat
// field). With it on, before either import path writes: an account at the
// seat-deposit address must be its module account or a plain account the
// first deposit can convert (requireSeatDepositAccount; import itself writes
// no account), and the imported bank balance must cover every locked,
// unreturned deposit the file carries. A shortfall refuses the import. A
// surplus, such as a coin sent before this binary blocked the address, is
// logged and not swept; InitChain discards events, so the node log is the
// import's record.
func (k Keeper) validateSeatDepositGenesisCustody(ctx context.Context, gs *types.GenesisState) error {
	if gs == nil || !gs.SeatDepositEnabled {
		return nil
	}
	if k.accountKeeper != nil {
		if _, err := k.seatDepositSquatter(ctx); err != nil {
			return err
		}
	}
	owed, err := types.SeatDepositsHeldInGenesis(gs)
	if err != nil {
		return err
	}
	if k.bankKeeper == nil {
		if owed.IsPositive() {
			return fmt.Errorf("seat deposits in genesis require the bank keeper")
		}
		return nil
	}
	held := k.bankKeeper.GetBalance(ctx, authtypes.NewModuleAddress(types.SeatDepositModuleName), "uzrn").Amount
	if held.LT(owed) {
		return fmt.Errorf("genesis seat-deposit account holds %s uzrn, less than the %s uzrn of locked, unreturned deposits", held, owed)
	}
	if held.GT(owed) {
		k.Logger(ctx).Info("seat-deposit account surplus at import, reported and not swept",
			"held_uzrn", held.String(), "owed_uzrn", owed.String(), "surplus_uzrn", held.Sub(owed).String())
	}
	return nil
}
