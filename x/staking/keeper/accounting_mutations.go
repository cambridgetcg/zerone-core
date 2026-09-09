package keeper

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/big"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/x/staking/types"
)

// Cached writes protect direct keeper callers as well as the SDK message router.
// Resource admission failures also roll back any preceding bank transfer.
func accountingMutation[T any](ctx sdk.Context, fn func(sdk.Context) (*T, error)) (out *T, err error) {
	cache, write := ctx.CacheContext()
	defer func() {
		if recovered := recover(); recovered != nil {
			if e, ok := recovered.(error); ok && errors.Is(e, types.ErrAccountingSafety) {
				out, err = nil, e
				return
			}
			panic(recovered)
		}
	}()
	out, err = fn(cache)
	if err == nil {
		write()
	}
	return out, err
}

func accountingCoins(amount *big.Int) sdk.Coins {
	return sdk.NewCoins(sdk.NewCoin("uzrn", sdkmath.NewIntFromBigInt(amount)))
}

func (k Keeper) changeAccountingDelegation(ctx sdk.Context, v *types.Validator, delegator string, delta *big.Int) error {
	if err := canonicalAccountingAddress(delegator); err != nil {
		return err
	}
	d, err := k.checkedAccountingDelegation(ctx, delegator, v.OperatorAddress)
	if err != nil && !errors.Is(err, types.ErrDelegationNotFound) {
		return err
	}
	if d == nil {
		if delta.Sign() <= 0 {
			return types.ErrDelegationNotFound
		}
		// An orphaned reverse entry is not an absent delegation.
		if k.getStore(ctx).Has(types.ValidatorDelegationIndexKey(v.OperatorAddress, delegator)) {
			return accountingError("orphaned reverse index")
		}
		d = &types.Delegation{DelegatorAddress: delegator, ValidatorAddress: v.OperatorAddress, Amount: "0", CreatedAtBlock: uint64(ctx.BlockHeight())}
	}
	old, _ := accountingAmount(d.Amount, false)
	remaining := new(big.Int).Add(old, delta)
	if remaining.Sign() < 0 {
		return types.ErrInsufficientDelegation
	}
	if remaining.BitLen() > 256 {
		return accountingError("claim amount overflow")
	}
	self, _ := accountingAmount(v.SelfDelegation, false)
	delegated, _ := accountingAmount(v.DelegatedStake, false)
	if delegator == v.OperatorAddress {
		self.Add(self, delta)
	} else {
		delegated.Add(delegated, delta)
	}
	if self.Sign() < 0 || delegated.Sign() < 0 {
		return accountingError("aggregate underflow")
	}
	total := new(big.Int).Add(self, delegated)
	if total.BitLen() > 256 {
		return accountingError("validator stake overflow")
	}
	if remaining.Sign() == 0 {
		k.DeleteDelegation(ctx, delegator, v.OperatorAddress)
	} else {
		d.Amount = remaining.String()
		k.SetDelegation(ctx, d)
	}
	v.SelfDelegation, v.DelegatedStake, v.TotalStake = self.String(), delegated.String(), total.String()
	if tier, changed := k.CheckTierTransition(ctx, v); changed {
		k.ApplyTierTransition(ctx, v, tier, "claim_backed_stake_change")
	}
	k.SetValidator(ctx, v)
	return nil
}

func (k Keeper) createAccountingUnbonding(ctx sdk.Context, delegator, validator string, amount *big.Int) (string, uint64, error) {
	if ctx.BlockHeight() < 0 {
		return "", 0, accountingError("negative block height")
	}
	height := uint64(ctx.BlockHeight())
	period := k.GetParams(ctx).UnbondingPeriod
	if period == 0 || period > uint64(math.MaxInt64)-height {
		return "", 0, accountingError("unbonding height overflow")
	}
	seqBytes := k.getStore(ctx).Get(types.UnbondingSeqKey)
	if seqBytes != nil && len(seqBytes) != 8 {
		return "", 0, accountingError("invalid unbonding sequence")
	}
	seq := k.GetUnbondingSeq(ctx)
	if seq == math.MaxUint64 {
		return "", 0, accountingError("unbonding sequence exhausted")
	}
	seq++
	id := fmt.Sprintf("%s_%s_%d_%d", delegator, validator, height, seq)
	if k.getStore(ctx).Has(types.UnbondingKey(id)) {
		return "", 0, accountingError("unbonding claim already exists")
	}
	k.SetUnbondingSeq(ctx, seq)
	k.SetUnbonding(ctx, &types.UnbondingEntry{Id: id, DelegatorAddress: delegator, ValidatorAddress: validator, Amount: amount.String(), CreatedAtHeight: height, CompletesAtHeight: height + period, Status: "pending"})
	return id, height + period, nil
}

