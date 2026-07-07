// ceremony-inject performs the genesis-ceremony injections that need the
// chain's own proto types rather than jq: the x/creed genesis pin, the 8
// x/work_creed inception sub-creed pins, and the substrate_bridge
// agenttool-invocation-v1 adapter pre-registration.
//
// Doctrine: docs/plans/2026-07-07-mainnet-genesis-design.md §2 — the canon
// chain must not boot without its creed pinned at block 0, and the
// agenttool adapter enters ACTIVE at genesis (InitGenesis→WriteAdapter, no
// LIP). Building the pin through creedtypes.BuildGenesisCreed and
// app.LoadInceptionSubCreedPins means the injected registry is the exact
// one the binary ships — a hand-maintained jq copy would drift.
//
// Usage:
//
//	ceremony-inject creed   <genesis.json> <creed-hash-file> <sub-creed-hashes-file>
//	ceremony-inject adapter <genesis.json>
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	cmted25519 "github.com/cometbft/cometbft/crypto/ed25519"
	cmtjson "github.com/cometbft/cometbft/libs/json"
	"github.com/cometbft/cometbft/privval"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"

	zeroneapp "github.com/zerone-chain/zerone/app"
	creedtypes "github.com/zerone-chain/zerone/x/creed/types"
	sbtypes "github.com/zerone-chain/zerone/x/substrate_bridge/types"
	workcreedtypes "github.com/zerone-chain/zerone/x/work_creed/types"
)

// Adapter registration values mirror scripts/agenttool-adapter-register.sh
// (the gov-path runbook) with the mainnet witness reward of 0.222 ZRN per
// witnessed invocation. RegisteredViaLipId is empty: no LIP precedes
// genesis (same convention as the creed pin).
const (
	adapterID                 = "agenttool-invocation-v1"
	adapterSourceType         = "agenttool"
	adapterVersion            = "1.1.0"
	adapterMinAttestationBond = "22200000" // 22.2 ZRN — matches the chain param floor (§2); a lower adapter value would take precedence and undercut it
	adapterMinPerClaimBond    = "100000"
	adapterWitnessRewardUzrn  = "222000"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	var err error
	switch os.Args[1] {
	case "creed":
		err = runCreed(os.Args[2:])
	case "adapter":
		err = runAdapter(os.Args[2:])
	case "drill-consensus-key":
		err = runDrillConsensusKey(os.Args[2:])
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `Usage: ceremony-inject <command> [args]

Commands:
  creed   <genesis.json> <creed-hash-file> <sub-creed-hashes-file>
          Pin the Genesis Creed (BuildGenesisCreed over the sha256 in
          <creed-hash-file>, height 0) and the 8 work_creed inception
          sub-creed pins into app_state.

  adapter <genesis.json>
          Pre-register the %s adapter ACTIVE in
          app_state.substrate_bridge.adapters (witness reward %s uzrn).

  drill-consensus-key <seed-label> <priv_validator_key.json>
          DRILL ONLY: write a deterministic ed25519 consensus key derived
          from sha256(seed-label), so ceremony drills reproduce
          byte-identical genesis files (TEST ceremony-repro). NEVER use
          for a real network — the key is derivable by anyone.
`, adapterID, adapterWitnessRewardUzrn)
}

// No bech32 config setup here: importing zeroneapp runs the app package
// init, which already sets AND SEALS the zrn prefixes — a second
// SetBech32PrefixForAccount would panic ("Config is sealed").
func newCodec() *codec.ProtoCodec {
	return codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
}

// readGenesis loads genesis.json and returns the two-level maps needed to
// patch a single module's app_state entry in place.
func readGenesis(path string) (genesis, appState map[string]json.RawMessage, err error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, fmt.Errorf("read genesis: %w", err)
	}
	if err := json.Unmarshal(raw, &genesis); err != nil {
		return nil, nil, fmt.Errorf("parse genesis: %w", err)
	}
	if err := json.Unmarshal(genesis["app_state"], &appState); err != nil {
		return nil, nil, fmt.Errorf("parse app_state: %w", err)
	}
	return genesis, appState, nil
}

