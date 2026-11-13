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
	// moreDomains adds moreMetabolismDomains.
	moreDomains bool
	// secondEpoch runs a second epoch on the same keeper and store after
	// changing two capacity inputs: DomainBaseCapacity and physics' depth.
	secondEpoch bool
}

// metabolismEpochEvents are one epoch's events: the fact loop's, and the
// per-domain groups EmitDomainPressureEvent appends after it.
type metabolismEpochEvents struct {
	loop []abci.Event
	tail map[string][]abci.Event
}

type metabolismRun struct {
	hash          string
	loopEvents    []abci.Event
	tailEvents    map[string][]abci.Event
	second        *metabolismEpochEvents
	gas           uint64
	statusChanges map[string]int
	stratumEvents int
	stratumByDom  map[string]int
	capacities    map[string]uint64
	physicsBefore keeper.DomainStats
	physicsAfter  keeper.DomainStats
}

type metabolismDomain struct {
	name    string
	depth   uint32
	penalty uint64 // capture penalty in bps; 0 means not flagged
}

// metabolismDomains: physics is a root domain cited from chemistry; biology
// sits at depth 2, so each capacity computation emits a stratum event, and has
// one mentorship graduation; geology is capture-flagged.
var metabolismDomains = []metabolismDomain{
	{"physics", 1, 0}, {"biology", 2, 0}, {"geology", 1, 300_000}, {"chemistry", 1, 0},
}

// moreMetabolismDomains each share a name prefix with a domain above, so a
// cache keyed by less than the whole name mixes them up. Each has no inbound
// citation, so with base capacity 20:
//   - biology-marine, at depth 3, has capacity 12 and emits stratum events;
//   - geo, at depth 2, loses its whole capacity to a capture penalty and is
//     floored at 1, which returns before the stratum step, so no event;
//   - chem-trace keeps 1 after a 95% penalty, which depth 2 cuts to 0.
var moreMetabolismDomains = []metabolismDomain{
	{"biology-marine", 3, 0}, {"geo", 2, 1_000_000}, {"chem-trace", 2, 950_000},
}

