package keeper_test

import (
	"fmt"
	"strings"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/zerone-chain/zerone/x/knowledge/keeper"
	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// Claim integrity, review fixes G2 (admission and dedup). Every rule here is
// behind the marker; each test also runs, or names, its marker-off twin,
// where knowledge 11 must still reproduce.
//
//   - SEC-2, F5 (decision (a)): a MsgSubmitClaim may not carry CONTRADICTS.
//   - SEC-3 (decision (b)): an EXPIRED fact can be revived, so it blocks dedup.
//   - SEC-4: the attribution template is reserved for the attribution itself.
//   - CI-REVIEW-1: a content hash shared by two pre-activation records is read
//     from the last-saved record on the live chain and from the last-admitted
//     record after an import (a known limit, pinned, not repaired).

const (
	g2AttrSubject   = "doi:10.1000/p6-smith-2019"
	g2AttrPredicate = "states:author"
	g2AttrObject    = "zrn-content:f789edd890d61613208c09ebc74248b09c2f170e40c07390c74c74a7db301026"
	g2AttrScope     = "ver=sha256:0580a051899e6a386f8213633badf896d07f21a370a8ba915f2bf93ba054bce3;loc=p4.para2"
	// g2AttrText is the plan §4.3 / spec §2.3 example fact_content (243 bytes).
	g2AttrText = "Attribution zerone.attribution/v1: " + g2AttrSubject + " " + g2AttrPredicate + " " + g2AttrObject + " at " + g2AttrScope
)

// g2Attribution shapes a MsgSubmitClaim as the §4.2 attribution of A1.
func g2Attribution(m *types.MsgSubmitClaim) {
	m.ClaimType = types.ClaimType_CLAIM_TYPE_RELATION
	m.Structure = &types.ClaimStructure{
		Subject: g2AttrSubject, Predicate: g2AttrPredicate, Object: g2AttrObject, Scope: g2AttrScope,
		TemporalScope: "2019-03-01",
		Tags:          []string{"attr-v1", "stmt:e25fa6b012331b0f8681c04b88d4ee7eacaa089d", "zc:f789edd890d61613208c09ebc74248b09c2f170e"},
	}
}

func g2Contradicts(target string) *types.ClaimRelation {
	return &types.ClaimRelation{Relation: types.RelationType_RELATION_TYPE_CONTRADICTS, TargetFactId: target}
}

func g2ContradictsRefusal(index int, target string) string {
	return fmt.Sprintf("claim relation %d names fact %s as CONTRADICTS; a claim cannot carry a CONTRADICTS relation — dispute the fact with MsgSubmitContradiction (`tx knowledge submit-contradiction`), whose admission checks the target's standing, the challenge minimum stake and the counter-claim text: invalid claim",
		index, target)
}

func g2ReservedRefusal(field string) string {
	return field + ` starts with the reserved attribution template "Attribution zerone.attribution/v1:"; only a CLAIM_TYPE_RELATION MsgSubmitClaim whose structure renders exactly that text ("Attribution zerone.attribution/v1: <subject> <predicate> <object> at <scope>") may carry it: invalid claim`
}

// g2AttrName is the attribution template's prefix without its trailing ':'.
const g2AttrName = "Attribution zerone.attribution/v1"

func g2DomainRefusal(domain string) string {
	return fmt.Sprintf(`domain %q contains "Attribution zerone.attribution/v1", the reserved attribution template's name; a content hash joins domain and text with ':', so a record in this domain could take an attribution's content hash with other text: invalid claim`,
		domain)
}

// g2AttrSplit is a domain and text whose content-hash preimage
// ("ZRN.claim.v1:" + domain + ":" + text) is A1's: the domain swallows A1's
// text up to one of its ':'.
type g2AttrSplit struct{ domain, text string }

// g2AttrSplits lists every such split of A1. Its domain, "physics", holds no
// ':', so every split lies at or after the template name's ':' and every
// split domain contains the name, while no split text opens with the prefix.
func g2AttrSplits(t *testing.T) []g2AttrSplit {
	t.Helper()
	want := keeper.ComputeClaimContentHash(g2AttrText, dedupDomain)
	var splits []g2AttrSplit
	for i := 0; i < len(g2AttrText); i++ {
		if g2AttrText[i] != ':' {
			continue
		}
		s := g2AttrSplit{domain: dedupDomain + ":" + g2AttrText[:i], text: g2AttrText[i+1:]}
		require.Equal(t, want, keeper.ComputeClaimContentHash(s.text, s.domain), "split %q", s.domain)
		require.False(t, strings.HasPrefix(s.text, g2AttrName+":"), "the text alone never shows the template")
		require.Contains(t, s.domain, g2AttrName)
		splits = append(splits, s)
	}
	return splits
}

func (f *v11Fixture) proposeDomain(t *testing.T, name string) {
	t.Helper()
	_, err := f.server.ProposeDomain(f.ctx, &types.MsgProposeDomain{Proposer: v11Addr("v12-g2-domains"), Name: name, Description: "Synthetic v12 review G2 domain."})
	require.NoError(t, err)
}

// requireRefusedQuietly runs submit and requires the exact refusal, no bank
// call and no store write.
func (f *v11Fixture) requireRefusedQuietly(t *testing.T, want string, submit func() error) {
	t.Helper()
	before, _ := f.storeDigest(t)
	sends := len(f.bank.sendCalls)
	require.EqualError(t, submit(), want)
	require.Len(t, f.bank.sendCalls, sends, "a refusal moves no coins")
	after, _ := f.storeDigest(t)
	require.Equal(t, before, after, "a refusal writes nothing")
}

func (f *v11Fixture) postConjecture(proposer, statement string) (string, error) {
	return f.postConjectureIn(proposer, dedupDomain, statement)
}

func (f *v11Fixture) postConjectureIn(proposer, domain, statement string) (string, error) {
	response, err := f.server.PostConjecture(f.ctx, &types.MsgPostConjecture{
		Proposer: proposer, Domain: domain, Category: "empirical", Stake: v11ReviewFee, Statement: statement,
		FalsificationPredicate: "Falsified if the pinned source at that locator does not carry the statement.",
	})
	if err != nil {
		return "", err
	}
	return response.ClaimId, nil
}

// SEC-2, F5. With the marker on, a MsgSubmitClaim carrying a CONTRADICTS
// relation is refused before any read of its targets, with a refusal that
// points at MsgSubmitContradiction. Without this, one REVIEW_FEE claim whose
// true but unrelated text a panel accepts contested up to 16 facts at once
// (SEC-2), skipping every contradiction admission check (F5). Other relation
// types are admitted as before and change no fact. Marker off: knowledge 11
// admits the claim and contests its target at admission.
func TestClaimIntegrity_SubmitClaimRefusesContradictsRelation(t *testing.T) {
	for _, sel := range dedupSelections {
		t.Run(sel.name, func(t *testing.T) {
			f := sel.setup(t)
			claimF, factF, _ := f.contraTarget(t, "v12-g2-target-f", "Synthetic v12 review G2: fact F, which a claim relation names as contradicted.")
			_, factG, _ := f.contraTarget(t, "v12-g2-target-g", "Synthetic v12 review G2: fact G, which the same claim also cites.")
			const text = "Synthetic v12 review G2: a finding that names fact F as contradicted through a relation."
			created := []v11Transition{{csUnspecified, csVerified, "verification", claimF}}
			if !sel.integrity {
				c := f.mustSubmitClaim(t, v11Addr("v12-g2-relation"), text, dedupDomain, csContradicts(factF))
				require.Equal(t, csContested, f.fact(t, factF).Status, "knowledge 11 contests F at admission")
				f.starve(t, c)
				f.requireNoLiveRounds(t)
				return
			}
			for _, tc := range []struct {
				name      string
				relations []*types.ClaimRelation
				want      string
			}{
				{"one CONTRADICTS", []*types.ClaimRelation{g2Contradicts(factF)}, g2ContradictsRefusal(0, factF)},
				{"CITES then CONTRADICTS", []*types.ClaimRelation{
					{Relation: types.RelationType_RELATION_TYPE_CITES, TargetFactId: factG}, g2Contradicts(factF),
				}, g2ContradictsRefusal(1, factF)},
				{"CONTRADICTS a fact that does not exist", []*types.ClaimRelation{g2Contradicts(dedupNowhere)}, g2ContradictsRefusal(0, dedupNowhere)},
			} {
				t.Run(tc.name, func(t *testing.T) {
					f.requireRefusedQuietly(t, tc.want, func() error {
						_, err := f.submitClaim(v11Addr("v12-g2-relation"), text, dedupDomain, func(m *types.MsgSubmitClaim) { m.Relations = tc.relations })
						return err
					})
				})
			}
			require.Equal(t, created, f.transitions(factF))

			c := f.mustSubmitClaim(t, v11Addr("v12-g2-relation"), text, dedupDomain, func(m *types.MsgSubmitClaim) {
				m.Relations = []*types.ClaimRelation{
					{Relation: types.RelationType_RELATION_TYPE_CITES, TargetFactId: factG},
					{Relation: types.RelationType_RELATION_TYPE_SUPPORTS, TargetFactId: factF},
				}
			})
			require.Len(t, f.claim(t, c).Relations, 2, "other relation types are admitted")
			require.Equal(t, csVerified, f.fact(t, factF).Status)
			require.Equal(t, created, f.transitions(factF))
			f.starve(t, c)
			f.requireNoLiveRounds(t)
		})
	}
}

// SEC-4. The attribution fact_content is a fixed template of public fields,
// so its content hash is the attribution tuple. With the marker on, any
// record text that opens with the template is refused unless it is a
// CLAIM_TYPE_RELATION MsgSubmitClaim whose structure renders exactly that
// text: on MsgSubmitClaim, MsgPostConjecture and MsgSubmitContradiction. A
// refusal moves no coins and writes nothing. The real attribution is then
// admitted. Marker off: knowledge 11 admits a conjecture carrying the text,
// whose 0x09 entry then blocks the real attribution (the squat).
func TestClaimIntegrity_AttributionTemplateIsReserved(t *testing.T) {
	require.Len(t, g2AttrText, 243, "the plan's example fact_content")
	for _, sel := range dedupSelections {
		t.Run(sel.name, func(t *testing.T) {
			f := sel.setup(t)
			_, factF, minimum := f.contraTarget(t, "v12-g2-attr-target", "Synthetic v12 review G2: a standing fact that a contradiction could name.")
			if !sel.integrity {
				f.inCache(func() {
					_, err := f.submitClaim(v11Addr("v12-g2-squatter"), g2AttrText, dedupDomain, nil)
					require.NoError(t, err, "knowledge 11 admits the template as an assertion")
				})
				squat, err := f.postConjecture(v11Addr("v12-g2-squatter"), g2AttrText)
				require.NoError(t, err, "knowledge 11 admits the template as a conjecture")
				f.requireSubmitRefused(t, v11Addr("v12-g2-extractor"), g2AttrText, g2Attribution,
					"duplicate claim: content hash matches existing claim "+squat)
				c := f.mustContradict(t, v11Addr("v12-g2-squatter-2"), factF, g2AttrText, minimum)
				require.Equal(t, c, f.entry(dedupContentKey(g2AttrText)), "knowledge 11 admits the template as a counter-claim")
				f.starve(t, squat)
				f.starve(t, c)
				f.requireNoLiveRounds(t)
				return
			}

			claimRefusal := g2ReservedRefusal("claim text")
			for _, tc := range []struct {
				name   string
				text   string
				mutate func(*types.MsgSubmitClaim)
			}{
				{"assertion without structure", g2AttrText, nil},
				{"assertion with the attribution structure", g2AttrText, func(m *types.MsgSubmitClaim) {
					g2Attribution(m)
					m.ClaimType = types.ClaimType_CLAIM_TYPE_ASSERTION
				}},
				{"unspecified type with the attribution structure", g2AttrText, func(m *types.MsgSubmitClaim) {
					g2Attribution(m)
					m.ClaimType = types.ClaimType_CLAIM_TYPE_UNSPECIFIED
				}},
				{"relation without structure", g2AttrText, func(m *types.MsgSubmitClaim) {
					m.ClaimType = types.ClaimType_CLAIM_TYPE_RELATION
				}},
				{"relation whose structure renders other text", g2AttrText, func(m *types.MsgSubmitClaim) {
					g2Attribution(m)
					m.Structure.Predicate = "reports:author"
				}},
				{"relation text extending the rendered text", g2AttrText + " (p. 4)", g2Attribution},
				{"relation text opening with the template alone", "Attribution zerone.attribution/v1: anything else at all", g2Attribution},
			} {
				t.Run("MsgSubmitClaim/"+tc.name, func(t *testing.T) {
					f.requireRefusedQuietly(t, claimRefusal, func() error {
						_, err := f.submitClaim(v11Addr("v12-g2-squatter"), tc.text, dedupDomain, tc.mutate)
						return err
					})
				})
			}
			t.Run("MsgPostConjecture", func(t *testing.T) {
				f.requireRefusedQuietly(t, g2ReservedRefusal("conjecture statement"), func() error {
					_, err := f.postConjecture(v11Addr("v12-g2-squatter"), g2AttrText)
					return err
				})
			})
			t.Run("MsgSubmitContradiction", func(t *testing.T) {
				f.requireRefusedQuietly(t, g2ReservedRefusal("counter-claim text"), func() error {
					_, err := f.contradict(v11Addr("v12-g2-squatter"), factF, g2AttrText, dedupDomain, minimum)
					return err
				})
			})

			// Text that only mentions the template, not at its start, is ordinary text.
			mention := "Synthetic v12 review G2: this text quotes Attribution zerone.attribution/v1: in passing."
			m := f.mustSubmitClaim(t, v11Addr("v12-g2-mention"), mention, dedupDomain, nil)

			a := f.mustSubmitClaim(t, v11Addr("v12-g2-extractor"), g2AttrText, dedupDomain, g2Attribution)
			require.Equal(t, a, f.entry(dedupContentKey(g2AttrText)), "the real attribution takes its own 0x09 entry")
			require.Equal(t, types.ClaimType_CLAIM_TYPE_RELATION, f.claim(t, a).ClaimType)
			f.requireSubmitRefused(t, v11Addr("v12-g2-copy"), g2AttrText, g2Attribution,
				"canonical duplicate: matches existing claim "+a+" (status CLAIM_STATUS_IN_VERIFICATION)")
			f.requireSubmitRefused(t, v11Addr("v12-g2-copy"), g2AttrText, func(m *types.MsgSubmitClaim) {
				g2Attribution(m)
				m.Structure.TemporalScope = "2019-03-02"
			}, "duplicate claim: content hash matches existing claim "+a+" (status CLAIM_STATUS_IN_VERIFICATION)")
			f.starve(t, m)
			f.starve(t, a)
			f.requireNoLiveRounds(t)
		})
	}
}

// SEC-4, review G2 re-review. The content-hash preimage
// "ZRN.claim.v1:" + domain + ":" + text is ambiguous, so a record whose domain
// swallows the head of A1's text, up to any ':' in it, takes A1's hash with
// text that never opens with the template. With the marker on, every path
// also refuses a domain that contains the template's name: each split, on
// MsgSubmitClaim and MsgPostConjecture in a proposed domain, and on
// MsgSubmitContradiction in a domain nobody proposed (it never checks that
// its domain exists). A domain that merely contains the name is refused too.
// Each refusal moves no coins and writes nothing, and the real attribution is
// then admitted. Marker off: knowledge 11 admits every split on every path,
// and each blocks the real attribution.
func TestClaimIntegrity_AttributionHashCannotBeSplitIntoDomain(t *testing.T) {
	splits := g2AttrSplits(t)
	require.Len(t, splits, 5, "one split at each ':' of A1's text: after the name, doi:, states:, zrn-content:, ver=sha256:")
	for _, sel := range dedupSelections {
		t.Run(sel.name, func(t *testing.T) {
			f := sel.setup(t)
			_, factF, minimum := f.contraTarget(t, "v12-g2-split-target", "Synthetic v12 review G2: a standing fact that a split contradiction names.")
			if !sel.integrity {
				for _, s := range splits {
					_, found := f.k.GetDomain(f.ctx, s.domain)
					require.False(t, found)
					f.inCache(func() {
						c, err := f.contradict(v11Addr("v12-g2-split-contra"), factF, s.text, s.domain, minimum)
						require.NoError(t, err, "knowledge 11 admits a contradiction in a domain nobody proposed")
						f.requireSubmitRefused(t, v11Addr("v12-g2-extractor"), g2AttrText, g2Attribution,
							"duplicate claim: content hash matches existing claim "+c)
					})
				}
				for _, s := range splits {
					f.proposeDomain(t, s.domain)
					f.inCache(func() {
						sq := f.mustSubmitClaim(t, v11Addr("v12-g2-squatter"), s.text, s.domain, nil)
						f.requireSubmitRefused(t, v11Addr("v12-g2-extractor"), g2AttrText, g2Attribution,
							"duplicate claim: content hash matches existing claim "+sq)
					})
					f.inCache(func() {
						sq, err := f.postConjectureIn(v11Addr("v12-g2-squatter"), s.domain, s.text)
						require.NoError(t, err)
						f.requireSubmitRefused(t, v11Addr("v12-g2-extractor"), g2AttrText, g2Attribution,
							"duplicate claim: content hash matches existing claim "+sq)
					})
				}
				f.requireNoLiveRounds(t)
				return
			}

			for i, s := range splits {
				t.Run(fmt.Sprintf("split %d/MsgSubmitContradiction in a domain nobody proposed", i), func(t *testing.T) {
					_, found := f.k.GetDomain(f.ctx, s.domain)
					require.False(t, found)
					f.requireRefusedQuietly(t, g2DomainRefusal(s.domain), func() error {
						_, err := f.contradict(v11Addr("v12-g2-split-contra"), factF, s.text, s.domain, minimum)
						return err
					})
				})
			}
			for _, s := range splits {
				f.proposeDomain(t, s.domain)
			}
			f.proposeDomain(t, g2AttrName)
			for i, s := range splits {
				t.Run(fmt.Sprintf("split %d/MsgSubmitClaim", i), func(t *testing.T) {
					f.requireRefusedQuietly(t, g2DomainRefusal(s.domain), func() error {
						_, err := f.submitClaim(v11Addr("v12-g2-squatter"), s.text, s.domain, nil)
						return err
					})
				})
				t.Run(fmt.Sprintf("split %d/MsgPostConjecture", i), func(t *testing.T) {
					f.requireRefusedQuietly(t, g2DomainRefusal(s.domain), func() error {
						_, err := f.postConjectureIn(v11Addr("v12-g2-squatter"), s.domain, s.text)
						return err
					})
				})
			}
			const plain = "Synthetic v12 review G2: ordinary text in a domain named after the template."
			t.Run("a domain that only contains the name", func(t *testing.T) {
				f.requireRefusedQuietly(t, g2DomainRefusal(g2AttrName), func() error {
					_, err := f.submitClaim(v11Addr("v12-g2-squatter"), plain, g2AttrName, nil)
					return err
				})
				f.requireRefusedQuietly(t, g2DomainRefusal(g2AttrName), func() error {
					_, err := f.postConjectureIn(v11Addr("v12-g2-squatter"), g2AttrName, plain)
					return err
				})
				f.requireRefusedQuietly(t, g2DomainRefusal(g2AttrName), func() error {
					_, err := f.contradict(v11Addr("v12-g2-split-contra"), factF, plain, g2AttrName, minimum)
					return err
				})
			})

			a := f.mustSubmitClaim(t, v11Addr("v12-g2-extractor"), g2AttrText, dedupDomain, g2Attribution)
			require.Equal(t, a, f.entry(dedupContentKey(g2AttrText)), "the real attribution takes its own 0x09 entry")
			f.starve(t, a)
			f.requireNoLiveRounds(t)
		})
	}
}

// SEC-4, review G2 re-review. A contradiction that names no domain takes its
// target's, and its content hash uses that effective domain, so the reserved
// check reads it too. A fact admitted before activation in a domain named
// like A1's first split cannot lend that domain to a contradiction carrying
// the rest of A1's text. Marker off: knowledge 11 admits it, and its 0x09
// entry blocks the real attribution.
func TestClaimIntegrity_ContradictionDefaultDomainIsChecked(t *testing.T) {
	s := g2AttrSplits(t)[0]
	for _, integrity := range []bool{true, false} {
		name := "marker off"
		if integrity {
			name = "marker on (upgraded in place)"
		}
		t.Run(name, func(t *testing.T) {
			f := setupV11SemanticsTest(t)
			f.proposeDomain(t, s.domain)
			old := f.mustSubmitClaim(t, v11Addr("v12-g2-split-old"), "Synthetic v12 review G2: a pre-activation fact in a domain named like the template.", s.domain, nil)
			factT := f.accept(t, old, v11Reviewers()...)
			require.Equal(t, s.domain, f.fact(t, factT).Domain)
			params, err := f.k.GetParams(f.ctx)
			require.NoError(t, err)
			minimum := keeper.EffectiveMinChallengeStake(params, f.fact(t, factT).Confidence).String()
			f.requireNoLiveRounds(t)
			if !integrity {
				c, err := f.contradict(v11Addr("v12-g2-split-contra"), factT, s.text, "", minimum)
				require.NoError(t, err)
				require.Equal(t, keeper.ComputeClaimContentHash(g2AttrText, dedupDomain), f.claim(t, c).ContentHash)
				f.requireSubmitRefused(t, v11Addr("v12-g2-extractor"), g2AttrText, g2Attribution,
					"duplicate claim: content hash matches existing claim "+c)
				f.starve(t, c)
				f.requireNoLiveRounds(t)
				return
			}
			require.NoError(t, f.k.ValidateClaimIntegrityActivation(f.ctx))
			require.NoError(t, f.k.EnableClaimIntegrity(f.ctx))
			f.requireRefusedQuietly(t, g2DomainRefusal(s.domain), func() error {
				_, err := f.contradict(v11Addr("v12-g2-split-contra"), factT, s.text, "", minimum)
				return err
			})
			a := f.mustSubmitClaim(t, v11Addr("v12-g2-extractor"), g2AttrText, dedupDomain, g2Attribution)
			require.Equal(t, a, f.entry(dedupContentKey(g2AttrText)), "the real attribution takes its own 0x09 entry")
			f.starve(t, a)
			f.requireNoLiveRounds(t)
		})
	}
}

// Review G2 re-review (spec §2.3, plan §4.3). Decision (a) leaves P2 two ways
// onto the chain, and they exclude each other: while a structured P2 claim is
// live or stands, a contradiction carrying P2's text is refused, as an
// identity clash from the same account in the same block and as a duplicate of
// P2 otherwise. Pinned, not red-first: P1ext's dedup already did this. Marker
// off: the same-account clash is the same, and another account's
// contradiction is admitted and takes P2's entry (knowledge 11).
func TestClaimIntegrity_StructuredPropositionCannotAlsoContest(t *testing.T) {
	for _, sel := range dedupSelections {
		t.Run(sel.name, func(t *testing.T) {
			f := sel.setup(t)
			_, factP1, minimum := f.contraTarget(t, "v12-g2-p1", "Synthetic v12 review G2: compound Z boils at 78.4 C at 1 atm for a pure sample.")
			const p2 = "Synthetic v12 review G2: compound Z boils at 81.0 C at 1 atm for a pure sample."
			author := v11Addr("v12-g2-p2")
			c2 := f.mustSubmitClaim(t, author, p2, dedupDomain, dedupStructured("boiling point of compound Z"))
			f.requireRefusedQuietly(t, "contradiction claim identity already exists", func() error {
				_, err := f.contradict(author, factP1, p2, dedupDomain, minimum)
				return err
			})
			if !sel.integrity {
				f.inCache(func() {
					c := f.mustContradict(t, v11Addr("v12-g2-p2-contra"), factP1, p2, minimum)
					require.Equal(t, c, f.entry(dedupContentKey(p2)), "knowledge 11: the contradiction takes P2's entry")
				})
				f.starve(t, c2)
				f.requireNoLiveRounds(t)
				return
			}
			f.requireContradictionRefused(t, v11Addr("v12-g2-p2-contra"), factP1, p2, minimum,
				"duplicate claim: content hash matches existing claim "+c2+" (status CLAIM_STATUS_IN_VERIFICATION)")
			f2 := f.accept(t, c2, v11Reviewers()...)
			f.requireContradictionRefused(t, v11Addr("v12-g2-p2-contra"), factP1, p2, minimum,
				"duplicate claim: content hash matches existing claim "+c2+" (status CLAIM_STATUS_ACCEPTED; fact "+f2+" FACT_STATUS_VERIFIED)")
			f.requireNoLiveRounds(t)
		})
	}
}

func g2Salt(claimID string, i int) []byte {
	return []byte(fmt.Sprintf("v12-g2-salt-%s-%02d", claimID[:8], i))
}

// g2Commit seats the three fixture reviewers on claimID's round with vote.
func (f *v11Fixture) g2Commit(t *testing.T, claimID, vote string) {
	t.Helper()
	round := f.roundOf(t, claimID)
	require.Equal(t, types.VerificationPhase_VERIFICATION_PHASE_COMMIT, round.Phase)
	attestation := &types.ReviewAttestation{Reason: "Synthetic v12 review G2 vote on the stated record only."}
	for i, reviewer := range v11Reviewers() {
		f.bank.balances[reviewer] = sdk.NewCoins(sdk.NewInt64Coin("uzrn", 100_000_000))
		hash, err := types.ComputeReviewCommitmentV2(round.CommitmentChainId, round.Id, reviewer, vote, v11Confidence, g2Salt(claimID, i), attestation)
		require.NoError(t, err)
		_, err = f.server.SubmitCommitment(f.ctx, &types.MsgSubmitCommitment{Verifier: reviewer, RoundId: round.Id, CommitHash: hash})
		require.NoError(t, err)
	}
}

// g2Reveal reveals what g2Commit committed.
func (f *v11Fixture) g2Reveal(t *testing.T, claimID, vote string) {
	t.Helper()
	round := f.roundOf(t, claimID)
	require.Equal(t, types.VerificationPhase_VERIFICATION_PHASE_REVEAL, round.Phase)
	attestation := &types.ReviewAttestation{Reason: "Synthetic v12 review G2 vote on the stated record only."}
	for i, reviewer := range v11Reviewers() {
		_, err := f.server.SubmitReveal(f.ctx, &types.MsgSubmitReveal{
			Verifier: reviewer, RoundId: round.Id, Vote: vote, Confidence: v11Confidence, Salt: g2Salt(claimID, i), Attestation: attestation,
		})
		require.NoError(t, err)
	}
}

// metabolizeUntil runs real metabolism epochs from *epoch until fact reaches
// status, and fails after a bounded number of epochs.
func (f *v11Fixture) metabolizeUntil(t *testing.T, factID string, epoch *uint64, status types.FactStatus) {
	t.Helper()
	for i := 0; i < 1000; i++ {
		if f.fact(t, factID).Status == status {
			return
		}
		*epoch++
		require.NoError(t, f.k.ProcessMetabolism(f.ctx, *epoch))
	}
	require.Equal(t, status, f.fact(t, factID).Status, "metabolism did not reach %s", status)
}

// factsWithContent counts stored facts carrying text in dedupDomain.
func (f *v11Fixture) factsWithContent(text string) int {
	n := 0
	f.k.IterateFacts(f.ctx, func(fact *types.Fact) bool {
		if fact.Content == text && fact.Domain == dedupDomain {
			n++
		}
		return false
	})
	return n
}

// SEC-3, decision (b). An accepted proposition P decays through real
// metabolism to EXPIRED. Patronage or metabolism can revive an EXPIRED fact,
// so with the marker on it still blocks its text and its structure: an exact
// resubmission, a reworded twin with the same structure and a conjecture
// carrying the text are all refused, naming the EXPIRED fact. Patronage then
// revives P, and P is still the only fact with that text, so no twin was
// created. Left alone, P decays to PRUNED, which is final, and the text is
// released. Marker off: every entry blocks, as in knowledge 11.
func TestClaimIntegrity_ExpiredFactBlocksDedup(t *testing.T) {
	for _, sel := range dedupSelections {
		t.Run(sel.name, func(t *testing.T) {
			f := sel.setup(t)
			const (
				text  = "Synthetic v12 review G2: proposition P, accepted, then left without energy until it expires."
				other = "Synthetic v12 review G2: other words for proposition P, offered while P is expired."
			)
			structured := dedupStructured("expiring proposition sample P")
			p := f.mustSubmitClaim(t, v11Addr("v12-g2-exp-author"), text, dedupDomain, structured)
			factP := f.accept(t, p, v11Reviewers()...)
			var epoch uint64
			f.metabolizeUntil(t, factP, &epoch, types.FactStatus_FACT_STATUS_AT_RISK)
			f.metabolizeUntil(t, factP, &epoch, types.FactStatus_FACT_STATUS_EXPIRED)

			if !sel.integrity {
				f.requireSubmitRefused(t, v11Addr("v12-g2-exp-copy"), text, nil, "duplicate claim: content hash matches existing claim "+p)
				f.requireNoLiveRounds(t)
				return
			}
			expired := " (status CLAIM_STATUS_ACCEPTED; fact " + factP + " FACT_STATUS_EXPIRED)"
			f.requireSubmitRefused(t, v11Addr("v12-g2-exp-copy"), text, nil, "duplicate claim: content hash matches existing claim "+p+expired)
			f.requireSubmitRefused(t, v11Addr("v12-g2-exp-twin"), other, structured,
				"canonical duplicate: matches existing claim "+factP+" (status FACT_STATUS_EXPIRED)")
			f.requireRefusedQuietly(t, "duplicate conjecture: content hash matches existing claim "+p+expired, func() error {
				_, err := f.postConjecture(v11Addr("v12-g2-exp-asker"), text)
				return err
			})

			f.inCache(func() {
				patron := v11Addr("v12-g2-exp-patron")
				f.bank.balances[patron] = sdk.NewCoins(sdk.NewInt64Coin("uzrn", 10))
				_, err := f.server.PatronizeFact(f.ctx, &types.MsgPatronizeFact{Patron: patron, FactId: factP, Amount: "1", DurationBlocks: 10_000_000})
				require.NoError(t, err)
				require.Equal(t, types.FactStatus_FACT_STATUS_ACTIVE, f.fact(t, factP).Status, "patronage revives the EXPIRED fact")
				require.Equal(t, 1, f.factsWithContent(text), "no twin of P exists to stand beside it")
				f.requireSubmitRefused(t, v11Addr("v12-g2-exp-copy"), text, nil,
					"duplicate claim: content hash matches existing claim "+p+" (status CLAIM_STATUS_ACCEPTED; fact "+factP+" FACT_STATUS_ACTIVE)")
			})

			f.metabolizeUntil(t, factP, &epoch, types.FactStatus_FACT_STATUS_PRUNED)
			y := f.mustSubmitClaim(t, v11Addr("v12-g2-exp-again"), text, dedupDomain, structured)
			require.Equal(t, y, f.entry(dedupContentKey(text)), "a pruned fact releases its text")
			f.starve(t, y)
			f.requireNoLiveRounds(t)
		})
	}
}

// CI-REVIEW-1, a known limit pinned here and not repaired (plan §3.6, spec
// §1.5). Before activation, knowledge 11 let a contradiction carry another
// claim's exact text (P1ext), and every save rewrote the 0x09 entry, so the
// entry names whichever record was saved last. Activation rewrites nothing:
// the live chain decides dedup from that last-saved record. Import replays
// first saves in admission order, so the entry names the last-admitted
// record instead. When the two differ in standing, live and imported chains
// disagree, in either direction:
//   - "accepted counter-claim, starved original": the live entry names the
//     starved original, so the live chain admits a third copy although the
//     accepted contradiction's fact stands; the import refuses it.
//   - "rejected counter-claim, accepted original": the live entry names the
//     accepted original, whose fact stands, so the live chain refuses; the
//     import names the rejected contradiction and admits a third copy.
func TestClaimIntegrity_LegacyContentCollisionLiveVsImport(t *testing.T) {
	for _, scenario := range []string{"accepted counter-claim, starved original", "rejected counter-claim, accepted original"} {
		t.Run(scenario, func(t *testing.T) {
			f := setupV11SemanticsTest(t)
			const text = "Synthetic v12 review G2: statement T, which a contradiction later copies verbatim."
			_, target, _ := f.contraTarget(t, "v12-g2-ci-target", "Synthetic v12 review G2: an accepted target for the copying contradiction.")
			x := f.mustSubmitClaim(t, v11Addr("v12-g2-ci-author"), text, dedupDomain, nil)
			f.nextBlock(t)
			c, err := f.contradict(v11Addr("v12-g2-ci-contra"), target, text, dedupDomain, v11ReviewFee)
			require.NoError(t, err)
			require.Equal(t, c, f.entry(dedupContentKey(text)), "knowledge 11: the contradiction takes the entry")

			// x was admitted first and c last; x is saved last in both scenarios.
			live, imported := x, c
			var standing string
			if scenario == "accepted counter-claim, starved original" {
				k := f.accept(t, c, v11Reviewers()...)
				f.starve(t, x)
				standing = " (status CLAIM_STATUS_ACCEPTED; fact " + k + " FACT_STATUS_VERIFIED)"
			} else {
				// Both panels commit; c's reveals close first, so x's verdict
				// is the later save.
				f.g2Commit(t, x, "accept")
				f.g2Commit(t, c, "reject")
				f.advanceTo(t, f.roundOf(t, c).CommitDeadline)
				f.beginBlock(t)
				f.g2Reveal(t, c, "reject")
				f.nextBlock(t)
				f.beginBlock(t)
				require.Equal(t, types.ClaimStatus_CLAIM_STATUS_REJECTED, f.claim(t, c).Status)
				require.Equal(t, c, f.entry(dedupContentKey(text)))
				f.g2Reveal(t, x, "accept")
				f.nextBlock(t)
				f.beginBlock(t)
				round := f.roundOf(t, x)
				require.Equal(t, types.Verdict_VERDICT_ACCEPT, round.Verdict)
				fx := keeper.GenerateFactID(x, round.VerdictBlock)
				require.Equal(t, csVerified, f.fact(t, fx).Status)
				standing = " (status CLAIM_STATUS_ACCEPTED; fact " + fx + " FACT_STATUS_VERIFIED)"
			}
			require.Equal(t, live, f.entry(dedupContentKey(text)), "the last save names the entry")
			f.requireNoLiveRounds(t)

			require.NoError(t, f.k.ValidateClaimIntegrityActivation(f.ctx))
			require.NoError(t, f.k.EnableClaimIntegrity(f.ctx))
			gs := f.k.ExportGenesis(f.ctx)
			require.True(t, gs.ClaimIntegrityEnabled)
			g := newClaimIntegrityGenesisFixture(t, f.ctx.ChainID(), proto.Clone(gs).(*types.GenesisState))
			g.advanceTo(t, uint64(f.ctx.BlockHeight()))
			require.Equal(t, imported, g.entry(dedupContentKey(text)), "import replays admission order: the last admitted names the entry")

			if scenario == "accepted counter-claim, starved original" {
				f.inCache(func() {
					_, err := f.submitClaim(v11Addr("v12-g2-ci-copy"), text, dedupDomain, nil)
					require.NoError(t, err, "live: the entry names the starved original, which no longer blocks")
				})
				g.requireSubmitRefused(t, v11Addr("v12-g2-ci-copy"), text, nil,
					"duplicate claim: content hash matches existing claim "+imported+standing)
				return
			}
			f.requireSubmitRefused(t, v11Addr("v12-g2-ci-copy"), text, nil,
				"duplicate claim: content hash matches existing claim "+live+standing)
			g.inCache(func() {
				_, err := g.submitClaim(v11Addr("v12-g2-ci-copy"), text, dedupDomain, nil)
				require.NoError(t, err, "import: the entry names the rejected contradiction, which no longer blocks")
			})
			require.Equal(t, keeper.ComputeClaimContentHash(text, dedupDomain), f.claim(t, x).ContentHash)
			require.Equal(t, f.claim(t, x).ContentHash, f.claim(t, c).ContentHash, "one content hash, two records")
		})
	}
}
