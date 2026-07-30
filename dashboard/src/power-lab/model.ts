export const POWER_LAB_SCHEMA = "zerone.local-power-reward-lab/v1";

export const POWER_SURFACE_IDS = [
  "consensus-block-ordering",
  "stake-governance",
  "epistemic-review",
  "scorer-registry-authorship",
  "semantic-cluster-adjudication",
  "controller-attestation",
  "proof-data-trust",
  "proposal-authorship",
  "treasury-authorization",
  "reward-flow",
  "model-families",
  "infrastructure",
] as const;

export type PowerSurfaceId = (typeof POWER_SURFACE_IDS)[number];

export const POWER_SURFACE_LABELS: Record<PowerSurfaceId, string> = {
  "consensus-block-ordering": "Consensus & block ordering",
  "stake-governance": "Stake governance",
  "epistemic-review": "Epistemic review",
  "scorer-registry-authorship": "Scorer & registry authorship",
  "semantic-cluster-adjudication": "Semantic-cluster adjudication",
  "controller-attestation": "Controller attestation",
  "proof-data-trust": "Proof & data trust",
  "proposal-authorship": "Proposal authorship",
  "treasury-authorization": "Treasury authorization",
  "reward-flow": "Reward flow",
  "model-families": "Model families",
  infrastructure: "Infrastructure",
};

export const LEDGER_LANE_IDS = [
  "validity",
  "novelty-priority",
  "significance-consequence",
  "attribution-credit",
  "funding-liability",
  "governance-authority",
] as const;

type LedgerLaneId = (typeof LEDGER_LANE_IDS)[number];

export const CAPACITY_SHADOW_EVENTS = [
  "accrue",
  "fund",
  "final-invalidation",
  "reattribute",
] as const;

const CAPACITY_SHADOW_VECTOR = [
  [1, 100, 0, 100, 0, 0, 0, 10],
  [2, 100, 30, 70, 0, 0, 0, 10],
  [3, 100, 30, 0, 60, 10, 0, 10],
  [4, 100, 30, 50, 10, 10, 50, 10],
] as const;

const TREE_BOUNDARY_KEYS = [
  "addsConsensusBehavior",
  "activatesRewards",
  "movesFunds",
  "grantsQualification",
  "authorizesSecurityTesting",
  "assertsProtocolSecurity",
  "performsNetworkRequests",
  "publishesConfidentialEvidence",
] as const;

const MATH_NODE_IDS = [
  "math-algebra-finite-fields@1",
  "math-lattices-polynomial-rings@1",
  "math-probability-information-complexity@1",
  "math-proofcraft@1",
] as const;

const LANE_STATUS: Record<LedgerLaneId, string> = {
  validity: "synthetic-evidence-only",
  "novelty-priority": "preclustered-assumption",
  "significance-consequence": "synthetic-score-input",
  "attribution-credit": "synthetic-controller-credits",
  "funding-liability": "counterfactual-only",
  "governance-authority": "absent",
};

export interface SourcePin {
  generatorVersion: string;
  tree: {
    schema: string;
    policyVersion: string;
    snapshotDate: string;
    sha256: string;
  };
  simulation: {
    sha256: string;
  };
  shadowLedger: {
    sha256: string;
  };
}

export interface MathNode {
  id: string;
  title: string;
  stage: "foundation";
  domain: "mathematics";
  prerequisites: string[];
  rewardEligibility: "qualification-only";
}

export interface PowerSurface {
  id: PowerSurfaceId;
  hhi: number;
  effectiveCount: number;
  nakamotoCount: number;
  largestShare: number;
  coalitionThreshold: number;
  passesIllustrativeFloor: boolean;
}

export interface ShadowCluster {
  id: string;
  artifactCount: number;
  highWater: number;
  lifetimeCap: number;
  newGrossAccrual: number;
  eligibleDemand: number;
  counterfactualFunded: number;
  fundedToDate: number;
  unfundedDemand: number;
  direct: number;
  commons: number;
  canonicalTreeReceipt: null;
  evidenceMilestone: null;
  extinguishedToDate: null;
  eligibilityLots: null;
  settlementZrn: 0;
  settlementState: "blocked";
}

