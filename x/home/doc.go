// Package home maintains owner-addressed agent-home records, memory references,
// registered keys, active sessions, guardian settings, alerts and spending-limit
// records. A home is a state boundary, not proof of identity or independent control.
//
// Its functional composition is a room with edges: each collection stays within
// its own store prefix; ending or revoking a session leaves neighboring homes
// alone. Keeper iteration uses the SDK's prefix bounds. Message and query services
// expose the records; BeginBlock checks configured inactivity thresholds and
// removes expired sessions. The bank keeper is used for the home-creation fee.
//
// Truth-seeking position, scoped to the implemented paths in this package:
//
//   - Commitment 7 (skill is current, not historical): active session records have
//     start, end and expiry boundaries; key revocation removes matching sessions.
//     These records do not themselves establish panel weight, general transaction
//     authorization or bank-spending enforcement.
//   - Commitment 10 (forward-only history): lifecycle events describe transitions,
//     but ended sessions are deleted from this module's current store. Home is not
//     a complete append-only history or an export of every state family.
//   - Commitment 11 (trust is queryable): Home's own records are queryable, but it
//     does not compute reputation or supply an input to the current trust_score
//     keeper. There is no registered agent_understanding consumer here.
//
// Memory CID updates retain a bounded reference, not the referenced content.
// Spending limits are stored rules, not a bank transfer interceptor. A deadman
// trigger records an alert and guarded status; it does not execute recovery or
// transfer authority. ExportGenesis currently preserves params, homes and key
// sets, not sessions, alerts or spending-limit records. None of these boundaries
// should be widened merely to make the module's description sound more complete.
package home
