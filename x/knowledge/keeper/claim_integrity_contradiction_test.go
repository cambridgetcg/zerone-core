package keeper_test

import (
	"fmt"
	"math/big"
	"sort"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/zerone-chain/zerone/x/knowledge/keeper"
	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// Claim integrity, Task 10 (P3a, P1ext): contradiction admission. With the
// marker on, MsgSubmitContradiction is admitted only when:
//   - its target stands (VERIFIED, ACTIVE, CONTESTED, CHALLENGED or AT_RISK);
//   - its stake is at least EffectiveMinChallengeStake for the target's
//     confidence, the minimum for challenging that fact;
//   - its counter-claim text does not repeat the text of a live or standing
//     record in its domain (the §5.4 table, through the 0x09 entry).
// All three are checked before collateral moves, so a refusal moves no coins
// and writes nothing. With the marker off, knowledge 11 admits all three; the
// marker-off twins pin that.
//
// No assertion reads a target's status after a contradiction is admitted:
// Task 12 changes whether admission flips it. Reviewers and challengers are
// never parties to the rounds they decide, so Task 14 does not change these
// either.

// contraTarget accepts a claim in dedupDomain and returns its claim ID, its
// fact ID and the minimum stake for disputing that fact.
func (f *v11Fixture) contraTarget(t *testing.T, author, text string) (claimID, factID, minStake string) {
	t.Helper()
	claimID = f.mustSubmitClaim(t, v11Addr(author), text, dedupDomain, nil)
	factID = f.accept(t, claimID, v11Reviewers()...)
	params, err := f.k.GetParams(f.ctx)
	require.NoError(t, err)
	minimum := keeper.EffectiveMinChallengeStake(params, f.fact(t, factID).Confidence)
	require.Equal(t, 1, minimum.Cmp(big.NewInt(1)), "the minimum is above 1 uzrn")
	return claimID, factID, minimum.String()
}

// requireContradictionRefused requires an exact refusal that moves no coins
// and writes nothing.
func (f *v11Fixture) requireContradictionRefused(t *testing.T, submitter, factID, counter, stake, want string) {
	t.Helper()
	before, _ := f.storeDigest(t)
	sends := len(f.bank.sendCalls)
	_, err := f.contradict(submitter, factID, counter, dedupDomain, stake)
	require.EqualError(t, err, want)
	require.Len(t, f.bank.sendCalls, sends, "a refusal moves no coins")
	after, _ := f.storeDigest(t)
	require.Equal(t, before, after, "a refusal writes nothing")
}

// mustContradict requires an admission whose first bank call locks exactly
// the stake from the submitter.
func (f *v11Fixture) mustContradict(t *testing.T, submitter, factID, counter, stake string) string {
	t.Helper()
	sends := len(f.bank.sendCalls)
	id, err := f.contradict(submitter, factID, counter, dedupDomain, stake)
	require.NoError(t, err)
	require.Greater(t, len(f.bank.sendCalls), sends)
	lock := f.bank.sendCalls[sends]
	require.Equal(t, submitter, lock.from)
	require.Equal(t, types.ModuleName, lock.to)
	require.Equal(t, stake+"uzrn", lock.coins.String())
	require.Equal(t, types.ClaimStatus_CLAIM_STATUS_IN_VERIFICATION, f.claim(t, id).Status)
	return id
}

func contraBelowMinimum(stake, minimum string, confidence uint64) string {
	return fmt.Sprintf("contradiction stake %s below effective minimum %s (fact confidence %d; a contradiction disputes the fact, so it costs at least what challenging it costs)",
		stake, minimum, confidence)
}

func contraNotStanding(factID string, status types.FactStatus) string {
	return fmt.Sprintf("fact %s is not standing (status: %s); only a verified, active, contested, challenged or at-risk fact can be contradicted",
		factID, status)
}

// P3 B replica: a 1-uzrn contradiction, or one a single uzrn below the
// minimum, is refused with a message that states the minimum. A stake at the
// minimum is admitted. With the marker off every positive stake is admitted.
func TestClaimIntegrity_ContradictionStakeMeetsChallengeMinimum(t *testing.T) {
	for _, sel := range dedupSelections {
		t.Run(sel.name, func(t *testing.T) {
			f := sel.setup(t)
			require.Equal(t, sel.integrity, f.claimIntegrityEnabled(t))
			_, factF, minimum := f.contraTarget(t, "v12-ct-stake-author", "Synthetic v12 probe P3: an accepted fact that a contradiction disputes at some stake.")
			confidence := f.fact(t, factF).Confidence
			minimumInt, ok := new(big.Int).SetString(minimum, 10)
			require.True(t, ok)
			belowMinimum := new(big.Int).Sub(minimumInt, big.NewInt(1)).String()
			counter := func(stake string) string {
				return "Synthetic v12 probe P3: a counter-observation staked at " + stake + " uzrn."
			}

			for _, stake := range []string{"1", belowMinimum} {
				if !sel.integrity {
					f.inCache(func() { f.mustContradict(t, v11Addr("v12-ct-stake-low"), factF, counter(stake), stake) })
					continue
				}
				f.requireContradictionRefused(t, v11Addr("v12-ct-stake-low"), factF, counter(stake), stake,
					contraBelowMinimum(stake, minimum, confidence))
			}
			c := f.mustContradict(t, v11Addr("v12-ct-stake-min"), factF, counter(minimum), minimum)
			f.starve(t, c)
			f.requireNoLiveRounds(t)
		})
	}
}

// A fact refuted by a real challenge panel cannot be contradicted, and neither
// can any other fact that does not stand. Every status is applied to an
// accepted fact in a discarded cache; the five standing ones admit a
// contradiction at the minimum stake. With the marker off every status admits.
func TestClaimIntegrity_ContradictionTargetMustStand(t *testing.T) {
	standing := map[types.FactStatus]bool{
		types.FactStatus_FACT_STATUS_VERIFIED:   true,
		types.FactStatus_FACT_STATUS_ACTIVE:     true,
		types.FactStatus_FACT_STATUS_CONTESTED:  true,
		types.FactStatus_FACT_STATUS_CHALLENGED: true,
		types.FactStatus_FACT_STATUS_AT_RISK:    true,
	}
	var statuses []int
	for value := range types.FactStatus_name {
		statuses = append(statuses, int(value))
	}
	sort.Ints(statuses)
	require.Len(t, statuses, 13)

	for _, sel := range dedupSelections {
		t.Run(sel.name, func(t *testing.T) {
			f := sel.setup(t)
			_, factG, minimumG := f.contraTarget(t, "v12-ct-refuted", "Synthetic v12 probe P3: an accepted fact that a challenge panel later refutes.")
			f.accept(t, f.challenge(t, v11Addr("v12-ct-challenger"), factG), v11Reviewers()...)
			require.Equal(t, types.FactStatus_FACT_STATUS_DISPROVEN, f.fact(t, factG).Status)
			const late = "Synthetic v12 probe P3: a counter-observation against a fact already refuted."
			if sel.integrity {
				f.requireContradictionRefused(t, v11Addr("v12-ct-late"), factG, late, minimumG,
					contraNotStanding(factG, types.FactStatus_FACT_STATUS_DISPROVEN))
			} else {
				f.inCache(func() { f.mustContradict(t, v11Addr("v12-ct-late"), factG, late, minimumG) })
			}

			_, factF, minimum := f.contraTarget(t, "v12-ct-status", "Synthetic v12 probe P3: an accepted fact whose status is varied in a cache.")
			for _, value := range statuses {
				status := types.FactStatus(value)
				f.inCache(func() {
					fact := f.fact(t, factF)
					fact.Status = status
					require.NoError(t, f.k.SetFact(f.ctx, fact))
					counter := "Synthetic v12 probe P3: a counter-observation against a fact in status " + status.String() + "."
					if standing[status] || !sel.integrity {
						f.mustContradict(t, v11Addr("v12-ct-status-c"), factF, counter, minimum)
						return
					}
					f.requireContradictionRefused(t, v11Addr("v12-ct-status-c"), factF, counter, minimum, contraNotStanding(factF, status))
				})
			}
			f.requireNoLiveRounds(t)
		})
	}
}

// P1ext replica. A counter-claim repeating a live claim's exact text, in the
// same block, is refused and the 0x09 entry still names the claim. Once that
// claim is INSUFFICIENT the same contradiction is admitted, its first save
// takes the entry, and while live it blocks a copy in turn. A standing fact's
// accepted claim holds its own text too. With the marker off the
// contradiction is admitted and takes the live claim's entry (knowledge 11).
func TestClaimIntegrity_ContradictionCannotTakeLiveClaimText(t *testing.T) {
	for _, sel := range dedupSelections {
		t.Run(sel.name, func(t *testing.T) {
			f := sel.setup(t)
			const (
				targetText = "Synthetic v12 probe P1ext: an accepted target fact for the contradiction."
				text       = "Synthetic v12 probe P1ext: statement X, copied verbatim into a contradiction."
			)
			claimF, factF, minimum := f.contraTarget(t, "v12-ct-p1x-target", targetText)
			x := f.mustSubmitClaim(t, v11Addr("v12-ct-p1x-author"), text, dedupDomain, nil)
			key := dedupContentKey(text)
			require.Equal(t, x, f.entry(key))

			if !sel.integrity {
				f.inCache(func() { f.mustContradict(t, v11Addr("v12-ct-p1x-self"), factF, targetText, minimum) })
				c := f.mustContradict(t, v11Addr("v12-ct-p1x-contra"), factF, text, minimum)
				require.Equal(t, c, f.entry(key), "knowledge 11: the contradiction takes the live claim's entry")
				f.starve(t, x)
				f.starve(t, c)
				f.requireNoLiveRounds(t)
				return
			}

			f.requireContradictionRefused(t, v11Addr("v12-ct-p1x-self"), factF, targetText, minimum,
				"duplicate claim: content hash matches existing claim "+claimF+" (status CLAIM_STATUS_ACCEPTED; fact "+factF+" FACT_STATUS_VERIFIED)")
			f.requireContradictionRefused(t, v11Addr("v12-ct-p1x-contra"), factF, text, minimum,
				"duplicate claim: content hash matches existing claim "+x+" (status CLAIM_STATUS_IN_VERIFICATION)")
			require.Equal(t, x, f.entry(key), "the refusal leaves the live claim's entry")

			f.starve(t, x)
			c := f.mustContradict(t, v11Addr("v12-ct-p1x-contra"), factF, text, minimum)
			require.NotEqual(t, x, c)
			require.Equal(t, c, f.entry(key), "the admitted contradiction's first save names it")
			f.requireSubmitRefused(t, v11Addr("v12-ct-p1x-copy"), text, nil,
				"duplicate claim: content hash matches existing claim "+c+" (status CLAIM_STATUS_IN_VERIFICATION)")
			f.requireContradictionRefused(t, v11Addr("v12-ct-p1x-again"), factF, text, minimum,
				"duplicate claim: content hash matches existing claim "+c+" (status CLAIM_STATUS_IN_VERIFICATION)")
			f.starve(t, c)
			require.Equal(t, c, f.entry(key), "the starved round's re-save does not move the entry")
			require.Equal(t, types.ClaimStatus_CLAIM_STATUS_INSUFFICIENT, f.claim(t, x).Status, "the old record is kept")
			f.requireNoLiveRounds(t)
		})
	}
}
