package keeper_test

import (
	"fmt"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/zerone-chain/zerone/x/capture_challenge/keeper"
	"github.com/zerone-chain/zerone/x/capture_challenge/types"
)

func TestAccountingEntryPointsReadMarkerOnce(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		for _, path := range []string{"begin_block", "phase_advance", "resolve", "auto_submit", "defense_adapter"} {
			t.Run(fmt.Sprintf("%t/%s", enabled, path), func(t *testing.T) {
				h := newChallengeAccountingHarnessWithMarker(t, enabled)
				// Multiple due records force traversal rather than only an empty
				// store or early duplicate return. Zero stake avoids bank faults.
				for i := 0; i < 3; i++ {
					id := fmt.Sprintf("old-%d", i)
					h.k.SetChallenge(h.ctx, &types.CaptureChallenge{
						Id: id, Challenger: testAddr(1), Domain: "math", Stake: "0",
						Status: types.ChallengeStatus_CHALLENGE_STATUS_UNDER_REVIEW, ReviewDeadline: 200,
					})
					h.k.SetDomainIndex(h.ctx, "math", id)
				}
				h.ctx = h.ctx.WithBlockHeight(200).WithEventManager(sdk.NewEventManager())
				h.writeFault.markerReads = 0
				var err error
				switch path {
				case "begin_block":
					err = h.module.BeginBlock(h.ctx)
				case "phase_advance":
					err = h.k.AdvanceChallengePhases(h.ctx, 200)
				case "resolve":
					h.ctx = h.ctx.WithBlockHeight(199)
					_, err = keeper.NewMsgServerImpl(h.k).ResolveChallenge(h.ctx, &types.MsgResolveChallenge{
						Authority: h.k.GetAuthority(), ChallengeId: "old-0", Outcome: types.ChallengeOutcome_CHALLENGE_OUTCOME_REJECTED,
					})
				case "auto_submit":
					err = h.k.AutoSubmitChallenge(h.ctx, "math", 42, 7, "synthetic signal")
				case "defense_adapter":
					err = keeper.NewCaptureDefenseAutoChallenger(h.k).AutoSubmitChallenge(h.ctx, "math", 42, 7, "synthetic signal")
				}
				require.NoError(t, err)
				require.Equal(t, 1, h.writeFault.markerReads)
				if path == "auto_submit" || path == "defense_adapter" {
					want := 3
					if enabled {
						want++
					}
					require.Len(t, h.k.GetChallengesByDomain(h.ctx, "math"), want)
				}
			})
		}
	}
}

func TestAccountingPendingEventRequiresActivation(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		t.Run(fmt.Sprint(enabled), func(t *testing.T) {
			h := newChallengeAccountingHarnessWithMarker(t, enabled)
			id := h.submit(t, 1, "math")
			ch := h.challenge(t, id)
			h.advance(t, ch.EvidenceDeadline)
			h.bank.failRefund = "before"
			h.ctx = h.ctx.WithEventManager(sdk.NewEventManager())
			h.advance(t, ch.ReviewDeadline)
			require.Equal(t, 1, h.bank.refundCalls)
			if enabled {
				require.Equal(t, sdk.Events{pendingRefundEvent(ch)}, h.ctx.EventManager().Events())
				require.Equal(t, types.ChallengeStatus_CHALLENGE_STATUS_UNDER_REVIEW, h.challenge(t, id).Status)
			} else {
				require.Empty(t, h.ctx.EventManager().Events())
				require.Equal(t, types.ChallengeStatus_CHALLENGE_STATUS_EXPIRED, h.challenge(t, id).Status)
			}
		})
	}
}
