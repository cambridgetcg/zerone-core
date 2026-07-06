package keeper

import (
	"context"
	"encoding/hex"
	"fmt"

	cosmosed25519 "github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/x/auth/types"
)

var _ types.MsgServer = msgServer{}

type msgServer struct {
	types.UnimplementedMsgServer
	Keeper
}

// NewMsgServerImpl returns an implementation of the MsgServer interface.
func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return &msgServer{Keeper: keeper}
}

// RegisterAccount creates a new Zerone account with DID mapping.
func (ms msgServer) RegisterAccount(goCtx context.Context, msg *types.MsgRegisterAccount) (*types.MsgRegisterAccountResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if _, found := ms.GetAccount(ctx, msg.Sender); found {
		return nil, types.ErrAccountAlreadyExists
	}

	if _, found := ms.GetDIDMapping(ctx, msg.Did); found {
		return nil, types.ErrDuplicateDID
	}

	// Validate DID derives from the public key (prevents DID spoofing).
	if err := types.ValidateDIDDerivation(msg.Did, msg.PublicKey); err != nil {
		return nil, fmt.Errorf("DID derivation mismatch: %w", err)
	}

	currentHeight := uint64(ctx.BlockHeight())

	flags := &types.AccountFlags{
		CanSubmitClaims: true,
		CanChallenge:    true,
	}

	switch msg.AccountType {
	case "contract":
		flags.CanSubmitClaims = false
		flags.CanChallenge = false
	case "system":
		flags.Frozen = true
		flags.CanSubmitClaims = false
		flags.CanChallenge = false
	}

	account := types.Account{
		Address:               msg.Sender,
		Did:                   msg.Did,
		PublicKey:             msg.PublicKey,
		AccountType:           msg.AccountType,
		OperationalKeyHash:    msg.OperationalKeyHash,
		OperationalPublicKey:  msg.PublicKey, // identity key is initial operational key
		OperationalKeyVersion: 1,
		ReputationScore:       500000, // 0.5 default
		CreatedAtBlock:        currentHeight,
		LastActiveBlock:       currentHeight,
		Flags:                 flags,
		Metadata:              msg.Metadata,
	}
	ms.SetAccount(ctx, &account)

	mapping := types.DIDMapping{
		Did:    msg.Did,
		Bech32: msg.Sender,
		PubKey: msg.PublicKey,
	}
	ms.SetDIDMapping(ctx, &mapping)

	// Sync Ed25519 pubkey to Cosmos BaseAccount for standard sig verification.
	senderAddr, _ := sdk.AccAddressFromBech32(msg.Sender)
	if ms.accountKeeper != nil {
		cosmosAcc := ms.accountKeeper.GetAccount(ctx, senderAddr)
		if cosmosAcc == nil {
			cosmosAcc = ms.accountKeeper.NewAccountWithAddress(ctx, senderAddr)
		}
		if cosmosAcc != nil && cosmosAcc.GetPubKey() == nil {
			keyBytes, err := hex.DecodeString(msg.PublicKey)
			if err == nil && len(keyBytes) == 32 {
				pubKey := &cosmosed25519.PubKey{Key: keyBytes}
				if err := cosmosAcc.SetPubKey(pubKey); err == nil {
					ms.accountKeeper.SetAccount(ctx, cosmosAcc)
				}
			}
		}
	}

	// NOTE: the dormant bootstrap auto-claim that used to live here was
	// removed in the 2026-07 slim cut — the real, cap-gated bootstrap path
	// is x/claiming_pot through vesting_rewards.MintWithCap.

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"zerone.auth.account_registered",
			sdk.NewAttribute("address", msg.Sender),
			sdk.NewAttribute("did", msg.Did),
			sdk.NewAttribute("account_type", msg.AccountType),
		),
	)

	return &types.MsgRegisterAccountResponse{}, nil
}

