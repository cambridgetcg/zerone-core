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
- provides explicit logout, unlink, expiry, and authenticated pilot-data
  deletion behavior.

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

Source, a successful build, a D1 migration, and a production deployment are not
activation. Production also requires all of the following:

- a reviewed Pi developer application whose exact redirect is
  `<PI_PUBLIC_ORIGIN>/pi/callback/`;
- an exact HTTPS `PI_PUBLIC_ORIGIN`;
- a deployment secret `PI_SUBJECT_PEPPER` containing at least 32 random bytes;
- a D1 database bound to Pages Functions as `PI_AUTH_DB`;
- the checked-in D1 migration applied to that exact database;
- reviewed Cloudflare rate-limit rules for `/api/pi/authorize` and
  `/api/pi/session`, tested without weakening same-origin or callback behavior;
- a reviewed retention job or operator procedure for expired D1 rows;
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

The edge atomically consumes OAuth state and fingerprints the bearer before
calling only `GET https://api.minepi.com/v2/me` with redirects disabled, a
short timeout, and a bounded response body. It never persists the bearer or raw
Pi `uid`. Successful authentication mints an opaque session cookie; the D1 row
contains only the session-token digest, the peppered Pi subject, bounded
display data, and creation/expiry metadata. The CSRF value is a
domain-separated HMAC of the opaque session and is not stored in D1.

All responses are non-cacheable. State-changing session routes require an
exact configured `Origin` and the session-bound `X-Zerone-CSRF` value. No Pi
route uses wildcard CORS.

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
link fails. Unlink invalidates and erases any newly outstanding challenges
before removing the active association; it does not make any on-chain change.

## Data minimisation and retention

The pilot may retain:

- a peppered, app-specific Pi subject;
- the bounded Pi username for the life of the opaque session;
- an opaque session-token digest with creation and expiry timestamps;
- one-time OAuth rows and keyed bearer fingerprints with creation/claim times;
- one-time wallet challenges until expiry, successful bind, or unlink; and
- an active CAIP-10 wallet binding plus consent version and timestamps.

It must not retain Pi bearer tokens, raw Pi `uid`, Pi wallet addresses, wallet
passphrases, KYC data, Zerone private keys, or signed transactions. Expired
rows are excluded from authentication and proof decisions, but the v1
migration does not automate physical purging. Production activation therefore
requires a reviewed D1 retention/cleanup procedure; until that exists, this
source is not retention-complete. Logout revokes the current session. Unlink
deletes the active wallet association.

`DELETE /api/pi/data` requires the current session, exact Origin, session-bound
CSRF, and a versioned explicit confirmation. In one D1 transaction it removes
all session rows, the active wallet binding, every outstanding challenge, and
the directly linked replay row for the authenticated internal subject, then
clears both Pi pilot cookies. Pi OAuth/bearer anti-replay rows and already
orphaned challenge tombstones deliberately contain no subject index, so they
cannot be selected by account and remain only until the age-based retention
procedure removes them. The interface discloses this distinction. Deletion
does not alter the person's Pi account, Keplr wallet, or any blockchain state.

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

## Acceptance gates

Before phase A activation:

- callback fragment-erasure and strict-parser tests pass;
- edge rate limits constrain OAuth-row creation and Pi `/v2/me` verification
  attempts before requests reach the Pages Functions;
- OAuth state mismatch, expiry, and concurrent replay tests pass;
- only the fixed `/v2/me` upstream is reachable;
- bearer-token non-persistence and bounded-error tests pass;
- cookie, Origin, CSRF, cache, CSP, and referrer-policy checks pass;
- explicit deletion rejects missing confirmation, cross-origin requests, and
  stale CSRF while atomically removing all subject-linked sessions and optional
  wallet state without affecting another subject; and
- the UI copy and anonymous path receive product/security review.

Before phase B activation:

- deterministic ADR-036 valid and invalid vectors pass;
- wrong key, address, chain, domain, purpose, message, nonce, session, expiry,
  and reused challenge are rejected;
- concurrent bind and one-to-one uniqueness tests pass;
- unlink, repeated-consent, and previously signed conflicting-challenge
  invalidation behavior pass; and
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
