#!/usr/bin/env python3
"""Exact finite translation checks for an explicitly reconstructed Eq. 9.

No network, paper code, external packages, or claims about infinite trajectories.
"""
from __future__ import annotations

import argparse
from dataclasses import dataclass
from decimal import Decimal, localcontext
from fractions import Fraction as F
import hashlib
import json
import os
from pathlib import Path
import sys

if not __debug__:
    print("Refused: this checker must run without Python optimization (-O/-OO).", file=sys.stderr)
    raise SystemExit(2)


def require(condition, message):
    if not condition:
        raise ValueError(message)


def exact(x):
    return f"{x.numerator}/{x.denominator}"


def approximate(x):
    with localcontext() as ctx:
        ctx.prec = 18
        return str(Decimal(x.numerator) / Decimal(x.denominator))


def value(x):
    return {"fraction": exact(x), "decimal_approximation": approximate(x)}


@dataclass(frozen=True)
class Affine:
    slope: F
    intercept: F

    def __call__(self, x):
        return self.slope * x + self.intercept

    def compose(self, other):
        return Affine(self.slope * other.slope,
                      self.slope * other.intercept + self.intercept)

    def conjugate(self, shift):
        # Point-valued map in y=x-shift coordinates: T_t(y)=T(y+t)-t.
        return Affine(self.slope, self.intercept + (self.slope - 1) * shift)

    def shift_input(self, shift):
        # Monotone B outputs are displacement/dual vectors: B_t(y)=B(y+t).
        return Affine(self.slope, self.intercept + self.slope * shift)

    def record(self):
        return {"slope": exact(self.slope), "intercept": exact(self.intercept)}


IDENTITY = Affine(F(1), F(0))
RELAXATION = F(1, 4)
LAMBDA = F(1, 2)
RHO = F(1, 2)
STEP_SIZE = F(1)
INERTIA_CAP = F(1)
SHIFTS = (F(-2), F(0), F(1), F(3, 2))
HORIZONS = (64, 128)
POLICIES = ("raw_origin_damping", "affine_combination_candidate")


def blend(left, right, right_weight):
    return Affine((1 - right_weight) * left.slope + right_weight * right.slope,
                  (1 - right_weight) * left.intercept + right_weight * right.intercept)


def resolvent(B):
    require(B.slope >= 0, "The scalar monotone operator must have nonnegative slope")
    denominator = 1 + STEP_SIZE * B.slope
    return Affine(1 / denominator, -STEP_SIZE * B.intercept / denominator)


@dataclass(frozen=True)
class Problem:
    name: str
    Q: Affine
    B: Affine
    g: Affine
    initial: F
    target: F
    b: F = RELAXATION
    lam: F = LAMBDA
    rho: F = RHO

    def translated(self, shift):
        return Problem(self.name, self.Q.conjugate(shift), self.B.shift_input(shift),
                       self.g.conjugate(shift), self.initial - shift, self.target - shift,
                       self.b, self.lam, self.rho)

    def operators(self):
        R = resolvent(self.B)
        # Two identical Q_j and strictly positive weights 1/2,1/2 give this Z.
        Z = blend(IDENTITY, self.Q, self.b)
        M = blend(IDENTITY, Z, self.lam)
        U = Z.compose(M).compose(R)
        return {"Q": self.Q, "R": R, "Z": Z, "M": M, "U": U, "g": self.g}

    def record(self):
        return {"name": self.name, "B": self.B.record(),
                "operators": {k: v.record() for k, v in self.operators().items()},
                "x0_equals_x1": exact(self.initial), "projected_target": exact(self.target),
                "b": exact(self.b), "lambda": exact(self.lam), "rho": exact(self.rho)}


def parameters(n):
    require(type(n) is int and n >= 1, "Positive integer step index required")
    delta = F(1, n + 1)
    return delta, delta * delta


def inertia(previous, current, n):
    _, tau = parameters(n)
    difference = current - previous
    coefficient = min(INERTIA_CAP, tau / abs(difference)) if difference else F(0)
    error = coefficient * difference
    require(abs(error) <= tau, "Inertial displacement exceeded tau")
    return current + error


