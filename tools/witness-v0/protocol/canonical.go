package protocol

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strconv"
	"unicode/utf8"
)

const (
	MaxDocumentBytes          = 1 << 20
	MaxJSONDepth              = 32
	MaxObjectMembers          = 256
	MaxArrayElements          = 4096
	MaxStringBytes            = 64 << 10
	MaxSafeJSONInteger uint64 = 9007199254740991
)

var (
	errDocumentTooLarge = errors.New("JSON document exceeds 1 MiB")
	errDepthExceeded    = errors.New("JSON nesting depth exceeds 32")
)

// CanonicalJSON parses a deliberately small interoperable JSON profile and
// returns its unique byte representation. Objects have unique keys sorted by
// UTF-8 bytes. Whitespace is omitted. Strings use the shortest JSON escapes.
// Bare numbers are limited to canonical, non-negative JavaScript-safe integers;
// protocol uint64 counters are decimal strings so JavaScript loses no precision.
func CanonicalJSON(input []byte) ([]byte, error) {
	if len(input) == 0 {
		return nil, errors.New("empty JSON document")
	}
	if len(input) > MaxDocumentBytes {
		return nil, errDocumentTooLarge
	}
	if !utf8.Valid(input) {
		return nil, errors.New("JSON document is not valid UTF-8")
	}
	if err := validateStringEscapes(input); err != nil {
		return nil, err
	}

	dec := json.NewDecoder(bytes.NewReader(input))
	dec.UseNumber()
	value, err := parseJSONValue(dec, 1)
	if err != nil {
		return nil, err
	}
	if tok, err := dec.Token(); err != io.EOF {
		if err == nil {
			return nil, fmt.Errorf("trailing JSON token %v", tok)
		}
		return nil, fmt.Errorf("trailing JSON data: %w", err)
	}

	var out bytes.Buffer
	if err := appendCanonical(&out, value); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

func parseJSONValue(dec *json.Decoder, depth int) (any, error) {
	if depth > MaxJSONDepth {
		return nil, errDepthExceeded
	}
	tok, err := dec.Token()
	if err != nil {
		return nil, fmt.Errorf("read JSON token: %w", err)
	}
	switch v := tok.(type) {
	case json.Delim:
		switch v {
		case '{':
			object := make(map[string]any)
			for dec.More() {
				if len(object) >= MaxObjectMembers {
					return nil, fmt.Errorf("object exceeds %d members", MaxObjectMembers)
				}
				keyToken, err := dec.Token()
				if err != nil {
					return nil, fmt.Errorf("read object key: %w", err)
				}
				key, ok := keyToken.(string)
				if !ok {
					return nil, errors.New("object key is not a string")
				}
				if len(key) > MaxStringBytes {
					return nil, fmt.Errorf("object key exceeds %d bytes", MaxStringBytes)
				}
				if bytes.IndexByte([]byte(key), 0) >= 0 {
					return nil, errors.New("object key contains U+0000")
				}
				if _, exists := object[key]; exists {
					return nil, fmt.Errorf("duplicate object key %q", key)
				}
				child, err := parseJSONValue(dec, depth+1)
				if err != nil {
					return nil, fmt.Errorf("key %q: %w", key, err)
				}
				object[key] = child
			}
			end, err := dec.Token()
			if err != nil || end != json.Delim('}') {
				return nil, errors.New("unterminated JSON object")
			}
			return object, nil
		case '[':
			array := make([]any, 0)
			for dec.More() {
				if len(array) >= MaxArrayElements {
					return nil, fmt.Errorf("array exceeds %d elements", MaxArrayElements)
				}
				child, err := parseJSONValue(dec, depth+1)
				if err != nil {
					return nil, fmt.Errorf("array element %d: %w", len(array), err)
				}
				array = append(array, child)
			}
			end, err := dec.Token()
			if err != nil || end != json.Delim(']') {
				return nil, errors.New("unterminated JSON array")
			}
			return array, nil
		default:
			return nil, fmt.Errorf("unexpected JSON delimiter %q", v)
		}
	case string:
		if len(v) > MaxStringBytes {
			return nil, fmt.Errorf("string exceeds %d bytes", MaxStringBytes)
		}
		if bytes.IndexByte([]byte(v), 0) >= 0 {
			return nil, errors.New("string contains U+0000")
		}
		return v, nil
	case json.Number:
		if !isCanonicalJSONUint(string(v)) {
			return nil, fmt.Errorf("number %q is not a canonical non-negative safe integer", v)
		}
		return v, nil
	case bool:
		return v, nil
	case nil:
		return nil, nil
	default:
		return nil, fmt.Errorf("unsupported JSON token type %T", tok)
	}
}

func validateStringEscapes(input []byte) error {
	inString := false
	for i := 0; i < len(input); i++ {
		switch input[i] {
		case '"':
			inString = !inString
		case '\\':
			if !inString {
				continue
			}
			i++
			if i >= len(input) {
				return errors.New("unterminated JSON escape")
			}
			if input[i] != 'u' {
				continue
			}
			first, ok := decodeHex16(input, i+1)
			if !ok {
				return errors.New("invalid JSON Unicode escape")
			}
			i += 4
			if first == 0 {
				return errors.New("JSON string contains escaped U+0000")
			}
			if first >= 0xdc00 && first <= 0xdfff {
				return errors.New("JSON string contains lone low surrogate")
			}
			if first >= 0xd800 && first <= 0xdbff {
				if i+6 >= len(input) || input[i+1] != '\\' || input[i+2] != 'u' {
					return errors.New("JSON string contains lone high surrogate")
				}
				second, ok := decodeHex16(input, i+3)
				if !ok || second < 0xdc00 || second > 0xdfff {
					return errors.New("JSON string contains unpaired high surrogate")
				}
				i += 6
			}
		}
	}
	return nil
}

func decodeHex16(input []byte, start int) (uint16, bool) {
	if start < 0 || start+4 > len(input) {
		return 0, false
	}
	var value uint16
	for _, b := range input[start : start+4] {
		value <<= 4
		switch {
		case b >= '0' && b <= '9':
			value |= uint16(b - '0')
		case b >= 'a' && b <= 'f':
			value |= uint16(b-'a') + 10
		case b >= 'A' && b <= 'F':
			value |= uint16(b-'A') + 10
		default:
			return 0, false
		}
	}
	return value, true
}

func appendCanonical(out *bytes.Buffer, value any) error {
	switch v := value.(type) {
	case map[string]any:
		keys := make([]string, 0, len(v))
		for key := range v {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		out.WriteByte('{')
		for i, key := range keys {
			if i > 0 {
				out.WriteByte(',')
			}
			appendJSONString(out, key)
			out.WriteByte(':')
			if err := appendCanonical(out, v[key]); err != nil {
				return err
			}
		}
		out.WriteByte('}')
	case []any:
		out.WriteByte('[')
		for i := range v {
			if i > 0 {
				out.WriteByte(',')
			}
			if err := appendCanonical(out, v[i]); err != nil {
				return err
			}
		}
		out.WriteByte(']')
	case string:
		appendJSONString(out, v)
	case json.Number:
		out.WriteString(string(v))
	case bool:
		if v {
			out.WriteString("true")
		} else {
			out.WriteString("false")
		}
	case nil:
		out.WriteString("null")
	default:
		return fmt.Errorf("cannot canonicalize %T", value)
	}
	return nil
}

func appendJSONString(out *bytes.Buffer, value string) {
	out.WriteByte('"')
	for _, r := range value {
		switch r {
		case '"', '\\':
			out.WriteByte('\\')
			out.WriteRune(r)
		case '\b':
			out.WriteString(`\b`)
		case '\f':
			out.WriteString(`\f`)
		case '\n':
			out.WriteString(`\n`)
		case '\r':
			out.WriteString(`\r`)
		case '\t':
			out.WriteString(`\t`)
		default:
			if r < 0x20 {
				out.WriteString(`\u00`)
				const hex = "0123456789abcdef"
				out.WriteByte(hex[byte(r)>>4])
				out.WriteByte(hex[byte(r)&0x0f])
			} else {
				out.WriteRune(r)
			}
		}
	}
	out.WriteByte('"')
}

func strictUnmarshal(canonical []byte, dst any) error {
	dec := json.NewDecoder(bytes.NewReader(canonical))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return err
	}
	if err := dec.Decode(new(any)); err != io.EOF {
		if err == nil {
			return errors.New("trailing JSON value")
		}
		return err
	}
	return nil
}

func isCanonicalUint(value string, allowZero bool) bool {
	if value == "0" {
		return allowZero
	}
	if len(value) == 0 || len(value) > 20 || value[0] < '1' || value[0] > '9' {
		return false
	}
	for i := 1; i < len(value); i++ {
		if value[i] < '0' || value[i] > '9' {
			return false
		}
	}
	_, err := strconv.ParseUint(value, 10, 64)
	return err == nil
}

func isCanonicalJSONUint(value string) bool {
	if !isCanonicalUint(value, true) {
		return false
	}
	parsed, err := strconv.ParseUint(value, 10, 64)
	return err == nil && parsed <= MaxSafeJSONInteger
}
