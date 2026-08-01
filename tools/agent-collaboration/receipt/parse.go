package receipt

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"unicode/utf8"
)

const (
	MaxManifestBytes     = 64 << 10
	MaxKeyBytes          = 16 << 10
	MaxConsentTermsBytes = 16 << 10
	MaxRequestBytes      = 64 << 10
	MaxReceiptBytes      = 128 << 10
	MaxHistoryReceipts   = 4096
	maxJSONDepth         = 32
)

// ParseManifest strictly decodes a collaboration manifest.
func ParseManifest(data []byte) (Manifest, error) {
	var manifest Manifest
	if err := decodeStrict(data, MaxManifestBytes, &manifest); err != nil {
		return Manifest{}, fmt.Errorf("manifest: %w", err)
	}
	root, err := requireObjectFields(data, "$", "schema", "mode", "collaboration_id", "created_at", "nonce", "participants", "effects")
	if err != nil {
		return Manifest{}, fmt.Errorf("manifest: %w", err)
	}
	if err := requireEffectsFields(root["effects"], "$.effects"); err != nil {
		return Manifest{}, fmt.Errorf("manifest: %w", err)
	}
	var rawParticipants []json.RawMessage
	if err := json.Unmarshal(root["participants"], &rawParticipants); err != nil {
		return Manifest{}, fmt.Errorf("manifest: $.participants must be an array: %w", err)
	}
	for index, raw := range rawParticipants {
		if _, err := requireObjectFields(raw, fmt.Sprintf("$.participants[%d]", index), "actor_id", "label", "key_id", "algorithm", "public_key"); err != nil {
			return Manifest{}, fmt.Errorf("manifest: %w", err)
		}
	}
	if err := ValidateManifest(manifest); err != nil {
		return Manifest{}, fmt.Errorf("manifest: %w", err)
	}
	return manifest, nil
}

// ParsePrivateKeyFile strictly decodes local secret key material.
func ParsePrivateKeyFile(data []byte) (PrivateKeyFile, error) {
	var key PrivateKeyFile
	if err := decodeStrict(data, MaxKeyBytes, &key); err != nil {
		return PrivateKeyFile{}, fmt.Errorf("private key: %w", err)
	}
	if _, err := requireObjectFields(data, "$", "schema", "actor_id", "label", "key_id", "algorithm", "public_key", "private_key"); err != nil {
		return PrivateKeyFile{}, fmt.Errorf("private key: %w", err)
	}
	if err := ValidatePrivateKeyFile(key); err != nil {
		return PrivateKeyFile{}, fmt.Errorf("private key: %w", err)
	}
	return key, nil
}

// ParsePublicKeyFile strictly decodes a roster entry.
func ParsePublicKeyFile(data []byte) (PublicKeyFile, error) {
	var key PublicKeyFile
	if err := decodeStrict(data, MaxKeyBytes, &key); err != nil {
		return PublicKeyFile{}, fmt.Errorf("public key: %w", err)
	}
	root, err := requireObjectFields(data, "$", "schema", "participant")
	if err != nil {
		return PublicKeyFile{}, fmt.Errorf("public key: %w", err)
	}
	if _, err := requireObjectFields(root["participant"], "$.participant", "actor_id", "label", "key_id", "algorithm", "public_key"); err != nil {
		return PublicKeyFile{}, fmt.Errorf("public key: %w", err)
	}
	if err := ValidatePublicKeyFile(key); err != nil {
		return PublicKeyFile{}, fmt.Errorf("public key: %w", err)
	}
	return key, nil
}

