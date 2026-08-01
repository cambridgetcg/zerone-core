package app

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"time"

	"cosmossdk.io/collections"
	upgradekeeper "cosmossdk.io/x/upgrade/keeper"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/gov"
	govkeeper "github.com/cosmos/cosmos-sdk/x/gov/keeper"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	govv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"

	emergencykeeper "github.com/zerone-chain/zerone/x/emergency/keeper"
	emergencytypes "github.com/zerone-chain/zerone/x/emergency/types"
	zeronegovkeeper "github.com/zerone-chain/zerone/x/gov/keeper"
)

const (
	msgSoftwareUpgradeTypeURL = "/cosmos.upgrade.v1beta1.MsgSoftwareUpgrade"
	msgCancelUpgradeTypeURL   = "/cosmos.upgrade.v1beta1.MsgCancelUpgrade"

	govQuarantineFailureReason  = "proposal permanently failed because application transaction quarantine permits only one exact expedited software-upgrade or cancel message"
	govRecoveryRevokedReason    = "proposal permanently failed because its emergency recovery authorization was revoked"
	govQuarantineInactiveDomain = byte(0)
	govQuarantineActiveDomain   = byte(1)
)

type emergencyQuarantineReader interface {
	IsHalted(context.Context) bool
}

// emergencyAwareGovAppModule preserves the upstream x/gov module in every
// respect except EndBlock. DeliverTx can finalize an emergency halt before
// EndBlock runs, so ante admission alone cannot stop an already queued
// governance proposal from executing in the halt block. While quarantined this
// wrapper permanently fails every due action except the deliberately narrow
// expedited recovery lane, then delegates tally and execution to upstream
// x/gov. Future-deadline proposals stay frozen in their existing queues; this
// keeps the halt-block workload bounded by the same due range upstream x/gov
// would otherwise process.
type emergencyAwareGovAppModule struct {
	gov.AppModule

	keeper    *govkeeper.Keeper
	emergency emergencykeeper.Keeper
	upgrade   *upgradekeeper.Keeper
	customGov zeronegovkeeper.Keeper
}

func newEmergencyAwareGovAppModule(
	module gov.AppModule,
	keeper *govkeeper.Keeper,
	emergency emergencykeeper.Keeper,
	upgrade *upgradekeeper.Keeper,
	customGov zeronegovkeeper.Keeper,
) emergencyAwareGovAppModule {
	return emergencyAwareGovAppModule{
		AppModule: module,
		keeper:    keeper,
		emergency: emergency,
		upgrade:   upgrade,
		customGov: customGov,
	}
}

