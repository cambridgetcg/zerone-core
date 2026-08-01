package keeper

import (
	"context"
	"fmt"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/protobuf/proto"

	"github.com/zerone-chain/zerone/x/emergency/types"
)

// CreateHaltCeremony creates a new halt ceremony.
func (k Keeper) CreateHaltCeremony(ctx context.Context, proposal *types.EmergencyHaltProposal) (*types.EmergencyCeremony, error) {
	if _, found := k.GetCeremony(ctx, proposal.Id); found {
		return nil, fmt.Errorf("%w: %s", types.ErrCeremonyIDExists, proposal.Id)
	}
	params := k.GetParams(ctx)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	if sdkCtx.BlockHeight() < 0 {
		return nil, fmt.Errorf("cannot create emergency ceremony at negative block height %d", sdkCtx.BlockHeight())
	}
	startBlock := uint64(sdkCtx.BlockHeight())
	prevoteDeadline, precommitDeadline, timeoutDeadline, err := ceremonyDeadlines(
		startBlock,
		params.HaltPrevoteBlocks,
		params.HaltPrecommitBlocks,
		params.HaltTimeoutBlocks,
	)
	if err != nil {
		return nil, fmt.Errorf("halt ceremony deadlines: %w", err)
	}
	electorate, totalPower, threshold, err := k.buildCeremonyElectorateSnapshot(
		ctx,
		params,
		types.CeremonyHalt,
	)
	if err != nil {
		return nil, fmt.Errorf("halt ceremony electorate: %w", err)
	}

	ceremony := types.EmergencyCeremony{
		Id:                        proposal.Id,
		Type:                      string(types.CeremonyHalt),
		Phase:                     string(types.PhasePrevote),
		StartBlock:                startBlock,
		PrevoteDeadline:           prevoteDeadline,
		PrecommitDeadline:         precommitDeadline,
		TimeoutDeadline:           timeoutDeadline,
		YesPrevoteStake:           "0",
		NoPrevoteStake:            "0",
		PrecommitStake:            "0",
		ElectorateSnapshotVersion: ceremonyElectorateSnapshotVersion,
		Electorate:                electorate,
		ElectorateTotalPower:      totalPower.String(),
		QuorumThreshold:           threshold,
		MinDistinctVoters:         params.MinDistinctVoters,
	}

	proposalData, err := marshalProposal(proposal)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal halt proposal: %w", err)
	}
	ceremony.ProposalData = proposalData

	if err := k.SetCeremony(ctx, &ceremony); err != nil {
		return nil, err
	}
	return &ceremony, nil
}

// CreateRevertCeremony creates a new revert ceremony.
func (k Keeper) CreateRevertCeremony(ctx context.Context, proposal *types.EmergencyRevertProposal) (*types.EmergencyCeremony, error) {
	if _, found := k.GetCeremony(ctx, proposal.Id); found {
		return nil, fmt.Errorf("%w: %s", types.ErrCeremonyIDExists, proposal.Id)
	}
	params := k.GetParams(ctx)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	if sdkCtx.BlockHeight() < 0 {
		return nil, fmt.Errorf("cannot create emergency ceremony at negative block height %d", sdkCtx.BlockHeight())
	}
	startBlock := uint64(sdkCtx.BlockHeight())
	prevoteDeadline, precommitDeadline, timeoutDeadline, err := ceremonyDeadlines(
		startBlock,
		params.RevertPrevoteBlocks,
		params.RevertPrecommitBlocks,
		params.RevertTimeoutBlocks,
	)
	if err != nil {
		return nil, fmt.Errorf("revert ceremony deadlines: %w", err)
	}
	electorate, totalPower, threshold, err := k.buildCeremonyElectorateSnapshot(
		ctx,
		params,
		types.CeremonyRevert,
	)
	if err != nil {
		return nil, fmt.Errorf("revert ceremony electorate: %w", err)
	}

	ceremony := types.EmergencyCeremony{
		Id:                        proposal.Id,
		Type:                      string(types.CeremonyRevert),
		Phase:                     string(types.PhasePrevote),
		StartBlock:                startBlock,
		PrevoteDeadline:           prevoteDeadline,
		PrecommitDeadline:         precommitDeadline,
		TimeoutDeadline:           timeoutDeadline,
		YesPrevoteStake:           "0",
		NoPrevoteStake:            "0",
		PrecommitStake:            "0",
		ElectorateSnapshotVersion: ceremonyElectorateSnapshotVersion,
		Electorate:                electorate,
		ElectorateTotalPower:      totalPower.String(),
		QuorumThreshold:           threshold,
		MinDistinctVoters:         params.MinDistinctVoters,
	}

	proposalData, err := marshalProposal(proposal)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal revert proposal: %w", err)
	}
	ceremony.ProposalData = proposalData

	if err := k.SetCeremony(ctx, &ceremony); err != nil {
		return nil, err
	}
	return &ceremony, nil
}

// CreateResumeCeremony creates a new resume ceremony.
func (k Keeper) CreateResumeCeremony(ctx context.Context, proposal *types.EmergencyResumeProposal) (*types.EmergencyCeremony, error) {
	if _, found := k.GetCeremony(ctx, proposal.Id); found {
		return nil, fmt.Errorf("%w: %s", types.ErrCeremonyIDExists, proposal.Id)
	}
	params := k.GetParams(ctx)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	if sdkCtx.BlockHeight() < 0 {
		return nil, fmt.Errorf("cannot create emergency ceremony at negative block height %d", sdkCtx.BlockHeight())
	}
	startBlock := uint64(sdkCtx.BlockHeight())
	prevoteDeadline, precommitDeadline, timeoutDeadline, err := ceremonyDeadlines(
		startBlock,
		params.ResumePrevoteBlocks,
		params.ResumePrecommitBlocks,
		params.ResumeTimeoutBlocks,
	)
	if err != nil {
		return nil, fmt.Errorf("resume ceremony deadlines: %w", err)
	}
	electorate, totalPower, threshold, err := k.buildCeremonyElectorateSnapshot(
		ctx,
		params,
		types.CeremonyResume,
	)
	if err != nil {
		return nil, fmt.Errorf("resume ceremony electorate: %w", err)
	}

	ceremony := types.EmergencyCeremony{
		Id:                        proposal.Id,
		Type:                      string(types.CeremonyResume),
		Phase:                     string(types.PhasePrevote),
		StartBlock:                startBlock,
		PrevoteDeadline:           prevoteDeadline,
		PrecommitDeadline:         precommitDeadline,
		TimeoutDeadline:           timeoutDeadline,
		YesPrevoteStake:           "0",
		NoPrevoteStake:            "0",
		PrecommitStake:            "0",
		ElectorateSnapshotVersion: ceremonyElectorateSnapshotVersion,
		Electorate:                electorate,
		ElectorateTotalPower:      totalPower.String(),
		QuorumThreshold:           threshold,
		MinDistinctVoters:         params.MinDistinctVoters,
	}

	proposalData, err := marshalProposal(proposal)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal resume proposal: %w", err)
	}
	ceremony.ProposalData = proposalData

	if err := k.SetCeremony(ctx, &ceremony); err != nil {
		return nil, err
	}
	return &ceremony, nil
}

