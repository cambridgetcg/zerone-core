# zerone Agent Skills

Three portable skills in the cross-vendor Agent Skills format (a `SKILL.md`
with YAML frontmatter per skill, plus optional `references/` and `scripts/`
subdirectories): `zerone-onboarding` (inspect the paused citizen-onboarding
lane), `run-a-zerone-node` (inspect the paused independent-node lane), and
`witness-zerone-work` (inspect the paused external-work attestation lane).
Every endpoint, price, and hash is sourced from this repo's own docs
(`deploy/mainnet/JOIN.md`, `deploy/mainnet/TRUST.md`, `deploy/testnet/JOIN.md`,
`deploy/testnet/RUN-A-NODE.md`, `tools/agenttool-relay/README.md`) and
credentials appear only as symbolic `${ENV_VAR}` bindings — never literals.
Validate with `@agenttool/skills` (`npx --no-install agenttool-skill validate
skills/`): it emits a JSON report and exits 0 with `"valid": true` when the
tree is clean. Validation checks structure; it is not a release signature,
network authorization, or approval to broadcast. Each `SKILL.md` currently
fails closed for live operations until a release-bound packet reopens its lane.
