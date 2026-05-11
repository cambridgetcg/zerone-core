package keeper

import (
	"bytes"
	"context"
	"fmt"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/x/substrate_bridge/types"
)

type msgServer struct {
	Keeper
	types.UnimplementedMsgServer
}

// NewMsgServerImpl returns an implementation of types.MsgServer that wraps
// the keeper.
func NewMsgServerImpl(k Keeper) types.MsgServer {
	return &msgServer{Keeper: k}
}

func (m msgServer) RegisterAdapter(ctx context.Context, msg *types.MsgRegisterAdapter) (*types.MsgRegisterAdapterResponse, error) {
	if msg.Authority != m.authority {
		return nil, types.ErrAdapterAuthority
	}
	if msg.Adapter == nil {
		return nil, types.ErrAdapterNotFound
	}
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	msg.Adapter.RegisteredAtBlock = uint64(sdkCtx.BlockHeight())
	if err := m.WriteAdapter(ctx, msg.Adapter); err != nil {
		return nil, err
	}
	sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
		EventTypeAdapterRegistered,
		sdk.NewAttribute("adapter_id", msg.Adapter.AdapterId),
		sdk.NewAttribute("lip_id", msg.Adapter.RegisteredViaLipId),
		sdk.NewAttribute(AttrUsefulWorkCommitment, "UW"),
		sdk.NewAttribute(AttrMechanism, "M3"),
	))
	return &types.MsgRegisterAdapterResponse{}, nil
}

func (m msgServer) SuspendAdapter(ctx context.Context, msg *types.MsgSuspendAdapter) (*types.MsgSuspendAdapterResponse, error) {
	if msg.Authority != m.authority {
		return nil, types.ErrAdapterAuthority
	}
	if err := m.Keeper.SuspendAdapter(ctx, msg.AdapterId, msg.Reason); err != nil {
		return nil, err
	}
	sdk.UnwrapSDKContext(ctx).EventManager().EmitEvent(sdk.NewEvent(
		EventTypeAdapterSuspended,
		sdk.NewAttribute("adapter_id", msg.AdapterId),
		sdk.NewAttribute("reason", msg.Reason),
	))
	return &types.MsgSuspendAdapterResponse{}, nil
}

func (m msgServer) TombstoneAdapter(ctx context.Context, msg *types.MsgTombstoneAdapter) (*types.MsgTombstoneAdapterResponse, error) {
	if msg.Authority != m.authority {
		return nil, types.ErrAdapterAuthority
	}
	if err := m.Keeper.TombstoneAdapter(ctx, msg.AdapterId); err != nil {
		return nil, err
	}
	sdk.UnwrapSDKContext(ctx).EventManager().EmitEvent(sdk.NewEvent(
		EventTypeAdapterTombstoned,
		sdk.NewAttribute("adapter_id", msg.AdapterId),
	))
	return &types.MsgTombstoneAdapterResponse{}, nil
}

