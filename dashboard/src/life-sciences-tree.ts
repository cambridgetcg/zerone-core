/// <reference lib="dom" />

export const LIFE_SCIENCES_TREE_ENDPOINT =
  "/standards/constructive-intelligence-life-sciences.v0.json";
export const LIFE_SCIENCES_TREE_SHA256 =
  "2b3a0b0b92797ef459c6ec02a38d1b9ebde62105a1b558088e44779d4595508d";
export const LIFE_SCIENCES_TREE_MAX_BYTES = 196_608;

const SCHEMA = "zerone.constructive-intelligence-life-sciences/v0";
const BASE_TREE_SHA256 =
  "8070d8d1b7ea28a314f5a8550c675d7ccbe5d9b234ef02d54d4913c650c01aaf";
const STAGES = [
  "foundation",
  "measurement",
  "inference",
  "validation",
  "integration",
  "crown",
] as const;
const BRANCHES = [
  "biomolecule-foundations",
  "protein-folding",
  "gene-expression",
  "integration",
] as const;
const EVIDENCE_LEVELS = ["LS0", "LS1", "LS2", "LS3", "LS4", "LS5"] as const;

type Stage = (typeof STAGES)[number];
type Branch = (typeof BRANCHES)[number];
type EvidenceLevel = (typeof EVIDENCE_LEVELS)[number];

export interface LifeSciencesNode {
  id: string;
  title: string;
  stage: Stage;
  branch: Branch;
  summary: string;
  prerequisites: string[];
  evidenceFloor: EvidenceLevel;
  claimClass: string;
  permittedConclusions: string[];
  forbiddenConclusions: string[];
  crown: boolean;
}

interface NonImplicationWall {
  id: string;
  premise: string;
  doesNotEstablish: string[];
}

export interface LifeSciencesOverlay {
  schema: typeof SCHEMA;
  status: "DRAFT";
  mode: "SHADOW_ONLY";
  snapshotDate: string;
  baseTreeSha256: string;
  riskClass: "GREEN_ONLY";
  refusedTopics: string[];
  minimumEffectiveControlClusters: number;
  nonImplicationWalls: NonImplicationWall[];
  nodes: LifeSciencesNode[];
}

export class LifeSciencesTreeError extends Error {
  constructor(message: string) {
    super(message);
    this.name = "LifeSciencesTreeError";
  }
}

function fail(path: string, message: string): never {
  throw new LifeSciencesTreeError(`${path}: ${message}`);
}

function object(value: unknown, path: string): Record<string, unknown> {
  if (value === null || typeof value !== "object" || Array.isArray(value)) {
    fail(path, "must be an object");
  }
  return value as Record<string, unknown>;
}

function text(value: unknown, path: string, max = 1_024): string {
  if (typeof value !== "string" || value.length === 0 || value.length > max) {
    fail(path, `must be a non-empty string no longer than ${max} characters`);
  }
  return value;
}

function stringArray(value: unknown, path: string, max = 64): string[] {
  if (!Array.isArray(value) || value.length > max) fail(path, "must be a bounded array");
  return value.map((item, index) => text(item, `${path}[${index}]`, 512));
}

function exactFalse(value: unknown, path: string): false {
  if (value !== false) fail(path, "must remain false");
  return false;
}

function member<T extends readonly string[]>(
  value: unknown,
  allowed: T,
  path: string,
): T[number] {
  if (typeof value !== "string" || !allowed.includes(value as T[number])) {
    fail(path, `must be one of ${allowed.join(", ")}`);
  }
  return value as T[number];
}

