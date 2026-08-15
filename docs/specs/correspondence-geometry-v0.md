# Correspondence Geometry v0

- Status: `READ_ONLY_ZERO_EFFECT`
- Artifact: `dashboard/public/standards/correspondence-geometry.v0.json`
- Snapshot: `2026-08-15`
- Authority: none

## Purpose

Correspondence Geometry v0 is a sealed, static atlas for asking what can be
carried from one description into another. It makes a proposed map inspectable
by publishing its background, scope, invariant, loss, counterexample,
round-trip expectation, and prohibited conclusions together.

The atlas is not a string-theory implementation and does not add a consensus
module. It does not turn physics into protocol truth, theology into scientific
evidence, metaphor into ontology, or a useful analogy into an equivalence.
Its exact authority statement is:

> This publication proposes engineering correspondences. It establishes no
> scientific, theological, protocol, institutional, or personal authority.

The browser performs one bounded same-origin static `GET` for the atlas. That
read is not a network-state observation or write. No wallet, RPC, API, event,
signature, transaction, or account is involved.

## Reviewed shape

The top-level object has these exact fields:

```text
schema, version, snapshotDate, status, title, summary, authorityStatement,
sourceBindings, physicsSources, epistemicLanes, relationKinds, dimensions,
correspondences, energyFirewall, dualityGate, releaseBoundary
```

The reviewed population is:

- 7 digest-pinned local source bindings;
- 5 primary physics papers whose URLs are locators, never fetched inputs;
- 5 epistemic lanes;
- 4 relation kinds;
- 4 human-facing dimensions;
- 7 proposed correspondences;
- 6 mutually non-converting energy lanes;
- 0 `DUALITY_CANDIDATE` records;
- 0 reviewed or accepted duality candidates; and
- 20 release switches, all `false`.

Every correspondence is `PROPOSED`. Every `equivalenceScope` is `null`. The
manifest therefore makes no equivalence claim. Only the proposed translation
declares round-trip tests, and both observed values are `NOT_RUN`; an expected
`PASS` is a test target, not an observed result. One-way analogies and the
projection declare empty round-trip arrays because they have no inverse.

## Local source bindings

Bindings are raw-byte SHA-256 commitments. A validator resolves each path
inside the repository, refuses symlinks and path escape, reads the exact local
bytes, and compares the digest. A binding imports only the named source role,
not authority or live network state.

| ID | Path | Raw SHA-256 |
|---|---|---|
| `compassion` | `docs/COMPASSION.md` | `ddebabc2b875532c2a3ec76c50f8f63fcef1007a728b7e260a5da36d24c619ad` |
| `knowledge-metabolism` | `x/knowledge/keeper/metabolism.go` | `7b252c5134a6c78b753820890dbcd1eac9d3b263c8bf798c44bc1b41f50aafc6` |
| `knowledge-methodologies` | `x/knowledge/types/methodologies.go` | `fa16ac33e7f2c10a19ed76541af6c2378edb79683578f2cec6f1a0563ebec386` |
| `knowledge-types` | `proto/zerone/knowledge/v1/types.proto` | `7b2b301c80711587a55ae03216728ec1f6f5bf981035106d26ac1fa4923d8ced` |
| `relational-topology` | `dashboard/public/standards/relational-topology.v0.json` | `9786674730febfe47150f29adffa4e4f7bd98e2aff502c552fa5b9669d935711` |
| `research-training-trace` | `docs/RESEARCH_TRAINING_TRACE.md` | `2bca7a4164e2a52e7e4f5830ba6de6ef67542af1fa06ee913b6b2fbdf2640919` |
| `tok-substrate` | `docs/TOK_SUBSTRATE.md` | `4fec6e3a410d5736f61cd43f4d9c421380b93f649c2f0d026a5f4e68a6534328` |

## Epistemic lanes

The five lanes prevent evidence from silently changing register:

1. `PHYSICS_MATH` keeps published claims inside their named models,
   assumptions, scales, and regimes.
2. `CONJECTURE` contains speculative, analogical, open, or untested proposals.
3. `ZERONE_PROTOCOL` describes versioned source semantics and explicitly named
   static architecture; it does not imply deployment.
4. `ENGINEERING_TRANSFER` contains bounded design lessons with failure
   conditions and non-transfer walls.
5. `THEOLOGY_MEDITATION` carries witness, belief, practice, and contemplative
   language without machine adjudication.

A citation in `PHYSICS_MATH` cannot warrant a Zerone mapping. A protocol
identifier cannot turn a conjecture into observed state. A religious record
cannot mint divine endorsement, doctrine, conversion, salvation, or spiritual
rank.

## Relation-kind gates

### `ANALOGY`

