// Package main contains a deterministic, non-consensus simulator for the
// constructive-intelligence reward design.
package main

import (
	"errors"
	"fmt"
	"math"
	"sort"
)

const (
	floatTolerance          = 1e-9
	amountAbsoluteFloor     = 1e-12
	amountULPMultiplier     = 4096
	minSimulationAmount     = 1e-6
	maxSimulationAmount     = 1e9
	maxClustersPerEpoch     = 1024
	maxAggregateAmount      = maxSimulationAmount * maxClustersPerEpoch
	maxSignalsPerPanel      = 256
	maxCreditsPerCluster    = 256
	maxPowerControllers     = 256
	minScoreEpsilon         = 1e-6
	maxScoreEpsilon         = 0.10
	engineArithmeticVersion = "constructive-rewards-float-v3"
)

var requiredPowerSurfaceNames = []string{
	"consensus-block-ordering",
	"stake-governance",
	"epistemic-review",
	"scorer-registry-authorship",
	"semantic-cluster-adjudication",
	"controller-attestation",
	"proof-data-trust",
	"proposal-authorship",
	"treasury-authorization",
	"reward-flow",
	"model-families",
	"infrastructure",
}

// amountTolerance is for exploratory monetary arithmetic only. It scales with
// representable float spacing and is never a substitute for consensus
// fixed-point arithmetic.
func amountTolerance(values ...float64) float64 {
	scale := 0.0
	for _, value := range values {
		scale = math.Max(scale, math.Abs(value))
	}
	if scale == 0 {
		return amountAbsoluteFloor
	}
	ulp := math.Nextafter(scale, math.Inf(1)) - scale
	return math.Max(amountAbsoluteFloor, amountULPMultiplier*ulp)
}

func amountsEqual(a, b float64) bool {
	return math.Abs(a-b) <= amountTolerance(a, b)
}

func clonePowerSurfaces(input map[string]map[string]float64) map[string]map[string]float64 {
	result := make(map[string]map[string]float64, len(input))
	for surface, powers := range input {
		result[surface] = cloneAmounts(powers)
	}
	return result
}

func cloneThresholds(input map[string]float64) map[string]float64 {
	return cloneAmounts(input)
}

func balancedPowerSurfaces(prefix string) map[string]map[string]float64 {
	result := make(map[string]map[string]float64, len(requiredPowerSurfaceNames))
	for _, surface := range requiredPowerSurfaceNames {
		controllers := make(map[string]float64, 4)
		for i := 0; i < 4; i++ {
			controllers[fmt.Sprintf("%s-%s-%d", prefix, surface, i)] = 1
		}
		result[surface] = controllers
	}
	return result
}

func defaultPowerThresholds() map[string]float64 {
	result := make(map[string]float64, len(requiredPowerSurfaceNames))
	for _, surface := range requiredPowerSurfaceNames {
		result[surface] = 0.34
	}
	return result
}

func cloneParams(input Params) Params {
	result := input
	result.PowerSurfaces = clonePowerSurfaces(input.PowerSurfaces)
	result.PowerCoalitionThresholds = cloneThresholds(input.PowerCoalitionThresholds)
	return result
}

// ScoreWeights separates compensable evidence dimensions. Provenance,
// validity, safety, quorum, and system power are gates: strength elsewhere
// cannot repair their failure.
type ScoreWeights struct {
	Marginal    float64 `json:"marginal"`
	Replication float64 `json:"replication"`
	Use         float64 `json:"use"`
}

// Params are illustrative simulation parameters, not proposed genesis values.
type Params struct {
	Budget                     float64                       `json:"budget"`
	Alpha                      float64                       `json:"alpha"`
	ControllerCapShare         float64                       `json:"controller_cap_share"`
	MinProvenance              float64                       `json:"min_provenance"`
	MinValidity                float64                       `json:"min_validity"`
	MinSafety                  float64                       `json:"min_safety"`
	ReplicationTarget          float64                       `json:"replication_target"`
	MinSubmittedSignals        int                           `json:"min_submitted_signals"`
	MinControllerSignals       int                           `json:"min_controller_signals"`
	MinEffectiveSignals        float64                       `json:"min_effective_signals"`
	SameFamilyCorrelationFloor float64                       `json:"same_family_correlation_floor"`
	PowerEffectiveTarget       float64                       `json:"power_effective_target"`
	MinPowerEffective          float64                       `json:"min_power_effective"`
	MinPowerNakamoto           int                           `json:"min_power_nakamoto"`
	PowerSurfaces              map[string]map[string]float64 `json:"power_surfaces"`
	PowerCoalitionThresholds   map[string]float64            `json:"power_coalition_thresholds"`
	ScoreEpsilon               float64                       `json:"score_epsilon"`
	Weights                    ScoreWeights                  `json:"weights"`
}

// DefaultParams returns a deliberately conservative, illustrative parameter
// set. No value here is suitable for consensus without empirical calibration.
func DefaultParams() Params {
	return Params{
		Budget:                     1_000,
		Alpha:                      0.65,
		ControllerCapShare:         0.20,
		MinProvenance:              0.80,
		MinValidity:                0.95,
		MinSafety:                  0.70,
		ReplicationTarget:          5,
		MinSubmittedSignals:        2,
		MinControllerSignals:       2,
		MinEffectiveSignals:        1.5,
		SameFamilyCorrelationFloor: 0.75,
		PowerEffectiveTarget:       4,
		MinPowerEffective:          2,
		MinPowerNakamoto:           2,
		PowerSurfaces:              balancedPowerSurfaces("policy"),
		PowerCoalitionThresholds:   defaultPowerThresholds(),
		ScoreEpsilon:               0.01,
		Weights: ScoreWeights{
			Marginal:    0.40,
			Replication: 0.30,
			Use:         0.30,
		},
	}
}

