package types

import (
	"fmt"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// ─────────────────────────────────────────────────────────────────────────────
// ValidateBasic / GetSigners for proto-generated messages
// ─────────────────────────────────────────────────────────────────────────────

// --- MsgSneeze ---

func (msg *MsgSneeze) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Sender)
	return []sdk.AccAddress{addr}
}

func (msg *MsgSneeze) ValidateBasic() error {
	if msg.Sender == "" {
		return fmt.Errorf("sender cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(msg.Sender); err != nil {
		return fmt.Errorf("invalid sender address: %w", err)
	}
	if msg.To == "" {
		return fmt.Errorf("to cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(msg.To); err != nil {
		return fmt.Errorf("invalid to address: %w", err)
	}
	if msg.Sender == msg.To {
		return ErrSelfSneeze
	}
	amt := new(big.Int)
	if _, ok := amt.SetString(msg.Amount, 10); !ok || amt.Sign() <= 0 {
		return fmt.Errorf("amount must be a positive integer")
	}
	// token_id is optional (defaults to configured ZO); if present, must be
	// non-empty and is validated against the configured token in the keeper.
	return nil
}

// --- MsgSetContagion ---

func (msg *MsgSetContagion) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Authority)
	return []sdk.AccAddress{addr}
}

func (msg *MsgSetContagion) ValidateBasic() error {
	if msg.Authority == "" {
		return fmt.Errorf("authority cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(msg.Authority); err != nil {
		return fmt.Errorf("invalid authority address: %w", err)
	}
	if msg.TokenId == "" {
		return fmt.Errorf("token_id cannot be empty")
	}
	if !positiveBigInt(msg.ReserveInitial) {
		return ErrInvalidReserve
	}
	if !positiveBigInt(msg.RewardSender) {
		return ErrInvalidReward
	}
	if !positiveBigInt(msg.RewardReceiver) {
		return ErrInvalidReward
	}
	return nil
}

func positiveBigInt(s string) bool {
	v := new(big.Int)
	if _, ok := v.SetString(s, 10); !ok {
		return false
	}
	return v.Sign() > 0
}

// ─────────────────────────────────────────────────────────────────────────────
// Params + Genesis helpers
// ─────────────────────────────────────────────────────────────────────────────

// DefaultParams returns the (empty) default params. Contagion has no
// governance-mutable params on purpose — configuration lives in
// ContagionState, which is set once and frozen.
func DefaultParams() Params { return Params{} }

// DefaultGenesis returns the default (unconfigured) genesis state. A real ZO
// chain supplies a configured state at genesis instead.
func DefaultGenesis() *GenesisState {
	p := DefaultParams()
	_ = p
	return &GenesisState{
		State: nil, // unconfigured until MsgSetContagion or genesis init
	}
}

// Validate validates the genesis state.
func (gs *GenesisState) Validate() error {
	// Params is intentionally empty; nothing to validate there.
	if gs.State == nil {
		// Unconfigured genesis is valid — module just won't fire sneezes until
		// MsgSetContagion is sent.
		return nil
	}
	if gs.State.TokenId == "" {
		return fmt.Errorf("genesis state token_id cannot be empty")
	}
	if !positiveBigInt(gs.State.ReserveInitial) {
		return fmt.Errorf("genesis state reserve_initial must be positive")
	}
	if !positiveBigInt(gs.State.RewardSender) || !positiveBigInt(gs.State.RewardReceiver) {
		return fmt.Errorf("genesis state rewards must be positive")
	}
	// If configured at genesis, authority MUST be empty (renounced).
	if gs.State.Configured && gs.State.Authority != "" {
		return fmt.Errorf("configured genesis state must have empty authority (renounced)")
	}
	return nil
}

// Validate is a no-op for the empty Params message.
func (p *Params) Validate() error { return nil }
