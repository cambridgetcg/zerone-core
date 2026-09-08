package types

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math/big"
	"regexp"
	"strconv"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	Denom                   = "uzrn"
	OccurrenceDomain        = "zerone/message_schedule/occurrence/v1"
	ActionDomain            = "zerone/message_schedule/action/v1"
	FailureCodeBankTransfer = "bank_transfer_failed"
	FailureCodeNextHeight   = "next_height_unrepresentable"
	// FailureCodeEscrowInvariant is reserved for diagnostics, not durable
	// receipts. An escrow invariant failure must fail-stop because the module
	// cannot truthfully claim that the remaining liability was refunded.
	FailureCodeEscrowInvariant = "escrow_invariant"
	DefaultExecutionFeeUzrn    = "100000"
	DefaultMaxTransferUzrn     = "1000000000000"

	// Immutable ceilings keep governance and imported genesis from turning a
	// bounded parameter into unbounded consensus work or allocation.
	HardMaxExecutionsPerSchedule     uint32 = 10_000
	HardMaxActiveSchedulesPerCreator uint32 = 128
	HardMaxDueRecordsPerBlock        uint32 = 256
	HardMaxQueryLimit                uint32 = 1_000
	MaxSDKBlockHeight                uint64 = uint64(^uint64(0) >> 1)
)

var canonicalID = regexp.MustCompile(`^schedule-([0-9]{20})$`)

func DefaultParams() *Params {
	return &Params{
		AcceptNewSchedules:           false,
		MinScheduleDelayBlocks:       2,
		MinIntervalBlocks:            10,
		MaxExecutionsPerSchedule:     365,
		MaxActiveSchedulesPerCreator: 32,
		MaxDueRecordsPerBlock:        64,
		MaxQueryLimit:                100,
		ExecutionFeeUzrn:             DefaultExecutionFeeUzrn,
		MaxTransferPerExecutionUzrn:  DefaultMaxTransferUzrn,
	}
}

func (p *Params) Validate() error {
	if p == nil {
		return fmt.Errorf("params are required")
	}
	if p.MinScheduleDelayBlocks == 0 {
		return fmt.Errorf("min_schedule_delay_blocks must be positive")
	}
	if p.MinScheduleDelayBlocks > MaxSDKBlockHeight {
		return fmt.Errorf("min_schedule_delay_blocks exceeds the SDK block-height range")
	}
	if p.MinIntervalBlocks == 0 {
		return fmt.Errorf("min_interval_blocks must be positive")
	}
	if p.MinIntervalBlocks > MaxSDKBlockHeight {
		return fmt.Errorf("min_interval_blocks exceeds the SDK block-height range")
	}
	if p.MaxExecutionsPerSchedule == 0 {
		return fmt.Errorf("max_executions_per_schedule must be positive")
	}
	if p.MaxExecutionsPerSchedule > HardMaxExecutionsPerSchedule {
		return fmt.Errorf("max_executions_per_schedule exceeds hard maximum %d", HardMaxExecutionsPerSchedule)
	}
	if p.MaxActiveSchedulesPerCreator == 0 {
		return fmt.Errorf("max_active_schedules_per_creator must be positive")
	}
	if p.MaxActiveSchedulesPerCreator > HardMaxActiveSchedulesPerCreator {
		return fmt.Errorf("max_active_schedules_per_creator exceeds hard maximum %d", HardMaxActiveSchedulesPerCreator)
	}
	if p.MaxDueRecordsPerBlock == 0 {
		return fmt.Errorf("max_due_records_per_block must be positive")
	}
	if p.MaxDueRecordsPerBlock > HardMaxDueRecordsPerBlock {
		return fmt.Errorf("max_due_records_per_block exceeds hard maximum %d", HardMaxDueRecordsPerBlock)
	}
	if p.MaxQueryLimit == 0 || p.MaxQueryLimit > HardMaxQueryLimit {
		return fmt.Errorf("max_query_limit must be in [1,%d]", HardMaxQueryLimit)
	}
	if _, err := ParsePositiveAmount(p.ExecutionFeeUzrn); err != nil {
		return fmt.Errorf("execution_fee_uzrn: %w", err)
	}
	if _, err := ParsePositiveAmount(p.MaxTransferPerExecutionUzrn); err != nil {
		return fmt.Errorf("max_transfer_per_execution_uzrn: %w", err)
	}
	fee, _ := ParsePositiveAmount(p.ExecutionFeeUzrn)
	maxTransfer, _ := ParsePositiveAmount(p.MaxTransferPerExecutionUzrn)
	count := new(big.Int).SetUint64(uint64(p.MaxExecutionsPerSchedule))
	maxLiability := new(big.Int).Mul(new(big.Int).Add(maxTransfer, fee), count)
	if maxLiability.BitLen() > 256 {
		return fmt.Errorf("maximum per-schedule liability exceeds the SDK 256-bit coin range")
	}
	return nil
}