// Validate rejects values that could make a sweep misleading or unstable.
func (p Params) Validate() error {
	values := []struct {
		name  string
		value float64
	}{
		{"budget", p.Budget},
		{"alpha", p.Alpha},
		{"controller cap share", p.ControllerCapShare},
		{"minimum provenance", p.MinProvenance},
		{"minimum validity", p.MinValidity},
		{"minimum safety", p.MinSafety},
		{"replication target", p.ReplicationTarget},
		{"minimum effective signals", p.MinEffectiveSignals},
		{"same-family correlation floor", p.SameFamilyCorrelationFloor},
		{"power effective target", p.PowerEffectiveTarget},
		{"minimum power effective count", p.MinPowerEffective},
		{"score epsilon", p.ScoreEpsilon},
		{"marginal weight", p.Weights.Marginal},
		{"replication weight", p.Weights.Replication},
		{"use weight", p.Weights.Use},
	}
	for _, item := range values {
		if math.IsNaN(item.value) || math.IsInf(item.value, 0) {
			return fmt.Errorf("%s must be finite", item.name)
		}
	}
	if p.Budget < minSimulationAmount || p.Budget > maxSimulationAmount {
		return fmt.Errorf(
			"budget must be in [%.6g, %.0f]",
			minSimulationAmount,
			maxSimulationAmount,
		)
	}
	if p.Alpha <= 0 || p.Alpha > 1 {
		return errors.New("alpha must be in (0, 1]")
	}
	if p.ControllerCapShare <= 0 || p.ControllerCapShare > 1 {
		return errors.New("controller cap share must be in (0, 1]")
	}
	unitValues := []struct {
		name  string
		value float64
	}{
		{"minimum provenance", p.MinProvenance},
		{"minimum validity", p.MinValidity},
		{"minimum safety", p.MinSafety},
		{"same-family correlation floor", p.SameFamilyCorrelationFloor},
		{"marginal weight", p.Weights.Marginal},
		{"replication weight", p.Weights.Replication},
		{"use weight", p.Weights.Use},
	}
	for _, item := range unitValues {
		if item.value < 0 || item.value > 1 {
			return fmt.Errorf("%s must be in [0, 1]", item.name)
		}
	}
	if p.ReplicationTarget <= 0 {
		return errors.New("replication target must be positive")
	}
	if p.MinSubmittedSignals <= 0 {
		return errors.New("minimum submitted signals must be positive")
	}
	if p.MinControllerSignals <= 0 ||
		p.MinControllerSignals != p.MinSubmittedSignals {
		return errors.New(
			"minimum submitted and controller signals must be equal and positive in the alias-neutral reference model",
		)
	}
	if p.MinEffectiveSignals <= 0 ||
		p.MinEffectiveSignals > float64(p.MinControllerSignals) {
		return errors.New("minimum effective signals must be in (0, minimum controller signals]")
	}
	if p.PowerEffectiveTarget <= 0 {
		return errors.New("power effective target must be positive")
	}
	if p.MinPowerEffective <= 0 || p.MinPowerEffective > p.PowerEffectiveTarget {
		return errors.New("minimum power effective count must be in (0, power effective target]")
	}
	if p.MinPowerNakamoto <= 0 {
		return errors.New("minimum power Nakamoto count must be positive")
	}
	if _, _, err := AuditRequiredPower(
		p.PowerSurfaces,
		p.PowerCoalitionThresholds,
	); err != nil {
		return fmt.Errorf("power policy: %w", err)
	}
	if p.ScoreEpsilon < minScoreEpsilon || p.ScoreEpsilon > maxScoreEpsilon {
		return fmt.Errorf(
			"score epsilon must be in [%.6g, %.2f]",
			minScoreEpsilon,
			maxScoreEpsilon,
		)
	}
	weightSum := p.Weights.Marginal + p.Weights.Replication + p.Weights.Use
	if math.Abs(weightSum-1) > floatTolerance {
		return fmt.Errorf("score weights must sum to 1, got %.12f", weightSum)
	}
	return nil
}

// Signal is one replication or review signal. Controller and Family are
// policy-attested correlation labels, not proposer-selected claims about a
// being's ontology. Every collapsed controller has one unit of epistemic
// weight; raw alias count cannot increase it.
type Signal struct {
	Controller string `json:"controller"`
	Family     string `json:"family,omitempty"`
}

// Evidence is attached to a semantic-equivalence cluster, not to a person or
// raw artifact count. Correlation is an optional symmetric pairwise matrix
// over lexicographically sorted, controller-collapsed signals. Global power
// surfaces belong to the immutable policy snapshot in Params, not a proposal.
type Evidence struct {
	Provenance   float64     `json:"provenance"`
	Validity     float64     `json:"validity"`
	Marginal     float64     `json:"marginal"`
	Use          float64     `json:"use"`
	Safety       float64     `json:"safety"`
	Replications []Signal    `json:"replications,omitempty"`
	Correlation  [][]float64 `json:"controller_correlation,omitempty"`
}

func (e Evidence) validate() error {
	values := []struct {
		name  string
		value float64
	}{
		{"provenance", e.Provenance},
		{"validity", e.Validity},
		{"marginal", e.Marginal},
		{"use", e.Use},
		{"safety", e.Safety},
	}
	for _, item := range values {
		if math.IsNaN(item.value) || math.IsInf(item.value, 0) ||
			item.value < 0 || item.value > 1 {
			return fmt.Errorf("%s must be finite and in [0, 1]", item.name)
		}
	}
	if len(e.Replications) > maxSignalsPerPanel {
		return fmt.Errorf(
			"replication panel exceeds exploratory limit of %d submitted signals",
			maxSignalsPerPanel,
		)
	}
	for i, signal := range e.Replications {
		if signal.Controller == "" {
			return fmt.Errorf("replication %d has no controller label", i)
		}
	}
	return nil
}

func validateCorrelationMatrix(matrix [][]float64, expected int) error {
	if len(matrix) != expected {
		return fmt.Errorf(
			"controller correlation matrix has %d rows, want %d",
			len(matrix),
			expected,
		)
	}
	for i := range matrix {
		if len(matrix[i]) != expected {
			return fmt.Errorf(
				"controller correlation row %d has %d columns, want %d",
				i,
				len(matrix[i]),
				expected,
			)
		}
	}
	for i := range matrix {
		for j, value := range matrix[i] {
			if math.IsNaN(value) || math.IsInf(value, 0) || value < 0 || value > 1 {
				return fmt.Errorf("correlation[%d][%d] must be in [0, 1]", i, j)
			}
			if math.Abs(value-matrix[j][i]) > floatTolerance {
				return fmt.Errorf("correlation matrix is not symmetric at [%d][%d]", i, j)
			}
			if i == j && math.Abs(value-1) > floatTolerance {
				return fmt.Errorf("correlation matrix diagonal[%d] must equal 1", i)
			}
		}
	}
	return validateCorrelationPSD(matrix)
}

