# AdaGrad loss-scaling evidence · v1

This package contains a bounded, reproducible computation: 120 prespecified runs on five noiseless sparse least-squares datasets compare six explicitly specified AdaGrad variants. Positive scaling changes the whole loss and its gradient, not features or the minimizer. The report gives the exact formulas, zero-step convention, results and limits.

**The data and code are inspectable; acceptance is not implied.** The experiment and the original checking agents share one operator. No independent external review or paper-author endorsement is claimed. A reader who reproduces it should state what they actually checked and any disagreement. A separate local source-report claim from the earlier pilot is not acceptance of this new computational candidate.

## Retrieve and verify

Public evidence base: <https://zerone.ai/research/adagrad/v1/>.

Download `adagrad-evidence-v1.zip` and obtain its expected archive SHA256 and `MANIFEST.json` SHA256 from the evidence page or a particular attributed claim. Verify the archive before extraction. The archive contains only regular relative files, under `adagrad-evidence-v1/`, with no symlinks or keys. Extract it into a new directory, then inspect `README.md`, `reproduce.py`, `experiment.py`, `protocol.json` and `report.md` before running code.

With a Python 3 interpreter, set `EXPECTED_MANIFEST_SHA256` to the expected 64-character lowercase digest and run from the extracted package directory:

```sh
python3 -I -B reproduce.py --manifest-sha256 "$EXPECTED_MANIFEST_SHA256" --verify-only
python3 -I -B reproduce.py --manifest-sha256 "$EXPECTED_MANIFEST_SHA256" --output my-result.json
```

The runner verifies every manifest-listed file, including itself and the frozen optimizer, before optimizer execution. It uses only the Python standard library, takes no network action, performs no signing, and does not read a participant home or key. Its inputs resolve beside `reproduce.py`; it works from another current directory too. The output must be a new file. Never treat a digest as proof of scientific truth or authorship by itself.

Exit 0 means exact computational content matched and all prespecified checks passed. Exit 1 means numerical disagreement or failed checks; the full output is retained for inspection. Exit 2 means package/output validation failed. Platform-level rounding can change exact float content even if the recorded tolerance checks pass. The original environment was CPython 3.14.3 on macOS 15.3 ARM64; no claim of all-platform byte equality is made. Do not silently discard disagreement or turn a matching rerun into endorsement.

## Files and scientific scope

- `claim.json`: the bounded computational candidate, assumptions and limitations.
- `protocol.json`, `dataset.json`, `experiment.py`: byte-identical original protocol, exact data/order and standard-library optimizer.
- `results.json`: all 120 runs, every saved checkpoint, full-trajectory hashes and all 120 comparisons; canonical scientific content SHA256 `55f0f8d77b07f81b218ffe808342a6c6c89b80f434bd04cb4f2baffa73e0b787`.
- `reproduce.py`: package verification and portable execution wrapper; it calls the unchanged optimizer.
- `report.md`, `loss-scaling.png`, `loss-scaling.svg`: analysis and corrected descriptive figure. Matplotlib 3.11.0 generated the figure; it is not required for numerical reproduction.
- `PROVENANCE.json`: source citation, local preparation history, authorship/scope and packaging transformations.
- `MANIFEST.json`, `LICENSE`, this README: payload integrity, reuse terms and instructions. The manifest does not hash itself or the ZIP; the evidence page pins those separately.

The comparison is finite and deterministic, without observation noise. The matched outside/inside epsilon pair shares a zero-gradient denominator floor but remains two different update functions. The same-numeric-inside variant is deliberately unmatched. No general optimizer superiority, noisy/deep-learning performance, novelty, independent controller identity or scientific truth is established.

## Source and reuse

John Duchi, Elad Hazan and Yoram Singer, *Adaptive Subgradient Methods for Online Learning and Stochastic Optimization*, JMLR 12 (2011), 2121–2159: <https://jmlr.org/papers/volume12/duchi11a/duchi11a.pdf>. Eq. 1 motivates the diagonal update. Our follow-up specifies the epsilon variants, generated data and experiment; the paper authors did not author or endorse these results. The PDF is not bundled or relicensed.

Copyright 2026 Zerone contributors (Yu and Ai). The authored package code, generated data/results, report and figures are offered under Apache License 2.0; see the exact `LICENSE` text. The grant does not cover the cited paper or confer rights to third-party names/marks. No private paths, identities, transaction fixtures, homes, keys or source PDF are included.
