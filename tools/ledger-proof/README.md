# Offline ledger proof verifier

This Go command reads saved, read-only CometBFT RPC responses from `zerone-dev-1` or
`zerone-1` and prints one JSON report on standard output. It has two commands:

- `transfer` checks that a single-signer `/cosmos.bank.v1beta1.MsgSend` of `uzrn` was
  included in a block, was signed by its sender, and, when asked, executed.
- `state` checks an account's balance or account record at a height against the
  application hash in the next signed header.

It does not sign, broadcast, query a network, read a keyring or write a file. It uses this
repository's pinned Cosmos SDK and CometBFT dependencies. It generalises
`tools/research-checkpoint`, whose input rules it keeps.

## Trust

Every report has an `attested_by` object: the label, the words that go with it, the pinned
validators and the repository path the pin was taken from. The pins are compiled into
`policy.go`:

| Chain | Pinned validator address | Voting power | Pin taken from |
| --- | --- | --- | --- |
| `zerone-dev-1` | `E85D65FC900DEFEB9FF160F1E6EEA2CA00768420` | 1000000 | key: `tools/research-checkpoint/verify.go`; power: `dashboard/public/research/checkpoints/2026-09-14-demimetric-v1/chain/validators.json` (height 183706) |
| `zerone-1` | `93DBC1A1D9A07E67423B193395C9EB6F1E41B844` | 11111 | `deploy/legacy-observer/checkpoint/main.go`, lines 32-34 |

Both entries carry the label `ONE_OPERATOR`. Its words are: one operator's validator key
signed these headers; this shows what that operator's chain recorded; it is not
independent confirmation and not a promise of finality. A valid signature from that key is
everything the proofs rest on.

A header at height H is trusted only when all of these hold:

- its chain ID is the policy's chain ID;
- the `/validators` response for H lists exactly the pinned validators, with the pinned
  keys and powers and nothing else;
- the pinned set hashes to the header's validators hash, and the header's proposer is in it;
- the header hashes to the commit's block ID, that block ID is well formed, and every
  commit signature's timestamp is one a protobuf timestamp can carry;
- every signature in the commit verifies, and the signatures for the block carry more than
  two thirds of the pinned power.

No flag, environment variable or input file can replace or extend the table. A chain ID
that is not in it is a usage error. If the validator set or any validator's voting power
changes, every header from then on is refused until a reviewed change to `policy.go`
updates the pin. Tests pass their own disposable policies through a function parameter
that the command line cannot reach.

The policy pins validator keys and powers, not a genesis digest. Each check stands on one
header and its commit, and no header is linked back to genesis, so a genesis digest would
bind nothing here. `tools/research-checkpoint` reads the `zerone-dev-1` genesis to find the
key; this command reuses that key directly.

## `transfer`

Save the complete JSON-RPC envelopes from these **read-only** requests:

| Input flag | RPC request |
| --- | --- |
| `--tx` | `/tx?hash=0x<TXHASH>&prove=true` |
| `--commit` | `/commit?height=<H>` |
| `--validators` | `/validators?height=<H>&page=1&per_page=100` |
| `--block-results` | `/block_results?height=<H>` |
| `--next-commit` | `/commit?height=<H+1>` |
| `--next-validators` | `/validators?height=<H+1>&page=1&per_page=100` |

The last three go together. `--require-execution` makes them mandatory. Then state what
you expect:

```sh
go run ./tools/ledger-proof transfer --chain zerone-1 \
  --tx tx.json --commit commit.json --validators validators.json \
  --block-results block-results.json \
  --next-commit next-commit.json --next-validators next-validators.json \
  --require-execution \
  --txhash "$TXHASH" --height "$HEIGHT" \
  --sender "$SENDER" --recipient "$RECIPIENT" --amount 250 \
  --account-number "$ACCOUNT_NUMBER" --pay-digest "$REQUEST_SHA256"
```

`--amount` is a positive integer of `uzrn` without leading zeros. Give at most one of
`--memo` (the exact memo, which may be empty) and `--pay-digest` (64 lowercase hex; the
memo must then be exactly `zr1:pay:<digest>`). The account number is needed to rebuild the
signed document; use the `state` command to prove it.

The report has these claims:

- **`inclusion`**: the transaction bytes hash to `--txhash`; the Merkle proof binds them to
  the header's data hash; the header at H is trusted. The proof does not bind the
  transaction's position: the same leaf and sibling hashes can also verify under another
  index and transaction count. So the index and count appear only under `execution`, once
  the committed results have fixed the count.
