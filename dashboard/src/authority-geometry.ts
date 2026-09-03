/// <reference lib="dom" />

export const AUTHORITY_GEOMETRY_ENDPOINT =
  "/standards/authority-geometry.v1.json";
export const AUTHORITY_GEOMETRY_SHA256 =
  "b5f7c6bc8e897b2431688229141409224a13ab39cd57b3e408ba8ad008dbd7d6";
export const AUTHORITY_GEOMETRY_MAX_BYTES = 65_536;
export const AUTHORITY_GEOMETRY_TIMEOUT_MS = 8_000;

const TOP_LEVEL_KEYS = [
  "schema",
  "revision",
  "snapshotDate",
  "status",
  "title",
  "summary",
  "sourceDesign",
  "observationScopes",
  "currentTruth",
  "releaseBoundary",
  "principles",
  "capabilities",
  "nodes",
  "edges",
  "forbiddenInfluence",
  "sourceAnchors",
  "currentFindings",
  "staticAuthorityGate",
  "activationGates",
  "releaseAssessment",
] as const;

const RELEASE_BOUNDARY_KEYS = [
  "addsConsensusBehavior",
  "registersUpgradeHandler",
  "schedulesUpgrade",
  "changesValidatorState",
  "changesGovernanceState",
  "changesDomainState",
  "movesFunds",
  "grantsQualification",
  "createsRewardOrKarma",
  "submitsTransaction",
  "assertsDecentralization",
  "establishesLiveNetworkState",
] as const;

const CURRENT_TRUTH_KEYS = [
  "scope",
  "liveEvidence",
  "networkNamedForContext",
  "sourceHasDualAuthoritySurfaces",
  "sourceUsesBondedWealthForSdkGovernance",
  "disclosedFoundingHouseholdRetainsEffectiveControl",
  "sourceTargetModulesComplete",
  "sourceAuthorityUnified",
  "sourceRegistersH4FreezeHandler",
  "sourceRegistersH4UnificationHandler",
  "sourceRegistersH5NonEconomicGovernanceHandler",
] as const;

export const AUTHORITY_GEOMETRY_SCOPE_IDS = [
  "LIVE_NETWORK",
  "CURRENT_SOURCE",
  "ACCEPTED_TARGET",
] as const;

const SCOPE_STATUSES = [
  "NOT_ESTABLISHED_BY_THIS_ARTIFACT",
  "PINNED_STATIC_INSPECTION",
  "DESIGN_ONLY",
] as const;

export const AUTHORITY_GEOMETRY_PRINCIPLE_IDS = [
  "distinction",
  "legibility",
  "non-economic-agency",
  "challenge-and-repair",
  "honest-transition",
] as const;

export const AUTHORITY_GEOMETRY_CAPABILITY_IDS = [
  "consensus-stake",
  "account-key-binding",
  "controller-identity",
  "verifier-profile",
  "domain-qualification",
  "knowledge-evidence",
  "ordinary-governance",
  "electorate-policy",
  "domain-registry",
  "legacy-claims",
  "research-fund-egress",
  "emergency-quarantine",
] as const;

export const AUTHORITY_GEOMETRY_NODE_IDS = [
  "bank-balance",
  "sdk-staking",
  "custom-staking",
  "sdk-gov",
  "custom-gov",
  "ontology",
  "knowledge",
  "zerone-auth",
  "controller",
  "verifier-profile",
  "qualification",
  "electorate",
  "legacy-claims",
  "vesting-rewards",
  "emergency",
  "alignment",
  "claiming-pot",
  "karma",
] as const;

const EDGE_IDS = [
  "current-liquid-to-sdk-stake",
  "current-sdk-stake-to-sdk-policy",
  "current-liquid-to-custom-stake",
  "current-custom-stake-to-custom-policy",
  "current-custom-stake-to-knowledge",
  "current-knowledge-to-custom-stake-slash",
  "current-custom-stake-to-qualification",
  "current-custom-stake-to-emergency",
  "current-custom-stake-to-alignment",
  "current-custom-stake-to-claiming-pot",
  "current-custom-policy-to-research",
  "current-sdk-policy-to-knowledge-fact",
  "target-auth-to-controller",
  "target-controller-to-profile",
  "target-profile-to-qualification",
  "target-ontology-to-qualification",
  "target-qualification-to-knowledge",
  "target-controller-to-electorate",
  "target-electorate-to-sdk-policy",
  "target-sdk-policy-to-ontology",
  "target-sdk-policy-to-research",
  "target-ontology-to-knowledge",
  "target-knowledge-to-profile",
  "target-emergency-to-sdk-policy",
  "target-custom-stake-to-legacy-exit",
  "target-knowledge-domain-to-ontology",
] as const;

const FORBIDDEN_INFLUENCE_IDS = [
  "wealth-to-policy",
  "wealth-to-qualification",
  "wealth-to-truth-power",
  "karma-to-authority",
] as const;

const FORBIDDEN_INFLUENCE_ENDPOINT_CONTRACT = [
  {
    sources: ["bank-balance", "sdk-staking", "custom-staking"],
    targets: ["sdk-gov", "custom-gov", "electorate"],
  },
  {
    sources: ["bank-balance", "sdk-staking", "custom-staking"],
    targets: ["qualification"],
  },
  {
    sources: ["bank-balance", "sdk-staking", "custom-staking"],
    targets: ["knowledge"],
  },
  {
    sources: ["karma"],
    targets: [
      "bank-balance",
      "sdk-staking",
      "sdk-gov",
      "electorate",
      "qualification",
      "knowledge",
      "vesting-rewards",
      "verifier-profile",
    ],
  },
] as const;

export const AUTHORITY_GEOMETRY_SOURCE_ANCHOR_IDS = [
  "authoritative-state-design",
  "app-wiring",
  "custom-gov-resolution",
  "custom-research-spend",
  "knowledge-writers",
  "knowledge-pause-writer",
  "ontology-writer",
  "research-router-restriction",
  "custom-emergency-electorate",
  "knowledge-staking-adapter",
  "knowledge-slash-callsites",
  "alignment-staking-adapter",
  "alignment-staking-callsites",
  "claiming-pot-staking-adapter",
  "claiming-pot-staking-callsite",
  "custom-gov-quarantine-wrapper",
  "custom-gov-emergency-hold-writer",
] as const;

export const AUTHORITY_GEOMETRY_FINDING_IDS = [
  "dual-staking-ledgers",
  "dual-governance-systems",
  "dual-domain-registries",
  "direct-fact-adoption",
  "legacy-research-disbursement",
  "alternate-quarantine-surfaces",
  "custom-staking-runtime-consumers",
] as const;

export const AUTHORITY_GEOMETRY_SURFACE_IDS = [
  "validator-update-writer",
  "ordinary-governance-execution",
  "domain-registry-writer",
  "direct-fact-adoption-writer",
  "research-disbursement-writer",
  "quarantine-pause-authority",
  "custom-staking-runtime-consumer",
] as const;

const NODE_ROLES = [
  "ECONOMIC_INPUT",
  "CANONICAL_WRITER",
  "LEGACY_WRITER",
  "BOUNDED_WRITER",
  "TARGET_MODULE",
  "EVIDENCE_ONLY",
  "CURRENT_CONSUMER",
] as const;

const NODE_IMPLEMENTATIONS = [
  "PRESENT_SOURCE",
  "PRESENT_AND_TARGET",
  "TARGET_ONLY",
  "PRESENT_REQUIRES_REDESIGN",
  "SOURCE_CONSTITUTION_ONLY",
] as const;

const EDGE_SCOPES = ["CURRENT_SOURCE", "ACCEPTED_TARGET"] as const;
const EDGE_EFFECTS = [
  "ECONOMIC_INFLUENCE",
  "EPISTEMIC_INFLUENCE",
  "CONTROL_INFLUENCE",
  "VALUE_WRITE",
  "KNOWLEDGE_WRITE",
  "IDENTITY_RELATION",
  "ELIGIBILITY_RELATION",
  "REFERENCE_RELATION",
  "CONTROL_RELATION",
  "DOMAIN_WRITE",
  "EVIDENCE_RELATION",
  "RETIREMENT_RELATION",
  "SYSTEM_SIGNAL_INFLUENCE",
  "VALUE_INFLUENCE",
] as const;

const TARGET_EDGE_EFFECT_CONTRACT = {
  "target-auth-to-controller": "IDENTITY_RELATION",
  "target-controller-to-profile": "IDENTITY_RELATION",
  "target-profile-to-qualification": "ELIGIBILITY_RELATION",
  "target-ontology-to-qualification": "REFERENCE_RELATION",
  "target-qualification-to-knowledge": "ELIGIBILITY_RELATION",
  "target-controller-to-electorate": "CONTROL_RELATION",
  "target-electorate-to-sdk-policy": "CONTROL_RELATION",
  "target-sdk-policy-to-ontology": "DOMAIN_WRITE",
  "target-sdk-policy-to-research": "VALUE_WRITE",
  "target-ontology-to-knowledge": "REFERENCE_RELATION",
  "target-knowledge-to-profile": "EVIDENCE_RELATION",
  "target-emergency-to-sdk-policy": "CONTROL_RELATION",
  "target-custom-stake-to-legacy-exit": "RETIREMENT_RELATION",
  "target-knowledge-domain-to-ontology": "RETIREMENT_RELATION",
} as const satisfies Partial<
  Record<(typeof EDGE_IDS)[number], (typeof EDGE_EFFECTS)[number]>
>;

const FINDING_SURFACES = [
  "CUSTOM_STAKING_RUNTIME",
  "ORDINARY_GOVERNANCE_EXECUTION",
  "DOMAIN_REGISTRY",
  "DIRECT_FACT_ADOPTION",
  "RESEARCH_DISBURSEMENT",
  "QUARANTINE_PAUSE",
] as const;

