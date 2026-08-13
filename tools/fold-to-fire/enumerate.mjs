#!/usr/bin/env node

/**
 * Exact finite enumerator for the Fold-to-Fire square-lattice model.
 *
 * This program produces bounded computational evidence. It does not prove the
 * asymptotic closing exponent or model an atomic protein.
 */

const DIRECTIONS = Object.freeze([
  Object.freeze([1, 0]),
  Object.freeze([0, 1]),
  Object.freeze([-1, 0]),
  Object.freeze([0, -1]),
]);

export const FOLD_TO_FIRE_SOLVER_VERSION = "fold-to-fire-exact-dfs/v0";
export const FOLD_TO_FIRE_MAX_EXACT_STEPS = 15;
export const FOLD_TO_FIRE_EVIDENCE_ROLE =
  "EXACT_FINITE_ENUMERATION_NOT_ASYMPTOTIC_PROOF";

function pointKey(x, y) {
  return `${x},${y}`;
}

function assertStepCount(stepCount) {
  if (
    !Number.isSafeInteger(stepCount) ||
    stepCount < 1 ||
    stepCount > FOLD_TO_FIRE_MAX_EXACT_STEPS
  ) {
    throw new RangeError(
      `stepCount must be an integer from 1 to ${FOLD_TO_FIRE_MAX_EXACT_STEPS}`,
    );
  }
}

function increment(polynomial, degree) {
  while (polynomial.length <= degree) polynomial.push(0n);
  polynomial[degree] += 1n;
}

function sumPolynomial(polynomial) {
  return polynomial.reduce((sum, coefficient) => sum + coefficient, 0n);
}

/**
 * Enumerate n-step nearest-neighbour self-avoiding walks on Z^2 with
 * v_0=(0,0) and v_1=(1,0). Rotations are therefore quotiented out, while
 * reflections remain distinct. A contact is a nearest-neighbour pair whose
 * sequence indices differ by at least two. A walk is catalytically competent
 * in this toy model when n >= 3 and v_n is adjacent to v_0.
 */
export function enumerateFoldToFire(stepCount) {
  assertStepCount(stepCount);

  const allByContacts = [];
  const activeByContacts = [];
  const visited = new Set([pointKey(0, 0), pointKey(1, 0)]);
  function visit(step, x, y, contacts) {
    if (step === stepCount) {
      increment(allByContacts, contacts);
      if (stepCount >= 3 && Math.abs(x) + Math.abs(y) === 1) {
        increment(activeByContacts, contacts);
      }
      return;
    }

    for (const [dx, dy] of DIRECTIONS) {
      const nextX = x + dx;
      const nextY = y + dy;
      const nextKey = pointKey(nextX, nextY);
      if (visited.has(nextKey)) continue;

      let addedContacts = 0;
      for (const [contactDx, contactDy] of DIRECTIONS) {
        const neighbourX = nextX + contactDx;
        const neighbourY = nextY + contactDy;
        if (neighbourX === x && neighbourY === y) {
          continue;
        }
        if (visited.has(pointKey(neighbourX, neighbourY))) {
          addedContacts += 1;
        }
      }

      visited.add(nextKey);
      visit(step + 1, nextX, nextY, contacts + addedContacts);
      visited.delete(nextKey);
    }
  }

  if (stepCount === 1) {
    increment(allByContacts, 0);
  } else {
    visit(1, 1, 0, 0);
  }

  const totalWalks = sumPolynomial(allByContacts);
  const activeWalks = sumPolynomial(activeByContacts);
  return Object.freeze({
    stepCount,
    allByContacts: Object.freeze(allByContacts),
    activeByContacts: Object.freeze(activeByContacts),
    totalWalks,
    activeWalks,
  });
}

export function serializeEnumeration(enumeration) {
  return {
    stepCount: enumeration.stepCount,
    allByContacts: enumeration.allByContacts.map(String),
    activeByContacts: enumeration.activeByContacts.map(String),
    totalWalks: String(enumeration.totalWalks),
    activeWalks: String(enumeration.activeWalks),
  };
}

/**
 * Convenience projection for plotting only. BigInt-to-Number conversion can
 * round, so exact evidence must use evaluatePolynomialRational instead.
 */
