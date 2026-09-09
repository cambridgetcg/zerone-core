package app

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"sort"
	"strings"

	"cosmossdk.io/collections"
	upgradetypes "cosmossdk.io/x/upgrade/types"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	"github.com/zerone-chain/zerone/internal/accountingmigration"
	zeronegovtypes "github.com/zerone-chain/zerone/x/gov/types"
	zeronestakingtypes "github.com/zerone-chain/zerone/x/staking/types"
)

// UpgradeNameAccountingAuthorityV1 owns custom staking 1->2 and custom governance
// 2->3. Registering its handler does not attest or authorize a production release.
const UpgradeNameAccountingAuthorityV1 = accountingmigration.UpgradeName

const accountingAuthorityNativeMarker = "chain_lineage_native_accounting-authority-v1"

const (
	accountingAuthorityMigrationMarker = "upgrade_marker_accounting-authority-v1"
	AccountingAuthorityPlanInfoSchema  = "zerone.accounting-authority/plan-v1"
	maxAccountingPlanInfoBytes         = 1024
	maxAccountingCommitmentRecords     = 50_000
	maxAccountingCommitmentBytes       = 32 << 20
)

// AccountingAuthorityPlanInfo commits stable selected state, not a guessed
// future H-1 AppHash. Any change to either custom store or custody rejects H.
type AccountingAuthorityPlanInfo struct {
	Schema                 string `json:"schema"`
	ChainID                string `json:"chain_id"`
	SourceVersionMapSHA256 string `json:"source_version_map_sha256"`
	StakingStateSHA256     string `json:"staking_state_sha256"`
	GovernanceStateSHA256  string `json:"governance_state_sha256"`
}

type accountingAuthorityReceipt struct {
	Schema     string `json:"schema"`
	Height     int64  `json:"height"`
	Info       string `json:"info"`
	PlanSHA256 string `json:"plan_sha256"`
}

func accountingAuthoritySourceVersionMap() module.VersionMap {
	// Frozen H3 output, including the two zero-version light-client modules.
	// This derives only from the immutable H2 list and explicit H3 changes;
	// it never follows this binary's moving module targets.
	vm := make(module.VersionMap, len(sdk053IBC10ExactSourceVersionMap))
	for _, entry := range sdk053IBC10ExactSourceVersionMap {
		vm[entry.name] = entry.version
	}
	delete(vm, legacyCapabilityStoreKey)
	delete(vm, legacyIBCFeeStoreKey)
	vm["06-solomachine"], vm["07-tendermint"] = 0, 0
	vm["emergency"], vm["ibc"], vm["transfer"] = 2, 8, 6
	return vm
}

func accountingAuthorityTargetVersionMap() module.VersionMap {
	vm := accountingAuthoritySourceVersionMap()
	vm[zeronestakingtypes.ModuleName], vm[zeronegovtypes.ModuleName] = 2, 3
	return vm
}

func requireAccountingExactVersionMap(actual, expected module.VersionMap) error {
	if !reflect.DeepEqual(actual, expected) {
		return fmt.Errorf("accounting authority requires exact full module version map")
	}
	return nil
}

func parseAccountingAuthorityPlanInfo(info string) (AccountingAuthorityPlanInfo, error) {
	var payload AccountingAuthorityPlanInfo
	if len(info) == 0 || len(info) > maxAccountingPlanInfoBytes {
		return payload, fmt.Errorf("accounting plan info exceeds size boundary")
	}
	decoder := json.NewDecoder(strings.NewReader(info))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil {
		return payload, fmt.Errorf("decode accounting plan info: %w", err)
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return payload, fmt.Errorf("accounting plan info has trailing JSON")
	}
	canonical, err := json.Marshal(payload)
	if err != nil || string(canonical) != info {
		return payload, fmt.Errorf("accounting plan info must be exact canonical JSON")
	}
	if payload.Schema != AccountingAuthorityPlanInfoSchema || payload.ChainID == "" || len(payload.ChainID) > 128 || strings.TrimSpace(payload.ChainID) != payload.ChainID {
		return payload, fmt.Errorf("invalid accounting plan schema or chain identity")
	}
	for _, value := range []string{payload.SourceVersionMapSHA256, payload.StakingStateSHA256, payload.GovernanceStateSHA256} {
		decoded, err := hex.DecodeString(value)
		if err != nil || len(decoded) != sha256.Size || value != strings.ToLower(value) {
			return payload, fmt.Errorf("accounting plan hashes must be 64 lowercase hexadecimal characters")
		}
	}
	expected, err := canonicalSDK053IBC10SourceVersionMapSHA256(accountingAuthoritySourceVersionMap())
	if err != nil {
		return payload, err
	}
	if payload.SourceVersionMapSHA256 != expected {
		return payload, fmt.Errorf("accounting plan source version-map digest mismatch")
	}
	return payload, nil
}

