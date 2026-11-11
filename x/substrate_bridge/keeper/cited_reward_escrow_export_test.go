package keeper_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/big"
	"testing"

	dbm "github.com/cosmos/cosmos-db"

	"cosmossdk.io/log"
	sdkmath "cosmossdk.io/math"
	"cosmossdk.io/store"
	storemetrics "cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	substratebridgemodule "github.com/zerone-chain/zerone/x/substrate_bridge"
	"github.com/zerone-chain/zerone/x/substrate_bridge/keeper"
	"github.com/zerone-chain/zerone/x/substrate_bridge/types"
)

// These tests hold the module genesis round trip for the cited-reward escrow
// (marker cited_reward_escrow_enabled). With the marker on, a cited-fact link
// settles with reward zero and its reward waits in the witness-reward entries
// (0x8C, 0x8D). The export must validate and import, and after the import the
// escrow must be released, deferred or cancelled exactly as it would have been
// without the round trip. With the marker off in the genesis, a cited-fact
// escrow is refused as it was before the marker existed.

const citedWitness = "att-w1"

func bridgeJSONCodec() codec.Codec {
	registry := codectypes.NewInterfaceRegistry()
	types.RegisterInterfaces(registry)
	return codec.NewProtoCodec(registry)
}

// newEmptyBridge is an empty store and a keeper with bank and vesting stubs,
// for InitGenesis to fill.
func newEmptyBridge(t *testing.T, submitter string) *citedEscrowFixture {
	t.Helper()
	storeKey := storetypes.NewKVStoreKey(types.StoreKey)
	db := dbm.NewMemDB()
	cms := store.NewCommitMultiStore(db, log.NewNopLogger(), storemetrics.NewNoOpMetrics())
	cms.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	require.NoError(t, cms.LoadLatestVersion())

	bk := &stubBankKeeper{}
	vk := &stubVestingKeeper{}
	k := keeper.NewKeeper(bridgeJSONCodec(), storeKey, "authority-addr", nil, nil, bk, vk)
	ctx := sdk.NewContext(cms, cmtproto.Header{Height: 1}, false, log.NewNopLogger())
	return &citedEscrowFixture{k: k, ctx: ctx, key: storeKey, bank: bk, vest: vk, submitter: submitter}
}

// newSettledEscrowFixture is the cited fixture with the adapter carrying a
// witness reward and a witness-only READY attestation beside the cited one,
// after one BeginBlocker has settled both at height 1. The witness reward is
// escrowed in both marker states; the cited reward is escrowed with the marker
// on and paid with it off. The attestation counter is advanced as submissions
// would have advanced it, because an armed export must carry it.
func newSettledEscrowFixture(t *testing.T, markerOn bool) *citedEscrowFixture {
	t.Helper()
	f := newCitedEscrowFixture(t, markerOn)
	f.k.NextAttestationID(f.ctx)
	f.k.NextAttestationID(f.ctx)
	adapter, found := f.k.GetAdapter(f.ctx, citedAdapter)
	require.True(t, found)
	adapter.WitnessRewardUzrn = "500"
	require.NoError(t, f.k.WriteAdapter(f.ctx, adapter))
	require.NoError(t, f.k.WriteAttestation(f.ctx, &types.ExternalAttestation{
		AttestationId: citedWitness, AdapterId: citedAdapter, Submitter: testSubmitter("witness"),
		BondUzrn: citedBond, SubmittedAtBlock: 6, Status: types.AttestationStatus_ATTESTATION_STATUS_READY,
		Link: sourcedLink(citedAdapter, "inv-w1", 6),
	}))

	require.NoError(t, f.k.BeginBlocker(f.ctx))
	_, citedPending := f.k.GetWitnessPendingReward(f.ctx, "att-c1")
	require.Equal(t, markerOn, citedPending)
	_, witnessPending := f.k.GetWitnessPendingReward(f.ctx, citedWitness)
	require.True(t, witnessPending)
	return f
}

// moduleRoundTrip exports f's module genesis as a chain export writes it,
// validates the JSON as a node validates a genesis file, and imports it into
// an empty store as InitChain does. The imported store equals the source.
func moduleRoundTrip(t *testing.T, f *citedEscrowFixture) *citedEscrowFixture {
	t.Helper()
	cdc := bridgeJSONCodec()
	bz := substratebridgemodule.NewAppModule(cdc, f.k).ExportGenesis(f.ctx, cdc)
	require.NoError(t, substratebridgemodule.AppModuleBasic{}.ValidateGenesis(cdc, nil, bz))

	var gs types.GenesisState
	require.NoError(t, cdc.UnmarshalJSON(bz, &gs))
	imported := newEmptyBridge(t, f.submitter)
	require.NoError(t, imported.k.InitGenesis(imported.ctx, &gs))
	require.Equal(t, qualificationStoreSnapshot(t, f.ctx, f.key), qualificationStoreSnapshot(t, imported.ctx, imported.key),
		"the import restores the module store exactly")
	return imported
}

