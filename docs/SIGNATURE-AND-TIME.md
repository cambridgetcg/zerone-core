# Signature and time in Zerone production

This document fixes the production interpretation of the Signature and
Timestamp concepts in `FIG_Finalv1.1.pdf` and
`Detailed_Description_Finalv2.2.pdf`. Those documents are design and patent
inputs, not an executable wire specification. In particular, their references
to Internet-synchronized node clocks are not safe as a deterministic consensus
rule.

This interpretation is pinned to the exact supplied source bytes:

- `FIG_Finalv1.1.pdf` — 12 pages, SHA-256
  `702a625d20d0250adf478154454db198a9058fc64f4b063059c8c7bff23375ec`;
- `Detailed_Description_Finalv2.2.pdf` — 27 pages, SHA-256
  `8a1744398ed8c43d3a3f153ced1b839e7f6a926974da57d4fbf47e5e0abe58bb`.

The figures establish a useful intent: an effect-bearing transaction or
schedule carries signer authority, replay protection, and a temporal boundary;
an informative message may omit a signature only when it cannot cause an
effect. Zerone realizes that intent through separate, explicit trust domains.
The detailed description makes those distinctions on pages 8–9 and walks the
transaction/schedule lifecycle on pages 15–17. Its Internet-clock assumptions
on pages 6 and 13 are treated as design context, not adopted as consensus
rules. The accompanying drawing's Figure 2 labels Signature 212 and Timestamp
213; Figures 11–15 supply the ingress, execution, and expiry context.

## Production map

| Domain | Signature authority | Replay boundary | Authoritative time |
| --- | --- | --- | --- |
| Ordinary transactions | Cosmos `TxRaw`, normally secp256k1 | account number, chain ID, account sequence, optional timeout height | committed consensus block height/time |
| Identity registration | `TxRaw` sender plus the submitted Ed25519 identity key | chain ID, sender, full self-certifying DID, key, account type, metadata | transaction timeout height and committed consensus block |
| Operational-key rotation | `TxRaw` sender, current Ed25519 operational key, and proposed new Ed25519 key | chain ID, sender, current key version, new key, signed acceptance horizon | containing block's consensus time |
| Release decisions | detached OpenPGP signature by the named phase authority | exact canonical payload hash and predecessor graph | signed decision deadline plus committed block evidence |
| Release components | keyless Fulcio certificate and Sigstore message signature | exact OCI manifest SHA-256, workflow identity, and source commit | authenticated observer time: Rekor v1 integrated time or RFC 3161 TSA time |
| Off-chain schedules | AgentTool owner policy; every emitted chain transaction still needs chain authority | scheduler job identity plus the resulting transaction sequence/timeout | scheduler time off chain; consensus time for all chain effects |

No validator transition may read the validator host's wall clock, call a time
server, or infer ordering from when an RPC request arrived. `ctx.BlockTime()` is
the consensus-provided header time shared by the state machine. Height remains
the preferred deadline whenever a height expresses the intended boundary.
Consensus time is authoritative for deterministic acceptance and ordering; it
is not cryptographic proof that a real-world event occurred at that instant.

Repository fields named `timestamp` do not gain authority from the name. The
alignment and emergency modules populate their persisted timestamps from
consensus block time. Wall-clock duration in the activation-preflight report is
local performance telemetry produced outside state transition execution and is
never an activation condition.

## Transaction signatures

The patent message in Figure 2 includes Signature 212 and Timestamp 213.
Zerone's ordinary transaction analogue is the standard Cosmos transaction
envelope:

- `TxRaw` authenticates the account that authorizes the messages;
- chain ID and account number separate signing domains;
- account sequence is the single-use nonce;
- fee, gas, memo, and message bytes are covered by the sign document; and
- timeout height provides an objective inclusion deadline when set.

This is where Zerone realizes the detailed description's page-16 preference
for ECDSA on secp256k1. The separate Ed25519 proofs below serve identity-key
possession and rotation continuity; they do not replace the Cosmos account
signature.

A sender wall-clock timestamp is deliberately not added to every message. It
would be redundant with the transaction nonce for replay prevention and unsafe
if validators compared it with their independent clocks. When a module needs a
wall-time-shaped deadline, the deadline is signed into the message and compared
only with consensus block time.

Likewise, an opaque optional field named `signature` is not automatically a
second authorization layer. For example, the training-attestation external
audit signature is preserved as supplied but is not parsed as consensus
authority; the attester's effect is authorized by `TxRaw`. A consumer that
wants to trust that optional external evidence must define and verify its own
format, signer, domain, and replay policy off chain.

## Identity registration

