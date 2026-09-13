#!/usr/bin/env python3
"""Prespecified sparse least-squares scaling experiment. Python standard library only."""
import argparse
import datetime
import hashlib
import json
import math
import os
from pathlib import Path
import platform
import random
import sys
import time


def canonical(value):
    return json.dumps(value, sort_keys=True, separators=(",", ":"), allow_nan=False).encode()


def sha(data):
    return hashlib.sha256(data).hexdigest()


def pin(path):
    return {"path": str(path.resolve()), "sha256": sha(path.read_bytes())}


def write_new(path, value):
    with path.open("x", encoding="utf-8") as stream:
        json.dump(value, stream, sort_keys=True, indent=2, allow_nan=False)
        stream.write("\n")


def finite(value, where):
    if not math.isfinite(value):
        raise ArithmeticError("Nonfinite value: " + where)
    return value


def dot(row, weights):
    return math.fsum(value * weights[i] for i, value in zip(row["indices"], row["values"]))


def make_dataset(protocol):
    d, n = protocol["dimension"], protocol["rows"]
    assert protocol["basis_rows"] == d and n >= d
    datasets = []
    for seed in protocol["seeds"]:
        rng = random.Random(seed)
        truth = [rng.choice(protocol["solution_values"]) for _ in range(d)]
        rows = [{"indices": [i], "values": [1]} for i in range(d)]
        for _ in range(n - d):
            indices = sorted(rng.sample(range(d), protocol["sparse_nonzeros_per_remaining_row"]))
            rows.append({"indices": indices,
                         "values": [rng.choice(protocol["feature_values"]) for _ in indices]})
        for row in rows:
            row["y"] = dot(row, truth)
        order_rng = random.Random(1000000 + seed)
        order = []
        while len(order) < protocol["updates"]:
            epoch = list(range(n))
            order_rng.shuffle(epoch)
            order.extend(epoch)
        datasets.append({"seed": seed, "dimension": d, "known_minimizer": truth,
                         "rows": rows, "sample_order": order[:protocol["updates"]]})
    return {"schema": "zerone-sparse-scaling-data/v1", "datasets": datasets}


def metrics(data, weights):
    n, d = len(data["rows"]), data["dimension"]
    squares, gradient = [], [0.0] * d
    for row in data["rows"]:
        residual = finite(dot(row, weights) - row["y"], "evaluation residual")
        squares.append(finite(residual * residual, "evaluation square"))
        for i, x in zip(row["indices"], row["values"]):
            gradient[i] += residual * x
    objective = finite(math.fsum(squares) / (2 * n), "full objective")
    gradient_norm = finite(math.sqrt(math.fsum(g * g for g in gradient)) / n, "unscaled full gradient norm")
    return objective, gradient_norm


def denominator(kind, accumulator, scale, protocol):
    e = protocol["epsilon_outside_base"]
    e2 = protocol["epsilon_inside_matched_base"]
    if kind == "ideal":
        return math.sqrt(accumulator)
    if kind == "outside_fixed":
        return math.sqrt(accumulator) + e
    if kind == "inside_matched_fixed":
        return math.sqrt(accumulator + e2)
    if kind == "inside_same_numeric_fixed":
        return math.sqrt(accumulator + e)
    if kind == "outside_coscaled":
        return math.sqrt(accumulator) + scale * e
    if kind == "inside_matched_coscaled":
        return math.sqrt(accumulator + scale * scale * e2)
    raise ValueError("Unknown variant " + kind)


