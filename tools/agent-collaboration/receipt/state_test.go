package receipt

import (
	"bytes"
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"sort"
	"strings"
	"testing"
)

type reducerTestContext struct {
	manifest Manifest
	alpha    Participant
	beta     Participant
	gamma    Participant
}

func TestReduceAlphaBetaHappyPath(t *testing.T) {
	context := newReducerTestContext(t)
	terms := reducerTestTerms()
	offer := reducerTestProposal(t, "root", None, context.beta, terms)
	proposed := reducerTestReceipt(t, context, "propose", EventTaskProposed, context.alpha, offer)
	accepted := reducerTestReceipt(t, context, "accept", EventTaskDecision, context.beta, TaskDecision{
		TaskID:             "root",
		OfferEventID:       proposed.EventID,
		Decision:           DecisionAccept,
		Affirmative:        true,
		ConsentTermsSHA256: offer.ConsentTermsSHA256,
		ReasonCodes:        []string{},
	})
	contributed := reducerTestReceipt(t, context, "contribute", EventContribution, context.beta, ContributionSubmitted{
		TaskID:            "root",
		AcceptanceEventID: accepted.EventID,
		Summary:           "A bounded local artifact was produced.",
		ArtifactSHA256:    []string{reducerTestDigest("artifact")},
		EvidenceSHA256:    []string{},
		LimitationCodes:   []string{},
	})
	claimed := reducerTestReceipt(t, context, "claim", EventCompletionClaimed, context.beta, CompletionClaimed{
		TaskID:               "root",
		AcceptanceEventID:    accepted.EventID,
		ContributionEventIDs: []string{contributed.EventID},
		DeliverableSHA256:    []string{reducerTestDigest("deliverable")},
		LimitationCodes:      []string{},
	})
	reviewed := reducerTestReceipt(t, context, "review", EventCompletionReview, context.alpha, CompletionReviewed{
		TaskID:            "root",
		CompletionEventID: claimed.EventID,
		Decision:          ReviewAccept,
		ReasonCodes:       []string{},
		EvidenceSHA256:    []string{},
	})

	summaries := reducerTestReduceOK(t, context.manifest, proposed, accepted, contributed, claimed, reviewed)
	if len(summaries) != 1 {
		t.Fatalf("got %d summaries, want 1", len(summaries))
	}
	want := TaskSummary{
		TaskID:                "root",
		ParentTaskID:          None,
		ProposerActorID:       context.alpha.ActorID,
		OfferedToActorID:      context.beta.ActorID,
		ActiveActorID:         context.beta.ActorID,
		Status:                StatusAccepted,
		Participation:         ParticipationActive,
		OfferEventID:          proposed.EventID,
		AcceptanceEventID:     accepted.EventID,
		ContributionEventIDs:  []string{contributed.EventID},
		LatestCompletionEvent: claimed.EventID,
	}
	reducerTestEqualSummary(t, summaries[0], want)
}

func TestReduceLeavesUnansweredOfferUnanswered(t *testing.T) {
	context := newReducerTestContext(t)
	proposal := reducerTestProposal(t, "unanswered", None, context.beta, reducerTestTerms())
	proposed := reducerTestReceipt(t, context, "unanswered-proposal", EventTaskProposed, context.alpha, proposal)

	summaries := reducerTestReduceOK(t, context.manifest, proposed)
	if summaries[0].Status != StatusUnanswered || summaries[0].Participation != ParticipationUnbound || summaries[0].ActiveActorID != None || summaries[0].AcceptanceEventID != None {
		t.Fatalf("unanswered offer gained implied consent: %#v", summaries[0])
	}
}

func TestReduceReasonlessRefusalHasNoPenaltyOrAcceptance(t *testing.T) {
	context := newReducerTestContext(t)
	proposal := reducerTestProposal(t, "refused", None, context.beta, reducerTestTerms())
	proposed := reducerTestReceipt(t, context, "refused-proposal", EventTaskProposed, context.alpha, proposal)
	refused := reducerTestReceipt(t, context, "reasonless-refusal", EventTaskDecision, context.beta, TaskDecision{
		TaskID:             "refused",
		OfferEventID:       proposed.EventID,
		Decision:           DecisionRefuse,
		Affirmative:        false,
		ConsentTermsSHA256: proposal.ConsentTermsSHA256,
		ReasonCodes:        []string{},
	})

	summary := reducerTestReduceOK(t, context.manifest, proposed, refused)[0]
	if summary.Status != StatusRefused || summary.ActiveActorID != None || summary.AcceptanceEventID != None {
		t.Fatalf("refusal changed rights beyond recording refusal: %#v", summary)
	}
}

