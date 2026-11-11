package keeper_test

import (
	"bytes"
	"fmt"
	"math/big"
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
	"github.com/zerone-chain/zerone/x/substrate_bridge/keeper"
	"github.com/zerone-chain/zerone/x/substrate_bridge/types"
)

// These tests hold the cited-reward escrow repair (consensus-repairs-v3 marker
// cited_reward_escrow_enabled) in both marker states. Marker off, a READY link
// with cited facts and no pending claims mints at settlement, as before. Marker
// on, it settles with its bond returned and nothing minted, and the reward
// computeReward gives waits in the witness escrow: released when the adapter
// is ACTIVE after the window, deferred while SUSPENDED, cancelled when
// TOMBSTONED. Links with pending claims and the rejection paths are the same
// in both states.

const (
	citedAdapter = "agenttool-invocation-v1"
	citedSource  = "inv-cited"
	citedWindow  = uint64(10)
	citedBond    = "1000000"
	// computeReward with DefaultParams (base 222,000), a 10,000 bps verified
	// ratio and AxisSubstrate 100,000: 222,000 + 222,000 x 100,000 x 5,000 /
	// 10^8 = 1,332,000. It differs from any witness reward on purpose.
	citedReward = "1332000"
)

var citedMarkerStates = []struct {
	name string
	on   bool
}{{"marker off", false}, {"marker on", true}}

type citedEscrowFixture struct {
	k         keeper.Keeper
	ctx       sdk.Context
	key       storetypes.StoreKey
	bank      *stubBankKeeper
	vest      *stubVestingKeeper
	submitter string
}

// newCitedEscrowFixture builds the full keeper (bank and vesting stubs, store
// key kept for snapshots), a 10-block window, an ACTIVE adapter that carries
// no witness reward, and a READY cited-only attestation "att-c1" that holds
// its source ref, as a real submission leaves it.
func newCitedEscrowFixture(t *testing.T, markerOn bool) *citedEscrowFixture {
	t.Helper()
	storeKey := storetypes.NewKVStoreKey(types.StoreKey)
	db := dbm.NewMemDB()
	cms := store.NewCommitMultiStore(db, log.NewNopLogger(), storemetrics.NewNoOpMetrics())
	cms.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	require.NoError(t, cms.LoadLatestVersion())

	registry := codectypes.NewInterfaceRegistry()
	types.RegisterInterfaces(registry)
	cdc := codec.NewProtoCodec(registry)

	bk := &stubBankKeeper{}
	vk := &stubVestingKeeper{}
	k := keeper.NewKeeper(cdc, storeKey, "authority-addr", nil, nil, bk, vk)
	ctx := sdk.NewContext(cms, cmtproto.Header{Height: 1}, false, log.NewNopLogger())

	params := types.DefaultParams()
	params.WitnessRewardChallengeWindowBlocks = citedWindow
	require.NoError(t, k.SetParams(ctx, params))
	k.SetDedupeArmed(ctx)
	if markerOn {
		require.NoError(t, k.EnableCitedRewardEscrow(ctx))
	}
	registerActiveAdapter(t, k, ctx, citedAdapter)

	f := &citedEscrowFixture{k: k, ctx: ctx, key: storeKey, bank: bk, vest: vk, submitter: testSubmitter("citer")}
	f.writeCited(t, "att-c1", f.submitter, citedSource, 5)
	k.SetSourceRef(ctx, citedAdapter, citedSource, "att-c1")
	return f
}

// weightedCitedLink is citedLink with AxisSubstrate 100,000, so the reward is
// citedReward.
func weightedCitedLink(sourceID string, fetchedAt uint64) *types.SubstrateLink {
	link := citedLink(citedAdapter, sourceID, fetchedAt)
	link.RecursionWeight = &types.AxisProjection{AxisSubstrate: 100_000}
	link.LinkHash = keeper.ComputeLinkHash(link)
	return link
}

