package app

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"path/filepath"
	"testing"
	"time"

	"cosmossdk.io/log"
	sdkmath "cosmossdk.io/math"
	abci "github.com/cometbft/cometbft/abci/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/baseapp"
	clienttx "github.com/cosmos/cosmos-sdk/client/tx"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkmempool "github.com/cosmos/cosmos-sdk/types/mempool"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/require"

	zeroneauthtypes "github.com/zerone-chain/zerone/x/auth/types"
	emergencytypes "github.com/zerone-chain/zerone/x/emergency/types"
	scheduletypes "github.com/zerone-chain/zerone/x/schedule/types"
)

const (
	schedulerTestChainID = "zerone-scheduler-local-test"
	schedulerTestGas     = uint64(200_000)
)

// These deterministic keys belong only to fresh, local test genesis. The normal
// constructor and its default NoOp mempool are intentional: protocol work has no
// dependency on a proposer's optional local transaction storage.
type schedulerTestFixture struct {
	app          *ZeroneApp
	home         string
	disk         bool
	privateKey   cryptotypes.PrivKey
	sender       sdk.AccAddress
	recipient    sdk.AccAddress
	genesisBytes []byte
}

type schedulerTestOptions struct {
	disk          bool
	acceptNew     bool // ONLY the explicitly enabled signed-creation fixtures.
	schedules     int
	due           uint64
	occurrences   uint32
	interval      uint64
	dueCap        uint32
	sendDisabled  bool
	releaseHeight uint64
	halted        bool
	guardian      bool // Fixture-only one-member council for signed emergency transitions.
	params        *scheduletypes.Params
	genesisBytes  []byte
	initialHeight int64 // Nonzero only for continuation-genesis fixtures.
}

func newSchedulerTestFixture(t *testing.T, options schedulerTestOptions) *schedulerTestFixture {
	t.Helper()
	key := secp256k1.GenPrivKeyFromSecret([]byte("scheduler local sender"))
	f := &schedulerTestFixture{
		home: t.TempDir(), disk: options.disk, privateKey: key,
		sender:    sdk.AccAddress(key.PubKey().Address()),
		recipient: sdk.AccAddress(secp256k1.GenPrivKeyFromSecret([]byte("scheduler local recipient")).PubKey().Address()),
	}
	f.open(t)
	t.Cleanup(func() {
		if f.app != nil {
			require.NoError(t, f.app.Close()) // BaseApp owns and closes its DB.
			f.app = nil
		}
	})
	genesisBytes := options.genesisBytes
	if genesisBytes == nil {
		genesisBytes = f.genesis(t, options)
	}
	f.genesisBytes = append([]byte(nil), genesisBytes...)
	consensusParams := *simtestutil.DefaultConsensusParams
	blockParams := *consensusParams.Block
	blockParams.MaxGas = int64(BlockGasLimit)
	consensusParams.Block = &blockParams
	initialHeight := options.initialHeight
	if initialHeight == 0 {
		initialHeight = 1
	}
	_, err := f.app.InitChain(&abci.RequestInitChain{
		ChainId: schedulerTestChainID, InitialHeight: initialHeight, Time: schedulerTestTime(initialHeight - 1),
		AppStateBytes: genesisBytes, ConsensusParams: &consensusParams,
	})
	require.NoError(t, err)
	f.block(t, initialHeight)
	return f
}

func (f *schedulerTestFixture) open(t *testing.T) {
	t.Helper()
	var db dbm.DB
	if f.disk {
		var err error
		db, err = dbm.NewDB("application", dbm.GoLevelDBBackend, filepath.Join(f.home, "data"))
		require.NoError(t, err)
	} else {
		db = dbm.NewMemDB()
	}
	f.app = NewZeroneApp(log.NewNopLogger(), db, nil, false,
		simtestutil.NewAppOptionsWithFlagHome(f.home), baseapp.SetChainID(schedulerTestChainID))
	require.IsType(t, sdkmempool.NoOpMempool{}, f.app.Mempool())
	require.False(t, f.app.activationPreflightReadOnly)
	require.NoError(t, f.app.LoadLatestVersion())
	// The runtime startup guard, not the offline activation-preflight bypass.
	require.NoError(t, f.app.ValidateSDK053IBC10StartupCoordination())
}

func (f *schedulerTestFixture) reopen(t *testing.T) {
	t.Helper()
	require.True(t, f.disk)
	require.NoError(t, f.app.Close())
	f.app = nil
	f.open(t) // New DB handle + normal app; NEVER InitChain on a restart.
}

