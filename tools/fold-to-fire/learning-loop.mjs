#!/usr/bin/env node
/**
 * Private, finite lab protocol; not a public ToK/witness/PoCA schema.
 * One JSON object on stdin, one JSON object on stdout, EOF terminates input.
 * No paths, commands, URLs, dynamic imports, or execution instructions in JSON.
 * The Go caller must also bound process time/output and bind these source files.
 *
 * Operations (unknown keys are rejected):
 *   {op:"fixture"} -> fixtureMaterial(): fixed tasks, polynomial contents, fault.
 *   {op:"cached",parent:Fact,tasks:Task[]} -> cachedMaterial(): cache contents.
 *   {op:"consume",view:"node"|"graph"|"rows",...Snapshot} -> consumeFacts().
 *   {op:"check",tasks:Task[],outputs:Output[]} -> checkOutputs(), local oracle.
 *   {op:"compare",before:Snapshot,after:Snapshot} -> compareControls().
 * Snapshot = {tasks,budget,facts,relations,history,historyComplete}.
 * Task = {n:integer,q:string}; only n=3,5,7 and q="1","3/2","2" are in scope.
 * Other bounded tasks produce ABSTAIN, never a larger enumeration.
 * Fact = {id,content,status,revision,provenance?}; content is EXACT JSON text.
 * provenance is an optional inert JSON object retaining available chain metadata.
 * IDs/revisions are bounded strings. contentDigest hashes UTF-8 content bytes;
 * factDigest also binds standing/revision/provenance (NOT the topology root).
 * Relation = {source,target,type:"REQUIRES"|"SUPERSEDES"}:
 * REQUIRES child -> parent; SUPERSEDES replacement -> old. No transitive truth.
 * History = {oldId,newId,outcome:"ACCEPTED"|"REJECTED"|"UNRESOLVED"}.
 * History is a supplied correction projection, NOT an independently verified log.
 * Caller retains original bundles/events separately, including missing evidence.
 *
 * All views receive identical fact content/provenance, tasks and budgets. "node"
 * uses embedded cache provenance, not relation/history tables; it may abstain or
 * recompute. "graph" and "rows" interpret exactly the same relation/event data.
 * A budget unit permits one production enumeration for one requested task.
 * Unsafe/unresolved evidence abstains; only absence of relevant facts permits
 * recomputation. The oracle runs only in check/compare, never inside consume.
 * Successful reuse is a local policy decision, not a truth/consensus assertion.
 */
import { createHash } from "node:crypto";
import { pathToFileURL } from "node:url";
import { enumerateFoldToFire, exactActivityFraction } from "./enumerate.mjs";
import { referenceActivityFraction } from "./reference.mjs";

export const MAX_INPUT_BYTES = 262144;
const NS = [3, 5, 7];
const QS = ["1", "3/2", "2"];
const VIEWS = ["node", "graph", "rows"];
const STATUSES = ["ACCEPTED", "VERIFIED", "CONTESTED", "REJECTED", "SUPERSEDED", "UNRESOLVED", "DISPROVEN", "PENDING"];
const SCHEMA = "tok-fold-fact/0";
const METHOD = "fold-to-fire-exact-dfs/v0";
const EFFECTS = Object.freeze({ allocation_status: "NOT_EVALUATED", economic_effect: "0", qualification_effect: "0", live_effect: "0" });