export interface Gate {
  class: "model" | "integration";
  name: string;
  passed: boolean;
  detail: string;
}

export interface CapacityShadowStep {
  event: (typeof CAPACITY_SHADOW_EVENTS)[number];
  epoch: number;
  accrued: number;
  funded: number;
  live: number;
  quarantined: number;
  extinguished: number;
  replacementUsed: number;
  deadline: number;
}

export interface PowerLabFixture {
  schema: typeof POWER_LAB_SCHEMA;
  authoritative: false;
  networkObserved: false;
  rewardBearing: false;
  transferableValue: false;
  chainStateRead: false;
  sources: SourcePin;
  treeBoundary: Record<(typeof TREE_BOUNDARY_KEYS)[number], false>;
  capability: {
    roots: string[];
    nodeCount: number;
    questCount: number;
    mathNodes: MathNode[];
    skillUnlockCreatesReward: false;
    protocolIssuanceGate: string;
  };
  powerStress: {
    kind: "synthetic-captured-policy-stress";
    causallyLinkedToIllustrativeEpoch: false;
    controllerInference: "synthetic-policy-labels";
    minimumEffectiveCount: number;
    minimumNakamotoCount: number;
    surfaces: PowerSurface[];
    history: null;
    uncertainty: null;
    jointPathCut: null;
  };
  shadowEpoch: {
    kind: "synthetic-balanced-policy-allocation";
    unit: "model-unit";
    budget: number;
    direct: number;
    commons: number;
    unallocated: number;
    unfundedDemand: number;
    clusters: ShadowCluster[];
  };
  capacityShadow: {
    schema: "zerone.constructive-capacity-shadow/v1";
    arithmeticVersion: "constructive-shadow-ledger-int-v1";
    unit: "model-unit";
    authoritative: false;
    networkObserved: false;
    rewardBearing: false;
    transferableValue: false;
    movesFunds: false;
    settlementZrn: 0;
    integrationReady: false;
    trace: CapacityShadowStep[];
    checks: {
      exact_partition: true;
      accrued_unchanged_by_reattribute: true;
      funded_unchanged_by_reattribute: true;
      extinguished_unchanged_by_reattribute: true;
      replacement_exposure_within_cap: true;
      replacement_inherited_deadline: true;
      replacement_generation_at_most_one: true;
      passed: true;
    };
  };
  ledgerLanes: Array<{
    id: LedgerLaneId;
    status: string;
    detail: string;
  }>;
  release: {
    settlementEnabled: false;
    claimable: false;
    settlementZrn: 0;
    modelChecksPassed: true;
    integrationReady: false;
    modelPassCount: 12;
    integrationFailCount: 19;
    gates: Gate[];
  };
}

export class PowerLabValidationError extends Error {
  readonly path: string;

  constructor(path: string, message: string) {
    super(`${path}: ${message}`);
    this.name = "PowerLabValidationError";
    this.path = path;
  }
}

type UnknownRecord = Record<string, unknown>;

function record(value: unknown, path: string): UnknownRecord {
  if (
    value === null ||
    typeof value !== "object" ||
    Array.isArray(value) ||
    Object.getPrototypeOf(value) !== Object.prototype
  ) {
    throw new PowerLabValidationError(path, "must be a plain object");
  }
  return value as UnknownRecord;
}

function array(value: unknown, path: string): unknown[] {
  if (!Array.isArray(value)) {
    throw new PowerLabValidationError(path, "must be an array");
  }
  return value;
}

function exactKeys(
  value: UnknownRecord,
  expected: readonly string[],
  path: string,
): void {
  const actual = Object.keys(value).sort();
  const wanted = [...expected].sort();
  if (
    actual.length !== wanted.length ||
    actual.some((key, index) => key !== wanted[index])
  ) {
    throw new PowerLabValidationError(
      path,
      `has keys [${actual.join(", ")}], expected [${wanted.join(", ")}]`,
    );
  }
}

function stringValue(value: unknown, path: string): string {
  if (typeof value !== "string" || value.length === 0) {
    throw new PowerLabValidationError(path, "must be a non-empty string");
  }
  return value;
}

