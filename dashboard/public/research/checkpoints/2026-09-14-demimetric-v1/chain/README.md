# Saved chain proof: research checkpoint, 14 September 2026

The 81-entry journal was committed by transaction
`8DFC740C5326E76B482B6D370F527311CB518934D6B64A4BF5EFDBD2806D0B8E`
in `zerone-dev-1` block **183706**, whose signed header time is
`2026-09-14T16:16:57.706310948Z`.

It is a 501-byte signed self-send of `1uzrn`; the fee is `2000000uzrn`,
gas limit `2000000`, gas used `100584`, and execution code `0`.
The account is operator-controlled, account number 7, signing sequence 2.
This is a storage checkpoint, not a native Fact or a scientific review round.

## Reproduce offline verification

Use the Zerone source containing `tools/research-checkpoint/` and Go as pinned in
the repository. Fetching source/dependencies can require network access; the
compiled verifier performs no network requests. The following uses the packaged
files already present in a source checkout. For downloaded archives, extract the
evidence ZIP into a fresh directory and the proof ZIP into that same directory,
then set `checkpoint_dir` to that directory instead.

```sh
checkpoint_dir=dashboard/public/research/checkpoints/2026-09-14-demimetric-v1
python3 -B tools/research-review/checkpoint.py verify \
  "$checkpoint_dir/journal.json" "$checkpoint_dir/chain/checkpoint.json" \
  --chain-id zerone-dev-1 --memo 'zerone:research:v1:ce6e799a-6713-41a9-b95f-57c50a223074:81:86c8163599c0098aa43b4f3599a81b02a03557553fc595b67b026fdfe108704e:23baf5b0a2e7f4a48fe2b86775ae2ef92e607f2d793270ff10219e61d2563847'
go run ./tools/research-checkpoint \
  --genesis "$checkpoint_dir/chain/genesis.json" \
  --tx "$checkpoint_dir/chain/tx-rpc.json" \
  --commit "$checkpoint_dir/chain/commit.json" \
  --validators "$checkpoint_dir/chain/validators.json" \
  --block-results "$checkpoint_dir/chain/block-results.json" \
  --next-commit "$checkpoint_dir/chain/next-commit.json" \
  --next-validators "$checkpoint_dir/chain/next-validators.json" \
  --height 183706 \
  --txhash 8DFC740C5326E76B482B6D370F527311CB518934D6B64A4BF5EFDBD2806D0B8E \
  --memo 'zerone:research:v1:ce6e799a-6713-41a9-b95f-57c50a223074:81:86c8163599c0098aa43b4f3599a81b02a03557553fc595b67b026fdfe108704e:23baf5b0a2e7f4a48fe2b86775ae2ef92e607f2d793270ff10219e61d2563847' \
  --sender zrn1q36wcvumlxa3qrmsu8cmkrjqsa4jqfchtzkuj8 \
  --account-number 7 --gas 2000000 --fee-uzrn 2000000 \
  --require-execution-proof --output ./my-checkpoint-verification.json
```

Choose a new output filename on a repeat run; the verifier refuses overwrite.
Compare its receipt with `chain/verification.json`. `CHAIN-MANIFEST.json` checks
package integrity; its presence alone is not an independent trust anchor.

## What is authenticated

- The exact journal file matches the memo's collection, count, file hash and head.
- The account's direct secp256k1 signature binds the memo, message and fee in its
  chain-specific SignDoc. Account number is supplied signing context.
- The transaction has a Merkle inclusion proof under block 183706's signed header.
- The validator signature matches the original genesis key, not an arbitrary key
  returned by a server. The verifier pins genesis SHA-256
  `f8b4570b37fd41d5b63c02fbae1f89225c197c75315e8629650716f6e5f79256`.
- Code, data and gas are bound through the signed result root in linked block
  183707. Event/log text and balance-query observations are not covered by that root.

This is verification under the original single development validator's key.
It is not independent validator diversity, an account-state proof, full chain
replay, external scientific review, or evidence of when an idea was first held.
Retain the expected genesis pin independently when establishing trust. Neither a
valid signature nor a consensus header time establishes a trustworthy wall clock
or present canonical-chain membership after a development reset.

The research files remain outside the chain and require replication. Keep both
archives, their manifests, and the verification source. Later journal additions
do not change this frozen 81-entry snapshot. A receipt recording this checkpoint
can be appended later without attempting to include its own future transaction.