export function parseLifeSciencesOverlay(value: unknown): LifeSciencesOverlay {
  const root = object(value, "$");
  if (root.schema !== SCHEMA) fail("$.schema", "unsupported schema");
  if (root.status !== "DRAFT") fail("$.status", "must remain DRAFT");
  if (root.mode !== "SHADOW_ONLY") fail("$.mode", "must remain SHADOW_ONLY");
  exactFalse(root.authoritative, "$.authoritative");
  exactFalse(root.networkObserved, "$.networkObserved");
  exactFalse(root.rewardBearing, "$.rewardBearing");

  const binding = object(root.baseTreeBinding, "$.baseTreeBinding");
  if (binding.sha256 !== BASE_TREE_SHA256) {
    fail("$.baseTreeBinding.sha256", "core-tree binding drifted");
  }

  const boundary = object(root.releaseBoundary, "$.releaseBoundary");
  for (const key of [
    "addsConsensusBehavior",
    "activatesRewards",
    "movesFunds",
    "grantsQualification",
    "grantsGovernanceAuthority",
    "createsKarmaMagnitude",
    "createsTransferableKarma",
    "authorizesExperiments",
    "authorizesClinicalDecisions",
    "performsNetworkRequests",
    "publishesRestrictedEvidence",
    "marksReleaseReady",
  ]) {
    exactFalse(boundary[key], `$.releaseBoundary.${key}`);
  }

  const economics = object(root.economics, "$.economics");
  if (
    economics.effect !== "NONE" ||
    economics.amount !== "0" ||
    economics.denom !== null ||
    economics.escrowReference !== null
  ) {
    fail("$.economics", "must remain NONE/0 with no denomination or escrow");
  }
  exactFalse(economics.rewardMultiplier, "$.economics.rewardMultiplier");

  const scope = object(root.scope, "$.scope");
  if (scope.riskClass !== "GREEN_ONLY") fail("$.scope.riskClass", "must remain GREEN_ONLY");
  const refusedTopics = stringArray(scope.refusedTopics, "$.scope.refusedTopics", 32);
  for (const topic of [
    "PATHOGENS",
    "TOXINS",
    "VIRULENCE",
    "IMMUNE_EVASION",
    "HOST_RANGE",
    "GERMLINE",
    "CLINICAL_DECISIONS",
    "IDENTIFIABLE_HUMAN_DATA",
    "RAW_HUMAN_GENOME",
    "OPERATIONAL_WET_LAB_ENABLEMENT",
  ]) {
    if (!refusedTopics.includes(topic)) fail("$.scope.refusedTopics", `missing ${topic}`);
  }

  const evidencePolicy = object(root.evidencePolicy, "$.evidencePolicy");
  if (
    evidencePolicy.equivalenceToCoreTree !== "NO_EQUIVALENCE_TO_E0_E6" ||
    evidencePolicy.equivalenceToPoca !== "NO_EQUIVALENCE_TO_POCA_TIERS"
  ) {
    fail("$.evidencePolicy", "LS levels must remain non-equivalent to core and PoCA tiers");
  }

  const independence = object(root.independence, "$.independence");
  if (independence.minimumEffectiveControlClusters !== 3) {
    fail("$.independence.minimumEffectiveControlClusters", "must remain 3");
  }
  exactFalse(independence.rawIdentityCountIsEvidence, "$.independence.rawIdentityCountIsEvidence");

  if (!Array.isArray(root.nonImplicationWalls) || root.nonImplicationWalls.length !== 4) {
    fail("$.nonImplicationWalls", "must contain the four scientific inference walls");
  }
  const nonImplicationWalls = root.nonImplicationWalls.map((item, index) => {
    const wall = object(item, `$.nonImplicationWalls[${index}]`);
    return {
      id: text(wall.id, `$.nonImplicationWalls[${index}].id`, 128),
      premise: text(wall.premise, `$.nonImplicationWalls[${index}].premise`, 128),
      doesNotEstablish: stringArray(
        wall.doesNotEstablish,
        `$.nonImplicationWalls[${index}].doesNotEstablish`,
        8,
      ),
    };
  });

  if (!Array.isArray(root.nodes) || root.nodes.length < 12 || root.nodes.length > 20) {
    fail("$.nodes", "must contain 12 through 20 nodes");
  }
  const nodes = root.nodes.map((item, index): LifeSciencesNode => {
    const node = object(item, `$.nodes[${index}]`);
    if (typeof node.crown !== "boolean") fail(`$.nodes[${index}].crown`, "must be boolean");
    return {
      id: text(node.id, `$.nodes[${index}].id`, 128),
      title: text(node.title, `$.nodes[${index}].title`, 256),
      stage: member(node.stage, STAGES, `$.nodes[${index}].stage`),
      branch: member(node.branch, BRANCHES, `$.nodes[${index}].branch`),
      summary: text(node.summary, `$.nodes[${index}].summary`, 1_024),
      prerequisites: stringArray(node.prerequisites, `$.nodes[${index}].prerequisites`, 4),
      evidenceFloor: member(
        node.evidenceFloor,
        EVIDENCE_LEVELS,
        `$.nodes[${index}].evidenceFloor`,
      ),
      claimClass: text(node.claimClass, `$.nodes[${index}].claimClass`, 128),
      permittedConclusions: stringArray(
        node.permittedConclusions,
        `$.nodes[${index}].permittedConclusions`,
        16,
      ),
      forbiddenConclusions: stringArray(
        node.forbiddenConclusions,
        `$.nodes[${index}].forbiddenConclusions`,
        16,
      ),
      crown: node.crown,
    };
  });

  const byId = new Map<string, LifeSciencesNode>();
  for (const node of nodes) {
    if (byId.has(node.id)) fail("$.nodes", `duplicate node ${node.id}`);
    byId.set(node.id, node);
  }
  for (const node of nodes) {
    for (const prerequisite of node.prerequisites) {
      if (!byId.has(prerequisite)) fail("$.nodes", `${node.id} has dangling prerequisite ${prerequisite}`);
    }
  }
  if (nodes.filter((node) => node.crown).length !== 1) {
    fail("$.nodes", "must contain exactly one crown");
  }

  return {
    schema: SCHEMA,
    status: "DRAFT",
    mode: "SHADOW_ONLY",
    snapshotDate: text(root.snapshotDate, "$.snapshotDate", 10),
    baseTreeSha256: BASE_TREE_SHA256,
    riskClass: "GREEN_ONLY",
    refusedTopics,
    minimumEffectiveControlClusters: 3,
    nonImplicationWalls,
    nodes,
  };
}

