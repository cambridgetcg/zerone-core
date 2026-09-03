package types

import (
	"fmt"
	"math/big"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/protobuf/proto"
)

// DefaultParams returns the default emergency module parameters.
//
// These values express commitment 10 (forward-only audit) made into a
// ceremony rather than a privileged switch. See doc.go for the
// contract; the values below are the chain's pre-committed posture
// about its own emergency authority.
func DefaultParams() Params {
	return Params{
		// Transaction quarantine requires 75%; reopening requires 80%.
		// RevertQuorum remains in state only for wire/genesis compatibility:
		// arbitrary height-only reverts are disabled.
		HaltQuorum:   750000, // 75% — supermajority to restrict transactions
		RevertQuorum: 800000, // legacy compatibility; revert messages fail closed
		ResumeQuorum: 800000, // 80% — evidence-bound reopening

		// Vote phase blocks: prevote and precommit are SHORT for halt
		// (11 blocks ≈ 28s) because containment is time-sensitive.
		// Legacy revert and active resume fields remain longer.
		HaltPrevoteBlocks:     11,
		HaltPrecommitBlocks:   11,
		HaltTimeoutBlocks:     44,
		RevertPrevoteBlocks:   22,
		RevertPrecommitBlocks: 22,
		RevertTimeoutBlocks:   111,
		ResumePrevoteBlocks:   22,
		ResumePrecommitBlocks: 22,
		ResumeTimeoutBlocks:   111,

		// Per-guardian and per-epoch caps: any single guardian can
		// open at most 1 proposal per epoch; the whole guardian set
		// can open at most 3. This rate-limits emergency machinery
		// from being weaponised as denial-of-service.
		MaxProposalsPerEpoch:            3,
		MaxProposalsPerGuardianPerEpoch: 1,
		CooldownBlocks:                  111,

		// Stake floor: total guardian stake must be at least 111,111
		// ZRN before halt proposals are accepted. This is the
		// "plurality is structural" floor — without enough guardians
		// committed, no single signer can declare an emergency.
		// MinDistinctVoters of 4 ensures at least four different
		// addresses participate in any quorum tally.
		MinGuardianStake:  "111111000000", // 111,111 ZRN — plurality requires committed stake
		MinDistinctVoters: 4,              // 4 distinct addresses minimum on any tally
		MaxRevertDepth:    111111,         // legacy compatibility; arbitrary revert is disabled

		// Epoch and quarantine escalation deadline: 1 day at 2521ms
		// blocks. Crossing MaxHaltDurationBlocks alerts operators but
		// NEVER reopens transaction admission. Recovery remains an
		// affirmative, evidence-bound guardian decision.
		EpochBlocks:           34272, // ~1 day
		GenesisCouncil:        []string{},
		CouncilExpiryBlock:    0,
		CouncilVirtualStake:   "11111000000", // 11,111 ZRN
		MaxHaltDurationBlocks: 34272,         // ~1 day escalation deadline; never auto-resume
	}
}

// DefaultGenesis returns the default genesis state.
func DefaultGenesis() *GenesisState {
	p := DefaultParams()
	return &GenesisState{
		Params: &p,
		Status: string(StatusNormal),
	}
}

