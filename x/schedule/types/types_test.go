package types_test

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/zerone-chain/zerone/x/schedule/types"
)

func TestCanonicalAmounts(t *testing.T) {
	for _, valid := range []string{"0", "1", "999999999999999999999999999999"} {
		_, err := types.ParseNonNegativeAmount(valid)
		require.NoError(t, err, valid)
	}
	for _, invalid := range []string{"", "00", "01", "+1", "-1", " 1", "1.0"} {
		_, err := types.ParseNonNegativeAmount(invalid)
		require.Error(t, err, invalid)
	}
}

func TestScheduleIDAndDigestsAreStable(t *testing.T) {
	id := types.FormatScheduleID(42)
	require.Equal(t, "schedule-00000000000000000042", id)
	parsed, err := types.ParseScheduleID(id)
	require.NoError(t, err)
	require.Equal(t, uint64(42), parsed)
	require.Equal(t,
		"ac1011bae29c4cc20a564c2038a48f20edb73bd469eeb83b30c7dadad136cc92",
		types.OccurrenceID("chain-1", id, 2, 3, 100),
	)
	require.Equal(t,
		"9cb1dc823f3b71c7713bb17126dc4fdf0a629ad7c8f671cdd8994a643eded06f",
		types.ActionDigest("zrn1recipient", "7", "100000"),
	)
}

func TestValidateTermsBoundariesAndOverflow(t *testing.T) {
	params := types.DefaultParams()
	require.NoError(t, types.ValidateTerms(100, 102, 0, 1, params))
	require.Error(t, types.ValidateTerms(100, 101, 0, 1, params))
	require.Error(t, types.ValidateTerms(100, 102, 1, 2, params))
	require.NoError(t, types.ValidateTerms(100, 102, 10, 2, params))
	require.Error(t, types.ValidateTerms(100, ^uint64(0)-1, 10, 2, params))
}

func TestGenesisRejectsLiabilityMismatch(t *testing.T) {
	genesis := types.DefaultGenesis()
	genesis.TotalEscrowUzrn = "1"
	require.ErrorContains(t, genesis.Validate(), "does not equal schedule liability")
}

func TestGenesisReceiptLifecycleHeightBoundaries(t *testing.T) {
	const chainID = "schedule-genesis-test-1"
	creator := sdk.AccAddress([]byte("genesis-creator-addr")).String()
	recipient := sdk.AccAddress([]byte("genesis-recipient-a")).String()
	makeGenesis := func(
		status types.ScheduleStatus,
		dueHeight, executedHeight, updatedHeight uint64,
	) *types.GenesisState {
		terminalReason := "all_occurrences_succeeded"
		outcome := types.ExecutionOutcome_EXECUTION_OUTCOME_SUCCEEDED
		failureCode := ""
		if status == types.ScheduleStatus_SCHEDULE_STATUS_FAILED {
			terminalReason = types.FailureCodeBankTransfer
			outcome = types.ExecutionOutcome_EXECUTION_OUTCOME_FAILED_AND_REFUNDED
			failureCode = types.FailureCodeBankTransfer
		} else if status == types.ScheduleStatus_SCHEDULE_STATUS_CANCELLED {
			terminalReason = "cancelled_by_creator"
		}
		id := types.FormatScheduleID(1)
		return &types.GenesisState{
			Params: types.DefaultParams(),
			Schedules: []*types.Schedule{{
				Id:                     id,
				Creator:                creator,
				Revision:               1,
				Status:                 status,
				Recipient:              recipient,
				AmountPerExecutionUzrn: "7",
				ExecutionFeeUzrn:       "3",
				ExecutionCount:         1,
				PrincipalRemainingUzrn: "0",
				FeeRemainingUzrn:       "0",
				CreatedHeight:          100,
				UpdatedHeight:          updatedHeight,
				LastExecutionHeight:    executedHeight,
				TerminalReason:         terminalReason,
			}},
			Receipts: []*types.ExecutionReceipt{{
				OccurrenceId:   types.OccurrenceID(chainID, id, 1, 1, dueHeight),
				ScheduleId:     id,
				Revision:       1,
				Sequence:       1,
				DueHeight:      dueHeight,
				ExecutedHeight: executedHeight,
				Recipient:      recipient,
				AmountUzrn:     "7",
				FeeUzrn:        "3",
				ActionSha256:   types.ActionDigest(recipient, "7", "3"),
				Outcome:        outcome,
				FailureCode:    failureCode,
			}},
			NextScheduleId:  2,
			TotalEscrowUzrn: "0",
		}
	}

	// Stored records do not apply today's mutable minimum delay to historical
	// occurrences. They do require the due height to follow creation, while
	// allowing both on-time and delayed execution.
	require.NoError(t, makeGenesis(types.ScheduleStatus_SCHEDULE_STATUS_COMPLETED, 101, 101, 101).ValidateForChainID(chainID))
	require.NoError(t, makeGenesis(types.ScheduleStatus_SCHEDULE_STATUS_COMPLETED, 101, 102, 102).ValidateForChainID(chainID))
	require.ErrorContains(
		t,
		makeGenesis(types.ScheduleStatus_SCHEDULE_STATUS_COMPLETED, 100, 102, 102).ValidateForChainID(chainID),
		"must follow",
	)

	// Completed and failed schedules are written by occurrence processing, so
	// updated_height is exactly the final execution height. Cancellation may
	// legitimately update an already-executed schedule at a later height.
	for _, status := range []types.ScheduleStatus{
		types.ScheduleStatus_SCHEDULE_STATUS_COMPLETED,
		types.ScheduleStatus_SCHEDULE_STATUS_FAILED,
	} {
		require.ErrorContains(t, makeGenesis(status, 101, 102, 103).Validate(), "must equal")
	}
	require.NoError(t, makeGenesis(types.ScheduleStatus_SCHEDULE_STATUS_CANCELLED, 101, 102, 103).ValidateForChainID(chainID))

	// Cancellation is not occurrence terminalization. A creator may execute at
	// revision 1, amend the remaining terms at revision 2, and then cancel, so
	// the final receipt is legitimately older than the terminal schedule.
	cancelledAfterAmendment := makeGenesis(types.ScheduleStatus_SCHEDULE_STATUS_CANCELLED, 101, 102, 103)
	cancelledAfterAmendment.Schedules[0].Revision = 2
	cancelledAfterAmendment.Schedules[0].Recipient = sdk.AccAddress([]byte("amended-recipient-aa")).String()
	require.NoError(t, cancelledAfterAmendment.ValidateForChainID(chainID))
}

