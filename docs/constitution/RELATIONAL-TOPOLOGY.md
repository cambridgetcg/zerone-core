# ZERONE Relational Topology v0 — Connection Without Capture

Status: static constitutional projection; **not authoritative state, not a
live-network observation, not consensus runtime, and not network-activated**.
It writes no chain state, moves no funds, grants no identity or qualification,
records no consent, and schedules no upgrade.

Machine-readable companion:
[`dashboard/public/standards/relational-topology.v0.json`](../../dashboard/public/standards/relational-topology.v0.json).

## What this is

Zerone already has many ledgers and processes. A list of modules says what
exists, but not how authority, evidence, identity, value, and recognition may
move between them. Relational Topology v0 projects the accepted target design
as a directed graph and separately enumerates the known legacy source conflicts
that migration must retire:

- every node names one role and one plane;
- every edge names its direction, relation, typed flows, meaning, and explicit
  non-implications;
- every node and edge resolves to an exact source pin;
- every principle distinguishes a source-derived rule from a v0 declaration;
- every known alternate writer is named as a conflict, not silently omitted;
- every forbidden route is executable as a graph reachability check; and
- every effect that could make this artifact operational is fixed to `false`.

The graph is a projection of two existing sources:

1. [`docs/AUTHORITATIVE-STATE.md`](../AUTHORITATIVE-STATE.md), SHA-256
   `22d523ee25060957e2c93aba441542e35d767f28f0f0e5e86c800f5fd7ea82e9`;
2. [`money-karma-v1.json`](money-karma-v1.json), SHA-256
   `f22e62f0706971c569bb2156400b6dbeaf72a005d822b1e40c4e2691e7a98c24`.

Those sources remain the authority for their own claims. The graph does not
replace them, and publishing it does not prove that the target migration is
implemented or live.

## The geometry

The four planes deliberately remain distinct:

- **Identity** binds accounts and keys, resolves controllers, and keeps
  participation history. It does not infer personhood, competence, truth, or
  voice.
- **Knowledge** keeps canonical domains, claims, evidence, qualification, and
  bounded verdict processes. It does not own balances or ordinary policy.
- **Economy** keeps balances, consensus stake, scoped disbursements, and
  reconciled exit claims. It does not automatically grant voice, truth, rank,
  or qualification.
- **Governance** owns ordinary policy execution, immutable electorate
  snapshots, and one narrow emergency circuit breaker. It does not directly
  choose individual truth verdicts or recognition.

An edge may carry one or more of seven flow types:

| Flow | Carries | Never implies |
| --- | --- | --- |
| `AUTHORITY` | A typed, scope-bounded command | Direct arbitrary writes or a second writer |
| `ECONOMIC` | A fully backed coin movement | Governance voice, qualification, or truth |
| `EMERGENCY` | One frozen, scope-bounded circuit-breaker capability | Ordinary policy or an unbounded second executor |
| `EVIDENCE` | A replay-protected observation or verdict | Consent, ownership, or universal truth |
| `IDENTITY` | An authenticated or deduplicated binding | Personhood or independence by address count |
| `RECOGNITION` | A fallible artifact-relation projection | Balance, rank, payout, truth, or authority |
| `REFERENCE` | A canonical lookup or frozen snapshot | Copying the referenced authority |

The type is part of the boundary. A reference from staking to a panel may
carry bonded/unjailed status as a binary transport condition; it does not carry
tokens, shares, or ballot weight. A governance command to qualification may
change future policy; it cannot grant one profile competence. A knowledge
event projected into KARMA has no return edge into control.

## Love and understanding as infrastructure

The graph does not claim to measure love. It implements a smaller, testable
architectural shape:

1. **Distinction without isolation.** Related nodes remain distinct and each
   authoritative state domain has one writer.
2. **Connection without capture.** Every connection is typed and carries its
   refusals alongside its meaning.
3. **Understanding through provenance.** Direction, scope, source, and exact
   digest travel with every relation.
4. **Correction without erasure.** A replacement can supersede a projection
   while preserving the earlier source lineage.