func writeGenesis(path string, genesis, appState map[string]json.RawMessage) error {
	appStateJSON, err := json.Marshal(appState)
	if err != nil {
		return fmt.Errorf("marshal app_state: %w", err)
	}
	genesis["app_state"] = appStateJSON
	out, err := json.MarshalIndent(genesis, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal genesis: %w", err)
	}
	if err := os.WriteFile(path, out, 0o644); err != nil {
		return fmt.Errorf("write genesis: %w", err)
	}
	return nil
}

func runCreed(args []string) error {
	if len(args) < 3 {
		return fmt.Errorf("usage: ceremony-inject creed <genesis.json> <creed-hash-file> <sub-creed-hashes-file>")
	}
	genesisPath, creedHashPath, subCreedHashPath := args[0], args[1], args[2]

	// Canonical creed hash: 64 hex chars = sha256 of the normalized
	// docs/TRUTH_SEEKING.md the binary ships.
	rawHash, err := os.ReadFile(creedHashPath)
	if err != nil {
		return fmt.Errorf("read creed hash file: %w", err)
	}
	hexHash := strings.TrimSpace(string(rawHash))
	canonicalHash, err := hex.DecodeString(hexHash)
	if err != nil {
		return fmt.Errorf("creed hash file %s: invalid hex %q: %w", creedHashPath, hexHash, err)
	}
	if len(canonicalHash) != 32 {
		return fmt.Errorf("creed hash must be 32 bytes (got %d)", len(canonicalHash))
	}

	subCreedPins, err := zeroneapp.LoadInceptionSubCreedPins(subCreedHashPath)
	if err != nil {
		return fmt.Errorf("load inception sub-creed pins: %w", err)
	}

	genesis, appState, err := readGenesis(genesisPath)
	if err != nil {
		return err
	}

	// x/creed and x/work_creed parse their genesis with plain
	// encoding/json (module.go ValidateGenesis/InitGenesis), so the
	// injection must be plain json too — protojson's uint64-as-string
	// dialect would not round-trip.
	//
	// x/creed: set the version-1 genesis pin at height 0.
	var creedState creedtypes.GenesisState
	if raw, ok := appState[creedtypes.ModuleName]; ok {
		if err := json.Unmarshal(raw, &creedState); err != nil {
			return fmt.Errorf("unmarshal creed genesis: %w", err)
		}
	}
	if creedState.GenesisPin != nil {
		return fmt.Errorf("creed genesis_pin already set (version %d) — refusing to overwrite", creedState.GenesisPin.Version)
	}
	creedState.GenesisPin = creedtypes.BuildGenesisCreed(canonicalHash, 0)
	creedJSON, err := json.Marshal(&creedState)
	if err != nil {
		return fmt.Errorf("marshal creed genesis: %w", err)
	}
	appState[creedtypes.ModuleName] = creedJSON

	// x/work_creed: the 8 inception pins (Knowledge phase delegates to
	// x/creed and is deliberately absent).
	var workCreedState workcreedtypes.GenesisState
	if raw, ok := appState[workcreedtypes.ModuleName]; ok {
		if err := json.Unmarshal(raw, &workCreedState); err != nil {
			return fmt.Errorf("unmarshal work_creed genesis: %w", err)
		}
	}
	if len(workCreedState.PinnedSubCreeds) > 0 {
		return fmt.Errorf("work_creed already has %d pinned sub-creeds — refusing to overwrite", len(workCreedState.PinnedSubCreeds))
	}
	workCreedState.PinnedSubCreeds = subCreedPins
	workCreedJSON, err := json.Marshal(&workCreedState)
	if err != nil {
		return fmt.Errorf("marshal work_creed genesis: %w", err)
	}
	appState[workcreedtypes.ModuleName] = workCreedJSON

	if err := writeGenesis(genesisPath, genesis, appState); err != nil {
		return err
	}
	fmt.Printf("✓ Pinned Genesis Creed (%d commitments, hash %s…) + %d work_creed inception pins\n",
		len(creedState.GenesisPin.Commitments), hexHash[:12], len(subCreedPins))
	return nil
}