function requireThat(condition, message) {
  if (!condition) throw new Error(message);
}
function object(value, keys, required = keys) {
  requireThat(value !== null && typeof value === "object" && !Array.isArray(value), "expected object");
  requireThat(Object.keys(value).every((key) => keys.includes(key)), "unknown field");
  requireThat(required.every((key) => Object.hasOwn(value, key)), "missing field");
}
function text(value, maximum = 128) {
  requireThat(typeof value === "string" && value.length > 0 && value.length <= maximum, "invalid bounded string");
}
function list(value, maximum) {
  requireThat(Array.isArray(value) && value.length <= maximum, "invalid bounded array");
}
function integer(value, minimum, maximum) {
  requireThat(Number.isSafeInteger(value) && value >= minimum && value <= maximum, "invalid bounded integer");
}
function decimal(value, positive = false) {
  requireThat(typeof value === "string" && /^(0|[1-9][0-9]{0,11})$/.test(value), "invalid decimal");
  requireThat(!positive || value !== "0", "expected positive decimal");
  return BigInt(value);
}
// Small bounded canonical JSON for worker digests. Receipts use the existing Go
// CanonicalJSON separately; no cross-language canonicalization is assumed here.
export function canonicalJSON(value) {
  function visit(item, depth) {
    requireThat(depth <= 12, "JSON nesting limit");
    if (item === null || typeof item === "boolean") return JSON.stringify(item);
    if (typeof item === "string") {
      requireThat(item.length <= 16384, "JSON string limit");
      return JSON.stringify(item);
    }
    if (typeof item === "number") {
      requireThat(Number.isSafeInteger(item), "JSON numbers must be safe integers");
      return String(item);
    }
    requireThat(item && typeof item === "object", "invalid JSON value");
    if (Array.isArray(item)) {
      list(item, 128);
      return `[${item.map((child) => visit(child, depth + 1)).join(",")}]`;
    }
    const keys = Object.keys(item).sort();
    requireThat(keys.length <= 32, "JSON key limit");
    return `{${keys.map((key) => `${JSON.stringify(key)}:${visit(item[key], depth + 1)}`).join(",")}}`;
  }
  const result = visit(value, 0);
  requireThat(Buffer.byteLength(result) <= MAX_INPUT_BYTES, "JSON byte limit");
  return result;
}
export const contentDigest = (content) => createHash("sha256").update(content, "utf8").digest("hex");
const digest = (value) => contentDigest(canonicalJSON(value));
const accepted = (fact) => ["ACCEPTED", "VERIFIED"].includes(fact.status);
const inScope = (task) => NS.includes(task.n) && QS.includes(task.q);
const rational = (q) => {
  const [numerator, denominator = "1"] = q.split("/");
  return { numerator: BigInt(numerator), denominator: BigInt(denominator) };
};
const fraction = (value) => ({ numerator: String(value.numerator), denominator: String(value.denominator) });
const equalFraction = (left, right) => left.numerator === right.numerator && left.denominator === right.denominator;
function tasksShape(tasks) {
  list(tasks, 9);
  requireThat(tasks.length > 0, "tasks cannot be empty");
  for (const task of tasks) {
    object(task, ["n", "q"]);
    integer(task.n, 0, 100);
    text(task.q, 16);
  }
  requireThat(new Set(tasks.map(canonicalJSON)).size === tasks.length, "duplicate task");
}
function fractionShape(value) {
  object(value, ["numerator", "denominator"]);
  const numerator = decimal(value.numerator);
  const denominator = decimal(value.denominator, true);
  requireThat(numerator <= denominator, "activity outside [0,1]");
  let a = numerator;
  let b = denominator;
  while (b) [a, b] = [b, a % b];
  requireThat(a === 1n, "fraction must be reduced");
}
function payloadShape(payload) {
  requireThat(payload && payload.schema === SCHEMA, "unsupported content schema");
  requireThat(NS.includes(payload.n), "unsupported polynomial n");
  if (payload.kind === "polynomial") {
    object(payload, ["schema", "kind", "n", "all", "active", "provenance"]);
    object(payload.provenance, ["method", "scope"]);
    requireThat(payload.provenance.method === METHOD && payload.provenance.scope === "finite-square-lattice", "unsupported polynomial provenance");
    for (const coefficients of [payload.all, payload.active]) {
      list(coefficients, 8);
      requireThat(coefficients.length > 0, "empty polynomial");
      // Keep even adversarial fixture arithmetic inside the output decimal cap.
      coefficients.forEach((coefficient) => requireThat(decimal(coefficient) <= 1000000n, "coefficient limit"));
    }
    requireThat(payload.all.some((coefficient) => coefficient !== "0"), "zero all polynomial");
    for (let i = 0; i < payload.active.length; i += 1) {
      requireThat(BigInt(payload.active[i]) <= BigInt(payload.all[i] ?? "0"), "active coefficient exceeds all");
    }
  } else {
    requireThat(payload.kind === "activity", "unsupported fact kind");
    object(payload, ["schema", "kind", "n", "q", "value", "provenance"]);
    requireThat(QS.includes(payload.q), "unsupported cached q");
    fractionShape(payload.value);
    object(payload.provenance, ["factId", "contentDigest"]);
    text(payload.provenance.factId);
    requireThat(/^[a-f0-9]{64}$/.test(payload.provenance.contentDigest), "invalid provenance digest");
  }
}
function parseFact(fact) {
  object(fact, ["id", "content", "status", "revision", "provenance"], ["id", "content", "status", "revision"]);
  text(fact.id);
  text(fact.content, 8192);
  text(fact.revision);
  requireThat(STATUSES.includes(fact.status), "unsupported fact status");
  if (fact.provenance !== undefined) {
    requireThat(fact.provenance && typeof fact.provenance === "object" && !Array.isArray(fact.provenance), "invalid fact provenance");
  }
  const payload = JSON.parse(fact.content);
  payloadShape(payload);
  return { ...fact, payload, ref: { id: fact.id, contentDigest: contentDigest(fact.content), factDigest: digest(fact), revision: fact.revision } };
}
function snapshotShape(snapshot) {
  object(snapshot, ["tasks", "budget", "facts", "relations", "history", "historyComplete"]);
  canonicalJSON(snapshot);
  tasksShape(snapshot.tasks);
  integer(snapshot.budget, 0, 9);
  list(snapshot.facts, 32);
  list(snapshot.relations, 64);
  list(snapshot.history, 32);
  requireThat(typeof snapshot.historyComplete === "boolean", "historyComplete must be boolean");
  const facts = snapshot.facts.map(parseFact);
  requireThat(new Set(facts.map((fact) => fact.id)).size === facts.length, "duplicate fact id");
  for (const edge of snapshot.relations) {
    object(edge, ["source", "target", "type"]);
    text(edge.source);
    text(edge.target);
    requireThat(["REQUIRES", "SUPERSEDES"].includes(edge.type), "unsupported relation type");
    requireThat(edge.source !== edge.target, "self relation");
  }
  for (const event of snapshot.history) {
    object(event, ["oldId", "newId", "outcome"]);
    text(event.oldId);
    text(event.newId);
    requireThat(event.oldId !== event.newId, "self correction");
    requireThat(["ACCEPTED", "REJECTED", "UNRESOLVED"].includes(event.outcome), "unsupported history outcome");
  }
  for (const entries of [snapshot.relations, snapshot.history]) {
    requireThat(new Set(entries.map(canonicalJSON)).size === entries.length, "duplicate relation/history");
  }
  return facts;
}
function evaluate(payload, q) {
  return fraction(exactActivityFraction({
    allByContacts: payload.all.map(BigInt), activeByContacts: payload.active.map(BigInt),
  }, rational(q)));
}