// Validate validates the genesis state.
func (gs *GenesisState) Validate() error {
	if gs.Params == nil {
		return fmt.Errorf("params cannot be nil")
	}
	normalizedParams := NormalizeLegacyParams(gs.Params)
	if err := normalizedParams.Validate(); err != nil {
		return err
	}
	status := EmergencyStatus(gs.Status)
	switch status {
	case StatusNormal, StatusHaltVoting, StatusHalted,
		StatusRevertVoting, StatusReverting, StatusResumeVoting:
	default:
		return fmt.Errorf("invalid emergency status %q", gs.Status)
	}
	if gs.ActiveHaltCeremonyId != "" {
		if strings.TrimSpace(gs.ActiveHaltCeremonyId) != gs.ActiveHaltCeremonyId ||
			len(gs.ActiveHaltCeremonyId) > 512 {
			return fmt.Errorf("active_halt_ceremony_id must be trimmed and at most 512 bytes")
		}
	}
	if status == StatusNormal &&
		(gs.ActiveHaltCeremonyId != "" ||
			gs.HaltStartBlock != 0 ||
			gs.LastHaltEscalationBlock != 0) {
		return fmt.Errorf("normal status cannot carry active quarantine linkage")
	}
	if gs.QuarantineReleaseBlock != 0 {
		if status != StatusNormal {
			return fmt.Errorf(
				"quarantine_release_block requires normal status, got %s",
				status,
			)
		}
		if gs.QuarantineReleaseBlock > MaxSDKBlockHeight-PostResumeCancellationGraceBlocks {
			return fmt.Errorf(
				"quarantine_release_block cannot represent the full post-resume grace window",
			)
		}
	}
	if (gs.ActiveHaltCeremonyId == "") != (gs.HaltStartBlock == 0) {
		return fmt.Errorf("active quarantine id and halt start block must be both present or both absent")
	}
	if gs.EpochProposalCount > normalizedParams.MaxProposalsPerEpoch {
		return fmt.Errorf(
			"epoch proposal count %d exceeds configured maximum %d",
			gs.EpochProposalCount,
			normalizedParams.MaxProposalsPerEpoch,
		)
	}
	var proposalCountTotal uint64
	previousGuardian := ""
	for i, counter := range gs.GuardianProposalCounts {
		if counter == nil {
			return fmt.Errorf("guardian proposal count at index %d cannot be nil", i)
		}
		if counter.Guardian == "" ||
			strings.TrimSpace(counter.Guardian) != counter.Guardian ||
			len(counter.Guardian) > 512 ||
			(i > 0 && counter.Guardian <= previousGuardian) {
			return fmt.Errorf("guardian proposal counts must have unique, trimmed addresses in strict lexical order")
		}
		if counter.Count == 0 ||
			counter.Count > normalizedParams.MaxProposalsPerGuardianPerEpoch {
			return fmt.Errorf(
				"guardian %q proposal count %d is outside configured range 1..%d",
				counter.Guardian,
				counter.Count,
				normalizedParams.MaxProposalsPerGuardianPerEpoch,
			)
		}
		if proposalCountTotal > ^uint64(0)-counter.Count {
			return fmt.Errorf("guardian proposal count total overflows uint64")
		}
		proposalCountTotal += counter.Count
		previousGuardian = counter.Guardian
	}
	if proposalCountTotal != gs.EpochProposalCount {
		return fmt.Errorf(
			"guardian proposal count total %d does not match epoch proposal count %d",
			proposalCountTotal,
			gs.EpochProposalCount,
		)
	}
	if gs.LastHaltEscalationBlock != 0 {
		if status != StatusHalted && status != StatusResumeVoting &&
			status != StatusRevertVoting && status != StatusReverting {
			return fmt.Errorf("halt escalation marker requires an active transaction quarantine")
		}
		if gs.HaltStartBlock == 0 ||
			normalizedParams.MaxHaltDurationBlocks > ^uint64(0)-gs.HaltStartBlock ||
			gs.LastHaltEscalationBlock <
				gs.HaltStartBlock+normalizedParams.MaxHaltDurationBlocks {
			return fmt.Errorf("halt escalation marker precedes the first quarantine deadline")
		}
	}
	if err := ValidateRecoveryAuthorization(
		gs.RecoveryAuthorization,
	); err != nil {
		return err
	}
	if gs.RecoveryAuthorization != nil {
		if status != StatusHalted && status != StatusResumeVoting {
			return fmt.Errorf(
				"recovery authorization requires an active transaction quarantine",
			)
		}
		if gs.ActiveHaltCeremonyId == "" ||
			gs.RecoveryAuthorization.HaltCeremonyId !=
				gs.ActiveHaltCeremonyId {
			return fmt.Errorf(
				"recovery authorization must match active_halt_ceremony_id",
			)
		}
	}

	ceremonies := make(map[string]*EmergencyCeremony, len(gs.Ceremonies))
	var active *EmergencyCeremony
	for i, ceremony := range gs.Ceremonies {
		if ceremony == nil {
			return fmt.Errorf("ceremony at index %d cannot be nil", i)
		}
		if err := validateGenesisCeremony(ceremony); err != nil {
			return fmt.Errorf("ceremony %d: %w", i, err)
		}
		if _, duplicate := ceremonies[ceremony.Id]; duplicate {
			return fmt.Errorf("duplicate ceremony id %q", ceremony.Id)
		}
		ceremonies[ceremony.Id] = ceremony
		if isNonterminalCeremony(ceremony) {
			if active != nil {
				return fmt.Errorf("multiple active emergency ceremonies %q and %q", active.Id, ceremony.Id)
			}
			active = ceremony
		}
	}

	switch status {
	case StatusNormal:
		if active != nil {
			return fmt.Errorf("normal status cannot contain active %s ceremony %q", active.Type, active.Id)
		}
	case StatusHaltVoting:
		if active == nil || active.Type != string(CeremonyHalt) {
			return fmt.Errorf("halt_voting status requires exactly one active halt ceremony")
		}
		if gs.ActiveHaltCeremonyId != "" {
			return fmt.Errorf("halt_voting status cannot carry finalized quarantine linkage")
		}
	case StatusHalted:
		if active != nil &&
			active.Type != string(CeremonyRecoveryAuthorization) {
			return fmt.Errorf("halted status cannot contain active %s ceremony %q", active.Type, active.Id)
		}
	case StatusResumeVoting:
		if active == nil || active.Type != string(CeremonyResume) {
			return fmt.Errorf("resume_voting status requires exactly one active resume ceremony")
		}
	case StatusRevertVoting, StatusReverting:
		// These legacy states are accepted only so InitGenesis can
		// terminalize their optional active revert ceremony and preserve
		// transaction quarantine without executing a rollback.
		if active != nil && active.Type != string(CeremonyRevert) {
			return fmt.Errorf("%s status cannot contain active %s ceremony %q", status, active.Type, active.Id)
		}
	}

	if gs.ActiveHaltCeremonyId != "" &&
		gs.ActiveHaltCeremonyId != legacyGenesisQuarantineMarker {
		halt, found := ceremonies[gs.ActiveHaltCeremonyId]
		if !found ||
			halt.Type != string(CeremonyHalt) ||
			halt.Phase != string(PhaseFinalized) {
			return fmt.Errorf(
				"active_halt_ceremony_id %q must reference a finalized halt ceremony",
				gs.ActiveHaltCeremonyId,
			)
		}
	}
	if active != nil && active.Type == string(CeremonyResume) {
		if active.ElectorateSnapshotVersion == ElectorateSnapshotVersionV1 &&
			gs.ActiveHaltCeremonyId == "" {
			return fmt.Errorf(
				"active snapshotted resume ceremony %q requires explicit quarantine linkage",
				active.Id,
			)
		}
		var proposal EmergencyResumeProposal
		if err := proto.Unmarshal(active.ProposalData, &proposal); err != nil {
			return fmt.Errorf("active resume ceremony %q has invalid proposal data: %w", active.Id, err)
		}
		if gs.ActiveHaltCeremonyId != "" &&
			proposal.HaltCeremonyId != gs.ActiveHaltCeremonyId {
			return fmt.Errorf(
				"active resume ceremony %q references halt %q, want %q",
				active.Id,
				proposal.HaltCeremonyId,
				gs.ActiveHaltCeremonyId,
			)
		}
	}
	if gs.RecoveryAuthorization != nil {
		authorizationCeremony, found := ceremonies[gs.RecoveryAuthorization.AuthorizationCeremonyId]
		if !found ||
			authorizationCeremony.Type !=
				string(CeremonyRecoveryAuthorization) ||
			authorizationCeremony.Phase != string(PhaseFinalized) {
			return fmt.Errorf(
				"recovery authorization must reference a finalized recovery_authorization ceremony",
			)
		}
		var proposal EmergencyRecoveryAuthorizationProposal
		if err := proto.Unmarshal(
			authorizationCeremony.ProposalData,
			&proposal,
		); err != nil {
			return fmt.Errorf(
				"recovery authorization ceremony has invalid proposal data: %w",
				err,
			)
		}
		if proposal.HaltCeremonyId !=
			gs.RecoveryAuthorization.HaltCeremonyId ||
			proposal.SdkGovProposalId !=
				gs.RecoveryAuthorization.SdkGovProposalId ||
			proposal.ActionSha256 !=
				gs.RecoveryAuthorization.ActionSha256 ||
			proposal.UpgradePlanSha256 !=
				gs.RecoveryAuthorization.UpgradePlanSha256 ||
			proposal.RecoveryManifestSha256 !=
				gs.RecoveryAuthorization.RecoveryManifestSha256 ||
			proposal.AuthorizedSubmitter !=
				gs.RecoveryAuthorization.AuthorizedSubmitter ||
			proposal.ActionType !=
				gs.RecoveryAuthorization.ActionType ||
			proposal.Generation !=
				gs.RecoveryAuthorization.Generation {
			return fmt.Errorf(
				"recovery authorization does not match its finalized ceremony",
			)
		}
	}
	return nil
}

