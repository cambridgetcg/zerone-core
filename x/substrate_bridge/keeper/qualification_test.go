package keeper_test

import (
	"bytes"
	"context"
	"fmt"
	"testing"

	"cosmossdk.io/log"
	"cosmossdk.io/store"
	storemetrics "cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	qualificationkeeper "github.com/zerone-chain/zerone/x/qualification/keeper"
	qualificationtypes "github.com/zerone-chain/zerone/x/qualification/types"
	"github.com/zerone-chain/zerone/x/substrate_bridge/keeper"
	"github.com/zerone-chain/zerone/x/substrate_bridge/types"
)

// The fixture uses the same qualification adapter as app.NewZeroneApp, real
// native qualification storage, and the real bridge message handler. Only bank
// transfers are tracked stubs: these tests do not establish bank conservation.
type qualificationFixture struct {
	bridge    keeper.Keeper
	native    qualificationkeeper.Keeper
	ctx       sdk.Context
	bridgeKey *storetypes.KVStoreKey
	nativeKey *storetypes.KVStoreKey
	bank      *stubBankKeeper
}

func newQualificationFixture(t *testing.T) qualificationFixture {
	t.Helper()
	bridgeKey := storetypes.NewKVStoreKey(types.StoreKey)
	nativeKey := storetypes.NewKVStoreKey(qualificationtypes.StoreKey)
	db := dbm.NewMemDB()
	cms := store.NewCommitMultiStore(db, log.NewNopLogger(), storemetrics.NewNoOpMetrics())
	cms.SetIAVLSyncPruning(true)
	cms.MountStoreWithDB(bridgeKey, storetypes.StoreTypeIAVL, db)
	cms.MountStoreWithDB(nativeKey, storetypes.StoreTypeIAVL, db)
	require.NoError(t, cms.LoadLatestVersion())
	t.Cleanup(func() { require.NoError(t, db.Close()) })

	registry := codectypes.NewInterfaceRegistry()
	types.RegisterInterfaces(registry)
	cdc := codec.NewProtoCodec(registry)
	native := qualificationkeeper.NewKeeper(runtime.NewKVStoreService(nativeKey), cdc, "authority-addr", nil)
	bank := &stubBankKeeper{}
	bridge := keeper.NewKeeper(cdc, bridgeKey, "authority-addr", nil,
		qualificationkeeper.NewSubstrateBridgeQualificationAdapter(native), bank, nil)
	ctx := sdk.NewContext(cms, cmtproto.Header{Height: 100}, false, log.NewNopLogger())
	require.NoError(t, bridge.SetParams(ctx, types.DefaultParams()))
	bridge.SetDedupeArmed(ctx)
	return qualificationFixture{bridge, native, ctx, bridgeKey, nativeKey, bank}
}

func (f qualificationFixture) message(t *testing.T, domain string, minimum types.QualificationStatus) *types.MsgSubmitExternalAttestation {
	t.Helper()
	const adapterID = "qualified-adapter-v1"
	require.NoError(t, f.bridge.WriteAdapter(f.ctx, &types.AdapterRegistration{
		AdapterId:                   adapterID,
		Status:                      types.AdapterStatus_ADAPTER_STATUS_ACTIVE,
		RequiredQualificationDomain: domain,
		MinQualificationStatus:      minimum,
	}))
	link := &types.SubstrateLink{AdapterId: adapterID, Source: validExternalSource(adapterID)}
	link.LinkHash = keeper.ComputeLinkHash(link)
	return &types.MsgSubmitExternalAttestation{
		Submitter: testSubmitter("qualification"), AdapterId: adapterID,
		WorkClassId: "translation", Link: link, BondUzrn: "2000000",
	}
}

func qualificationStoreSnapshot(t *testing.T, ctx sdk.Context, key storetypes.StoreKey) map[string][]byte {
	t.Helper()
	iter := ctx.KVStore(key).Iterator(nil, nil)
	defer iter.Close()
	entries := make(map[string][]byte)
	for ; iter.Valid(); iter.Next() {
		entries[string(iter.Key())] = bytes.Clone(iter.Value())
	}
	return entries
}