func accountingPlanDigest(plan upgradetypes.Plan) string {
	encoded, _ := json.Marshal(struct {
		Name   string `json:"name"`
		Height int64  `json:"height"`
		Info   string `json:"info"`
	}{plan.Name, plan.Height, plan.Info})
	digest := sha256.Sum256(encoded)
	return hex.EncodeToString(digest[:])
}

// BuildAccountingAuthorityPlanInfo reads the complete committed IAVL stores and
// selected native bank custody. Use an independently authenticated, stopped
// database copy and supply its chain ID through ctx. The result is not authority
// to freeze state, schedule an upgrade, or repair a discrepant claimant ledger.
func (app *ZeroneApp) BuildAccountingAuthorityPlanInfo(ctx sdk.Context) (result string, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			result = ""
			err = fmt.Errorf("invalid accounting plan source: %v", recovered)
		}
	}()
	vm, err := app.UpgradeKeeper.GetModuleVersionMap(ctx)
	if err != nil {
		return "", err
	}
	if err := requireAccountingExactVersionMap(vm, accountingAuthoritySourceVersionMap()); err != nil {
		return "", err
	}
	if err := app.validateSDK053IBC10CompletedOrNativeLineage(); err != nil {
		return "", fmt.Errorf("accounting predecessor is not verified H3: %w", err)
	}
	if ctx.ChainID() == "" {
		return "", fmt.Errorf("accounting plan preparation requires an independently established chain ID")
	}
	if err := app.requireAccountingSourceUnsealed(ctx); err != nil {
		return "", err
	}
	if err := app.ZeroneStakingKeeper.ValidateAccountingSafety(ctx); err != nil {
		return "", err
	}
	if err := app.ZeroneGovKeeper.ValidateAccountingMigration(ctx); err != nil {
		return "", err
	}
	if err := app.requireAccountingModuleAccountPermissions(ctx); err != nil {
		return "", err
	}
	rootIDs, err := app.activationSubstoreCommitIDs([]string{banktypes.StoreKey, zeronestakingtypes.StoreKey, zeronegovtypes.StoreKey, authtypes.StoreKey, upgradetypes.StoreKey, "knowledge"})
	if err != nil {
		return "", err
	}
	// Source identities are evidence too: pending cache edits must not repair a
	// wrong committed version map, remove a done marker, or alter permissions.
	moduleAccountKeys := make(map[string]bool, len(maccPerms))
	for name := range maccPerms {
		key := append(bytes.Clone(authtypes.AddressStoreKeyPrefix.Bytes()), authtypes.NewModuleAddress(name)...)
		moduleAccountKeys[string(key)] = true
	}
	for _, source := range []struct {
		name      string
		selectKey func([]byte) (bool, error)
	}{
		{authtypes.StoreKey, func(key []byte) (bool, error) { return moduleAccountKeys[string(key)], nil }},
		{upgradetypes.StoreKey, func(key []byte) (bool, error) {
			return len(key) > 0 && (key[0] == upgradetypes.VersionMapByte || key[0] == upgradetypes.DoneByte), nil
		}},
		{"knowledge", func(key []byte) (bool, error) { return bytes.HasPrefix(key, []byte{0x7f, 0x01}), nil }},
	} {
		count, totalBytes := 0, 0
		records, err := app.collectVerifiedCommittedRecords(source.name, rootIDs[source.name], func(key, value []byte) (bool, error) {
			selected, err := source.selectKey(key)
			if err != nil || !selected {
				return selected, err
			}
			count++
			totalBytes += len(key) + len(value)
			if count > maxAccountingCommitmentRecords || totalBytes > maxAccountingCommitmentBytes || len(key) > 1024 || len(value) > 256*1024 {
				return false, fmt.Errorf("accounting source identity resource ceiling exceeded")
			}
			return true, nil
		})
		if err != nil {
			return "", err
		}
		if err := app.verifyAccountingExecutionRows(ctx, source.name, records, source.selectKey); err != nil {
			return "", err
		}
	}
	bankKeyCodec := collections.PairKeyCodec(sdk.AccAddressKey, collections.StringKey)
	addresses := map[string]string{
		string(authtypes.NewModuleAddress(zeronestakingtypes.ModuleName)): zeronestakingtypes.ModuleName,
		string(authtypes.NewModuleAddress(zeronegovtypes.ModuleName)):     zeronegovtypes.ModuleName,
	}
	balances := map[string][]upgradePrestateRecord{}
	bankSelectedBytes := 0
	_, err = app.collectVerifiedCommittedRecords(banktypes.StoreKey, rootIDs[banktypes.StoreKey], func(key, value []byte) (bool, error) {
		prefix := banktypes.BalancesPrefix.Bytes()
		if !bytes.HasPrefix(key, prefix) {
			return false, nil
		}
		read, pair, err := bankKeyCodec.Decode(key[len(prefix):])
		if err != nil || read != len(key)-len(prefix) {
			return false, fmt.Errorf("invalid committed bank balance key")
		}
		if name, selected := addresses[string(pair.K1())]; selected {
			if err := sdk.ValidateDenom(pair.K2()); err != nil {
				return false, err
			}
			if len(balances[name]) >= 10_000 {
				return false, fmt.Errorf("accounting custody denomination ceiling exceeded")
			}
			bankSelectedBytes += len(key) + len(value)
			if bankSelectedBytes > maxAccountingCommitmentBytes || len(key) > 1024 || len(value) > 256*1024 {
				return false, fmt.Errorf("accounting custody byte ceiling exceeded")
			}
			balances[name] = append(balances[name], upgradePrestateRecord{Key: bytes.Clone(key), Value: bytes.Clone(value)})
		}
		return false, nil
	})
	if err != nil {
		return "", err
	}
	digests := map[string]string{}
	for _, name := range []string{zeronestakingtypes.ModuleName, zeronegovtypes.ModuleName} {
		count, byteCount := 0, 0
		records, err := app.collectVerifiedCommittedRecords(name, rootIDs[name], func(key, value []byte) (bool, error) {
			count++
			byteCount += len(key) + len(value)
			if count > maxAccountingCommitmentRecords || byteCount > maxAccountingCommitmentBytes || len(key) > 1024 || len(value) > 256*1024 {
				return false, fmt.Errorf("accounting commitment resource ceiling exceeded for %s", name)
			}
			return true, nil
		})
		if err != nil {
			return "", err
		}
		if err := app.verifyAccountingExecutionRows(ctx, name, records, nil); err != nil {
			return "", err
		}
		if err := app.verifyAccountingExecutionRows(ctx, banktypes.StoreKey, balances[name], func(key []byte) (bool, error) {
			prefix := banktypes.BalancesPrefix.Bytes()
			if !bytes.HasPrefix(key, prefix) {
				return false, nil
			}
			read, pair, err := bankKeyCodec.Decode(key[len(prefix):])
			if err != nil || read != len(key)-len(prefix) {
				return false, fmt.Errorf("invalid execution bank key")
			}
			return addresses[string(pair.K1())] == name, nil
		}); err != nil {
			return "", err
		}
		hasher := sha256.New()
		hasher.Write([]byte("zerone/accounting-authority/state-v1\x00" + name + "\x00"))
		for _, part := range []struct {
			name    string
			records []upgradePrestateRecord
		}{{"custom-store", records}, {"bank-custody", balances[name]}} {
			hasher.Write([]byte(part.name + "\x00"))
			sort.Slice(part.records, func(i, j int) bool { return bytes.Compare(part.records[i].Key, part.records[j].Key) < 0 })
			var length [8]byte
			binary.BigEndian.PutUint64(length[:], uint64(len(part.records)))
			hasher.Write(length[:])
			for _, record := range part.records {
				// The committed raw view must agree with the execution context;
				// pre-block callers cannot substitute pending writes for evidence.
				storeName := name
				if part.name == "bank-custody" {
					storeName = banktypes.StoreKey
				}
				if !bytes.Equal(ctx.KVStore(app.keys[storeName]).Get(record.Key), record.Value) {
					return "", fmt.Errorf("accounting execution context differs from committed %s", storeName)
				}
				for _, field := range [][]byte{record.Key, record.Value} {
					binary.BigEndian.PutUint64(length[:], uint64(len(field)))
					hasher.Write(length[:])
					hasher.Write(field)
				}
			}
		}
		digests[name] = hex.EncodeToString(hasher.Sum(nil))
	}
	vmDigest, err := canonicalSDK053IBC10SourceVersionMapSHA256(vm)
	if err != nil {
		return "", err
	}
	encoded, err := json.Marshal(AccountingAuthorityPlanInfo{Schema: AccountingAuthorityPlanInfoSchema, ChainID: ctx.ChainID(), SourceVersionMapSHA256: vmDigest, StakingStateSHA256: digests[zeronestakingtypes.ModuleName], GovernanceStateSHA256: digests[zeronegovtypes.ModuleName]})
	if err != nil {
		return "", err
	}
	if _, err := parseAccountingAuthorityPlanInfo(string(encoded)); err != nil {
		return "", err
	}
	return string(encoded), nil
}

