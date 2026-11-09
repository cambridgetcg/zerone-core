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

// ZeroneAppConfig extends the SDK server config with oracle settings.
type ZeroneAppConfig struct {
	serverconfig.Config `mapstructure:",squash"`
	Oracle              OracleConfig `mapstructure:"oracle"`
}

const oracleConfigTemplate = `

###############################################################################
###                      Oracle Sidecar Configuration                       ###
###############################################################################

[oracle]

# Enable oracle queries for the Proof-of-Truth vote-extension commit phase.
# This binary compiles that transport off (potVoteExtensionsReleaseEnabled =
# false in app/vote_extensions.go), so zeroned never queries the oracle.
# Enabling it only builds an unused client and logs "oracle sidecar enabled"
# at startup.
# Where the transport is on, a validator with no usable oracle verdict casts
# no vote for that claim. It does not fall back to accept.
enabled = {{ .Oracle.Enabled }}

# HTTP endpoint of the zerone-oracle sidecar.
endpoint = "{{ .Oracle.Endpoint }}"

# Maximum time to wait for an oracle response.
# Must be less than the vote extension timeout (typically 2-3s).
timeout = "{{ .Oracle.Timeout }}"

# Minimum confidence threshold to use the oracle's verdict.
# A verdict below this threshold counts as uncertain, and the validator casts
# no vote for that claim.
min-confidence = {{ printf "%.2f" .Oracle.MinConfidence }}
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
	}

	customTemplate := serverconfig.DefaultConfigTemplate + oracleConfigTemplate

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
