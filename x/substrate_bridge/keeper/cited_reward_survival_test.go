package keeper_test

import (
	"bytes"
	"context"
	"fmt"
	"testing"

	dbm "github.com/cosmos/cosmos-db"

	"cosmossdk.io/log"
	"cosmossdk.io/store"
	storemetrics "cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/zerone-chain/zerone/internal/activation"
	knowledgetypes "github.com/zerone-chain/zerone/x/knowledge/types"
	"github.com/zerone-chain/zerone/x/substrate_bridge/keeper"
	"github.com/zerone-chain/zerone/x/substrate_bridge/types"
)

// These tests hold the cited-fact survival rule of the cited-reward escrow
// repair (consensus-repairs-v3 marker cited_reward_escrow_enabled). With the
// marker on, when the window of an escrowed cited-fact reward closes and the
// adapter rule would release it, the cited facts decide: cancel if one is
// missing, DISPROVEN or REVOKED; defer one window if one is CHALLENGED or
// CONTESTED; release otherwise. AT_RISK, EXPIRED and PRUNED do not stop the
// release. With the marker off the reward mints at settlement and the sweep
// reads no fact. Karma edges for the cited facts are emitted at settlement in
// both states.

// stubSurvivalKnowledgeKeeper satisfies types.KnowledgeKeeper with an
// in-memory fact map and counts reads, so a test can show where facts are and
// are not read.
type stubSurvivalKnowledgeKeeper struct {
	facts map[string]*knowledgetypes.Fact
	reads int
}

func (s *stubSurvivalKnowledgeKeeper) GetFact(_ context.Context, factID string) (*knowledgetypes.Fact, bool) {
	s.reads++
	f, ok := s.facts[factID]
	return f, ok
}

type citedSurvivalFixture struct {
	*citedEscrowFixture
	kk *stubSurvivalKnowledgeKeeper
}

// newCitedSurvivalFixture is newCitedEscrowFixture with a knowledge keeper: a
// 10-block window, an ACTIVE adapter with no witness reward, and a READY
// attestation "att-c1" that holds its source ref and cites factIDs, each a
// VERIFIED fact with an author so settlement emits its karma edge.
func newCitedSurvivalFixture(t *testing.T, markerOn bool, factIDs ...string) *citedSurvivalFixture {
	t.Helper()
	storeKey := storetypes.NewKVStoreKey(types.StoreKey)
	db := dbm.NewMemDB()
	cms := store.NewCommitMultiStore(db, log.NewNopLogger(), storemetrics.NewNoOpMetrics())
	cms.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	require.NoError(t, cms.LoadLatestVersion())

	registry := codectypes.NewInterfaceRegistry()
	types.RegisterInterfaces(registry)
	cdc := codec.NewProtoCodec(registry)

	kk := &stubSurvivalKnowledgeKeeper{facts: map[string]*knowledgetypes.Fact{}}
	for _, id := range factIDs {
		kk.facts[id] = &knowledgetypes.Fact{
			Id: id, Domain: "d", Submitter: testSubmitter("author-" + id),
			Status: knowledgetypes.FactStatus_FACT_STATUS_VERIFIED,
		}
	}
	bk := &stubBankKeeper{}
	vk := &stubVestingKeeper{}
	k := keeper.NewKeeper(cdc, storeKey, "authority-addr", kk, nil, bk, vk)
	ctx := sdk.NewContext(cms, cmtproto.Header{Height: 1}, false, log.NewNopLogger())

	params := types.DefaultParams()
	params.WitnessRewardChallengeWindowBlocks = citedWindow
	require.NoError(t, k.SetParams(ctx, params))
	k.SetDedupeArmed(ctx)
	if markerOn {
		require.NoError(t, k.EnableCitedRewardEscrow(ctx))
	}
	registerActiveAdapter(t, k, ctx, citedAdapter)

	f := &citedSurvivalFixture{
		citedEscrowFixture: &citedEscrowFixture{k: k, ctx: ctx, key: storeKey, bank: bk, vest: vk, submitter: testSubmitter("citer")},
		kk:                 kk,
	}
	require.NoError(t, k.WriteAttestation(ctx, &types.ExternalAttestation{
		AttestationId: "att-c1", AdapterId: citedAdapter, Submitter: f.submitter,
		BondUzrn: citedBond, SubmittedAtBlock: 5,
		Status: types.AttestationStatus_ATTESTATION_STATUS_READY,
		Link:   survivalLink(citedSource, 5, factIDs...),
	}))
	k.SetSourceRef(ctx, citedAdapter, citedSource, "att-c1")
	return f
}