func (f *schedulerTestFixture) genesis(t *testing.T, options schedulerTestOptions) []byte {
	t.Helper()
	genesis := f.app.DefaultGenesis()
	var scheduleGenesis scheduletypes.GenesisState
	require.NoError(t, json.Unmarshal(genesis[scheduletypes.ModuleName], &scheduleGenesis))
	require.False(t, scheduleGenesis.Params.AcceptNewSchedules, "normal application defaults must remain closed")
	if options.params != nil {
		// Copy through JSON rather than copying protobuf's internal mutex.
		paramsJSON, err := json.Marshal(options.params)
		require.NoError(t, err)
		require.NoError(t, json.Unmarshal(paramsJSON, scheduleGenesis.Params))
	}
	scheduleGenesis.Params.AcceptNewSchedules = options.acceptNew
	if options.dueCap != 0 {
		scheduleGenesis.Params.MaxDueRecordsPerBlock = options.dueCap
	}
	liability := new(big.Int)
	for i := 1; i <= options.schedules; i++ {
		// Explicitly populated, fully backed TEST genesis. These committed terms
		// need not satisfy today's admission limits (parameters are prospective).
		count := options.occurrences
		if count == 0 {
			count = 1
		}
		principal := sdkmath.NewInt(50).MulRaw(int64(count)).String()
		fees := sdkmath.NewInt(100_000).MulRaw(int64(count)).String()
		s := &scheduletypes.Schedule{
			Id: scheduletypes.FormatScheduleID(uint64(i)), Creator: f.sender.String(),
			Recipient: f.recipient.String(), Revision: 1,
			Status:                 scheduletypes.ScheduleStatus_SCHEDULE_STATUS_ACTIVE,
			AmountPerExecutionUzrn: "50", ExecutionFeeUzrn: "100000",
			NextExecutionHeight: options.due, IntervalBlocks: options.interval,
			RemainingExecutions: count, PrincipalRemainingUzrn: principal, FeeRemainingUzrn: fees,
			CreatedHeight: 1, UpdatedHeight: 1,
		}
		scheduleGenesis.Schedules = append(scheduleGenesis.Schedules, s)
		liability.Add(liability, big.NewInt(100_050*int64(count)))
	}
	scheduleGenesis.NextScheduleId = uint64(options.schedules + 1)
	scheduleGenesis.TotalEscrowUzrn = liability.String()
	require.NoError(t, scheduleGenesis.ValidateForChainID(schedulerTestChainID))
	var err error
	genesis[scheduletypes.ModuleName], err = json.Marshal(&scheduleGenesis)
	require.NoError(t, err)
	account := authtypes.NewBaseAccount(f.sender, f.privateKey.PubKey(), 0, 0)
	genesis[authtypes.ModuleName] = f.app.AppCodec().MustMarshalJSON(authtypes.NewGenesisState(
		authtypes.DefaultParams(), []authtypes.GenesisAccount{account}))
	consensusKey := ed25519.GenPrivKeyFromSecret([]byte("scheduler local consensus")).PubKey()
	consensusKeyAny, err := codectypes.NewAnyWithValue(consensusKey)
	require.NoError(t, err)
	validatorAddress := sdk.ValAddress(consensusKey.Address())
	stakingGenesis := stakingtypes.DefaultGenesisState()
	stakingGenesis.Validators = []stakingtypes.Validator{{
		OperatorAddress: validatorAddress.String(), ConsensusPubkey: consensusKeyAny,
		Status: stakingtypes.Bonded, Tokens: sdk.DefaultPowerReduction,
		DelegatorShares: sdkmath.LegacyOneDec(), MinSelfDelegation: sdkmath.ZeroInt(),
		Commission: stakingtypes.NewCommission(sdkmath.LegacyZeroDec(), sdkmath.LegacyZeroDec(), sdkmath.LegacyZeroDec()),
	}}
	stakingGenesis.Delegations = []stakingtypes.Delegation{stakingtypes.NewDelegation(
		f.sender.String(), validatorAddress.String(), sdkmath.LegacyOneDec())}
	genesis[stakingtypes.ModuleName] = f.app.AppCodec().MustMarshalJSON(stakingGenesis)
	bankGenesis := banktypes.DefaultGenesisState()
	bankGenesis.Balances = []banktypes.Balance{
		{Address: f.sender.String(), Coins: sdk.NewCoins(sdk.NewInt64Coin(BondDenom, 1_000_000_000))},
		{Address: authtypes.NewModuleAddress(stakingtypes.BondedPoolName).String(), Coins: sdk.NewCoins(sdk.NewCoin(BondDenom, sdk.DefaultPowerReduction))},
	}
	if liability.Sign() > 0 {
		bankGenesis.Balances = append(bankGenesis.Balances, banktypes.Balance{
			Address: authtypes.NewModuleAddress(scheduletypes.ModuleName).String(),
			Coins:   sdk.NewCoins(sdk.NewCoin(BondDenom, sdkmath.NewIntFromBigInt(liability))),
		})
	}
	for _, balance := range bankGenesis.Balances {
		bankGenesis.Supply = bankGenesis.Supply.Add(balance.Coins...)
	}
	if options.sendDisabled {
		bankGenesis.SendEnabled = []banktypes.SendEnabled{{Denom: BondDenom, Enabled: false}}
	}
	genesis[banktypes.ModuleName] = f.app.AppCodec().MustMarshalJSON(bankGenesis)
	if options.releaseHeight != 0 || options.halted || options.guardian {
		var emergencyGenesis emergencytypes.GenesisState
		require.NoError(t, json.Unmarshal(genesis[emergencytypes.ModuleName], &emergencyGenesis))
		emergencyGenesis.QuarantineReleaseBlock = options.releaseHeight
		if options.guardian {
			emergencyGenesis.Params.GenesisCouncil = []string{f.sender.String()}
			emergencyGenesis.Params.CouncilExpiryBlock = 100
			emergencyGenesis.Params.MinDistinctVoters = 1
			emergencyGenesis.Params.CooldownBlocks = 0
			emergencyGenesis.Params.MaxProposalsPerEpoch = 10
			emergencyGenesis.Params.MaxProposalsPerGuardianPerEpoch = 10
		}
		if options.halted {
			require.Zero(t, options.releaseHeight)
			emergencyGenesis.Status = string(emergencytypes.StatusHalted)
			emergencyGenesis.ActiveHaltCeremonyId = "legacy-genesis-quarantine"
			emergencyGenesis.HaltStartBlock = 1
			emergencyGenesis.Params.MaxHaltDurationBlocks = 2
		}
		require.NoError(t, emergencyGenesis.Validate())
		genesis[emergencytypes.ModuleName], err = json.Marshal(&emergencyGenesis)
		require.NoError(t, err)
	}
	bz, err := json.Marshal(genesis)
	require.NoError(t, err)
	return bz
}

func schedulerTestTime(height int64) time.Time {
	return time.Unix(1_800_000_000+height, 0).UTC()
}

func schedulerTestRequest(height int64, txs [][]byte) *abci.RequestFinalizeBlock {
	hash := sha256.Sum256([]byte(fmt.Sprintf("scheduler-test-block/%d", height)))
	return &abci.RequestFinalizeBlock{Height: height, Time: schedulerTestTime(height), Hash: hash[:], Txs: txs}
}

func (f *schedulerTestFixture) ctx() sdk.Context {
	return f.app.NewUncachedContext(false, cmtproto.Header{
		Height: f.app.LastBlockHeight(), ChainID: schedulerTestChainID, Time: schedulerTestTime(f.app.LastBlockHeight()),
	})
}