// validateCorrelationPSD performs a deterministic Cholesky-style
// positive-semidefinite check. A pairwise table that cannot be a correlation
// matrix is rejected instead of being laundered into an independence count.
func validateCorrelationPSD(matrix [][]float64) error {
	const psdTolerance = 1e-10
	n := len(matrix)
	lower := make([][]float64, n)
	for i := range lower {
		lower[i] = make([]float64, n)
	}
	for i := 0; i < n; i++ {
		for j := 0; j <= i; j++ {
			residual := matrix[i][j]
			for k := 0; k < j; k++ {
				residual -= lower[i][k] * lower[j][k]
			}
			if i == j {
				if residual < -psdTolerance {
					return errors.New("effective correlation matrix must be positive semidefinite")
				}
				if residual < 0 {
					residual = 0
				}
				lower[i][j] = math.Sqrt(residual)
				continue
			}
			if lower[j][j] <= psdTolerance {
				if math.Abs(residual) > psdTolerance {
					return errors.New("effective correlation matrix must be positive semidefinite")
				}
				continue
			}
			lower[i][j] = residual / lower[j][j]
		}
	}
	return nil
}

// collapseSignals gives each controller exactly one unit-weight signal before
// panel arithmetic. The optional correlation matrix is already indexed by the
// lexicographically sorted controller set, so adding aliases cannot change its
// dimension, controller weight, or entries. Family labels must be consistent
// within one controller.
func collapseSignals(signals []Signal, correlation [][]float64, sameFamilyFloor float64) ([]Signal, [][]float64, error) {
	e := Evidence{Replications: signals}
	if err := e.validate(); err != nil {
		return nil, nil, err
	}
	if math.IsNaN(sameFamilyFloor) || math.IsInf(sameFamilyFloor, 0) ||
		sameFamilyFloor < 0 || sameFamilyFloor > 1 {
		return nil, nil, errors.New("same-family correlation floor must be finite and in [0, 1]")
	}
	if len(signals) == 0 {
		if len(correlation) != 0 {
			return nil, nil, errors.New("empty panel cannot have a correlation matrix")
		}
		return nil, nil, nil
	}

	families := make(map[string]string)
	for _, signal := range signals {
		current := families[signal.Controller]
		if signal.Family == "" {
			continue
		}
		if current != "" && current != signal.Family {
			return nil, nil, fmt.Errorf(
				"controller %q has conflicting family labels %q and %q",
				signal.Controller,
				current,
				signal.Family,
			)
		}
		families[signal.Controller] = signal.Family
	}
	controllers := make([]string, 0, len(families))
	seenControllers := make(map[string]struct{})
	for _, signal := range signals {
		if _, exists := seenControllers[signal.Controller]; exists {
			continue
		}
		seenControllers[signal.Controller] = struct{}{}
		controllers = append(controllers, signal.Controller)
	}
	for controller := range families {
		if _, exists := seenControllers[controller]; exists {
			continue
		}
		controllers = append(controllers, controller)
	}
	sort.Strings(controllers)

	collapsed := make([]Signal, len(controllers))
	for i, controller := range controllers {
		collapsed[i] = Signal{
			Controller: controller,
			Family:     families[controller],
		}
	}

	effectiveCorrelation := make([][]float64, len(collapsed))
	if len(correlation) == 0 {
		for i := range collapsed {
			effectiveCorrelation[i] = make([]float64, len(collapsed))
			effectiveCorrelation[i][i] = 1
		}
		for i := 0; i < len(collapsed); i++ {
			for j := i + 1; j < len(collapsed); j++ {
				if collapsed[i].Family != "" &&
					collapsed[i].Family == collapsed[j].Family {
					effectiveCorrelation[i][j] = sameFamilyFloor
					effectiveCorrelation[j][i] = sameFamilyFloor
				}
			}
		}
	} else {
		if err := validateCorrelationMatrix(correlation, len(collapsed)); err != nil {
			return nil, nil, err
		}
		for i := range correlation {
			effectiveCorrelation[i] = append([]float64(nil), correlation[i]...)
		}
		for i := 0; i < len(collapsed); i++ {
			for j := i + 1; j < len(collapsed); j++ {
				if collapsed[i].Family != "" &&
					collapsed[i].Family == collapsed[j].Family &&
					effectiveCorrelation[i][j]+floatTolerance < sameFamilyFloor {
					return nil, nil, fmt.Errorf(
						"correlation[%d][%d] %.12f is below same-family floor %.12f",
						i,
						j,
						effectiveCorrelation[i][j],
						sameFamilyFloor,
					)
				}
			}
		}
	}
	if err := validateCorrelationMatrix(effectiveCorrelation, len(collapsed)); err != nil {
		return nil, nil, err
	}
	return collapsed, effectiveCorrelation, nil
}

// EffectiveIndependentCount computes
//
//	       (sum_i w_i)^2
//	n = ---------------------------
//	    sum_i w_i^2 + sum_i!=j rho_ij w_i w_j
//
// after true controller collapse and conservative family correlation.
func EffectiveIndependentCount(signals []Signal, correlation [][]float64, sameFamilyFloor float64) (float64, error) {
	collapsed, effectiveCorrelation, err := collapseSignals(signals, correlation, sameFamilyFloor)
	if err != nil {
		return 0, err
	}
	if len(collapsed) == 0 {
		return 0, nil
	}

	totalWeight := float64(len(collapsed))
	denominator := float64(len(collapsed))
	for i := 0; i < len(collapsed); i++ {
		for j := i + 1; j < len(collapsed); j++ {
			rho := effectiveCorrelation[i][j]
			denominator += 2 * rho
		}
	}
	if denominator <= 0 {
		return 0, errors.New("effective-count denominator must be positive")
	}
	result := totalWeight * totalWeight / denominator
	if result < 1 {
		return 1, nil
	}
	if result > float64(len(collapsed)) {
		return float64(len(collapsed)), nil
	}
	return result, nil
}

// ScoreDetail exposes the gate and every normalized component for audit.
type ScoreDetail struct {
	GatePassed        bool            `json:"gate_passed"`
	SubmittedSignals  int             `json:"submitted_signals"`
	ControllerSignals int             `json:"controller_signals"`
	EffectiveSignals  float64         `json:"effective_signals"`
	PowerSurfaces     []SurfaceMetric `json:"power_surfaces"`
	WeakestPower      float64         `json:"weakest_power"`
	MinNakamoto       int             `json:"minimum_nakamoto_count"`
	Marginal          float64         `json:"marginal"`
	Replication       float64         `json:"replication"`
	Use               float64         `json:"use"`
	Decentralization  float64         `json:"decentralization"`
	Safety            float64         `json:"safety"`
	Total             float64         `json:"total"`
}

