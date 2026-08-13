import { createHash } from "node:crypto";
import { existsSync, lstatSync, readFileSync, realpathSync } from "node:fs";
import { dirname, isAbsolute, relative, resolve, sep } from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";

// Offline by construction: source URLs are exact locators and are never fetched.
export const FOLD_TO_FIRE_SCHEMA =
  "zerone.constructive-intelligence-fold-to-fire/v0";
export const FOLD_TO_FIRE_MAX_BYTES = 65_536;
export const FOLD_TO_FIRE_RAW_SHA256 =
  "3fb78beaec220b4f62219a120ea33f46cfbe5ca1e76286929ae7b1120ccf4033";
export const FOLD_TO_FIRE_CANONICAL_SHA256 =
  "545c14c655494886b502f4c81eb1b71a99caec297f063a2097cded9dad3b893b";

const SCRIPT_PATH = fileURLToPath(import.meta.url);
const REPOSITORY_ROOT = resolve(dirname(SCRIPT_PATH), "../..");
const REPOSITORY_ROOT_REAL = realpathSync(REPOSITORY_ROOT);
const MAX_JSON_NESTING = 16;
const HEX_SHA256 = /^[0-9a-f]{64}$/;
const DECIMAL = /^(?:0|[1-9][0-9]*)$/;
const FRACTION = /^(?:0|[1-9][0-9]*)\/(?:[1-9][0-9]*)$/;
const SAFE_PATH = /^(?!\/)(?!.*(?:^|\/)\.\.(?:\/|$))[A-Za-z0-9._/-]+$/;

const TOP_LEVEL_KEYS = [
  "schema",
  "status",
  "mode",
  "title",
  "snapshotDate",
  "authoritative",
  "networkObserved",
  "sourceBindings",
  "releaseBoundary",
  "economics",
  "modelBoundary",
  "enumeration",
  "frontierProblem",
  "weightedBridge",
  "nonImplicationWalls",
  "sources",
];
const SOURCE_BINDING_KEYS = [
  "id",
  "path",
  "schema",
  "rawSha256",
  "canonicalSha256",
  "boundary",
];
const RELEASE_BOUNDARY_KEYS = [
  "changesConsensus",
  "writesNetworkState",
  "performsNetworkRequests",
  "activatesRewards",
  "movesFunds",
  "createsKarmaEvent",
  "createsKarmaMagnitude",
  "grantsQualification",
  "grantsGovernance",
  "grantsAuthority",
  "ranksPersons",
  "authorizesExperiment",
  "authorizesWetLab",
  "authorizesClinicalDecision",
  "containsSequencePayload",
  "claimsProteinPrediction",
  "claimsCatalystDesign",
];
const ECONOMICS_KEYS = [
  "effect",
  "amount",
  "denom",
  "rewardMultiplier",
  "escrowReference",
];
const MODEL_BOUNDARY_KEYS = [
  "lattice",
  "walkConvention",
  "contactDefinition",
  "activeDefinition",
  "rotationTreatment",
  "reflectionTreatment",
  "minimumActiveSteps",
  "proteinInterpretation",
];
const ENUMERATION_KEYS = [
  "solver",
  "maximumExactSteps",
  "evidenceRole",
  "rows",
];
const ROW_KEYS = [
  "stepCount",
  "allByContacts",
  "activeByContacts",
  "totalWalks",
  "activeWalks",
  "closingFraction",
];
const FRONTIER_KEYS = [
  "status",
  "domain",
  "statement",
  "exponent",
  "quantifier",
  "interpretation",
  "evidenceStatus",
  "computationDoesNotProve",
  "weightedBridgeDoesNotEstablish",
];
const WEIGHTED_KEYS = [
  "status",
  "domain",
  "qDomain",
  "partitionFunction",
  "activePartitionFunction",
  "effectiveFlux",
  "interpretation",
  "qNotEqualOneExponentTransfer",
  "noveltyAuditRequired",
  "notClaimedAsEstablishedOpenProblem",
];
const WALL_KEYS = ["id", "premise", "doesNotEstablish"];
const SOURCE_KEYS = ["id", "title", "url", "role"];