func runMetabolismEpoch(t *testing.T, sc metabolismCacheScenario, legacy bool) metabolismRun {
	t.Helper()

	storeKey := storetypes.NewKVStoreKey(types.StoreKey)
	db := dbm.NewMemDB()
	cms := store.NewCommitMultiStore(db, log.NewNopLogger(), storemetrics.NewNoOpMetrics())
	cms.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	require.NoError(t, cms.LoadLatestVersion())

	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	k := keeper.NewKeeper(runtime.NewKVStoreService(storeKey), cdc, "zrn1authority", newTrackingBankKeeper(), newTrackingStakingKeeper())
	domains := metabolismDomains
	if sc.moreDomains {
		domains = append(append([]metabolismDomain{}, metabolismDomains...), moreMetabolismDomains...)
	}
	ontology := newCapacityMockOntologyKeeper()
	capture := &mockCaptureDefenseForCapacity{flagged: map[string]bool{}, penalties: map[string]uint64{}}
	for _, d := range domains {
		ontology.setDepth(d.name, d.depth)
		if d.penalty > 0 {
			capture.flagged[d.name] = true
			capture.penalties[d.name] = d.penalty
		}
	}
	k.SetOntologyKeeper(ontology)
	k.SetCaptureDefenseKeeper(capture)

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

	for _, d := range domains {
		domain := d.name
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

	capacities := capacitiesAt(k, cms, domains)

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
		loopEvents:    loop,
		tailEvents:    tail,
		gas:           gas,
		statusChanges: map[string]int{},
		stratumByDom:  map[string]int{},
		capacities:    capacities,
		physicsBefore: *before,
		physicsAfter:  *after,
	}

	if sc.secondEpoch {
		// Change two capacity inputs between the epochs, as a parameter
		// change and an ontology change would. The second epoch must use the
		// new capacities, so a cache that outlived the first epoch differs.
		change := sdk.NewContext(cms.CacheMultiStore(), cmtproto.Header{Height: 300_001}, false, log.NewNopLogger())
		params, err := k.GetParams(change)
		require.NoError(t, err)
		params.DomainBaseCapacity = 2
		require.NoError(t, k.SetParams(change, params))
		ontology.setDepth("physics", 4)
		change.MultiStore().(storetypes.CacheMultiStore).Write()
		cms.Commit()
		changed := capacitiesAt(k, cms, domains)
		require.NotEqual(t, capacities["physics"], changed["physics"], "the input change did not change physics' capacity")

		block := cms.CacheMultiStore()
		ctx := sdk.NewContext(block, cmtproto.Header{Height: 310_000}, false, log.NewNopLogger()).
			WithEventManager(sdk.NewEventManager()).
			WithGasMeter(storetypes.NewInfiniteGasMeter())
		if legacy {
			require.NoError(t, k.LegacyProcessMetabolismForTest(ctx, metabolismCacheEpoch+1))
		} else {
			require.NoError(t, k.ProcessMetabolism(ctx, metabolismCacheEpoch+1))
		}
		events := ctx.EventManager().ABCIEvents()
		block.Write()
		cms.Commit()
		loop, tail := splitMetabolismEvents(t, events)
		run.second = &metabolismEpochEvents{loop: loop, tail: tail}
	}
	run.hash = fmt.Sprintf("%X", cms.LastCommitID().Hash)

	for _, ev := range loop {
		switch ev.Type {
		case "zerone.knowledge.stratum_capacity_applied":
			run.stratumEvents++
			run.stratumByDom[eventAttr(ev, "domain")]++
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

// capacitiesAt reads each domain's carrying capacity from the last commit,
// in a context whose writes and events are discarded.
func capacitiesAt(k keeper.Keeper, cms storetypes.CommitMultiStore, domains []metabolismDomain) map[string]uint64 {
	ctx := sdk.NewContext(cms.CacheMultiStore(), cmtproto.Header{}, false, log.NewNopLogger()).
		WithEventManager(sdk.NewEventManager())
	out := make(map[string]uint64, len(domains))
	for _, d := range domains {
		out[d.name] = k.GetDomainCarryingCapacity(ctx, d.name)
	}
	return out
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
// domain_pressure_changed. Where the domain's stratum step applies, it is
// preceded by the two stratum events of its capacity and pressure reads. A
// domain without that step emits no stratum event anywhere in its epoch, so
// the loop's last event cannot be mistaken for part of its group.
func splitMetabolismEvents(t *testing.T, events []abci.Event) ([]abci.Event, map[string][]abci.Event) {
	t.Helper()
	tail := make(map[string][]abci.Event)
	i := len(events)
	for i > 0 && events[i-1].Type == "zerone.knowledge.domain_pressure_changed" {
		domain := eventAttr(events[i-1], "domain")
		j := i - 1
		for j > 0 && i-1-j < 2 && events[j-1].Type == "zerone.knowledge.stratum_capacity_applied" &&
			eventAttr(events[j-1], "domain") == domain {
			j--
		}
		require.Contains(t, []int{1, 3}, i-j, "domain %s: pressure group of %d events", domain, i-j)
		_, seen := tail[domain]
		require.False(t, seen, "domain %s pressure event twice", domain)
		tail[domain] = events[j:i]
		i = j
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
	if sc.secondEpoch {
		require.NotNil(t, cached.second)
		require.NotNil(t, legacy.second)
		require.Equal(t, legacy.second.loop, cached.second.loop, "second-epoch loop events differ from the f787860c path")
		require.Equal(t, legacy.second.tail, cached.second.tail, "second-epoch domain pressure events differ from the f787860c path")
	}
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

// Domains that share name prefixes, a capture-floored domain at depth 2 and a
// capacity-0 domain (moreMetabolismDomains). A cache keyed by less than the
// whole domain name, or a floor branch that emitted a stratum event, would
// differ from the f787860c path here.
func TestProcessMetabolism_CapacityCacheMatchesLegacyAcrossDomainShapes(t *testing.T) {
	for _, recordIntegrity := range []bool{true, false} {
		t.Run(fmt.Sprintf("record_integrity=%v", recordIntegrity), func(t *testing.T) {
			sc := metabolismCacheScenario{recordIntegrity: recordIntegrity, baseCapacity: 20, factsPerDomain: 40, moreDomains: true}
			cached, _ := requireSameMetabolismEpoch(t, sc)

			require.Equal(t, uint64(12), cached.capacities["biology-marine"])
			require.Equal(t, uint64(1), cached.capacities["geo"])
			require.Equal(t, uint64(0), cached.capacities["chem-trace"])
			require.NotEqual(t, cached.capacities["biology"], cached.capacities["biology-marine"])
			require.NotEqual(t, cached.capacities["geology"], cached.capacities["geo"])
			require.NotEqual(t, cached.capacities["chemistry"], cached.capacities["chem-trace"])
			require.Positive(t, cached.stratumByDom["biology-marine"], "no stratum event for biology-marine")
			require.Positive(t, cached.stratumByDom["chem-trace"], "no stratum event for chem-trace")
			require.Zero(t, cached.stratumByDom["geo"], "the capture floor emitted a stratum event")
		})
	}
}

// Two epochs on one keeper and store, with DomainBaseCapacity and physics'
// depth changed between them. A capacity cache that outlived its epoch, for
// example one kept on the keeper, would apply the first epoch's capacities in
// the second.
func TestProcessMetabolism_CapacityCacheDoesNotOutliveItsEpoch(t *testing.T) {
	for _, recordIntegrity := range []bool{true, false} {
		t.Run(fmt.Sprintf("record_integrity=%v", recordIntegrity), func(t *testing.T) {
			sc := metabolismCacheScenario{recordIntegrity: recordIntegrity, baseCapacity: 20, factsPerDomain: 40, moreDomains: true, secondEpoch: true}
			cached, _ := requireSameMetabolismEpoch(t, sc)
			require.NotEmpty(t, cached.second.loop, "the second epoch processed nothing")
		})
	}
}