export function fixtureMaterial() {
  const tasks = NS.flatMap((n) => QS.map((q) => ({ n, q })));
  const polynomials = NS.map((n) => {
    const enumeration = enumerateFoldToFire(n);
    const payload = { schema: SCHEMA, kind: "polynomial", n,
      all: enumeration.allByContacts.map(String), active: enumeration.activeByContacts.map(String),
      provenance: { method: METHOD, scope: "finite-square-lattice" } };
    return { n, content: canonicalJSON(payload) };
  });
  const injected = JSON.parse(polynomials.find((item) => item.n === 5).content);
  requireThat(canonicalJSON(injected.all) === '["41","22","8"]', "fault precondition changed");
  injected.all = ["42", "21", "8"];
  return { protocol: "tok-fold-learning-worker/0", tasks, polynomials,
    fault: { label: "LOCAL_COEFFICIENT_FAULT_NOT_SEALED_EVIDENCE", n: 5, content: canonicalJSON(injected),
      change: { before: ["41", "22", "8"], after: ["42", "21", "8"] } },
    policy: { n: NS, q: QS, budgetUnit: "one enumeration per task", referenceRole: "LOCAL_CROSS_CHECK_NOT_INDEPENDENT_EVIDENCE" },
    effects: EFFECTS };
}

