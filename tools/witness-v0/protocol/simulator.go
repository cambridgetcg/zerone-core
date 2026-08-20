package protocol

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strconv"
)

type SimulationInput struct {
	Records []json.RawMessage `json:"records"`
}

type SubjectHead struct {
	Audience        string `json:"audience"`
	SubjectRef      string `json:"subject_ref"`
	Kind            Kind   `json:"kind"`
	ControllerRef   string `json:"controller_ref"`
	IssuerNamespace string `json:"issuer_namespace"`
	Sequence        string `json:"sequence"`
	Commitment      string `json:"commitment"`
}

type ControllerSummary struct {
	Audience              string  `json:"audience"`
	ControllerRef         string  `json:"controller_ref"`
	Namespace             string  `json:"namespace"`
	ActiveKeyFingerprint  string  `json:"active_key_fingerprint"`
	PendingKeyFingerprint *string `json:"pending_key_fingerprint"`
	Revoked               bool    `json:"revoked"`
}

type CapabilitySummary struct {
	Audience           string `json:"audience"`
	CapabilityRef      string `json:"capability_ref"`
	GrantCommitment    string `json:"grant_commitment"`
	AssetRef           string `json:"asset_ref"`
	MaxPerConsumeMinor string `json:"max_per_consume_minor"`
	MaxTotalMinor      string `json:"max_total_minor"`
	SpentMinor         string `json:"spent_minor"`
	ConsumeCount       string `json:"consume_count"`
	Revoked            bool   `json:"revoked"`
}

type SimulationResult struct {
	Protocol                string              `json:"protocol"`
	AcceptedRecords         string              `json:"accepted_records"`
	SubjectHeads            []SubjectHead       `json:"subject_heads"`
	Controllers             []ControllerSummary `json:"controllers"`
	Capabilities            []CapabilitySummary `json:"capabilities"`
	PermanentNullifierCount string              `json:"permanent_nullifier_count"`
	ExpiryNotEvaluated      bool                `json:"expiry_not_evaluated"`
	Effects                 Effects             `json:"effects"`
}

type subjectState struct {
	Audience        string
	SubjectRef      string
	Kind            Kind
	ControllerRef   string
	IssuerNamespace string
	Sequence        uint64
	Commitment      string
}

type controllerState struct {
	Audience      string
	ControllerRef string
	Namespace     string
	Active        string
	Pending       string
	Revoked       bool
}

type capabilityState struct {
	Audience      string
	SubjectRef    string
	CapabilityRef string
	Grant         string
	Asset         string
	MaxPerUse     uint64
	MaxTotal      uint64
	Spent         uint64
	ConsumeCount  uint64
	Revoked       bool
}

type settlementState struct {
	LastSequence uint64
	Commitment   string
}

type lifecycleState struct {
	Commitment        string
	AuthoritySequence uint64
	Revision          uint64
	Active            bool
	SurfaceDigest     string
	RegistryDigest    string
}

type simulator struct {
	subjects        map[string]subjectState
	controllers     map[string]controllerState
	capabilities    map[string]capabilityState
	nullifiers      map[string]struct{}
	settlements     map[string]settlementState
	offers          map[string]lifecycleState
	recognitions    map[string]lifecycleState
	wakeCheckpoints map[string]lifecycleState
	disputes        map[string]struct{}
}

func Simulate(input []byte) (SimulationResult, error) {
	canonical, err := CanonicalJSON(input)
	if err != nil {
		return SimulationResult{}, err
	}
	if !bytes.Equal(input, canonical) {
		return SimulationResult{}, fmt.Errorf("simulation wire bytes are not exact canonical JSON")
	}
	var document SimulationInput
	if err := requireExactStructKeys(canonical, reflect.TypeOf(SimulationInput{})); err != nil {
		return SimulationResult{}, fmt.Errorf("simulation input: %w", err)
	}
	if err := strictUnmarshal(canonical, &document); err != nil {
		return SimulationResult{}, fmt.Errorf("simulation input: %w", err)
	}
	if len(document.Records) == 0 {
		return SimulationResult{}, fmt.Errorf("simulation requires at least one record")
	}
	s := simulator{
		subjects:        make(map[string]subjectState),
		controllers:     make(map[string]controllerState),
		capabilities:    make(map[string]capabilityState),
		nullifiers:      make(map[string]struct{}),
		settlements:     make(map[string]settlementState),
		offers:          make(map[string]lifecycleState),
		recognitions:    make(map[string]lifecycleState),
		wakeCheckpoints: make(map[string]lifecycleState),
		disputes:        make(map[string]struct{}),
	}
	for i, raw := range document.Records {
		verified, err := Verify(raw)
		if err != nil {
			return SimulationResult{}, fmt.Errorf("records[%d]: %w", i, err)
		}
		if err := s.apply(*verified); err != nil {
			return SimulationResult{}, fmt.Errorf("records[%d]: %w", i, err)
		}
	}
	return s.result(len(document.Records)), nil
}