// survivalLink is weightedCitedLink citing factIDs, so the reward is
// citedReward however many facts it cites.
func survivalLink(sourceID string, fetchedAt uint64, factIDs ...string) *types.SubstrateLink {
	link := weightedCitedLink(sourceID, fetchedAt)
	link.CitedFacts = nil
	for _, id := range factIDs {
		link.CitedFacts = append(link.CitedFacts, &types.FactCitation{FactId: id, CitationType: types.CitationType_CITATION_TYPE_CITES})
	}
	link.LinkHash = keeper.ComputeLinkHash(link)
	return link
}

// setFact sets a cited fact's status, or with missing true deletes the fact
// from the knowledge store and ignores status.
func (f *citedSurvivalFixture) setFact(id string, status knowledgetypes.FactStatus, missing bool) {
	if missing {
		delete(f.kk.facts, id)
		return
	}
	f.kk.facts[id].Status = status
}

// at is the fixture's context moved blocks past its height.
func (f *citedSurvivalFixture) at(blocks uint64) sdk.Context {
	return f.ctx.WithBlockHeight(f.ctx.BlockHeight() + int64(blocks))
}

// sweep runs the standalone sweep at ctx and returns how many facts it read.
func (f *citedSurvivalFixture) sweep(t *testing.T, ctx sdk.Context) int {
	t.Helper()
	before := f.kk.reads
	require.NoError(t, f.k.SweepWitnessRewards(ctx))
	return f.kk.reads - before
}

func (f *citedSurvivalFixture) events(eventType string) []map[string]string {
	return eventAttributes(f.ctx.EventManager().Events(), eventType)
}

// karmaEdgesFor counts the karma edges settlement emitted for a cited fact.
func (f *citedSurvivalFixture) karmaEdgesFor(factID string) int {
	n := 0
	for _, e := range f.events(types.EventTypeKarmaEdge) {
		if e["ref_id"] == factID {
			n++
		}
	}
	return n
}

// requireReleased checks the escrow paid citedReward once on top of the bond.
func (f *citedSurvivalFixture) requireReleased(t *testing.T) {
	t.Helper()
	require.Equal(t, citedReward, f.minted())
	require.Equal(t, "2332000", f.paid(f.submitter), "bond and reward")
	require.Equal(t, citedReward, f.attestation(t, "att-c1").RewardUzrn)
	_, pending := f.k.GetWitnessPendingReward(f.ctx, "att-c1")
	require.False(t, pending)
}

// requireCancelled checks the escrow is gone with nothing minted, only the
// bond paid, and one witness_reward_cancelled event giving reason.
func (f *citedSurvivalFixture) requireCancelled(t *testing.T, reason string) {
	t.Helper()
	require.Equal(t, "0", f.minted(), "nothing minted on cancel")
	require.Equal(t, citedBond, f.paid(f.submitter), "only the bond, ever")
	require.Equal(t, "0", f.attestation(t, "att-c1").RewardUzrn)
	_, pending := f.k.GetWitnessPendingReward(f.ctx, "att-c1")
	require.False(t, pending)
	require.Equal(t, []map[string]string{{
		keeper.AttrAttestationID: "att-c1", keeper.AttrAdapterID: citedAdapter,
		keeper.AttrRecipient: f.submitter, keeper.AttrReason: reason,
	}}, f.events(keeper.EventTypeWitnessRewardCancelled))
	require.Empty(t, f.events(keeper.EventTypeWitnessRewardReleased))
}

// requireDeferredTo checks the escrow still waits, whole, until deadline, and
// that nothing was decided.
func (f *citedSurvivalFixture) requireDeferredTo(t *testing.T, deadline uint64) {
	t.Helper()
	pr, pending := f.k.GetWitnessPendingReward(f.ctx, "att-c1")
	require.True(t, pending, "deferred, not decided")
	require.Equal(t, deadline, pr.Deadline)
	require.Equal(t, citedReward, pr.Amount)
	require.True(t, f.ctx.KVStore(f.key).Has(types.WitnessDeadlineIndexKey(deadline, "att-c1")))
	require.Equal(t, "0", f.minted())
	require.Empty(t, f.events(keeper.EventTypeWitnessRewardCancelled))
	require.Empty(t, f.events(keeper.EventTypeWitnessRewardReleased))
}

