package types

import (
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Emergency status constants.
type EmergencyStatus string

const (
	StatusNormal       EmergencyStatus = "normal"
	StatusHaltVoting   EmergencyStatus = "halt_voting"
	StatusHalted       EmergencyStatus = "halted"
	StatusRevertVoting EmergencyStatus = "revert_voting"
	StatusReverting    EmergencyStatus = "reverting"
	StatusResumeVoting EmergencyStatus = "resume_voting"
)

// Ceremony type constants.
type CeremonyType string

const (
	CeremonyHalt                  CeremonyType = "halt"
	CeremonyRevert                CeremonyType = "revert"
	CeremonyResume                CeremonyType = "resume"
	CeremonyRecoveryAuthorization CeremonyType = "recovery_authorization"
)

// Ceremony phase constants.
type CeremonyPhase string

const (
	PhasePrevote   CeremonyPhase = "prevote"
	PhasePrecommit CeremonyPhase = "precommit"
	PhaseFinalized CeremonyPhase = "finalized"
	PhaseFailed    CeremonyPhase = "failed"
)

// Audit action constants.
type AuditAction string

const (
	AuditHaltProposed                   AuditAction = "halt_proposed"
	AuditHaltPrevote                    AuditAction = "halt_prevote"
	AuditHaltPrecommit                  AuditAction = "halt_precommit"
	AuditHaltExecuted                   AuditAction = "halt_executed"
	AuditHaltFailed                     AuditAction = "halt_failed"
	AuditRevertProposed                 AuditAction = "revert_proposed"
	AuditRevertPrevote                  AuditAction = "revert_prevote"
	AuditRevertPrecommit                AuditAction = "revert_precommit"
	AuditRevertExecuted                 AuditAction = "revert_executed"
	AuditRevertFailed                   AuditAction = "revert_failed"
	AuditResumeProposed                 AuditAction = "resume_proposed"
	AuditResumePrevote                  AuditAction = "resume_prevote"
	AuditResumePrecommit                AuditAction = "resume_precommit"
	AuditResumeExecuted                 AuditAction = "resume_executed"
	AuditResumeFailed                   AuditAction = "resume_failed"
	AuditRecoveryAuthorizationProposed  AuditAction = "recovery_authorization_proposed"
	AuditRecoveryAuthorizationPrevote   AuditAction = "recovery_authorization_prevote"
	AuditRecoveryAuthorizationPrecommit AuditAction = "recovery_authorization_precommit"
	AuditRecoveryAuthorized             AuditAction = "recovery_authorized"
	AuditRecoveryRevoked                AuditAction = "recovery_revoked"
	AuditRecoveryAuthorizationFailed    AuditAction = "recovery_authorization_failed"
	AuditLegacyNormalized               AuditAction = "legacy_state_normalized"
)

// Guardian tier constant (must match staking module TierGuardian value = 4).
const TierGuardian = uint32(4)

const (
	MaxEmergencyReasonBytes       = 4096
	MaxRecoveryJustificationBytes = 4096
	MaxEmergencyElectorateSize    = 100
	SHA256HexLength               = 64
	ElectorateSnapshotVersionV1   = uint32(1)

	// PostResumeCancellationGraceBlocks retains the quarantine release marker
	// for ten complete DeliverTx windows after admission reopens. Consumers with
	// pre-committed effects can use that interval to let owners cancel or reduce
	// work before paused backlogs resume.
	PostResumeCancellationGraceBlocks uint64 = 10
	MaxSDKBlockHeight                        = uint64(^uint64(0) >> 1)
)

const (
	MsgProposeHaltTypeURL                  = "/zerone.emergency.v1.MsgProposeHalt"
	MsgVoteHaltTypeURL                     = "/zerone.emergency.v1.MsgVoteHalt"
	MsgProposeRevertTypeURL                = "/zerone.emergency.v1.MsgProposeRevert"
	MsgVoteRevertTypeURL                   = "/zerone.emergency.v1.MsgVoteRevert"
	MsgProposeResumeTypeURL                = "/zerone.emergency.v1.MsgProposeResume"
	MsgVoteResumeTypeURL                   = "/zerone.emergency.v1.MsgVoteResume"
	MsgProposeRecoveryAuthorizationTypeURL = "/zerone.emergency.v1.MsgProposeRecoveryAuthorization"
	MsgVoteRecoveryAuthorizationTypeURL    = "/zerone.emergency.v1.MsgVoteRecoveryAuthorization"
)

// --- Ceremony helpers ---

// GetPrevote returns the prevote for a voter, if it exists.
func (c *EmergencyCeremony) GetPrevote(voter string) (*EmergencyVote, bool) {
	for _, entry := range c.Prevotes {
		if entry.Key == voter {
			return entry.Value, true
		}
	}
	return nil, false
}

// SetPrevote adds or updates a prevote for a voter.
func (c *EmergencyCeremony) SetPrevote(voter string, vote *EmergencyVote) {
	for i, entry := range c.Prevotes {
		if entry.Key == voter {
			c.Prevotes[i].Value = vote
			return
		}
	}
	c.Prevotes = append(c.Prevotes, &PrevoteEntry{Key: voter, Value: vote})
}

// GetPrecommit returns the precommit for a voter, if it exists.
func (c *EmergencyCeremony) GetPrecommit(voter string) (*EmergencyPrecommit, bool) {
	for _, entry := range c.Precommits {
		if entry.Key == voter {
			return entry.Value, true
		}
	}
	return nil, false
}

// SetPrecommit adds or updates a precommit for a voter.
func (c *EmergencyCeremony) SetPrecommit(voter string, precommit *EmergencyPrecommit) {
	for i, entry := range c.Precommits {
		if entry.Key == voter {
			c.Precommits[i].Value = precommit
			return
		}
	}
	c.Precommits = append(c.Precommits, &PrecommitEntry{Key: voter, Value: precommit})
}

// --- Message validation ---

func (msg *MsgProposeHalt) ValidateBasic() error {
	if msg.Proposer == "" {
		return fmt.Errorf("proposer cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(msg.Proposer); err != nil {
		return fmt.Errorf("invalid proposer address: %w", err)
	}
	if msg.Reason == "" {
		return fmt.Errorf("reason cannot be empty")
	}
	if len(msg.Reason) > MaxEmergencyReasonBytes {
		return fmt.Errorf("reason cannot exceed %d bytes", MaxEmergencyReasonBytes)
	}
	return nil
}

func (msg *MsgProposeHalt) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Proposer)
	return []sdk.AccAddress{addr}
}

