# Existing-ledger census — 2026-09-09

Status: **committed-state inspection of the observed `zerone-1` network;
not an independently trusted checkpoint, production release, or complete
historical claims reconciliation.**

The official legacy custom-staking census passed at application height
**1,261,575**. Its complete custom ledger contains **zero claims and zero
custody balance**. A separate complete native-bank inventory found **16 native
balance records summing to 13,667,372,296 uzrn**, exactly equal to the stored
native supply. Neither check found an amount to repair within its scope.

These are findings about an existing committed state. The
[accounting repair](../specs/accounting-authority-v1.md) remains prospective
source, and the [authority design](../AUTHORITATIVE-STATE.md) remains an
uncompleted transition. An empty custom ledger does not establish that every
other module is free of obligations or that the network adopted those changes.

## State identity and acquisition

| Evidence | Exact observation |
| --- | --- |
| Application height | `1261575` |
| Post-commit AppHash | `a381775589d84a74bbf365b9b2add26201465e20980cf50a53544f7a81256106` |
| Binding block-header height | `1261576` |
| Public genesis file SHA-256 | `c30a523b9764fb76c84a53d99fcdabb966d16e7a4d3f15426ab7af5e8576170e` |
| Observed running executable SHA-256 | `94d76a0a2a8dc6667e1c6ae504d37e0f5874e64b9026aabff35bb22978667ea8` |
| Reviewed census source | `b0168835d161cff5de209c996d1055648dee4ca7` |
| Official census executable SHA-256 | `1da55ac52c343f8490b2ca63a18889ce3bdfca7a63fd8e935913baf20bc920f7` |
| Copied database canonical file-manifest SHA-256 | `96fe248bd271e86826c7f17a2b1c3e0c0163dcaaa5b9086ea542215a291052a1` |
| Copied database canonical snapshot SHA-256 | `1089f33e72b85fe9f21fffdc01945c066cb747bb3db78719013bbbefe0a12f9a` |

The active executable was read separately from the running process and disk,
with matching hashes. Its embedded build information identifies Go 1.24.13,
Cosmos SDK 0.50.15, IBC-Go 8.8.0 and the CometBFT 0.38.20 dependency. It has no
usable source revision: the census-source pin above is the reader's provenance,
not a reconstructed source identity for that legacy executable. The RPC's
CometBFT `0.38.19` string agrees with the version constant actually embedded in
the 0.38.20 dependency; that apparent version discrepancy is resolved.

A fresh observer used this executable and the exact public genesis, with new
identities and zero voting power. It restored a publicly offered snapshot and
caught up with consensus. It had no published ports, production keys or public
services. After clean shutdown, only the complete `application.db` was copied:
28 regular files, 48,312,662 bytes. Source/copy manifests before and after
archiving, local extraction, and the accounting reads matched exactly. The
inspection opened the disposable copy with backend-enforced read-only access.

The canonical commits, validator-set hashes, signatures, block hashes and
consecutive block links were verified. The stopped observer's ABCI root agreed
with the public header at H+1 and the copied database's rootmulti commitment.
This authenticates consistency with the **observed sole-validator network**.
Both RPC aliases lead to the same upstream; they are not independent witnesses.
The previously recorded signer-custody failure remains unresolved, so these
checks do not establish an independent trust anchor or restore signer custody.

The public HTTPS gateway accepts GET reads but rejects the JSON-RPC POST reads
used by CometBFT's light client. The observer therefore used the already exposed
direct RPC. This is a client-compatibility finding, not evidence of an alternate
network or a reason to infer independent witnesses.

## Custom-staking and native-bank results

The official `legacy-v1` census checked all leaves in `zerone_staking`, `bank`
and SDK `staking`, with complete traversal counts, per-leaf membership proofs,
store roots and the externally supplied application root.

| Store | Complete leaf count | Finding |
| --- | ---: | --- |
| `zerone_staking` | 6 | Four tier configurations, one parameter record, one initialization sentinel; no validators, delegations, unbondings or claimant indexes. |
| `bank` | 36 | Sixteen primary native balances; supply and other bank namespaces also traversed. |
| SDK `staking` | 10,010 | One bonded validator with 11,111,000,000 tokens; custom census does not reconcile all SDK claim semantics. |

Custom custody obeys `B = D + U = 0`. All four stored tier configurations match
their parameter copies. This is a complete current custom-staking claim census,
not a claim about every historical transaction or another module's liabilities.

A separately built and independently reviewed private derivative enumerated
every primary native bank balance, without depending on account discovery or
assuming that an address has an auth record. Its total is
`13,667,372,296 uzrn`, stored supply is `13,667,372,296 uzrn`, and the difference
is **zero**. No other denomination's primary balance or supply was present.
It reused the frozen read-only backend and complete proof scanner; its added
enumerator, source hashes, binary, tests and execution receipt are retained
separately from the official census.

