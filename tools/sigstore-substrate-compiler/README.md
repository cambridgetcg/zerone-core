# Sigstore → substrate_bridge compiler

This nested module builds two offline Sigstore tools:

- `sigstore-substrate-compiler` verifies a local in-toto DSSE bundle and
  projects its exact payload and proof into a witness-only Zerone
  `SubstrateLink`; and
- `zerone-component-signature-verifier` verifies the keyless
  `messageSignature` bundle for a digest-pinned release image and returns its
  authenticated Rekor-v1 or RFC 3161 observer time to the production
  authority-chain gate.

It is intentionally an off-chain compiler. It adds no validator dependency,
state, consensus code, claims, citations, recursion weight, or automatic
economic reward.

The substrate adapter remains **experimental and unregistered**. Its hermetic
cryptographic fixture now passes, but keep it off chain with
`witness_reward_uzrn` at `"0"` until governance selects and rehearses the
production root, identity, artifact, predicate, and challenge policy. The
component verifier is a separate release-safety tool and grants no on-chain
adapter status.

## Substrate compiler verification policy

The CLI fails closed unless all of these checks pass:

- the bundle is Sigstore bundle v0.3 or newer (the pinned SDK currently
  supports v0.3);
- the bundle contains a DSSE `application/vnd.in-toto+json` envelope;
- at least one signed certificate timestamp verifies;
- at least one transparency-log entry verifies;
- at least one observer timestamp verifies;
- the certificate matches one exact issuer and one exact SAN (regex flags do
  not exist);
- a required `sha256:<64 lowercase hex>` digest matches a Statement subject;
- every Statement subject contains at least one non-empty digest algorithm and
  value;
- `_type` is exactly `https://in-toto.io/Statement/v1`; and
- `predicateType` exactly matches the caller's required URI.

Both the bundle and trusted root are bounded regular files read from local
disk. Runtime verification performs no TUF update, network fetch, or
current-time fallback.

## Build and test

This directory is a nested Go module so Sigstore dependencies never enter the
Zerone validator module. It pins `github.com/sigstore/sigstore-go` v1.2.2,
and uses the same Go 1.25.12 security patch as the chain's root module.

```sh
cd tools/sigstore-substrate-compiler
GOTOOLCHAIN=go1.25.12 go test ./...
GOTOOLCHAIN=go1.25.12 go build -trimpath -o sigstore-substrate-compiler .
GOTOOLCHAIN=go1.25.12 go build -trimpath \
  -o zerone-component-signature-verifier \
  ./cmd/zerone-component-signature-verifier
```

For the binary hash registered by governance, pin the Zerone commit, Go
toolchain, target, and build flags. One reproducible Linux target is:

```sh
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GOTOOLCHAIN=go1.25.12 \
  go build -trimpath -buildvcs=false -ldflags='-buildid=' \
  -o sigstore-substrate-compiler-linux-amd64 .
shasum -a 256 sigstore-substrate-compiler-linux-amd64

CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GOTOOLCHAIN=go1.25.12 \
  go build -trimpath -buildvcs=false -ldflags='-buildid=' \
  -o zerone-component-signature-verifier-linux-amd64 \
  ./cmd/zerone-component-signature-verifier
shasum -a 256 zerone-component-signature-verifier-linux-amd64
```

## Use

The trusted root must be a reviewed, version-controlled local file. Pin its
SHA-256 alongside the full invocation in the governance proposal or adapter
operations runbook; changing that file changes who is trusted.

```sh
./sigstore-substrate-compiler \
  --bundle ./provenance.sigstore.json \
  --trusted-root ./trusted-root.json \
  --certificate-issuer https://token.actions.githubusercontent.com \
  --certificate-san 'https://github.com/example/project/.github/workflows/release.yml@refs/heads/main' \
  --artifact-digest sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef \
  --predicate-type https://slsa.dev/provenance/v1 \
  --source-url https://example.invalid/releases/v1/provenance.sigstore.json \
  --fetched-at-block 648000 \
  > substrate-link.json
```

