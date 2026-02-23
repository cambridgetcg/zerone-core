package app

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	govtypes "github.com/zerone-chain/zerone/x/gov/types"
)

// SybilFundingDecorator intercepts MsgSend and MsgMultiSend transactions to
// record sender->recipient funding relationships for sybil vote-weight decay.
//
// Placed after IncrementSequenceDecorator (signer authenticated) and before
// the Zerone post-auth decorators.
type SybilFundingDecorator struct {
	govKeeper govtypes.FundingRecorder
}

// NewSybilFundingDecorator creates a new SybilFundingDecorator.
func NewSybilFundingDecorator(gk govtypes.FundingRecorder) SybilFundingDecorator {
	return SybilFundingDecorator{govKeeper: gk}
}

func (sfd SybilFundingDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	// Nil-safe: skip if gov keeper not wired
	if sfd.govKeeper == nil {
		return next(ctx, tx, simulate)
	}

	// Skip simulation — recording is a state mutation
	if simulate {
		return next(ctx, tx, simulate)
	}

	blockHeight := uint64(ctx.BlockHeight())

	for _, msg := range tx.GetMsgs() {
		switch m := msg.(type) {
		case *banktypes.MsgSend:
			sfd.recordSend(ctx, m.FromAddress, m.ToAddress, m.Amount, blockHeight)
		case *banktypes.MsgMultiSend:
			sfd.recordMultiSend(ctx, m, blockHeight)
		}
	}

	return next(ctx, tx, simulate)
}

// recordSend records a single sender->recipient funding relationship.
// Only records transfers that include the uzrn denomination.
func (sfd SybilFundingDecorator) recordSend(ctx sdk.Context, sender, recipient string, amount sdk.Coins, blockHeight uint64) {
	zrnAmount := amount.AmountOf(BondDenom)
	if zrnAmount.IsZero() {
		return
	}
	sfd.govKeeper.RecordFunding(ctx, sender, recipient, zrnAmount.String(), blockHeight)
}

// recordMultiSend records funding relationships from a MsgMultiSend.
// Each input sender is recorded as funding each output recipient proportionally.
func (sfd SybilFundingDecorator) recordMultiSend(ctx sdk.Context, msg *banktypes.MsgMultiSend, blockHeight uint64) {
	for _, input := range msg.Inputs {
		zrnAmount := input.Coins.AmountOf(BondDenom)
		if zrnAmount.IsZero() {
			continue
		}
		for _, output := range msg.Outputs {
			outAmount := output.Coins.AmountOf(BondDenom)
			if outAmount.IsZero() {
				continue
			}
			sfd.govKeeper.RecordFunding(ctx, input.Address, output.Address, outAmount.String(), blockHeight)
		}
	}
}