type JsonObject = Record<string, unknown>;
export type AuthorityGeometryScopeId =
  (typeof AUTHORITY_GEOMETRY_SCOPE_IDS)[number];
export type AuthorityGeometryCapabilityId =
  (typeof AUTHORITY_GEOMETRY_CAPABILITY_IDS)[number];
export type AuthorityGeometryNodeId =
  (typeof AUTHORITY_GEOMETRY_NODE_IDS)[number];
export type AuthorityGeometryEdgeScope = (typeof EDGE_SCOPES)[number];

export interface AuthorityGeometrySourceDesign {
  repositoryPath: string;
  sha256: string;
  decisionDate: string;
  status: "ACCEPTED_SOURCE_DESIGN_ONLY";
}

export interface AuthorityGeometryObservationScope {
  id: AuthorityGeometryScopeId;
  status: (typeof SCOPE_STATUSES)[number];
  description: string;
}

export interface AuthorityGeometryCurrentTruth {
  scope: "CURRENT_SOURCE_AND_DISCLOSED_CONTEXT";
  liveEvidence: "NOT_ESTABLISHED_BY_THIS_ARTIFACT";
  networkNamedForContext: "zerone-1";
  sourceHasDualAuthoritySurfaces: true;
  sourceUsesBondedWealthForSdkGovernance: true;
  disclosedFoundingHouseholdRetainsEffectiveControl: true;
  sourceTargetModulesComplete: false;
  sourceAuthorityUnified: false;
  sourceRegistersH4FreezeHandler: false;
  sourceRegistersH4UnificationHandler: false;
  sourceRegistersH5NonEconomicGovernanceHandler: false;
}

export type AuthorityGeometryReleaseBoundary = {
  [K in (typeof RELEASE_BOUNDARY_KEYS)[number]]: false;
};

export interface AuthorityGeometryPrinciple {
  id: (typeof AUTHORITY_GEOMETRY_PRINCIPLE_IDS)[number];
  label: string;
  rule: string;
}

export interface AuthorityGeometryCapability {
  id: AuthorityGeometryCapabilityId;
  label: string;
  targetWriter: AuthorityGeometryNodeId;
  currentStateSurfaces: AuthorityGeometryNodeId[];
}

export interface AuthorityGeometryNode {
  id: AuthorityGeometryNodeId;
  label: string;
  module: string;
  role: (typeof NODE_ROLES)[number];
  implementation: (typeof NODE_IMPLEMENTATIONS)[number];
  targetCapabilities: AuthorityGeometryCapabilityId[];
}

export interface AuthorityGeometryProtections {
  consentBoundary: string;
  challengeRoute: string;
  repairRoute: string;
  exitRoute: string;
}

export interface AuthorityGeometryEdge {
  id: (typeof EDGE_IDS)[number];
  from: AuthorityGeometryNodeId;
  to: AuthorityGeometryNodeId;
  relationship: string;
  scope: AuthorityGeometryEdgeScope;
  effect: (typeof EDGE_EFFECTS)[number];
  evidence: string[];
  protections: AuthorityGeometryProtections;
}

export interface AuthorityGeometryForbiddenInfluence {
  id: (typeof FORBIDDEN_INFLUENCE_IDS)[number];
  label: string;
  sources: AuthorityGeometryNodeId[];
  targets: AuthorityGeometryNodeId[];
  acceptedTargetPathCount: 0;
}

export interface AuthorityGeometrySourceAnchor {
  id: (typeof AUTHORITY_GEOMETRY_SOURCE_ANCHOR_IDS)[number];
  path: string;
  sha256: string;
  requiredSnippets: string[];
  forbiddenSnippets: string[];
}

export interface AuthorityGeometryFinding {
  id: (typeof AUTHORITY_GEOMETRY_FINDING_IDS)[number];
  label: string;
  status: "OPEN";
  releaseSurface: (typeof FINDING_SURFACES)[number];
  nodes: AuthorityGeometryNodeId[];
  evidence: string[];
}

export interface AuthorityGeometrySurfaceCheck {
  id: (typeof AUTHORITY_GEOMETRY_SURFACE_IDS)[number];
  status: "PASS_STATIC_INSPECTION" | "FAIL";
  findingIds: (typeof AUTHORITY_GEOMETRY_FINDING_IDS)[number][];
}

export interface AuthorityGeometryStaticGate {
  authoritativeStateReleaseGate: "H4-02";
  status: "FAIL_CURRENT_SOURCE";
  mode: "REPORT_SUCCEEDS_TARGET_GATE_REFUSES";
  surfaceChecks: AuthorityGeometrySurfaceCheck[];
}

export interface AuthorityGeometryActivationGate {
  id: string;
  status: "NOT_EVIDENCED";
  summary: string;
}

export interface AuthorityGeometryReleaseAssessmentRecord {
  overall: "NO_GO";
  staticAuthoritySurfacesPassing: 1;
  staticAuthoritySurfacesTotal: 7;
  h4GatesEvidenced: 0;
  h4GatesTotal: 24;
  h5GatesEvidenced: 0;
  h5GatesTotal: 14;
  targetGateMustExitNonZero: true;
}

export interface AuthorityGeometry {
  schema: "zerone.authority-geometry/v1";
  revision: "1.0.0";
  snapshotDate: string;
  status: "SOURCE_OBSERVATORY_ONLY";
  title: string;
  summary: string;
  sourceDesign: AuthorityGeometrySourceDesign;
  observationScopes: AuthorityGeometryObservationScope[];
  currentTruth: AuthorityGeometryCurrentTruth;
  releaseBoundary: AuthorityGeometryReleaseBoundary;
  principles: AuthorityGeometryPrinciple[];
  capabilities: AuthorityGeometryCapability[];
  nodes: AuthorityGeometryNode[];
  edges: AuthorityGeometryEdge[];
  forbiddenInfluence: AuthorityGeometryForbiddenInfluence[];
  sourceAnchors: AuthorityGeometrySourceAnchor[];
  currentFindings: AuthorityGeometryFinding[];
  staticAuthorityGate: AuthorityGeometryStaticGate;
  activationGates: {
    h4: AuthorityGeometryActivationGate[];
    h5: AuthorityGeometryActivationGate[];
  };
  releaseAssessment: AuthorityGeometryReleaseAssessmentRecord;
}

export interface AuthorityGeometryAssessment {
  overall: "NO_GO";
  currentFindingCount: number;
  currentEdgeCount: number;
  targetEdgeCount: number;
  dualSurfaceCapabilityCount: number;
  targetForbiddenPathCount: number;
  staticAuthoritySurfacesPassing: number;
  staticAuthoritySurfacesTotal: number;
  h4GatesEvidenced: number;
  h4GatesTotal: number;
  h5GatesEvidenced: number;
  h5GatesTotal: number;
  targetGateSatisfied: false;
}

export interface AuthorityGeometryFetchOptions {
  fetcher?: typeof fetch;
  baseUrl?: string;
  timeoutMs?: number;
}

export class AuthorityGeometryDataError extends Error {
  constructor(message: string) {
    super(message);
    this.name = "AuthorityGeometryDataError";
  }
}

function fail(path: string, message: string): never {
  throw new AuthorityGeometryDataError(`${path}: ${message}`);
}

function asObject(value: unknown, path: string): JsonObject {
  if (typeof value !== "object" || value === null || Array.isArray(value)) {
    fail(path, "expected an object");
  }
  return value as JsonObject;
}

function exactKeys(
  value: JsonObject,
  expected: readonly string[],
  path: string,
): void {
  const actual = Object.keys(value).sort();
  const wanted = [...expected].sort();
  if (
    actual.length !== wanted.length ||
    actual.some((key, index) => key !== wanted[index])
  ) {
    fail(path, `expected exactly: ${wanted.join(", ")}`);
  }
}

function asArray(value: unknown, path: string, maximum: number): unknown[] {
  if (!Array.isArray(value)) fail(path, "expected an array");
  if (value.length > maximum) fail(path, `must contain at most ${maximum} items`);
  return value;
}

function asString(value: unknown, path: string, maximum = 8_192): string {
  if (typeof value !== "string" || value.length === 0 || value.length > maximum) {
    fail(path, `expected a non-empty string of at most ${maximum} characters`);
  }
  return value;
}

function asEnum<T extends readonly string[]>(
  value: unknown,
  allowed: T,
  path: string,
): T[number] {
  if (typeof value !== "string" || !allowed.includes(value)) {
    fail(path, `expected one of ${allowed.join(", ")}`);
  }
  return value as T[number];
}

function requireLiteral<T extends string | number | boolean>(
  value: unknown,
  expected: T,
  path: string,
): T {
  if (value !== expected) fail(path, `must remain ${JSON.stringify(expected)}`);
  return expected;
}

function asIsoDate(value: unknown, path: string): string {
  const result = asString(value, path, 10);
  if (!/^\d{4}-\d{2}-\d{2}$/.test(result)) fail(path, "expected YYYY-MM-DD");
  const milliseconds = Date.parse(`${result}T00:00:00Z`);
  if (
    !Number.isFinite(milliseconds) ||
    new Date(milliseconds).toISOString().slice(0, 10) !== result
  ) {
    fail(path, "expected a real calendar date");
  }
  return result;
}

function asSha256(value: unknown, path: string): string {
  const result = asString(value, path, 64);
  if (!/^[a-f0-9]{64}$/.test(result)) fail(path, "expected lowercase SHA-256");
  return result;
}

