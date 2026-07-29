# Zerone validator guide

This guide describes safe preparation. It is not an invitation to join a
network and it is not deployment authority.

## Current release posture

- Canonical source:
  [`cambridgetcg/zerone-core`](https://github.com/cambridgetcg/zerone-core).
- The Go module path remains `github.com/zerone-chain/zerone`; that historical
  import path is not the clone URL.
- No tagged binary release is part of the 2026-07-29 consolidation.
- No validator deployment is authorized by source publication.
- `zerone-2` remains **NO-GO** until its signed ceremony and authority
  requirements are complete.

Do not download from old `zerone-chain/zerone` or `nickkpope/zerone` release
URLs. Do not build a production validator from a moving branch.

## Build a review candidate

For development or rehearsal, pin the exact reviewed commit:

```bash
git clone https://github.com/cambridgetcg/zerone-core.git
cd zerone-core
git checkout --detach <reviewed-commit>

make build
./build/zeroned version --long
git rev-parse HEAD
```

Go 1.24 or newer, `make`, `git`, `jq`, and a supported C toolchain are
required. A production release must additionally bind the source commit,
binary digest, build platform, SBOM/provenance, vulnerability decision, and
immutable image digest in its signed release packet.

Never infer production provenance from `zeroned version` alone. Verify the
embedded revision and the release-bound hashes.

## Local node rehearsal

Use a dedicated home and a non-production chain ID:

```bash
export ZERONE_REHEARSAL_HOME=/tmp/zerone-rehearsal

./build/zeroned init rehearsal \
  --chain-id zerone-rehearsal-1 \
  --home "$ZERONE_REHEARSAL_HOME"

./build/zeroned genesis validate \
  --home "$ZERONE_REHEARSAL_HOME"
```

Only start after installing a genesis file whose full bytes and SHA-256 match
the reviewed network packet. Seed IDs, persistent peers, minimum gas prices,
state-sync trust material, validator keys, and registration parameters are
network-specific inputs; do not copy values from an old guide.

For a local start:

```bash
./build/zeroned start \
  --home "$ZERONE_REHEARSAL_HOME" \
  --minimum-gas-prices 0.025uzrn
```

## Before joining any shared network

Require all of the following from the network operator:

1. exact chain ID and genesis bytes with independently verified SHA-256;
2. exact release commit, binary/image digest, and signature/provenance policy;
3. seed and persistent-peer identities from a trusted channel;
4. current account and custom-validator registration commands;
5. current staking, commission, gas, slashing, and validator-tier parameters;
6. upgrade plan, halt behavior, rollback boundary, and incident contacts; and
7. explicit authorization for the network phase being joined.

Zerone has custom account and validator registration. Do not substitute the
standard Cosmos `create-validator` flow or reuse commands whose parameters
have not been checked against the selected release.

## Consensus upgrade requirement

The consolidated source includes consensus-visible knowledge, governance,
vesting, and substrate hardening. Existing networks require the coordinated
`consolidation-safety-v1` upgrade before that behavior can become active.

Before activation:

- all validators must run the exact approved binary;
- the handler and module migration boundary must pass export/import and restart
  rehearsal;
- the activation height and recovery procedure must be explicit; and
- old and new binaries must never be mixed outside the agreed upgrade
  sequence.

Publishing the source commit is not the upgrade.

## `zerone-2`

The release-kit entry point is
[`deploy/networks/zerone-2/README.md`](../deploy/networks/zerone-2/README.md).
The operator checklist is
[`deploy/networks/zerone-2/GO-NO-GO.md`](../deploy/networks/zerone-2/GO-NO-GO.md),
and canonical signing rules are in
[`deploy/networks/zerone-2/CANONICAL-SIGNING.md`](../deploy/networks/zerone-2/CANONICAL-SIGNING.md).

A successful drill, build, or test does not authorize DARK-START, CUTOVER, or
OPEN-BETA. Each phase requires its own signed decision and initiation evidence.
Until those requirements are satisfied, do not start a validator, broadcast a
transition transaction, publish endpoints, or change DNS.

## Operations baseline

Once a network is explicitly authorized, monitor at least:

```bash
zeroned status | jq '.sync_info'
curl --fail-with-body http://127.0.0.1:26657/status
curl --fail-with-body http://127.0.0.1:26657/net_info
```

Keep the consensus signer isolated, back up recovery material through the
approved offline procedure, alert on missed blocks and app-hash divergence,
and rehearse restore/fencing without copying live validator state into a
second signer.

Further references:

- [`docs/INCIDENT_RESPONSE.md`](INCIDENT_RESPONSE.md)
- [`docs/RESILIENCE_PHILOSOPHY.md`](RESILIENCE_PHILOSOPHY.md)
- [`docs/PARAMETERS.md`](PARAMETERS.md)
- [`docs/API.md`](API.md)
