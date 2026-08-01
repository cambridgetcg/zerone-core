# Validator recovery gate

`validator-recovery-gate` is a standard-library-only, offline verifier for the
decision that precedes validator recovery after a custody or hostile-runtime
incident. It does not move stake, sign transactions, read private key files,
compile genesis, release quarantine, or perform an upgrade.

Its output is one deterministic v2 report:

- `GO_CONTROLLED_TRANSITION`
- `GO_FORK_REGENESIS`
- `NO_GO`

A GO report is journal evidence, not authority to mutate a chain.

## Fail-closed decision model

The signed custody assessment selects the route. An operator cannot override
that route.

| Valid custody facts | Required route | Possible result |
| --- | --- | --- |
| Every required finding and privileged identity is `PASS`; no identity is marked `RETIRE` | `CONTROLLED_TRANSITION` | `GO_CONTROLLED_TRANSITION` |
| Any required finding is missing, `FAIL`, or `UNKNOWN`; or an identity needs retirement | `FORK_REGENESIS` | `GO_FORK_REGENESIS` only if the audited profile can repair it |
| Invalid assessment, wrong route material, unsupported rewrite, or missing evidence | fail closed | `NO_GO` |

The only implemented fork profile is `consensus-key-only`. It retains exactly
one independently proven `PASS/RETAIN` SDK operator and replaces its consensus
and node identity. It cannot repair ambiguous canonical history, an unsafe
governance authority, an unsafe SDK operator, or any retired privileged
identity. Those cases remain `NO_GO` until a separate compiler profile and
policy receive independent review.

## Normative schemas

The Go structures in [`schema.go`](schema.go) define field order and the exact
JSON contract:

- `zerone.ops.validator-signer-policy/v2`
- `zerone.ops.validator-custody-assessment/v2`
- `zerone.ops.validator-controlled-transition/v2`
- `zerone.ops.validator-fork-policy/v2`
- `zerone.ops.validator-fork-release/v2`
- `zerone.ops.validator-fork-choice/v2`
- `zerone.ops.validator-recovery-gate-report/v2`
- `zerone.fork-genesis.report/v1`

Inputs must be exact compact bytes produced by Go `encoding/json.Marshal` for
the relevant structure: no whitespace, newline, unknown field, duplicate key,
trailing value, or `null` array where `[]` is required. Hashes and public keys
are lowercase hexadecimal unless a field explicitly requires another
encoding.

Every input has an external exact-file SHA-256 pin. A document self hash never
replaces that independent pin.

The assessment, controlled plan, fork release, fork choice, compiler report,
and gate report are self-hashed. Clear the final self-hash field, marshal the
whole document, and SHA-256 those bytes. Signer policies and the fork policy
are trust roots with no self hash; the verifier receives their exact file
digests through a separate trusted channel.

## Public validator identity

`ValidatorIdentity` contains no secret material. The verifier derives rather
than trusts its public identifiers:

- `sdk_operator_address` is canonical lowercase BIP-0173 Bech32 with HRP
  `zrnvaloper` and exactly 20 payload bytes;
- `consensus_public_key` and `node_public_key` are distinct 32-byte Ed25519
  keys encoded as lowercase hex;
- `consensus_address` is uppercase hex of the first 20 bytes of
  `SHA-256(consensus_public_key)`;
- `node_id` is lowercase hex of the first 20 bytes of
  `SHA-256(node_public_key)`;
- validator-key, node-key, and signing-state file digests are pairwise
  distinct and each resolves to exactly one typed evidence entry.

Freshness is cross-role. A new node key cannot reuse an old consensus key, a
new consensus key cannot reuse an old node key, and no new key-file digest can
equal any old key-file digest.

## Pinned approval authority

Custody and controlled approvals are authorized by separate, exact-pinned
`SignerPolicy` files. Each policy binds:

- purpose;
- incident and chain;
- the assessment digest for controlled operation;
- exact signer tuples `(role, identity, control_domain, public_key)`;
- minimum approvals, identities, and control domains;
- the exact mandatory role set.

The verifier never learns authority from the approvals themselves. A valid
signature from an unlisted key is rejected. Old consensus and node keys, and
identities marked `RETIRE`, cannot approve recovery.

Fork release and choice approvals use the independently pinned `ForkPolicy`.
That policy also pins exactly two independent genesis reproducers. Signers and
reproducers must have distinct identities, control domains, and Ed25519 keys.

Approval statements are domain separated. For a document approval:

1. Replace `approvals` with `[]` and its self-hash with `""`.
2. Marshal and SHA-256 that body.
3. SHA-256:

   ```text
   domain || body_digest ||
   0x00 || role ||
   0x00 || identity ||
   0x00 || control_domain ||
   0x00 || public_key_bytes
   ```

4. Sign the 32 statement-digest bytes with Ed25519.

The exact domains are constants in [`schema.go`](schema.go).

## Evidence closure

Every custody finding and privileged-identity assessment digest must resolve
to exactly one correctly typed `Evidence` entry. The same is true for:

- checkpoint block ID, app hash, signed commit, and validator set;
- old and new validator key files, node key files, and signing state;
- power snapshot and every stake-inventory page;
- binary, image, provenance, SBOM, rehearsal, topology, and journal head;
- source export, rewrite tool, rewrite policy, selected genesis, and both
  exact compiler report files;
- supply, IBC, and module reconciliation artifacts.

An arbitrary 64-hex label without an exact evidence link cannot open the gate.

## Controlled-transition contract

A controlled GO additionally requires:

- custody evaluation strictly after the signed checkpoint time;
- the exposure review to end exactly at checkpoint height;
- admission exactly `OPEN`;
- bond height strictly after the checkpoint;
- activation exactly `B+2`;
- consenting power strictly greater than two thirds;
- complete delegation, unbonding, and redelegation pagination with empty next
  keys;