function asStringArray(
  value: unknown,
  path: string,
  maximum: number,
): string[] {
  const result = asArray(value, path, maximum).map((entry, index) =>
    asString(entry, `${path}[${index}]`),
  );
  if (new Set(result).size !== result.length) fail(path, "contains a duplicate");
  return result;
}

function requireExactIdSequence(
  values: readonly string[],
  expected: readonly string[],
  path: string,
): void {
  if (
    values.length !== expected.length ||
    values.some((value, index) => value !== expected[index])
  ) {
    fail(path, `must preserve exact identifiers: ${expected.join(", ")}`);
  }
}

function requiredAt<T>(values: readonly T[], index: number, path: string): T {
  const value = values[index];
  if (value === undefined) fail(path, "unexpected array position");
  return value;
}

function parseObservationScope(
  value: unknown,
  path: string,
  index: number,
): AuthorityGeometryObservationScope {
  const source = asObject(value, path);
  exactKeys(source, ["id", "status", "description"], path);
  const id = asEnum(source.id, AUTHORITY_GEOMETRY_SCOPE_IDS, `${path}.id`);
  const status = asEnum(source.status, SCOPE_STATUSES, `${path}.status`);
  requireLiteral(id, requiredAt(AUTHORITY_GEOMETRY_SCOPE_IDS, index, path), `${path}.id`);
  requireLiteral(status, requiredAt(SCOPE_STATUSES, index, path), `${path}.status`);
  return { id, status, description: asString(source.description, `${path}.description`) };
}

function parsePrinciple(
  value: unknown,
  path: string,
  index: number,
): AuthorityGeometryPrinciple {
  const source = asObject(value, path);
  exactKeys(source, ["id", "label", "rule"], path);
  const id = asEnum(
    source.id,
    AUTHORITY_GEOMETRY_PRINCIPLE_IDS,
    `${path}.id`,
  );
  requireLiteral(
    id,
    requiredAt(AUTHORITY_GEOMETRY_PRINCIPLE_IDS, index, path),
    `${path}.id`,
  );
  return {
    id,
    label: asString(source.label, `${path}.label`, 128),
    rule: asString(source.rule, `${path}.rule`),
  };
}

function parseNodeId(value: unknown, path: string): AuthorityGeometryNodeId {
  return asEnum(value, AUTHORITY_GEOMETRY_NODE_IDS, path);
}

function parseCapabilityId(
  value: unknown,
  path: string,
): AuthorityGeometryCapabilityId {
  return asEnum(value, AUTHORITY_GEOMETRY_CAPABILITY_IDS, path);
}

function parseNodeIdArray(
  value: unknown,
  path: string,
  maximum = AUTHORITY_GEOMETRY_NODE_IDS.length,
): AuthorityGeometryNodeId[] {
  const result = asArray(value, path, maximum).map((entry, index) =>
    parseNodeId(entry, `${path}[${index}]`),
  );
  if (new Set(result).size !== result.length) fail(path, "contains a duplicate node");
  return result;
}

function parseCapabilityIdArray(
  value: unknown,
  path: string,
): AuthorityGeometryCapabilityId[] {
  const result = asArray(
    value,
    path,
    AUTHORITY_GEOMETRY_CAPABILITY_IDS.length,
  ).map((entry, index) => parseCapabilityId(entry, `${path}[${index}]`));
  if (new Set(result).size !== result.length) {
    fail(path, "contains a duplicate capability");
  }
  return result;
}

function parseCapability(
  value: unknown,
  path: string,
  index: number,
): AuthorityGeometryCapability {
  const source = asObject(value, path);
  exactKeys(source, ["id", "label", "targetWriter", "currentStateSurfaces"], path);
  const id = parseCapabilityId(source.id, `${path}.id`);
  requireLiteral(
    id,
    requiredAt(AUTHORITY_GEOMETRY_CAPABILITY_IDS, index, path),
    `${path}.id`,
  );
  return {
    id,
    label: asString(source.label, `${path}.label`, 256),
    targetWriter: parseNodeId(source.targetWriter, `${path}.targetWriter`),
    currentStateSurfaces: parseNodeIdArray(
      source.currentStateSurfaces,
      `${path}.currentStateSurfaces`,
    ),
  };
}

function parseNode(
  value: unknown,
  path: string,
  index: number,
): AuthorityGeometryNode {
  const source = asObject(value, path);
  exactKeys(
    source,
    ["id", "label", "module", "role", "implementation", "targetCapabilities"],
    path,
  );
  const id = parseNodeId(source.id, `${path}.id`);
  requireLiteral(
    id,
    requiredAt(AUTHORITY_GEOMETRY_NODE_IDS, index, path),
    `${path}.id`,
  );
  return {
    id,
    label: asString(source.label, `${path}.label`, 128),
    module: asString(source.module, `${path}.module`, 256),
    role: asEnum(source.role, NODE_ROLES, `${path}.role`),
    implementation: asEnum(
      source.implementation,
      NODE_IMPLEMENTATIONS,
      `${path}.implementation`,
    ),
    targetCapabilities: parseCapabilityIdArray(
      source.targetCapabilities,
      `${path}.targetCapabilities`,
    ),
  };
}

function parseProtections(
  value: unknown,
  path: string,
): AuthorityGeometryProtections {
  const source = asObject(value, path);
  exactKeys(
    source,
    ["consentBoundary", "challengeRoute", "repairRoute", "exitRoute"],
    path,
  );
  return {
    consentBoundary: asString(source.consentBoundary, `${path}.consentBoundary`),
    challengeRoute: asString(source.challengeRoute, `${path}.challengeRoute`),
    repairRoute: asString(source.repairRoute, `${path}.repairRoute`),
    exitRoute: asString(source.exitRoute, `${path}.exitRoute`),
  };
}

function parseEdge(
  value: unknown,
  path: string,
  index: number,
): AuthorityGeometryEdge {
  const source = asObject(value, path);
  exactKeys(
    source,
    [
      "id",
      "from",
      "to",
      "relationship",
      "scope",
      "effect",
      "evidence",
      "protections",
    ],
    path,
  );
  const id = asEnum(source.id, EDGE_IDS, `${path}.id`);
  requireLiteral(id, requiredAt(EDGE_IDS, index, path), `${path}.id`);
  const evidence = asStringArray(source.evidence, `${path}.evidence`, 16);
  if (evidence.length === 0) fail(`${path}.evidence`, "must not be empty");
  const scope = asEnum(source.scope, EDGE_SCOPES, `${path}.scope`);
  const effect = asEnum(source.effect, EDGE_EFFECTS, `${path}.effect`);
  const expectedTargetEffect = (
    TARGET_EDGE_EFFECT_CONTRACT as Partial<
      Record<(typeof EDGE_IDS)[number], (typeof EDGE_EFFECTS)[number]>
    >
  )[id];
  if (expectedTargetEffect === undefined) {
    requireLiteral(scope, "CURRENT_SOURCE", `${path}.scope`);
  } else {
    requireLiteral(scope, "ACCEPTED_TARGET", `${path}.scope`);
    requireLiteral(effect, expectedTargetEffect, `${path}.effect`);
  }
  return {
    id,
    from: parseNodeId(source.from, `${path}.from`),
    to: parseNodeId(source.to, `${path}.to`),
    relationship: (() => {
      const relationship = asString(source.relationship, `${path}.relationship`, 128);
      if (!/^[A-Z][A-Z0-9]*(?:_[A-Z0-9]+)*$/.test(relationship)) {
        fail(`${path}.relationship`, "expected an uppercase snake-case relationship");
      }
      return relationship;
    })(),
    scope,
    effect,
    evidence,
    protections: parseProtections(source.protections, `${path}.protections`),
  };
}

function parseForbiddenInfluence(
  value: unknown,
  path: string,
  index: number,
): AuthorityGeometryForbiddenInfluence {
  const source = asObject(value, path);
  exactKeys(
    source,
    ["id", "label", "sources", "targets", "acceptedTargetPathCount"],
    path,
  );
  const id = asEnum(source.id, FORBIDDEN_INFLUENCE_IDS, `${path}.id`);
  requireLiteral(
    id,
    requiredAt(FORBIDDEN_INFLUENCE_IDS, index, path),
    `${path}.id`,
  );
  const sources = parseNodeIdArray(source.sources, `${path}.sources`);
  const targets = parseNodeIdArray(source.targets, `${path}.targets`);
  if (sources.length === 0 || targets.length === 0) {
    fail(path, "sources and targets must not be empty");
  }
  const endpointContract = requiredAt(
    FORBIDDEN_INFLUENCE_ENDPOINT_CONTRACT,
    index,
    path,
  );
  requireExactIdSequence(sources, endpointContract.sources, `${path}.sources`);
  requireExactIdSequence(targets, endpointContract.targets, `${path}.targets`);
  return {
    id,
    label: asString(source.label, `${path}.label`, 256),
    sources,
    targets,
    acceptedTargetPathCount: requireLiteral(
      source.acceptedTargetPathCount,
      0,
      `${path}.acceptedTargetPathCount`,
    ),
  };
}

function asRepositoryPath(value: unknown, path: string): string {
  const result = asString(value, path, 512);
  if (
    result.startsWith("/") ||
    result.includes("\\") ||
    result.includes("\0") ||
    result.split("/").some((segment) => segment === "" || segment === "." || segment === "..")
  ) {
    fail(path, "expected a safe repository-relative path");
  }
  return result;
}

