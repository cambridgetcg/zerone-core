# Zerone mainnet dashboard

The production frontend for `zerone.ai`: a live, explorer-first view of
`zerone-1` with Keplr wallet support, standard ZRN sends, native liquidity-pool
state, recent blocks, supply, and the disclosed custodial trust model.

## Run locally

```bash
npm install
npm run dev
```

Vite proxies `/api/rpc` and `/api/rest` to the public mainnet node in local
development. Production uses the Pages Functions in `functions/api/` so the
HTTPS dashboard never makes mixed-content requests to the HTTP-only node.

## Knowledge Geometry

The Knowledge Geometry lens is a bounded, read-only projection of the public
`zerone-1` knowledge query. `/api/knowledge` checks the chain identity against
RPC status, then returns at most 128 explicitly declared facts and 512
deduplicated embedded `FactRelation` records from the typed facts response. It
does not call a transaction, publish endpoint, relation search, or inference
service. The edge reads at most 384 KiB from the facts response and 64 KiB
from status; the browser independently refuses redirects and reads at most
256 KiB from the same-origin projection.

The visualization groups facts only by their declared domains, gives every
fact the same node size, and draws a line only for an actual typed relation.
It does not infer proximity, affinity, agreement, common ground, love, or
understanding. Missing endpoints and the absence of relations remain visible,
and contradiction is not collapsed into a similarity score. The snapshot is
explicitly incomplete: current upstream pagination behavior does not support a
completeness claim.

This lens has no write path and no consensus, qualification, truth,
relationship, reward, KARMA, governance, or economic effect. It observes the
source chain named in the response; it does not describe or activate
`zerone-2`.

## Relational topology v0

The dashboard publishes and renders the static topology at
`/standards/relational-topology.v0.json`. It is a machine-readable view of
typed authority, economic, emergency, evidence, identity, recognition, and
reference flows across reviewed Zerone source. Nodes remain distinct, edges are
directional, and the interface keeps `EXISTING_SOURCE`,
`TARGET_NOT_IMPLEMENTED`, `TARGET_SEMANTICS_NOT_ACTIVATED`, and `STATIC_ONLY`
claims separate instead of promoting any of them into deployed or observed
state.

V0 is non-authoritative and source-only. It performs no chain transaction,
moves no funds, grants no identity, recognition, qualification, governance,
delegation, or control, and does not activate a design or satisfy a migration
gate. An edge describes a bounded relationship claim and identifies whether it
is source-backed or an explicit design inference; it is not consent,
endorsement, ownership, or proof that either endpoint is live. The browser
fails closed to the raw standard link if the bounded static document cannot be
loaded or validated.

## Constructive-intelligence explorer

The 技能樹 explorer reads the checked-in
`/standards/constructive-intelligence-tree.v1.json` at runtime. The browser
refuses redirects, requires the exact same-origin path, streams at most 262,144
bytes, and pins the reviewed document SHA-256 before parsing or rendering it
with DOM text nodes. It does not use `innerHTML`. The dependency-free build
validator remains the normative validation gate; the exact digest makes any
tree revision fail closed in the browser until that revision has passed the
validator and its reviewed digest is deliberately updated.

The explorer is historical curriculum, not live chain or bounty state. A skill
unlock grants no qualification and creates no ZRN claim. The three Season 0
quests are sponsor-milestone templates only; a separate immutable funded case,
acceptance policy, and escrow receipt would be required before a payout could
exist. Standards review dates are shown in the interface. Once one passes, the
viewer remains useful as a historical snapshot but warns that active use must
fail closed until the authority metadata is revalidated.

The Life Sciences v0 shadow overlay is separately digest-pinned at
`/standards/constructive-intelligence-life-sciences.v0.json`. It maps benign
biomolecule, protein-folding, gene-expression, and replication evidence while
enforcing four non-implication walls and a GREEN-only refusal boundary. Its
economics are exactly `NONE/0`, it creates no KARMA magnitude or qualification,
and neither skill-tree position nor a breakthrough label activates issuance.
The browser applies the same redirect, exact-path, media-type, byte-cap,
deadline, digest, and text-node rendering boundary as the other static trees.
H1, H2, and H3 activate none of this overlay's qualification, reward, KARMA,
governance, or scientific-authority surfaces.