function hex(bytes: ArrayBuffer): string {
  return Array.from(new Uint8Array(bytes), (byte) => byte.toString(16).padStart(2, "0")).join("");
}

export async function fetchLifeSciencesOverlay(
  fetcher: typeof fetch = fetch,
): Promise<LifeSciencesOverlay> {
  const response = await fetcher(LIFE_SCIENCES_TREE_ENDPOINT, {
    cache: "no-store",
    headers: { Accept: "application/json" },
  });
  if (!response.ok) throw new LifeSciencesTreeError(`Life-science overlay returned HTTP ${response.status}`);
  const raw = await response.text();
  const bytes = new TextEncoder().encode(raw);
  if (bytes.byteLength > LIFE_SCIENCES_TREE_MAX_BYTES) {
    throw new LifeSciencesTreeError("Life-science overlay exceeds its byte limit");
  }
  const digest = hex(await crypto.subtle.digest("SHA-256", bytes));
  if (digest !== LIFE_SCIENCES_TREE_SHA256) {
    throw new LifeSciencesTreeError("Life-science overlay bytes do not match the reviewed SHA-256");
  }
  let parsed: unknown;
  try {
    parsed = JSON.parse(raw);
  } catch {
    throw new LifeSciencesTreeError("Life-science overlay is not valid JSON");
  }
  return parseLifeSciencesOverlay(parsed);
}

function el<K extends keyof HTMLElementTagNameMap>(
  tag: K,
  className?: string,
  copy?: string,
): HTMLElementTagNameMap[K] {
  const node = document.createElement(tag);
  if (className) node.className = className;
  if (copy !== undefined) node.textContent = copy;
  return node;
}

function label(value: string): string {
  return value
    .toLowerCase()
    .split(/[-_]/)
    .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
    .join(" ");
}