function literal(
  value: unknown,
  expected: string | boolean | number | null,
  path: string,
): void {
  if (value !== expected) {
    throw new PowerLabValidationError(path, `must equal ${String(expected)}`);
  }
}

function finiteNumber(
  value: unknown,
  path: string,
  minimum = 0,
  maximum = Number.MAX_SAFE_INTEGER,
): number {
  if (
    typeof value !== "number" ||
    !Number.isFinite(value) ||
    value < minimum ||
    value > maximum
  ) {
    throw new PowerLabValidationError(
      path,
      `must be finite and in [${minimum}, ${maximum}]`,
    );
  }
  return value;
}

function integer(
  value: unknown,
  path: string,
  minimum = 0,
  maximum = Number.MAX_SAFE_INTEGER,
): number {
  const result = finiteNumber(value, path, minimum, maximum);
  if (!Number.isInteger(result)) {
    throw new PowerLabValidationError(path, "must be an integer");
  }
  return result;
}

function sha256(value: unknown, path: string): void {
  if (
    typeof value !== "string" ||
    !/^[a-f0-9]{64}$/.test(value)
  ) {
    throw new PowerLabValidationError(path, "must be a lowercase SHA-256");
  }
}

function near(a: number, b: number): boolean {
  return Math.abs(a - b) <= 1e-8 * Math.max(1, Math.abs(a), Math.abs(b));
}

function validateSources(value: unknown): void {
  const sources = record(value, "$.sources");
  exactKeys(
    sources,
    ["generatorVersion", "tree", "simulation", "shadowLedger"],
    "$.sources",
  );
  literal(
    sources.generatorVersion,
    "power-lab-projection-v1",
    "$.sources.generatorVersion",
  );

  const tree = record(sources.tree, "$.sources.tree");
  exactKeys(
    tree,
    ["schema", "policyVersion", "snapshotDate", "sha256"],
    "$.sources.tree",
  );
  literal(
    tree.schema,
    "zerone.constructive-intelligence-tree/v1",
    "$.sources.tree.schema",
  );
  literal(tree.policyVersion, "1.0.0", "$.sources.tree.policyVersion");
  if (
    typeof tree.snapshotDate !== "string" ||
    !/^\d{4}-\d{2}-\d{2}$/.test(tree.snapshotDate)
  ) {
    throw new PowerLabValidationError(
      "$.sources.tree.snapshotDate",
      "must be an ISO date",
    );
  }
  sha256(tree.sha256, "$.sources.tree.sha256");

  const simulation = record(sources.simulation, "$.sources.simulation");
  exactKeys(simulation, ["sha256"], "$.sources.simulation");
  sha256(simulation.sha256, "$.sources.simulation.sha256");

  const shadowLedger = record(
    sources.shadowLedger,
    "$.sources.shadowLedger",
  );
  exactKeys(shadowLedger, ["sha256"], "$.sources.shadowLedger");
  sha256(shadowLedger.sha256, "$.sources.shadowLedger.sha256");
}

function validateTreeBoundary(value: unknown): void {
  const boundary = record(value, "$.treeBoundary");
  exactKeys(boundary, TREE_BOUNDARY_KEYS, "$.treeBoundary");
  for (const key of TREE_BOUNDARY_KEYS) {
    literal(boundary[key], false, `$.treeBoundary.${key}`);
  }
}

function validateStringArray(value: unknown, path: string): string[] {
  const values = array(value, path).map((entry, index) =>
    stringValue(entry, `${path}[${index}]`),
  );
  if (new Set(values).size !== values.length) {
    throw new PowerLabValidationError(path, "must not contain duplicates");
  }
  return values;
}