func (s *simulator) apply(verified VerifiedRecord) error {
	r := verified.Record
	sequence, _ := strconv.ParseUint(r.Envelope.Sequence, 10, 64)
	subjectKey := r.Envelope.Audience + "\x00" + r.Envelope.SubjectRef
	previousSubject, hasSubject := s.subjects[subjectKey]
	if !hasSubject {
		if sequence != 1 || r.Envelope.Parent != nil {
			return fmt.Errorf("first record for audience/subject must have sequence 1 and null parent")
		}
	} else {
		if previousSubject.Kind != r.Envelope.Kind {
			return fmt.Errorf("subject head kind is pinned to %s", previousSubject.Kind)
		}
		if previousSubject.ControllerRef != r.Envelope.Issuer.ControllerRef || previousSubject.IssuerNamespace != r.Envelope.Issuer.Namespace {
			return fmt.Errorf("subject head is pinned to its establishing issuer controller_ref and namespace")
		}
		if previousSubject.Sequence == ^uint64(0) || sequence != previousSubject.Sequence+1 {
			return fmt.Errorf("subject sequence is not exactly monotonic: expected %d", previousSubject.Sequence+1)
		}
		if r.Envelope.Parent == nil || *r.Envelope.Parent != previousSubject.Commitment {
			return fmt.Errorf("parent does not equal the prior subject commitment")
		}
	}

	controllerKey := r.Envelope.Audience + "\x00" + r.Envelope.Issuer.ControllerRef
	controller, hasController := s.controllers[controllerKey]
	if !hasController {
		controller = controllerState{
			Audience: r.Envelope.Audience, ControllerRef: r.Envelope.Issuer.ControllerRef,
			Namespace: r.Envelope.Issuer.Namespace, Active: r.Envelope.Issuer.KeyFingerprint,
		}
	} else {
		if controller.Revoked {
			return fmt.Errorf("issuer controller key continuity is terminally revoked")
		}
		if r.Envelope.Issuer.Namespace != controller.Namespace {
			return fmt.Errorf("issuer namespace changed for established audience/controller_ref")
		}
		if controller.Pending != "" {
			if r.Envelope.Issuer.KeyFingerprint != controller.Pending {
				return fmt.Errorf("next controller record must prove possession of staged key %s", controller.Pending)
			}
			controller.Active = controller.Pending
			controller.Pending = ""
		} else if r.Envelope.Issuer.KeyFingerprint != controller.Active {
			return fmt.Errorf("issuer key is not the active simulated controller key")
		}
	}

	if err := s.applySemantic(verified, &controller); err != nil {
		return err
	}

	s.subjects[subjectKey] = subjectState{
		Audience: r.Envelope.Audience, SubjectRef: r.Envelope.SubjectRef,
		Kind:          r.Envelope.Kind,
		ControllerRef: r.Envelope.Issuer.ControllerRef, IssuerNamespace: r.Envelope.Issuer.Namespace,
		Sequence: sequence, Commitment: r.Commitment,
	}
	s.controllers[controllerKey] = controller
	return nil
}