func TestParamsRejectValuesAboveImmutableCeilings(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*types.Params)
	}{
		{
			name: "schedule delay",
			mutate: func(params *types.Params) {
				params.MinScheduleDelayBlocks = types.MaxSDKBlockHeight + 1
			},
		},
		{
			name: "recurrence interval",
			mutate: func(params *types.Params) {
				params.MinIntervalBlocks = types.MaxSDKBlockHeight + 1
			},
		},
		{
			name: "executions",
			mutate: func(params *types.Params) {
				params.MaxExecutionsPerSchedule = types.HardMaxExecutionsPerSchedule + 1
			},
		},
		{
			name: "active schedules",
			mutate: func(params *types.Params) {
				params.MaxActiveSchedulesPerCreator = types.HardMaxActiveSchedulesPerCreator + 1
			},
		},
		{
			name: "due records",
			mutate: func(params *types.Params) {
				params.MaxDueRecordsPerBlock = types.HardMaxDueRecordsPerBlock + 1
			},
		},
		{
			name: "query limit",
			mutate: func(params *types.Params) {
				params.MaxQueryLimit = types.HardMaxQueryLimit + 1
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			params := types.DefaultParams()
			test.mutate(params)
			require.Error(t, params.Validate())
		})
	}
}

func TestV2WireIdentitiesDoNotCollideWithRetiredScheduler(t *testing.T) {
	require.Equal(t, "/zerone.schedule.v2.MsgCreateSchedule", sdk.MsgTypeURL(&types.MsgCreateSchedule{}))
	require.Equal(t, "/zerone.schedule.v2.MsgUpdateSchedule", sdk.MsgTypeURL(&types.MsgUpdateSchedule{}))
	require.Equal(t, "/zerone.schedule.v2.MsgCancelSchedule", sdk.MsgTypeURL(&types.MsgCancelSchedule{}))

	amino := codec.NewLegacyAmino()
	types.RegisterCodec(amino)
	raw, err := amino.MarshalJSON(&types.MsgCreateSchedule{})
	require.NoError(t, err)
	require.Contains(t, string(raw), `"type":"zerone_message_schedule_v2/CreateSchedule"`)
	require.NotContains(t, string(raw), `"type":"zerone_schedule/CreateSchedule"`)
}
