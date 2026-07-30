import { createHash } from "node:crypto";
import {
  existsSync,
  readFileSync,
  writeFileSync,
} from "node:fs";
import { dirname, resolve } from "node:path";
import { spawnSync } from "node:child_process";
import { fileURLToPath } from "node:url";
import { validateConstructiveIntelligenceTree } from "./validate-constructive-intelligence-tree.mjs";

const LAB_SCHEMA = "zerone.local-power-reward-lab/v1";
const POWER_SURFACES = [
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
];
const LEDGER_LANES = [
  {
    id: "validity",
    status: "synthetic-evidence-only",
    detail:
      "The score fixture has provenance, validity and safety inputs; no canonical proof or receipt is consumed.",
  },
  {
    id: "novelty-priority",
    status: "preclustered-assumption",
    detail:
      "Artifacts arrive in assumed semantic clusters; priority commitments, disputes and adjudication are absent.",
  },
  {
    id: "significance-consequence",
    status: "synthetic-score-input",
    detail:
      "Marginal-use inputs are illustrative and are not independently observed adoption or descendant impact.",
  },
  {
    id: "attribution-credit",
    status: "synthetic-controller-credits",
    detail:
      "Controller credit partitions are fixed fixtures; controller attestation and attribution appeals are absent.",
  },
  {
    id: "funding-liability",
    status: "counterfactual-only",
    detail:
      "Model-unit allocations and backlog are calculated without escrow, expiry lots, extinguishment or bank settlement.",
  },
  {
    id: "governance-authority",
    status: "absent",
    detail:
      "No proposal, chamber approval, activation height, payout authority or chain release decision is represented.",
  },
];

const scriptDirectory = dirname(fileURLToPath(import.meta.url));
const dashboardDirectory = resolve(scriptDirectory, "..");
const repositoryRoot = resolve(dashboardDirectory, "..");
const treePath = resolve(
  dashboardDirectory,
  "public/standards/constructive-intelligence-tree.v1.json",
);
const fixturePath = resolve(
  dashboardDirectory,
  "src/power-lab/shadow.generated.json",
);

const sha256 = (contents) =>
  createHash("sha256").update(contents).digest("hex");

const treeBytes = readFileSync(treePath);
const tree = JSON.parse(treeBytes.toString("utf8"));
const treeSummary = validateConstructiveIntelligenceTree(
  structuredClone(tree),
);

const simulationRun = spawnSync(
  "go",
  [
    "run",
    "./tools/constructive-rewards",
    "-mode",
    "report",
    "-format",
    "json",
  ],
  {
    cwd: repositoryRoot,
    encoding: "utf8",
    maxBuffer: 16 * 1024 * 1024,
  },
);
if (simulationRun.error) throw simulationRun.error;
if (simulationRun.status !== 0) {
  throw new Error(
    `constructive-rewards report failed (${simulationRun.status}): ${simulationRun.stderr}`,
  );
}
const simulationBytes = Buffer.from(simulationRun.stdout, "utf8");
const report = JSON.parse(simulationRun.stdout);

const shadowRun = spawnSync(
  "go",
  [
    "run",
    "./tools/constructive-rewards",
    "-mode",
    "shadow",
    "-format",
    "json",
  ],
  {
    cwd: repositoryRoot,
    encoding: "utf8",
    maxBuffer: 16 * 1024 * 1024,
  },
);
if (shadowRun.error) throw shadowRun.error;
if (shadowRun.status !== 0) {
  throw new Error(
    `constructive-rewards shadow failed (${shadowRun.status}): ${shadowRun.stderr}`,
  );
}
const shadowBytes = Buffer.from(shadowRun.stdout, "utf8");
const shadowReport = JSON.parse(shadowRun.stdout);
if (
  shadowReport.schema !== "zerone.constructive-capacity-shadow/v1" ||
  shadowReport.authoritative !== false ||
  shadowReport.network_observed !== false ||
  shadowReport.reward_bearing !== false ||
  shadowReport.transferable_value !== false ||
  shadowReport.moves_funds !== false ||
  shadowReport.settlement_zrn !== 0 ||
  shadowReport.integration_ready !== false ||
  shadowReport.checks?.passed !== true ||
  shadowReport.trace?.length !== 4
) {
  throw new Error("the local lab requires the reviewed zero-value exact shadow trace");
}

const modelGates = report.release_gates.filter(
  (gate) => gate.class === "model",
);
const integrationGates = report.release_gates.filter(
  (gate) => gate.class === "integration",
);
if (
  report.model_checks_passed !== true ||
  report.integration_ready !== false ||
  modelGates.length !== 12 ||
  integrationGates.length !== 19 ||
  modelGates.some((gate) => gate.passed !== true) ||
  integrationGates.some((gate) => gate.passed !== false)
) {
  throw new Error(
    "the local lab requires the reviewed 12-pass/19-closed simulator posture",
  );
}

const metricsByName = new Map(
  report.current_power_surfaces.map((surface) => [surface.surface, surface]),
);
if (
  metricsByName.size !== POWER_SURFACES.length ||
  POWER_SURFACES.some((name) => !metricsByName.has(name))
) {
  throw new Error("simulator power stress fixture does not have exactly 12 surfaces");
}

const minimumEffective = report.parameters.min_power_effective;
const minimumNakamoto = report.parameters.min_power_nakamoto;
const powerSurfaces = POWER_SURFACES.map((name) => {
  const metric = metricsByName.get(name);
  const coalitionThreshold =
    report.parameters.power_coalition_thresholds[name];
  return {
    id: name,
    hhi: metric.hhi,
    effectiveCount: metric.effective_count,
    nakamotoCount: metric.nakamoto_count,
    largestShare: metric.largest_share,
    coalitionThreshold,
    passesIllustrativeFloor:
      metric.effective_count >= minimumEffective &&
      metric.nakamoto_count >= minimumNakamoto,
  };
});