// escrowOutcome is what a sequence of blocks did to one bridge: what it
// minted and paid, the events it emitted, and the module store after it.
type escrowOutcome struct {
	minted string
	paid   map[string]string
	events sdk.Events
	store  map[string][]byte
}

func observeEscrow(t *testing.T, f *citedEscrowFixture, steps func(ctx sdk.Context)) escrowOutcome {
	t.Helper()
	mintedBefore := new(big.Int)
	if m := f.vest.minted[types.AuditBountyPoolModuleName]; m != nil {
		mintedBefore.Set(m)
	}
	paidBefore := map[string]sdkmath.Int{}
	for addr, amount := range f.bank.payments {
		paidBefore[addr] = amount
	}

	ctx := f.ctx.WithEventManager(sdk.NewEventManager())
	steps(ctx)

	out := escrowOutcome{minted: "0", paid: map[string]string{}, events: ctx.EventManager().Events(),
		store: qualificationStoreSnapshot(t, f.ctx, f.key)}
	if m := f.vest.minted[types.AuditBountyPoolModuleName]; m != nil {
		out.minted = new(big.Int).Sub(m, mintedBefore).String()
	}
	for addr, amount := range f.bank.payments {
		before, ok := paidBefore[addr]
		if !ok {
			before = sdkmath.ZeroInt()
		}
		if delta := amount.Sub(before); !delta.IsZero() {
			out.paid[addr] = delta.String()
		}
	}
	return out
}

// After export and import, in both marker states, each escrow is released,
// deferred and then released, or cancelled exactly as it is on the chain that
// never exported: the same mint, payments, events and module store.
func TestCitedReward_GenesisRoundTripSweepsAsWithout(t *testing.T) {
	citer, witness := testSubmitter("citer"), testSubmitter("witness")
	for _, state := range citedMarkerStates {
		for _, tc := range []struct {
			name  string
			steps func(t *testing.T, f *citedEscrowFixture) func(sdk.Context)
			// By marker state. With the marker off the cited reward was
			// paid at settle, before the export.
			wantMinted map[bool]string
			wantPaid   map[bool]map[string]string
		}{
			{
				name: "released at the deadline",
				steps: func(t *testing.T, f *citedEscrowFixture) func(sdk.Context) {
					return func(ctx sdk.Context) {
						require.NoError(t, f.k.BeginBlocker(ctx.WithBlockHeight(ctx.BlockHeight()+int64(citedWindow)-1)))
						require.NoError(t, f.k.BeginBlocker(ctx.WithBlockHeight(ctx.BlockHeight()+int64(citedWindow))))
					}
				},
				wantMinted: map[bool]string{true: "1332500", false: "500"},
				wantPaid:   map[bool]map[string]string{true: {citer: citedReward, witness: "500"}, false: {witness: "500"}},
			},
			{
				name: "deferred while suspended, released on reinstatement",
				steps: func(t *testing.T, f *citedEscrowFixture) func(sdk.Context) {
					return func(ctx sdk.Context) {
						require.NoError(t, f.k.SuspendAdapter(ctx, citedAdapter, "under investigation"))
						first := ctx.WithBlockHeight(ctx.BlockHeight() + int64(citedWindow))
						require.NoError(t, f.k.BeginBlocker(first))
						require.Equal(t, "0", f.attestation(t, citedWitness).RewardUzrn, "deferred, not decided")
						adapter, found := f.k.GetAdapter(ctx, citedAdapter)
						require.True(t, found)
						adapter.Status = types.AdapterStatus_ADAPTER_STATUS_ACTIVE
						require.NoError(t, f.k.WriteAdapter(ctx, adapter))
						require.NoError(t, f.k.BeginBlocker(first.WithBlockHeight(first.BlockHeight()+int64(citedWindow))))
					}
				},
				wantMinted: map[bool]string{true: "1332500", false: "500"},
				wantPaid:   map[bool]map[string]string{true: {citer: citedReward, witness: "500"}, false: {witness: "500"}},
			},
			{
				name: "cancelled by tombstoning",
				steps: func(t *testing.T, f *citedEscrowFixture) func(sdk.Context) {
					return func(ctx sdk.Context) {
						require.NoError(t, f.k.TombstoneAdapter(ctx.WithBlockHeight(ctx.BlockHeight()+1), citedAdapter))
						require.NoError(t, f.k.BeginBlocker(ctx.WithBlockHeight(ctx.BlockHeight()+int64(citedWindow))))
					}
				},
				wantMinted: map[bool]string{true: "0", false: "0"},
				wantPaid:   map[bool]map[string]string{true: {}, false: {}},
			},
		} {
			t.Run(state.name+"/"+tc.name, func(t *testing.T) {
				original := newSettledEscrowFixture(t, state.on)
				imported := moduleRoundTrip(t, original)

				without := observeEscrow(t, original, tc.steps(t, original))
				with := observeEscrow(t, imported, tc.steps(t, imported))

				require.Equal(t, tc.wantMinted[state.on], without.minted)
				require.Equal(t, tc.wantPaid[state.on], without.paid)
				require.Equal(t, without.minted, with.minted)
				require.Equal(t, without.paid, with.paid)
				require.Equal(t, without.events, with.events)
				require.Equal(t, without.store, with.store)

				for _, f := range []*citedEscrowFixture{original, imported} {
					for _, id := range []string{"att-c1", citedWitness} {
						_, pending := f.k.GetWitnessPendingReward(f.ctx, id)
						require.False(t, pending, "%s resolved", id)
					}
				}
				if tc.name == "cancelled by tombstoning" {
					cancelled := eventAttributes(with.events, keeper.EventTypeWitnessRewardCancelled)
					wantCancelled := 1
					if state.on {
						wantCancelled = 2
					}
					require.Len(t, cancelled, wantCancelled, "every escrow still pending at the tombstone is cancelled")
					wantReward := citedReward // paid at settle, before the export
					if state.on {
						wantReward = "0"
					}
					require.Equal(t, wantReward, imported.attestation(t, "att-c1").RewardUzrn)
				}
			})
		}
	}
}