function validateCapability(value: unknown): void {
  const capability = record(value, "$.capability");
  exactKeys(
    capability,
    [
      "roots",
      "nodeCount",
      "questCount",
      "mathNodes",
      "skillUnlockCreatesReward",
      "protocolIssuanceGate",
    ],
    "$.capability",
  );
  const roots = validateStringArray(capability.roots, "$.capability.roots");
  if (roots.length !== 1 || roots[0] !== "math-proofcraft@1") {
    throw new PowerLabValidationError(
      "$.capability.roots",
      "must preserve the canonical proofcraft root",
    );
  }
  if (integer(capability.nodeCount, "$.capability.nodeCount") !== 30) {
    throw new PowerLabValidationError(
      "$.capability.nodeCount",
      "must preserve tree-v1's 30 nodes",
    );
  }
  if (integer(capability.questCount, "$.capability.questCount") !== 3) {
    throw new PowerLabValidationError(
      "$.capability.questCount",
      "must preserve tree-v1's three quests",
    );
  }
  literal(
    capability.skillUnlockCreatesReward,
    false,
    "$.capability.skillUnlockCreatesReward",
  );
  literal(
    capability.protocolIssuanceGate,
    "recursive-useful-work-only",
    "$.capability.protocolIssuanceGate",
  );

  const nodes = array(capability.mathNodes, "$.capability.mathNodes");
  if (nodes.length !== MATH_NODE_IDS.length) {
    throw new PowerLabValidationError(
      "$.capability.mathNodes",
      "must contain exactly four canonical mathematics nodes",
    );
  }
  nodes.forEach((rawNode, index) => {
    const path = `$.capability.mathNodes[${index}]`;
    const node = record(rawNode, path);
    exactKeys(
      node,
      [
        "id",
        "title",
        "stage",
        "domain",
        "prerequisites",
        "rewardEligibility",
      ],
      path,
    );
    literal(node.id, MATH_NODE_IDS[index] ?? "", `${path}.id`);
    stringValue(node.title, `${path}.title`);
    literal(node.stage, "foundation", `${path}.stage`);
    literal(node.domain, "mathematics", `${path}.domain`);
    const prerequisites = validateStringArray(
      node.prerequisites,
      `${path}.prerequisites`,
    );
    for (const prerequisite of prerequisites) {
      if (!MATH_NODE_IDS.includes(prerequisite as (typeof MATH_NODE_IDS)[number])) {
        throw new PowerLabValidationError(
          `${path}.prerequisites`,
          `contains non-mathematics node ${prerequisite}`,
        );
      }
    }
    literal(
      node.rewardEligibility,
      "qualification-only",
      `${path}.rewardEligibility`,
    );
  });
}

function validatePowerStress(value: unknown): void {
  const stress = record(value, "$.powerStress");
  exactKeys(
    stress,
    [
      "kind",
      "causallyLinkedToIllustrativeEpoch",
      "controllerInference",
      "minimumEffectiveCount",
      "minimumNakamotoCount",
      "surfaces",
      "history",
      "uncertainty",
      "jointPathCut",
    ],
    "$.powerStress",
  );
  literal(
    stress.kind,
    "synthetic-captured-policy-stress",
    "$.powerStress.kind",
  );
  literal(
    stress.causallyLinkedToIllustrativeEpoch,
    false,
    "$.powerStress.causallyLinkedToIllustrativeEpoch",
  );
  literal(
    stress.controllerInference,
    "synthetic-policy-labels",
    "$.powerStress.controllerInference",
  );
  const minimumEffective = finiteNumber(
    stress.minimumEffectiveCount,
    "$.powerStress.minimumEffectiveCount",
    Number.MIN_VALUE,
  );
  const minimumNakamoto = integer(
    stress.minimumNakamotoCount,
    "$.powerStress.minimumNakamotoCount",
    1,
  );
  literal(stress.history, null, "$.powerStress.history");
  literal(stress.uncertainty, null, "$.powerStress.uncertainty");
  literal(stress.jointPathCut, null, "$.powerStress.jointPathCut");

  const surfaces = array(stress.surfaces, "$.powerStress.surfaces");
  if (surfaces.length !== POWER_SURFACE_IDS.length) {
    throw new PowerLabValidationError(
      "$.powerStress.surfaces",
      "must contain exactly twelve surfaces",
    );
  }
  surfaces.forEach((rawSurface, index) => {
    const path = `$.powerStress.surfaces[${index}]`;
    const surface = record(rawSurface, path);
    exactKeys(
      surface,
      [
        "id",
        "hhi",
        "effectiveCount",
        "nakamotoCount",
        "largestShare",
        "coalitionThreshold",
        "passesIllustrativeFloor",
      ],
      path,
    );
    literal(surface.id, POWER_SURFACE_IDS[index] ?? "", `${path}.id`);
    const hhi = finiteNumber(surface.hhi, `${path}.hhi`, Number.MIN_VALUE, 1);
    const effective = finiteNumber(
      surface.effectiveCount,
      `${path}.effectiveCount`,
      1,
      256,
    );
    if (!near(hhi * effective, 1)) {
      throw new PowerLabValidationError(
        path,
        "effective count must equal the reciprocal of HHI",
      );
    }
    const nakamoto = integer(
      surface.nakamotoCount,
      `${path}.nakamotoCount`,
      1,
      256,
    );
    finiteNumber(surface.largestShare, `${path}.largestShare`, 0, 1);
    finiteNumber(
      surface.coalitionThreshold,
      `${path}.coalitionThreshold`,
      Number.MIN_VALUE,
      1,
    );
    const expectedPass =
      effective >= minimumEffective && nakamoto >= minimumNakamoto;
    literal(
      surface.passesIllustrativeFloor,
      expectedPass,
      `${path}.passesIllustrativeFloor`,
    );
  });
}

