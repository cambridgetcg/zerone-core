/// <reference lib="dom" />

export const RELATIONAL_TOPOLOGY_ENDPOINT =
  "/standards/relational-topology.v0.json";
export const RELATIONAL_TOPOLOGY_SHA256 =
  "9786674730febfe47150f29adffa4e4f7bd98e2aff502c552fa5b9669d935711";
export const RELATIONAL_TOPOLOGY_MAX_BYTES = 131_072;

const TOPOLOGY_SCHEMA = "zerone.relational-topology/v0";
const SAFE_ID = /^[a-z][a-z0-9-]{0,95}$/u;
const SAFE_REPO_PATH = /^docs\/[A-Za-z0-9][A-Za-z0-9._/-]{0,239}$/u;
const SHA256 = /^[0-9a-f]{64}$/u;
const BIDI_CONTROLS = /[\u061c\u200e\u200f\u202a-\u202e\u2066-\u2069]/u;
const UNSAFE_CONTROLS = /[\u0000-\u0008\u000b\u000c\u000e-\u001f\u007f]/u;

export const RELATIONAL_FLOWS = [
  "AUTHORITY",
  "ECONOMIC",
  "EMERGENCY",
  "EVIDENCE",
  "IDENTITY",
  "RECOGNITION",
  "REFERENCE",
] as const;

export type RelationalFlow = (typeof RELATIONAL_FLOWS)[number];
export type RelationalPlaneId =
  | "economy"
  | "governance"
  | "identity"
  | "knowledge";
export type RelationalNodeRole =
  | "BOUNDED_HANDLER"
  | "BOUNDED_PROCESS"
  | "CIRCUIT_BREAKER"
  | "DERIVED_PROJECTION"
  | "EXIT_ONLY_LEDGER"
  | "SOLE_AUTHORITY"
  | "SUPPORTING_STATE";
export type RelationalImplementation =
  | "EXISTING_SOURCE"
  | "STATIC_ONLY"
  | "TARGET_NOT_IMPLEMENTED"
  | "TARGET_SEMANTICS_NOT_ACTIVATED";
export type RelationalEdgeStatus =
  | "DESIGN_INFERENCE"
  | "EXISTING_SOURCE"
  | "STATIC_ONLY"
  | "TARGET_NOT_ACTIVATED";

export interface RelationalTopologyStatus {
  artifact: "STATIC_CONSTITUTIONAL_PROJECTION";
  snapshotDate: string;
  describes: "ACCEPTED_TARGET_RELATIONAL_ARCHITECTURE_AND_SOURCE_CONFLICTS";
  authoritative: false;
  currentNetworkStateClaimed: false;
  consensusRuntimeDeployed: false;
  networkActivated: false;
}

export interface RelationalSourcePin {
  id: string;
  path: string;
  sha256: string;
  role: "ECONOMIC_RECOGNITION_BOUNDARY" | "NORMATIVE_TARGET_AUTHORITY";
}

export interface RelationalPrinciple {
  id: string;
  name: string;
  basis: "DECLARED_V0_PRINCIPLE" | "SOURCE_DERIVED";
  statement: string;
  structuralRule: string;
  sourceRefs: string[];
}

export interface RelationalPlane {
  id: RelationalPlaneId;
  name: string;
  meaning: string;
}

export interface RelationalNode {
  id: string;
  label: string;
  plane: RelationalPlaneId;
  role: RelationalNodeRole;
  implementation: RelationalImplementation;
  summary: string;
  sourceRef: string;
  owns: string[];
  mustNotOwn: string[];
}

export interface RelationalEdge {
  id: string;
  from: string;
  to: string;
  relation:
    | "ADMITS"
    | "AUTHENTICATES"
    | "BINDS"
    | "CANCELS"
    | "DECIDES"
    | "DEDUPLICATES"
    | "DISBURSES"
    | "EXECUTES"
    | "FILTERS"
    | "GOVERNS"
    | "PROJECTS"
    | "QUALIFIES"
    | "REFERENCES"
    | "SNAPSHOTS"
    | "SUPPLIES"
    | "UPDATES"
    | "WITHDRAWS";
  flows: RelationalFlow[];
  status: RelationalEdgeStatus;
  meaning: string;
  doesNotImply: string[];
  sourceRef: string;
}

export interface RelationalCurrentConflict {
  id: string;
  surface: string;
  conflictsWith: string;
  risk: string;
  requiredDisposition: string;
  sourceRef: string;
}

export interface RelationalForbiddenPath {
  id: string;
  flow: RelationalFlow;
  scope: "SAME_FLOW_REACHABILITY";
  from: string[];
  to: string[];
  reason: string;
}

export interface RelationalReleaseBoundary {
  changesConsensus: false;
  schedulesUpgrade: false;
  writesChainState: false;
  movesFunds: false;
  grantsAuthority: false;
  grantsIdentity: false;
  grantsQualification: false;
  recordsConsent: false;
  activatesRewards: false;
  claimsLiveMigration: false;
}

export interface RelationalTopology {
  schema: typeof TOPOLOGY_SCHEMA;
  version: 0;
  title: string;
  summary: string;
  status: RelationalTopologyStatus;
  sourcePins: RelationalSourcePin[];
  principles: RelationalPrinciple[];
  planes: RelationalPlane[];
  nodes: RelationalNode[];
  edges: RelationalEdge[];
  currentConflicts: RelationalCurrentConflict[];
  forbiddenPaths: RelationalForbiddenPath[];
  releaseBoundary: RelationalReleaseBoundary;
}

export interface RelationalTopologyFetchOptions {
  fetcher?: (
    input: RequestInfo | URL,
    init?: RequestInit,
  ) => Promise<Response>;
  timeoutMs?: number;
  baseUrl?: string;
}

export class RelationalTopologyDataError extends Error {
  constructor(message: string) {
    super(message);
    this.name = "RelationalTopologyDataError";
  }
}

type JsonObject = Record<string, unknown>;

