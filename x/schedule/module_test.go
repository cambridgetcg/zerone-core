package schedule

import (
	"bytes"
	"encoding/json"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/zerone-chain/zerone/x/schedule/types"
)

func TestModuleGenesisStrictDecoderRejectsUnknownFields(t *testing.T) {
	makeUnknown := func(t *testing.T, nested bool) json.RawMessage {
		t.Helper()
		raw, err := json.Marshal(types.DefaultGenesis())
		require.NoError(t, err)
		var value map[string]any
		decoder := json.NewDecoder(bytes.NewReader(raw))
		decoder.UseNumber()
		require.NoError(t, decoder.Decode(&value))
		if nested {
			value["params"].(map[string]any)["future_consensus_field"] = true
		} else {
			value["future_consensus_field"] = true
		}
		raw, err = json.Marshal(value)
		require.NoError(t, err)
		return raw
	}

	basic := AppModuleBasic{}
	defaultRaw := AppModuleBasic{}.DefaultGenesis(nil)
	duplicateParams := append([]byte(`{"params":null,`), defaultRaw[1:]...)
	for _, test := range []struct {
		name string
		raw  json.RawMessage
	}{
		{name: "top level", raw: makeUnknown(t, false)},
		{name: "nested params", raw: makeUnknown(t, true)},
		{name: "duplicate key", raw: duplicateParams},
		{name: "trailing value", raw: append(defaultRaw, []byte(` {}`)...)},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := basic.ValidateGenesis(nil, nil, test.raw)
			require.Error(t, err)
			if test.name == "trailing value" {
				require.ErrorContains(t, err, "multiple JSON values")
			} else if test.name == "duplicate key" {
				require.ErrorContains(t, err, `duplicate JSON object key "params"`)
			} else {
				require.ErrorContains(t, err, "unknown field")
			}
		})
	}

	unknown := makeUnknown(t, false)
	require.PanicsWithValue(
		t,
		`unmarshal schedule genesis: json: unknown field "future_consensus_field"`,
		func() {
			(AppModule{}).InitGenesis(sdk.Context{}, nil, unknown)
		},
	)
}

func TestStrictModuleGenesisDecodeRoundTrip(t *testing.T) {
	const chainID = "module-round-trip-1"
	id := types.FormatScheduleID(1)
	creator := sdk.AccAddress(bytes.Repeat([]byte{0x31}, 20)).String()
	recipient := sdk.AccAddress(bytes.Repeat([]byte{0x32}, 20)).String()
	genesis := types.DefaultGenesis()
	genesis.Schedules = []*types.Schedule{{
		Id:                     id,
		Creator:                creator,
		Revision:               1,
		Status:                 types.ScheduleStatus_SCHEDULE_STATUS_COMPLETED,
		Recipient:              recipient,
		AmountPerExecutionUzrn: "7",
		ExecutionFeeUzrn:       "3",
		ExecutionCount:         1,
		PrincipalRemainingUzrn: "0",
		FeeRemainingUzrn:       "0",
		CreatedHeight:          10,
		UpdatedHeight:          12,
		LastExecutionHeight:    12,
		TerminalReason:         "all_occurrences_succeeded",
	}}
	genesis.Receipts = []*types.ExecutionReceipt{{
		OccurrenceId:   types.OccurrenceID(chainID, id, 1, 1, 12),
		ScheduleId:     id,
		Revision:       1,
		Sequence:       1,
		DueHeight:      12,
		ExecutedHeight: 12,
		Recipient:      recipient,
		AmountUzrn:     "7",
		FeeUzrn:        "3",
		ActionSha256:   types.ActionDigest(recipient, "7", "3"),
		Outcome:        types.ExecutionOutcome_EXECUTION_OUTCOME_SUCCEEDED,
	}}
	genesis.NextScheduleId = 2
	require.NoError(t, genesis.ValidateForChainID(chainID))

	raw, err := json.Marshal(genesis)
	require.NoError(t, err)
	decoded, err := decodeGenesisStrict(raw)
	require.NoError(t, err)
	require.NoError(t, decoded.ValidateForChainID(chainID))
	require.True(t, proto.Equal(genesis, decoded))
	reencoded, err := json.Marshal(decoded)
	require.NoError(t, err)
	require.JSONEq(t, string(raw), string(reencoded))
}
