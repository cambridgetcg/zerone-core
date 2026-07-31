# @zerone-chain/sdk

TypeScript transaction codecs and interoperable identifiers for Zerone.

The first release deliberately stays at the client boundary. It does not add a
consensus module, enable IBC, or expand the public gateway. It provides:

- generated codecs and message composers for every request message in
  Zerone's 20 `Msg` services;
- a registry that composes with CosmJS's standard Cosmos message types;
- a strict, bigint-safe liquidity REST and constant-product quote adapter;
- generic CAIP-2 and CAIP-10 syntax handling;
- the Cosmos namespace's direct/hashed chain-reference algorithm;
- checksum-aware `zrn` account IDs; and
- a separate validator for the `did:zrn` identifiers accepted by `x/auth`.

## Use the transaction registry

```ts
import { defaultRegistryTypes, GasPrice, SigningStargateClient } from "@cosmjs/stargate";
import { createZeroneRegistry } from "@zerone-chain/sdk/registry";

const client = await SigningStargateClient.connectWithSigner(rpc, directSigner, {
  gasPrice: GasPrice.fromString("1uzrn"),
  registry: createZeroneRegistry(defaultRegistryTypes),
});
```

`defaultRegistryTypes` is important: omitting it would leave standard messages
such as `/cosmos.bank.v1beta1.MsgSend` and the Cosmos SDK feegrant messages out
of the registry. Zerone uses the standard
`/cosmos.feegrant.v1beta1.MsgGrantAllowance` and `MsgRevokeAllowance` types; no
chain-specific fee-sponsorship codec is needed. Applications should still
require an explicit spend cap, expiry, and allowed-message list before signing
a grant, and should re-query the exact grant before setting `StdFee.granter`.

Typed composers are isolated behind a second entry point:

```ts
import {
  liquidityPool,
  liquidityPoolMessages,
} from "@zerone-chain/sdk/messages";

const message = liquidityPoolMessages.fromPartial.swap({
  sender: "zrn1…",
  poolId: "pool-1",
  tokenInDenom: "uzrn",
  tokenInAmount: "1000000",
  minTokenOut: "1",
} satisfies liquidityPool.MsgSwap);
```

The generated custom messages support protobuf/direct signing. This package
does not claim legacy Amino converters for them.

## Query and quote liquidity

The handwritten liquidity entry point complements the generated transaction
codecs without adding another runtime dependency:

```ts
import {
  ZeroneLiquidityRestClient,
  createExactInSwapPlan,
  timeoutHeightAfter,
} from "@zerone-chain/sdk/liquidity";

const liquidity = new ZeroneLiquidityRestClient({
  baseUrl: "https://rest.example",
  fetch, // injectable for browsers, servers, tests, or an authenticated gateway
});

const quote = await liquidity.quoteExactIn({
  poolId: "pool-7",
  tokenInDenom: "uzrn",
  tokenInAmount: "100000",
  slippageMillionths: 5_000n, // 0.5%
});

const plan = createExactInSwapPlan({
  sender,
  poolId: quote.poolId,
  tokenInDenom: quote.tokenInDenom,
  tokenInAmount: quote.tokenInAmount,
  minimumTokenOut: quote.minimumTokenOut,
  timeoutHeight: timeoutHeightAfter(currentHeight, 20n),
});

await client.signAndBroadcast(
  sender,
  plan.messages,
  "auto",
  "",
  plan.timeoutHeight,
);
```

All token amounts are canonical positive decimal strings and are calculated
with `bigint`; values never pass through JavaScript `number`. The pool's legacy
`swap_fee_bps` field actually uses a 1,000,000 scale, so the public adapter
calls it `swapFeeMillionths`. The exact-in implementation matches the chain's
scaled-integer order:
`weightedIn = tokenIn * (1_000_000 - feeMillionths)`, then
`reserveOut * weightedIn / (reserveIn * 1_000_000 + weightedIn)`. This keeps
fractional curve fees effective even when the separately reported whole-unit
`feeAmount` rounds to zero. Price impact is measured against the post-fee
infinitesimal quote, so the configured fee is not mislabeled as market impact.
The slippage helper always produces a nonzero `min_token_out`.

