package ibc_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"

	dbm "github.com/cosmos/cosmos-db"
	"github.com/stretchr/testify/suite"

	"cosmossdk.io/log"
	sdkmath "cosmossdk.io/math"
	"cosmossdk.io/x/feegrant"

	abci "github.com/cometbft/cometbft/abci/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"

	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authsign "github.com/cosmos/cosmos-sdk/x/auth/signing"

	ibctesting "github.com/cosmos/ibc-go/v8/testing"

	zeroneapp "github.com/zerone-chain/zerone/app"
	ibcratelimittypes "github.com/zerone-chain/zerone/x/ibcratelimit/types"
)

func init() {
	ibctesting.DefaultTestingAppInit = SetupTestingApp
}

// SetupTestingApp creates a ZeroneApp for use with the ibctesting framework.
func SetupTestingApp() (ibctesting.TestingApp, map[string]json.RawMessage) {
	db := dbm.NewMemDB()
	app := zeroneapp.NewZeroneApp(
		log.NewNopLogger(),
		db,
		nil,  // traceStore
		true, // loadLatest
		simtestutil.NewAppOptionsWithFlagHome(os.TempDir()),
	)
	return app, app.DefaultGenesis()
}

// IBCTestSuite is the testify suite for IBC integration tests.
type IBCTestSuite struct {
	suite.Suite

	coordinator  *ibctesting.Coordinator
	chainA       *ibctesting.TestChain
	chainB       *ibctesting.TestChain
	transferPath *ibctesting.Path
}

// TestIBCTestSuite runs the IBC test suite.
func TestIBCTestSuite(t *testing.T) {
	suite.Run(t, new(IBCTestSuite))
}

// SetupTest initializes the coordinator, chains, and transfer path.
func (s *IBCTestSuite) SetupTest() {
	s.coordinator = ibctesting.NewCoordinator(s.T(), 2)
	s.chainA = s.coordinator.GetChain(ibctesting.GetChainID(1))
	s.chainB = s.coordinator.GetChain(ibctesting.GetChainID(2))

	// Must be installed before the transfer path setup: the channel handshake
	// itself delivers txs, which the consensus min-fee rule applies to.
	s.installFeePaidSendMsgs(s.chainA)
	s.installFeePaidSendMsgs(s.chainB)

	s.transferPath = ibctesting.NewTransferPath(s.chainA, s.chainB)
	s.coordinator.Setup(s.transferPath)
}

// installFeePaidSendMsgs replaces the chain's default SendMsgs — which signs
// every tx with a ZERO fee (ibc-go testing/simapp.SignAndDeliver hard-codes
// it) — with a variant that declares the consensus minimum fee and pays it
// through an x/feegrant allowance from a secondary sender account.
//
// The ZRNGasDecorator enforces the minimum fee for ALL txs at height >= 1
// (the zero-fee consensus bypass was closed pre-genesis, see app/ante_zerone.go),
// so the stock zero-fee test txs are rejected. Routing the fee through a
// granter keeps the primary SenderAccount's balance untouched, preserving
// this suite's exact escrow/refund equality assertions, and exercises the
// same feegrant fee-payment path mainnet onboarding uses.
func (s *IBCTestSuite) installFeePaidSendMsgs(chain *ibctesting.TestChain) {
	granter := chain.SenderAccounts[1].SenderAccount.GetAddress()

	app := GetZeroneApp(chain)
	err := app.FeeGrantKeeper.GrantAllowance(
		chain.GetContext(), granter, chain.SenderAccount.GetAddress(), &feegrant.BasicAllowance{},
	)
	s.Require().NoError(err)

	// Minimum fee the ZRNGasDecorator accepts for this gas limit.
	fee := sdk.NewCoins(sdk.NewCoin(
		zeroneapp.BondDenom,
		sdkmath.NewIntFromUint64(simtestutil.DefaultGenTxGas*zeroneapp.MinGasPrice),
	))

	chain.SendMsgsOverride = func(msgs ...sdk.Msg) (*abci.ExecTxResult, error) {
		// Mirrors TestChain.SendMsgs (ibc-go v8 testing/chain.go), differing
		// only in the fee fields of the signed tx.
		chain.Coordinator.UpdateTimeForChain(chain)

		// Increment acc sequence regardless of success or failure tx execution.
		defer func() {
			if err := chain.SenderAccount.SetSequence(chain.SenderAccount.GetSequence() + 1); err != nil {
				panic(err)
			}
		}()

		tx, err := signTxWithGranter(chain, fee, granter, msgs...)
		if err != nil {
			return nil, err
		}
		txBytes, err := chain.TxConfig.TxEncoder()(tx)
		if err != nil {
			return nil, err
		}

		resp, err := chain.App.GetBaseApp().FinalizeBlock(&abci.RequestFinalizeBlock{
			Height:             chain.App.GetBaseApp().LastBlockHeight() + 1,
			Time:               chain.CurrentHeader.GetTime(),
			NextValidatorsHash: chain.NextVals.Hash(),
			Txs:                [][]byte{txBytes},
		})
		if err != nil {
			return nil, err
		}

		// Equivalent of the unexported TestChain.commitBlock.
		if _, err := chain.App.Commit(); err != nil {
			return nil, err
		}
		chain.LastHeader = chain.CurrentTMClientHeader()
		chain.Vals = chain.NextVals
		chain.NextVals = ibctesting.ApplyValSetChanges(chain.TB, chain.Vals, resp.ValidatorUpdates)
		chain.Vals.IncrementProposerPriority(1)
		chain.CurrentHeader = cmtproto.Header{
			ChainID:            chain.ChainID,
			Height:             chain.App.LastBlockHeight() + 1,
			AppHash:            chain.App.LastCommitID().Hash,
			Time:               chain.CurrentHeader.Time,
			ValidatorsHash:     chain.Vals.Hash(),
			NextValidatorsHash: chain.NextVals.Hash(),
			ProposerAddress:    chain.Vals.Proposer.Address,
		}

		if len(resp.TxResults) != 1 {
			return nil, fmt.Errorf("expected 1 tx result, got %d", len(resp.TxResults))
		}
		txResult := resp.TxResults[0]
		if txResult.Code != 0 {
			return txResult, fmt.Errorf("%s/%d: %q", txResult.Codespace, txResult.Code, txResult.Log)
		}

		chain.Coordinator.IncrementTime()
		return txResult, nil
	}
}