const TOP_LEVEL_KEYS = [
  "schema",
  "version",
  "title",
  "summary",
  "status",
  "sourcePins",
  "principles",
  "planes",
  "nodes",
  "edges",
  "currentConflicts",
  "forbiddenPaths",
  "releaseBoundary",
] as const;
const STATUS_KEYS = [
  "artifact",
  "snapshotDate",
  "describes",
  "authoritative",
  "currentNetworkStateClaimed",
  "consensusRuntimeDeployed",
  "networkActivated",
] as const;
const SOURCE_PIN_KEYS = ["id", "path", "sha256", "role"] as const;
const PRINCIPLE_KEYS = [
  "id",
  "name",
  "basis",
  "statement",
  "structuralRule",
  "sourceRefs",
] as const;
const PLANE_KEYS = ["id", "name", "meaning"] as const;
const NODE_KEYS = [
  "id",
  "label",
  "plane",
  "role",
  "implementation",
  "summary",
  "sourceRef",
  "owns",
  "mustNotOwn",
] as const;
const EDGE_KEYS = [
  "id",
  "from",
  "to",
  "relation",
  "flows",
  "status",
  "meaning",
  "doesNotImply",
  "sourceRef",
] as const;
const CONFLICT_KEYS = [
  "id",
  "surface",
  "conflictsWith",
  "risk",
  "requiredDisposition",
  "sourceRef",
] as const;
const FORBIDDEN_KEYS = ["id", "flow", "scope", "from", "to", "reason"] as const;
const RELEASE_KEYS = [
  "changesConsensus",
  "schedulesUpgrade",
  "writesChainState",
  "movesFunds",
  "grantsAuthority",
  "grantsIdentity",
  "grantsQualification",
  "recordsConsent",
  "activatesRewards",
  "claimsLiveMigration",
] as const;

const NODE_ROLES = new Set<RelationalNodeRole>([
  "BOUNDED_HANDLER",
  "BOUNDED_PROCESS",
  "CIRCUIT_BREAKER",
  "DERIVED_PROJECTION",
  "EXIT_ONLY_LEDGER",
  "SOLE_AUTHORITY",
  "SUPPORTING_STATE",
]);
const IMPLEMENTATIONS = new Set<RelationalImplementation>([
  "EXISTING_SOURCE",
  "STATIC_ONLY",
  "TARGET_NOT_IMPLEMENTED",
  "TARGET_SEMANTICS_NOT_ACTIVATED",
]);
const EDGE_STATUSES = new Set<RelationalEdgeStatus>([
  "DESIGN_INFERENCE",
  "EXISTING_SOURCE",
  "STATIC_ONLY",
  "TARGET_NOT_ACTIVATED",
]);
const EDGE_RELATIONS = new Set<RelationalEdge["relation"]>([
  "ADMITS",
  "AUTHENTICATES",
  "BINDS",
  "CANCELS",
  "DECIDES",
  "DEDUPLICATES",
  "DISBURSES",
  "EXECUTES",
  "FILTERS",
  "GOVERNS",
  "PROJECTS",
  "QUALIFIES",
  "REFERENCES",
  "SNAPSHOTS",
  "SUPPLIES",
  "UPDATES",
  "WITHDRAWS",
]);
const FLOWS = new Set<RelationalFlow>(RELATIONAL_FLOWS);

const EXPECTED_SOURCE_PINS: ReadonlyMap<
  string,
  Pick<RelationalSourcePin, "path" | "sha256" | "role">
> = new Map([
  [
    "authoritative-state",
    {
      path: "docs/AUTHORITATIVE-STATE.md",
      sha256: "22d523ee25060957e2c93aba441542e35d767f28f0f0e5e86c800f5fd7ea82e9",
      role: "NORMATIVE_TARGET_AUTHORITY",
    },
  ],
  [
    "money-karma",
    {
      path: "docs/constitution/money-karma-v1.json",
      sha256: "f22e62f0706971c569bb2156400b6dbeaf72a005d822b1e40c4e2691e7a98c24",
      role: "ECONOMIC_RECOGNITION_BOUNDARY",
    },
  ],
] as const);

function fail(path: string, message: string): never {
  throw new RelationalTopologyDataError(`${path}: ${message}`);
}

function object(value: unknown, path: string): JsonObject {
  if (value === null || typeof value !== "object" || Array.isArray(value)) {
    fail(path, "must be an object");
  }
  const prototype = Object.getPrototypeOf(value);
  if (prototype !== Object.prototype && prototype !== null) {
    fail(path, "must be a plain object");
  }
  return value as JsonObject;
}

function exactKeys(
  value: JsonObject,
  expected: readonly string[],
  path: string,
): void {
  const actual = Reflect.ownKeys(value);
  if (actual.some((key) => typeof key !== "string")) {
    fail(path, "contains a non-string field");
  }
  const names = (actual as string[]).sort();
  const wanted = [...expected].sort();
  if (
    names.length !== wanted.length ||
    names.some((name, index) => name !== wanted[index])
  ) {
    const unknown = names.filter((name) => !wanted.includes(name));
    const missing = wanted.filter((name) => !names.includes(name));
    fail(
      path,
      `contains unknown or missing fields (unknown: ${unknown.join(", ") || "none"}; missing: ${missing.join(", ") || "none"})`,
    );
  }
}

function denseArray(value: unknown, path: string, maximum: number): unknown[] {
  if (!Array.isArray(value) || Object.getPrototypeOf(value) !== Array.prototype) {
    fail(path, "must be an ordinary array");
  }
  if (value.length > maximum) fail(path, `must contain at most ${maximum} items`);
  for (let index = 0; index < value.length; index += 1) {
    if (!Object.prototype.hasOwnProperty.call(value, index)) {
      fail(path, `must be dense; missing slot ${index}`);
    }
  }
  return value;
}

function text(value: unknown, path: string, maximum = 1_024): string {
  if (
    typeof value !== "string" ||
    value.length === 0 ||
    value.length > maximum ||
    value.trim() !== value ||
    BIDI_CONTROLS.test(value) ||
    UNSAFE_CONTROLS.test(value)
  ) {
    fail(path, `must be safe, trimmed text no longer than ${maximum} characters`);
  }
  return value;
}

function id(value: unknown, path: string): string {
  const result = text(value, path, 96);
  if (!SAFE_ID.test(result)) fail(path, "must be a lowercase kebab identifier");
  return result;
}

