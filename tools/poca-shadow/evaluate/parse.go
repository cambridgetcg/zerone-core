package evaluate

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

const maxDocumentBytes = 1 << 20

// ParseProfile strictly decodes and validates a profile document.
func ParseProfile(data []byte) (Profile, error) {
	var profile Profile
	if err := decodeStrict(data, &profile); err != nil {
		return Profile{}, fmt.Errorf("profile: %w", err)
	}
	if err := validateRequiredProfileFields(data); err != nil {
		return Profile{}, fmt.Errorf("profile: %w", err)
	}
	if err := validateProfile(profile); err != nil {
		return Profile{}, fmt.Errorf("profile: %w", err)
	}
	return profile, nil
}

// ParseEvidence strictly decodes and validates an evidence bundle's intrinsic
// shape. Cross-document references are checked by Evaluate.
func ParseEvidence(data []byte) (EvidenceBundle, error) {
	var bundle EvidenceBundle
	if err := decodeStrict(data, &bundle); err != nil {
		return EvidenceBundle{}, fmt.Errorf("evidence bundle: %w", err)
	}
	if err := validateRequiredEvidenceFields(data); err != nil {
		return EvidenceBundle{}, fmt.Errorf("evidence bundle: %w", err)
	}
	if err := validateEvidenceBundle(bundle); err != nil {
		return EvidenceBundle{}, fmt.Errorf("evidence bundle: %w", err)
	}
	return bundle, nil
}