const legacyGenesisQuarantineMarker = "legacy-genesis-quarantine"

func isNonterminalCeremony(ceremony *EmergencyCeremony) bool {
	return ceremony.Phase == string(PhasePrevote) || ceremony.Phase == string(PhasePrecommit)
}

func validateGenesisCeremony(ceremony *EmergencyCeremony) error {
	if ceremony.Id == "" ||
		strings.TrimSpace(ceremony.Id) != ceremony.Id ||
		len(ceremony.Id) > 512 {
		return fmt.Errorf("id must be non-empty, trimmed, and at most 512 bytes")
	}
	ceremonyType := CeremonyType(ceremony.Type)
	switch ceremonyType {
	case CeremonyHalt, CeremonyRevert, CeremonyResume,
		CeremonyRecoveryAuthorization:
	default:
		return fmt.Errorf("invalid type %q", ceremony.Type)
	}
	switch CeremonyPhase(ceremony.Phase) {
	case PhasePrevote, PhasePrecommit, PhaseFinalized, PhaseFailed:
	default:
		return fmt.Errorf("invalid phase %q", ceremony.Phase)
	}

	hasSnapshotData := ceremony.ElectorateSnapshotVersion != 0 ||
		len(ceremony.Electorate) != 0 ||
		ceremony.ElectorateTotalPower != "" ||
		ceremony.QuorumThreshold != 0 ||
		ceremony.MinDistinctVoters != 0
	if !hasSnapshotData {
		// Pre-hardening records remain query/export compatible. InitGenesis
		// terminalizes any such non-terminal record before it can progress or
		// apply an effect; terminal records are inert history.
		return nil
	}
	if ceremony.ElectorateSnapshotVersion != ElectorateSnapshotVersionV1 {
		return fmt.Errorf("unsupported electorate snapshot version %d", ceremony.ElectorateSnapshotVersion)
	}
	if ceremony.PrevoteDeadline <= ceremony.StartBlock {
		return fmt.Errorf("prevote deadline must be after start block")
	}
	if ceremony.PrecommitDeadline <= ceremony.PrevoteDeadline {
		return fmt.Errorf("precommit deadline must be after prevote deadline")
	}
	if ceremony.TimeoutDeadline < ceremony.PrecommitDeadline {
		return fmt.Errorf("timeout deadline cannot precede precommit deadline")
	}
	if err := validateCeremonyProposal(ceremony, ceremonyType); err != nil {
		return err
	}
	if ceremony.QuorumThreshold == 0 || ceremony.QuorumThreshold > 1_000_000 {
		return fmt.Errorf("invalid snapshotted quorum threshold %d", ceremony.QuorumThreshold)
	}
	if ceremony.MinDistinctVoters == 0 {
		return fmt.Errorf("snapshotted min_distinct_voters must be positive")
	}
	if len(ceremony.Electorate) == 0 {
		return fmt.Errorf("electorate snapshot cannot be empty")
	}
	if len(ceremony.Electorate) > MaxEmergencyElectorateSize {
		return fmt.Errorf(
			"electorate has %d members, exceeds consensus maximum %d",
			len(ceremony.Electorate),
			MaxEmergencyElectorateSize,
		)
	}
	if uint64(len(ceremony.Electorate)) < ceremony.MinDistinctVoters {
		return fmt.Errorf(
			"electorate has %d members, below snapshotted min_distinct_voters %d",
			len(ceremony.Electorate),
			ceremony.MinDistinctVoters,
		)
	}

	total, ok := new(big.Int).SetString(ceremony.ElectorateTotalPower, 10)
	if !ok || total.Sign() <= 0 {
		return fmt.Errorf("invalid electorate total power %q", ceremony.ElectorateTotalPower)
	}
	powers := make(map[string]*big.Int, len(ceremony.Electorate))
	actualTotal := new(big.Int)
	previousAddress := ""
	for i, member := range ceremony.Electorate {
		if member == nil {
			return fmt.Errorf("nil electorate member at index %d", i)
		}
		if member.Address == "" ||
			strings.TrimSpace(member.Address) != member.Address ||
			(i > 0 && member.Address <= previousAddress) {
			return fmt.Errorf("electorate addresses must be unique and in strict lexical order")
		}
		if _, err := sdk.AccAddressFromBech32(member.Address); err != nil {
			return fmt.Errorf(
				"electorate member %q is not a signable account address: %w",
				member.Address,
				err,
			)
		}
		power, ok := new(big.Int).SetString(member.Power, 10)
		if !ok || power.Sign() <= 0 {
			return fmt.Errorf("electorate member %q has invalid power %q", member.Address, member.Power)
		}
		powers[member.Address] = power
		actualTotal.Add(actualTotal, power)
		previousAddress = member.Address
	}
	if actualTotal.Cmp(total) != 0 {
		return fmt.Errorf("electorate total mismatch: members=%s stored=%s", actualTotal, total)
	}
	return validateGenesisCeremonyTallies(ceremony, powers)
}

