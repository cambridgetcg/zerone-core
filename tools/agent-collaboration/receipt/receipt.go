package receipt

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strconv"
	"time"

	voied25519 "github.com/oasisprotocol/curve25519-voi/primitives/ed25519"
)

var strictEd25519Options = &voied25519.Options{Verify: &voied25519.VerifyOptions{}}

// MarshalDocument emits stable compact JSON. Signatures and IDs are computed
// over their typed canonical subdocuments, so insignificant input whitespace
// is never security-sensitive.
func MarshalDocument(value any) ([]byte, error) {
	return canonicalJSON(value)
}

// BuildNextReceipt verifies the complete current history and caller-pinned
// head, signs one event with an OS-CSPRNG nonce, and verifies the candidate
// history's consent transition before returning it. The caller must hold the
// journal append lock from this call through publication.
func BuildNextReceipt(manifest Manifest, history []SignedReceipt, expectedCollaborationID, expectedHead string, request EventRequest, key PrivateKeyFile) (SignedReceipt, VerificationReport, error) {
	if err := validateDigest("expected_collaboration_id", expectedCollaborationID); err != nil {
		return SignedReceipt{}, VerificationReport{}, err
	}
	if expectedHead != None {
		if err := validateDigest("expected_head", expectedHead); err != nil {
			return SignedReceipt{}, VerificationReport{}, err
		}
	}
	current, err := VerifyHistory(manifest, history)
	if err != nil {
		return SignedReceipt{}, VerificationReport{}, fmt.Errorf("current history: %w", err)
	}
	if current.HeadReceiptSHA256 != expectedHead {
		return SignedReceipt{}, VerificationReport{}, fmt.Errorf("current history head is %q, not caller-pinned %q", current.HeadReceiptSHA256, expectedHead)
	}
	if current.CollaborationID != expectedCollaborationID {
		return SignedReceipt{}, VerificationReport{}, fmt.Errorf("current collaboration ID is %q, not caller-pinned %q", current.CollaborationID, expectedCollaborationID)
	}
	created, err := buildReceiptWithRandom(manifest, uint64(len(history)+1), expectedHead, request, key, rand.Reader)
	if err != nil {
		return SignedReceipt{}, VerificationReport{}, err
	}
	candidate := append(append([]SignedReceipt(nil), history...), created)
	report, err := VerifyHistory(manifest, candidate)
	if err != nil {
		return SignedReceipt{}, VerificationReport{}, fmt.Errorf("candidate history: %w", err)
	}
	return created, report, nil
}

func buildReceiptWithRandom(manifest Manifest, sequence uint64, previous string, request EventRequest, key PrivateKeyFile, randomness io.Reader) (SignedReceipt, error) {
	if err := ValidateManifest(manifest); err != nil {
		return SignedReceipt{}, fmt.Errorf("manifest: %w", err)
	}
	if sequence == 0 {
		return SignedReceipt{}, errors.New("sequence must be positive")
	}
	if request.Schema != EventRequestSchema {
		return SignedReceipt{}, fmt.Errorf("request schema must be %q", EventRequestSchema)
	}
	if request.ActorID != key.ActorID {
		return SignedReceipt{}, errors.New("request actor_id does not match the signing key")
	}
	participant, exists := participantFor(manifest, request.ActorID)
	if !exists {
		return SignedReceipt{}, errors.New("request actor_id is not in the manifest roster")
	}
	if err := requireKeyMatchesParticipant(key, participant); err != nil {
		return SignedReceipt{}, err
	}
	privateKey, err := privateSigningKey(key)
	if err != nil {
		return SignedReceipt{}, err
	}
	if err := validateTimestamp("occurred_at", request.OccurredAt); err != nil {
		return SignedReceipt{}, err
	}
	_, payload, err := DecodePayload(request.Kind, request.Payload)
	if err != nil {
		return SignedReceipt{}, err
	}
	if sequence == 1 {
		if previous != None {
			return SignedReceipt{}, fmt.Errorf("first receipt previous hash must be %q", None)
		}
		if request.Kind != EventTaskProposed {
			return SignedReceipt{}, fmt.Errorf("first receipt kind must be %q", EventTaskProposed)
		}
	} else if err := validateDigest("previous_receipt_sha256", previous); err != nil {
		return SignedReceipt{}, err
	}
	nonce, err := randomNonceWithReader(randomness)
	if err != nil {
		return SignedReceipt{}, err
	}
	event := Event{
		Schema:                EventSchema,
		CollaborationID:       manifest.CollaborationID,
		Sequence:              strconv.FormatUint(sequence, 10),
		PreviousReceiptSHA256: previous,
		Kind:                  request.Kind,
		ActorID:               request.ActorID,
		ActorKeyID:            participant.KeyID,
		OccurredAt:            request.OccurredAt,
		Nonce:                 nonce,
		Payload:               payload,
		Effects:               ZeroEffects(),
	}
	eventIDBytes, err := eventDigest(event)
	if err != nil {
		return SignedReceipt{}, err
	}
	receipt := SignedReceipt{
		Schema:  ReceiptSchema,
		EventID: digestText(eventIDBytes),
		Event:   event,
		Signature: Signature{
			Algorithm: AlgorithmEd25519,
			KeyID:     participant.KeyID,
			Value:     "ed25519:" + hex.EncodeToString(ed25519.Sign(privateKey, eventIDBytes)),
		},
		ReceiptSHA256: "",
	}
	receiptIDBytes, err := receiptDigest(receipt)
	if err != nil {
		return SignedReceipt{}, err
	}
	receipt.ReceiptSHA256 = digestText(receiptIDBytes)
	if err := verifyReceipt(manifest, receipt, sequence, previous); err != nil {
		return SignedReceipt{}, fmt.Errorf("self-verify signed receipt: %w", err)
	}
	return receipt, nil
}