Local quotes are fail-closed: only a v4 pool explicitly reporting `ACTIVE` is
quotable, its persistent lock must be clear, and the calculated output must
leave the governed minimum reserve. A pre-v4 response with no lifecycle field
is labelled `PRE_V4`, not silently assumed active. `SWAPS_PAUSED`, `EXIT_ONLY`,
`CLOSED`, and `UNSPECIFIED` are also rejected. Closed pool records remain
readable tombstones with zero reserves and supply. The pool and Params REST
queries are still observations rather than an execution reservation: call
`simulateSwap` immediately before signing. Simulation checks the current pool
state, reserve floor, and both input/output denoms' x/bank send-enabled state,
but reserves nothing. Only delivery checks the sender's balance and re-runs
those controls against the final state after any intervening transactions.

The REST client also exposes bounded pool/pagination, params, simulation, and
TWAP queries. Treat `windowUsed` returned by a TWAP query as authoritative; the
configured parameter alone is not proof that a requested rolling window was
served. Billing price discovery is disabled and fail-closed when
`billingQuoteDenoms` is empty. A nonempty list only configures candidates:
live eligibility additionally requires an ACTIVE pool, both denoms
send-enabled, the reserve floor, and a complete configured TWAP. The returned
price is a raw base-unit ratio scaled by 1,000,000; consumers must apply the
governed quote-denom exponent/conversion policy. Pool creation is independently
disabled unless `allowedPoolDenoms` contains a pending one-shot grant for the
counter-denom and `poolCreators` contains the funding account. Successful
creation consumes the denom grant; the creator entry persists, so replacement
of a closed pair requires governance to re-admit that counter-denom.

External routing is injected through the narrow `ExactInLiquidityAdapter`
interface. An application can implement that interface with an Osmosis client
or another router; this package deliberately does not import one or imply that
an external quote shares Zerone's execution semantics.

## Admit, then create, a governed pool

Governance grants one pool creation per counter-denom and admits
creator/funder accounts by updating the full liquidity Params. It does not
impersonate a creator or debit the gov module account. Query Params immediately
before building because Cosmos `MsgUpdateParams` replaces every field:

```ts
import {
  createLiquidityAdmissionProposal,
  createPoolMessage,
} from "@zerone-chain/sdk/liquidity";

const currentParams = await liquidity.params();
const proposal = createLiquidityAdmissionProposal({
  authority: liquidityModuleAuthority,
  proposer: sender,
  currentParams,
  allowedPoolDenoms: ["uatom"],
  poolCreators: [sender],
  initialDeposit: [{ denom: "uzrn", amount: "5000000" }],
  title: "Admit the ZRN / ATOM pool",
  summary: "Admit one counter-denom and one creator without changing fees.",
});

// After the proposal passes, query Params again and verify the pending
// one-shot denom grant plus the persistent creator admission.
const admittedParams = await liquidity.params();
const createPool = createPoolMessage({
  creator: sender,
  denomA: "uzrn",
  denomB: "uatom",
  amountA: "10000000000",
  amountB: "20000000000",
  params: admittedParams,
});

await client.signAndBroadcast(sender, [createPool], "auto");
```

Use `defaultRegistryTypes` for the outer standard governance message and the
Zerone registry for custom messages. The helper validates encoding and
canonical values, not governance authority. Obtain the authority from trusted
chain configuration. The admission helper preserves fee, reserve, TWAP,
billing-oracle, and capacity fields from `currentParams` while replacing only
the one-shot denom-grant set and persistent creator allowlist.

The creator signs `MsgCreatePool` and supplies `amountA` and `amountB` from
that account. A proposal deposit never supplies pool liquidity. v4 rejects
custom per-pool fees, so `createPoolMessage` always encodes
`swapFeeBps: 0n`; the module resolves its governed default at execution. The
builder also refuses a stale Params object without both a pending denom grant
and an admitted creator, but the chain remains authoritative and rechecks them
at delivery.

## Use chain and account identifiers