The earlier selected-module query at H1,261,513 covered only part of the bank
ledger. Its residual was an unenumerated remainder, not missing coins. The
complete H1,261,575 inventory closes that enumeration gap. Different heights
remain separately labelled, and native balance equality does not prove
beneficial ownership, circulation, or full backing of each module's claims.

## Complete state inventory and actual custody

A second independently reviewed private derivative traversed **all 37 mounted
stores**, retaining every raw key/value pair. It verified **88,051 leaves**
against their committed roots, with 15,943,766 logical input bytes. Its scanner,
root verifier and read-only backend were unchanged; its added collector required
every mounted store's version to equal H and bounded retained output. Complete
raw-state enumeration is distinct from interpreting every module's accounting.

All 22 primary auth accounts were decoded from those committed bytes; 13 are
stored module accounts. Every native-balance address has a stored auth account.
The seven nonzero module balances and the other nine balances partition the
complete native inventory at this same height:

| Stored account classification | Native balance, uzrn |
| --- | ---: |
| SDK bonded pool | 11,111,000,000 |
| Distribution | 58,025,745 |
| Development fund | 37,804,015 |
| Vesting rewards | 16,553,211 |
| Knowledge | 7,313,952 |
| Research fund | 6,268,516 |
| Protocol treasury | 1,188,000 |
| Nine non-module auth accounts, combined | 2,429,218,857 |
| **Total** | **13,667,372,296** |

These classifications identify committed ledger accounts, not beneficial owners.
The module names and permissions come from stored auth records, with address-key
and standard module-address agreement; query-generated defaults are excluded.

Under the frozen source key schemas, the complete current stores contain:

- **Qualification:** parameters, an endorsement counter and initialization;
  no qualifications or endorsements.
- **Liquidity pool:** parameters and initialization; no pools or TWAP records.
- **Custom governance:** parameters, counters, funding-transfer history and a
  research-phase record; no LIPs, votes, research-spend proposals, seat elections
  or phase-transition proposals. Funding-transfer history is not a per-staker
  governance claim ledger.
- **SDK governance:** six historical proposals marked `PASSED`, with no current
  deposit or voting proposal among them. Their retained total-deposit fields
  are historical aggregates, not six additional present refund claims.
- **Knowledge:** 27 claims with stored stake fields of 200,000 uzrn each;
  25 accepted and two insufficient. All 27 associated rounds are complete.
  There are 22 active domain records; all proposer fields are empty. Completed
  status and a retained stake amount do not prove payment or an unpaid claim;
  knowledge's 7,313,952 uzrn still needs settlement-history reconciliation.

The recorded upgrade history has five completed upgrade keys, ending with
`agenttool-seam-v1` at H527,603. It contains no H1/H2/H3 or
`accounting-authority-v1` completion record. Stored custom versions remain
`zerone_staking=1` and `zerone_gov=2`. This ledger has not established the exact
post-H3 predecessor required by the merged accounting handler; deploying that
handler directly is not a validated next step. State interpretation uses the
recorded reader schemas and does not recover the legacy executable's missing
source provenance.

## Funded records reconciled at the same checkpoint

The separately reviewed SDK staking inventory also passed. It found one bonded
validator and one primary self-delegation. Delegation shares sum exactly to
the validator's `11,111,000,000.000000000000000000` shares. Bonded token
obligations equal the bonded pool's **11,111,000,000 uzrn**. There are no primary
unbonding delegations or redelegations; the not-bonded pool's balance and primary
obligations both equal zero. The separate committed-auth interpretation confirms
both pool identities. Secondary indexes, queues and withdrawal readiness remain
outside this primary-obligation check.

Distribution's canonical module-balance equation passes exactly, using the
deployed SDK 0.50.15 accounting rules and decimal precision:

```text
community pool                 1,160,514.90 uzrn
validator outstanding rewards 56,865,230.10 uzrn
                             -------------
distribution bank balance     58,025,745.00 uzrn
```

Outstanding rewards also equal current rewards `51,178,707.09` plus accumulated
commission `5,686,523.01`. Commission is already included in outstanding rewards;
adding it again to the module equation would invent a deficit. These checks do
not calculate a recipient's rounded withdrawal or authorize disbursement.

Vesting has **25 active schedules**, each with nominal total `200,000`, released
`0`, stored claimable `170,000` and reserve `30,000` uzrn. The recipient, claim
and active indexes match all 25 schedules. At this height the captured category
configuration's release cap has been reached:

- Total nominal schedule amount: **5,000,000 uzrn**.
- Remaining recipient release under the captured configuration: **4,250,000 uzrn**.
- Included reserve component: **750,000 uzrn**, not an additional recipient claim.
- Vesting bank balance: **16,553,211 uzrn**.

