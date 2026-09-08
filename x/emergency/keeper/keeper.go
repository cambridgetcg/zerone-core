package keeper

import (
	"context"
	"fmt"
	"math/big"

	"cosmossdk.io/core/store"
	"cosmossdk.io/log"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/protobuf/proto"

	"github.com/zerone-chain/zerone/x/emergency/types"
)

// Keeper manages the emergency module's state.
type Keeper struct {
	storeService           store.KVStoreService
	cdc                    codec.BinaryCodec
	authority              string
	stakingKeeper          types.StakingKeeper
	recoveryTargetVerifier types.RecoveryAuthorizationTargetVerifier
}

// SetRecoveryAuthorizationTargetVerifier wires the app-owned SDK governance
// and x/upgrade state verifier after all keepers have been constructed.
func (k *Keeper) SetRecoveryAuthorizationTargetVerifier(
	verifier types.RecoveryAuthorizationTargetVerifier,
) {
	k.recoveryTargetVerifier = verifier
}

// GuardianReadiness is the queryable precondition for opening and finalizing
// an emergency quarantine ceremony. It deliberately describes the custom
// Zerone Guardian electorate, not CometBFT consensus voting power.
type GuardianReadiness struct {
	EligibleGuardians uint64
	EffectiveStake    *big.Int
	Ready             bool
	Reason            string
}

// NewKeeper creates a new emergency module Keeper.
func NewKeeper(
	storeService store.KVStoreService,
	cdc codec.BinaryCodec,
	authority string,
	stakingKeeper types.StakingKeeper,
) Keeper {
	return Keeper{
		storeService:  storeService,
		cdc:           cdc,
		authority:     authority,
		stakingKeeper: stakingKeeper,
	}
}

// prefixEndBytes returns the end key for prefix iteration (exclusive).
func prefixEndBytes(prefix []byte) []byte {
	if len(prefix) == 0 {
		return nil
	}
	end := make([]byte, len(prefix))
	copy(end, prefix)
	for i := len(end) - 1; i >= 0; i-- {
		end[i]++
		if end[i] != 0 {
			return end
		}
	}
	return nil
}

// Logger returns a module-scoped logger.
func (k Keeper) Logger(ctx context.Context) log.Logger {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	return sdkCtx.Logger().With("module", "x/"+types.ModuleName)
}

// GetAuthority returns the module authority address.
func (k Keeper) GetAuthority() string {
	return k.authority
}

// IsGuardian checks if an address is a Guardian-tier validator or genesis council member.
func (k Keeper) IsGuardian(ctx context.Context, operatorAddr string) bool {
	val, found := k.stakingKeeper.GetValidator(ctx, operatorAddr)
	if found && val.Tier == types.TierGuardian && val.IsActive {
		return true
	}
	// H-5: Check genesis emergency council.
	params := k.GetParams(ctx)
	if k.isCouncilActive(ctx, params) {
		return k.isCouncilMember(operatorAddr, params)
	}
	return false
}

// GetGuardianStake returns the total effective stake of all active Guardians.
func (k Keeper) GetGuardianStake(ctx context.Context) *big.Int {
	total := new(big.Int)
	eligible := make(map[string]struct{})
	guardians, err := k.stakingKeeper.GetGuardianValidators(ctx)
	if err != nil {
		return total
	}
	for _, v := range guardians {
		if v.IsActive {
			stake, ok := new(big.Int).SetString(v.TotalStake, 10)
			if ok && stake.Sign() > 0 {
				if _, duplicate := eligible[v.Address]; duplicate {
					continue
				}
				eligible[v.Address] = struct{}{}
				total.Add(total, stake)
			}
		}
	}
	// H-5: Add council virtual stake during bootstrap.
	params := k.GetParams(ctx)
	if k.isCouncilActive(ctx, params) && len(params.GenesisCouncil) > 0 {
		virtualPerMember, ok := new(big.Int).SetString(params.CouncilVirtualStake, 10)
		if ok && virtualPerMember.Sign() > 0 {
			for _, member := range params.GenesisCouncil {
				if _, duplicate := eligible[member]; duplicate {
					continue
				}
				eligible[member] = struct{}{}
				total.Add(total, virtualPerMember)
			}
		}
	}
	return total
}

