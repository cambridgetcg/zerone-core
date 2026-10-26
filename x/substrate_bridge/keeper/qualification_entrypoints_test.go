package keeper_test

import (
	"bytes"
	"context"
	"fmt"
	"testing"

	errorsmod "cosmossdk.io/errors"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/zerone-chain/zerone/internal/activation"
	qualificationtypes "github.com/zerone-chain/zerone/x/qualification/types"
	"github.com/zerone-chain/zerone/x/substrate_bridge/keeper"
	"github.com/zerone-chain/zerone/x/substrate_bridge/types"
)

func qualificationErrorText(err error) string {
	codespace, code, message := errorsmod.ABCIInfo(err, false)
	return fmt.Sprintf("%s/%d/%s", codespace, code, message)
}

// Compare the public entry point to the frozen 77521a0d handler, not to another
// invocation of the new helper. Both fixtures use the real native adapter.
func TestQualificationSemanticsMarkerOffMatchesLegacy(t *testing.T) {
	for _, scenario := range []string{
		"open", "active-refused", "revoked-admitted", "missing", "nil-record", "unwired",
		"nil-link", "bad-hash", "missing-source", "adapter-mismatch", "work-class",
		"low-bond", "bad-address", "duplicate-source", "fresh-auto-arm", "unarmed-history",
	} {
		t.Run(scenario, func(t *testing.T) {
			prepare := func() (qualificationFixture, *types.MsgSubmitExternalAttestation) {
				f := newQualificationFixture(t)
				domain := "physics"
				if scenario == "open" {
					domain = ""
				}
				msg := f.message(t, domain, types.QualificationStatus_QUALIFICATION_STATUS_ACTIVE)
				status := qualificationtypes.QualificationStatus_QUALIFICATION_STATUS_REVOKED
				if scenario == "active-refused" {
					status = qualificationtypes.QualificationStatus_QUALIFICATION_STATUS_ACTIVE
				}
				if scenario != "missing" {
					f.native.SetQualification(f.ctx, &qualificationtypes.DomainQualification{
						Validator: msg.Submitter, Domain: "physics", Status: status,
					})
				}
				switch scenario {
				case "nil-record":
					f.useQualificationKeeper(qualificationLookup(func(context.Context, string, string) (*qualificationtypes.DomainQualification, bool) {
						return nil, true
					}))
				case "unwired":
					f.useQualificationKeeper(nil)
				case "nil-link":
					msg.Link = nil
				case "bad-hash":
					msg.Link.LinkHash = []byte("bad")
				case "missing-source":
					msg.Link.Source = nil
					msg.Link.LinkHash = keeper.ComputeLinkHash(msg.Link)
				case "adapter-mismatch":
					msg.AdapterId = "different"
				case "work-class":
					a, _ := f.bridge.GetAdapter(f.ctx, msg.AdapterId)
					a.AllowedClassIds = []string{"another-class"}
					require.NoError(t, f.bridge.WriteAdapter(f.ctx, a))
				case "low-bond":
					msg.BondUzrn = "1"
				case "bad-address":
					f.useQualificationKeeper(nil)
					msg.Submitter = "not-bech32"
				case "duplicate-source":
					f.bridge.SetSourceRef(f.ctx, msg.AdapterId, msg.Link.Source.SourceId, "previous-holder")
				case "fresh-auto-arm", "unarmed-history":
					f.ctx.KVStore(f.bridgeKey).Delete(types.DedupeArmedKey)
					if scenario == "unarmed-history" {
						require.NoError(t, f.bridge.WriteAttestation(f.ctx, &types.ExternalAttestation{
							AttestationId: "old", AdapterId: msg.AdapterId,
							Status: types.AttestationStatus_ATTESTATION_STATUS_SETTLED,
						}))
					}
				}
				f.ctx = f.ctx.WithGasMeter(storetypes.NewGasMeter(1_000_000)).WithEventManager(sdk.NewEventManager())
				return f, msg
			}
			old, oldMsg := prepare()
			current, currentMsg := prepare()
			var oldWrites, currentWrites []qualificationWrite
			old.ctx = old.ctx.WithMultiStore(qualificationWriteRecorder{old.ctx.MultiStore(), &oldWrites})
			current.ctx = current.ctx.WithMultiStore(qualificationWriteRecorder{current.ctx.MultiStore(), &currentWrites})
			oldResp, oldErr := keeper.NewLegacyQualificationMsgServerForTest(old.bridge).SubmitExternalAttestation(old.ctx, oldMsg)
			resp, err := keeper.NewMsgServerImpl(current.bridge).SubmitExternalAttestation(current.ctx, currentMsg)
			require.Equal(t, oldWrites, currentWrites, "exact writes and ordering, not only final state")
			require.Equal(t, qualificationErrorText(oldErr), qualificationErrorText(err))
			require.True(t, proto.Equal(oldResp, resp))
			require.Equal(t, old.ctx.GasMeter().GasConsumed(), current.ctx.GasMeter().GasConsumed(), "marker-off gas must equal the predecessor")
			require.Equal(t, old.ctx.EventManager().Events(), current.ctx.EventManager().Events())
			require.Equal(t, old.bank, current.bank)
			require.Equal(t, qualificationStoreSnapshot(t, old.ctx, old.bridgeKey), qualificationStoreSnapshot(t, current.ctx, current.bridgeKey))
			require.Equal(t, qualificationStoreSnapshot(t, old.ctx, old.nativeKey), qualificationStoreSnapshot(t, current.ctx, current.nativeKey))
		})
	}
}

