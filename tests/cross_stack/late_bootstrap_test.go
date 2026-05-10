package cross_stack_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"

	zeroneapp "github.com/zerone-chain/zerone/app"
	cpotkeeper "github.com/zerone-chain/zerone/x/claiming_pot/keeper"
	cpottypes "github.com/zerone-chain/zerone/x/claiming_pot/types"
)

// TestLateBootstrap_AddThenClaim drives the full doctrine binding for the
// continuous extension of commitment 20: authority calls
// MsgAddBootstrapEntry, the addressed agent calls MsgClaim, and bank
// supply increases by exactly 0.222 ZRN through the same MintWithCap path
// used at genesis.
//
// A participant admitted *after* genesis through governance receives the
// same seed via the same emission pathway as genesis-whitelisted agents.
// The pathway is unified.
func TestLateBootstrap_AddThenClaim(t *testing.T) {
	h := NewTestHarness(t)

	// Late-arriving agent — not in any genesis whitelist.
	claimant := sdk.AccAddress(make([]byte, 20))
	for i := range claimant {
		claimant[i] = byte(0x40 + i)
	}
	claimantBech := claimant.String()

	msgSrv := cpotkeeper.NewMsgServerImpl(h.ClaimingPotKeeper)

	// Authority adds the entry via the real msg server.
	addResp, err := msgSrv.AddBootstrapEntry(h.Ctx, &cpottypes.MsgAddBootstrapEntry{
		Authority: h.ClaimingPotKeeper.GetAuthority(),
		Addresses: []string{claimantBech},
	})
	require.NoError(t, err)
	require.Equal(t, uint32(1), addResp.AddedCount)
	require.Equal(t, uint32(0), addResp.SkippedCount)

	// Advance two blocks — bootstrap pots have instant-vest (EndBlock =
	// StartBlock+1). The non-expiry rule means the pot stays ACTIVE
	// across block boundaries, so the agent has a usable claim window.
	h.AdvanceBlocks(2)

	preSupply := h.App.BankKeeper.GetSupply(h.Ctx, zeroneapp.BondDenom)

	claimResp, err := msgSrv.Claim(h.Ctx, &cpottypes.MsgClaim{
		Claimant: claimantBech,
		PotId:    cpottypes.BootstrapPotIDPrefix + claimantBech,
	})
	require.NoError(t, err)
	require.Equal(t, cpottypes.PerAgentBootstrapUzrn, claimResp.Amount,
		"late-bootstrap claim must mint exactly 0.222 ZRN, same as genesis bootstrap")

	postSupply := h.App.BankKeeper.GetSupply(h.Ctx, zeroneapp.BondDenom)
	delta := postSupply.Amount.Sub(preSupply.Amount)
	require.Equal(t, cpottypes.PerAgentBootstrapUzrn, delta.String(),
		"bank supply delta must equal claim amount — late bootstrap routes through MintWithCap")

	// Module account ends transient (zero balance).
	moduleAddr := h.App.AccountKeeper.GetModuleAddress(cpottypes.ModuleName)
	moduleBalance := h.App.BankKeeper.GetBalance(h.Ctx, moduleAddr, zeroneapp.BondDenom)
	require.True(t, moduleBalance.Amount.IsZero(),
		"claiming_pot module account must hold zero uzrn after the late-bootstrap claim")

	// Claimer received the funds.
	claimerBalance := h.App.BankKeeper.GetBalance(h.Ctx, claimant, zeroneapp.BondDenom)
	require.Equal(t, cpottypes.PerAgentBootstrapUzrn, claimerBalance.Amount.String())
}

