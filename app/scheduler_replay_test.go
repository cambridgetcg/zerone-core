package app

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/big"
	"sort"
	"strings"
	"testing"

	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	scheduletypes "github.com/zerone-chain/zerone/x/schedule/types"
)

type schedulerStateSnapshot struct {
	Height    int64
	AppHash   string
	Supply    string
	Balances  []string
	Recipient string
	Liability string
	NextID    uint64
	Schedules []string
	Receipts  []string
	RawStore  []string
}

func schedulerProtoBytes(t *testing.T, message proto.Message) []byte {
	t.Helper()
	bz, err := (proto.MarshalOptions{Deterministic: true}).Marshal(message)
	require.NoError(t, err)
	return bz
}

func schedulerKV(key, value []byte) string {
	return hex.EncodeToString(key) + "=" + hex.EncodeToString(value)
}

// Read persisted KV state directly; export/import is deliberately NOT used.
// Besides replica equality, compare each raw index against its exact expected
// key/value set. Export reconstructs these indexes and could hide an extra,
// missing, incorrectly ordered due key or misdirected occurrence pointer.
func schedulerSnapshot(t *testing.T, f *schedulerTestFixture) schedulerStateSnapshot {
	t.Helper()
	ctx := f.ctx()
	k := f.app.ScheduleKeeper
	require.NoError(t, k.AssertEscrowInvariant(ctx))
	state := schedulerStateSnapshot{
		Height: f.app.LastBlockHeight(), AppHash: hex.EncodeToString(f.app.LastCommitID().Hash),
		Supply:    f.app.BankKeeper.GetSupply(ctx, BondDenom).String(),
		Recipient: f.app.BankKeeper.GetBalance(ctx, f.recipient, BondDenom).Amount.String(),
		Liability: k.TotalEscrow(ctx).String(), NextID: k.PeekNextScheduleID(ctx),
	}
	balanceSum := new(big.Int)
	f.app.BankKeeper.IterateAllBalances(ctx, func(address sdk.AccAddress, coin sdk.Coin) bool {
		state.Balances = append(state.Balances, address.String()+"="+coin.String())
		require.Equal(t, BondDenom, coin.Denom, "fixture has only one denomination")
		balanceSum.Add(balanceSum, coin.Amount.BigInt())
		return false
	})
	sort.Strings(state.Balances)
	require.Equal(t, f.app.BankKeeper.GetSupply(ctx, BondDenom).Amount.String(), balanceSum.String())

	families := [][]byte{scheduletypes.DueKeyPrefix, scheduletypes.CreatorKeyPrefix,
		scheduletypes.ActiveCreatorPrefix, scheduletypes.ReceiptKeyPrefix, scheduletypes.OccurrenceKeyPrefix}
	expected := make(map[byte][]string)
	liability := new(big.Int)
	schedules := k.GetAllSchedules(ctx)
	for _, s := range schedules {
		require.NoError(t, scheduletypes.ValidateStoredSchedule(s, k.GetParams(ctx)))
		state.Schedules = append(state.Schedules, hex.EncodeToString(schedulerProtoBytes(t, s)))
		creator, err := sdk.AccAddressFromBech32(s.Creator)
		require.NoError(t, err)
		expected[scheduletypes.CreatorKeyPrefix[0]] = append(expected[scheduletypes.CreatorKeyPrefix[0]],
			schedulerKV(scheduletypes.CreatorKey(creator, s.Id), []byte{1}))
		if s.Status == scheduletypes.ScheduleStatus_SCHEDULE_STATUS_ACTIVE {
			expected[scheduletypes.ActiveCreatorPrefix[0]] = append(expected[scheduletypes.ActiveCreatorPrefix[0]],
				schedulerKV(scheduletypes.ActiveCreatorKey(creator, s.Id), []byte{1}))
			expected[scheduletypes.DueKeyPrefix[0]] = append(expected[scheduletypes.DueKeyPrefix[0]],
				schedulerKV(scheduletypes.DueKey(s.NextExecutionHeight, s.Id), []byte{1}))
		}
		principal, err := scheduletypes.ParseNonNegativeAmount(s.PrincipalRemainingUzrn)
		require.NoError(t, err)
		fees, err := scheduletypes.ParseNonNegativeAmount(s.FeeRemainingUzrn)
		require.NoError(t, err)
		liability.Add(liability, principal).Add(liability, fees)
	}
	require.Equal(t, state.Liability, liability.String())
	seen := make(map[string]bool)
	counts := make(map[string]uint32)
	for _, r := range k.GetAllReceipts(ctx) {
		require.NoError(t, scheduletypes.ValidateReceipt(r))
		require.False(t, seen[r.OccurrenceId], "duplicate committed occurrence")
		seen[r.OccurrenceId] = true
		counts[r.ScheduleId]++
		require.Equal(t, counts[r.ScheduleId], r.Sequence)
		require.Equal(t, scheduletypes.OccurrenceID(schedulerTestChainID, r.ScheduleId, r.Revision, r.Sequence, r.DueHeight), r.OccurrenceId)
		byOccurrence, found := k.GetReceiptByOccurrence(ctx, r.OccurrenceId)
		require.True(t, found)
		raw := schedulerProtoBytes(t, r)
		require.Equal(t, raw, schedulerProtoBytes(t, byOccurrence))
		state.Receipts = append(state.Receipts, hex.EncodeToString(raw))
		expected[scheduletypes.ReceiptKeyPrefix[0]] = append(expected[scheduletypes.ReceiptKeyPrefix[0]],
			schedulerKV(scheduletypes.ReceiptKey(r.ScheduleId, r.Sequence), raw))
		expected[scheduletypes.OccurrenceKeyPrefix[0]] = append(expected[scheduletypes.OccurrenceKeyPrefix[0]],
			schedulerKV(scheduletypes.OccurrenceKey(r.OccurrenceId), scheduletypes.ReceiptKey(r.ScheduleId, r.Sequence)))
	}
	for _, s := range schedules {
		require.Equal(t, s.ExecutionCount, counts[s.Id], "each processed occurrence must have exactly one receipt")
	}
	store := ctx.KVStore(f.app.keys[scheduletypes.StoreKey])
	iterator := store.Iterator(nil, nil)
	actual := make(map[byte][]string)
	for ; iterator.Valid(); iterator.Next() {
		key, value := iterator.Key(), iterator.Value()
		require.NotEmpty(t, key)
		kv := schedulerKV(key, value)
		state.RawStore = append(state.RawStore, kv)
		actual[key[0]] = append(actual[key[0]], kv)
	}
	err := iterator.Error()
	// SDK root cache iterators use this exact sentinel at normal exhaustion.
	if err != nil && err.Error() != "invalid cacheMergeIterator" && err.Error() != "invalid memIterator" {
		require.NoError(t, err)
	}
	require.NoError(t, iterator.Close())
	require.True(t, sort.StringsAreSorted(state.RawStore), "raw iteration order must be canonical")
	for _, family := range families {
		sort.Strings(expected[family[0]])
		require.Equal(t, expected[family[0]], actual[family[0]], "index prefix %x", family)
	}
	return state
}