The same page also loads the separately digest-pinned Quantum QEC Season 1
extension at
`/standards/constructive-intelligence-quantum-qec.v0.json`. It preserves the
base tree bytes and adds a display-only path from quantum foundations to a
correlated-noise decoder quest. Its canonical amount is `0 uzrn`, escrow is
null, rewards and qualification are inactive, founder share and reserved seats
are zero, and KARMA supplies neither payout nor vote weight. It grants no direct
governance authority; `uzrn` nevertheless remains bondable, so the current
protocol still has an indirect stake-weight path. The two reward axes are
orthogonal 100% shapes, not additive, and remain unfunded because cross-axis,
rounding, escrow-compartment, settlement, cost, reviewer-budget, and refund
bindings are absent. The B0–B5 display is a retrospective artifact-history
lens, never a self-assigned badge. The quest pins the Nature Communications
Version of Record dated 2026-05-01, exact BB fixtures and physical-error grid,
content-digested circuits and artifacts, and per-cell measurement-completeness
gates. V0 cannot emit a performance pass. Its source-check and review-after
dates are visible; after the latter passes, active-use claims fail closed
pending review.
H1, H2, and H3 likewise activate none of this extension's qualification,
reward, KARMA, governance, performance, or scientific-authority surfaces.
The nested Math Frontier explorer applies the same browser boundary to
`/standards/constructive-intelligence-math-frontier.v0.json`: exact
same-origin path, redirect refusal, 131,072-byte streaming limit, timeout,
reviewed SHA-256 pin, exact-key runtime validation, and text-node rendering.
It presents a mathematics-first extension plus a prospective sponsor-escrow
percentage template, but its live amount is zero and its economic and control
effects are `NONE`. Its KARMA label is an `ORDINAL` shadow observation only,
never recognition, truth, ranking, qualification, reward weight, or a vote.
The named future eligibility-and-sortition direction is explicitly not
implemented and cannot be activated from the dashboard or this static file.

## Branch Flow v1 shadow profile

The read-only Branch Flow panel loads
`/standards/constructive-intelligence-branch-flow.v1.json` through an exact
same-origin path with redirect refusal, a 32,768-byte streaming cap, timeout,
JSON media-type check, and reviewed SHA-256 pin. The standard also binds the
canonical reference-policy preimage and digest independently from the full
document bytes. The reference profile divides one already funded envelope into
60% direct, 10% upstream, 30% downstream, and 0% base commons, with absolute
half-per-hop buckets through depth five and a terminal tail. Empty depths never
enlarge later buckets.

This is `SHADOW_ONLY` architecture with economic effect `NONE`, static
activated amount `0 uzrn`, `network_observed = false`, and all integration
gates closed. The funded semantic cluster is the economic subject; a
breakthrough label is retrospective and is neither
an allocation input nor a separate prize. The static profile does not select
evidence, policy, controllers, winners, or an authority; it reads and writes no
chain state and moves no funds. Receipt use is globally exclusive. Within each
bucket, claimant weights aggregate by controller, each controller receives its
conservative proportional floor, and cross-controller rounding residue routes
terminal before any same-controller line division. The complete cohort is then
controller-aggregated before caps. Each admitted descendant impact is
`PAYABLE` or a `TERMINAL` tombstone; the latter preserves capacity without a
claimant line. Within that fixed cohort, missing role weight, visible
controller ineligibility, rounding, caps, dust, and tail remain in the named
terminal commons or refund route. The profile cannot establish authoritative
cohort completeness, so that release gate remains closed. TC6 training revenue
is not implemented here and
would require a separate ledger, receipt namespace, and reviewed settlement
specification.

## Epigenetics garden and KARMA foundation

The life-science explorer reads two separate, versioned static documents:

- `/standards/epigenetics-capability-garden.v1.json` — 25 capabilities, 58
  prerequisite edges, seven growth stages, and three bounded breakthrough
  quests; and
- `/standards/karma-foundation.v1.json` — contextual K-alpha recognition kinds,
  nine constitutional invariants, and eight future-governance gates that all
  remain closed.