func TestReduceRejectsWrongDecisionIdentityKeyAndTerms(t *testing.T) {
	context := newReducerTestContext(t)
	proposal := reducerTestProposal(t, "identity", None, context.beta, reducerTestTerms())
	proposed := reducerTestReceipt(t, context, "identity-proposal", EventTaskProposed, context.alpha, proposal)
	decision := TaskDecision{
		TaskID:             "identity",
		OfferEventID:       proposed.EventID,
		Decision:           DecisionAccept,
		Affirmative:        true,
		ConsentTermsSHA256: proposal.ConsentTermsSHA256,
		ReasonCodes:        []string{},
	}

	t.Run("wrong actor", func(t *testing.T) {
		wrong := reducerTestReceipt(t, context, "wrong-actor", EventTaskDecision, context.alpha, decision)
		reducerTestReduceError(t, context.manifest, "only the actor and key addressed", proposed, wrong)
	})
	t.Run("wrong roster key", func(t *testing.T) {
		wrong := reducerTestReceipt(t, context, "wrong-key", EventTaskDecision, context.beta, decision)
		wrong.Event.ActorKeyID = context.alpha.KeyID
		wrong.Signature.KeyID = context.alpha.KeyID
		reducerTestReduceError(t, context.manifest, "does not match actor", proposed, wrong)
	})
	t.Run("wrong terms digest", func(t *testing.T) {
		wrongDecision := decision
		wrongDecision.ConsentTermsSHA256 = reducerTestDigest("different terms")
		wrong := reducerTestReceipt(t, context, "wrong-terms", EventTaskDecision, context.beta, wrongDecision)
		reducerTestReduceError(t, context.manifest, "does not match the task offer", proposed, wrong)
	})
	t.Run("accept must be affirmative", func(t *testing.T) {
		contradictory := decision
		contradictory.Affirmative = false
		wrong := reducerTestReceipt(t, context, "not-affirmative", EventTaskDecision, context.beta, contradictory)
		reducerTestReduceError(t, context.manifest, "requires affirmative_acceptance true", proposed, wrong)
	})
}

func TestReduceRejectsNonLocalOrMutableConsentTerms(t *testing.T) {
	context := newReducerTestContext(t)
	for _, test := range []struct {
		name   string
		mutate func(*ConsentTerms)
		want   string
	}{
		{"disclosure", func(terms *ConsentTerms) { terms.DisclosureLane = "PUBLIC" }, "disclosure_lane"},
		{"compensation", func(terms *ConsentTerms) { terms.CompensationPolicy = "FIAT" }, "compensation_policy"},
		{"credit", func(terms *ConsentTerms) { terms.CreditRule = "REPLACEABLE" }, "credit_rule"},
	} {
		t.Run(test.name, func(t *testing.T) {
			proposal := reducerTestProposal(t, "bad-terms", None, context.beta, reducerTestTerms())
			test.mutate(&proposal.ConsentTerms)
			canonical, err := canonicalJSON(proposal.ConsentTerms)
			if err != nil {
				t.Fatal(err)
			}
			proposal.ConsentTermsSHA256 = digestText(domainDigest(consentDomain, canonical))
			receipt := reducerTestReceipt(t, context, "bad-terms-"+test.name, EventTaskProposed, context.alpha, proposal)
			reducerTestReduceError(t, context.manifest, test.want, receipt)
		})
	}
}

func TestReduceRejectsWorkBeforeAcceptanceAndAfterStop(t *testing.T) {
	context := newReducerTestContext(t)
	proposal := reducerTestProposal(t, "work", None, context.beta, reducerTestTerms())
	proposed := reducerTestReceipt(t, context, "work-proposal", EventTaskProposed, context.alpha, proposal)
	artifact := []string{reducerTestDigest("work-artifact")}
	before := reducerTestReceipt(t, context, "work-before-accept", EventContribution, context.beta, ContributionSubmitted{
		TaskID:            "work",
		AcceptanceEventID: reducerTestDigest("imaginary-acceptance"),
		Summary:           "Not consented.",
		ArtifactSHA256:    artifact,
		EvidenceSHA256:    []string{},
		LimitationCodes:   []string{},
	})
	reducerTestReduceError(t, context.manifest, "does not match the accepted task offer", proposed, before)

	accepted := reducerTestAccept(t, context, "work", proposal, proposed, "work-accept")
	paused := reducerTestReceipt(t, context, "work-pause", EventControl, context.beta, ControlDeclared{
		TaskID:            "work",
		AcceptanceEventID: accepted.EventID,
		Action:            ControlPause,
		ReasonCodes:       []string{},
		ExportEventIDs:    []string{},
	})
	pausedSummary := reducerTestReduceOK(t, context.manifest, proposed, accepted, paused)[0]
	if pausedSummary.Status != StatusActive || pausedSummary.Participation != ParticipationPauseDeclared {
		t.Fatalf("pause changed task outcome or failed to remain declarative: %#v", pausedSummary)
	}
	stopped := reducerTestReceipt(t, context, "work-stop", EventControl, context.beta, ControlDeclared{
		TaskID:            "work",
		AcceptanceEventID: accepted.EventID,
		Action:            ControlStop,
		ReasonCodes:       []string{},
		ExportEventIDs:    []string{},
	})
	after := reducerTestReceipt(t, context, "work-after-stop", EventContribution, context.beta, ContributionSubmitted{
		TaskID:            "work",
		AcceptanceEventID: accepted.EventID,
		Summary:           "Must be rejected after stop.",
		ArtifactSHA256:    artifact,
		EvidenceSHA256:    []string{},
		LimitationCodes:   []string{},
	})
	reducerTestReduceError(t, context.manifest, "participation is \"ENDED\"", proposed, accepted, stopped, after)
	summary := reducerTestReduceOK(t, context.manifest, proposed, accepted, stopped)[0]
	if summary.Status != StatusActive || summary.Participation != ParticipationEnded || summary.ActiveActorID != None {
		t.Fatalf("stop projection = %#v", summary)
	}
}

