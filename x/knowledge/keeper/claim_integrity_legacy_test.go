package keeper_test

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"testing"

	"cosmossdk.io/log"
	"cosmossdk.io/store"
	storemetrics "cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/zerone-chain/zerone/x/knowledge/keeper"
	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// Characterization of knowledge-11 claim semantics at a4272bfa. It is NOT an
// approval of them. Each test reproduces, in process, one defect observed by
// the 2026-10-16 sandbox probes (P1, P1ext, P2, P3, P5, P6) and named in
// docs/plans/2026-10-16-claims-attribution-verdict-integrity.md. The repairs
// are gated by the claim-integrity marker (0x87); with that marker off, every
// assertion here and the golden store digest must keep holding byte for byte.
//
// These run the real keeper, message handlers and phase advancement on real
// SDK stores. The tracking bank records calls, not transfers or custody.
// AdvanceRoundPhases stands in for BeginBlock; skipped heights run nothing.

const (
	v11ReviewFee  = "1000000"
	v11Confidence = uint64(800_000)
)

type v11Fixture struct {
	k      keeper.Keeper
	ctx    sdk.Context
	bank   *trackingBankKeeper
	key    *storetypes.KVStoreKey
	server types.MsgServer
}

// setupV11SemanticsTest is DefaultGenesis with the four knowledge-11
// selections on, as on zerone-dev-1. No bootstrap funding and no seeded facts:
// every record below is created through a message handler or a round.
func setupV11SemanticsTest(t *testing.T) *v11Fixture {
	t.Helper()
	gs := types.DefaultGenesis()
	gs.RecordIntegrityEnabled = true
	gs.ReviewNeutralityEnabled = true
	gs.ClaimRecordsEnabled = true
	gs.FundSettlementEnabled = true
	gs.ClaimIntegrityEnabled = false // knowledge 11 has no claim-integrity marker.
	// Nor the consensus-repairs-v1 marker (0x90) that native genesis now sets.
	gs.ReformulationVindicationOrderingEnabled = false
	// Nor the consensus-repairs-v2 markers (0x91 onwards).
	gs.InconclusiveRoundAccountingEnabled = false
	gs.CalibrationSmoothingEnabled = false

	db := dbm.NewMemDB()
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	ms := store.NewCommitMultiStore(db, log.NewNopLogger(), storemetrics.NewNoOpMetrics())
	ms.SetIAVLSyncPruning(true)
	key := storetypes.NewKVStoreKey(types.StoreKey)
	ms.MountStoreWithDB(key, storetypes.StoreTypeIAVL, db)
	require.NoError(t, ms.LoadLatestVersion())
	ctx := sdk.NewContext(ms, cmtproto.Header{ChainID: "v11-semantics-test", Height: 100}, false, log.NewNopLogger())
	bank := newTrackingBankKeeper()
	k := keeper.NewKeeper(runtime.NewKVStoreService(key), codec.NewProtoCodec(codectypes.NewInterfaceRegistry()), "authority", bank, newTrackingStakingKeeper())
	require.NoError(t, k.InitGenesis(ctx, gs))
	return &v11Fixture{k: k, ctx: ctx.WithEventManager(sdk.NewEventManager()), bank: bank, key: key, server: keeper.NewMsgServerImpl(k)}
}

// v11Addr is a deterministic account; seeds longer than 20 bytes would alias.
func v11Addr(seed string) string {
	if len(seed) > 20 {
		panic("v11Addr seed exceeds the 20-byte address: " + seed)
	}
	return makeValidBech32Addr(seed)
}

func v11Reviewers() []string {
	return []string{v11Addr("v11-reviewer-1"), v11Addr("v11-reviewer-2"), v11Addr("v11-reviewer-3")}
}

func (f *v11Fixture) advanceTo(t *testing.T, height uint64) {
	t.Helper()
	require.GreaterOrEqual(t, height, uint64(f.ctx.BlockHeight()), "fixture height never moves backwards")
	f.ctx = f.ctx.WithBlockHeight(int64(height))
}

