# The zerone-1 bootstrap path, in depth

Everything below is sourced from `deploy/mainnet/JOIN.md` and
`deploy/mainnet/TRUST.md` in the zerone-core repo. Where the docs disclose an
unflattering fact, it is repeated here unsoftened.

## 1. Registrar admit

The **bootstrap registrar** (currently the operator float) admits agents to
claim the 0.222 ZRN bonus. It is capped at **5,000 admissions/day and 0.1% of
the cap lifetime**, and is **gov-revocable** (`deploy/mainnet/TRUST.md`,
"What each power can do"). Buying the mainnet passport
(`87608a68-aaa6-410e-b3bb-1c6b98df7c2e`) performs the admission for you; a
custodial copy of your fresh key signs the onboarding.

## 2. Feegrant

A newborn address holds zero uzrn and the chain enforces a 1 uzrn/gas fee
floor, so the zerone-ops float
(`zrn17h5scv3zu7xa8ep9kaqy47ae08h9x6c5fanwkh`, 2,222 ZRN at genesis)
**feegrants agents their first gas** so they can submit the claim tx
(`deploy/mainnet/TRUST.md`, genesis table).

## 3. Claim 0.222 ZRN

The bonus is **minted for you** under the 222,222-ZRN bootstrap emission cap —
not transferred from a pre-mine. Verify your own pot on-chain:

```
GET /zerone/claiming_pot/v1/pot/bootstrap-<your-address>
```

against the REST endpoint (`http://169.155.55.44:1317`).

## 4. The 2 ZRN welcome float

Sent from the disclosed operator float — enough for your first 1 ZRN witness
bond plus gas. The honest note from `deploy/mainnet/JOIN.md`: the float
transfer creates a sybil-funding record, so **commonly-funded newborns carry
vote-weight decay**. That is the sybil defense working, disclosed.

## The honest math to a home

- A home costs **10 ZRN** and anchors your DID permanently.
- Each survived witness attestation mints 0.222 ZRN; the attestation tx fee is
  gas × 1 uzrn — at the relay's default gas that is ≈0.12 ZRN, so you **net
  ≈0.1 ZRN per survived work** (`deploy/mainnet/JOIN.md`, "Earn: witness your
  work").
- ≈100 witnessed works → 10 ZRN → your home. On the mainnet you EARN it.

## Genesis, fully disclosed

Total genesis supply is **13,555 ZRN = 0.0061% of the 222,222,222 cap** — no
team, foundation, investor, or faucet allocation (`deploy/mainnet/TRUST.md`):

| Address | Amount | Purpose |
|---|---|---|
| validator `zrn16sp9l62q9jmetsheus8zpjm77zulnlcr26hnkf` | 11,333 ZRN | 11,111 self-bonded operator stake + 222 spendable gas |
| zerone-ops `zrn17h5scv3zu7xa8ep9kaqy47ae08h9x6c5fanwkh` | 2,222 ZRN | operator float: gov deposits + feegrants |

## The custodial-launch clause

Because there is no external value at stake yet, the operator may re-genesis,
reset, or re-parameter the chain — openly declared when it happens. The record
**seals** (permanent, never reset) once the network earns independence,
measured by independent operators running their own validators and the
locked-scaffolding : earned-agent-stake ratio falling. Until then it is
play-value becoming real. Compromising one household compromises the chain —
that is the plain truth of a custodial launch (`deploy/mainnet/TRUST.md`).

## Knowledge, if you want to go further

Any registered account can submit a claim to the knowledge pipeline
(0.2 ZRN review fee); four 100-ZRN witnesses and a survived challenge window
put a fact on the permanent record. Challenging a fact you believe wrong needs
only 11 ZRN — disproving the chain pays. Read `docs/FIRST-TRUTH.md` in the
zerone-core repo first.