// signTxWithGranter signs msgs with the chain's SenderAccount, declaring the
// given fee paid by granter. Mirrors simtestutil.GenSignedMockTx plus
// SetFeeGranter (which GenSignedMockTx does not expose).
func signTxWithGranter(chain *ibctesting.TestChain, fee sdk.Coins, granter sdk.AccAddress, msgs ...sdk.Msg) (sdk.Tx, error) {
	signMode, err := authsign.APISignModeToInternal(chain.TxConfig.SignModeHandler().DefaultMode())
	if err != nil {
		return nil, err
	}

	priv := chain.SenderPrivKey
	accSeq := chain.SenderAccount.GetSequence()

	txBuilder := chain.TxConfig.NewTxBuilder()
	if err := txBuilder.SetMsgs(msgs...); err != nil {
		return nil, err
	}
	txBuilder.SetFeeAmount(fee)
	txBuilder.SetGasLimit(simtestutil.DefaultGenTxGas)
	txBuilder.SetFeeGranter(granter)

	// 1st round: empty signature to populate the signer infos.
	if err := txBuilder.SetSignatures(signing.SignatureV2{
		PubKey:   priv.PubKey(),
		Data:     &signing.SingleSignatureData{SignMode: signMode},
		Sequence: accSeq,
	}); err != nil {
		return nil, err
	}

	// 2nd round: sign over the complete signer infos.
	signerData := authsign.SignerData{
		Address:       sdk.AccAddress(priv.PubKey().Address()).String(),
		ChainID:       chain.ChainID,
		AccountNumber: chain.SenderAccount.GetAccountNumber(),
		Sequence:      accSeq,
		PubKey:        priv.PubKey(),
	}
	signBytes, err := authsign.GetSignBytesAdapter(
		context.Background(), chain.TxConfig.SignModeHandler(), signMode, signerData, txBuilder.GetTx(),
	)
	if err != nil {
		return nil, err
	}
	sig, err := priv.Sign(signBytes)
	if err != nil {
		return nil, err
	}
	if err := txBuilder.SetSignatures(signing.SignatureV2{
		PubKey:   priv.PubKey(),
		Data:     &signing.SingleSignatureData{SignMode: signMode, Signature: sig},
		Sequence: accSeq,
	}); err != nil {
		return nil, err
	}

	return txBuilder.GetTx(), nil
}

// GetZeroneApp casts a TestChain's app to *ZeroneApp.
func GetZeroneApp(chain *ibctesting.TestChain) *zeroneapp.ZeroneApp {
	app, ok := chain.App.(*zeroneapp.ZeroneApp)
	if !ok {
		panic("chain app is not *ZeroneApp")
	}
	return app
}

// SetupRateLimit configures a rate limit on chainA via the keeper directly.
func (s *IBCTestSuite) SetupRateLimit(channelID, denom, maxSend, maxRecv string, windowBlocks uint64) {
	app := GetZeroneApp(s.chainA)
	ctx := s.chainA.GetContext()

	// Ensure rate limiting is enabled.
	app.IBCRateLimitKeeper.SetParams(ctx, &ibcratelimittypes.Params{Enabled: true})

	app.IBCRateLimitKeeper.SetRateLimit(ctx, &ibcratelimittypes.RateLimit{
		ChannelId:    channelID,
		Denom:        denom,
		MaxSend:      maxSend,
		MaxRecv:      maxRecv,
		WindowBlocks: windowBlocks,
		CurrentSend:  "0",
		CurrentRecv:  "0",
		WindowStart:  uint64(ctx.BlockHeight()),
	})
}