// ParseConsentTerms strictly decodes the exact terms object used by task and
// handoff offers. It is useful for offline digest construction.
func ParseConsentTerms(data []byte) (ConsentTerms, error) {
	var terms ConsentTerms
	if err := decodeStrict(data, MaxConsentTermsBytes, &terms); err != nil {
		return ConsentTerms{}, fmt.Errorf("consent terms: %w", err)
	}
	if _, err := requireObjectFields(data, "$", "role", "artifact", "purpose", "disclosure_lane", "term", "workload_cap", "credit_rule", "compensation_policy"); err != nil {
		return ConsentTerms{}, fmt.Errorf("consent terms: %w", err)
	}
	if err := validateConsentTerms("consent_terms", terms); err != nil {
		return ConsentTerms{}, fmt.Errorf("consent terms: %w", err)
	}
	return terms, nil
}

// ParseEventRequest strictly decodes one unsigned local request.
func ParseEventRequest(data []byte) (EventRequest, error) {
	var request EventRequest
	if err := decodeStrict(data, MaxRequestBytes, &request); err != nil {
		return EventRequest{}, fmt.Errorf("event request: %w", err)
	}
	if _, err := requireObjectFields(data, "$", "schema", "kind", "actor_id", "occurred_at", "payload"); err != nil {
		return EventRequest{}, fmt.Errorf("event request: %w", err)
	}
	if request.Schema != EventRequestSchema {
		return EventRequest{}, fmt.Errorf("event request: schema must be %q", EventRequestSchema)
	}
	if _, canonical, err := DecodePayload(request.Kind, request.Payload); err != nil {
		return EventRequest{}, fmt.Errorf("event request: %w", err)
	} else {
		request.Payload = canonical
	}
	return request, nil
}

// ParseSignedReceipt strictly decodes one immutable receipt.
func ParseSignedReceipt(data []byte) (SignedReceipt, error) {
	var receipt SignedReceipt
	if err := decodeStrict(data, MaxReceiptBytes, &receipt); err != nil {
		return SignedReceipt{}, fmt.Errorf("receipt: %w", err)
	}
	root, err := requireObjectFields(data, "$", "schema", "event_id", "event", "signature", "receipt_sha256")
	if err != nil {
		return SignedReceipt{}, fmt.Errorf("receipt: %w", err)
	}
	eventRoot, err := requireObjectFields(root["event"], "$.event", "schema", "collaboration_id", "sequence", "previous_receipt_sha256", "kind", "actor_id", "actor_key_id", "occurred_at", "nonce", "payload", "effects")
	if err != nil {
		return SignedReceipt{}, fmt.Errorf("receipt: %w", err)
	}
	if err := requireEffectsFields(eventRoot["effects"], "$.event.effects"); err != nil {
		return SignedReceipt{}, fmt.Errorf("receipt: %w", err)
	}
	if _, err := requireObjectFields(root["signature"], "$.signature", "algorithm", "key_id", "value"); err != nil {
		return SignedReceipt{}, fmt.Errorf("receipt: %w", err)
	}
	if _, canonical, err := DecodePayload(receipt.Event.Kind, receipt.Event.Payload); err != nil {
		return SignedReceipt{}, fmt.Errorf("receipt: %w", err)
	} else {
		receipt.Event.Payload = canonical
	}
	return receipt, nil
}