function parseSourceAnchor(
  value: unknown,
  path: string,
  index: number,
): AuthorityGeometrySourceAnchor {
  const source = asObject(value, path);
  exactKeys(
    source,
    ["id", "path", "sha256", "requiredSnippets", "forbiddenSnippets"],
    path,
  );
  const id = asEnum(source.id, AUTHORITY_GEOMETRY_SOURCE_ANCHOR_IDS, `${path}.id`);
  requireLiteral(
    id,
    requiredAt(AUTHORITY_GEOMETRY_SOURCE_ANCHOR_IDS, index, path),
    `${path}.id`,
  );
  const requiredSnippets = asStringArray(
    source.requiredSnippets,
    `${path}.requiredSnippets`,
    32,
  );
  if (requiredSnippets.length === 0) {
    fail(`${path}.requiredSnippets`, "must not be empty");
  }
  const forbiddenSnippets = asStringArray(
    source.forbiddenSnippets,
    `${path}.forbiddenSnippets`,
    32,
  );
  if (forbiddenSnippets.some((snippet) => requiredSnippets.includes(snippet))) {
    fail(path, "a snippet cannot be both required and forbidden");
  }
  return {
    id,
    path: asRepositoryPath(source.path, `${path}.path`),
    sha256: asSha256(source.sha256, `${path}.sha256`),
    requiredSnippets,
    forbiddenSnippets,
  };
}

function parseFinding(
  value: unknown,
  path: string,
  index: number,
): AuthorityGeometryFinding {
  const source = asObject(value, path);
  exactKeys(source, ["id", "label", "status", "releaseSurface", "nodes", "evidence"], path);
  const id = asEnum(source.id, AUTHORITY_GEOMETRY_FINDING_IDS, `${path}.id`);
  requireLiteral(
    id,
    requiredAt(AUTHORITY_GEOMETRY_FINDING_IDS, index, path),
    `${path}.id`,
  );
  const nodes = parseNodeIdArray(source.nodes, `${path}.nodes`);
  const evidence = asStringArray(source.evidence, `${path}.evidence`, 16);
  if (nodes.length < 2) fail(`${path}.nodes`, "must identify at least two surfaces");
  if (evidence.length === 0) fail(`${path}.evidence`, "must not be empty");
  return {
    id,
    label: asString(source.label, `${path}.label`, 256),
    status: requireLiteral(source.status, "OPEN", `${path}.status`),
    releaseSurface: asEnum(
      source.releaseSurface,
      FINDING_SURFACES,
      `${path}.releaseSurface`,
    ),
    nodes,
    evidence,
  };
}

function parseSurfaceCheck(
  value: unknown,
  path: string,
  index: number,
): AuthorityGeometrySurfaceCheck {
  const source = asObject(value, path);
  exactKeys(source, ["id", "status", "findingIds"], path);
  const id = asEnum(source.id, AUTHORITY_GEOMETRY_SURFACE_IDS, `${path}.id`);
  requireLiteral(
    id,
    requiredAt(AUTHORITY_GEOMETRY_SURFACE_IDS, index, path),
    `${path}.id`,
  );
  const findingIds = asArray(
    source.findingIds,
    `${path}.findingIds`,
    AUTHORITY_GEOMETRY_FINDING_IDS.length,
  ).map((entry, findingIndex) =>
    asEnum(
      entry,
      AUTHORITY_GEOMETRY_FINDING_IDS,
      `${path}.findingIds[${findingIndex}]`,
    ),
  );
  if (new Set(findingIds).size !== findingIds.length) {
    fail(`${path}.findingIds`, "contains a duplicate finding");
  }
  const status = asEnum(
    source.status,
    ["PASS_STATIC_INSPECTION", "FAIL"] as const,
    `${path}.status`,
  );
  if ((status === "FAIL") !== (findingIds.length > 0)) {
    fail(path, "FAIL must name findings and PASS_STATIC_INSPECTION must name none");
  }
  return { id, status, findingIds };
}

function parseActivationGate(
  value: unknown,
  path: string,
  prefix: "H4" | "H5",
  index: number,
): AuthorityGeometryActivationGate {
  const source = asObject(value, path);
  exactKeys(source, ["id", "status", "summary"], path);
  const expectedId = `${prefix}-${(index + 1).toString().padStart(2, "0")}`;
  return {
    id: requireLiteral(source.id, expectedId, `${path}.id`),
    status: requireLiteral(source.status, "NOT_EVIDENCED", `${path}.status`),
    summary: asString(source.summary, `${path}.summary`, 512),
  };
}

function parseReleaseAssessment(
  value: unknown,
  path: string,
): AuthorityGeometryReleaseAssessmentRecord {
  const source = asObject(value, path);
  exactKeys(
    source,
    [
      "overall",
      "staticAuthoritySurfacesPassing",
      "staticAuthoritySurfacesTotal",
      "h4GatesEvidenced",
      "h4GatesTotal",
      "h5GatesEvidenced",
      "h5GatesTotal",
      "targetGateMustExitNonZero",
    ],
    path,
  );
  return {
    overall: requireLiteral(source.overall, "NO_GO", `${path}.overall`),
    staticAuthoritySurfacesPassing: requireLiteral(
      source.staticAuthoritySurfacesPassing,
      1,
      `${path}.staticAuthoritySurfacesPassing`,
    ),
    staticAuthoritySurfacesTotal: requireLiteral(
      source.staticAuthoritySurfacesTotal,
      7,
      `${path}.staticAuthoritySurfacesTotal`,
    ),
    h4GatesEvidenced: requireLiteral(
      source.h4GatesEvidenced,
      0,
      `${path}.h4GatesEvidenced`,
    ),
    h4GatesTotal: requireLiteral(source.h4GatesTotal, 24, `${path}.h4GatesTotal`),
    h5GatesEvidenced: requireLiteral(
      source.h5GatesEvidenced,
      0,
      `${path}.h5GatesEvidenced`,
    ),
    h5GatesTotal: requireLiteral(source.h5GatesTotal, 14, `${path}.h5GatesTotal`),
    targetGateMustExitNonZero: requireLiteral(
      source.targetGateMustExitNonZero,
      true,
      `${path}.targetGateMustExitNonZero`,
    ),
  };
}

function hasDirectedPath(
  adjacency: ReadonlyMap<AuthorityGeometryNodeId, readonly AuthorityGeometryNodeId[]>,
  source: AuthorityGeometryNodeId,
  target: AuthorityGeometryNodeId,
): boolean {
  const pending = [source];
  const visited = new Set<AuthorityGeometryNodeId>();
  while (pending.length > 0) {
    const current = pending.shift();
    if (current === undefined || visited.has(current)) continue;
    visited.add(current);
    if (current === target && current !== source) return true;
    pending.push(...(adjacency.get(current) ?? []));
  }
  return false;
}

function carriesTargetInfluence(edge: AuthorityGeometryEdge): boolean {
  return (
    edge.scope === "ACCEPTED_TARGET" &&
    edge.effect !== "REFERENCE_RELATION" &&
    edge.effect !== "EVIDENCE_RELATION" &&
    edge.effect !== "RETIREMENT_RELATION"
  );
}

