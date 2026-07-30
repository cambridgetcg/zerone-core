package main

import (
	"errors"
	"fmt"
	"sort"
	"sync"
)

// ClusterProposal contains evidence, never authoritative accounting state.
type ClusterProposal struct {
	EventID     string   `json:"event_id"`
	ClusterID   string   `json:"cluster_id"`
	ArtifactIDs []string `json:"artifact_ids,omitempty"`
	Evidence    Evidence `json:"evidence"`
}

type registeredCluster struct {
	credits []ControllerCredit
	state   ClusterState
}

type RegisteredClusterSnapshot struct {
	Credits []ControllerCredit `json:"credits"`
	State   ClusterState       `json:"state"`
}

// EngineSnapshot contains every input that can change future simulator
// behavior. It is a replay fixture, not a consensus state format.
type EngineSnapshot struct {
	ArithmeticVersion string                               `json:"arithmetic_version"`
	Parameters        Params                               `json:"parameters"`
	Clusters          map[string]RegisteredClusterSnapshot `json:"clusters"`
	SeenEpochIDs      []string                             `json:"seen_epoch_ids"`
	SeenEventIDs      []string                             `json:"seen_event_ids"`
}

// Engine owns immutable cluster caps/credits, high-water state, cumulative
// funding, cumulative direct targets, policy-owned power surfaces, epoch IDs,
// and replay IDs. Its mutex makes calculation commits serializable; it does
// not pretend external bank transfers exist.
type Engine struct {
	mu         sync.Mutex
	params     Params
	clusters   map[string]registeredCluster
	seenEpochs map[string]struct{}
	seenEvents map[string]struct{}
}

func NewEngine(params Params) (*Engine, error) {
	if err := params.Validate(); err != nil {
		return nil, err
	}
	return &Engine{
		params:     cloneParams(params),
		clusters:   make(map[string]registeredCluster),
		seenEpochs: make(map[string]struct{}),
		seenEvents: make(map[string]struct{}),
	}, nil
}

// RegisterCluster snapshots one semantic cluster's lifetime cap and
// dependency-DAG credit partition before evidence can earn a reward.
func (e *Engine) RegisterCluster(id string, lifetimeCap float64, credits []ControllerCredit) error {
	if e == nil {
		return errors.New("engine is nil")
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.registerClusterLocked(id, lifetimeCap, credits)
}

func (e *Engine) registerClusterLocked(id string, lifetimeCap float64, credits []ControllerCredit) error {
	if len(e.clusters) >= maxClustersPerEpoch {
		return fmt.Errorf(
			"engine exceeds exploratory limit of %d registered clusters",
			maxClustersPerEpoch,
		)
	}
	if _, exists := e.clusters[id]; exists {
		return fmt.Errorf("cluster %q is already registered", id)
	}
	canonical, _, err := canonicalCredits(credits)
	if err != nil {
		return err
	}
	cluster := Cluster{
		ID:          id,
		LifetimeCap: lifetimeCap,
		Credits:     canonical,
	}
	if err := cluster.validate(); err != nil {
		return err
	}
	e.clusters[id] = registeredCluster{
		credits: canonical,
		state: ClusterState{
			LifetimeCap:            lifetimeCap,
			ControllerDirectToDate: make(map[string]float64),
		},
	}
	return nil
}

// RunEpoch is atomic with respect to simulator state. A repeated epoch,
// repeated event, unknown cluster, or invalid proposal changes nothing.
func (e *Engine) RunEpoch(epochID string, proposals []ClusterProposal) (EpochResult, error) {
	if e == nil {
		return EpochResult{}, errors.New("engine is nil")
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if epochID == "" {
		return EpochResult{}, errors.New("epoch ID is required")
	}
	if _, replay := e.seenEpochs[epochID]; replay {
		return EpochResult{}, fmt.Errorf("epoch %q already processed", epochID)
	}

	sorted := append([]ClusterProposal(nil), proposals...)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].ClusterID == sorted[j].ClusterID {
			return sorted[i].EventID < sorted[j].EventID
		}
		return sorted[i].ClusterID < sorted[j].ClusterID
	})
	pendingEvents := make(map[string]struct{}, len(sorted))
	pendingClusters := make(map[string]struct{}, len(sorted))
	clusters := make([]Cluster, 0, len(sorted))
	for _, proposal := range sorted {
		if proposal.EventID == "" {
			return EpochResult{}, errors.New("proposal event ID is required")
		}
		if _, replay := e.seenEvents[proposal.EventID]; replay {
			return EpochResult{}, fmt.Errorf("event %q already processed", proposal.EventID)
		}
		if _, duplicate := pendingEvents[proposal.EventID]; duplicate {
			return EpochResult{}, fmt.Errorf("event %q repeats within epoch", proposal.EventID)
		}
		if _, duplicate := pendingClusters[proposal.ClusterID]; duplicate {
			return EpochResult{}, fmt.Errorf("cluster %q repeats within epoch", proposal.ClusterID)
		}
		registered, exists := e.clusters[proposal.ClusterID]
		if !exists {
			return EpochResult{}, fmt.Errorf("cluster %q is not registered", proposal.ClusterID)
		}
		pendingEvents[proposal.EventID] = struct{}{}
		pendingClusters[proposal.ClusterID] = struct{}{}
		clusters = append(clusters, Cluster{
			ID:                     proposal.ClusterID,
			ArtifactIDs:            append([]string(nil), proposal.ArtifactIDs...),
			Evidence:               proposal.Evidence,
			PriorHighWater:         registered.state.HighWater,
			LifetimeCap:            registered.state.LifetimeCap,
			FundedToDate:           registered.state.FundedToDate,
			ControllerDirectToDate: cloneAmounts(registered.state.ControllerDirectToDate),
			Credits:                cloneCredits(registered.credits),
		})
		clusters[len(clusters)-1].Evidence = cloneEvidence(proposal.Evidence)
	}

	result, err := EvaluateEpoch(clusters, e.params)
	if err != nil {
		return EpochResult{}, err
	}

	for clusterID, state := range result.States {
		registered := e.clusters[clusterID]
		registered.state = cloneClusterState(state)
		e.clusters[clusterID] = registered
	}
	e.seenEpochs[epochID] = struct{}{}
	for eventID := range pendingEvents {
		e.seenEvents[eventID] = struct{}{}
	}
	return result, nil
}

