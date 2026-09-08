package app

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth/ante"
	ibcante "github.com/cosmos/ibc-go/v10/modules/core/ante"
)

// checkTxCommittedHeightDecorator fills the empty CheckTx header height that
// SDK 0.53.8 BaseApp.Init installs on LoadLatestVersion. LastBlockHeight reads
// the loaded commit ID, not a proposal or volatile FinalizeBlock height. Match
// SDK Commit's CheckTx convention: committed H, NOT H+1 (IsHalted already
// accounts for CheckTx targeting the next proposal). Preserve the original
// time/header metadata; it is not available from a committed height alone.
// All subsequent Ante checks run normally, including fees, timeouts, signatures
// and account policy. Genesis, simulation and consensus contexts are untouched.
type checkTxCommittedHeightDecorator struct {
	lastBlockHeight func() int64
}

func (d checkTxCommittedHeightDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	checkMode := ctx.ExecMode() == sdk.ExecModeCheck || ctx.ExecMode() == sdk.ExecModeReCheck
	if !simulate && checkMode && (ctx.IsCheckTx() || ctx.IsReCheckTx()) && ctx.BlockHeight() == 0 {
		if committedHeight := d.lastBlockHeight(); committedHeight > 0 {
			ctx = ctx.WithBlockHeight(committedHeight)
		}
	}
	return next(ctx, tx, simulate)
}

// NewAnteHandler returns an AnteHandler with:
// 1. Standard Cosmos SDK decorators (explicit chain, not wrapped)
// 2. ZRN-specific gas cost validation
// 3. Fee routing: 19.67% development, 3.33% research, ~77% normal distribution
// 4. Zerone custom decorators:
//   - Bootstrap gas-free period for PoT bootstrap
//   - Emergency transaction quarantine (emergency coordination plus an exact
//     expedited SDK software-upgrade proposal lane)
//   - Frozen fee-granter enforcement before fee deduction
//   - DID resolution and validation
//   - Frozen account enforcement + LastActiveBlock tracking
//   - Account capability enforcement
//   - Funding-correlation telemetry (observational; never applied to vote weight)
func NewAnteHandler(app *ZeroneApp) sdk.AnteHandler {
	return sdk.ChainAnteDecorators(
		// Restore only the missing restart CheckTx height, before any height-
		// sensitive checks. No consensus execution mode uses this fallback.
		checkTxCommittedHeightDecorator{lastBlockHeight: app.LastBlockHeight},

		// --- IBC ---
		ibcante.NewRedundantRelayDecorator(app.IBCKeeper),

		// --- Gas Meter Init (must be before any gas consumption) ---
		ante.NewSetUpContextDecorator(),

		// --- Bootstrap Gas-Free (RETIRED: window = 0 at mainnet; no-op, kept for gov re-activation) ---
		NewBootstrapGasFreeDecorator(),

		// --- Emergency transaction quarantine (consensus continues) ---
		NewEmergencyHaltDecorator(
			app.EmergencyKeeper,
			newSDKGovRecoveryProposalReader(
				app.GovKeeper,
				app.EmergencyKeeper,
				app.UpgradeKeeper,
				&app.ZeroneGovKeeper,
			),
		),

		// --- ZRN Pre-Auth (gas meter available) ---
		NewZRNGasDecorator(),
		NewFeeRouterDecorator(app.BankKeeper),

		// --- Standard Cosmos SDK Decorators ---
		ante.NewExtensionOptionsDecorator(nil),
		ante.NewValidateBasicDecorator(),
		ante.NewTxTimeoutHeightDecorator(),
		ante.NewValidateMemoDecorator(app.AccountKeeper),
		ante.NewConsumeGasForTxSizeDecorator(app.AccountKeeper),
		NewZeroneFeeGranterDecorator(app.ZeroneAuthKeeper),
		ante.NewDeductFeeDecorator(app.AccountKeeper, app.BankKeeper, app.FeeGrantKeeper, nil),
		ante.NewSetPubKeyDecorator(app.AccountKeeper),
		ante.NewValidateSigCountDecorator(app.AccountKeeper),
		ante.NewSigGasConsumeDecorator(app.AccountKeeper, ante.DefaultSigVerificationGasConsumer),
		ante.NewSigVerificationDecorator(app.AccountKeeper, app.txConfig.SignModeHandler()),
		NewEmergencyAuthenticationDecorator(),
		ante.NewIncrementSequenceDecorator(app.AccountKeeper),

		// --- Funding Correlation Telemetry (not vote-weight evidence) ---
		NewSybilFundingDecorator(&app.ZeroneGovKeeper),

		// --- Zerone Post-Auth (signer authenticated) ---
		NewZeroneDIDDecorator(app.ZeroneAuthKeeper),
		NewZeroneAccountDecorator(app.ZeroneAuthKeeper),
		NewZeroneCapabilityDecorator(app.ZeroneAuthKeeper),
	)
}
