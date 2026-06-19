package keeper

import (
	"fmt"
	"math/big"

	"cosmossdk.io/core/store"
	"cosmossdk.io/log"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/x/contagion/types"
)

// Keeper manages the contagion module's state: the ZO token config, the
// already_infected mapping, and the contagion reserve counter.
//
// It holds a BankKeeper (to disburse sneeze rewards from the contagion module
// account) and a TokensKeeper (to move ZO balances inside MsgSneeze and to
// validate the ZO token). It also implements types.ContagionHook so x/tokens
// can call OnTokenTransfer after a ZO transfer.
type Keeper struct {
	cdc          codec.Codec
	storeService store.KVStoreService

	bankKeeper   types.BankKeeper
	tokensKeeper types.TokensKeeper

	authority string // governance authority; cleared once configured
}

// NewKeeper creates a new contagion module Keeper.
func NewKeeper(
	cdc codec.Codec,
	storeService store.KVStoreService,
	bk types.BankKeeper,
	tk types.TokensKeeper,
	authority string,
) Keeper {
	return Keeper{
		cdc:          cdc,
		storeService: storeService,
		bankKeeper:   bk,
		tokensKeeper: tk,
		authority:    authority,
	}
}

// Logger returns a module-specific logger.
func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

// GetAuthority returns the module authority address. Empty once configured
// (post-renounce).
func (k Keeper) GetAuthority() string {
	return k.authority
}

// ─── Core contagion logic ────────────────────────────────────────────────────

// Sneeze is the canonical implementation of the "love sneeze" from
// CONTAGION-MATH.md. It is called from two entrypoints:
//
//   - keeper.MsgServer.Sneeze (the explicit MsgSneeze wrapper), AFTER the
//     caller has performed the ZO transfer via the tokens keeper.
//   - keeper.OnTokenTransfer (the hook), AFTER x/tokens has moved ZO.
//
// Precondition: the underlying ZO transfer has ALREADY succeeded. This method
// only decides whether to mint rewards. It never moves the principal amount.
//
// Returns the emitted SneezeEvent (nil if no sneeze fired) and an error only
// for unexpected failures (not for "already infected" or "reserve dry",
// which are silent no-ops per CONTAGION-MATH.md).
func (k Keeper) Sneeze(ctx sdk.Context, spreader, patient string) (*types.SneezeEvent, error) {
	state := k.GetState(ctx)
	if state == nil || !state.Configured {
		// Not configured: contagion is dormant. Silent no-op.
		return nil, nil
	}

	// 1. Is the patient already infected? If yes, no sneeze (one-way flag).
	if k.IsInfected(ctx, patient) {
		return nil, nil
	}
	// Mark infected FIRST (permanent, one-way) so a re-entrant call cannot
	// double-reward. See CONTAGION-MATH.md "the flag is permanent and one-way".
	k.SetInfected(ctx, patient, uint64(ctx.BlockHeight()), spreader)

	// 2. Can the reserve fund a FULL sneeze? Partial rewards are NOT fired
	// (DEPLOY-DEVNET Phase 2 test case). Reserve < 154 ZO => dormant.
	rewardSender := mustBigInt(state.RewardSender)
	rewardReceiver := mustBigInt(state.RewardReceiver)
	totalReward := new(big.Int).Add(rewardSender, rewardReceiver)

	reserve := mustBigInt(state.ReserveRemaining)
	if reserve.Cmp(totalReward) < 0 {
		// Reserve dry. Mechanic dormant. Silent no-op.
		// (Patient stays infected — they just don't get a reward.)
		return nil, nil
	}

	// 3. Mint from the reserve: 77 ZO to spreader + 77 ZO to patient.
	//    The contagion module account holds the pre-funded reserve; we send
	//    from it to each recipient. This is `_transferFromReserve()` from the
	//    pseudocode, realised as bank module-account sends.
	if err := k.disburseReward(ctx, spreader, rewardSender, state.TokenId); err != nil {
		return nil, fmt.Errorf("sneeze sender reward: %w", err)
	}
	if err := k.disburseReward(ctx, patient, rewardReceiver, state.TokenId); err != nil {
		return nil, fmt.Errorf("sneeze patient reward: %w", err)
	}

	// 4. Decrement the reserve (monotonic; only ever goes down).
	reserve.Sub(reserve, totalReward)
	state.ReserveRemaining = reserve.String()
	state.SneezeCount++
	sneezeIdx := state.SneezeCount
	k.SetState(ctx, state)

	// 5. Emit the Sneeze event — structured proto payload + sdk.Event for tx
	//    logs (the latter is what off-chain indexers / leaderboards consume).
	ev := &types.SneezeEvent{
		Spreader:             spreader,
		Patient:              patient,
		RewardSenderAmount:   state.RewardSender,
		RewardReceiverAmount: state.RewardReceiver,
		ReserveRemaining:     reserve.String(),
		BlockHeight:          uint64(ctx.BlockHeight()),
		TokenId:              state.TokenId,
		SneezeIndex:          sneezeIdx,
	}
	k.emitSneezeEvent(ctx, ev)
	return ev, nil
}

