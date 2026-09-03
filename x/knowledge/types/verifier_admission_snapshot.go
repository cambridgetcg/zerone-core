package types

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"sort"

	"github.com/cosmos/cosmos-sdk/types/bech32"
)

const (
	// VerifierAdmissionSnapshotVersion domain-separates the source-only
	// verifier-admission contract. A future consensus integration must reject
	// every other version rather than guessing how its fields should be read.
	VerifierAdmissionSnapshotVersion = "zrn.verifier-admission.v1"

	// MaxVerifierAdmissionCandidates bounds canonicalisation work. The future
	// provider must return the complete reviewed source set, so truncation is
	// never an acceptable response to this limit.
	MaxVerifierAdmissionCandidates = 1_024

	ControllerReviewReviewed   ControllerReviewStatus = "reviewed"
	ControllerReviewUnreviewed ControllerReviewStatus = "unreviewed"

	VerifierBonded   VerifierBondStatus = "bonded"
	VerifierUnbonded VerifierBondStatus = "unbonded"

	DomainQualified   DomainQualificationStatus = "qualified"
	DomainUnqualified DomainQualificationStatus = "unqualified"

	VerifierAdmissionExclusionUnreviewedController = "unreviewed_controller"
	VerifierAdmissionExclusionUnbonded             = "unbonded"
	VerifierAdmissionExclusionZeroBondedStake      = "zero_bonded_stake"
	VerifierAdmissionExclusionUnqualified          = "unqualified"
	VerifierAdmissionExclusionZeroQualification    = "zero_qualification_weight"
)

// ControllerReviewStatus distinguishes a reviewed common-control conclusion
// from a known negative result. The empty value is unknown and always fails
// snapshot construction.
type ControllerReviewStatus string

// VerifierBondStatus is the reviewed SDK bonding state. It deliberately does
// not include balance, self-delegation, or virtual/effective selection stake.
// The empty value is unknown and always fails snapshot construction.
type VerifierBondStatus string

// DomainQualificationStatus is the reviewed qualification state at the
// snapshot height. The empty value is unknown and always fails construction.
type DomainQualificationStatus string

// VerifierAdmissionCandidateEvidence is one provider observation at one exact
// height. Pointer weights make "observed zero" different from "not observed";
// missing values fail closed instead of acquiring a minimum weight.
//
// ControllerIdentityHash is an opaque SHA-256 commitment to the reviewed
// controller identity. ProviderEvidenceHash commits the source records and
// membership proofs used to derive this observation. This package validates
// and commits those claims; it does not manufacture or attest them.
type VerifierAdmissionCandidateEvidence struct {
	ValidatorAddress       string
	ConsensusKeyHash       string
	ControllerIdentityHash string
	ControllerReviewStatus ControllerReviewStatus
	BondStatus             VerifierBondStatus
	BondedStake            *uint64
	QualificationStatus    DomainQualificationStatus
	QualificationWeight    *uint64
	ProviderEvidenceHash   string
}

// VerifierAdmissionSnapshotInput is the complete evidence envelope consumed
// by BuildVerifierAdmissionSnapshot. ObservedAppHash and InputEvidenceRoot
// bind the exact chain view and complete provider manifest at ObservedHeight.
type VerifierAdmissionSnapshotInput struct {
	RoundID           string
	ClaimID           string
	Domain            string
	ObservedHeight    uint64
	ObservedAppHash   string
	InputEvidenceRoot string
	MinimumSeats      uint32
	Candidates        []VerifierAdmissionCandidateEvidence
}

