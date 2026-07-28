package keeper_test

import (
	"context"
	"fmt"
	"testing"

	dbm "github.com/cosmos/cosmos-db"
	"github.com/stretchr/testify/require"

	"cosmossdk.io/log"
	"cosmossdk.io/store"
	storemetrics "cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	knowledgetypes "github.com/zerone-chain/zerone/x/knowledge/types"
	"github.com/zerone-chain/zerone/x/substrate_bridge/keeper"
	"github.com/zerone-chain/zerone/x/substrate_bridge/types"
)

// stubKarmaKnowledgeKeeper satisfies types.KnowledgeKeeper with an in-memory
// fact map, so tests can control which cited facts resolve at settlement.
type stubKarmaKnowledgeKeeper struct {
	facts map[string]*knowledgetypes.Fact
}

func (s *stubKarmaKnowledgeKeeper) GetFact(ctx context.Context, factID string) (*knowledgetypes.Fact, bool) {
	f, ok := s.facts[factID]
	return f, ok
}

func (s *stubKarmaKnowledgeKeeper) GetClaim(ctx context.Context, claimID string) (*knowledgetypes.Claim, bool) {
	return nil, false
}

func (s *stubKarmaKnowledgeKeeper) SetClaim(ctx context.Context, claim *knowledgetypes.Claim) error {
	return nil
}

// setupKarmaExternalKeeper mirrors setupSubstrateBridgeKeeperFull but also
// wires a stub knowledge keeper — the EXTERNAL_CITE path resolves cited
// facts through it.
func setupKarmaExternalKeeper(t *testing.T, kk *stubKarmaKnowledgeKeeper) (keeper.Keeper, sdk.Context) {
	t.Helper()

	storeKey := storetypes.NewKVStoreKey(types.StoreKey)
	db := dbm.NewMemDB()
	cms := store.NewCommitMultiStore(db, log.NewNopLogger(), storemetrics.NewNoOpMetrics())
	cms.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	if err := cms.LoadLatestVersion(); err != nil {
		t.Fatalf("load store: %v", err)
	}

	registry := codectypes.NewInterfaceRegistry()
	types.RegisterInterfaces(registry)
	cdc := codec.NewProtoCodec(registry)

	k := keeper.NewKeeper(cdc, storeKey, "authority-addr", kk, nil, &stubBankKeeper{}, nil, &stubVestingKeeper{})

	ctx := sdk.NewContext(cms, cmtproto.Header{Height: 1}, false, log.NewNopLogger())

	if err := k.SetParams(ctx, defaultSubstrateBridgeParams()); err != nil {
		t.Fatalf("set params: %v", err)
	}
	k.SetDedupeArmed(ctx)

	return k, ctx
}

func writeActiveAdapter(t *testing.T, k keeper.Keeper, ctx sdk.Context, adapterID string) {
	t.Helper()
	require.NoError(t, k.WriteAdapter(ctx, &types.AdapterRegistration{
		AdapterId: adapterID, Status: types.AdapterStatus_ADAPTER_STATUS_ACTIVE,
	}))
}

func citedFacts(n int) []*types.FactCitation {
	cf := make([]*types.FactCitation, 0, n)
	for i := 0; i < n; i++ {
		cf = append(cf, &types.FactCitation{FactId: fmt.Sprintf("fact-%d", i)})
	}
	return cf
}

// karmaEdgeEvents filters the context's emitted events down to karma edges.
func karmaEdgeEvents(ctx sdk.Context) []sdk.Event {
	var out []sdk.Event
	for _, ev := range ctx.EventManager().Events() {
		if ev.Type == types.EventTypeKarmaEdge {
			out = append(out, ev)
		}
	}
	return out
}

func attrValue(t *testing.T, ev sdk.Event, key string) string {
	t.Helper()
	for _, a := range ev.Attributes {
		if a.Key == key {
			return a.Value
		}
	}
	t.Fatalf("event %s has no attribute %q", ev.Type, key)
	return ""
}

func hasAttr(ev sdk.Event, key string) bool {
	for _, a := range ev.Attributes {
		if a.Key == key {
			return true
		}
	}
	return false
}

// ─── The cap: reject-not-truncate at admission ──────────────────────────────

func TestValidateLink_TooManyCitedFactsRejected(t *testing.T) {
	k, ctx := setupKarmaExternalKeeper(t, &stubKarmaKnowledgeKeeper{})
	writeActiveAdapter(t, k, ctx, "wiki-v1")

	link := &types.SubstrateLink{
		AdapterId:  "wiki-v1",
		CitedFacts: citedFacts(types.MaxCitedFactsPerLink + 1), // 17
	}
	require.ErrorIs(t, k.ValidateLink(ctx, link, defaultSubstrateBridgeParams()), types.ErrTooManyCitedFacts)
}

