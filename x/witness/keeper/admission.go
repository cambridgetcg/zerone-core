package keeper

import (
	"context"
	"errors"
	"fmt"

	core "github.com/zerone-chain/zerone/tools/witness-v0/protocol"
	witnesstypes "github.com/zerone-chain/zerone/x/witness/types"
)

var (
	ErrDisabled                 = errors.New("witness consensus carrier is disabled")
	ErrEmptyRecord              = errors.New("witness record is empty")
	ErrRecordTooLarge           = errors.New("witness record exceeds 32 KiB")
	ErrAudienceConfiguration    = errors.New("invalid witness audience configuration")
	ErrVerifierRequired         = errors.New("explicit deterministic verifier is required")
	ErrControllerPolicyRequired = errors.New("explicit controller policy is required")
	ErrStateMutatorRequired     = errors.New("explicit state mutator is required")
	ErrVerification             = errors.New("witness verification failed")
	ErrProtocolMismatch         = errors.New("witness protocol mismatch")
	ErrAudienceMismatch         = errors.New("witness audience mismatch")
	ErrUnsupportedKindAction    = errors.New("unsupported witness kind/action")
	ErrNotConsensusAdmissible   = errors.New("witness kind is not consensus admissible")
	ErrDormantInvariant         = errors.New("dormant witness scaffold has no admission path")
)

// VerifyFunc must perform exact canonical-byte, schema, commitment, and strict
// signature verification deterministically.
type VerifyFunc func(record []byte) (witnesstypes.VerifiedRecord, error)

// ControllerPolicyFunc is mandatory explicit configuration, but is
// deliberately unreachable while every kind is NOT_CONSENSUS_ADMISSIBLE.
type ControllerPolicyFunc func(context.Context, witnesstypes.VerifiedRecord) error

// StateMutatorFunc is a future integration seam only. It is deliberately
// unreachable in this source-only scaffold. No store implementation exists.
type StateMutatorFunc func(context.Context, witnesstypes.VerifiedRecord) error

type Decision struct {
	Admitted bool
	Kind     witnesstypes.Kind
	Action   witnesstypes.Action
	Audience string
	Status   string
	Blockers []string
}

type Admitter struct {
	config           witnesstypes.AdmissionConfig
	verify           VerifyFunc
	controllerPolicy ControllerPolicyFunc
	mutateState      StateMutatorFunc
}

// NewAdmitter refuses implicit dependencies. The returned value still cannot
// admit a record because the frozen core readiness matrix blocks every kind.
func NewAdmitter(config witnesstypes.AdmissionConfig, verify VerifyFunc, controllerPolicy ControllerPolicyFunc, mutateState StateMutatorFunc) (Admitter, error) {
	if verify == nil {
		return Admitter{}, ErrVerifierRequired
	}
	if controllerPolicy == nil {
		return Admitter{}, ErrControllerPolicyRequired
	}
	if mutateState == nil {
		return Admitter{}, ErrStateMutatorRequired
	}
	return Admitter{config: config, verify: verify, controllerPolicy: controllerPolicy, mutateState: mutateState}, nil
}

// VerifyFrozenCore is the only supplied verifier adapter. It delegates all
// canonical, schema, commitment, signature, and payload semantics to the
// frozen core rather than duplicating consensus rules under x/witness.
func VerifyFrozenCore(record []byte) (witnesstypes.VerifiedRecord, error) {
	verified, err := core.Verify(record)
	if err != nil {
		return witnesstypes.VerifiedRecord{}, err
	}
	return witnesstypes.VerifiedRecord{
		Protocol:      verified.Record.Envelope.Protocol,
		Kind:          verified.Record.Envelope.Kind,
		Action:        verified.Record.Envelope.Action,
		Audience:      verified.Record.Envelope.Audience,
		SubjectRef:    verified.Record.Envelope.SubjectRef,
		ControllerRef: verified.Record.Envelope.Issuer.ControllerRef,
		Commitment:    verified.Record.Commitment,
	}, nil
}

func (admitter Admitter) Admit(_ context.Context, record []byte) (Decision, error) {
	if !admitter.config.Enabled {
		return Decision{}, ErrDisabled
	}
	if len(record) == 0 {
		return Decision{}, ErrEmptyRecord
	}
	if len(record) > witnesstypes.MaxRecordBytes {
		return Decision{}, ErrRecordTooLarge
	}
	if err := admitter.config.ValidateAudienceBinding(); err != nil {
		return Decision{}, fmt.Errorf("%w: %v", ErrAudienceConfiguration, err)
	}

	verified, err := admitter.verify(append([]byte(nil), record...))
	if err != nil {
		return Decision{}, fmt.Errorf("%w: %v", ErrVerification, err)
	}
	decision := Decision{Kind: verified.Kind, Action: verified.Action, Audience: verified.Audience}
	if verified.Protocol != witnesstypes.Protocol {
		return decision, ErrProtocolMismatch
	}
	if verified.Audience != admitter.config.Audience {
		return decision, fmt.Errorf("%w: got %q, want %q", ErrAudienceMismatch, verified.Audience, admitter.config.Audience)
	}
	if !witnesstypes.IsAllowedAction(verified.Kind, verified.Action) {
		return decision, fmt.Errorf("%w: %s/%s", ErrUnsupportedKindAction, verified.Kind, verified.Action)
	}
	readiness, ok := witnesstypes.CurrentActivationReadiness(verified.Kind)
	if !ok {
		return decision, fmt.Errorf("%w: no readiness entry for %s", ErrUnsupportedKindAction, verified.Kind)
	}
	decision.Status = readiness.Status
	decision.Blockers = append([]string(nil), readiness.Blockers...)
	if readiness.Status != "CONSENSUS_ADMISSIBLE" {
		return decision, fmt.Errorf("%w: %s/%s status=%s", ErrNotConsensusAdmissible, verified.Kind, verified.Action, readiness.Status)
	}

	// This invariant is intentionally unconditional in v0. There is no call to
	// controllerPolicy or mutateState anywhere in the dormant source tree.
	// Future activation requires a separately reviewed change.
	return decision, ErrDormantInvariant
}