func (f *v11Fixture) beginBlock(t *testing.T) {
	t.Helper()
	require.NoError(t, f.k.AdvanceRoundPhases(f.ctx))
}

func (f *v11Fixture) submitClaim(author, text, domain string, mutate func(*types.MsgSubmitClaim)) (string, error) {
	msg := &types.MsgSubmitClaim{Submitter: author, FactContent: text, Domain: domain, Category: "empirical", Stake: v11ReviewFee}
	if mutate != nil {
		mutate(msg)
	}
	response, err := f.server.SubmitClaim(f.ctx, msg)
	if err != nil {
		return "", err
	}
	return response.ClaimId, nil
}

func (f *v11Fixture) mustSubmitClaim(t *testing.T, author, text, domain string, mutate func(*types.MsgSubmitClaim)) string {
	t.Helper()
	id, err := f.submitClaim(author, text, domain, mutate)
	require.NoError(t, err)
	return id
}

func (f *v11Fixture) contradict(submitter, factID, counter, domain, stake string) (string, error) {
	response, err := f.server.SubmitContradiction(f.ctx, &types.MsgSubmitContradiction{
		Submitter: submitter, FactId: factID, CounterClaim: counter, Domain: domain, Category: "empirical", Stake: stake,
	})
	if err != nil {
		return "", err
	}
	// The response field is named counter_fact_id but carries the counter-claim ID.
	return response.CounterFactId, nil
}

func (f *v11Fixture) challenge(t *testing.T, challenger, factID string) string {
	t.Helper()
	params, err := f.k.GetParams(f.ctx)
	require.NoError(t, err)
	stake := keeper.EffectiveMinChallengeStake(params, f.fact(t, factID).Confidence).String()
	response, err := f.server.ChallengeFact(f.ctx, &types.MsgChallengeFact{
		Challenger: challenger, FactId: factID, Stake: stake, Reason: "Synthetic v11 characterization challenge of this fact.",
	})
	require.NoError(t, err)
	round, found := f.k.GetVerificationRound(f.ctx, response.RoundId)
	require.True(t, found)
	return round.ClaimId
}

func (f *v11Fixture) claim(t *testing.T, id string) *types.Claim {
	t.Helper()
	claim, found := f.k.GetClaim(f.ctx, id)
	require.True(t, found, "claim %s", id)
	return claim
}

func (f *v11Fixture) fact(t *testing.T, id string) *types.Fact {
	t.Helper()
	fact, found := f.k.GetFact(f.ctx, id)
	require.True(t, found, "fact %s", id)
	return fact
}

func (f *v11Fixture) roundOf(t *testing.T, claimID string) *types.VerificationRound {
	t.Helper()
	round, found := f.k.GetVerificationRound(f.ctx, f.claim(t, claimID).VerificationRoundId)
	require.True(t, found, "round of claim %s", claimID)
	return round
}

// decide seats a panel that commits and reveals one vote, then aggregates on
// the next block. It returns the completed round.
func (f *v11Fixture) decide(t *testing.T, claimID, vote string, reviewers ...string) *types.VerificationRound {
	t.Helper()
	round := f.roundOf(t, claimID)
	require.Equal(t, types.VerificationPhase_VERIFICATION_PHASE_COMMIT, round.Phase)
	attestation := &types.ReviewAttestation{Reason: "Synthetic v11 characterization review of the stated record only."}
	for i, reviewer := range reviewers {
		f.bank.balances[reviewer] = sdk.NewCoins(sdk.NewInt64Coin("uzrn", 100_000_000))
		salt := []byte(fmt.Sprintf("v11-review-salt-%02d", i))
		hash, err := types.ComputeReviewCommitmentV2(round.CommitmentChainId, round.Id, reviewer, vote, v11Confidence, salt, attestation)
		require.NoError(t, err)
		_, err = f.server.SubmitCommitment(f.ctx, &types.MsgSubmitCommitment{Verifier: reviewer, RoundId: round.Id, CommitHash: hash})
		require.NoError(t, err)
	}
	f.advanceTo(t, round.CommitDeadline)
	f.beginBlock(t)
	for i, reviewer := range reviewers {
		_, err := f.server.SubmitReveal(f.ctx, &types.MsgSubmitReveal{
			Verifier: reviewer, RoundId: round.Id, Vote: vote, Confidence: v11Confidence,
			Salt: []byte(fmt.Sprintf("v11-review-salt-%02d", i)), Attestation: attestation,
		})
		require.NoError(t, err)
	}
	f.advanceTo(t, round.CommitDeadline+1)
	f.beginBlock(t)
	done := f.roundOf(t, claimID)
	require.Equal(t, types.VerificationPhase_VERIFICATION_PHASE_COMPLETE, done.Phase)
	return done
}