export function evaluatePolynomialApproximate(polynomial, q) {
  if (typeof q !== "number" || !Number.isFinite(q) || q <= 0) {
    throw new RangeError("q must be a finite positive number");
  }
  let value = 0;
  for (let degree = polynomial.length - 1; degree >= 0; degree -= 1) {
    value = value * q + Number(polynomial[degree]);
  }
  return value;
}

function greatestCommonDivisor(left, right) {
  let a = left < 0n ? -left : left;
  let b = right < 0n ? -right : right;
  while (b !== 0n) {
    const remainder = a % b;
    a = b;
    b = remainder;
  }
  return a;
}

function asPositiveRational(value, path) {
  if (
    typeof value !== "object" ||
    value === null ||
    typeof value.numerator !== "bigint" ||
    typeof value.denominator !== "bigint" ||
    value.numerator <= 0n ||
    value.denominator <= 0n
  ) {
    throw new RangeError(`${path} must be a positive BigInt rational`);
  }
  const divisor = greatestCommonDivisor(value.numerator, value.denominator);
  return {
    numerator: value.numerator / divisor,
    denominator: value.denominator / divisor,
  };
}

export function evaluatePolynomialRational(polynomial, q) {
  const rational = asPositiveRational(q, "q");
  const maximumDegree = Math.max(0, polynomial.length - 1);
  let numerator = 0n;
  for (let degree = 0; degree <= maximumDegree; degree += 1) {
    numerator +=
      (polynomial[degree] ?? 0n) *
      rational.numerator ** BigInt(degree) *
      rational.denominator ** BigInt(maximumDegree - degree);
  }
  const denominator = rational.denominator ** BigInt(maximumDegree);
  const divisor = greatestCommonDivisor(numerator, denominator);
  return Object.freeze({
    numerator: numerator / divisor,
    denominator: denominator / divisor,
  });
}

export function exactActivityFraction(enumeration, q) {
  const all = evaluatePolynomialRational(enumeration.allByContacts, q);
  const active = evaluatePolynomialRational(enumeration.activeByContacts, q);
  const numerator = active.numerator * all.denominator;
  const denominator = active.denominator * all.numerator;
  const divisor = greatestCommonDivisor(numerator, denominator);
  return Object.freeze({
    numerator: numerator / divisor,
    denominator: denominator / divisor,
  });
}

function parseCli(arguments_) {
  let maximum = FOLD_TO_FIRE_MAX_EXACT_STEPS;
  let json = false;
  for (let index = 0; index < arguments_.length; index += 1) {
    const argument = arguments_[index];
    if (argument === "--json") {
      json = true;
      continue;
    }
    if (argument === "--max-step") {
      const raw = arguments_[index + 1];
      if (raw === undefined) throw new Error("--max-step requires an integer");
      maximum = Number(raw);
      index += 1;
      continue;
    }
    throw new Error(`unknown argument: ${argument}`);
  }
  assertStepCount(maximum);
  if (maximum < 3) throw new RangeError("--max-step must be at least 3");
  return { maximum, json };
}

function main() {
  try {
    const { maximum, json } = parseCli(process.argv.slice(2));
    const rows = [];
    for (let stepCount = 3; stepCount <= maximum; stepCount += 2) {
      rows.push(serializeEnumeration(enumerateFoldToFire(stepCount)));
    }
    if (json) {
      process.stdout.write(
        `${JSON.stringify(
          {
            solver: FOLD_TO_FIRE_SOLVER_VERSION,
            convention: "v0=(0,0);v1=(1,0);reflections-distinct",
            evidenceRole: FOLD_TO_FIRE_EVIDENCE_ROLE,
            rows,
          },
          null,
          2,
        )}\n`,
      );
      return;
    }
    console.table(
      rows.map((row) => ({
        n: row.stepCount,
        walks: row.totalWalks,
        active: row.activeWalks,
        probabilityApproximate:
          Number(row.activeWalks) / Number(row.totalWalks),
      })),
    );
  } catch (error) {
    console.error(error instanceof Error ? error.message : String(error));
    process.exitCode = 2;
  }
}

if (import.meta.url === new URL(process.argv[1], "file:").href) main();