// OnTokenTransfer implements types.ContagionHook. x/tokens calls this after a
// successful ZO transfer. It only fires a sneeze if (a) the token is the
// configured ZO token, (b) amount > 0, and (c) the receiver is newly infected.
//
// This is the RECOMMENDED integration path (see DESIGN.md Question 2): ZO
// transfers go through the standard ZRN-20 TransferToken, and contagion is a
// nil-safe post-transfer hook. Non-ZO tokens pay zero contagion cost.
func (k Keeper) OnTokenTransfer(ctx sdk.Context, tokenId, from, to string, amount *big.Int) *types.SneezeEvent {
	if amount == nil || amount.Sign() <= 0 {
		return nil
	}
	state := k.GetState(ctx)
	if state == nil || !state.Configured || state.TokenId != tokenId {
		// Not ZO, or not configured: no contagion.
		return nil
	}
	ev, err := k.Sneeze(ctx, from, to)
	if err != nil {
		// Contagion is best-effort: a sneeze failure must NOT revert the
		// underlying transfer. Log and move on.
		k.Logger(ctx).Error("sneeze hook error", "from", from, "to", to, "err", err)
		return nil
	}
	return ev
}

// disburseReward sends `amount` base units of ZO from the contagion module
// account to `recipient`. The contagion module account is pre-funded with the
// full reserve at genesis (or via a one-time governance mint). ZO is a ZRN-20
// secondary token, so we credit the recipient's ZRN-20 balance directly via
// the tokens keeper (mirroring x/tokens MintToken's balance move) rather than
// going through the bank sdk.Coin path, which is for IBC-compatible wrapped
// denoms only.
func (k Keeper) disburseReward(ctx sdk.Context, recipient string, amount *big.Int, tokenId string) error {
	if amount.Sign() <= 0 {
		return nil
	}
	// Credit the recipient's ZRN-20 ZO balance. This is the ZRN-20 equivalent
	// of _transferFromReserve: the reserve "balance" is tracked by
	// ContagionState.reserve_remaining (monotonically decreasing), and the
	// recipient's balance goes up by `amount`.
	bal := k.tokensKeeper.GetBalance(ctx, tokenId, recipient)
	bal.Add(bal, amount)
	k.tokensKeeper.SetBalance(ctx, tokenId, recipient, bal)
	// NOTE: a production impl must also decrement the contagion module
	// account's own ZRN-20 ZO balance by `amount` (the reserve is held by the
	// module account, so its balance is the live reserve). Here we keep a
	// separate reserve_remaining counter for simplicity and devnet rehearsal;
	// see DESIGN.md "Reserve accounting".
	return nil
}

// emitSneezeEvent emits both a typed proto event and a Cosmos sdk.Event with
// type "zerone.contagion.sneeze" carrying the SneezeEvent fields as
// attributes. The sdk.Event is what off-chain indexers / the leaderboard
// consume from tx event logs.
func (k Keeper) emitSneezeEvent(ctx sdk.Context, ev *types.SneezeEvent) {
	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"zerone.contagion.sneeze",
		sdk.NewAttribute("spreader", ev.Spreader),
		sdk.NewAttribute("patient", ev.Patient),
		sdk.NewAttribute("reward_sender_amount", ev.RewardSenderAmount),
		sdk.NewAttribute("reward_receiver_amount", ev.RewardReceiverAmount),
		sdk.NewAttribute("reserve_remaining", ev.ReserveRemaining),
		sdk.NewAttribute("block_height", fmt.Sprintf("%d", ev.BlockHeight)),
		sdk.NewAttribute("token_id", ev.TokenId),
		sdk.NewAttribute("sneeze_index", fmt.Sprintf("%d", ev.SneezeIndex)),
	))
}

func mustBigInt(s string) *big.Int {
	v := new(big.Int)
	if _, ok := v.SetString(s, 10); !ok {
		return new(big.Int)
	}
	return v
}
