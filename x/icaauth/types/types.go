package types

import (
	"fmt"
	"time"
)

// MinTimeout is the minimum allowed timeout for ICA transactions (60 seconds).
var MinTimeout = uint64(60 * time.Second)

// DefaultParams returns default icaauth parameters.
// SECURITY: MsgTransfer is intentionally excluded from AllowedHostMsgTypes (P0-6).
func DefaultParams() *Params {
	return &Params{
		MaxRemoteAccountsPerOwner: 5,
		AllowedHostMsgTypes: []string{
			"/cosmos.bank.v1beta1.MsgSend",
			"/cosmos.staking.v1beta1.MsgDelegate",
			"/cosmos.staking.v1beta1.MsgUndelegate",
			"/cosmos.staking.v1beta1.MsgBeginRedelegate",
			"/cosmos.distribution.v1beta1.MsgWithdrawDelegatorReward",
			"/cosmos.gov.v1beta1.MsgVote",
		},
		RegistrationCooldown: 100, // blocks
		MaxMessagesPerTx:     5,
	}
}

// DefaultGenesis returns the default genesis state for icaauth.
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Params:  DefaultParams(),
		Records: nil,
	}
}

// Validate validates the genesis state.
func (gs *GenesisState) Validate() error {
	if gs.Params == nil {
		return fmt.Errorf("params cannot be nil")
	}
	if err := gs.Params.Validate(); err != nil {
		return fmt.Errorf("invalid params: %w", err)
	}
	for i, rec := range gs.Records {
		if rec.Owner == "" {
			return fmt.Errorf("records[%d]: owner cannot be empty", i)
		}
	}
	return nil
}

// Validate validates the Params.
func (p *Params) Validate() error {
	if p.MaxRemoteAccountsPerOwner == 0 {
		return fmt.Errorf("max_remote_accounts_per_owner must be positive")
	}
	if p.MaxMessagesPerTx == 0 {
		return fmt.Errorf("max_messages_per_tx must be positive")
	}
	return nil
}

// ValidateBasic for MsgRegisterAccount.
func (m *MsgRegisterAccount) ValidateBasic() error {
	if m.Owner == "" {
		return fmt.Errorf("owner cannot be empty")
	}
	if m.ConnectionId == "" {
		return fmt.Errorf("connection_id cannot be empty")
	}
	return nil
}

// ValidateBasic for MsgSubmitTx.
func (m *MsgSubmitTx) ValidateBasic() error {
	if m.Owner == "" {
		return fmt.Errorf("owner cannot be empty")
	}
	if m.ConnectionId == "" {
		return fmt.Errorf("connection_id cannot be empty")
	}
	if len(m.Msgs) == 0 {
		return fmt.Errorf("msgs cannot be empty")
	}
	if m.TimeoutNs == 0 {
		return fmt.Errorf("timeout_ns cannot be zero")
	}
	if m.TimeoutNs < MinTimeout {
		return fmt.Errorf("timeout_ns must be at least %d (60s)", MinTimeout)
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
	return m.Params.Validate()
}