func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Params:          DefaultParams(),
		Schedules:       []*Schedule{},
		Receipts:        []*ExecutionReceipt{},
		NextScheduleId:  1,
		TotalEscrowUzrn: "0",
	}
}

func (gs *GenesisState) Validate() error {
	if gs == nil {
		return fmt.Errorf("genesis is required")
	}
	if err := gs.Params.Validate(); err != nil {
		return fmt.Errorf("invalid params: %w", err)
	}
	// MaxUint64 is the allocator's durable exhausted sentinel: the final
	// allocatable ID is MaxUint64-1, after which new creates fail closed.  It
	// must remain exportable and importable so exhaustion does not strand an
	// otherwise valid chain state.
	if gs.NextScheduleId == 0 {
		return fmt.Errorf("next_schedule_id must be positive")
	}
	declared, err := ParseNonNegativeAmount(gs.TotalEscrowUzrn)
	if err != nil {
		return fmt.Errorf("total_escrow_uzrn: %w", err)
	}
	seenSchedules := make(map[string]*Schedule, len(gs.Schedules))
	activeByCreator := make(map[string]uint32)
	liability := new(big.Int)
	var maxID uint64
	for i, schedule := range gs.Schedules {
		if err := ValidateStoredSchedule(schedule, gs.Params); err != nil {
			return fmt.Errorf("schedule[%d]: %w", i, err)
		}
		if _, exists := seenSchedules[schedule.Id]; exists {
			return fmt.Errorf("duplicate schedule id %q", schedule.Id)
		}
		seenSchedules[schedule.Id] = schedule
		if schedule.Status == ScheduleStatus_SCHEDULE_STATUS_ACTIVE {
			activeByCreator[schedule.Creator]++
			if activeByCreator[schedule.Creator] > HardMaxActiveSchedulesPerCreator {
				return fmt.Errorf(
					"creator %s exceeds hard active schedule maximum %d",
					schedule.Creator,
					HardMaxActiveSchedulesPerCreator,
				)
			}
		}
		idNum, err := ParseScheduleID(schedule.Id)
		if err != nil {
			return err
		}
		if idNum > maxID {
			maxID = idNum
		}
		principal, _ := ParseNonNegativeAmount(schedule.PrincipalRemainingUzrn)
		fees, _ := ParseNonNegativeAmount(schedule.FeeRemainingUzrn)
		liability.Add(liability, principal).Add(liability, fees)
	}
	if gs.NextScheduleId <= maxID {
		return fmt.Errorf("next_schedule_id %d must be greater than existing id %d", gs.NextScheduleId, maxID)
	}
	if liability.Cmp(declared) != 0 {
		return fmt.Errorf("total_escrow_uzrn %s does not equal schedule liability %s", declared, liability)
	}
	seenOccurrences := make(map[string]bool, len(gs.Receipts))
	seenSequences := make(map[string]bool, len(gs.Receipts))
	receiptsBySchedule := make(map[string]map[uint32]*ExecutionReceipt)
	for i, receipt := range gs.Receipts {
		if err := ValidateReceipt(receipt); err != nil {
			return fmt.Errorf("receipt[%d]: %w", i, err)
		}
		if seenOccurrences[receipt.OccurrenceId] {
			return fmt.Errorf("duplicate occurrence_id %q", receipt.OccurrenceId)
		}
		seenOccurrences[receipt.OccurrenceId] = true
		sequenceKey := fmt.Sprintf("%s/%d", receipt.ScheduleId, receipt.Sequence)
		if seenSequences[sequenceKey] {
			return fmt.Errorf("duplicate receipt sequence %s", sequenceKey)
		}
		seenSequences[sequenceKey] = true
		schedule, exists := seenSchedules[receipt.ScheduleId]
		if !exists {
			return fmt.Errorf("receipt references unknown schedule %q", receipt.ScheduleId)
		}
		if receipt.DueHeight <= schedule.CreatedHeight {
			return fmt.Errorf(
				"receipt due_height %d must follow schedule %s created_height %d",
				receipt.DueHeight,
				receipt.ScheduleId,
				schedule.CreatedHeight,
			)
		}
		if receipt.Sequence > schedule.ExecutionCount {
			return fmt.Errorf("receipt sequence %d exceeds schedule %s execution_count %d", receipt.Sequence, receipt.ScheduleId, schedule.ExecutionCount)
		}
		if receipt.Revision > schedule.Revision {
			return fmt.Errorf("receipt revision %d exceeds schedule %s revision %d", receipt.Revision, receipt.ScheduleId, schedule.Revision)
		}
		bySequence := receiptsBySchedule[receipt.ScheduleId]
		if bySequence == nil {
			bySequence = make(map[uint32]*ExecutionReceipt)
			receiptsBySchedule[receipt.ScheduleId] = bySequence
		}
		bySequence[receipt.Sequence] = receipt
	}
	for _, schedule := range gs.Schedules {
		bySequence := receiptsBySchedule[schedule.Id]
		if len(bySequence) != int(schedule.ExecutionCount) {
			return fmt.Errorf(
				"schedule %s execution_count %d does not equal receipt count %d",
				schedule.Id,
				schedule.ExecutionCount,
				len(bySequence),
			)
		}
		var previous *ExecutionReceipt
		for sequence := uint32(1); sequence <= schedule.ExecutionCount; sequence++ {
			receipt, exists := bySequence[sequence]
			if !exists {
				return fmt.Errorf("schedule %s is missing receipt sequence %d", schedule.Id, sequence)
			}
			if previous != nil {
				if receipt.Revision < previous.Revision {
					return fmt.Errorf("schedule %s receipt revisions decrease at sequence %d", schedule.Id, sequence)
				}
				if receipt.DueHeight <= previous.ExecutedHeight || receipt.ExecutedHeight <= previous.ExecutedHeight {
					return fmt.Errorf("schedule %s receipt heights do not advance at sequence %d", schedule.Id, sequence)
				}
			}
			last := sequence == schedule.ExecutionCount
			occurrenceTerminal := last && (schedule.Status == ScheduleStatus_SCHEDULE_STATUS_COMPLETED ||
				schedule.Status == ScheduleStatus_SCHEDULE_STATUS_FAILED)
			if occurrenceTerminal && receipt.Revision != schedule.Revision {
				return fmt.Errorf(
					"schedule %s terminal receipt revision %d does not equal current revision %d",
					schedule.Id,
					receipt.Revision,
					schedule.Revision,
				)
			}
			if (occurrenceTerminal || receipt.Revision == schedule.Revision) &&
				!receiptMatchesScheduleTerms(receipt, schedule) {
				return fmt.Errorf(
					"schedule %s receipt sequence %d at current revision does not match stored action terms",
					schedule.Id,
					sequence,
				)
			}
			if receipt.Outcome == ExecutionOutcome_EXECUTION_OUTCOME_FAILED_AND_REFUNDED {
				if !last || schedule.Status != ScheduleStatus_SCHEDULE_STATUS_FAILED {
					return fmt.Errorf("schedule %s has a non-terminal or non-failed refunded receipt at sequence %d", schedule.Id, sequence)
				}
			} else if last && schedule.Status == ScheduleStatus_SCHEDULE_STATUS_FAILED {
				return fmt.Errorf("failed schedule %s does not end in a failed-and-refunded receipt", schedule.Id)
			}
			previous = receipt
		}
		if schedule.ExecutionCount == 0 {
			if schedule.LastExecutionHeight != 0 {
				return fmt.Errorf("schedule %s has last_execution_height without a receipt", schedule.Id)
			}
		} else if previous == nil || schedule.LastExecutionHeight != previous.ExecutedHeight {
			return fmt.Errorf("schedule %s last_execution_height does not match its final receipt", schedule.Id)
		}
		if schedule.Status == ScheduleStatus_SCHEDULE_STATUS_FAILED &&
			previous != nil && schedule.TerminalReason != previous.FailureCode {
			return fmt.Errorf("failed schedule %s terminal_reason does not match its final receipt", schedule.Id)
		}
		if schedule.Status == ScheduleStatus_SCHEDULE_STATUS_COMPLETED && schedule.ExecutionCount == 0 {
			return fmt.Errorf("completed schedule %s has no execution receipt", schedule.Id)
		}
		if schedule.Status == ScheduleStatus_SCHEDULE_STATUS_FAILED && schedule.ExecutionCount == 0 {
			return fmt.Errorf("failed schedule %s has no failure receipt", schedule.Id)
		}
	}
	return nil
}

