package types

// K-alpha karma recognition: events only. The ledger that prices edges is
// K-beta; recognition precedes pricing, so no edge here carries a magnitude.

// MaxCitedFactsPerLink bounds len(link.cited_facts) twice: at admission
// (ValidateLink — reject, never truncate) for new links, and again inside
// emitExternalCiteKarma at settlement, because attestations admitted under a
// pre-cap binary settle after the upgrade with whatever fan-out they carry
// (K-alpha ships no migration). Each cited fact costs one knowledge-store
// read and one karma event at settlement, which runs in the BeginBlocker
// drain; caller-supplied fan-out on that path must be bounded (the
// axis-bounds lesson again). Const at K-alpha; K-beta lifts it into Params
// so governance can move it.
const MaxCitedFactsPerLink = 16

// The karma edge event. One type across the chain; legacy string events,
// matching the codebase's universal event style. The emitter builds the
// event from literal strings so the static event audit can scan it; these
// consts exist for consumers and tests, and must stay equal to the literals.
const (
	EventTypeKarmaEdge = "zerone.karma.edge"

	AttrKarmaBeneficiary  = "beneficiary"  // bech32 — who is recognized
	AttrKarmaKind         = "kind"         // e.g. "external": relied on by an external system
	AttrKarmaState        = "state"        // "RECOGNIZED" | "ORDINAL"
	AttrKarmaCounterparty = "counterparty" // bech32 or empty — who occasioned the recognition
	AttrKarmaRefID        = "ref_id"       // the object recognized (a fact id here)
	AttrKarmaDomain       = "domain"       // may be empty
	AttrKarmaSelf         = "self"         // "true" iff beneficiary == counterparty
	AttrKarmaRegister     = "register"     // constant confession — see KarmaRegisterPricedCoherence

	KarmaKindExternal = "external"
	KarmaStateOrdinal = "ORDINAL"

	// KarmaRegisterPricedCoherence rides every karma edge (design §4,
	// doctrine X-5): karma records priced coherence and priced reliance,
	// not truth.
	KarmaRegisterPricedCoherence = "priced-coherence"
)