func (f qualificationFixture) assertSubmission(t *testing.T, msg *types.MsgSubmitExternalAttestation, accepted bool) {
	t.Helper()
	beforeBridge := qualificationStoreSnapshot(t, f.ctx, f.bridgeKey)
	beforeNative := qualificationStoreSnapshot(t, f.ctx, f.nativeKey)
	resp, err := keeper.NewMsgServerImpl(f.bridge).SubmitExternalAttestation(sdk.WrapSDKContext(f.ctx), msg)
	require.Equal(t, beforeNative, qualificationStoreSnapshot(t, f.ctx, f.nativeKey), "the bridge must not renumber native statuses")
	if !accepted {
		require.ErrorIs(t, err, types.ErrInsufficientQualification)
		require.Nil(t, resp)
		require.Zero(t, f.bank.deposits)
		require.Empty(t, f.ctx.EventManager().Events())
		require.Equal(t, beforeBridge, qualificationStoreSnapshot(t, f.ctx, f.bridgeKey), "qualification refusal must not reserve a source, ID or bond")
		return
	}
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Equal(t, 1, f.bank.deposits)
	att, found := f.bridge.GetAttestation(f.ctx, resp.AttestationId)
	require.True(t, found)
	require.Equal(t, types.AttestationStatus_ATTESTATION_STATUS_READY, att.Status)
	require.Equal(t, msg.Submitter, att.Submitter)
	require.Equal(t, msg.BondUzrn, att.BondUzrn)
	holder, found := f.bridge.GetSourceRef(f.ctx, msg.AdapterId, msg.Link.Source.SourceId)
	require.True(t, found)
	require.Equal(t, resp.AttestationId, holder)
	events := f.ctx.EventManager().Events()
	require.Len(t, events, 1)
	require.Equal(t, keeper.EventTypeExternalAttestationSubmitted, events[0].Type)
}

func TestSubmitExternalAttestation_QualificationStatusMatrix(t *testing.T) {
	minima := []types.QualificationStatus{
		types.QualificationStatus_QUALIFICATION_STATUS_UNSPECIFIED,
		types.QualificationStatus_QUALIFICATION_STATUS_PROBATIONARY,
		types.QualificationStatus_QUALIFICATION_STATUS_ACTIVE,
		types.QualificationStatus_QUALIFICATION_STATUS_DISTINGUISHED,
		-1, 4, 2147483647,
	}
	// Literal admission tables, independent of the implementation. The legacy
	// rows characterize the defect; they are not the desired qualification rule.
	statuses := []struct {
		status qualificationtypes.QualificationStatus
		legacy [7]bool
		fixed  [7]bool
	}{
		{qualificationtypes.QualificationStatus_QUALIFICATION_STATUS_UNSPECIFIED, [7]bool{true}, [7]bool{}},
		{qualificationtypes.QualificationStatus_QUALIFICATION_STATUS_ACTIVE, [7]bool{true, true}, [7]bool{true, true, true}},
		{qualificationtypes.QualificationStatus_QUALIFICATION_STATUS_PROBATIONARY, [7]bool{true, true, true}, [7]bool{true, true}},
		{qualificationtypes.QualificationStatus_QUALIFICATION_STATUS_SUSPENDED, [7]bool{true, true, true, true}, [7]bool{}},
		{qualificationtypes.QualificationStatus_QUALIFICATION_STATUS_REVOKED, [7]bool{true, true, true, true, false, true}, [7]bool{}},
		{qualificationtypes.QualificationStatus_QUALIFICATION_STATUS_EXPIRED, [7]bool{true, true, true, true, false, true}, [7]bool{}},
		{-1, [7]bool{true, true, true, true, true, true, true}, [7]bool{}},
		{6, [7]bool{true, true, true, true, false, true}, [7]bool{}},
		{2147483647, [7]bool{true, true, true, true, false, true, true}, [7]bool{}},
	}
	// Adding a declared enum requires an explicit row/column and policy review.
	for value := range qualificationtypes.QualificationStatus_name {
		found := false
		for _, row := range statuses {
			found = found || int32(row.status) == value
		}
		require.True(t, found, "missing native status %d", value)
	}
	for value := range types.QualificationStatus_name {
		require.Contains(t, minima, types.QualificationStatus(value))
	}
	for _, enabled := range []bool{false, true} {
		for _, row := range statuses {
			for column, minimum := range minima {
				t.Run(fmt.Sprintf("enabled=%t/native=%s/minimum=%s", enabled, row.status, minimum), func(t *testing.T) {
					f := newQualificationFixture(t)
					if enabled {
						require.NoError(t, f.bridge.EnableQualificationSemantics(f.ctx))
					}
					msg := f.message(t, "physics", minimum)
					f.native.SetQualification(f.ctx, &qualificationtypes.DomainQualification{
						Validator: msg.Submitter, Domain: "physics", Status: row.status,
					})
					want := row.legacy[column]
					if enabled {
						want = row.fixed[column]
					}
					f.assertSubmission(t, msg, want)
				})
			}
		}
	}
}

