package main

import (
	"encoding/json"
	"errors"
	"math/big"
	"sort"
)

const (
	reportSchema      = "zerone.legacy-snapshot-accounting/v0"
	maxReportBytes    = 64 << 20
	moduleCustody     = "module_custody"
	nonModuleLocation = "non_module_location"
	unknownLocation   = "unknown_location"
)

type inputEvidence struct {
	Schema                   string `json:"schema"`
	Bytes                    int    `json:"bytes"`
	RawSHA256MatchesExpected bool   `json:"raw_sha256_matches_expected"`
}

type trustBoundary struct {
	ProvenanceAuthenticated bool   `json:"provenance_authenticated"`
	SignaturesVerified      bool   `json:"signatures_verified"`
	StateProofsVerified     bool   `json:"state_proofs_verified"`
	Limitation              string `json:"limitation"`
}

type custodyRow struct {
	Address        string   `json:"address"`
	AccountType    string   `json:"account_type"`
	ModuleName     string   `json:"module_name,omitempty"`
	Amount         string   `json:"amount_uzrn"`
	Classification string   `json:"classification"`
	Eligibility    string   `json:"eligibility"`
	Restrictions   string   `json:"restrictions"`
	Gaps           []string `json:"gaps"`
}

type custodyTotals struct {
	ModuleCustody     string `json:"module_custody_uzrn"`
	NonModuleLocation string `json:"non_module_location_uzrn"`
	UnknownLocation   string `json:"unknown_location_uzrn"`
	OwnerCount        int    `json:"owner_count"`
	ModuleCount       int    `json:"module_custody_count"`
	NonModuleCount    int    `json:"non_module_location_count"`
	UnknownCount      int    `json:"unknown_location_count"`
}

type validatorEvidence struct {
	Treatment string      `json:"treatment"`
	Tokens    string      `json:"tokens_uzrn"`
	Rows      []validator `json:"rows"`
}

type accountingReport struct {
	Schema              string            `json:"schema"`
	Result              string            `json:"result"`
	InputSHA256         string            `json:"input_sha256"`
	Input               inputEvidence     `json:"input"`
	Source              sourceCheckpoint  `json:"source"`
	Trust               trustBoundary     `json:"trust"`
	Denom               string            `json:"denom"`
	Supply              string            `json:"source_supply_uzrn"`
	Totals              custodyTotals     `json:"custody_totals"`
	Owners              []custodyRow      `json:"custody_rows"`
	Validators          validatorEvidence `json:"bonded_validator_metadata"`
	Eligibility         string            `json:"eligibility"`
	EntitlementTotal    *string           `json:"entitlement_total"`
	EntitlementStatus   string            `json:"entitlement_total_status"`
	ReserveRequirement  *string           `json:"reserve_requirement"`
	ReserveStatus       string            `json:"reserve_requirement_status"`
	EconomicEffect      string            `json:"economic_effect"`
	PayoutAuthorization bool              `json:"payout_authorization"`
	Gaps                []string          `json:"gaps"`
	ReportSHA256        string            `json:"report_sha256"`
}

func accountSnapshot(data []byte, expected string) ([]byte, error) {
	if !isSHA256(expected) {
		return nil, errors.New("expected SHA-256 must be 64 lowercase hexadecimal characters")
	}
	if len(data) == 0 || len(data) > maxInputBytes {
		return nil, errors.New("snapshot is empty or exceeds input byte limit")
	}
	actual := digest(data)
	if actual != expected {
		return nil, errors.New("snapshot raw SHA-256 does not match expected digest")
	}
	s, err := decodeSnapshot(data)
	if err != nil {
		return nil, err
	}
	if err := validateSnapshot(s); err != nil {
		return nil, err
	}
	return buildReport(s, actual, len(data))
}