function renderOverlay(root: HTMLElement, overlay: LifeSciencesOverlay): void {
  const shell = el("div", "ls-tree");
  const facts = el("div", "ls-tree-facts");
  for (const [name, value] of [
    ["Profile", `${overlay.status} · ${overlay.mode.replaceAll("_", " ")}`],
    ["Graph", `${overlay.nodes.length} nodes · 4 branches`],
    ["Evidence", "LS0–LS5 · no tier equivalence"],
    ["Economics", "NONE · amount 0 · no denomination"],
  ]) {
    const item = el("div");
    item.append(el("span", undefined, name), el("strong", undefined, value));
    facts.append(item);
  }

  const boundary = el("div", "ls-tree-boundary");
  boundary.append(
    el("strong", undefined, "GREEN-only shadow map"),
    el(
      "p",
      undefined,
      `No experiments, clinical decisions, qualification, governance, rewards, or funds. Crown review needs ${overlay.minimumEffectiveControlClusters} effective control clusters and a clear challenge state.`,
    ),
  );

  const walls = el("div", "ls-tree-walls");
  overlay.nonImplicationWalls.forEach((wall) => {
    const card = el("article");
    card.append(
      el("span", undefined, label(wall.premise)),
      el("strong", undefined, "≠"),
      el("p", undefined, wall.doesNotEstablish.map(label).join(" or ")),
    );
    walls.append(card);
  });

  const map = el("div", "ls-tree-map");
  for (const branch of BRANCHES) {
    const section = el("section", `ls-tree-branch branch-${branch}`);
    const branchNodes = overlay.nodes.filter((node) => node.branch === branch);
    section.append(
      el("span", "card-kicker", label(branch)),
      el("h3", undefined, branch === "integration" ? "Cross-domain bosses" : label(branch)),
    );
    const list = el("ol");
    branchNodes.forEach((node) => {
      const item = el("li");
      const details = el("details", node.crown ? "is-crown" : undefined);
      const summary = el("summary");
      const meta = el("span", "ls-tree-node-meta");
      meta.append(
        el("span", undefined, node.evidenceFloor),
        el("span", undefined, label(node.stage)),
      );
      summary.append(meta, el("strong", undefined, node.title));
      const body = el("div", "ls-tree-node-body");
      body.append(el("p", undefined, node.summary));
      if (node.prerequisites.length > 0) {
        body.append(
          el(
            "small",
            undefined,
            `Requires: ${node.prerequisites.map((id) => overlay.nodes.find((candidate) => candidate.id === id)?.title ?? id).join(" · ")}`,
          ),
        );
      }
      if (node.forbiddenConclusions.length > 0) {
        body.append(
          el(
            "small",
            "ls-tree-refusal",
            `Does not establish: ${node.forbiddenConclusions.map(label).join(" · ")}`,
          ),
        );
      }
      details.append(summary, body);
      item.append(details);
      list.append(item);
    });
    section.append(list);
    map.append(section);
  }

  const footer = el("div", "ls-tree-footer");
  const raw = el("a", "button button-ghost", "Open profile JSON ↗");
  raw.href = LIFE_SCIENCES_TREE_ENDPOINT;
  raw.target = "_blank";
  raw.rel = "noreferrer";
  footer.append(
    el(
      "p",
      undefined,
      `Static snapshot ${overlay.snapshotDate} · base tree ${overlay.baseTreeSha256.slice(0, 12)}… · not scientific truth or payment state.`,
    ),
    raw,
  );

  shell.append(facts, boundary, walls, map, footer);
  root.replaceChildren(shell);
  root.setAttribute("aria-busy", "false");
}

export async function initialiseLifeSciencesTree(root: HTMLElement): Promise<void> {
  root.setAttribute("aria-busy", "true");
  try {
    renderOverlay(root, await fetchLifeSciencesOverlay());
  } catch (error) {
    const failure = el("div", "ci-tree-load-error");
    failure.setAttribute("role", "alert");
    failure.append(
      el("strong", undefined, "The life-science overlay could not be verified."),
      el("p", undefined, error instanceof Error ? error.message : "The static profile is unavailable."),
    );
    root.replaceChildren(failure);
    root.setAttribute("aria-busy", "false");
  }
}