type schedulerBlockEvidence struct {
	FinalizeBytes []byte
	ResponseHash  [32]byte
	CommitBytes   []byte
	State         schedulerStateSnapshot
}

// Ordinary consensus path: proposal validation precedes finalization.
func schedulerProcessFinalizeCommit(t *testing.T, f *schedulerTestFixture, requestBytes []byte) schedulerBlockEvidence {
	t.Helper()
	req := new(abci.RequestFinalizeBlock)
	require.NoError(t, req.Unmarshal(requestBytes))
	f.process(t, req)
	return schedulerFinalizeCommit(t, f, requestBytes)
}

// Direct Comet replay path: NO PrepareProposal or ProcessProposal. On a reopened
// app this exercises SDK 0.53.8's nil-finalizeBlockState replay branch. Unmarshal
// afresh for every replica; authoritative bytes, never shared request objects.
func schedulerFinalizeCommit(t *testing.T, f *schedulerTestFixture, requestBytes []byte) schedulerBlockEvidence {
	t.Helper()
	req := new(abci.RequestFinalizeBlock)
	require.NoError(t, req.Unmarshal(requestBytes))
	response, err := f.app.FinalizeBlock(req)
	require.NoError(t, err)
	finalizeBytes, err := response.Marshal()
	require.NoError(t, err)
	commit, err := f.app.Commit()
	require.NoError(t, err)
	commitBytes, err := commit.Marshal()
	require.NoError(t, err)
	require.Equal(t, req.Height, f.app.LastBlockHeight())
	require.Equal(t, response.AppHash, f.app.LastCommitID().Hash)
	return schedulerBlockEvidence{
		FinalizeBytes: finalizeBytes, ResponseHash: sha256.Sum256(finalizeBytes),
		CommitBytes: commitBytes, State: schedulerSnapshot(t, f),
	}
}

