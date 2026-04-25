package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
)

// dialectic is read-only — no Msg types to register.
func RegisterCodec(_ *codec.LegacyAmino)              {}
func RegisterInterfaces(_ cdctypes.InterfaceRegistry) {}