function literal<T extends string | number | boolean>(
  value: unknown,
  expected: T,
  path: string,
): T {
  if (value !== expected) fail(path, `must remain ${String(expected)}`);
  return expected;
}

function enumValue<T extends string>(
  value: unknown,
  allowed: ReadonlySet<T>,
  path: string,
): T {
  const result = text(value, path, 96) as T;
  if (!allowed.has(result)) fail(path, "has an unsupported value");
  return result;
}

function sortedStrings(
  value: unknown,
  path: string,
  maximum: number,
  allowEmpty = true,
): string[] {
  const result = denseArray(value, path, maximum).map((item, index) =>
    id(item, `${path}[${index}]`),
  );
  if (!allowEmpty && result.length === 0) fail(path, "must not be empty");
  for (let index = 1; index < result.length; index += 1) {
    if ((result[index - 1] ?? "") >= (result[index] ?? "")) {
      fail(path, "must be strictly sorted with no duplicates");
    }
  }
  return result;
}

function sortedEnums<T extends string>(
  value: unknown,
  path: string,
  allowed: ReadonlySet<T>,
  maximum: number,
): T[] {
  const result = denseArray(value, path, maximum).map((item, index) =>
    enumValue(item, allowed, `${path}[${index}]`),
  );
  if (result.length === 0) fail(path, "must not be empty");
  for (let index = 1; index < result.length; index += 1) {
    if ((result[index - 1] ?? "") >= (result[index] ?? "")) {
      fail(path, "must be strictly sorted with no duplicates");
    }
  }
  return result;
}

function safeRepoPath(value: unknown, path: string): string {
  const result = text(value, path, 240);
  if (
    !SAFE_REPO_PATH.test(result) ||
    result.includes("//") ||
    result.split("/").some((segment) => segment === "." || segment === "..")
  ) {
    fail(path, "must be a safe docs/ repository path");
  }
  return result;
}

function sha256(value: unknown, path: string): string {
  const result = text(value, path, 64);
  if (!SHA256.test(result)) fail(path, "must be lowercase SHA-256 hex");
  return result;
}

function rejectDuplicateKeysAndDepth(raw: string): void {
  let offset = 0;
  const whitespace = (): void => {
    while (/\s/u.test(raw[offset] ?? "")) offset += 1;
  };
  const scanString = (): string => {
    const start = offset;
    offset += 1;
    let escaped = false;
    while (offset < raw.length) {
      const character = raw[offset];
      if (escaped) escaped = false;
      else if (character === "\\") escaped = true;
      else if (character === '"') {
        offset += 1;
        return JSON.parse(raw.slice(start, offset)) as string;
      }
      offset += 1;
    }
    fail("$", "contains an unterminated JSON string");
  };
  const scanValue = (path: string, depth: number): void => {
    if (depth > 48) fail(path, "JSON nesting exceeds 48");
    whitespace();
    const token = raw[offset];
    if (token === "{") {
      offset += 1;
      whitespace();
      const keys = new Set<string>();
      if (raw[offset] === "}") {
        offset += 1;
        return;
      }
      while (offset < raw.length) {
        whitespace();
        if (raw[offset] !== '"') fail(path, "contains a malformed object key");
        const key = scanString();
        if (keys.has(key)) fail(`${path}.${key}`, "is a duplicate JSON key");
        keys.add(key);
        whitespace();
        if (raw[offset] !== ":") fail(path, "contains a malformed object separator");
        offset += 1;
        scanValue(`${path}.${key}`, depth + 1);
        whitespace();
        if (raw[offset] === "}") {
          offset += 1;
          return;
        }
        if (raw[offset] !== ",") fail(path, "contains a malformed object delimiter");
        offset += 1;
      }
      fail(path, "contains an unterminated object");
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
        scanValue(`${path}[${index}]`, depth + 1);
        whitespace();
        if (raw[offset] === "]") {
          offset += 1;
          return;
        }
        if (raw[offset] !== ",") fail(path, "contains a malformed array delimiter");
        offset += 1;
        index += 1;
      }
      fail(path, "contains an unterminated array");
    }
    if (token === '"') {
      scanString();
      return;
    }
    const start = offset;
    while (offset < raw.length && !/[\s,\]}]/u.test(raw[offset] ?? "")) {
      offset += 1;
    }
    if (offset === start) fail(path, "contains a malformed JSON value");
  };
  scanValue("$", 0);
  whitespace();
  if (offset !== raw.length) fail("$", "contains trailing JSON data");
}

function ensureStrictOrder<T extends { id: string }>(items: T[], path: string): void {
  for (let index = 1; index < items.length; index += 1) {
    if ((items[index - 1]?.id ?? "") >= (items[index]?.id ?? "")) {
      fail(path, "must be strictly sorted by id with no duplicates");
    }
  }
}

function ensureResolved(ref: string, known: ReadonlySet<string>, path: string): void {
  if (!known.has(ref)) fail(path, `references unknown id ${ref}`);
}

function ensureAcyclic(edges: RelationalEdge[], flow: RelationalFlow): void {
  const adjacency = new Map<string, string[]>();
  for (const edge of edges) {
    if (!edge.flows.includes(flow)) continue;
    const targets = adjacency.get(edge.from) ?? [];
    targets.push(edge.to);
    adjacency.set(edge.from, targets);
  }
  const visiting = new Set<string>();
  const visited = new Set<string>();
  const visit = (nodeId: string): void => {
    if (visiting.has(nodeId)) fail("$.edges", `${flow} flow contains a cycle at ${nodeId}`);
    if (visited.has(nodeId)) return;
    visiting.add(nodeId);
    for (const target of adjacency.get(nodeId) ?? []) visit(target);
    visiting.delete(nodeId);
    visited.add(nodeId);
  };
  for (const nodeId of adjacency.keys()) visit(nodeId);
}

export function hasRelationalFlowPath(
  edges: readonly RelationalEdge[],
  flow: RelationalFlow,
  from: string,
  to: string,
): boolean {
  const adjacency = new Map<string, string[]>();
  for (const edge of edges) {
    if (!edge.flows.includes(flow)) continue;
    const targets = adjacency.get(edge.from) ?? [];
    targets.push(edge.to);
    adjacency.set(edge.from, targets);
  }
  const pending = [from];
  const visited = new Set<string>();
  while (pending.length > 0) {
    const current = pending.shift();
    if (current === undefined || visited.has(current)) continue;
    if (current === to) return true;
    visited.add(current);
    pending.push(...(adjacency.get(current) ?? []));
  }
  return false;
}

