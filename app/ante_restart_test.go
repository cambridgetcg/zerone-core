package app

import (
	"errors"
	"testing"
	"time"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	"github.com/stretchr/testify/require"
)

func TestAnteRestartCommittedHeightScope(t *testing.T) {
	for _, tc := range []struct {
		name                     string
		mode                     sdk.ExecMode
		check, recheck, simulate bool
		height, committed, want  int64
	}{
		{"check", sdk.ExecModeCheck, true, false, false, 0, 9, 9},
		{"recheck", sdk.ExecModeReCheck, true, true, false, 0, 9, 9},
		{"known_header", sdk.ExecModeCheck, true, false, false, 7, 9, 7},
		{"unknown_startup", sdk.ExecModeCheck, true, false, false, 0, 0, 0},
		{"invalid_commit", sdk.ExecModeCheck, true, false, false, 0, -1, 0},
		{"negative_header", sdk.ExecModeCheck, true, false, false, -1, 9, -1},
		{"genesis", sdk.ExecModeCheck, false, false, false, 0, 9, 0},
		{"simulate_flag", sdk.ExecModeCheck, true, false, true, 0, 9, 0},
		{"simulate_mode", sdk.ExecModeSimulate, true, false, false, 0, 9, 0},
		{"prepare", sdk.ExecModePrepareProposal, true, false, false, 0, 9, 0},
		{"process", sdk.ExecModeProcessProposal, true, false, false, 0, 9, 0},
		{"finalize", sdk.ExecModeFinalize, true, false, false, 0, 9, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := sdk.Context{}.WithBlockHeader(cmtproto.Header{Height: tc.height, ChainID: "ante-restart"}).
				WithIsCheckTx(tc.check).WithExecMode(tc.mode)
			if tc.recheck {
				ctx = ctx.WithIsReCheckTx(true).WithExecMode(tc.mode)
			}
			called := false
			nextErr := errors.New("next Ante check refused")
			got, err := (checkTxCommittedHeightDecorator{lastBlockHeight: func() int64 { return tc.committed }}).
				AnteHandle(ctx, nil, tc.simulate, func(nextCtx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
					called = true
					require.Equal(t, tc.want, nextCtx.BlockHeight())
					require.Equal(t, time.Time{}, nextCtx.BlockTime(), "never invent time")
					require.Equal(t, ctx.ChainID(), nextCtx.ChainID())
					require.Equal(t, ctx.ExecMode(), nextCtx.ExecMode())
					require.Equal(t, ctx.HeaderInfo(), nextCtx.HeaderInfo())
					require.Equal(t, tc.simulate, simulate)
					return nextCtx, nextErr
				})
			require.True(t, called)
			require.ErrorIs(t, err, nextErr)
			require.Equal(t, tc.want, got.BlockHeight())
		})
	}
}

func TestSchedulerRestartCheckTxRetainsAnteRefusals(t *testing.T) {
	for _, kind := range []string{"fee", "signature", "timeout", "sequence"} {
		t.Run(kind, func(t *testing.T) {
			f := newSchedulerTestFixture(t, schedulerTestOptions{disk: true, schedules: 1, due: 10, releaseHeight: 1})
			f.block(t, 2)
			sequence := uint64(0)
			if kind == "sequence" {
				sequence = 7
			}
			tx := f.cancel(t, 1, 1, 0, sequence)
			decoded, err := f.app.TxConfig().TxDecoder()(tx)
			require.NoError(t, err)
			builder, err := f.app.TxConfig().WrapTxBuilder(decoded)
			require.NoError(t, err)
			var want uint32
			switch kind {
			case "fee":
				builder.SetFeeAmount(nil)
				want = sdkerrors.ErrInsufficientFee.ABCICode()
			case "signature":
				sigs, err := decoded.(authsigning.SigVerifiableTx).GetSignaturesV2()
				require.NoError(t, err)
				sigs[0].Data.(*signing.SingleSignatureData).Signature[0] ^= 1
				require.NoError(t, builder.SetSignatures(sigs...))
				want = sdkerrors.ErrUnauthorized.ABCICode()
			case "timeout":
				builder.SetTimeoutHeight(1)
				want = sdkerrors.ErrTxTimeoutHeight.ABCICode()
			case "sequence":
				want = sdkerrors.ErrWrongSequence.ABCICode()
			}
			tx, err = f.app.TxConfig().TxEncoder()(builder.GetTx())
			require.NoError(t, err)
			before := schedulerSnapshot(t, f)
			f.reopen(t)
			require.Zero(t, f.app.GetContextForCheckTx(nil).BlockHeight())
			checked := f.check(t, tx)
			require.Equal(t, want, checked.Code, checked.Log)
			require.Equal(t, before, schedulerSnapshot(t, f))
		})
	}
}
