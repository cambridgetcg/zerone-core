package app

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// PrepareProposalSelectionHandlerForTest runs PrepareProposal's byte and gas
// selection with a pass-through replay, for unsigned selection fixtures that
// cannot pass the ante handler and run outside BaseApp.PrepareProposal.
// Production always replays through BaseApp.PrepareProposalVerifyTx; the
// cross-stack proposal coherence tests exercise that path.
func (app *ZeroneApp) PrepareProposalSelectionHandlerForTest() sdk.PrepareProposalHandler {
	return app.prepareProposalHandler(proposalTxVerifier(app.txConfig.TxEncoder()))
}

// PrepareProposalHandlerWithVerifierForTest runs PrepareProposal with a
// caller-supplied replay, for fixtures that make one candidate's replay fail
// or panic.
func (app *ZeroneApp) PrepareProposalHandlerWithVerifierForTest(verify func(sdk.Tx) ([]byte, error)) sdk.PrepareProposalHandler {
	return app.prepareProposalHandler(proposalTxVerifier(verify))
}
