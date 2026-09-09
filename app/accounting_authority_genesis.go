package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"

	upgradetypes "cosmossdk.io/x/upgrade/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	accountingAuthorityGenesisKey     = "accounting_authority"
	accountingAuthorityImportedMarker = "chain_lineage_imported_accounting-authority-v1"
)

// An imported receipt is historical provenance supplied by genesis. It never
// creates an x/upgrade done record or asserts that the new chain executed H.
type accountingAuthorityGenesis struct {
	Schema  string                      `json:"schema"`
	Origin  string                      `json:"origin"`
	Receipt *accountingAuthorityReceipt `json:"receipt,omitempty"`
}

func validateAccountingReceipt(receipt accountingAuthorityReceipt) error {
	if receipt.Schema != "zerone.accounting-authority/applied-v1" || receipt.Height <= 0 {
		return fmt.Errorf("invalid accounting activation receipt identity")
	}
	if _, err := parseAccountingAuthorityPlanInfo(receipt.Info); err != nil {
		return err
	}
	if receipt.PlanSHA256 != accountingPlanDigest(upgradetypes.Plan{Name: UpgradeNameAccountingAuthorityV1, Height: receipt.Height, Info: receipt.Info}) {
		return fmt.Errorf("accounting activation receipt plan digest mismatch")
	}
	return nil
}

