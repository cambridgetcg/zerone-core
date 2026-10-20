// consensus-repairs-census reads only an explicitly attested, stopped-node copy.
// No network, genesis import, key access, scheduling, reconciliation or DB write.
package main

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"cosmossdk.io/log"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/zerone-chain/zerone/app"
)

type options map[string]interface{}

func (o options) Get(key string) interface{} { return o[key] }

func main() { os.Exit(run(os.Args[1:], os.Stdout, os.Stderr)) }
func run(args []string, out, diagnostics io.Writer) (code int) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintf(diagnostics, "census refused: %v\n", r)
			code = 1
		}
	}()
	f := flag.NewFlagSet("consensus-repairs-census", flag.ContinueOnError)
	f.SetOutput(diagnostics)
	home := f.String("home", "", "absolute stopped-node COPY home (only data/application.db is opened)")
	chain := f.String("chain-id", "", "independently observed chain identity (asserted, not inferred from database)")
	height := f.Int64("expected-height", 0, "independently observed committed height")
	hash := f.String("expected-app-hash", "", "exact lowercase 32-byte committed AppHash")
	copied := f.Bool("copied-db", false, "attest this is a disposable copy, not a live or only database")
	if err := f.Parse(args); err != nil {
		return 1
	}
	decoded, err := hex.DecodeString(*hash)
	if f.NArg() != 0 || !*copied || *chain == "" || *height <= 0 || err != nil || len(decoded) != 32 || strings.ToLower(*hash) != *hash {
		fmt.Fprintln(diagnostics, "require --copied-db, --home, --chain-id, --expected-height and lowercase --expected-app-hash")
		return 1
	}
	data, err := copiedDataDirectory(*home)
	if err != nil {
		fmt.Fprintln(diagnostics, err)
		return 1
	}
	db, err := dbm.NewGoLevelDBWithOpts("application", data, &opt.Options{ReadOnly: true, ErrorIfMissing: true})
	if err != nil {
		fmt.Fprintln(diagnostics, err)
		return 1
	}
	closed := false
	defer func() {
		if !closed {
			_ = db.Close()
		}
	}()
	candidate := app.NewActivationPreflightApp(log.NewNopLogger(), db, nil, false, options{flags.FlagHome: *home}, baseapp.SetChainID(*chain), baseapp.SetIAVLDisableFastNode(true))
	if err := candidate.LoadLatestVersion(); err != nil {
		fmt.Fprintln(diagnostics, err)
		return 1
	}
	commit := candidate.CommitMultiStore().LastCommitID()
	if commit.Version != *height || hex.EncodeToString(commit.Hash) != *hash {
		fmt.Fprintln(diagnostics, "committed height/AppHash differs from independently expected tuple")
		return 1
	}
	ctx := candidate.NewUncachedContext(true, cmtproto.Header{Height: *height, ChainID: *chain})
	report, censusErr := candidate.CensusConsensusRepairsV1(ctx)
	versions, versionErr := candidate.UpgradeKeeper.GetModuleVersionMap(ctx)
	closeErr := db.Close()
	closed = true
	if err := errors.Join(versionErr, closeErr); err != nil {
		fmt.Fprintln(diagnostics, err)
		return 1
	}
	value := struct {
		Schema   string                     `json:"schema"`
		ChainID  string                     `json:"asserted_chain_id"`
		AppHash  string                     `json:"app_hash"`
		Versions map[string]uint64          `json:"module_versions"`
		Census   app.ConsensusRepairsCensus `json:"census"`
		Passed   bool                       `json:"passed"`
		Error    string                     `json:"error,omitempty"`
	}{Schema: "zerone.consensus-repairs-census/v1", ChainID: *chain, AppHash: *hash, Versions: versions, Census: report, Passed: censusErr == nil}
	if censusErr != nil {
		value.Error = censusErr.Error()
	}
	encoder := json.NewEncoder(out)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(value); err != nil {
		fmt.Fprintln(diagnostics, err)
		return 1
	}
	if censusErr != nil {
		return 2
	}
	return 0
}

func copiedDataDirectory(home string) (string, error) {
	if !filepath.IsAbs(home) || filepath.Clean(home) != home || home == string(filepath.Separator) {
		return "", fmt.Errorf("--home must be an exact absolute copy path")
	}
	// Refuse known private/live identities by path, before reading them. Reject
	// every symlink component, not just a final alias into a protected home.
	for _, part := range strings.Split(home, string(filepath.Separator)) {
		if part == ".zeroned" || part == "zerone-private" {
			return "", fmt.Errorf("refusing protected node home")
		}
	}
	data := filepath.Join(home, "data")
	dbPath := filepath.Join(data, "application.db")
	for current := dbPath; current != filepath.Dir(current); current = filepath.Dir(current) {
		info, err := os.Lstat(current)
		if err != nil {
			return "", err
		}
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("copy path components must be non-symlink directories")
		}
	}
	return data, nil
}