func decodeStrict(data []byte, dst any) error {
	if len(data) == 0 {
		return errors.New("document is empty")
	}
	if len(data) > maxDocumentBytes {
		return fmt.Errorf("document exceeds %d-byte limit", maxDocumentBytes)
	}
	if err := rejectDuplicateObjectKeys(data); err != nil {
		return err
	}

	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
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

func rejectDuplicateObjectKeys(data []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	var walk func(string) error
	walk = func(path string) error {
		token, err := decoder.Token()
		if err != nil {
			return fmt.Errorf("decode JSON token at %s: %w", path, err)
		}
		delim, ok := token.(json.Delim)
		if !ok {
			if token == nil {
				return fmt.Errorf("JSON null is not allowed at %s", path)
			}
			return nil
		}
		switch delim {
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
				if err := walk(path + "." + key); err != nil {
					return err
				}
			}
			if _, err := decoder.Token(); err != nil {
				return fmt.Errorf("close JSON object at %s: %w", path, err)
			}
		case '[':
			index := 0
			for decoder.More() {
				if err := walk(fmt.Sprintf("%s[%d]", path, index)); err != nil {
					return err
				}
				index++
			}
			if _, err := decoder.Token(); err != nil {
				return fmt.Errorf("close JSON array at %s: %w", path, err)
			}
		default:
			return fmt.Errorf("unexpected JSON delimiter %q at %s", delim, path)
		}
		return nil
	}

	if err := walk("$"); err != nil {
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

func validateRequiredProfileFields(data []byte) error {
	root, err := requireObjectFields(data, "$",
		"schema", "profile_id", "profile_version", "title", "status",
		"assurance_mode", "standards", "requirements", "nodes",
		"crown_node_id", "challenge_policy", "economics",
	)
	if err != nil {
		return err
	}
	if err := requireArrayObjectFields(root["standards"], "$.standards",
		"uri", "version", "status", "target",
	); err != nil {
		return err
	}
	if err := requireArrayObjectFields(root["requirements"], "$.requirements",
		"id", "kind", "verification_rule", "policy_digest", "observer_role",
		"min_count", "min_independent_control_clusters",
		"require_observer_independent_from_claimant", "hard_guardrail",
	); err != nil {
		return err
	}
	if err := rejectEmptyOptionalArrayFields(root["requirements"], "$.requirements", "predicate_type"); err != nil {
		return err
	}
	if err := requireArrayObjectFields(root["nodes"], "$.nodes",
		"id", "stage", "tier", "title", "prerequisites", "requirement_ids",
	); err != nil {
		return err
	}
	if _, err := requireObjectFields(root["challenge_policy"], "$.challenge_policy",
		"unresolved_challenge_blocks_crown",
	); err != nil {
		return err
	}
	if _, err := requireObjectFields(root["economics"], "$.economics",
		"mode", "amount_uzrn",
	); err != nil {
		return err
	}
	return nil
}

func validateRequiredEvidenceFields(data []byte) error {
	root, err := requireObjectFields(data, "$",
		"schema", "profile_id", "profile_version", "subject", "baseline_digest",
		"lineage_digest", "participants", "evidence",
		"unresolved_challenge_digests",
	)
	if err != nil {
		return err
	}
	if _, err := requireObjectFields(root["subject"], "$.subject",
		"name", "media_type", "digest",
	); err != nil {
		return err
	}
	if err := rejectEmptyOptionalObjectFields(root["subject"], "$.subject", "source_uri"); err != nil {
		return err
	}
	if err := requireArrayObjectFields(root["participants"], "$.participants",
		"id", "role", "identity", "control_cluster_claim",
	); err != nil {
		return err
	}
	if err := requireArrayObjectFields(root["evidence"], "$.evidence",
		"id", "requirement_id", "producer_participant_id",
		"observer_participant_id", "result", "verification_rule",
		"policy_digest", "subject_digest", "environment_digest",
		"statement_digest", "verification_receipt_digest",
	); err != nil {
		return err
	}
	if err := rejectEmptyOptionalArrayFields(root["evidence"], "$.evidence", "predicate_type", "source_uri"); err != nil {
		return err
	}
	var challenges []json.RawMessage
	if err := json.Unmarshal(root["unresolved_challenge_digests"], &challenges); err != nil {
		return fmt.Errorf("$.unresolved_challenge_digests must be an array: %w", err)
	}
	return nil
}

func requireArrayObjectFields(raw json.RawMessage, path string, fields ...string) error {
	var items []json.RawMessage
	if err := json.Unmarshal(raw, &items); err != nil {
		return fmt.Errorf("%s must be an array: %w", path, err)
	}
	for i, item := range items {
		if _, err := requireObjectFields(item, fmt.Sprintf("%s[%d]", path, i), fields...); err != nil {
			return err
		}
	}
	return nil
}

func rejectEmptyOptionalArrayFields(raw json.RawMessage, path string, fields ...string) error {
	var items []json.RawMessage
	if err := json.Unmarshal(raw, &items); err != nil {
		return fmt.Errorf("%s must be an array: %w", path, err)
	}
	for i, item := range items {
		if err := rejectEmptyOptionalObjectFields(item, fmt.Sprintf("%s[%d]", path, i), fields...); err != nil {
			return err
		}
	}
	return nil
}

func rejectEmptyOptionalObjectFields(raw json.RawMessage, path string, fields ...string) error {
	var object map[string]json.RawMessage
	if err := json.Unmarshal(raw, &object); err != nil {
		return fmt.Errorf("%s must be an object: %w", path, err)
	}
	for _, field := range fields {
		value, exists := object[field]
		if !exists {
			continue
		}
		var text string
		if err := json.Unmarshal(value, &text); err != nil {
			return fmt.Errorf("%s.%s must be a string: %w", path, field, err)
		}
		if text == "" {
			return fmt.Errorf("%s.%s must be omitted rather than empty", path, field)
		}
	}
	return nil
}

func requireObjectFields(raw json.RawMessage, path string, fields ...string) (map[string]json.RawMessage, error) {
	var object map[string]json.RawMessage
	if err := json.Unmarshal(raw, &object); err != nil {
		return nil, fmt.Errorf("%s must be an object: %w", path, err)
	}
	if object == nil {
		return nil, fmt.Errorf("%s must be an object", path)
	}
	for _, field := range fields {
		value, ok := object[field]
		if !ok {
			return nil, fmt.Errorf("%s is missing required field %q", path, field)
		}
		if bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			return nil, fmt.Errorf("%s.%s must not be null", path, field)
		}
	}
	return object, nil
}