`MsgRegisterAccount` has two separate authorities. `TxRaw` authorizes the
Cosmos account that pays for and owns the registration, while
`identity_proof_signature` proves possession of the independently stored
Ed25519 identity key before its self-certifying DID can be claimed. The latter
signs:

```text
"zerone.auth/register-account/v1" || 0x00 ||
u32(len(chain_id)) || chain_id ||
u32(len(sender)) || sender ||
u32(len(did)) || did ||
identity_public_key[32] ||
u32(len(account_type)) || account_type ||
u32(len(metadata)) || metadata
```

All lengths are unsigned 32-bit big-endian values. Addresses must be the
canonical lowercase 20-byte Zerone Bech32 form. The DID must be exactly
`did:zrn:<full-64-lowercase-hex-public-key>`. Keys, and the `R` point and `S`
scalar of accepted signatures, must use canonical Ed25519 encodings; keys and
`R` must be non-small-order points in the prime-order subgroup. Account type is
a closed vocabulary. The handler derives the initial operational-key SHA-256
commitment from the proven key rather than trusting an unrelated hash.

The CLI `registration-sign-bytes` command emits these bytes without reading a
private key. The one-shot `onboard` command persists the identity seed before
broadcast with file and directory synchronization and creates the proof
locally. Reuse accepts only an owner-held, owner-only, regular non-symlink file
whose stored address and key derivation match exactly. The offline
`verify-registration-proof` command is used by the
signed bootstrap gate even in check-only mode, before a malformed proof can
consume the first bootstrap sequence. Check-only mode also authenticates the
private successor and block-1 anchor, then verifies the ordinary `TxRaw`
signature against its current account number and sequence; it omits only the
final submission. Chain ID, sender, DID uniqueness, and the normal `TxRaw`
sequence prevent proof reuse as a different registration.

## Operational-key rotation

`MsgRotateKey` requires three signatures across two layers:

1. The normal `TxRaw` signature proves that the Cosmos account authorized the
   transaction.
2. `authorization_signature` proves continuity from the account's current
   Ed25519 operational key.
3. `new_key_confirmation_signature` proves possession and acceptance by the
   proposed new Ed25519 operational key.

The second signature covers these exact bytes, with all integers big-endian:

```text
"zerone.auth/rotate-key/v1" || 0x00 ||
u32(len(chain_id)) || chain_id ||
u32(len(sender)) || sender ||
u32(current_key_version) ||
i64(authorization_expires_at_unix) ||
new_operational_key[32]
```

The new-key confirmation signs the identical field sequence under the distinct
domain `"zerone.auth/accept-key/v1"`, so neither proof can be substituted for
the other.

The handler requires canonical, prime-subgroup 32-byte Ed25519 keys and
canonical 64-byte signatures. The signed expiry must be strictly later than the
containing block's consensus time and no more than ten minutes later. This is a
maximum future acceptance horizon, not proof of when a signer physically made
the signature. The current key version is a single-use nonce: after a
successful rotation, the version increments, so the same pair of proofs cannot
be replayed. Binding chain ID and sender prevents cross-chain and cross-account
reuse. The block-height cooldown anchor survives genesis export/import.

Operational Ed25519 keys remain separate from Cosmos `BaseAccount` transaction
keys. Rotation never replaces the account's secp256k1 key. Registration derives
the canonical lowercase SHA-256 operational-key commitment from the submitted
public key and rejects a conflicting caller-supplied hash.

The historical `zerone-1` registration path accepted a lowercase `did:zrn:`
prefix followed by either 32 or 64 hexadecimal characters, compared the suffix
case-insensitively with the identity key (or its first 32 hex characters), and
permitted the operational-key hash to be omitted. The halt binary therefore
has one export-only compatibility boundary: when the source BaseApp chain ID
is exactly `zerone-1`, it preserves either historical DID spelling and may
preserve an empty hash on an unrotated version-1 record whose identity and
operational public keys match. It does not normalize an identifier, derive a
replacement hash, invent a proof of possession, or fabricate rotation history.
The public keys, DID derivation and exact account/mapping agreement, heights,
flags, and version/key relationship remain strictly validated; a non-derived
DID or nonempty incorrect hash is corruption.

This exception does not make the snapshot importable genesis. Generic genesis
validation has no trusted chain-ID input and remains strict, as does
`InitGenesis` even if its context says `zerone-1`. All new registrations write
the derived hash and full lowercase DID, and exports from `zerone-2`, an empty
chain ID, or any other chain reject the legacy shape. The read-only DID query
on exact `zerone-1` accepts the old 32/64-hex grammar so committed mappings stay
queryable, but performs a byte-exact lookup and never normalizes or writes the
request. Every other chain retains strict current DID validation. The CLI
export process loads the source genesis chain ID from a regular non-symlink
file, rejects any configured chain-ID
override that disagrees, binds the source value explicitly into BaseApp, and
then binds it into the module-export context. The resulting `zerone-1` state
export is historical evidence only and is excluded from successor state.

