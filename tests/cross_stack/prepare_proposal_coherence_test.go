package cross_stack_test

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	sdkmath "cosmossdk.io/math"

	abci "github.com/cometbft/cometbft/abci/types"
	clienttx "github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authsign "github.com/cosmos/cosmos-sdk/x/auth/signing"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	zeroneapp "github.com/zerone-chain/zerone/app"
)

// PrepareProposal/ProcessProposal coherence (CometBFT ABCI++ requirement): a
// block prepared by a correct proposer must be accepted by ProcessProposal.
// ProcessProposal replays the full ante handler at the proposal height and
// time, while CheckTx admitted the mempool at the committed height. These
// fixtures run the real BaseApp PrepareProposal and ProcessProposal over
// signed transactions on committed state.

const coherenceMaxTxBytes = 4 << 20

type coherenceTx struct {
	sequence         uint64
	gas              uint64
	fee              uint64
	memo             string
	timeoutHeight    uint64
	timeoutTimestamp time.Time
}

func signedCoherenceTx(
	t *testing.T,
	h *TestHarness,
	privateKey *secp256k1.PrivKey,
	accountNumber uint64,
	spec coherenceTx,
) []byte {
	t.Helper()

	txConfig := h.App.TxConfig()
	sender := sdk.AccAddress(privateKey.PubKey().Address())
	builder := txConfig.NewTxBuilder()
	require.NoError(t, builder.SetMsgs(banktypes.NewMsgSend(
		sender,
		sdk.AccAddress(bytes.Repeat([]byte{0x42}, 20)),
		sdk.NewCoins(sdk.NewInt64Coin(zeroneapp.BondDenom, 1)),
	)))
	builder.SetGasLimit(spec.gas)
	builder.SetFeeAmount(sdk.NewCoins(sdk.NewCoin(
		zeroneapp.BondDenom,
		sdkmath.NewIntFromUint64(spec.fee),
	)))
	builder.SetMemo(spec.memo)
	builder.SetTimeoutHeight(spec.timeoutHeight)
	if !spec.timeoutTimestamp.IsZero() {
		builder.SetTimeoutTimestamp(spec.timeoutTimestamp)
	}

	signMode := signing.SignMode_SIGN_MODE_DIRECT
	require.NoError(t, builder.SetSignatures(signing.SignatureV2{
		PubKey:   privateKey.PubKey(),
		Data:     &signing.SingleSignatureData{SignMode: signMode},
		Sequence: spec.sequence,
	}))
	signature, err := clienttx.SignWithPrivKey(
		context.Background(),
		signMode,
		authsign.SignerData{
			Address:       sender.String(),
			ChainID:       testChainID,
			AccountNumber: accountNumber,
			Sequence:      spec.sequence,
			PubKey:        privateKey.PubKey(),
		},
		builder,
		privateKey,
		txConfig,
		spec.sequence,
	)
	require.NoError(t, err)
	require.NoError(t, builder.SetSignatures(signature))

	txBytes, err := txConfig.TxEncoder()(builder.GetTx())
	require.NoError(t, err)
	return txBytes
}

// commitCoherenceBlock runs FinalizeBlock and Commit for the next height so
// BaseApp's CheckTx state sits on that committed header, as it does on a node.
func commitCoherenceBlock(t *testing.T, h *TestHarness, blockTime time.Time, txs ...[]byte) *abci.ResponseFinalizeBlock {
	t.Helper()
	response, err := h.App.FinalizeBlock(&abci.RequestFinalizeBlock{
		Height: h.App.LastBlockHeight() + 1,
		Time:   blockTime,
		Txs:    txs,
	})
	require.NoError(t, err)
	_, err = h.App.Commit()
	require.NoError(t, err)
	return response
}

func checkCoherenceTx(t *testing.T, h *TestHarness, txBytes []byte, kind abci.CheckTxType) uint32 {
	t.Helper()
	response, err := h.App.CheckTx(&abci.RequestCheckTx{Tx: txBytes, Type: kind})
	require.NoError(t, err)
	return response.Code
}

func prepareNextProposal(t *testing.T, h *TestHarness, proposalTime time.Time, txs ...[]byte) [][]byte {
	t.Helper()
	response, err := h.App.PrepareProposal(&abci.RequestPrepareProposal{
		Height:     h.App.LastBlockHeight() + 1,
		Time:       proposalTime,
		MaxTxBytes: coherenceMaxTxBytes,
		Txs:        txs,
	})
	require.NoError(t, err)
	return response.Txs
}