// VerifierAdmissionObservation is the canonical, auditable form of a provider
// observation. SelectionWeight is exactly the known SDK-bonded stake for an
// admitted seat and zero for every excluded observation.
type VerifierAdmissionObservation struct {
	ValidatorAddress       string                    `json:"validator_address"`
	ConsensusKeyHash       string                    `json:"consensus_key_hash"`
	ControllerIdentityHash string                    `json:"controller_identity_hash"`
	ControllerReviewStatus ControllerReviewStatus    `json:"controller_review_status"`
	BondStatus             VerifierBondStatus        `json:"bond_status"`
	BondedStake            uint64                    `json:"bonded_stake"`
	QualificationStatus    DomainQualificationStatus `json:"qualification_status"`
	QualificationWeight    uint64                    `json:"qualification_weight"`
	ProviderEvidenceHash   string                    `json:"provider_evidence_hash"`
	Admitted               bool                      `json:"admitted"`
	ExclusionReason        string                    `json:"exclusion_reason,omitempty"`
	SelectionWeight        uint64                    `json:"selection_weight"`
}

// VerifierAdmissionSeat is the immutable one-controller/one-validator panel
// seat projected from an admitted observation.
type VerifierAdmissionSeat struct {
	ValidatorAddress       string `json:"validator_address"`
	ConsensusKeyHash       string `json:"consensus_key_hash"`
	ControllerIdentityHash string `json:"controller_identity_hash"`
	BondedStake            uint64 `json:"bonded_stake"`
	QualificationWeight    uint64 `json:"qualification_weight"`
	SelectionWeight        uint64 `json:"selection_weight"`
}

// VerifierAdmissionSnapshot is a deterministic source-only artifact. It is
// not consensus state until a separately reviewed release persists it in a
// VerificationRound and rechecks it on both commit and reveal admission.
// SnapshotHash detects any mutation to its height, chain evidence, domain,
// candidate observations, exclusions, or seats.
type VerifierAdmissionSnapshot struct {
	Version           string                         `json:"version"`
	RoundID           string                         `json:"round_id"`
	ClaimID           string                         `json:"claim_id"`
	Domain            string                         `json:"domain"`
	ObservedHeight    uint64                         `json:"observed_height"`
	ObservedAppHash   string                         `json:"observed_app_hash"`
	InputEvidenceRoot string                         `json:"input_evidence_root"`
	MinimumSeats      uint32                         `json:"minimum_seats"`
	Observations      []VerifierAdmissionObservation `json:"observations"`
	Seats             []VerifierAdmissionSeat        `json:"seats"`
	SnapshotHash      string                         `json:"snapshot_hash"`
}

