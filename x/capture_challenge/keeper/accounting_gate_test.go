package keeper_test

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/require"

	"github.com/zerone-chain/zerone/internal/activation"
	"github.com/zerone-chain/zerone/x/capture_challenge/keeper"
	"github.com/zerone-chain/zerone/x/capture_challenge/types"
)

func TestAccountingGatesFailClosedOnMarkerErrors(t *testing.T) {
	for _, bad := range []string{"malformed", "unreadable"} {
		for _, path := range []string{"begin_block", "phase_advance", "resolve", "resolve_unauthorized", "resolve_missing", "auto_submit", "defense_adapter"} {
			t.Run(bad+"/"+path, func(t *testing.T) {
				h := newChallengeAccountingHarnessWithMarker(t, false)
				id := h.submit(t, 1, "math")
				ch := h.challenge(t, id)
				h.advance(t, ch.EvidenceDeadline)
				h.ctx = h.ctx.WithBlockHeight(int64(ch.ReviewDeadline)).WithEventManager(sdk.NewEventManager())
				if bad == "malformed" {
					h.ctx.KVStore(h.keys[types.StoreKey]).Set(types.ChallengeAccountingEnabledKey, []byte{0})
				} else {
					h.writeFault.markerReadErr = true
				}
				before := h.snapshot(t)
				h.ctx = h.ctx.WithGasMeter(storetypes.NewGasMeter(10_000_000))
				h.writeFault.markerReads = 0
				var err error
				switch path {
				case "begin_block":
					err = h.module.BeginBlock(h.ctx)
				case "phase_advance":
					err = h.k.AdvanceChallengePhases(h.ctx, ch.ReviewDeadline)
				case "resolve", "resolve_unauthorized", "resolve_missing":
					authority, challengeID := h.k.GetAuthority(), id
					if path == "resolve_unauthorized" {
						authority = testAddr(2)
					}
					if path == "resolve_missing" {
						challengeID = "missing"
					}
					_, err = keeper.NewMsgServerImpl(h.k).ResolveChallenge(h.ctx, &types.MsgResolveChallenge{
						Authority: authority, ChallengeId: challengeID, Outcome: types.ChallengeOutcome_CHALLENGE_OUTCOME_REJECTED,
					})
				case "auto_submit":
					err = h.k.AutoSubmitChallenge(h.ctx, "math", 1, 1, "synthetic signal")
				case "defense_adapter":
					err = keeper.NewCaptureDefenseAutoChallenger(h.k).AutoSubmitChallenge(h.ctx, "math", 1, 1, "synthetic signal")
				}
				require.ErrorContains(t, err, "marker")
				require.ErrorIs(t, err, activation.ErrMarkerFault)
				require.Equal(t, 1, h.writeFault.markerReads)
				require.Zero(t, h.ctx.GasMeter().GasConsumed(), "fault must precede ordinary reads as well as writes")
				require.Equal(t, before, h.snapshot(t), "a bad selector must not choose either accounting path")
				require.Empty(t, h.ctx.EventManager().Events())
				require.Zero(t, h.bank.refundCalls)
			})
		}
	}
}

func TestAccountingAutomaticReplacementAtSuccessfulExpiry(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		t.Run(fmt.Sprint(enabled), func(t *testing.T) {
			h := newChallengeAccountingHarnessWithMarker(t, enabled)
			require.NoError(t, h.k.AutoSubmitChallenge(h.ctx, "math", 1, 1, "first signal"))
			oldID := keeper.GenerateChallengeID(authtypes.NewModuleAddress(types.ModuleName).String(), "math", h.ctx.BlockHeight())
			deadline := h.challenge(t, oldID).ReviewDeadline
			h.ctx = h.ctx.WithBlockHeight(int64(deadline - 1))
			require.NoError(t, h.k.AutoSubmitChallenge(h.ctx, "math", 1, 1, "still flagged"))
			require.Len(t, h.k.GetChallengesByDomain(h.ctx, "math"), 1)
			h.ctx = h.ctx.WithBlockHeight(int64(deadline)).WithEventManager(sdk.NewEventManager())
			// The producer runs before this module's BeginBlock. No bank fault
			// is involved: an automatic challenge has no stake to refund.
			require.NoError(t, h.k.AutoSubmitChallenge(h.ctx, "math", 1, 1, "still flagged"))
			want := 1
			if enabled {
				want = 2
				require.Equal(t, "zerone.capture_challenge.auto_challenge_submitted", h.ctx.EventManager().Events()[0].Type)
			} else {
				require.Empty(t, h.ctx.EventManager().Events())
			}
			require.Len(t, h.k.GetChallengesByDomain(h.ctx, "math"), want)
			h.advance(t, deadline)
			require.Equal(t, types.ChallengeStatus_CHALLENGE_STATUS_EXPIRED, h.challenge(t, oldID).Status)
			require.Zero(t, h.bank.refundCalls)
		})
	}
}

