package keeper_test

import (
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

	"github.com/zerone-chain/zerone/x/substrate_bridge/keeper"
	"github.com/zerone-chain/zerone/x/substrate_bridge/types"
)

// newUnarmedKeeper builds a keeper WITHOUT arming the dedupe flag — the only
// setup that skips it, mirroring a node that ran neither InitGenesis nor the
// substrate-dedupe-v1 handler. A bank stub is wired so a submission that gets
// past the guard can escrow its bond.
func newUnarmedKeeper(t *testing.T) (keeper.Keeper, sdk.Context) {
	t.Helper()
	storeKey := storetypes.NewKVStoreKey(types.StoreKey)
	db := dbm.NewMemDB()
	cms := store.NewCommitMultiStore(db, log.NewNopLogger(), storemetrics.NewNoOpMetrics())
	cms.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	require.NoError(t, cms.LoadLatestVersion())

	registry := codectypes.NewInterfaceRegistry()
	types.RegisterInterfaces(registry)
	cdc := codec.NewProtoCodec(registry)

	k := keeper.NewKeeper(cdc, storeKey, "authority-addr", nil, nil, &stubBankKeeper{}, nil, nil)
	ctx := sdk.NewContext(cms, cmtproto.Header{Height: 1}, false, log.NewNopLogger())
	require.NoError(t, k.SetParams(ctx, defaultSubstrateBridgeParams()))
	return k, ctx
}

// newUnarmedKeeperFull is newUnarmedKeeper with bank + vesting stubs so a mint
// (or its absence) can be asserted on the deferred-settlement path.
func newUnarmedKeeperFull(t *testing.T) (keeper.Keeper, sdk.Context, *stubBankKeeper, *stubVestingKeeper) {
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
	k := keeper.NewKeeper(cdc, storeKey, "authority-addr", nil, nil, bk, nil, vk)
	ctx := sdk.NewContext(cms, cmtproto.Header{Height: 1}, false, log.NewNopLogger())
	require.NoError(t, k.SetParams(ctx, defaultSubstrateBridgeParams()))
	return k, ctx, bk, vk
}

// sourcedLink builds a valid witness-only link for (adapter, source) with a
// correct canonical hash. fetchedAt lets replay tests vary a hashed field.
func sourcedLink(adapterID, sourceID string, fetchedAt uint64) *types.SubstrateLink {
	link := &types.SubstrateLink{
		AdapterId: adapterID,
		Source: &types.ExternalSource{
			AdapterId:      adapterID,
			SourceId:       sourceID,
			ContentHash:    []byte{0xAA, 0xBB},
			FetchedAtBlock: fetchedAt,
		},
	}
	link.LinkHash = keeper.ComputeLinkHash(link)
	return link
}

func submitSourced(t *testing.T, k keeper.Keeper, ctx sdk.Context, link *types.SubstrateLink) (*types.MsgSubmitExternalAttestationResponse, error) {
	t.Helper()
	srv := keeper.NewMsgServerImpl(k)
	return srv.SubmitExternalAttestation(sdk.WrapSDKContext(ctx), &types.MsgSubmitExternalAttestation{
		Submitter:   testSubmitter("alice"),
		AdapterId:   link.AdapterId,
		WorkClassId: "translation",
		Link:        link,
		BondUzrn:    "222000",
	})
}

func registerActiveAdapter(t *testing.T, k keeper.Keeper, ctx sdk.Context, adapterID string) {
	t.Helper()
	require.NoError(t, k.WriteAdapter(ctx, &types.AdapterRegistration{
		AdapterId: adapterID, Status: types.AdapterStatus_ADAPTER_STATUS_ACTIVE,
	}))
}