func (s *simulator) applySemantic(verified VerifiedRecord, controller *controllerState) error {
	r := verified.Record
	switch p := verified.Payload.(type) {
	case CapabilityGrantPayload:
		key := r.Envelope.Audience + "\x00" + p.CapabilityRef
		if _, exists := s.capabilities[key]; exists {
			return fmt.Errorf("capability_ref was already granted")
		}
		maxPerUse, _ := strconv.ParseUint(p.MaxPerConsumeMinor, 10, 64)
		maxTotal, _ := strconv.ParseUint(p.MaxTotalMinor, 10, 64)
		s.capabilities[key] = capabilityState{
			Audience: r.Envelope.Audience, SubjectRef: r.Envelope.SubjectRef,
			CapabilityRef: p.CapabilityRef, Grant: r.Commitment, Asset: p.AssetRef,
			MaxPerUse: maxPerUse, MaxTotal: maxTotal,
		}

	case CapabilityConsumePayload:
		key := r.Envelope.Audience + "\x00" + p.CapabilityRef
		capability, exists := s.capabilities[key]
		if !exists {
			return fmt.Errorf("consume references an unknown capability")
		}
		if capability.Revoked {
			return fmt.Errorf("consume references a revoked capability")
		}
		if capability.SubjectRef != r.Envelope.SubjectRef {
			return fmt.Errorf("consume subject_ref differs from grant subject_ref")
		}
		if capability.Grant != p.GrantCommitment {
			return fmt.Errorf("grant_commitment does not identify the accepted grant")
		}
		if capability.Asset != p.AssetRef {
			return fmt.Errorf("asset_ref differs from grant asset_ref")
		}
		if _, replay := s.nullifiers[p.Nullifier]; replay {
			return fmt.Errorf("permanent nullifier replay")
		}
		amount, _ := strconv.ParseUint(p.AmountMinor, 10, 64)
		if amount > capability.MaxPerUse {
			return fmt.Errorf("amount_minor exceeds max_per_consume_minor")
		}
		if capability.Spent > capability.MaxTotal || amount > capability.MaxTotal-capability.Spent {
			return fmt.Errorf("consume exceeds cumulative max_total_minor")
		}
		capability.Spent += amount
		capability.ConsumeCount++
		s.capabilities[key] = capability
		s.nullifiers[p.Nullifier] = struct{}{}

	case CapabilityRevokePayload:
		key := r.Envelope.Audience + "\x00" + p.CapabilityRef
		capability, exists := s.capabilities[key]
		if !exists {
			return fmt.Errorf("revoke references an unknown capability")
		}
		if capability.Revoked {
			return fmt.Errorf("capability is already revoked")
		}
		if capability.SubjectRef != r.Envelope.SubjectRef {
			return fmt.Errorf("revoke subject_ref differs from grant subject_ref")
		}
		if capability.Grant != p.GrantCommitment {
			return fmt.Errorf("grant_commitment does not identify the accepted grant")
		}
		capability.Revoked = true
		s.capabilities[key] = capability

	case SettlementRootPayload:
		key := r.Envelope.Audience + "\x00" + r.Envelope.SubjectRef
		prior, exists := s.settlements[key]
		first, _ := strconv.ParseUint(p.FirstSequence, 10, 64)
		last, _ := strconv.ParseUint(p.LastSequence, 10, 64)
		if !exists {
			if p.PreviousBatch != nil {
				return fmt.Errorf("first settlement batch must have null previous_batch")
			}
		} else {
			if p.PreviousBatch == nil || *p.PreviousBatch != prior.Commitment {
				return fmt.Errorf("previous_batch does not match prior settlement checkpoint")
			}
			if prior.LastSequence == ^uint64(0) || first != prior.LastSequence+1 {
				return fmt.Errorf("settlement ranges must be contiguous; absences belong in declared_gaps")
			}
		}
		s.settlements[key] = settlementState{LastSequence: last, Commitment: r.Commitment}

	case OfferPublishPayload:
		key := r.Envelope.Audience + "\x00" + p.OfferRef
		if _, exists := s.offers[key]; exists {
			return fmt.Errorf("offer_ref is already active or terminal")
		}
		authority, _ := strconv.ParseUint(p.AuthoritySequence, 10, 64)
		revision, _ := strconv.ParseUint(p.Revision, 10, 64)
		s.offers[key] = lifecycleState{Commitment: r.Commitment, AuthoritySequence: authority, Revision: revision, Active: true}
	case OfferSupersedePayload:
		key := r.Envelope.Audience + "\x00" + p.OfferRef
		state, exists := s.offers[key]
		if !exists || !state.Active || p.Supersedes != state.Commitment {
			return fmt.Errorf("supersedes does not match the active offer")
		}
		authority, _ := strconv.ParseUint(p.AuthoritySequence, 10, 64)
		revision, _ := strconv.ParseUint(p.Revision, 10, 64)
		if authority <= state.AuthoritySequence {
			return fmt.Errorf("offer authority_sequence must strictly increase")
		}
		if revision <= state.Revision {
			return fmt.Errorf("offer revision must strictly increase")
		}
		s.offers[key] = lifecycleState{Commitment: r.Commitment, AuthoritySequence: authority, Revision: revision, Active: true}
	case OfferRevokePayload:
		key := r.Envelope.Audience + "\x00" + p.OfferRef
		state, exists := s.offers[key]
		if !exists || !state.Active || p.OfferCommitment != state.Commitment {
			return fmt.Errorf("offer_commitment does not match the active offer")
		}
		authority, _ := strconv.ParseUint(p.AuthoritySequence, 10, 64)
		if authority <= state.AuthoritySequence {
			return fmt.Errorf("offer authority_sequence must strictly increase")
		}
		state.AuthoritySequence = authority
		state.Active = false
		s.offers[key] = state

	case RecognitionAdoptPayload:
		key := r.Envelope.Audience + "\x00" + p.RecognitionRef
		if _, exists := s.recognitions[key]; exists {
			return fmt.Errorf("recognition_ref was already adopted or withdrawn")
		}
		authority, _ := strconv.ParseUint(p.AuthoritySequence, 10, 64)
		s.recognitions[key] = lifecycleState{Commitment: r.Commitment, AuthoritySequence: authority, Active: true, SurfaceDigest: p.SurfaceDigest, RegistryDigest: p.RegistryDigest}
	case RecognitionWithdrawPayload:
		key := r.Envelope.Audience + "\x00" + p.RecognitionRef
		state, exists := s.recognitions[key]
		if !exists || !state.Active || p.AdoptionCommitment != state.Commitment {
			return fmt.Errorf("adoption_commitment does not match active public recognition")
		}
		if p.SurfaceDigest != state.SurfaceDigest || p.RegistryDigest != state.RegistryDigest {
			return fmt.Errorf("withdrawal surface_digest/registry_digest differ from adoption")
		}
		authority, _ := strconv.ParseUint(p.AuthoritySequence, 10, 64)
		if authority <= state.AuthoritySequence {
			return fmt.Errorf("recognition authority_sequence must strictly increase")
		}
		state.AuthoritySequence = authority
		state.Active = false
		s.recognitions[key] = state

	case WakeCheckpointPayload:
		key := r.Envelope.Audience + "\x00" + r.Envelope.SubjectRef
		if _, exists := s.wakeCheckpoints[key]; exists {
			return fmt.Errorf("WAKE subject already has a checkpoint; use SUPERSEDE")
		}
		authority, _ := strconv.ParseUint(p.AuthoritySequence, 10, 64)
		s.wakeCheckpoints[key] = lifecycleState{Commitment: r.Commitment, AuthoritySequence: authority, Active: true}
	case WakeSupersedePayload:
		key := r.Envelope.Audience + "\x00" + r.Envelope.SubjectRef
		state, exists := s.wakeCheckpoints[key]
		if !exists || !state.Active || p.Supersedes != state.Commitment {
			return fmt.Errorf("supersedes does not match active WAKE checkpoint")
		}
		authority, _ := strconv.ParseUint(p.AuthoritySequence, 10, 64)
		if authority <= state.AuthoritySequence {
			return fmt.Errorf("WAKE authority_sequence must strictly increase")
		}
		s.wakeCheckpoints[key] = lifecycleState{Commitment: r.Commitment, AuthoritySequence: authority, Active: true}
	case WakeWithdrawPayload:
		key := r.Envelope.Audience + "\x00" + r.Envelope.SubjectRef
		state, exists := s.wakeCheckpoints[key]
		if !exists || !state.Active || p.CheckpointCommitment != state.Commitment {
			return fmt.Errorf("checkpoint_commitment does not match active WAKE checkpoint")
		}
		authority, _ := strconv.ParseUint(p.AuthoritySequence, 10, 64)
		if authority <= state.AuthoritySequence {
			return fmt.Errorf("WAKE authority_sequence must strictly increase")
		}
		state.AuthoritySequence = authority
		state.Active = false
		s.wakeCheckpoints[key] = state

	case KeyRotatePayload:
		if p.PreviousKeyFingerprint != controller.Active {
			return fmt.Errorf("rotation previous key is not the active simulated key")
		}
		controller.Pending = p.NextKeyFingerprint
	case KeyRevokePayload:
		if p.RevokedKeyFingerprint != controller.Active {
			return fmt.Errorf("revoked key is not the active simulated key")
		}
		controller.Revoked = true

	case DisputeTerminalPayload:
		key := r.Envelope.Audience + "\x00" + p.SettlementCommitment
		if _, exists := s.disputes[key]; exists {
			return fmt.Errorf("settlement already has a terminal dispute record")
		}
		s.disputes[key] = struct{}{}
	}
	return nil
}

