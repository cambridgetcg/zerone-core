package keeper

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"math/big"
	"os"
	"testing"

	corestore "cosmossdk.io/core/store"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/zerone-chain/zerone/internal/activation"
	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// Pin the exact function bodies at the v2 foundation, not just agreement
// between two potentially edited implementations of the legacy behavior.
func TestGenesisAccountingLegacyBodiesPinned(t *testing.T) {
	source, err := os.ReadFile("genesis_legacy.go")
	require.NoError(t, err)
	fset := token.NewFileSet()
	parsed, err := parser.ParseFile(fset, "genesis_legacy.go", source, 0)
	require.NoError(t, err)
	want := map[string]string{
		"initGenesisLegacy":   "61acabd44db545cb77b2680c866d23f06a5b09ecadcc4beebba88d6cf88bdad4",
		"exportGenesisLegacy": "2c7a13d519092a26b51429649c0f62e88f177830878e829c17daed746716c2da",
	}
	for _, decl := range parsed.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || want[fn.Name.Name] == "" {
			continue
		}
		start, end := fset.Position(fn.Body.Pos()).Offset, fset.Position(fn.Body.End()).Offset
		require.Equal(t, want[fn.Name.Name], fmt.Sprintf("%x", sha256.Sum256(source[start:end])))
		delete(want, fn.Name.Name)
	}
	require.Empty(t, want)
}

// Read-only bank double: any attempt to mint or transfer panics. Real SDK bank
// transfers and genesis backing are exercised in genesis_accounting_test.go.
type accountingZeroBank struct{ types.BankKeeper }

func (accountingZeroBank) GetBalance(_ context.Context, _ sdk.AccAddress, denom string) sdk.Coin {
	return sdk.NewInt64Coin(denom, 0)
}

func accountingEmptyGenesis() *types.GenesisState {
	gs := types.DefaultGenesis()
	gs.TrainingFundAllocation = "0"
	gs.AccountingState = &types.KnowledgeGenesisAccounting{TrainingFundNonEscrowBalance: "0"}
	return gs
}

func TestGenesisAccountingLegacyImportParity(t *testing.T) {
	for _, mode := range []string{"default", "existing-primary-keys", "bad-allocation", "bad-relation"} {
		t.Run(mode, func(t *testing.T) {
			legacy, current := setupRelationGenesis(t, nil), setupRelationGenesis(t, nil)
			gs := types.DefaultGenesis()
			gs.InconclusiveRoundAccountingEnabled = false
			gs.CalibrationSmoothingEnabled = false
			gs.ZeroConfidenceFloorEnabled = false
			require.Nil(t, gs.AccountingState)
			switch mode {
			case "existing-primary-keys":
				for _, f := range []handoffFixture{legacy, current} {
					store := f.ctx.KVStore(f.keys[0])
					store.Set(types.TrainingFundEscrowLockedKey("retained"), []byte("123"))
					store.Set(append(bytes.Clone(types.BootstrapClaimCountPrefix), []byte("Exact")...), binary.BigEndian.AppendUint64(nil, 10))
				}
			case "bad-allocation":
				gs.TrainingFundAllocation = "-1"
			case "bad-relation":
				gs.FactRelationState = &types.FactRelationGenesis{Relations: []*types.FactRelation{{SourceFactId: "missing", TargetFactId: "absent"}}}
			}
			legacy.ctx = legacy.ctx.WithGasMeter(storetypes.NewInfiniteGasMeter())
			current.ctx = current.ctx.WithGasMeter(storetypes.NewInfiniteGasMeter())
			oldErr := legacy.k.initGenesisLegacy(legacy.ctx, gs)
			newErr := current.k.InitGenesis(current.ctx, gs)
			require.Equal(t, fmt.Sprint(oldErr), fmt.Sprint(newErr))
			require.Equal(t, legacy.ctx.GasMeter().GasConsumed(), current.ctx.GasMeter().GasConsumed())
			require.Equal(t, legacy.ctx.EventManager().Events(), current.ctx.EventManager().Events())
			require.Equal(t, legacy.snapshot(t), current.snapshot(t))
		})
	}
}

