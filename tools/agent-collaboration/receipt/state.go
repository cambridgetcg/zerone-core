package receipt

import (
	"fmt"
	"sort"
)

const (
	maxTaskTextEntries = 128
	maxTaskDigests     = 256
)

type reducerTask struct {
	summary TaskSummary

	proposerKeyID string
	offeredKeyID  string
	consentDigest string

	acceptedActorID string
	acceptedKeyID   string
	contributions   map[string]contributionScope
}

type contributionScope struct {
	taskID  string
	actorID string
}

type handoffScope struct {
	taskID        string
	offeredActor  string
	offeredKey    string
	consentDigest string
}

type stateReducer struct {
	manifest         Manifest
	roster           map[string]Participant
	tasks            map[string]*reducerTask
	events           map[string]struct{}
	handoffs         map[string]handoffScope
	handoffDecisions map[string]struct{}
	contributions    map[string]contributionScope
}

// reduceVerified projects a cryptographically verified receipt sequence into
// its consent/task state. It is deliberately unexported so unsigned in-memory
// objects cannot be reduced through an authoritative-looking public API.
func reduceVerified(manifest Manifest, receipts []SignedReceipt) ([]TaskSummary, error) {
	if err := ValidateManifest(manifest); err != nil {
		return nil, fmt.Errorf("manifest: %w", err)
	}

	reducer := stateReducer{
		manifest:         manifest,
		roster:           make(map[string]Participant, len(manifest.Participants)),
		tasks:            make(map[string]*reducerTask),
		events:           make(map[string]struct{}, len(receipts)),
		handoffs:         make(map[string]handoffScope),
		handoffDecisions: make(map[string]struct{}),
		contributions:    make(map[string]contributionScope),
	}
	for _, participant := range manifest.Participants {
		reducer.roster[participant.ActorID] = participant
	}

	for index := range receipts {
		receipt := receipts[index]
		if err := reducer.reduceReceipt(receipt); err != nil {
			return nil, fmt.Errorf("receipts[%d] %s: %w", index, receipt.Event.Kind, err)
		}
		reducer.events[receipt.EventID] = struct{}{}
	}

	summaries := make([]TaskSummary, 0, len(reducer.tasks))
	for _, task := range reducer.tasks {
		summary := task.summary
		summary.ContributionEventIDs = append([]string(nil), summary.ContributionEventIDs...)
		sort.Strings(summary.ContributionEventIDs)
		if summary.ContributionEventIDs == nil {
			summary.ContributionEventIDs = []string{}
		}
		summaries = append(summaries, summary)
	}
	sort.Slice(summaries, func(left, right int) bool {
		return summaries[left].TaskID < summaries[right].TaskID
	})
	return summaries, nil
}

func (reducer *stateReducer) reduceReceipt(receipt SignedReceipt) error {
	if err := validateDigest("event_id", receipt.EventID); err != nil {
		return err
	}
	if _, exists := reducer.events[receipt.EventID]; exists {
		return fmt.Errorf("duplicate event_id %q", receipt.EventID)
	}
	if receipt.Event.CollaborationID != reducer.manifest.CollaborationID {
		return fmt.Errorf("event collaboration_id does not match the manifest")
	}
	participant, exists := reducer.roster[receipt.Event.ActorID]
	if !exists {
		return fmt.Errorf("event actor_id %q is not in the manifest roster", receipt.Event.ActorID)
	}
	if receipt.Event.ActorKeyID != participant.KeyID {
		return fmt.Errorf("event actor_key_id does not match actor %q in the manifest roster", receipt.Event.ActorID)
	}
	if receipt.Signature.Algorithm != AlgorithmEd25519 {
		return fmt.Errorf("signature algorithm must be %q", AlgorithmEd25519)
	}
	if receipt.Signature.KeyID != receipt.Event.ActorKeyID {
		return fmt.Errorf("signature key_id does not match event actor_key_id")
	}

	payload, _, err := DecodePayload(receipt.Event.Kind, receipt.Event.Payload)
	if err != nil {
		return err
	}
	switch typed := payload.(type) {
	case *TaskProposed:
		return reducer.taskProposed(receipt, *typed)
	case *TaskDecision:
		return reducer.taskDecision(receipt, *typed)
	case *ContributionSubmitted:
		return reducer.contributionSubmitted(receipt, *typed)
	case *CompletionClaimed:
		return reducer.completionClaimed(receipt, *typed)
	case *CompletionReviewed:
		return reducer.completionReviewed(receipt, *typed)
	case *HandoffOffered:
		return reducer.handoffOffered(receipt, *typed)
	case *ControlDeclared:
		return reducer.controlDeclared(receipt, *typed)
	default:
		return fmt.Errorf("unsupported decoded payload type %T", payload)
	}
}

