# Validator home manifest

`validator-home-manifest` is a standard-library-only local integrity gate for
moving a stopped Zerone validator home from a read-only isolated snapshot to
an empty volume on a different filesystem device. It commits to exact bytes
without printing private keys, signatures, or sign bytes.

The manifest binds:

- canonical source and destination paths, device IDs, and root inodes;
- the external destination-volume ID and evidence digest;
- chain ID, initial height, genesis digest, last height, and AppHash;
- seed-derived Ed25519 consensus and node identities;
- strict Comet signing-state coordinates and signature/sign-byte digests;
- exact file sets for `application.db`, `blockstore.db`, and `state.db`;
- every regular file and directory under the validator home;
- stopped-process identity and restart-inhibit evidence; and
- a canonical manifest self-hash.

Private key and signing-state files must be owner-only and have exactly one
hard link. Symlinks and special files anywhere in the home are refused.

## Required external evidence

The workflow consumes three canonical, self-hashed external documents:

- `zerone.validator-restart-inhibit-evidence/v1` identifies the supervisor
  unit and asserts that restart and start are blocked;
- `zerone.validator-read-only-snapshot-evidence/v1` identifies the snapshot
  and asserts that it is read-only, isolated, and includes ACLs and xattrs;
- `zerone.validator-volume-control-evidence/v1` identifies the provider volume
  and asserts immutable ID, encryption at rest, freshness, and emptiness
  before restore.

These documents are assertions supplied by the operator or control plane.
Their self-hashes are not signatures. This tool does not query Fly.io or
another provider, authenticate a volume ID, inspect provider-side encryption,
or prove supervisor state. Pin and sign them under the deployment policy, and
retain the provider API responses as separate evidence.

The exact field order and schemas are defined in `schema.go`. A producer
computes `evidence_sha256` over the compact JSON structure with that field
empty, then writes the filled compact document with one trailing newline.

The restart-inhibit evidence must be collected no later than the stop
evidence. Snapshot and volume-freshness evidence must be collected no earlier
than the stop evidence. All documents must use canonical UTC timestamps.

## Workflow

First inhibit supervisor restart, stop the source signer, and expose the
source bytes at the canonical path that will be used by `create`. Capture the
former process start time and a digest covering its PID/start time/executable
identity. The restart-inhibit document must name the same source path and
process identity.

```sh
go run ./tools/validator-home-manifest capture-stop \
  --home /srv/zerone-source-snapshot \
  --pid 4812 \
  --process-start-time 2026-07-30T08:41:12Z \
  --process-identity-sha256 aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa \
  --restart-inhibit-evidence /secure/evidence/restart-inhibit.json \
  --last-height 9001 \
  --app-hash 0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef \
  --method "supervisor inhibited; process absent; observer height bound" \
  --observer "observer-a.example" \
  --out /secure/evidence/source-stopped.json
```

The source passed to `create` must now be locally read-only. The mounted
destination must be writable, empty, and on a different `st_dev`. Creation
checks source-process absence and destination emptiness both before and after
the scan.

```sh
go run ./tools/validator-home-manifest create \
  --home /srv/zerone-source-snapshot \
  --destination-home /srv/zerone-destination \
  --stopped-evidence /secure/evidence/source-stopped.json \
  --restart-inhibit-evidence /secure/evidence/restart-inhibit.json \
  --snapshot-evidence /secure/evidence/source-snapshot.json \
  --volume-evidence /secure/evidence/destination-volume.json \
  --destination-volume-id vol_immutable_identifier \
  --out /secure/evidence/validator-home.json
```

Copy the complete snapshot to the destination without starting either signer.
Then verify against the same external evidence:

```sh
go run ./tools/validator-home-manifest verify \
  --home /srv/zerone-destination \
  --manifest /secure/evidence/validator-home.json \
  --restart-inhibit-evidence /secure/evidence/restart-inhibit.json \
  --snapshot-evidence /secure/evidence/source-snapshot.json \
  --volume-evidence /secure/evidence/destination-volume.json \
  --destination-volume-id vol_immutable_identifier
```

Only a separate, authorized release control should decide whether to start
the destination. The local summary includes
`external_control_assertions=unverified` deliberately.

## Trust boundary and refusals

Process absence is checked with signal 0 against the recorded PID before and
after creation and verification. PID reuse fails closed, but absence of that
PID alone cannot prove that no other process holds the consensus key; the
restart-inhibit and host-isolation controls remain essential.

The tool hashes file bytes, sizes, and permission modes. It does not locally
record owners, ACLs, or xattrs; instead it requires explicit external evidence
for a read-only isolated snapshot that includes ACLs and xattrs. Use a custody
mechanism that preserves those attributes during transfer.

Control and home files are read without following the final path component.
Outputs are installed without replacement, require an existing non-symlinked
parent path, and are file- and directory-synced. Creation and verification
also refuse malformed or ambiguous JSON, content drift, non-canonical
Ed25519 keys, inconsistent Comet state, same-device source/destination roots,
nonempty destinations, and any height for which H+1 would overflow.
