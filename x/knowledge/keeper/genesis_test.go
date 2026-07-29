package keeper_test

import (
	"strings"
	"testing"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// ─── InitGenesis ─────────────────────────────────────────────────────────────

func TestInitGenesis_DefaultGenesis(t *testing.T) {
	k, ctx := setupKnowledgeTest(t)

	params, err := k.GetParams(ctx)
	require.NoError(t, err)
	require.Equal(t, uint64(3), params.MinVerifiers)
	require.Equal(t, uint64(22), params.MaxVerifiers)
}

func TestInitGenesis_CustomParams(t *testing.T) {
	k, ctx := setupKnowledgeTest(t)

	custom := types.DefaultParams()
	custom.MinVerifiers = 7
	custom.MaxVerifiers = 44
	custom.CommitPhaseBlocks = 10

	gs := &types.GenesisState{
		Params:  &custom,
		Domains: types.DefaultDomains(),
	}
	require.NoError(t, k.InitGenesis(ctx, gs))

	got, err := k.GetParams(ctx)
	require.NoError(t, err)
	require.Equal(t, uint64(7), got.MinVerifiers)
	require.Equal(t, uint64(44), got.MaxVerifiers)
	require.Equal(t, uint64(10), got.CommitPhaseBlocks)
}

func TestInitGenesis_DefaultDomains(t *testing.T) {
	k, ctx := setupKnowledgeTest(t)

	var count int
	k.IterateDomains(ctx, func(_ *types.Domain) bool {
		count++
		return false
	})
	// 16 epistemic domains from DefaultGenesis + 4 doctrine domains
	// seeded by LoadDoctrineFacts (SL-M1).
	require.Equal(t, 22, count)
}

func TestInitGenesis_WithPreexistingFacts(t *testing.T) {
	k, ctx := setupKnowledgeTest(t)

	fact := &types.Fact{
		Id:      "genesis-fact-1",
		Content: "Water boils at 100C at sea level",
		Domain:  "physics",
		Status:  types.FactStatus_FACT_STATUS_VERIFIED,
	}

	gs := &types.GenesisState{
		Params:  func() *types.Params { p := types.DefaultParams(); return &p }(),
		Domains: types.DefaultDomains(),
		Facts:   []*types.Fact{fact},
	}
	require.NoError(t, k.InitGenesis(ctx, gs))

	got, found := k.GetFact(ctx, "genesis-fact-1")
	require.True(t, found)
	require.Equal(t, "Water boils at 100C at sea level", got.Content)
}

func TestInitGenesis_WithPreexistingClaims(t *testing.T) {
	k, ctx := setupKnowledgeTest(t)

	claim := &types.Claim{
		Id:          "genesis-claim-1",
		FactContent: "Test genesis claim content here",
		Domain:      "mathematics",
		Submitter:   "zrn1sub",
		Status:      types.ClaimStatus_CLAIM_STATUS_PENDING,
		Stake:       "1000000",
	}

	gs := &types.GenesisState{
		Params:        func() *types.Params { p := types.DefaultParams(); return &p }(),
		Domains:       types.DefaultDomains(),
		PendingClaims: []*types.Claim{claim},
	}
	require.NoError(t, k.InitGenesis(ctx, gs))

	got, found := k.GetClaim(ctx, "genesis-claim-1")
	require.True(t, found)
	require.Equal(t, "Test genesis claim content here", got.FactContent)
}

func TestInitGenesis_SkipsNilEntries(t *testing.T) {
	k, ctx := setupKnowledgeTest(t)

	gs := &types.GenesisState{
		Params:        func() *types.Params { p := types.DefaultParams(); return &p }(),
		Domains:       []*types.Domain{nil, {Name: "valid_domain", Status: types.DomainStatus_DOMAIN_STATUS_ACTIVE}, nil},
		Facts:         []*types.Fact{nil},
		PendingClaims: []*types.Claim{nil},
		ActiveRounds:  []*types.VerificationRound{nil},
	}
	require.NoError(t, k.InitGenesis(ctx, gs))

	_, found := k.GetDomain(ctx, "valid_domain")
	require.True(t, found)
}

// ─── Fund allocation export/import ──────────────────────────────────────────

func TestInitGenesis_ExportedFundBalancesAreNotReminted(t *testing.T) {
	k, ctx, bank := setupKnowledgeTestWithBank(t)

	// setupKnowledgeTestWithBank starts from DefaultGenesis, so the bootstrap
	// fund is already non-zero. Seed a non-zero training fund balance as x/bank
	// would restore it before x/knowledge during a full app import.
	trainingBalance := sdkmath.NewInt(7_654_321)
	bank.moduleBalances[types.TrainingFundModuleName] = sdk.NewCoins(
		sdk.NewCoin("uzrn", trainingBalance),
	)

	exported := k.ExportGenesis(ctx)
	bootstrapBefore := bank.moduleBalances[types.BootstrapFundModuleName].AmountOf("uzrn")
	trainingBefore := bank.moduleBalances[types.TrainingFundModuleName].AmountOf("uzrn")
	mintedBefore := bank.minted.AmountOf("uzrn")

	require.True(t, bootstrapBefore.IsPositive())
	require.Equal(t, bootstrapBefore.String(), exported.BootstrapFundAllocation)
	require.Equal(t, trainingBefore.String(), exported.TrainingFundAllocation)

	// Re-import the exported knowledge state after the bank balances already
	// exist. Neither allocation may be minted a second time.
	require.NoError(t, k.InitGenesis(ctx, exported))
	require.True(t, bank.moduleBalances[types.BootstrapFundModuleName].AmountOf("uzrn").Equal(bootstrapBefore))
	require.True(t, bank.moduleBalances[types.TrainingFundModuleName].AmountOf("uzrn").Equal(trainingBefore))
	require.True(t, bank.minted.AmountOf("uzrn").Equal(mintedBefore),
		"export/import must not increase minted supply")
}

func TestInitGenesis_FundAllocationsMintOnlyMissingBalance(t *testing.T) {
	k, ctx, bank := setupKnowledgeTestWithBank(t)

	// Simulate a partially pre-funded bank genesis. The knowledge allocations
	// are target balances: InitGenesis should mint only each shortfall.
	bank.moduleBalances[types.BootstrapFundModuleName] = sdk.NewCoins(sdk.NewInt64Coin("uzrn", 400))
	bank.moduleBalances[types.TrainingFundModuleName] = sdk.NewCoins(sdk.NewInt64Coin("uzrn", 500))
	mintedBefore := bank.minted.AmountOf("uzrn")

	gs := types.DefaultGenesis()
	gs.BootstrapFundAllocation = "1000"
	gs.TrainingFundAllocation = "2000"
	require.NoError(t, k.InitGenesis(ctx, gs))

	require.Equal(t, "1000", bank.moduleBalances[types.BootstrapFundModuleName].AmountOf("uzrn").String())
	require.Equal(t, "2000", bank.moduleBalances[types.TrainingFundModuleName].AmountOf("uzrn").String())
	mintedDelta := bank.minted.AmountOf("uzrn").Sub(mintedBefore)
	require.Equal(t, "2100", mintedDelta.String(), "must mint only 600 + 1500 uzrn shortfalls")
}

func TestInitGenesis_RejectsMalformedOrUnboundedFundAllocationsWithoutPanic(t *testing.T) {
	tests := []struct {
		name     string
		field    string
		value    string
		wantText string
	}{
		{name: "negative bootstrap", field: "bootstrap", value: "-1", wantText: "bootstrap fund"},
		{name: "noncanonical bootstrap", field: "bootstrap", value: "01", wantText: "bootstrap fund"},
		{name: "overflow bootstrap", field: "bootstrap", value: strings.Repeat("9", 100), wantText: "bootstrap fund"},
		{name: "malformed training", field: "training", value: "1.0", wantText: "training fund"},
		{name: "overflow training", field: "training", value: strings.Repeat("9", 100), wantText: "training fund"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			k, ctx, bank := setupKnowledgeTestWithBank(t)
			mintedBefore := bank.minted.AmountOf("uzrn")
			gs := types.DefaultGenesis()
			if tc.field == "bootstrap" {
				gs.BootstrapFundAllocation = tc.value
			} else {
				gs.TrainingFundAllocation = tc.value
			}

			var err error
			require.NotPanics(t, func() { err = k.InitGenesis(ctx, gs) })
			require.ErrorContains(t, err, tc.wantText)
			require.True(t, bank.minted.AmountOf("uzrn").Equal(mintedBefore),
				"invalid allocation must not mint")
		})
	}
}

