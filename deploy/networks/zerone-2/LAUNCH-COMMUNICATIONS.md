# `zerone-2` evidence-first launch communications

Status: **internal preparation; public launch NO-GO**

This plan turns attention into informed participation. It does not authorize a
post, announcement, endpoint, token promotion, validator invitation, or network
launch. Public communication must follow the evidence order in
[`CUTOVER.md`](CUTOVER.md). The exact transition notice must be public before
CUTOVER is signed, but the current artifact graph has no earlier signed
decision authorizing that publication. The pre-notice and CUTOVER phases
therefore remain NO-GO until a distinct main-key decision binds the exact notice
bytes, URL, deadline, and narrow publication-only scope. Only OPEN-BETA may
publish live `zerone-2` coordinates.

## Persuasion standard

Zerone should persuade with inspectable evidence and a specific invitation,
not urgency or implied certainty:

1. state what exists now;
2. show the artifact that proves it;
3. disclose what remains centralized, disabled, or unknown;
4. ask the audience for one concrete contribution; and
5. publish corrections with the same prominence as the original claim.

Do not use scarcity, price, return, inevitability, decentralization, Byzantine
resilience, production-readiness, or migration language unless the exact
signed evidence makes that statement true. Never describe the custom-staking
census as a balance migration, validator roster, claim entitlement, or launch
authority.

## Audiences and honest invitations

| Audience | Lead with | Ask for | Evidence they should receive |
| --- | --- | --- | --- |
| Validator operators | Reproducible daemon and failure behavior | Independently reproduce the four-node rehearsal and review custody isolation | Exact commit, commands, logs, AppHash, fault matrix, binary provenance |
| Protocol and security engineers | Explicit authority gaps and fail-closed gates | Review one named gate or submit an adversarial test | Authority graph, test source, signed release inputs, open blocker IDs |
| Agent builders | Witness-and-record purpose and deliberately dark surfaces | Build against local fixtures; identify the smallest useful read-only flow | API/schema docs, local chain recipe, disabled-surface disclosure |
| Researchers and stewards | Truth claims separated from consensus and custody claims | Challenge the model, threat assumptions, and governance boundary | Doctrine pins, trust map, architecture record, correction history |
| General community | A clear account of what can be observed today | Verify published hashes and follow readiness updates | Human summary linked directly to machine-verifiable artifacts |

No audience should be asked to send funds, import a key, run an unreviewed
binary, connect to an unpublished peer, or rely on a rehearsal artifact as live
network state.

## Claim-to-proof ledger

Every outward-facing claim must link to its proof in the same message or on the
first linked page.

| Permitted claim | Minimum proof | Current posture |
| --- | --- | --- |
| "The daemon builds and its unit/integration gates pass" | Exact source commit and CI run for that commit | Candidate evidence only until remote CI passes |
| "Four local validators reached consensus" | Retained four-node report with equal AppHash and exact binary/source provenance | Rehearsal claim only |
| "The network continues with one validator offline" | Recorded 3/4 continuation window and heights | Rehearsal claim only |
| "The network halts without two-thirds voting power" | Recorded 2/4 halt window, then recovery evidence | Rehearsal claim only |
| "One signed message commits once and exact replay is rejected" | Committed MsgSend bytes/proof, one sequence increment, replay response, and unchanged post-replay state | Rehearsal claim only |
| "Legacy custom-staking custody was inventoried" | Self-sealed census at application height `A`, AppHash `E` recomputed from all multistore roots, transition-signed release-bound execution evidence, FINAL report/evidence/signature hash binding, and independently verified census-binary provenance | Production evidence absent; the transition signature is an attestation, not mechanical execution proof |
| "A release candidate exists" | Signed tag, reproducible binary, SBOM, provenance, vulnerability decision, immutable images, and passing release gates | NO-GO |
| "`zerone-2` is live" | Signed OPEN-BETA decision and initiation evidence, committed history link, published immutable coordinates, and passing public probes | NO-GO |
| "`zerone-2` is decentralized or Byzantine-resilient" | At least four independent failure domains and no actor with one-third voting power, plus the applicable source and operational gates | Prohibited at initial custodial beta |

Screenshots, dashboards, unpublished logs, a Git branch, and a passing local
test are supporting context; none substitutes for the required signed artifact.

## Attention sequence

