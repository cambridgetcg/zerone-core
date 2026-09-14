# Offline research checkpoint verifier

This Go command verifies a research-journal hash checkpoint made by a single
Cosmos bank self-send on `zerone-dev-1`. It does not sign, broadcast, query a
network, read a keyring, create a Fact, or open a scientific review round.
It uses this repository's pinned Cosmos SDK and CometBFT dependencies.

The accepted transaction has exactly one `/cosmos.bank.v1beta1.MsgSend`, from the
expected signer to itself, with exactly `1uzrn`. It has one direct secp256k1
signature, an explicit expected gas limit and fee, and no fee delegation,
extensions, tip, unordered execution or timeout. Its memo is exactly:

```text
zerone:research:v1:<collection_uuid>:<entry_count>:<export_sha256>:<head_sha256>
```

The count is 1–1,000 and the memo must fit 256 ASCII bytes. UUID and digest text
is lowercase. `tools/research-review/checkpoint.py` separately validates the
entire journal and binds its exact bytes and head to this memo. The Go command
does not assert that a file was published or obtained merely because its hash
appears in a transaction.

## Trust and evidence

The command requires the original SDK genesis file with this **built-in** pin:

```text
f8b4570b37fd41d5b63c02fbae1f89225c197c75315e8629650716f6e5f79256
```

It extracts the sole initial Ed25519 consensus key from that file's validator
gentx and also checks the retained original public-key pin. There is no flag to
replace these pins. Each fetched validator response must contain exactly that
one key; its address and voting power must match the signed header's validator
hash. A self-consistent header and validator response from a different key are
refused. A future membership change needs a separately reviewed trust policy.

Save the complete JSON-RPC envelopes from these **read-only** endpoints:

| Input flag | Public RPC request |
| --- | --- |
| `--tx` | `/tx?hash=0x<TXHASH>&prove=true` |
| `--commit` | `/commit?height=<H>` |
| `--validators` | `/validators?height=<H>&page=1&per_page=100` |
| `--block-results` | `/block_results?height=<H>` |
| `--next-commit` | `/commit?height=<H+1>` |
| `--next-validators` | `/validators?height=<H+1>&page=1&per_page=100` |

The last three inputs are optional as a group. Prefer all three with
`--require-execution-proof` once the next block exists. SDK CLI output is not a
substitute for these RPC envelopes. Evidence files must be regular, nonempty
UTF-8 JSON files no larger than 8 MiB each. Final-component symlinks, FIFOs,
duplicate object keys, trailing JSON values and excessive nesting are refused.

```sh
go run ./tools/research-checkpoint \
  --genesis sdk-genesis.json \
  --tx tx-rpc.json --commit commit.json --validators validators.json \
  --height "$HEIGHT" --txhash "$TXHASH" \
  --memo "$MEMO" --sender "$SENDER" --account-number "$ACCOUNT_NUMBER" \
  --gas 2000000 --fee-uzrn 2000000 \
  --block-results block-results.json \
  --next-commit next-commit.json --next-validators next-validators.json \
  --require-execution-proof --output verification.json
```

Expected values come from the separately retained checkpoint and signing
context, not by blindly copying whatever the untrusted proof claims. The
account number is needed to reconstruct the signed Cosmos SignDoc. Successful
signature verification binds that context; this command does not independently
prove account-state membership. Gas and fee above reflect the advertised
development profile, not a universal minimum or future fee quote.

## What a successful receipt proves

The verifier checks the exact transaction hash, transaction Merkle proof against
the header's `DataHash`, the header hash against the commit's BlockID, and the
commit signature against the original pinned validator. It checks the complete
transaction intent, signer-address derivation and direct sender signature under
the chain ID and supplied account number. Canonical protobuf re-encoding and
explicit unknown-field checks on the two accepted Any wrappers refuse ambiguous
or unsupported encodings. This is intentionally a narrow checkpoint format, not a
general Cosmos transaction verifier.

With the optional proof, it verifies the original-key-signed next header, its
link to this block, and all supplied deterministic transaction results against
`H+1.LastResultsHash`. The matching result index must have code zero. CometBFT
commits `Code`, `Data`, `GasWanted` and `GasUsed`; it does **not** commit logs,
codespace or events in that root. Those fields are not represented as proved.
Without these inputs, the receipt explicitly labels code and gas as
`endpoint_observation_only`; transaction inclusion alone does not prove success.

The new receipt is mode 0600 and is never allowed to overwrite an existing path.
It records hashes of the input bytes and the precise proof scope. Keep the inputs
as well as the receipt so someone else can rerun the verifier. A receipt by itself
is not a new trust anchor. A failed output write can leave a partial file, which
must not be represented as a successful invocation.

This single-operator development anchor does not provide independent consensus,
prove custody, guarantee trustworthy time, establish current canonical-chain
membership after a reset, or replay the application. A checkpoint is the
signer's record of a hash, not an endorsement of every attributed author or a
scientific verdict. Keep the journal and its evidence retrievable separately;
RPC, block and application-history retention are not permanent guarantees.

## Focused checks

```sh
go test ./tools/research-checkpoint
go vet ./tools/research-checkpoint
```

Tests create disposable in-memory signing identities and blocks. They cover
valid inclusion and result-root binding, wrong trust anchors, altered bytes and
proofs, altered sender context and intent, uncommitted result fields, ambiguous
JSON/protobuf encodings, special-file refusal and receipt no-overwrite behavior.
