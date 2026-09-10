package types

import (
	"fmt"
	"math"

	"google.golang.org/protobuf/encoding/protowire"
)

const ClaimReviewPolicyField protowire.Number = 28
const RoundReviewPolicyField protowire.Number = 16

// ValidateRawPolicyField prevents protobuf's uint32 truncation and last-field
// wins behavior from turning a corrupt or unknown policy into legacy policy 0.
func ValidateRawPolicyField(data []byte, field protowire.Number) error {
	_, err := RawPolicyFieldPresent(data, field)
	return err
}

// RawPolicyFieldPresent validates the wire representation and reports explicit
// field presence. An exact predecessor that predates a field may require its
// absence, including an explicitly encoded zero. The byte ceiling matches the
// existing per-namespace record inventory; proto3 records contain no groups.
func RawPolicyFieldPresent(data []byte, field protowire.Number) (bool, error) {
	if len(data) > 64<<20 || field <= 0 {
		return false, fmt.Errorf("policy record exceeds bounds or has invalid field number")
	}
	present := false
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return false, fmt.Errorf("malformed policy record tag")
		}
		data = data[n:]
		if num == field {
			if present || typ != protowire.VarintType {
				return false, fmt.Errorf("duplicate or wrong-wire policy field %d", field)
			}
			value, size := protowire.ConsumeVarint(data)
			if size < 0 || value > math.MaxUint32 || protowire.SizeVarint(value) != size {
				return false, fmt.Errorf("invalid uint32 policy field %d", field)
			}
			if err := ValidateReviewPolicyVersion(uint32(value), true); err != nil {
				return false, err
			}
			present = true
			data = data[size:]
			continue
		}
		if typ == protowire.StartGroupType || typ == protowire.EndGroupType {
			return false, fmt.Errorf("unsupported group in proto3 policy record")
		}
		size := protowire.ConsumeFieldValue(num, typ, data)
		if size < 0 {
			return false, fmt.Errorf("malformed policy record field")
		}
		data = data[size:]
	}
	return present, nil
}
