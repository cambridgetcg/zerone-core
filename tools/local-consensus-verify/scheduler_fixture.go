package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	schedule "github.com/zerone-chain/zerone/x/schedule/types"
)

const (
	fixtureFirstDue         uint64 = 40
	fixtureKillDue          uint64 = 120
	fixtureRecipientInitial int64  = 1000007 // Includes the explicitly synthetic historical payment of 7.
)

type fixtureOccurrence struct {
	ID            uint64
	Sequence      uint32
	Due, Executed uint64
}

// These expected execution heights are prescribed independently of the keeper:
// cap=2 selects IDs 2,3 at 40, ID4 is late at 41 and recurs at 51, not 50.
var fixtureOccurrences = []fixtureOccurrence{
	{2, 1, fixtureFirstDue, fixtureFirstDue},
	{3, 1, fixtureFirstDue, fixtureFirstDue},
	{4, 1, fixtureFirstDue, fixtureFirstDue + 1},
	{4, 2, fixtureFirstDue + 11, fixtureFirstDue + 11},
	{5, 1, fixtureKillDue, fixtureKillDue},
}

func fixtureAddress(label string) sdk.AccAddress {
	digest := sha256.Sum256([]byte("zerone/local-consensus-rehearsal/test-only/" + label))
	return sdk.AccAddress(digest[:20])
}

// fixtureBankBoundaryAddress is a keyless LOCAL test account, funded with exactly
// one uzrn and never spent. Its 20-byte all-FF bank balance key sorts after escrow
// but before the empty-valued denom-address indexes. This known-layout guard lets
// strict ICS23 verify zero escrow after the final payment; it is NOT a general
// proof-compatibility fix or a production solution. Arbitrary bank layouts can
// still hit the pinned ICS23 empty-index-neighbor failure (see diagnostic tests).
func fixtureBankBoundaryAddress() sdk.AccAddress {
	return sdk.AccAddress(bytes.Repeat([]byte{0xff}, 20))
}

func fixtureReceipt(chainID string, s *schedule.Schedule, sequence uint32, due, executed uint64) *schedule.ExecutionReceipt {
	return &schedule.ExecutionReceipt{
		OccurrenceId: schedule.OccurrenceID(chainID, s.Id, s.Revision, sequence, due),
		ScheduleId:   s.Id, Revision: s.Revision, Sequence: sequence,
		DueHeight: due, ExecutedHeight: executed,
		Recipient: s.Recipient, AmountUzrn: s.AmountPerExecutionUzrn, FeeUzrn: s.ExecutionFeeUzrn,
		ActionSha256: schedule.ActionDigest(s.Recipient, s.AmountPerExecutionUzrn, s.ExecutionFeeUzrn),
		Outcome:      schedule.ExecutionOutcome_EXECUTION_OUTCOME_SUCCEEDED,
	}
}

func schedulerFixture(chainID string) *schedule.GenesisState {
	gs := schedule.DefaultGenesis()
	gs.Params.AcceptNewSchedules = false
	gs.Params.MaxDueRecordsPerBlock = 2
	gs.NextScheduleId = 6
	for i, amount := range []int64{7, 11, 13, 17, 19} {
		count := uint32(1)
		var interval uint64
		if i == 3 {
			count, interval = 2, 10
		}
		due := fixtureFirstDue
		if i == 4 {
			due = fixtureKillDue
		}
		s := &schedule.Schedule{
			Id: schedule.FormatScheduleID(uint64(i + 1)), Creator: fixtureAddress("creator").String(), Recipient: fixtureAddress("recipient").String(),
			Status: schedule.ScheduleStatus_SCHEDULE_STATUS_ACTIVE, Revision: 1,
			CreatedHeight: 1, UpdatedHeight: 1, NextExecutionHeight: due,
			IntervalBlocks: interval, RemainingExecutions: count,
			AmountPerExecutionUzrn: strconv.FormatInt(amount, 10), ExecutionFeeUzrn: schedule.DefaultExecutionFeeUzrn,
			PrincipalRemainingUzrn: strconv.FormatInt(amount*int64(count), 10), FeeRemainingUzrn: strconv.FormatInt(100000*int64(count), 10),
		}
		if i == 0 {
			s.Status = schedule.ScheduleStatus_SCHEDULE_STATUS_COMPLETED
			s.UpdatedHeight, s.LastExecutionHeight, s.ExecutionCount = 4, 4, 1
			s.NextExecutionHeight, s.RemainingExecutions = 0, 0
			s.PrincipalRemainingUzrn, s.FeeRemainingUzrn = "0", "0"
			s.TerminalReason = "all_occurrences_succeeded"
			gs.Receipts = append(gs.Receipts, fixtureReceipt(chainID, s, 1, 3, 4))
		}
		gs.Schedules = append(gs.Schedules, s)
	}
	gs.TotalEscrowUzrn = "500077"
	return gs
}

