package keeper_test

// DET-4 (consensus review 2026-11-05): ProcessMetabolism recomputed each
// domain's carrying capacity for every fact, walking the whole domain's
// citations each time. It now computes capacity once per domain per epoch.
// These tests replay one metabolism epoch from identical prestate through the
// new code and through the verbatim f787860c path
// (metabolism_legacy_reference_test.go), and require the same committed store
// hash and the same events. Events from the fact loop must match in order.
// The domain_pressure_changed events after the loop come out in Go map order
// in both paths (review finding DET-6, not consensus), so those are compared
// as one group per domain.

import (
	"fmt"
	"testing"

	"cosmossdk.io/log"
	"cosmossdk.io/store"
	storemetrics "cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"
	"github.com/stretchr/testify/require"

	abci "github.com/cometbft/cometbft/abci/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/x/knowledge/keeper"
	"github.com/zerone-chain/zerone/x/knowledge/types"
)

const metabolismCacheEpoch = uint64(30)

type metabolismCacheScenario struct {
	recordIntegrity bool
	baseCapacity    uint64
	factsPerDomain  int
}

type metabolismRun struct {
	hash          string
	loopEvents    []abci.Event
	tailEvents    map[string][]abci.Event
	gas           uint64
	statusChanges map[string]int
	stratumEvents int
	physicsBefore keeper.DomainStats
	physicsAfter  keeper.DomainStats
}

// metabolismDomains: physics is a root domain cited from chemistry; biology
// sits at depth 2, so each capacity computation emits a stratum event, and has
// one mentorship graduation; geology is capture-flagged.
var metabolismDomains = []string{"physics", "biology", "geology", "chemistry"}