func TestQualificationSemanticsGas(t *testing.T) {
	for _, tc := range []struct {
		name    string
		domain  string
		enabled bool
		status  qualificationtypes.QualificationStatus
		gas     uint64
	}{
		{"open/off", "", false, qualificationtypes.QualificationStatus_QUALIFICATION_STATUS_ACTIVE, 24570},
		{"open/on", "", true, qualificationtypes.QualificationStatus_QUALIFICATION_STATUS_ACTIVE, 24570},
		{"gated/off/refused", "physics", false, qualificationtypes.QualificationStatus_QUALIFICATION_STATUS_ACTIVE, 6927},
		{"gated/off/admitted", "physics", false, qualificationtypes.QualificationStatus_QUALIFICATION_STATUS_REVOKED, 25960},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newQualificationFixture(t)
			msg := f.message(t, tc.domain, types.QualificationStatus_QUALIFICATION_STATUS_ACTIVE)
			f.native.SetQualification(f.ctx, &qualificationtypes.DomainQualification{Validator: msg.Submitter, Domain: "physics", Status: tc.status})
			if tc.enabled {
				require.NoError(t, f.bridge.EnableQualificationSemantics(f.ctx))
			}
			f.ctx = f.ctx.WithGasMeter(storetypes.NewGasMeter(1_000_000))
			_, err := keeper.NewMsgServerImpl(f.bridge).SubmitExternalAttestation(f.ctx, msg)
			if tc.name == "gated/off/refused" {
				require.ErrorIs(t, err, types.ErrInsufficientQualification)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, tc.gas, f.ctx.GasMeter().GasConsumed(), "fixture-specific predecessor gas_used")
			t.Logf("gas_used=%d", f.ctx.GasMeter().GasConsumed())
		})
	}
}

func TestRegisterAdapterQualificationMinimum(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		for _, domain := range []string{"", "physics"} {
			for _, minimum := range []types.QualificationStatus{0, 1, 2, 3, -1, 4, 2147483647} {
				t.Run(fmt.Sprintf("on=%t/domain=%s/min=%d", enabled, domain, minimum), func(t *testing.T) {
					f := newQualificationFixture(t)
					if enabled {
						require.NoError(t, f.bridge.EnableQualificationSemantics(f.ctx))
					}
					msg := &types.MsgRegisterAdapter{Authority: "authority-addr", Adapter: &types.AdapterRegistration{
						AdapterId: "new-adapter", Status: types.AdapterStatus_ADAPTER_STATUS_ACTIVE,
						RequiredQualificationDomain: domain, MinQualificationStatus: minimum,
					}}
					before := qualificationStoreSnapshot(t, f.ctx, f.bridgeKey)
					resp, err := keeper.NewMsgServerImpl(f.bridge).RegisterAdapter(f.ctx, msg)
					if enabled && (minimum < 0 || minimum > 2) {
						require.ErrorIs(t, err, types.ErrInvalidQualificationMinimum)
						require.Nil(t, resp)
						require.Equal(t, before, qualificationStoreSnapshot(t, f.ctx, f.bridgeKey))
						require.Empty(t, f.ctx.EventManager().Events())
						require.Zero(t, msg.Adapter.RegisteredAtBlock)
					} else {
						require.NoError(t, err)
						a, found := f.bridge.GetAdapter(f.ctx, msg.Adapter.AdapterId)
						require.True(t, found)
						require.Equal(t, minimum, a.MinQualificationStatus)
						require.Equal(t, uint64(100), a.RegisteredAtBlock)
						require.Len(t, f.ctx.EventManager().Events(), 1)
					}
				})
			}
		}
	}
}

