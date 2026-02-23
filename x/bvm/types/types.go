package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	MaxContractGasLimit     uint64 = 10_000_000
	MaxScheduledGasPerBlock uint64 = 5_000_000
	MaxDeployBytecodeSize          = 1_048_576 // 1 MB stateless limit
	MaxInputDataSize               = 65_536    // 64 KB
)

// ValidateBasic performs stateless validation on MsgDeployContract.
func (m *MsgDeployContract) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Deployer); err != nil {
		return fmt.Errorf("invalid deployer address: %w", err)
	}
	if len(m.Bytecode) == 0 {
		return fmt.Errorf("bytecode cannot be empty")
	}
	if len(m.Bytecode) > MaxDeployBytecodeSize {
		return fmt.Errorf("bytecode too large: %d bytes (max %d)", len(m.Bytecode), MaxDeployBytecodeSize)
	}
	return nil
}

func (m *MsgDeployContract) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(m.Deployer)
	return []sdk.AccAddress{addr}
}

// ValidateBasic performs stateless validation on MsgCallContract.
func (m *MsgCallContract) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Caller); err != nil {
		return fmt.Errorf("invalid caller address: %w", err)
	}
	if m.ContractAddress == "" {
		return fmt.Errorf("contract address cannot be empty")
	}
	if m.GasLimit == 0 {
		return fmt.Errorf("gas limit must be positive")
	}
	if m.GasLimit > MaxContractGasLimit {
		return fmt.Errorf("gas limit %d exceeds maximum %d", m.GasLimit, MaxContractGasLimit)
	}
	if len(m.InputData) > MaxInputDataSize {
		return fmt.Errorf("input data too large: %d bytes (max %d)", len(m.InputData), MaxInputDataSize)
	}
	return nil
}

func (m *MsgCallContract) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(m.Caller)
	return []sdk.AccAddress{addr}
}

// ValidateBasic performs stateless validation on MsgScheduleExecution.
func (m *MsgScheduleExecution) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Scheduler); err != nil {
		return fmt.Errorf("invalid scheduler address: %w", err)
	}
	if m.ContractAddress == "" {
		return fmt.Errorf("contract address cannot be empty")
	}
	return nil
}

func (m *MsgScheduleExecution) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(m.Scheduler)
	return []sdk.AccAddress{addr}
}

// ValidateBasic performs stateless validation on MsgScheduleContract.
func (m *MsgScheduleContract) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Caller); err != nil {
		return fmt.Errorf("invalid caller address: %w", err)
	}
	if m.ContractAddress == "" {
		return fmt.Errorf("contract address cannot be empty")
	}
	if m.ExecuteAtBlock == 0 {
		return fmt.Errorf("execute_at_block must be positive")
	}
	if m.MaxGas == 0 {
		return fmt.Errorf("max_gas must be positive")
	}
	if m.MaxGas > MaxContractGasLimit {
		return fmt.Errorf("max_gas %d exceeds maximum %d", m.MaxGas, MaxContractGasLimit)
	}
	return nil
}

func (m *MsgScheduleContract) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(m.Caller)
	return []sdk.AccAddress{addr}
}

// ValidateBasic performs stateless validation on MsgCancelSchedule.
func (m *MsgCancelSchedule) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Caller); err != nil {
		return fmt.Errorf("invalid caller address: %w", err)
	}
	if m.ScheduleId == "" {
		return fmt.Errorf("schedule_id cannot be empty")
	}
	return nil
}

func (m *MsgCancelSchedule) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(m.Caller)
	return []sdk.AccAddress{addr}
}

// ValidateBasic performs stateless validation on MsgUpdateContractState.
func (m *MsgUpdateContractState) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Authority); err != nil {
		return fmt.Errorf("invalid authority address: %w", err)
	}
	if m.ContractAddress == "" {
		return fmt.Errorf("contract address cannot be empty")
	}
	if m.Key == "" {
		return fmt.Errorf("key cannot be empty")
	}
	return nil
}

func (m *MsgUpdateContractState) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(m.Authority)
	return []sdk.AccAddress{addr}
}

// ValidateBasic performs stateless validation on MsgUpdateParams.
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