function validateCluster(value: unknown, index: number): ShadowCluster {
  const path = `$.shadowEpoch.clusters[${index}]`;
  const cluster = record(value, path);
  exactKeys(
    cluster,
    [
      "id",
      "artifactCount",
      "highWater",
      "lifetimeCap",
      "newGrossAccrual",
      "eligibleDemand",
      "counterfactualFunded",
      "fundedToDate",
      "unfundedDemand",
      "direct",
      "commons",
      "canonicalTreeReceipt",
      "evidenceMilestone",
      "extinguishedToDate",
      "eligibilityLots",
      "settlementZrn",
      "settlementState",
    ],
    path,
  );
  stringValue(cluster.id, `${path}.id`);
  integer(cluster.artifactCount, `${path}.artifactCount`, 0, 10_000);
  const highWater = finiteNumber(cluster.highWater, `${path}.highWater`, 0, 1);
  const lifetimeCap = finiteNumber(
    cluster.lifetimeCap,
    `${path}.lifetimeCap`,
    Number.MIN_VALUE,
  );
  const gross = finiteNumber(
    cluster.newGrossAccrual,
    `${path}.newGrossAccrual`,
  );
  const demand = finiteNumber(
    cluster.eligibleDemand,
    `${path}.eligibleDemand`,
  );
  const funded = finiteNumber(
    cluster.counterfactualFunded,
    `${path}.counterfactualFunded`,
  );
  const fundedToDate = finiteNumber(
    cluster.fundedToDate,
    `${path}.fundedToDate`,
  );
  const unfunded = finiteNumber(
    cluster.unfundedDemand,
    `${path}.unfundedDemand`,
  );
  const direct = finiteNumber(cluster.direct, `${path}.direct`);
  const commons = finiteNumber(cluster.commons, `${path}.commons`);

  if (!near(highWater * lifetimeCap, gross)) {
    throw new PowerLabValidationError(
      path,
      "first-epoch gross accrual must equal high-water × lifetime cap",
    );
  }
  if (!near(demand - funded, unfunded)) {
    throw new PowerLabValidationError(
      path,
      "unfunded demand must equal eligible demand minus counterfactual allocation",
    );
  }
  if (!near(direct + commons, funded)) {
    throw new PowerLabValidationError(
      path,
      "direct plus commons must conserve the counterfactual allocation",
    );
  }
  if (!near(fundedToDate, funded)) {
    throw new PowerLabValidationError(
      path,
      "the checked-in first epoch must start with no prior funding",
    );
  }
  for (const key of [
    "canonicalTreeReceipt",
    "evidenceMilestone",
    "extinguishedToDate",
    "eligibilityLots",
  ]) {
    literal(cluster[key], null, `${path}.${key}`);
  }
  literal(cluster.settlementZrn, 0, `${path}.settlementZrn`);
  literal(cluster.settlementState, "blocked", `${path}.settlementState`);
  return cluster as unknown as ShadowCluster;
}

