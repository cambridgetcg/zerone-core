package types

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Params:       DefaultParams(),
		Adapters:     nil,
		StateEntries: nil,
	}
}

// IsAllowedGenesisStateKey reports whether key belongs to replay/economic
// substrate_bridge state that may be round-tripped through GenesisState.
// Params and adapter records/indexes are intentionally absent: their typed
// genesis fields are the sole initialization path for those keyspaces.
func IsAllowedGenesisStateKey(key []byte) bool {
	if len(key) == 0 {
		return false
	}

	prefix := key[:1]
	switch {
	case bytes.Equal(prefix, LineageEdgePrefix),
		bytes.Equal(prefix, LineageByUpstreamPrefix),
		bytes.Equal(prefix, LineageByDownstreamPrefix),
		bytes.Equal(prefix, LineageRoyaltyAccumulatorPrefix),
		bytes.Equal(prefix, ExternalAttestationPrefix),
		bytes.Equal(prefix, PendingFactIndexPrefix),
		bytes.Equal(prefix, WitnessPendingRewardPrefix):
		return len(key) > 1
	case bytes.Equal(prefix, AttestationByStatusPrefix):
		// prefix | status byte | nonempty attestation id
		return len(key) > 2
	case bytes.Equal(prefix, AttestationPendingClaimsPrefix):
		// prefix | nonempty attestation id | 0x00 | nonempty claim id
		return len(key) > 3
	case bytes.Equal(key, AttestationIDCounterKey), bytes.Equal(key, DedupeArmedKey):
		return len(key) == 1
	case bytes.Equal(prefix, WitnessDeadlineIndexPrefix):
		// prefix | uint64 deadline | nonempty attestation id
		return len(key) > 9
	case bytes.Equal(prefix, SourceRefPrefix):
		// prefix | uint32 adapter-id length | adapter id | nonempty source id
		if len(key) <= 5 {
			return false
		}
		adapterLen := int(binary.BigEndian.Uint32(key[1:5]))
		return adapterLen > 0 && len(key) > 5+adapterLen
	default:
		return false
	}
}

// IsTypedGenesisStateKey reports whether key is already represented by a typed
// GenesisState field and therefore must not also appear in StateEntries.
func IsTypedGenesisStateKey(key []byte) bool {
	if len(key) == 0 {
		return false
	}
	if bytes.Equal(key, ParamsKey) {
		return true
	}
	prefix := key[:1]
	return (bytes.Equal(prefix, AdapterRegistrationPrefix) && len(key) > 1) ||
		(bytes.Equal(prefix, AdapterByStatusPrefix) && len(key) > 2)
}

func (gs *GenesisState) Validate() error {
	if gs == nil {
		return fmt.Errorf("genesis state must not be nil")
	}
	if gs.Params == nil {
		return fmt.Errorf("params must not be nil")
	}
	if err := gs.Params.Validate(); err != nil {
		return err
	}
	seen := map[string]bool{}
	for index, a := range gs.Adapters {
		if a == nil {
			return fmt.Errorf("adapter at index %d must not be nil", index)
		}
		if a.AdapterId == "" {
			return fmt.Errorf("adapter at index %d must have a non-empty adapter_id", index)
		}
		if seen[a.AdapterId] {
			return fmt.Errorf("duplicate adapter_id in genesis: %s", a.AdapterId)
		}
		seen[a.AdapterId] = true
	}

	seenStateKeys := make(map[string]struct{}, len(gs.StateEntries))
	for index, entry := range gs.StateEntries {
		if entry == nil {
			return fmt.Errorf("state entry at index %d must not be nil", index)
		}
		if !IsAllowedGenesisStateKey(entry.Key) {
			return fmt.Errorf("state entry at index %d uses forbidden or malformed key %x", index, entry.Key)
		}
		if len(entry.Value) == 0 {
			return fmt.Errorf("state entry at index %d has an empty value", index)
		}
		key := string(entry.Key)
		if _, duplicate := seenStateKeys[key]; duplicate {
			return fmt.Errorf("duplicate state entry key: %x", entry.Key)
		}
		seenStateKeys[key] = struct{}{}
	}
	return validateGenesisStateEntries(gs)
}
