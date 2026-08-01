package cross_stack_test

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	vestingtypes "github.com/zerone-chain/zerone/x/vesting_rewards/types"
)

// Pointer fields make omission distinguishable from an explicit false, zero,
// or empty string. The contract is closed: DisallowUnknownFields rejects
// additions while these presence checks reject deletions.
type moneyKarmaConstitution struct {
	Schema  *string `json:"schema"`
	Version *uint64 `json:"version"`
	Status  struct {
		Artifact               *string `json:"artifact"`
		RuntimeDeployed        *bool   `json:"runtime_deployed"`
		NetworkActivated       *bool   `json:"network_activated"`
		ClaimsPresentNoControl *bool   `json:"claims_present_no_control"`
	} `json:"status"`
	Separations struct {
		MoneyGrantsVoice           *bool `json:"money_grants_voice"`
		KarmaIsMoney               *bool `json:"karma_is_money"`
		RecognitionGrantsAuthority *bool `json:"recognition_grants_authority"`
	} `json:"separations"`
	FounderRenunciation struct {
		ShareBps                                       *uint64 `json:"share_bps"`
		Address                                        *string `json:"address"`
		Scope                                          *string `json:"scope"`
		V2FieldsRestorable                             *bool   `json:"v2_fields_restorable"`
		V2BeneficiarySubstitutionAllowed               *bool   `json:"v2_beneficiary_substitution_allowed"`
		OrdinaryParameterGovernanceMayChange           *bool   `json:"ordinary_parameter_governance_may_change"`
		FutureCoordinatedCodeChangeTechnicallyPossible *bool   `json:"future_coordinated_code_change_technically_possible"`
		CompatibilityFieldsOnly                        *bool   `json:"compatibility_fields_only"`
	} `json:"founder_renunciation"`
	Karma struct {
		Stage               *string `json:"stage"`
		Representation      *string `json:"representation"`
		EventType           *string `json:"event_type"`
		Meaning             *string `json:"meaning"`
		Transferable        *bool   `json:"transferable"`
		Delegable           *bool   `json:"delegable"`
		Saleable            *bool   `json:"saleable"`
		Collateralizable    *bool   `json:"collateralizable"`
		Inheritable         *bool   `json:"inheritable"`
		Denom               *bool   `json:"denom"`
		Balance             *bool   `json:"balance"`
		Bank                *bool   `json:"bank"`
		IBC                 *bool   `json:"ibc"`
		AMM                 *bool   `json:"amm"`
		RewardMultiplier    *bool   `json:"reward_multiplier"`
		Payout              *bool   `json:"payout"`
		GovernanceConsumer  *bool   `json:"governance_consumer"`
		NumericMagnitude    *bool   `json:"numeric_magnitude"`
		DedicatedStateStore *bool   `json:"dedicated_state_store"`
	} `json:"karma"`
	ResearchGovernance struct {
		SpendingPolicy                            *string `json:"spending_policy"`
		RuntimeIndependencePredicateEnforced      *bool   `json:"runtime_independence_predicate_enforced"`
		CurrentAuthorityManagedSpendingPathExists *bool   `json:"current_authority_managed_spending_path_exists"`
		ArtifactAuthorizesSpending                *bool   `json:"artifact_authorizes_spending"`
		CurrentKarmaConsumer                      *bool   `json:"current_karma_consumer"`
		FutureBoundary                            struct {
			ControllerCollapsed          *bool `json:"controller_collapsed"`
			BoundedEligibility           *bool `json:"bounded_eligibility"`
			BoundedSortition             *bool `json:"bounded_sortition"`
			IndependentCheckChamber      *bool `json:"independent_check_chamber"`
			RotationAndConcentrationCaps *bool `json:"rotation_and_concentration_caps"`
			ConflictDisclosure           *bool `json:"conflict_disclosure"`
			ChallengeAndTimelock         *bool `json:"challenge_and_timelock"`
			ScalarWeighting              *bool `json:"scalar_weighting"`
			UnilateralExecution          *bool `json:"unilateral_execution"`
		} `json:"future_boundary"`
	} `json:"research_governance"`
	OperationalCaveat struct {
		AsOf                                   *string `json:"as_of"`
		SoleFoundingHouseholdControlsValidator *bool   `json:"sole_founding_household_controls_validator"`
		SoleFoundingHouseholdControlsOperator  *bool   `json:"sole_founding_household_controls_operator"`
		SoleFoundingHouseholdControlsVote      *bool   `json:"sole_founding_household_controls_effective_vote"`
		PresentNoControlAchieved               *bool   `json:"present_no_control_achieved"`
		IndependentGovernanceProven            *bool   `json:"independent_governance_proven"`
	} `json:"operational_caveat"`
	Effects struct {
		Economic   *bool `json:"economic"`
		Governance *bool `json:"governance"`
		Consensus  *bool `json:"consensus"`
		Authority  *bool `json:"authority"`
	} `json:"effects"`
}