The sequence is gate-driven, not date-driven. Before a phase's initiation
event, a failed gate leaves the process in the preceding safe/private phase.
After initiation, operators must follow that signed phase's exact
completion/abandonment boundary; after the old chain freezes, recovery is
forward-only and must never be described as a return to an earlier phase.

### 0. Engineering notebook — now

- Publish nothing automatically.
- Prepare short, reproducible notes about one result at a time: consensus
  liveness, deliberate halt, recovery, replay rejection, or census verification.
- Label every item `LOCAL REHEARSAL`, include the exact commit, and list the
  production blockers beside the result.
- Invite review of a narrow artifact rather than broad endorsement.

Success is measured by independent reproductions, useful findings, and closed
blockers—not impressions or follower counts.

### 1. Release-candidate briefing — after RELEASE verifies

- Publish the signed release identity, reproducible build instructions, SBOM,
  vulnerability decision, immutable images, and explicit NO-GO/GO status.
- Give reviewers a bounded verification path that does not require a secret or
  a live production endpoint.
- Hold a technical walkthrough only after the packet is available for advance
  inspection.

RELEASE does not authorize a live-network announcement or the old-chain
transition.

### 2. Transition pre-notice — blocked on distinct signed authority

- Add and independently review a main-key pre-notice decision that binds the
  exact notice bytes, destination URL, publication deadline, and
  publication-only scope. A later CUTOVER decision cannot retroactively
  authorize its own prerequisite.
- Only after that decision verifies, publish the exact notice stating the
  proposed `F/A/H` boundary, observation window, expected service changes,
  custody model, and abort/forward-only boundaries.
- Capture byte-bound publication evidence, then complete and sign CUTOVER as
  specified by the operational contract.
- Maintain one timestamped status page; corrections append rather than silently
  replacing earlier claims.

Do not publish successor peers, endpoints, DNS, or a "live" claim in this phase.

### 3. Open beta — only after OPEN-BETA initiation verifies

- Publish the immutable genesis/release hashes, history-link transaction,
  validator/edge/gateway roles, public coordinates, and current trust limits.
- Lead with "public custodial beta" while one operator controls consensus.
- Provide separate paths for observers, independent full-node operators, and
  security reports. Keep hosted transaction submission and public gRPC absent
  unless separately authorized.

### 4. Post-launch record

- Publish the evidence bundle and a plain-language postmortem, including every
  deviation and near miss.
- Report uptime together with validator concentration and failure-domain count.
- Maintain a public blocker ledger for claims that remain unavailable.

## Message shape

Use this structure for a candidate update:

> **State:** Local rehearsal / release candidate / transition / open beta
>
> **Observed:** one precise result
>
> **Verify:** exact commit, artifact hash, and command or signed packet
>
> **Limits:** what this result does not prove
>
> **Invitation:** one bounded review or reproduction task

Example for the present phase:

> **State:** LOCAL REHEARSAL — not a network launch.
>
> **Observed:** four disposable validators converged; 3/4 continued, 2/4 halted,
> and the restored set recovered.
>
> **Verify:** use the exact source commit and retained rehearsal report linked
> from the engineering handoff.
>
> **Limits:** this does not prove production custody, independent validator
> operation, release provenance, or `zerone-2` activation.
>
> **Invitation:** reproduce one fault transition and compare the final AppHash.

Replace all placeholders with immutable values before use. Never reuse an old
success statement after the source commit or artifact bytes change.

## Response and incident discipline

- One named status record is authoritative during a launch window.
- Before external publication, designate a private security intake path in a
  separately authenticated and approved operator notice; the current RELEASE
  schema does not bind that field. Do not ask reporters to disclose an exploit
  publicly.
- A discrepancy in hashes, authority, supply, validator identity, or history
  link changes the public state to `NO-GO` or `INCIDENT` immediately.
- State what is known, what is being checked, and the next update time. Avoid a
  root-cause claim until evidence supports it.
- Preserve the original statement and append the correction/postmortem.

## Before any message leaves the workspace

- [ ] A non-circular signed decision already authorizes the exact communication;
      a later phase may not retroactively authorize its prerequisite.
- [ ] Every technical claim has a direct proof link and exact commit/hash.
- [ ] The trust, custody, validator concentration, and disabled-feature limits
      are visible without opening a second document.
- [ ] The message makes no token-price, return, scarcity, or investment claim.
- [ ] A second reviewer reproduced the cited result or verified the signed
      evidence independently.
- [ ] The owner has approved the actual external post and channel.