// requirePaidAtSettle is the marker-off outcome: the reward minted when the
// attestation settled, and a later sweep has nothing to decide.
func (f *citedSurvivalFixture) requirePaidAtSettle(t *testing.T) {
	t.Helper()
	require.Equal(t, citedReward, f.minted())
	require.Equal(t, "2332000", f.paid(f.submitter))
	_, pending := f.k.GetWitnessPendingReward(f.ctx, "att-c1")
	require.False(t, pending)
	require.Empty(t, f.events(keeper.EventTypeWitnessRewardEscrowed))
	require.Empty(t, f.events(keeper.EventTypeWitnessRewardCancelled))
}

// Every cited fact stands at the deadline: marker on, the escrow is released
// once, after reading each cited fact. Marker off, the reward was paid at
// settlement and the sweep reads no fact.
func TestCitedSurvival_ReleasedWhileEveryCitedFactStands(t *testing.T) {
	for _, state := range citedMarkerStates {
		t.Run(state.name, func(t *testing.T) {
			f := newCitedSurvivalFixture(t, state.on, "fact-1", "fact-2", "fact-3")
			f.setFact("fact-2", knowledgetypes.FactStatus_FACT_STATUS_ACTIVE, false)
			f.setFact("fact-3", knowledgetypes.FactStatus_FACT_STATUS_PROVISIONAL, false)
			require.NoError(t, f.k.SettleAttestation(f.ctx, "att-c1"))

			reads := f.sweep(t, f.at(citedWindow))
			if !state.on {
				require.Zero(t, reads, "marker off, the sweep reads no fact")
				f.requirePaidAtSettle(t)
				return
			}
			require.Equal(t, 3, reads)
			f.requireReleased(t)
			require.Len(t, f.events(keeper.EventTypeWitnessRewardReleased), 1)

			require.Zero(t, f.sweep(t, f.at(citedWindow)), "resolved escrows are not read again")
			require.Equal(t, citedReward, f.minted(), "exactly once")
		})
	}
}

// A cited fact that is DISPROVEN, REVOKED or missing at the deadline cancels
// the escrow through the existing cancellation path and event, with a reason
// that names the fact. Nothing is minted. The karma edge emitted at settlement
// stands. Marker off, the reward was paid at settlement and the sweep changes
// nothing.
func TestCitedSurvival_CancelledWhenACitedFactFalls(t *testing.T) {
	for _, state := range citedMarkerStates {
		for _, tc := range []struct {
			name    string
			status  knowledgetypes.FactStatus
			missing bool
			reason  string
		}{
			{"disproven", knowledgetypes.FactStatus_FACT_STATUS_DISPROVEN, false, "cited fact fact-1 is FACT_STATUS_DISPROVEN"},
			{"revoked", knowledgetypes.FactStatus_FACT_STATUS_REVOKED, false, "cited fact fact-1 is FACT_STATUS_REVOKED"},
			{"missing", knowledgetypes.FactStatus_FACT_STATUS_UNSPECIFIED, true, "cited fact fact-1 missing from the knowledge store"},
		} {
			t.Run(state.name+"/"+tc.name, func(t *testing.T) {
				f := newCitedSurvivalFixture(t, state.on, "fact-1")
				require.NoError(t, f.k.SettleAttestation(f.ctx, "att-c1"))
				require.Equal(t, 1, f.karmaEdgesFor("fact-1"), "recognition at settlement")

				f.setFact("fact-1", tc.status, tc.missing)
				reads := f.sweep(t, f.at(citedWindow))

				require.Equal(t, 1, f.karmaEdgesFor("fact-1"), "the karma edge is not withdrawn")
				if !state.on {
					require.Zero(t, reads)
					f.requirePaidAtSettle(t)
					return
				}
				require.Equal(t, 1, reads)
				f.requireCancelled(t, tc.reason)
			})
		}
	}
}

