// Command agent-collaboration maintains an offline, zero-effect journal of
// signed task offers, protocol decisions, contributions, and reviews.
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/zerone-chain/zerone/tools/agent-collaboration/journal"
	"github.com/zerone-chain/zerone/tools/agent-collaboration/receipt"
)

const (
	maxJournalReceipts = receipt.MaxHistoryReceipts
	maxJournalBytes    = 16 << 20
)

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "agent-collaboration: %v\n", err)
		os.Exit(1)
	}
}

func run(arguments []string, stdout, stderr io.Writer) error {
	if len(arguments) == 0 {
		return errors.New("a command is required; use --help for the offline command list")
	}
	switch arguments[0] {
	case "-h", "--help", "help":
		_, err := io.WriteString(stdout, topLevelUsage)
		return err
	case "keygen":
		return runKeygen(arguments[1:], stdout, stderr)
	case "public":
		return runPublic(arguments[1:], stdout, stderr)
	case "consent-digest":
		return runConsentDigest(arguments[1:], stdout, stderr)
	case "init":
		return runInit(arguments[1:], stdout, stderr)
	case "append":
		return runAppend(arguments[1:], stdout, stderr)
	case "verify":
		return runVerify(arguments[1:], stdout, stderr)
	case "demo":
		return runDemo(arguments[1:], stdout, stderr)
	default:
		return fmt.Errorf("unknown command %q; use --help for the offline command list", arguments[0])
	}
}

const topLevelUsage = `usage: agent-collaboration COMMAND [OPTIONS]

Offline commands:
  keygen          create one private local signing key
  public          derive a roster-safe public-key document
  consent-digest  validate and digest one exact consent-terms document
  init            create one closed local collaboration journal
  append          verify, sign, and append one pinned event
  verify          verify and project a complete local journal
  demo            print an ephemeral Alpha/Beta transcript
`

func parseCommandFlags(flags *flag.FlagSet, arguments []string, stdout io.Writer) (bool, error) {
	for _, argument := range arguments {
		if argument == "-h" || argument == "--help" {
			flags.SetOutput(stdout)
			break
		}
	}
	err := flags.Parse(arguments)
	if errors.Is(err, flag.ErrHelp) {
		return true, nil
	}
	return false, err
}

func runKeygen(arguments []string, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("keygen", flag.ContinueOnError)
	flags.SetOutput(stderr)
	var label, output string
	flags.StringVar(&label, "label", "", "bounded local display label (required; non-authoritative)")
	flags.StringVar(&output, "out", "", "new private-key JSON path (required; created 0600, never replaced)")
	if help, err := parseCommandFlags(flags, arguments, stdout); err != nil {
		return err
	} else if help {
		return nil
	}
	if flags.NArg() != 0 || label == "" || output == "" {
		return errors.New("keygen requires --label and --out; positional arguments are not accepted")
	}
	privateKey, publicKey, err := receipt.GenerateKey(label)
	if err != nil {
		return err
	}
	privateJSON, err := receipt.MarshalDocument(privateKey)
	if err != nil {
		return err
	}
	if err := journal.CreateNoReplace(output, privateJSON, 0o600); err != nil {
		return fmt.Errorf("write private key: %w", err)
	}
	return writeJSON(stdout, publicKey)
}

func runPublic(arguments []string, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("public", flag.ContinueOnError)
	flags.SetOutput(stderr)
	var keyPath, output string
	flags.StringVar(&keyPath, "key", "", "private-key JSON path (required)")
	flags.StringVar(&output, "out", "", "new public-key JSON path (required; never replaced)")
	if help, err := parseCommandFlags(flags, arguments, stdout); err != nil {
		return err
	} else if help {
		return nil
	}
	if flags.NArg() != 0 || keyPath == "" || output == "" {
		return errors.New("public requires --key and --out; positional arguments are not accepted")
	}
	key, err := loadPrivateKey(keyPath)
	if err != nil {
		return err
	}
	publicKey, err := receipt.PublicFromPrivate(key)
	if err != nil {
		return err
	}
	encoded, err := receipt.MarshalDocument(publicKey)
	if err != nil {
		return err
	}
	if err := journal.CreateNoReplace(output, encoded, 0o600); err != nil {
		return fmt.Errorf("write public key: %w", err)
	}
	return writeJSON(stdout, publicKey)
}

