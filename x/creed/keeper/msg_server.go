package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/x/creed/types"
)

type msgServer struct {
	types.UnimplementedMsgServer
	keeper Keeper
}

func NewMsgServerImpl(k Keeper) types.MsgServer { return &msgServer{keeper: k} }

var _ types.MsgServer = &msgServer{}

// AnchorPin records a new PinnedCreed at version+1. The handler
// enforces the structural invariants commitments 6 and 10 require.
func (m *msgServer) AnchorPin(ctx context.Context, msg *types.MsgAnchorPin) (*types.MsgAnchorPinResponse, error) {
	if msg == nil || msg.Pin == nil {
		return nil, fmt.Errorf("pin required")
	}

	if msg.Authority != m.keeper.GetAuthority() {
		return nil, types.ErrUnauthorized.Wrapf("expected %s, got %s", m.keeper.GetAuthority(), msg.Authority)
	}

	params := m.keeper.GetParams(ctx)
	if !params.DirectAnchorEnabled {
		// When direct anchoring is disabled, this public handler remains sealed.
		// SourceLip is provenance metadata, not proof that governance passed; an
		// actual governance dispatch must use a separately configured path.
		if msg.SourceLip == "" {
			return nil, types.ErrSourceLIPRequired.Wrap("commitment 6: amendment must cite the LIP that authorized it")
		}
		return nil, types.ErrDirectAnchorDisabled.Wrap("commitment 6: direct creed anchoring is disabled; use the separately configured gov-dispatch path")
	}

	pin := msg.Pin

	// Stamp the height and pin.
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	pin.PinnedAtHeight = uint64(sdkCtx.BlockHeight())
	if msg.SourceLip != "" {
		pin.PinnedViaLip = msg.SourceLip
	}
	if err := m.keeper.AnchorPin(ctx, pin); err != nil {
		return nil, err
	}

	// Voice layer: announce the amendment so off-chain observers
	// can compose creed-drift dashboards in the same vocabulary
	// the creed itself uses.
	hashAttr := fmt.Sprintf("%x", pin.CanonicalHash)
	sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
		"zerone.creed.pinned",
		sdk.NewAttribute("version", fmt.Sprintf("%d", pin.Version)),
		sdk.NewAttribute("canonical_hash", hashAttr),
		sdk.NewAttribute("source_lip", pin.PinnedViaLip),
		sdk.NewAttribute("commitment_count", fmt.Sprintf("%d", len(pin.Commitments))),
		sdk.NewAttribute("creed_commitment", "6,10,19"),
	))

	return &types.MsgAnchorPinResponse{NewVersion: pin.Version}, nil
}

// UpdateCouncilMember adds, updates, or deactivates a Creed Council
// registry entry. It is authority-gated. source_lip is required when direct
// anchoring is disabled, but this handler does not independently verify LIP
// passage; current LIP tally does not consume the council registry.
func (m *msgServer) UpdateCouncilMember(ctx context.Context, msg *types.MsgUpdateCouncilMember) (*types.MsgUpdateCouncilMemberResponse, error) {
	if msg == nil || msg.Member == nil {
		return nil, types.ErrInvalidCouncilMember.Wrap("member required")
	}
	if msg.Authority != m.keeper.GetAuthority() {
		return nil, types.ErrUnauthorized.Wrapf("expected %s, got %s", m.keeper.GetAuthority(), msg.Authority)
	}

	params := m.keeper.GetParams(ctx)
	if !params.DirectAnchorEnabled && msg.SourceLip == "" {
		return nil, types.ErrSourceLIPRequired.Wrap("commitment 19: post-disable, council changes must cite the LIP that authorized them")
	}

	if msg.Member.Address == "" {
		return nil, types.ErrInvalidCouncilMember.Wrap("address required")
	}
	if msg.Member.VotingWeightBps > 1_000_000 {
		return nil, types.ErrInvalidCouncilMember.Wrap("voting_weight_bps must be ≤ 1_000_000")
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	height := uint64(sdkCtx.BlockHeight())

	// Preserve admitted_at_height for existing seats unless caller
	// explicitly bumps it. Stamp it for new ones.
	if existing, ok := m.keeper.GetCouncilMember(ctx, msg.Member.Address); ok {
		if msg.Member.AdmittedAtHeight == 0 {
			msg.Member.AdmittedAtHeight = existing.AdmittedAtHeight
		}
	} else {
		if msg.Member.AdmittedAtHeight == 0 {
			msg.Member.AdmittedAtHeight = height
		}
	}
	if msg.SourceLip != "" {
		msg.Member.AdmittedViaLip = msg.SourceLip
	}

	if err := m.keeper.SetCouncilMember(ctx, msg.Member); err != nil {
		return nil, err
	}

	// Voice layer: announce membership change so off-chain
	// observers can compose council-roster dashboards.
	sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
		"zerone.creed.council_member_updated",
		sdk.NewAttribute("address", msg.Member.Address),
		sdk.NewAttribute("active", fmt.Sprintf("%t", msg.Member.Active)),
		sdk.NewAttribute("voting_weight_bps", fmt.Sprintf("%d", msg.Member.VotingWeightBps)),
		sdk.NewAttribute("source_lip", msg.Member.AdmittedViaLip),
		sdk.NewAttribute("creed_commitment", "19"),
	))

	return &types.MsgUpdateCouncilMemberResponse{}, nil
}

func (m *msgServer) UpdateParams(ctx context.Context, msg *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	if msg.Authority != m.keeper.GetAuthority() {
		return nil, types.ErrUnauthorized.Wrapf("expected %s, got %s", m.keeper.GetAuthority(), msg.Authority)
	}
	if msg.Params == nil {
		return nil, types.ErrInvalidParams.Wrap("params required")
	}
	current := m.keeper.GetParams(ctx)
	if msg.Params.Authority != current.Authority {
		return nil, types.ErrInvalidParams.Wrap("params.authority is compatibility-only and runtime-immutable")
	}
	if err := m.keeper.SetParams(ctx, msg.Params); err != nil {
		return nil, err
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
		"zerone.creed.params_updated",
		sdk.NewAttribute("authority", msg.Authority),
		sdk.NewAttribute("direct_anchor_enabled", fmt.Sprintf("%t", msg.Params.DirectAnchorEnabled)),
	))

	return &types.MsgUpdateParamsResponse{}, nil
}
