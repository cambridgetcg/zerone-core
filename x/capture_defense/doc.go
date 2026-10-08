// Package capture_defense stores verification observations, legacy reputation
// scores and per-domain concentration metrics. A flag is an input to existing
// capture-response rules, not proof of collusion or independent controllers.
//
// Current source wiring (app/app.go):
//   - knowledge records history and reputation through KnowledgeCaptureDefenseAdapter;
//     capture_defense also reads knowledge's verification activity.
//   - ontology supplies domain depth for the existing HHI threshold.
//   - capture_defense pushes metrics/evidence into automatic capture_challenge cases
//     during flagged-domain analysis; rejected challenges clear capture flags.
//     capture_challenge's RunRiskAnalysis is a no-op, and the challenge adapter's
//     GetCaptureMetrics accessor has no production caller.
//   - alignment supplies analysis pacing. The reverse flag-count input to
//     alignment's security sensor is deliberately NOT wired; connecting it would
//     change committed scores, pacing and LIP timing.
//
// BeginBlocker retains reputation decay and periodic analysis. These paths and
// their consumers need an explicit protocol boundary before retirement; shortening
// this description does not disable them. Unwired partnerships hooks remain
// optional code, not a registered partnership module or an active reward path.
//
// The query service is defined in proto/zerone/capture_defense/v1/query.proto;
// client/cli exposes its parameters, reputation, metrics, cross-stratum requirements
// and flagged domains. QueryStructuralImmunity is an unregistered Go helper, not
// an additional RPC. The shared purpose/connection map is dashboard/module-guide.json.
// Source registration and available queries do not establish live-network adoption.
package capture_defense