// DecodePayload strictly decodes a kind-specific payload and also returns its
// canonical typed JSON representation.
func DecodePayload(kind string, raw json.RawMessage) (any, json.RawMessage, error) {
	var destination any
	var fields []string
	switch kind {
	case EventTaskProposed:
		destination = &TaskProposed{}
		fields = []string{"task_id", "parent_task_id", "objective", "offered_to_actor_id", "offered_to_actor_key_id", "acceptance_required", "consent_terms", "consent_terms_sha256", "acceptance_criteria", "required_artifact_sha256"}
	case EventTaskDecision:
		destination = &TaskDecision{}
		fields = []string{"task_id", "offer_event_id", "decision", "affirmative_acceptance", "consent_terms_sha256", "reason_codes"}
	case EventContribution:
		destination = &ContributionSubmitted{}
		fields = []string{"task_id", "acceptance_event_id", "summary", "artifact_sha256", "evidence_sha256", "limitation_codes"}
	case EventCompletionClaimed:
		destination = &CompletionClaimed{}
		fields = []string{"task_id", "acceptance_event_id", "contribution_event_ids", "deliverable_sha256", "limitation_codes"}
	case EventCompletionReview:
		destination = &CompletionReviewed{}
		fields = []string{"task_id", "completion_event_id", "decision", "reason_codes", "evidence_sha256"}
	case EventHandoffOffered:
		destination = &HandoffOffered{}
		fields = []string{"task_id", "acceptance_event_id", "offered_to_actor_id", "offered_to_actor_key_id", "acceptance_required", "consent_terms", "consent_terms_sha256", "context_artifact_sha256"}
	case EventControl:
		destination = &ControlDeclared{}
		fields = []string{"task_id", "acceptance_event_id", "action", "reason_codes", "export_event_ids"}
	default:
		return nil, nil, fmt.Errorf("unsupported event kind %q", kind)
	}
	if err := decodeStrict(raw, MaxRequestBytes, destination); err != nil {
		return nil, nil, fmt.Errorf("%s payload: %w", kind, err)
	}
	object, err := requireObjectFields(raw, "$.payload", fields...)
	if err != nil {
		return nil, nil, fmt.Errorf("%s payload: %w", kind, err)
	}
	if kind == EventTaskProposed {
		if _, err := requireObjectFields(object["consent_terms"], "$.payload.consent_terms", "role", "artifact", "purpose", "disclosure_lane", "term", "workload_cap", "credit_rule", "compensation_policy"); err != nil {
			return nil, nil, fmt.Errorf("%s payload: %w", kind, err)
		}
	}
	if kind == EventHandoffOffered {
		if _, err := requireObjectFields(object["consent_terms"], "$.payload.consent_terms", "role", "artifact", "purpose", "disclosure_lane", "term", "workload_cap", "credit_rule", "compensation_policy"); err != nil {
			return nil, nil, fmt.Errorf("%s payload: %w", kind, err)
		}
	}
	canonical, err := canonicalJSON(destination)
	if err != nil {
		return nil, nil, err
	}
	return destination, canonical, nil
}

func decodeStrict(data []byte, maximum int, destination any) error {
	if len(data) == 0 {
		return errors.New("document is empty")
	}
	if len(data) > maximum {
		return fmt.Errorf("document exceeds %d-byte limit", maximum)
	}
	if !utf8.Valid(data) {
		return errors.New("document is not valid UTF-8")
	}
	if err := inspectJSON(data); err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return fmt.Errorf("decode JSON: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("multiple JSON values are not allowed")
		}
		return fmt.Errorf("decode trailing JSON: %w", err)
	}
	return nil
}

func inspectJSON(data []byte) error {
	if err := rejectUnpairedSurrogateEscapes(data); err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	var walk func(string, int) error
	walk = func(path string, depth int) error {
		if depth > maxJSONDepth {
			return fmt.Errorf("JSON nesting exceeds depth %d at %s", maxJSONDepth, path)
		}
		token, err := decoder.Token()
		if err != nil {
			return fmt.Errorf("decode JSON token at %s: %w", path, err)
		}
		delimiter, ok := token.(json.Delim)
		if !ok {
			switch token.(type) {
			case nil:
				return fmt.Errorf("JSON null is not allowed at %s", path)
			case float64, json.Number:
				return fmt.Errorf("JSON numbers are not allowed at %s; use canonical decimal strings", path)
			}
			return nil
		}
		switch delimiter {
		case '{':
			seen := make(map[string]struct{})
			for decoder.More() {
				keyToken, err := decoder.Token()
				if err != nil {
					return fmt.Errorf("decode JSON object key at %s: %w", path, err)
				}
				key, ok := keyToken.(string)
				if !ok {
					return fmt.Errorf("non-string JSON object key at %s", path)
				}
				if _, exists := seen[key]; exists {
					return fmt.Errorf("duplicate JSON object key %q at %s", key, path)
				}
				seen[key] = struct{}{}
				if err := walk(fmt.Sprintf("%s[%q]", path, key), depth+1); err != nil {
					return err
				}
			}
			_, err = decoder.Token()
			return err
		case '[':
			index := 0
			for decoder.More() {
				if err := walk(fmt.Sprintf("%s[%d]", path, index), depth+1); err != nil {
					return err
				}
				index++
			}
			_, err = decoder.Token()
			return err
		default:
			return fmt.Errorf("unexpected JSON delimiter %q at %s", delimiter, path)
		}
	}
	if err := walk("$", 1); err != nil {
		return err
	}
	if _, err := decoder.Token(); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("multiple JSON values are not allowed")
		}
		return fmt.Errorf("decode trailing JSON token: %w", err)
	}
	return nil
}