func (reducer *stateReducer) taskProposed(receipt SignedReceipt, proposed TaskProposed) error {
	if err := validateLocalID("payload.task_id", proposed.TaskID); err != nil {
		return err
	}
	if _, exists := reducer.tasks[proposed.TaskID]; exists {
		return fmt.Errorf("task_id %q already exists", proposed.TaskID)
	}
	if err := validateText("payload.objective", proposed.Objective, 1, 4096); err != nil {
		return err
	}
	offered, exists := reducer.roster[proposed.OfferedToActorID]
	if !exists {
		return fmt.Errorf("payload.offered_to_actor_id %q is not in the manifest roster", proposed.OfferedToActorID)
	}
	if offered.KeyID != proposed.OfferedToActorKeyID {
		return fmt.Errorf("payload.offered_to_actor_key_id does not match the manifest roster")
	}
	if proposed.OfferedToActorID == receipt.Event.ActorID {
		return fmt.Errorf("a proposer cannot offer a task to itself")
	}
	if !proposed.AcceptanceRequired {
		return fmt.Errorf("payload.acceptance_required must be true")
	}
	if err := validateConsentTerms("payload.consent_terms", proposed.ConsentTerms); err != nil {
		return err
	}
	if err := validateConsentDigest("payload.consent_terms_sha256", proposed.ConsentTerms, proposed.ConsentTermsSHA256); err != nil {
		return err
	}
	if err := validateSortedTextSet("payload.acceptance_criteria", proposed.AcceptanceCriteria, maxTaskTextEntries, 1, 512); err != nil {
		return err
	}
	if err := validateSortedDigestSet("payload.required_artifact_sha256", proposed.RequiredArtifactSHA256, maxTaskDigests); err != nil {
		return err
	}

	if proposed.ParentTaskID != None {
		if err := validateLocalID("payload.parent_task_id", proposed.ParentTaskID); err != nil {
			return err
		}
		parent, exists := reducer.tasks[proposed.ParentTaskID]
		if !exists {
			return fmt.Errorf("parent task %q does not exist", proposed.ParentTaskID)
		}
		switch parent.summary.Status {
		case StatusActive, StatusClaimed, StatusContested:
		default:
			return fmt.Errorf("parent task %q is not in a live accepted scope", proposed.ParentTaskID)
		}
		if parent.acceptedActorID != receipt.Event.ActorID || parent.acceptedKeyID != receipt.Event.ActorKeyID {
			return fmt.Errorf("child task proposer has no live accepted scope on parent task %q", proposed.ParentTaskID)
		}
		if parent.summary.Participation != ParticipationActive {
			return fmt.Errorf("parent task %q participation is not active; resume or create a fresh root offer", proposed.ParentTaskID)
		}
	}

	reducer.tasks[proposed.TaskID] = &reducerTask{
		summary: TaskSummary{
			TaskID:                proposed.TaskID,
			ParentTaskID:          proposed.ParentTaskID,
			ProposerActorID:       receipt.Event.ActorID,
			OfferedToActorID:      proposed.OfferedToActorID,
			ActiveActorID:         None,
			Status:                StatusUnanswered,
			Participation:         ParticipationUnbound,
			OfferEventID:          receipt.EventID,
			AcceptanceEventID:     None,
			ContributionEventIDs:  []string{},
			LatestCompletionEvent: None,
		},
		proposerKeyID: receipt.Event.ActorKeyID,
		offeredKeyID:  proposed.OfferedToActorKeyID,
		consentDigest: proposed.ConsentTermsSHA256,
		contributions: make(map[string]contributionScope),
	}
	return nil
}

