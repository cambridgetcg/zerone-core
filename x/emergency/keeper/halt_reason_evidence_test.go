package keeper_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	corestore "cosmossdk.io/core/store"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	emergencymodule "github.com/zerone-chain/zerone/x/emergency"
	"github.com/zerone-chain/zerone/x/emergency/keeper"
	"github.com/zerone-chain/zerone/x/emergency/types"
	govtypes "github.com/zerone-chain/zerone/x/gov/types"
)

func TestHaltReasonImportedLegacyPayloadFailsClosed(t *testing.T) {
	wrongID, err := proto.Marshal(&types.EmergencyHaltProposal{Id: "different", Reason: types.HaltReasonResearchFund})
	require.NoError(t, err)
	emptyReason, err := proto.Marshal(&types.EmergencyHaltProposal{Id: "legacy"})
	require.NoError(t, err)
	for _, tc := range []struct {
		name    string
		payload []byte
	}{
		{"malformed", []byte{0xff}}, {"wrong_id", wrongID}, {"empty_reason", emptyReason},
	} {
		t.Run(tc.name, func(t *testing.T) {
			k, g, ctx, _ := setupHaltReasonKeepers(t)
			gs := types.DefaultGenesis()
			gs.HaltReasonScopingEnabled = false
			gs.Ceremonies = []*types.EmergencyCeremony{{Id: "legacy", Type: "halt", Phase: "finalized", StartBlock: 45_000_000, ProposalData: tc.payload}}
			require.NoError(t, gs.Validate(), "pre-snapshot imports bypass payload validation")
			raw, err := json.Marshal(gs)
			require.NoError(t, err)
			cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
			am := emergencymodule.NewAppModule(cdc, k)
			am.InitGenesis(ctx, cdc, raw)
			stored, found := k.GetCeremony(ctx, "legacy")
			require.True(t, found)
			require.Equal(t, tc.payload, stored.ProposalData, "import preserves retained terminal evidence")
			require.NoError(t, k.EnableHaltReasonScoping(ctx))
			selected := g.WithHaltReasonScoping(true)
			_, met := selected.CheckPhaseExitConditions(ctx)
			require.False(t, met)
			require.ErrorIs(t, selected.ValidatePhaseRollback(ctx), types.ErrHaltEvidence)
			var exported types.GenesisState
			require.NoError(t, json.Unmarshal(am.ExportGenesis(ctx, cdc), &exported))
			require.True(t, exported.HaltReasonScopingEnabled)
			require.Equal(t, tc.payload, exported.Ceremonies[0].ProposalData)
		})
	}
}

type haltIteratorFaultService struct {
	corestore.KVStoreService
	mode string
}

func (s haltIteratorFaultService) OpenKVStore(ctx context.Context) corestore.KVStore {
	return haltIteratorFaultStore{s.KVStoreService.OpenKVStore(ctx), s.mode}
}

type haltIteratorFaultStore struct {
	corestore.KVStore
	mode string
}

func (s haltIteratorFaultStore) Iterator(start, end []byte) (corestore.Iterator, error) {
	if s.mode == "open" {
		return nil, errors.New("injected iterator open")
	}
	it, err := s.KVStore.Iterator(start, end)
	if err != nil {
		return nil, err
	}
	return haltIteratorFault{it, s.mode}, nil
}

type haltIteratorFault struct {
	corestore.Iterator
	mode string
}

func (i haltIteratorFault) Error() error {
	if i.mode == "terminal" {
		return errors.New("injected iterator traversal")
	}
	return i.Iterator.Error()
}
func (i haltIteratorFault) Close() error {
	err := i.Iterator.Close()
	if i.mode == "close" {
		return errors.New("injected iterator close")
	}
	return err
}
func TestHaltReasonIteratorFaultDoesNotEraseJustification(t *testing.T) {
	for _, mode := range []string{"open", "terminal", "close"} {
		for _, gridlock := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/gridlock=%t", mode, gridlock), func(t *testing.T) {
				k, g, ctx, key := setupHaltReasonKeepers(t)
				storeHaltReasonCeremony(t, k, ctx, "valid", types.HaltReasonResearchFund, "finalized", 45_000_000)
				if gridlock {
					for i := 0; i < govtypes.RollbackGridlockThreshold; i++ {
						g.SetResearchSpendProposal(ctx, &govtypes.ResearchSpendProposal{ProposalId: uint64(i + 1), Stage: string(govtypes.ResearchStageExpired)})
					}
				}
				k = keeper.NewKeeper(haltIteratorFaultService{runtime.NewKVStoreService(key), mode}, nil, "authority", nil)
				require.NoError(t, k.EnableHaltReasonScoping(ctx))
				a := keeper.NewGovEmergencyAdapter(k)
				g.SetEmergencyKeeper(a)
				count, unknown := a.CountResearchFundHalts(ctx, 40_000_000)
				require.True(t, unknown)
				if mode == "open" {
					require.Zero(t, count, "an unread halt is not positive evidence")
				} else {
					require.Equal(t, uint64(1), count, "preserve already verified evidence")
				}
				selected := g.WithHaltReasonScoping(true)
				_, met := selected.CheckPhaseExitConditions(ctx)
				require.False(t, met)
				if gridlock || mode != "open" {
					require.NoError(t, selected.ValidatePhaseRollback(ctx))
				} else {
					require.ErrorIs(t, selected.ValidatePhaseRollback(ctx), types.ErrHaltEvidence)
				}
			})
		}
	}
}

func TestHaltReasonIteratorFailuresAreEvidenceErrors(t *testing.T) {
	for _, mode := range []string{"open", "terminal", "close"} {
		t.Run(mode, func(t *testing.T) {
			_, g, ctx, key := setupHaltReasonKeepers(t)
			k := keeper.NewKeeper(haltIteratorFaultService{runtime.NewKVStoreService(key), mode}, nil, "authority", nil)
			require.NoError(t, k.EnableHaltReasonScoping(ctx))
			g.SetEmergencyKeeper(keeper.NewGovEmergencyAdapter(k))
			selected := g.WithHaltReasonScoping(true)
			_, met := selected.CheckPhaseExitConditions(ctx)
			require.False(t, met)
			require.ErrorIs(t, selected.ValidatePhaseRollback(ctx), types.ErrHaltEvidence)
		})
	}
}