export function cachedMaterial(parent, tasks) {
  canonicalJSON(parent);
  const fact = parseFact(parent);
  tasksShape(tasks);
  requireThat(fact.payload.kind === "polynomial", "cached parent must be polynomial");
  return { cached: tasks.map((task) => {
    requireThat(inScope(task) && task.n === fact.payload.n, "cached task outside parent scope");
    const payload = { schema: SCHEMA, kind: "activity", ...task, value: evaluate(fact.payload, task.q),
      provenance: { factId: fact.id, contentDigest: fact.ref.contentDigest } };
    return { ...task, content: canonicalJSON(payload) };
  }) };
}

// Only direct relationships and explicit correction records are interpreted.
function historyProblem(snapshot, facts, n) {
  const byId = new Map(facts.map((fact) => [fact.id, fact]));
  const relevant = (id) => byId.get(id)?.payload.n === n;
  if (!snapshot.historyComplete && facts.some((fact) => fact.payload.n === n)) return "INCOMPLETE_HISTORY";
  // A record with neither endpoint present cannot be classified as disconnected.
  if (snapshot.history.some((event) => !byId.has(event.oldId) && !byId.has(event.newId))) return "MISSING_OR_WRONG_CORRECTION_ENDPOINT";
  const events = snapshot.history.filter((event) => relevant(event.oldId) || relevant(event.newId));
  for (const event of events) {
    const old = byId.get(event.oldId);
    const replacement = byId.get(event.newId);
    if (!old || !replacement || old.payload.kind !== "polynomial" || replacement.payload.kind !== "polynomial" ||
        old.payload.n !== replacement.payload.n) return "MISSING_OR_WRONG_CORRECTION_ENDPOINT";
    if (event.outcome !== "ACCEPTED") return "UNRESOLVED_OR_REJECTED_CORRECTION";
    if (accepted(old) || !accepted(replacement)) return "CONTRADICTORY_CORRECTION_STANDING";
    if (!snapshot.relations.some((edge) => edge.type === "SUPERSEDES" && edge.source === replacement.id && edge.target === old.id)) {
      return "MISSING_OR_WRONG_SUPERSEDES";
    }
  }
  for (const edge of snapshot.relations.filter((edge) => edge.type === "SUPERSEDES" && (relevant(edge.source) || relevant(edge.target)))) {
    if (!events.some((event) => event.newId === edge.source && event.oldId === edge.target)) return "MISSING_CORRECTION_HISTORY";
  }
  return null;
}