const SOURCE_BINDINGS = Object.freeze([
  Object.freeze({
    id: "constructive-intelligence-math-frontier-v0",
    path: "dashboard/public/standards/constructive-intelligence-math-frontier.v0.json",
    schema: "zerone.constructive-intelligence-math-frontier/v0",
    rawSha256:
      "4fdcd54c35c69c26a28c385275688351ee2a9131702e81bacf100de8d7612456",
    canonicalSha256:
      "b6260de31969a56e601a5a81f1b4f7c1c68fcd34f9aa68b35ce2701c3f012503",
    boundary:
      "Pins curriculum and proof-boundary bytes only; imports no quest, reward template, qualification, governance, or KARMA effect.",
  }),
  Object.freeze({
    id: "constructive-intelligence-life-sciences-v0",
    path: "dashboard/public/standards/constructive-intelligence-life-sciences.v0.json",
    schema: "zerone.constructive-intelligence-life-sciences/v0",
    rawSha256:
      "64dc2c5b2e21dfc9697d173317254ce651dede8661993ece7b380b7e1421496e",
    canonicalSha256:
      "a208ea9e30a16ccfbb74f3f19298a5d3f93d7f87273b0b5aa10bf72e0e708822",
    boundary:
      "Pins scientific claim-boundary bytes only; imports no sequence, wet-lab, clinical, qualification, reward, governance, or network authority.",
  }),
  Object.freeze({
    id: "money-karma-v1",
    path: "docs/constitution/money-karma-v1.json",
    schema: "zerone.money-karma.constitution/v1",
    rawSha256:
      "f22e62f0706971c569bb2156400b6dbeaf72a005d822b1e40c4e2691e7a98c24",
    canonicalSha256:
      "a41286c936d3ab83d1cbd782b119cf3b434518ba80859edfe76f0de184143b7b",
    boundary:
      "Pins constitutional separation bytes only; creates no KARMA event, magnitude, balance, reward, money, qualification, governance weight, or authority.",
  }),
]);

const EXPECTED_ROWS = Object.freeze([
  Object.freeze({
    stepCount: 3,
    allByContacts: ["7", "2"],
    activeByContacts: ["0", "2"],
    totalWalks: "9",
    activeWalks: "2",
    closingFraction: "2/9",
  }),
  Object.freeze({
    stepCount: 5,
    allByContacts: ["41", "22", "8"],
    activeByContacts: ["0", "0", "6"],
    totalWalks: "71",
    activeWalks: "6",
    closingFraction: "6/71",
  }),
  Object.freeze({
    stepCount: 7,
    allByContacts: ["235", "184", "86", "38"],
    activeByContacts: ["0", "4", "0", "24"],
    totalWalks: "543",
    activeWalks: "28",
    closingFraction: "28/543",
  }),
  Object.freeze({
    stepCount: 9,
    allByContacts: ["1331", "1344", "850", "346", "196"],
    activeByContacts: ["0", "10", "40", "0", "90"],
    totalWalks: "4067",
    activeWalks: "140",
    closingFraction: "20/581",
  }),
  Object.freeze({
    stepCount: 11,
    allByContacts: ["7485", "9244", "6900", "3888", "1606", "888", "62"],
    activeByContacts: ["0", "54", "120", "240", "0", "306", "24"],
    totalWalks: "30073",
    activeWalks: "744",
    closingFraction: "744/30073",
  }),
  Object.freeze({
    stepCount: 13,
    allByContacts: ["41867", "60884", "52934", "33472", "19076", "7444", "3978", "720"],
    activeByContacts: ["0", "252", "672", "770", "1092", "112", "966", "252"],
    totalWalks: "220375",
    activeWalks: "4116",
    closingFraction: "4116/220375",
  }),
  Object.freeze({
    stepCount: 15,
    allByContacts: ["233157", "389792", "383628", "276892", "169214", "91128", "37466", "17324", "5410", "138"],
    activeByContacts: ["0", "1232", "3264", "4496", "3904", "4672", "1408", "2976", "1504", "48"],
    totalWalks: "1604149",
    activeWalks: "23504",
    closingFraction: "23504/1604149",
  }),
]);

