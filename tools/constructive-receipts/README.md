# constructive-receipts

`constructive-receipts` is the offline, zero-value tree-v1 ↔ PoCA receipt
bridge. It re-evaluates local PoCA profile/evidence bytes and accepts only the
exact reviewed constructive-intelligence tree v1 document and
`protocol-software-supply-chain@2026q3` node.

Run the published honest refusal:

```bash
go run ./tools/constructive-receipts \
  --request docs/examples/constructive-receipts/zerone-release-partial-v0.request.json \
  --tree dashboard/public/standards/constructive-intelligence-tree.v1.json \
  --profile docs/examples/poca/slsa-build-l2-v0.profile.json \
  --evidence docs/examples/poca/zerone-release-partial-v0.evidence.json
```

The result is `REFUSED`: the PoCA profile is `DRAFT` and lacks the exact
Sigstore policy binding required by this bridge. PoCA `E2_CONFORMANT` is not
tree `E2` or tree `E3`.

Every possible output retains:

```text
assurance = UNVERIFIED_SHADOW_PROJECTION
tree.granted_attainment_evidence = NONE
qualification = NONE
economic_effect = NONE
amount_uzrn = "0"
consumption_state = NOT_RECORDED
replay_protection = NONE_OFFLINE
```

The consumption key is deterministic, but this command is not a replay ledger
or entitlement service. It performs no network request, Sigstore
authentication, chain read/write, qualification update, escrow action, or
reward calculation.

See the
[constructive receipt shadow v0 specification](../../docs/specs/attestations/constructive-receipt-shadow-v0.md)
for the candidate predicate, digest algorithms, schemas, known-answer vectors,
and limitations.