export function consumeFacts(snapshot, view) {
  requireThat(VIEWS.includes(view), "unsupported view");
  const facts = snapshotShape(snapshot);
  const byId = new Map(facts.map((fact) => [fact.id, fact]));
  const costs = { enumerations: 0, polynomialEvaluations: 0, referenceChecks: 0 };
  const outputs = snapshot.tasks.map((task) => {
    const result = (decision, reason, value = null, used = []) => ({ task, decision, reason, value,
      used: [...new Map(used.map((fact) => [fact.id, fact.ref])).values()].sort((a, b) => a.id < b.id ? -1 : a.id > b.id ? 1 : 0) });
    if (!inScope(task)) return result("ABSTAIN", "OUT_OF_SCOPE");
    const relevant = facts.filter((fact) => fact.payload.n === task.n &&
      (fact.payload.kind === "polynomial" || fact.payload.q === task.q));
    if (view !== "node") {
      const problem = historyProblem(snapshot, facts, task.n);
      if (problem) return result("ABSTAIN", problem);
    }
    if (relevant.some((fact) => fact.payload.kind === "polynomial" && ["UNRESOLVED", "PENDING", "CONTESTED"].includes(fact.status))) {
      return result("ABSTAIN", "UNRESOLVED_POLYNOMIAL");
    }
    const usable = relevant.filter(accepted);
    const polynomials = usable.filter((fact) => fact.payload.kind === "polynomial");
    const caches = usable.filter((fact) => fact.payload.kind === "activity");
    const answers = [];
    const used = [];
    for (const cache of caches) {
      const source = cache.payload.provenance;
      const parent = byId.get(source.factId);
      if (!parent || parent.payload.kind !== "polynomial" || parent.payload.n !== task.n ||
          !accepted(parent) || parent.ref.contentDigest !== source.contentDigest) {
        return result("ABSTAIN", "STALE_OR_UNRESOLVED_CACHE_PROVENANCE");
      }
      if (view !== "node") {
        // This fixture supports exactly the cache's embedded polynomial prerequisite.
        const requires = snapshot.relations.filter((edge) => edge.type === "REQUIRES" && edge.source === cache.id);
        if (requires.length !== 1 || requires[0].target !== parent.id) return result("ABSTAIN", "MISSING_OR_WRONG_REQUIRES");
      }
      costs.polynomialEvaluations += 1;
      if (!equalFraction(cache.payload.value, evaluate(parent.payload, task.q))) return result("ABSTAIN", "CACHE_CONTENT_DISAGREEMENT");
      answers.push(cache.payload.value);
      used.push(cache, parent);
    }
    for (const polynomial of polynomials) {
      // Fixture polynomials have no supported prerequisites; do not infer their truth.
      if (view !== "node" && snapshot.relations.some((edge) => edge.type === "REQUIRES" && edge.source === polynomial.id)) {
        return result("ABSTAIN", "UNSUPPORTED_POLYNOMIAL_REQUIRES");
      }
      costs.polynomialEvaluations += 1;
      answers.push(evaluate(polynomial.payload, task.q));
      used.push(polynomial);
    }
    if (answers.length) {
      if (!answers.every((answer) => equalFraction(answer, answers[0]))) return result("ABSTAIN", "CONTRADICTORY_ACCEPTED_CONTENT");
      return result("REUSE", caches.length ? "CACHE_WITH_CURRENT_PROVENANCE" : "CURRENT_POLYNOMIAL", answers[0], used);
    }
    if (relevant.length) return result("ABSTAIN", "NO_ACCEPTED_EVIDENCE");
    if (costs.enumerations >= snapshot.budget) return result("ABSTAIN", "NO_EVIDENCE_BUDGET_EXHAUSTED");
    costs.enumerations += 1;
    return result("RECOMPUTE", "NO_REUSABLE_EVIDENCE_LOCAL_ENUMERATION",
      fraction(exactActivityFraction(enumerateFoldToFire(task.n), rational(task.q))));
  });
  return { protocol: "tok-fold-learning-worker/0", view, inputDigest: digest({ view, ...snapshot }),
    commonPayloadDigest: digest({ tasks: snapshot.tasks, budget: snapshot.budget, facts: snapshot.facts }),
    outputs, costs, effects: EFFECTS };
}

export function checkOutputs(tasks, outputs) {
  tasksShape(tasks);
  list(outputs, 9);
  requireThat(tasks.length === outputs.length, "output/task count mismatch");
  const metrics = { exactAnswerAgreement: 0, incorrectReuse: 0, incorrectAnswers: 0, abstentions: 0, recomputations: 0, reuse: 0, outOfScope: 0 };
  const checks = tasks.map((task, index) => {
    const output = outputs[index];
    object(output, ["task", "decision", "reason", "value", "used"]);
    requireThat(canonicalJSON(output.task) === canonicalJSON(task), "output/task mismatch");
    requireThat(["REUSE", "RECOMPUTE", "ABSTAIN"].includes(output.decision), "invalid decision");
    text(output.reason);
    list(output.used, 32);
    for (const ref of output.used) {
      object(ref, ["id", "contentDigest", "factDigest", "revision"]);
      text(ref.id);
      text(ref.revision);
      for (const key of ["contentDigest", "factDigest"]) requireThat(/^[a-f0-9]{64}$/.test(ref[key]), "invalid used digest");
    }
    requireThat((output.decision === "REUSE") === (output.used.length > 0), "decision/used mismatch");
    if (output.decision === "ABSTAIN") requireThat(output.value === null, "abstention has value");
    else fractionShape(output.value);
    if (!inScope(task)) {
      requireThat(output.decision === "ABSTAIN", "out-of-scope answer");
      metrics.outOfScope += 1;
      metrics.abstentions += 1;
      return { task, expected: null, verdict: "OUT_OF_SCOPE" };
    }
    const expected = fraction(referenceActivityFraction(task.n, rational(task.q)));
    if (output.decision === "ABSTAIN") {
      metrics.abstentions += 1;
      return { task, expected, verdict: "ABSTAIN" };
    }
    if (output.decision === "RECOMPUTE") metrics.recomputations += 1;
    else metrics.reuse += 1;
    const correct = equalFraction(output.value, expected);
    if (correct) metrics.exactAnswerAgreement += 1;
    else {
      metrics.incorrectAnswers += 1;
      if (output.decision === "REUSE") metrics.incorrectReuse += 1;
    }
    return { task, expected, verdict: correct ? "EXACT" : "INCORRECT" };
  });
  return { role: "LOCAL_CROSS_CHECK_NOT_INDEPENDENT_EVIDENCE", checks, metrics,
    costs: { referenceChecks: tasks.filter(inScope).length } };
}

