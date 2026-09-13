package types

import (
	"fmt"
	"math"

	"google.golang.org/protobuf/encoding/protowire"
)

const (
	ClaimFundingTermsField protowire.Number = 29
	RoundClaimRefundField  protowire.Number = 17
)

func ValidateRawClaimFunding(data []byte) error {
	_, err := RawFundSettlementFieldPresent(data, ClaimFundingTermsField)
	return err
}

func ValidateRawRoundRefund(data []byte) error {
	_, err := RawFundSettlementFieldPresent(data, RoundClaimRefundField)
	return err
}

// RawFundSettlementFieldPresent checks new singular message fields before
// protobuf decoding can merge duplicate messages or truncate numeric values.
// Unrelated historical fields keep their existing decoding contract.
func RawFundSettlementFieldPresent(data []byte, field protowire.Number) (bool, error) {
	if len(data) > 64<<20 || (field != ClaimFundingTermsField && field != RoundClaimRefundField) {
		return false, fmt.Errorf("invalid funding record bound or field")
	}
	present := false
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return false, fmt.Errorf("malformed funding record tag")
		}
		data = data[n:]
		if num == field {
			if present || typ != protowire.BytesType || n != protowire.SizeTag(num) {
				return false, fmt.Errorf("duplicate, noncanonical or wrong-wire funding field")
			}
			value, size := protowire.ConsumeBytes(data)
			if size < 0 || size-len(value) != protowire.SizeVarint(uint64(len(value))) {
				return false, fmt.Errorf("invalid funding field length")
			}
			if err := validateRawFundMessage(value, field); err != nil {
				return false, err
			}
			present = true
			data = data[size:]
			continue
		}
		if typ == protowire.StartGroupType || typ == protowire.EndGroupType {
			return false, fmt.Errorf("unsupported group in funding record")
		}
		size := protowire.ConsumeFieldValue(num, typ, data)
		if size < 0 {
			return false, fmt.Errorf("malformed funding record field")
		}
		data = data[size:]
	}
	return present, nil
}

func validateRawFundMessage(data []byte, outer protowire.Number) error {
	if len(data) > 4096 {
		return fmt.Errorf("funding instruction exceeds byte bound")
	}
	seen := make(map[protowire.Number]bool, 6)
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		maxField := protowire.Number(6)
		if outer == RoundClaimRefundField {
			maxField = 4
		}
		if n < 0 || num < 1 || num > maxField || seen[num] || n != protowire.SizeTag(num) {
			return fmt.Errorf("unknown, duplicate or malformed funding instruction field")
		}
		seen[num] = true
		data = data[n:]
		isNumber := (outer == ClaimFundingTermsField && num <= 2) || (outer == RoundClaimRefundField && num >= 3)
		if isNumber {
			if typ != protowire.VarintType {
				return fmt.Errorf("wrong-wire funding instruction number")
			}
			value, size := protowire.ConsumeVarint(data)
			if size < 0 || size != protowire.SizeVarint(value) {
				return fmt.Errorf("noncanonical funding instruction number")
			}
			if outer == ClaimFundingTermsField && (value > math.MaxUint32 || (num == 1 && value != uint64(ClaimFundingPolicyV1)) || (num == 2 && value != 1 && value != 2)) {
				return fmt.Errorf("unsupported raw funding policy or kind")
			}
			data = data[size:]
		} else {
			if typ != protowire.BytesType {
				return fmt.Errorf("wrong-wire funding instruction text")
			}
			value, size := protowire.ConsumeBytes(data)
			if size < 0 || size-len(value) != protowire.SizeVarint(uint64(len(value))) {
				return fmt.Errorf("noncanonical funding instruction text length")
			}
			data = data[size:]
		}
	}
	return nil
}
