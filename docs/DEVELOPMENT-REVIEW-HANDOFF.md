# Development review handoff

Status: **prepared 23 September 2026; participant review not yet completed.**
This handoff supports one fresh, bounded review on `zerone-dev-1`. The
[participant guide](SHARED-DEVELOPMENT.md) supplies commands and the
[development publication](https://zerone.ai/development/) supplies exact pins.
The chain still has one operator-controlled validator.

## Review the package before opening a round

A useful first subject is the additive knowledge-11 participant package. A
willing reviewer can inspect the public files before a time-limited claim is
submitted. The proposed scope is whether the published package matches its
recorded source/descriptor and preserves access to the original knowledge-10
replay material. This is a packaging claim, not scientific validation, binary
reproducibility or a security audit unless the reviewer actually performs that
additional work.

The [package review example](examples/development-review-participant-package.md)
provides a bounded draft assertion and reproducible byte checks. It has not
opened a native review round or selected the reviewer's judgment.

Prepare the following evidence for the reviewer:

| Evidence | Bounded check |
| --- | --- |
| Current pinned `release.json`, selected archive and `SOURCE.json` | Check exact archive/binary hashes and the declared runtime source `5542b221864ca0080e143fa82abbef4cd2064f71`. A declared source alone does not prove the binary was built from it. |
| Packaged `network.json` and published descriptor pin | Verify their exact hash and knowledge-11 runtime binding; retain the publication version and observation height. |
| [Original release](../deploy/networks/zerone-dev-1/release-knowledge-10.json) and [descriptor](../deploy/networks/zerone-dev-1/network-knowledge-10.json) | Check preserved predecessor identities against the earlier publication; distinguish current participant initialization from genesis replay. |
| Original upgrade bundle and [runtime instructions](../deploy/networks/zerone-dev-1/README.md#stage-a-knowledge-10-to-11-upgrade) | Check the separate replay path across height 300,000; do not claim replay was executed unless it was. |

The author should submit only what these actual checks support, with evidence
locations/hashes, method and limits in the ordinary claim's reasoning. The
reviewer chooses their own vote, reason, confidence, method and scope. A
preparing agent must not choose or sign that judgment on their behalf.

## Keep control and outcomes explicit

Each participant keeps their own home and keys. Record the public reviewer
address, relevant author/operator relationship and any remaining uncertainty
about control, without publishing private identifying information. A reviewer
within the operator's control context may make their own judgment, but that
exercise does not establish a separate effective controller. Another account,
model or machine alone does not establish independence. Independent-controller
participation remains a separate milestone where it has not been established.

The current minimum for a decisive review is **three admitted accounts**.
One real reviewer may therefore leave an `INCONCLUSIVE` result and still be
paid for a valid timely reveal. Do not create dummy reviewers, seek an
unsupported favorable vote or change a review merely to reach the threshold.
Refusal, silence and stopping carry no adverse person score.

## Run the native workflow

1. Verify the current participant package and initialize each new home using
   the guide. Existing homes keep their original files and pins. Inspect the
   evidence and plan the scope before opening the round.
2. Submit the bounded fresh claim. Retain its actual Claim ID, signed receipt,
   positive committed height, execution code zero and immutable `funding_terms`.
   The old prime-polynomial exercise is historical and cannot reopen its review
   window for this task.
3. The reviewer signs their own `commit` and `reveal` against that Claim ID.
   Use the actual round deadlines; `--wait-for-phase` can time out and does not
   extend them. Preserve the revealed reason and actual terminal outcome.
4. Retain any warranted correction with its original assertion and new artifact
   hash. A journal response can reference the exact Claim ID. A narrower native
   claim is a new admission with new fees; it does not replace the old record.
   A native `challenge` requires a real derived **Fact ID** and counterevidence.
   Without a Fact, leave that path unexercised. Do not manufacture an error or
   acceptance to force it. See [record selection](MODULE-CONSOLIDATION-SEQUENCE.md#choose-the-record-before-admission).
5. Record funding terms, reviewer/refund recipients and amounts, and
   `paid_at_block`. Zero means pending; a positive marker records application
   state. Compare bank observations at stated heights, accounting for other
   transactions and fees. Neither a payment plan nor a paid marker alone is an
   independently verified bank transfer. A separately replaying node can
   strengthen state checks within the disclosed single-operator trust model.

Timely valid reveals share the fixed 55% review pool, including dissent and
inconclusive outcomes. Unused review budget returns when no eligible reveal
exists; challenge remainders refund on every terminal verdict. Preserve pending
obligations instead of creating another claim to obtain payment. See the
[funding specification](specs/knowledge-fund-settlement-v1.md).

Keep a small public result record linking the exact pins, claim/review receipts,
evidence, actual outcome, any correction and payment observations. State which
checks were executed and which were not. Report controller independence,
review completion, correction and payment separately; this prepared handoff
and local automated tests establish none of those participant results.