func (msg *MsgVoteHalt) ValidateBasic() error {
	if msg.Voter == "" {
		return fmt.Errorf("voter cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(msg.Voter); err != nil {
		return fmt.Errorf("invalid voter address: %w", err)
	}
	if msg.ProposalId == "" {
		return fmt.Errorf("proposal_id cannot be empty")
	}
	return nil
}

func (msg *MsgVoteHalt) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Voter)
	return []sdk.AccAddress{addr}
}

func (msg *MsgProposeRevert) ValidateBasic() error {
	if msg.Proposer == "" {
		return fmt.Errorf("proposer cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(msg.Proposer); err != nil {
		return fmt.Errorf("invalid proposer address: %w", err)
	}
	if msg.RevertToHeight == 0 {
		return fmt.Errorf("revert_to_height must be > 0")
	}
	return nil
}

func (msg *MsgProposeRevert) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Proposer)
	return []sdk.AccAddress{addr}
}

func (msg *MsgVoteRevert) ValidateBasic() error {
	if msg.Voter == "" {
		return fmt.Errorf("voter cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(msg.Voter); err != nil {
		return fmt.Errorf("invalid voter address: %w", err)
	}
	if msg.ProposalId == "" {
		return fmt.Errorf("proposal_id cannot be empty")
	}
	return nil
}

func (msg *MsgVoteRevert) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Voter)
	return []sdk.AccAddress{addr}
}

func (msg *MsgProposeResume) ValidateBasic() error {
	if msg.Proposer == "" {
		return fmt.Errorf("proposer cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(msg.Proposer); err != nil {
		return fmt.Errorf("invalid proposer address: %w", err)
	}
	if msg.Justification == "" {
		return fmt.Errorf("justification cannot be empty")
	}
	if len(msg.Justification) > MaxRecoveryJustificationBytes {
		return fmt.Errorf("justification cannot exceed %d bytes", MaxRecoveryJustificationBytes)
	}
	if !IsLowerSHA256(msg.RecoveryManifestSha256) {
		return fmt.Errorf("recovery_manifest_sha256 must be exactly %d lowercase hexadecimal characters", SHA256HexLength)
	}
	return nil
}

func (msg *MsgProposeResume) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Proposer)
	return []sdk.AccAddress{addr}
}

func (msg *MsgVoteResume) ValidateBasic() error {
	if msg.Voter == "" {
		return fmt.Errorf("voter cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(msg.Voter); err != nil {
		return fmt.Errorf("invalid voter address: %w", err)
	}
	if msg.ProposalId == "" {
		return fmt.Errorf("proposal_id cannot be empty")
	}
	return nil
}

func (msg *MsgVoteResume) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Voter)
	return []sdk.AccAddress{addr}
}