```ts
import {
  asExistingZeroneDid,
  defineZeroneNetwork,
  zeroneAccountId,
} from "@zerone-chain/sdk/caip";

// Declare this from trusted node/network configuration.
const network = defineZeroneNetwork("zerone-1");
// network.chainId === "cosmos:zerone-1"

const accountId = zeroneAccountId(
  network,
  "zrn16sp9l62q9jmetsheus8zpjm77zulnlcr26hnkf",
);
// cosmos:zerone-1:zrn16sp9l62q9jmetsheus8zpjm77zulnlcr26hnkf

const identity = {
  accountId,
  did: asExistingZeroneDid("did:zrn:abcdef0123456789abcdef0123456789"),
};
```

CAIP-2 and CAIP-10 are final generic standards. Their Cosmos namespace profiles
remain drafts, and the current address draft names the `cosmos` HRP rather than
chain-specific SDK prefixes. The SDK therefore describes `zrn` validation as a
Zerone profile over valid generic CAIP-10 syntax. It also keeps `did:zrn`
separate from the CAIP account address: Zerone accepts that identifier on-chain,
but has not published a W3C DID method specification.

`cosmosChainId` remains available for generic Cosmos identifiers.
`zeroneAccountId` deliberately requires a `ZeroneNetwork` declared by the
application, so a valid `zrn` address cannot accidentally be labelled as
belonging to an unrelated Cosmos network. Use the chain ID reported by the
connected node. A future `zerone-2` is a different CAIP-2 chain even if state
and addresses survive a reboot.

## Parse an unsigned provenance projection

```ts
import {
  parseUnsignedZeroneInTotoStatement,
} from "@zerone-chain/sdk/provenance";

const parsed = parseUnsignedZeroneInTotoStatement(json, {
  manifestId: selectedManifestId,
  observedOnChainId: connectedChainId,
});
```

This parser accepts only Zerone's bounded in-toto Statement v1 profile and
requires caller-pinned manifest and observed-chain values. Its result is
explicitly unsigned: `authenticated` and `signatureVerified` remain `false`.
The parser does not fetch URLs, verify Sigstore material, or turn a current
state projection into historical proof.

## Develop

```bash
npm ci
npm run check
npm run build
npm run check:publish
```

`check:publish` runs the package's normal `npm pack` lifecycle, installs that
exact tarball into a fresh temporary strict NodeNext project, and type-checks
and executes imports from the root, `caip`, `liquidity`, `messages`,
`provenance`, and `registry` public entry points. The temporary package and
consumer are removed after the gate.

Regeneration uses the package's pinned local Buf CLI:

```bash
npm run generate
npm run check
npm run build
```

The generator exports the repository's locked Buf module, runs
`@hyperweb/telescope`, and records a digest over the protobuf sources, Buf
locks, generator, and npm lock. CI rejects stale generated code and stale
distribution files.

`@hyperweb/telescope` is generation-only and is not shipped in the runtime
package. Its current CLI dependency tree contains audit findings in old glob
and interactive-console packages, so run generation only against trusted local
protobuf sources. `npm audit --omit=dev` verifies the shipped dependency graph.

## Security boundary

Codecs provide serialization, not authority or safety policy. Before exposing
custom transaction controls:

- require a direct signer and confirm the RPC-reported chain ID;
- simulate, set explicit fees and limits, and validate every chain response;
- do not expose `MsgRotateKey` until the chain verifies its
  `authorization_signature`; and
- do not expose substrate attestations until adapter identity equality and the
  versioned, exhaustive link commitment are enforced server-side.

The mainnet dashboard uses this SDK to validate CAIP account IDs, and
cross-checks the chain's read-only `AccountIdentifier` projection when that
query is available. It still exposes only standard bank sends, so it does not
load the custom transaction registry into that path.

## Standards

- [CAIP-2](https://standards.chainagnostic.org/CAIPs/caip-2)
- [CAIP-10](https://standards.chainagnostic.org/CAIPs/caip-10)
- [Cosmos CAIP-2 namespace draft](https://namespaces.chainagnostic.org/cosmos/caip2)
- [Cosmos CAIP-10 namespace draft](https://namespaces.chainagnostic.org/cosmos/caip10)
- [CosmJS](https://github.com/cosmos/cosmjs)
- [Telescope](https://github.com/hyperweb-io/telescope)
