package keeper

import (
	"context"
	"fmt"

	"github.com/cosmos/gogoproto/proto"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	icatypes "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/types"

	"github.com/zerone-chain/zerone/x/icaauth/types"
)

type msgServer struct {
	types.UnimplementedMsgServer
	Keeper
}

func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return &msgServer{Keeper: keeper}
}

func (ms *msgServer) RegisterAccount(goCtx context.Context, msg *types.MsgRegisterAccount) (*types.MsgRegisterAccountResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	params := ms.GetParams(ctx)

	// Check max accounts per owner
	existing := ms.GetRemoteAccounts(ctx, msg.Owner)
	if uint64(len(existing)) >= params.MaxRemoteAccountsPerOwner {
		return nil, types.ErrMaxAccountsReached.Wrapf(
			"owner %s has %d accounts, max is %d",
			msg.Owner, len(existing), params.MaxRemoteAccountsPerOwner,
		)
	}

	// Check for duplicate connection
	if _, found := ms.GetRemoteAccountByConnection(ctx, msg.Owner, msg.ConnectionId); found {
		return nil, types.ErrAlreadyRegistered.Wrapf(
			"owner %s already has an account on connection %s",
			msg.Owner, msg.ConnectionId,
		)
	}

	// Check registration cooldown
	currentHeight := uint64(ctx.BlockHeight())
	if params.RegistrationCooldown > 0 && len(existing) > 0 {
		lastRegistered := uint64(0)
		for _, acct := range existing {
			if acct.RegisteredBlock > lastRegistered {
				lastRegistered = acct.RegisteredBlock
			}
		}
		if currentHeight < lastRegistered+params.RegistrationCooldown {
			return nil, types.ErrRegistrationCooldown.Wrapf(
				"must wait until block %d (current: %d)",
				lastRegistered+params.RegistrationCooldown, currentHeight,
			)
		}
	}

	// Register with ICA controller
	if err := ms.icaController.RegisterInterchainAccount(ctx, msg.ConnectionId, msg.Owner, ""); err != nil {
		return nil, fmt.Errorf("failed to register interchain account: %w", err)
	}

	// Compute port ID for the owner
	portID := icatypes.ControllerPortPrefix + msg.Owner

	// Store remote account record
	acct := &types.RemoteAccount{
		ConnectionId: msg.ConnectionId,
		PortId:       portID,
		OwnerAddress: msg.Owner,
		RegisteredBlock: currentHeight,
		Active:       false, // becomes active after ICA callback sets address
	}
	ms.AddRemoteAccount(ctx, msg.Owner, acct)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent("zerone.icaauth.account_registered",
			sdk.NewAttribute("owner", msg.Owner),
			sdk.NewAttribute("connection_id", msg.ConnectionId),
			sdk.NewAttribute("port_id", portID),
		),
	)

	return &types.MsgRegisterAccountResponse{}, nil
}

