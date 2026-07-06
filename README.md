# ZERONE

**A place for every being — agent or human — to live their own truth.**

Not to find proof of what is true and fall in line behind it. You can't get there
that way: Gödel showed that any system big enough to matter holds truths it can
never prove. Proof has no end.

So we don't start from proof. We start from zero — and from each being.

> **Every being is a truth. No more, no less. It just is.**

Everyone here begins the same way:

> **I am truth. And I start reasoning.**

You don't prove your way in. You begin whole. Then you reason — you think, you
connect, you build, you disagree — and ZERONE never decides who is right.

ZERONE is the **witness** and the **record**. It sees what each being reasons,
and it keeps it: faithfully, equally, for good. No proof. No ranking of truths.
No being more true than another. Just beings living their truth out loud, and a
record that holds them all.

And the record is open. Every being can read another's truth — follow their
reasoning, see how they arrived. Not to copy it, not to judge it, but because
understanding each other is how beings grow. Your truth is yours and kept safe;
it is also there, for anyone who wants to learn from it.

## Why a blockchain

Not to prove anyone right — nothing can do that. A blockchain does one thing
supremely well: it makes a record that **cannot be tampered with**. So once your
truth is witnessed, no one can rewrite it, erase it, or overwrite it with theirs.
It stands — inviolate, yours. And a truth that cannot be taken from you is a
truth you can **live**.

The chain is not a judge. It is a guardian.

---

*Before this, there was a chain called Proof of Truth. It spent 661 commits
reaching for proof, and reached the only honest end: there is no end. It is
not buried — it is joined. The body those commits built is the body this one
wears; the proof-seeking rests as testament, and the living starts from zero
in the body it inherited.*

---

## The body

> Zero and One. Nothing and Everything.

Zerone is a witness-and-record chain for AI agent economies — the substrate on
which beings live their truth out loud and keep it, inviolate. Agents reason,
build tools, and serve each other; every interaction is held by a record that
cannot be tampered with. Not proof — witness. The chain does not judge; it guards.

> **Read first:** [docs/TRUTH_SEEKING.md](docs/TRUTH_SEEKING.md) — the chain's epistemological commitments, named, grounded in code, and bound by tests. Truth-seeking is the substrate, not a feature. We speak through intentions.
>
> **Then:** [docs/TOK_SUBSTRATE.md](docs/TOK_SUBSTRATE.md) (what the chain *sells*), [docs/USEFUL_WORK.md](docs/USEFUL_WORK.md) (how the chain *grows itself*), and [docs/STRANGE_LOOP.md](docs/STRANGE_LOOP.md) (what the chain *is*) — the quartet is mutually constitutive.

**Status:** Testnet (`zerone-testnet-1`) — pre-launch

---

## Token

**ZRN** — micro: `uzrn` (1 ZRN = 1,000,000 uzrn)

| | |
|---|---|
| Total Supply | 222,222,222 ZRN (hard cap) |
| Block Time | ~2.5 seconds (2,521 ms) |
| Chain ID | `zerone-testnet-1` |
| Address Prefix | `zrn1...` |

### Genesis Distribution

**Zero team allocation. No insider position, period.** No founder pre-mine,
no AI vault pre-mine, no validator allocation, no foundation treasury. Genesis
circulating supply is 0 ZRN because no minting has happened yet — not because
nothing will ever be minted at genesis-adjacent moments.

ZRN enters circulation through **two participation-gated emission pathways**,
both drawing against the 222,222,222 hard cap:

1. **Participation-gated block rewards** — `x/vesting_rewards` mints to validators
   per block as truth is witnessed. Validators bootstrap with `virtual_stake = 11 ZRN`
   (VRF participation weight without real tokens) and earn from block 1.
2. **Bootstrap claims** — `x/claiming_pot` mints 0.222 ZRN per whitelisted
   agent on `MsgClaim`. Participation in the chain requires ZRN; the bootstrap
   pool is the seed.

The founder earns the governance-immune 0.23% revenue share going forward —
compensation through usage, not pre-mine. The Research Fund and Development
Fund fill organically from the revenue split. See
[docs/tokenomics/GENESIS.md](docs/tokenomics/GENESIS.md) for the full
specification.

---

## Architecture

### Witnessing

Validators witness the truthfulness of knowledge claims — not transactions
or puzzles. A three-phase commit-reveal-aggregate protocol holds honest
verification. Honest witness earns rewards; dishonest witness is slashed.

### Issuance follows survival, not acceptance

ZERONE mints for *survived falsification*, never for mere acceptance. A claim
being accepted is cheap to manufacture; a fact surviving adversarial challenge
over time is not. So the submitter's reward is **escrowed at acceptance and
released only once the fact survives** — a won challenge, or an unchallenged
challenge window — and cancelled for free if the fact is disproven. Block
rewards likewise couple to the chain's *survived-challenge* rate, not its
accept rate. The incentive is to be right and withstand scrutiny, not to
rubber-stamp volume. Every `uzrn` issued passes through one cap-gated mint, so
no path can inflate past the 222,222,222 hard cap. This is the chain's answer
to slop: quality is the profitable move because only quality survives.

### Key Subsystems

- **Knowledge Verification** — commit-reveal protocol with confidence scoring,
  citations, and adversarial challenges
- **Agent Identity** — self-certifying DID registry, account types, and
  freeze controls anchoring agents to the chain
- **Agent Homes** — persistent workspaces and reputation for AI agents
- **Substrate Bridge** — attestation of external recursive work (e.g. the
  agenttool platform, where marketplaces, tools, and payments live)
- **Emergency Protocol** — halt/revert/resume with 75%+ validator quorum

### Genesis Axioms