func validateCeremonyProposal(ceremony *EmergencyCeremony, ceremonyType CeremonyType) error {
	if len(ceremony.ProposalData) == 0 {
		return fmt.Errorf("proposal_data cannot be empty")
	}
	switch ceremonyType {
	case CeremonyHalt:
		var proposal EmergencyHaltProposal
		if err := proto.Unmarshal(ceremony.ProposalData, &proposal); err != nil {
			return fmt.Errorf("invalid halt proposal data: %w", err)
		}
		if proposal.Id != ceremony.Id {
			return fmt.Errorf("halt proposal id %q does not match ceremony id %q", proposal.Id, ceremony.Id)
		}
		if proposal.Proposer == "" || proposal.Reason == "" {
			return fmt.Errorf("halt proposal must include proposer and reason")
		}
	case CeremonyRevert:
		var proposal EmergencyRevertProposal
		if err := proto.Unmarshal(ceremony.ProposalData, &proposal); err != nil {
			return fmt.Errorf("invalid revert proposal data: %w", err)
		}
		if proposal.Id != ceremony.Id {
			return fmt.Errorf("revert proposal id %q does not match ceremony id %q", proposal.Id, ceremony.Id)
		}
	case CeremonyResume:
		var proposal EmergencyResumeProposal
		if err := proto.Unmarshal(ceremony.ProposalData, &proposal); err != nil {
			return fmt.Errorf("invalid resume proposal data: %w", err)
		}
		if proposal.Id != ceremony.Id {
			return fmt.Errorf("resume proposal id %q does not match ceremony id %q", proposal.Id, ceremony.Id)
		}
		if proposal.Proposer == "" {
			return fmt.Errorf("resume proposal must include proposer")
		}
		if isNonterminalCeremony(ceremony) &&
			(proposal.HaltCeremonyId == "" ||
				proposal.Justification == "" ||
				!IsLowerSHA256(proposal.RecoveryManifestSha256)) {
			// Old active resume records are accepted without the new fields
			// only when their entire electorate snapshot is also absent;
			// InitGenesis will terminalize them below.
			hasSnapshot := ceremony.ElectorateSnapshotVersion != 0 ||
				len(ceremony.Electorate) != 0 ||
				ceremony.ElectorateTotalPower != "" ||
				ceremony.QuorumThreshold != 0 ||
				ceremony.MinDistinctVoters != 0
			if hasSnapshot {
				return fmt.Errorf("active resume proposal lacks valid quarantine linkage or recovery manifest")
			}
		}
	case CeremonyRecoveryAuthorization:
		var proposal EmergencyRecoveryAuthorizationProposal
		if err := proto.Unmarshal(ceremony.ProposalData, &proposal); err != nil {
			return fmt.Errorf(
				"invalid recovery authorization proposal data: %w",
				err,
			)
		}
		if proposal.Id != ceremony.Id {
			return fmt.Errorf(
				"recovery authorization proposal id %q does not match ceremony id %q",
				proposal.Id,
				ceremony.Id,
			)
		}
		if proposal.Proposer == "" ||
			proposal.HaltCeremonyId == "" ||
			proposal.SdkGovProposalId == 0 ||
			proposal.Generation == 0 ||
			!IsLowerSHA256(proposal.ActionSha256) ||
			!IsLowerSHA256(proposal.UpgradePlanSha256) ||
			!IsLowerSHA256(proposal.RecoveryManifestSha256) ||
			proposal.Justification == "" {
			return fmt.Errorf(
				"recovery authorization proposal is incomplete",
			)
		}
		if _, err := sdk.AccAddressFromBech32(
			proposal.AuthorizedSubmitter,
		); err != nil {
			return fmt.Errorf(
				"recovery authorization proposal has invalid authorized submitter: %w",
				err,
			)
		}
		switch proposal.ActionType {
		case "software_upgrade", "cancel_upgrade", "revoke":
		default:
			return fmt.Errorf(
				"recovery authorization proposal has invalid action type %q",
				proposal.ActionType,
			)
		}
	}
	return nil
}

