package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"cosmossdk.io/client/v2/autocli"
	"cosmossdk.io/core/appmodule"
	"cosmossdk.io/log"
	sdkevidencecli "cosmossdk.io/x/evidence/client/cli"
	feegrantcli "cosmossdk.io/x/feegrant/client/cli"
	upgradecli "cosmossdk.io/x/upgrade/client/cli"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/spf13/cast"
	"github.com/spf13/cobra"
	"golang.org/x/sys/unix"

	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/config"
	"github.com/cosmos/cosmos-sdk/client/debug"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/keys"
	"github.com/cosmos/cosmos-sdk/client/pruning"
	"github.com/cosmos/cosmos-sdk/client/rpc"
	"github.com/cosmos/cosmos-sdk/client/snapshot"
	addresscodec "github.com/cosmos/cosmos-sdk/codec/address"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/server"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	sdkauthcli "github.com/cosmos/cosmos-sdk/x/auth/client/cli"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	sdkbankcli "github.com/cosmos/cosmos-sdk/x/bank/client/cli"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	sdkdistrcli "github.com/cosmos/cosmos-sdk/x/distribution/client/cli"
	"github.com/cosmos/cosmos-sdk/x/genutil"
	genutilcli "github.com/cosmos/cosmos-sdk/x/genutil/client/cli"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
	sdkgovcli "github.com/cosmos/cosmos-sdk/x/gov/client/cli"
	sdkstakingcli "github.com/cosmos/cosmos-sdk/x/staking/client/cli"

	"github.com/zerone-chain/zerone/app"
	alignmentcli "github.com/zerone-chain/zerone/x/alignment/client/cli"
	zeroneauthcli "github.com/zerone-chain/zerone/x/auth/client/cli"
	capturechallengecli "github.com/zerone-chain/zerone/x/capture_challenge/client/cli"
	capturedefensecli "github.com/zerone-chain/zerone/x/capture_defense/client/cli"
	claimingpotcli "github.com/zerone-chain/zerone/x/claiming_pot/client/cli"
	emergencycli "github.com/zerone-chain/zerone/x/emergency/client/cli"
	govcli "github.com/zerone-chain/zerone/x/gov/client/cli"
	homecli "github.com/zerone-chain/zerone/x/home/client/cli"
	ibcratelimitcli "github.com/zerone-chain/zerone/x/ibcratelimit/client/cli"
	knowledgecli "github.com/zerone-chain/zerone/x/knowledge/client/cli"
	liquiditypoolcli "github.com/zerone-chain/zerone/x/liquiditypool/client/cli"
	ontologycli "github.com/zerone-chain/zerone/x/ontology/client/cli"
	qualificationcli "github.com/zerone-chain/zerone/x/qualification/client/cli"
	stakingcli "github.com/zerone-chain/zerone/x/staking/client/cli"
	tokenscli "github.com/zerone-chain/zerone/x/tokens/client/cli"
	vestingrewardscli "github.com/zerone-chain/zerone/x/vesting_rewards/client/cli"
)

// NewRootCmd creates the root command for the zeroned daemon.
func NewRootCmd() *cobra.Command {
	encodingConfig := app.MakeEncodingConfig()

	initClientCtx := client.Context{}.
		WithCodec(encodingConfig.Codec).
		WithInterfaceRegistry(encodingConfig.InterfaceRegistry).
		WithTxConfig(encodingConfig.TxConfig).
		WithLegacyAmino(encodingConfig.Amino).
		WithInput(os.Stdin).
		WithAccountRetriever(authtypes.AccountRetriever{}).
		WithHomeDir(app.DefaultNodeHome).
		WithViper("")

	rootCmd := &cobra.Command{
		Use:   app.AppName,
		Short: "Zerone blockchain node — Proof of Truth consensus",
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			cmd.SetOut(cmd.OutOrStdout())
			cmd.SetErr(cmd.ErrOrStderr())

			initClientCtx, err := client.ReadPersistentCommandFlags(initClientCtx, cmd.Flags())
			if err != nil {
				return err
			}

			initClientCtx, err = config.ReadFromClientConfig(initClientCtx)
			if err != nil {
				return err
			}

			if err := client.SetCmdClientContextHandler(initClientCtx, cmd); err != nil {
				return err
			}

			customAppTemplate, customAppConfig := initAppConfig()
			customCMTConfig := initCometBFTConfig()

			return server.InterceptConfigsPreRunHandler(cmd, customAppTemplate, customAppConfig, customCMTConfig)
		},
	}

	initRootCmd(rootCmd, encodingConfig)

	// AutoCLI: generate query/tx commands for SDK modules (bank, auth,
	// staking, distribution, gov, slashing, etc.) from protobuf service
	// definitions. Only adds commands that are not already registered
	// manually, so existing custom module commands are preserved.
	tempApp := app.NewZeroneApp(
		log.NewNopLogger(),
		dbm.NewMemDB(),
		nil,
		false,
		emptyAppOptions{},
	)

	moduleMap := make(map[string]appmodule.AppModule)
	for name, mod := range tempApp.ModuleManager.Modules {
		if m, ok := mod.(appmodule.AppModule); ok {
			moduleMap[name] = m
		}
	}

	autoCliOpts := autocli.AppOptions{
		Modules:               moduleMap,
		AddressCodec:          addresscodec.NewBech32Codec(app.AccountAddressPrefix),
		ValidatorAddressCodec: runtime.ValidatorAddressCodec(addresscodec.NewBech32Codec(app.AccountAddressPrefix + "valoper")),
		ConsensusAddressCodec: runtime.ConsensusAddressCodec(addresscodec.NewBech32Codec(app.AccountAddressPrefix + "valcons")),
		ClientCtx:             initClientCtx,
	}

	if err := autoCliOpts.EnhanceRootCommand(rootCmd); err != nil {
		panic(err)
	}

	return rootCmd
}