// accept runs a three-reviewer ACCEPT panel and returns the created fact ID.
func (f *v11Fixture) accept(t *testing.T, claimID string, reviewers ...string) string {
	t.Helper()
	if len(reviewers) == 0 {
		reviewers = v11Reviewers()
	}
	round := f.decide(t, claimID, "accept", reviewers...)
	require.Equal(t, types.Verdict_VERDICT_ACCEPT, round.Verdict)
	require.Equal(t, types.ClaimStatus_CLAIM_STATUS_ACCEPTED, f.claim(t, claimID).Status)
	factID := keeper.GenerateFactID(claimID, round.VerdictBlock)
	f.fact(t, factID)
	return factID
}

// starve lets a round reach its reveal deadline with no panel seated.
func (f *v11Fixture) starve(t *testing.T, claimID string) {
	t.Helper()
	round := f.roundOf(t, claimID)
	require.Empty(t, round.Commits)
	f.advanceTo(t, round.RevealDeadline)
	f.beginBlock(t)
	done := f.roundOf(t, claimID)
	require.Equal(t, types.VerificationPhase_VERIFICATION_PHASE_COMPLETE, done.Phase)
	require.Equal(t, types.Verdict_VERDICT_INCONCLUSIVE, done.Verdict)
	require.Equal(t, types.ClaimStatus_CLAIM_STATUS_INSUFFICIENT, f.claim(t, claimID).Status)
}

func (f *v11Fixture) requireNoLiveRounds(t *testing.T) {
	t.Helper()
	live := 0
	f.k.IterateActiveRounds(f.ctx, func(*types.VerificationRound) bool { live++; return false })
	require.Zero(t, live, "a characterization must not leak a live round into the next one")
}

func (f *v11Fixture) contentIndex(t *testing.T, text, domain string) string {
	t.Helper()
	id, found := f.k.GetClaimByContentHash(f.ctx, keeper.ComputeClaimContentHash(text, domain))
	require.True(t, found)
	return id
}

// storeDigest is sha256 over uvarint(len k)‖k‖uvarint(len v)‖v for every
// knowledge-store entry in key order.
func (f *v11Fixture) storeDigest(t *testing.T) (string, int) {
	t.Helper()
	h := sha256.New()
	var n [binary.MaxVarintLen64]byte
	entries := 0
	it := f.ctx.KVStore(f.key).Iterator(nil, nil)
	defer func() { require.NoError(t, it.Close()) }()
	for ; it.Valid(); it.Next() {
		key, value := it.Key(), it.Value()
		h.Write(n[:binary.PutUvarint(n[:], uint64(len(key)))])
		h.Write(key)
		h.Write(n[:binary.PutUvarint(n[:], uint64(len(value)))])
		h.Write(value)
		entries++
	}
	return hex.EncodeToString(h.Sum(nil)), entries
}

func v11Event(ctx sdk.Context, eventType string) []map[string]string {
	var out []map[string]string
	for _, event := range ctx.EventManager().Events() {
		if event.Type != eventType {
			continue
		}
		attrs := map[string]string{}
		for _, attr := range event.Attributes {
			attrs[attr.Key] = attr.Value
		}
		out = append(out, attrs)
	}
	return out
}

type v11Transition struct {
	prior, next    types.FactStatus
	cause, causeID string
}