// TestSubmit_DuplicateSourceRejected pins the replay wall's front door: the
// exact same submission a second time is refused before any coin moves.
func TestSubmit_DuplicateSourceRejected(t *testing.T) {
	k, ctx := setupSubstrateBridgeKeeperWithBank(t)
	registerActiveAdapter(t, k, ctx, "agenttool-invocation-v1")

	resp, err := submitSourced(t, k, ctx, sourcedLink("agenttool-invocation-v1", "inv-42", 100))
	require.NoError(t, err)

	holder, taken := k.GetSourceRef(ctx, "agenttool-invocation-v1", "inv-42")
	require.True(t, taken)
	require.Equal(t, resp.AttestationId, holder)

	_, err = submitSourced(t, k, ctx, sourcedLink("agenttool-invocation-v1", "inv-42", 100))
	require.ErrorIs(t, err, types.ErrDuplicateSource)
}

// TestSubmit_ReplayWithVariedHashStillRejected pins the reason the wall keys
// on (adapter_id, source_id) and not link_hash: varying fetched_at_block
// produces a fresh, self-consistent hash for the same work — the replay the
// off-chain relay ledger was the only thing stopping.
func TestSubmit_ReplayWithVariedHashStillRejected(t *testing.T) {
	k, ctx := setupSubstrateBridgeKeeperWithBank(t)
	registerActiveAdapter(t, k, ctx, "agenttool-invocation-v1")

	first := sourcedLink("agenttool-invocation-v1", "inv-42", 100)
	replay := sourcedLink("agenttool-invocation-v1", "inv-42", 999)
	require.NotEqual(t, first.LinkHash, replay.LinkHash, "fixture must vary the canonical hash")

	_, err := submitSourced(t, k, ctx, first)
	require.NoError(t, err)

	_, err = submitSourced(t, k, ctx, replay)
	require.ErrorIs(t, err, types.ErrDuplicateSource)
}

// TestSubmit_SourceRequired pins the identity requirement: no source, no
// attestation — an unkeyed submission would sit outside the replay wall.
func TestSubmit_SourceRequired(t *testing.T) {
	k, ctx := setupSubstrateBridgeKeeperWithBank(t)
	registerActiveAdapter(t, k, ctx, "wiki-v1")

	noSource := &types.SubstrateLink{AdapterId: "wiki-v1"}
	noSource.LinkHash = keeper.ComputeLinkHash(noSource)
	_, err := submitSourced(t, k, ctx, noSource)
	require.ErrorIs(t, err, types.ErrSourceRequired)

	emptyID := sourcedLink("wiki-v1", "", 100)
	_, err = submitSourced(t, k, ctx, emptyID)
	require.ErrorIs(t, err, types.ErrSourceRequired)
}

// TestSubmit_AdapterIdMismatchRejected pins single-adapter identity: bond and
// allow-list checks read msg.AdapterId while validation reads link.adapter_id,
// so a split submission is refused outright.
func TestSubmit_AdapterIdMismatchRejected(t *testing.T) {
	k, ctx := setupSubstrateBridgeKeeperWithBank(t)
	registerActiveAdapter(t, k, ctx, "wiki-v1")

	link := sourcedLink("wiki-v1", "article-9", 100)
	srv := keeper.NewMsgServerImpl(k)
	_, err := srv.SubmitExternalAttestation(sdk.WrapSDKContext(ctx), &types.MsgSubmitExternalAttestation{
		Submitter:   testSubmitter("alice"),
		AdapterId:   "other-v1",
		WorkClassId: "translation",
		Link:        link,
		BondUzrn:    "222000",
	})
	require.ErrorIs(t, err, types.ErrAdapterIdMismatch)
}

