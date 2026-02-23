package types

import (
	"fmt"
	"math/big"
)

// DefaultParams returns the default ibcratelimit parameters.
func DefaultParams() *Params {
	return &Params{
		Enabled: true,
	}
}

// DefaultGenesis returns the default genesis state for ibcratelimit.
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Params:     DefaultParams(),
		RateLimits: nil,
	}
}

// Validate validates the genesis state.
func (gs *GenesisState) Validate() error {
	if gs.Params == nil {
		return fmt.Errorf("params cannot be nil")
	}
	for i, rl := range gs.RateLimits {
		if err := ValidateRateLimit(rl); err != nil {
			return fmt.Errorf("rate_limits[%d]: %w", i, err)
		}
	}
	return nil
}

// ValidateRateLimit validates a single RateLimit entry.
func ValidateRateLimit(rl *RateLimit) error {
	if rl.ChannelId == "" {
		return fmt.Errorf("channel_id cannot be empty")
	}
	if rl.Denom == "" {
		return fmt.Errorf("denom cannot be empty")
	}
	maxSend := new(big.Int)
	if _, ok := maxSend.SetString(rl.MaxSend, 10); !ok || maxSend.Sign() <= 0 {
		return fmt.Errorf("max_send must be a positive integer, got %q", rl.MaxSend)
	}
	maxRecv := new(big.Int)
	if _, ok := maxRecv.SetString(rl.MaxRecv, 10); !ok || maxRecv.Sign() <= 0 {
		return fmt.Errorf("max_recv must be a positive integer, got %q", rl.MaxRecv)
	}
	if rl.WindowBlocks == 0 {
		return fmt.Errorf("window_blocks must be positive")
	}
	return nil
}

// ValidateBasic for MsgAddRateLimit.
func (m *MsgAddRateLimit) ValidateBasic() error {
	if m.Authority == "" {
		return fmt.Errorf("authority cannot be empty")
	}
	if m.ChannelId == "" {
		return fmt.Errorf("channel_id cannot be empty")
	}
	if m.Denom == "" {
		return fmt.Errorf("denom cannot be empty")
	}
	maxSend := new(big.Int)
	if _, ok := maxSend.SetString(m.MaxSend, 10); !ok || maxSend.Sign() <= 0 {
		return fmt.Errorf("max_send must be a positive integer")
	}
	maxRecv := new(big.Int)
	if _, ok := maxRecv.SetString(m.MaxRecv, 10); !ok || maxRecv.Sign() <= 0 {
		return fmt.Errorf("max_recv must be a positive integer")
	}
	if m.WindowBlocks == 0 {
		return fmt.Errorf("window_blocks must be positive")
	}
	return nil
}

// ValidateBasic for MsgRemoveRateLimit.
func (m *MsgRemoveRateLimit) ValidateBasic() error {
	if m.Authority == "" {
		return fmt.Errorf("authority cannot be empty")
	}
	if m.ChannelId == "" {
		return fmt.Errorf("channel_id cannot be empty")
	}
	if m.Denom == "" {
		return fmt.Errorf("denom cannot be empty")
	}
	return nil
}

// ValidateBasic for MsgUpdateParams.
func (m *MsgUpdateParams) ValidateBasic() error {
	if m.Authority == "" {
		return fmt.Errorf("authority cannot be empty")
	}
	if m.Params == nil {
		return fmt.Errorf("params cannot be nil")
	}
	return nil
}