func (f *v11Fixture) transitions(factID string) []v11Transition {
	var out []v11Transition
	for _, tr := range f.k.GetStatusHistory(f.ctx, factID) {
		out = append(out, v11Transition{tr.PriorStatus, tr.NewStatus, tr.CauseEventType, tr.CauseId})
	}
	return out
}

// ─── Scenarios (shared by the named tests and the golden digest) ─────────────

// P1: a starved claim keeps the content-hash index, so its exact text in its
// domain can never be submitted again. Only exact text is locked.
func v11StarvedClaimLocksExactText(t *testing.T, f *v11Fixture) {
	const text = "Synthetic v11 probe P1: the starve-then-resubmit fixture statement X, left without reviewers."
	x := f.mustSubmitClaim(t, v11Addr("v11-p1-author"), text, "physics", nil)
	f.starve(t, x)
	require.Equal(t, x, f.contentIndex(t, text, "physics"))

	before, _ := f.storeDigest(t)
	_, err := f.submitClaim(v11Addr("v11-p1-resubmitter"), text, "physics", nil)
	require.EqualError(t, err, "duplicate claim: content hash matches existing claim "+x)
	after, _ := f.storeDigest(t)
	require.Equal(t, before, after, "the refusal writes nothing")
	require.Equal(t, types.ClaimStatus_CLAIM_STATUS_INSUFFICIENT, f.claim(t, x).Status)

	// The lock is exact: a one-character variant, or the same text in another
	// domain, has a different content hash and is admitted.
	variant := f.mustSubmitClaim(t, v11Addr("v11-p1-variant"), text+" ", "physics", nil)
	elsewhere := f.mustSubmitClaim(t, v11Addr("v11-p1-other-domain"), text, "chemistry", nil)
	require.NotEqual(t, x, variant)
	f.starve(t, variant)
	f.starve(t, elsewhere)
	f.requireNoLiveRounds(t)
}

// P1ext: MsgSubmitContradiction has no content-hash check. A counter-claim
// carrying a live claim's exact text is admitted and takes over its index
// entry; once both end, the text stays locked under the contradiction's ID.
func v11ContradictionOverwritesContentIndex(t *testing.T, f *v11Fixture) {
	const text = "Synthetic v11 probe P1ext: statement X, later copied verbatim into a contradiction."
	target := f.accept(t, f.mustSubmitClaim(t, v11Addr("v11-p1x-target"), "Synthetic v11 probe P1ext: an accepted target fact for the contradiction.", "physics", nil))
	x := f.mustSubmitClaim(t, v11Addr("v11-p1x-author"), text, "physics", nil)
	require.Equal(t, x, f.contentIndex(t, text, "physics"))

	f.advanceTo(t, uint64(f.ctx.BlockHeight())+1)
	c, err := f.contradict(v11Addr("v11-p1x-contradict"), target, text, "physics", "1")
	require.NoError(t, err, "a 1-uzrn contradiction carrying a live claim's exact text is admitted")
	require.NotEqual(t, x, c)
	require.Equal(t, c, f.contentIndex(t, text, "physics"), "the contradiction overwrites the live claim's 0x09 entry")
	require.Equal(t, types.ClaimStatus_CLAIM_STATUS_IN_VERIFICATION, f.claim(t, x).Status)
	require.Equal(t, f.claim(t, x).ContentHash, f.claim(t, c).ContentHash)
	require.Equal(t, types.FactStatus_FACT_STATUS_CONTESTED, f.fact(t, target).Status)

	f.starve(t, x)
	f.starve(t, c)
	require.Equal(t, types.FactStatus_FACT_STATUS_VERIFIED, f.fact(t, target).Status)
	_, err = f.submitClaim(v11Addr("v11-p1x-resubmit"), text, "physics", nil)
	require.EqualError(t, err, "duplicate claim: content hash matches existing claim "+c)
	f.requireNoLiveRounds(t)
}