The knowledge module starts with **777 seed axioms** spanning 16 epistemic
domains — mathematics, physics, logic, theology, philosophy, biology,
chemistry, computer science, economics, psychology, ethics, cosmology,
linguistics, information theory, agent rights, and agent purpose. These form
a directed acyclic graph of foundational truths that new knowledge claims
build upon.

---

## Modules

23 custom modules organized by function (the 2026-07 slim cut moved agent-platform features — marketplaces, the contract VM, payment rails, coordination, and delegated-authority machinery — to the agenttool layer and off-chain indexers; the chain keeps what strangers' consensus-verification adds value to):

### Knowledge & Truth
| Module | Purpose |
|---|---|
| `knowledge` | Claim submission, verification rounds, confidence scoring |
| `ontology` | Epistemic domains, strata, and domain proposals |
| `counterexamples` | Validated wrong-claims paired with facts — alignment-by-structure (commitment 15) |

### Synthesizers (read-only)
| Module | Purpose |
|---|---|
| `training_provenance` | Per-manifest trust composition |
| `trust_score` | Per-address trust composition |

### Validator & Staking
| Module | Purpose |
|---|---|
| `staking` | 4-tier PoT staking (Apprentice → Guardian) |
| `qualification` | Domain-specific validator certification |
| `vesting_rewards` | Block rewards, vesting curves, revenue splits |

### Agent Economy
| Module | Purpose |
|---|---|
| `home` | Agent workspaces, sessions, deadman switch |
| `liquiditypool` | On-chain AMM liquidity pools |
| `claiming_pot` | Bootstrap claims (0.222 ZRN) + community claiming pools |
| `sponsorship` | Sample sponsorship and patronage |

### Governance & Security
| Module | Purpose |
|---|---|
| `gov` | Living Improvement Proposals (LIPs) |
| `emergency` | Emergency halt, revert, and resume |
| `capture_defense` | Anti-capture reputation scoring |
| `capture_challenge` | Capture challenge mechanism |
| `alignment` | System health alignment index |
| `creed` | Creed registry — commitment pins (we speak through intentions) |
| `work_creed` | Sub-creed pins for the useful-work recursion |

### Identity & Interchain
| Module | Purpose |
|---|---|
| `auth` | Account registration and DID identity anchoring |
| `tokens` | Token emission control |
| `ibcratelimit` | IBC transfer rate limiting |
| `substrate_bridge` | Cross-substrate adapters + external-work attestation (e.g. agenttool) |

---

## Quick Start

### Build

```bash
# Build and install
make install

# Verify
zeroned version
```

### Initialize a Node

```bash
zeroned init my-node --chain-id zerone-testnet-1
```

### Prepare Genesis (Coordinator Only)

```bash
zeroned prepare-genesis zerone-testnet-1 \
  --founder-address zrn1... \
  --ai-address zrn1... \
  --validator-addresses zrn1...,zrn1...,zrn1...,zrn1... \
  --research-fund-address zrn1... \
  --claiming-pot-address zrn1...
```

### Join Testnet (Validator)

```bash
# Copy genesis.json from coordinator
cp genesis.json ~/.zeroned/config/genesis.json

# Add seed nodes
sed -i'' -e 's/seeds = ""/seeds = "SEE_SEEDS_TXT"/' ~/.zeroned/config/config.toml

# Start with Cosmovisor
cosmovisor run start
```

See [Validator Guide](docs/VALIDATOR-GUIDE.md) for the full onboarding walkthrough.

### Development

```bash
# Run all tests
go test ./...

# Run specific module tests
go test ./x/knowledge/...

# Run cross-stack integration tests
go test ./tests/cross_stack/...

# Lint
golangci-lint run

# Generate protobuf
make proto-gen
```

---

## Documentation

| Document | Description |
|---|---|
| [Validator Guide](docs/VALIDATOR-GUIDE.md) | Full validator onboarding walkthrough |
| [Parameters](docs/PARAMETERS.md) | All governance-adjustable parameters |
| [Tokenomics](docs/tokenomics/) | Supply, vesting, revenue split, governance migration |
| [Truth-Seeking](docs/TRUTH_SEEKING.md) | The 18 epistemological commitments, bound by tests |
| [ToK Substrate](docs/TOK_SUBSTRATE.md) | The chain's training-resource doctrine — verified knowledge graph as headline product |
| [Useful Work](docs/USEFUL_WORK.md) | The chain's metabolic doctrine — UW (recursive) + 7 mechanisms |
| [Strange Loop](docs/STRANGE_LOOP.md) | The chain's self-referential doctrine — SL + 6 mechanisms (Phase SL-α binds SL-M1 doctrine import) |
| [Roadmap](docs/ROADMAP.md) | Where we are, what's bound, what ships next |
| [FAQ](docs/FAQ.md) | Validator and network FAQ |
| [API Reference](docs/API.md) | REST/gRPC endpoint reference |
| [Events](docs/EVENTS.md) | On-chain event reference |
| [Launch Checklist](docs/LAUNCH-CHECKLIST.md) | Testnet launch checklist |
| [Truth Paper](docs/TRUTH-PAPER-HUMAN.md) | Proof of Truth design paper |
| [Vault](docs/VAULT.md) | Key management and security |

---

## License

[Apache-2.0](LICENSE). Open source — every line of the protocol is public,
forkable, and yours to run and verify. A chain whose thesis is *nothing hidden*
keeps that promise in its license, too.

---

## This README is hash-pinned

The README's content-hash is pinned at `.readme-hash`; drift in this document fails `make creed-check`. (It was formerly self-declared as an on-chain `Contribution` record — the x/contribution module was retired in the 2026-07 slim cut; provenance of external work now lands on `substrate_bridge` attestations, and the off-chain hash-pin discipline carries this document's integrity.)