type qualificationLookup func(context.Context, string, string) (*qualificationtypes.DomainQualification, bool)

func (lookup qualificationLookup) GetDomainQualification(ctx context.Context, address, domain string) (*qualificationtypes.DomainQualification, bool) {
	return lookup(ctx, address, domain)
}

func (f *qualificationFixture) useQualificationKeeper(qk types.QualificationKeeper) {
	f.bridge = keeper.NewKeeper(codec.NewProtoCodec(codectypes.NewInterfaceRegistry()),
		f.bridgeKey, "authority-addr", nil, qk, f.bank, nil)
}

func TestSubmitExternalAttestation_QualificationAvailability(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		for _, test := range []string{"missing", "other-domain", "other-submitter", "nil-record", "not-found-record", "unwired", "open", "open-unwired"} {
			t.Run(fmt.Sprintf("enabled=%t/%s", enabled, test), func(t *testing.T) {
				f := newQualificationFixture(t)
				if enabled {
					require.NoError(t, f.bridge.EnableQualificationSemantics(f.ctx))
				}
				domain := "physics"
				open := test == "open" || test == "open-unwired"
				if open {
					domain = ""
				}
				msg := f.message(t, domain, types.QualificationStatus_QUALIFICATION_STATUS_ACTIVE)
				switch test {
				case "other-domain":
					f.native.SetQualification(f.ctx, &qualificationtypes.DomainQualification{
						Validator: msg.Submitter, Domain: "chemistry", Status: qualificationtypes.QualificationStatus_QUALIFICATION_STATUS_ACTIVE,
					})
				case "other-submitter":
					f.native.SetQualification(f.ctx, &qualificationtypes.DomainQualification{
						Validator: testSubmitter("someone-else"), Domain: domain, Status: qualificationtypes.QualificationStatus_QUALIFICATION_STATUS_ACTIVE,
					})
				case "nil-record":
					f.useQualificationKeeper(qualificationLookup(func(context.Context, string, string) (*qualificationtypes.DomainQualification, bool) {
						return nil, true
					}))
				case "not-found-record":
					f.useQualificationKeeper(qualificationLookup(func(context.Context, string, string) (*qualificationtypes.DomainQualification, bool) {
						return &qualificationtypes.DomainQualification{Status: qualificationtypes.QualificationStatus_QUALIFICATION_STATUS_ACTIVE}, false
					}))
				case "unwired", "open-unwired":
					f.useQualificationKeeper(nil)
				case "open":
					f.useQualificationKeeper(qualificationLookup(func(context.Context, string, string) (*qualificationtypes.DomainQualification, bool) {
						t.Fatal("an open adapter must not query qualifications")
						return nil, false
					}))
				}
				f.assertSubmission(t, msg, open || (!enabled && test == "unwired"))
			})
		}
	}
}