func runConsentDigest(arguments []string, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("consent-digest", flag.ContinueOnError)
	flags.SetOutput(stderr)
	var termsPath string
	flags.StringVar(&termsPath, "terms", "", "exact consent-terms JSON path (required)")
	if help, err := parseCommandFlags(flags, arguments, stdout); err != nil {
		return err
	} else if help {
		return nil
	}
	if flags.NArg() != 0 || termsPath == "" {
		return errors.New("consent-digest requires --terms; positional arguments are not accepted")
	}
	data, err := journal.ReadRegular(termsPath, receipt.MaxConsentTermsBytes, false)
	if err != nil {
		return fmt.Errorf("read consent terms: %w", err)
	}
	terms, err := receipt.ParseConsentTerms(data)
	if err != nil {
		return err
	}
	digest, err := receipt.ConsentTermsDigest(terms)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintln(stdout, digest); err != nil {
		return fmt.Errorf("write consent digest: %w", err)
	}
	return nil
}

type repeatedStrings []string

func (values *repeatedStrings) String() string { return strings.Join(*values, ",") }
func (values *repeatedStrings) Set(value string) error {
	if value == "" {
		return errors.New("path must not be empty")
	}
	*values = append(*values, value)
	return nil
}

func runInit(arguments []string, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("init", flag.ContinueOnError)
	flags.SetOutput(stderr)
	var journalPath, createdAt string
	var participants repeatedStrings
	flags.StringVar(&journalPath, "journal", "", "new local journal directory (required)")
	flags.StringVar(&createdAt, "at", "", "canonical UTC RFC3339 seconds (default: current local clock claim)")
	flags.Var(&participants, "participant", "public-key JSON path (repeat 2-16 times)")
	if help, err := parseCommandFlags(flags, arguments, stdout); err != nil {
		return err
	} else if help {
		return nil
	}
	if flags.NArg() != 0 || journalPath == "" || len(participants) < 2 || len(participants) > 16 {
		return errors.New("init requires --journal and between 2 and 16 --participant files; positional arguments are not accepted")
	}
	if createdAt == "" {
		createdAt = canonicalNow()
	}
	roster := make([]receipt.Participant, 0, len(participants))
	for _, path := range participants {
		data, err := journal.ReadRegular(path, receipt.MaxKeyBytes, false)
		if err != nil {
			return fmt.Errorf("read participant: %w", err)
		}
		publicKey, err := receipt.ParsePublicKeyFile(data)
		if err != nil {
			return err
		}
		roster = append(roster, publicKey.Participant)
	}
	manifest, err := receipt.NewManifest(roster, createdAt)
	if err != nil {
		return err
	}
	encoded, err := receipt.MarshalDocument(manifest)
	if err != nil {
		return err
	}
	if err := journal.CreateJournal(journalPath); err != nil {
		return err
	}
	if err := journal.CreateNoReplace(filepath.Join(journalPath, "manifest.json"), encoded, 0o600); err != nil {
		return fmt.Errorf("write manifest into new journal: %w", err)
	}
	return writeJSON(stdout, manifest)
}

