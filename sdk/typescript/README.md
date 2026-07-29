# @zerone-chain/sdk

TypeScript transaction codecs and interoperable identifiers for Zerone.

The first release deliberately stays at the client boundary. It does not add a
consensus module, enable IBC, or expand the public gateway. It provides:

- generated codecs and message composers for all 165 request messages in
  Zerone's 20 `Msg` services;
- a registry that composes with CosmJS's standard Cosmos message types;
- generic CAIP-2 and CAIP-10 syntax handling;
- the Cosmos namespace's direct/hashed chain-reference algorithm;
- checksum-aware `zrn` account IDs;
- canonical CIDv1 preflight for new agent-home memory references;
- bounded feegrant builders for sponsor-funded onboarding; and
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

## Validate memory CIDs before signing

```ts
import { asZeroneMemoryCid, parseCanonicalCidV1 } from "@zerone-chain/sdk/cid";

const memoryCid = asZeroneMemoryCid(
  "bafzbeigai3eoy2ccc7ybwjfz5r3rdxqrinwi4rwytly24tdbh6yk7zslrm",
);
const details = parseCanonicalCidV1(memoryCid);
// details.codec, details.multihashCode, details.digestBytes
```

The helper requires CIDv1 in Zerone's chosen lowercase-base32 representation
and applies `x/home`'s current 256-byte text limit. CID itself permits other
multibase strings for the same identifier. Parsing validates only the content
address structure; it does not hash the referenced bytes or establish their
availability, authenticity, or confidentiality.

This helper is opt-in. The generated `MsgUpdateMemoryCID` codec still accepts a
plain string, while the Go CLI invokes the preflight automatically. Current
validator consensus continues to accept historical opaque memory references
until a coordinated upgrade defines and audits a chain-wide policy.

## Sponsor bounded onboarding fees

```ts
import { defineZeroneNetwork } from "@zerone-chain/sdk/caip";
import {
  makeBoundedFeeGrant,
  makeRevokeFeeGrant,
  makeSponsoredFee,
} from "@zerone-chain/sdk/feegrant";

const network = defineZeroneNetwork("zerone-1");
const granter = "zrn16sp9l62q9jmetsheus8zpjm77zulnlcr26hnkf";
const grantee = "zrn1qypqxpq9qcrsszg2pvxq6rs0zqg3yyc5s75sh2";

const grant = makeBoundedFeeGrant({
  network,
  granter,
  grantee,
  spendLimit: [{ denom: "uzrn", amount: "100000" }],
  expiration: new Date(Date.now() + 30 * 24 * 60 * 60 * 1000),
  allowedMessageTypeUrls: ["/zerone.claiming_pot.v1.MsgClaim"],
});

// Sign and broadcast `grant` as the granter. Standard feegrant codecs are in
// CosmJS's defaultRegistryTypes.

const fee = makeSponsoredFee({
  network,
  granter,
  amount: [{ denom: "uzrn", amount: "2500" }],
  gas: "200000",
});
// Sign and broadcast the allowed onboarding message as `grantee`, passing
// `fee` to SigningStargateClient.signAndBroadcast.

const revoke = makeRevokeFeeGrant({ network, granter, grantee });
```

The grant builder always nests a finite `BasicAllowance` inside an
`AllowedMsgAllowance`. It requires a positive spend limit, a future expiry,
and one or more exact protobuf message type URLs. It rejects duplicate or
noncanonical coins, wildcard-like URLs, self-grants, and a conservative set of
emergency, upgrade, governance, parameter, admin, freeze, unfreeze, and key
rotation controls.

Feegrant pays transaction fees only: it does not let the grantee sign messages
as the granter. The helper does not query, broadcast, or prove that the
allowance exists; applications should query the connected chain, show the
budget and expiry to both parties, and handle revocation. These guardrails are
client policy rather than new consensus validation, so lower-level Cosmos SDK
clients can still construct other allowance shapes. CosmJS applications can
use `setupFeegrantExtension` from `@cosmjs/stargate` for allowance queries.

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