// TestRejection_ReleasesSource pins the release rule: rejection punishes the
// submission (bond burned), not the work's identity — the source is free for
// an honest resubmission afterwards. Squatting a source with a junk
// submission therefore cannot lock it forever.
func TestRejection_ReleasesSource(t *testing.T) {
	k, ctx, _, _ := setupSubstrateBridgeKeeperFull(t)
	registerActiveAdapter(t, k, ctx, "wiki-v1")

	// A cited fact makes the settle path evaluate ratios (knowledge keeper is
	// nil in the fixture, so citation existence is not re-checked).
	link := sourcedLink("wiki-v1", "article-9", 100)
	link.CitedFacts = []*types.FactCitation{{FactId: "fact-1", CitationType: types.CitationType_CITATION_TYPE_CITES}}
	link.LinkHash = keeper.ComputeLinkHash(link)

	resp, err := submitSourced(t, k, ctx, link)
	require.NoError(t, err)

	// Push the stored attestation over the rejection threshold (1/1 rejected
	// = 10000 bps >= default 5000) and settle it.
	att, found := k.GetAttestation(ctx, resp.AttestationId)
	require.True(t, found)
	att.RejectedCount = 1
	require.NoError(t, k.WriteAttestation(ctx, att))
	require.NoError(t, k.SettleAttestation(ctx, resp.AttestationId))

	att, found = k.GetAttestation(ctx, resp.AttestationId)
	require.True(t, found)
	require.Equal(t, types.AttestationStatus_ATTESTATION_STATUS_REJECTED, att.Status)

	_, taken := k.GetSourceRef(ctx, "wiki-v1", "article-9")
	require.False(t, taken, "rejection must release the source reference")

	// Honest resubmission of the same work now passes the wall.
	_, err = submitSourced(t, k, ctx, sourcedLink("wiki-v1", "article-9", 200))
	require.NoError(t, err)
}

// TestSettlement_ConsumesSourceForever pins the hold rule: once the work
// settled (and could have minted), its source can never be attested again.
func TestSettlement_ConsumesSourceForever(t *testing.T) {
	k, ctx, _, _ := setupSubstrateBridgeKeeperFull(t)
	registerActiveAdapter(t, k, ctx, "agenttool-invocation-v1")

	resp, err := submitSourced(t, k, ctx, sourcedLink("agenttool-invocation-v1", "inv-42", 100))
	require.NoError(t, err)

	// Witness-only link is READY at submit; settle it (bond returns whole).
	require.NoError(t, k.SettleAttestation(ctx, resp.AttestationId))
	att, found := k.GetAttestation(ctx, resp.AttestationId)
	require.True(t, found)
	require.Equal(t, types.AttestationStatus_ATTESTATION_STATUS_SETTLED, att.Status)

	_, err = submitSourced(t, k, ctx, sourcedLink("agenttool-invocation-v1", "inv-42", 999))
	require.ErrorIs(t, err, types.ErrDuplicateSource)
}

// TestSeedSourceRefs pins the upgrade migration: live and settled history is
// indexed, rejected history is not, sourceless pre-dedupe records are
// skipped, and pre-existing replays are counted rather than overwritten.
func TestSeedSourceRefs(t *testing.T) {
	k, ctx := setupSubstrateBridgeKeeper(t)

	write := func(id string, status types.AttestationStatus, link *types.SubstrateLink) {
		t.Helper()
		require.NoError(t, k.WriteAttestation(ctx, &types.ExternalAttestation{
			AttestationId: id, AdapterId: "agenttool-invocation-v1", Submitter: "alice",
			Status: status, Link: link,
		}))
	}

	write("att-100-1", types.AttestationStatus_ATTESTATION_STATUS_SETTLED, sourcedLink("agenttool-invocation-v1", "inv-1", 10))
	write("att-100-2", types.AttestationStatus_ATTESTATION_STATUS_READY, sourcedLink("agenttool-invocation-v1", "inv-2", 11))
	write("att-100-3", types.AttestationStatus_ATTESTATION_STATUS_REJECTED, sourcedLink("agenttool-invocation-v1", "inv-3", 12))
	write("att-100-4", types.AttestationStatus_ATTESTATION_STATUS_SETTLED, nil) // pre-dedupe, no link
	// A replay that already happened before the wall existed: same source as att-100-1.
	write("att-100-5", types.AttestationStatus_ATTESTATION_STATUS_SETTLED, sourcedLink("agenttool-invocation-v1", "inv-1", 99))

	seeded, duplicates, sourceless, err := k.SeedSourceRefs(ctx)
	require.NoError(t, err)
	require.Equal(t, uint64(2), seeded)     // inv-1 (settled) + inv-2 (in-flight)
	require.Equal(t, uint64(1), duplicates) // att-100-5 lost inv-1 to att-100-1
	require.Equal(t, uint64(1), sourceless) // att-100-4 has no source

	holder, taken := k.GetSourceRef(ctx, "agenttool-invocation-v1", "inv-1")
	require.True(t, taken)
	require.Equal(t, "att-100-1", holder, "settled holder keeps the ref")

	_, taken = k.GetSourceRef(ctx, "agenttool-invocation-v1", "inv-2")
	require.True(t, taken)

	_, taken = k.GetSourceRef(ctx, "agenttool-invocation-v1", "inv-3")
	require.False(t, taken, "rejected history is released by rule")

	// Idempotent: a second pass seeds nothing new. The already-present refs
	// are re-encountered and counted as duplicates, never overwritten.
	seeded2, _, _, err := k.SeedSourceRefs(ctx)
	require.NoError(t, err)
	require.Equal(t, uint64(0), seeded2)
	holder, _ = k.GetSourceRef(ctx, "agenttool-invocation-v1", "inv-1")
	require.Equal(t, "att-100-1", holder)
}

