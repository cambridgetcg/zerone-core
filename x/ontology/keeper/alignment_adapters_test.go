package keeper_test

import (
	"bytes"
	"context"
	"fmt"
	"math/big"
	"testing"

	"cosmossdk.io/log"
	sdkmath "cosmossdk.io/math"
	"cosmossdk.io/store"
	storemetrics "cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	alignment "github.com/zerone-chain/zerone/x/alignment"
	alignmentkeeper "github.com/zerone-chain/zerone/x/alignment/keeper"
	alignmenttypes "github.com/zerone-chain/zerone/x/alignment/types"
	knowledgekeeper "github.com/zerone-chain/zerone/x/knowledge/keeper"
	knowledgetypes "github.com/zerone-chain/zerone/x/knowledge/types"
	"github.com/zerone-chain/zerone/x/ontology/keeper"
	"github.com/zerone-chain/zerone/x/ontology/types"
)

// These fixed observations isolate the ontology signal, not actual stake or supply.
type domainCountStaking struct{}

func (domainCountStaking) GetTotalStaked(context.Context) *big.Int        { return big.NewInt(1) }
func (domainCountStaking) GetActiveValidatorCount(context.Context) uint64 { return 1 }
func (domainCountStaking) GetTargetValidatorCount(context.Context) uint64 { return 4 }

type domainCountSupply struct{}

func (domainCountSupply) GetTotalSupply(context.Context) *big.Int { return big.NewInt(100) }

type domainCountFlags uint64

func (f domainCountFlags) GetFlaggedDomainCount(context.Context) uint64 { return uint64(f) }

type domainCountFixture struct {
	ctx       sdk.Context
	ontology  keeper.Keeper
	alignment alignmentkeeper.Keeper
	knowledge knowledgekeeper.Keeper
	module    alignment.AppModule
	bank      *mockBankKeeper
	key       *storetypes.KVStoreKey
}

// Real in-memory stores, ontology message handling, alignment EndBlock and
// knowledge pacing readers; no node, signed transport or actual bank custody.
func newDomainCountFixture(t *testing.T, domains []*types.Domain) domainCountFixture {
	t.Helper()
	db := dbm.NewMemDB()
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	ms := store.NewCommitMultiStore(db, log.NewNopLogger(), storemetrics.NewNoOpMetrics())
	ms.SetIAVLSyncPruning(true)
	keys := storetypes.NewKVStoreKeys(types.StoreKey, alignmenttypes.StoreKey, knowledgetypes.StoreKey)
	for _, key := range keys {
		ms.MountStoreWithDB(key, storetypes.StoreTypeIAVL, db)
	}
	require.NoError(t, ms.LoadLatestVersion())
	ctx := sdk.NewContext(ms, cmtproto.Header{Height: 100}, false, log.NewNopLogger())
	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	bank := newMockBankKeeper()
	ok := keeper.NewKeeper(cdc, runtime.NewKVStoreService(keys[types.StoreKey]), bank, "authority")
	genesis := types.DefaultGenesis()
	genesis.Domains = domains
	ok.InitGenesis(ctx, genesis)
	kk := knowledgekeeper.NewKeeper(runtime.NewKVStoreService(keys[knowledgetypes.StoreKey]), cdc, "authority", nil, nil)
	kp := knowledgetypes.DefaultParams()
	kp.MinReviewFee = "100000"
	kp.ClaimCooldownBlocks = 50
	require.NoError(t, kk.SetParams(ctx, &kp))
	ak := alignmentkeeper.NewKeeper(runtime.NewKVStoreService(keys[alignmenttypes.StoreKey]), cdc, "authority",
		knowledgekeeper.NewAlignmentKnowledgeAdapter(kk), domainCountStaking{}, keeper.NewAlignmentOntologyAdapter(ok), nil, domainCountSupply{})
	ak.InitGenesis(ctx, alignmenttypes.DefaultGenesis())
	kk.SetPacingKeeper(alignmentkeeper.NewAlignmentPacingAdapter(ak))
	return domainCountFixture{ctx: ctx, ontology: ok, alignment: ak, knowledge: kk,
		module: alignment.NewAppModule(cdc, ak), bank: bank, key: keys[types.StoreKey]}
}

