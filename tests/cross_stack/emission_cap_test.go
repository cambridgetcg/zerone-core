package cross_stack_test

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"

	zeroneapp "github.com/zerone-chain/zerone/app"
	cpotkeeper "github.com/zerone-chain/zerone/x/claiming_pot/keeper"
	cpottypes "github.com/zerone-chain/zerone/x/claiming_pot/types"
)

// ════════════════════════════════════════════════════════════════════
// Doctrine binding: TWO emission pathways, ONE cap-gated mint entry.
//
// The chain has exactly two pathways for ZRN to enter circulation —
// PoT block rewards (x/vesting_rewards) and bootstrap claims
// (x/claiming_pot). Both pathways gate through MintWithCap. This test
// drives a real bootstrap claim through the live keepers and confirms
// the bank's uzrn supply increases by the claim amount (proving mint,
// not transfer from a pre-funded module account).
//
// Doctrine: docs/tokenomics/GENESIS.md ("Zero Team Allocation — Two
// Emission Paths, Both Gated by Participation"); docs/tokenomics/
// SUPPLY.md ("Emission Pathways"). The pre-fund-then-transfer model
// would let bootstrap claims leak supply outside the cap-tracking
// counter; this test refuses that model.
// ════════════════════════════════════════════════════════════════════

// TestEmissionCap_BootstrapClaimMintsOnDemand confirms that a successful
// MsgClaim against the claiming_pot module increases bank.GetSupply
// for uzrn by exactly the claim amount, and that the claiming_pot
// module account ends the transaction with zero balance (transient
// conduit semantics).
func TestEmissionCap_BootstrapClaimMintsOnDemand(t *testing.T) {
	h := NewTestHarness(t)

	// Authority creates a fully-vested whitelisted pot. We bypass the
	// MsgCreatePot authority gate by calling SetPot directly — the
	// authority gate is exercised in claiming_pot/keeper unit tests; the
	// emission-pathway test cares about the mint, not the authority
	// path.
	claimant := sdk.AccAddress(make([]byte, 20))
	for i := range claimant {
		claimant[i] = byte(0x10 + i) // arbitrary non-zero address
	}
	claimantBech := claimant.String()

	pot := &cpottypes.ClaimingPot{
		Id:            "test-emission-pot",
		Name:          "Emission-Pathway Test Pot",
		TotalAmount:   "1000000", // 1 ZRN
		ClaimedAmount: "0",
		Schedule: &cpottypes.VestingSchedule{
			StartBlock: 1,
			EndBlock:   10,
		},
		Eligibility: &cpottypes.EligibilityCriteria{
			Whitelist: []string{claimantBech},
		},
		Status: cpottypes.PotStatus_POT_STATUS_ACTIVE,
	}
	h.ClaimingPotKeeper.SetPot(h.Ctx, pot)

	// Advance past the pot's EndBlock so the entire amount is vested and
	// claimable in a single MsgClaim. The harness InitChains at height 1.
	h.AdvanceBlocks(20)

	// Snapshot supply + cap counter BEFORE the claim.
	preSupply := h.App.BankKeeper.GetSupply(h.Ctx, zeroneapp.BondDenom)
	preMinted := h.VestingRewardsKeeper.GetTotalMinted(sdk.UnwrapSDKContext(h.Ctx))

	// Claim through the real msg server.
	msgSrv := cpotkeeper.NewMsgServerImpl(h.ClaimingPotKeeper)
	resp, err := msgSrv.Claim(h.Ctx, &cpottypes.MsgClaim{
		Claimant: claimantBech,
		PotId:    pot.Id,
	})
	require.NoError(t, err, "Claim must succeed for whitelisted, vested claimant")
	require.NotEmpty(t, resp.Amount, "Claim response must carry the minted amount")

	// Bank supply increased by exactly the claim amount.
	postSupply := h.App.BankKeeper.GetSupply(h.Ctx, zeroneapp.BondDenom)
	delta := postSupply.Amount.Sub(preSupply.Amount)
	require.Equal(t, resp.Amount, delta.String(),
		"bank supply delta (%s) must equal claim amount (%s) — bootstrap pathway must MINT, not transfer", delta, resp.Amount)

	// vesting_rewards TotalMinted counter advanced by the same amount —
	// confirming the bootstrap pathway gated through MintWithCap and
	// participates in the same cap budget as block rewards.
	postMinted := h.VestingRewardsKeeper.GetTotalMinted(sdk.UnwrapSDKContext(h.Ctx))
	mintedDelta := new(big.Int).Sub(postMinted, preMinted)
	require.Equal(t, resp.Amount, mintedDelta.String(),
		"vesting_rewards.TotalMinted delta (%s) must equal claim amount (%s) — both emission pathways share the cap counter", mintedDelta, resp.Amount)

	// Module account is transient — should have zero uzrn balance after
	// the claim. The mint flowed in and the SendCoinsFromModuleToAccount
	// flowed it back out within the same transaction.
	moduleAddr := h.App.AccountKeeper.GetModuleAddress(cpottypes.ModuleName)
	moduleBalance := h.App.BankKeeper.GetBalance(h.Ctx, moduleAddr, zeroneapp.BondDenom)
	require.True(t, moduleBalance.Amount.IsZero(),
		"claiming_pot module account must hold zero uzrn after claim (transient conduit, not custodian); got %s", moduleBalance)

	// Claimer received the funds.
	claimerBalance := h.App.BankKeeper.GetBalance(h.Ctx, claimant, zeroneapp.BondDenom)
	require.Equal(t, resp.Amount, claimerBalance.Amount.String(),
		"claimer balance must equal the minted amount")
}
