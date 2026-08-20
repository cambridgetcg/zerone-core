package types_test

import (
	"math"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/zerone-chain/zerone/x/sponsorship/types"
)

func legacyOrder(id, sponsor string) *types.BountyOrder {
	return &types.BountyOrder{
		Id: id, Sponsor: sponsor, Domain: "math", PricePerArtifact: "100",
		TargetCount: 1, FulfilledCount: 1, EscrowRemaining: "0",
		StartBlock: 10, EndBlock: 100, Status: types.BountyStatus_BOUNTY_STATUS_FULFILLED,
	}
}

func legacyFulfillment(bountyID, factID, worker string) *types.BountyFulfillment {
	return &types.BountyFulfillment{
		BountyId: bountyID, FactId: factID, Worker: worker, AmountPaid: "100", FulfilledAtBlock: 50,
	}
}

func TestGenesisValidate_LegacyCrossBountyFactReplayRemainsImportable(t *testing.T) {
	sponsor := mkAddr("legacy-genesis-sponsor")
	worker := mkAddr("legacy-genesis-worker1")
	gs := &types.GenesisState{
		Params: types.DefaultParams(),
		Orders: []*types.BountyOrder{
			legacyOrder("bounty-1", sponsor), legacyOrder("bounty-2", sponsor),
		},
		Fulfillments: []*types.BountyFulfillment{
			legacyFulfillment("bounty-1", "same-legacy-fact", worker),
			legacyFulfillment("bounty-2", "same-legacy-fact", worker),
		},
		NextBountyId: 3,
	}
	require.NoError(t, gs.Validate())
}

func TestGenesisValidate_V2CrossBountyReplayRejected(t *testing.T) {
	sponsor := mkAddr("v2-genesis-sponsor-12")
	worker := mkAddr("v2-genesis-worker-123")
	contract := validWorkContract()
	contract.WorkerAddress = worker
	receipt := strings.Repeat("a", 64)
	artifact := strings.Repeat("5", 64)
	nullifier := types.ComputeSettlementNullifier(
		contract.WorkSpecHash, contract.AcceptanceHash, contract.InputRoot, contract.EnvironmentRoot, artifact,
		contract.WorkerAddress,
	)
	order1 := legacyOrder("bounty-1", sponsor)
	order1.WorkContract = contract
	order2 := legacyOrder("bounty-2", sponsor)
	copyContract := *contract
	order2.WorkContract = &copyContract
	fulfillment1 := legacyFulfillment("bounty-1", "same-v2-fact", worker)
	fulfillment1.WorkReceiptHash, fulfillment1.SettlementNullifier, fulfillment1.ArtifactRoot = receipt, nullifier, artifact
	fulfillment2 := legacyFulfillment("bounty-2", "same-v2-fact", worker)
	fulfillment2.WorkReceiptHash, fulfillment2.SettlementNullifier, fulfillment2.ArtifactRoot = receipt, nullifier, artifact
	gs := &types.GenesisState{
		Params: types.DefaultParams(), Orders: []*types.BountyOrder{order1, order2},
		Fulfillments: []*types.BountyFulfillment{fulfillment1, fulfillment2}, NextBountyId: 3,
	}
	require.Error(t, gs.Validate())
}