- fresh validator identity and key-file evidence;
- exact pinned controlled signer policy and all mandatory approvals.

## Fork/re-genesis contract

A fork GO consumes the exact selected genesis and two exact
`zerone.fork-genesis.report/v1` files. It requires:

- new chain revision greater than the old revision;
- initial height exactly checkpoint `H+1`;
- rewrite profile exactly `consensus-key-only`;
- old SDK operator exactly `PASS/RETAIN`;
- no retired privileged identity;
- exactly one bonded, unjailed retained operator with a fresh consensus key;
- consensus power equal to staking tokens divided by Cosmos
  `DefaultPowerReduction` (`1,000,000`);
- exactly two trusted, distinct reproduction signatures.

Each reproduction signature is domain separated and binds the release inputs,
selected genesis digest, reproducer tuple, and exact compiler-report file
digest. The reports must be semantically identical except for reproducer
fields and their self hashes.

The verifier mirrors the compiler contract instead of trusting report labels:

- report subject, checkpoint, source export, rewrite policy/tool, custody
  assessment, fork policy, old/new keys and addresses, output genesis, and
  exact report-file digests must cross-bind;
- report module keys must exactly equal `app_state` module keys;
- every reported after-hash must equal canonical JSON from the exact selected
  module;
- emergency, slashing, and staking rewrites are mandatory;
- IBC and transfer change exactly when their allowed v8-to-v10 migration is
  declared; a current v10 export correctly emits `schema_migrations:[]`;
- modules outside the audited change set remain byte-semantically unchanged.

The selected genesis itself must:

- have `app_hash:null`, the new chain ID, time after the checkpoint, and
  initial height `H+1`;
- contain exactly one matching consensus validator and one matching bonded,
  unjailed staking validator;
- contain none of the old consensus key/address in any of the compiler's seven
  canonical encodings;
- start emergency state as `halted` with
  `legacy-genesis-quarantine` at `H+1`, no release block, and no preloaded
  recovery authorization;
- contain no genesis transactions or pending evidence;
- have empty SDK and Zerone governance work and an empty `upgrade` object;
- have empty IBC v10 client, connection, channel, packet, transfer escrow,
  interchain-account, fee, and rate-limit state.

Fresh-operator forks, governance rewrites, non-empty IBC migration, economics
rewrites, and automatic quarantine release are deliberately unsupported.

## Gate report verification

`report_sha256` proves only envelope integrity. A self-hashed GO can be forged
by anyone.

Code consuming a report must use `verifyGateReportWithInputs`, which validates
the envelope, re-evaluates the exact pinned source documents, and requires a
byte-for-byte identical report. `validateGateReportEnvelope` is diagnostic
only and is explicitly not an authorization check.

GO reports reject inactive-route inputs: controlled reports cannot carry fork
digests, and fork reports cannot carry controlled-plan digests.

## CLI

Build from the repository root:

```sh
go build -o validator-recovery-gate ./tools/validator-recovery-gate
```

Controlled route:

```sh
./validator-recovery-gate evaluate \
  --chain-id zerone-1 \
  --incident-id incident-2026-001 \
  --custody-policy custody-policy.json \
  --custody-policy-sha256 "$CUSTODY_POLICY_SHA256" \
  --assessment assessment.json \
  --assessment-sha256 "$ASSESSMENT_SHA256" \
  --controlled-policy controlled-policy.json \
  --controlled-policy-sha256 "$CONTROLLED_POLICY_SHA256" \
  --controlled controlled-transition.json \
  --controlled-sha256 "$CONTROLLED_SHA256"
```

Fork route:

```sh
./validator-recovery-gate evaluate \
  --chain-id zerone-1 \
  --incident-id incident-2026-001 \
  --custody-policy custody-policy.json \
  --custody-policy-sha256 "$CUSTODY_POLICY_SHA256" \
  --assessment assessment.json \
  --assessment-sha256 "$ASSESSMENT_SHA256" \
  --fork-policy fork-policy.json \
  --fork-policy-sha256 "$FORK_POLICY_SHA256" \
  --fork-release fork-release.json \
  --fork-release-sha256 "$FORK_RELEASE_SHA256" \
  --fork-choice fork-choice.json \
  --fork-choice-sha256 "$FORK_CHOICE_SHA256" \
  --genesis genesis.json \
  --genesis-sha256 "$GENESIS_SHA256" \
  --compiler-report-a report-a.json \
  --compiler-report-a-sha256 "$REPORT_A_SHA256" \
  --compiler-report-b report-b.json \
  --compiler-report-b-sha256 "$REPORT_B_SHA256"
```

Exit codes:

- `0`: a GO report was emitted;
- `1`: deterministic `NO_GO`, or trusted chain/incident mismatch;
- `2`: usage, unsafe file, pin, or exact-JSON error.

The CLI accepts only non-symlink regular files opened with no-follow,
non-blocking flags. Limits are 1 MiB for ordinary documents, 4 MiB per compiler
report, and 256 MiB for genesis. Common key-file paths and secret-bearing JSON
fields are refused, and errors never echo document values.

## Verification

```sh
go test ./tools/validator-recovery-gate
go test -race ./tools/validator-recovery-gate
go vet ./tools/validator-recovery-gate
```

The suite includes valid controlled, v8-migration, and current-v10 fork
fixtures plus adversarial cases for lying derived addresses, cross-role key
reuse, unlinked evidence, untrusted policy signers, forged self-hashed GO
reports, unsigned and same-key reproductions, random report pins,
report/genesis disagreement, module-hash substitution, pending upgrade,
non-empty IBC, genesis transactions, jailed validators, missing quarantine,
old-key residue, non-canonical JSON, symlinks, and oversized files.