func runAppend(arguments []string, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("append", flag.ContinueOnError)
	flags.SetOutput(stderr)
	var journalPath, keyPath, requestPath, expectedID, expectedHead string
	flags.StringVar(&journalPath, "journal", "", "local collaboration journal (required)")
	flags.StringVar(&keyPath, "key", "", "actor private-key JSON (required)")
	flags.StringVar(&requestPath, "request", "", "unsigned event-request JSON (required)")
	flags.StringVar(&expectedID, "expect-collaboration-id", "", "caller-pinned manifest collaboration ID (required)")
	flags.StringVar(&expectedHead, "expect-head", "", "caller-pinned current receipt hash or NONE (required)")
	if help, err := parseCommandFlags(flags, arguments, stdout); err != nil {
		return err
	} else if help {
		return nil
	}
	if flags.NArg() != 0 || journalPath == "" || keyPath == "" || requestPath == "" || expectedID == "" || expectedHead == "" {
		return errors.New("append requires --journal, --key, --request, --expect-collaboration-id, and --expect-head; positional arguments are not accepted")
	}
	key, err := loadPrivateKey(keyPath)
	if err != nil {
		return err
	}
	requestBytes, err := journal.ReadRegular(requestPath, receipt.MaxRequestBytes, false)
	if err != nil {
		return fmt.Errorf("read event request: %w", err)
	}
	request, err := receipt.ParseEventRequest(requestBytes)
	if err != nil {
		return err
	}
	var finalReport receipt.VerificationReport
	err = journal.WithAppendLock(journalPath, func(locked *journal.LockedJournal) error {
		loaded, err := loadJournal(journalPath, true)
		if err != nil {
			return err
		}
		if loaded.manifest.CollaborationID != expectedID {
			return errors.New("manifest collaboration ID differs from --expect-collaboration-id")
		}
		if loaded.report.HeadReceiptSHA256 != expectedHead {
			return fmt.Errorf("journal head differs from --expect-head: current head is %s", loaded.report.HeadReceiptSHA256)
		}
		if len(loaded.receipts) >= maxJournalReceipts {
			return fmt.Errorf("candidate would exceed %d-receipt journal limit", maxJournalReceipts)
		}
		sequence := uint64(len(loaded.receipts) + 1)
		created, candidateReport, err := receipt.BuildNextReceipt(loaded.manifest, loaded.receipts, expectedID, expectedHead, request, key)
		if err != nil {
			return err
		}
		encoded, err := receipt.MarshalDocument(created)
		if err != nil {
			return err
		}
		if err := validateCandidateJournalBounds(len(loaded.receipts), loaded.totalBytes, len(encoded)); err != nil {
			return err
		}
		finalReport = candidateReport
		digest := strings.TrimPrefix(created.ReceiptSHA256, "sha256:")
		if _, err := locked.PublishReceipt(sequence, digest, encoded); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}
	return writeJSON(stdout, finalReport)
}

func validateCandidateJournalBounds(existingReceipts, existingBytes, candidateBytes int) error {
	if existingReceipts < 0 || existingBytes < 0 || candidateBytes <= 0 {
		return errors.New("candidate journal sizes must be positive and internally consistent")
	}
	if existingReceipts >= maxJournalReceipts {
		return fmt.Errorf("candidate would exceed %d-receipt journal limit", maxJournalReceipts)
	}
	if candidateBytes > receipt.MaxReceiptBytes {
		return fmt.Errorf("candidate receipt exceeds %d-byte limit", receipt.MaxReceiptBytes)
	}
	if existingBytes > maxJournalBytes || candidateBytes > maxJournalBytes-existingBytes {
		return fmt.Errorf("candidate would exceed %d-byte aggregate journal limit", maxJournalBytes)
	}
	return nil
}

func runVerify(arguments []string, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("verify", flag.ContinueOnError)
	flags.SetOutput(stderr)
	var journalPath, expectedID, expectedHead string
	flags.StringVar(&journalPath, "journal", "", "local collaboration journal (required)")
	flags.StringVar(&expectedID, "expect-collaboration-id", "", "optional caller-pinned collaboration ID")
	flags.StringVar(&expectedHead, "expect-head", "", "optional caller-pinned receipt head")
	if help, err := parseCommandFlags(flags, arguments, stdout); err != nil {
		return err
	} else if help {
		return nil
	}
	if flags.NArg() != 0 || journalPath == "" {
		return errors.New("verify requires --journal; positional arguments are not accepted")
	}
	loaded, err := loadJournal(journalPath, false)
	if err != nil {
		return err
	}
	if expectedID != "" && loaded.manifest.CollaborationID != expectedID {
		return errors.New("manifest collaboration ID differs from --expect-collaboration-id")
	}
	if expectedHead != "" && loaded.report.HeadReceiptSHA256 != expectedHead {
		return errors.New("verified journal head differs from --expect-head")
	}
	return writeJSON(stdout, loaded.report)
}

func loadPrivateKey(path string) (receipt.PrivateKeyFile, error) {
	data, err := journal.ReadRegular(path, receipt.MaxKeyBytes, true)
	if err != nil {
		return receipt.PrivateKeyFile{}, fmt.Errorf("read private key: %w", err)
	}
	return receipt.ParsePrivateKeyFile(data)
}

type loadedJournal struct {
	manifest   receipt.Manifest
	receipts   []receipt.SignedReceipt
	report     receipt.VerificationReport
	totalBytes int
}

func loadJournal(path string, appendLockHeld bool) (loadedJournal, error) {
	if err := journal.ValidateLayout(path, appendLockHeld); err != nil {
		return loadedJournal{}, err
	}
	manifestBytes, err := journal.ReadRegular(filepath.Join(path, "manifest.json"), receipt.MaxManifestBytes, true)
	if err != nil {
		return loadedJournal{}, fmt.Errorf("read manifest: %w", err)
	}
	manifest, err := receipt.ParseManifest(manifestBytes)
	if err != nil {
		return loadedJournal{}, err
	}
	canonicalManifest, err := receipt.MarshalDocument(manifest)
	if err != nil {
		return loadedJournal{}, err
	}
	if !bytes.Equal(manifestBytes, canonicalManifest) {
		return loadedJournal{}, errors.New("journal manifest bytes are not the canonical typed encoding")
	}
	paths, err := journal.ReceiptPaths(path)
	if err != nil {
		return loadedJournal{}, err
	}
	if len(paths) > maxJournalReceipts {
		return loadedJournal{}, fmt.Errorf("journal exceeds %d-receipt limit", maxJournalReceipts)
	}
	receipts := make([]receipt.SignedReceipt, 0, len(paths))
	totalBytes := len(manifestBytes)
	for index, receiptPath := range paths {
		data, err := journal.ReadRegular(receiptPath, receipt.MaxReceiptBytes, true)
		if err != nil {
			return loadedJournal{}, fmt.Errorf("read receipt %d: %w", index+1, err)
		}
		totalBytes += len(data)
		if totalBytes > maxJournalBytes {
			return loadedJournal{}, fmt.Errorf("journal exceeds %d-byte aggregate limit", maxJournalBytes)
		}
		parsed, err := receipt.ParseSignedReceipt(data)
		if err != nil {
			return loadedJournal{}, err
		}
		canonicalReceipt, err := receipt.MarshalDocument(parsed)
		if err != nil {
			return loadedJournal{}, err
		}
		if !bytes.Equal(data, canonicalReceipt) {
			return loadedJournal{}, fmt.Errorf("receipt %d bytes are not the canonical typed encoding", index+1)
		}
		wantName := fmt.Sprintf("%020d-%s.json", index+1, strings.TrimPrefix(parsed.ReceiptSHA256, "sha256:"))
		if filepath.Base(receiptPath) != wantName {
			return loadedJournal{}, fmt.Errorf("receipt %d filename does not match its sequence and digest", index+1)
		}
		receipts = append(receipts, parsed)
	}
	report, err := receipt.VerifyHistory(manifest, receipts)
	if err != nil {
		return loadedJournal{}, err
	}
	return loadedJournal{manifest: manifest, receipts: receipts, report: report, totalBytes: totalBytes}, nil
}

func canonicalNow() string {
	return time.Now().UTC().Truncate(time.Second).Format(time.RFC3339)
}

func writeJSON(output io.Writer, value any) error {
	encoder := json.NewEncoder(output)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(value); err != nil {
		return fmt.Errorf("encode JSON output: %w", err)
	}
	return nil
}

// runDemo is implemented separately to keep the normal command paths small.
func runDemo(arguments []string, stdout, stderr io.Writer) error {
	return runInternalDemo(arguments, stdout, stderr)
}
