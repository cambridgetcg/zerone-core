# Substrate — Sub-creed of the Useful-Work Doctrine

> Lifecycle phase: `Substrate` — ZERONE-improving work: code, governance, ops, audits, doctrine, taxonomy amendments.
>
> Parent: [USEFUL_WORK.md](../USEFUL_WORK.md). Truth-floor: all 20 truth-seeking commitments apply.
>
> Status at inception (2026-05-10): three commitments.
>
> **Recursion note:** Substrate is the phase under which the chain self-modifies. Substrate Contributions are governed by the same regime as everything else (truth-floor + provenance), with one extra discipline — recusal — and an extra audit trail. The chain owes its self-improvers, but not retroactively after they're replaced.

---

## S1. Chain-modifying contributions name their `depends_on_marker` and revert path

A `MODULE_PROPOSAL` or `PIPELINE_IMPROVEMENT` Contribution must declare:
1. The chain-side marker(s) the modification depends on (e.g., `x/knowledge.panel.tally`, `ante.gas-validator`).
2. The revert path — the gov LIP class and operational steps that would undo the change if needed.

A change with no declared depends_on_marker is unrecoverable; a change with no declared revert path is irreversible by design.

**Why:** Substrate work compounds. Without depends_on_marker, the chain cannot tell which modifications power which behaviors. Without revert path, recursion-conferral is one-way and capture risk is uncapped. Commitment 10 (forward-only audit) requires the change be visible; the revert path is S1's own discipline on top of that visibility.

**Echoes:** truth-seeking 10, 12.

## S2. Contributors recuse on votes affecting their own contributions

A contributor named in `Contribution.contributors` must recuse from voting on a `CategoryRecursionConferral` LIP that references the contribution. Violations are slashable. Recusal is declared via attestation; the attestation is what the chain checks.

**Why:** Self-dealing on Substrate is the chain paying itself to pay itself. Recusal is the procedural answer to the structural risk. Commitment 9 (cartel detection has consequence) operationalized at the Substrate-LIP level.

**Echoes:** truth-seeking 9.

## S3. Reward-formula changes require simulation against historical contribution data

A Substrate Contribution that modifies any reward formula (admission stipend, lineage decay, recursion multiplier, royalty pool split) must include a simulation showing how the proposed formula would have allocated rewards on the chain's actual recent history. The simulation is part of the attestation; it is reproducible and challengeable.

**Why:** Reward formulas have second-order effects that are hard to predict from first principles. A simulation against history grounds the change in observable consequences. Without it, formula tweaks are persuasion; with it, they're a measurement.

**Echoes:** truth-seeking 1, 4, 14.