func pendingRefundEvent(ch *types.CaptureChallenge) sdk.Event {
	return sdk.NewEvent("zerone.capture_challenge.expiry_refund_pending",
		sdk.NewAttribute("challenge_id", ch.Id),
		sdk.NewAttribute("domain", ch.Domain),
		sdk.NewAttribute("stake", ch.Stake),
		sdk.NewAttribute("review_deadline", fmt.Sprintf("%d", ch.ReviewDeadline)))
}

func TestAccountingPendingAnnouncementsAreThrottledNotRefunds(t *testing.T) {
	h := newChallengeAccountingHarness(t)
	id := h.submit(t, 1, "math")
	ch := h.challenge(t, id)
	h.advance(t, ch.EvidenceDeadline)
	h.bank.failRefund = "before"
	var output bytes.Buffer
	h.ctx = h.ctx.WithLogger(log.NewLogger(&output, log.OutputJSONOption()))
	for _, offset := range []uint64{0, 1, 99, 100, 101, 200} {
		h.ctx = h.ctx.WithEventManager(sdk.NewEventManager())
		h.advance(t, ch.ReviewDeadline+offset)
		if offset%100 == 0 {
			require.Equal(t, sdk.Events{pendingRefundEvent(ch)}, h.ctx.EventManager().Events())
		} else {
			require.Empty(t, h.ctx.EventManager().Events())
		}
	}
	require.Equal(t, 6, h.bank.refundCalls, "log throttling must not introduce payment backoff")
	require.Equal(t, 3, strings.Count(output.String(), `"message":"expiry refund pending"`))
	require.Equal(t, 3, strings.Count(output.String(), `"level":"error"`))
	require.Equal(t, types.ChallengeStatus_CHALLENGE_STATUS_UNDER_REVIEW, h.challenge(t, id).Status)
	h.assertBacking(t)
}

// This is an activation-blocker characterization, NOT a safety claim for an
// unreconciled chain. A marker is prospective and cannot repair old pool debt.
func TestAccountingLegacyPhantomPoolRemainsActivationBlocker(t *testing.T) {
	h := newChallengeAccountingHarnessWithMarker(t, false)
	h.k.SetChallenge(h.ctx, &types.CaptureChallenge{
		Id: "old-rejected", Challenger: testAddr(4), Domain: "math", Stake: "100000000",
		Status:     types.ChallengeStatus_CHALLENGE_STATUS_RESOLVED,
		Resolution: &types.CaptureResolution{Outcome: types.ChallengeOutcome_CHALLENGE_OUTCOME_REJECTED},
	})
	h.k.SetBountyPool(h.ctx, &types.DomainBountyPool{Domain: "math", Balance: "100000000"})
	h.k.SetChallenge(h.ctx, &types.CaptureChallenge{
		Id: "old-expired", Challenger: testAddr(5), Stake: "10000000", ReviewDeadline: 50,
		Status: types.ChallengeStatus_CHALLENGE_STATUS_EXPIRED,
	})
	winner := h.submit(t, 1, "math")
	innocent := h.submit(t, 2, "physics")
	h.advance(t, h.challenge(t, winner).EvidenceDeadline)
	before := h.snapshot(t)
	require.NoError(t, h.k.EnableChallengeAccounting(h.ctx))
	before[types.StoreKey][string(types.ChallengeAccountingEnabledKey)] = "\x01"
	require.Equal(t, before, h.snapshot(t), "activation must not rewrite or claw back old records or money")
	h.resolve(t, winner, types.ChallengeOutcome_CHALLENGE_OUTCOME_UPHELD)
	require.Equal(t, "10000000", h.challenge(t, winner).Resolution.RewardAmount)
	require.Equal(t, "0", h.moduleBalance(types.ModuleName), "phantom reward has consumed the innocent escrow")
	attemptsBefore := h.bank.refundCalls
	deadline := h.challenge(t, innocent).ReviewDeadline
	for offset := uint64(0); offset < 7; offset++ {
		h.advance(t, deadline+offset)
	}
	require.Equal(t, attemptsBefore+7, h.bank.refundCalls)
	require.Equal(t, types.ChallengeStatus_CHALLENGE_STATUS_UNDER_REVIEW, h.challenge(t, innocent).Status)
	require.Equal(t, "0", h.bank.GetBalance(h.ctx, sdk.MustAccAddressFromBech32(testAddr(2)), "uzrn").Amount.String())
	require.Equal(t, types.ChallengeStatus_CHALLENGE_STATUS_EXPIRED, h.challenge(t, "old-expired").Status)
	require.Equal(t, "0", h.bank.GetBalance(h.ctx, sdk.MustAccAddressFromBech32(testAddr(5)), "uzrn").Amount.String())
	active, err := keeper.NewQueryServerImpl(h.k).ActiveChallenges(h.ctx, &types.QueryActiveChallengesRequest{})
	require.NoError(t, err)
	require.Len(t, active.Challenges, 1)
	require.Equal(t, innocent, active.Challenges[0].Id)
	h.assertSupply(t)
}
