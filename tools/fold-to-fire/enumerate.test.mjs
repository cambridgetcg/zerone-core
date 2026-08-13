import assert from "node:assert/strict";
import { spawnSync } from "node:child_process";
import { describe, it } from "node:test";

import {
  FOLD_TO_FIRE_MAX_EXACT_STEPS,
  FOLD_TO_FIRE_EVIDENCE_ROLE,
  enumerateFoldToFire,
  evaluatePolynomialRational,
  exactActivityFraction,
  evaluatePolynomialApproximate,
  serializeEnumeration,
} from "./enumerate.mjs";

const DIRECTIONS = [
  [1, 0],
  [0, 1],
  [-1, 0],
  [0, -1],
];

function key(x, y) {
  return `${x},${y}`;
}

// Deliberately slower reference kernel: score contacts only after a complete
// walk has been generated. It shares neither the incremental contact counter
// nor its bookkeeping. This is a local cross-check, not independent evidence.
function referenceEnumeration(stepCount) {
  const all = [];
  const active = [];
  const path = [
    [0, 0],
    [1, 0],
  ];
  const visited = new Set(path.map(([x, y]) => key(x, y)));

  function add(polynomial, degree) {
    while (polynomial.length <= degree) polynomial.push(0n);
    polynomial[degree] += 1n;
  }

  function score() {
    let contacts = 0;
    for (let left = 0; left < path.length; left += 1) {
      for (let right = left + 2; right < path.length; right += 1) {
        const [leftX, leftY] = path[left];
        const [rightX, rightY] = path[right];
        if (Math.abs(leftX - rightX) + Math.abs(leftY - rightY) === 1) {
          contacts += 1;
        }
      }
    }
    add(all, contacts);
    const [endX, endY] = path.at(-1);
    if (stepCount >= 3 && Math.abs(endX) + Math.abs(endY) === 1) {
      add(active, contacts);
    }
  }

  function walk() {
    if (path.length === stepCount + 1) {
      score();
      return;
    }
    const [x, y] = path.at(-1);
    for (const [dx, dy] of DIRECTIONS) {
      const next = [x + dx, y + dy];
      const nextKey = key(next[0], next[1]);
      if (visited.has(nextKey)) continue;
      visited.add(nextKey);
      path.push(next);
      walk();
      path.pop();
      visited.delete(nextKey);
    }
  }

  walk();
  return { all, active };
}

describe("Fold-to-Fire exact enumerator", () => {
  it("reproduces the hand-solvable contact polynomials", () => {
    assert.deepEqual(serializeEnumeration(enumerateFoldToFire(3)), {
      stepCount: 3,
      allByContacts: ["7", "2"],
      activeByContacts: ["0", "2"],
      totalWalks: "9",
      activeWalks: "2",
    });
    assert.deepEqual(serializeEnumeration(enumerateFoldToFire(5)), {
      stepCount: 5,
      allByContacts: ["41", "22", "8"],
      activeByContacts: ["0", "0", "6"],
      totalWalks: "71",
      activeWalks: "6",
    });
  });

  it("reproduces standard q=1 walk and closure counts", () => {
    const expected = [
      [3, 9n, 2n],
      [5, 71n, 6n],
      [7, 543n, 28n],
      [9, 4_067n, 140n],
      [11, 30_073n, 744n],
      [13, 220_375n, 4_116n],
      [15, 1_604_149n, 23_504n],
    ];
    for (const [stepCount, totalWalks, activeWalks] of expected) {
      const result = enumerateFoldToFire(stepCount);
      assert.equal(result.totalWalks, totalWalks, `n=${stepCount} walks`);
      assert.equal(result.activeWalks, activeWalks, `n=${stepCount} active`);
    }
  });

  it("keeps the backbone bond out of contacts and respects square-lattice parity", () => {
    const oneStep = enumerateFoldToFire(1);
    assert.equal(oneStep.totalWalks, 1n);
    assert.equal(oneStep.activeWalks, 0n);
    assert.deepEqual(oneStep.allByContacts, [1n]);

    for (let stepCount = 2; stepCount <= 14; stepCount += 2) {
      const result = enumerateFoldToFire(stepCount);
      assert.equal(result.activeWalks, 0n, `n=${stepCount} active`);
    }
  });

  it("agrees with a post-scored reference kernel through n=11", () => {
    for (let stepCount = 3; stepCount <= 11; stepCount += 2) {
      const exact = enumerateFoldToFire(stepCount);
      const reference = referenceEnumeration(stepCount);
      assert.deepEqual(exact.allByContacts, reference.all, `n=${stepCount} all`);
      assert.deepEqual(
        exact.activeByContacts,
        reference.active,
        `n=${stepCount} active`,
      );
    }
  });

  it("evaluates weighted polynomials without changing exact evidence", () => {
    const result = enumerateFoldToFire(3);
    assert.equal(evaluatePolynomialApproximate(result.allByContacts, 2), 11);
    assert.equal(evaluatePolynomialApproximate(result.activeByContacts, 2), 4);
    assert.deepEqual(
      evaluatePolynomialRational(result.allByContacts, {
        numerator: 3n,
        denominator: 2n,
      }),
      { numerator: 10n, denominator: 1n },
    );
    assert.deepEqual(
      exactActivityFraction(result, { numerator: 2n, denominator: 1n }),
      { numerator: 4n, denominator: 11n },
    );
  });

  it("keeps every active coefficient within its all-walk coefficient", () => {
    for (let stepCount = 3; stepCount <= 15; stepCount += 2) {
      const result = enumerateFoldToFire(stepCount);
      for (let degree = 0; degree < result.activeByContacts.length; degree += 1) {
        assert.ok(
          result.activeByContacts[degree] <=
            (result.allByContacts[degree] ?? 0n),
          `n=${stepCount}, contact=${degree}`,
        );
      }
    }
  });

  it("rejects unsupported bounds and weights", () => {
    for (const value of [0, 1.5, FOLD_TO_FIRE_MAX_EXACT_STEPS + 1]) {
      assert.throws(() => enumerateFoldToFire(value), RangeError);
    }
    for (const value of [0, -1, Number.NaN, Number.POSITIVE_INFINITY]) {
      assert.throws(
        () => evaluatePolynomialApproximate([1n], value),
        RangeError,
      );
    }
    assert.throws(
      () =>
        evaluatePolynomialRational([1n], {
          numerator: 1n,
          denominator: 0n,
        }),
      RangeError,
    );
  });

  it("emits machine-readable evidence with the sealed role vocabulary", () => {
    const execution = spawnSync(
      process.execPath,
      [new URL("./enumerate.mjs", import.meta.url).pathname, "--max-step", "5", "--json"],
      { encoding: "utf8" },
    );
    assert.equal(execution.status, 0, execution.stderr);
    const output = JSON.parse(execution.stdout);
    assert.equal(output.evidenceRole, FOLD_TO_FIRE_EVIDENCE_ROLE);
    assert.deepEqual(
      output.rows.map(({ stepCount, totalWalks, activeWalks }) => [
        stepCount,
        totalWalks,
        activeWalks,
      ]),
      [
        [3, "9", "2"],
        [5, "71", "6"],
      ],
    );
  });
});