// initRootCmd registers all sub-commands on the root command.
func initRootCmd(rootCmd *cobra.Command, encodingConfig app.EncodingConfig) {
	rootCmd.AddCommand(
		genutilcli.InitCmd(app.ModuleBasics, app.DefaultNodeHome),
		debug.Cmd(),
		pruning.Cmd(newApp, app.DefaultNodeHome),
		snapshot.Cmd(newApp),
		activationPreflightCmd(),
		recoveryActionDigestCmd(),
	)

	server.AddCommands(rootCmd, app.DefaultNodeHome, newApp, appExport, addModuleInitFlags)

	// Key management
	rootCmd.AddCommand(keys.Commands())

	// Genesis subcommand
	rootCmd.AddCommand(genesisCommand(encodingConfig))

	// Top-level add-genesis-account (expected by boot-test.sh)
	rootCmd.AddCommand(AddGenesisAccountCmd(app.DefaultNodeHome))

	// Status / query / tx
	rootCmd.AddCommand(server.StatusCommand())
	rootCmd.AddCommand(verifyFrozenTerminalCmd())
	rootCmd.AddCommand(queryCommand(encodingConfig))
	rootCmd.AddCommand(txCommand(encodingConfig))
}

// addModuleInitFlags adds module-specific init flags to the start command.
func addModuleInitFlags(startCmd *cobra.Command) {
	// Module-specific flags can be added here if needed.
}

