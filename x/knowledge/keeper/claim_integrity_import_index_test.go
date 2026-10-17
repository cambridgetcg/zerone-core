package keeper_test

import (
	"fmt"
	"slices"
	"sort"
	"testing"

	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/zerone-chain/zerone/x/knowledge/keeper"
	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// Claim integrity, import of the subject (0x32) and tag (0x33) indexes. A
// round writes both when it creates a fact (IndexFactBySubject), and nothing
// else writes them. The export carries no index, so on knowledge 11 an import
// restores every fact with its structure and tags but none of its entries:
// FactsByTag and FactsBySubject answer nothing, and the attribution lookup
// FactsByTag("zc:<40hex>") of plan §5.5 fails on any chain started from an
// export (final acceptance, e2e row 10). With the flag on InitGenesis rebuilds
// both indexes for every imported fact that carries a structure (plan §3.6).

const (
	importIndexDomain = dedupDomain
	importIndexShared = "import-index-shared"
	importIndexX      = "import index shared subject X"
	importIndexY      = "import index shared subject Y"
	importIndexM      = "import index sample M"
)

// importIndexStructured is a structured assertion on subject with one shared
// tag; object keeps the canonical forms of one subject apart.
func importIndexStructured(subject, object string, tags ...string) func(*types.MsgSubmitClaim) {
	return func(m *types.MsgSubmitClaim) {
		m.ClaimType = types.ClaimType_CLAIM_TYPE_ASSERTION
		m.Structure = &types.ClaimStructure{
			Subject: subject, Predicate: "has dry mass in grams", Object: object, Scope: "dry sample at 20 C",
			Tags: append([]string{importIndexShared}, tags...),
		}
	}
}

// acceptTogether runs one ACCEPT panel over claims submitted in the same
// block, so one AdvanceRoundPhases call completes every round and creates
// every fact at one height. It returns the fact IDs in argument order.
func (f *v11Fixture) acceptTogether(t *testing.T, claimIDs ...string) []string {
	t.Helper()
	reviewers := v11Reviewers()
	attestation := &types.ReviewAttestation{Reason: "Synthetic v12 import-index review of the stated record only."}
	salt := func(r, i int) []byte { return []byte(fmt.Sprintf("v12-ix-salt-%02d-%02d", r, i)) }
	rounds := make([]*types.VerificationRound, len(claimIDs))
	for r, claimID := range claimIDs {
		rounds[r] = f.roundOf(t, claimID)
		require.Equal(t, types.VerificationPhase_VERIFICATION_PHASE_COMMIT, rounds[r].Phase)
		require.Equal(t, rounds[0].CommitDeadline, rounds[r].CommitDeadline, "the claims share a block")
		for i, reviewer := range reviewers {
			f.bank.balances[reviewer] = sdk.NewCoins(sdk.NewInt64Coin("uzrn", 100_000_000))
			hash, err := types.ComputeReviewCommitmentV2(rounds[r].CommitmentChainId, rounds[r].Id, reviewer, "accept", v11Confidence, salt(r, i), attestation)
			require.NoError(t, err)
			_, err = f.server.SubmitCommitment(f.ctx, &types.MsgSubmitCommitment{Verifier: reviewer, RoundId: rounds[r].Id, CommitHash: hash})
			require.NoError(t, err)
		}
	}
	f.advanceTo(t, rounds[0].CommitDeadline)
	f.beginBlock(t)
	for r := range claimIDs {
		for i, reviewer := range reviewers {
			_, err := f.server.SubmitReveal(f.ctx, &types.MsgSubmitReveal{
				Verifier: reviewer, RoundId: rounds[r].Id, Vote: "accept", Confidence: v11Confidence,
				Salt: salt(r, i), Attestation: attestation,
			})
			require.NoError(t, err)
		}
	}
	f.advanceTo(t, rounds[0].CommitDeadline+1)
	f.beginBlock(t)
	facts := make([]string, len(claimIDs))
	for r, claimID := range claimIDs {
		done := f.roundOf(t, claimID)
		require.Equal(t, types.Verdict_VERDICT_ACCEPT, done.Verdict)
		require.Equal(t, uint64(f.ctx.BlockHeight()), done.VerdictBlock)
		facts[r] = keeper.GenerateFactID(claimID, done.VerdictBlock)
		f.fact(t, facts[r])
	}
	return facts
}

// factIndexEntries lists every subject and tag index entry, key and value, in
// key order.
func (f *v11Fixture) factIndexEntries(t *testing.T) []string {
	t.Helper()
	var entries []string
	for _, prefix := range [][]byte{types.FactSubjectPrefix, types.FactTagPrefix} {
		it := storetypes.KVStorePrefixIterator(f.ctx.KVStore(f.key), prefix)
		for ; it.Valid(); it.Next() {
			entries = append(entries, fmt.Sprintf("%x=%x", it.Key(), it.Value()))
		}
		require.NoError(t, it.Close())
	}
	return entries
}

type importIndexTag struct {
	tag   string
	facts []string
}

type importIndexSubject struct {
	subject, fact string
}

// requireFactIndexes asks the query server, as a consumer would, for every
// tag and subject. An empty fact names no fact.
func (f *v11Fixture) requireFactIndexes(t *testing.T, tags []importIndexTag, subjects []importIndexSubject) {
	t.Helper()
	query := keeper.NewQueryServerImpl(f.k)
	for _, want := range tags {
		response, err := query.FactsByTag(f.ctx, &types.QueryFactsByTagRequest{Tag: want.tag})
		require.NoError(t, err)
		var got []string
		for _, fact := range response.Facts {
			got = append(got, fact.Id)
		}
		require.Equal(t, want.facts, got, "FactsByTag(%q)", want.tag)
	}
	for _, want := range subjects {
		response, err := query.FactsBySubject(f.ctx, &types.QueryFactsBySubjectRequest{Domain: importIndexDomain, Subject: want.subject})
		require.NoError(t, err)
		var got string
		if len(response.Facts) > 0 {
			require.Len(t, response.Facts, 1)
			got = response.Facts[0].Id
		}
		require.Equal(t, want.fact, got, "FactsBySubject(%q)", want.subject)
	}
}

// subjectWarnings submits, on a discarded cache, a claim with subject X and
// the predicate of its facts, and returns the duplicate-subject warnings.
func (f *v11Fixture) subjectWarnings(t *testing.T) []map[string]string {
	t.Helper()
	var warnings []map[string]string
	f.inCache(func() {
		f.ctx = f.ctx.WithEventManager(sdk.NewEventManager())
		f.mustSubmitClaim(t, v11Addr("v12-ix-x-third"), "Synthetic v12 import index: sample X weighed a third time.", importIndexDomain, importIndexStructured(importIndexX, "1.7"))
		warnings = v11Event(f.ctx, "zerone.knowledge.duplicate_subject_warning")
	})
	return warnings
}

// Export then import keeps FactsByTag and FactsBySubject. On the live chain
// the one subject entry per domain and subject names the fact created last:
// later blocks win, and within one block AdvanceRoundPhases completes rounds
// in round-ID order. With the flag on InitGenesis replays the index writes in
// that creation order (VerifiedAtBlock, the round ID the fact's claim names,
// then the fact ID), so the imported entries equal the exporting chain's
// entry for entry whatever order the file lists the facts in, and admission
// again warns of the same duplicate subject. With the flag off nothing is
// rebuilt (knowledge 11).
func TestClaimIntegrity_ImportRebuildsSubjectAndTagIndexes(t *testing.T) {
	f := setupClaimIntegrityTest(t)
	attribution := f.accept(t, f.mustSubmitClaim(t, v11Addr("v12-ix-extractor"), g2AttrText, importIndexDomain, g2Attribution))
	multi := f.accept(t, f.mustSubmitClaim(t, v11Addr("v12-ix-multi"), "Synthetic v12 import index: sample M carries several tags.", importIndexDomain,
		importIndexStructured(importIndexM, "7.25", "Thermodynamics", " entropy ", "heat", "", "ENTROPY")))
	earlier := f.accept(t, f.mustSubmitClaim(t, v11Addr("v12-ix-x-first"), "Synthetic v12 import index: sample X weighed first.", importIndexDomain, importIndexStructured(importIndexX, "1.5")))
	later := f.accept(t, f.mustSubmitClaim(t, v11Addr("v12-ix-x-second"), "Synthetic v12 import index: sample X weighed again later.", importIndexDomain, importIndexStructured(importIndexX, "1.6")))
	require.Greater(t, f.fact(t, later).VerifiedAtBlock, f.fact(t, earlier).VerifiedAtBlock)
	yClaims := []string{
		f.mustSubmitClaim(t, v11Addr("v12-ix-y-one"), "Synthetic v12 import index: sample Y weighed by lab one (run 2).", importIndexDomain, importIndexStructured(importIndexY, "2.5")),
		f.mustSubmitClaim(t, v11Addr("v12-ix-y-two"), "Synthetic v12 import index: sample Y weighed by lab two (run 2).", importIndexDomain, importIndexStructured(importIndexY, "2.6")),
	}
	yFacts := f.acceptTogether(t, yClaims...)
	require.Equal(t, f.fact(t, yFacts[0]).VerifiedAtBlock, f.fact(t, yFacts[1]).VerifiedAtBlock)
	yLast, yGreaterID := yFacts[0], yFacts[0]
	if f.claim(t, yClaims[1]).VerificationRoundId > f.claim(t, yClaims[0]).VerificationRoundId {
		yLast = yFacts[1]
	}
	if yFacts[1] > yFacts[0] {
		yGreaterID = yFacts[1]
	}
	// The texts ("run 2") were chosen so the two orders disagree: an import
	// that broke the tie by fact ID alone would name the other fact.
	require.NotEqual(t, yGreaterID, yLast, "the round created last does not hold the greater fact ID")
	require.Equal(t, yLast, f.k.FindFactBySubjectPredicate(f.ctx, importIndexDomain, importIndexY, ""), "live: the round completed last in the block writes last")
	f.requireNoLiveRounds(t)

	shared := []string{earlier, later, yFacts[0], yFacts[1], multi}
	sort.Strings(shared) // FactsByTag lists a tag's facts in fact-ID order.
	tags := []importIndexTag{
		{"zc:f789edd890d61613208c09ebc74248b09c2f170e", []string{attribution}},
		{"ZC:F789EDD890D61613208C09EBC74248B09C2F170E", []string{attribution}},
		{"attr-v1", []string{attribution}},
		{"thermodynamics", []string{multi}},
		{"entropy", []string{multi}},
		{"heat", []string{multi}},
		{importIndexShared, shared},
	}
	subjects := []importIndexSubject{
		{g2AttrSubject, attribution},
		{importIndexM, multi},
		{importIndexX, later},
		{importIndexY, yLast},
	}
	f.requireFactIndexes(t, tags, subjects)
	live := f.factIndexEntries(t)
	require.Len(t, live, 4+3+3+5, "four subject entries; tag entries: three of the attribution, three more of M, the shared tag of five facts")
	require.Equal(t, []map[string]string{{"existing_fact_id": later, "subject": importIndexX}}, f.subjectWarnings(t))
	exported := f.k.ExportGenesis(f.ctx)

	var none []importIndexTag
	for _, tag := range tags {
		none = append(none, importIndexTag{tag: tag.tag})
	}
	var unnamed []importIndexSubject
	for _, subject := range subjects {
		unnamed = append(unnamed, importIndexSubject{subject: subject.subject})
	}
	for _, on := range []bool{true, false} {
		for _, reversed := range []bool{false, true} {
			t.Run(fmt.Sprintf("flag=%t/reversed=%t", on, reversed), func(t *testing.T) {
				gs := proto.Clone(exported).(*types.GenesisState)
				require.True(t, gs.ClaimIntegrityEnabled)
				gs.ClaimIntegrityEnabled = on
				if reversed {
					slices.Reverse(gs.Facts)
				}
				var listed []string
				for _, fact := range gs.Facts {
					listed = append(listed, fact.Id)
				}
				g := newClaimIntegrityGenesisFixture(t, "claim-integrity-index-import", gs)
				for i, fact := range gs.Facts {
					require.Equal(t, listed[i], fact.Id, "InitGenesis does not reorder the caller's file")
				}
				g.advanceTo(t, uint64(f.ctx.BlockHeight()))
				for _, id := range shared {
					require.NotNil(t, g.fact(t, id).Structure, "the fact and its structure are imported either way")
				}
				if !on {
					require.Empty(t, g.factIndexEntries(t), "knowledge 11 rebuilds no index on import")
					g.requireFactIndexes(t, none, unnamed)
					require.Empty(t, g.subjectWarnings(t))
					return
				}
				require.Equal(t, live, g.factIndexEntries(t), "the imported indexes equal the exporting chain's, entry for entry")
				g.requireFactIndexes(t, tags, subjects)
				require.Equal(t, []map[string]string{{"existing_fact_id": later, "subject": importIndexX}}, g.subjectWarnings(t))
			})
		}
	}
}
