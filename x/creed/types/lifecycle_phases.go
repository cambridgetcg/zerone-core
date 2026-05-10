package types

// LifecyclePhase identifies one of the nine root categories in the
// useful-work taxonomy (per docs/superpowers/specs/2026-05-10-recursive-
// useful-work-merged-design.md §4.2). The 9 phases are doctrinally
// fixed at inception; adding/removing a phase is a doctrine amendment
// requiring full governance passage.
//
// Each phase (except KNOWLEDGE, which delegates to TRUTH_SEEKING.md)
// has its own sub-creed under docs/sub_creeds/<phase>.md, hash-pinned
// in .sub-creed-hashes, and a per-phase meta-test in
// tests/cross_stack/useful_work_invariants_test.go.
//
// The numeric values are stable; the names are case-sensitive matches
// to docs/sub_creeds/<phase>.md filename basenames (lowercased).
type LifecyclePhase uint32

const (
	LifecyclePhaseFoundation   LifecyclePhase = 0
	LifecyclePhaseKnowledge    LifecyclePhase = 1
	LifecyclePhaseCuration     LifecyclePhase = 2
	LifecyclePhaseAugmentation LifecyclePhase = 3
	LifecyclePhaseTraining     LifecyclePhase = 4
	LifecyclePhaseEvaluation   LifecyclePhase = 5
	LifecyclePhaseAlignment    LifecyclePhase = 6
	LifecyclePhaseSubstrate    LifecyclePhase = 7
	LifecyclePhaseTools        LifecyclePhase = 8
)

// CanonicalLifecyclePhases is the canonical name-by-number registry of
// the 9 lifecycle phases. Order is doctrinally fixed; new phases append
// (never insert) via doctrine amendment.
//
// The Knowledge phase delegates its sub-creed to docs/TRUTH_SEEKING.md
// (no docs/sub_creeds/knowledge.md file). The HasSubCreedDoc field
// marks this asymmetry.
type LifecyclePhaseDef struct {
	Number         LifecyclePhase
	Name           string // lowercase, matches sub_creeds/<name>.md filename
	HasSubCreedDoc bool   // false only for Knowledge (delegates to truth-seeking)
}

var CanonicalLifecyclePhases = []LifecyclePhaseDef{
	{LifecyclePhaseFoundation, "foundation", true},
	{LifecyclePhaseKnowledge, "knowledge", false},
	{LifecyclePhaseCuration, "curation", true},
	{LifecyclePhaseAugmentation, "augmentation", true},
	{LifecyclePhaseTraining, "training", true},
	{LifecyclePhaseEvaluation, "evaluation", true},
	{LifecyclePhaseAlignment, "alignment", true},
	{LifecyclePhaseSubstrate, "substrate", true},
	{LifecyclePhaseTools, "tools", true},
}
