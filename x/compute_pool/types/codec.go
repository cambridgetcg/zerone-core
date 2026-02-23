package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// RegisterCodec registers the compute_pool module types with the legacy amino codec.
func RegisterCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgRegisterProvider{}, "zerone_compute_pool/RegisterProvider", nil)
	cdc.RegisterConcrete(&MsgUnregisterProvider{}, "zerone_compute_pool/UnregisterProvider", nil)
	cdc.RegisterConcrete(&MsgHeartbeat{}, "zerone_compute_pool/Heartbeat", nil)
	cdc.RegisterConcrete(&MsgUpdatePrice{}, "zerone_compute_pool/UpdatePrice", nil)
	cdc.RegisterConcrete(&MsgRedeemCredits{}, "zerone_compute_pool/RedeemCredits", nil)
	cdc.RegisterConcrete(&MsgUpdateParams{}, "zerone_compute_pool/UpdateParams", nil)
}

// RegisterInterfaces registers the compute_pool module interface implementations.
func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgRegisterProvider{},
		&MsgUnregisterProvider{},
		&MsgHeartbeat{},
		&MsgUpdatePrice{},
		&MsgRedeemCredits{},
		&MsgUpdateParams{},
	)
}