// P2 B1: the v1 canonical form drops object and temporal_scope from an
// ASSERTION and lowercases every argument, so distinct findings collide.
func v11CanonicalDropsObject(t *testing.T, f *v11Fixture) {
	structure := func(object, temporal, scope string) *types.ClaimStructure {
		return &types.ClaimStructure{Subject: "boiling point of compound Z", Predicate: "equals degrees C", Object: object, Scope: scope, TemporalScope: temporal}
	}
	const form = `assert("chemistry", "boiling point of compound z", "equals degrees c", "pure sample at 1 atm")`
	assertion := types.ClaimType_CLAIM_TYPE_ASSERTION
	require.Equal(t, form, types.BuildCanonicalForm(assertion, structure("78.4", "", "pure sample at 1 atm"), "chemistry"))
	require.Equal(t, form, types.BuildCanonicalForm(assertion, structure("81.0", "2021-01-01", "pure sample at 1 atm"), "chemistry"))
	require.Equal(t,
		types.BuildCanonicalForm(assertion, structure("5", "", "output 5 mW"), "chemistry"),
		types.BuildCanonicalForm(assertion, structure("5", "", "output 5 MW"), "chemistry"))

	labA := f.mustSubmitClaim(t, v11Addr("v11-p2-lab-a"), "Synthetic v11 probe P2a: Lab A reports the boiling point of compound Z at 1 atm is 78.4 C.", "chemistry",
		func(m *types.MsgSubmitClaim) {
			m.ClaimType = assertion
			m.Structure = structure("78.4", "", "pure sample at 1 atm")
		})
	require.Equal(t, form, f.claim(t, labA).CanonicalForm)
	labB := func() error {
		_, err := f.submitClaim(v11Addr("v11-p2-lab-b"), "Synthetic v11 probe P2b: Lab B reports the boiling point of compound Z at 1 atm is 81.0 C.", "chemistry",
			func(m *types.MsgSubmitClaim) {
				m.ClaimType = assertion
				m.Structure = structure("81.0", "", "pure sample at 1 atm")
			})
		return err
	}
	require.EqualError(t, labB(), "canonical duplicate: matches existing claim "+labA)
	f.starve(t, labA)
	require.EqualError(t, labB(), "canonical duplicate: matches existing claim "+labA, "the 0x34 entry outlives a starved claim")
	f.requireNoLiveRounds(t)
}

// P2 S1/S2: a canonical form without a structure is never checked, so any
// claim can reserve a structured claim's canonical slot.
func v11UnstructuredCanonicalSquat(t *testing.T, f *v11Fixture) {
	squat := f.mustSubmitClaim(t, v11Addr("v11-p2-squatter"), "Synthetic v11 probe P2s: an unrelated sentence that reserves a canonical form without any structure.", "chemistry",
		func(m *types.MsgSubmitClaim) {
			m.CanonicalForm = `assert("chemistry", "melting point of compound z", "equals degrees c", "pure sample at 1 atm")`
		})
	require.Nil(t, f.claim(t, squat).Structure)
	require.NotEmpty(t, f.claim(t, squat).CanonicalHash)

	before, _ := f.storeDigest(t)
	_, err := f.submitClaim(v11Addr("v11-p2-lab-c"), "Synthetic v11 probe P2s: Lab C reports the melting point of compound Z at 1 atm is minus 114 C.", "chemistry",
		func(m *types.MsgSubmitClaim) {
			m.ClaimType = types.ClaimType_CLAIM_TYPE_ASSERTION
			m.Structure = &types.ClaimStructure{Subject: "Melting point of compound Z", Predicate: "equals degrees C", Object: "-114", Scope: "pure sample at 1 atm"}
		})
	require.EqualError(t, err, "canonical duplicate: matches existing claim "+squat)
	after, _ := f.storeDigest(t)
	require.Equal(t, before, after)
	f.starve(t, squat)
	f.requireNoLiveRounds(t)
}

