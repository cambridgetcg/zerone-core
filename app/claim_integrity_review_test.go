package app

import (
	"fmt"
	"testing"

	"cosmossdk.io/log"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	knowledgekeeper "github.com/zerone-chain/zerone/x/knowledge/keeper"
	knowledgetypes "github.com/zerone-chain/zerone/x/knowledge/types"
)

// Claim integrity, Task 14 (P5), on the vote-extension path. PreBlocker
// stores each VRF-verified injected commitment through
// storeInjectedCommitment, which uses the keeper's StoreCommitmentInRound, the
// same path as MsgSubmitCommitment. With the 0x87 marker on, a validator whose
// operator key is the claim's submitter is refused there: the commitment is
// skipped and logged, and the block goes on. The release latch
// (potVoteExtensionsReleaseEnabled) keeps ProcessVoteExtInjection itself
// closed, so the helper is driven directly. Only a legacy-scheme round
// accepts an operator address at all; a scheme-2 round refuses one as a
// non-canonical reviewer before this check.

type recordedLog struct {
	level, msg string
	keyVals    []any
}

type recordingLogger struct{ entries *[]recordedLog }

func (l recordingLogger) add(level, msg string, keyVals []any) {
	*l.entries = append(*l.entries, recordedLog{level, msg, keyVals})
}
func (l recordingLogger) Info(msg string, keyVals ...any)  { l.add("info", msg, keyVals) }
func (l recordingLogger) Warn(msg string, keyVals ...any)  { l.add("warn", msg, keyVals) }
func (l recordingLogger) Error(msg string, keyVals ...any) { l.add("error", msg, keyVals) }
func (l recordingLogger) Debug(msg string, keyVals ...any) { l.add("debug", msg, keyVals) }
func (l recordingLogger) With(...any) log.Logger           { return l }
func (l recordingLogger) Impl() any                        { return l }

func TestClaimIntegrityVoteExtensionSkipsPartyCommitment(t *testing.T) {
	for _, integrity := range []bool{true, false} {
		t.Run(fmt.Sprintf("marker=%t", integrity), func(t *testing.T) {
			app, ctx := settlementFixture(t)
			if !integrity {
				ctx.KVStore(app.keys[knowledgetypes.StoreKey]).Delete([]byte(knowledgekeeper.ClaimIntegrityEnabledStoreKey))
			}
			enabled, err := app.KnowledgeKeeper.ClaimIntegrityEnabled(ctx)
			require.NoError(t, err)
			require.Equal(t, integrity, enabled)

			submitter := settlementAddress(81)
			claim := &knowledgetypes.Claim{
				Id: "vote-extension-party-claim", Submitter: submitter.String(), Stake: "100",
				FactContent: "A legacy-scheme claim whose submitter also runs a validator.", Domain: "physics", Category: "empirical",
				ClaimType: knowledgetypes.ClaimType_CLAIM_TYPE_ASSERTION, Status: knowledgetypes.ClaimStatus_CLAIM_STATUS_IN_VERIFICATION,
				VerificationRoundId: "vote-extension-party-round",
			}
			round := &knowledgetypes.VerificationRound{
				Id: "vote-extension-party-round", ClaimId: claim.Id, StartedAtBlock: 90, Phase: knowledgetypes.VerificationPhase_VERIFICATION_PHASE_COMMIT,
				CommitDeadline: 290, RevealDeadline: 490, AggregationDeadline: 540,
			}
			require.NoError(t, app.KnowledgeKeeper.SetClaim(ctx, claim))
			require.NoError(t, app.KnowledgeKeeper.SetVerificationRound(ctx, round))
			require.Equal(t, knowledgetypes.CommitmentSchemeLegacy, round.CommitmentScheme)

			var entries []recordedLog
			logger := recordingLogger{&entries}
			commit := func(validator sdk.ValAddress) bool {
				return app.storeInjectedCommitment(ctx, logger, round.Id, &knowledgetypes.CommitEntry{
					Verifier: validator.String(), CommitHash: []byte("vote-extension-commitment-hash-32"), CommittedAtBlock: uint64(ctx.BlockHeight()),
				})
			}
			committers := func() []string {
				stored, found := app.KnowledgeKeeper.GetVerificationRound(ctx, round.Id)
				require.True(t, found)
				var out []string
				for _, c := range stored.Commits {
					out = append(out, c.Verifier)
				}
				return out
			}

			party := sdk.ValAddress(submitter)
			bystander := sdk.ValAddress(settlementAddress(82))
			if !integrity {
				require.True(t, commit(party), "knowledge 11 lets the submitter's validator commit")
				require.True(t, commit(bystander))
				require.Equal(t, []string{party.String(), bystander.String()}, committers())
				require.Empty(t, entries)
				return
			}
			require.False(t, commit(party), "the submitter's validator commitment is skipped")
			require.Empty(t, committers())
			require.Equal(t, []recordedLog{{
				level: "debug", msg: "skipped commitment from vote extension",
				keyVals: []any{
					"round_id", round.Id,
					"validator", party.String(),
					"reason", fmt.Sprintf("reviewer %s is a party to claim %s (submitter): verifier not qualified for domain", party, claim.Id),
				},
			}}, entries)
			require.True(t, commit(bystander), "a non-party validator still commits")
			require.Equal(t, []string{bystander.String()}, committers())
			require.Len(t, entries, 1)
		})
	}
}