func TestSchedulerIndependentLevelDBReplicasAndDueBoundaryReopens(t *testing.T) {
	const due = int64(4)
	control := newSchedulerTestFixture(t, schedulerTestOptions{
		disk: true, schedules: 3, due: uint64(due), occurrences: 2, interval: 2, dueCap: 1,
	})
	beforeAndAfterCommit := newSchedulerTestFixture(t, schedulerTestOptions{disk: true, genesisBytes: control.genesisBytes})
	uncommittedFinalize := newSchedulerTestFixture(t, schedulerTestOptions{disk: true, genesisBytes: control.genesisBytes})
	replicas := []*schedulerTestFixture{control, beforeAndAfterCommit, uncommittedFinalize}
	for i, f := range replicas {
		require.Equal(t, control.genesisBytes, f.genesisBytes)
		require.False(t, f.app.ScheduleKeeper.GetParams(f.ctx()).AcceptNewSchedules)
		require.Equal(t, schedulerSnapshot(t, control), schedulerSnapshot(t, f))
		for j := 0; j < i; j++ {
			require.NotEqual(t, replicas[j].home, f.home)
			require.NotSame(t, replicas[j].app, f.app)
		}
	}

	send := func(amount int64, sequence uint64) []byte {
		return control.signedMsg(t, banktypes.NewMsgSend(control.sender, control.recipient,
			sdk.NewCoins(sdk.NewInt64Coin(BondDenom, amount))), sequence)
	}
	authoritativeSend := send(1, 0)
	// Distinct local CheckTx state, including a nonce successor and a competing
	// transfer absent from the final stream. No app-mempool implementation.
	for _, tx := range [][]byte{authoritativeSend, send(2, 1)} {
		checked := control.check(t, tx)
		require.Zero(t, checked.Code, checked.Log)
	}
	require.NotZero(t, beforeAndAfterCommit.check(t, []byte("not a transaction")).Code)
	competing := send(99, 0)
	checked := uncommittedFinalize.check(t, competing)
	require.Zero(t, checked.Code, checked.Log)
	require.Equal(t, [][]byte{competing}, beforeAndAfterCommit.prepare(t, 2, [][]byte{competing}))
	require.Empty(t, uncommittedFinalize.prepare(t, 2, nil))

	firstState := schedulerSnapshot(t, control)
	var durableBeforeDue schedulerStateSnapshot
	immutableReceipts := make(map[string][]byte)
	for height := int64(2); height <= 11; height++ {
		var candidates [][]byte
		if height == 2 {
			candidates = [][]byte{authoritativeSend}
		}
		prepared := control.prepare(t, height, candidates)
		if height == 2 {
			require.Equal(t, candidates, prepared)
		} else {
			require.Empty(t, prepared)
		}
		requestBytes, err := schedulerTestRequest(height, prepared).Marshal()
		require.NoError(t, err)

		var uncommittedBytes []byte
		if height == due {
			// Boundary 1: reopen after Commit(D-1), retaining no app/DB handle.
			beforeAndAfterCommit.reopen(t)
			require.Equal(t, due-1, beforeAndAfterCommit.app.LastBlockHeight())
			require.Equal(t, durableBeforeDue, schedulerSnapshot(t, beforeAndAfterCommit))

			// Boundary 2: FinalizeBlock(D) computes payment, receipt and AppHash,
			// but NO Commit occurs. Closing is a controlled process boundary, not
			// an arbitrary mid-fsync power-loss simulation.
			req := new(abci.RequestFinalizeBlock)
			require.NoError(t, req.Unmarshal(requestBytes))
			uncommittedFinalize.process(t, req)
			uncommitted, err := uncommittedFinalize.app.FinalizeBlock(req)
			require.NoError(t, err)
			uncommittedBytes, err = uncommitted.Marshal()
			require.NoError(t, err)
			require.Contains(t, schedulerEventTypes(uncommitted.Events), "zerone.message_schedule.executed")
			require.NotEqual(t, durableBeforeDue.AppHash, hex.EncodeToString(uncommitted.AppHash))
			require.Equal(t, due-1, uncommittedFinalize.app.LastBlockHeight())
			uncommittedFinalize.reopen(t)
			require.Equal(t, durableBeforeDue, schedulerSnapshot(t, uncommittedFinalize))
		}

		var expected schedulerBlockEvidence
		for i, f := range replicas {
			var evidence schedulerBlockEvidence
			if height == due && f == uncommittedFinalize {
				// Actual replay from durable D-1 must not prime SDK finalization
				// state via ProcessProposal, unlike ordinary consensus below.
				evidence = schedulerFinalizeCommit(t, f, requestBytes)
			} else {
				evidence = schedulerProcessFinalizeCommit(t, f, requestBytes)
			}
			if i == 0 {
				expected = evidence
			} else {
				// Exact full ABCI response bytes include AppHash, transaction
				// results, gas, data and every transaction/block event.
				require.Equal(t, expected, evidence, "replica %d at height %d", i, height)
			}
			require.Equal(t, firstState.Supply, evidence.State.Supply)
		}
		if height == due-1 {
			durableBeforeDue = expected.State
		}
		if height == due {
			require.Equal(t, uncommittedBytes, expected.FinalizeBytes, "replayed D reproduces even the discarded response exactly")
			// Boundary 3: reopen immediately AFTER Commit(D).
			beforeAndAfterCommit.reopen(t)
			require.Equal(t, due, beforeAndAfterCommit.app.LastBlockHeight())
			require.Equal(t, expected.State, schedulerSnapshot(t, beforeAndAfterCommit))
		}
		var response abci.ResponseFinalizeBlock
		require.NoError(t, response.Unmarshal(expected.FinalizeBytes))
		if height == 2 {
			schedulerRequireTx(t, &response, 0, 0)
		} else {
			require.Empty(t, response.TxResults)
			require.Equal(t, uint64(1), control.app.AccountKeeper.GetAccount(control.ctx(), control.sender).GetSequence(),
				"empty ordinary blocks do not consume signer sequences")
		}

		// Independent timeline oracle: cap=1, three records due at D. The first
		// recurrence is based on actual execution; old overdue records outrank
		// newly recurring ones. There are six payments, never a catch-up burst.
		executions := [][]int64{{4, 7}, {5, 8}, {6, 9}}
		dueHeights := [][]uint64{{4, 6}, {4, 7}, {4, 8}}
		var paid int64
		for i, executionHeights := range executions {
			s := control.schedule(t, uint64(i+1))
			var count uint32
			for occurrence, executed := range executionHeights {
				if height < executed {
					continue
				}
				count++
				paid++
				r, found := control.app.ScheduleKeeper.GetReceipt(control.ctx(), s.Id, uint32(occurrence+1))
				require.True(t, found)
				require.Equal(t, uint64(executed), r.ExecutedHeight)
				require.Equal(t, dueHeights[i][occurrence], r.DueHeight)
				require.Equal(t, scheduletypes.ExecutionOutcome_EXECUTION_OUTCOME_SUCCEEDED, r.Outcome)
				raw := schedulerProtoBytes(t, r)
				if prior, exists := immutableReceipts[r.OccurrenceId]; exists {
					require.Equal(t, prior, raw, "committed receipts are immutable across later execution and restarts")
				} else {
					immutableReceipts[r.OccurrenceId] = append([]byte(nil), raw...)
				}
			}
			require.Equal(t, count, s.ExecutionCount)
			require.Equal(t, uint32(2)-count, s.RemainingExecutions)
			if count == 2 {
				require.Equal(t, scheduletypes.ScheduleStatus_SCHEDULE_STATUS_COMPLETED, s.Status)
				require.Zero(t, s.NextExecutionHeight)
			} else {
				require.Equal(t, scheduletypes.ScheduleStatus_SCHEDULE_STATUS_ACTIVE, s.Status)
				next := uint64(due)
				if count == 1 {
					next = uint64(executionHeights[0] + 2)
				}
				require.Equal(t, next, s.NextExecutionHeight)
			}
		}
		require.Equal(t, fmt.Sprint(1+paid*50), expected.State.Recipient)
		require.Equal(t, fmt.Sprint(600_300-paid*100_050), expected.State.Liability)
		require.Len(t, expected.State.Receipts, int(paid))
		if height >= due && height <= 9 {
			require.Equal(t, 1, strings.Count(strings.Join(schedulerEventTypes(response.Events), ","), "zerone.message_schedule.executed"))
		} else {
			require.NotContains(t, schedulerEventTypes(response.Events), "zerone.message_schedule.executed")
		}
	}
	require.Len(t, immutableReceipts, 6)
}

