package keeper

import (
	"context"
	"fmt"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/x/contagion/types"
)

type msgServer struct {
	Keeper
	types.UnimplementedMsgServer
}

// NewMsgServerImpl returns an implementation of the MsgServer interface.
func NewMsgServerImpl(keeper Keeper) *msgServer {
	return &msgServer{Keeper: keeper}
}

var _ types.MsgServer = (*msgServer)(nil)

// ─── MsgSneeze (wrapper path) ──────────────────────────────────────────────────

// Sneeze is the explicit user-facing contagion entrypoint. It:
//  1. validates the ZO token,
//  2. performs the ZRN-20 transfer of `amount` from sender to `to` via the
//     tokens keeper (delegating — contagion does NOT reimplement transfer),
//  3. runs the contagion check via keeper.Sneeze().
//
// This is the fallback path used when a transfer happens outside the
// x/tokens hook (e.g. an airdrop script) or during devnet rehearsal before the
// hook is wired. See DESIGN.md Question 2.
func (s *msgServer) Sneeze(goCtx context.Context, msg *types.MsgSneeze) (*types.MsgSneezeResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	state := s.GetState(ctx)
	if state == nil || !state.Configured {
		return nil, types.ErrNotConfigured
	}

	// Resolve the ZO token id: explicit override must match config; empty
	// defaults to the configured ZO token.
	tokenId := state.TokenId
	if msg.TokenId != "" && msg.TokenId != state.TokenId {
		return nil, types.ErrTokenIdMismatch
	}

	// Validate the ZO token exists in the tokens module.
	tok := s.tokensKeeper.GetToken(ctx, tokenId)
	if tok == nil {
		return nil, types.ErrTokenNotFound
	}
	if tok.Paused {
		// Reuse the tokens-module-style error conceptually; contagion surfaces
		// it as a generic invalid-transfer cause.
		return nil, fmt.Errorf("ZO token is paused")
	}

	amount := new(big.Int)
	if _, ok := amount.SetString(msg.Amount, 10); !ok || amount.Sign() <= 0 {
		return nil, types.ErrInvalidAmount
	}

	// 1. Perform the real ZRN-20 transfer via the tokens keeper.
	if err := s.tokensKeeper.Transfer(ctx, tokenId, msg.Sender, msg.To, amount); err != nil {
		return nil, fmt.Errorf("zo transfer failed: %w", err)
	}

	// 2. Run the contagion check. Sneeze is best-effort: an already-infected
	// recipient or a dry reserve returns (sneezed=false) without error.
	ev, err := s.Keeper.Sneeze(ctx, msg.Sender, msg.To)
	if err != nil {
		return nil, err
	}

	resp := &types.MsgSneezeResponse{
		Sneezed:          ev != nil,
		ReserveRemaining: state.ReserveRemaining,
		SneezeIndex:      state.SneezeCount,
	}
	if ev != nil {
		resp.RewardSender = ev.RewardSenderAmount
		resp.RewardReceiver = ev.RewardReceiverAmount
	}
	return resp, nil
}

// ─── MsgSetContagion (one-time, self-disabling) ───────────────────────────────

// SetContagion configures ZO for contagion: the ZO token id, the reserve size,
// and the per-sneeze reward amounts. It is callable ONLY by `authority` and
// ONLY while the module is unconfigured. On success it flips `configured` to
// true and clears `authority` — the on-chain equivalent of PARAMETERS.md's
// "Full contract renounce / owner() returns 0x0". There is intentionally no
// UpdateContagion message; the formula is immutable post-deploy.
func (s *msgServer) SetContagion(goCtx context.Context, msg *types.MsgSetContagion) (*types.MsgSetContagionResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	existing := s.GetState(ctx)
	if existing != nil && existing.Configured {
		// Already configured — formula is frozen forever.
		return nil, types.ErrAlreadyConfigured
	}

	// Authority check: only the configured governance address may initialise.
	// After first set, authority is cleared, so this can never be re-called.
	if s.authority == "" || msg.Authority != s.authority {
		return nil, types.ErrUnauthorized
	}

	// Validate the ZO token exists and decimals match.
	tok := s.tokensKeeper.GetToken(ctx, msg.TokenId)
	if tok == nil {
		return nil, types.ErrTokenNotFound
	}
	if tok.Decimals != msg.Decimals {
		return nil, types.ErrDecimalsMismatch
	}

	reserveInitial := new(big.Int)
	if _, ok := reserveInitial.SetString(msg.ReserveInitial, 10); !ok || reserveInitial.Sign() <= 0 {
		return nil, types.ErrInvalidReserve
	}
	rewardSender := new(big.Int)
	if _, ok := rewardSender.SetString(msg.RewardSender, 10); !ok || rewardSender.Sign() <= 0 {
		return nil, types.ErrInvalidReward
	}
	rewardReceiver := new(big.Int)
	if _, ok := rewardReceiver.SetString(msg.RewardReceiver, 10); !ok || rewardReceiver.Sign() <= 0 {
		return nil, types.ErrInvalidReward
	}

	// Reserve must be able to fund at least one full sneeze to be useful, but
	// we do NOT require it — a reserve below 154 ZO simply means the mechanic
	// starts dormant (matches CONTAGION-MATH.md "reserve dry" behaviour).

	state := &types.ContagionState{
		TokenId:          msg.TokenId,
		ReserveInitial:   msg.ReserveInitial,
		ReserveRemaining: msg.ReserveInitial, // starts full
		RewardSender:     msg.RewardSender,
		RewardReceiver:   msg.RewardReceiver,
		Decimals:         msg.Decimals,
		Configured:       true,
		Authority:        "", // RENOUNCE: cleared on first set.
		SneezeCount:      0,
	}
	s.SetState(ctx, state)

	// Clear the in-memory authority so no future SetContagion can succeed.
	// (The on-chain state already has authority="" from above.)
	s.authority = ""

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"zerone.contagion.configured",
		sdk.NewAttribute("token_id", msg.TokenId),
		sdk.NewAttribute("reserve_initial", msg.ReserveInitial),
		sdk.NewAttribute("reward_sender", msg.RewardSender),
		sdk.NewAttribute("reward_receiver", msg.RewardReceiver),
		sdk.NewAttribute("decimals", fmt.Sprintf("%d", msg.Decimals)),
		sdk.NewAttribute("renounced", "true"),
	))

	return &types.MsgSetContagionResponse{}, nil
}