func (am emergencyAwareGovAppModule) EndBlock(goCtx context.Context) error {
	if am.keeper == nil {
		return fmt.Errorf("quarantine-aware SDK governance is not configured")
	}
	_, reviewHeld, err := am.customGov.GetEmergencyTransitionHold(
		sdk.UnwrapSDKContext(goCtx),
	)
	if err != nil {
		return err
	}
	if !am.emergency.IsHalted(goCtx) && !reviewHeld {
		return am.AppModule.EndBlock(goCtx)
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	count, digest, err := am.failDisallowedQueuedProposals(ctx)
	if err != nil {
		return err
	}
	if count > 0 {
		ctx.EventManager().EmitEvent(sdk.NewEvent(
			"zerone.gov.proposals_quarantined",
			sdk.NewAttribute("failed_count", fmt.Sprintf("%d", count)),
			sdk.NewAttribute("queue_manifest_sha256", fmt.Sprintf("%x", digest)),
			sdk.NewAttribute("allowed_lane", "expedited_single_upgrade_or_cancel"),
		))
	}

	// The remaining queue contains only exact expedited recovery actions.
	if err := am.AppModule.EndBlock(goCtx); err != nil {
		return err
	}
	return am.reconcileAuthorizedRecoveryProposal(ctx)
}

func (am emergencyAwareGovAppModule) failDisallowedQueuedProposals(
	ctx sdk.Context,
) (int, [sha256.Size]byte, error) {
	hasher := sha256.New()
	count := 0

	dueRange := collections.NewPrefixUntilPairRange[time.Time, uint64](
		ctx.BlockTime(),
	)
	inactive, err := am.keeper.InactiveProposalsQueue.Iterate(ctx, dueRange)
	if err != nil {
		return 0, [sha256.Size]byte{}, fmt.Errorf(
			"iterate inactive SDK governance queue during quarantine: %w",
			err,
		)
	}
	inactiveEntries, err := inactive.KeyValues()
	if err != nil {
		return 0, [sha256.Size]byte{}, fmt.Errorf(
			"read inactive SDK governance queue during quarantine: %w",
			err,
		)
	}
	active, err := am.keeper.ActiveProposalsQueue.Iterate(ctx, dueRange)
	if err != nil {
		return 0, [sha256.Size]byte{}, fmt.Errorf(
			"iterate active SDK governance queue during quarantine: %w",
			err,
		)
	}
	activeEntries, err := active.KeyValues()
	if err != nil {
		return 0, [sha256.Size]byte{}, fmt.Errorf(
			"read active SDK governance queue during quarantine: %w",
			err,
		)
	}

	for _, entry := range inactiveEntries {
		failed, err := am.failDisallowedQueuedProposal(
			ctx,
			govQuarantineInactiveDomain,
			entry.Key.K1(),
			entry.Key.K2(),
			entry.Value,
			govv1.StatusDepositPeriod,
			func() error {
				return am.keeper.InactiveProposalsQueue.Remove(ctx, entry.Key)
			},
		)
		if err != nil {
			return 0, [sha256.Size]byte{}, err
		}
		if failed {
			appendGovQuarantineManifestEntry(
				hasher,
				govQuarantineInactiveDomain,
				entry.Key.K1(),
				entry.Key.K2(),
				entry.Value,
			)
			count++
		}
	}

	for _, entry := range activeEntries {
		failed, err := am.failDisallowedQueuedProposal(
			ctx,
			govQuarantineActiveDomain,
			entry.Key.K1(),
			entry.Key.K2(),
			entry.Value,
			govv1.StatusVotingPeriod,
			func() error {
				return am.keeper.ActiveProposalsQueue.Remove(ctx, entry.Key)
			},
		)
		if err != nil {
			return 0, [sha256.Size]byte{}, err
		}
		if failed {
			appendGovQuarantineManifestEntry(
				hasher,
				govQuarantineActiveDomain,
				entry.Key.K1(),
				entry.Key.K2(),
				entry.Value,
			)
			count++
		}
	}

	var digest [sha256.Size]byte
	copy(digest[:], hasher.Sum(nil))
	return count, digest, nil
}

func (am emergencyAwareGovAppModule) failDisallowedQueuedProposal(
	ctx sdk.Context,
	domain byte,
	deadline time.Time,
	keyProposalID uint64,
	valueProposalID uint64,
	expectedStatus govv1.ProposalStatus,
	removeQueueEntry func() error,
) (bool, error) {
	proposal, err := am.keeper.Proposals.Get(ctx, keyProposalID)
	decodable := err == nil
	switch {
	case err == nil:
	case errors.Is(err, collections.ErrEncoding):
		proposal.Id = keyProposalID
	case errors.Is(err, collections.ErrNotFound):
		// A missing/corrupt queue target is inert, but its deposits and queue
		// entry still need deterministic cleanup before upstream EndBlock.
		proposal = govv1.Proposal{Id: keyProposalID}
	default:
		return false, fmt.Errorf(
			"read SDK governance proposal %d during quarantine: %w",
			keyProposalID,
			err,
		)
	}

	allowed := decodable &&
		keyProposalID == valueProposalID &&
		proposal.Id == keyProposalID &&
		proposal.Status == expectedStatus &&
		proposalQueueDeadlineMatches(
			proposal,
			expectedStatus,
			deadline,
		) &&
		am.isAuthorizedRecoveryProposal(ctx, proposal)
	if allowed {
		return false, nil
	}

	if err := removeQueueEntry(); err != nil {
		return false, fmt.Errorf(
			"remove quarantined SDK governance queue entry domain=%d deadline=%s key_id=%d value_id=%d: %w",
			domain,
			deadline.UTC().Format(time.RFC3339Nano),
			keyProposalID,
			valueProposalID,
			err,
		)
	}
	if err := am.keeper.RefundAndDeleteDeposits(ctx, keyProposalID); err != nil {
		return false, fmt.Errorf(
			"refund quarantined SDK governance proposal %d deposits: %w",
			keyProposalID,
			err,
		)
	}
	if decodable &&
		(proposal.Status == govv1.StatusPassed ||
			proposal.Status == govv1.StatusRejected ||
			proposal.Status == govv1.StatusFailed) {
		// A latent corrupt alias can become due after the canonical record was
		// already terminalized. Remove the inert queue/deposit residue, but
		// never rewrite the immutable terminal audit outcome.
		return true, nil
	}

	// Preserve the original executable-message audit record whenever it was
	// decoded successfully. Only an undecodable/synthetic proposal needs its
	// messages cleared so SetProposal can persist a queryable failed record.
	proposal.Id = keyProposalID
	if !decodable {
		proposal.Messages = nil
	}
	proposal.Status = govv1.StatusFailed
	proposal.FailedReason = govQuarantineFailureReason
	if err := am.keeper.SetProposal(ctx, proposal); err != nil {
		return false, fmt.Errorf(
			"persist quarantined SDK governance proposal %d failure: %w",
			keyProposalID,
			err,
		)
	}
	return true, nil
}

func proposalQueueDeadlineMatches(
	proposal govv1.Proposal,
	expectedStatus govv1.ProposalStatus,
	deadline time.Time,
) bool {
	switch expectedStatus {
	case govv1.StatusDepositPeriod:
		return proposal.DepositEndTime != nil &&
			proposal.DepositEndTime.Equal(deadline)
	case govv1.StatusVotingPeriod:
		return proposal.VotingEndTime != nil &&
			proposal.VotingEndTime.Equal(deadline)
	default:
		return false
	}
}

type govQuarantineManifestWriter interface {
	Write([]byte) (int, error)
}

func appendGovQuarantineManifestEntry(
	writer govQuarantineManifestWriter,
	domain byte,
	deadline time.Time,
	keyProposalID uint64,
	valueProposalID uint64,
) {
	_, _ = writer.Write([]byte{domain})
	deadlineText := []byte(deadline.UTC().Format(time.RFC3339Nano))
	var scalar [8]byte
	binary.BigEndian.PutUint64(scalar[:], uint64(len(deadlineText)))
	_, _ = writer.Write(scalar[:])
	_, _ = writer.Write(deadlineText)
	binary.BigEndian.PutUint64(scalar[:], keyProposalID)
	_, _ = writer.Write(scalar[:])
	binary.BigEndian.PutUint64(scalar[:], valueProposalID)
	_, _ = writer.Write(scalar[:])
}

// sdkGovRecoveryProposalReader composes the SDK-governance sequence and
// proposal stores, the active emergency incident, and x/upgrade plan state.
// Any lookup or decode error is deliberately treated as "not allowed" by ante.
type sdkGovRecoveryProposalReader struct {
	keeper    *govkeeper.Keeper
	emergency emergencykeeper.Keeper
	upgrade   *upgradekeeper.Keeper
	customGov *zeronegovkeeper.Keeper
}

func newSDKGovRecoveryProposalReader(
	keeper *govkeeper.Keeper,
	emergency emergencykeeper.Keeper,
	upgrade *upgradekeeper.Keeper,
	customGov ...*zeronegovkeeper.Keeper,
) sdkGovRecoveryProposalReader {
	reader := sdkGovRecoveryProposalReader{
		keeper:    keeper,
		emergency: emergency,
		upgrade:   upgrade,
	}
	if len(customGov) > 0 {
		reader.customGov = customGov[0]
	}
	return reader
}

func (r sdkGovRecoveryProposalReader) IsGovernanceReviewHeld(
	ctx sdk.Context,
) (bool, error) {
	if r.customGov == nil {
		return false, nil
	}
	_, found, err := r.customGov.GetEmergencyTransitionHold(ctx)
	return found, err
}

// VerifyRecoveryAuthorizationTarget is called both when a Guardian ceremony
// opens and when it finalizes. Proposal submission is still quarantined, so
// proposalID must remain the exact next sequence value. For cancellation, the
// plan digest must continue to identify the currently scheduled plan.
func (r sdkGovRecoveryProposalReader) VerifyRecoveryAuthorizationTarget(
	ctx context.Context,
	proposalID uint64,
	actionSHA256 string,
	upgradePlanSHA256 string,
	actionType string,
) error {
	if r.keeper == nil || r.upgrade == nil {
		return fmt.Errorf("SDK governance or upgrade keeper is unavailable")
	}
	if !emergencytypes.IsLowerSHA256(actionSHA256) ||
		!emergencytypes.IsLowerSHA256(upgradePlanSHA256) {
		return fmt.Errorf("recovery action and plan digests must be canonical SHA-256 values")
	}
	switch actionType {
	case "software_upgrade":
	case "cancel_upgrade":
		plan, err := r.upgrade.GetUpgradePlan(ctx)
		if err != nil {
			return fmt.Errorf(
				"read scheduled plan targeted by recovery cancellation: %w",
				err,
			)
		}
		if got := UpgradePlanSHA256(plan); got != upgradePlanSHA256 {
			return fmt.Errorf(
				"scheduled plan digest %s does not match Guardian target %s",
				got,
				upgradePlanSHA256,
			)
		}
	case "revoke":
		authorization, ok := r.liveAuthorization(ctx)
		if !ok ||
			authorization.SdkGovProposalId != proposalID ||
			authorization.ActionSha256 != actionSHA256 ||
			authorization.UpgradePlanSha256 != upgradePlanSHA256 {
			return fmt.Errorf(
				"recovery revocation does not match the live authorization",
			)
		}
		return nil
	default:
		return fmt.Errorf("unsupported recovery action type %q", actionType)
	}
	nextProposalID, err := r.keeper.ProposalID.Peek(ctx)
	if err != nil {
		return fmt.Errorf("read next SDK governance proposal id: %w", err)
	}
	if proposalID != nextProposalID {
		return fmt.Errorf(
			"Guardian target proposal id %d is not the exact next SDK governance id %d",
			proposalID,
			nextProposalID,
		)
	}
	if _, err := r.keeper.Proposals.Get(ctx, proposalID); err == nil {
		return fmt.Errorf(
			"SDK governance proposal %d already exists before authorization finalization",
			proposalID,
		)
	} else if !errors.Is(err, collections.ErrNotFound) {
		return fmt.Errorf(
			"check pre-authorized SDK governance proposal %d absence: %w",
			proposalID,
			err,
		)
	}
	return nil
}

func (r sdkGovRecoveryProposalReader) IsAuthorizedRecoverySubmission(
	ctx sdk.Context,
	msg *govv1.MsgSubmitProposal,
) bool {
	if msg == nil || !msg.Expedited ||
		len(msg.Messages) != 1 ||
		msg.Proposer == "" {
		return false
	}
	authorization, ok := r.liveAuthorization(ctx)
	if !ok ||
		msg.Proposer != authorization.AuthorizedSubmitter {
		return false
	}
	nextProposalID, err := r.keeper.ProposalID.Peek(ctx)
	if err != nil ||
		nextProposalID != authorization.SdkGovProposalId {
		return false
	}
	return r.matchesAuthorizedAction(
		ctx,
		msg.Messages[0],
		authorization,
	)
}

func (r sdkGovRecoveryProposalReader) IsExpeditedRecoveryProposal(
	ctx sdk.Context,
	proposalID uint64,
) bool {
	if r.keeper == nil {
		return false
	}
	proposal, err := r.keeper.Proposals.Get(ctx, proposalID)
	if err != nil {
		return false
	}
	return r.IsAuthorizedRecoveryProposal(ctx, proposal)
}

func (r sdkGovRecoveryProposalReader) IsAuthorizedRecoveryProposal(
	ctx sdk.Context,
	proposal govv1.Proposal,
) bool {
	authorization, ok := r.liveAuthorization(ctx)
	if !ok ||
		proposal.Id != authorization.SdkGovProposalId ||
		!proposal.Expedited ||
		proposal.Proposer != authorization.AuthorizedSubmitter ||
		len(proposal.Messages) != 1 {
		return false
	}
	return r.matchesAuthorizedAction(
		ctx,
		proposal.Messages[0],
		authorization,
	)
}

func (r sdkGovRecoveryProposalReader) liveAuthorization(
	ctx context.Context,
) (*emergencytypes.EmergencyRecoveryAuthorization, bool) {
	if r.keeper == nil || r.upgrade == nil {
		return nil, false
	}
	authorization, found, err := r.emergency.GetRecoveryAuthorization(ctx)
	if err != nil || !found ||
		authorization.Outcome != "" ||
		authorization.HaltCeremonyId == "" ||
		authorization.HaltCeremonyId !=
			r.emergency.GetActiveHaltCeremonyId(ctx) ||
		!r.emergency.IsHalted(ctx) {
		return nil, false
	}
	return authorization, true
}

func (r sdkGovRecoveryProposalReader) matchesAuthorizedAction(
	ctx context.Context,
	action *codectypes.Any,
	authorization *emergencytypes.EmergencyRecoveryAuthorization,
) bool {
	if action == nil ||
		RecoveryActionSHA256(action) != authorization.ActionSha256 {
		return false
	}
	switch action.TypeUrl {
	case msgSoftwareUpgradeTypeURL:
		if authorization.ActionType != "software_upgrade" {
			return false
		}
		var msg upgradetypes.MsgSoftwareUpgrade
		if err := msg.Unmarshal(action.Value); err != nil ||
			msg.Authority != r.keeper.GetAuthority() ||
			msg.Plan.ValidateBasic() != nil ||
			msg.Plan.Height <=
				sdk.UnwrapSDKContext(ctx).BlockHeight() {
			return false
		}
		return UpgradePlanSHA256(msg.Plan) ==
			authorization.UpgradePlanSha256
	case msgCancelUpgradeTypeURL:
		if authorization.ActionType != "cancel_upgrade" {
			return false
		}
		var msg upgradetypes.MsgCancelUpgrade
		if err := msg.Unmarshal(action.Value); err != nil ||
			msg.Authority != r.keeper.GetAuthority() {
			return false
		}
		plan, err := r.upgrade.GetUpgradePlan(ctx)
		return err == nil &&
			UpgradePlanSHA256(plan) ==
				authorization.UpgradePlanSha256
	default:
		return false
	}
}

// isExactRecoveryUpgradeMessages accepts one action only. In particular,
// arbitrary governance messages, mixed batches, authz wrappers, and legacy
// content proposals stay quarantined.
func isExactRecoveryUpgradeMessages(messages []*codectypes.Any) bool {
	if len(messages) != 1 || messages[0] == nil {
		return false
	}

	switch messages[0].TypeUrl {
	case msgSoftwareUpgradeTypeURL, msgCancelUpgradeTypeURL:
		return true
	default:
		return false
	}
}

var _ EmergencyRecoveryProposalReader = sdkGovRecoveryProposalReader{}
var _ emergencytypes.RecoveryAuthorizationTargetVerifier = sdkGovRecoveryProposalReader{}

// RecoveryActionSHA256 hashes exactly the TypeURL and raw Any value that SDK
// governance will persist and execute.
func RecoveryActionSHA256(action *codectypes.Any) string {
	if action == nil {
		return ""
	}
	hasher := sha256.New()
	_, _ = hasher.Write([]byte("zerone.emergency/recovery-action/v1\x00"))
	writeRecoveryDigestField(hasher, []byte(action.TypeUrl))
	writeRecoveryDigestField(hasher, action.Value)
	return fmt.Sprintf("%x", hasher.Sum(nil))
}

// UpgradePlanSHA256 hashes all consensus-relevant plan fields with explicit
// framing. For cancellation this binds the exact plan that currently exists.
func UpgradePlanSHA256(plan upgradetypes.Plan) string {
	hasher := sha256.New()
	_, _ = hasher.Write([]byte("zerone.emergency/upgrade-plan/v1\x00"))
	writeRecoveryDigestField(hasher, []byte(plan.Name))
	var height [8]byte
	binary.BigEndian.PutUint64(height[:], uint64(plan.Height))
	_, _ = hasher.Write(height[:])
	writeRecoveryDigestField(hasher, []byte(plan.Info))
	return fmt.Sprintf("%x", hasher.Sum(nil))
}

func writeRecoveryDigestField(writer govQuarantineManifestWriter, value []byte) {
	var size [8]byte
	binary.BigEndian.PutUint64(size[:], uint64(len(value)))
	_, _ = writer.Write(size[:])
	_, _ = writer.Write(value)
}

func (am emergencyAwareGovAppModule) isAuthorizedRecoveryProposal(
	ctx sdk.Context,
	proposal govv1.Proposal,
) bool {
	return newSDKGovRecoveryProposalReader(
		am.keeper,
		am.emergency,
		am.upgrade,
	).IsAuthorizedRecoveryProposal(ctx, proposal)
}

func (am emergencyAwareGovAppModule) reconcileAuthorizedRecoveryProposal(
	ctx sdk.Context,
) error {
	authorization, found, err := am.emergency.GetRecoveryAuthorization(ctx)
	if err != nil || !found {
		return err
	}
	if authorization.Outcome == "revoked" {
		return am.failRevokedRecoveryProposal(ctx, authorization)
	}
	if authorization.Outcome != "" {
		return nil
	}
	proposal, err := am.keeper.Proposals.Get(
		ctx,
		authorization.SdkGovProposalId,
	)
	if errors.Is(err, collections.ErrNotFound) {
		nextID, nextErr := am.keeper.ProposalID.Peek(ctx)
		if nextErr != nil {
			return fmt.Errorf(
				"read SDK governance sequence while reconciling recovery authorization: %w",
				nextErr,
			)
		}
		switch {
		case nextID == authorization.SdkGovProposalId:
			return nil
		case nextID > authorization.SdkGovProposalId:
			if err := am.emergency.MarkRecoveryAuthorizationTerminal(
				ctx,
				authorization.SdkGovProposalId,
				authorization.ActionSha256,
				"failed",
			); err != nil {
				return err
			}
			ctx.EventManager().EmitEvent(sdk.NewEvent(
				"zerone.emergency.recovery_authorization_terminal",
				sdk.NewAttribute(
					"sdk_gov_proposal_id",
					fmt.Sprintf(
						"%d",
						authorization.SdkGovProposalId,
					),
				),
				sdk.NewAttribute("outcome", "failed"),
				sdk.NewAttribute(
					"reason",
					"preauthorized_proposal_id_was_skipped",
				),
			))
			return nil
		default:
			return fmt.Errorf(
				"recovery authorization proposal id %d is ahead of SDK governance next id %d",
				authorization.SdkGovProposalId,
				nextID,
			)
		}
	}
	if err != nil {
		return fmt.Errorf(
			"read authorized SDK governance proposal %d: %w",
			authorization.SdkGovProposalId,
			err,
		)
	}
	switch proposal.Status {
	case govv1.StatusPassed:
		return am.markRecoveryAuthorizationOutcome(
			ctx,
			authorization,
			"passed",
			"proposal_passed_and_action_executed",
		)
	case govv1.StatusFailed:
		return am.markRecoveryAuthorizationOutcome(
			ctx,
			authorization,
			"failed",
			"proposal_action_failed",
		)
	case govv1.StatusRejected:
		return am.markRecoveryAuthorizationOutcome(
			ctx,
			authorization,
			"rejected",
			"proposal_rejected",
		)
	case govv1.StatusDepositPeriod, govv1.StatusVotingPeriod:
	default:
		return am.failAuthorizedRecoveryProposal(
			ctx,
			proposal,
			authorization,
			"authorized recovery proposal entered an unsupported nonterminal status",
		)
	}

	reader := newSDKGovRecoveryProposalReader(
		am.keeper,
		am.emergency,
		am.upgrade,
	)
	if !reader.IsAuthorizedRecoveryProposal(ctx, proposal) {
		reason := "authorized recovery proposal no longer matches its incident, action, submitter, or target plan"
		if proposal.Status == govv1.StatusVotingPeriod &&
			!proposal.Expedited {
			reason = "expedited recovery proposal did not pass and cannot be converted into an ordinary post-incident proposal"
		}
		return am.failAuthorizedRecoveryProposal(
			ctx,
			proposal,
			authorization,
			reason,
		)
	}
	return nil
}

// failRevokedRecoveryProposal closes the exact proposal linked to a revoked
// Guardian capability before transaction admission can reopen. Future queue
// entries are otherwise invisible to the quarantine due-range scan, and an
// upstream x/gov EndBlock after resume could execute the revoked action.
//
// A proposal that was already terminal is immutable audit history. Absence is
// also valid when Guardians revoke an authorization before its exact proposal
// is submitted.
func (am emergencyAwareGovAppModule) failRevokedRecoveryProposal(
	ctx sdk.Context,
	authorization *emergencytypes.EmergencyRecoveryAuthorization,
) error {
	proposal, err := am.keeper.Proposals.Get(
		ctx,
		authorization.SdkGovProposalId,
	)
	if errors.Is(err, collections.ErrNotFound) {
		return nil
	}
	if err != nil {
		return fmt.Errorf(
			"read revoked recovery proposal %d: %w",
			authorization.SdkGovProposalId,
			err,
		)
	}
	switch proposal.Status {
	case govv1.StatusPassed, govv1.StatusRejected, govv1.StatusFailed:
		return nil
	case govv1.StatusDepositPeriod, govv1.StatusVotingPeriod:
		return am.terminalizeRecoveryProposal(
			ctx,
			proposal,
			govRecoveryRevokedReason,
		)
	default:
		return fmt.Errorf(
			"revoked recovery proposal %d has unsupported status %s",
			proposal.Id,
			proposal.Status,
		)
	}
}

func (am emergencyAwareGovAppModule) failAuthorizedRecoveryProposal(
	ctx sdk.Context,
	proposal govv1.Proposal,
	authorization *emergencytypes.EmergencyRecoveryAuthorization,
	reason string,
) error {
	if err := am.terminalizeRecoveryProposal(
		ctx,
		proposal,
		reason,
	); err != nil {
		return err
	}
	return am.markRecoveryAuthorizationOutcome(
		ctx,
		authorization,
		"failed",
		reason,
	)
}

func (am emergencyAwareGovAppModule) terminalizeRecoveryProposal(
	ctx sdk.Context,
	proposal govv1.Proposal,
	reason string,
) error {
	switch proposal.Status {
	case govv1.StatusDepositPeriod:
		if proposal.DepositEndTime == nil {
			return fmt.Errorf(
				"authorized recovery proposal %d deposit status has no deadline",
				proposal.Id,
			)
		}
		key := collections.Join(*proposal.DepositEndTime, proposal.Id)
		has, err := am.keeper.InactiveProposalsQueue.Has(ctx, key)
		if err != nil {
			return err
		}
		if has {
			if err := am.keeper.InactiveProposalsQueue.Remove(
				ctx,
				key,
			); err != nil {
				return err
			}
		}
	case govv1.StatusVotingPeriod:
		if proposal.VotingEndTime == nil {
			return fmt.Errorf(
				"authorized recovery proposal %d voting status has no deadline",
				proposal.Id,
			)
		}
		key := collections.Join(*proposal.VotingEndTime, proposal.Id)
		has, err := am.keeper.ActiveProposalsQueue.Has(ctx, key)
		if err != nil {
			return err
		}
		if has {
			if err := am.keeper.ActiveProposalsQueue.Remove(
				ctx,
				key,
			); err != nil {
				return err
			}
		}
	}
	if err := am.keeper.RefundAndDeleteDeposits(
		ctx,
		proposal.Id,
	); err != nil {
		return fmt.Errorf(
			"refund terminalized recovery proposal %d deposits: %w",
			proposal.Id,
			err,
		)
	}
	proposal.Status = govv1.StatusFailed
	proposal.FailedReason = reason
	if err := am.keeper.SetProposal(ctx, proposal); err != nil {
		return fmt.Errorf(
			"persist terminalized recovery proposal %d: %w",
			proposal.Id,
			err,
		)
	}
	return nil
}

func (am emergencyAwareGovAppModule) markRecoveryAuthorizationOutcome(
	ctx sdk.Context,
	authorization *emergencytypes.EmergencyRecoveryAuthorization,
	outcome string,
	reason string,
) error {
	if err := am.emergency.MarkRecoveryAuthorizationTerminal(
		ctx,
		authorization.SdkGovProposalId,
		authorization.ActionSha256,
		outcome,
	); err != nil {
		return err
	}
	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"zerone.emergency.recovery_authorization_terminal",
		sdk.NewAttribute(
			"authorization_ceremony_id",
			authorization.AuthorizationCeremonyId,
		),
		sdk.NewAttribute(
			"sdk_gov_proposal_id",
			fmt.Sprintf("%d", authorization.SdkGovProposalId),
		),
		sdk.NewAttribute("outcome", outcome),
		sdk.NewAttribute("reason", reason),
	))
	return nil
}