def step(problem, previous, current, n, policy):
    delta, _ = parameters(n)
    alpha = inertia(previous, current, n)
    Ualpha = problem.operators()["U"](alpha)
    if policy == "raw_origin_damping":
        ell = problem.rho * Ualpha
    elif policy == "affine_combination_candidate":
        ell = (1 - problem.rho) * alpha + problem.rho * Ualpha
    else:
        raise ValueError("Unknown policy")
    return delta * problem.g(current) + (1 - delta) * ell


def trajectory(problem, policy, updates):
    require(updates in HORIZONS, "This checker supports exactly 64 or 128 updates")
    xs = [problem.initial, problem.initial]
    for n in range(1, updates + 1):
        xs.append(step(problem, xs[-2], xs[-1], n, policy))
    return xs


def check_operator_conjugacy(problem, translated, shift):
    original_ops, translated_ops = problem.operators(), translated.operators()
    probes = (F(-3), F(-1, 2), F(0), F(1), F(3, 2), F(4))
    for name, operator in original_ops.items():
        require(operator.conjugate(shift) == translated_ops[name],
                "Operator conjugacy mismatch: " + name)
        for x in probes:
            require(translated_ops[name](x - shift) + shift == operator(x),
                    "Point-map conjugacy failed: " + name)
    for x in probes:
        require(translated.B(x - shift) == problem.B(x), "Monotone operator input shift failed")
    return len(original_ops) * len(probes) + len(probes)


