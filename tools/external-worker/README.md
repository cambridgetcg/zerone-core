# External knowledge worker

A bounded external computation connected to Zerone's existing contribution
journal: either **explicitly unpaid compute-only** work or an operator-mediated,
**test-token-only bounty** using the SDK bank. It does not
change consensus, mint tokens, activate a useful-work module, or treat a payment
as scientific acceptance. Actual execution receipts belong outside this checkout.

The product direction is global compute contributing to the Tree of Knowledge.
This is one complete narrow work class, not a permissionless job market. The
existing [useful-work](../../docs/USEFUL_WORK.md) and
[ToK](../../docs/TOK_SUBSTRATE.md) targets remain broader than this implementation.

## What the first task checks

Bitcoin Core v27.0 documents that duplicate-last Merkle padding can give
different-length leaf lists the same root, and detects equal adjacent pairs
before padding as a mutation signal.[1] The task reproduces that construction,
not a new Bitcoin vulnerability or a full-node validation result.

A fresh seed derives an odd number of 32-byte leaves (3–4095). The worker compares
the original list with a list explicitly repeating its final leaf. It returns
both double-SHA256 roots, mutation flags and counts. The coordinator recomputes
them with a separate reduction implementation. Digests are in raw byte order,
not the reversed display convention of transaction IDs. No paper-provided code,
network requests, arbitrary task code or model judgments execute in the worker.

This is bounded, small computation. It demonstrates transport, checking,
attribution and optional test settlement—not GPU throughput, novel discovery, cheap verification
of arbitrary research or proof that physical compute was expended. Verification
recomputes this work. Copied correct outputs cannot establish who performed it.

## Responsibility and trust boundaries

- `worker.py`: stdlib-only task validation, fixed computation, separate verifier
  and CLI. The paid-v1 task includes recipient, amount, funding transaction and
  genesis; compute-only tasks have no reward. The schema and complete unsigned
  task are hash-bound. Declared reward terms are not verified funding.
- `coordinator.py`: verifies before creating a new task journal; retains source,
  question, method, experiment, finding and review with typed relations. It
  rechecks the complete task/result/journal before opening a payer. Input JSON is
  data. The existing private literature collection is never imported or uploaded.
- `native.py`: uses an existing pinned participant home's signing implementation.
  It refuses non-loopback/shared/production descriptors. The caller must be on an
  isolated `local_test=true` genesis; the string `zerone-dev-1` is not sufficient.
  It reads the prefunding transfer and available balance, signs an exact SDK
  `MsgSend`, and checks the committed transaction plus H−1/H bank changes/supply.
- `settlement.py`: pure checks for exact recipient/amount/memo/fees, observed bank
  deltas and per-task recovery. The memo binds task ID, result artifact and the
  pre-payment journal snapshot. Payment evidence is appended afterward, avoiding
  a circular hash dependency.
- [External knowledge worker workflow](../../.github/workflows/external-worker.yml):
  opt-in GitHub-hosted computation, read-only repository permission, no signing
  keys or payments, fixed time limit and an allowlisted output artifact. Its
  environment metadata is not proof of independent worker control.

For bounty tasks, funds are **prefunded sponsor balances**, not protocol-locked task escrow.
The sponsor still controls acceptance and payment. Native neutral-review fees
are a different mechanism: valid timely reveals are payable irrespective of a
computation's quality. This tool does not rewrite that policy or fabricate
reviewers to obtain an accepted Fact.

Task IDs and a durable payer-home reservation prevent automatic duplicate payment
within that retained home. Keep the complete payer home and never restore an
older copy or bypass it with manual transfers. This is not a globally enforced
on-chain task-consumption registry. A pending, failed or ambiguous attempt blocks
new signing; inspect/reconcile the exact retained transaction through the existing
participant client. Never compensate uncertainty by signing another payment.

The compute process receives no wallet keys. A bounty task nominates a recipient;
that nomination and a worker label do not prove control of the recipient key.
A GitHub-hosted run under the sponsor's repository remains one operator control
context even though computation occurs on another machine. Independent operator
participation, safe open admission, artifact storage guarantees and trustless
settlement remain separate milestones. Endpoint observations are not independent
inclusion proofs, and development balances have no promised value.

## Run without funding (compute-only)

Use Python 3.11+ and set `SEED_HEX` to 64 lowercase hexadecimal characters
(32 bytes). Leaf counts are odd integers from 3 through 4095. No node,
signer home, recipient or synthetic funding values are needed.

```sh
python3 -B tools/external-worker/worker.py make-compute-task \
  --seed-hex "$SEED_HEX" --leaf-count 4095 --output /ABSOLUTE/task.json
python3 -B tools/external-worker/worker.py run \
  --task /ABSOLUTE/task.json --output /ABSOLUTE/result.json \
  --worker-id 'DECLARED_PUBLIC_WORKER_LABEL'
python3 -B tools/external-worker/worker.py verify \
  --task /ABSOLUTE/task.json --result /ABSOLUTE/result.json
python3 -B tools/external-worker/coordinator.py prepare \
  --task /ABSOLUTE/task.json --result /ABSOLUTE/result.json \
  --output /ABSOLUTE/new-task-workspace
python3 -B tools/external-worker/coordinator.py inspect \
  --workspace /ABSOLUTE/new-task-workspace
```

