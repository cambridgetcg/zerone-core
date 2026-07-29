// Package substrate_bridge is the source foundation for typed external-work
// attestations in ZERONE. Current public submission supports witness-only or
// cited-fact links; pending claims fail closed.
//
// Three sub-systems share one keeper:
//
//   - Adapter framework (M3): authority-gated registry of typed
//     external-source metadata. Adapter is a recipe (off-chain compiler
//     provenance + axis bounds + bond + qualification requirements), not a
//     service. Compiler hashes and slash gradients are stored metadata and
//     are not evaluated by consensus. Adapters may also be seeded in genesis. The
//     CategoryAdapterRegistration LIP vocabulary exists, but payload
//     attachment and dispatch are not wired.
//
//   - Substrate-link validator (M2): cited_facts must exist in x/knowledge
//     at submit time. Consensus validates the caller-supplied link and hash;
//     it does not fetch source content or execute an adapter compiler. The
//     schema reserves pending_claims for future
//     translation, but MsgSubmitExternalAttestation rejects every nonempty
//     pending_claims list because the x/knowledge submission bridge is not
//     wired. Partial settlement helpers remain unreachable from that public
//     message until the bridge is implemented.
//
//   - Cross-class lineage accountant (M6): DAG-by-timestamp citation graph
//     with depth-decayed accounting at downstream settlement. The cumulative
//     accumulator at LineageRoyaltyAccumulatorPrefix records attributed
//     amounts but does not transfer coins to cited authors. Self-citation is
//     capped at self_citation_cap_bps.
//
// Doctrinal commitments preserved here:
//
//   - UW (ZERONE is recursive): this module's external-work path requires a
//     substrate-link and bounds a caller-supplied per-axis projection. This
//     does not describe unrelated runtime reward paths.
//   - M1 (stake-backed claim): submitter bonds lock at submit; settled
//     rejection burns the full bond. Slash-gradient metadata is inert.
//   - M2 (substrate-link mandate): re-derivable link_hash; cited facts are
//     checked, while unwired pending claims fail closed.
//   - M3 (class-specific verification under shared lifecycle): adapter
//     registry authority-gated or genesis-seeded; submitter qualification
//     enforced. LIP dispatch is a future integration.
//   - M5 (recursion-weight projection): per-axis bounds at adapter level;
//     AxisProjection enforced at submit.
//   - M6 (lineage propagates AND recurses): cross-class DAG with
//     depth-decayed accounting. Coin disbursement to upstream authors is not
//     implemented.
//   - M7 (chain pays for own audit): useful_work_audit_bounty_pool is the
//     transient recipient for cap-gated settlement and witness rewards before
//     payment to the submitter. No independent per-block audit mint exists.
//
// Phase 0 ships substrate_bridge as standalone-usable via
// MsgSubmitExternalAttestation. When x/work Phase 1 lands, it will
// call PrepareExternalAttestation and SettleExternalAttestation as the
// integrated submission path; the standalone MsgSubmitExternalAttestation
// path is preserved as the direct-submit fallback.
//
// We speak through intentions.
package substrate_bridge