- **`authorisation`**: the `--tx` bytes hash to `--txhash`; they decode as a `TxRaw`
  whose canonical re-encoding is byte-identical; the body has exactly one `MsgSend` from
  `--sender` to `--recipient` moving exactly one coin of `uzrn` in the stated amount; the
  one signer has a secp256k1 key in `SIGN_MODE_DIRECT` whose address is `--sender`; the
  signature verifies over the document built from the body bytes, the auth-info bytes, the
  chain ID and `--account-number`. A given `--memo` or `--pay-digest` must match.
- **`execution`**: the block results at H rebuild to the last-results hash in the header at
  H+1, using only the fields CometBFT commits (code, data, gas wanted, gas used); their
  count equals the proof's transaction count; the header at H+1 is trusted, names the
  block at H as its last block, and carries the validators hash the header at H named as
  next; the result at the transaction's index is read, and the `/tx` response's result
  must agree with it. The `execution` facts then give `tx_index` and `tx_count`.
- **`transfer_effect`**: proven only when the three claims above are proven and the
  committed result code is zero.
- **`events`** and **`logs`**: always `NOT_PROVEN`. CometBFT 0.38 commits neither.

Without the H+1 inputs, the report adds `endpoint_result`: the code the `/tx` response
states, marked `endpoint_observation_only` because no signed header binds it. A non-zero
code there makes the command exit 1, though inclusion and authorisation hold.

Authorisation is refused as unsupported, with the reason in the report, for: more than one
message; any message type but `MsgSend`; more coins, or another denom; extension options;
unordered transactions and timeout timestamps; tips; more than one signer or signature;
multisig signers; any sign mode but
`SIGN_MODE_DIRECT`, amino-json included; any key type but secp256k1; non-canonical
protobuf, and unknown fields in the message or public-key wrappers; a memo that is not
valid UTF-8.

These are reported, not judged: the fee coins, gas limit, fee payer, fee granter, timeout
height, memo and signer sequence. A fee payer other than the sender is reported as
written. The chain also needs that payer's signature, and a transaction carrying a second
signature is refused above.

The memo is also classified by a closed grammar:

| `memo_class` | Memo |
| --- | --- |
| `none` | empty |
| `pay` | `zr1:pay:<64 lowercase hex>` |
| `anc` | `zr1:anc:<32 lowercase hex stream>:<decimal counter from 1, no leading zeros, that fits an unsigned 64-bit integer>:<64 lowercase hex head>`, only on a transfer of exactly 1 `uzrn` from an account to itself |
| `other` | anything else, near misses included |

The class reads the signed memo. It says nothing about what a digest or head commits to.

## `state`

Save these **read-only** responses, where H is the height you want the value at:

| Input flag | RPC request |
| --- | --- |
| `--query` | `/abci_query?path="/store/<store>/key"&data=0x<KEY>&height=<H>&prove=true` |
| `--next-commit` | `/commit?height=<H+1>` |
| `--next-validators` | `/validators?height=<H+1>&page=1&per_page=100` |

The command derives the store and key from `--kind` and `--address` itself and refuses a
response for any other key or store. The layouts are the same in cosmos-sdk v0.50.15 (the
line `zerone-1` runs) and v0.53.8. Both were read from the SDK source, so the policy
entries do not name a line:

| `--kind` | Store | Key |
| --- | --- | --- |
| `balance` | `bank` | `02` ‖ one byte giving the address length ‖ address bytes ‖ denom bytes |
| `account` | `acc` | `01` ‖ address bytes |

`zeroned debug addr <address>` prints an address's bytes as hex. For a 20-byte address
and `uzrn`, the balance key is `0214` + the 40 hex characters of the address + `757a726e`.
The report repeats the derived key as `key_hex`.

```sh
go run ./tools/ledger-proof state --chain zerone-1 \
  --query query.json --next-commit next-commit.json --next-validators next-validators.json \
  --height "$HEIGHT" --kind balance --address "$ADDRESS"
```

`--denom` defaults to `uzrn` and applies only to `--kind balance`. `--address` is
lowercase bech32 with the prefix `zrn`, 20 or 32 bytes.

The query response must have code 0, the height H and the derived key. It must carry
exactly two proof operations: an `ics23:iavl` proof for that key, then an `ics23:simple`
proof for the store's root. Both run through the store's standard proof runtime
(`cosmossdk.io/store/rootmulti.DefaultProofRuntime`) against the app hash in the trusted
header at H+1. With a value, the operations prove that the key holds it. Without one, they
prove that the key is absent. The proof format is the same in `cosmossdk.io/store` v1.1.1
(the older line) and v1.1.2.

The claims are `app_hash` and `stored_value`, then:

- **balance**: `x/bank` stores a non-zero balance as decimal text and deletes a zero one.
  A proven absence is therefore reported as a proven balance of `0` with
  `"stored": false`. A stored value that is not canonical decimal text, such as an older
  coin encoding, is `NOT_PROVEN`.