// A cited fact that is CHALLENGED or CONTESTED at the deadline defers the
// escrow one window, as a SUSPENDED adapter does. At the next deadline the
// sweep decides again: released if the fact stands again, cancelled if it was
// disproven meanwhile. Marker off, the reward was paid at settlement.
func TestCitedSurvival_DeferredWhileACitedFactIsChallenged(t *testing.T) {
	for _, state := range citedMarkerStates {
		for _, challenged := range []knowledgetypes.FactStatus{
			knowledgetypes.FactStatus_FACT_STATUS_CHALLENGED,
			knowledgetypes.FactStatus_FACT_STATUS_CONTESTED,
		} {
			for _, then := range []struct {
				name   string
				status knowledgetypes.FactStatus
			}{
				{"stands again", knowledgetypes.FactStatus_FACT_STATUS_VERIFIED},
				{"disproven", knowledgetypes.FactStatus_FACT_STATUS_DISPROVEN},
			} {
				t.Run(fmt.Sprintf("%s/%s/%s", state.name, challenged, then.name), func(t *testing.T) {
					f := newCitedSurvivalFixture(t, state.on, "fact-1")
					require.NoError(t, f.k.SettleAttestation(f.ctx, "att-c1"))
					f.setFact("fact-1", challenged, false)

					first := f.at(citedWindow)
					reads := f.sweep(t, first)
					if !state.on {
						require.Zero(t, reads)
						f.requirePaidAtSettle(t)
						return
					}
					require.Equal(t, 1, reads)
					second := uint64(first.BlockHeight()) + citedWindow
					f.requireDeferredTo(t, second)

					require.Zero(t, f.sweep(t, f.at(citedWindow+citedWindow-1)), "nothing is due before the new deadline")
					f.requireDeferredTo(t, second)

					f.setFact("fact-1", then.status, false)
					f.sweep(t, f.at(citedWindow+citedWindow))
					if then.status == knowledgetypes.FactStatus_FACT_STATUS_DISPROVEN {
						f.requireCancelled(t, "cited fact fact-1 is FACT_STATUS_DISPROVEN")
						return
					}
					f.requireReleased(t)
				})
			}
		}
	}
}

// A fact that lost standing without itself being found false (energy decay,
// niche competition, a disproven parent): AT_RISK, EXPIRED and PRUNED neither
// cancel nor defer the escrow.
func TestCitedSurvival_DisuseDoesNotStopRelease(t *testing.T) {
	for _, status := range []knowledgetypes.FactStatus{
		knowledgetypes.FactStatus_FACT_STATUS_AT_RISK,
		knowledgetypes.FactStatus_FACT_STATUS_EXPIRED,
		knowledgetypes.FactStatus_FACT_STATUS_PRUNED,
	} {
		t.Run(status.String(), func(t *testing.T) {
			f := newCitedSurvivalFixture(t, true, "fact-1")
			require.NoError(t, f.k.SettleAttestation(f.ctx, "att-c1"))
			f.setFact("fact-1", status, false)

			require.Equal(t, 1, f.sweep(t, f.at(citedWindow)))
			f.requireReleased(t)
			require.Empty(t, f.events(keeper.EventTypeWitnessRewardCancelled))
		})
	}
}

// The statuses the rule does not name release, as the rule's "otherwise
// release" says: SUPERSEDED, PENDING, UNSPECIFIED and a value this build does
// not know. This pins the present choice so that a change to it is deliberate.
func TestCitedSurvival_UnplacedStatusesRelease(t *testing.T) {
	for _, status := range []knowledgetypes.FactStatus{
		knowledgetypes.FactStatus_FACT_STATUS_SUPERSEDED,
		knowledgetypes.FactStatus_FACT_STATUS_PENDING,
		knowledgetypes.FactStatus_FACT_STATUS_UNSPECIFIED,
		knowledgetypes.FactStatus(99),
	} {
		t.Run(status.String(), func(t *testing.T) {
			f := newCitedSurvivalFixture(t, true, "fact-1")
			require.NoError(t, f.k.SettleAttestation(f.ctx, "att-c1"))
			f.setFact("fact-1", status, false)

			require.Equal(t, 1, f.sweep(t, f.at(citedWindow)))
			f.requireReleased(t)
			require.Empty(t, f.events(keeper.EventTypeWitnessRewardCancelled))
		})
	}
}