// TestSeedSourceRefs_MintedTwinWinsHolder pins the migration-window fix: when
// a pre-wall source has both a SETTLED (already-minted) and an in-flight
// duplicate, the SETTLED one must hold the ref — even when the in-flight one
// sorts first in KV order — so the in-flight twin's later rejection can never
// free a source that already paid out.
func TestSeedSourceRefs_MintedTwinWinsHolder(t *testing.T) {
	k, ctx, _, _ := setupSubstrateBridgeKeeperFull(t)

	// att-100-8 sorts before att-100-9 lexicographically; the in-flight one
	// is deliberately the earlier ID so naive first-in-KV-order would seat it.
	// The in-flight twin carries a cited fact so its settle can be forced
	// down the REJECTED path (the release path we must prove is holder-gated).
	inflight := sourcedLink("agenttool-invocation-v1", "inv-race", 10)
	inflight.CitedFacts = []*types.FactCitation{{FactId: "f", CitationType: types.CitationType_CITATION_TYPE_CITES}}
	inflight.LinkHash = keeper.ComputeLinkHash(inflight)
	settled := sourcedLink("agenttool-invocation-v1", "inv-race", 20)
	require.NoError(t, k.WriteAttestation(ctx, &types.ExternalAttestation{
		AttestationId: "att-100-8", AdapterId: "agenttool-invocation-v1", Submitter: testSubmitter("a"),
		Status: types.AttestationStatus_ATTESTATION_STATUS_AWAITING_RESOLUTION, Link: inflight, RejectedCount: 1,
	}))
	require.NoError(t, k.WriteAttestation(ctx, &types.ExternalAttestation{
		AttestationId: "att-100-9", AdapterId: "agenttool-invocation-v1", Submitter: testSubmitter("b"),
		Status: types.AttestationStatus_ATTESTATION_STATUS_SETTLED, Link: settled,
	}))

	_, _, _, err := k.SeedSourceRefs(ctx)
	require.NoError(t, err)

	holder, taken := k.GetSourceRef(ctx, "agenttool-invocation-v1", "inv-race")
	require.True(t, taken)
	require.Equal(t, "att-100-9", holder, "the minted twin must hold the ref")

	// The in-flight twin now settles REJECTED (rejection ratio 100%). Its
	// release must NOT free the source, because it is not the holder — the
	// settled twin is.
	require.NoError(t, k.SettleAttestation(ctx, "att-100-8"))
	att, _ := k.GetAttestation(ctx, "att-100-8")
	require.Equal(t, types.AttestationStatus_ATTESTATION_STATUS_REJECTED, att.Status)

	holder, taken = k.GetSourceRef(ctx, "agenttool-invocation-v1", "inv-race")
	require.True(t, taken, "minted source must stay locked after the in-flight twin rejects")
	require.Equal(t, "att-100-9", holder)
}

// citedLink builds a cited-fact link (so settlement takes the reward path) for
// (adapter, source), with a correct canonical hash.
func citedLink(adapterID, sourceID string, fetchedAt uint64) *types.SubstrateLink {
	link := sourcedLink(adapterID, sourceID, fetchedAt)
	link.CitedFacts = []*types.FactCitation{{FactId: "fact-1", CitationType: types.CitationType_CITATION_TYPE_CITES}}
	link.LinkHash = keeper.ComputeLinkHash(link)
	return link
}

