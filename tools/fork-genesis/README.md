# Deterministic fork-genesis compiler

`fork-genesis` is an offline, fail-closed compiler for the narrow recovery case
where the current consensus key must be treated as suspect but the validator
operator, accounts, balances, staking economics, and chain history remain
authoritative.

It is not a general genesis editor. The only supported profile is
`consensus-key-only`.

## Safety boundary

The compiler requires and proves all of the following:

- the source is a non-zero-height `H+1` export bound to an exact source block ID
  and app hash, last signed block time, signed commit, validator set, and export
  digest;
- the chain revision increases and the old and new Ed25519 consensus keys differ;
- the exact compiler executable, independently pinned fork policy, incident, and
  custody assessment are bound before compilation;
- exactly one exported validator is bonded, unjailed, and matches the policy's
  operator, old key, and staking power;
- the old key is replaced in Comet genesis, SDK staking, slashing state, and,
  when present, separately pinned Zerone validator metadata;
- a tombstoned signer, pending consensus evidence, governance or upgrade
  state, or live IBC state is a hard refusal;
- `genutil.gen_txs` is exactly empty, proving the input is not an initial
  genesis with transactions waiting to be replayed;
- every source application module is included in the sorted before/after
  digest report, and any change outside the exact reviewed rewrite set is a
  hard refusal;
- the target begins in consensus quarantine at `H+1`;
- at least two sorted, policy-authorized reproducer tuples exist, each with a
  distinct identity, control domain, and Ed25519 attestation key;
- the policy, input, output, and report are all SHA-256 bound.

Cosmos exports may retain the network's original launch `genesis_time`, so that
field is never used as the recovery clock. `source_last_block_time` must come
from the signed checkpoint and `target_genesis_time` must be strictly later.

The retained SDK operator is allowed only under
`operator_disposition: "RETAIN_PROVEN_SAFE"`, with the old consensus key as the
sole prohibited consensus key and no prohibited privileged identity. The
validator recovery gate must independently prove a `PASS/RETAIN` custody
assessment for that exact operator. If operator custody is missing, failed, or
unknown—as it currently is for a legacy image that may have embedded operator
authority—this compiler cannot produce a releasable recovery.

The deployed legacy chain uses IBC-Go v8 while the target code uses IBC-Go v10.
Because this profile requires IBC state to be empty, the compiler performs only
two reviewed empty-state schema migrations and records them in the report:

- `denom_traces: []` becomes `denoms: []`;
- removed v8 channel parameters are accepted only at their exact reviewed
  default, then v10 client/channel-v2 empty state is added.

Anything else is outside this compiler's authority. In particular, it cannot
replace an operator or governance account, preserve active IBC, rewrite token
economics, revive a tombstoned validator, or make a custody decision.

## Build and run

```sh
go build -o ./build/fork-genesis ./tools/fork-genesis

./build/fork-genesis \
  --input /absolute/path/source-export.json \
  --input-sha256 "$SOURCE_EXPORT_SHA256" \
  --policy /absolute/path/policy.json \
  --policy-sha256 "$POLICY_FILE_SHA256" \
  --reproducer-id reproducer-a \
  --output /absolute/new/path/target-genesis.json \
  --report /absolute/new/path/reproducer-a-report.json
```

Input paths must be canonical absolute regular files beneath real,
non-symlinked parent directories. The source export must be the exact compact
canonical SDK export JSON emitted to `zeroned export` stdout: duplicate object keys,
reformatted exports, and trailing data are refused. Output paths must not
already exist; new files are installed without replacement and synced with
mode `0600`. A successful run prints one `GO_FORK_REGENESIS` line. Exit status
`1` means a policy or recovery refusal; status `2` means usage or local I/O
failure.

The policy must be exact compact canonical JSON with no trailing newline. Set
`policy_sha256` to the SHA-256 of the same canonical object with
`policy_sha256` equal to the empty string. The CLI additionally requires the
SHA-256 of the final policy file bytes. Build the compiler first, hash that
exact executable, and bind the digest as `rewrite_tool_sha256`; the CLI hashes
its own executable and refuses a mismatch.

Run the compiler in two independent environments using different authorized
`--reproducer-id` values. The target genesis bytes and
`output_genesis_sha256` must match exactly. Reports intentionally differ by
reproducer tuple and report self-hash. A report is not proof that its named
environment actually ran the compiler: each reproducer must sign an external
attestation over the exact printed `report_file_sha256`, output genesis digest,
incident, and pinned policy using the private half of its policy-authorized
attestation key. The recovery gate verifies those signed attestations against
the independently pinned reproducer policy. Put the printed
`report_file_sha256` values—not the internal `report_self_sha256` values—into
those attestations and the release envelope.

## Release sequence

Compiler success is necessary but not sufficient to start a fork. The
cycle-free release order is:

1. seal the custody assessment;
2. independently pin the fork policy;
3. compile the policy-bound genesis in two independent control domains;
4. seal the release around both compiler reports and the matching genesis;
5. sign the final fork choice that pins the exact sealed release;
6. run the recovery gate.

The final fork choice is intentionally absent from the rewrite policy: including
it would create a hash cycle because the choice must pin the already-built
release. The recovery gate must also accept the stopped-home manifest,
deterministic rehearsal report, fresh-volume evidence, topology evidence, and
signed decision journal. Keep the old signer stopped and quarantined
throughout.

See [policy.schema.json](policy.schema.json) and
[report.schema.json](report.schema.json) for the machine-readable envelopes.
