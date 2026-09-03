package app

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"cosmossdk.io/log"
	sdkmath "cosmossdk.io/math"
	dbm "github.com/cosmos/cosmos-db"

	abci "github.com/cometbft/cometbft/abci/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/cosmos/cosmos-sdk/baseapp"
	clienttx "github.com/cosmos/cosmos-sdk/client/tx"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	ed25519 "github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkmempool "github.com/cosmos/cosmos-sdk/types/mempool"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/require"

	scheduletypes "github.com/zerone-chain/zerone/x/schedule/types"
)

const (
	proposalTestChainID = "zerone-proposal-test"
	proposalTestGas     = uint64(200_000)
)

type proposalTestFixture struct {
	app                 *ZeroneApp
	privateKey          cryptotypes.PrivKey
	sender              sdk.AccAddress
	secondaryPrivateKey cryptotypes.PrivKey
	secondarySender     sdk.AccAddress
	tertiaryPrivateKey  cryptotypes.PrivKey
	tertiarySender      sdk.AccAddress
	recipient           sdk.AccAddress
}

type proposalTestSigner struct {
	privateKey    cryptotypes.PrivKey
	address       sdk.AccAddress
	accountNumber uint64
	sequence      uint64
}

func proposalTestSenderKey() cryptotypes.PrivKey {
	return secp256k1.GenPrivKeyFromSecret([]byte("zerone proposal test sender"))
}

func newProposalTestFixture(
	t *testing.T,
	pool sdkmempool.Mempool,
	privateKey cryptotypes.PrivKey,
	maxGas int64,
) proposalTestFixture {
	return newProposalTestFixtureWithScheduleAdmission(t, pool, privateKey, maxGas, false)
}

