package keeper

// ToK voice-layer event types and attribute keys.
// These events are emitted from AssembleToKBundle to surface which doctrine
// commitments are being preserved at each extraction step.
const (
	EventTypeToKBundleExtracted    = "tok_bundle_extracted"
	EventTypeToKSnapshotRootPinned = "tok_snapshot_root_pinned"

	AttrToKCommitment    = "tok_commitment"   // value: "TC1", "TC2,TC5", etc.
	AttrToKSelectorKind  = "selector_kind"
	AttrToKBundleSize    = "node_count"
	AttrToKSnapshotRoot  = "snapshot_root"
	AttrToKSnapshotBlock = "snapshot_block"

	// ─── ToK cascade bundling (TC4) ─────────────────────────────────────
	EventTypeCascadeReplayed   = "cascade_replayed"   // TC4: bundle extraction signal
	EventTypeCascadeCompleted  = "cascade_completed"   // TC4: aggregate end-of-cascade signal
)
