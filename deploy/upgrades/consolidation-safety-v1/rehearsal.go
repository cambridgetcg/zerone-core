//go:build ignore

// Compile this file separately against each frozen source closure, never trunk.
// No daemon, RPC, signer, genesis import, module-map patch or handler shortcut.
package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"cosmossdk.io/core/header"
	"cosmossdk.io/log"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	abci "github.com/cometbft/cometbft/abci/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	zeroneapp "github.com/zerone-chain/zerone/app"
)

const upgradeName = "consolidation-safety-v1"
const oldCommit = "2e37c4c86c31e67515d6aa07b6fd406083e012ea"
const h1Commit = "65c19cd8b00bdfff9b80705b776fd0d49719398a"

var sourceCommit string // Set by the frozen-source build, recorded in every receipt.

type options map[string]interface{}

func (o options) Get(name string) interface{} { return o[name] }

func check(err error) {
	if err != nil {
		panic(err)
	}
}
func require(ok bool, message string) {
	if !ok {
		panic(message)
	}
}
func contextAt(a *zeroneapp.ZeroneApp, chain string, height int64) sdk.Context {
	return a.NewUncachedContext(false, cmtproto.Header{Height: height, ChainID: chain}).
		WithHeaderInfo(header.Info{Height: height, ChainID: chain})
}
func open(home, chain string) *zeroneapp.ZeroneApp {
	info, err := os.Lstat(filepath.Join(home, "data", "application.db"))
	check(err)
	require(info.IsDir() && info.Mode()&os.ModeSymlink == 0, "missing regular application.db directory")
	db, err := dbm.NewDB("application", dbm.GoLevelDBBackend, filepath.Join(home, "data"))
	check(err)
	// true is essential: H1's real constructor runs its lineage wall here.
	return zeroneapp.NewZeroneApp(log.NewNopLogger(), db, nil, true,
		options{"home": home, "oracle.enabled": false, "unsafe-skip-upgrades": []int{}},
		baseapp.SetChainID(chain))
}
func snapshot(a *zeroneapp.ZeroneApp, chain string) map[string]interface{} {
	ctx := contextAt(a, chain, a.LastBlockHeight())
	vm, err := a.UpgradeKeeper.GetModuleVersionMap(ctx)
	check(err)
	done, err := a.UpgradeKeeper.GetDoneHeight(ctx, upgradeName)
	check(err)
	return map[string]interface{}{"height": a.LastBlockHeight(),
		"app_hash":        strings.ToLower(hex.EncodeToString(a.LastCommitID().Hash)),
		"module_versions": vm, "done_height": done,
		"marker": a.KnowledgeKeeper.ReadMigrationMarker(ctx, "upgrade_marker_"+upgradeName)}
}
func finalize(a *zeroneapp.ZeroneApp, height int64, at time.Time) {
	result, err := a.FinalizeBlock(&abci.RequestFinalizeBlock{Height: height, Time: at})
	check(err)
	_, err = a.Commit()
	check(err)
	require(a.LastBlockHeight() == height, "commit height mismatch")
	require(bytes.Equal(a.LastCommitID().Hash, result.AppHash), "commit hash differs from FinalizeBlock")
}
func completed(a *zeroneapp.ZeroneApp, chain string, height int64) {
	ctx := contextAt(a, chain, a.LastBlockHeight())
	vm, err := a.UpgradeKeeper.GetModuleVersionMap(ctx)
	check(err)
	require(reflect.DeepEqual(vm, a.ModuleManager.GetVersionMap()), "full target VersionMap mismatch")
	for name, version := range map[string]uint64{"knowledge": 6, "claiming_pot": 2, "liquiditypool": 5, "vesting_rewards": 1} {
		require(vm[name] == version, "H1 boundary mismatch: "+name)
	}
	done, err := a.UpgradeKeeper.GetDoneHeight(ctx, upgradeName)
	check(err)
	require(done == height, "H1 done height mismatch")
	require(a.KnowledgeKeeper.ReadMigrationMarker(ctx, "upgrade_marker_"+upgradeName) == "migrated", "H1 marker missing")
	_, err = a.UpgradeKeeper.GetUpgradePlan(ctx)
	require(errors.Is(err, upgradetypes.ErrNoUpgradePlanFound), "pending plan remains after H1")
}