// citedEscrowEntryIndex is the index of att-c1's 0x8C entry in gs.
func citedEscrowEntryIndex(t *testing.T, gs *types.GenesisState) int {
	t.Helper()
	for i, entry := range gs.StateEntries {
		if bytes.Equal(entry.Key, types.WitnessPendingRewardKey("att-c1")) {
			return i
		}
	}
	t.Fatal("no pending cited reward in the export")
	return -1
}

// With the marker off in the genesis, a pending cited-fact escrow is refused at
// validation and at import with the error it had before the marker existed,
// and the import writes nothing.
func TestCitedReward_GenesisWithMarkerOffRefusesCitedEscrow(t *testing.T) {
	f := newSettledEscrowFixture(t, true)
	exported := f.k.ExportGenesis(f.ctx)
	require.True(t, exported.CitedRewardEscrowEnabled)
	exported.CitedRewardEscrowEnabled = false
	want := fmt.Sprintf("state entry at index %d witness reward belongs to a non-witness attestation",
		citedEscrowEntryIndex(t, exported))

	require.EqualError(t, exported.Validate(), want)
	cdc := bridgeJSONCodec()
	require.EqualError(t, substratebridgemodule.AppModuleBasic{}.ValidateGenesis(cdc, nil, cdc.MustMarshalJSON(exported)), want)

	imported := newEmptyBridge(t, f.submitter)
	require.EqualError(t, imported.k.InitGenesis(imported.ctx, exported), want)
	require.Empty(t, qualificationStoreSnapshot(t, imported.ctx, imported.key), "a refused import writes nothing")
}

// With the marker on, an exported cited-fact escrow that no longer agrees with
// its attestation is still refused at import.
func TestCitedReward_GenesisWithMarkerOnRefusesMalformedCitedEscrow(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(t *testing.T, gs *types.GenesisState)
		want   string
	}{
		{"recipient is not the submitter", func(t *testing.T, gs *types.GenesisState) {
			entry := gs.StateEntries[citedEscrowEntryIndex(t, gs)]
			var pr keeper.WitnessPendingReward
			require.NoError(t, json.Unmarshal(entry.Value, &pr))
			pr.Recipient = testSubmitter("someone-else")
			bz, err := json.Marshal(pr)
			require.NoError(t, err)
			entry.Value = bz
		}, "witness recipient does not match its attestation"},
		{"attestation has pending claims", func(t *testing.T, gs *types.GenesisState) {
			for _, entry := range gs.StateEntries {
				if !bytes.Equal(entry.Key, types.AttestationKey("att-c1")) {
					continue
				}
				var att types.ExternalAttestation
				require.NoError(t, proto.Unmarshal(entry.Value, &att))
				att.Link.PendingClaims = []*types.PendingClaim{{ClaimContent: "pending", Domain: "d"}}
				bz, err := proto.Marshal(&att)
				require.NoError(t, err)
				entry.Value = bz
			}
		}, "witness reward belongs to a non-witness attestation"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newSettledEscrowFixture(t, true)
			exported := f.k.ExportGenesis(f.ctx)
			require.NoError(t, exported.Validate())
			tc.mutate(t, exported)
			want := fmt.Sprintf("state entry at index %d %s", citedEscrowEntryIndex(t, exported), tc.want)

			imported := newEmptyBridge(t, f.submitter)
			require.EqualError(t, imported.k.InitGenesis(imported.ctx, exported), want)
			require.Empty(t, qualificationStoreSnapshot(t, imported.ctx, imported.key))
		})
	}
}
