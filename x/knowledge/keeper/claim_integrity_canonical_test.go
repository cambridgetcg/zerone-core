package keeper_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/zerone-chain/zerone/x/knowledge/keeper"
	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// Claim integrity, Task 9 (P2): canonical form v2. With the marker on, a
// structured claim's canonical form is always derived from its structure by
// types.BuildCanonicalFormV2. A declared form must equal it after
// types.NormalizeCanonicalFormForLookup, and a form without a structure is
// refused, so no claim can reserve another's canonical slot. Object, scope,
// temporal scope, case and claim type all take part in the hash. With the
// marker off, v1 runs unchanged and the P2 collisions still reproduce.
//
// Reviewers and challengers are never parties and no scenario submits a
// contradiction, so later v12 tasks (10-14) do not change what these observe.

const (
	canonAssertion     = types.ClaimType_CLAIM_TYPE_ASSERTION
	canonComputational = types.ClaimType_CLAIM_TYPE_COMPUTATIONAL
)

func canonStructure(object, scope string) *types.ClaimStructure {
	return &types.ClaimStructure{Subject: "boiling point of compound Z", Predicate: "equals degrees C at 1 atm", Object: object, Scope: scope}
}

func canonClaim(claimType types.ClaimType, structure *types.ClaimStructure, declared string) func(*types.MsgSubmitClaim) {
	return func(m *types.MsgSubmitClaim) {
		m.ClaimType = claimType
		m.Structure = structure
		m.CanonicalForm = declared
	}
}

// canonFormFor is the form a selection derives: v2 with the marker on, v1 off.
func canonFormFor(integrity bool, claimType types.ClaimType, structure *types.ClaimStructure) string {
	if integrity {
		return types.BuildCanonicalFormV2(claimType, structure, dedupDomain)
	}
	return types.BuildCanonicalForm(claimType, structure, dedupDomain)
}

// P2 B1 replica, plus each other place where v1 drops or folds a difference:
// the object, the temporal scope, the claim type and letter case. With the
// marker on every finding is admitted under its own hash; with it off each
// collides with the first claim of its pair.
func TestClaimIntegrity_CanonicalV2SeparatesDistinctFindings(t *testing.T) {
	type finding struct {
		author, text string
		claimType    types.ClaimType
		structure    *types.ClaimStructure
	}
	pairs := []struct {
		name          string
		first, second finding
	}{
		{"object 78.4 vs 81.0",
			finding{"v12-cf-lab-a", "Synthetic v12 probe P2: Lab A reports compound Z boils at 78.4 C at 1 atm.", canonAssertion, canonStructure("78.4", "pure sample")},
			finding{"v12-cf-lab-b", "Synthetic v12 probe P2: Lab B reports compound Z boils at 81.0 C at 1 atm.", canonAssertion, canonStructure("81.0", "pure sample")}},
		{"temporal scope",
			finding{"v12-cf-t-first", "Synthetic v12 probe P2: compound Y boiled at 64.7 C when first measured.", canonAssertion,
				&types.ClaimStructure{Subject: "boiling point of compound Y", Predicate: "equals degrees C", Object: "64.7", TemporalScope: "2019-03-01"}},
			finding{"v12-cf-t-second", "Synthetic v12 probe P2: compound Y boiled at 64.7 C when measured again.", canonAssertion,
				&types.ClaimStructure{Subject: "boiling point of compound Y", Predicate: "equals degrees C", Object: "64.7", TemporalScope: "2021-01-01"}}},
		{"claim type",
			finding{"v12-cf-asserted", "Synthetic v12 probe P2: compound X boils at 99.1 C, as measured.", canonAssertion, &types.ClaimStructure{Subject: "boiling point of compound X", Predicate: "equals degrees C", Object: "99.1"}},
			finding{"v12-cf-computed", "Synthetic v12 probe P2: compound X boils at 99.1 C, as computed.", canonComputational, &types.ClaimStructure{Subject: "boiling point of compound X", Predicate: "equals degrees C", Object: "99.1"}}},
		{"5 mW vs 5 MW",
			finding{"v12-cf-milli", "Synthetic v12 probe P2: the laser in setup L emits five milliwatts.", canonAssertion, &types.ClaimStructure{Subject: "laser in setup L", Predicate: "emits", Object: "5 mW"}},
			finding{"v12-cf-mega", "Synthetic v12 probe P2: the laser in setup L emits five megawatts.", canonAssertion, &types.ClaimStructure{Subject: "laser in setup L", Predicate: "emits", Object: "5 MW"}}},
	}
	for _, sel := range dedupSelections {
		t.Run(sel.name, func(t *testing.T) {
			f := sel.setup(t)
			require.Equal(t, sel.integrity, f.claimIntegrityEnabled(t))
			var live []string
			for _, pair := range pairs {
				first := f.mustSubmitClaim(t, v11Addr(pair.first.author), pair.first.text, dedupDomain, canonClaim(pair.first.claimType, pair.first.structure, ""))
				live = append(live, first)
				claim := f.claim(t, first)
				require.Equal(t, canonFormFor(sel.integrity, pair.first.claimType, pair.first.structure), claim.CanonicalForm, pair.name)
				require.Equal(t, types.HashCanonicalForm(claim.CanonicalForm), claim.CanonicalHash)
				require.Equal(t, first, f.entry(types.CanonicalHashKey(claim.CanonicalHash)))
				require.Equal(t, sel.integrity, strings.HasPrefix(claim.CanonicalForm, types.CanonicalFormV2Prefix))

				mutate := canonClaim(pair.second.claimType, pair.second.structure, "")
				if !sel.integrity {
					f.requireSubmitRefused(t, v11Addr(pair.second.author), pair.second.text, mutate,
						"canonical duplicate: matches existing claim "+first)
					continue
				}
				second := f.mustSubmitClaim(t, v11Addr(pair.second.author), pair.second.text, dedupDomain, mutate)
				live = append(live, second)
				require.NotEqual(t, claim.CanonicalHash, f.claim(t, second).CanonicalHash, pair.name)
				require.Equal(t, second, f.entry(types.CanonicalHashKey(f.claim(t, second).CanonicalHash)))
				require.Equal(t, first, f.entry(types.CanonicalHashKey(claim.CanonicalHash)), "the first claim keeps its own entry")
			}
			for _, id := range live {
				f.starve(t, id)
			}
			f.requireNoLiveRounds(t)
		})
	}
}

