package keeper_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/bech32"
	"github.com/stretchr/testify/require"

	"github.com/zerone-chain/zerone/x/knowledge/keeper"
	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// Claim integrity, Task 14 (P5): a party to a claim may not sit on its panel.
// With the marker on, StoreCommitmentInRound, the one store path shared by
// MsgSubmitCommitment and the vote-extension injection, refuses a commitment
// from:
//   - the claim's submitter;
//   - the defender of the one fact the claim disputes, meaning that fact's
//     submitter: the ProvisionalFactId of a MsgChallengeFact or
//     MsgChallengeProvisionalFact claim, or else the single CONTRADICTS target
//     of a CHALLENGE_DEPOSIT claim with exactly one relation
//     (MsgSubmitContradiction).
// A MsgSubmitClaim's relation list is chosen by its submitter and is never
// read, so naming facts cannot recuse their defenders (plan §10). A reviewer
// is a party when its address string equals the party's, or when both decode
// as bech32 to the same bytes under any prefix (an operator or upper-case
// encoding of the same key). Reveals are not guarded: a commitment admitted
// before activation still reveals and settles. A record the check cannot read
// refuses the commitment. With the marker off, knowledge 11 lets parties
// review; the marker-off twins pin that.

const rvSubmitter = "submitter"

func rvDefender(factID string) string { return "defender of fact " + factID }

func rvParty(reviewer, claimID, role string) string {
	return fmt.Sprintf("reviewer %s is a party to claim %s (%s): verifier not qualified for domain", reviewer, claimID, role)
}

var rvAttestation = &types.ReviewAttestation{Reason: "Synthetic v12 probe P5: a review of the stated record only."}

func rvSalt(reviewer string) []byte { return []byte("v12-rv-salt-" + reviewer) }

// rvCommitment funds reviewer and returns its scheme-2 commitment to vote on
// claimID's round.
func (f *v11Fixture) rvCommitment(t *testing.T, claimID, reviewer, vote string) *types.MsgSubmitCommitment {
	t.Helper()
	round := f.roundOf(t, claimID)
	f.bank.balances[reviewer] = sdk.NewCoins(sdk.NewInt64Coin("uzrn", 100_000_000))
	hash, err := types.ComputeReviewCommitmentV2(round.CommitmentChainId, round.Id, reviewer, vote, v11Confidence, rvSalt(reviewer), rvAttestation)
	require.NoError(t, err)
	return &types.MsgSubmitCommitment{Verifier: reviewer, RoundId: round.Id, CommitHash: hash}
}

func (f *v11Fixture) rvStore(roundID, reviewer string, hash []byte) error {
	return f.k.StoreCommitmentInRound(f.ctx, roundID, &types.CommitEntry{
		Verifier: reviewer, CommitHash: hash, CommittedAtBlock: uint64(f.ctx.BlockHeight()),
	})
}

// requireCommitRefused requires the transaction path and the direct store
// path to refuse reviewer on claimID's round with want, writing nothing.
func (f *v11Fixture) requireCommitRefused(t *testing.T, claimID, reviewer, want string) {
	t.Helper()
	msg := f.rvCommitment(t, claimID, reviewer, "accept")
	before, _ := f.storeDigest(t)
	_, err := f.server.SubmitCommitment(f.ctx, msg)
	require.EqualError(t, err, want, "transaction path")
	require.ErrorIs(t, err, types.ErrUnqualifiedVerifier)
	err = f.rvStore(msg.RoundId, reviewer, msg.CommitHash)
	require.EqualError(t, err, want, "direct store path")
	require.ErrorIs(t, err, types.ErrUnqualifiedVerifier)
	after, _ := f.storeDigest(t)
	require.Equal(t, before, after, "a refused commitment writes nothing")
}