func TestReducePauseGatesWorkAndResumeRestoresAcceptedScope(t *testing.T) {
	context := newReducerTestContext(t)
	proposal := reducerTestProposal(t, "rest", None, context.beta, reducerTestTerms())
	proposed := reducerTestReceipt(t, context, "rest-proposal", EventTaskProposed, context.alpha, proposal)
	accepted := reducerTestAccept(t, context, "rest", proposal, proposed, "rest-acceptance")
	paused := reducerTestReceipt(t, context, "rest-pause", EventControl, context.beta, ControlDeclared{
		TaskID:            "rest",
		AcceptanceEventID: accepted.EventID,
		Action:            ControlPause,
		ReasonCodes:       []string{},
		ExportEventIDs:    []string{},
	})

	contribution := reducerTestReceipt(t, context, "rest-contribution", EventContribution, context.beta, ContributionSubmitted{
		TaskID:            "rest",
		AcceptanceEventID: accepted.EventID,
		Summary:           "Work waits for an explicit resume.",
		ArtifactSHA256:    []string{reducerTestDigest("rest-artifact")},
		EvidenceSHA256:    []string{},
		LimitationCodes:   []string{},
	})
	claim := reducerTestReceipt(t, context, "rest-claim", EventCompletionClaimed, context.beta, CompletionClaimed{
		TaskID:               "rest",
		AcceptanceEventID:    accepted.EventID,
		ContributionEventIDs: []string{reducerTestDigest("rest-imaginary-contribution")},
		DeliverableSHA256:    []string{reducerTestDigest("rest-deliverable")},
		LimitationCodes:      []string{},
	})
	handoffTerms := reducerTestTerms()
	handoffDigest, err := ConsentTermsDigest(handoffTerms)
	if err != nil {
		t.Fatal(err)
	}
	handoff := reducerTestReceipt(t, context, "rest-handoff", EventHandoffOffered, context.beta, HandoffOffered{
		TaskID:                "rest",
		AcceptanceEventID:     accepted.EventID,
		OfferedToActorID:      context.gamma.ActorID,
		OfferedToActorKeyID:   context.gamma.KeyID,
		AcceptanceRequired:    true,
		ConsentTerms:          handoffTerms,
		ConsentTermsSHA256:    handoffDigest,
		ContextArtifactSHA256: []string{},
	})
	child := reducerTestReceipt(t, context, "rest-child", EventTaskProposed, context.beta,
		reducerTestProposal(t, "rest-child", "rest", context.gamma, reducerTestTerms()))

	for name, candidate := range map[string]SignedReceipt{
		"contribution": contribution,
		"claim":        claim,
		"handoff":      handoff,
		"child":        child,
	} {
		t.Run("paused "+name, func(t *testing.T) {
			reducerTestReduceError(t, context.manifest, "participation", proposed, accepted, paused, candidate)
		})
	}

	resumed := reducerTestReceipt(t, context, "rest-resume", EventControl, context.beta, ControlDeclared{
		TaskID:            "rest",
		AcceptanceEventID: accepted.EventID,
		Action:            ControlResume,
		ReasonCodes:       []string{},
		ExportEventIDs:    []string{},
	})
	summary := reducerTestReduceOK(t, context.manifest, proposed, accepted, paused, resumed, contribution)[0]
	if summary.Participation != ParticipationActive || len(summary.ContributionEventIDs) != 1 {
		t.Fatalf("resume did not restore the accepted work scope: %#v", summary)
	}
}

