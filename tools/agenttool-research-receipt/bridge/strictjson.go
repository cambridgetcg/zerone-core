// Package bridge implements the offline, zero-effect AgentTool Research
// Commons receipt projection. It deliberately has no network, RPC, wallet,
// database, signing, custody, or chain dependency.
package bridge

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode/utf8"
)

const (
	maxDocumentBytes = 64 << 10
	maxJSONDepth     = 16
	maxJSONNodes     = 4096
	maxSafeInteger   = int64(9_007_199_254_740_991)
)

// decodeClosed rejects ambiguous JSON before decoding it into a closed Go
// structure. RC-0.1 wire objects are ASCII-only: they contain identifiers,
// digests, enums, integers, and booleans, never research prose or raw evidence.
func decodeClosed(data []byte, destination any, nullablePaths ...string) error {
	if len(data) == 0 {
		return errors.New("document is empty")
	}
	if len(data) > maxDocumentBytes {
		return fmt.Errorf("document exceeds %d-byte limit", maxDocumentBytes)
	}
	if !utf8.Valid(data) {
		return errors.New("document is not valid UTF-8")
	}
	nullable := make(map[string]struct{}, len(nullablePaths))
	for _, path := range nullablePaths {
		nullable[path] = struct{}{}
	}
	if err := inspectJSON(data, nullable); err != nil {
		return err
	}

	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	decoder.UseNumber()
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

func inspectJSON(data []byte, nullable map[string]struct{}) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	nodes := 0

	var walk func(path string, depth int) error
	walk = func(path string, depth int) error {
		if depth > maxJSONDepth {
			return fmt.Errorf("JSON exceeds depth limit at %s", path)
		}
		nodes++
		if nodes > maxJSONNodes {
			return fmt.Errorf("JSON exceeds %d-node limit", maxJSONNodes)
		}

		token, err := decoder.Token()
		if err != nil {
			return fmt.Errorf("decode JSON token at %s: %w", path, err)
		}
		switch value := token.(type) {
		case nil:
			if _, ok := nullable[path]; !ok {
				return fmt.Errorf("JSON null is not allowed at %s", path)
			}
			return nil
		case string:
			if err := validateASCIIString(path, value); err != nil {
				return err
			}
			return nil
		case json.Number:
			if strings.HasPrefix(value.String(), "-") {
				return fmt.Errorf("number at %s must not be negative or negative zero", path)
			}
			integer, err := strconv.ParseInt(value.String(), 10, 64)
			if err != nil || integer < 0 || integer > maxSafeInteger {
				return fmt.Errorf("number at %s must be a non-negative safe integer", path)
			}
			return nil
		case bool:
			return nil
		case json.Delim:
			switch value {
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
					if err := validateASCIIString(path+".<key>", key); err != nil {
						return err
					}
					if _, exists := seen[key]; exists {
						return fmt.Errorf("duplicate JSON object key %q at %s", key, path)
					}
					seen[key] = struct{}{}
					if err := walk(path+"."+key, depth+1); err != nil {
						return err
					}
				}
				if _, err := decoder.Token(); err != nil {
					return fmt.Errorf("close JSON object at %s: %w", path, err)
				}
				return nil
			case '[':
				index := 0
				for decoder.More() {
					if err := walk(fmt.Sprintf("%s[%d]", path, index), depth+1); err != nil {
						return err
					}
					index++
				}
				if _, err := decoder.Token(); err != nil {
					return fmt.Errorf("close JSON array at %s: %w", path, err)
				}
				return nil
			default:
				return fmt.Errorf("unexpected JSON delimiter %q at %s", value, path)
			}
		default:
			return fmt.Errorf("unsupported JSON token %T at %s", token, path)
		}
	}

	if err := walk("$", 0); err != nil {
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

func validateASCIIString(path, value string) error {
	for _, character := range value {
		if character < 0x20 || character > 0x7e {
			return fmt.Errorf("string at %s must contain printable ASCII only", path)
		}
	}
	return nil
}

func requireExactFields(raw json.RawMessage, path string, fields ...string) (map[string]json.RawMessage, error) {
	var object map[string]json.RawMessage
	if err := json.Unmarshal(raw, &object); err != nil {
		return nil, fmt.Errorf("%s must be an object: %w", path, err)
	}
	if object == nil {
		return nil, fmt.Errorf("%s must be an object", path)
	}
	if len(object) != len(fields) {
		return nil, fmt.Errorf("%s must contain exactly %d fields", path, len(fields))
	}
	for _, field := range fields {
		if _, ok := object[field]; !ok {
			return nil, fmt.Errorf("%s is missing required field %q", path, field)
		}
	}
	return object, nil
}
