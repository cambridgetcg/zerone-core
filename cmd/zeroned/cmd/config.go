package cmd

import (
	"time"

	serverconfig "github.com/cosmos/cosmos-sdk/server/config"

	cmtcfg "github.com/cometbft/cometbft/config"
)

// initAppConfig returns the default server configuration template and values
// with zerone-specific overrides.
func initAppConfig() (string, interface{}) {
	srvCfg := serverconfig.DefaultConfig()

	// Minimum gas price prevents spam; 0.025 uzrn ≈ negligible for real users.
	srvCfg.MinGasPrices = "0.025uzrn"

	// Enable REST API and Swagger UI by default for testnet convenience.
	srvCfg.API.Enable = true
	srvCfg.API.Swagger = true

	return serverconfig.DefaultConfigTemplate, srvCfg
}

// initCometBFTConfig returns the default CometBFT configuration with
// zerone-specific overrides.
func initCometBFTConfig() *cmtcfg.Config {
	cfg := cmtcfg.DefaultConfig()

	// Zerone targets 2521ms block time.
	cfg.Consensus.TimeoutCommit = 2521 * time.Millisecond

	return cfg
}
