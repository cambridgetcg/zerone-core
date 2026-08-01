package cross_stack_test

import (
	"testing"

	txv1beta1 "cosmossdk.io/api/cosmos/tx/v1beta1"
	"github.com/stretchr/testify/require"

	sdkmath "cosmossdk.io/math"

	gogoproto "github.com/cosmos/gogoproto/proto"
	protov2 "google.golang.org/protobuf/proto"

	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	zeroneapp "github.com/zerone-chain/zerone/app"
	zeroneauthtypes "github.com/zerone-chain/zerone/x/auth/types"
)

func TestAntePolicy_OmittedStoredPubKeyStillEnforcesFrozenAccount(t *testing.T) {
	h := NewTestHarness(t)

	privKey := secp256k1.GenPrivKey()
	sender := sdk.AccAddress(privKey.PubKey().Address())
	recipient := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address())

	baseAccount := h.AccountKeeper.NewAccountWithAddress(h.Ctx, sender)
	require.NoError(t, baseAccount.SetPubKey(privKey.PubKey()))
	h.AccountKeeper.SetAccount(h.Ctx, baseAccount)
	require.NoError(
		t,
		h.FundAccount(
			sender,
			sdk.NewCoins(sdk.NewCoin(zeroneapp.BondDenom, sdkmath.NewInt(1_000_000))),
		),
	)

	h.AuthKeeper.SetAccount(h.Ctx, &zeroneauthtypes.Account{
		Address:     sender.String(),
		Did:         "did:zrn:abcdef0123456789abcdef0123456789",
		AccountType: "agent",
		Flags:       &zeroneauthtypes.AccountFlags{Frozen: true},
	})

	txConfig := h.App.TxConfig()
	builder := txConfig.NewTxBuilder()
	require.NoError(t, builder.SetMsgs(&banktypes.MsgSend{
		FromAddress: sender.String(),
		ToAddress:   recipient.String(),
		Amount:      sdk.NewCoins(sdk.NewInt64Coin(zeroneapp.BondDenom, 1)),
	}))

	const gasLimit uint64 = 200_000
	builder.SetGasLimit(gasLimit)
	builder.SetFeeAmount(sdk.NewCoins(sdk.NewCoin(
		zeroneapp.BondDenom,
		sdkmath.NewIntFromUint64(gasLimit*zeroneapp.MinGasPrice),
	)))

	// Build the exact wire shape accepted by Cosmos once BaseAccount stores the
	// key: SignerInfo carries a mode and sequence, but no public key.
	placeholder := signing.SignatureV2{
		PubKey: nil,
		Data: &signing.SingleSignatureData{
			SignMode: signing.SignMode_SIGN_MODE_DIRECT,
		},
		Sequence: baseAccount.GetSequence(),
	}
	require.NoError(t, builder.SetSignatures(placeholder))

	unsignedBytes, err := txConfig.TxEncoder()(builder.GetTx())
	require.NoError(t, err)

	var txRaw txtypes.TxRaw
	require.NoError(t, gogoproto.Unmarshal(unsignedBytes, &txRaw))
	signBytes, err := (protov2.MarshalOptions{Deterministic: true}).Marshal(&txv1beta1.SignDoc{
		BodyBytes:     txRaw.BodyBytes,
		AuthInfoBytes: txRaw.AuthInfoBytes,
		ChainId:       testChainID,
		AccountNumber: baseAccount.GetAccountNumber(),
	})
	require.NoError(t, err)
	rawSignature, err := privKey.Sign(signBytes)
	require.NoError(t, err)
	require.True(t, privKey.PubKey().VerifySignature(signBytes, rawSignature))
	txRaw.Signatures = [][]byte{rawSignature}
	txBytes, err := gogoproto.Marshal(&txRaw)
	require.NoError(t, err)
	decodedTx, err := txConfig.TxDecoder()(txBytes)
	require.NoError(t, err)

	sigTx, ok := decodedTx.(authsigning.SigVerifiableTx)
	require.True(t, ok)
	pubKeys, err := sigTx.GetPubKeys()
	require.NoError(t, err)
	require.Len(t, pubKeys, 1)
	require.Nil(t, pubKeys[0], "test must exercise an omitted SignerInfo.public_key")
	signers, err := sigTx.GetSigners()
	require.NoError(t, err)
	require.Equal(t, [][]byte{sender.Bytes()}, signers)

	// Exercise Zerone's policy on a real, validly signed TxRaw. Cosmos SDK
	// v0.50.15's V2 signing adapter currently panics on this otherwise
	// permitted omitted-key shape before standard verification completes; the
	// custom policy must not independently rely on that upstream behavior.
	_, err = zeroneapp.NewZeroneAccountDecorator(h.AuthKeeper).AnteHandle(
		h.Ctx,
		decodedTx,
		false,
		func(ctx sdk.Context, _ sdk.Tx, _ bool) (sdk.Context, error) {
			return ctx, nil
		},
	)
	require.Error(t, err)
	require.True(t, zeroneauthtypes.ErrAccountFrozen.Is(err), "unexpected policy error: %v", err)
}
