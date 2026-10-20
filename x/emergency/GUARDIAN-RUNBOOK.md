# Guardian halt reasons and research-fund governance

This is the guardian-facing contract for PR #125. It does not authorize an
upgrade, proposal, vote, broadcast, or release of an emergency transition hold.

## Reason vocabulary

After `HaltReasonScopingEnabled` is selected, the exact reserved reason
`research_fund` (Go constant `types.HaltReasonResearchFund`) is positive evidence
of research-fund misuse. The `propose-halt [reason]` command still accepts free
text; no existing proposal or retained reason is rewritten.

A finalized halt with any other reason, including `research_fund_misuse`,
`research fund misuse: multisig drained`, case variants, trailing spaces and
apparently unrelated prose, is unclassified. Within the current phase it blocks
forward phase progress, but it does not justify rollback. Malformed evidence has
the same conservative result. This deliberately avoids the reviewed fail-open
policy. It can hold governance on an unrelated halt; do not describe that cost
as a structured category system or as Yu accepting exact-token fail-open rules.
Record supporting narrative in the incident process, not as a suffix to the
reserved token. A future structured reason/category needs separate schema and
policy review.

## Height and state contract

Only finalized halt ceremonies with StartBlock in the inclusive interval
`[PhaseStartedAtBlock, current height]` contribute. StartBlock is initiation,
not finalization. A phase start of zero covers all history. Rollback/re-advance
updates the phase boundary. Non-finalized ceremonies are not evidence.

The forward query's `EmergencyHaltsFromMisuse` value is a conservative gate
metric: one can mean unknown evidence, not a verified incident count. Rollback
returns an evidence error rather than converting that sentinel to justification.
At activation time pending rollbacks are revalidated; invalid targets or missing
or unknown evidence cancel the pending transition with the existing cancellation
event. The marker-off path retains legacy all-history/all-reasons behavior.

## Carrier / relaunch checklist (not implemented by this lane)

- Independently review this exact candidate and the reason-policy trade-off.
- Census retained ceremonies, including imports without electorate snapshots;
  those legacy records can bypass newer genesis payload validation.
- Inspect pending transitions and rollbacks, their recorded target and evidence.
  Stored ConditionsSnapshot counts remain historical; runtime revalidation is
  not a replacement for the carrier's state census.
- Preserve the emergency transition hold. No enabling step releases it, changes
  balances, redistributes funds or rewrites ceremonies/audit records.
- The named `consensus-repairs-v1` carrier owns migration/version coordination.
  Enable only in its reviewed cache and assert the marker and both predicates.
- Foundation genesis has `halt_reason_scoping_enabled`; DefaultGenesis is true,
  an old genesis without the field imports false. Use module InitGenesis and
  ExportGenesis, not only the keeper's pinned legacy genesis methods. A relaunch
  must explicitly choose the field and assert it after InitGenesis.
- The older operations snapshot and upgrade-prestate selectors omit key 0x14.
  Their success is not evidence the marker survived; add carrier assertions.

Marker read errors wrap `activation.ErrMarkerFault` and propagate before writes
from SubmitLIP, the governance query, and the module BeginBlock hook. Evidence
errors are ordinary gate failures, not consensus-fault panics.