// ValidateForChainID adds the chain-domain check that cannot be performed by
// offline module genesis validation. InitGenesis calls it before writing any
// indexes, preventing a forged imported occurrence from poisoning a different
// schedule's future global occurrence key.
func (gs *GenesisState) ValidateForChainID(chainID string) error {
	if err := gs.Validate(); err != nil {
		return err
	}
	if len(gs.Receipts) > 0 && chainID == "" {
		return fmt.Errorf("chain id is required to validate imported receipts")
	}
	for i, receipt := range gs.Receipts {
		expected := OccurrenceID(
			chainID,
			receipt.ScheduleId,
			receipt.Revision,
			receipt.Sequence,
			receipt.DueHeight,
		)
		if receipt.OccurrenceId != expected {
			return fmt.Errorf("receipt[%d] occurrence_id does not match its chain-bound occurrence", i)
		}
	}
	return nil
}

func ValidateStoredSchedule(s *Schedule, _ *Params) error {
	if s == nil {
		return fmt.Errorf("schedule is nil")
	}
	if _, err := ParseScheduleID(s.Id); err != nil {
		return err
	}
	creator, err := sdk.AccAddressFromBech32(s.Creator)
	if err != nil {
		return fmt.Errorf("invalid creator: %w", err)
	}
	if creator.String() != s.Creator {
		return fmt.Errorf("creator must use canonical bech32 encoding")
	}
	recipient, err := sdk.AccAddressFromBech32(s.Recipient)
	if err != nil {
		return fmt.Errorf("invalid recipient: %w", err)
	}
	if recipient.String() != s.Recipient {
		return fmt.Errorf("recipient must use canonical bech32 encoding")
	}
	if s.Revision == 0 {
		return fmt.Errorf("revision must be positive")
	}
	if s.Revision == ^uint64(0) {
		return fmt.Errorf("revision must leave amendment capacity")
	}
	if uint64(s.ExecutionCount)+uint64(s.RemainingExecutions) > uint64(HardMaxExecutionsPerSchedule) {
		return fmt.Errorf("lifetime executions exceed hard maximum %d", HardMaxExecutionsPerSchedule)
	}
	if s.CreatedHeight == 0 || s.CreatedHeight > MaxSDKBlockHeight {
		return fmt.Errorf("created_height must be within the positive SDK block-height range")
	}
	if s.UpdatedHeight < s.CreatedHeight || s.UpdatedHeight > MaxSDKBlockHeight {
		return fmt.Errorf("updated_height must be within the SDK block-height range and not precede creation")
	}
	if s.LastExecutionHeight > MaxSDKBlockHeight {
		return fmt.Errorf("last_execution_height exceeds the SDK block-height range")
	}
	if s.ExecutionCount == 0 && s.LastExecutionHeight != 0 {
		return fmt.Errorf("last_execution_height requires a prior execution")
	}
	if s.ExecutionCount > 0 && (s.LastExecutionHeight == 0 || s.LastExecutionHeight > s.UpdatedHeight) {
		return fmt.Errorf("executed schedule requires a valid last_execution_height")
	}
	amount, err := ParsePositiveAmount(s.AmountPerExecutionUzrn)
	if err != nil {
		return fmt.Errorf("amount_per_execution_uzrn: %w", err)
	}
	fee, err := ParsePositiveAmount(s.ExecutionFeeUzrn)
	if err != nil {
		return fmt.Errorf("execution_fee_uzrn: %w", err)
	}
	principal, err := ParseNonNegativeAmount(s.PrincipalRemainingUzrn)
	if err != nil {
		return fmt.Errorf("principal_remaining_uzrn: %w", err)
	}
	fees, err := ParseNonNegativeAmount(s.FeeRemainingUzrn)
	if err != nil {
		return fmt.Errorf("fee_remaining_uzrn: %w", err)
	}
	switch s.Status {
	case ScheduleStatus_SCHEDULE_STATUS_ACTIVE:
		if s.RemainingExecutions == 0 || s.NextExecutionHeight == 0 {
			return fmt.Errorf("active schedule requires remaining executions and next height")
		}
		if s.NextExecutionHeight > MaxSDKBlockHeight || s.NextExecutionHeight <= s.UpdatedHeight {
			return fmt.Errorf("active next_execution_height must be future and within the SDK block-height range")
		}
		if s.IntervalBlocks > MaxSDKBlockHeight {
			return fmt.Errorf("interval_blocks exceeds the SDK block-height range")
		}
		if s.TerminalReason != "" {
			return fmt.Errorf("active schedule has terminal_reason")
		}
		if s.RemainingExecutions > 1 && s.IntervalBlocks == 0 {
			return fmt.Errorf("recurring schedule requires a positive interval")
		}
		expectedPrincipal := new(big.Int).Mul(amount, new(big.Int).SetUint64(uint64(s.RemainingExecutions)))
		expectedFees := new(big.Int).Mul(fee, new(big.Int).SetUint64(uint64(s.RemainingExecutions)))
		if principal.Cmp(expectedPrincipal) != 0 || fees.Cmp(expectedFees) != 0 {
			return fmt.Errorf("active escrow does not match remaining occurrences")
		}
	case ScheduleStatus_SCHEDULE_STATUS_COMPLETED,
		ScheduleStatus_SCHEDULE_STATUS_CANCELLED,
		ScheduleStatus_SCHEDULE_STATUS_FAILED:
		if s.RemainingExecutions != 0 || s.NextExecutionHeight != 0 || principal.Sign() != 0 || fees.Sign() != 0 {
			return fmt.Errorf("terminal schedule retains pending work or escrow")
		}
		if s.TerminalReason == "" {
			return fmt.Errorf("terminal schedule requires terminal_reason")
		}
		switch s.Status {
		case ScheduleStatus_SCHEDULE_STATUS_COMPLETED:
			if s.TerminalReason != "all_occurrences_succeeded" {
				return fmt.Errorf("completed schedule has invalid terminal_reason")
			}
			if s.UpdatedHeight != s.LastExecutionHeight {
				return fmt.Errorf("completed schedule updated_height must equal last_execution_height")
			}
		case ScheduleStatus_SCHEDULE_STATUS_CANCELLED:
			if s.TerminalReason != "cancelled_by_creator" {
				return fmt.Errorf("cancelled schedule has invalid terminal_reason")
			}
		case ScheduleStatus_SCHEDULE_STATUS_FAILED:
			if !IsKnownFailureCode(s.TerminalReason) {
				return fmt.Errorf("failed schedule has unknown terminal_reason")
			}
			if s.UpdatedHeight != s.LastExecutionHeight {
				return fmt.Errorf("failed schedule updated_height must equal last_execution_height")
			}
		}
	default:
		return fmt.Errorf("invalid schedule status %s", s.Status)
	}
	return nil
}

