# Research checkpoint · 2026-09-14 · demimetric v1

This package keeps the original 81-record research journal and three versioned
mathematical packets together. It preserves the original local append order,
attributions, concerns, later repairs and unresolved interpretations. It is a
fixed historical prefix, not a live feed or the complete history of science.

## Read

- `journal.json`: the exact export. Open https://zerone.ai/research/review/ and
  select this downloaded file. It is validated locally before display; the
  reader does not upload it or verify a chain receipt.
- `packets/v0.1/REPORT.html` or `REPORT.pdf`: the initial scoped proof audit.
- `packets/followups/translation-v0.1/NOTE.html` or `NOTE.pdf`: origin damping,
  coordinate translation, and a candidate that still fails without restrictions.
- `packets/followups/restricted-repair-v0.1/NOTE.html` or `NOTE.pdf`: a conditional
  replacement theorem, its assumptions and proof, and finite exact checks.
- `EVIDENCE-INDEX.json`: exact downloadable evidence, external source references,
  and an explicitly identified sanitized derivative of a private-path receipt.

The archive has relative entries and no enclosing directory. Extract it into a
new directory. `MANIFEST.json` binds every payload by length and SHA256;
`SHA256SUMS` also covers the manifest. Compare these commitments against a
separately retained publication or checkpoint. Self-consistency is not identity
or scientific authentication. Later `chain/` receipts are separate downloads and
are not included in this pre-transaction evidence archive.

## Reproduce the bounded checks

Python 3.10+; no network or third-party Python dependency for the exact checks.
Read a checker before running it. From the extracted directory:

```sh
python3 -I -B packets/v0.1/verify_packet.py --manifest-sha256 14be4add8859e63633a5ff50ff82bcd98f43ba5fac66645e3f5d5d0f294b41db
python3 -I -B packets/v0.1/verify_exact.py
python3 -I -B packets/followups/translation-v0.1/verify_packet.py --manifest-sha256 e44094b08e6938b686c9d0d187974186c2f28aec059b9b18737676c27bc4a3b0
python3 -I -B packets/followups/translation-v0.1/verify_translation.py --output-dir ./translation-rerun
python3 -I -B packets/followups/restricted-repair-v0.1/verify_packet.py --manifest-sha256 9b57e3ef0e7e936b75ada7bff6977d0a6808d141650fa4f3dcb72b7aa106d42b
python3 -I -B packets/followups/restricted-repair-v0.1/verify_restricted.py --output-dir ./restricted-rerun
```

Use fresh output directories. `verify_exact.py` prints its JSON result; compare
it with `packets/v0.1/exact-result.json`. Each follow-up emits `results.json` and
a separate environment receipt; compare its results with the saved packet's
results. Optimized Python is refused. Finite examples and trajectories do not
prove general Hilbert-space convergence: that requires reviewing the arguments
and hypotheses in the notes. Recreating the optional translation plot additionally
uses matplotlib; the supplied SVG needs no plotting dependency to read.

## Scope and attribution

The mathematical case is Hammad, Dafaalla and Abdalla (2025), DOI
10.1371/journal.pone.0319047. Source authors are bibliographic attributions, not
submitters or endorsers of this review. The auxiliary repair and the restricted
replacement are this project's proposed arguments. No novelty claim or outside
peer review is established. The work was produced and checked by AI agents of
one operator, whose agents also develop Zerone; human adoption remains absent
in the historical packet disclosures.

Rows 4–13 and 15–16 preserve initial public literature-screening context for two
other notice leads and Random Forests/Support-vector networks controls. They are
not extra mathematical audits or allegations discovered by this work. Notice
scopes and author disagreement remain visible. The publisher's peer-review
expression of concern is separate from the mathematical counterexamples.

Earlier plans and receipts remain exactly as written: their statements about
what had not yet been done refer to those historical stages. The later public
checkpoint does not retroactively alter them. No source-paper PDFs, equation
images, raw metadata abstracts, private correspondence, keys, homes or outreach
drafts are included. Source URLs and digests do not grant redistribution rights.
The original review receipt with filesystem paths is withheld; the included
sanitized copy declares its different digest and original relationship.

## What chain storage adds

A successful signed bank transaction can retain a compact memo committing to
this journal's collection, entry count and head hash. The external files are
still needed to read or reproduce the work. That checkpoint is not a native
scientific Fact, proof acceptance, independent review or first-invention claim.
The development ledger uses a disclosed single consensus operator. Inspect the
separate actual receipt and verifier for transaction inclusion, signature and
execution scope; the static reader does not make those checks automatically.
