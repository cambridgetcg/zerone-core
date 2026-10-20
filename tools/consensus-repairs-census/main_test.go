package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"cosmossdk.io/log"
	sdkmath "cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	abci "github.com/cometbft/cometbft/abci/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/baseapp"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdked25519 "github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/require"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/zerone-chain/zerone/app"
	gov "github.com/zerone-chain/zerone/x/gov/types"
	knowledge "github.com/zerone-chain/zerone/x/knowledge/types"
	customstaking "github.com/zerone-chain/zerone/x/staking/types"
	"google.golang.org/protobuf/proto"
)

func sourceCopy(t *testing.T, badMarker bool, extraStore ...bool) (string, int64, string) {
	t.Helper()
	home, err := filepath.EvalSymlinks(t.TempDir())
	require.NoError(t, err)
	data := filepath.Join(home, "data")
	require.NoError(t, os.Mkdir(data, 0700))
	db, err := dbm.NewGoLevelDBWithOpts("application", data, &opt.Options{})
	require.NoError(t, err)
	a := app.NewZeroneApp(log.NewNopLogger(), db, nil, false, options{"home": home}, baseapp.SetChainID("zerone-rehearsal-census"), baseapp.SetIAVLDisableFastNode(true))
	if len(extraStore) > 0 && extraStore[0] {
		a.CommitMultiStore().MountStoreWithDB(storetypes.NewKVStoreKey("unmounted-copy-store"), storetypes.StoreTypeIAVL, nil)
	}
	require.NoError(t, a.LoadLatestVersion())
	cdc := a.AppCodec()
	genesis := a.DefaultGenesis()
	pub := sdked25519.GenPrivKey().PubKey()
	address := sdk.AccAddress(pub.Address())
	valAddr := sdk.ValAddress(pub.Address())
	any, err := codectypes.NewAnyWithValue(pub)
	require.NoError(t, err)
	genesis["auth"] = cdc.MustMarshalJSON(authtypes.NewGenesisState(authtypes.DefaultParams(), []authtypes.GenesisAccount{authtypes.NewBaseAccount(address, pub, 0, 0)}))
	validator := stakingtypes.Validator{OperatorAddress: valAddr.String(), ConsensusPubkey: any, Status: stakingtypes.Bonded, Tokens: sdk.DefaultPowerReduction, DelegatorShares: sdkmath.LegacyOneDec(), Commission: stakingtypes.NewCommission(sdkmath.LegacyZeroDec(), sdkmath.LegacyZeroDec(), sdkmath.LegacyZeroDec()), MinSelfDelegation: sdkmath.ZeroInt()}
	genesis["staking"] = cdc.MustMarshalJSON(stakingtypes.NewGenesisState(stakingtypes.DefaultParams(), []stakingtypes.Validator{validator}, []stakingtypes.Delegation{stakingtypes.NewDelegation(address.String(), valAddr.String(), sdkmath.LegacyOneDec())}))
	coins := sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdk.DefaultPowerReduction))
	genesis["bank"] = cdc.MustMarshalJSON(banktypes.NewGenesisState(banktypes.DefaultGenesisState().Params, []banktypes.Balance{{Address: authtypes.NewModuleAddress(stakingtypes.BondedPoolName).String(), Coins: coins}}, coins, nil, nil))
	raw, err := json.Marshal(genesis)
	require.NoError(t, err)
	ctx := a.NewUncachedContext(false, cmtproto.Header{ChainID: "zerone-rehearsal-census"})
	_, err = a.InitChainer(ctx, &abci.RequestInitChain{ChainId: ctx.ChainID(), AppStateBytes: raw})
	require.NoError(t, err)
	for store, key := range map[string]byte{"zerone_gov": 0x18, "knowledge": 0x90, "capture_defense": 0x08, "qualification": 0x40, "capture_challenge": 0x05, "vesting_rewards": 0x0c, "zerone_ontology": 0x06, "emergency": 0x14, "substrate_bridge": 0x90, "zerone_ibcrl": 0x04} {
		ctx.KVStore(a.GetStoreKeyForTests(store)).Delete([]byte{key})
	}
	if badMarker {
		ctx.KVStore(a.GetStoreKeyForTests("qualification")).Set([]byte{0x40}, []byte{})
	}
	registry, err := json.Marshal(&customstaking.Validator{OperatorAddress: address.String(), TotalStake: "0", SelfDelegation: "0", DelegatedStake: "0"})
	require.NoError(t, err)
	ctx.KVStore(a.GetStoreKeyForTests("zerone_staking")).Set(append([]byte{1}, []byte(address.String())...), registry)
	// Synthetic non-discontinuous source; not a dev-1 mint-ledger measurement.
	ctx.KVStore(a.GetStoreKeyForTests("vesting_rewards")).Set([]byte{0x0b}, []byte("1"))
	vm := a.CurrentModuleVersionMap()
	for name, version := range map[string]uint64{"knowledge": 12, "zerone_gov": 3, "capture_defense": 1, "capture_challenge": 1, "qualification": 1, "vesting_rewards": 3, "alignment": 1, "zerone_ontology": 1, "emergency": 2, "substrate_bridge": 1, "ibcratelimit": 1} {
		vm[name] = version
	}
	require.NoError(t, a.UpgradeKeeper.SetModuleVersionMap(ctx, vm))
	// Detailed held records belong in the offline JSON, not the permanent receipt.
	bounty, err := proto.Marshal(&knowledge.AugmentationBounty{Id: "offline\x00bounty"})
	require.NoError(t, err)
	ctx.KVStore(a.GetStoreKeyForTests("knowledge")).Set(knowledge.AugmentationBountyKey("offline\x00bounty"), bounty)
	rollback, err := json.Marshal(&gov.PhaseTransitionProposal{LipID: "offline-rollback", TargetPhase: gov.ResearchFundPhase_RESEARCH_FUND_PHASE_GENESIS_PAIR, IsRollback: true, Stage: gov.PhaseTransitionStagePending})
	require.NoError(t, err)
	ctx.KVStore(a.GetStoreKeyForTests("zerone_gov")).Set([]byte("\x14offline-rollback"), rollback)
	lip, err := json.Marshal(&gov.LIP{Id: "offline-rollback", Category: gov.CategoryPhaseRollback})
	require.NoError(t, err)
	ctx.KVStore(a.GetStoreKeyForTests("zerone_gov")).Set(gov.LIPKey("offline-rollback"), lip)
	commit := a.CommitMultiStore().Commit()
	require.NoError(t, db.Close())
	return home, commit.Version, hex.EncodeToString(commit.Hash)
}
func fileDigests(t *testing.T, home string) map[string]string {
	t.Helper()
	result := map[string]string{}
	require.NoError(t, filepath.WalkDir(home, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		hash := sha256.Sum256(data)
		result[path] = hex.EncodeToString(hash[:])
		return nil
	}))
	return result
}
func TestReadOnlyCopiedDBCensus(t *testing.T) {
	for _, fault := range []bool{false, true} {
		t.Run(strconv.FormatBool(fault), func(t *testing.T) {
			home, height, hash := sourceCopy(t, fault)
			before := fileDigests(t, home)
			args := []string{"--copied-db", "--home", home, "--chain-id", "zerone-rehearsal-census", "--expected-height", strconv.FormatInt(height, 10), "--expected-app-hash", hash}
			var out, diagnostics bytes.Buffer
			code := run(args, &out, &diagnostics)
			want := 0
			if fault {
				want = 2
			}
			require.Equal(t, want, code, diagnostics.String()+out.String())
			var report map[string]interface{}
			require.NoError(t, json.Unmarshal(out.Bytes(), &report))
			require.Equal(t, !fault, report["passed"])
			var details struct {
				Census app.ConsensusRepairsCensus `json:"census"`
			}
			require.NoError(t, json.Unmarshal(out.Bytes(), &details))
			require.Len(t, details.Census.PendingRollbacks, 1)
			require.Equal(t, "offline-rollback", details.Census.PendingRollbacks[0].LIPID)
			require.Equal(t, 1, bytes.Count(out.Bytes(), []byte(`"lip_id": "offline-rollback"`)))
			require.NotContains(t, out.String(), "pending legacy rollback")
			require.Contains(t, out.String(), hex.EncodeToString(knowledge.AugmentationBountyKey("offline\x00bounty")))
			require.Contains(t, out.String(), "NUL bounty ID")
			if fault {
				require.Contains(t, out.String(), "present-empty")
			}
			require.Equal(t, before, fileDigests(t, home), "read-only tool leaves every copied source file byte-identical")
			args[len(args)-1] = strings.Repeat("a", 64)
			out.Reset()
			diagnostics.Reset()
			require.Equal(t, 1, run(args, &out, &diagnostics))
			require.Empty(t, out.String())
			require.Contains(t, diagnostics.String(), "height/AppHash")
		})
	}
}
func TestCommandRefusesMissingAttestationAndProtectedPaths(t *testing.T) {
	var out, diagnostics bytes.Buffer
	require.Equal(t, 1, run(nil, &out, &diagnostics))
	require.Empty(t, out.String())
	for _, path := range []string{"relative", "/", filepath.Join(t.TempDir(), ".zeroned"), "/Users/Shared/zerone-private"} {
		_, err := copiedDataDirectory(path)
		require.Error(t, err)
	}
	home, err := filepath.EvalSymlinks(t.TempDir())
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(filepath.Join(home, "data", "application.db"), 0700))
	require.NoError(t, os.WriteFile(filepath.Join(home, "data", "application.db", "LOCK"), nil, 0600))
	require.NoError(t, os.Symlink(home, filepath.Join(home, "alias")))
	_, err = copiedDataDirectory(filepath.Join(home, "alias"))
	require.ErrorContains(t, err, "non-symlink")
}