func TestGenesisValidate_V2WorkerAssignmentAndNullifierBinding(t *testing.T) {
	sponsor := mkAddr("v2-worker-genesis-sp")
	worker := mkAddr("v2-worker-genesis-wk")
	contract := validWorkContract()
	contract.WorkerAddress = worker
	artifact := strings.Repeat("5", 64)
	order := legacyOrder("bounty-1", sponsor)
	order.WorkContract = contract
	fulfillment := legacyFulfillment(order.Id, "worker-bound-fact", worker)
	fulfillment.WorkReceiptHash = strings.Repeat("a", 64)
	fulfillment.ArtifactRoot = artifact
	fulfillment.SettlementNullifier = types.ComputeSettlementNullifier(
		contract.WorkSpecHash, contract.AcceptanceHash, contract.InputRoot,
		contract.EnvironmentRoot, artifact, worker,
	)
	gs := &types.GenesisState{
		Params: types.DefaultParams(), Orders: []*types.BountyOrder{order},
		Fulfillments: []*types.BountyFulfillment{fulfillment}, NextBountyId: 2,
	}
	require.NoError(t, gs.Validate())

	fulfillment.Worker = mkAddr("wrong-genesis-worker")
	require.Error(t, gs.Validate(), "fulfillment worker must equal contract assignment")
	fulfillment.Worker = worker
	fulfillment.SettlementNullifier = strings.Repeat("b", 64)
	require.Error(t, gs.Validate(), "genesis must rederive the worker-bound nullifier")
}

func TestGenesisValidate_EscrowMathAndCounts(t *testing.T) {
	sponsor := mkAddr("genesis-open-sponsor12")
	order := &types.BountyOrder{
		Id: "bounty-1", Sponsor: sponsor, Domain: "math", PricePerArtifact: "100",
		TargetCount: 3, FulfilledCount: 1, EscrowRemaining: "201",
		StartBlock: 10, EndBlock: 100, Status: types.BountyStatus_BOUNTY_STATUS_ACTIVE,
		WorkContract: validWorkContract(),
	}
	gs := &types.GenesisState{Params: types.DefaultParams(), Orders: []*types.BountyOrder{order}, NextBountyId: 2}
	require.Error(t, gs.Validate(), "remaining liability must be exactly price * unpaid slots")

	order.EscrowRemaining = "200"
	require.Error(t, gs.Validate(), "fulfilled_count must equal exported fulfillment records")
}

func TestAmountValidationAndNullifierDeterminism(t *testing.T) {
	_, err := types.ParsePositiveAmount("01")
	require.Error(t, err)
	_, err = types.ParsePositiveAmount("-1")
	require.Error(t, err)
	_, err = types.ParsePositiveAmount("1" + strings.Repeat("0", 100))
	require.Error(t, err, "oversized values must be rejected before sdkmath.NewIntFromBigInt")
	legacyRemaining, err := types.NormalizeLegacyNonNegativeAmount("+002")
	require.NoError(t, err)
	require.Equal(t, "2", legacyRemaining)
	legacyZero, err := types.NormalizeLegacyNonNegativeAmount("000")
	require.NoError(t, err)
	require.Equal(t, "0", legacyZero)
	_, err = types.NormalizeLegacyNonNegativeAmount("-1")
	require.Error(t, err)

	work := strings.Repeat("1", 64)
	acceptance := strings.Repeat("2", 64)
	input := strings.Repeat("3", 64)
	environment := strings.Repeat("4", 64)
	artifact := strings.Repeat("2", 64)
	worker := mkAddr("nullifier-worker-123")
	n := types.ComputeSettlementNullifier(work, acceptance, input, environment, artifact, worker)
	require.Len(t, n, 64)
	require.Equal(t, n, types.ComputeSettlementNullifier(work, acceptance, input, environment, artifact, worker))
	require.NotEqual(t, n, types.ComputeSettlementNullifier(strings.Repeat("a", 64), acceptance, input, environment, artifact, worker))
	require.NotEqual(t, n, types.ComputeSettlementNullifier(work, strings.Repeat("b", 64), input, environment, artifact, worker))
	require.NotEqual(t, n, types.ComputeSettlementNullifier(work, acceptance, strings.Repeat("c", 64), environment, artifact, worker))
	require.NotEqual(t, n, types.ComputeSettlementNullifier(work, acceptance, input, strings.Repeat("d", 64), artifact, worker))
	require.NotEqual(t, n, types.ComputeSettlementNullifier(work, acceptance, input, environment, strings.Repeat("e", 64), worker))
	require.NotEqual(t, n, types.ComputeSettlementNullifier(work, acceptance, input, environment, artifact, mkAddr("other-nullifier-work")))
	require.Equal(t,
		"008c1206e6517a690f46d17beb87fd120152fc87b842dfeb880b7f0f9d47a1ae",
		types.ComputeSettlementNullifier(
			strings.Repeat("1", 64), strings.Repeat("2", 64), strings.Repeat("3", 64),
			strings.Repeat("4", 64), strings.Repeat("5", 64), "zrn1v3jkzervd9hx2ttfdejx27pdwdcx7m3dwexsmf",
		),
		"cross-language settlement vector",
	)
}

