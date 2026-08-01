// Package claiming_pot implements Zerone's bootstrap-claim emission pathway.
//
// docs/TRUTH_SEEKING.md commitment 20 says post-genesis issuance follows
// participation and genesis privilege must be disclosed. The live zerone-1
// genesis created 13,555 ZRN of operator-controlled validator
// collateral/gas and transferable operations float. Beyond that disclosed
// scaffolding, bootstrap claims and substrate-bridge rewards are enabled
// cap-gated issuance families; configurable knowledge bounties and token
// emissions are default-off. Vesting-rewards v2 retires the former automatic
// PoT block mint. This package owns only the bootstrap path.
//
// The structural boundaries are:
//
//   - Claim routes through vesting_rewards.MintWithCap. Tokens are minted into
//     the claiming_pot module account and forwarded to the claimant in the same
//     transaction; the module is a conduit, not a pre-funded treasury.
//   - Each bootstrap entry is a one-address pot for 222,000 uzrn (0.222 ZRN).
//     Bootstrap pots do not auto-expire and become terminal only after claim.
//   - MsgAddBootstrapEntry accepts either the gov authority or the optional
//     Params.BootstrapRegistrar. The published zerone-1 genesis sets the
//     registrar to the operator operations address; that is a real custodial
//     admission power, not community governance.
//   - Registrar admission is bounded by BootstrapDailyAdmissionCap and the
//     lifetime BootstrapEmissionCapUzrn. Governance admission bypasses the
//     daily window but not the lifetime emission cap. Governance can revoke
//     the registrar by setting it to the empty string.
//   - Duplicate entries are skipped and an over-cap batch fails atomically.
//     No entry consumes mint headroom until its participant claims.
//
// This is commitment 20's narrow promise: post-genesis ZRN emitted through
// this module follows the admitted participant's claim and shares the global
// hard-cap check. It is not a promise that admission itself is permissionless,
// decentralized, or governance-only. Operational onboarding from this source
// head remains paused until a release-bound packet re-authorizes it.
//
// Refusal voice:
//
//   - ErrCapReached: "bootstrap mint refused (commitment 20:
//     issuance follows participation, hard cap reached)".
//
// Events carry creed_commitment="20" so observers can distinguish bootstrap
// issuance and admission from the other cap-gated pathways.
package claiming_pot