func validateGenesisCeremonyTallies(
	ceremony *EmergencyCeremony,
	powers map[string]*big.Int,
) error {
	yes := new(big.Int)
	no := new(big.Int)
	approved := make(map[string]struct{}, len(ceremony.Prevotes))
	seenPrevotes := make(map[string]struct{}, len(ceremony.Prevotes))
	for i, entry := range ceremony.Prevotes {
		if entry == nil || entry.Value == nil || entry.Key == "" || entry.Key != entry.Value.Voter {
			return fmt.Errorf("invalid prevote entry at index %d", i)
		}
		if _, duplicate := seenPrevotes[entry.Key]; duplicate {
			return fmt.Errorf("duplicate prevote from %q", entry.Key)
		}
		power, found := powers[entry.Key]
		if !found {
			return fmt.Errorf("prevote from %q is outside electorate snapshot", entry.Key)
		}
		seenPrevotes[entry.Key] = struct{}{}
		if entry.Value.Approve {
			yes.Add(yes, power)
			approved[entry.Key] = struct{}{}
		} else {
			no.Add(no, power)
		}
	}

	precommit := new(big.Int)
	seenPrecommits := make(map[string]struct{}, len(ceremony.Precommits))
	for i, entry := range ceremony.Precommits {
		if entry == nil || entry.Value == nil || entry.Key == "" || entry.Key != entry.Value.Voter {
			return fmt.Errorf("invalid precommit entry at index %d", i)
		}
		if _, duplicate := seenPrecommits[entry.Key]; duplicate {
			return fmt.Errorf("duplicate precommit from %q", entry.Key)
		}
		if _, found := approved[entry.Key]; !found {
			return fmt.Errorf("precommit from %q lacks an approving prevote", entry.Key)
		}
		power, found := powers[entry.Key]
		if !found {
			return fmt.Errorf("precommit from %q is outside electorate snapshot", entry.Key)
		}
		seenPrecommits[entry.Key] = struct{}{}
		precommit.Add(precommit, power)
	}

	for _, tally := range []struct {
		name   string
		stored string
		want   *big.Int
	}{
		{"yes_prevote_stake", ceremony.YesPrevoteStake, yes},
		{"no_prevote_stake", ceremony.NoPrevoteStake, no},
		{"precommit_stake", ceremony.PrecommitStake, precommit},
	} {
		got, ok := new(big.Int).SetString(tally.stored, 10)
		if !ok || got.Sign() < 0 || got.Cmp(tally.want) != 0 {
			return fmt.Errorf("%s does not match snapshotted votes", tally.name)
		}
	}
	return nil
}

