package keeper

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"math"
	"time"

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
	if msg == nil {
		return nil, fmt.Errorf("registration message cannot be nil")
	}
	if err := msg.ValidateBasic(); err != nil {
		return nil, fmt.Errorf("invalid registration: %w", err)
	}
	if ctx.BlockHeight() <= 0 {
		return nil, fmt.Errorf("invalid consensus block height %d", ctx.BlockHeight())
	}
	params := ms.GetParams(ctx)
	if uint64(len(msg.Metadata)) > uint64(params.MaxMetadataLength) {
		return nil, fmt.Errorf("metadata exceeds maximum length of %d bytes", params.MaxMetadataLength)
	}

	identityKey, err := types.DecodeEd25519PublicKeyHex(msg.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", types.ErrInvalidPublicKey, err)
	}
	proofBytes, err := types.AccountRegistrationProofSignBytes(
		ctx.ChainID(),
		msg.Sender,
		msg.Did,
		identityKey,
		msg.AccountType,
		msg.Metadata,
	)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", types.ErrInvalidIdentityProof, err)
	}
	if err := types.VerifyEd25519Signature(identityKey, proofBytes, msg.IdentityProofSignature); err != nil {
		return nil, fmt.Errorf("%w: %v", types.ErrInvalidIdentityProof, err)
	}

	if _, found := ms.GetAccount(ctx, msg.Sender); found {
		return nil, types.ErrAccountAlreadyExists
	}

	if _, found := ms.GetDIDMapping(ctx, msg.Did); found {
		return nil, types.ErrDuplicateDID
	}

	operationalKeyHash, err := types.OperationalKeyHash(identityKey)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", types.ErrInvalidPublicKey, err)
	}
	if msg.OperationalKeyHash != "" && msg.OperationalKeyHash != operationalKeyHash {
		return nil, fmt.Errorf("%w: operational_key_hash does not match public_key", types.ErrInvalidPublicKey)
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
		OperationalKeyHash:    operationalKeyHash,
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
	if msg == nil {
		return nil, fmt.Errorf("key-rotation message cannot be nil")
	}
	if err := msg.ValidateBasic(); err != nil {
		return nil, fmt.Errorf("invalid key rotation: %w", err)
	}
	if ctx.BlockHeight() <= 0 {
		return nil, fmt.Errorf("invalid consensus block height %d", ctx.BlockHeight())
	}

	account, found := ms.GetAccount(ctx, msg.Sender)
	if !found {
		return nil, types.ErrAccountNotFound
	}

	if account.Flags != nil && account.Flags.Frozen {
		return nil, types.ErrAccountFrozen
	}

	if account.OperationalKeyVersion == 0 || account.OperationalKeyVersion == math.MaxUint32 {
		return nil, fmt.Errorf("%w: current operational key version is invalid", types.ErrInvalidPublicKey)
	}
	currentOperationalKey, err := types.DecodeEd25519PublicKeyHex(account.OperationalPublicKey)
	if err != nil {
		return nil, fmt.Errorf("%w: stored operational key is invalid: %v", types.ErrInvalidPublicKey, err)
	}
	if bytes.Equal(currentOperationalKey, msg.NewOperationalKey) {
		return nil, fmt.Errorf("%w: new operational key must differ from the current key", types.ErrInvalidKeyType)
	}

	blockTime := ctx.BlockTime().UTC()
	if blockTime.IsZero() {
		return nil, fmt.Errorf("%w: consensus block time is unavailable", types.ErrInvalidAuthorizationSig)
	}
	expiresAt := time.Unix(msg.AuthorizationExpiresAtUnix, 0).UTC()
	if !expiresAt.After(blockTime) {
		return nil, types.ErrKeyAuthorizationExpired
	}
	if expiresAt.After(blockTime.Add(types.KeyRotationAuthorizationMaxTTL)) {
		return nil, types.ErrKeyAuthorizationTooFar
	}
	signBytes, err := types.KeyRotationAuthorizationSignBytes(
		ctx.ChainID(),
		msg.Sender,
		account.OperationalKeyVersion,
		msg.NewOperationalKey,
		msg.AuthorizationExpiresAtUnix,
	)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", types.ErrInvalidAuthorizationSig, err)
	}
	if err := types.VerifyEd25519Signature(currentOperationalKey, signBytes, msg.AuthorizationSignature); err != nil {
		return nil, fmt.Errorf("%w: %v", types.ErrInvalidAuthorizationSig, err)
	}
	acceptanceBytes, err := types.KeyRotationAcceptanceSignBytes(
		ctx.ChainID(),
		msg.Sender,
		account.OperationalKeyVersion,
		msg.NewOperationalKey,
		msg.AuthorizationExpiresAtUnix,
	)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", types.ErrInvalidNewKeyProof, err)
	}
	if err := types.VerifyEd25519Signature(
		msg.NewOperationalKey,
		acceptanceBytes,
		msg.NewKeyConfirmationSignature,
	); err != nil {
		return nil, fmt.Errorf("%w: %v", types.ErrInvalidNewKeyProof, err)
	}

	params := ms.GetParams(ctx)
	lastRotation := ms.GetLastRotation(ctx, msg.Sender)
	currentHeight := uint64(ctx.BlockHeight())

	if lastRotation > 0 {
		if currentHeight < lastRotation || currentHeight-lastRotation < params.KeyRotationCooldown {
			return nil, types.ErrKeyRotationCooldown
		}
	}

	newPubKeyHex := hex.EncodeToString(msg.NewOperationalKey)
	account.OperationalPublicKey = newPubKeyHex
	account.OperationalKeyHash, err = types.OperationalKeyHash(msg.NewOperationalKey)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", types.ErrInvalidPublicKey, err)
	}
	account.OperationalKeyVersion++
	ms.SetLastRotation(ctx, msg.Sender, currentHeight)

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
	if msg == nil {
		return nil, fmt.Errorf("freeze message cannot be nil")
	}
	if err := msg.ValidateBasic(); err != nil {
		return nil, fmt.Errorf("invalid freeze request: %w", err)
	}

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
	if msg == nil {
		return nil, fmt.Errorf("unfreeze message cannot be nil")
	}
	if err := msg.ValidateBasic(); err != nil {
		return nil, fmt.Errorf("invalid unfreeze request: %w", err)
	}

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
	if msg == nil {
		return nil, fmt.Errorf("update-params message cannot be nil")
	}
	if err := msg.ValidateBasic(); err != nil {
		return nil, fmt.Errorf("invalid update-params request: %w", err)
	}

	if ms.GetAuthority() != msg.Authority {
		return nil, types.ErrUnauthorized
	}

	if msg.Params == nil {
		return nil, fmt.Errorf("params cannot be nil")
	}
	var oversizedAddress string
	ms.IterateAccounts(ctx, func(account *types.Account) bool {
		if uint64(len(account.Metadata)) > uint64(msg.Params.MaxMetadataLength) {
			oversizedAddress = account.Address
			return true
		}
		return false
	})
	if oversizedAddress != "" {
		return nil, fmt.Errorf("max_metadata_length would invalidate account %s", oversizedAddress)
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