func processNextProposal(t *testing.T, h *TestHarness, proposalTime time.Time, txs ...[]byte) abci.ResponseProcessProposal_ProposalStatus {
	t.Helper()
	response, err := h.App.ProcessProposal(&abci.RequestProcessProposal{
		Height: h.App.LastBlockHeight() + 1,
		Time:   proposalTime,
		Txs:    txs,
	})
	require.NoError(t, err)
	return response.Status
}

type coherenceSigner struct {
	key           *secp256k1.PrivKey
	accountNumber uint64
}

// newCoherenceChain installs two funded signers, leaves the consensus block
// gas unlimited and commits one real block at committedTime, so CheckTx runs
// at that committed height and time.
func newCoherenceChain(t *testing.T, committedTime time.Time) (*TestHarness, coherenceSigner, coherenceSigner) {
	t.Helper()
	h := NewTestHarness(t)
	signers := make([]coherenceSigner, 2)
	for i := range signers {
		privateKey := secp256k1.GenPrivKey()
		sender := sdk.AccAddress(privateKey.PubKey().Address())
		account := h.AccountKeeper.NewAccountWithAddress(h.Ctx, sender)
		require.NoError(t, account.SetPubKey(privateKey.PubKey()))
		h.AccountKeeper.SetAccount(h.Ctx, account)
		require.NoError(t, h.FundAccount(sender, sdk.NewCoins(sdk.NewCoin(
			zeroneapp.BondDenom,
			sdkmath.NewInt(proposalTestBalance),
		))))
		signers[i] = coherenceSigner{key: privateKey, accountNumber: account.GetAccountNumber()}
	}
	params, err := h.App.ConsensusKeeper.ParamsStore.Get(h.Ctx)
	require.NoError(t, err)
	require.NotNil(t, params.Block)
	params.Block.MaxGas = -1
	require.NoError(t, h.App.ConsensusKeeper.ParamsStore.Set(h.Ctx, params))
	h.CommitHMinusOne()
	commitCoherenceBlock(t, h, committedTime)
	return h, signers[0], signers[1]
}

func requireCoherentProposal(
	t *testing.T,
	h *TestHarness,
	proposalTime time.Time,
	candidates [][]byte,
	want [][]byte,
) {
	t.Helper()
	require.Equal(t, abci.ResponseProcessProposal_REJECT,
		processNextProposal(t, h, proposalTime, candidates...),
		"fixture must contain a candidate that the proposal-height replay refuses")

	prepared := prepareNextProposal(t, h, proposalTime, candidates...)
	require.Equal(t, want, prepared)
	require.Equal(t, abci.ResponseProcessProposal_ACCEPT,
		processNextProposal(t, h, proposalTime, prepared...),
		"ProcessProposal must accept the block PrepareProposal offered")

	// The prepared block also finalizes and the chain moves on.
	response := commitCoherenceBlock(t, h, proposalTime, prepared...)
	require.Len(t, response.TxResults, len(want))
	for i, result := range response.TxResults {
		require.Zerof(t, result.Code, "tx %d: %s", i, result.Log)
	}
}

func TestPrepareProposalCoherence_KeepsValidCandidatesUnchanged(t *testing.T) {
	committedTime := time.Unix(1_750_000_000, 0).UTC()
	h, first, second := newCoherenceChain(t, committedTime)
	candidates := [][]byte{
		signedCoherenceTx(t, h, first.key, first.accountNumber, coherenceTx{sequence: 0, gas: 200_000, fee: 200_000}),
		signedCoherenceTx(t, h, first.key, first.accountNumber, coherenceTx{sequence: 1, gas: 200_000, fee: 200_000}),
		signedCoherenceTx(t, h, second.key, second.accountNumber, coherenceTx{sequence: 0, gas: 200_000, fee: 200_000}),
	}
	proposalTime := committedTime.Add(5 * time.Second)

	prepared := prepareNextProposal(t, h, proposalTime, candidates...)
	require.Equal(t, candidates, prepared, "valid candidates keep their order and bytes")
	require.Equal(t, abci.ResponseProcessProposal_ACCEPT, processNextProposal(t, h, proposalTime, prepared...))
}

func TestPrepareProposalCoherence_OmitsTimeoutHeightAtCommittedHeight(t *testing.T) {
	committedTime := time.Unix(1_750_000_000, 0).UTC()
	h, first, second := newCoherenceChain(t, committedTime)
	committedHeight := uint64(h.App.LastBlockHeight())

	expiring := signedCoherenceTx(t, h, first.key, first.accountNumber, coherenceTx{
		sequence: 0, gas: 200_000, fee: 200_000, timeoutHeight: committedHeight,
	})
	valid := signedCoherenceTx(t, h, second.key, second.accountNumber, coherenceTx{
		sequence: 0, gas: 200_000, fee: 200_000,
	})
	require.Zero(t, checkCoherenceTx(t, h, expiring, abci.CheckTxType_New),
		"CheckTx at the committed height admits a timeout equal to that height")

	requireCoherentProposal(t, h, committedTime.Add(5*time.Second),
		[][]byte{expiring, valid}, [][]byte{valid})

	require.NotZero(t, checkCoherenceTx(t, h, expiring, abci.CheckTxType_Recheck),
		"the next commit's recheck evicts the expired transaction")
}

