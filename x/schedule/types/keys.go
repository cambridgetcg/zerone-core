package types

import (
	"encoding/binary"
)

const (
	// ModuleName deliberately differs from the retired generic scheduler's
	// "schedule" namespace. Reusing that name could make an upgraded binary
	// interpret the old module's incompatible state and shared escrow account.
	ModuleName = "message_schedule"
	StoreKey   = ModuleName

	// CLIName preserves the concise public command while consensus state and
	// escrow remain isolated in the new module namespace.
	CLIName      = "schedule"
	RouterKey    = ModuleName
	QuerierRoute = ModuleName
)

var (
	ParamsKey           = []byte{0x00}
	ScheduleCounterKey  = []byte{0x01}
	TotalEscrowKey      = []byte{0x02}
	ScheduleKeyPrefix   = []byte{0x10}
	DueKeyPrefix        = []byte{0x11}
	CreatorKeyPrefix    = []byte{0x12}
	ActiveCreatorPrefix = []byte{0x13}
	ReceiptKeyPrefix    = []byte{0x20}
	OccurrenceKeyPrefix = []byte{0x21}
)

func ScheduleKey(id string) []byte {
	return join(ScheduleKeyPrefix, []byte(id))
}

func DueKey(height uint64, id string) []byte {
	heightBz := make([]byte, 8)
	binary.BigEndian.PutUint64(heightBz, height)
	return join(DueKeyPrefix, heightBz, []byte(id))
}

func DueThroughHeightEnd(height uint64) []byte {
	if height == ^uint64(0) {
		return prefixEndBytes(DueKeyPrefix)
	}
	heightBz := make([]byte, 8)
	binary.BigEndian.PutUint64(heightBz, height+1)
	return join(DueKeyPrefix, heightBz)
}

func CreatorPrefix(address []byte) []byte {
	return lengthPrefixed(CreatorKeyPrefix, address)
}

func CreatorKey(address []byte, id string) []byte {
	return join(CreatorPrefix(address), []byte(id))
}

func ActiveCreatorKey(address []byte, id string) []byte {
	return join(lengthPrefixed(ActiveCreatorPrefix, address), []byte(id))
}

func ActiveCreatorAddressPrefix(address []byte) []byte {
	return lengthPrefixed(ActiveCreatorPrefix, address)
}

func ReceiptSchedulePrefix(scheduleID string) []byte {
	return lengthPrefixed(ReceiptKeyPrefix, []byte(scheduleID))
}

func ReceiptKey(scheduleID string, sequence uint32) []byte {
	seqBz := make([]byte, 4)
	binary.BigEndian.PutUint32(seqBz, sequence)
	return join(ReceiptSchedulePrefix(scheduleID), seqBz)
}

func OccurrenceKey(occurrenceID string) []byte {
	return join(OccurrenceKeyPrefix, []byte(occurrenceID))
}

func ParseDueKey(key []byte) (uint64, string, bool) {
	if len(key) < len(DueKeyPrefix)+8 || key[0] != DueKeyPrefix[0] {
		return 0, "", false
	}
	return binary.BigEndian.Uint64(key[1:9]), string(key[9:]), true
}

func prefixEndBytes(prefix []byte) []byte {
	end := append([]byte(nil), prefix...)
	for i := len(end) - 1; i >= 0; i-- {
		end[i]++
		if end[i] != 0 {
			return end[:i+1]
		}
	}
	return nil
}

// PrefixEndBytes returns the exclusive iterator end for a byte prefix.
func PrefixEndBytes(prefix []byte) []byte {
	return prefixEndBytes(prefix)
}

func join(parts ...[]byte) []byte {
	length := 0
	for _, part := range parts {
		length += len(part)
	}
	out := make([]byte, 0, length)
	for _, part := range parts {
		out = append(out, part...)
	}
	return out
}

func lengthPrefixed(prefix, value []byte) []byte {
	lengthBz := make([]byte, 2)
	binary.BigEndian.PutUint16(lengthBz, uint16(len(value)))
	return join(prefix, lengthBz, value)
}
