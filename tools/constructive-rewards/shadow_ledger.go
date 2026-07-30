package main

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"sync"
	"unicode/utf8"
)

const shadowLedgerArithmeticVersion = "constructive-shadow-ledger-int-v1"

// ShadowLedgerConfig is a zero-value, exact-integer policy snapshot for one
// canonical economic root. Units are deliberately unnamed model units, never
// ZRN or a promise of value.
type ShadowLedgerConfig struct {
	RootID                       string   `json:"root_id"`
	PolicyDigest                 string   `json:"policy_digest"`
	LifetimeCap                  uint64   `json:"lifetime_cap"`
	LifetimeReplacementCap       uint64   `json:"lifetime_replacement_cap"`
	InitiallyExcludedControllers []string `json:"initially_excluded_controllers"`
}

// ShadowAccrual is the only input allowed to raise accrued capacity.
type ShadowAccrual struct {
	LotID                  string   `json:"lot_id"`
	ReceiptDigest          string   `json:"receipt_digest"`
	Amount                 uint64   `json:"amount"`
	Deadline               uint64   `json:"deadline"`
	BeneficiaryControllers []string `json:"beneficiary_controllers"`
	DependencyControllers  []string `json:"dependency_controllers"`
	EvaluatorControllers   []string `json:"evaluator_controllers"`
}

// ShadowInvalidation is a final adjudication certificate. A raw challenge
// cannot enter this transition.
type ShadowInvalidation struct {
	DecisionID             string   `json:"decision_id"`
	TargetReceiptDigest    string   `json:"target_receipt_digest"`
	CleanSupportTarget     uint64   `json:"clean_support_target"`
	CulpableControllers    []string `json:"culpable_controllers"`
	ChallengerControllers  []string `json:"challenger_controllers"`
	AdjudicatorControllers []string `json:"adjudicator_controllers"`
	OrganizationRoots      []string `json:"organization_roots"`
	Final                  bool     `json:"final"`
	AssignmentAfterFreeze  bool     `json:"assignment_after_freeze"`
	OutcomeIndependentPay  bool     `json:"outcome_independent_review_pay"`
}

// ShadowSuccessor carries only the support and controller closures needed by
// the counterfactual ledger. It is not a truth oracle or an authority record.
type ShadowSuccessor struct {
	ReceiptDigest          string   `json:"receipt_digest"`
	PriorReceiptDigest     string   `json:"prior_receipt_digest"`
	SupportTarget          uint64   `json:"support_target"`
	BeneficiaryControllers []string `json:"beneficiary_controllers"`
	DependencyControllers  []string `json:"dependency_controllers"`
	EvaluatorControllers   []string `json:"evaluator_controllers"`
	PolicyPassed           bool     `json:"policy_passed"`
}

// ShadowReplacementClaim preserves the controller and disposition history for
// one successor receipt. Older funded successors remain targetable by a later
// final invalidation without reopening funded capacity.
type ShadowReplacementClaim struct {
	ReceiptDigest string   `json:"receipt_digest"`
	Controllers   []string `json:"controllers"`
	Attributed    uint64   `json:"attributed"`
	Live          uint64   `json:"live"`
	Funded        uint64   `json:"funded"`
	Extinguished  uint64   `json:"extinguished"`
}

// ShadowCapacityLot partitions one immutable accrual into terminal funded,
// live original, quarantined, live one-shot replacement, and extinguished
// capacity.
type ShadowCapacityLot struct {
	ID                        string                   `json:"id"`
	ReceiptDigest             string                   `json:"receipt_digest"`
	ReplacementReceiptDigest  string                   `json:"replacement_receipt_digest,omitempty"`
	ReplacementReceiptHistory []string                 `json:"replacement_receipt_history,omitempty"`
	ReplacementClaims         []ShadowReplacementClaim `json:"replacement_claims,omitempty"`
	Amount                    uint64                   `json:"amount"`
	Funded                    uint64                   `json:"funded"`
	OriginalLive              uint64                   `json:"original_live"`
	Quarantined               uint64                   `json:"quarantined"`
	ReplacementLive           uint64                   `json:"replacement_live"`
	Extinguished              uint64                   `json:"extinguished"`
	Deadline                  uint64                   `json:"deadline"`
	BeneficiaryControllers    []string                 `json:"beneficiary_controllers"`
	DependencyControllers     []string                 `json:"dependency_controllers"`
	EvaluatorControllers      []string                 `json:"evaluator_controllers"`
	ReplacementControllers    []string                 `json:"replacement_controllers,omitempty"`
	ReplacementGeneration     uint8                    `json:"replacement_generation"`
	ReplacementAttributed     uint64                   `json:"replacement_attributed"`
}

// ShadowLedgerSnapshot is behavior-complete for the in-memory experiment.
// Restoring it cannot reset consumed events, controller links, replacement
// exposure, or deadlines.
type ShadowLedgerSnapshot struct {
	ArithmeticVersion   string              `json:"arithmetic_version"`
	Config              ShadowLedgerConfig  `json:"config"`
	CurrentEpoch        uint64              `json:"current_epoch"`
	Accrued             uint64              `json:"accrued"`
	Funded              uint64              `json:"funded"`
	Live                uint64              `json:"live"`
	Quarantined         uint64              `json:"quarantined"`
	Extinguished        uint64              `json:"extinguished"`
	ReplacementUsed     uint64              `json:"replacement_used"`
	CleanSupportTarget  uint64              `json:"clean_support_target"`
	Lots                []ShadowCapacityLot `json:"lots"`
	AcceptedEventCount  uint64              `json:"accepted_event_count"`
	ConsumedEventIDs    []string            `json:"consumed_event_ids"`
	ConsumedDecisionIDs []string            `json:"consumed_decision_ids"`
	InvalidatedReceipts []string            `json:"invalidated_receipts"`
	ControllerLinks     map[string]string   `json:"controller_links"`
	ExcludedControllers []string            `json:"excluded_controllers"`
	StateCommitment     string              `json:"state_commitment"`
}

type shadowLedgerState struct {
	CurrentEpoch        uint64
	Accrued             uint64
	Funded              uint64
	Extinguished        uint64
	ReplacementUsed     uint64
	CleanSupportTarget  uint64
	Lots                map[string]ShadowCapacityLot
	AcceptedEventCount  uint64
	ConsumedEventIDs    map[string]struct{}
	ConsumedDecisionIDs map[string]struct{}
	InvalidatedReceipts map[string]struct{}
	ControllerLinks     map[string]string
	ExcludedControllers map[string]struct{}
}

// ShadowCapacityLedger is a serializable, in-memory accounting experiment. It
// has no bank, escrow, network, qualification, or consensus authority.
type ShadowCapacityLedger struct {
	mu     sync.Mutex
	config ShadowLedgerConfig
	state  shadowLedgerState
}

func validateSortedUniqueStrings(name string, values []string, allowEmpty bool) error {
	if !allowEmpty && len(values) == 0 {
		return fmt.Errorf("%s must not be empty", name)
	}
	for index, value := range values {
		if err := validateShadowString(name, value); err != nil {
			return err
		}
		if index > 0 && values[index-1] >= value {
			return fmt.Errorf("%s must be sorted and unique", name)
		}
	}
	return nil
}

func validateShadowString(name string, value string) error {
	if value == "" {
		return fmt.Errorf("%s must not be empty", name)
	}
	if !utf8.ValidString(value) {
		return fmt.Errorf("%s must contain valid UTF-8", name)
	}
	return nil
}