function validateGraph(geometry: AuthorityGeometry): void {
  const nodeById = new Map(geometry.nodes.map((node) => [node.id, node]));
  const capabilityById = new Map(
    geometry.capabilities.map((capability) => [capability.id, capability]),
  );
  const anchorIds = new Set(geometry.sourceAnchors.map((anchor) => anchor.id));
  const usedAnchorIds = new Set<AuthorityGeometrySourceAnchor["id"]>();
  const findingIds = new Set(geometry.currentFindings.map((finding) => finding.id));

  if (nodeById.size !== geometry.nodes.length) fail("$.nodes", "duplicate node ID");
  if (capabilityById.size !== geometry.capabilities.length) {
    fail("$.capabilities", "duplicate capability ID");
  }

  for (const capability of geometry.capabilities) {
    if (!nodeById.has(capability.targetWriter)) {
      fail(`$.capabilities.${capability.id}.targetWriter`, "unresolved node");
    }
    for (const surface of capability.currentStateSurfaces) {
      if (!nodeById.has(surface)) {
        fail(`$.capabilities.${capability.id}.currentStateSurfaces`, "unresolved node");
      }
    }
    const owners = geometry.nodes.filter((node) =>
      node.targetCapabilities.includes(capability.id),
    );
    if (owners.length !== 1 || owners[0]?.id !== capability.targetWriter) {
      fail(
        `$.capabilities.${capability.id}`,
        "must have exactly one target writer and matching node ownership",
      );
    }
  }

  for (const node of geometry.nodes) {
    for (const capabilityId of node.targetCapabilities) {
      const capability = capabilityById.get(capabilityId);
      if (capability === undefined || capability.targetWriter !== node.id) {
        fail(
          `$.nodes.${node.id}.targetCapabilities`,
          "capability ownership does not match targetWriter",
        );
      }
    }
  }

  for (const edge of geometry.edges) {
    const from = nodeById.get(edge.from);
    const to = nodeById.get(edge.to);
    if (from === undefined || to === undefined) {
      fail(`$.edges.${edge.id}`, "contains an unresolved endpoint");
    }
    for (const evidenceId of edge.evidence) {
      if (!anchorIds.has(evidenceId as AuthorityGeometrySourceAnchor["id"])) {
        fail(`$.edges.${edge.id}.evidence`, `unresolved source anchor ${evidenceId}`);
      }
      usedAnchorIds.add(evidenceId as AuthorityGeometrySourceAnchor["id"]);
    }
    if (
      edge.scope === "CURRENT_SOURCE" &&
      (from.implementation === "TARGET_ONLY" || to.implementation === "TARGET_ONLY")
    ) {
      fail(`$.edges.${edge.id}.scope`, "current-source edge reaches a target-only node");
    }
    if (
      edge.scope === "ACCEPTED_TARGET" &&
      (edge.evidence.length !== 1 || edge.evidence[0] !== "authoritative-state-design")
    ) {
      fail(
        `$.edges.${edge.id}.evidence`,
        "accepted-target edges must be sourced only to the accepted design",
      );
    }
    if (
      edge.scope === "CURRENT_SOURCE" &&
      edge.evidence.includes("authoritative-state-design")
    ) {
      fail(
        `$.edges.${edge.id}.evidence`,
        "current-source evidence cannot be replaced by target-design prose",
      );
    }
  }

  for (const finding of geometry.currentFindings) {
    for (const nodeId of finding.nodes) {
      if (!nodeById.has(nodeId)) fail(`$.currentFindings.${finding.id}.nodes`, "unresolved node");
    }
    for (const evidenceId of finding.evidence) {
      if (!anchorIds.has(evidenceId as AuthorityGeometrySourceAnchor["id"])) {
        fail(
          `$.currentFindings.${finding.id}.evidence`,
          `unresolved source anchor ${evidenceId}`,
        );
      }
      usedAnchorIds.add(evidenceId as AuthorityGeometrySourceAnchor["id"]);
    }
  }

  for (const anchor of geometry.sourceAnchors) {
    if (!usedAnchorIds.has(anchor.id)) {
      fail(`$.sourceAnchors.${anchor.id}`, "source anchor is not referenced by a graph fact");
    }
  }

  const referencedFindings = new Set<string>();
  for (const check of geometry.staticAuthorityGate.surfaceChecks) {
    for (const findingId of check.findingIds) {
      if (!findingIds.has(findingId)) {
        fail(`$.staticAuthorityGate.${check.id}`, `unresolved finding ${findingId}`);
      }
      referencedFindings.add(findingId);
    }
  }
  if (
    referencedFindings.size !== findingIds.size ||
    [...findingIds].some((findingId) => !referencedFindings.has(findingId))
  ) {
    fail(
      "$.staticAuthorityGate.surfaceChecks",
      "must account for every current finding",
    );
  }

  const targetAdjacency = new Map<
    AuthorityGeometryNodeId,
    AuthorityGeometryNodeId[]
  >();
  for (const edge of geometry.edges) {
    if (!carriesTargetInfluence(edge)) continue;
    const neighbours = targetAdjacency.get(edge.from) ?? [];
    neighbours.push(edge.to);
    targetAdjacency.set(edge.from, neighbours);
  }
  for (const boundary of geometry.forbiddenInfluence) {
    let pathCount = 0;
    for (const source of boundary.sources) {
      for (const target of boundary.targets) {
        if (hasDirectedPath(targetAdjacency, source, target)) pathCount += 1;
      }
    }
    if (pathCount !== boundary.acceptedTargetPathCount || pathCount !== 0) {
      fail(
        `$.forbiddenInfluence.${boundary.id}.acceptedTargetPathCount`,
        `computed ${pathCount}; accepted target must contain zero forbidden paths`,
      );
    }
  }
}

export function assessAuthorityGeometry(
  geometry: AuthorityGeometry,
): AuthorityGeometryAssessment {
  const staticPassing = geometry.staticAuthorityGate.surfaceChecks.filter(
    (check) => check.status === "PASS_STATIC_INSPECTION",
  ).length;
  const h4Evidenced = geometry.activationGates.h4.filter(
    (gate) => gate.status !== "NOT_EVIDENCED",
  ).length;
  const h5Evidenced = geometry.activationGates.h5.filter(
    (gate) => gate.status !== "NOT_EVIDENCED",
  ).length;
  const currentEdgeCount = geometry.edges.filter(
    (edge) => edge.scope === "CURRENT_SOURCE",
  ).length;
  const targetEdgeCount = geometry.edges.length - currentEdgeCount;
  const dualSurfaceCapabilityCount = geometry.capabilities.filter(
    (capability) => capability.currentStateSurfaces.length > 1,
  ).length;

  const assessment: AuthorityGeometryAssessment = {
    overall: "NO_GO",
    currentFindingCount: geometry.currentFindings.length,
    currentEdgeCount,
    targetEdgeCount,
    dualSurfaceCapabilityCount,
    targetForbiddenPathCount: geometry.forbiddenInfluence.reduce(
      (total, boundary) => total + boundary.acceptedTargetPathCount,
      0,
    ),
    staticAuthoritySurfacesPassing: staticPassing,
    staticAuthoritySurfacesTotal: geometry.staticAuthorityGate.surfaceChecks.length,
    h4GatesEvidenced: h4Evidenced,
    h4GatesTotal: geometry.activationGates.h4.length,
    h5GatesEvidenced: h5Evidenced,
    h5GatesTotal: geometry.activationGates.h5.length,
    targetGateSatisfied: false,
  };

  const recorded = geometry.releaseAssessment;
  if (
    assessment.staticAuthoritySurfacesPassing !==
      recorded.staticAuthoritySurfacesPassing ||
    assessment.staticAuthoritySurfacesTotal !== recorded.staticAuthoritySurfacesTotal ||
    assessment.h4GatesEvidenced !== recorded.h4GatesEvidenced ||
    assessment.h4GatesTotal !== recorded.h4GatesTotal ||
    assessment.h5GatesEvidenced !== recorded.h5GatesEvidenced ||
    assessment.h5GatesTotal !== recorded.h5GatesTotal
  ) {
    fail("$.releaseAssessment", "recorded counts do not match computed assessment");
  }
  return assessment;
}

export function assertAuthorityGeometryTargetGate(
  geometry: AuthorityGeometry,
): never {
  const assessment = assessAuthorityGeometry(geometry);
  throw new AuthorityGeometryDataError(
    `target gate REFUSED: ${assessment.staticAuthoritySurfacesPassing}/${assessment.staticAuthoritySurfacesTotal} static surfaces pass, H4 ${assessment.h4GatesEvidenced}/${assessment.h4GatesTotal}, H5 ${assessment.h5GatesEvidenced}/${assessment.h5GatesTotal}; current source remains NO-GO`,
  );
}