// ScoreEvidence evaluates the cluster's current evidence state. Replication is
// logarithmic and saturates at the target effective independent count.
func ScoreEvidence(e Evidence, p Params) (ScoreDetail, error) {
	if err := p.Validate(); err != nil {
		return ScoreDetail{}, err
	}
	if err := e.validate(); err != nil {
		return ScoreDetail{}, err
	}
	collapsed, _, err := collapseSignals(
		e.Replications,
		e.Correlation,
		p.SameFamilyCorrelationFloor,
	)
	if err != nil {
		return ScoreDetail{}, err
	}
	detail := ScoreDetail{
		SubmittedSignals:  len(e.Replications),
		ControllerSignals: len(collapsed),
		Marginal:          e.Marginal,
		Use:               e.Use,
		Safety:            e.Safety,
	}
	nEff, err := EffectiveIndependentCount(
		e.Replications,
		e.Correlation,
		p.SameFamilyCorrelationFloor,
	)
	if err != nil {
		return ScoreDetail{}, err
	}
	detail.EffectiveSignals = nEff
	if nEff > 0 {
		detail.Replication = math.Min(1, math.Log1p(nEff)/math.Log1p(p.ReplicationTarget))
	}
	powerMetrics, weakestPower, err := AuditRequiredPower(
		p.PowerSurfaces,
		p.PowerCoalitionThresholds,
	)
	if err != nil {
		return ScoreDetail{}, fmt.Errorf("power surfaces: %w", err)
	}
	detail.PowerSurfaces = powerMetrics
	detail.WeakestPower = weakestPower
	detail.MinNakamoto = math.MaxInt
	for _, metric := range powerMetrics {
		if metric.NakamotoCount < detail.MinNakamoto {
			detail.MinNakamoto = metric.NakamotoCount
		}
	}
	detail.Decentralization = math.Min(1, weakestPower/p.PowerEffectiveTarget)
	detail.GatePassed = e.Provenance >= p.MinProvenance &&
		e.Validity >= p.MinValidity &&
		e.Safety >= p.MinSafety &&
		detail.SubmittedSignals >= p.MinSubmittedSignals &&
		detail.ControllerSignals >= p.MinControllerSignals &&
		detail.EffectiveSignals >= p.MinEffectiveSignals &&
		detail.WeakestPower >= p.MinPowerEffective &&
		detail.MinNakamoto >= p.MinPowerNakamoto
	if !detail.GatePassed {
		return detail, nil
	}
	components := []struct {
		value  float64
		weight float64
	}{
		{detail.Marginal, p.Weights.Marginal},
		{detail.Replication, p.Weights.Replication},
		{detail.Use, p.Weights.Use},
	}
	var logProduct float64
	for _, component := range components {
		softened := p.ScoreEpsilon + (1-p.ScoreEpsilon)*component.value
		logProduct += component.weight * math.Log(softened)
	}
	product := math.Exp(logProduct)
	detail.Total = math.Max(0, (product-p.ScoreEpsilon)/(1-p.ScoreEpsilon))
	return detail, nil
}

// ControllerCredit is a dependency-DAG credit share. A controller appears at
// most once per cluster; artifact count never creates extra credit entries.
type ControllerCredit struct {
	Controller string   `json:"controller"`
	Credit     float64  `json:"credit"`
	Roles      []string `json:"roles,omitempty"`
}

// Cluster is one semantic-equivalence class. ArtifactIDs are retained only for
// audit: their count never enters the score or budget formula.
type Cluster struct {
	ID                     string             `json:"id"`
	ArtifactIDs            []string           `json:"artifact_ids,omitempty"`
	Evidence               Evidence           `json:"evidence"`
	PriorHighWater         float64            `json:"prior_high_water"`
	LifetimeCap            float64            `json:"lifetime_cap"`
	FundedToDate           float64            `json:"funded_to_date"`
	ControllerDirectToDate map[string]float64 `json:"controller_direct_to_date,omitempty"`
	Credits                []ControllerCredit `json:"credits"`
}

func (c Cluster) validate() error {
	if c.ID == "" {
		return errors.New("cluster ID is required")
	}
	if c.PriorHighWater < 0 || c.PriorHighWater > 1 ||
		math.IsNaN(c.PriorHighWater) || math.IsInf(c.PriorHighWater, 0) {
		return fmt.Errorf("cluster %q high-water mark must be in [0, 1]", c.ID)
	}
	if c.LifetimeCap < minSimulationAmount || c.LifetimeCap > maxSimulationAmount ||
		math.IsNaN(c.LifetimeCap) || math.IsInf(c.LifetimeCap, 0) {
		return fmt.Errorf(
			"cluster %q lifetime cap must be finite and in [%.6g, %.0f]",
			c.ID,
			minSimulationAmount,
			maxSimulationAmount,
		)
	}
	if c.FundedToDate < 0 || c.FundedToDate > maxSimulationAmount ||
		math.IsNaN(c.FundedToDate) || math.IsInf(c.FundedToDate, 0) {
		return fmt.Errorf("cluster %q funded-to-date must be finite and in [0, %.0f]", c.ID, maxSimulationAmount)
	}
	if len(c.Credits) == 0 {
		return fmt.Errorf("cluster %q requires at least one controller credit", c.ID)
	}
	if len(c.Credits) > maxCreditsPerCluster {
		return fmt.Errorf(
			"cluster %q exceeds exploratory limit of %d controller credits",
			c.ID,
			maxCreditsPerCluster,
		)
	}
	seen := make(map[string]struct{}, len(c.Credits))
	for i, credit := range c.Credits {
		if credit.Controller == "" {
			return fmt.Errorf("cluster %q credit %d has no controller", c.ID, i)
		}
		if credit.Credit <= 0 || credit.Credit > maxSimulationAmount ||
			math.IsNaN(credit.Credit) || math.IsInf(credit.Credit, 0) {
			return fmt.Errorf("cluster %q credit %d must be finite and in (0, %.0f]", c.ID, i, maxSimulationAmount)
		}
		if _, duplicate := seen[credit.Controller]; duplicate {
			return fmt.Errorf("cluster %q repeats controller %q; aggregate it before evaluation", c.ID, credit.Controller)
		}
		seen[credit.Controller] = struct{}{}
	}
	for controller, paid := range c.ControllerDirectToDate {
		if _, exists := seen[controller]; !exists {
			return fmt.Errorf("cluster %q has paid state for unknown controller %q", c.ID, controller)
		}
		if paid < 0 || paid > maxSimulationAmount ||
			math.IsNaN(paid) || math.IsInf(paid, 0) {
			return fmt.Errorf("cluster %q paid state for %q is out of range", c.ID, controller)
		}
	}
	return c.Evidence.validate()
}

