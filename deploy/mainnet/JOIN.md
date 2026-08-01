# Observe `zerone-1` — live custodial mainnet

`zerone-1` is already running. It is a custodial network whose single
household currently controls the validator, governance vote, 11,111 ZRN
self-bond, 222 ZRN validator gas balance, and 2,222 ZRN transferable
operations float. Read [TRUST.md](./TRUST.md) before treating its record as
independent.

This page is an observation surface, not an onboarding, broadcast, validator,
upgrade, or reset authorization.

## Read-only network surfaces

| Surface | Value |
|---|---|
| RPC (CometBFT) | `http://169.155.55.44:26657` |
| REST (LCD) | `http://169.155.55.44:1317` |
| Chain ID | `zerone-1` |
| Denom | `uzrn` (1 ZRN = 1,000,000 uzrn) |
| Published `artifacts/genesis.json` file SHA-256 | `c30a523b9764fb76c84a53d99fcdabb966d16e7a4d3f15426ab7af5e8576170e` |
| Canonical RPC `.result.genesis` representation SHA-256 | `16ac346f329d2a931ad9a7d51dbe9e35605482b006ef39b3ac7804376e9bcb66` |

Read-only checks:

```bash
curl http://169.155.55.44:26657/status
curl "http://169.155.55.44:1317/cosmos/bank/v1beta1/supply/by_denom?denom=uzrn"
```

Independently compare the RPC genesis representation, published artifact, and
[genesis manifest](./artifacts/GENESIS-MANIFEST.md). The manifest is
hash-bound but currently has no detached signature. These two hashes cover
different byte representations; do not substitute one when verifying the
other.

## Onboarding and transaction lanes are paused

Earlier versions of this page advertised an agenttool “mainnet passport,” a
fresh key, registrar admission, a 0.222 ZRN bootstrap mint, a 2 ZRN operator
float transfer, and direct relay/knowledge transactions. Those were
custodial-launch procedures, not timeless protocol guarantees. Their current
listing, key-custody terms, registrar state, adapter state, fees, bonds,
balances, and deliverables have not been reverified against a release-bound
packet.

Do not purchase, promise, automate, or broadcast that lane from this source
head. `scripts/mainnet-onboard.sh` and the legacy registration helpers exit
fail-closed for this reason. Reopening onboarding requires a packet that binds:

1. the exact source release and binary/image digest;
2. current chain height, app version, upgrade plan, and activation height;
3. genesis representation and trusted peer identities;
4. current registrar, adapter, fee, bond, reward, and balance state;
5. key-custody and recovery terms;
6. exact transactions plus pre/post-state checks; and
7. explicit operator authorization.

Permissionless funding correlations may still be recorded for analysis, but
this source line does **not** reduce governance vote weight from those records:
an untrusted sender must not be able to poison another wallet's vote.

## Node and validator operation is paused

Do not build moving `main` and install it on `zerone-1`. The pending lineage is
`consolidation-safety-v1` → `founder-renunciation-v1`, each with its own release
binary and later height. The current combined source refuses the older name
while vesting-rewards is still v1, so H1 requires its exact reviewed historical
binary. There is no `liquiditypool-safety-v2` handler.
Source publication is not chain activation.

Joining or upgrading a node requires a signed packet binding the exact commit,
binary digest, live genesis representation, peer identities, upgrade height,
rollback boundary, and post-upgrade verification. See the
[validator safety guide](../../docs/VALIDATOR-GUIDE.md).

## Supply claim

The hard cap is 222,222,222 ZRN. The disclosed live genesis supply was 13,555
ZRN: 11,333 ZRN controlled by the launch validator and 2,222 ZRN controlled by
the operations account. There was no separate team, foundation, investor-sale,
research, or faucet allocation. These narrow facts do not erase the real
custodial power of the operator-controlled stake and float.
