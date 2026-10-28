# Observe `zerone-1` — live custodial mainnet

`zerone-1` is already running. It is a custodial network whose single
household currently controls the validator, governance vote, 11,111 ZRN
self-bond, 222 ZRN validator gas balance, and 2,222 ZRN transferable
operations float. Read [TRUST.md](./TRUST.md) before treating its record as
independent.

This page describes observation and the separately scoped legacy observer
release. Account onboarding, validator admission, upgrades and resets remain
paused. Household broadcasts are paused too, apart from the transactions listed
under [Transactions since the pause](#transactions-since-the-pause).

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
custodial-launch procedures, not timeless protocol guarantees. The passport
listing was paused on 2026-10-26 and removed on 2026-10-28, and the
household's seller no longer handles it. Key-custody terms, registrar state,
adapter state, fees, bonds, balances, and deliverables have not been
reverified against a release-bound packet.

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

## Transactions since the pause

Since 2026-09-09, when this page first said broadcasts were paused, `zerone-1`
has recorded only these transactions:

- **2026-10-17, 15:19-15:26 UTC (heights 1,370,872, 1,370,882 and
  1,370,886):** three `MsgSubmitExternalAttestation` transactions from the
  household's unattended agenttool relay. Supply rose 1.572366 ZRN: 0.906366
  ZRN of block rewards for the three blocks that carried them, and 0.666 ZRN of
  witness rewards released to the relay's account 201 blocks after each
  attestation. The relay was stopped on 2026-10-26.
- **2026-10-28, 10:45-10:51 UTC (heights 1,401,996 and 1,402,007):** a
  `MsgRegisterAccount` and a `MsgSubmitClaim` from one account. Their two
  blocks minted 0.600906 ZRN of block rewards.
- **2026-10-28, 11:48 UTC (heights 1,402,121 and 1,402,122):** the governance
  proposal and vote that sealed the training-fund mint. Proposal 7 passed at
  height 1,402,141 (11:58 UTC) and set `training_fund_base_reward` to `0`.
  The blocks that carried these transactions minted 0.600906 ZRN of block
  rewards. No training-fund disbursement has been claimed on `zerone-1`.

On `zerone-1` today, every block that carries a transaction mints a block
reward, so any further broadcast will also raise supply. Check the record
yourself:

```bash
curl -G http://169.155.55.44:26657/tx_search \
  --data-urlencode 'query="tx.height>1262000"' \
  --data-urlencode 'per_page="100"'
```

## Compatible legacy observers

A separately signed [legacy observer package](../legacy-observer/README.md)
can follow this existing chain using the identified SDK 0.50 application, a reviewed dependency patch, and
a fresh zero-power identity. Use only an immutable published release whose
manifest, checkpoint and rehearsal receipt verify. Building the observer
tooling from source does not itself create a released package.

This role restores a recent snapshot and follows new blocks. It neither
replays the complete history nor repairs the sole validator's open custody
finding. Its loopback-only RPC uses a `nop` mempool, and the release grants
no transaction, account, validator, reset or upgrade authority. The checkpoint
has a bounded bootstrap expiry; its two RPC URLs share one upstream.

## Validator operation and upgrades remain paused

Do not build moving `main` and install it on `zerone-1`. A legacy network must
first cross release-matched H1 `consolidation-safety-v1`, which preserves
vesting_rewards V1, and H2 `founder-renunciation-v1`, which alone advances it
to V2, before the H3 `sdk-0.53-ibc-10` transition represented by current
source. Source publication is not chain activation; see
[UPGRADES.md](../../docs/UPGRADES.md).

Joining as a validator or upgrading the live runtime requires a signed packet
binding the exact commit, binary digest, live genesis representation, peer
identities, upgrade height, rollback boundary, and post-upgrade verification.
The unchanged legacy observer package does not satisfy or waive that gate. See the
[validator safety guide](../../docs/VALIDATOR-GUIDE.md).

## Supply claim

The hard cap is 222,222,222 ZRN. The disclosed live genesis supply was 13,555
ZRN: 11,333 ZRN controlled by the launch validator and 2,222 ZRN controlled by
the operations account. There was no separate team, foundation, investor-sale,
research, or faucet allocation. These narrow facts do not erase the real
custodial power of the operator-controlled stake and float.