func rejectUnpairedSurrogateEscapes(data []byte) error {
	inString := false
	for index := 0; index < len(data); index++ {
		switch data[index] {
		case '"':
			inString = !inString
		case '\\':
			if !inString || index+1 >= len(data) {
				continue
			}
			if data[index+1] != 'u' {
				index++
				continue
			}
			value, ok := escapedHex16(data, index)
			if !ok {
				continue // The JSON decoder will report malformed escape syntax.
			}
			if value >= 0xdc00 && value <= 0xdfff {
				return errors.New("unpaired UTF-16 low-surrogate escape is not allowed")
			}
			if value < 0xd800 || value > 0xdbff {
				index += 5
				continue
			}
			next := index + 6
			low, paired := escapedHex16(data, next)
			if !paired || low < 0xdc00 || low > 0xdfff {
				return errors.New("unpaired UTF-16 high-surrogate escape is not allowed")
			}
			index = next + 5
		}
	}
	return nil
}

func escapedHex16(data []byte, slash int) (uint16, bool) {
	if slash < 0 || slash+6 > len(data) || data[slash] != '\\' || data[slash+1] != 'u' {
		return 0, false
	}
	var value uint16
	for _, character := range data[slash+2 : slash+6] {
		value <<= 4
		switch {
		case character >= '0' && character <= '9':
			value |= uint16(character - '0')
		case character >= 'a' && character <= 'f':
			value |= uint16(character-'a') + 10
		case character >= 'A' && character <= 'F':
			value |= uint16(character-'A') + 10
		default:
			return 0, false
		}
	}
	return value, true
}

func requireObjectFields(raw json.RawMessage, path string, fields ...string) (map[string]json.RawMessage, error) {
	var object map[string]json.RawMessage
	if err := json.Unmarshal(raw, &object); err != nil {
		return nil, fmt.Errorf("%s must be an object: %w", path, err)
	}
	if object == nil {
		return nil, fmt.Errorf("%s must be an object", path)
	}
	expected := make(map[string]struct{}, len(fields))
	for _, field := range fields {
		expected[field] = struct{}{}
		value, ok := object[field]
		if !ok {
			return nil, fmt.Errorf("%s is missing required field %q", path, field)
		}
		if bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			return nil, fmt.Errorf("%s.%s must not be null", path, field)
		}
	}
	for field := range object {
		if _, ok := expected[field]; !ok {
			return nil, fmt.Errorf("%s contains unexpected exact field %q", path, field)
		}
	}
	if len(object) != len(expected) {
		return nil, fmt.Errorf("%s must contain exactly %d fields", path, len(expected))
	}
	return object, nil
}

func requireEffectsFields(raw json.RawMessage, path string) error {
	_, err := requireObjectFields(raw, path, "network", "chain", "economic", "fiat", "zrn", "reward", "karma", "governance", "ownership", "qualification", "membership", "endorsement", "authority", "attribution")
	return err
}