func fixtureExpectation(chainID string, height int64) *schedulerExpectation {
	state := proto.Clone(schedulerFixture(chainID)).(*schedule.GenesisState)
	e := &schedulerExpectation{Schema: schedulerExpectationSchema, ChainID: chainID, StateHeight: height, State: state, RequireEmptyBlock: true}
	paid := int64(0)
	liability := int64(500077)
	for _, occurrence := range fixtureOccurrences {
		s := state.Schedules[occurrence.ID-1]
		r := fixtureReceipt(chainID, s, occurrence.Sequence, occurrence.Due, occurrence.Executed)
		if occurrence.Executed > uint64(height) {
			e.AbsentOccurrences = append(e.AbsentOccurrences, r)
			continue
		}
		amount, _ := strconv.ParseInt(s.AmountPerExecutionUzrn, 10, 64)
		paid += amount
		liability -= amount + 100000
		s.ExecutionCount++
		s.RemainingExecutions--
		s.LastExecutionHeight, s.UpdatedHeight = occurrence.Executed, occurrence.Executed
		s.PrincipalRemainingUzrn = strconv.FormatInt(amount*int64(s.RemainingExecutions), 10)
		s.FeeRemainingUzrn = strconv.FormatInt(100000*int64(s.RemainingExecutions), 10)
		if s.RemainingExecutions == 0 {
			s.Status = schedule.ScheduleStatus_SCHEDULE_STATUS_COMPLETED
			s.NextExecutionHeight = 0
			s.TerminalReason = "all_occurrences_succeeded"
		} else {
			s.NextExecutionHeight = occurrence.Executed + s.IntervalBlocks
		}
		state.Receipts = append(state.Receipts, r)
	}
	// Probe every known due key at every checkpoint, plus the incorrect catch-up
	// height. Membership is generated from active state; consumed/future keys
	// not currently indexed require cryptographic absence.
	for _, d := range []dueExpectation{
		{schedule.FormatScheduleID(2), fixtureFirstDue}, {schedule.FormatScheduleID(3), fixtureFirstDue},
		{schedule.FormatScheduleID(4), fixtureFirstDue}, {schedule.FormatScheduleID(4), fixtureFirstDue + 10},
		{schedule.FormatScheduleID(4), fixtureFirstDue + 11}, {schedule.FormatScheduleID(5), fixtureKillDue},
	} {
		indexed := false
		for _, s := range state.Schedules {
			if s.Id == d.ScheduleID && s.NextExecutionHeight == d.Height {
				indexed = true
			}
		}
		if !indexed {
			e.AbsentDue = append(e.AbsentDue, d)
		}
	}
	// A catch-up-at-50 occurrence must never exist, even after late completion.
	e.AbsentOccurrences = append(e.AbsentOccurrences, fixtureReceipt(chainID, state.Schedules[3], 2, fixtureFirstDue+10, fixtureFirstDue+10))
	state.TotalEscrowUzrn = strconv.FormatInt(liability, 10)
	e.Balances = []balanceExpectation{
		{fixtureAddress("creator").String(), "1000000"},
		{fixtureAddress("recipient").String(), strconv.FormatInt(fixtureRecipientInitial+paid, 10)},
		{authtypes.NewModuleAddress(schedule.ModuleName).String(), state.TotalEscrowUzrn},
		{fixtureBankBoundaryAddress().String(), "1"}, // Never scheduled or spent; known proof-layout boundary.
	}
	return e
}

// marshalSchedulerFixtureGenesis matches the module's encoding/json contract:
// numeric uint64s/enums and string amounts, without a float64 intermediate.
// Override only the admission tag so the harness sees an explicit false instead
// of the generated Params tag omitting it. The remaining fields keep their tags.
func marshalSchedulerFixtureGenesis(gs *schedule.GenesisState) ([]byte, error) {
	type fixtureParams struct {
		*schedule.Params
		AcceptNewSchedules bool `json:"accept_new_schedules"`
	}
	return json.Marshal(struct {
		*schedule.GenesisState
		Params fixtureParams `json:"params"`
	}{
		GenesisState: gs,
		Params:       fixtureParams{Params: gs.Params, AcceptNewSchedules: gs.Params.AcceptNewSchedules},
	})
}