func (reducer *stateReducer) taskDecision(receipt SignedReceipt, decision TaskDecision) error {
	if err := validateLocalID("payload.task_id", decision.TaskID); err != nil {
		return err
	}
	if err := validateDigest("payload.offer_event_id", decision.OfferEventID); err != nil {
		return err
	}
	if err := validateDigest("payload.consent_terms_sha256", decision.ConsentTermsSHA256); err != nil {
		return err
	}
	if err := validateSortedTextSet("payload.reason_codes", decision.ReasonCodes, maxTaskTextEntries, 1, 128); err != nil {
		return err
	}
	if err := validateAffirmativeDecision(decision.Decision, decision.Affirmative); err != nil {
		return err
	}

	if handoff, exists := reducer.handoffs[decision.OfferEventID]; exists {
		if decision.TaskID != handoff.taskID {
			return fmt.Errorf("handoff offer and decision task_id differ")
		}
		if receipt.Event.ActorID != handoff.offeredActor || receipt.Event.ActorKeyID != handoff.offeredKey {
			return fmt.Errorf("only the actor and key addressed by the handoff offer may decide it")
		}
		if decision.ConsentTermsSHA256 != handoff.consentDigest {
			return fmt.Errorf("handoff decision consent_terms_sha256 does not match the offer")
		}
		if _, decided := reducer.handoffDecisions[decision.OfferEventID]; decided {
			return fmt.Errorf("handoff offer has already received a decision")
		}
		if decision.Decision == DecisionAccept {
			return fmt.Errorf("handoff acceptance is not supported in v0; create a separately consented child task")
		}
		reducer.handoffDecisions[decision.OfferEventID] = struct{}{}
		return nil
	}

	task, exists := reducer.tasks[decision.TaskID]
	if !exists {
		return fmt.Errorf("task %q does not exist", decision.TaskID)
	}
	if task.summary.Status != StatusUnanswered {
		return fmt.Errorf("task %q is no longer unanswered", decision.TaskID)
	}
	if decision.OfferEventID != task.summary.OfferEventID {
		return fmt.Errorf("payload.offer_event_id does not match the task offer")
	}
	if decision.ConsentTermsSHA256 != task.consentDigest {
		return fmt.Errorf("payload.consent_terms_sha256 does not match the task offer")
	}
	if receipt.Event.ActorID != task.summary.OfferedToActorID || receipt.Event.ActorKeyID != task.offeredKeyID {
		return fmt.Errorf("only the actor and key addressed by the task offer may decide it")
	}

	if decision.Decision == DecisionRefuse {
		task.summary.Status = StatusRefused
		return nil
	}
	task.acceptedActorID = receipt.Event.ActorID
	task.acceptedKeyID = receipt.Event.ActorKeyID
	task.summary.ActiveActorID = receipt.Event.ActorID
	task.summary.AcceptanceEventID = receipt.EventID
	task.summary.Status = StatusActive
	task.summary.Participation = ParticipationActive
	return nil
}

func (reducer *stateReducer) contributionSubmitted(receipt SignedReceipt, contribution ContributionSubmitted) error {
	task, err := reducer.taskForActiveActor(receipt, contribution.TaskID, contribution.AcceptanceEventID, "contribution")
	if err != nil {
		return err
	}
	if task.summary.Status != StatusActive && task.summary.Status != StatusContested {
		return fmt.Errorf("task %q does not accept contributions in status %q", contribution.TaskID, task.summary.Status)
	}
	if err := requireParticipationActive(task, "contribution"); err != nil {
		return err
	}
	if err := validateText("payload.summary", contribution.Summary, 1, 4096); err != nil {
		return err
	}
	if err := validateSortedDigestSet("payload.artifact_sha256", contribution.ArtifactSHA256, maxTaskDigests); err != nil {
		return err
	}
	if err := validateSortedDigestSet("payload.evidence_sha256", contribution.EvidenceSHA256, maxTaskDigests); err != nil {
		return err
	}
	if len(contribution.ArtifactSHA256)+len(contribution.EvidenceSHA256) == 0 {
		return fmt.Errorf("a contribution must reference at least one artifact or evidence digest")
	}
	if err := validateSortedTextSet("payload.limitation_codes", contribution.LimitationCodes, maxTaskTextEntries, 1, 128); err != nil {
		return err
	}

	scope := contributionScope{taskID: contribution.TaskID, actorID: receipt.Event.ActorID}
	if _, exists := reducer.contributions[receipt.EventID]; exists {
		return fmt.Errorf("contribution event_id %q already exists", receipt.EventID)
	}
	task.contributions[receipt.EventID] = scope
	reducer.contributions[receipt.EventID] = scope
	task.summary.ContributionEventIDs = append(task.summary.ContributionEventIDs, receipt.EventID)
	return nil
}