// An identical structure is still one canonical slot. With the marker on, an
// UNSPECIFIED type, padding, whitespace runs and NFD spell the same v2 form.
func TestClaimIntegrity_CanonicalV2RefusesIdenticalStructure(t *testing.T) {
	structure := &types.ClaimStructure{Subject: "boiling point of compound Z", Predicate: "equals degrees C at 1 atm", Object: "78.4", Scope: "pure sample from Z\u00fcrich"}
	respelled := &types.ClaimStructure{Subject: "  boiling \t point of compound Z\n", Predicate: "equals degrees C at 1\u00a0atm", Object: " 78.4", Scope: "pure sample from Zu\u0308rich "}
	for _, sel := range dedupSelections {
		t.Run(sel.name, func(t *testing.T) {
			f := sel.setup(t)
			a := f.mustSubmitClaim(t, v11Addr("v12-cf-same-a"), "Synthetic v12 probe P2: Lab A reports compound Z from Zurich boils at 78.4 C.", dedupDomain,
				canonClaim(canonAssertion, structure, ""))
			blocker := a
			if sel.integrity {
				blocker += " (status CLAIM_STATUS_IN_VERIFICATION)"
			}
			f.requireSubmitRefused(t, v11Addr("v12-cf-same-b"), "Synthetic v12 probe P2: a reworded report that compound Z from Zurich boils at 78.4 C.",
				canonClaim(canonAssertion, structure, ""), "canonical duplicate: matches existing claim "+blocker)

			respelledText := "Synthetic v12 probe P2: the same structure, typed with odd spacing and decomposed accents."
			if sel.integrity {
				require.Equal(t, types.BuildCanonicalFormV2(canonAssertion, structure, dedupDomain),
					types.BuildCanonicalFormV2(types.ClaimType_CLAIM_TYPE_UNSPECIFIED, respelled, dedupDomain))
				f.requireSubmitRefused(t, v11Addr("v12-cf-same-c"), respelledText,
					canonClaim(types.ClaimType_CLAIM_TYPE_UNSPECIFIED, respelled, ""), "canonical duplicate: matches existing claim "+blocker)
				f.starve(t, a)
			} else {
				// v1 keeps whitespace inside an argument, so this respelling escapes it.
				c := f.mustSubmitClaim(t, v11Addr("v12-cf-same-c"), respelledText, dedupDomain, canonClaim(types.ClaimType_CLAIM_TYPE_UNSPECIFIED, respelled, ""))
				require.NotEqual(t, f.claim(t, a).CanonicalHash, f.claim(t, c).CanonicalHash)
				f.starve(t, a)
				f.starve(t, c)
			}
			f.requireNoLiveRounds(t)
		})
	}
}

