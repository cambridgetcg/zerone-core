package app

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkmempool "github.com/cosmos/cosmos-sdk/types/mempool"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/stretchr/testify/require"
)

func TestApplicationMempoolSupportsStoredKeyWireShape(t *testing.T) {
	from := sdk.AccAddress([]byte("stored-key-sender___"))
	to := sdk.AccAddress([]byte("stored-key-receiver_"))
	builder := MakeEncodingConfig().TxConfig.NewTxBuilder()
	require.NoError(t, builder.SetMsgs(banktypes.NewMsgSend(
		from,
		to,
		sdk.NewCoins(sdk.NewInt64Coin(BondDenom, 1)),
	)))
	require.NoError(t, builder.SetSignatures(signing.SignatureV2{
		PubKey: nil,
		Data: &signing.SingleSignatureData{
			SignMode: signing.SignMode_SIGN_MODE_DIRECT,
		},
		Sequence: 7,
	}))
	tx := builder.GetTx()

	var signers []sdkmempool.SignerData
	require.NotPanics(t, func() {
		var err error
		signers, err = newMessageSignerExtractionAdapter().GetSigners(tx)
		require.NoError(t, err)
	})
	require.Len(t, signers, 1)
	require.Equal(t, from.String(), signers[0].Signer.String())
	require.Equal(t, uint64(7), signers[0].Sequence)

	pool := NewApplicationMempool(10)
	ctx := sdk.Context{}.WithPriority(1)
	require.NotPanics(t, func() {
		require.NoError(t, pool.Insert(ctx, tx))
	})
	require.Equal(t, 1, pool.CountTx())
	require.NotPanics(t, func() {
		require.NoError(t, pool.Remove(tx))
	})
	require.Zero(t, pool.CountTx())
	require.ErrorIs(t, pool.Remove(tx), sdkmempool.ErrTxNotFound)
}

func TestApplicationMempoolRequiresBoundedPositiveCapacity(t *testing.T) {
	for _, maxTx := range []int{ApplicationMempoolMinTxs, ApplicationMempoolMaxTxs} {
		require.NoError(t, ValidateApplicationMempoolMaxTx(maxTx))
		require.NotPanics(t, func() {
			pool := NewApplicationMempool(maxTx)
			require.NotNil(t, pool)
		})
	}

	for _, maxTx := range []int{-1, 0, ApplicationMempoolMaxTxs + 1} {
		require.Error(t, ValidateApplicationMempoolMaxTx(maxTx))
		require.Panics(t, func() {
			NewApplicationMempool(maxTx)
		})
	}
}