func TestMoneyKarma_ConstitutionIsStrictAndEffectless(t *testing.T) {
	repoRoot, err := findRepoRoot()
	require.NoError(t, err)

	contractPath := filepath.Join(repoRoot, "docs", "constitution", "money-karma-v1.json")
	f, err := os.Open(contractPath)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, f.Close()) })

	decoder := json.NewDecoder(f)
	decoder.DisallowUnknownFields()
	var contract moneyKarmaConstitution
	require.NoError(t, decoder.Decode(&contract), "machine constitution must strict-parse")
	var trailing any
	require.ErrorIs(t, decoder.Decode(&trailing), io.EOF, "machine constitution must contain one JSON value")

	requireString(t, "schema", contract.Schema, "zerone.money-karma.constitution/v1")
	requireUint64(t, "version", contract.Version, 1)
	requireString(t, "status.artifact", contract.Status.Artifact, "SOURCE_CONSTITUTION")
	requireFalse(t, map[string]*bool{
		"status.runtime_deployed":                  contract.Status.RuntimeDeployed,
		"status.network_activated":                 contract.Status.NetworkActivated,
		"status.claims_present_no_control":         contract.Status.ClaimsPresentNoControl,
		"separations.money_grants_voice":           contract.Separations.MoneyGrantsVoice,
		"separations.karma_is_money":               contract.Separations.KarmaIsMoney,
		"separations.recognition_grants_authority": contract.Separations.RecognitionGrantsAuthority,
		"founder.v2_fields_restorable":             contract.FounderRenunciation.V2FieldsRestorable,
		"founder.v2_beneficiary_substitution":      contract.FounderRenunciation.V2BeneficiarySubstitutionAllowed,
		"founder.ordinary_governance_may_change":   contract.FounderRenunciation.OrdinaryParameterGovernanceMayChange,
		"karma.transferable":                       contract.Karma.Transferable,
		"karma.delegable":                          contract.Karma.Delegable,
		"karma.saleable":                           contract.Karma.Saleable,
		"karma.collateralizable":                   contract.Karma.Collateralizable,
		"karma.inheritable":                        contract.Karma.Inheritable,
		"karma.denom":                              contract.Karma.Denom,
		"karma.balance":                            contract.Karma.Balance,
		"karma.bank":                               contract.Karma.Bank,
		"karma.ibc":                                contract.Karma.IBC,
		"karma.amm":                                contract.Karma.AMM,
		"karma.reward_multiplier":                  contract.Karma.RewardMultiplier,
		"karma.payout":                             contract.Karma.Payout,
		"karma.governance_consumer":                contract.Karma.GovernanceConsumer,
		"karma.numeric_magnitude":                  contract.Karma.NumericMagnitude,
		"karma.dedicated_state_store":              contract.Karma.DedicatedStateStore,
		"research.artifact_authorizes_spending":    contract.ResearchGovernance.ArtifactAuthorizesSpending,
		"research.runtime_independence_enforced":   contract.ResearchGovernance.RuntimeIndependencePredicateEnforced,
		"research.current_karma_consumer":          contract.ResearchGovernance.CurrentKarmaConsumer,
		"research.future.scalar_weighting":         contract.ResearchGovernance.FutureBoundary.ScalarWeighting,
		"research.future.unilateral_execution":     contract.ResearchGovernance.FutureBoundary.UnilateralExecution,
		"operations.present_no_control_achieved":   contract.OperationalCaveat.PresentNoControlAchieved,
		"operations.independent_governance_proven": contract.OperationalCaveat.IndependentGovernanceProven,
		"effects.economic":                         contract.Effects.Economic,
		"effects.governance":                       contract.Effects.Governance,
		"effects.consensus":                        contract.Effects.Consensus,
		"effects.authority":                        contract.Effects.Authority,
	})
	requireTrue(t, map[string]*bool{
		"founder.compatibility_fields_only":                 contract.FounderRenunciation.CompatibilityFieldsOnly,
		"founder.future_code_change_technically_possible":   contract.FounderRenunciation.FutureCoordinatedCodeChangeTechnicallyPossible,
		"research.current_authority_spending_path_exists":   contract.ResearchGovernance.CurrentAuthorityManagedSpendingPathExists,
		"research.future.controller_collapsed":              contract.ResearchGovernance.FutureBoundary.ControllerCollapsed,
		"research.future.bounded_eligibility":               contract.ResearchGovernance.FutureBoundary.BoundedEligibility,
		"research.future.bounded_sortition":                 contract.ResearchGovernance.FutureBoundary.BoundedSortition,
		"research.future.independent_check_chamber":         contract.ResearchGovernance.FutureBoundary.IndependentCheckChamber,
		"research.future.rotation_and_concentration_caps":   contract.ResearchGovernance.FutureBoundary.RotationAndConcentrationCaps,
		"research.future.conflict_disclosure":               contract.ResearchGovernance.FutureBoundary.ConflictDisclosure,
		"research.future.challenge_and_timelock":            contract.ResearchGovernance.FutureBoundary.ChallengeAndTimelock,
		"operations.sole_household_controls_validator":      contract.OperationalCaveat.SoleFoundingHouseholdControlsValidator,
		"operations.sole_household_controls_operator":       contract.OperationalCaveat.SoleFoundingHouseholdControlsOperator,
		"operations.sole_household_controls_effective_vote": contract.OperationalCaveat.SoleFoundingHouseholdControlsVote,
	})

	requireUint64(t, "founder.share_bps", contract.FounderRenunciation.ShareBps, 0)
	requireString(t, "founder.address", contract.FounderRenunciation.Address, "")
	requireString(t, "founder.scope", contract.FounderRenunciation.Scope, "VESTING_REWARDS_V2_FOUNDER_COMPATIBILITY_FIELDS")
	requireString(t, "karma.stage", contract.Karma.Stage, "K_ALPHA")
	requireString(t, "karma.representation", contract.Karma.Representation, "EVENT_ONLY")
	requireString(t, "karma.event_type", contract.Karma.EventType, "zerone.karma.edge")
	requireString(t, "karma.meaning", contract.Karma.Meaning, "DOMAIN_RELATIONS_NOT_HUMAN_WORTH_OR_TRUTH")
	requireString(t, "research.spending_policy", contract.ResearchGovernance.SpendingPolicy, "NORMATIVE_FAIL_CLOSED_UNTIL_GENUINELY_INDEPENDENT")
	requireString(t, "operational_caveat.as_of", contract.OperationalCaveat.AsOf, "2026-08-01")
}