func runMetabolismEpoch(t *testing.T, sc metabolismCacheScenario, legacy bool) metabolismRun {
	t.Helper()

	storeKey := storetypes.NewKVStoreKey(types.StoreKey)
	db := dbm.NewMemDB()
	cms := store.NewCommitMultiStore(db, log.NewNopLogger(), storemetrics.NewNoOpMetrics())
	cms.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	require.NoError(t, cms.LoadLatestVersion())

	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	k := keeper.NewKeeper(runtime.NewKVStoreService(storeKey), cdc, "zrn1authority", newTrackingBankKeeper(), newTrackingStakingKeeper())
	ontology := newCapacityMockOntologyKeeper()
	ontology.setDepth("physics", 1)
	ontology.setDepth("biology", 2)
	ontology.setDepth("geology", 1)
	ontology.setDepth("chemistry", 1)
	k.SetOntologyKeeper(ontology)
	k.SetCaptureDefenseKeeper(&mockCaptureDefenseForCapacity{
		flagged:   map[string]bool{"geology": true},
		penalties: map[string]uint64{"geology": 300_000},
	})

	setup := sdk.NewContext(cms.CacheMultiStore(), cmtproto.Header{Height: 100}, false, log.NewNopLogger())
	gs := types.DefaultGenesis()
	if !sc.recordIntegrity {
		// The historical record semantics, as in setupKnowledgeTest.
		gs.ReviewNeutralityEnabled = false
		gs.ClaimRecordsEnabled = false
		gs.FundSettlementEnabled = false
		gs.ClaimIntegrityEnabled = false
		gs.RecordIntegrityEnabled = false
	}
	require.NoError(t, k.InitGenesis(setup, gs))
	params, err := k.GetParams(setup)
	require.NoError(t, err)
	params.DomainBaseCapacity = sc.baseCapacity
	require.NoError(t, k.SetParams(setup, params))

	for _, domain := range metabolismDomains {
		for i := 0; i < sc.factsPerDomain; i++ {
			f := &types.Fact{
				Id:              fmt.Sprintf("%s-%04d", domain, i),
				Domain:          domain,
				Content:         fmt.Sprintf("a statement about %s, number %d, long enough to add a content cost", domain, i),
				Confidence:      800_000,
				EnergyCap:       1_000_000,
				Submitter:       "zrn1submitter",
				VerifiedAtBlock: 1,
			}
			// Five kinds of fact, so the epoch moves facts in every direction.
			switch i % 5 {
			case 0: // healthy, stays
				f.Status, f.Energy = types.FactStatus_FACT_STATUS_VERIFIED, 900_000
			case 1: // drops to at-risk
				f.Status, f.Energy = types.FactStatus_FACT_STATUS_VERIFIED, 0
			case 2: // at risk long enough to expire: population falls mid-loop
				f.Status, f.Energy, f.AtRiskSinceEpoch = types.FactStatus_FACT_STATUS_AT_RISK, 0, metabolismCacheEpoch-5
			case 3: // expired long enough to be pruned
				f.Status, f.Energy, f.AtRiskSinceEpoch = types.FactStatus_FACT_STATUS_EXPIRED, 0, metabolismCacheEpoch-25
			case 4: // at risk, recovers
				f.Status, f.Energy, f.AtRiskSinceEpoch = types.FactStatus_FACT_STATUS_AT_RISK, 900_000, metabolismCacheEpoch-1
			}
			require.NoError(t, k.SetFact(setup, f))
			active := f.Status == types.FactStatus_FACT_STATUS_VERIFIED
			k.IncrementDomainFactCount(setup, domain, active, f.Energy)
		}
	}
	// Cross-domain citations into physics and biology (inbound capacity bonus).
	for i := 0; i < sc.factsPerDomain; i++ {
		src := fmt.Sprintf("chemistry-%04d", i)
		require.NoError(t, k.SetFactRelation(setup, &types.FactRelation{SourceFactId: src, TargetFactId: fmt.Sprintf("physics-%04d", i), Relation: types.RelationType_RELATION_TYPE_SUPPORTS}))
		if i%2 == 0 {
			require.NoError(t, k.SetFactRelation(setup, &types.FactRelation{SourceFactId: src, TargetFactId: fmt.Sprintf("biology-%04d", i), Relation: types.RelationType_RELATION_TYPE_SUPPORTS}))
		}
	}
	k.ApplyMentorshipDividend(setup, "biology", "zrn1mentor", "zrn1mentee")
	setup.MultiStore().(storetypes.CacheMultiStore).Write()
	cms.Commit()

	block := cms.CacheMultiStore()
	ctx := sdk.NewContext(block, cmtproto.Header{Height: 300_000}, false, log.NewNopLogger()).
		WithEventManager(sdk.NewEventManager()).
		WithGasMeter(storetypes.NewInfiniteGasMeter())
	before, _ := k.GetDomainStats(ctx, "physics")
	gasBefore := ctx.GasMeter().GasConsumed()
	if legacy {
		require.NoError(t, k.LegacyProcessMetabolismForTest(ctx, metabolismCacheEpoch))
	} else {
		require.NoError(t, k.ProcessMetabolism(ctx, metabolismCacheEpoch))
	}
	gas := ctx.GasMeter().GasConsumed() - gasBefore
	after, _ := k.GetDomainStats(ctx, "physics")
	events := ctx.EventManager().ABCIEvents()
	block.Write()
	cms.Commit()

	loop, tail := splitMetabolismEvents(t, events)
	run := metabolismRun{
		hash:          fmt.Sprintf("%X", cms.LastCommitID().Hash),
		loopEvents:    loop,
		tailEvents:    tail,
		gas:           gas,
		statusChanges: map[string]int{},
		physicsBefore: *before,
		physicsAfter:  *after,
	}
	for _, ev := range loop {
		switch ev.Type {
		case "zerone.knowledge.stratum_capacity_applied":
			run.stratumEvents++
		default:
			for _, attr := range ev.Attributes {
				if attr.Key == "new_status" {
					run.statusChanges[attr.Value]++
				}
			}
		}
	}
	return run
}

func eventAttr(ev abci.Event, key string) string {
	for _, attr := range ev.Attributes {
		if attr.Key == key {
			return attr.Value
		}
	}
	return ""
}

// splitMetabolismEvents separates the fact loop's events from the per-domain
// groups EmitDomainPressureEvent appends after it. Each group ends with
// domain_pressure_changed; for biology (depth 2) it is preceded by the two
// stratum events of its capacity and pressure reads.
func splitMetabolismEvents(t *testing.T, events []abci.Event) ([]abci.Event, map[string][]abci.Event) {
	t.Helper()
	tail := make(map[string][]abci.Event)
	i := len(events)
	for i > 0 && events[i-1].Type == "zerone.knowledge.domain_pressure_changed" {
		domain := eventAttr(events[i-1], "domain")
		size := 1
		if domain == "biology" {
			size = 3
		}
		require.GreaterOrEqual(t, i, size)
		group := events[i-size : i]
		for _, ev := range group[:size-1] {
			require.Equal(t, "zerone.knowledge.stratum_capacity_applied", ev.Type)
			require.Equal(t, domain, eventAttr(ev, "domain"))
		}
		_, seen := tail[domain]
		require.False(t, seen, "domain %s pressure event twice", domain)
		tail[domain] = group
		i -= size
	}
	return events[:i], tail
}