func newProposalTestFixtureWithScheduleAdmission(
	t *testing.T,
	pool sdkmempool.Mempool,
	privateKey cryptotypes.PrivKey,
	maxGas int64,
	acceptNewSchedules bool,
	maxBytesOverride ...int64,
) proposalTestFixture {
	t.Helper()
	require.LessOrEqual(t, len(maxBytesOverride), 1)

	sender := sdk.AccAddress(privateKey.PubKey().Address())
	secondaryPrivateKey := secp256k1.GenPrivKeyFromSecret(
		[]byte("zerone proposal test secondary sender"),
	)
	secondarySender := sdk.AccAddress(secondaryPrivateKey.PubKey().Address())
	tertiaryPrivateKey := secp256k1.GenPrivKeyFromSecret(
		[]byte("zerone proposal test tertiary sender"),
	)
	tertiarySender := sdk.AccAddress(tertiaryPrivateKey.PubKey().Address())
	recipient := sdk.AccAddress(
		secp256k1.GenPrivKeyFromSecret([]byte("zerone proposal test recipient")).PubKey().Address(),
	)
	application := NewZeroneApp(
		log.NewNopLogger(),
		dbm.NewMemDB(),
		nil,
		false,
		simtestutil.NewAppOptionsWithFlagHome(t.TempDir()),
		baseapp.SetChainID(proposalTestChainID),
		baseapp.SetMempool(pool),
	)
	require.NoError(t, application.LoadLatestVersion())

	genesis := application.DefaultGenesis()
	var scheduleGenesis scheduletypes.GenesisState
	require.NoError(t, json.Unmarshal(genesis[scheduletypes.ModuleName], &scheduleGenesis))
	require.NotNil(t, scheduleGenesis.Params)
	scheduleGenesis.Params.AcceptNewSchedules = acceptNewSchedules
	scheduleGenesisJSON, err := json.Marshal(&scheduleGenesis)
	require.NoError(t, err)
	genesis[scheduletypes.ModuleName] = scheduleGenesisJSON
	account := authtypes.NewBaseAccount(sender, privateKey.PubKey(), 0, 0)
	secondaryAccount := authtypes.NewBaseAccount(
		secondarySender,
		secondaryPrivateKey.PubKey(),
		1,
		0,
	)
	tertiaryAccount := authtypes.NewBaseAccount(
		tertiarySender,
		tertiaryPrivateKey.PubKey(),
		2,
		0,
	)
	genesis[authtypes.ModuleName] = application.AppCodec().MustMarshalJSON(
		authtypes.NewGenesisState(
			authtypes.DefaultParams(),
			[]authtypes.GenesisAccount{account, secondaryAccount, tertiaryAccount},
		),
	)
	initialCoins := sdk.NewCoins(sdk.NewInt64Coin(BondDenom, 1_000_000_000))
	consensusKey := ed25519.GenPrivKeyFromSecret(
		[]byte("zerone proposal test consensus key"),
	).PubKey()
	consensusKeyAny, err := codectypes.NewAnyWithValue(consensusKey)
	require.NoError(t, err)
	bondAmount := sdk.DefaultPowerReduction
	validatorAddress := sdk.ValAddress(consensusKey.Address())
	validator := stakingtypes.Validator{
		OperatorAddress:   validatorAddress.String(),
		ConsensusPubkey:   consensusKeyAny,
		Status:            stakingtypes.Bonded,
		Tokens:            bondAmount,
		DelegatorShares:   sdkmath.LegacyOneDec(),
		Commission:        stakingtypes.NewCommission(sdkmath.LegacyZeroDec(), sdkmath.LegacyZeroDec(), sdkmath.LegacyZeroDec()),
		MinSelfDelegation: sdkmath.ZeroInt(),
	}
	stakingGenesis := stakingtypes.DefaultGenesisState()
	stakingGenesis.Validators = []stakingtypes.Validator{validator}
	stakingGenesis.Delegations = []stakingtypes.Delegation{stakingtypes.NewDelegation(
		sender.String(),
		validatorAddress.String(),
		sdkmath.LegacyOneDec(),
	)}
	genesis[stakingtypes.ModuleName] = application.AppCodec().MustMarshalJSON(stakingGenesis)

	bondedCoins := sdk.NewCoins(sdk.NewCoin(BondDenom, bondAmount))
	bankGenesis := banktypes.DefaultGenesisState()
	bankGenesis.Balances = []banktypes.Balance{
		{Address: sender.String(), Coins: initialCoins},
		{Address: secondarySender.String(), Coins: initialCoins},
		{Address: tertiarySender.String(), Coins: initialCoins},
		{
			Address: authtypes.NewModuleAddress(stakingtypes.BondedPoolName).String(),
			Coins:   bondedCoins,
		},
	}
	bankGenesis.Supply = initialCoins.Add(initialCoins...).Add(initialCoins...).Add(bondedCoins...)
	genesis[banktypes.ModuleName] = application.AppCodec().MustMarshalJSON(bankGenesis)

	genesisBytes, err := json.Marshal(genesis)
	require.NoError(t, err)
	consensusParams := *simtestutil.DefaultConsensusParams
	blockParams := *consensusParams.Block
	blockParams.MaxGas = maxGas
	if len(maxBytesOverride) == 1 {
		blockParams.MaxBytes = maxBytesOverride[0]
	}
	consensusParams.Block = &blockParams
	_, err = application.InitChain(&abci.RequestInitChain{
		ChainId:         proposalTestChainID,
		InitialHeight:   1,
		AppStateBytes:   genesisBytes,
		ConsensusParams: &consensusParams,
	})
	require.NoError(t, err)
	_, err = application.FinalizeBlock(&abci.RequestFinalizeBlock{
		Height: 1,
		Time:   time.Unix(1, 0).UTC(),
	})
	require.NoError(t, err)
	_, err = application.Commit()
	require.NoError(t, err)

	return proposalTestFixture{
		app:                 application,
		privateKey:          privateKey,
		sender:              sender,
		secondaryPrivateKey: secondaryPrivateKey,
		secondarySender:     secondarySender,
		tertiaryPrivateKey:  tertiaryPrivateKey,
		tertiarySender:      tertiarySender,
		recipient:           recipient,
	}
}

func (fixture proposalTestFixture) signedSend(
	t *testing.T,
	sequence uint64,
	memo string,
	omitPublicKey bool,
	corruptSignature bool,
) []byte {
	t.Helper()
	return fixture.signedMsg(
		t,
		banktypes.NewMsgSend(
			fixture.sender,
			fixture.recipient,
			sdk.NewCoins(sdk.NewInt64Coin(BondDenom, 1)),
		),
		sequence,
		memo,
		omitPublicKey,
		corruptSignature,
	)
}

