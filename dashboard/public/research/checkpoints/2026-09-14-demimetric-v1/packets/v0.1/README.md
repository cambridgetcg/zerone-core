# Technical review packet 2026-001 / v0.1

Start with [REPORT.pdf](REPORT.pdf) or the accessible [HTML reading copy](REPORT.html).
[REPORT.md](REPORT.md) is the editable scientific source. The report identifies
which statements are counterexamples, which require an interpretation of the
paper, and which assumptions make the proposed auxiliary repair valid.

From this directory, using Python 3.10+:

```sh
python3 -B verify_packet.py
python3 -B verify_exact.py
```

The first command checks the manifest and exact file set, including the verifier
itself. Compare the manifest hash against a separately retained copy if you
need to detect a replaced manifest. A self-consistent packet is not an identity
signature, external timestamp or proof of truth. The second command repeats
the exact rational examples and bounded recurrence checks. Its JSON should
match `exact-result.json`; a nonzero exit means failure. No network access or
third-party Python library is needed. Optimized Python mode is refused because
it would disable assertions.

The infinite-time and Hilbert-space arguments require proof review; they are
not established by a finite run. The rho=1 comparison is a control outside the
paper's stated rho interval. `SOURCES.json` identifies the publisher versions
inspected; full third-party papers are not included. `REVIEW.md` records the
separately tasked internal check and its limitations. `AI-DISCLOSURE.json`
records assistance and the presently missing human adoption.

This packet is a draft for an outside reviewer. It has not been submitted to a
journal or institution, endorsed by the source authors, externally reviewed,
or recorded on-chain. Keep the version and digest with any later assessment.