function validateShadowEpoch(value: unknown): void {
  const epoch = record(value, "$.shadowEpoch");
  exactKeys(
    epoch,
    [
      "kind",
      "unit",
      "budget",
      "direct",
      "commons",
      "unallocated",
      "unfundedDemand",
      "clusters",
    ],
    "$.shadowEpoch",
  );
  literal(
    epoch.kind,
    "synthetic-balanced-policy-allocation",
    "$.shadowEpoch.kind",
  );
  literal(epoch.unit, "model-unit", "$.shadowEpoch.unit");
  const budget = finiteNumber(
    epoch.budget,
    "$.shadowEpoch.budget",
    Number.MIN_VALUE,
  );
  const direct = finiteNumber(epoch.direct, "$.shadowEpoch.direct");
  const commons = finiteNumber(epoch.commons, "$.shadowEpoch.commons");
  const unallocated = finiteNumber(
    epoch.unallocated,
    "$.shadowEpoch.unallocated",
  );
  const unfunded = finiteNumber(
    epoch.unfundedDemand,
    "$.shadowEpoch.unfundedDemand",
  );
  if (!near(direct + commons + unallocated, budget)) {
    throw new PowerLabValidationError(
      "$.shadowEpoch",
      "direct + commons + unallocated must equal budget",
    );
  }

  const rawClusters = array(epoch.clusters, "$.shadowEpoch.clusters");
  if (rawClusters.length === 0) {
    throw new PowerLabValidationError(
      "$.shadowEpoch.clusters",
      "must contain at least one synthetic cluster",
    );
  }
  const clusters = rawClusters.map(validateCluster);
  const identifiers = clusters.map((cluster) => cluster.id);
  if (
    new Set(identifiers).size !== identifiers.length ||
    identifiers.some(
      (identifier, index) =>
        index > 0 && identifier <= (identifiers[index - 1] ?? ""),
    )
  ) {
    throw new PowerLabValidationError(
      "$.shadowEpoch.clusters",
      "cluster IDs must be unique and sorted",
    );
  }
  const clusterFunded = clusters.reduce(
    (sum, cluster) => sum + cluster.counterfactualFunded,
    0,
  );
  const clusterUnfunded = clusters.reduce(
    (sum, cluster) => sum + cluster.unfundedDemand,
    0,
  );
  const clusterDirect = clusters.reduce(
    (sum, cluster) => sum + cluster.direct,
    0,
  );
  const clusterCommons = clusters.reduce(
    (sum, cluster) => sum + cluster.commons,
    0,
  );
  if (
    !near(clusterFunded, budget - unallocated) ||
    !near(clusterUnfunded, unfunded) ||
    !near(clusterDirect, direct) ||
    !near(clusterCommons, commons)
  ) {
    throw new PowerLabValidationError(
      "$.shadowEpoch.clusters",
      "cluster totals must match epoch totals",
    );
  }
}