// BuildVerifierAdmissionSnapshot validates and canonicalises a complete
// provider observation set. It never falls back to balance, unqualified
// validators, unknown evidence, a weight floor, or multiple seats controlled
// by one reviewed identity.
func BuildVerifierAdmissionSnapshot(input VerifierAdmissionSnapshotInput) (*VerifierAdmissionSnapshot, error) {
	if input.RoundID == "" {
		return nil, fmt.Errorf("round_id is required")
	}
	if input.ClaimID == "" {
		return nil, fmt.Errorf("claim_id is required")
	}
	if input.Domain == "" {
		return nil, fmt.Errorf("domain is required")
	}
	if input.ObservedHeight == 0 {
		return nil, fmt.Errorf("observed_height must be positive")
	}
	if err := ValidateSHA256Hex("observed_app_hash", input.ObservedAppHash); err != nil {
		return nil, err
	}
	if err := ValidateSHA256Hex("input_evidence_root", input.InputEvidenceRoot); err != nil {
		return nil, err
	}
	if input.MinimumSeats == 0 {
		return nil, fmt.Errorf("minimum_seats must be positive")
	}
	if len(input.Candidates) == 0 {
		return nil, fmt.Errorf("candidate evidence is required")
	}
	if len(input.Candidates) > MaxVerifierAdmissionCandidates {
		return nil, fmt.Errorf("candidate evidence count %d exceeds hard cap %d", len(input.Candidates), MaxVerifierAdmissionCandidates)
	}

	candidates := append([]VerifierAdmissionCandidateEvidence(nil), input.Candidates...)
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].ValidatorAddress < candidates[j].ValidatorAddress
	})

	observations := make([]VerifierAdmissionObservation, 0, len(candidates))
	seats := make([]VerifierAdmissionSeat, 0, len(candidates))
	seenValidators := make(map[string]struct{}, len(candidates))
	seenConsensusKeys := make(map[string]string, len(candidates))
	seenControllers := make(map[string]string, len(candidates))
	for i := range candidates {
		candidate := candidates[i]
		if err := validateVerifierAdmissionCandidate(i, candidate); err != nil {
			return nil, err
		}
		if _, exists := seenValidators[candidate.ValidatorAddress]; exists {
			return nil, fmt.Errorf("duplicate validator_address %q", candidate.ValidatorAddress)
		}
		seenValidators[candidate.ValidatorAddress] = struct{}{}
		if first, exists := seenConsensusKeys[candidate.ConsensusKeyHash]; exists {
			return nil, fmt.Errorf(
				"consensus_key_hash %s ambiguously identifies validators %s and %s",
				candidate.ConsensusKeyHash,
				first,
				candidate.ValidatorAddress,
			)
		}
		seenConsensusKeys[candidate.ConsensusKeyHash] = candidate.ValidatorAddress
		if first, exists := seenControllers[candidate.ControllerIdentityHash]; exists {
			return nil, fmt.Errorf(
				"controller_identity_hash %s ambiguously controls validators %s and %s",
				candidate.ControllerIdentityHash,
				first,
				candidate.ValidatorAddress,
			)
		}
		seenControllers[candidate.ControllerIdentityHash] = candidate.ValidatorAddress

		observation := canonicalAdmissionObservation(candidate)
		observations = append(observations, observation)
		if observation.Admitted {
			seats = append(seats, seatFromAdmissionObservation(observation))
		}
	}

	if uint32(len(seats)) < input.MinimumSeats {
		return nil, fmt.Errorf(
			"admitted verifier seats %d below required minimum %d",
			len(seats),
			input.MinimumSeats,
		)
	}

	snapshot := &VerifierAdmissionSnapshot{
		Version:           VerifierAdmissionSnapshotVersion,
		RoundID:           input.RoundID,
		ClaimID:           input.ClaimID,
		Domain:            input.Domain,
		ObservedHeight:    input.ObservedHeight,
		ObservedAppHash:   input.ObservedAppHash,
		InputEvidenceRoot: input.InputEvidenceRoot,
		MinimumSeats:      input.MinimumSeats,
		Observations:      observations,
		Seats:             seats,
	}
	snapshot.SnapshotHash = computeVerifierAdmissionSnapshotHash(snapshot)
	return snapshot, nil
}