func TestPrepareProposalCoherence_OmitsTimeoutTimestampBetweenBlocks(t *testing.T) {
	committedTime := time.Unix(1_750_000_000, 0).UTC()
	h, first, second := newCoherenceChain(t, committedTime)

	expiring := signedCoherenceTx(t, h, first.key, first.accountNumber, coherenceTx{
		sequence: 0, gas: 200_000, fee: 200_000,
		timeoutTimestamp: committedTime.Add(2 * time.Second),
	})
	valid := signedCoherenceTx(t, h, second.key, second.accountNumber, coherenceTx{
		sequence: 0, gas: 200_000, fee: 200_000,
	})
	require.Zero(t, checkCoherenceTx(t, h, expiring, abci.CheckTxType_New),
		"CheckTx at the committed block time admits a later timeout timestamp")

	requireCoherentProposal(t, h, committedTime.Add(5*time.Second),
		[][]byte{expiring, valid}, [][]byte{valid})
}

func TestPrepareProposalCoherence_OmitsCandidateInvalidatedByEarlierCandidate(t *testing.T) {
	committedTime := time.Unix(1_750_000_000, 0).UTC()

	t.Run("same sequence", func(t *testing.T) {
		h, first, second := newCoherenceChain(t, committedTime)
		original := signedCoherenceTx(t, h, first.key, first.accountNumber, coherenceTx{
			sequence: 0, gas: 200_000, fee: 200_000, memo: "first",
		})
		replay := signedCoherenceTx(t, h, first.key, first.accountNumber, coherenceTx{
			sequence: 0, gas: 200_000, fee: 200_000, memo: "second",
		})
		valid := signedCoherenceTx(t, h, second.key, second.accountNumber, coherenceTx{
			sequence: 0, gas: 200_000, fee: 200_000,
		})

		requireCoherentProposal(t, h, committedTime.Add(5*time.Second),
			[][]byte{original, replay, valid}, [][]byte{original, valid})
	})

	t.Run("fee balance", func(t *testing.T) {
		h, first, second := newCoherenceChain(t, committedTime)
		// The first fee leaves 50,000 uzrn, short of the second's 200,000.
		draining := signedCoherenceTx(t, h, first.key, first.accountNumber, coherenceTx{
			sequence: 0, gas: 200_000, fee: proposalTestBalance - 50_000,
		})
		unfunded := signedCoherenceTx(t, h, first.key, first.accountNumber, coherenceTx{
			sequence: 1, gas: 200_000, fee: 200_000,
		})
		valid := signedCoherenceTx(t, h, second.key, second.accountNumber, coherenceTx{
			sequence: 0, gas: 200_000, fee: 200_000,
		})

		requireCoherentProposal(t, h, committedTime.Add(5*time.Second),
			[][]byte{draining, unfunded, valid}, [][]byte{draining, valid})
	})
}

func TestPrepareProposalCoherence_ProposesReplayedBytes(t *testing.T) {
	committedTime := time.Unix(1_750_000_000, 0).UTC()
	h, first, _ := newCoherenceChain(t, committedTime)
	canonical := signedCoherenceTx(t, h, first.key, first.accountNumber, coherenceTx{
		sequence: 0, gas: 200_000, fee: 200_000,
	})
	// A TxRaw that repeats body_bytes, first empty, decodes to the same signed
	// transaction. The proposal carries the bytes PrepareProposal replayed.
	reframed := append([]byte{0x0a, 0x00}, canonical...)
	decoded, err := h.App.TxConfig().TxDecoder()(reframed)
	require.NoError(t, err, "fixture must remain decodable")
	reencoded, err := h.App.TxConfig().TxEncoder()(decoded)
	require.NoError(t, err)
	require.Equal(t, canonical, reencoded)

	proposalTime := committedTime.Add(5 * time.Second)
	prepared := prepareNextProposal(t, h, proposalTime, reframed)
	require.Equal(t, [][]byte{canonical}, prepared)
	require.Equal(t, abci.ResponseProcessProposal_ACCEPT, processNextProposal(t, h, proposalTime, prepared...))
}