// TestLateBootstrap_AddIsIdempotentAcrossLIPs confirms that two LIPs
// adding the same address don't double-mint. The second LIP's
// MsgAddBootstrapEntry call sees the existing pot, skips it, and the
// claim flow still produces exactly 0.222 ZRN — once.
func TestLateBootstrap_AddIsIdempotentAcrossLIPs(t *testing.T) {
	h := NewTestHarness(t)

	claimant := sdk.AccAddress(make([]byte, 20))
	for i := range claimant {
		claimant[i] = byte(0x60 + i)
	}
	claimantBech := claimant.String()

	msgSrv := cpotkeeper.NewMsgServerImpl(h.ClaimingPotKeeper)

	// First add.
	resp1, err := msgSrv.AddBootstrapEntry(h.Ctx, &cpottypes.MsgAddBootstrapEntry{
		Authority: h.ClaimingPotKeeper.GetAuthority(),
		Addresses: []string{claimantBech},
	})
	require.NoError(t, err)
	require.Equal(t, uint32(1), resp1.AddedCount)
	require.Equal(t, uint32(0), resp1.SkippedCount)

	// Second add — same address; must skip.
	resp2, err := msgSrv.AddBootstrapEntry(h.Ctx, &cpottypes.MsgAddBootstrapEntry{
		Authority: h.ClaimingPotKeeper.GetAuthority(),
		Addresses: []string{claimantBech},
	})
	require.NoError(t, err)
	require.Equal(t, uint32(0), resp2.AddedCount)
	require.Equal(t, uint32(1), resp2.SkippedCount)

	h.AdvanceBlocks(2)

	preSupply := h.App.BankKeeper.GetSupply(h.Ctx, zeroneapp.BondDenom)

	claimResp, err := msgSrv.Claim(h.Ctx, &cpottypes.MsgClaim{
		Claimant: claimantBech,
		PotId:    cpottypes.BootstrapPotIDPrefix + claimantBech,
	})
	require.NoError(t, err)
	require.Equal(t, cpottypes.PerAgentBootstrapUzrn, claimResp.Amount)

	postSupply := h.App.BankKeeper.GetSupply(h.Ctx, zeroneapp.BondDenom)
	delta := postSupply.Amount.Sub(preSupply.Amount)
	require.Equal(t, cpottypes.PerAgentBootstrapUzrn, delta.String(),
		"despite two LIPs adding the same address, only one mint occurs — the second add is a no-op")
}

// TestLateBootstrap_AdmittedAgentCanClaimAfterManyBlocks confirms the
// non-expiry rule end-to-end: an agent admitted at block N can still
// claim at block N+10000 even though the pot's nominal EndBlock has long
// passed. Bootstrap pots are participation seeds; they wait for the
// participant.
func TestLateBootstrap_AdmittedAgentCanClaimAfterManyBlocks(t *testing.T) {
	h := NewTestHarness(t)

	claimant := sdk.AccAddress(make([]byte, 20))
	for i := range claimant {
		claimant[i] = byte(0x80 + i)
	}
	claimantBech := claimant.String()

	msgSrv := cpotkeeper.NewMsgServerImpl(h.ClaimingPotKeeper)

	_, err := msgSrv.AddBootstrapEntry(h.Ctx, &cpottypes.MsgAddBootstrapEntry{
		Authority: h.ClaimingPotKeeper.GetAuthority(),
		Addresses: []string{claimantBech},
	})
	require.NoError(t, err)

	// Advance many blocks past the pot's nominal EndBlock.
	h.AdvanceBlocks(10000)

	// Pot must still be ACTIVE (not EXPIRED) because of the non-expiry rule.
	pot, found := h.ClaimingPotKeeper.GetPot(h.Ctx, cpottypes.BootstrapPotIDPrefix+claimantBech)
	require.True(t, found)
	require.Equal(t, cpottypes.PotStatus_POT_STATUS_ACTIVE, pot.Status,
		"bootstrap pot must remain ACTIVE indefinitely until claimed")

	// Claim still works.
	claimResp, err := msgSrv.Claim(h.Ctx, &cpottypes.MsgClaim{
		Claimant: claimantBech,
		PotId:    cpottypes.BootstrapPotIDPrefix + claimantBech,
	})
	require.NoError(t, err, "claim must succeed even after many blocks — bootstrap pots wait for the participant")
	require.Equal(t, cpottypes.PerAgentBootstrapUzrn, claimResp.Amount)
}