func TestSchedulerLaterOccurrenceRefundFailureDiscardsEarlierSuccessOnDirectReplay(t *testing.T) {
	control := newSchedulerTestFixture(t, schedulerTestOptions{disk: true, schedules: 2, due: 3})
	faulted := newSchedulerTestFixture(t, schedulerTestOptions{disk: true, genesisBytes: control.genesisBytes})
	control.block(t, 2)
	faulted.block(t, 2)
	before := schedulerSnapshot(t, faulted)
	require.Equal(t, schedulerSnapshot(t, control), before)
	module := authtypes.NewModuleAddress(scheduletypes.ModuleName)
	collector := authtypes.NewModuleAddress(authtypes.FeeCollectorName)
	beforeFees := faulted.app.BankKeeper.GetBalance(faulted.ctx(), collector, BondDenom).Amount
	var principalCalls, feeCalls, refundCalls int
	faultEnabled := true
	faulted.app.BankKeeper.AppendSendRestriction(func(ctx context.Context, from, to sdk.AccAddress, _ sdk.Coins) (sdk.AccAddress, error) {
		if !faultEnabled || !from.Equals(module) {
			return to, nil
		}
		sdkCtx := sdk.UnwrapSDKContext(ctx)
		switch {
		case to.Equals(faulted.recipient):
			principalCalls++
		case to.Equals(collector):
			feeCalls++
			if feeCalls == 1 {
				return to, nil // First occurrence fully succeeds in this SAME volatile block.
			}
			require.Equal(t, 2, feeCalls)
			require.Equal(t, "100", faulted.app.BankKeeper.GetBalance(ctx, faulted.recipient, BondDenom).Amount.String())
			first, found := faulted.app.ScheduleKeeper.GetSchedule(sdkCtx, scheduletypes.FormatScheduleID(1))
			require.True(t, found)
			require.Equal(t, scheduletypes.ScheduleStatus_SCHEDULE_STATUS_COMPLETED, first.Status)
			firstReceipt, found := faulted.app.ScheduleKeeper.GetReceipt(sdkCtx, first.Id, 1)
			require.True(t, found, "earlier occurrence already wrote its success receipt")
			require.Equal(t, scheduletypes.ExecutionOutcome_EXECUTION_OUTCOME_SUCCEEDED, firstReceipt.Outcome)
			require.Equal(t, beforeFees.AddRaw(100_000), faulted.app.BankKeeper.GetBalance(ctx, collector, BondDenom).Amount)
			return to, fmt.Errorf("fixture second occurrence fee failure after principal")
		case to.Equals(faulted.sender):
			refundCalls++
			require.Equal(t, "50", faulted.app.BankKeeper.GetBalance(ctx, faulted.recipient, BondDenom).Amount.String(), "only first occurrence survives in the volatile parent branch")
			require.Equal(t, beforeFees.AddRaw(100_000), faulted.app.BankKeeper.GetBalance(ctx, collector, BondDenom).Amount)
			_, found := faulted.app.ScheduleKeeper.GetReceipt(sdkCtx, scheduletypes.FormatScheduleID(2), 1)
			require.False(t, found)
			return to, fmt.Errorf("fixture second occurrence refund failure")
		}
		return to, nil
	})
	defer func() { faultEnabled = false }()
	req := schedulerTestRequest(3, nil)
	faulted.process(t, req)
	response, err := faulted.app.FinalizeBlock(req)
	require.ErrorContains(t, err, "second occurrence fee failure")
	require.ErrorContains(t, err, "refund failed")
	require.ErrorContains(t, err, "escrow invariant")
	require.Nil(t, response, "no successful ABCI result on failed finalization")
	require.Equal(t, 2, principalCalls)
	require.Equal(t, 2, feeCalls)
	require.Equal(t, 1, refundCalls)
	require.Equal(t, before, schedulerSnapshot(t, faulted), "neither earlier success nor later failure reached committed root")
	// Never Commit after the ABCI error. Reopen from the owned durable D-1,
	// clearing only our local closure fault. This is not power-loss/fsync proof.
	faultEnabled = false
	faulted.reopen(t)
	require.Equal(t, before, schedulerSnapshot(t, faulted))
	requestBytes, err := req.Marshal()
	require.NoError(t, err)
	want := schedulerProcessFinalizeCommit(t, control, requestBytes)
	got := schedulerFinalizeCommit(t, faulted, requestBytes) // Direct replay: no proposal calls.
	require.Equal(t, want, got, "full ABCI bytes/hash, balances, receipts and raw indexes agree")
	require.Equal(t, "100", got.State.Recipient)
	require.Equal(t, "0", got.State.Liability)
	require.Len(t, got.State.Receipts, 2)
	faulted.reopen(t)
	require.Equal(t, got.State, schedulerSnapshot(t, faulted))
	requestBytes, err = schedulerTestRequest(4, nil).Marshal()
	require.NoError(t, err)
	want = schedulerProcessFinalizeCommit(t, control, requestBytes)
	got = schedulerProcessFinalizeCommit(t, faulted, requestBytes)
	require.Equal(t, want, got)
	require.Len(t, got.State.Receipts, 2)
	require.Equal(t, "100", got.State.Recipient, "no duplicate committed payment")
}

