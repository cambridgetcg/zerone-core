package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"strconv"
	"strings"
)

const (
	maxJSONDepth    = 256
	maxArrayEntries = 1_000_000
)

// rejectDuplicateKeys runs before any map unmarshal. encoding/json otherwise
// silently keeps the final value for a duplicate object key, which would make
// an offline migration decision ambiguous.
func rejectDuplicateKeys(data []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := walkJSON(decoder, "$", 0); err != nil {
		return err
	}
	token, err := decoder.Token()
	if err == io.EOF {
		return nil
	}
	if err != nil {
		return fmt.Errorf("decode JSON trailer: %w", err)
	}
	return fmt.Errorf("schema ambiguity: unexpected JSON value after root: %v", token)
}

func walkJSON(decoder *json.Decoder, path string, depth int) error {
	if depth > maxJSONDepth {
		return fmt.Errorf("schema ambiguity: JSON nesting exceeds %d levels at %s", maxJSONDepth, path)
	}
	token, err := decoder.Token()
	if err != nil {
		return fmt.Errorf("decode JSON at %s: %w", path, err)
	}
	delim, composite := token.(json.Delim)
	if !composite {
		return nil
	}
	switch delim {
	case '{':
		seen := make(map[string]struct{})
		for decoder.More() {
			keyToken, err := decoder.Token()
			if err != nil {
				return fmt.Errorf("decode object key at %s: %w", path, err)
			}
			key, ok := keyToken.(string)
			if !ok {
				return fmt.Errorf("decode object key at %s: key is not a string", path)
			}
			if _, duplicate := seen[key]; duplicate {
				return fmt.Errorf("schema ambiguity: duplicate JSON key %q at %s", key, path)
			}
			seen[key] = struct{}{}
			if err := walkJSON(decoder, path+"."+key, depth+1); err != nil {
				return err
			}
		}
		end, err := decoder.Token()
		if err != nil {
			return fmt.Errorf("decode object end at %s: %w", path, err)
		}
		if end != json.Delim('}') {
			return fmt.Errorf("decode object end at %s: got %v", path, end)
		}
	case '[':
		index := 0
		for decoder.More() {
			if index >= maxArrayEntries {
				return fmt.Errorf("schema ambiguity: array at %s exceeds %d entries", path, maxArrayEntries)
			}
			if err := walkJSON(decoder, fmt.Sprintf("%s[%d]", path, index), depth+1); err != nil {
				return err
			}
			index++
		}
		end, err := decoder.Token()
		if err != nil {
			return fmt.Errorf("decode array end at %s: %w", path, err)
		}
		if end != json.Delim(']') {
			return fmt.Errorf("decode array end at %s: got %v", path, end)
		}
	default:
		return fmt.Errorf("decode JSON at %s: unexpected delimiter %q", path, delim)
	}
	return nil
}

func decodeObject(raw json.RawMessage, path string, required, allowed []string) (map[string]json.RawMessage, error) {
	if len(raw) == 0 || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return nil, fmt.Errorf("schema ambiguity: %s must be a JSON object", path)
	}
	var object map[string]json.RawMessage
	if err := json.Unmarshal(raw, &object); err != nil {
		return nil, fmt.Errorf("decode %s: %w", path, err)
	}
	if object == nil {
		return nil, fmt.Errorf("schema ambiguity: %s must be a JSON object", path)
	}
	allowedSet := make(map[string]struct{}, len(allowed))
	for _, key := range allowed {
		allowedSet[key] = struct{}{}
	}
	for key := range object {
		if _, ok := allowedSet[key]; !ok {
			return nil, fmt.Errorf("schema ambiguity: unexpected field %s.%s", path, key)
		}
	}
	for _, key := range required {
		if _, ok := object[key]; !ok {
			return nil, fmt.Errorf("schema ambiguity: required field %s.%s is missing", path, key)
		}
	}
	return object, nil
}

func decodeArray(raw json.RawMessage, path string) ([]json.RawMessage, error) {
	if len(raw) == 0 || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return nil, fmt.Errorf("schema ambiguity: %s must be an exported JSON array, not null", path)
	}
	var values []json.RawMessage
	if err := json.Unmarshal(raw, &values); err != nil {
		return nil, fmt.Errorf("decode %s: %w", path, err)
	}
	if values == nil {
		return nil, fmt.Errorf("schema ambiguity: %s must be an exported JSON array", path)
	}
	if len(values) > maxArrayEntries {
		return nil, fmt.Errorf("schema ambiguity: %s exceeds %d entries", path, maxArrayEntries)
	}
	return values, nil
}