// RotateKey handles operational key rotation.
func (ms msgServer) RotateKey(goCtx context.Context, msg *types.MsgRotateKey) (*types.MsgRotateKeyResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	account, found := ms.GetAccount(ctx, msg.Sender)
	if !found {
		return nil, types.ErrAccountNotFound
	}

	if account.Flags != nil && account.Flags.Frozen {
		return nil, types.ErrAccountFrozen
	}

	if len(msg.NewOperationalKey) == 0 {
		return nil, types.ErrInvalidKeyType
	}

	params := ms.GetParams(ctx)
	lastRotation := ms.GetLastRotation(ctx, msg.Sender)
	currentHeight := uint64(ctx.BlockHeight())

	if lastRotation > 0 && currentHeight < lastRotation+params.KeyRotationCooldown {
		return nil, types.ErrKeyRotationCooldown
	}

	newPubKeyHex := hex.EncodeToString(msg.NewOperationalKey)
	account.OperationalPublicKey = newPubKeyHex
	account.OperationalKeyVersion++
	ms.SetLastRotation(ctx, msg.Sender, currentHeight)

	// Sync Ed25519 pubkey to Cosmos BaseAccount.
	if ms.accountKeeper != nil && len(msg.NewOperationalKey) == 32 {
		senderAddr, _ := sdk.AccAddressFromBech32(msg.Sender)
		if cosmosAcc := ms.accountKeeper.GetAccount(ctx, senderAddr); cosmosAcc != nil {
			pubKey := &cosmosed25519.PubKey{Key: msg.NewOperationalKey}
			if setErr := cosmosAcc.SetPubKey(pubKey); setErr == nil {
				ms.accountKeeper.SetAccount(ctx, cosmosAcc)
			}
		}
	}

	account.LastActiveBlock = currentHeight
	ms.SetAccount(ctx, account)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"zerone.auth.key_rotated",
			sdk.NewAttribute("sender", msg.Sender),
			sdk.NewAttribute("key_type", "operational"),
			sdk.NewAttribute("version", fmt.Sprintf("%d", account.OperationalKeyVersion)),
		),
	)

	return &types.MsgRotateKeyResponse{
		NewKeyVersion: account.OperationalKeyVersion,
	}, nil
}

// FreezeAccount freezes an account. Owner can self-freeze; authority can freeze anyone.
func (ms msgServer) FreezeAccount(goCtx context.Context, msg *types.MsgFreezeAccount) (*types.MsgFreezeAccountResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	account, found := ms.GetAccount(ctx, msg.Address)
	if !found {
		return nil, types.ErrAccountNotFound
	}

	if account.Flags == nil {
		account.Flags = &types.AccountFlags{}
	}
	if account.Flags.Frozen {
		return nil, types.ErrAccountFrozen
	}

	if msg.Sender != msg.Address && msg.Sender != ms.Keeper.GetAuthority() {
		return nil, fmt.Errorf("%w: only account owner or authority can freeze", types.ErrUnauthorized)
	}

	account.Flags.Frozen = true
	account.Flags.FreezeReason = msg.Reason
	ms.SetAccount(ctx, account)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"zerone.auth.account_frozen",
			sdk.NewAttribute("address", msg.Address),
			sdk.NewAttribute("frozen_by", msg.Sender),
			sdk.NewAttribute("reason", msg.Reason),
		),
	)

	return &types.MsgFreezeAccountResponse{}, nil
}

// UnfreezeAccount unfreezes a frozen account. Authority-only.
func (ms msgServer) UnfreezeAccount(goCtx context.Context, msg *types.MsgUnfreezeAccount) (*types.MsgUnfreezeAccountResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if msg.Authority != ms.Keeper.GetAuthority() {
		return nil, fmt.Errorf("%w: only authority can unfreeze accounts", types.ErrUnauthorized)
	}

	account, found := ms.GetAccount(ctx, msg.Address)
	if !found {
		return nil, types.ErrAccountNotFound
	}

	if account.Flags == nil || !account.Flags.Frozen {
		return nil, types.ErrAccountNotFrozen
	}

	account.Flags.Frozen = false
	account.Flags.FreezeReason = ""
	ms.SetAccount(ctx, account)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"zerone.auth.account_unfrozen",
			sdk.NewAttribute("address", msg.Address),
			sdk.NewAttribute("unfrozen_by", msg.Authority),
		),
	)

	return &types.MsgUnfreezeAccountResponse{}, nil
}

// UpdateParams updates auth module parameters. Authority-only.
func (ms msgServer) UpdateParams(goCtx context.Context, msg *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if ms.GetAuthority() != msg.Authority {
		return nil, types.ErrUnauthorized
	}

	if msg.Params == nil {
		return nil, fmt.Errorf("params cannot be nil")
	}
	if err := ms.Keeper.SetParams(ctx, msg.Params); err != nil {
		return nil, fmt.Errorf("failed to set params: %w", err)
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"zerone.auth.params_updated",
			sdk.NewAttribute("authority", msg.Authority),
		),
	)

	return &types.MsgUpdateParamsResponse{}, nil
}