The identity seed is only the first operational private key. After each
rotation, custody must retain and back up the newly current operational private
key; otherwise the next rotation is permanently unavailable because the
retired on-chain recovery path is not part of the slim chain.

Run the CLI helper once for each proof:

```text
zeroned tx zerone_auth rotation-sign-bytes <new-op-key-hex> \
  --proof <authorization-or-acceptance> \
  --current-key-version <current-version> \
  --authorization-expires-at-unix <unix-seconds> \
  --chain-id <chain-id> --from <sender-key>
```

The matching TypeScript helpers emit the same two domain-separated byte
sequences. Neither implementation reads a private key; signing stays in the
user's custody provider.

## Schedules and derived transactions

Figures 11–15 describe schedule ingress, validation, execution, and dynamic
expiry. They also distinguish a user-signed schedule from later transactions
derived by an immutable schedule process. Zerone does not currently expose that
historical BVM/scheduler as a production consensus feature. Scheduling belongs
to off-chain AgentTool automation, while each effect-bearing transaction that
reaches Zerone must still satisfy the normal chain authorization and replay
rules.

A pre-signed transaction may be held for later broadcast only when its sequence,
fees, messages, and timeout height are acceptable for that exact future use.
AgentTool must not rewrite it. If AgentTool constructs a fresh transaction at
run time, the configured signer must authorize those exact bytes then.

Any future on-chain scheduler requires a separately reviewed LIP. At minimum it
must define bounded work, cancellation, deterministic due-time evaluation,
authorization lineage from schedule to derived action, monotonic nonces, and
atomic failure behavior. Merely recreating the removed legacy scheduler would
not satisfy this contract.

## Release signatures and timestamps

Release authority and artifact provenance answer different questions:

- OpenPGP signatures authorize a particular RELEASE, DARK, CUTOVER, FINAL, or
  OPEN payload within its declared scope. Their packet creation times support
  chronology checks but do not replace signed deadlines or committed evidence.
- Sigstore proves that the exact GitHub Actions workflow identity at the exact
  Fulcio `sourceRepositoryDigest` commit signed each digest-pinned component
  manifest after it matched the component-specific full digest ref approved in
  the protected signing environment. This is signing authorization, not proof
  that the workflow built the image and not deploy authority by itself. The
  production verifier accepts only a local, hash-pinned trusted
  root, exact issuer, exact SAN, exact source commit, a certificate SCT, a
  transparency-log inclusion proof on every supplied log entry, and a signed
  observer timestamp.
- Component `signed_at` must equal a cryptographically verified observer time.
  For Rekor v1 this may be its SET-authenticated `integratedTime`; Rekor v2 has
  no integrated time, so it must be an RFC 3161 timestamp-authority
  countersignature over the component signature. A JSON field claiming a time
  is not evidence by itself.
- `verified_at` records when an operator assembled the evidence. It is not the
  signing time and must not be used to order consensus events.

The authority-chain verifier performs Sigstore verification offline after the
OpenPGP RELEASE signature has authenticated the hashes of both the verifier
binary and frozen trusted root. It never downloads TUF state, contacts Rekor, or
falls back to the workstation's current time while deciding whether a component
signature is valid.

The protected signing run also signs the exact same-run Frontier macOS intake
archive. Its canonical internal manifest binds the source commit, Bun version,
register, bundled JavaScript, and universal ACL helper; the same Fulcio, Rekor,
and RFC 3161 observer-time requirements apply. That signature authenticates the
off-chain distribution unit but does not authorize a chain launch or effect.

## Launch boundary

The source implementation and hermetic cryptographic tests are necessary but
do not themselves authorize production. A production GO still requires the
three real digest-pinned images, keyless bundles from the protected
`zerone-production-signing` environment (pre-created with required reviewers,
`main` restriction, the exact `ZERONE_PRODUCTION_SIGNING_POLICY` sentinel, and
three environment-scoped approved component digest refs),
a reviewed frozen Sigstore trusted
root, the hash-pinned verifier binary, real OpenPGP authority packets, exact
signed transactions, and current live evidence at every phase gate.

Until those bytes exist and the complete authority bundle passes on the
dedicated Linux release workstation, `zerone-2` activation remains a NO-GO.
