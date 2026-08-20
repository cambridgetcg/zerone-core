# Dormant WITNESS consensus-carrier scaffold

Status: source-only, disabled, and not registered.

This directory is a fail-closed design seam for a possible future Zerone
carrier of `kingdom.witnessed-agent-economy/0.1`. It is not a Cosmos SDK module
today. There is deliberately no `module.go`, `AppModule`, message service,
protobuf, generated code, keeper store, genesis state, CLI, app/module-manager
registration, migration, or chain parameter.

## Current admission boundary

Admission proceeds only far enough to reject:

1. The zero-value configuration is disabled.
2. When explicitly enabled for preflight testing, a raw record is bounded to
   32 KiB before verifier parsing.
3. The caller must supply both a lowercase chain ID and the exact audience
   `zerone:<chain-id>`. The verified record must carry that exact audience.
4. A deterministic exact-wire verifier function, controller-policy function,
   and state-mutator function must all be explicit non-nil constructor
   dependencies. No permissive defaults exist. The supplied verifier adapter
   delegates directly to the frozen core verifier.
5. The kind/action pair and per-kind readiness are projected directly from the
   frozen core; `x/witness` owns no duplicate consensus table.
6. Per-kind activation readiness is checked before controller authorization or
   state access. Every current kind is `NOT_CONSENSUS_ADMISSIBLE`.

Therefore every current record rejects before
the controller-policy or state-mutator function. The source tree contains no
call to either function. These function seams make the boundary testable; they
are not an enabled execution path.

The scaffold has no reward, mint, burn, distribution, staking, bank, KARMA,
NEN, score, reputation, or economic-effect hook. A valid publisher signature
still proves only commitment-key control and does not establish controller,
root/quorum, identity, consent, truth, quality, or payment authority.

## Future compact state—not enabled

A future audited carrier would need a separately specified, bounded store. The
minimum contemplated state is:

- subject heads keyed by `(audience, subject_ref)`, containing the pinned kind,
  controller reference/namespace, sequence, parent, and current commitment;
- explicit controller-policy state keyed by `(audience, controller_ref)`, with
  policy digest, active/pending commitment-key fingerprints, revocation, and an
  independently authorized transfer/recovery rule; and
- permanent capability nullifiers keyed by `(audience, nullifier)`, retaining
  enough commitment metadata to refuse replay forever.

Per-kind lifecycle state, authenticated settlement ordering and cross-batch
receipt proofs, gas bounds, storage layout/versioning, genesis/migrations,
governance authority, query privacy, and deterministic consensus verifier
placement all remain unspecified activation blockers. None of this state is
declared, initialized, read, or written by the current scaffold.

No chain carriage may be enabled merely by changing `Enabled`: the immutable
readiness table still rejects every kind, and the final dormant invariant has
no success or mutation branch. Activation requires a new reviewed protocol and
explicit app/genesis/module wiring.