func (f domainCountFixture) count() uint64 {
	return keeper.NewAlignmentOntologyAdapter(f.ontology).GetDomainCount(f.ctx)
}

func (f domainCountFixture) rawOntology(t *testing.T) map[string][]byte {
	t.Helper()
	rows := make(map[string][]byte)
	it := f.ctx.KVStore(f.key).Iterator(nil, nil)
	defer func() { require.NoError(t, it.Close()) }()
	for ; it.Valid(); it.Next() {
		rows[string(it.Key())] = bytes.Clone(it.Value())
	}
	return rows
}

func (f domainCountFixture) propose(t *testing.T, name string) string {
	t.Helper()
	proposer := testAddr("count")
	f.bank.fundAccount(proposer, sdkmath.NewInt(1_000_000))
	response, err := keeper.NewMsgServerImpl(f.ontology).ProposeDomain(f.ctx, &types.MsgProposeDomain{
		Proposer: proposer, Name: name, DisplayName: name, Description: "Active-domain count regression",
		Stratum: uint32(types.StratumEmpirical), Stake: "1000000",
	})
	require.NoError(t, err)
	proposal, found := f.ontology.GetProposal(f.ctx, response.ProposalId)
	require.True(t, found)
	require.Zero(t, proposal.VotesFor)
	domain, found := f.ontology.GetDomain(f.ctx, name)
	require.True(t, found)
	require.Equal(t, "proposed", domain.Status)
	return response.ProposalId
}

func TestAlignmentDomainCountLifecycleStates(t *testing.T) {
	for _, tc := range []struct {
		name     string
		statuses []string
		want     uint64
	}{
		{name: "empty"},
		{name: "active", statuses: []string{"active", "active"}, want: 2},
		{name: "proposed", statuses: []string{"proposed"}},
		{name: "deprecated", statuses: []string{"deprecated"}},
		{name: "archived", statuses: []string{"archived"}},
		{name: "unknown is not active", statuses: []string{"", "ACTIVE", "unknown"}},
		{name: "mixed", statuses: []string{"proposed", "active", "deprecated", "archived", "active", ""}, want: 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			domains := make([]*types.Domain, 0, len(tc.statuses))
			for i, status := range tc.statuses {
				domains = append(domains, &types.Domain{Name: fmt.Sprintf("domain_%d", i), Status: status, Stratum: uint32(types.StratumEmpirical)})
			}
			f := newDomainCountFixture(t, domains)
			before := f.rawOntology(t)
			assert.Equal(t, tc.want, f.count())
			// All-status readers and the capacity guard must retain inactive records.
			require.Len(t, f.ontology.GetAllDomains(f.ctx), len(domains))
			require.Equal(t, uint32(len(domains)), f.ontology.CountDomainsInStratum(f.ctx, types.StratumEmpirical))
			query, err := keeper.NewQueryServerImpl(f.ontology).AllDomains(f.ctx, &types.QueryAllDomainsRequest{})
			require.NoError(t, err)
			require.Len(t, query.Domains, len(domains))
			exported := f.ontology.ExportGenesis(f.ctx)
			require.Len(t, exported.Domains, len(domains))
			for _, domain := range domains {
				retained, found := f.ontology.GetDomain(f.ctx, domain.Name)
				require.True(t, found)
				require.True(t, proto.Equal(domain, retained))
			}
			require.Equal(t, before, f.rawOntology(t), "counting must not rewrite legacy state")
			// Existing exported proposed/deprecated records need no cleanup migration.
			reloaded := newDomainCountFixture(t, exported.Domains)
			assert.Equal(t, tc.want, reloaded.count())
		})
	}
}

