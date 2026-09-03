package cross_stack_test

import (
	"bytes"
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	sdkmath "cosmossdk.io/math"

	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	zeroneapp "github.com/zerone-chain/zerone/app"
	zeroneauthtypes "github.com/zerone-chain/zerone/x/auth/types"
)

const proposalTestBalance = 100_000_000

func installProposalSigner(
	t *testing.T,
	h *TestHarness,
	frozen bool,
	maxBlockGas int64,
) (*secp256k1.PrivKey, uint64) {
	t.Helper()

	privateKey := secp256k1.GenPrivKey()
	sender := sdk.AccAddress(privateKey.PubKey().Address())
	account := h.AccountKeeper.NewAccountWithAddress(h.Ctx, sender)
	require.NoError(t, account.SetPubKey(privateKey.PubKey()))
	h.AccountKeeper.SetAccount(h.Ctx, account)
	require.NoError(t, h.FundAccount(
		sender,
		sdk.NewCoins(sdk.NewCoin(
			zeroneapp.BondDenom,
			sdkmath.NewInt(proposalTestBalance),
		)),
	))

	if frozen {
		h.AuthKeeper.SetAccount(h.Ctx, &zeroneauthtypes.Account{
			Address:     sender.String(),
			Did:         "did:zrn:" + string(bytes.Repeat([]byte{'a'}, 64)),
			AccountType: "agent",
			Flags:       &zeroneauthtypes.AccountFlags{Frozen: true},
		})
	}

	params, err := h.App.ConsensusKeeper.ParamsStore.Get(h.Ctx)
	require.NoError(t, err)
	require.NotNil(t, params.Block)
	params.Block.MaxGas = maxBlockGas
	require.NoError(t, h.App.ConsensusKeeper.ParamsStore.Set(h.Ctx, params))

	accountNumber := account.GetAccountNumber()
	h.CommitHMinusOne()
	return privateKey, accountNumber
}

func signedProposalTx(
	t *testing.T,
	h *TestHarness,
	privateKey *secp256k1.PrivKey,
	accountNumber uint64,
	sequence uint64,
	gas uint64,
	feeAmount uint64,
) []byte {
	t.Helper()

	sender := sdk.AccAddress(privateKey.PubKey().Address())
	recipient := sdk.AccAddress(bytes.Repeat([]byte{0x42}, 20))
	message := banktypes.NewMsgSend(
		sender,
		recipient,
		sdk.NewCoins(sdk.NewInt64Coin(zeroneapp.BondDenom, 1)),
	)
	fee := sdk.NewCoins(sdk.NewCoin(
		zeroneapp.BondDenom,
		sdkmath.NewIntFromUint64(feeAmount),
	))
	tx, err := simtestutil.GenSignedMockTx(
		rand.New(rand.NewSource(int64(sequence)+1)),
		h.App.TxConfig(),
		[]sdk.Msg{message},
		fee,
		gas,
		testChainID,
		[]uint64{accountNumber},
		[]uint64{sequence},
		privateKey,
	)
	require.NoError(t, err)
	txBytes, err := h.App.TxConfig().TxEncoder()(tx)
	require.NoError(t, err)
	return txBytes
}

func processProposalAtNextHeight(
	t *testing.T,
	h *TestHarness,
	txs ...[]byte,
) *abci.ResponseProcessProposal {
	t.Helper()
	height := h.App.LastBlockHeight() + 1
	response, err := h.App.ProcessProposal(&abci.RequestProcessProposal{
		Height: height,
		Time:   time.Unix(1_750_000_000+height, 0).UTC(),
		Txs:    txs,
	})
	require.NoError(t, err)
	return response
}

