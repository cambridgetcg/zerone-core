package keeper

import (
	"bytes"
	"context"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/x/substrate_bridge/types"
)

// These handler bodies are preserved verbatim from msg_server.go at
// 77521a0daf14420836483ae9d8ea3fbcc1ae0d40 (the predecessor of PR #126),
// with only method names changed. Do not route them through the repaired
// qualification helper. Historical comments are not current activation advice:
// no required-domain adapter should be registered before consensus-repairs-v1.
func (m msgServer) registerAdapterLegacy(ctx context.Context, msg *types.MsgRegisterAdapter) (*types.MsgRegisterAdapterResponse, error) {
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

func (m msgServer) submitExternalAttestationLegacy(ctx context.Context, msg *types.MsgSubmitExternalAttestation) (*types.MsgSubmitExternalAttestationResponse, error) {
	// Fail closed unless the dedupe index is seeded — otherwise the
	// uniqueness checks below would run against an empty index and a replay
	// would look free. A fresh chain (no attestations to seed) self-arms
	// here; an existing chain whose substrate-dedupe-v1 handler has not run
	// is refused. See keeper/source_ref.go:ensureDedupeArmed.
	if err := m.ensureDedupeArmed(ctx); err != nil {
		return nil, err
	}
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

	// 2b. Pending claims are fail-closed until their translation into
	// x/knowledge is implemented. Accepting a bond before canonical knowledge
	// provenance exists would trap the attestation until timeout and slashing.
	if len(msg.Link.PendingClaims) > 0 {
		return nil, types.ErrPendingClaimsNotSupported
	}

	// 2c. One submission, one adapter: the allow-list, qualification and
	// bond checks below read msg.AdapterId while link validation reads
	// link.adapter_id — they gate the same submission only when equal.
	if msg.AdapterId != msg.Link.AdapterId {
		return nil, types.ErrAdapterIdMismatch
	}

	// 2d. The source reference is the DECLARED economic identity of the work,
	// and each declared (adapter_id, source_id) is attestable at most once.
	// This is the chain-side replay wall: without it the same settled work
	// could be re-submitted (fetched_at_block or an axis weight varied yields
	// a fresh link_hash) and mint again at every settlement. Rejection
	// releases the source (see settleRejected); a minted holder keeps it
	// forever, and settlement mints only for the ref-holder (see
	// authorizeSourceMint) so a duplicate reaching settle mints nothing.
	//
	// Scope: the guarantee is over the DECLARED identity. The chain does not
	// verify that source_id faithfully names distinct work (no oracle over the
	// adapter's off-chain source), so an adversary who can submit at all may
	// relabel the same work under a fresh source_id and mint again — the same
	// residual class as fabricating sources or front-running, bounded by the
	// adapter's qualification gate (RequiredQualificationDomain), not by this
	// wall. Open adapters (e.g. agenttool-invocation-v1) should set one.
	if msg.Link.Source == nil || msg.Link.Source.SourceId == "" {
		return nil, types.ErrSourceRequired
	}
	if holder, taken := m.GetSourceRef(ctx, msg.Link.AdapterId, msg.Link.Source.SourceId); taken {
		return nil, types.ErrDuplicateSource.Wrapf("source %q held by %s", msg.Link.Source.SourceId, holder)
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
		if !found || qual == nil || uint32(qual.Status) < uint32(adapter.MinQualificationStatus) {
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

	// Create a READY attestation; pending-claim submissions were rejected above.
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	attID := m.NextAttestationID(ctx)
	att := &types.ExternalAttestation{
		AttestationId:    attID,
		AdapterId:        msg.AdapterId,
		WorkClassId:      msg.WorkClassId,
		Submitter:        msg.Submitter,
		BondUzrn:         msg.BondUzrn,
		Link:             msg.Link,
		Status:           types.AttestationStatus_ATTESTATION_STATUS_READY,
		SubmittedAtBlock: uint64(sdkCtx.BlockHeight()),
		CommittedAtBlock: uint64(sdkCtx.BlockHeight()),
	}

	if err := m.WriteAttestation(ctx, att); err != nil {
		return nil, err
	}
	m.SetSourceRef(ctx, msg.Link.AdapterId, msg.Link.Source.SourceId, attID)

	// 7. Emit event with useful_work_commitment and mechanism tags.
	sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
		EventTypeExternalAttestationSubmitted,
		sdk.NewAttribute("attestation_id", attID),
		sdk.NewAttribute("adapter_id", msg.AdapterId),
		sdk.NewAttribute("work_class_id", msg.WorkClassId),
		sdk.NewAttribute("source_id", msg.Link.Source.SourceId),
		sdk.NewAttribute("bond_uzrn", msg.BondUzrn),
		sdk.NewAttribute(AttrUsefulWorkCommitment, "UW"),
		sdk.NewAttribute(AttrMechanism, "M1,M2,M3"),
	))

	return &types.MsgSubmitExternalAttestationResponse{AttestationId: attID}, nil
}
