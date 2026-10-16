package keeper_test

import (
	"fmt"
	"slices"
	"sort"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/zerone-chain/zerone/x/knowledge/keeper"
	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// Claim integrity, Task 8 (P1): a dedup index entry (0x09 content hash or
// 0x34 canonical hash) blocks admission only while the record it names is
// live or standing (plan §5.4). With the marker on, only a claim's first save
// writes its entries, and an accepted claim's fact keeps the 0x34 entry. Each
// behaviour also runs with the marker off, where knowledge 11's permanent lock
// must still reproduce.
//
// Reviewers and challengers are never parties, no scenario submits a
// contradiction or an explicit canonical form, and canonical hashes are read
// from the stored claim instead of being spelled out. Later v12 tasks (9-14)
// therefore do not change what these tests observe, with one exception: once
// Task 9 made marker-on hashes v2, the flag-off import branch of
// TestClaimIntegrity_ImportReplaysAdmissionOrder can no longer submit a v1
// claim that meets an exported v2 canonical entry.

const (
	dedupDomain = "physics"
	// dedupNowhere is a well-formed record ID that names no record.
	dedupNowhere = "0123456789abcdef0123456789abcdef"
)

var dedupSelections = []struct {
	name      string
	setup     func(*testing.T) *v11Fixture
	integrity bool
}{
	{"marker on", setupClaimIntegrityTest, true},
	{"marker off", setupV11SemanticsTest, false},
}

func dedupStructured(subject string) func(*types.MsgSubmitClaim) {
	return func(m *types.MsgSubmitClaim) {
		m.ClaimType = types.ClaimType_CLAIM_TYPE_ASSERTION
		m.Structure = &types.ClaimStructure{Subject: subject, Predicate: "has dry mass in grams", Object: "12.5", Scope: "dry sample at 20 C"}
	}
}

// entry reads a raw dedup index entry; "" when absent.
func (f *v11Fixture) entry(key []byte) string {
	return string(f.ctx.KVStore(f.key).Get(key))
}

func dedupContentKey(text string) []byte {
	return types.ContentHashKey(keeper.ComputeClaimContentHash(text, dedupDomain))
}

// inCache runs fn on a discarded cache of the fixture's state.
func (f *v11Fixture) inCache(fn func()) {
	saved := f.ctx
	f.ctx, _ = saved.CacheContext()
	defer func() { f.ctx = saved }()
	fn()
}

// requireSubmitRefused requires an exact refusal that writes nothing.
func (f *v11Fixture) requireSubmitRefused(t *testing.T, author, text string, mutate func(*types.MsgSubmitClaim), want string) {
	t.Helper()
	before, _ := f.storeDigest(t)
	_, err := f.submitClaim(author, text, dedupDomain, mutate)
	require.EqualError(t, err, want)
	after, _ := f.storeDigest(t)
	require.Equal(t, before, after, "a refusal writes nothing")
}

// canonicalHashOf derives a structured claim's canonical hash by admitting it
// in a discarded cache, so the test never spells out a canonical form.
func (f *v11Fixture) canonicalHashOf(t *testing.T, text string, mutate func(*types.MsgSubmitClaim)) string {
	t.Helper()
	var hash string
	f.inCache(func() {
		hash = f.claim(t, f.mustSubmitClaim(t, v11Addr("v12-dd-hash-probe"), text, dedupDomain, mutate)).CanonicalHash
	})
	require.NotEmpty(t, hash)
	return hash
}

// P1 replica. A starved claim releases its exact text; the resubmission's
// first save takes the 0x09 entry and, while live, blocks in turn with a
// refusal that names it and its status. Re-saves of either claim never move
// the entry.
func TestClaimIntegrity_StarvedClaimAllowsResubmission(t *testing.T) {
	for _, sel := range dedupSelections {
		t.Run(sel.name, func(t *testing.T) {
			f := sel.setup(t)
			require.Equal(t, sel.integrity, f.claimIntegrityEnabled(t))
			const text = "Synthetic v12 probe P1: statement X, starved and then submitted again verbatim."
			x := f.mustSubmitClaim(t, v11Addr("v12-p1-author"), text, dedupDomain, nil)
			f.starve(t, x)
			require.Equal(t, x, f.entry(dedupContentKey(text)))

			if !sel.integrity {
				f.requireSubmitRefused(t, v11Addr("v12-p1-resubmit"), text, nil,
					"duplicate claim: content hash matches existing claim "+x)
				return
			}
			y := f.mustSubmitClaim(t, v11Addr("v12-p1-resubmit"), text, dedupDomain, nil)
			require.NotEqual(t, x, y)
			require.Equal(t, y, f.entry(dedupContentKey(text)), "the resubmission's first save names it")
			require.Equal(t, types.ClaimStatus_CLAIM_STATUS_IN_VERIFICATION, f.claim(t, y).Status)
			f.requireSubmitRefused(t, v11Addr("v12-p1-third"), text, nil,
				"duplicate claim: content hash matches existing claim "+y+" (status CLAIM_STATUS_IN_VERIFICATION)")

			require.NoError(t, f.k.SetClaim(f.ctx, f.claim(t, x)))
			require.Equal(t, y, f.entry(dedupContentKey(text)), "re-saving the old terminal claim does not move the entry")
			f.starve(t, y)
			require.Equal(t, y, f.entry(dedupContentKey(text)), "the starved round's re-save does not move it either")
			require.Equal(t, types.ClaimStatus_CLAIM_STATUS_INSUFFICIENT, f.claim(t, x).Status, "the old record is kept")
			f.requireNoLiveRounds(t)
		})
	}
}

// With the marker on, a re-save leaves whatever entry it finds; with it off,
// every save rewrites both entries to the saved claim (knowledge 11).
func TestClaimIntegrity_ResaveDoesNotMoveIndex(t *testing.T) {
	for _, sel := range dedupSelections {
		t.Run(sel.name, func(t *testing.T) {
			f := sel.setup(t)
			x := f.mustSubmitClaim(t, v11Addr("v12-dd-resave"), "Synthetic v12 probe P1: a structured claim whose entries are re-pointed.", dedupDomain,
				dedupStructured("resave probe sample R"))
			claim := f.claim(t, x)
			contentKey, canonicalKey := types.ContentHashKey(claim.ContentHash), types.CanonicalHashKey(claim.CanonicalHash)
			require.NotEmpty(t, claim.CanonicalHash)
			require.Equal(t, x, f.entry(contentKey))
			require.Equal(t, x, f.entry(canonicalKey))

			kv := f.ctx.KVStore(f.key)
			kv.Set(contentKey, []byte(dedupNowhere))
			kv.Set(canonicalKey, []byte(dedupNowhere))
			require.NoError(t, f.k.SetClaim(f.ctx, claim))
			want := dedupNowhere
			if !sel.integrity {
				want = x
			}
			require.Equal(t, want, f.entry(contentKey))
			require.Equal(t, want, f.entry(canonicalKey))
			f.starve(t, x)
			require.Equal(t, want, f.entry(contentKey))
			require.Equal(t, want, f.entry(canonicalKey))
			f.requireNoLiveRounds(t)
		})
	}
}

// A live claim blocks, an accepted claim blocks while its fact stands, and a
// fact refuted by a direct challenge releases both entries for re-assertion
// as a new record. The old fact keeps its status and history.
func TestClaimIntegrity_StandingFactBlocksAndDisprovenReleases(t *testing.T) {
	for _, sel := range dedupSelections {
		t.Run(sel.name, func(t *testing.T) {
			f := sel.setup(t)
			reviewers := v11Reviewers()
			const (
				text  = "Synthetic v12 probe P1: structured statement S, accepted, refuted, then re-asserted."
				other = "Synthetic v12 probe P1: a second lab words structured statement S differently."
			)
			structured := dedupStructured("standing probe sample S")
			x := f.mustSubmitClaim(t, v11Addr("v12-dd-author"), text, dedupDomain, structured)
			claimX := f.claim(t, x)
			contentKey, canonicalKey := types.ContentHashKey(claimX.ContentHash), types.CanonicalHashKey(claimX.CanonicalHash)

			status := func(s string) string {
				if !sel.integrity {
					return ""
				}
				return " (status " + s + ")"
			}
			f.requireSubmitRefused(t, v11Addr("v12-dd-copy"), text, nil,
				"duplicate claim: content hash matches existing claim "+x+status("CLAIM_STATUS_IN_VERIFICATION"))
			f.requireSubmitRefused(t, v11Addr("v12-dd-lab-b"), other, structured,
				"canonical duplicate: matches existing claim "+x+status("CLAIM_STATUS_IN_VERIFICATION"))

			factF := f.accept(t, x, reviewers...)
			canonicalHolder := factF
			if !sel.integrity {
				canonicalHolder = x // the acceptance re-save points 0x34 back at the claim
			}
			require.Equal(t, x, f.entry(contentKey))
			require.Equal(t, canonicalHolder, f.entry(canonicalKey))
			f.requireSubmitRefused(t, v11Addr("v12-dd-copy"), text, nil,
				"duplicate claim: content hash matches existing claim "+x+status("CLAIM_STATUS_ACCEPTED; fact "+factF+" FACT_STATUS_VERIFIED"))
			f.requireSubmitRefused(t, v11Addr("v12-dd-lab-b"), other, structured,
				"canonical duplicate: matches existing claim "+canonicalHolder+status("FACT_STATUS_VERIFIED"))

			f.accept(t, f.challenge(t, v11Addr("v12-dd-challenger"), factF), reviewers...)
			require.Equal(t, types.FactStatus_FACT_STATUS_DISPROVEN, f.fact(t, factF).Status)
			if !sel.integrity {
				f.requireSubmitRefused(t, v11Addr("v12-dd-reassert"), text, structured,
					"canonical duplicate: matches existing claim "+x)
				f.requireSubmitRefused(t, v11Addr("v12-dd-reassert"), text, nil,
					"duplicate claim: content hash matches existing claim "+x)
				f.requireNoLiveRounds(t)
				return
			}
			history := f.transitions(factF)
			z := f.mustSubmitClaim(t, v11Addr("v12-dd-reassert"), text, dedupDomain, structured)
			require.NotEqual(t, x, z)
			require.Equal(t, claimX.CanonicalHash, f.claim(t, z).CanonicalHash)
			require.Equal(t, z, f.entry(contentKey), "the re-assertion's first save names it")
			require.Equal(t, z, f.entry(canonicalKey))
			require.Equal(t, types.FactStatus_FACT_STATUS_DISPROVEN, f.fact(t, factF).Status)
			require.Equal(t, history, f.transitions(factF), "the old fact keeps its history")
			require.Equal(t, types.ClaimStatus_CLAIM_STATUS_ACCEPTED, f.claim(t, x).Status)
			f.starve(t, z)
			f.requireNoLiveRounds(t)
		})
	}
}

// After acceptance with the marker on, 0x34 holds the fact ID and
// FactByCanonical resolves it through the index. With the marker off the
// acceptance re-save points 0x34 back at the claim ID (the inferred defect);
// FactByCanonical still finds the fact, but only through its scan of every
// fact. An accepted claim's fact never takes an entry that already names
// another record.
func TestClaimIntegrity_AcceptedFactKeepsCanonicalEntry(t *testing.T) {
	for _, sel := range dedupSelections {
		t.Run(sel.name, func(t *testing.T) {
			f := sel.setup(t)
			reviewers := v11Reviewers()
			x := f.mustSubmitClaim(t, v11Addr("v12-dd-canon"), "Synthetic v12 probe P1: a structured claim found again by its canonical hash.", dedupDomain,
				dedupStructured("canonical probe sample C"))
			hash := f.claim(t, x).CanonicalHash
			factF := f.accept(t, x, reviewers...)
			require.Equal(t, hash, f.fact(t, factF).CanonicalHash)
			if sel.integrity {
				require.Equal(t, factF, f.entry(types.CanonicalHashKey(hash)))
			} else {
				require.Equal(t, x, f.entry(types.CanonicalHashKey(hash)))
			}
			response, err := keeper.NewQueryServerImpl(f.k).FactByCanonical(f.ctx, &types.QueryFactByCanonicalRequest{CanonicalHash: hash})
			require.NoError(t, err)
			require.Equal(t, factF, response.Fact.Id)

			// Another record already holds W's entry when W is accepted.
			w := f.mustSubmitClaim(t, v11Addr("v12-dd-newer"), "Synthetic v12 probe P1: a structured claim whose entry a newer record holds.", dedupDomain,
				dedupStructured("canonical probe sample W"))
			wKey := types.CanonicalHashKey(f.claim(t, w).CanonicalHash)
			f.ctx.KVStore(f.key).Set(wKey, []byte(dedupNowhere))
			factW := f.accept(t, w, reviewers...)
			if sel.integrity {
				require.Equal(t, dedupNowhere, f.entry(wKey), "the accepted fact does not take another record's entry")
			} else {
				require.Equal(t, w, f.entry(wKey))
			}
			require.NotEmpty(t, factW)
			f.requireNoLiveRounds(t)
		})
	}
}

// Every claim status, applied to the record the 0x09 entry names. Live and
// unknown states block; negative or no-verdict states release; ACCEPTED
// blocks unless its fact has fallen, and a missing round or fact fails closed.
func TestClaimIntegrity_ClaimStatusDecidesWhetherEntryBlocks(t *testing.T) {
	f := setupClaimIntegrityTest(t)
	const text = "Synthetic v12 probe P1: a claim whose recorded status is varied in a cache."
	x := f.mustSubmitClaim(t, v11Addr("v12-dd-status"), text, dedupDomain, nil)
	round := f.roundOf(t, x)
	require.Zero(t, round.VerdictBlock)
	releases := map[types.ClaimStatus]bool{
		types.ClaimStatus_CLAIM_STATUS_REJECTED:     true,
		types.ClaimStatus_CLAIM_STATUS_MALFORMED:    true,
		types.ClaimStatus_CLAIM_STATUS_INSUFFICIENT: true,
		types.ClaimStatus_CLAIM_STATUS_EXPIRED:      true,
	}
	var statuses []int
	for value := range types.ClaimStatus_name {
		statuses = append(statuses, int(value))
	}
	sort.Ints(statuses)
	require.Len(t, statuses, 13)
	for _, value := range statuses {
		status := types.ClaimStatus(value)
		f.inCache(func() {
			claim := f.claim(t, x)
			claim.Status = status
			require.NoError(t, f.k.SetClaim(f.ctx, claim))
			y, err := f.submitClaim(v11Addr("v12-dd-probe"), text, dedupDomain, nil)
			if releases[status] {
				require.NoError(t, err, status.String())
				require.Equal(t, y, f.entry(dedupContentKey(text)), status.String())
				return
			}
			want := status.String()
			if status == types.ClaimStatus_CLAIM_STATUS_ACCEPTED {
				want += "; fact " + keeper.GenerateFactID(x, 0) + " missing"
			}
			require.EqualError(t, err, "duplicate claim: content hash matches existing claim "+x+" (status "+want+")")
		})
	}

	// An ACCEPTED claim whose verdict round is missing also fails closed.
	f.inCache(func() {
		claim := f.claim(t, x)
		claim.Status = types.ClaimStatus_CLAIM_STATUS_ACCEPTED
		require.NoError(t, f.k.SetClaim(f.ctx, claim))
		f.ctx.KVStore(f.key).Delete(types.RoundKey(round.Id))
		_, err := f.submitClaim(v11Addr("v12-dd-probe"), text, dedupDomain, nil)
		require.EqualError(t, err, "duplicate claim: content hash matches existing claim "+x+
			" (status CLAIM_STATUS_ACCEPTED; round "+round.Id+" missing)")
	})
	// So does an ACCEPTED claim that names no round at all. SetClaim never
	// clears a funded claim's round, so the record is planted directly.
	f.inCache(func() {
		claim := f.claim(t, x)
		claim.Status = types.ClaimStatus_CLAIM_STATUS_ACCEPTED
		claim.VerificationRoundId = ""
		bz, err := proto.Marshal(claim)
		require.NoError(t, err)
		f.ctx.KVStore(f.key).Set(types.ClaimKey(x), bz)
		_, err = f.submitClaim(v11Addr("v12-dd-probe"), text, dedupDomain, nil)
		require.EqualError(t, err, "duplicate claim: content hash matches existing claim "+x+
			" (status CLAIM_STATUS_ACCEPTED; no round)")
	})
	// An unreadable round record is returned to the tx and writes nothing.
	f.inCache(func() {
		claim := f.claim(t, x)
		claim.Status = types.ClaimStatus_CLAIM_STATUS_ACCEPTED
		require.NoError(t, f.k.SetClaim(f.ctx, claim))
		f.ctx.KVStore(f.key).Set(types.RoundKey(round.Id), []byte{0xff, 0xff, 0xff})
		before, _ := f.storeDigest(t)
		_, err := f.submitClaim(v11Addr("v12-dd-probe"), text, dedupDomain, nil)
		require.ErrorContains(t, err, "dedup index entry "+x+": read round "+round.Id+": ")
		after, _ := f.storeDigest(t)
		require.Equal(t, before, after)
	})
	f.starve(t, x)
	f.requireNoLiveRounds(t)
}

// Every fact status, applied to an accepted claim's fact. The 0x09 entry
// names the claim and the 0x34 entry names the fact; both resolve to the same
// fact. Fallen facts release; standing and unknown states block. A deleted
// fact fails closed behind the claim, while the 0x34 entry then names nothing.
func TestClaimIntegrity_FactStatusDecidesWhetherEntryBlocks(t *testing.T) {
	f := setupClaimIntegrityTest(t)
	const (
		text  = "Synthetic v12 probe P1: an accepted claim whose fact status is varied in a cache."
		other = "Synthetic v12 probe P1: other words for the accepted claim whose fact status varies."
	)
	structured := dedupStructured("fact status probe sample F")
	x := f.mustSubmitClaim(t, v11Addr("v12-dd-fact"), text, dedupDomain, structured)
	factF := f.accept(t, x, v11Reviewers()...)
	require.Equal(t, factF, f.entry(types.CanonicalHashKey(f.claim(t, x).CanonicalHash)))
	releases := map[types.FactStatus]bool{
		types.FactStatus_FACT_STATUS_SUPERSEDED: true,
		types.FactStatus_FACT_STATUS_EXPIRED:    true,
		types.FactStatus_FACT_STATUS_DISPROVEN:  true,
		types.FactStatus_FACT_STATUS_REVOKED:    true,
		types.FactStatus_FACT_STATUS_PRUNED:     true,
	}
	var statuses []int
	for value := range types.FactStatus_name {
		statuses = append(statuses, int(value))
	}
	sort.Ints(statuses)
	require.Len(t, statuses, 13)
	for _, value := range statuses {
		status := types.FactStatus(value)
		f.inCache(func() {
			fact := f.fact(t, factF)
			fact.Status = status
			require.NoError(t, f.k.SetFact(f.ctx, fact))
			f.inCache(func() {
				_, err := f.submitClaim(v11Addr("v12-dd-copy"), text, dedupDomain, nil)
				if releases[status] {
					require.NoError(t, err, status.String())
					return
				}
				require.EqualError(t, err, "duplicate claim: content hash matches existing claim "+x+
					" (status CLAIM_STATUS_ACCEPTED; fact "+factF+" "+status.String()+")")
			})
			_, err := f.submitClaim(v11Addr("v12-dd-lab-b"), other, dedupDomain, structured)
			if releases[status] {
				require.NoError(t, err, status.String())
				return
			}
			require.EqualError(t, err, "canonical duplicate: matches existing claim "+factF+" (status "+status.String()+")")
		})
	}

	f.inCache(func() {
		require.NoError(t, f.k.DeleteFact(f.ctx, factF))
		f.inCache(func() {
			_, err := f.submitClaim(v11Addr("v12-dd-copy"), text, dedupDomain, nil)
			require.EqualError(t, err, "duplicate claim: content hash matches existing claim "+x+
				" (status CLAIM_STATUS_ACCEPTED; fact "+factF+" missing)")
		})
		_, err := f.submitClaim(v11Addr("v12-dd-lab-b"), other, dedupDomain, structured)
		require.NoError(t, err, "a 0x34 entry naming a deleted fact names nothing")
	})
	f.requireNoLiveRounds(t)
}

// REJECTED and MALFORMED verdicts, reached through real panels, release the
// text. With the marker off they lock it for good.
func TestClaimIntegrity_NegativeVerdictsReleaseText(t *testing.T) {
	for _, sel := range dedupSelections {
		t.Run(sel.name, func(t *testing.T) {
			f := sel.setup(t)
			for _, verdict := range []struct {
				vote   string
				status types.ClaimStatus
			}{
				{"reject", types.ClaimStatus_CLAIM_STATUS_REJECTED},
				{"malformed", types.ClaimStatus_CLAIM_STATUS_MALFORMED},
			} {
				text := "Synthetic v12 probe P1: a statement the panel decided " + verdict.vote + ", then submitted again."
				x := f.mustSubmitClaim(t, v11Addr("v12-dd-"+verdict.vote), text, dedupDomain, nil)
				f.decide(t, x, verdict.vote, v11Reviewers()...)
				require.Equal(t, verdict.status, f.claim(t, x).Status)
				if !sel.integrity {
					f.requireSubmitRefused(t, v11Addr("v12-dd-again"), text, nil, "duplicate claim: content hash matches existing claim "+x)
					continue
				}
				y := f.mustSubmitClaim(t, v11Addr("v12-dd-again"), text, dedupDomain, nil)
				require.Equal(t, y, f.entry(dedupContentKey(text)))
				f.starve(t, y)
			}
			f.requireNoLiveRounds(t)
		})
	}
}

// An entry that names no record does not block with the marker on.
func TestClaimIntegrity_DanglingEntryAllowsAdmission(t *testing.T) {
	for _, sel := range dedupSelections {
		t.Run(sel.name, func(t *testing.T) {
			f := sel.setup(t)
			const text = "Synthetic v12 probe P1: a statement whose content entry names no record."
			f.ctx.KVStore(f.key).Set(dedupContentKey(text), []byte(dedupNowhere))
			const structuredText = "Synthetic v12 probe P1: a structured statement whose canonical entry names no record."
			structured := dedupStructured("dangling probe sample N")
			canonicalKey := types.CanonicalHashKey(f.canonicalHashOf(t, structuredText, structured))
			f.ctx.KVStore(f.key).Set(canonicalKey, []byte(dedupNowhere))

			if !sel.integrity {
				f.requireSubmitRefused(t, v11Addr("v12-dd-dangling"), text, nil, "duplicate claim: content hash matches existing claim "+dedupNowhere)
				f.requireSubmitRefused(t, v11Addr("v12-dd-dangling"), structuredText, structured, "canonical duplicate: matches existing claim "+dedupNowhere)
				return
			}
			y := f.mustSubmitClaim(t, v11Addr("v12-dd-dangling"), text, dedupDomain, nil)
			require.Equal(t, y, f.entry(dedupContentKey(text)))
			z := f.mustSubmitClaim(t, v11Addr("v12-dd-dangling-2"), structuredText, dedupDomain, structured)
			require.Equal(t, z, f.entry(canonicalKey))
			f.starve(t, y)
			f.starve(t, z)
			f.requireNoLiveRounds(t)
		})
	}
}

// An entry naming an unreadable record, or naming nothing at all, refuses the
// tx and writes nothing: it never admits a possible duplicate.
func TestClaimIntegrity_CorruptEntryFailsClosed(t *testing.T) {
	const text = "Synthetic v12 probe P1: a statement whose content entry is corrupt."
	const corruptID = "fedcba9876543210fedcba9876543210"
	for _, sel := range dedupSelections {
		for _, corrupt := range []struct {
			name  string
			entry string
			plant func(f *v11Fixture)
			err   string
		}{
			{"claim record", corruptID, func(f *v11Fixture) {
				f.ctx.KVStore(f.key).Set(types.ClaimKey(corruptID), []byte{0xff, 0xff, 0xff})
			}, "dedup index entry " + corruptID + ": read claim: "},
			{"fact record", corruptID, func(f *v11Fixture) {
				f.ctx.KVStore(f.key).Set(types.FactKey(corruptID), []byte{0xff, 0xff, 0xff})
			}, "dedup index entry " + corruptID + ": read fact: "},
			{"empty entry", "", func(*v11Fixture) {}, "dedup index entry is empty"},
		} {
			t.Run(sel.name+"/"+corrupt.name, func(t *testing.T) {
				f := sel.setup(t)
				corrupt.plant(f)
				f.ctx.KVStore(f.key).Set(dedupContentKey(text), []byte(corrupt.entry))
				before, _ := f.storeDigest(t)
				_, err := f.submitClaim(v11Addr("v12-dd-corrupt"), text, dedupDomain, nil)
				if sel.integrity {
					require.ErrorContains(t, err, corrupt.err)
				} else {
					require.EqualError(t, err, "duplicate claim: content hash matches existing claim "+corrupt.entry)
				}
				after, _ := f.storeDigest(t)
				require.Equal(t, before, after)
			})
		}
	}
}

// MsgPostConjecture uses the same status-aware check on its statement.
func TestClaimIntegrity_ConjectureDedupParity(t *testing.T) {
	for _, sel := range dedupSelections {
		t.Run(sel.name, func(t *testing.T) {
			f := sel.setup(t)
			const statement = "Synthetic v12 probe P1: conjecture Q2, starved and then posed again verbatim."
			post := func(proposer string) (string, error) {
				response, err := f.server.PostConjecture(f.ctx, &types.MsgPostConjecture{
					Proposer: v11Addr(proposer), Domain: dedupDomain, Category: "empirical", Stake: v11ReviewFee, Statement: statement,
					FalsificationPredicate: "Exhibit one synthetic counterexample to conjecture Q2 in this fixture.",
				})
				if err != nil {
					return "", err
				}
				return response.ClaimId, nil
			}
			x, err := post("v12-dd-asker")
			require.NoError(t, err)
			f.starve(t, x)
			if !sel.integrity {
				_, err = post("v12-dd-asker-2")
				require.EqualError(t, err, "duplicate conjecture: content hash matches existing claim "+x)
				return
			}
			y, err := post("v12-dd-asker-2")
			require.NoError(t, err)
			require.Equal(t, y, f.entry(dedupContentKey(statement)))
			before, _ := f.storeDigest(t)
			_, err = post("v12-dd-asker-3")
			require.EqualError(t, err, "duplicate conjecture: content hash matches existing claim "+y+" (status CLAIM_STATUS_IN_VERIFICATION)")
			after, _ := f.storeDigest(t)
			require.Equal(t, before, after)
			f.starve(t, y)
			f.requireNoLiveRounds(t)
		})
	}
}

// Export then import keeps the §5.4 rule that a dedup entry names the latest
// admitted record. A genesis file lists claims in claim-ID order and
// InitGenesis replays each claim's first save. With the flag on it replays
// them in admission order (SubmittedAtBlock, then ID), so whatever order the
// file lists them in, a starved claim never takes the entries back from its
// resubmission, and a third copy is still refused while the resubmission is
// live or its fact stands. With the flag off the file order still decides
// (knowledge 11) and every entry blocks. After import 0x34 names the accepted
// claim rather than its fact: it decides the same way, and FactByCanonical
// still returns the fact.
func TestClaimIntegrity_ImportReplaysAdmissionOrder(t *testing.T) {
	f := setupClaimIntegrityTest(t)
	const (
		text  = "Synthetic v12 probe P1: statement E, starved, resubmitted, exported and imported."
		other = "Synthetic v12 probe P1: a second lab words statement E differently after import."
	)
	structured := dedupStructured("import probe sample E")
	x := f.mustSubmitClaim(t, v11Addr("v12-ei-author"), text, dedupDomain, structured)
	claimX := f.claim(t, x)
	contentKey, canonicalKey := types.ContentHashKey(claimX.ContentHash), types.CanonicalHashKey(claimX.CanonicalHash)
	f.starve(t, x)
	y := f.mustSubmitClaim(t, v11Addr("v12-ei-resubmit"), text, dedupDomain, structured)
	require.Greater(t, f.claim(t, y).SubmittedAtBlock, claimX.SubmittedAtBlock)
	require.Equal(t, y, f.entry(contentKey))
	require.Equal(t, y, f.entry(canonicalKey))
	live, liveHeight := f.k.ExportGenesis(f.ctx), f.ctx.BlockHeight()
	factF := f.accept(t, y, v11Reviewers()...)
	require.Equal(t, factF, f.entry(canonicalKey))
	accepted, acceptedHeight := f.k.ExportGenesis(f.ctx), f.ctx.BlockHeight()
	f.requireNoLiveRounds(t)

	for _, export := range []struct {
		name   string
		gs     *types.GenesisState
		height int64
		status string
	}{
		{"resubmission live", live, liveHeight, "CLAIM_STATUS_IN_VERIFICATION"},
		{"resubmission accepted", accepted, acceptedHeight, "CLAIM_STATUS_ACCEPTED; fact " + factF + " FACT_STATUS_VERIFIED"},
	} {
		for _, on := range []bool{true, false} {
			for _, reversed := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/flag=%t/reversed=%t", export.name, on, reversed), func(t *testing.T) {
					gs := proto.Clone(export.gs).(*types.GenesisState)
					require.True(t, gs.ClaimIntegrityEnabled)
					gs.ClaimIntegrityEnabled = on
					if reversed {
						slices.Reverse(gs.PendingClaims)
					}
					var listed []string
					for _, claim := range gs.PendingClaims {
						listed = append(listed, claim.Id)
					}
					require.ElementsMatch(t, []string{x, y}, listed)
					g := newClaimIntegrityGenesisFixture(t, "claim-integrity-dedup-import", gs)
					for i, claim := range gs.PendingClaims {
						require.Equal(t, listed[i], claim.Id, "InitGenesis does not reorder the caller's file")
					}
					g.advanceTo(t, uint64(export.height))
					require.Equal(t, on, g.claimIntegrityEnabled(t))

					if !on {
						last := listed[len(listed)-1]
						require.Equal(t, last, g.entry(contentKey), "the file order decides (knowledge 11)")
						require.Equal(t, last, g.entry(canonicalKey))
						g.requireSubmitRefused(t, v11Addr("v12-ei-copy"), text, nil,
							"duplicate claim: content hash matches existing claim "+last)
						// The exported claims carry v2 canonical hashes (Task 9),
						// and a marker-off admission derives the v1 form, so no
						// new submission can meet the 0x34 entry checked above.
						v1 := &types.MsgSubmitClaim{}
						structured(v1)
						require.NotEqual(t, claimX.CanonicalHash, types.HashCanonicalForm(types.BuildCanonicalForm(v1.ClaimType, v1.Structure, dedupDomain)))
						return
					}
					require.Equal(t, y, g.entry(contentKey), "the latest admission takes the entry whatever the file order")
					require.Equal(t, y, g.entry(canonicalKey))
					g.requireSubmitRefused(t, v11Addr("v12-ei-copy"), text, nil,
						"duplicate claim: content hash matches existing claim "+y+" (status "+export.status+")")
					g.requireSubmitRefused(t, v11Addr("v12-ei-lab-b"), other, structured,
						"canonical duplicate: matches existing claim "+y+" (status "+export.status+")")
					if export.gs == accepted {
						response, err := keeper.NewQueryServerImpl(g.k).FactByCanonical(g.ctx, &types.QueryFactByCanonicalRequest{CanonicalHash: claimX.CanonicalHash})
						require.NoError(t, err)
						require.Equal(t, factF, response.Fact.Id)
					}
				})
			}
		}
	}
}