func TestGenesisValidate_AcceptsExhaustedNextBountyIDSentinel(t *testing.T) {
	gs := types.DefaultGenesis()
	gs.NextBountyId = math.MaxUint64
	require.NoError(t, gs.Validate())
}

func TestGenesisValidate_ClampsLegacyOnlyMaxActiveParam(t *testing.T) {
	sponsor := mkAddr("legacy-param-cap-spon")
	gs := &types.GenesisState{
		Params: &types.Params{
			MinTargetCount: 1, MinDurationBlocks: 100,
			MaxActiveBountiesPerSponsor: types.MaxActiveBountiesPerSponsorHardCap + 50,
		},
		Orders: []*types.BountyOrder{{
			Id: "bounty-1", Sponsor: sponsor, Domain: "math", PricePerArtifact: "1",
			TargetCount: 1, EscrowRemaining: "1", StartBlock: 10, EndBlock: 100,
			Status: types.BountyStatus_BOUNTY_STATUS_ACTIVE,
		}},
		NextBountyId: 2,
	}
	require.NoError(t, gs.Validate())
	require.Equal(t, types.MaxActiveBountiesPerSponsorHardCap, gs.Params.MaxActiveBountiesPerSponsor)

	gs.Orders[0].WorkContract = validWorkContract()
	gs.Params.MaxActiveBountiesPerSponsor = types.MaxActiveBountiesPerSponsorHardCap + 1
	require.Error(t, gs.Validate(), "bound v2 genesis must never use legacy parameter normalization")
}

func TestWorkContract_ZeroAndPositiveCorroborationPoliciesValidate(t *testing.T) {
	contract := validWorkContract()
	contract.MinCorroborations = 0
	require.NoError(t, contract.Validate(), "zero means closed challenge window with no survived formal challenge")
	contract.MinCorroborations = 2
	require.NoError(t, contract.Validate(), "sponsors may demand additional survived challenges")
	contract.WorkerAddress = ""
	require.Error(t, contract.Validate(), "bound work must preassign a worker")
	contract.WorkerAddress = "not-bech32"
	require.Error(t, contract.Validate(), "worker assignment must be a valid account address")
	contract.WorkerAddress = strings.ToUpper(mkAddr("canonical-worker-alias"))
	require.Error(t, contract.Validate(), "uppercase aliases must not create distinct worker-bound nullifiers")
}

func TestWorkContract_WorkerAddressCanonicalityClosesAliasReplay(t *testing.T) {
	contract := validWorkContract()
	canonical := mkAddr("canonical-null-worker")
	alias := strings.ToUpper(canonical)
	contract.WorkerAddress = canonical
	require.NoError(t, contract.Validate())

	canonicalNullifier := types.ComputeSettlementNullifier(
		contract.WorkSpecHash, contract.AcceptanceHash, contract.InputRoot,
		contract.EnvironmentRoot, strings.Repeat("5", 64), canonical,
	)
	aliasNullifier := types.ComputeSettlementNullifier(
		contract.WorkSpecHash, contract.AcceptanceHash, contract.InputRoot,
		contract.EnvironmentRoot, strings.Repeat("5", 64), alias,
	)
	require.NotEqual(t, canonicalNullifier, aliasNullifier,
		"raw address bytes are consensus input, so aliases must be rejected at admission")

	contract.WorkerAddress = alias
	require.Error(t, contract.Validate())
}