// The deferral has no limit: while a cited fact stays CONTESTED, every new
// deadline defers the escrow one more window, with nothing minted, no event,
// and exactly one pending entry and one deadline-index entry.
func TestCitedSurvival_DeferralRepeatsWhileContested(t *testing.T) {
	f := newCitedSurvivalFixture(t, true, "fact-1")
	require.NoError(t, f.k.SettleAttestation(f.ctx, "att-c1"))
	f.setFact("fact-1", knowledgetypes.FactStatus_FACT_STATUS_CONTESTED, false)

	for n := uint64(1); n <= 5; n++ {
		at := f.at(n * citedWindow)
		require.Equal(t, 1, f.sweep(t, at))
		f.requireDeferredTo(t, uint64(at.BlockHeight())+citedWindow)
		indexEntries := 0
		iter := f.ctx.KVStore(f.key).Iterator(types.WitnessDeadlineIndexPrefix, storetypes.PrefixEndBytes(types.WitnessDeadlineIndexPrefix))
		for ; iter.Valid(); iter.Next() {
			indexEntries++
		}
		require.NoError(t, iter.Close())
		require.Equal(t, 1, indexEntries, "one deadline-index entry after deferral %d", n)
	}
}

// A pending reward whose attestation is missing reads no fact and is
// cancelled by the release path with its own reason, in both marker states.
func TestCitedSurvival_MissingAttestationCancelledAtRelease(t *testing.T) {
	for _, state := range citedMarkerStates {
		t.Run(state.name, func(t *testing.T) {
			f := newCitedSurvivalFixture(t, state.on, "fact-1")
			f.k.SetWitnessPendingReward(f.ctx, keeper.WitnessPendingReward{
				AttestationId: "att-ghost", AdapterId: citedAdapter, Recipient: f.submitter,
				Amount: citedReward, Deadline: uint64(f.ctx.BlockHeight()) + citedWindow,
			})

			require.Zero(t, f.sweep(t, f.at(citedWindow)))
			require.Equal(t, "0", f.minted())
			_, pending := f.k.GetWitnessPendingReward(f.ctx, "att-ghost")
			require.False(t, pending)
			require.Equal(t, []map[string]string{{
				keeper.AttrAttestationID: "att-ghost", keeper.AttrAdapterID: citedAdapter,
				keeper.AttrRecipient: f.submitter, keeper.AttrReason: "attestation missing at release",
			}}, f.events(keeper.EventTypeWitnessRewardCancelled))
		})
	}
}

// Admission bounds a new link at MaxCitedFactsPerLink, but the sweep reads
// every fact a stored link cites. A link stored with one fact past the bound,
// as a link admitted before the bound existed could be, is cancelled when only
// that last fact is DISPROVEN.
func TestCitedSurvival_ReadsEveryCitedFactPastTheAdmissionBound(t *testing.T) {
	n := types.MaxCitedFactsPerLink + 1
	ids := make([]string, n)
	for i := range ids {
		ids[i] = fmt.Sprintf("fact-%d", i+1)
	}
	f := newCitedSurvivalFixture(t, true, ids...)
	require.Len(t, f.attestation(t, "att-c1").Link.CitedFacts, n)
	require.NoError(t, f.k.SettleAttestation(f.ctx, "att-c1"))
	last := ids[n-1]
	f.setFact(last, knowledgetypes.FactStatus_FACT_STATUS_DISPROVEN, false)

	require.Equal(t, n, f.sweep(t, f.at(citedWindow)))
	f.requireCancelled(t, fmt.Sprintf("cited fact %s is FACT_STATUS_DISPROVEN", last))
}

