package app

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth/ante"
	ibcante "github.com/cosmos/ibc-go/v8/modules/core/ante"
)

// NewAnteHandler returns an AnteHandler with:
// 1. Standard Cosmos SDK decorators (explicit chain, not wrapped)
// 2. ZRN-specific gas cost validation
// 3. Fee routing: 7% to research fund, 93% to validators
// 4. Zerone custom decorators:
//   - Bootstrap gas-free period for PoT bootstrap
//   - Emergency halt gate (block non-emergency txs when halted)
//   - DID resolution and validation
//   - Frozen account enforcement + LastActiveBlock tracking
//   - Session key capability enforcement
//   - Sybil funding tracker for vote-weight decay
func NewAnteHandler(app *ZeroneApp) sdk.AnteHandler {
	return sdk.ChainAnteDecorators(
		// --- IBC ---
		ibcante.NewRedundantRelayDecorator(app.IBCKeeper),

		// --- Gas Meter Init (must be before any gas consumption) ---
		ante.NewSetUpContextDecorator(),

		// --- Bootstrap Gas-Free (RETIRED: window = 0 at mainnet; no-op, kept for gov re-activation) ---
		NewBootstrapGasFreeDecorator(),

		// --- Emergency Halt Gate (blocks non-emergency txs when chain is halted) ---
		NewEmergencyHaltDecorator(app.EmergencyKeeper),

		// --- ZRN Pre-Auth (gas meter available) ---
		NewZRNGasDecorator(),
		NewFeeRouterDecorator(app.BankKeeper),

		// --- Standard Cosmos SDK Decorators ---
		ante.NewExtensionOptionsDecorator(nil),
		ante.NewValidateBasicDecorator(),
		ante.NewTxTimeoutHeightDecorator(),
		ante.NewValidateMemoDecorator(app.AccountKeeper),
		ante.NewConsumeGasForTxSizeDecorator(app.AccountKeeper),
		ante.NewDeductFeeDecorator(app.AccountKeeper, app.BankKeeper, app.FeeGrantKeeper, nil),
		ante.NewSetPubKeyDecorator(app.AccountKeeper),
		ante.NewValidateSigCountDecorator(app.AccountKeeper),
		ante.NewSigGasConsumeDecorator(app.AccountKeeper, ante.DefaultSigVerificationGasConsumer),
		ante.NewSigVerificationDecorator(app.AccountKeeper, app.txConfig.SignModeHandler()),
		ante.NewIncrementSequenceDecorator(app.AccountKeeper),

		// --- Sybil Funding Tracker (records MsgSend sender->recipient for vote decay) ---
		NewSybilFundingDecorator(&app.ZeroneGovKeeper),

		// --- Zerone Post-Auth (signer authenticated) ---
		NewZeroneDIDDecorator(app.ZeroneAuthKeeper),
		NewZeroneAccountDecorator(app.ZeroneAuthKeeper),
		NewZeroneCapabilityDecorator(app.ZeroneAuthKeeper, app.AccountKeeper),
	)
}