func (f *citedEscrowFixture) writeCited(t *testing.T, id, submitter, sourceID string, submittedAt uint64) {
	t.Helper()
	require.NoError(t, f.k.WriteAttestation(f.ctx, &types.ExternalAttestation{
		AttestationId: id, AdapterId: citedAdapter, Submitter: submitter,
		BondUzrn: citedBond, SubmittedAtBlock: submittedAt,
		Status: types.AttestationStatus_ATTESTATION_STATUS_READY,
		Link:   weightedCitedLink(sourceID, submittedAt),
	}))
}

func (f *citedEscrowFixture) minted() string {
	m := f.vest.minted[types.AuditBountyPoolModuleName]
	if m == nil {
		return "0"
	}
	return m.String()
}

func (f *citedEscrowFixture) paid(addr string) string {
	p, ok := f.bank.payments[addr]
	if !ok {
		return "0"
	}
	return p.String()
}

func (f *citedEscrowFixture) attestation(t *testing.T, id string) *types.ExternalAttestation {
	t.Helper()
	att, found := f.k.GetAttestation(f.ctx, id)
	require.True(t, found)
	return att
}

func (f *citedEscrowFixture) setAdapterStatus(t *testing.T, status types.AdapterStatus) {
	t.Helper()
	adapter, found := f.k.GetAdapter(f.ctx, citedAdapter)
	require.True(t, found)
	adapter.Status = status
	require.NoError(t, f.k.WriteAdapter(f.ctx, adapter))
}

func eventAttributes(events sdk.Events, eventType string) []map[string]string {
	var out []map[string]string
	for _, e := range events {
		if e.Type != eventType {
			continue
		}
		attrs := map[string]string{}
		for _, a := range e.Attributes {
			attrs[a.Key] = a.Value
		}
		out = append(out, attrs)
	}
	return out
}

// Marker off, the cited-only link mints at settlement as before. Marker on, it
// settles with reward zero and its bond back whole, nothing mints, and the
// computed reward is escrowed under the window through the witness store,
// deadline index and event.
func TestCitedReward_EscrowedAtSettle(t *testing.T) {
	for _, state := range citedMarkerStates {
		t.Run(state.name, func(t *testing.T) {
			f := newCitedEscrowFixture(t, state.on)
			require.NoError(t, f.k.SettleAttestation(f.ctx, "att-c1"))

			att := f.attestation(t, "att-c1")
			require.Equal(t, types.AttestationStatus_ATTESTATION_STATUS_SETTLED, att.Status)
			settled := eventAttributes(f.ctx.EventManager().Events(), keeper.EventTypeExternalAttestationSettled)
			require.Len(t, settled, 1)
			escrowed := eventAttributes(f.ctx.EventManager().Events(), keeper.EventTypeWitnessRewardEscrowed)
			pr, pending := f.k.GetWitnessPendingReward(f.ctx, "att-c1")

			if !state.on {
				require.Equal(t, citedReward, att.RewardUzrn)
				require.Equal(t, citedReward, f.minted())
				require.Equal(t, "2332000", f.paid(f.submitter), "bond and reward at settlement")
				require.Equal(t, citedReward, settled[0][keeper.AttrRewardUzrn])
				require.False(t, pending)
				require.Empty(t, escrowed)
				return
			}

			require.Equal(t, "0", att.RewardUzrn, "nothing paid at settle")
			require.Equal(t, "0", f.minted(), "nothing minted at settle")
			require.Equal(t, citedBond, f.paid(f.submitter), "bond back whole")
			require.Equal(t, "0", settled[0][keeper.AttrRewardUzrn])

			require.True(t, pending)
			deadline := uint64(f.ctx.BlockHeight()) + citedWindow
			require.Equal(t, keeper.WitnessPendingReward{
				AttestationId: "att-c1", AdapterId: citedAdapter, Recipient: f.submitter,
				Amount: citedReward, Deadline: deadline,
			}, pr)
			require.True(t, f.ctx.KVStore(f.key).Has(types.WitnessDeadlineIndexKey(deadline, "att-c1")))
			require.Equal(t, []map[string]string{{
				keeper.AttrAttestationID: "att-c1", keeper.AttrAdapterID: citedAdapter,
				keeper.AttrRecipient: f.submitter, keeper.AttrRewardUzrn: citedReward,
				keeper.AttrDeadline: fmt.Sprint(deadline), keeper.AttrUsefulWorkCommitment: "UW",
				keeper.AttrMechanism: "M4",
			}}, escrowed)
		})
	}
}

