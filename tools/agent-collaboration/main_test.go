package main

import (
	"bytes"
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/zerone-chain/zerone/tools/agent-collaboration/journal"
	"github.com/zerone-chain/zerone/tools/agent-collaboration/receipt"
)

func TestRunDemoProducesVerifiableZeroEffectTranscript(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if err := run([]string{"demo", "--at", "2026-08-01T12:00:00Z"}, &stdout, &stderr); err != nil {
		t.Fatalf("run demo: %v (stderr: %s)", err, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("demo wrote stderr: %s", stderr.String())
	}
	if strings.Contains(stdout.String(), "ed25519-seed:") {
		t.Fatal("demo output leaked private key material")
	}
	var transcript demoTranscript
	decoder := json.NewDecoder(&stdout)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&transcript); err != nil {
		t.Fatalf("decode transcript: %v", err)
	}
	report, err := receipt.VerifyHistory(transcript.Manifest, transcript.Receipts)
	if err != nil {
		t.Fatalf("verify transcript: %v", err)
	}
	if !report.Valid || report.EventCount != "5" || report.Tasks[0].Status != receipt.StatusAccepted {
		t.Fatalf("demo report: %#v", report)
	}
	if report.Effects != receipt.ZeroEffects() {
		t.Fatalf("demo effects: %#v", report.Effects)
	}
}