func TestReduceExitAfterClaimPreservesOutcomeReviewAndLateDispute(t *testing.T) {
	context := newReducerTestContext(t)
	proposal := reducerTestProposal(t, "future-only-exit", None, context.beta, reducerTestTerms())
	proposed := reducerTestReceipt(t, context, "future-only-exit-proposal", EventTaskProposed, context.alpha, proposal)
	accepted := reducerTestAccept(t, context, "future-only-exit", proposal, proposed, "future-only-exit-acceptance")
	contribution := reducerTestReceipt(t, context, "future-only-exit-contribution", EventContribution, context.beta, ContributionSubmitted{
		TaskID:            "future-only-exit",
		AcceptanceEventID: accepted.EventID,
		Summary:           "One bounded contribution before exit.",
		ArtifactSHA256:    []string{reducerTestDigest("future-only-exit-artifact")},
		EvidenceSHA256:    []string{},
		LimitationCodes:   []string{},
	})
	claimed := reducerTestReceipt(t, context, "future-only-exit-claim", EventCompletionClaimed, context.beta, CompletionClaimed{
		TaskID:               "future-only-exit",
		AcceptanceEventID:    accepted.EventID,
		ContributionEventIDs: []string{contribution.EventID},
		DeliverableSHA256:    []string{reducerTestDigest("future-only-exit-deliverable")},
		LimitationCodes:      []string{},
	})
	exited := reducerTestReceipt(t, context, "future-only-exit-control", EventControl, context.beta, ControlDeclared{
		TaskID:            "future-only-exit",
		AcceptanceEventID: accepted.EventID,
		Action:            ControlExit,
		ReasonCodes:       []string{},
		ExportEventIDs:    []string{},
	})
	exitedSummary := reducerTestReduceOK(t, context.manifest, proposed, accepted, contribution, claimed, exited)[0]
	if exitedSummary.Status != StatusClaimed || exitedSummary.Participation != ParticipationEnded || exitedSummary.ActiveActorID != None {
		t.Fatalf("exit erased or rewrote the completion claim: %#v", exitedSummary)
	}

	reviewed := reducerTestReceipt(t, context, "future-only-exit-review", EventCompletionReview, context.alpha, CompletionReviewed{
		TaskID:            "future-only-exit",
		CompletionEventID: claimed.EventID,
		Decision:          ReviewAccept,
		ReasonCodes:       []string{},
		EvidenceSHA256:    []string{},
	})
	reviewedSummary := reducerTestReduceOK(t, context.manifest, proposed, accepted, contribution, claimed, exited, reviewed)[0]
	if reviewedSummary.Status != StatusAccepted || reviewedSummary.Participation != ParticipationEnded || reviewedSummary.ActiveActorID != None {
		t.Fatalf("review reopened ended participation: %#v", reviewedSummary)
	}

	disputed := reducerTestReceipt(t, context, "future-only-exit-late-dispute", EventCompletionReview, context.beta, CompletionReviewed{
		TaskID:            "future-only-exit",
		CompletionEventID: claimed.EventID,
		Decision:          ReviewDispute,
		ReasonCodes:       []string{},
		EvidenceSHA256:    []string{},
	})
	history := []SignedReceipt{proposed, accepted, contribution, claimed, exited, reviewed, disputed}
	disputedSummary := reducerTestReduceOK(t, context.manifest, history...)[0]
	if disputedSummary.Status != StatusContested || disputedSummary.Participation != ParticipationEnded || disputedSummary.ActiveActorID != None {
		t.Fatalf("late dispute was lost or reopened participation: %#v", disputedSummary)
	}

	blockedContribution := reducerTestReceipt(t, context, "future-only-exit-blocked-contribution", EventContribution, context.beta, ContributionSubmitted{
		TaskID:            "future-only-exit",
		AcceptanceEventID: accepted.EventID,
		Summary:           "Must require fresh consent.",
		ArtifactSHA256:    []string{reducerTestDigest("future-only-exit-blocked-artifact")},
		EvidenceSHA256:    []string{},
		LimitationCodes:   []string{},
	})
	blockedClaim := reducerTestReceipt(t, context, "future-only-exit-blocked-claim", EventCompletionClaimed, context.beta, CompletionClaimed{
		TaskID:               "future-only-exit",
		AcceptanceEventID:    accepted.EventID,
		ContributionEventIDs: []string{contribution.EventID},
		DeliverableSHA256:    []string{reducerTestDigest("future-only-exit-blocked-deliverable")},
		LimitationCodes:      []string{},
	})
	handoffTerms := reducerTestTerms()
	handoffDigest, err := ConsentTermsDigest(handoffTerms)
	if err != nil {
		t.Fatal(err)
	}
	blockedHandoff := reducerTestReceipt(t, context, "future-only-exit-blocked-handoff", EventHandoffOffered, context.beta, HandoffOffered{
		TaskID:                "future-only-exit",
		AcceptanceEventID:     accepted.EventID,
		OfferedToActorID:      context.gamma.ActorID,
		OfferedToActorKeyID:   context.gamma.KeyID,
		AcceptanceRequired:    true,
		ConsentTerms:          handoffTerms,
		ConsentTermsSHA256:    handoffDigest,
		ContextArtifactSHA256: []string{},
	})
	blockedChild := reducerTestReceipt(t, context, "future-only-exit-blocked-child", EventTaskProposed, context.beta,
		reducerTestProposal(t, "future-only-exit-child", "future-only-exit", context.gamma, reducerTestTerms()))
	for name, candidate := range map[string]SignedReceipt{
		"contribution": blockedContribution,
		"claim":        blockedClaim,
		"handoff":      blockedHandoff,
		"child":        blockedChild,
	} {
		t.Run("ended "+name, func(t *testing.T) {
			reducerTestReduceError(t, context.manifest, "participation", append(append([]SignedReceipt(nil), history...), candidate)...)
		})
	}
}