func TestRegisterAdapterMarkerOffMatchesLegacy(t *testing.T) {
	for _, scenario := range []string{"valid", "unsupported-minimum", "unauthorized", "nil-adapter", "empty-id", "tombstoned", "status-change"} {
		t.Run(scenario, func(t *testing.T) {
			prepare := func() (qualificationFixture, *types.MsgRegisterAdapter) {
				f := newQualificationFixture(t)
				msg := &types.MsgRegisterAdapter{Authority: "authority-addr", Adapter: &types.AdapterRegistration{AdapterId: "new-adapter", Status: types.AdapterStatus_ADAPTER_STATUS_ACTIVE}}
				switch scenario {
				case "unsupported-minimum":
					msg.Adapter.MinQualificationStatus = -1
				case "unauthorized":
					msg.Authority = "other"
				case "nil-adapter":
					msg.Adapter = nil
				case "empty-id":
					msg.Adapter.AdapterId = ""
				case "tombstoned", "status-change":
					status := types.AdapterStatus_ADAPTER_STATUS_TOMBSTONED
					if scenario == "status-change" {
						status = types.AdapterStatus_ADAPTER_STATUS_SUSPENDED
					}
					require.NoError(t, f.bridge.WriteAdapter(f.ctx, &types.AdapterRegistration{AdapterId: "new-adapter", Status: status}))
				}
				f.ctx = f.ctx.WithGasMeter(storetypes.NewGasMeter(1_000_000))
				return f, msg
			}
			old, oldMsg := prepare()
			current, currentMsg := prepare()
			var oldWrites, currentWrites []qualificationWrite
			old.ctx = old.ctx.WithMultiStore(qualificationWriteRecorder{old.ctx.MultiStore(), &oldWrites})
			current.ctx = current.ctx.WithMultiStore(qualificationWriteRecorder{current.ctx.MultiStore(), &currentWrites})
			oldResp, oldErr := keeper.NewLegacyQualificationMsgServerForTest(old.bridge).RegisterAdapter(old.ctx, oldMsg)
			resp, err := keeper.NewMsgServerImpl(current.bridge).RegisterAdapter(current.ctx, currentMsg)
			require.Equal(t, oldWrites, currentWrites, "exact writes and ordering, not only final state")
			require.Equal(t, qualificationErrorText(oldErr), qualificationErrorText(err))
			require.True(t, proto.Equal(oldResp, resp))
			require.True(t, proto.Equal(oldMsg, currentMsg), "including legacy input stamping on error")
			require.Equal(t, old.ctx.GasMeter().GasConsumed(), current.ctx.GasMeter().GasConsumed())
			require.Equal(t, old.ctx.EventManager().Events(), current.ctx.EventManager().Events())
			require.Equal(t, old.bank, current.bank)
			require.Equal(t, qualificationStoreSnapshot(t, old.ctx, old.bridgeKey), qualificationStoreSnapshot(t, current.ctx, current.bridgeKey))
			require.Equal(t, qualificationStoreSnapshot(t, old.ctx, old.nativeKey), qualificationStoreSnapshot(t, current.ctx, current.nativeKey))
		})
	}
}

type qualificationWrite struct {
	store, operation string
	key, value       []byte
}

type qualificationWriteRecorder struct {
	storetypes.MultiStore
	writes *[]qualificationWrite
}

