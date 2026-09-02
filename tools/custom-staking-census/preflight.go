package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"google.golang.org/protobuf/encoding/protowire"
)

const (
	maxJSONTokens               = 65_536
	maxJSONArrayElements        = 4_096
	maxJSONDepth                = 32
	maxJSONStringBytes          = 64 << 10
	maxJSONNumberBytes          = 128
	maxSDKValidatorFieldBytes   = 64 << 10
	maxSDKValidatorUnbondingIDs = 65_536
)

var errDecodeResourceLimit = errors.New("decode resource limit exceeded")

func decodeResourceLimitf(format string, args ...any) error {
	return fmt.Errorf("%w: %s", errDecodeResourceLimit, fmt.Sprintf(format, args...))
}

type jsonContainer struct {
	delimiter json.Delim
	elements  int
}

// preflightJSON tokenizes a bounded raw value before decoding it into generated
// structs. This prevents a compact array such as [{},{},...] from expanding
// into an unbounded typed slice before its semantic cardinality is checked.
func preflightJSON(raw []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()

	var (
		containers     []jsonContainer
		tokenCount     int
		topLevelValues int
	)
	for {
		token, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return fmt.Errorf("tokenize JSON: %w", err)
		}
		tokenCount++
		if tokenCount > maxJSONTokens {
			return decodeResourceLimitf("JSON exceeds %d tokens", maxJSONTokens)
		}

		if delimiter, ok := token.(json.Delim); ok {
			switch delimiter {
			case '{', '[':
				if len(containers) == 0 {
					topLevelValues++
				} else if err := countJSONArrayElement(containers); err != nil {
					return err
				}
				containers = append(containers, jsonContainer{delimiter: delimiter})
				if len(containers) > maxJSONDepth {
					return decodeResourceLimitf("JSON exceeds nesting depth %d", maxJSONDepth)
				}
			case '}', ']':
				if len(containers) == 0 {
					return errors.New("JSON contains an unmatched closing delimiter")
				}
				containers = containers[:len(containers)-1]
			default:
				return fmt.Errorf("JSON contains unsupported delimiter %q", delimiter)
			}
			continue
		}

		if len(containers) == 0 {
			topLevelValues++
		} else if err := countJSONArrayElement(containers); err != nil {
			return err
		}
		switch value := token.(type) {
		case string:
			if len(value) > maxJSONStringBytes {
				return decodeResourceLimitf("JSON string exceeds %d decoded bytes", maxJSONStringBytes)
			}
		case json.Number:
			if len(value.String()) > maxJSONNumberBytes {
				return decodeResourceLimitf("JSON number exceeds %d bytes", maxJSONNumberBytes)
			}
		}
	}
	if len(containers) != 0 {
		return errors.New("JSON has an unclosed container")
	}
	if topLevelValues != 1 {
		return fmt.Errorf("JSON must contain exactly one top-level value, got %d", topLevelValues)
	}
	return nil
}

func countJSONArrayElement(containers []jsonContainer) error {
	index := len(containers) - 1
	if containers[index].delimiter != '[' {
		return nil
	}
	containers[index].elements++
	if containers[index].elements > maxJSONArrayElements {
		return decodeResourceLimitf("JSON array exceeds %d elements", maxJSONArrayElements)
	}
	return nil
}

// preflightCommitInfoProto bounds repeated StoreInfo allocation before the
// generated protobuf unmarshaler is allowed to construct its slice.
func preflightCommitInfoProto(raw []byte) error {
	var (
		seenVersion   bool
		seenTimestamp bool
		storeCount    int
	)
	for len(raw) > 0 {
		number, wireType, consumed, err := consumeProtoTag(raw, "root commit info")
		if err != nil {
			return err
		}
		raw = raw[consumed:]
		switch number {
		case 1:
			if seenVersion {
				return errors.New("root commit info repeats version")
			}
			seenVersion = true
			consumed, err = consumeProtoVarint(raw, wireType, "root commit info version")
		case 2:
			var payload []byte
			payload, consumed, err = consumeProtoBytes(raw, wireType, "root commit store info")
			if err == nil {
				storeCount++
				if storeCount > maxCommitInfoStores {
					return decodeResourceLimitf("root commit info exceeds %d mounted stores", maxCommitInfoStores)
				}
				err = preflightStoreInfoProto(payload)
			}
		case 3:
			if seenTimestamp {
				return errors.New("root commit info repeats timestamp")
			}
			seenTimestamp = true
			_, consumed, err = consumeProtoBytes(raw, wireType, "root commit timestamp")
		default:
			return fmt.Errorf("root commit info contains unknown protobuf field %d", number)
		}
		if err != nil {
			return err
		}
		raw = raw[consumed:]
	}
	return nil
}

func preflightStoreInfoProto(raw []byte) error {
	var seen [3]bool
	for len(raw) > 0 {
		number, wireType, consumed, err := consumeProtoTag(raw, "root StoreInfo")
		if err != nil {
			return err
		}
		raw = raw[consumed:]
		if number < 1 || number > 2 {
			return fmt.Errorf("root StoreInfo contains unknown protobuf field %d", number)
		}
		if seen[number] {
			return fmt.Errorf("root StoreInfo repeats protobuf field %d", number)
		}
		seen[number] = true
		switch number {
		case 1:
			var name []byte
			name, consumed, err = consumeProtoBytes(raw, wireType, "root StoreInfo name")
			if err == nil && len(name) > 128 {
				return decodeResourceLimitf("root StoreInfo name exceeds 128 bytes")
			}
		case 2:
			var commitID []byte
			commitID, consumed, err = consumeProtoBytes(raw, wireType, "root StoreInfo commit ID")
			if err == nil {
				err = preflightCommitIDProto(commitID)
			}
		}
		if err != nil {
			return err
		}
		raw = raw[consumed:]
	}
	return nil
}

