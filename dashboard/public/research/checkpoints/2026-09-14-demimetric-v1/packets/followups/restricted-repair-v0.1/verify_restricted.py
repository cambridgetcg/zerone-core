#!/usr/bin/env python3
"""Exact, bounded corroboration of seven declared scalar repair fixtures.

The Fraction, affine conjugation and capped-inertia conventions follow the
initiative's translation-v0.1/verify_translation.py. This self-contained checker
adds finite-piece algebra, residual estimates and restricted-class controls.
No paper code, dependencies, network, repository writes or limit certification.
"""
from __future__ import annotations
import argparse
from bisect import bisect_left
from dataclasses import dataclass
from decimal import Decimal, localcontext
from fractions import Fraction as F
import hashlib
import json
import os
from pathlib import Path
import platform
import sys

if not __debug__:
    print("Refused: run without -O/-OO.", file=sys.stderr)
    raise SystemExit(2)


def require(condition, message):
    if not condition:
        raise ValueError(message)


def exact(value):
    return f"{value.numerator}/{value.denominator}"


def canonical(value):
    return json.dumps(value, sort_keys=True, separators=(",", ":"), allow_nan=False).encode()


def digest(value):
    return hashlib.sha256(canonical(value)).hexdigest()


def compact(value):
    text = exact(value)
    with localcontext() as context:
        context.prec = 16
        decimal = str(Decimal(value.numerator) / Decimal(value.denominator))
    return {"exact_fraction": text if len(text) <= 180 else None,
            "exact_fraction_sha256": hashlib.sha256(text.encode()).hexdigest(),
            "numerator_bits": value.numerator.bit_length(),
            "denominator_bits": value.denominator.bit_length(),
            "decimal_for_display_only": decimal}


@dataclass(frozen=True)
class Affine:
    a: F
    c: F

    def __call__(self, x):
        return self.a*x + self.c

    def compose(self, other):
        return Affine(self.a*other.a, self.a*other.c+self.c)


@dataclass(frozen=True)
class Piecewise:
    """Only finite continuous affine pieces used by the predeclared fixtures."""
    cuts: tuple
    pieces: tuple

    def __post_init__(self):
        require(len(self.pieces) == len(self.cuts)+1, "Piece count mismatch")
        require(tuple(sorted(set(self.cuts))) == self.cuts, "Cuts must strictly increase")
        require(len(self.cuts) <= 64, "Fixture algebra exceeded its declared piece bound")
        for i, x in enumerate(self.cuts):
            require(self.pieces[i](x) == self.pieces[i+1](x), "Fixture map is discontinuous")

    def at(self, x):
        return self.pieces[bisect_left(self.cuts, x)]

    def __call__(self, x):
        return self.at(x)(x)

    def domains(self):
        bounds = (None,)+self.cuts+(None,)
        return tuple((bounds[i], bounds[i+1]) for i in range(len(self.pieces)))

    def conjugate(self, t):
        return Piecewise(tuple(c-t for c in self.cuts),
                         tuple(Affine(p.a, p.c+(p.a-1)*t) for p in self.pieces))

    def record(self):
        return {"cuts": [exact(v) for v in self.cuts],
                "pieces": [{"slope": exact(p.a), "intercept": exact(p.c)} for p in self.pieces]}


def affine(a, c=F(0)):
    return Piecewise((), (Affine(F(a), F(c)),))


I = affine(F(1))


def inside(x, low, high):
    return (low is None or x >= low) and (high is None or x <= high)


def middle(low, high):
    return (low+high)/2 if low is not None and high is not None else (
        high-1 if high is not None else low+1 if low is not None else F(0))


def build(cuts, coefficients):
    cuts = tuple(sorted(set(cuts)))
    bounds = (None,)+cuts+(None,)
    new_cuts, pieces = [], []
    for low, high in zip(bounds, bounds[1:]):
        p = coefficients(middle(low, high))
        if pieces and pieces[-1] == p:
            continue
        if pieces:
            new_cuts.append(low)
        pieces.append(p)
    return Piecewise(tuple(new_cuts), tuple(pieces))


def average(maps, weights):
    require(len(maps) == len(weights), "Weight count mismatch")
    def coefficients(x):
        parts = [m.at(x) for m in maps]
        return Affine(sum((w*p.a for w, p in zip(weights, parts)), F(0)),
                      sum((w*p.c for w, p in zip(weights, parts)), F(0)))
    return build([c for m in maps for c in m.cuts], coefficients)