func (s *simulator) result(accepted int) SimulationResult {
	result := SimulationResult{
		Protocol: Protocol, AcceptedRecords: strconv.Itoa(accepted),
		PermanentNullifierCount: strconv.Itoa(len(s.nullifiers)), ExpiryNotEvaluated: true,
		Effects: Effects{Scope: "RECORD_CONSTRUCTION_AND_OFFLINE_VALIDATION_ONLY", Authority: "NONE", Economic: "NONE", Reputation: "NONE"},
	}
	for _, state := range s.subjects {
		result.SubjectHeads = append(result.SubjectHeads, SubjectHead{
			Audience: state.Audience, SubjectRef: state.SubjectRef,
			Kind:          state.Kind,
			ControllerRef: state.ControllerRef, IssuerNamespace: state.IssuerNamespace,
			Sequence: strconv.FormatUint(state.Sequence, 10), Commitment: state.Commitment,
		})
	}
	sort.Slice(result.SubjectHeads, func(i, j int) bool {
		if result.SubjectHeads[i].Audience != result.SubjectHeads[j].Audience {
			return result.SubjectHeads[i].Audience < result.SubjectHeads[j].Audience
		}
		return result.SubjectHeads[i].SubjectRef < result.SubjectHeads[j].SubjectRef
	})
	for _, state := range s.controllers {
		entry := ControllerSummary{Audience: state.Audience, ControllerRef: state.ControllerRef, Namespace: state.Namespace, ActiveKeyFingerprint: state.Active, Revoked: state.Revoked}
		if state.Pending != "" {
			pending := state.Pending
			entry.PendingKeyFingerprint = &pending
		}
		result.Controllers = append(result.Controllers, entry)
	}
	sort.Slice(result.Controllers, func(i, j int) bool {
		if result.Controllers[i].Audience != result.Controllers[j].Audience {
			return result.Controllers[i].Audience < result.Controllers[j].Audience
		}
		return result.Controllers[i].ControllerRef < result.Controllers[j].ControllerRef
	})
	for _, state := range s.capabilities {
		result.Capabilities = append(result.Capabilities, CapabilitySummary{
			Audience: state.Audience, CapabilityRef: state.CapabilityRef, GrantCommitment: state.Grant,
			AssetRef: state.Asset, MaxPerConsumeMinor: strconv.FormatUint(state.MaxPerUse, 10),
			MaxTotalMinor: strconv.FormatUint(state.MaxTotal, 10), SpentMinor: strconv.FormatUint(state.Spent, 10),
			ConsumeCount: strconv.FormatUint(state.ConsumeCount, 10), Revoked: state.Revoked,
		})
	}
	sort.Slice(result.Capabilities, func(i, j int) bool {
		if result.Capabilities[i].Audience != result.Capabilities[j].Audience {
			return result.Capabilities[i].Audience < result.Capabilities[j].Audience
		}
		return result.Capabilities[i].CapabilityRef < result.Capabilities[j].CapabilityRef
	})
	if result.SubjectHeads == nil {
		result.SubjectHeads = []SubjectHead{}
	}
	if result.Controllers == nil {
		result.Controllers = []ControllerSummary{}
	}
	if result.Capabilities == nil {
		result.Capabilities = []CapabilitySummary{}
	}
	return result
}

func (result SimulationResult) CanonicalJSON() ([]byte, error) {
	encoded, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}
	return CanonicalJSON(encoded)
}