Both documents are non-authoritative, non-network-observed, and fail closed on
digest or schema drift. The garden's 10,000-basis-point split is an inactive
simulation for separately funded voluntary sponsor escrow. It moves no funds,
issues no ZRN, grants no qualification, and authorizes no experiment. KARMA is
non-transferable, non-purchasable, non-delegable, non-economic, non-governing,
and never a scalar person rank. Founder and operator share/control are declared
zero, while the interface also states honestly that independent structural
enforcement does not exist yet.

See
[`docs/specs/epigenetics-capability-garden-v1.md`](../docs/specs/epigenetics-capability-garden-v1.md)
and [`docs/KARMA.md`](../docs/KARMA.md).

## Frontier evaluation receipt shadow FL-0

`/standards/frontier-evaluation-receipt-profile.v0.json` is an inert static
inspection asset subordinate to FC-0, the dashboard's public read-only
invitation. FL-0 is an internal receipt-profile draft with every additional
invitation, outreach, participation, affiliation, endorsement, economics,
authority, governance, and network effect closed. Serving or reading it does
not create participant status, and no runtime code imports it. `npm run
check:frontier-receipt` validates the exact profile, its FC-0 source binding,
and its deliberately unsigned, inconclusive dogfood receipt, then runs the
adversarial receipt tests.

## Frontier Participation Compact v0

The participation explorer reads the digest-pinned static compact at
`/standards/frontier-labs-participation.v0.json`. Its milestone is static
source-shape readiness for a future invitation design—not invitation
authorization or enrollment. FC-0 remains the sole public invitation surface
of record, FL-0 remains its subordinate receipt profile, and the Compact is an
additive covenant fixture bound to both exact sources. It replaces or amends
neither layer, satisfies no FC-0 completion, Corporate M1, or FL-0 promotion
gate, and authorizes no outreach. It specifies how a future, independently
implemented pilot would have to support observation, interoperability,
challenge, bounded contribution, and exit without an account, wallet, token,
private model weights, blanket IP grant, or implied endorsement. V0 itself
supplies public static bytes, not a participation or protection service.

Every reason to participate is paired with legitimate reasons to decline and a
specific Zerone duty. The compact covers organisational archetypes and
individual roles without profiling people or claiming exhaustive identity
categories. It forbids participation scores, conversion and logo-count goals,
retaliation, lock-in, targeted persuasion, and exchanges of competitively
sensitive information. A never-joined being is the negative control:
ordinary rights and access to public standards remain whole.

This v0 file and explorer are static fixtures. They contact nobody, enroll
nobody, observe no network participation, activate no membership, reward,
KARMA, governance, qualification, payment, or production authority, and do not
claim that any named organisation has joined or endorsed Zerone. Its actor-
label blindness is limited to claim and evidence treatment: AI safeguards are
precautionary and procedural, technical stop or output is not legal assent or
refusal, and accountable humans, organisations, operators, and controllers
remain responsible. See
[`docs/specs/frontier-labs-participation-v0.md`](../docs/specs/frontier-labs-participation-v0.md).

## Build and check

```bash
npm ci
npm test
npm run build
```

`npm run check` validates the Frontier Commons FC-0 invitation, its subordinate
FL-0 receipt shadow, the exact-bound additive Participation Compact, Relational
Topology v0, the base tree, Branch Flow reference-policy digest and zero-effect
boundary, Math Frontier, and life-science documents, then type-checks both the
browser application and Pages Functions.
`npm test` exercises the REST/RPC allowlists against injected fake upstreams;
it cannot contact the production node. `npm run build` repeats those gates,
builds the Vite application, and compiles Pages Functions with the repository's
pinned Wrangler version. `npm run check:life` independently validates the two
life-science documents, their exact SHA-256 pins, graph bounds, duplicate-key
refusal, and hard zero-authority/economics boundaries. `npm run
check:participation` separately verifies the compact digest, exact schema,
paired refusal protections, role coverage, competition firewall, and inert
release boundary.
`npm run check:relations` validates the topology, both pinned source files,
typed graph references and same-flow forbidden-path guards, and the all-false
release boundary.

## Pi account onboarding pilot