export function parseAuthorityGeometry(value: unknown): AuthorityGeometry {
  const root = asObject(value, "$");
  exactKeys(root, TOP_LEVEL_KEYS, "$");
  requireLiteral(root.schema, "zerone.authority-geometry/v1", "$.schema");
  requireLiteral(root.revision, "1.0.0", "$.revision");
  requireLiteral(root.status, "SOURCE_OBSERVATORY_ONLY", "$.status");

  const sourceDesignSource = asObject(root.sourceDesign, "$.sourceDesign");
  exactKeys(
    sourceDesignSource,
    ["repositoryPath", "sha256", "decisionDate", "status"],
    "$.sourceDesign",
  );
  const sourceDesign: AuthorityGeometrySourceDesign = {
    repositoryPath: requireLiteral(
      sourceDesignSource.repositoryPath,
      "docs/AUTHORITATIVE-STATE.md",
      "$.sourceDesign.repositoryPath",
    ),
    sha256: requireLiteral(
      asSha256(sourceDesignSource.sha256, "$.sourceDesign.sha256"),
      "22d523ee25060957e2c93aba441542e35d767f28f0f0e5e86c800f5fd7ea82e9",
      "$.sourceDesign.sha256",
    ),
    decisionDate: asIsoDate(sourceDesignSource.decisionDate, "$.sourceDesign.decisionDate"),
    status: requireLiteral(
      sourceDesignSource.status,
      "ACCEPTED_SOURCE_DESIGN_ONLY",
      "$.sourceDesign.status",
    ),
  };

  const observationScopes = asArray(
    root.observationScopes,
    "$.observationScopes",
    AUTHORITY_GEOMETRY_SCOPE_IDS.length,
  ).map((scope, index) =>
    parseObservationScope(scope, `$.observationScopes[${index}]`, index),
  );
  requireExactIdSequence(
    observationScopes.map((scope) => scope.id),
    AUTHORITY_GEOMETRY_SCOPE_IDS,
    "$.observationScopes",
  );

  const currentTruthSource = asObject(root.currentTruth, "$.currentTruth");
  exactKeys(currentTruthSource, CURRENT_TRUTH_KEYS, "$.currentTruth");
  const currentTruth: AuthorityGeometryCurrentTruth = {
    scope: requireLiteral(
      currentTruthSource.scope,
      "CURRENT_SOURCE_AND_DISCLOSED_CONTEXT",
      "$.currentTruth.scope",
    ),
    liveEvidence: requireLiteral(
      currentTruthSource.liveEvidence,
      "NOT_ESTABLISHED_BY_THIS_ARTIFACT",
      "$.currentTruth.liveEvidence",
    ),
    networkNamedForContext: requireLiteral(
      currentTruthSource.networkNamedForContext,
      "zerone-1",
      "$.currentTruth.networkNamedForContext",
    ),
    sourceHasDualAuthoritySurfaces: requireLiteral(
      currentTruthSource.sourceHasDualAuthoritySurfaces,
      true,
      "$.currentTruth.sourceHasDualAuthoritySurfaces",
    ),
    sourceUsesBondedWealthForSdkGovernance: requireLiteral(
      currentTruthSource.sourceUsesBondedWealthForSdkGovernance,
      true,
      "$.currentTruth.sourceUsesBondedWealthForSdkGovernance",
    ),
    disclosedFoundingHouseholdRetainsEffectiveControl: requireLiteral(
      currentTruthSource.disclosedFoundingHouseholdRetainsEffectiveControl,
      true,
      "$.currentTruth.disclosedFoundingHouseholdRetainsEffectiveControl",
    ),
    sourceTargetModulesComplete: requireLiteral(
      currentTruthSource.sourceTargetModulesComplete,
      false,
      "$.currentTruth.sourceTargetModulesComplete",
    ),
    sourceAuthorityUnified: requireLiteral(
      currentTruthSource.sourceAuthorityUnified,
      false,
      "$.currentTruth.sourceAuthorityUnified",
    ),
    sourceRegistersH4FreezeHandler: requireLiteral(
      currentTruthSource.sourceRegistersH4FreezeHandler,
      false,
      "$.currentTruth.sourceRegistersH4FreezeHandler",
    ),
    sourceRegistersH4UnificationHandler: requireLiteral(
      currentTruthSource.sourceRegistersH4UnificationHandler,
      false,
      "$.currentTruth.sourceRegistersH4UnificationHandler",
    ),
    sourceRegistersH5NonEconomicGovernanceHandler: requireLiteral(
      currentTruthSource.sourceRegistersH5NonEconomicGovernanceHandler,
      false,
      "$.currentTruth.sourceRegistersH5NonEconomicGovernanceHandler",
    ),
  };

  const boundarySource = asObject(root.releaseBoundary, "$.releaseBoundary");
  exactKeys(boundarySource, RELEASE_BOUNDARY_KEYS, "$.releaseBoundary");
  for (const key of RELEASE_BOUNDARY_KEYS) {
    requireLiteral(boundarySource[key], false, `$.releaseBoundary.${key}`);
  }
  const releaseBoundary = Object.fromEntries(
    RELEASE_BOUNDARY_KEYS.map((key) => [key, false]),
  ) as AuthorityGeometryReleaseBoundary;

  const principles = asArray(
    root.principles,
    "$.principles",
    AUTHORITY_GEOMETRY_PRINCIPLE_IDS.length,
  ).map((principle, index) => parsePrinciple(principle, `$.principles[${index}]`, index));
  requireExactIdSequence(
    principles.map((principle) => principle.id),
    AUTHORITY_GEOMETRY_PRINCIPLE_IDS,
    "$.principles",
  );

  const capabilities = asArray(
    root.capabilities,
    "$.capabilities",
    AUTHORITY_GEOMETRY_CAPABILITY_IDS.length,
  ).map((capability, index) =>
    parseCapability(capability, `$.capabilities[${index}]`, index),
  );
  requireExactIdSequence(
    capabilities.map((capability) => capability.id),
    AUTHORITY_GEOMETRY_CAPABILITY_IDS,
    "$.capabilities",
  );

  const nodes = asArray(
    root.nodes,
    "$.nodes",
    AUTHORITY_GEOMETRY_NODE_IDS.length,
  ).map((node, index) => parseNode(node, `$.nodes[${index}]`, index));
  requireExactIdSequence(
    nodes.map((node) => node.id),
    AUTHORITY_GEOMETRY_NODE_IDS,
    "$.nodes",
  );

  const edges = asArray(root.edges, "$.edges", EDGE_IDS.length).map((edge, index) =>
    parseEdge(edge, `$.edges[${index}]`, index),
  );
  requireExactIdSequence(edges.map((edge) => edge.id), EDGE_IDS, "$.edges");

  const forbiddenInfluence = asArray(
    root.forbiddenInfluence,
    "$.forbiddenInfluence",
    FORBIDDEN_INFLUENCE_IDS.length,
  ).map((boundary, index) =>
    parseForbiddenInfluence(boundary, `$.forbiddenInfluence[${index}]`, index),
  );
  requireExactIdSequence(
    forbiddenInfluence.map((boundary) => boundary.id),
    FORBIDDEN_INFLUENCE_IDS,
    "$.forbiddenInfluence",
  );

  const sourceAnchors = asArray(
    root.sourceAnchors,
    "$.sourceAnchors",
    AUTHORITY_GEOMETRY_SOURCE_ANCHOR_IDS.length,
  ).map((anchor, index) =>
    parseSourceAnchor(anchor, `$.sourceAnchors[${index}]`, index),
  );
  requireExactIdSequence(
    sourceAnchors.map((anchor) => anchor.id),
    AUTHORITY_GEOMETRY_SOURCE_ANCHOR_IDS,
    "$.sourceAnchors",
  );
  const designAnchor = sourceAnchors[0];
  if (
    designAnchor?.path !== sourceDesign.repositoryPath ||
    designAnchor.sha256 !== sourceDesign.sha256
  ) {
    fail("$.sourceDesign", "must match the authoritative-state source anchor");
  }

  const currentFindings = asArray(
    root.currentFindings,
    "$.currentFindings",
    AUTHORITY_GEOMETRY_FINDING_IDS.length,
  ).map((finding, index) =>
    parseFinding(finding, `$.currentFindings[${index}]`, index),
  );
  requireExactIdSequence(
    currentFindings.map((finding) => finding.id),
    AUTHORITY_GEOMETRY_FINDING_IDS,
    "$.currentFindings",
  );

  const staticGateSource = asObject(root.staticAuthorityGate, "$.staticAuthorityGate");
  exactKeys(
    staticGateSource,
    ["authoritativeStateReleaseGate", "status", "mode", "surfaceChecks"],
    "$.staticAuthorityGate",
  );
  const surfaceChecks = asArray(
    staticGateSource.surfaceChecks,
    "$.staticAuthorityGate.surfaceChecks",
    AUTHORITY_GEOMETRY_SURFACE_IDS.length,
  ).map((check, index) =>
    parseSurfaceCheck(check, `$.staticAuthorityGate.surfaceChecks[${index}]`, index),
  );
  requireExactIdSequence(
    surfaceChecks.map((check) => check.id),
    AUTHORITY_GEOMETRY_SURFACE_IDS,
    "$.staticAuthorityGate.surfaceChecks",
  );
  const staticAuthorityGate: AuthorityGeometryStaticGate = {
    authoritativeStateReleaseGate: requireLiteral(
      staticGateSource.authoritativeStateReleaseGate,
      "H4-02",
      "$.staticAuthorityGate.authoritativeStateReleaseGate",
    ),
    status: requireLiteral(
      staticGateSource.status,
      "FAIL_CURRENT_SOURCE",
      "$.staticAuthorityGate.status",
    ),
    mode: requireLiteral(
      staticGateSource.mode,
      "REPORT_SUCCEEDS_TARGET_GATE_REFUSES",
      "$.staticAuthorityGate.mode",
    ),
    surfaceChecks,
  };

  const activationSource = asObject(root.activationGates, "$.activationGates");
  exactKeys(activationSource, ["h4", "h5"], "$.activationGates");
  const h4 = asArray(activationSource.h4, "$.activationGates.h4", 24).map(
    (gate, index) => parseActivationGate(gate, `$.activationGates.h4[${index}]`, "H4", index),
  );
  const h5 = asArray(activationSource.h5, "$.activationGates.h5", 14).map(
    (gate, index) => parseActivationGate(gate, `$.activationGates.h5[${index}]`, "H5", index),
  );
  if (h4.length !== 24) fail("$.activationGates.h4", "requires exactly 24 closed gates");
  if (h5.length !== 14) fail("$.activationGates.h5", "requires exactly 14 closed gates");

  const geometry: AuthorityGeometry = {
    schema: "zerone.authority-geometry/v1",
    revision: "1.0.0",
    snapshotDate: asIsoDate(root.snapshotDate, "$.snapshotDate"),
    status: "SOURCE_OBSERVATORY_ONLY",
    title: asString(root.title, "$.title", 128),
    summary: asString(root.summary, "$.summary"),
    sourceDesign,
    observationScopes,
    currentTruth,
    releaseBoundary,
    principles,
    capabilities,
    nodes,
    edges,
    forbiddenInfluence,
    sourceAnchors,
    currentFindings,
    staticAuthorityGate,
    activationGates: { h4, h5 },
    releaseAssessment: parseReleaseAssessment(
      root.releaseAssessment,
      "$.releaseAssessment",
    ),
  };
  validateGraph(geometry);
  assessAuthorityGeometry(geometry);
  return geometry;
}

function rejectExcessiveJsonNesting(raw: string): void {
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
      if (depth > 64) fail("$", "JSON nesting exceeds 64 levels");
    } else if (character === "}" || character === "]") {
      depth -= 1;
    }
  }
}

function rejectDuplicateJsonKeys(raw: string): void {
  let offset = 0;
  const whitespace = (): void => {
    while (/\s/.test(raw[offset] ?? "")) offset += 1;
  };
  const scanString = (): string => {
    const start = offset;
    offset += 1;
    while (offset < raw.length) {
      if (raw[offset] === "\\") offset += 2;
      else if (raw[offset] === '"') {
        offset += 1;
        return JSON.parse(raw.slice(start, offset)) as string;
      } else offset += 1;
    }
    return fail("$", "unterminated JSON string");
  };
  const scanValue = (path: string): void => {
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
        const key = scanString();
        if (keys.has(key)) fail(`${path}.${key}`, "duplicate JSON object key");
        keys.add(key);
        whitespace();
        offset += 1;
        scanValue(`${path}.${key}`);
        whitespace();
        if (raw[offset] === "}") {
          offset += 1;
          return;
        }
        offset += 1;
      }
      return;
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
        offset += 1;
        index += 1;
      }
      return;
    }
    if (token === '"') {
      scanString();
      return;
    }
    while (offset < raw.length && !/[\s,\]}]/.test(raw[offset] ?? "")) offset += 1;
  };
  scanValue("$");
}

export function parseAuthorityGeometryJson(raw: string): AuthorityGeometry {
  if (new TextEncoder().encode(raw).byteLength > AUTHORITY_GEOMETRY_MAX_BYTES) {
    throw new AuthorityGeometryDataError(
      `Authority Geometry exceeds ${AUTHORITY_GEOMETRY_MAX_BYTES} UTF-8 bytes`,
    );
  }
  let value: unknown;
  try {
    value = JSON.parse(raw) as unknown;
  } catch {
    throw new AuthorityGeometryDataError("Authority Geometry contains malformed JSON");
  }
  rejectExcessiveJsonNesting(raw);
  rejectDuplicateJsonKeys(raw);
  return parseAuthorityGeometry(value);
}