// Marker on: the sweep is a no-op before the deadline; at the deadline, with
// the adapter ACTIVE, it mints the escrowed reward once, pays it, records it on
// the attestation and propagates lineage on what was paid. Lineage does not
// move at settle.
func TestCitedReward_ReleasedAfterWindow(t *testing.T) {
	f := newCitedEscrowFixture(t, true)
	upstream := testSubmitter("upstream")
	require.NoError(t, f.k.WriteAttestation(f.ctx, &types.ExternalAttestation{
		AttestationId: "att-up", AdapterId: citedAdapter, Submitter: upstream, SubmittedAtBlock: 1,
		Status: types.AttestationStatus_ATTESTATION_STATUS_SETTLED,
	}))
	require.NoError(t, f.k.CreateLineageEdge(f.ctx, &types.LineageEdge{
		UpstreamAttestationId: "att-up", DownstreamAttestationId: "att-c1",
		CitationType: types.CitationType_CITATION_TYPE_CITES, ContributionShareBps: 10000,
	}))

	require.NoError(t, f.k.SettleAttestation(f.ctx, "att-c1"))
	_, accrued := f.k.GetLineageAccumulator(f.ctx, "att-up")
	require.False(t, accrued, "nothing paid at settle, so no lineage")

	f.k.SweepWitnessRewards(f.ctx.WithBlockHeight(f.ctx.BlockHeight() + 5))
	require.Equal(t, "0", f.minted(), "before the deadline the sweep does nothing")

	due := f.ctx.WithBlockHeight(f.ctx.BlockHeight() + int64(citedWindow))
	f.k.SweepWitnessRewards(due)
	require.Equal(t, citedReward, f.minted())
	require.Equal(t, "2332000", f.paid(f.submitter), "bond and reward")
	require.Equal(t, citedReward, f.attestation(t, "att-c1").RewardUzrn)
	_, pending := f.k.GetWitnessPendingReward(f.ctx, "att-c1")
	require.False(t, pending)
	acc, accrued := f.k.GetLineageAccumulator(f.ctx, "att-up")
	require.True(t, accrued, "lineage propagates on the released reward")
	require.NotEqual(t, "0", acc.CumulativeUzrn)

	f.k.SweepWitnessRewards(due)
	require.Equal(t, citedReward, f.minted(), "exactly once")
	require.Equal(t, "2332000", f.paid(f.submitter))
}

// Marker on: a SUSPENDED adapter at the deadline defers the escrow one window;
// reinstatement then releases it.
func TestCitedReward_DeferredWhileSuspended(t *testing.T) {
	f := newCitedEscrowFixture(t, true)
	require.NoError(t, f.k.SettleAttestation(f.ctx, "att-c1"))
	require.NoError(t, f.k.SuspendAdapter(f.ctx, citedAdapter, "under investigation"))

	first := f.ctx.WithBlockHeight(f.ctx.BlockHeight() + int64(citedWindow))
	f.k.SweepWitnessRewards(first)
	pr, pending := f.k.GetWitnessPendingReward(f.ctx, "att-c1")
	require.True(t, pending, "deferred, not decided")
	require.Equal(t, uint64(first.BlockHeight())+citedWindow, pr.Deadline)
	require.Equal(t, citedReward, pr.Amount)
	require.Equal(t, "0", f.minted())

	f.setAdapterStatus(t, types.AdapterStatus_ADAPTER_STATUS_ACTIVE)
	f.k.SweepWitnessRewards(first.WithBlockHeight(first.BlockHeight() + int64(citedWindow)))
	require.Equal(t, citedReward, f.minted())
	require.Equal(t, "2332000", f.paid(f.submitter))
}

