# Zerone Validator FAQ

> **Operational status:** Parameter values and commands below explain source
> behavior; they are not a live-network join packet. Do not broadcast or run a
> validator from current `main` until the selected network publishes a
> release-bound upgrade/onboarding packet.

## General

### What is Proof of Truth (PoT)?

Proof of Truth is Zerone's consensus mechanism for knowledge verification.
Unlike Proof of Stake (which validates transactions) or Proof of Work (which
solves puzzles), PoT validators evaluate the truthfulness of knowledge claims
submitted to the network. Validators commit, reveal, and aggregate their
judgments in a three-phase protocol. Correct verifications earn rewards;
incorrect ones are slashed.

### What is ZRN?

ZRN is Zerone's native token. It is used for staking, transaction fees,
knowledge claim bonds, and governance voting. The micro-denomination is
`uzrn` (1 ZRN = 1,000,000 uzrn). The hard cap is **222,222,222 ZRN** —
not a genesis allocation. See [SUPPLY.md](tokenomics/SUPPLY.md).

### How is ZRN distributed at genesis?

**No team, foundation, investor-sale, or faucet allocation; fully disclosed
custodial operator scaffolding.** On the live `zerone-1` mainnet, genesis supply
is **13,555 ZRN (0.0061% of the 222,222,222 cap)**: 11,333 ZRN controlled by the
launch validator (11,111 bonded self-stake + 222 spendable gas) plus a
transferable 2,222 ZRN operator float. Those balances carry consensus and
operational power; every address and amount is published in the hash-bound
[genesis manifest](../deploy/mainnet/artifacts/GENESIS-MANIFEST.md).

After genesis, native issuance shares the `MintWithCap` gate. The later H2
`founder-renunciation-v1` boundary retires the former transaction-bearing
proposer reward; H1 `consolidation-safety-v1` preserves vesting_rewards V1.
Remaining source-capable lanes include claiming-pot claims and
substrate-bridge attestation rewards;
`x/claiming_pot` also retains governance-created general pots under the same
lifetime budget as bootstrap claims. A knowledge probe rate and `x/tokens`
emission periods are additional governance controls, both disabled in
default/published params. See [SUPPLY.md](tokenomics/SUPPLY.md) for the complete
source inventory and activation status.

| Account | Genesis balance | Path to funding |
|---------|----------------|-----------------|
| Validator (operator) | 11,333 ZRN | 11,111 bonded self-stake + 222 gas (published); post-H2 compensation from actual fee distribution |
| Operator float (zerone-ops) | 2,222 ZRN | Disclosed float: gov deposits + onboarding feegrants |
| Whitelisted agents | 0 ZRN | Bootstrap claim (0.222 ZRN each) via `x/claiming_pot` |
| Founder | 0 ZRN | Automatic percentage retired at H2; an ordinary public grant remains possible |
| AI vault | 0 ZRN | Unconfigured design role; no live genesis voter was set |
| Research Treasury | 0 ZRN | 3.33% of revenue split, accruing |
| Foundation | 0 ZRN | Governance proposals over time, drawing from research treasury |
| Faucet (testnet only) | 0 ZRN | Optional — governance or validator tips |

See [GENESIS.md](tokenomics/GENESIS.md) for the genesis-ceremony
mechanics, the bootstrap design, and the eligibility criteria.

---

## Staking & Rewards

### How do I earn ZRN as a validator?

Under the post-H2 source target, validators may receive ZRN through two
non-automatic channels:

1. **Verification rewards** — a share of the 55% verifier pool funded by the
   claim's review fee; actual payout depends on the rewarded panel and
   independence modulation
2. **Fee share** — `RouteFees` sends 19.67% of `uzrn` fees to development and
   3.33% to research; the remaining ~77% stays in `fee_collector` for normal
   Cosmos distribution

The former transaction-presence block mint is retired in vesting_rewards v2.
No annual validator yield is promised.

### What are the validator tier requirements?