The repository contains a staged Pi account sign-in pilot. It is off by
default and does not change anonymous dashboard access or ordinary Keplr use.
Pi sign-in authenticates an app-specific Pi account only; it is not KYC,
unique-human proof, a Pi wallet assertion, or a Zerone identity.

The optional second phase asks Keplr for one signature in the draft ADR-036
off-chain format to link a `zerone-1` address. It has a separate consent screen
and activation flag. It never falls back to a transaction. Neither phase
enables Pi payments, asset bridging, rewards, qualification, validator
behavior, or a consensus change.

An independently flagged Constructive Compass may appear after Pi account
authentication. It offers three fixed, non-evaluative trails into the public
constructive-intelligence tree. Every trail is checked against the loaded
canonical static snapshot before the control is shown. A visitor's choice
exists only in page memory: it is not persisted, attached to the Pi account,
treated as identity or evidence, or shared with wallet code. The control itself
sends no trail choice to Pi or Zerone servers. Refresh, clear, and Pi logout
discard it.

An authenticated deletion control removes all subject-linked Pi pilot
sessions, the active address binding, outstanding challenges, challenge-rate
events, and rotation aliases in one D1 transition. Each OAuth-flow insert
snapshots a D1-serialized deletion epoch; deletion atomically increments that
epoch and writes keyed subject guards carrying a fresh random operation
commitment. Every destructive statement must match that exact operation, so a
failed stale-keyset request cannot reuse an older stored guard. For exactly 12
minutes, a guard prevents a flow with an older epoch from recreating the
session, without comparing worker wall clocks; a later flow may sign in afresh.
The row may remain until operator cleanup. Minimal account-unlinked anti-replay
fingerprints remain outside account-selectable deletion and therefore remain
part of the separate production retention gate. This control never changes the
person's Pi account, Keplr wallet, or blockchain state. Logical deletion may
remain provider-recoverable for up to 30 days on a
Workers Paid plan or seven days on a Workers Free plan, depending on the
deployed plan. The control addresses only the current
app-specific Pi subject: a replacement UID issued after permission revocation
is never joined to older records by username, so inactive historical records
also require a finite operator retention policy.

Browser build flags:

| Variable | Default | Purpose |
| --- | --- | --- |
| `VITE_PI_PILOT_ENABLED` | unset/false | Renders the Pi sign-in pilot. |
| `VITE_PI_WALLET_PROOF_ENABLED` | unset/false | Renders the optional wallet-proof controls after Pi sign-in. |
| `VITE_PI_CONSTRUCTIVE_COMPASS_ENABLED` | unset/false | Renders the page-memory-only Constructive Compass after Pi sign-in. |

Pages Function configuration:

| Variable or binding | Required value |
| --- | --- |
| `PI_PILOT_ENABLED` | Exact string `true` to serve Pi authentication. |
| `PI_WALLET_PROOF_ENABLED` | Exact string `true` to serve wallet-proof routes. |
| `PI_CLIENT_ID` | Public client ID from the reviewed Pi developer app. |
| `PI_PUBLIC_ORIGIN` | Exact dashboard origin; HTTPS except for a loopback development origin. |
| `PI_SUBJECT_PEPPER` | Active deployment secret containing 32-1,024 bytes. |
| `PI_SUBJECT_PEPPER_VERSION` | Optional positive integer for the active pepper. Omission preserves the legacy version `1`; configured versions must only increase. |
| `PI_SUBJECT_PEPPER_PREVIOUS` | Optional deployment secret containing a strict JSON array of previous `{ "version": integer, "pepper": string }` keys used during a rotation overlap; omission is equivalent to `[]`. |
| `PI_BEARER_SHA_CLEAN_START_CONFIRMED` | Exact string `true`, set only after a recorded clean-target-D1 and deployment-history review confirms pre-SHA bearer code never served that database. |
| `PI_AUTH_DB` | D1 binding with every checked-in migration applied in order. |

