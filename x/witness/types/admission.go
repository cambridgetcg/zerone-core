package types

import (
	"fmt"
	"regexp"
	"sort"

	core "github.com/zerone-chain/zerone/tools/witness-v0/protocol"
)

const (
	Protocol       = core.Protocol
	MaxRecordBytes = 32 * 1024

	ActivationStatusNotConsensusAdmissible = core.ActivationStatusNotConsensusAdmissible
)

type Kind = core.Kind
type Action = core.Action

const (
	KindKingdomReleaseRoot         = core.KindKingdomReleaseRoot
	KindAgentToolSettlementRoot    = core.KindAgentToolSettlementRoot
	KindAgentToolCapability        = core.KindAgentToolCapability
	KindAgentToolPublicRecognition = core.KindAgentToolPublicRecognition
	KindAgentToolOffer             = core.KindAgentToolOffer
	KindWakePublicCheckpoint       = core.KindWakePublicCheckpoint
	KindIssuerKeyContinuity        = core.KindIssuerKeyContinuity
	KindArtifactLineage            = core.KindArtifactLineage
	KindCollaborationCheckpoint    = core.KindCollaborationCheckpoint
	KindDisputeTerminal            = core.KindDisputeTerminal
)

const (
	ActionCheckpoint = core.ActionCheckpoint
	ActionGrant      = core.ActionGrant
	ActionConsume    = core.ActionConsume
	ActionRevoke     = core.ActionRevoke
	ActionAdopt      = core.ActionAdopt
	ActionWithdraw   = core.ActionWithdraw
	ActionSupersede  = core.ActionSupersede
	ActionSettle     = core.ActionSettle
	ActionRotate     = core.ActionRotate
	ActionPublish    = core.ActionPublish
)

type ActionPair struct {
	Kind   Kind
	Action Action
}

type AdmissionConfig struct {
	// Enabled is false in the zero value. No genesis or app wiring exists that
	// can change it in the current scaffold.
	Enabled bool
	// ChainID and Audience must both be supplied by the caller. Audience must
	// equal the exact replay domain "zerone:" + ChainID.
	ChainID  string
	Audience string
}

type VerifiedRecord struct {
	Protocol      string
	Kind          Kind
	Action        Action
	Audience      string
	SubjectRef    string
	ControllerRef string
	Commitment    string
}

type ActivationReadiness = core.ActivationReadiness

var chainIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,95}$`)

func DefaultAdmissionConfig() AdmissionConfig { return AdmissionConfig{} }

func ExpectedAudience(chainID string) (string, error) {
	if !chainIDPattern.MatchString(chainID) {
		return "", fmt.Errorf("chain ID must match %s", chainIDPattern.String())
	}
	return "zerone:" + chainID, nil
}

func (config AdmissionConfig) ValidateAudienceBinding() error {
	expected, err := ExpectedAudience(config.ChainID)
	if err != nil {
		return err
	}
	if config.Audience != expected {
		return fmt.Errorf("audience must be exactly %q", expected)
	}
	return nil
}

// IsAllowedAction projects the frozen core's defensive action matrix; the
// carrier owns no second consensus table.
func IsAllowedAction(kind Kind, action Action) bool {
	for _, candidate := range core.KindActionMatrix()[kind] {
		if candidate == action {
			return true
		}
	}
	return false
}

// ClosedActionPairs deterministically projects the frozen core matrix.
func ClosedActionPairs() []ActionPair {
	matrix := core.KindActionMatrix()
	pairs := make([]ActionPair, 0, 18)
	for kind, actions := range matrix {
		for _, action := range actions {
			pairs = append(pairs, ActionPair{Kind: kind, Action: action})
		}
	}
	sort.Slice(pairs, func(i, j int) bool {
		if pairs[i].Kind != pairs[j].Kind {
			return pairs[i].Kind < pairs[j].Kind
		}
		return pairs[i].Action < pairs[j].Action
	})
	return pairs
}

// CurrentActivationReadiness projects the frozen core's defensive readiness
// matrix; callers cannot inject or mutate an activation override.
func CurrentActivationReadiness(kind Kind) (ActivationReadiness, bool) {
	for _, readiness := range core.ActivationReadinessMatrix() {
		if readiness.Kind == kind {
			return readiness, true
		}
	}
	return ActivationReadiness{}, false
}

func CurrentActivationReadinessMatrix() []ActivationReadiness {
	return core.ActivationReadinessMatrix()
}