// VerifyHistory verifies the manifest, every content/signature/hash link, all
// replay invariants, and the consent state machine before returning a report.
func VerifyHistory(manifest Manifest, receipts []SignedReceipt) (VerificationReport, error) {
	if len(receipts) > MaxHistoryReceipts {
		return VerificationReport{}, fmt.Errorf("history exceeds %d-receipt limit", MaxHistoryReceipts)
	}
	if err := ValidateManifest(manifest); err != nil {
		return VerificationReport{}, fmt.Errorf("manifest: %w", err)
	}
	previous := None
	previousTime, _ := time.Parse(time.RFC3339, manifest.CreatedAt)
	eventIDs := make(map[string]struct{}, len(receipts))
	receiptIDs := make(map[string]struct{}, len(receipts))
	nonces := make(map[string]struct{}, len(receipts))
	for index, receipt := range receipts {
		sequence := uint64(index + 1)
		if err := verifyReceipt(manifest, receipt, sequence, previous); err != nil {
			return VerificationReport{}, fmt.Errorf("receipt sequence %d: %w", sequence, err)
		}
		if _, exists := eventIDs[receipt.EventID]; exists {
			return VerificationReport{}, fmt.Errorf("receipt sequence %d replays event_id", sequence)
		}
		if _, exists := receiptIDs[receipt.ReceiptSHA256]; exists {
			return VerificationReport{}, fmt.Errorf("receipt sequence %d replays receipt_sha256", sequence)
		}
		nonceKey := receipt.Event.ActorKeyID + "\x00" + receipt.Event.Nonce
		if _, exists := nonces[nonceKey]; exists {
			return VerificationReport{}, fmt.Errorf("receipt sequence %d reuses an actor nonce", sequence)
		}
		occurredAt, _ := time.Parse(time.RFC3339, receipt.Event.OccurredAt)
		if occurredAt.Before(previousTime) {
			return VerificationReport{}, fmt.Errorf("receipt sequence %d regresses occurred_at", sequence)
		}
		previousTime = occurredAt
		eventIDs[receipt.EventID] = struct{}{}
		receiptIDs[receipt.ReceiptSHA256] = struct{}{}
		nonces[nonceKey] = struct{}{}
		previous = receipt.ReceiptSHA256
	}
	if len(receipts) > 0 && receipts[0].Event.Kind != EventTaskProposed {
		return VerificationReport{}, fmt.Errorf("first receipt kind must be %q", EventTaskProposed)
	}
	tasks, err := reduceVerified(manifest, receipts)
	if err != nil {
		return VerificationReport{}, fmt.Errorf("reduce collaboration state: %w", err)
	}
	assurance := AssuranceNoSignedEvents
	if len(receipts) > 0 {
		assurance = AssuranceEventKeyPossession
	}
	report := VerificationReport{
		Schema:            ReportSchema,
		Valid:             true,
		Mode:              ModeInternalLocal,
		Assurance:         assurance,
		CollaborationID:   manifest.CollaborationID,
		EventCount:        strconv.Itoa(len(receipts)),
		HeadReceiptSHA256: previous,
		Effects:           ZeroEffects(),
		Tasks:             tasks,
		Limitations: []string{
			"NO_CHAIN_NETWORK_ECONOMIC_REWARD_KARMA_OR_GOVERNANCE_EFFECT",
			"SIGNATURES_PROVE_EXACT_LOCAL_KEY_POSSESSION_ONLY",
			"TASK_STATUS_IS_A_SIGNED_PROTOCOL_DECLARATION_NOT_TRUTH_QUALITY_OR_LEGAL_AUTHORITY",
			"FREE_TEXT_AND_CONTENT_DIGESTS_ARE_UNINTERPRETED_DECLARATIONS",
			"VERIFICATION_REPORT_IS_UNSIGNED_REVERIFY_PINNED_JOURNAL_BYTES",
			"UNSIGNED_MANIFEST_DOES_NOT_PROVE_POSSESSION_OF_NON_SIGNING_ROSTER_KEYS",
		},
	}
	return report, nil
}

