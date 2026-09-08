package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strconv"
	"strings"
	"unicode/utf8"
)

func decodeSnapshot(data []byte) (snapshot, error) {
	var s snapshot
	if len(data) == 0 || len(data) > maxInputBytes || !utf8.Valid(data) {
		return s, errors.New("snapshot must be bounded, nonempty UTF-8 JSON")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := walkSnapshotJSON(decoder, reflect.TypeOf(s), 0); err != nil {
		return s, fmt.Errorf("snapshot JSON: %w", err)
	}
	if _, err := decoder.Token(); !errors.Is(err, io.EOF) {
		return s, errors.New("snapshot must contain exactly one JSON value")
	}
	// The token pass enforces exact case-sensitive field names, presence and
	// null handling; DisallowUnknownFields alone cannot enforce those properties.
	decoder = json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&s); err != nil {
		return s, errors.New("snapshot fields do not match snapshot-v3 types")
	}
	return s, nil
}

// A bounded token pass, following relaunch-snapshot's duplicate-key walk.
// Struct tags are the single field list for this one wire format. This accepts
// only its structs, slices, strings, booleans and nonnegative integer numbers;
// it is not an extensible schema loader and has no dynamic input-driven types.
func walkSnapshotJSON(d *json.Decoder, shape reflect.Type, depth int) error {
	if depth > maxDepth {
		return errors.New("JSON depth limit exceeded")
	}
	token, err := d.Token()
	if err != nil {
		return errors.New("malformed JSON")
	}
	switch shape.Kind() {
	case reflect.Struct:
		if token != json.Delim('{') {
			return errors.New("expected object, not null or another type")
		}
		fields := make(map[string]reflect.StructField, shape.NumField())
		for i := 0; i < shape.NumField(); i++ {
			field := shape.Field(i)
			name, _, _ := strings.Cut(field.Tag.Get("json"), ",")
			fields[name] = field
		}
		seen := make(map[string]bool, len(fields))
		for d.More() {
			keyToken, err := d.Token()
			if err != nil {
				return errors.New("malformed JSON object key")
			}
			key, ok := keyToken.(string)
			if !ok || len(key) > maxStringBytes {
				return errors.New("invalid or oversized JSON object key")
			}
			if seen[key] {
				return errors.New("duplicate JSON object key")
			}
			seen[key] = true
			field, ok := fields[key]
			if !ok {
				return errors.New("unknown or incorrectly cased snapshot-v3 field")
			}
			if err := walkSnapshotJSON(d, field.Type, depth+1); err != nil {
				return fmt.Errorf("%s: %w", key, err)
			}
		}
		// Iterate the struct, not the map, so the first missing-field error is stable.
		for i := 0; i < shape.NumField(); i++ {
			field := shape.Field(i)
			name, option, _ := strings.Cut(field.Tag.Get("json"), ",")
			if !seen[name] && option != "omitempty" {
				return fmt.Errorf("missing required field %s", name)
			}
		}
		if end, err := d.Token(); err != nil || end != json.Delim('}') {
			return errors.New("malformed object end")
		}
	case reflect.Slice:
		// snapshot-v3's capture uses nil slices for empty inventories.
		if token == nil {
			return nil
		}
		if token != json.Delim('[') {
			return errors.New("expected array or null empty inventory")
		}
		limit := maxOwners
		if shape.Elem() == reflect.TypeOf(validator{}) {
			limit = maxValidators
		}
		for count := 0; d.More(); count++ {
			if count >= limit {
				return errors.New("JSON row limit exceeded")
			}
			if err := walkSnapshotJSON(d, shape.Elem(), depth+1); err != nil {
				return fmt.Errorf("row %d: %w", count, err)
			}
		}
		if end, err := d.Token(); err != nil || end != json.Delim(']') {
			return errors.New("malformed array end")
		}
	case reflect.String:
		value, ok := token.(string)
		if !ok || len(value) > maxStringBytes || strings.ContainsRune(value, utf8.RuneError) {
			return errors.New("expected bounded string without invalid Unicode or replacement characters")
		}
	case reflect.Bool:
		if _, ok := token.(bool); !ok {
			return errors.New("expected explicit boolean")
		}
	case reflect.Int, reflect.Int64:
		value, ok := token.(json.Number)
		if !ok {
			return errors.New("expected integer number")
		}
		integer, err := strconv.ParseInt(string(value), 10, 64)
		if err != nil || integer < 0 || strconv.FormatInt(integer, 10) != string(value) {
			return errors.New("expected canonical nonnegative int64 number")
		}
	default:
		return errors.New("unsupported internal wire type")
	}
	return nil
}