function validateCapacityShadow(value: unknown): void {
  const shadow = record(value, "$.capacityShadow");
  exactKeys(
    shadow,
    [
      "schema",
      "arithmeticVersion",
      "unit",
      "authoritative",
      "networkObserved",
      "rewardBearing",
      "transferableValue",
      "movesFunds",
      "settlementZrn",
      "integrationReady",
      "trace",
      "checks",
    ],
    "$.capacityShadow",
  );
  literal(
    shadow.schema,
    "zerone.constructive-capacity-shadow/v1",
    "$.capacityShadow.schema",
  );
  literal(
    shadow.arithmeticVersion,
    "constructive-shadow-ledger-int-v1",
    "$.capacityShadow.arithmeticVersion",
  );
  literal(shadow.unit, "model-unit", "$.capacityShadow.unit");
  for (const key of [
    "authoritative",
    "networkObserved",
    "rewardBearing",
    "transferableValue",
    "movesFunds",
    "integrationReady",
  ]) {
    literal(shadow[key], false, `$.capacityShadow.${key}`);
  }
  literal(shadow.settlementZrn, 0, "$.capacityShadow.settlementZrn");

  const trace = array(shadow.trace, "$.capacityShadow.trace");
  if (trace.length !== CAPACITY_SHADOW_EVENTS.length) {
    throw new PowerLabValidationError(
      "$.capacityShadow.trace",
      "must contain the four exact accounting transitions",
    );
  }
  trace.forEach((rawStep, index) => {
    const path = `$.capacityShadow.trace[${index}]`;
    const step = record(rawStep, path);
    exactKeys(
      step,
      [
        "event",
        "epoch",
        "accrued",
        "funded",
        "live",
        "quarantined",
        "extinguished",
        "replacementUsed",
        "deadline",
      ],
      path,
    );
    literal(step.event, CAPACITY_SHADOW_EVENTS[index] ?? "", `${path}.event`);
    const expected = CAPACITY_SHADOW_VECTOR[index];
    if (expected === undefined) {
      throw new PowerLabValidationError(path, "has no canonical vector row");
    }
    const fields = [
      "epoch",
      "accrued",
      "funded",
      "live",
      "quarantined",
      "extinguished",
      "replacementUsed",
      "deadline",
    ] as const;
    fields.forEach((field, fieldIndex) => {
      integer(step[field], `${path}.${field}`);
      literal(step[field], expected[fieldIndex] ?? -1, `${path}.${field}`);
    });
    if (
      step.accrued !==
      Number(step.funded) +
        Number(step.live) +
        Number(step.quarantined) +
        Number(step.extinguished)
    ) {
      throw new PowerLabValidationError(
        path,
        "must preserve A = Z + L + Q + X exactly",
      );
    }
  });

  const checks = record(shadow.checks, "$.capacityShadow.checks");
  const checkKeys = [
    "exact_partition",
    "accrued_unchanged_by_reattribute",
    "funded_unchanged_by_reattribute",
    "extinguished_unchanged_by_reattribute",
    "replacement_exposure_within_cap",
    "replacement_inherited_deadline",
    "replacement_generation_at_most_one",
    "passed",
  ] as const;
  exactKeys(checks, checkKeys, "$.capacityShadow.checks");
  for (const key of checkKeys) {
    literal(checks[key], true, `$.capacityShadow.checks.${key}`);
  }
}

function validateLedgerLanes(value: unknown): void {
  const lanes = array(value, "$.ledgerLanes");
  if (lanes.length !== LEDGER_LANE_IDS.length) {
    throw new PowerLabValidationError(
      "$.ledgerLanes",
      "must preserve six separate ledgers",
    );
  }
  lanes.forEach((rawLane, index) => {
    const path = `$.ledgerLanes[${index}]`;
    const lane = record(rawLane, path);
    exactKeys(lane, ["id", "status", "detail"], path);
    const expectedId = LEDGER_LANE_IDS[index];
    if (!expectedId) {
      throw new PowerLabValidationError(path, "has no canonical ledger ID");
    }
    literal(lane.id, expectedId, `${path}.id`);
    literal(lane.status, LANE_STATUS[expectedId], `${path}.status`);
    stringValue(lane.detail, `${path}.detail`);
  });
}