func TestMoneyKarma_FounderRenunciationIsExecutable(t *testing.T) {
	current := vestingtypes.DefaultParams()
	require.Zero(t, current.FounderShareBps)
	require.Empty(t, current.FounderAddress)
	require.NoError(t, vestingtypes.ValidateParams(current))

	withShare := *current
	withShare.FounderShareBps = 1
	require.Error(t, vestingtypes.ValidateParams(&withShare), "any non-zero founder share must be invalid")
	require.ErrorIs(t, vestingtypes.ValidateFounderShareChange(current, &withShare), vestingtypes.ErrFounderShareRenounced)

	withAddress := *current
	withAddress.FounderAddress = "zerone1beneficiarycannotrestorefoundertap"
	require.Error(t, vestingtypes.ValidateParams(&withAddress), "any founder address must be invalid")
	require.ErrorIs(t, vestingtypes.ValidateFounderShareChange(current, &withAddress), vestingtypes.ErrFounderShareRenounced)

	withBoth := *current
	withBoth.FounderShareBps = 70_000
	withBoth.FounderAddress = "zerone1renamedbeneficiaryisstillabeneficiary"
	require.Error(t, vestingtypes.ValidateParams(&withBoth))
	require.ErrorIs(t, vestingtypes.ValidateFounderShareChange(current, &withBoth), vestingtypes.ErrFounderShareRenounced)

	proposedZero := *current
	require.NoError(t, vestingtypes.ValidateFounderShareChange(current, &proposedZero), "the only admissible proposal preserves zero and empty")
}

