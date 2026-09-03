# Fail-closed signed-authority Fly deployment

Every checked-in Fly profile deliberately carries an invalid image placeholder.
Make each reviewed config outside Git and replace all active placeholders with
final app, volume, peer, origin, region, and full immutable image values.

Production DARK/CUTOVER/OPEN deployments must derive their expectations from
the current signed phase authority. Do not manually transcribe
app/image/role/hash arguments:

```bash
deploy/fly-deploy-authorized.sh --check \
  /secure/RELEASE-PACKET.json \
  /secure/RELEASE-PACKET.json.sig \
  /secure/DARK-START-DECISION.json \
  /secure/DARK-START-DECISION.json.sig \
  /secure/fly.edge.toml \
  zerone_2_edge_private \
  '<out-of-band-main-signer-full-fingerprint>'

deploy/fly-deploy-authorized.sh \
  /secure/RELEASE-PACKET.json \
  /secure/RELEASE-PACKET.json.sig \
  /secure/DARK-START-DECISION.json \
  /secure/DARK-START-DECISION.json.sig \
  /secure/fly.edge.toml \
  zerone_2_edge_private \
  '<out-of-band-main-signer-full-fingerprint>'
```

The OPEN form has the full closing authority chain:

```bash
deploy/fly-deploy-authorized.sh --check \
  /secure/RELEASE-PACKET.json /secure/RELEASE-PACKET.json.sig \
  /secure/OPEN-BETA-DECISION.json /secure/OPEN-BETA-DECISION.json.sig \
  /secure/fly.edge.public.toml zerone_2_edge_public \
  '<out-of-band-main-signer-full-fingerprint>' \
  /secure/OPEN-BETA-INITIATION-EVIDENCE.json \
  /secure/OPEN-BETA-INITIATION-EVIDENCE.json.sig \
  /secure/FINAL-CHECKPOINT.json /secure/FINAL-CHECKPOINT.json.sig \
  '<out-of-band-transition-signer-full-fingerprint>' \
  /secure/open-authority-bundle
```

The wrapper snapshots RELEASE, its detached signature, the phase authority,
its detached signature, and the config. It requires canonical JSON, exactly
one OpenPGP `VALIDSIG` matching the supplied out-of-band full fingerprint, an
exact `GO`, and a config key within that payload's scope. It verifies the
release-to-phase hash and image-component joins, extracts the app, complete
image reference, role, image component, and config SHA-256 directly from the
signed phase bytes, and rejects reserved old-chain app identities or
inconsistent release topology. The archive gateway is the sole phase-dependent
public config: RELEASE fixes its static mapping/renderer/template, verified
FINAL supplies A/E/B, and OPEN signs the exact deterministic output hash.

DARK deployment before block 1 also checks wall-clock time against the signed
deadline; its main-key initiation evidence permits the scoped private
continuation afterward. CUTOVER is accepted only by
`mainnet/fly-cutover-authorized.sh`, which enforces observer-first/signer-last
order and fresh trusted-chain/height checks. OPEN appends its evidence
pair, transition-signed `FINAL-CHECKPOINT.json` and signature, and the
out-of-band full transition-key fingerprint. Late, mismatched, failed,
wrong-transaction, incomplete authority-chain, or wrong-key evidence is
rejected.

Production OPEN checks and deployments run only from the dedicated
`linux/amd64` release workstation. OPEN executes the exact signed Linux/AMD64
`zeroned-zerone-1-release` binary to verify Comet evidence; a macOS host,
native rebuild, emulator wrapper, or test-double binary is not a production
verification path.

The gate parses TOML structurally. Private profiles must have no Fly service;
the public edge may expose only P2P TCP 26656; public gateways may expose only
their HTTP origin on 80/443; and stateful profiles must use the exact mounts and
`immediate` deployment strategy fixed by policy. Exact per-profile root and
environment-key sets reject Fly file/process/entrypoint/release-command
overrides, loader variables, and mounts on stateless gateways.

Only these authority paths can deploy:

- DARK-START: private z2 validator, edge profiles, and private gateway;
- CUTOVER: z1 observer first and z1 halt signer last, only through the specialized gate;
- generated ARCHIVE-ADOPTION: private z1 candidate and final archive origin;
- OPEN-BETA: public z2 edge, z2 query gateway, and z1 archive gateway.

The release packet alone authorizes no deployment. For archive profiles,
reproduce `ARCHIVE-ADOPTION-AUTHORITY.json` and both TOMLs with
`deploy/mainnet/render-archive-configs.sh` on two machines before signing or
deploying; never hand-author that `MATCH` payload. Deploy candidate/final only
with `deploy/mainnet/fly-deploy-archive-authorized.sh`; it reruns the renderer,
byte-compares the signed adoption authority, verifies the transition signature,
and sends only the reproduced config to the low-level gate.

The lower-level `deploy/fly-deploy-pinned.sh` is called by the signed-authority
wrapper and may be used directly only for inert local validation/tests. It
accepts one lowercase registry/repository path followed by exactly `@sha256:`
and 64 lowercase hexadecimal characters. Tags, tagged digests, uppercase or
short digests, placeholders, duplicate image keys, image keys outside
`[build]`, missing/duplicate roles, and app/role/image/config values differing
from the signed phase authority fail closed.

Before invoking Fly, the wrapper snapshots the exact config bytes, clears
ambient app selectors, and passes explicit `--app`, `--config`, and `--image`
values. Fly therefore cannot build/upload the repository, select an ambient
app, or replace the validated digest with a tag. The deployment gate does not
replace registry access control, signed image/provenance/SBOM verification,
vulnerability review, secret scanning, or phase-chain signature verification.

Focused local tests never contact Fly:

```bash
bash deploy/fly-deploy-pinned_test.sh
bash deploy/validate-fly-phase-config_test.sh
bash deploy/fly-deploy-authorized_test.sh
```
