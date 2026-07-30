# `poca-shadow`

`poca-shadow` is Zerone's offline Proof of Constructive Adaptation v0
evaluator. It validates a versioned capability DAG and normalized local
evidence, then emits a deterministic shadow certificate.

It performs no network access, signature verification, chain query,
transaction, qualification update, mint, vesting operation, or reward
calculation. Every accepted input produces:

```json
{
  "assurance": "UNVERIFIED_SHADOW_PROJECTION",
  "reward": {
    "economic_effect": "NONE",
    "amount_uzrn": "0",
    "reason": "shadow-profile-v0"
  }
}
```

Run the synthetic partial fixture from the repository root:

```bash
go run ./tools/poca-shadow \
  --profile docs/examples/poca/slsa-build-l2-v0.profile.json \
  --evidence docs/examples/poca/zerone-release-partial-v0.evidence.json
```

Use `--require-crown --expect-profile-digest sha256:<reviewed-digest>` as a CI
gate. The digest must come from a separately reviewed and committed profile;
deriving and accepting it in the same unreviewed job is not pinning. Use
`--format in-toto` to emit an unsigned in-toto Statement v1 for a separate
pinned DSSE/Sigstore workflow.

The contract, refusal semantics, and machine-readable schemas live in
[`proof-constructive-adaptation-v0.md`](../../docs/specs/attestations/proof-constructive-adaptation-v0.md).