// TestSettle_NonHolderDuplicateMintsNothing pins the runtime mint guard: an
// attestation that reaches settlement without holding its source ref — a
// pre-arming duplicate of an already-claimed source — settles and returns its
// bond but mints NOTHING, so the source's single mint is never doubled.
func TestSettle_NonHolderDuplicateMintsNothing(t *testing.T) {
	k, ctx, _, vk := setupSubstrateBridgeKeeperFull(t)

	// A already holds source S (as after the seed seats a settled twin) and has
	// minted. B is a pre-arming duplicate of S that never claimed the ref.
	k.SetSourceRef(ctx, "agenttool-invocation-v1", "inv-dup", "att-A")
	require.NoError(t, k.WriteAttestation(ctx, &types.ExternalAttestation{
		AttestationId: "att-B", AdapterId: "agenttool-invocation-v1", Submitter: testSubmitter("b"),
		Status: types.AttestationStatus_ATTESTATION_STATUS_READY,
		Link:   citedLink("agenttool-invocation-v1", "inv-dup", 50),
	}))

	require.NoError(t, k.SettleAttestation(ctx, "att-B"))

	att, _ := k.GetAttestation(ctx, "att-B")
	require.Equal(t, types.AttestationStatus_ATTESTATION_STATUS_SETTLED, att.Status)
	require.Equal(t, "0", att.RewardUzrn, "a non-holder duplicate must mint nothing at settlement")
	require.Empty(t, vk.minted, "no reward minted for the non-holder duplicate")

	// The ref is untouched — A still owns the source's single mint.
	holder, _ := k.GetSourceRef(ctx, "agenttool-invocation-v1", "inv-dup")
	require.Equal(t, "att-A", holder)
}

// TestSettle_SeededLoserMintsNothing pins the migration-window fix end to end:
// two legacy in-flight twins share a source; the seed seats one as holder, and
// when the OTHER (the loser) later settles down the reward path it mints
// nothing — closing the double-mint the two-pass seed alone did not.
func TestSettle_SeededLoserMintsNothing(t *testing.T) {
	k, ctx, _, vk := setupSubstrateBridgeKeeperFull(t)

	require.NoError(t, k.WriteAttestation(ctx, &types.ExternalAttestation{
		AttestationId: "att-100-1", AdapterId: "agenttool-invocation-v1", Submitter: testSubmitter("a"),
		Status: types.AttestationStatus_ATTESTATION_STATUS_AWAITING_RESOLUTION,
		Link:   citedLink("agenttool-invocation-v1", "inv-twin", 10),
	}))
	require.NoError(t, k.WriteAttestation(ctx, &types.ExternalAttestation{
		AttestationId: "att-100-2", AdapterId: "agenttool-invocation-v1", Submitter: testSubmitter("b"),
		Status: types.AttestationStatus_ATTESTATION_STATUS_AWAITING_RESOLUTION,
		Link:   citedLink("agenttool-invocation-v1", "inv-twin", 20),
	}))

	_, _, _, err := k.SeedSourceRefs(ctx)
	require.NoError(t, err)
	holder, _ := k.GetSourceRef(ctx, "agenttool-invocation-v1", "inv-twin")
	require.Equal(t, "att-100-1", holder, "first in-flight twin holds the ref")

	// The holder settles and mints (it is the ref-holder).
	require.NoError(t, k.SettleAttestation(ctx, "att-100-1"))
	holderAtt, _ := k.GetAttestation(ctx, "att-100-1")
	require.Equal(t, types.AttestationStatus_ATTESTATION_STATUS_SETTLED, holderAtt.Status)
	require.NotEqual(t, "0", holderAtt.RewardUzrn, "the ref-holder mints normally")
	mintedAfterHolder := new(big.Int).Set(vk.minted[types.AuditBountyPoolModuleName])

	// The loser settles too but mints nothing — no second mint for inv-twin.
	require.NoError(t, k.SettleAttestation(ctx, "att-100-2"))
	loserAtt, _ := k.GetAttestation(ctx, "att-100-2")
	require.Equal(t, "0", loserAtt.RewardUzrn, "the seeded loser must mint nothing")
	require.Equal(t, 0, mintedAfterHolder.Cmp(vk.minted[types.AuditBountyPoolModuleName]),
		"total minted for the source is unchanged by the loser's settlement")
}

