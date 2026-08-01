package main

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"sort"
	"time"

	"github.com/zerone-chain/zerone/tools/agent-collaboration/receipt"
)

type demoTranscript struct {
	Schema       string                     `json:"schema"`
	Note         string                     `json:"note"`
	Manifest     receipt.Manifest           `json:"manifest"`
	Receipts     []receipt.SignedReceipt    `json:"receipts"`
	Verification receipt.VerificationReport `json:"verification"`
}

func runInternalDemo(arguments []string, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("demo", flag.ContinueOnError)
	flags.SetOutput(stderr)
	var at string
	flags.StringVar(&at, "at", "", "canonical UTC RFC3339 seconds (default: current local clock claim)")
	if help, err := parseCommandFlags(flags, arguments, stdout); err != nil {
		return err
	} else if help {
		return nil
	}
	if flags.NArg() != 0 {
		return errors.New("demo accepts no positional arguments")
	}
	if at == "" {
		at = canonicalNow()
	}
	base, err := time.Parse(time.RFC3339, at)
	if err != nil || base.Format(time.RFC3339) != at {
		return errors.New("--at must be canonical UTC RFC3339 seconds")
	}

	alphaPrivate, alphaPublic, err := receipt.GenerateKey("Alpha")
	if err != nil {
		return err
	}
	betaPrivate, betaPublic, err := receipt.GenerateKey("Beta")
	if err != nil {
		return err
	}
	manifest, err := receipt.NewManifest([]receipt.Participant{alphaPublic.Participant, betaPublic.Participant}, at)
	if err != nil {
		return err
	}

	terms := receipt.ConsentTerms{
		Role:               "collaborator",
		Artifact:           "signed-alpha-beta-receipt-journal",
		Purpose:            "shape-one-internal-task-through-explicit-collaboration",
		DisclosureLane:     receipt.DisclosureLocal,
		Term:               "one-internal-rehearsal",
		WorkloadCap:        "one-bounded-task",
		CreditRule:         receipt.CreditAppendOnly,
		CompensationPolicy: receipt.None,
	}
	termsDigest, err := receipt.ConsentTermsDigest(terms)
	if err != nil {
		return err
	}
	deliverable := textDigest("alpha-beta signed journal rehearsal")
	receipts := make([]receipt.SignedReceipt, 0, 5)
	head := receipt.None
	appendEvent := func(offset int, kind string, key receipt.PrivateKeyFile, payload any) error {
		raw, err := receipt.MarshalDocument(payload)
		if err != nil {
			return err
		}
		request := receipt.EventRequest{
			Schema:     receipt.EventRequestSchema,
			Kind:       kind,
			ActorID:    key.ActorID,
			OccurredAt: base.Add(time.Duration(offset) * time.Second).Format(time.RFC3339),
			Payload:    raw,
		}
		created, report, err := receipt.BuildNextReceipt(manifest, receipts, manifest.CollaborationID, head, request, key)
		if err != nil {
			return err
		}
		receipts = append(receipts, created)
		head = created.ReceiptSHA256
		if report.HeadReceiptSHA256 != head {
			return errors.New("demo candidate report head mismatch")
		}
		return nil
	}

	criteria := []string{
		"alpha-can-verify-beta-signed-contribution",
		"journal-remains-zero-effect-and-offline",
	}
	sort.Strings(criteria)
	if err := appendEvent(0, receipt.EventTaskProposed, alphaPrivate, receipt.TaskProposed{
		TaskID:                 "alpha-beta-internal-v0",
		ParentTaskID:           receipt.None,
		Objective:              "exercise-one-explicit-offer-accept-contribute-review-loop",
		OfferedToActorID:       betaPrivate.ActorID,
		OfferedToActorKeyID:    betaPrivate.KeyID,
		AcceptanceRequired:     true,
		ConsentTerms:           terms,
		ConsentTermsSHA256:     termsDigest,
		AcceptanceCriteria:     criteria,
		RequiredArtifactSHA256: []string{},
	}); err != nil {
		return fmt.Errorf("demo proposal: %w", err)
	}
	proposalID := receipts[len(receipts)-1].EventID
	if err := appendEvent(1, receipt.EventTaskDecision, betaPrivate, receipt.TaskDecision{
		TaskID:             "alpha-beta-internal-v0",
		OfferEventID:       proposalID,
		Decision:           receipt.DecisionAccept,
		Affirmative:        true,
		ConsentTermsSHA256: termsDigest,
		ReasonCodes:        []string{},
	}); err != nil {
		return fmt.Errorf("demo acceptance: %w", err)
	}
	acceptanceID := receipts[len(receipts)-1].EventID
	if err := appendEvent(2, receipt.EventContribution, betaPrivate, receipt.ContributionSubmitted{
		TaskID:            "alpha-beta-internal-v0",
		AcceptanceEventID: acceptanceID,
		Summary:           "beta-created-a-content-addressed-local-deliverable",
		ArtifactSHA256:    []string{deliverable},
		EvidenceSHA256:    []string{},
		LimitationCodes:   []string{"SYNTHETIC_INTERNAL_REHEARSAL"},
	}); err != nil {
		return fmt.Errorf("demo contribution: %w", err)
	}
	contributionID := receipts[len(receipts)-1].EventID
	if err := appendEvent(3, receipt.EventCompletionClaimed, betaPrivate, receipt.CompletionClaimed{
		TaskID:               "alpha-beta-internal-v0",
		AcceptanceEventID:    acceptanceID,
		ContributionEventIDs: []string{contributionID},
		DeliverableSHA256:    []string{deliverable},
		LimitationCodes:      []string{"SYNTHETIC_INTERNAL_REHEARSAL"},
	}); err != nil {
		return fmt.Errorf("demo completion claim: %w", err)
	}
	completionID := receipts[len(receipts)-1].EventID
	if err := appendEvent(4, receipt.EventCompletionReview, alphaPrivate, receipt.CompletionReviewed{
		TaskID:            "alpha-beta-internal-v0",
		CompletionEventID: completionID,
		Decision:          receipt.ReviewAccept,
		ReasonCodes:       []string{},
		EvidenceSHA256:    []string{deliverable},
	}); err != nil {
		return fmt.Errorf("demo review: %w", err)
	}
	report, err := receipt.VerifyHistory(manifest, receipts)
	if err != nil {
		return err
	}
	return writeJSON(stdout, demoTranscript{
		Schema:       "zerone.agent-collaboration-demo/v0",
		Note:         "Alpha and Beta are local role labels; this transcript has no chain, economic, reward, KARMA, governance, ownership, or authority effect.",
		Manifest:     manifest,
		Receipts:     receipts,
		Verification: report,
	})
}

func textDigest(value string) string {
	digest := sha256.Sum256([]byte(value))
	return "sha256:" + hex.EncodeToString(digest[:])
}