// requireCommitAdmitted requires both paths to admit reviewer on claimID's
// round, each in a discarded cache.
func (f *v11Fixture) requireCommitAdmitted(t *testing.T, claimID, reviewer string) {
	t.Helper()
	msg := f.rvCommitment(t, claimID, reviewer, "accept")
	f.inCache(func() {
		_, err := f.server.SubmitCommitment(f.ctx, msg)
		require.NoError(t, err, "transaction path")
		require.NotNil(t, findCommit(f.roundOf(t, claimID), reviewer))
	})
	f.inCache(func() {
		require.NoError(t, f.rvStore(msg.RoundId, reviewer, msg.CommitHash), "direct store path")
		require.NotNil(t, findCommit(f.roundOf(t, claimID), reviewer))
	})
}

func findCommit(round *types.VerificationRound, reviewer string) *types.CommitEntry {
	for _, commit := range round.Commits {
		if commit.Verifier == reviewer {
			return commit
		}
	}
	return nil
}

func rvCommitters(round *types.VerificationRound) []string {
	var out []string
	for _, commit := range round.Commits {
		out = append(out, commit.Verifier)
	}
	return out
}

func rvPaid(t *testing.T, round *types.VerificationRound) []string {
	t.Helper()
	require.NotNil(t, round.VerifierRewardSettlement)
	var out []string
	for _, payment := range round.VerifierRewardSettlement.Payments {
		out = append(out, payment.Verifier)
	}
	return out
}

// Plan §7 #13. Marker on: the submitter's commitment is refused on both
// paths, a panel of non-parties decides the claim, and only they are paid.
// Then the defender (the fact's submitter) and the challenger are both
// refused on a MsgChallengeFact round. Marker off: knowledge 11 admits both.
func TestClaimIntegrity_SubmitterAndDefenderCannotReview(t *testing.T) {
	for _, sel := range dedupSelections {
		t.Run(sel.name, func(t *testing.T) {
			f := sel.setup(t)
			require.Equal(t, sel.integrity, f.claimIntegrityEnabled(t))
			author := v11Addr("v12-rv-author")
			claimT := f.mustSubmitClaim(t, author, "Synthetic v12 probe P5: the author of this claim tries to review it.", dedupDomain, nil)
			if sel.integrity {
				f.requireCommitRefused(t, claimT, author, rvParty(author, claimT, rvSubmitter))
			} else {
				f.requireCommitAdmitted(t, claimT, author)
			}
			factT := f.accept(t, claimT, v11Reviewers()...)
			round := f.roundOf(t, claimT)
			require.Equal(t, v11Reviewers(), rvCommitters(round))
			require.ElementsMatch(t, v11Reviewers(), rvPaid(t, round), "only non-parties are paid")
			require.Equal(t, author, f.fact(t, factT).Submitter)

			challenger := v11Addr("v12-rv-challenger")
			c := f.challenge(t, challenger, factT)
			if sel.integrity {
				f.requireCommitRefused(t, c, author, rvParty(author, c, rvDefender(factT)))
				f.requireCommitRefused(t, c, challenger, rvParty(challenger, c, rvSubmitter))
			} else {
				f.requireCommitAdmitted(t, c, author)
				f.requireCommitAdmitted(t, c, challenger)
			}
			survived := f.decide(t, c, "reject", v11Reviewers()...)
			require.Equal(t, types.Verdict_VERDICT_REJECT, survived.Verdict)
			require.ElementsMatch(t, v11Reviewers(), rvPaid(t, survived))
			require.Equal(t, vdActive, f.fact(t, factT).Status)
			f.requireNoLiveRounds(t)
		})
	}
}