def compose(outer, inner):
    cuts = list(inner.cuts)
    for part, (low, high) in zip(inner.pieces, inner.domains()):
        if part.a:
            for boundary in outer.cuts:
                root = (boundary-part.c)/part.a
                if inside(root, low, high):
                    cuts.append(root)
    return build(cuts, lambda x: outer.at(inner(x)).compose(inner.at(x)))


def normalized_intervals(intervals):
    result = []
    for low, high in sorted(intervals, key=lambda item: (item[0] is not None, item[0] or F(0))):
        if result and (result[-1][1] is None or low is None or low <= result[-1][1]):
            old_low, old_high = result[-1]
            result[-1] = (old_low, None if old_high is None or high is None else max(old_high, high))
        else:
            result.append((low, high))
    return tuple(result)


def fixed_set(mapping):
    found = []
    for part, (low, high) in zip(mapping.pieces, mapping.domains()):
        if part.a == 1:
            if part.c == 0:
                found.append((low, high))
        else:
            root = part.c/(1-part.a)
            if inside(root, low, high):
                found.append((root, root))
    return normalized_intervals(found)


def intersection(left, right):
    found = []
    for a, b in left:
        for c, d in right:
            low = c if a is None else a if c is None else max(a, c)
            high = d if b is None else b if d is None else min(b, d)
            if low is None or high is None or low <= high:
                found.append((low, high))
    return normalized_intervals(found)


def sets_record(intervals):
    return [{"lower": None if lo is None else exact(lo),
             "upper": None if hi is None else exact(hi)} for lo, hi in intervals]


def quadratic_nonnegative(A, B, C, low, high):
    if A < 0 and (low is None or high is None):
        return False
    if A == 0 and ((low is None and B > 0) or (high is None and B < 0)):
        return False
    probes = [x for x in (low, high) if x is not None]
    if A > 0 and inside(-B/(2*A), low, high):
        probes.append(-B/(2*A))
    if not probes:
        probes.append(F(0))
    return all(A*x*x+B*x+C >= 0 for x in probes)


def demimetric_certificates(mapping, phi, p):
    certificates = []
    for part, (low, high) in zip(mapping.pieces, mapping.domains()):
        a, c = 1-part.a, -part.c
        A, B, C = phi*a-a*a, phi*(c-p*a)-2*a*c, -phi*p*c-c*c
        require(quadratic_nonnegative(A, B, C, low, high), "Demimetric interval certificate failed")
        certificates.append({"domain": sets_record(((low, high),))[0],
                             "gap_polynomial_coefficients": [exact(v) for v in (A, B, C)]})
    return certificates


@dataclass(frozen=True)
class Problem:
    name: str
    Q: tuple
    phi: tuple
    weights: tuple
    B: tuple
    eta: tuple
    g: Piecewise
    target: F
    b: F = F(1, 4)
    lam: F = F(1, 2)
    rho: F = F(3, 4)
    initial: F = F(0)
    admitted: bool = True

    def restrictions(self):
        reasons = []
        if not self.b > 0 or any(p <= 0 or self.b*p > 2 for p in self.phi):
            reasons.append("b>0, phi>0, b*phi<=2 required")
        if not 0 < self.lam < 1:
            reasons.append("lambda must be interior")
        if not 0 < self.rho < 1:
            reasons.append("rho must be interior")
        if any(w <= 0 for w in self.weights) or sum(self.weights) != 1:
            reasons.append("fixed positive normalized weights required")
        if len(self.g.pieces) != 1 or abs(self.g.pieces[0].a) >= 1:
            reasons.append("affine g must be a contraction")
        return reasons

    def operators(self):
        R = []
        for B, eta in zip(self.B, self.eta):
            require(B.a >= 0 and eta > 0, "Nonmonotone B or invalid resolvent step")
            R.append(affine(1/(1+eta*B.a), -eta*B.c/(1+eta*B.a)))
        V = tuple(average((I, q), (1-self.b, self.b)) for q in self.Q)
        Z = average(V, self.weights)
        M = average((I, Z), (1-self.lam, self.lam))
        composite_R = I
        for r in R:
            composite_R = compose(r, composite_R)
        U = compose(Z, compose(M, composite_R))
        L = average((I, U), (1-self.rho, self.rho))
        return {"R": tuple(R), "V": V, "Z": Z, "M": M, "U": U, "L": L}

    def shifted(self, t):
        return Problem(self.name, tuple(q.conjugate(t) for q in self.Q), self.phi,
                       self.weights, tuple(Affine(B.a, B.c+B.a*t) for B in self.B),
                       self.eta, self.g.conjugate(t), self.target-t, self.b,
                       self.lam, self.rho, self.initial-t, self.admitted)