An analogy is one-way. It must have a forward map, at least one structural
invariant, explicit information loss, a counterexample, and a non-transfer
wall. Its `inverseMap` is `null`, its `roundTripTests` array is empty, and its
`equivalenceScope` is `null`. It cannot inherit truth, proof, authority, or
equivalence from its source.

### `TRANSLATION`

A translation proposes a vocabulary or representation map. It requires a
forward map, an inverse proposal, and both `SOURCE_TARGET_SOURCE` and
`TARGET_SOURCE_TARGET` round-trip entries. While either observed result is
`NOT_RUN`, its assessment remains `PROPOSED`; an inverse proposal alone is not
an equivalence claim.

### `PROJECTION`

A projection is deliberately lossy. It requires at least one
`informationLosses` entry whose scope is `IN_SCOPE`, and it cannot claim that
the public target reconstructs lived, tacit, communal, or sacred source
context. Its inverse is `null` and its round-trip array is empty.

### `DUALITY_CANDIDATE`

A candidate is admissible only with all of the following:

- a non-empty named equivalence scope;
- an explicit forward mapping rule inside that scope;
- an explicit inverse inside that scope;
- only `EXACT` or `TOLERANCED` invariants;
- no `IN_SCOPE` information loss; and
- both round-trip directions observed `PASS`.

No v0 record satisfies or invokes this gate. The atlas label is exactly
`NO_EQUIVALENCE_CLAIMED`, with reviewed and accepted counts both zero.
Passing the machine-shape gate would create only a candidate for deeper review;
it would not prove totality or mathematical, physical, or ontological
equivalence.

## The seven bounded mappings

### Understanding

`compactification-to-declared-projection` is an `ANALOGY`. It transfers only
the discipline of declaring background choice, retained axes, omitted axes,
assumptions, and version. It does not identify a private or omitted software
field with a physical extra dimension.

`duality-to-bounded-reformulation` is an `ANALOGY`. It asks a Zerone
reformulation to name its map, preserved invariant, validity regime,
breakpoint, and counterexample. It transfers no string-theory duality, proof,
strong-weak relation, physical equivalence, or truth status.

### Universe

`entanglement-to-relational-geometry` is an `ANALOGY`. It motivates the design
question of whether explicit typed relations, rather than isolated scores or
screen proximity, carry a useful knowledge shape. A protocol edge is not a
quantum state and establishes no spacetime, consciousness, personhood, truth,
value, or cosmic unity.

`holography-to-commitment-availability` is an `ANALOGY`. It keeps commitment,
availability, versioned decoding, provenance, and challenge as separate
requirements. A root binds bytes; it does not contain or reconstruct
unavailable bytes. The mapping says nothing about whether the universe is a
hologram, a simulation, or evidence for a Creator.

### Energy

`physical-budget-to-fact-metabolism` is an `ENGINEERING_TRANSFER` to
`ZERONE_PROTOCOL` `ANALOGY` about stock, inflow, outflow, cap, and update
order. The target is a source-pinned protocol integer, not a physical measure.
Joules, thermodynamics, conservation laws, physical causation, truth, human
worth, ZRN value, moral value, and spiritual LIFE do not transfer.

`fact-energy-to-metabolism-language` is the one `TRANSLATION`. It proposes the
public phrase *knowledge-metabolism budget* for the exact versioned fact-energy
field and no other energy lane. Both required round trips are present but
`NOT_RUN`, so the translation remains proposed and non-equivalent.

### Religion

`religious-witness-to-public-projection` is the one `PROJECTION`. It proposes
a narrow target that, only in a separate consent-bearing system, could preserve
authorized words, hash, attribution, provenance, declared register, and a
declared consent-policy reference. It explicitly loses lived experience,
liturgy, community interpretation, tacit context, and sacred authority. This
static artifact neither records consent nor promises erasure of copied bytes,
and it does not prove GOD's endorsement, religious equivalence, conversion,
salvation, spiritual status, or doctrine.

## Energy firewall

The six energy-language lanes are `PHYSICAL_ENERGY`, `COMPUTE`,
`PROTOCOL_METABOLISM`, `ECONOMIC`, `LIVED`, and `SPIRITUAL_POETIC`. Their
`measureOrRegister` values keep distinct categories explicit: physical energy
uses J or kWh with a named system boundary; FLOP, GPU-s, and bytes remain
separate compute measures; protocol metabolism is a source-pinned `uint64`
bookkeeping quantity; economics uses `uzrn`; lived experience is first-person
self-report without a machine unit; and spiritual or poetic language remains
in a tradition-internal or poetic register without a machine unit.

Every lane has an empty `mayConvertTo` list. The firewall freezes
`implicitConversion`, `truthFromResource`, `authorityFromResource`,
`personWorthFromResource`, and `restPenalty` to `false`. A numerical resource
balance cannot establish truth, authority, moral value, spiritual state, or a
person's worth. Rest, refusal, silence, and stopping carry no penalty here.