func decodeString(raw json.RawMessage, path string, allowEmpty bool) (string, error) {
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return "", fmt.Errorf("schema ambiguity: %s must be a JSON string: %w", path, err)
	}
	if !allowEmpty && value == "" {
		return "", fmt.Errorf("schema ambiguity: %s must not be empty", path)
	}
	return value, nil
}

func decodeStringArray(raw json.RawMessage, path string) ([]string, error) {
	values, err := decodeArray(raw, path)
	if err != nil {
		return nil, err
	}
	result := make([]string, len(values))
	for i, value := range values {
		result[i], err = decodeString(value, fmt.Sprintf("%s[%d]", path, i), false)
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}

func decodeUintString(raw json.RawMessage, path string, positive bool) (string, error) {
	value, err := decodeString(raw, path, false)
	if err != nil {
		return "", err
	}
	if strings.HasPrefix(value, "+") || strings.TrimSpace(value) != value {
		return "", fmt.Errorf("schema ambiguity: %s must be a canonical protobuf uint64 string", path)
	}
	number, err := strconv.ParseUint(value, 10, 64)
	if err != nil || strconv.FormatUint(number, 10) != value || (positive && number == 0) {
		requirement := "non-negative"
		if positive {
			requirement = "positive"
		}
		return "", fmt.Errorf("schema ambiguity: %s must be a canonical %s protobuf uint64 string", path, requirement)
	}
	return value, nil
}

func decodeBase64(raw json.RawMessage, path string) ([]byte, error) {
	value, err := decodeString(raw, path, true)
	if err != nil {
		return nil, err
	}
	decoded, err := base64.StdEncoding.Strict().DecodeString(value)
	if err != nil {
		return nil, fmt.Errorf("schema ambiguity: %s is not canonical padded base64: %w", path, err)
	}
	if base64.StdEncoding.EncodeToString(decoded) != value {
		return nil, fmt.Errorf("schema ambiguity: %s is not canonical padded base64", path)
	}
	return decoded, nil
}

func parseCoins(raw json.RawMessage, path string) ([]Coin, error) {
	entries, err := decodeArray(raw, path)
	if err != nil {
		return nil, err
	}
	coins := make([]Coin, len(entries))
	seen := make(map[string]struct{}, len(entries))
	previousDenom := ""
	for i, entry := range entries {
		location := fmt.Sprintf("%s[%d]", path, i)
		object, err := decodeObject(entry, location, []string{"denom", "amount"}, []string{"denom", "amount"})
		if err != nil {
			return nil, err
		}
		denom, err := decodeString(object["denom"], location+".denom", false)
		if err != nil {
			return nil, err
		}
		if !coinDenomPattern.MatchString(denom) {
			return nil, fmt.Errorf("schema ambiguity: %s.denom is not a Cosmos SDK v0.50 denomination", location)
		}
		if _, duplicate := seen[denom]; duplicate {
			return nil, fmt.Errorf("schema ambiguity: duplicate denomination %q at %s", denom, path)
		}
		if previousDenom != "" && denom <= previousDenom {
			return nil, fmt.Errorf("schema ambiguity: denominations at %s are not strictly sorted", path)
		}
		seen[denom] = struct{}{}
		previousDenom = denom
		amountText, err := decodeString(object["amount"], location+".amount", false)
		if err != nil {
			return nil, err
		}
		if len(amountText) > 78 {
			return nil, fmt.Errorf("schema ambiguity: %s.amount exceeds the Cosmos SDK 256-bit integer bound", location)
		}
		amount, ok := new(big.Int).SetString(amountText, 10)
		if !ok || amount.Sign() <= 0 || amount.String() != amountText {
			return nil, fmt.Errorf("schema ambiguity: %s.amount must be a canonical positive integer string", location)
		}
		if amount.BitLen() > 256 {
			return nil, fmt.Errorf("schema ambiguity: %s.amount exceeds the Cosmos SDK 256-bit integer bound", location)
		}
		coins[i] = Coin{Denom: denom, Amount: amountText}
	}
	return coins, nil
}
