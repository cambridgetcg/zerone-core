package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// RegisterCodec registers the toolbox module types with the legacy amino codec.
func RegisterCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgRegisterTool{}, "zerone_toolbox/RegisterTool", nil)
	cdc.RegisterConcrete(&MsgCallTool{}, "zerone_toolbox/CallTool", nil)
	cdc.RegisterConcrete(&MsgAddContributor{}, "zerone_toolbox/AddContributor", nil)
	cdc.RegisterConcrete(&MsgAcceptContributorship{}, "zerone_toolbox/AcceptContributorship", nil)
	cdc.RegisterConcrete(&MsgUpgradeTool{}, "zerone_toolbox/UpgradeTool", nil)
	cdc.RegisterConcrete(&MsgDeprecateTool{}, "zerone_toolbox/DeprecateTool", nil)
	cdc.RegisterConcrete(&MsgRetireTool{}, "zerone_toolbox/RetireTool", nil)
	cdc.RegisterConcrete(&MsgLockShares{}, "zerone_toolbox/LockShares", nil)
	cdc.RegisterConcrete(&MsgUpdateDependency{}, "zerone_toolbox/UpdateDependency", nil)
	cdc.RegisterConcrete(&MsgToolHeartbeat{}, "zerone_toolbox/ToolHeartbeat", nil)
	cdc.RegisterConcrete(&MsgUpdateParams{}, "zerone_toolbox/UpdateParams", nil)
}

// RegisterInterfaces registers the toolbox module interface implementations.
func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgRegisterTool{},
		&MsgCallTool{},
		&MsgAddContributor{},
		&MsgAcceptContributorship{},
		&MsgUpgradeTool{},
		&MsgDeprecateTool{},
		&MsgRetireTool{},
		&MsgLockShares{},
		&MsgUpdateDependency{},
		&MsgToolHeartbeat{},
		&MsgUpdateParams{},
	)
}