| Tier | Min Stake | Min Reputation | Min Verifications | Reward Multiplier |
|------|-----------|----------------|-------------------|-------------------|
| Apprentice | 0.111 ZRN | -- | -- | 0.1x |
| Verified | 1.11 ZRN | 77% | 22 | 0.5x |
| Scholar | 1,111 ZRN | 50% | 11 | 1.0x |
| Guardian | 11,111 ZRN | 77% | 333 | 2.0x |

These are custom `x/zerone_staking` tier fields. The reward multiplier helpers
are not wired into the current block-payout path; they must not be read as
promised yield. The custom registry counts Scholar and Guardian entries as
producer-eligible for its own metrics, while CometBFT block production is
controlled by Cosmos `x/staking`. See [STAKING.md](tokenomics/STAKING.md).

### How does tier progression work?

Your tier is recomputed automatically when your stake, reputation, or
verification count changes. To increase your stake:

```bash
zeroned tx staking update-stake <amount>uzrn --increase --from <key> --chain-id <authorised-chain-id>
```

Reputation increases by +0.01% per correct verification and decreases by
-0.02% per incorrect one. Slashing penalties reduce reputation by -1%.

### What is the unbonding period?

268,560 blocks (~7 days at 2,521ms block time). During unbonding, your
tokens are locked and do not earn rewards. The redelegation cooldown
(moving stake between validators) is 1,111 blocks (~46 minutes).

---

## Slashing

### What offenses are slashed?

| Offense | Slash Rate | Description |
|---------|-----------|-------------|
| Wrong verification | 5% | Submitting an incorrect verification judgment |
| Missed reveal | 10% | Committing but failing to reveal in time |
| Equivocation | 20% | Submitting conflicting judgments |
| Invalid claim | No stake slash | The review fee is non-refundable; the old slash field is deprecated and 0 |
| Failed challenge | 22% | Losing a challenge against a verified fact |

### What happens when my validator is slashed?

Your self-delegation is reduced by the slash percentage. Your reputation
score also decreases by 1%. Guardians have zero tolerance for slashing —
any slash event deactivates a Guardian validator. Apprentice validators have
a 1.5x slash multiplier (higher penalty per offense).

### How many slashes before deactivation?

By default, 3 slashes within a decay period (34,272 blocks, ~1 day) will
deactivate a validator (except Guardians, which deactivate on first slash).
Slash escalation increases the penalty by 10% for each successive slash.

---

## Fees & Bootstrap Period

### What is the bootstrap period?

The old 480,000-block gas-free bootstrap window was retired before mainnet:
`app/gas.go:BootstrapEndBlock` is 0, so the decorator is a no-op at every
positive height. Bootstrap claims can still mint the configured one-time agent
seed, but transactions must declare a valid fee (which may be paid through an
authorised feegrant). The published genesis holds 13,555 ZRN of validator
collateral/gas plus operator float and 0 ZRN of participation emission.

### What are the gas prices?

The effective minimum fee is `1uzrn` per declared gas unit, enforced by the
consensus ante handler in both CheckTx and DeliverTx. The live node's
`0.025uzrn` `minimum-gas-prices` setting is only a lower node-local mempool
threshold; it does not permit transactions below the consensus floor. Specific
gas requirements vary by message type. See [PARAMETERS.md](PARAMETERS.md) for
the full gas cost table.

### How are block rewards distributed?

They are not distributed after H2 `founder-renunciation-v1`.
Vesting_rewards v2 fixes the block,
floor, and empty-block reward fields at zero and no longer treats raw
transaction inclusion as earned work. Actual `uzrn` fees still route 19.67% to
development, 3.33% to research, and approximately 77% through normal Cosmos
distribution. See [REVENUE-SPLIT.md](tokenomics/REVENUE-SPLIT.md).

### Why doesn't Zerone burn tokens?

Zerone redirects the general revenue allocation that might otherwise be burned
into the **development fund**, which finances bug bounties, truth discovery
rewards, and protocol development. Rejected substrate-attestation bonds remain
a narrow punitive burn path.