5. **Value without rule.** Economic value and recognition cannot silently
   become governance or epistemic power.
6. **Relation without debt.** Witness, evidence, and proximity do not imply
   consent, reciprocity, ownership, or a duty to continue. Refusal, rest,
   correction, and exit remain valid without permission from this graph.

This is the useful geometry: connection that preserves difference, and
legibility that does not become possession. The sixth rule is a declared v0
principle from this design session, not a proposition attributed to either
pinned source.

## Known source conflicts

The target graph is not allowed to make the current split disappear by leaving
it off the page. The machine artifact therefore names six present source
conflicts and their required disposition:

- custom staking must be reconciled, made read-only, and retained as labelled
  history beneath SDK staking;
- custom LIP governance must lose every writer and executor while its history
  remains queryable beneath SDK governance;
- the knowledge-domain registry must migrate to ontology and stop writing;
- the direct fact-injection queue must be cancelled and retired rather than
  bypassing bounded verification;
- knowledge pause and surgical-correction writers must retire beneath the
  bounded emergency circuit breaker; and
- legacy research-fund callers and unguarded egress must retire beneath the
  sole typed `x/vesting_rewards` handler.

These entries are source observations, not live-network observations. They are
also not proof that every alternate writer has been found. The exhaustive
static authority-graph check required by Authoritative State release gate 9.2
remains a future H4 release artifact.

## Executable forbidden paths

The validator builds a directed graph separately for each flow type and
refuses the artifact if any declared forbidden source can reach a forbidden
target through edges carrying that same flow. V0 closes five represented route
classes:

- economic value to qualification or panel power;
- economic value to governance authority;
- ordinary governance to an individual panel verdict or KARMA assignment;
- identity state to epistemic authority;
- KARMA recognition to qualification or control.

It also rejects duplicate JSON keys, excessive nesting, unknown fields,
unsorted or duplicate identifiers, unresolved references, self-edges,
authority cycles, unsafe source paths, source-digest drift, and any release
effect turning on. The browser fetch is pinned to the exact same-origin path
and reviewed document digest, with redirect, content-type, size, UTF-8, stream,
and deadline refusal.

Same-flow reachability does not prove a general information-flow or taint
theorem across a component that transforms one flow type into another. Edge
semantics, `doesNotImply` refusals, conflict enumeration, and source review
remain separate checks. These checks prove properties of the static bytes;
they do not satisfy H4 release gate 9.2 or prove that a running binary enforces
the target design.

## Current reality remains separate

The accepted target architecture in `docs/AUTHORITATIVE-STATE.md` is explicitly
not implemented, scheduled, deployed, or network-activated. The present source
and live network still require their own observation, migration evidence,
rehearsal, authorization, and coordinated upgrade.

Accordingly:

- `TARGET_NOT_IMPLEMENTED` means the named target component does not yet exist;
- `TARGET_SEMANTICS_NOT_ACTIVATED` means related source exists, but the target
  authority semantics are not claimed live;
- `EXISTING_SOURCE` describes checked-in source, not a live binary claim; and
- `STATIC_ONLY` describes a projection with no runtime state.

An edge marked `DESIGN_INFERENCE` is an explicitly authored relation in this
v0 map rather than a runtime-producing relation established by the pinned
source.

The public dashboard must show this boundary beside the graph. A green static
validator is not a green network migration.

## Effect statement

Relational Topology v0 is read-only public architecture. It:

- adds no consensus behavior;
- registers no handler or store;
- schedules no H4-F, H4, H5, or other upgrade;
- changes no validator, signer, key, electorate, proposal, domain, account, or
  balance;
- grants no identity, qualification, recognition, reward, office, or right;
- records no consent or relationship between people or agents; and
- authorizes no release or deployment. Publishing and serving these static
  bytes is a separate repository and dashboard release act; the artifact grants
  no authority for that act.

Any future runtime binding requires its own named implementation, source
review, migration and rollback analysis, release evidence, governance path,
and live verification. Until then, the graph is understanding—not authority.