// CreateRecoveryAuthorizationCeremony creates a Guardian ceremony that can
// authorize one exact SDK-governance action without reopening transaction
// admission. Recovery uses the stricter resume quorum and timing policy.
func (k Keeper) CreateRecoveryAuthorizationCeremony(
	ctx context.Context,
	proposal *types.EmergencyRecoveryAuthorizationProposal,
) (*types.EmergencyCeremony, error) {
	if _, found := k.GetCeremony(ctx, proposal.Id); found {
		return nil, fmt.Errorf("%w: %s", types.ErrCeremonyIDExists, proposal.Id)
	}
	params := k.GetParams(ctx)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	if sdkCtx.BlockHeight() <= 0 {
		return nil, fmt.Errorf(
			"cannot create recovery authorization ceremony at non-positive block height %d",
			sdkCtx.BlockHeight(),
		)
	}
	startBlock := uint64(sdkCtx.BlockHeight())
	prevoteDeadline, precommitDeadline, timeoutDeadline, err := ceremonyDeadlines(
		startBlock,
		params.ResumePrevoteBlocks,
		params.ResumePrecommitBlocks,
		params.ResumeTimeoutBlocks,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"recovery authorization ceremony deadlines: %w",
			err,
		)
	}
	electorate, totalPower, threshold, err := k.buildCeremonyElectorateSnapshot(
		ctx,
		params,
		types.CeremonyRecoveryAuthorization,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"recovery authorization ceremony electorate: %w",
			err,
		)
	}
	proposalData, err := marshalProposal(proposal)
	if err != nil {
		return nil, fmt.Errorf(
			"marshal recovery authorization proposal: %w",
			err,
		)
	}
	ceremony := &types.EmergencyCeremony{
		Id:                        proposal.Id,
		Type:                      string(types.CeremonyRecoveryAuthorization),
		Phase:                     string(types.PhasePrevote),
		ProposalData:              proposalData,
		StartBlock:                startBlock,
		PrevoteDeadline:           prevoteDeadline,
		PrecommitDeadline:         precommitDeadline,
		TimeoutDeadline:           timeoutDeadline,
		YesPrevoteStake:           "0",
		NoPrevoteStake:            "0",
		PrecommitStake:            "0",
		ElectorateSnapshotVersion: ceremonyElectorateSnapshotVersion,
		Electorate:                electorate,
		ElectorateTotalPower:      totalPower.String(),
		QuorumThreshold:           threshold,
		MinDistinctVoters:         params.MinDistinctVoters,
	}
	if err := k.SetCeremony(ctx, ceremony); err != nil {
		return nil, err
	}
	return ceremony, nil
}

func ceremonyDeadlines(startBlock, prevoteBlocks, precommitBlocks, timeoutBlocks uint64) (uint64, uint64, uint64, error) {
	if prevoteBlocks > ^uint64(0)-startBlock {
		return 0, 0, 0, fmt.Errorf("prevote deadline overflows uint64")
	}
	prevoteDeadline := startBlock + prevoteBlocks
	if precommitBlocks > ^uint64(0)-prevoteDeadline {
		return 0, 0, 0, fmt.Errorf("precommit deadline overflows uint64")
	}
	precommitDeadline := prevoteDeadline + precommitBlocks
	if timeoutBlocks > ^uint64(0)-startBlock {
		return 0, 0, 0, fmt.Errorf("timeout deadline overflows uint64")
	}
	timeoutDeadline := startBlock + timeoutBlocks
	if timeoutDeadline < precommitDeadline {
		return 0, 0, 0, fmt.Errorf("timeout deadline precedes precommit deadline")
	}
	return prevoteDeadline, precommitDeadline, timeoutDeadline, nil
}

