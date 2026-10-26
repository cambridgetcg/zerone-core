package keeper_test

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"

	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/require"

	"github.com/zerone-chain/zerone/x/capture_challenge/keeper"
	"github.com/zerone-chain/zerone/x/capture_challenge/types"
)

// The fixture was captured with an offline Go overlay of the four production
// keeper files from 77521a0d (the module subtree is identical to 2e37c4c).
// It pins all auth/bank/challenge bytes, ordered events, errors, send attempts
// and charged gas. It intentionally pins the defects while the marker is off.
// There is no in-test update mode: do not regenerate it with repaired source.
type legacyAccountingResult struct {
	StateSHA256  string `json:"state_sha256"`
	EventsSHA256 string `json:"events_sha256"`
	Error        string `json:"error"`
	Gas          uint64 `json:"gas"`
	RefundCalls  int    `json:"refund_calls"`
}

func TestAccountingLegacyCompatibility(t *testing.T) {
	want := map[string]legacyAccountingResult{}
	bz, readErr := os.ReadFile("testdata/accounting_legacy_77521a0d.json")
	if readErr == nil {
		require.NoError(t, json.Unmarshal(bz, &want))
	}
	cases := []string{
		"resolve_upheld_before", "resolve_upheld_deadline", "resolve_partial_before", "resolve_partial_deadline",
		"resolve_rejected_before", "resolve_rejected_deadline", "resolve_rejected_no_pool", "resolve_rejected_zero",
		"resolve_invalid", "resolve_unauthorized", "resolve_missing", "resolve_wrong_status", "resolve_bad_address", "resolve_unfunded",
		"expiry_success", "expiry_unfunded", "expiry_before", "expiry_after", "expiry_write_before", "expiry_write_after",
		"expiry_zero", "expiry_bad_address", "expiry_bad_stake", "expiry_negative", "expiry_not_due", "expiry_evidence", "expiry_terminal",
		"auto_before", "auto_deadline", "auto_after", "auto_terminal", "auto_new",
	}
	for _, name := range cases {
		t.Run(name, func(t *testing.T) {
			h := newChallengeAccountingHarnessWithMarker(t, false)
			ch := &types.CaptureChallenge{
				Id: "legacy", Challenger: testAddr(1), Domain: "math", Stake: "10000000",
				Status:           types.ChallengeStatus_CHALLENGE_STATUS_UNDER_REVIEW,
				EvidenceDeadline: 50, ReviewDeadline: 200,
			}
			height := int64(200)
			funds := int64(10_000_000)
			var act func() error
			if strings.HasPrefix(name, "resolve_") {
				height = 199
				if strings.HasSuffix(name, "_deadline") {
					height = 200
				}
				if name != "resolve_rejected_no_pool" {
					h.k.SetBountyPool(h.ctx, &types.DomainBountyPool{Domain: "math", Balance: "100000000"})
					funds += 100_000_000
				}
				msg := &types.MsgResolveChallenge{Authority: h.k.GetAuthority(), ChallengeId: ch.Id, Reason: "legacy fixture"}
				switch {
				case strings.Contains(name, "upheld"):
					msg.Outcome = types.ChallengeOutcome_CHALLENGE_OUTCOME_UPHELD
				case strings.Contains(name, "partial"):
					msg.Outcome = types.ChallengeOutcome_CHALLENGE_OUTCOME_PARTIAL
				default:
					msg.Outcome = types.ChallengeOutcome_CHALLENGE_OUTCOME_REJECTED
				}
				switch name {
				case "resolve_invalid":
					msg.Outcome = types.ChallengeOutcome_CHALLENGE_OUTCOME_UNSPECIFIED
				case "resolve_unauthorized":
					msg.Authority = testAddr(2)
				case "resolve_missing":
					msg.ChallengeId = "missing"
				case "resolve_wrong_status":
					ch.Status = types.ChallengeStatus_CHALLENGE_STATUS_EVIDENCE
				case "resolve_bad_address":
					ch.Challenger = "invalid-address"
				case "resolve_unfunded":
					funds = 0
				case "resolve_rejected_zero":
					ch.Stake = "0"
				}
				act = func() error {
					_, err := keeper.NewMsgServerImpl(h.k).ResolveChallenge(h.ctx, msg)
					return err
				}
			} else if strings.HasPrefix(name, "expiry_") {
				switch name {
				case "expiry_unfunded":
					funds = 0
				case "expiry_zero":
					ch.Stake = "0"
				case "expiry_bad_address":
					ch.Challenger = "invalid-address"
				case "expiry_bad_stake":
					ch.Stake = "invalid"
				case "expiry_negative":
					ch.Stake = "-1"
				case "expiry_not_due":
					height = 199
				case "expiry_evidence":
					ch.Status = types.ChallengeStatus_CHALLENGE_STATUS_EVIDENCE
				case "expiry_terminal":
					ch.Status = types.ChallengeStatus_CHALLENGE_STATUS_EXPIRED
				}
				act = func() error { return h.module.BeginBlock(h.ctx) }
			} else {
				switch name {
				case "auto_before":
					height = 199
				case "auto_after":
					height = 201
				case "auto_terminal":
					ch.Status = types.ChallengeStatus_CHALLENGE_STATUS_EXPIRED
				}
				act = func() error { return h.k.AutoSubmitChallenge(h.ctx, "math", 42, 7, "legacy signal") }
			}
			if name != "auto_new" {
				h.k.SetChallenge(h.ctx, ch)
				h.k.SetDomainIndex(h.ctx, ch.Domain, ch.Id)
			}
			if funds > 0 {
				h.fund(t, authtypes.NewModuleAddress(types.ModuleName), funds)
			}
			if name == "expiry_before" || name == "expiry_after" {
				h.bank.failRefund = strings.TrimPrefix(name, "expiry_")
			}
			if strings.HasPrefix(name, "expiry_write_") {
				h.writeFault.key = append(append([]byte{}, types.ChallengeKeyPrefix...), []byte(ch.Id)...)
				h.writeFault.mode = strings.TrimPrefix(name, "expiry_write_")
			}
			h.ctx = h.ctx.WithBlockHeight(height).WithEventManager(sdk.NewEventManager()).WithGasMeter(storetypes.NewGasMeter(10_000_000))
			err := act()
			got := legacyAccountingResult{Gas: h.ctx.GasMeter().GasConsumed(), RefundCalls: h.bank.refundCalls}
			if err != nil {
				got.Error = err.Error()
			}
			events, err := json.Marshal(h.ctx.EventManager().Events())
			require.NoError(t, err)
			got.EventsSHA256 = fmt.Sprintf("%x", sha256.Sum256(events))
			// Hex encoding prevents JSON's invalid-UTF8 replacement from hiding
			// differences in raw store keys or protobuf values.
			state := h.snapshot(t)
			encoded := map[string]map[string]string{}
			for store, entries := range state {
				encoded[store] = map[string]string{}
				for key, value := range entries {
					encoded[store][hex.EncodeToString([]byte(key))] = hex.EncodeToString([]byte(value))
				}
			}
			raw, err := json.Marshal(encoded)
			require.NoError(t, err)
			got.StateSHA256 = fmt.Sprintf("%x", sha256.Sum256(raw))
			result, err := json.Marshal(got)
			require.NoError(t, err)
			t.Logf("LEGACY_RESULT %s %s", name, result)
			require.NoError(t, readErr, "legacy oracle fixture must exist")
			expected, ok := want[name]
			require.True(t, ok, "legacy oracle must cover %s", name)
			require.Equal(t, expected, got)
		})
	}
	require.Len(t, want, len(cases), "no stale or untested oracle entries")
}
