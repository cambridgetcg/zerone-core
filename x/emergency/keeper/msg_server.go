package keeper

import (
	"context"
	"crypto/sha256"
	"fmt"
	"math/big"

	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/x/emergency/types"
)

type msgServer struct {
	types.UnimplementedMsgServer
	Keeper
}

// NewMsgServerImpl returns an implementation of the MsgServer interface.
func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return &msgServer{Keeper: keeper}
}

var _ types.MsgServer = msgServer{}

// checkCeremonyProposalRateLimit applies one shared proposal budget to every
// live emergency ceremony lane. Resume and recovery-authorization proposals
// must not provide an attacker with fresh, unbounded retries merely because the
// chain is already quarantined.
func (k msgServer) checkCeremonyProposalRateLimit(
	goCtx context.Context,
	proposer string,
	currentBlock uint64,
) error {
	params := k.GetParams(goCtx)
	if count := k.GetGuardianProposalCount(goCtx, proposer); count >=
		params.MaxProposalsPerGuardianPerEpoch {
		return types.ErrProposalLimitExceeded.Wrapf(
			"guardian %s reached the per-epoch proposal limit %d",
			proposer,
			params.MaxProposalsPerGuardianPerEpoch,
		)
	}
	if count := k.GetEpochProposalCount(goCtx); count >=
		params.MaxProposalsPerEpoch {
		return types.ErrProposalLimitExceeded.Wrapf(
			"epoch proposal limit %d reached",
			params.MaxProposalsPerEpoch,
		)
	}
	lastBlock := k.GetLastProposalBlock(goCtx)
	if lastBlock > currentBlock {
		return types.ErrCooldownActive.Wrapf(
			"persisted last proposal block %d is ahead of current block %d",
			lastBlock,
			currentBlock,
		)
	}
	if lastBlock > 0 &&
		(currentBlock-lastBlock) < params.CooldownBlocks {
		return types.ErrCooldownActive.Wrapf(
			"%d blocks remaining",
			params.CooldownBlocks-(currentBlock-lastBlock),
		)
	}
	return nil
}

func (k msgServer) recordCeremonyProposal(
	goCtx context.Context,
	proposer string,
	currentBlock uint64,
) {
	k.IncrementGuardianProposalCount(goCtx, proposer)
	k.IncrementEpochProposalCount(goCtx)
	k.SetLastProposalBlock(goCtx, currentBlock)
	ctx := sdk.UnwrapSDKContext(goCtx)
	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"zerone.emergency.ceremony_proposal_budget_consumed",
		sdk.NewAttribute("proposer", proposer),
		sdk.NewAttribute(
			"block_height",
			fmt.Sprintf("%d", currentBlock),
		),
		sdk.NewAttribute(
			"guardian_epoch_count",
			fmt.Sprintf(
				"%d",
				k.GetGuardianProposalCount(goCtx, proposer),
			),
		),
		sdk.NewAttribute(
			"global_epoch_count",
			fmt.Sprintf("%d", k.GetEpochProposalCount(goCtx)),
		),
	))
}