func TestMoneyKarma_NoSensitiveProductionModuleConsumesKarma(t *testing.T) {
	repoRoot, err := findRepoRoot()
	require.NoError(t, err)

	moduleDirs := []string{
		"x/gov",
		"x/staking",
		"x/vesting_rewards",
		"x/claiming_pot",
		"x/liquiditypool",
		"x/tokens",
		"x/qualification",
		"x/trust_score",
		"x/creed",
		"x/emergency",
	}
	var findings []string
	for _, relDir := range moduleDirs {
		dirFindings, scanErr := scanProductionGoForWord(filepath.Join(repoRoot, relDir), "karma", repoRoot)
		require.NoError(t, scanErr, "AST-scan %s", relDir)
		findings = append(findings, dirFindings...)
	}
	sort.Strings(findings)
	require.Empty(t, findings,
		"KARMA must have no governance, staking, payout, token, qualification, trust, creed, or emergency consumer; production AST references found:\n%s",
		strings.Join(findings, "\n"))
}

func TestMoneyKarma_KAlphaProducersRemainEventOnlyAndOrdinal(t *testing.T) {
	repoRoot, err := findRepoRoot()
	require.NoError(t, err)

	producerRoots := []string{
		filepath.Join(repoRoot, "x", "knowledge"),
		filepath.Join(repoRoot, "x", "substrate_bridge"),
	}
	var producers []string
	for _, producerRoot := range producerRoots {
		err = filepath.WalkDir(producerRoot, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			fset := token.NewFileSet()
			file, parseErr := parser.ParseFile(fset, path, nil, 0)
			if parseErr != nil {
				return parseErr
			}
			for _, decl := range file.Decls {
				fn, ok := decl.(*ast.FuncDecl)
				if !ok || fn.Body == nil {
					continue
				}
				auditKarmaCallExtras(t, fset, fn)
				if !functionEmitsKarmaEdge(fn) {
					continue
				}
				rel, relErr := filepath.Rel(repoRoot, path)
				require.NoError(t, relErr)
				producers = append(producers, filepath.ToSlash(rel)+":"+fn.Name.Name)
				auditKarmaEmitter(t, fset, fn)
			}
			return nil
		})
		require.NoError(t, err)
	}

	sort.Strings(producers)
	require.Equal(t, []string{
		"x/knowledge/keeper/rounds.go:emitKarmaEdgeState",
		"x/substrate_bridge/keeper/settlement.go:emitExternalCiteKarma",
	}, producers, "new KARMA event producers require explicit constitutional review")
}

func requireFalse(t *testing.T, fields map[string]*bool) {
	t.Helper()
	for name, value := range fields {
		require.NotNil(t, value, "%s must be explicitly present", name)
		require.False(t, *value, "%s must remain false", name)
	}
}

func requireTrue(t *testing.T, fields map[string]*bool) {
	t.Helper()
	for name, value := range fields {
		require.NotNil(t, value, "%s must be explicitly present", name)
		require.True(t, *value, "%s must remain true", name)
	}
}

func requireString(t *testing.T, name string, actual *string, expected string) {
	t.Helper()
	require.NotNil(t, actual, "%s must be explicitly present", name)
	require.Equal(t, expected, *actual, "%s drifted", name)
}

func requireUint64(t *testing.T, name string, actual *uint64, expected uint64) {
	t.Helper()
	require.NotNil(t, actual, "%s must be explicitly present", name)
	require.Equal(t, expected, *actual, "%s drifted", name)
}

// scanProductionGoForWord uses the Go AST, not source grep: comments are
// deliberately excluded, while identifiers, import paths, and string literals
// remain visible. Test files cannot satisfy or trip a production invariant.
func scanProductionGoForWord(dir, needle, repoRoot string) ([]string, error) {
	var findings []string
	err := filepath.WalkDir(dir, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return err
		}
		ast.Inspect(file, func(node ast.Node) bool {
			var kind, value string
			switch typed := node.(type) {
			case *ast.Ident:
				kind, value = "identifier", typed.Name
			case *ast.ImportSpec:
				kind, value = "import", unquoteGoString(typed.Path.Value)
			case *ast.BasicLit:
				if typed.Kind == token.STRING {
					kind, value = "string", unquoteGoString(typed.Value)
				}
			}
			if kind != "" && strings.Contains(strings.ToLower(value), strings.ToLower(needle)) {
				rel, relErr := filepath.Rel(repoRoot, path)
				if relErr != nil {
					rel = path
				}
				pos := fset.Position(node.Pos())
				findings = append(findings, fmt.Sprintf("%s:%d: %s %q", filepath.ToSlash(rel), pos.Line, kind, value))
			}
			return true
		})
		return nil
	})
	return findings, err
}

