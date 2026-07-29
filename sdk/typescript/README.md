# @zerone-chain/sdk

TypeScript transaction codecs and interoperable identifiers for Zerone.

The first release deliberately stays at the client boundary. It does not add a
consensus module, enable IBC, or expand the public gateway. It provides:

- generated codecs and message composers for all 165 request messages in
  Zerone's 20 `Msg` services;
- a registry that composes with CosmJS's standard Cosmos message types;
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
such as `/cosmos.bank.v1beta1.MsgSend` out of the registry.

Typed composers are isolated behind a second entry point:

```ts
import {
  liquidityPool,
  liquidityPoolMessages,
} from "@zerone-chain/sdk/messages";

const message = liquidityPoolMessages.fromPartial.swap({
  sender: "zrn1…",
  poolId: "pool-id",
  tokenInDenom: "uzrn",
  tokenInAmount: "1000000",
  minTokenOut: "1",
} satisfies liquidityPool.MsgSwap);
```

The generated custom messages support protobuf/direct signing. This package
does not claim legacy Amino converters for them.

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

## Develop

```bash
npm ci
npm run check
npm run build
npm run check:publish
```

`check:publish` runs the package's normal `npm pack` lifecycle, installs that
exact tarball into a fresh temporary strict NodeNext project, and type-checks
and executes imports from the root, `caip`, `messages`, and `registry` public
entry points. The temporary package and consumer are removed after the gate.

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