func (ms *msgServer) safeRegisterValidator(goCtx context.Context, msg *types.MsgRegisterValidator) (*types.MsgRegisterValidatorResponse, error) {
	return accountingMutation(sdk.UnwrapSDKContext(goCtx), func(ctx sdk.Context) (*types.MsgRegisterValidatorResponse, error) {
		if msg == nil {
			return nil, accountingError("missing registration")
		}
		if err := msg.ValidateBasic(); err != nil {
			return nil, err
		}
		if len(msg.ConsensusPubkey) > 4096 {
			return nil, accountingError("legacy consensus-key description too large")
		}
		if err := canonicalAccountingAddress(msg.Operator); err != nil {
			return nil, err
		}
		if _, err := accountingAmount(msg.SelfDelegation, true); err != nil {
			return nil, err
		}
		if k := ms.getStore(ctx); k.Has(types.ValidatorKey(msg.Operator)) || k.Has(types.DelegationKey(msg.Operator, msg.Operator)) {
			return nil, types.ErrValidatorAlreadyExists
		}
		if msg.Did != "" && ms.getStore(ctx).Has(types.ValidatorByDIDKey(msg.Did)) {
			return nil, types.ErrDIDAlreadyRegistered
		}
		resp, err := ms.legacyRegisterValidator(ctx, msg)
		if err != nil {
			return nil, err
		}
		if _, err := ms.checkedAccountingValidator(ctx, msg.Operator); err != nil {
			return nil, err
		}
		return resp, nil
	})
}

func (ms *msgServer) safeDelegate(goCtx context.Context, msg *types.MsgDelegate) (*types.MsgDelegateResponse, error) {
	return accountingMutation(sdk.UnwrapSDKContext(goCtx), func(ctx sdk.Context) (*types.MsgDelegateResponse, error) {
		if msg == nil {
			return nil, accountingError("missing delegation")
		}
		amount, err := accountingAmount(msg.Amount, true)
		if err != nil {
			return nil, err
		}
		v, err := ms.checkedAccountingValidator(ctx, msg.Validator)
		if err != nil {
			return nil, err
		}
		if !v.IsActive {
			return nil, types.ErrValidatorInactive
		}
		if err := canonicalAccountingAddress(msg.Delegator); err != nil {
			return nil, err
		}
		if err := ms.changeAccountingDelegation(ctx, v, msg.Delegator, amount); err != nil {
			return nil, err
		}
		addr, _ := sdk.AccAddressFromBech32(msg.Delegator)
		if err := ms.bankKeeper.SendCoinsFromAccountToModule(ctx, addr, types.ModuleName, accountingCoins(amount)); err != nil {
			return nil, err
		}
		d, err := ms.checkedAccountingDelegation(ctx, msg.Delegator, msg.Validator)
		if err != nil {
			return nil, err
		}
		ctx.EventManager().EmitEvent(sdk.NewEvent("zerone.staking.delegation_created", sdk.NewAttribute("delegator", msg.Delegator), sdk.NewAttribute("validator", msg.Validator), sdk.NewAttribute("amount", amount.String())))
		return &types.MsgDelegateResponse{NewDelegation: d.Amount}, nil
	})
}