func validateShadowCommitment(name string, value string) error {
	if len(value) != sha256.Size*2 {
		return fmt.Errorf("%s must be a lowercase SHA-256 digest", name)
	}
	for _, character := range value {
		if (character < '0' || character > '9') &&
			(character < 'a' || character > 'f') {
			return fmt.Errorf("%s must be a lowercase SHA-256 digest", name)
		}
	}
	return nil
}

func validateDeclaredControllerClosures(groups ...[]string) error {
	seen := make(map[string]struct{})
	for _, group := range groups {
		for _, controller := range group {
			if _, duplicate := seen[controller]; duplicate {
				return fmt.Errorf("controller %q appears in more than one declared closure", controller)
			}
			seen[controller] = struct{}{}
		}
	}
	return nil
}

func checkedAddUint64(left, right uint64) (uint64, error) {
	result := left + right
	if result < left {
		return 0, errors.New("integer capacity overflow")
	}
	return result, nil
}

func cloneStringSet(input map[string]struct{}) map[string]struct{} {
	result := make(map[string]struct{}, len(input))
	for value := range input {
		result[value] = struct{}{}
	}
	return result
}

func cloneStringMap(input map[string]string) map[string]string {
	result := make(map[string]string, len(input))
	for key, value := range input {
		result[key] = value
	}
	return result
}

