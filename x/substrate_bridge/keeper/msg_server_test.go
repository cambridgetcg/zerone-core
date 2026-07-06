package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/x/substrate_bridge/keeper"
	"github.com/zerone-chain/zerone/x/substrate_bridge/types"
)

func TestMsgServer_RegisterAdapter_RequiresAuthority(t *testing.T) {
	k, ctx := setupSubstrateBridgeKeeper(t)
	srv := keeper.NewMsgServerImpl(k)

	_, err := srv.RegisterAdapter(sdk.WrapSDKContext(ctx), &types.MsgRegisterAdapter{
		Authority: "wrong-authority",
		Adapter:   &types.AdapterRegistration{AdapterId: "wiki-v1"},
	})
	require.ErrorIs(t, err, types.ErrAdapterAuthority)
}

func TestMsgServer_RegisterAdapter_Happy(t *testing.T) {
	k, ctx := setupSubstrateBridgeKeeper(t)
	srv := keeper.NewMsgServerImpl(k)
	_, err := srv.RegisterAdapter(sdk.WrapSDKContext(ctx), &types.MsgRegisterAdapter{
		Authority: k.Authority(),
		Adapter: &types.AdapterRegistration{
			AdapterId: "wiki-v1",
			Status:    types.AdapterStatus_ADAPTER_STATUS_ACTIVE,
		},
	})
	require.NoError(t, err)
	_, found := k.GetAdapter(ctx, "wiki-v1")
	require.True(t, found)
}

func TestMsgServer_SubmitExternalAttestation_LinkHashEnforced(t *testing.T) {
	k, ctx := setupSubstrateBridgeKeeperWithKnowledge(t)
	require.NoError(t, k.WriteAdapter(ctx, &types.AdapterRegistration{
		AdapterId: "wiki-v1", Status: types.AdapterStatus_ADAPTER_STATUS_ACTIVE,
	}))

	link := &types.SubstrateLink{AdapterId: "wiki-v1"}
	link.LinkHash = []byte{0x01, 0x02, 0x03} // intentionally wrong

	srv := keeper.NewMsgServerImpl(k)
	_, err := srv.SubmitExternalAttestation(sdk.WrapSDKContext(ctx), &types.MsgSubmitExternalAttestation{
		Submitter:   "zrn1sub",
		AdapterId:   "wiki-v1",
		WorkClassId: "translation",
		Link:        link,
		BondUzrn:    "222000",
	})
	require.ErrorIs(t, err, types.ErrLinkHashMismatch)
}

// TestMsgServer_SubmitExternalAttestation_PendingClaimsFailClosed pins the
// fail-closed door: until pending-claim translation into x/knowledge is
// wired (ToK Plan 4), a link carrying pending claims is refused at submit —
// accepting it would strand the bond in an unresolvable AWAITING state and
// slash it on timeout.
func TestMsgServer_SubmitExternalAttestation_PendingClaimsFailClosed(t *testing.T) {
	k, ctx := setupSubstrateBridgeKeeperWithKnowledge(t)
	require.NoError(t, k.WriteAdapter(ctx, &types.AdapterRegistration{
		AdapterId: "wiki-v1", Status: types.AdapterStatus_ADAPTER_STATUS_ACTIVE,
	}))

	link := &types.SubstrateLink{
		AdapterId:     "wiki-v1",
		PendingClaims: []*types.PendingClaim{{ClaimContent: "the sky is blue", Domain: "physics"}},
	}
	link.LinkHash = keeper.ComputeLinkHash(link)

	srv := keeper.NewMsgServerImpl(k)
	_, err := srv.SubmitExternalAttestation(sdk.WrapSDKContext(ctx), &types.MsgSubmitExternalAttestation{
		Submitter:   "zrn1sub",
		AdapterId:   "wiki-v1",
		WorkClassId: "translation",
		Link:        link,
		BondUzrn:    "2000000",
	})
	require.ErrorIs(t, err, types.ErrPendingClaimsNotSupported)
}