export function parseRelationalTopology(value: unknown): RelationalTopology {
  const root = object(value, "$");
  exactKeys(root, TOP_LEVEL_KEYS, "$");

  const statusValue = object(root.status, "$.status");
  exactKeys(statusValue, STATUS_KEYS, "$.status");
  const status: RelationalTopologyStatus = {
    artifact: literal(
      statusValue.artifact,
      "STATIC_CONSTITUTIONAL_PROJECTION",
      "$.status.artifact",
    ),
    snapshotDate: literal(statusValue.snapshotDate, "2026-08-11", "$.status.snapshotDate"),
    describes: literal(
      statusValue.describes,
      "ACCEPTED_TARGET_RELATIONAL_ARCHITECTURE_AND_SOURCE_CONFLICTS",
      "$.status.describes",
    ),
    authoritative: literal(statusValue.authoritative, false, "$.status.authoritative"),
    currentNetworkStateClaimed: literal(
      statusValue.currentNetworkStateClaimed,
      false,
      "$.status.currentNetworkStateClaimed",
    ),
    consensusRuntimeDeployed: literal(
      statusValue.consensusRuntimeDeployed,
      false,
      "$.status.consensusRuntimeDeployed",
    ),
    networkActivated: literal(
      statusValue.networkActivated,
      false,
      "$.status.networkActivated",
    ),
  };

  const sourcePins = denseArray(root.sourcePins, "$.sourcePins", 8).map(
    (item, index): RelationalSourcePin => {
      const path = `$.sourcePins[${index}]`;
      const source = object(item, path);
      exactKeys(source, SOURCE_PIN_KEYS, path);
      const sourceId = id(source.id, `${path}.id`);
      const expected = EXPECTED_SOURCE_PINS.get(sourceId);
      if (expected === undefined) fail(`${path}.id`, "is not an accepted source pin");
      const sourcePath = safeRepoPath(source.path, `${path}.path`);
      const sourceDigest = sha256(source.sha256, `${path}.sha256`);
      return {
        id: sourceId,
        path: literal(sourcePath, expected.path, `${path}.path`),
        sha256: literal(sourceDigest, expected.sha256, `${path}.sha256`),
        role: literal(source.role, expected.role, `${path}.role`),
      };
    },
  );
  ensureStrictOrder(sourcePins, "$.sourcePins");
  if (sourcePins.length !== EXPECTED_SOURCE_PINS.size) {
    fail("$.sourcePins", "must contain the two exact reviewed source pins");
  }
  const sourceIds = new Set(sourcePins.map((source) => source.id));

  const principles = denseArray(root.principles, "$.principles", 16).map(
    (item, index): RelationalPrinciple => {
      const path = `$.principles[${index}]`;
      const principle = object(item, path);
      exactKeys(principle, PRINCIPLE_KEYS, path);
      const basis = enumValue(
        principle.basis,
        new Set(["DECLARED_V0_PRINCIPLE", "SOURCE_DERIVED"] as const),
        `${path}.basis`,
      );
      const sourceRefs = sortedStrings(principle.sourceRefs, `${path}.sourceRefs`, 8);
      for (const sourceRef of sourceRefs) {
        ensureResolved(sourceRef, sourceIds, `${path}.sourceRefs`);
      }
      if (basis === "SOURCE_DERIVED" && sourceRefs.length === 0) {
        fail(`${path}.sourceRefs`, "source-derived principles require provenance");
      }
      if (basis === "DECLARED_V0_PRINCIPLE" && sourceRefs.length !== 0) {
        fail(`${path}.sourceRefs`, "declared v0 principles must not borrow source provenance");
      }
      return {
        id: id(principle.id, `${path}.id`),
        name: text(principle.name, `${path}.name`, 120),
        basis,
        statement: text(principle.statement, `${path}.statement`, 600),
        structuralRule: text(principle.structuralRule, `${path}.structuralRule`, 600),
        sourceRefs,
      };
    },
  );
  ensureStrictOrder(principles, "$.principles");

  const planeIdsAllowed = new Set<RelationalPlaneId>([
    "economy",
    "governance",
    "identity",
    "knowledge",
  ]);
  const planes = denseArray(root.planes, "$.planes", 8).map(
    (item, index): RelationalPlane => {
      const path = `$.planes[${index}]`;
      const plane = object(item, path);
      exactKeys(plane, PLANE_KEYS, path);
      return {
        id: enumValue(plane.id, planeIdsAllowed, `${path}.id`),
        name: text(plane.name, `${path}.name`, 80),
        meaning: text(plane.meaning, `${path}.meaning`, 500),
      };
    },
  );
  ensureStrictOrder(planes, "$.planes");
  if (planes.length !== planeIdsAllowed.size) fail("$.planes", "must contain all four planes");
  const planeIds = new Set(planes.map((plane) => plane.id));

  const nodes = denseArray(root.nodes, "$.nodes", 64).map(
    (item, index): RelationalNode => {
      const path = `$.nodes[${index}]`;
      const node = object(item, path);
      exactKeys(node, NODE_KEYS, path);
      const sourceRef = id(node.sourceRef, `${path}.sourceRef`);
      ensureResolved(sourceRef, sourceIds, `${path}.sourceRef`);
      const owns = sortedStrings(node.owns, `${path}.owns`, 32);
      const mustNotOwn = sortedStrings(node.mustNotOwn, `${path}.mustNotOwn`, 32);
      for (const claim of owns) {
        if (mustNotOwn.includes(claim)) fail(path, `${claim} appears in owns and mustNotOwn`);
      }
      return {
        id: id(node.id, `${path}.id`),
        label: text(node.label, `${path}.label`, 120),
        plane: enumValue(node.plane, planeIdsAllowed, `${path}.plane`),
        role: enumValue(node.role, NODE_ROLES, `${path}.role`),
        implementation: enumValue(
          node.implementation,
          IMPLEMENTATIONS,
          `${path}.implementation`,
        ),
        summary: text(node.summary, `${path}.summary`, 800),
        sourceRef,
        owns,
        mustNotOwn,
      };
    },
  );
  ensureStrictOrder(nodes, "$.nodes");
  const nodeIds = new Set(nodes.map((node) => node.id));
  const ownedDomains = new Map<string, string>();
  for (const node of nodes) {
    ensureResolved(node.plane, planeIds, `$.nodes.${node.id}.plane`);
    for (const domain of node.owns) {
      const prior = ownedDomains.get(domain);
      if (prior !== undefined) fail("$.nodes", `${domain} is owned by both ${prior} and ${node.id}`);
      ownedDomains.set(domain, node.id);
    }
  }

  const edges = denseArray(root.edges, "$.edges", 128).map(
    (item, index): RelationalEdge => {
      const path = `$.edges[${index}]`;
      const edge = object(item, path);
      exactKeys(edge, EDGE_KEYS, path);
      const from = id(edge.from, `${path}.from`);
      const to = id(edge.to, `${path}.to`);
      ensureResolved(from, nodeIds, `${path}.from`);
      ensureResolved(to, nodeIds, `${path}.to`);
      if (from === to) fail(path, "self-edges are not permitted");
      const sourceRef = id(edge.sourceRef, `${path}.sourceRef`);
      ensureResolved(sourceRef, sourceIds, `${path}.sourceRef`);
      return {
        id: id(edge.id, `${path}.id`),
        from,
        to,
        relation: enumValue(edge.relation, EDGE_RELATIONS, `${path}.relation`),
        flows: sortedEnums(edge.flows, `${path}.flows`, FLOWS, RELATIONAL_FLOWS.length),
        status: enumValue(edge.status, EDGE_STATUSES, `${path}.status`),
        meaning: text(edge.meaning, `${path}.meaning`, 1_000),
        doesNotImply: sortedStrings(
          edge.doesNotImply,
          `${path}.doesNotImply`,
          24,
          false,
        ),
        sourceRef,
      };
    },
  );
  ensureStrictOrder(edges, "$.edges");
  const byNode = new Map(nodes.map((node) => [node.id, node]));
  const incident = new Set<string>();
  for (const edge of edges) {
    incident.add(edge.from);
    incident.add(edge.to);
    const source = byNode.get(edge.from);
    const target = byNode.get(edge.to);
    if (source === undefined || target === undefined) fail("$.edges", "contains an unresolved node");
    if (
      edge.flows.includes("AUTHORITY") &&
      source.role !== "SOLE_AUTHORITY" &&
      source.role !== "CIRCUIT_BREAKER"
    ) {
      fail(`$.edges.${edge.id}.flows`, "AUTHORITY must originate at a sole authority or circuit breaker");
    }
    if (edge.flows.includes("EMERGENCY") && source.role !== "CIRCUIT_BREAKER") {
      fail(`$.edges.${edge.id}.flows`, "EMERGENCY must originate at a circuit breaker");
    }
    if (edge.flows.includes("ECONOMIC") && (source.plane !== "economy" || target.plane !== "economy")) {
      fail(`$.edges.${edge.id}.flows`, "ECONOMIC flow must remain inside the economy plane");
    }
    if (edge.flows.includes("RECOGNITION") && target.role !== "DERIVED_PROJECTION") {
      fail(`$.edges.${edge.id}.flows`, "RECOGNITION must terminate at a derived projection");
    }
  }
  for (const node of nodes) {
    if (!incident.has(node.id)) fail("$.nodes", `${node.id} is isolated`);
  }
  ensureAcyclic(edges, "AUTHORITY");

  const currentConflicts = denseArray(
    root.currentConflicts,
    "$.currentConflicts",
    32,
  ).map((item, index): RelationalCurrentConflict => {
    const path = `$.currentConflicts[${index}]`;
    const conflict = object(item, path);
    exactKeys(conflict, CONFLICT_KEYS, path);
    const conflictsWith = id(conflict.conflictsWith, `${path}.conflictsWith`);
    ensureResolved(conflictsWith, nodeIds, `${path}.conflictsWith`);
    const sourceRef = id(conflict.sourceRef, `${path}.sourceRef`);
    ensureResolved(sourceRef, sourceIds, `${path}.sourceRef`);
    return {
      id: id(conflict.id, `${path}.id`),
      surface: text(conflict.surface, `${path}.surface`, 800),
      conflictsWith,
      risk: text(conflict.risk, `${path}.risk`, 800),
      requiredDisposition: text(
        conflict.requiredDisposition,
        `${path}.requiredDisposition`,
        120,
      ),
      sourceRef,
    };
  });
  ensureStrictOrder(currentConflicts, "$.currentConflicts");

  const forbiddenPaths = denseArray(root.forbiddenPaths, "$.forbiddenPaths", 32).map(
    (item, index): RelationalForbiddenPath => {
      const path = `$.forbiddenPaths[${index}]`;
      const forbidden = object(item, path);
      exactKeys(forbidden, FORBIDDEN_KEYS, path);
      const from = sortedStrings(forbidden.from, `${path}.from`, 32, false);
      const to = sortedStrings(forbidden.to, `${path}.to`, 32, false);
      for (const source of from) ensureResolved(source, nodeIds, `${path}.from`);
      for (const target of to) ensureResolved(target, nodeIds, `${path}.to`);
      if (from.some((source) => to.includes(source))) fail(path, "source and target sets must be disjoint");
      return {
        id: id(forbidden.id, `${path}.id`),
        flow: enumValue(forbidden.flow, FLOWS, `${path}.flow`),
        scope: literal(
          forbidden.scope,
          "SAME_FLOW_REACHABILITY",
          `${path}.scope`,
        ),
        from,
        to,
        reason: text(forbidden.reason, `${path}.reason`, 1_000),
      };
    },
  );
  ensureStrictOrder(forbiddenPaths, "$.forbiddenPaths");
  for (const forbidden of forbiddenPaths) {
    for (const source of forbidden.from) {
      for (const target of forbidden.to) {
        if (hasRelationalFlowPath(edges, forbidden.flow, source, target)) {
          fail(
            `$.forbiddenPaths.${forbidden.id}`,
            `${forbidden.flow} flow reaches ${target} from ${source}`,
          );
        }
      }
    }
  }

  const releaseValue = object(root.releaseBoundary, "$.releaseBoundary");
  exactKeys(releaseValue, RELEASE_KEYS, "$.releaseBoundary");
  const releaseBoundary: RelationalReleaseBoundary = {
    changesConsensus: literal(releaseValue.changesConsensus, false, "$.releaseBoundary.changesConsensus"),
    schedulesUpgrade: literal(releaseValue.schedulesUpgrade, false, "$.releaseBoundary.schedulesUpgrade"),
    writesChainState: literal(releaseValue.writesChainState, false, "$.releaseBoundary.writesChainState"),
    movesFunds: literal(releaseValue.movesFunds, false, "$.releaseBoundary.movesFunds"),
    grantsAuthority: literal(releaseValue.grantsAuthority, false, "$.releaseBoundary.grantsAuthority"),
    grantsIdentity: literal(releaseValue.grantsIdentity, false, "$.releaseBoundary.grantsIdentity"),
    grantsQualification: literal(
      releaseValue.grantsQualification,
      false,
      "$.releaseBoundary.grantsQualification",
    ),
    recordsConsent: literal(releaseValue.recordsConsent, false, "$.releaseBoundary.recordsConsent"),
    activatesRewards: literal(releaseValue.activatesRewards, false, "$.releaseBoundary.activatesRewards"),
    claimsLiveMigration: literal(
      releaseValue.claimsLiveMigration,
      false,
      "$.releaseBoundary.claimsLiveMigration",
    ),
  };

  return {
    schema: literal(root.schema, TOPOLOGY_SCHEMA, "$.schema"),
    version: literal(root.version, 0, "$.version"),
    title: text(root.title, "$.title", 160),
    summary: text(root.summary, "$.summary", 1_000),
    status,
    sourcePins,
    principles,
    planes,
    nodes,
    edges,
    currentConflicts,
    forbiddenPaths,
    releaseBoundary,
  };
}