func ValidateReceipt(r *ExecutionReceipt) error {
	if r == nil {
		return fmt.Errorf("receipt is nil")
	}
	if err := validateDigest(r.OccurrenceId); err != nil {
		return fmt.Errorf("occurrence_id: %w", err)
	}
	if _, err := ParseScheduleID(r.ScheduleId); err != nil {
		return err
	}
	if r.Revision == 0 || r.Sequence == 0 || r.DueHeight == 0 || r.ExecutedHeight == 0 {
		return fmt.Errorf("revision, sequence, due_height, and executed_height must be positive")
	}
	if r.ExecutedHeight < r.DueHeight {
		return fmt.Errorf("executed_height cannot precede due_height")
	}
	recipient, err := sdk.AccAddressFromBech32(r.Recipient)
	if err != nil {
		return fmt.Errorf("invalid recipient: %w", err)
	}
	if recipient.String() != r.Recipient {
		return fmt.Errorf("recipient must use canonical bech32 encoding")
	}
	if _, err := ParsePositiveAmount(r.AmountUzrn); err != nil {
		return fmt.Errorf("amount_uzrn: %w", err)
	}
	if _, err := ParsePositiveAmount(r.FeeUzrn); err != nil {
		return fmt.Errorf("fee_uzrn: %w", err)
	}
	if err := validateDigest(r.ActionSha256); err != nil {
		return fmt.Errorf("action_sha256: %w", err)
	}
	if r.ActionSha256 != ActionDigest(r.Recipient, r.AmountUzrn, r.FeeUzrn) {
		return fmt.Errorf("action_sha256 does not match receipt action")
	}
	if r.Revision == ^uint64(0) {
		return fmt.Errorf("receipt revision must leave amendment capacity")
	}
	if r.DueHeight > MaxSDKBlockHeight || r.ExecutedHeight > MaxSDKBlockHeight {
		return fmt.Errorf("receipt heights exceed the SDK block-height range")
	}
	switch r.Outcome {
	case ExecutionOutcome_EXECUTION_OUTCOME_SUCCEEDED:
		if r.FailureCode != "" {
			return fmt.Errorf("successful receipt has failure_code")
		}
	case ExecutionOutcome_EXECUTION_OUTCOME_FAILED_AND_REFUNDED:
		if !IsKnownFailureCode(r.FailureCode) {
			return fmt.Errorf("failed receipt requires a known failure_code")
		}
	default:
		return fmt.Errorf("invalid execution outcome %s", r.Outcome)
	}
	return nil
}

