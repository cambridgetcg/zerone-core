package types

import (
	capabilitytypes "github.com/cosmos/ibc-go/modules/capability/types"
	icatypes "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// ICAControllerKeeper defines the expected ICA controller keeper interface.
type ICAControllerKeeper interface {
	RegisterInterchainAccount(ctx sdk.Context, connectionID, owner, version string) error
	GetInterchainAccountAddress(ctx sdk.Context, connectionID, portID string) (string, bool)
	GetOpenActiveChannel(ctx sdk.Context, connectionID, portID string) (string, bool)
	SendTx(ctx sdk.Context, chanCap *capabilitytypes.Capability, connectionID, portID string, icaPacketData icatypes.InterchainAccountPacketData, timeoutTimestamp uint64) (uint64, error)
}