func TestGenesisAccountingAdmissionLimitsSurviveImport(t *testing.T) {
	source := setupRelationGenesis(t, types.DefaultGenesis())
	source.k.bankKeeper = accountingZeroBank{}
	for i := 0; i < 10; i++ {
		require.NoError(t, source.k.IncrementBootstrapClaimCount(source.ctx, "ExactAddress"))
	}
	for i := 0; i < 100; i++ {
		require.NoError(t, source.k.IncrementBootstrapEpochCount(source.ctx, 0))
	}
	exported := source.k.ExportGenesis(source.ctx)
	fresh := setupRelationGenesis(t, nil)
	fresh.k.bankKeeper = accountingZeroBank{}
	require.NoError(t, fresh.k.InitGenesis(fresh.ctx, exported))
	for _, f := range []handoffFixture{source, fresh} {
		params, err := f.k.GetParams(f.ctx)
		require.NoError(t, err)
		fee := sdk.NewCoins(sdk.NewInt64Coin("uzrn", 100000))
		require.ErrorContains(t, f.k.validateAndPayFromBootstrapFund(f.ctx, "ExactAddress", big.NewInt(100000), fee, params), "all 10 bootstrap fund claims")
		require.ErrorContains(t, f.k.validateAndPayFromBootstrapFund(f.ctx, "other", big.NewInt(100000), fee, params), "epoch limit reached (100/100)")
	}
}

func TestGenesisAccountingPresentReplacesAndAbsentPreserves(t *testing.T) {
	f := setupRelationGenesis(t, types.DefaultGenesis())
	f.k.bankKeeper = accountingZeroBank{}
	store := f.ctx.KVStore(f.keys[0])
	store.Set(types.TrainingFundEscrowLockedKey("old"), []byte("0"))
	require.NoError(t, f.k.IncrementBootstrapClaimCount(f.ctx, "old"))
	require.NoError(t, f.k.IncrementBootstrapEpochCount(f.ctx, 9))
	legacy := types.DefaultGenesis()
	require.NoError(t, f.k.InitGenesis(f.ctx, legacy))
	require.True(t, store.Has(types.TrainingFundEscrowLockedKey("old")))
	require.EqualValues(t, 1, f.k.GetBootstrapClaimCount(f.ctx, "old"))
	gs := accountingEmptyGenesis()
	gs.AccountingState.BootstrapAddressCounts = []*types.BootstrapAddressCount{{Address: "Exact", Count: ^uint64(0)}, {Address: "exact", Count: 0}}
	gs.AccountingState.BootstrapEpochCounts = []*types.BootstrapEpochCount{{Epoch: 0, Count: 0}}
	require.NoError(t, f.k.InitGenesis(f.ctx, gs))
	require.False(t, store.Has(types.TrainingFundEscrowLockedKey("old")))
	require.Zero(t, f.k.GetBootstrapClaimCount(f.ctx, "old"))
	require.Zero(t, f.k.GetBootstrapEpochCount(f.ctx, 9))
	require.Equal(t, ^uint64(0), f.k.GetBootstrapClaimCount(f.ctx, "Exact"))
	require.Zero(t, f.k.GetBootstrapClaimCount(f.ctx, "exact"))
	out := f.k.ExportGenesis(f.ctx)
	require.Len(t, out.AccountingState.BootstrapAddressCounts, 2)
	require.Len(t, out.AccountingState.BootstrapEpochCounts, 1)
	require.NoError(t, f.k.InitGenesis(f.ctx, accountingEmptyGenesis()))
	require.Empty(t, f.k.ExportGenesis(f.ctx).AccountingState.BootstrapAddressCounts)
}