// GetGuardianEffectiveStake returns a single guardian's effective stake.
func (k Keeper) GetGuardianEffectiveStake(ctx context.Context, operatorAddr string) *big.Int {
	val, found := k.stakingKeeper.GetValidator(ctx, operatorAddr)
	if found && val.Tier == types.TierGuardian && val.IsActive {
		stake, ok := new(big.Int).SetString(val.TotalStake, 10)
		if ok {
			return stake
		}
	}
	// H-5: Council member virtual stake.
	params := k.GetParams(ctx)
	if k.isCouncilActive(ctx, params) && k.isCouncilMember(operatorAddr, params) {
		virtualStake, ok := new(big.Int).SetString(params.CouncilVirtualStake, 10)
		if ok {
			return virtualStake
		}
	}
	return new(big.Int)
}

// GetGuardianReadiness reports whether the current custom Guardian/council
// electorate can satisfy the module's structural preconditions. It is
// observability, not a promise that quorum will vote.
func (k Keeper) GetGuardianReadiness(ctx context.Context) GuardianReadiness {
	params := k.GetParams(ctx)
	eligible := make(map[string]struct{})
	total := new(big.Int)

	guardians, err := k.stakingKeeper.GetGuardianValidators(ctx)
	if err != nil {
		return GuardianReadiness{
			EffectiveStake: total,
			Reason:         "custom guardian electorate is unreadable",
		}
	}
	for _, guardian := range guardians {
		if !guardian.IsActive || guardian.Tier != types.TierGuardian {
			continue
		}
		stake, ok := new(big.Int).SetString(guardian.TotalStake, 10)
		if !ok || stake.Sign() <= 0 {
			continue
		}
		if _, duplicate := eligible[guardian.Address]; duplicate {
			continue
		}
		eligible[guardian.Address] = struct{}{}
		total.Add(total, stake)
	}

	councilActive := k.isCouncilActive(ctx, params)
	if councilActive {
		virtualStake, ok := new(big.Int).SetString(params.CouncilVirtualStake, 10)
		if !ok || virtualStake.Sign() <= 0 {
			return GuardianReadiness{
				EligibleGuardians: uint64(len(eligible)),
				EffectiveStake:    total,
				Reason:            "genesis council virtual stake is invalid",
			}
		}
		for _, member := range params.GenesisCouncil {
			if _, duplicate := eligible[member]; duplicate {
				continue
			}
			eligible[member] = struct{}{}
			total.Add(total, virtualStake)
		}
	}

	readiness := GuardianReadiness{
		EligibleGuardians: uint64(len(eligible)),
		EffectiveStake:    total,
	}
	if readiness.EligibleGuardians > types.MaxEmergencyElectorateSize {
		readiness.Reason = "eligible custom guardians exceed the emergency electorate consensus maximum"
		return readiness
	}
	if readiness.EligibleGuardians < params.MinDistinctVoters {
		readiness.Reason = "eligible custom guardians are below min_distinct_voters"
		return readiness
	}
	if total.Sign() == 0 {
		readiness.Reason = "effective custom guardian stake is zero"
		return readiness
	}
	if !councilActive {
		minStake, ok := new(big.Int).SetString(params.MinGuardianStake, 10)
		if !ok || total.Cmp(minStake) < 0 {
			readiness.Reason = "effective custom guardian stake is below min_guardian_stake"
			return readiness
		}
	}
	readiness.Ready = true
	readiness.Reason = "custom guardian electorate satisfies structural preconditions"
	return readiness
}

// isCouncilMember checks if an address is in the genesis emergency council.
func (k Keeper) isCouncilMember(operatorAddr string, params *types.Params) bool {
	for _, member := range params.GenesisCouncil {
		if member == operatorAddr {
			return true
		}
	}
	return false
}

