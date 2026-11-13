package types

import (
	"fmt"

	"google.golang.org/protobuf/encoding/protowire"
)

const (
	RoundCommitsField            protowire.Number = 6
	RoundSeatTermsField          protowire.Number = 18
	RoundSeatDepositReturnsField protowire.Number = 19
	CommitSeatDepositField       protowire.Number = 4
)

// seatDepositMessageBound bounds one encoded seat-terms message, one encoded
// returns plan and one encoded return. 512 returns of two bech32 addresses
// and amounts stay well inside it.
const seatDepositMessageBound = 256 << 10

// ValidateRawRoundSeatDeposit checks the seat-deposit fields of an encoded
// round before protobuf decoding can merge duplicate messages, accept a
// noncanonical or empty encoding, or keep unknown inner fields. Field 18
// (seat terms) and field 19 (returns plan) appear at most once, as nonempty,
// canonically delimited messages whose inner fields are known, appear once
// (returns repeat) and use their own wire types. In each commitment (field 6)
// field 4 (the seat's deposit) appears at most once, nonempty and canonically
// delimited. Unrelated fields keep their existing decoding contract; a
// commitment's other fields are skipped, not judged. It costs no gas.
func ValidateRawRoundSeatDeposit(data []byte) error {
	if len(data) > 64<<20 {
		return fmt.Errorf("seat deposit record exceeds bounds")
	}
	var terms, returns bool
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("malformed seat deposit record tag")
		}
		data = data[n:]
		switch {
		case num == RoundSeatTermsField || num == RoundSeatDepositReturnsField:
			if (num == RoundSeatTermsField && terms) || (num == RoundSeatDepositReturnsField && returns) {
				return fmt.Errorf("duplicate seat deposit field %d", num)
			}
			value, size, err := consumeCanonicalSeatMessage(data, typ, n, num)
			if err != nil {
				return err
			}
			if num == RoundSeatTermsField {
				err = validateRawSeatTerms(value)
				terms = true
			} else {
				err = validateRawSeatDepositReturns(value)
				returns = true
			}
			if err != nil {
				return err
			}
			data = data[size:]
		case num == RoundCommitsField && typ == protowire.BytesType:
			value, size := protowire.ConsumeBytes(data)
			if size < 0 {
				return fmt.Errorf("malformed seat deposit record field")
			}
			if err := validateRawCommitSeatDeposit(value); err != nil {
				return err
			}
			data = data[size:]
		default:
			if typ == protowire.StartGroupType || typ == protowire.EndGroupType {
				return fmt.Errorf("unsupported group in seat deposit record")
			}
			size := protowire.ConsumeFieldValue(num, typ, data)
			if size < 0 {
				return fmt.Errorf("malformed seat deposit record field")
			}
			data = data[size:]
		}
	}
	return nil
}

// consumeCanonicalSeatMessage reads one length-delimited value whose tag was
// n bytes long, refusing a wrong wire type, a noncanonical tag or length, an
// empty value and an oversized one.
func consumeCanonicalSeatMessage(data []byte, typ protowire.Type, n int, num protowire.Number) ([]byte, int, error) {
	if typ != protowire.BytesType || n != protowire.SizeTag(num) {
		return nil, 0, fmt.Errorf("noncanonical or wrong-wire seat deposit field %d", num)
	}
	value, size := protowire.ConsumeBytes(data)
	if size < 0 || size-len(value) != protowire.SizeVarint(uint64(len(value))) {
		return nil, 0, fmt.Errorf("invalid seat deposit field %d length", num)
	}
	if len(value) == 0 || len(value) > seatDepositMessageBound {
		return nil, 0, fmt.Errorf("empty or oversized seat deposit field %d", num)
	}
	return value, size, nil
}

// walkRawSeatMessage visits each inner field of a seat-deposit message. kinds
// names every known field and its wire type; repeated lists fields that may
// appear more than once. Unknown fields, groups, noncanonical tags and
// lengths, varints and duplicates are refused.
func walkRawSeatMessage(data []byte, kinds map[protowire.Number]protowire.Type, repeated map[protowire.Number]bool, visit func(protowire.Number, []byte) error) error {
	seen := make(map[protowire.Number]bool, len(kinds))
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("malformed seat deposit field tag")
		}
		want, known := kinds[num]
		if !known || typ != want || n != protowire.SizeTag(num) || (seen[num] && !repeated[num]) {
			return fmt.Errorf("unknown, duplicate or malformed seat deposit inner field %d", num)
		}
		seen[num] = true
		data = data[n:]
		switch typ {
		case protowire.VarintType:
			value, size := protowire.ConsumeVarint(data)
			if size < 0 || size != protowire.SizeVarint(value) {
				return fmt.Errorf("noncanonical seat deposit number")
			}
			data = data[size:]
		default:
			value, size := protowire.ConsumeBytes(data)
			if size < 0 || size-len(value) != protowire.SizeVarint(uint64(len(value))) {
				return fmt.Errorf("noncanonical seat deposit field length")
			}
			if visit != nil {
				if err := visit(num, value); err != nil {
					return err
				}
			}
			data = data[size:]
		}
	}
	return nil
}

func validateRawSeatTerms(data []byte) error {
	return walkRawSeatMessage(data, map[protowire.Number]protowire.Type{1: protowire.BytesType}, nil, nil)
}

func validateRawSeatDepositReturns(data []byte) error {
	kinds := map[protowire.Number]protowire.Type{1: protowire.VarintType, 2: protowire.BytesType}
	return walkRawSeatMessage(data, kinds, map[protowire.Number]bool{2: true}, func(num protowire.Number, value []byte) error {
		if num != 2 {
			return nil
		}
		if len(value) == 0 || len(value) > seatDepositMessageBound {
			return fmt.Errorf("empty or oversized seat deposit return")
		}
		returnKinds := map[protowire.Number]protowire.Type{1: protowire.BytesType, 2: protowire.BytesType, 3: protowire.VarintType}
		return walkRawSeatMessage(value, returnKinds, nil, nil)
	})
}

// validateRawCommitSeatDeposit checks only field 4 of one commitment; every
// other field keeps the decoding contract it had before the field existed.
func validateRawCommitSeatDeposit(data []byte) error {
	seen := false
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("malformed commitment tag")
		}
		data = data[n:]
		if num == CommitSeatDepositField {
			if seen {
				return fmt.Errorf("duplicate commitment seat deposit")
			}
			if _, _, err := consumeCanonicalSeatMessage(data, typ, n, num); err != nil {
				return fmt.Errorf("commitment seat deposit: %w", err)
			}
			seen = true
		}
		size := protowire.ConsumeFieldValue(num, typ, data)
		if size < 0 {
			return fmt.Errorf("malformed commitment field")
		}
		data = data[size:]
	}
	return nil
}