func (f *schedulerTestFixture) signedMsg(t *testing.T, msg sdk.Msg, sequence uint64) []byte {
	t.Helper()
	builder := f.app.TxConfig().NewTxBuilder()
	require.NoError(t, builder.SetMsgs(msg))
	builder.SetGasLimit(schedulerTestGas)
	builder.SetFeeAmount(sdk.NewCoins(sdk.NewInt64Coin(BondDenom, int64(schedulerTestGas))))
	require.NoError(t, builder.SetSignatures(signing.SignatureV2{
		PubKey: f.privateKey.PubKey(), Sequence: sequence,
		Data: &signing.SingleSignatureData{SignMode: signing.SignMode_SIGN_MODE_DIRECT},
	}))
	signature, err := clienttx.SignWithPrivKey(context.Background(), signing.SignMode_SIGN_MODE_DIRECT,
		authsigning.SignerData{Address: f.sender.String(), ChainID: schedulerTestChainID,
			AccountNumber: 0, Sequence: sequence, PubKey: f.privateKey.PubKey()},
		builder, f.privateKey, f.app.TxConfig(), sequence)
	require.NoError(t, err)
	require.NoError(t, builder.SetSignatures(signature))
	bz, err := f.app.TxConfig().TxEncoder()(builder.GetTx())
	require.NoError(t, err)
	return bz
}

func (f *schedulerTestFixture) prepare(t *testing.T, height int64, txs [][]byte) [][]byte {
	t.Helper()
	prepared, err := f.app.PrepareProposal(&abci.RequestPrepareProposal{
		Height: height, Time: schedulerTestTime(height), MaxTxBytes: simtestutil.DefaultConsensusParams.Block.MaxBytes, Txs: txs,
	})
	require.NoError(t, err)
	return prepared.Txs
}

func (f *schedulerTestFixture) process(t *testing.T, req *abci.RequestFinalizeBlock) {
	t.Helper()
	processed, err := f.app.ProcessProposal(&abci.RequestProcessProposal{
		Height: req.Height, Time: req.Time, Hash: append([]byte(nil), req.Hash...), Txs: req.Txs,
	})
	require.NoError(t, err)
	require.Equal(t, abci.ResponseProcessProposal_ACCEPT, processed.Status)
}

func (f *schedulerTestFixture) block(t *testing.T, height int64, txs ...[]byte) *abci.ResponseFinalizeBlock {
	t.Helper()
	prepared := f.prepare(t, height, txs)
	require.Equal(t, len(txs), len(prepared))
	for i := range txs {
		require.Equal(t, txs[i], prepared[i])
	}
	req := schedulerTestRequest(height, prepared)
	f.process(t, req)
	response, err := f.app.FinalizeBlock(req)
	require.NoError(t, err)
	_, err = f.app.Commit()
	require.NoError(t, err)
	require.Equal(t, height, f.app.LastBlockHeight())
	require.Equal(t, response.AppHash, f.app.LastCommitID().Hash)
	require.NoError(t, f.app.ScheduleKeeper.AssertEscrowInvariant(f.ctx()))
	return response
}

func (f *schedulerTestFixture) check(t *testing.T, tx []byte) *abci.ResponseCheckTx {
	t.Helper()
	response, err := f.app.CheckTx(&abci.RequestCheckTx{Tx: tx, Type: abci.CheckTxType_New})
	require.NoError(t, err)
	return response
}

func (f *schedulerTestFixture) schedule(t *testing.T, id uint64) *scheduletypes.Schedule {
	t.Helper()
	s, found := f.app.ScheduleKeeper.GetSchedule(f.ctx(), scheduletypes.FormatScheduleID(id))
	require.True(t, found)
	return s
}

func (f *schedulerTestFixture) cancel(t *testing.T, id, revision uint64, count uint32, sequence uint64) []byte {
	t.Helper()
	return f.signedMsg(t, &scheduletypes.MsgCancelSchedule{Creator: f.sender.String(),
		ScheduleId: scheduletypes.FormatScheduleID(id), ExpectedRevision: revision, ExpectedExecutionCount: count}, sequence)
}

func schedulerRequireTx(t *testing.T, response *abci.ResponseFinalizeBlock, index int, code uint32) {
	t.Helper()
	require.Greater(t, len(response.TxResults), index)
	require.Equal(t, code, response.TxResults[index].Code, response.TxResults[index].Log)
}

