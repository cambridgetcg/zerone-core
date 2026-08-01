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

## Build and check

```bash
npm ci
npm test
npm run build
```

`npm run check` type-checks both the browser application and Pages Functions.
`npm test` exercises the REST/RPC allowlists against injected fake upstreams;
it cannot contact the production node. `npm run build` repeats those gates,
builds the Vite application, and compiles Pages Functions with the repository's
pinned Wrangler version.

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
exists only in page memory: it is not posted, persisted, attached to the Pi
account, treated as identity or evidence, or shared with wallet code. Refresh,
clear, and Pi logout discard it.

An authenticated deletion control removes all subject-linked Pi pilot
sessions, the active address binding, outstanding challenges, and rotation
aliases in one D1 transition. A short-lived keyed subject tombstone prevents
an OAuth flow started before deletion from recreating the session. Minimal
unindexed anti-replay fingerprints remain outside account-selectable deletion
and therefore remain part of the separate production retention gate. This
control never changes the person's Pi account, Keplr wallet, or blockchain
state.

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
| `PI_SUBJECT_PEPPER` | Deployment secret containing at least 32 random bytes. |
| `PI_AUTH_DB` | D1 binding with the checked-in migration applied. |

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
then validate phase A before enabling phase B. Preview and production must not
share a subject pepper, D1 database, cookie state, or Pi redirect registration.
See
[`docs/specs/pi-account-onboarding-pilot-v1.md`](../docs/specs/pi-account-onboarding-pilot-v1.md)
for the threat model, privacy boundary, acceptance gates, and rollback order.

## Deploy

The Cloudflare Pages project is `zerone-ai`. From this directory:

```bash
wrangler pages deploy dist --project-name zerone-ai --branch main
```

Run `npm run build` first. A non-`main` branch creates a no-index preview.

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
- Pi callback tokens are erased from the URL before parsing and are posted once
  to a same-origin edge route. Bearers and raw Pi account subjects are never
  persisted.
- Pi routes are separate from the wildcard-CORS RPC/REST proxy. They require
  exact-origin and session-bound CSRF checks for mutations, return `no-store`,
  and use D1 for single-use state, sessions, challenges, and binding uniqueness.
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