func (reducer *stateReducer) completionClaimed(receipt SignedReceipt, claimed CompletionClaimed) error {
	task, err := reducer.taskForActiveActor(receipt, claimed.TaskID, claimed.AcceptanceEventID, "completion claim")
	if err != nil {
		return err
	}
	if task.summary.Status != StatusActive && task.summary.Status != StatusContested {
		return fmt.Errorf("task %q does not accept completion claims in status %q", claimed.TaskID, task.summary.Status)
	}
	if err := requireParticipationActive(task, "completion claim"); err != nil {
		return err
	}
	if err := validateSortedDigestSet("payload.contribution_event_ids", claimed.ContributionEventIDs, maxTaskDigests); err != nil {
		return err
	}
	if len(claimed.ContributionEventIDs) == 0 {
		return fmt.Errorf("payload.contribution_event_ids must not be empty")
	}
	for _, contributionID := range claimed.ContributionEventIDs {
		scope, exists := reducer.contributions[contributionID]
		if !exists {
			return fmt.Errorf("contribution_event_id %q does not exist", contributionID)
		}
		if scope.taskID != claimed.TaskID || scope.actorID != receipt.Event.ActorID {
			return fmt.Errorf("contribution_event_id %q is outside this task and actor scope", contributionID)
		}
	}
	if err := validateSortedDigestSet("payload.deliverable_sha256", claimed.DeliverableSHA256, maxTaskDigests); err != nil {
		return err
	}
	if len(claimed.DeliverableSHA256) == 0 {
		return fmt.Errorf("payload.deliverable_sha256 must not be empty")
	}
	if err := validateSortedTextSet("payload.limitation_codes", claimed.LimitationCodes, maxTaskTextEntries, 1, 128); err != nil {
		return err
	}

	task.summary.LatestCompletionEvent = receipt.EventID
	task.summary.Status = StatusClaimed
	return nil
}

func (reducer *stateReducer) completionReviewed(receipt SignedReceipt, reviewed CompletionReviewed) error {
	if err := validateLocalID("payload.task_id", reviewed.TaskID); err != nil {
		return err
	}
	if err := validateDigest("payload.completion_event_id", reviewed.CompletionEventID); err != nil {
		return err
	}
	if err := validateSortedTextSet("payload.reason_codes", reviewed.ReasonCodes, maxTaskTextEntries, 1, 128); err != nil {
		return err
	}
	if err := validateSortedDigestSet("payload.evidence_sha256", reviewed.EvidenceSHA256, maxTaskDigests); err != nil {
		return err
	}
	task, exists := reducer.tasks[reviewed.TaskID]
	if !exists {
		return fmt.Errorf("task %q does not exist", reviewed.TaskID)
	}
	if reviewed.CompletionEventID != task.summary.LatestCompletionEvent {
		return fmt.Errorf("payload.completion_event_id is not the latest completion claim")
	}

	switch reviewed.Decision {
	case ReviewAccept:
		if task.summary.Status != StatusClaimed {
			return fmt.Errorf("task %q has no unreviewed latest completion claim", reviewed.TaskID)
		}
		if receipt.Event.ActorID != task.summary.ProposerActorID || receipt.Event.ActorKeyID != task.proposerKeyID {
			return fmt.Errorf("only the original proposer may protocol-accept a completion claim")
		}
		task.summary.Status = StatusAccepted
	case ReviewDispute:
		if task.summary.Status != StatusClaimed && task.summary.Status != StatusAccepted {
			return fmt.Errorf("task %q has no current completion claim to dispute", reviewed.TaskID)
		}
		isProposer := receipt.Event.ActorID == task.summary.ProposerActorID && receipt.Event.ActorKeyID == task.proposerKeyID
		isParticipant := receipt.Event.ActorID == task.acceptedActorID && receipt.Event.ActorKeyID == task.acceptedKeyID
		if !isProposer && !isParticipant {
			return fmt.Errorf("only the proposer or participant with accepted scope may dispute a completion claim")
		}
		task.summary.Status = StatusContested
	default:
		return fmt.Errorf("payload.decision must be %q or %q", ReviewAccept, ReviewDispute)
	}
	return nil
}