export function parseRelationalTopologyJson(raw: string): RelationalTopology {
  if (new TextEncoder().encode(raw).byteLength > RELATIONAL_TOPOLOGY_MAX_BYTES) {
    fail("$", `exceeds the ${RELATIONAL_TOPOLOGY_MAX_BYTES}-byte limit`);
  }
  let parsed: unknown;
  try {
    parsed = JSON.parse(raw) as unknown;
  } catch (error) {
    fail("$", `is invalid JSON: ${error instanceof Error ? error.message : "unknown error"}`);
  }
  rejectDuplicateKeysAndDepth(raw);
  return parseRelationalTopology(parsed);
}

async function sha256Hex(bytes: Uint8Array): Promise<string> {
  if (globalThis.crypto?.subtle === undefined) {
    throw new RelationalTopologyDataError("Web Crypto SHA-256 is unavailable");
  }
  const digest = await globalThis.crypto.subtle.digest("SHA-256", bytes as BufferSource);
  return [...new Uint8Array(digest)]
    .map((value) => value.toString(16).padStart(2, "0"))
    .join("");
}

async function readBoundedResponse(
  response: Response,
  signal: AbortSignal,
): Promise<Uint8Array> {
  if (response.body === null) {
    throw new RelationalTopologyDataError("Relational topology returned an empty body");
  }
  const reader = response.body.getReader();
  const chunks: Uint8Array[] = [];
  let length = 0;
  let rejectOnAbort: ((reason?: unknown) => void) | undefined;
  const aborted = new Promise<never>((_resolve, reject) => {
    rejectOnAbort = reject;
  });
  const onAbort = (): void => {
    rejectOnAbort?.(signal.reason ?? new Error("Relational topology request timed out"));
  };
  signal.addEventListener("abort", onAbort, { once: true });
  if (signal.aborted) onAbort();
  try {
    while (true) {
      const { done, value } = await Promise.race([reader.read(), aborted]);
      if (done) break;
      length += value.byteLength;
      if (length > RELATIONAL_TOPOLOGY_MAX_BYTES) {
        void reader.cancel().catch(() => undefined);
        throw new RelationalTopologyDataError("Relational topology exceeds its byte limit");
      }
      chunks.push(value);
    }
  } catch (error) {
    if (signal.aborted) {
      void reader.cancel(signal.reason).catch(() => undefined);
      throw new RelationalTopologyDataError("Relational topology request timed out");
    }
    throw error;
  } finally {
    signal.removeEventListener("abort", onAbort);
    try {
      reader.releaseLock();
    } catch {
      // A hostile pending read is abandoned after refusal.
    }
  }
  const bytes = new Uint8Array(length);
  let offset = 0;
  for (const chunk of chunks) {
    bytes.set(chunk, offset);
    offset += chunk.byteLength;
  }
  return bytes;
}