func TestProcessProposalRunsFullAnteChain(t *testing.T) {
	t.Run("valid signed transaction", func(t *testing.T) {
		h := NewTestHarness(t)
		privateKey, accountNumber := installProposalSigner(t, h, false, -1)
		txBytes := signedProposalTx(t, h, privateKey, accountNumber, 0, 200_000, 200_000)

		response := processProposalAtNextHeight(t, h, txBytes)
		require.Equal(t, abci.ResponseProcessProposal_ACCEPT, response.Status)
	})

	t.Run("unsigned transaction", func(t *testing.T) {
		h := NewTestHarness(t)
		privateKey, _ := installProposalSigner(t, h, false, -1)
		sender := sdk.AccAddress(privateKey.PubKey().Address())
		builder := h.App.TxConfig().NewTxBuilder()
		require.NoError(t, builder.SetMsgs(banktypes.NewMsgSend(
			sender,
			sdk.AccAddress(bytes.Repeat([]byte{0x24}, 20)),
			sdk.NewCoins(sdk.NewInt64Coin(zeroneapp.BondDenom, 1)),
		)))
		builder.SetGasLimit(200_000)
		builder.SetFeeAmount(sdk.NewCoins(sdk.NewInt64Coin(
			zeroneapp.BondDenom,
			200_000,
		)))
		txBytes, err := h.App.TxConfig().TxEncoder()(builder.GetTx())
		require.NoError(t, err)

		response := processProposalAtNextHeight(t, h, txBytes)
		require.Equal(t, abci.ResponseProcessProposal_REJECT, response.Status)
	})

	t.Run("underpriced fee", func(t *testing.T) {
		h := NewTestHarness(t)
		privateKey, accountNumber := installProposalSigner(t, h, false, -1)
		txBytes := signedProposalTx(t, h, privateKey, accountNumber, 0, 200_000, 199_999)

		response := processProposalAtNextHeight(t, h, txBytes)
		require.Equal(t, abci.ResponseProcessProposal_REJECT, response.Status)
	})

	t.Run("out of order sequence", func(t *testing.T) {
		h := NewTestHarness(t)
		privateKey, accountNumber := installProposalSigner(t, h, false, -1)
		txBytes := signedProposalTx(t, h, privateKey, accountNumber, 1, 200_000, 200_000)

		response := processProposalAtNextHeight(t, h, txBytes)
		require.Equal(t, abci.ResponseProcessProposal_REJECT, response.Status)
	})

	t.Run("frozen signer", func(t *testing.T) {
		h := NewTestHarness(t)
		privateKey, accountNumber := installProposalSigner(t, h, true, -1)
		txBytes := signedProposalTx(t, h, privateKey, accountNumber, 0, 200_000, 200_000)

		response := processProposalAtNextHeight(t, h, txBytes)
		require.Equal(t, abci.ResponseProcessProposal_REJECT, response.Status)
	})
}

func TestProcessProposalPreservesZeroConsensusMaxGas(t *testing.T) {
	h := NewTestHarness(t)
	privateKey, accountNumber := installProposalSigner(t, h, false, 0)
	txBytes := signedProposalTx(t, h, privateKey, accountNumber, 0, 200_000, 200_000)

	response := processProposalAtNextHeight(t, h, txBytes)
	require.Equal(t, abci.ResponseProcessProposal_REJECT, response.Status,
		"a zero consensus gas ceiling must reject a positive-gas transaction")
}

func TestProcessProposalEnforcesAggregateConsensusGas(t *testing.T) {
	const (
		blockGas  = int64(20_000_000)
		firstGas  = uint64(10_000_000)
		exactGas  = uint64(10_000_000)
		excessGas = uint64(10_000_001)
	)

	for name, test := range map[string]struct {
		secondGas uint64
		status    abci.ResponseProcessProposal_ProposalStatus
	}{
		"exact limit": {secondGas: exactGas, status: abci.ResponseProcessProposal_ACCEPT},
		"over limit":  {secondGas: excessGas, status: abci.ResponseProcessProposal_REJECT},
	} {
		t.Run(name, func(t *testing.T) {
			h := NewTestHarness(t)
			privateKey, accountNumber := installProposalSigner(t, h, false, blockGas)
			first := signedProposalTx(t, h, privateKey, accountNumber, 0, firstGas, firstGas)
			second := signedProposalTx(
				t, h, privateKey, accountNumber, 1, test.secondGas, test.secondGas,
			)

			response := processProposalAtNextHeight(t, h, first, second)
			require.Equal(t, test.status, response.Status)
		})
	}
}