func TestQualificationSemanticsActivation(t *testing.T) {
	f := newQualificationFixture(t)
	require.Equal(t, []byte{0x90}, types.QualificationSemanticsKey)
	require.NoError(t, f.bridge.InitGenesis(f.ctx, types.DefaultGenesis()))
	enabled, err := f.bridge.QualificationSemanticsEnabled(f.ctx)
	require.NoError(t, err)
	require.False(t, enabled, "default genesis must not activate a consensus change")
	before := qualificationStoreSnapshot(t, f.ctx, f.bridgeKey)
	beforeNative := qualificationStoreSnapshot(t, f.ctx, f.nativeKey)
	require.NoError(t, f.bridge.EnableQualificationSemantics(f.ctx))
	enabled, err = f.bridge.QualificationSemanticsEnabled(f.ctx)
	require.NoError(t, err)
	require.True(t, enabled)
	before[string(types.QualificationSemanticsKey)] = []byte{1}
	require.Equal(t, before, qualificationStoreSnapshot(t, f.ctx, f.bridgeKey), "activation writes only its marker")
	require.NoError(t, f.bridge.EnableQualificationSemantics(f.ctx))
	require.Equal(t, before, qualificationStoreSnapshot(t, f.ctx, f.bridgeKey), "activation is idempotent")
	require.Equal(t, beforeNative, qualificationStoreSnapshot(t, f.ctx, f.nativeKey))
	require.Empty(t, f.ctx.EventManager().Events())
	require.Zero(t, f.bank.deposits)
}

func TestQualificationSemanticsMalformedMarkerRefuses(t *testing.T) {
	for _, marker := range [][]byte{{0}, {2}, {1, 1}} {
		t.Run(fmt.Sprintf("%x", marker), func(t *testing.T) {
			f := newQualificationFixture(t)
			msg := f.message(t, "physics", types.QualificationStatus_QUALIFICATION_STATUS_PROBATIONARY)
			f.native.SetQualification(f.ctx, &qualificationtypes.DomainQualification{
				Validator: msg.Submitter, Domain: "physics", Status: qualificationtypes.QualificationStatus_QUALIFICATION_STATUS_ACTIVE,
			})
			f.ctx.KVStore(f.bridgeKey).Set(types.QualificationSemanticsKey, marker)
			before := qualificationStoreSnapshot(t, f.ctx, f.bridgeKey)
			enabled, err := f.bridge.QualificationSemanticsEnabled(f.ctx)
			require.False(t, enabled)
			require.ErrorContains(t, err, "malformed substrate bridge qualification semantics marker")
			require.Error(t, f.bridge.EnableQualificationSemantics(f.ctx), "activation must not repair corrupt state")
			resp, err := keeper.NewMsgServerImpl(f.bridge).SubmitExternalAttestation(sdk.WrapSDKContext(f.ctx), msg)
			require.Nil(t, resp)
			require.ErrorContains(t, err, "malformed substrate bridge qualification semantics marker")
			require.Equal(t, before, qualificationStoreSnapshot(t, f.ctx, f.bridgeKey))
			require.Zero(t, f.bank.deposits)
			require.Empty(t, f.ctx.EventManager().Events())
		})
	}
}

func TestQualificationSemanticsGenesisRoundTrip(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		t.Run(fmt.Sprintf("enabled=%t", enabled), func(t *testing.T) {
			f := newQualificationFixture(t)
			if enabled {
				require.NoError(t, f.bridge.EnableQualificationSemantics(f.ctx))
			}
			msg := f.message(t, "physics", types.QualificationStatus_QUALIFICATION_STATUS_ACTIVE)
			gs := f.bridge.ExportGenesis(f.ctx)
			require.NoError(t, gs.Validate())
			encoded, err := proto.Marshal(gs)
			require.NoError(t, err)
			var decoded types.GenesisState
			require.NoError(t, proto.Unmarshal(encoded, &decoded))
			imported := newQualificationFixture(t)
			require.NoError(t, imported.bridge.InitGenesis(imported.ctx, &decoded))
			actual, err := imported.bridge.QualificationSemanticsEnabled(imported.ctx)
			require.NoError(t, err)
			require.Equal(t, enabled, actual)
			reexported, err := proto.Marshal(imported.bridge.ExportGenesis(imported.ctx))
			require.NoError(t, err)
			require.Equal(t, encoded, reexported)
			require.Equal(t, qualificationStoreSnapshot(t, f.ctx, f.bridgeKey), qualificationStoreSnapshot(t, imported.ctx, imported.bridgeKey))
			// Import must retain the selected behavior, not merely the marker bytes.
			imported.native.SetQualification(imported.ctx, &qualificationtypes.DomainQualification{
				Validator: msg.Submitter, Domain: "physics", Status: qualificationtypes.QualificationStatus_QUALIFICATION_STATUS_ACTIVE,
			})
			imported.assertSubmission(t, msg, enabled)
		})
	}
}