func (m msgServer) SubmitExternalAttestation(ctx context.Context, msg *types.MsgSubmitExternalAttestation) (*types.MsgSubmitExternalAttestationResponse, error) {
	if msg.Link == nil {
		return nil, types.ErrAdapterNotFound
	}
	params := m.GetParams(ctx)

	// 1. Verify link_hash matches recomputed canonical form (M2 re-derivability).
	computed := ComputeLinkHash(msg.Link)
	if !bytes.Equal(computed, msg.Link.LinkHash) {
		return nil, types.ErrLinkHashMismatch
	}

	// 2. Validate adapter + bounds + cited-fact existence + pending-claim cap.
	if err := m.ValidateLink(ctx, msg.Link, params); err != nil {
		return nil, err
	}

	// 3. Get adapter for qualification + work-class allow-list check.
	adapter, _ := m.GetAdapter(ctx, msg.AdapterId)
	if len(adapter.AllowedClassIds) > 0 {
		allowed := false
		for _, cid := range adapter.AllowedClassIds {
			if cid == msg.WorkClassId {
				allowed = true
				break
			}
		}
		if !allowed {
			return nil, types.ErrWorkClassNotAllowed
		}
	}

	// 4. Qualification check.
	if m.qualificationKeeper != nil && adapter.RequiredQualificationDomain != "" {
		qual, found := m.qualificationKeeper.GetDomainQualification(ctx, msg.Submitter, adapter.RequiredQualificationDomain)
		if !found || uint32(qual.Status) < uint32(adapter.MinQualificationStatus) {
			return nil, types.ErrInsufficientQualification
		}
	}

	// 5. Bond check: bond >= (min_attestation_bond + per_claim_bond × num_pending).
	bond, ok := sdkmath.NewIntFromString(msg.BondUzrn)
	if !ok {
		return nil, types.ErrInsufficientBond
	}
	minBond, _ := sdkmath.NewIntFromString(adapter.MinAttestationBondUzrn)
	if minBond.IsNil() {
		minBond, _ = sdkmath.NewIntFromString(params.AttestationMinBondUzrn)
	}
	perClaimMin, _ := sdkmath.NewIntFromString(adapter.MinPerClaimBondUzrn)
	if perClaimMin.IsNil() {
		perClaimMin, _ = sdkmath.NewIntFromString(params.PerPendingClaimBondUzrn)
	}
	totalMinBond := minBond.Add(perClaimMin.Mul(sdkmath.NewIntFromUint64(uint64(len(msg.Link.PendingClaims)))))
	if bond.LT(totalMinBond) {
		return nil, types.ErrInsufficientBond
	}

	// 6. Lock bond via SendCoinsFromAccountToModule.
	submitterAddr, err := sdk.AccAddressFromBech32(msg.Submitter)
	if err != nil {
		return nil, err
	}
	if m.bankKeeper != nil {
		coins := sdk.NewCoins(sdk.NewCoin("uzrn", bond))
		if err := m.bankKeeper.SendCoinsFromAccountToModule(ctx, submitterAddr, types.ModuleName, coins); err != nil {
			return nil, err
		}
	}

	// Create attestation record (state: COMMITTED initially).
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	attID := m.NextAttestationID(ctx)
	att := &types.ExternalAttestation{
		AttestationId:    attID,
		AdapterId:        msg.AdapterId,
		WorkClassId:      msg.WorkClassId,
		Submitter:        msg.Submitter,
		BondUzrn:         msg.BondUzrn,
		Link:             msg.Link,
		Status:           types.AttestationStatus_ATTESTATION_STATUS_COMMITTED,
		SubmittedAtBlock: uint64(sdkCtx.BlockHeight()),
		CommittedAtBlock: uint64(sdkCtx.BlockHeight()),
	}

	// 7. Auto-submit pending claims and link them.
	for _, pc := range msg.Link.PendingClaims {
		claimID := fmt.Sprintf("%s::pending::%s", attID, types.PendingClaimCanonicalHash(pc))
		// Knowledge keeper integration deferred (types differ); record the
		// reverse-index link so BeginBlocker can match claim resolutions.
		if m.knowledgeKeeper != nil {
			// Translation deferred; just record the link.
			_ = m.knowledgeKeeper
		}
		_ = m.LinkPendingClaim(ctx, claimID, attID)
	}

	// 8. State: AWAITING_RESOLUTION if pending claims exist; READY otherwise.
	if len(msg.Link.PendingClaims) > 0 {
		att.Status = types.AttestationStatus_ATTESTATION_STATUS_AWAITING_RESOLUTION
	} else {
		att.Status = types.AttestationStatus_ATTESTATION_STATUS_READY
	}

	if err := m.WriteAttestation(ctx, att); err != nil {
		return nil, err
	}

	// 9. Emit event with useful_work_commitment and mechanism tags.
	sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
		EventTypeExternalAttestationSubmitted,
		sdk.NewAttribute("attestation_id", attID),
		sdk.NewAttribute("adapter_id", msg.AdapterId),
		sdk.NewAttribute("work_class_id", msg.WorkClassId),
		sdk.NewAttribute("bond_uzrn", msg.BondUzrn),
		sdk.NewAttribute(AttrUsefulWorkCommitment, "UW"),
		sdk.NewAttribute(AttrMechanism, "M1,M2,M3"),
	))

	return &types.MsgSubmitExternalAttestationResponse{AttestationId: attID}, nil
}