const epoch = report.illustrative_epoch;
const clusters = epoch.clusters.map((cluster) => {
  const direct = cluster.allocations.reduce(
    (sum, allocation) => sum + allocation.direct,
    0,
  );
  const commons = cluster.allocations.reduce(
    (sum, allocation) => sum + allocation.overflow_to_commons,
    0,
  );
  return {
    id: cluster.id,
    artifactCount: cluster.artifact_count,
    highWater: cluster.new_high_water,
    lifetimeCap: epoch.states[cluster.id].lifetime_cap,
    newGrossAccrual: cluster.new_gross_accrual,
    eligibleDemand: cluster.eligible_demand,
    counterfactualFunded: cluster.funded_budget,
    fundedToDate: cluster.new_funded,
    unfundedDemand: Math.max(
      0,
      cluster.eligible_demand - cluster.funded_budget,
    ),
    direct,
    commons,
    canonicalTreeReceipt: null,
    evidenceMilestone: null,
    extinguishedToDate: null,
    eligibilityLots: null,
    settlementZrn: 0,
    settlementState: "blocked",
  };
});

const mathNodes = tree.nodes
  .filter((node) => node.domain === "mathematics")
  .map((node) => ({
    id: node.id,
    title: node.title,
    stage: node.stage,
    domain: node.domain,
    prerequisites: node.prerequisites,
    rewardEligibility: node.rewardEligibility,
  }));

const fixture = {
  schema: LAB_SCHEMA,
  authoritative: false,
  networkObserved: false,
  rewardBearing: false,
  transferableValue: false,
  chainStateRead: false,
  sources: {
    generatorVersion: "power-lab-projection-v1",
    tree: {
      schema: tree.schema,
      policyVersion: tree.policyVersion,
      snapshotDate: tree.snapshotDate,
      sha256: sha256(treeBytes),
    },
    simulation: {
      sha256: sha256(simulationBytes),
    },
    shadowLedger: {
      sha256: sha256(shadowBytes),
    },
  },
  treeBoundary: {
    ...tree.releaseBoundary,
  },
  capability: {
    roots: tree.roots,
    nodeCount: treeSummary.nodeCount,
    questCount: treeSummary.questCount,
    mathNodes,
    skillUnlockCreatesReward: tree.policy.funding.skillUnlockCreatesReward,
    protocolIssuanceGate: tree.policy.funding.protocolIssuanceGate,
  },
  powerStress: {
    kind: "synthetic-captured-policy-stress",
    causallyLinkedToIllustrativeEpoch: false,
    controllerInference: "synthetic-policy-labels",
    minimumEffectiveCount: minimumEffective,
    minimumNakamotoCount: minimumNakamoto,
    surfaces: powerSurfaces,
    history: null,
    uncertainty: null,
    jointPathCut: null,
  },
  shadowEpoch: {
    kind: "synthetic-balanced-policy-allocation",
    unit: "model-unit",
    budget: epoch.budget,
    direct: epoch.direct_total,
    commons: epoch.commons_total,
    unallocated: epoch.unallocated,
    unfundedDemand: epoch.unfunded_demand,
    clusters,
  },
  capacityShadow: {
    schema: shadowReport.schema,
    arithmeticVersion: shadowReport.arithmetic_version,
    unit: shadowReport.unit,
    authoritative: shadowReport.authoritative,
    networkObserved: shadowReport.network_observed,
    rewardBearing: shadowReport.reward_bearing,
    transferableValue: shadowReport.transferable_value,
    movesFunds: shadowReport.moves_funds,
    settlementZrn: shadowReport.settlement_zrn,
    integrationReady: shadowReport.integration_ready,
    trace: shadowReport.trace.map(({ event, snapshot }) => ({
      event,
      epoch: snapshot.current_epoch,
      accrued: snapshot.accrued,
      funded: snapshot.funded,
      live: snapshot.live,
      quarantined: snapshot.quarantined,
      extinguished: snapshot.extinguished,
      replacementUsed: snapshot.replacement_used,
      deadline: snapshot.lots[0]?.deadline,
    })),
    checks: shadowReport.checks,
  },
  ledgerLanes: LEDGER_LANES,
  release: {
    settlementEnabled: false,
    claimable: false,
    settlementZrn: 0,
    modelChecksPassed: report.model_checks_passed,
    integrationReady: report.integration_ready,
    modelPassCount: modelGates.length,
    integrationFailCount: integrationGates.length,
    gates: report.release_gates,
  },
};

const rendered = `${JSON.stringify(fixture, null, 2)}\n`;
if (process.argv.includes("--check")) {
  if (!existsSync(fixturePath)) {
    throw new Error(`generated fixture is missing: ${fixturePath}`);
  }
  const checkedIn = readFileSync(fixturePath, "utf8");
  if (checkedIn !== rendered) {
    throw new Error(
      "power-lab fixture is stale; run `npm run generate:lab-fixture`",
    );
  }
  console.log(
    `Power-lab fixture OK: ${treeSummary.nodeCount} nodes, ${powerSurfaces.length} surfaces, ${modelGates.length}/${integrationGates.length} gate posture`,
  );
} else {
  writeFileSync(fixturePath, rendered, "utf8");
  console.log(`Wrote ${fixturePath}`);
}