// Marker on: tombstoning the adapter inside the window cancels the escrow, and
// nothing ever mints. Marker off, the reward was already minted at settle.
func TestCitedReward_CancelledOnTombstone(t *testing.T) {
	for _, state := range citedMarkerStates {
		t.Run(state.name, func(t *testing.T) {
			f := newCitedEscrowFixture(t, state.on)
			require.NoError(t, f.k.SettleAttestation(f.ctx, "att-c1"))
			require.NoError(t, f.k.TombstoneAdapter(f.ctx, citedAdapter))

			_, pending := f.k.GetWitnessPendingReward(f.ctx, "att-c1")
			require.False(t, pending)
			f.k.SweepWitnessRewards(f.ctx.WithBlockHeight(f.ctx.BlockHeight() + int64(citedWindow)))

			if !state.on {
				require.Equal(t, citedReward, f.minted(), "marker off mints at settle; tombstoning cannot reach it")
				require.Equal(t, "2332000", f.paid(f.submitter))
				return
			}
			require.Equal(t, "0", f.minted())
			require.Equal(t, citedBond, f.paid(f.submitter), "only the bond, ever")
			require.Equal(t, "0", f.attestation(t, "att-c1").RewardUzrn)
			cancelled := eventAttributes(f.ctx.EventManager().Events(), keeper.EventTypeWitnessRewardCancelled)
			require.Len(t, cancelled, 1)
			require.Equal(t, "att-c1", cancelled[0][keeper.AttrAttestationID])
		})
	}
}

// An adapter's witness reward is for witness-only links. A cited-only link
// through an adapter that carries one, as the genesis adapter does, earns the
// reward computeReward gives: paid at settle with the marker off, and with the
// marker on escrowed and released at that same amount, once.
func TestCitedReward_ComputedRewardNotAdapterWitnessReward(t *testing.T) {
	for _, state := range citedMarkerStates {
		t.Run(state.name, func(t *testing.T) {
			f := newCitedEscrowFixture(t, state.on)
			adapter, found := f.k.GetAdapter(f.ctx, citedAdapter)
			require.True(t, found)
			adapter.WitnessRewardUzrn = "500"
			require.NoError(t, f.k.WriteAdapter(f.ctx, adapter))

			require.NoError(t, f.k.SettleAttestation(f.ctx, "att-c1"))
			escrowed := eventAttributes(f.ctx.EventManager().Events(), keeper.EventTypeWitnessRewardEscrowed)
			pr, pending := f.k.GetWitnessPendingReward(f.ctx, "att-c1")
			if state.on {
				require.True(t, pending)
				require.Equal(t, citedReward, pr.Amount, "the computed reward, not the adapter's witness reward")
				require.Len(t, escrowed, 1)
				require.Equal(t, citedReward, escrowed[0][keeper.AttrRewardUzrn])
				require.Equal(t, "0", f.minted())
			} else {
				require.False(t, pending)
				require.Empty(t, escrowed)
				require.Equal(t, citedReward, f.minted())
			}

			f.k.SweepWitnessRewards(f.ctx.WithBlockHeight(f.ctx.BlockHeight() + int64(citedWindow)))
			require.Equal(t, citedReward, f.minted(), "one mint of the computed reward")
			require.Equal(t, "2332000", f.paid(f.submitter), "bond and computed reward")
			require.Equal(t, citedReward, f.attestation(t, "att-c1").RewardUzrn)
			released := eventAttributes(f.ctx.EventManager().Events(), keeper.EventTypeWitnessRewardReleased)
			if state.on {
				require.Len(t, released, 1)
				require.Equal(t, citedReward, released[0][keeper.AttrRewardUzrn])
			} else {
				require.Empty(t, released)
			}
		})
	}
}

// A cited-only attestation that reaches settlement without holding its source
// ref mints nothing in either state, and with the marker on it escrows nothing
// either. The holder keeps the ref.
func TestCitedReward_NonHolderEscrowsNothing(t *testing.T) {
	for _, state := range citedMarkerStates {
		t.Run(state.name, func(t *testing.T) {
			f := newCitedEscrowFixture(t, state.on)
			dup := testSubmitter("duplicate")
			f.writeCited(t, "att-dup", dup, citedSource, 6)

			require.NoError(t, f.k.SettleAttestation(f.ctx, "att-dup"))

			att := f.attestation(t, "att-dup")
			require.Equal(t, types.AttestationStatus_ATTESTATION_STATUS_SETTLED, att.Status)
			require.Equal(t, "0", att.RewardUzrn)
			require.Equal(t, "0", f.minted())
			require.Equal(t, citedBond, f.paid(dup), "bond back whole")
			_, pending := f.k.GetWitnessPendingReward(f.ctx, "att-dup")
			require.False(t, pending, "a non-holder escrows nothing")
			require.Empty(t, eventAttributes(f.ctx.EventManager().Events(), keeper.EventTypeWitnessRewardEscrowed))
			holder, _ := f.k.GetSourceRef(f.ctx, citedAdapter, citedSource)
			require.Equal(t, "att-c1", holder)
		})
	}
}

