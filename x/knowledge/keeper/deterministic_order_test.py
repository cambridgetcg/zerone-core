#!/usr/bin/env python3
"""Force both map traversal orders in the real knowledge keeper (no source edits).

Run from any directory, with a writable scratch directory:
  GOFLAGS=-mod=readonly GOPROXY=off python3 -B deterministic_order_test.py \
      --artifacts <scratch>

To reproduce the predecessor defect with the SAME Go assertions:
  ... --source-ref 77521a0daf14420836483ae9d8ea3fbcc1ae0d40 --expect-defect

The compiler overlays change only map traversal, not tally/transfer bodies.
On repaired source the tally is order-independent; pruning key collection is
perturbed BEFORE its enabled sort. Historical overlays retain the new marker
API solely so current tests compile; historical consumers do not read it.
No random-map sampling, source restoration, dependency download or node is used.
Artifacts are retained, including compiler overlays and complete Go JSON logs.
"""

import argparse
import json
import os
from pathlib import Path
import subprocess
import tempfile


ROOT = Path(__file__).resolve().parents[3]
OWNER = Path("x/knowledge/keeper")
CASES = {
    "training_economics.go": (
        "stakeTally", "v", "s", "types.AugmentationVerdict", "uint64", "f3bVerdictKeys"
    ),
    "vindication.go": (
        "allPending", "factId", "entries", "string", "[]types.VindicationEntry", "f3bFactKeys"
    ),
}
TESTS = {"TestReformulationOrderTies", "TestReformulationOrderControls", "TestVindicationOrderTransfers"}


def force_order(source, spec, descending):
    mapping, key, value, key_type, value_type, helper = spec
    old = f"for {key}, {value} := range {mapping} {{"
    fixed = f"for {key} := range {mapping} {{"
    if source.count(old) == 1 and fixed not in source:
        needle = old
        replacement = f"for _, {key} := range {helper}({mapping}) {{\n\t\t{value} := {mapping}[{key}]"
    elif source.count(fixed) == 1 and old not in source:
        needle = fixed
        replacement = f"for _, {key} := range {helper}({mapping}) {{"
    else:
        raise AssertionError(f"unrecognized map loop for {mapping}; inspect before updating the harness")
    comparison = "<" if descending else ">"
    # Insertion sort in a scratch-only helper avoids changing source imports.
    helper_source = f"""
func {helper}(values map[{key_type}]{value_type}) []{key_type} {{
    keys := make([]{key_type}, 0, len(values))
    for key := range values {{ keys = append(keys, key) }}
    for i := 1; i < len(keys); i++ {{
        for j := i; j > 0 && keys[j-1] {comparison} keys[j]; j-- {{
            keys[j-1], keys[j] = keys[j], keys[j-1]
        }}
    }}
    return keys
}}
"""
    return source.replace(needle, replacement, 1) + helper_source


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--artifacts", type=Path, required=True)
    parser.add_argument("--source-ref", help="overlay only the two production owners from this local Git object")
    parser.add_argument("--expect-defect", action="store_true")
    args = parser.parse_args()
    artifacts = Path(tempfile.mkdtemp(prefix="forced-order-", dir=args.artifacts.resolve()))
    print(f"artifacts: {artifacts}", flush=True)
    originals = {}
    current_training = (ROOT / OWNER / "training_economics.go").read_text()
    marker_api = current_training[current_training.index("// EconomicDeterminismEnabledStoreKey"):current_training.index("// ─── Training-value weight")]
    for name in CASES:
        path = OWNER / name
        originals[name] = (
            subprocess.check_output(["git", "show", f"{args.source_ref}:{path}"], cwd=ROOT, text=True)
            if args.source_ref else (ROOT / path).read_text()
        )
        if args.source_ref and name == "training_economics.go":
            assert "EconomicDeterminismEnabledStoreKey" not in originals[name], "use current-source mode for marker-aware revisions"
            originals[name] = originals[name].replace('"context"', '"bytes"\n\t"context"', 1) + "\n" + marker_api
    env = dict(os.environ, GOFLAGS="-mod=readonly", GOPROXY="off", TMPDIR=str(artifacts))
    results = {}
    for order in ("ascending", "descending"):
        directory = artifacts / order
        directory.mkdir()
        replacements = {}
        for name, spec in CASES.items():
            target = directory / name
            target.write_text(force_order(originals[name], spec, order == "descending"))
            replacements[str(ROOT / OWNER / name)] = str(target)
        overlay = directory / "overlay.json"
        overlay.write_text(json.dumps({"Replace": replacements}, indent=2) + "\n")
        command = ["go", "test", "-overlay", str(overlay), "./x/knowledge/keeper", "-run",
                   "^(TestReformulationOrder|TestVindicationOrder)", "-count=1", "-json"]
        with (directory / "test.jsonl").open("w") as stdout, (directory / "stderr.txt").open("w") as stderr:
            result = subprocess.run(command, cwd=ROOT, env=env, stdout=stdout, stderr=stderr, timeout=300)
        outcomes = {}
        for line in (directory / "test.jsonl").read_text().splitlines():
            event = json.loads(line)
            if event.get("Test") in TESTS and event["Action"] in {"pass", "fail", "skip"}:
                outcomes[event["Test"]] = event["Action"]
        results[order] = {"returncode": result.returncode, "tests": outcomes, "command": command}
        print(json.dumps({"order": order, **results[order]}, sort_keys=True), flush=True)
        # A missing module, build error, timeout or skipped test is not a RED proof.
        assert set(outcomes) == TESTS and "skip" not in outcomes.values(), "all real Go tests must execute"
    (artifacts / "results.json").write_text(json.dumps(results, indent=2) + "\n")
    if args.expect_defect:
        # No-winner-on-tie rejects BOTH predecessor enum choices. Overflow is
        # also visible in the controls, independently of traversal direction.
        assert results["ascending"]["returncode"] == 1
        assert results["ascending"]["tests"]["TestReformulationOrderTies"] == "fail"
        assert results["ascending"]["tests"]["TestReformulationOrderControls"] == "fail"
        assert results["ascending"]["tests"]["TestVindicationOrderTransfers"] == "pass"
        assert results["descending"]["returncode"] == 1
        for test in ("TestReformulationOrderTies", "TestVindicationOrderTransfers"):
            assert results["descending"]["tests"][test] == "fail", f"defect not detected: {test}"
        print("PASS: both predecessor defects detected under forced descending traversal")
    else:
        assert results["ascending"]["returncode"] == 0
        assert set(results["ascending"]["tests"].values()) == {"pass"}
        assert results["descending"]["returncode"] == 0
        assert set(results["descending"]["tests"].values()) == {"pass"}
        print("PASS: both forced traversal orders preserve the required outcomes")


if __name__ == "__main__":
    main()