// A complete committed-root scan authenticates expected rows. This second scan
// also rejects extra pending keys or new bank denominations in the block cache,
// rather than checking only keys that existed in committed state.
func (app *ZeroneApp) verifyAccountingExecutionRows(ctx sdk.Context, name string, expected []upgradePrestateRecord, selectKey func([]byte) (bool, error)) (err error) {
	iterator := ctx.KVStore(app.keys[name]).Iterator(nil, nil)
	defer func() {
		if closeErr := iterator.Close(); err == nil && closeErr != nil {
			err = closeErr
		}
	}()
	index, count, totalBytes := 0, 0, 0
	for ; iterator.Valid(); iterator.Next() {
		count++
		totalBytes += len(iterator.Key()) + len(iterator.Value())
		if count > maxSelectedPrestateRecordCount || totalBytes > maxSelectedPrestateRecordBytes {
			return fmt.Errorf("accounting execution scan resource ceiling exceeded")
		}
		selected := true
		if selectKey != nil {
			var selectErr error
			selected, selectErr = selectKey(iterator.Key())
			if selectErr != nil {
				return selectErr
			}
		}
		if !selected {
			continue
		}
		if index >= len(expected) || !bytes.Equal(iterator.Key(), expected[index].Key) || !bytes.Equal(iterator.Value(), expected[index].Value) {
			return fmt.Errorf("accounting execution context differs from complete committed %s", name)
		}
		index++
	}
	// SDK cache iterators report these sentinels after ordinary exhaustion.
	// Do not mask any other backend failure.
	if err := iterator.Error(); err != nil && (iterator.Valid() || (err.Error() != "invalid cacheMergeIterator" && err.Error() != "invalid memIterator")) {
		return err
	}
	if index != len(expected) {
		return fmt.Errorf("accounting execution context omits committed %s records", name)
	}
	return nil
}

