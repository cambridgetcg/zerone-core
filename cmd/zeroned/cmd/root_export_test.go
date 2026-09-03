package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"cosmossdk.io/log"
	"github.com/cosmos/cosmos-db"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"

	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/server"
)

func TestNewExportAppBindsChainIDFromSourceGenesis(t *testing.T) {
	home := t.TempDir()
	configDir := filepath.Join(home, "config")
	require.NoError(t, os.MkdirAll(configDir, 0o700))
	require.NoError(t, os.WriteFile(
		filepath.Join(configDir, "genesis.json"),
		[]byte(`{"chain_id":"zerone-1"}`),
		0o600,
	))
	appOpts := viper.New()
	appOpts.Set(flags.FlagHome, home)
	appOpts.Set(server.FlagPruning, "default")
	application, err := newExportApp(
		log.NewNopLogger(),
		db.NewMemDB(),
		nil,
		false,
		appOpts,
	)
	require.NoError(t, err)
	require.Equal(t, "zerone-1", application.ChainID())
}

func TestNewExportAppRejectsChainIDOverrideMismatch(t *testing.T) {
	home := t.TempDir()
	configDir := filepath.Join(home, "config")
	require.NoError(t, os.MkdirAll(configDir, 0o700))
	require.NoError(t, os.WriteFile(
		filepath.Join(configDir, "genesis.json"),
		[]byte(`{"chain_id":"zerone-2"}`),
		0o600,
	))
	appOpts := viper.New()
	appOpts.Set(flags.FlagHome, home)
	appOpts.Set(flags.FlagChainID, "zerone-1")
	appOpts.Set(server.FlagPruning, "default")
	application, err := newExportApp(
		log.NewNopLogger(),
		db.NewMemDB(),
		nil,
		false,
		appOpts,
	)
	require.Nil(t, application)
	require.ErrorContains(t, err, `configured chain ID "zerone-1" does not match source genesis chain ID "zerone-2"`)
}

func TestExportSourceChainIDRejectsGenesisSymlink(t *testing.T) {
	home := t.TempDir()
	configDir := filepath.Join(home, "config")
	require.NoError(t, os.MkdirAll(configDir, 0o700))
	target := filepath.Join(home, "real-genesis.json")
	require.NoError(t, os.WriteFile(target, []byte(`{"chain_id":"zerone-1"}`), 0o600))
	require.NoError(t, os.Symlink(target, filepath.Join(configDir, "genesis.json")))
	appOpts := viper.New()
	appOpts.Set(flags.FlagHome, home)
	_, err := exportSourceChainID(appOpts)
	require.ErrorContains(t, err, "regular non-symlink file")
}

func TestExportSourceChainIDRejectsGenesisTraversal(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "home")
	require.NoError(t, os.MkdirAll(filepath.Join(home, "config"), 0o700))
	external := filepath.Join(root, "external")
	require.NoError(t, os.MkdirAll(external, 0o700))
	require.NoError(t, os.WriteFile(
		filepath.Join(external, "genesis.json"),
		[]byte(`{"chain_id":"zerone-1"}`),
		0o600,
	))
	appOpts := viper.New()
	appOpts.Set(flags.FlagHome, home)
	appOpts.Set("genesis_file", "config/../../external/genesis.json")
	_, err := exportSourceChainID(appOpts)
	require.ErrorContains(t, err, "unsafe path component")
}

func TestExportSourceChainIDRejectsSymlinkedGenesisAncestor(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "home")
	require.NoError(t, os.MkdirAll(home, 0o700))
	externalConfig := filepath.Join(root, "external-config")
	require.NoError(t, os.MkdirAll(externalConfig, 0o700))
	require.NoError(t, os.WriteFile(
		filepath.Join(externalConfig, "genesis.json"),
		[]byte(`{"chain_id":"zerone-1"}`),
		0o600,
	))
	require.NoError(t, os.Symlink(externalConfig, filepath.Join(home, "config")))
	appOpts := viper.New()
	appOpts.Set(flags.FlagHome, home)
	_, err := exportSourceChainID(appOpts)
	require.ErrorContains(t, err, "real non-symlink directory")
}

func TestExportSourceChainIDAcceptsCustomRelativeGenesisFile(t *testing.T) {
	home := t.TempDir()
	customDir := filepath.Join(home, "config", "archive")
	require.NoError(t, os.MkdirAll(customDir, 0o700))
	require.NoError(t, os.WriteFile(
		filepath.Join(customDir, "source-genesis.json"),
		[]byte(`{"chain_id":"zerone-1"}`),
		0o600,
	))
	appOpts := viper.New()
	appOpts.Set(flags.FlagHome, home)
	appOpts.Set("genesis_file", filepath.Join("config", "archive", "source-genesis.json"))
	chainID, err := exportSourceChainID(appOpts)
	require.NoError(t, err)
	require.Equal(t, "zerone-1", chainID)
}

func TestExportSourceChainIDRequiresHome(t *testing.T) {
	_, err := exportSourceChainID(viper.New())
	require.ErrorContains(t, err, "non-empty home directory")
}