func TestQualificationSemanticsGenesisRejectsMalformedMarker(t *testing.T) {
	for _, test := range []struct {
		name  string
		key   []byte
		value []byte
	}{
		{"zero", []byte{0x90}, []byte{0}},
		{"other", []byte{0x90}, []byte{2}},
		{"multiple", []byte{0x90}, []byte{1, 1}},
		{"empty", []byte{0x90}, nil},
		{"key-suffix", []byte{0x90, 0x01}, []byte{1}},
	} {
		t.Run(test.name, func(t *testing.T) {
			f := newQualificationFixture(t)
			before := qualificationStoreSnapshot(t, f.ctx, f.bridgeKey)
			gs := types.DefaultGenesis()
			gs.StateEntries = []*types.GenesisStateEntry{{Key: test.key, Value: test.value}}
			require.Error(t, gs.Validate())
			require.Error(t, f.bridge.InitGenesis(f.ctx, gs))
			require.Equal(t, before, qualificationStoreSnapshot(t, f.ctx, f.bridgeKey))
		})
	}
}

func TestQualificationSemanticsActivationPreservesLegacyAttestation(t *testing.T) {
	f := newQualificationFixture(t)
	msg := f.message(t, "physics", types.QualificationStatus_QUALIFICATION_STATUS_ACTIVE)
	f.native.SetQualification(f.ctx, &qualificationtypes.DomainQualification{
		Validator: msg.Submitter, Domain: "physics", Status: qualificationtypes.QualificationStatus_QUALIFICATION_STATUS_REVOKED,
	})
	// The predecessor really admits this record; do not fabricate history with
	// WriteAttestation. Activation is prospective, not a retroactive forfeiture.
	f.assertSubmission(t, msg, true)
	id, found := f.bridge.GetSourceRef(f.ctx, msg.AdapterId, msg.Link.Source.SourceId)
	require.True(t, found)
	before := qualificationStoreSnapshot(t, f.ctx, f.bridgeKey)
	beforeNative := qualificationStoreSnapshot(t, f.ctx, f.nativeKey)
	require.NoError(t, f.bridge.EnableQualificationSemantics(f.ctx))
	before[string(types.QualificationSemanticsKey)] = []byte{1}
	require.Equal(t, before, qualificationStoreSnapshot(t, f.ctx, f.bridgeKey))
	require.Equal(t, beforeNative, qualificationStoreSnapshot(t, f.ctx, f.nativeKey))
	gs := f.bridge.ExportGenesis(f.ctx)
	require.NoError(t, gs.Validate(), "historical admission is not re-run on export/import")
	imported := newQualificationFixture(t)
	require.NoError(t, imported.bridge.InitGenesis(imported.ctx, gs))
	require.Equal(t, before, qualificationStoreSnapshot(t, imported.ctx, imported.bridgeKey))
	// A new source under the same revoked qualification is now refused before
	// escrow, while the previously admitted source and obligation remain intact.
	msg.Link.Source.SourceId = "after-activation"
	msg.Link.LinkHash = keeper.ComputeLinkHash(msg.Link)
	f.ctx = f.ctx.WithEventManager(sdk.NewEventManager())
	f.bank.deposits = 0
	f.assertSubmission(t, msg, false)
	require.NoError(t, f.bridge.BeginBlocker(f.ctx))
	att, found := f.bridge.GetAttestation(f.ctx, id)
	require.True(t, found)
	require.Equal(t, types.AttestationStatus_ATTESTATION_STATUS_SETTLED, att.Status)
	require.Equal(t, "0", att.RewardUzrn)
	require.Equal(t, "2000000", f.bank.payments[msg.Submitter].String(), "existing witness bond exit remains available")
	holder, found := f.bridge.GetSourceRef(f.ctx, msg.AdapterId, "source-1")
	require.True(t, found)
	require.Equal(t, id, holder)
	require.NoError(t, f.bridge.BeginBlocker(f.ctx))
	require.Equal(t, "2000000", f.bank.payments[msg.Submitter].String(), "settled record must not refund twice")
}