SHIFTS = (F(-2), F(0), F(1), F(3, 2))
GRID = tuple(F(k, 4) for k in range(-16, 25))
UPDATES = 32


def problems():
    weights = (F(1, 2), F(1, 2))
    zeros = (Affine(F(0), F(0)), Affine(F(0), F(0)))
    etas = (F(1), F(1))
    constant = affine(F(0), F(1))
    piece = Piecewise((F(1), F(3, 2)), (Affine(F(0), F(0)), Affine(F(2), F(-2)), Affine(F(0), F(1))))
    def negative(name, q, phi, b, lam=F(1, 2), admitted=True, rho=F(3, 4)):
        return Problem(name, (affine(q), affine(q)), (phi, phi), weights, zeros,
                       etas, constant, F(0), b, lam, rho, admitted=admitted)
    return (
        Problem("identity_target1", (I, I), (F(1), F(1)), weights, zeros, etas, constant, F(1)),
        Problem("affine_target2", (affine(F(1, 2), F(1)), affine(F(-1), F(4))),
                (F(1), F(2)), (F(1, 3), F(2, 3)),
                (Affine(F(1), F(-2)), Affine(F(2), F(-4))),
                (F(1), F(1, 2)), affine(F(1, 4), F(1)), F(2)),
        Problem("piecewise_target2", (piece.conjugate(F(-2)), I), (F(1), F(1)),
                (F(2, 3), F(1, 3)), zeros, etas, affine(F(1, 4), F(1)), F(2)),
        negative("strict_minus3", F(-3), F(4), F(1, 4)),
        negative("admitted_saturation", F(-1), F(2), F(1)),
        negative("excluded_saturation_lambda1", F(-1), F(2), F(1), F(1), False),
        negative("excluded_old_minus3", F(-3), F(4), F(3, 4), admitted=False, rho=F(9, 10)),
    )


def residual_estimate(problem, ops, a):
    p = problem.target
    ys = [a]
    for R in ops["R"]:
        ys.append(R(ys[-1]))
    beta = ys[-1]; gamma = ops["M"](beta); U = ops["U"](a); L = ops["L"](a)
    require(ops["Z"](gamma) == U, "Composite evaluation mismatch")
    def D(x):
        return sum((w*problem.b*(2/phi-problem.b)*(x-q(x))**2
                    for w, phi, q in zip(problem.weights, problem.phi, problem.Q)), F(0))
    def J(x):
        return sum((w*(v(x)-ops["Z"](x))**2 for w, v in zip(problem.weights, ops["V"])), F(0))
    S = sum(((x-y)**2 for x, y in zip(ys, ys[1:])), F(0))
    terms = {"resolvent_prefix_loss": S,
             "lambda_D_beta": problem.lam*D(beta),
             "lambda_J_beta": problem.lam*J(beta),
             "mixing_loss": problem.lam*(1-problem.lam)*(beta-ops["Z"](beta))**2,
             "D_gamma": D(gamma), "J_gamma": J(gamma)}
    joint = sum(terms.values(), F(0))
    loss_U = (a-p)**2-(U-p)**2
    bound_L = problem.rho*joint+problem.rho*(1-problem.rho)*(a-U)**2
    loss_L = (a-p)**2-(L-p)**2
    require(loss_U >= joint and loss_L >= bound_L, "Joint bound failed")
    if problem.admitted:
        require(all(x >= 0 for x in terms.values()), "Admitted lower bound has negative term")
    weighted = sum((w*problem.b/phi*(beta-q(beta))**2
                    for w, phi, q in zip(problem.weights, problem.phi, problem.Q)), F(0))
    require(0 <= weighted <= (beta-p)*(beta-ops["Z"](beta)), "Inner-product residual bound failed")
    for x in (a, beta, gamma):
        for q, phi in zip(problem.Q, problem.phi):
            r = x-q(x)
            require(phi*(x-p)*r >= r*r, "Demimetric probe failed")
    for R, x, y in zip(ops["R"], ys, ys[1:]):
        require((y-p)**2+(x-y)**2 <= (x-p)**2, "Firm resolvent bound failed")
    return {**terms, "joint": joint, "loss_U": loss_U, "slack_U": loss_U-joint,
            "bound_L": bound_L, "loss_L": loss_L, "slack_L": loss_L-bound_L,
            "weighted_original_residual": weighted}


def trajectory(problem, ops):
    states = [problem.initial, problem.initial]
    for n in range(1, UPDATES+1):
        delta = F(1, n+1); tau = delta*delta
        diff = states[-1]-states[-2]
        e = min(F(1), tau/abs(diff))*diff if diff else F(0)
        require(abs(e) <= tau, "Inertial bound failed")
        a = states[-1]+e
        states.append(delta*problem.g(states[-1])+(1-delta)*ops["L"](a))
    return states


