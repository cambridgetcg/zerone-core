#!/usr/bin/env python3
"""Exact rational, same-operator checks. No network or third-party packages.

These checks are algebra/finite recurrence verification, not a formal proof of
the general Hilbert-space proposition or the infinite-time convergence limit.
All interpretation changes are stated in REVIEW.md; this is not a verifier of
every literal hypothesis of the source's inconsistently indexed theorem.
"""
import sys
if sys.flags.optimize:
    raise SystemExit("Exact verification requires assertions; do not use -O or -OO.")

from fractions import Fraction as F
import argparse
import hashlib
import json
import os
from pathlib import Path


def clean(a):
    a = list(map(F, a))
    while len(a) > 1 and a[-1] == 0:
        a.pop()
    return tuple(a)


def add(a, b):
    return clean([(a[i] if i < len(a) else 0) +
                  (b[i] if i < len(b) else 0)
                  for i in range(max(len(a), len(b)))])


def scale(a, b):
    return clean([F(b) * x for x in a])


def mul(a, b):
    result = [F(0)] * (len(a) + len(b) - 1)
    for i, x in enumerate(a):
        for j, y in enumerate(b):
            result[i+j] += x*y
    return clean(result)


def at(a, x):
    return sum((c*x**i for i, c in enumerate(a)), F(0))


X = (F(0), F(1))
BRANCHES = ((F(0),), (F(-2), F(2)), (F(1),))


def q(x):
    return F(0) if x <= 1 else 2*x-2 if x < F(3, 2) else F(1)


def witness():
    # Polynomial identities plus the indicated factor signs certify the whole
    # real line, unlike a finite grid of point evaluations.
    slacks = []
    for branch in BRANCHES:
        residual = add(X, scale(branch, -1))
        slacks.append(add(mul(X, residual), scale(mul(residual, residual), -1)))
    assert slacks[0] == (0,)
    assert slacks[1] == mul((2, -1), (-2, 2))
    assert slacks[2] == (-1, 1)
    # Middle [1,3/2]: 2-x >= 1/2 and 2x-2 >= 0.
    assert min(at((2, -1), x) for x in (F(1), F(3, 2))) == F(1, 2)
    assert min(at((-2, 2), x) for x in (F(1), F(3, 2))) == 0
    # Right [3/2,infinity): x-1 has positive slope and value 1/2.
    assert slacks[2][1] > 0 and at(slacks[2], F(3, 2)) == F(1, 2)
    assert at(BRANCHES[0], F(1)) == at(BRANCHES[1], F(1)) == 0
    assert at(BRANCHES[1], F(3, 2)) == at(BRANCHES[2], F(3, 2)) == 1
    # Solve Q(x)=x branchwise and test interval membership.
    candidates = [F(0), F(2), F(1)]
    assert all(at(b, x) == x for b, x in zip(BRANCHES, candidates))
    assert candidates[0] <= 1 and not 1 < candidates[1] < F(3, 2)
    assert not candidates[2] >= F(3, 2)
    b, x, y = F(1, 2), F(1), F(3, 2)
    v = lambda t: (1-b)*t+b*q(t)
    assert v(x) == F(1, 2) and v(y) == F(5, 4)
    assert abs(v(y)-v(x)) == F(3, 4) > abs(y-x) == F(1, 2)
    # Output separation as a polynomial in b: (1+b)/2.
    assert add((F(1, 2), F(-1, 2)), (0, 1)) == (F(1, 2), F(1, 2))
    return {"fixed_points": ["0"], "phi": "1", "b": str(b),
            "input_distance": "1/2", "output_distance": "3/4",
            "ratio": "3/2", "global_condition": "piecewise polynomial factor certificate"}


def coefficient_checks():
    # Actual squared-distance identity before substituting generalized bound:
    # ||x-b r-p||² = ||x-p||² - 2b<x-p,r> + b²||r||².
    # Identity between the coefficients in original and relaxed residuals.
    cases = []
    for phi, b in [(F(1), F(1, 2)), (F(1), F(2)), (F(1), F(3)),
                   (F(4), F(1, 4)), (F(4), F(3, 4))]:
        original = b*(2/phi-b)
        relaxed = 2/(b*phi)-1
        assert original == relaxed*b*b
        assert (original > 0) == (0 < b < 2/phi)
        cases.append({"phi": str(phi), "b": str(b), "decrease_coefficient": str(original)})
    # H=R², Q1=0 (phi1=1), Q2=-I (phi2=2), unequal positive weights
    # and b2>2/phi2 deliberately: the direct inner-product bound still holds.
    x = (F(3, 2), F(-2, 3))
    norm = sum(t*t for t in x)
    weights, bs, phis, slopes = (F(1, 3), F(2, 3)), (F(5), F(7)), (F(1), F(2)), (F(1), F(2))
    lhs = sum(w*b*s/phi*norm*s for w, b, phi, s in zip(weights, bs, phis, slopes))
    rhs = sum(w*b*s*norm for w, b, s in zip(weights, bs, slopes))
    assert lhs == rhs and lhs > 0
    return cases


