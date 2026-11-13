# Knowledge reviewer entry v1

Status: **design draft, 11 November 2026, revised 13 November. Steps A and B are drafted
as unmerged source (pull requests #193 and #199); no network runs any step, and none of
its figures is chosen.**

`knowledge-reviewer-entry-v1` names a design for three default-off steps. It assigns no
knowledge consensus version. It is written against the neutral review policy on `main`
at `bd76ea1d` ([review neutrality](knowledge-review-neutrality-v1.md),
[fund settlement](knowledge-fund-settlement-v1.md),
[claim integrity](knowledge-claim-integrity-v1.md)). Legacy `zerone-1` runs
`2e37c4c86c31e67515d6aa07b6fd406083e012ea` (`2e37c4c` below), which has none of those
releases, so nothing here reaches it until it has run every upgrade in between (§7).
Publishing this document changes no chain.

**Terms.** A *seat* is one account's commitment in one review round. A *counted* seat's
reveal is counted in the verdict and paid; an *uncounted* seat (§4) is new here. *The
figure* is the amount a counted seat needs, 100 ZRN at both source commits named here.
A *controller* is whoever decides what an account signs. The *ante* is the checks a
transaction passes before its message runs. The *review budget* is the 55% of a claim's
payment that its reviewers share. *Policy 0* is the legacy review policy and *policy 1*
the neutral one. *Scheme 2* is the commitment hash that binds a reviewer to its chain
and round. A round has a *commit window*, then a *reveal window* that ends at its
reveal deadline, then an *aggregation deadline*. A round *completes* when it moves to
COMPLETE: a policy-1 round does so at its reveal deadline, or earlier once it has at
least `min_verifiers` seats and every seat has revealed. A *marker* is a stored flag
that turns a rule on. A *carrier* is a named upgrade that sets markers.

The design lets an account outside the founding household put a review on a round's
record without deciding the round. A seat that decides still needs the figure, which
this document does not set (§3, §6).

An agent session drafted it on 10 and 11 November 2026, after Yu agreed to an agent's
proposal to write it. Yu has not approved its rules or chosen any figure in it.
Statements about code were read at the named commit. "Run" marks a statement that was
also executed. The runs were scratch tests on copies of the named commits, outside this
repository: keeper-level, with the bank mocked, so a send between commitments was a
balance moved by the test and payments were read from the frozen settlement plan. Gas
figures were run through the full ante.

| When | What an account outside the household can do |
| --- | --- |
| 10 November 2026, `zerone-1` | Submit one claim, if the door admitted it. The door admits only agents on the household hub. It cannot review, and ZRN has no market. |
| 10 November 2026, `zerone-dev-1` | Practise a counted review with faucet ZRN. |
| After step A (§2) | Nothing new: A3 keeps the figure at 100 ZRN or above until step B. |
| After step B (§3) | Nothing new without the figure. A counted seat now locks it. |
| After step C (§4) | Leave an uncounted review, if someone pays its fees (§5). |

## 1. The gate at `bd76ea1d` and `2e37c4c`

1. **The balance is checked, not locked.** `SubmitCommitment` admits a commitment only
   if the signer's `uzrn` bank balance is at least 100,000,000 (100 ZRN). It reads the
   whole balance once, after the transaction fee has been taken, and moves nothing.
   (Review neutrality calls the balance spendable; the code reads the whole balance.)
   The figure is a constant in the source at both commits and on both consensus-repairs
   carriers; no parameter holds it. Reveal, counting and payment never read a balance
   again.
2. **One balance seats many accounts.** A round identifies a seat by its address only.
   Run at `bd76ea1d`: one 100 ZRN balance, moved from each address to the next inside
   one commit window, filled all 512 seats of a round, and each earlier seat then held
   nothing. Run at `2e37c4c`: five addresses were seated the same way. `2e37c4c` has no
   seat limit; the 512 limit (`CommitSeatHardCap`) arrived later, and its comment says
   a tighter limit must wait for seat bonds.
3. **Anyone can make accounts.** A commitment sent as an ordinary transaction needs an
   account registered with `zerone_auth`. Registration is permissionless and costs only
   its fee. That check is in the ante alone, and messages executed by the
   interchain-accounts host skip the ante. Governance proposal 2 on `zerone-1` shut
   that host: its parameters read `host_enabled` false with an empty allow-list at
   height 1,441,232 on 11 November 2026. No record on chain says which person or agent
   controls an account. The funding records the ante writes are read by nothing.
4. **Policy 1 counts accounts.** In a policy-1 round every valid reveal counts as one,
   whoever made it. A verdict needs at least `min_verifiers` reveals (3), one side
   holding at least `confidence_threshold` of them (770,000 per million), and at least
   three reveals on that side. Every valid, timely reveal takes an equal share of the
   review budget. No reveal is penalised for its vote or for being absent. The
   neutrality release says of itself that it "does not claim Sybil resistance".
5. **So the figure is the only bound on a verdict.** A holder of 100 ZRN can seat as
   many accounts as a verdict needs. Run at `bd76ea1d`: three seats with no other
   reviewer made a claim a fact at the capped confidence 880,000; a challenger with
   three seats turned a verified fact DISPROVEN, and the settlement returned the whole
   challenge deposit to them, 45% as refund and 55% as review pay. On `zerone-1` only
   the six accounts of item 7 hold the figure.
6. **Reviewing an ordinary claim costs more than it pays.** Run at `bd76ea1d`: a
   commitment and a reveal together used about 200,000 to 250,000 gas in rounds of one
   to four reviewers, and more for each seat already taken. That is at least 0.20 to
   0.25 ZRN at the floor of 1 uzrn per gas. A claim's review fee on `zerone-1` was at
   least 0.2 ZRN when read on 11 November 2026; three reviewers would share 0.11 ZRN of
   it. A contradiction or challenge is different: at `bd76ea1d` an 11 ZRN deposit gives
   its reviewers 6.05 ZRN to share.
7. **The door stops before review.** The agents' door gives an admitted account a seed
   of 0.222 ZRN and a fee allowance that pays for claiming the seed, registering and
   submitting one claim. The seed is below the figure, and the allowance does not pay
   for a commitment or a reveal. Read on 10 November 2026 at height 1,440,773, six
   accounts that can sign on `zerone-1` held at least 100 ZRN. Four are the founding
   household's panel keys listed in [`docs/FIRST-TRUTH.md`](../FIRST-TRUTH.md). The
   other two received their balance from the household's operations account, which is
   one of the four. All six have reviewed before. No other account holds enough, though
   any registered account could be lent the balance (item 2).

`zerone-1`'s own rounds use policy 0. There a reveal's weight follows a custom staking
record, only reviewers whose vote matches the verdict are paid, and the commitment
scheme predates scheme 2. This specification gives no entry on policy-0 rounds.

## 2. Step A: the figure becomes a parameter

A separate change turns the constant into a governable parameter with the same default
behind a default-off marker. Its draft was unpublished when this was written, and this
document does not describe it. A1 to A5 are what this design needs of it.

That change makes the figure a decision instead of a line of code. It does not make a
lower figure safe. While the balance is only checked, a figure of X lets any holder of
X take as many seats as it wants in every open round, for fees alone: about 0.2 to
0.3 ZRN a seat, rising with the round's size. The table gives the seats a buyer needs
for a verdict when other counted reviewers reveal against it.

| Reveals against the buyer | Seats the buyer needs | Buyer's share of the reveals |
| --- | --- | --- |
| 0 | 3 | 3 of 3 |
| 1 | 4 | 4 of 5 |
| 2 | 7 | 7 of 9 |
| 3 | 11 | 11 of 14 |
| 4 | 14 | 14 of 18 |

Blocking is cheaper than winning. One dissenting seat keeps up to three agreeing seats
below 770,000 (3 of 4 is 750,000), and two keep up to six below it (6 of 8).

**A1.** With the marker off, the constant governs exactly as before, and a change to
the parameter is refused.

**A2.** With the marker on, an absent or empty stored value is valid and reads as the
default 100,000,000 uzrn. Any other value must be a canonical positive `uzrn` integer:
base-10 digits, no sign, no leading zero, no whitespace, within uint64. Zero is
refused. Nothing may read "empty" as "no gate".

**A3.** While the deposit rule of §3 is not active on a chain, a figure below the
default is refused: at genesis validation, in any upgrade that writes it, and when
`MsgUpdateParams` executes.

**A4.** The upgrade that introduces the parameter writes no parameter, because an
absent value reads as the default. An upgrade that must change a knowledge parameter
decodes the stored set strictly and stops on a decode error. It does not read through
`GetParams`, which returns the defaults when the stored bytes fail to decode.

**A5.** Wherever a balance is checked and not locked, it is compared with the larger of
the stored figure and 100,000,000 uzrn. A lower figure applies only to seats that lock
it (D1, D2).

## 3. Step B: lock the figure as a deposit

**D1.** A round created while this rule is active, with review policy 1 (or 2, §4) and
commitment scheme 2, records its seat terms: the counted-seat deposit, copied from the
parameter of §2 at creation. A later parameter change does not alter an open round.
The parameter's meaning changes with this step: in rounds with seat terms it is an
amount to lock, and elsewhere it stays a check that is never below 100 ZRN (A5). Its
description must say so.

**D2.** In such a round, `SubmitCommitment` moves the round's deposit from the signer
to a module account that holds seat deposits and nothing else, and records the amount
with the seat. The account is on the app's blocked list, so nothing else can pay into
it. If the transfer fails, the commitment is refused. The transfer needs spendable
coins, where the check it replaces counted the whole balance. Like the module's other
deposits, it is a keeper transfer, so a governance halt on sending uzrn does not stop it.

The account's address can be computed by anyone, and before the binary that blocks it,
anyone can create an account there by sending it a coin or by creating a vesting
account at it. No key exists for a module address, so such an account has never
signed. The first deposit converts an account that has never signed into the module
account, and activation is refused while any other kind of account holds the address.
Coins found there become a surplus under D6.

**D3.** In such a round a seat is stored only by a transaction signed by the seat's
account. The handler itself checks that the signer is registered and allowed to
review, which only the ante checks at `bd76ea1d`. The validator vote-extension route
stores no commitment in such a round: it names an account that did not sign. Genesis
import stores seats without a transfer and is covered by D6.

**D4.** Every seat's deposit returns to the address that locked it, in full: whatever
the verdict, whatever the seat's vote, and whether or not it revealed. Nothing is
forfeited. A deposit prices holding a seat; it is not a penalty for a vote, an error or
silence. Each return is its own obligation with its own paid record, funded only from
the seat-deposit account and independent of reviewer pay and refunds. It becomes
payable when the round completes, and in any case from the round's aggregation
deadline, so a round that cannot complete does not hold deposits.

**D5.** Completion, or the aggregation deadline if the round has not completed by
then, freezes the round's returns once, as a plan beside the reviewer payment and the
claim refund. The existing bounded payment queue pays it. No plan may be
too large for that queue: its per-block budget covers the largest possible plan, or a
larger plan is paid alone in a block. A failed transfer leaves that return pending and
repeats nothing already paid.

**D6.** The balance of the seat-deposit account is never less than the sum of locked,
unreturned deposits. Tests check it, and genesis import and any in-place migration
refuse a state where it fails. It is not a condition that halts a chain. The target
design requires the two to be equal; this rule departs from it in one way: a surplus is
reported and not swept, so a stray coin cannot block an import. Export and import carry
open rounds' deposits and pending returns exactly.

**D7.** A round without seat terms, whether created before activation or under
policy 0, keeps the balance check, under A5.

What this changes: a balance can no longer be passed along. N seats held at once need
N deposits, each from its commitment until its round completes. At `zerone-1`'s window
lengths and 30-second blocks that is at most about 3 hours 20 minutes, about 100
minutes for a seat taken in the last commit block, and about a minute when at least
three counted seats exist and all reveal in the first reveal block. One silent counted
seat keeps every seat locked until the reveal deadline, and a round whose completion
fails keeps them until its aggregation deadline, about 3 hours 45 minutes.

At a figure of 100 ZRN the 14 seats of the last table row need 1,400 ZRN locked;
`zerone-1`'s whole supply was 13,673 ZRN on 10 November 2026, of which 2,562 ZRN was
not bonded to its validator. At 1 ZRN the same seats need 14 ZRN.

Honest reviewers lock too. A reviewer holding B sits on at most B divided by the figure
rounds at once, where at `bd76ea1d` it can sit on all of them. On 10 November 2026 four
of the six accounts of §1 item 7 held enough for only one seat at a time at 100 ZRN.
A reviewer holding exactly the figure must pay its reveal fee from what is left after
the lock. Measured by a review of the step-B draft (#199, handler gas, not through the
ante, not repeated): a
deposited commitment costs about 17,000 more gas than one without, and the chain's
first deposit about 30,700 more; each seat adds about 11 bytes to the round record, so
in a full round of 512 seats the last commitment costs about 232,000 more gas and the
last reveal about 220,000 more, on a base of about 6.33 million.

What it does not change: the deposit comes back, so it prices a verdict only in
capital held for between about a minute and a few hours, and in fees. A holder with
enough ZRN can still take the seats in the table. The figure therefore stays a real
decision about who decides, and this document does not make it.

Review neutrality introduced no new collateral and described the check as a recyclable
balance, not a locked bond. This step adds collateral, replaces that check in rounds
with seat terms, and adds a custody path that any census of knowledge liabilities must
include.

## 4. Step C: an uncounted seat

A deposit small enough for a newcomer cannot buy a seat that decides, because of the
table in §2. It can buy a place on the round's record.

**U1.** A claim admitted while this rule is active carries review policy 2, copied
into its rounds. Policy 2 is policy 1 except for the uncounted seats defined here.
Policy-1 rounds keep review neutrality's two statements exactly, that every admitted
account is counted and every valid, timely reveal is paid, and take no uncounted
seat. Policy 2 is refused before activation.

**U2.** `MsgSubmitCommitment` gains an enum field for the seat class. Its zero value
is counted, so existing clients commit counted seats. The reveal names no class. An
address holds at most one seat in a round, of either class.

**U3.** A policy-2 round's seat terms also carry the uncounted-seat deposit and the
round's limit on uncounted seats, copied from parameters at creation. An uncounted
seat locks that deposit under D2 to D6. An uncounted commitment beyond the limit is
refused. Counted seats are limited to 512 less the round's uncounted limit, so an
uncounted seat never refuses a counted one. Validation refuses a limit above 64 and a
deposit that is not a canonical positive `uzrn` integer. An absent deposit reads as its
default and an absent limit reads as 0, so a parameter update that omits them closes
uncounted seats and opens nothing.

**U4.** Uncounted commitments, reveals and statements are stored outside the round
record. `VerificationRound.reveals` keeps counted reveals only. At `bd76ea1d` every
commitment and every reveal rewrites the whole round record, so its gas grows with the
seats before it; no counted commitment, counted reveal or verdict may read or write an
uncounted byte, and no counted message's gas may depend on how many uncounted seats
exist. Completion reads uncounted seats only to freeze their deposit returns. Queries
and events return uncounted reveals with the round, marked uncounted. A reader built
before this rule sees only counted reveals.

**U5.** An uncounted reveal is not counted. Every outcome, payment and refund of a
round is that of the same round with its uncounted seats removed.

**U6.** An uncounted seat cannot hold a round open: early completion waits only for
counted seats. An uncounted reveal is accepted until the round's reveal deadline, even
after the round has completed. Scheme 2 fixed its vote at commitment, and it changes
nothing already decided.

**U7.** An uncounted reveal takes no share of the review budget. An uncounted seat
costs a small fraction of a counted one, so paying it would let whoever holds the most
cheap seats take most of every review budget from the counted reviewers. Paying
uncounted reviews is left for a version that can bound seats per controller.

**U8.** The party rules of claim integrity apply to both classes: a claim's submitter
and the single defender of a disputed fact take no seat.

**U9.** The chain derives nothing from uncounted reviews. It keeps no count of them per
account, compares none with the verdict, and uses none for admission, weight or
payment. The same holds off chain: no dashboard or indexer turns them into a score or
a ranking ([`docs/KARMA.md`](../KARMA.md)). Such a record can be curated, by not
revealing and by abandoning addresses, so agreement with verdicts is not evidence of
judgement. An uncounted review is a signed statement that readers can weigh, and it is
as permanent as a counted one. A counted seat is reached by locking the figure, and by
nothing computed from past reviews.

**S1.** In a round created while this rule is active, every commitment, counted or
uncounted, carries a controller statement: valid UTF-8, 1 to 256 bytes, stored and
emitted exactly as signed, with control characters and bidirectional formatting
characters refused. It says who decides this account's reviews, any relationship to
the claim's submitter or to the network's operator, and any uncertainty about either.
It names people only by public handle, role or address, never by private identifying
information. It is public from the moment of commitment and cannot be removed.

**S2.** The chain never reads the statement. Two seats with the same statement are both
admitted, and a statement proves nothing about independence. It is an unverified
statement under the signer's key, which anyone can quote back to it; naming another
party says nothing about that party. A reviewer that will not name its controller says
so, for example "not disclosed". A statement can also give away its own vote before the
reveal; that is the signer's choice and binds nobody else.

Proposed defaults, for the owner to set: an uncounted-seat deposit of 20,000 uzrn and
32 uncounted seats per round. 20,000 uzrn is what an admitted account has left of its
seed after paying its own claim's 0.2 ZRN review fee, less a margin; it works only if
someone else pays the account's transaction fees (§5). The limit of 32 is the default
genesis value; a stored parameter set without the field reads 0 (U3). Filling a round's
32 uncounted seats would cost about 3 ZRN or more in fees, estimated from the gas of a
counted commitment, and keep every other newcomer off that round's record.

This limit is a tight cap of the kind that the 512 limit's comment says must wait for
seat bonds. Here the bond cannot be reused but is never forfeited. That is accepted because
an uncounted seat neither decides nor holds a round open.

## 5. What an admitted agent could then do, and what still stops it

With §3 and §4 active, an account that holds 20,000 uzrn, and whose transaction fees
someone else pays, could commit to an uncounted seat, reveal a reasoned vote on someone
else's claim, and get its deposit back. Its review would stay on the round's record.

On its own it cannot even commit. The two transactions cost about 0.2 ZRN in fees,
taken before the deposit moves. That is nearly the whole 0.222 ZRN seed, and about
nine times the 0.022 ZRN left after the account's own claim. The door's fee allowance
paid for neither message on 11 November 2026. A fee allowance selects messages by
type, so one that pays for commitments and reveals pays for both classes; only the
counted-seat deposit keeps a fee-paid account off counted seats. Extending the
allowance is door policy, and the ZRN behind it is the household's. Both are the
owner's decisions.

It still cannot decide anything. Counted seats stay with whoever holds the figure, and
on `zerone-1` that is the six accounts of §1 item 7.

## 6. What this version does not solve

- **One controller, many seats.** Nothing here ties seats to controllers. The accepted
  target design ([`docs/AUTHORITATIVE-STATE.md`](../AUTHORITATIVE-STATE.md), §3.2 and
  §4.3) gives each controller one seat and one whole ballot, and until then allows only
  a disclosed, enumerated roster. A counted seat for an outside reviewer that does not
  depend on capital needs that roster, or the controller records the target names.
  Either is a new version of this design.
- **Capital still decides who counts.** The size of the deposit decides whether a
  review counts. The target forbids that: balance, stake and wealth-derived tiers must
  not change who is selected or what a vote weighs, and it uses escrow only to back a
  penalty. This version narrows one thing, since a balance can no longer be reused, and
  adds another, a second deposit level. Both are transitional.
- **Standing from record.** A reviewer's standing does not grow here from reviews that
  agreed with verdicts or survived challenge, for three reasons. Review neutrality
  retired standing built from agreement. The target gives each controller one whole
  ballot, so weight has nowhere to grow. And while one controller can hold many seats,
  such a record can be manufactured. A later version could admit to counted seats on
  adjudicated track record, which would change eligibility and not weight (§8).
- **Attendance.** The table in §2 assumes counted reviewers reveal against a buyer.
  After height 854,278 (4 August 2026) nobody committed to review on `zerone-1`. One
  claim was submitted in that time, on 28 October, and its round closed with no
  commitment (read 11 November 2026 at height 1,441,185). A deposit does not make
  reviewers attend.
- **Forfeiture.** Whether a seat that commits and never reveals should lose anything is
  a policy question this version answers "no" (D4). Changing that answer changes review
  neutrality, not only this document.
- **Crowding.** One controller can fill every uncounted seat of a round. Nothing here
  prices that beyond fees and the deposit.
- **Reveal storage.** Each counted reveal rewrites the round record, so a reveal's gas
  grows with the reveals before it, and the per-transaction gas limit bounds how many
  large reveals a round can take. This version does not change how counted reveals are
  stored.
- **Parameters read at completion.** Quorum and threshold are read when a round
  completes, not recorded at its creation. This version records only seat terms.

## 7. Activation and where it can run

Each step has its own marker: off on an existing chain until a named upgrade turns it
on, on in the default genesis of a new chain, read without gas as the consensus-repairs
stack reads its markers, and never turned off. The deposit marker requires the
parameter, review-neutrality and [record-integrity](knowledge-record-integrity-v1.md)
markers. The uncounted-seat marker
requires the deposit marker.

Before its marker is on, a commitment that names the uncounted class or carries a
statement is refused. So is an imported round or seat that carries seat terms, a class
or a statement. Raw encodings are checked before decoding, as the predecessor releases
check theirs.

One difference cannot wait for the marker. The seat-deposit account is on the app's
blocked list from the binary that adds it, whatever the marker says, so a send to its
address that the predecessor accepts is refused. A binary carrying step B is therefore
adopted only at an upgrade height, never swapped in on a running chain.

The steps change `MsgSubmitCommitment`, the round and seat records, the knowledge
parameters and genesis state. They need regenerated protobuf and TypeScript codecs,
and a re-pin of the [authority geometry](authority-geometry-v1.md) wherever a pinned
file changes.

- **A rehearsal chain first.** A fresh `zerone-rehearsal-*` chain built from the
  default genesis has every marker on. It is the first place the three steps can run
  together on real nodes. On 11 November 2026 such chains ran only on the owner's own
  machines, so a rehearsal tests the code, not outside entry.
- **`zerone-dev-1`.** It runs policy 1, and its faucet gives 1,000 development ZRN per
  address, so the 100 ZRN figure does not bind there: a faucet-funded account can take
  a counted seat. It can show the mechanics after it has run its outstanding upgrades
  and a carrier for these steps. It cannot show the exclusion this document is about.
- **`zerone-1`.** It needs every upgrade between `2e37c4c` and a carrier for these
  steps, each from its own binary. No such carrier exists. The existing
  consensus-repairs carriers refuse `zerone-1` by chain-id, and no recorded decision
  widens them.

## 8. Decisions this document leaves to the owner

1. The figure, on `zerone-1` and on any other shared network, and when to lower it.
2. The uncounted-seat deposit and the number of uncounted seats per round.
3. Whether the door's fee allowance pays for commitments and reveals, which would pay
   for both classes, and from what float.
4. Whether a disclosed roster of outside counted reviewers is the next version.
5. Whether uncounted reviews are ever paid, and whether an absent seat ever forfeits.
6. Whether and when any network activates any step, and whether a carrier is built for
   `zerone-1`. Each is a chain act.
7. Whether a reviewer's standing should ever grow from its record, for example from
   reviews that survived challenge. This version adds none (U9, §6).

## 9. What an implementation must show

- With every marker off, each message, record, event and stored byte is what the
  predecessor produces, and each new field is refused (§7). The one stated exception
  is the blocked seat-deposit address (§7).
- Each run of §1, added to the repository as a named test.
- A1 to A5: the default, the refusals, the floor while deposits are off, an upgrade
  that leaves every stored parameter as it was, and a round without seat terms that
  ignores a lower figure.
- D2 and D3: a signer that cannot lock is refused; one balance cannot hold two seats
  at once; no route stores a seat without a signed transaction and a deposit.
- D4 to D6: every outcome returns every deposit once, with real bank transfers; a
  round that cannot complete still returns them from its aggregation deadline; a
  failed transfer repeats nothing; a full round's plan does not stall the payment
  queue; the account balance covers locked deposits across restart, export and import.
- U2 and U3: one seat per address; a commitment beyond the uncounted limit is refused;
  uncounted seats never refuse a counted one.
- U4: a counted commitment and a counted reveal use the same gas with and without a
  full set of maximum-size uncounted reveals.
- U5 to U7: rounds with only uncounted reveals, with uncounted reveals on both sides of
  a narrow counted vote, with uncounted seats that never reveal, and with an uncounted
  reveal after early completion, each end, pay and refund as the same round without
  uncounted seats would.
- S1 and S2: the statement's bounds and refusals, and that no rule reads it.
- A rehearsal with signed transactions on two binaries across the upgrade height, with
  rounds open across it keeping their terms (D1, D7).

Synthetic reviewers in such tests show how the code behaves. They are not outside
review, and no test here shows that an outside reviewer has come in.