// The defender is excluded on every round that disputes exactly one fact:
// a MsgSubmitContradiction round (its one CONTRADICTS target) and a
// MsgChallengeProvisionalFact round (the conjecture's proposer). The dispute's
// own submitter is excluded too.
func TestClaimIntegrity_DisputeDefenderCannotReview(t *testing.T) {
	for _, sel := range dedupSelections {
		t.Run(sel.name, func(t *testing.T) {
			check := func(t *testing.T, f *v11Fixture, claimID, reviewer, role string) {
				t.Helper()
				if sel.integrity {
					f.requireCommitRefused(t, claimID, reviewer, rvParty(reviewer, claimID, role))
				} else {
					f.requireCommitAdmitted(t, claimID, reviewer)
				}
			}

			t.Run("contradiction", func(t *testing.T) {
				f := sel.setup(t)
				defender := "v12-rv-f"
				_, factF, minimum := f.contraTarget(t, defender, "Synthetic v12 probe P5: fact F, whose submitter defends it against a contradiction.")
				jones := v11Addr("v12-rv-jones")
				a := f.mustContradict(t, jones, factF, "Synthetic v12 probe P5: a contradiction of fact F that a panel must decide.", minimum)
				claim := f.claim(t, a)
				require.Equal(t, types.ClaimFundingKind_CLAIM_FUNDING_KIND_CHALLENGE_DEPOSIT, claim.FundingTerms.Kind)
				require.Len(t, claim.Relations, 1)
				check(t, f, a, v11Addr(defender), rvDefender(factF))
				check(t, f, a, jones, rvSubmitter)
				f.requireCommitAdmitted(t, a, v11Addr("v12-rv-bystander"))
			})

			t.Run("provisional challenge", func(t *testing.T) {
				f := sel.setup(t)
				proposer := v11Addr("v12-rv-proposer")
				conjecture, err := f.server.PostConjecture(f.ctx, &types.MsgPostConjecture{
					Proposer: proposer, Domain: dedupDomain, Category: "empirical", Stake: v11ReviewFee,
					Statement:              "Synthetic v12 probe P5: conjecture Q, defended by its proposer.",
					FalsificationPredicate: "Exhibit one synthetic counterexample to conjecture Q in this fixture.",
				})
				require.NoError(t, err)
				factQ := f.accept(t, conjecture.ClaimId, v11Reviewers()...)
				params, err := f.k.GetParams(f.ctx)
				require.NoError(t, err)
				refuter := v11Addr("v12-rv-refuter")
				opened, err := f.server.ChallengeProvisionalFact(f.ctx, &types.MsgChallengeProvisionalFact{
					Challenger: refuter, FactId: factQ, ClaimId: conjecture.ClaimId, Stake: params.MinChallengeStake,
					Reason: "Synthetic v12 probe P5: a provisional challenge that a panel decides.",
				})
				require.NoError(t, err)
				round, found := f.k.GetVerificationRound(f.ctx, opened.ChallengeId)
				require.True(t, found)
				check(t, f, round.ClaimId, proposer, rvDefender(factQ))
				check(t, f, round.ClaimId, refuter, rvSubmitter)
				f.requireCommitAdmitted(t, round.ClaimId, v11Addr("v12-rv-bystander"))
			})
		})
	}
}

// A MsgSubmitClaim carrying CONTRADICTS relations is a REVIEW_FEE claim whose
// relation list its submitter chose. With the marker on such a claim is
// refused at admission (review G2, decision (a)), so the case is a claim
// admitted before activation whose round is still open after it. Only the
// submitter is excluded, whether it names one target or the maximum of 16,
// so naming facts recuses none of their defenders.
func TestClaimIntegrity_SubmitClaimRelationsRecuseNoDefender(t *testing.T) {
	require.Equal(t, 16, types.MaxRelationsPerClaim)
	for _, sel := range dedupSelections {
		t.Run(sel.name, func(t *testing.T) {
			f := setupV11SemanticsTest(t)
			var defenders, facts []string
			for i := 0; i < types.MaxRelationsPerClaim; i++ {
				defender := v11Addr(fmt.Sprintf("v12-rv-defender-%02d", i))
				claimID := f.mustSubmitClaim(t, defender, fmt.Sprintf("Synthetic v12 probe P5: fact %02d, one of sixteen targets of a single claim.", i), dedupDomain, nil)
				defenders = append(defenders, defender)
				facts = append(facts, f.accept(t, claimID, v11Reviewers()...))
			}
			counts := []int{1, types.MaxRelationsPerClaim}
			claims := map[int]string{}
			for _, n := range counts {
				claims[n] = f.mustSubmitClaim(t, v11Addr(fmt.Sprintf("v12-rv-multi-%02d", n)), fmt.Sprintf("Synthetic v12 probe P5: one claim that contradicts %d accepted facts at once.", n), dedupDomain,
					func(m *types.MsgSubmitClaim) {
						for _, factID := range facts[:n] {
							m.Relations = append(m.Relations, &types.ClaimRelation{Relation: types.RelationType_RELATION_TYPE_CONTRADICTS, TargetFactId: factID})
						}
					})
			}
			if sel.integrity {
				require.NoError(t, f.k.ValidateClaimIntegrityActivation(f.ctx))
				require.NoError(t, f.k.EnableClaimIntegrity(f.ctx))
			}
			require.Equal(t, sel.integrity, f.claimIntegrityEnabled(t))
			for _, n := range counts {
				t.Run(fmt.Sprintf("%d targets", n), func(t *testing.T) {
					submitter, c := v11Addr(fmt.Sprintf("v12-rv-multi-%02d", n)), claims[n]
					claim := f.claim(t, c)
					require.Len(t, claim.Relations, n)
					require.Equal(t, types.ClaimFundingKind_CLAIM_FUNDING_KIND_REVIEW_FEE, claim.FundingTerms.Kind)
					if sel.integrity {
						f.requireCommitRefused(t, c, submitter, rvParty(submitter, c, rvSubmitter))
					} else {
						f.requireCommitAdmitted(t, c, submitter)
					}
					for _, defender := range defenders[:n] {
						f.requireCommitAdmitted(t, c, defender)
					}
				})
			}
		})
	}
}