func (fixture proposalTestFixture) signedMsg(
	t *testing.T,
	msg sdk.Msg,
	sequence uint64,
	memo string,
	omitPublicKey bool,
	corruptSignature bool,
) []byte {
	t.Helper()
	builder := fixture.app.TxConfig().NewTxBuilder()
	require.NoError(t, builder.SetMsgs(msg))
	builder.SetGasLimit(proposalTestGas)
	builder.SetFeeAmount(sdk.NewCoins(sdk.NewInt64Coin(BondDenom, int64(proposalTestGas))))
	builder.SetMemo(memo)

	placeholderPublicKey := fixture.privateKey.PubKey()
	if omitPublicKey {
		placeholderPublicKey = nil
	}
	require.NoError(t, builder.SetSignatures(signing.SignatureV2{
		PubKey: placeholderPublicKey,
		Data: &signing.SingleSignatureData{
			SignMode: signing.SignMode_SIGN_MODE_DIRECT,
		},
		Sequence: sequence,
	}))

	signature, err := clienttx.SignWithPrivKey(
		context.Background(),
		signing.SignMode_SIGN_MODE_DIRECT,
		authsigning.SignerData{
			Address:       fixture.sender.String(),
			ChainID:       proposalTestChainID,
			AccountNumber: 0,
			Sequence:      sequence,
			PubKey:        fixture.privateKey.PubKey(),
		},
		builder,
		fixture.privateKey,
		fixture.app.TxConfig(),
		sequence,
	)
	require.NoError(t, err)
	if omitPublicKey {
		signature.PubKey = nil
	}
	if corruptSignature {
		single, ok := signature.Data.(*signing.SingleSignatureData)
		require.True(t, ok)
		require.NotEmpty(t, single.Signature)
		single.Signature = append([]byte(nil), single.Signature...)
		single.Signature[0] ^= 0xff
	}
	require.NoError(t, builder.SetSignatures(signature))

	txBytes, err := fixture.app.TxConfig().TxEncoder()(builder.GetTx())
	require.NoError(t, err)
	return txBytes
}

func (fixture proposalTestFixture) signedMultiSignerTx(
	t *testing.T,
	msgs []sdk.Msg,
	signers []proposalTestSigner,
	feeAmount int64,
) []byte {
	t.Helper()

	builder := fixture.app.TxConfig().NewTxBuilder()
	require.NoError(t, builder.SetMsgs(msgs...))
	builder.SetGasLimit(proposalTestGas)
	builder.SetFeeAmount(sdk.NewCoins(sdk.NewInt64Coin(BondDenom, feeAmount)))

	placeholders := make([]signing.SignatureV2, len(signers))
	for i, signer := range signers {
		placeholders[i] = signing.SignatureV2{
			PubKey: signer.privateKey.PubKey(),
			Data: &signing.SingleSignatureData{
				SignMode: signing.SignMode_SIGN_MODE_DIRECT,
			},
			Sequence: signer.sequence,
		}
	}
	require.NoError(t, builder.SetSignatures(placeholders...))

	signatures := make([]signing.SignatureV2, len(signers))
	for i, signer := range signers {
		signature, err := clienttx.SignWithPrivKey(
			context.Background(),
			signing.SignMode_SIGN_MODE_DIRECT,
			authsigning.SignerData{
				Address:       signer.address.String(),
				ChainID:       proposalTestChainID,
				AccountNumber: signer.accountNumber,
				Sequence:      signer.sequence,
				PubKey:        signer.privateKey.PubKey(),
			},
			builder,
			signer.privateKey,
			fixture.app.TxConfig(),
			signer.sequence,
		)
		require.NoError(t, err)
		signatures[i] = signature
	}
	require.NoError(t, builder.SetSignatures(signatures...))

	txBytes, err := fixture.app.TxConfig().TxEncoder()(builder.GetTx())
	require.NoError(t, err)
	return txBytes
}