func IsKnownFailureCode(code string) bool {
	switch code {
	case FailureCodeBankTransfer, FailureCodeNextHeight:
		return true
	default:
		return false
	}
}

func receiptMatchesScheduleTerms(receipt *ExecutionReceipt, schedule *Schedule) bool {
	return receipt.Recipient == schedule.Recipient &&
		receipt.AmountUzrn == schedule.AmountPerExecutionUzrn &&
		receipt.FeeUzrn == schedule.ExecutionFeeUzrn
}

func ValidateTerms(currentHeight, executionHeight, interval uint64, count uint32, params *Params) error {
	if count == 0 || count > params.MaxExecutionsPerSchedule {
		return fmt.Errorf("execution count must be in [1,%d]", params.MaxExecutionsPerSchedule)
	}
	if currentHeight > MaxSDKBlockHeight || executionHeight > MaxSDKBlockHeight {
		return fmt.Errorf("execution heights must fit the signed SDK block-height range")
	}
	if currentHeight > MaxSDKBlockHeight-params.MinScheduleDelayBlocks || executionHeight < currentHeight+params.MinScheduleDelayBlocks {
		return fmt.Errorf("execution height must be at least %d blocks ahead", params.MinScheduleDelayBlocks)
	}
	if err := ValidateRecurrence(interval, count, params); err != nil {
		return err
	}
	if count > 1 {
		remainingIntervals := uint64(count - 1)
		if interval > MaxSDKBlockHeight/remainingIntervals || executionHeight > MaxSDKBlockHeight-(interval*remainingIntervals) {
			return fmt.Errorf("recurrence heights exceed the signed SDK block-height range")
		}
	}
	return nil
}

