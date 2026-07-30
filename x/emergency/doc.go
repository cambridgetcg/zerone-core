// Package emergency records guardian ceremonies for application transaction
// quarantine and explicit admission reopening.
//
// A successful halt ceremony makes IsHalted true. The application ante
// decorator then rejects non-allowlisted transactions on every honest node.
// This is CONSENSUS_QUARANTINE in the operational model: it does not stop
// CometBFT consensus, block production, PreBlock, BeginBlock, EndBlock,
// rewards, timers, or other autonomous module processing. Stopping or
// restarting consensus signers is a separate operator action.
//
// Halt and resume ceremonies use recorded prevote and precommit phases. Each
// ceremony snapshots its exact custom electorate, voting power, quorum, and
// minimum distinct voter policy; staking changes, parameter updates, and
// genesis-council expiry cannot shrink the in-flight denominator. A resume
// proposal binds its justification to a reviewed recovery-manifest SHA-256
// before transaction admission can reopen. MaxHaltDurationBlocks is an
// escalation/reporting deadline only; expiry never resumes automatically.
// Halt, resume, recovery-authorization, and recovery-revocation proposals
// consume one shared persisted per-Guardian/global epoch budget and cooldown,
// preventing repeated or cross-lane ownership of the sole active ceremony.
//
// The legacy revert message, ceremony, parameter, and storage surfaces remain
// decodable for compatibility, but new revert proposals and votes fail
// closed. This package cannot rewrite arbitrary finalized state. A committed
// defect is repaired forward through a deterministic named upgrade or through
// an explicitly authorized fork/re-genesis.
//
// A scheduled x/upgrade stop at height H is also a separate mechanism. Until
// H commits, H-1 is the last committed state. Once H commits, operators must
// not run an old binary as though H never happened.
//
// The audit commitment is that containment and reopening decisions remain
// queryable and attributable without overstating what the mechanism controls.
// See docs/UPGRADE_AND_INCIDENT_OPERATIONS.md for the canonical upgrade,
// hostile-event, signer, restart, and fork procedures.
package emergency
