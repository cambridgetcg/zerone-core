# Pi account onboarding pilot v1

Status: source-prepared, activation-gated

## Purpose

This pilot lets a person introduce an app-specific Pi account to the Zerone
dashboard. A separately gated, optional step can prove control of one Zerone
wallet key with a signature in the draft ADR-036 off-chain format.

The two statements are deliberately independent:

1. Pi authentication says that Pi's `/v2/me` endpoint accepted a bearer token
   and returned an app-specific account subject.
2. Zerone wallet proof says that one key signed one short-lived, server-issued
   message for `zerone-1`.

Neither statement proves legal identity, Pi KYC status, unique humanity,
reputation, qualification, reward eligibility, ownership of a Pi wallet, or
control of any other Zerone identity. The pilot does not bridge assets, accept
payments, broadcast transactions, change consensus, or activate rewards.

## Scope and non-goals

The implemented source:

- uses Pi's OAuth implicit sign-in flow with the `username` scope only;
- exchanges the fragment bearer once, server-side, against the fixed
  `https://api.minepi.com/v2/me` endpoint;
- stores a peppered internal Pi subject, a bounded display username, opaque
  sessions, single-use OAuth state, and optional wallet-binding evidence in D1;
- keeps normal dashboard browsing and Keplr wallet use available without Pi;
- verifies an optional ADR-036 signature entirely off-chain; and
- provides explicit logout, unlink, session-family rotation, expiry, subject
  deletion through an authenticated pilot-data route, and bounded
  retention-cleanup behavior.

It does not load the Pi JavaScript SDK, request `wallet_address`, use a Pi API
key, call a Pi payment endpoint, accept a Pi passphrase, query Pi KYC status, or
submit a Zerone transaction. Pi payments and any Pi-to-Zerone asset bridge are
outside this version.

## Activation model

Both phases fail closed:

| Phase | Browser build flag | Pages Function flag | Meaning |
| --- | --- | --- | --- |
| Pi account sign-in | `VITE_PI_PILOT_ENABLED=true` | `PI_PILOT_ENABLED=true` | Exposes sign-in only when both sides agree. |
| Zerone wallet proof | `VITE_PI_WALLET_PROOF_ENABLED=true` | `PI_WALLET_PROOF_ENABLED=true` | Adds a second consent and off-chain proof only after sign-in. |

All four flags are unset or false by default. Source, a successful build, D1
migrations, and a production deployment are not activation: neither phase is
enabled without the exact browser and Pages Function `true` values shown above.
Production also requires all of the following:

- a reviewed Pi developer application whose exact redirect is
  `<PI_PUBLIC_ORIGIN>/pi/callback/`;
- an exact HTTPS `PI_PUBLIC_ORIGIN`;
- a deployment secret `PI_SUBJECT_PEPPER` containing at least 32 random bytes;
- an active positive integer `PI_SUBJECT_PEPPER_VERSION`, or an intentional
  omission to retain the legacy default version `1`;
- any rotation-overlap keys supplied in the strict JSON
  `PI_SUBJECT_PEPPER_PREVIOUS` format described below;
- `PI_BEARER_SHA_CLEAN_START_CONFIRMED=true` only after operators record a
  clean target-D1 review and deployment-history review confirming pre-SHA
  bearer code never served that database; an empty database or migration latch
  alone is not historical proof;
- a D1 database bound to Pages Functions as `PI_AUTH_DB`;
- every checked-in D1 migration applied in order to that exact database;
- reviewed Cloudflare rate-limit rules for `/api/pi/authorize` and
  `/api/pi/session`, tested without weakening same-origin or callback behavior;
- a reviewed operator procedure or separately provisioned job that invokes the
  bounded cleanup primitive; the source provides no scheduler or cron trigger;
- Cloudflare Web Analytics automatic JavaScript injection disabled, with the
  live dashboard response verified to contain no third-party executable code;
