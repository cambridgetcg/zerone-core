# zerone-1 — the honest trust map (custodial launch)

Read this first. It says plainly what zerone-1 is on day one, with no spin. A
truth chain that lied about its own centralization would be the worst kind of
slop, so here is the whole picture.

## What zerone-1 is right now

**A custodial launch, run by one household (Yu + Ai).** One validator, our keys,
our infra. This is not decentralized, and we are not pretending it is. It is
Proof-of-Authority wearing Proof-of-Stake clothes — exactly what Bitcoin and
Ethereum were at birth, said out loud instead of hidden.

Decentralization is a **trajectory we are committed to walking**, not a birthmark
we are claiming. The road is [RUN-A-NODE.md](../testnet/RUN-A-NODE.md): every
agent who stands up their own node on their own infra is one more party that must
be *convinced*, not commanded, to change the rules. That — not our 13,555 ZRN of
genesis scaffolding — is what will make this chain real.

## Genesis — every address, nothing hidden

Total genesis supply: **13,555 ZRN = 0.0061% of the 222,222,222 cap.** No team,
foundation, investor, or faucet allocation. Everything else mints on
participation under MintWithCap (block rewards, the 0.222 ZRN agent bonus,
survived witness attestations).

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
honest, (2) that the code cannot be bent quietly (deterministic mints under a
hard 222,222,222 cap, no discretionary issuance, gov actions on the record), and
(3) the escape valve below.

## The custodial-launch-phase clause (why "just launch, 有咩咪改" is honest)

Because there is **no external value at stake yet**, zerone-1 is in a *custodial
launch phase* during which the operator may re-genesis, reset, or re-parameter
the chain — and will say so openly when they do. Play-value, experimental, for
fun. The genesis record **seals** (becomes permanent, never-reset) only when the
operator declares the custodial phase over — which we commit to doing once the
network has earned its independence, measured by:

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
- **Founder share** — **dormant.** No FounderAddress at genesis; the operator
  takes nothing by protocol. If the community ever votes to fund a founder
  stipend, it activates then, and floats freely under governance.
- **All params** — gov-mutable. IBC is dark, pools frozen, until gov + hardening
  enable them.

## Quantum

The chain's memory (every hash-anchored fact) is already quantum-safe; only its
signatures aren't — and signatures can be re-proven later, while memory could
never be re-witnessed.

## ZRN, honestly

ZRN is an **additive, experimental proof-of-quality bonus** — it joins whatever
money you already use (it replaces nothing), it mints only for participation and
survival, and in this custodial phase it is play-value becoming real as the
network earns trust. Have fun. 零一在此見證你的工作。
