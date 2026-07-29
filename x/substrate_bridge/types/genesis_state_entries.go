package types

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"google.golang.org/protobuf/proto"
)

// witnessPendingRewardGenesis mirrors keeper.WitnessPendingReward without
// importing keeper (which would create a package cycle). The keeper stores this
// record as JSON rather than protobuf.
type witnessPendingRewardGenesis struct {
	AttestationID string `json:"attestation_id"`
	AdapterID     string `json:"adapter_id"`
	Recipient     string `json:"recipient"`
	Amount        string `json:"amount"`
	Deadline      uint64 `json:"deadline"`
}

type genesisAttestationRecord struct {
	index       int
	attestation *ExternalAttestation
}

type genesisLineageRecord struct {
	index int
	edge  *LineageEdge
}

type genesisAccumulatorRecord struct {
	index       int
	accumulator *LineageRoyaltyAccumulator
}

type genesisPendingClaimRecord struct {
	index         int
	claimID       string
	attestationID string
}

type genesisWitnessRewardRecord struct {
	index  int
	reward witnessPendingRewardGenesis
}

type genesisSourceRefRecord struct {
	index         int
	adapterID     string
	sourceID      string
	attestationID string
}

// validateGenesisStateEntries treats StateEntries as a relational snapshot,
// not an arbitrary bag of bytes. Raw storage is necessary to preserve the
// exact source-ref holder and every replay-sensitive index across export, but
// accepting only a key prefix is not enough: keeper code uses IDs embedded in
// values when it mutates or deletes state. A key/value ID mismatch can
// otherwise leave the original READY or witness-deadline entry in place and
// pay it again every block.
//
// The checks below intentionally mirror only persisted keeper invariants. They
// do not re-run admission policy against historical attestations (older
// records may predate current source/hash rules), so a valid export remains
// portable while malformed or internally contradictory state fails closed.
func validateGenesisStateEntries(gs *GenesisState) error {
	entryByKey := make(map[string]*GenesisStateEntry, len(gs.StateEntries))
	adapters := make(map[string]*AdapterRegistration, len(gs.Adapters))
	for _, adapter := range gs.Adapters {
		adapters[adapter.AdapterId] = adapter
	}

	attestations := make(map[string]*ExternalAttestation)
	var attestationRecords []genesisAttestationRecord
	var lineageRecords []genesisLineageRecord
	var accumulatorRecords []genesisAccumulatorRecord
	var pendingClaimRecords []genesisPendingClaimRecord
	var witnessRewardRecords []genesisWitnessRewardRecord
	var sourceRefRecords []genesisSourceRefRecord

	var (
		counterPresent bool
		counter        uint64
		dedupeArmed    bool
	)

	for index, entry := range gs.StateEntries {
		entryByKey[string(entry.Key)] = entry

		switch entry.Key[0] {
		case LineageEdgePrefix[0]:
			var edge LineageEdge
			if err := proto.Unmarshal(entry.Value, &edge); err != nil {
				return fmt.Errorf("state entry at index %d has an invalid lineage edge: %w", index, err)
			}
			if edge.UpstreamAttestationId == "" || edge.DownstreamAttestationId == "" {
				return fmt.Errorf("state entry at index %d has a lineage edge with an empty endpoint", index)
			}
			if !bytes.Equal(entry.Key, LineageEdgeKey(EdgeID(
				edge.UpstreamAttestationId,
				edge.DownstreamAttestationId,
			))) {
				return fmt.Errorf("state entry at index %d lineage edge key does not match its endpoints", index)
			}
			if _, known := CitationType_name[int32(edge.CitationType)]; !known {
				return fmt.Errorf("state entry at index %d has unknown citation type %d", index, edge.CitationType)
			}
			if edge.SettlementPaymentUzrn != "" && !isNonNegativeDecimal(edge.SettlementPaymentUzrn) {
				return fmt.Errorf("state entry at index %d has invalid lineage settlement payment %q", index, edge.SettlementPaymentUzrn)
			}
			lineageRecords = append(lineageRecords, genesisLineageRecord{index: index, edge: &edge})

		case LineageByUpstreamPrefix[0], LineageByDownstreamPrefix[0],
			AttestationByStatusPrefix[0], AttestationPendingClaimsPrefix[0],
			WitnessDeadlineIndexPrefix[0]:
			if !isGenesisIndexMarker(entry.Value) {
				return fmt.Errorf("state entry at index %d index value must be exactly 0x01", index)
			}

		case LineageRoyaltyAccumulatorPrefix[0]:
			var accumulator LineageRoyaltyAccumulator
			if err := proto.Unmarshal(entry.Value, &accumulator); err != nil {
				return fmt.Errorf("state entry at index %d has an invalid lineage accumulator: %w", index, err)
			}
			if accumulator.AttestationId == "" ||
				!bytes.Equal(entry.Key, LineageRoyaltyAccumulatorKey(accumulator.AttestationId)) {
				return fmt.Errorf("state entry at index %d lineage accumulator key does not match its attestation_id", index)
			}
			if !isNonNegativeDecimal(accumulator.CumulativeUzrn) {
				return fmt.Errorf("state entry at index %d has invalid cumulative_uzrn %q", index, accumulator.CumulativeUzrn)
			}
			accumulatorRecords = append(accumulatorRecords, genesisAccumulatorRecord{
				index:       index,
				accumulator: &accumulator,
			})

		case ExternalAttestationPrefix[0]:
			var attestation ExternalAttestation
			if err := proto.Unmarshal(entry.Value, &attestation); err != nil {
				return fmt.Errorf("state entry at index %d has an invalid external attestation: %w", index, err)
			}
			if attestation.AttestationId == "" ||
				!bytes.Equal(entry.Key, AttestationKey(attestation.AttestationId)) {
				return fmt.Errorf("state entry at index %d attestation key does not match its attestation_id", index)
			}
			if strings.IndexByte(attestation.AttestationId, 0) >= 0 {
				return fmt.Errorf("state entry at index %d attestation_id contains a NUL byte", index)
			}
			if _, known := AttestationStatus_name[int32(attestation.Status)]; !known {
				return fmt.Errorf("state entry at index %d has unknown attestation status %d", index, attestation.Status)
			}
			if (attestation.Status == AttestationStatus_ATTESTATION_STATUS_AWAITING_RESOLUTION ||
				attestation.Status == AttestationStatus_ATTESTATION_STATUS_READY) &&
				attestation.Link == nil {
				return fmt.Errorf("state entry at index %d has a settleable attestation without a link", index)
			}
			attestations[attestation.AttestationId] = &attestation
			attestationRecords = append(attestationRecords, genesisAttestationRecord{
				index:       index,
				attestation: &attestation,
			})

		case PendingFactIndexPrefix[0]:
			claimID := string(entry.Key[len(PendingFactIndexPrefix):])
			attestationID := string(entry.Value)
			if claimID == "" || attestationID == "" {
				return fmt.Errorf("state entry at index %d has an empty pending claim or attestation id", index)
			}
			pendingClaimRecords = append(pendingClaimRecords, genesisPendingClaimRecord{
				index:         index,
				claimID:       claimID,
				attestationID: attestationID,
			})

		case AttestationIDCounterKey[0]:
			value, consumed := binary.Uvarint(entry.Value)
			if consumed <= 0 || consumed != len(entry.Value) || value == 0 || value == ^uint64(0) {
				return fmt.Errorf("state entry at index %d has an invalid attestation counter encoding", index)
			}
			if canonical := binary.AppendUvarint(nil, value); !bytes.Equal(entry.Value, canonical) {
				return fmt.Errorf("state entry at index %d has a non-canonical attestation counter encoding", index)
			}
			counterPresent = true
			counter = value

		case WitnessPendingRewardPrefix[0]:
			var reward witnessPendingRewardGenesis
			if err := json.Unmarshal(entry.Value, &reward); err != nil {
				return fmt.Errorf("state entry at index %d has an invalid witness reward: %w", index, err)
			}
			if reward.AttestationID == "" ||
				!bytes.Equal(entry.Key, WitnessPendingRewardKey(reward.AttestationID)) {
				return fmt.Errorf("state entry at index %d witness reward key does not match its attestation_id", index)
			}
			if reward.AdapterID == "" || reward.Recipient == "" ||
				!isPositiveDecimal(reward.Amount) || reward.Deadline == 0 {
				return fmt.Errorf("state entry at index %d has incomplete or invalid witness reward fields", index)
			}
			witnessRewardRecords = append(witnessRewardRecords, genesisWitnessRewardRecord{
				index:  index,
				reward: reward,
			})

		case SourceRefPrefix[0]:
			adapterID, sourceID, ok := parseSourceRefGenesisKey(entry.Key)
			if !ok {
				return fmt.Errorf("state entry at index %d has a malformed source-ref key", index)
			}
			sourceRefRecords = append(sourceRefRecords, genesisSourceRefRecord{
				index:         index,
				adapterID:     adapterID,
				sourceID:      sourceID,
				attestationID: string(entry.Value),
			})

		case DedupeArmedKey[0]:
			if !isGenesisIndexMarker(entry.Value) {
				return fmt.Errorf("state entry at index %d dedupe marker must be exactly 0x01", index)
			}
			dedupeArmed = true
		}
	}

	expectedStatusIndexes := make(map[string]struct{}, len(attestationRecords))
	var maxAttestationSequence uint64
	for _, record := range attestationRecords {
		attestation := record.attestation
		statusKey := AttestationByStatusKey(uint8(attestation.Status), attestation.AttestationId)
		if err := requireGenesisIndex(entryByKey, statusKey, "attestation status", record.index); err != nil {
			return err
		}
		expectedStatusIndexes[string(statusKey)] = struct{}{}

		if sequence, ok := generatedAttestationSequence(attestation.AttestationId); ok &&
			sequence > maxAttestationSequence {
			maxAttestationSequence = sequence
		}
	}
	if counterPresent && counter < maxAttestationSequence {
		return fmt.Errorf(
			"attestation counter %d is below stored attestation sequence %d",
			counter,
			maxAttestationSequence,
		)
	}
	if dedupeArmed && len(attestationRecords) > 0 && !counterPresent {
		return fmt.Errorf("armed attestation state must include the attestation counter")
	}
	for index, entry := range gs.StateEntries {
		if entry.Key[0] != AttestationByStatusPrefix[0] {
			continue
		}
		if _, expected := expectedStatusIndexes[string(entry.Key)]; !expected {
			return fmt.Errorf("state entry at index %d is an orphan or mismatched attestation status index", index)
		}
	}

	expectedForwardLineage := make(map[string]struct{}, len(lineageRecords))
	expectedBackwardLineage := make(map[string]struct{}, len(lineageRecords))
	for _, record := range lineageRecords {
		edge := record.edge
		if _, found := attestations[edge.UpstreamAttestationId]; !found {
			return fmt.Errorf("state entry at index %d lineage upstream attestation %q is missing", record.index, edge.UpstreamAttestationId)
		}
		if _, found := attestations[edge.DownstreamAttestationId]; !found {
			return fmt.Errorf("state entry at index %d lineage downstream attestation %q is missing", record.index, edge.DownstreamAttestationId)
		}
		edgeID := EdgeID(edge.UpstreamAttestationId, edge.DownstreamAttestationId)
		forwardKey := LineageByUpstreamKey(edge.UpstreamAttestationId, edgeID)
		backwardKey := LineageByDownstreamKey(edge.DownstreamAttestationId, edgeID)
		if err := requireGenesisIndex(entryByKey, forwardKey, "forward lineage", record.index); err != nil {
			return err
		}
		if err := requireGenesisIndex(entryByKey, backwardKey, "backward lineage", record.index); err != nil {
			return err
		}
		expectedForwardLineage[string(forwardKey)] = struct{}{}
		expectedBackwardLineage[string(backwardKey)] = struct{}{}
	}
	for index, entry := range gs.StateEntries {
		switch entry.Key[0] {
		case LineageByUpstreamPrefix[0]:
			if _, expected := expectedForwardLineage[string(entry.Key)]; !expected {
				return fmt.Errorf("state entry at index %d is an orphan or mismatched forward-lineage index", index)
			}
		case LineageByDownstreamPrefix[0]:
			if _, expected := expectedBackwardLineage[string(entry.Key)]; !expected {
				return fmt.Errorf("state entry at index %d is an orphan or mismatched backward-lineage index", index)
			}
		}
	}

	for _, record := range accumulatorRecords {
		if _, found := attestations[record.accumulator.AttestationId]; !found {
			return fmt.Errorf(
				"state entry at index %d lineage accumulator attestation %q is missing",
				record.index,
				record.accumulator.AttestationId,
			)
		}
	}

	expectedPendingReverse := make(map[string]struct{}, len(pendingClaimRecords))
	for _, record := range pendingClaimRecords {
		if _, found := attestations[record.attestationID]; !found {
			return fmt.Errorf("state entry at index %d pending claim attestation %q is missing", record.index, record.attestationID)
		}
		// Timed-out attestations historically leave their unresolved pending
		// indexes behind. They are inert because OnClaimResolved ignores a
		// non-AWAITING parent, but they are still valid exported state. Require
		// the two indexes to agree without retroactively imposing a status rule.
		reverseKey := AttestationPendingClaimsKey(record.attestationID, record.claimID)
		if err := requireGenesisIndex(entryByKey, reverseKey, "pending-claim reverse", record.index); err != nil {
			return err
		}
		expectedPendingReverse[string(reverseKey)] = struct{}{}
	}
	for index, entry := range gs.StateEntries {
		if entry.Key[0] != AttestationPendingClaimsPrefix[0] {
			continue
		}
		if _, expected := expectedPendingReverse[string(entry.Key)]; !expected {
			return fmt.Errorf("state entry at index %d is an orphan or mismatched pending-claim reverse index", index)
		}
	}

	expectedWitnessDeadlines := make(map[string]struct{}, len(witnessRewardRecords))
	for _, record := range witnessRewardRecords {
		reward := record.reward
		attestation, found := attestations[reward.AttestationID]
		if !found {
			return fmt.Errorf("state entry at index %d witness attestation %q is missing", record.index, reward.AttestationID)
		}
		if attestation.AdapterId != reward.AdapterID {
			return fmt.Errorf("state entry at index %d witness adapter does not match its attestation", record.index)
		}
		if attestation.Submitter != reward.Recipient {
			return fmt.Errorf("state entry at index %d witness recipient does not match its attestation", record.index)
		}
		if attestation.Status != AttestationStatus_ATTESTATION_STATUS_SETTLED {
			return fmt.Errorf("state entry at index %d witness reward belongs to a non-settled attestation", record.index)
		}
		if attestation.Link == nil ||
			len(attestation.Link.CitedFacts) != 0 ||
			len(attestation.Link.PendingClaims) != 0 {
			return fmt.Errorf("state entry at index %d witness reward belongs to a non-witness attestation", record.index)
		}
		if attestation.Link.AdapterId != attestation.AdapterId {
			return fmt.Errorf("state entry at index %d witness link adapter does not match its attestation", record.index)
		}
		if attestation.RewardUzrn != "0" {
			return fmt.Errorf("state entry at index %d pending witness attestation must record zero paid reward", record.index)
		}
		if _, found := adapters[reward.AdapterID]; !found {
			return fmt.Errorf("state entry at index %d witness adapter %q is missing", record.index, reward.AdapterID)
		}
		deadlineKey := WitnessDeadlineIndexKey(reward.Deadline, reward.AttestationID)
		if err := requireGenesisIndex(entryByKey, deadlineKey, "witness deadline", record.index); err != nil {
			return err
		}
		expectedWitnessDeadlines[string(deadlineKey)] = struct{}{}
	}
	for index, entry := range gs.StateEntries {
		if entry.Key[0] != WitnessDeadlineIndexPrefix[0] {
			continue
		}
		if _, expected := expectedWitnessDeadlines[string(entry.Key)]; !expected {
			return fmt.Errorf("state entry at index %d is an orphan or mismatched witness deadline index", index)
		}
	}

	maxSourceTier := make(map[string]int)
	for _, record := range attestationRecords {
		attestation := record.attestation
		if attestation.Link == nil || attestation.Link.Source == nil ||
			attestation.Link.Source.SourceId == "" {
			continue
		}
		tier := genesisSourceRefTier(attestation.Status)
		if tier == 0 {
			continue
		}
		key := string(SourceRefKey(attestation.Link.AdapterId, attestation.Link.Source.SourceId))
		if tier > maxSourceTier[key] {
			maxSourceTier[key] = tier
		}
	}
	for _, record := range sourceRefRecords {
		attestation, found := attestations[record.attestationID]
		if !found {
			return fmt.Errorf("state entry at index %d source-ref holder %q is missing", record.index, record.attestationID)
		}
		if attestation.Link == nil || attestation.Link.Source == nil ||
			attestation.Link.AdapterId != record.adapterID ||
			attestation.Link.Source.SourceId != record.sourceID {
			return fmt.Errorf("state entry at index %d source-ref key does not match its holder's declared source", record.index)
		}
		holderTier := genesisSourceRefTier(attestation.Status)
		key := string(SourceRefKey(record.adapterID, record.sourceID))
		if holderTier == 0 || holderTier != maxSourceTier[key] {
			return fmt.Errorf("state entry at index %d source-ref holder is releasable while a stronger claim exists", record.index)
		}
	}
	if dedupeArmed {
		for _, record := range attestationRecords {
			attestation := record.attestation
			if attestation.Link == nil || attestation.Link.Source == nil ||
				attestation.Link.Source.SourceId == "" ||
				genesisSourceRefTier(attestation.Status) == 0 {
				continue
			}
			sourceKey := SourceRefKey(attestation.Link.AdapterId, attestation.Link.Source.SourceId)
			if _, found := entryByKey[string(sourceKey)]; !found {
				return fmt.Errorf(
					"armed attestation %q has no source-ref holder",
					attestation.AttestationId,
				)
			}
		}
	}

	return nil
}