// P3: a contradiction flips its target at admission. The reversal path never
// consults the accepted counter-fact K or its CONTRADICTS edge, so a 1-uzrn B
// that starves restores F to VERIFIED while K still stands.
func v11StarvedContradictionRestoresUpheldTarget(t *testing.T, f *v11Fixture) {
	reviewers := v11Reviewers()
	factF := f.accept(t, f.mustSubmitClaim(t, v11Addr("v11-p3-author"), "Synthetic v11 probe P3: compound Q reduced tumour volume in mice by 30 percent.", "biology", nil), reviewers...)

	a, err := f.contradict(v11Addr("v11-p3-jones"), factF, "Synthetic v11 probe P3: a replication found compound Q did not reduce tumour volume.", "biology", v11ReviewFee)
	require.NoError(t, err)
	require.Equal(t, types.FactStatus_FACT_STATUS_CONTESTED, f.fact(t, factF).Status, "admission alone contests F")
	factK := f.accept(t, a, reviewers...)
	require.Equal(t, types.FactStatus_FACT_STATUS_VERIFIED, f.fact(t, factK).Status)
	require.Equal(t, types.FactStatus_FACT_STATUS_CONTESTED, f.fact(t, factF).Status)
	incoming, err := f.k.GetIncomingRelations(f.ctx, factF)
	require.NoError(t, err)
	require.Len(t, incoming, 1)
	require.Equal(t, factK, incoming[0].SourceFactId)
	require.Equal(t, types.RelationType_RELATION_TYPE_CONTRADICTS, incoming[0].Relation)

	f.advanceTo(t, uint64(f.ctx.BlockHeight())+1)
	b, err := f.contradict(v11Addr("v11-p3-griefer"), factF, "Synthetic v11 probe P3: a 1-uzrn contradiction left deliberately without reviewers.", "biology", "1")
	require.NoError(t, err, "1 uzrn is enough collateral")
	require.Equal(t, "1", f.claim(t, b).FundingTerms.PaidAmount)
	require.Equal(t, types.FactStatus_FACT_STATUS_CONTESTED, f.fact(t, factF).Status)

	f.ctx = f.ctx.WithEventManager(sdk.NewEventManager())
	f.starve(t, b)
	require.Equal(t, types.FactStatus_FACT_STATUS_VERIFIED, f.fact(t, factF).Status, "a starved contradiction launders the upheld one")
	require.Equal(t, []map[string]string{{"fact_id": factF, "reverted_by_claim": b}}, v11Event(f.ctx, "zerone.knowledge.contradiction_reversed"))
	require.Equal(t, types.FactStatus_FACT_STATUS_VERIFIED, f.fact(t, factK).Status, "K still stands")
	require.Equal(t, types.ClaimStatus_CLAIM_STATUS_ACCEPTED, f.claim(t, a).Status)
	incoming, err = f.k.GetIncomingRelations(f.ctx, factF)
	require.NoError(t, err)
	require.Len(t, incoming, 1, "the standing CONTRADICTS edge is unchanged")
	f.requireNoLiveRounds(t)
}

// P5: a submitter may sit on the panel for their own claim, and a fact's
// submitter may sit on the panel that judges a challenge to it. Both are paid.
func v11SubmitterMayReviewOwnClaim(t *testing.T, f *v11Fixture) {
	author := v11Addr("v11-p5-author")
	reviewers := v11Reviewers()
	panel := []string{author, reviewers[0], reviewers[1]}
	paid := func(round *types.VerificationRound) []string {
		require.NotNil(t, round.VerifierRewardSettlement)
		var out []string
		for _, payment := range round.VerifierRewardSettlement.Payments {
			out = append(out, payment.Verifier)
		}
		return out
	}

	claimT := f.mustSubmitClaim(t, author, "Synthetic v11 probe P5: the author of this claim also reviews it.", "physics", nil)
	factT := f.accept(t, claimT, panel...)
	round := f.roundOf(t, claimT)
	require.Equal(t, author, round.Commits[0].Verifier, "the submitter's commit is admitted")
	require.Equal(t, author, round.Reveals[0].Verifier)
	require.Contains(t, paid(round), author, "and the submitter is paid for reviewing their own claim")

	challengeT := f.challenge(t, v11Addr("v11-p5-challenger"), factT)
	require.Equal(t, types.FactStatus_FACT_STATUS_CHALLENGED, f.fact(t, factT).Status)
	survived := f.decide(t, challengeT, "reject", panel...)
	require.Equal(t, types.Verdict_VERDICT_REJECT, survived.Verdict)
	require.Equal(t, author, survived.Commits[0].Verifier, "the defender judges the challenge to their own fact")
	require.Contains(t, paid(survived), author)
	require.Equal(t, types.ClaimStatus_CLAIM_STATUS_REJECTED, f.claim(t, challengeT).Status)
	require.Equal(t, types.FactStatus_FACT_STATUS_ACTIVE, f.fact(t, factT).Status)
	f.requireNoLiveRounds(t)
}