// ensureNoActiveEmergencyGovProposals fails a named activation if an
// executable SDK-governance proposal contains emergency coordination
// messages, directly or through authz/governance wrappers. Its input is the
// root-verified union of deposit/voting proposals and every proposal referenced
// by ActiveProposalsQueue. The prestate collector first requires every queue
// record to match a voting-period proposal and its exact voting deadline.
//
// Current SDK governance already requires an embedded message's sole signer to
// be the governance module account, which rejects direct Guardian
// impersonation at submission. This audit still protects imported or manually
// repaired legacy state. The post-signature execution marker in x/emergency is
// an independent runtime backstop.
func (app *ZeroneApp) ensureNoActiveEmergencyGovProposals(
	ctx context.Context,
	committedRecords []upgradePrestateRecord,
) error {
	if app.GovKeeper == nil {
		return fmt.Errorf("SDK governance keeper is unavailable")
	}
	prefix := govtypes.ProposalsKeyPrefix.Bytes()
	for _, record := range committedRecords {
		if !bytes.HasPrefix(record.Key, prefix) ||
			len(record.Key) != len(prefix)+8 {
			return fmt.Errorf(
				"invalid committed SDK governance proposal key %x",
				record.Key,
			)
		}
		proposalID := binary.BigEndian.Uint64(record.Key[len(prefix):])
		var envelope govv1.Proposal
		if err := envelope.Unmarshal(record.Value); err != nil {
			return fmt.Errorf(
				"decode committed SDK governance proposal envelope %d during emergency authority audit: %w",
				proposalID,
				err,
			)
		}
		if envelope.Id != proposalID {
			return fmt.Errorf(
				"committed SDK governance proposal key %d does not match encoded id %d",
				proposalID,
				envelope.Id,
			)
		}
		var proposal govv1.Proposal
		if err := app.appCodec.Unmarshal(record.Value, &proposal); err != nil {
			return fmt.Errorf(
				"unpack executable SDK governance proposal %d during emergency authority audit: %w",
				proposalID,
				err,
			)
		}
		messages, err := proposal.GetMsgs()
		if err != nil {
			return fmt.Errorf(
				"decode executable SDK governance proposal %d during emergency authority audit: %w",
				proposalID,
				err,
			)
		}
		unsafe, err := containsEmergencyExecutableMessage(messages)
		if err != nil {
			return fmt.Errorf(
				"inspect executable SDK governance proposal %d during emergency authority audit: %w",
				proposalID,
				err,
			)
		}
		if unsafe {
			return fmt.Errorf(
				"executable SDK governance proposal %d contains emergency coordination execution; cancel it and remove any stale active-queue entry before activation",
				proposalID,
			)
		}
	}
	_ = ctx
	return nil
}