func (reducer *stateReducer) handoffOffered(receipt SignedReceipt, offered HandoffOffered) error {
	task, err := reducer.taskForActiveActor(receipt, offered.TaskID, offered.AcceptanceEventID, "handoff offer")
	if err != nil {
		return err
	}
	if task.summary.Status != StatusActive && task.summary.Status != StatusContested {
		return fmt.Errorf("task %q does not accept handoff offers in status %q", offered.TaskID, task.summary.Status)
	}
	if err := requireParticipationActive(task, "handoff offer"); err != nil {
		return err
	}
	target, exists := reducer.roster[offered.OfferedToActorID]
	if !exists {
		return fmt.Errorf("payload.offered_to_actor_id %q is not in the manifest roster", offered.OfferedToActorID)
	}
	if target.KeyID != offered.OfferedToActorKeyID {
		return fmt.Errorf("payload.offered_to_actor_key_id does not match the manifest roster")
	}
	if offered.OfferedToActorID == receipt.Event.ActorID {
		return fmt.Errorf("a participant cannot offer a handoff to itself")
	}
	if !offered.AcceptanceRequired {
		return fmt.Errorf("payload.acceptance_required must be true")
	}
	if err := validateConsentTerms("payload.consent_terms", offered.ConsentTerms); err != nil {
		return err
	}
	if err := validateConsentDigest("payload.consent_terms_sha256", offered.ConsentTerms, offered.ConsentTermsSHA256); err != nil {
		return err
	}
	if err := validateSortedDigestSet("payload.context_artifact_sha256", offered.ContextArtifactSHA256, maxTaskDigests); err != nil {
		return err
	}
	reducer.handoffs[receipt.EventID] = handoffScope{
		taskID:        offered.TaskID,
		offeredActor:  offered.OfferedToActorID,
		offeredKey:    offered.OfferedToActorKeyID,
		consentDigest: offered.ConsentTermsSHA256,
	}
	return nil
}

func (reducer *stateReducer) controlDeclared(receipt SignedReceipt, control ControlDeclared) error {
	task, err := reducer.taskForActiveActor(receipt, control.TaskID, control.AcceptanceEventID, "control declaration")
	if err != nil {
		return err
	}
	if task.summary.Participation == ParticipationEnded {
		return fmt.Errorf("task %q participation has ended; a fresh offer and acceptance are required", control.TaskID)
	}
	if err := validateSortedTextSet("payload.reason_codes", control.ReasonCodes, maxTaskTextEntries, 1, 128); err != nil {
		return err
	}
	if err := validateSortedDigestSet("payload.export_event_ids", control.ExportEventIDs, maxTaskDigests); err != nil {
		return err
	}
	for _, eventID := range control.ExportEventIDs {
		if _, exists := reducer.events[eventID]; !exists {
			return fmt.Errorf("export_event_id %q does not refer to an earlier event", eventID)
		}
	}

	switch control.Action {
	case ControlPause:
		if task.summary.Participation != ParticipationActive {
			return fmt.Errorf("PAUSE requires participation %q", ParticipationActive)
		}
		// Rest is a declaration, not a terminal task outcome or penalty.
		task.summary.Participation = ParticipationPauseDeclared
		return nil
	case ControlResume:
		if task.summary.Participation != ParticipationPauseDeclared {
			return fmt.Errorf("RESUME requires participation %q", ParticipationPauseDeclared)
		}
		task.summary.Participation = ParticipationActive
		return nil
	case ControlStop, ControlExit:
		task.summary.Participation = ParticipationEnded
		task.summary.ActiveActorID = None
		return nil
	default:
		return fmt.Errorf("payload.action must be PAUSE, RESUME, STOP, or EXIT")
	}
}