// VerifyVerifierAdmissionSnapshot recomputes all admission decisions and the
// canonical digest. A future store/read boundary should call this before
// relying on a persisted snapshot.
func VerifyVerifierAdmissionSnapshot(snapshot *VerifierAdmissionSnapshot) error {
	if snapshot == nil {
		return fmt.Errorf("verifier admission snapshot is required")
	}
	if snapshot.Version != VerifierAdmissionSnapshotVersion {
		return fmt.Errorf("unsupported verifier admission snapshot version %q", snapshot.Version)
	}
	if snapshot.RoundID == "" || snapshot.ClaimID == "" || snapshot.Domain == "" {
		return fmt.Errorf("snapshot round_id, claim_id, and domain are required")
	}
	if snapshot.ObservedHeight == 0 {
		return fmt.Errorf("snapshot observed_height must be positive")
	}
	if err := ValidateSHA256Hex("observed_app_hash", snapshot.ObservedAppHash); err != nil {
		return err
	}
	if err := ValidateSHA256Hex("input_evidence_root", snapshot.InputEvidenceRoot); err != nil {
		return err
	}
	if snapshot.MinimumSeats == 0 {
		return fmt.Errorf("snapshot minimum_seats must be positive")
	}
	if len(snapshot.Observations) == 0 || len(snapshot.Observations) > MaxVerifierAdmissionCandidates {
		return fmt.Errorf("snapshot observation count %d is outside 1..%d", len(snapshot.Observations), MaxVerifierAdmissionCandidates)
	}

	derivedSeats := make([]VerifierAdmissionSeat, 0, len(snapshot.Observations))
	seenConsensusKeys := make(map[string]string, len(snapshot.Observations))
	seenControllers := make(map[string]string, len(snapshot.Observations))
	previousValidator := ""
	for i := range snapshot.Observations {
		observation := snapshot.Observations[i]
		candidate := candidateFromAdmissionObservation(observation)
		if err := validateVerifierAdmissionCandidate(i, candidate); err != nil {
			return fmt.Errorf("invalid canonical observation: %w", err)
		}
		if i > 0 && observation.ValidatorAddress <= previousValidator {
			return fmt.Errorf("snapshot observations are not in strict validator_address order")
		}
		previousValidator = observation.ValidatorAddress
		if first, exists := seenConsensusKeys[observation.ConsensusKeyHash]; exists {
			return fmt.Errorf(
				"snapshot consensus_key_hash %s identifies validators %s and %s",
				observation.ConsensusKeyHash,
				first,
				observation.ValidatorAddress,
			)
		}
		seenConsensusKeys[observation.ConsensusKeyHash] = observation.ValidatorAddress
		if first, exists := seenControllers[observation.ControllerIdentityHash]; exists {
			return fmt.Errorf(
				"snapshot controller_identity_hash %s controls validators %s and %s",
				observation.ControllerIdentityHash,
				first,
				observation.ValidatorAddress,
			)
		}
		seenControllers[observation.ControllerIdentityHash] = observation.ValidatorAddress

		derived := canonicalAdmissionObservation(candidate)
		if observation != derived {
			return fmt.Errorf("snapshot observation for validator %s has a non-canonical admission decision", observation.ValidatorAddress)
		}
		if derived.Admitted {
			derivedSeats = append(derivedSeats, seatFromAdmissionObservation(derived))
		}
	}
	if uint32(len(derivedSeats)) < snapshot.MinimumSeats {
		return fmt.Errorf("snapshot admitted verifier seats %d below required minimum %d", len(derivedSeats), snapshot.MinimumSeats)
	}
	if !equalAdmissionSeats(snapshot.Seats, derivedSeats) {
		return fmt.Errorf("snapshot seats do not equal canonical admitted observations")
	}
	if err := ValidateSHA256Hex("snapshot_hash", snapshot.SnapshotHash); err != nil {
		return err
	}
	if want := computeVerifierAdmissionSnapshotHash(snapshot); snapshot.SnapshotHash != want {
		return fmt.Errorf("snapshot_hash mismatch: got %s want %s", snapshot.SnapshotHash, want)
	}
	return nil
}