func canonicalCredits(credits []ControllerCredit) ([]ControllerCredit, float64, error) {
	sorted := cloneCredits(credits)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Controller < sorted[j].Controller
	})
	var total float64
	for _, credit := range sorted {
		total += credit.Credit
		if math.IsInf(total, 0) || math.IsNaN(total) || total > maxSimulationAmount {
			return nil, 0, errors.New("controller credit sum exceeds exploratory range")
		}
	}
	if total <= 0 {
		return nil, 0, errors.New("controller credit sum must be positive")
	}
	return sorted, total, nil
}

func cloneCredits(input []ControllerCredit) []ControllerCredit {
	result := make([]ControllerCredit, len(input))
	for i, credit := range input {
		result[i] = credit
		result[i].Roles = append([]string(nil), credit.Roles...)
	}
	return result
}

func cumulativeCreditTargets(total float64, credits []ControllerCredit, creditTotal float64) []float64 {
	targets := make([]float64, len(credits))
	remaining := total
	for i, credit := range credits {
		if i == len(credits)-1 {
			targets[i] = remaining
			break
		}
		target := total * credit.Credit / creditTotal
		target = math.Max(0, math.Min(target, remaining))
		targets[i] = target
		remaining -= target
	}
	return targets
}

func (c Cluster) validateEconomicState(p Params) error {
	credits, creditTotal, err := canonicalCredits(c.Credits)
	if err != nil {
		return fmt.Errorf("cluster %q: %w", c.ID, err)
	}
	maxFunded := c.LifetimeCap * c.PriorHighWater
	if c.FundedToDate > maxFunded &&
		!amountsEqual(c.FundedToDate, maxFunded) {
		return fmt.Errorf(
			"cluster %q funded-to-date %.12f exceeds prior cumulative target %.12f",
			c.ID,
			c.FundedToDate,
			maxFunded,
		)
	}
	targets := cumulativeCreditTargets(c.FundedToDate, credits, creditTotal)
	for i, credit := range credits {
		expected := math.Min(
			targets[i],
			p.ControllerCapShare*c.LifetimeCap,
		)
		actual := c.ControllerDirectToDate[credit.Controller]
		if !amountsEqual(actual, expected) {
			return fmt.Errorf(
				"cluster %q controller %q paid-to-date %.12f does not match cumulative target %.12f",
				c.ID,
				credit.Controller,
				actual,
				expected,
			)
		}
	}
	return nil
}

// ControllerAllocation records the delta from a cumulative controller target.
type ControllerAllocation struct {
	Controller             string  `json:"controller"`
	CreditShare            float64 `json:"credit_share"`
	Gross                  float64 `json:"gross"`
	PriorDirect            float64 `json:"prior_direct"`
	CumulativeDirectTarget float64 `json:"cumulative_direct_target"`
	Direct                 float64 `json:"direct"`
	Overflow               float64 `json:"overflow_to_commons"`
}

// ClusterResult is an auditable evaluation record.
type ClusterResult struct {
	ID               string                 `json:"id"`
	ArtifactCount    int                    `json:"artifact_count"`
	Score            ScoreDetail            `json:"score"`
	PriorHighWater   float64                `json:"prior_high_water"`
	NewHighWater     float64                `json:"new_high_water"`
	Delta            float64                `json:"delta"`
	GrossEntitlement float64                `json:"new_gross_accrual"`
	EligibleDemand   float64                `json:"eligible_demand"`
	PriorityWeight   float64                `json:"priority_weight"`
	FundedBudget     float64                `json:"funded_budget"`
	PriorFunded      float64                `json:"prior_funded"`
	NewFunded        float64                `json:"new_funded"`
	Allocations      []ControllerAllocation `json:"allocations"`
}

// ClusterState is the state a trusted engine, not a caller, carries forward.
type ClusterState struct {
	HighWater              float64            `json:"high_water"`
	LifetimeCap            float64            `json:"lifetime_cap"`
	FundedToDate           float64            `json:"funded_to_date"`
	ControllerDirectToDate map[string]float64 `json:"controller_direct_to_date"`
}

// EpochResult always conserves Budget:
// DirectTotal + CommonsTotal + Unallocated == Budget, within float tolerance.
type EpochResult struct {
	Budget           float64                 `json:"budget"`
	DirectTotal      float64                 `json:"direct_total"`
	CommonsTotal     float64                 `json:"commons_total"`
	Unallocated      float64                 `json:"unallocated"`
	UnfundedDemand   float64                 `json:"unfunded_demand"`
	ControllerDirect map[string]float64      `json:"controller_direct"`
	HighWater        map[string]float64      `json:"high_water"`
	States           map[string]ClusterState `json:"states"`
	Clusters         []ClusterResult         `json:"clusters"`
}