func buildReport(s snapshot, inputSHA256 string, inputBytes int) ([]byte, error) {
	// Bound serialization before allocating expanded rows. JSON escapes at most
	// six bytes per input byte; fixed per-row allowances cover field names/gaps.
	estimate := 64 << 10
	for _, row := range s.Owners {
		estimate += 1024 + 6*(len(row.Address)+len(row.AccountType)+len(row.ModuleName)+len(row.Amount))
	}
	for _, row := range s.Validators {
		estimate += 1024 + 6*(len(row.OperatorAddress)+len(row.ConsensusPubKey.Type)+len(row.ConsensusPubKey.Key)+len(row.Tokens))
	}
	if estimate > maxReportBytes {
		return nil, errors.New("report exceeds conservative serialization byte limit")
	}

	sort.Slice(s.Owners, func(i, j int) bool { return s.Owners[i].Address < s.Owners[j].Address })
	sort.Slice(s.Validators, func(i, j int) bool { return s.Validators[i].OperatorAddress < s.Validators[j].OperatorAddress })
	r := accountingReport{
		Schema:            reportSchema,
		Result:            "RECONCILED_CUSTODY_ONLY",
		InputSHA256:       inputSHA256,
		Input:             inputEvidence{Schema: s.Schema, Bytes: inputBytes, RawSHA256MatchesExpected: true},
		Source:            s.Source,
		Trust:             trustBoundary{Limitation: "Raw digest equality binds only supplied bytes. Source labels and F/A/H assertions are untrusted evidence, not authenticated provenance. No signatures, REST state proofs, address control or beneficial ownership are verified."},
		Denom:             s.Denom,
		Supply:            s.Supply,
		Owners:            make([]custodyRow, 0, len(s.Owners)),
		Validators:        validatorEvidence{Treatment: "METADATA_ONLY_NOT_ADDITIONAL_PRINCIPAL", Rows: append([]validator{}, s.Validators...)},
		Eligibility:       "UNDETERMINED",
		EntitlementStatus: "UNKNOWN",
		ReserveStatus:     "UNKNOWN",
		EconomicEffect:    "NONE",
		Gaps: []string{
			"checkpoint_selection_and_provenance_not_authenticated",
			"account_labels_are_not_address_control_or_beneficial_ownership_proof",
			"staking_unbonding_vesting_liabilities_and_restrictions_not_reconciled",
			"module_lp_and_ibc_claimants_not_added_to_backing_balances",
			"contingent_or_unfunded_rewards_not_existing_supply",
			"post_anchor_A_state_census_not_F_state_eligibility_evidence",
			"other_assets_and_obligations_outside_native_uzrn_inventory",
			"eligibility_policy_opt_in_and_funded_reserves_not_established",
		},
	}
	if s.Source.DeclaredGenesisFileSHA256 == "" {
		r.Gaps = append(r.Gaps, "declared_raw_genesis_file_digest_not_supplied")
	}
	module, nonModule, unknown := new(big.Int), new(big.Int), new(big.Int)
	for _, entry := range s.Owners {
		row := classify(entry)
		amount, err := parseAmount(entry.Amount)
		if err != nil {
			return nil, err
		}
		switch row.Classification {
		case moduleCustody:
			module.Add(module, amount)
			r.Totals.ModuleCount++
		case nonModuleLocation:
			nonModule.Add(nonModule, amount)
			r.Totals.NonModuleCount++
		case unknownLocation:
			unknown.Add(unknown, amount)
			r.Totals.UnknownCount++
		}
		r.Owners = append(r.Owners, row)
	}
	total := new(big.Int).Add(new(big.Int).Add(module, nonModule), unknown)
	if total.String() != s.Supply {
		return nil, errors.New("custody partition does not reconcile to source supply")
	}
	r.Totals.ModuleCustody = module.String()
	r.Totals.NonModuleLocation = nonModule.String()
	r.Totals.UnknownLocation = unknown.String()
	r.Totals.OwnerCount = len(r.Owners)
	bonded := new(big.Int)
	for _, row := range s.Validators {
		amount, err := parseAmount(row.Tokens)
		if err != nil {
			return nil, err
		}
		bonded.Add(bonded, amount)
	}
	r.Validators.Tokens = bonded.String()

	// Match custom-staking-census/report.go:193-202: compact typed JSON, in
	// declared field order, with the hash VALUE blank (field remains present).
	// Neither the final digest value nor the stdout newline is hashed.
	unsealed, err := json.Marshal(r)
	if err != nil {
		return nil, err
	}
	r.ReportSHA256 = digest(unsealed)
	sealed, err := json.Marshal(r)
	if err != nil {
		return nil, err
	}
	if len(sealed) > maxReportBytes {
		return nil, errors.New("report exceeds output byte limit")
	}
	return sealed, nil
}

func classify(entry owner) custodyRow {
	row := custodyRow{
		Address: entry.Address, AccountType: entry.AccountType, ModuleName: entry.ModuleName, Amount: entry.Amount,
		Classification: unknownLocation, Eligibility: "UNDETERMINED", Restrictions: "UNKNOWN",
		Gaps: []string{"beneficial_ownership_unproven", "migration_eligibility_not_assessed"},
	}
	switch entry.AccountType {
	case moduleAccount:
		row.Classification = moduleCustody
		row.Gaps = append(row.Gaps, "module_label_not_personal_ownership", "beneficial_claims_against_custody_not_reconciled")
	case baseAccount:
		row.Classification = nonModuleLocation
		row.Gaps = append(row.Gaps, "base_account_label_not_unrestricted_personal_entitlement")
	case "/cosmos.vesting.v1beta1.BaseVestingAccount",
		"/cosmos.vesting.v1beta1.ContinuousVestingAccount",
		"/cosmos.vesting.v1beta1.DelayedVestingAccount",
		"/cosmos.vesting.v1beta1.PeriodicVestingAccount",
		"/cosmos.vesting.v1beta1.PermanentLockedAccount":
		row.Classification = nonModuleLocation
		row.Gaps = append(row.Gaps, "vesting_schedule_delegation_and_spendability_unavailable")
	case "bank_only":
		row.Gaps = append(row.Gaps, "auth_account_metadata_missing")
	default:
		row.Gaps = append(row.Gaps, "account_type_unrecognized")
	}
	return row
}
