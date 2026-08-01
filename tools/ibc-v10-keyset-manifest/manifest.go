package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
)

const (
	planInfoSchema           = "zerone.sdk-0.53-ibc-10/legacy-ibc-keyset/v1"
	maxPlanInfoBytes         = 2048
	maxKeysPerDomain         = 100_000
	maxLogicalBytesPerDomain = 32 << 20
)

type keySetManifest struct {
	KeyCount   string `json:"key_count"`
	KeysSHA256 string `json:"keys_sha256"`
}

// planInfo deliberately mirrors app.sdk053IBC10PlanInfo. Field order is part
// of the canonical JSON contract accepted by the upgrade handler.
type planInfo struct {
	Schema               string         `json:"schema"`
	ChannelUpgrades      keySetManifest `json:"channel_upgrades"`
	PruningSequenceStart keySetManifest `json:"pruning_sequence_start"`
}

func buildPlanInfo(channelUpgradeKeys, pruningSequenceKeys [][]byte) ([]byte, error) {
	channelUpgradeKeys, err := normalizeManifestKeys(
		[]byte(channelUpgradesLogicalPrefix),
		channelUpgradeKeys,
	)
	if err != nil {
		return nil, err
	}
	pruningSequenceKeys, err = normalizeManifestKeys(
		[]byte(pruningSequenceLogicalPrefix),
		pruningSequenceKeys,
	)
	if err != nil {
		return nil, err
	}

	info := planInfo{
		Schema:               planInfoSchema,
		ChannelUpgrades:      makeKeySetManifest(channelUpgradeKeys),
		PruningSequenceStart: makeKeySetManifest(pruningSequenceKeys),
	}
	bz, err := json.Marshal(info)
	if err != nil {
		return nil, fmt.Errorf("marshal plan.Info: %w", err)
	}
	if len(bz) > maxPlanInfoBytes {
		return nil, fmt.Errorf("plan.Info exceeds %d bytes", maxPlanInfoBytes)
	}
	return bz, nil
}

func normalizeManifestKeys(prefix []byte, keys [][]byte) ([][]byte, error) {
	if len(keys) > maxKeysPerDomain {
		return nil, fmt.Errorf(
			"manifest for prefix %q exceeds %d keys",
			prefix,
			maxKeysPerDomain,
		)
	}

	normalized := make([][]byte, len(keys))
	totalBytes := 0
	for i, key := range keys {
		if !bytes.HasPrefix(key, prefix) {
			return nil, fmt.Errorf("manifest key %q is outside prefix %q", key, prefix)
		}
		if len(key) > maxLogicalBytesPerDomain-totalBytes {
			return nil, fmt.Errorf(
				"manifest for prefix %q exceeds %d aggregate logical key bytes",
				prefix,
				maxLogicalBytesPerDomain,
			)
		}
		totalBytes += len(key)
		normalized[i] = bytes.Clone(key)
	}

	sort.Slice(normalized, func(i, j int) bool {
		return bytes.Compare(normalized[i], normalized[j]) < 0
	})
	for i := 1; i < len(normalized); i++ {
		if bytes.Equal(normalized[i-1], normalized[i]) {
			return nil, fmt.Errorf(
				"manifest for prefix %q contains duplicate key %q",
				prefix,
				normalized[i],
			)
		}
	}
	return normalized, nil
}

func makeKeySetManifest(keys [][]byte) keySetManifest {
	return keySetManifest{
		KeyCount:   strconv.FormatUint(uint64(len(keys)), 10),
		KeysSHA256: hashLengthPrefixedKeys(keys),
	}
}

func hashLengthPrefixedKeys(keys [][]byte) string {
	hasher := sha256.New()
	var length [8]byte
	for _, key := range keys {
		binary.BigEndian.PutUint64(length[:], uint64(len(key)))
		_, _ = hasher.Write(length[:])
		_, _ = hasher.Write(key)
	}
	return hex.EncodeToString(hasher.Sum(nil))
}