function validateRelease(value: unknown): void {
  const release = record(value, "$.release");
  exactKeys(
    release,
    [
      "settlementEnabled",
      "claimable",
      "settlementZrn",
      "modelChecksPassed",
      "integrationReady",
      "modelPassCount",
      "integrationFailCount",
      "gates",
    ],
    "$.release",
  );
  literal(release.settlementEnabled, false, "$.release.settlementEnabled");
  literal(release.claimable, false, "$.release.claimable");
  literal(release.settlementZrn, 0, "$.release.settlementZrn");
  literal(release.modelChecksPassed, true, "$.release.modelChecksPassed");
  literal(release.integrationReady, false, "$.release.integrationReady");
  literal(release.modelPassCount, 12, "$.release.modelPassCount");
  literal(release.integrationFailCount, 19, "$.release.integrationFailCount");

  const gates = array(release.gates, "$.release.gates");
  if (gates.length !== 31) {
    throw new PowerLabValidationError(
      "$.release.gates",
      "must contain 12 model and 19 integration gates",
    );
  }
  const names = new Set<string>();
  let modelCount = 0;
  let integrationCount = 0;
  gates.forEach((rawGate, index) => {
    const path = `$.release.gates[${index}]`;
    const gate = record(rawGate, path);
    exactKeys(gate, ["class", "name", "passed", "detail"], path);
    const gateClass = stringValue(gate.class, `${path}.class`);
    const name = stringValue(gate.name, `${path}.name`);
    stringValue(gate.detail, `${path}.detail`);
    if (names.has(name)) {
      throw new PowerLabValidationError(`${path}.name`, "must be unique");
    }
    names.add(name);
    if (gateClass === "model") {
      if (integrationCount > 0) {
        throw new PowerLabValidationError(
          `${path}.class`,
          "model gates must precede integration gates",
        );
      }
      literal(gate.passed, true, `${path}.passed`);
      modelCount += 1;
    } else if (gateClass === "integration") {
      literal(gate.passed, false, `${path}.passed`);
      integrationCount += 1;
    } else {
      throw new PowerLabValidationError(
        `${path}.class`,
        "must be model or integration",
      );
    }
  });
  if (modelCount !== 12 || integrationCount !== 19) {
    throw new PowerLabValidationError(
      "$.release.gates",
      "must contain exactly 12 passing model and 19 closed integration gates",
    );
  }
}

export function parsePowerLabFixture(value: unknown): PowerLabFixture {
  const fixture = record(value, "$");
  exactKeys(
    fixture,
    [
      "schema",
      "authoritative",
      "networkObserved",
      "rewardBearing",
      "transferableValue",
      "chainStateRead",
      "sources",
      "treeBoundary",
      "capability",
      "powerStress",
      "shadowEpoch",
      "capacityShadow",
      "ledgerLanes",
      "release",
    ],
    "$",
  );
  literal(fixture.schema, POWER_LAB_SCHEMA, "$.schema");
  for (const key of [
    "authoritative",
    "networkObserved",
    "rewardBearing",
    "transferableValue",
    "chainStateRead",
  ]) {
    literal(fixture[key], false, `$.${key}`);
  }
  validateSources(fixture.sources);
  validateTreeBoundary(fixture.treeBoundary);
  validateCapability(fixture.capability);
  validatePowerStress(fixture.powerStress);
  validateShadowEpoch(fixture.shadowEpoch);
  validateCapacityShadow(fixture.capacityShadow);
  validateLedgerLanes(fixture.ledgerLanes);
  validateRelease(fixture.release);
  return fixture as unknown as PowerLabFixture;
}

const units = new Intl.NumberFormat("en-GB", {
  minimumFractionDigits: 0,
  maximumFractionDigits: 3,
});

export function formatModelUnits(value: number): string {
  return `${units.format(value)} MU`;
}

export function formatPercent(value: number): string {
  return new Intl.NumberFormat("en-GB", {
    style: "percent",
    maximumFractionDigits: 1,
  }).format(value);
}

export function mathNodeLevels(nodes: MathNode[]): MathNode[][] {
  const byId = new Map(nodes.map((node) => [node.id, node]));
  const depths = new Map<string, number>();
  const visit = (id: string, visiting = new Set<string>()): number => {
    const known = depths.get(id);
    if (known !== undefined) return known;
    if (visiting.has(id)) {
      throw new PowerLabValidationError(
        "$.capability.mathNodes",
        "contains a prerequisite cycle",
      );
    }
    const node = byId.get(id);
    if (!node) {
      throw new PowerLabValidationError(
        "$.capability.mathNodes",
        `is missing ${id}`,
      );
    }
    const nextVisiting = new Set(visiting);
    nextVisiting.add(id);
    const depth =
      node.prerequisites.length === 0
        ? 0
        : 1 +
          Math.max(
            ...node.prerequisites.map((prerequisite) =>
              visit(prerequisite, nextVisiting),
            ),
          );
    depths.set(id, depth);
    return depth;
  };
  for (const node of nodes) visit(node.id);
  const maximum = Math.max(...depths.values());
  return Array.from({ length: maximum + 1 }, (_, depth) =>
    nodes.filter((node) => depths.get(node.id) === depth),
  );
}