// ProposeHalt handles a Guardian's proposal to quarantine ordinary
// application transactions. CometBFT consensus continues.
func (k msgServer) ProposeHalt(goCtx context.Context, msg *types.MsgProposeHalt) (*types.MsgProposeHaltResponse, error) {
	if !types.HasAuthenticatedEmergencyTx(goCtx) {
		return nil, types.ErrUnauthenticatedEmergencyExecution
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	params := k.GetParams(goCtx)

	if !k.IsGuardian(goCtx, msg.Proposer) {
		return nil, fmt.Errorf("%w: %s", types.ErrNotGuardian, msg.Proposer)
	}

	status := k.GetEmergencyStatus(goCtx)
	if status != types.StatusNormal {
		return nil, fmt.Errorf("%w: current status is %s, must be normal", types.ErrStatusConflict, status)
	}
	if k.IsHalted(goCtx) {
		return nil, fmt.Errorf(
			"%w: transaction quarantine release latch remains active through block %d",
			types.ErrStatusConflict,
			k.GetQuarantineReleaseBlock(goCtx),
		)
	}

	if _, found := k.GetActiveCeremony(goCtx); found {
		return nil, fmt.Errorf("%w", types.ErrCeremonyActive)
	}

	// Guardian stake floor (skip if council active — H-5).
	if !k.isCouncilActive(goCtx, params) {
		guardianStake := k.GetGuardianStake(goCtx)
		minStake, ok := new(big.Int).SetString(params.MinGuardianStake, 10)
		if ok && guardianStake.Cmp(minStake) < 0 {
			return nil, fmt.Errorf("%w: total guardian stake %s < minimum %s (commitment 10: emergency transaction quarantine requires plural backing; the audit trail demands witnesses)", types.ErrInsufficientGuardians, guardianStake.String(), params.MinGuardianStake)
		}
	}

	if ctx.BlockHeight() < 0 {
		return nil, fmt.Errorf(
			"cannot propose emergency halt at negative block height %d",
			ctx.BlockHeight(),
		)
	}
	currentBlock := uint64(ctx.BlockHeight())
	if err := k.checkCeremonyProposalRateLimit(
		goCtx,
		msg.Proposer,
		currentBlock,
	); err != nil {
		return nil, err
	}

	proposalId := fmt.Sprintf("halt-%d-%s", ctx.BlockHeight(), msg.Proposer)
	proposal := types.EmergencyHaltProposal{
		Id:              proposalId,
		Proposer:        msg.Proposer,
		Reason:          msg.Reason,
		ProposedAtBlock: currentBlock,
	}

	ceremony, err := k.CreateHaltCeremony(goCtx, &proposal)
	if err != nil {
		return nil, err
	}

	k.SetEmergencyStatus(goCtx, types.StatusHaltVoting)
	k.recordCeremonyProposal(goCtx, msg.Proposer, currentBlock)

	k.AddAuditEntry(goCtx, &types.EmergencyAuditEntry{
		Timestamp:   ctx.BlockTime().Unix(),
		BlockNumber: currentBlock,
		Action:      string(types.AuditHaltProposed),
		Actor:       msg.Proposer,
		CeremonyId:  ceremony.Id,
		Details:     msg.Reason,
	})

	ctx.EventManager().EmitEvent(
		sdk.NewEvent("zerone.emergency.halt_proposed",
			sdk.NewAttribute("ceremony_id", ceremony.Id),
			sdk.NewAttribute("proposer", msg.Proposer),
			sdk.NewAttribute("reason", msg.Reason),
		),
	)

	return &types.MsgProposeHaltResponse{ProposalId: ceremony.Id}, nil
}

// VoteHalt handles a Guardian's vote on a halt ceremony.
func (k msgServer) VoteHalt(goCtx context.Context, msg *types.MsgVoteHalt) (*types.MsgVoteHaltResponse, error) {
	if !types.HasAuthenticatedEmergencyTx(goCtx) {
		return nil, types.ErrUnauthenticatedEmergencyExecution
	}
	ceremony, found := k.GetCeremony(goCtx, msg.ProposalId)
	if !found {
		return nil, fmt.Errorf("%w: %s", types.ErrNoCeremony, msg.ProposalId)
	}

	if ceremony.Type != string(types.CeremonyHalt) {
		return nil, fmt.Errorf("%w: ceremony %s is not a halt ceremony", types.ErrInvalidPhase, msg.ProposalId)
	}

	if ceremony.Phase == string(types.PhasePrecommit) {
		if !msg.Approve {
			return nil, fmt.Errorf(
				"%w: negative precommit is not an affirmative commitment; reject during prevote or abstain",
				types.ErrInvalidPhase,
			)
		}
		err := k.AddPrecommit(goCtx, msg.ProposalId, &types.EmergencyPrecommit{Voter: msg.Voter})
		if err != nil {
			return nil, err
		}
		k.addVoteAudit(goCtx, types.AuditHaltPrecommit, msg.Voter, msg.ProposalId)
	} else {
		err := k.AddPrevote(goCtx, msg.ProposalId, &types.EmergencyVote{Voter: msg.Voter, Approve: msg.Approve})
		if err != nil {
			return nil, err
		}
		k.addVoteAudit(goCtx, types.AuditHaltPrevote, msg.Voter, msg.ProposalId)
	}

	finalized, err := k.CheckCeremonyProgress(goCtx, msg.ProposalId)
	if err != nil {
		return nil, err
	}
	ceremony, found = k.GetCeremony(goCtx, msg.ProposalId)
	if !found {
		return nil, fmt.Errorf("%w: %s", types.ErrNoCeremony, msg.ProposalId)
	}

	if finalized {
		k.HandleCeremonyFinalization(goCtx, msg.ProposalId)
	} else if ceremony.Phase == string(types.PhaseFailed) {
		k.HandleCeremonyFailure(goCtx, msg.ProposalId)
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	ctx.EventManager().EmitEvent(
		sdk.NewEvent("zerone.emergency.vote_halt",
			sdk.NewAttribute("ceremony_id", msg.ProposalId),
			sdk.NewAttribute("voter", msg.Voter),
			sdk.NewAttribute("approve", fmt.Sprintf("%v", msg.Approve)),
		),
	)

	return &types.MsgVoteHaltResponse{
		QuorumReached: ceremony.Phase == string(types.PhasePrecommit) || ceremony.Phase == string(types.PhaseFinalized),
		ChainHalted:   finalized,
	}, nil
}

// ProposeRevert is retained for wire compatibility but fails closed. A height
// alone cannot authenticate a canonical block, AppHash, snapshot, IBC packet
// set, or fork choice. Recovery must be forward-only or use an explicit,
// hash-bound social-fork ceremony outside this legacy message.
func (k msgServer) ProposeRevert(_ context.Context, _ *types.MsgProposeRevert) (*types.MsgProposeRevertResponse, error) {
	// event-audit: fail-closed; failed transaction events are not durable.
	return nil, types.ErrUnsafeRevertDisabled
}

// VoteRevert is disabled with ProposeRevert. Legacy ceremony records remain
// queryable and exportable, but no new height-only rollback can finalize.
func (k msgServer) VoteRevert(_ context.Context, _ *types.MsgVoteRevert) (*types.MsgVoteRevertResponse, error) {
	// event-audit: fail-closed; failed transaction events are not durable.
	return nil, types.ErrUnsafeRevertDisabled
}

// ProposeResume handles a Guardian's proposal to resume chain operations.
func (k msgServer) ProposeResume(goCtx context.Context, msg *types.MsgProposeResume) (*types.MsgProposeResumeResponse, error) {
	if !types.HasAuthenticatedEmergencyTx(goCtx) {
		return nil, types.ErrUnauthenticatedEmergencyExecution
	}
	ctx := sdk.UnwrapSDKContext(goCtx)

	if !k.IsGuardian(goCtx, msg.Proposer) {
		return nil, fmt.Errorf("%w: %s", types.ErrNotGuardian, msg.Proposer)
	}

	status := k.GetEmergencyStatus(goCtx)
	if status != types.StatusHalted {
		return nil, fmt.Errorf("%w: chain must be halted to propose resume", types.ErrHaltRequired)
	}

	if _, found := k.GetActiveCeremony(goCtx); found {
		return nil, fmt.Errorf("%w", types.ErrCeremonyActive)
	}
	activeHaltID := k.GetActiveHaltCeremonyId(goCtx)
	if activeHaltID == "" {
		return nil, errorsmod.Wrap(types.ErrHaltRequired, "active quarantine has no halt ceremony id")
	}
	if msg.Justification == "" || len(msg.Justification) > types.MaxRecoveryJustificationBytes {
		return nil, fmt.Errorf("resume requires a non-empty justification of at most %d bytes", types.MaxRecoveryJustificationBytes)
	}
	if !types.IsLowerSHA256(msg.RecoveryManifestSha256) {
		return nil, fmt.Errorf("resume requires recovery_manifest_sha256 as exactly %d lowercase hexadecimal characters", types.SHA256HexLength)
	}

	if ctx.BlockHeight() < 0 {
		return nil, fmt.Errorf(
			"cannot propose emergency resume at negative block height %d",
			ctx.BlockHeight(),
		)
	}
	currentBlock := uint64(ctx.BlockHeight())
	haltStart := k.GetHaltStartBlock(goCtx)
	if haltStart == 0 {
		return nil, errorsmod.Wrap(
			types.ErrHaltRequired,
			"active quarantine has no persisted finalization height",
		)
	}
	if currentBlock <= haltStart {
		return nil, errorsmod.Wrapf(
			types.ErrCooldownActive,
			"resume requires a committed block after halt finalization at height %d",
			haltStart,
		)
	}
	if err := k.checkCeremonyProposalRateLimit(
		goCtx,
		msg.Proposer,
		currentBlock,
	); err != nil {
		return nil, err
	}

	generation := uint64(1)
	previous, found, err := k.getResumeAttempt(
		goCtx,
		activeHaltID,
		msg.Proposer,
	)
	if err != nil {
		return nil, err
	}
	if found {
		previousCeremony, ceremonyFound := k.GetCeremony(
			goCtx,
			previous.CeremonyID,
		)
		if !ceremonyFound {
			return nil, fmt.Errorf(
				"resume attempt index references missing ceremony %q",
				previous.CeremonyID,
			)
		}
		var previousProposal types.EmergencyResumeProposal
		if previousCeremony.Type != string(types.CeremonyResume) ||
			unmarshalProposal(
				previousCeremony.ProposalData,
				&previousProposal,
			) != nil ||
			previousProposal.Id != previous.CeremonyID ||
			previousProposal.Proposer != msg.Proposer ||
			previousProposal.HaltCeremonyId != activeHaltID ||
			previousProposal.RecoveryManifestSha256 !=
				previous.RecoveryManifestSHA256 {
			return nil, fmt.Errorf(
				"resume attempt index does not match ceremony %q",
				previous.CeremonyID,
			)
		}
		if previousCeremony.Phase != string(types.PhaseFailed) {
			if isNonterminalEmergencyCeremony(previousCeremony) {
				return nil, types.ErrCeremonyActive
			}
			return nil, errorsmod.Wrapf(
				types.ErrStatusConflict,
				"resume attempt index references non-failed ceremony %q in phase %s",
				previous.CeremonyID,
				previousCeremony.Phase,
			)
		}
		if previous.RecoveryManifestSHA256 == msg.RecoveryManifestSha256 {
			return nil, errorsmod.Wrapf(
				types.ErrProposalLimitExceeded,
				"guardian %s already opened recovery generation %d with manifest %s for quarantine %s; retry requires changed evidence",
				msg.Proposer,
				previous.Generation,
				msg.RecoveryManifestSha256,
				activeHaltID,
			)
		}
		if currentBlock <= previousCeremony.StartBlock {
			return nil, errorsmod.Wrapf(
				types.ErrCooldownActive,
				"recovery generation %d at height %d must commit before retry",
				previous.Generation,
				previousCeremony.StartBlock,
			)
		}
		if previous.Generation == ^uint64(0) {
			return nil, errorsmod.Wrapf(
				types.ErrProposalLimitExceeded,
				"recovery generation exhausted for guardian %s and quarantine %s",
				msg.Proposer,
				activeHaltID,
			)
		}
		generation = previous.Generation + 1
	}

	haltIDHash := sha256.Sum256([]byte(activeHaltID))
	proposalId := fmt.Sprintf(
		"resume-%d-%x-g%d-%s",
		ctx.BlockHeight(),
		haltIDHash[:8],
		generation,
		msg.Proposer,
	)
	proposal := types.EmergencyResumeProposal{
		Id:                     proposalId,
		Proposer:               msg.Proposer,
		HaltCeremonyId:         activeHaltID,
		Justification:          msg.Justification,
		RecoveryManifestSha256: msg.RecoveryManifestSha256,
	}

	ceremony, err := k.CreateResumeCeremony(goCtx, &proposal)
	if err != nil {
		return nil, err
	}
	if err := k.setResumeAttempt(goCtx, activeHaltID, msg.Proposer, resumeAttempt{
		Generation:             generation,
		RecoveryManifestSHA256: msg.RecoveryManifestSha256,
		CeremonyID:             ceremony.Id,
	}); err != nil {
		return nil, err
	}
	k.recordCeremonyProposal(goCtx, msg.Proposer, currentBlock)

	k.SetEmergencyStatus(goCtx, types.StatusResumeVoting)

	k.AddAuditEntry(goCtx, &types.EmergencyAuditEntry{
		Timestamp:   ctx.BlockTime().Unix(),
		BlockNumber: uint64(ctx.BlockHeight()),
		Action:      string(types.AuditResumeProposed),
		Actor:       msg.Proposer,
		CeremonyId:  ceremony.Id,
		Details: fmt.Sprintf(
			"resume generation %d proposed; recovery_manifest_sha256=%s; justification=%s",
			generation,
			msg.RecoveryManifestSha256,
			msg.Justification,
		),
	})

	ctx.EventManager().EmitEvent(
		sdk.NewEvent("zerone.emergency.resume_proposed",
			sdk.NewAttribute("ceremony_id", ceremony.Id),
			sdk.NewAttribute("proposer", msg.Proposer),
			sdk.NewAttribute("halt_ceremony_id", proposal.HaltCeremonyId),
			sdk.NewAttribute("recovery_generation", fmt.Sprintf("%d", generation)),
			sdk.NewAttribute("recovery_manifest_sha256", msg.RecoveryManifestSha256),
		),
	)

	return &types.MsgProposeResumeResponse{ProposalId: ceremony.Id}, nil
}

// VoteResume handles a Guardian's vote on a resume ceremony.
func (k msgServer) VoteResume(goCtx context.Context, msg *types.MsgVoteResume) (*types.MsgVoteResumeResponse, error) {
	if !types.HasAuthenticatedEmergencyTx(goCtx) {
		return nil, types.ErrUnauthenticatedEmergencyExecution
	}
	ceremony, found := k.GetCeremony(goCtx, msg.ProposalId)
	if !found {
		return nil, fmt.Errorf("%w: %s", types.ErrNoCeremony, msg.ProposalId)
	}

	if ceremony.Type != string(types.CeremonyResume) {
		return nil, fmt.Errorf("%w: ceremony %s is not a resume ceremony", types.ErrInvalidPhase, msg.ProposalId)
	}

	if ceremony.Phase == string(types.PhasePrecommit) {
		if !msg.Approve {
			return nil, fmt.Errorf(
				"%w: negative precommit is not an affirmative commitment; reject during prevote or abstain",
				types.ErrInvalidPhase,
			)
		}
		err := k.AddPrecommit(goCtx, msg.ProposalId, &types.EmergencyPrecommit{Voter: msg.Voter})
		if err != nil {
			return nil, err
		}
		k.addVoteAudit(goCtx, types.AuditResumePrecommit, msg.Voter, msg.ProposalId)
	} else {
		err := k.AddPrevote(goCtx, msg.ProposalId, &types.EmergencyVote{Voter: msg.Voter, Approve: msg.Approve})
		if err != nil {
			return nil, err
		}
		k.addVoteAudit(goCtx, types.AuditResumePrevote, msg.Voter, msg.ProposalId)
	}

	finalized, err := k.CheckCeremonyProgress(goCtx, msg.ProposalId)
	if err != nil {
		return nil, err
	}
	ceremony, found = k.GetCeremony(goCtx, msg.ProposalId)
	if !found {
		return nil, fmt.Errorf("%w: %s", types.ErrNoCeremony, msg.ProposalId)
	}

	if finalized {
		k.HandleCeremonyFinalization(goCtx, msg.ProposalId)
	} else if ceremony.Phase == string(types.PhaseFailed) {
		k.HandleCeremonyFailure(goCtx, msg.ProposalId)
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	ctx.EventManager().EmitEvent(
		sdk.NewEvent("zerone.emergency.vote_resume",
			sdk.NewAttribute("ceremony_id", msg.ProposalId),
			sdk.NewAttribute("voter", msg.Voter),
			sdk.NewAttribute("approve", fmt.Sprintf("%v", msg.Approve)),
		),
	)

	return &types.MsgVoteResumeResponse{
		QuorumReached: ceremony.Phase == string(types.PhasePrecommit) || ceremony.Phase == string(types.PhaseFinalized),
		ChainResumed:  finalized,
	}, nil
}

// ProposeRecoveryAuthorization opens a Guardian ceremony for the exact next
// SDK-governance proposal ID. Ordinary SDK proposal submission remains closed
// until this ceremony finalizes, so the pre-authorized sequence cannot be
// front-run by another proposal.
func (k msgServer) ProposeRecoveryAuthorization(
	goCtx context.Context,
	msg *types.MsgProposeRecoveryAuthorization,
) (*types.MsgProposeRecoveryAuthorizationResponse, error) {
	if !types.HasAuthenticatedEmergencyTx(goCtx) {
		return nil, types.ErrUnauthenticatedEmergencyExecution
	}
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}
	if !k.IsGuardian(goCtx, msg.Proposer) {
		return nil, fmt.Errorf("%w: %s", types.ErrNotGuardian, msg.Proposer)
	}
	if k.GetEmergencyStatus(goCtx) != types.StatusHalted {
		return nil, fmt.Errorf(
			"%w: recovery authorization requires stable halted status",
			types.ErrHaltRequired,
		)
	}
	activeHaltID := k.GetActiveHaltCeremonyId(goCtx)
	if activeHaltID == "" {
		return nil, errorsmod.Wrap(
			types.ErrHaltRequired,
			"active quarantine has no halt ceremony id",
		)
	}
	if _, found := k.GetActiveCeremony(goCtx); found {
		return nil, types.ErrCeremonyActive
	}
	if _, _, activated, err := k.GetOperationsSafetyV2Activation(
		goCtx,
	); err != nil {
		return nil, err
	} else if !activated {
		return nil, fmt.Errorf(
			"recovery authorization requires operations-safety v2 activation",
		)
	}
	if k.recoveryTargetVerifier == nil {
		return nil, fmt.Errorf(
			"recovery authorization target verifier is unavailable",
		)
	}
	currentAuthorization, currentFound, err :=
		k.GetRecoveryAuthorization(goCtx)
	if err != nil {
		return nil, err
	}
	if msg.ActionType == "revoke" {
		if !currentFound ||
			currentAuthorization.Outcome != "" ||
			currentAuthorization.HaltCeremonyId != activeHaltID ||
			currentAuthorization.SdkGovProposalId !=
				msg.SdkGovProposalId ||
			currentAuthorization.ActionSha256 != msg.ActionSha256 ||
			currentAuthorization.RecoveryManifestSha256 !=
				msg.RecoveryManifestSha256 ||
			currentAuthorization.UpgradePlanSha256 !=
				msg.UpgradePlanSha256 ||
			currentAuthorization.AuthorizedSubmitter !=
				msg.AuthorizedSubmitter {
			return nil, fmt.Errorf(
				"recovery revocation must exactly identify the live incident authorization",
			)
		}
	} else if currentFound &&
		currentAuthorization.HaltCeremonyId == activeHaltID &&
		currentAuthorization.Outcome == "" {
		return nil, fmt.Errorf(
			"live recovery authorization %q must reach a terminal outcome or be revoked by Guardian quorum before a replacement can open",
			currentAuthorization.AuthorizationCeremonyId,
		)
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	if ctx.BlockHeight() <= 0 {
		return nil, fmt.Errorf(
			"recovery authorization requires a positive block height",
		)
	}
	currentBlock := uint64(ctx.BlockHeight())
	if err := k.checkCeremonyProposalRateLimit(
		goCtx,
		msg.Proposer,
		currentBlock,
	); err != nil {
		return nil, err
	}
	if err := k.recoveryTargetVerifier.VerifyRecoveryAuthorizationTarget(
		goCtx,
		msg.SdkGovProposalId,
		msg.ActionSha256,
		msg.UpgradePlanSha256,
		msg.ActionType,
	); err != nil {
		return nil, fmt.Errorf(
			"recovery authorization target preflight failed: %w",
			err,
		)
	}

	generation := uint64(1)
	if currentFound &&
		currentAuthorization.HaltCeremonyId == activeHaltID {
		if msg.ActionType == "revoke" {
			generation = currentAuthorization.Generation
		} else {
			if currentAuthorization.Generation == ^uint64(0) {
				return nil, fmt.Errorf(
					"recovery authorization generation overflow",
				)
			}
			generation = currentAuthorization.Generation + 1
		}
	}

	haltIDHash := sha256.Sum256([]byte(activeHaltID))
	proposalID := fmt.Sprintf(
		"recovery-auth-%d-%x-g%d-%d-%s-%s",
		ctx.BlockHeight(),
		haltIDHash[:8],
		generation,
		msg.SdkGovProposalId,
		msg.ActionType,
		msg.Proposer,
	)
	proposal := &types.EmergencyRecoveryAuthorizationProposal{
		Id:                     proposalID,
		Proposer:               msg.Proposer,
		HaltCeremonyId:         activeHaltID,
		SdkGovProposalId:       msg.SdkGovProposalId,
		ActionSha256:           msg.ActionSha256,
		RecoveryManifestSha256: msg.RecoveryManifestSha256,
		Justification:          msg.Justification,
		UpgradePlanSha256:      msg.UpgradePlanSha256,
		AuthorizedSubmitter:    msg.AuthorizedSubmitter,
		ActionType:             msg.ActionType,
		Generation:             generation,
	}
	ceremony, err := k.CreateRecoveryAuthorizationCeremony(goCtx, proposal)
	if err != nil {
		return nil, err
	}
	k.recordCeremonyProposal(goCtx, msg.Proposer, currentBlock)
	k.AddAuditEntry(goCtx, &types.EmergencyAuditEntry{
		Timestamp:   ctx.BlockTime().Unix(),
		BlockNumber: currentBlock,
		Action:      string(types.AuditRecoveryAuthorizationProposed),
		Actor:       msg.Proposer,
		CeremonyId:  ceremony.Id,
		Details: fmt.Sprintf(
			"proposed recovery authorization generation %d for next SDK governance proposal %d; action_type=%s; action_sha256=%s; upgrade_plan_sha256=%s; recovery_manifest_sha256=%s; authorized_submitter=%s; justification=%s",
			generation,
			msg.SdkGovProposalId,
			msg.ActionType,
			msg.ActionSha256,
			msg.UpgradePlanSha256,
			msg.RecoveryManifestSha256,
			msg.AuthorizedSubmitter,
			msg.Justification,
		),
	})
	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"zerone.emergency.recovery_authorization_proposed",
		sdk.NewAttribute("ceremony_id", ceremony.Id),
		sdk.NewAttribute("halt_ceremony_id", activeHaltID),
		sdk.NewAttribute(
			"sdk_gov_proposal_id",
			fmt.Sprintf("%d", msg.SdkGovProposalId),
		),
		sdk.NewAttribute(
			"generation",
			fmt.Sprintf("%d", generation),
		),
		sdk.NewAttribute("action_type", msg.ActionType),
		sdk.NewAttribute("action_sha256", msg.ActionSha256),
		sdk.NewAttribute(
			"upgrade_plan_sha256",
			msg.UpgradePlanSha256,
		),
		sdk.NewAttribute(
			"recovery_manifest_sha256",
			msg.RecoveryManifestSha256,
		),
		sdk.NewAttribute(
			"authorized_submitter",
			msg.AuthorizedSubmitter,
		),
	))
	return &types.MsgProposeRecoveryAuthorizationResponse{
		ProposalId: ceremony.Id,
	}, nil
}

// VoteRecoveryAuthorization records a prevote or affirmative precommit for an
// exact recovery capability. Finalization leaves admission quarantined.
func (k msgServer) VoteRecoveryAuthorization(
	goCtx context.Context,
	msg *types.MsgVoteRecoveryAuthorization,
) (*types.MsgVoteRecoveryAuthorizationResponse, error) {
	if !types.HasAuthenticatedEmergencyTx(goCtx) {
		return nil, types.ErrUnauthenticatedEmergencyExecution
	}
	ceremony, found := k.GetCeremony(goCtx, msg.ProposalId)
	if !found {
		return nil, fmt.Errorf("%w: %s", types.ErrNoCeremony, msg.ProposalId)
	}
	if ceremony.Type != string(types.CeremonyRecoveryAuthorization) {
		return nil, fmt.Errorf(
			"%w: ceremony %s is not a recovery authorization",
			types.ErrInvalidPhase,
			msg.ProposalId,
		)
	}
	if k.GetEmergencyStatus(goCtx) != types.StatusHalted {
		return nil, fmt.Errorf(
			"%w: recovery authorization voting requires halted status",
			types.ErrHaltRequired,
		)
	}
	if ceremony.Phase == string(types.PhasePrecommit) {
		if !msg.Approve {
			return nil, fmt.Errorf(
				"%w: negative precommit is not an affirmative commitment",
				types.ErrInvalidPhase,
			)
		}
		if err := k.AddPrecommit(
			goCtx,
			msg.ProposalId,
			&types.EmergencyPrecommit{Voter: msg.Voter},
		); err != nil {
			return nil, err
		}
		k.addVoteAudit(
			goCtx,
			types.AuditRecoveryAuthorizationPrecommit,
			msg.Voter,
			msg.ProposalId,
		)
	} else {
		if err := k.AddPrevote(
			goCtx,
			msg.ProposalId,
			&types.EmergencyVote{
				Voter:   msg.Voter,
				Approve: msg.Approve,
			},
		); err != nil {
			return nil, err
		}
		k.addVoteAudit(
			goCtx,
			types.AuditRecoveryAuthorizationPrevote,
			msg.Voter,
			msg.ProposalId,
		)
	}
	finalized, err := k.CheckCeremonyProgress(goCtx, msg.ProposalId)
	if err != nil {
		return nil, err
	}
	if finalized {
		k.HandleCeremonyFinalization(goCtx, msg.ProposalId)
	}
	updated, found := k.GetCeremony(goCtx, msg.ProposalId)
	if !found {
		return nil, fmt.Errorf("%w: %s", types.ErrNoCeremony, msg.ProposalId)
	}
	if updated.Phase == string(types.PhaseFailed) {
		k.HandleCeremonyFailure(goCtx, msg.ProposalId)
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"zerone.emergency.vote_recovery_authorization",
		sdk.NewAttribute("ceremony_id", msg.ProposalId),
		sdk.NewAttribute("voter", msg.Voter),
		sdk.NewAttribute("approve", fmt.Sprintf("%t", msg.Approve)),
	))
	authorization, authorized, authErr :=
		k.GetRecoveryAuthorization(goCtx)
	if authErr != nil {
		return nil, authErr
	}
	return &types.MsgVoteRecoveryAuthorizationResponse{
		QuorumReached: updated.Phase == string(types.PhasePrecommit) ||
			updated.Phase == string(types.PhaseFinalized),
		RecoveryAuthorized: authorized &&
			updated.Phase == string(types.PhaseFinalized) &&
			authorization.AuthorizationCeremonyId == updated.Id &&
			authorization.Outcome == "",
	}, nil
}