func TestSchedulerSignedCreationEscrowsAndExecutes(t *testing.T) {
	f := newSchedulerTestFixture(t, schedulerTestOptions{acceptNew: true})
	initialSender := f.app.BankKeeper.GetBalance(f.ctx(), f.sender, BondDenom).Amount
	tx := f.signedMsg(t, &scheduletypes.MsgCreateSchedule{
		Creator: f.sender.String(), Recipient: f.recipient.String(), AmountPerExecutionUzrn: "50",
		FirstExecutionHeight: 4, ExecutionCount: 1,
	}, 0)
	checked := f.check(t, tx)
	require.Zero(t, checked.Code, checked.Log)
	schedulerRequireTx(t, f.block(t, 2, tx), 0, 0)
	s := f.schedule(t, 1)
	require.Equal(t, scheduletypes.ScheduleStatus_SCHEDULE_STATUS_ACTIVE, s.Status)
	require.Equal(t, uint64(4), s.NextExecutionHeight)
	require.Equal(t, "100050", f.app.ScheduleKeeper.TotalEscrow(f.ctx()).String())
	require.Equal(t, "100050", f.app.BankKeeper.GetBalance(f.ctx(), authtypes.NewModuleAddress(scheduletypes.ModuleName), BondDenom).Amount.String())
	require.Empty(t, f.app.ScheduleKeeper.GetAllReceipts(f.ctx()))
	require.Equal(t, initialSender.SubRaw(int64(schedulerTestGas)+100_050), f.app.BankKeeper.GetBalance(f.ctx(), f.sender, BondDenom).Amount,
		"creation charges its ordinary transaction fee and prefunds all principal plus execution fees")
	f.block(t, 3)
	before := schedulerSnapshot(t, f)
	feeCollector := authtypes.NewModuleAddress(authtypes.FeeCollectorName)
	beforeFee := f.app.BankKeeper.GetBalance(f.ctx(), feeCollector, BondDenom).Amount
	beforeSender := f.app.BankKeeper.GetBalance(f.ctx(), f.sender, BondDenom).Amount
	paid := f.block(t, 4)
	require.Empty(t, paid.TxResults, "an empty ordinary block must still execute protocol work")
	require.Contains(t, schedulerEventTypes(paid.Events), "zerone.message_schedule.executed")
	s = f.schedule(t, 1)
	require.Equal(t, scheduletypes.ScheduleStatus_SCHEDULE_STATUS_COMPLETED, s.Status)
	require.Equal(t, uint32(1), s.ExecutionCount)
	require.Zero(t, s.RemainingExecutions)
	require.Zero(t, s.NextExecutionHeight)
	require.Equal(t, uint64(4), s.LastExecutionHeight)
	require.Equal(t, "50", f.app.BankKeeper.GetBalance(f.ctx(), f.recipient, BondDenom).Amount.String())
	require.Equal(t, "0", f.app.ScheduleKeeper.TotalEscrow(f.ctx()).String())
	r, found := f.app.ScheduleKeeper.GetReceipt(f.ctx(), s.Id, 1)
	require.True(t, found)
	require.Equal(t, scheduletypes.ExecutionOutcome_EXECUTION_OUTCOME_SUCCEEDED, r.Outcome)
	require.Equal(t, uint64(4), r.DueHeight)
	require.Equal(t, uint64(4), r.ExecutedHeight)
	require.Equal(t, "50", r.AmountUzrn)
	require.Equal(t, "100000", r.FeeUzrn)
	require.Equal(t, scheduletypes.OccurrenceID(schedulerTestChainID, s.Id, 1, 1, 4), r.OccurrenceId)
	after := schedulerSnapshot(t, f)
	require.Equal(t, before.Supply, after.Supply)
	require.Equal(t, beforeFee.AddRaw(100_000), f.app.BankKeeper.GetBalance(f.ctx(), feeCollector, BondDenom).Amount)
	require.Equal(t, beforeSender, f.app.BankKeeper.GetBalance(f.ctx(), f.sender, BondDenom).Amount,
		"execution consumes committed escrow, not a fresh creator debit")
	f.block(t, 5)
	later := schedulerSnapshot(t, f)
	require.Equal(t, after.Receipts, later.Receipts)
	require.Equal(t, after.Recipient, later.Recipient)
}

func TestSchedulerClosedAdmissionAndNonIncreasingAmendments(t *testing.T) {
	f := newSchedulerTestFixture(t, schedulerTestOptions{schedules: 1, due: 10})
	require.False(t, f.app.ScheduleKeeper.GetParams(f.ctx()).AcceptNewSchedules)
	require.Equal(t, "100050", f.app.ScheduleKeeper.TotalEscrow(f.ctx()).String())
	create := f.signedMsg(t, &scheduletypes.MsgCreateSchedule{
		Creator: f.sender.String(), Recipient: f.recipient.String(), AmountPerExecutionUzrn: "50",
		FirstExecutionHeight: 10, ExecutionCount: 1,
	}, 0)
	// CheckTx/ProcessProposal validate Ante, not this module's admission switch.
	checked := f.check(t, create)
	require.Zero(t, checked.Code, checked.Log)
	schedulerRequireTx(t, f.block(t, 2, create), 0, scheduletypes.ErrAdmissionClosed.ABCICode())
	require.Equal(t, uint64(2), f.app.ScheduleKeeper.PeekNextScheduleID(f.ctx()))
	update := func(amount string, sequence uint64) []byte {
		return f.signedMsg(t, &scheduletypes.MsgUpdateSchedule{
			Creator: f.sender.String(), ScheduleId: scheduletypes.FormatScheduleID(1), ExpectedRevision: 1,
			Recipient: f.recipient.String(), AmountPerExecutionUzrn: amount, NextExecutionHeight: 10, RemainingExecutions: 1,
		}, sequence)
	}
	schedulerRequireTx(t, f.block(t, 3, update("51", 1)), 0, scheduletypes.ErrAdmissionClosed.ABCICode())
	require.Equal(t, uint64(1), f.schedule(t, 1).Revision)
	schedulerRequireTx(t, f.block(t, 4, update("40", 2)), 0, 0)
	require.Equal(t, uint64(2), f.schedule(t, 1).Revision)
	require.Equal(t, "100040", f.app.ScheduleKeeper.TotalEscrow(f.ctx()).String())
	schedulerRequireTx(t, f.block(t, 5, f.cancel(t, 1, 1, 0, 3)), 0, scheduletypes.ErrRevisionConflict.ABCICode())
	schedulerRequireTx(t, f.block(t, 6, f.cancel(t, 1, 2, 0, 4)), 0, 0)
	require.Equal(t, scheduletypes.ScheduleStatus_SCHEDULE_STATUS_CANCELLED, f.schedule(t, 1).Status)
	require.Equal(t, "0", f.app.ScheduleKeeper.TotalEscrow(f.ctx()).String())
	require.Empty(t, f.app.ScheduleKeeper.GetAllReceipts(f.ctx()))
	schedulerSnapshot(t, f)
}

func TestSchedulerSelectedVersusUnselectedOverdueCancellation(t *testing.T) {
	f := newSchedulerTestFixture(t, schedulerTestOptions{schedules: 3, due: 3, occurrences: 2, interval: 10, dueCap: 1})
	f.block(t, 2)
	f.block(t, 3) // Selects only schedule 1. Schedules 2 and 3 become overdue.
	require.Zero(t, f.schedule(t, 2).ExecutionCount)
	response := f.block(t, 4, f.cancel(t, 2, 1, 0, 0), f.cancel(t, 3, 1, 0, 1))
	// BeginBlock selects overdue schedule 2 before the signed cancellation. Its
	// stale execution-count CAS fails; schedule 3 is outside that bounded prefix.
	schedulerRequireTx(t, response, 0, scheduletypes.ErrExecutionConflict.ABCICode())
	schedulerRequireTx(t, response, 1, 0)
	require.Equal(t, uint32(1), f.schedule(t, 2).ExecutionCount)
	require.Equal(t, uint64(14), f.schedule(t, 2).NextExecutionHeight, "late fixed-delay is executed height + interval")
	require.Equal(t, scheduletypes.ScheduleStatus_SCHEDULE_STATUS_CANCELLED, f.schedule(t, 3).Status)
	require.Zero(t, f.schedule(t, 3).ExecutionCount)
	require.Equal(t, "100", f.app.BankKeeper.GetBalance(f.ctx(), f.recipient, BondDenom).Amount.String())
	require.Len(t, f.app.ScheduleKeeper.GetAllReceipts(f.ctx()), 2)
	schedulerSnapshot(t, f)
}

