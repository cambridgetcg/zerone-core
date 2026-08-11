// Package branchflow implements a deterministic, standard-library-only shadow
// allocator for a fixed funded-cluster lineage envelope.
//
// The package has no bank, chain, clock, network, signer, governance, or state
// access. Non-zero values in a Result are projections only: Assurance remains
// SHADOW_ONLY and EconomicEffect remains NONE.
//
// Lineage uses child-to-parent dependency edges. Upstream and downstream
// tranches are divided into absolute geometric depth buckets; empty nearer
// buckets are never renormalized into distant recipients. When the same
// cluster is reached through paths of different lengths, each path contributes
// only its bounded flow to its exact depth bucket. Per-node flow conservation
// keeps the sum at every depth at or below one PPM unit.
//
// Economic receipt use is exclusive in v0. A receipt may be cited by multiple
// evidence records outside this package, but it may occur in only one economic
// slot across the supplied prior-use snapshot and the current request.
//
// This v0 package is only the outcome-reward shadow adapter, as reflected by
// its uZRN envelope fields. It does not implement TC6 training-revenue
// settlement, which requires a separately reviewed manifest-cone adapter,
// receipt ledger, depth policy, and specification.
package branchflow