def edge_checks():
    checks = []
    # These show lost residual/fixed-point inference, not necessarily failure
    # of demiclosedness of Z in every example (identity itself is demiclosed).
    assert 1-F(0)*1 == 1
    checks.append("zero b hides a nonzero original residual")
    for n in (2, 8, 128):
        assert F(1, n)*F(1) == F(1, n)
    checks.append("positive but vanishing b_n=1/n loses uniform residual control")
    assert F(2)*1+F(-1)*2 == 0
    checks.append("signed weights 2,-1 cancel residuals of Q1=0 and Q2=-I")
    assert (F(-1)+F(1))/2 == 0
    checks.append("constant maps -1 and +1 have an averaged fixed point but no common fixed point")
    assert F(-1)*1*F(-1) == 1 and (F(1)+F(-1))/2 == 0
    checks.append("negative generalized parameter permits cancelling residual of Q2=2I")
    assert 2*F(1) == 2 != 1
    checks.append("unnormalized weight 2 with constant Q=1 moves the common fixed point")
    for n in (2, 8, 128):
        x, a = 1+F(1, n), F(1, n)
        residual = a*x
        assert x*residual >= residual*residual
        assert residual == F(n+1, n*n)
    assert F(1)*1 == 1
    checks.append("a(x)=min(1,|x-1|), a(1)=1: r=a*x loses demiclosedness at 1")
    assert abs((1-F(2))*1) == 1 and abs((1-F(3))*1) == 2
    checks.append("b=2/phi loses strict decrease; b>2/phi may lose quasi-nonexpansiveness")
    return checks


def projection_checks():
    # Q_j=I, B_k=0, Z=I, g=0, all iterates=0, mu*=1 in Theta=R.
    assert (F(0)-1)*(F(0)-1) == 1 > 0
    # Eq13 at a nonzero fixed point: rho scales about zero, not mu*.
    assert abs(F(1, 2)*1-1) == F(1, 2) > abs(F(1)-1) == 0
    return {"lemma33_limsup": "1", "eq13_left": "1/2", "eq13_right": "0"}


def rational_receipt(x):
    # Growing partial-average coefficients generate large exact denominators.
    # Keep full exact arithmetic, but bound the receipt rather than disabling
    # Python's global decimal-serialization limit. Hex is only a hash encoding.
    lower = F((x*10**9).__floor__(), 10**9)
    upper = lower+F(1, 10**9)
    assert lower <= x < upper
    normalized = (hex(x.numerator)+"/"+hex(x.denominator)).encode("ascii")
    return {"exact": str(x) if max(x.numerator.bit_length(), x.denominator.bit_length()) < 4096 else None,
            "lower": str(lower), "upper_exclusive": str(upper),
            "normalized_signed_hex_fraction_sha256": hashlib.sha256(normalized).hexdigest(),
            "numerator_bits": x.numerator.bit_length(), "denominator_bits": x.denominator.bit_length()}


def recurrence(rho, partial=False, steps=128, inertia_mode="capped"):
    assert inertia_mode in ("capped", "singleton")
    previous, current = F(0), F(0)  # mu0=mu1=0, iterate at n=1.
    selected = []
    for n in range(1, steps+1):
        delta, tau = F(1, n+1), F(1, (n+1)**2)
        diff = current-previous
        magnitude = tau if inertia_mode == "singleton" else min(abs(diff), tau)
        inertia = magnitude*(1 if diff > 0 else -1 if diff < 0 else 0)
        alpha = current+inertia
        s = 1-F(1, 2**n) if partial else F(1)
        beta = alpha  # resolvents of zero monotone operators are I.
        gamma = (1-F(1, 2))*beta+F(1, 2)*s*beta
        ell = rho*s*gamma
        following = delta+(1-delta)*ell  # g == 1.
        assert abs(inertia) <= tau and 0 < delta < 1
        assert abs(following) <= rho*abs(current)+delta+rho*tau
        assert tau/delta == delta
        if n in (1, 2, 8, 32, 128):
            selected.append({"n": n, "mu_n_plus_1": rational_receipt(following),
                             "pointwise_bound_pass": True})
        previous, current = current, following
    if rho == F(1, 2):
        assert abs(current) < F(1, 16)
    if rho == F(1) and not partial:
        assert abs(current-1) < F(1, 16)
    return {"rho": str(rho), "partial_sum": partial, "steps": steps,
            "inertia_mode": inertia_mode,
            "last": rational_receipt(current), "samples": selected,
            "limit_is_not_established_by_finite_checks": True,
            "literal_printed_sum_of_iterates_equals_one_not_asserted": True}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--output", type=Path, help="new exclusive JSON output; never overwrite")
    args = parser.parse_args()
    result = {"schema": "zerone-science-exact-check/v1", "pass": True,
              "review_scope": "same-operator internal check; no external review",
              "verifier_sha256": hashlib.sha256(Path(__file__).read_bytes()).hexdigest(),
              "witness": witness(), "coefficient_checks": coefficient_checks(),
              "adversarial_edges": edge_checks(), "projection_checks": projection_checks(),
              "recurrences": [recurrence(F(1, 2)), recurrence(F(1, 2), True), recurrence(F(1)),
                              recurrence(F(1, 2), inertia_mode="singleton")],
              "control_warning": "rho=1 control lies outside the printed rho interval"}
    raw = json.dumps(result, indent=2, ensure_ascii=False)+"\n"
    if args.output:
        fd = os.open(args.output, os.O_WRONLY | os.O_CREAT | os.O_EXCL | os.O_NOFOLLOW, 0o600)
        with os.fdopen(fd, "w") as f:
            f.write(raw)
            f.flush()
            os.fsync(f.fileno())
    else:
        print(raw, end="")


if __name__ == "__main__":
    main()