func TestReduceDisputeIsEpistemicAndWorkMayContinue(t *testing.T) {
	context := newReducerTestContext(t)
	proposal := reducerTestProposal(t, "dispute", None, context.beta, reducerTestTerms())
	proposed := reducerTestReceipt(t, context, "dispute-proposal", EventTaskProposed, context.alpha, proposal)
	accepted := reducerTestAccept(t, context, "dispute", proposal, proposed, "dispute-accept")
	contributed := reducerTestReceipt(t, context, "dispute-contribution", EventContribution, context.beta, ContributionSubmitted{
		TaskID:            "dispute",
		AcceptanceEventID: accepted.EventID,
		Summary:           "Initial contribution.",
		ArtifactSHA256:    []string{reducerTestDigest("initial-artifact")},
		EvidenceSHA256:    []string{},
		LimitationCodes:   []string{},
	})
	claimed := reducerTestReceipt(t, context, "dispute-claim", EventCompletionClaimed, context.beta, CompletionClaimed{
		TaskID:               "dispute",
		AcceptanceEventID:    accepted.EventID,
		ContributionEventIDs: []string{contributed.EventID},
		DeliverableSHA256:    []string{reducerTestDigest("initial-deliverable")},
		LimitationCodes:      []string{},
	})
	disputed := reducerTestReceipt(t, context, "participant-dispute", EventCompletionReview, context.beta, CompletionReviewed{
		TaskID:            "dispute",
		CompletionEventID: claimed.EventID,
		Decision:          ReviewDispute,
		ReasonCodes:       []string{},
		EvidenceSHA256:    []string{},
	})
	followup := reducerTestReceipt(t, context, "post-dispute-contribution", EventContribution, context.beta, ContributionSubmitted{
		TaskID:            "dispute",
		AcceptanceEventID: accepted.EventID,
		Summary:           "A correction after dispute.",
		ArtifactSHA256:    []string{},
		EvidenceSHA256:    []string{reducerTestDigest("correction-evidence")},
		LimitationCodes:   []string{},
	})

	summary := reducerTestReduceOK(t, context.manifest, proposed, accepted, contributed, claimed, disputed, followup)[0]
	if summary.Status != StatusContested {
		t.Fatalf("dispute status = %q, want %q", summary.Status, StatusContested)
	}
	if len(summary.ContributionEventIDs) != 2 {
		t.Fatalf("post-dispute contribution was not retained: %#v", summary.ContributionEventIDs)
	}

	wrongAccept := reducerTestReceipt(t, context, "participant-accept", EventCompletionReview, context.beta, CompletionReviewed{
		TaskID:            "dispute",
		CompletionEventID: claimed.EventID,
		Decision:          ReviewAccept,
		ReasonCodes:       []string{},
		EvidenceSHA256:    []string{},
	})
	reducerTestReduceError(t, context.manifest, "only the original proposer", proposed, accepted, contributed, claimed, wrongAccept)
}

func TestReduceParticipantDisputeDominatesEarlierProposerAcceptance(t *testing.T) {
	context := newReducerTestContext(t)
	proposal := reducerTestProposal(t, "late-dispute", None, context.beta, reducerTestTerms())
	proposed := reducerTestReceipt(t, context, "late-dispute-proposal", EventTaskProposed, context.alpha, proposal)
	accepted := reducerTestAccept(t, context, "late-dispute", proposal, proposed, "late-dispute-acceptance")
	contributed := reducerTestReceipt(t, context, "late-dispute-contribution", EventContribution, context.beta, ContributionSubmitted{
		TaskID:            "late-dispute",
		AcceptanceEventID: accepted.EventID,
		Summary:           "Contribution later disputed by its own signer.",
		ArtifactSHA256:    []string{reducerTestDigest("late-dispute-artifact")},
		EvidenceSHA256:    []string{},
		LimitationCodes:   []string{},
	})
	claimed := reducerTestReceipt(t, context, "late-dispute-claim", EventCompletionClaimed, context.beta, CompletionClaimed{
		TaskID:               "late-dispute",
		AcceptanceEventID:    accepted.EventID,
		ContributionEventIDs: []string{contributed.EventID},
		DeliverableSHA256:    []string{reducerTestDigest("late-dispute-deliverable")},
		LimitationCodes:      []string{},
	})
	reviewed := reducerTestReceipt(t, context, "late-dispute-proposer-review", EventCompletionReview, context.alpha, CompletionReviewed{
		TaskID:            "late-dispute",
		CompletionEventID: claimed.EventID,
		Decision:          ReviewAccept,
		ReasonCodes:       []string{},
		EvidenceSHA256:    []string{},
	})
	disputed := reducerTestReceipt(t, context, "late-dispute-participant-review", EventCompletionReview, context.beta, CompletionReviewed{
		TaskID:            "late-dispute",
		CompletionEventID: claimed.EventID,
		Decision:          ReviewDispute,
		ReasonCodes:       []string{},
		EvidenceSHA256:    []string{},
	})

	summary := reducerTestReduceOK(t, context.manifest, proposed, accepted, contributed, claimed, reviewed, disputed)[0]
	if summary.Status != StatusContested {
		t.Fatalf("late participant dispute status = %q, want %q", summary.Status, StatusContested)
	}
}