// Marker on, the seeded-loser case: two legacy in-flight twins share a source,
// the seed seats one as holder, the holder escrows, the loser escrows nothing,
// and the sweep mints once for the source.
func TestCitedReward_SeededLoserEscrowsNothing(t *testing.T) {
	k, ctx, bk, vk := setupSubstrateBridgeKeeperFull(t)
	require.NoError(t, k.EnableCitedRewardEscrow(ctx))
	registerActiveAdapter(t, k, ctx, citedAdapter)
	a, b := testSubmitter("a"), testSubmitter("b")
	for _, twin := range []struct {
		id, submitter string
		fetchedAt     uint64
	}{{"att-100-1", a, 10}, {"att-100-2", b, 20}} {
		require.NoError(t, k.WriteAttestation(ctx, &types.ExternalAttestation{
			AttestationId: twin.id, AdapterId: citedAdapter, Submitter: twin.submitter, BondUzrn: citedBond,
			Status: types.AttestationStatus_ATTESTATION_STATUS_AWAITING_RESOLUTION,
			Link:   citedLink(citedAdapter, "inv-twin", twin.fetchedAt),
		}))
	}
	_, _, _, err := k.SeedSourceRefs(ctx)
	require.NoError(t, err)
	holder, _ := k.GetSourceRef(ctx, citedAdapter, "inv-twin")
	require.Equal(t, "att-100-1", holder)

	require.NoError(t, k.SettleAttestation(ctx, "att-100-1"))
	require.NoError(t, k.SettleAttestation(ctx, "att-100-2"))
	_, holderPending := k.GetWitnessPendingReward(ctx, "att-100-1")
	require.True(t, holderPending, "the holder escrows")
	_, loserPending := k.GetWitnessPendingReward(ctx, "att-100-2")
	require.False(t, loserPending, "the loser escrows nothing")
	require.Nil(t, vk.minted[types.AuditBountyPoolModuleName])

	window := types.DefaultParams().WitnessRewardChallengeWindowBlocks
	k.SweepWitnessRewards(ctx.WithBlockHeight(ctx.BlockHeight() + int64(window)))
	require.Equal(t, big.NewInt(222000), vk.minted[types.AuditBountyPoolModuleName], "one mint for the source")
	require.Equal(t, "1222000", bk.payments[a].String())
	require.Equal(t, citedBond, bk.payments[b].String())
}

// Regression for the order inside EscrowWitnessReward: a sourced witness-only
// link through an adapter with no witness reward still claims its free source
// ref at settlement, escrows nothing and mints nothing, in both states.
func TestWitnessReward_NoRewardAdapterStillClaimsSourceRef(t *testing.T) {
	for _, state := range citedMarkerStates {
		t.Run(state.name, func(t *testing.T) {
			f := newCitedEscrowFixture(t, state.on)
			require.NoError(t, f.k.WriteAttestation(f.ctx, &types.ExternalAttestation{
				AttestationId: "att-witness", AdapterId: citedAdapter, Submitter: testSubmitter("witness"),
				BondUzrn: citedBond, Status: types.AttestationStatus_ATTESTATION_STATUS_READY,
				Link: sourcedLink(citedAdapter, "inv-witness", 7),
			}))
			_, held := f.k.GetSourceRef(f.ctx, citedAdapter, "inv-witness")
			require.False(t, held)

			require.NoError(t, f.k.SettleAttestation(f.ctx, "att-witness"))

			holder, held := f.k.GetSourceRef(f.ctx, citedAdapter, "inv-witness")
			require.True(t, held, "the witness-only link claims its source ref as before")
			require.Equal(t, "att-witness", holder)
			_, pending := f.k.GetWitnessPendingReward(f.ctx, "att-witness")
			require.False(t, pending)
			require.Equal(t, "0", f.minted())
			require.Equal(t, "0", f.attestation(t, "att-witness").RewardUzrn)
		})
	}
}