The Python API is `make_compute_task(seed_hex, leaf_count)`. Its closed
`zerone.external-compute-task/v1` schema has exactly `schema`, `task_id`, `kind`,
`seed_hex` and `leaf_count`; `reward` is forbidden, even `null`. It uses the same
result, independent verifier and public hosted-task transport as the bounty path.
Task IDs bind the schema too: stripping a paid task's reward or swapping its
schema does not preserve a valid task ID. `make-task --reward` and
`make_task(seed_hex, leaf_count, reward_dict)` remain the explicit paid-v1 path;
missing reward never means unpaid work.

`prepare` retains the scoped journal with no funding, nominated recipient or
payment eligibility. Its receipt and `inspect` report `paid: false` and
`payment_eligible: false`. `funding` and `pay` refuse compute-only tasks before
loading a payer or accessing its home, even with `--confirm-test-token-payment`.
Files and the new workspace cannot overwrite existing outputs or follow symlinks.
The loop ends at the verified local journal, not payment or scientific acceptance.

## Run the test-token bounty loop

Use Python 3.11+ and an explicitly authorized fresh isolated runtime from
[the existing runtime guide](../../deploy/networks/zerone-dev-1/README.md).
Initialize a new sponsor participant with the compatible pinned binary and
`--allow-loopback-test`; obtain a faucet grant and read its committed transfer
back. Keep all participant/node homes private. No command below starts or upgrades
a shared network. The examples are command templates, not executed evidence.

Create `reward.json` with exactly these fields, using actual observed values:

```json
{
  "chain_id": "zerone-dev-1",
  "genesis_sha256": "REPLACE_WITH_EXACT_LOCAL_TEST_GENESIS_SHA256",
  "recipient": "REPLACE_WITH_CHECKSUM_VALID_ZRN_RECIPIENT",
  "amount_uzrn": "10000000",
  "funding_txhash": "REPLACE_WITH_EXACT_UPPERCASE_COMMITTED_FUNDING_TXHASH"
}
```

The selected reward is a bounded nominal test amount, not a market price or profit
forecast. The sponsor separately pays transaction fees. Use a fresh random 32-byte
seed encoded as 64 lowercase hexadecimal characters; it distinguishes task inputs,
not worker identity or a new scientific question.

```sh
python3 -B tools/external-worker/worker.py make-task \
  --seed-hex "$SEED_HEX" --leaf-count 4095 \
  --reward /ABSOLUTE/reward.json --output /ABSOLUTE/task.json
python3 -B tools/external-worker/coordinator.py funding \
  --task /ABSOLUTE/task.json --payer-home /ABSOLUTE/private-sponsor-home
```

Give **only the public task** to a willing worker, who can run:

```sh
python3 -B tools/external-worker/worker.py run \
  --task /ABSOLUTE/task.json --output /ABSOLUTE/result.json \
  --worker-id 'DECLARED_PUBLIC_WORKER_LABEL'
```

Alternatively, from an authorized checkout, dispatch the hosted worker with the
JSON file on standard input as `task_json`, using the workflow-dispatch API/`gh`.
The input is parsed as data, never interpolated into shell code. After completion,
read back the exact run ID, source commit, job result and artifact identity using
GitHub's API; download `external-worker-result`. Compare its returned task with
the original, and compare the worker-script hash to that exact commit. Save the
provider observations beside the task; the result's worker label is not enough.
The workflow neither contacts the local chain nor receives sponsor credentials.

On the coordinator machine:

```sh
python3 -B tools/external-worker/worker.py verify \
  --task /ABSOLUTE/task.json --result /ABSOLUTE/result.json
python3 -B tools/external-worker/coordinator.py prepare \
  --task /ABSOLUTE/task.json --result /ABSOLUTE/result.json \
  --output /ABSOLUTE/new-task-workspace
python3 -B tools/external-worker/coordinator.py inspect \
  --workspace /ABSOLUTE/new-task-workspace
python3 -B tools/external-worker/coordinator.py pay \
  --workspace /ABSOLUTE/new-task-workspace \
  --payer-home /ABSOLUTE/private-sponsor-home --confirm-test-token-payment
```

`prepare` creates a new journal only after verification. It refuses existing or
partly written output directories rather than overwriting them. Canonical copies
of task/result are retained; their stored byte hashes identify the evidence.
`pay` is the signing command. A repeated invocation recovers the same committed
payment under the retained payer state, not another transfer. An interrupted
pre-sign reservation with no identified attempt fails closed for operator review.

Open `journal.json` or the later `paid-journal.json` in
[Research Review](https://zerone.ai/research/review/). A task journal is a
contribution graph, not automatic admission to the native Fact graph. To submit
one deliberately chosen assertion, use the existing
[research workflow adapter](../research-review/README.md#carry-a-record-through-native-review)
with `preparation.json`'s `finding_id`, an honest bounded claim, and the same
isolated sponsor. Native review status and direct bounty payment remain separate.

## Checks

```sh
TMPDIR=/ABSOLUTE/WRITABLE/SCRATCH \
  python3 -B -m unittest discover -s tools/external-worker -p 'test_*.py' -v
actionlint .github/workflows/external-worker.yml
```

Unit tests use explicitly synthetic bank observations to fault-inject signing
boundaries; they are not payment evidence. A completed external compute-only loop
requires a real provider run, actual returned result and a verified journal readback.
The bounty loop additionally requires a committed native transfer with matched bank
deltas. Preserve those exact artifacts
and distinguish task execution, scientific scope, custody, and payment in reports.

## Sources

[1] https://raw.githubusercontent.com/bitcoin/bitcoin/v27.0/src/consensus/merkle.cpp — Bitcoin Core v27.0 Merkle tree duplicate-leaf construction and mutation detection
