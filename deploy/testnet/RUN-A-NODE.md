# Run a `zerone-testnet-1` Node — Paused

The former bootstrap and validator instructions are retired.

`zerone-testnet-1` remains live as a legacy, play-value network, but its running
binary predates the consolidated source on `main`. The current branch includes
consensus-sensitive changes that require the named
`consolidation-safety-v1` upgrade. Starting a node from current source against
the legacy genesis has not been compatibility-tested and is not authorised.

For read-only endpoint checks, use [JOIN.md](JOIN.md). Do not run
`node-bootstrap.sh`, create a validator, or replace a running network binary
from this source head.

A safe operator guide requires:

1. an exact release commit and reproducible binary digest;
2. the canonical live genesis representation and peer identities;
3. a governance-approved upgrade height;
4. pre-upgrade state and migration-marker audits; and
5. post-upgrade consensus and query verification.

General validator preparation that does not connect or mutate a network is
documented in [the validator guide](../../docs/VALIDATOR-GUIDE.md).
