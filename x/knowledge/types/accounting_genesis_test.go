package types_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/zerone-chain/zerone/x/knowledge/types"
)

func validAccountingGenesis() *types.GenesisState {
	gs := types.DefaultGenesis()
	gs.TrainingFundAllocation = "5000"
	gs.AugmentationBounties = []*types.AugmentationBounty{{Id: "active", Active: true}, {Id: "retained"}}
	gs.AccountingState = &types.KnowledgeGenesisAccounting{
		AugmentationEscrows:          []*types.AugmentationEscrowBalance{{BountyId: "active", RemainingUzrn: "4500"}, {BountyId: "retained", RemainingUzrn: "100"}},
		BootstrapAddressCounts:       []*types.BootstrapAddressCount{{Address: "ExactAddress", Count: 10}},
		BootstrapEpochCounts:         []*types.BootstrapEpochCount{{Epoch: 0, Count: 10}, {Epoch: 9, Count: 0}},
		TrainingFundNonEscrowBalance: "400",
	}
	return gs
}

func TestGenesisAccountingValidation(t *testing.T) {
	good := validAccountingGenesis()
	require.NoError(t, good.Validate())
	for _, tc := range []struct {
		name   string
		mutate func(*types.GenesisState)
	}{
		{"missing allocation", func(g *types.GenesisState) { g.TrainingFundAllocation = "" }},
		{"missing non-escrow", func(g *types.GenesisState) { g.AccountingState.TrainingFundNonEscrowBalance = "" }},
		{"wrong allocation", func(g *types.GenesisState) { g.TrainingFundAllocation = "4999" }},
		{"excess allocation", func(g *types.GenesisState) { g.TrainingFundAllocation = "5001" }},
		{"non-escrow exceeds allocation", func(g *types.GenesisState) { g.AccountingState.TrainingFundNonEscrowBalance = "5001" }},
		{"missing owner", func(g *types.GenesisState) { g.AugmentationBounties = g.AugmentationBounties[:1] }},
		{"nil owner", func(g *types.GenesisState) { g.AugmentationBounties[0] = nil }},
		{"empty owner", func(g *types.GenesisState) { g.AugmentationBounties[0].Id = "" }},
		{"duplicate owner", func(g *types.GenesisState) {
			g.AugmentationBounties = append(g.AugmentationBounties, g.AugmentationBounties[0])
		}},
		{"nil escrow", func(g *types.GenesisState) { g.AccountingState.AugmentationEscrows[0] = nil }},
		{"duplicate escrow", func(g *types.GenesisState) {
			g.AccountingState.AugmentationEscrows = append(g.AccountingState.AugmentationEscrows, g.AccountingState.AugmentationEscrows[0])
		}},
		{"nil address", func(g *types.GenesisState) { g.AccountingState.BootstrapAddressCounts[0] = nil }},
		{"empty address", func(g *types.GenesisState) { g.AccountingState.BootstrapAddressCounts[0].Address = "" }},
		{"invalid UTF-8 address", func(g *types.GenesisState) { g.AccountingState.BootstrapAddressCounts[0].Address = string([]byte{255}) }},
		{"duplicate address", func(g *types.GenesisState) {
			g.AccountingState.BootstrapAddressCounts = append(g.AccountingState.BootstrapAddressCounts, g.AccountingState.BootstrapAddressCounts[0])
		}},
		{"nil epoch", func(g *types.GenesisState) { g.AccountingState.BootstrapEpochCounts[0] = nil }},
		{"duplicate epoch zero", func(g *types.GenesisState) {
			g.AccountingState.BootstrapEpochCounts = append(g.AccountingState.BootstrapEpochCounts, g.AccountingState.BootstrapEpochCounts[0])
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			gs := proto.Clone(good).(*types.GenesisState)
			tc.mutate(gs)
			require.Error(t, gs.Validate())
			require.Error(t, types.ValidateGenesisAccounting(gs))
		})
	}
	for _, amount := range []string{"", "-1", "+1", "01", " 1", "1.0", strings.Repeat("9", 100)} {
		for _, field := range []string{"escrow", "non-escrow", "allocation"} {
			t.Run(field+"/"+amount, func(t *testing.T) {
				gs := proto.Clone(good).(*types.GenesisState)
				switch field {
				case "escrow":
					gs.AccountingState.AugmentationEscrows[0].RemainingUzrn = amount
				case "non-escrow":
					gs.AccountingState.TrainingFundNonEscrowBalance = amount
				case "allocation":
					gs.TrainingFundAllocation = amount
				}
				require.NotPanics(t, func() { require.Error(t, types.ValidateGenesisAccounting(gs)) })
			})
		}
	}
}

func TestGenesisAccountingAbsentEmptyAndBoundaries(t *testing.T) {
	gs := types.DefaultGenesis()
	require.Nil(t, gs.AccountingState)
	require.NoError(t, gs.Validate(), "legacy/native genesis remains valid")
	gs.TrainingFundAllocation = "0"
	gs.AccountingState = &types.KnowledgeGenesisAccounting{TrainingFundNonEscrowBalance: "0"}
	require.NoError(t, gs.Validate(), "present empty inventory is explicit zero, not unknown")
	gs.AccountingState.BootstrapAddressCounts = []*types.BootstrapAddressCount{{Address: "Exact", Count: ^uint64(0)}, {Address: "exact", Count: 0}}
	gs.AccountingState.BootstrapEpochCounts = []*types.BootstrapEpochCount{{Epoch: ^uint64(0), Count: ^uint64(0)}, {Epoch: 0, Count: 0}}
	require.NoError(t, gs.Validate(), "no address normalization or counter reset")
	max := "115792089237316195423570985008687907853269984665640564039457584007913129639935"
	gs.TrainingFundAllocation = max
	gs.AccountingState.TrainingFundNonEscrowBalance = max
	gs.AugmentationBounties = []*types.AugmentationBounty{{Id: "b"}}
	gs.AccountingState.AugmentationEscrows = []*types.AugmentationEscrowBalance{{BountyId: "b", RemainingUzrn: max}}
	require.NotPanics(t, func() { require.Error(t, gs.Validate()) }, "sum overflow must return an error")
}