func (app *ZeroneApp) validateAccountingAuthoritySource(ctx sdk.Context, plan upgradetypes.Plan, fromVM module.VersionMap) error {
	if plan.Name != UpgradeNameAccountingAuthorityV1 || plan.Height <= 0 {
		return fmt.Errorf("invalid accounting upgrade identity")
	}
	if err := requireAccountingExactVersionMap(fromVM, accountingAuthoritySourceVersionMap()); err != nil {
		return err
	}
	if err := requireAccountingExactVersionMap(app.ModuleManager.GetVersionMap(), accountingAuthorityTargetVersionMap()); err != nil {
		return fmt.Errorf("accounting target: %w", err)
	}
	if err := requireAccountingTransitionOwner(plan.Name, fromVM, app.ModuleManager.GetVersionMap()); err != nil {
		return err
	}
	info, err := parseAccountingAuthorityPlanInfo(plan.Info)
	if err != nil {
		return err
	}
	if ctx.ChainID() != info.ChainID {
		return fmt.Errorf("accounting upgrade chain identity mismatch")
	}
	if err := app.requireAccountingSourceUnsealed(ctx); err != nil {
		return err
	}
	expectedInfo, err := app.BuildAccountingAuthorityPlanInfo(ctx)
	if err != nil {
		return err
	}
	if expectedInfo != plan.Info {
		return fmt.Errorf("accounting state or custody drifted from the exact plan commitment")
	}
	return nil
}