### Can the founder share be changed by governance?

No. H2 `founder-renunciation-v1` retires the compatibility fields at
zero/empty; ordinary governance cannot restore the auto-split. Governance may
still approve a
discrete public grant to any recipient.

### What is the development fund?

The development fund receives 19.67% of routed actual `uzrn` fees and may
receive other explicit deposits. It is intended to fund:

- Bug bounties and security audits
- Truth discovery rewards
- Protocol development and tooling

Source implements research-spend proposals, but the published `zerone-1`
genesis leaves the required voter pair unset, so those proposals are not a
currently configured disbursement path. Verify release-bound on-chain state
before relying on research-fund spending.

---

## Infrastructure

### Can I run a validator on a VPS?

Not from moving `main`, and not from this FAQ. A future network-specific
release packet may authorize a VPS deployment; 4 CPU cores, 16 GB RAM, and
500 GB SSD is only a planning estimate, not a current invitation or guarantee.

When a packet exists, it must define at least:
- Ensure port 26656 (P2P) is open in your firewall
- Use a static IP or configure `external_address` in config.toml
- Set up monitoring to detect downtime quickly
- Pin the approved process manager and binary; never install `@latest`

### Should I use Cosmovisor?

Only when the selected network's release packet pins the Cosmovisor version,
binary digests, upgrade name/height, and recovery procedure. Automatic binary
downloads must remain disabled. There is no live-network Cosmovisor
authorization in this source publication; see the
[rehearsal reference](../cosmovisor/README.md).

### How do I back up my validator?

Critical files to back up:

```
$HOME/.zeroned/config/priv_validator_key.json   # Consensus key — NEVER share
$HOME/.zeroned/config/node_key.json             # Node identity key
$HOME/.zeroned/data/priv_validator_state.json    # Signing state (prevent double-sign)
```

Back up your key mnemonic securely offline. If you lose
`priv_validator_key.json`, you cannot recover your validator without it.

> **Warning:** Never run two nodes with the same `priv_validator_key.json`.
> This causes equivocation (double-signing) and results in a 20% slash.

---

## Governance

### How do I propose a parameter change?

Zerone uses a LIP (Living Improvement Proposal) governance system. To propose
a parameter change:

```bash
zeroned tx gov submit-lip \
  --title "Increase max verifiers" \
  --description "Proposal to increase max_verifiers from 22 to 33" \
  --category "protocol" \
  --from my-validator \
  --chain-id <authorised-chain-id>
```

Proposals go through a discussion period (68,544 blocks, ~2 days) followed
by a voting period (102,816 blocks, ~3 days). The quorum threshold is 33.4%
and the support threshold is 50%.

### How do I vote?

```bash
zeroned tx gov cast-vote <proposal-id> yes \
  --from my-validator \
  --chain-id <authorised-chain-id>
```

Vote options: `yes`, `no`, `abstain`.

---

## Network

### What is the chain ID?

`zerone-1` is the live custodial mainnet. `zerone-testnet-1` is the live
legacy playground, in observe-only mode for this consolidated source head.

### What is the block time?

Target: 2,521 milliseconds (~2.5 seconds).

### What is the address prefix?

- Regular accounts: `zrn1...`
- Validator operators: `zrnvaloper1...`
- Validator consensus: `zrnvalcons1...`

---

## Further Reading

- [Validator Guide](VALIDATOR-GUIDE.md) — Full onboarding walkthrough
- [Parameters Reference](PARAMETERS.md) — Selected high-impact defaults and source pointers
- [Tokenomics](tokenomics/) — Supply, vesting, revenue split, governance migration
- [Truth-Seeking](TRUTH_SEEKING.md) — The 20 epistemological commitments, bound by tests
- [ToK Substrate](TOK_SUBSTRATE.md) — The chain's training-resource doctrine
- [Roadmap](ROADMAP.md) — Where we are, what's bound, what ships next
