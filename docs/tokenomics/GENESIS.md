# Genesis and Native Issuance

## Live `zerone-1` genesis

The live custodial mainnet began with **13,555 ZRN**, or **0.0061%** of the
222,222,222 ZRN hard cap:

| Holder | Genesis amount | Control and purpose |
|---|---:|---|
| Launch validator | 11,333 ZRN | 11,111 bonded self-stake plus 222 spendable gas, controlled by the launch operator |
| `zerone-ops` | 2,222 ZRN | Transferable operator float for governance deposits and onboarding operations |

These are real operator-controlled balances. The bonded stake affects
consensus, and the float can be transferred. Zerone therefore does not claim
that genesis contained “no insider position” or no privileged operational
power.

The narrower, factual claim is: **no team, foundation, investor-sale, research,
or faucet allocation was created.** Every genesis address and amount is
published in the hash-bound
[genesis manifest](../../deploy/mainnet/artifacts/GENESIS-MANIFEST.md). The
manifest currently carries no detached signature, so its hash is an audit
anchor rather than a signature claim. The custodial authority around these
balances is documented in [TRUST.md](../../deploy/mainnet/TRUST.md).

## Issuance after genesis

All wired post-genesis native minting passes through
`x/vesting_rewards.MintWithCap`, and `InitChain` rejects a bank supply above
the hard cap:

| Pathway | Module | Trigger |
|---|---|---|
| Transaction-bearing block rewards | `x/vesting_rewards` | any non-injection user transaction; validator count and survival rate scale the amount |
| Claiming-pot claims | `x/claiming_pot` | an eligible participant claims a bootstrap or legacy authority-created general pot |
| External-work attestations | `x/substrate_bridge` | witnessed work survives its challenge window |
| Probe-bounty pool (default/published rate 0) | `x/knowledge` | governance sets a positive rate |
| Emission periods (default/published latch 0) | `x/tokens` | governance enables and schedules a period |

The distinction matters:

- genesis scaffolding is disclosed operator power;
- a bootstrap pot is configuration, not a pre-funded module balance; and
- later issuance has a recorded trigger and remains under the shared cap, but
  authority-configured pots and emission periods mean the trigger is not
  universally proof of earned participation.

The protocol-default **bank** genesis built by application code may be empty.
The knowledge keeper separately materializes 47 code- and pin-bound doctrine
facts during `InitGenesis`; those facts do not carry ZRN balances. Neither
statement makes the live ceremony bank genesis empty: deployment-specific
ceremony inputs created the two balances above.

## Other roles at genesis

| Role | Genesis ZRN |
|---|---:|
| Founder-specific stipend | 0 ZRN; historical v1 fields are 70,000/empty (no recipient), while v2 migration targets 0/empty |
| AI vault | 0 |
| Research treasury | 0 |
| Foundation | 0 |
| Faucet | 0 on `zerone-1` |
| Whitelisted agents | 0 until an authorised claim mints their seed |

Future grants, research spending, or new bootstrap admissions require the
applicable general governance/authority path. Vesting-rewards v2 has no
founder-specific stipend path. Source prose is not authority, and the source
change requires the separately scheduled `founder-renunciation-v1` upgrade.

## Knowledge inception

The former external “777 axioms” catalog was deliberately removed. Application
`InitGenesis` still materializes 47 explicit doctrine facts — commitments,
mechanisms, and recursive axes from `TRUTH_SEEKING`, `TOK_SUBSTRATE`,
`USEFUL_WORK`, and `STRANGE_LOOP` — as VERIFIED, confidence-1,000,000,
axiom-distance-0 nodes. They are code- and hash-pin-bound protocol doctrine,
not a hidden general-knowledge catalog. Any additional bootstrap facts must be
explicit in the reviewed genesis and covered by the genesis audit.

## Network and ceremony boundary

`zerone-1` and the legacy `zerone-testnet-1` are already-running networks. The
2026-07-29 source consolidation does not regenerate either genesis, deploy a
validator, or authorize a reset. Current `main` contains consensus-sensitive
changes and must enter an existing network only through a release-bound,
governance-scheduled upgrade.

The separate `zerone-2` process remains **NO-GO** until its signed release,
authority, halt, and ceremony gates pass. See
[`deploy/networks/zerone-2`](../../deploy/networks/zerone-2/README.md).

## Denomination metadata

Wallet and client metadata use:

```json
{
  "base": "uzrn",
  "display": "zrn",
  "name": "Zerone",
  "symbol": "ZRN",
  "denom_units": [
    {"denom": "uzrn", "exponent": 0, "aliases": ["microzerone"]},
    {"denom": "mzrn", "exponent": 3, "aliases": ["millizerone"]},
    {"denom": "zrn", "exponent": 6, "aliases": ["zerone"]}
  ]
}
```