// Validate validates the params.
func (p *Params) Validate() error {
	if p.HaltQuorum == 0 || p.HaltQuorum > 1000000 {
		return fmt.Errorf("halt_quorum must be between 1 and 1000000, got %d", p.HaltQuorum)
	}
	if p.RevertQuorum == 0 || p.RevertQuorum > 1000000 {
		return fmt.Errorf("revert_quorum must be between 1 and 1000000, got %d", p.RevertQuorum)
	}
	if p.ResumeQuorum == 0 || p.ResumeQuorum > 1000000 {
		return fmt.Errorf("resume_quorum must be between 1 and 1000000, got %d", p.ResumeQuorum)
	}
	timings := []struct {
		name      string
		prevote   uint64
		precommit uint64
		timeout   uint64
	}{
		{"halt", p.HaltPrevoteBlocks, p.HaltPrecommitBlocks, p.HaltTimeoutBlocks},
		{"revert", p.RevertPrevoteBlocks, p.RevertPrecommitBlocks, p.RevertTimeoutBlocks},
		{"resume", p.ResumePrevoteBlocks, p.ResumePrecommitBlocks, p.ResumeTimeoutBlocks},
	}
	for _, timing := range timings {
		if timing.prevote == 0 || timing.precommit == 0 || timing.timeout == 0 {
			return fmt.Errorf("%s ceremony timing values must be > 0", timing.name)
		}
		if timing.prevote > ^uint64(0)-timing.precommit {
			return fmt.Errorf("%s ceremony phase duration overflows uint64", timing.name)
		}
		phaseDuration := timing.prevote + timing.precommit
		if timing.timeout < phaseDuration {
			return fmt.Errorf(
				"%s_timeout_blocks must be >= prevote + precommit (%d), got %d",
				timing.name,
				phaseDuration,
				timing.timeout,
			)
		}
	}
	if p.MaxProposalsPerEpoch == 0 || p.MaxProposalsPerGuardianPerEpoch == 0 {
		return fmt.Errorf("emergency proposal limits must be > 0")
	}
	minStake, ok := new(big.Int).SetString(p.MinGuardianStake, 10)
	if !ok || minStake.Sign() <= 0 {
		return fmt.Errorf("min_guardian_stake must be a positive base-10 integer")
	}
	if p.MinDistinctVoters == 0 {
		return fmt.Errorf("min_distinct_voters must be > 0")
	}
	if p.MaxRevertDepth == 0 {
		return fmt.Errorf("max_revert_depth must be > 0")
	}
	if p.EpochBlocks == 0 {
		return fmt.Errorf("epoch_blocks must be > 0")
	}
	if p.MaxHaltDurationBlocks == 0 {
		return fmt.Errorf("max_halt_duration_blocks must be > 0")
	}
	if len(p.GenesisCouncil) > 0 {
		if p.CouncilExpiryBlock == 0 {
			return fmt.Errorf("council_expiry_block must be > 0 when genesis_council is configured")
		}
		virtualStake, ok := new(big.Int).SetString(p.CouncilVirtualStake, 10)
		if !ok || virtualStake.Sign() <= 0 {
			return fmt.Errorf("council_virtual_stake must be a positive base-10 integer")
		}
		seen := make(map[string]struct{}, len(p.GenesisCouncil))
		for _, member := range p.GenesisCouncil {
			if member == "" {
				return fmt.Errorf("genesis_council cannot contain an empty address")
			}
			if _, err := sdk.AccAddressFromBech32(member); err != nil {
				return fmt.Errorf("genesis_council contains invalid account address %q: %w", member, err)
			}
			if _, duplicate := seen[member]; duplicate {
				return fmt.Errorf("genesis_council contains duplicate address %q", member)
			}
			seen[member] = struct{}{}
		}
	}
	return nil
}

