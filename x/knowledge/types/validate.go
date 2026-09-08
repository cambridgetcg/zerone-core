package types

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"cosmossdk.io/errors"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// MaxRelationsPerClaim caps the typed relations one claim may carry. Every
// relation costs per-relation work downstream (validation, karma edge events),
// so an unbounded slice would make that work unbounded per tx.
// K-beta paramifies this (DoR A-3).
const MaxRelationsPerClaim = 16

// CommitSeatHardCap bounds a verification round's commit list on the tx path
// (SubmitCommitment). It is a state-growth and BeginBlocker-work backstop,
// deliberately far above any quorum (mainnet effective minimum ≤ 6, mainnet
// MaxVerifiers = 22) — NOT the design §2.6 tight seat cap. A tight cap at
// MaxVerifiers cannot ship before C-2's seat bonds: without a bond, seats are
// free to hold (the 100-ZRN gate is a recyclable balance snapshot), so 22
// addresses could fill every seat the block a round opens, never reveal, and
// expire every claim on the chain INCONCLUSIVE for one tx fee each. At this
// backstop the same exclusion needs CommitSeatHardCap funded addresses
// winning the inclusion race in every commit window against a per-commit
// round rewrite whose gas cost grows with the commit list, while honest
// verifiers need one seat each. The vote-extension write path is bounded
// separately by VRF selection over the registered validator set (app/abci.go
// verifies selection before storing) and is not gated by this constant.
// K-beta paramifies this alongside the C-2 seat bonds.
const CommitSeatHardCap = 512

const (
	MaxChallengeEvidenceIDs = 16
	MaxChallengeReasonBytes = 4_096
	MaxFactRatingMemoBytes  = 256
)

// ValidateChallengeEvidenceIDs is stateless. The consensus challenge handlers
// must also check that each evidence fact exists, before writing any claim.
// These IDs must never be copied into Claim.References.
func ValidateChallengeEvidenceIDs(ids []string) error {
	if len(ids) > MaxChallengeEvidenceIDs {
		return fmt.Errorf("challenge evidence exceeds %d fact IDs", MaxChallengeEvidenceIDs)
	}
	seen := make(map[string]bool, len(ids))
	for _, id := range ids {
		if err := ValidateFactUseFactID(id); err != nil {
			return fmt.Errorf("invalid challenge evidence: %w", err)
		}
		if seen[id] {
			return fmt.Errorf("duplicate challenge evidence fact ID")
		}
		seen[id] = true
	}
	return nil
}

func ValidateChallengeReason(reason string) error {
	if len(reason) > MaxChallengeReasonBytes || !utf8.ValidString(reason) || strings.TrimSpace(reason) == "" {
		return fmt.Errorf("challenge reason must be nonblank UTF-8, at most %d bytes", MaxChallengeReasonBytes)
	}
	return nil
}

func validateFactUseIdentity(consumer, factID string) error {
	if _, err := CanonicalFactUseConsumer(consumer); err != nil {
		return err
	}
	return ValidateFactUseFactID(factID)
}

// ValidateBasic does not establish cohort admission, fact existence, quota or
// signature validity. The consensus handler must run it too, then check state.
func (msg *MsgReportFactUse) ValidateBasic() error {
	if msg == nil {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "nil fact-use report")
	}
	if err := validateFactUseIdentity(msg.Consumer, msg.FactId); err != nil {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, err.Error())
	}
	return nil
}

func (msg *MsgRateFact) ValidateBasic() error {
	if msg == nil {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "nil fact rating")
	}
	if err := validateFactUseIdentity(msg.Rater, msg.FactId); err != nil {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, err.Error())
	}
	if len(msg.Memo) > MaxFactRatingMemoBytes || !utf8.ValidString(msg.Memo) {
		return errors.Wrapf(sdkerrors.ErrInvalidRequest, "rating memo must be UTF-8, at most %d bytes", MaxFactRatingMemoBytes)
	}
	return nil
}

func validateChallenge(reason string, evidenceIDs []string) error {
	if err := ValidateChallengeReason(reason); err != nil {
		return errors.Wrap(ErrInvalidChallenge, err.Error())
	}
	if err := ValidateChallengeEvidenceIDs(evidenceIDs); err != nil {
		return errors.Wrap(ErrInvalidChallenge, err.Error())
	}
	return nil
}

func (msg *MsgChallengeFact) ValidateBasic() error {
	if msg == nil {
		return errors.Wrap(ErrInvalidChallenge, "nil challenge")
	}
	if err := ValidateFactUseFactID(msg.FactId); err != nil {
		return errors.Wrap(ErrInvalidChallenge, err.Error())
	}
	return validateChallenge(msg.Reason, msg.EvidenceIds)
}

func (msg *MsgChallengeProvisionalFact) ValidateBasic() error {
	if msg == nil {
		return errors.Wrap(ErrInvalidChallenge, "nil provisional challenge")
	}
	if err := ValidateFactUseFactID(msg.FactId); err != nil {
		return errors.Wrap(ErrInvalidChallenge, err.Error())
	}
	return validateChallenge(msg.Reason, msg.EvidenceIds)
}

// ValidateBasic rejects claims carrying more than MaxRelationsPerClaim
// relations. Stateless; cosmos-sdk v0.50 baseapp calls it on any message
// implementing sdk.HasValidateBasic (validateBasicTxMsgs), as does this
// app's ProcessProposal (app/abci.go).
//
// MsgSubmitClaim is the only knowledge Msg carrying a repeated ClaimRelation
// field; MsgSubmitContradiction and the rest carry none (their []string
// citation fields are not relation edges).
func (msg *MsgSubmitClaim) ValidateBasic() error {
	if len(msg.Relations) > MaxRelationsPerClaim {
		return errors.Wrapf(ErrInvalidClaim,
			"claim carries %d relations, max %d", len(msg.Relations), MaxRelationsPerClaim)
	}
	return nil
}