const EXPECTED_WALLS = Object.freeze([
  Object.freeze({
    id: "FINITE_ENUMERATION_NOT_ASYMPTOTIC_PROOF",
    premise: "EXACT_FINITE_ENUMERATION",
    doesNotEstablish: ["ASYMPTOTIC_EXPONENT", "CONJECTURE_PROOF"],
  }),
  Object.freeze({
    id: "UNWEIGHTED_EXPONENT_NOT_WEIGHTED_EXPONENT",
    premise: "Q_EQUALS_ONE_CLOSING_CONJECTURE",
    doesNotEstablish: ["PROTEIN_FOLDING_SCALING_LAW", "Q_NOT_EQUAL_ONE_EXPONENT"],
  }),
  Object.freeze({
    id: "LATTICE_POLYMER_NOT_ATOMIC_PROTEIN",
    premise: "SQUARE_LATTICE_SELF_AVOIDING_WALK",
    doesNotEstablish: ["ATOMIC_PROTEIN_FOLD", "BIOLOGICAL_FUNCTION", "FOLDING_KINETICS", "FOLDING_PATHWAY"],
  }),
  Object.freeze({
    id: "ACTIVE_CONTACT_NOT_ENZYME_CATALYSIS",
    premise: "ENDPOINT_ADJACENCY",
    doesNotEstablish: ["ACTIVE_SITE_GEOMETRY", "BINDING", "CATALYTIC_RATE", "TRANSITION_STATE_STABILIZATION"],
  }),
  Object.freeze({
    id: "GEOMETRY_FACTOR_NOT_CHEMICAL_RATE",
    premise: "CATALYTIC_COMPETENCE_FRACTION",
    doesNotEstablish: ["EQUILIBRIUM_CONSTANT", "KAPPA", "REACTION_RATE"],
  }),
  Object.freeze({
    id: "FOLDING_NOT_FOLDING_CATALYST",
    premise: "CLIENT_FOLDING",
    doesNotEstablish: ["ATP_CHAPERONE_NONEQUILIBRIUM_ACTIVITY", "PDI_ACTIVITY", "PPIASE_ACTIVITY"],
  }),
  Object.freeze({
    id: "SCIENCE_NOT_KARMA_AUTHORITY",
    premise: "MATHEMATICAL_OR_BIOPHYSICAL_RESULT",
    doesNotEstablish: ["AUTHORITY", "GOVERNANCE", "KARMA_EVENT", "KARMA_MAGNITUDE", "MONEY", "PERSON_WORTH", "QUALIFICATION", "REWARD"],
  }),
]);