func ValidateRecurrence(interval uint64, count uint32, params *Params) error {
	if count == 1 {
		if interval != 0 {
			return fmt.Errorf("one-shot schedule must have interval_blocks=0")
		}
		return nil
	}
	if interval < params.MinIntervalBlocks {
		return fmt.Errorf("recurring schedule interval must be at least %d blocks", params.MinIntervalBlocks)
	}
	return nil
}

func ParsePositiveAmount(value string) (*big.Int, error) {
	amount, err := ParseNonNegativeAmount(value)
	if err != nil {
		return nil, err
	}
	if amount.Sign() <= 0 {
		return nil, fmt.Errorf("must be a positive base-10 integer")
	}
	return amount, nil
}

func ParseNonNegativeAmount(value string) (*big.Int, error) {
	if value == "" || strings.TrimLeft(value, "0") != value && value != "0" {
		return nil, fmt.Errorf("must use canonical base-10 integer encoding")
	}
	for _, ch := range value {
		if ch < '0' || ch > '9' {
			return nil, fmt.Errorf("must use canonical base-10 integer encoding")
		}
	}
	amount := new(big.Int)
	if _, ok := amount.SetString(value, 10); !ok || amount.Sign() < 0 {
		return nil, fmt.Errorf("must be a non-negative base-10 integer")
	}
	if amount.BitLen() > 256 {
		return nil, fmt.Errorf("exceeds the SDK 256-bit coin range")
	}
	return amount, nil
}

func FormatScheduleID(id uint64) string {
	return fmt.Sprintf("schedule-%020d", id)
}

func ParseScheduleID(id string) (uint64, error) {
	match := canonicalID.FindStringSubmatch(id)
	if match == nil {
		return 0, fmt.Errorf("invalid canonical schedule id %q", id)
	}
	n, err := strconv.ParseUint(match[1], 10, 64)
	if err != nil || n == 0 {
		return 0, fmt.Errorf("invalid canonical schedule id %q", id)
	}
	return n, nil
}