// genesisCommand returns the genesis subcommand tree.
func genesisCommand(encodingConfig app.EncodingConfig) *cobra.Command {
	gentxModule := app.ModuleBasics[genutiltypes.ModuleName].(genutil.AppModuleBasic)

	cmd := &cobra.Command{
		Use:                        "genesis",
		Short:                      "Application genesis-related subcommands",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(
		genutilcli.GenTxCmd(
			app.ModuleBasics,
			encodingConfig.TxConfig,
			banktypes.GenesisBalancesIterator{},
			app.DefaultNodeHome,
			encodingConfig.InterfaceRegistry.SigningContext().ValidatorAddressCodec(),
		),
		genutilcli.CollectGenTxsCmd(
			banktypes.GenesisBalancesIterator{},
			app.DefaultNodeHome,
			gentxModule.GenTxValidator,
			encodingConfig.InterfaceRegistry.SigningContext().ValidatorAddressCodec(),
		),
		genutilcli.ValidateGenesisCmd(app.ModuleBasics),
	)

	return cmd
}

// queryCommand returns the root query command.
func queryCommand(_ app.EncodingConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:                        "query",
		Aliases:                    []string{"q"},
		Short:                      "Querying subcommands",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(
		rpc.QueryEventForTxCmd(),
		server.QueryBlockCmd(),
		server.QueryBlocksCmd(),
		server.QueryBlockResultsCmd(),
	)

	// SDK auth utility query commands
	cmd.AddCommand(
		sdkauthcli.QueryTxsByEventsCmd(),
		sdkauthcli.QueryTxCmd(),
	)

	// Zerone custom module query commands
	cmd.AddCommand(
		alignmentcli.NewQueryCmd(),
		zeroneauthcli.GetQueryCmd(),
		capturechallengecli.NewQueryCmd(),
		capturedefensecli.NewQueryCmd(),
		claimingpotcli.NewQueryCmd(),
		emergencycli.NewQueryCmd(),
		govcli.NewQueryCmd(),
		homecli.NewQueryCmd(),
		ibcratelimitcli.NewQueryCmd(),
		knowledgecli.GetQueryCmd(),
		liquiditypoolcli.NewQueryCmd(),
		ontologycli.NewQueryCmd(),
		qualificationcli.NewQueryCmd(),
		stakingcli.GetQueryCmd(),
		tokenscli.NewQueryCmd(),
		vestingrewardscli.NewQueryCmd(),
	)

	return cmd
}

// txCommand returns the root tx command.
func txCommand(encodingConfig app.EncodingConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:                        "tx",
		Short:                      "Transactions subcommands",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(
		flags.LineBreak,
	)

	// SDK auth utility tx commands
	cmd.AddCommand(
		sdkauthcli.GetEncodeCommand(),
		sdkauthcli.GetDecodeCommand(),
		sdkauthcli.GetBroadcastCommand(),
		sdkauthcli.GetSignCommand(),
		sdkauthcli.GetSignBatchCommand(),
		sdkauthcli.GetMultiSignCommand(),
		sdkauthcli.GetMultiSignBatchCmd(),
		sdkauthcli.GetValidateSignaturesCommand(),
		sdkauthcli.GetSimulateCmd(),
	)

	// SDK module tx commands
	ac := encodingConfig.InterfaceRegistry.SigningContext().AddressCodec()
	valAc := encodingConfig.InterfaceRegistry.SigningContext().ValidatorAddressCodec()
	cmd.AddCommand(
		sdkbankcli.NewTxCmd(ac),
		sdkdistrcli.NewTxCmd(valAc, ac),
		sdkgovcli.NewTxCmd(nil),
		sdkstakingcli.NewTxCmd(valAc, ac),
		sdkevidencecli.GetTxCmd(nil),
		feegrantcli.GetTxCmd(ac),
		upgradecli.GetTxCmd(ac),
	)

	// Zerone custom module tx commands
	cmd.AddCommand(
		alignmentcli.NewTxCmd(),
		zeroneauthcli.GetTxCmd(),
		capturechallengecli.NewTxCmd(),
		capturedefensecli.NewTxCmd(),
		claimingpotcli.NewTxCmd(),
		emergencycli.NewTxCmd(),
		govcli.NewTxCmd(),
		homecli.NewTxCmd(),
		ibcratelimitcli.NewTxCmd(),
		knowledgecli.GetTxCmd(),
		liquiditypoolcli.NewTxCmd(),
		ontologycli.NewTxCmd(),
		qualificationcli.NewTxCmd(),
		stakingcli.GetTxCmd(),
		tokenscli.NewTxCmd(),
		vestingrewardscli.NewTxCmd(),
	)

	return cmd
}

// newApp creates a new ZeroneApp with the given options.
func newApp(
	logger log.Logger,
	db dbm.DB,
	traceStore io.Writer,
	appOpts servertypes.AppOptions,
) servertypes.Application {
	baseappOptions := server.DefaultBaseappOptions(appOpts)

	return app.NewZeroneApp(
		logger, db, traceStore, true, appOpts,
		baseappOptions...,
	)
}

// appExport creates a new ZeroneApp and exports its state.
func appExport(
	logger log.Logger,
	db dbm.DB,
	traceStore io.Writer,
	height int64,
	forZeroHeight bool,
	jailAllowedAddrs []string,
	appOpts servertypes.AppOptions,
	modulesToExport []string,
) (servertypes.ExportedApp, error) {
	homePath := cast.ToString(appOpts.Get(flags.FlagHome))
	if homePath == "" {
		return servertypes.ExportedApp{}, fmt.Errorf("state export requires a non-empty home directory")
	}

	zeroneApp, err := newExportApp(logger, db, traceStore, height == -1, appOpts)
	if err != nil {
		return servertypes.ExportedApp{}, err
	}

	if height != -1 {
		if err := zeroneApp.LoadHeight(height); err != nil {
			return servertypes.ExportedApp{}, err
		}
	}

	return zeroneApp.ExportAppStateAndValidators(forZeroHeight, jailAllowedAddrs, modulesToExport)
}

// newExportApp applies the same SDK BaseApp options as normal daemon startup,
// then binds the chain ID parsed from the source home's genesis file. An
// environment or command-line chain-id override must never make a different
// source home eligible for the narrow zerone-1 historical export boundary.
func newExportApp(
	logger log.Logger,
	db dbm.DB,
	traceStore io.Writer,
	loadLatest bool,
	appOpts servertypes.AppOptions,
) (*app.ZeroneApp, error) {
	chainID, err := exportSourceChainID(appOpts)
	if err != nil {
		return nil, err
	}
	baseappOptions := server.DefaultBaseappOptions(appOpts)
	// Last option wins. This makes the source-genesis value authoritative even
	// when a matching explicit override was supplied.
	baseappOptions = append(baseappOptions, baseapp.SetChainID(chainID))
	return app.NewZeroneApp(
		logger,
		db,
		traceStore,
		loadLatest,
		appOpts,
		baseappOptions...,
	), nil
}

func exportSourceChainID(appOpts servertypes.AppOptions) (string, error) {
	homeDir := cast.ToString(appOpts.Get(flags.FlagHome))
	if homeDir == "" {
		return "", fmt.Errorf("state export requires a non-empty home directory")
	}
	genesisRelativePath, err := exportGenesisRelativePath(
		cast.ToString(appOpts.Get("genesis_file")),
	)
	if err != nil {
		return "", err
	}
	genesisFile, genesisPath, err := openExportGenesisNoFollow(homeDir, genesisRelativePath)
	if err != nil {
		return "", err
	}
	chainID, parseErr := genutiltypes.ParseChainIDFromGenesis(genesisFile)
	closeErr := genesisFile.Close()
	if parseErr != nil {
		return "", fmt.Errorf("parse chain ID from source genesis %s: %w", genesisPath, parseErr)
	}
	if closeErr != nil {
		return "", fmt.Errorf("close source genesis %s: %w", genesisPath, closeErr)
	}
	configuredChainID := cast.ToString(appOpts.Get(flags.FlagChainID))
	if configuredChainID != "" && configuredChainID != chainID {
		return "", fmt.Errorf(
			"configured chain ID %q does not match source genesis chain ID %q",
			configuredChainID,
			chainID,
		)
	}
	return chainID, nil
}

// exportGenesisRelativePath accepts CometBFT's standard home-relative
// genesis_file form, but confines export identity to the selected home's
// config tree. Rejecting non-canonical components up front prevents a path such
// as config/../../other-home/genesis.json from selecting the zerone-1 legacy
// export exception for unrelated application state.
func exportGenesisRelativePath(configured string) (string, error) {
	if configured == "" {
		configured = filepath.Join("config", "genesis.json")
	}
	if filepath.IsAbs(configured) {
		return "", fmt.Errorf("genesis_file must be relative to the selected home")
	}
	components := strings.Split(configured, string(filepath.Separator))
	if len(components) < 2 || components[0] != "config" {
		return "", fmt.Errorf("genesis_file must name a file beneath the selected home/config directory")
	}
	for _, component := range components {
		if component == "" || component == "." || component == ".." {
			return "", fmt.Errorf("genesis_file contains an unsafe path component")
		}
	}
	clean := filepath.Clean(configured)
	if clean != configured {
		return "", fmt.Errorf("genesis_file must use a canonical relative path")
	}
	return clean, nil
}

// openExportGenesisNoFollow first resolves the selected home itself so normal
// platform aliases (for example macOS /var -> /private/var) remain usable. It
// then opens that canonical directory from the filesystem root one component
// at a time, and traverses every genesis_file edge relative to held directory
// descriptors. O_NOFOLLOW makes a symlink or raced replacement fail closed;
// the final descriptor is accepted only when it names a regular file.
func openExportGenesisNoFollow(homeDir, relativePath string) (*os.File, string, error) {
	absoluteHome, err := filepath.Abs(homeDir)
	if err != nil {
		return nil, "", fmt.Errorf("resolve source home %s: %w", homeDir, err)
	}
	canonicalHome, err := filepath.EvalSymlinks(absoluteHome)
	if err != nil {
		return nil, "", fmt.Errorf("resolve source home %s: %w", absoluteHome, err)
	}
	canonicalHome = filepath.Clean(canonicalHome)
	home, err := openAbsoluteDirectoryNoFollow(canonicalHome)
	if err != nil {
		return nil, "", fmt.Errorf("securely open source home %s: %w", canonicalHome, err)
	}

	components := strings.Split(relativePath, string(filepath.Separator))
	current := home
	currentPath := canonicalHome
	for _, component := range components[:len(components)-1] {
		nextPath := filepath.Join(currentPath, component)
		descriptor, openErr := unix.Openat(
			int(current.Fd()),
			component,
			unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC|unix.O_NONBLOCK,
			0,
		)
		if openErr != nil {
			_ = current.Close()
			return nil, "", fmt.Errorf(
				"source genesis parent %s must be a real non-symlink directory: %w",
				nextPath,
				openErr,
			)
		}
		next := os.NewFile(uintptr(descriptor), nextPath)
		if next == nil {
			_ = unix.Close(descriptor)
			_ = current.Close()
			return nil, "", fmt.Errorf("bind source genesis directory descriptor %s", nextPath)
		}
		if closeErr := current.Close(); closeErr != nil {
			_ = next.Close()
			return nil, "", fmt.Errorf("close traversed source genesis directory %s: %w", currentPath, closeErr)
		}
		current = next
		currentPath = nextPath
	}

	basename := components[len(components)-1]
	genesisPath := filepath.Join(currentPath, basename)
	descriptor, openErr := unix.Openat(
		int(current.Fd()),
		basename,
		unix.O_RDONLY|unix.O_NOFOLLOW|unix.O_CLOEXEC|unix.O_NONBLOCK,
		0,
	)
	if openErr != nil {
		_ = current.Close()
		return nil, "", fmt.Errorf(
			"source genesis %s must be a regular non-symlink file: %w",
			genesisPath,
			openErr,
		)
	}
	genesisFile := os.NewFile(uintptr(descriptor), genesisPath)
	if genesisFile == nil {
		_ = unix.Close(descriptor)
		_ = current.Close()
		return nil, "", fmt.Errorf("bind source genesis file descriptor %s", genesisPath)
	}
	if closeErr := current.Close(); closeErr != nil {
		_ = genesisFile.Close()
		return nil, "", fmt.Errorf("close source genesis parent %s: %w", currentPath, closeErr)
	}
	info, statErr := genesisFile.Stat()
	if statErr != nil {
		_ = genesisFile.Close()
		return nil, "", fmt.Errorf("inspect source genesis %s: %w", genesisPath, statErr)
	}
	if !info.Mode().IsRegular() {
		_ = genesisFile.Close()
		return nil, "", fmt.Errorf("source genesis %s must be a regular non-symlink file", genesisPath)
	}
	return genesisFile, genesisPath, nil
}

func openAbsoluteDirectoryNoFollow(path string) (*os.File, error) {
	if !filepath.IsAbs(path) || filepath.Clean(path) != path {
		return nil, fmt.Errorf("directory path must be absolute and canonical")
	}
	descriptor, err := unix.Open(
		string(filepath.Separator),
		unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC|unix.O_NONBLOCK,
		0,
	)
	if err != nil {
		return nil, fmt.Errorf("open filesystem root: %w", err)
	}
	current := os.NewFile(uintptr(descriptor), string(filepath.Separator))
	if current == nil {
		_ = unix.Close(descriptor)
		return nil, fmt.Errorf("bind filesystem root descriptor")
	}
	components := strings.Split(strings.TrimPrefix(path, string(filepath.Separator)), string(filepath.Separator))
	if len(components) == 1 && components[0] == "" {
		components = nil
	}
	for _, component := range components {
		if component == "" || component == "." || component == ".." {
			_ = current.Close()
			return nil, fmt.Errorf("directory path contains an unsafe component")
		}
		nextPath := filepath.Join(current.Name(), component)
		nextDescriptor, openErr := unix.Openat(
			int(current.Fd()),
			component,
			unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC|unix.O_NONBLOCK,
			0,
		)
		if openErr != nil {
			_ = current.Close()
			return nil, fmt.Errorf("directory component %s must be a real non-symlink directory: %w", nextPath, openErr)
		}
		next := os.NewFile(uintptr(nextDescriptor), nextPath)
		if next == nil {
			_ = unix.Close(nextDescriptor)
			_ = current.Close()
			return nil, fmt.Errorf("bind directory descriptor %s", nextPath)
		}
		if closeErr := current.Close(); closeErr != nil {
			_ = next.Close()
			return nil, fmt.Errorf("close traversed directory %s: %w", current.Name(), closeErr)
		}
		current = next
	}
	return current, nil
}

// emptyAppOptions satisfies servertypes.AppOptions for creating a temporary
// app instance used only to extract module metadata for AutoCLI.
type emptyAppOptions struct{}

func (o emptyAppOptions) Get(_ string) interface{} { return nil }

// SDK bech32 config is initialized in app.init() — no duplicate needed here.