// P6: SetFact infers a cause from the new status alone. A contradiction flip
// is recorded as a cascade with no cause ID, its reversal as the fact's own
// verification, challenge open/restore/survival as "unknown", and the fall as
// challenge_disproven with no cause ID. Only the cascade to dependants carries
// an exact cause.
func v11TransitionCausesInferred(t *testing.T, f *v11Fixture) {
	reviewers := v11Reviewers()
	claimG := f.mustSubmitClaim(t, v11Addr("v11-p6-author"), "Synthetic v11 probe P6: fact G whose history records every status change.", "physics", nil)
	factG := f.accept(t, claimG, reviewers...)
	claimH := f.mustSubmitClaim(t, v11Addr("v11-p6-dependent"), "Synthetic v11 probe P6: fact H, which requires fact G to hold.", "physics",
		func(m *types.MsgSubmitClaim) {
			m.Relations = []*types.ClaimRelation{{Relation: types.RelationType_RELATION_TYPE_REQUIRES, TargetFactId: factG}}
		})
	factH := f.accept(t, claimH, reviewers...)

	contradiction, err := f.contradict(v11Addr("v11-p6-contradict"), factG, "Synthetic v11 probe P6: a staked contradiction of G that starves.", "physics", v11ReviewFee)
	require.NoError(t, err)
	f.starve(t, contradiction)
	relationClaim := f.mustSubmitClaim(t, v11Addr("v11-p6-relation"), "Synthetic v11 probe P6: a claim carrying a CONTRADICTS relation to G.", "physics",
		func(m *types.MsgSubmitClaim) {
			m.Relations = []*types.ClaimRelation{{Relation: types.RelationType_RELATION_TYPE_CONTRADICTS, TargetFactId: factG}}
		})
	require.Equal(t, types.FactStatus_FACT_STATUS_CONTESTED, f.fact(t, factG).Status)
	f.starve(t, relationClaim)

	f.starve(t, f.challenge(t, v11Addr("v11-p6-challenger-1"), factG))
	require.Equal(t, types.FactStatus_FACT_STATUS_ACTIVE, f.fact(t, factG).Status)
	survived := f.challenge(t, v11Addr("v11-p6-challenger-2"), factG)
	require.Equal(t, types.Verdict_VERDICT_REJECT, f.decide(t, survived, "reject", reviewers...).Verdict)
	disproving := f.challenge(t, v11Addr("v11-p6-challenger-3"), factG)
	f.accept(t, disproving, reviewers...)
	require.Equal(t, types.FactStatus_FACT_STATUS_DISPROVEN, f.fact(t, factG).Status)
	require.Equal(t, types.FactStatus_FACT_STATUS_CONTESTED, f.fact(t, factH).Status)

	const (
		unspecified = types.FactStatus_FACT_STATUS_UNSPECIFIED
		verified    = types.FactStatus_FACT_STATUS_VERIFIED
		contested   = types.FactStatus_FACT_STATUS_CONTESTED
		challenged  = types.FactStatus_FACT_STATUS_CHALLENGED
		active      = types.FactStatus_FACT_STATUS_ACTIVE
		disproven   = types.FactStatus_FACT_STATUS_DISPROVEN
	)
	require.Equal(t, []v11Transition{
		{unspecified, verified, "verification", claimG},
		{verified, contested, "cascade", ""},               // MsgSubmitContradiction admission
		{contested, verified, "verification", claimG},      // its starved reversal
		{verified, contested, "cascade", ""},               // SubmitClaim+CONTRADICTS admission
		{contested, verified, "verification", claimG},      // its starved reversal
		{verified, challenged, "unknown", ""},              // challenge 1 opened
		{challenged, active, "unknown", ""},                // challenge 1 starved
		{active, challenged, "unknown", ""},                // challenge 2 opened
		{challenged, active, "unknown", ""},                // challenge 2 rejected (survived)
		{active, challenged, "unknown", ""},                // challenge 3 opened
		{challenged, disproven, "challenge_disproven", ""}, // challenge 3 accepted
	}, f.transitions(factG))
	require.Equal(t, []v11Transition{
		{unspecified, verified, "verification", claimH},
		{verified, contested, "cascade", factG},
	}, f.transitions(factH))
	f.requireNoLiveRounds(t)
}