The balance covers the nominal schedules. The **11,553,211 uzrn above their
nominal total remains unclassified**; it is not declared freely spendable or
assigned to a new claimant. These are current-record arithmetic checks, with
legacy source-provenance and historical settlement limits retained.

All 25 schedules link to the 25 accepted knowledge claims and matching
recipients. Their creation heights equal the associated facts' challenge-window
ends, exactly 34,272 blocks after the recorded verdicts; the captured challenge
duration is also 34,272. This is consistent with the source's
[survival-window release mechanism](../../x/knowledge/keeper/survival_escrow.go).
The stage timestamps agree under that interpretation; they do not establish
historical payment receipts or the unknown binary's source identity.

## Authority and reconciliation priorities

The evidence shifts attention from repairing a hypothetical custom-staking
deficit to the funds and authority that actually exist:

1. **Keep the reconciled staking baseline intact.** Custom claims, SDK primary
   delegation shares and both SDK pool equations pass at this checkpoint.
   Preserve those individual records and exact equations in later rehearsals;
   complete secondary-index, queue and withdrawal checks before an operational
   transition. There is no demonstrated staking deficit to repair here.
2. **Finish funded module settlement evidence.** Distribution's current module
   equation and vesting's current schedules reconcile. Knowledge, the remaining
   vesting balance and development/research/treasury custody need a documented
   disposition. Their balances alone do not establish how much is owed, to whom,
   or which disbursement is authorized. Preserve obligations, successful
   transaction history and failed-payout ambiguity before selecting a
   migration. The complete current qualification, custom-governance and
   liquidity-pool scans narrow the current record scope; historical omission
   or payout ambiguity still requires transaction evidence.
3. **Resolve operational trust and source lineage before activation.** The
   observed network retains one validator and legacy runtime. Its source
   revision cannot be inferred from a build label. The custody incident,
   replacement signer, exact source/binary binding and recovery evidence need
   their own completed transition; the observer census supplies none of those
   authorities.
4. **Implement authority retirement in the accepted order.** Stored
   `vesting_rewards` module-account permissions at H1,261,575 still include
   `minter` and `burner`. The account query alone can lazily synthesize defaults,
   so this observation was checked against the committed auth record. The
   source repair is not H4-F, H4 or H5, and no census pass substitutes for the
   required freeze, reconciliation, migration and governance boundaries.
5. **Consolidate source from the preserved work.** Follow the accompanying
   [worktree inventory](worktree-consolidation-2026-09-09.md). Eight dirty trees
   have scoped private recovery copies. Most dirty local-main content already
   has a published or staged equivalent; the node-first and separate PR #66
   copies retain distinct work. Review maintenance deltas individually before
   feature expansion or cleanup.

No discrepancy justifies an automatic mint, haircut, surplus assignment,
invented claimant, replay of a historical governance action or bulk worktree
merge. The checkpoint must be recaptured and revalidated for any later
transition; it is not a frozen production boundary.

## Evidence retention and operational close

The full raw records, block proofs, executable provenance, database archive,
source/copy manifests, independent reviews, commands, exit codes and report
digests remain in private operator evidence. This public report contains
sanitized findings and commitments rather than wallet-level inventories or
custody incident details.

The following are SHA-256 hashes of the complete retained report files,
including their final newline, distinct from each report's embedded self-hash:

| Retained report | File SHA-256 |
| --- | --- |
| Official custom-staking census | `50b00c1f187b532ccc3b26a5269a48bc95ec7bd577680ad0d4adc4962af47577` |
| Complete native-bank inventory | `43ec47e52cdaaba26ec5e7f1ae8ddfdd008233630f97ac11af32dcf754e880b8` |
| SDK primary staking obligations | `48222f6e1aa937f89e79769f3830b5a069ba513b90c3a5cafcda4020af3b0d12` |
| Complete raw-store inventory | `61b492ef9ae713226e24ae05a6eb32414e62d0834facaa209e04e00bc41cd31c` |

The official census, complete native-bank inventory, SDK primary-obligation
inventory and complete raw-store inventory each exited **0 / PASS**. The bank
reader's 58 tests include canonical and legacy balance
encodings, complete discovery, arbitrary-precision arithmetic, mismatch and
missing-supply findings, malformed state, wrong roots, bounds and unchanged
database files. The all-store collector's 56 tests include the inherited
scanner/backend checks, an additional mounted store, deterministic output,
wrong evidence, missing trees, stale store versions and bounds. The SDK reader's
58 tests include exact shares, both pool equations, unbondings, redelegations,
malformed state, arithmetic findings and unchanged database files. This report
itself changes documentation only.

The observer exited cleanly with restart disabled. The analysis workstation
was stopped, preserving its volume. Final public reads showed production
advancing from H1,261,597 to H1,261,598, with `catching_up=false` and the same sole
validator. No production ledger, validator executable, configuration, key or
upgrade plan was changed by this census.