func TestValidateLink_ExactlyMaxCitedFactsAccepted(t *testing.T) {
	kk := &stubKarmaKnowledgeKeeper{facts: map[string]*knowledgetypes.Fact{}}
	for i := 0; i < types.MaxCitedFactsPerLink; i++ {
		id := fmt.Sprintf("fact-%d", i)
		kk.facts[id] = &knowledgetypes.Fact{Id: id, Submitter: testSubmitter("author")}
	}
	k, ctx := setupKarmaExternalKeeper(t, kk)
	writeActiveAdapter(t, k, ctx, "wiki-v1")

	link := &types.SubstrateLink{
		AdapterId:  "wiki-v1",
		CitedFacts: citedFacts(types.MaxCitedFactsPerLink), // 16, all resolvable
	}
	require.NoError(t, k.ValidateLink(ctx, link, defaultSubstrateBridgeParams()))
}

// The production relay's buildLink sets only `source` — no cited facts. The
// cap must not touch that path (the same live-safety line the axis-bounds
// gate holds in TestValidateLink_UnweightedLinkStillPassesUnboundedAdapter).
func TestValidateLink_SourceOnlyLinkUnaffectedByCitedFactsCap(t *testing.T) {
	k, ctx := setupKarmaExternalKeeper(t, &stubKarmaKnowledgeKeeper{})
	writeActiveAdapter(t, k, ctx, "agenttool-invocation-v1")

	sourceOnly := &types.SubstrateLink{
		AdapterId: "agenttool-invocation-v1",
		Source:    &types.ExternalSource{SourceId: "inv-1", ContentHash: []byte{0x01}},
	}
	require.NoError(t, k.ValidateLink(ctx, sourceOnly, defaultSubstrateBridgeParams()))

	empty := &types.SubstrateLink{AdapterId: "agenttool-invocation-v1"}
	require.NoError(t, k.ValidateLink(ctx, empty, defaultSubstrateBridgeParams()))
}

// ─── EXTERNAL_CITE recognition at settlement ────────────────────────────────

func TestSettleAttestation_EmitsExternalEdgePerCitedFact(t *testing.T) {
	authorA := testSubmitter("author-a")
	authorB := testSubmitter("author-b")
	attester := testSubmitter("attester")
	kk := &stubKarmaKnowledgeKeeper{facts: map[string]*knowledgetypes.Fact{
		"fact-a": {Id: "fact-a", Submitter: authorA, Domain: "history"},
		"fact-b": {Id: "fact-b", Submitter: authorB},
	}}
	k, ctx := setupKarmaExternalKeeper(t, kk)
	writeActiveAdapter(t, k, ctx, "wiki-v1")

	require.NoError(t, k.WriteAttestation(ctx, &types.ExternalAttestation{
		AttestationId: "att-cite", AdapterId: "wiki-v1", Submitter: attester,
		BondUzrn: "1000000",
		Status:   types.AttestationStatus_ATTESTATION_STATUS_READY,
		Link: &types.SubstrateLink{
			CitedFacts: []*types.FactCitation{{FactId: "fact-a"}, {FactId: "fact-b"}},
		},
	}))

	require.NoError(t, k.SettleAttestation(ctx, "att-cite"))
	settled, _ := k.GetAttestation(ctx, "att-cite")
	require.Equal(t, types.AttestationStatus_ATTESTATION_STATUS_SETTLED, settled.Status)

	edges := karmaEdgeEvents(ctx)
	require.Len(t, edges, 2, "one external edge per cited fact")

	byRef := map[string]sdk.Event{}
	for _, ev := range edges {
		byRef[attrValue(t, ev, types.AttrKarmaRefID)] = ev

		require.Equal(t, types.KarmaKindExternal, attrValue(t, ev, types.AttrKarmaKind))
		require.Equal(t, types.KarmaStateOrdinal, attrValue(t, ev, types.AttrKarmaState))
		require.Equal(t, attester, attrValue(t, ev, types.AttrKarmaCounterparty))
		// No magnitude at K-alpha: the ledger that prices edges is K-beta.
		require.False(t, hasAttr(ev, "magnitude"), "K-alpha edges carry no magnitude")
		require.False(t, hasAttr(ev, "magnitude_uzrn"), "K-alpha edges carry no magnitude")
		// Distinct author and attester: no self mark.
		require.False(t, hasAttr(ev, types.AttrKarmaSelf))
	}

	require.Equal(t, authorA, attrValue(t, byRef["fact-a"], types.AttrKarmaBeneficiary), "edge is author-addressed")
	require.Equal(t, "history", attrValue(t, byRef["fact-a"], types.AttrKarmaDomain))
	require.Equal(t, authorB, attrValue(t, byRef["fact-b"], types.AttrKarmaBeneficiary), "edge is author-addressed")
	require.Equal(t, "", attrValue(t, byRef["fact-b"], types.AttrKarmaDomain), "domain may be empty")
}