func TestSchedulerFailedFinalizeKeepsDurableHeightAndReplaysAfterLocalFaultCleared(t *testing.T) {
	control := newSchedulerTestFixture(t, schedulerTestOptions{disk: true, schedules: 1, due: 3})
	faulted := newSchedulerTestFixture(t, schedulerTestOptions{disk: true, genesisBytes: control.genesisBytes})
	control.block(t, 2)
	faulted.block(t, 2)
	before := schedulerSnapshot(t, faulted)
	require.Equal(t, schedulerSnapshot(t, control), before)
	// Isolated fixture fault: both real x/bank module payment AND refund fail.
	// No mock keeper, production injector, or claim of real-node fault repair.
	blocked := faulted.app.BankKeeper.GetBlockedAddresses()
	blocked[faulted.recipient.String()] = true
	blocked[faulted.sender.String()] = true
	req := schedulerTestRequest(3, nil)
	faulted.process(t, req)
	_, err := faulted.app.FinalizeBlock(req)
	require.ErrorContains(t, err, "refund failed")
	require.ErrorContains(t, err, "escrow invariant")
	require.Equal(t, int64(2), faulted.app.LastBlockHeight())
	require.Equal(t, before, schedulerSnapshot(t, faulted))
	// Do not call Commit on an ABCI error. Clear ONLY the test-local fault and
	// discard the failed volatile branch by reopening the same durable home.
	delete(blocked, faulted.recipient.String())
	delete(blocked, faulted.sender.String())
	faulted.reopen(t)
	require.Equal(t, before, schedulerSnapshot(t, faulted))
	requestBytes, err := req.Marshal()
	require.NoError(t, err)
	want := schedulerProcessFinalizeCommit(t, control, requestBytes)
	got := schedulerFinalizeCommit(t, faulted, requestBytes)
	require.Equal(t, want, got)
	require.Equal(t, int64(3), got.State.Height)
	require.Equal(t, "50", got.State.Recipient)
	require.Equal(t, "0", got.State.Liability)
	require.Len(t, got.State.Receipts, 1)
}