func parseAccountingReceipt(raw string) (*accountingAuthorityReceipt, error) {
	if len(raw) == 0 || len(raw) > 4096 {
		return nil, fmt.Errorf("accounting receipt size boundary")
	}
	var receipt accountingAuthorityReceipt
	decoder := json.NewDecoder(bytes.NewBufferString(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&receipt); err != nil {
		return nil, err
	}
	encoded, err := json.Marshal(receipt)
	if err != nil || string(encoded) != raw {
		return nil, fmt.Errorf("accounting receipt must be canonical JSON")
	}
	if err := validateAccountingReceipt(receipt); err != nil {
		return nil, err
	}
	return &receipt, nil
}

func parseAccountingAuthorityGenesis(raw []byte) (accountingAuthorityGenesis, error) {
	lineage := accountingAuthorityGenesis{Schema: "zerone.accounting-authority/genesis-v1", Origin: "native"}
	if len(raw) == 0 {
		return lineage, nil
	}
	lineage = accountingAuthorityGenesis{}
	if len(raw) > 8192 {
		return lineage, fmt.Errorf("accounting genesis metadata exceeds size boundary")
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	if err := uniqueAccountingGenesisJSON(decoder, 0); err != nil {
		return lineage, err
	}
	if _, err := decoder.Token(); err != io.EOF {
		return lineage, fmt.Errorf("accounting genesis metadata has trailing JSON")
	}
	decoder = json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&lineage); err != nil {
		return lineage, err
	}
	if lineage.Schema != "zerone.accounting-authority/genesis-v1" {
		return lineage, fmt.Errorf("invalid accounting genesis metadata schema")
	}
	switch lineage.Origin {
	case "native", "legacy":
		if lineage.Receipt != nil {
			return lineage, fmt.Errorf("%s accounting genesis cannot supply a migration receipt", lineage.Origin)
		}
	case "migrated", "imported-migrated":
		if lineage.Receipt == nil {
			return lineage, fmt.Errorf("migrated accounting genesis requires its exact historical receipt")
		}
		if err := validateAccountingReceipt(*lineage.Receipt); err != nil {
			return lineage, err
		}
	default:
		return lineage, fmt.Errorf("unknown accounting genesis origin")
	}
	return lineage, nil
}

func uniqueAccountingGenesisJSON(decoder *json.Decoder, depth int) error {
	if depth > 8 {
		return fmt.Errorf("accounting genesis metadata nesting boundary")
	}
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	delimiter, isContainer := token.(json.Delim)
	if !isContainer {
		if depth == 0 {
			return fmt.Errorf("accounting genesis metadata must be an object")
		}
		return nil
	}
	if delimiter != '{' {
		return fmt.Errorf("accounting genesis metadata permits objects only")
	}
	seen := map[string]bool{}
	for decoder.More() {
		key, err := decoder.Token()
		if err != nil {
			return err
		}
		name, ok := key.(string)
		if !ok || seen[name] {
			return fmt.Errorf("duplicate accounting genesis metadata field")
		}
		seen[name] = true
		allowed := map[string]bool{"schema": true, "origin": true, "receipt": true}
		if depth == 1 {
			allowed = map[string]bool{"schema": true, "height": true, "info": true, "plan_sha256": true}
		}
		if depth > 1 || !allowed[name] {
			return fmt.Errorf("unknown accounting genesis metadata field %q", name)
		}
		if len(seen) > 16 {
			return fmt.Errorf("accounting genesis metadata field ceiling")
		}
		if err := uniqueAccountingGenesisJSON(decoder, depth+1); err != nil {
			return err
		}
	}
	_, err = decoder.Token()
	return err
}

func (app *ZeroneApp) accountingAuthorityGenesisMetadata(ctx sdk.Context, latest int64) (accountingAuthorityGenesis, error) {
	lineage := accountingAuthorityGenesis{Schema: "zerone.accounting-authority/genesis-v1"}
	native, hasNative, err := app.KnowledgeKeeper.ReadMigrationMarkerPresenceChecked(ctx, accountingAuthorityNativeMarker)
	if err != nil {
		return lineage, err
	}
	migrated, hasMigrated, err := app.KnowledgeKeeper.ReadMigrationMarkerPresenceChecked(ctx, accountingAuthorityMigrationMarker)
	if err != nil {
		return lineage, err
	}
	imported, hasImported, err := app.KnowledgeKeeper.ReadMigrationMarkerPresenceChecked(ctx, accountingAuthorityImportedMarker)
	if err != nil {
		return lineage, err
	}
	done, err := app.UpgradeKeeper.GetDoneHeight(ctx, UpgradeNameAccountingAuthorityV1)
	if err != nil {
		return lineage, err
	}
	count := 0
	for _, present := range []bool{hasNative, hasMigrated, hasImported} {
		if present {
			count++
		}
	}
	if count == 0 && done == 0 && !app.ZeroneStakingKeeper.AccountingSafetyEnabled(ctx) && !app.ZeroneGovKeeper.AccountingSafetyEnabled(ctx) {
		lineage.Origin = "legacy"
		return lineage, nil
	}
	if count != 1 {
		return lineage, fmt.Errorf("accounting state must have exactly one native, applied, or imported lineage")
	}
	if !app.ZeroneStakingKeeper.AccountingSafetyEnabled(ctx) || !app.ZeroneGovKeeper.AccountingSafetyEnabled(ctx) {
		return lineage, fmt.Errorf("accounting lineage requires both module safety markers")
	}
	switch {
	case hasNative:
		if native != "genesis" || done != 0 {
			return lineage, fmt.Errorf("invalid native accounting lineage")
		}
		lineage.Origin = "native"
	case hasMigrated:
		receipt, err := parseAccountingReceipt(migrated)
		if err != nil {
			return lineage, err
		}
		if done != receipt.Height || done > latest {
			return lineage, fmt.Errorf("accounting receipt and committed upgrade done height disagree")
		}
		info, _ := parseAccountingAuthorityPlanInfo(receipt.Info)
		if ctx.ChainID() != "" && ctx.ChainID() != info.ChainID {
			return lineage, fmt.Errorf("accounting applied receipt belongs to another chain")
		}
		lineage.Origin, lineage.Receipt = "migrated", receipt
	case hasImported:
		if done != 0 {
			return lineage, fmt.Errorf("imported accounting receipt cannot fabricate an applied upgrade")
		}
		receipt, err := parseAccountingReceipt(imported)
		if err != nil {
			return lineage, err
		}
		lineage.Origin, lineage.Receipt = "imported-migrated", receipt
	}
	return lineage, nil
}