func TestSettleAttestation_SelfCiteIsMarked(t *testing.T) {
	attester := testSubmitter("self-citer")
	kk := &stubKarmaKnowledgeKeeper{facts: map[string]*knowledgetypes.Fact{
		"fact-own": {Id: "fact-own", Submitter: attester},
	}}
	k, ctx := setupKarmaExternalKeeper(t, kk)
	writeActiveAdapter(t, k, ctx, "wiki-v1")

	require.NoError(t, k.WriteAttestation(ctx, &types.ExternalAttestation{
		AttestationId: "att-self", AdapterId: "wiki-v1", Submitter: attester,
		BondUzrn: "1000000",
		Status:   types.AttestationStatus_ATTESTATION_STATUS_READY,
		Link: &types.SubstrateLink{
			CitedFacts: []*types.FactCitation{{FactId: "fact-own"}},
		},
	}))

	require.NoError(t, k.SettleAttestation(ctx, "att-self"))

	edges := karmaEdgeEvents(ctx)
	require.Len(t, edges, 1)
	require.Equal(t, attester, attrValue(t, edges[0], types.AttrKarmaBeneficiary))
	require.Equal(t, attester, attrValue(t, edges[0], types.AttrKarmaCounterparty))
	require.Equal(t, "true", attrValue(t, edges[0], types.AttrKarmaSelf))
}

// A cited fact that no longer resolves at settlement is skipped silently:
// settlement runs in the BeginBlocker drain, where an error is a consensus
// hazard and recognition is never worth one.
func TestSettleAttestation_UnresolvableCitedFactSkippedWithoutError(t *testing.T) {
	author := testSubmitter("author")
	kk := &stubKarmaKnowledgeKeeper{facts: map[string]*knowledgetypes.Fact{
		"fact-live": {Id: "fact-live", Submitter: author},
		// "fact-gone" deliberately absent.
	}}
	k, ctx := setupKarmaExternalKeeper(t, kk)
	writeActiveAdapter(t, k, ctx, "wiki-v1")

	require.NoError(t, k.WriteAttestation(ctx, &types.ExternalAttestation{
		AttestationId: "att-gone", AdapterId: "wiki-v1", Submitter: testSubmitter("attester"),
		BondUzrn: "1000000",
		Status:   types.AttestationStatus_ATTESTATION_STATUS_READY,
		Link: &types.SubstrateLink{
			CitedFacts: []*types.FactCitation{{FactId: "fact-live"}, {FactId: "fact-gone"}},
		},
	}))

	require.NoError(t, k.SettleAttestation(ctx, "att-gone"))
	settled, _ := k.GetAttestation(ctx, "att-gone")
	require.Equal(t, types.AttestationStatus_ATTESTATION_STATUS_SETTLED, settled.Status)

	edges := karmaEdgeEvents(ctx)
	require.Len(t, edges, 1, "only the resolvable fact earns an edge")
	require.Equal(t, "fact-live", attrValue(t, edges[0], types.AttrKarmaRefID))
	require.Equal(t, author, attrValue(t, edges[0], types.AttrKarmaBeneficiary))
}

// Rejection is not reliance: an attestation that settles REJECTED emits no
// external recognition, even when its link cites resolvable facts.
func TestSettleAttestation_RejectedEmitsNoExternalEdges(t *testing.T) {
	kk := &stubKarmaKnowledgeKeeper{facts: map[string]*knowledgetypes.Fact{
		"fact-a": {Id: "fact-a", Submitter: testSubmitter("author")},
	}}
	k, ctx := setupKarmaExternalKeeper(t, kk)
	writeActiveAdapter(t, k, ctx, "wiki-v1")

	// 3 of 4 pending claims rejected (75%) trips the 50% rejection threshold.
	require.NoError(t, k.WriteAttestation(ctx, &types.ExternalAttestation{
		AttestationId: "att-rej", AdapterId: "wiki-v1", Submitter: testSubmitter("attester"),
		BondUzrn: "1000000",
		Status:   types.AttestationStatus_ATTESTATION_STATUS_READY,
		Link: &types.SubstrateLink{
			CitedFacts:    []*types.FactCitation{{FactId: "fact-a"}},
			PendingClaims: []*types.PendingClaim{{}, {}, {}, {}},
		},
		VerifiedCount: 1, RejectedCount: 3,
	}))

	require.NoError(t, k.SettleAttestation(ctx, "att-rej"))
	settled, _ := k.GetAttestation(ctx, "att-rej")
	require.Equal(t, types.AttestationStatus_ATTESTATION_STATUS_REJECTED, settled.Status)
	require.Empty(t, karmaEdgeEvents(ctx))
}