const EXPECTED_SOURCES = Object.freeze([
  ["saw-closing-probability", "On the probability that self-avoiding walk ends at a given point", "https://doi.org/10.1214/14-AOP993", "States the planar n^{-59/32+o(1)} closing prediction and proves weaker rigorous endpoint-delocalization bounds."],
  ["saw-counting-joining-closing", "On self-avoiding polygons and walks: counting, joining and closing", "https://arxiv.org/abs/1504.05286", "Gives the exact closing-probability identity in terms of polygon and walk counts and stronger rigorous upper bounds than finite enumeration supplies."],
  ["saw-square-enumeration", "Enumeration of self-avoiding walks on the square lattice", "https://doi.org/10.1088/0305-4470/37/21/002", "Provides established exact-enumeration context for square-lattice self-avoiding-walk counts."],
  ["saw-critical-exponents-open", "On the existence of critical exponents for self-avoiding walks", "https://doi.org/10.1088/1751-8121/ac943a", "Provides contemporary evidence and open-problem context for rigorous two-dimensional self-avoiding-walk critical exponents."],
  ["interacting-saw-precedent", "Exact enumeration study of free energies of interacting polygons and walks in two dimensions", "https://doi.org/10.1088/0305-4470/31/20/010", "Establishes interacting contact-weighted self-avoiding-walk precedent; this profile claims no novelty for contact weighting itself."],
  ["folding-chemical-landscapes", "On the relationship between folding and chemical landscapes in enzyme catalysis", "https://doi.org/10.1073/pnas.0803405105", "Motivates keeping conformational availability separate from the chemical activation barrier."],
  ["prolyl-isomerase-folding", "Catalysis of protein folding by prolyl isomerase", "https://doi.org/10.1038/329268a0", "Documents PPIase acceleration of otherwise slow proline-isomerization-limited refolding steps."],
  ["disulfide-isomerase-folding", "Efficient catalysis of disulphide bond rearrangements by protein disulphide isomerase", "https://doi.org/10.1038/365185a0", "Documents PDI acceleration of disulfide rearrangement in kinetically trapped folding intermediates."],
  ["atp-chaperone-nonequilibrium", "Chaperones convert the energy from ATP into the nonequilibrium stabilization of native proteins", "https://doi.org/10.1038/s41589-018-0013-8", "Separates ATP-driven chaperone machines from equilibrium catalyst shorthand."],
].map(([id, title, url, role]) => Object.freeze({ id, title, url, role })));

export class FoldToFireValidationError extends Error {
  constructor(path, message) {
    super(`${path}: ${message}`);
    this.name = "FoldToFireValidationError";
    this.path = path;
  }
}

function fail(path, message) {
  throw new FoldToFireValidationError(path, message);
}

function record(value, path) {
  if (typeof value !== "object" || value === null || Array.isArray(value)) {
    fail(path, "must be an object");
  }
  const prototype = Object.getPrototypeOf(value);
  if (prototype !== Object.prototype && prototype !== null) {
    fail(path, "must be a plain object");
  }
  return value;
}

function exactKeys(value, expected, path) {
  const actual = Object.keys(value);
  if (
    actual.length !== expected.length ||
    actual.some((key, index) => key !== expected[index])
  ) {
    fail(path, `must contain the exact v0 fields in order: ${expected.join(", ")}`);
  }
}

function exact(value, expected, path) {
  if (value !== expected) fail(path, `must equal ${String(expected)}`);
}

function exactJson(value, expected, path) {
  if (JSON.stringify(value) !== JSON.stringify(expected)) {
    fail(path, "must equal the reviewed v0 value");
  }
}

function falseOnly(value, path) {
  exact(value, false, path);
}

function trueOnly(value, path) {
  exact(value, true, path);
}

function sha256(value) {
  return createHash("sha256").update(value).digest("hex");
}

export function canonicalJson(value) {
  if (value === null || typeof value !== "object") return JSON.stringify(value);
  if (Array.isArray(value)) return `[${value.map(canonicalJson).join(",")}]`;
  return `{${Object.keys(value)
    .sort()
    .map((key) => `${JSON.stringify(key)}:${canonicalJson(value[key])}`)
    .join(",")}}`;
}