// ─── ExportGenesis ──────────────────────────────────────────────────────────

func TestExportGenesis_RoundTrip(t *testing.T) {
	k, ctx := setupKnowledgeTest(t)

	// Add extra state
	makeTestFact(t, k, ctx, "export-fact-1", "Export test content", "physics", "empirical", "zrn1sub", 800_000)

	// Export
	exported := k.ExportGenesis(ctx)
	require.NotNil(t, exported)
	require.NotNil(t, exported.Params)
	require.GreaterOrEqual(t, len(exported.Domains), 18)

	// Verify fact is in export
	var foundFact bool
	for _, f := range exported.Facts {
		if f.Id == "export-fact-1" {
			foundFact = true
			require.Equal(t, "Export test content", f.Content)
		}
	}
	require.True(t, foundFact)
}

func TestExportGenesis_PreservesParams(t *testing.T) {
	k, ctx := setupKnowledgeTest(t)

	params := types.DefaultParams()
	params.MinVerifiers = 11
	require.NoError(t, k.SetParams(ctx, &params))

	exported := k.ExportGenesis(ctx)
	require.Equal(t, uint64(11), exported.Params.MinVerifiers)
}

func TestExportGenesis_IncludesActiveRounds(t *testing.T) {
	k, ctx := setupKnowledgeTest(t)

	round := makeRoundInPhase("export-round", "c1", types.VerificationPhase_VERIFICATION_PHASE_COMMIT, 100)
	require.NoError(t, k.SetVerificationRound(ctx, round))

	exported := k.ExportGenesis(ctx)
	var foundRound bool
	for _, r := range exported.ActiveRounds {
		if r.Id == "export-round" {
			foundRound = true
		}
	}
	require.True(t, foundRound)
}

