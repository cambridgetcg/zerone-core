package app

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	zeroneauthtypes "github.com/zerone-chain/zerone/x/auth/types"
	knowledge "github.com/zerone-chain/zerone/x/knowledge/types"
)

func TestFactUsePaidGasAndCapabilityClassification(t *testing.T) {
	for _, msg := range []sdk.Msg{&knowledge.MsgReportFactUse{}, &knowledge.MsgRateFact{}} {
		t.Run(sdk.MsgTypeURL(msg), func(t *testing.T) {
			typeURL := sdk.MsgTypeURL(msg)
			cost, mapped := msgTypeURLToGas[typeURL]
			require.True(t, mapped, "feedback must not use unknown-message fallback")
			require.Positive(t, cost)
			require.Equal(t, cost, lookupMsgGas(typeURL))
			require.False(t, BootstrapGasFreeTypes[typeURL])
			require.True(t, isFactFeedbackMsg(typeURL))
			require.True(t, isZeroneSpecificMsg(typeURL))
			require.False(t, isClaimMsg(typeURL))

			ak, _, ctx := setupBothKeepers(t)
			decorator := NewZeroneCapabilityDecorator(ak)
			addr := sdk.AccAddress(make([]byte, 20)).String()
			require.ErrorIs(t, decorator.checkAccountCapability(ctx, addr, msg), zeroneauthtypes.ErrAccountCapabilityDenied)
			registerZeroneAccountWithType(t, ak, ctx, addr, "contract", &zeroneauthtypes.AccountFlags{})
			require.NoError(t, decorator.checkAccountCapability(ctx, addr, msg), "feedback is not a claim; separate handler cohort gate still applies")
		})
	}
	require.Equal(t, uint64(50_000), lookupMsgGas("/zerone.knowledge.v1.MsgReportFactUse"))
	require.Equal(t, uint64(20_000), lookupMsgGas("/zerone.knowledge.v1.MsgRateFact"))
	require.False(t, IsSystemTransaction("report_fact_use"))
	require.False(t, IsSystemTransaction("rate_fact"))
}