func cloneAmounts(input map[string]float64) map[string]float64 {
	result := make(map[string]float64, len(input))
	for key, value := range input {
		result[key] = value
	}
	return result
}

func cloneClusterState(state ClusterState) ClusterState {
	return ClusterState{
		HighWater:              state.HighWater,
		LifetimeCap:            state.LifetimeCap,
		FundedToDate:           state.FundedToDate,
		ControllerDirectToDate: cloneAmounts(state.ControllerDirectToDate),
	}
}

func cloneEvidence(input Evidence) Evidence {
	result := input
	result.Replications = append([]Signal(nil), input.Replications...)
	result.Correlation = make([][]float64, len(input.Correlation))
	for i := range input.Correlation {
		result.Correlation[i] = append([]float64(nil), input.Correlation[i]...)
	}
	return result
}

func sortedSetKeys(input map[string]struct{}) []string {
	result := make([]string, 0, len(input))
	for key := range input {
		result = append(result, key)
	}
	sort.Strings(result)
	return result
}

// Snapshot returns a defensive, behavior-complete replay fixture.
func (e *Engine) Snapshot() EngineSnapshot {
	if e == nil {
		return EngineSnapshot{}
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	result := EngineSnapshot{
		ArithmeticVersion: engineArithmeticVersion,
		Parameters:        cloneParams(e.params),
		Clusters:          make(map[string]RegisteredClusterSnapshot, len(e.clusters)),
		SeenEpochIDs:      sortedSetKeys(e.seenEpochs),
		SeenEventIDs:      sortedSetKeys(e.seenEvents),
	}
	for id, registered := range e.clusters {
		result.Clusters[id] = RegisteredClusterSnapshot{
			Credits: cloneCredits(registered.credits),
			State:   cloneClusterState(registered.state),
		}
	}
	return result
}

// RestoreEngine validates and restores a snapshot produced by Snapshot.
func RestoreEngine(snapshot EngineSnapshot) (*Engine, error) {
	if snapshot.ArithmeticVersion != engineArithmeticVersion {
		return nil, fmt.Errorf(
			"arithmetic version %q is not supported",
			snapshot.ArithmeticVersion,
		)
	}
	engine, err := NewEngine(snapshot.Parameters)
	if err != nil {
		return nil, err
	}
	if len(snapshot.Clusters) > maxClustersPerEpoch {
		return nil, fmt.Errorf(
			"snapshot exceeds exploratory limit of %d clusters",
			maxClustersPerEpoch,
		)
	}
	for id, registered := range snapshot.Clusters {
		canonical, _, err := canonicalCredits(registered.Credits)
		if err != nil {
			return nil, fmt.Errorf("cluster %q: %w", id, err)
		}
		cluster := Cluster{
			ID:                     id,
			PriorHighWater:         registered.State.HighWater,
			LifetimeCap:            registered.State.LifetimeCap,
			FundedToDate:           registered.State.FundedToDate,
			ControllerDirectToDate: cloneAmounts(registered.State.ControllerDirectToDate),
			Credits:                canonical,
		}
		if err := cluster.validate(); err != nil {
			return nil, err
		}
		if err := cluster.validateEconomicState(engine.params); err != nil {
			return nil, err
		}
		engine.clusters[id] = registeredCluster{
			credits: canonical,
			state:   cloneClusterState(registered.State),
		}
	}
	for _, epochID := range snapshot.SeenEpochIDs {
		if epochID == "" {
			return nil, errors.New("snapshot contains an empty epoch ID")
		}
		if _, duplicate := engine.seenEpochs[epochID]; duplicate {
			return nil, fmt.Errorf("snapshot repeats epoch ID %q", epochID)
		}
		engine.seenEpochs[epochID] = struct{}{}
	}
	for _, eventID := range snapshot.SeenEventIDs {
		if eventID == "" {
			return nil, errors.New("snapshot contains an empty event ID")
		}
		if _, duplicate := engine.seenEvents[eventID]; duplicate {
			return nil, fmt.Errorf("snapshot repeats event ID %q", eventID)
		}
		engine.seenEvents[eventID] = struct{}{}
	}
	return engine, nil
}
