package app

import (
	"encoding/hex"
	"fmt"
	"os"
	"strings"

	creedtypes "github.com/zerone-chain/zerone/x/creed/types"
	workcreedtypes "github.com/zerone-chain/zerone/x/work_creed/types"
)

// LoadInceptionSubCreedPins materializes the inception-pin set for
// x/work_creed by joining three build-time inputs:
//
//  1. CanonicalLifecyclePhases (the 9 phase enum)
//  2. CanonicalSubCreeds       (per-phase commitment registry)
//  3. .sub-creed-hashes        (sha256 of each docs/sub_creeds/<phase>.md)
//
// The resulting slice is sorted by phase number ascending, with the
// Knowledge phase deliberately omitted (it delegates its sub-creed to
// docs/TRUTH_SEEKING.md → x/creed.PinnedCreed).
//
// Each pin is constructed at version 1, anchored_at_block 0, with an
// empty source_lip (no LIP precedes genesis; post-genesis amendments
// must cite their authorizing LIP per commitment 19).
//
// Returns an error if the hash file is missing, malformed, missing a
// required phase, or contains an entry for the Knowledge phase (which
// would violate the genesis validator).
//
// hashFilePath is relative to the working directory; callers (genesis
// populators, tests) typically pass ".sub-creed-hashes" when invoked
// from the repo root.
func LoadInceptionSubCreedPins(hashFilePath string) ([]*workcreedtypes.PinnedSubCreed, error) {
	raw, err := os.ReadFile(hashFilePath)
	if err != nil {
		return nil, fmt.Errorf("read sub-creed hash file %q: %w", hashFilePath, err)
	}
	hashes, err := parseSubCreedHashes(string(raw))
	if err != nil {
		return nil, err
	}

	pins := make([]*workcreedtypes.PinnedSubCreed, 0, 8)
	for _, phase := range creedtypes.CanonicalLifecyclePhases {
		if !phase.HasSubCreedDoc {
			// Knowledge: delegates to x/creed; assert absent from hash file.
			if _, present := hashes[phase.Name]; present {
				return nil, fmt.Errorf("phase %q has no sub-creed doc but appears in %s",
					phase.Name, hashFilePath)
			}
			continue
		}

		hexHash, ok := hashes[phase.Name]
		if !ok {
			return nil, fmt.Errorf("phase %q missing from %s", phase.Name, hashFilePath)
		}
		hashBytes, err := hex.DecodeString(hexHash)
		if err != nil {
			return nil, fmt.Errorf("phase %q: invalid hex hash %q: %w", phase.Name, hexHash, err)
		}
		if len(hashBytes) != 32 {
			return nil, fmt.Errorf("phase %q: hash must be 32 bytes (got %d)", phase.Name, len(hashBytes))
		}

		def, ok := creedtypes.SubCreedFor(phase.Number)
		if !ok {
			return nil, fmt.Errorf("phase %q (#%d) missing from CanonicalSubCreeds",
				phase.Name, phase.Number)
		}
		codes := make([]string, 0, len(def.Commitments))
		for _, c := range def.Commitments {
			codes = append(codes, c.Code)
		}

		pins = append(pins, &workcreedtypes.PinnedSubCreed{
			Phase:           uint32(phase.Number),
			PhaseName:       phase.Name,
			Version:         1,
			CanonicalHash:   hashBytes,
			AnchoredAtBlock: 0,
			SourceLip:       "",
			CommitmentCodes: codes,
		})
	}
	return pins, nil
}

// parseSubCreedHashes parses the .sub-creed-hashes file format:
// each non-empty, non-comment line is "<phase> <hex-sha256>".
func parseSubCreedHashes(content string) (map[string]string, error) {
	out := map[string]string{}
	for lineNo, line := range strings.Split(strings.TrimSpace(content), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		fields := strings.Fields(trimmed)
		if len(fields) != 2 {
			return nil, fmt.Errorf("line %d: malformed (want \"<phase> <hex>\", got %q)",
				lineNo+1, line)
		}
		phase, hexHash := fields[0], fields[1]
		if _, dup := out[phase]; dup {
			return nil, fmt.Errorf("line %d: duplicate phase %q", lineNo+1, phase)
		}
		out[phase] = hexHash
	}
	return out, nil
}