func (ms *msgServer) safeUndelegate(goCtx context.Context, msg *types.MsgUndelegate) (*types.MsgUndelegateResponse, error) {
	return accountingMutation(sdk.UnwrapSDKContext(goCtx), func(ctx sdk.Context) (*types.MsgUndelegateResponse, error) {
		if msg == nil {
			return nil, accountingError("missing undelegation")
		}
		amount, err := accountingAmount(msg.Amount, true)
		if err != nil {
			return nil, err
		}
		v, err := ms.checkedAccountingValidator(ctx, msg.Validator)
		if err != nil {
			return nil, err
		}
		if err := ms.changeAccountingDelegation(ctx, v, msg.Delegator, new(big.Int).Neg(amount)); err != nil {
			return nil, err
		}
		id, completes, err := ms.createAccountingUnbonding(ctx, msg.Delegator, msg.Validator, amount)
		if err != nil {
			return nil, err
		}
		ctx.EventManager().EmitEvent(sdk.NewEvent("zerone.staking.delegation_unbonding", sdk.NewAttribute("delegator", msg.Delegator), sdk.NewAttribute("validator", msg.Validator), sdk.NewAttribute("amount", amount.String()), sdk.NewAttribute("completes_at", fmt.Sprint(completes))))
		return &types.MsgUndelegateResponse{UnbondingId: id, CompletesAtHeight: completes}, nil
	})
}

func (ms *msgServer) safeRedelegate(goCtx context.Context, msg *types.MsgRedelegate) (*types.MsgRedelegateResponse, error) {
	return accountingMutation(sdk.UnwrapSDKContext(goCtx), func(ctx sdk.Context) (*types.MsgRedelegateResponse, error) {
		if msg == nil {
			return nil, accountingError("missing redelegation")
		}
		if msg.SrcValidator == msg.DstValidator {
			return nil, types.ErrSameValidator
		}
		if ctx.BlockHeight() <= 0 {
			return nil, accountingError("redelegation requires a positive height")
		}
		amount, err := accountingAmount(msg.Amount, true)
		if err != nil {
			return nil, err
		}
		src, err := ms.checkedAccountingValidator(ctx, msg.SrcValidator)
		if err != nil {
			return nil, err
		}
		dst, err := ms.checkedAccountingValidator(ctx, msg.DstValidator)
		if err != nil {
			return nil, err
		}
		if !dst.IsActive {
			return nil, types.ErrValidatorInactive
		}
		if err := canonicalAccountingAddress(msg.Delegator); err != nil {
			return nil, err
		}
		cooldown := ms.getStore(ctx).Get(types.RedelegationCooldownKey(msg.Delegator))
		if cooldown != nil && len(cooldown) != 8 {
			return nil, accountingError("invalid cooldown")
		}
		last := ms.GetLastRedelegationHeight(ctx, msg.Delegator)
		height := uint64(ctx.BlockHeight())
		wait := ms.GetParams(ctx).RedelegationCooldownBlocks
		if last > height || (last > 0 && height-last < wait) {
			return nil, types.ErrRedelegationCooldown
		}
		if err := ms.changeAccountingDelegation(ctx, src, msg.Delegator, new(big.Int).Neg(amount)); err != nil {
			return nil, err
		}
		if err := ms.changeAccountingDelegation(ctx, dst, msg.Delegator, amount); err != nil {
			return nil, err
		}
		ms.SetLastRedelegationHeight(ctx, msg.Delegator, height)
		ctx.EventManager().EmitEvent(sdk.NewEvent("zerone.staking.delegation_redelegated", sdk.NewAttribute("delegator", msg.Delegator), sdk.NewAttribute("src_validator", msg.SrcValidator), sdk.NewAttribute("dst_validator", msg.DstValidator), sdk.NewAttribute("amount", amount.String())))
		return &types.MsgRedelegateResponse{}, nil
	})
}

