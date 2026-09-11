#!/usr/bin/env python3
"""Reproduce a deliberately incomplete review and the counterexample it misses.

python3 docs/examples/local-claims/prime-polynomial.py review
python3 docs/examples/local-claims/prime-polynomial.py counterexample

Output is UTF-8 JSON with sorted keys, compact separators and one final LF.
Hash those exact bytes to obtain the evidence ID used by the local walkthrough.
All participants in that walkthrough are synthetic accounts of one operator.
"""
import argparse
import json
from math import isqrt


def prime(number):
    return number >= 2 and all(number % divisor for divisor in range(2, isqrt(number) + 1))


def evidence(kind):
    if kind == "review":
        rows = [{"n": n, "value": n * n + n + 41, "prime": prime(n * n + n + 41)}
                for n in range(40)]
        assert all(row["prime"] for row in rows)
        return {"method": "trial division through integer square root", "range_inclusive": [0, 39],
                "rows": rows, "scope": "Does not test n=40; cannot establish the full submitted claim."}
    value = 40 * 40 + 40 + 41
    assert value == 41 * 41 and not prime(value)
    return {"n": 40, "value": value, "factors": [41, 41], "prime": False,
            "scope": "One counterexample disproves the claim over every integer n from 0 through 40."}


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("kind", choices=("review", "counterexample"))
    args = parser.parse_args()
    print(json.dumps(evidence(args.kind), sort_keys=True, separators=(",", ":")))
