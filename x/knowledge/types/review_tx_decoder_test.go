package types

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdktx "github.com/cosmos/cosmos-sdk/types/tx"
	signing "github.com/cosmos/cosmos-sdk/types/tx/signing"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

// Exercise the SDK's ADR-027/recursive unknown-field decoder with a genuinely
// signed envelope. This covers the pre-ante codec boundary, not account/fee
// admission; the two-binary rehearsal separately exercises CheckTx/DeliverTx.
func TestReviewSDKTxDecoderNestedAttestation(t *testing.T) {
	sdk.GetConfig().SetBech32PrefixForAccount("zrn", "zrnpub")
	registry := codectypes.NewInterfaceRegistry()
	RegisterInterfaces(registry)
	cryptocodec.RegisterInterfaces(registry)
	cdc := codec.NewProtoCodec(registry)
	key := secp256k1.GenPrivKey()
	verifier := sdk.AccAddress(key.PubKey().Address()).String()
	message := &MsgSubmitReveal{Verifier: verifier, RoundId: "signed-round", Vote: "accept", Confidence: 800000, Salt: []byte("public-test-salt-0123456789"), Attestation: &ReviewAttestation{Reason: "Checked the declared fixture α.", MethodId: "M-COMPUTATIONAL", Scope: "Fixture only", EvidenceIds: []string{"sha256:fixture-a", "fact:fixture-b"}}}
	envelope := func(msg *MsgSubmitReveal) []byte {
		any, err := codectypes.NewAnyWithValue(msg)
		require.NoError(t, err)
		body, err := (&sdktx.TxBody{Messages: []*codectypes.Any{any}}).Marshal()
		require.NoError(t, err)
		pub, err := codectypes.NewAnyWithValue(key.PubKey())
		require.NoError(t, err)
		auth, err := (&sdktx.AuthInfo{SignerInfos: []*sdktx.SignerInfo{{PublicKey: pub, ModeInfo: &sdktx.ModeInfo{Sum: &sdktx.ModeInfo_Single_{Single: &sdktx.ModeInfo_Single{Mode: signing.SignMode_SIGN_MODE_DIRECT}}}, Sequence: 3}}, Fee: &sdktx.Fee{GasLimit: 500000}}).Marshal()
		require.NoError(t, err)
		signed, err := (&sdktx.SignDoc{BodyBytes: body, AuthInfoBytes: auth, ChainId: "decoder-local-1", AccountNumber: 9}).Marshal()
		require.NoError(t, err)
		signature, err := key.Sign(signed)
		require.NoError(t, err)
		require.True(t, key.PubKey().VerifySignature(signed, signature))
		raw, err := (&sdktx.TxRaw{BodyBytes: body, AuthInfoBytes: auth, Signatures: [][]byte{signature}}).Marshal()
		require.NoError(t, err)
		return raw
	}
	decoded, err := authtx.DefaultTxDecoder(cdc)(envelope(message))
	require.NoError(t, err)
	msgs := decoded.GetMsgs()
	require.Len(t, msgs, 1)
	actual, ok := msgs[0].(*MsgSubmitReveal)
	require.True(t, ok)
	require.True(t, proto.Equal(message, actual))
	bad := proto.Clone(message).(*MsgSubmitReveal)
	bad.Attestation.ProtoReflect().SetUnknown([]byte{0xf8, 0x07, 1})
	_, err = authtx.DefaultTxDecoder(cdc)(envelope(bad))
	require.Error(t, err)
	require.Contains(t, err.Error(), "errUnknownField")
	require.Contains(t, err.Error(), "ReviewAttestation")
}
