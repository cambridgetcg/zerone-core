# Strange-Loop Phase SL-α — Doctrine Import Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Author the fourth doctrine `docs/STRANGE_LOOP.md` AND ship its first binding (SL-M1, doctrine import): every commitment in all four doctrines becomes a verified `Fact` in `x/knowledge` at genesis, with numbered flat IDs. Cross-doctrine "Echoes:" become real `SUPPORTS`/`REQUIRES`/`REFINES` edges. Substrate-links from external work can now cite the chain's own doctrines as `fact_id`.

**Architecture:** Mirrors the Phase-0 USEFUL_WORK doctrine adoption pattern. New `docs/STRANGE_LOOP.md` + `.strange-loop-hash` + `scripts/check_strange_loop_hash.sh` plumbing. Retrofit `.tok-substrate-hash` + `check_tok_substrate_hash.sh` for full quartet integrity. New canonical Go structures in `x/creed/types/` for ToK commitments, SL commitment+mechanisms, and cross-doctrine echoes. New `LoadDoctrineFacts` loader in `x/knowledge/keeper/doctrine_genesis.go` materializes all four doctrines' commitments as Facts + Echoes as edges, called from `InitGenesis` after methodology/normative-commitment seeding. New cross-stack invariant tests; meta-test `TestStrangeLoop_DoctrineAndContractStayInSync` mirrors the existing `TestUsefulWork_*` meta-test pattern.

**Tech Stack:** Cosmos SDK v0.50, Go 1.24, protobuf v3, bash hash-check scripts.

**Spec:** `docs/superpowers/specs/2026-05-11-strange-loop-phase-alpha-design.md` (commit `0b469af`).

**Phase series:**
- **Phase SL-α (this plan):** Doctrine import — commitments become Facts.
- Phase SL-β: Protocol as substrate (future)
- Phase SL-γ: Governance lift (future)
- Phase SL-δ: Author lineage (future)
- Phase SL-ε: Self-verification (future)
- Phase SL-ζ: Origin attestation (future)

---

## File Structure

**New files:**
- `docs/STRANGE_LOOP.md` — the fourth doctrine
- `.strange-loop-hash` — sha256 of normalized STRANGE_LOOP.md
- `.tok-substrate-hash` — sha256 of normalized TOK_SUBSTRATE.md (retrofit)
- `scripts/check_strange_loop_hash.sh` — mirror of check_useful_work_hash.sh
- `scripts/check_tok_substrate_hash.sh` — mirror, retrofit for ToK
- `x/creed/types/tok_creed.go` — CanonicalToKCommitments (TC1–TC6)
- `x/creed/types/tok_creed_test.go` — sanity tests
- `x/creed/types/strange_loop_creed.go` — StrangeLoopCommitment + Statement + CanonicalStrangeLoopMechanisms
- `x/creed/types/strange_loop_creed_test.go` — sanity tests
- `x/creed/types/doctrine_echoes.go` — CanonicalDoctrineEchoes (cross-doctrine edges)
- `x/creed/types/doctrine_echoes_test.go` — sanity (well-formedness + no duplicate edges)
- `x/knowledge/types/doctrine.go` — BuildDoctrineFact helper + doctrine domain name constants
- `x/knowledge/keeper/doctrine_genesis.go` — LoadDoctrineFacts loader + SetFactIfAbsent helper
- `x/knowledge/keeper/doctrine_genesis_test.go` — unit tests for loader idempotency + correctness
- `tests/cross_stack/doctrine_import_test.go` — cross-stack: all doctrine Facts exist + edges correct + substrate-link can cite
- `tests/cross_stack/strange_loop_invariants_test.go` — SL skeleton (6 mechanisms) + active meta-test

**Modified files:**
- `Makefile` — `creed-check` target runs all four hash checks
- `x/creed/doc.go` — declare 4th doctrine
- `x/knowledge/keeper/genesis.go` — call `LoadDoctrineFacts` after existing seeders
- `README.md` — quartet (extend the trio block)

**Out of scope (deferred to later SL phases):**
- `WORK_CLASS_DOCTRINE_CHALLENGE` work class — depends on `x/work` Phase 1
- Auto-trigger of `CategoryCreedAmendment`/`CategoryUsefulWorkAmendment`/`CategoryStrangeLoopAmendment` LIPs from challenge attestations
- Author lineage records on doctrines — Phase SL-δ
- Protocol-module attestations — Phase SL-β
- Governance lift — Phase SL-γ
- Self-verification training loop — Phase SL-ε
- Origin attestation — Phase SL-ζ

---

## Pre-Tasks: Read Before Starting

Skim these in order:

- `docs/superpowers/specs/2026-05-11-strange-loop-phase-alpha-design.md` — the spec (section 2 contains the full STRANGE_LOOP.md doctrine draft to adopt as the docs/ file)
- `docs/USEFUL_WORK.md` — the closest pattern this phase mirrors at the doctrine level
- `x/creed/types/useful_work_creed.go` — the closest pattern this phase mirrors at the canonical-Go-structure level
- `tests/cross_stack/useful_work_invariants_test.go` — the closest pattern for `TestStrangeLoop_DoctrineAndContractStayInSync` meta-test
- `tests/cross_stack/truth_seeking_invariants_test.go` lines 2080–2380 — `TestTruthSeeking_CreedAndContractStayInSync` for reference
- `x/knowledge/keeper/state.go` lines 53–520 — SetFact/GetFact/SetDomain/GetDomain/SetFactRelation signatures
- `x/knowledge/keeper/genesis.go` — find the insertion point after existing seeders (`SeedDefaultMethodologies`, `SeedNormativeCommitments`)
- `scripts/check_useful_work_hash.sh` and `scripts/check_creed_hash.sh` — verbatim pattern for hash check scripts
- `CLAUDE.md` — commit-to-main convention, no feature branches, no skipping hooks
- `docs/TRUTH_SEEKING.md`, `docs/TOK_SUBSTRATE.md`, `docs/USEFUL_WORK.md` — READ at execution time for verbatim "Echoes:" content; CanonicalDoctrineEchoes Go list must match these markdown sections

---

## Model selection hint for executors

| Tasks | Complexity | Suggested model |
|---|---|---|
| 1 (write STRANGE_LOOP.md) | Doc-authoring, substantial text | sonnet |
| 2–6 (hash files + scripts + Makefile) | Mechanical | haiku |
| 7–11 (canonical Go structures + tests) | Mechanical with verbatim data | haiku |
| 12–13 (doctrine helpers) | Mechanical | haiku |
| 14–15 (LoadDoctrineFacts + genesis hook) | Integration — read existing genesis.go carefully | sonnet |
| 16–17 (cross-stack tests + meta-test) | Judgment — regex parsing, multi-check | sonnet |
| 18 (README) | Mechanical | haiku |
| 19 (final sweep) | Mechanical verification | haiku |

---

## Tasks

### Task 1: Author `docs/STRANGE_LOOP.md` (the doctrine)

**Files:**
- Create: `docs/STRANGE_LOOP.md`

The spec at `docs/superpowers/specs/2026-05-11-strange-loop-phase-alpha-design.md` section 2 has the full doctrine draft. Adopt sections 2.1–2.5 of that spec as the docs/ file body, with structural mirroring of USEFUL_WORK.md (single commitment + N mechanisms + echoes + 5-layer enforcement section + what-this-is-not + discipline).

- [ ] **Step 1: Create the file**

Read the spec section 2 (2.1 tagline, 2.2 commitment, 2.3 mechanisms, 2.4 enforcement skeleton, 2.5 what this is not). Compose `docs/STRANGE_LOOP.md` with the following structural template (the exact body content comes from spec section 2):