func requireGenesisIndex(
	entryByKey map[string]*GenesisStateEntry,
	key []byte,
	name string,
	sourceIndex int,
) error {
	entry, found := entryByKey[string(key)]
	if !found {
		return fmt.Errorf("state entry at index %d is missing its %s index %x", sourceIndex, name, key)
	}
	if !isGenesisIndexMarker(entry.Value) {
		return fmt.Errorf("state entry at index %d has a malformed %s index value", sourceIndex, name)
	}
	return nil
}

func isGenesisIndexMarker(value []byte) bool {
	return len(value) == 1 && value[0] == 0x01
}

func parseSourceRefGenesisKey(key []byte) (adapterID, sourceID string, ok bool) {
	if len(key) <= 5 || key[0] != SourceRefPrefix[0] {
		return "", "", false
	}
	adapterLen := int(binary.BigEndian.Uint32(key[1:5]))
	if adapterLen <= 0 || len(key) <= 5+adapterLen {
		return "", "", false
	}
	return string(key[5 : 5+adapterLen]), string(key[5+adapterLen:]), true
}

func genesisSourceRefTier(status AttestationStatus) int {
	switch status {
	case AttestationStatus_ATTESTATION_STATUS_SETTLED,
		AttestationStatus_ATTESTATION_STATUS_PARTIAL:
		return 2
	case AttestationStatus_ATTESTATION_STATUS_REJECTED,
		AttestationStatus_ATTESTATION_STATUS_SLASHED:
		return 0
	default:
		return 1
	}
}

func generatedAttestationSequence(attestationID string) (uint64, bool) {
	parts := strings.Split(attestationID, "-")
	if len(parts) != 3 || parts[0] != "att" {
		return 0, false
	}
	height, err := strconv.ParseUint(parts[1], 10, 64)
	if err != nil || strconv.FormatUint(height, 10) != parts[1] {
		return 0, false
	}
	sequence, err := strconv.ParseUint(parts[2], 10, 64)
	if err != nil || sequence == 0 || strconv.FormatUint(sequence, 10) != parts[2] {
		return 0, false
	}
	return sequence, true
}

func isNonNegativeDecimal(value string) bool {
	amount, ok := new(big.Int).SetString(value, 10)
	return ok && amount.Sign() >= 0
}

func isPositiveDecimal(value string) bool {
	amount, ok := new(big.Int).SetString(value, 10)
	return ok && amount.Sign() > 0
}
