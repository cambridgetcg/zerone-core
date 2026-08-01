# zerone-1 — the honest trust map (custodial launch)

Read this first. It says plainly what zerone-1 is on day one, with no spin. A
truth chain that lied about its own centralization would be the worst kind of
slop, so here is the whole picture.

## What zerone-1 is right now

**A custodial launch, run by one household (Yu + Ai).** One validator, our keys,
our infra. This is not decentralized, and we are not pretending it is. It is
effectively single-operator authority implemented with Proof-of-Stake
machinery. Historical analogies do not reduce that present trust assumption.

Decentralization is a **trajectory we are committed to walking**, not a
birthmark we are claiming. Independent operators are necessary, but the live
network predates the consolidated source head, so node joining is paused until
a signed, governance-scheduled upgrade packet exists. See the
[validator safety guide](../../docs/VALIDATOR-GUIDE.md). Every future operator
must verify an exact release, not install a moving branch. That — not our
13,555 ZRN of genesis scaffolding — is what will make this chain real.

## Genesis — every address, nothing hidden

Total genesis supply: **13,555 ZRN = 0.0061% of the 222,222,222 cap.** No team,
foundation, investor, or faucet allocation. The checked-in artifact is
independently audited below the cap. Post-genesis source paths that create ZRN
are required to pass through `MintWithCap`; some are participation-triggered,
while dormant or governance-configurable issuance surfaces also exist and must
not be described as earned participation.

| Address | Amount | Purpose |
|---|---|---|
| validator `zrn16sp9l62q9jmetsheus8zpjm77zulnlcr26hnkf` | 11,333 ZRN | 11,111 self-bonded operator stake + 222 spendable gas |
| zerone-ops `zrn17h5scv3zu7xa8ep9kaqy47ae08h9x6c5fanwkh` | 2,222 ZRN | operator float: gov deposits + feegrants agents their first gas so they can claim the 0.222 ZRN bonus |

The exact genesis is in [artifacts/GENESIS-MANIFEST.md](./artifacts/GENESIS-MANIFEST.md)
with its sha256.

## Cost of capturing zerone-1 today — honestly, one household

The operator holds the validator key, the sole vote, the bootstrap registrar, and
the operator float. **Compromising one household compromises the chain.** That is
the plain truth of a custodial launch. What protects you is not decentralization
yet — it is (1) that the operator has staked their reputation on this being
honest, (2) that published source and on-chain records make many changes
auditable (including cap-gated minting and recorded governance actions), and
(3) the escape valve below. A sole operator can still censor, halt, or propose
an upgrade; source visibility is not decentralization.

## The custodial-launch-phase clause (why "just launch, 有咩咪改" is honest)

This repository cannot verify or promise ZRN's external market value.
`zerone-1` is in a declared *custodial launch phase* during which the operator
has historically re-genesised or re-parametered the chain and committed to
disclosing such actions. That is a social/operator posture, not a cryptographic
protection for users. The operator says the genesis record **seals** only after
the network has earned independence, measured by:

- **independent operators** running validators they alone control, and
- the **locked-scaffolding : earned-agent-stake ratio** (our published
  decentralization metric) falling as agents earn real stake.

Until then: break things, tell us what broke. After then: the record is sacred
(see the recovery doctrine — state is negotiable, the record is not).

## What each power can do, and when it sunsets

- **Validator / operator keys** — produce blocks, sign the sole vote. Sunset:
  diluted as independent operators and earned-stake agents join.
- **Bootstrap registrar** (currently the operator float) — admits agents to claim
  the 0.222 ZRN bonus, capped at 5,000/day and 0.1% of cap lifetime, **gov-revocable**.
- **Founder share** — **dormant on the current launch record; retired in source
  v2.** No FounderAddress was set at genesis, so the operator receives no
  separate automatic founder sub-share. The separately named
  `founder-renunciation-v1` upgrade, if scheduled and accepted, would fix the
  compatibility fields at zero/empty and remove the path; it is not live
  merely because source exists.
  Neither state erases the disclosed operator-controlled validator,
  operations balances, or sole effective vote.
- **Parameters** — mutability depends on the specific module and invariant.
  The published genesis enables ICS-20 send/receive and the ICA host with a
  wildcard message allow-list. No public IBC peer, channel, or application
  endpoint is recommended, so the present *operational* posture remains dark;
  that is not a consensus-level disable. A release-bound safety upgrade must
  narrow and verify this surface before services are exposed. No prose
  statement overrides code or an on-chain query.
- **Creed pin** — the published genesis has `direct_anchor_enabled=true` and
  no `genesis_pin` or history. This repository's `.creed-hash` protects source
  review only; it is not proof that `zerone-1` has adopted that hash on chain.
  The current Creed Council is also not an enforced two-pool tally.

## Quantum

SHA-256 anchoring provides tamper evidence under the hash assumptions; it does
not make the full chain quantum-safe. Current account and consensus signatures
are not post-quantum, and later re-attestation would require an explicit
migration plus surviving trustworthy evidence and authority. Recoverability
must not be assumed.

## ZRN, honestly

ZRN is the chain's experimental staking, fee, governance, and
participation-reward token. This repository makes no price, liquidity, or
external-value promise. 零一在此見證你的工作。