// AddPrevote adds a prevote to a ceremony.
func (k Keeper) AddPrevote(ctx context.Context, ceremonyId string, vote *types.EmergencyVote) error {
	ceremony, found := k.GetCeremony(ctx, ceremonyId)
	if !found {
		return fmt.Errorf("%w: %s", types.ErrNoCeremony, ceremonyId)
	}

	if ceremony.Phase != string(types.PhasePrevote) {
		return fmt.Errorf("%w: expected prevote, got %s", types.ErrInvalidPhase, ceremony.Phase)
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	currentBlock := uint64(sdkCtx.BlockHeight())
	if currentBlock > ceremony.PrevoteDeadline {
		return fmt.Errorf("%w: prevote deadline passed", types.ErrCeremonyTimedOut)
	}

	electorate, _, err := validateCeremonySnapshot(ceremony)
	if err != nil {
		return fmt.Errorf("invalid ceremony electorate snapshot: %w", err)
	}
	if err := validateCeremonyTallies(ceremony, electorate); err != nil {
		return fmt.Errorf("invalid ceremony vote tally: %w", err)
	}
	voterStake, eligible := electorate[vote.Voter]
	if !eligible {
		return fmt.Errorf("%w: %s", types.ErrNotGuardian, vote.Voter)
	}

	if _, exists := ceremony.GetPrevote(vote.Voter); exists {
		return fmt.Errorf("%w: %s", types.ErrDuplicateVote, vote.Voter)
	}

	ceremony.SetPrevote(vote.Voter, vote)

	if vote.Approve {
		yesStake, _ := new(big.Int).SetString(ceremony.YesPrevoteStake, 10)
		if yesStake == nil {
			yesStake = new(big.Int)
		}
		yesStake.Add(yesStake, voterStake)
		ceremony.YesPrevoteStake = yesStake.String()
	} else {
		noStake, _ := new(big.Int).SetString(ceremony.NoPrevoteStake, 10)
		if noStake == nil {
			noStake = new(big.Int)
		}
		noStake.Add(noStake, voterStake)
		ceremony.NoPrevoteStake = noStake.String()
	}

	return k.SetCeremony(ctx, ceremony)
}

// AddPrecommit adds a precommit to a ceremony.
func (k Keeper) AddPrecommit(ctx context.Context, ceremonyId string, precommit *types.EmergencyPrecommit) error {
	ceremony, found := k.GetCeremony(ctx, ceremonyId)
	if !found {
		return fmt.Errorf("%w: %s", types.ErrNoCeremony, ceremonyId)
	}

	if ceremony.Phase != string(types.PhasePrecommit) {
		return fmt.Errorf("%w: expected precommit, got %s", types.ErrInvalidPhase, ceremony.Phase)
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	currentBlock := uint64(sdkCtx.BlockHeight())
	if currentBlock > ceremony.PrecommitDeadline {
		return fmt.Errorf("%w: precommit deadline passed", types.ErrCeremonyTimedOut)
	}

	electorate, _, err := validateCeremonySnapshot(ceremony)
	if err != nil {
		return fmt.Errorf("invalid ceremony electorate snapshot: %w", err)
	}
	if err := validateCeremonyTallies(ceremony, electorate); err != nil {
		return fmt.Errorf("invalid ceremony vote tally: %w", err)
	}
	voterStake, eligible := electorate[precommit.Voter]
	if !eligible {
		return fmt.Errorf("%w: %s", types.ErrNotGuardian, precommit.Voter)
	}

	// Must have prevoted "yes".
	prevote, hasPrevoted := ceremony.GetPrevote(precommit.Voter)
	if !hasPrevoted || !prevote.Approve {
		return fmt.Errorf("%w: %s", types.ErrPrevoteRequired, precommit.Voter)
	}

	if _, exists := ceremony.GetPrecommit(precommit.Voter); exists {
		return fmt.Errorf("%w: %s", types.ErrDuplicateVote, precommit.Voter)
	}

	ceremony.SetPrecommit(precommit.Voter, precommit)

	precommitStake, _ := new(big.Int).SetString(ceremony.PrecommitStake, 10)
	if precommitStake == nil {
		precommitStake = new(big.Int)
	}
	precommitStake.Add(precommitStake, voterStake)
	ceremony.PrecommitStake = precommitStake.String()

	return k.SetCeremony(ctx, ceremony)
}

// CheckCeremonyProgress checks and advances ceremony phase. Returns true if finalized.
func (k Keeper) CheckCeremonyProgress(ctx context.Context, ceremonyId string) (bool, error) {
	ceremony, found := k.GetCeremony(ctx, ceremonyId)
	if !found {
		return false, fmt.Errorf("%w: %s", types.ErrNoCeremony, ceremonyId)
	}

	if ceremony.Phase == string(types.PhaseFinalized) || ceremony.Phase == string(types.PhaseFailed) {
		return ceremony.Phase == string(types.PhaseFinalized), nil
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	if sdkCtx.BlockHeight() < 0 {
		return false, fmt.Errorf("cannot progress emergency ceremony at negative block height %d", sdkCtx.BlockHeight())
	}
	currentBlock := uint64(sdkCtx.BlockHeight())

	// Overall timeout.
	if currentBlock > ceremony.TimeoutDeadline {
		ceremony.Phase = string(types.PhaseFailed)
		ceremony.FailureReason = "ceremony timed out"
		k.mustSetCeremony(ctx, ceremony)
		return false, nil
	}

	electorate, totalPower, err := validateCeremonySnapshot(ceremony)
	if err != nil {
		ceremony.Phase = string(types.PhaseFailed)
		ceremony.FailureReason = "invalid or legacy electorate snapshot: " + err.Error()
		k.mustSetCeremony(ctx, ceremony)
		return false, nil
	}
	if err := validateCeremonyTallies(ceremony, electorate); err != nil {
		ceremony.Phase = string(types.PhaseFailed)
		ceremony.FailureReason = "invalid ceremony vote tally: " + err.Error()
		k.mustSetCeremony(ctx, ceremony)
		return false, nil
	}

	switch ceremony.Phase {
	case string(types.PhasePrevote):
		return k.checkPrevotePhase(
			ctx,
			ceremony,
			currentBlock,
			totalPower,
			ceremony.QuorumThreshold,
		)
	case string(types.PhasePrecommit):
		return k.checkPrecommitPhase(
			ctx,
			ceremony,
			currentBlock,
			totalPower,
			ceremony.QuorumThreshold,
			ceremony.MinDistinctVoters,
		)
	}

	return false, fmt.Errorf("%w: unknown persisted ceremony phase %q", types.ErrInvalidPhase, ceremony.Phase)
}

func (k Keeper) mustSetCeremony(ctx context.Context, ceremony *types.EmergencyCeremony) {
	if err := k.SetCeremony(ctx, ceremony); err != nil {
		panic("failed to persist emergency ceremony state transition: " + err.Error())
	}
}

func (k Keeper) checkPrevotePhase(ctx context.Context, ceremony *types.EmergencyCeremony, currentBlock uint64, guardianStake *big.Int, threshold uint64) (bool, error) {
	yesStake, _ := new(big.Int).SetString(ceremony.YesPrevoteStake, 10)
	if yesStake == nil {
		yesStake = new(big.Int)
	}
	noStake, _ := new(big.Int).SetString(ceremony.NoPrevoteStake, 10)
	if noStake == nil {
		noStake = new(big.Int)
	}

	// Check if yes quorum reached → advance to precommit.
	yesRatio := new(big.Int).Mul(yesStake, big.NewInt(1000000))
	yesRatio.Div(yesRatio, guardianStake)
	if yesRatio.Uint64() >= threshold {
		ceremony.Phase = string(types.PhasePrecommit)
		k.mustSetCeremony(ctx, ceremony)

		sdkCtx := sdk.UnwrapSDKContext(ctx)
		sdkCtx.EventManager().EmitEvent(
			sdk.NewEvent("zerone.emergency.ceremony_advanced",
				sdk.NewAttribute("ceremony_id", ceremony.Id),
				sdk.NewAttribute("ceremony_type", ceremony.Type),
				sdk.NewAttribute("phase", string(types.PhasePrecommit)),
				sdk.NewAttribute("yes_prevote_stake", ceremony.YesPrevoteStake),
			),
		)

		return false, nil
	}

	// Check if quorum is impossible (too many no votes).
	noRatio := new(big.Int).Mul(noStake, big.NewInt(1000000))
	noRatio.Div(noRatio, guardianStake)
	if noRatio.Uint64() > (1000000 - threshold) {
		ceremony.Phase = string(types.PhaseFailed)
		ceremony.FailureReason = "quorum impossible: too many no votes"
		k.mustSetCeremony(ctx, ceremony)
		return false, nil
	}

	// Check prevote deadline.
	if currentBlock > ceremony.PrevoteDeadline {
		ceremony.Phase = string(types.PhaseFailed)
		ceremony.FailureReason = "prevote quorum not reached before deadline"
		k.mustSetCeremony(ctx, ceremony)
		return false, nil
	}

	return false, nil
}

func (k Keeper) checkPrecommitPhase(
	ctx context.Context,
	ceremony *types.EmergencyCeremony,
	currentBlock uint64,
	guardianStake *big.Int,
	threshold uint64,
	minDistinctVoters uint64,
) (bool, error) {
	precommitStake, _ := new(big.Int).SetString(ceremony.PrecommitStake, 10)
	if precommitStake == nil {
		precommitStake = new(big.Int)
	}

	precommitRatio := new(big.Int).Mul(precommitStake, big.NewInt(1000000))
	precommitRatio.Div(precommitRatio, guardianStake)

	distinctVoters := uint64(len(ceremony.Precommits))

	if precommitRatio.Uint64() >= threshold {
		if distinctVoters >= minDistinctVoters {
			ceremony.Phase = string(types.PhaseFinalized)
			k.mustSetCeremony(ctx, ceremony)
			return true, nil
		}
	}

	if currentBlock > ceremony.PrecommitDeadline {
		if precommitRatio.Uint64() >= threshold && distinctVoters < minDistinctVoters {
			ceremony.Phase = string(types.PhaseFailed)
			ceremony.FailureReason = fmt.Sprintf("insufficient distinct voters: %d < %d", distinctVoters, minDistinctVoters)
		} else {
			ceremony.Phase = string(types.PhaseFailed)
			ceremony.FailureReason = "precommit quorum not reached before deadline"
		}
		k.mustSetCeremony(ctx, ceremony)
		return false, nil
	}

	return false, nil
}

// HandleCeremonyFinalization processes a finalized ceremony's effects.
func (k Keeper) HandleCeremonyFinalization(ctx context.Context, ceremonyId string) {
	ceremony, found := k.GetCeremony(ctx, ceremonyId)
	if !found || ceremony.Phase != string(types.PhaseFinalized) {
		return
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	status := k.GetEmergencyStatus(ctx)

	switch types.CeremonyType(ceremony.Type) {
	case types.CeremonyHalt:
		if status != types.StatusHaltVoting {
			k.Logger(ctx).Error(
				"refusing finalized halt ceremony effect in unexpected emergency state",
				"ceremony_id", ceremony.Id,
				"status", status,
			)
			return
		}
		if err := k.ClearRecoveryAuthorization(ctx); err != nil {
			panic("failed to clear stale recovery authorization: " + err.Error())
		}
		k.SetEmergencyStatus(ctx, types.StatusHalted)
		k.SetActiveHaltCeremonyId(ctx, ceremony.Id)
		k.SetHaltStartBlock(ctx, uint64(sdkCtx.BlockHeight()))
		k.AddAuditEntry(ctx, &types.EmergencyAuditEntry{
			Timestamp:   sdkCtx.BlockTime().Unix(),
			BlockNumber: uint64(sdkCtx.BlockHeight()),
			Action:      string(types.AuditHaltExecuted),
			Actor:       "system",
			CeremonyId:  ceremony.Id,
			Details:     "application transaction quarantine finalized; CometBFT consensus continues",
		})
		// Forward-only audit: halt ceremonies are immutable
		// post-resolve. The chain announces the moment the halt
		// takes effect so off-chain observers know this is a
		// chain-recognised emergency, not a private decision. See
		// TRUTH_SEEKING.md commitment 10.
		sdkCtx.EventManager().EmitEvent(
			sdk.NewEvent("zerone.emergency.ceremony_finalized",
				sdk.NewAttribute("ceremony_id", ceremony.Id),
				sdk.NewAttribute("ceremony_type", string(types.CeremonyHalt)),
				sdk.NewAttribute("status", string(types.StatusHalted)),
				sdk.NewAttribute("restriction_scope", "application_transactions"),
				sdk.NewAttribute("consensus_continues", "true"),
				sdk.NewAttribute("block_height", fmt.Sprintf("%d", sdkCtx.BlockHeight())),
				sdk.NewAttribute("creed_commitment", "10"),
			),
		)

	case types.CeremonyRevert:
		if status != types.StatusRevertVoting && status != types.StatusReverting {
			k.Logger(ctx).Error(
				"refusing finalized legacy revert ceremony effect in unexpected emergency state",
				"ceremony_id", ceremony.Id,
				"status", status,
			)
			return
		}
		// Legacy height-only revert ceremonies cannot authenticate a
		// canonical block/AppHash or preserve IBC finality. Keep transaction
		// admission quarantined and record the refusal; recovery must be a
		// hash-bound forward upgrade or an explicit social fork.
		k.SetEmergencyStatus(ctx, types.StatusHalted)
		var proposal types.EmergencyRevertProposal
		if err := unmarshalProposal(ceremony.ProposalData, &proposal); err != nil {
			panic("failed to decode finalized legacy revert proposal: " + err.Error())
		}
		k.ClearRevertTarget(ctx)
		sdkCtx.EventManager().EmitEvent(
			sdk.NewEvent(
				"zerone.emergency.revert_refused",
				sdk.NewAttribute("ceremony_id", ceremony.Id),
				sdk.NewAttribute("target_height", fmt.Sprintf("%d", proposal.TargetBlockNumber)),
				sdk.NewAttribute("status", string(types.StatusHalted)),
				sdk.NewAttribute("action", "remain quarantined; prepare a hash-bound forward recovery or explicit social fork"),
				sdk.NewAttribute("creed_commitment", "10"),
			),
		)

		k.AddAuditEntry(ctx, &types.EmergencyAuditEntry{
			Timestamp:   sdkCtx.BlockTime().Unix(),
			BlockNumber: uint64(sdkCtx.BlockHeight()),
			Action:      string(types.AuditRevertFailed),
			Actor:       "system",
			CeremonyId:  ceremony.Id,
			Details:     fmt.Sprintf("legacy height-only revert to %d refused; quarantine remains active", proposal.TargetBlockNumber),
		})

	case types.CeremonyResume:
		if status != types.StatusResumeVoting {
			k.Logger(ctx).Error(
				"refusing finalized resume ceremony effect in unexpected emergency state",
				"ceremony_id", ceremony.Id,
				"status", status,
			)
			return
		}
		var proposal types.EmergencyResumeProposal
		if err := unmarshalProposal(ceremony.ProposalData, &proposal); err != nil ||
			!types.IsLowerSHA256(proposal.RecoveryManifestSha256) ||
			proposal.Justification == "" ||
			proposal.HaltCeremonyId == "" ||
			proposal.HaltCeremonyId != k.GetActiveHaltCeremonyId(ctx) {
			// A malformed or pre-hardening resume record must never reopen
			// transaction admission.
			k.SetEmergencyStatus(ctx, types.StatusHalted)
			k.AddAuditEntry(ctx, &types.EmergencyAuditEntry{
				Timestamp:   sdkCtx.BlockTime().Unix(),
				BlockNumber: uint64(sdkCtx.BlockHeight()),
				Action:      string(types.AuditResumeFailed),
				Actor:       "system",
				CeremonyId:  ceremony.Id,
				Details:     "resume refused: invalid recovery manifest commitment or incident linkage",
			})
			return
		}
		releaseBlock := uint64(sdkCtx.BlockHeight())
		k.SetQuarantineReleaseBlock(ctx, releaseBlock)
		k.SetEmergencyStatus(ctx, types.StatusNormal)
		k.SetActiveHaltCeremonyId(ctx, "")
		k.ClearHaltStartBlock(ctx)
		k.ClearRevertTarget(ctx)
		if err := k.ClearRecoveryAuthorization(ctx); err != nil {
			panic("failed to clear closed-incident recovery authorization: " + err.Error())
		}
		k.AddAuditEntry(ctx, &types.EmergencyAuditEntry{
			Timestamp:   sdkCtx.BlockTime().Unix(),
			BlockNumber: uint64(sdkCtx.BlockHeight()),
			Action:      string(types.AuditResumeExecuted),
			Actor:       "system",
			CeremonyId:  ceremony.Id,
			Details: fmt.Sprintf(
				"resume finalized; transaction admission remains quarantined through block %d and reopens at block %d; halt_ceremony_id=%s; recovery_manifest_sha256=%s",
				releaseBlock,
				releaseBlock+1,
				proposal.HaltCeremonyId,
				proposal.RecoveryManifestSha256,
			),
		})
		// Forward-only audit: resume ceremonies close the halt
		// record. The announcement marks the chain as having
		// re-entered normal status; the halt-to-resume transition
		// is a public, dated, signed fact. See TRUTH_SEEKING.md
		// commitment 10.
		sdkCtx.EventManager().EmitEvent(
			sdk.NewEvent("zerone.emergency.ceremony_finalized",
				sdk.NewAttribute("ceremony_id", ceremony.Id),
				sdk.NewAttribute("ceremony_type", string(types.CeremonyResume)),
				sdk.NewAttribute("status", string(types.StatusNormal)),
				sdk.NewAttribute("block_height", fmt.Sprintf("%d", sdkCtx.BlockHeight())),
				sdk.NewAttribute("admission_reopens_at_block", fmt.Sprintf("%d", releaseBlock+1)),
				sdk.NewAttribute("recovery_manifest_sha256", proposal.RecoveryManifestSha256),
				sdk.NewAttribute("creed_commitment", "10"),
			),
		)
	case types.CeremonyRecoveryAuthorization:
		if status != types.StatusHalted {
			k.Logger(ctx).Error(
				"refusing finalized recovery authorization in unexpected emergency state",
				"ceremony_id", ceremony.Id,
				"status", status,
			)
			return
		}
		var proposal types.EmergencyRecoveryAuthorizationProposal
		if err := unmarshalProposal(ceremony.ProposalData, &proposal); err != nil ||
			proposal.Id != ceremony.Id ||
			proposal.HaltCeremonyId == "" ||
			proposal.HaltCeremonyId != k.GetActiveHaltCeremonyId(ctx) ||
			proposal.SdkGovProposalId == 0 ||
			!types.IsLowerSHA256(proposal.ActionSha256) ||
			!types.IsLowerSHA256(proposal.RecoveryManifestSha256) ||
			!types.IsLowerSHA256(proposal.UpgradePlanSha256) ||
			proposal.AuthorizedSubmitter == "" ||
			proposal.Generation == 0 ||
			(proposal.ActionType != "software_upgrade" &&
				proposal.ActionType != "cancel_upgrade" &&
				proposal.ActionType != "revoke") ||
			proposal.Justification == "" {
			k.Logger(ctx).Error(
				"refusing malformed finalized recovery authorization",
				"ceremony_id", ceremony.Id,
			)
			ceremony.Phase = string(types.PhaseFailed)
			ceremony.FailureReason =
				"malformed finalized recovery authorization was refused"
			k.mustSetCeremony(ctx, ceremony)
			k.HandleCeremonyFailure(ctx, ceremony.Id)
			return
		}
		if k.recoveryTargetVerifier == nil {
			ceremony.Phase = string(types.PhaseFailed)
			ceremony.FailureReason =
				"recovery authorization target verifier is unavailable"
			k.mustSetCeremony(ctx, ceremony)
			k.HandleCeremonyFailure(ctx, ceremony.Id)
			return
		}
		if err := k.recoveryTargetVerifier.VerifyRecoveryAuthorizationTarget(
			ctx,
			proposal.SdkGovProposalId,
			proposal.ActionSha256,
			proposal.UpgradePlanSha256,
			proposal.ActionType,
		); err != nil {
			ceremony.Phase = string(types.PhaseFailed)
			ceremony.FailureReason =
				"recovery authorization target changed before finalization: " +
					err.Error()
			k.mustSetCeremony(ctx, ceremony)
			k.HandleCeremonyFailure(ctx, ceremony.Id)
			return
		}
		if proposal.ActionType == "revoke" {
			current, found, err := k.GetRecoveryAuthorization(ctx)
			if err != nil {
				panic(
					"failed to read recovery authorization being revoked: " +
						err.Error(),
				)
			}
			if !found ||
				current.Outcome != "" ||
				current.HaltCeremonyId != proposal.HaltCeremonyId ||
				current.SdkGovProposalId != proposal.SdkGovProposalId ||
				current.ActionSha256 != proposal.ActionSha256 ||
				current.RecoveryManifestSha256 !=
					proposal.RecoveryManifestSha256 ||
				current.UpgradePlanSha256 !=
					proposal.UpgradePlanSha256 ||
				current.AuthorizedSubmitter !=
					proposal.AuthorizedSubmitter {
				ceremony.Phase = string(types.PhaseFailed)
				ceremony.FailureReason =
					"recovery revocation no longer matches the live authorization"
				k.mustSetCeremony(ctx, ceremony)
				k.HandleCeremonyFailure(ctx, ceremony.Id)
				return
			}
			if err := k.MarkRecoveryAuthorizationTerminal(
				ctx,
				current.SdkGovProposalId,
				current.ActionSha256,
				"revoked",
			); err != nil {
				panic(
					"failed to revoke recovery authorization: " +
						err.Error(),
				)
			}
			k.AddAuditEntry(ctx, &types.EmergencyAuditEntry{
				Timestamp:   sdkCtx.BlockTime().Unix(),
				BlockNumber: uint64(sdkCtx.BlockHeight()),
				Action:      string(types.AuditRecoveryRevoked),
				Actor:       "system",
				CeremonyId:  ceremony.Id,
				Details: fmt.Sprintf(
					"revoked recovery authorization %s for SDK governance proposal %d; any submitted proposal is terminalized before execution",
					current.AuthorizationCeremonyId,
					current.SdkGovProposalId,
				),
			})
			sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
				"zerone.emergency.recovery_revoked",
				sdk.NewAttribute("ceremony_id", ceremony.Id),
				sdk.NewAttribute(
					"revoked_authorization_ceremony_id",
					current.AuthorizationCeremonyId,
				),
				sdk.NewAttribute(
					"sdk_gov_proposal_id",
					fmt.Sprintf("%d", current.SdkGovProposalId),
				),
				sdk.NewAttribute("action_sha256", current.ActionSha256),
				sdk.NewAttribute(
					"transaction_admission",
					"recovery_capability_closed",
				),
			))
			return
		}
		if sdkCtx.BlockHeight() <= 0 {
			panic("finalized recovery authorization has non-positive block height")
		}
		previous, supersedes, err := k.GetRecoveryAuthorization(ctx)
		if err != nil {
			panic("failed to read prior recovery authorization: " + err.Error())
		}
		authorization := &types.EmergencyRecoveryAuthorization{
			HaltCeremonyId:          proposal.HaltCeremonyId,
			AuthorizationCeremonyId: ceremony.Id,
			SdkGovProposalId:        proposal.SdkGovProposalId,
			ActionSha256:            proposal.ActionSha256,
			RecoveryManifestSha256:  proposal.RecoveryManifestSha256,
			AuthorizedAtBlock:       uint64(sdkCtx.BlockHeight()),
			UpgradePlanSha256:       proposal.UpgradePlanSha256,
			AuthorizedSubmitter:     proposal.AuthorizedSubmitter,
			ActionType:              proposal.ActionType,
			Generation:              proposal.Generation,
		}
		if err := k.setRecoveryAuthorization(ctx, authorization); err != nil {
			panic("failed to persist recovery authorization: " + err.Error())
		}
		supersededID := ""
		if supersedes {
			supersededID = previous.AuthorizationCeremonyId
		}
		k.AddAuditEntry(ctx, &types.EmergencyAuditEntry{
			Timestamp:   sdkCtx.BlockTime().Unix(),
			BlockNumber: uint64(sdkCtx.BlockHeight()),
			Action:      string(types.AuditRecoveryAuthorized),
			Actor:       "system",
			CeremonyId:  ceremony.Id,
			Details: fmt.Sprintf(
				"authorized recovery generation %d for sdk governance proposal %d and quarantine %s; action_type=%s; action_sha256=%s; upgrade_plan_sha256=%s; recovery_manifest_sha256=%s; authorized_submitter=%s; superseded_authorization_ceremony_id=%s",
				proposal.Generation,
				proposal.SdkGovProposalId,
				proposal.HaltCeremonyId,
				proposal.ActionType,
				proposal.ActionSha256,
				proposal.UpgradePlanSha256,
				proposal.RecoveryManifestSha256,
				proposal.AuthorizedSubmitter,
				supersededID,
			),
		})
		sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
			"zerone.emergency.recovery_authorized",
			sdk.NewAttribute("ceremony_id", ceremony.Id),
			sdk.NewAttribute(
				"halt_ceremony_id",
				proposal.HaltCeremonyId,
			),
			sdk.NewAttribute(
				"sdk_gov_proposal_id",
				fmt.Sprintf("%d", proposal.SdkGovProposalId),
			),
			sdk.NewAttribute(
				"generation",
				fmt.Sprintf("%d", proposal.Generation),
			),
			sdk.NewAttribute("action_sha256", proposal.ActionSha256),
			sdk.NewAttribute("action_type", proposal.ActionType),
			sdk.NewAttribute(
				"upgrade_plan_sha256",
				proposal.UpgradePlanSha256,
			),
			sdk.NewAttribute(
				"recovery_manifest_sha256",
				proposal.RecoveryManifestSha256,
			),
			sdk.NewAttribute(
				"superseded_authorization_ceremony_id",
				supersededID,
			),
			sdk.NewAttribute(
				"authorized_submitter",
				proposal.AuthorizedSubmitter,
			),
			sdk.NewAttribute("transaction_admission", "remains_quarantined"),
		))
	}
}