func TestSchedulerParametersAreProspectiveForCommittedTerms(t *testing.T) {
	params := scheduletypes.DefaultParams()
	params.MaxExecutionsPerSchedule = 1
	params.MaxTransferPerExecutionUzrn = "1"
	params.MinIntervalBlocks = 100
	params.ExecutionFeeUzrn = "200000"
	// A closed, backed import represents terms committed BEFORE these reduced
	// limits. This is not a signed governance-parameter-change rehearsal.
	f := newSchedulerTestFixture(t, schedulerTestOptions{
		schedules: 1, due: 3, occurrences: 2, interval: 2, params: params,
	})
	f.block(t, 2)
	f.block(t, 3)
	require.Equal(t, uint64(5), f.schedule(t, 1).NextExecutionHeight)
	require.Equal(t, "100000", f.schedule(t, 1).ExecutionFeeUzrn)
	f.block(t, 4)
	f.block(t, 5)
	require.Equal(t, uint32(2), f.schedule(t, 1).ExecutionCount)
	require.Equal(t, "100", f.app.BankKeeper.GetBalance(f.ctx(), f.recipient, BondDenom).Amount.String())
	require.Equal(t, "0", f.app.ScheduleKeeper.TotalEscrow(f.ctx()).String())
	schedulerSnapshot(t, f)
}

func TestSchedulerSignedSelfFreezeBlocksCancelNotEscrowPayment(t *testing.T) {
	f := newSchedulerTestFixture(t, schedulerTestOptions{disk: true, schedules: 1, due: 5})
	identity := ed25519.GenPrivKeyFromSecret([]byte("scheduler local identity proof"))
	publicKey := identity.PubKey().Bytes()
	keyHex := hex.EncodeToString(publicKey)
	did := "did:zrn:" + keyHex
	proof, err := zeroneauthtypes.AccountRegistrationProofSignBytes(
		schedulerTestChainID, f.sender.String(), did, publicKey, "agent", "")
	require.NoError(t, err)
	proofSignature, err := identity.Sign(proof)
	require.NoError(t, err)
	registration := f.signedMsg(t, &zeroneauthtypes.MsgRegisterAccount{
		Sender: f.sender.String(), Did: did, PublicKey: keyHex, AccountType: "agent", IdentityProofSignature: proofSignature,
	}, 0)
	schedulerRequireTx(t, f.block(t, 2, registration), 0, 0)
	account, found := f.app.ZeroneAuthKeeper.GetAccount(f.ctx(), f.sender.String())
	require.True(t, found)
	require.False(t, account.Flags.Frozen)
	freeze := f.signedMsg(t, &zeroneauthtypes.MsgFreezeAccount{
		Sender: f.sender.String(), Address: f.sender.String(), Reason: "local self-freeze regression",
	}, 1)
	checked := f.check(t, freeze)
	require.Zero(t, checked.Code, checked.Log)
	schedulerRequireTx(t, f.block(t, 3, freeze), 0, 0)
	account, found = f.app.ZeroneAuthKeeper.GetAccount(f.ctx(), f.sender.String())
	require.True(t, found)
	require.True(t, account.Flags.Frozen)
	beforeRestart := schedulerSnapshot(t, f)
	f.reopen(t)
	require.Equal(t, beforeRestart, schedulerSnapshot(t, f))
	require.Zero(t, f.app.GetContextForCheckTx(nil).BlockHeight())
	cancel := f.cancel(t, 1, 1, 0, 2)
	checked = f.check(t, cancel)
	require.Equal(t, zeroneauthtypes.ErrAccountFrozen.ABCICode(), checked.Code, checked.Log)
	processed, err := f.app.ProcessProposal(&abci.RequestProcessProposal{
		Height: 4, Time: schedulerTestTime(4), Txs: [][]byte{cancel},
	})
	require.NoError(t, err)
	require.Equal(t, abci.ResponseProcessProposal_REJECT, processed.Status)
	f.block(t, 4) // Never force an Ante-rejected proposal into the valid stream.
	f.block(t, 5)
	require.Equal(t, scheduletypes.ScheduleStatus_SCHEDULE_STATUS_COMPLETED, f.schedule(t, 1).Status)
	require.Equal(t, "50", f.app.BankKeeper.GetBalance(f.ctx(), f.recipient, BondDenom).Amount.String())
	require.Equal(t, uint64(2), f.app.AccountKeeper.GetAccount(f.ctx(), f.sender).GetSequence())
	schedulerSnapshot(t, f)
}

func TestSchedulerBankSendDisabledIsNotModulePaymentSwitch(t *testing.T) {
	f := newSchedulerTestFixture(t, schedulerTestOptions{schedules: 1, due: 3, sendDisabled: true})
	require.False(t, f.app.BankKeeper.IsSendEnabledDenom(f.ctx(), BondDenom))
	send := f.signedMsg(t, banktypes.NewMsgSend(f.sender, f.recipient, sdk.NewCoins(sdk.NewInt64Coin(BondDenom, 1))), 0)
	schedulerRequireTx(t, f.block(t, 2, send), 0, banktypes.ErrSendDisabled.ABCICode())
	f.block(t, 3)
	require.Equal(t, "50", f.app.BankKeeper.GetBalance(f.ctx(), f.recipient, BondDenom).Amount.String())
	require.Equal(t, scheduletypes.ScheduleStatus_SCHEDULE_STATUS_COMPLETED, f.schedule(t, 1).Status)
	schedulerSnapshot(t, f)
}

