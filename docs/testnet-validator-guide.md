# Zerone Testnet Validator Guide — Retired

`zerone-testnet-1` is a live legacy playground, but there is no validator join
packet authorised for this consolidated source head. The running network
predates the ordered H1 `consolidation-safety-v1`, H2
`founder-renunciation-v1`, and H3 `sdk-0.53-ibc-10` boundaries; current `main`
must not be treated as a drop-in validator or genesis-replay binary for it.
H1 preserves vesting_rewards V1 and H2 alone advances it to V2.

Do not use cached copies of the former guide. In particular, do not download
genesis data or binaries from the old `nickkpope/zerone` or
`zerone-chain/zerone` locations, connect to the former `80.78.19.135` peer, or
submit a standard Cosmos `create-validator` transaction. None of those steps is
an authorised Zerone deployment path.

The legacy public RPC may be used for observation. Its existence is not binary
provenance or permission to join. Current sources of truth:

- [legacy testnet status](../networks/zerone-testnet-1/README.md)
- [validator operations and custom registration](VALIDATOR-GUIDE.md)
- [source-release and relaunch posture](../README.md#release-posture)
- [local development economics](testnet-economics.md)

For local experimentation, use the repository's localnet tooling. Local keys,
balances, peers, and faucet state are disposable development fixtures and
confer no standing on a public network.

A replacement validator guide must ship with a signed release manifest,
canonical genesis representation, authority chain, peer set, upgrade height,
and an explicit compatibility decision. Until then, this page is a safety
notice, not an invitation to join.