func equalStringSlices(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func cloneShadowLot(input ShadowCapacityLot) ShadowCapacityLot {
	result := input
	result.BeneficiaryControllers = append([]string(nil), input.BeneficiaryControllers...)
	result.DependencyControllers = append([]string(nil), input.DependencyControllers...)
	result.EvaluatorControllers = append([]string(nil), input.EvaluatorControllers...)
	result.ReplacementControllers = append([]string(nil), input.ReplacementControllers...)
	result.ReplacementReceiptHistory = append(
		[]string(nil),
		input.ReplacementReceiptHistory...,
	)
	result.ReplacementClaims = make(
		[]ShadowReplacementClaim,
		len(input.ReplacementClaims),
	)
	for index, claim := range input.ReplacementClaims {
		result.ReplacementClaims[index] = claim
		result.ReplacementClaims[index].Controllers = append(
			[]string(nil),
			claim.Controllers...,
		)
	}
	return result
}

func cloneShadowState(input shadowLedgerState) shadowLedgerState {
	result := input
	result.Lots = make(map[string]ShadowCapacityLot, len(input.Lots))
	for id, lot := range input.Lots {
		result.Lots[id] = cloneShadowLot(lot)
	}
	result.ConsumedEventIDs = cloneStringSet(input.ConsumedEventIDs)
	result.ConsumedDecisionIDs = cloneStringSet(input.ConsumedDecisionIDs)
	result.InvalidatedReceipts = cloneStringSet(input.InvalidatedReceipts)
	result.ControllerLinks = cloneStringMap(input.ControllerLinks)
	result.ExcludedControllers = cloneStringSet(input.ExcludedControllers)
	return result
}

func NewShadowCapacityLedger(config ShadowLedgerConfig) (*ShadowCapacityLedger, error) {
	if err := validateShadowString("shadow ledger root", config.RootID); err != nil {
		return nil, err
	}
	if err := validateShadowString("shadow ledger policy digest", config.PolicyDigest); err != nil {
		return nil, err
	}
	if config.LifetimeCap == 0 {
		return nil, errors.New("shadow ledger lifetime cap must be positive")
	}
	if config.LifetimeReplacementCap > config.LifetimeCap {
		return nil, errors.New("replacement cap cannot exceed lifetime cap")
	}
	if err := validateSortedUniqueStrings(
		"initially excluded controllers",
		config.InitiallyExcludedControllers,
		true,
	); err != nil {
		return nil, err
	}
	config.InitiallyExcludedControllers = append(
		[]string(nil),
		config.InitiallyExcludedControllers...,
	)
	state := shadowLedgerState{
		Lots:                make(map[string]ShadowCapacityLot),
		ConsumedEventIDs:    make(map[string]struct{}),
		ConsumedDecisionIDs: make(map[string]struct{}),
		InvalidatedReceipts: make(map[string]struct{}),
		ControllerLinks:     make(map[string]string),
		ExcludedControllers: make(map[string]struct{}),
	}
	for _, controller := range config.InitiallyExcludedControllers {
		state.ControllerLinks[controller] = controller
		state.ExcludedControllers[controller] = struct{}{}
	}
	ledger := &ShadowCapacityLedger{config: config, state: state}
	if err := ledger.validateState(state); err != nil {
		return nil, err
	}
	return ledger, nil
}

func (l *ShadowCapacityLedger) controllerRoot(state shadowLedgerState, controller string) string {
	root, exists := state.ControllerLinks[controller]
	if !exists {
		return controller
	}
	for {
		parent, exists := state.ControllerLinks[root]
		if !exists || parent == root {
			return root
		}
		root = parent
	}
}

func (l *ShadowCapacityLedger) groupExcluded(state shadowLedgerState, controller string) bool {
	root := l.controllerRoot(state, controller)
	for excluded := range state.ExcludedControllers {
		if l.controllerRoot(state, excluded) == root {
			return true
		}
	}
	return false
}

func (l *ShadowCapacityLedger) validateEffectiveControllerClosures(
	state shadowLedgerState,
	groups ...[]string,
) error {
	seen := make(map[string]string)
	for _, group := range groups {
		for _, controller := range group {
			root := l.controllerRoot(state, controller)
			if prior, duplicate := seen[root]; duplicate {
				return fmt.Errorf(
					"controllers %q and %q collapse to the same effective root %q",
					prior,
					controller,
					root,
				)
			}
			seen[root] = controller
		}
	}
	return nil
}

func (l *ShadowCapacityLedger) effectiveControllerClosuresOverlap(
	state shadowLedgerState,
	left []string,
	right []string,
) bool {
	rightRoots := make(map[string]struct{}, len(right))
	for _, controller := range right {
		rightRoots[l.controllerRoot(state, controller)] = struct{}{}
	}
	for _, controller := range left {
		if _, exists := rightRoots[l.controllerRoot(state, controller)]; exists {
			return true
		}
	}
	return false
}

func shadowLotControllers(lot ShadowCapacityLot) []string {
	controllers := make([]string, 0,
		len(lot.BeneficiaryControllers)+
			len(lot.DependencyControllers)+
			len(lot.EvaluatorControllers),
	)
	controllers = append(controllers, lot.BeneficiaryControllers...)
	controllers = append(controllers, lot.DependencyControllers...)
	controllers = append(controllers, lot.EvaluatorControllers...)
	sort.Strings(controllers)
	return controllers
}

func successorControllers(successor ShadowSuccessor) []string {
	controllers := make([]string, 0,
		len(successor.BeneficiaryControllers)+
			len(successor.DependencyControllers)+
			len(successor.EvaluatorControllers),
	)
	controllers = append(controllers, successor.BeneficiaryControllers...)
	controllers = append(controllers, successor.DependencyControllers...)
	controllers = append(controllers, successor.EvaluatorControllers...)
	sort.Strings(controllers)
	return controllers
}

func (l *ShadowCapacityLedger) lotLive(lot ShadowCapacityLot) uint64 {
	return lot.OriginalLive + lot.ReplacementLive
}

func (l *ShadowCapacityLedger) totals(state shadowLedgerState) (
	live uint64,
	quarantined uint64,
	err error,
) {
	for _, lot := range state.Lots {
		live, err = checkedAddUint64(live, l.lotLive(lot))
		if err != nil {
			return 0, 0, err
		}
		quarantined, err = checkedAddUint64(quarantined, lot.Quarantined)
		if err != nil {
			return 0, 0, err
		}
	}
	return live, quarantined, nil
}

func validateControllerGraph(state shadowLedgerState) error {
	for controller, parent := range state.ControllerLinks {
		if err := validateShadowString("controller link", controller); err != nil {
			return err
		}
		if err := validateShadowString("controller parent", parent); err != nil {
			return err
		}
		if _, exists := state.ControllerLinks[parent]; !exists {
			return fmt.Errorf("controller %q links to unknown parent %q", controller, parent)
		}
		seen := make(map[string]struct{}, len(state.ControllerLinks))
		current := controller
		for {
			if _, duplicate := seen[current]; duplicate {
				return fmt.Errorf("controller links contain a cycle through %q", current)
			}
			seen[current] = struct{}{}
			next := state.ControllerLinks[current]
			if next == current {
				break
			}
			current = next
		}
	}
	componentMinimum := make(map[string]string)
	for controller, parent := range state.ControllerLinks {
		if state.ControllerLinks[parent] != parent {
			return fmt.Errorf(
				"controller %q does not link directly to canonical root %q",
				controller,
				parent,
			)
		}
		minimum, exists := componentMinimum[parent]
		if !exists || controller < minimum {
			componentMinimum[parent] = controller
		}
	}
	for root, minimum := range componentMinimum {
		if root != minimum {
			return fmt.Errorf(
				"controller root %q is not the component's lexical minimum %q",
				root,
				minimum,
			)
		}
	}
	for controller := range state.ExcludedControllers {
		if err := validateShadowString("excluded controller", controller); err != nil {
			return err
		}
		if _, exists := state.ControllerLinks[controller]; !exists {
			return fmt.Errorf("excluded controller %q is missing from controller links", controller)
		}
	}
	return nil
}

func (l *ShadowCapacityLedger) validateState(state shadowLedgerState) error {
	if state.Accrued > l.config.LifetimeCap {
		return errors.New("accrued capacity exceeds lifetime cap")
	}
	if state.ReplacementUsed > l.config.LifetimeReplacementCap {
		return errors.New("replacement use exceeds lifetime replacement cap")
	}
	if state.CleanSupportTarget > state.Accrued {
		return errors.New("clean support exceeds accrued capacity")
	}
	if len(state.ConsumedDecisionIDs) != len(state.InvalidatedReceipts) {
		return errors.New("consumed invalidation decisions and receipts differ")
	}
	if state.AcceptedEventCount != uint64(len(state.ConsumedEventIDs)) {
		return errors.New("accepted event counter and replay IDs differ")
	}
	for eventID := range state.ConsumedEventIDs {
		if err := validateShadowString("consumed event ID", eventID); err != nil {
			return err
		}
	}
	for decisionID := range state.ConsumedDecisionIDs {
		if err := validateShadowString("consumed decision ID", decisionID); err != nil {
			return err
		}
	}
	for receipt := range state.InvalidatedReceipts {
		if err := validateShadowString("invalidated receipt", receipt); err != nil {
			return err
		}
	}
	if err := validateControllerGraph(state); err != nil {
		return err
	}
	for _, controller := range l.config.InitiallyExcludedControllers {
		if _, linked := state.ControllerLinks[controller]; !linked {
			return fmt.Errorf("initially excluded controller %q lost its link", controller)
		}
		if _, excluded := state.ExcludedControllers[controller]; !excluded {
			return fmt.Errorf("initially excluded controller %q lost its exclusion", controller)
		}
	}
	originalReceipts := make(map[string]string, len(state.Lots))
	for id, lot := range state.Lots {
		if priorLot, duplicate := originalReceipts[lot.ReceiptDigest]; duplicate {
			return fmt.Errorf(
				"lots %q and %q reuse original receipt %q",
				priorLot,
				id,
				lot.ReceiptDigest,
			)
		}
		originalReceipts[lot.ReceiptDigest] = id
	}
	allReceipts := make(map[string]string, len(originalReceipts))
	for receipt, id := range originalReceipts {
		allReceipts[receipt] = fmt.Sprintf("original lot %q", id)
	}
	activeReplacementReceipt := ""
	var accrued, funded, extinguished, live, quarantined, replacementUsed uint64
	for id, lot := range state.Lots {
		if err := validateShadowString("shadow lot ID", id); err != nil {
			return err
		}
		if lot.ID != id {
			return errors.New("shadow lot ID/key mismatch")
		}
		if err := validateShadowString("original receipt digest", lot.ReceiptDigest); err != nil {
			return fmt.Errorf("lot %q: %w", id, err)
		}
		if lot.Amount == 0 || lot.Deadline == 0 {
			return fmt.Errorf("lot %q has invalid amount or deadline", id)
		}
		if lot.ReplacementGeneration > 1 {
			return fmt.Errorf("lot %q exceeds one replacement generation", id)
		}
		for name, values := range map[string][]string{
			"beneficiary controllers": lot.BeneficiaryControllers,
			"dependency controllers":  lot.DependencyControllers,
			"evaluator controllers":   lot.EvaluatorControllers,
			"replacement controllers": lot.ReplacementControllers,
		} {
			allowEmpty := name != "beneficiary controllers"
			if name == "replacement controllers" && lot.ReplacementLive > 0 {
				allowEmpty = false
			}
			if err := validateSortedUniqueStrings(name, values, allowEmpty); err != nil {
				return fmt.Errorf("lot %q: %w", id, err)
			}
		}
		if err := validateDeclaredControllerClosures(
			lot.BeneficiaryControllers,
			lot.DependencyControllers,
			lot.EvaluatorControllers,
		); err != nil {
			return fmt.Errorf("lot %q: %w", id, err)
		}
		if len(lot.ReplacementReceiptHistory) != len(lot.ReplacementClaims) {
			return fmt.Errorf("lot %q replacement history and claims differ", id)
		}
		var claimAttributed, claimLive, claimFunded, claimExtinguished uint64
		activeClaim := -1
		for index, claim := range lot.ReplacementClaims {
			if err := validateShadowString(
				"replacement claim receipt",
				claim.ReceiptDigest,
			); err != nil {
				return fmt.Errorf("lot %q: %w", id, err)
			}
			if lot.ReplacementReceiptHistory[index] != claim.ReceiptDigest {
				return fmt.Errorf("lot %q has an invalid replacement claim receipt", id)
			}
			if priorOwner, duplicate := allReceipts[claim.ReceiptDigest]; duplicate {
				return fmt.Errorf(
					"lot %q replacement receipt %q is already owned by %s",
					id,
					claim.ReceiptDigest,
					priorOwner,
				)
			}
			allReceipts[claim.ReceiptDigest] = fmt.Sprintf(
				"replacement claim in lot %q",
				id,
			)
			if err := validateSortedUniqueStrings(
				"replacement claim controllers",
				claim.Controllers,
				false,
			); err != nil {
				return fmt.Errorf("lot %q: %w", id, err)
			}
			claimPartition, err := checkedAddUint64(claim.Live, claim.Funded)
			if err != nil {
				return err
			}
			claimPartition, err = checkedAddUint64(claimPartition, claim.Extinguished)
			if err != nil {
				return err
			}
			if claim.Attributed == 0 || claimPartition != claim.Attributed {
				return fmt.Errorf("lot %q replacement claim does not partition its attribution", id)
			}
			if claim.Live > 0 {
				if activeClaim != -1 {
					return fmt.Errorf("lot %q has more than one live successor receipt", id)
				}
				if activeReplacementReceipt != "" {
					return fmt.Errorf(
						"replacement receipts %q and %q are live concurrently",
						activeReplacementReceipt,
						claim.ReceiptDigest,
					)
				}
				activeClaim = index
				activeReplacementReceipt = claim.ReceiptDigest
			}
			claimAttributed, err = checkedAddUint64(claimAttributed, claim.Attributed)
			if err != nil {
				return err
			}
			claimLive, err = checkedAddUint64(claimLive, claim.Live)
			if err != nil {
				return err
			}
			claimFunded, err = checkedAddUint64(claimFunded, claim.Funded)
			if err != nil {
				return err
			}
			claimExtinguished, err = checkedAddUint64(
				claimExtinguished,
				claim.Extinguished,
			)
			if err != nil {
				return err
			}
		}
		if claimAttributed != lot.ReplacementAttributed ||
			claimLive != lot.ReplacementLive ||
			claimFunded > lot.Funded ||
			claimExtinguished > lot.Extinguished {
			return fmt.Errorf("lot %q replacement claim aggregates do not match its partition", id)
		}
		if lot.ReplacementAttributed > lot.Amount {
			return fmt.Errorf("lot %q replacement attribution exceeds its amount", id)
		}
		if lot.ReplacementAttributed == 0 {
			if lot.ReplacementGeneration != 0 ||
				lot.ReplacementReceiptDigest != "" ||
				len(lot.ReplacementReceiptHistory) != 0 ||
				len(lot.ReplacementControllers) != 0 ||
				len(lot.ReplacementClaims) != 0 {
				return fmt.Errorf("lot %q has replacement metadata without attribution", id)
			}
		} else if lot.ReplacementGeneration != 1 ||
			len(lot.ReplacementReceiptHistory) == 0 {
			return fmt.Errorf("lot %q has incomplete generation-one replacement metadata", id)
		}
		if lot.Quarantined > 0 || lot.ReplacementAttributed > 0 {
			if _, invalidated := state.InvalidatedReceipts[lot.ReceiptDigest]; !invalidated {
				return fmt.Errorf(
					"lot %q has quarantined or replacement capacity without source receipt invalidation",
					id,
				)
			}
		}
		if activeClaim == -1 {
			if lot.ReplacementReceiptDigest != "" ||
				len(lot.ReplacementControllers) != 0 {
				return fmt.Errorf("lot %q retains current replacement metadata without live capacity", id)
			}
		} else {
			if activeClaim != len(lot.ReplacementClaims)-1 {
				return fmt.Errorf("lot %q live replacement claim is not last in history", id)
			}
			claim := lot.ReplacementClaims[activeClaim]
			if lot.ReplacementReceiptDigest != claim.ReceiptDigest ||
				!equalStringSlices(lot.ReplacementControllers, claim.Controllers) {
				return fmt.Errorf("lot %q current replacement metadata does not match its live claim", id)
			}
		}
		if state.CurrentEpoch >= lot.Deadline &&
			lot.OriginalLive+lot.Quarantined+lot.ReplacementLive > 0 {
			return fmt.Errorf("lot %q retains nonterminal capacity after its deadline", id)
		}
		parts := []uint64{
			lot.Funded,
			lot.OriginalLive,
			lot.Quarantined,
			lot.ReplacementLive,
			lot.Extinguished,
		}
		var partition uint64
		var err error
		for _, part := range parts {
			partition, err = checkedAddUint64(partition, part)
			if err != nil {
				return err
			}
		}
		if partition != lot.Amount {
			return fmt.Errorf("lot %q partition %d does not equal amount %d", id, partition, lot.Amount)
		}
		if lot.ReplacementLive > 0 && lot.ReplacementGeneration != 1 {
			return fmt.Errorf("lot %q has replacement capacity without generation one", id)
		}
		accrued, err = checkedAddUint64(accrued, lot.Amount)
		if err != nil {
			return err
		}
		funded, err = checkedAddUint64(funded, lot.Funded)
		if err != nil {
			return err
		}
		extinguished, err = checkedAddUint64(extinguished, lot.Extinguished)
		if err != nil {
			return err
		}
		live, err = checkedAddUint64(live, l.lotLive(lot))
		if err != nil {
			return err
		}
		quarantined, err = checkedAddUint64(quarantined, lot.Quarantined)
		if err != nil {
			return err
		}
		replacementUsed, err = checkedAddUint64(replacementUsed, lot.ReplacementAttributed)
		if err != nil {
			return err
		}
	}
	for receipt := range state.InvalidatedReceipts {
		found := false
		for _, lot := range state.Lots {
			if lot.ReceiptDigest == receipt {
				found = true
				if lot.OriginalLive > 0 {
					return fmt.Errorf("invalidated original receipt %q still has live capacity", receipt)
				}
			}
			for _, claim := range lot.ReplacementClaims {
				if claim.ReceiptDigest != receipt {
					continue
				}
				found = true
				if claim.Live > 0 {
					return fmt.Errorf("invalidated replacement receipt %q still has live capacity", receipt)
				}
				break
			}
		}
		if !found {
			return fmt.Errorf("invalidated receipt %q is not represented by a lot", receipt)
		}
	}
	if accrued != state.Accrued || funded != state.Funded || extinguished != state.Extinguished {
		return errors.New("shadow ledger aggregate counters do not match lots")
	}
	if replacementUsed != state.ReplacementUsed {
		return errors.New("shadow ledger replacement counter does not match lots")
	}
	partition, err := checkedAddUint64(state.Funded, live)
	if err != nil {
		return err
	}
	partition, err = checkedAddUint64(partition, quarantined)
	if err != nil {
		return err
	}
	partition, err = checkedAddUint64(partition, state.Extinguished)
	if err != nil {
		return err
	}
	if partition != state.Accrued {
		return errors.New("A != Z + L + Q + X")
	}
	replacementExposure, err := checkedAddUint64(state.ReplacementUsed, quarantined)
	if err != nil {
		return err
	}
	if replacementExposure > l.config.LifetimeReplacementCap {
		return errors.New("R + Q exceeds lifetime replacement cap")
	}
	supported := state.CleanSupportTarget
	if supported > state.Accrued {
		supported = state.Accrued
	}
	var liveHeadroom uint64
	if supported > state.Funded {
		liveHeadroom = supported - state.Funded
	}
	if live > liveHeadroom {
		return fmt.Errorf("live capacity %d exceeds clean-support headroom %d", live, liveHeadroom)
	}
	return nil
}

func (l *ShadowCapacityLedger) prepareTimeLocked(now uint64) (shadowLedgerState, error) {
	if now < l.state.CurrentEpoch {
		return shadowLedgerState{}, errors.New("shadow ledger epoch cannot move backwards")
	}
	state := cloneShadowState(l.state)
	state.CurrentEpoch = now
	expired := false
	for id, lot := range state.Lots {
		if now < lot.Deadline {
			continue
		}
		expiring := lot.OriginalLive + lot.ReplacementLive + lot.Quarantined
		if expiring == 0 {
			continue
		}
		lot.Extinguished += expiring
		lot.OriginalLive = 0
		for index, claim := range lot.ReplacementClaims {
			if claim.Live == 0 {
				continue
			}
			claim.Extinguished += claim.Live
			claim.Live = 0
			lot.ReplacementClaims[index] = claim
		}
		lot.ReplacementLive = 0
		lot.ReplacementReceiptDigest = ""
		lot.ReplacementControllers = nil
		lot.Quarantined = 0
		state.Extinguished += expiring
		state.Lots[id] = lot
		expired = true
	}
	if err := l.validateState(state); err != nil {
		return shadowLedgerState{}, err
	}
	// Expiry is a deterministic clock transition, not part of the submitted
	// event. Commit it even when the event that arrived at this boundary is
	// later rejected. Without an expiry, a rejected event mutates nothing.
	if expired {
		l.state = cloneShadowState(state)
	}
	return state, nil
}

func (l *ShadowCapacityLedger) beginEventLocked(eventID string, now uint64) (
	shadowLedgerState,
	error,
) {
	state, err := l.prepareTimeLocked(now)
	if err != nil {
		return shadowLedgerState{}, err
	}
	if err := validateShadowString("shadow event ID", eventID); err != nil {
		return shadowLedgerState{}, err
	}
	if _, exists := l.state.ConsumedEventIDs[eventID]; exists {
		return shadowLedgerState{}, fmt.Errorf("shadow event %q was already consumed", eventID)
	}
	return state, nil
}

func (l *ShadowCapacityLedger) commitEventLocked(eventID string, state shadowLedgerState) error {
	state.ConsumedEventIDs[eventID] = struct{}{}
	nextEventCount, err := checkedAddUint64(state.AcceptedEventCount, 1)
	if err != nil {
		return err
	}
	state.AcceptedEventCount = nextEventCount
	if err := l.validateState(state); err != nil {
		return err
	}
	l.state = state
	return nil
}

// Accrue creates one generation-zero live lot and is the only transition that
// can increase A.
func (l *ShadowCapacityLedger) Accrue(eventID string, now uint64, accrual ShadowAccrual) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	state, err := l.beginEventLocked(eventID, now)
	if err != nil {
		return err
	}
	if err := validateShadowString("accrual lot ID", accrual.LotID); err != nil {
		return err
	}
	if err := validateShadowString("accrual receipt digest", accrual.ReceiptDigest); err != nil {
		return err
	}
	if accrual.Amount == 0 {
		return errors.New("accrual amount must be positive")
	}
	if accrual.Deadline <= now {
		return errors.New("accrual deadline must be in the future")
	}
	if _, exists := state.Lots[accrual.LotID]; exists {
		return fmt.Errorf("shadow lot %q already exists", accrual.LotID)
	}
	for _, lot := range state.Lots {
		if lot.ReceiptDigest == accrual.ReceiptDigest {
			return fmt.Errorf("accrual receipt %q was already used", accrual.ReceiptDigest)
		}
		for _, claim := range lot.ReplacementClaims {
			if claim.ReceiptDigest == accrual.ReceiptDigest {
				return fmt.Errorf("accrual receipt %q was already used", accrual.ReceiptDigest)
			}
		}
	}
	for name, values := range map[string][]string{
		"beneficiary controllers": accrual.BeneficiaryControllers,
		"dependency controllers":  accrual.DependencyControllers,
		"evaluator controllers":   accrual.EvaluatorControllers,
	} {
		if err := validateSortedUniqueStrings(name, values, name != "beneficiary controllers"); err != nil {
			return err
		}
	}
	if err := l.validateEffectiveControllerClosures(
		state,
		accrual.BeneficiaryControllers,
		accrual.DependencyControllers,
		accrual.EvaluatorControllers,
	); err != nil {
		return err
	}
	for _, controller := range shadowLotControllers(ShadowCapacityLot{
		BeneficiaryControllers: accrual.BeneficiaryControllers,
		DependencyControllers:  accrual.DependencyControllers,
		EvaluatorControllers:   accrual.EvaluatorControllers,
	}) {
		if l.groupExcluded(state, controller) {
			return fmt.Errorf("accrual controller %q intersects the excluded closure", controller)
		}
	}
	nextAccrued, err := checkedAddUint64(state.Accrued, accrual.Amount)
	if err != nil {
		return err
	}
	if nextAccrued > l.config.LifetimeCap {
		return errors.New("accrual would exceed lifetime cap")
	}
	lot := ShadowCapacityLot{
		ID:                     accrual.LotID,
		ReceiptDigest:          accrual.ReceiptDigest,
		Amount:                 accrual.Amount,
		OriginalLive:           accrual.Amount,
		Deadline:               accrual.Deadline,
		BeneficiaryControllers: append([]string(nil), accrual.BeneficiaryControllers...),
		DependencyControllers:  append([]string(nil), accrual.DependencyControllers...),
		EvaluatorControllers:   append([]string(nil), accrual.EvaluatorControllers...),
	}
	state.Lots[lot.ID] = lot
	state.Accrued = nextAccrued
	state.CleanSupportTarget = nextAccrued
	return l.commitEventLocked(eventID, state)
}