// A cited-only link whose computed reward is zero (reachable only with
// attestation_min_bond_uzrn "0") mints nothing, escrows nothing and leaves a
// free source ref unclaimed in both states: marker off, finalizeSettle
// authorizes the source only for a positive reward; marker on,
// escrowCitedReward returns before authorizeSourceMint.
func TestCitedReward_ZeroRewardLeavesSourceRefFree(t *testing.T) {
	for _, state := range citedMarkerStates {
		t.Run(state.name, func(t *testing.T) {
			f := newCitedEscrowFixture(t, state.on)
			params := f.k.GetParams(f.ctx)
			params.AttestationMinBondUzrn = "0"
			require.NoError(t, f.k.SetParams(f.ctx, params))
			submitter := testSubmitter("zero")
			require.NoError(t, f.k.WriteAttestation(f.ctx, &types.ExternalAttestation{
				AttestationId: "att-zero", AdapterId: citedAdapter, Submitter: submitter, BondUzrn: citedBond,
				Status: types.AttestationStatus_ATTESTATION_STATUS_READY,
				Link:   citedLink(citedAdapter, "inv-zero", 11),
			}))
			_, held := f.k.GetSourceRef(f.ctx, citedAdapter, "inv-zero")
			require.False(t, held)

			require.NoError(t, f.k.SettleAttestation(f.ctx, "att-zero"))

			att := f.attestation(t, "att-zero")
			require.Equal(t, types.AttestationStatus_ATTESTATION_STATUS_SETTLED, att.Status)
			require.Equal(t, "0", att.RewardUzrn)
			require.Equal(t, "0", f.minted())
			require.Equal(t, citedBond, f.paid(submitter), "bond back whole")
			_, pending := f.k.GetWitnessPendingReward(f.ctx, "att-zero")
			require.False(t, pending)
			require.Empty(t, eventAttributes(f.ctx.EventManager().Events(), keeper.EventTypeWitnessRewardEscrowed))
			_, held = f.k.GetSourceRef(f.ctx, citedAdapter, "inv-zero")
			require.False(t, held, "a zero reward claims no source ref")
		})
	}
}

// Links with pending claims pay at settlement in both states, with no escrow.
func TestCitedReward_PendingClaimLinksPayAtSettle(t *testing.T) {
	for _, state := range citedMarkerStates {
		t.Run(state.name, func(t *testing.T) {
			f := newCitedEscrowFixture(t, state.on)
			link := weightedCitedLink("inv-pending", 8)
			link.PendingClaims = []*types.PendingClaim{{ClaimContent: "one", Domain: "d"}, {ClaimContent: "two", Domain: "d"}}
			link.LinkHash = keeper.ComputeLinkHash(link)
			submitter := testSubmitter("pending")
			require.NoError(t, f.k.WriteAttestation(f.ctx, &types.ExternalAttestation{
				AttestationId: "att-pending", AdapterId: citedAdapter, Submitter: submitter, BondUzrn: citedBond,
				Status: types.AttestationStatus_ATTESTATION_STATUS_READY, VerifiedCount: 2, Link: link,
			}))

			require.NoError(t, f.k.SettleAttestation(f.ctx, "att-pending"))

			att := f.attestation(t, "att-pending")
			require.Equal(t, types.AttestationStatus_ATTESTATION_STATUS_SETTLED, att.Status)
			require.Equal(t, citedReward, att.RewardUzrn)
			require.Equal(t, citedReward, f.minted())
			require.Equal(t, "2332000", f.paid(submitter))
			_, pending := f.k.GetWitnessPendingReward(f.ctx, "att-pending")
			require.False(t, pending)
		})
	}
}

