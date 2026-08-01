package keeper

import (
	"context"
	"encoding/binary"
	"fmt"
	"sort"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/protobuf/proto"

	"github.com/zerone-chain/zerone/x/emergency/types"
)

// MigrateOperationsSafetyV1 reconciles emergency state written by binaries
// that predate strict status and single-active-ceremony invariants. It reads
// the store without hardened getters so the named activation cannot panic
// before reaching its migration.
//
// Unknown status bytes and undecodable ceremony records are irreparable
// preconditions: the upgrade stops explicitly. Structurally inconsistent but
// decodable legacy ceremonies are deterministically terminalized. At most one
// fully snapshotted ceremony matching the persisted voting status survives.
func (k Keeper) MigrateOperationsSafetyV1(ctx context.Context) (uint64, error) {
	snapshot, err := k.ReadOperationsSafetySnapshot(ctx)
	if err != nil {
		return 0, err
	}
	return k.MigrateOperationsSafetyV1FromSnapshot(ctx, snapshot)
}

// MigrateOperationsSafetyV1FromSnapshot applies the one-time reconciliation
// from a complete, immutable H-1 record set. The coordinated upgrade handler
// authenticates that set by reconstructing its committed IAVL root and binding
// the subroot through root CommitInfo to the H-1 app hash before any mutation.
func (k Keeper) MigrateOperationsSafetyV1FromSnapshot(
	ctx context.Context,
	snapshot OperationsSafetySnapshot,
) (uint64, error) {
	records, err := operationsSafetySnapshotMap(snapshot)
	if err != nil {
		return 0, fmt.Errorf("validate operations-safety snapshot: %w", err)
	}
	store := k.storeService.OpenKVStore(ctx)

	var normalizedParams types.Params
	rawParams := records[string(types.ParamsKey)]
	if rawParams == nil {
		normalizedParams = types.DefaultParams()
	} else {
		var persistedParams types.Params
		if err := proto.Unmarshal(rawParams, &persistedParams); err != nil {
			return 0, fmt.Errorf(
				"irreparable emergency state: decode persisted params: %w",
				err,
			)
		}
		normalizedParams = *types.NormalizeLegacyParams(&persistedParams)
	}
	if err := normalizedParams.Validate(); err != nil {
		return 0, fmt.Errorf(
			"irreparable emergency state: validate normalized params: %w",
			err,
		)
	}

	rawStatus := records[string(types.HaltStatusKey)]
	status := types.StatusNormal
	statusWasNormalized := len(rawStatus) == 0
	if len(rawStatus) != 0 {
		status = types.EmergencyStatus(rawStatus)
		switch status {
		case types.StatusNormal, types.StatusHaltVoting, types.StatusHalted,
			types.StatusRevertVoting, types.StatusReverting, types.StatusResumeVoting:
		default:
			return 0, fmt.Errorf(
				"irreparable emergency state: unknown persisted status %q",
				string(rawStatus),
			)
		}
	}

	ceremonies := make([]*types.EmergencyCeremony, 0)
	for _, record := range snapshot.Records {
		if len(record.Key) == 0 || record.Key[0] != types.CeremonyKeyPrefix[0] {
			continue
		}
		var ceremony types.EmergencyCeremony
		if err := proto.Unmarshal(record.Value, &ceremony); err != nil {
			return 0, fmt.Errorf(
				"irreparable emergency state: decode ceremony at key %x: %w",
				record.Key,
				err,
			)
		}
		expectedKey := types.CeremonyKey(ceremony.Id)
		if ceremony.Id == "" || string(expectedKey) != string(record.Key) {
			return 0, fmt.Errorf(
				"irreparable emergency state: ceremony key %x does not match id %q",
				record.Key,
				ceremony.Id,
			)
		}
		if ceremony.Type == string(types.CeremonyRecoveryAuthorization) {
			return 0, fmt.Errorf(
				"irreparable emergency migration lineage: source v1 contains reserved recovery authorization ceremony %q",
				ceremony.Id,
			)
		}
		ceremonies = append(ceremonies, &ceremony)
	}
	sort.Slice(ceremonies, func(i, j int) bool {
		return ceremonies[i].Id < ceremonies[j].Id
	})
	for _, ceremony := range ceremonies {
		if len(ceremony.Electorate) > types.MaxEmergencyElectorateSize {
			return 0, fmt.Errorf(
				"irreparable emergency state: ceremony %q electorate has %d members, exceeds consensus maximum %d",
				ceremony.Id,
				len(ceremony.Electorate),
				types.MaxEmergencyElectorateSize,
			)
		}
	}
	derivedResumeAttempts, err := deriveResumeAttempts(ceremonies)
	if err != nil {
		return 0, fmt.Errorf(
			"irreparable emergency state: derive resume attempt index: %w",
			err,
		)
	}
	if err := validatePersistedResumeAttempts(
		snapshot,
		derivedResumeAttempts,
	); err != nil {
		return 0, fmt.Errorf(
			"irreparable emergency state: validate resume attempt index: %w",
			err,
		)
	}

	haltStartBz := records[string(types.HaltStartBlockKey)]
	if haltStartBz != nil && len(haltStartBz) != 8 {
		return 0, fmt.Errorf(
			"irreparable emergency state: quarantine start block has length %d",
			len(haltStartBz),
		)
	}
	persistedHaltStart := uint64(0)
	if haltStartBz != nil {
		persistedHaltStart = binary.BigEndian.Uint64(haltStartBz)
	}
	escalationBz := records[string(types.LastHaltEscalationBlockKey)]
	if escalationBz != nil && len(escalationBz) != 8 {
		return 0, fmt.Errorf(
			"irreparable emergency state: quarantine escalation block has length %d",
			len(escalationBz),
		)
	}
	persistedEscalation := uint64(0)
	if escalationBz != nil {
		persistedEscalation = binary.BigEndian.Uint64(escalationBz)
	}

	var expectedType types.CeremonyType
	targetStatus := status
	legacyRevertDetails := ""
	switch status {
	case types.StatusHaltVoting:
		expectedType = types.CeremonyHalt
	case types.StatusResumeVoting:
		expectedType = types.CeremonyResume
	case types.StatusRevertVoting, types.StatusReverting:
		// Height-only rollback authority is retired at this activation.
		targetStatus = types.StatusHalted
		heightBz := records[string(types.RevertTargetHeightKey)]
		if heightBz != nil && len(heightBz) != 8 {
			return 0, fmt.Errorf(
				"irreparable emergency state: legacy revert target height has length %d",
				len(heightBz),
			)
		}
		hashBz := records[string(types.RevertTargetHashKey)]
		ceremonyBz := records[string(types.RevertCeremonyIdKey)]
		height := uint64(0)
		if heightBz != nil {
			height = binary.BigEndian.Uint64(heightBz)
		}
		legacyRevertDetails = fmt.Sprintf(
			"retired legacy revert target cleared at activation: height=%d block_hash=%q ceremony_id=%q; no rollback executed",
			height,
			string(hashBz),
			string(ceremonyBz),
		)
	}

	canonicalHaltID := ""
	canonicalHaltStart := uint64(0)
	if targetStatus == types.StatusHalted ||
		targetStatus == types.StatusResumeVoting {
		persistedHaltID := string(
			records[string(types.ActiveHaltCeremonyIdKey)],
		)
		if start, valid := validatedQuarantineLink(
			persistedHaltID,
			ceremonies,
		); valid {
			canonicalHaltID = persistedHaltID
			canonicalHaltStart = start
		} else {
			canonicalHaltID, canonicalHaltStart =
				inferQuarantineLink(ceremonies)
		}
		if canonicalHaltID == "" {
			canonicalHaltID = legacyGenesisQuarantineID
		}
		if persistedHaltStart != 0 {
			canonicalHaltStart = persistedHaltStart
		}
		if canonicalHaltStart == 0 {
			canonicalHaltStart = uint64(1)
			if sdkCtx := sdk.UnwrapSDKContext(ctx); sdkCtx.BlockHeight() > 0 {
				canonicalHaltStart = uint64(sdkCtx.BlockHeight())
			}
		}
		if persistedEscalation != 0 {
			if normalizedParams.MaxHaltDurationBlocks >
				^uint64(0)-canonicalHaltStart {
				return 0, fmt.Errorf(
					"irreparable emergency state: quarantine deadline overflows uint64",
				)
			}
			firstDeadline :=
				canonicalHaltStart + normalizedParams.MaxHaltDurationBlocks
			if persistedEscalation < firstDeadline {
				return 0, fmt.Errorf(
					"irreparable emergency state: escalation block %d precedes first quarantine deadline %d",
					persistedEscalation,
					firstDeadline,
				)
			}
		}
	}

	var survivor *types.EmergencyCeremony
	if expectedType != "" {
		candidates := make([]*types.EmergencyCeremony, 0, 1)
		for _, ceremony := range ceremonies {
			if !isNonterminalEmergencyCeremony(ceremony) ||
				ceremony.Type != string(expectedType) ||
				!isUsableSnapshottedCeremony(ceremony) {
				continue
			}
			if expectedType == types.CeremonyResume {
				var proposal types.EmergencyResumeProposal
				if canonicalHaltID == "" ||
					unmarshalProposal(ceremony.ProposalData, &proposal) != nil ||
					proposal.HaltCeremonyId != canonicalHaltID {
					continue
				}
			}
			candidates = append(candidates, ceremony)
		}
		// Legacy v1 has no authenticated active-ceremony pointer. Selecting
		// one of multiple equally valid candidates by key order would invent
		// authority. Preserve a survivor only when the committed state proves
		// uniqueness; otherwise terminalize every candidate below.
		if len(candidates) == 1 {
			survivor = candidates[0]
		}
		if survivor == nil {
			if status == types.StatusHaltVoting {
				targetStatus = types.StatusNormal
			} else {
				targetStatus = types.StatusHalted
			}
		}
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	block := uint64(0)
	if sdkCtx.BlockHeight() > 0 {
		block = uint64(sdkCtx.BlockHeight())
	}
	k.SetParams(ctx, &normalizedParams)
	var terminalized uint64
	for _, ceremony := range ceremonies {
		if !isNonterminalEmergencyCeremony(ceremony) || ceremony == survivor {
			continue
		}
		ceremony.Phase = string(types.PhaseFailed)
		ceremony.FailureReason = fmt.Sprintf(
			"retired at %s activation: legacy ceremony is not the unique valid %s ceremony for persisted status %s",
			"upgrade-incident-operations-v1",
			expectedType,
			status,
		)
		if err := k.SetCeremony(ctx, ceremony); err != nil {
			return 0, err
		}
		terminalized++
	}
	if terminalized > 0 {
		k.AddAuditEntry(ctx, &types.EmergencyAuditEntry{
			Timestamp:   sdkCtx.BlockTime().Unix(),
			BlockNumber: block,
			Action:      string(types.AuditLegacyNormalized),
			Actor:       "system",
			Details: fmt.Sprintf(
				"operations-safety activation terminalized %d legacy ceremonies; each record retains its individual failure_reason",
				terminalized,
			),
		})
	}

	if statusWasNormalized || targetStatus != status {
		k.SetEmergencyStatus(ctx, targetStatus)
		k.AddAuditEntry(ctx, &types.EmergencyAuditEntry{
			Timestamp:   sdkCtx.BlockTime().Unix(),
			BlockNumber: block,
			Action:      string(types.AuditLegacyNormalized),
			Actor:       "system",
			Details: fmt.Sprintf(
				"operations-safety activation normalized emergency status from %q to %q",
				string(rawStatus),
				targetStatus,
			),
		})
	} else {
		// Persist even a known status so a proto-default/empty value cannot
		// remain absent after successful activation.
		k.SetEmergencyStatus(ctx, targetStatus)
	}

	if status == types.StatusRevertVoting || status == types.StatusReverting {
		for _, key := range [][]byte{
			types.RevertTargetHeightKey,
			types.RevertTargetHashKey,
			types.RevertCeremonyIdKey,
		} {
			if err := store.Delete(key); err != nil {
				return 0, fmt.Errorf("clear retired legacy revert target key %x: %w", key, err)
			}
		}
		k.AddAuditEntry(ctx, &types.EmergencyAuditEntry{
			Timestamp:   sdkCtx.BlockTime().Unix(),
			BlockNumber: block,
			Action:      string(types.AuditRevertFailed),
			Actor:       "system",
			Details:     legacyRevertDetails,
		})
	}

	if survivor == nil {
		k.setActiveCeremonyID(ctx, "")
	} else {
		k.setActiveCeremonyID(ctx, survivor.Id)
	}

	switch targetStatus {
	case types.StatusNormal, types.StatusHaltVoting:
		k.SetActiveHaltCeremonyId(ctx, "")
		k.ClearHaltStartBlock(ctx)
	case types.StatusHalted, types.StatusResumeVoting:
		k.SetActiveHaltCeremonyId(ctx, canonicalHaltID)
		k.SetHaltStartBlock(ctx, canonicalHaltStart)
		if persistedEscalation != 0 {
			k.SetLastHaltEscalationBlock(ctx, persistedEscalation)
		}
	}
	if err := k.replaceResumeAttemptIndex(
		ctx,
		derivedResumeAttempts,
	); err != nil {
		return 0, fmt.Errorf("rebuild resume attempt index: %w", err)
	}

	return terminalized, nil
}

func validatedQuarantineLink(
	id string,
	ceremonies []*types.EmergencyCeremony,
) (uint64, bool) {
	if id == legacyGenesisQuarantineID {
		return 0, true
	}
	if id == "" {
		return 0, false
	}
	for _, ceremony := range ceremonies {
		if ceremony != nil &&
			ceremony.Id == id &&
			ceremony.Type == string(types.CeremonyHalt) &&
			ceremony.Phase == string(types.PhaseFinalized) {
			return ceremony.StartBlock, true
		}
	}
	return 0, false
}

func isNonterminalEmergencyCeremony(ceremony *types.EmergencyCeremony) bool {
	return ceremony.Phase != string(types.PhaseFinalized) &&
		ceremony.Phase != string(types.PhaseFailed)
}

func isUsableSnapshottedCeremony(ceremony *types.EmergencyCeremony) bool {
	if ceremony.Phase != string(types.PhasePrevote) &&
		ceremony.Phase != string(types.PhasePrecommit) {
		return false
	}
	electorate, _, err := validateCeremonySnapshot(ceremony)
	if err != nil {
		return false
	}
	if err := validateCeremonyTallies(ceremony, electorate); err != nil {
		return false
	}
	if ceremony.PrevoteDeadline <= ceremony.StartBlock ||
		ceremony.PrecommitDeadline <= ceremony.PrevoteDeadline ||
		ceremony.TimeoutDeadline < ceremony.PrecommitDeadline {
		return false
	}
	switch types.CeremonyType(ceremony.Type) {
	case types.CeremonyHalt:
		var proposal types.EmergencyHaltProposal
		return unmarshalProposal(ceremony.ProposalData, &proposal) == nil &&
			proposal.Id == ceremony.Id &&
			proposal.Proposer != "" &&
			proposal.Reason != ""
	case types.CeremonyResume:
		var proposal types.EmergencyResumeProposal
		return unmarshalProposal(ceremony.ProposalData, &proposal) == nil &&
			proposal.Id == ceremony.Id &&
			proposal.Proposer != "" &&
			proposal.HaltCeremonyId != "" &&
			proposal.Justification != "" &&
			types.IsLowerSHA256(proposal.RecoveryManifestSha256)
	default:
		return false
	}
}