func run() {
	mode := flag.String("mode", "", "stage or migrate")
	home := flag.String("home", "", "disposable working copy created by state-copy-drill.py")
	chain := flag.String("chain-id", "", "zerone-testnet-1 or zerone-h1-fixture-1 only")
	height := flag.Int64("expected-height", 0, "exact starting committed height")
	hash := flag.String("expected-app-hash", "", "exact starting application hash")
	at := flag.String("next-block-time", "", "explicit synthetic next block RFC3339 time")
	info := flag.String("plan-info", "", "canonical H1 plan info")
	flag.Parse()
	require(*mode == "stage" || *mode == "migrate", "invalid mode")
	require(*chain == "zerone-testnet-1" || *chain == "zerone-h1-fixture-1", "not a testnet drill chain")
	require(*home != "", "explicit disposable home required")
	marker, err := os.ReadFile(filepath.Join(*home, ".h1-disposable-copy"))
	check(err)
	require(string(marker) == "local-only; never a validator home\n", "wrapper ownership marker missing")
	require((*mode == "stage" && sourceCommit == oldCommit) || (*mode == "migrate" && sourceCommit == h1Commit), "wrong helper source")
	blockTime, err := time.Parse(time.RFC3339, *at)
	check(err)
	a := open(*home, *chain)
	before := snapshot(a, *chain)
	require(a.LastBlockHeight() == *height && *height > 0, "source height mismatch")
	require(before["app_hash"] == strings.ToLower(*hash), "source app hash mismatch")
	ctx := contextAt(a, *chain, *height)
	var target int64
	switch *mode {
	case "stage":
		require(!a.UpgradeKeeper.HasHandler(upgradeName), "old binary unexpectedly has H1 handler")
		_, err := a.UpgradeKeeper.GetUpgradePlan(ctx)
		require(errors.Is(err, upgradetypes.ErrNoUpgradePlanFound), "existing plan: refuse to replace it")
		done, err := a.UpgradeKeeper.GetDoneHeight(ctx, upgradeName)
		check(err)
		require(done == 0, "H1 already completed")
		target = *height + 2
		plan := upgradetypes.Plan{Name: upgradeName, Height: target, Info: *info}
		// LOCAL-ONLY mutation, not governance approval. No application inventory changes.
		check(a.UpgradeKeeper.ScheduleUpgrade(ctx, plan))
		finalize(a, *height+1, blockTime)
		check(a.Close())
		// Reopen so the old halt is based on the committed plan, not a cache.
		a = open(*home, *chain)
		committed, err := a.UpgradeKeeper.GetUpgradePlan(contextAt(a, *chain, target-1))
		check(err)
		require(committed.Name == plan.Name && committed.Height == plan.Height && committed.Info == plan.Info, "plan did not commit")
		_, err = a.FinalizeBlock(&abci.RequestFinalizeBlock{Height: target, Time: blockTime.Add(time.Second)})
		require(err != nil && strings.Contains(err.Error(), `UPGRADE "consolidation-safety-v1" NEEDED`), "old FinalizeBlock did not halt for H1")
		// NEVER commit the failed block. x/upgrade writes the real disk halt packet.
		disk, err := a.UpgradeKeeper.ReadUpgradeInfoFromDisk()
		check(err)
		require(disk.Name == plan.Name && disk.Height == plan.Height && disk.Info == plan.Info, "disk packet mismatch")
	case "migrate":
		// Getting here proves the actual H1 loadLatest=true constructor accepted pending lineage.
		plan, err := a.UpgradeKeeper.GetUpgradePlan(ctx)
		check(err)
		target = *height + 1
		require(plan.Name == upgradeName && plan.Height == target && plan.Info == *info, "H1 plan mismatch")
		finalize(a, target, blockTime)
		completed(a, *chain, target)
		check(a.Close())
		a = open(*home, *chain) // actual completed-lineage restart wall
		completed(a, *chain, target)
		finalize(a, target+1, blockTime.Add(time.Second))
		check(a.Close())
		a = open(*home, *chain)
		completed(a, *chain, target)
	}
	after := snapshot(a, *chain)
	check(a.Close())
	check(json.NewEncoder(os.Stdout).Encode(map[string]interface{}{
		"schema": "zerone-h1-offline-abci-phase/v1", "passed": true, "mode": *mode,
		"source_commit": sourceCommit, "chain_id": *chain, "target_height": target,
		"before": before, "after": after, "next_block_time": *at,
		"requests": "empty transactions, votes, evidence and proposer; one-second intervals",
		"scope":    "synthetic ABCI continuation; no Comet consensus, signatures, RPC or governance vote",
	}))
}
func main() {
	defer func() {
		if failure := recover(); failure != nil {
			fmt.Fprintln(os.Stderr, "REFUSED:", failure)
			os.Exit(1)
		}
	}()
	run()
}