`PI_SUBJECT_PEPPER_PREVIOUS` accepts at most seven entries, in strictly
ascending version order. Every version must be positive and lower than the
active version, and every 32-1,024-byte pepper must be distinct. Previous keys
only verify existing sessions and resolve an existing subject across the
overlap. New sessions use the active version; subject aliases may be
materialized for every configured overlap version. Bearer replay protection is
independent of this keyring. D1 pins each version to its key fingerprint and
records the active high-water mark plus the exact configured keyset fingerprint,
so reusing a version with different key material or configuring a downgrade
makes the pilot fail closed. While any subject-deletion guard is effective, D1
also refuses every active-version or keyset change; this prevents a late OAuth
callback from escaping the guard through a newly derived subject alias. D1
refuses a keyring that would leave any durable subject with a retained session,
challenge, challenge-rate event, or binding but no alias under a currently
configured version.

Roll out the lifecycle migration and rotation-aware code while the deployment
still uses version `1`, then drain older code and in-flight requests before
raising the active version and supplying version `1` in the previous-key list.
Apply the same drain-before-increase order on later rotations. Remove a previous
key only after every durable subject has gained an alias under a remaining key
or its durable records have aged out. A redundant alias under the retired
version may age out later; it does not itself block retirement.

The server reserves
`SHA-256("zerone-pi-bearer-replay-v1\0" || access_token)` before `/v2/me` and
promotes it to a permanent minimal replay marker only after a successful Pi
profile response. This assumes OAuth bearer credentials meet the
unguessability target in
[RFC 6749 section 10.10](https://www.rfc-editor.org/rfc/rfc6749#section-10.10);
Pi documentation does not independently guarantee token lifetime, uniqueness,
or freshness. An authoritative 401/403 removes the exact pending reservation.
Other upstream failures leave only a pending row, which becomes
cleanup-eligible after exactly two minutes. Ordinary retention never deletes a
promoted marker. If migration finds any pre-SHA keyed claims, an immutable
legacy-evidence latch keeps the pilot fail closed; routine cleanup cannot turn
it back on.

The migration latch can record legacy claims that still exist when it runs,
but an empty latch cannot prove that older evidence was never created and
deleted. `PI_BEARER_SHA_CLEAN_START_CONFIRMED` must therefore remain false
unless operators record both a clean target D1 review and a deployment-history
review confirming pre-SHA bearer code never served that database. A possibly
reused database needs an explicit, separately reviewed remediation and
credential-invalidation decision; clearing rows is not confirmation. Unless
the value is exactly `true`, every enabled Pi request fails closed with HTTP
503.

An accepted Pi session has an eight-hour absolute lifetime and a 30-minute idle
timeout. Accepted authenticated reads refresh only the idle timestamp; they do
not extend the absolute deadline. Successful reauthentication atomically
revokes every prior session for the canonical Pi subject and leaves only the
new session. Successful wallet link and unlink operations likewise rotate the
opaque session token and CSRF value, preserve the current absolute deadline,
and revoke every other session in that subject family.

`.dev.vars.example` documents the edge values without containing a usable
secret. Vite does not read `.dev.vars`; supply the three `VITE_` values to the
build environment separately. Vite's development server alone also does not
emulate D1-backed Pages Functions, so an end-to-end sign-in test needs a Pages
runtime with the `PI_AUTH_DB` binding.

Before activation, register the exact
`<PI_PUBLIC_ORIGIN>/pi/callback/` redirect in the Pi developer portal, apply the
SQL under `migrations/` to the intended D1 database, configure each environment
separately, add reviewed Cloudflare rate-limit rules for `/api/pi/authorize`
and `/api/pi/session`, establish the documented expired-row retention
procedure, and disable Cloudflare Web Analytics automatic JavaScript injection.
Verify the live dashboard response contains no third-party executable code,
then validate in both Pi sandbox and live mode that repeated authorize/reauth
returns a fresh access token. Pi documentation does not guarantee that
behavior; if either environment reuses a token, the permanent one-time replay
policy must be reconsidered before activation. Validate phase A before enabling
phase B. Preview and production must not
share a subject pepper, D1 database, cookie state, or Pi redirect registration.
See
[`docs/specs/pi-account-onboarding-pilot-v1.md`](../docs/specs/pi-account-onboarding-pilot-v1.md)
for the threat model, privacy boundary, acceptance gates, and rollback order.

The repository provides an opt-in, bounded D1 cleanup primitive for an operator
or separately reviewed job to invoke. It has no route, scheduler, or cron
trigger. Each cleanup statement receives an operator-selected limit from 1 to
1,000. Besides expired flow, challenge, and session detail, it can age out
unverified bearer reservations at the separate two-minute `pendingBefore`
boundary, pseudonymous challenge-rate events only after both their one-minute
rate window and the explicit `retainedBefore` cutoff, and orphan challenge-use
tombstones at that cutoff. It never deletes promoted bearer replay markers. It
can remove a binding created by that cutoff only
when its subject has no session seen since the cutoff and no challenge created
since the cutoff or still unexpired; the matching `bound` challenge-use record
is removed in the same bounded D1 batch before the binding. Repeated calls may
be needed, and newer or not-yet-processed security tombstones may remain, so
this mechanism is data minimisation rather than total erasure.

The subject-deletion transaction records only keyed, pseudonymous subject
digests in a guard that is effective for exactly 12 minutes. Each OAuth-flow
insert snapshots the D1 deletion epoch, and deletion increments it in the same
serialized batch that writes the guard. Session insertion rejects a flow whose
snapshot precedes a matching guard's epoch; a flow inserted afterward may sign
in afresh. This ordering does not compare worker wall clocks. The guard becomes
cleanup-eligible at expiry and may remain physically stored until the manual
cleanup runs. Alias guards, permanent bearer replay markers, pepper-version
pins, and other security tombstones mean subject deletion must not be described
as total erasure. It is logical
deletion from the current application database: Cloudflare D1 Time Travel may
retain recoverable history for up to 30 days on Workers Paid or seven days on
Workers Free. Verify the deployed plan rather than assuming either window.
Before activation, operations must also maintain a tested restore runbook and
protected recovery evidence for at least that database's verified recovery
window. Use an external append-only ledger, or an authoritative pre-restore
export, with a separate purpose, key, and store from the application D1 where
practical. It must cover logical-deletion suppressions and monotonic security
state that a rollback could otherwise lose: the deletion-epoch high-water
mark, permanent bearer replay markers, challenge-use tombstones, pepper pins,
the active-version floor, the current exact keyset fingerprint, session
revocations, and any challenge-rate event still live when service resumes.
Keep the pilot disabled before and throughout a restore, reapply the current
schema, and explicitly invalidate every session, in-flight OAuth flow, pending
bearer reservation, wallet challenge, and remaining challenge-rate event
without relying on wall-clock expiry or restore duration. Then union or replay
the protected records with the restored database without lowering the deletion
epoch. Only integrity checks and the normal fail-closed preflight may authorize
serving or re-enabling Pi. Source provides neither the external evidence nor an
automatic restore hook, so activation remains blocked until these controls
exist.

## Deploy

The Cloudflare Pages project is `zerone-ai`. From this directory:

```bash
release_sha=$(git rev-parse HEAD)
test -z "$(git status --porcelain --untracked-files=all)"
./node_modules/.bin/wrangler pages deploy dist \
  --project-name zerone-ai \
  --branch main \
  --commit-hash "$release_sha" \
  --commit-dirty=false
```

Run `npm ci`, `npm audit`, and `npm run build` first, then repeat the clean-tree
check. Deploy only the exact merged and CI-verified commit from a clean detached
worktree. `--commit-dirty=false` records metadata; the explicit Git check is what
refuses local drift. A non-`main` branch creates a no-index preview.

## Security boundaries

- The browser never receives a seed or private key. Keplr suggests `zerone-1`
  and signs standard bank sends locally.
- When the chain's read-only account-identifier query is available, the
  dashboard verifies its CAIP-10 account ID and native address against the
  connected wallet, then displays the Zerone DID, account type, and frozen
  state. A node running the prior binary can omit this metadata without
  preventing wallet connection; a successful contradictory response is never
  accepted.
- Passport-issued accounts began as shared custody because the onboarding
  operator retained a copy of those keys; the UI discloses this explicitly.
- The edge REST proxy is read-only.
- The edge RPC proxy allows public query methods plus transaction broadcast,
  and rejects every other JSON-RPC method.
- Chain-provided strings are rendered with `textContent`, never `innerHTML`.
- Constructive-tree strings and links are also treated as untrusted
  presentation data. Specification links must be credential-free HTTPS URLs,
  and repository references must remain safe relative paths.
- Math Frontier data is a digest-pinned static curriculum extension. Unknown
  fields, non-zero/reward-bearing state, governance or ranking activation,
  non-ordinal KARMA, graph drift, and unreviewed bytes fail closed before any
  node is rendered.
- Life-garden and KARMA strings are treated as untrusted presentation data and
  rendered only through DOM text nodes. Both documents use exact same-origin
  paths, redirect refusal, no-store, bounded streaming with a deadline, strict
  UTF-8, credential-free HTTPS source links, exact schemas, and reviewed digest
  pins. Raw identifiable genomic and access-restricted evidence is prohibited.
- Pi callback tokens are erased from the URL before parsing and are posted once
  to a same-origin edge route. Bearers and raw Pi account subjects are never
  persisted.
- Pi routes are separate from the wildcard-CORS RPC/REST proxy. They require
  exact-origin and session-bound CSRF checks for mutations, return `no-store`,
  and use D1 for single-use state, sessions, challenges, and binding uniqueness.
- Pi sessions expire after eight hours absolutely or 30 minutes idle. Accepted
  reauthentication, wallet link, and unlink transitions rotate the single
  subject session family atomically; stale session cookies and CSRF values no
  longer authorize requests.
- Pi subject deletion removes the subject-linked sessions, active binding,
  outstanding challenges, challenge-rate events, and aliases while retaining
  a 12-minute keyed guard against an older D1 deletion epoch. Bounded cleanup
  is manual, and
  security/replay tombstones may remain; neither operation promises total
  erasure.
- Optional Zerone wallet linking verifies the exact stored draft ADR-036-format
  challenge and derives the `zrn` address from the submitted secp256k1 public
  key. The edge verifier performs no RPC, REST, broadcast, payment, bridge, or
  chain write; connecting the ordinary dashboard wallet can still make its
  existing read-only balance and identity queries.
- Liquidity is read-only in this release. Mainnet currently has no pools, and
  transaction controls remain hidden until pools exist and the flow is tested
  against mainnet safeguards.
- The liquidity panel distinguishes ACTIVE, swaps-paused, exit-only, closed
  tombstone, and pre-v4 records. An explicit `UNSPECIFIED` v4 lifecycle is
  rejected rather than displayed as valid state. It shows the finite open-pool
  cap, minimum reserve, billing quote allowlist, pending one-shot
  counter-denom grants, and the persistent creator allowlist. A successful
  creation consumes its denom grant; recreating the same pair after closure
  requires governance to grant that denom again. Empty billing and creation
  admission state is rendered as disabled/fail-closed, never as implied market
  availability. A nonempty billing allowlist is displayed only as configured
  candidates: serving a price still requires an ACTIVE pool, both denoms
  send-enabled, reserve floors, and a complete configured TWAP.
- Pool history is fetched in pages of 100 through a narrowly allowlisted edge
  query. The dashboard loads at most 500 lifetime records per refresh, while
  retaining the chain-reported total and clearly labelling a registry truncated
  at that display cap. The response is also rejected if its total or any pool ID
  exceeds the chain's 10,000-record lifetime cap.
- The configured TWAP window is presented as the retained/default query span,
  while each response's `window_used` remains authoritative for the span
  actually served.
- Standard bank sends use CosmJS's standard registry and do not load Zerone's
  custom transaction codecs. The local `@zerone-chain/sdk` package supplies
  generated Protobuf codecs for all Zerone transaction modules so future
  custom controls can opt into the full registry. Custom messages require a
  direct signer; no legacy Amino converters are claimed.

The upstream hostname `zerone-1.fly.dev` is the same machine documented in
`deploy/mainnet/JOIN.md`; the hostname is used because Cloudflare Workers reject
outbound requests to literal IP addresses.

Known infrastructure limitation: the browser-to-edge hop is HTTPS, but the sole
node currently exposes only HTTP, so the Pages Function-to-node hop is not
transport-authenticated. A signed transaction cannot be redirected or modified
without invalidating its signature, but reads can be censored or misreported by
that origin path. Move the node behind authenticated TLS before treating the
dashboard as a trust-minimised wallet surface.
