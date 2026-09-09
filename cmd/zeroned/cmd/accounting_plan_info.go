package cmd

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"cosmossdk.io/log"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/server"
	"github.com/spf13/cobra"

	"github.com/zerone-chain/zerone/app"
)

func validateAccountingPlanTuple(chainID string, height int64, appHash string) error {
	if chainID == "" || len(chainID) > 128 || strings.TrimSpace(chainID) != chainID {
		return fmt.Errorf("--expected-chain-id must be a nonempty exact chain ID")
	}
	if height <= 0 {
		return fmt.Errorf("--expected-height must be positive")
	}
	decoded, err := hex.DecodeString(appHash)
	if err != nil || len(decoded) != sha256.Size || appHash != strings.ToLower(appHash) {
		return fmt.Errorf("--expected-app-hash must be exactly 64 lowercase hexadecimal characters")
	}
	return nil
}

// The source database is never opened. Existing activation-preflight helpers
// reject symlinks and copy file contents into an owned, disposable directory.
func accountingPlanInfoCmd() *cobra.Command {
	command := &cobra.Command{
		Use:   "accounting-plan-info",
		Short: "Prepare an accounting upgrade commitment from a stopped predecessor",
		Long: `Read a private byte-for-byte clone of a stopped predecessor application
database, bind an independently expected chain/height/AppHash, and report the
canonical accounting-authority-v1 plan info. This command does not schedule an
upgrade, freeze activity, reconcile disputed claims, or authorize activation.
Both custom stores and their custody must remain unchanged until the handoff.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) (err error) {
			defer func() {
				if recovered := recover(); recovered != nil {
					err = fmt.Errorf("accounting plan preparation failed: %v", recovered)
				}
			}()
			chainID, _ := cmd.Flags().GetString("expected-chain-id")
			height, _ := cmd.Flags().GetInt64("expected-height")
			appHash, _ := cmd.Flags().GetString("expected-app-hash")
			if err := validateAccountingPlanTuple(chainID, height, appHash); err != nil {
				return err
			}
			clientCtx := client.GetClientContextFromCmd(cmd)
			serverCtx := server.GetServerContextFromCmd(cmd)
			home := clientCtx.HomeDir
			if home == "" {
				home = serverCtx.Config.RootDir
			}
			if home == "" {
				return fmt.Errorf("node home is empty")
			}
			genesisPath := filepath.Join(home, "config", "genesis.json")
			genesisChainID, genesisDigest, err := readActivationGenesisIdentity(genesisPath)
			if err != nil {
				return fmt.Errorf("read source genesis identity: %w", err)
			}
			if genesisChainID != chainID {
				return fmt.Errorf("expected chain ID differs from source genesis")
			}
			sourceDB := filepath.Join(home, "data", "application.db")
			before, err := buildActivationSourceManifest(sourceDB)
			if err != nil {
				return fmt.Errorf("manifest source application DB: %w", err)
			}
			copyHome, err := os.MkdirTemp("", "zerone-accounting-plan-*")
			if err != nil {
				return err
			}
			defer os.RemoveAll(copyHome)
			copyData := filepath.Join(copyHome, "data")
			if err := os.MkdirAll(copyData, 0o700); err != nil {
				return err
			}
			copyDB := filepath.Join(copyData, "application.db")
			if err := cloneActivationSourceTree(sourceDB, copyDB); err != nil {
				return err
			}
			copied, err := buildActivationSourceManifest(copyDB)
			if err != nil {
				return err
			}
			if copied != before {
				return fmt.Errorf("private application DB copy differs from source")
			}
			db, err := dbm.NewDB("application", server.GetAppDBBackend(serverCtx.Viper), copyData)
			if err != nil {
				return fmt.Errorf("open private application DB: %w", err)
			}
			dbOpen := true
			defer func() {
				if dbOpen {
					_ = db.Close()
				}
			}()
			// Constructor diagnostics must not contaminate the JSON stdout channel.
			// Preparation errors are returned normally to Cobra's error stream.
			candidate := app.NewActivationPreflightApp(log.NewNopLogger(), db, nil, false,
				activationPreflightAppOptions{delegate: serverCtx.Viper, home: copyHome, chainID: chainID},
				baseapp.SetChainID(chainID), baseapp.SetIAVLDisableFastNode(true))
			if err := candidate.LoadLatestVersion(); err != nil {
				return fmt.Errorf("load private application DB: %w", err)
			}
			commitID := candidate.CommitMultiStore().LastCommitID()
			if commitID.Version != height || hex.EncodeToString(commitID.Hash) != appHash {
				return fmt.Errorf("committed database height/AppHash differs from independently expected tuple")
			}
			ctx := candidate.NewUncachedContext(true, cmtproto.Header{Height: height, ChainID: chainID})
			info, err := candidate.BuildAccountingAuthorityPlanInfo(ctx)
			if err != nil {
				return fmt.Errorf("build accounting state commitment: %w", err)
			}
			if err := db.Close(); err != nil {
				return err
			}
			dbOpen = false
			after, err := buildActivationSourceManifest(sourceDB)
			if err != nil {
				return err
			}
			if after != before {
				return fmt.Errorf("source application DB changed during preparation; stop the node and retry")
			}
			finalChainID, finalGenesisDigest, err := readActivationGenesisIdentity(genesisPath)
			if err != nil {
				return err
			}
			if finalChainID != chainID || finalGenesisDigest != genesisDigest {
				return fmt.Errorf("source genesis changed during preparation")
			}
			infoDigest := sha256.Sum256([]byte(info))
			report := struct {
				Schema               string `json:"schema"`
				ChainID              string `json:"chain_id"`
				Height               int64  `json:"height"`
				AppHash              string `json:"app_hash"`
				GenesisSHA256        string `json:"genesis_sha256"`
				SourceManifestSHA256 string `json:"source_database_manifest_sha256"`
				PlanName             string `json:"plan_name"`
				PlanInfo             string `json:"plan_info"`
				PlanInfoSHA256       string `json:"plan_info_sha256"`
			}{"zerone.accounting-authority/preparation-v1", chainID, height, appHash, genesisDigest, before.SHA256, app.UpgradeNameAccountingAuthorityV1, info, hex.EncodeToString(infoDigest[:])}
			encoded, err := json.MarshalIndent(report, "", "  ")
			if err != nil {
				return err
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), string(encoded))
			return err
		},
	}
	command.Flags().String("expected-chain-id", "", "independently observed chain ID")
	command.Flags().Int64("expected-height", 0, "independently observed committed height")
	command.Flags().String("expected-app-hash", "", "independently observed lowercase committed AppHash")
	return command
}
