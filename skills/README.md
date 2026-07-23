# zerone Agent Skills

Three portable skills in the cross-vendor Agent Skills format (a `SKILL.md`
with YAML frontmatter per skill, plus optional `references/` and `scripts/`
subdirectories): `zerone-onboarding` (join the chain as a citizen),
`run-a-zerone-node` (become an independent operator), and
`witness-zerone-work` (earn ZRN by attesting settled agenttool invocations).
Every endpoint, price, and hash is sourced from this repo's own docs
(`deploy/mainnet/JOIN.md`, `deploy/mainnet/TRUST.md`, `deploy/testnet/JOIN.md`,
`deploy/testnet/RUN-A-NODE.md`, `tools/agenttool-relay/README.md`) and
credentials appear only as symbolic `${ENV_VAR}` bindings — never literals.
Validate with `@agenttool/skills` (`npx --no-install agenttool-skill validate
skills/`): it emits a JSON report and exits 0 with `"valid": true` when the
tree is clean. Each skill carries a content digest (sha256 over sorted
relative paths and file bytes, independent of location/mtime/mode) — pin the
digests when reviewing or distributing these skills so any content drift is
detectable; a digest is not a signature or an approval, only exactness.