function readSourceBinding(binding, expected, index) {
  const path = `$.sourceBindings[${index}]`;
  record(binding, path);
  exactKeys(binding, SOURCE_BINDING_KEYS, path);
  for (const key of SOURCE_BINDING_KEYS) {
    exact(binding[key], expected[key], `${path}.${key}`);
  }
  if (!HEX_SHA256.test(binding.rawSha256) || !HEX_SHA256.test(binding.canonicalSha256)) {
    fail(path, "digests must be lowercase SHA-256 values");
  }
  if (!SAFE_PATH.test(binding.path)) fail(`${path}.path`, "must be a safe repository path");
  const absolute = resolve(REPOSITORY_ROOT, binding.path);
  if (!existsSync(absolute)) fail(`${path}.path`, "must exist");
  const stat = lstatSync(absolute);
  if (stat.isSymbolicLink() || !stat.isFile()) {
    fail(`${path}.path`, "must resolve directly to a regular non-symlink file");
  }
  const real = realpathSync(absolute);
  const contained = relative(REPOSITORY_ROOT_REAL, real);
  if (
    contained === "" ||
    isAbsolute(contained) ||
    contained === ".." ||
    contained.startsWith(`..${sep}`)
  ) {
    fail(`${path}.path`, "must remain inside the repository");
  }
  const raw = readFileSync(real);
  exact(sha256(raw), binding.rawSha256, `${path}.rawSha256`);
  let parsed;
  try {
    parsed = JSON.parse(raw);
  } catch {
    fail(`${path}.path`, "must contain JSON");
  }
  exact(sha256(canonicalJson(parsed)), binding.canonicalSha256, `${path}.canonicalSha256`);
  exact(parsed.schema, binding.schema, `${path}.schema`);
}

function greatestCommonDivisor(left, right) {
  let a = left < 0n ? -left : left;
  let b = right < 0n ? -right : right;
  while (b !== 0n) [a, b] = [b, a % b];
  return a;
}

function validateRow(row, expected, index) {
  const path = `$.enumeration.rows[${index}]`;
  record(row, path);
  exactKeys(row, ROW_KEYS, path);
  exactJson(row, expected, path);
  for (const [key, values] of [
    ["allByContacts", row.allByContacts],
    ["activeByContacts", row.activeByContacts],
  ]) {
    if (!Array.isArray(values) || values.length === 0) fail(`${path}.${key}`, "must be nonempty");
    values.forEach((value, valueIndex) => {
      if (!DECIMAL.test(value)) fail(`${path}.${key}[${valueIndex}]`, "must be a canonical decimal string");
    });
  }
  const total = row.allByContacts.reduce((sum, value) => sum + BigInt(value), 0n);
  const active = row.activeByContacts.reduce((sum, value) => sum + BigInt(value), 0n);
  exact(total.toString(), row.totalWalks, `${path}.totalWalks`);
  exact(active.toString(), row.activeWalks, `${path}.activeWalks`);
  if (!FRACTION.test(row.closingFraction)) fail(`${path}.closingFraction`, "must be a reduced positive fraction");
  const [numerator, denominator] = row.closingFraction.split("/").map(BigInt);
  const divisor = greatestCommonDivisor(active, total);
  exact(numerator.toString(), (active / divisor).toString(), `${path}.closingFraction`);
  exact(denominator.toString(), (total / divisor).toString(), `${path}.closingFraction`);
}