// NormalizeLegacyParams fills only fields that older releases admitted but
// the hardened validator now requires. Values rejected by the old validator
// (for example quorum above 100%) remain invalid and fail closed.
func NormalizeLegacyParams(input *Params) *Params {
	if input == nil {
		return nil
	}
	p := proto.Clone(input).(*Params)
	defaults := DefaultParams()

	if p.HaltQuorum == 0 {
		p.HaltQuorum = defaults.HaltQuorum
	}
	if p.RevertQuorum == 0 {
		p.RevertQuorum = defaults.RevertQuorum
	}
	if p.ResumeQuorum == 0 {
		p.ResumeQuorum = defaults.ResumeQuorum
	}
	if p.HaltPrevoteBlocks != 0 &&
		p.HaltPrecommitBlocks != 0 &&
		p.HaltTimeoutBlocks != 0 &&
		(p.HaltPrevoteBlocks > ^uint64(0)-p.HaltPrecommitBlocks ||
			p.HaltTimeoutBlocks < p.HaltPrevoteBlocks+p.HaltPrecommitBlocks) {
		p.HaltPrevoteBlocks = defaults.HaltPrevoteBlocks
		p.HaltPrecommitBlocks = defaults.HaltPrecommitBlocks
		p.HaltTimeoutBlocks = defaults.HaltTimeoutBlocks
	}
	if p.RevertPrevoteBlocks == 0 {
		p.RevertPrevoteBlocks = defaults.RevertPrevoteBlocks
	}
	if p.RevertPrecommitBlocks == 0 {
		p.RevertPrecommitBlocks = defaults.RevertPrecommitBlocks
	}
	if p.RevertTimeoutBlocks == 0 {
		p.RevertTimeoutBlocks = defaults.RevertTimeoutBlocks
	}
	if p.RevertPrevoteBlocks > ^uint64(0)-p.RevertPrecommitBlocks ||
		p.RevertTimeoutBlocks < p.RevertPrevoteBlocks+p.RevertPrecommitBlocks {
		p.RevertPrevoteBlocks = defaults.RevertPrevoteBlocks
		p.RevertPrecommitBlocks = defaults.RevertPrecommitBlocks
		p.RevertTimeoutBlocks = defaults.RevertTimeoutBlocks
	}
	if p.ResumePrevoteBlocks == 0 {
		p.ResumePrevoteBlocks = defaults.ResumePrevoteBlocks
	}
	if p.ResumePrecommitBlocks == 0 {
		p.ResumePrecommitBlocks = defaults.ResumePrecommitBlocks
	}
	if p.ResumeTimeoutBlocks == 0 {
		p.ResumeTimeoutBlocks = defaults.ResumeTimeoutBlocks
	}
	if p.ResumePrevoteBlocks > ^uint64(0)-p.ResumePrecommitBlocks ||
		p.ResumeTimeoutBlocks < p.ResumePrevoteBlocks+p.ResumePrecommitBlocks {
		p.ResumePrevoteBlocks = defaults.ResumePrevoteBlocks
		p.ResumePrecommitBlocks = defaults.ResumePrecommitBlocks
		p.ResumeTimeoutBlocks = defaults.ResumeTimeoutBlocks
	}
	if p.MaxProposalsPerEpoch == 0 {
		p.MaxProposalsPerEpoch = defaults.MaxProposalsPerEpoch
	}
	if p.MaxProposalsPerGuardianPerEpoch == 0 {
		p.MaxProposalsPerGuardianPerEpoch = defaults.MaxProposalsPerGuardianPerEpoch
	}
	if minStake, ok := new(big.Int).SetString(p.MinGuardianStake, 10); !ok || minStake.Sign() <= 0 {
		p.MinGuardianStake = defaults.MinGuardianStake
	}
	if p.MinDistinctVoters == 0 {
		p.MinDistinctVoters = defaults.MinDistinctVoters
	}
	if p.MaxRevertDepth == 0 {
		p.MaxRevertDepth = defaults.MaxRevertDepth
	}
	if len(p.GenesisCouncil) > 0 {
		seen := make(map[string]struct{}, len(p.GenesisCouncil))
		validCouncil := p.CouncilExpiryBlock > 0
		virtualStake, ok := new(big.Int).SetString(p.CouncilVirtualStake, 10)
		validCouncil = validCouncil && ok && virtualStake.Sign() > 0
		for _, member := range p.GenesisCouncil {
			if _, err := sdk.AccAddressFromBech32(member); err != nil {
				validCouncil = false
				break
			}
			if _, duplicate := seen[member]; duplicate {
				validCouncil = false
				break
			}
			seen[member] = struct{}{}
		}
		if !validCouncil {
			// Invalid legacy council members could inflate the denominator but
			// can never sign a valid message. Disable that bootstrap authority.
			p.GenesisCouncil = []string{}
			p.CouncilExpiryBlock = 0
			p.CouncilVirtualStake = defaults.CouncilVirtualStake
		}
	}
	return p
}
