package keeper

import (
	"context"

	"github.com/zerone-chain/zerone/x/substrate_bridge/types"
)

// NewLegacyQualificationMsgServerForTest exposes the frozen predecessor only to
// external-package tests. Production callers can use only the gated server.
func NewLegacyQualificationMsgServerForTest(k Keeper) types.MsgServer {
	return legacyQualificationMsgServer{msgServer{Keeper: k}}
}

type legacyQualificationMsgServer struct{ msgServer }

func (m legacyQualificationMsgServer) RegisterAdapter(ctx context.Context, msg *types.MsgRegisterAdapter) (*types.MsgRegisterAdapterResponse, error) {
	return m.registerAdapterLegacy(ctx, msg)
}

func (m legacyQualificationMsgServer) SubmitExternalAttestation(ctx context.Context, msg *types.MsgSubmitExternalAttestation) (*types.MsgSubmitExternalAttestationResponse, error) {
	return m.submitExternalAttestationLegacy(ctx, msg)
}