// EvaluateEpoch accrues entitlement only from new evidence above an
// irreversible economic high-water mark. Gross accrual is the difference of a
// cumulative target,
//
//	T_C(H) = LifetimeCap_C * H,
//
// so revocation followed by recovery cannot accrue the same economic target
// twice. Eligible demand is cumulative accrued entitlement minus funded to
// date, so scarcity leaves a non-guaranteed backlog rather than rewarding
// paced disclosure. The engine applies a cumulative controller cap within each
// immutable semantic cluster. A separate program-wide cross-cluster exposure
// cap and staged role/commons settlement remain integration requirements.
func EvaluateEpoch(clusters []Cluster, p Params) (EpochResult, error) {
	if err := p.Validate(); err != nil {
		return EpochResult{}, err
	}
	if len(clusters) > maxClustersPerEpoch {
		return EpochResult{}, fmt.Errorf(
			"epoch exceeds exploratory limit of %d clusters",
			maxClustersPerEpoch,
		)
	}
	result := EpochResult{
		Budget:           p.Budget,
		ControllerDirect: make(map[string]float64),
		HighWater:        make(map[string]float64),
		States:           make(map[string]ClusterState),
	}

	sorted := append([]Cluster(nil), clusters...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].ID < sorted[j].ID })
	seenClusters := make(map[string]struct{}, len(sorted))
	var totalDemand float64
	for i := range sorted {
		cluster := &sorted[i]
		if err := cluster.validate(); err != nil {
			return EpochResult{}, err
		}
		credits, _, err := canonicalCredits(cluster.Credits)
		if err != nil {
			return EpochResult{}, fmt.Errorf("cluster %q: %w", cluster.ID, err)
		}
		cluster.Credits = credits
		if err := cluster.validateEconomicState(p); err != nil {
			return EpochResult{}, err
		}
		if _, duplicate := seenClusters[cluster.ID]; duplicate {
			return EpochResult{}, fmt.Errorf("duplicate cluster ID %q", cluster.ID)
		}
		seenClusters[cluster.ID] = struct{}{}

		score, err := ScoreEvidence(cluster.Evidence, p)
		if err != nil {
			return EpochResult{}, err
		}
		newHighWater := math.Max(cluster.PriorHighWater, score.Total)
		delta := math.Max(0, newHighWater-cluster.PriorHighWater)
		newAccrued := cluster.LifetimeCap * newHighWater
		entitlement := cluster.LifetimeCap * delta
		eligibleDemand := math.Max(0, newAccrued-cluster.FundedToDate)
		priority := 0.0
		if eligibleDemand > 0 {
			priority = math.Pow(eligibleDemand, p.Alpha)
		}
		if math.IsNaN(entitlement) || math.IsInf(entitlement, 0) ||
			math.IsNaN(eligibleDemand) || math.IsInf(eligibleDemand, 0) ||
			math.IsNaN(priority) || math.IsInf(priority, 0) {
			return EpochResult{}, fmt.Errorf("cluster %q entitlement exceeds exploratory range", cluster.ID)
		}
		result.HighWater[cluster.ID] = newHighWater
		result.Clusters = append(result.Clusters, ClusterResult{
			ID:               cluster.ID,
			ArtifactCount:    len(cluster.ArtifactIDs),
			Score:            score,
			PriorHighWater:   cluster.PriorHighWater,
			NewHighWater:     newHighWater,
			Delta:            delta,
			GrossEntitlement: entitlement,
			EligibleDemand:   eligibleDemand,
			PriorityWeight:   priority,
			PriorFunded:      cluster.FundedToDate,
		})
		totalDemand += eligibleDemand
		if math.IsNaN(totalDemand) || math.IsInf(totalDemand, 0) ||
			totalDemand > maxAggregateAmount {
			return EpochResult{}, errors.New("aggregate entitlement exceeds exploratory range")
		}
	}

	demands := make([]float64, len(result.Clusters))
	for i := range result.Clusters {
		demands[i] = result.Clusters[i].EligibleDemand
	}
	funded := allocateCappedConcave(demands, p.Budget, p.Alpha)
	var fundedTotal float64
	for i := range result.Clusters {
		result.Clusters[i].FundedBudget = funded[i]
		result.Clusters[i].NewFunded = sorted[i].FundedToDate + funded[i]
		fundedTotal += funded[i]
	}
	if math.IsNaN(fundedTotal) || math.IsInf(fundedTotal, 0) ||
		fundedTotal > p.Budget {
		return EpochResult{}, errors.New("funded allocation is outside the epoch budget")
	}
	result.Unallocated = p.Budget - fundedTotal
	result.UnfundedDemand = math.Max(0, totalDemand-fundedTotal)

	for clusterIndex := range result.Clusters {
		cluster := sorted[clusterIndex]
		clusterResult := &result.Clusters[clusterIndex]
		credits, creditTotal, err := canonicalCredits(cluster.Credits)
		if err != nil {
			return EpochResult{}, fmt.Errorf("cluster %q: %w", cluster.ID, err)
		}
		priorTargets := cumulativeCreditTargets(
			cluster.FundedToDate,
			credits,
			creditTotal,
		)
		newTargets := cumulativeCreditTargets(
			clusterResult.NewFunded,
			credits,
			creditTotal,
		)
		directState := make(map[string]float64, len(credits))
		var clusterDirect float64
		for creditIndex, credit := range credits {
			share := credit.Credit / creditTotal
			gross := math.Max(0, newTargets[creditIndex]-priorTargets[creditIndex])
			priorDirect := cluster.ControllerDirectToDate[credit.Controller]
			cumulativeTarget := math.Min(
				newTargets[creditIndex],
				p.ControllerCapShare*cluster.LifetimeCap,
			)
			direct := math.Max(0, cumulativeTarget-priorDirect)
			if direct > gross && !amountsEqual(direct, gross) {
				return EpochResult{}, fmt.Errorf(
					"cluster %q controller %q direct delta %.12f exceeds funded gross %.12f",
					cluster.ID,
					credit.Controller,
					direct,
					gross,
				)
			}
			if direct > gross {
				direct = gross
			}
			overflow := gross - direct
			clusterResult.Allocations = append(clusterResult.Allocations, ControllerAllocation{
				Controller:             credit.Controller,
				CreditShare:            share,
				Gross:                  gross,
				PriorDirect:            priorDirect,
				CumulativeDirectTarget: cumulativeTarget,
				Direct:                 direct,
				Overflow:               overflow,
			})
			directState[credit.Controller] = cumulativeTarget
			result.ControllerDirect[credit.Controller] += direct
			result.DirectTotal += direct
			result.CommonsTotal += overflow
			clusterDirect += direct
		}
		if !amountsEqual(
			clusterDirect+sumAllocationOverflow(clusterResult.Allocations),
			clusterResult.FundedBudget,
		) {
			return EpochResult{}, fmt.Errorf("cluster %q allocation does not conserve funded budget", cluster.ID)
		}
		result.States[cluster.ID] = ClusterState{
			HighWater:              clusterResult.NewHighWater,
			LifetimeCap:            cluster.LifetimeCap,
			FundedToDate:           clusterResult.NewFunded,
			ControllerDirectToDate: directState,
		}
	}
	accounted := result.DirectTotal + result.CommonsTotal + result.Unallocated
	if !amountsEqual(accounted, p.Budget) {
		return EpochResult{}, fmt.Errorf(
			"epoch accounting %.12f does not equal budget %.12f",
			accounted,
			p.Budget,
		)
	}

	return result, nil
}

func sumAllocationOverflow(allocations []ControllerAllocation) float64 {
	var total float64
	for _, allocation := range allocations {
		total += allocation.Overflow
	}
	return total
}