func TestReadOnlyCopiedDBCensusWithSymlinkTMPDIR(t *testing.T) {
	root := t.TempDir()
	alias := filepath.Join(root, "tmp-alias")
	require.NoError(t, os.Symlink(root, alias))
	t.Setenv("TMPDIR", alias)
	home, height, hash := sourceCopy(t, false)
	resolved, err := filepath.EvalSymlinks(home)
	require.NoError(t, err)
	require.Equal(t, resolved, home)
	var out, diagnostics bytes.Buffer
	require.Zero(t, run([]string{"--copied-db", "--home", home, "--chain-id", "zerone-rehearsal-census", "--expected-height", strconv.FormatInt(height, 10), "--expected-app-hash", hash}, &out, &diagnostics), diagnostics.String())
}

func TestCommandRefusesLoadedWorkingHashMismatch(t *testing.T) {
	// The committed metadata includes a store not mounted by this binary.
	// LastCommitID still matches; WorkingHash must independently refuse it.
	home, height, hash := sourceCopy(t, false, true)
	before := fileDigests(t, home)
	var out, diagnostics bytes.Buffer
	code := run([]string{"--copied-db", "--home", home, "--chain-id", "zerone-rehearsal-census", "--expected-height", strconv.FormatInt(height, 10), "--expected-app-hash", hash}, &out, &diagnostics)
	require.Equal(t, 1, code, diagnostics.String()+out.String())
	require.Empty(t, out.String())
	require.Contains(t, diagnostics.String(), "loaded working AppHash")
	require.Equal(t, before, fileDigests(t, home))
}

func TestCommandRefusesMissingOrSymlinkLOCKWithoutCreatingFiles(t *testing.T) {
	for _, symlink := range []bool{false, true} {
		t.Run(strconv.FormatBool(symlink), func(t *testing.T) {
			home, height, hash := sourceCopy(t, false)
			lock := filepath.Join(home, "data", "application.db", "LOCK")
			require.NoError(t, os.Remove(lock))
			if symlink {
				target := filepath.Join(home, "lock-target")
				require.NoError(t, os.WriteFile(target, nil, 0600))
				require.NoError(t, os.Symlink(target, lock))
			}
			before := fileDigests(t, home)
			var out, diagnostics bytes.Buffer
			code := run([]string{"--copied-db", "--home", home, "--chain-id", "zerone-rehearsal-census", "--expected-height", strconv.FormatInt(height, 10), "--expected-app-hash", hash}, &out, &diagnostics)
			require.Equal(t, 1, code)
			require.Empty(t, out.String())
			require.Contains(t, diagnostics.String(), "LOCK")
			require.Equal(t, before, fileDigests(t, home))
		})
	}
}