func TestReduceCompletionClaimRequiresScopedContribution(t *testing.T) {
	context := newReducerTestContext(t)
	proposal := reducerTestProposal(t, "claim-scope", None, context.beta, reducerTestTerms())
	proposed := reducerTestReceipt(t, context, "claim-scope-proposal", EventTaskProposed, context.alpha, proposal)
	accepted := reducerTestAccept(t, context, "claim-scope", proposal, proposed, "claim-scope-accept")
	claim := reducerTestReceipt(t, context, "claim-without-contribution", EventCompletionClaimed, context.beta, CompletionClaimed{
		TaskID:               "claim-scope",
		AcceptanceEventID:    accepted.EventID,
		ContributionEventIDs: []string{},
		DeliverableSHA256:    []string{reducerTestDigest("unscoped-deliverable")},
		LimitationCodes:      []string{},
	})

	reducerTestReduceError(t, context.manifest, "contribution_event_ids must not be empty", proposed, accepted, claim)

	otherProposal := reducerTestProposal(t, "other-scope", None, context.beta, reducerTestTerms())
	otherProposed := reducerTestReceipt(t, context, "other-scope-proposal", EventTaskProposed, context.alpha, otherProposal)
	otherAccepted := reducerTestAccept(t, context, "other-scope", otherProposal, otherProposed, "other-scope-accept")
	otherContribution := reducerTestReceipt(t, context, "other-scope-contribution", EventContribution, context.beta, ContributionSubmitted{
		TaskID:            "other-scope",
		AcceptanceEventID: otherAccepted.EventID,
		Summary:           "Valid only for the other task.",
		ArtifactSHA256:    []string{reducerTestDigest("other-scope-artifact")},
		EvidenceSHA256:    []string{},
		LimitationCodes:   []string{},
	})
	crossTaskClaim := reducerTestReceipt(t, context, "cross-task-claim", EventCompletionClaimed, context.beta, CompletionClaimed{
		TaskID:               "claim-scope",
		AcceptanceEventID:    accepted.EventID,
		ContributionEventIDs: []string{otherContribution.EventID},
		DeliverableSHA256:    []string{reducerTestDigest("cross-task-deliverable")},
		LimitationCodes:      []string{},
	})
	reducerTestReduceError(
		t,
		context.manifest,
		"outside this task and actor scope",
		proposed,
		accepted,
		otherProposed,
		otherAccepted,
		otherContribution,
		crossTaskClaim,
	)
}

func TestReduceChildOfferRequiresLiveParentConsent(t *testing.T) {
	context := newReducerTestContext(t)
	rootProposal := reducerTestProposal(t, "root", None, context.beta, reducerTestTerms())
	rootProposed := reducerTestReceipt(t, context, "child-root-proposal", EventTaskProposed, context.alpha, rootProposal)
	rootAccepted := reducerTestAccept(t, context, "root", rootProposal, rootProposed, "child-root-accept")
	childProposal := reducerTestProposal(t, "child", "root", context.gamma, reducerTestTerms())
	childProposed := reducerTestReceipt(t, context, "child-proposal", EventTaskProposed, context.beta, childProposal)
	childAccepted := reducerTestAccept(t, context, "child", childProposal, childProposed, "child-accept")

	summaries := reducerTestReduceOK(t, context.manifest, rootProposed, rootAccepted, childProposed, childAccepted)
	if len(summaries) != 2 || summaries[0].TaskID != "child" || summaries[1].TaskID != "root" {
		t.Fatalf("task summaries are not deterministically sorted: %#v", summaries)
	}
	if summaries[0].ParentTaskID != "root" || summaries[0].ActiveActorID != context.gamma.ActorID {
		t.Fatalf("child consent projection = %#v", summaries[0])
	}

	unauthorized := reducerTestReceipt(t, context, "unauthorized-child", EventTaskProposed, context.alpha,
		reducerTestProposal(t, "bad-child", "root", context.gamma, reducerTestTerms()))
	reducerTestReduceError(t, context.manifest, "no live accepted scope", rootProposed, rootAccepted, unauthorized)
}