def run_one(data, scale, variant, protocol):
    weights = [0.0] * data["dimension"]
    accumulators = [0.0] * data["dimension"]
    trajectory = [tuple(weights)]
    checkpoints = []
    initial_objective, _ = metrics(data, weights)
    assert initial_objective > 0
    zero_denominator_updates = 0

    def checkpoint(step):
        objective, gradient_norm = metrics(data, weights)
        checkpoints.append({"step": step, "unscaled_objective": objective,
                            "normalized_objective": objective / initial_objective,
                            "unscaled_full_gradient_norm": gradient_norm,
                            "weights": list(weights)})

    checkpoint(0)
    for step, row_index in enumerate(data["sample_order"], start=1):
        row = data["rows"][row_index]
        residual = finite(dot(row, weights) - row["y"], f"training residual step{step}")
        # One unchanged sample loss, multiplied uniformly by c. No feature scaling.
        for i, x in zip(row["indices"], row["values"]):
            gradient = finite(scale * residual * x, f"gradient step{step} coordinate{i}")
            accumulators[i] = finite(accumulators[i] + gradient * gradient, f"accumulator step{step} coordinate{i}")
            den = finite(denominator(variant, accumulators[i], scale, protocol), "denominator")
            if den == 0:
                assert variant == "ideal" and gradient == 0 and accumulators[i] == 0
                delta = 0.0  # Explicit ideal 0/0 = 0; untouched coordinates also stay fixed.
                zero_denominator_updates += 1
            else:
                delta = finite(protocol["learning_rate"] * gradient / den, "update")
            weights[i] = finite(weights[i] - delta, f"weight step{step} coordinate{i}")
        trajectory.append(tuple(weights))
        if step % protocol["checkpoint_every"] == 0 or step == protocol["updates"]:
            checkpoint(step)
    assert len(trajectory) == protocol["updates"] + 1
    return {"status": "PASS", "seed": data["seed"], "scale": scale, "variant": variant,
            "dataset_sha256": sha(canonical(data)), "trajectory_sha256": sha(canonical(trajectory)),
            "trajectory_shape": [len(trajectory), data["dimension"]],
            "checkpoints": checkpoints, "zero_denominator_updates": zero_denominator_updates,
            "final_accumulator_min": min(accumulators), "final_accumulator_max": max(accumulators)}, trajectory


def compare(left, right):
    assert len(left) == len(right)
    worst = {"max_abs_coordinate_difference": 0.0, "step": 0, "coordinate": 0}
    for step, (a, b) in enumerate(zip(left, right)):
        assert len(a) == len(b)
        for coordinate, (x, y) in enumerate(zip(a, b)):
            difference = finite(abs(x - y), "trajectory difference")
            if difference > worst["max_abs_coordinate_difference"]:
                worst = {"max_abs_coordinate_difference": difference, "step": step, "coordinate": coordinate}
    return worst


def execute(protocol, dataset):
    runs, comparisons = [], []
    scales = [float(c) for c in protocol["scale_strings"]]
    variants = [variant["id"] for variant in protocol["variants"]]
    assert all(c > 0 and math.isfinite(c) for c in scales)
    assert protocol["epsilon_inside_matched_base"] == protocol["epsilon_outside_base"] ** 2
    for data in dataset["datasets"]:
        # Saved basis rows make the claimed minimizer uniquely identifiable.
        for i in range(data["dimension"]):
            assert data["rows"][i]["indices"] == [i] and data["rows"][i]["values"] == [1]
        assert metrics(data, data["known_minimizer"]) == (0.0, 0.0)
        trajectories = {}
        for variant in variants:
            for c in scales:
                try:
                    record, trajectory = run_one(data, c, variant, protocol)
                    trajectories[variant, c] = trajectory
                    runs.append(record)
                except (ArithmeticError, AssertionError, ValueError) as error:
                    runs.append({"status": "FAIL", "seed": data["seed"], "scale": c,
                                 "variant": variant, "error": str(error)})
        for variant in variants:
            baseline = {"outside_coscaled": "outside_fixed", "inside_matched_coscaled": "inside_matched_fixed"}.get(variant, variant)
            for c in scales:
                if (variant, c) not in trajectories or (baseline, 1.0) not in trajectories:
                    comparisons.append({"seed": data["seed"], "scale": c, "variant": variant,
                                        "check_pass": False, "error": "Missing trajectory after failed run"})
                    continue
                value = compare(trajectories[variant, c], trajectories[baseline, 1.0])
                check = "descriptive_difference"
                passed = True
                if variant == "ideal" and c != 1.0:
                    check, passed = "ideal_invariance", value["max_abs_coordinate_difference"] <= 1e-10
                elif variant.endswith("coscaled"):
                    check, passed = "coscaled_equivalence", value["max_abs_coordinate_difference"] <= 1e-10
                elif variant.endswith("fixed") and c == 1e-6:
                    check, passed = "fixed_stabilizer_break", value["max_abs_coordinate_difference"] > 1e-6
                comparisons.append({"seed": data["seed"], "scale": c, "variant": variant,
                                    "baseline_variant": baseline, "baseline_scale": 1.0,
                                    "check": check, "check_pass": passed, **value})
    assert len(runs) == protocol["number_of_runs"] == 120
    passed = all(run["status"] == "PASS" for run in runs) and all(c["check_pass"] for c in comparisons)
    return {"pass": passed, "runs": runs, "trajectory_comparisons": comparisons}