func TestGenesisAccountingInvalidImportHasNoWrites(t *testing.T) {
	for _, mode := range []string{"nil-bank", "duplicate", "dangling", "malformed", "sum"} {
		t.Run(mode, func(t *testing.T) {
			f := setupRelationGenesis(t, nil)
			f.k.bankKeeper = accountingZeroBank{}
			gs := accountingEmptyGenesis()
			switch mode {
			case "nil-bank":
				f.k.bankKeeper = nil
			case "duplicate":
				gs.AccountingState.BootstrapEpochCounts = []*types.BootstrapEpochCount{{}, {}}
			case "dangling":
				gs.AccountingState.AugmentationEscrows = []*types.AugmentationEscrowBalance{{BountyId: "missing", RemainingUzrn: "0"}}
			case "malformed":
				gs.AccountingState.TrainingFundNonEscrowBalance = "00"
			case "sum":
				gs.TrainingFundAllocation = "1"
			}
			before := f.snapshot(t)
			require.Error(t, f.k.InitGenesis(f.ctx, gs))
			require.Equal(t, before, f.snapshot(t))
			require.Empty(t, f.ctx.EventManager().Events())
		})
	}
}

func TestGenesisAccountingStoreWriteFailuresAreAtomic(t *testing.T) {
	for _, op := range []string{"set", "delete"} {
		for _, prefix := range genesisAccountingPrefixes {
			for _, after := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/%x/after=%t", op, prefix, after), func(t *testing.T) {
					f := setupRelationGenesis(t, types.DefaultGenesis())
					f.k.bankKeeper = accountingZeroBank{}
					gs := accountingEmptyGenesis()
					gs.AugmentationBounties = []*types.AugmentationBounty{{Id: "b"}}
					gs.AccountingState.AugmentationEscrows = []*types.AugmentationEscrowBalance{{BountyId: "b", RemainingUzrn: "0"}}
					gs.AccountingState.BootstrapAddressCounts = []*types.BootstrapAddressCount{{Address: "Exact", Count: 10}}
					gs.AccountingState.BootstrapEpochCounts = []*types.BootstrapEpochCount{{Epoch: 0, Count: 100}}
					require.NoError(t, f.k.InitGenesis(f.ctx, gs))
					gs.Params.MinVerifiers++
					f.ctx = f.ctx.WithEventManager(sdk.NewEventManager())
					before := f.snapshot(t)
					*f.kFault = handoffFault{op, prefix, after}
					require.Error(t, f.k.InitGenesis(f.ctx, gs))
					require.Equal(t, before, f.snapshot(t), "no partial genesis, including after-staging failures")
					require.Empty(t, f.ctx.EventManager().Events())
				})
			}
		}
	}
}

func TestGenesisAccountingExportRejectsCorruption(t *testing.T) {
	for _, tc := range []struct {
		name       string
		key, value []byte
	}{
		{"empty escrow ID", bytes.Clone(types.TrainingFundEscrowLockedKeyPrefix), []byte("0")},
		{"bad amount", types.TrainingFundEscrowLockedKey("b"), []byte("00")},
		{"negative amount", types.TrainingFundEscrowLockedKey("b"), []byte("-1")},
		{"unbacked", types.TrainingFundEscrowLockedKey("b"), []byte("1")},
		{"dangling", types.TrainingFundEscrowLockedKey("missing"), []byte("0")},
		{"short counter", append(bytes.Clone(types.BootstrapClaimCountPrefix), 'a'), []byte{1}},
		{"long counter", append(bytes.Clone(types.BootstrapClaimCountPrefix), 'a'), make([]byte, 9)},
		{"empty address", bytes.Clone(types.BootstrapClaimCountPrefix), make([]byte, 8)},
		{"invalid address UTF8", append(bytes.Clone(types.BootstrapClaimCountPrefix), 255), make([]byte, 8)},
		{"short epoch", append(bytes.Clone(types.BootstrapEpochCountPrefix), 0), make([]byte, 8)},
		{"long epoch", append(bytes.Clone(types.BootstrapEpochCountPrefix), make([]byte, 9)...), make([]byte, 8)},
		{"epoch bad value", binary.BigEndian.AppendUint64(bytes.Clone(types.BootstrapEpochCountPrefix), 0), []byte{0}},
		{"bad bounty wire", types.AugmentationBountyKey("b"), []byte{255}},
		{"bounty mismatch", types.AugmentationBountyKey("b"), nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := setupRelationGenesis(t, types.DefaultGenesis())
			f.k.bankKeeper = accountingZeroBank{}
			require.NoError(t, f.k.SetAugmentationBounty(f.ctx, &types.AugmentationBounty{Id: "b"}))
			value := tc.value
			if value == nil {
				value = []byte{10, 1, 'c'}
			}
			f.ctx.KVStore(f.keys[0]).Set(tc.key, value)
			require.Panics(t, func() { f.k.ExportGenesis(f.ctx) })
		})
	}
}