// Filter actual x/bank transfer-success events, not the scheduler's processed
// occurrence event (which deliberately also reports failed_and_refunded).
func schedulerBankTransfers(events []abci.Event, from, to string) []abci.Event {
	var matches []abci.Event
	for _, event := range events {
		if event.Type != banktypes.EventTypeTransfer {
			continue
		}
		var sender, recipient string
		for _, attribute := range event.Attributes {
			switch attribute.Key {
			case banktypes.AttributeKeySender:
				sender = attribute.Value
			case banktypes.AttributeKeyRecipient:
				recipient = attribute.Value
			}
		}
		if sender == from && recipient == to {
			matches = append(matches, event)
		}
	}
	return matches
}

func TestSchedulerFeeFailureRollsBackPrincipalAndRefundsAllEscrow(t *testing.T) {
	f := newSchedulerTestFixture(t, schedulerTestOptions{disk: true, schedules: 1, due: 3, occurrences: 2, interval: 10})
	f.block(t, 2)
	before := schedulerSnapshot(t, f)
	beforeSender := f.app.BankKeeper.GetBalance(f.ctx(), f.sender, BondDenom).Amount
	module := authtypes.NewModuleAddress(scheduletypes.ModuleName)
	collector := authtypes.NewModuleAddress(authtypes.FeeCollectorName)
	beforeFees := f.app.BankKeeper.GetBalance(f.ctx(), collector, BondDenom).Amount
	require.False(t, f.app.AccountKeeper.HasAccount(f.ctx(), f.recipient))
	var principalCalls, feeCalls, refundCalls int
	faultEnabled := true
	// Fixture-local real-bank restriction: principal succeeds, then fee fails.
	// Keep the production keeper and all existing bank restrictions intact.
	f.app.BankKeeper.AppendSendRestriction(func(ctx context.Context, from, to sdk.AccAddress, coins sdk.Coins) (sdk.AccAddress, error) {
		if !faultEnabled || !from.Equals(module) {
			return to, nil
		}
		sdkCtx := sdk.UnwrapSDKContext(ctx)
		switch {
		case to.Equals(f.recipient):
			principalCalls++
		case to.Equals(collector):
			feeCalls++
			require.Equal(t, "50", f.app.BankKeeper.GetBalance(ctx, f.recipient, BondDenom).Amount.String(), "principal actually wrote before fee failure")
			require.True(t, f.app.AccountKeeper.HasAccount(ctx, f.recipient))
			require.Len(t, schedulerBankTransfers(sdkCtx.EventManager().ABCIEvents(), module.String(), f.recipient.String()), 1)
			return to, fmt.Errorf("fixture fee transfer refused after principal write")
		case to.Equals(f.sender):
			refundCalls++
			require.Equal(t, "200100uzrn", coins.String())
			require.Equal(t, "0", f.app.BankKeeper.GetBalance(ctx, f.recipient, BondDenom).Amount.String(), "discard principal branch before refund")
			require.False(t, f.app.AccountKeeper.HasAccount(ctx, f.recipient))
		}
		return to, nil
	})
	defer func() { faultEnabled = false }()
	response := f.block(t, 3)
	require.Equal(t, 1, principalCalls)
	require.Equal(t, 1, feeCalls)
	require.Equal(t, 1, refundCalls)
	require.Empty(t, schedulerBankTransfers(response.Events, module.String(), f.recipient.String()))
	require.Empty(t, schedulerBankTransfers(response.Events, module.String(), collector.String()))
	require.Len(t, schedulerBankTransfers(response.Events, module.String(), f.sender.String()), 1)
	for _, event := range response.Events {
		if event.Type == banktypes.EventTypeCoinReceived {
			for _, attribute := range event.Attributes {
				require.False(t, attribute.Key == banktypes.AttributeKeyReceiver && attribute.Value == f.recipient.String(), "rolled-back recipient credit event leaked")
			}
		}
		if event.Type == "zerone.message_schedule.executed" {
			for _, attribute := range event.Attributes {
				if attribute.Key == "outcome" {
					require.Equal(t, "failed_and_refunded", attribute.Value)
				}
			}
		}
	}
	require.Equal(t, beforeSender.AddRaw(200_100), f.app.BankKeeper.GetBalance(f.ctx(), f.sender, BondDenom).Amount)
	require.Equal(t, beforeFees, f.app.BankKeeper.GetBalance(f.ctx(), collector, BondDenom).Amount)
	require.False(t, f.app.AccountKeeper.HasAccount(f.ctx(), f.recipient))
	s := f.schedule(t, 1)
	require.Equal(t, scheduletypes.ScheduleStatus_SCHEDULE_STATUS_FAILED, s.Status)
	require.Equal(t, uint32(1), s.ExecutionCount)
	require.Zero(t, s.RemainingExecutions)
	r, found := f.app.ScheduleKeeper.GetReceipt(f.ctx(), s.Id, 1)
	require.True(t, found)
	require.Equal(t, scheduletypes.ExecutionOutcome_EXECUTION_OUTCOME_FAILED_AND_REFUNDED, r.Outcome)
	require.Equal(t, scheduletypes.FailureCodeBankTransfer, r.FailureCode)
	after := schedulerSnapshot(t, f)
	require.Equal(t, before.Supply, after.Supply)
	require.Equal(t, "0", after.Recipient)
	require.Equal(t, "0", after.Liability)
	require.Len(t, after.Receipts, 1)
	faultEnabled = false
	f.reopen(t)
	require.Equal(t, after, schedulerSnapshot(t, f))
	f.block(t, 4)
	require.Equal(t, after.Receipts, schedulerSnapshot(t, f).Receipts)
}