def main():
    if not __debug__:
        raise RuntimeError("Run without -O; assertions are part of the experiment checks.")
    os.umask(0o077)
    parser = argparse.ArgumentParser()
    sub = parser.add_subparsers(dest="command", required=True)
    prepare = sub.add_parser("prepare")
    prepare.add_argument("--protocol", type=Path, required=True)
    prepare.add_argument("--dataset", type=Path, required=True)
    prepare.add_argument("--manifest", type=Path, required=True)
    run = sub.add_parser("run")
    run.add_argument("--manifest", type=Path, required=True)
    run.add_argument("--output", type=Path, required=True)
    args = parser.parse_args()
    if args.command == "prepare":
        protocol = json.loads(args.protocol.read_text())
        write_new(args.dataset, make_dataset(protocol))
        manifest = {"schema": "zerone-sparse-scaling-experiment-manifest/v1",
                    "prepared_before_results_utc": datetime.datetime.now(datetime.timezone.utc).isoformat(),
                    "protocol": pin(args.protocol), "dataset": pin(args.dataset), "code": pin(Path(__file__)),
                    "preparation_scope": "Local same-operator pre-specification, not independently timestamped preregistration."}
        write_new(args.manifest, manifest)
        print(json.dumps({"prepared": True, "manifest": pin(args.manifest)}))
        return 0
    manifest = json.loads(args.manifest.read_text())
    for key in ("protocol", "dataset", "code"):
        assert pin(Path(manifest[key]["path"])) == manifest[key], "Frozen input drift: " + key
    assert Path(manifest["code"]["path"]).resolve() == Path(__file__).resolve()
    protocol = json.loads(Path(manifest["protocol"]["path"]).read_text())
    dataset = json.loads(Path(manifest["dataset"]["path"]).read_text())
    start = time.perf_counter()
    content = execute(protocol, dataset)
    result = {"schema": "zerone-sparse-scaling-experiment-results/v1",
              "completed_at_utc": datetime.datetime.now(datetime.timezone.utc).isoformat(),
              "experiment_manifest": pin(args.manifest), "source_inputs": manifest,
              "environment": {"python": sys.version, "platform": platform.platform(),
                              "implementation": platform.python_implementation(), "runtime_dependencies": "Python standard library only"},
              "computational_content_sha256": sha(canonical(content)), "runtime_seconds": time.perf_counter() - start,
              "scope": "Prespecified120 finite runs on five generated noiseless sparse convex problems; same operator, not external replication or a model-performance benchmark.",
              **content}
    write_new(args.output, result)
    print(json.dumps({"pass": content["pass"], "runs": len(content["runs"]),
                      "output": pin(args.output), "runtime_seconds": result["runtime_seconds"]}))
    return 0 if content["pass"] else 1


if __name__ == "__main__":
    sys.exit(main())