type accountingIteratorService struct {
	corestore.KVStoreService
	prefix []byte
	mode   string
}

func (s accountingIteratorService) OpenKVStore(ctx context.Context) corestore.KVStore {
	return accountingIteratorStore{s.KVStoreService.OpenKVStore(ctx), s.prefix, s.mode}
}

type accountingIteratorStore struct {
	corestore.KVStore
	prefix []byte
	mode   string
}

func (s accountingIteratorStore) Iterator(start, end []byte) (corestore.Iterator, error) {
	if !bytes.Equal(start, s.prefix) {
		return s.KVStore.Iterator(start, end)
	}
	if s.mode == "open" {
		return nil, errors.New("injected accounting iterator open")
	}
	it, err := s.KVStore.Iterator(start, end)
	if err != nil {
		return nil, err
	}
	return historyIteratorFault{it, s.mode}, nil
}

func TestGenesisAccountingIteratorErrorsRefuseExportAndImport(t *testing.T) {
	for _, prefix := range append(append([][]byte{}, genesisAccountingPrefixes...), types.AugmentationBountyKeyPrefix) {
		for _, mode := range []string{"open", "error", "close"} {
			t.Run(fmt.Sprintf("%x/%s", prefix, mode), func(t *testing.T) {
				f := setupRelationGenesis(t, types.DefaultGenesis())
				f.k.bankKeeper = accountingZeroBank{}
				if mode == "error" {
					// The iterator must also fail closed after yielding valid rows,
					// not just when its inventory is empty.
					gs := accountingEmptyGenesis()
					gs.AugmentationBounties = []*types.AugmentationBounty{{Id: "b"}}
					gs.AccountingState.AugmentationEscrows = []*types.AugmentationEscrowBalance{{BountyId: "b", RemainingUzrn: "0"}}
					gs.AccountingState.BootstrapAddressCounts = []*types.BootstrapAddressCount{{Address: "retained", Count: 1}}
					gs.AccountingState.BootstrapEpochCounts = []*types.BootstrapEpochCount{{Epoch: 9, Count: 1}}
					require.NoError(t, f.k.InitGenesis(f.ctx, gs))
					f.ctx = f.ctx.WithEventManager(sdk.NewEventManager())
				}
				f.k.storeService = accountingIteratorService{f.k.storeService, prefix, mode}
				require.Panics(t, func() { f.k.ExportGenesis(f.ctx) })
				if bytes.Equal(prefix, types.AugmentationBountyKeyPrefix) {
					return
				}
				before := f.snapshot(t)
				require.Error(t, f.k.InitGenesis(f.ctx, accountingEmptyGenesis()))
				require.Equal(t, before, f.snapshot(t))
				require.Empty(t, f.ctx.EventManager().Events())
			})
		}
	}
}

func TestGenesisAccountingBanklessAndMarkerFaultBoundaries(t *testing.T) {
	f := setupRelationGenesis(t, types.DefaultGenesis())
	require.Nil(t, f.k.ExportGenesis(f.ctx).AccountingState, "without bank, do not attest an unknown balance as zero")
	require.NoError(t, f.k.IncrementBootstrapClaimCount(f.ctx, "address"))
	require.Panics(t, func() { f.k.ExportGenesis(f.ctx) }, "never drop nonempty inventory without bank")
	f = setupRelationGenesis(t, types.DefaultGenesis())
	f.ctx.KVStore(f.keys[0]).Set(types.ZeroConfidenceFloorEnabledKey, []byte{2})
	var recovered any
	func() { defer func() { recovered = recover() }(); f.k.ExportGenesis(f.ctx) }()
	require.ErrorIs(t, recovered.(error), activation.ErrMarkerFault)
}
