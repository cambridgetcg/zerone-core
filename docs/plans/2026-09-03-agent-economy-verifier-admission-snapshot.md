# Agent-economy verifier-admission snapshot contract

Status: **SOURCE-ONLY CONTRACT — NOT WIRED TO CONSENSUS**

This cut defines and tests the smallest truthful boundary needed before a
computational verification panel can be made economic. It does not activate
the agent economy, write a migration marker, register an upgrade handler,
change a `VerificationRound`, accept a commitment, select a validator, or move
ZRN.

The implementation is
`x/knowledge/types/verifier_admission_snapshot.go`. It deliberately accepts
reviewed evidence as input rather than deriving controller or bonding claims
from the current keepers. The current interfaces cannot supply those facts
without pretending:

- `knowledge.StakingKeeper` exposes effective stake, which includes virtual
  selection stake, not an exact SDK-bonded stake observation;
- neither staking nor `zerone_auth` exposes a reviewed common-controller
  record. A validator DID or account DID is not by itself proof that two
  operator accounts are independently controlled;
- the qualification adapter returns convenient booleans and weights but no
  authenticated record/proof or exact observed AppHash; and
- the vote-extension path remains source-latched off because a self-declared
  operator address is not bound to the authenticated CometBFT voter and the
  VRF proof is not checked against that voter’s consensus public key.

## Contract

`BuildVerifierAdmissionSnapshot` consumes one complete provider manifest for
one round, claim, domain, block height, and AppHash. Its digest binds all of
those values, the manifest evidence root, every per-validator evidence hash,
every eligibility observation, every exclusion, and every admitted seat.

Construction is deterministic across provider iteration order. Candidate
observations are sorted by canonical lowercase Zerone operator address. The
builder rejects:

- an absent height, AppHash, manifest root, consensus-key hash, controller
  hash, or provider evidence hash;
- unknown controller-review, SDK-bonding, stake, domain-qualification, or
  qualification-weight state;
- contradictory unbonded/nonzero-SDK-bonded or unqualified/nonzero-weight
  observations;
- malformed or noncanonical operator addresses and digests;
- duplicate validator observations or consensus-key identities;
- two validator addresses claiming the same reviewed controller identity; and
- a resulting reviewed panel smaller than the required quorum.

Known negative states are retained as evidence but receive zero selection
weight. This includes an unreviewed controller, an unbonded validator, known
zero SDK-bonded stake, an unqualified validator, and known zero qualification
weight. An admitted seat’s selection weight is exactly its observed
SDK-bonded stake. There is no balance, virtual-stake, or minimum-weight
fallback.

`VerifyVerifierAdmissionSnapshot` re-derives all decisions, seat projection,
ordering, and the snapshot digest. A future persistence/read boundary must run
that verifier before relying on the record; changing the height, domain,
AppHash, source evidence, stake, qualification, decision, order, or seat makes
the artifact invalid.

## Required future wiring

This contract is not sufficient to open the activation marker. A later
release still needs all of the following as one reviewed consensus change:

1. Add an authoritative provider that reads a complete, authenticated state
   view at the current round-creation height and returns the exact AppHash and
   complete manifest root. Store-read, decode, proof, iterator, or close
   failures must abort the round.
2. Define “SDK bonded” against the canonical validator set and expose that
   exact stake separately from Zerone virtual/effective selection stake.
3. Add a governed, auditable controller-review record. The record needs a
   lifecycle and conflict policy; a DID string or funding correlation cannot
   be silently promoted into proof of common control.
4. Expose exact qualification records and their proof at the same height. Do
   not use the existing insufficient-qualified fallback for computational
   claims.
5. Persist the verified snapshot in each computational `VerificationRound`,
   include it in genesis/export validation, and prohibit replacement after
   round creation.
6. Admit commits and reveals only for snapshot seats, recheck that the
   controller, bond, and qualification remain eligible at each admission
   height, and define the terminal policy when eligibility is revoked between
   phases.
7. Bind each VRF seed to the round and snapshot hash; bind the operator and
   consensus key to the authenticated CometBFT voter; verify the proof; and
   calculate selection against the immutable seat weights and total.
8. Add adversarial end-to-end tests for stale height proofs, incomplete
   manifests, controller collisions, key rotation, zero/unknown stake,
   qualification expiry, equivocation, commit/reveal revocation, and restart
   or export/import of the snapshot.

Until that wiring and an independent review exist, the snapshot package is a
falsifiable data contract only. The activation marker remains absent and the
official vote-extension transport remains disabled.