func requireParticipationActive(task *reducerTask, operation string) error {
	if task.summary.Participation != ParticipationActive {
		return fmt.Errorf("task %q participation is %q; cannot make a %s until it is active under this acceptance", task.summary.TaskID, task.summary.Participation, operation)
	}
	return nil
}

func (reducer *stateReducer) taskForActiveActor(receipt SignedReceipt, taskID, acceptanceEventID, operation string) (*reducerTask, error) {
	if err := validateLocalID("payload.task_id", taskID); err != nil {
		return nil, err
	}
	if err := validateDigest("payload.acceptance_event_id", acceptanceEventID); err != nil {
		return nil, err
	}
	task, exists := reducer.tasks[taskID]
	if !exists {
		return nil, fmt.Errorf("task %q does not exist", taskID)
	}
	if task.summary.AcceptanceEventID == None || acceptanceEventID != task.summary.AcceptanceEventID {
		return nil, fmt.Errorf("%s acceptance_event_id does not match the accepted task offer", operation)
	}
	if receipt.Event.ActorID != task.acceptedActorID || receipt.Event.ActorKeyID != task.acceptedKeyID {
		return nil, fmt.Errorf("only the actor and key holding the accepted scope may make a %s", operation)
	}
	return task, nil
}

func validateConsentTerms(path string, terms ConsentTerms) error {
	for _, field := range []struct {
		name  string
		value string
		max   int
	}{
		{"role", terms.Role, 256},
		{"artifact", terms.Artifact, 2048},
		{"purpose", terms.Purpose, 2048},
		{"term", terms.Term, 512},
		{"workload_cap", terms.WorkloadCap, 512},
	} {
		if err := validateText(path+"."+field.name, field.value, 1, field.max); err != nil {
			return err
		}
	}
	if terms.DisclosureLane != DisclosureLocal {
		return fmt.Errorf("%s.disclosure_lane must be %q", path, DisclosureLocal)
	}
	if terms.CompensationPolicy != None {
		return fmt.Errorf("%s.compensation_policy must be %q", path, None)
	}
	if terms.CreditRule != CreditAppendOnly {
		return fmt.Errorf("%s.credit_rule must be %q", path, CreditAppendOnly)
	}
	return nil
}

func validateConsentDigest(path string, terms ConsentTerms, claimed string) error {
	if err := validateDigest(path, claimed); err != nil {
		return err
	}
	want, err := ConsentTermsDigest(terms)
	if err != nil {
		return fmt.Errorf("compute consent terms digest: %w", err)
	}
	if claimed != want {
		return fmt.Errorf("%s does not match the exact consent terms", path)
	}
	return nil
}

func validateAffirmativeDecision(decision string, affirmative bool) error {
	switch decision {
	case DecisionAccept:
		if !affirmative {
			return fmt.Errorf("ACCEPT requires affirmative_acceptance true")
		}
	case DecisionRefuse:
		if affirmative {
			return fmt.Errorf("REFUSE requires affirmative_acceptance false")
		}
	default:
		return fmt.Errorf("payload.decision must be %q or %q", DecisionAccept, DecisionRefuse)
	}
	return nil
}

func validateSortedTextSet(path string, values []string, maximum, minimumBytes, maximumBytes int) error {
	return validateSortedUnique(path, values, maximum, func(itemPath, value string) error {
		return validateText(itemPath, value, minimumBytes, maximumBytes)
	})
}

func validateSortedDigestSet(path string, values []string, maximum int) error {
	return validateSortedUnique(path, values, maximum, validateDigest)
}