// rvLegacyRound admits a claim on a store with no selection on, so its round
// keeps the legacy commitment scheme (the only one an operator address can
// commit to), then selects record integrity, review neutrality, claim
// records, fund settlement and, when integrity is set, claim integrity in
// place. It returns the claim ID and the submitter's address bytes.
func rvLegacyRound(t *testing.T, integrity bool) (*v11Fixture, string, sdk.AccAddress) {
	t.Helper()
	gs := types.DefaultGenesis()
	gs.RecordIntegrityEnabled = false
	gs.ReviewNeutralityEnabled = false
	gs.ClaimRecordsEnabled = false
	gs.FundSettlementEnabled = false
	gs.ClaimIntegrityEnabled = false
	f := newClaimIntegrityGenesisFixture(t, "claim-integrity-legacy-scheme", gs)
	author := v11Addr("v12-rv-legacy")
	c := f.mustSubmitClaim(t, author, "Synthetic v12 probe P5: a legacy-scheme claim whose author holds a validator key.", dedupDomain, nil)
	round := f.roundOf(t, c)
	require.Equal(t, types.CommitmentSchemeLegacy, round.CommitmentScheme)
	enables := []func(context.Context) error{f.k.EnableRecordIntegrity, f.k.EnableReviewNeutrality, f.k.EnableClaimRecords, f.k.EnableFundSettlement}
	if integrity {
		enables = append(enables, f.k.EnableClaimIntegrity)
	}
	for _, enable := range enables {
		require.NoError(t, enable(f.ctx))
	}
	require.Equal(t, integrity, f.claimIntegrityEnabled(t))
	require.Equal(t, types.CommitmentSchemeLegacy, f.roundOf(t, c).CommitmentScheme)
	key, err := sdk.AccAddressFromBech32(author)
	require.NoError(t, err)
	return f, c, key
}