export function compareControls(before, after) {
  snapshotShape(before);
  snapshotShape(after);
  requireThat(canonicalJSON(before.tasks) === canonicalJSON(after.tasks) && before.budget === after.budget, "comparison tasks/budgets differ");
  function phase(snapshot) {
    return Object.fromEntries(VIEWS.map((view) => {
      const use = consumeFacts(snapshot, view);
      return [view, { use, check: checkOutputs(snapshot.tasks, use.outputs) }];
    }));
  }
  const phases = { before: phase(before), after: phase(after) };
  const subtract = (left, right) => Object.fromEntries(Object.keys(left).map((key) => [key, left[key] - right[key]]));
  return { ...phases,
    graphRowsAgree: Object.values(phases).every((item) => canonicalJSON(item.graph.use.outputs) === canonicalJSON(item.rows.use.outputs)),
    afterMinusBefore: Object.fromEntries(VIEWS.map((view) => [view, subtract(phases.after[view].check.metrics, phases.before[view].check.metrics)])),
    graphMinusNode: Object.fromEntries(Object.entries(phases).map(([name, item]) => [name, subtract(item.graph.check.metrics, item.node.check.metrics)])),
    referenceCosts: { referenceChecks: Object.values(phases).reduce((total, item) => total + VIEWS.reduce((sum, view) => sum + item[view].check.costs.referenceChecks, 0), 0) },
    effects: EFFECTS };
}

export function handleRequest(request) {
  canonicalJSON(request);
  const { op, ...input } = request;
  switch (op) {
    case "fixture": object(input, []); return fixtureMaterial();
    case "cached": object(input, ["parent", "tasks"]); return cachedMaterial(input.parent, input.tasks);
    case "consume": {
      const { view, ...snapshot } = input;
      return consumeFacts(snapshot, view);
    }
    case "check": object(input, ["tasks", "outputs"]); return checkOutputs(input.tasks, input.outputs);
    case "compare": object(input, ["before", "after"]); return compareControls(input.before, input.after);
    default: throw new Error("unsupported operation");
  }
}

async function main() {
  try {
    requireThat(process.argv.length === 2, "worker accepts JSON stdin only");
    let length = 0;
    const chunks = [];
    for await (const chunk of process.stdin) {
      length += chunk.length;
      requireThat(length <= MAX_INPUT_BYTES, "input byte limit");
      chunks.push(chunk);
    }
    const request = JSON.parse(new TextDecoder("utf-8", { fatal: true }).decode(Buffer.concat(chunks)));
    const output = JSON.stringify(handleRequest(request));
    requireThat(Buffer.byteLength(output) <= 1048576, "output byte limit");
    process.stdout.write(`${output}\n`);
  } catch (error) {
    process.stdout.write(`${JSON.stringify({ error: "INVALID_REQUEST", message: error instanceof Error ? error.message : "invalid request" })}\n`);
    process.exitCode = 2;
  }
}
if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) await main();