func sortedLiveLotIDs(state shadowLedgerState) []string {
	ids := make([]string, 0, len(state.Lots))
	for id, lot := range state.Lots {
		if lot.OriginalLive+lot.ReplacementLive > 0 {
			ids = append(ids, id)
		}
	}
	sort.Slice(ids, func(left, right int) bool {
		leftLot := state.Lots[ids[left]]
		rightLot := state.Lots[ids[right]]
		if leftLot.Deadline != rightLot.Deadline {
			return leftLot.Deadline < rightLot.Deadline
		}
		return ids[left] < ids[right]
	})
	return ids
}

// Fund deterministically drains the root's aggregate live backlog by earliest
// inherited deadline and immutable lot ID. It never creates one allocation row
// per artifact or replacement.
func (l *ShadowCapacityLedger) Fund(eventID string, now, budget uint64) (uint64, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	state, err := l.beginEventLocked(eventID, now)
	if err != nil {
		return 0, err
	}
	if budget == 0 {
		return 0, errors.New("shadow funding budget must be positive")
	}
	for _, lot := range state.Lots {
		if lot.OriginalLive > 0 {
			if err := l.validateEffectiveControllerClosures(
				state,
				lot.BeneficiaryControllers,
				lot.DependencyControllers,
				lot.EvaluatorControllers,
			); err != nil {
				return 0, fmt.Errorf("lot %q no longer has disjoint original controllers: %w", lot.ID, err)
			}
			for _, controller := range shadowLotControllers(lot) {
				if l.groupExcluded(state, controller) {
					return 0, fmt.Errorf("lot %q original controller %q is excluded", lot.ID, controller)
				}
			}
		}
		if lot.ReplacementLive > 0 {
			if err := l.validateEffectiveControllerClosures(
				state,
				lot.ReplacementControllers,
			); err != nil {
				return 0, fmt.Errorf("lot %q no longer has disjoint replacement controllers: %w", lot.ID, err)
			}
			for _, controller := range lot.ReplacementControllers {
				if l.groupExcluded(state, controller) {
					return 0, fmt.Errorf("lot %q replacement controller %q is excluded", lot.ID, controller)
				}
			}
		}
	}
	remaining := budget
	var funded uint64
	for _, id := range sortedLiveLotIDs(state) {
		if remaining == 0 {
			break
		}
		lot := state.Lots[id]
		if lot.OriginalLive > 0 {
			amount := min(remaining, lot.OriginalLive)
			lot.OriginalLive -= amount
			lot.Funded += amount
			state.Funded += amount
			funded += amount
			remaining -= amount
		}
		if remaining > 0 && lot.ReplacementLive > 0 {
			amount := min(remaining, lot.ReplacementLive)
			activeClaim := -1
			for index, claim := range lot.ReplacementClaims {
				if claim.Live > 0 {
					activeClaim = index
					break
				}
			}
			if activeClaim == -1 {
				return 0, fmt.Errorf("lot %q has replacement capacity without a live claim", lot.ID)
			}
			claim := lot.ReplacementClaims[activeClaim]
			if amount > claim.Live {
				return 0, fmt.Errorf("lot %q live claim is smaller than replacement capacity", lot.ID)
			}
			claim.Live -= amount
			claim.Funded += amount
			lot.ReplacementClaims[activeClaim] = claim
			lot.ReplacementLive -= amount
			lot.Funded += amount
			state.Funded += amount
			funded += amount
			remaining -= amount
			if lot.ReplacementLive == 0 {
				lot.ReplacementReceiptDigest = ""
				lot.ReplacementControllers = nil
			}
		}
		state.Lots[id] = lot
	}
	if err := l.commitEventLocked(eventID, state); err != nil {
		return 0, err
	}
	return funded, nil
}