func (app *ZeroneApp) requireAccountingSourceUnsealed(ctx sdk.Context) error {
	if app.ZeroneStakingKeeper.AccountingSafetyEnabled(ctx) || app.ZeroneGovKeeper.AccountingSafetyEnabled(ctx) {
		return fmt.Errorf("accounting source requires both safety markers absent")
	}
	for _, marker := range []string{accountingAuthorityNativeMarker, accountingAuthorityMigrationMarker, accountingAuthorityImportedMarker} {
		_, present, err := app.KnowledgeKeeper.ReadMigrationMarkerPresenceChecked(ctx, marker)
		if err != nil {
			return err
		}
		if present {
			return fmt.Errorf("accounting source already has lineage marker %q", marker)
		}
	}
	done, err := app.UpgradeKeeper.GetDoneHeight(ctx, UpgradeNameAccountingAuthorityV1)
	if err != nil {
		return err
	}
	if done != 0 {
		return fmt.Errorf("accounting source already has done height")
	}
	return nil
}

func (app *ZeroneApp) requireAccountingModuleAccountPermissions(ctx sdk.Context) error {
	names := make([]string, 0, len(maccPerms))
	for name := range maccPerms {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		existing := app.AccountKeeper.GetAccount(ctx, authtypes.NewModuleAddress(name))
		if existing == nil {
			continue
		}
		account, ok := existing.(sdk.ModuleAccountI)
		if !ok || account.GetName() != name || !equalStringSets(account.GetPermissions(), maccPerms[name]) {
			return fmt.Errorf("accounting migration refuses unrelated module-account permission drift for %q", name)
		}
	}
	return nil
}

func (app *ZeroneApp) registerAccountingAuthorityUpgrade() {
	app.UpgradeKeeper.SetUpgradeHandler(UpgradeNameAccountingAuthorityV1, func(goCtx context.Context, plan upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
		ctx := sdk.UnwrapSDKContext(goCtx)
		if ctx.BlockHeight() != plan.Height || ctx.HeaderInfo().Height != plan.Height || app.CommitMultiStore().LastCommitID().Version != plan.Height-1 {
			return nil, fmt.Errorf("accounting migration requires exact committed H-1/H boundary")
		}
		if app.UpgradeKeeper.IsSkipHeight(plan.Height) {
			return nil, fmt.Errorf("accounting migration cannot be unsafe-skipped")
		}
		if err := app.validateAccountingAuthoritySource(ctx, plan, fromVM); err != nil {
			return nil, err
		}
		cache, write := ctx.CacheContext()
		authorized, err := accountingmigration.WithOwner(cache, plan.Name)
		if err != nil {
			return nil, err
		}
		toVM, err := app.ModuleManager.RunMigrations(authorized, app.configurator, fromVM)
		if err != nil {
			return nil, err
		}
		app.ReconcileModuleAccountPerms(authorized)
		if err := app.ZeroneStakingKeeper.ValidateAccountingSafety(authorized); err != nil {
			return nil, err
		}
		if err := app.ZeroneGovKeeper.ValidateAccountingSafety(authorized); err != nil {
			return nil, err
		}
		receipt, err := json.Marshal(accountingAuthorityReceipt{Schema: "zerone.accounting-authority/applied-v1", Height: plan.Height, Info: plan.Info, PlanSHA256: accountingPlanDigest(plan)})
		if err != nil {
			return nil, err
		}
		if err := app.KnowledgeKeeper.WriteMigrationMarker(authorized, accountingAuthorityMigrationMarker, string(receipt)); err != nil {
			return nil, err
		}
		write()
		return toVM, nil
	})
}

