package keeper

import (
	"bytes"
	"context"
	"fmt"
	"math/big"
	"strconv"

	sdkmath "cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/protobuf/proto"

	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// Both keys are derived from the plans retained in primary round records.
// The cursor bounds per-block work without permanently starving later rounds.
var pendingVerifierRewardPrefix = []byte{0x82}
var pendingVerifierRewardCursor = []byte{0x83}

const PendingVerifierRewardBatchSize = 16

// One maximum-size review batch, including its possible withholding transfer.
// A second batch waits for the next block rather than multiplying this work.
const PendingVerifierRewardTransferBudget = types.CommitSeatHardCap + 1

func pendingVerifierRewardKey(roundID string) []byte {
	return append(bytes.Clone(pendingVerifierRewardPrefix), []byte(roundID)...)
}

func (k Keeper) getVerificationRoundChecked(ctx context.Context, id string) (*types.VerificationRound, error) {
	bz, err := k.storeService.OpenKVStore(ctx).Get(types.RoundKey(id))
	if err != nil {
		return nil, err
	}
	if bz == nil {
		return nil, fmt.Errorf("verification round %s not found", id)
	}
	if err := types.ValidateRawPolicyField(bz, types.RoundReviewPolicyField); err != nil {
		return nil, err
	}
	var round types.VerificationRound
	if err := proto.Unmarshal(bz, &round); err != nil {
		return nil, fmt.Errorf("decode verification round %s: %w", id, err)
	}
	if round.Id != id {
		return nil, fmt.Errorf("verification round key/id mismatch")
	}
	return &round, nil
}

// SyncPendingVerifierRewardIndex is called within the atomic round setter and
// on explicit genesis reconstruction. The plan, never this index, is authority.
func (k Keeper) SyncPendingVerifierRewardIndex(ctx context.Context, round *types.VerificationRound) error {
	if err := types.ValidateVerifierRewardSettlement(round); err != nil {
		return err
	}
	store := k.storeService.OpenKVStore(ctx)
	key := pendingVerifierRewardKey(round.Id)
	if round.VerifierRewardSettlement != nil && round.VerifierRewardSettlement.PaidAtBlock == 0 {
		return store.Set(key, []byte{1})
	}
	return store.Delete(key)
}

func (k Keeper) buildVerifierRewardSettlement(ctx context.Context, claim *types.Claim, round *types.VerificationRound, result *VerificationResult) error {
	if len(result.Rewards) == 0 {
		return nil
	}
	fee, ok := new(big.Int).SetString(claim.Stake, 10)
	if !ok || fee.Sign() <= 0 || !fee.IsUint64() {
		return fmt.Errorf("review fee cannot be represented by verifier pool arithmetic")
	}
	pool := verifierPoolFromFee(fee.Uint64())
	if pool == 0 {
		return nil
	}
	params, err := k.GetParams(ctx)
	if err != nil {
		return err
	}
	plan := &types.VerifierRewardSettlement{CreatedAtBlock: round.VerdictBlock}
	perVerifier, remainder := pool/uint64(len(result.Rewards)), pool%uint64(len(result.Rewards))
	var withheldTotal uint64
	for i, reward := range result.Rewards {
		amount := perVerifier
		if i == 0 {
			amount += remainder
		}
		modulated := amount
		if round.ReviewPolicyVersion == types.ReviewPolicyLegacy {
			var err error
			modulated, err = k.checkedIndependenceMultiplier(ctx, reward.Verifier, amount, params)
			if err != nil {
				return err
			}
		}
		withheld := amount - modulated
		withheldTotal += withheld
		plan.Payments = append(plan.Payments, &types.VerifierRewardPayment{
			Verifier: reward.Verifier, Amount: strconv.FormatUint(modulated, 10), Withheld: strconv.FormatUint(withheld, 10),
		})
	}
	plan.WithheldTotal = strconv.FormatUint(withheldTotal, 10)
	round.VerifierRewardSettlement = plan
	return types.ValidateVerifierRewardSettlement(round)
}

// Freeze only rewards derived from readable, internally consistent policy
// state. The legacy multiplier keeps its historical behavior before activation.
func (k Keeper) checkedIndependenceMultiplier(ctx context.Context, verifier string, amount uint64, params *types.Params) (uint64, error) {
	if params == nil {
		return 0, fmt.Errorf("missing verifier reward parameters")
	}
	if amount == 0 || params.IndependenceRewardStrengthBps == 0 {
		return amount, nil
	}
	rec, found, err := k.GetValidatorIndependence(ctx, verifier)
	if err != nil {
		return 0, fmt.Errorf("read verifier independence for %s: %w", verifier, err)
	}
	if found && rec.MinorityVotes > rec.TotalVotes {
		return 0, fmt.Errorf("invalid verifier independence counters for %s", verifier)
	}
	if !found || rec.TotalVotes == 0 {
		return amount, nil
	}
	const bps uint64 = 1_000_000
	conformityBps := safeMulDiv(rec.TotalVotes-rec.MinorityVotes, bps, rec.TotalVotes)
	penaltyBps := safeMulDiv(conformityBps, params.IndependenceRewardStrengthBps, bps)
	if penaltyBps >= bps {
		penaltyBps = bps - 1
	}
	return safeMulDiv(amount, bps-penaltyBps, bps), nil
}

// TrySettleVerifierRewards transfers a complete frozen batch and marks it paid
// in one SDK cache. A failure preserves every payment instruction and transfers
// nothing; it does not reverse the independently recorded scientific verdict.
func (k Keeper) TrySettleVerifierRewards(ctx context.Context, roundID string) error {
	enabled, err := k.RecordIntegrityEnabled(ctx)
	if err != nil {
		return err
	}
	if !enabled {
		return fmt.Errorf("verifier settlement requires record integrity activation")
	}
	round, err := k.getVerificationRoundChecked(ctx, roundID)
	if err != nil {
		return err
	}
	if err := types.ValidateVerifierRewardSettlement(round); err != nil {
		return err
	}
	plan := round.VerifierRewardSettlement
	if plan == nil || plan.PaidAtBlock != 0 {
		return nil
	}
	if k.bankKeeper == nil {
		return fmt.Errorf("bank keeper unavailable; verifier rewards remain unpaid")
	}
	ctxSDK := sdk.UnwrapSDKContext(ctx)
	if ctxSDK.BlockHeight() <= 0 || uint64(ctxSDK.BlockHeight()) < plan.CreatedAtBlock {
		return fmt.Errorf("invalid verifier settlement payment height")
	}
	cache, write := ctxSDK.CacheContext()
	for _, payment := range plan.Payments {
		amount, _ := types.SettlementAmount(payment.Amount)
		if amount == 0 {
			continue
		}
		addr, err := sdk.AccAddressFromBech32(payment.Verifier)
		if err != nil {
			return err
		}
		coins := sdk.NewCoins(sdk.NewCoin("uzrn", sdkmath.NewIntFromUint64(amount)))
		if err := k.bankKeeper.SendCoinsFromModuleToAccount(cache, types.ModuleName, addr, coins); err != nil {
			return fmt.Errorf("verifier payment remains unpaid: %w", err)
		}
	}
	withheld, _ := types.SettlementAmount(plan.WithheldTotal)
	if withheld > 0 {
		coins := sdk.NewCoins(sdk.NewCoin("uzrn", sdkmath.NewIntFromUint64(withheld)))
		if err := k.bankKeeper.SendCoinsFromModuleToModule(cache, types.ModuleName, developmentFundModule, coins); err != nil {
			return fmt.Errorf("verifier withholding remains unpaid: %w", err)
		}
	}
	plan.PaidAtBlock = uint64(cache.BlockHeight())
	if err := k.SetVerificationRound(cache, round); err != nil {
		return err
	}
	for _, payment := range plan.Payments {
		cache.EventManager().EmitEvent(sdk.NewEvent("zerone.knowledge.verifier_rewarded",
			sdk.NewAttribute("verifier", payment.Verifier),
			sdk.NewAttribute("round_id", roundID),
			sdk.NewAttribute("amount_uzrn", payment.Amount),
			sdk.NewAttribute("withheld_uzrn", payment.Withheld),
			sdk.NewAttribute("payment_status", "paid"),
			sdk.NewAttribute("paid_at_block", strconv.FormatUint(plan.PaidAtBlock, 10)),
		))
	}
	write()
	return nil
}

// ProcessPendingVerifierRewards examines at most one fixed batch each block.
// Bank errors leave a pending plan and advance the cursor; corrupt index/state
// is returned explicitly instead of being treated as a paid or absent record.
func (k Keeper) ProcessPendingVerifierRewards(ctx context.Context) error {
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
		if round.VerifierRewardSettlement == nil || round.VerifierRewardSettlement.PaidAtBlock != 0 {
			return fmt.Errorf("pending verifier reward index does not match primary round")
		}
		legs := len(round.VerifierRewardSettlement.Payments)
		if round.VerifierRewardSettlement.WithheldTotal != "0" {
			legs++
		}
		if legs > transferBudget {
			// Leave the cursor before this round so it gets the full budget
			// next block, including when earlier rounds remain unfunded.
			exhausted = false
			break
		}
		transferBudget -= legs
		lastProcessed = key
		if err := k.TrySettleVerifierRewards(ctx, id); err != nil {
			k.Logger(ctx).Error("verifier reward remains pending", "round_id", id, "error", err)
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

// RebuildPendingVerifierRewardIndex reconstructs only derived data during an
// explicit genesis import. It never creates a plan for a historical round.
func (k Keeper) RebuildPendingVerifierRewardIndex(ctx context.Context) error {
	cache, write := sdk.UnwrapSDKContext(ctx).CacheContext()
	store := k.storeService.OpenKVStore(cache)
	it, err := store.Iterator(pendingVerifierRewardPrefix, storetypes.PrefixEndBytes(pendingVerifierRewardPrefix))
	if err != nil {
		return err
	}
	var oldKeys [][]byte
	for ; it.Valid(); it.Next() {
		oldKeys = append(oldKeys, bytes.Clone(it.Key()))
	}
	if err := closeSurvivalIterator(it); err != nil {
		return err
	}
	for _, key := range oldKeys {
		if err := store.Delete(key); err != nil {
			return err
		}
	}
	if err := store.Delete(pendingVerifierRewardCursor); err != nil {
		return err
	}
	it, err = store.Iterator(types.VerificationRoundKeyPrefix, storetypes.PrefixEndBytes(types.VerificationRoundKeyPrefix))
	if err != nil {
		return err
	}
	var scanErr error
	for ; it.Valid(); it.Next() {
		if scanErr = types.ValidateRawPolicyField(it.Value(), types.RoundReviewPolicyField); scanErr != nil {
			break
		}
		var round types.VerificationRound
		if scanErr = proto.Unmarshal(it.Value(), &round); scanErr != nil {
			break
		}
		if !bytes.Equal(it.Key(), types.RoundKey(round.Id)) {
			scanErr = fmt.Errorf("verification round key/id mismatch")
			break
		}
		if scanErr = k.SyncPendingVerifierRewardIndex(cache, &round); scanErr != nil {
			break
		}
	}
	closeErr := closeSurvivalIterator(it)
	if scanErr != nil {
		return scanErr
	}
	if closeErr != nil {
		return closeErr
	}
	write()
	return nil
}