// writeSchedulerFixture only changes a fresh, marked harness scratch directory.
// No creator private key exists; no scheduler instruction is broadcast at runtime.
func writeSchedulerFixture(root, chainID string) error {
	if chainID != "zerone-consensus-rehearsal-1" {
		return fmt.Errorf("scheduler fixture is restricted to the local rehearsal chain ID")
	}
	root, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	resolved, err := filepath.EvalSymlinks(root)
	if err != nil {
		return err
	}
	if !strings.HasPrefix(filepath.Base(resolved), "zerone-consensus-rehearsal.") {
		return fmt.Errorf("fixture root must be a marked rehearsal scratch directory")
	}
	marker, err := os.Lstat(filepath.Join(root, ".zerone-consensus-rehearsal-owned"))
	if err != nil || !marker.Mode().IsRegular() {
		return fmt.Errorf("fixture root has no regular ownership marker")
	}
	genesisPath := filepath.Join(root, "coordinator", "config", "genesis.json")
	for _, p := range []string{filepath.Join(root, "coordinator"), filepath.Dir(genesisPath), genesisPath, filepath.Join(root, "reports")} {
		info, err := os.Lstat(p)
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("refusing symlink fixture path %s", p)
		}
	}
	bz, err := os.ReadFile(genesisPath)
	if err != nil {
		return err
	}
	var genesis map[string]json.RawMessage
	if err := json.Unmarshal(bz, &genesis); err != nil {
		return err
	}
	var actualChainID string
	if err := json.Unmarshal(genesis["chain_id"], &actualChainID); err != nil {
		return err
	}
	if err := verifyExpectedChainID(chainID, actualChainID); err != nil {
		return err
	}
	var appState map[string]json.RawMessage
	if err := json.Unmarshal(genesis["app_state"], &appState); err != nil {
		return err
	}
	var previous schedule.GenesisState
	if err := protojson.Unmarshal(appState[schedule.ModuleName], &previous); err != nil {
		return fmt.Errorf("fresh scheduler genesis missing: %w", err)
	}
	if previous.Params == nil || previous.Params.AcceptNewSchedules || len(previous.Schedules) != 0 || len(previous.Receipts) != 0 || previous.TotalEscrowUzrn != "0" || previous.NextScheduleId != 1 {
		return fmt.Errorf("refusing non-fresh or open scheduler genesis")
	}
	gs := schedulerFixture(chainID)
	if err := gs.ValidateForChainID(chainID); err != nil {
		return err
	}
	appState[schedule.ModuleName], err = marshalSchedulerFixtureGenesis(gs)
	if err != nil {
		return err
	}
	var auth map[string]json.RawMessage
	if err := json.Unmarshal(appState["auth"], &auth); err != nil {
		return err
	}
	var accounts []map[string]any
	if err := json.Unmarshal(auth["accounts"], &accounts); err != nil {
		return err
	}
	var bank map[string]json.RawMessage
	if err := json.Unmarshal(appState["bank"], &bank); err != nil {
		return err
	}
	type coin struct {
		Denom  string `json:"denom"`
		Amount string `json:"amount"`
	}
	type balance struct {
		Address string `json:"address"`
		Coins   []coin `json:"coins"`
	}
	var balances []balance
	if err := json.Unmarshal(bank["balances"], &balances); err != nil {
		return err
	}
	for _, fixture := range []struct{ address, amount, module string }{
		{fixtureAddress("creator").String(), "1000000", ""},
		{fixtureAddress("recipient").String(), strconv.FormatInt(fixtureRecipientInitial, 10), ""},
		{authtypes.NewModuleAddress(schedule.ModuleName).String(), gs.TotalEscrowUzrn, schedule.ModuleName},
		{authtypes.NewModuleAddress(authtypes.FeeCollectorName).String(), "100000", authtypes.FeeCollectorName},
		{fixtureBankBoundaryAddress().String(), "1", ""}, // Included in supply, not in scheduler principal or fees.
	} {
		for _, b := range balances {
			if b.Address == fixture.address {
				return fmt.Errorf("fixture address already funded")
			}
		}
		for _, a := range accounts {
			base := a
			if nested, ok := a["base_account"].(map[string]any); ok {
				base = nested
			}
			if base["address"] == fixture.address {
				return fmt.Errorf("fixture address already has an auth account")
			}
		}
		base := map[string]any{"address": fixture.address, "pub_key": nil, "account_number": "0", "sequence": "0"}
		if fixture.module == "" {
			base["@type"] = "/cosmos.auth.v1beta1.BaseAccount"
			accounts = append(accounts, base)
		} else {
			accounts = append(accounts, map[string]any{"@type": "/cosmos.auth.v1beta1.ModuleAccount", "base_account": base, "name": fixture.module, "permissions": []string{}})
		}
		balances = append(balances, balance{fixture.address, []coin{{schedule.Denom, fixture.amount}}})
	}
	// Recompute all-denomination supply from balances instead of inventing an
	// unbacked module liability or relying on a bank InitGenesis repair.
	sums := map[string]*big.Int{}
	for _, b := range balances {
		for _, c := range b.Coins {
			amount, err := schedule.ParseNonNegativeAmount(c.Amount)
			if err != nil {
				return err
			}
			if sums[c.Denom] == nil {
				sums[c.Denom] = new(big.Int)
			}
			sums[c.Denom].Add(sums[c.Denom], amount)
		}
	}
	coins := sdk.NewCoins()
	for denom, sum := range sums {
		parsed, err := sdk.ParseCoinNormalized(sum.String() + denom)
		if err != nil {
			return err
		}
		coins = coins.Add(parsed)
	}
	bank["balances"], err = json.Marshal(balances)
	if err != nil {
		return err
	}
	bank["supply"], err = json.Marshal(coins)
	if err != nil {
		return err
	}
	auth["accounts"], err = json.Marshal(accounts)
	if err != nil {
		return err
	}
	appState["auth"], err = json.Marshal(auth)
	if err != nil {
		return err
	}
	appState["bank"], err = json.Marshal(bank)
	if err != nil {
		return err
	}
	genesis["app_state"], err = json.Marshal(appState)
	if err != nil {
		return err
	}
	genesis["initial_height"] = json.RawMessage(`"10"`)
	bz, err = json.MarshalIndent(genesis, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(genesisPath, bz, 0600); err != nil {
		return err
	}
	for _, height := range []int64{39, 40, 41, 51, 120} {
		e := fixtureExpectation(chainID, height)
		if _, err := schedulerKeys(e, chainID); err != nil {
			return err
		}
		if err := writeFixtureJSON(filepath.Join(root, "reports", fmt.Sprintf("scheduler-expect-%d.json", height)), e); err != nil {
			return err
		}
	}
	return writeFixtureJSON(filepath.Join(root, "reports", "scheduler-test-manifest.json"), map[string]any{
		"schema": "zerone.local-scheduler-fixture/v1", "test_only": true, "accept_new_schedules": false,
		"history":   "SYNTHETIC TEST-ONLY imported receipt: due 3, executed 4; not observed daemon execution; genesis begins at 10",
		"requester": "synthetic address with no signing key; no runtime scheduler broadcasts",
		"creator":   fixtureAddress("creator").String(), "recipient": fixtureAddress("recipient").String(),
		"escrow_address": authtypes.NewModuleAddress(schedule.ModuleName).String(), "initial_liability_uzrn": gs.TotalEscrowUzrn,
		"recipient_final_uzrn": strconv.FormatInt(fixtureRecipientInitial+77, 10), "due_cap": 2,
		"term_due_height": fixtureFirstDue, "kill_due_height": fixtureKillDue,
		"expected_late_execution_height": fixtureFirstDue + 1, "expected_recurrence_height": fixtureFirstDue + 11,
		"boundary_limit": "Measured process outages across due blocks only; no claim of exact FinalizeBlock/Commit kill boundary or mid-fsync recovery",
		"bank_proof_boundary": map[string]any{
			"test_only": true, "address": fixtureBankBoundaryAddress().String(), "amount_uzrn": "1",
			"keyless": true, "never_spent": true,
			"reason": "LOCAL known-layout guard: 20-byte all-FF address places a nonempty bank balance leaf after escrow and before empty-valued denom-address indexes; extra coin is included in supply, not scheduled principal or fees",
			"limit":  "NOT general proof compatibility or a production solution: pinned SDK/ICS23 still rejects zero-balance absence in arbitrary layouts with an empty index neighbor; diagnostic rejection tests remain unchanged. No verifier relaxation, range-completeness or live-state proof claim",
		},
	})
}

func writeFixtureJSON(path string, value any) error {
	bz, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	_, writeErr := f.Write(append(bz, '\n'))
	closeErr := f.Close()
	if writeErr != nil {
		return writeErr
	}
	return closeErr
}