// Both rejection paths burn the bond and escrow nothing in both states. Each
// link cites a fact and carries pending claims: two rejected (ratio 10,000 bps
// against the 5,000 threshold), or twenty unresolved (1 of 21 verified is 476
// bps, under the 1,000 floor).
func TestCitedReward_RejectionPathsUnchanged(t *testing.T) {
	for _, state := range citedMarkerStates {
		for _, tc := range []struct {
			name                string
			pending             int
			verified, rejected  uint32
			wantReasonSubstring string
		}{
			{"rejection ratio", 2, 0, 2, "rejection ratio"},
			{"verified-ratio floor", 20, 0, 0, "verified ratio"},
		} {
			t.Run(state.name+"/"+tc.name, func(t *testing.T) {
				f := newCitedEscrowFixture(t, state.on)
				link := citedLink(citedAdapter, "inv-reject", 9)
				for i := 0; i < tc.pending; i++ {
					link.PendingClaims = append(link.PendingClaims, &types.PendingClaim{ClaimContent: fmt.Sprint(i), Domain: "d"})
				}
				link.LinkHash = keeper.ComputeLinkHash(link)
				require.NoError(t, f.k.WriteAttestation(f.ctx, &types.ExternalAttestation{
					AttestationId: "att-reject", AdapterId: citedAdapter, Submitter: testSubmitter("reject"), BondUzrn: citedBond,
					Status:        types.AttestationStatus_ATTESTATION_STATUS_AWAITING_RESOLUTION,
					VerifiedCount: tc.verified, RejectedCount: tc.rejected, Link: link,
				}))

				require.NoError(t, f.k.SettleAttestation(f.ctx, "att-reject"))

				att := f.attestation(t, "att-reject")
				require.Equal(t, types.AttestationStatus_ATTESTATION_STATUS_REJECTED, att.Status)
				require.Contains(t, att.RejectionReason, tc.wantReasonSubstring)
				require.Equal(t, citedBond, f.bank.burned.String())
				require.Equal(t, "0", f.minted())
				_, pending := f.k.GetWitnessPendingReward(f.ctx, "att-reject")
				require.False(t, pending)
			})
		}
	}
}

// BeginBlocker drains a cited-only READY attestation: marker off it pays at
// once, marker on it escrows, and a later BeginBlocker releases the reward.
func TestCitedReward_BeginBlockerDrainsAndReleases(t *testing.T) {
	for _, state := range citedMarkerStates {
		t.Run(state.name, func(t *testing.T) {
			f := newCitedEscrowFixture(t, state.on)
			require.NoError(t, f.k.BeginBlocker(f.ctx))
			require.Equal(t, types.AttestationStatus_ATTESTATION_STATUS_SETTLED, f.attestation(t, "att-c1").Status)
			if state.on {
				require.Equal(t, citedBond, f.paid(f.submitter))
				require.Equal(t, "0", f.minted())
			} else {
				require.Equal(t, "2332000", f.paid(f.submitter))
			}

			require.NoError(t, f.k.BeginBlocker(f.ctx.WithBlockHeight(f.ctx.BlockHeight()+int64(citedWindow))))
			require.Equal(t, citedReward, f.minted(), "one mint, at settle (off) or at release (on)")
			require.Equal(t, "2332000", f.paid(f.submitter))
		})
	}
}

// citedMarkerReadCounter counts reads of the cited-reward escrow marker key in
// the bridge store, including reads through a CacheContext taken from it.
type citedMarkerReadCounter struct {
	storetypes.MultiStore
	key   storetypes.StoreKey
	reads *int
}

func (ms citedMarkerReadCounter) GetKVStore(key storetypes.StoreKey) storetypes.KVStore {
	kv := ms.MultiStore.GetKVStore(key)
	if key == ms.key {
		return citedMarkerCountingStore{kv, ms.reads}
	}
	return kv
}

func (ms citedMarkerReadCounter) CacheMultiStore() storetypes.CacheMultiStore {
	return citedMarkerCountingCache{ms.MultiStore.CacheMultiStore(), ms.key, ms.reads}
}

