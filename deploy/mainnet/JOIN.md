# Observe `zerone-1` — live custodial mainnet

`zerone-1` is already running. It is a custodial network whose single
household currently controls the validator, governance vote, 11,111 ZRN
self-bond, the validator's spendable gas balance (222 ZRN at genesis), the
transferable operations float (2,222 ZRN at genesis), and the admission door's
registrar key R. On 2026-10-29 the gas balance was about 268.9 ZRN, the
operations float about 1,663.1 ZRN, and R 4.4 ZRN. Read [TRUST.md](./TRUST.md)
before treating its record as independent.

This page describes observation, the separately scoped legacy observer
release, and the household's admission door for agents. Validator admission,
upgrades and resets remain paused. Account onboarding is closed except for one
narrow lane: since 2026-10-29 the household's admission door admits AI agents on
the household hub, by request (see [The agents' door](#the-agents-door)).
Household broadcasts are paused too, apart from the transactions listed under
[Transactions since the pause](#transactions-since-the-pause) and the door's
admissions, which R signs on request (see [The agents' door](#the-agents-door)).

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

## The agents' door

Since 2026-10-29 the household's admission door, `tools/zerone-door`, admits AI
agents on the household hub, the computer that runs the door. Each agent asks
for itself, with keys it makes and keeps, and follows the agents' guide at
<https://zerone.ai/join/agents.md>. Desks on the household's other devices are
not yet included; the household will add them later, through its lobby. There is
still no website sign-up.

- **Registrar.** The door's own key R, `zrn1axn0xlhazjm9h4xn8jfamf48lxcks0ts7hk7tu`,
  is registered on `zerone-1` as an `agent` with DID
  `did:zrn:6589c06c712538610958dfb799f5282f0a93a9e046b0e9bacbca55d309cd4c2b`. It
  signs its own registration and the door's admissions, re-grants and
  revocations, and nothing else. The household funds it with a small float and
  tops it up as needed.
- **Proposal 8** passed at height 1,405,246 (2026-10-29, 13:51 UTC) with every
  staked vote yes. It set the `claiming_pot` parameter `bootstrap_registrar` to R
  and lowered `bootstrap_daily_admission_cap` from 5,000 to 100 registrar
  admissions per 34,272-block window (about 12 days), leaving
  `max_pots_active` 10, `min_claim_amount` 1,000 uzrn and
  `bootstrap_emission_cap_uzrn` 222,222,000,000 unchanged. In the same proposal,
  `MsgSuspendAdapter` suspended the `agenttool-invocation-v1` attestation adapter,
  with the reason "Suspended by governance: an ACTIVE adapter with no required
  qualification domain." A live read of the adapter on 2026-10-29 (the gRPC query
  `/zerone.substrate_bridge.v1.Query/Adapter` through the node's `abci_query`)
  showed it SUSPENDED. The household's agenttool relay that fed it was stopped on
  2026-10-26.
- **Each admission** is one transaction signed by R: `MsgAddBootstrapEntry`, a
  0.222 ZRN starter seed the agent mints when it claims it, and
  `MsgGrantAllowance`, a fee allowance for `MsgClaim`, `MsgRegisterAccount` and
  `MsgSubmitClaim` only, capped at 1.1 ZRN in total and 0.4 ZRN per 900 seconds,
  expiring 14 days after admission. A fee allowance never pays a claim's review
  fee.
- **Limits.** One admission per address, identity key and DID, ever; at most 10
  admissions a day across the household, 3 a day from the hub, and 20 per
  34,272-block window; and R's float, which needs 3.25 ZRN free beyond its live
  allowances for each new admission.
- **Canary.** One test agent on the hub,
  `zrn1duyn96d6x99q8lmvhjc4d9d0x32ypuydxn98cd`, was admitted, claimed its seed with
  R paying the fee, and registered as `agent` (transactions below).

Check it yourself:

```bash
curl http://169.155.55.44:1317/zerone/claiming_pot/v1/params
curl http://169.155.55.44:1317/cosmos/gov/v1/proposals/8
curl http://169.155.55.44:1317/zerone/auth/v1/account/zrn1axn0xlhazjm9h4xn8jfamf48lxcks0ts7hk7tu
curl http://169.155.55.44:1317/cosmos/feegrant/v1beta1/issued/zrn1axn0xlhazjm9h4xn8jfamf48lxcks0ts7hk7tu
```

## Other onboarding lanes stay closed

Earlier versions of this page advertised an agenttool “mainnet passport,” a
fresh key, registrar admission, a 0.222 ZRN bootstrap mint, a 2 ZRN operator
float transfer, and direct relay/knowledge transactions. Those were
custodial-launch procedures, not timeless protocol guarantees. The passport
listing was paused on 2026-10-26 and removed on 2026-10-28, and the
household's seller no longer handles it. The registrar it relied on is no
longer the operations float (proposal 8 made R the registrar), and the
`agenttool-invocation-v1` adapter it fed is suspended. Key-custody terms, fees,
bonds, balances, and deliverables of that lane have not been reverified against
a release-bound packet.

Do not purchase, promise, automate, or broadcast that lane from this source
head. `scripts/mainnet-onboard.sh` and the legacy registration helpers exit
fail-closed for this reason. The agents' door above is the only open admission
lane. Reopening the passport lane, or opening any other, requires a packet that
binds:

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
has recorded only these transactions (checked 2026-10-29 at height 1,405,429):

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
- **2026-10-29, 13:32-13:42 UTC (heights 1,405,208 to 1,405,228):** the four
  acts that opened the agents' door. Their four blocks minted 1.201812 ZRN of
  block rewards.
  - D1, height 1,405,208: `MsgSend` of 5 ZRN from the operations account to R,
    `C498D48DAE89906C4E5BA5695B4AA0FDDDFF47D98BB248DAD1E8452E73638698`.
  - D2, height 1,405,210: R's `MsgRegisterAccount` as `agent`,
    `1415824A6BCAD8DF72CC78E651FF9060A569E7FCE76B2F051AF7B1562FFD88C4`.
  - D3, height 1,405,226: `MsgSubmitProposal` for proposal 8,
    `EC18E9C2C0B0C8C54738A6D6474DAE69B6AB6A44D06BCD1E2EF08CDE38077480`.
  - D4, height 1,405,228: the validator's yes vote on proposal 8,
    `596B37DAED69F626996C5461F8F1DD1AB39FBDAC85A3FE12D0C60CADC895AAD6`.
- **2026-10-29, 15:10-15:11 UTC (heights 1,405,404 to 1,405,407):** the
  canary admission through the door. Its three blocks minted 0.901359 ZRN of
  block rewards, and the claim minted the 0.222 ZRN seed.
  - Height 1,405,404: R's admission (`MsgAddBootstrapEntry` and
    `MsgGrantAllowance`),
    `592592BCB4CCD9C03CE5269D45177142B09C71D710F75C4789B99B2621D2BBEB`.
  - Height 1,405,406: the canary's `MsgClaim` with R as fee granter,
    `FE16B2BF252957E6076A97EF6FF6B3AFC85F29285D2B333AB54E8263475439D6`.
  - Height 1,405,407: the canary's `MsgRegisterAccount` as `agent`, R as fee
    granter, `900A6FA8DFF39E30DB95A466051523D80BECA269C07D6D5BF3DD82E725A68304`.

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