// The adapter rule decides first, exactly as before, and the cited-fact rule
// applies only where it would release.
func TestCitedSurvival_AdapterRuleTakesPrecedence(t *testing.T) {
	// A SUSPENDED adapter defers even when a cited fact has fallen, without
	// reading it. Reinstated, the next deadline cancels on the fallen fact.
	t.Run("suspended defers before a disproven fact cancels", func(t *testing.T) {
		f := newCitedSurvivalFixture(t, true, "fact-1")
		require.NoError(t, f.k.SettleAttestation(f.ctx, "att-c1"))
		f.setFact("fact-1", knowledgetypes.FactStatus_FACT_STATUS_DISPROVEN, false)
		require.NoError(t, f.k.SuspendAdapter(f.ctx, citedAdapter, "under investigation"))

		first := f.at(citedWindow)
		require.Zero(t, f.sweep(t, first), "the adapter decided; no fact read")
		f.requireDeferredTo(t, uint64(first.BlockHeight())+citedWindow)

		f.setAdapterStatus(t, types.AdapterStatus_ADAPTER_STATUS_ACTIVE)
		require.Equal(t, 1, f.sweep(t, f.at(citedWindow+citedWindow)))
		f.requireCancelled(t, "cited fact fact-1 is FACT_STATUS_DISPROVEN")
	})

	// A SUSPENDED adapter with every cited fact standing defers, and
	// reinstatement releases, as before.
	t.Run("suspended defers while facts stand", func(t *testing.T) {
		f := newCitedSurvivalFixture(t, true, "fact-1")
		require.NoError(t, f.k.SettleAttestation(f.ctx, "att-c1"))
		require.NoError(t, f.k.SuspendAdapter(f.ctx, citedAdapter, "under investigation"))

		first := f.at(citedWindow)
		require.Zero(t, f.sweep(t, first))
		f.requireDeferredTo(t, uint64(first.BlockHeight())+citedWindow)

		f.setAdapterStatus(t, types.AdapterStatus_ADAPTER_STATUS_ACTIVE)
		require.Equal(t, 1, f.sweep(t, f.at(citedWindow+citedWindow)))
		f.requireReleased(t)
	})

	// TombstoneAdapter cancels the escrow at once with its own reason, as
	// before, while every cited fact stands.
	t.Run("tombstoning cancels at once", func(t *testing.T) {
		f := newCitedSurvivalFixture(t, true, "fact-1")
		require.NoError(t, f.k.SettleAttestation(f.ctx, "att-c1"))
		require.NoError(t, f.k.TombstoneAdapter(f.ctx, citedAdapter))
		f.requireCancelled(t, "adapter tombstoned")

		require.Zero(t, f.sweep(t, f.at(citedWindow)))
		require.Len(t, f.events(keeper.EventTypeWitnessRewardCancelled), 1)
	})

	// A TOMBSTONED adapter at the deadline (the sweep's backstop) cancels with
	// the adapter reason even when the cited-fact rule alone would defer.
	t.Run("tombstoned at the deadline cancels before a challenged fact defers", func(t *testing.T) {
		f := newCitedSurvivalFixture(t, true, "fact-1")
		require.NoError(t, f.k.SettleAttestation(f.ctx, "att-c1"))
		f.setFact("fact-1", knowledgetypes.FactStatus_FACT_STATUS_CHALLENGED, false)
		f.setAdapterStatus(t, types.AdapterStatus_ADAPTER_STATUS_TOMBSTONED)

		require.Zero(t, f.sweep(t, f.at(citedWindow)))
		f.requireCancelled(t, "adapter tombstoned inside challenge window")
	})
}

// A witness-only escrow cites nothing: in both marker states it is released at
// the deadline with no fact read, whatever the knowledge store holds.
func TestCitedSurvival_WitnessOnlyEscrowUntouched(t *testing.T) {
	for _, state := range citedMarkerStates {
		t.Run(state.name, func(t *testing.T) {
			f := newCitedSurvivalFixture(t, state.on, "fact-1")
			f.setFact("fact-1", knowledgetypes.FactStatus_FACT_STATUS_DISPROVEN, false)
			adapter, found := f.k.GetAdapter(f.ctx, citedAdapter)
			require.True(t, found)
			adapter.WitnessRewardUzrn = "500"
			require.NoError(t, f.k.WriteAdapter(f.ctx, adapter))
			witness := testSubmitter("witness")
			require.NoError(t, f.k.WriteAttestation(f.ctx, &types.ExternalAttestation{
				AttestationId: "att-w", AdapterId: citedAdapter, Submitter: witness, BondUzrn: citedBond,
				Status: types.AttestationStatus_ATTESTATION_STATUS_READY,
				Link:   sourcedLink(citedAdapter, "inv-witness", 7),
			}))

			before := f.kk.reads
			require.NoError(t, f.k.SettleAttestation(f.ctx, "att-w"))
			require.Equal(t, before, f.kk.reads)
			pr, pending := f.k.GetWitnessPendingReward(f.ctx, "att-w")
			require.True(t, pending)
			require.Equal(t, "500", pr.Amount)

			require.Zero(t, f.sweep(t, f.at(citedWindow)))
			require.Equal(t, "500", f.minted())
			require.Equal(t, "1000500", f.paid(witness))
			require.Equal(t, "500", f.attestation(t, "att-w").RewardUzrn)
			require.Empty(t, f.events(keeper.EventTypeWitnessRewardCancelled))
		})
	}
}