// TestBeginBlocker_DefersSettlementWhileUnarmed pins the plan-less-restart
// guard: settlement (a mint path reached only via BeginBlocker) must not run
// against an unseeded index, or an in-flight twin of an already-minted source
// would mint a second time. While unarmed, BeginBlocker defers — attestations
// stay in flight, nothing mints; after the handler seeds + arms, the same
// BeginBlocker settles the twin with its reward correctly suppressed.
func TestBeginBlocker_DefersSettlementWhileUnarmed(t *testing.T) {
	k, ctx, _, vk := newUnarmedKeeperFull(t)
	registerActiveAdapter(t, k, ctx, "agenttool-invocation-v1")

	// twin-1 already SETTLED and minted before the index existed (no 0x8E ref).
	require.NoError(t, k.WriteAttestation(ctx, &types.ExternalAttestation{
		AttestationId: "att-1-1", AdapterId: "agenttool-invocation-v1", Submitter: testSubmitter("a"),
		Status: types.AttestationStatus_ATTESTATION_STATUS_SETTLED,
		Link:   citedLink("agenttool-invocation-v1", "inv-X", 10),
	}))
	// twin-2 is an in-flight duplicate of the same source, awaiting the drain.
	require.NoError(t, k.WriteAttestation(ctx, &types.ExternalAttestation{
		AttestationId: "att-2-1", AdapterId: "agenttool-invocation-v1", Submitter: testSubmitter("b"),
		Status: types.AttestationStatus_ATTESTATION_STATUS_READY,
		Link:   citedLink("agenttool-invocation-v1", "inv-X", 20),
	}))
	require.False(t, k.IsDedupeArmed(ctx))

	// Unarmed BeginBlocker must defer: twin-2 stays READY, nothing mints.
	require.NoError(t, k.BeginBlocker(ctx))
	att, _ := k.GetAttestation(ctx, "att-2-1")
	require.Equal(t, types.AttestationStatus_ATTESTATION_STATUS_READY, att.Status,
		"settlement must be deferred while unarmed")
	require.Empty(t, vk.minted, "no mint against an unseeded index")

	// The handler seeds (twin-1's SETTLED ref) + arms; now settlement resumes.
	_, _, _, err := k.SeedSourceRefs(ctx)
	require.NoError(t, err)
	k.SetDedupeArmed(ctx)

	require.NoError(t, k.BeginBlocker(ctx))
	att, _ = k.GetAttestation(ctx, "att-2-1")
	require.Equal(t, types.AttestationStatus_ATTESTATION_STATUS_SETTLED, att.Status)
	require.Equal(t, "0", att.RewardUzrn, "the duplicate twin mints nothing after seed+arm")
	require.Empty(t, vk.minted, "still no second mint for inv-X")
	holder, _ := k.GetSourceRef(ctx, "agenttool-invocation-v1", "inv-X")
	require.Equal(t, "att-1-1", holder, "the pre-index minted twin holds the source")
}