func functionEmitsKarmaEdge(fn *ast.FuncDecl) bool {
	emits := false
	ast.Inspect(fn.Body, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok || selectorName(call.Fun) != "NewEvent" || len(call.Args) == 0 {
			return true
		}
		if stringLiteral(call.Args[0]) == "zerone.karma.edge" {
			emits = true
		}
		return true
	})
	return emits
}

func auditKarmaEmitter(t *testing.T, fset *token.FileSet, fn *ast.FuncDecl) {
	t.Helper()
	allowedKeys := map[string]bool{
		"beneficiary":  true,
		"kind":         true,
		"state":        true,
		"counterparty": true,
		"ref_id":       true,
		"domain":       true,
		"register":     true,
		"self":         true,
	}
	seenKeys := map[string]bool{}
	emitCalls := 0
	var forbidden []string

	ast.Inspect(fn.Body, func(node ast.Node) bool {
		switch typed := node.(type) {
		case *ast.Ident:
			lower := strings.ToLower(typed.Name)
			if lower == "coin" || lower == "coins" || lower == "denom" || lower == "balance" ||
				lower == "magnitude" || lower == "amount" || lower == "bank" || lower == "bankkeeper" {
				forbidden = append(forbidden, fmt.Sprintf("%s:%d identifier %q", fn.Name.Name, fset.Position(typed.Pos()).Line, typed.Name))
			}
		case *ast.CallExpr:
			name := selectorName(typed.Fun)
			lower := strings.ToLower(name)
			if name == "EmitEvent" {
				emitCalls++
			}
			if strings.HasPrefix(lower, "set") || strings.HasPrefix(lower, "store") ||
				strings.HasPrefix(lower, "save") || strings.HasPrefix(lower, "delete") ||
				strings.HasPrefix(lower, "insert") || strings.HasPrefix(lower, "update") ||
				strings.Contains(lower, "coin") || strings.Contains(lower, "bank") ||
				strings.Contains(lower, "mint") || strings.Contains(lower, "burn") ||
				strings.Contains(lower, "transfer") || strings.Contains(lower, "delegate") ||
				strings.Contains(lower, "vote") || strings.Contains(lower, "proposal") {
				forbidden = append(forbidden, fmt.Sprintf("%s:%d call %q", fn.Name.Name, fset.Position(typed.Pos()).Line, name))
			}
			if name == "NewAttribute" {
				if len(typed.Args) < 2 {
					forbidden = append(forbidden, fmt.Sprintf("%s:%d malformed event attribute", fn.Name.Name, fset.Position(typed.Pos()).Line))
					return true
				}
				key := stringLiteral(typed.Args[0])
				if !allowedKeys[key] {
					forbidden = append(forbidden, fmt.Sprintf("%s:%d non-ordinal attribute key %q", fn.Name.Name, fset.Position(typed.Pos()).Line, key))
				} else {
					seenKeys[key] = true
				}
			}
		}
		return true
	})

	require.Empty(t, forbidden, "K-alpha producer must emit ordinal events without coin, magnitude, store, payout, or governance operations")
	require.Positive(t, emitCalls, "%s must actually emit its event", fn.Name.Name)
	require.Equal(t, allowedKeys, seenKeys, "%s event attributes must remain the closed ordinal vocabulary", fn.Name.Name)
}