func preflightCommitIDProto(raw []byte) error {
	var seen [3]bool
	for len(raw) > 0 {
		number, wireType, consumed, err := consumeProtoTag(raw, "root CommitID")
		if err != nil {
			return err
		}
		raw = raw[consumed:]
		if number < 1 || number > 2 {
			return fmt.Errorf("root CommitID contains unknown protobuf field %d", number)
		}
		if seen[number] {
			return fmt.Errorf("root CommitID repeats protobuf field %d", number)
		}
		seen[number] = true
		switch number {
		case 1:
			consumed, err = consumeProtoVarint(raw, wireType, "root CommitID version")
		case 2:
			var hash []byte
			hash, consumed, err = consumeProtoBytes(raw, wireType, "root CommitID hash")
			if err == nil && len(hash) > 32 {
				return decodeResourceLimitf("root CommitID hash exceeds 32 bytes")
			}
		}
		if err != nil {
			return err
		}
		raw = raw[consumed:]
	}
	return nil
}

// preflightSDKValidatorProto bounds the sole repeated field and rejects
// duplicate singular fields before the generated SDK unmarshaler allocates.
func preflightSDKValidatorProto(raw []byte) error {
	var (
		seen             [14]bool
		unbondingIDCount int
	)
	for len(raw) > 0 {
		number, wireType, consumed, err := consumeProtoTag(raw, "SDK validator")
		if err != nil {
			return err
		}
		raw = raw[consumed:]
		if number < 1 || number > 13 {
			return fmt.Errorf("SDK validator contains unknown protobuf field %d", number)
		}
		if number == 13 {
			var added int
			added, consumed, err = countUnbondingIDs(raw, wireType, maxSDKValidatorUnbondingIDs-unbondingIDCount)
			if err == nil {
				unbondingIDCount += added
			}
		} else {
			if seen[number] {
				return fmt.Errorf("SDK validator repeats singular protobuf field %d", number)
			}
			seen[number] = true
			switch number {
			case 1, 2, 5, 6, 7, 9, 10, 11:
				var payload []byte
				payload, consumed, err = consumeProtoBytes(raw, wireType, "SDK validator length-delimited field")
				if err == nil && len(payload) > maxSDKValidatorFieldBytes {
					return decodeResourceLimitf(
						"SDK validator singular field exceeds %d bytes",
						maxSDKValidatorFieldBytes,
					)
				}
			case 3, 4, 8, 12:
				consumed, err = consumeProtoVarint(raw, wireType, "SDK validator varint field")
			}
		}
		if err != nil {
			return err
		}
		raw = raw[consumed:]
	}
	return nil
}

func countUnbondingIDs(raw []byte, wireType protowire.Type, remaining int) (int, int, error) {
	if remaining < 0 {
		return 0, 0, decodeResourceLimitf(
			"SDK validator exceeds %d unbonding IDs",
			maxSDKValidatorUnbondingIDs,
		)
	}
	if wireType == protowire.VarintType {
		consumed, err := consumeProtoVarint(raw, wireType, "SDK validator unbonding ID")
		if err != nil {
			return 0, 0, err
		}
		if remaining == 0 {
			return 0, 0, decodeResourceLimitf(
				"SDK validator exceeds %d unbonding IDs",
				maxSDKValidatorUnbondingIDs,
			)
		}
		return 1, consumed, nil
	}
	if wireType != protowire.BytesType {
		return 0, 0, fmt.Errorf("SDK validator packed unbonding IDs use wire type %d", wireType)
	}
	payload, consumed, err := consumeProtoBytes(raw, wireType, "SDK validator packed unbonding IDs")
	if err != nil {
		return 0, 0, err
	}
	count := 0
	for len(payload) > 0 {
		if count >= remaining {
			return 0, 0, decodeResourceLimitf(
				"SDK validator exceeds %d unbonding IDs",
				maxSDKValidatorUnbondingIDs,
			)
		}
		_, itemBytes := protowire.ConsumeVarint(payload)
		if itemBytes < 0 {
			return 0, 0, fmt.Errorf("decode SDK validator packed unbonding ID: %w", protowire.ParseError(itemBytes))
		}
		payload = payload[itemBytes:]
		count++
	}
	return count, consumed, nil
}

func consumeProtoTag(raw []byte, context string) (protowire.Number, protowire.Type, int, error) {
	number, wireType, consumed := protowire.ConsumeTag(raw)
	if consumed < 0 {
		return 0, 0, 0, fmt.Errorf("decode %s protobuf tag: %w", context, protowire.ParseError(consumed))
	}
	return number, wireType, consumed, nil
}

func consumeProtoVarint(raw []byte, wireType protowire.Type, context string) (int, error) {
	if wireType != protowire.VarintType {
		return 0, fmt.Errorf("%s uses wire type %d, expected varint", context, wireType)
	}
	_, consumed := protowire.ConsumeVarint(raw)
	if consumed < 0 {
		return 0, fmt.Errorf("decode %s: %w", context, protowire.ParseError(consumed))
	}
	return consumed, nil
}

func consumeProtoBytes(raw []byte, wireType protowire.Type, context string) ([]byte, int, error) {
	if wireType != protowire.BytesType {
		return nil, 0, fmt.Errorf("%s uses wire type %d, expected length-delimited", context, wireType)
	}
	payload, consumed := protowire.ConsumeBytes(raw)
	if consumed < 0 {
		return nil, 0, fmt.Errorf("decode %s: %w", context, protowire.ParseError(consumed))
	}
	return payload, consumed, nil
}