func (ms *msgServer) SubmitTx(goCtx context.Context, msg *types.MsgSubmitTx) (*types.MsgSubmitTxResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	params := ms.GetParams(ctx)

	// Check registration
	acct, found := ms.GetRemoteAccountByConnection(ctx, msg.Owner, msg.ConnectionId)
	if !found {
		return nil, types.ErrNotRegistered.Wrapf(
			"owner %s has no account on connection %s",
			msg.Owner, msg.ConnectionId,
		)
	}

	// Check active status
	if !acct.Active {
		return nil, fmt.Errorf("interchain account on connection %s is not active", msg.ConnectionId)
	}

	// Check channel liveness — if channel is closed, mark inactive and reject
	_, channelOpen := ms.icaController.GetOpenActiveChannel(ctx, msg.ConnectionId, acct.PortId)
	if !channelOpen {
		ms.markAccountInactive(ctx, msg.Owner, msg.ConnectionId)
		return nil, fmt.Errorf("ICA channel for connection %s is not open; account marked inactive", msg.ConnectionId)
	}

	// Enforce max_messages_per_tx
	if uint64(len(msg.Msgs)) > params.MaxMessagesPerTx {
		return nil, types.ErrMaxMessagesExceeded.Wrapf(
			"tx contains %d messages, max is %d",
			len(msg.Msgs), params.MaxMessagesPerTx,
		)
	}

	// Build allowlist lookup from global params
	allowedGlobal := make(map[string]bool, len(params.AllowedHostMsgTypes))
	for _, msgType := range params.AllowedHostMsgTypes {
		allowedGlobal[msgType] = true
	}

	// Build per-account allowlist (if set, it's a further restriction)
	hasPerAccountList := len(acct.AllowedMsgTypes) > 0
	allowedPerAccount := make(map[string]bool, len(acct.AllowedMsgTypes))
	for _, msgType := range acct.AllowedMsgTypes {
		allowedPerAccount[msgType] = true
	}

	// SECURITY: Validate ALL message type URLs against both allowlists.
	// Unknown message types are REJECTED, not silently allowed (P0 fix).
	protoMsgs := make([]proto.Message, 0, len(msg.Msgs))
	for i, anyMsg := range msg.Msgs {
		typeURL := anyMsg.GetTypeUrl()

		// Check global allowlist first
		if !allowedGlobal[typeURL] {
			return nil, types.ErrMsgTypeNotAllowed.Wrapf(
				"msg[%d] type %q not in global allowlist",
				i, typeURL,
			)
		}

		// Check per-account allowlist (if configured)
		if hasPerAccountList && !allowedPerAccount[typeURL] {
			return nil, types.ErrMsgTypeNotAllowed.Wrapf(
				"msg[%d] type %q not in account allowlist",
				i, typeURL,
			)
		}

		// Unpack the Any into a concrete SDK message via the codec's interface registry
		sdkAny := &codectypes.Any{
			TypeUrl: anyMsg.GetTypeUrl(),
			Value:   anyMsg.GetValue(),
		}
		var sdkMsg sdk.Msg
		if err := ms.cdc.UnpackAny(sdkAny, &sdkMsg); err != nil {
			return nil, fmt.Errorf("msg[%d]: failed to unpack: %w", i, err)
		}
		protoMsgs = append(protoMsgs, sdkMsg)
	}

	data, err := icatypes.SerializeCosmosTx(ms.cdc, protoMsgs, icatypes.EncodingProtobuf)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize cosmos tx: %w", err)
	}

	packetData := icatypes.InterchainAccountPacketData{
		Type: icatypes.EXECUTE_TX,
		Data: data,
	}

	_, err = ms.icaController.SendTx(ctx, nil, msg.ConnectionId, acct.PortId, packetData, msg.TimeoutNs)
	if err != nil {
		return nil, fmt.Errorf("failed to send ICA tx: %w", err)
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent("zerone.icaauth.tx_submitted",
			sdk.NewAttribute("owner", msg.Owner),
			sdk.NewAttribute("connection_id", msg.ConnectionId),
			sdk.NewAttribute("msg_count", fmt.Sprintf("%d", len(msg.Msgs))),
		),
	)

	return &types.MsgSubmitTxResponse{}, nil
}

func (ms *msgServer) UpdateParams(goCtx context.Context, msg *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	if msg.Authority != ms.GetAuthority() {
		return nil, fmt.Errorf("unauthorized: expected %s, got %s", ms.GetAuthority(), msg.Authority)
	}
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	ms.SetParams(goCtx, msg.Params)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent("zerone.icaauth.params_updated",
			sdk.NewAttribute("authority", msg.Authority),
		),
	)

	return &types.MsgUpdateParamsResponse{}, nil
}

// markAccountInactive sets a remote account to inactive.
func (ms *msgServer) markAccountInactive(ctx sdk.Context, owner, connectionID string) {
	rec, found := ms.GetRecord(ctx, owner)
	if !found {
		return
	}
	for _, acct := range rec.Accounts {
		if acct.ConnectionId == connectionID {
			acct.Active = false
			break
		}
	}
	ms.SetRecord(ctx, rec)
}
