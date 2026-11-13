package app

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/stretchr/testify/require"

	knowledgetypes "github.com/zerone-chain/zerone/x/knowledge/types"
)

// The seat-deposit account (reviewer-entry step B, rule D2) holds seat
// deposits and nothing else. It is a maccPerms entry with no permissions, so
// blockedModuleAccountAddrs puts it on the bank's blocked list: no send,
// multi-send, IBC receive or withdraw address can pay into it.
//
// This is the one behaviour of this binary that differs from 3a844c48 with the
// seat-deposit marker off: the address is blocked at app construction, whatever
// the marker. A MsgSend to it, which 3a844c48 accepts, is refused here. That is
// a stated deviation from marker-off parity, pinned by this test.
func TestSeatDepositModuleAccountIsBlocked(t *testing.T) {
	perms, listed := maccPerms[knowledgetypes.SeatDepositModuleName]
	require.True(t, listed)
	require.Empty(t, perms, "the account can neither mint, burn nor stake")
	addr := authtypes.NewModuleAddress(knowledgetypes.SeatDepositModuleName)
	require.True(t, blockedModuleAccountAddrs()[addr.String()])

	app, ctx := settlementFixture(t)
	require.True(t, app.BankKeeper.BlockedAddr(addr))
	sender := settlementAddress(201)
	coins := sdk.NewCoins(sdk.NewInt64Coin("uzrn", 1))
	require.NoError(t, app.BankKeeper.MintCoins(ctx, knowledgetypes.BootstrapFundModuleName, coins))
	require.NoError(t, app.BankKeeper.SendCoinsFromModuleToAccount(ctx, knowledgetypes.BootstrapFundModuleName, sender, coins))
	_, err := bankkeeper.NewMsgServerImpl(app.BankKeeper).Send(ctx, &banktypes.MsgSend{FromAddress: sender.String(), ToAddress: addr.String(), Amount: coins})
	require.ErrorContains(t, err, "not allowed to receive funds")
	require.True(t, app.BankKeeper.GetBalance(ctx, addr, "uzrn").IsZero())
	require.Nil(t, app.AccountKeeper.GetAccount(ctx, addr), "no account is created until the first deposit")
}

// Marker-off parity for the accounting-authority upgrade (review finding on
// the first draft). Its module-account permission check and its auth evidence
// set read maccPerms, which 3a844c48 builds without knowledge_seat_deposits. A
// plain account at that address, which any binary before this one lets anyone
// create with a one-uzrn send, is therefore not permission drift for that
// upgrade, and its auth record is not part of the upgrade's evidence, exactly
// as at 3a844c48, whatever the seat-deposit marker.
func TestSeatDepositAddressIsOutsideTheAccountingAuthorityCheck(t *testing.T) {
	app, ctx, _ := newAccountingAuthorityFixture(t)
	addr := authtypes.NewModuleAddress(knowledgetypes.SeatDepositModuleName)
	require.NoError(t, app.requireAccountingModuleAccountPermissions(ctx))
	app.AccountKeeper.SetAccount(ctx, app.AccountKeeper.NewAccountWithAddress(ctx, addr))
	require.NoError(t, app.requireAccountingModuleAccountPermissions(ctx), "a squatted seat-deposit address is not drift for this upgrade")

	keys := accountingAuthorityModuleAccountKeys()
	require.Len(t, keys, len(maccPerms)-1)
	require.False(t, keys[string(append(authtypes.AddressStoreKeyPrefix.Bytes(), addr...))], "the seat-deposit account is not accounting evidence")
	for name := range maccPerms {
		if name != knowledgetypes.SeatDepositModuleName {
			require.True(t, keys[string(append(authtypes.AddressStoreKeyPrefix.Bytes(), authtypes.NewModuleAddress(name)...))], name)
		}
	}

	// Every other module account is still checked.
	other := authtypes.NewModuleAddress(knowledgetypes.VindicationEscrowModuleName)
	require.Nil(t, app.AccountKeeper.GetAccount(ctx, other))
	app.AccountKeeper.SetAccount(ctx, app.AccountKeeper.NewAccountWithAddress(ctx, other))
	require.ErrorContains(t, app.requireAccountingModuleAccountPermissions(ctx), knowledgetypes.VindicationEscrowModuleName)
}