func TestApplicationMempoolOmittedPublicKeyFinalizesIdenticallyToNoOp(t *testing.T) {
	privateKey := proposalTestSenderKey()
	type result struct {
		appHash          []byte
		recipientBalance sdk.Coin
	}
	results := make(map[string]result)

	for name, pool := range map[string]sdkmempool.Mempool{
		"application": NewApplicationMempool(10),
		"noop":        sdkmempool.NoOpMempool{},
	} {
		t.Run(name, func(t *testing.T) {
			fixture := newProposalTestFixture(t, pool, privateKey, int64(BlockGasLimit))
			txBytes := fixture.signedSend(t, 0, "", true, false)
			decoded, err := fixture.app.TxDecode(txBytes)
			require.NoError(t, err)
			sigTx, ok := decoded.(authsigning.SigVerifiableTx)
			require.True(t, ok)
			signatures, err := sigTx.GetSignaturesV2()
			require.NoError(t, err)
			require.Len(t, signatures, 1)
			require.Nil(t, signatures[0].PubKey, "the regression transaction must omit SignerInfo.public_key")

			checked, err := fixture.app.CheckTx(&abci.RequestCheckTx{
				Tx:   txBytes,
				Type: abci.CheckTxType_New,
			})
			require.NoError(t, err)
			require.Zero(t, checked.Code, checked.Log)
			if name == "application" {
				require.Equal(t, 1, fixture.app.Mempool().CountTx())
			}

			requestTxs := [][]byte(nil)
			if name == "noop" {
				requestTxs = [][]byte{txBytes}
			}
			prepared, err := fixture.app.PrepareProposal(&abci.RequestPrepareProposal{
				Height:     2,
				MaxTxBytes: BlockMaxBytesLimit,
				Txs:        requestTxs,
			})
			require.NoError(t, err)
			require.Equal(t, [][]byte{txBytes}, prepared.Txs)

			processed, err := fixture.app.ProcessProposal(&abci.RequestProcessProposal{
				Height: 2,
				Txs:    prepared.Txs,
			})
			require.NoError(t, err)
			require.Equal(t, abci.ResponseProcessProposal_ACCEPT, processed.Status)

			finalized, err := fixture.app.FinalizeBlock(&abci.RequestFinalizeBlock{
				Height: 2,
				Time:   time.Unix(2, 0).UTC(),
				Txs:    prepared.Txs,
			})
			require.NoError(t, err)
			require.Len(t, finalized.TxResults, 1)
			require.Zero(t, finalized.TxResults[0].Code, finalized.TxResults[0].Log)
			_, err = fixture.app.Commit()
			require.NoError(t, err)

			ctx := fixture.app.NewUncachedContext(false, cmtproto.Header{Height: 2})
			results[name] = result{
				appHash:          append([]byte(nil), fixture.app.LastCommitID().Hash...),
				recipientBalance: fixture.app.BankKeeper.GetBalance(ctx, fixture.recipient, BondDenom),
			}
			if name == "application" {
				require.Zero(t, fixture.app.Mempool().CountTx())
			}
		})
	}

	require.Equal(t, sdk.NewInt64Coin(BondDenom, 1), results["application"].recipientBalance)
	require.Equal(t, results["noop"].recipientBalance, results["application"].recipientBalance)
	require.Equal(t, results["noop"].appHash, results["application"].appHash)
}

func TestApplicationMempoolCapacitySkipPreservesNonceSuccessor(t *testing.T) {
	fixture := newProposalTestFixture(
		t,
		NewApplicationMempool(10),
		proposalTestSenderKey(),
		int64(BlockGasLimit),
	)
	largeTx := fixture.signedSend(t, 0, strings.Repeat("x", 128), false, false)
	smallTx := fixture.signedSend(t, 1, "", false, false)
	require.Greater(t, proposalTxProtoSize(largeTx), proposalTxProtoSize(smallTx))

	for _, txBytes := range [][]byte{largeTx, smallTx} {
		checked, err := fixture.app.CheckTx(&abci.RequestCheckTx{
			Tx:   txBytes,
			Type: abci.CheckTxType_New,
		})
		require.NoError(t, err)
		require.Zero(t, checked.Code, checked.Log)
	}
	require.Equal(t, 2, fixture.app.Mempool().CountTx())

	prepared, err := fixture.app.PrepareProposal(&abci.RequestPrepareProposal{
		Height:     2,
		MaxTxBytes: proposalTxProtoSize(smallTx),
	})
	require.NoError(t, err)
	require.Empty(t, prepared.Txs)
	require.Equal(t, 2, fixture.app.Mempool().CountTx(),
		"a capacity-dependent nonce failure must not evict a valid successor")

	prepared, err = fixture.app.PrepareProposal(&abci.RequestPrepareProposal{
		Height:     2,
		MaxTxBytes: BlockMaxBytesLimit,
	})
	require.NoError(t, err)
	require.Equal(t, [][]byte{largeTx, smallTx}, prepared.Txs)
	require.Equal(t, 2, fixture.app.Mempool().CountTx())
}

func TestApplicationMempoolNonceGapPreservesFutureTransaction(t *testing.T) {
	fixture := newProposalTestFixture(
		t,
		NewApplicationMempool(10),
		proposalTestSenderKey(),
		int64(BlockGasLimit),
	)
	sequenceZero := fixture.signedSend(t, 0, "sequence-zero", false, false)
	sequenceOne := fixture.signedSend(t, 1, "sequence-one", false, false)

	for _, txBytes := range [][]byte{sequenceZero, sequenceOne} {
		checked, err := fixture.app.CheckTx(&abci.RequestCheckTx{
			Tx:   txBytes,
			Type: abci.CheckTxType_New,
		})
		require.NoError(t, err)
		require.Zero(t, checked.Code, checked.Log)
	}
	require.Equal(t, 2, fixture.app.Mempool().CountTx())

	decodedZero, err := fixture.app.TxDecode(sequenceZero)
	require.NoError(t, err)
	require.NoError(t, fixture.app.Mempool().Remove(decodedZero))
	require.Equal(t, 1, fixture.app.Mempool().CountTx())

	prepared, err := fixture.app.PrepareProposal(&abci.RequestPrepareProposal{
		Height:     2,
		MaxTxBytes: BlockMaxBytesLimit,
	})
	require.NoError(t, err)
	require.Empty(t, prepared.Txs)
	require.Equal(t, 1, fixture.app.Mempool().CountTx(),
		"a future sequence must remain pooled while its predecessor is absent")

	require.NoError(t, fixture.app.Mempool().Insert(sdk.Context{}.WithPriority(1), decodedZero))
	prepared, err = fixture.app.PrepareProposal(&abci.RequestPrepareProposal{
		Height:     2,
		MaxTxBytes: BlockMaxBytesLimit,
	})
	require.NoError(t, err)
	require.Equal(t, [][]byte{sequenceZero, sequenceOne}, prepared.Txs)
	require.Equal(t, 2, fixture.app.Mempool().CountTx())
}