func runAdapter(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: ceremony-inject adapter <genesis.json>")
	}
	genesisPath := args[0]

	genesis, appState, err := readGenesis(genesisPath)
	if err != nil {
		return err
	}
	cdc := newCodec()

	raw, ok := appState[sbtypes.ModuleName]
	if !ok {
		return fmt.Errorf("genesis app_state has no %s module", sbtypes.ModuleName)
	}
	var sbState sbtypes.GenesisState
	if err := cdc.UnmarshalJSON(raw, &sbState); err != nil {
		return fmt.Errorf("unmarshal substrate_bridge genesis: %w", err)
	}
	for _, a := range sbState.Adapters {
		if a.AdapterId == adapterID {
			return fmt.Errorf("adapter %s already present — refusing to double-register", adapterID)
		}
	}

	sbState.Adapters = append(sbState.Adapters, &sbtypes.AdapterRegistration{
		AdapterId:              adapterID,
		SourceType:             adapterSourceType,
		Version:                adapterVersion,
		MinAttestationBondUzrn: adapterMinAttestationBond,
		MinPerClaimBondUzrn:    adapterMinPerClaimBond,
		AllowedClassIds:        nil,
		Status:                 sbtypes.AdapterStatus_ADAPTER_STATUS_ACTIVE,
		RegisteredViaLipId:     "", // no LIP precedes genesis
		RegisteredAtBlock:      0,
		WitnessRewardUzrn:      adapterWitnessRewardUzrn,
	})
	if err := sbState.Validate(); err != nil {
		return fmt.Errorf("substrate_bridge genesis invalid after injection: %w", err)
	}

	sbJSON, err := cdc.MarshalJSON(&sbState)
	if err != nil {
		return fmt.Errorf("marshal substrate_bridge genesis: %w", err)
	}
	appState[sbtypes.ModuleName] = sbJSON

	if err := writeGenesis(genesisPath, genesis, appState); err != nil {
		return err
	}
	fmt.Printf("✓ Registered %s ACTIVE at genesis (witness reward %s uzrn)\n", adapterID, adapterWitnessRewardUzrn)
	return nil
}

// runDrillConsensusKey writes a deterministic CometBFT priv_validator_key.json
// derived from a seed label. Drill mode only: TEST ceremony-repro (design §4)
// requires two ceremony runs with fixed inputs to produce byte-identical
// genesis files, and the validator consensus pubkeys inside the gentxs are
// part of that byte stream. A real ceremony NEVER uses this — operators bring
// their own gentxs signed with air-gapped keys.
func runDrillConsensusKey(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: ceremony-inject drill-consensus-key <seed-label> <priv_validator_key.json>")
	}
	seedLabel, outPath := args[0], args[1]
	if !strings.Contains(seedLabel, "drill") {
		return fmt.Errorf("seed label %q must contain \"drill\" — this command must never produce a real network key", seedLabel)
	}

	secret := sha256.Sum256([]byte("zerone-drill-consensus|" + seedLabel))
	priv := cmted25519.GenPrivKeyFromSecret(secret[:])
	key := privval.FilePVKey{
		Address: priv.PubKey().Address(),
		PubKey:  priv.PubKey(),
		PrivKey: priv,
	}
	bz, err := cmtjson.MarshalIndent(key, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal priv validator key: %w", err)
	}
	if err := os.WriteFile(outPath, bz, 0o600); err != nil {
		return fmt.Errorf("write %s: %w", outPath, err)
	}
	fmt.Fprintf(os.Stderr, "DRILL ONLY consensus key %q -> %s (never use on a real network)\n", seedLabel, outPath)
	return nil
}