- **account**, **account_number**, **sequence**: a plain
  `/cosmos.auth.v1beta1.BaseAccount` that names `--address` gives its address, account
  number and sequence. A proven absence gives `"exists": false`, and number and sequence
  are `NOT_APPLICABLE`. Any other account type is reported by its type URL, with account
  number and sequence `NOT_PROVEN`.

The SDK refuses proof queries at height 1 or below. A node can prune old heights, so save
the state proof and the header at H+1 promptly.

## Report and exit status

The report's `schema` is `zerone.ledger-proof/transfer/v1` or
`zerone.ledger-proof/state/v1`. Each claim is `PROVEN`, `NOT_PROVEN` or `NOT_APPLICABLE`,
with a short reason when it is not proven. A claim reads `PROVEN` only when the code has
checked it. The facts beside the claims (`transaction`, `block`, `execution`, `header`,
`balance`, `account`) are what the evidence says, and are worth only as much as the claims
over them. `endpoint_result` is an observation and proves nothing. When authorisation
fails, `transaction` holds only the hash and length. The report also echoes what you
expected and records the SHA-256 of each input file.

| Exit | Meaning |
| --- | --- |
| 0 | Every proof asked for holds: for `transfer`, inclusion and authorisation, plus execution and its effect when the H+1 inputs are given, or, without them, a `/tx` response that states code 0; for `state`, the app hash, the stored value and every claim for the kind. |
| 1 | A proof does not hold, or without the H+1 inputs the `/tx` response states a non-zero code (the report is still printed in both cases); or an evidence file was refused (no report). |
| 2 | Usage error: a missing, repeated or malformed flag (boolean flags included), or a chain without a compiled policy. |

## Evidence files

Each input must be a regular, nonempty UTF-8 JSON file of at most 8 MiB holding a JSON-RPC
2.0 success envelope. Final-component symlinks, FIFOs, devices, directories, duplicate
object keys, trailing JSON values and nesting deeper than 64 levels are refused, as in
`tools/research-checkpoint`. The envelope may hold only the members `jsonrpc`, `id`,
`result` and `error`, by those exact names, so a member such as `Result` cannot stand in
for `result`. SDK CLI output is not a substitute for these RPC envelopes.
Keep the inputs with the report so anyone can rerun it. A report on its own is not a new
trust anchor.

## Limits

- Single-key secp256k1 signers and `MsgSend` only. No multisig. One message per
  transaction.
- Statements (a balance reconciled over a range of heights) and anchor-history checks are
  not built yet.
- On `zerone-1` (the 2e37c4c lineage), a block that carries a transaction also credits its
  producer. This tool does not account for such credits.
  [`docs/tokenomics/GENESIS.md`](../../docs/tokenomics/GENESIS.md) describes that
  block-reward lane and the upgrade that retires it.
- Without the H+1 inputs, a transfer's index in its block and the block's transaction
  count are not proven, and the result code is only the endpoint's statement.
- If a block holds the same transaction bytes more than once, the index, the result code
  and `transfer_effect` describe the occurrence the `/tx` response names.
- It reads one commit per height and does not look for a second, conflicting commit.
- The validator set must fit one `/validators` page of 100.
- A proof shows what the pinned key signed. It does not show that a chain is the only one,
  or that a recorded value will not be reversed.

## Focused checks

```sh
go test ./tools/ledger-proof -count=1
go vet ./tools/ledger-proof
```

The tests create disposable signing identities, blocks and an in-memory IAVL multistore.
Two checks also read, offline, evidence this repository already publishes for
`tools/research-checkpoint`: the `zerone-dev-1` pin is compared with the validator
response it cites, and that published transfer is verified end to end under the compiled
`zerone-dev-1` pin, so real RPC responses are tested as well as ones the tests encode. No
test adds chain data of its own. The store is written with the SDK's own key and value
codecs and read with this tool's hand-derived keys, so a layout difference fails the
tests. They cover the valid paths; the two-thirds power threshold; one invalid signature
beside enough valid power; a commit by a key outside the policy; an extra, missing,
re-weighted or replaced validator, or a response that leaves validators to another page; a
header that names another validator set; a malformed commit block ID; evidence under
another chain's policy; altered transaction bytes, proofs, indexes, totals and heights; a
proof relabelled to another index and count; every refused transaction shape above, with
its reason; a wrong account number; each memo class and near misses of its grammar; a
non-zero result code, committed or stated by the endpoint; block results that do not
rebuild; headers at H and H+1 that do not chain; and, for state, existence, absence,
another address, another store, an altered value, a third proof operation, a proof checked
against the wrong app hash or against the header at H, a non-base account and a stored
account with unknown fields. They also cover every input-file refusal, envelope members
that differ from the four names only in case, usage errors (a repeated boolean flag
included) and exit codes, and they check that no report marks events or logs as proven.