// citedMarkerCountingCache is the cache branch of citedMarkerReadCounter, so a
// marker read inside a CacheContext (finalizeSettle, releaseWitnessReward) is
// counted too. The alias names the embedded field, which may not share the
// CacheMultiStore method's name.
type citedMarkerCountingCache struct {
	cacheBranch
	key   storetypes.StoreKey
	reads *int
}

type cacheBranch = storetypes.CacheMultiStore

func (ms citedMarkerCountingCache) GetKVStore(key storetypes.StoreKey) storetypes.KVStore {
	kv := ms.cacheBranch.GetKVStore(key)
	if key == ms.key {
		return citedMarkerCountingStore{kv, ms.reads}
	}
	return kv
}

func (ms citedMarkerCountingCache) CacheMultiStore() storetypes.CacheMultiStore {
	return citedMarkerCountingCache{ms.cacheBranch.CacheMultiStore(), ms.key, ms.reads}
}

type citedMarkerCountingStore struct {
	storetypes.KVStore
	reads *int
}

func (s citedMarkerCountingStore) Get(key []byte) []byte {
	if bytes.Equal(key, types.CitedRewardEscrowEnabledKey) {
		*s.reads++
	}
	return s.KVStore.Get(key)
}

// Each entry point reads the marker exactly once, however many attestations
// BeginBlocker drains. A malformed marker is returned as a fault wrapping
// activation.ErrMarkerFault before any write: no state change, no event, no
// coin movement and no gas.
func TestCitedReward_EntryPointsReadOnceAndFaultBeforeWrites(t *testing.T) {
	for _, marker := range [][]byte{nil, {1}, {0}, {2}, {1, 1}} {
		for _, entry := range []string{"settle", "begin-block"} {
			t.Run(fmt.Sprintf("marker=%x/%s", marker, entry), func(t *testing.T) {
				f := newCitedEscrowFixture(t, bytes.Equal(marker, []byte{1}))
				if marker != nil && !bytes.Equal(marker, []byte{1}) {
					f.ctx.KVStore(f.key).Set(types.CitedRewardEscrowEnabledKey, marker)
				}
				// A second READY attestation (witness-only) and a due escrow give
				// BeginBlocker more than one settle and a sweep to do.
				require.NoError(t, f.k.WriteAttestation(f.ctx, &types.ExternalAttestation{
					AttestationId: "att-w", AdapterId: citedAdapter, Submitter: testSubmitter("w"), BondUzrn: citedBond,
					Status: types.AttestationStatus_ATTESTATION_STATUS_READY, Link: &types.SubstrateLink{},
				}))
				f.k.SetWitnessPendingReward(f.ctx, keeper.WitnessPendingReward{
					AttestationId: "att-old", AdapterId: citedAdapter, Recipient: testSubmitter("old"), Amount: "5", Deadline: 1,
				})
				before := qualificationStoreSnapshot(t, f.ctx, f.key)

				reads := 0
				ctx := f.ctx.WithMultiStore(citedMarkerReadCounter{f.ctx.MultiStore(), f.key, &reads}).
					WithGasMeter(storetypes.NewGasMeter(10_000_000)).
					WithEventManager(sdk.NewEventManager())
				var err error
				if entry == "settle" {
					err = f.k.SettleAttestation(ctx, "att-c1")
				} else {
					err = f.k.BeginBlocker(ctx)
				}
				require.Equal(t, 1, reads)

				if marker != nil && !bytes.Equal(marker, []byte{1}) {
					require.ErrorIs(t, err, activation.ErrMarkerFault)
					require.ErrorContains(t, err, "malformed cited-reward escrow marker")
					require.Zero(t, ctx.GasMeter().GasConsumed())
					require.Equal(t, before, qualificationStoreSnapshot(t, f.ctx, f.key))
					require.Empty(t, ctx.EventManager().Events())
					require.Empty(t, f.bank.payments)
					require.Nil(t, f.vest.minted)
					return
				}
				require.NoError(t, err)
				require.Equal(t, types.AttestationStatus_ATTESTATION_STATUS_SETTLED, f.attestation(t, "att-c1").Status)
				_, pending := f.k.GetWitnessPendingReward(f.ctx, "att-c1")
				require.Equal(t, marker != nil, pending, "escrowed only with the marker on")
			})
		}
	}
}
