package cross_stack_test

import (
	"context"
	"testing"

	sdkmath "cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authsign "github.com/cosmos/cosmos-sdk/x/auth/signing"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/stretchr/testify/require"

	zeroneapp "github.com/zerone-chain/zerone/app"
	zeroneauthtypes "github.com/zerone-chain/zerone/x/auth/types"
)

// Cosmos SDK v0.50.15 panicked in x/auth/tx.GetSigningTxData when a signer
// omitted PublicKey, even though omission is valid once the account already
// stores that key. SDK v0.53.8 backports the nil guard. Exercise the complete
// Zerone ante chain so the dependency cannot regress unnoticed.
func TestAnteSDK053_KnownSignerMayOmitPublicKey(t *testing.T) {
	h := NewTestHarness(t)
	tx, _ := signedTxWithOmittedStoredPublicKey(t, h)

	ctx := h.Ctx.WithIsSigverifyTx(true)
	var anteErr error
	require.NotPanics(t, func() {
		_, anteErr = zeroneapp.NewAnteHandler(h.App)(ctx, tx, false)
	})
	require.NoError(t, anteErr)
}

func TestAnteSDK053_OmittedStoredPublicKeyStillEnforcesFrozenAccount(t *testing.T) {
	h := NewTestHarness(t)
	tx, sender := signedTxWithOmittedStoredPublicKey(t, h)

	h.AuthKeeper.SetAccount(h.Ctx, &zeroneauthtypes.Account{
		Address:     sender.String(),
		Did:         "did:zrn:abcdef0123456789abcdef0123456789",
		AccountType: "agent",
		Flags:       &zeroneauthtypes.AccountFlags{Frozen: true},
	})

	ctx := h.Ctx.WithIsSigverifyTx(true)
	var anteErr error
	require.NotPanics(t, func() {
		_, anteErr = zeroneapp.NewAnteHandler(h.App)(ctx, tx, false)
	})
	require.Error(t, anteErr)
	require.True(t, zeroneauthtypes.ErrAccountFrozen.Is(anteErr),
		"full v0.53 ante chain must authenticate the stored key before enforcing Zerone freeze policy: %v", anteErr)
}

func signedTxWithOmittedStoredPublicKey(t *testing.T, h *TestHarness) (sdk.Tx, sdk.AccAddress) {
	t.Helper()

	privKey := secp256k1.GenPrivKey()
	sender := sdk.AccAddress(privKey.PubKey().Address())
	recipient := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address())

	require.NoError(t, h.FundAccount(
		sender,
		sdk.NewCoins(sdk.NewCoin(zeroneapp.BondDenom, sdkmath.NewInt(1_000_000))),
	))

	account := h.AccountKeeper.NewAccountWithAddress(h.Ctx, sender)
	require.NoError(t, account.SetPubKey(privKey.PubKey()))
	h.AccountKeeper.SetAccount(h.Ctx, account)

	txConfig := h.App.TxConfig()
	txBuilder := txConfig.NewTxBuilder()
	require.NoError(t, txBuilder.SetMsgs(banktypes.NewMsgSend(
		sender,
		recipient,
		sdk.NewCoins(sdk.NewCoin(zeroneapp.BondDenom, sdkmath.NewInt(1))),
	)))
	const gasLimit uint64 = 200_000
	txBuilder.SetGasLimit(gasLimit)
	txBuilder.SetFeeAmount(sdk.NewCoins(sdk.NewCoin(
		zeroneapp.BondDenom,
		sdkmath.NewIntFromUint64(gasLimit*zeroneapp.MinGasPrice),
	)))

	signMode, err := authsign.APISignModeToInternal(txConfig.SignModeHandler().DefaultMode())
	require.NoError(t, err)

	// Populate AuthInfo with the deliberately omitted key before generating
	// sign bytes. This call reached the nil dereference on SDK v0.50.15.
	require.NoError(t, txBuilder.SetSignatures(signing.SignatureV2{
		PubKey:   nil,
		Data:     &signing.SingleSignatureData{SignMode: signMode},
		Sequence: account.GetSequence(),
	}))

	signerData := authsign.SignerData{
		Address:       sender.String(),
		ChainID:       h.Ctx.ChainID(),
		AccountNumber: account.GetAccountNumber(),
		Sequence:      account.GetSequence(),
		PubKey:        privKey.PubKey(),
	}
	signBytes, err := authsign.GetSignBytesAdapter(
		context.Background(),
		txConfig.SignModeHandler(),
		signMode,
		signerData,
		txBuilder.GetTx(),
	)
	require.NoError(t, err)

	signature, err := privKey.Sign(signBytes)
	require.NoError(t, err)
	require.NoError(t, txBuilder.SetSignatures(signing.SignatureV2{
		PubKey: nil,
		Data: &signing.SingleSignatureData{
			SignMode:  signMode,
			Signature: signature,
		},
		Sequence: account.GetSequence(),
	}))

	tx := txBuilder.GetTx()
	sigTx, ok := tx.(authsign.SigVerifiableTx)
	require.True(t, ok)
	pubKeys, err := sigTx.GetPubKeys()
	require.NoError(t, err)
	require.Len(t, pubKeys, 1)
	require.Nil(t, pubKeys[0], "fixture must omit SignerInfo.public_key")
	signers, err := sigTx.GetSigners()
	require.NoError(t, err)
	require.Equal(t, [][]byte{sender.Bytes()}, signers)

	return tx, sender
}