// isCouncilActive checks if the genesis council is still active.
func (k Keeper) isCouncilActive(ctx context.Context, params *types.Params) bool {
	if params.CouncilExpiryBlock == 0 || len(params.GenesisCouncil) == 0 {
		return false
	}
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	return uint64(sdkCtx.BlockHeight()) < params.CouncilExpiryBlock
}

// --- Genesis ---

const legacyGenesisQuarantineID = "legacy-genesis-quarantine"

// InitGenesis initializes the module's state from genesis.
func (k Keeper) InitGenesis(ctx context.Context, genState *types.GenesisState) {
	if err := genState.Validate(); err != nil {
		panic("invalid emergency genesis: " + err.Error())
	}
	if genState.Params != nil {
		normalizedParams := types.NormalizeLegacyParams(genState.Params)
		k.SetParams(ctx, normalizedParams)
	}
	k.ResetEpochCounters(ctx)
	for _, counter := range genState.GuardianProposalCounts {
		k.SetGuardianProposalCount(ctx, counter.Guardian, counter.Count)
	}
	k.SetEpochProposalCount(ctx, genState.EpochProposalCount)
	k.SetLastProposalBlock(ctx, genState.LastProposalBlock)
	k.setActiveCeremonyID(ctx, "")
	status := types.EmergencyStatus(genState.Status)
	normalizedCeremonies := make([]*types.EmergencyCeremony, 0, len(genState.Ceremonies))
	for _, ceremony := range genState.Ceremonies {
		normalized := proto.Clone(ceremony).(*types.EmergencyCeremony)
		if normalized.Phase == string(types.PhasePrevote) ||
			normalized.Phase == string(types.PhasePrecommit) {
			switch {
			case status == types.StatusRevertVoting ||
				status == types.StatusReverting ||
				normalized.Type == string(types.CeremonyRevert):
				normalized.Phase = string(types.PhaseFailed)
				normalized.FailureReason = "legacy revert voting retired: height-only rollback is disabled"
				status = types.StatusHalted
			case normalized.ElectorateSnapshotVersion != types.ElectorateSnapshotVersionV1:
				normalized.Phase = string(types.PhaseFailed)
				normalized.FailureReason = "legacy active ceremony retired: immutable electorate snapshot is absent"
				if normalized.Type == string(types.CeremonyHalt) {
					status = types.StatusNormal
				} else {
					status = types.StatusHalted
				}
			case normalized.Type == string(types.CeremonyResume):
				var proposal types.EmergencyResumeProposal
				if err := unmarshalProposal(normalized.ProposalData, &proposal); err != nil ||
					proposal.HaltCeremonyId == "" ||
					proposal.Justification == "" ||
					!types.IsLowerSHA256(proposal.RecoveryManifestSha256) {
					normalized.Phase = string(types.PhaseFailed)
					normalized.FailureReason = "legacy resume retired: recovery manifest or quarantine linkage is invalid"
					status = types.StatusHalted
				}
			}
		}
		normalizedCeremonies = append(normalizedCeremonies, normalized)
		if err := k.SetCeremony(ctx, normalized); err != nil {
			panic("failed to init genesis ceremony: " + err.Error())
		}
	}
	resumeAttempts, err := deriveResumeAttempts(normalizedCeremonies)
	if err != nil {
		panic("failed to derive emergency resume attempt index: " + err.Error())
	}
	if err := k.replaceResumeAttemptIndex(ctx, resumeAttempts); err != nil {
		panic("failed to rebuild emergency resume attempt index: " + err.Error())
	}
	for _, entry := range genState.AuditLog {
		if entry == nil {
			continue
		}
		k.AddAuditEntry(ctx, entry)
	}

	// Height-only revert execution is disabled. Normalize imported legacy
	// revert states into the same transaction quarantine from which an
	// evidence-bound resume can be proposed.
	if status == types.StatusRevertVoting || status == types.StatusReverting {
		status = types.StatusHalted
	}
	// A partially exported resume vote without its active ceremony cannot
	// progress. Remaining quarantined is safer than reopening admission.
	if status == types.StatusResumeVoting {
		active, found := k.GetActiveCeremony(ctx)
		if !found || active.Type != string(types.CeremonyResume) {
			status = types.StatusHalted
		}
	}
	k.SetEmergencyStatus(ctx, status)
	if genState.QuarantineReleaseBlock != 0 {
		k.SetQuarantineReleaseBlock(ctx, genState.QuarantineReleaseBlock)
	} else {
		k.ClearQuarantineReleaseBlock(ctx)
	}

	if status == types.StatusHalted || status == types.StatusResumeVoting {
		activeHaltID := genState.ActiveHaltCeremonyId
		inferredHaltStart := uint64(0)
		if activeHaltID == "" {
			// Backward-compatible migration for exports created before
			// active quarantine linkage was included in GenesisState.
			activeHaltID, inferredHaltStart = inferQuarantineLink(normalizedCeremonies)
			if activeHaltID == "" {
				activeHaltID = legacyGenesisQuarantineID
			}
		}
		k.SetActiveHaltCeremonyId(ctx, activeHaltID)
		haltStart := genState.HaltStartBlock
		if haltStart == 0 {
			haltStart = inferredHaltStart
		}
		if haltStart == 0 {
			sdkCtx := sdk.UnwrapSDKContext(ctx)
			if sdkCtx.BlockHeight() > 0 {
				haltStart = uint64(sdkCtx.BlockHeight())
			} else {
				haltStart = 1
			}
		}
		k.SetHaltStartBlock(ctx, haltStart)
		if genState.LastHaltEscalationBlock != 0 {
			k.SetLastHaltEscalationBlock(ctx, genState.LastHaltEscalationBlock)
		}
	} else {
		k.SetActiveHaltCeremonyId(ctx, "")
		k.ClearHaltStartBlock(ctx)
	}
	if genState.RecoveryAuthorization != nil {
		if err := k.setRecoveryAuthorization(
			ctx,
			proto.Clone(
				genState.RecoveryAuthorization,
			).(*types.EmergencyRecoveryAuthorization),
		); err != nil {
			panic("failed to init recovery authorization: " + err.Error())
		}
	} else if err := k.ClearRecoveryAuthorization(ctx); err != nil {
		panic("failed to clear recovery authorization at genesis: " + err.Error())
	}
}