func TestAlignmentDomainCountLifecycleTransitions(t *testing.T) {
	f := newDomainCountFixture(t, types.DefaultDomains())
	require.Equal(t, uint64(7), f.count())
	proposalID := f.propose(t, "count_lifecycle")
	assert.Equal(t, uint64(7), f.count(), "an unendorsed proposal is not active")
	server := keeper.NewMsgServerImpl(f.ontology)
	for i := uint32(0); i < f.ontology.GetParams(f.ctx).MinEndorsements; i++ {
		_, err := server.VoteDomainProposal(f.ctx, &types.MsgVoteDomainProposal{
			Voter: testAddr(fmt.Sprintf("v%d", i)), ProposalId: proposalID, Approve: true,
		})
		require.NoError(t, err)
	}
	require.Equal(t, uint64(8), f.count(), "passed proposal becomes active")
	require.NoError(t, f.ontology.DeprecateDomain(f.ctx, "count_lifecycle"))
	assert.Equal(t, uint64(7), f.count())
	require.NoError(t, f.ontology.ActivateDomain(f.ctx, "count_lifecycle"))
	require.Equal(t, uint64(8), f.count())
	require.NoError(t, f.ontology.ArchiveDomain(f.ctx, "count_lifecycle"))
	assert.Equal(t, uint64(7), f.count())
}

func TestAlignmentProposedDomainsDoNotChangeHealthOrPacing(t *testing.T) {
	f := newDomainCountFixture(t, types.DefaultDomains())
	require.NoError(t, f.module.EndBlock(f.ctx))
	before, found := f.alignment.GetHealthIndex(f.ctx, 100)
	require.True(t, found)
	require.Equal(t, uint64(49_000), before.DimensionalScores.GovernanceParticipation)
	require.Equal(t, alignmenttypes.CategoryCritical, before.Category)
	require.Equal(t, "200000", f.knowledge.GetEffectiveMinReviewFee(f.ctx))
	require.Equal(t, uint64(100), f.knowledge.GetEffectiveCooldown(f.ctx, "physics"))

	for i := 0; i < 93; i++ {
		f.propose(t, fmt.Sprintf("unendorsed_%02d", i))
	}
	require.Len(t, f.ontology.GetAllDomains(f.ctx), 100)
	assert.Equal(t, uint64(7), f.count())
	// Use the real scheduled observation/state writer, not a supplied category.
	f.ctx = f.ctx.WithBlockHeight(200)
	require.NoError(t, f.module.EndBlock(f.ctx))
	after, found := f.alignment.GetHealthIndex(f.ctx, 200)
	require.True(t, found)
	fee := f.knowledge.GetEffectiveMinReviewFee(f.ctx)
	cooldown := f.knowledge.GetEffectiveCooldown(f.ctx, "physics")
	t.Logf("governance %d -> %d; composite %d -> %d; health %s -> %s; fee 200000 -> %s; cooldown 100 -> %d",
		before.DimensionalScores.GovernanceParticipation, after.DimensionalScores.GovernanceParticipation,
		before.CompositeScore, after.CompositeScore, before.Category, after.Category, fee, cooldown)
	assert.Equal(t, before.DimensionalScores.GovernanceParticipation, after.DimensionalScores.GovernanceParticipation)
	assert.Equal(t, before.CompositeScore, after.CompositeScore)
	assert.Equal(t, before.Category, after.Category)
	assert.Equal(t, "200000", fee)
	assert.Equal(t, uint64(100), cooldown)
	retained, found := f.alignment.GetHealthIndex(f.ctx, 100)
	require.True(t, found)
	require.True(t, proto.Equal(before, retained), "historical observations are not rewritten")
}