type accountingModuleVersions struct {
	staking uint64
	gov     uint64
}

var (
	accountingLegacyVersions = accountingModuleVersions{staking: 1, gov: 2}
	accountingTargetVersions = accountingModuleVersions{staking: 2, gov: 3}
)

func accountingVersions(vm module.VersionMap) (accountingModuleVersions, error) {
	staking, hasStaking := vm[zeronestakingtypes.ModuleName]
	gov, hasGov := vm[zeronegovtypes.ModuleName]
	pair := accountingModuleVersions{staking: staking, gov: gov}
	if !hasStaking || !hasGov || (pair != accountingLegacyVersions && pair != accountingTargetVersions) {
		return pair, fmt.Errorf("accounting authority requires one complete known version pair: zerone_staking=%d (present=%t), zerone_gov=%d (present=%t)", staking, hasStaking, gov, hasGov)
	}
	return pair, nil
}

// requireAccountingTransitionOwner guards both historical broad migrations and
// H3's direct migration call. A frozen H3 source map alone cannot pin its target.
func requireAccountingTransitionOwner(planName string, fromVM, targetVM module.VersionMap) error {
	from, err := accountingVersions(fromVM)
	if err != nil {
		return fmt.Errorf("upgrade %q source: %w", planName, err)
	}
	target, err := accountingVersions(targetVM)
	if err != nil {
		return fmt.Errorf("upgrade %q target: %w", planName, err)
	}
	if from == target {
		return nil
	}
	if from != accountingLegacyVersions || target != accountingTargetVersions || planName != UpgradeNameAccountingAuthorityV1 {
		return fmt.Errorf("upgrade %q cannot carry reserved accounting transition zerone_staking 1->2 and zerone_gov 2->3; sole owner is %q", planName, UpgradeNameAccountingAuthorityV1)
	}
	return nil
}

// validateAccountingGenesisSelection runs before InitGenesis changes any store.
// Missing flags identify legacy exports, which must not silently become a new
// accounting lineage. New defaults and exports carry both explicit flags.
func validateAccountingGenesisSelection(genesis map[string]json.RawMessage) error {
	lineage, err := parseAccountingAuthorityGenesis(genesis[accountingAuthorityGenesisKey])
	if err != nil {
		return err
	}
	if lineage.Origin == "legacy" {
		return fmt.Errorf("legacy accounting export requires the reviewed migration, not native InitGenesis")
	}
	for _, name := range []string{zeronestakingtypes.ModuleName, zeronegovtypes.ModuleName} {
		var selection struct {
			Enabled bool `json:"accounting_safety_enabled"`
		}
		if err := json.Unmarshal(genesis[name], &selection); err != nil {
			return fmt.Errorf("decode %s accounting genesis: %w", name, err)
		}
		if !selection.Enabled {
			return fmt.Errorf("native %s genesis requires explicit accounting_safety_enabled=true; legacy state requires separately authorized %q migration", name, UpgradeNameAccountingAuthorityV1)
		}
	}
	return nil
}