func requireSameMetabolismEpoch(t *testing.T, sc metabolismCacheScenario) (metabolismRun, metabolismRun) {
	t.Helper()
	cached := runMetabolismEpoch(t, sc, false)
	legacy := runMetabolismEpoch(t, sc, true)
	require.Equal(t, legacy.hash, cached.hash, "store hash differs from the f787860c path")
	require.Equal(t, len(legacy.loopEvents), len(cached.loopEvents), "loop event count differs from the f787860c path")
	for i := range legacy.loopEvents {
		require.Equal(t, legacy.loopEvents[i], cached.loopEvents[i], "loop event %d differs from the f787860c path", i)
	}
	require.Equal(t, legacy.tailEvents, cached.tailEvents, "domain pressure events differ from the f787860c path")
	return cached, legacy
}

func TestProcessMetabolism_CapacityCacheMatchesLegacy(t *testing.T) {
	for _, recordIntegrity := range []bool{true, false} {
		t.Run(fmt.Sprintf("record_integrity=%v", recordIntegrity), func(t *testing.T) {
			sc := metabolismCacheScenario{recordIntegrity: recordIntegrity, baseCapacity: 20, factsPerDomain: 40}
			cached, _ := requireSameMetabolismEpoch(t, sc)

			// The scenario must exercise what a wrong cache would get wrong:
			// facts expire and are pruned, so the live population falls
			// during the loop, and the depth-2 domain emits stratum events.
			require.Positive(t, cached.statusChanges[types.FactStatus_FACT_STATUS_EXPIRED.String()], "no fact expired: %v", cached.statusChanges)
			require.Positive(t, cached.statusChanges[types.FactStatus_FACT_STATUS_PRUNED.String()], "no fact was pruned: %v", cached.statusChanges)
			require.Less(t, cached.physicsAfter.AtRiskCount, cached.physicsBefore.AtRiskCount, "population did not fall during the epoch")
			require.Positive(t, cached.stratumEvents, "no stratum event emitted in the loop")
			require.NotEmpty(t, cached.tailEvents, "no domain pressure event")
		})
	}
}

// With DomainBaseCapacity 0, death pressure is off and the loop computes no
// capacity, so it emits no stratum event.
func TestProcessMetabolism_CapacityCacheMatchesLegacyWithoutPressure(t *testing.T) {
	cached, _ := requireSameMetabolismEpoch(t, metabolismCacheScenario{recordIntegrity: true, baseCapacity: 0, factsPerDomain: 20})
	require.Zero(t, cached.stratumEvents)
}

// The work the fix removes, measured deterministically as store gas: the old
// path walks a domain's citations once per fact, the new one once per domain.
func TestProcessMetabolism_CapacityComputedOncePerDomain(t *testing.T) {
	small := metabolismCacheScenario{recordIntegrity: true, baseCapacity: 20, factsPerDomain: 40}
	large := metabolismCacheScenario{recordIntegrity: true, baseCapacity: 20, factsPerDomain: 160}
	smallCached, smallLegacy := requireSameMetabolismEpoch(t, small)
	largeCached, largeLegacy := requireSameMetabolismEpoch(t, large)
	t.Logf("store gas, 40 facts/domain: legacy %d, cached %d", smallLegacy.gas, smallCached.gas)
	t.Logf("store gas, 160 facts/domain: legacy %d, cached %d", largeLegacy.gas, largeCached.gas)

	// Four times the facts: the old path grows super-linearly (measured 9.2x;
	// its quadratic part is the cited domains' walk), the new one linearly
	// (3.9x). Store gas is deterministic, so these bounds are stable.
	require.Greater(t, largeLegacy.gas, 8*smallLegacy.gas)
	require.Less(t, largeCached.gas, 5*smallCached.gas)
	require.Less(t, 4*largeCached.gas, largeLegacy.gas)
}