func TestSchedulerBlockedRecipientFailsAndRefundsAtomically(t *testing.T) {
	f := newSchedulerTestFixture(t, schedulerTestOptions{schedules: 1, due: 3, occurrences: 2, interval: 10})
	f.block(t, 2)
	before := f.app.BankKeeper.GetBalance(f.ctx(), f.sender, BondDenom)
	// A test-local bank blocklist change after valid genesis, not a governance
	// API or a new production policy. The actual bank keeper remains installed.
	blocked := f.app.BankKeeper.GetBlockedAddresses()
	blocked[f.recipient.String()] = true
	defer delete(blocked, f.recipient.String())
	f.block(t, 3)
	s := f.schedule(t, 1)
	require.Equal(t, scheduletypes.ScheduleStatus_SCHEDULE_STATUS_FAILED, s.Status)
	require.Equal(t, uint32(1), s.ExecutionCount, "processed is not successfully paid")
	require.Equal(t, "0", f.app.BankKeeper.GetBalance(f.ctx(), f.recipient, BondDenom).Amount.String())
	require.Equal(t, before.Amount.AddRaw(200_100), f.app.BankKeeper.GetBalance(f.ctx(), f.sender, BondDenom).Amount)
	require.Equal(t, "0", f.app.ScheduleKeeper.TotalEscrow(f.ctx()).String())
	r, found := f.app.ScheduleKeeper.GetReceipt(f.ctx(), s.Id, 1)
	require.True(t, found)
	require.Equal(t, scheduletypes.ExecutionOutcome_EXECUTION_OUTCOME_FAILED_AND_REFUNDED, r.Outcome)
	require.Equal(t, scheduletypes.FailureCodeBankTransfer, r.FailureCode)
	snapshot := schedulerSnapshot(t, f)
	f.block(t, 4)
	require.Equal(t, snapshot.Receipts, schedulerSnapshot(t, f).Receipts)
}

func TestSchedulerEmergencyHaltPausesProtocolWorkWithoutTimerResume(t *testing.T) {
	f := newSchedulerTestFixture(t, schedulerTestOptions{disk: true, schedules: 1, due: 3, halted: true})
	before := schedulerSnapshot(t, f)
	for height := int64(2); height <= 6; height++ {
		cancel := f.cancel(t, 1, 1, 0, 0)
		checked := f.check(t, cancel)
		require.Equal(t, emergencytypes.ErrChainHalted.ABCICode(), checked.Code, checked.Log)
		response := f.block(t, height)
		require.Empty(t, response.TxResults)
		require.NotContains(t, schedulerEventTypes(response.Events), "zerone.message_schedule.executed")
		require.True(t, f.app.EmergencyKeeper.IsHalted(f.ctx()))
		require.Zero(t, f.app.EmergencyKeeper.GetQuarantineReleaseBlock(f.ctx()))
		state := schedulerSnapshot(t, f)
		require.Equal(t, before.RawStore, state.RawStore)
		require.Equal(t, before.Balances, state.Balances)
		if height == 4 {
			// The two-block local escalation deadline has passed. Neither time
			// nor a normal restart grants affirmative resume authority.
			f.reopen(t)
			require.Equal(t, state, schedulerSnapshot(t, f))
		}
	}
}

func TestSchedulerEmergencyResumeGraceBoundaries(t *testing.T) {
	const resumeHeight = int64(4)
	// The persisted release latch represents an affirmative resume's output.
	// This tests runtime boundaries, not a signed Guardian voting ceremony.
	f := newSchedulerTestFixture(t, schedulerTestOptions{
		disk: true, schedules: 3, due: 3, releaseHeight: uint64(resumeHeight),
	})
	for height := int64(2); height <= resumeHeight+11; height++ {
		if height == resumeHeight || height == resumeHeight+11 {
			before := schedulerSnapshot(t, f)
			f.reopen(t)
			require.Equal(t, before, schedulerSnapshot(t, f))
		}
		if height <= resumeHeight {
			cancel := f.cancel(t, 2, 1, 0, 0)
			checked := f.check(t, cancel)
			require.Equal(t, emergencytypes.ErrChainHalted.ABCICode(), checked.Code, checked.Log)
			processed, err := f.app.ProcessProposal(&abci.RequestProcessProposal{
				Height: height, Time: schedulerTestTime(height), Txs: [][]byte{cancel},
			})
			require.NoError(t, err)
			require.Equal(t, abci.ResponseProcessProposal_REJECT, processed.Status)
		}
		var txs [][]byte
		if height == resumeHeight+1 {
			// Ordinary signed cancellation is available during the scheduler-only grace.
			tx := f.cancel(t, 2, 1, 0, 0)
			checked := f.check(t, tx)
			require.Zero(t, checked.Code, checked.Log)
			before := schedulerSnapshot(t, f)
			f.reopen(t)
			require.Equal(t, before, schedulerSnapshot(t, f))
			// SDK 0.53.8 loads an empty CheckTx header. App Ante must use the
			// trusted committed height H (not H+1) without inventing block time.
			require.Zero(t, f.app.GetContextForCheckTx(nil).BlockHeight())
			require.True(t, f.app.GetContextForCheckTx(nil).BlockTime().IsZero())
			restartedCheck := f.check(t, tx)
			require.Zero(t, restartedCheck.Code, restartedCheck.Log)
			txs = [][]byte{tx}
		}
		if height == resumeHeight+10 {
			// Restart after committing H+9: local admission must be immediately
			// available for the final H+10 cancellation opportunity.
			before := schedulerSnapshot(t, f)
			f.reopen(t)
			require.Equal(t, before, schedulerSnapshot(t, f))
			require.Equal(t, resumeHeight+9, f.app.LastBlockHeight())
			require.Zero(t, f.app.GetContextForCheckTx(nil).BlockHeight())
			tx := f.cancel(t, 3, 1, 0, 1)
			checked := f.check(t, tx)
			require.Zero(t, checked.Code, checked.Log)
			txs = [][]byte{tx}
		}
		response := f.block(t, height, txs...)
		if len(txs) > 0 {
			schedulerRequireTx(t, response, 0, 0)
			sequence := f.app.AccountKeeper.GetAccount(f.ctx(), f.sender).GetSequence()
			checked := f.check(t, f.cancel(t, 1, 1, 0, sequence))
			require.Zero(t, checked.Code, "after-commit CheckTx remains admissible: %s", checked.Log)
			cancelledID := uint64(2)
			if height == resumeHeight+10 {
				cancelledID = 3
			}
			require.Equal(t, scheduletypes.ScheduleStatus_SCHEDULE_STATUS_CANCELLED, f.schedule(t, cancelledID).Status)
			require.Zero(t, f.schedule(t, cancelledID).ExecutionCount)
		}
		ctx := f.ctx()
		if height <= resumeHeight {
			require.True(t, f.app.EmergencyKeeper.IsHalted(ctx), "H remains quarantined")
		} else {
			require.False(t, f.app.EmergencyKeeper.IsHalted(ctx), "ordinary admission opens at H+1")
		}
		if height <= resumeHeight+10 {
			require.Equal(t, uint64(resumeHeight), f.app.EmergencyKeeper.GetQuarantineReleaseBlock(ctx))
			require.Zero(t, f.schedule(t, 1).ExecutionCount, "protocol work stays paused THROUGH H+10")
			require.Empty(t, f.app.ScheduleKeeper.GetAllReceipts(ctx))
			require.Equal(t, "0", f.app.BankKeeper.GetBalance(ctx, f.recipient, BondDenom).Amount.String())
		} else {
			require.Zero(t, f.app.EmergencyKeeper.GetQuarantineReleaseBlock(ctx))
			require.Equal(t, uint32(1), f.schedule(t, 1).ExecutionCount)
			receipt, found := f.app.ScheduleKeeper.GetReceipt(ctx, scheduletypes.FormatScheduleID(1), 1)
			require.True(t, found)
			require.Equal(t, uint64(3), receipt.DueHeight)
			require.Equal(t, uint64(resumeHeight+11), receipt.ExecutedHeight)
			require.Equal(t, "50", f.app.BankKeeper.GetBalance(ctx, f.recipient, BondDenom).Amount.String())
		}
		schedulerSnapshot(t, f)
	}
}

