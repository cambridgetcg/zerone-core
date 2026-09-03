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

// initAppConfig returns the default server configuration template and values
// with zerone-specific overrides.
func initAppConfig() (string, interface{}) {
	srvCfg := serverconfig.DefaultConfig()

	// Minimum gas price prevents spam; 0.025 uzrn ≈ negligible for real users.
	srvCfg.MinGasPrices = "0.025uzrn"

	// Enable REST API and Swagger UI by default for testnet convenience.
	srvCfg.API.Enable = true
	srvCfg.API.Swagger = true

	// Keep Zerone's safe priority/nonce application pool enabled and bounded by
	// default. This must
	// match the checked-in app.toml template and the zerone-2 runtime profile so
	// a node started outside the production entrypoint does not silently fall
	// back to the SDK's NoOpMempool default.
	srvCfg.Mempool.MaxTxs = 5000

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

	// Pin the network-facing half of the production pooling profile here as
	// well as in the runtime entrypoint. These values are local admission and
	// gossip policy, not consensus parameters, but leaving the SDK defaults in
	// generated config would expose non-containerized validators to materially
	// different memory and transaction-size bounds.
	cfg.Mempool.Type = cmtcfg.MempoolTypeFlood
	cfg.Mempool.Recheck = true
	cfg.Mempool.RecheckTimeout = 5 * time.Second
	cfg.Mempool.Broadcast = true
	cfg.Mempool.WalPath = ""
	cfg.Mempool.Size = 5000
	cfg.Mempool.MaxTxsBytes = 64 * 1024 * 1024
	cfg.Mempool.CacheSize = 10000
	cfg.Mempool.KeepInvalidTxsInCache = false
	cfg.Mempool.MaxTxBytes = 256 * 1024
	cfg.Mempool.MaxBatchBytes = 0
	cfg.Mempool.ExperimentalMaxGossipConnectionsToPersistentPeers = 10
	cfg.Mempool.ExperimentalMaxGossipConnectionsToNonPersistentPeers = 10

	return cfg
}