function validateProfile(profile) {
  record(profile, "$");
  exactKeys(profile, TOP_LEVEL_KEYS, "$");
  exact(profile.schema, FOLD_TO_FIRE_SCHEMA, "$.schema");
  exact(profile.status, "SEALED_STATIC_PROFILE", "$.status");
  exact(profile.mode, "READ_ONLY_ZERO_EFFECT", "$.mode");
  exact(profile.title, "Fold-to-Fire: exact finite polymer geometry and a bounded catalytic bridge", "$.title");
  exact(profile.snapshotDate, "2026-08-13", "$.snapshotDate");
  falseOnly(profile.authoritative, "$.authoritative");
  falseOnly(profile.networkObserved, "$.networkObserved");

  if (!Array.isArray(profile.sourceBindings) || profile.sourceBindings.length !== SOURCE_BINDINGS.length) {
    fail("$.sourceBindings", "must contain the three exact immutable source bindings");
  }
  profile.sourceBindings.forEach((binding, index) =>
    readSourceBinding(binding, SOURCE_BINDINGS[index], index),
  );

  record(profile.releaseBoundary, "$.releaseBoundary");
  exactKeys(profile.releaseBoundary, RELEASE_BOUNDARY_KEYS, "$.releaseBoundary");
  for (const key of RELEASE_BOUNDARY_KEYS) {
    falseOnly(profile.releaseBoundary[key], `$.releaseBoundary.${key}`);
  }
  record(profile.economics, "$.economics");
  exactKeys(profile.economics, ECONOMICS_KEYS, "$.economics");
  exactJson(profile.economics, {
    effect: "NONE",
    amount: "0",
    denom: null,
    rewardMultiplier: false,
    escrowReference: null,
  }, "$.economics");

  record(profile.modelBoundary, "$.modelBoundary");
  exactKeys(profile.modelBoundary, MODEL_BOUNDARY_KEYS, "$.modelBoundary");
  exactJson(profile.modelBoundary, {
    lattice: "Z2_SQUARE_NEAREST_NEIGHBOUR",
    walkConvention: "v0=(0,0);v1=(1,0);reflections-distinct",
    contactDefinition: "A nearest-neighbour vertex pair whose sequence indices differ by at least two.",
    activeDefinition: "For n at least 3, the final vertex is a nearest neighbour of the origin.",
    rotationTreatment: "ROTATIONS_QUOTIENTED_BY_FIXING_FIRST_STEP_EAST",
    reflectionTreatment: "REFLECTIONS_REMAIN_DISTINCT",
    minimumActiveSteps: 3,
    proteinInterpretation: "ABSTRACT_POLYMER_GEOMETRY_ANALOGY_NOT_ATOMIC_PROTEIN_MODEL",
  }, "$.modelBoundary");

  record(profile.enumeration, "$.enumeration");
  exactKeys(profile.enumeration, ENUMERATION_KEYS, "$.enumeration");
  exact(profile.enumeration.solver, "fold-to-fire-exact-dfs/v0", "$.enumeration.solver");
  exact(profile.enumeration.maximumExactSteps, 15, "$.enumeration.maximumExactSteps");
  exact(profile.enumeration.evidenceRole, "EXACT_FINITE_ENUMERATION_NOT_ASYMPTOTIC_PROOF", "$.enumeration.evidenceRole");
  if (!Array.isArray(profile.enumeration.rows) || profile.enumeration.rows.length !== EXPECTED_ROWS.length) {
    fail("$.enumeration.rows", "must contain exact odd-step rows from n=3 through n=15");
  }
  profile.enumeration.rows.forEach((row, index) => validateRow(row, EXPECTED_ROWS[index], index));

  record(profile.frontierProblem, "$.frontierProblem");
  exactKeys(profile.frontierProblem, FRONTIER_KEYS, "$.frontierProblem");
  exactJson(profile.frontierProblem, {
    status: "ESTABLISHED_OPEN_CONJECTURE",
    domain: "UNWEIGHTED_SQUARE_LATTICE_SELF_AVOIDING_WALK",
    statement: "A_n(1)/Z_n(1)=n^{-59/32+o(1)} along odd n as n tends to infinity.",
    exponent: "59/32",
    quantifier: "ODD_N_TO_INFINITY",
    interpretation: "UNIFORM_CLOSING_PROBABILITY",
    evidenceStatus: "LITERATURE_CONJECTURE_NOT_PROVED",
    computationDoesNotProve: true,
    weightedBridgeDoesNotEstablish: true,
  }, "$.frontierProblem");

  record(profile.weightedBridge, "$.weightedBridge");
  exactKeys(profile.weightedBridge, WEIGHTED_KEYS, "$.weightedBridge");
  exactJson(profile.weightedBridge, {
    status: "BESPOKE_RESEARCH_BRIDGE",
    domain: "POSITIVE_CONTACT_WEIGHTED_FINITE_SELF_AVOIDING_WALK",
    qDomain: "q>0",
    partitionFunction: "Z_n(q)=sum_omega q^{C(omega)}",
    activePartitionFunction: "A_n(q)=sum_{omega in Active_n} q^{C(omega)}",
    effectiveFlux: "J_n(q)=kappa*A_n(q)/Z_n(q)",
    interpretation: "GEOMETRIC_AVAILABILITY_TIMES_DECLARED_CHEMICAL_RATE",
    qNotEqualOneExponentTransfer: "OUT_OF_SCOPE_NOT_CLAIMED",
    noveltyAuditRequired: true,
    notClaimedAsEstablishedOpenProblem: true,
  }, "$.weightedBridge");

  if (!Array.isArray(profile.nonImplicationWalls)) fail("$.nonImplicationWalls", "must be an array");
  exactJson(profile.nonImplicationWalls, EXPECTED_WALLS, "$.nonImplicationWalls");
  profile.nonImplicationWalls.forEach((wall, index) => {
    record(wall, `$.nonImplicationWalls[${index}]`);
    exactKeys(wall, WALL_KEYS, `$.nonImplicationWalls[${index}]`);
  });
  if (!Array.isArray(profile.sources)) fail("$.sources", "must be an array");
  exactJson(profile.sources, EXPECTED_SOURCES, "$.sources");
  profile.sources.forEach((source, index) => {
    record(source, `$.sources[${index}]`);
    exactKeys(source, SOURCE_KEYS, `$.sources[${index}]`);
  });

  const canonicalSha256 = sha256(canonicalJson(profile));
  exact(canonicalSha256, FOLD_TO_FIRE_CANONICAL_SHA256, "$");
  return Object.freeze({
    schema: FOLD_TO_FIRE_SCHEMA,
    rowCount: profile.enumeration.rows.length,
    maximumExactSteps: profile.enumeration.maximumExactSteps,
    sourceCount: profile.sources.length,
    sourceBindingCount: profile.sourceBindings.length,
    canonicalSha256,
  });
}

