# ToK feedback boundary: local rehearsal and release gates

**Default: NO_GO. Source only.** This packet does not schedule an upgrade, clear
custody, authorize a production key, or select a reset/successor chain. Historical
1/1 images are SUSPECT; no independent custody assessment is supplied. There is
no accepted candidate commit or executable digest yet. No full historical-binary
handoff has been executed for this packet. Do not edit those facts into success
because a keeper/handler test passes.

## Freeze the whole consensus delta

H1 `65c19cd8b00bdfff9b80705b776fd0d49719398a`; H2
`36728afbf71905a077a0863b41536fa9279109dd`; H3
`335bb94f0fd54d3752dcb397263b7e84fb1116b4`, tree
`769f67f1cfa108be3d31cace7777cf954f731c42`. Obtain independently reviewed source,
build recipes, executable digests and provenance for **each** boundary. A source
hash in plan info does not prove what binary produced a chain state. H1/H2/H3
cannot be performed by the candidate; its loader/handler refuse that shortcut.

Review `git diff <accepted-H3> <frozen-candidate> -- app x proto go.mod go.sum`
and the entire remaining source delta. The candidate base already carries
post-H3 ABCI, export, authentication/key-rotation/admission and query changes,
some without module-version bumps. See `release.json` for the explicit review
scope. Do not narrow review to the ToK patch or infer source acceptance from a
full module version map alone. Keep new use/rating fitness and energy zero once
reports exist; disabling/pruning never clears `EverReported`.

## Prepare a local fixture, not a production state copy

Run `rehearse.py` only with separately generated disposable local keys, a chain
ID beginning `tok-feedback-rehearsal-`, and no peers, seeds, PEX, remote signer,
public API or public gRPC. Never import a live/default home or a production key.
Path confinement and a `synthetic_fixture_only` flag do **not** prove key origin;
independently inspect the fixture's creation provenance before execution.

First execute the exact H1→H2→H3 binaries on that local ledger through real named
upgrade halts and commits. Use existing reviewed per-boundary ceremony tools;
this packet does not generate genesis markers, fake completion history, skip
upgrades or reset a ledger. Preserve the real H1/H2/H3 ordered done heights,
H2 plan-identity hash, H3 `migrated-with-loader-proof-v1` marker and exact full
post-H3 module map. Native H3 genesis is not a substitute.

Under that **old H3** binary, schedule the local feedback plan using the standard
SDK upgrade path in a separately authorized local fixture preparation. This
driver has no transaction commands. Bind its Info exactly to `release.json`'s
`plan_info`, then clean-stop safely before H−1 (allow several blocks to observe
the fixture). Copy only this stopped local fixture into a new unique directory
under `local-rehearsals/`. Record every file digest only after clean stop. Never
retry by deleting prior evidence; use another directory.

Input manifest fields (`manifest.json` in that unique directory):

- `schema`: `zerone.tok-feedback-v1/local-rehearsal/v1`
- `synthetic_fixture_only`: true, only after provenance inspection
- `candidate_consensus_delta_reviewed`: true, only after actual review
- `h3_source`, `h3_tree`: exact pins above
- `old_binary`, `new_binary`: distinct absolute executable paths inside the directory
- `old_binary_sha256`, `new_binary_sha256`: independently checked 64-hex digests
- `stopped_fixture_home`: absolute local fixture home inside the directory
- `stopped_fixture_files_sha256`: relative-file-path → SHA256 map, no symlinks
- `chain_id`: exact local fixture genesis chain ID
- `upgrade_height`: integer > 5 matching its already committed plan
- `ports`: six distinct free ports 20000–60000, old/new/restored RPC+P2P pairs

Use Python 3.11 or later. Default invocation validates without starting anything:

```sh
python3 deploy/upgrades/tok-feedback-v1/rehearse.py \
  /absolute/path/to/deploy/upgrades/tok-feedback-v1/local-rehearsals/run/manifest.json
```

Only after reviewing those inputs, append `--execute-local`. This:

1. Acquires an exclusive local driver fence; copies the stopped fixture to a
   separate old home; starts only the supplied old binary on loopback ports.
2. Observes H−1 and waits for the old binary's actual upgrade halt. Missing halt,
   wrong plan, advance past H−1 or dirty stop is failure, not successful handoff.
3. Makes a stopped H−1 backup, verifies copied file hashes, and starts the exact
   new binary from a separate copy with distinct ports and the old process dead.
4. Observes H and H+1. H's app hash is read from H+1's block header.
5. Clean-stops new, copies its POST-H state to another home, verifies all files,
   restarts **new** from the cold copy and checks the retained H app hash. Never
   starts old on post-H state. The fixture and H−1 backup remain untouched.

The result stays NO_GO and records local evidence only. Cold reopening committed
blocks is not independent deterministic block replay. To test determinism,
replay identical block/transaction/time inputs into two independently restored
H−1 databases; freshly proposed blocks have different times and need not have
equal app hashes. That independent replay is a separate acceptance obligation.

## Required H−1/H negative and positive acceptance

- Old H3 must halt before H; candidate must refuse H1/H2/H3 homes/plans and a
  skipped upgrade, malformed/missing markers, native H3, wrong/missing/extra module
  versions, early V7 and divergent local/on-chain plan info. Do not use a
  source-only handler call as executable-handoff evidence.
- Candidate startup at actual H−1 must pass both the unchanged H3 validator and
  the additional ToK validator. `verify-activation-prestate --home <stopped-local-copy>
  --expected-chain-id <local-chain> --expected-height <H-1>
  --expected-app-hash <independently-observed-lowercase-hex>` has a separate
  post-H3 root-verification/cache dry-run path. It intentionally remains `activation_ready:false` because it
  cannot prove custody or historical binary provenance.
- At H commit, verify V7, the named upgrade marker, module migration marker and
  done height; defaults disabled, cohort empty, limits 100/1,000, retained cap
  2,000; no legacy receipt promotion or zero-as-unlimited interpretation.
- Record same-height total supply, every module balance, IBC clients/channels,
  retained obligations, facts/relations/status counters (including gaps),
  cascades/rounds/actual completion metadata, receipts/rating/expiry/cursor/latch.
  Compare H−1→H changes to the exact reviewed migration diff. Do not invent absent
  old history. Test graph/receipt export→import→cold-open equality independently.
- Run generated signing→transport→CheckTx→FinalizeBlock/Commit→query acceptance,
  semantic replay, bad signatures, quotas, storage-failure atomicity, closed
  epochs, all challenge outcomes, supply invariants and retained-cap gas tests.
- Crash/stop at H−1 and restore; after H use forward repair only. Keep only one
  validator process or signer live at once. If SIGTERM fails, the driver leaves
  a failure and does not call forced termination a clean backup. Inspect and
  fence manually; never PID-kill from a truncated process listing.

## Production is a different gate

This driver accepts no production profile. A trusted production stopped-state
copy needs independent custody/provenance, signer fencing, authorization,
capacity and recovery evidence before the existing operations runbook can be
used. No suspect sole signer may attest itself clean. No genesis rewrite,
production signing, transaction scheduling, deployment or reset is performed
here. After custody and all release gates pass, execute a release-matched testnet
first; production launch claims additionally require both 1,000 committed blocks
and 60 minutes of stable monitored operation. Disclose sole-operator/f=0 limits.