func (msg *MsgProposeRecoveryAuthorization) ValidateBasic() error {
	if msg.Proposer == "" {
		return fmt.Errorf("proposer cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(msg.Proposer); err != nil {
		return fmt.Errorf("invalid proposer address: %w", err)
	}
	if msg.SdkGovProposalId == 0 {
		return fmt.Errorf("sdk_gov_proposal_id must be positive")
	}
	if !IsLowerSHA256(msg.ActionSha256) {
		return fmt.Errorf(
			"action_sha256 must be exactly %d lowercase hexadecimal characters",
			SHA256HexLength,
		)
	}
	if !IsLowerSHA256(msg.RecoveryManifestSha256) {
		return fmt.Errorf(
			"recovery_manifest_sha256 must be exactly %d lowercase hexadecimal characters",
			SHA256HexLength,
		)
	}
	if !IsLowerSHA256(msg.UpgradePlanSha256) {
		return fmt.Errorf(
			"upgrade_plan_sha256 must be exactly %d lowercase hexadecimal characters",
			SHA256HexLength,
		)
	}
	if _, err := sdk.AccAddressFromBech32(msg.AuthorizedSubmitter); err != nil {
		return fmt.Errorf("invalid authorized_submitter address: %w", err)
	}
	switch msg.ActionType {
	case "software_upgrade", "cancel_upgrade", "revoke":
	default:
		return fmt.Errorf(
			"action_type must be software_upgrade, cancel_upgrade, or revoke",
		)
	}
	if msg.Justification == "" {
		return fmt.Errorf("justification cannot be empty")
	}
	if len(msg.Justification) > MaxRecoveryJustificationBytes {
		return fmt.Errorf(
			"justification cannot exceed %d bytes",
			MaxRecoveryJustificationBytes,
		)
	}
	return nil
}

func (msg *MsgProposeRecoveryAuthorization) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Proposer)
	return []sdk.AccAddress{addr}
}

func (msg *MsgVoteRecoveryAuthorization) ValidateBasic() error {
	if msg.Voter == "" {
		return fmt.Errorf("voter cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(msg.Voter); err != nil {
		return fmt.Errorf("invalid voter address: %w", err)
	}
	if msg.ProposalId == "" {
		return fmt.Errorf("proposal_id cannot be empty")
	}
	return nil
}

func (msg *MsgVoteRecoveryAuthorization) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Voter)
	return []sdk.AccAddress{addr}
}

func (msg *MsgUpdateParams) ValidateBasic() error {
	if msg.Authority == "" {
		return fmt.Errorf("authority cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(msg.Authority); err != nil {
		return fmt.Errorf("invalid authority address: %w", err)
	}
	return nil
}

func (msg *MsgUpdateParams) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Authority)
	return []sdk.AccAddress{addr}
}

// --- Codec registration ---

func RegisterCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgProposeHalt{}, "emergency/MsgProposeHalt", nil)
	cdc.RegisterConcrete(&MsgVoteHalt{}, "emergency/MsgVoteHalt", nil)
	cdc.RegisterConcrete(&MsgProposeRevert{}, "emergency/MsgProposeRevert", nil)
	cdc.RegisterConcrete(&MsgVoteRevert{}, "emergency/MsgVoteRevert", nil)
	cdc.RegisterConcrete(&MsgProposeResume{}, "emergency/MsgProposeResume", nil)
	cdc.RegisterConcrete(&MsgVoteResume{}, "emergency/MsgVoteResume", nil)
	cdc.RegisterConcrete(&MsgProposeRecoveryAuthorization{}, "emergency/MsgProposeRecoveryAuthorization", nil)
	cdc.RegisterConcrete(&MsgVoteRecoveryAuthorization{}, "emergency/MsgVoteRecoveryAuthorization", nil)
	cdc.RegisterConcrete(&MsgUpdateParams{}, "emergency/MsgUpdateParams", nil)
}

func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgProposeHalt{},
		&MsgVoteHalt{},
		&MsgProposeRevert{},
		&MsgVoteRevert{},
		&MsgProposeResume{},
		&MsgVoteResume{},
		&MsgProposeRecoveryAuthorization{},
		&MsgVoteRecoveryAuthorization{},
		&MsgUpdateParams{},
	)
}

// IsLowerSHA256 reports whether value is a canonical lowercase SHA-256 hex
// digest. Recovery ceremonies bind to canonical manifest bytes through this
// exact representation.
func IsLowerSHA256(value string) bool {
	if len(value) != SHA256HexLength || value != strings.ToLower(value) {
		return false
	}
	decoded, err := hex.DecodeString(value)
	return err == nil && len(decoded) == SHA256HexLength/2
}

// IsEmergencyMsg returns true if the given message belongs to the emergency
// coordination module. The ante layer may apply a narrower policy; in
// particular, legacy height-only revert messages are decoded but fail closed in
// the message server.
func IsEmergencyMsg(msg sdk.Msg) bool {
	switch msg.(type) {
	case *MsgProposeHalt, *MsgVoteHalt,
		*MsgProposeRevert, *MsgVoteRevert,
		*MsgProposeResume, *MsgVoteResume,
		*MsgProposeRecoveryAuthorization,
		*MsgVoteRecoveryAuthorization:
		return true
	default:
		return false
	}
}

// IsEmergencyTypeURL is the Any-level equivalent of IsEmergencyMsg. It lets
// admission and upgrade audits reject emergency coordination hidden inside
// governance/authz wrappers without trusting an Any's cached concrete value.
func IsEmergencyTypeURL(typeURL string) bool {
	switch typeURL {
	case MsgProposeHaltTypeURL, MsgVoteHaltTypeURL,
		MsgProposeRevertTypeURL, MsgVoteRevertTypeURL,
		MsgProposeResumeTypeURL, MsgVoteResumeTypeURL,
		MsgProposeRecoveryAuthorizationTypeURL,
		MsgVoteRecoveryAuthorizationTypeURL:
		return true
	default:
		return false
	}
}