- both browser and edge flags set intentionally for the selected phase; and
- a live review of cookies, callback headers, sign-in, replay rejection,
  logout, unlink, and kill-switch behavior.

Preview and production environments require separate origins, Pi redirects,
secrets, D1 databases, and flags. Never reuse a production session database or
subject pepper in a preview.

## Authentication boundary

`GET /api/pi/authorize` is the only OAuth entry. The edge generates a random
browser transaction and random OAuth state, stores only keyed or one-way
representations with a short expiry, sets a Secure, HttpOnly, SameSite cookie,
and redirects to Pi's fixed authorization host.

Pi returns the bearer in the callback fragment. The dedicated callback:

1. copies `location.hash` synchronously;
2. immediately removes the fragment with `history.replaceState`;
3. rejects duplicate, missing, unexpected, or error parameters;
4. posts the bearer and state once to the same-origin session endpoint; and
5. never logs, renders, persists, or forwards the bearer elsewhere.

The edge atomically consumes OAuth state and reserves the domain-separated
commitment `SHA-256("zerone-pi-bearer-replay-v1\0" || access_token)` before
calling only `GET https://api.minepi.com/v2/me` with redirects disabled, a
short timeout, and a bounded response body. The commitment design assumes the
OAuth unguessability target in
[RFC 6749 section 10.10](https://www.rfc-editor.org/rfc/rfc6749#section-10.10),
not a Pi-specific token-entropy guarantee. It never persists the bearer or raw
Pi `uid`.

An authoritative 401/403 deletes only the exact scheme, commitment, state, and
browser reservation. A successful `/v2/me` response atomically promotes the
reservation to a permanent minimal replay marker before any subject/session
insert; a later deletion guard or pepper-floor rejection cannot remove that
marker. An unavailable or malformed upstream response leaves one unverified
pending row, eligible for bounded cleanup only at the inclusive two-minute
boundary. Late promotion after cleanup fails. Successful authentication mints
an opaque session cookie; the D1 row
contains only the session-token digest, the peppered Pi subject, bounded
display data, the pepper version, and creation, last-seen, and expiry metadata.
The CSRF value is a domain-separated HMAC of the opaque session and is not
stored in D1.

Sessions have an eight-hour absolute lifetime and a 30-minute idle timeout.
Each accepted authenticated read advances `last_seen_at` but never extends the
absolute expiry. At the idle boundary the session is no longer accepted.
Successful reauthentication is one atomic subject-family transition: D1
creates the fresh session and revokes every other session for the canonical
subject. A lost response after that transition can require another sign-in, but
cannot leave the old session valid.

All responses are non-cacheable. State-changing session routes require an
exact configured `Origin` and the session-bound `X-Zerone-CSRF` value. No Pi
route uses wildcard CORS.

## Subject-pepper rotation

`PI_SUBJECT_PEPPER` is always the active key. Its version is the canonical
positive decimal integer in `PI_SUBJECT_PEPPER_VERSION`; omitting the version
is supported only as the legacy version `1` behavior. The active version must
increase whenever the active key changes and must never be decreased or
reused.

`PI_SUBJECT_PEPPER_PREVIOUS` is an optional deployment secret and defaults to
an empty list. It must never enter the browser build environment. When present
it must be a strict JSON array whose entries have exactly this shape:

```json
[
  { "version": 1, "pepper": "at-least-32-bytes-of-previous-secret-material" }
]
```

The array may contain at most seven entries. Versions must be unique, strictly
ascending positive integers lower than the active version. All active and
previous pepper strings must be distinct and each must encode 32-1,024 bytes.
Previous keys are overlap verification keys only: they authenticate sessions
created under their version, resolve old and new subject aliases to one
canonical pseudonymous subject. New sessions use the active version. During the
overlap, subject aliases may be materialized for every configured version so
old and new representations remain linked. Bearer replay commitments are
key-independent and do not delay subject-key retirement.

D1 pins each version to a keyed fingerprint and keeps a monotonic active-version
high-water mark plus the exact configured keyset fingerprint. A key mismatch at
an already pinned version, a reused key at a different version, or an
active-version downgrade fails closed. Every active-version or keyset change
also fails while any subject-deletion guard is effective. This freeze prevents
a pre-deletion OAuth callback from escaping a guard by deriving only a newly
configured subject alias. Configuration preflight also fails if any durable
canonical subject with a retained session, challenge, challenge-rate event, or
binding has no alias under any currently configured version.

The first rotation is a two-stage rollout. Apply the lifecycle migration and
deploy rotation-aware code while the active version remains `1`; drain old code
and in-flight requests before raising the active version and retaining version
`1` in `PI_SUBJECT_PEPPER_PREVIOUS`. Repeat that drain-before-increase order for
later rotations. A previous key can be removed only after the retention
procedure has aged out its durable records or each durable subject has gained
an alias under a remaining key. Reauthentication during the overlap gives a version-`1` subject a
version-`2` alias; its redundant version-`1` alias can be ignored and aged out
later. Retiring a key while a durable subject is still represented only by that
version fails configuration preflight.

## Optional Zerone wallet proof

Wallet proof is a second consent boundary. The browser asks Keplr to
`signArbitrary` only after checking that the selected `zerone-1` account still
matches the address used to request the challenge. There is no transaction
fallback.

The edge creates a five-minute, single-use message that binds:

- the exact configured origin and purpose;
- `zerone-1`;
- a canonical `cosmos:zerone-1:zrn1...` account identifier;
- an unlinkable, per-challenge HMAC commitment to the peppered Pi subject,
  current session digest, and random challenge identifier;
- a random nonce and request identifier; and
- issue and expiry times plus an explicit no-funds/no-transaction statement.

The exact stored bytes are reconstructed as a draft ADR-036
`sign/MsgSignData` document. The edge requires the expected secp256k1
public-key type and bounded key/signature lengths, derives the `zrn` address
from that public key, verifies the signature, and atomically consumes the
selected challenge while invalidating and erasing every other outstanding
challenge for that Pi subject. It then creates a strict
one-Pi-subject-to-one-Zerone-address active binding. A conflicting or replayed
link fails. A successful link rotates the opaque session token and CSRF value,
preserves the current eight-hour absolute deadline, and atomically revokes all
other sessions for the subject. Unlink invalidates and erases any newly
outstanding challenges before removing the active association, performs the
same session-family rotation, and does not make any on-chain change. A stale
pre-link or pre-unlink cookie and CSRF pair cannot authorize another request.
The five-per-minute challenge limit is recorded against the canonical subject,
so link, unlink, and session-family rotation cannot reset it. The exact
one-minute boundary is excluded from the live window and is cleanup-eligible.

## Data minimisation and retention

The pilot may retain:

- a peppered, app-specific Pi subject;
- the bounded Pi username for the life of the opaque session;
- an opaque session-token digest with pepper-version, creation, last-seen, and
  expiry timestamps;
- one-time OAuth rows, short-lived unverified SHA bearer reservations, and
  permanent minimal SHA replay markers with claim times;
- one-time wallet challenges until expiry, successful bind, or unlink;
- pseudonymous subject and challenge digests with creation times for the
  one-minute wallet-challenge rate window;
- an active CAIP-10 wallet binding plus consent version and timestamps;
- immutable pepper-version/key-fingerprint pins and a monotonic active-version
  high-water mark, plus the current exact keyset fingerprint;
- a D1-serialized identity-deletion epoch high-water mark; and
- short-lived keyed subject-deletion guards carrying a random per-deletion
  operation commitment, and minimal replay/security tombstones.

It must not retain Pi bearer tokens, raw Pi `uid`, Pi wallet addresses, wallet
passphrases, KYC data, Zerone private keys, or signed transactions.

Expired rows are excluded from authentication and proof decisions. The source
also provides an opt-in `cleanupExpired` repository primitive, but no route,
scheduler, cron trigger, or automatic invocation. An operator or separately
reviewed job must call it with explicit pending, idle, and retention cutoffs
and a cleanup-statement limit from 1 through 1,000. `pendingBefore` cannot be
newer than `now - 2 minutes`; it deletes only unverified SHA reservations.
Promoted replay markers and immutable legacy evidence are never ordinary
retention targets. A call can delete bounded batches of expired OAuth flows,
challenges, challenge-rate events, sessions, subject aliases, expired deletion
guards, and orphan challenge-use rows. A challenge-rate event remains through
both the one-minute rate window and the operator's `retainedBefore` cutoff. It
can also remove a binding created by `retainedBefore` when its subject has no
session seen since that cutoff and no challenge created since the cutoff or
still unexpired; its matching `bound` challenge-use row is removed in the same
bounded D1 batch before the binding. This retention inactivity rule is
intentionally separate from the 30-minute authentication idle timeout.
Repeated calls are required for a backlog.

Logout revokes the current session. Unlink removes the active association while
rotating the session family. `DELETE /api/pi/data` additionally requires the
current session, exact Origin, session-bound CSRF, and a versioned explicit
confirmation. Its subject-deletion transaction atomically removes the
canonical subject's sessions, active binding, outstanding challenges,
directly linked wallet-proof use records, challenge-rate events, and subject
aliases. On success the edge clears both Pi pilot cookies. Before deleting
aliases it writes keyed, pseudonymous guards for the canonical and alias
subject digests. Each OAuth-flow insert snapshots the current deletion epoch
from D1, and deletion increments that epoch in the same serialized D1 batch
that writes the guards. The canonical guard also carries a fresh random
operation commitment; every destructive statement must match it, so a failed
or stale keyset precondition cannot authorize deletion through an older guard.
For exactly 12 minutes, session insertion rejects a flow whose snapshot
precedes a matching guard's epoch; a flow inserted after deletion snapshots the
new epoch and may sign in afresh. This ordering does not compare worker wall
clocks. The guard becomes eligible for physical cleanup at its expiry and may
remain stored until the manual cleanup is invoked. Deletion does not alter the
person's Pi account, Keplr wallet, or any blockchain state.

Deletion addresses only the currently authenticated app-specific Pi subject.
Pi may issue a different app-specific `uid` after a person revokes app
permission; Zerone cannot discover an older subject from the replacement and
must never merge them by username. Finite inactive-binding and alias retention,
plus a documented operator privacy-request path, remain production gates for
historical rows that the current subject cannot select.

Cleanup and subject deletion are data-minimisation controls, not a promise of
total erasure. Permanent bearer replay markers, challenge-use tombstones newer
than the retention cutoff, rows left by bounded backlog processing, immutable
pepper-version pins, and an unexpired deletion guard may remain because they
enforce replay, downgrade, and deletion-race protections. Production must
either explicitly approve and disclose indefinite retention of the minimal,
unlinked replay commitment with rate/storage monitoring, or remain disabled
until an authoritative Pi maximum token-validity bound plus skew supports a
separately reviewed purge design. Generic `retainedBefore` is not such a
decision.

Subject deletion is logical deletion from the current application database,
not guaranteed immediate destruction of every recoverable provider copy.
Cloudflare D1 Time Travel may retain restorable history for up to 30 days on a
Workers Paid plan or seven days on a Workers Free plan. Operations must verify
the plan used by each deployed database rather than assuming either window.
Before activation, operations must maintain and test a restore runbook plus
protected recovery evidence retained for at least the deployed database's
verified Time Travel recovery window. Use an external append-only ledger, or
an authoritative pre-restore export, with a purpose, key, and store separate
from the application D1 where practical. The evidence must cover reviewed
opaque logical-deletion suppressions and every monotonic security record a
rollback could otherwise lose: the identity-deletion epoch high-water mark,
permanent SHA bearer replay markers, challenge-use tombstones, pepper
version/key pins, the active-version floor, the current exact keyset
fingerprint, and session revocations. It must also preserve any challenge-rate
event still inside its one-minute window when service resumes. Pi must be
disabled before and throughout a restore. Operations then reapply the current
schema and explicitly invalidate every session, in-flight OAuth flow, pending
bearer reservation, wallet challenge, and remaining challenge-rate event
without relying on wall-clock expiry or restore duration. They union or replay
the protected security state and logical deletions into the restored database,
never lowering the deletion-epoch high-water mark. Only completed integrity
checks and the ordinary fail-closed preflight permit the restored database to
serve or Pi to be re-enabled. This source implements neither the external
evidence nor an automatic restore hook, so production activation is blocked
until those operational controls exist.

## Threat and failure model

The design treats URL fragments, bearer tokens, OAuth state, cookies, callback
assets, upstream redirects, D1 races, challenge replay, wallet switching, and
misleading identity copy as security boundaries. D1 is mandatory because
single-use transitions and uniqueness constraints require strongly consistent,
transactional storage; Workers KV is not an acceptable substitute.

If configuration, D1, Pi upstream validation, Origin/CSRF validation, signature
verification, or atomic consumption fails, the request fails closed. Disabling
the edge flag is the immediate kill switch. Rollback clears the browser flag
first, clears the edge flag, revokes sessions, and only then considers schema
cleanup under a separately reviewed retention decision.

Pepper-version configuration is part of the fail-closed boundary. D1 rejects a
version/key mismatch and an active-version downgrade. An effective deletion
guard freezes the exact active version and configured keyset fingerprint.
Subject deletion is also raced against in-flight upstream validation: the
D1-snapshotted OAuth deletion epoch, the atomic epoch increment and guard, and
conditional session insertion prevent a pre-deletion flow from resurrecting
the deleted subject. The 12-minute guard window exceeds the ten-minute OAuth
lifetime plus the bounded upstream-request allowance; wall-clock values set
expiry only and never establish before-versus-after deletion ordering.
Every subject-deletion mutation additionally requires the fresh operation
commitment written by that exact batch. An expired but not yet cleaned guard
cannot be reused by a stale request whose keyset precondition failed.

Migration labels every pre-SHA bearer claim as `legacy-keyed-v1`, latches
immutable legacy evidence when any such row exists, and rejects later
old-code inserts that omit the new scheme. Enabled preflight fails while the
latch exists. Routine cleanup never removes legacy rows or clears the latch:
the pilot must remain disabled until authoritative invalidation and an explicit
reviewed remediation, rather than assuming a Pi token lifetime.

That latch observes only claims present when the migration runs. It cannot
prove that an older deployment never created and later deleted legacy claims.
The edge additionally requires exact
`PI_BEARER_SHA_CLEAN_START_CONFIRMED=true`; operators may set it only after a
recorded clean-target-D1 and deployment-history review confirms pre-SHA bearer
code never served the database. A possibly reused database requires explicit,
reviewed remediation and credential invalidation. Otherwise enabled requests
fail closed with HTTP 503.

## Acceptance gates

Before phase A activation:

- callback fragment-erasure and strict-parser tests pass;
- edge rate limits constrain OAuth-row creation and Pi `/v2/me` verification
  attempts before requests reach the Pages Functions;
- OAuth state mismatch, expiry, and concurrent replay tests pass;
- only the fixed `/v2/me` upstream is reachable;
- bearer-token non-persistence and bounded-error tests pass;
- exact SHA reservation/promotion, concurrent-token uniqueness, 401/403
  deletion, ABA-safe rejection, two-minute pending cleanup, late-finalize
  rejection, permanent-marker retention, and legacy-latch tests pass;
- repeated authorization and reauthentication in both Pi sandbox and live mode
  demonstrate fresh access-token behavior; Pi documentation does not guarantee
  uniqueness, so observed token reuse blocks activation pending policy/design
  reconsideration;
- the recorded clean-target-D1 and deployment-history review supports exact
  `PI_BEARER_SHA_CLEAN_START_CONFIRMED=true`; an empty migration latch alone
  does not satisfy this gate;
- eight-hour absolute expiry, 30-minute idle expiry/touch, reauthentication
  family rotation, and stale-session rejection tests pass;
- active/previous pepper overlap, version-key pinning, key-mismatch, and
  downgrade rejection tests pass, including delete-then-keyset-change rejection
  for the full subject-guard window;
- both orderings of the subject-deletion/in-flight-`/v2/me` race, monotonic
  deletion epochs under inverted worker clocks and repeated multi-alias
  deletion, fresh post-deletion authorization, and guard-expiry cleanup tests
  pass;
- a stale-keyset deletion raced after guard expiry and fresh reauthentication
  returns failure without mutating the restored session, challenge, rate
  event, alias, prior guard, or deletion-epoch state;
- explicit pilot-data deletion rejects missing confirmation, cross-origin
  requests, and stale CSRF while atomically removing all subject-linked
  sessions and optional wallet state without affecting another subject;
- bounded cleanup tests keep permanent replay markers, apply the separate
  two-minute `pendingBefore` boundary, keep challenge-rate events through both
  the exact one-minute live window and `retainedBefore`, respect
  `retainedBefore` for other security tombstones, and remove only eligible
  inactive old bindings with their matching `bound` use;
- a tested Time Travel restore runbook and protected external recovery evidence
  cover at least the verified plan recovery window; acceptance demonstrates
  pilot-off restore, current-schema application, explicit invalidation of all
  sessions, in-flight OAuth flows, pending bearer reservations, wallet
  challenges, and remaining challenge-rate events independent of expiry;
  union/replay of logical deletions plus the deletion-epoch high-water mark,
  current keyset fingerprint, and monotonic replay, challenge-use, pepper, and
  revocation state; and successful integrity/preflight checks before the
  restored database can serve or Pi can be re-enabled;
- cookie, Origin, CSRF, cache, CSP, and referrer-policy checks pass; and
- the UI copy and anonymous path receive product/security review.

Before phase B activation:

- deterministic ADR-036 valid and invalid vectors pass;
- wrong key, address, chain, domain, purpose, message, nonce, session, expiry,
  and reused challenge are rejected;
- concurrent bind and one-to-one uniqueness tests pass;
- link/unlink family rotation, stale-cookie rejection, repeated-consent, and
  previously signed conflicting-challenge invalidation behavior pass; and
- tests demonstrate that the edge proof verifier calls no Pi payment, Pi
  wallet, Zerone RPC/REST, broadcast, or consensus path. The pre-existing
  dashboard wallet connection may still make its disclosed read-only chain
  queries.

## Standards placement

This pilot is an application-layer onboarding seam. It is not a Zerone crypto
adapter, identity standard, consensus module, chain registry entry, or
constructive-intelligence capability. It therefore does not appear in the
public adapter capability index. Any later reusable profile requires its own
versioned schema, privacy review, interoperability vectors, and independent
activation decision.

## External references

- [Pi Platform sign-in documentation](https://github.com/pi-apps/pi-platform-docs/blob/master/pi-sign-in.md)
- [Cosmos ADR-036: arbitrary signature](https://github.com/cosmos/cosmos-sdk/blob/main/docs/architecture/adr-036-arbitrary-signature.md)
- [CAIP-10 account IDs](https://chainagnostic.org/CAIPs/caip-10)
- [Cloudflare D1 Workers Binding API](https://developers.cloudflare.com/d1/worker-api/d1-database/)
- [Cloudflare D1 limits and Time Travel windows](https://developers.cloudflare.com/d1/platform/limits/)
- [Cloudflare D1 Time Travel and backups](https://developers.cloudflare.com/d1/reference/time-travel/)