func TestSchedulerNewHaltVotePreservesResumeGraceAcrossContinuation(t *testing.T) {
	f := newSchedulerTestFixture(t, schedulerTestOptions{disk: true, schedules: 1, due: 3, halted: true, guardian: true})
	resume := f.signedMsg(t, &emergencytypes.MsgProposeResume{
		Proposer: f.sender.String(), Justification: "isolated local continuation test",
		RecoveryManifestSha256: fmt.Sprintf("%064x", 1),
	}, 0)
	schedulerRequireTx(t, f.block(t, 2, resume), 0, 0)
	ceremony, found := f.app.EmergencyKeeper.GetActiveCeremony(f.ctx())
	require.True(t, found)
	for height := int64(3); height <= 4; height++ {
		vote := f.signedMsg(t, &emergencytypes.MsgVoteResume{
			Voter: f.sender.String(), ProposalId: ceremony.Id, Approve: true,
		}, uint64(height-2))
		schedulerRequireTx(t, f.block(t, height, vote), 0, 0)
	}
	const release = uint64(4)
	require.Equal(t, release, f.app.EmergencyKeeper.GetQuarantineReleaseBlock(f.ctx()))
	propose := f.signedMsg(t, &emergencytypes.MsgProposeHalt{
		Proposer: f.sender.String(), Reason: "later vote is not quarantine authority",
	}, 3)
	schedulerRequireTx(t, f.block(t, 5, propose), 0, 0)
	require.Equal(t, emergencytypes.StatusHaltVoting, f.app.EmergencyKeeper.GetEmergencyStatus(f.ctx()))
	require.False(t, f.app.EmergencyKeeper.IsHalted(f.ctx()))
	require.Equal(t, release, f.app.EmergencyKeeper.GetQuarantineReleaseBlock(f.ctx()))
	before := schedulerSnapshot(t, f)
	exported, err := f.app.ExportAppStateAndValidators(false, nil, nil)
	require.NoError(t, err)
	require.Equal(t, int64(6), exported.Height)
	var modules map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(exported.AppState, &modules))
	var emergencyGenesis emergencytypes.GenesisState
	require.NoError(t, json.Unmarshal(modules[emergencytypes.ModuleName], &emergencyGenesis))
	require.NoError(t, emergencyGenesis.Validate())
	// A continuation import is distinct from replay. It validates/rebuilds
	// genesis state; the separate LevelDB tests compare persisted raw indexes.
	continued := newSchedulerTestFixture(t, schedulerTestOptions{
		genesisBytes: exported.AppState, initialHeight: exported.Height,
	})
	require.Equal(t, before.RawStore, schedulerSnapshot(t, continued).RawStore)
	for height := int64(6); height <= int64(release)+11; height++ {
		f.block(t, height)
		if height > exported.Height {
			continued.block(t, height)
		}
		for _, replica := range []*schedulerTestFixture{f, continued} {
			ctx := replica.ctx()
			require.False(t, replica.app.ScheduleKeeper.GetParams(ctx).AcceptNewSchedules)
			require.Equal(t, emergencytypes.StatusHaltVoting, replica.app.EmergencyKeeper.GetEmergencyStatus(ctx))
			require.False(t, replica.app.EmergencyKeeper.IsHalted(ctx))
			if height <= int64(release)+10 {
				require.Equal(t, release, replica.app.EmergencyKeeper.GetQuarantineReleaseBlock(ctx))
				require.Zero(t, replica.schedule(t, 1).ExecutionCount)
				require.Empty(t, replica.app.ScheduleKeeper.GetAllReceipts(ctx))
				require.Equal(t, "0", replica.app.BankKeeper.GetBalance(ctx, replica.recipient, BondDenom).Amount.String())
			} else {
				require.Zero(t, replica.app.EmergencyKeeper.GetQuarantineReleaseBlock(ctx))
				require.Equal(t, uint32(1), replica.schedule(t, 1).ExecutionCount)
				require.Equal(t, "50", replica.app.BankKeeper.GetBalance(ctx, replica.recipient, BondDenom).Amount.String())
			}
		}
		require.Equal(t, schedulerSnapshot(t, f).RawStore, schedulerSnapshot(t, continued).RawStore)
	}
}

func schedulerEventTypes(events []abci.Event) []string {
	var names []string
	for _, event := range events {
		names = append(names, event.Type)
	}
	return names
}