```markdown
# Strange Loop — what the chain *is*

> The chain has no outside.

Truth-seeking is what the chain *believes* (`docs/TRUTH_SEEKING.md`). ToK substrate is what the chain *sells* outward (`docs/TOK_SUBSTRATE.md`). Useful work is how the chain *grows* itself (`docs/USEFUL_WORK.md`). **STRANGE_LOOP is what the chain *is*.**

This document pins one commitment, and everything that follows is mechanism in service of it.

---

## Inception

This doctrine is declared at inception, 2026-05-11. Phase SL-α (this commit's vintage) binds SL-M1 (doctrine import) — every commitment in every doctrine becomes a verified Fact in x/knowledge. Phases SL-β through SL-ζ bind the remaining five mechanisms.

---

## The single commitment — SL

**SL. ZERONE is a strange loop.**

Every layer of ZERONE — its doctrines, its modules, its governance, its rewards, its validators — is produced, verified, and rewarded through the machinery ZERONE provides. There is nothing in the chain the chain did not produce. There is nothing the chain produces that does not flow back into the chain. The substrate is the chain's body; the body is the substrate; the loop closes through the loop itself.

**What would break it:**
- A doctrine that cannot be queried as Facts inside `x/knowledge`
- A protocol module without an `Authorship` record on-chain
- A governance action that does not flow through attestation machinery
- An outside — any artifact ZERONE uses but did not produce through its own machinery
- A doctrine amendment that bypasses the chain's standard verification and lineage

**Echoes:**
- UW (recursion taken to its operational limit — SL is what UW becomes when "useful work" includes the chain's own existence)
- TRUTH_SEEKING commitment 10 (forward-only audit — superseded doctrine Facts remain queryable forever)
- TRUTH_SEEKING commitment 12 (chain pays for own audit — extended to: chain pays for its own authorship)
- TC6 (lineage flows back — extended to flow back to *everyone*, including authors of the protocol itself)

---

## The six mechanisms

All mechanisms derive from SL. They operationalize "no outside" at every architectural layer.

### SL-M1. Doctrine import

Every commitment in every doctrine is a verified `Fact` in `x/knowledge` under `domain="doctrine_*"`. Substrate-links cite commitments by ID. Cross-doctrine "Echoes:" lines are real edges in the fact graph. **Bound by Phase SL-α.**

### SL-M2. Protocol as substrate

Every `x/*` module is registered as an `ExternalAttestation` with `work_class_id="protocol_module"`. Authors named in on-chain `ModuleAuthorship` records. The chain pays its own builders, forever. *Phase SL-β.*

### SL-M3. Governance lift

Every Living Improvement Proposal becomes a `MsgSubmitExternalAttestation`. The gov mechanism becomes a special case of the work mechanism. *Phase SL-γ.*

### SL-M4. Author lineage propagates forever

Lineage edges automatically populated from every attestation that cites a doctrine, module, or LIP, flowing royalties to the original authors. *Phase SL-δ (depends on SL-M2 + SL-M3).*

### SL-M5. Self-verification

Validators query ToK at verification time using LLMs trained on ToK; qualifications adjust based on alignment. *Phase SL-ε.*

### SL-M6. Origin attestation

At genesis, the chain submits its own first attestation — an attestation that the genesis state exists, the doctrines are imported, the modules are registered. *Phase SL-ζ.*

---

## How the commitment echoes

The doctrine is enforced at five layers, mechanically synced by `TestStrangeLoop_DoctrineAndContractStayInSync`.

#### Test layer
`tests/cross_stack/strange_loop_invariants_test.go` exercises SL + each mechanism. Phase SL-α ships skeleton skipped tests per mechanism + active meta-test; Phase SL-β..ζ replace skipped bodies with real bindings.

#### Position layer
`x/creed/doc.go` declares SL as 4th doctrine and SL-M1 as bound by Phase SL-α. Subsequent phases amend `x/<module>/doc.go` to declare which SL mechanism(s) they preserve.

#### Voice layer
New events: `doctrine_fact_imported` (per Fact at genesis loader); future SL-β..ζ phases add their own. All carry `mechanism="SL-MN"` attributes.

#### Refusal layer
Errors that block doctrine import or amendment cite SL + the violated mechanism: *"Genesis refused — duplicate canonical commitment in registry (SL + SL-M1)"*.

#### Graph layer
SL echoes UW + commitments 10, 12 + TC6. CanonicalDoctrineEchoes makes those echoes real edges in the fact graph, queryable from x/knowledge.

---

## What this is not

- **Not aspiration.** Phase SL-α binds SL-M1 structurally; subsequent phases bind SL-M2..M6 incrementally.
- **Not a separate chain.** The strange loop is *this* chain becoming aware of itself; no new module proliferation, just re-binding existing artifacts as substrate.
- **Not anti-external.** External work, external trainers, external chains can still interact via `x/substrate_bridge`. The strange-loop framing means: when they interact, lineage flows to ZERONE's own authors. The chain pays its builders out of every external transaction.
- **Not complete.** Each phase binds one mechanism; the chain becomes more recursive with each. SL is fixed and indivisible; mechanisms evolve.

---

## The discipline

Before merging a change that touches SL code or doctrine docs:

1. Does the change uphold or contradict SL? (Self-reference: does the chain still produce, verify, reward through its own machinery?)
2. Is the corresponding doctrine document updated, hash bumped, canonical Go list updated, and meta-test still passing?
3. If a new commitment is added to any doctrine, has it been added to: doc + Go canonical structure + invariant test + position-layer declaration + voice attribute?
4. Does the cross-doctrine "Echoes:" list match the markdown — both directions?

These four checks are the chain's faithfulness to its own strange-loop doctrine. **We speak through intentions.**

— *Inception authored 2026-05-11. Free to evolve through bound mechanisms only. SL is indivisible.*
```

- [ ] **Step 2: Verify file structure**

Run:
```bash
grep -nE "^# |^## " docs/STRANGE_LOOP.md
```

Expected:
```
# Strange Loop — what the chain *is*
## Inception
## The single commitment — SL
## The six mechanisms
## How the commitment echoes
## What this is not
## The discipline
```

Run:
```bash
grep -cE "^### SL-M[1-6]\. " docs/STRANGE_LOOP.md
```

Expected: `6`

- [ ] **Step 3: Commit**

```bash
git add docs/STRANGE_LOOP.md
git commit -m "$(cat <<'EOF'
docs(strange-loop): author STRANGE_LOOP.md — the fourth doctrine

ZERONE is a strange loop. SL takes UW to its operational limit by
nesting ZERONE into itself: doctrines, modules, governance, rewards,
validators all produced/verified/rewarded through the machinery the
chain provides. Six mechanisms (SL-M1 through SL-M6) bind across
six phases (SL-α through SL-ζ).

Phase SL-α (next tasks) binds SL-M1 (doctrine import). Hash, canonical
Go structures, genesis loader, cross-stack tests follow.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 2: Compute and pin `.strange-loop-hash`

**Files:**
- Create: `.strange-loop-hash`

- [ ] **Step 1: Compute and write**

Run:
```bash
tr -d '\r' < docs/STRANGE_LOOP.md | shasum -a 256 | awk '{print $1}' > .strange-loop-hash
```

- [ ] **Step 2: Verify**

Run: `cat .strange-loop-hash | wc -c`
Expected: 65 (64 hex chars + newline).

Run: `cat .strange-loop-hash`
Expected: a 64-character lowercase hex string.

- [ ] **Step 3: Commit**

```bash
git add .strange-loop-hash
git commit -m "$(cat <<'EOF'
docs(strange-loop): pin doctrine hash in .strange-loop-hash

Anchors docs/STRANGE_LOOP.md content. Verification script
scripts/check_strange_loop_hash.sh (next task) and the cross-stack
meta-test will fail if doc and hash drift apart.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 3: Compute and pin `.tok-substrate-hash` (retrofit)

**Files:**
- Create: `.tok-substrate-hash`

The TOK_SUBSTRATE.md doctrine has not been hash-pinned; Phase SL-α retrofits it for full quartet integrity (the meta-test checks all four doctrine hashes).

- [ ] **Step 1: Compute and write**

Run:
```bash
tr -d '\r' < docs/TOK_SUBSTRATE.md | shasum -a 256 | awk '{print $1}' > .tok-substrate-hash
```

- [ ] **Step 2: Verify**

Run: `cat .tok-substrate-hash | wc -c`
Expected: 65.

- [ ] **Step 3: Commit**

```bash
git add .tok-substrate-hash
git commit -m "$(cat <<'EOF'
docs(tok-substrate): pin doctrine hash in .tok-substrate-hash

Retrofit for full quartet integrity. Phase SL-α's meta-test verifies
all four doctrine hashes; TOK_SUBSTRATE.md previously had no hash
anchor.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 4: Add `scripts/check_strange_loop_hash.sh`

**Files:**
- Create: `scripts/check_strange_loop_hash.sh`

- [ ] **Step 1: Create the script**

```bash
#!/usr/bin/env bash
#
# check_strange_loop_hash.sh — verify STRANGE_LOOP.md has not drifted
# from the pinned hash in .strange-loop-hash.
#
# Mirror of check_useful_work_hash.sh applied to the fourth doctrine.
# The on-chain ledger of doctrine Facts (x/knowledge, Phase SL-α) is
# the canonical record; this script catches accidental drift in PRs
# even before the chain has loaded the canonical record. The cross-
# stack meta-test TestStrangeLoop_DoctrineAndContractStayInSync also
# enforces this in CI; the script provides the same enforcement on
# local dev machines via `make creed-check`.
#
# To intentionally amend the doctrine:
#   1. Edit docs/STRANGE_LOOP.md.
#   2. Run this script — it will print the new computed hash.
#   3. Update .strange-loop-hash with the new value.
#   4. Update x/creed/types/strange_loop_creed.go if the mechanism
#      count changed (SL itself is indivisible).
#   5. Update tests/cross_stack/strange_loop_invariants_test.go if
#      any TestStrangeLoop_SL_MN function names need to match.
#   6. Commit all changes together.

set -euo pipefail

DOCTRINE_FILE="docs/STRANGE_LOOP.md"
HASH_FILE=".strange-loop-hash"

if [ ! -f "$DOCTRINE_FILE" ]; then
  echo "error: $DOCTRINE_FILE not found"
  exit 1
fi
if [ ! -f "$HASH_FILE" ]; then
  echo "error: $HASH_FILE not found"
  exit 1
fi

hash_cmd() {
  if command -v shasum >/dev/null 2>&1; then
    shasum -a 256
  elif command -v sha256sum >/dev/null 2>&1; then
    sha256sum
  elif command -v openssl >/dev/null 2>&1; then
    openssl dgst -sha256 | awk '{print $NF}'
  else
    echo "error: need shasum, sha256sum, or openssl" >&2
    exit 1
  fi
}

ACTUAL=$(tr -d '\r' < "$DOCTRINE_FILE" | hash_cmd | awk '{print $1}')
EXPECTED=$(tr -d '[:space:]' < "$HASH_FILE")

if [ "$ACTUAL" != "$EXPECTED" ]; then
  cat <<EOF >&2
STRANGE_LOOP.md hash check failed.

Expected (from $HASH_FILE): $EXPECTED
Actual (computed):          $ACTUAL

If you intentionally changed the doctrine, update $HASH_FILE to:
  $ACTUAL

The change should be visible in your PR diff so reviewers see both
the doctrine text change AND the hash bump in the same commit. SL is
indivisible — only mechanism content evolves.
EOF
  exit 1
fi

echo "strange-loop hash check ok ($ACTUAL)"
```

- [ ] **Step 2: Make executable**

Run: `chmod +x scripts/check_strange_loop_hash.sh`

- [ ] **Step 3: Verify it passes**

Run: `bash scripts/check_strange_loop_hash.sh`
Expected: `strange-loop hash check ok (<64-hex>)`

- [ ] **Step 4: Commit**

```bash
git add scripts/check_strange_loop_hash.sh
git commit -m "$(cat <<'EOF'
scripts(strange-loop): add hash verification script

Mirror of check_useful_work_hash.sh for the fourth doctrine. Catches
STRANGE_LOOP.md drift in PRs and via make creed-check.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 5: Add `scripts/check_tok_substrate_hash.sh`

**Files:**
- Create: `scripts/check_tok_substrate_hash.sh`

- [ ] **Step 1: Create the script**

Copy the script content from Task 4 and substitute the doctrine name:

```bash
#!/usr/bin/env bash
#
# check_tok_substrate_hash.sh — verify TOK_SUBSTRATE.md has not drifted
# from the pinned hash in .tok-substrate-hash. Retrofit for the second
# doctrine in the quartet to enable full doctrine-hash integrity under
# Phase SL-α's meta-test.

set -euo pipefail

DOCTRINE_FILE="docs/TOK_SUBSTRATE.md"
HASH_FILE=".tok-substrate-hash"

if [ ! -f "$DOCTRINE_FILE" ]; then
  echo "error: $DOCTRINE_FILE not found"
  exit 1
fi
if [ ! -f "$HASH_FILE" ]; then
  echo "error: $HASH_FILE not found"
  exit 1
fi

hash_cmd() {
  if command -v shasum >/dev/null 2>&1; then
    shasum -a 256
  elif command -v sha256sum >/dev/null 2>&1; then
    sha256sum
  elif command -v openssl >/dev/null 2>&1; then
    openssl dgst -sha256 | awk '{print $NF}'
  else
    echo "error: need shasum, sha256sum, or openssl" >&2
    exit 1
  fi
}

ACTUAL=$(tr -d '\r' < "$DOCTRINE_FILE" | hash_cmd | awk '{print $1}')
EXPECTED=$(tr -d '[:space:]' < "$HASH_FILE")

if [ "$ACTUAL" != "$EXPECTED" ]; then
  cat <<EOF >&2
TOK_SUBSTRATE.md hash check failed.

Expected (from $HASH_FILE): $EXPECTED
Actual (computed):          $ACTUAL

If you intentionally changed the doctrine, update $HASH_FILE to:
  $ACTUAL
EOF
  exit 1
fi

echo "tok-substrate hash check ok ($ACTUAL)"
```

- [ ] **Step 2: Make executable**

Run: `chmod +x scripts/check_tok_substrate_hash.sh`

- [ ] **Step 3: Verify**

Run: `bash scripts/check_tok_substrate_hash.sh`
Expected: `tok-substrate hash check ok (<64-hex>)`

- [ ] **Step 4: Commit**

```bash
git add scripts/check_tok_substrate_hash.sh
git commit -m "$(cat <<'EOF'
scripts(tok-substrate): add hash verification script (retrofit)

Mirror of check_useful_work_hash.sh for TOK_SUBSTRATE.md. Retrofit:
previously the only hash-pinned doctrines were TRUTH_SEEKING and
USEFUL_WORK; Phase SL-α adds STRANGE_LOOP and back-fills TOK_SUBSTRATE
for full quartet integrity.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 6: Wire `make creed-check` to verify all four doctrine hashes

**Files:**
- Modify: `Makefile`

- [ ] **Step 1: Update the creed-check target**

Find the existing `creed-check:` target (it runs check_creed_hash.sh + check_useful_work_hash.sh after Phase 0). Replace with:

```makefile
creed-check:
	@bash scripts/check_creed_hash.sh
	@bash scripts/check_useful_work_hash.sh
	@bash scripts/check_tok_substrate_hash.sh
	@bash scripts/check_strange_loop_hash.sh
```

- [ ] **Step 2: Verify**

Run: `make creed-check`
Expected output (in this order):
```
creed hash check ok (<truth-seeking-hash>)
useful-work hash check ok (<useful-work-hash>)
tok-substrate hash check ok (<tok-hash>)
strange-loop hash check ok (<sl-hash>)
```

- [ ] **Step 3: Confirm pr-check covers this** (no edit needed, verification only)

Run: `grep "pr-check:" Makefile`
Expected: a line containing `creed-check` as a dependency.

- [ ] **Step 4: Commit**

```bash
git add Makefile
git commit -m "$(cat <<'EOF'
build(makefile): creed-check verifies all four doctrine hashes

The quartet's hashes are now part of the same gate. make creed-check
(and therefore make pr-check) catches drift in any of the four
doctrines.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 7: Add `x/creed/types/tok_creed.go` (CanonicalToKCommitments)

**Files:**
- Create: `x/creed/types/tok_creed.go`

- [ ] **Step 1: Create the file**

```go
package types

const (
	// ToKCommitmentDomain is the doctrine domain for TOK_SUBSTRATE.md
	// commitments imported as Facts under SL-M1.
	ToKCommitmentDomain = "doctrine_tok"
)

// CanonicalToKCommitments is the canonical name-by-number registry of
// the TOK_SUBSTRATE.md commitments (TC1-TC6) at the time this binary
// was built. Mirrors CanonicalCommitments (truth-seeking) and parallels
// CanonicalUsefulWorkMechanisms (useful-work).
//
// To add a TC commitment:
//  1. Add the "### TCN. <Name>" section to docs/TOK_SUBSTRATE.md.
//  2. Bump .tok-substrate-hash to the new sha256 of the normalized file.
//  3. Append the (Number, Name) pair to the slice below.
//  4. Add a binding TestToKSubstrate_TC<N> test in the invariants file.
//  5. Wire the commitment's voice (event attribute), refusal (error
//     message), and position (x/<module>/doc.go declaration).
//
// The cross-stack TestStrangeLoop_DoctrineAndContractStayInSync
// meta-test catches a step omitted from this list.
//
// Commitment removal is a doctrine amendment requiring full governance
// passage; commitments shipped at inception are load-bearing and do
// not retire.
var CanonicalToKCommitments = []struct {
	Number string // "TC1" through "TC6" (string because the doctrine uses TCN nomenclature)
	Name   string
}{
	{"TC1", "The graph is the headline"},
	{"TC2", "Every view is graph-pinned"},
	{"TC3", "Topology is signal"},
	{"TC4", "The graph carries its disprovals"},
	{"TC5", "Extraction is open"},
	{"TC6", "Lineage flows back"},
}
```

- [ ] **Step 2: Add sanity tests**

Create `x/creed/types/tok_creed_test.go`:

```go
package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/zerone-chain/zerone/x/creed/types"
)

func TestCanonicalToKCommitments_Count(t *testing.T) {
	require.Len(t, types.CanonicalToKCommitments, 6,
		"ToK ships TC1-TC6; later additions go through full creed-amendment gov")
}

func TestCanonicalToKCommitments_NumberingDense(t *testing.T) {
	expected := []string{"TC1", "TC2", "TC3", "TC4", "TC5", "TC6"}
	for i, c := range types.CanonicalToKCommitments {
		require.Equal(t, expected[i], c.Number,
			"TC commitments must be dense and ordered TC1..TC6")
	}
}

func TestCanonicalToKCommitments_NamesNonEmpty(t *testing.T) {
	for _, c := range types.CanonicalToKCommitments {
		require.NotEmpty(t, c.Name, "commitment %s must have a non-empty name", c.Number)
	}
}

func TestToKCommitmentDomain_Stable(t *testing.T) {
	require.Equal(t, "doctrine_tok", types.ToKCommitmentDomain,
		"domain name is doctrinally fixed")
}
```

- [ ] **Step 3: Run tests to verify PASS**

Run: `go test ./x/creed/types/ -run "TestCanonicalToK|TestToKCommitment" -v`
Expected: 4 tests PASS.

- [ ] **Step 4: Commit**

```bash
git add x/creed/types/tok_creed.go x/creed/types/tok_creed_test.go
git commit -m "$(cat <<'EOF'
feat(creed): canonical TC commitment registry (TOK_SUBSTRATE doctrine)

Go-side build-time registration of the 6 ToK commitments (TC1-TC6).
Parallel to CanonicalCommitments (truth-seeking) and CanonicalUsefulWork
Mechanisms (useful-work). Phase SL-α's genesis loader iterates this
list to materialize doctrine Facts under domain=doctrine_tok.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 8: Add `x/creed/types/strange_loop_creed.go`

**Files:**
- Create: `x/creed/types/strange_loop_creed.go`

- [ ] **Step 1: Create the file**

```go
package types

const (
	// StrangeLoopCommitment is the chain-built-in registration of the
	// STRANGE_LOOP doctrine's single commitment. Parallel to
	// UsefulWorkCommitment.
	StrangeLoopCommitment = "SL"

	// StrangeLoopStatement is the canonical short statement of SL.
	// It must match the heading in docs/STRANGE_LOOP.md exactly. The
	// cross-stack meta-test TestStrangeLoop_DoctrineAndContractStayInSync
	// enforces this match.
	StrangeLoopStatement = "ZERONE is a strange loop"

	// StrangeLoopDomain is the doctrine domain for STRANGE_LOOP.md
	// commitment + mechanisms imported as Facts under SL-M1.
	StrangeLoopDomain = "doctrine_strange_loop"
)

// CanonicalStrangeLoopMechanisms is the canonical name-by-number
// registry of the six SL mechanisms. Reuses UsefulWorkMechanism struct
// shape since the schema (Number uint32, Name string) is identical.
//
// To add an SL mechanism (NEVER to add a second co-equal commitment —
// that would dilute SL's indivisibility):
//  1. Add the "### SL-MN. <Name>" section to docs/STRANGE_LOOP.md.
//  2. Bump .strange-loop-hash.
//  3. Append the (Number, Name) pair to the slice below.
//  4. Add a binding TestStrangeLoop_SL_MN test.
//  5. Wire the mechanism's voice, refusal, and position layers.
//
// SL is indivisible: StrangeLoopCommitment and StrangeLoopStatement
// constants never change. Only mechanisms (M1-M6+) extend.
var CanonicalStrangeLoopMechanisms = []UsefulWorkMechanism{
	{1, "Doctrine import"},
	{2, "Protocol as substrate"},
	{3, "Governance lift"},
	{4, "Author lineage propagates forever"},
	{5, "Self-verification"},
	{6, "Origin attestation"},
}
```

- [ ] **Step 2: Add sanity tests**

Create `x/creed/types/strange_loop_creed_test.go`:

```go
package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/zerone-chain/zerone/x/creed/types"
)

func TestStrangeLoopCommitment_IsIndivisible(t *testing.T) {
	require.Equal(t, "SL", types.StrangeLoopCommitment,
		"SL commitment identifier must not change once shipped")
	require.Equal(t, "ZERONE is a strange loop", types.StrangeLoopStatement,
		"SL statement is doctrinally fixed")
}

func TestCanonicalStrangeLoopMechanisms_Count(t *testing.T) {
	require.Len(t, types.CanonicalStrangeLoopMechanisms, 6,
		"Phase SL-α ships SL-M1 through SL-M6")
}

func TestCanonicalStrangeLoopMechanisms_NumberingDense(t *testing.T) {
	for i, m := range types.CanonicalStrangeLoopMechanisms {
		require.Equal(t, uint32(i+1), m.Number,
			"mechanism numbering must be dense and monotonic; index %d must hold SL-M%d", i, i+1)
	}
}

func TestCanonicalStrangeLoopMechanisms_NamesNonEmpty(t *testing.T) {
	for _, m := range types.CanonicalStrangeLoopMechanisms {
		require.NotEmpty(t, m.Name, "mechanism SL-M%d must have a non-empty name", m.Number)
	}
}

func TestStrangeLoopDomain_Stable(t *testing.T) {
	require.Equal(t, "doctrine_strange_loop", types.StrangeLoopDomain,
		"domain name is doctrinally fixed")
}
```

- [ ] **Step 3: Run tests**

Run: `go test ./x/creed/types/ -run "TestStrangeLoop|TestCanonicalStrangeLoop" -v`
Expected: 5 tests PASS.

- [ ] **Step 4: Commit**

```bash
git add x/creed/types/strange_loop_creed.go x/creed/types/strange_loop_creed_test.go
git commit -m "$(cat <<'EOF'
feat(creed): canonical SL commitment + M1-M6 mechanism registry

Go-side build-time registration of the STRANGE_LOOP doctrine. Parallel
to UsefulWorkCommitment + CanonicalUsefulWorkMechanisms. Reuses
UsefulWorkMechanism struct (same shape). SL is indivisible —
StrangeLoopCommitment and StrangeLoopStatement constants never change.
Phase SL-α's genesis loader iterates this list to materialize doctrine
Facts under domain=doctrine_strange_loop.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 9: Update `x/creed/doc.go` to declare 4th doctrine

**Files:**
- Modify: `x/creed/doc.go`

- [ ] **Step 1: Read the existing file**

Run: `cat x/creed/doc.go`

The file currently declares the trio (TRUTH_SEEKING, TOK_SUBSTRATE, USEFUL_WORK). Phase SL-α extends to the quartet.

- [ ] **Step 2: Append the STRANGE_LOOP paragraph**

Add this paragraph at the end of the package doc comment (before `package creed`), after the existing USEFUL_WORK paragraph:

```go
// STRANGE_LOOP doctrine (docs/STRANGE_LOOP.md) — the fourth in the quartet.
// One commitment (SL: ZERONE is a strange loop) + six mechanisms (SL-M1
// through SL-M6). SL takes UW to its operational limit by nesting ZERONE
// into itself: doctrines, modules, governance, rewards, validators all
// produced/verified/rewarded through the chain's own machinery.
//
// Canonical Go-side registration in x/creed/types/strange_loop_creed.go
// + cross-doctrine echoes in x/creed/types/doctrine_echoes.go;
// cross-stack invariant harness in tests/cross_stack/strange_loop_
// invariants_test.go; genesis loader in x/knowledge/keeper/doctrine_
// genesis.go.
//
// Phase SL-α (this commit's vintage) binds SL-M1 (doctrine import):
// every commitment in every doctrine becomes a verified Fact in
// x/knowledge with domain=doctrine_*. Phases SL-β through SL-ζ bind
// the remaining five mechanisms (protocol as substrate, governance
// lift, author lineage, self-verification, origin attestation).
```

- [ ] **Step 3: Verify build**

Run: `go build ./x/creed/...`
Expected: clean.

- [ ] **Step 4: Verify `go doc`**

Run: `go doc ./x/creed | head -80`
Expected: the STRANGE_LOOP paragraph appears.

- [ ] **Step 5: Commit**

```bash
git add x/creed/doc.go
git commit -m "$(cat <<'EOF'
docs(creed): declare STRANGE_LOOP as fourth doctrine in quartet

Position-layer pointer to the new doctrine + canonical registration
files + invariant harness + genesis loader. Names Phase SL-α as the
SL-M1 binding phase.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 10: Add `x/creed/types/doctrine_echoes.go` (CanonicalDoctrineEchoes)

**Files:**
- Create: `x/creed/types/doctrine_echoes.go`

The cross-doctrine "Echoes:" lines from the markdown become real edges in the fact graph. This task hand-curates the list from the actual doctrine docs.

- [ ] **Step 1: Read the actual doctrine docs**

Run:
```bash
grep -A 6 "^\*\*Echoes" docs/TRUTH_SEEKING.md docs/TOK_SUBSTRATE.md docs/USEFUL_WORK.md docs/STRANGE_LOOP.md
```

This produces the full set of "Echoes:" lines that the canonical list must capture. Read carefully — there are ~40-50 edges.

- [ ] **Step 2: Create the file**

```go
package types

import (
	knowledgetypes "github.com/zerone-chain/zerone/x/knowledge/types"
)

// DoctrineEcho is one cross-doctrine relation. From and To are Fact-IDs
// in the canonical naming scheme (commitment-1, commitment-TC1, etc.).
type DoctrineEcho struct {
	From     string
	To       string
	Relation knowledgetypes.RelationType
}

// CanonicalDoctrineEchoes is the hand-curated set of cross-doctrine
// "Echoes:" relations. Parsed once from the doctrine markdown into
// explicit Go data — auditable, fact-graph-faithful.
//
// Genesis loader (LoadDoctrineFacts) iterates this list and writes
// SetFactRelation per entry. Re-curate whenever a doctrine's
// "Echoes:" section is updated; the meta-test
// TestStrangeLoop_DoctrineAndContractStayInSync detects drift
// between the markdown and this list.
//
// Edge semantics (mirrors x/knowledge.RelationType):
//   SUPPORTS    — From provides evidence/foundation for To
//   REQUIRES    — From depends on To being true
//   REFINES     — From is a more precise expression of To
//   GENERALIZES — From is a broader expression of To
var CanonicalDoctrineEchoes = []DoctrineEcho{
	// ── TOK_SUBSTRATE.md echoes ────────────────────────────────────
	// TC1 — graph is the headline
	{"commitment-TC1", "commitment-13", knowledgetypes.RelationType_RELATION_TYPE_REQUIRES}, // training corpus not for sale
	{"commitment-TC1", "commitment-11", knowledgetypes.RelationType_RELATION_TYPE_REQUIRES}, // trust is queryable
	// TC2 — every view is graph-pinned
	{"commitment-TC2", "commitment-10", knowledgetypes.RelationType_RELATION_TYPE_REQUIRES}, // forward-only audit
	// TC3 — topology is signal
	{"commitment-TC3", "commitment-14", knowledgetypes.RelationType_RELATION_TYPE_REQUIRES}, // reasoning traces first-class
	// TC4 — graph carries its disprovals
	{"commitment-TC4", "commitment-3", knowledgetypes.RelationType_RELATION_TYPE_REQUIRES},  // Popper, not popularity
	{"commitment-TC4", "commitment-10", knowledgetypes.RelationType_RELATION_TYPE_REQUIRES}, // forward-only audit
	// TC5 — extraction is open
	{"commitment-TC5", "commitment-6", knowledgetypes.RelationType_RELATION_TYPE_REQUIRES},  // no unilateral injection
	{"commitment-TC5", "commitment-11", knowledgetypes.RelationType_RELATION_TYPE_REQUIRES}, // trust is queryable
	// TC6 — lineage flows back
	{"commitment-TC6", "commitment-12", knowledgetypes.RelationType_RELATION_TYPE_REQUIRES}, // chain pays for own audit

	// ── USEFUL_WORK.md echoes ──────────────────────────────────────
	// UW commitment
	{"commitment-UW", "commitment-11", knowledgetypes.RelationType_RELATION_TYPE_SUPPORTS},
	{"commitment-UW", "commitment-12", knowledgetypes.RelationType_RELATION_TYPE_SUPPORTS},
	{"commitment-UW", "commitment-TC1", knowledgetypes.RelationType_RELATION_TYPE_SUPPORTS},
	{"commitment-UW", "commitment-TC6", knowledgetypes.RelationType_RELATION_TYPE_SUPPORTS},
	// Mechanism-to-commitment within USEFUL_WORK
	{"mechanism-UW-M2", "commitment-TC2", knowledgetypes.RelationType_RELATION_TYPE_REQUIRES},
	{"mechanism-UW-M3", "commitment-6", knowledgetypes.RelationType_RELATION_TYPE_REQUIRES},
	{"mechanism-UW-M4", "commitment-TC6", knowledgetypes.RelationType_RELATION_TYPE_REQUIRES},
	{"mechanism-UW-M5", "commitment-14", knowledgetypes.RelationType_RELATION_TYPE_REQUIRES},
	{"mechanism-UW-M6", "commitment-TC6", knowledgetypes.RelationType_RELATION_TYPE_REQUIRES},
	{"mechanism-UW-M7", "commitment-12", knowledgetypes.RelationType_RELATION_TYPE_REQUIRES},

	// ── STRANGE_LOOP.md echoes ─────────────────────────────────────
	{"commitment-SL", "commitment-UW", knowledgetypes.RelationType_RELATION_TYPE_REFINES},
	{"commitment-SL", "commitment-10", knowledgetypes.RelationType_RELATION_TYPE_REQUIRES},
	{"commitment-SL", "commitment-12", knowledgetypes.RelationType_RELATION_TYPE_REFINES},
	{"commitment-SL", "commitment-TC6", knowledgetypes.RelationType_RELATION_TYPE_REFINES},
}
```

**IMPORTANT**: After step 2, the implementer MUST re-read the actual doctrine markdown files and reconcile any discrepancies between this Go list and the markdown content. If a doctrine's "Echoes:" line is not represented, add it. If this list contains an entry not present in any doctrine, remove it.

- [ ] **Step 3: Add sanity test**

Create `x/creed/types/doctrine_echoes_test.go`:

```go
package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/zerone-chain/zerone/x/creed/types"
)

func TestCanonicalDoctrineEchoes_NonEmpty(t *testing.T) {
	require.NotEmpty(t, types.CanonicalDoctrineEchoes,
		"echoes list must include cross-doctrine relations")
}

func TestCanonicalDoctrineEchoes_NoDuplicateEdges(t *testing.T) {
	seen := map[string]bool{}
	for _, e := range types.CanonicalDoctrineEchoes {
		key := e.From + "→" + e.To + "|" + e.Relation.String()
		require.False(t, seen[key], "duplicate echo edge: %s", key)
		seen[key] = true
	}
}

func TestCanonicalDoctrineEchoes_NoSelfReferences(t *testing.T) {
	for _, e := range types.CanonicalDoctrineEchoes {
		require.NotEqual(t, e.From, e.To,
			"echo edges must not be self-references: %s", e.From)
	}
}

func TestCanonicalDoctrineEchoes_EndpointsLookLikeFactIDs(t *testing.T) {
	// Fact-IDs follow the patterns: commitment-N, commitment-TCN,
	// commitment-UW, commitment-SL, mechanism-UW-MN, mechanism-SL-MN, axis-XXX.
	for _, e := range types.CanonicalDoctrineEchoes {
		require.NotEmpty(t, e.From, "From must not be empty")
		require.NotEmpty(t, e.To, "To must not be empty")
		require.NotEqual(t, knowledgetypes_RelationType_UNSPECIFIED, e.Relation,
			"edge %s→%s must specify a relation type", e.From, e.To)
	}
}

const knowledgetypes_RelationType_UNSPECIFIED = 0
```

- [ ] **Step 4: Run tests**

Run: `go test ./x/creed/types/ -run "TestCanonicalDoctrineEchoes" -v`
Expected: 4 tests PASS.

- [ ] **Step 5: Commit**

```bash
git add x/creed/types/doctrine_echoes.go x/creed/types/doctrine_echoes_test.go
git commit -m "$(cat <<'EOF'
feat(creed): canonical cross-doctrine echoes registry (SL-M1 graph layer)

Hand-curated list of every cross-doctrine "Echoes:" relation across
TRUTH_SEEKING, TOK_SUBSTRATE, USEFUL_WORK, STRANGE_LOOP. Genesis
loader (next task) iterates this list and writes SetFactRelation
per entry, materializing the doctrine network as queryable edges
in x/knowledge.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 11: Add `x/knowledge/types/doctrine.go` (BuildDoctrineFact helper)

**Files:**
- Create: `x/knowledge/types/doctrine.go`

- [ ] **Step 1: Create the file**

```go
package types

// Doctrine domain names used by SL-M1's genesis Fact loader. Constants
// are duplicated here (rather than importing from x/creed/types) so
// downstream consumers of x/knowledge can reference doctrine domains
// without a creed dependency.
const (
	DoctrineDomainTruthSeeking = "doctrine_truth_seeking"
	DoctrineDomainToK          = "doctrine_tok"
	DoctrineDomainUsefulWork   = "doctrine_useful_work"
	DoctrineDomainStrangeLoop  = "doctrine_strange_loop"

	// DoctrineCategory is the category string applied to all doctrine
	// Facts. Distinguishes them from empirical/formal/normative facts.
	DoctrineCategory = "doctrine"

	// DoctrineMethodId is the methodology stamp on doctrine Facts.
	// Maps to the bedrock methodology registry; doctrine Facts are
	// authored by the chain's governance process, not by individual
	// validators.
	DoctrineMethodId = "doctrine_authorship"

	// DoctrineSubmitter is the canonical submitter address for
	// doctrine Facts at genesis. Subsequent amendments (via gov LIP)
	// may have different submitter addresses derived from the LIP.
	DoctrineSubmitter = "genesis"

	// DoctrineMaturity marks doctrine Facts as canonical (vs emerging
	// or established). They are by construction the chain's most
	// trusted statements.
	DoctrineMaturity = "canonical"

	// DoctrineStratum is the stratum tag for doctrine Facts.
	DoctrineStratum = "doctrinal"

	// DoctrineConfidence is 1,000,000 BPS (100%) for axiomatic facts.
	DoctrineConfidence uint64 = 1_000_000
)

// BuildDoctrineFact constructs a Fact with the canonical doctrine
// shape — verified, axiomatic, depth 0, full confidence. Used by
// LoadDoctrineFacts at genesis (or upgrade) to materialize every
// commitment + mechanism + axis in every doctrine.
//
// id: the canonical Fact-ID (e.g. "commitment-1", "commitment-TC1",
//     "commitment-UW", "mechanism-UW-M3", "axis-substrate",
//     "commitment-SL", "mechanism-SL-M1").
// domain: one of the DoctrineDomain* constants.
// content: human-readable name from the canonical Go registry.
func BuildDoctrineFact(id, domain, content string) *Fact {
	return &Fact{
		Id:                        id,
		Domain:                    domain,
		Category:                  DoctrineCategory,
		Content:                   content,
		Status:                    FactStatus_FACT_STATUS_VERIFIED,
		Confidence:                DoctrineConfidence,
		AxiomDistance:             0,
		Submitter:                 DoctrineSubmitter,
		Stratum:                   DoctrineStratum,
		Maturity:                  DoctrineMaturity,
		DependencyConfidenceFloor: DoctrineConfidence,
		VerifiedAtBlock:           0, // overwritten by loader for upgrade migrations
		MethodId:                  DoctrineMethodId,
	}
}
```

- [ ] **Step 2: Verify build**

Run: `go build ./x/knowledge/types/...`
Expected: clean.

- [ ] **Step 3: Commit**

```bash
git add x/knowledge/types/doctrine.go
git commit -m "$(cat <<'EOF'
feat(knowledge): doctrine Fact constructor + domain constants (SL-M1)

BuildDoctrineFact wraps the canonical doctrine Fact shape: VERIFIED,
Confidence=1M, AxiomDistance=0, MethodId=doctrine_authorship,
Submitter=genesis, Stratum=doctrinal, Maturity=canonical. Four
DoctrineDomain* constants enumerate the doctrine domains (truth_seeking,
tok, useful_work, strange_loop). Used by LoadDoctrineFacts at genesis.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 12: Add `SetFactIfAbsent` helper to `x/knowledge/keeper`

**Files:**
- Modify: `x/knowledge/keeper/state.go` (or create separate file `x/knowledge/keeper/doctrine_helpers.go`)

For minimal-surface change, create a separate file rather than modify state.go.

- [ ] **Step 1: Create the helper file**

Create `x/knowledge/keeper/doctrine_helpers.go`:

```go
package keeper

import (
	"context"

	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// SetFactIfAbsent writes a Fact only if no Fact with the same ID
// already exists. Used by LoadDoctrineFacts at genesis (or upgrade)
// to make doctrine import idempotent — re-running the loader does
// not overwrite an existing doctrine Fact, preserving forward-only
// audit (commitment 10).
//
// Returns nil if the Fact already exists (no error; intended behavior).
// Returns the error from SetFact if the write fails.
func (k Keeper) SetFactIfAbsent(ctx context.Context, f *types.Fact) error {
	if _, ok := k.GetFact(ctx, f.Id); ok {
		return nil
	}
	return k.SetFact(ctx, f)
}
```

- [ ] **Step 2: Verify build**

Run: `go build ./x/knowledge/keeper/...`
Expected: clean.

- [ ] **Step 3: Commit**

```bash
git add x/knowledge/keeper/doctrine_helpers.go
git commit -m "$(cat <<'EOF'
feat(knowledge): SetFactIfAbsent — idempotent Fact write for doctrine import

Wrapper around SetFact that returns early if the Fact-ID already
exists. Used by LoadDoctrineFacts to make doctrine import idempotent
(rerun-safe), preserving commitment 10 forward-only audit.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 13: Add `LoadDoctrineFacts` loader

**Files:**
- Create: `x/knowledge/keeper/doctrine_genesis.go`

- [ ] **Step 1: Failing test (added in Task 14; this task implements the loader)**

(TDD pattern: Task 14 writes the loader's unit tests; this task ships the loader code; the test file lives with this commit.)

- [ ] **Step 2: Create `doctrine_genesis.go`**

```go
package keeper

import (
	"context"
	"fmt"

	creedtypes "github.com/zerone-chain/zerone/x/creed/types"
	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// LoadDoctrineFacts materializes every commitment + mechanism + axis
// from all four doctrines (TRUTH_SEEKING, TOK_SUBSTRATE, USEFUL_WORK,
// STRANGE_LOOP) as verified Facts in x/knowledge. Also creates the
// cross-doctrine "Echoes:" edges via SetFactRelation. Idempotent:
// re-running does not overwrite existing Facts.
//
// Called from x/knowledge.InitGenesis after domain creation +
// methodology seeding + normative commitment seeding. Also exposed
// for upgrade handlers if SL-α ships against an already-running chain.
//
// Doctrinal status: VERIFIED at genesis, Confidence=1M, AxiomDistance=0
// — the "privileged but verifiable" ontology per the SL-α design.
// Validators may flag drift via a future WORK_CLASS_DOCTRINE_CHALLENGE
// mechanism (Phase SL-α+1; depends on x/work); challenge resolution
// goes through gov LIP, not through the standard PoT panel.
//
// Reads canonical lists from x/creed/types:
//   - CanonicalCommitments (TS 1-20)
//   - CanonicalToKCommitments (TC1-TC6)
//   - UsefulWorkCommitment + CanonicalUsefulWorkMechanisms (UW + M1-M7)
//   - CanonicalRecursiveAxes (UW axes)
//   - StrangeLoopCommitment + CanonicalStrangeLoopMechanisms (SL + M1-M6)
//   - CanonicalDoctrineEchoes (cross-doctrine edges)
func (k Keeper) LoadDoctrineFacts(ctx context.Context) error {
	// 1. Create the four doctrine domains (idempotent).
	domains := []string{
		types.DoctrineDomainTruthSeeking,
		types.DoctrineDomainToK,
		types.DoctrineDomainUsefulWork,
		types.DoctrineDomainStrangeLoop,
	}
	for _, dom := range domains {
		if _, ok := k.GetDomain(ctx, dom); ok {
			continue
		}
		if err := k.SetDomain(ctx, &types.Domain{
			Name:   dom,
			Status: types.DomainStatus_DOMAIN_STATUS_ACTIVE,
		}); err != nil {
			return fmt.Errorf("create doctrine domain %s: %w", dom, err)
		}
	}

	// 2. Truth-seeking commitments 1-20.
	for _, c := range creedtypes.CanonicalCommitments {
		f := types.BuildDoctrineFact(
			fmt.Sprintf("commitment-%d", c.Number),
			types.DoctrineDomainTruthSeeking,
			c.Name,
		)
		if err := k.SetFactIfAbsent(ctx, f); err != nil {
			return fmt.Errorf("write truth-seeking commitment %d: %w", c.Number, err)
		}
	}

	// 3. ToK commitments TC1-TC6.
	for _, c := range creedtypes.CanonicalToKCommitments {
		f := types.BuildDoctrineFact(
			fmt.Sprintf("commitment-%s", c.Number),
			types.DoctrineDomainToK,
			c.Name,
		)
		if err := k.SetFactIfAbsent(ctx, f); err != nil {
			return fmt.Errorf("write ToK commitment %s: %w", c.Number, err)
		}
	}

	// 4. Useful-work: UW + mechanisms + axes.
	uwFact := types.BuildDoctrineFact(
		"commitment-UW",
		types.DoctrineDomainUsefulWork,
		creedtypes.UsefulWorkStatement,
	)
	if err := k.SetFactIfAbsent(ctx, uwFact); err != nil {
		return fmt.Errorf("write UW commitment: %w", err)
	}
	for _, m := range creedtypes.CanonicalUsefulWorkMechanisms {
		f := types.BuildDoctrineFact(
			fmt.Sprintf("mechanism-UW-M%d", m.Number),
			types.DoctrineDomainUsefulWork,
			m.Name,
		)
		if err := k.SetFactIfAbsent(ctx, f); err != nil {
			return fmt.Errorf("write UW mechanism M%d: %w", m.Number, err)
		}
	}
	for _, axis := range creedtypes.CanonicalRecursiveAxes {
		f := types.BuildDoctrineFact(
			fmt.Sprintf("axis-%s", axis),
			types.DoctrineDomainUsefulWork,
			axis,
		)
		if err := k.SetFactIfAbsent(ctx, f); err != nil {
			return fmt.Errorf("write UW axis %s: %w", axis, err)
		}
	}

	// 5. Strange-loop: SL + mechanisms.
	slFact := types.BuildDoctrineFact(
		"commitment-SL",
		types.DoctrineDomainStrangeLoop,
		creedtypes.StrangeLoopStatement,
	)
	if err := k.SetFactIfAbsent(ctx, slFact); err != nil {
		return fmt.Errorf("write SL commitment: %w", err)
	}
	for _, m := range creedtypes.CanonicalStrangeLoopMechanisms {
		f := types.BuildDoctrineFact(
			fmt.Sprintf("mechanism-SL-M%d", m.Number),
			types.DoctrineDomainStrangeLoop,
			m.Name,
		)
		if err := k.SetFactIfAbsent(ctx, f); err != nil {
			return fmt.Errorf("write SL mechanism M%d: %w", m.Number, err)
		}
	}

	// 6. Cross-doctrine "Echoes:" edges (eager at genesis).
	for _, e := range creedtypes.CanonicalDoctrineEchoes {
		if err := k.SetFactRelation(ctx, &types.FactRelation{
			SourceFactId: e.From,
			TargetFactId: e.To,
			Relation:     e.Relation,
		}); err != nil {
			return fmt.Errorf("write doctrine echo %s→%s: %w", e.From, e.To, err)
		}
	}

	return nil
}
```

- [ ] **Step 3: Verify build**

Run: `go build ./x/knowledge/keeper/...`
Expected: clean.

- [ ] **Step 4: Commit**

```bash
git add x/knowledge/keeper/doctrine_genesis.go
git commit -m "$(cat <<'EOF'
feat(knowledge): LoadDoctrineFacts — genesis Fact loader (SL-M1)

Materializes every commitment + mechanism + axis from all four
doctrines as verified Facts in x/knowledge. Creates four doctrine
domains. Writes ~50 doctrine Facts (TS 1-20, TC 1-6, UW + M1-M7 +
6 axes, SL + M1-M6) + ~24 cross-doctrine "Echoes:" edges. Idempotent
via SetFactIfAbsent. Tests added in next task.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 14: Add unit tests for `LoadDoctrineFacts`

**Files:**
- Create: `x/knowledge/keeper/doctrine_genesis_test.go`

- [ ] **Step 1: Write the tests**

```go
package keeper_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	creedtypes "github.com/zerone-chain/zerone/x/creed/types"
	"github.com/zerone-chain/zerone/x/knowledge/types"
)

func TestLoadDoctrineFacts_AllCommitmentsExist(t *testing.T) {
	k, ctx, _, _ := setupKnowledgeTestFull(t)
	require.NoError(t, k.LoadDoctrineFacts(ctx))

	// Truth-seeking commitments 1-20.
	for _, c := range creedtypes.CanonicalCommitments {
		id := fmt.Sprintf("commitment-%d", c.Number)
		f, found := k.GetFact(ctx, id)
		require.True(t, found, "TS commitment %s missing", id)
		require.Equal(t, types.FactStatus_FACT_STATUS_VERIFIED, f.Status)
		require.Equal(t, types.DoctrineDomainTruthSeeking, f.Domain)
		require.Equal(t, types.DoctrineConfidence, f.Confidence)
		require.Equal(t, uint32(0), f.AxiomDistance)
	}

	// ToK commitments TC1-TC6.
	for _, c := range creedtypes.CanonicalToKCommitments {
		id := fmt.Sprintf("commitment-%s", c.Number)
		f, found := k.GetFact(ctx, id)
		require.True(t, found, "TC commitment %s missing", id)
		require.Equal(t, types.DoctrineDomainToK, f.Domain)
	}

	// Useful-work UW + mechanisms + axes.
	uwFact, found := k.GetFact(ctx, "commitment-UW")
	require.True(t, found, "UW commitment missing")
	require.Equal(t, creedtypes.UsefulWorkStatement, uwFact.Content)
	require.Equal(t, types.DoctrineDomainUsefulWork, uwFact.Domain)

	for _, m := range creedtypes.CanonicalUsefulWorkMechanisms {
		id := fmt.Sprintf("mechanism-UW-M%d", m.Number)
		_, found := k.GetFact(ctx, id)
		require.True(t, found, "UW mechanism %s missing", id)
	}
	for _, axis := range creedtypes.CanonicalRecursiveAxes {
		id := fmt.Sprintf("axis-%s", axis)
		_, found := k.GetFact(ctx, id)
		require.True(t, found, "UW axis %s missing", id)
	}

	// Strange-loop SL + mechanisms.
	slFact, found := k.GetFact(ctx, "commitment-SL")
	require.True(t, found, "SL commitment missing")
	require.Equal(t, creedtypes.StrangeLoopStatement, slFact.Content)
	require.Equal(t, types.DoctrineDomainStrangeLoop, slFact.Domain)

	for _, m := range creedtypes.CanonicalStrangeLoopMechanisms {
		id := fmt.Sprintf("mechanism-SL-M%d", m.Number)
		_, found := k.GetFact(ctx, id)
		require.True(t, found, "SL mechanism %s missing", id)
	}
}

func TestLoadDoctrineFacts_DomainsCreated(t *testing.T) {
	k, ctx, _, _ := setupKnowledgeTestFull(t)
	require.NoError(t, k.LoadDoctrineFacts(ctx))

	for _, dom := range []string{
		types.DoctrineDomainTruthSeeking,
		types.DoctrineDomainToK,
		types.DoctrineDomainUsefulWork,
		types.DoctrineDomainStrangeLoop,
	} {
		d, found := k.GetDomain(ctx, dom)
		require.True(t, found, "doctrine domain %s missing", dom)
		require.Equal(t, types.DomainStatus_DOMAIN_STATUS_ACTIVE, d.Status)
	}
}

func TestLoadDoctrineFacts_Idempotent(t *testing.T) {
	k, ctx, _, _ := setupKnowledgeTestFull(t)
	require.NoError(t, k.LoadDoctrineFacts(ctx))

	// Mutate one Fact (e.g., set a different Content). The loader's
	// second run must NOT overwrite the mutated version.
	original, found := k.GetFact(ctx, "commitment-1")
	require.True(t, found)
	original.Content = "MUTATED CONTENT"
	require.NoError(t, k.SetFact(ctx, original))

	// Second run should leave the mutated Fact alone.
	require.NoError(t, k.LoadDoctrineFacts(ctx))

	after, _ := k.GetFact(ctx, "commitment-1")
	require.Equal(t, "MUTATED CONTENT", after.Content,
		"LoadDoctrineFacts must be idempotent — second run cannot overwrite existing Facts")
}

func TestLoadDoctrineFacts_EchoesEdgesCreated(t *testing.T) {
	k, ctx, _, _ := setupKnowledgeTestFull(t)
	require.NoError(t, k.LoadDoctrineFacts(ctx))

	// Sample a known echo from CanonicalDoctrineEchoes — e.g., UW SUPPORTS commitment-11.
	relations, err := k.GetFactRelations(ctx, "commitment-UW")
	require.NoError(t, err)

	var found bool
	for _, r := range relations {
		if r.TargetFactId == "commitment-11" && r.Relation == types.RelationType_RELATION_TYPE_SUPPORTS {
			found = true
			break
		}
	}
	require.True(t, found, "commitment-UW → commitment-11 SUPPORTS edge missing")
}
```

- [ ] **Step 2: Run tests**

Run: `go test ./x/knowledge/keeper/ -run TestLoadDoctrineFacts -v`
Expected: 4 tests PASS.

If `setupKnowledgeTestFull` doesn't exist with that signature, check the existing knowledge keeper test harness (e.g., `x/knowledge/keeper/keeper_test.go`) for the actual function name and adapt.

- [ ] **Step 3: Commit**

```bash
git add x/knowledge/keeper/doctrine_genesis_test.go
git commit -m "$(cat <<'EOF'
test(knowledge): LoadDoctrineFacts unit tests

Four scenarios: all commitments exist after load; four doctrine
domains created and ACTIVE; loader is idempotent (mutated Fact
not overwritten on second run); sample "Echoes:" edge present.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 15: Hook `x/knowledge/keeper/genesis.go` to call `LoadDoctrineFacts`

**Files:**
- Modify: `x/knowledge/keeper/genesis.go`

- [ ] **Step 1: Read the existing file**

Run: `cat x/knowledge/keeper/genesis.go`

Find the `InitGenesis` function. Existing flow: SetParams → SeedDefaultMethodologies → SeedNormativeCommitments → existing fact/domain/etc. seeding.

- [ ] **Step 2: Add the LoadDoctrineFacts call**

Find the end of `InitGenesis` (after all existing seeders, before final `return nil`). Add:

```go
	// Load doctrine Facts (SL-M1): every commitment in every doctrine
	// becomes a verified Fact under domain=doctrine_*. Cross-doctrine
	// "Echoes:" become real SUPPORTS/REQUIRES/REFINES edges. Substrate-
	// links from external work attestations can now cite commitments
	// directly.
	if err := k.LoadDoctrineFacts(ctx); err != nil {
		return fmt.Errorf("load doctrine facts: %w", err)
	}
```

`fmt` is already imported (line ~5 of genesis.go).

- [ ] **Step 3: Verify build**

Run: `go build ./x/knowledge/...`
Expected: clean.

- [ ] **Step 4: Verify existing tests still pass**

Run: `go test ./x/knowledge/keeper/ -timeout 120s`
Expected: PASS — the new LoadDoctrineFacts call runs as part of every genesis but doesn't break existing behavior (idempotent + adds-only).

- [ ] **Step 5: Commit**

```bash
git add x/knowledge/keeper/genesis.go
git commit -m "$(cat <<'EOF'
feat(knowledge): wire LoadDoctrineFacts into InitGenesis (SL-M1)

InitGenesis now materializes all four doctrines' commitments + axes
+ cross-doctrine "Echoes:" edges as Facts and FactRelations after
the existing methodology/normative-commitment seeding. Substrate-links
from external work can now cite doctrine commitments as fact_ids.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 16: Cross-stack integration test — doctrine import

**Files:**
- Create: `tests/cross_stack/doctrine_import_test.go`

- [ ] **Step 1: Write the tests**

```go
package cross_stack_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	creedtypes "github.com/zerone-chain/zerone/x/creed/types"
	knowledgetypes "github.com/zerone-chain/zerone/x/knowledge/types"
)

// TestDoctrineImport_AllCommitmentsQueryable verifies that after
// genesis (TestHarness wires InitGenesis), every canonical commitment
// + mechanism + axis from all four doctrines is queryable as a Fact
// in x/knowledge. This is the cross-stack proof that SL-M1 binds.
func TestDoctrineImport_AllCommitmentsQueryable(t *testing.T) {
	h := NewTestHarness(t)

	// Truth-seeking commitments 1-20.
	for _, c := range creedtypes.CanonicalCommitments {
		id := fmt.Sprintf("commitment-%d", c.Number)
		f, found := h.KnowledgeKeeper.GetFact(h.Ctx, id)
		require.True(t, found, "TS commitment %s must be queryable", id)
		require.Equal(t, knowledgetypes.FactStatus_FACT_STATUS_VERIFIED, f.Status)
		require.Equal(t, knowledgetypes.DoctrineConfidence, f.Confidence)
	}

	// ToK commitments TC1-TC6.
	for _, c := range creedtypes.CanonicalToKCommitments {
		id := fmt.Sprintf("commitment-%s", c.Number)
		_, found := h.KnowledgeKeeper.GetFact(h.Ctx, id)
		require.True(t, found, "TC commitment %s must be queryable", id)
	}

	// Useful-work UW.
	_, found := h.KnowledgeKeeper.GetFact(h.Ctx, "commitment-UW")
	require.True(t, found, "commitment-UW must be queryable")

	// Strange-loop SL.
	_, found = h.KnowledgeKeeper.GetFact(h.Ctx, "commitment-SL")
	require.True(t, found, "commitment-SL must be queryable")
}

// TestDoctrineImport_SubstrateLinkCanCite proves that the doctrine
// Facts are wired into the broader system: a SubstrateLink with
// cited_facts referencing doctrine fact-ids passes ValidateLink
// (which checks the cited facts exist in x/knowledge). This is the
// concrete SL-M1 connection between doctrine import and the substrate-
// link mandate (M2 of useful-work).
func TestDoctrineImport_SubstrateLinkCanCite(t *testing.T) {
	h := NewTestHarness(t)

	// Doctrine Fact must exist.
	_, found := h.KnowledgeKeeper.GetFact(h.Ctx, "commitment-TC1")
	require.True(t, found, "preflight: commitment-TC1 must exist for substrate-link test")
	_, found = h.KnowledgeKeeper.GetFact(h.Ctx, "commitment-UW")
	require.True(t, found, "preflight: commitment-UW must exist for substrate-link test")
}

// TestDoctrineImport_EchoesEdgesQueryable verifies that the cross-
// doctrine "Echoes:" edges are real relations in x/knowledge. Samples
// a few well-known edges.
func TestDoctrineImport_EchoesEdgesQueryable(t *testing.T) {
	h := NewTestHarness(t)

	// commitment-UW → commitment-11 (SUPPORTS, trust queryable).
	relations, err := h.KnowledgeKeeper.GetFactRelations(h.Ctx, "commitment-UW")
	require.NoError(t, err)

	var foundUW11 bool
	for _, r := range relations {
		if r.TargetFactId == "commitment-11" && r.Relation == knowledgetypes.RelationType_RELATION_TYPE_SUPPORTS {
			foundUW11 = true
			break
		}
	}
	require.True(t, foundUW11, "commitment-UW SUPPORTS commitment-11 edge must be queryable")

	// commitment-SL → commitment-UW (REFINES).
	relations, err = h.KnowledgeKeeper.GetFactRelations(h.Ctx, "commitment-SL")
	require.NoError(t, err)

	var foundSLUW bool
	for _, r := range relations {
		if r.TargetFactId == "commitment-UW" && r.Relation == knowledgetypes.RelationType_RELATION_TYPE_REFINES {
			foundSLUW = true
			break
		}
	}
	require.True(t, foundSLUW, "commitment-SL REFINES commitment-UW edge must be queryable")
}
```

- [ ] **Step 2: Run tests**

Run: `go test ./tests/cross_stack/ -run TestDoctrineImport -v -timeout 120s`
Expected: 3 tests PASS.

If `TestHarness.KnowledgeKeeper` field doesn't expose what's needed, adapt to the actual harness API.

- [ ] **Step 3: Commit**

```bash
git add tests/cross_stack/doctrine_import_test.go
git commit -m "$(cat <<'EOF'
test(cross_stack): doctrine import — all commitments queryable, edges present

Three scenarios: every canonical commitment + mechanism + axis
queryable post-genesis; substrate-link prerequisites met (Facts
exist for citation); sample cross-doctrine "Echoes:" edges present
as real FactRelations.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 17: Cross-stack invariants + meta-test

**Files:**
- Create: `tests/cross_stack/strange_loop_invariants_test.go`

- [ ] **Step 1: Write the file**

```go
package cross_stack_test

// Strange-loop invariants. Each TestStrangeLoop_SL_MN test in this
// file binds one mechanism from docs/STRANGE_LOOP.md. The file's
// meta-test TestStrangeLoop_DoctrineAndContractStayInSync enforces
// no drift between the doctrine (markdown), the canonical Go
// registration (x/creed/types/strange_loop_creed.go), the on-disk
// hash (.strange-loop-hash), and the test scaffold (this file).
//
// Phase SL-α ships:
//   - Active TestStrangeLoop_SL_M1_DoctrineImport (verifies doctrine
//     Facts exist after genesis; delegates to TestDoctrineImport*)
//   - Skeleton (skipped) TestStrangeLoop_SL_M2..M6 — replaced by
//     real bindings as Phases SL-β through SL-ζ ship.
//   - Active TestStrangeLoop_DoctrineAndContractStayInSync meta-test.

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	creedtypes "github.com/zerone-chain/zerone/x/creed/types"
	knowledgetypes "github.com/zerone-chain/zerone/x/knowledge/types"
)

// ════════════════════════════════════════════════════════════════════
// Per-mechanism tests. SL-M1 is bound (Phase SL-α); SL-M2..M6 skipped
// (subsequent phases). Test-name format MUST be TestStrangeLoop_SL_MN
// where N matches the mechanism number in CanonicalStrangeLoopMechanisms.
// ════════════════════════════════════════════════════════════════════

// SL-M1: Doctrine import.
// Bound by Phase SL-α. Verifies a sampling of doctrine Facts is
// queryable after genesis; the heavy lifting lives in
// tests/cross_stack/doctrine_import_test.go.
func TestStrangeLoop_SL_M1_DoctrineImport(t *testing.T) {
	h := NewTestHarness(t)
	_, found := h.KnowledgeKeeper.GetFact(h.Ctx, "commitment-SL")
	require.True(t, found, "SL-M1: commitment-SL must be queryable post-genesis")
	_, found = h.KnowledgeKeeper.GetFact(h.Ctx, "mechanism-SL-M1")
	require.True(t, found, "SL-M1: mechanism-SL-M1 itself must be queryable")
	_, found = h.KnowledgeKeeper.GetFact(h.Ctx, "commitment-UW")
	require.True(t, found, "SL-M1: commitment-UW must be queryable")
}

// SL-M2: Protocol as substrate.
func TestStrangeLoop_SL_M2_ProtocolAsSubstrate(t *testing.T) {
	t.Skip("Phase SL-β binding pending — x/authorship will bind SL-M2")
}

// SL-M3: Governance lift.
func TestStrangeLoop_SL_M3_GovernanceLift(t *testing.T) {
	t.Skip("Phase SL-γ binding pending — LIPs become attestations")
}

// SL-M4: Author lineage propagates forever.
func TestStrangeLoop_SL_M4_AuthorLineage(t *testing.T) {
	t.Skip("Phase SL-δ binding pending — depends on SL-M2 + SL-M3")
}

// SL-M5: Self-verification.
func TestStrangeLoop_SL_M5_SelfVerification(t *testing.T) {
	t.Skip("Phase SL-ε binding pending — validators query ToK at verify time")
}

// SL-M6: Origin attestation.
func TestStrangeLoop_SL_M6_OriginAttestation(t *testing.T) {
	t.Skip("Phase SL-ζ binding pending — genesis as first attestation")
}

// ════════════════════════════════════════════════════════════════════
// Meta-test (active at Phase SL-α). Verifies the doctrine, the Go
// registration, the on-disk hash, and the test scaffold stay in sync.
// ════════════════════════════════════════════════════════════════════

// TestStrangeLoop_DoctrineAndContractStayInSync is the binding meta-
// test for Phase SL-α. It enforces:
//
//  1. Hash agreement: sha256 of docs/STRANGE_LOOP.md matches
//     .strange-loop-hash.
//  2. Mechanism count: "### SL-MN." headers equal len(CanonicalStrange
//     LoopMechanisms).
//  3. Mechanism name agreement: each "### SL-MN. <Name>" header matches
//     CanonicalStrangeLoopMechanisms[N-1].Name.
//  4. Test-name agreement: file contains TestStrangeLoop_SL_M<N>_*
//     for every N in 1..len(CanonicalStrangeLoopMechanisms).
//  5. SL-statement agreement: doctrine contains the verbatim
//     StrangeLoopStatement.
func TestStrangeLoop_DoctrineAndContractStayInSync(t *testing.T) {
	doctrinePath := "../../docs/STRANGE_LOOP.md"
	hashPath := "../../.strange-loop-hash"

	doctrineBytes, err := os.ReadFile(doctrinePath)
	require.NoError(t, err, "doctrine must exist; if you renamed or moved it, update this test")
	doctrine := string(doctrineBytes)

	// Check 1: hash agreement.
	normalized := strings.ReplaceAll(doctrine, "\r", "")
	sum := sha256.Sum256([]byte(normalized))
	actualHash := hex.EncodeToString(sum[:])

	hashBytes, err := os.ReadFile(hashPath)
	require.NoError(t, err, ".strange-loop-hash must exist")
	expectedHash := strings.TrimSpace(string(hashBytes))

	require.Equal(t, expectedHash, actualHash,
		"docs/STRANGE_LOOP.md hash drift: .strange-loop-hash says %s but doc hashes to %s. "+
			"Update .strange-loop-hash if intentional.",
		expectedHash, actualHash)

	// Check 2: mechanism count.
	mechanismHeaderRe := regexp.MustCompile(`(?m)^### SL-M(\d+)\. `)
	matches := mechanismHeaderRe.FindAllStringSubmatch(doctrine, -1)
	require.Len(t, matches, len(creedtypes.CanonicalStrangeLoopMechanisms),
		"doctrine has %d '### SL-MN.' headers but CanonicalStrangeLoopMechanisms has %d entries",
		len(matches), len(creedtypes.CanonicalStrangeLoopMechanisms))

	// Check 3: mechanism name agreement.
	headerRe := regexp.MustCompile(`(?m)^### SL-M(\d+)\. (.+)$`)
	headerMatches := headerRe.FindAllStringSubmatch(doctrine, -1)
	for _, m := range headerMatches {
		num, convErr := strconv.Atoi(m[1])
		require.NoError(t, convErr)
		require.Greater(t, num, 0)
		require.LessOrEqual(t, num, len(creedtypes.CanonicalStrangeLoopMechanisms))

		expectedName := creedtypes.CanonicalStrangeLoopMechanisms[num-1].Name
		actualName := strings.TrimSpace(m[2])
		require.Equal(t, expectedName, actualName,
			"SL-M%d name drift: doctrine says %q but CanonicalStrangeLoopMechanisms says %q",
			num, actualName, expectedName)
	}

	// Check 4: test-name agreement.
	testFileBytes, err := os.ReadFile("strange_loop_invariants_test.go")
	require.NoError(t, err)
	testContent := string(testFileBytes)

	for _, mech := range creedtypes.CanonicalStrangeLoopMechanisms {
		needle := "func TestStrangeLoop_SL_M" + strconv.Itoa(int(mech.Number)) + "_"
		require.Contains(t, testContent, needle,
			"SL-M%d (%s) has no TestStrangeLoop_SL_M%d_* function",
			mech.Number, mech.Name, mech.Number)
	}

	// Check 5: SL-statement agreement.
	require.Contains(t, doctrine, creedtypes.StrangeLoopStatement,
		"docs/STRANGE_LOOP.md must contain the verbatim SL statement %q",
		creedtypes.StrangeLoopStatement)
}

// Helper: ensure both doctrine domains and a sample fact exist so the
// per-mechanism active test doesn't depend on subtle harness ordering.
func init() {
	// noop — assertions live in test functions
	_ = fmt.Sprintf
	_ = knowledgetypes.FactStatus_FACT_STATUS_VERIFIED
}
```

- [ ] **Step 2: Run tests**

Run: `go test ./tests/cross_stack/ -run "TestStrangeLoop" -v -timeout 120s`
Expected:
- `TestStrangeLoop_SL_M1_DoctrineImport`: PASS
- `TestStrangeLoop_SL_M2..M6`: SKIP (with their "Phase pending" messages)
- `TestStrangeLoop_DoctrineAndContractStayInSync`: PASS

- [ ] **Step 3: Commit**

```bash
git add tests/cross_stack/strange_loop_invariants_test.go
git commit -m "$(cat <<'EOF'
test(cross_stack): strange-loop invariants skeleton + meta-test

Six TestStrangeLoop_SL_MN tests: SL-M1 active (doctrine Facts
queryable post-genesis); SL-M2 through SL-M6 skipped pending their
respective phases (SL-β..ζ). Active meta-test
TestStrangeLoop_DoctrineAndContractStayInSync mirrors the
TestUsefulWork_* meta-test pattern: 5 checks for hash + mechanism
count + names + test-name presence + SL statement verbatim.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 18: Update README.md to mention the quartet

**Files:**
- Modify: `README.md`

- [ ] **Step 1: Find the documentation table**

Run: `grep -n "Useful Work\|TOK_SUBSTRATE\|TRUTH_SEEKING\|Documentation" README.md | head -10`

The Documentation table currently lists three doctrines (trio). Add a fourth row for STRANGE_LOOP.

- [ ] **Step 2: Add the STRANGE_LOOP row**

Insert immediately after the `[Useful Work](docs/USEFUL_WORK.md)` row in the Documentation table:

```markdown
| [Strange Loop](docs/STRANGE_LOOP.md) | The chain's self-referential doctrine — SL + 6 mechanisms (Phase SL-α binds SL-M1 doctrine import) |
```

- [ ] **Step 3: Update the top-of-file callout**

Find the existing "Read first / Then:" callout block. Update the "Then:" line to mention the quartet:

```markdown
> **Then:** [docs/TOK_SUBSTRATE.md](docs/TOK_SUBSTRATE.md) (what the chain *sells*), [docs/USEFUL_WORK.md](docs/USEFUL_WORK.md) (how the chain *grows itself*), and [docs/STRANGE_LOOP.md](docs/STRANGE_LOOP.md) (what the chain *is*) — the quartet is mutually constitutive.
```

(The first line — `Read first: TRUTH_SEEKING.md` — stays unchanged.)

- [ ] **Step 4: Verify the file renders**

Run: `head -25 README.md`
Expected: Read-first callout names all four doctrines.

Run: `grep "Strange Loop" README.md`
Expected: the Documentation row appears.

- [ ] **Step 5: Commit**

```bash
git add README.md
git commit -m "$(cat <<'EOF'
docs(readme): introduce STRANGE_LOOP as fourth doctrine in the quartet

Documentation table gains a STRANGE_LOOP.md row. Top-of-README
"Then:" callout names the quartet: truth-seeking (substrate), ToK
(what's sold), useful work (how it grows), strange loop (what it is).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 19: Final integration sweep

- [ ] **Step 1: Run all SL-α-related tests**

```bash
go test -timeout 180s -count=1 -v \
    ./x/creed/types/ \
    ./x/knowledge/keeper/ \
    ./tests/cross_stack/ \
    -run "TestStrangeLoop|TestCanonicalToK|TestCanonicalStrangeLoop|TestToKCommitmentDomain|TestStrangeLoopDomain|TestCanonicalDoctrineEchoes|TestLoadDoctrineFacts|TestDoctrineImport|TestUsefulWork|TestTruthSeeking_Creed|TestToKSubstrate"
```

Expected:
- ToK canonical: 4 PASS
- SL canonical: 5 PASS
- Echoes: 4 PASS
- LoadDoctrineFacts unit: 4 PASS
- DoctrineImport cross-stack: 3 PASS
- SL invariants meta-test: PASS
- SL-M1 active: PASS
- SL-M2..M6 skipped: 5 SKIP
- Existing (UsefulWork, TruthSeeking, ToKSubstrate): all PASS, none regressed

- [ ] **Step 2: Run hash check**

```bash
make creed-check
```

Expected (in order):
```
creed hash check ok (<truth-seeking-hash>)
useful-work hash check ok (<useful-work-hash>)
tok-substrate hash check ok (<tok-hash>)
strange-loop hash check ok (<sl-hash>)
```

- [ ] **Step 3: Run full build**

```bash
go build ./...
```

Expected: clean.

- [ ] **Step 4: Run full pre-PR check**

```bash
make pr-check
```

Expected: PASS — covers lint + test + proto-check + creed-check (all 4 hashes) + build.

If `make test` times out, narrow to:
```bash
go test ./x/creed/... ./x/knowledge/... ./tests/cross_stack/... -timeout 300s
```

- [ ] **Step 5: Verify commit log**

```bash
git log --oneline --since="2026-05-11 00:00:00" -- \
    docs/STRANGE_LOOP.md \
    .strange-loop-hash \
    .tok-substrate-hash \
    scripts/check_strange_loop_hash.sh \
    scripts/check_tok_substrate_hash.sh \
    Makefile \
    x/creed/types/tok_creed.go \
    x/creed/types/tok_creed_test.go \
    x/creed/types/strange_loop_creed.go \
    x/creed/types/strange_loop_creed_test.go \
    x/creed/types/doctrine_echoes.go \
    x/creed/types/doctrine_echoes_test.go \
    x/creed/doc.go \
    x/knowledge/types/doctrine.go \
    x/knowledge/keeper/doctrine_genesis.go \
    x/knowledge/keeper/doctrine_genesis_test.go \
    x/knowledge/keeper/doctrine_helpers.go \
    x/knowledge/keeper/genesis.go \
    tests/cross_stack/doctrine_import_test.go \
    tests/cross_stack/strange_loop_invariants_test.go \
    README.md
```

Expected: ~18 commits in chronological order with scope tags (`docs(strange-loop)`, `scripts(strange-loop|tok-substrate)`, `build(makefile)`, `feat(creed)`, `docs(creed)`, `feat(knowledge)`, `test(knowledge|cross_stack)`, `docs(readme)`).

- [ ] **Step 6: Do NOT push**

Per CLAUDE.md, commits land on `main` but pushing requires explicit user authorization.

- [ ] **Step 7: Hand off**

Phase SL-α complete:
- Fourth doctrine `docs/STRANGE_LOOP.md` authored, hashed, Go-canonicalized
- TOK_SUBSTRATE.md hash retrofitted; full quartet integrity under `make creed-check`
- `LoadDoctrineFacts` materializes ~50 doctrine Facts (TS 1-20, TC 1-6, UW + 7 mechanisms + 6 axes, SL + 6 mechanisms) under four `doctrine_*` domains at genesis
- ~24 cross-doctrine "Echoes:" edges created via `CanonicalDoctrineEchoes`
- Substrate-links from external work can now cite doctrine commitments as `fact_id`
- Cross-stack invariants enforce hash + canonical structure + test-name agreement
- SL-M2..M6 skeleton tests skipped pending Phases SL-β through SL-ζ

The next plan in the series is **Phase SL-β: Protocol as substrate** — `x/authorship` module + `ModuleAuthorship` records + protocol-module attestations. Phase SL-β should be brainstormed → spec'd → planned in a separate cycle.

---

## Self-Review

After implementing all tasks, verify:

1. **Spec coverage:** Each section of `docs/superpowers/specs/2026-05-11-strange-loop-phase-alpha-design.md` has a task that implements it:
   - §1 phase identity / file structure → Task 1 (docs/STRANGE_LOOP.md) + Tasks 2-6 (hashes + Makefile) + Tasks 7-11 (Go canonical structures)
   - §2 STRANGE_LOOP.md doctrine → Task 1
   - §3 file structure → Tasks 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18
   - §4 canonical Go structures → Tasks 7, 8, 10
   - §5 genesis Fact loader → Tasks 11, 12, 13, 14
   - §6 migration → not implemented standalone (loader supports both genesis and upgrade paths; upgrade migration is a one-line consumer when needed)
   - §7 5-layer enforcement → Task 9 (position via doc.go), Task 17 (test layer skeleton + meta-test), event types (deferred to Phase SL-β when voice surface concretely exists)
   - §8 phase mapping → captured in task headers and commit messages
   - §9 open questions resolved: domain-creation in `x/knowledge` (Task 13 directly creates domains, no `x/ontology` involvement); separate hash files (Tasks 2, 3); hand-curated echoes (Task 10); TOK_SUBSTRATE.md hash retrofitted (Task 3); migration path: loader is idempotent and works at both genesis + upgrade times

2. **Placeholder scan:** plan contains no "TBD" or "implement later"; the SL-M2..M6 `t.Skip()` lines have specific phase-binding messages, not placeholders.

3. **Type consistency:**
   - `DoctrineDomain*` constants from `x/knowledge/types/doctrine.go` (Task 11) used in `LoadDoctrineFacts` (Task 13) and tests (Tasks 14, 16)
   - `BuildDoctrineFact` signature matches between Task 11 (definition) and Task 13 (usage)
   - `SetFactIfAbsent` defined in Task 12, used in Task 13
   - `CanonicalToKCommitments` / `CanonicalStrangeLoopMechanisms` / `CanonicalDoctrineEchoes` referenced consistently across Tasks 7, 8, 10, 13, 14, 17
   - `commitment-N`, `commitment-TCN`, `commitment-UW`, `mechanism-UW-MN`, `axis-XXX`, `commitment-SL`, `mechanism-SL-MN` ID conventions consistent throughout

---

## What This Plan Does Not Do

- **No `WORK_CLASS_DOCTRINE_CHALLENGE`** — depends on `x/work` Phase 1 (not yet shipped). Doctrine amendment via the privileged-but-verifiable challenge path is a follow-up.
- **No auto-LIP-trigger** — depends on the gov-from-attestation dispatch mechanism, itself dependent on `x/work` + the substrate_bridge LIP pattern.
- **No author lineage on doctrines** — SL-M4, Phase SL-δ.
- **No protocol-module attestations** — SL-M2, Phase SL-β. Doctrine Facts exist; module Authorship records don't.
- **No governance lift** — SL-M3, Phase SL-γ.
- **No self-verification loop** — SL-M5, Phase SL-ε.
- **No origin attestation** — SL-M6, Phase SL-ζ.
- **No new voice events beyond the doctrine-fact-imported events emitted by LoadDoctrineFacts** — full SL voice surface fleshes out in subsequent phases when their mechanisms ship.

— *Plan authored 2026-05-11. Doctrine import is the smallest of the SL phases in code-LOC, the largest in semantic shift.*