// inferQuarantineLink recovers the best deterministic incident link available
// in an old export from finalized halt state only. An active resume proposal
// cannot authenticate its own incident link.
func inferQuarantineLink(ceremonies []*types.EmergencyCeremony) (string, uint64) {
	var selected *types.EmergencyCeremony
	for _, ceremony := range ceremonies {
		if ceremony == nil ||
			ceremony.Type != string(types.CeremonyHalt) ||
			ceremony.Phase != string(types.PhaseFinalized) {
			continue
		}
		if selected == nil ||
			ceremony.StartBlock > selected.StartBlock ||
			(ceremony.StartBlock == selected.StartBlock && ceremony.Id > selected.Id) {
			selected = ceremony
		}
	}
	if selected == nil {
		return "", 0
	}
	return selected.Id, selected.StartBlock
}

// ExportGenesis exports the module's state.
func (k Keeper) ExportGenesis(ctx context.Context) *types.GenesisState {
	ceremonies := k.GetAllCeremonies(ctx)
	auditLog := k.GetAuditLog(ctx)
	params := proto.Clone(k.GetParams(ctx)).(*types.Params)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	if sdkCtx.BlockHeight() >= 0 &&
		params.CouncilExpiryBlock != 0 &&
		uint64(sdkCtx.BlockHeight()) >= params.CouncilExpiryBlock {
		// Council expiry is irreversible in every exported continuation.
		// Retaining an expired absolute height would revive bootstrap
		// authority if an operator later chose a zero-height genesis.
		params.GenesisCouncil = nil
		params.CouncilExpiryBlock = 0
	}
	recoveryAuthorization, found, err := k.GetRecoveryAuthorization(ctx)
	if err != nil {
		panic("failed to export recovery authorization: " + err.Error())
	}
	if !found {
		recoveryAuthorization = nil
	}
	return &types.GenesisState{
		Params:                  params,
		Status:                  string(k.GetEmergencyStatus(ctx)),
		Ceremonies:              ceremonies,
		AuditLog:                auditLog,
		ActiveHaltCeremonyId:    k.GetActiveHaltCeremonyId(ctx),
		HaltStartBlock:          k.GetHaltStartBlock(ctx),
		GuardianProposalCounts:  k.GetAllGuardianProposalCounts(ctx),
		EpochProposalCount:      k.GetEpochProposalCount(ctx),
		LastProposalBlock:       k.GetLastProposalBlock(ctx),
		LastHaltEscalationBlock: k.GetLastHaltEscalationBlock(ctx),
		QuarantineReleaseBlock:  k.GetQuarantineReleaseBlock(ctx),
		RecoveryAuthorization:   recoveryAuthorization,
	}
}