export async function fetchRelationalTopology(
  options: RelationalTopologyFetchOptions = {},
): Promise<RelationalTopology> {
  const fetcher = options.fetcher ?? globalThis.fetch;
  if (fetcher === undefined) {
    throw new RelationalTopologyDataError("Relational topology is unavailable");
  }
  const baseUrl =
    options.baseUrl ??
    (typeof window === "undefined" ? "https://zerone.ai/" : window.location.href);
  const expectedUrl = new URL(RELATIONAL_TOPOLOGY_ENDPOINT, baseUrl);
  const controller = new AbortController();
  const deadline = globalThis.setTimeout(
    () => controller.abort(new Error("Relational topology request timed out")),
    options.timeoutMs ?? 8_000,
  );
  try {
    let response: Response;
    let rejectFetchOnAbort: ((reason?: unknown) => void) | undefined;
    const fetchAborted = new Promise<never>((_resolve, reject) => {
      rejectFetchOnAbort = reject;
    });
    const onFetchAbort = (): void => {
      rejectFetchOnAbort?.(controller.signal.reason ?? new Error("request timed out"));
    };
    controller.signal.addEventListener("abort", onFetchAbort, { once: true });
    if (controller.signal.aborted) onFetchAbort();
    try {
      response = await Promise.race([
        fetcher(RELATIONAL_TOPOLOGY_ENDPOINT, {
          cache: "no-store",
          credentials: "omit",
          headers: { Accept: "application/json" },
          redirect: "error",
          referrerPolicy: "no-referrer",
          signal: controller.signal,
        }),
        fetchAborted,
      ]);
    } catch {
      if (controller.signal.aborted) {
        throw new RelationalTopologyDataError("Relational topology request timed out");
      }
      throw new RelationalTopologyDataError("Relational topology is unavailable");
    } finally {
      controller.signal.removeEventListener("abort", onFetchAbort);
    }
    if (!response.ok) {
      throw new RelationalTopologyDataError(
        `Relational topology returned HTTP ${response.status}`,
      );
    }
    if (response.redirected) {
      throw new RelationalTopologyDataError("Relational topology response was redirected");
    }
    let actualUrl: URL;
    try {
      actualUrl = new URL(response.url);
    } catch {
      throw new RelationalTopologyDataError("Relational topology returned an invalid final URL");
    }
    if (actualUrl.href !== expectedUrl.href || actualUrl.origin !== expectedUrl.origin) {
      throw new RelationalTopologyDataError(
        "Relational topology left its exact same-origin path",
      );
    }
    const mediaType =
      response.headers.get("content-type")?.split(";", 1)[0]?.trim().toLowerCase() ?? "";
    if (mediaType !== "application/json") {
      throw new RelationalTopologyDataError(
        "Relational topology returned a non-application/json response",
      );
    }
    const declaredLength = response.headers.get("content-length");
    if (declaredLength !== null) {
      const length = Number(declaredLength);
      if (
        !/^\d+$/u.test(declaredLength) ||
        !Number.isSafeInteger(length) ||
        length > RELATIONAL_TOPOLOGY_MAX_BYTES
      ) {
        throw new RelationalTopologyDataError("Relational topology exceeds its byte limit");
      }
    }
    const bytes = await readBoundedResponse(response, controller.signal);
    if ((await sha256Hex(bytes)) !== RELATIONAL_TOPOLOGY_SHA256) {
      throw new RelationalTopologyDataError(
        "Relational topology bytes do not match the reviewed SHA-256",
      );
    }
    let raw: string;
    try {
      raw = new TextDecoder("utf-8", { fatal: true }).decode(bytes);
    } catch {
      throw new RelationalTopologyDataError("Relational topology is not valid UTF-8");
    }
    return parseRelationalTopologyJson(raw);
  } finally {
    globalThis.clearTimeout(deadline);
  }
}