// The submitter's key is one actor in every bech32 encoding: its operator
// (valoper) address, an upper-case copy and another prefix are all refused.
// A different key and a string that is not bech32 are not parties. Only a
// legacy-scheme round reaches this check with such strings; a scheme-2 round
// accepts only canonical account addresses.
func TestClaimIntegrity_PartyMatchesEveryEncodingOfItsKey(t *testing.T) {
	foreign, err := bech32.ConvertAndEncode("cosmos", make([]byte, 20))
	require.NoError(t, err)
	for _, sel := range dedupSelections {
		t.Run(sel.name, func(t *testing.T) {
			f, c, key := rvLegacyRound(t, sel.integrity)
			roundID := f.roundOf(t, c).Id
			other, err := sdk.AccAddressFromBech32(v11Addr("v12-rv-other"))
			require.NoError(t, err)
			cosmos, err := bech32.ConvertAndEncode("cosmos", key)
			require.NoError(t, err)
			require.NotEqual(t, foreign, cosmos)
			hash := []byte("v12-rv-legacy-commitment-hash-32")

			for _, reviewer := range []string{sdk.ValAddress(key).String(), strings.ToUpper(key.String()), cosmos} {
				t.Run(reviewer, func(t *testing.T) {
					require.NotEqual(t, key.String(), reviewer)
					if !sel.integrity {
						f.inCache(func() { require.NoError(t, f.rvStore(roundID, reviewer, hash)) })
						return
					}
					before, _ := f.storeDigest(t)
					err := f.rvStore(roundID, reviewer, hash)
					require.EqualError(t, err, rvParty(reviewer, c, rvSubmitter))
					require.ErrorIs(t, err, types.ErrUnqualifiedVerifier)
					after, _ := f.storeDigest(t)
					require.Equal(t, before, after)
				})
			}
			for _, reviewer := range []string{sdk.ValAddress(other).String(), foreign, "zrn1testvalidator", strings.ToUpper(other.String())} {
				f.inCache(func() { require.NoError(t, f.rvStore(roundID, reviewer, hash), reviewer) })
			}
			// The transaction path never carries an operator address this far:
			// its balance gate parses the verifier as an account address first.
			_, err = f.server.SubmitCommitment(f.ctx, &types.MsgSubmitCommitment{Verifier: sdk.ValAddress(key).String(), RoundId: roundID, CommitHash: hash})
			require.ErrorContains(t, err, "invalid verifier address")
		})
	}
}

// A commitment made before activation still reveals and settles: reveals are
// not guarded, and the submitter is paid as knowledge 11 promised. After
// activation, in place or by import, a new commitment from the same submitter
// is refused while a non-party may still join the panel.
func TestClaimIntegrity_PreActivationSelfCommitStillSettles(t *testing.T) {
	for _, act := range contestActivations {
		t.Run(act.name, func(t *testing.T) {
			f := setupV11SemanticsTest(t)
			author := v11Addr("v12-rv-pre-author")
			claimT := f.mustSubmitClaim(t, author, "Synthetic v12 probe P5: a claim its author committed on before activation.", dedupDomain, nil)
			reviewers := v11Reviewers()
			early := []string{author, reviewers[0], reviewers[1]}
			for _, reviewer := range early {
				_, err := f.server.SubmitCommitment(f.ctx, f.rvCommitment(t, claimT, reviewer, "accept"))
				require.NoError(t, err)
			}

			f = act.activate(t, f)
			for _, reviewer := range early {
				f.bank.balances[reviewer] = sdk.NewCoins(sdk.NewInt64Coin("uzrn", 100_000_000))
			}
			round := f.roundOf(t, claimT)
			require.Equal(t, early, rvCommitters(round))
			resend := f.rvCommitment(t, claimT, author, "reject")
			_, err := f.server.SubmitCommitment(f.ctx, resend)
			require.EqualError(t, err, rvParty(author, claimT, rvSubmitter), "a new commitment from the submitter is refused")
			late := f.rvCommitment(t, claimT, reviewers[2], "accept")
			_, err = f.server.SubmitCommitment(f.ctx, late)
			require.NoError(t, err)
			panel := append(early, reviewers[2])

			f.advanceTo(t, round.CommitDeadline)
			f.beginBlock(t)
			for _, reviewer := range panel {
				_, err := f.server.SubmitReveal(f.ctx, &types.MsgSubmitReveal{
					Verifier: reviewer, RoundId: round.Id, Vote: "accept", Confidence: v11Confidence,
					Salt: rvSalt(reviewer), Attestation: rvAttestation,
				})
				require.NoError(t, err, "reveal by %s", reviewer)
			}
			f.advanceTo(t, round.CommitDeadline+1)
			f.beginBlock(t)
			done := f.roundOf(t, claimT)
			require.Equal(t, types.VerificationPhase_VERIFICATION_PHASE_COMPLETE, done.Phase)
			require.Equal(t, types.Verdict_VERDICT_ACCEPT, done.Verdict)
			require.Equal(t, panel, rvCommitters(done))
			require.Contains(t, rvPaid(t, done), author, "the pre-activation self-commit settles")
			require.ElementsMatch(t, panel, rvPaid(t, done))
			f.fact(t, keeper.GenerateFactID(claimT, done.VerdictBlock))
		})
	}
}