def run():
    fixture_results, trajectory_records = [], []
    counts = {"fixtures": 7, "admitted": 5, "expected_restriction_refusals": 2,
              "grid_joint_estimates": 0, "coordinate_trajectory_comparisons": 0,
              "operator_conjugacy_comparisons": 0, "all_domain_Q_segments": 0,
              "fixed_set_comparisons": 0}
    identity_states = None
    for problem in problems():
        issues = problem.restrictions()
        require((not issues) == problem.admitted, "Unexpected restriction result")
        ops = problem.operators()
        theta = ((None, None),)
        for mapping in problem.Q+ops["R"]:
            theta = intersection(theta, fixed_set(mapping))
            require(mapping(problem.target) == problem.target, "Target not a common fixed point")
        expected = ((None, None),) if problem.name == "identity_target1" else ((problem.target, problem.target),)
        require(theta == expected, "Original fixed set mismatch")
        require(theta == ((problem.target, problem.target),) or problem.g(problem.target) == problem.target,
                "Selected target does not satisfy projection equation")
        certificates = []
        for q, phi in zip(problem.Q, problem.phi):
            cert = demimetric_certificates(q, phi, problem.target)
            certificates.append(cert); counts["all_domain_Q_segments"] += len(cert)
        for R in ops["R"]:
            require(len(R.pieces) == 1 and 0 <= R.pieces[0].a <= 1, "Affine resolvent not firm")
        fix_U, fix_L = fixed_set(ops["U"]), fixed_set(ops["L"])
        require((fix_U == theta) == problem.admitted, "Unexpected composite fixed-set result")
        require(fix_U == fix_L, "Averaging changed fixed set")
        base_states = trajectory(problem, ops)
        if problem.name == "identity_target1":
            identity_states = base_states
        if not problem.admitted:
            require(ops["U"] == I and ops["L"] == I, "Countercontrol is not identity")
            require(base_states == identity_states, "Countercontrol does not equal identity trajectory")
        cases = []
        for shift in SHIFTS:
            shifted = problem.shifted(shift); shifted_ops = shifted.operators()
            for key in ("R", "V"):
                for original, transformed in zip(ops[key], shifted_ops[key]):
                    require(original.conjugate(shift) == transformed, "Conjugacy mismatch "+key)
                    counts["operator_conjugacy_comparisons"] += 1
            for key in ("Z", "M", "U", "L"):
                require(ops[key].conjugate(shift) == shifted_ops[key], "Conjugacy mismatch "+key)
                counts["operator_conjugacy_comparisons"] += 1
            for original, transformed in zip(problem.Q, shifted.Q):
                require(original.conjugate(shift) == transformed, "Q conjugacy mismatch")
                counts["operator_conjugacy_comparisons"] += 1
            require(problem.g.conjugate(shift) == shifted.g, "g conjugacy mismatch")
            counts["operator_conjugacy_comparisons"] += 1
            shifted_theta = tuple((None if a is None else a-shift, None if b is None else b-shift) for a, b in theta)
            require((fixed_set(shifted_ops["U"]) == shifted_theta) == problem.admitted,
                    "Translated fixed-set result mismatch")
            counts["fixed_set_comparisons"] += 1
            grid_values = []
            for physical in GRID:
                original = residual_estimate(problem, ops, physical)
                transformed = residual_estimate(shifted, shifted_ops, physical-shift)
                require(original == transformed, "Residual estimate changed with coordinates")
                grid_values.append(original); counts["grid_joint_estimates"] += 1
                for B, Bt in zip(problem.B, shifted.B):
                    require(B(physical) == Bt(physical-shift), "Displacement-valued B translation failed")
            shifted_states = trajectory(shifted, shifted_ops)
            pulled = [value+shift for value in shifted_states]
            require(pulled == base_states, "Full physical trajectory mismatch")
            counts["coordinate_trajectory_comparisons"] += len(base_states)
            trajectory_records.append({"case": problem.name, "shift": exact(shift),
                "state_count": len(base_states), "physical_states_sha256": digest([exact(v) for v in pulled]),
                "coordinate_states_sha256": digest([exact(v) for v in shifted_states]),
                "endpoint_physical": compact(pulled[-1]), "target_physical": exact(problem.target)})
            cases.append({"shift": exact(shift), "minimum_joint": exact(min(v["joint"] for v in grid_values)),
                "minimum_U_slack": exact(min(v["slack_U"] for v in grid_values)),
                "minimum_L_slack": exact(min(v["slack_L"] for v in grid_values)),
                "nonnegative_terms_required": problem.admitted})
        sample = residual_estimate(problem, ops, problem.target+F(1, 2))
        extra = None
        if problem.name == "piecewise_target2":
            x, y = problem.target+1, problem.target+F(3, 2)
            require(abs(ops["Z"](y)-ops["Z"](x)) > abs(y-x), "Expected Z nonexpansiveness control failed")
            extra = {"x": exact(x), "y": exact(y), "input_distance": exact(abs(y-x)),
                     "Z_distance": exact(abs(ops["Z"](y)-ops["Z"](x))),
                     "meaning": "Admitted strong quasi-nonexpansive construction need not be pairwise nonexpansive"}
        fixture_results.append({"name": problem.name, "admitted": problem.admitted,
            "expected_refusal_reasons": issues, "b": exact(problem.b), "phi": [exact(p) for p in problem.phi],
            "b_phi": [exact(problem.b*p) for p in problem.phi], "lambda": exact(problem.lam),
            "rho": exact(problem.rho), "weights": [exact(w) for w in problem.weights],
            "Q": [q.record() for q in problem.Q], "B": [{"slope": exact(B.a), "intercept": exact(B.c)} for B in problem.B],
            "eta": [exact(e) for e in problem.eta], "g": problem.g.record(),
            "Z": ops["Z"].record(), "U": ops["U"].record(), "L": ops["L"].record(),
            "original_common_fixed_set": sets_record(theta), "Fix_U": sets_record(fix_U), "Fix_L": sets_record(fix_L),
            "demimetric_interval_certificates": certificates,
            "sample_a": exact(problem.target+F(1, 2)), "sample_joint_terms": {k: exact(v) for k, v in sample.items()},
            "grid_summaries": cases, "pairwise_nonexpansiveness_countercontrol": extra})
    require(counts["grid_joint_estimates"] == 1148, "Grid count drift")
    require(counts["coordinate_trajectory_comparisons"] == 952, "Trajectory count drift")
    return {"schema": "restricted-repair-exact-check/v1", "status": "PASS", "counts": counts,
            "grid": [exact(v) for v in GRID], "shifts": [exact(v) for v in SHIFTS], "updates": UPDATES,
            "indexing": "x0=x1=initial; n1..32 compute x2..x33; 34 retained states per trajectory",
            "fixtures": fixture_results, "trajectories": trajectory_records,
            "trajectory_records_sha256": digest(trajectory_records),
            "limits": ["Seven declared one-dimensional maps; exact algebra for their finite affine pieces",
                       "Finite trajectories do not prove convergence, a rate or a general theorem",
                       "Admitted saturation plus interior mixing is distinct from the excluded combined endpoint",
                       "Same-controller assistant corroboration; no external independent reviewer",
                       "No unqualified claim about the source paper's literal notation"]}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--output-dir", type=Path, required=True)
    args = parser.parse_args()
    require(not args.output_dir.exists() and not args.output_dir.is_symlink(), "Output directory already exists")
    require(args.output_dir.parent.is_dir(), "Output parent must already exist")
    protocol = Path(__file__).with_name("PROTOCOL.md").read_bytes()
    result = run()
    raw = canonical(result)
    source_sha = hashlib.sha256(Path(__file__).read_bytes()).hexdigest()
    receipt = {"status": "PASS", "python": sys.version, "platform": platform.platform(),
               "arithmetic": "stdlib fractions.Fraction; decimal values are presentation only",
               "source_sha256": source_sha, "protocol_sha256": hashlib.sha256(protocol).hexdigest(),
               "results_content_sha256": hashlib.sha256(raw).hexdigest(), "counts": result["counts"]}
    args.output_dir.mkdir(mode=0o700)
    hashes = []
    for name, value in (("results.json", result), ("receipt.json", receipt)):
        data = (json.dumps(value, indent=2, sort_keys=True, allow_nan=False)+"\n").encode()
        with (args.output_dir/name).open("xb") as stream:
            os.chmod(args.output_dir/name, 0o600); stream.write(data); stream.flush(); os.fsync(stream.fileno())
        hashes.append(hashlib.sha256(data).hexdigest()+"  "+name)
    with (args.output_dir/"SHA256SUMS").open("x") as stream:
        stream.write("\n".join(hashes)+"\n")
    print(json.dumps(receipt, sort_keys=True))


if __name__ == "__main__":
    try:
        main()
    except (ValueError, OSError) as error:
        print("Restricted check refused: "+str(error), file=sys.stderr)
        raise SystemExit(2)