// Knowledge's emitter accepts variadic ordinal metadata. Audit direct metadata
// and locally-built slices so that this escape hatch cannot grow a numeric
// magnitude without a failing test. The one transparent wrapper is allowed to
// forward its parameter because every call into that wrapper is audited too.
func auditKarmaCallExtras(t *testing.T, fset *token.FileSet, fn *ast.FuncDecl) {
	t.Helper()
	tracked := map[string]bool{}
	ast.Inspect(fn.Body, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		name := directCallName(call.Fun)
		fixed := 0
		switch name {
		case "emitKarmaEdge":
			fixed = 6
		case "emitKarmaEdgeState":
			fixed = 7
		default:
			return true
		}
		for _, extra := range call.Args[fixed:] {
			if ident, ok := extra.(*ast.Ident); ok {
				tracked[ident.Name] = true
				continue
			}
			auditOrdinalExtraAttribute(t, fset, extra)
		}
		return true
	})

	if len(tracked) == 0 || fn.Name.Name == "emitKarmaEdge" {
		return
	}

	for trackedName := range tracked {
		if functionParameterNamed(fn, trackedName) {
			t.Fatalf("%s:%d dynamically forwards KARMA attributes through parameter %q", fn.Name.Name, fset.Position(fn.Pos()).Line, trackedName)
		}
	}

	ast.Inspect(fn.Body, func(node ast.Node) bool {
		assign, ok := node.(*ast.AssignStmt)
		if !ok {
			return true
		}
		for _, lhs := range assign.Lhs {
			ident, ok := lhs.(*ast.Ident)
			if !ok || !tracked[ident.Name] {
				continue
			}
			for _, rhs := range assign.Rhs {
				appendCall, ok := rhs.(*ast.CallExpr)
				if !ok || directCallName(appendCall.Fun) != "append" || len(appendCall.Args) < 2 {
					t.Fatalf("%s:%d KARMA attribute slice %q has a dynamic assignment", fn.Name.Name, fset.Position(rhs.Pos()).Line, ident.Name)
				}
				base, ok := appendCall.Args[0].(*ast.Ident)
				if !ok || base.Name != ident.Name {
					t.Fatalf("%s:%d KARMA attribute slice %q is replaced instead of locally appended", fn.Name.Name, fset.Position(rhs.Pos()).Line, ident.Name)
				}
				for _, appended := range appendCall.Args[1:] {
					auditOrdinalExtraAttribute(t, fset, appended)
				}
			}
		}
		return true
	})
}

func auditOrdinalExtraAttribute(t *testing.T, fset *token.FileSet, expr ast.Expr) {
	t.Helper()
	call, ok := expr.(*ast.CallExpr)
	require.True(t, ok, "KARMA metadata at line %d must be an explicit sdk.NewAttribute call", fset.Position(expr.Pos()).Line)
	require.Equal(t, "NewAttribute", selectorName(call.Fun), "KARMA metadata at line %d must be an event attribute", fset.Position(expr.Pos()).Line)
	require.Len(t, call.Args, 2)

	key := stringLiteral(call.Args[0])
	require.Contains(t, map[string]bool{"correct": true, "verdict": true}, key,
		"KARMA metadata key %q at line %d is outside the ordinal allowlist", key, fset.Position(expr.Pos()).Line)

	if value := stringLiteral(call.Args[1]); value != "" {
		require.Contains(t, map[string]bool{"true": true, "false": true}, value,
			"literal KARMA metadata must be boolean, never numeric magnitude")
		return
	}
	valueCall, ok := call.Args[1].(*ast.CallExpr)
	require.True(t, ok, "KARMA metadata value at line %d must be boolean or an enum String()", fset.Position(call.Args[1].Pos()).Line)
	require.Equal(t, "String", selectorName(valueCall.Fun), "KARMA metadata value at line %d must be ordinal", fset.Position(call.Args[1].Pos()).Line)
}

func functionParameterNamed(fn *ast.FuncDecl, name string) bool {
	if fn.Type.Params == nil {
		return false
	}
	for _, field := range fn.Type.Params.List {
		for _, fieldName := range field.Names {
			if fieldName.Name == name {
				return true
			}
		}
	}
	return false
}

func directCallName(expr ast.Expr) string {
	switch typed := expr.(type) {
	case *ast.Ident:
		return typed.Name
	case *ast.SelectorExpr:
		return typed.Sel.Name
	default:
		return ""
	}
}

func selectorName(expr ast.Expr) string {
	selector, ok := expr.(*ast.SelectorExpr)
	if !ok {
		return ""
	}
	return selector.Sel.Name
}

func stringLiteral(expr ast.Expr) string {
	literal, ok := expr.(*ast.BasicLit)
	if !ok || literal.Kind != token.STRING {
		return ""
	}
	return unquoteGoString(literal.Value)
}

func unquoteGoString(value string) string {
	unquoted, err := strconv.Unquote(value)
	if err != nil {
		return value
	}
	return unquoted
}