// rvChallenged accepts fact T and opens a MsgChallengeFact on it. It returns
// T's submitter, T, the challenger and the challenge claim ID.
func (f *v11Fixture) rvChallenged(t *testing.T, tag string) (author, factT, challenger, c string) {
	t.Helper()
	author = v11Addr("v12-rv-" + tag + "-a")
	claimT := f.mustSubmitClaim(t, author, "Synthetic v12 probe P5 ("+tag+"): fact T, whose records the party check reads.", dedupDomain, nil)
	factT = f.accept(t, claimT, v11Reviewers()...)
	challenger = v11Addr("v12-rv-" + tag + "-c")
	return author, factT, challenger, f.challenge(t, challenger, factT)
}

// The check refuses a commitment it cannot clear: a malformed marker, an
// unreadable claim or an unreadable disputed fact. Both paths refuse, and
// nothing is written.
func TestClaimIntegrity_ReviewerPartyCheckFailsClosed(t *testing.T) {
	for _, tc := range []struct {
		name  string
		fault func(f *v11Fixture, factT, c string)
		want  func(f *v11Fixture, factT, c, roundID string) string
	}{
		{"malformed marker",
			func(f *v11Fixture, _, _ string) {
				f.ctx.KVStore(f.key).Set([]byte(keeper.ClaimIntegrityEnabledStoreKey), []byte{2})
			},
			func(*v11Fixture, string, string, string) string { return "malformed claim integrity marker" }},
		{"claim unreadable",
			func(f *v11Fixture, _, c string) { f.ctx.KVStore(f.key).Set(types.ClaimKey(c), []byte{0xff}) },
			func(_ *v11Fixture, _, c, roundID string) string {
				return fmt.Sprintf("read claim %s of round %s: ", c, roundID)
			}},
		{"disputed fact unreadable",
			func(f *v11Fixture, factT, _ string) { f.ctx.KVStore(f.key).Set(types.FactKey(factT), []byte{0xff}) },
			func(_ *v11Fixture, factT, c, _ string) string {
				return fmt.Sprintf("read disputed fact %s of claim %s: ", factT, c)
			}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := setupClaimIntegrityTest(t)
			_, factT, _, c := f.rvChallenged(t, "fault")
			reviewer := v11Addr("v12-rv-bystander")
			msg := f.rvCommitment(t, c, reviewer, "accept")
			f.requireCommitAdmitted(t, c, reviewer)
			tc.fault(f, factT, c)
			before, _ := f.storeDigest(t)
			err := f.rvStore(msg.RoundId, reviewer, msg.CommitHash)
			require.ErrorContains(t, err, tc.want(f, factT, c, msg.RoundId), "direct store path")
			require.NotErrorIs(t, err, types.ErrUnqualifiedVerifier, "a fault is not a party finding")
			_, err = f.server.SubmitCommitment(f.ctx, msg)
			require.Error(t, err, "transaction path")
			after, _ := f.storeDigest(t)
			require.Equal(t, before, after, "a refused commitment writes nothing")
		})
	}
}

// An absent record names no party, as getClaimRecordChecked treats an absent
// historical claim. Facts are never deleted on chain, so this is store
// damage, not a path a transaction can reach: with T's record gone, the
// challenger is still refused as the submitter and nobody is refused as
// defender. A round whose claim record is absent names no party at all; the
// legacy-round tests in abci_test.go (DefaultGenesis, marker on) commit to
// such rounds unchanged, and a neutral round without its claim is refused
// when it is saved.
func TestClaimIntegrity_AbsentDisputedFactNamesNoDefender(t *testing.T) {
	f := setupClaimIntegrityTest(t)
	author, factT, challenger, c := f.rvChallenged(t, "absent")
	f.ctx.KVStore(f.key).Delete(types.FactKey(factT))
	_, found := f.k.GetFact(f.ctx, factT)
	require.False(t, found)
	f.requireCommitRefused(t, c, challenger, rvParty(challenger, c, rvSubmitter))
	f.requireCommitAdmitted(t, c, author)
}
