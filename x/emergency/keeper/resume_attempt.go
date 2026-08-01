package keeper

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"sort"

	"github.com/zerone-chain/zerone/x/emergency/types"
)

const resumeAttemptEncodingVersion = byte(1)

// resumeAttempt is the O(1) anti-replay and retry cursor for one
// quarantine/proposer pair. A failed generation may advance only when the
// canonical recovery-manifest digest changes.
type resumeAttempt struct {
	Generation             uint64
	RecoveryManifestSHA256 string
	CeremonyID             string
}

func encodeResumeAttempt(attempt resumeAttempt) ([]byte, error) {
	if attempt.Generation == 0 {
		return nil, fmt.Errorf("resume attempt generation must be positive")
	}
	if !types.IsLowerSHA256(attempt.RecoveryManifestSHA256) {
		return nil, fmt.Errorf("resume attempt recovery manifest is not a canonical SHA-256")
	}
	if attempt.CeremonyID == "" {
		return nil, fmt.Errorf("resume attempt ceremony id cannot be empty")
	}
	encoded := make(
		[]byte,
		1+8+types.SHA256HexLength+len(attempt.CeremonyID),
	)
	encoded[0] = resumeAttemptEncodingVersion
	binary.BigEndian.PutUint64(encoded[1:9], attempt.Generation)
	copy(encoded[9:9+types.SHA256HexLength], attempt.RecoveryManifestSHA256)
	copy(encoded[9+types.SHA256HexLength:], attempt.CeremonyID)
	return encoded, nil
}

func decodeResumeAttempt(encoded []byte) (resumeAttempt, error) {
	const fixedLength = 1 + 8 + types.SHA256HexLength
	if len(encoded) <= fixedLength {
		return resumeAttempt{}, fmt.Errorf(
			"resume attempt encoding has length %d, want more than %d",
			len(encoded),
			fixedLength,
		)
	}
	if encoded[0] != resumeAttemptEncodingVersion {
		return resumeAttempt{}, fmt.Errorf(
			"unsupported resume attempt encoding version %d",
			encoded[0],
		)
	}
	attempt := resumeAttempt{
		Generation: binary.BigEndian.Uint64(encoded[1:9]),
		RecoveryManifestSHA256: string(
			encoded[9 : 9+types.SHA256HexLength],
		),
		CeremonyID: string(encoded[fixedLength:]),
	}
	if _, err := encodeResumeAttempt(attempt); err != nil {
		return resumeAttempt{}, err
	}
	return attempt, nil
}

func (k Keeper) getResumeAttempt(
	ctx context.Context,
	quarantineID string,
	proposer string,
) (resumeAttempt, bool, error) {
	store := k.storeService.OpenKVStore(ctx)
	encoded, err := store.Get(types.ResumeAttemptKey(quarantineID, proposer))
	if err != nil {
		return resumeAttempt{}, false, fmt.Errorf("read resume attempt: %w", err)
	}
	if encoded == nil {
		return resumeAttempt{}, false, nil
	}
	attempt, err := decodeResumeAttempt(encoded)
	if err != nil {
		return resumeAttempt{}, false, fmt.Errorf(
			"decode resume attempt for quarantine %q proposer %q: %w",
			quarantineID,
			proposer,
			err,
		)
	}
	return attempt, true, nil
}

func (k Keeper) setResumeAttempt(
	ctx context.Context,
	quarantineID string,
	proposer string,
	attempt resumeAttempt,
) error {
	encoded, err := encodeResumeAttempt(attempt)
	if err != nil {
		return err
	}
	store := k.storeService.OpenKVStore(ctx)
	if err := store.Set(
		types.ResumeAttemptKey(quarantineID, proposer),
		encoded,
	); err != nil {
		return fmt.Errorf("persist resume attempt: %w", err)
	}
	return nil
}

type resumeAttemptCandidate struct {
	key                    []byte
	startBlock             uint64
	ceremonyID             string
	recoveryManifestSHA256 string
}

