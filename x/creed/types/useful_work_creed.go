package types

// UsefulWorkCommitment is the chain-built-in registration of the
// USEFUL_WORK doctrine's single commitment. Parallel to
// CanonicalCommitments (which holds truth-seeking commitments 1-20),
// kept separate because UW is structurally different — one commitment
// + N mechanisms, not N co-equal commitments.
//
// UW is fixed and indivisible: this constant never changes its value
// or its statement once shipped. Only mechanisms (M1-M7) extend.
const UsefulWorkCommitment = "UW"

// UsefulWorkStatement is the canonical short statement of UW.
// It must match the heading in docs/USEFUL_WORK.md exactly. The
// cross-stack meta-test TestUsefulWork_DoctrineAndContractStayInSync
// enforces this match.
const UsefulWorkStatement = "ZERONE is recursive"

// UsefulWorkMechanism describes one of the M1-M7 mechanisms that
// enforce UW. Number is the mechanism index (1..7); Name is the
// short label that must match the corresponding "### MN. <Name>"
// header in docs/USEFUL_WORK.md.
type UsefulWorkMechanism struct {
	Number uint32
	Name   string
}

// CanonicalUsefulWorkMechanisms is the canonical name-by-number
// registry of the seven mechanisms that enforce UW at the time this
// binary was built.
//
// To add a mechanism (NEVER to add a second co-equal commitment —
// that would dilute UW's indivisibility):
//  1. Add the "### MN. <Name>" section to docs/USEFUL_WORK.md.
//  2. Bump .useful-work-hash to the new sha256 of the normalized file.
//  3. Append the (Number, Name) pair to the slice below.
//  4. Add a binding TestUW_MN test in
//     tests/cross_stack/useful_work_invariants_test.go.
//  5. Wire the mechanism's voice (event attribute), refusal (error
//     message), and position (x/<module>/doc.go declaration).
//
// The TestUsefulWork_DoctrineAndContractStayInSync meta-test catches a
// step omitted from this list.
//
// Mechanism removal is a doctrine amendment requiring full governance
// passage (LIP class-registration revocation under M3). Mechanisms
// shipped at inception are load-bearing and do not retire.
var CanonicalUsefulWorkMechanisms = []UsefulWorkMechanism{
	{1, "Stake-backed claim"},
	{2, "Substrate-link mandate"},
	{3, "Class-specific verification under shared lifecycle"},
	{4, "Reward formula"},
	{5, "Recursion-weight projection over six axes"},
	{6, "Lineage propagates AND recurses"},
	{7, "The chain pays for the audit of its own useful work"},
}

// CanonicalRecursiveAxes is the canonical name-by-number registry of
// the six recursive axes that compose recursion-weight (M5). Axes are
// fixed by the doctrine — adding/removing an axis is a doctrine
// amendment requiring full governance passage and a hash bump.
//
// Per-axis weights and per-axis scoring formulas are governance-tunable
// at the parameter layer; only the axis identity is doctrinally fixed.
var CanonicalRecursiveAxes = []string{
	"substrate",
	"verification",
	"classification",
	"attribution",
	"tooling",
	"interface",
}
