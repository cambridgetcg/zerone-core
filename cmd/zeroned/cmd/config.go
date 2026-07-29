package cmd

import (
	"time"

	serverconfig "github.com/cosmos/cosmos-sdk/server/config"

	cmtcfg "github.com/cometbft/cometbft/config"
)

// OracleConfig holds configuration for the validator evaluation oracle sidecar.
type OracleConfig struct {
	// Enabled enables oracle queries during vote extension commit phase.
	Enabled bool `mapstructure:"enabled"`
	// Endpoint is the HTTP URL of the oracle sidecar.
	Endpoint string `mapstructure:"endpoint"`
	// Timeout is the maximum time to wait for an oracle response.
	Timeout time.Duration `mapstructure:"timeout"`
	// MinConfidence is the minimum confidence threshold to use the oracle's verdict.
	MinConfidence float64 `mapstructure:"min-confidence"`
}

// VoteExtensionsConfig holds this node's Proof-of-Truth verifier identity.
type VoteExtensionsConfig struct {
	// ValidatorAddress is this node's own bech32 valoper (operator) address.
	// Set it to opt the node into acting as a PoT verifier: it will produce
	// vote extensions (VRF selection + commit/reveal) once the chain enables
	// vote_extensions_enable_height. Leave empty on non-validator nodes.
	ValidatorAddress string `mapstructure:"validator-address"`
}

// ZeroneAppConfig extends the SDK server config with zerone-specific settings.
type ZeroneAppConfig struct {
	serverconfig.Config `mapstructure:",squash"`
	Oracle              OracleConfig         `mapstructure:"oracle"`
	VoteExtensions      VoteExtensionsConfig `mapstructure:"vote-extensions"`
}

const oracleConfigTemplate = `

###############################################################################
###                      Oracle Sidecar Configuration                       ###
###############################################################################

[oracle]

# Enable oracle queries during vote extension commit phase.
# When enabled, the validator queries the oracle sidecar before voting.
# Oracle failure never blocks consensus — falls back to default accept.
enabled = {{ .Oracle.Enabled }}

# HTTP endpoint of the zerone-oracle sidecar.
endpoint = "{{ .Oracle.Endpoint }}"

# Maximum time to wait for an oracle response.
# Must be less than the vote extension timeout (typically 2-3s).
timeout = "{{ .Oracle.Timeout }}"

# Minimum confidence threshold to use the oracle's verdict.
# Verdicts below this threshold are treated as uncertain (default: accept).
# Protects validators from slashing on weak oracle signals.
min-confidence = {{ printf "%.2f" .Oracle.MinConfidence }}
`

const voteExtensionsConfigTemplate = `

###############################################################################
###                  Proof-of-Truth Vote Extensions                         ###
###############################################################################

[vote-extensions]

# This node's own bech32 valoper (operator) address. Set it to opt the node
# into acting as a Proof-of-Truth verifier: it will produce vote extensions
# (VRF selection + commit/reveal) once the chain enables vote extensions.
# Leave empty on non-validator nodes.
validator-address = "{{ .VoteExtensions.ValidatorAddress }}"
`

// initAppConfig returns the default server configuration template and values
// with zerone-specific overrides.
func initAppConfig() (string, interface{}) {
	srvCfg := serverconfig.DefaultConfig()

	// Minimum gas price prevents spam; 0.025 uzrn ≈ negligible for real users.
	srvCfg.MinGasPrices = "0.025uzrn"

	// Enable REST API and Swagger UI by default for testnet convenience.
	srvCfg.API.Enable = true
	srvCfg.API.Swagger = true

	customConfig := ZeroneAppConfig{
		Config: *srvCfg,
		Oracle: OracleConfig{
			Enabled:       false,
			Endpoint:      "http://localhost:8081",
			Timeout:       2 * time.Second,
			MinConfidence: 0.6,
		},
		VoteExtensions: VoteExtensionsConfig{
			ValidatorAddress: "",
		},
	}

	customTemplate := serverconfig.DefaultConfigTemplate + oracleConfigTemplate + voteExtensionsConfigTemplate

	return customTemplate, customConfig
}

// initCometBFTConfig returns the default CometBFT configuration with
// zerone-specific overrides.
func initCometBFTConfig() *cmtcfg.Config {
	cfg := cmtcfg.DefaultConfig()

	// Zerone targets 2521ms block time.
	cfg.Consensus.TimeoutCommit = 2521 * time.Millisecond

	return cfg
}