// allocateCappedConcave is a deterministic capped water-fill. It never funds a
// cluster above its entitlement. Concavity matters only when aggregate demand
// exceeds the epoch budget.
func allocateCappedConcave(demands []float64, budget, alpha float64) []float64 {
	result := make([]float64, len(demands))
	var totalDemand float64
	for _, demand := range demands {
		totalDemand += demand
	}
	if totalDemand <= budget {
		copy(result, demands)
		return result
	}

	active := make([]int, 0, len(demands))
	for i, demand := range demands {
		if demand > 0 {
			active = append(active, i)
		}
	}
	remaining := budget
	for remaining > 0 && len(active) > 0 {
		var weightTotal float64
		for _, i := range active {
			weightTotal += math.Pow(demands[i], alpha)
		}
		if weightTotal <= 0 {
			break
		}

		capped := make(map[int]struct{})
		for _, i := range active {
			proposed := remaining * math.Pow(demands[i], alpha) / weightTotal
			unfunded := demands[i] - result[i]
			if proposed >= unfunded {
				capped[i] = struct{}{}
			}
		}
		if len(capped) > 0 {
			next := active[:0]
			for _, i := range active {
				if _, isCapped := capped[i]; isCapped {
					unfunded := demands[i] - result[i]
					result[i] += unfunded
					remaining -= unfunded
					continue
				}
				next = append(next, i)
			}
			active = next
			continue
		}
		for _, i := range active {
			share := remaining * math.Pow(demands[i], alpha) / weightTotal
			result[i] += share
		}
		remaining = 0
	}
	var allocated float64
	for i := range result {
		result[i] = math.Max(0, math.Min(result[i], demands[i]))
		allocated += result[i]
	}
	if allocated > budget {
		trimAllocationToBudget(result, budget)
	}
	return result
}

// trimAllocationToBudget removes any directional floating-point overshoot
// from the last positive allocations. Subtracting the apparent excess can
// round back to the same float, so Nextafter guarantees progress toward zero.
func trimAllocationToBudget(allocations []float64, budget float64) {
	for {
		var allocated float64
		for _, allocation := range allocations {
			allocated += allocation
		}
		if allocated <= budget {
			return
		}
		excess := allocated - budget
		adjusted := false
		for i := len(allocations) - 1; i >= 0; i-- {
			if allocations[i] <= 0 {
				continue
			}
			if excess >= allocations[i] {
				allocations[i] = 0
			} else {
				candidate := allocations[i] - excess
				allocations[i] = math.Nextafter(candidate, 0)
			}
			adjusted = true
			break
		}
		if !adjusted {
			return
		}
	}
}

func validateBinaryForecast(forecast, outcome float64) error {
	values := []struct {
		name  string
		value float64
	}{
		{"forecast", forecast},
		{"outcome", outcome},
	}
	for _, item := range values {
		if item.value < 0 || item.value > 1 ||
			math.IsNaN(item.value) || math.IsInf(item.value, 0) {
			return fmt.Errorf("%s must be in [0, 1]", item.name)
		}
	}
	if outcome != 0 && outcome != 1 {
		return errors.New("resolved binary outcome must be exactly 0 or 1")
	}
	return nil
}

// BinaryBrierScore is bounded in [0,1] and strictly proper in expectation.
func BinaryBrierScore(forecast, outcome float64) (float64, error) {
	if err := validateBinaryForecast(forecast, outcome); err != nil {
		return 0, err
	}
	return 1 - math.Pow(forecast-outcome, 2), nil
}

// BrierSkill is a signed diagnostic relative to a public baseline. It is never
// clipped by outcome and is not itself the payment function.
func BrierSkill(forecast, outcome, baseline float64) (float64, error) {
	if baseline < 0 || baseline > 1 || math.IsNaN(baseline) || math.IsInf(baseline, 0) {
		return 0, errors.New("baseline must be in [0, 1]")
	}
	score, err := BinaryBrierScore(forecast, outcome)
	if err != nil {
		return 0, err
	}
	baselineScore, err := BinaryBrierScore(baseline, outcome)
	if err != nil {
		return 0, err
	}
	return score - baselineScore, nil
}

// ReviewerPayment preserves strict propriety by using a positive affine
// transform of the proper score. Cost reimbursement is outcome-independent.
func ReviewerPayment(forecast, outcome, costReimbursement, bonusScale float64) (float64, error) {
	if costReimbursement < 0 || bonusScale <= 0 ||
		math.IsNaN(costReimbursement) || math.IsInf(costReimbursement, 0) ||
		math.IsNaN(bonusScale) || math.IsInf(bonusScale, 0) {
		return 0, errors.New("cost reimbursement must be finite and non-negative; bonus scale must be finite and positive")
	}
	score, err := BinaryBrierScore(forecast, outcome)
	if err != nil {
		return 0, err
	}
	payment := costReimbursement + bonusScale*score
	if math.IsNaN(payment) || math.IsInf(payment, 0) || payment > maxSimulationAmount {
		return 0, errors.New("reviewer payment exceeds exploratory range")
	}
	return payment, nil
}

// ActorPower makes the proposed non-conversion boundary executable. Liquid
// rewards and wealth are observable but do not enter either voice function.
type ActorPower struct {
	Qualified       bool    `json:"qualified"`
	ConflictFree    bool    `json:"conflict_free"`
	RewardBalance   float64 `json:"reward_balance"`
	LiquidWealth    float64 `json:"liquid_wealth"`
	CivicCredential float64 `json:"civic_credential"`
}

// ReviewVoice is equal, capped epistemic voice for a qualified, conflict-free
// controller. Bonds can secure behavior but cannot multiply this weight.
func ReviewVoice(actor ActorPower) float64 {
	if !actor.Qualified || !actor.ConflictFree {
		return 0
	}
	return 1
}

// GovernanceVoice is derived only from a separately issued civic credential.
// It intentionally ignores scientific credit, liquid wealth, and ZRN rewards.
func GovernanceVoice(actor ActorPower) float64 {
	if actor.CivicCredential <= 0 ||
		math.IsNaN(actor.CivicCredential) ||
		math.IsInf(actor.CivicCredential, 0) {
		return 0
	}
	return math.Min(1, actor.CivicCredential)
}

// SurfaceMetric measures concentration on one controller-level power surface.
type SurfaceMetric struct {
	Surface        string  `json:"surface"`
	HHI            float64 `json:"hhi"`
	EffectiveCount float64 `json:"effective_count"`
	NakamotoCount  int     `json:"nakamoto_count"`
	LargestShare   float64 `json:"largest_share"`
}