// Several cited facts: the sweep reads them in citation order. A fallen fact
// cancels even when it is the last one, and outranks an earlier challenged
// fact; the first fallen fact is the one named, and the sweep stops reading
// there. A challenged fact last of all defers.
func TestCitedSurvival_SeveralCitedFacts(t *testing.T) {
	disproven := knowledgetypes.FactStatus_FACT_STATUS_DISPROVEN
	revoked := knowledgetypes.FactStatus_FACT_STATUS_REVOKED
	challenged := knowledgetypes.FactStatus_FACT_STATUS_CHALLENGED
	for _, tc := range []struct {
		name     string
		statuses map[string]knowledgetypes.FactStatus
		reads    int
		reason   string // empty: deferred
	}{
		{"only the last disproven", map[string]knowledgetypes.FactStatus{"fact-3": disproven}, 3, "cited fact fact-3 is FACT_STATUS_DISPROVEN"},
		{"only the last challenged", map[string]knowledgetypes.FactStatus{"fact-3": challenged}, 3, ""},
		{"first challenged, last disproven", map[string]knowledgetypes.FactStatus{"fact-1": challenged, "fact-3": disproven}, 3, "cited fact fact-3 is FACT_STATUS_DISPROVEN"},
		{"first disproven, last revoked", map[string]knowledgetypes.FactStatus{"fact-1": disproven, "fact-3": revoked}, 1, "cited fact fact-1 is FACT_STATUS_DISPROVEN"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newCitedSurvivalFixture(t, true, "fact-1", "fact-2", "fact-3")
			require.NoError(t, f.k.SettleAttestation(f.ctx, "att-c1"))
			for id, status := range tc.statuses {
				f.setFact(id, status, false)
			}

			first := f.at(citedWindow)
			require.Equal(t, tc.reads, f.sweep(t, first))
			if tc.reason == "" {
				f.requireDeferredTo(t, uint64(first.BlockHeight())+citedWindow)
				return
			}
			f.requireCancelled(t, tc.reason)
		})
	}
}

// Marker off, the sweep is the one that ran before the marker existed: an
// escrow whose attestation cites facts is released at the deadline whatever
// those facts say, and no fact is read. (Settlement writes no such escrow with
// the marker off; this pins the sweep itself.)
func TestCitedSurvival_MarkerOffSweepReadsNoFact(t *testing.T) {
	for _, tc := range []struct {
		name    string
		status  knowledgetypes.FactStatus
		missing bool
	}{
		{"disproven", knowledgetypes.FactStatus_FACT_STATUS_DISPROVEN, false},
		{"missing", knowledgetypes.FactStatus_FACT_STATUS_UNSPECIFIED, true},
		{"challenged", knowledgetypes.FactStatus_FACT_STATUS_CHALLENGED, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newCitedSurvivalFixture(t, false, "fact-1")
			att := f.attestation(t, "att-c1")
			att.Status = types.AttestationStatus_ATTESTATION_STATUS_SETTLED
			att.RewardUzrn = "0"
			require.NoError(t, f.k.WriteAttestation(f.ctx, att))
			f.k.SetWitnessPendingReward(f.ctx, keeper.WitnessPendingReward{
				AttestationId: "att-c1", AdapterId: citedAdapter, Recipient: f.submitter,
				Amount: citedReward, Deadline: uint64(f.ctx.BlockHeight()) + citedWindow,
			})
			f.setFact("fact-1", tc.status, tc.missing)

			require.Zero(t, f.sweep(t, f.at(citedWindow)))
			require.Equal(t, citedReward, f.minted())
			require.Equal(t, citedReward, f.paid(f.submitter))
			_, pending := f.k.GetWitnessPendingReward(f.ctx, "att-c1")
			require.False(t, pending)
			require.Empty(t, f.events(keeper.EventTypeWitnessRewardCancelled))
		})
	}
}

