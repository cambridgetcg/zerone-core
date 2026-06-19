package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// RegisterCodec registers the contagion module's concrete message types with
// the legacy amino codec. Names follow the zerone_<module>/<MsgName> convention
// used by x/tokens.
func RegisterCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgSneeze{}, "zerone_contagion/Sneeze", nil)
	cdc.RegisterConcrete(&MsgSetContagion{}, "zerone_contagion/SetContagion", nil)
}

// RegisterInterfaces registers the contagion module's message implementations
// against sdk.Msg so they are recognised by the interface registry.
func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgSneeze{},
		&MsgSetContagion{},
	)
}

var (
	Amino     = codec.NewLegacyAmino()
	ModuleCdc = codec.NewProtoCodec(cdctypes.NewInterfaceRegistry())
)
