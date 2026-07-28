package types

import "cosmossdk.io/errors"

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
