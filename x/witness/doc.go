// Package witness is a dormant, source-only consensus-carrier scaffold for
// kingdom.witnessed-agent-economy/0.1.
//
// It is intentionally not a Cosmos SDK module: there is no AppModule,
// registration, genesis state, message service, CLI, or store implementation.
// Current readiness marks every closed record kind NOT_CONSENSUS_ADMISSIBLE,
// so admission always stops before controller authorization or state mutation.
package witness