// initializeAccountingAuthority runs only after the entire application genesis,
// including bank backing and both custom modules, has initialized successfully.
func (app *ZeroneApp) initializeAccountingAuthority(ctx sdk.Context, genesis GenesisState) error {
	lineage, err := parseAccountingAuthorityGenesis(genesis[accountingAuthorityGenesisKey])
	if err != nil {
		return err
	}
	cacheCtx, write := ctx.CacheContext()
	if err := app.requireAccountingSourceUnsealed(cacheCtx); err != nil {
		return fmt.Errorf("accounting genesis lineage must begin unsealed: %w", err)
	}
	if err := app.ZeroneStakingKeeper.EnableAccountingSafety(cacheCtx); err != nil {
		return fmt.Errorf("native custom staking accounting: %w", err)
	}
	if err := app.ZeroneGovKeeper.EnableAccountingSafety(cacheCtx); err != nil {
		return fmt.Errorf("native custom governance accounting: %w", err)
	}
	marker, value := accountingAuthorityNativeMarker, "genesis"
	if lineage.Origin != "native" {
		marker = accountingAuthorityImportedMarker
		encoded, err := json.Marshal(lineage.Receipt)
		if err != nil {
			return err
		}
		value = string(encoded)
	}
	if err := app.KnowledgeKeeper.WriteMigrationMarker(cacheCtx, marker, value); err != nil {
		return fmt.Errorf("write native accounting lineage: %w", err)
	}
	write()
	return nil
}

// ValidateAccountingAuthorityStartup admits source state only at the exact
// scheduled H-1 boundary; target state requires native, applied, or explicitly
// imported accounting lineage with distinct meanings.
func (app *ZeroneApp) ValidateAccountingAuthorityStartup() (err error) {
	if app.activationPreflightReadOnly {
		return nil
	}
	latest := app.CommitMultiStore().LastCommitID().Version
	if latest == 0 {
		return nil // A fresh database has no module version map until InitChain.
	}
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("invalid accounting startup state: %v", recovered)
		}
	}()
	ctx := app.NewUncachedContext(true, cmtproto.Header{Height: latest, ChainID: app.ChainID()})
	vm, err := app.UpgradeKeeper.GetModuleVersionMap(ctx)
	if err != nil {
		return fmt.Errorf("read accounting startup versions: %w", err)
	}
	versions, err := accountingVersions(vm)
	if err != nil {
		return err
	}
	if versions == accountingLegacyVersions {
		localPlan, err := app.UpgradeKeeper.ReadUpgradeInfoFromDisk()
		if err != nil {
			return err
		}
		plan, err := app.UpgradeKeeper.GetUpgradePlan(ctx)
		if err != nil {
			return err
		}
		if plan.Name != UpgradeNameAccountingAuthorityV1 || latest == int64(^uint64(0)>>1) || plan.Height != latest+1 || localPlan.Name != plan.Name || localPlan.Height != plan.Height || localPlan.Info != plan.Info || app.UpgradeKeeper.IsSkipHeight(plan.Height) {
			return fmt.Errorf("legacy accounting startup requires the exact non-skipped on-chain and local H-1 accounting plan")
		}
		info, err := parseAccountingAuthorityPlanInfo(plan.Info)
		if err != nil {
			return err
		}
		// Some offline constructors have no BaseApp chain ID; the on-chain
		// plan then supplies context only. ABCI execution independently checks
		// the real network chain ID at H before writing any migration state.
		if ctx.ChainID() == "" {
			ctx = ctx.WithChainID(info.ChainID)
		}
		return app.validateAccountingAuthoritySource(ctx, plan, vm)
	}
	if err := requireAccountingExactVersionMap(vm, accountingAuthorityTargetVersionMap()); err != nil {
		return err
	}
	if !app.ZeroneStakingKeeper.AccountingSafetyEnabled(ctx) || !app.ZeroneGovKeeper.AccountingSafetyEnabled(ctx) {
		return fmt.Errorf("accounting target versions require both committed safety markers")
	}
	_, err = app.accountingAuthorityGenesisMetadata(ctx, latest)
	return err
}