func TestCLIJournalAlphaBetaOfferAndAcceptance(t *testing.T) {
	directory := t.TempDir()
	alphaPrivate := filepath.Join(directory, "alpha.private.json")
	betaPrivate := filepath.Join(directory, "beta.private.json")
	alphaPublic := filepath.Join(directory, "alpha.public.json")
	betaPublic := filepath.Join(directory, "beta.public.json")

	alpha := runKeyLifecycle(t, "Alpha", alphaPrivate, alphaPublic)
	beta := runKeyLifecycle(t, "Beta", betaPrivate, betaPublic)
	for _, path := range []string{alphaPrivate, betaPrivate} {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm()&0o077 != 0 {
			t.Fatalf("private key %s mode = %o", filepath.Base(path), info.Mode().Perm())
		}
	}

	journalPath := filepath.Join(directory, "alpha-beta.journal")
	initOutput := runOK(t, []string{
		"init", "--journal", journalPath,
		"--participant", alphaPublic,
		"--participant", betaPublic,
		"--at", "2026-08-01T12:00:00Z",
	})
	var manifest receipt.Manifest
	decodeOutput(t, initOutput, &manifest)

	emptyOutput := runOK(t, []string{"verify", "--journal", journalPath, "--expect-collaboration-id", manifest.CollaborationID, "--expect-head", receipt.None})
	var empty receipt.VerificationReport
	decodeOutput(t, emptyOutput, &empty)
	if empty.EventCount != "0" || empty.HeadReceiptSHA256 != receipt.None {
		t.Fatalf("empty journal report: %#v", empty)
	}
	if empty.Assurance != receipt.AssuranceNoSignedEvents {
		t.Fatalf("empty journal assurance = %q", empty.Assurance)
	}

	terms := receipt.ConsentTerms{
		Role:               "collaborator",
		Artifact:           "one-local-receipt",
		Purpose:            "internal-alpha-beta-test",
		DisclosureLane:     receipt.DisclosureLocal,
		Term:               "one-task",
		WorkloadCap:        "one-contribution",
		CreditRule:         receipt.CreditAppendOnly,
		CompensationPolicy: receipt.None,
	}
	termsDigest, err := receipt.ConsentTermsDigest(terms)
	if err != nil {
		t.Fatal(err)
	}
	proposalRequest := receipt.EventRequest{
		Schema:     receipt.EventRequestSchema,
		Kind:       receipt.EventTaskProposed,
		ActorID:    alpha.Participant.ActorID,
		OccurredAt: "2026-08-01T12:00:00Z",
		Payload: mustJSON(t, receipt.TaskProposed{
			TaskID:                 "alpha-beta-cli-v0",
			ParentTaskID:           receipt.None,
			Objective:              "prove explicit offer and acceptance through the real local journal",
			OfferedToActorID:       beta.Participant.ActorID,
			OfferedToActorKeyID:    beta.Participant.KeyID,
			AcceptanceRequired:     true,
			ConsentTerms:           terms,
			ConsentTermsSHA256:     termsDigest,
			AcceptanceCriteria:     []string{"receipt-is-locally-verifiable"},
			RequiredArtifactSHA256: []string{},
		}),
	}
	proposalPath := filepath.Join(directory, "proposal.request.json")
	writeTestFile(t, proposalPath, mustJSON(t, proposalRequest), 0o600)
	proposalOutput := runOK(t, []string{
		"append", "--journal", journalPath,
		"--key", alphaPrivate,
		"--request", proposalPath,
		"--expect-collaboration-id", manifest.CollaborationID,
		"--expect-head", receipt.None,
	})
	var proposed receipt.VerificationReport
	decodeOutput(t, proposalOutput, &proposed)
	if proposed.Tasks[0].Status != receipt.StatusUnanswered {
		t.Fatalf("proposal silently became accepted: %#v", proposed.Tasks[0])
	}

	paths, err := journal.ReceiptPaths(journalPath)
	if err != nil || len(paths) != 1 {
		t.Fatalf("proposal receipt paths = %#v, err %v", paths, err)
	}
	proposalReceipt := readReceipt(t, paths[0])
	decisionRequest := receipt.EventRequest{
		Schema:     receipt.EventRequestSchema,
		Kind:       receipt.EventTaskDecision,
		ActorID:    beta.Participant.ActorID,
		OccurredAt: "2026-08-01T12:00:01Z",
		Payload: mustJSON(t, receipt.TaskDecision{
			TaskID:             "alpha-beta-cli-v0",
			OfferEventID:       proposalReceipt.EventID,
			Decision:           receipt.DecisionAccept,
			Affirmative:        true,
			ConsentTermsSHA256: termsDigest,
			ReasonCodes:        []string{},
		}),
	}
	decisionPath := filepath.Join(directory, "decision.request.json")
	writeTestFile(t, decisionPath, mustJSON(t, decisionRequest), 0o600)
	decisionOutput := runOK(t, []string{
		"append", "--journal", journalPath,
		"--key", betaPrivate,
		"--request", decisionPath,
		"--expect-collaboration-id", manifest.CollaborationID,
		"--expect-head", proposed.HeadReceiptSHA256,
	})
	var accepted receipt.VerificationReport
	decodeOutput(t, decisionOutput, &accepted)
	if accepted.Tasks[0].Status != receipt.StatusActive || accepted.Tasks[0].ActiveActorID != beta.Participant.ActorID {
		t.Fatalf("accepted task report: %#v", accepted.Tasks[0])
	}

	verifiedOutput := runOK(t, []string{
		"verify", "--journal", journalPath,
		"--expect-collaboration-id", manifest.CollaborationID,
		"--expect-head", accepted.HeadReceiptSHA256,
	})
	var verified receipt.VerificationReport
	decodeOutput(t, verifiedOutput, &verified)
	if verified.EventCount != "2" || verified.Effects != receipt.ZeroEffects() {
		t.Fatalf("verified report: %#v", verified)
	}

	journalAlias := filepath.Join(directory, "journal-alias")
	if err := os.Symlink(journalPath, journalAlias); err == nil {
		for _, spelling := range []string{journalAlias + string(os.PathSeparator), journalAlias + string(os.PathSeparator) + "."} {
			var aliasOut, aliasErr bytes.Buffer
			err := run([]string{"verify", "--journal", spelling}, &aliasOut, &aliasErr)
			if err == nil || aliasOut.Len() != 0 {
				t.Fatalf("verify followed journal symlink spelling %q: err=%v stdout=%s", spelling, err, aliasOut.String())
			}
		}
	} else {
		t.Logf("directory symlink regression skipped: %v", err)
	}

	var staleOut, staleErr bytes.Buffer
	err = run([]string{
		"append", "--journal", journalPath,
		"--key", betaPrivate,
		"--request", decisionPath,
		"--expect-collaboration-id", manifest.CollaborationID,
		"--expect-head", proposed.HeadReceiptSHA256,
	}, &staleOut, &staleErr)
	if err == nil || !strings.Contains(err.Error(), "current head") || staleOut.Len() != 0 {
		t.Fatalf("stale append result: err=%v stdout=%s", err, staleOut.String())
	}

	if err := filepath.WalkDir(journalPath, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if bytes.Contains(data, []byte("ed25519-seed:")) {
			t.Fatalf("journal file %s contains private key material", filepath.Base(path))
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	firstReceiptBytes, err := os.ReadFile(paths[0])
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(paths[0], append([]byte(" "), firstReceiptBytes...), 0o600); err != nil {
		t.Fatal(err)
	}
	var noncanonicalOut, noncanonicalErr bytes.Buffer
	err = run([]string{"verify", "--journal", journalPath}, &noncanonicalOut, &noncanonicalErr)
	if err == nil || !strings.Contains(err.Error(), "not the canonical typed encoding") || noncanonicalOut.Len() != 0 {
		t.Fatalf("noncanonical journal bytes: err=%v stdout=%s", err, noncanonicalOut.String())
	}
	if err := os.WriteFile(paths[0], firstReceiptBytes, 0o600); err != nil {
		t.Fatal(err)
	}

	privateBytes, err := os.ReadFile(alphaPrivate)
	if err != nil {
		t.Fatal(err)
	}
	hiddenKey := filepath.Join(journalPath, "alpha.private.json")
	writeTestFile(t, hiddenKey, privateBytes, 0o600)
	var rejectedOut, rejectedErr bytes.Buffer
	err = run([]string{"verify", "--journal", journalPath}, &rejectedOut, &rejectedErr)
	if err == nil || !strings.Contains(err.Error(), "unexpected journal-root entry") || rejectedOut.Len() != 0 {
		t.Fatalf("journal with hidden root key: err=%v stdout=%s", err, rejectedOut.String())
	}
}

func TestCandidateJournalBoundsFailBeforePublication(t *testing.T) {
	for _, test := range []struct {
		name           string
		existingCount  int
		existingBytes  int
		candidateBytes int
		wantError      bool
	}{
		{"within bounds", maxJournalReceipts - 1, maxJournalBytes - 1, 1, false},
		{"receipt count", maxJournalReceipts, 1, 1, true},
		{"candidate bytes", 0, 1, receipt.MaxReceiptBytes + 1, true},
		{"aggregate bytes", 0, maxJournalBytes, 1, true},
		{"empty candidate", 0, 1, 0, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := validateCandidateJournalBounds(test.existingCount, test.existingBytes, test.candidateBytes)
			if (err != nil) != test.wantError {
				t.Fatalf("validateCandidateJournalBounds() error = %v, wantError %v", err, test.wantError)
			}
		})
	}
}

func TestConsentDigestAndHelpAreUsableOffline(t *testing.T) {
	terms := receipt.ConsentTerms{
		Role:               "collaborator",
		Artifact:           "one-local-artifact",
		Purpose:            "internal-alpha-beta-test",
		DisclosureLane:     receipt.DisclosureLocal,
		Term:               "one-task",
		WorkloadCap:        "one-contribution",
		CreditRule:         receipt.CreditAppendOnly,
		CompensationPolicy: receipt.None,
	}
	want, err := receipt.ConsentTermsDigest(terms)
	if err != nil {
		t.Fatal(err)
	}
	termsPath := filepath.Join(t.TempDir(), "terms.json")
	writeTestFile(t, termsPath, mustJSON(t, terms), 0o600)
	if got := strings.TrimSpace(runOK(t, []string{"consent-digest", "--terms", termsPath})); got != want {
		t.Fatalf("consent digest = %q, want %q", got, want)
	}

	if output := runOK(t, []string{"--help"}); !strings.Contains(output, "consent-digest") || !strings.Contains(output, "demo") {
		t.Fatalf("top-level help = %q", output)
	}
	if output := runOK(t, []string{"consent-digest", "--help"}); !strings.Contains(output, "Usage of consent-digest") || !strings.Contains(output, "-terms") {
		t.Fatalf("subcommand help = %q", output)
	}

	invalid := bytes.Replace(mustJSON(t, terms), []byte(`{"role":`), []byte(`{"unexpected":"x","role":`), 1)
	invalidPath := filepath.Join(filepath.Dir(termsPath), "invalid-terms.json")
	writeTestFile(t, invalidPath, invalid, 0o600)
	var stdout, stderr bytes.Buffer
	if err := run([]string{"consent-digest", "--terms", invalidPath}, &stdout, &stderr); err == nil || stdout.Len() != 0 {
		t.Fatalf("invalid terms result: err=%v stdout=%q", err, stdout.String())
	}

	arguments := []string{"init", "--journal", filepath.Join(t.TempDir(), "never-created")}
	for range 17 {
		arguments = append(arguments, "--participant", "unread-path")
	}
	stdout.Reset()
	stderr.Reset()
	if err := run(arguments, &stdout, &stderr); err == nil || !strings.Contains(err.Error(), "between 2 and 16") {
		t.Fatalf("oversized init roster error = %v", err)
	}
}

func TestProductionCodeHasNoNetworkExecOrChainClientSurface(t *testing.T) {
	root := toolRoot(t)
	forbiddenImports := []string{
		"net", "net/", "os/exec", "github.com/cosmos", "cosmossdk.io",
		"agenttool-relay", "/x/knowledge", "/x/home", "/x/auth",
	}
	forbiddenSelectors := map[string]struct{}{
		"Dial": {}, "DialContext": {}, "Listen": {}, "Socket": {}, "Connect": {},
		"Command": {}, "CommandContext": {}, "StartProcess": {}, "ForkExec": {}, "Exec": {},
	}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		lower := strings.ToLower(string(data))
		for _, forbidden := range []string{"http://", "https://", "zeroned ", "broadcast_tx", "agenttool.dev"} {
			if strings.Contains(lower, forbidden) {
				t.Errorf("%s contains forbidden production literal %q", filepath.Base(path), forbidden)
			}
		}
		parsed, err := parser.ParseFile(token.NewFileSet(), path, data, 0)
		if err != nil {
			return err
		}
		for _, imported := range parsed.Imports {
			name, err := strconv.Unquote(imported.Path.Value)
			if err != nil {
				return err
			}
			for _, forbidden := range forbiddenImports {
				if name == forbidden || strings.HasPrefix(name, forbidden) || strings.Contains(name, forbidden) {
					t.Errorf("%s imports forbidden capability %q", filepath.Base(path), name)
				}
			}
		}
		ast.Inspect(parsed, func(node ast.Node) bool {
			selector, ok := node.(*ast.SelectorExpr)
			if ok {
				if _, forbidden := forbiddenSelectors[selector.Sel.Name]; forbidden {
					t.Errorf("%s calls forbidden capability selector %s", filepath.Base(path), selector.Sel.Name)
				}
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func runKeyLifecycle(t *testing.T, label, privatePath, publicPath string) receipt.PublicKeyFile {
	t.Helper()
	keygenOutput := runOK(t, []string{"keygen", "--label", label, "--out", privatePath})
	var announced receipt.PublicKeyFile
	decodeOutput(t, keygenOutput, &announced)
	if strings.Contains(keygenOutput, "ed25519-seed:") {
		t.Fatal("keygen stdout leaked private material")
	}
	publicOutput := runOK(t, []string{"public", "--key", privatePath, "--out", publicPath})
	var exported receipt.PublicKeyFile
	decodeOutput(t, publicOutput, &exported)
	if announced != exported {
		t.Fatalf("keygen/public projection drift\nannounced: %#v\nexported: %#v", announced, exported)
	}
	return exported
}

func runOK(t *testing.T, arguments []string) string {
	t.Helper()
	var stdout, stderr bytes.Buffer
	if err := run(arguments, &stdout, &stderr); err != nil {
		t.Fatalf("run %v: %v (stderr: %s)", arguments, err, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("run %v wrote stderr: %s", arguments, stderr.String())
	}
	return stdout.String()
}

func decodeOutput(t *testing.T, output string, destination any) {
	t.Helper()
	decoder := json.NewDecoder(strings.NewReader(output))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		t.Fatalf("decode output: %v\n%s", err, output)
	}
}

func mustJSON(t *testing.T, value any) []byte {
	t.Helper()
	encoded, err := receipt.MarshalDocument(value)
	if err != nil {
		t.Fatal(err)
	}
	return encoded
}

func writeTestFile(t *testing.T, path string, data []byte, mode fs.FileMode) {
	t.Helper()
	if err := os.WriteFile(path, data, mode); err != nil {
		t.Fatal(err)
	}
}

func readReceipt(t *testing.T, path string) receipt.SignedReceipt {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := receipt.ParseSignedReceipt(data)
	if err != nil {
		t.Fatal(err)
	}
	return parsed
}

func toolRoot(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Dir(filename)
}