// ─── GenesisState Validation ────────────────────────────────────────────────

func TestGenesisValidate_Default(t *testing.T) {
	gs := types.DefaultGenesis()
	require.NoError(t, gs.Validate())
}

func TestGenesisValidate_NilParams(t *testing.T) {
	gs := types.DefaultGenesis()
	gs.Params = nil
	require.Error(t, gs.Validate())
}

func TestGenesisValidate_InvalidParams(t *testing.T) {
	gs := types.DefaultGenesis()
	gs.Params.MinVerifiers = 0 // invalid
	require.Error(t, gs.Validate())
}

func TestGenesisValidate_FundAllocationsAreCanonicalAndBounded(t *testing.T) {
	for _, field := range []string{"bootstrap", "training"} {
		for _, value := range []string{"-1", "00", "+1", "1.0", " 1", strings.Repeat("9", 100)} {
			t.Run(field+"/"+value, func(t *testing.T) {
				gs := types.DefaultGenesis()
				if field == "bootstrap" {
					gs.BootstrapFundAllocation = value
				} else {
					gs.TrainingFundAllocation = value
				}
				require.Error(t, gs.Validate())
			})
		}
	}

	gs := types.DefaultGenesis()
	gs.BootstrapFundAllocation = ""
	gs.TrainingFundAllocation = ""
	require.NoError(t, gs.Validate(), "historical empty allocations remain zero-compatible")
}

func TestGenesisValidate_DuplicateFactIDs(t *testing.T) {
	gs := types.DefaultGenesis()
	gs.Facts = []*types.Fact{
		{Id: "same-id", Content: "First"},
		{Id: "same-id", Content: "Second"},
	}
	require.Error(t, gs.Validate())
}

func TestGenesisValidate_DuplicateClaimIDs(t *testing.T) {
	gs := types.DefaultGenesis()
	gs.PendingClaims = []*types.Claim{
		{Id: "dup-claim"},
		{Id: "dup-claim"},
	}
	require.Error(t, gs.Validate())
}

func TestGenesisValidate_DuplicateDomainNames(t *testing.T) {
	gs := types.DefaultGenesis()
	gs.Domains = append(gs.Domains, &types.Domain{Name: "mathematics"})
	require.Error(t, gs.Validate())
}

func TestGenesisValidate_InvalidReference(t *testing.T) {
	gs := types.DefaultGenesis()
	gs.Facts = []*types.Fact{
		{Id: "fact-1", Content: "Valid", References: []string{"nonexistent"}},
	}
	require.Error(t, gs.Validate())
}

func TestGenesisValidate_ValidReferences(t *testing.T) {
	gs := types.DefaultGenesis()
	gs.Facts = []*types.Fact{
		{Id: "fact-a", Content: "Base fact"},
		{Id: "fact-b", Content: "Depends on A", References: []string{"fact-a"}},
	}
	require.NoError(t, gs.Validate())
}
