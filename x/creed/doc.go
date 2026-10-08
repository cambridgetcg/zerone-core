// Package creed stores optional, versioned hash commitments to the source creed
// in docs/TRUTH_SEEKING.md. It stores pins, commitment entries and council records,
// not the document text or evidence that its claims are true.
//
// Source responsibilities:
//   - keeper/keeper.go owns pin storage and the shared descending history walk.
//     Queries and genesis export visit stored records, including sparse imported
//     histories; export retains ascending history and a separate current pin.
//   - keeper/msg_server.go gates direct anchoring on the SDK gov authority and
//     direct_anchor_enabled. That parameter defaults to true.
//   - The custom x/gov module can call AnchorPinFromBytes for an attached creed
//     amendment. This dispatch primitive and the council registry do not establish
//     an enabled amendment category or separate human/AI quorum.
//   - x/work_creed owns genesis-supplied pins for other lifecycle phases. It rejects
//     Knowledge-phase pins; that phase retains this module's optional registry.
//
// Doctrine declarations belong in types/*_creed.go and the corresponding documents
// under docs/. x/knowledge/keeper/doctrine_genesis.go owns their corpus import;
// doctrine import and a document pin are different records. Phase plans belong in
// those design documents, not in this package's implementation status.
//
// scripts/check_creed_hash.sh checks repository bytes against .creed-hash. Neither
// that check nor module registration demonstrates adoption by a running network.
// Default creed genesis and the published zerone-1 genesis contain no pin.
// See dashboard/module-guide.json for module purposes, labels and connections;
// docs/AUTHORITATIVE-STATE.md separates source, pending design and runtime evidence.
package creed