// UpdateParams handles MsgUpdateParams — governance-gated parameter update.
func (k msgServer) UpdateParams(goCtx context.Context, msg *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	if k.GetAuthority() != msg.Authority {
		return nil, fmt.Errorf("unauthorized: expected %s, got %s", k.GetAuthority(), msg.Authority)
	}

	if msg.Params == nil {
		return nil, fmt.Errorf("params cannot be nil")
	}
	if err := msg.Params.Validate(); err != nil {
		return nil, fmt.Errorf("invalid emergency params: %w", err)
	}
	if err := validateMonotonicCouncilRetirement(
		goCtx,
		k.GetParams(goCtx),
		msg.Params,
	); err != nil {
		return nil, fmt.Errorf(
			"invalid emergency params for genesis council retirement: %w",
			err,
		)
	}
	if current := k.GetEpochProposalCount(goCtx); current > msg.Params.MaxProposalsPerEpoch {
		return nil, fmt.Errorf(
			"invalid emergency params for live state: epoch proposal count %d exceeds proposed maximum %d",
			current,
			msg.Params.MaxProposalsPerEpoch,
		)
	}
	for _, counter := range k.GetAllGuardianProposalCounts(goCtx) {
		if counter.Count > msg.Params.MaxProposalsPerGuardianPerEpoch {
			return nil, fmt.Errorf(
				"invalid emergency params for live state: guardian %q proposal count %d exceeds proposed maximum %d",
				counter.Guardian,
				counter.Count,
				msg.Params.MaxProposalsPerGuardianPerEpoch,
			)
		}
	}
	if marker := k.GetLastHaltEscalationBlock(goCtx); marker != 0 {
		start := k.GetHaltStartBlock(goCtx)
		if start == 0 ||
			msg.Params.MaxHaltDurationBlocks > ^uint64(0)-start ||
			start+msg.Params.MaxHaltDurationBlocks > marker {
			return nil, fmt.Errorf(
				"invalid emergency params for live state: first quarantine deadline from start %d with duration %d exceeds persisted escalation marker %d",
				start,
				msg.Params.MaxHaltDurationBlocks,
				marker,
			)
		}
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	k.SetParams(goCtx, msg.Params)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent("zerone.emergency.params_updated",
			sdk.NewAttribute("authority", msg.Authority),
		),
	)

	return &types.MsgUpdateParamsResponse{}, nil
}

