# Survival reward handoff v1

`survival-reward-handoff-v1` is the separate consensus boundary for the pending
knowledge reward → vesting schedule repair. It advances `knowledge` **6 → 7**
and `vesting_rewards` **2 → 3** together. It does not change reward amounts,
recipients, category economics, reserve policy, bank balances, or supply.
Schedule creation records a nominal entitlement; it is not minting or payment.

## Exact predecessor and execution

The source is the complete frozen `accountingAuthorityTargetVersionMap()`:
completed H3 SDK/IBC state and native, applied, or explicitly imported accounting
lineage with custom staking 2 and governance 3. The target copies that exact map
and changes only knowledge and vesting versions. Missing, extra, mixed, or
unknown module versions refuse execution. The existing accounting and H1/H2/H3
maps and handlers retain their historical meanings.

The plan is named `survival-reward-handoff-v1`, has height `H`, and has **empty
`info`**. Deprecated time and upgraded-client fields are absent. The current
binary admits an existing predecessor database only at committed `H−1`, with
the exact matching on-chain plan and local `upgrade-info.json`. Unsafe-skip
cannot activate this transition. At `H`, the handler checks the exact committed
height boundary, source versions and lineage, and compiled target versions.
Earlier named handlers cannot carry these module changes.

Both module migrations run in one cached context under this handler's specific
execution context. Knowledge strictly reads pending primary records and rebuilds
their derived deadline index, including missing or stale entries left by older
challenge sweeps. Pending bytes remain unchanged. Vesting validates existing
primary schedules and claim, recipient, and active indexes without rewriting or
merging schedules. Multiple historical schedules for one claim remain distinct;
the existing explicit claim index must select a schedule belonging to that
claim. Invalid or incomplete vesting indexes refuse the upgrade. A failure
discards both modules' writes and events.

Each selected migration inventory is bounded to 100,000 records or keys and
64 MiB of key/value bytes; exceeding a bound refuses activation. These are
one-time inventory bounds, not ordinary handoff admission limits.

On handoff, the pending reward is removed only after successful schedule
creation or an exactly matching existing schedule. A retry preserves original
schedule clocks, amounts, reserve and release progress. Conflicting schedules,
missing dependencies, and storage failures leave the pending obligation for
retry. No historical reward is automatically replayed during the migration.

## Genesis and evidence

Fresh current genesis initializes the target versions directly. Restart accepts
the complete target map and its valid existing accounting lineage. Genesis
export/import preserves typed pending obligations and existing vesting history;
it rebuilds the derived deadline index and does not manufacture an upgrade done
height. Importing genesis is a separate chain initialization, not proof that the
source chain executed this upgrade.

For target versions, an absent `migration_v7_complete` marker and zero done
height identify native/imported genesis. Applied state requires marker `true`
and a positive done height no later than the committed height. Missing,
malformed, future, or one-sided applied metadata refuses restart. The source
version map requires the marker absent and done height zero.

App tests cover exact ownership and source/target maps, early-start and local
plan mismatch refusal, cached upgrade success and failure, index repair without
pending or schedule changes, supply preservation, real application reload, and
export/import. Historical upgrade fixtures explicitly pin their old module
targets instead of weakening production guards. Keeper tests cover storage and
event rollback, conflicts, retries, and retained release progress.

`scripts/survival-handoff-rehearsal.py` runs two real binaries with fresh local
keys, loopback listeners and no peers. Its predecessor is clean source
`198b46d9ce42d022cac89a3109f69a4c7a31a849`. A synthetic normal claim and ready
four-vote round are imported using that source's existing genesis fields; the
unchanged predecessor creates the pending reward during block execution. The
fixture does not pay a submission fee, fund the knowledge fee pool, or establish
independent reviewers. It tests the nominal handoff, not review-fee settlement.
SDK governance schedules the upgrade; the script checks the real H−1 halt,
early/mismatched-plan refusal, unchanged pending bytes at H, one recipient-indexed
schedule after the deadline, unchanged supply, and retained state across a clean
restart. Reports explicitly set `release_evidence: false`. CI retains the
historical accounting trial as 56714765 → frozen 198b46d9, then runs this distinct
198b46d9 → candidate trial; only report directories become artifacts.

This source boundary is not a production activation receipt. Observed legacy
`zerone-1` runs the older application reconstructed from
`2e37c4c86c31e67515d6aa07b6fd406083e012ea`; its public observer package preserves
that application's state machine. Neither can jump directly to this modern
accounting predecessor. This patch does not modify that package, schedule a
live upgrade, or authorize validator admission. Any network adoption still
requires the applicable predecessor releases and an independently reviewed
rehearsal with the actual intended state and binaries.