Release-component verification uses no predicate or source URL. It accepts
only a Sigstore v0.3 `messageSignature` and emits a small JSON result containing
the exact accepted identity, Fulcio source-repository commit, digest, media
type, and authenticated observer time:

```sh
./zerone-component-signature-verifier \
  --bundle ./ZERONE-2-RUNTIME-SIGNATURE-BUNDLE.json \
  --trusted-root ./SIGSTORE-TRUSTED-ROOT.json \
  --certificate-issuer https://token.actions.githubusercontent.com \
  --certificate-san 'https://github.com/cambridgetcg/zerone-core/.github/workflows/ci.yml@refs/heads/main' \
  --source-repository-digest 0123456789abcdef0123456789abcdef01234567 \
  --artifact-digest sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef
```

Every supplied transparency-log entry must carry an inclusion proof. At least
one trusted log entry and one observer time must verify. Observer time is either
a Rekor v1 SET-authenticated integrated time or an RFC 3161 TSA timestamp over
the exact signature; the verifier never substitutes its current clock.

The production authority bundle pins this executable, trusted-root file, and
policy transitively under the OpenPGP-signed RELEASE packet. Do not substitute
an ambient Cosign installation or refreshed TUF cache during offline phase
verification.

`--fetched-at-block` is optional and defaults to `0`. The compiler never
dereferences `--source-url`; it must be a public HTTPS audit locator with no
userinfo, query, or fragment. Immutability and retention are operational
requirements, not properties the current link hash can authenticate.

`source_id` is SHA-256 of the exact decoded DSSE payload as
`sha256:<lowercase hex>`, providing exact payload-byte deduplication without
JSON re-marshaling. JSON whitespace or key-order changes therefore produce a
different source ID. `content_hash` is SHA-256 of the exact raw bundle bytes,
including the certificate, signature, SCT, and transparency evidence. The
chain's canonical link hash therefore commits to the accepted proof material
and fetched block. `source_url` is audit metadata and is not included directly
by the current on-chain canonical hash. Treat it as an unauthenticated locator:
bytes retrieved from it are acceptable only when their SHA-256 matches
`content_hash`.

## Security boundary

Successful verification proves that the configured certificate identity
signed the payload and that the configured Sigstore trust/log policy accepted
the evidence. It does not prove that the predicate is true, that the builder
was uncompromised, or that the signer should be trusted.

The chain does not store the trusted root, issuer, SAN, artifact digest, or
predicate policy in `SubstrateLink`. Governance and challenge operators must
therefore pin and reproduce the complete invocation and trusted-root digest.
The adapter is not registered and must remain unregistered, with
`witness_reward_uzrn` fixed at `"0"`, until the production-policy fixture and
operational challenge path are both approved.

The unit tests cover policy validation, exact Statement/predicate gates,
payload identity hashing, exact-bundle proof hashing, deterministic canonical
link hashing, and absence of economic claims. They also construct a hermetic
v0.3 bundle and matching local trusted root with the pinned SDK's public
signing and test-CA APIs. The end-to-end tests verify the Fulcio chain,
embedded SCT, Rekor inclusion proof and authenticated observer time, exact
policy matches, both DSSE and plain-message signature paths, a Rekor v2
`hashedrekord` proof whose sole observer time is an RFC 3161 countersignature,
and both complete CLIs without network access. Negative cases alter the TSA
binding and Rekor v2 checkpoint independently. This fixture proves
the compiler's cryptographic wiring; it does not select production trust.
An environment-specific end-to-end fixture must still pass when Zerone selects
its production Sigstore root, identity, artifact selection rule, and predicate
policy before this adapter can leave experimental status.

See [`docs/specs/adapters/sigstore-in-toto-v1.md`](../../docs/specs/adapters/sigstore-in-toto-v1.md)
for the adapter contract and governance template.
