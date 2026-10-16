package keeper_test

import (
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

	"github.com/zerone-chain/zerone/internal/claimintegritymigration"
	"github.com/zerone-chain/zerone/internal/fundsettlementmigration"
	"github.com/zerone-chain/zerone/x/knowledge/keeper"
	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// Claim-integrity activation (knowledge v12; plan §3.1-§3.3). The 0x87 marker
// only selects semantics: no behaviour reads it until Tasks 7-14. Module
// registration and the ConsensusVersion bump are Task 6.

const claimIntegrityReceipt = "migration_v12_complete"

var claimIntegrityPredecessors = []string{
	keeper.RecordIntegrityEnabledStoreKey,
	keeper.ReviewNeutralityEnabledStoreKey,
	keeper.ClaimRecordsEnabledStoreKey,
	keeper.FundSettlementEnabledStoreKey,
}

func claimIntegrityReceiptKey() string {
	return string(append([]byte{0x7f, 0x01}, claimIntegrityReceipt...))
}

// newKnowledgeStoreFixture is an uninitialised keeper on a real IAVL store.
func newKnowledgeStoreFixture(t *testing.T, chainID string) *v11Fixture {
	t.Helper()
	db := dbm.NewMemDB()
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	ms := store.NewCommitMultiStore(db, log.NewNopLogger(), storemetrics.NewNoOpMetrics())
	ms.SetIAVLSyncPruning(true)
	key := storetypes.NewKVStoreKey(types.StoreKey)
	ms.MountStoreWithDB(key, storetypes.StoreTypeIAVL, db)
	require.NoError(t, ms.LoadLatestVersion())
	ctx := sdk.NewContext(ms, cmtproto.Header{ChainID: chainID, Height: 100}, false, log.NewNopLogger())
	bank := newTrackingBankKeeper()
	k := keeper.NewKeeper(runtime.NewKVStoreService(key), codec.NewProtoCodec(codectypes.NewInterfaceRegistry()), "authority", bank, newTrackingStakingKeeper())
	return &v11Fixture{k: k, ctx: ctx.WithEventManager(sdk.NewEventManager()), bank: bank, key: key, server: keeper.NewMsgServerImpl(k)}
}

func newClaimIntegrityGenesisFixture(t *testing.T, chainID string, gs *types.GenesisState) *v11Fixture {
	t.Helper()
	f := newKnowledgeStoreFixture(t, chainID)
	require.NoError(t, f.k.InitGenesis(f.ctx, gs))
	return f
}

// setupClaimIntegrityTest is DefaultGenesis (all five selections, 0x87
// included) plus bootstrap funding. Its marker-off twin is
// setupV11SemanticsTest.
func setupClaimIntegrityTest(t *testing.T) *v11Fixture {
	t.Helper()
	gs := types.DefaultGenesis()
	gs.BootstrapFundAllocation = "22222000000"
	return newClaimIntegrityGenesisFixture(t, "claim-integrity-test", gs)
}

func (f *v11Fixture) claimIntegrityEnabled(t *testing.T) bool {
	t.Helper()
	enabled, err := f.k.ClaimIntegrityEnabled(f.ctx)
	require.NoError(t, err)
	return enabled
}

func (f *v11Fixture) claimIntegrityReceiptPresent(t *testing.T) bool {
	t.Helper()
	_, marked, err := f.k.ReadMigrationMarkerPresenceChecked(f.ctx, claimIntegrityReceipt)
	require.NoError(t, err)
	return marked
}

func TestClaimIntegrityMarkerReadsAreExact(t *testing.T) {
	require.Equal(t, "\x87", keeper.ClaimIntegrityEnabledStoreKey)
	f := setupV11SemanticsTest(t)
	kv := f.ctx.KVStore(f.key)
	key := []byte(keeper.ClaimIntegrityEnabledStoreKey)
	require.Nil(t, kv.Get(key), "the v11 fixture has no marker")
	require.False(t, f.claimIntegrityEnabled(t), "absent reads as off")

	kv.Set(key, []byte{1})
	require.True(t, f.claimIntegrityEnabled(t))

	for _, bad := range [][]byte{{0}, {1, 1}, {2}} {
		t.Run(fmt.Sprintf("%x", bad), func(t *testing.T) {
			kv.Set(key, bad)
			enabled, err := f.k.ClaimIntegrityEnabled(f.ctx)
			require.ErrorContains(t, err, "malformed claim integrity marker")
			require.False(t, enabled, "a malformed marker is never read as on")
			require.Error(t, f.k.EnableClaimIntegrity(f.ctx), "a malformed marker is never repaired")
			require.Error(t, f.k.ValidateClaimIntegrityActivation(f.ctx))
			require.Equal(t, bad, kv.Get(key))
		})
	}
}

func TestClaimIntegrityEnableRequiresEveryPredecessor(t *testing.T) {
	for _, predecessor := range claimIntegrityPredecessors {
		for _, state := range []string{"absent", "malformed"} {
			t.Run(fmt.Sprintf("%x/%s", predecessor, state), func(t *testing.T) {
				f := setupV11SemanticsTest(t)
				kv := f.ctx.KVStore(f.key)
				if state == "absent" {
					kv.Delete([]byte(predecessor))
				} else {
					kv.Set([]byte(predecessor), []byte{2})
				}
				before := relationContractSnapshot(t, f.ctx, f.key)
				require.ErrorContains(t, f.k.EnableClaimIntegrity(f.ctx), "claim integrity requires")
				require.Error(t, f.k.ValidateClaimIntegrityActivation(f.ctx))
				require.Equal(t, before, relationContractSnapshot(t, f.ctx, f.key), "a refusal writes nothing")
			})
		}
	}

	f := setupV11SemanticsTest(t)
	require.NoError(t, f.k.ValidateClaimIntegrityActivation(f.ctx))
	require.NoError(t, f.k.EnableClaimIntegrity(f.ctx))
	require.True(t, f.claimIntegrityEnabled(t))
	require.Equal(t, []byte{1}, f.ctx.KVStore(f.key).Get([]byte(keeper.ClaimIntegrityEnabledStoreKey)))
	before := relationContractSnapshot(t, f.ctx, f.key)
	require.NoError(t, f.k.EnableClaimIntegrity(f.ctx), "enabling an enabled marker is a no-op")
	require.Equal(t, before, relationContractSnapshot(t, f.ctx, f.key))
	require.ErrorContains(t, f.k.ValidateClaimIntegrityActivation(f.ctx), "unenabled predecessor")
	require.False(t, f.claimIntegrityReceiptPresent(t), "enabling is not an upgrade receipt")
}

func TestClaimIntegrityGenesisRequiresFundSettlement(t *testing.T) {
	gs := types.DefaultGenesis()
	require.True(t, gs.ClaimIntegrityEnabled, "native genesis selects claim integrity")
	require.NoError(t, gs.Validate())

	gs.FundSettlementEnabled = false
	require.ErrorContains(t, gs.Validate(), "claim integrity requires fund settlement")
	require.ErrorContains(t, types.ValidateGenesisRounds(gs), "claim integrity requires fund settlement")
	f := newKnowledgeStoreFixture(t, "claim-integrity-genesis-refusal")
	require.ErrorContains(t, f.k.InitGenesis(f.ctx, gs), "claim integrity requires fund settlement")
	require.Empty(t, relationContractSnapshot(t, f.ctx, f.key), "refused before any genesis write")

	gs.ClaimIntegrityEnabled = false
	require.NoError(t, gs.Validate(), "the predecessor selection stays valid without claim integrity")
}

func TestClaimIntegrityNativeGenesisWritesMarker(t *testing.T) {
	f := setupClaimIntegrityTest(t)
	require.Equal(t, []byte{1}, f.ctx.KVStore(f.key).Get([]byte(keeper.ClaimIntegrityEnabledStoreKey)))
	require.True(t, f.claimIntegrityEnabled(t))
	require.False(t, f.claimIntegrityReceiptPresent(t), "native genesis invents no upgrade receipt")

	off := setupV11SemanticsTest(t)
	require.Nil(t, off.ctx.KVStore(off.key).Get([]byte(keeper.ClaimIntegrityEnabledStoreKey)))
	require.False(t, off.claimIntegrityEnabled(t))
}

func TestClaimIntegrityGenesisRoundTripsFlag(t *testing.T) {
	for _, on := range []bool{true, false} {
		t.Run(fmt.Sprintf("enabled=%t", on), func(t *testing.T) {
			gs := types.DefaultGenesis()
			gs.ClaimIntegrityEnabled = on
			source := newClaimIntegrityGenesisFixture(t, "claim-integrity-export", gs)
			require.Equal(t, on, source.claimIntegrityEnabled(t))

			exported := source.k.ExportGenesis(source.ctx)
			require.Equal(t, on, exported.ClaimIntegrityEnabled)
			require.True(t, exported.FundSettlementEnabled)
			require.NoError(t, exported.Validate())

			imported := newClaimIntegrityGenesisFixture(t, "claim-integrity-import", exported)
			require.Equal(t, on, imported.claimIntegrityEnabled(t))
			require.Equal(t, on, imported.k.ExportGenesis(imported.ctx).ClaimIntegrityEnabled)
			require.False(t, imported.claimIntegrityReceiptPresent(t), "import invents no upgrade receipt")
		})
	}

	f := setupClaimIntegrityTest(t)
	f.ctx.KVStore(f.key).Set([]byte(keeper.ClaimIntegrityEnabledStoreKey), []byte{2})
	require.Panics(t, func() { f.k.ExportGenesis(f.ctx) }, "a malformed marker is never exported as off")
}

func claimIntegrityOwnerContext(t *testing.T, ctx sdk.Context) sdk.Context {
	t.Helper()
	owned, err := claimintegritymigration.WithOwner(ctx, claimintegritymigration.UpgradeName)
	require.NoError(t, err)
	return owned
}

func TestMigrate11to12RefusesWithoutOwnerOrUnmarkedSource(t *testing.T) {
	require.Equal(t, "knowledge-claim-integrity-v1", claimintegritymigration.UpgradeName)
	_, err := claimintegritymigration.WithOwner(sdk.Context{}, fundsettlementmigration.UpgradeName)
	require.Error(t, err, "no other upgrade name can own 11->12")

	for _, scenario := range []string{"no owner", "predecessor owner", "marker set", "malformed marker", "receipt present", "missing fund settlement"} {
		t.Run(scenario, func(t *testing.T) {
			f := setupV11SemanticsTest(t)
			kv := f.ctx.KVStore(f.key)
			ctx := claimIntegrityOwnerContext(t, f.ctx)
			switch scenario {
			case "no owner":
				ctx = f.ctx
			case "predecessor owner":
				owned, err := fundsettlementmigration.WithOwner(f.ctx, fundsettlementmigration.UpgradeName)
				require.NoError(t, err)
				ctx = owned
			case "marker set":
				require.NoError(t, f.k.EnableClaimIntegrity(f.ctx))
			case "malformed marker":
				kv.Set([]byte(keeper.ClaimIntegrityEnabledStoreKey), []byte{2})
			case "receipt present":
				require.NoError(t, f.k.WriteMigrationMarker(f.ctx, claimIntegrityReceipt, "true"))
			case "missing fund settlement":
				kv.Delete([]byte(keeper.FundSettlementEnabledStoreKey))
			}
			before := relationContractSnapshot(t, f.ctx, f.key)
			err := keeper.NewMigrator(f.k).Migrate11to12(ctx)
			require.Error(t, err)
			if scenario == "no owner" || scenario == "predecessor owner" {
				require.ErrorContains(t, err, claimintegritymigration.UpgradeName)
			}
			require.Equal(t, before, relationContractSnapshot(t, f.ctx, f.key), "a refused migration writes nothing")
		})
	}
}

func TestMigrate11to12ChangesExactlyMarkerAndReceipt(t *testing.T) {
	f := setupV11SemanticsTest(t)
	// A live record makes the full-store diff meaningful: it must survive byte for byte.
	f.mustSubmitClaim(t, v11Addr("v12-migrate-author"), "Synthetic v12 activation probe: this pending claim is never rewritten.", "physics", nil)
	before := relationContractSnapshot(t, f.ctx, f.key)
	require.NotContains(t, before, keeper.ClaimIntegrityEnabledStoreKey)
	require.NotContains(t, before, claimIntegrityReceiptKey())

	ctx := claimIntegrityOwnerContext(t, f.ctx)
	require.NoError(t, keeper.NewMigrator(f.k).Migrate11to12(ctx))

	after := relationContractSnapshot(t, f.ctx, f.key)
	require.Len(t, after, len(before)+2)
	require.Equal(t, []byte{1}, after[keeper.ClaimIntegrityEnabledStoreKey])
	require.Equal(t, []byte("true"), after[claimIntegrityReceiptKey()])
	delete(after, keeper.ClaimIntegrityEnabledStoreKey)
	delete(after, claimIntegrityReceiptKey())
	require.Equal(t, before, after, "only the marker and its receipt change")
	require.True(t, f.claimIntegrityEnabled(t))

	again := relationContractSnapshot(t, f.ctx, f.key)
	require.Error(t, keeper.NewMigrator(f.k).Migrate11to12(ctx), "the migration runs once")
	require.Equal(t, again, relationContractSnapshot(t, f.ctx, f.key))
}