func validateVerifierAdmissionCandidate(index int, candidate VerifierAdmissionCandidateEvidence) error {
	if err := validateCanonicalZeroneAdmissionAddress(candidate.ValidatorAddress); err != nil {
		return fmt.Errorf("candidate %d validator_address: %w", index, err)
	}
	for _, field := range []struct {
		name  string
		value string
	}{
		{"consensus_key_hash", candidate.ConsensusKeyHash},
		{"controller_identity_hash", candidate.ControllerIdentityHash},
		{"provider_evidence_hash", candidate.ProviderEvidenceHash},
	} {
		if err := ValidateSHA256Hex(field.name, field.value); err != nil {
			return fmt.Errorf("candidate %d: %w", index, err)
		}
	}
	switch candidate.ControllerReviewStatus {
	case ControllerReviewReviewed, ControllerReviewUnreviewed:
	default:
		return fmt.Errorf("candidate %d controller review status %q is unknown", index, candidate.ControllerReviewStatus)
	}
	switch candidate.BondStatus {
	case VerifierBonded, VerifierUnbonded:
	default:
		return fmt.Errorf("candidate %d bond status %q is unknown", index, candidate.BondStatus)
	}
	if candidate.BondedStake == nil {
		return fmt.Errorf("candidate %d bonded stake is unknown", index)
	}
	if candidate.BondStatus == VerifierUnbonded && *candidate.BondedStake != 0 {
		return fmt.Errorf("candidate %d unbonded observation has nonzero SDK-bonded stake %d", index, *candidate.BondedStake)
	}
	switch candidate.QualificationStatus {
	case DomainQualified, DomainUnqualified:
	default:
		return fmt.Errorf("candidate %d qualification status %q is unknown", index, candidate.QualificationStatus)
	}
	if candidate.QualificationWeight == nil {
		return fmt.Errorf("candidate %d qualification weight is unknown", index)
	}
	if candidate.QualificationStatus == DomainUnqualified && *candidate.QualificationWeight != 0 {
		return fmt.Errorf("candidate %d unqualified observation has nonzero qualification weight %d", index, *candidate.QualificationWeight)
	}
	return nil
}

func canonicalAdmissionObservation(candidate VerifierAdmissionCandidateEvidence) VerifierAdmissionObservation {
	observation := VerifierAdmissionObservation{
		ValidatorAddress:       candidate.ValidatorAddress,
		ConsensusKeyHash:       candidate.ConsensusKeyHash,
		ControllerIdentityHash: candidate.ControllerIdentityHash,
		ControllerReviewStatus: candidate.ControllerReviewStatus,
		BondStatus:             candidate.BondStatus,
		BondedStake:            *candidate.BondedStake,
		QualificationStatus:    candidate.QualificationStatus,
		QualificationWeight:    *candidate.QualificationWeight,
		ProviderEvidenceHash:   candidate.ProviderEvidenceHash,
	}
	switch {
	case candidate.ControllerReviewStatus != ControllerReviewReviewed:
		observation.ExclusionReason = VerifierAdmissionExclusionUnreviewedController
	case candidate.BondStatus != VerifierBonded:
		observation.ExclusionReason = VerifierAdmissionExclusionUnbonded
	case *candidate.BondedStake == 0:
		observation.ExclusionReason = VerifierAdmissionExclusionZeroBondedStake
	case candidate.QualificationStatus != DomainQualified:
		observation.ExclusionReason = VerifierAdmissionExclusionUnqualified
	case *candidate.QualificationWeight == 0:
		observation.ExclusionReason = VerifierAdmissionExclusionZeroQualification
	default:
		observation.Admitted = true
		observation.SelectionWeight = *candidate.BondedStake
	}
	return observation
}

func candidateFromAdmissionObservation(observation VerifierAdmissionObservation) VerifierAdmissionCandidateEvidence {
	bondedStake := observation.BondedStake
	qualificationWeight := observation.QualificationWeight
	return VerifierAdmissionCandidateEvidence{
		ValidatorAddress:       observation.ValidatorAddress,
		ConsensusKeyHash:       observation.ConsensusKeyHash,
		ControllerIdentityHash: observation.ControllerIdentityHash,
		ControllerReviewStatus: observation.ControllerReviewStatus,
		BondStatus:             observation.BondStatus,
		BondedStake:            &bondedStake,
		QualificationStatus:    observation.QualificationStatus,
		QualificationWeight:    &qualificationWeight,
		ProviderEvidenceHash:   observation.ProviderEvidenceHash,
	}
}

func seatFromAdmissionObservation(observation VerifierAdmissionObservation) VerifierAdmissionSeat {
	return VerifierAdmissionSeat{
		ValidatorAddress:       observation.ValidatorAddress,
		ConsensusKeyHash:       observation.ConsensusKeyHash,
		ControllerIdentityHash: observation.ControllerIdentityHash,
		BondedStake:            observation.BondedStake,
		QualificationWeight:    observation.QualificationWeight,
		SelectionWeight:        observation.SelectionWeight,
	}
}

