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
the same conservative result. Neither unknown nor malformed evidence vetoes
rollback justified by gridlock or a valid `research_fund` halt. Unknown evidence
alone still holds the phase; it is not silently classified as misuse or ignored
by the zero-tolerance forward gate. This is not a structured category system.
Record supporting narrative in the incident process, not as a suffix to the
reserved token. A future structured reason/category needs separate schema and
policy review.

## Height and state contract

Only finalized halt ceremonies with StartBlock in the inclusive interval
`[PhaseStartedAtBlock, current height]` contribute. StartBlock is initiation,
not finalization. A phase start of zero covers all history. Rollback/re-advance
updates the phase boundary. Non-finalized ceremonies are not evidence.

The evidence API returns the valid count and an independent unknown flag, not an
error. The forward query's `EmergencyHaltsFromMisuse` is the valid count plus one
if unknown evidence exists (saturating at uint64 max), not a verified incident
count. Rollback succeeds on gridlock OR a positive valid count. Only when neither
exists does unknown evidence return `ErrHaltEvidence` (codespace `emergency`, ABCI
code 21); without unknown evidence the rejection is `ErrNoRollbackJustification`.
Pending rollbacks use the same rule at execution, plus current-phase/target
validation. Unsupported rollbacks cancel with the existing event; unknown
evidence never cancels an otherwise justified rollback. The marker-off path
retains legacy all-history/all-reasons behavior.

## Carrier / relaunch checklist (not implemented by this lane)

- Independently review this exact candidate and the reason-policy trade-off.
- Census retained ceremonies, including imports without electorate snapshots;
  those legacy records can bypass newer genesis payload validation.
  Envelope decoding and key/ID checks precede the type and height filters: one
  malformed envelope or mismatched key/ID anywhere in history, including a
  non-halt record, keeps the forward gate closed. Malformed halt payloads are
  relevant only for finalized, in-window halts. Enumeration continues past bad
  records; surfaced iterator open/traversal/close faults set the unknown flag
  without discarding valid records already read. An open failure has no valid
  count, but still cannot veto independent gridlock. The SDK iterator helper
  cannot prove the absence of hidden parent traversal errors; census committed
  state independently before enabling. This does not assert live corruption.
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
scanning reads no marker and cannot return a marker fault. There are no secondary
marker-fault panics in the forward counter or pending-rollback loop.