// deriveResumeAttempts reconstructs the latest cursor for every
// quarantine/proposer pair from immutable ceremony history. Legacy malformed
// or pre-evidence records are inert and intentionally do not consume a secured
// recovery generation.
func deriveResumeAttempts(
	ceremonies []*types.EmergencyCeremony,
) (map[string]resumeAttempt, error) {
	candidates := make([]resumeAttemptCandidate, 0)
	for _, ceremony := range ceremonies {
		if ceremony == nil || ceremony.Type != string(types.CeremonyResume) {
			continue
		}
		var proposal types.EmergencyResumeProposal
		if err := unmarshalProposal(ceremony.ProposalData, &proposal); err != nil ||
			proposal.Id != ceremony.Id ||
			proposal.Proposer == "" ||
			proposal.HaltCeremonyId == "" ||
			!types.IsLowerSHA256(proposal.RecoveryManifestSha256) {
			continue
		}
		key := types.ResumeAttemptKey(
			proposal.HaltCeremonyId,
			proposal.Proposer,
		)
		candidates = append(candidates, resumeAttemptCandidate{
			key:                    key,
			startBlock:             ceremony.StartBlock,
			ceremonyID:             ceremony.Id,
			recoveryManifestSHA256: proposal.RecoveryManifestSha256,
		})
	}
	sort.Slice(candidates, func(i, j int) bool {
		if comparison := bytes.Compare(candidates[i].key, candidates[j].key); comparison != 0 {
			return comparison < 0
		}
		if candidates[i].startBlock != candidates[j].startBlock {
			return candidates[i].startBlock < candidates[j].startBlock
		}
		return candidates[i].ceremonyID < candidates[j].ceremonyID
	})

	derived := make(map[string]resumeAttempt)
	lastStartBlock := make(map[string]uint64)
	for _, candidate := range candidates {
		key := string(candidate.key)
		previous := derived[key]
		if previous.Generation != 0 &&
			lastStartBlock[key] == candidate.startBlock {
			return nil, fmt.Errorf(
				"resume attempts %q and %q share block %d for one quarantine/proposer pair",
				previous.CeremonyID,
				candidate.ceremonyID,
				candidate.startBlock,
			)
		}
		if previous.Generation != 0 &&
			previous.RecoveryManifestSHA256 ==
				candidate.recoveryManifestSHA256 {
			return nil, fmt.Errorf(
				"resume attempts %q and %q repeat unchanged recovery evidence",
				previous.CeremonyID,
				candidate.ceremonyID,
			)
		}
		if previous.Generation == ^uint64(0) {
			return nil, fmt.Errorf(
				"resume attempt generation overflows for key %x",
				candidate.key,
			)
		}
		derived[key] = resumeAttempt{
			Generation:             previous.Generation + 1,
			RecoveryManifestSHA256: candidate.recoveryManifestSHA256,
			CeremonyID:             candidate.ceremonyID,
		}
		lastStartBlock[key] = candidate.startBlock
	}
	return derived, nil
}

func validatePersistedResumeAttempts(
	snapshot OperationsSafetySnapshot,
	derived map[string]resumeAttempt,
) error {
	persistedCount := 0
	for _, record := range snapshot.Records {
		if !bytes.HasPrefix(record.Key, types.ResumeAttemptKeyPrefix) {
			continue
		}
		persistedCount++
		expected, found := derived[string(record.Key)]
		if !found {
			return fmt.Errorf(
				"resume attempt index key %x has no evidence-bound ceremony history",
				record.Key,
			)
		}
		actual, err := decodeResumeAttempt(record.Value)
		if err != nil {
			return fmt.Errorf(
				"decode resume attempt index key %x: %w",
				record.Key,
				err,
			)
		}
		if actual != expected {
			return fmt.Errorf(
				"resume attempt index key %x does not match ceremony history",
				record.Key,
			)
		}
	}
	if persistedCount != 0 && persistedCount != len(derived) {
		return fmt.Errorf(
			"resume attempt index is incomplete: persisted %d, derived %d",
			persistedCount,
			len(derived),
		)
	}
	return nil
}

func (k Keeper) replaceResumeAttemptIndex(
	ctx context.Context,
	derived map[string]resumeAttempt,
) error {
	store := k.storeService.OpenKVStore(ctx)
	iter, err := store.Iterator(
		types.ResumeAttemptKeyPrefix,
		prefixEndBytes(types.ResumeAttemptKeyPrefix),
	)
	if err != nil {
		return fmt.Errorf("open resume attempt index iterator: %w", err)
	}
	keys := make([][]byte, 0)
	for ; iter.Valid(); iter.Next() {
		keys = append(keys, bytes.Clone(iter.Key()))
	}
	iterationErr := terminalIteratorError(iter)
	closeErr := iter.Close()
	if iterationErr != nil {
		return fmt.Errorf("iterate resume attempt index: %w", iterationErr)
	}
	if closeErr != nil {
		return fmt.Errorf("close resume attempt index iterator: %w", closeErr)
	}
	for _, key := range keys {
		if err := store.Delete(key); err != nil {
			return fmt.Errorf("clear resume attempt index key %x: %w", key, err)
		}
	}

	sortedKeys := make([]string, 0, len(derived))
	for key := range derived {
		sortedKeys = append(sortedKeys, key)
	}
	sort.Strings(sortedKeys)
	for _, key := range sortedKeys {
		encoded, err := encodeResumeAttempt(derived[key])
		if err != nil {
			return err
		}
		if err := store.Set([]byte(key), encoded); err != nil {
			return fmt.Errorf("rebuild resume attempt index key %x: %w", key, err)
		}
	}
	return nil
}