func TestApplicationMempoolMultiSignerFutureDependencyMakesProgress(t *testing.T) {
	fixture := newProposalTestFixture(
		t,
		NewApplicationMempool(10),
		proposalTestSenderKey(),
		int64(BlockGasLimit),
	)

	// The predecessor is indexed under A and advances A0+B0. The successor is
	// indexed under C and needs C0+B1. Give the successor higher priority so the
	// SDK's first-signer priority index presents it before the predecessor.
	predecessor := fixture.signedMultiSignerTx(
		t,
		[]sdk.Msg{
			banktypes.NewMsgSend(
				fixture.sender,
				fixture.recipient,
				sdk.NewCoins(sdk.NewInt64Coin(BondDenom, 1)),
			),
			banktypes.NewMsgSend(
				fixture.secondarySender,
				fixture.recipient,
				sdk.NewCoins(sdk.NewInt64Coin(BondDenom, 1)),
			),
		},
		[]proposalTestSigner{
			{privateKey: fixture.privateKey, address: fixture.sender, accountNumber: 0, sequence: 0},
			{privateKey: fixture.secondaryPrivateKey, address: fixture.secondarySender, accountNumber: 1, sequence: 0},
		},
		int64(proposalTestGas),
	)
	successor := fixture.signedMultiSignerTx(
		t,
		[]sdk.Msg{
			banktypes.NewMsgSend(
				fixture.tertiarySender,
				fixture.recipient,
				sdk.NewCoins(sdk.NewInt64Coin(BondDenom, 1)),
			),
			banktypes.NewMsgSend(
				fixture.secondarySender,
				fixture.recipient,
				sdk.NewCoins(sdk.NewInt64Coin(BondDenom, 1)),
			),
		},
		[]proposalTestSigner{
			{privateKey: fixture.tertiaryPrivateKey, address: fixture.tertiarySender, accountNumber: 2, sequence: 0},
			{privateKey: fixture.secondaryPrivateKey, address: fixture.secondarySender, accountNumber: 1, sequence: 1},
		},
		int64(proposalTestGas*2),
	)

	// CheckTx order establishes the valid B0 -> B1 dependency while the fee
	// ratio makes the successor the first proposal candidate.
	for _, txBytes := range [][]byte{predecessor, successor} {
		checked, err := fixture.app.CheckTx(&abci.RequestCheckTx{
			Tx:   txBytes,
			Type: abci.CheckTxType_New,
		})
		require.NoError(t, err)
		require.Zero(t, checked.Code, checked.Log)
	}
	require.Equal(t, 2, fixture.app.Mempool().CountTx())
	var firstCandidate []byte
	sdkmempool.SelectBy(sdk.Context{}, fixture.app.Mempool(), nil, func(tx sdk.Tx) bool {
		var err error
		firstCandidate, err = fixture.app.TxEncode(tx)
		require.NoError(t, err)
		return false
	})
	require.Equal(t, successor, firstCandidate,
		"the regression requires the secondary-signature successor to be visited first")

	prepared, err := fixture.app.PrepareProposal(&abci.RequestPrepareProposal{
		Height:     2,
		MaxTxBytes: BlockMaxBytesLimit,
	})
	require.NoError(t, err)
	require.Equal(t, [][]byte{predecessor}, prepared.Txs,
		"a future secondary-signer dependency must not block its predecessor")

	processed, err := fixture.app.ProcessProposal(&abci.RequestProcessProposal{
		Height: 2,
		Txs:    prepared.Txs,
	})
	require.NoError(t, err)
	require.Equal(t, abci.ResponseProcessProposal_ACCEPT, processed.Status)
	finalized, err := fixture.app.FinalizeBlock(&abci.RequestFinalizeBlock{
		Height: 2,
		Time:   time.Unix(2, 0).UTC(),
		Txs:    prepared.Txs,
	})
	require.NoError(t, err)
	require.Len(t, finalized.TxResults, 1)
	require.Zero(t, finalized.TxResults[0].Code, finalized.TxResults[0].Log)
	_, err = fixture.app.Commit()
	require.NoError(t, err)
	require.Equal(t, 1, fixture.app.Mempool().CountTx())

	prepared, err = fixture.app.PrepareProposal(&abci.RequestPrepareProposal{
		Height:     3,
		MaxTxBytes: BlockMaxBytesLimit,
	})
	require.NoError(t, err)
	require.Equal(t, [][]byte{successor}, prepared.Txs,
		"the preserved successor must become selectable after its predecessor commits")

	processed, err = fixture.app.ProcessProposal(&abci.RequestProcessProposal{
		Height: 3,
		Txs:    prepared.Txs,
	})
	require.NoError(t, err)
	require.Equal(t, abci.ResponseProcessProposal_ACCEPT, processed.Status)
}