def run():
    problems = (
        Problem("identity_zero_operator", IDENTITY, Affine(F(0), F(0)),
                Affine(F(0), F(1)), F(0), F(1)),
        Problem("nonidentity_common_fixed_point", Affine(F(1, 2), F(1)),
                Affine(F(1), F(-2)), Affine(F(0), F(1)), F(0), F(2)),
    )
    summaries, retained, plot_series = [], [], []
    totals = {"operator_point_checks": 0, "corresponding_state_one_step_checks": 0,
              "candidate_exact_trajectory_comparisons": 0, "prefix_checks": 0}
    for problem in problems:
        common = problem.target
        for name in ("Q", "R", "Z", "M", "U"):
            require(problem.operators()[name](common) == common, "Common point not fixed: " + name)
        for policy in POLICIES:
            original_long = trajectory(problem, policy, 128)
            original_short = trajectory(problem, policy, 64)
            require(original_long[:66] == original_short, "Original 64/128 prefix mismatch")
            totals["prefix_checks"] += 1
            for shift in SHIFTS:
                translated = problem.translated(shift)
                totals["operator_point_checks"] += check_operator_conjugacy(problem, translated, shift)
                shifted_long = trajectory(translated, policy, 128)
                shifted_short = trajectory(translated, policy, 64)
                require(shifted_long[:66] == shifted_short, "Translated 64/128 prefix mismatch")
                totals["prefix_checks"] += 1
                one_step_defects = []
                # These input pairs correspond exactly, even after the separately
                # evolved translated trajectory has diverged from the original.
                for n in range(1, 129):
                    previous, current = original_long[n - 1], original_long[n]
                    base = step(problem, previous, current, n, policy)
                    shifted_once = step(translated, previous - shift, current - shift, n, policy)
                    actual = shifted_once + shift - base
                    delta, _ = parameters(n)
                    expected = ((1 - delta) * (1 - problem.rho) * shift
                                if policy == "raw_origin_damping" else F(0))
                    require(actual == expected, "Corresponding-state one-step defect mismatch")
                    require(inertia(previous - shift, current - shift, n) + shift
                            == inertia(previous, current, n), "Inertia is not equivariant")
                    one_step_defects.append(actual)
                    totals["corresponding_state_one_step_checks"] += 1
                pulled_back = [y + shift for y in shifted_long]
                differences = [y - x for x, y in zip(original_long, pulled_back)]
                if policy == "affine_combination_candidate":
                    require(all(d == 0 for d in differences), "Candidate trajectory not equivariant")
                    totals["candidate_exact_trajectory_comparisons"] += len(differences)
                elif shift == 0:
                    require(all(d == 0 for d in differences), "Zero-shift control failed")
                else:
                    require(differences[2] == (1 - F(1, 2)) * (1 - problem.rho) * shift,
                            "First independent step did not exhibit the predicted defect")
                    require(any(differences[n + 1] != one_step_defects[n - 1] for n in range(2, 129)),
                            "Diverged trajectories unexpectedly all matched one-step defects")
                for horizon in HORIZONS:
                    end = horizon + 1
                    summaries.append({"case": problem.name, "policy": policy, "shift": exact(shift),
                        "updates": horizon, "last_state_index": end,
                        "original_endpoint": value(original_long[end]),
                        "translated_endpoint_pulled_back": value(pulled_back[end]),
                        "independent_trajectory_difference": value(differences[end]),
                        "corresponding_state_one_step_defect_at_same_n": value(one_step_defects[horizon - 1]),
                        "all_trajectory_states_equivariant": all(d == 0 for d in differences[:end + 1]),
                        "one_step_formula_applied_to_diverged_trajectory": False})
                exact_run = {"case": problem.name, "policy": policy, "shift": exact(shift),
                    "problem": problem.record(), "translated_problem": translated.record(),
                    "states_x0_through_x129": [exact(x) for x in original_long],
                    "translated_states_y0_through_y129": [exact(x) for x in shifted_long],
                    "corresponding_state_one_step_defects_n1_through_n128": [exact(x) for x in one_step_defects]}
                exact_bytes = json.dumps(exact_run, sort_keys=True, separators=(",", ":")).encode()
                retained.append({"case": problem.name, "policy": policy, "shift": exact(shift),
                    "exact_run_sha256": hashlib.sha256(exact_bytes).hexdigest(),
                    "hashed_preimage_schema": "exact run object defined in this source: original/translated problem and all exact rational states/defects",
                    "exact_state_count_per_trajectory": len(original_long),
                    "first_corresponding_state_defect": exact(one_step_defects[0]),
                    "last_corresponding_state_defect": exact(one_step_defects[-1])})
                if problem.name == "identity_zero_operator":
                    plot_series.append({"policy": policy, "shift": exact(shift),
                        "physical_pulled_back_positions": [
                            {"n": n, "value_plot_only": approximate(x)} for n, x in enumerate(pulled_back)]})
    adversarial = Problem("candidate_general_class_countercheck", Affine(F(-3), F(0)),
        Affine(F(0), F(0)), Affine(F(0), F(1)), F(0), F(0), F(3, 4), F(1, 2), F(9, 10))
    aops = adversarial.operators()
    require(aops["Z"] == Affine(F(-2), F(0)) and aops["U"] == IDENTITY,
            "Adversarial affine composition mismatch")
    require(adversarial.b < adversarial.rho < 1, "Adversarial allowed parameter range failed")
    # For Q=-3I and phi=4, phi*x*(x-Qx) and (x-Qx)^2 both have x^2 coefficient16.
    require(F(4) * (1 - adversarial.Q.slope) == (1 - adversarial.Q.slope) ** 2,
            "Generalized-demimetric equality coefficients differ")
    ax = trajectory(adversarial, "affine_combination_candidate", 128)
    ix = trajectory(problems[0], "affine_combination_candidate", 128)
    require(ax == ix, "Adversarial candidate does not equal identity-candidate trajectory")
    optional = Problem("static_expansive_composition", Affine(F(-7), F(0)),
        Affine(F(0), F(0)), Affine(F(0), F(1)), F(0), F(0), F(1, 2), F(1, 2), F(9, 10))
    require(optional.operators()["U"] == Affine(F(3), F(0)), "Static U=3I check failed")
    require(F(8) * (1 - optional.Q.slope) == (1 - optional.Q.slope) ** 2,
            "Static generalized-demimetric equality coefficients differ")
    adversarial_result = {"problem": adversarial.record(), "phi": "4/1", "b_times_phi": "3/1",
        "checks": "Z=-2I; U=I; ell=alpha; exact generalized-demimetric equality; b<rho<1",
        "candidate_matches_identity_candidate_all_130_states": True,
        "exact_candidate_trajectory_sha256": hashlib.sha256(
            json.dumps([exact(x) for x in ax], separators=(",", ":")).encode()).hexdigest(),
        "endpoints": [{"updates": n, "value": value(ax[n + 1]), "target": "0/1"} for n in HORIZONS],
        "interpretation": "Symmetry does not establish general convergence. Finite sequence equals identity candidate; any asymptotic conclusion requires the separate analytic argument.",
        "optional_static_only": {"problem": optional.record(), "phi": "8/1", "U_slope": "3/1",
            "candidate_ell_slope": "14/5", "trajectory_not_run": True,
            "no_finite_check_is_claimed_to_prove_divergence": True}}
    result = {"schema": "zerone-exact-translation-check/v1", "status": "PASS",
        "source_sha256": hashlib.sha256(Path(__file__).read_bytes()).hexdigest(),
        "arithmetic": "fractions.Fraction throughout all checks; decimal strings are presentation only",
        "parameters": {"b": exact(RELAXATION), "lambda": exact(LAMBDA), "rho": exact(RHO),
            "resolvent_step": exact(STEP_SIZE), "inertia_cap": exact(INERTIA_CAP),
            "delta_n": "1/(n+1)", "tau_n": "delta_n^2", "two_Q_weights": ["1/2", "1/2"]},
        "indexing": "x0=x1=initial; update n computes x[n+1]; 64/128 updates end at x65/x129",
        "coordinate_rule": "y=x-t; Q_t(y)=Q(y+t)-t, g_t(y)=g(y+t)-t, B_t(y)=B(y+t)",
        "raw_corresponding_input_defect": "pulled-back minus original = (1-delta_n)*(1-rho)*t",
        "candidate_ell": "(1-rho)*alpha + rho*U(alpha), U=Z composed with ((1-lambda)I+lambda Z) composed with R",
        "cases": [p.record() for p in problems], "check_counts": totals, "comparisons": summaries,
        "exact_run_digests": retained, "adversarial_candidate": adversarial_result,
        "aggregate_exact_run_digest": hashlib.sha256(
            json.dumps(retained, sort_keys=True, separators=(",", ":")).encode()).hexdigest(),
        "limits": ["Finite exact 64/128-update checks, not a limit proof or convergence theorem",
            "The affine-combination candidate is tested for translation equivariance only",
            "Eq9 reconstruction and corrected normalized weights are explicit; printed sum-mu and Eq2 ambiguity are unresolved",
            "Full independently evolved raw trajectories are not governed by the corresponding-input one-step defect formula",
            "Standalone local checker, no article code, network, repository edits, external reviewers or publication"]}
    return result, {"schema": "zerone-translation-plot-data/v1", "plot_values_only": True,
        "case": "identity_zero_operator", "a": "1/1", "physical_target": "1/1",
        "raw_or_candidate_mathematical_validation_uses_only_Fraction": True,
        "description": "Each value is y_n+t in original physical coordinates; decimal strings are for plotting only.",
        "series": plot_series}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--output-dir", required=True, type=Path, help="Fresh nonexistent directory under an existing parent")
    args = parser.parse_args()
    require(not args.output_dir.exists() and not args.output_dir.is_symlink(), "Output directory already exists")
    result, plot_data = run()
    args.output_dir.mkdir(mode=0o700)
    hashes = []
    for name, obj in (("results.json", result), ("plot-data.json", plot_data)):
        raw = (json.dumps(obj, sort_keys=True, indent=2, ensure_ascii=True, allow_nan=False) + "\n").encode()
        path = args.output_dir / name
        with path.open("xb") as stream:
            os.chmod(path, 0o600)
            stream.write(raw)
            stream.flush()
            os.fsync(stream.fileno())
        hashes.append(hashlib.sha256(raw).hexdigest() + "  " + name)
    (args.output_dir / "SHA256SUMS").write_text("\n".join(hashes) + "\n")
    os.chmod(args.output_dir / "SHA256SUMS", 0o600)
    print(json.dumps({"status": "PASS", "checks": result["check_counts"], "comparisons": len(result["comparisons"])}))


if __name__ == "__main__":
    try:
        main()
    except (ValueError, OSError) as exc:
        print("Translation check refused: " + str(exc), file=sys.stderr)
        raise SystemExit(2)