// P2 S1/S2 replica. With the marker on, a canonical form without a structure
// is refused at S1, whatever its version, so the structured S2 is admitted.
// With it off, S1 reserves S2's slot.
func TestClaimIntegrity_CanonicalFormRequiresStructure(t *testing.T) {
	s2 := &types.ClaimStructure{Subject: "melting point of compound Z", Predicate: "equals degrees C", Object: "-114", Scope: "pure sample at 1 atm"}
	const (
		squatText = "Synthetic v12 probe P2: an unrelated sentence that tries to reserve a canonical form without any structure."
		s2Text    = "Synthetic v12 probe P2: Lab C reports the melting point of compound Z at 1 atm is minus 114 C."
	)
	for _, sel := range dedupSelections {
		t.Run(sel.name, func(t *testing.T) {
			f := sel.setup(t)
			form := canonFormFor(sel.integrity, canonAssertion, s2)
			declare := func(declared string) func(*types.MsgSubmitClaim) {
				return func(m *types.MsgSubmitClaim) { m.CanonicalForm = declared }
			}
			if !sel.integrity {
				squat := f.mustSubmitClaim(t, v11Addr("v12-cf-squatter"), squatText, dedupDomain, declare(form))
				require.Nil(t, f.claim(t, squat).Structure)
				f.requireSubmitRefused(t, v11Addr("v12-cf-lab-c"), s2Text, canonClaim(canonAssertion, s2, ""),
					"canonical duplicate: matches existing claim "+squat)
				f.starve(t, squat)
				f.requireNoLiveRounds(t)
				return
			}
			for _, declared := range []string{form, types.BuildCanonicalForm(canonAssertion, s2, dedupDomain), "anything at all"} {
				f.requireSubmitRefused(t, v11Addr("v12-cf-squatter"), squatText, declare(declared), "canonical form requires claim structure")
			}
			s2ID := f.mustSubmitClaim(t, v11Addr("v12-cf-lab-c"), s2Text, dedupDomain, canonClaim(canonAssertion, s2, ""))
			require.Equal(t, types.HashCanonicalForm(form), f.claim(t, s2ID).CanonicalHash)
			require.Equal(t, s2ID, f.entry(types.CanonicalHashKey(types.HashCanonicalForm(form))))
			// A structureless claim that declares nothing is still admitted, with no canonical hash.
			plain := f.mustSubmitClaim(t, v11Addr("v12-cf-plain"), squatText, dedupDomain, nil)
			require.Empty(t, f.claim(t, plain).CanonicalForm)
			require.Empty(t, f.claim(t, plain).CanonicalHash)
			f.starve(t, s2ID)
			f.starve(t, plain)
			f.requireNoLiveRounds(t)
		})
	}
}

// A declared canonical form must be the one the structure derives. With the
// marker on that is the v2 form (padding is trimmed); a v1 form, a lowercased
// v2 form or another structure's form is refused. With it off, v1 rules hold.
func TestClaimIntegrity_DeclaredCanonicalFormMustMatchStructure(t *testing.T) {
	structure, other := canonStructure("78.4", "pure sample"), canonStructure("81.0", "pure sample")
	const mismatch = "canonical form does not match declared claim structure"
	text := func(n string) string {
		return "Synthetic v12 probe P2: a claim that declares its canonical form, variant " + n + "."
	}
	for _, sel := range dedupSelections {
		t.Run(sel.name, func(t *testing.T) {
			f := sel.setup(t)
			v1 := types.BuildCanonicalForm(canonAssertion, structure, dedupDomain)
			v2 := types.BuildCanonicalFormV2(canonAssertion, structure, dedupDomain)
			if !sel.integrity {
				f.requireSubmitRefused(t, v11Addr("v12-cf-declare"), text("v2"), canonClaim(canonAssertion, structure, v2), mismatch)
				id := f.mustSubmitClaim(t, v11Addr("v12-cf-declare"), text("v1"), dedupDomain, canonClaim(canonAssertion, structure, v1))
				require.Equal(t, v1, f.claim(t, id).CanonicalForm)
				f.starve(t, id)
				f.requireNoLiveRounds(t)
				return
			}
			for name, declared := range map[string]string{
				"v1":              v1,
				"lowercased v2":   strings.ToLower(v2),
				"other structure": types.BuildCanonicalFormV2(canonAssertion, other, dedupDomain),
				"other type":      types.BuildCanonicalFormV2(canonComputational, structure, dedupDomain),
			} {
				f.requireSubmitRefused(t, v11Addr("v12-cf-declare"), text(name), canonClaim(canonAssertion, structure, declared), mismatch)
			}
			id := f.mustSubmitClaim(t, v11Addr("v12-cf-declare"), text("padded v2"), dedupDomain, canonClaim(canonAssertion, structure, "\n "+v2+"\t"))
			require.Equal(t, v2, f.claim(t, id).CanonicalForm, "the stored form is the derived one")
			require.Equal(t, types.HashCanonicalForm(v2), f.claim(t, id).CanonicalHash)
			f.starve(t, id)
			f.requireNoLiveRounds(t)
		})
	}
}

