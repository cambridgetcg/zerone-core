package cmd

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/spf13/cobra"

	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/server"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"

	"github.com/zerone-chain/zerone/app"
)

type activationPreflightAppOptions struct {
	delegate servertypes.AppOptions
	home     string
	chainID  string
}

func (o activationPreflightAppOptions) Get(key string) interface{} {
	if key == flags.FlagHome {
		return o.home
	}
	if key == flags.FlagChainID {
		return o.chainID
	}
	return o.delegate.Get(key)
}

type activationSourceManifest struct {
	SHA256    string
	FileCount int
	Bytes     int64
}

func activationPreflightCmd() *cobra.Command {
	command := &cobra.Command{
		Use:   "verify-activation-prestate",
		Short: "Verify the committed H-1 upgrade/incident prestate",
		Long: `Run the exact complete-IAVL/root-CommitInfo verifier used by the
guarded upgrade handlers and print a stable JSON report. Run this only against
a stopped node or an isolated copy of its home directory. The command never
opens the source application database: it verifies a byte-for-byte private
clone, binds a required independently expected chain/H-1/AppHash tuple, and
refuses the result if the source DB or genesis changes during verification.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			clientCtx := client.GetClientContextFromCmd(cmd)
			serverCtx := server.GetServerContextFromCmd(cmd)
			home := clientCtx.HomeDir
			if home == "" {
				home = serverCtx.Config.RootDir
			}
			if home == "" {
				return fmt.Errorf("node home is empty")
			}
			serverCtx.Config.SetRoot(home)

			expectedChainID, err := cmd.Flags().GetString("expected-chain-id")
			if err != nil {
				return err
			}
			expectedHeight, err := cmd.Flags().GetInt64("expected-height")
			if err != nil {
				return err
			}
			expectedAppHash, err := cmd.Flags().GetString("expected-app-hash")
			if err != nil {
				return err
			}
			if expectedChainID == "" {
				return fmt.Errorf("--expected-chain-id is required")
			}
			if expectedHeight <= 0 {
				return fmt.Errorf("--expected-height must be positive")
			}
			if expectedAppHash != strings.ToLower(expectedAppHash) {
				return fmt.Errorf(
					"--expected-app-hash must be lowercase hexadecimal",
				)
			}
			appHashBytes, err := hex.DecodeString(expectedAppHash)
			if err != nil || len(appHashBytes) != sha256.Size {
				return fmt.Errorf(
					"--expected-app-hash must be exactly %d lowercase hexadecimal characters",
					sha256.Size*2,
				)
			}

			sourceGenesis := filepath.Join(home, "config", "genesis.json")
			genesisChainID, genesisSHA256, err :=
				readActivationGenesisIdentity(sourceGenesis)
			if err != nil {
				return fmt.Errorf("read source genesis identity: %w", err)
			}
			if genesisChainID != expectedChainID {
				return fmt.Errorf(
					"expected chain id %q does not match source genesis chain id %q",
					expectedChainID,
					genesisChainID,
				)
			}
			sourceUpgradeInfo := filepath.Join(
				home,
				"data",
				upgradetypes.UpgradeInfoFilename,
			)
			diskPlan, upgradeInfoSHA256, err :=
				readActivationUpgradeInfo(sourceUpgradeInfo)
			if err != nil {
				return fmt.Errorf("read source upgrade-info: %w", err)
			}
			if diskPlan.Height != expectedHeight+1 {
				return fmt.Errorf(
					"upgrade-info plan %q targets height %d, not required H=%d derived from expected H-1=%d",
					diskPlan.Name,
					diskPlan.Height,
					expectedHeight+1,
					expectedHeight,
				)
			}

			sourceApplicationDB := filepath.Join(
				home,
				"data",
				"application.db",
			)
			sourceBefore, err := buildActivationSourceManifest(
				sourceApplicationDB,
			)
			if err != nil {
				return fmt.Errorf("manifest source application DB: %w", err)
			}
			copyHome, err := os.MkdirTemp(
				"",
				"zerone-activation-preflight-*",
			)
			if err != nil {
				return fmt.Errorf("create isolated verification home: %w", err)
			}
			defer os.RemoveAll(copyHome)
			copyData := filepath.Join(copyHome, "data")
			if err := os.MkdirAll(copyData, 0o700); err != nil {
				return fmt.Errorf("create isolated data directory: %w", err)
			}
			copyApplicationDB := filepath.Join(copyData, "application.db")
			if err := cloneActivationSourceTree(
				sourceApplicationDB,
				copyApplicationDB,
			); err != nil {
				return fmt.Errorf("clone source application DB: %w", err)
			}
			copyManifest, err := buildActivationSourceManifest(
				copyApplicationDB,
			)
			if err != nil {
				return fmt.Errorf("manifest verification copy: %w", err)
			}
			if copyManifest != sourceBefore {
				return fmt.Errorf(
					"verification copy manifest differs from source: source=%+v copy=%+v",
					sourceBefore,
					copyManifest,
				)
			}

			db, err := dbm.NewDB(
				"application",
				server.GetAppDBBackend(serverCtx.Viper),
				copyData,
			)
			if err != nil {
				return fmt.Errorf(
					"open application DB (stop the daemon or use a copied home): %w",
					err,
				)
			}
			dbOpen := true
			defer func() {
				if dbOpen {
					_ = db.Close()
				}
			}()

			appOptions := activationPreflightAppOptions{
				delegate: serverCtx.Viper,
				home:     copyHome,
				chainID:  genesisChainID,
			}
			zeroneApp := app.NewActivationPreflightApp(
				serverCtx.Logger,
				db,
				nil,
				false,
				appOptions,
				baseapp.SetChainID(genesisChainID),
				baseapp.SetIAVLDisableFastNode(true),
			)
			if err := zeroneApp.LoadLatestVersion(); err != nil {
				return fmt.Errorf(
					"load isolated application copy: %w",
					err,
				)
			}
			report, err := zeroneApp.VerifyScheduledActivationPrestate()
			if err != nil {
				return fmt.Errorf("activation prestate verification failed: %w", err)
			}
			if report.Height != expectedHeight ||
				report.AppHash != expectedAppHash {
				return fmt.Errorf(
					"verified application tuple chain=%q height=%d app_hash=%s does not match required tuple chain=%q height=%d app_hash=%s",
					genesisChainID,
					report.Height,
					report.AppHash,
					expectedChainID,
					expectedHeight,
					expectedAppHash,
				)
			}
			diskInfoDigest := sha256.Sum256([]byte(diskPlan.Info))
			if diskPlan.Name != report.PlanName ||
				diskPlan.Height != report.PlanHeight ||
				fmt.Sprintf("%x", diskInfoDigest) !=
					report.PlanInfoSHA256 {
				return fmt.Errorf(
					"source upgrade-info is not the exact committed plan: disk=(%q,%d,%x) report=(%q,%d,%s)",
					diskPlan.Name,
					diskPlan.Height,
					diskInfoDigest,
					report.PlanName,
					report.PlanHeight,
					report.PlanInfoSHA256,
				)
			}
			if err := db.Close(); err != nil {
				return fmt.Errorf("close isolated application DB: %w", err)
			}
			dbOpen = false
			sourceAfterVerification, err :=
				buildActivationSourceManifest(sourceApplicationDB)
			if err != nil {
				return fmt.Errorf(
					"manifest source node data after verification: %w",
					err,
				)
			}
			if sourceAfterVerification != sourceBefore {
				return fmt.Errorf(
					"source application DB changed during verification; stop the daemon and retry",
				)
			}
			finalGenesisChainID, finalGenesisSHA256, err :=
				readActivationGenesisIdentity(sourceGenesis)
			if err != nil {
				return fmt.Errorf(
					"re-read source genesis identity after verification: %w",
					err,
				)
			}
			if finalGenesisChainID != genesisChainID ||
				finalGenesisSHA256 != genesisSHA256 {
				return fmt.Errorf(
					"source genesis changed during verification; refusing readiness report",
				)
			}
			finalDiskPlan, finalUpgradeInfoSHA256, err :=
				readActivationUpgradeInfo(sourceUpgradeInfo)
			if err != nil {
				return fmt.Errorf(
					"re-read source upgrade-info after verification: %w",
					err,
				)
			}
			if finalUpgradeInfoSHA256 != upgradeInfoSHA256 ||
				finalDiskPlan.Name != diskPlan.Name ||
				finalDiskPlan.Height != diskPlan.Height ||
				finalDiskPlan.Info != diskPlan.Info {
				return fmt.Errorf(
					"source upgrade-info changed during verification; refusing readiness report",
				)
			}
			report.ChainID = genesisChainID
			report.GenesisSHA256 = genesisSHA256
			report.UpgradeInfoSHA256 = upgradeInfoSHA256
			report.SourceDataManifestSHA256 = sourceBefore.SHA256
			report.SourceDataFileCount = sourceBefore.FileCount
			report.SourceDataBytes = sourceBefore.Bytes
			report.CompletedChecks = append(
				report.CompletedChecks,
				"source_database_never_opened",
				"isolated_copy_manifest_exact",
				"source_manifest_unchanged_after_verification",
				"expected_chain_height_app_hash_tuple_matched",
				"genesis_chain_id_and_digest_bound",
				"local_upgrade_info_exactly_matches_committed_plan",
			)
			canonicalReport, err := json.Marshal(report)
			if err != nil {
				return fmt.Errorf(
					"encode canonical activation preflight report: %w",
					err,
				)
			}
			reportDigest := sha256.Sum256(canonicalReport)
			report.ReportSHA256 = fmt.Sprintf("%x", reportDigest)
			output, err := json.MarshalIndent(report, "", "  ")
			if err != nil {
				return fmt.Errorf("encode activation preflight report: %w", err)
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), string(output))
			return err
		},
	}
	command.Flags().String(
		"expected-chain-id",
		"",
		"independently observed chain ID required for readiness",
	)
	command.Flags().Int64(
		"expected-height",
		0,
		"independently observed committed H-1 height required for readiness",
	)
	command.Flags().String(
		"expected-app-hash",
		"",
		"independently observed lowercase committed H-1 AppHash required for readiness",
	)
	return command
}

func readActivationGenesisIdentity(path string) (string, string, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return "", "", err
	}
	if !info.Mode().IsRegular() {
		return "", "", fmt.Errorf(
			"genesis must be a regular non-symlink file, got mode %s",
			info.Mode(),
		)
	}
	file, err := os.Open(path)
	if err != nil {
		return "", "", err
	}
	hasher := sha256.New()
	chainID, parseErr := genutiltypes.ParseChainIDFromGenesis(
		io.TeeReader(file, hasher),
	)
	_, hashTailErr := io.Copy(hasher, file)
	closeErr := file.Close()
	if parseErr != nil {
		return "", "", parseErr
	}
	if hashTailErr != nil {
		return "", "", hashTailErr
	}
	if closeErr != nil {
		return "", "", closeErr
	}
	if chainID == "" {
		return "", "", fmt.Errorf("genesis chain_id is empty")
	}
	return chainID, fmt.Sprintf("%x", hasher.Sum(nil)), nil
}

func readActivationUpgradeInfo(
	path string,
) (upgradetypes.Plan, string, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return upgradetypes.Plan{}, "", err
	}
	if !info.Mode().IsRegular() {
		return upgradetypes.Plan{}, "", fmt.Errorf(
			"upgrade-info must be a regular non-symlink file, got mode %s",
			info.Mode(),
		)
	}
	if info.Size() <= 0 || info.Size() > 64<<10 {
		return upgradetypes.Plan{}, "", fmt.Errorf(
			"upgrade-info size must be between 1 and %d bytes, got %d",
			64<<10,
			info.Size(),
		)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return upgradetypes.Plan{}, "", err
	}
	var plan upgradetypes.Plan
	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	if err := decoder.Decode(&plan); err != nil {
		return upgradetypes.Plan{}, "", err
	}
	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return upgradetypes.Plan{}, "", fmt.Errorf(
				"upgrade-info contains trailing JSON",
			)
		}
		return upgradetypes.Plan{}, "", err
	}
	if err := plan.ValidateBasic(); err != nil {
		return upgradetypes.Plan{}, "", err
	}
	digest := sha256.Sum256(raw)
	return plan, fmt.Sprintf("%x", digest), nil
}

func buildActivationSourceManifest(
	root string,
) (activationSourceManifest, error) {
	entries := make([]string, 0)
	fileCount := 0
	var totalBytes int64
	err := filepath.WalkDir(
		root,
		func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			relative, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			if relative == "." && entry.IsDir() {
				return nil
			}
			if relative == "." {
				relative = filepath.Base(root)
			}
			info, err := entry.Info()
			if err != nil {
				return err
			}
			switch {
			case entry.IsDir():
				entries = append(
					entries,
					fmt.Sprintf("d\x00%s\x00%o", filepath.ToSlash(relative), info.Mode().Perm()),
				)
				return nil
			case info.Mode().IsRegular():
				file, err := os.Open(path)
				if err != nil {
					return err
				}
				hasher := sha256.New()
				_, copyErr := io.Copy(hasher, file)
				closeErr := file.Close()
				if copyErr != nil {
					return copyErr
				}
				if closeErr != nil {
					return closeErr
				}
				entries = append(
					entries,
					fmt.Sprintf(
						"f\x00%s\x00%o\x00%d\x00%x",
						filepath.ToSlash(relative),
						info.Mode().Perm(),
						info.Size(),
						hasher.Sum(nil),
					),
				)
				fileCount++
				totalBytes += info.Size()
				return nil
			default:
				return fmt.Errorf(
					"unsupported non-regular source entry %q with mode %s",
					relative,
					info.Mode(),
				)
			}
		},
	)
	if err != nil {
		return activationSourceManifest{}, err
	}
	sort.Strings(entries)
	hasher := sha256.New()
	_, _ = hasher.Write([]byte("zerone/activation-source-manifest/v1\x00"))
	var scalar [8]byte
	binary.BigEndian.PutUint64(scalar[:], uint64(len(entries)))
	_, _ = hasher.Write(scalar[:])
	for _, entry := range entries {
		binary.BigEndian.PutUint64(scalar[:], uint64(len(entry)))
		_, _ = hasher.Write(scalar[:])
		_, _ = hasher.Write([]byte(entry))
	}
	return activationSourceManifest{
		SHA256:    fmt.Sprintf("%x", hasher.Sum(nil)),
		FileCount: fileCount,
		Bytes:     totalBytes,
	}, nil
}

func cloneActivationSourceTree(source, destination string) error {
	return filepath.WalkDir(
		source,
		func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			relative, err := filepath.Rel(source, path)
			if err != nil {
				return err
			}
			target := destination
			if relative != "." {
				target = filepath.Join(destination, relative)
			}
			info, err := entry.Info()
			if err != nil {
				return err
			}
			switch {
			case entry.IsDir():
				return os.Mkdir(target, info.Mode().Perm())
			case info.Mode().IsRegular():
				input, err := os.Open(path)
				if err != nil {
					return err
				}
				output, err := os.OpenFile(
					target,
					os.O_CREATE|os.O_EXCL|os.O_WRONLY,
					info.Mode().Perm(),
				)
				if err != nil {
					_ = input.Close()
					return err
				}
				_, copyErr := io.Copy(output, input)
				syncErr := output.Sync()
				outputCloseErr := output.Close()
				inputCloseErr := input.Close()
				if copyErr != nil {
					return copyErr
				}
				if syncErr != nil {
					return syncErr
				}
				if outputCloseErr != nil {
					return outputCloseErr
				}
				return inputCloseErr
			default:
				return fmt.Errorf(
					"refusing to clone non-regular source entry %q with mode %s",
					relative,
					info.Mode(),
				)
			}
		},
	)
}
