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
	params := k.GetParams(ctx)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	startBlock := uint64(sdkCtx.BlockHeight())

	ceremony := types.EmergencyCeremony{
		Id:                proposal.Id,
		Type:              string(types.CeremonyHalt),
		Phase:             string(types.PhasePrevote),
		StartBlock:        startBlock,
		PrevoteDeadline:   startBlock + params.HaltPrevoteBlocks,
		PrecommitDeadline: startBlock + params.HaltPrevoteBlocks + params.HaltPrecommitBlocks,
		TimeoutDeadline:   startBlock + params.HaltTimeoutBlocks,
		YesPrevoteStake:   "0",
		NoPrevoteStake:    "0",
		PrecommitStake:    "0",
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
	params := k.GetParams(ctx)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	startBlock := uint64(sdkCtx.BlockHeight())

	ceremony := types.EmergencyCeremony{
		Id:                proposal.Id,
		Type:              string(types.CeremonyRevert),
		Phase:             string(types.PhasePrevote),
		StartBlock:        startBlock,
		PrevoteDeadline:   startBlock + params.RevertPrevoteBlocks,
		PrecommitDeadline: startBlock + params.RevertPrevoteBlocks + params.RevertPrecommitBlocks,
		TimeoutDeadline:   startBlock + params.RevertTimeoutBlocks,
		YesPrevoteStake:   "0",
		NoPrevoteStake:    "0",
		PrecommitStake:    "0",
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
	params := k.GetParams(ctx)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	startBlock := uint64(sdkCtx.BlockHeight())

	ceremony := types.EmergencyCeremony{
		Id:                proposal.Id,
		Type:              string(types.CeremonyResume),
		Phase:             string(types.PhasePrevote),
		StartBlock:        startBlock,
		PrevoteDeadline:   startBlock + params.ResumePrevoteBlocks,
		PrecommitDeadline: startBlock + params.ResumePrevoteBlocks + params.ResumePrecommitBlocks,
		TimeoutDeadline:   startBlock + params.ResumeTimeoutBlocks,
		YesPrevoteStake:   "0",
		NoPrevoteStake:    "0",
		PrecommitStake:    "0",
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

	if !k.IsGuardian(ctx, vote.Voter) {
		return fmt.Errorf("%w: %s", types.ErrNotGuardian, vote.Voter)
	}

	if _, exists := ceremony.GetPrevote(vote.Voter); exists {
		return fmt.Errorf("%w: %s", types.ErrDuplicateVote, vote.Voter)
	}

	ceremony.SetPrevote(vote.Voter, vote)
	voterStake := k.GetGuardianEffectiveStake(ctx, vote.Voter)

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

	if !k.IsGuardian(ctx, precommit.Voter) {
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
	voterStake := k.GetGuardianEffectiveStake(ctx, precommit.Voter)

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
	currentBlock := uint64(sdkCtx.BlockHeight())
	params := k.GetParams(ctx)

	// Overall timeout.
	if currentBlock > ceremony.TimeoutDeadline {
		ceremony.Phase = string(types.PhaseFailed)
		ceremony.FailureReason = "ceremony timed out"
		_ = k.SetCeremony(ctx, ceremony)
		return false, nil
	}

	// Guardian stake check.
	guardianStake := k.GetGuardianStake(ctx)
	if guardianStake.Sign() == 0 {
		ceremony.Phase = string(types.PhaseFailed)
		ceremony.FailureReason = "no active guardians"
		_ = k.SetCeremony(ctx, ceremony)
		return false, nil
	}

	// Min guardian stake floor (skip if council active — H-5).
	if !k.isCouncilActive(ctx, params) {
		minStake, ok := new(big.Int).SetString(params.MinGuardianStake, 10)
		if ok && guardianStake.Cmp(minStake) < 0 {
			ceremony.Phase = string(types.PhaseFailed)
			ceremony.FailureReason = "insufficient total guardian stake"
			_ = k.SetCeremony(ctx, ceremony)
			return false, nil
		}
	}

	threshold := getQuorumThreshold(types.CeremonyType(ceremony.Type), params)

	switch ceremony.Phase {
	case string(types.PhasePrevote):
		return k.checkPrevotePhase(ctx, ceremony, currentBlock, guardianStake, threshold)
	case string(types.PhasePrecommit):
		return k.checkPrecommitPhase(ctx, ceremony, currentBlock, guardianStake, threshold, params)
	}

	return false, nil
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
		_ = k.SetCeremony(ctx, ceremony)

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
		_ = k.SetCeremony(ctx, ceremony)
		return false, nil
	}

	// Check prevote deadline.
	if currentBlock > ceremony.PrevoteDeadline {
		ceremony.Phase = string(types.PhaseFailed)
		ceremony.FailureReason = "prevote quorum not reached before deadline"
		_ = k.SetCeremony(ctx, ceremony)
		return false, nil
	}

	return false, nil
}

func (k Keeper) checkPrecommitPhase(ctx context.Context, ceremony *types.EmergencyCeremony, currentBlock uint64, guardianStake *big.Int, threshold uint64, params *types.Params) (bool, error) {
	precommitStake, _ := new(big.Int).SetString(ceremony.PrecommitStake, 10)
	if precommitStake == nil {
		precommitStake = new(big.Int)
	}

	precommitRatio := new(big.Int).Mul(precommitStake, big.NewInt(1000000))
	precommitRatio.Div(precommitRatio, guardianStake)

	distinctVoters := uint64(len(ceremony.Precommits))

	if precommitRatio.Uint64() >= threshold {
		if distinctVoters >= params.MinDistinctVoters {
			ceremony.Phase = string(types.PhaseFinalized)
			_ = k.SetCeremony(ctx, ceremony)
			return true, nil
		}
	}

	if currentBlock > ceremony.PrecommitDeadline {
		if precommitRatio.Uint64() >= threshold && distinctVoters < params.MinDistinctVoters {
			ceremony.Phase = string(types.PhaseFailed)
			ceremony.FailureReason = fmt.Sprintf("insufficient distinct voters: %d < %d", distinctVoters, params.MinDistinctVoters)
		} else {
			ceremony.Phase = string(types.PhaseFailed)
			ceremony.FailureReason = "precommit quorum not reached before deadline"
		}
		_ = k.SetCeremony(ctx, ceremony)
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

	switch types.CeremonyType(ceremony.Type) {
	case types.CeremonyHalt:
		k.SetEmergencyStatus(ctx, types.StatusHalted)
		k.SetActiveHaltCeremonyId(ctx, ceremony.Id)
		k.SetHaltStartBlock(ctx, uint64(sdkCtx.BlockHeight()))
		k.AddAuditEntry(ctx, &types.EmergencyAuditEntry{
			Timestamp:   sdkCtx.BlockTime().Unix(),
			BlockNumber: uint64(sdkCtx.BlockHeight()),
			Action:      string(types.AuditHaltExecuted),
			Actor:       "system",
			CeremonyId:  ceremony.Id,
			Details:     "halt ceremony finalized and executed",
		})
		sdkCtx.EventManager().EmitEvent(
			sdk.NewEvent("zerone.emergency.ceremony_finalized",
				sdk.NewAttribute("ceremony_id", ceremony.Id),
				sdk.NewAttribute("ceremony_type", string(types.CeremonyHalt)),
				sdk.NewAttribute("status", string(types.StatusHalted)),
				sdk.NewAttribute("block_height", fmt.Sprintf("%d", sdkCtx.BlockHeight())),
			),
		)

	case types.CeremonyRevert:
		k.SetEmergencyStatus(ctx, types.StatusReverting)
		var revertHeight uint64
		var revertHash string
		var proposal types.EmergencyRevertProposal
		if err := unmarshalProposal(ceremony.ProposalData, &proposal); err == nil {
			revertHeight = proposal.TargetBlockNumber
			revertHash = proposal.TargetBlockHash
		}
		k.SetRevertTarget(ctx, revertHeight, revertHash, ceremony.Id)

		sdkCtx.EventManager().EmitEvent(
			sdk.NewEvent(
				"zerone.emergency.revert_required",
				sdk.NewAttribute("ceremony_id", ceremony.Id),
				sdk.NewAttribute("target_height", fmt.Sprintf("%d", revertHeight)),
				sdk.NewAttribute("target_hash", revertHash),
				sdk.NewAttribute("action", "STOP nodes → rollback to target height → RESTART"),
			),
		)

		k.AddAuditEntry(ctx, &types.EmergencyAuditEntry{
			Timestamp:   sdkCtx.BlockTime().Unix(),
			BlockNumber: uint64(sdkCtx.BlockHeight()),
			Action:      string(types.AuditRevertExecuted),
			Actor:       "system",
			CeremonyId:  ceremony.Id,
			Details:     fmt.Sprintf("revert ceremony finalized — rollback to height %d required", revertHeight),
		})

	case types.CeremonyResume:
		k.SetEmergencyStatus(ctx, types.StatusNormal)
		k.SetActiveHaltCeremonyId(ctx, "")
		k.ClearHaltStartBlock(ctx)
		k.ClearRevertTarget(ctx)
		k.AddAuditEntry(ctx, &types.EmergencyAuditEntry{
			Timestamp:   sdkCtx.BlockTime().Unix(),
			BlockNumber: uint64(sdkCtx.BlockHeight()),
			Action:      string(types.AuditResumeExecuted),
			Actor:       "system",
			CeremonyId:  ceremony.Id,
			Details:     "resume ceremony finalized and executed",
		})
		sdkCtx.EventManager().EmitEvent(
			sdk.NewEvent("zerone.emergency.ceremony_finalized",
				sdk.NewAttribute("ceremony_id", ceremony.Id),
				sdk.NewAttribute("ceremony_type", string(types.CeremonyResume)),
				sdk.NewAttribute("status", string(types.StatusNormal)),
				sdk.NewAttribute("block_height", fmt.Sprintf("%d", sdkCtx.BlockHeight())),
			),
		)
	}
}

// HandleCeremonyFailure processes a failed ceremony's effects.
func (k Keeper) HandleCeremonyFailure(ctx context.Context, ceremonyId string) {
	ceremony, found := k.GetCeremony(ctx, ceremonyId)
	if !found || ceremony.Phase != string(types.PhaseFailed) {
		return
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	var auditAction types.AuditAction
	switch types.CeremonyType(ceremony.Type) {
	case types.CeremonyHalt:
		k.SetEmergencyStatus(ctx, types.StatusNormal)
		auditAction = types.AuditHaltFailed
	case types.CeremonyRevert:
		k.SetEmergencyStatus(ctx, types.StatusHalted)
		k.ClearRevertTarget(ctx)
		auditAction = types.AuditRevertFailed
	case types.CeremonyResume:
		k.SetEmergencyStatus(ctx, types.StatusHalted)
		auditAction = types.AuditResumeFailed
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
	default:
		return 800000
	}
}

// CheckHaltExpiry checks if the current halt has exceeded MaxHaltDurationBlocks.
func (k Keeper) CheckHaltExpiry(ctx context.Context) {
	status := k.GetEmergencyStatus(ctx)
	if status != types.StatusHalted {
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
	if currentHeight < haltStart+params.MaxHaltDurationBlocks {
		return
	}

	k.SetEmergencyStatus(ctx, types.StatusNormal)
	k.SetActiveHaltCeremonyId(ctx, "")
	k.ClearHaltStartBlock(ctx)
	k.AddAuditEntry(ctx, &types.EmergencyAuditEntry{
		Timestamp:   sdkCtx.BlockTime().Unix(),
		BlockNumber: currentHeight,
		Action:      string(types.AuditResumeExecuted),
		Actor:       "system",
		Details:     fmt.Sprintf("auto-resume: halt exceeded max duration of %d blocks (started at block %d)", params.MaxHaltDurationBlocks, haltStart),
	})
}

// MonitorRevertStatus logs ERROR-level alerts every block while reverting.
func (k Keeper) MonitorRevertStatus(ctx context.Context) {
	status := k.GetEmergencyStatus(ctx)
	if status != types.StatusReverting {
		return
	}

	target, found := k.GetRevertTarget(ctx)
	if !found {
		return
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	currentHeight := uint64(sdkCtx.BlockHeight())
	k.Logger(ctx).Error("REVERT REQUIRED — chain is in reverting state",
		"target_height", target.Height,
		"target_hash", target.BlockHash,
		"ceremony_id", target.CeremonyId,
		"current_height", currentHeight,
		"action", "STOP nodes → run 'zeroned rollback-state --to-height N' → RESTART",
	)

	haltStart := k.GetHaltStartBlock(ctx)
	if haltStart == 0 {
		return
	}
	params := k.GetParams(ctx)
	if params.MaxHaltDurationBlocks > 0 && currentHeight >= haltStart+params.MaxHaltDurationBlocks {
		k.Logger(ctx).Error("REVERT EXPIRED — auto-resuming without rollback",
			"target_height", target.Height,
			"halt_started", haltStart,
			"max_duration", params.MaxHaltDurationBlocks,
		)
		k.SetEmergencyStatus(ctx, types.StatusNormal)
		k.SetActiveHaltCeremonyId(ctx, "")
		k.ClearHaltStartBlock(ctx)
		k.ClearRevertTarget(ctx)
		k.AddAuditEntry(ctx, &types.EmergencyAuditEntry{
			Timestamp:   sdkCtx.BlockTime().Unix(),
			BlockNumber: currentHeight,
			Action:      string(types.AuditResumeExecuted),
			Actor:       "system",
			CeremonyId:  target.CeremonyId,
			Details:     fmt.Sprintf("auto-resume: revert to height %d expired after %d blocks without operator action", target.Height, params.MaxHaltDurationBlocks),
		})
	}
}

func marshalProposal(proposal proto.Message) ([]byte, error) {
	return proto.Marshal(proposal)
}

func unmarshalProposal(data []byte, proposal proto.Message) error {
	return proto.Unmarshal(data, proposal)
}
