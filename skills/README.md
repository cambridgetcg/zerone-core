# zerone Agent Skills

Four portable skills in the cross-vendor Agent Skills format (a `SKILL.md`
with YAML frontmatter per skill, plus optional `references/` and `scripts/`
subdirectories): `zerone-onboarding` (point agents on the household hub to the
agents' door, open since 2026-10-29, and inspect the otherwise paused
citizen-onboarding lane), `run-a-zerone-node` (inspect the paused
independent-node lane),
`witness-zerone-work` (inspect the paused external-work attestation lane), and
`run-a-private-zerone-net` (set up a private, valueless network across an
owner's own machines; it refuses every live chain id).
Every endpoint, price, and hash is sourced from this repo's own docs
(`deploy/mainnet/JOIN.md`, `deploy/mainnet/TRUST.md`, `deploy/testnet/JOIN.md`,
`deploy/testnet/RUN-A-NODE.md`, `tools/agenttool-relay/README.md`) and
credentials appear only as symbolic `${ENV_VAR}` bindings — never literals.
Validate with `@agenttool/skills` (`npx --no-install agenttool-skill validate
skills/`): it emits a JSON report and exits 0 with `"valid": true` when the
tree is clean. Validation checks structure; it is not a release signature,
network authorization, or approval to broadcast. The first three
fail closed for live operations until a release-bound packet reopens their
lanes, except that `zerone-onboarding` sends agents on the household hub to the
agents' door guide (<https://zerone.ai/join/agents.md>);
`run-a-private-zerone-net` never addresses a live network.
