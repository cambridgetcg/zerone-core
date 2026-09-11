# Put a claim into a block

This walkthrough uses the current source on a persistent local chain. It signs
real Cosmos SDK transactions and reads the retained Tree of Knowledge records.
The browser page reads history; signing happens through the CLI and the local
test keyring. Python 3.11+ is the only extra runtime.

All participants here belong to one operator. Their balances are development
funds and their agreement is not independent scientific endorsement. The
example deliberately shows how an accepted claim can still be wrong.

## Start a knowledge sandbox

Use the exact source revision from the [node guide](https://zerone.ai/nodes/).
Build as described in [LOCAL-NODE.md](LOCAL-NODE.md), then choose a new home:

```sh
python3 scripts/local-node.py init --home "$HOME/zerone-local-claims" \
  --binary "$PWD/build/zeroned" --chain-id zerone-local-claims \
  --rpc-port 48657 --p2p-port 48656 --knowledge-profile
python3 scripts/local-node.py start --home "$HOME/zerone-local-claims"
```

Leave this terminal running. `--knowledge-profile` funds `user`, `reviewer1`,
`reviewer2`, `reviewer3` and `challenger` with 1,000 local ZRN each, alongside
the local consensus validator. It sets commit/reveal/aggregation windows to
300/300/5 blocks at genesis, leaving time to read and perform the manual steps.
The three-reviewer minimum and 77% verdict threshold remain unchanged. Add
`--fast-review` at initialization only for automated exercises (60/60/5 blocks).
The helper refuses to change an existing home.

In a second terminal, from the same checkout, register the development actors:

```sh
python3 scripts/claim-workflow.py onboard --home "$HOME/zerone-local-claims"
python3 scripts/claim-workflow.py serve --home "$HOME/zerone-local-claims" --port 48700
```

Open **http://127.0.0.1:48700**. Use a third terminal for signed actions. The
page never requests a wallet or signs a transaction. It shows only claims
tracked by this local workflow, their retained history, and local transaction
receipts; it is not a global claim census.

## What the records mean

A successful submit transaction stores a **Claim immediately**, already
IN_VERIFICATION when its round starts. The record includes its author, exact
statement, submitted block, method and
reasoning. A Fact is a later derived record, if the review process accepts the
claim. Transaction execution, review phase, review verdict and Fact status are
separate pieces of information.

Commitments bind each reviewer's address, chain, round, vote, confidence, salt,
reason, method, scope and ordered evidence IDs. The workflow saves that exact
tuple before sending the commitment and reveals it without asking you to
retype it. Keep the same home between these steps. Unrevealed tuples stay out
of the browser response. A missed deadline remains a missed deadline; restarting
the tool does not reset block phases.

The current policy requires each reviewer to have at least 100 ZRN in the
account balance, plus fees. This is an admission threshold, not evidence that
the accounts are independent. Two accepts and one reject produce an **INCONCLUSIVE** outcome at
the current threshold. That is a useful, retained result, not a failed system.

## Reproducible evidence

The [example generator](examples/local-claims/prime-polynomial.py) performs trial
division using Python integers. Its two small outputs are kept with this source:

| Evidence | SHA-256 of exact file bytes |
| --- | --- |
| [Review over n=0…39](examples/local-claims/scoped-review.json) | `00b74d248602875cc4d554e95a9d327b612925bad7e2a988d7188c3a8c22b65f` |
| [Counterexample at n=40](examples/local-claims/counterexample.json) | `b16ced168dc69be4c09d1cdebc804254abc7fa193ce49eda6299772dabdd28ed` |

Reproduce either output with `python3 docs/examples/local-claims/prime-polynomial.py
review` or `counterexample`. Evidence IDs use `sha256:<digest>` in this example;
these source links provide the retrieval location. The chain retains the IDs,
not these files. For real research, retain the bytes in durable storage and
include their retrievable location as well as their hash. A hash alone does not
make missing evidence available.

Ordinary `MsgSubmitClaim` has method and reasoning fields but no dedicated
evidence-ID field. This walkthrough identifies the evidence in reasoning;
reviews and contradictions use their existing typed evidence-ID fields. It
does not invent a new consensus message or treat an external evidence hash as
an existing Fact ID.

## Submit, review and challenge

The first claim is deliberately overbroad. Do not use this voting pattern as a
scientific review method: the reviewers explicitly check only 0…39, then vote
to accept a statement extending through 40. The purpose is to retain that
mismatch and its counterexample visibly.

```sh
python3 scripts/claim-workflow.py submit --home "$HOME/zerone-local-claims" \
  --content 'Synthetic deliberately overbroad assertion: for every integer n from 0 through 40 inclusive, n*n+n+41 is prime.' \
  --method M-COMPUTATIONAL \
  --reasoning 'Deliberately flawed local fixture. Evidence: docs/examples/local-claims/scoped-review.json in this source checkout; sha256:00b74d248602875cc4d554e95a9d327b612925bad7e2a988d7188c3a8c22b65f. The evidence covers only n=0 through 39, not the full assertion.'
```

`M-COMPUTATIONAL` is an existing methodology ID in the native genesis. A
nonempty method must name a registered methodology; the description of the
actual work belongs in reasoning and review scope.

The returned JSON contains `claim_id`, `txhash`, committed `height` and execution
`code`. Copy the actual claim ID into a shell variable:

```sh
CLAIM_ID='replace-with-the-returned-claim_id'
python3 scripts/claim-workflow.py history --home "$HOME/zerone-local-claims" --claim "$CLAIM_ID"
```

The history already contains the pending claim. During its COMMIT phase:

```sh
for REVIEWER in reviewer1 reviewer2 reviewer3; do
  python3 scripts/claim-workflow.py commit --home "$HOME/zerone-local-claims" \
    --claim "$CLAIM_ID" --actor "$REVIEWER" --vote accept \
    --method M-COMPUTATIONAL \
    --reason 'Trial division found primes for all checked inputs; this does not establish the broader range. Deliberately incomplete review for this synthetic exercise.' \
    --scope 'Synthetic local fixture only: n=0 through 39 inclusive; n=40 was not checked. These accounts share one controller.' \
    --evidence sha256:00b74d248602875cc4d554e95a9d327b612925bad7e2a988d7188c3a8c22b65f || break
done
```

Reveal those saved tuples when the round enters REVEAL. The first command can
wait for that phase; subsequent commands use the same saved inputs:

```sh
for REVIEWER in reviewer1 reviewer2 reviewer3; do
  python3 scripts/claim-workflow.py reveal --home "$HOME/zerone-local-claims" \
    --claim "$CLAIM_ID" --actor "$REVIEWER" --wait-for-phase || break
done
```

After aggregation, inspect the original claim in the page or `history` again.
Three accepts produce an accepted verdict and a derived Fact. Copy the actual
Fact ID from `record.facts[].fact.id` and submit the counterexample:

```sh
FACT_ID='replace-with-the-derived-fact-id'
python3 scripts/claim-workflow.py challenge --home "$HOME/zerone-local-claims" \
  --fact "$FACT_ID" \
  --content 'Synthetic counterexample: at n=40, n*n+n+41 is 1681 = 41*41, so the stated universal primality claim is false.' \
  --reason 'The original reviews stopped at 39. Direct integer calculation at 40 gives a composite value; see docs/examples/local-claims/counterexample.json in this source checkout.' \
  --evidence sha256:b16ced168dc69be4c09d1cdebc804254abc7fa193ce49eda6299772dabdd28ed
```

This command uses the existing **SubmitContradiction** message. Its returned
`claim_id` identifies a counterclaim, which has its own review round. The
original claim's history now joins the counterclaim, its exact reason and
ordered evidence IDs. Review that counterclaim using the same commit/reveal
commands, with the counterclaim ID, the counterexample evidence, and the scope
“n=40; exact integer multiplication.” Acceptance can create a counter-Fact and
a canonical contradiction edge. It does **not** automatically delete or mark
the original Fact disproved; inspect the retained statuses and both arguments.

For an inconclusive example, submit another distinct claim after the submission
cooldown, then use two `accept` votes and one `reject` with their actual reasons.
Do not change a review merely to reach a threshold. The native integration test
exercises this result as well as the accepted-claim/counterclaim exchange.

## Persistence and limits

The workflow keeps its transaction journal and private review tuples under
`<home>/claim-workflow`. It stores the signed transaction hash before broadcast,
then checks for a committed execution result. Mempool acceptance is not success.
An unresolved transaction remains unresolved and prevents another write until
it is reconciled; do not repeat a raw send to resolve a timeout. `snapshot`
queries pending hashes. If a transaction was prepared but never sent, the
explicit `retry --txhash HASH --home HOME` command checks its current result,
then can rebroadcast only the exact retained signed bytes. It never signs a
replacement transaction or changes a saved review.

Each transaction currently uses an explicit development gas limit of 2,000,000
and fee of 2,000,000 uzrn. These generous fixed fees are separate from the
queried claim review fee or the contradiction deposit (11,000,000 uzrn by
default). Receipts expose actual gas use and signed byte length. The helper
keeps at most 100 transaction attempts and 100 review tuples; the viewer bounds
the complete response to 8 MiB. This is a small development workbench. A shared
service will need measured query/index and retention decisions as usage grows.

Ctrl-C stops the viewer. Ctrl-C in the node terminal stops that owned node and
preserves its state. Use the same `start` and `serve` commands to return. The
retained binary, genesis and local identity are checked on restart. This helper
does not upgrade an existing chain or connect it to `zerone-1`.

ClaimHistory joins stored claims, reviews, derived Facts and directly linked
challenges. It reports missing historical records explicitly and has bounded
query costs; it is not a proof of completeness or independently verified block
inclusion. See [the record specification](specs/knowledge-claim-records-v1.md).
The local viewer does not fetch arbitrary evidence URLs. The test keyring is
for development; review production custody before an actual launch.