// FactByCanonical resolves the form the selection derived (and its padded
// spelling and hash) to the accepted fact; the other version's form misses.
func TestClaimIntegrity_FactByCanonicalFindsV2Form(t *testing.T) {
	structure := canonStructure("78.4", "pure sample")
	for _, sel := range dedupSelections {
		t.Run(sel.name, func(t *testing.T) {
			f := sel.setup(t)
			x := f.mustSubmitClaim(t, v11Addr("v12-cf-query"), "Synthetic v12 probe P2: Lab A reports compound Z boils at 78.4 C, found again by its form.", dedupDomain,
				canonClaim(canonAssertion, structure, ""))
			factF := f.accept(t, x)
			query := keeper.NewQueryServerImpl(f.k)
			form := canonFormFor(sel.integrity, canonAssertion, structure)
			for _, req := range []*types.QueryFactByCanonicalRequest{
				{CanonicalForm: form},
				{CanonicalForm: " \n" + form + "\t"},
				{CanonicalHash: types.HashCanonicalForm(form)},
			} {
				response, err := query.FactByCanonical(f.ctx, req)
				require.NoError(t, err, "%+v", req)
				require.Equal(t, factF, response.Fact.Id)
			}
			other := canonFormFor(!sel.integrity, canonAssertion, structure)
			_, err := query.FactByCanonical(f.ctx, &types.QueryFactByCanonicalRequest{CanonicalForm: other})
			require.Equal(t, codes.NotFound, status.Code(err), "the other version's form: %s", other)

			response, err := query.FactByCanonical(f.ctx, &types.QueryFactByCanonicalRequest{
				CanonicalForm: canonFormFor(sel.integrity, canonAssertion, canonStructure("81.0", "pure sample")),
			})
			if sel.integrity {
				require.Equal(t, codes.NotFound, status.Code(err), "81.0 is another finding")
			} else {
				require.NoError(t, err, "v1 drops the object (P2)")
				require.Equal(t, factF, response.Fact.Id, "so the 81.0 form finds the 78.4 fact")
			}
			f.requireNoLiveRounds(t)
		})
	}
}

// Plan §3.6: activation rewrites no record. A pre-activation v1 entry does not
// interact with v2, so a reworded twin of a pre-v12 structured claim is
// admitted once; exact text is still caught by the content hash, and v1 forms
// still find pre-v12 facts.
func TestClaimIntegrity_LegacyV1CanonicalEntriesDoNotInteract(t *testing.T) {
	f := setupV11SemanticsTest(t)
	structure := canonStructure("78.4", "pure sample")
	const legacyText = "Synthetic v12 probe P2: a pre-activation structured claim with a v1 canonical form."
	legacy := f.mustSubmitClaim(t, v11Addr("v12-cf-legacy"), legacyText, dedupDomain, canonClaim(canonAssertion, structure, ""))
	legacyFact := f.accept(t, legacy)
	v1 := types.BuildCanonicalForm(canonAssertion, structure, dedupDomain)
	require.Equal(t, v1, f.claim(t, legacy).CanonicalForm)

	require.NoError(t, f.k.EnableClaimIntegrity(f.ctx))
	require.True(t, f.claimIntegrityEnabled(t))
	f.requireSubmitRefused(t, v11Addr("v12-cf-legacy-copy"), legacyText, canonClaim(canonAssertion, structure, ""),
		"duplicate claim: content hash matches existing claim "+legacy+" (status CLAIM_STATUS_ACCEPTED; fact "+legacyFact+" FACT_STATUS_VERIFIED)")
	twin := f.mustSubmitClaim(t, v11Addr("v12-cf-legacy-twin"), "Synthetic v12 probe P2: the same structure, reworded after activation.", dedupDomain,
		canonClaim(canonAssertion, structure, ""))
	require.Equal(t, types.BuildCanonicalFormV2(canonAssertion, structure, dedupDomain), f.claim(t, twin).CanonicalForm)
	f.requireSubmitRefused(t, v11Addr("v12-cf-legacy-third"), "Synthetic v12 probe P2: the same structure, reworded a second time.",
		canonClaim(canonAssertion, structure, ""), "canonical duplicate: matches existing claim "+twin+" (status CLAIM_STATUS_IN_VERIFICATION)")

	response, err := keeper.NewQueryServerImpl(f.k).FactByCanonical(f.ctx, &types.QueryFactByCanonicalRequest{CanonicalForm: v1})
	require.NoError(t, err)
	require.Equal(t, legacyFact, response.Fact.Id)
	f.starve(t, twin)
	f.requireNoLiveRounds(t)
}