async function sha256Hex(bytes: Uint8Array): Promise<string> {
  if (globalThis.crypto?.subtle === undefined) {
    throw new AuthorityGeometryDataError("SHA-256 verification is unavailable");
  }
  const input = new ArrayBuffer(bytes.byteLength);
  new Uint8Array(input).set(bytes);
  const digest = await globalThis.crypto.subtle.digest("SHA-256", input);
  return [...new Uint8Array(digest)]
    .map((byte) => byte.toString(16).padStart(2, "0"))
    .join("");
}

async function readBoundedResponse(
  response: Response,
  signal: AbortSignal,
): Promise<Uint8Array> {
  if (response.body === null) {
    throw new AuthorityGeometryDataError("Authority Geometry returned an empty body");
  }
  const reader = response.body.getReader();
  const chunks: Uint8Array[] = [];
  let total = 0;
  let rejectAbort: ((reason?: unknown) => void) | undefined;
  const aborted = new Promise<never>((_resolve, reject) => {
    rejectAbort = reject;
  });
  const onAbort = (): void => {
    rejectAbort?.(
      signal.reason ?? new DOMException("request timed out", "TimeoutError"),
    );
  };
  if (signal.aborted) onAbort();
  else signal.addEventListener("abort", onAbort, { once: true });
  try {
    while (true) {
      const result = await Promise.race([reader.read(), aborted]);
      if (result.done) break;
      total += result.value.byteLength;
      if (total > AUTHORITY_GEOMETRY_MAX_BYTES) {
        void reader.cancel("response exceeded size limit").catch(() => undefined);
        throw new AuthorityGeometryDataError("Authority Geometry exceeded its size limit");
      }
      chunks.push(result.value);
    }
  } finally {
    signal.removeEventListener("abort", onAbort);
    try {
      reader.releaseLock();
    } catch {
      // A refused stream may already have released its lock.
    }
  }
  const bytes = new Uint8Array(total);
  let offset = 0;
  for (const chunk of chunks) {
    bytes.set(chunk, offset);
    offset += chunk.byteLength;
  }
  return bytes;
}

function assertExactResponseUrl(
  response: Response,
  baseUrl: string | undefined,
): void {
  if (response.redirected) {
    throw new AuthorityGeometryDataError("Authority Geometry response was redirected");
  }
  if (baseUrl === undefined) return;
  let expected: URL;
  let actual: URL;
  try {
    expected = new URL(AUTHORITY_GEOMETRY_ENDPOINT, baseUrl);
    actual = new URL(response.url);
  } catch {
    throw new AuthorityGeometryDataError("Authority Geometry returned an invalid final URL");
  }
  if (
    expected.origin !== actual.origin ||
    expected.pathname !== actual.pathname ||
    expected.search !== actual.search ||
    expected.hash !== actual.hash
  ) {
    throw new AuthorityGeometryDataError(
      "Authority Geometry left its canonical same-origin path",
    );
  }
}

const HTTP_TOKEN = "[!#$%&'*+.^_`|~0-9A-Za-z-]+";
const HTTP_QUOTED_STRING = '"(?:[^"\\\\\\r\\n]|\\\\[\\t -~])*"';
const JSON_MEDIA_TYPE = new RegExp(
  `^application/(?:json|${HTTP_TOKEN}\\+json)` +
    `(?:\\s*;\\s*${HTTP_TOKEN}\\s*=\\s*(?:${HTTP_TOKEN}|${HTTP_QUOTED_STRING}))*\\s*$`,
  "i",
);

function isJsonMediaType(value: string): boolean {
  return JSON_MEDIA_TYPE.test(value);
}

function refuseEarlyResponse(
  response: Response,
  controller: AbortController,
  error: AuthorityGeometryDataError,
): never {
  if (!controller.signal.aborted) controller.abort(error);
  if (response.body !== null) {
    try {
      void response.body.cancel(error).catch(() => undefined);
    } catch {
      // A hostile or already-locked response body must not delay refusal.
    }
  }
  throw error;
}

export async function fetchAuthorityGeometry(
  options: AuthorityGeometryFetchOptions = {},
): Promise<AuthorityGeometry> {
  const fetcher = options.fetcher ?? globalThis.fetch;
  if (typeof fetcher !== "function") {
    throw new AuthorityGeometryDataError("Authority Geometry fetch is unavailable");
  }
  const timeoutMs = options.timeoutMs ?? AUTHORITY_GEOMETRY_TIMEOUT_MS;
  if (!Number.isSafeInteger(timeoutMs) || timeoutMs < 1 || timeoutMs > 15_000) {
    throw new AuthorityGeometryDataError("Authority Geometry timeout is out of bounds");
  }
  const baseUrl =
    options.baseUrl ??
    (typeof globalThis.location?.href === "string"
      ? globalThis.location.href
      : undefined);
  if (baseUrl !== undefined) {
    const endpointUrl = new URL(AUTHORITY_GEOMETRY_ENDPOINT, baseUrl);
    const base = new URL(baseUrl);
    if (endpointUrl.origin !== base.origin) {
      throw new AuthorityGeometryDataError("Authority Geometry endpoint is not same-origin");
    }
  }

  const controller = new AbortController();
  const deadline = globalThis.setTimeout(
    () => controller.abort(new DOMException("request timed out", "TimeoutError")),
    timeoutMs,
  );
  try {
    let response: Response;
    let rejectFetchAbort: ((reason?: unknown) => void) | undefined;
    const fetchAborted = new Promise<never>((_resolve, reject) => {
      rejectFetchAbort = reject;
    });
    const onFetchAbort = (): void => {
      rejectFetchAbort?.(
        controller.signal.reason ??
          new DOMException("request timed out", "TimeoutError"),
      );
    };
    if (controller.signal.aborted) onFetchAbort();
    else controller.signal.addEventListener("abort", onFetchAbort, { once: true });
    try {
      response = await Promise.race([
        fetcher(AUTHORITY_GEOMETRY_ENDPOINT, {
          cache: "no-store",
          credentials: "same-origin",
          headers: { Accept: "application/json" },
          redirect: "error",
          signal: controller.signal,
        }),
        fetchAborted,
      ]);
    } catch (error) {
      if (controller.signal.aborted) {
        throw new AuthorityGeometryDataError("Authority Geometry request timed out");
      }
      throw new AuthorityGeometryDataError(
        error instanceof Error
          ? `Authority Geometry is unavailable: ${error.message}`
          : "Authority Geometry is unavailable",
      );
    } finally {
      controller.signal.removeEventListener("abort", onFetchAbort);
    }
    if (!response.ok) {
      refuseEarlyResponse(
        response,
        controller,
        new AuthorityGeometryDataError(
          `Authority Geometry returned HTTP ${response.status}`,
        ),
      );
    }
    try {
      assertExactResponseUrl(response, baseUrl);
    } catch (error) {
      refuseEarlyResponse(
        response,
        controller,
        error instanceof AuthorityGeometryDataError
          ? error
          : new AuthorityGeometryDataError("Authority Geometry returned an invalid final URL"),
      );
    }
    const contentType = response.headers.get("content-type");
    if (contentType === null || !isJsonMediaType(contentType)) {
      refuseEarlyResponse(
        response,
        controller,
        new AuthorityGeometryDataError("Authority Geometry returned non-JSON content"),
      );
    }
    const declaredLength = response.headers.get("content-length");
    if (
      declaredLength !== null &&
      (!/^\d+$/.test(declaredLength) ||
        Number(declaredLength) > AUTHORITY_GEOMETRY_MAX_BYTES)
    ) {
      refuseEarlyResponse(
        response,
        controller,
        new AuthorityGeometryDataError("Authority Geometry exceeded its size limit"),
      );
    }
    let bytes: Uint8Array;
    try {
      bytes = await readBoundedResponse(response, controller.signal);
    } catch (error) {
      if (controller.signal.aborted) {
        throw new AuthorityGeometryDataError("Authority Geometry request timed out");
      }
      throw error;
    }
    if ((await sha256Hex(bytes)) !== AUTHORITY_GEOMETRY_SHA256) {
      throw new AuthorityGeometryDataError(
        "Authority Geometry did not match the reviewed canonical digest",
      );
    }
    let raw: string;
    try {
      raw = new TextDecoder("utf-8", { fatal: true }).decode(bytes);
    } catch {
      throw new AuthorityGeometryDataError("Authority Geometry was not valid UTF-8");
    }
    return parseAuthorityGeometryJson(raw);
  } finally {
    globalThis.clearTimeout(deadline);
  }
}

function element<K extends keyof HTMLElementTagNameMap>(
  tag: K,
  className?: string,
  text?: string,
): HTMLElementTagNameMap[K] {
  const node = document.createElement(tag);
  if (className !== undefined) node.className = className;
  if (text !== undefined) node.textContent = text;
  return node;
}

function humanize(value: string): string {
  const words = value.replaceAll("_", " ").replaceAll("-", " ").toLowerCase();
  return words.charAt(0).toUpperCase() + words.slice(1);
}

function appendMetric(root: HTMLElement, label: string, value: string): void {
  const metric = element("div");
  metric.append(element("strong", undefined, value), element("span", undefined, label));
  root.append(metric);
}

