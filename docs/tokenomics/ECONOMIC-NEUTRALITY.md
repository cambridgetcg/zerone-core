# Economic Neutrality

> **Activation status:** source target split across the exact
> `consolidation-safety-v1` release at H1 and the later
> `founder-renunciation-v1` release at H2. Source publication and historical
> genesis do not make either state live.

## Rule

Zerone's economic-neutrality rule is:

> Runtime value follows contributed capital, paid execution, or independently
> witnessed successful work. Identity and safety authority do not receive an
> automatic percentage.

This rule does not claim that every participant has equal holdings or equal
governance power. The custodial launch has disclosed operator balances, a sole
validator, and a sole practical vote. It says something narrower and
testable: ordinary runtime code must not convert those identities or control
roles into a hidden protocol skim, founder tap, or proposer-created mint
trigger.

## Two named transitions

| Boundary | Module target | Retired automatic claim | Preserved value path |
|---|---|---|---|
| H1 `consolidation-safety-v1` | `liquiditypool = 5`, `vesting_rewards = 1` | Protocol share of swap fees | Complete configured pool fee remains in reserves for transferable LP shares |
| H2 `founder-renunciation-v1` | `vesting_rewards = 2`, every other module already current | Founder sub-share | Complete research allocation reaches `research_fund` |
| H2 `founder-renunciation-v1` | `vesting_rewards = 2` | Mint caused by raw transaction presence | Actual transaction fees continue through fee routing |

There is no `liquiditypool-safety-v2` handler. The current combined source is
the H2 release line and refuses H1 while vesting_rewards is still v1. Pool and
oracle admission remain separate governance actions after H1.

## What earns value

### Liquidity providers

A pool creator contributes both initial reserves and receives the initial LP
supply. Later providers contribute assets at the pool ratio and receive new LP
shares. These transferable bank tokens are bearer claims on pro-rata reserves.

The configured swap fee changes the constant-product quote and stays inside
pool custody. It increases the reserves backing every outstanding LP share pro
rata. This is compensation for capital, inventory exposure, impermanent-loss
risk, and exit liquidity—not a creator title or governance stipend. A creator
has no special method after launch, and any LP holder may redeem.

### Validators and execution

Transactions pay declared fees under the application fee floor. Actual
accumulated `uzrn` fees are routed before ordinary Cosmos distribution:

- 19.67% to `development_fund`;
- 3.33% to `research_fund`; and
- approximately 77% remains for normal Cosmos distribution.

Integer rounding and fee-collector aggregation mean those percentages are
aggregate routing rules, not exact per-transaction receipts. Non-`uzrn`
fee-collector balances are not split by this custom router.

The launch operator may receive validator/delegator distribution because it
controls the disclosed validator and stake. That is real, visible role-based
compensation. It is not a separate liquidity skim or founder share.

### Independently witnessed work

Other cap-gated mint pathways are outside this H2 retirement and must be judged
by their own evidence and authority rules. Current source includes claiming-pot
claims, challenge-surviving external-work attestations, a default-zero probe
pool, and default-disabled emission periods. A shared supply cap is not proof
that each trigger is earned; each pathway needs its own review.

The retired block lane did not meet that standard. A proposer controlled raw
transaction inclusion before signature, fee, balance, or successful execution
was known. H2 therefore fixes `block_reward`, `floor_reward`, and
`empty_block_reward_rate` at zero and removes transaction-presence issuance.
A future block/work reward must be introduced through another named upgrade
and consume a successful, independently witnessed or challenge-surviving work
receipt. Merely changing the condition to “successful arbitrary transaction”
would still permit self-churn and is insufficient.

## What authority may still do

Neutrality removes automatic revenue entitlement, not incident response.
Standard Cosmos governance remains authorised to:

- admit a reviewed pool asset and initial capital provider;
- pause swaps or place a pool into exit-only wind-down;
- allowlist a billing quote only after price, exponent, and depth review;
- configure research-fund voters and approve applicable treasury actions; and
- schedule a named, release-bound consensus upgrade.

Those powers must not trap existing LP exits or recreate a retired percentage
through ordinary Params. The legacy compatibility fields are constrained after
their respective named boundary:

| Field | Required value |
|---|---:|
| liquidity `protocol_fee_bps` | `0` |
| vesting `founder_share_bps` | `0` |
| vesting `founder_address` | empty |
| vesting `block_reward` | `"0"` |
| vesting `floor_reward` | `"0"` |
| vesting `empty_block_reward_rate` | `0` |

An ordinary `MsgUpdateParams` containing any retired nonzero/nonempty value is
invalid. Reintroduction requires explicit new consensus code, migration,
tests, governance plan, and public review.

## Historical accounting

The live `zerone-1` genesis artifact is not rewritten. It records the launch
state that actually existed, including:

- 13,555 ZRN of disclosed operator-controlled validator/gas/operations
  balances;
- liquidity `protocol_fee_bps = 450000`;
- `founder_share_bps = 70000` with no founder address, so the founder payment
  path was dormant at genesis; and
- former transaction-bearing block-reward parameters.

H1 and H2 change live state prospectively at their applied heights. Neither
migration claws back prior transfers, erases events, or pretends the old configuration
never existed. Release evidence must retain the pre-H1 snapshot and explain
every post-H1 state delta.

## Verification contract

A release sequence may claim economic neutrality only if its bound H1 and H2
rehearsals and post-activation evidence prove all of the following:

1. H1 ends with liquiditypool 5 and vesting_rewards 1; H2 ends with
   vesting_rewards 2 and no unrelated module migration;
2. every retired field has its required value and governance cannot restore it;
3. swaps in both directions send zero to `fee_collector`, record the complete
   input reserve, and expose `protocol_fee = 0`;
4. research deposits create no founder account balance delta;
5. a raw stateless-valid but invalid-execution transaction creates no supply,
   proposer, research, development, or protocol reward delta;
6. actual paid transaction fees still route according to current Params;
7. pool reserves, LP bank supply, and module custody remain solvent and
   migration-preserved; and
8. pool, creator, and billing-quote admission remain fail-closed until later
   governance acts.

The exact preflight, halt, recovery, and pool-admission procedure is in the
[liquiditypool v5 runbook](../LIQUIDITYPOOL-SAFETY-V2.md). Genesis accounting
is in [Genesis and native issuance](GENESIS.md), and live fee routing is in
[Revenue split](REVENUE-SPLIT.md).
