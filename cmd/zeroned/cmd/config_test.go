package cmd

import (
	"bytes"
	"path/filepath"
	"testing"
	"text/template"
	"time"

	cmtcfg "github.com/cometbft/cometbft/config"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"

	"github.com/cosmos/cosmos-sdk/server"

	zeroneapp "github.com/zerone-chain/zerone/app"
)

type mempoolTestAppOptions map[string]interface{}

func (options mempoolTestAppOptions) Get(key string) interface{} {
	return options[key]
}

func TestGeneratedConfigPinsProductionMempoolProfile(t *testing.T) {
	t.Run("application priority-nonce pool", func(t *testing.T) {
		appTemplate, appConfig := initAppConfig()
		var rendered bytes.Buffer
		tmpl, err := template.New("app-config").Parse(appTemplate)
		require.NoError(t, err)
		require.NoError(t, tmpl.Execute(&rendered, appConfig))

		parsed := parseTOML(t, rendered.Bytes())
		require.Equal(t, 5000, parsed.GetInt("mempool.max-txs"))
	})

	t.Run("Comet network pool", func(t *testing.T) {
		config := initCometBFTConfig()
		require.NoError(t, config.ValidateBasic())

		path := filepath.Join(t.TempDir(), "config.toml")
		cmtcfg.WriteConfigFile(path, config)
		parsed := viper.New()
		parsed.SetConfigFile(path)
		require.NoError(t, parsed.ReadInConfig())

		require.Equal(t, cmtcfg.MempoolTypeFlood, parsed.GetString("mempool.type"))
		require.True(t, parsed.GetBool("mempool.recheck"))
		require.Equal(t, 5*time.Second, parsed.GetDuration("mempool.recheck_timeout"))
		require.True(t, parsed.GetBool("mempool.broadcast"))
		require.Equal(t, "", parsed.GetString("mempool.wal_dir"))
		require.Equal(t, 5000, parsed.GetInt("mempool.size"))
		require.Equal(t, int64(64*1024*1024), parsed.GetInt64("mempool.max_txs_bytes"))
		require.Equal(t, 10000, parsed.GetInt("mempool.cache_size"))
		require.False(t, parsed.GetBool("mempool.keep-invalid-txs-in-cache"))
		require.Equal(t, 256*1024, parsed.GetInt("mempool.max_tx_bytes"))
		require.Zero(t, parsed.GetInt("mempool.max_batch_bytes"))
		require.Equal(t, 10, parsed.GetInt("mempool.experimental_max_gossip_connections_to_persistent_peers"))
		require.Equal(t, 10, parsed.GetInt("mempool.experimental_max_gossip_connections_to_non_persistent_peers"))
	})
}

func TestStartupRejectsUnsafeApplicationMempoolCapacity(t *testing.T) {
	for _, maxTx := range []int{
		zeroneapp.ApplicationMempoolMinTxs,
		zeroneapp.ApplicationMempoolMaxTxs,
	} {
		options := mempoolTestAppOptions{server.FlagMempoolMaxTxs: maxTx}
		require.Equal(t, maxTx, validatedApplicationMempoolMaxTx(options))
	}

	for _, maxTx := range []int{
		-1,
		0,
		zeroneapp.ApplicationMempoolMaxTxs + 1,
	} {
		options := mempoolTestAppOptions{server.FlagMempoolMaxTxs: maxTx}
		require.Panics(t, func() {
			validatedApplicationMempoolMaxTx(options)
		})
	}
	require.Panics(t, func() {
		validatedApplicationMempoolMaxTx(mempoolTestAppOptions{})
	}, "a missing config key casts to SDK MaxTx=0 and must fail closed")
}

func parseTOML(t *testing.T, contents []byte) *viper.Viper {
	t.Helper()
	parsed := viper.New()
	parsed.SetConfigType("toml")
	require.NoError(t, parsed.ReadConfig(bytes.NewReader(contents)))
	return parsed
}
