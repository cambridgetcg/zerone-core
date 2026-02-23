package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Channel status constants.
const (
	ChannelStatusOpen     = "open"
	ChannelStatusClosing  = "closing"
	ChannelStatusDisputed = "disputed"
	ChannelStatusSettled  = "settled"
)

// DefaultParams returns default module parameters.
func DefaultParams() *Params {
	return &Params{
		MinDeposit:           "1000000",  // 1 ZRN
		MinTimeoutBlocks:     100,
		MaxTimeoutBlocks:     1000000,
		DisputeWindowBlocks:  500,
		DefaultSettlementFreq: 100,
		MaxChannelsPerPair:   10,
		ChannelOpenFee:       "100000", // 0.1 ZRN
	}
}

// DefaultGenesis returns the default genesis state.
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Params: DefaultParams(),
	}
}

// Validate validates the genesis state.
func (gs *GenesisState) Validate() error {
	if gs.Params == nil {
		return fmt.Errorf("params cannot be nil")
	}
	return gs.Params.Validate()
}

// Validate validates the module parameters.
func (p *Params) Validate() error {
	if p.MinTimeoutBlocks == 0 {
		return fmt.Errorf("min timeout blocks must be positive")
	}
	if p.MaxTimeoutBlocks == 0 {
		return fmt.Errorf("max timeout blocks must be positive")
	}
	if p.DisputeWindowBlocks == 0 {
		return fmt.Errorf("dispute window blocks must be positive")
	}
	if p.MaxChannelsPerPair == 0 {
		return fmt.Errorf("max channels per pair must be positive")
	}
	return nil
}

// --- ValidateBasic methods ---

func (m *MsgOpenChannel) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Payer); err != nil {
		return fmt.Errorf("invalid payer address: %w", err)
	}
	if _, err := sdk.AccAddressFromBech32(m.Receiver); err != nil {
		return fmt.Errorf("invalid receiver address: %w", err)
	}
	if m.Payer == m.Receiver {
		return fmt.Errorf("payer and receiver cannot be the same address")
	}
	if m.Deposit == "" || m.Deposit == "0" {
		return fmt.Errorf("deposit must be positive")
	}
	if m.TimeoutBlocks == 0 {
		return fmt.Errorf("timeout blocks must be positive")
	}
	return nil
}

func (m *MsgDepositChannel) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Depositor); err != nil {
		return fmt.Errorf("invalid depositor address: %w", err)
	}
	if m.ChannelId == "" {
		return fmt.Errorf("channel id cannot be empty")
	}
	if m.Amount == "" || m.Amount == "0" {
		return fmt.Errorf("amount must be positive")
	}
	return nil
}

func (m *MsgUpdateState) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Sender); err != nil {
		return fmt.Errorf("invalid sender address: %w", err)
	}
	if m.ChannelId == "" {
		return fmt.Errorf("channel id cannot be empty")
	}
	if m.Update == nil {
		return fmt.Errorf("update cannot be nil")
	}
	return nil
}

func (m *MsgCloseChannel) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Closer); err != nil {
		return fmt.Errorf("invalid closer address: %w", err)
	}
	if m.ChannelId == "" {
		return fmt.Errorf("channel id cannot be empty")
	}
	return nil
}

func (m *MsgDisputeChannel) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Disputer); err != nil {
		return fmt.Errorf("invalid disputer address: %w", err)
	}
	if m.ChannelId == "" {
		return fmt.Errorf("channel id cannot be empty")
	}
	return nil
}

func (m *MsgClaimExpired) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Claimer); err != nil {
		return fmt.Errorf("invalid claimer address: %w", err)
	}
	if m.ChannelId == "" {
		return fmt.Errorf("channel id cannot be empty")
	}
	return nil
}

func (msg *MsgUpdateParams) ValidateBasic() error {
	if msg.Authority == "" {
		return fmt.Errorf("authority cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(msg.Authority); err != nil {
		return fmt.Errorf("invalid authority address: %w", err)
	}
	if msg.Params == nil {
		return fmt.Errorf("params cannot be nil")
	}
	return msg.Params.Validate()
}

func (msg *MsgUpdateParams) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Authority)
	return []sdk.AccAddress{addr}
}