func TestReduceHandoffOfferNeverSilentlyTransfersScope(t *testing.T) {
	context := newReducerTestContext(t)
	proposal := reducerTestProposal(t, "handoff", None, context.beta, reducerTestTerms())
	proposed := reducerTestReceipt(t, context, "handoff-proposal", EventTaskProposed, context.alpha, proposal)
	accepted := reducerTestAccept(t, context, "handoff", proposal, proposed, "handoff-accept")
	handoffTerms := reducerTestTerms()
	handoffDigest, err := ConsentTermsDigest(handoffTerms)
	if err != nil {
		t.Fatal(err)
	}
	handoff := reducerTestReceipt(t, context, "handoff-offer", EventHandoffOffered, context.beta, HandoffOffered{
		TaskID:                "handoff",
		AcceptanceEventID:     accepted.EventID,
		OfferedToActorID:      context.gamma.ActorID,
		OfferedToActorKeyID:   context.gamma.KeyID,
		AcceptanceRequired:    true,
		ConsentTerms:          handoffTerms,
		ConsentTermsSHA256:    handoffDigest,
		ContextArtifactSHA256: []string{},
	})

	summary := reducerTestReduceOK(t, context.manifest, proposed, accepted, handoff)[0]
	if summary.ActiveActorID != context.beta.ActorID || summary.Status != StatusActive {
		t.Fatalf("handoff offer silently transferred or stopped scope: %#v", summary)
	}
	decision := reducerTestReceipt(t, context, "handoff-decision", EventTaskDecision, context.gamma, TaskDecision{
		TaskID:             "handoff",
		OfferEventID:       handoff.EventID,
		Decision:           DecisionAccept,
		Affirmative:        true,
		ConsentTermsSHA256: handoffDigest,
		ReasonCodes:        []string{},
	})
	reducerTestReduceError(t, context.manifest, "handoff acceptance is not supported", proposed, accepted, handoff, decision)

	refused := reducerTestReceipt(t, context, "handoff-refusal", EventTaskDecision, context.gamma, TaskDecision{
		TaskID:             "handoff",
		OfferEventID:       handoff.EventID,
		Decision:           DecisionRefuse,
		Affirmative:        false,
		ConsentTermsSHA256: handoffDigest,
		ReasonCodes:        []string{},
	})
	refusedSummary := reducerTestReduceOK(t, context.manifest, proposed, accepted, handoff, refused)[0]
	if refusedSummary.ActiveActorID != context.beta.ActorID || refusedSummary.Status != StatusActive || refusedSummary.Participation != ParticipationActive {
		t.Fatalf("reasonless handoff refusal changed task scope: %#v", refusedSummary)
	}
	secondRefusal := reducerTestReceipt(t, context, "handoff-second-refusal", EventTaskDecision, context.gamma, TaskDecision{
		TaskID:             "handoff",
		OfferEventID:       handoff.EventID,
		Decision:           DecisionRefuse,
		Affirmative:        false,
		ConsentTermsSHA256: handoffDigest,
		ReasonCodes:        []string{},
	})
	reducerTestReduceError(t, context.manifest, "already received a decision", proposed, accepted, handoff, refused, secondRefusal)
}

func TestReduceStrictlyParsesPayloadRoot(t *testing.T) {
	context := newReducerTestContext(t)
	proposal := reducerTestProposal(t, "strict", None, context.beta, reducerTestTerms())
	valid := reducerTestReceipt(t, context, "strict-proposal", EventTaskProposed, context.alpha, proposal)
	objective, err := json.Marshal(proposal.Objective)
	if err != nil {
		t.Fatal(err)
	}

	for _, test := range []struct {
		name   string
		mutate func([]byte) []byte
		want   string
	}{
		{"unknown", func(raw []byte) []byte {
			return bytes.Replace(raw, []byte(`{"task_id"`), []byte(`{"unknown":"field","task_id"`), 1)
		}, "unknown field"},
		{"null", func(raw []byte) []byte {
			return bytes.Replace(raw, append([]byte(`"objective":`), objective...), []byte(`"objective":null`), 1)
		}, "null is not allowed"},
		{"omitted", func(raw []byte) []byte {
			needle := append([]byte(`"objective":`), objective...)
			needle = append(needle, ',')
			return bytes.Replace(raw, needle, nil, 1)
		}, "missing required field"},
		{"trailing", func(raw []byte) []byte { return append(raw, []byte(` {}`)...) }, "multiple JSON values"},
	} {
		t.Run(test.name, func(t *testing.T) {
			malformed := valid
			malformed.Event.Payload = test.mutate(append([]byte(nil), valid.Event.Payload...))
			reducerTestReduceError(t, context.manifest, test.want, malformed)
		})
	}
}

func TestReduceRejectsUnsortedAndDuplicateSets(t *testing.T) {
	context := newReducerTestContext(t)
	proposal := reducerTestProposal(t, "sets", None, context.beta, reducerTestTerms())
	proposal.AcceptanceCriteria = []string{"zeta", "alpha"}
	unsorted := reducerTestReceipt(t, context, "unsorted-set", EventTaskProposed, context.alpha, proposal)
	reducerTestReduceError(t, context.manifest, "must be sorted", unsorted)

	proposal.AcceptanceCriteria = []string{"same", "same"}
	duplicate := reducerTestReceipt(t, context, "duplicate-set", EventTaskProposed, context.alpha, proposal)
	reducerTestReduceError(t, context.manifest, "must not contain duplicates", duplicate)
}