func OccurrenceID(chainID, scheduleID string, revision uint64, sequence uint32, dueHeight uint64) string {
	return digest(OccurrenceDomain, chainID, scheduleID, uint64Bytes(revision), uint32Bytes(sequence), uint64Bytes(dueHeight))
}

func ActionDigest(recipient, amountUzrn, feeUzrn string) string {
	return digest(ActionDomain, recipient, amountUzrn, feeUzrn)
}

func digest(parts ...interface{}) string {
	h := sha256.New()
	for _, part := range parts {
		var value []byte
		switch typed := part.(type) {
		case string:
			value = []byte(typed)
		case []byte:
			value = typed
		default:
			panic("unsupported digest part")
		}
		length := make([]byte, 4)
		binary.BigEndian.PutUint32(length, uint32(len(value)))
		_, _ = h.Write(length)
		_, _ = h.Write(value)
	}
	return hex.EncodeToString(h.Sum(nil))
}

func uint64Bytes(value uint64) []byte {
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, value)
	return bz
}

func uint32Bytes(value uint32) []byte {
	bz := make([]byte, 4)
	binary.BigEndian.PutUint32(bz, value)
	return bz
}

func validateDigest(value string) error {
	if len(value) != sha256.Size*2 {
		return fmt.Errorf("must be 64 lowercase hexadecimal characters")
	}
	decoded, err := hex.DecodeString(value)
	if err != nil || hex.EncodeToString(decoded) != value {
		return fmt.Errorf("must be 64 lowercase hexadecimal characters")
	}
	return nil
}

// ValidateDigest verifies the canonical encoding used by occurrence and action
// identifiers without interpreting the digest as authorization.
func ValidateDigest(value string) error {
	return validateDigest(value)
}

func signer(address string) []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(address)
	return []sdk.AccAddress{addr}
}

func (m *MsgCreateSchedule) GetSigners() []sdk.AccAddress { return signer(m.Creator) }
func (m *MsgUpdateSchedule) GetSigners() []sdk.AccAddress { return signer(m.Creator) }
func (m *MsgCancelSchedule) GetSigners() []sdk.AccAddress { return signer(m.Creator) }
func (m *MsgUpdateParams) GetSigners() []sdk.AccAddress   { return signer(m.Authority) }

func (m *MsgCreateSchedule) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Creator); err != nil {
		return fmt.Errorf("invalid creator: %w", err)
	}
	if _, err := sdk.AccAddressFromBech32(m.Recipient); err != nil {
		return fmt.Errorf("invalid recipient: %w", err)
	}
	if _, err := ParsePositiveAmount(m.AmountPerExecutionUzrn); err != nil {
		return fmt.Errorf("amount_per_execution_uzrn: %w", err)
	}
	if m.FirstExecutionHeight == 0 || m.ExecutionCount == 0 {
		return fmt.Errorf("first_execution_height and execution_count must be positive")
	}
	return nil
}

func (m *MsgUpdateSchedule) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Creator); err != nil {
		return fmt.Errorf("invalid creator: %w", err)
	}
	if _, err := ParseScheduleID(m.ScheduleId); err != nil {
		return err
	}
	if m.ExpectedRevision == 0 {
		return fmt.Errorf("expected_revision must be positive")
	}
	if _, err := sdk.AccAddressFromBech32(m.Recipient); err != nil {
		return fmt.Errorf("invalid recipient: %w", err)
	}
	if _, err := ParsePositiveAmount(m.AmountPerExecutionUzrn); err != nil {
		return fmt.Errorf("amount_per_execution_uzrn: %w", err)
	}
	if m.NextExecutionHeight == 0 || m.RemainingExecutions == 0 {
		return fmt.Errorf("next_execution_height and remaining_executions must be positive")
	}
	return nil
}

func (m *MsgCancelSchedule) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Creator); err != nil {
		return fmt.Errorf("invalid creator: %w", err)
	}
	if _, err := ParseScheduleID(m.ScheduleId); err != nil {
		return err
	}
	if m.ExpectedRevision == 0 {
		return fmt.Errorf("expected_revision must be positive")
	}
	return nil
}

func (m *MsgUpdateParams) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Authority); err != nil {
		return fmt.Errorf("invalid authority: %w", err)
	}
	return m.Params.Validate()
}