func TestProposalAdmissionHardCountAndZeroGasBoundaries(t *testing.T) {
	fixture := newProposalTestFixture(
		t,
		NewApplicationMempool(10),
		proposalTestSenderKey(),
		0,
	)
	txBytes := fixture.signedSend(t, 0, "", false, false)
	decoded, err := fixture.app.TxDecode(txBytes)
	require.NoError(t, err)
	require.NoError(t, fixture.app.Mempool().Insert(sdk.Context{}.WithPriority(1), decoded))

	prepared, err := fixture.app.PrepareProposal(&abci.RequestPrepareProposal{
		Height:     2,
		MaxTxBytes: BlockMaxBytesLimit,
	})
	require.NoError(t, err)
	require.Empty(t, prepared.Txs, "consensus max_gas=0 must admit no regular transaction")

	processed, err := fixture.app.ProcessProposal(&abci.RequestProcessProposal{
		Height: 2,
		Txs:    [][]byte{txBytes},
	})
	require.NoError(t, err)
	require.Equal(t, abci.ResponseProcessProposal_REJECT, processed.Status)

	tooMany := make([][]byte, MaxProposalTxCount+1)
	processed, err = fixture.app.ProcessProposal(&abci.RequestProcessProposal{
		Height: 2,
		Txs:    tooMany,
	})
	require.NoError(t, err)
	require.Equal(t, abci.ResponseProcessProposal_REJECT, processed.Status)
	require.LessOrEqual(t, uint64(MaxProposalRegularTxCount)*MinGasLimit, BlockGasLimit)
	require.Greater(t, uint64(MaxProposalRegularTxCount+1)*MinGasLimit, BlockGasLimit)
	require.Equal(t, MaxProposalRegularTxCount+1, MaxProposalTxCount)
}

func TestLowerConsensusMaxGasRejectsBeforeApplicationPoolInsertion(t *testing.T) {
	const loweredConsensusMaxGas = int64(100_000)
	fixture := newProposalTestFixture(
		t,
		NewApplicationMempool(10),
		proposalTestSenderKey(),
		loweredConsensusMaxGas,
	)
	txBytes := fixture.signedSend(t, 0, "above-lowered-consensus-max-gas", false, false)
	require.Greater(t, proposalTestGas, uint64(loweredConsensusMaxGas))

	checked, err := fixture.app.CheckTx(&abci.RequestCheckTx{
		Tx:   txBytes,
		Type: abci.CheckTxType_New,
	})
	require.NoError(t, err)
	require.NotZero(t, checked.Code)
	require.Contains(t, checked.Log, "block max gas")
	require.Zero(t, fixture.app.Mempool().CountTx(),
		"a transaction CometBFT must reject may not survive as an application-pool ghost")

	prepared, err := fixture.app.PrepareProposal(&abci.RequestPrepareProposal{
		Height:     2,
		MaxTxBytes: BlockMaxBytesLimit,
	})
	require.NoError(t, err)
	require.Empty(t, prepared.Txs)

	processed, err := fixture.app.ProcessProposal(&abci.RequestProcessProposal{
		Height: 2,
		Txs:    [][]byte{txBytes},
	})
	require.NoError(t, err)
	require.Equal(t, abci.ResponseProcessProposal_REJECT, processed.Status)

	finalized, err := fixture.app.FinalizeBlock(&abci.RequestFinalizeBlock{
		Height: 2,
		Time:   time.Unix(2, 0).UTC(),
		Txs:    [][]byte{txBytes},
	})
	require.NoError(t, err)
	require.Len(t, finalized.TxResults, 1)
	require.NotZero(t, finalized.TxResults[0].Code)
	require.Contains(t, finalized.TxResults[0].Log, "block max gas")
}