func TestAlignmentDomainCountRetainsLegacyHistoryUntilNextObservation(t *testing.T) {
	f := newDomainCountFixture(t, types.DefaultDomains())
	for i := 0; i < 93; i++ {
		f.propose(t, fmt.Sprintf("retained_%02d", i))
	}
	// Explicit legacy fixture: the pre-fix regression produces 700000 from
	// these 100 records. Retain its observation/category as upgrade prestate;
	// this is not an upgrade handler or a replay of the old binary.
	legacyObservation := f.alignment.ObserveAll(f.ctx)
	legacyObservation.GovernanceParticipation = 700_000
	f.alignment.SetObservation(f.ctx, legacyObservation)
	legacyScores := f.alignment.ComputeScores(f.ctx, legacyObservation)
	f.alignment.SetScores(f.ctx, legacyScores)
	category := f.alignment.CategorizeHealth(f.ctx, legacyScores.Composite)
	require.Equal(t, alignmenttypes.CategoryDegraded, category)
	legacyHealth := f.alignment.BuildHealthIndex(legacyScores, category, 0)
	f.alignment.SetHealthIndex(f.ctx, legacyHealth)
	legacyState := &alignmenttypes.AlignmentState{
		Enabled: true, PreviousCategory: category, LastObservationHeight: 100, ObservationCount: 1,
	}
	f.alignment.SetState(f.ctx, legacyState)
	ontologyBefore := f.rawOntology(t)

	// Merely running the corrected binary does not recompute stored pacing
	// before the existing scheduler runs an observation.
	f.ctx = f.ctx.WithBlockHeight(101)
	require.NoError(t, f.module.EndBlock(f.ctx))
	require.True(t, proto.Equal(legacyState, f.alignment.GetState(f.ctx)))
	_, found := f.alignment.GetObservation(f.ctx, 101)
	require.False(t, found)
	require.Equal(t, "133333", f.knowledge.GetEffectiveMinReviewFee(f.ctx))
	require.Equal(t, uint64(66), f.knowledge.GetEffectiveCooldown(f.ctx, "physics"))

	f.ctx = f.ctx.WithBlockHeight(200)
	require.NoError(t, f.module.EndBlock(f.ctx))
	current, found := f.alignment.GetHealthIndex(f.ctx, 200)
	require.True(t, found)
	require.Equal(t, uint64(49_000), current.DimensionalScores.GovernanceParticipation)
	require.Equal(t, alignmenttypes.CategoryCritical, current.Category)
	require.Equal(t, "200000", f.knowledge.GetEffectiveMinReviewFee(f.ctx))
	require.Equal(t, uint64(100), f.knowledge.GetEffectiveCooldown(f.ctx, "physics"))

	observation, found := f.alignment.GetObservation(f.ctx, 100)
	require.True(t, found)
	require.True(t, proto.Equal(legacyObservation, observation))
	scores, found := f.alignment.GetScores(f.ctx, 100)
	require.True(t, found)
	require.True(t, proto.Equal(legacyScores, scores))
	health, found := f.alignment.GetHealthIndex(f.ctx, 100)
	require.True(t, found)
	require.True(t, proto.Equal(legacyHealth, health))
	require.Equal(t, ontologyBefore, f.rawOntology(t), "legacy proposals remain intact")
}

// The app currently leaves capture-defense flags unwired. Exercise the
// conditional consumer without changing that production activation boundary.
func TestAlignmentCapturePenaltyUsesActiveDomains(t *testing.T) {
	for _, tc := range []struct {
		name   string
		active int
		flags  domainCountFlags
		want   uint64
	}{
		{name: "one of four", active: 4, flags: 1, want: 187_500},
		{name: "no flags", active: 4, want: 250_000},
		{name: "excess flags retain cap", active: 4, flags: 9},
		{name: "zero active retains denominator guard", flags: 1, want: 250_000},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var domains []*types.Domain
			for i := 0; i < tc.active; i++ {
				domains = append(domains, &types.Domain{Name: fmt.Sprintf("active_%d", i), Status: "active", Stratum: uint32(types.StratumEmpirical)})
			}
			f := newDomainCountFixture(t, domains)
			f.alignment.SetCaptureDefenseKeeper(tc.flags)
			require.Equal(t, tc.want, f.alignment.ObserveAll(f.ctx).NetworkSecurity)
			for _, status := range []string{"proposed", "deprecated", "archived"} {
				f.ontology.SetDomain(f.ctx, &types.Domain{Name: status, Status: status, Stratum: uint32(types.StratumEmpirical)})
			}
			assert.Equal(t, tc.want, f.alignment.ObserveAll(f.ctx).NetworkSecurity, "inactive records must not dilute capture risk")
		})
	}
}
