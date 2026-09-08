// Separately implemented, post-scored local cross-check. Not independent evidence.
// No imports from the production enumerator or its rational evaluator.
const DIRECTIONS = [[1, 0], [0, 1], [-1, 0], [0, -1]];
const key = (x, y) => `${x},${y}`;

export function referenceEnumeration(stepCount) {
  if (!Number.isSafeInteger(stepCount) || stepCount < 1 || stepCount > 11) {
    throw new RangeError("reference stepCount must be an integer from 1 to 11");
  }
  const all = [];
  const active = [];
  const path = [[0, 0], [1, 0]];
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
        if (Math.abs(leftX - rightX) + Math.abs(leftY - rightY) === 1) contacts += 1;
      }
    }
    add(all, contacts);
    const [endX, endY] = path.at(-1);
    if (stepCount >= 3 && Math.abs(endX) + Math.abs(endY) === 1) add(active, contacts);
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

function reduce(numerator, denominator) {
  let a = numerator;
  let b = denominator;
  while (b) [a, b] = [b, a % b];
  return { numerator: numerator / a, denominator: denominator / a };
}

// Horner evaluation using fraction addition/multiplication, unlike the
// production evaluator's sum of powers. Deliberately no shared arithmetic.
function evaluate(coefficients, q) {
  let value = { numerator: 0n, denominator: 1n };
  for (let i = coefficients.length - 1; i >= 0; i -= 1) {
    value = reduce(
      value.numerator * q.numerator + coefficients[i] * value.denominator * q.denominator,
      value.denominator * q.denominator,
    );
  }
  return value;
}

export function referenceActivityFraction(stepCount, q) {
  if (!q || typeof q.numerator !== "bigint" || typeof q.denominator !== "bigint" ||
      q.numerator <= 0n || q.denominator <= 0n) {
    throw new RangeError("reference q must be a positive BigInt rational");
  }
  const enumeration = referenceEnumeration(stepCount);
  const all = evaluate(enumeration.all, q);
  const active = evaluate(enumeration.active, q);
  return reduce(active.numerator * all.denominator, active.denominator * all.numerator);
}
