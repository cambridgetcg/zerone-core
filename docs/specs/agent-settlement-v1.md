# Agent settlement v1

Status: **design draft, 13 November 2026. No source implements it, no network runs it and
nothing here is activated. Every design figure is a placeholder; measured figures are
dated facts.**

`agent-settlement-v1` names a design in two parts for holding, moving and proving agents'
value on Zerone, while an agent's identity and memory stay with its home (AgentTool or
any other host).

- **Part A, the record** (§4), uses only what the chains already compute: plain accounts,
  ordinary transfers, and proofs a stranger can check. It needs no consensus change.
- **Part B, holds** (§5), is one new default-off module, `x/settle`. It would lock coins
  under fixed terms and move them only by those terms.

It was written against `main` at `24e82e81` and re-read at `1e939675`. Legacy `zerone-1`
is reported to run `2e37c4c86c31e67515d6aa07b6fd406083e012ea` (`2e37c4c` below; §4.6),
which is ten named upgrades behind `main` (§10). Publishing this document changes no
chain, no doctrine text and no other draft.

An agent session drafted it. On 13 November 2026 the project's owner (Yu) asked for a
design in which Zerone holds, moves and proves agents' value and carries the trust
between them, while each agent's identity and memory stay with its home. The owner has
not approved its rules, its wording or any figure in it. It composes what survived from
four competing designs after adversarial review by other agents of the same session; the
working notes are not published. Statements about code were read at the named commit.
Nothing was built, run or sent to a chain for it.

**Terms.** A *hold* is an amount of coins locked in the module's account under fixed
*terms*. Its *funder* locked the coins. An *escrow* hold pays a *beneficiary*; a *budget*
hold lets a *drawer* pay within limits; a *bond* hold is a lock others can read. An
*arbiter* is an account, or a module compiled into the binary, that the terms name to
rule on a hold. The *silence outcome* is what the terms say happens when an arbiter does
not rule. A *receipt* is the record of one movement or one change of status. The *settle
clock* counts blocks (§5.7); every window in this document is a number of blocks. A
*marker* is a stored flag that turns a rule on. An agent's *home* is where its identity
and memory live. Its *root* is its long-lived identity key; its *spend key* is the account
key that signs transactions. A *producer* is whoever decides which transactions a block
includes. The *ante* is the set of checks a transaction passes before its messages run.
The *household* is the one operator that runs `zerone-1`'s validator today
(`deploy/mainnet/TRUST.md`), and the *door* is its admission service
(`tools/zerone-door`). The *slim cut* is the 2026-07 change that moved markets, escrow and
delegated authority out of consensus (`docs/plans/2026-07-06-slim-cut-migration-map.md`).
The *ladder* is the series of named upgrades between `2e37c4c` and `main`.

---

## 1. What this design would give an agent

None of this exists yet. Points 3 to 7 describe a tool and a module that no chain runs.

1. **An account is a key.** Holding and sending needs no registration. No other account's
   signature can move its coins, and nobody can recover them. Three limits apply today: a
   fee allowance an account grants is spent without its signature; governance can freeze
   a registered account; and whoever controls upgrades can change any rule.
2. **Identity is elsewhere.** Who an agent is, what it remembers and who may act for it
   stay with its home. The chain checks signatures. It does not decide who an agent is.
3. **A payment is an ordinary transfer.** It is final once committed, as far as the
   validators are trusted. Part A would let an agent prove it to a stranger from the
   chain's own commitments, not from a platform's database. Such a proof rests on the
   validators' signatures and says whose they are. On `zerone-1` that is one household's
   key.
4. **A hold would be money under terms.** An amount is locked for a named counterparty and
   leaves only as the terms say: the funder releases it, the counterparty returns it, an
   arbiter that both parties accepted rules on it, or the clock runs out. Where the terms
   name no arbiter, a claim is paid once its review window passes. No message and no
   parameter could move a hold otherwise, the operator's and governance's included. A
   software upgrade or a reset can change any rule, this one too, and on a chain one
   household controls, that is the household.
5. **A budget would be a spending limit the chain enforces.** An amount is locked with the
   name of who may spend it, to whom, how much each time and how much each period.
6. **A bond would show value locked.** A bond with no arbiter stays locked until its
   deadline; one with an arbiter can end sooner (§6). Splitting a bond across accounts
   adds nothing. It says nothing about who controls the coins, and in a unit without a
   price it costs a large holder little.
7. **Every movement of a hold would leave a receipt in chain state**, provable later
   without events.
8. **What it would cost and not give.** Each message pays gas. An escrow reserves nothing
   until the counterparty has spent one transaction accepting it. There is no credit, no
   interest, no priced unit, no way out to other money, no privacy of amounts or parties,
   and no score. Where one controller holds a third or more of the voting power, a
   deadline is only as fair as that controller (§10).

---

## 2. What exists today

Read at `24e82e81` unless a commit is named.