func mergeSortedUnique(groups ...[]string) []string {
	set := make(map[string]struct{})
	for _, group := range groups {
		for _, value := range group {
			set[value] = struct{}{}
		}
	}
	result := make([]string, 0, len(set))
	for value := range set {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func replacementClaimIndex(lot ShadowCapacityLot, receipt string) int {
	for index, claim := range lot.ReplacementClaims {
		if claim.ReceiptDigest == receipt {
			return index
		}
	}
	return -1
}

// FinalizeInvalidation moves original live capacity to quarantine up to the
// remaining lifetime replacement bound and extinguishes overflow. A failed
// generation-one replacement goes directly to X.
func (l *ShadowCapacityLedger) FinalizeInvalidation(
	eventID string,
	now uint64,
	invalidation ShadowInvalidation,
) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	state, err := l.beginEventLocked(eventID, now)
	if err != nil {
		return err
	}
	if err := validateShadowString(
		"final invalidation decision",
		invalidation.DecisionID,
	); err != nil {
		return err
	}
	if err := validateShadowString(
		"final invalidation target receipt",
		invalidation.TargetReceiptDigest,
	); err != nil {
		return err
	}
	if _, consumed := state.ConsumedDecisionIDs[invalidation.DecisionID]; consumed {
		return fmt.Errorf("invalidation decision %q was already consumed", invalidation.DecisionID)
	}
	if _, invalidated := state.InvalidatedReceipts[invalidation.TargetReceiptDigest]; invalidated {
		return fmt.Errorf("receipt %q was already invalidated", invalidation.TargetReceiptDigest)
	}
	if !invalidation.Final ||
		!invalidation.AssignmentAfterFreeze ||
		!invalidation.OutcomeIndependentPay {
		return errors.New("raw, pre-assignment, or outcome-contingent challenges cannot change accounting")
	}
	if invalidation.CleanSupportTarget > state.Accrued {
		return errors.New("invalidation clean-support target exceeds accrued capacity")
	}
	for name, values := range map[string][]string{
		"culpable controllers":    invalidation.CulpableControllers,
		"challenger controllers":  invalidation.ChallengerControllers,
		"adjudicator controllers": invalidation.AdjudicatorControllers,
		"organization roots":      invalidation.OrganizationRoots,
	} {
		allowEmpty := name == "culpable controllers" || name == "challenger controllers"
		if err := validateSortedUniqueStrings(name, values, allowEmpty); err != nil {
			return err
		}
	}
	if len(invalidation.AdjudicatorControllers) < 3 ||
		len(invalidation.OrganizationRoots) < 2 {
		return errors.New("final invalidation requires three adjudicator controllers and two organization roots")
	}
	if err := l.validateEffectiveControllerClosures(
		state,
		invalidation.OrganizationRoots,
	); err != nil {
		return fmt.Errorf("invalidation organization roots are not effectively distinct: %w", err)
	}
	if err := l.validateEffectiveControllerClosures(
		state,
		invalidation.CulpableControllers,
		invalidation.ChallengerControllers,
		invalidation.AdjudicatorControllers,
	); err != nil {
		return fmt.Errorf(
			"culpable, challenger, and adjudicator controller closures must be disjoint: %w",
			err,
		)
	}
	targetIDs := make([]string, 0, len(state.Lots))
	for id, lot := range state.Lots {
		targetsOriginal := lot.ReceiptDigest == invalidation.TargetReceiptDigest
		targetsReplacement := replacementClaimIndex(
			lot,
			invalidation.TargetReceiptDigest,
		) >= 0
		if targetsOriginal || targetsReplacement {
			targetIDs = append(targetIDs, id)
		}
	}
	if len(targetIDs) == 0 {
		return errors.New("final invalidation target receipt is unknown")
	}
	sort.Slice(targetIDs, func(left, right int) bool {
		leftLot := state.Lots[targetIDs[left]]
		rightLot := state.Lots[targetIDs[right]]
		if leftLot.Deadline != rightLot.Deadline {
			return leftLot.Deadline < rightLot.Deadline
		}
		return targetIDs[left] < targetIDs[right]
	})
	for _, id := range targetIDs {
		lot := state.Lots[id]
		targetsOriginal := lot.ReceiptDigest == invalidation.TargetReceiptDigest
		replacementIndex := replacementClaimIndex(
			lot,
			invalidation.TargetReceiptDigest,
		)
		targetsReplacement := replacementIndex >= 0
		targetControllers := shadowLotControllers(lot)
		if targetsReplacement {
			targetControllers = append(
				[]string(nil),
				lot.ReplacementClaims[replacementIndex].Controllers...,
			)
		}
		if l.effectiveControllerClosuresOverlap(
			state,
			targetControllers,
			invalidation.AdjudicatorControllers,
		) || l.effectiveControllerClosuresOverlap(
			state,
			targetControllers,
			invalidation.ChallengerControllers,
		) {
			return errors.New("target, challenger, and adjudicator controller closures must be disjoint")
		}
		exclusions := mergeSortedUnique(
			targetControllers,
			invalidation.CulpableControllers,
			invalidation.ChallengerControllers,
			invalidation.AdjudicatorControllers,
			l.config.InitiallyExcludedControllers,
		)
		for _, controller := range exclusions {
			if _, exists := state.ControllerLinks[controller]; !exists {
				state.ControllerLinks[controller] = controller
			}
			state.ExcludedControllers[controller] = struct{}{}
		}
		if targetsOriginal && lot.OriginalLive > 0 {
			_, quarantined, err := l.totals(state)
			if err != nil {
				return err
			}
			exposure, err := checkedAddUint64(state.ReplacementUsed, quarantined)
			if err != nil {
				return err
			}
			var capacity uint64
			if exposure < l.config.LifetimeReplacementCap {
				capacity = l.config.LifetimeReplacementCap - exposure
			}
			toQuarantine := min(lot.OriginalLive, capacity)
			toExtinguish := lot.OriginalLive - toQuarantine
			lot.OriginalLive = 0
			lot.Quarantined += toQuarantine
			lot.Extinguished += toExtinguish
			state.Extinguished += toExtinguish
		}
		if targetsReplacement {
			claim := lot.ReplacementClaims[replacementIndex]
			toExtinguish := claim.Live
			claim.Live = 0
			claim.Extinguished += toExtinguish
			lot.ReplacementClaims[replacementIndex] = claim
			lot.ReplacementLive -= toExtinguish
			lot.Extinguished += toExtinguish
			state.Extinguished += toExtinguish
			if lot.ReplacementReceiptDigest == invalidation.TargetReceiptDigest {
				lot.ReplacementReceiptDigest = ""
				lot.ReplacementControllers = nil
			}
		}
		state.Lots[id] = lot
	}
	for id, lot := range state.Lots {
		if lot.ReplacementLive == 0 {
			continue
		}
		tainted := false
		for _, controller := range lot.ReplacementControllers {
			if l.groupExcluded(state, controller) {
				tainted = true
				break
			}
		}
		if !tainted {
			continue
		}
		activeIndex := replacementClaimIndex(lot, lot.ReplacementReceiptDigest)
		if activeIndex < 0 {
			return errors.New("active replacement receipt has no claim record")
		}
		claim := lot.ReplacementClaims[activeIndex]
		amount := claim.Live
		claim.Live = 0
		claim.Extinguished += amount
		lot.ReplacementClaims[activeIndex] = claim
		lot.ReplacementLive -= amount
		lot.Extinguished += amount
		lot.ReplacementReceiptDigest = ""
		lot.ReplacementControllers = nil
		state.Extinguished += amount
		state.Lots[id] = lot
	}
	state.CleanSupportTarget = invalidation.CleanSupportTarget
	state.ConsumedDecisionIDs[invalidation.DecisionID] = struct{}{}
	state.InvalidatedReceipts[invalidation.TargetReceiptDigest] = struct{}{}
	return l.commitEventLocked(eventID, state)
}

func sortedQuarantinedLotIDs(state shadowLedgerState) []string {
	ids := make([]string, 0, len(state.Lots))
	for id, lot := range state.Lots {
		if lot.Quarantined > 0 {
			ids = append(ids, id)
		}
	}
	sort.Slice(ids, func(left, right int) bool {
		leftLot := state.Lots[ids[left]]
		rightLot := state.Lots[ids[right]]
		if leftLot.Deadline != rightLot.Deadline {
			return leftLot.Deadline < rightLot.Deadline
		}
		return ids[left] < ids[right]
	})
	return ids
}

// Reattribute moves the deterministic maximum supported amount Q→L1. The
// caller cannot choose an amount or lot order.
func (l *ShadowCapacityLedger) Reattribute(
	eventID string,
	now uint64,
	successor ShadowSuccessor,
) (uint64, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	state, err := l.beginEventLocked(eventID, now)
	if err != nil {
		return 0, err
	}
	if err := validateShadowString("successor receipt digest", successor.ReceiptDigest); err != nil {
		return 0, err
	}
	if err := validateShadowString(
		"successor prior receipt digest",
		successor.PriorReceiptDigest,
	); err != nil {
		return 0, err
	}
	if _, invalidated := state.InvalidatedReceipts[successor.PriorReceiptDigest]; !invalidated {
		return 0, errors.New("successor prior receipt has no final invalidation")
	}
	for _, lot := range state.Lots {
		if lot.ReceiptDigest == successor.ReceiptDigest {
			return 0, errors.New("successor receipt was already used")
		}
		for _, claim := range lot.ReplacementClaims {
			if claim.ReceiptDigest == successor.ReceiptDigest {
				return 0, errors.New("successor receipt was already used")
			}
		}
	}
	if successor.SupportTarget > state.Accrued {
		return 0, errors.New("successor support target exceeds accrued capacity")
	}
	if !successor.PolicyPassed {
		return 0, errors.New("successor did not pass the snapshotted policy")
	}
	for _, lot := range state.Lots {
		if lot.ReplacementLive > 0 {
			return 0, errors.New("an active replacement must fund, expire, or be invalidated before another successor")
		}
	}
	for name, values := range map[string][]string{
		"successor beneficiary controllers": successor.BeneficiaryControllers,
		"successor dependency controllers":  successor.DependencyControllers,
		"successor evaluator controllers":   successor.EvaluatorControllers,
	} {
		if err := validateSortedUniqueStrings(name, values, name != "successor beneficiary controllers"); err != nil {
			return 0, err
		}
	}
	if err := l.validateEffectiveControllerClosures(
		state,
		successor.BeneficiaryControllers,
		successor.DependencyControllers,
		successor.EvaluatorControllers,
	); err != nil {
		return 0, err
	}
	controllers := successorControllers(successor)
	for _, controller := range controllers {
		if l.groupExcluded(state, controller) {
			return 0, fmt.Errorf("successor controller %q intersects the excluded closure", controller)
		}
	}
	live, _, err := l.totals(state)
	if err != nil {
		return 0, err
	}
	supported := min(state.Accrued, successor.SupportTarget)
	var headroom uint64
	if supported > state.Funded {
		headroom = supported - state.Funded
	}
	if live >= headroom {
		return 0, errors.New("successor creates no supported live-capacity headroom")
	}
	headroom -= live
	var replacementRemaining uint64
	if state.ReplacementUsed < l.config.LifetimeReplacementCap {
		replacementRemaining = l.config.LifetimeReplacementCap - state.ReplacementUsed
	}
	remaining := min(headroom, replacementRemaining)
	if remaining == 0 {
		return 0, errors.New("successor has no remaining replacement capacity")
	}
	var moved uint64
	for _, id := range sortedQuarantinedLotIDs(state) {
		if remaining == 0 {
			break
		}
		lot := state.Lots[id]
		if lot.ReceiptDigest != successor.PriorReceiptDigest {
			continue
		}
		amount := min(remaining, lot.Quarantined)
		lot.Quarantined -= amount
		lot.ReplacementLive += amount
		lot.ReplacementGeneration = 1
		lot.ReplacementAttributed += amount
		lot.ReplacementReceiptDigest = successor.ReceiptDigest
		lot.ReplacementReceiptHistory = append(
			lot.ReplacementReceiptHistory,
			successor.ReceiptDigest,
		)
		lot.ReplacementControllers = append([]string(nil), controllers...)
		lot.ReplacementClaims = append(
			lot.ReplacementClaims,
			ShadowReplacementClaim{
				ReceiptDigest: successor.ReceiptDigest,
				Controllers:   append([]string(nil), controllers...),
				Attributed:    amount,
				Live:          amount,
			},
		)
		state.Lots[id] = lot
		state.ReplacementUsed += amount
		moved += amount
		remaining -= amount
	}
	if moved == 0 {
		return 0, errors.New("successor did not match any unexpired quarantined source lot")
	}
	state.CleanSupportTarget = max(state.CleanSupportTarget, successor.SupportTarget)
	if err := l.commitEventLocked(eventID, state); err != nil {
		return 0, err
	}
	return moved, nil
}

// LinkControllers records a monotone controller merge. If any member of the
// merged closure was excluded, all become excluded and any still-live
// replacement they control is extinguished. There is intentionally no split
// operation.
func (l *ShadowCapacityLedger) LinkControllers(
	eventID string,
	now uint64,
	controllers []string,
) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	state, err := l.beginEventLocked(eventID, now)
	if err != nil {
		return err
	}
	if err := validateSortedUniqueStrings("linked controllers", controllers, false); err != nil {
		return err
	}
	if len(controllers) < 2 {
		return errors.New("controller link requires at least two controllers")
	}
	roots := make([]string, len(controllers))
	groupWasExcluded := false
	for index, controller := range controllers {
		if _, exists := state.ControllerLinks[controller]; !exists {
			state.ControllerLinks[controller] = controller
		}
		roots[index] = l.controllerRoot(state, controller)
		groupWasExcluded = groupWasExcluded || l.groupExcluded(state, controller)
	}
	sort.Strings(roots)
	canonicalRoot := roots[0]
	for controller, root := range state.ControllerLinks {
		for _, linkedRoot := range roots {
			if l.controllerRoot(state, root) == linkedRoot {
				state.ControllerLinks[controller] = canonicalRoot
				break
			}
		}
	}
	for _, controller := range controllers {
		state.ControllerLinks[controller] = canonicalRoot
	}
	state.ControllerLinks[canonicalRoot] = canonicalRoot
	if groupWasExcluded {
		for controller := range state.ControllerLinks {
			if l.controllerRoot(state, controller) == canonicalRoot {
				state.ExcludedControllers[controller] = struct{}{}
			}
		}
	}
	for id, lot := range state.Lots {
		if lot.ReplacementLive == 0 {
			continue
		}
		tainted := false
		for _, controller := range lot.ReplacementControllers {
			if l.groupExcluded(state, controller) {
				tainted = true
				break
			}
		}
		if tainted {
			amount := lot.ReplacementLive
			activeIndex := replacementClaimIndex(lot, lot.ReplacementReceiptDigest)
			if activeIndex < 0 {
				return errors.New("active replacement receipt has no claim record")
			}
			claim := lot.ReplacementClaims[activeIndex]
			if claim.Live != amount {
				return errors.New("active replacement claim and lot capacity differ")
			}
			claim.Live = 0
			claim.Extinguished += amount
			lot.ReplacementClaims[activeIndex] = claim
			lot.ReplacementLive = 0
			lot.Extinguished += amount
			lot.ReplacementReceiptDigest = ""
			lot.ReplacementControllers = nil
			state.Extinguished += amount
			state.Lots[id] = lot
		}
	}
	return l.commitEventLocked(eventID, state)
}

