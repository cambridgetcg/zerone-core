// Package work_creed stores genesis-supplied sub-creed pins for useful-work phases.
// Its keeper holds the latest pin per phase and supports get/set, ascending
// iteration and genesis import/export. Default genesis is empty; there are no
// current transaction, query or post-genesis amendment/history APIs.
//
// The source documents live at docs/sub_creeds/<phase>.md, under the useful-work
// doctrine in docs/USEFUL_WORK.md. Pins commit to document bytes, not truth or
// agreement, and source registration does not demonstrate runtime adoption.
//
// Knowledge is deliberately excluded: SetSubCreedPin rejects phase=1. Its source
// doctrine is docs/TRUTH_SEEKING.md and its optional pin registry belongs to
// x/creed. That ownership boundary is not a call into creed or an automatic pin.
//
// See dashboard/module-guide.json for the shared module catalog. Proposed
// amendment/history APIs remain design in
// docs/superpowers/specs/2026-05-10-recursive-useful-work-merged-design.md, section 10.1.
package work_creed
