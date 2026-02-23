package types

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Health category constants.
const (
	CategoryCritical = "critical"
	CategoryDegraded = "degraded"
	CategoryHealthy  = "healthy"
)

// Dimension name constants.
const (
	DimKnowledgeQuality       = "knowledge_quality"
	DimEconomicStability      = "economic_stability"
	DimGovernanceParticipation = "governance_participation"
	DimNetworkSecurity        = "network_security"
	DimStakingRatio           = "staking_ratio"
)

// NeutralBPS is the neutral score returned when a sensor is unavailable.
const NeutralBPS = uint64(500_000)

// --- Message validation ---

func (msg *MsgUpdateParams) ValidateBasic() error {
	if msg.Authority == "" {
		return fmt.Errorf("authority cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(msg.Authority); err != nil {
		return fmt.Errorf("invalid authority address: %w", err)
	}
	if msg.Params != nil {
		if err := msg.Params.Validate(); err != nil {
			return err
		}
	}
	return nil
}

func (msg *MsgUpdateParams) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Authority)
	return []sdk.AccAddress{addr}
}

func (msg *MsgActivate) ValidateBasic() error {
	if msg.Authority == "" {
		return fmt.Errorf("authority cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(msg.Authority); err != nil {
		return fmt.Errorf("invalid authority address: %w", err)
	}
	return nil
}

func (msg *MsgActivate) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Authority)
	return []sdk.AccAddress{addr}
}

// --- Codec registration ---

func RegisterCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgUpdateParams{}, "alignment/MsgUpdateParams", nil)
	cdc.RegisterConcrete(&MsgActivate{}, "alignment/MsgActivate", nil)
}

func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgUpdateParams{},
		&MsgActivate{},
	)
}