function rejectExcessiveJsonNesting(raw, label) {
  let depth = 0;
  let inString = false;
  let escaped = false;
  for (const character of raw) {
    if (inString) {
      if (escaped) escaped = false;
      else if (character === "\\") escaped = true;
      else if (character === '"') inString = false;
      continue;
    }
    if (character === '"') inString = true;
    else if (character === "{" || character === "[") {
      depth += 1;
      if (depth > MAX_JSON_NESTING) fail(label, `JSON nesting exceeds ${MAX_JSON_NESTING}`);
    } else if (character === "}" || character === "]") depth -= 1;
  }
}

function rejectDuplicateJsonKeys(raw, label) {
  let offset = 0;
  const whitespace = () => {
    while (/\s/.test(raw[offset] ?? "")) offset += 1;
  };
  const scanString = () => {
    const start = offset;
    offset += 1;
    while (offset < raw.length) {
      if (raw[offset] === "\\") offset += 2;
      else if (raw[offset] === '"') {
        offset += 1;
        return JSON.parse(raw.slice(start, offset));
      } else offset += 1;
    }
    fail(label, "unterminated JSON string");
  };
  const scanValue = (path) => {
    whitespace();
    const token = raw[offset];
    if (token === "{") {
      offset += 1;
      whitespace();
      const keys = new Set();
      if (raw[offset] === "}") {
        offset += 1;
        return;
      }
      while (offset < raw.length) {
        whitespace();
        if (raw[offset] !== '"') fail(path, "malformed object key");
        const key = scanString();
        if (keys.has(key)) fail(`${path}.${key}`, "duplicate JSON object key");
        keys.add(key);
        whitespace();
        if (raw[offset] !== ":") fail(path, "malformed object separator");
        offset += 1;
        scanValue(`${path}.${key}`);
        whitespace();
        if (raw[offset] === "}") {
          offset += 1;
          return;
        }
        if (raw[offset] !== ",") fail(path, "malformed object delimiter");
        offset += 1;
      }
      fail(path, "unterminated object");
    }
    if (token === "[") {
      offset += 1;
      whitespace();
      if (raw[offset] === "]") {
        offset += 1;
        return;
      }
      let index = 0;
      while (offset < raw.length) {
        scanValue(`${path}[${index}]`);
        whitespace();
        if (raw[offset] === "]") {
          offset += 1;
          return;
        }
        if (raw[offset] !== ",") fail(path, "malformed array delimiter");
        offset += 1;
        index += 1;
      }
      fail(path, "unterminated array");
    }
    if (token === '"') {
      scanString();
      return;
    }
    const start = offset;
    while (offset < raw.length && !/[\s,\]}]/.test(raw[offset] ?? "")) offset += 1;
    if (start === offset) fail(path, "malformed JSON value");
  };
  scanValue("$");
  whitespace();
  if (offset !== raw.length) fail(label, "contains trailing JSON data");
}