func TestLowerConsensusMaxBytesRejectsBeforeApplicationPoolInsertion(t *testing.T) {
	const loweredConsensusMaxBytes = int64(10 * 1024)
	fixture := newProposalTestFixtureWithScheduleAdmission(
		t,
		NewApplicationMempool(10),
		proposalTestSenderKey(),
		int64(BlockGasLimit),
		false,
		loweredConsensusMaxBytes,
	)
	txBytes := fixture.signedSend(
		t,
		0,
		strings.Repeat("x", int(loweredConsensusMaxBytes)),
		false,
		false,
	)
	require.Greater(t, proposalTxProtoSize(txBytes), loweredConsensusMaxBytes)
	require.Less(t, len(txBytes), 256*1024,
		"control transaction must remain below the frozen local Comet max_tx_bytes")

	checked, err := fixture.app.CheckTx(&abci.RequestCheckTx{
		Tx:   txBytes,
		Type: abci.CheckTxType_New,
	})
	require.NoError(t, err)
	require.NotZero(t, checked.Code)
	require.Contains(t, checked.Log, "effective block byte limit")
	require.Zero(t, fixture.app.Mempool().CountTx(),
		"a permanently impossible transaction may not survive as an application-pool ghost")

	processed, err := fixture.app.ProcessProposal(&abci.RequestProcessProposal{
		Height: 2,
		Txs:    [][]byte{txBytes},
	})
	require.NoError(t, err)
	require.Equal(t, abci.ResponseProcessProposal_REJECT, processed.Status)

	finalized, err := fixture.app.FinalizeBlock(&abci.RequestFinalizeBlock{
		Height: 2,
		Time:   time.Unix(2, 0).UTC(),
		Txs:    [][]byte{txBytes},
	})
	require.NoError(t, err)
	require.Len(t, finalized.TxResults, 1)
	require.NotZero(t, finalized.TxResults[0].Code)
	require.Contains(t, finalized.TxResults[0].Log, "effective block byte limit")
}

func TestProcessProposalRejectsCorruptedSignatureWithValidControl(t *testing.T) {
	fixture := newProposalTestFixture(
		t,
		NewApplicationMempool(10),
		proposalTestSenderKey(),
		int64(BlockGasLimit),
	)
	validTx := fixture.signedSend(t, 0, "signature-isolated", false, false)
	corruptTx := fixture.signedSend(t, 0, "signature-isolated", false, true)
	var validRaw, corruptRaw txtypes.TxRaw
	require.NoError(t, fixture.app.AppCodec().Unmarshal(validTx, &validRaw))
	require.NoError(t, fixture.app.AppCodec().Unmarshal(corruptTx, &corruptRaw))
	require.Equal(t, validRaw.BodyBytes, corruptRaw.BodyBytes)
	require.Equal(t, validRaw.AuthInfoBytes, corruptRaw.AuthInfoBytes)
	require.Len(t, validRaw.Signatures, 1)
	require.Len(t, corruptRaw.Signatures, 1)
	require.Equal(t, validRaw.Signatures[0][1:], corruptRaw.Signatures[0][1:])
	require.NotEqual(t, validRaw.Signatures[0][0], corruptRaw.Signatures[0][0],
		"only one signature byte should differ between the valid control and corrupt transaction")

	processed, err := fixture.app.ProcessProposal(&abci.RequestProcessProposal{
		Height: 2,
		Txs:    [][]byte{validTx},
	})
	require.NoError(t, err)
	require.Equal(t, abci.ResponseProcessProposal_ACCEPT, processed.Status,
		"the control transaction must prove gas, fees, balance, sequence, and policy are valid")

	processed, err = fixture.app.ProcessProposal(&abci.RequestProcessProposal{
		Height: 2,
		Txs:    [][]byte{corruptTx},
	})
	require.NoError(t, err)
	require.Equal(t, abci.ResponseProcessProposal_REJECT, processed.Status)
}