func (ms *msgServer) safeUpdateValidatorStake(goCtx context.Context, msg *types.MsgUpdateValidatorStake) (*types.MsgUpdateValidatorStakeResponse, error) {
	return accountingMutation(sdk.UnwrapSDKContext(goCtx), func(ctx sdk.Context) (*types.MsgUpdateValidatorStakeResponse, error) {
		if msg == nil {
			return nil, accountingError("missing self-delegation update")
		}
		amount, err := accountingAmount(msg.Amount, true)
		if err != nil {
			return nil, err
		}
		v, err := ms.checkedAccountingValidator(ctx, msg.Operator)
		if err != nil {
			return nil, err
		}
		delta := new(big.Int).Set(amount)
		if !msg.Increase {
			delta.Neg(delta)
		}
		if err := ms.changeAccountingDelegation(ctx, v, msg.Operator, delta); err != nil {
			return nil, err
		}
		if msg.Increase {
			addr, _ := sdk.AccAddressFromBech32(msg.Operator)
			if err := ms.bankKeeper.SendCoinsFromAccountToModule(ctx, addr, types.ModuleName, accountingCoins(amount)); err != nil {
				return nil, err
			}
		} else {
			if _, _, err := ms.createAccountingUnbonding(ctx, msg.Operator, msg.Operator, amount); err != nil {
				return nil, err
			}
			if v.SelfDelegation == "0" && v.Tier == types.TierApprentice && v.TotalVerifications < 22 {
				v.IsActive = false
				ms.SetValidator(ctx, v)
			}
		}
		ctx.EventManager().EmitEvent(sdk.NewEvent("zerone.staking.update_validator_stake", sdk.NewAttribute("operator", msg.Operator), sdk.NewAttribute("new_stake", v.SelfDelegation)))
		return &types.MsgUpdateValidatorStakeResponse{}, nil
	})
}

func (ms *msgServer) safeUpdateParams(goCtx context.Context, msg *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	return accountingMutation(sdk.UnwrapSDKContext(goCtx), func(ctx sdk.Context) (*types.MsgUpdateParamsResponse, error) {
		if msg == nil || msg.Params == nil {
			return nil, types.ErrInvalidParams
		}
		if err := validateAccountingParams(msg.Params); err != nil {
			return nil, err
		}
		return ms.legacyUpdateParams(ctx, msg)
	})
}

func (k Keeper) recordSealedMonetarySlash(ctx sdk.Context, validator, reason, destination string) {
	ctx.EventManager().EmitEvent(sdk.NewEvent("zerone.staking.monetary_slash_not_applied", sdk.NewAttribute("validator", validator), sdk.NewAttribute("reason", reason), sdk.NewAttribute("requested_destination", destination), sdk.NewAttribute("actual_amount", "0"), sdk.NewAttribute("policy", "explicit_task_escrow_required")))
}

func (k Keeper) processAccountingUnbondings(ctx sdk.Context) error {
	// Discover without silently dropping malformed unbondings. Validate the whole
	// ledger once if any claim is due, before paying the first claimant.
	iter := k.getStore(ctx).Iterator(types.UnbondingKeyPrefix, []byte{types.UnbondingKeyPrefix[0] + 1})
	mature := []*types.UnbondingEntry{}
	count := 0
	for ; iter.Valid(); iter.Next() {
		count++
		if count > MaxAccountingStoreEntries {
			iter.Close()
			return accountingError("unbonding capacity exceeded")
		}
		var u types.UnbondingEntry
		if err := decodeAccountingJSON(iter.Value(), &u); err != nil {
			iter.Close()
			return err
		}
		if _, _, err := validateAccountingUnbonding(&u); err != nil {
			iter.Close()
			return err
		}
		if ctx.BlockHeight() >= 0 && u.Status == "pending" && u.CompletesAtHeight <= uint64(ctx.BlockHeight()) {
			mature = append(mature, &u)
		}
	}
	if err := errors.Join(accountingIteratorError(iter), iter.Close()); err != nil {
		return accountingError("unbonding scan failed: %v", err)
	}
	if len(mature) == 0 {
		return nil
	}
	if err := k.ValidateAccountingSafety(ctx); err != nil {
		return err
	}
	_, err := accountingMutation(ctx, func(cache sdk.Context) (*struct{}, error) {
		for _, u := range mature {
			addr, _ := sdk.AccAddressFromBech32(u.DelegatorAddress)
			amount, _ := accountingAmount(u.Amount, true)
			if err := k.bankKeeper.SendCoinsFromModuleToAccount(cache, types.ModuleName, addr, accountingCoins(amount)); err != nil {
				return nil, err
			}
			u.Status = "completed"
			k.SetUnbonding(cache, u)
			cache.EventManager().EmitEvent(sdk.NewEvent("zerone.staking.unbonding_completed", sdk.NewAttribute("delegator", u.DelegatorAddress), sdk.NewAttribute("amount", u.Amount), sdk.NewAttribute("unbonding_id", u.Id)))
		}
		return &struct{}{}, nil
	})
	return err
}