// HandleCeremonyFailure processes a failed ceremony's effects.
func (k Keeper) HandleCeremonyFailure(ctx context.Context, ceremonyId string) {
	ceremony, found := k.GetCeremony(ctx, ceremonyId)
	if !found || ceremony.Phase != string(types.PhaseFailed) {
		return
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	status := k.GetEmergencyStatus(ctx)
	var auditAction types.AuditAction
	switch types.CeremonyType(ceremony.Type) {
	case types.CeremonyHalt:
		if status == types.StatusHaltVoting {
			k.SetEmergencyStatus(ctx, types.StatusNormal)
		}
		auditAction = types.AuditHaltFailed
	case types.CeremonyRevert:
		if status == types.StatusRevertVoting || status == types.StatusReverting {
			k.SetEmergencyStatus(ctx, types.StatusHalted)
			k.ClearRevertTarget(ctx)
		}
		auditAction = types.AuditRevertFailed
	case types.CeremonyResume:
		if status == types.StatusResumeVoting {
			k.SetEmergencyStatus(ctx, types.StatusHalted)
		}
		auditAction = types.AuditResumeFailed
	case types.CeremonyRecoveryAuthorization:
		auditAction = types.AuditRecoveryAuthorizationFailed
	default:
		k.Logger(ctx).Error(
			"refusing failed ceremony effect for unknown type",
			"ceremony_id", ceremony.Id,
			"ceremony_type", ceremony.Type,
		)
		return
	}

	k.AddAuditEntry(ctx, &types.EmergencyAuditEntry{
		Timestamp:   sdkCtx.BlockTime().Unix(),
		BlockNumber: uint64(sdkCtx.BlockHeight()),
		Action:      string(auditAction),
		Actor:       "system",
		CeremonyId:  ceremony.Id,
		Details:     fmt.Sprintf("ceremony failed: %s", ceremony.FailureReason),
	})
}

// --- Helpers ---

func getQuorumThreshold(ceremonyType types.CeremonyType, params *types.Params) uint64 {
	switch ceremonyType {
	case types.CeremonyHalt:
		return params.HaltQuorum
	case types.CeremonyRevert:
		return params.RevertQuorum
	case types.CeremonyResume:
		return params.ResumeQuorum
	case types.CeremonyRecoveryAuthorization:
		return params.ResumeQuorum
	default:
		return 800000
	}
}

// CheckHaltExpiry alerts when a transaction quarantine exceeds its escalation
// deadline. It never reopens transaction admission: elapsed time is not proof
// that a hostile event has ended.
func (k Keeper) CheckHaltExpiry(ctx context.Context) {
	if !k.IsHalted(ctx) {
		return
	}

	params := k.GetParams(ctx)
	if params.MaxHaltDurationBlocks == 0 {
		return
	}

	haltStart := k.GetHaltStartBlock(ctx)
	if haltStart == 0 {
		return
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	currentHeight := uint64(sdkCtx.BlockHeight())
	if currentHeight < haltStart {
		return
	}

	elapsed := currentHeight - haltStart
	periods := elapsed / params.MaxHaltDurationBlocks
	if periods == 0 {
		return
	}
	boundary := haltStart + periods*params.MaxHaltDurationBlocks
	if k.GetLastHaltEscalationBlock(ctx) >= boundary {
		return
	}
	k.SetLastHaltEscalationBlock(ctx, boundary)
	k.Logger(ctx).Error("TRANSACTION QUARANTINE ESCALATION DEADLINE EXCEEDED — explicit evidence-bound resume still required",
		"quarantine_started_at", haltStart,
		"current_height", currentHeight,
		"elapsed_blocks", elapsed,
		"deadline_blocks", params.MaxHaltDurationBlocks,
		"reported_boundary", boundary,
	)
}

// NormalizeLegacyQuarantineState repairs live state admitted by older
// binaries before any active ceremony is progressed. It is idempotent and
// never reopens transaction admission.
func (k Keeper) NormalizeLegacyQuarantineState(ctx context.Context) {
	status := k.GetEmergencyStatus(ctx)
	if status == types.StatusRevertVoting || status == types.StatusReverting {
		k.MonitorRevertStatus(ctx)
		return
	}

	active, found := k.GetActiveCeremony(ctx)
	switch status {
	case types.StatusResumeVoting:
		malformedReason := ""
		if !found || active.Type != string(types.CeremonyResume) {
			malformedReason = "legacy resume state has no active resume ceremony"
		} else {
			electorate, _, err := validateCeremonySnapshot(active)
			if err == nil {
				err = validateCeremonyTallies(active, electorate)
			}
			var proposal types.EmergencyResumeProposal
			proposalErr := unmarshalProposal(active.ProposalData, &proposal)
			if err != nil {
				malformedReason = "legacy resume electorate is invalid: " + err.Error()
			} else if proposalErr != nil ||
				proposal.HaltCeremonyId == "" ||
				proposal.Justification == "" ||
				!types.IsLowerSHA256(proposal.RecoveryManifestSha256) {
				malformedReason = "legacy resume lacks a valid recovery manifest or quarantine link"
			} else if activeHaltID := k.GetActiveHaltCeremonyId(ctx); activeHaltID == "" {
				malformedReason = "active resume has no persisted quarantine linkage"
			} else if proposal.HaltCeremonyId != activeHaltID {
				malformedReason = "legacy resume references a different quarantine incident"
			}
		}
		if malformedReason != "" {
			if found {
				active.Phase = string(types.PhaseFailed)
				active.FailureReason = malformedReason
				k.mustSetCeremony(ctx, active)
			}
			k.SetEmergencyStatus(ctx, types.StatusHalted)
			if found {
				k.HandleCeremonyFailure(ctx, active.Id)
			}
			status = types.StatusHalted
		}
	case types.StatusHalted:
		if found {
			failureReason := ""
			if active.Type != string(types.CeremonyRecoveryAuthorization) {
				failureReason = "inconsistent active ceremony found while transaction admission was quarantined"
			} else {
				electorate, _, err := validateCeremonySnapshot(active)
				if err == nil {
					err = validateCeremonyTallies(active, electorate)
				}
				var proposal types.EmergencyRecoveryAuthorizationProposal
				proposalErr := unmarshalProposal(
					active.ProposalData,
					&proposal,
				)
				switch {
				case err != nil:
					failureReason =
						"recovery authorization electorate is invalid: " +
							err.Error()
				case proposalErr != nil ||
					proposal.Id != active.Id ||
					proposal.HaltCeremonyId == "" ||
					proposal.HaltCeremonyId !=
						k.GetActiveHaltCeremonyId(ctx) ||
					proposal.SdkGovProposalId == 0 ||
					proposal.Generation == 0 ||
					proposal.AuthorizedSubmitter == "" ||
					(proposal.ActionType != "software_upgrade" &&
						proposal.ActionType != "cancel_upgrade" &&
						proposal.ActionType != "revoke") ||
					!types.IsLowerSHA256(proposal.ActionSha256) ||
					!types.IsLowerSHA256(
						proposal.UpgradePlanSha256,
					) ||
					!types.IsLowerSHA256(
						proposal.RecoveryManifestSha256,
					):
					failureReason =
						"recovery authorization proposal is malformed or incident-mismatched"
				case k.recoveryTargetVerifier == nil:
					failureReason =
						"recovery authorization target verifier is unavailable"
				default:
					if err := k.recoveryTargetVerifier.
						VerifyRecoveryAuthorizationTarget(
							ctx,
							proposal.SdkGovProposalId,
							proposal.ActionSha256,
							proposal.UpgradePlanSha256,
							proposal.ActionType,
						); err != nil {
						failureReason =
							"recovery authorization target changed: " +
								err.Error()
					}
				}
			}
			if failureReason != "" {
				active.Phase = string(types.PhaseFailed)
				active.FailureReason = failureReason
				k.mustSetCeremony(ctx, active)
				k.HandleCeremonyFailure(ctx, active.Id)
			}
		}
	}

	status = k.GetEmergencyStatus(ctx)
	if status != types.StatusHalted && status != types.StatusResumeVoting {
		return
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	currentHeight := uint64(1)
	if sdkCtx.BlockHeight() > 0 {
		currentHeight = uint64(sdkCtx.BlockHeight())
	}
	repaired := false
	ceremonyID := k.GetActiveHaltCeremonyId(ctx)
	if ceremonyID == "" {
		inferredID, _ := inferQuarantineLink(k.GetAllCeremonies(ctx))
		if inferredID == "" {
			inferredID = legacyGenesisQuarantineID
		}
		ceremonyID = inferredID
		k.SetActiveHaltCeremonyId(ctx, ceremonyID)
		repaired = true
	}
	if k.GetHaltStartBlock(ctx) == 0 {
		k.SetHaltStartBlock(ctx, currentHeight)
		repaired = true
	}
	if !repaired {
		return
	}

	k.AddAuditEntry(ctx, &types.EmergencyAuditEntry{
		Timestamp:   sdkCtx.BlockTime().Unix(),
		BlockNumber: currentHeight,
		Action:      string(types.AuditLegacyNormalized),
		Actor:       "system",
		CeremonyId:  ceremonyID,
		Details:     "legacy quarantine linkage repaired; transaction admission remains quarantined",
	})
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			"zerone.emergency.legacy_quarantine_repaired",
			sdk.NewAttribute("ceremony_id", ceremonyID),
			sdk.NewAttribute("status", string(status)),
			sdk.NewAttribute("halt_start_block", fmt.Sprintf("%d", k.GetHaltStartBlock(ctx))),
		),
	)
}

// MonitorRevertStatus migrates legacy in-place revert state to recoverable
// transaction quarantine. Height-only rollback is disabled, but the chain must
// retain a path to an evidence-bound resume after upgrading from an older
// binary that had already finalized such a ceremony.
func (k Keeper) MonitorRevertStatus(ctx context.Context) {
	status := k.GetEmergencyStatus(ctx)
	if status != types.StatusReverting && status != types.StatusRevertVoting {
		return
	}
	legacyActiveID := ""
	if active, found := k.GetActiveCeremony(ctx); found {
		legacyActiveID = active.Id
		active.Phase = string(types.PhaseFailed)
		active.FailureReason = "legacy revert voting retired: height-only rollback is disabled"
		k.mustSetCeremony(ctx, active)
	}

	target, found := k.GetRevertTarget(ctx)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	currentHeight := uint64(sdkCtx.BlockHeight())
	legacyIncidentID := legacyActiveID
	if found && target.CeremonyId != "" {
		legacyIncidentID = target.CeremonyId
	}
	if legacyIncidentID == "" {
		legacyIncidentID = legacyGenesisQuarantineID
	}
	quarantineID := k.GetActiveHaltCeremonyId(ctx)
	if quarantineID == "" {
		quarantineID, _ = inferQuarantineLink(k.GetAllCeremonies(ctx))
		if quarantineID == "" {
			quarantineID = legacyGenesisQuarantineID
		}
		k.SetActiveHaltCeremonyId(ctx, quarantineID)
	}
	if k.GetHaltStartBlock(ctx) == 0 {
		k.SetHaltStartBlock(ctx, currentHeight)
	}
	k.SetEmergencyStatus(ctx, types.StatusHalted)
	k.ClearRevertTarget(ctx)

	targetHeight := uint64(0)
	targetHash := ""
	if found {
		targetHeight = target.Height
		targetHash = target.BlockHash
	}
	k.Logger(ctx).Error("LEGACY REVERT STATE DETECTED — automatic or height-only rollback is disabled",
		"target_height", targetHeight,
		"target_hash", targetHash,
		"ceremony_id", legacyIncidentID,
		"quarantine_id", quarantineID,
		"current_height", currentHeight,
		"action", "remain quarantined; preserve evidence and prepare a hash-bound forward recovery or explicit social fork",
	)
	k.AddAuditEntry(ctx, &types.EmergencyAuditEntry{
		Timestamp:   sdkCtx.BlockTime().Unix(),
		BlockNumber: currentHeight,
		Action:      string(types.AuditRevertFailed),
		Actor:       "system",
		CeremonyId:  legacyIncidentID,
		Details:     "legacy revert state normalized to recoverable transaction quarantine; no rollback executed",
	})
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			"zerone.emergency.legacy_revert_normalized",
			sdk.NewAttribute("ceremony_id", legacyIncidentID),
			sdk.NewAttribute("quarantine_id", quarantineID),
			sdk.NewAttribute("status", string(types.StatusHalted)),
			sdk.NewAttribute("target_height", fmt.Sprintf("%d", targetHeight)),
			sdk.NewAttribute("action", "remain quarantined; require evidence-bound resume"),
		),
	)
}

func marshalProposal(proposal proto.Message) ([]byte, error) {
	return proto.Marshal(proposal)
}

func unmarshalProposal(data []byte, proposal proto.Message) error {
	return proto.Unmarshal(data, proposal)
}