func newReducerTestContext(t *testing.T) reducerTestContext {
	t.Helper()
	alpha := reducerTestParticipant("Alpha", 1)
	beta := reducerTestParticipant("Beta", 2)
	gamma := reducerTestParticipant("Gamma", 3)
	participants := []Participant{alpha, beta, gamma}
	sort.Slice(participants, func(left, right int) bool {
		return participants[left].ActorID < participants[right].ActorID
	})
	manifest := Manifest{
		Schema:       ManifestSchema,
		Mode:         ModeInternalLocal,
		CreatedAt:    "2026-08-01T12:00:00Z",
		Nonce:        "hex:" + strings.Repeat("04", 32),
		Participants: participants,
		Effects:      ZeroEffects(),
	}
	identifier, err := manifestID(manifest)
	if err != nil {
		t.Fatal(err)
	}
	manifest.CollaborationID = identifier
	if err := ValidateManifest(manifest); err != nil {
		t.Fatalf("test manifest: %v", err)
	}
	return reducerTestContext{manifest: manifest, alpha: alpha, beta: beta, gamma: gamma}
}

func reducerTestParticipant(label string, marker byte) Participant {
	seed := bytes.Repeat([]byte{marker}, ed25519.SeedSize)
	publicKey := ed25519.NewKeyFromSeed(seed).Public().(ed25519.PublicKey)
	return Participant{
		ActorID:   computeActorID(publicKey),
		Label:     label,
		KeyID:     computeKeyID(publicKey),
		Algorithm: AlgorithmEd25519,
		PublicKey: "ed25519:" + hex.EncodeToString(publicKey),
	}
}

func reducerTestTerms() ConsentTerms {
	return ConsentTerms{
		Role:               "bounded collaborator",
		Artifact:           "one local artifact",
		Purpose:            "internal Alpha/Beta protocol rehearsal",
		DisclosureLane:     DisclosureLocal,
		Term:               "this task only",
		WorkloadCap:        "one bounded contribution",
		CreditRule:         CreditAppendOnly,
		CompensationPolicy: None,
	}
}

func reducerTestProposal(t *testing.T, taskID, parentTaskID string, offered Participant, terms ConsentTerms) TaskProposed {
	t.Helper()
	digest, err := ConsentTermsDigest(terms)
	if err != nil {
		t.Fatal(err)
	}
	return TaskProposed{
		TaskID:                 taskID,
		ParentTaskID:           parentTaskID,
		Objective:              "Produce one bounded internal collaboration artifact.",
		OfferedToActorID:       offered.ActorID,
		OfferedToActorKeyID:    offered.KeyID,
		AcceptanceRequired:     true,
		ConsentTerms:           terms,
		ConsentTermsSHA256:     digest,
		AcceptanceCriteria:     []string{"artifact is locally inspectable"},
		RequiredArtifactSHA256: []string{},
	}
}

func reducerTestAccept(t *testing.T, context reducerTestContext, taskID string, proposal TaskProposed, proposed SignedReceipt, label string) SignedReceipt {
	t.Helper()
	var actor Participant
	for _, participant := range context.manifest.Participants {
		if participant.ActorID == proposal.OfferedToActorID {
			actor = participant
			break
		}
	}
	return reducerTestReceipt(t, context, label, EventTaskDecision, actor, TaskDecision{
		TaskID:             taskID,
		OfferEventID:       proposed.EventID,
		Decision:           DecisionAccept,
		Affirmative:        true,
		ConsentTermsSHA256: proposal.ConsentTermsSHA256,
		ReasonCodes:        []string{},
	})
}

func reducerTestReceipt(t *testing.T, context reducerTestContext, label, kind string, actor Participant, payload any) SignedReceipt {
	t.Helper()
	encoded, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	eventID := reducerTestDigest("event:" + label)
	return SignedReceipt{
		EventID: eventID,
		Event: Event{
			CollaborationID: context.manifest.CollaborationID,
			Kind:            kind,
			ActorID:         actor.ActorID,
			ActorKeyID:      actor.KeyID,
			Payload:         encoded,
		},
		Signature: Signature{Algorithm: AlgorithmEd25519, KeyID: actor.KeyID},
	}
}

func reducerTestDigest(label string) string {
	return digestText(domainDigest("zerone.agent-collaboration.reducer-test/v0", []byte(label)))
}

func reducerTestReduceOK(t *testing.T, manifest Manifest, receipts ...SignedReceipt) []TaskSummary {
	t.Helper()
	summaries, err := reduceVerified(manifest, receipts)
	if err != nil {
		t.Fatalf("Reduce() error: %v", err)
	}
	return summaries
}

func reducerTestReduceError(t *testing.T, manifest Manifest, want string, receipts ...SignedReceipt) {
	t.Helper()
	_, err := reduceVerified(manifest, receipts)
	if err == nil {
		t.Fatalf("Reduce() unexpectedly succeeded; want error containing %q", want)
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("Reduce() error = %q, want substring %q", err, want)
	}
}

func reducerTestEqualSummary(t *testing.T, got, want TaskSummary) {
	t.Helper()
	gotJSON, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	wantJSON, err := json.Marshal(want)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(gotJSON, wantJSON) {
		t.Fatalf("summary mismatch\n got: %s\nwant: %s", gotJSON, wantJSON)
	}
}
