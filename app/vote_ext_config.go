package app

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"cosmossdk.io/log"
	"github.com/cosmos/cosmos-sdk/client/flags"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	"github.com/spf13/cast"
)

// voteExtValidatorAddressOpt is the app.toml key that opts a node into acting
// as a Proof-of-Truth verifier. It carries the node's own bech32 valoper
// (operator) address, which stake-weighted VRF selection is keyed on.
const voteExtValidatorAddressOpt = "vote-extensions.validator-address"

// ConfigureVoteExtensions wires this node's PoT vote-extension config from
// node-local key material and app.toml settings, if — and only if — the
// operator has opted in by setting [vote-extensions] validator-address.
//
// Opt-in and fail-safe by design:
//   - If validator-address is unset, the node is a non-verifier and
//     VoteExtConfig stays nil (ExtendVote returns emptyVoteExtension).
//   - If the consensus key file is missing or unreadable, we log and leave
//     the node a non-verifier rather than aborting startup.
//
// This only populates configuration. CometBFT does not invoke ExtendVote
// until the consensus param vote_extensions_enable_height is reached, so a
// configured node produces no vote extensions until that height — landing
// this wiring changes nothing on a chain that has not enabled them.
func (app *ZeroneApp) ConfigureVoteExtensions(appOpts servertypes.AppOptions, logger log.Logger) {
	logger = logger.With("module", "vote-extensions")

	valoper := cast.ToString(appOpts.Get(voteExtValidatorAddressOpt))
	if valoper == "" {
		// Node is not configured as a PoT verifier — nothing to do.
		return
	}

	home := cast.ToString(appOpts.Get(flags.FlagHome))
	if home == "" {
		logger.Error("validator-address is set but the node home dir is unknown; not acting as a PoT verifier")
		return
	}

	keyFile := filepath.Join(home, "config", "priv_validator_key.json")
	privKey, err := loadConsensusPrivKey(keyFile)
	if err != nil {
		logger.Error("could not load consensus key; not acting as a PoT verifier",
			"validator", valoper, "key_file", keyFile, "err", err)
		return
	}

	dataDir := filepath.Join(home, "data")
	app.SetVoteExtConfig(&VoteExtensionConfig{
		ValidatorAddress:    valoper,
		ValidatorPrivateKey: privKey,
		LocalStore:          NewLocalCommitmentStore(dataDir),
	})

	logger.Info("configured as PoT verifier",
		"validator", valoper,
		"commitment_store", filepath.Join(dataDir, commitmentFileName),
		"oracle_attached", app.oracleClient != nil,
	)
}

// loadConsensusPrivKey reads the Ed25519 consensus private key from a CometBFT
// priv_validator_key.json without going through CometBFT's file-privval loader,
// which calls os.Exit on a malformed file — unacceptable during app assembly.
// Returns the raw key bytes (64-byte Ed25519 private key, or a 32-byte seed);
// x/knowledge/crypto.GenerateVRF accepts either form.
func loadConsensusPrivKey(keyFile string) ([]byte, error) {
	bz, err := os.ReadFile(keyFile)
	if err != nil {
		return nil, err
	}

	var f struct {
		PrivKey struct {
			Value string `json:"value"`
		} `json:"priv_key"`
	}
	if err := json.Unmarshal(bz, &f); err != nil {
		return nil, fmt.Errorf("parse %s: %w", filepath.Base(keyFile), err)
	}
	if f.PrivKey.Value == "" {
		return nil, fmt.Errorf("no priv_key in %s", filepath.Base(keyFile))
	}

	key, err := base64.StdEncoding.DecodeString(f.PrivKey.Value)
	if err != nil {
		return nil, fmt.Errorf("decode priv_key: %w", err)
	}
	if len(key) != 64 && len(key) != 32 {
		return nil, fmt.Errorf("unexpected priv_key length %d (want 32 or 64)", len(key))
	}
	return key, nil
}