var v11Scenarios = []struct {
	name string
	run  func(*testing.T, *v11Fixture)
}{
	{"StarvedClaimLocksExactText", v11StarvedClaimLocksExactText},
	{"ContradictionOverwritesContentIndex", v11ContradictionOverwritesContentIndex},
	{"CanonicalDropsObject", v11CanonicalDropsObject},
	{"UnstructuredCanonicalSquat", v11UnstructuredCanonicalSquat},
	{"StarvedContradictionRestoresUpheldTarget", v11StarvedContradictionRestoresUpheldTarget},
	{"SubmitterMayReviewOwnClaim", v11SubmitterMayReviewOwnClaim},
	{"TransitionCausesInferred", v11TransitionCausesInferred},
}

func TestV11_StarvedClaimLocksExactText(t *testing.T) {
	v11StarvedClaimLocksExactText(t, setupV11SemanticsTest(t))
}

func TestV11_ContradictionOverwritesContentIndex(t *testing.T) {
	v11ContradictionOverwritesContentIndex(t, setupV11SemanticsTest(t))
}

func TestV11_CanonicalDropsObject(t *testing.T) {
	v11CanonicalDropsObject(t, setupV11SemanticsTest(t))
}

func TestV11_UnstructuredCanonicalSquat(t *testing.T) {
	v11UnstructuredCanonicalSquat(t, setupV11SemanticsTest(t))
}

func TestV11_StarvedContradictionRestoresUpheldTarget(t *testing.T) {
	v11StarvedContradictionRestoresUpheldTarget(t, setupV11SemanticsTest(t))
}

func TestV11_SubmitterMayReviewOwnClaim(t *testing.T) {
	v11SubmitterMayReviewOwnClaim(t, setupV11SemanticsTest(t))
}

func TestV11_TransitionCausesInferred(t *testing.T) {
	v11TransitionCausesInferred(t, setupV11SemanticsTest(t))
}

// v11GoldenKnowledgeStoreDigest was recorded on a4272bfa (knowledge v11): the
// whole knowledge store after every scenario above runs, in order, in one
// setupV11SemanticsTest fixture (493 entries at height 5511). With the
// claim-integrity marker off, no later change may move it.
const v11GoldenKnowledgeStoreDigest = "f849e0992025e8a25d1029dbaac3cb3c1439e095b3f722f0b7bda40f439ba7d3"

func TestV11_GoldenKnowledgeStoreDigest(t *testing.T) {
	f := setupV11SemanticsTest(t)
	for _, scenario := range v11Scenarios {
		require.True(t, t.Run(scenario.name, func(t *testing.T) { scenario.run(t, f) }), "scenario %s", scenario.name)
	}
	digest, entries := f.storeDigest(t)
	t.Logf("knowledge store after v11 scenarios: height=%d entries=%d sha256=%s", f.ctx.BlockHeight(), entries, digest)
	require.Equal(t, v11GoldenKnowledgeStoreDigest, digest)
}
