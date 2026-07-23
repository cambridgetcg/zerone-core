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

## Build and check

```bash
npm run check
npm run build
```

## Deploy

The Cloudflare Pages project is `zerone-ai`. From this directory:

```bash
wrangler pages deploy dist --project-name zerone-ai --branch main
```

Run `npm run build` first. A non-`main` branch creates a no-index preview.

## Security boundaries

- The browser never receives a seed or private key. Keplr suggests `zerone-1`
  and signs standard bank sends locally.
- Passport-issued accounts began as shared custody because the onboarding
  operator retained a copy of those keys; the UI discloses this explicitly.
- The edge REST proxy is read-only.
- The edge RPC proxy allows public query methods plus transaction broadcast,
  and rejects every other JSON-RPC method.
- Chain-provided strings are rendered with `textContent`, never `innerHTML`.
- Liquidity is read-only in this release. Mainnet currently has no pools, and
  custom swap/add/remove messages still need generated Protobuf types registered
  with CosmJS before transaction controls should be exposed.

The upstream hostname `zerone-1.fly.dev` is the same machine documented in
`deploy/mainnet/JOIN.md`; the hostname is used because Cloudflare Workers reject
outbound requests to literal IP addresses.

Known infrastructure limitation: the browser-to-edge hop is HTTPS, but the sole
node currently exposes only HTTP, so the Pages Function-to-node hop is not
transport-authenticated. A signed transaction cannot be redirected or modified
without invalidating its signature, but reads can be censored or misreported by
that origin path. Move the node behind authenticated TLS before treating the
dashboard as a trust-minimised wallet surface.