// MeasurePower normalizes non-negative controller power and reports HHI,
// effective count, and the smallest coalition reaching threshold.
func MeasurePower(surface string, powers map[string]float64, threshold float64) (SurfaceMetric, error) {
	if surface == "" {
		return SurfaceMetric{}, errors.New("power surface name is required")
	}
	if threshold <= 0 || threshold > 1 ||
		math.IsNaN(threshold) || math.IsInf(threshold, 0) {
		return SurfaceMetric{}, errors.New("coalition threshold must be in (0, 1]")
	}
	if len(powers) > maxPowerControllers {
		return SurfaceMetric{}, fmt.Errorf(
			"power surface exceeds exploratory limit of %d controllers",
			maxPowerControllers,
		)
	}
	controllers := make([]string, 0, len(powers))
	for controller := range powers {
		controllers = append(controllers, controller)
	}
	sort.Strings(controllers)
	var total float64
	shares := make([]float64, 0, len(powers))
	for _, controller := range controllers {
		power := powers[controller]
		if controller == "" {
			return SurfaceMetric{}, errors.New("power controller label is required")
		}
		if power < 0 || power > maxSimulationAmount ||
			math.IsNaN(power) || math.IsInf(power, 0) {
			return SurfaceMetric{}, fmt.Errorf(
				"power for %q must be finite and in [0, %.0f]",
				controller,
				maxSimulationAmount,
			)
		}
		total += power
		if math.IsNaN(total) || math.IsInf(total, 0) ||
			total > maxSimulationAmount {
			return SurfaceMetric{}, errors.New("power surface total exceeds exploratory range")
		}
	}
	if total <= 0 {
		return SurfaceMetric{}, errors.New("power surface total must be positive")
	}
	var hhi, largest float64
	for _, controller := range controllers {
		power := powers[controller]
		share := power / total
		shares = append(shares, share)
		hhi += share * share
		largest = math.Max(largest, share)
	}
	sort.Sort(sort.Reverse(sort.Float64Slice(shares)))
	var cumulative float64
	nakamoto := 0
	for _, share := range shares {
		cumulative += share
		nakamoto++
		if cumulative+floatTolerance >= threshold {
			break
		}
	}
	return SurfaceMetric{
		Surface:        surface,
		HHI:            hhi,
		EffectiveCount: 1 / hhi,
		NakamotoCount:  nakamoto,
		LargestShare:   largest,
	}, nil
}

// AuditPower refuses to hide a captured surface inside a healthy average. Its
// overall effective count is the minimum across all supplied surfaces.
func AuditPower(surfaces map[string]map[string]float64, threshold float64) ([]SurfaceMetric, float64, error) {
	names := make([]string, 0, len(surfaces))
	for name := range surfaces {
		names = append(names, name)
	}
	sort.Strings(names)
	weakest := math.Inf(1)
	metrics := make([]SurfaceMetric, 0, len(names))
	for _, name := range names {
		metric, err := MeasurePower(name, surfaces[name], threshold)
		if err != nil {
			return nil, 0, err
		}
		metrics = append(metrics, metric)
		weakest = math.Min(weakest, metric.EffectiveCount)
	}
	if len(metrics) == 0 {
		return nil, 0, errors.New("at least one power surface is required")
	}
	return metrics, weakest, nil
}

// AuditRequiredPower validates the exact policy-owned surface set and applies
// each surface's own coalition threshold. Extra or omitted surfaces fail
// closed; a healthy decoy cannot replace a captured required path.
func AuditRequiredPower(
	surfaces map[string]map[string]float64,
	thresholds map[string]float64,
) ([]SurfaceMetric, float64, error) {
	if len(surfaces) != len(requiredPowerSurfaceNames) {
		return nil, 0, fmt.Errorf(
			"power snapshot has %d surfaces, want exactly %d",
			len(surfaces),
			len(requiredPowerSurfaceNames),
		)
	}
	if len(thresholds) != len(requiredPowerSurfaceNames) {
		return nil, 0, fmt.Errorf(
			"power policy has %d thresholds, want exactly %d",
			len(thresholds),
			len(requiredPowerSurfaceNames),
		)
	}
	required := make(map[string]struct{}, len(requiredPowerSurfaceNames))
	for _, surface := range requiredPowerSurfaceNames {
		required[surface] = struct{}{}
	}
	for surface := range surfaces {
		if _, exists := required[surface]; !exists {
			return nil, 0, fmt.Errorf("unexpected power surface %q", surface)
		}
	}
	for surface := range thresholds {
		if _, exists := required[surface]; !exists {
			return nil, 0, fmt.Errorf("unexpected power threshold %q", surface)
		}
	}

	metrics := make([]SurfaceMetric, 0, len(requiredPowerSurfaceNames))
	weakest := math.Inf(1)
	for _, surface := range requiredPowerSurfaceNames {
		powers, exists := surfaces[surface]
		if !exists {
			return nil, 0, fmt.Errorf("required power surface %q is missing", surface)
		}
		threshold, exists := thresholds[surface]
		if !exists {
			return nil, 0, fmt.Errorf("required power threshold %q is missing", surface)
		}
		metric, err := MeasurePower(surface, powers, threshold)
		if err != nil {
			return nil, 0, err
		}
		metrics = append(metrics, metric)
		weakest = math.Min(weakest, metric.EffectiveCount)
	}
	return metrics, weakest, nil
}

// NaiveSqrtStakeShare shows the address-splitting failure of per-address
// concavity. It is included as an adversarial comparator, never a proposal.
func NaiveSqrtStakeShare(controllerStake, otherStake float64, aliases int) (float64, error) {
	if controllerStake < 0 || otherStake < 0 ||
		math.IsNaN(controllerStake) || math.IsNaN(otherStake) ||
		math.IsInf(controllerStake, 0) || math.IsInf(otherStake, 0) ||
		aliases < 1 {
		return 0, errors.New("stakes must be finite and non-negative and aliases positive")
	}
	controllerWeight := float64(aliases) * math.Sqrt(controllerStake/float64(aliases))
	otherWeight := math.Sqrt(otherStake)
	if controllerWeight+otherWeight == 0 {
		return 0, errors.New("total stake must be positive")
	}
	return controllerWeight / (controllerWeight + otherWeight), nil
}

// NaiveArtifactShare shows the fixed-budget salami attack when every artifact
// is treated as a separate equally-scored contribution.
func NaiveArtifactShare(attackerArtifacts, honestArtifacts int) (float64, error) {
	if attackerArtifacts < 1 || honestArtifacts < 1 {
		return 0, errors.New("artifact counts must be positive")
	}
	return float64(attackerArtifacts) / float64(attackerArtifacts+honestArtifacts), nil
}