// --- Helpers ---

// validateMonotonicCouncilRetirement keeps the genesis council a bootstrap
// authority with a one-way lifetime. Governance may shorten its expiry or
// retire it immediately, but cannot add/change members, change their virtual
// power, extend an expiry, or resurrect an expired/cleared council.
func validateMonotonicCouncilRetirement(
	goCtx context.Context,
	current *types.Params,
	proposed *types.Params,
) error {
	if current == nil || proposed == nil {
		return fmt.Errorf("current and proposed params must be present")
	}
	sdkCtx := sdk.UnwrapSDKContext(goCtx)
	height := uint64(0)
	if sdkCtx.BlockHeight() > 0 {
		height = uint64(sdkCtx.BlockHeight())
	}
	currentActive := len(current.GenesisCouncil) > 0 &&
		current.CouncilExpiryBlock > height

	if !currentActive {
		if len(proposed.GenesisCouncil) != 0 ||
			proposed.CouncilExpiryBlock != 0 {
			return fmt.Errorf(
				"expired or cleared genesis council cannot be re-enabled",
			)
		}
		return nil
	}

	if len(proposed.GenesisCouncil) == 0 {
		if proposed.CouncilExpiryBlock != 0 {
			return fmt.Errorf(
				"cleared genesis council must have zero expiry",
			)
		}
		return nil
	}
	if len(proposed.GenesisCouncil) != len(current.GenesisCouncil) {
		return fmt.Errorf("genesis council membership is immutable")
	}
	for i := range current.GenesisCouncil {
		if proposed.GenesisCouncil[i] != current.GenesisCouncil[i] {
			return fmt.Errorf("genesis council membership is immutable")
		}
	}
	currentPower, currentOK := new(big.Int).SetString(
		current.CouncilVirtualStake,
		10,
	)
	proposedPower, proposedOK := new(big.Int).SetString(
		proposed.CouncilVirtualStake,
		10,
	)
	if !currentOK || !proposedOK || currentPower.Cmp(proposedPower) != 0 {
		return fmt.Errorf("genesis council virtual stake is immutable")
	}
	if proposed.CouncilExpiryBlock > current.CouncilExpiryBlock {
		return fmt.Errorf(
			"genesis council expiry cannot increase from %d to %d",
			current.CouncilExpiryBlock,
			proposed.CouncilExpiryBlock,
		)
	}
	if proposed.CouncilExpiryBlock <= height {
		return fmt.Errorf(
			"genesis council expiring at or before current height %d must be cleared",
			height,
		)
	}
	return nil
}

func (k msgServer) addVoteAudit(ctx context.Context, action types.AuditAction, voter, ceremonyId string) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	k.AddAuditEntry(ctx, &types.EmergencyAuditEntry{
		Timestamp:   sdkCtx.BlockTime().Unix(),
		BlockNumber: uint64(sdkCtx.BlockHeight()),
		Action:      string(action),
		Actor:       voter,
		CeremonyId:  ceremonyId,
	})
}
