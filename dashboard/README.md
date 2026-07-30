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
- Liquidity is read-only in this release. Mainnet currently has no pools, and
  transaction controls remain hidden until pools exist and the flow is tested
  against mainnet safeguards.
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