function el<K extends keyof HTMLElementTagNameMap>(
  tag: K,
  className?: string,
  copy?: string,
): HTMLElementTagNameMap[K] {
  const node = document.createElement(tag);
  if (className !== undefined) node.className = className;
  if (copy !== undefined) node.textContent = copy;
  return node;
}

function sourceUrl(path: string): string {
  return `https://github.com/cambridgetcg/zerone-core/blob/main/${path}`;
}

export function renderRelationalTopology(
  root: HTMLElement,
  topology: RelationalTopology,
): void {
  const shell = el("div", "relational-topology-shell");
  const facts = el("div", "relational-topology-facts");
  const factData = [
    ["Map status", "SOURCE ONLY"],
    ["Distinct nodes", String(topology.nodes.length)],
    ["Typed edges", String(topology.edges.length)],
    ["Known conflicts", String(topology.currentConflicts.length)],
    ["Forbidden routes", `${topology.forbiddenPaths.length} · same-flow`],
  ] as const;
  for (const [label, value] of factData) {
    const fact = el("div", "relational-topology-fact");
    fact.append(el("span", undefined, label), el("strong", undefined, value));
    facts.append(fact);
  }

  const boundary = el("div", "relational-topology-boundary");
  boundary.append(
    el("strong", undefined, "STATIC PROJECTION · TARGET NOT LIVE"),
    el(
      "p",
      undefined,
      "The bytes map accepted target relations and known source conflicts. They do not observe the network, satisfy H4 release gates, grant authority, move value, record consent, or activate a migration.",
    ),
  );

  const principleSection = el("section", "relational-topology-principles");
  principleSection.append(
    el("h3", undefined, "Six rules for connection without capture"),
    el(
      "p",
      "relational-topology-instructions",
      "Source-derived rules retain their pins; the relation-without-debt principle is visibly declared by v0 rather than attributed backward.",
    ),
  );
  const principleList = el("div", "relational-topology-principle-list");
  for (const principle of topology.principles) {
    const card = el("article", "relational-topology-principle");
    card.append(
      el("span", undefined, principle.basis.replaceAll("_", " ")),
      el("h4", undefined, principle.name),
      el("p", undefined, principle.statement),
      el("small", undefined, principle.structuralRule),
    );
    principleList.append(card);
  }
  principleSection.append(principleList);

  const graph = el("div", "relational-topology-graph");
  graph.setAttribute("role", "region");
  graph.setAttribute("aria-label", "Relational topology node and edge graph");
  graph.tabIndex = 0;
  const nodeGrid = el("div", "relational-topology-nodes");
  const edgePanel = el("div", "relational-topology-edge-panel");
  const edgeHeading = el("div", "relational-topology-edge-heading");
  const edgeHeadingTitle = el("strong", undefined, "All typed relations");
  const edgeCount = el(
    "span",
    "relational-topology-edge-count",
    String(topology.edges.length),
  );
  edgeCount.setAttribute("aria-live", "polite");
  edgeHeading.append(edgeHeadingTitle, edgeCount);
  const edgeList = el("ol", "relational-topology-edge-list");
  const edgeRows: Array<{ edge: RelationalEdge; row: HTMLLIElement }> = [];

  for (const edge of topology.edges) {
    const row = el("li");
    const article = el("article", "relational-topology-edge");
    article.setAttribute(
      "aria-label",
      `${edge.from} ${edge.relation.toLowerCase()} ${edge.to}`,
    );
    article.append(
      el(
        "span",
        "relational-topology-edge-route",
        `${edge.from} → ${edge.to} · ${edge.relation}`,
      ),
    );
    const flows = el("div", "relational-topology-edge-flows");
    for (const flow of edge.flows) {
      const pill = el("span", "relational-topology-edge-flow", flow);
      pill.dataset.flow = flow;
      flows.append(pill);
    }
    article.append(
      flows,
      el("p", "relational-topology-edge-meaning", edge.meaning),
      el(
        "p",
        "relational-topology-edge-refusal",
        `Does not imply: ${edge.doesNotImply.join(", ")}.`,
      ),
    );
    row.append(article);
    edgeList.append(row);
    edgeRows.push({ edge, row });
  }

  const nodeButtons = new Map<string, HTMLButtonElement>();
  let selectedNode: string | null = null;
  const selectNode = (nodeId: string | null): void => {
    selectedNode = nodeId;
    let shown = 0;
    for (const [idValue, button] of nodeButtons) {
      const selected = idValue === selectedNode;
      button.classList.toggle("is-selected", selected);
      button.setAttribute("aria-pressed", String(selected));
    }
    for (const { edge, row } of edgeRows) {
      const related =
        selectedNode === null || edge.from === selectedNode || edge.to === selectedNode;
      row.hidden = !related;
      row.firstElementChild?.classList.toggle(
        "is-related",
        selectedNode !== null && related,
      );
      if (related) shown += 1;
    }
    edgeHeadingTitle.textContent =
      selectedNode === null ? "All typed relations" : `Relations touching ${selectedNode}`;
    edgeCount.textContent = String(shown);
  };

  const planeById = new Map(topology.planes.map((plane) => [plane.id, plane]));
  for (const node of topology.nodes) {
    const button = el("button", "relational-topology-node");
    button.type = "button";
    button.dataset.kind = node.role;
    button.dataset.plane = node.plane;
    button.setAttribute("aria-pressed", "false");
    const plane = planeById.get(node.plane);
    button.append(
      el(
        "span",
        "relational-topology-node-kicker",
        `${plane?.name ?? node.plane} · ${node.implementation.replaceAll("_", " ")}`,
      ),
      el("strong", "relational-topology-node-title", node.label),
      el("span", "relational-topology-node-summary", node.summary),
    );
    button.addEventListener("click", () => {
      selectNode(selectedNode === node.id ? null : node.id);
    });
    nodeButtons.set(node.id, button);
    nodeGrid.append(button);
  }
  edgePanel.append(edgeHeading, edgeList);
  graph.append(nodeGrid, edgePanel);

  const conflictSection = el("section", "relational-topology-conflicts");
  conflictSection.append(
    el("h3", undefined, "Known source conflicts remain visible"),
    el(
      "p",
      "relational-topology-instructions",
      "These six conflicts are migration debt, not a claim of exhaustive H4 authority-graph coverage or live-network observation.",
    ),
  );
  const conflictList = el("div", "relational-topology-conflict-list");
  for (const conflict of topology.currentConflicts) {
    const card = el("article", "relational-topology-conflict");
    card.append(
      el("span", undefined, `CONFLICTS WITH ${conflict.conflictsWith}`),
      el("h4", undefined, conflict.id.replaceAll("-", " ")),
      el("p", undefined, conflict.surface),
      el("small", undefined, conflict.risk),
      el("code", undefined, conflict.requiredDisposition),
    );
    conflictList.append(card);
  }
  conflictSection.append(conflictList);

  const guardSection = el("section", "relational-topology-guards");
  guardSection.append(
    el("h3", undefined, "Five same-flow reachability guards"),
    el(
      "p",
      "relational-topology-instructions",
      "Each guard is executable over one declared flow. It is not a cross-flow taint proof and cannot authorize H4.",
    ),
  );
  const guardList = el("ul", "relational-topology-guard-list");
  for (const guard of topology.forbiddenPaths) {
    const item = el("li");
    item.append(
      el("strong", undefined, `${guard.flow} · ${guard.id}`),
      el("p", undefined, guard.reason),
    );
    guardList.append(item);
  }
  guardSection.append(guardList);

  const sources = el("div", "relational-topology-sources");
  for (const source of topology.sourcePins) {
    const card = el("article", "relational-topology-source");
    const link = el("a", undefined, source.path);
    link.href = sourceUrl(source.path);
    link.target = "_blank";
    link.rel = "noreferrer";
    card.append(
      el("strong", undefined, source.role.replaceAll("_", " ")),
      link,
      el("code", undefined, `sha256:${source.sha256}`),
    );
    sources.append(card);
  }

  const footer = el("div", "relational-topology-footer");
  const controls = el("div");
  const reset = el("button", "relational-topology-reset", "Show every edge");
  reset.type = "button";
  reset.addEventListener("click", () => selectNode(null));
  const raw = el("a", "relational-topology-raw", "Open raw JSON ↗");
  raw.href = RELATIONAL_TOPOLOGY_ENDPOINT;
  raw.target = "_blank";
  raw.rel = "noreferrer";
  const design = el("a", "relational-topology-raw", "Read design record ↗");
  design.href =
    "https://github.com/cambridgetcg/zerone-core/blob/main/docs/constitution/RELATIONAL-TOPOLOGY.md";
  design.target = "_blank";
  design.rel = "noreferrer";
  controls.append(reset, raw, design);
  footer.append(
    el(
      "p",
      undefined,
      "Digest-pinned static architecture · no consensus, network observation, authority, economics, identity, qualification, consent, or reward effect.",
    ),
    controls,
  );

  shell.append(
    facts,
    boundary,
    principleSection,
    graph,
    conflictSection,
    guardSection,
    sources,
    footer,
  );
  root.replaceChildren(shell);
  root.setAttribute("aria-busy", "false");
}

export async function initialiseRelationalTopology(root: HTMLElement): Promise<void> {
  root.setAttribute("aria-busy", "true");
  try {
    renderRelationalTopology(root, await fetchRelationalTopology());
  } catch (error) {
    const failure = el("div", "relational-topology-load-error");
    failure.setAttribute("role", "alert");
    const raw = el("a", "relational-topology-raw", "Open raw static standard ↗");
    raw.href = RELATIONAL_TOPOLOGY_ENDPOINT;
    raw.target = "_blank";
    raw.rel = "noreferrer";
    failure.append(
      el("strong", undefined, "The relational topology could not be verified."),
      el(
        "p",
        undefined,
        error instanceof Error ? error.message : "The static topology is unavailable.",
      ),
      raw,
    );
    root.replaceChildren(failure);
    root.setAttribute("aria-busy", "false");
  }
}