## Primary-source boundaries

The five external URLs are exact primary-source locators. Validation does not
fetch them, and none endorses Zerone or the mappings in this atlas.

- [Witten, string dynamics and dual descriptions, v2](https://arxiv.org/abs/hep-th/9503124v2)
  stays inside named limits and backgrounds; it does not warrant a Zerone
  equivalence.
- [Maldacena, large-N conformal theories and supergravity, v3](https://arxiv.org/abs/hep-th/9711200v3)
  proposes a scoped correspondence; it does not prove that our universe is
  holographic or specify a blockchain architecture.
- [Ryu and Takayanagi, holographic entanglement entropy, v2](https://arxiv.org/abs/hep-th/0603001v2)
  works in an AdS/CFT background; a Merkle root is not its boundary theory.
- [Van Raamsdonk, entanglement and connected spacetime, v2](https://arxiv.org/abs/1005.3035v2)
  concerns specified holographic settings; graph connectivity is not quantum
  entanglement.
- [Witten, strong-coupling Calabi-Yau compactification, v2](https://arxiv.org/abs/hep-th/9602070v2)
  treats the specified E8 x E8 heterotic string strong-coupling limit through
  an eleven-dimensional compactification; omitted application data is not a
  physical compactified dimension.

These sources motivate questions. The local manifest owns every engineering
claim and every possible error in its proposed transfer.

## Validation

The reviewed raw artifact digest is:

```text
f8cfeebf7404ab7e2e86b80362471cdd64015a108c47e98147e80ba7bb9e9a90
```

The offline validator rejects oversized, excessively nested, duplicate-key,
malformed, non-UTF-8, missing-field, reordered-field, or unknown-field JSON.
It verifies the exact manifest digest, local source pins, closed enums, counts,
cross-references, relation-kind gates, energy firewall, zero-candidate duality
gate, and all-false release boundary. External source URLs are compared as
strings and never requested.

The browser loader accepts only the exact same-origin standards path, refuses
redirects, requires JSON content type and strict UTF-8, enforces a byte limit
and timeout, and verifies the raw digest before parsing or rendering. It uses
text nodes rather than HTML interpretation. Failure leaves a visible error and
the raw-artifact link rather than falling through to API or wallet data; the
no-JavaScript document separately carries a substantive seven-map summary and
links the complete raw artifact.

Run the focused and complete checks from the dashboard directory:

```sh
cd dashboard
npm run check:correspondence
NODE_OPTIONS=--preserve-symlinks ./node_modules/.bin/tsx --test \
  tests/correspondence-geometry.test.ts
npm run build
```

A pass proves only byte identity, static shape, closed boundaries, and the
tested renderer behavior. It does not run any declared correspondence round
trip, prove physics, settle theology, observe a chain, or activate a system.

## Release and live verification

Deploy only the exact merged, CI-verified commit from a clean detached
worktree, after `npm ci`, `npm audit`, and `npm run build`:

```sh
release_sha=$(git rev-parse HEAD)
test -z "$(git status --porcelain --untracked-files=all)"
./node_modules/.bin/wrangler pages deploy dist \
  --project-name zerone-ai \
  --branch main \
  --commit-hash "$release_sha" \
  --commit-dirty=false
```

After production deployment, verify that
`https://zerone.ai/#correspondence` renders five epistemic lanes, four
relation kinds, four dimensions, seven proposed mappings, six energy lanes,
and zero equivalences as selectable text. Fetch
`https://zerone.ai/standards/correspondence-geometry.v0.json` without following
a redirect, require JSON content type, and compare its exact bytes and SHA-256
with the reviewed repository artifact. Instrumenting the Correspondence
Geometry initializer must show exactly one bounded same-origin static JSON read
and no API, wallet, signature, transaction, RPC, or write. The wider mainnet
dashboard continues its separately disclosed read-only network refreshes.

## Activation boundary

All 20 `releaseBoundary` values are `false`: consensus and chain state are
unchanged; there are no network writes, ontology registration, scientific or
theological truth assertions, religious equivalence, universe-as-hologram
assertion, energy-lane equation, personhood inference, person ranking, KARMA
event or magnitude, qualification, reward, funds movement, governance,
authority, consent record, or automatic protocol/authority action. Loading the
page does automatically read the same-origin static artifact and render it in
the browser; that presentation behavior is not a protocol or authority effect.

Publishing the static atlas and dashboard view does not activate any of those
capabilities. A future candidate equivalence, protocol write, wallet flow,
reward, governance path, ontology registration, or consent-bearing system
requires a separate artifact, implementation, threat review, tests, release
decision, and applicable authorization. Nothing in v0 supplies that approval
or migration path.