// ExportGenesisForZeroHeight returns emergency state safe to import at height
// zero. Live incident state carries absolute deadlines and is therefore
// refused rather than guessed or silently reopened.
func (k Keeper) ExportGenesisForZeroHeight(ctx context.Context) (*types.GenesisState, error) {
	genState := k.ExportGenesis(ctx)
	if status := types.EmergencyStatus(genState.Status); status != types.StatusNormal {
		return nil, fmt.Errorf(
			"zero-height export requires normal emergency status, got %s; close or socially coordinate the incident first",
			status,
		)
	}
	if active, found := k.GetActiveCeremony(ctx); found {
		return nil, fmt.Errorf(
			"zero-height export refuses active emergency ceremony %q (%s/%s)",
			active.Id,
			active.Type,
			active.Phase,
		)
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	if sdkCtx.BlockHeight() < 0 {
		return nil, fmt.Errorf("zero-height export cannot rebase negative block height %d", sdkCtx.BlockHeight())
	}
	currentHeight := uint64(sdkCtx.BlockHeight())
	if genState.QuarantineReleaseBlock != 0 {
		if genState.QuarantineReleaseBlock >
			types.MaxSDKBlockHeight-types.PostResumeCancellationGraceBlocks {
			return nil, fmt.Errorf(
				"zero-height export found an unrepresentable quarantine release grace window at block %d",
				genState.QuarantineReleaseBlock,
			)
		}
		graceEnds := genState.QuarantineReleaseBlock + types.PostResumeCancellationGraceBlocks
		if currentHeight <= graceEnds {
			return nil, fmt.Errorf(
				"zero-height export would erase the post-resume cancellation grace through block %d",
				graceEnds,
			)
		}
		// An expired absolute grace marker must not become a fresh quarantine
		// after rebasing the exported chain to height zero.
		genState.QuarantineReleaseBlock = 0
	}
	if genState.LastProposalBlock > currentHeight {
		return nil, fmt.Errorf(
			"last emergency proposal block %d exceeds export height %d",
			genState.LastProposalBlock,
			currentHeight,
		)
	}
	if genState.LastProposalBlock != 0 {
		elapsed := currentHeight - genState.LastProposalBlock
		if elapsed < genState.Params.CooldownBlocks {
			return nil, fmt.Errorf(
				"zero-height export would erase an active emergency proposal cooldown (%d blocks remain)",
				genState.Params.CooldownBlocks-elapsed,
			)
		}
		// The cooldown has fully elapsed. Zero is the canonical rebased form;
		// epoch counters remain preserved (and therefore conservative).
		genState.LastProposalBlock = 0
	}

	if genState.Params.CouncilExpiryBlock > currentHeight {
		genState.Params.CouncilExpiryBlock -= currentHeight
	}
	if err := genState.Validate(); err != nil {
		return nil, fmt.Errorf("rebased zero-height emergency genesis is invalid: %w", err)
	}
	return genState, nil
}