function parseRaw(raw) {
  if (typeof raw !== "string") fail("$", "must be a JSON string");
  if (Buffer.byteLength(raw, "utf8") > FOLD_TO_FIRE_MAX_BYTES) {
    fail("$", `exceeds ${FOLD_TO_FIRE_MAX_BYTES} UTF-8 bytes`);
  }
  rejectExcessiveJsonNesting(raw, "$");
  let value;
  try {
    value = JSON.parse(raw);
  } catch (error) {
    fail("$", `invalid JSON: ${error.message}`);
  }
  rejectDuplicateJsonKeys(raw, "$");
  return value;
}

export function parseAndValidateConstructiveIntelligenceFoldToFire(
  raw,
  { pinRawDigest = true } = {},
) {
  const profile = parseRaw(raw);
  const rawSha256 = sha256(raw);
  if (pinRawDigest) exact(rawSha256, FOLD_TO_FIRE_RAW_SHA256, "$");
  return Object.freeze({ ...validateProfile(profile), rawSha256 });
}

export function validateConstructiveIntelligenceFoldToFire(profile) {
  let raw;
  try {
    raw = JSON.stringify(profile);
  } catch (error) {
    fail("$", `must be JSON-serializable: ${error.message}`);
  }
  return parseAndValidateConstructiveIntelligenceFoldToFire(raw, {
    pinRawDigest: false,
  });
}

function readInputFile(inputPath) {
  const absolute = resolve(inputPath);
  if (!existsSync(absolute)) fail("$input", "file does not exist");
  const stat = lstatSync(absolute);
  if (stat.isSymbolicLink() || !stat.isFile()) {
    fail("$input", "must be a regular non-symlink file");
  }
  if (stat.size > FOLD_TO_FIRE_MAX_BYTES) {
    fail("$input", `exceeds ${FOLD_TO_FIRE_MAX_BYTES} bytes`);
  }
  return readFileSync(absolute, "utf8");
}

function runCli() {
  if (process.argv.length !== 3) {
    console.error(
      "usage: node scripts/validate-constructive-intelligence-fold-to-fire.mjs PROFILE_JSON",
    );
    process.exitCode = 2;
    return;
  }
  try {
    const result = parseAndValidateConstructiveIntelligenceFoldToFire(
      readInputFile(process.argv[2]),
    );
    console.log(
      `constructive-intelligence Fold-to-Fire: PASS (${result.rowCount} exact finite rows through n=${result.maximumExactSteps}; ${result.sourceCount} exact source locators; zero effect; profile sha256 ${result.rawSha256})`,
    );
  } catch (error) {
    console.error(`constructive-intelligence Fold-to-Fire: FAIL: ${error.message}`);
    process.exitCode = 1;
  }
}

if (
  process.argv[1] &&
  import.meta.url === pathToFileURL(resolve(process.argv[1])).href
) {
  runCli();
}