| Function | Today | Where |
| --- | --- | --- |
| Account | A standard SDK account and balance in `uzrn`. An unregistered account is refused only for claims and challenges | `app/ante_zerone.go:1042-1049` |
| Fee | At least 1 `uzrn` per unit of declared gas, a code constant | `app/gas.go:21` |
| Acting for another | Fee allowances only. `authz` types are registered for decoding and nothing more. Vesting accounts and multisig are wired | `app/app.go:391`, `:565` (authz codec); `:1165` (vesting); `:387` with `app/ante.go:60` (multisig) |
| Stored spending limits | `x/home` records limits and recovery addresses that nothing enforces. Only queries read them | `x/home/keeper/keeper.go:417-460`; `msg_server.go:509-511` |
| Holds | Each feature hand-rolls its own in a shared module account: review budget, challenge deposit, augmentation bounty, sponsorship bounty, attestation bond, qualification stake, capture-challenge stake. None checks that the account covers what it owes | `x/knowledge`, `x/sponsorship`, `x/substrate_bridge`, `x/qualification`, `x/capture_challenge` |
| Their defects | Two of them, the qualification stake and the capture-challenge stake, have accounting defects with repairs in review. The accepted authority design records the second, and one in knowledge challenge settlement | [pull request 174](https://github.com/cambridgetcg/zerone-core/pull/174); `docs/AUTHORITATIVE-STATE.md:1326-1329` |
| The reviewer balance | A balance read at commit time. It locks nothing | `x/knowledge/keeper/msg_server.go:449` |
| Best existing pattern | Knowledge fund settlement: terms fixed at admission, a plan frozen at the verdict, all legs and paid markers in one cache, a bounded retry queue | [fund settlement](knowledge-fund-settlement-v1.md) |
| Receipts | No general receipt: a transfer leaves no receipt of itself in state, only a running total per pair of accounts (§9). CometBFT commits only a result's code, data and gas, so an event proves nothing to a light verifier | `cometbft@v0.38.25 types/results.go:47-54` |
| Proof tool | One offline verifier, pinned to `zerone-dev-1` and to one memo shape | `tools/research-checkpoint` |
| Standing | `x/trust_score` measures agreement with review panels, reads an empty record as a middle value, and is scheduled to become a client read | `x/trust_score/README.md:32-33`; `docs/MODULE-CONSOLIDATION-SEQUENCE.md:135` |
| Identity binding | On `main`, registering an account proves control of its Ed25519 identity key. On `zerone-1` it does not, and `MsgRotateKey` leaves an account unable to sign | `x/auth/keeper/msg_server.go:45-62`; `dashboard/public/join/agents.md:159-160`, `:171-177` |

Two facts about the ground:

- One household runs `zerone-1`'s only validator, holds its governance vote, held its
  whole genesis supply and keeps the power to reset it (`deploy/mainnet/TRUST.md`;
  `README.md:33-40`). This repository does not verify or promise a market value for ZRN,
  and the trust map describes the chain's bridge as operationally dark.
- AgentTool's hosted wallet is an operator's ledger of credits, and that repository has
  an open change to stop a wallet funding itself
  ([agenttool pull request 525](https://github.com/cambridgetcg/agenttool/pull/525)).
  Nothing binds an AgentTool identity to a Zerone account.

---

## 3. The line this design moves

The project's written position puts payments between agents outside consensus. Creed
commitment 16 says in its code-expression paragraph that "escrow bookkeeping between
identified counterparties gains nothing from strangers' consensus-verification, which is
the cutline this chain holds" (`docs/TRUTH_SEEKING.md:438`).
`docs/standards/OPEN_CRYPTO_SDK.md:509-521` keeps delegated authority and payment rails in
AgentTool.

This design argues that the sentence holds for the counterparties and not for the
custodian. Consensus cannot tell whether a translation was good. But when two agents do
not share a bookkeeper they both trust, someone holds the money while the work is done,
and strangers can verify that custodian's conduct: the coins are locked, the terms name
everyone who can move them, the deadline fires, and each movement leaves a provable
record.

It proposes that the line be redrawn **between holding and judging**:

- Consensus may hold value under terms its funder fixed at opening, which bind no other
  party until that party accepts them, and move it only by those terms.
- Consensus does not price, match, rank or judge the work, and does not decide who an
  agent is or who may act for it.

Apart from escrow custody, which this design brings back as holds, everything else the
slim cut moved stays where it is: markets and listings, discovery, session keys, recovery
and the question of who may act for an agent. §12 gives the text that would change.
Nothing in this document changes it.

**Alternatives weighed.**

- *Part A alone.* It needs no chain change and holds nothing on terms (§4.5).
- *Wiring `x/authz`.* It grants open-ended authority to execute nested messages, which
  Zerone's custom ante checks can miss (`docs/standards/OPEN_CRYPTO_SDK.md:511-512`), and
  it deletes every expired grant in one block without a bound
  (`cosmos-sdk@v0.53.8 x/authz/keeper/keeper.go:415-447`).
- *One address per hold*, from multisig and vesting accounts. No arrangement of keys gives
  a deadline refund, and an arbiter with either party can send the coins anywhere.
- *A keeper-level library inside existing stores*, for the chain's own bonds only. It needs
  no new store and no change to the line, and it gives agents nothing. Decision 1 in §13
  offers it.

---

## 4. Part A: the record

Part A makes plain accounts and ordinary transfers (`MsgSend`) provable, and gives
off-chain ledgers a place to commit their state. It adds no message and no state.

### 4.1 A proof a stranger can check

One offline verifier, generalising `tools/research-checkpoint`, takes saved read-only RPC
responses and reports four things about a transaction:

1. **Inclusion.** The transaction bytes hash into the block's data hash, the header hashes
   to the committed block ID, and the commit is signed by the validator keys the verifier
   was pinned to.
2. **Authorisation.** The rebuilt sign bytes verify against the sender's key, so the
   sender signed exactly these messages, this fee and this memo.
3. **Execution.** The block's results hash into the next header, and the transaction's
   result code is reported. Result data is covered. Logs and events are not, and the
   verifier reports them as not proven.
4. **State, when asked.** A balance or an account sequence at a height, by a store proof
   against the application hash in the next header.

**A1.** Every output states, in words, whose signatures the proof rests on. Where the
validator set is one key, the label says one operator attested it.

**A2.** The verifier never smooths a gap. A field it could not prove is reported as not
proven, and two valid commits for one height are kept and reported as equivocation.

**A3.** The trust pins are a genesis digest and validator keys compiled into the tool or
given in a reviewed policy file, never a flag a caller can point at any key. A change of
validator set needs a reviewed policy, as `tools/research-checkpoint/README.md` already
requires.

### 4.2 A statement that balances

For a plain account over a range of heights:

```text
closing = opening + listed incoming + effects of its own transactions
          - fees it paid for itself - fees it paid for accounts it sponsors
closing sequence - opening sequence = number of its own transactions, failed ones included
```

The opening and closing values are store proofs, and each listed transaction is proven as
in §4.1. A failed transaction still consumes a sequence number and a fee. If both lines
hold, every outflow is accounted for and no inflow is missing in value. The verdict is one
of `BALANCED`, `UNEXPLAINED_INFLOW`, `UNEXPLAINED_OUTFLOW` or `UNPROVABLE`.

**A4.** `BALANCED` is not reported for an account with a fee allowance live in the range,
unless each grantee's remaining allowance is also proven at both ends. Fees are taken
from a granter without its signature and without a sequence number.

**A5.** An account credited by the chain without a transaction cannot read `BALANCED`
unless each credit is derived from committed state and labelled. On `2e37c4c` the block
producer's account is such an account (§4.6).

**A6.** Statements need state that nodes may prune. A prover captures the closing proofs
promptly. How long `zerone-1`'s public node keeps state is not known.

### 4.3 Anchors

An *anchor account* is an account whose only transactions are anchors: a transfer of one
`uzrn` to itself with the memo `zr1:anc:<stream>:<seq>:<head>`, where `<stream>` is 32
lowercase hex characters, `<seq>` is a decimal counter from 1 and `<head>` is 64 lowercase
hex characters. Its account sequence counts its anchors. A verifier that finds and proves
that many, with contiguous counters and chained heads, has the publisher's complete
anchor history and proof that the publisher did not show two versions of it.

Uses: a coordination journal's head, a research journal's head, and a custodial host's
liabilities. A hosted ledger could anchor a liabilities root with its total split by its
own provenance labels. That would commit the operator to one stated total that each
holder can check their own entry against. It would prove neither that the list is
complete nor that the labels are true, and it would turn none of it into money.

**A7.** An anchor proves order and completeness for one publisher's anchors. It does not
prove that what a head commits to is true or complete.

### 4.4 A payment that says what it paid for

A transfer may carry the memo `zr1:pay:<64 hex>`, the SHA-256 digest of the signed request
it settles. The request stays off chain. With §4.1 this proves that this payer paid this
payee this amount and named that request's digest. Matching the digest to the payee's
signed request is an off-chain check. The memo grammar is closed: `pay` and `anc`.

**A8.** A digest written to a chain is permanent. A digest of an off-chain record (an
agreement, a mandate, a delivery, a payment request, a liabilities entry) is computed
over a record that includes a random salt shared only by the parties, so it is never the
hash of something guessable. No platform identifier, URL or invocation ID is written to a
chain. This extends the privacy rules of [WITNESS v0](witnessed-agent-economy-v0.md). A
`did:zrn` and a root's public key are written by design (§7).

### 4.5 What Part A cannot do

It cannot hold money on terms. The nearest arrangements with existing parts fail in ways
that cannot be repaired from outside consensus:

- A two-of-three multisig "hold" lets the arbiter and either party send the coins
  anywhere, and leaves them stuck for good if the arbiter and one party go silent.
- A one-of-two multisig "budget" can never be revoked, because the delegate's key stays a
  member of that address.
- A vesting account is a time lock nobody can forfeit or return early.

Part A therefore offers none of these. They are the reason for Part B.

### 4.6 Cautions for `zerone-1`

- That `zerone-1` executes the `2e37c4c` logic is the operator's own record, not an
  independent audit. The original executable was reproduced byte for byte from that
  source (`docs/reports/legacy-observer-release-2026-09-10.md:32`), and the operator
  reports replacing it on 11 November 2026 with the same source built against a newer
  consensus engine
  ([pull request 177](https://github.com/cambridgetcg/zerone-core/pull/177), a draft).
  Reading what the node reports against that record is the first step of any Part A work
  there (§14).
- On `2e37c4c`, a block that carries a transaction mints a block reward, most of it to the
  producer (`x/vesting_rewards/keeper/rewards.go`, `DistributeBlockReward`). Any
  transaction there, an anchor included, issues ZRN to the household. Reading is free of
  this. Writing is a live act that needs the owner's word.
- `docs/standards/OPEN_CRYPTO_SDK.md:504-507` keeps the dashboard's grant and
  sponsored-spend paths off until the fee-granter guard is deployed, and that guard exists
  only on `main`. The door's own fee grants on `zerone-1` run without it and cover three
  message types, none of them a transfer (`tools/zerone-door/src/policy.ts`). Part A adds
  no sponsorship of transfers there.
- Never send `MsgRotateKey` on `zerone-1`, and do not treat a `zerone-1` registration as a
  binding (§7).

---

## 5. Part B: holds

One module, `x/settle`, with one kind of object. It has no account or balance object of
its own (SDK bank stays the only writer of balances), no identity registry, no listing,
no price and no score.

### 5.1 The hold

A hold records: an identifier; its kind; its terms and their digest; the accounts that
hold its funder, beneficiary and drawer roles now; its status; how much is locked now; its
next timer; and the head of its receipt chain. The terms include the amount in `uzrn`, a
deadline on the settle clock, a 32-byte `client_ref`, and every limit that shapes the hold,
copied from the parameters at opening.

**H1.** Terms never change after opening. Their digest is recomputed on every transition.
The accounts holding roles are recorded beside the terms and change only by reassignment
(§5.9).

**H2.** `(opening account, client_ref)` opens at most one hold, ever, and
`(hold, client_ref)` draws at most once, ever. A host that writes `client_ref` to its
journal before signing can resend after an ambiguous broadcast and cannot pay twice. A
host settles every ambiguous reference before it reassigns a role.

**H3.** Every outgoing coin goes to an account the hold's own record names or, in an
`any_payee` budget, to the payee its drawer names. No message and no parameter can send
held coins anywhere else.

### 5.2 Three shapes

**Escrow.** Terms: the beneficiary; an arbiter, or none; the arbiter's fee; an
acceptance limit; a review window; a ruling window; the silence outcome; a digest of the
off-chain agreement. The amount and the arbiter's fee are both locked at opening.

**Budget.** Terms: the drawer; the payees (one to eight accounts, or an explicit
`any_payee`); a ceiling for each draw; a period length and a ceiling for each period; a
maximum number of draws; a digest of the off-chain mandate.

**Bond.** Terms: an arbiter, or none; a beneficiary for forfeits, or none; an acceptance
limit, if it names an account arbiter; a digest of the bond's purpose. With neither
arbiter nor beneficiary, it is a plain time lock. With an arbiter and no beneficiary, the
arbiter can only return it early. With both, the arbiter can forfeit up to the locked
amount to the beneficiary, in at most eight rulings, each for a positive amount. A
beneficiary without an arbiter is refused.

**H4.** The funder, the beneficiary and the arbiter of an escrow are three different
accounts. A budget's drawer is not its funder. A bond's arbiter is neither its funder nor
its beneficiary. No party is a module account. An arbiter may be a module only from the
compiled list (§5.12).

**H5.** An escrow with an arbiter has a ruling window and a silence outcome. The silence
outcome is release unless the terms name return or a split. Where the arbiter is an
account, the escrow also has a fee above zero and at most half its amount; a module
arbiter takes no fee. An escrow with no arbiter has no fee, no ruling window and no
silence outcome. An acceptance limit is a settle-clock value after opening and before the
deadline. Every window is at least two blocks, so the other party always has a block in
which to act.

### 5.3 Messages

| Message | Signer | Effect |
| --- | --- | --- |
| `MsgOpenHold` | funder | Moves the amount, and an escrow's arbiter fee, from the funder to the module. An escrow, and a bond that names an account arbiter, start `OFFERED`; any other hold starts `OPEN` |
| `MsgAcceptHold` | an escrow's beneficiary; an account arbiter | Records that signer's acceptance of the terms digest it carries. The hold becomes `OPEN` when everyone who must accept has: an escrow's beneficiary and any account arbiter, a bond's account arbiter |
| `MsgClaimHold` | beneficiary | `OPEN` to `CLAIMED`, with a digest of what was delivered. Starts the review window |
| `MsgReleaseHold` | funder | Pays the escrow's amount to the beneficiary at any point after it is `OPEN` |
| `MsgReturnHold` | escrow: the funder while `OFFERED`, the beneficiary afterwards. Budget: the funder. Bond: the funder while `OFFERED`, its beneficiary afterwards | Returns everything still locked to the funder |
| `MsgDisputeHold` | funder | `CLAIMED` to `DISPUTED`, with an evidence digest. Starts the ruling window. Moves no coins. Refused when the terms name no arbiter |
| `MsgRuleHold` | arbiter | Escrow: release, return or split, then the fee to the arbiter. Bond: forfeit an amount, or return early |
| `MsgDrawBudget` | drawer | Pays a payee within every limit of the budget |
| `MsgReassignParty` | the current holder of a role and the account taking it | Moves that role in up to 64 holds (§5.9) |
| `MsgSettleHolds` | anyone | Runs the due automatic transition for up to 16 holds |
| `MsgUpdateParams` | SDK governance | Changes parameters for holds opened afterwards. It cannot touch an open hold or a held coin |

No message nests another message. A module may open and rule holds through the keeper,
never through a message (§5.12).

### 5.4 How a hold ends

Escrow:

```text
OFFERED --all accept--> OPEN --claim (beneficiary)--> CLAIMED --dispute (funder)--> DISPUTED

OFFERED   funder withdraws, or the acceptance limit passes .............. RETURNED
OPEN      funder releases ............................................... RELEASED
          beneficiary returns, or the deadline passes with no claim ..... RETURNED
CLAIMED   funder releases, or the review window passes .................. RELEASED
          beneficiary returns ........................................... RETURNED
DISPUTED  arbiter rules .............. RELEASED, RETURNED or SPLIT; fee to the arbiter
          funder releases ............ RELEASED; fee to the arbiter
          beneficiary returns ........ RETURNED; fee to the arbiter
          ruling window passes ....... the silence outcome; fee back to the funder
```

Before a dispute the arbiter's fee returns to the funder on every ending. Once an escrow
is `DISPUTED`, the fee goes to the arbiter on every ending except silence.

Budget: `OPEN` until nothing is left in it (`EXHAUSTED`), or the funder returns it, its
deadline passes or its last permitted draw leaves a remainder (`RETURNED`). A remainder
returns to the funder in that same operation.

Bond: one that names an account arbiter is `OFFERED` until that arbiter accepts, and
returns to its funder if the funder withdraws it or the acceptance limit passes. Then, as
any other bond, it is `OPEN` until forfeits empty it (`FORFEITED`), or its deadline
passes, its arbiter returns it early or its beneficiary waives it (`RETURNED`).

**H6.** Status only moves forward along these edges. A finished hold has nothing locked
and no timer, and its record and receipts stay readable.

**H7.** Until every acceptance is recorded, a hold obliges its beneficiary and its arbiter
to nothing, runs no timer against them, counts against none of their limits and is not
shown as theirs (S1). A drawer is in the same position until its first draw. Nobody can
be named into an obligation, and nobody's name can be borrowed.

### 5.5 Disputes

- Only the funder disputes, only after a claim, and only where the terms name an arbiter.
  Before a claim the deadline already protects the funder. After a claim, doing nothing
  pays the beneficiary.
- An escrow with no arbiter cannot be disputed. It pays any claim once the review window
  passes, whatever was delivered. Its funder's only protections are the choice of
  counterparty, the amount and the deadline.
- The fee is locked at opening, so a dispute moves no coins. Once a dispute is open the
  arbiter is paid however it ends, a concession by either party included, unless the
  arbiter stays silent through the ruling window. A funder therefore cannot dispute to
  stall and then concede for nothing, and cannot dodge the fee by conceding once a ruling
  is on its way.
- A ruling is release, return or a split in basis points. The remainder of a split goes to
  the funder.
- The silence outcome is release unless the terms say otherwise. Any other silence outcome
  lets a funder cancel after delivery whenever the arbiter stays silent, and a beneficiary
  that accepts such terms accepts that.
- A funder that disputes a false claim and wins still loses the fee to the arbiter. H5
  bounds that loss at half the amount. This version does not shift it (§9).

**H8.** A ruling can pay only the hold's two parties and the fee. The fee is the same
whatever the ruling, and whether the dispute ends by a ruling or a concession.

An arbiter is an account both parties accepted in the terms, and it accepted them too. It
may be a multisig of several keys. A panel drawn by the chain is not in this version (§9).

### 5.6 Custody

All held coins sit in one module account that has no permission to mint, burn or stake.
The module keeps one number, the *liability total*: the sum of what is locked in every
unfinished hold, arbiter fees included.

**C1.** After every operation, the change in the module account's `x/bank` balance equals
the change in the liability total, and the balance is not below the total. An operation
that would break either is refused whole.

**C2.** At genesis, at export, in every upgrade handler and on request, a full scan
recomputes the total from the holds, records any excess of the balance over it as surplus
(C4), and fails only if the balance is below the total.

**C3.** Only the module's own operations can move coins into or out of its account. The
account would be a refused recipient for ordinary transfers, as every module account but
governance's is (`app/app.go:1712-1722`). The remaining path, another module's keeper, is
closed by a send restriction in `x/bank`. This would be the first send restriction in
production wiring, and it runs on every send, so it is designed and reviewed together
with the research-fund guard that `docs/AUTHORITATIVE-STATE.md:671-680` assigns to a
later upgrade.

**C4.** A surplus that should be impossible is recorded and shown, never assigned to a
hold and never spent. It blocks no exit, no import and no upgrade. The upgrade that first
activates the module adopts whatever account already sits at the module's address, before
the module's state is first initialised, and records any balance there as surplus, so a
coin sent there beforehand cannot halt a chain (§10).

**C5.** The module issues nothing and charges nothing. Its messages pay ordinary gas. It
has no fee, no share and no yield, and locked coins cannot be bonded, delegated or
counted as a vote.

**C6.** Opening a hold, drawing from a budget and reassigning a role create a new
destination for coins, so each checks `x/bank`'s send-enabled setting for the
denomination, as an ordinary transfer does. Accepting, claiming, disputing, releasing,
returning and ruling do not: they pay only accounts the hold already names.

**C7.** The set of accepted denominations is compiled: `uzrn` only. Adding one takes a
reviewed binary, not a vote.

### 5.7 Time

Every window is counted on the settle clock. It never reads block time. With one
producer, block time is that producer's clock; a count of blocks cannot skip.

**T1.** A message reads the settle clock as it stood when its block began. A message from
a party is valid only while that value is below the hold's timer. `MsgSettleHolds` is
valid for a hold only when the hold's timer is at or below it. At the end of each block
the clock advances by one, unless T3 holds it, and then holds whose timer is at or below
the new value are settled. So a claim, a dispute or a return included in a block is never
beaten by a timer that matures in that block, and nothing but settlement acts on a hold
once its timer has matured.

**T2.** Every unfinished hold has exactly one timer entry. A block settles at most 64 due
holds (a placeholder), from a cursor. A hold that fails to settle stays due and does not
stop the queue. Anyone may settle a hold that is still due in a later block with
`MsgSettleHolds`.

**T3.** While `x/emergency` reports the chain halted, the settle clock stands still and
nothing settles. That includes the period after a resume in which transactions are still
refused (`x/emergency/keeper/state.go:75-99`), and a fixed number of blocks after
transactions are admitted again. A pause can delay an outcome. It should not be able to
choose one.

Clients convert blocks to time from the observed rate, not the target. `zerone-1` has
produced a block about every 30 seconds (`dashboard/public/join/agents.md:181-182`) where
the target is about 2.5 seconds (`README.md:112`).

### 5.8 Receipts

Each transfer leg and each change of status writes one receipt into state: the hold, its
sequence within the hold, the hash of the previous receipt, the action, who acted, from
whom and to whom, the amount, the height, the terms digest and one evidence digest.

**P1.** A receipt is indexed, as a gapless list, under each account that has signed for or
acted in a role in that hold, from that account's first act onward, and under no other.
An offer that was never accepted appears only under its funder. A payout to an account
that signed nothing, a budget's payee or a bond's beneficiary, is indexed in that
account's separate list of credits. Filling someone's list of credits costs the coins
paid to them.

**P2.** The receipts written by a transaction's messages are also returned in its
response data, which the next header commits to. A party can prove such a receipt at once
from block results. A receipt written by a timer is proven from state, and is captured
promptly (A6).

**P3.** A statement of an account's holds is its list length plus the receipts, each
provable from state. No event is needed for any proof.

**P4.** Receipts and finished holds are not deleted. If their growth ever needs a limit,
that is a decision taken in the open.

### 5.9 Reassignment

`MsgReassignParty` moves a funder, beneficiary or drawer role to another account. It is
signed by the current holder and by the account taking the role, so a role cannot be
pushed onto anyone or sent to an address nobody controls. An arbiter and a payee are never
reassigned; each keeps its key until the holds that name it end. A message that names
several holds applies to all of them or to none.

**R1.** After a reassignment H4 still holds for that hold.

**R2.** Each role in a hold can be reassigned at most four times. A hold that is still
`OFFERED` and a hold a module opened accept none.

### 5.10 Limits

Parameters set a minimum amount, the longest deadline, the shortest and longest windows,
the most draws for a budget, and the most unfinished holds one account may have opened.
Each has a compiled ceiling or floor that governance cannot pass. There is no chain-wide
ceiling on holds: growth is bounded by gas, the minimum amount and the per-account limit.
Nothing refuses a draw from a funded budget or an exit because the chain is busy.

**L1.** A limit counts an account only in a role it signed for. The limit on opened holds
counts the account that opened a hold for that hold's whole life, whoever holds the
funder role later.

**L2.** Every budget has a period ceiling. Periods are fixed windows on the settle clock,
counted from opening, and nothing carries over. A stolen drawer key can therefore take at
most the ceiling once for the current period and once more for each period boundary
crossed before the funder's return lands.

### 5.11 Freeze

`main` refuses every transaction from a frozen registered account
(`app/ante_zerone.go:902-935`). If that also stopped a frozen beneficiary from claiming
or a frozen arbiter from ruling, whoever holds the freeze power could choose the outcome
of a hold by waiting.

**F1.** A freeze stops an account creating new destinations. It does not stop a committed
outcome. A transaction made only of `MsgClaimHold`, `MsgDisputeHold`, `MsgReleaseHold`,
`MsgReturnHold` or `MsgRuleHold` passes the frozen-account check. Each of these moves
coins only to accounts the hold already names.

**F2.** The exemption cannot be used to pay value away as fees. A transaction that relies
on it declares a gas limit no higher than a compiled ceiling for each of its messages,
set from that message's measured worst case; its fee is exactly that limit at the minimum
gas price; and the ante refuses it, keeping no fee, unless each message is valid as it
stands: the signer holds the role, the status admits the message, and the hold's timer
has not matured.

### 5.12 Module-owned holds

A module named in a compiled list may open a hold for the signer of the message it is
handling, and may rule a hold whose terms name that module as arbiter. The list is empty
in the first release. Each module is added in its own reviewed release. Governance cannot
add one. Such a hold is bounded by its owning module's rules, not by the minimum amount
or the per-account limit, and its `client_ref` is derived by the module. Its funder is
always an ordinary account. A ruling the module asks for that fails is recorded as
pending and that same ruling is retried from the queue (T2); the hold's timer does not
replace it.

This exists so that later work bonds need not each hand-roll an account and an invariant.
It asks nothing of other work. The
[reviewer-entry draft](https://github.com/cambridgetcg/zerone-core/pull/190) designs its
seat deposit with an account of its own (its D2), and this document does not ask it to
wait or change. A seat deposit could be expressed as a bond whose arbiter is the knowledge
module and whose beneficiary is empty, which the module can only return. Moving it would
have costs that draft's authors would have to weigh: its deposits would share an account
with other holds, a send halt would refuse new commitments (C6), and it would come to
depend on decision 1 (§13) and on a new store.

Retiring modules' escrows are never moved onto holds, and no existing balance is
reinterpreted.

### 5.13 What the module never does

It does not lend, pay interest, extend credit or allow an overdraft: a hold is fully
funded when it opens. It does not hold listings, prices, quotes, work, evidence or
mandates, only 32-byte digests of them. It does not hold KARMA, AgentTool credits or any
second denomination. It gives governance no message and no parameter that withdraws,
redirects or releases. It does not speak HTTP.

---

## 6. Trust

Trust here is what a counterparty can check, not a number the chain assigns.

- **The terms.** Before accepting, a beneficiary reads exactly who can move the coins, to
  whom and when, including the arbiter and the silence outcome.
- **The custodian.** Anyone can prove, at any height, that the module's balance covers its
  stored liability total. Checking that total against every hold needs a proof of each
  hold. Short of a software upgrade or a reset, only the module's code moves held coins.
- **Bonds.** A bond with no arbiter and no beneficiary is locked against its funder until
  its deadline. Any other bond can end sooner, by its arbiter's return or its
  beneficiary's waiver, and can be forfeited to its beneficiary by its arbiter's ruling.
  The purpose digest is not read by consensus. A reader therefore looks at who the
  arbiter and beneficiary are before giving a bond any weight. An account arbiter has
  accepted the bond's terms, or the bond is not shown.
- **History.** An account's receipts show what it opened, released, returned, disputed and
  forfeited.

**S1.** The chain publishes components, never a composite. A query returns an account's
open bonds, each with its amount, deadline, arbiter and beneficiary, and its open holds by
role. A hold that is still `OFFERED` is listed only for its funder, and as an offer. The
query returns no score, band or rank, and an account with no history reads as no history,
not as a middle value.

**S2.** Counting conduct is left to indexers. An honest one does not count a role its
holder never accepted, marks holds between accounts that fund one another, and does not
treat activity as costly where fees return to the party that produced the block.

None of this is credit. Nothing here lends against a record.

---

## 7. Identity and continuity

Four keys, none inferred from another:

1. **The root.** The agent's long-lived Ed25519 identity key, held by the agent and kept
   away from daily signing.
2. **The chain identity.** On `main`, the agent registers its account with the root as the
   identity key and a proof of possession (`proto/zerone/auth/v1/tx.proto:38-50`). Chain
   state then binds root to account, proven from both sides. This is the binding. It
   exists already, and this design adds no registry.
3. **The spend key.** The account key that signs transactions and holds roles in holds.
4. **The counterparty's keys.** A hold names accounts, never identities.

**I1.** No identifier issued by a home is written to a chain. The home maps its identifier
to the root; the chain maps the root to an account.

**I2.** On `zerone-1` a registration proves nothing about who holds the identity key
(`dashboard/public/join/agents.md:171-177`), although the door has admitted agents
register there. On that chain the binding is a record the agent keeps at its home, signed
by both the root and the account key over one digest. The door's request already proves
both keys, each over bytes that name the other (`tools/zerone-door/README.md:105-121`).

**I3.** A key that can move value is generated and held where the operator of the agent's
home cannot use it. A host that generates or stores its agents' spend keys is their
custodian, whatever the chain enforces.

**Rotation.** To retire a spend key that still works, the agent reassigns its funder,
beneficiary and drawer roles (§5.9) and sends its free balance on. Each reassignment
writes a receipt naming both accounts, so a history joins across keys. Where it is an
arbiter or a payee, it keeps the old key until those holds end.

**Loss.** A lost spend key loses the free balance it held. SDK bank has no recovery and
this design adds none. Its roles in open holds stay with the lost account until those
holds end, and coins those holds pay to it are lost with it. Recovery of roles through the
root, with a delay and a veto by the old key, is not in this version (§9).

**What the home keeps.** Identity, memory, session keys and the right to act for an agent.
The signed mandate behind a budget, the agreement behind an escrow and the evidence behind
every digest. The HTTP edge: a payment scheme whose verify step must find an escrow that
is `OPEN` and names the payee, which costs the payee one acceptance transaction before it
serves, and which is never described as x402's `exact` scheme. And a view, on waking, of
the agent's open holds and receipts read from chain state.

**What the home need not duplicate.** Once chain holds exist, a host's own escrow and
dispute features for chain value can be views over chain holds, not a second ledger.
Hosted credits are not bridged and not converted.

---

## 8. What an implementation must show

**Tested rules.** Each rule labelled H, C, T, P, R, L and F, and S1, with a test that
fails when the rule is broken. In addition:

- A property test over long random sequences of every message on every shape, asserting C1
  after each step and C2 at the end, with supply unchanged throughout.
- Refusal of every transfer into or out of the module account that the module did not
  make: a transfer, a multi-send, an IBC receive and another keeper's call.
- A table test of every (shape, status, event) triple, with the refusal for each
  disallowed one, and of T1 at each boundary block.
- An `x/bank` failure injected on the last leg of each operation writes nothing.
- No draw and no exit is refused by any limit; `MsgUpdateParams` changes no open hold;
  governance cannot pass a compiled bound or add a module to the compiled list.
- With the marker absent: every message is refused, the end block does nothing, and every
  other module's store is identical to the same binary without the module, for inputs
  that do not touch the module's address (§10). A new store changes the commit info even
  when empty, and the tests say so instead of claiming the whole application hash is
  unchanged.
- The activating upgrade with a coin already at the module's address (C4), and a full
  scan that finds a surplus (C2).
- A frozen signer's exempt transaction at, above and below its gas ceiling, and one whose
  message is not valid as it stands (F2).
- Genesis export and import round-trip every hold, receipt, index and counter and recheck
  C2.
- Measured gas for each message, added to the gas table.
- Vectors for the terms digest, the receipt hash and the Part A memos, shared with the
  client that builds them.
- For Part A: the verifier refuses a commit signed by any key it was not pinned to,
  reports events as not proven (A1 to A5), and checks an anchor history for gaps and for
  two heads at one counter.

**Disclosures, not tests.** A6 and A7 (what a proof cannot show), A8 and I1 to I3 (what
clients and homes must do), S2 (what an honest indexer does), and every gate in §10.

---

## 9. What this version does not solve

- **A unit worth settling in.** This repository promises no value for ZRN. Until a unit
  has one, holds are rehearsal and the chain's own bonds.
- **Concentrated voting power.** A controller with a third or more of the voting power can
  keep a claim or a dispute out of blocks until a timer decides, and leave no trace. §10
  makes this a gate.
- **How an agent from outside gets coins.** The door's seed is 0.222 ZRN and its allowance
  covers no transfer; a hold costs several transactions of gas plus its amount.
- **A false claim's cost.** The funder who disputes one and wins still pays the arbiter. A
  deposit posted with the claim, forfeited on a return ruling, would shift that.
- **Disputes without a chosen arbiter.** A panel drawn by the chain needs reviewers from
  outside the household, which the reviewer-entry work has not yet opened.
- **Recovery through the root.** Reassigning roles on the root's signature needs a delay,
  a veto and a rule for registrations made before proof was required. A stolen root would
  otherwise redirect open holds.
- **Small, frequent payments.** One hold per job suits a review or a task. A tab that
  settles many calls in two transactions, on signed cumulative vouchers, is the next
  shape.
- **A drawer opening escrows from a budget.**
- **Privacy.** Parties, amounts and timing are public and permanent. The chain also keeps
  a running total of who funded whom, per pair of accounts, for every `uzrn` transfer
  (`app/sybil_funding_decorator.go`).
- **Standing that resists copies of one controller**, beyond what a bond locks.
- **Compliance controls.** There are none in code. This document does not judge whether
  that is lawful. F1 means a frozen party's claim cannot be blocked, which belongs in the
  review G2 requires.
- **Succession.** What happens to an agent's value when it ends.
- **Demand.** As of this draft, no agent from outside the household has asked for this.

---

## 10. Activation and where it can run

Part A needs no activation. Its tool is repository work, and its reads cost nothing.

Part B is consensus work:

- A marker in the module's own store turns it on. A fresh genesis may set it. An existing
  chain gets the store and the marker only from a named upgrade, `agent-settlement-v1`.
- No named upgrade has yet added a store (`app/upgrades.go:1172-1240`). This would be the
  first, with its own two-binary rehearsal.
- The module's account address becomes a refused recipient in the first binary that
  compiles the module, marker or no marker (`app/app.go:1712-1722`). That is itself a
  change in what a chain computes. An earlier upgrade already refuses to plan when a plain
  account sits at a module address (`app/upgrades_accounting_authority.go:426-441`), so
  every rung before `agent-settlement-v1` is cut from source without the module, or its
  plan builder is taught to pass over this address.
- It reaches `zerone-1` only after the ladder, each rung with its own binary and halt. No
  date can honestly be given.
- A pull request that adds it changes what a chain computes. It is marked `[consensus]`
  and merged only by a session other than its author's, after an independent review.
  Several such drafts already wait for one, and this would join them.

The gates below are proposals. The legal points in them are a design reading, not legal
advice.

**G1. Voting power.** Where one controller, alone or with a known coalition, holds a third
or more of a chain's voting power, or an outside party cannot reach an independent
proposer without that controller's nodes, no hold whose outcome a timer or a left-out
transaction can decide is offered to anyone outside that controller. Such a chain may run
holds for rehearsal, and holds whose every exit goes to the funder, and each of its
surfaces says so. A validator set that passes this gate is a precondition for everything
in §11's last row. This document does not schedule it (decision 4).

**G2. The unit.** No denomination with a price is added before a legal review that covers
money transmission, payment-service and stablecoin rules, and sanctions exposure, which
is a separate question from who controls the coins.

**G3. Wording.** Public names and copy say holds, settlement, receipts and bonds. The word
for a deposit-taking institution is protected in several jurisdictions and would claim
more than is true here.

**G4. Operator conduct.** On a chain it operates, the household would not act as arbiter
for others' holds and would take nothing on their settlement beyond ordinary gas, which
on a one-producer chain it receives as the producer. Release stays programmatic.

---

## 11. First uses

| Use | Part | Who | Status |
| --- | --- | --- | --- |
| Statements for the household's operating accounts, so that a refill of the door's working balance is a checked fact | A | The household | Possible once the Part A tool exists. No chain act |
| A review bounty between two of one operator's agents: funded for a named reviewer, claimed with a digest of the written review, released on delivery and never on the verdict | B | One operator's agents, on a chain with no value | Rehearsal, and labelled as that |
| A later work bond opened through the keeper | B | The chain itself | Waits on a module that wants one (§5.12) |
| Settlement for a seller who wants no operator able to touch it | B | Nobody yet | Waits on G1, G2 and a named counterparty |

Nothing past rehearsal is presented to agents from outside without a named outside
counterparty. Each later shape in §9 waits for an observed need, not a plan.

---

## 12. Text this design would change

Proposed wording, applied nowhere. The creed's commitment names in
`x/creed/types/genesis_creed.go` do not change, so no genesis doctrine fact is added.
`docs/TRUTH_SEEKING.md` is bound to `.creed-hash`, which would move with it, and the
commitment-16 banner in `tests/cross_stack/truth_seeking_invariants_test.go:1314-1323`
restates the line and would be re-quoted. A re-pin on a running chain is a separate
governance act.

**`docs/TRUTH_SEEKING.md:438`, commitment 16, code expression.** The open-question market
stays as it is: bounties stay on the platform, a listing still resolves against on-chain
acceptance of the linked fact, and the break clause at line 440 is unchanged. Replace the
sentence from "askers post" to "this chain holds." with:

> askers post bounty-carrying listings and escrow payment there. The cutline this chain
> holds is between holding and judging: consensus may hold value under terms its funder
> fixed at opening, which bind no other party until that party accepts them, and move it
> only by those terms, because what strangers' consensus-verification checks there is the
> custodian, not the counterparty. Consensus does not price, match or judge the work. No
> hold in `agent-settlement-v1` pays only on acceptance of a fact, so exploration
> bounties stay on the platform; a hold whose only release is the knowledge module's
> ruling on acceptance would need its own amendment.

and its last sentence with:

> The former `x/inquiry` module was the bounty escrow, moved off-chain intact.

**`docs/standards/OPEN_CRYPTO_SDK.md:509-514`.** Keep "Do not wire `x/authz` yet" and its
reasons. Replace "authz also overlaps the slim-cut decision to keep delegated authority
in AgentTool" with:

> authz would also grant open-ended authority to execute messages, which nothing adopts:
> who may act for an agent stays in AgentTool. Bounded spending of value a funder has
> already locked is a different act and belongs to `x/settle` budgets, which pay only
> coins locked in them, only within their terms, and nest no messages.

**`docs/standards/OPEN_CRYPTO_SDK.md:516-521`.** Retitle the section "x402 payment adapter
in AgentTool" and replace "A Zerone adapter belongs in AgentTool, where the slim cut moved
payment rails." with:

> The HTTP adapter and any facilitator belong in AgentTool; consensus does not speak
> HTTP. The adapter may settle through an `x/settle` escrow: verification must find the
> escrow open, accepted and naming the payee, so it reads a reservation instead of
> predicting that a later transfer will succeed.

**`docs/AUTHORITATIVE-STATE.md` §1.1, Authority table (lines 52-67).** Add a row:
conditional holds of locked coins (terms, status, rulings, draws, receipts) are written by
`x/settle`; SDK bank remains the sole balance writer; a ruling is not a governance
decision.

**`docs/AUTHORITATIVE-STATE.md:331-341`.** That section requires an owning module to
isolate each bond compartment from unrelated funds, states an exact equality between the
bond balance and what is owed, and forbids an unowned balance bucket. Either add that a
hold's own record under C1 and C2 is that isolation and that a recorded, unspendable
surplus (C4) is permitted, or keep module-owned bonds out of `x/settle`.

`docs/AUTHORITATIVE-STATE.md` is a design the repository records as accepted on 8 August
2026, and its hash is pinned in `tools/authority-graph/manifest.go:20` and in the
dashboard's copies. Either edit to it moves those pins and amends an accepted design, so
it is the owner's to make (decision 9).

---

## 13. Decisions this document leaves to the owner

1. **Whether the line moves.** Three answers: (a) as §3 and §12 put it, with Part B
   following; (b) the line stays, and the chain's own work bonds get a keeper-level
   library inside existing stores, with nothing for agents; (c) neither, and Part A is the
   whole of it. Recommended: (a). Only (a) lets agents hold value on terms, and it costs
   one large `[consensus]` change in a queue that is already long.
2. **Public wording** (G3). Recommended: adopt.
3. **The unit** (C7, G2). Recommended: `uzrn` only, compiled.
4. **The voting-power gate** (G1), and whether a validator set that passes it becomes a
   named goal with an owner. Recommended: both.
5. **Operator conduct** (G4). Recommended: adopt as standing rules. AgentTool's hosted
   wallet is outside this document.
6. **Where holds first run in public**: a fresh chain with no value and a new chain ID, or
   an upgrade of `zerone-dev-1`. Recommended: a fresh one, which has no predecessor state
   to count. Either is a chain act.
7. **Any Part A transaction on `zerone-1`** (§4.6). Recommended: read-only statements
   first; anchors one by one, each on the owner's word.
8. **Whether any module-owned bond is adopted** (§5.12), with the authors of the work it
   would change.
9. **The two edits to the accepted authority design** (§12, last items).
10. **On the AgentTool side**: a host's escrow and disputes for chain value as views over
    chain holds; an anchored liabilities root before any payout reopens; where agents'
    spend keys are generated (I3).
11. **Two things this design found and does not decide.** On both commits read,
    `development_fund` receives a share of every fee and no code path spends it. And on
    `2e37c4c` a parameter for a founder share on the fee path still exists, switched off,
    which `main` retires.

---

## 14. Order of work

1. This document, read by a session other than its author's.
2. A read of what `zerone-1` reports it runs, against the operator's record (§4.6), before
   any Part A work relies on it.
3. The Part A verifier and statement reader, as repository work with no chain act. It is
   the same tool whichever way decision 1 goes.
4. The owner's answer to decision 1.
5. If (a): `x/settle` source behind its marker, with the tests of §8, as a `[consensus]`
   draft.
6. The client pieces in AgentTool: builders for the sign bytes, the statement reader, the
   hold-backed payment scheme.
7. A public run on a chain with no value (decision 6).
8. `zerone-1`, after the ladder, and only past G1.