function renderPillarPlane(
  geometry: AuthorityGeometry,
  nodeId: "sdk-staking" | "sdk-gov" | "ontology",
  description: string,
): HTMLElement {
  const nodeById = new Map(geometry.nodes.map((node) => [node.id, node]));
  const node = nodeById.get(nodeId);
  if (node === undefined) fail("$.nodes", `missing target pillar ${nodeId}`);
  const plane = element("article", "authority-geometry-plane");
  plane.append(
    element("p", undefined, "ACCEPTED TARGET · DESIGN ONLY"),
    element("h4", undefined, node.label),
    element("p", undefined, description),
  );
  const capabilities = geometry.capabilities.filter(
    (capability) => capability.targetWriter === nodeId,
  );
  const capabilityList = element("ul");
  for (const capability of capabilities) {
    const item = element("li");
    item.append(
      element("strong", undefined, capability.label),
      element("span", undefined, "Exactly one accepted target writer"),
    );
    capabilityList.append(item);
  }
  const relationships = geometry.edges.filter(
    (edge) =>
      edge.scope === "ACCEPTED_TARGET" &&
      (edge.from === nodeId || edge.to === nodeId),
  );
  if (relationships.length > 0) {
    const relationList = element("ul");
    for (const edge of relationships) {
      const item = element("li");
      const from = nodeById.get(edge.from)?.label ?? edge.from;
      const to = nodeById.get(edge.to)?.label ?? edge.to;
      item.append(
        element("strong", undefined, `${from} → ${to}`),
        element("span", undefined, humanize(edge.relationship)),
      );
      relationList.append(item);
    }
    plane.append(capabilityList, relationList);
  } else {
    plane.append(capabilityList);
  }
  return plane;
}

export function renderAuthorityGeometry(
  geometry: AuthorityGeometry,
): HTMLElement {
  const assessment = assessAuthorityGeometry(geometry);
  const shell = element("section", "authority-geometry-shell");
  shell.setAttribute("aria-label", "Authority Geometry source observatory");

  const status = element("header", "authority-geometry-status");
  status.append(
    element("p", undefined, "SOURCE OBSERVATORY · TARGET NO-GO"),
    element("h3", undefined, geometry.title),
    element("p", undefined, geometry.summary),
  );

  const metrics = element("div", "authority-geometry-metrics");
  appendMetric(metrics, "overall activation", assessment.overall.replace("_", "-"));
  appendMetric(
    metrics,
    "static surfaces passing",
    `${assessment.staticAuthoritySurfacesPassing}/${assessment.staticAuthoritySurfacesTotal}`,
  );
  appendMetric(
    metrics,
    "H4 gates evidenced",
    `${assessment.h4GatesEvidenced}/${assessment.h4GatesTotal}`,
  );
  appendMetric(
    metrics,
    "H5 gates evidenced",
    `${assessment.h5GatesEvidenced}/${assessment.h5GatesTotal}`,
  );

  const planes = element("div", "authority-geometry-planes");
  planes.append(
    renderPillarPlane(
      geometry,
      "sdk-staking",
      "Sole accepted target writer for consensus stake, validator power, bonded status, delegations, and consensus faults.",
    ),
    renderPillarPlane(
      geometry,
      "sdk-gov",
      "Sole accepted target executor for ordinary proposals, ballots, tallies, decisions, and typed atomic execution.",
    ),
    renderPillarPlane(
      geometry,
      "ontology",
      "Sole accepted target writer for domain identity, hierarchy, strata, lifecycle, aliases, and tombstones.",
    ),
  );

  const principles = element("section", "authority-geometry-principles");
  principles.append(element("h4", undefined, "Geometry principles"));
  const principleList = element("ul");
  for (const principle of geometry.principles) {
    const item = element("li");
    item.append(
      element("strong", undefined, principle.label),
      element("p", undefined, principle.rule),
    );
    principleList.append(item);
  }
  principles.append(principleList);

  const findings = element("section", "authority-geometry-findings");
  findings.append(
    element("h4", undefined, "Current findings"),
    element(
      "p",
      undefined,
      "These open findings make the H4 static-authority target gate refuse.",
    ),
  );
  const findingList = element("ul");
  for (const finding of geometry.currentFindings) {
    const item = element("li");
    item.append(
      element("strong", undefined, finding.label),
      element("span", undefined, humanize(finding.releaseSurface)),
    );
    findingList.append(item);
  }
  findings.append(findingList);

  const tableWrap = element("div", "authority-geometry-table-wrap");
  const table = element("table", "authority-geometry-table");
  const caption = element(
    "caption",
    undefined,
    "Static authority release-gate assessment",
  );
  const head = element("thead");
  const headerRow = element("tr");
  for (const heading of ["Surface", "Status", "Open evidence"]) {
    headerRow.append(element("th", undefined, heading));
  }
  head.append(headerRow);
  const body = element("tbody");
  for (const check of geometry.staticAuthorityGate.surfaceChecks) {
    const row = element("tr");
    row.append(
      element("th", undefined, humanize(check.id)),
      element("td", undefined, humanize(check.status)),
      element(
        "td",
        undefined,
        check.findingIds.length === 0
          ? "No open finding in this static inspection"
          : check.findingIds.map(humanize).join("; "),
      ),
    );
    body.append(row);
  }
  table.append(caption, head, body);
  tableWrap.append(table);

  const relationshipTableWrap = element("div", "authority-geometry-table-wrap");
  const relationshipTable = element("table", "authority-geometry-table");
  const relationshipCaption = element(
    "caption",
    undefined,
    "Every classified authority relationship and its protections",
  );
  const relationshipHead = element("thead");
  const relationshipHeaderRow = element("tr");
  for (const heading of [
    "Edge",
    "From",
    "To",
    "Relationship",
    "Scope",
    "Effect",
    "Evidence",
    "Consent boundary",
    "Challenge route",
    "Repair route",
    "Exit route",
  ]) {
    const cell = element("th", undefined, heading);
    cell.setAttribute("scope", "col");
    relationshipHeaderRow.append(cell);
  }
  relationshipHead.append(relationshipHeaderRow);
  const relationshipBody = element("tbody");
  const nodeById = new Map(geometry.nodes.map((node) => [node.id, node]));
  for (const edge of geometry.edges) {
    const row = element("tr");
    const edgeCell = element("th", undefined, edge.id);
    edgeCell.setAttribute("scope", "row");
    const from = nodeById.get(edge.from);
    const to = nodeById.get(edge.to);
    row.append(
      edgeCell,
      element("td", undefined, `${from?.label ?? edge.from} (${edge.from})`),
      element("td", undefined, `${to?.label ?? edge.to} (${edge.to})`),
      element("td", undefined, edge.relationship),
      element("td", undefined, edge.scope),
      element("td", undefined, edge.effect),
      element("td", undefined, edge.evidence.join(", ")),
      element("td", undefined, edge.protections.consentBoundary),
      element("td", undefined, edge.protections.challengeRoute),
      element("td", undefined, edge.protections.repairRoute),
      element("td", undefined, edge.protections.exitRoute),
    );
    relationshipBody.append(row);
  }
  relationshipTable.append(relationshipCaption, relationshipHead, relationshipBody);
  relationshipTableWrap.append(relationshipTable);

  const footer = element("footer", "authority-geometry-footer");
  const links = element("div");
  const rawLink = element("a", undefined, "Canonical Authority Geometry JSON");
  rawLink.href = AUTHORITY_GEOMETRY_ENDPOINT;
  const designLink = element("a", undefined, "Accepted authoritative-state design");
  designLink.href =
    "https://github.com/cambridgetcg/zerone-core/blob/0558c915e34acc11ed681795ab595240018b0e76/docs/AUTHORITATIVE-STATE.md";
  designLink.target = "_blank";
  designLink.rel = "noreferrer";
  links.append(rawLink, document.createTextNode(" · "), designLink);
  footer.append(
    element(
      "p",
      undefined,
      `Accepted source design: ${geometry.sourceDesign.repositoryPath} · SHA-256 ${geometry.sourceDesign.sha256}`,
    ),
    element(
      "p",
      undefined,
      "This static artifact does not schedule H4 or H5, change a validator, submit a transaction, move funds, grant qualification, or establish live network state.",
    ),
    links,
  );

  shell.append(
    status,
    metrics,
    planes,
    principles,
    findings,
    tableWrap,
    relationshipTableWrap,
    footer,
  );
  return shell;
}

function renderAuthorityGeometryError(
  root: HTMLElement,
  error: unknown,
): void {
  const panel = element("div", "authority-geometry-error");
  panel.setAttribute("role", "alert");
  panel.append(
    element("strong", undefined, "The reviewed Authority Geometry did not load."),
    element(
      "p",
      undefined,
      error instanceof Error
        ? error.message
        : "The static source observatory is unavailable or failed validation.",
    ),
  );
  const retry = element("button", undefined, "Retry");
  retry.type = "button";
  retry.addEventListener("click", () => void initialiseAuthorityGeometry(root));
  const raw = element("a", undefined, "Open canonical JSON");
  raw.href = AUTHORITY_GEOMETRY_ENDPOINT;
  panel.append(retry, raw);
  root.replaceChildren(panel);
  root.ariaBusy = "false";
}

export async function initialiseAuthorityGeometry(root: HTMLElement): Promise<void> {
  root.ariaBusy = "true";
  const loading = element(
    "div",
    "authority-geometry-loading",
    "Loading the reviewed source geometry…",
  );
  loading.setAttribute("role", "status");
  root.replaceChildren(loading);
  try {
    const geometry = await fetchAuthorityGeometry();
    root.replaceChildren(renderAuthorityGeometry(geometry));
    root.ariaBusy = "false";
  } catch (error) {
    renderAuthorityGeometryError(root, error);
  }
}