func (ms qualificationWriteRecorder) GetKVStore(key storetypes.StoreKey) storetypes.KVStore {
	return qualificationRecordingStore{ms.MultiStore.GetKVStore(key), key.Name(), ms.writes}
}

type qualificationRecordingStore struct {
	storetypes.KVStore
	name   string
	writes *[]qualificationWrite
}

func (s qualificationRecordingStore) Set(key, value []byte) {
	*s.writes = append(*s.writes, qualificationWrite{s.name, "set", bytes.Clone(key), bytes.Clone(value)})
	s.KVStore.Set(key, value)
}

func (s qualificationRecordingStore) Delete(key []byte) {
	*s.writes = append(*s.writes, qualificationWrite{s.name, "delete", bytes.Clone(key), nil})
	s.KVStore.Delete(key)
}

type qualificationReadCounter struct {
	storetypes.MultiStore
	key   storetypes.StoreKey
	reads *int
}

func (ms qualificationReadCounter) GetKVStore(key storetypes.StoreKey) storetypes.KVStore {
	store := ms.MultiStore.GetKVStore(key)
	if key == ms.key {
		return qualificationCountingStore{store, ms.reads}
	}
	return store
}

type qualificationCountingStore struct {
	storetypes.KVStore
	reads *int
}

func (s qualificationCountingStore) Get(key []byte) []byte {
	if bytes.Equal(key, types.QualificationSemanticsEnabledKey) {
		*s.reads++
	}
	return s.KVStore.Get(key)
}

func TestQualificationSemanticsEntryPointsReadOnceAndFaultBeforeWrites(t *testing.T) {
	for _, marker := range [][]byte{nil, {1}, {0}, {2}, {1, 1}} {
		for _, entry := range []string{"open", "gated", "register"} {
			t.Run(fmt.Sprintf("marker=%x/%s", marker, entry), func(t *testing.T) {
				f := newQualificationFixture(t)
				domain := "physics"
				if entry == "open" {
					domain = ""
				}
				msg := f.message(t, domain, types.QualificationStatus_QUALIFICATION_STATUS_PROBATIONARY)
				f.native.SetQualification(f.ctx, &qualificationtypes.DomainQualification{Validator: msg.Submitter, Domain: "physics", Status: qualificationtypes.QualificationStatus_QUALIFICATION_STATUS_ACTIVE})
				if bytes.Equal(marker, []byte{1}) {
					require.NoError(t, f.bridge.EnableQualificationSemantics(f.ctx))
				} else if marker != nil {
					f.ctx.KVStore(f.bridgeKey).Set(types.QualificationSemanticsEnabledKey, marker)
				}
				// ensureDedupeArmed would self-arm; a fault must precede even that write.
				f.ctx.KVStore(f.bridgeKey).Delete(types.DedupeArmedKey)
				before := qualificationStoreSnapshot(t, f.ctx, f.bridgeKey)
				beforeNative := qualificationStoreSnapshot(t, f.ctx, f.nativeKey)
				reads := 0
				f.ctx = f.ctx.WithMultiStore(qualificationReadCounter{f.ctx.MultiStore(), f.bridgeKey, &reads}).WithGasMeter(storetypes.NewGasMeter(1_000_000))
				server := keeper.NewMsgServerImpl(f.bridge)
				var err error
				if entry == "register" {
					_, err = server.RegisterAdapter(f.ctx, &types.MsgRegisterAdapter{Authority: "authority-addr", Adapter: &types.AdapterRegistration{AdapterId: "another"}})
				} else {
					_, err = server.SubmitExternalAttestation(f.ctx, msg)
				}
				require.Equal(t, 1, reads)
				if marker != nil && !bytes.Equal(marker, []byte{1}) {
					require.ErrorIs(t, err, activation.ErrMarkerFault)
					require.Zero(t, f.ctx.GasMeter().GasConsumed())
					require.Equal(t, before, qualificationStoreSnapshot(t, f.ctx, f.bridgeKey))
					require.Equal(t, beforeNative, qualificationStoreSnapshot(t, f.ctx, f.nativeKey))
					require.Empty(t, f.ctx.EventManager().Events())
					require.Zero(t, f.bank.deposits)
				} else {
					require.NoError(t, err)
				}
			})
		}
	}
}