// TestSweepWitnessRewards_PreUpgradeDuplicateReleasesOnce pins the witness
// release lane of the mint-once invariant: two witness-reward escrows for the
// same source, written before the escrow-time gate existed, must not both mint
// after the upgrade. The seed seats one holder; the sweep releases only that
// one and cancels the duplicate unpaid.
func TestSweepWitnessRewards_PreUpgradeDuplicateReleasesOnce(t *testing.T) {
	k, ctx, _, vk := setupSubstrateBridgeKeeperFull(t) // armed by helper
	registerActiveAdapter(t, k, ctx, "agenttool-invocation-v1")

	for _, id := range []string{"att-100-1", "att-100-2"} {
		require.NoError(t, k.WriteAttestation(ctx, &types.ExternalAttestation{
			AttestationId: id, AdapterId: "agenttool-invocation-v1", Submitter: testSubmitter(id),
			Status: types.AttestationStatus_ATTESTATION_STATUS_SETTLED,
			Link:   sourcedLink("agenttool-invocation-v1", "inv-witness", 10),
		}))
		// A pending witness reward written directly, as before the gate existed.
		k.SetWitnessPendingReward(ctx, keeper.WitnessPendingReward{
			AttestationId: id, AdapterId: "agenttool-invocation-v1",
			Recipient: testSubmitter(id), Amount: "222000", Deadline: 1,
		})
	}

	// Seed seats one holder (att-100-1); the 0x8C escrow entries are untouched.
	_, _, _, err := k.SeedSourceRefs(ctx)
	require.NoError(t, err)
	holder, _ := k.GetSourceRef(ctx, "agenttool-invocation-v1", "inv-witness")
	require.Equal(t, "att-100-1", holder)

	// Sweep past the deadline: only the ref-holder's reward mints.
	k.SweepWitnessRewards(ctx)

	require.Equal(t, 0, big.NewInt(222000).Cmp(vk.minted[types.AuditBountyPoolModuleName]),
		"only the source's ref-holder mints its single witness reward, not both")
	// The duplicate's escrow is gone (canceled, not left to retry).
	_, stillPending := k.GetWitnessPendingReward(ctx, "att-100-2")
	require.False(t, stillPending, "the non-holder duplicate's escrow is canceled")
}

// TestSubmit_FreshChainSelfArms pins the fresh-chain path: a node that never
// ran genesis arming still enforces safely on a chain with no history — the
// first-ever submission self-arms (nothing to seed) and proceeds.
func TestSubmit_FreshChainSelfArms(t *testing.T) {
	k, ctx := newUnarmedKeeper(t)
	registerActiveAdapter(t, k, ctx, "agenttool-invocation-v1")
	require.False(t, k.IsDedupeArmed(ctx))

	_, err := submitSourced(t, k, ctx, sourcedLink("agenttool-invocation-v1", "inv-1", 10))
	require.NoError(t, err)
	require.True(t, k.IsDedupeArmed(ctx), "first submission on an empty chain arms enforcement")

	// The replay wall is live immediately — the same source is now blocked.
	_, err = submitSourced(t, k, ctx, sourcedLink("agenttool-invocation-v1", "inv-1", 99))
	require.ErrorIs(t, err, types.ErrDuplicateSource)
}

// TestSubmit_ExistingChainFailsClosed pins the mis-deploy guard: a node whose
// index was never seeded but whose chain ALREADY has attestations (a plan-less
// restart of an existing chain that skipped the substrate-dedupe-v1 handler)
// refuses submissions rather than replay-mint against an empty index.
func TestSubmit_ExistingChainFailsClosed(t *testing.T) {
	k, ctx := newUnarmedKeeper(t)
	registerActiveAdapter(t, k, ctx, "agenttool-invocation-v1")

	// Simulate prior history: an attestation record exists (as after a prior
	// submission) but the index is empty and enforcement unarmed.
	require.NoError(t, k.WriteAttestation(ctx, &types.ExternalAttestation{
		AttestationId: "att-1-1", AdapterId: "agenttool-invocation-v1", Submitter: testSubmitter("old"),
		Status: types.AttestationStatus_ATTESTATION_STATUS_SETTLED, Link: sourcedLink("agenttool-invocation-v1", "inv-old", 1),
	}))
	require.False(t, k.IsDedupeArmed(ctx))

	_, err := submitSourced(t, k, ctx, sourcedLink("agenttool-invocation-v1", "inv-1", 10))
	require.ErrorIs(t, err, types.ErrDedupeNotArmed)

	// Once the handler seeds + arms, submissions resume.
	k.SetDedupeArmed(ctx)
	_, err = submitSourced(t, k, ctx, sourcedLink("agenttool-invocation-v1", "inv-1", 10))
	require.NoError(t, err)
}