func TestCommittedScheduleEscrowsAndExecutesThroughFullApplication(t *testing.T) {
	fixture := newProposalTestFixtureWithScheduleAdmission(
		t,
		NewApplicationMempool(10),
		proposalTestSenderKey(),
		int64(BlockGasLimit),
		true,
	)
	const (
		firstExecutionHeight = uint64(4)
		transferAmount       = int64(50)
	)
	create := &scheduletypes.MsgCreateSchedule{
		Creator:                fixture.sender.String(),
		Recipient:              fixture.recipient.String(),
		AmountPerExecutionUzrn: sdkmath.NewInt(transferAmount).String(),
		FirstExecutionHeight:   firstExecutionHeight,
		IntervalBlocks:         0,
		ExecutionCount:         1,
	}
	txBytes := fixture.signedMsg(t, create, 0, "", true, false)

	checked, err := fixture.app.CheckTx(&abci.RequestCheckTx{
		Tx:   txBytes,
		Type: abci.CheckTxType_New,
	})
	require.NoError(t, err)
	require.Zero(t, checked.Code, checked.Log)
	require.Equal(t, 1, fixture.app.Mempool().CountTx())

	prepared, err := fixture.app.PrepareProposal(&abci.RequestPrepareProposal{
		Height:     2,
		MaxTxBytes: BlockMaxBytesLimit,
	})
	require.NoError(t, err)
	require.Equal(t, [][]byte{txBytes}, prepared.Txs)
	processed, err := fixture.app.ProcessProposal(&abci.RequestProcessProposal{
		Height: 2,
		Txs:    prepared.Txs,
	})
	require.NoError(t, err)
	require.Equal(t, abci.ResponseProcessProposal_ACCEPT, processed.Status)
	finalized, err := fixture.app.FinalizeBlock(&abci.RequestFinalizeBlock{
		Height: 2,
		Time:   time.Unix(2, 0).UTC(),
		Txs:    prepared.Txs,
	})
	require.NoError(t, err)
	require.Len(t, finalized.TxResults, 1)
	require.Zero(t, finalized.TxResults[0].Code, finalized.TxResults[0].Log)
	_, err = fixture.app.Commit()
	require.NoError(t, err)

	scheduleID := scheduletypes.FormatScheduleID(1)
	ctx := fixture.app.NewUncachedContext(false, cmtproto.Header{Height: 2})
	schedule, found := fixture.app.ScheduleKeeper.GetSchedule(ctx, scheduleID)
	require.True(t, found)
	require.Equal(t, scheduletypes.ScheduleStatus_SCHEDULE_STATUS_ACTIVE, schedule.Status)
	require.Equal(t, firstExecutionHeight, schedule.NextExecutionHeight)
	require.Equal(t, "100050", fixture.app.ScheduleKeeper.TotalEscrow(ctx).String())
	require.Equal(t, sdk.NewInt64Coin(BondDenom, 100_050), fixture.app.BankKeeper.GetBalance(
		ctx,
		authtypes.NewModuleAddress(scheduletypes.ModuleName),
		BondDenom,
	))

	for height := int64(3); height <= int64(firstExecutionHeight); height++ {
		finalized, err = fixture.app.FinalizeBlock(&abci.RequestFinalizeBlock{
			Height: height,
			Time:   time.Unix(height, 0).UTC(),
		})
		require.NoError(t, err)
		_, err = fixture.app.Commit()
		require.NoError(t, err)
	}

	ctx = fixture.app.NewUncachedContext(false, cmtproto.Header{Height: int64(firstExecutionHeight)})
	schedule, found = fixture.app.ScheduleKeeper.GetSchedule(ctx, scheduleID)
	require.True(t, found)
	require.Equal(t, scheduletypes.ScheduleStatus_SCHEDULE_STATUS_COMPLETED, schedule.Status)
	require.Equal(t, uint32(1), schedule.ExecutionCount)
	require.Zero(t, schedule.RemainingExecutions)
	require.Equal(t, firstExecutionHeight, schedule.LastExecutionHeight)
	require.Equal(t, "0", fixture.app.ScheduleKeeper.TotalEscrow(ctx).String())
	require.Equal(t, sdk.NewInt64Coin(BondDenom, transferAmount), fixture.app.BankKeeper.GetBalance(
		ctx,
		fixture.recipient,
		BondDenom,
	))
	require.True(t, fixture.app.BankKeeper.GetBalance(
		ctx,
		authtypes.NewModuleAddress(scheduletypes.ModuleName),
		BondDenom,
	).IsZero())
	receipt, found := fixture.app.ScheduleKeeper.GetReceipt(ctx, scheduleID, 1)
	require.True(t, found)
	require.Equal(t, scheduletypes.ExecutionOutcome_EXECUTION_OUTCOME_SUCCEEDED, receipt.Outcome)
	require.Equal(t, firstExecutionHeight, receipt.DueHeight)
	require.Equal(t, firstExecutionHeight, receipt.ExecutedHeight)
	require.Equal(t, sdkmath.NewInt(transferAmount).String(), receipt.AmountUzrn)
}