// BeginBlocker passes the marker it read to the sweep: marker on, a cited
// escrow whose fact was disproven inside the window is cancelled at the
// deadline. Marker off, the first BeginBlocker paid the reward.
func TestCitedSurvival_BeginBlockerCancelsOnFallenFact(t *testing.T) {
	for _, state := range citedMarkerStates {
		t.Run(state.name, func(t *testing.T) {
			f := newCitedSurvivalFixture(t, state.on, "fact-1")
			require.NoError(t, f.k.BeginBlocker(f.ctx))
			require.Equal(t, types.AttestationStatus_ATTESTATION_STATUS_SETTLED, f.attestation(t, "att-c1").Status)
			f.setFact("fact-1", knowledgetypes.FactStatus_FACT_STATUS_DISPROVEN, false)

			require.NoError(t, f.k.BeginBlocker(f.at(citedWindow)))
			if !state.on {
				f.requirePaidAtSettle(t)
				return
			}
			f.requireCancelled(t, "cited fact fact-1 is FACT_STATUS_DISPROVEN")
		})
	}
}

// The standalone sweep reads the marker exactly once. A malformed marker is
// returned as a fault wrapping activation.ErrMarkerFault before any write and
// before any fact is read: no state change, no event, no coin movement and no
// gas. BeginBlocker reads it once for settlement and sweep together.
func TestCitedSurvival_SweepReadsMarkerOnceAndFaultPropagates(t *testing.T) {
	for _, marker := range [][]byte{nil, {1}, {0}, {2}, {1, 1}} {
		for _, entry := range []string{"sweep", "begin-block"} {
			t.Run(fmt.Sprintf("marker=%x/%s", marker, entry), func(t *testing.T) {
				on := bytes.Equal(marker, []byte{1})
				f := newCitedSurvivalFixture(t, on, "fact-1")
				require.NoError(t, f.k.SettleAttestation(f.ctx, "att-c1"))
				f.setFact("fact-1", knowledgetypes.FactStatus_FACT_STATUS_DISPROVEN, false)
				if marker != nil && !on {
					f.ctx.KVStore(f.key).Set(types.CitedRewardEscrowEnabledKey, marker)
				}
				mintedBefore, paidBefore := f.minted(), f.paid(f.submitter)
				before := qualificationStoreSnapshot(t, f.ctx, f.key)
				factReads := f.kk.reads

				reads := 0
				ctx := f.at(citedWindow).
					WithMultiStore(citedMarkerReadCounter{f.ctx.MultiStore(), f.key, &reads}).
					WithGasMeter(storetypes.NewGasMeter(10_000_000)).
					WithEventManager(sdk.NewEventManager())
				var err error
				if entry == "sweep" {
					err = f.k.SweepWitnessRewards(ctx)
				} else {
					err = f.k.BeginBlocker(ctx)
				}
				require.Equal(t, 1, reads)

				if marker != nil && !on {
					require.ErrorIs(t, err, activation.ErrMarkerFault)
					require.ErrorContains(t, err, "malformed cited-reward escrow marker")
					require.Zero(t, ctx.GasMeter().GasConsumed())
					require.Equal(t, before, qualificationStoreSnapshot(t, f.ctx, f.key))
					require.Empty(t, ctx.EventManager().Events())
					require.Equal(t, factReads, f.kk.reads, "no fact read")
					require.Equal(t, mintedBefore, f.minted())
					require.Equal(t, paidBefore, f.paid(f.submitter))
					return
				}
				require.NoError(t, err)
				if !on {
					f.requirePaidAtSettle(t)
					return
				}
				_, pending := f.k.GetWitnessPendingReward(f.ctx, "att-c1")
				require.False(t, pending)
				require.Equal(t, "0", f.minted())
				cancelled := eventAttributes(ctx.EventManager().Events(), keeper.EventTypeWitnessRewardCancelled)
				require.Len(t, cancelled, 1)
				require.Equal(t, "cited fact fact-1 is FACT_STATUS_DISPROVEN", cancelled[0][keeper.AttrReason])
			})
		}
	}
}