func verifyReceipt(manifest Manifest, receipt SignedReceipt, expectedSequence uint64, expectedPrevious string) error {
	if receipt.Schema != ReceiptSchema {
		return fmt.Errorf("schema must be %q", ReceiptSchema)
	}
	if receipt.Event.Schema != EventSchema {
		return fmt.Errorf("event schema must be %q", EventSchema)
	}
	if receipt.Event.CollaborationID != manifest.CollaborationID {
		return errors.New("event collaboration_id does not match manifest")
	}
	sequence, err := validateSequence(receipt.Event.Sequence)
	if err != nil {
		return err
	}
	if sequence != expectedSequence {
		return fmt.Errorf("sequence mismatch: expected %d, got %d", expectedSequence, sequence)
	}
	if receipt.Event.PreviousReceiptSHA256 != expectedPrevious {
		return fmt.Errorf("previous receipt mismatch: expected %s", expectedPrevious)
	}
	if expectedSequence == 1 {
		if expectedPrevious != None {
			return errors.New("first receipt must link to NONE")
		}
	} else if err := validateDigest("previous_receipt_sha256", receipt.Event.PreviousReceiptSHA256); err != nil {
		return err
	}
	if err := validateTimestamp("occurred_at", receipt.Event.OccurredAt); err != nil {
		return err
	}
	if _, err := parsePrefixedHex("nonce", receipt.Event.Nonce, "hex:", 32); err != nil {
		return err
	}
	if err := validateEffects(receipt.Event.Effects); err != nil {
		return err
	}
	participant, exists := participantFor(manifest, receipt.Event.ActorID)
	if !exists {
		return errors.New("event actor_id is not in manifest roster")
	}
	if receipt.Event.ActorKeyID != participant.KeyID {
		return errors.New("event actor_key_id does not match manifest")
	}
	if receipt.Signature.Algorithm != AlgorithmEd25519 {
		return fmt.Errorf("signature algorithm must be %q", AlgorithmEd25519)
	}
	if receipt.Signature.KeyID != participant.KeyID {
		return errors.New("signature key_id does not match event actor key")
	}
	eventIDBytes, err := eventDigest(receipt.Event)
	if err != nil {
		return err
	}
	if receipt.EventID != digestText(eventIDBytes) {
		return errors.New("event_id does not match canonical event")
	}
	signature, err := decodeSignature(receipt.Signature.Value)
	if err != nil {
		return err
	}
	publicKey, err := publicSigningKey(participant)
	if err != nil {
		return err
	}
	if !strictVerifyEd25519(publicKey, eventIDBytes, signature) {
		return errors.New("invalid Ed25519 signature")
	}
	receiptIDBytes, err := receiptDigest(receipt)
	if err != nil {
		return err
	}
	if receipt.ReceiptSHA256 != digestText(receiptIDBytes) {
		return errors.New("receipt_sha256 does not match canonical signed receipt")
	}
	if _, err := parseDigest(receipt.EventID); err != nil {
		return fmt.Errorf("event_id: %w", err)
	}
	if _, err := parseDigest(receipt.ReceiptSHA256); err != nil {
		return fmt.Errorf("receipt_sha256: %w", err)
	}
	return nil
}

func strictVerifyEd25519(publicKey ed25519.PublicKey, message, signature []byte) bool {
	return voied25519.VerifyWithOptions(voied25519.PublicKey(publicKey), message, signature, strictEd25519Options)
}

// DecodeDocuments is a small helper for tests and callers that already hold
// bounded receipt bytes. It preserves order and stops on the first ambiguity.
func DecodeDocuments(documents [][]byte) ([]SignedReceipt, error) {
	if len(documents) > MaxHistoryReceipts {
		return nil, fmt.Errorf("document set exceeds %d-receipt limit", MaxHistoryReceipts)
	}
	receipts := make([]SignedReceipt, 0, len(documents))
	for index, document := range documents {
		receipt, err := ParseSignedReceipt(document)
		if err != nil {
			return nil, fmt.Errorf("document %d: %w", index+1, err)
		}
		receipts = append(receipts, receipt)
	}
	return receipts, nil
}