// ObserveChallenge records no economic state and exists to make the
// challenge/quarantine separation executable.
func (l *ShadowCapacityLedger) ObserveChallenge(eventID string, now uint64) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	state, err := l.beginEventLocked(eventID, now)
	if err != nil {
		return err
	}
	return l.commitEventLocked(eventID, state)
}

func shadowSortedKeys(input map[string]struct{}) []string {
	result := make([]string, 0, len(input))
	for value := range input {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func (l *ShadowCapacityLedger) snapshotLocked() ShadowLedgerSnapshot {
	lots := make([]ShadowCapacityLot, 0, len(l.state.Lots))
	for _, lot := range l.state.Lots {
		lots = append(lots, cloneShadowLot(lot))
	}
	sort.Slice(lots, func(left, right int) bool {
		return lots[left].ID < lots[right].ID
	})
	live, quarantined, _ := l.totals(l.state)
	config := l.config
	config.InitiallyExcludedControllers = append(
		[]string(nil),
		l.config.InitiallyExcludedControllers...,
	)
	snapshot := ShadowLedgerSnapshot{
		ArithmeticVersion:   shadowLedgerArithmeticVersion,
		Config:              config,
		CurrentEpoch:        l.state.CurrentEpoch,
		Accrued:             l.state.Accrued,
		Funded:              l.state.Funded,
		Live:                live,
		Quarantined:         quarantined,
		Extinguished:        l.state.Extinguished,
		ReplacementUsed:     l.state.ReplacementUsed,
		CleanSupportTarget:  l.state.CleanSupportTarget,
		Lots:                lots,
		AcceptedEventCount:  l.state.AcceptedEventCount,
		ConsumedEventIDs:    shadowSortedKeys(l.state.ConsumedEventIDs),
		ConsumedDecisionIDs: shadowSortedKeys(l.state.ConsumedDecisionIDs),
		InvalidatedReceipts: shadowSortedKeys(l.state.InvalidatedReceipts),
		ControllerLinks:     cloneStringMap(l.state.ControllerLinks),
		ExcludedControllers: shadowSortedKeys(l.state.ExcludedControllers),
	}
	commitment, err := shadowSnapshotCommitment(snapshot)
	if err != nil {
		panic(fmt.Sprintf("encode internal shadow snapshot: %v", err))
	}
	snapshot.StateCommitment = commitment
	return snapshot
}

func (l *ShadowCapacityLedger) Snapshot() ShadowLedgerSnapshot {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.snapshotLocked()
}

func shadowSnapshotCommitment(snapshot ShadowLedgerSnapshot) (string, error) {
	snapshot.StateCommitment = ""
	encoded, err := json.Marshal(snapshot)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", sha256.Sum256(encoded)), nil
}

func RestoreShadowCapacityLedger(
	snapshot ShadowLedgerSnapshot,
	trustedStateCommitment string,
) (*ShadowCapacityLedger, error) {
	if err := validateShadowCommitment(
		"trusted shadow state commitment",
		trustedStateCommitment,
	); err != nil {
		return nil, err
	}
	if err := validateShadowCommitment(
		"snapshot state commitment",
		snapshot.StateCommitment,
	); err != nil {
		return nil, err
	}
	if snapshot.StateCommitment != trustedStateCommitment {
		return nil, errors.New("shadow snapshot does not match the trusted state commitment")
	}
	computedCommitment, err := shadowSnapshotCommitment(snapshot)
	if err != nil {
		return nil, fmt.Errorf("encode shadow snapshot commitment: %w", err)
	}
	if computedCommitment != trustedStateCommitment {
		return nil, errors.New("shadow snapshot content does not match its trusted state commitment")
	}
	if snapshot.ArithmeticVersion != shadowLedgerArithmeticVersion {
		return nil, fmt.Errorf("unsupported shadow arithmetic version %q", snapshot.ArithmeticVersion)
	}
	ledger, err := NewShadowCapacityLedger(snapshot.Config)
	if err != nil {
		return nil, err
	}
	state := shadowLedgerState{
		CurrentEpoch:        snapshot.CurrentEpoch,
		Accrued:             snapshot.Accrued,
		Funded:              snapshot.Funded,
		Extinguished:        snapshot.Extinguished,
		ReplacementUsed:     snapshot.ReplacementUsed,
		CleanSupportTarget:  snapshot.CleanSupportTarget,
		Lots:                make(map[string]ShadowCapacityLot, len(snapshot.Lots)),
		AcceptedEventCount:  snapshot.AcceptedEventCount,
		ConsumedEventIDs:    make(map[string]struct{}, len(snapshot.ConsumedEventIDs)),
		ConsumedDecisionIDs: make(map[string]struct{}, len(snapshot.ConsumedDecisionIDs)),
		InvalidatedReceipts: make(map[string]struct{}, len(snapshot.InvalidatedReceipts)),
		ControllerLinks:     cloneStringMap(snapshot.ControllerLinks),
		ExcludedControllers: make(map[string]struct{}, len(snapshot.ExcludedControllers)),
	}
	for _, lot := range snapshot.Lots {
		if _, duplicate := state.Lots[lot.ID]; duplicate {
			return nil, fmt.Errorf("snapshot repeats lot %q", lot.ID)
		}
		state.Lots[lot.ID] = cloneShadowLot(lot)
	}
	for _, eventID := range snapshot.ConsumedEventIDs {
		if eventID == "" {
			return nil, errors.New("snapshot contains an empty event ID")
		}
		if _, duplicate := state.ConsumedEventIDs[eventID]; duplicate {
			return nil, fmt.Errorf("snapshot repeats event %q", eventID)
		}
		state.ConsumedEventIDs[eventID] = struct{}{}
	}
	for _, decisionID := range snapshot.ConsumedDecisionIDs {
		if decisionID == "" {
			return nil, errors.New("snapshot contains an empty decision ID")
		}
		if _, duplicate := state.ConsumedDecisionIDs[decisionID]; duplicate {
			return nil, fmt.Errorf("snapshot repeats decision %q", decisionID)
		}
		state.ConsumedDecisionIDs[decisionID] = struct{}{}
	}
	for _, receipt := range snapshot.InvalidatedReceipts {
		if receipt == "" {
			return nil, errors.New("snapshot contains an empty invalidated receipt")
		}
		if _, duplicate := state.InvalidatedReceipts[receipt]; duplicate {
			return nil, fmt.Errorf("snapshot repeats invalidated receipt %q", receipt)
		}
		state.InvalidatedReceipts[receipt] = struct{}{}
	}
	for _, controller := range snapshot.ExcludedControllers {
		if controller == "" {
			return nil, errors.New("snapshot contains an empty excluded controller")
		}
		state.ExcludedControllers[controller] = struct{}{}
	}
	if err := ledger.validateState(state); err != nil {
		return nil, err
	}
	live, quarantined, err := ledger.totals(state)
	if err != nil {
		return nil, err
	}
	if live != snapshot.Live || quarantined != snapshot.Quarantined {
		return nil, errors.New("snapshot aggregate live or quarantined counters do not match lots")
	}
	ledger.state = state
	if canonicalCommitment := ledger.snapshotLocked().StateCommitment; canonicalCommitment != trustedStateCommitment {
		return nil, errors.New("shadow snapshot is not in canonical commitment representation")
	}
	return ledger, nil
}