func equalAdmissionSeats(left, right []VerifierAdmissionSeat) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

func validateCanonicalZeroneAdmissionAddress(address string) error {
	hrp, addressBytes, err := bech32.DecodeAndConvert(address)
	if err != nil {
		return fmt.Errorf("invalid bech32: %w", err)
	}
	if hrp != "zrn" {
		return fmt.Errorf("prefix must be zrn, got %q", hrp)
	}
	if len(addressBytes) != 20 {
		return fmt.Errorf("decoded length must be 20 bytes, got %d", len(addressBytes))
	}
	canonical, err := bech32.ConvertAndEncode("zrn", addressBytes)
	if err != nil {
		return fmt.Errorf("cannot canonicalise address: %w", err)
	}
	if address != canonical {
		return fmt.Errorf("address must use canonical lowercase bech32 encoding")
	}
	return nil
}

func computeVerifierAdmissionSnapshotHash(snapshot *VerifierAdmissionSnapshot) string {
	h := sha256.New()
	h.Write([]byte("ZRN.verifier-admission-snapshot.v1\x00"))
	writeAdmissionString(h, snapshot.Version)
	writeAdmissionString(h, snapshot.RoundID)
	writeAdmissionString(h, snapshot.ClaimID)
	writeAdmissionString(h, snapshot.Domain)
	writeAdmissionUint64(h, snapshot.ObservedHeight)
	writeAdmissionString(h, snapshot.ObservedAppHash)
	writeAdmissionString(h, snapshot.InputEvidenceRoot)
	writeAdmissionUint32(h, snapshot.MinimumSeats)
	writeAdmissionUint32(h, uint32(len(snapshot.Observations)))
	for _, observation := range snapshot.Observations {
		writeAdmissionString(h, observation.ValidatorAddress)
		writeAdmissionString(h, observation.ConsensusKeyHash)
		writeAdmissionString(h, observation.ControllerIdentityHash)
		writeAdmissionString(h, string(observation.ControllerReviewStatus))
		writeAdmissionString(h, string(observation.BondStatus))
		writeAdmissionUint64(h, observation.BondedStake)
		writeAdmissionString(h, string(observation.QualificationStatus))
		writeAdmissionUint64(h, observation.QualificationWeight)
		writeAdmissionString(h, observation.ProviderEvidenceHash)
		if observation.Admitted {
			h.Write([]byte{1})
		} else {
			h.Write([]byte{0})
		}
		writeAdmissionString(h, observation.ExclusionReason)
		writeAdmissionUint64(h, observation.SelectionWeight)
	}
	writeAdmissionUint32(h, uint32(len(snapshot.Seats)))
	for _, seat := range snapshot.Seats {
		writeAdmissionString(h, seat.ValidatorAddress)
		writeAdmissionString(h, seat.ConsensusKeyHash)
		writeAdmissionString(h, seat.ControllerIdentityHash)
		writeAdmissionUint64(h, seat.BondedStake)
		writeAdmissionUint64(h, seat.QualificationWeight)
		writeAdmissionUint64(h, seat.SelectionWeight)
	}
	return hex.EncodeToString(h.Sum(nil))
}

type admissionHashWriter interface {
	Write([]byte) (int, error)
}

func writeAdmissionString(writer admissionHashWriter, value string) {
	writeAdmissionUint64(writer, uint64(len(value)))
	_, _ = writer.Write([]byte(value))
}

func writeAdmissionUint32(writer admissionHashWriter, value uint32) {
	var encoded [4]byte
	binary.BigEndian.PutUint32(encoded[:], value)
	_, _ = writer.Write(encoded[:])
}

func writeAdmissionUint64(writer admissionHashWriter, value uint64) {
	var encoded [8]byte
	binary.BigEndian.PutUint64(encoded[:], value)
	_, _ = writer.Write(encoded[:])
}
